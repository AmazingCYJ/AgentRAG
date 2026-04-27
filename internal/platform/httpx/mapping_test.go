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

func TestQueryTermMappingCRUDAndPaging(t *testing.T) {
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

	createPayload, _ := json.Marshal(map[string]any{
		"sourceTerm": "oa系统",
		"targetTerm": "OA",
		"matchType":  1,
		"priority":   10,
		"enabled":    true,
		"remark":     "办公系统关键词归一化",
	})
	createRequest, err := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("http://127.0.0.1:%d/mappings", server.GetListenedPort()),
		bytes.NewReader(createPayload),
	)
	if err != nil {
		t.Fatalf("create mapping request failed: %v", err)
	}
	createRequest.Header.Set("Authorization", token)
	createRequest.Header.Set("Content-Type", "application/json")

	createResponse, err := http.DefaultClient.Do(createRequest)
	if err != nil {
		t.Fatalf("request create mapping failed: %v", err)
	}
	defer createResponse.Body.Close()

	var createBody struct {
		Code string `json:"code"`
		Data string `json:"data"`
	}
	if err := json.NewDecoder(createResponse.Body).Decode(&createBody); err != nil {
		t.Fatalf("decode create mapping response failed: %v", err)
	}
	if createBody.Code != "0" {
		t.Fatalf("expected create code 0, got %s", createBody.Code)
	}
	if createBody.Data == "" {
		t.Fatal("expected created mapping id")
	}

	pageRequest, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("http://127.0.0.1:%d/mappings?current=1&size=10&keyword=oa", server.GetListenedPort()),
		nil,
	)
	if err != nil {
		t.Fatalf("create page request failed: %v", err)
	}
	pageRequest.Header.Set("Authorization", token)

	pageResponse, err := http.DefaultClient.Do(pageRequest)
	if err != nil {
		t.Fatalf("request mappings page failed: %v", err)
	}
	defer pageResponse.Body.Close()

	var pageBody struct {
		Code string `json:"code"`
		Data struct {
			Records []struct {
				ID         string `json:"id"`
				SourceTerm string `json:"sourceTerm"`
				TargetTerm string `json:"targetTerm"`
				Enabled    bool   `json:"enabled"`
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
		t.Fatalf("expected mapping page records, got total=%d len=%d", pageBody.Data.Total, len(pageBody.Data.Records))
	}

	detailRequest, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("http://127.0.0.1:%d/mappings/%s", server.GetListenedPort(), createBody.Data),
		nil,
	)
	if err != nil {
		t.Fatalf("create detail request failed: %v", err)
	}
	detailRequest.Header.Set("Authorization", token)

	detailResponse, err := http.DefaultClient.Do(detailRequest)
	if err != nil {
		t.Fatalf("request mapping detail failed: %v", err)
	}
	defer detailResponse.Body.Close()

	var detailBody struct {
		Code string `json:"code"`
		Data struct {
			ID         string `json:"id"`
			SourceTerm string `json:"sourceTerm"`
			TargetTerm string `json:"targetTerm"`
			MatchType  int    `json:"matchType"`
			Priority   int    `json:"priority"`
			Enabled    bool   `json:"enabled"`
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

	updatePayload, _ := json.Marshal(map[string]any{
		"sourceTerm": "oa平台",
		"targetTerm": "OA",
		"matchType":  2,
		"priority":   5,
		"enabled":    false,
		"remark":     "更新备注",
	})
	updateRequest, err := http.NewRequest(
		http.MethodPut,
		fmt.Sprintf("http://127.0.0.1:%d/mappings/%s", server.GetListenedPort(), createBody.Data),
		bytes.NewReader(updatePayload),
	)
	if err != nil {
		t.Fatalf("create update request failed: %v", err)
	}
	updateRequest.Header.Set("Authorization", token)
	updateRequest.Header.Set("Content-Type", "application/json")

	updateResponse, err := http.DefaultClient.Do(updateRequest)
	if err != nil {
		t.Fatalf("request mapping update failed: %v", err)
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
		fmt.Sprintf("http://127.0.0.1:%d/mappings/%s", server.GetListenedPort(), createBody.Data),
		nil,
	)
	if err != nil {
		t.Fatalf("create delete request failed: %v", err)
	}
	deleteRequest.Header.Set("Authorization", token)

	deleteResponse, err := http.DefaultClient.Do(deleteRequest)
	if err != nil {
		t.Fatalf("request mapping delete failed: %v", err)
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

func TestQueryTermMappingPersistsThroughConfiguredDatabase(t *testing.T) {
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
	createPayload, _ := json.Marshal(map[string]any{
		"sourceTerm": "财务系统",
		"targetTerm": "Finance",
		"matchType":  1,
		"priority":   3,
		"enabled":    true,
		"remark":     "数据库持久化",
	})
	createRequest, err := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("http://127.0.0.1:%d/mappings", server.GetListenedPort()),
		bytes.NewReader(createPayload),
	)
	if err != nil {
		t.Fatalf("create mapping request failed: %v", err)
	}
	createRequest.Header.Set("Authorization", token)
	createRequest.Header.Set("Content-Type", "application/json")
	createResponse, err := http.DefaultClient.Do(createRequest)
	if err != nil {
		t.Fatalf("request create mapping failed: %v", err)
	}
	var createBody struct {
		Code string `json:"code"`
		Data string `json:"data"`
	}
	if err := json.NewDecoder(createResponse.Body).Decode(&createBody); err != nil {
		createResponse.Body.Close()
		t.Fatalf("decode create mapping failed: %v", err)
	}
	createResponse.Body.Close()
	if createBody.Code != "0" || createBody.Data == "" {
		t.Fatalf("unexpected create mapping response %#v", createBody)
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
		fmt.Sprintf("http://127.0.0.1:%d/mappings/%s", recreated.GetListenedPort(), createBody.Data),
		nil,
	)
	if err != nil {
		t.Fatalf("create detail request failed: %v", err)
	}
	detailRequest.Header.Set("Authorization", recreatedToken)
	detailResponse, err := http.DefaultClient.Do(detailRequest)
	if err != nil {
		t.Fatalf("request recreated mapping detail failed: %v", err)
	}
	defer detailResponse.Body.Close()

	var detailBody struct {
		Code string `json:"code"`
		Data struct {
			ID         string `json:"id"`
			SourceTerm string `json:"sourceTerm"`
			TargetTerm string `json:"targetTerm"`
		} `json:"data"`
	}
	if err := json.NewDecoder(detailResponse.Body).Decode(&detailBody); err != nil {
		t.Fatalf("decode recreated mapping detail failed: %v", err)
	}
	if detailBody.Code != "0" || detailBody.Data.ID != createBody.Data {
		t.Fatalf("unexpected recreated mapping detail %#v", detailBody)
	}
	if detailBody.Data.SourceTerm != "财务系统" || detailBody.Data.TargetTerm != "Finance" {
		t.Fatalf("unexpected persisted mapping %#v", detailBody.Data)
	}
}
