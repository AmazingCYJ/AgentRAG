package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	domainauth "github.com/AmazingCYJ/AgentRAG/internal/domain/auth"
	domainknowledge "github.com/AmazingCYJ/AgentRAG/internal/domain/knowledge"
	"github.com/AmazingCYJ/AgentRAG/internal/platform/resp"
	"github.com/gogf/gf/v2/net/ghttp"
)

// KnowledgeDocumentHandler 提供知识库文档接口。
type KnowledgeDocumentHandler struct {
	authService      *domainauth.Service
	knowledgeService *domainknowledge.Service
}

// NewKnowledgeDocumentHandler 创建文档处理器。
func NewKnowledgeDocumentHandler(authService *domainauth.Service, knowledgeService *domainknowledge.Service) *KnowledgeDocumentHandler {
	return &KnowledgeDocumentHandler{
		authService:      authService,
		knowledgeService: knowledgeService,
	}
}

type knowledgeDocumentUpdateRequest struct {
	DocName         string `json:"docName"`
	ProcessMode     string `json:"processMode"`
	ChunkStrategy   string `json:"chunkStrategy"`
	ChunkConfig     string `json:"chunkConfig"`
	PipelineID      string `json:"pipelineId"`
	SourceLocation  string `json:"sourceLocation"`
	ScheduleEnabled *int   `json:"scheduleEnabled"`
	ScheduleCron    string `json:"scheduleCron"`
}

// Upload 上传文档。
func (h *KnowledgeDocumentHandler) Upload(r *ghttp.Request) {
	profile, err := h.authService.CurrentUser(r.GetHeader("Authorization"))
	if err != nil {
		resp.WriteError(r, http.StatusUnauthorized, "401", "未登录")
		return
	}

	if err := r.Request.ParseMultipartForm(32 << 20); err != nil {
		resp.WriteError(r, http.StatusBadRequest, "400", "上传参数错误")
		return
	}
	file, fileHeader, _ := r.Request.FormFile("file")
	textContent := ""
	if file != nil {
		if content, readErr := io.ReadAll(io.LimitReader(file, 4<<20)); readErr == nil {
			textContent = string(content)
		}
		_ = file.Close()
	}

	item, err := h.knowledgeService.UploadDocument(r.Get("kb-id").String(), domainknowledge.KnowledgeDocumentUploadRequest{
		SourceType:      r.Request.FormValue("sourceType"),
		SourceLocation:  r.Request.FormValue("sourceLocation"),
		TextContent:     textContent,
		ScheduleEnabled: strings.EqualFold(r.Request.FormValue("scheduleEnabled"), "true"),
		ScheduleCron:    r.Request.FormValue("scheduleCron"),
		ProcessMode:     r.Request.FormValue("processMode"),
		ChunkStrategy:   r.Request.FormValue("chunkStrategy"),
		ChunkConfig:     r.Request.FormValue("chunkConfig"),
		PipelineID:      r.Request.FormValue("pipelineId"),
		FileName: func() string {
			if fileHeader != nil {
				return fileHeader.Filename
			}
			return ""
		}(),
		FileSize: func() int64 {
			if fileHeader != nil {
				return fileHeader.Size
			}
			return 0
		}(),
	}, profile.Username)
	if err != nil {
		writeKnowledgeError(r, err)
		return
	}
	resp.WriteSuccess(r, item)
}

// StartChunk 启动文档分块。
func (h *KnowledgeDocumentHandler) StartChunk(r *ghttp.Request) {
	if _, err := h.authService.CurrentUser(r.GetHeader("Authorization")); err != nil {
		resp.WriteError(r, http.StatusUnauthorized, "401", "未登录")
		return
	}
	if err := h.knowledgeService.StartDocumentChunk(r.Get("doc-id").String()); err != nil {
		writeKnowledgeError(r, err)
		return
	}
	resp.WriteSuccess(r, nil)
}

// Delete 删除文档。
func (h *KnowledgeDocumentHandler) Delete(r *ghttp.Request) {
	if _, err := h.authService.CurrentUser(r.GetHeader("Authorization")); err != nil {
		resp.WriteError(r, http.StatusUnauthorized, "401", "未登录")
		return
	}
	if err := h.knowledgeService.DeleteDocument(r.Get("doc-id").String()); err != nil {
		writeKnowledgeError(r, err)
		return
	}
	resp.WriteSuccess(r, nil)
}

// Get 查询文档详情。
func (h *KnowledgeDocumentHandler) Get(r *ghttp.Request) {
	if _, err := h.authService.CurrentUser(r.GetHeader("Authorization")); err != nil {
		resp.WriteError(r, http.StatusUnauthorized, "401", "未登录")
		return
	}
	item, err := h.knowledgeService.GetDocument(r.Get("docId").String())
	if err != nil {
		writeKnowledgeError(r, err)
		return
	}
	resp.WriteSuccess(r, item)
}

// Update 更新文档。
func (h *KnowledgeDocumentHandler) Update(r *ghttp.Request) {
	profile, err := h.authService.CurrentUser(r.GetHeader("Authorization"))
	if err != nil {
		resp.WriteError(r, http.StatusUnauthorized, "401", "未登录")
		return
	}
	var req knowledgeDocumentUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.WriteError(r, http.StatusBadRequest, "400", "请求参数错误")
		return
	}
	if err := h.knowledgeService.UpdateDocument(r.Get("docId").String(), domainknowledge.KnowledgeDocumentUpdateRequest{
		DocName:         req.DocName,
		ProcessMode:     req.ProcessMode,
		ChunkStrategy:   req.ChunkStrategy,
		ChunkConfig:     req.ChunkConfig,
		PipelineID:      req.PipelineID,
		SourceLocation:  req.SourceLocation,
		ScheduleEnabled: req.ScheduleEnabled,
		ScheduleCron:    req.ScheduleCron,
		UpdatedBy:       profile.Username,
	}); err != nil {
		writeKnowledgeError(r, err)
		return
	}
	resp.WriteSuccess(r, nil)
}

// Page 分页查询文档。
func (h *KnowledgeDocumentHandler) Page(r *ghttp.Request) {
	if _, err := h.authService.CurrentUser(r.GetHeader("Authorization")); err != nil {
		resp.WriteError(r, http.StatusUnauthorized, "401", "未登录")
		return
	}
	page, err := h.knowledgeService.PageDocuments(r.Get("kb-id").String(), domainknowledge.KnowledgeDocumentPageRequest{
		Current: r.Get("current").Int(),
		Size:    r.Get("size").Int(),
		Status:  strings.TrimSpace(r.Get("status").String()),
		Keyword: strings.TrimSpace(r.Get("keyword").String()),
	})
	if err != nil {
		writeKnowledgeError(r, err)
		return
	}
	resp.WriteSuccess(r, page)
}

// Search 搜索文档。
func (h *KnowledgeDocumentHandler) Search(r *ghttp.Request) {
	if _, err := h.authService.CurrentUser(r.GetHeader("Authorization")); err != nil {
		resp.WriteError(r, http.StatusUnauthorized, "401", "未登录")
		return
	}
	resp.WriteSuccess(r, h.knowledgeService.SearchDocuments(
		strings.TrimSpace(r.Get("keyword").String()),
		r.Get("limit").Int(),
	))
}

// Enable 启用或禁用文档。
func (h *KnowledgeDocumentHandler) Enable(r *ghttp.Request) {
	if _, err := h.authService.CurrentUser(r.GetHeader("Authorization")); err != nil {
		resp.WriteError(r, http.StatusUnauthorized, "401", "未登录")
		return
	}
	if err := h.knowledgeService.EnableDocument(r.Get("docId").String(), r.Get("value").Bool()); err != nil {
		writeKnowledgeError(r, err)
		return
	}
	resp.WriteSuccess(r, nil)
}

// ChunkLogs 查询文档分块日志。
func (h *KnowledgeDocumentHandler) ChunkLogs(r *ghttp.Request) {
	if _, err := h.authService.CurrentUser(r.GetHeader("Authorization")); err != nil {
		resp.WriteError(r, http.StatusUnauthorized, "401", "未登录")
		return
	}
	page, err := h.knowledgeService.PageChunkLogs(r.Get("docId").String(), r.Get("current").Int(), r.Get("size").Int())
	if err != nil {
		writeKnowledgeError(r, err)
		return
	}
	resp.WriteSuccess(r, page)
}

func isKnowledgeNotFound(err error) bool {
	return errors.Is(err, domainknowledge.ErrKnowledgeBaseNotFound) ||
		errors.Is(err, domainknowledge.ErrDocumentNotFound) ||
		errors.Is(err, domainknowledge.ErrChunkNotFound)
}
