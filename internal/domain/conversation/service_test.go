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

type memoryConversationRepository struct {
	sessions  []Session
	messages  []Message
	feedbacks []Feedback
}

func (r *memoryConversationRepository) LoadConversations() ([]Session, []Message, []Feedback, error) {
	sessions := make([]Session, len(r.sessions))
	copy(sessions, r.sessions)
	messages := make([]Message, len(r.messages))
	copy(messages, r.messages)
	feedbacks := make([]Feedback, len(r.feedbacks))
	copy(feedbacks, r.feedbacks)
	return sessions, messages, feedbacks, nil
}

func (r *memoryConversationRepository) SaveConversations(sessions []Session, messages []Message, feedbacks []Feedback) error {
	r.sessions = make([]Session, len(sessions))
	copy(r.sessions, sessions)
	r.messages = make([]Message, len(messages))
	copy(r.messages, messages)
	r.feedbacks = make([]Feedback, len(feedbacks))
	copy(r.feedbacks, feedbacks)
	return nil
}

func TestServiceLoadsAndPersistsThroughRepository(t *testing.T) {
	now := time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC)
	repo := &memoryConversationRepository{
		sessions: []Session{
			{
				ConversationID: "conv_db",
				UserID:         "u_admin",
				Title:          "数据库会话",
				LastTime:       now,
			},
		},
		messages: []Message{
			{
				ID:             "msg_db",
				ConversationID: "conv_db",
				UserID:         "u_admin",
				Role:           "assistant",
				Content:        "数据库消息",
				CreateTime:     now,
			},
		},
	}
	service := NewServiceWithRepository(repo)

	sessions := service.ListByUserID("u_admin")
	if len(sessions) != 1 || sessions[0].Title != "数据库会话" {
		t.Fatalf("expected repository session, got %#v", sessions)
	}
	messages := service.ListMessages("conv_db", "u_admin")
	if len(messages) != 1 || messages[0].Content != "数据库消息" {
		t.Fatalf("expected repository message, got %#v", messages)
	}
	service.AppendMessage(Message{
		ID:             "msg_new",
		ConversationID: "conv_db",
		UserID:         "u_admin",
		Role:           "user",
		Content:        "新增消息",
		CreateTime:     now.Add(time.Minute),
	})
	if _, ok := findConversationMessage(repo.messages, "msg_new"); !ok {
		t.Fatal("expected appended message to be saved through repository")
	}
}

func findConversationMessage(messages []Message, id string) (Message, bool) {
	for _, message := range messages {
		if message.ID == id {
			return message, true
		}
	}
	return Message{}, false
}
