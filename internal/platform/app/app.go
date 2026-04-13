package app

import (
	"errors"
	"os"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
)

// App 表示应用启动时的最小运行上下文。
type App struct {
	ConfigPath string
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
	return &App{
		ConfigPath: configPath,
		HTTPServer: g.Server("agentrag-api"),
	}, nil
}
