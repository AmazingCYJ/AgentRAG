package httpx

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	appconfig "github.com/AmazingCYJ/AgentRAG/internal/platform/config"
	"github.com/gogf/gf/v2/util/guid"
)

func TestGetSystemSettingsReturnsExpectedShape(t *testing.T) {
	server := newServer(&appconfig.Config{
		HTTP: appconfig.HTTPConfig{Port: 8080},
		Auth: appconfig.AuthConfig{
			JWTSecret: "test-secret",
			TokenTTL:  time.Hour,
			Bootstrap: appconfig.BootstrapUserConfig{
				UserID:   "u_admin",
				Username: "admin",
				Password: "admin123",
				Role:     "admin",
			},
		},
	}, guid.S())
	server.SetAddr("127.0.0.1:0")
	if err := server.Start(); err != nil {
		t.Fatalf("start server failed: %v", err)
	}
	defer server.Shutdown()
	time.Sleep(100 * time.Millisecond)

	token := loginAndGetToken(t, server.GetListenedPort())
	request, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://127.0.0.1:%d/rag/settings", server.GetListenedPort()), nil)
	if err != nil {
		t.Fatalf("create settings request failed: %v", err)
	}
	request.Header.Set("Authorization", token)

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("request settings failed: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.StatusCode)
	}

	var body struct {
		Code string `json:"code"`
		Data struct {
			Upload struct {
				MaxFileSize    int64 `json:"maxFileSize"`
				MaxRequestSize int64 `json:"maxRequestSize"`
			} `json:"upload"`
			RAG struct {
				Default struct {
					CollectionName string `json:"collectionName"`
					Dimension      int    `json:"dimension"`
					MetricType     string `json:"metricType"`
				} `json:"default"`
				Memory struct {
					TitleMaxLength int `json:"titleMaxLength"`
				} `json:"memory"`
			} `json:"rag"`
			AI struct {
				Providers map[string]struct {
					URL       string            `json:"url"`
					Endpoints map[string]string `json:"endpoints"`
				} `json:"providers"`
				Embedding struct {
					Candidates []struct {
						ID       string `json:"id"`
						Provider string `json:"provider"`
						Model    string `json:"model"`
					} `json:"candidates"`
				} `json:"embedding"`
			} `json:"ai"`
		} `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode settings response failed: %v", err)
	}
	if body.Code != "0" {
		t.Fatalf("expected code 0, got %s", body.Code)
	}
	if body.Data.Upload.MaxFileSize <= 0 {
		t.Fatalf("expected positive max file size, got %d", body.Data.Upload.MaxFileSize)
	}
	if body.Data.RAG.Default.CollectionName == "" {
		t.Fatal("expected rag default collection name")
	}
	if body.Data.RAG.Default.Dimension <= 0 {
		t.Fatalf("expected positive dimension, got %d", body.Data.RAG.Default.Dimension)
	}
	if body.Data.RAG.Memory.TitleMaxLength <= 0 {
		t.Fatalf("expected positive title max length, got %d", body.Data.RAG.Memory.TitleMaxLength)
	}
	if len(body.Data.AI.Providers) == 0 {
		t.Fatal("expected at least one provider")
	}
	if len(body.Data.AI.Embedding.Candidates) == 0 {
		t.Fatal("expected at least one embedding candidate")
	}
}

func TestGetSystemSettingsReflectsConfiguredAIModels(t *testing.T) {
	server := newServer(&appconfig.Config{
		HTTP: appconfig.HTTPConfig{Port: 8080},
		Auth: appconfig.AuthConfig{
			JWTSecret: "test-secret",
			TokenTTL:  time.Hour,
			Bootstrap: appconfig.BootstrapUserConfig{
				UserID:   "u_admin",
				Username: "admin",
				Password: "admin123",
				Role:     "admin",
			},
		},
		AI: appconfig.AIConfig{
			APIKey:            "secret-value",
			BaseURL:           "https://llm.example.com/v1",
			Model:             "qwen-plus",
			DeepThinkingModel: "qwen-max",
		},
	}, guid.S())
	server.SetAddr("127.0.0.1:0")
	if err := server.Start(); err != nil {
		t.Fatalf("start server failed: %v", err)
	}
	defer server.Shutdown()
	time.Sleep(100 * time.Millisecond)

	token := loginAndGetToken(t, server.GetListenedPort())
	request, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://127.0.0.1:%d/rag/settings", server.GetListenedPort()), nil)
	if err != nil {
		t.Fatalf("create settings request failed: %v", err)
	}
	request.Header.Set("Authorization", token)

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("request settings failed: %v", err)
	}
	defer response.Body.Close()

	var body struct {
		Code string `json:"code"`
		Data struct {
			AI struct {
				Providers map[string]struct {
					URL    string `json:"url"`
					APIKey string `json:"apiKey"`
				} `json:"providers"`
				Chat struct {
					DefaultModel      string `json:"defaultModel"`
					DeepThinkingModel string `json:"deepThinkingModel"`
					Candidates        []struct {
						ID      string `json:"id"`
						Model   string `json:"model"`
						URL     string `json:"url"`
						Enabled bool   `json:"enabled"`
					} `json:"candidates"`
				} `json:"chat"`
			} `json:"ai"`
		} `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode settings response failed: %v", err)
	}
	if body.Code != "0" {
		t.Fatalf("expected code 0, got %s", body.Code)
	}
	if body.Data.AI.Chat.DefaultModel != "qwen-plus" {
		t.Fatalf("expected configured chat model qwen-plus, got %s", body.Data.AI.Chat.DefaultModel)
	}
	if body.Data.AI.Chat.DeepThinkingModel != "qwen-max" {
		t.Fatalf("expected configured deep thinking model qwen-max, got %s", body.Data.AI.Chat.DeepThinkingModel)
	}
	if body.Data.AI.Providers["openai"].URL != "https://llm.example.com/v1" {
		t.Fatalf("expected provider url from config, got %s", body.Data.AI.Providers["openai"].URL)
	}
	if body.Data.AI.Providers["openai"].APIKey != "" {
		t.Fatal("settings endpoint must not expose raw api key")
	}
	if len(body.Data.AI.Chat.Candidates) == 0 || body.Data.AI.Chat.Candidates[0].Model != "qwen-plus" {
		t.Fatalf("expected configured model candidate, got %#v", body.Data.AI.Chat.Candidates)
	}
}
