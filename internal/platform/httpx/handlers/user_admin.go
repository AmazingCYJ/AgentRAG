package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	domainauth "github.com/AmazingCYJ/AgentRAG/internal/domain/auth"
	domainusermgmt "github.com/AmazingCYJ/AgentRAG/internal/domain/usermgmt"
	"github.com/AmazingCYJ/AgentRAG/internal/platform/resp"
	"github.com/gogf/gf/v2/net/ghttp"
)

// UserAdminHandler 提供用户管理接口。
type UserAdminHandler struct {
	authService *domainauth.Service
	userService *domainusermgmt.Service
}

// NewUserAdminHandler 创建用户管理处理器。
func NewUserAdminHandler(authService *domainauth.Service, userService *domainusermgmt.Service) *UserAdminHandler {
	return &UserAdminHandler{
		authService: authService,
		userService: userService,
	}
}

type userSaveRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Role     string `json:"role"`
	Avatar   string `json:"avatar"`
}

type changePasswordRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

// Page 分页查询用户。
func (h *UserAdminHandler) Page(r *ghttp.Request) {
	if !h.requireAdmin(r) {
		return
	}
	resp.WriteSuccess(r, h.userService.Page(
		r.Get("current").Int(),
		r.Get("size").Int(),
		strings.TrimSpace(r.Get("keyword").String()),
	))
}

// Create 创建用户。
func (h *UserAdminHandler) Create(r *ghttp.Request) {
	if !h.requireAdmin(r) {
		return
	}
	var req userSaveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.WriteError(r, http.StatusBadRequest, "400", "请求参数错误")
		return
	}
	id, err := h.userService.Create(domainusermgmt.CreateRequest{
		Username: req.Username,
		Password: req.Password,
		Role:     req.Role,
		Avatar:   req.Avatar,
	})
	if err != nil {
		writeUserError(r, err)
		return
	}
	resp.WriteSuccess(r, id)
}

// Update 更新用户。
func (h *UserAdminHandler) Update(r *ghttp.Request) {
	if !h.requireAdmin(r) {
		return
	}
	var req userSaveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.WriteError(r, http.StatusBadRequest, "400", "请求参数错误")
		return
	}
	if err := h.userService.Update(r.Get("id").String(), domainusermgmt.UpdateRequest{
		Username: req.Username,
		Password: req.Password,
		Role:     req.Role,
		Avatar:   req.Avatar,
	}); err != nil {
		writeUserError(r, err)
		return
	}
	resp.WriteSuccess(r, nil)
}

// Delete 删除用户。
func (h *UserAdminHandler) Delete(r *ghttp.Request) {
	if !h.requireAdmin(r) {
		return
	}
	if err := h.userService.Delete(r.Get("id").String()); err != nil {
		writeUserError(r, err)
		return
	}
	resp.WriteSuccess(r, nil)
}

// ChangePassword 修改当前登录用户密码。
func (h *UserAdminHandler) ChangePassword(r *ghttp.Request) {
	if _, err := h.authService.CurrentUser(r.GetHeader("Authorization")); err != nil {
		resp.WriteError(r, http.StatusUnauthorized, "401", "未登录")
		return
	}
	var req changePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.WriteError(r, http.StatusBadRequest, "400", "请求参数错误")
		return
	}
	if err := h.authService.ChangePassword(req.CurrentPassword, req.NewPassword); err != nil {
		if errors.Is(err, domainauth.ErrInvalidCurrentPassword) || errors.Is(err, domainauth.ErrNewPasswordRequired) {
			resp.WriteError(r, http.StatusBadRequest, "400", err.Error())
			return
		}
		resp.WriteError(r, http.StatusInternalServerError, "500", "修改密码失败")
		return
	}
	resp.WriteSuccess(r, nil)
}

func (h *UserAdminHandler) requireAdmin(r *ghttp.Request) bool {
	profile, err := h.authService.CurrentUser(r.GetHeader("Authorization"))
	if err != nil {
		resp.WriteError(r, http.StatusUnauthorized, "401", "未登录")
		return false
	}
	if !strings.EqualFold(profile.Role, "admin") {
		resp.WriteError(r, http.StatusForbidden, "403", "无权限")
		return false
	}
	return true
}

func writeUserError(r *ghttp.Request, err error) {
	switch {
	case errors.Is(err, domainusermgmt.ErrUserNotFound):
		resp.WriteError(r, http.StatusNotFound, "404", err.Error())
	case errors.Is(err, domainusermgmt.ErrUsernameRequired),
		errors.Is(err, domainusermgmt.ErrPasswordRequired),
		errors.Is(err, domainusermgmt.ErrUsernameExists),
		errors.Is(err, domainusermgmt.ErrProtectedUser):
		resp.WriteError(r, http.StatusBadRequest, "400", err.Error())
	default:
		resp.WriteError(r, http.StatusInternalServerError, "500", "用户操作失败")
	}
}
