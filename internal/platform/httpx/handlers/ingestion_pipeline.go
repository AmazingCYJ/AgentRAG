package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	domainauth "github.com/AmazingCYJ/AgentRAG/internal/domain/auth"
	domainingestion "github.com/AmazingCYJ/AgentRAG/internal/domain/ingestion"
	"github.com/AmazingCYJ/AgentRAG/internal/platform/resp"
	"github.com/gogf/gf/v2/net/ghttp"
)

// IngestionPipelineHandler 提供流水线接口。
type IngestionPipelineHandler struct {
	authService      *domainauth.Service
	ingestionService *domainingestion.Service
}

// NewIngestionPipelineHandler 创建流水线处理器。
func NewIngestionPipelineHandler(authService *domainauth.Service, ingestionService *domainingestion.Service) *IngestionPipelineHandler {
	return &IngestionPipelineHandler{
		authService:      authService,
		ingestionService: ingestionService,
	}
}

type ingestionPipelineSaveRequest struct {
	Name        string                                `json:"name"`
	Description string                                `json:"description"`
	Nodes       []domainingestion.PipelineNodeRequest `json:"nodes"`
}

// Create 创建流水线。
func (h *IngestionPipelineHandler) Create(r *ghttp.Request) {
	profile, err := h.authService.CurrentUser(r.GetHeader("Authorization"))
	if err != nil {
		resp.WriteError(r, http.StatusUnauthorized, "401", "未登录")
		return
	}
	var req ingestionPipelineSaveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.WriteError(r, http.StatusBadRequest, "400", "请求参数错误")
		return
	}
	item, err := h.ingestionService.CreatePipeline(domainingestion.PipelineSaveRequest{
		Name:        req.Name,
		Description: req.Description,
		Nodes:       req.Nodes,
		CreatedBy:   profile.Username,
	})
	if err != nil {
		writeIngestionError(r, err)
		return
	}
	resp.WriteSuccess(r, item)
}

// Update 更新流水线。
func (h *IngestionPipelineHandler) Update(r *ghttp.Request) {
	if _, err := h.authService.CurrentUser(r.GetHeader("Authorization")); err != nil {
		resp.WriteError(r, http.StatusUnauthorized, "401", "未登录")
		return
	}
	var req ingestionPipelineSaveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.WriteError(r, http.StatusBadRequest, "400", "请求参数错误")
		return
	}
	item, err := h.ingestionService.UpdatePipeline(r.Get("id").String(), domainingestion.PipelineSaveRequest{
		Name:        req.Name,
		Description: req.Description,
		Nodes:       req.Nodes,
	})
	if err != nil {
		writeIngestionError(r, err)
		return
	}
	resp.WriteSuccess(r, item)
}

// Get 查询单个流水线。
func (h *IngestionPipelineHandler) Get(r *ghttp.Request) {
	if _, err := h.authService.CurrentUser(r.GetHeader("Authorization")); err != nil {
		resp.WriteError(r, http.StatusUnauthorized, "401", "未登录")
		return
	}
	item, err := h.ingestionService.GetPipeline(r.Get("id").String())
	if err != nil {
		writeIngestionError(r, err)
		return
	}
	resp.WriteSuccess(r, item)
}

// Page 分页查询流水线。
func (h *IngestionPipelineHandler) Page(r *ghttp.Request) {
	if _, err := h.authService.CurrentUser(r.GetHeader("Authorization")); err != nil {
		resp.WriteError(r, http.StatusUnauthorized, "401", "未登录")
		return
	}
	resp.WriteSuccess(r, h.ingestionService.PagePipelines(
		r.Get("pageNo").Int(),
		r.Get("pageSize").Int(),
		strings.TrimSpace(r.Get("keyword").String()),
	))
}

// Delete 删除流水线。
func (h *IngestionPipelineHandler) Delete(r *ghttp.Request) {
	if _, err := h.authService.CurrentUser(r.GetHeader("Authorization")); err != nil {
		resp.WriteError(r, http.StatusUnauthorized, "401", "未登录")
		return
	}
	if err := h.ingestionService.DeletePipeline(r.Get("id").String()); err != nil {
		writeIngestionError(r, err)
		return
	}
	resp.WriteSuccess(r, nil)
}

func writeIngestionError(r *ghttp.Request, err error) {
	switch {
	case errors.Is(err, domainingestion.ErrPipelineNotFound), errors.Is(err, domainingestion.ErrTaskNotFound):
		resp.WriteError(r, http.StatusNotFound, "404", err.Error())
	case errors.Is(err, domainingestion.ErrPipelineNameRequired), errors.Is(err, domainingestion.ErrPipelineIDRequired):
		resp.WriteError(r, http.StatusBadRequest, "400", err.Error())
	default:
		resp.WriteError(r, http.StatusInternalServerError, "500", "数据通道操作失败")
	}
}
