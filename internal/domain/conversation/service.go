package conversation

import (
	"errors"
	"sort"
	"strings"
	"sync"
	"time"
)

var (
	// ErrConversationNotFound 表示会话不存在。
	ErrConversationNotFound = errors.New("会话不存在")
	// ErrMessageNotFound 表示消息不存在。
	ErrMessageNotFound = errors.New("消息不存在")
	// ErrInvalidVote 表示反馈值不合法。
	ErrInvalidVote = errors.New("反馈值必须为 1 或 -1")
)

// Session 表示内部会话记录。
type Session struct {
	ConversationID string
	UserID         string
	Title          string
	LastTime       time.Time
}

// Message 表示内部消息记录。
type Message struct {
	ID               string
	ConversationID   string
	UserID           string
	Role             string
	Content          string
	ThinkingContent  string
	ThinkingDuration *int
	Vote             *int
	CreateTime       time.Time
}

// SessionView 是前端会话列表所需结构。
type SessionView struct {
	ConversationID string    `json:"conversationId"`
	Title          string    `json:"title"`
	LastTime       time.Time `json:"lastTime,omitempty"`
}

// MessageView 是前端消息列表所需结构。
type MessageView struct {
	ID               string     `json:"id"`
	ConversationID   string     `json:"conversationId"`
	Role             string     `json:"role"`
	Content          string     `json:"content"`
	ThinkingContent  string     `json:"thinkingContent,omitempty"`
	ThinkingDuration *int       `json:"thinkingDuration,omitempty"`
	Vote             *int       `json:"vote"`
	CreateTime       *time.Time `json:"createTime,omitempty"`
}

// Service 提供当前阶段最小可用的会话与消息内存存储。
type Service struct {
	mu       sync.RWMutex
	sessions map[string]map[string]Session
	messages map[string]map[string][]Message
}

// NewService 创建会话服务。
func NewService() *Service {
	return &Service{
		sessions: make(map[string]map[string]Session),
		messages: make(map[string]map[string][]Message),
	}
}

// UpsertConversation 写入或更新会话记录。
func (s *Service) UpsertConversation(session Session) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.sessions[session.UserID] == nil {
		s.sessions[session.UserID] = make(map[string]Session)
	}
	s.sessions[session.UserID][session.ConversationID] = session
}

// AppendMessage 为指定会话追加消息。
func (s *Service) AppendMessage(message Message) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.messages[message.UserID] == nil {
		s.messages[message.UserID] = make(map[string][]Message)
	}
	s.messages[message.UserID][message.ConversationID] = append(
		s.messages[message.UserID][message.ConversationID],
		message,
	)
}

// ListByUserID 返回用户会话列表，按最近时间倒序。
func (s *Service) ListByUserID(userID string) []SessionView {
	s.mu.RLock()
	defer s.mu.RUnlock()

	records := s.sessions[userID]
	if len(records) == 0 {
		return []SessionView{}
	}

	result := make([]SessionView, 0, len(records))
	for _, item := range records {
		result = append(result, SessionView{
			ConversationID: item.ConversationID,
			Title:          item.Title,
			LastTime:       item.LastTime,
		})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].LastTime.After(result[j].LastTime)
	})
	return result
}

// ListMessages 返回指定会话的消息列表，按创建时间正序。
func (s *Service) ListMessages(conversationID, userID string) []MessageView {
	s.mu.RLock()
	defer s.mu.RUnlock()

	records := s.messages[userID][conversationID]
	if len(records) == 0 {
		return []MessageView{}
	}

	result := make([]MessageView, 0, len(records))
	for _, item := range records {
		createTime := item.CreateTime
		result = append(result, MessageView{
			ID:               item.ID,
			ConversationID:   item.ConversationID,
			Role:             item.Role,
			Content:          item.Content,
			ThinkingContent:  item.ThinkingContent,
			ThinkingDuration: item.ThinkingDuration,
			Vote:             item.Vote,
			CreateTime:       &createTime,
		})
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].CreateTime == nil {
			return false
		}
		if result[j].CreateTime == nil {
			return true
		}
		return result[i].CreateTime.Before(*result[j].CreateTime)
	})
	return result
}

// Rename 更新指定会话标题。
func (s *Service) Rename(conversationID, userID, title string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	records := s.sessions[userID]
	if records == nil {
		return ErrConversationNotFound
	}
	session, ok := records[conversationID]
	if !ok {
		return ErrConversationNotFound
	}
	session.Title = strings.TrimSpace(title)
	records[conversationID] = session
	return nil
}

// Delete 删除指定会话以及其全部消息。
func (s *Service) Delete(conversationID, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	records := s.sessions[userID]
	if records == nil {
		return ErrConversationNotFound
	}
	if _, ok := records[conversationID]; !ok {
		return ErrConversationNotFound
	}
	delete(records, conversationID)
	if s.messages[userID] != nil {
		delete(s.messages[userID], conversationID)
	}
	return nil
}

// SubmitFeedback 更新指定消息的点赞或点踩状态。
func (s *Service) SubmitFeedback(messageID, userID string, vote int) error {
	if vote != 1 && vote != -1 {
		return ErrInvalidVote
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	records := s.messages[userID]
	if records == nil {
		return ErrMessageNotFound
	}

	for conversationID, messages := range records {
		for index := range messages {
			if messages[index].ID != messageID {
				continue
			}
			nextVote := vote
			messages[index].Vote = &nextVote
			records[conversationID] = messages
			return nil
		}
	}
	return ErrMessageNotFound
}
