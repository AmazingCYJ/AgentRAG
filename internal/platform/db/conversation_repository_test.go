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
		[]domainconversation.Feedback{
			{
				ID:             "fb_1",
				MessageID:      "msg_1",
				ConversationID: "conv_1",
				UserID:         "u_admin",
				Vote:           vote,
				Reason:         "useful",
				Comment:        "answer is clear",
				CreateTime:     now.Add(2 * time.Second),
				UpdateTime:     now.Add(2 * time.Second),
			},
		},
	); err != nil {
		t.Fatalf("save conversations failed: %v", err)
	}

	sessions, messages, feedbacks, err := repository.LoadConversations()
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
	if len(feedbacks) != 1 || feedbacks[0].Reason != "useful" || feedbacks[0].Comment != "answer is clear" {
		t.Fatalf("unexpected feedbacks %#v", feedbacks)
	}
}

func TestSQLConversationRepositoryBootstrapsLegacyTables(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite database failed: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	now := time.Date(2026, 5, 6, 10, 0, 0, 0, time.UTC)
	if _, err := database.Exec(`
CREATE TABLE agentrag_conversations (
    conversation_id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    title TEXT NOT NULL,
    last_time TIMESTAMP NOT NULL
)`); err != nil {
		t.Fatalf("create legacy conversations table failed: %v", err)
	}
	if _, err := database.Exec(`
CREATE TABLE agentrag_conversation_messages (
    id TEXT PRIMARY KEY,
    conversation_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    role TEXT NOT NULL,
    content TEXT NOT NULL,
    thinking_content TEXT NOT NULL DEFAULT '',
    thinking_duration INTEGER NULL,
    vote INTEGER NULL,
    create_time TIMESTAMP NOT NULL
)`); err != nil {
		t.Fatalf("create legacy messages table failed: %v", err)
	}
	if _, err := database.Exec(`INSERT INTO agentrag_conversations (conversation_id, user_id, title, last_time) VALUES (?, ?, ?, ?)`, "conv_legacy", "u_admin", "旧会话", now); err != nil {
		t.Fatalf("insert legacy conversation failed: %v", err)
	}
	if _, err := database.Exec(`INSERT INTO agentrag_conversation_messages (id, conversation_id, user_id, role, content, vote, create_time) VALUES (?, ?, ?, ?, ?, ?, ?)`, "msg_legacy", "conv_legacy", "u_admin", "assistant", "旧消息", 1, now); err != nil {
		t.Fatalf("insert legacy message failed: %v", err)
	}

	repository := NewSQLConversationRepository(database)
	if err := repository.Bootstrap(); err != nil {
		t.Fatalf("bootstrap legacy conversation tables failed: %v", err)
	}
	sessions, messages, feedbacks, err := repository.LoadConversations()
	if err != nil {
		t.Fatalf("load migrated conversations failed: %v", err)
	}
	if len(sessions) != 1 || sessions[0].Title != "旧会话" {
		t.Fatalf("unexpected migrated sessions %#v", sessions)
	}
	if len(messages) != 1 || messages[0].Content != "旧消息" || messages[0].Vote == nil || *messages[0].Vote != 1 {
		t.Fatalf("unexpected migrated messages %#v", messages)
	}
	if len(feedbacks) != 1 || feedbacks[0].MessageID != "msg_legacy" || feedbacks[0].Vote != 1 {
		t.Fatalf("unexpected migrated feedbacks %#v", feedbacks)
	}
}
