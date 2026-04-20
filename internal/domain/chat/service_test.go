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
}

func (g *fakeGenerator) Generate(_ context.Context, _ GenerateRequest) (GenerateResult, error) {
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
