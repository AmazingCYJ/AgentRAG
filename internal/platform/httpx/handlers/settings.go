package handlers

import (
	"net/http"

	domainauth "github.com/AmazingCYJ/AgentRAG/internal/domain/auth"
	domainsettings "github.com/AmazingCYJ/AgentRAG/internal/domain/settings"
	"github.com/AmazingCYJ/AgentRAG/internal/platform/resp"
	"github.com/gogf/gf/v2/net/ghttp"
)

// SettingsHandler 提供系统配置查询接口。
type SettingsHandler struct {
	authService     *domainauth.Service
	settingsService *domainsettings.Service
}

// NewSettingsHandler 创建设置处理器。
func NewSettingsHandler(authService *domainauth.Service, settingsService *domainsettings.Service) *SettingsHandler {
	return &SettingsHandler{
		authService:     authService,
		settingsService: settingsService,
	}
}

// Get 返回当前系统配置。
func (h *SettingsHandler) Get(r *ghttp.Request) {
	if _, err := h.authService.CurrentUser(r.GetHeader("Authorization")); err != nil {
		resp.WriteError(r, http.StatusUnauthorized, "401", "未登录")
		return
	}
	resp.WriteSuccess(r, h.settingsService.Get())
}
