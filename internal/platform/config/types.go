package config

import "time"

// Config 表示 AgentRAG 当前阶段的最小配置集合。
type Config struct {
	HTTP HTTPConfig `json:"http"`
	Auth AuthConfig `json:"auth"`
}

// HTTPConfig 定义 HTTP 服务监听配置。
type HTTPConfig struct {
	Port int `json:"port"`
}

// AuthConfig 定义当前阶段认证相关配置。
type AuthConfig struct {
	JWTSecret string              `json:"jwtSecret"`
	TokenTTL  time.Duration       `json:"tokenTtl"`
	Bootstrap BootstrapUserConfig `json:"bootstrap"`
}

// BootstrapUserConfig 定义启动时内置的引导账号。
type BootstrapUserConfig struct {
	UserID   string `json:"userId"`
	Username string `json:"username"`
	Password string `json:"password"`
	Role     string `json:"role"`
	Avatar   string `json:"avatar"`
}
