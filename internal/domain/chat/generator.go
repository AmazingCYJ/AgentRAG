package chat

import (
	"context"
	"encoding/json"
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
	RewriteQuestion  string
}

// GenerateResult 定义聊天生成输出。
type GenerateResult struct {
	Thinking string
	Answer   string
	Steps    []WorkflowStep
}

// WorkflowStep 定义聊天 workflow 的单个阶段。
type WorkflowStep struct {
	NodeID     string
	NodeType   string
	NodeName   string
	Status     string
	DurationMs int64
	Detail     string
}

// Generator 定义聊天生成器接口。
type Generator interface {
	Generate(ctx context.Context, req GenerateRequest) (GenerateResult, error)
}

// ContextRetriever 定义知识上下文检索函数。
type ContextRetriever func(ctx context.Context, query string, limit int) (string, error)

// QueryRewriter 定义查询改写函数。
type QueryRewriter func(ctx context.Context, question string) (string, error)

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

// ToolParamExtractor 定义工具参数提取函数。
type ToolParamExtractor func(ctx context.Context, schema mcpToolSchema, question string) (map[string]any, error)

type fallbackGenerator struct{}

type workflowOptions struct {
	retrievalLimit  int
	rewritePrompt   string
	toolParamPrompt string
	knowledgePrompt string
}

func buildWorkflowOptions(cfg appconfig.AIWorkflowConfig) workflowOptions {
	options := workflowOptions{
		retrievalLimit:  4,
		rewritePrompt:   defaultRewritePrompt(),
		toolParamPrompt: defaultToolParamPrompt(),
		knowledgePrompt: defaultKnowledgePrompt(),
	}
	if cfg.RetrievalLimit > 0 {
		options.retrievalLimit = cfg.RetrievalLimit
	}
	if strings.TrimSpace(cfg.RewritePrompt) != "" {
		options.rewritePrompt = strings.TrimSpace(cfg.RewritePrompt)
	}
	if strings.TrimSpace(cfg.ToolParamPrompt) != "" {
		options.toolParamPrompt = strings.TrimSpace(cfg.ToolParamPrompt)
	}
	if strings.TrimSpace(cfg.KnowledgePrompt) != "" {
		options.knowledgePrompt = strings.TrimSpace(cfg.KnowledgePrompt)
	}
	return options
}

func (g *fallbackGenerator) Generate(_ context.Context, req GenerateRequest) (GenerateResult, error) {
	thinking := ""
	if req.DeepThinking {
		thinking = buildThinkingText(req.Question)
	}
	return GenerateResult{
		Thinking: thinking,
		Answer:   buildResponseText(req.Question, req.DeepThinking),
		Steps: []WorkflowStep{
			{
				NodeID:     "fallback_generate",
				NodeType:   "LLM",
				NodeName:   "Fallback Generate",
				Status:     "success",
				DurationMs: 1,
			},
		},
	}, nil
}

// NewGeneratorFromConfig 根据配置创建聊天生成器。
func NewGeneratorFromConfig(
	cfg appconfig.AIConfig,
	retriever ContextRetriever,
	resolver RouteResolver,
	toolCaller ToolCaller,
) Generator {
	options := buildWorkflowOptions(cfg.Workflow)
	var generator Generator
	if strings.TrimSpace(cfg.APIKey) == "" || strings.TrimSpace(cfg.Model) == "" {
		generator = &fallbackGenerator{}
	} else {
		einoGenerator, err := NewEinoGenerator(cfg, options)
		if err != nil {
			generator = &fallbackGenerator{}
		} else {
			generator = einoGenerator
		}
	}
	return wrapWithRewrite(
		wrapWithRouting(generator, retriever, resolver, toolCaller, options.retrievalLimit),
		BuildEinoQueryRewriter(cfg),
	)
}

// EinoGenerator 基于 Eino OpenAI chat model 封装聊天生成。
type EinoGenerator struct {
	systemPrompt    string
	knowledgePrompt string
	defaultModel    *einoopenai.ChatModel
	reasoningModel  *einoopenai.ChatModel
}

// NewEinoGenerator 创建 Eino 聊天生成器。
func NewEinoGenerator(cfg appconfig.AIConfig, options workflowOptions) (*EinoGenerator, error) {
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
		systemPrompt:    defaultSystemPrompt(cfg.SystemPrompt),
		knowledgePrompt: options.knowledgePrompt,
		defaultModel:    defaultModel,
		reasoningModel:  reasoningModel,
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
			g.knowledgePrompt+"\n\n"+req.KnowledgeContext,
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
		Steps: []WorkflowStep{
			{
				NodeID:     "model_generate",
				NodeType:   "LLM",
				NodeName:   "Eino Generate",
				Status:     "success",
				DurationMs: 1,
			},
		},
	}, nil
}

func defaultSystemPrompt(prompt string) string {
	if strings.TrimSpace(prompt) == "" {
		return "你是 AgentRAG 的智能问答助手，请使用简洁、专业、可执行的中文回答用户问题。"
	}
	return strings.TrimSpace(prompt)
}

func defaultKnowledgePrompt() string {
	return "以下是从知识库检索到的上下文，请优先依据这些内容回答；如果信息不足，请明确说明。"
}

func defaultRewritePrompt() string {
	return "你是 AgentRAG 的查询改写器。请把用户问题改写成适合检索和工具路由的独立中文查询，只输出改写后的查询，不要解释。"
}

func defaultToolParamPrompt() string {
	return "你是一个工具参数提取器。请严格输出 JSON 对象，不要输出额外解释。只返回工具定义中声明过的参数。"
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
	contextText, err := g.retriever(ctx, effectiveQuery(req), g.limit)
	if err == nil {
		req.KnowledgeContext = strings.TrimSpace(contextText)
	}
	return g.inner.Generate(ctx, req)
}

type rewriteGenerator struct {
	inner    Generator
	rewriter QueryRewriter
}

func wrapWithRewrite(inner Generator, rewriter QueryRewriter) Generator {
	if inner == nil {
		inner = &fallbackGenerator{}
	}
	if rewriter == nil {
		return inner
	}
	return &rewriteGenerator{
		inner:    inner,
		rewriter: rewriter,
	}
}

func (g *rewriteGenerator) Generate(ctx context.Context, req GenerateRequest) (GenerateResult, error) {
	rewritten, err := g.rewriter(ctx, req.Question)
	if err == nil {
		rewritten = strings.TrimSpace(rewritten)
		if rewritten != "" && rewritten != strings.TrimSpace(req.Question) {
			req.RewriteQuestion = rewritten
		}
	}

	result, err := g.inner.Generate(ctx, req)
	if err != nil {
		return GenerateResult{}, err
	}
	if req.RewriteQuestion == "" {
		return result, nil
	}
	step := WorkflowStep{
		NodeID:     "rewrite_query",
		NodeType:   "REWRITER",
		NodeName:   "Rewrite Query",
		Status:     "success",
		DurationMs: 1,
		Detail:     req.RewriteQuestion,
	}
	result.Steps = append([]WorkflowStep{step}, result.Steps...)
	return result, nil
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
	query := effectiveQuery(req)
	decision := RouteDecision{Kind: RouteKindNone}
	if g.resolver != nil {
		if resolved, err := g.resolver(ctx, query); err == nil {
			decision = resolved
		}
	}
	steps := []WorkflowStep{
		{
			NodeID:     "route_intent",
			NodeType:   "ROUTER",
			NodeName:   "Route Intent",
			Status:     "success",
			DurationMs: 1,
			Detail:     string(decision.Kind),
		},
	}

	switch decision.Kind {
	case RouteKindTool:
		if g.toolCaller != nil && strings.TrimSpace(decision.ToolID) != "" {
			if text, err := g.toolCaller(ctx, decision.ToolID, query); err == nil {
				req.KnowledgeContext = strings.TrimSpace(text)
				steps = append(steps, WorkflowStep{
					NodeID:     "call_tool",
					NodeType:   "TOOL",
					NodeName:   "Call MCP Tool",
					Status:     "success",
					DurationMs: 1,
					Detail:     decision.ToolID,
				})
			}
		}
	case RouteKindKnowledge:
		if g.retriever != nil {
			if contextText, err := g.retriever(ctx, query, g.limit); err == nil {
				req.KnowledgeContext = strings.TrimSpace(contextText)
				steps = append(steps, WorkflowStep{
					NodeID:     "retrieve_context",
					NodeType:   "RETRIEVER",
					NodeName:   "Retrieve Knowledge",
					Status:     "success",
					DurationMs: 1,
				})
			}
		}
	default:
		if g.retriever != nil {
			if contextText, err := g.retriever(ctx, query, g.limit); err == nil {
				req.KnowledgeContext = strings.TrimSpace(contextText)
				if req.KnowledgeContext != "" {
					steps = append(steps, WorkflowStep{
						NodeID:     "retrieve_context",
						NodeType:   "RETRIEVER",
						NodeName:   "Retrieve Knowledge",
						Status:     "success",
						DurationMs: 1,
					})
				}
			}
		}
	}

	result, err := g.inner.Generate(ctx, req)
	if err != nil {
		return GenerateResult{}, err
	}
	result.Steps = append(steps, result.Steps...)
	return result, nil
}

func effectiveQuery(req GenerateRequest) string {
	if strings.TrimSpace(req.RewriteQuestion) != "" {
		return strings.TrimSpace(req.RewriteQuestion)
	}
	return strings.TrimSpace(req.Question)
}

// BuildEinoQueryRewriter 基于 Eino 模型创建查询改写函数。
func BuildEinoQueryRewriter(cfg appconfig.AIConfig) QueryRewriter {
	if strings.TrimSpace(cfg.APIKey) == "" || strings.TrimSpace(cfg.Model) == "" {
		return nil
	}

	model, err := einoopenai.NewChatModel(context.Background(), &einoopenai.ChatModelConfig{
		APIKey:  cfg.APIKey,
		BaseURL: cfg.BaseURL,
		Model:   cfg.Model,
		Timeout: cfg.Timeout,
	})
	if err != nil {
		return nil
	}

	options := buildWorkflowOptions(cfg.Workflow)
	return func(ctx context.Context, question string) (string, error) {
		resp, err := model.Generate(ctx, []*schema.Message{
			schema.SystemMessage(options.rewritePrompt),
			schema.UserMessage(strings.TrimSpace(question)),
		})
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(resp.Content), nil
	}
}

// BuildEinoToolParamExtractor 基于 Eino 模型创建参数提取函数。
func BuildEinoToolParamExtractor(cfg appconfig.AIConfig) ToolParamExtractor {
	if strings.TrimSpace(cfg.APIKey) == "" || strings.TrimSpace(cfg.Model) == "" {
		return nil
	}
	options := buildWorkflowOptions(cfg.Workflow)

	model, err := einoopenai.NewChatModel(context.Background(), &einoopenai.ChatModelConfig{
		APIKey:  cfg.APIKey,
		BaseURL: cfg.BaseURL,
		Model:   cfg.Model,
		Timeout: cfg.Timeout,
	})
	if err != nil {
		return nil
	}

	return func(ctx context.Context, schemaDef mcpToolSchema, question string) (map[string]any, error) {
		userPrompt := "工具定义如下：\n" + buildToolDefinitionPrompt(schemaDef) + "\n请根据以上工具定义，从下面的问题中提取参数：\n" + question
		resp, err := model.Generate(ctx, []*schema.Message{
			schema.SystemMessage(options.toolParamPrompt),
			schema.UserMessage(userPrompt),
		})
		if err != nil {
			return nil, err
		}
		raw := strings.TrimSpace(resp.Content)
		raw = strings.TrimPrefix(raw, "```json")
		raw = strings.TrimPrefix(raw, "```")
		raw = strings.TrimSuffix(raw, "```")
		raw = strings.TrimSpace(raw)
		if json.Valid([]byte(raw)) {
			return parseToolArguments(raw, schemaDef)
		}
		start := strings.Index(raw, "{")
		end := strings.LastIndex(raw, "}")
		if start >= 0 && end > start {
			return parseToolArguments(raw[start:end+1], schemaDef)
		}
		return map[string]any{}, nil
	}
}
