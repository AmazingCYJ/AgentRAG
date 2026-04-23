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
	Question         string
	DeepThinking     bool
	KnowledgeContext string
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

// ContextRetriever 定义知识上下文检索函数。
type ContextRetriever func(ctx context.Context, query string, limit int) (string, error)

// RouteKind 定义聊天路由类型。
type RouteKind string

const (
	RouteKindNone      RouteKind = "none"
	RouteKindKnowledge RouteKind = "knowledge"
	RouteKindTool      RouteKind = "tool"
)

// RouteDecision 定义路由决策结果。
type RouteDecision struct {
	Kind   RouteKind
	ToolID string
}

// RouteResolver 定义聊天路由决策函数。
type RouteResolver func(ctx context.Context, question string) (RouteDecision, error)

// ToolCaller 定义 MCP 工具调用函数。
type ToolCaller func(ctx context.Context, toolID, question string) (string, error)

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
func NewGeneratorFromConfig(
	cfg appconfig.AIConfig,
	retriever ContextRetriever,
	resolver RouteResolver,
	toolCaller ToolCaller,
) Generator {
	var generator Generator
	if strings.TrimSpace(cfg.APIKey) == "" || strings.TrimSpace(cfg.Model) == "" {
		generator = &fallbackGenerator{}
	} else {
		einoGenerator, err := NewEinoGenerator(cfg)
		if err != nil {
			generator = &fallbackGenerator{}
		} else {
			generator = einoGenerator
		}
	}
	return wrapWithRouting(generator, retriever, resolver, toolCaller, 4)
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
	}
	if strings.TrimSpace(req.KnowledgeContext) != "" {
		messages = append(messages, schema.SystemMessage(
			"以下是从知识库检索到的上下文，请优先依据这些内容回答；如果信息不足，请明确说明。\n\n"+req.KnowledgeContext,
		))
	}
	messages = append(messages, schema.UserMessage(strings.TrimSpace(req.Question)))
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

type retrievalGenerator struct {
	inner     Generator
	retriever ContextRetriever
	limit     int
}

func wrapWithRetriever(inner Generator, retriever ContextRetriever, limit int) Generator {
	if inner == nil {
		inner = &fallbackGenerator{}
	}
	if retriever == nil {
		return inner
	}
	if limit <= 0 {
		limit = 4
	}
	return &retrievalGenerator{
		inner:     inner,
		retriever: retriever,
		limit:     limit,
	}
}

func (g *retrievalGenerator) Generate(ctx context.Context, req GenerateRequest) (GenerateResult, error) {
	contextText, err := g.retriever(ctx, req.Question, g.limit)
	if err == nil {
		req.KnowledgeContext = strings.TrimSpace(contextText)
	}
	return g.inner.Generate(ctx, req)
}

type routingGenerator struct {
	inner      Generator
	retriever  ContextRetriever
	resolver   RouteResolver
	toolCaller ToolCaller
	limit      int
}

func wrapWithRouting(
	inner Generator,
	retriever ContextRetriever,
	resolver RouteResolver,
	toolCaller ToolCaller,
	limit int,
) Generator {
	if inner == nil {
		inner = &fallbackGenerator{}
	}
	if limit <= 0 {
		limit = 4
	}
	return &routingGenerator{
		inner:      inner,
		retriever:  retriever,
		resolver:   resolver,
		toolCaller: toolCaller,
		limit:      limit,
	}
}

func (g *routingGenerator) Generate(ctx context.Context, req GenerateRequest) (GenerateResult, error) {
	decision := RouteDecision{Kind: RouteKindNone}
	if g.resolver != nil {
		if resolved, err := g.resolver(ctx, req.Question); err == nil {
			decision = resolved
		}
	}

	switch decision.Kind {
	case RouteKindTool:
		if g.toolCaller != nil && strings.TrimSpace(decision.ToolID) != "" {
			if text, err := g.toolCaller(ctx, decision.ToolID, req.Question); err == nil {
				req.KnowledgeContext = strings.TrimSpace(text)
			}
		}
	case RouteKindKnowledge:
		if g.retriever != nil {
			if contextText, err := g.retriever(ctx, req.Question, g.limit); err == nil {
				req.KnowledgeContext = strings.TrimSpace(contextText)
			}
		}
	default:
		if g.retriever != nil {
			if contextText, err := g.retriever(ctx, req.Question, g.limit); err == nil {
				req.KnowledgeContext = strings.TrimSpace(contextText)
			}
		}
	}

	return g.inner.Generate(ctx, req)
}
