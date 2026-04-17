package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	domainauth "github.com/AmazingCYJ/AgentRAG/internal/domain/auth"
	domainsamplequestion "github.com/AmazingCYJ/AgentRAG/internal/domain/samplequestion"
	"github.com/AmazingCYJ/AgentRAG/internal/platform/resp"
	"github.com/gogf/gf/v2/net/ghttp"
)

// SampleQuestionHandler 提供示例问题相关接口。
type SampleQuestionHandler struct {
	authService           *domainauth.Service
	sampleQuestionService *domainsamplequestion.Service
}

// NewSampleQuestionHandler 创建示例问题处理器。
func NewSampleQuestionHandler(authService *domainauth.Service, sampleQuestionService *domainsamplequestion.Service) *SampleQuestionHandler {
	return &SampleQuestionHandler{
		authService:           authService,
		sampleQuestionService: sampleQuestionService,
	}
}

type sampleQuestionSaveRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Question    string `json:"question"`
}

// ListWelcome 返回欢迎页示例问题。
func (h *SampleQuestionHandler) ListWelcome(r *ghttp.Request) {
	if _, err := h.authService.CurrentUser(r.GetHeader("Authorization")); err != nil {
		resp.WriteError(r, http.StatusUnauthorized, "401", "未登录")
		return
	}
	resp.WriteSuccess(r, h.sampleQuestionService.ListWelcome())
}

// Page 返回示例问题分页结果。
func (h *SampleQuestionHandler) Page(r *ghttp.Request) {
	if _, err := h.authService.CurrentUser(r.GetHeader("Authorization")); err != nil {
		resp.WriteError(r, http.StatusUnauthorized, "401", "未登录")
		return
	}

	current := r.Get("current").Int()
	size := r.Get("size").Int()
	keyword := strings.TrimSpace(r.Get("keyword").String())
	resp.WriteSuccess(r, h.sampleQuestionService.Page(current, size, keyword))
}

// Get 返回示例问题详情。
func (h *SampleQuestionHandler) Get(r *ghttp.Request) {
	if _, err := h.authService.CurrentUser(r.GetHeader("Authorization")); err != nil {
		resp.WriteError(r, http.StatusUnauthorized, "401", "未登录")
		return
	}

	item, err := h.sampleQuestionService.GetByID(r.Get("id").String())
	if err != nil {
		resp.WriteError(r, http.StatusNotFound, "404", err.Error())
		return
	}
	resp.WriteSuccess(r, item)
}

// Create 新增示例问题。
func (h *SampleQuestionHandler) Create(r *ghttp.Request) {
	if _, err := h.authService.CurrentUser(r.GetHeader("Authorization")); err != nil {
		resp.WriteError(r, http.StatusUnauthorized, "401", "未登录")
		return
	}

	var req sampleQuestionSaveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.WriteError(r, http.StatusBadRequest, "400", "请求参数错误")
		return
	}
	id, err := h.sampleQuestionService.Create(domainsamplequestion.SaveRequest{
		Title:       req.Title,
		Description: req.Description,
		Question:    req.Question,
	})
	if err != nil {
		if errors.Is(err, domainsamplequestion.ErrQuestionRequired) {
			resp.WriteError(r, http.StatusBadRequest, "400", err.Error())
			return
		}
		resp.WriteError(r, http.StatusInternalServerError, "500", "创建示例问题失败")
		return
	}
	resp.WriteSuccess(r, id)
}

// Update 修改示例问题。
func (h *SampleQuestionHandler) Update(r *ghttp.Request) {
	if _, err := h.authService.CurrentUser(r.GetHeader("Authorization")); err != nil {
		resp.WriteError(r, http.StatusUnauthorized, "401", "未登录")
		return
	}

	var req sampleQuestionSaveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.WriteError(r, http.StatusBadRequest, "400", "请求参数错误")
		return
	}
	err := h.sampleQuestionService.Update(r.Get("id").String(), domainsamplequestion.SaveRequest{
		Title:       req.Title,
		Description: req.Description,
		Question:    req.Question,
	})
	if err != nil {
		if errors.Is(err, domainsamplequestion.ErrQuestionRequired) {
			resp.WriteError(r, http.StatusBadRequest, "400", err.Error())
			return
		}
		if errors.Is(err, domainsamplequestion.ErrQuestionNotFound) {
			resp.WriteError(r, http.StatusNotFound, "404", err.Error())
			return
		}
		resp.WriteError(r, http.StatusInternalServerError, "500", "更新示例问题失败")
		return
	}
	resp.WriteSuccess(r, nil)
}

// Delete 删除示例问题。
func (h *SampleQuestionHandler) Delete(r *ghttp.Request) {
	if _, err := h.authService.CurrentUser(r.GetHeader("Authorization")); err != nil {
		resp.WriteError(r, http.StatusUnauthorized, "401", "未登录")
		return
	}

	if err := h.sampleQuestionService.Delete(r.Get("id").String()); err != nil {
		resp.WriteError(r, http.StatusNotFound, "404", err.Error())
		return
	}
	resp.WriteSuccess(r, nil)
}
