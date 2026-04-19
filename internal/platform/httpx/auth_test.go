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

func TestCreatedUserCanLogin(t *testing.T) {
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

	adminToken := loginAndGetToken(t, server.GetListenedPort())
	createPayload, _ := json.Marshal(map[string]string{
		"username": "alice",
		"password": "alice123",
		"role":     "user",
	})
	createRequest, err := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("http://127.0.0.1:%d/users", server.GetListenedPort()),
		bytes.NewReader(createPayload),
	)
	if err != nil {
		t.Fatalf("create user request failed: %v", err)
	}
	createRequest.Header.Set("Authorization", adminToken)
	createRequest.Header.Set("Content-Type", "application/json")

	createResponse, err := http.DefaultClient.Do(createRequest)
	if err != nil {
		t.Fatalf("request create user failed: %v", err)
	}
	createResponse.Body.Close()

	loginPayload, _ := json.Marshal(map[string]string{
		"username": "alice",
		"password": "alice123",
	})
	loginResponse, err := http.Post(
		fmt.Sprintf("http://127.0.0.1:%d/auth/login", server.GetListenedPort()),
		"application/json",
		bytes.NewReader(loginPayload),
	)
	if err != nil {
		t.Fatalf("login created user failed: %v", err)
	}
	defer loginResponse.Body.Close()

	var loginBody struct {
		Code string `json:"code"`
		Data struct {
			UserID   string `json:"userId"`
			Username string `json:"username"`
			Token    string `json:"token"`
		} `json:"data"`
	}
	if err := json.NewDecoder(loginResponse.Body).Decode(&loginBody); err != nil {
		t.Fatalf("decode created user login failed: %v", err)
	}
	if loginBody.Code != "0" {
		t.Fatalf("expected created user login code 0, got %s", loginBody.Code)
	}
	if loginBody.Data.Username != "alice" {
		t.Fatalf("expected username alice, got %s", loginBody.Data.Username)
	}
	if loginBody.Data.Token == "" {
		t.Fatal("expected created user login token")
	}
}

func TestChangedPasswordTakesEffectOnLogin(t *testing.T) {
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
	changePayload, _ := json.Marshal(map[string]string{
		"currentPassword": "admin123",
		"newPassword":     "admin456",
	})
	changeRequest, err := http.NewRequest(
		http.MethodPut,
		fmt.Sprintf("http://127.0.0.1:%d/user/password", server.GetListenedPort()),
		bytes.NewReader(changePayload),
	)
	if err != nil {
		t.Fatalf("create change password request failed: %v", err)
	}
	changeRequest.Header.Set("Authorization", token)
	changeRequest.Header.Set("Content-Type", "application/json")

	changeResponse, err := http.DefaultClient.Do(changeRequest)
	if err != nil {
		t.Fatalf("request change password failed: %v", err)
	}
	changeResponse.Body.Close()

	oldLoginPayload, _ := json.Marshal(map[string]string{
		"username": "admin",
		"password": "admin123",
	})
	oldLoginResponse, err := http.Post(
		fmt.Sprintf("http://127.0.0.1:%d/auth/login", server.GetListenedPort()),
		"application/json",
		bytes.NewReader(oldLoginPayload),
	)
	if err != nil {
		t.Fatalf("login with old password failed: %v", err)
	}
	defer oldLoginResponse.Body.Close()

	var oldLoginBody struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(oldLoginResponse.Body).Decode(&oldLoginBody); err != nil {
		t.Fatalf("decode old password login response failed: %v", err)
	}
	if oldLoginBody.Code == "0" {
		t.Fatal("expected old password login to fail")
	}

	newLoginPayload, _ := json.Marshal(map[string]string{
		"username": "admin",
		"password": "admin456",
	})
	newLoginResponse, err := http.Post(
		fmt.Sprintf("http://127.0.0.1:%d/auth/login", server.GetListenedPort()),
		"application/json",
		bytes.NewReader(newLoginPayload),
	)
	if err != nil {
		t.Fatalf("login with new password failed: %v", err)
	}
	defer newLoginResponse.Body.Close()

	var newLoginBody struct {
		Code string `json:"code"`
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.NewDecoder(newLoginResponse.Body).Decode(&newLoginBody); err != nil {
		t.Fatalf("decode new password login response failed: %v", err)
	}
	if newLoginBody.Code != "0" {
		t.Fatalf("expected new password login code 0, got %s", newLoginBody.Code)
	}
	if newLoginBody.Data.Token == "" {
		t.Fatal("expected new password login token")
	}
}
