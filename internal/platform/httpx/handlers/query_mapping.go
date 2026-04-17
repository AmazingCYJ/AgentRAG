package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	domainauth "github.com/AmazingCYJ/AgentRAG/internal/domain/auth"
	domainquerymapping "github.com/AmazingCYJ/AgentRAG/internal/domain/querymapping"
	"github.com/AmazingCYJ/AgentRAG/internal/platform/resp"
	"github.com/gogf/gf/v2/net/ghttp"
)

// QueryMappingHandler 提供关键词映射相关接口。
type QueryMappingHandler struct {
	authService         *domainauth.Service
	queryMappingService *domainquerymapping.Service
}

// NewQueryMappingHandler 创建关键词映射处理器。
func NewQueryMappingHandler(authService *domainauth.Service, queryMappingService *domainquerymapping.Service) *QueryMappingHandler {
	return &QueryMappingHandler{
		authService:         authService,
		queryMappingService: queryMappingService,
	}
}

type queryMappingSaveRequest struct {
	SourceTerm string `json:"sourceTerm"`
	TargetTerm string `json:"targetTerm"`
	MatchType  int    `json:"matchType"`
	Priority   int    `json:"priority"`
	Enabled    bool   `json:"enabled"`
	Remark     string `json:"remark"`
}

// Page 返回映射规则分页结果。
func (h *QueryMappingHandler) Page(r *ghttp.Request) {
	if _, err := h.authService.CurrentUser(r.GetHeader("Authorization")); err != nil {
		resp.WriteError(r, http.StatusUnauthorized, "401", "未登录")
		return
	}
	resp.WriteSuccess(r, h.queryMappingService.Page(
		r.Get("current").Int(),
		r.Get("size").Int(),
		strings.TrimSpace(r.Get("keyword").String()),
	))
}

// Get 返回映射规则详情。
func (h *QueryMappingHandler) Get(r *ghttp.Request) {
	if _, err := h.authService.CurrentUser(r.GetHeader("Authorization")); err != nil {
		resp.WriteError(r, http.StatusUnauthorized, "401", "未登录")
		return
	}
	item, err := h.queryMappingService.GetByID(r.Get("id").String())
	if err != nil {
		writeQueryMappingError(r, err)
		return
	}
	resp.WriteSuccess(r, item)
}

// Create 新增映射规则。
func (h *QueryMappingHandler) Create(r *ghttp.Request) {
	if _, err := h.authService.CurrentUser(r.GetHeader("Authorization")); err != nil {
		resp.WriteError(r, http.StatusUnauthorized, "401", "未登录")
		return
	}
	var req queryMappingSaveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.WriteError(r, http.StatusBadRequest, "400", "请求参数错误")
		return
	}
	id, err := h.queryMappingService.Create(domainquerymapping.SaveRequest{
		SourceTerm: req.SourceTerm,
		TargetTerm: req.TargetTerm,
		MatchType:  req.MatchType,
		Priority:   req.Priority,
		Enabled:    req.Enabled,
		Remark:     req.Remark,
	})
	if err != nil {
		writeQueryMappingError(r, err)
		return
	}
	resp.WriteSuccess(r, id)
}

// Update 更新映射规则。
func (h *QueryMappingHandler) Update(r *ghttp.Request) {
	if _, err := h.authService.CurrentUser(r.GetHeader("Authorization")); err != nil {
		resp.WriteError(r, http.StatusUnauthorized, "401", "未登录")
		return
	}
	var req queryMappingSaveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.WriteError(r, http.StatusBadRequest, "400", "请求参数错误")
		return
	}
	err := h.queryMappingService.Update(r.Get("id").String(), domainquerymapping.SaveRequest{
		SourceTerm: req.SourceTerm,
		TargetTerm: req.TargetTerm,
		MatchType:  req.MatchType,
		Priority:   req.Priority,
		Enabled:    req.Enabled,
		Remark:     req.Remark,
	})
	if err != nil {
		writeQueryMappingError(r, err)
		return
	}
	resp.WriteSuccess(r, nil)
}

// Delete 删除映射规则。
func (h *QueryMappingHandler) Delete(r *ghttp.Request) {
	if _, err := h.authService.CurrentUser(r.GetHeader("Authorization")); err != nil {
		resp.WriteError(r, http.StatusUnauthorized, "401", "未登录")
		return
	}
	if err := h.queryMappingService.Delete(r.Get("id").String()); err != nil {
		writeQueryMappingError(r, err)
		return
	}
	resp.WriteSuccess(r, nil)
}

func writeQueryMappingError(r *ghttp.Request, err error) {
	switch {
	case errors.Is(err, domainquerymapping.ErrSourceTermRequired),
		errors.Is(err, domainquerymapping.ErrTargetTermRequired):
		resp.WriteError(r, http.StatusBadRequest, "400", err.Error())
	case errors.Is(err, domainquerymapping.ErrMappingNotFound):
		resp.WriteError(r, http.StatusNotFound, "404", err.Error())
	default:
		resp.WriteError(r, http.StatusInternalServerError, "500", "关键词映射操作失败")
	}
}
