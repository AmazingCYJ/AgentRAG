package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	domainauth "github.com/AmazingCYJ/AgentRAG/internal/domain/auth"
	domainchat "github.com/AmazingCYJ/AgentRAG/internal/domain/chat"
	"github.com/AmazingCYJ/AgentRAG/internal/platform/resp"
	"github.com/gogf/gf/v2/net/ghttp"
)

// ChatHandler 提供当前阶段最小可用的流式对话接口。
type ChatHandler struct {
	authService *domainauth.Service
	chatService *domainchat.Service
}

// NewChatHandler 创建聊天处理器。
func NewChatHandler(authService *domainauth.Service, chatService *domainchat.Service) *ChatHandler {
	return &ChatHandler{
		authService: authService,
		chatService: chatService,
	}
}

// StreamChat 处理 SSE 流式聊天请求。
func (h *ChatHandler) StreamChat(r *ghttp.Request) {
	profile, err := h.authService.CurrentUser(r.GetHeader("Authorization"))
	if err != nil {
		resp.WriteError(r, http.StatusUnauthorized, "401", "未登录")
		return
	}

	question := strings.TrimSpace(r.Get("question").String())
	if question == "" {
		resp.WriteError(r, http.StatusBadRequest, "400", "问题不能为空")
		return
	}

	writer := newSSEWriter(r)
	err = h.chatService.StreamChat(r.Context(), domainchat.StreamRequest{
		UserID:         profile.UserID,
		Question:       question,
		ConversationID: strings.TrimSpace(r.Get("conversationId").String()),
		DeepThinking:   r.Get("deepThinking").Bool(),
	}, writer)
	if err == nil || errors.Is(err, context.Canceled) {
		return
	}

	_ = writer.Event("error", map[string]string{
		"error": err.Error(),
	})
	_ = writer.Event("done", struct{}{})
}

// Stop 取消指定流式任务。
func (h *ChatHandler) Stop(r *ghttp.Request) {
	if _, err := h.authService.CurrentUser(r.GetHeader("Authorization")); err != nil {
		resp.WriteError(r, http.StatusUnauthorized, "401", "未登录")
		return
	}

	taskID := strings.TrimSpace(r.Get("taskId").String())
	if taskID == "" {
		resp.WriteError(r, http.StatusBadRequest, "400", "taskId 不能为空")
		return
	}

	h.chatService.StopTask(taskID)
	resp.WriteSuccess(r, nil)
}

type sseWriter struct {
	request *ghttp.Request
}

func newSSEWriter(r *ghttp.Request) *sseWriter {
	headers := r.Response.Header()
	headers.Set("Content-Type", "text/event-stream;charset=UTF-8")
	headers.Set("Cache-Control", "no-cache, no-transform")
	headers.Set("Connection", "keep-alive")
	headers.Set("X-Accel-Buffering", "no")

	return &sseWriter{request: r}
}

func (w *sseWriter) Event(name string, payload any) error {
	content, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	w.request.Response.Writef("event: %s\n", name)
	w.request.Response.Writef("data: %s\n\n", string(content))
	w.request.Response.Flush()
	return nil
}
