package handlers

import (
	"net/http"
	"strings"

	domainauth "github.com/AmazingCYJ/AgentRAG/internal/domain/auth"
	domaindashboard "github.com/AmazingCYJ/AgentRAG/internal/domain/dashboard"
	"github.com/AmazingCYJ/AgentRAG/internal/platform/resp"
	"github.com/gogf/gf/v2/net/ghttp"
)

// DashboardHandler 提供后台仪表盘接口。
type DashboardHandler struct {
	authService      *domainauth.Service
	dashboardService *domaindashboard.Service
}

// NewDashboardHandler 创建仪表盘处理器。
func NewDashboardHandler(authService *domainauth.Service, dashboardService *domaindashboard.Service) *DashboardHandler {
	return &DashboardHandler{
		authService:      authService,
		dashboardService: dashboardService,
	}
}

// Overview 返回概览数据。
func (h *DashboardHandler) Overview(r *ghttp.Request) {
	if !h.requireAdmin(r) {
		return
	}
	resp.WriteSuccess(r, h.dashboardService.LoadOverview(strings.TrimSpace(r.Get("window").String())))
}

// Performance 返回性能数据。
func (h *DashboardHandler) Performance(r *ghttp.Request) {
	if !h.requireAdmin(r) {
		return
	}
	resp.WriteSuccess(r, h.dashboardService.LoadPerformance(strings.TrimSpace(r.Get("window").String())))
}

// Trends 返回趋势数据。
func (h *DashboardHandler) Trends(r *ghttp.Request) {
	if !h.requireAdmin(r) {
		return
	}
	resp.WriteSuccess(r, h.dashboardService.LoadTrends(
		strings.TrimSpace(r.Get("metric").String()),
		strings.TrimSpace(r.Get("window").String()),
		strings.TrimSpace(r.Get("granularity").String()),
	))
}

func (h *DashboardHandler) requireAdmin(r *ghttp.Request) bool {
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
