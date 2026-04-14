package resp

import "github.com/gogf/gf/v2/net/ghttp"

const (
	// CodeOK 表示成功响应码，需兼容前端既有约定。
	CodeOK = "0"
)

// Body 定义统一响应结构。
type Body struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// WriteSuccess 写入成功响应。
func WriteSuccess(r *ghttp.Request, data any) {
	r.Response.WriteJson(Body{
		Code:    CodeOK,
		Message: "success",
		Data:    data,
	})
}

// WriteError 写入失败响应并保留 HTTP 状态码。
func WriteError(r *ghttp.Request, httpStatus int, code, message string) {
	r.Response.WriteHeader(httpStatus)
	r.Response.WriteJson(Body{
		Code:    code,
		Message: message,
	})
}
