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

func TestHealthRouteReturnsUnifiedPayload(t *testing.T) {
	server := newServer(&appconfig.Config{
		HTTP: appconfig.HTTPConfig{Port: 8080},
	}, guid.S())
	server.SetAddr("127.0.0.1:0")
	if err := server.Start(); err != nil {
		t.Fatalf("start server failed: %v", err)
	}
	defer server.Shutdown()
	time.Sleep(100 * time.Millisecond)

	response, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/health", server.GetListenedPort()))
	if err != nil {
		t.Fatalf("request health endpoint failed: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.StatusCode)
	}

	var body struct {
		Code    string         `json:"code"`
		Message string         `json:"message"`
		Data    map[string]any `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("unmarshal response failed: %v", err)
	}
	if body.Code != "0" {
		t.Fatalf("expected code 0, got %s", body.Code)
	}
	if body.Message != "success" {
		t.Fatalf("expected success message, got %s", body.Message)
	}
	if body.Data["status"] != "ok" {
		t.Fatalf("expected health status ok, got %#v", body.Data["status"])
	}
}

func TestAPIPrefixHealthRouteReturnsUnifiedPayload(t *testing.T) {
	server := newServer(&appconfig.Config{
		HTTP: appconfig.HTTPConfig{Port: 8080},
	}, guid.S())
	server.SetAddr("127.0.0.1:0")
	if err := server.Start(); err != nil {
		t.Fatalf("start server failed: %v", err)
	}
	defer server.Shutdown()
	time.Sleep(100 * time.Millisecond)

	response, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/api/ragent/health", server.GetListenedPort()))
	if err != nil {
		t.Fatalf("request prefixed health endpoint failed: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.StatusCode)
	}

	var body struct {
		Code    string         `json:"code"`
		Message string         `json:"message"`
		Data    map[string]any `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("unmarshal response failed: %v", err)
	}
	if body.Code != "0" {
		t.Fatalf("expected code 0, got %s", body.Code)
	}
	if body.Data["status"] != "ok" {
		t.Fatalf("expected health status ok, got %#v", body.Data["status"])
	}
}

func TestAPIPrefixPreflightReturnsCORSHeaders(t *testing.T) {
	server := newServer(&appconfig.Config{
		HTTP: appconfig.HTTPConfig{Port: 8080},
	}, guid.S())
	server.SetAddr("127.0.0.1:0")
	if err := server.Start(); err != nil {
		t.Fatalf("start server failed: %v", err)
	}
	defer server.Shutdown()
	time.Sleep(100 * time.Millisecond)

	request, err := http.NewRequest(http.MethodOptions, fmt.Sprintf("http://127.0.0.1:%d/api/ragent/auth/login", server.GetListenedPort()), nil)
	if err != nil {
		t.Fatalf("create preflight request failed: %v", err)
	}
	request.Header.Set("Origin", "http://localhost:5173")
	request.Header.Set("Access-Control-Request-Method", "POST")
	request.Header.Set("Access-Control-Request-Headers", "authorization,content-type")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("request preflight failed: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected preflight status 200, got %d", response.StatusCode)
	}
	if response.Header.Get("Access-Control-Allow-Origin") == "" {
		t.Fatal("expected Access-Control-Allow-Origin header")
	}
	if response.Header.Get("Access-Control-Allow-Headers") == "" {
		t.Fatal("expected Access-Control-Allow-Headers header")
	}
}
