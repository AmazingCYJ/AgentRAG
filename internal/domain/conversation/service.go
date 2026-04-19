package conversation

import (
	"errors"
	"sort"
	"strings"
	"sync"
	"time"

	platformstate "github.com/AmazingCYJ/AgentRAG/internal/platform/state"
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
	store    *platformstate.FileStore
}

// StatsSnapshot 定义会话统计快照。
type StatsSnapshot struct {
	TotalSessions  int
	RecentSessions int
	TotalMessages  int
	RecentMessages int
}

// NewService 创建会话服务。
func NewService(store *platformstate.FileStore) *Service {
	service := &Service{
		sessions: make(map[string]map[string]Session),
		messages: make(map[string]map[string][]Message),
		store:    store,
	}
	if snapshot, err := service.loadSnapshot(); err == nil {
		for _, session := range snapshot.ConversationSessions {
			if service.sessions[session.UserID] == nil {
				service.sessions[session.UserID] = make(map[string]Session)
			}
			service.sessions[session.UserID][session.ConversationID] = Session{
				ConversationID: session.ConversationID,
				UserID:         session.UserID,
				Title:          session.Title,
				LastTime:       session.LastTime,
			}
		}
		for _, message := range snapshot.ConversationMessages {
			if service.messages[message.UserID] == nil {
				service.messages[message.UserID] = make(map[string][]Message)
			}
			service.messages[message.UserID][message.ConversationID] = append(service.messages[message.UserID][message.ConversationID], Message{
				ID:               message.ID,
				ConversationID:   message.ConversationID,
				UserID:           message.UserID,
				Role:             message.Role,
				Content:          message.Content,
				ThinkingContent:  message.ThinkingContent,
				ThinkingDuration: cloneIntPointer(message.ThinkingDuration),
				Vote:             cloneIntPointer(message.Vote),
				CreateTime:       message.CreateTime,
			})
		}
	}
	return service
}

func (s *Service) loadSnapshot() (platformstate.Snapshot, error) {
	if s.store == nil {
		return platformstate.Snapshot{}, nil
	}
	return s.store.Load()
}

// UpsertConversation 写入或更新会话记录。
func (s *Service) UpsertConversation(session Session) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.sessions[session.UserID] == nil {
		s.sessions[session.UserID] = make(map[string]Session)
	}
	s.sessions[session.UserID][session.ConversationID] = session
	_ = s.persistLocked()
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
	_ = s.persistLocked()
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
	if err := s.persistLocked(); err != nil {
		return err
	}
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
	if err := s.persistLocked(); err != nil {
		return err
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
			if err := s.persistLocked(); err != nil {
				return err
			}
			return nil
		}
	}
	return ErrMessageNotFound
}

// StatsSnapshot 返回当前会话与消息统计。
func (s *Service) StatsSnapshot() StatsSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cutoff := time.Now().Add(-24 * time.Hour)
	result := StatsSnapshot{}
	for _, sessionsByUser := range s.sessions {
		for _, session := range sessionsByUser {
			result.TotalSessions++
			if session.LastTime.After(cutoff) {
				result.RecentSessions++
			}
		}
	}
	for _, messagesByUser := range s.messages {
		for _, messageList := range messagesByUser {
			for _, message := range messageList {
				result.TotalMessages++
				if message.CreateTime.After(cutoff) {
					result.RecentMessages++
				}
			}
		}
	}
	return result
}

func cloneIntPointer(value *int) *int {
	if value == nil {
		return nil
	}
	next := *value
	return &next
}

func (s *Service) persistLocked() error {
	if s.store == nil {
		return nil
	}
	sessionRecords := make([]platformstate.ConversationSessionRecord, 0)
	for _, sessionsByUser := range s.sessions {
		for _, session := range sessionsByUser {
			sessionRecords = append(sessionRecords, platformstate.ConversationSessionRecord{
				ConversationID: session.ConversationID,
				UserID:         session.UserID,
				Title:          session.Title,
				LastTime:       session.LastTime,
			})
		}
	}
	messageRecords := make([]platformstate.ConversationMessageRecord, 0)
	for _, messagesByUser := range s.messages {
		for _, messageList := range messagesByUser {
			for _, message := range messageList {
				messageRecords = append(messageRecords, platformstate.ConversationMessageRecord{
					ID:               message.ID,
					ConversationID:   message.ConversationID,
					UserID:           message.UserID,
					Role:             message.Role,
					Content:          message.Content,
					ThinkingContent:  message.ThinkingContent,
					ThinkingDuration: cloneIntPointer(message.ThinkingDuration),
					Vote:             cloneIntPointer(message.Vote),
					CreateTime:       message.CreateTime,
				})
			}
		}
	}
	sort.Slice(sessionRecords, func(i, j int) bool {
		if sessionRecords[i].UserID == sessionRecords[j].UserID {
			return sessionRecords[i].ConversationID < sessionRecords[j].ConversationID
		}
		return sessionRecords[i].UserID < sessionRecords[j].UserID
	})
	sort.Slice(messageRecords, func(i, j int) bool {
		if messageRecords[i].UserID == messageRecords[j].UserID {
			if messageRecords[i].ConversationID == messageRecords[j].ConversationID {
				return messageRecords[i].CreateTime.Before(messageRecords[j].CreateTime)
			}
			return messageRecords[i].ConversationID < messageRecords[j].ConversationID
		}
		return messageRecords[i].UserID < messageRecords[j].UserID
	})
	return s.store.Update(func(snapshot *platformstate.Snapshot) {
		snapshot.ConversationSessions = sessionRecords
		snapshot.ConversationMessages = messageRecords
	})
}
