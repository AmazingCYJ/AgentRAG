package config

// Config 表示 AgentRAG 当前阶段的最小配置集合。
type Config struct {
	HTTP HTTPConfig `json:"http"`
}

// HTTPConfig 定义 HTTP 服务监听配置。
type HTTPConfig struct {
	Port int `json:"port"`
}
