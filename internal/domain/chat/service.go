package chat

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	domainconversation "github.com/AmazingCYJ/AgentRAG/internal/domain/conversation"
	domainragtrace "github.com/AmazingCYJ/AgentRAG/internal/domain/ragtrace"
	"github.com/gogf/gf/v2/util/guid"
)

var (
	// ErrQuestionRequired 表示提问内容为空。
	ErrQuestionRequired = errors.New("问题不能为空")
)

const (
	defaultSessionTitle  = "新对话"
	thinkingChunkSize    = 10
	responseChunkSize    = 18
	historyMessageLimit  = 12
	defaultMaxConcurrent = 8
	prepareDelay         = 40 * time.Millisecond
	chunkDelay           = 20 * time.Millisecond
	conversationIDPrefix = "conv_"
	rejectMessage        = "当前对话请求较多，请稍后再试。"
)

// StreamRequest 定义流式聊天所需的输入参数。
type StreamRequest struct {
	UserID         string
	Username       string
	Question       string
	ConversationID string
	DeepThinking   bool
}

// EventWriter 抽象 SSE 事件输出能力，便于服务层复用与测试。
type EventWriter interface {
	Event(name string, payload any) error
}

type streamMeta struct {
	ConversationID string `json:"conversationId"`
	TaskID         string `json:"taskId"`
}

type streamMessage struct {
	Type  string `json:"type"`
	Delta string `json:"delta"`
}

type streamCompletion struct {
	MessageID string `json:"messageId,omitempty"`
	Title     string `json:"title,omitempty"`
}

type taskHandle struct {
	cancel context.CancelFunc
}

// Service 提供当前阶段最小可用的流式聊天与任务取消能力。
type Service struct {
	conversationService *domainconversation.Service
	traceService        *domainragtrace.Service
	generator           Generator

	mu            sync.Mutex
	tasks         map[string]taskHandle
	activeStreams int
	maxConcurrent int

	now    func() time.Time
	newID  func() string
	waitFn func(context.Context, time.Duration) error
}

// NewService 创建聊天服务。
func NewService(
	conversationService *domainconversation.Service,
	traceService *domainragtrace.Service,
	generators ...Generator,
) *Service {
	var generator Generator = &fallbackGenerator{}
	if len(generators) > 0 && generators[0] != nil {
		generator = generators[0]
	}
	return &Service{
		conversationService: conversationService,
		traceService:        traceService,
		generator:           generator,
		tasks:               make(map[string]taskHandle),
		maxConcurrent:       defaultMaxConcurrent,
		now:                 time.Now,
		newID: func() string {
			return guid.S()
		},
		waitFn: waitWithContext,
	}
}

// SetMaxConcurrent 设置本实例允许的最大并发流式对话数。
func (s *Service) SetMaxConcurrent(limit int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.maxConcurrent = limit
}

// StreamChat 以 SSE 方式输出当前阶段的最小对话结果。
func (s *Service) StreamChat(ctx context.Context, req StreamRequest, writer EventWriter) error {
	question := strings.TrimSpace(req.Question)
	if question == "" {
		return ErrQuestionRequired
	}

	conversationID := strings.TrimSpace(req.ConversationID)
	if conversationID == "" {
		conversationID = conversationIDPrefix + compactID(s.newID())
	}
	history := s.recentHistory(conversationID, req.UserID, historyMessageLimit)
	taskID := compactID(s.newID())
	title := buildConversationTitle(question)
	startedAt := s.now()

	s.conversationService.UpsertConversation(domainconversation.Session{
		ConversationID: conversationID,
		UserID:         req.UserID,
		Title:          defaultSessionTitle,
		LastTime:       startedAt,
	})
	s.conversationService.AppendMessage(domainconversation.Message{
		ID:             compactID(s.newID()),
		ConversationID: conversationID,
		UserID:         req.UserID,
		Role:           "user",
		Content:        question,
		CreateTime:     startedAt,
	})

	if !s.tryAcquireStream() {
		return s.finishRejected(writer, req, conversationID, taskID, title, startedAt)
	}
	defer s.releaseStream()

	taskCtx, cancel := context.WithCancel(ctx)
	s.registerTask(taskID, cancel)
	defer s.unregisterTask(taskID)

	if err := writer.Event("meta", streamMeta{
		ConversationID: conversationID,
		TaskID:         taskID,
	}); err != nil {
		return err
	}

	if err := s.waitFn(taskCtx, prepareDelay); err != nil {
		return s.finishCanceled(writer, req, conversationID, taskID, title, "", "", startedAt, nil)
	}

	generated, err := s.generator.Generate(taskCtx, GenerateRequest{
		Question:     question,
		DeepThinking: req.DeepThinking,
		History:      history,
	})
	if err != nil {
		return err
	}

	thinkingContent := generated.Thinking
	if req.DeepThinking && thinkingContent != "" {
		for _, chunk := range splitText(thinkingContent, thinkingChunkSize) {
			if err := s.waitFn(taskCtx, chunkDelay); err != nil {
				return s.finishCanceled(writer, req, conversationID, taskID, title, "", thinkingContent, startedAt, generated.Steps)
			}
			if err := writer.Event("message", streamMessage{
				Type:  "think",
				Delta: chunk,
			}); err != nil {
				return err
			}
		}
	}

	responseContent := ""
	for _, chunk := range splitText(generated.Answer, responseChunkSize) {
		if err := s.waitFn(taskCtx, chunkDelay); err != nil {
			return s.finishCanceled(writer, req, conversationID, taskID, title, responseContent, thinkingContent, startedAt, generated.Steps)
		}
		if err := writer.Event("message", streamMessage{
			Type:  "response",
			Delta: chunk,
		}); err != nil {
			return err
		}
		responseContent += chunk
	}

	messageID := s.persistAssistantMessage(req.UserID, conversationID, title, responseContent, thinkingContent, startedAt)
	s.recordTrace(req, conversationID, taskID, "success", "", startedAt, s.now(), generated.Steps)
	if err := writer.Event("finish", streamCompletion{
		MessageID: messageID,
		Title:     title,
	}); err != nil {
		return err
	}
	return writer.Event("done", struct{}{})
}

func (s *Service) finishRejected(
	writer EventWriter,
	req StreamRequest,
	conversationID, taskID, title string,
	startedAt time.Time,
) error {
	if err := writer.Event("meta", streamMeta{
		ConversationID: conversationID,
		TaskID:         taskID,
	}); err != nil {
		return err
	}
	if err := writer.Event("reject", streamMessage{
		Type:  "response",
		Delta: rejectMessage,
	}); err != nil {
		return err
	}
	messageID := s.persistAssistantMessage(req.UserID, conversationID, title, rejectMessage, "", startedAt)
	s.recordTrace(req, conversationID, taskID, "failed", rejectMessage, startedAt, s.now(), []WorkflowStep{
		{
			NodeID:     "reject_concurrency",
			NodeType:   "RATE_LIMIT",
			NodeName:   "Reject Concurrency",
			Status:     "failed",
			DurationMs: 1,
			Detail:     rejectMessage,
		},
	})
	if err := writer.Event("finish", streamCompletion{
		MessageID: messageID,
		Title:     title,
	}); err != nil {
		return err
	}
	return writer.Event("done", struct{}{})
}

func (s *Service) recentHistory(conversationID, userID string, limit int) []HistoryMessage {
	if s.conversationService == nil {
		return nil
	}
	messages := s.conversationService.ListMessages(conversationID, userID)
	if len(messages) == 0 {
		return nil
	}
	if limit <= 0 {
		limit = 6
	}
	if len(messages) > limit {
		messages = messages[len(messages)-limit:]
	}

	history := make([]HistoryMessage, 0, len(messages))
	for _, message := range messages {
		role := strings.ToLower(strings.TrimSpace(message.Role))
		content := strings.TrimSpace(message.Content)
		if content == "" || (role != "user" && role != "assistant") {
			continue
		}
		history = append(history, HistoryMessage{
			Role:    role,
			Content: content,
		})
	}
	return history
}

func (s *Service) tryAcquireStream() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.maxConcurrent <= 0 {
		s.activeStreams++
		return true
	}
	if s.activeStreams >= s.maxConcurrent {
		return false
	}
	s.activeStreams++
	return true
}

func (s *Service) releaseStream() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.activeStreams > 0 {
		s.activeStreams--
	}
}

// StopTask 取消指定流式任务，未知任务按幂等成功处理。
func (s *Service) StopTask(taskID string) {
	s.mu.Lock()
	handle, ok := s.tasks[taskID]
	s.mu.Unlock()
	if ok {
		handle.cancel()
	}
}

func (s *Service) finishCanceled(
	writer EventWriter,
	req StreamRequest,
	conversationID, taskID, title, responseContent, thinkingContent string,
	startedAt time.Time,
	steps []WorkflowStep,
) error {
	messageID := s.persistAssistantMessage(req.UserID, conversationID, title, responseContent, thinkingContent, startedAt)
	s.recordTrace(req, conversationID, taskID, "failed", "用户停止生成", startedAt, s.now(), steps)
	if err := writer.Event("cancel", streamCompletion{
		MessageID: messageID,
		Title:     title,
	}); err != nil {
		return err
	}
	return writer.Event("done", struct{}{})
}

func (s *Service) persistAssistantMessage(
	userID, conversationID, title, responseContent, thinkingContent string,
	startedAt time.Time,
) string {
	lastTime := s.now()
	s.conversationService.UpsertConversation(domainconversation.Session{
		ConversationID: conversationID,
		UserID:         userID,
		Title:          title,
		LastTime:       lastTime,
	})

	var thinkingDuration *int
	if thinkingContent != "" {
		seconds := int(lastTime.Sub(startedAt).Seconds())
		if seconds < 1 {
			seconds = 1
		}
		thinkingDuration = &seconds
	}

	messageID := compactID(s.newID())
	s.conversationService.AppendMessage(domainconversation.Message{
		ID:               messageID,
		ConversationID:   conversationID,
		UserID:           userID,
		Role:             "assistant",
		Content:          responseContent,
		ThinkingContent:  thinkingContent,
		ThinkingDuration: thinkingDuration,
		CreateTime:       lastTime,
	})
	return messageID
}

func (s *Service) recordTrace(
	req StreamRequest,
	conversationID, taskID, status, errorMessage string,
	startedAt, endedAt time.Time,
	steps []WorkflowStep,
) {
	if s.traceService == nil {
		return
	}
	s.traceService.RecordChatTrace(domainragtrace.ChatTraceRecord{
		TraceName:      buildConversationTitle(req.Question),
		ConversationID: conversationID,
		TaskID:         taskID,
		UserName:       req.Username,
		UserID:         req.UserID,
		Status:         status,
		ErrorMessage:   errorMessage,
		DurationMs:     endedAt.Sub(startedAt).Milliseconds(),
		StartTime:      startedAt,
		EndTime:        endedAt,
		DeepThinking:   req.DeepThinking,
		Steps:          toTraceSteps(steps),
	})
}

func toTraceSteps(steps []WorkflowStep) []domainragtrace.ChatTraceStep {
	if len(steps) == 0 {
		return nil
	}
	result := make([]domainragtrace.ChatTraceStep, 0, len(steps))
	for _, step := range steps {
		result = append(result, domainragtrace.ChatTraceStep{
			NodeID:     step.NodeID,
			NodeType:   step.NodeType,
			NodeName:   step.NodeName,
			Status:     strings.ToUpper(strings.TrimSpace(step.Status)),
			DurationMs: step.DurationMs,
			Detail:     step.Detail,
		})
	}
	return result
}

func (s *Service) registerTask(taskID string, cancel context.CancelFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.tasks[taskID] = taskHandle{cancel: cancel}
}

func (s *Service) unregisterTask(taskID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.tasks, taskID)
}

func waitWithContext(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			return nil
		}
	}

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func compactID(raw string) string {
	return strings.ReplaceAll(raw, "-", "")
}

func buildThinkingText(question string) string {
	return "正在分析你的问题并整理与“" + question + "”相关的上下文。"
}

func buildResponseText(question string, deepThinking bool, knowledgeContext string, history []HistoryMessage) string {
	contextText := strings.TrimSpace(knowledgeContext)
	if contextText != "" {
		return "根据当前可用上下文，关于“" + question + "”的回答如下：\n\n" + compactContextAnswer(contextText)
	}
	historyText := compactHistoryAnswer(history)
	if historyText != "" {
		return "结合上一轮对话，关于“" + question + "”的回答如下：\n\n" + historyText
	}
	if deepThinking {
		return "我已经收到你的问题：“" + question + "”。当前未配置可用模型，也没有检索到可引用上下文，因此只能返回本地兜底说明。"
	}
	return "我已经收到你的问题：“" + question + "”。当前未配置可用模型，也没有检索到可引用上下文。"
}

func compactHistoryAnswer(history []HistoryMessage) string {
	if len(history) == 0 {
		return ""
	}
	parts := make([]string, 0, len(history))
	for _, message := range history {
		content := strings.TrimSpace(message.Content)
		if content == "" {
			continue
		}
		role := "用户"
		if strings.EqualFold(message.Role, "assistant") {
			role = "助手"
		}
		parts = append(parts, role+"："+content)
	}
	if len(parts) == 0 {
		return ""
	}
	return compactContextAnswer(strings.Join(parts, "\n"))
}

func compactContextAnswer(contextText string) string {
	runes := []rune(strings.TrimSpace(contextText))
	if len(runes) <= 600 {
		return string(runes)
	}
	return string(runes[:600]) + "..."
}

func buildConversationTitle(question string) string {
	title := strings.TrimSpace(question)
	if title == "" {
		return defaultSessionTitle
	}

	runes := []rune(title)
	if len(runes) <= 18 {
		return title
	}
	return string(runes[:18]) + "..."
}

func splitText(content string, chunkSize int) []string {
	if chunkSize <= 0 {
		return []string{content}
	}

	runes := []rune(content)
	if len(runes) == 0 {
		return []string{}
	}

	result := make([]string, 0, (len(runes)+chunkSize-1)/chunkSize)
	for start := 0; start < len(runes); start += chunkSize {
		end := start + chunkSize
		if end > len(runes) {
			end = len(runes)
		}
		result = append(result, string(runes[start:end]))
	}
	return result
}
