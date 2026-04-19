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

func TestUserCRUDAndPasswordChange(t *testing.T) {
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
				Avatar:   "",
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

	createPayload, _ := json.Marshal(map[string]any{
		"username": "alice",
		"password": "alice123",
		"role":     "user",
		"avatar":   "https://example.com/avatar.png",
	})
	createRequest, err := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("http://127.0.0.1:%d/users", server.GetListenedPort()),
		bytes.NewReader(createPayload),
	)
	if err != nil {
		t.Fatalf("create user request failed: %v", err)
	}
	createRequest.Header.Set("Authorization", token)
	createRequest.Header.Set("Content-Type", "application/json")

	createResponse, err := http.DefaultClient.Do(createRequest)
	if err != nil {
		t.Fatalf("request create user failed: %v", err)
	}
	defer createResponse.Body.Close()

	var createBody struct {
		Code string `json:"code"`
		Data string `json:"data"`
	}
	if err := json.NewDecoder(createResponse.Body).Decode(&createBody); err != nil {
		t.Fatalf("decode create user response failed: %v", err)
	}
	if createBody.Code != "0" {
		t.Fatalf("expected create user code 0, got %s", createBody.Code)
	}
	if createBody.Data == "" {
		t.Fatal("expected created user id")
	}

	pageRequest, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("http://127.0.0.1:%d/users?current=1&size=10&keyword=%s", server.GetListenedPort(), "alice"),
		nil,
	)
	if err != nil {
		t.Fatalf("create users page request failed: %v", err)
	}
	pageRequest.Header.Set("Authorization", token)

	pageResponse, err := http.DefaultClient.Do(pageRequest)
	if err != nil {
		t.Fatalf("request users page failed: %v", err)
	}
	defer pageResponse.Body.Close()

	var pageBody struct {
		Code string `json:"code"`
		Data struct {
			Records []struct {
				ID       string `json:"id"`
				Username string `json:"username"`
				Role     string `json:"role"`
				Avatar   string `json:"avatar"`
			} `json:"records"`
			Total int `json:"total"`
		} `json:"data"`
	}
	if err := json.NewDecoder(pageResponse.Body).Decode(&pageBody); err != nil {
		t.Fatalf("decode users page response failed: %v", err)
	}
	if pageBody.Code != "0" {
		t.Fatalf("expected users page code 0, got %s", pageBody.Code)
	}
	if pageBody.Data.Total == 0 || len(pageBody.Data.Records) == 0 {
		t.Fatal("expected users page records")
	}

	updatePayload, _ := json.Marshal(map[string]any{
		"username": "alice-updated",
		"role":     "admin",
		"avatar":   "https://example.com/avatar-2.png",
	})
	updateRequest, err := http.NewRequest(
		http.MethodPut,
		fmt.Sprintf("http://127.0.0.1:%d/users/%s", server.GetListenedPort(), createBody.Data),
		bytes.NewReader(updatePayload),
	)
	if err != nil {
		t.Fatalf("create update user request failed: %v", err)
	}
	updateRequest.Header.Set("Authorization", token)
	updateRequest.Header.Set("Content-Type", "application/json")

	updateResponse, err := http.DefaultClient.Do(updateRequest)
	if err != nil {
		t.Fatalf("request update user failed: %v", err)
	}
	defer updateResponse.Body.Close()

	var updateBody struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(updateResponse.Body).Decode(&updateBody); err != nil {
		t.Fatalf("decode update user response failed: %v", err)
	}
	if updateBody.Code != "0" {
		t.Fatalf("expected update user code 0, got %s", updateBody.Code)
	}

	changePasswordPayload, _ := json.Marshal(map[string]any{
		"currentPassword": "admin123",
		"newPassword":     "admin456",
	})
	changePasswordRequest, err := http.NewRequest(
		http.MethodPut,
		fmt.Sprintf("http://127.0.0.1:%d/user/password", server.GetListenedPort()),
		bytes.NewReader(changePasswordPayload),
	)
	if err != nil {
		t.Fatalf("create change password request failed: %v", err)
	}
	changePasswordRequest.Header.Set("Authorization", token)
	changePasswordRequest.Header.Set("Content-Type", "application/json")

	changePasswordResponse, err := http.DefaultClient.Do(changePasswordRequest)
	if err != nil {
		t.Fatalf("request change password failed: %v", err)
	}
	defer changePasswordResponse.Body.Close()

	var changePasswordBody struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(changePasswordResponse.Body).Decode(&changePasswordBody); err != nil {
		t.Fatalf("decode change password response failed: %v", err)
	}
	if changePasswordBody.Code != "0" {
		t.Fatalf("expected change password code 0, got %s", changePasswordBody.Code)
	}

	deleteRequest, err := http.NewRequest(
		http.MethodDelete,
		fmt.Sprintf("http://127.0.0.1:%d/users/%s", server.GetListenedPort(), createBody.Data),
		nil,
	)
	if err != nil {
		t.Fatalf("create delete user request failed: %v", err)
	}
	deleteRequest.Header.Set("Authorization", token)

	deleteResponse, err := http.DefaultClient.Do(deleteRequest)
	if err != nil {
		t.Fatalf("request delete user failed: %v", err)
	}
	defer deleteResponse.Body.Close()

	var deleteBody struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(deleteResponse.Body).Decode(&deleteBody); err != nil {
		t.Fatalf("decode delete user response failed: %v", err)
	}
	if deleteBody.Code != "0" {
		t.Fatalf("expected delete user code 0, got %s", deleteBody.Code)
	}
}

func TestDashboardEndpointsReturnExpectedShapes(t *testing.T) {
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
				Avatar:   "",
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

	overviewRequest, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("http://127.0.0.1:%d/admin/dashboard/overview?window=24h", server.GetListenedPort()),
		nil,
	)
	if err != nil {
		t.Fatalf("create overview request failed: %v", err)
	}
	overviewRequest.Header.Set("Authorization", token)

	overviewResponse, err := http.DefaultClient.Do(overviewRequest)
	if err != nil {
		t.Fatalf("request overview failed: %v", err)
	}
	defer overviewResponse.Body.Close()

	var overviewBody struct {
		Code string `json:"code"`
		Data struct {
			Window        string `json:"window"`
			CompareWindow string `json:"compareWindow"`
			UpdatedAt     int64  `json:"updatedAt"`
			KPIs          struct {
				TotalUsers struct {
					Value int `json:"value"`
				} `json:"totalUsers"`
				ActiveUsers struct {
					Value int `json:"value"`
				} `json:"activeUsers"`
				TotalSessions struct {
					Value int `json:"value"`
				} `json:"totalSessions"`
				Sessions24h struct {
					Value int `json:"value"`
				} `json:"sessions24h"`
				TotalMessages struct {
					Value int `json:"value"`
				} `json:"totalMessages"`
				Messages24h struct {
					Value int `json:"value"`
				} `json:"messages24h"`
			} `json:"kpis"`
		} `json:"data"`
	}
	if err := json.NewDecoder(overviewResponse.Body).Decode(&overviewBody); err != nil {
		t.Fatalf("decode overview response failed: %v", err)
	}
	if overviewBody.Code != "0" {
		t.Fatalf("expected overview code 0, got %s", overviewBody.Code)
	}
	if overviewBody.Data.Window == "" || overviewBody.Data.UpdatedAt == 0 {
		t.Fatal("expected overview window and updatedAt")
	}

	perfRequest, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("http://127.0.0.1:%d/admin/dashboard/performance?window=24h", server.GetListenedPort()),
		nil,
	)
	if err != nil {
		t.Fatalf("create performance request failed: %v", err)
	}
	perfRequest.Header.Set("Authorization", token)

	perfResponse, err := http.DefaultClient.Do(perfRequest)
	if err != nil {
		t.Fatalf("request performance failed: %v", err)
	}
	defer perfResponse.Body.Close()

	var perfBody struct {
		Code string `json:"code"`
		Data struct {
			Window       string  `json:"window"`
			AvgLatencyMs int64   `json:"avgLatencyMs"`
			P95LatencyMs int64   `json:"p95LatencyMs"`
			SuccessRate  float64 `json:"successRate"`
			ErrorRate    float64 `json:"errorRate"`
			NoDocRate    float64 `json:"noDocRate"`
			SlowRate     float64 `json:"slowRate"`
		} `json:"data"`
	}
	if err := json.NewDecoder(perfResponse.Body).Decode(&perfBody); err != nil {
		t.Fatalf("decode performance response failed: %v", err)
	}
	if perfBody.Code != "0" {
		t.Fatalf("expected performance code 0, got %s", perfBody.Code)
	}
	if perfBody.Data.Window == "" {
		t.Fatal("expected performance window")
	}

	trendRequest, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("http://127.0.0.1:%d/admin/dashboard/trends?metric=sessions&window=7d&granularity=day", server.GetListenedPort()),
		nil,
	)
	if err != nil {
		t.Fatalf("create trends request failed: %v", err)
	}
	trendRequest.Header.Set("Authorization", token)

	trendResponse, err := http.DefaultClient.Do(trendRequest)
	if err != nil {
		t.Fatalf("request trends failed: %v", err)
	}
	defer trendResponse.Body.Close()

	var trendBody struct {
		Code string `json:"code"`
		Data struct {
			Metric      string `json:"metric"`
			Window      string `json:"window"`
			Granularity string `json:"granularity"`
			Series      []struct {
				Name string `json:"name"`
				Data []struct {
					TS    int64 `json:"ts"`
					Value int   `json:"value"`
				} `json:"data"`
			} `json:"series"`
		} `json:"data"`
	}
	if err := json.NewDecoder(trendResponse.Body).Decode(&trendBody); err != nil {
		t.Fatalf("decode trends response failed: %v", err)
	}
	if trendBody.Code != "0" {
		t.Fatalf("expected trends code 0, got %s", trendBody.Code)
	}
	if trendBody.Data.Metric != "sessions" {
		t.Fatalf("expected metric sessions, got %s", trendBody.Data.Metric)
	}
	if len(trendBody.Data.Series) == 0 || len(trendBody.Data.Series[0].Data) == 0 {
		t.Fatal("expected trend series data")
	}
}
