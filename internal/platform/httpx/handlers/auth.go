package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	domainauth "github.com/AmazingCYJ/AgentRAG/internal/domain/auth"
	"github.com/AmazingCYJ/AgentRAG/internal/platform/resp"
	"github.com/gogf/gf/v2/net/ghttp"
)

// AuthHandler 提供当前阶段最小认证接口。
type AuthHandler struct {
	service *domainauth.Service
}

// NewAuthHandler 创建认证处理器。
func NewAuthHandler(service *domainauth.Service) *AuthHandler {
	return &AuthHandler{service: service}
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Login 处理账号登录。
func (h *AuthHandler) Login(r *ghttp.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.WriteError(r, http.StatusBadRequest, "400", "请求参数错误")
		return
	}
	result, err := h.service.Login(req.Username, req.Password)
	if err != nil {
		if errors.Is(err, domainauth.ErrInvalidCredentials) {
			resp.WriteError(r, http.StatusUnauthorized, "401", err.Error())
			return
		}
		resp.WriteError(r, http.StatusInternalServerError, "500", "登录失败")
		return
	}
	resp.WriteSuccess(r, result)
}

// Logout 处理当前阶段登出请求。
func (h *AuthHandler) Logout(r *ghttp.Request) {
	resp.WriteSuccess(r, nil)
}

// CurrentUser 返回当前登录用户资料。
func (h *AuthHandler) CurrentUser(r *ghttp.Request) {
	profile, err := h.service.CurrentUser(r.GetHeader("Authorization"))
	if err != nil {
		resp.WriteError(r, http.StatusUnauthorized, "401", "未登录")
		return
	}
	resp.WriteSuccess(r, profile)
}
