package httpx

import (
	appconfig "github.com/AmazingCYJ/AgentRAG/internal/platform/config"
	"github.com/AmazingCYJ/AgentRAG/internal/platform/httpx/handlers"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
)

// NewServer 构造当前阶段的最小 HTTP 服务，并注册基础路由。
func NewServer(cfg *appconfig.Config) *ghttp.Server {
	return newServer(cfg, "agentrag-api")
}

func newServer(cfg *appconfig.Config, name string) *ghttp.Server {
	server := g.Server(name)
	server.SetDumpRouterMap(false)
	if cfg != nil && cfg.HTTP.Port > 0 {
		server.SetPort(cfg.HTTP.Port)
	}
	server.BindHandler("/health", handlers.Health)
	return server
}
