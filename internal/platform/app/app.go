package app

import (
	"errors"
	"os"

	appconfig "github.com/AmazingCYJ/AgentRAG/internal/platform/config"
	"github.com/AmazingCYJ/AgentRAG/internal/platform/httpx"
	"github.com/gogf/gf/v2/net/ghttp"
)

// App 表示应用启动时的最小运行上下文。
type App struct {
	ConfigPath string
	Config     *appconfig.Config
	HTTPServer *ghttp.Server
}

// NewApp 负责在启动早期校验基础配置文件是否存在。
func NewApp(configPath string) (*App, error) {
	if configPath == "" {
		return nil, errors.New("配置文件路径不能为空")
	}
	if _, err := os.Stat(configPath); err != nil {
		return nil, err
	}
	cfg, err := appconfig.Load(configPath)
	if err != nil {
		return nil, err
	}
	server := httpx.NewServer(cfg)
	return &App{
		ConfigPath: configPath,
		Config:     cfg,
		HTTPServer: server,
	}, nil
}
