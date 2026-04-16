package handlers

import (
	"encoding/json"

	domainauth "github.com/AmazingCYJ/AgentRAG/internal/domain/auth"
	domainconversation "github.com/AmazingCYJ/AgentRAG/internal/domain/conversation"
	"github.com/AmazingCYJ/AgentRAG/internal/platform/resp"
	"github.com/gogf/gf/v2/net/ghttp"
)

// ConversationHandler 提供会话列表与消息列表接口。
type ConversationHandler struct {
	authService         *domainauth.Service
	conversationService *domainconversation.Service
}

// NewConversationHandler 创建会话处理器。
func NewConversationHandler(authService *domainauth.Service, conversationService *domainconversation.Service) *ConversationHandler {
	return &ConversationHandler{
		authService:         authService,
		conversationService: conversationService,
	}
}

// ListConversations 返回当前用户会话列表。
func (h *ConversationHandler) ListConversations(r *ghttp.Request) {
	profile, err := h.authService.CurrentUser(r.GetHeader("Authorization"))
	if err != nil {
		resp.WriteError(r, 401, "401", "未登录")
		return
	}
	resp.WriteSuccess(r, h.conversationService.ListByUserID(profile.UserID))
}

// ListMessages 返回指定会话消息列表。
func (h *ConversationHandler) ListMessages(r *ghttp.Request) {
	profile, err := h.authService.CurrentUser(r.GetHeader("Authorization"))
	if err != nil {
		resp.WriteError(r, 401, "401", "未登录")
		return
	}
	resp.WriteSuccess(r, h.conversationService.ListMessages(r.Get("conversationId").String(), profile.UserID))
}

type renameConversationRequest struct {
	Title string `json:"title"`
}

// Rename 更新指定会话标题。
func (h *ConversationHandler) Rename(r *ghttp.Request) {
	profile, err := h.authService.CurrentUser(r.GetHeader("Authorization"))
	if err != nil {
		resp.WriteError(r, 401, "401", "未登录")
		return
	}
	var req renameConversationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.WriteError(r, 400, "400", "请求参数错误")
		return
	}
	if err := h.conversationService.Rename(r.Get("conversationId").String(), profile.UserID, req.Title); err != nil {
		resp.WriteError(r, 404, "404", err.Error())
		return
	}
	resp.WriteSuccess(r, nil)
}

// Delete 删除指定会话与关联消息。
func (h *ConversationHandler) Delete(r *ghttp.Request) {
	profile, err := h.authService.CurrentUser(r.GetHeader("Authorization"))
	if err != nil {
		resp.WriteError(r, 401, "401", "未登录")
		return
	}
	if err := h.conversationService.Delete(r.Get("conversationId").String(), profile.UserID); err != nil {
		resp.WriteError(r, 404, "404", err.Error())
		return
	}
	resp.WriteSuccess(r, nil)
}
