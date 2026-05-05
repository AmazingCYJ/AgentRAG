package db

import (
	"database/sql"
	"testing"
	"time"

	domainconversation "github.com/AmazingCYJ/AgentRAG/internal/domain/conversation"
	_ "modernc.org/sqlite"
)

func TestSQLConversationRepositorySavesAndLoadsRecords(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite database failed: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	repository := NewSQLConversationRepository(database)
	if err := repository.Bootstrap(); err != nil {
		t.Fatalf("bootstrap conversation tables failed: %v", err)
	}
	now := time.Date(2026, 5, 1, 10, 30, 0, 0, time.UTC)
	thinkingDuration := 120
	vote := 1
	if err := repository.SaveConversations(
		[]domainconversation.Session{
			{
				ConversationID: "conv_1",
				UserID:         "u_admin",
				Title:          "SQL 会话",
				LastTime:       now,
			},
		},
		[]domainconversation.Message{
			{
				ID:               "msg_1",
				ConversationID:   "conv_1",
				UserID:           "u_admin",
				Role:             "assistant",
				Content:          "SQL 消息",
				ThinkingContent:  "SQL 思考",
				ThinkingDuration: &thinkingDuration,
				Vote:             &vote,
				CreateTime:       now.Add(time.Second),
			},
		},
	); err != nil {
		t.Fatalf("save conversations failed: %v", err)
	}

	sessions, messages, err := repository.LoadConversations()
	if err != nil {
		t.Fatalf("load conversations failed: %v", err)
	}
	if len(sessions) != 1 || sessions[0].Title != "SQL 会话" {
		t.Fatalf("unexpected sessions %#v", sessions)
	}
	if len(messages) != 1 || messages[0].Content != "SQL 消息" {
		t.Fatalf("unexpected messages %#v", messages)
	}
	if messages[0].ThinkingDuration == nil || *messages[0].ThinkingDuration != thinkingDuration {
		t.Fatalf("unexpected thinking duration %#v", messages[0].ThinkingDuration)
	}
	if messages[0].Vote == nil || *messages[0].Vote != vote {
		t.Fatalf("unexpected vote %#v", messages[0].Vote)
	}
}
