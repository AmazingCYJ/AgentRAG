package db

import (
	"database/sql"

	domainconversation "github.com/AmazingCYJ/AgentRAG/internal/domain/conversation"
)

// SQLConversationRepository 使用关系型数据库持久化会话和消息。
type SQLConversationRepository struct {
	database *sql.DB
}

// NewSQLConversationRepository 创建会话 SQL 仓储。
func NewSQLConversationRepository(database *sql.DB) *SQLConversationRepository {
	return &SQLConversationRepository{database: database}
}

// Bootstrap 初始化会话和消息表结构。
func (r *SQLConversationRepository) Bootstrap() error {
	if _, err := r.database.Exec(`
CREATE TABLE IF NOT EXISTS agentrag_conversations (
    id TEXT PRIMARY KEY,
    conversation_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    title TEXT NOT NULL,
    last_time TIMESTAMP,
    create_time TIMESTAMP NOT NULL,
    update_time TIMESTAMP NOT NULL,
    deleted INTEGER NOT NULL DEFAULT 0,
    UNIQUE (conversation_id, user_id)
)`); err != nil {
		return err
	}
	r.migrateLegacyConversationTables()
	if _, err := r.database.Exec(`
CREATE INDEX IF NOT EXISTS idx_agentrag_conversations_user_time
ON agentrag_conversations (user_id, last_time)`); err != nil {
		return err
	}
	if _, err := r.database.Exec(`
CREATE TABLE IF NOT EXISTS agentrag_conversation_summaries (
    id TEXT PRIMARY KEY,
    conversation_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    last_message_id TEXT NOT NULL,
    content TEXT NOT NULL,
    create_time TIMESTAMP NOT NULL,
    update_time TIMESTAMP NOT NULL,
    deleted INTEGER NOT NULL DEFAULT 0
)`); err != nil {
		return err
	}
	if _, err := r.database.Exec(`
CREATE INDEX IF NOT EXISTS idx_agentrag_conversation_summaries_conv_user
ON agentrag_conversation_summaries (conversation_id, user_id)`); err != nil {
		return err
	}
	if _, err := r.database.Exec(`
CREATE TABLE IF NOT EXISTS agentrag_conversation_messages (
    id TEXT PRIMARY KEY,
    conversation_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    role TEXT NOT NULL,
    content TEXT NOT NULL,
    thinking_content TEXT NOT NULL DEFAULT '',
    thinking_duration INTEGER NULL,
    create_time TIMESTAMP NOT NULL,
    update_time TIMESTAMP NOT NULL,
    deleted INTEGER NOT NULL DEFAULT 0
)`); err != nil {
		return err
	}
	if _, err := r.database.Exec(`
CREATE INDEX IF NOT EXISTS idx_agentrag_conversation_messages_conv_user_time
ON agentrag_conversation_messages (conversation_id, user_id, create_time)`); err != nil {
		return err
	}
	_, err := r.database.Exec(`
CREATE TABLE IF NOT EXISTS agentrag_message_feedback (
    id TEXT PRIMARY KEY,
    message_id TEXT NOT NULL,
    conversation_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    vote INTEGER NOT NULL,
    reason TEXT NOT NULL DEFAULT '',
    comment TEXT NOT NULL DEFAULT '',
    create_time TIMESTAMP NOT NULL,
    update_time TIMESTAMP NOT NULL,
    deleted INTEGER NOT NULL DEFAULT 0,
    UNIQUE (message_id, user_id)
)`)
	if err != nil {
		return err
	}
	return r.migrateLegacyFeedback()
}

// LoadConversations 从数据库加载全部会话和消息。
func (r *SQLConversationRepository) LoadConversations() ([]domainconversation.Session, []domainconversation.Message, []domainconversation.Feedback, error) {
	sessions, err := r.loadSessions()
	if err != nil {
		return nil, nil, nil, err
	}
	messages, err := r.loadMessages()
	if err != nil {
		return nil, nil, nil, err
	}
	feedbacks, err := r.loadFeedbacks()
	if err != nil {
		return nil, nil, nil, err
	}
	applyFeedbackVotes(messages, feedbacks)
	return sessions, messages, feedbacks, nil
}

func (r *SQLConversationRepository) loadSessions() ([]domainconversation.Session, error) {
	rows, err := r.database.Query(`
SELECT conversation_id, user_id, title, last_time
FROM agentrag_conversations
WHERE deleted = 0
ORDER BY user_id ASC, last_time DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sessions := []domainconversation.Session{}
	for rows.Next() {
		var session domainconversation.Session
		if err := rows.Scan(&session.ConversationID, &session.UserID, &session.Title, &session.LastTime); err != nil {
			return nil, err
		}
		sessions = append(sessions, session)
	}
	return sessions, rows.Err()
}

func (r *SQLConversationRepository) loadMessages() ([]domainconversation.Message, error) {
	rows, err := r.database.Query(`
SELECT id, conversation_id, user_id, role, content, thinking_content, thinking_duration, create_time
FROM agentrag_conversation_messages
WHERE deleted = 0
ORDER BY user_id ASC, conversation_id ASC, create_time ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	messages := []domainconversation.Message{}
	for rows.Next() {
		var message domainconversation.Message
		var thinkingDuration sql.NullInt64
		if err := rows.Scan(&message.ID, &message.ConversationID, &message.UserID, &message.Role, &message.Content, &message.ThinkingContent, &thinkingDuration, &message.CreateTime); err != nil {
			return nil, err
		}
		if thinkingDuration.Valid {
			value := int(thinkingDuration.Int64)
			message.ThinkingDuration = &value
		}
		messages = append(messages, message)
	}
	return messages, rows.Err()
}

func (r *SQLConversationRepository) loadFeedbacks() ([]domainconversation.Feedback, error) {
	rows, err := r.database.Query(`
SELECT id, message_id, conversation_id, user_id, vote, reason, comment, create_time, update_time
FROM agentrag_message_feedback
WHERE deleted = 0
ORDER BY user_id ASC, message_id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	feedbacks := []domainconversation.Feedback{}
	for rows.Next() {
		var feedback domainconversation.Feedback
		if err := rows.Scan(&feedback.ID, &feedback.MessageID, &feedback.ConversationID, &feedback.UserID, &feedback.Vote, &feedback.Reason, &feedback.Comment, &feedback.CreateTime, &feedback.UpdateTime); err != nil {
			return nil, err
		}
		feedbacks = append(feedbacks, feedback)
	}
	return feedbacks, rows.Err()
}

// SaveConversations 覆盖保存当前会话和消息集合。
func (r *SQLConversationRepository) SaveConversations(sessions []domainconversation.Session, messages []domainconversation.Message, feedbacks []domainconversation.Feedback) error {
	tx, err := r.database.Begin()
	if err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM agentrag_message_feedback`); err != nil {
		_ = tx.Rollback()
		return err
	}
	if _, err := tx.Exec(`DELETE FROM agentrag_conversation_messages`); err != nil {
		_ = tx.Rollback()
		return err
	}
	if _, err := tx.Exec(`DELETE FROM agentrag_conversation_summaries`); err != nil {
		_ = tx.Rollback()
		return err
	}
	if _, err := tx.Exec(`DELETE FROM agentrag_conversations`); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := saveSessions(tx, sessions); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := saveMessages(tx, messages); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := saveFeedbacks(tx, feedbacks); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func saveSessions(tx *sql.Tx, sessions []domainconversation.Session) error {
	stmt, err := tx.Prepare(`
INSERT INTO agentrag_conversations (id, conversation_id, user_id, title, last_time, create_time, update_time, deleted)
VALUES (?, ?, ?, ?, ?, ?, ?, 0)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, session := range sessions {
		lastTime := normalizeTime(session.LastTime)
		if _, err := stmt.Exec(session.ConversationID+"_"+session.UserID, session.ConversationID, session.UserID, session.Title, lastTime, lastTime, lastTime); err != nil {
			return err
		}
	}
	return nil
}

func saveMessages(tx *sql.Tx, messages []domainconversation.Message) error {
	stmt, err := tx.Prepare(`
INSERT INTO agentrag_conversation_messages
    (id, conversation_id, user_id, role, content, thinking_content, thinking_duration, create_time, update_time, deleted)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 0)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, message := range messages {
		if _, err := stmt.Exec(
			message.ID,
			message.ConversationID,
			message.UserID,
			message.Role,
			message.Content,
			message.ThinkingContent,
			nullInt(message.ThinkingDuration),
			normalizeTime(message.CreateTime),
			normalizeTime(message.CreateTime),
		); err != nil {
			return err
		}
	}
	return nil
}

func saveFeedbacks(tx *sql.Tx, feedbacks []domainconversation.Feedback) error {
	stmt, err := tx.Prepare(`
INSERT INTO agentrag_message_feedback
    (id, message_id, conversation_id, user_id, vote, reason, comment, create_time, update_time, deleted)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 0)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, feedback := range feedbacks {
		if _, err := stmt.Exec(
			feedback.ID,
			feedback.MessageID,
			feedback.ConversationID,
			feedback.UserID,
			feedback.Vote,
			feedback.Reason,
			feedback.Comment,
			normalizeTime(feedback.CreateTime),
			normalizeTime(feedback.UpdateTime),
		); err != nil {
			return err
		}
	}
	return nil
}

func nullInt(value *int) any {
	if value == nil {
		return nil
	}
	return *value
}

func applyFeedbackVotes(messages []domainconversation.Message, feedbacks []domainconversation.Feedback) {
	feedbackByKey := make(map[string]domainconversation.Feedback, len(feedbacks))
	for _, feedback := range feedbacks {
		feedbackByKey[feedback.UserID+"|"+feedback.MessageID] = feedback
	}
	for index := range messages {
		feedback, ok := feedbackByKey[messages[index].UserID+"|"+messages[index].ID]
		if !ok {
			continue
		}
		vote := feedback.Vote
		messages[index].Vote = &vote
	}
}

func (r *SQLConversationRepository) migrateLegacyConversationTables() {
	for _, statement := range []string{
		`ALTER TABLE agentrag_conversations ADD COLUMN id TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE agentrag_conversations ADD COLUMN create_time TIMESTAMP`,
		`ALTER TABLE agentrag_conversations ADD COLUMN update_time TIMESTAMP`,
		`ALTER TABLE agentrag_conversations ADD COLUMN deleted INTEGER NOT NULL DEFAULT 0`,
		`UPDATE agentrag_conversations SET id = conversation_id || '_' || user_id WHERE id = ''`,
		`UPDATE agentrag_conversations SET create_time = last_time WHERE create_time IS NULL`,
		`UPDATE agentrag_conversations SET update_time = last_time WHERE update_time IS NULL`,
		`ALTER TABLE agentrag_conversation_messages ADD COLUMN update_time TIMESTAMP`,
		`ALTER TABLE agentrag_conversation_messages ADD COLUMN deleted INTEGER NOT NULL DEFAULT 0`,
		`UPDATE agentrag_conversation_messages SET update_time = create_time WHERE update_time IS NULL`,
	} {
		_, _ = r.database.Exec(statement)
	}
}

func (r *SQLConversationRepository) migrateLegacyFeedback() error {
	if !columnExists(r.database, "agentrag_conversation_messages", "vote") {
		return nil
	}
	_, err := r.database.Exec(`
INSERT OR IGNORE INTO agentrag_message_feedback
    (id, message_id, conversation_id, user_id, vote, reason, comment, create_time, update_time, deleted)
SELECT id || '_' || user_id, id, conversation_id, user_id, vote, '', '', create_time, create_time, 0
FROM agentrag_conversation_messages
WHERE vote IS NOT NULL`)
	return err
}
