package handlers

import (
	"encoding/json"
	"io"
	"net/http"

	domainauth "github.com/AmazingCYJ/AgentRAG/internal/domain/auth"
	domainingestion "github.com/AmazingCYJ/AgentRAG/internal/domain/ingestion"
	"github.com/AmazingCYJ/AgentRAG/internal/platform/resp"
	"github.com/gogf/gf/v2/net/ghttp"
)

// IngestionTaskHandler 提供采集任务接口。
type IngestionTaskHandler struct {
	authService      *domainauth.Service
	ingestionService *domainingestion.Service
}

// NewIngestionTaskHandler 创建采集任务处理器。
func NewIngestionTaskHandler(authService *domainauth.Service, ingestionService *domainingestion.Service) *IngestionTaskHandler {
	return &IngestionTaskHandler{
		authService:      authService,
		ingestionService: ingestionService,
	}
}

type ingestionTaskCreateRequest struct {
	PipelineID    string                     `json:"pipelineId"`
	Source        domainingestion.TaskSource `json:"source"`
	Metadata      map[string]any             `json:"metadata"`
	VectorSpaceID map[string]any             `json:"vectorSpaceId"`
}

// Create 创建并执行采集任务。
func (h *IngestionTaskHandler) Create(r *ghttp.Request) {
	profile, err := h.authService.CurrentUser(r.GetHeader("Authorization"))
	if err != nil {
		resp.WriteError(r, http.StatusUnauthorized, "401", "未登录")
		return
	}
	var req ingestionTaskCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.WriteError(r, http.StatusBadRequest, "400", "请求参数错误")
		return
	}
	result, err := h.ingestionService.CreateTask(domainingestion.TaskCreateRequest{
		PipelineID:    req.PipelineID,
		Source:        req.Source,
		Metadata:      req.Metadata,
		VectorSpaceID: req.VectorSpaceID,
		CreatedBy:     profile.Username,
	})
	if err != nil {
		writeIngestionError(r, err)
		return
	}
	resp.WriteSuccess(r, result)
}

// Upload 上传文件并执行采集任务。
func (h *IngestionTaskHandler) Upload(r *ghttp.Request) {
	profile, err := h.authService.CurrentUser(r.GetHeader("Authorization"))
	if err != nil {
		resp.WriteError(r, http.StatusUnauthorized, "401", "未登录")
		return
	}
	if err := r.Request.ParseMultipartForm(32 << 20); err != nil {
		resp.WriteError(r, http.StatusBadRequest, "400", "上传参数错误")
		return
	}
	file, fileHeader, err := r.Request.FormFile("file")
	if err != nil {
		resp.WriteError(r, http.StatusBadRequest, "400", "缺少上传文件")
		return
	}
	defer file.Close()
	content, err := io.ReadAll(io.LimitReader(file, 4<<20))
	if err != nil {
		resp.WriteError(r, http.StatusBadRequest, "400", "读取上传文件失败")
		return
	}

	result, err := h.ingestionService.UploadTask(domainingestion.UploadTaskRequest{
		PipelineID: r.Get("pipelineId").String(),
		FileName:   fileHeader.Filename,
		FileSize:   fileHeader.Size,
		Content:    content,
		CreatedBy:  profile.Username,
	})
	if err != nil {
		writeIngestionError(r, err)
		return
	}
	resp.WriteSuccess(r, result)
}

// Get 查询任务详情。
func (h *IngestionTaskHandler) Get(r *ghttp.Request) {
	if _, err := h.authService.CurrentUser(r.GetHeader("Authorization")); err != nil {
		resp.WriteError(r, http.StatusUnauthorized, "401", "未登录")
		return
	}
	item, err := h.ingestionService.GetTask(r.Get("id").String())
	if err != nil {
		writeIngestionError(r, err)
		return
	}
	resp.WriteSuccess(r, item)
}

// Nodes 查询任务节点记录。
func (h *IngestionTaskHandler) Nodes(r *ghttp.Request) {
	if _, err := h.authService.CurrentUser(r.GetHeader("Authorization")); err != nil {
		resp.WriteError(r, http.StatusUnauthorized, "401", "未登录")
		return
	}
	items, err := h.ingestionService.ListTaskNodes(r.Get("id").String())
	if err != nil {
		writeIngestionError(r, err)
		return
	}
	resp.WriteSuccess(r, items)
}

// Page 分页查询任务。
func (h *IngestionTaskHandler) Page(r *ghttp.Request) {
	if _, err := h.authService.CurrentUser(r.GetHeader("Authorization")); err != nil {
		resp.WriteError(r, http.StatusUnauthorized, "401", "未登录")
		return
	}
	resp.WriteSuccess(r, h.ingestionService.PageTasks(
		r.Get("pageNo").Int(),
		r.Get("pageSize").Int(),
		r.Get("status").String(),
	))
}
