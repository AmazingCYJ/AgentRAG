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
    conversation_id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    title TEXT NOT NULL,
    last_time TIMESTAMP NOT NULL
)`); err != nil {
		return err
	}
	_, err := r.database.Exec(`
CREATE TABLE IF NOT EXISTS agentrag_conversation_messages (
    id TEXT PRIMARY KEY,
    conversation_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    role TEXT NOT NULL,
    content TEXT NOT NULL,
    thinking_content TEXT NOT NULL DEFAULT '',
    thinking_duration INTEGER NULL,
    vote INTEGER NULL,
    create_time TIMESTAMP NOT NULL
)`)
	return err
}

// LoadConversations 从数据库加载全部会话和消息。
func (r *SQLConversationRepository) LoadConversations() ([]domainconversation.Session, []domainconversation.Message, error) {
	sessions, err := r.loadSessions()
	if err != nil {
		return nil, nil, err
	}
	messages, err := r.loadMessages()
	if err != nil {
		return nil, nil, err
	}
	return sessions, messages, nil
}

func (r *SQLConversationRepository) loadSessions() ([]domainconversation.Session, error) {
	rows, err := r.database.Query(`
SELECT conversation_id, user_id, title, last_time
FROM agentrag_conversations
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
SELECT id, conversation_id, user_id, role, content, thinking_content, thinking_duration, vote, create_time
FROM agentrag_conversation_messages
ORDER BY user_id ASC, conversation_id ASC, create_time ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	messages := []domainconversation.Message{}
	for rows.Next() {
		var message domainconversation.Message
		var thinkingDuration sql.NullInt64
		var vote sql.NullInt64
		if err := rows.Scan(&message.ID, &message.ConversationID, &message.UserID, &message.Role, &message.Content, &message.ThinkingContent, &thinkingDuration, &vote, &message.CreateTime); err != nil {
			return nil, err
		}
		if thinkingDuration.Valid {
			value := int(thinkingDuration.Int64)
			message.ThinkingDuration = &value
		}
		if vote.Valid {
			value := int(vote.Int64)
			message.Vote = &value
		}
		messages = append(messages, message)
	}
	return messages, rows.Err()
}

// SaveConversations 覆盖保存当前会话和消息集合。
func (r *SQLConversationRepository) SaveConversations(sessions []domainconversation.Session, messages []domainconversation.Message) error {
	tx, err := r.database.Begin()
	if err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM agentrag_conversation_messages`); err != nil {
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
	return tx.Commit()
}

func saveSessions(tx *sql.Tx, sessions []domainconversation.Session) error {
	stmt, err := tx.Prepare(`
INSERT INTO agentrag_conversations (conversation_id, user_id, title, last_time)
VALUES (?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, session := range sessions {
		if _, err := stmt.Exec(session.ConversationID, session.UserID, session.Title, normalizeTime(session.LastTime)); err != nil {
			return err
		}
	}
	return nil
}

func saveMessages(tx *sql.Tx, messages []domainconversation.Message) error {
	stmt, err := tx.Prepare(`
INSERT INTO agentrag_conversation_messages (id, conversation_id, user_id, role, content, thinking_content, thinking_duration, vote, create_time)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`)
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
			nullInt(message.Vote),
			normalizeTime(message.CreateTime),
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
