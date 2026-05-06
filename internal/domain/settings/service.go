package settings

import (
	"strings"

	appconfig "github.com/AmazingCYJ/AgentRAG/internal/platform/config"
)

// Service 提供当前阶段只读系统配置视图。
type Service struct {
	ai appconfig.AIConfig
}

// NewService 创建系统配置服务。
func NewService(cfg ...appconfig.AIConfig) *Service {
	service := &Service{}
	if len(cfg) > 0 {
		service.ai = cfg[0]
	}
	return service
}

// SystemSettings 定义前端系统配置页所需结构。
type SystemSettings struct {
	Upload UploadSettings `json:"upload"`
	RAG    RAGSettings    `json:"rag"`
	AI     AISettings     `json:"ai"`
}

// UploadSettings 定义上传限制。
type UploadSettings struct {
	MaxFileSize    int64 `json:"maxFileSize"`
	MaxRequestSize int64 `json:"maxRequestSize"`
}

// RAGSettings 定义 RAG 相关配置。
type RAGSettings struct {
	Default      DefaultSettings      `json:"default"`
	QueryRewrite QueryRewriteSettings `json:"queryRewrite"`
	RateLimit    RateLimitSettings    `json:"rateLimit"`
	Memory       MemorySettings       `json:"memory"`
}

// DefaultSettings 定义默认向量集合配置。
type DefaultSettings struct {
	CollectionName string `json:"collectionName"`
	Dimension      int    `json:"dimension"`
	MetricType     string `json:"metricType"`
}

// QueryRewriteSettings 定义查询改写配置。
type QueryRewriteSettings struct {
	Enabled            bool `json:"enabled"`
	MaxHistoryMessages int  `json:"maxHistoryMessages"`
	MaxHistoryChars    int  `json:"maxHistoryChars"`
}

// RateLimitSettings 定义限流配置。
type RateLimitSettings struct {
	Global GlobalRateLimit `json:"global"`
}

// GlobalRateLimit 定义全局并发限制。
type GlobalRateLimit struct {
	Enabled        bool `json:"enabled"`
	MaxConcurrent  int  `json:"maxConcurrent"`
	MaxWaitSeconds int  `json:"maxWaitSeconds"`
	LeaseSeconds   int  `json:"leaseSeconds"`
	PollIntervalMs int  `json:"pollIntervalMs"`
}

// MemorySettings 定义上下文记忆相关配置。
type MemorySettings struct {
	HistoryKeepTurns  int  `json:"historyKeepTurns"`
	SummaryStartTurns int  `json:"summaryStartTurns"`
	SummaryEnabled    bool `json:"summaryEnabled"`
	TTLMinutes        int  `json:"ttlMinutes"`
	SummaryMaxChars   int  `json:"summaryMaxChars"`
	TitleMaxLength    int  `json:"titleMaxLength"`
}

// AISettings 定义 AI 模型相关配置。
type AISettings struct {
	Providers map[string]ProviderConfig `json:"providers"`
	Selection SelectionSettings         `json:"selection"`
	Stream    StreamSettings            `json:"stream"`
	Chat      ModelGroup                `json:"chat"`
	Embedding ModelGroup                `json:"embedding"`
	Rerank    ModelGroup                `json:"rerank"`
}

// ProviderConfig 定义模型提供方配置。
type ProviderConfig struct {
	URL       string            `json:"url"`
	APIKey    string            `json:"apiKey,omitempty"`
	Endpoints map[string]string `json:"endpoints"`
}

// SelectionSettings 定义模型选择策略。
type SelectionSettings struct {
	FailureThreshold int `json:"failureThreshold"`
	OpenDurationMs   int `json:"openDurationMs"`
}

// StreamSettings 定义流式输出配置。
type StreamSettings struct {
	MessageChunkSize int `json:"messageChunkSize"`
}

// ModelGroup 定义一组模型配置。
type ModelGroup struct {
	DefaultModel      string           `json:"defaultModel,omitempty"`
	DeepThinkingModel string           `json:"deepThinkingModel,omitempty"`
	Candidates        []ModelCandidate `json:"candidates"`
}

// ModelCandidate 定义单个模型候选项。
type ModelCandidate struct {
	ID               string `json:"id"`
	Provider         string `json:"provider"`
	Model            string `json:"model"`
	URL              string `json:"url,omitempty"`
	Dimension        int    `json:"dimension,omitempty"`
	Priority         int    `json:"priority,omitempty"`
	Enabled          bool   `json:"enabled"`
	SupportsThinking bool   `json:"supportsThinking"`
}

// Get 返回当前阶段的默认系统配置。
func (s *Service) Get() SystemSettings {
	baseURL := strings.TrimSpace(s.ai.BaseURL)
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	chatGroup := s.buildChatModelGroup(baseURL)
	return SystemSettings{
		Upload: UploadSettings{
			MaxFileSize:    50 * 1024 * 1024,
			MaxRequestSize: 100 * 1024 * 1024,
		},
		RAG: RAGSettings{
			Default: DefaultSettings{
				CollectionName: "agentrag_docs",
				Dimension:      1024,
				MetricType:     "COSINE",
			},
			QueryRewrite: QueryRewriteSettings{
				Enabled:            true,
				MaxHistoryMessages: 12,
				MaxHistoryChars:    8000,
			},
			RateLimit: RateLimitSettings{
				Global: GlobalRateLimit{
					Enabled:        true,
					MaxConcurrent:  8,
					MaxWaitSeconds: 15,
					LeaseSeconds:   30,
					PollIntervalMs: 200,
				},
			},
			Memory: MemorySettings{
				HistoryKeepTurns:  12,
				SummaryStartTurns: 8,
				SummaryEnabled:    true,
				TTLMinutes:        1440,
				SummaryMaxChars:   1200,
				TitleMaxLength:    24,
			},
		},
		AI: AISettings{
			Providers: map[string]ProviderConfig{
				"openai": {
					URL: baseURL,
					Endpoints: map[string]string{
						"chat":      "/chat/completions",
						"embedding": "/embeddings",
						"rerank":    "/responses",
					},
				},
				"local": {
					URL: "http://127.0.0.1:11434",
					Endpoints: map[string]string{
						"chat":      "/api/chat",
						"embedding": "/api/embeddings",
					},
				},
			},
			Selection: SelectionSettings{
				FailureThreshold: 3,
				OpenDurationMs:   30000,
			},
			Stream: StreamSettings{
				MessageChunkSize: 32,
			},
			Chat: chatGroup,
			Embedding: ModelGroup{
				DefaultModel: "embedding-openai-large",
				Candidates: []ModelCandidate{
					{
						ID:        "embedding-openai-large",
						Provider:  "openai",
						Model:     "text-embedding-3-large",
						Dimension: 3072,
						Priority:  1,
						Enabled:   true,
					},
					{
						ID:        "embedding-local-bge",
						Provider:  "local",
						Model:     "bge-m3",
						Dimension: 1024,
						Priority:  2,
						Enabled:   true,
					},
				},
			},
			Rerank: ModelGroup{
				DefaultModel: "rerank-local-bge",
				Candidates: []ModelCandidate{
					{
						ID:       "rerank-local-bge",
						Provider: "local",
						Model:    "bge-reranker-v2-m3",
						Priority: 1,
						Enabled:  true,
					},
				},
			},
		},
	}
}

func (s *Service) buildChatModelGroup(baseURL string) ModelGroup {
	defaultModel := strings.TrimSpace(s.ai.Model)
	deepThinkingModel := strings.TrimSpace(s.ai.DeepThinkingModel)
	if defaultModel == "" {
		defaultModel = "gpt-4.1-mini"
	}
	if deepThinkingModel == "" {
		deepThinkingModel = "gpt-5"
	}

	candidates := []ModelCandidate{
		{
			ID:               "chat-openai-" + normalizeModelID(defaultModel),
			Provider:         "openai",
			Model:            defaultModel,
			URL:              baseURL,
			Priority:         1,
			Enabled:          true,
			SupportsThinking: false,
		},
	}
	if deepThinkingModel != "" && deepThinkingModel != defaultModel {
		candidates = append(candidates, ModelCandidate{
			ID:               "chat-openai-" + normalizeModelID(deepThinkingModel),
			Provider:         "openai",
			Model:            deepThinkingModel,
			URL:              baseURL,
			Priority:         2,
			Enabled:          true,
			SupportsThinking: true,
		})
	}
	return ModelGroup{
		DefaultModel:      defaultModel,
		DeepThinkingModel: deepThinkingModel,
		Candidates:        candidates,
	}
}

func normalizeModelID(model string) string {
	value := strings.ToLower(strings.TrimSpace(model))
	replacer := strings.NewReplacer(".", "-", "_", "-", "/", "-", ":", "-")
	value = replacer.Replace(value)
	if value == "" {
		return "default"
	}
	return value
}
