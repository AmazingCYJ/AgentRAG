package mcpserver

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestInitializeReturnsProtocolAndServerInfo(t *testing.T) {
	server := httptest.NewServer(NewHTTPHandler())
	defer server.Close()

	payload, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params":  map[string]any{},
	})
	response, err := http.Post(server.URL+"/mcp", "application/json", bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("post initialize failed: %v", err)
	}
	defer response.Body.Close()

	var body struct {
		JSONRPC string `json:"jsonrpc"`
		ID      int    `json:"id"`
		Result  struct {
			ProtocolVersion string `json:"protocolVersion"`
			ServerInfo      struct {
				Name    string `json:"name"`
				Version string `json:"version"`
			} `json:"serverInfo"`
			Capabilities map[string]any `json:"capabilities"`
		} `json:"result"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode initialize response failed: %v", err)
	}
	if body.JSONRPC != "2.0" {
		t.Fatalf("expected jsonrpc 2.0, got %s", body.JSONRPC)
	}
	if body.Result.ProtocolVersion == "" {
		t.Fatal("expected protocol version")
	}
	if body.Result.ServerInfo.Name != "ragent-mcp-server" {
		t.Fatalf("expected server name ragent-mcp-server, got %s", body.Result.ServerInfo.Name)
	}
}

func TestToolsListReturnsBuiltInTools(t *testing.T) {
	server := httptest.NewServer(NewHTTPHandler())
	defer server.Close()

	payload, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/list",
		"params":  map[string]any{},
	})
	response, err := http.Post(server.URL+"/mcp", "application/json", bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("post tools/list failed: %v", err)
	}
	defer response.Body.Close()

	var body struct {
		Result struct {
			Tools []struct {
				Name        string `json:"name"`
				Description string `json:"description"`
			} `json:"tools"`
		} `json:"result"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode tools/list response failed: %v", err)
	}
	if len(body.Result.Tools) < 3 {
		t.Fatalf("expected at least 3 tools, got %d", len(body.Result.Tools))
	}
	names := map[string]bool{}
	for _, tool := range body.Result.Tools {
		names[tool.Name] = true
	}
	for _, expected := range []string{"weather_query", "ticket_query", "sales_query"} {
		if !names[expected] {
			t.Fatalf("expected tool %s in list", expected)
		}
	}
}

func TestToolsCallWeatherReturnsContent(t *testing.T) {
	server := httptest.NewServer(NewHTTPHandler())
	defer server.Close()

	payload, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      3,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "weather_query",
			"arguments": map[string]any{
				"city":      "北京",
				"queryType": "current",
			},
		},
	})
	response, err := http.Post(server.URL+"/mcp", "application/json", bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("post tools/call failed: %v", err)
	}
	defer response.Body.Close()

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
		t.Fatalf("decode tools/call response failed: %v", err)
	}
	if body.Result.IsError {
		t.Fatal("expected weather tool call to succeed")
	}
	if len(body.Result.Content) == 0 {
		t.Fatal("expected content from weather tool")
	}
	if body.Result.Content[0].Type != "text" {
		t.Fatalf("expected text content type, got %s", body.Result.Content[0].Type)
	}
	if body.Result.Content[0].Text == "" {
		t.Fatal("expected non-empty weather response text")
	}
}
