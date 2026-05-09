package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	domainauth "github.com/AmazingCYJ/AgentRAG/internal/domain/auth"
	domainintenttree "github.com/AmazingCYJ/AgentRAG/internal/domain/intenttree"
	"github.com/AmazingCYJ/AgentRAG/internal/platform/resp"
	"github.com/gogf/gf/v2/net/ghttp"
)

// IntentTreeHandler 提供意图树相关接口。
type IntentTreeHandler struct {
	authService       *domainauth.Service
	intentTreeService *domainintenttree.Service
}

// NewIntentTreeHandler 创建意图树处理器。
func NewIntentTreeHandler(authService *domainauth.Service, intentTreeService *domainintenttree.Service) *IntentTreeHandler {
	return &IntentTreeHandler{
		authService:       authService,
		intentTreeService: intentTreeService,
	}
}

type intentNodeSaveRequest struct {
	KBID                string   `json:"kbId"`
	IntentCode          string   `json:"intentCode"`
	Name                string   `json:"name"`
	Level               int      `json:"level"`
	ParentCode          string   `json:"parentCode"`
	Description         string   `json:"description"`
	Examples            []string `json:"examples"`
	MCPToolID           string   `json:"mcpToolId"`
	TopK                *int     `json:"topK"`
	Kind                int      `json:"kind"`
	SortOrder           int      `json:"sortOrder"`
	Enabled             int      `json:"enabled"`
	PromptSnippet       string   `json:"promptSnippet"`
	PromptTemplate      string   `json:"promptTemplate"`
	ParamPromptTemplate string   `json:"paramPromptTemplate"`
	CollectionName      string   `json:"collectionName"`
}

type intentNodeBatchRequest struct {
	IDs []string `json:"ids"`
}

// Tree 返回完整意图树。
func (h *IntentTreeHandler) Tree(r *ghttp.Request) {
	if _, err := h.authService.CurrentUser(r.GetHeader("Authorization")); err != nil {
		resp.WriteError(r, http.StatusUnauthorized, "401", "未登录")
		return
	}
	resp.WriteSuccess(r, h.intentTreeService.GetFullTree())
}

// CreateNode 创建意图节点。
func (h *IntentTreeHandler) CreateNode(r *ghttp.Request) {
	if _, err := h.authService.CurrentUser(r.GetHeader("Authorization")); err != nil {
		resp.WriteError(r, http.StatusUnauthorized, "401", "未登录")
		return
	}
	var req intentNodeSaveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.WriteError(r, http.StatusBadRequest, "400", "请求参数错误")
		return
	}

	id, err := h.intentTreeService.CreateNode(domainintenttree.CreateRequest{
		KBID:                req.KBID,
		IntentCode:          req.IntentCode,
		Name:                req.Name,
		Level:               req.Level,
		ParentCode:          req.ParentCode,
		Description:         req.Description,
		Examples:            req.Examples,
		MCPToolID:           req.MCPToolID,
		TopK:                req.TopK,
		Kind:                req.Kind,
		SortOrder:           req.SortOrder,
		Enabled:             req.Enabled,
		PromptSnippet:       req.PromptSnippet,
		PromptTemplate:      req.PromptTemplate,
		ParamPromptTemplate: req.ParamPromptTemplate,
	})
	if err != nil {
		writeIntentTreeError(r, err)
		return
	}
	resp.WriteSuccess(r, id)
}

// UpdateNode 更新意图节点。
func (h *IntentTreeHandler) UpdateNode(r *ghttp.Request) {
	if _, err := h.authService.CurrentUser(r.GetHeader("Authorization")); err != nil {
		resp.WriteError(r, http.StatusUnauthorized, "401", "未登录")
		return
	}
	id := r.Get("id").String()
	if id == "" {
		resp.WriteError(r, http.StatusBadRequest, "400", "节点ID错误")
		return
	}
	var req intentNodeSaveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.WriteError(r, http.StatusBadRequest, "400", "请求参数错误")
		return
	}

	err := h.intentTreeService.UpdateNode(id, domainintenttree.UpdateRequest{
		Name:                req.Name,
		Level:               req.Level,
		ParentCode:          req.ParentCode,
		Description:         req.Description,
		Examples:            req.Examples,
		CollectionName:      req.CollectionName,
		MCPToolID:           req.MCPToolID,
		TopK:                req.TopK,
		Kind:                req.Kind,
		SortOrder:           req.SortOrder,
		Enabled:             req.Enabled,
		PromptSnippet:       req.PromptSnippet,
		PromptTemplate:      req.PromptTemplate,
		ParamPromptTemplate: req.ParamPromptTemplate,
	})
	if err != nil {
		writeIntentTreeError(r, err)
		return
	}
	resp.WriteSuccess(r, nil)
}

// DeleteNode 删除意图节点。
func (h *IntentTreeHandler) DeleteNode(r *ghttp.Request) {
	if _, err := h.authService.CurrentUser(r.GetHeader("Authorization")); err != nil {
		resp.WriteError(r, http.StatusUnauthorized, "401", "未登录")
		return
	}
	id := r.Get("id").String()
	if id == "" {
		resp.WriteError(r, http.StatusBadRequest, "400", "节点ID错误")
		return
	}
	if err := h.intentTreeService.DeleteNode(id); err != nil {
		writeIntentTreeError(r, err)
		return
	}
	resp.WriteSuccess(r, nil)
}

// BatchEnable 批量启用节点。
func (h *IntentTreeHandler) BatchEnable(r *ghttp.Request) {
	h.handleBatchUpdate(r, func(ids []string) {
		h.intentTreeService.BatchEnableNodes(ids)
	})
}

// BatchDisable 批量停用节点。
func (h *IntentTreeHandler) BatchDisable(r *ghttp.Request) {
	h.handleBatchUpdate(r, func(ids []string) {
		h.intentTreeService.BatchDisableNodes(ids)
	})
}

// BatchDelete 批量删除节点。
func (h *IntentTreeHandler) BatchDelete(r *ghttp.Request) {
	h.handleBatchUpdate(r, func(ids []string) {
		h.intentTreeService.BatchDeleteNodes(ids)
	})
}

func (h *IntentTreeHandler) handleBatchUpdate(r *ghttp.Request, action func(ids []string)) {
	if _, err := h.authService.CurrentUser(r.GetHeader("Authorization")); err != nil {
		resp.WriteError(r, http.StatusUnauthorized, "401", "未登录")
		return
	}
	var req intentNodeBatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.WriteError(r, http.StatusBadRequest, "400", "请求参数错误")
		return
	}
	action(req.IDs)
	resp.WriteSuccess(r, nil)
}

func writeIntentTreeError(r *ghttp.Request, err error) {
	switch {
	case errors.Is(err, domainintenttree.ErrIntentCodeRequired),
		errors.Is(err, domainintenttree.ErrIntentNameRequired),
		errors.Is(err, domainintenttree.ErrIntentCodeExists),
		errors.Is(err, domainintenttree.ErrParentNotFound),
		errors.Is(err, domainintenttree.ErrTopKInvalid):
		resp.WriteError(r, http.StatusBadRequest, "400", err.Error())
	case errors.Is(err, domainintenttree.ErrNodeNotFound):
		resp.WriteError(r, http.StatusNotFound, "404", err.Error())
	default:
		resp.WriteError(r, http.StatusInternalServerError, "500", "意图树操作失败")
	}
}
