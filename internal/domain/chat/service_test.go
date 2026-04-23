package chat

import (
	"context"
	"testing"
	"time"

	domainconversation "github.com/AmazingCYJ/AgentRAG/internal/domain/conversation"
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
