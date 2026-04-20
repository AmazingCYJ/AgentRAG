package chat

import (
	"context"
	"strings"

	appconfig "github.com/AmazingCYJ/AgentRAG/internal/platform/config"
	einoopenai "github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/schema"
)

// GenerateRequest 定义聊天生成输入。
type GenerateRequest struct {
	Question     string
	DeepThinking bool
}

// GenerateResult 定义聊天生成输出。
type GenerateResult struct {
	Thinking string
	Answer   string
}

// Generator 定义聊天生成器接口。
type Generator interface {
	Generate(ctx context.Context, req GenerateRequest) (GenerateResult, error)
}

type fallbackGenerator struct{}

func (g *fallbackGenerator) Generate(_ context.Context, req GenerateRequest) (GenerateResult, error) {
	thinking := ""
	if req.DeepThinking {
		thinking = buildThinkingText(req.Question)
	}
	return GenerateResult{
		Thinking: thinking,
		Answer:   buildResponseText(req.Question, req.DeepThinking),
	}, nil
}

// NewGeneratorFromConfig 根据配置创建聊天生成器。
func NewGeneratorFromConfig(cfg appconfig.AIConfig) Generator {
	if strings.TrimSpace(cfg.APIKey) == "" || strings.TrimSpace(cfg.Model) == "" {
		return &fallbackGenerator{}
	}

	generator, err := NewEinoGenerator(cfg)
	if err != nil {
		return &fallbackGenerator{}
	}
	return generator
}

// EinoGenerator 基于 Eino OpenAI chat model 封装聊天生成。
type EinoGenerator struct {
	systemPrompt   string
	defaultModel   *einoopenai.ChatModel
	reasoningModel *einoopenai.ChatModel
}

// NewEinoGenerator 创建 Eino 聊天生成器。
func NewEinoGenerator(cfg appconfig.AIConfig) (*EinoGenerator, error) {
	defaultModel, err := einoopenai.NewChatModel(context.Background(), &einoopenai.ChatModelConfig{
		APIKey:  cfg.APIKey,
		BaseURL: cfg.BaseURL,
		Model:   cfg.Model,
		Timeout: cfg.Timeout,
	})
	if err != nil {
		return nil, err
	}

	reasoningModel := defaultModel
	if strings.TrimSpace(cfg.DeepThinkingModel) != "" && cfg.DeepThinkingModel != cfg.Model {
		reasoningModel, err = einoopenai.NewChatModel(context.Background(), &einoopenai.ChatModelConfig{
			APIKey:  cfg.APIKey,
			BaseURL: cfg.BaseURL,
			Model:   cfg.DeepThinkingModel,
			Timeout: cfg.Timeout,
		})
		if err != nil {
			return nil, err
		}
	}

	return &EinoGenerator{
		systemPrompt:   defaultSystemPrompt(cfg.SystemPrompt),
		defaultModel:   defaultModel,
		reasoningModel: reasoningModel,
	}, nil
}

// Generate 调用 Eino 模型生成回答。
func (g *EinoGenerator) Generate(ctx context.Context, req GenerateRequest) (GenerateResult, error) {
	model := g.defaultModel
	if req.DeepThinking && g.reasoningModel != nil {
		model = g.reasoningModel
	}

	messages := []*schema.Message{
		schema.SystemMessage(g.systemPrompt),
		schema.UserMessage(strings.TrimSpace(req.Question)),
	}
	resp, err := model.Generate(ctx, messages)
	if err != nil {
		return GenerateResult{}, err
	}
	return GenerateResult{
		Thinking: strings.TrimSpace(resp.ReasoningContent),
		Answer:   strings.TrimSpace(resp.Content),
	}, nil
}

func defaultSystemPrompt(prompt string) string {
	if strings.TrimSpace(prompt) == "" {
		return "你是 AgentRAG 的智能问答助手，请使用简洁、专业、可执行的中文回答用户问题。"
	}
	return strings.TrimSpace(prompt)
}
