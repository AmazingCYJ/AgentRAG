package httpx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	appconfig "github.com/AmazingCYJ/AgentRAG/internal/platform/config"
	"github.com/gogf/gf/v2/util/guid"
)

func TestMCPRouteListsToolsFromMainAPIServer(t *testing.T) {
	server := newServer(&appconfig.Config{
		HTTP: appconfig.HTTPConfig{Port: 8080},
	}, guid.S())
	server.SetAddr("127.0.0.1:0")
	if err := server.Start(); err != nil {
		t.Fatalf("start server failed: %v", err)
	}
	defer server.Shutdown()
	time.Sleep(100 * time.Millisecond)

	payload, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/list",
		"params":  map[string]any{},
	})
	response, err := http.Post(
		fmt.Sprintf("http://127.0.0.1:%d/mcp", server.GetListenedPort()),
		"application/json",
		bytes.NewReader(payload),
	)
	if err != nil {
		t.Fatalf("post mcp tools/list failed: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected mcp status 200, got %d", response.StatusCode)
	}
	var body struct {
		JSONRPC string `json:"jsonrpc"`
		Result  struct {
			Tools []struct {
				Name string `json:"name"`
			} `json:"tools"`
		} `json:"result"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode mcp response failed: %v", err)
	}
	if body.JSONRPC != "2.0" {
		t.Fatalf("expected jsonrpc 2.0, got %s", body.JSONRPC)
	}
	if len(body.Result.Tools) < 3 {
		t.Fatalf("expected built-in tools, got %#v", body.Result.Tools)
	}
}

func TestMCPPrefixedRouteCallsToolFromMainAPIServer(t *testing.T) {
	server := newServer(&appconfig.Config{
		HTTP: appconfig.HTTPConfig{Port: 8080},
	}, guid.S())
	server.SetAddr("127.0.0.1:0")
	if err := server.Start(); err != nil {
		t.Fatalf("start server failed: %v", err)
	}
	defer server.Shutdown()
	time.Sleep(100 * time.Millisecond)

	payload, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "weather_query",
			"arguments": map[string]any{
				"city": "上海",
			},
		},
	})
	response, err := http.Post(
		fmt.Sprintf("http://127.0.0.1:%d/api/ragent/mcp", server.GetListenedPort()),
		"application/json",
		bytes.NewReader(payload),
	)
	if err != nil {
		t.Fatalf("post prefixed mcp tools/call failed: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected prefixed mcp status 200, got %d", response.StatusCode)
	}
	var body struct {
		Result struct {
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
			IsError bool `json:"isError"`
		} `json:"result"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode prefixed mcp response failed: %v", err)
	}
	if body.Result.IsError {
		t.Fatal("expected mcp weather tool to succeed")
	}
	if len(body.Result.Content) == 0 || body.Result.Content[0].Text == "" {
		t.Fatalf("expected mcp tool content, got %#v", body.Result.Content)
	}
}
