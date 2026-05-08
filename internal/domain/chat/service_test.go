package chat

import (
	"context"
	"strings"
	"testing"
	"time"

	domainconversation "github.com/AmazingCYJ/AgentRAG/internal/domain/conversation"
	domainragtrace "github.com/AmazingCYJ/AgentRAG/internal/domain/ragtrace"
	appconfig "github.com/AmazingCYJ/AgentRAG/internal/platform/config"
)

type fakeWriter struct {
	events []string
}

func (w *fakeWriter) Event(name string, payload any) error {
	w.events = append(w.events, name)
	return nil
}

type fakeGenerator struct {
	thinking string
	answer   string
	lastReq  GenerateRequest
}

type GeneratorFunc func(context.Context, GenerateRequest) (GenerateResult, error)

func (f GeneratorFunc) Generate(ctx context.Context, req GenerateRequest) (GenerateResult, error) {
	return f(ctx, req)
}

func (g *fakeGenerator) Generate(_ context.Context, _ GenerateRequest) (GenerateResult, error) {
	return GenerateResult{
		Thinking: g.thinking,
		Answer:   g.answer,
	}, nil
}

func (g *fakeGenerator) GenerateWithCapture(_ context.Context, req GenerateRequest) (GenerateResult, error) {
	g.lastReq = req
	return GenerateResult{
		Thinking: g.thinking,
		Answer:   g.answer,
	}, nil
}

func TestStreamChatUsesConfiguredGeneratorOutput(t *testing.T) {
	conversationService := domainconversation.NewService(nil)
	service := NewService(
		conversationService,
		nil,
		&fakeGenerator{
			thinking: "这是生成器思考内容",
			answer:   "这是生成器回答内容",
		},
	)
	service.waitFn = func(_ context.Context, _ time.Duration) error { return nil }
	service.now = func() time.Time {
		return time.Date(2026, 4, 20, 12, 0, 0, 0, time.UTC)
	}
	writer := &fakeWriter{}

	err := service.StreamChat(context.Background(), StreamRequest{
		UserID:       "u_admin",
		Username:     "admin",
		Question:     "测试问题",
		DeepThinking: true,
	}, writer)
	if err != nil {
		t.Fatalf("stream chat failed: %v", err)
	}

	messages := conversationService.ListMessages(conversationService.ListByUserID("u_admin")[0].ConversationID, "u_admin")
	if len(messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(messages))
	}
	if messages[1].ThinkingContent != "这是生成器思考内容" {
		t.Fatalf("expected generator thinking content, got %s", messages[1].ThinkingContent)
	}
	if messages[1].Content != "这是生成器回答内容" {
		t.Fatalf("expected generator answer content, got %s", messages[1].Content)
	}
}

func TestStreamChatRecordsGeneratorWorkflowSteps(t *testing.T) {
	conversationService := domainconversation.NewService(nil)
	traceService := domainragtrace.NewService(nil)
	service := NewService(
		conversationService,
		traceService,
		&fakeGenerator{
			answer: "工作流回答",
		},
	)
	service.generator = GeneratorFunc(func(_ context.Context, _ GenerateRequest) (GenerateResult, error) {
		return GenerateResult{
			Answer: "工作流回答",
			Steps: []WorkflowStep{
				{
					NodeID:     "retrieve_context",
					NodeType:   "RETRIEVER",
					NodeName:   "Retrieve Knowledge",
					Status:     "success",
					DurationMs: 12,
					Detail:     "命中知识库",
				},
			},
		}, nil
	})
	service.waitFn = func(_ context.Context, _ time.Duration) error { return nil }
	service.now = func() time.Time {
		return time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC)
	}

	err := service.StreamChat(context.Background(), StreamRequest{
		UserID:   "u_admin",
		Username: "admin",
		Question: "测试工作流步骤",
	}, &fakeWriter{})
	if err != nil {
		t.Fatalf("stream chat failed: %v", err)
	}

	runs := traceService.PageRuns(domainragtrace.RunQuery{Size: 10})
	var traceID string
	for _, run := range runs.Records {
		if run.TraceName == "测试工作流步骤" {
			traceID = run.TraceID
			break
		}
	}
	if traceID == "" {
		t.Fatal("expected trace run for chat")
	}
	nodes, err := traceService.ListNodes(traceID)
	if err != nil {
		t.Fatalf("list trace nodes failed: %v", err)
	}
	for _, node := range nodes {
		if node.NodeID == "retrieve_context" && node.NodeType == "RETRIEVER" && strings.Contains(node.ExtraData, "命中知识库") && node.ErrorMessage == "" {
			return
		}
	}
	t.Fatalf("expected retriever workflow node in trace, got %#v", nodes)
}

type captureGenerator struct {
	lastReq GenerateRequest
}

func (g *captureGenerator) Generate(_ context.Context, req GenerateRequest) (GenerateResult, error) {
	g.lastReq = req
	return GenerateResult{
		Answer: "ok",
	}, nil
}

func TestRetrievalGeneratorInjectsKnowledgeContext(t *testing.T) {
	base := &captureGenerator{}
	generator := wrapWithRetriever(base, func(_ context.Context, query string, limit int) (string, error) {
		if query != "怎么配置请假流程" {
			t.Fatalf("unexpected query %s", query)
		}
		if limit != 4 {
			t.Fatalf("unexpected retrieval limit %d", limit)
		}
		return "文档《OA请假手册》：请先进入审批中心。", nil
	}, 4)

	_, err := generator.Generate(context.Background(), GenerateRequest{
		Question:     "怎么配置请假流程",
		DeepThinking: true,
	})
	if err != nil {
		t.Fatalf("generate with retriever failed: %v", err)
	}
	if base.lastReq.KnowledgeContext == "" {
		t.Fatal("expected knowledge context to be injected")
	}
	if base.lastReq.KnowledgeContext != "文档《OA请假手册》：请先进入审批中心。" {
		t.Fatalf("unexpected knowledge context %s", base.lastReq.KnowledgeContext)
	}
}

func TestFallbackGeneratorUsesKnowledgeContextWhenModelUnavailable(t *testing.T) {
	generator := &fallbackGenerator{}

	result, err := generator.Generate(context.Background(), GenerateRequest{
		Question:         "怎么配置请假流程",
		KnowledgeContext: "文档《OA请假手册》：请先进入审批中心，再选择请假流程模板。",
	})
	if err != nil {
		t.Fatalf("fallback generate failed: %v", err)
	}
	if result.Answer == "" {
		t.Fatal("expected fallback answer")
	}
	if !strings.Contains(result.Answer, "审批中心") {
		t.Fatalf("expected answer to use knowledge context, got %s", result.Answer)
	}
	if strings.Contains(result.Answer, "占位答案") || strings.Contains(result.Answer, "最小可用回答") {
		t.Fatalf("fallback should not return placeholder text when context exists, got %s", result.Answer)
	}
}

func TestRoutingGeneratorUsesToolContextWhenMCPRouteSelected(t *testing.T) {
	base := &captureGenerator{}
	generator := wrapWithRouting(
		base,
		func(_ context.Context, _ string, _ int) (string, error) {
			t.Fatal("retriever should not be called for MCP route")
			return "", nil
		},
		func(_ context.Context, question string) (RouteDecision, error) {
			if question != "北京天气怎么样" {
				t.Fatalf("unexpected question %s", question)
			}
			return RouteDecision{
				Kind:   RouteKindTool,
				ToolID: "weather_query",
			}, nil
		},
		func(_ context.Context, toolID, question string) (string, error) {
			if toolID != "weather_query" {
				t.Fatalf("unexpected tool id %s", toolID)
			}
			if question != "北京天气怎么样" {
				t.Fatalf("unexpected tool question %s", question)
			}
			return "【北京天气】晴 24°C", nil
		},
		4,
	)

	_, err := generator.Generate(context.Background(), GenerateRequest{
		Question: "北京天气怎么样",
	})
	if err != nil {
		t.Fatalf("generate with routing failed: %v", err)
	}
	if base.lastReq.KnowledgeContext != "【北京天气】晴 24°C" {
		t.Fatalf("expected tool context, got %s", base.lastReq.KnowledgeContext)
	}
}

func TestParameterExtractorParsesOnlyDeclaredToolArguments(t *testing.T) {
	rawJSON := `{"city":"北京","queryType":"forecast","unexpected":"drop"}`
	result, err := parseToolArguments(rawJSON, mcpToolSchema{
		toolID: "weather_query",
		parameters: map[string]mcpToolParameter{
			"city": {
				description: "城市名称",
				typ:         "string",
				required:    true,
			},
			"queryType": {
				description: "查询类型",
				typ:         "string",
				required:    false,
			},
		},
	})
	if err != nil {
		t.Fatalf("parse tool arguments failed: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 parsed args, got %d", len(result))
	}
	if result["city"] != "北京" {
		t.Fatalf("expected city 北京, got %#v", result["city"])
	}
	if _, ok := result["unexpected"]; ok {
		t.Fatal("unexpected field should be filtered out")
	}
}

func TestRoutingGeneratorEmitsWorkflowSteps(t *testing.T) {
	base := &captureGenerator{}
	generator := wrapWithRouting(
		base,
		func(_ context.Context, _ string, _ int) (string, error) {
			return "知识上下文", nil
		},
		func(_ context.Context, _ string) (RouteDecision, error) {
			return RouteDecision{Kind: RouteKindKnowledge}, nil
		},
		nil,
		4,
	)

	result, err := generator.Generate(context.Background(), GenerateRequest{
		Question: "请总结知识库内容",
	})
	if err != nil {
		t.Fatalf("generate with workflow steps failed: %v", err)
	}
	if len(result.Steps) < 2 {
		t.Fatalf("expected at least 2 workflow steps, got %d", len(result.Steps))
	}
	if result.Steps[0].NodeType != "ROUTER" {
		t.Fatalf("expected first step ROUTER, got %s", result.Steps[0].NodeType)
	}
	if result.Steps[1].NodeType != "RETRIEVER" {
		t.Fatalf("expected second step RETRIEVER, got %s", result.Steps[1].NodeType)
	}
}

func TestBuildWorkflowOptionsDefaultsInvalidRetrievalLimit(t *testing.T) {
	options := buildWorkflowOptions(appconfig.AIWorkflowConfig{RetrievalLimit: -1})
	if options.retrievalLimit != 4 {
		t.Fatalf("expected default retrieval limit 4, got %d", options.retrievalLimit)
	}
}

func TestNewGeneratorFromConfigUsesConfiguredRetrievalLimit(t *testing.T) {
	usedLimit := 0
	generator := NewGeneratorFromConfig(
		appconfig.AIConfig{
			Workflow: appconfig.AIWorkflowConfig{RetrievalLimit: 7},
		},
		func(_ context.Context, _ string, limit int) (string, error) {
			usedLimit = limit
			return "知识上下文", nil
		},
		func(_ context.Context, _ string) (RouteDecision, error) {
			return RouteDecision{Kind: RouteKindKnowledge}, nil
		},
		nil,
	)

	_, err := generator.Generate(context.Background(), GenerateRequest{Question: "测试问题"})
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}
	if usedLimit != 7 {
		t.Fatalf("expected configured retrieval limit 7, got %d", usedLimit)
	}
}

func TestRewriteGeneratorUsesRewrittenQueryForRoutingAndRetrieval(t *testing.T) {
	base := &captureGenerator{}
	routedQuestion := ""
	retrievedQuery := ""
	generator := wrapWithRewrite(
		wrapWithRouting(
			base,
			func(_ context.Context, query string, _ int) (string, error) {
				retrievedQuery = query
				return "知识上下文", nil
			},
			func(_ context.Context, question string) (RouteDecision, error) {
				routedQuestion = question
				return RouteDecision{Kind: RouteKindKnowledge}, nil
			},
			nil,
			4,
		),
		func(_ context.Context, question string) (string, error) {
			if question != "它怎么配置" {
				t.Fatalf("unexpected original question %s", question)
			}
			return "审批流程怎么配置", nil
		},
	)

	result, err := generator.Generate(context.Background(), GenerateRequest{
		Question: "它怎么配置",
	})
	if err != nil {
		t.Fatalf("generate with rewrite failed: %v", err)
	}
	if routedQuestion != "审批流程怎么配置" {
		t.Fatalf("expected routed rewritten question, got %s", routedQuestion)
	}
	if retrievedQuery != "审批流程怎么配置" {
		t.Fatalf("expected retrieval rewritten query, got %s", retrievedQuery)
	}
	if base.lastReq.Question != "它怎么配置" {
		t.Fatalf("expected final answer to keep original question, got %s", base.lastReq.Question)
	}
	if len(result.Steps) == 0 || result.Steps[0].NodeType != "REWRITER" {
		t.Fatalf("expected first workflow step REWRITER, got %#v", result.Steps)
	}
	if result.Steps[0].Detail != "审批流程怎么配置" {
		t.Fatalf("expected rewrite detail, got %s", result.Steps[0].Detail)
	}
}
