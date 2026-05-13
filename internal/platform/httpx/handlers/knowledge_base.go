package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	domainauth "github.com/AmazingCYJ/AgentRAG/internal/domain/auth"
	domainknowledge "github.com/AmazingCYJ/AgentRAG/internal/domain/knowledge"
	"github.com/AmazingCYJ/AgentRAG/internal/platform/resp"
	"github.com/gogf/gf/v2/net/ghttp"
)

// KnowledgeBaseHandler 提供知识库基础管理接口。
type KnowledgeBaseHandler struct {
	authService      *domainauth.Service
	knowledgeService *domainknowledge.Service
}

// NewKnowledgeBaseHandler 创建知识库处理器。
func NewKnowledgeBaseHandler(authService *domainauth.Service, knowledgeService *domainknowledge.Service) *KnowledgeBaseHandler {
	return &KnowledgeBaseHandler{
		authService:      authService,
		knowledgeService: knowledgeService,
	}
}

type knowledgeBaseSaveRequest struct {
	Name           string `json:"name"`
	EmbeddingModel string `json:"embeddingModel"`
	CollectionName string `json:"collectionName"`
}

// Create 创建知识库。
func (h *KnowledgeBaseHandler) Create(r *ghttp.Request) {
	profile, err := h.authService.CurrentUser(r.GetHeader("Authorization"))
	if err != nil {
		resp.WriteError(r, http.StatusUnauthorized, "401", "未登录")
		return
	}

	var req knowledgeBaseSaveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.WriteError(r, http.StatusBadRequest, "400", "请求参数错误")
		return
	}
	id, err := h.knowledgeService.CreateKnowledgeBase(domainknowledge.KnowledgeBaseCreateRequest{
		Name:           req.Name,
		EmbeddingModel: req.EmbeddingModel,
		CollectionName: req.CollectionName,
		CreatedBy:      profile.Username,
	})
	if err != nil {
		writeKnowledgeError(r, err)
		return
	}
	resp.WriteSuccess(r, id)
}

// Update 更新知识库。
func (h *KnowledgeBaseHandler) Update(r *ghttp.Request) {
	if _, err := h.authService.CurrentUser(r.GetHeader("Authorization")); err != nil {
		resp.WriteError(r, http.StatusUnauthorized, "401", "未登录")
		return
	}

	var req knowledgeBaseSaveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.WriteError(r, http.StatusBadRequest, "400", "请求参数错误")
		return
	}
	if err := h.knowledgeService.UpdateKnowledgeBase(r.Get("kb-id").String(), domainknowledge.KnowledgeBaseUpdateRequest{
		Name:           req.Name,
		EmbeddingModel: req.EmbeddingModel,
	}); err != nil {
		writeKnowledgeError(r, err)
		return
	}
	resp.WriteSuccess(r, nil)
}

// Delete 删除知识库。
func (h *KnowledgeBaseHandler) Delete(r *ghttp.Request) {
	if _, err := h.authService.CurrentUser(r.GetHeader("Authorization")); err != nil {
		resp.WriteError(r, http.StatusUnauthorized, "401", "未登录")
		return
	}
	if err := h.knowledgeService.DeleteKnowledgeBase(r.Context(), r.Get("kb-id").String()); err != nil {
		writeKnowledgeError(r, err)
		return
	}
	resp.WriteSuccess(r, nil)
}

// Get 查询知识库详情。
func (h *KnowledgeBaseHandler) Get(r *ghttp.Request) {
	if _, err := h.authService.CurrentUser(r.GetHeader("Authorization")); err != nil {
		resp.WriteError(r, http.StatusUnauthorized, "401", "未登录")
		return
	}
	item, err := h.knowledgeService.GetKnowledgeBase(r.Get("kb-id").String())
	if err != nil {
		writeKnowledgeError(r, err)
		return
	}
	resp.WriteSuccess(r, item)
}

// Page 分页查询知识库。
func (h *KnowledgeBaseHandler) Page(r *ghttp.Request) {
	if _, err := h.authService.CurrentUser(r.GetHeader("Authorization")); err != nil {
		resp.WriteError(r, http.StatusUnauthorized, "401", "未登录")
		return
	}
	resp.WriteSuccess(r, h.knowledgeService.PageKnowledgeBases(domainknowledge.KnowledgeBasePageRequest{
		Current: r.Get("current").Int(),
		Size:    r.Get("size").Int(),
		Name:    strings.TrimSpace(r.Get("name").String()),
	}))
}

// ChunkStrategies 返回支持的分块策略。
func (h *KnowledgeBaseHandler) ChunkStrategies(r *ghttp.Request) {
	if _, err := h.authService.CurrentUser(r.GetHeader("Authorization")); err != nil {
		resp.WriteError(r, http.StatusUnauthorized, "401", "未登录")
		return
	}
	resp.WriteSuccess(r, h.knowledgeService.ListChunkStrategies())
}

func writeKnowledgeError(r *ghttp.Request, err error) {
	switch {
	case errors.Is(err, domainknowledge.ErrKnowledgeBaseNotFound),
		errors.Is(err, domainknowledge.ErrDocumentNotFound),
		errors.Is(err, domainknowledge.ErrChunkNotFound):
		resp.WriteError(r, http.StatusNotFound, "404", err.Error())
	case errors.Is(err, domainknowledge.ErrKnowledgeBaseNameRequired),
		errors.Is(err, domainknowledge.ErrEmbeddingModelRequired),
		errors.Is(err, domainknowledge.ErrCollectionNameRequired),
		errors.Is(err, domainknowledge.ErrDocumentNameRequired),
		errors.Is(err, domainknowledge.ErrChunkContentRequired):
		resp.WriteError(r, http.StatusBadRequest, "400", err.Error())
	default:
		resp.WriteError(r, http.StatusInternalServerError, "500", "知识库操作失败")
	}
}
