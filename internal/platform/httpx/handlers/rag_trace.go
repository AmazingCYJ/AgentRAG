package handlers

import (
	"errors"
	"net/http"
	"strings"

	domainauth "github.com/AmazingCYJ/AgentRAG/internal/domain/auth"
	domainragtrace "github.com/AmazingCYJ/AgentRAG/internal/domain/ragtrace"
	"github.com/AmazingCYJ/AgentRAG/internal/platform/resp"
	"github.com/gogf/gf/v2/net/ghttp"
)

// RagTraceHandler 提供链路追踪查询接口。
type RagTraceHandler struct {
	authService     *domainauth.Service
	ragTraceService *domainragtrace.Service
}

// NewRagTraceHandler 创建链路追踪处理器。
func NewRagTraceHandler(authService *domainauth.Service, ragTraceService *domainragtrace.Service) *RagTraceHandler {
	return &RagTraceHandler{
		authService:     authService,
		ragTraceService: ragTraceService,
	}
}

// PageRuns 返回链路运行分页列表。
func (h *RagTraceHandler) PageRuns(r *ghttp.Request) {
	if _, err := h.authService.CurrentUser(r.GetHeader("Authorization")); err != nil {
		resp.WriteError(r, http.StatusUnauthorized, "401", "未登录")
		return
	}
	resp.WriteSuccess(r, h.ragTraceService.PageRuns(domainragtrace.RunQuery{
		Current:        r.Get("current").Int(),
		Size:           r.Get("size").Int(),
		TraceID:        strings.TrimSpace(r.Get("traceId").String()),
		ConversationID: strings.TrimSpace(r.Get("conversationId").String()),
		TaskID:         strings.TrimSpace(r.Get("taskId").String()),
		Status:         strings.TrimSpace(r.Get("status").String()),
	}))
}

// Detail 返回指定 Trace 详情。
func (h *RagTraceHandler) Detail(r *ghttp.Request) {
	if _, err := h.authService.CurrentUser(r.GetHeader("Authorization")); err != nil {
		resp.WriteError(r, http.StatusUnauthorized, "401", "未登录")
		return
	}
	detail, err := h.ragTraceService.Detail(r.Get("traceId").String())
	if err != nil {
		writeRagTraceError(r, err)
		return
	}
	resp.WriteSuccess(r, detail)
}

// Nodes 返回指定 Trace 的全部节点。
func (h *RagTraceHandler) Nodes(r *ghttp.Request) {
	if _, err := h.authService.CurrentUser(r.GetHeader("Authorization")); err != nil {
		resp.WriteError(r, http.StatusUnauthorized, "401", "未登录")
		return
	}
	nodes, err := h.ragTraceService.ListNodes(r.Get("traceId").String())
	if err != nil {
		writeRagTraceError(r, err)
		return
	}
	resp.WriteSuccess(r, nodes)
}

func writeRagTraceError(r *ghttp.Request, err error) {
	if errors.Is(err, domainragtrace.ErrTraceNotFound) {
		resp.WriteError(r, http.StatusNotFound, "404", err.Error())
		return
	}
	resp.WriteError(r, http.StatusInternalServerError, "500", "链路查询失败")
}
