package handlers

import (
	"github.com/AmazingCYJ/AgentRAG/internal/platform/resp"
	"github.com/gogf/gf/v2/net/ghttp"
)

// Health 返回最小健康状态，供服务探活与启动校验使用。
func Health(r *ghttp.Request) {
	resp.WriteSuccess(r, map[string]string{
		"status": "ok",
	})
}
