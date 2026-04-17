package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	domainauth "github.com/AmazingCYJ/AgentRAG/internal/domain/auth"
	domainknowledge "github.com/AmazingCYJ/AgentRAG/internal/domain/knowledge"
	"github.com/AmazingCYJ/AgentRAG/internal/platform/resp"
	"github.com/gogf/gf/v2/net/ghttp"
)

// KnowledgeChunkHandler 提供文档 Chunk 管理接口。
type KnowledgeChunkHandler struct {
	authService      *domainauth.Service
	knowledgeService *domainknowledge.Service
}

// NewKnowledgeChunkHandler 创建 Chunk 处理器。
func NewKnowledgeChunkHandler(authService *domainauth.Service, knowledgeService *domainknowledge.Service) *KnowledgeChunkHandler {
	return &KnowledgeChunkHandler{
		authService:      authService,
		knowledgeService: knowledgeService,
	}
}

type knowledgeChunkSaveRequest struct {
	Content string `json:"content"`
	Index   *int   `json:"index"`
	ChunkID string `json:"chunkId"`
}

type knowledgeChunkBatchRequest struct {
	ChunkIDs []string `json:"chunkIds"`
}

// Page 分页查询 Chunk。
func (h *KnowledgeChunkHandler) Page(r *ghttp.Request) {
	if _, err := h.authService.CurrentUser(r.GetHeader("Authorization")); err != nil {
		resp.WriteError(r, http.StatusUnauthorized, "401", "未登录")
		return
	}
	var enabled *int
	if value := strings.TrimSpace(r.Get("enabled").String()); value != "" {
		parsed := r.Get("enabled").Int()
		enabled = &parsed
	}
	page, err := h.knowledgeService.PageChunks(r.Get("doc-id").String(), domainknowledge.KnowledgeChunkPageRequest{
		Current: r.Get("current").Int(),
		Size:    r.Get("size").Int(),
		Enabled: enabled,
	})
	if err != nil {
		writeKnowledgeError(r, err)
		return
	}
	resp.WriteSuccess(r, page)
}

// Create 创建 Chunk。
func (h *KnowledgeChunkHandler) Create(r *ghttp.Request) {
	if _, err := h.authService.CurrentUser(r.GetHeader("Authorization")); err != nil {
		resp.WriteError(r, http.StatusUnauthorized, "401", "未登录")
		return
	}
	var req knowledgeChunkSaveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.WriteError(r, http.StatusBadRequest, "400", "请求参数错误")
		return
	}
	item, err := h.knowledgeService.CreateChunk(r.Get("doc-id").String(), domainknowledge.KnowledgeChunkCreateRequest{
		Content: req.Content,
		Index:   req.Index,
		ChunkID: req.ChunkID,
	})
	if err != nil {
		writeKnowledgeError(r, err)
		return
	}
	resp.WriteSuccess(r, item)
}

// Update 更新 Chunk 内容。
func (h *KnowledgeChunkHandler) Update(r *ghttp.Request) {
	if _, err := h.authService.CurrentUser(r.GetHeader("Authorization")); err != nil {
		resp.WriteError(r, http.StatusUnauthorized, "401", "未登录")
		return
	}
	var req knowledgeChunkSaveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.WriteError(r, http.StatusBadRequest, "400", "请求参数错误")
		return
	}
	if err := h.knowledgeService.UpdateChunk(r.Get("doc-id").String(), r.Get("chunk-id").String(), domainknowledge.KnowledgeChunkUpdateRequest{
		Content: req.Content,
	}); err != nil {
		writeKnowledgeError(r, err)
		return
	}
	resp.WriteSuccess(r, nil)
}

// Delete 删除 Chunk。
func (h *KnowledgeChunkHandler) Delete(r *ghttp.Request) {
	if _, err := h.authService.CurrentUser(r.GetHeader("Authorization")); err != nil {
		resp.WriteError(r, http.StatusUnauthorized, "401", "未登录")
		return
	}
	if err := h.knowledgeService.DeleteChunk(r.Get("doc-id").String(), r.Get("chunk-id").String()); err != nil {
		writeKnowledgeError(r, err)
		return
	}
	resp.WriteSuccess(r, nil)
}

// Enable 启用或禁用单个 Chunk。
func (h *KnowledgeChunkHandler) Enable(r *ghttp.Request) {
	if _, err := h.authService.CurrentUser(r.GetHeader("Authorization")); err != nil {
		resp.WriteError(r, http.StatusUnauthorized, "401", "未登录")
		return
	}
	if err := h.knowledgeService.ToggleChunk(r.Get("doc-id").String(), r.Get("chunk-id").String(), r.Get("value").Bool()); err != nil {
		writeKnowledgeError(r, err)
		return
	}
	resp.WriteSuccess(r, nil)
}

// BatchEnable 批量启用或禁用 Chunk。
func (h *KnowledgeChunkHandler) BatchEnable(r *ghttp.Request) {
	if _, err := h.authService.CurrentUser(r.GetHeader("Authorization")); err != nil {
		resp.WriteError(r, http.StatusUnauthorized, "401", "未登录")
		return
	}
	var req knowledgeChunkBatchRequest
	if r.GetBodyString() != "" {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			resp.WriteError(r, http.StatusBadRequest, "400", "请求参数错误")
			return
		}
	}
	if err := h.knowledgeService.BatchToggleChunks(r.Get("doc-id").String(), req.ChunkIDs, r.Get("value").Bool()); err != nil {
		writeKnowledgeError(r, err)
		return
	}
	resp.WriteSuccess(r, nil)
}
