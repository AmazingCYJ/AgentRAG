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

func TestSQLConversationRepositoryReconcilesSavedRecords(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite database failed: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	repository := NewSQLConversationRepository(database)
	if err := repository.Bootstrap(); err != nil {
		t.Fatalf("bootstrap conversation tables failed: %v", err)
	}
	now := time.Date(2026, 5, 10, 15, 0, 0, 0, time.UTC)
	thinkingDuration := 120
	vote := 1
	if err := repository.SaveConversations(
		[]domainconversation.Session{
			{ConversationID: "conv_keep", UserID: "u_admin", Title: "旧会话", LastTime: now},
			{ConversationID: "conv_remove", UserID: "u_admin", Title: "删除会话", LastTime: now.Add(time.Minute)},
		},
		[]domainconversation.Message{
			{ID: "msg_keep", ConversationID: "conv_keep", UserID: "u_admin", Role: "assistant", Content: "旧消息", ThinkingDuration: &thinkingDuration, CreateTime: now.Add(time.Second)},
			{ID: "msg_remove", ConversationID: "conv_remove", UserID: "u_admin", Role: "assistant", Content: "删除消息", CreateTime: now.Add(2 * time.Second)},
		},
		[]domainconversation.Feedback{
			{ID: "fb_keep", MessageID: "msg_keep", ConversationID: "conv_keep", UserID: "u_admin", Vote: vote, Reason: "old", Comment: "old comment", CreateTime: now.Add(3 * time.Second), UpdateTime: now.Add(3 * time.Second)},
			{ID: "fb_remove", MessageID: "msg_remove", ConversationID: "conv_remove", UserID: "u_admin", Vote: vote, Reason: "remove", CreateTime: now.Add(4 * time.Second), UpdateTime: now.Add(4 * time.Second)},
		},
	); err != nil {
		t.Fatalf("save initial conversation snapshot failed: %v", err)
	}
	if _, err := database.Exec(`
INSERT INTO agentrag_conversation_summaries
    (id, conversation_id, user_id, last_message_id, content, create_time, update_time, deleted)
VALUES (?, ?, ?, ?, ?, ?, ?, 0)`, "summary_keep", "conv_keep", "u_admin", "msg_keep", "summary", now, now); err != nil {
		t.Fatalf("insert conversation summary failed: %v", err)
	}

	updatedThinkingDuration := 240
	updatedVote := -1
	if err := repository.SaveConversations(
		[]domainconversation.Session{
			{ConversationID: "conv_keep", UserID: "u_admin", Title: "新会话", LastTime: now.Add(10 * time.Minute)},
			{ConversationID: "conv_new", UserID: "u_admin", Title: "新增会话", LastTime: now.Add(11 * time.Minute)},
		},
		[]domainconversation.Message{
			{ID: "msg_keep", ConversationID: "conv_keep", UserID: "u_admin", Role: "assistant", Content: "新消息", ThinkingContent: "新思考", ThinkingDuration: &updatedThinkingDuration, CreateTime: now.Add(time.Second)},
			{ID: "msg_new", ConversationID: "conv_new", UserID: "u_admin", Role: "user", Content: "新增消息", CreateTime: now.Add(12 * time.Minute)},
		},
		[]domainconversation.Feedback{
			{ID: "fb_keep", MessageID: "msg_keep", ConversationID: "conv_keep", UserID: "u_admin", Vote: updatedVote, Reason: "updated", Comment: "updated comment", CreateTime: now.Add(3 * time.Second), UpdateTime: now.Add(13 * time.Minute)},
			{ID: "fb_new", MessageID: "msg_new", ConversationID: "conv_new", UserID: "u_admin", Vote: vote, Reason: "new", CreateTime: now.Add(14 * time.Minute), UpdateTime: now.Add(14 * time.Minute)},
		},
	); err != nil {
		t.Fatalf("save reconciled conversation snapshot failed: %v", err)
	}

	sessions, messages, feedbacks, err := repository.LoadConversations()
	if err != nil {
		t.Fatalf("load reconciled conversations failed: %v", err)
	}
	if len(sessions) != 2 || len(messages) != 2 || len(feedbacks) != 2 {
		t.Fatalf("expected 2 sessions/messages/feedbacks after reconcile, got sessions=%#v messages=%#v feedbacks=%#v", sessions, messages, feedbacks)
	}
	sessionsByID := map[string]domainconversation.Session{}
	for _, session := range sessions {
		sessionsByID[session.ConversationID] = session
	}
	if _, ok := sessionsByID["conv_remove"]; ok {
		t.Fatal("expected missing conversation to be deleted")
	}
	if sessionsByID["conv_keep"].Title != "新会话" || sessionsByID["conv_new"].Title != "新增会话" {
		t.Fatalf("expected conversations to be updated and inserted, got %#v", sessionsByID)
	}
	messagesByID := map[string]domainconversation.Message{}
	for _, message := range messages {
		messagesByID[message.ID] = message
	}
	if _, ok := messagesByID["msg_remove"]; ok {
		t.Fatal("expected missing message to be deleted")
	}
	if messagesByID["msg_keep"].Content != "新消息" || messagesByID["msg_keep"].ThinkingContent != "新思考" {
		t.Fatalf("expected existing message to be updated, got %#v", messagesByID["msg_keep"])
	}
	if messagesByID["msg_keep"].ThinkingDuration == nil || *messagesByID["msg_keep"].ThinkingDuration != updatedThinkingDuration {
		t.Fatalf("expected updated thinking duration, got %#v", messagesByID["msg_keep"].ThinkingDuration)
	}
	if messagesByID["msg_keep"].Vote == nil || *messagesByID["msg_keep"].Vote != updatedVote {
		t.Fatalf("expected feedback vote to be applied to message, got %#v", messagesByID["msg_keep"].Vote)
	}
	feedbacksByID := map[string]domainconversation.Feedback{}
	for _, feedback := range feedbacks {
		feedbacksByID[feedback.ID] = feedback
	}
	if _, ok := feedbacksByID["fb_remove"]; ok {
		t.Fatal("expected missing feedback to be deleted")
	}
	if feedbacksByID["fb_keep"].Reason != "updated" || feedbacksByID["fb_keep"].Comment != "updated comment" {
		t.Fatalf("expected existing feedback to be updated, got %#v", feedbacksByID["fb_keep"])
	}

	var summaryCount int
	if err := database.QueryRow(`SELECT COUNT(*) FROM agentrag_conversation_summaries`).Scan(&summaryCount); err != nil {
		t.Fatalf("count conversation summaries failed: %v", err)
	}
	if summaryCount != 0 {
		t.Fatalf("expected conversation summaries to be cleared, got %d", summaryCount)
	}

	if err := repository.SaveConversations(nil, nil, nil); err != nil {
		t.Fatalf("save empty conversation snapshot failed: %v", err)
	}
	sessions, messages, feedbacks, err = repository.LoadConversations()
	if err != nil {
		t.Fatalf("load empty conversation snapshot failed: %v", err)
	}
	if len(sessions) != 0 || len(messages) != 0 || len(feedbacks) != 0 {
		t.Fatalf("expected empty snapshot to clear records, got sessions=%#v messages=%#v feedbacks=%#v", sessions, messages, feedbacks)
	}
}

func TestSQLConversationRepositoryRejectsDuplicateSnapshotIDs(t *testing.T) {
	now := time.Date(2026, 5, 10, 16, 0, 0, 0, time.UTC)
	testCases := []struct {
		name      string
		sessions  []domainconversation.Session
		messages  []domainconversation.Message
		feedbacks []domainconversation.Feedback
	}{
		{
			name: "sessions",
			sessions: []domainconversation.Session{
				{ConversationID: "conv_keep", UserID: "u_admin", Title: "新会话", LastTime: now},
				{ConversationID: "conv_keep", UserID: "u_admin", Title: "重复会话", LastTime: now},
			},
		},
		{
			name:     "messages",
			sessions: []domainconversation.Session{{ConversationID: "conv_keep", UserID: "u_admin", Title: "会话", LastTime: now}},
			messages: []domainconversation.Message{
				{ID: "msg_keep", ConversationID: "conv_keep", UserID: "u_admin", Role: "assistant", Content: "新消息", CreateTime: now},
				{ID: "msg_keep", ConversationID: "conv_keep", UserID: "u_admin", Role: "assistant", Content: "重复消息", CreateTime: now},
			},
		},
		{
			name:     "feedbacks",
			sessions: []domainconversation.Session{{ConversationID: "conv_keep", UserID: "u_admin", Title: "会话", LastTime: now}},
			messages: []domainconversation.Message{{ID: "msg_keep", ConversationID: "conv_keep", UserID: "u_admin", Role: "assistant", Content: "消息", CreateTime: now}},
			feedbacks: []domainconversation.Feedback{
				{ID: "fb_keep", MessageID: "msg_keep", ConversationID: "conv_keep", UserID: "u_admin", Vote: 1, CreateTime: now, UpdateTime: now},
				{ID: "fb_keep", MessageID: "msg_keep", ConversationID: "conv_keep", UserID: "u_admin", Vote: -1, CreateTime: now, UpdateTime: now},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			database, err := sql.Open("sqlite", ":memory:")
			if err != nil {
				t.Fatalf("open sqlite database failed: %v", err)
			}
			t.Cleanup(func() { _ = database.Close() })

			repository := NewSQLConversationRepository(database)
			if err := repository.Bootstrap(); err != nil {
				t.Fatalf("bootstrap conversation tables failed: %v", err)
			}
			if err := repository.SaveConversations(
				[]domainconversation.Session{{ConversationID: "conv_old", UserID: "u_admin", Title: "旧会话", LastTime: now}},
				[]domainconversation.Message{{ID: "msg_old", ConversationID: "conv_old", UserID: "u_admin", Role: "assistant", Content: "旧消息", CreateTime: now}},
				[]domainconversation.Feedback{{ID: "fb_old", MessageID: "msg_old", ConversationID: "conv_old", UserID: "u_admin", Vote: 1, CreateTime: now, UpdateTime: now}},
			); err != nil {
				t.Fatalf("save initial conversation snapshot failed: %v", err)
			}

			err = repository.SaveConversations(testCase.sessions, testCase.messages, testCase.feedbacks)
			if err == nil {
				t.Fatal("expected duplicate id error")
			}

			sessions, messages, feedbacks, err := repository.LoadConversations()
			if err != nil {
				t.Fatalf("load conversations after duplicate id failure failed: %v", err)
			}
			if len(sessions) != 1 || sessions[0].ConversationID != "conv_old" || len(messages) != 1 || messages[0].ID != "msg_old" || len(feedbacks) != 1 || feedbacks[0].ID != "fb_old" {
				t.Fatalf("expected existing records to remain unchanged, got sessions=%#v messages=%#v feedbacks=%#v", sessions, messages, feedbacks)
			}
		})
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
