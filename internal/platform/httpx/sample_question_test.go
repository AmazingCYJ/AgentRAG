package httpx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	appconfig "github.com/AmazingCYJ/AgentRAG/internal/platform/config"
	"github.com/gogf/gf/v2/util/guid"
	_ "modernc.org/sqlite"
)

func TestListWelcomeSampleQuestions(t *testing.T) {
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
	request, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://127.0.0.1:%d/rag/sample-questions", server.GetListenedPort()), nil)
	if err != nil {
		t.Fatalf("create request failed: %v", err)
	}
	request.Header.Set("Authorization", token)

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("request sample questions failed: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.StatusCode)
	}

	var body struct {
		Code string `json:"code"`
		Data []struct {
			ID       string `json:"id"`
			Title    string `json:"title"`
			Question string `json:"question"`
		} `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode welcome sample questions failed: %v", err)
	}
	if body.Code != "0" {
		t.Fatalf("expected code 0, got %s", body.Code)
	}
	if len(body.Data) == 0 {
		t.Fatal("expected seeded sample questions")
	}
}

func TestSampleQuestionCRUDAndPaging(t *testing.T) {
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

	createPayload, _ := json.Marshal(map[string]string{
		"title":       "任务拆解",
		"description": "把目标拆成步骤",
		"question":    "请把项目上线任务拆成可执行步骤",
	})
	createRequest, err := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("http://127.0.0.1:%d/sample-questions", server.GetListenedPort()),
		bytes.NewReader(createPayload),
	)
	if err != nil {
		t.Fatalf("create sample-question request failed: %v", err)
	}
	createRequest.Header.Set("Authorization", token)
	createRequest.Header.Set("Content-Type", "application/json")

	createResponse, err := http.DefaultClient.Do(createRequest)
	if err != nil {
		t.Fatalf("request create sample-question failed: %v", err)
	}
	defer createResponse.Body.Close()

	var createBody struct {
		Code string `json:"code"`
		Data string `json:"data"`
	}
	if err := json.NewDecoder(createResponse.Body).Decode(&createBody); err != nil {
		t.Fatalf("decode create response failed: %v", err)
	}
	if createBody.Code != "0" {
		t.Fatalf("expected create code 0, got %s", createBody.Code)
	}
	if createBody.Data == "" {
		t.Fatal("expected created id")
	}

	pageRequest, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("http://127.0.0.1:%d/sample-questions?current=1&size=10&keyword=%s", server.GetListenedPort(), "任务"),
		nil,
	)
	if err != nil {
		t.Fatalf("create page request failed: %v", err)
	}
	pageRequest.Header.Set("Authorization", token)

	pageResponse, err := http.DefaultClient.Do(pageRequest)
	if err != nil {
		t.Fatalf("request sample-question page failed: %v", err)
	}
	defer pageResponse.Body.Close()

	var pageBody struct {
		Code string `json:"code"`
		Data struct {
			Records []struct {
				ID       string `json:"id"`
				Title    string `json:"title"`
				Question string `json:"question"`
			} `json:"records"`
			Total   int `json:"total"`
			Size    int `json:"size"`
			Current int `json:"current"`
			Pages   int `json:"pages"`
		} `json:"data"`
	}
	if err := json.NewDecoder(pageResponse.Body).Decode(&pageBody); err != nil {
		t.Fatalf("decode page response failed: %v", err)
	}
	if pageBody.Code != "0" {
		t.Fatalf("expected page code 0, got %s", pageBody.Code)
	}
	if pageBody.Data.Total <= 0 || len(pageBody.Data.Records) == 0 {
		t.Fatalf("expected paged records, got total=%d len=%d", pageBody.Data.Total, len(pageBody.Data.Records))
	}

	detailRequest, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("http://127.0.0.1:%d/sample-questions/%s", server.GetListenedPort(), createBody.Data),
		nil,
	)
	if err != nil {
		t.Fatalf("create detail request failed: %v", err)
	}
	detailRequest.Header.Set("Authorization", token)

	detailResponse, err := http.DefaultClient.Do(detailRequest)
	if err != nil {
		t.Fatalf("request sample-question detail failed: %v", err)
	}
	defer detailResponse.Body.Close()

	var detailBody struct {
		Code string `json:"code"`
		Data struct {
			ID          string `json:"id"`
			Title       string `json:"title"`
			Description string `json:"description"`
			Question    string `json:"question"`
		} `json:"data"`
	}
	if err := json.NewDecoder(detailResponse.Body).Decode(&detailBody); err != nil {
		t.Fatalf("decode detail response failed: %v", err)
	}
	if detailBody.Code != "0" {
		t.Fatalf("expected detail code 0, got %s", detailBody.Code)
	}
	if detailBody.Data.ID != createBody.Data {
		t.Fatalf("expected detail id %s, got %s", createBody.Data, detailBody.Data.ID)
	}

	updatePayload, _ := json.Marshal(map[string]string{
		"title":       "任务拆解-更新",
		"description": "更新后的描述",
		"question":    "请输出按周排期的上线步骤",
	})
	updateRequest, err := http.NewRequest(
		http.MethodPut,
		fmt.Sprintf("http://127.0.0.1:%d/sample-questions/%s", server.GetListenedPort(), createBody.Data),
		bytes.NewReader(updatePayload),
	)
	if err != nil {
		t.Fatalf("create update request failed: %v", err)
	}
	updateRequest.Header.Set("Authorization", token)
	updateRequest.Header.Set("Content-Type", "application/json")

	updateResponse, err := http.DefaultClient.Do(updateRequest)
	if err != nil {
		t.Fatalf("request update failed: %v", err)
	}
	defer updateResponse.Body.Close()

	var updateBody struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(updateResponse.Body).Decode(&updateBody); err != nil {
		t.Fatalf("decode update response failed: %v", err)
	}
	if updateBody.Code != "0" {
		t.Fatalf("expected update code 0, got %s", updateBody.Code)
	}

	deleteRequest, err := http.NewRequest(
		http.MethodDelete,
		fmt.Sprintf("http://127.0.0.1:%d/sample-questions/%s", server.GetListenedPort(), createBody.Data),
		nil,
	)
	if err != nil {
		t.Fatalf("create delete request failed: %v", err)
	}
	deleteRequest.Header.Set("Authorization", token)

	deleteResponse, err := http.DefaultClient.Do(deleteRequest)
	if err != nil {
		t.Fatalf("request delete failed: %v", err)
	}
	defer deleteResponse.Body.Close()

	var deleteBody struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(deleteResponse.Body).Decode(&deleteBody); err != nil {
		t.Fatalf("decode delete response failed: %v", err)
	}
	if deleteBody.Code != "0" {
		t.Fatalf("expected delete code 0, got %s", deleteBody.Code)
	}
}

func TestSampleQuestionPersistsThroughConfiguredDatabase(t *testing.T) {
	dsn := filepath.Join(t.TempDir(), "agentrag.db")
	cfg := appconfig.Config{
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
		Database: appconfig.DatabaseConfig{
			Driver: "sqlite",
			DSN:    dsn,
		},
	}
	server := newServer(&cfg, guid.S())
	server.SetAddr("127.0.0.1:0")
	if err := server.Start(); err != nil {
		t.Fatalf("start server failed: %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	token := loginAndGetToken(t, server.GetListenedPort())
	createPayload, _ := json.Marshal(map[string]string{
		"title":    "数据库样例",
		"question": "数据库持久化问题",
	})
	createRequest, err := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("http://127.0.0.1:%d/sample-questions", server.GetListenedPort()),
		bytes.NewReader(createPayload),
	)
	if err != nil {
		t.Fatalf("create sample-question request failed: %v", err)
	}
	createRequest.Header.Set("Authorization", token)
	createRequest.Header.Set("Content-Type", "application/json")
	createResponse, err := http.DefaultClient.Do(createRequest)
	if err != nil {
		t.Fatalf("request create sample-question failed: %v", err)
	}
	var createBody struct {
		Code string `json:"code"`
		Data string `json:"data"`
	}
	if err := json.NewDecoder(createResponse.Body).Decode(&createBody); err != nil {
		createResponse.Body.Close()
		t.Fatalf("decode create response failed: %v", err)
	}
	createResponse.Body.Close()
	if createBody.Code != "0" || createBody.Data == "" {
		t.Fatalf("unexpected create response %#v", createBody)
	}
	server.Shutdown()

	recreated := newServer(&cfg, guid.S())
	recreated.SetAddr("127.0.0.1:0")
	if err := recreated.Start(); err != nil {
		t.Fatalf("start recreated server failed: %v", err)
	}
	defer recreated.Shutdown()
	time.Sleep(100 * time.Millisecond)
	recreatedToken := loginAndGetToken(t, recreated.GetListenedPort())
	detailRequest, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("http://127.0.0.1:%d/sample-questions/%s", recreated.GetListenedPort(), createBody.Data),
		nil,
	)
	if err != nil {
		t.Fatalf("create detail request failed: %v", err)
	}
	detailRequest.Header.Set("Authorization", recreatedToken)
	detailResponse, err := http.DefaultClient.Do(detailRequest)
	if err != nil {
		t.Fatalf("request recreated detail failed: %v", err)
	}
	defer detailResponse.Body.Close()

	var detailBody struct {
		Code string `json:"code"`
		Data struct {
			ID       string `json:"id"`
			Question string `json:"question"`
		} `json:"data"`
	}
	if err := json.NewDecoder(detailResponse.Body).Decode(&detailBody); err != nil {
		t.Fatalf("decode recreated detail failed: %v", err)
	}
	if detailBody.Code != "0" || detailBody.Data.ID != createBody.Data {
		t.Fatalf("unexpected recreated detail %#v", detailBody)
	}
	if detailBody.Data.Question != "数据库持久化问题" {
		t.Fatalf("expected persisted question, got %s", detailBody.Data.Question)
	}
}
