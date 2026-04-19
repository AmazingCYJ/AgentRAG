package conversation

import (
	"path/filepath"
	"testing"
	"time"

	platformstate "github.com/AmazingCYJ/AgentRAG/internal/platform/state"
)

func TestConversationStatePersistsAcrossServiceRecreation(t *testing.T) {
	store, err := platformstate.NewFileStore(filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatalf("create state store failed: %v", err)
	}

	service := NewService(store)
	lastTime := time.Date(2026, 4, 19, 10, 0, 0, 0, time.UTC)
	service.UpsertConversation(Session{
		ConversationID: "conv_1",
		UserID:         "u_admin",
		Title:          "测试会话",
		LastTime:       lastTime,
	})
	service.AppendMessage(Message{
		ID:             "m1",
		ConversationID: "conv_1",
		UserID:         "u_admin",
		Role:           "user",
		Content:        "hello",
		CreateTime:     lastTime,
	})

	recreated := NewService(store)
	sessions := recreated.ListByUserID("u_admin")
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session after recreation, got %d", len(sessions))
	}
	if sessions[0].Title != "测试会话" {
		t.Fatalf("expected title 测试会话, got %s", sessions[0].Title)
	}
	messages := recreated.ListMessages("conv_1", "u_admin")
	if len(messages) != 1 {
		t.Fatalf("expected 1 message after recreation, got %d", len(messages))
	}
	if messages[0].Content != "hello" {
		t.Fatalf("expected content hello, got %s", messages[0].Content)
	}
}
