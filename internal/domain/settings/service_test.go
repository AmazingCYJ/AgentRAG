package settings

import (
	"testing"

	appconfig "github.com/AmazingCYJ/AgentRAG/internal/platform/config"
)

func TestServiceGetReflectsConfiguredAISettings(t *testing.T) {
	service := NewService(appconfig.AIConfig{
		APIKey:            "secret-value",
		BaseURL:           "https://llm.example.com/v1",
		Model:             "qwen-plus",
		DeepThinkingModel: "qwen-max",
	})

	result := service.Get()
	if result.AI.Providers["openai"].URL != "https://llm.example.com/v1" {
		t.Fatalf("expected configured provider url, got %s", result.AI.Providers["openai"].URL)
	}
	if result.AI.Providers["openai"].APIKey != "" {
		t.Fatal("settings must not expose raw api key")
	}
	if result.AI.Chat.DefaultModel != "qwen-plus" {
		t.Fatalf("expected default model qwen-plus, got %s", result.AI.Chat.DefaultModel)
	}
	if result.AI.Chat.DeepThinkingModel != "qwen-max" {
		t.Fatalf("expected deep thinking model qwen-max, got %s", result.AI.Chat.DeepThinkingModel)
	}
	if len(result.AI.Chat.Candidates) != 2 {
		t.Fatalf("expected two chat candidates, got %d", len(result.AI.Chat.Candidates))
	}
	if result.AI.Chat.Candidates[0].URL != "https://llm.example.com/v1" {
		t.Fatalf("expected candidate url from config, got %s", result.AI.Chat.Candidates[0].URL)
	}
}

func TestServiceGetReflectsConfiguredRateLimit(t *testing.T) {
	enabled := true
	service := NewServiceWithConfig(&appconfig.Config{
		RAG: appconfig.RAGConfig{
			RateLimit: appconfig.RAGRateLimitConfig{
				Global: appconfig.GlobalRateLimitConfig{
					Enabled:        &enabled,
					MaxConcurrent:  3,
					MaxWaitSeconds: 5,
					LeaseSeconds:   10,
					PollIntervalMs: 100,
				},
			},
		},
	})

	result := service.Get()
	if !result.RAG.RateLimit.Global.Enabled {
		t.Fatal("expected configured rate limit enabled")
	}
	if result.RAG.RateLimit.Global.MaxConcurrent != 3 {
		t.Fatalf("expected max concurrent 3, got %d", result.RAG.RateLimit.Global.MaxConcurrent)
	}
	if result.RAG.RateLimit.Global.MaxWaitSeconds != 5 {
		t.Fatalf("expected max wait seconds 5, got %d", result.RAG.RateLimit.Global.MaxWaitSeconds)
	}
}
