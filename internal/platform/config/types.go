package config

import "time"

// Config 表示 AgentRAG 当前阶段的最小配置集合。
type Config struct {
	HTTP     HTTPConfig     `json:"http"`
	Auth     AuthConfig     `json:"auth"`
	AI       AIConfig       `json:"ai"`
	RAG      RAGConfig      `json:"rag"`
	Database DatabaseConfig `json:"database"`
	State    StateConfig    `json:"state"`
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

// AIConfig 定义当前阶段最小可用的聊天模型配置。
type AIConfig struct {
	APIKey            string           `json:"apiKey"`
	BaseURL           string           `json:"baseUrl"`
	Model             string           `json:"model"`
	DeepThinkingModel string           `json:"deepThinkingModel"`
	SystemPrompt      string           `json:"systemPrompt"`
	Timeout           time.Duration    `json:"timeout"`
	Workflow          AIWorkflowConfig `json:"workflow"`
}

// AIWorkflowConfig 定义聊天 workflow 的可调策略配置。
type AIWorkflowConfig struct {
	RetrievalLimit  int    `json:"retrievalLimit"`
	RewritePrompt   string `json:"rewritePrompt"`
	ToolParamPrompt string `json:"toolParamPrompt"`
	KnowledgePrompt string `json:"knowledgePrompt"`
}

// RAGConfig 定义 RAG 运行策略配置。
type RAGConfig struct {
	RateLimit RAGRateLimitConfig `json:"rateLimit"`
}

// RAGRateLimitConfig 定义 RAG 限流配置。
type RAGRateLimitConfig struct {
	Global GlobalRateLimitConfig `json:"global"`
}

// GlobalRateLimitConfig 定义全局聊天并发限流配置。
type GlobalRateLimitConfig struct {
	Enabled        *bool `json:"enabled"`
	MaxConcurrent  int   `json:"maxConcurrent"`
	MaxWaitSeconds int   `json:"maxWaitSeconds"`
	LeaseSeconds   int   `json:"leaseSeconds"`
	PollIntervalMs int   `json:"pollIntervalMs"`
}

// StateConfig 定义本地状态持久化配置。
type StateConfig struct {
	File string `json:"file"`
}

// DatabaseConfig 定义关系型数据库连接配置。
type DatabaseConfig struct {
	Driver string `json:"driver"`
	DSN    string `json:"dsn"`
}
