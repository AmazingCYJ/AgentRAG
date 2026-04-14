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

func TestLoginRouteReturnsUserWithToken(t *testing.T) {
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

	payload, _ := json.Marshal(map[string]string{
		"username": "admin",
		"password": "admin123",
	})
	response, err := http.Post(
		fmt.Sprintf("http://127.0.0.1:%d/auth/login", server.GetListenedPort()),
		"application/json",
		bytes.NewReader(payload),
	)
	if err != nil {
		t.Fatalf("login request failed: %v", err)
	}
	defer response.Body.Close()

	var body struct {
		Code string `json:"code"`
		Data struct {
			UserID string `json:"userId"`
			Token  string `json:"token"`
			Role   string `json:"role"`
		} `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode login response failed: %v", err)
	}
	if body.Code != "0" {
		t.Fatalf("expected code 0, got %s", body.Code)
	}
	if body.Data.UserID != "u_admin" {
		t.Fatalf("expected user id u_admin, got %s", body.Data.UserID)
	}
	if body.Data.Token == "" {
		t.Fatal("expected login token")
	}
}

func TestCurrentUserRequiresAuthToken(t *testing.T) {
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

	request, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://127.0.0.1:%d/user/me", server.GetListenedPort()), nil)
	if err != nil {
		t.Fatalf("create request failed: %v", err)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("request current user failed: %v", err)
	}
	defer response.Body.Close()

	var body struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode current user response failed: %v", err)
	}
	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", response.StatusCode)
	}
	if body.Message != "未登录" {
		t.Fatalf("expected 未登录 message, got %s", body.Message)
	}
}

func TestCurrentUserReturnsProfileFromToken(t *testing.T) {
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

	payload, _ := json.Marshal(map[string]string{
		"username": "admin",
		"password": "admin123",
	})
	loginResponse, err := http.Post(
		fmt.Sprintf("http://127.0.0.1:%d/auth/login", server.GetListenedPort()),
		"application/json",
		bytes.NewReader(payload),
	)
	if err != nil {
		t.Fatalf("login request failed: %v", err)
	}
	defer loginResponse.Body.Close()

	var loginBody struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.NewDecoder(loginResponse.Body).Decode(&loginBody); err != nil {
		t.Fatalf("decode login response failed: %v", err)
	}

	request, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://127.0.0.1:%d/user/me", server.GetListenedPort()), nil)
	if err != nil {
		t.Fatalf("create request failed: %v", err)
	}
	request.Header.Set("Authorization", loginBody.Data.Token)

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("request current user failed: %v", err)
	}
	defer response.Body.Close()

	var body struct {
		Code string `json:"code"`
		Data struct {
			UserID   string `json:"userId"`
			Username string `json:"username"`
			Role     string `json:"role"`
		} `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode current user response failed: %v", err)
	}
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.StatusCode)
	}
	if body.Code != "0" {
		t.Fatalf("expected code 0, got %s", body.Code)
	}
	if body.Data.UserID != "u_admin" {
		t.Fatalf("expected user id u_admin, got %s", body.Data.UserID)
	}
	if body.Data.Username != "admin" {
		t.Fatalf("expected username admin, got %s", body.Data.Username)
	}
}
