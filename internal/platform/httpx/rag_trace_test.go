package httpx

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	domainconversation "github.com/AmazingCYJ/AgentRAG/internal/domain/conversation"
	appconfig "github.com/AmazingCYJ/AgentRAG/internal/platform/config"
	"github.com/gogf/gf/v2/util/guid"
	_ "modernc.org/sqlite"
)

func TestRagTraceEndpointsExposeChatRunData(t *testing.T) {
	conversationService := domainconversation.NewService(nil)
	server := startChatTestServer(t, conversationService)
	defer server.Shutdown()

	token := loginAndGetToken(t, server.GetListenedPort())
	request, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("http://127.0.0.1:%d/rag/v3/chat?question=%s&deepThinking=true", server.GetListenedPort(), "请给出测试链路"),
		nil,
	)
	if err != nil {
		t.Fatalf("create chat request failed: %v", err)
	}
	request.Header.Set("Authorization", token)

	client := &http.Client{Timeout: 5 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("request chat stream failed: %v", err)
	}
	defer response.Body.Close()

	events := readSSEEventsUntilDone(t, response.Body)
	if len(events) == 0 {
		t.Fatal("expected sse events")
	}

	var meta struct {
		ConversationID string `json:"conversationId"`
		TaskID         string `json:"taskId"`
	}
	if err := json.Unmarshal(events[0].Data, &meta); err != nil {
		t.Fatalf("decode meta failed: %v", err)
	}
	if meta.TaskID == "" {
		t.Fatal("expected task id from chat stream")
	}

	runsRequest, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("http://127.0.0.1:%d/rag/traces/runs?current=1&size=10&taskId=%s", server.GetListenedPort(), meta.TaskID),
		nil,
	)
	if err != nil {
		t.Fatalf("create runs request failed: %v", err)
	}
	runsRequest.Header.Set("Authorization", token)

	runsResponse, err := client.Do(runsRequest)
	if err != nil {
		t.Fatalf("request trace runs failed: %v", err)
	}
	defer runsResponse.Body.Close()

	var runsBody struct {
		Code string `json:"code"`
		Data struct {
			Records []struct {
				TraceID        string `json:"traceId"`
				ConversationID string `json:"conversationId"`
				TaskID         string `json:"taskId"`
				Status         string `json:"status"`
				DurationMs     int64  `json:"durationMs"`
			} `json:"records"`
			Total   int `json:"total"`
			Current int `json:"current"`
			Pages   int `json:"pages"`
		} `json:"data"`
	}
	if err := json.NewDecoder(runsResponse.Body).Decode(&runsBody); err != nil {
		t.Fatalf("decode runs response failed: %v", err)
	}
	if runsBody.Code != "0" {
		t.Fatalf("expected runs code 0, got %s", runsBody.Code)
	}
	if runsBody.Data.Total == 0 || len(runsBody.Data.Records) == 0 {
		t.Fatal("expected at least one trace run")
	}
	run := runsBody.Data.Records[0]
	if run.TaskID != meta.TaskID {
		t.Fatalf("expected task id %s, got %s", meta.TaskID, run.TaskID)
	}
	if run.ConversationID != meta.ConversationID {
		t.Fatalf("expected conversation id %s, got %s", meta.ConversationID, run.ConversationID)
	}
	if run.TraceID == "" {
		t.Fatal("expected trace id")
	}

	detailRequest, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("http://127.0.0.1:%d/rag/traces/runs/%s", server.GetListenedPort(), run.TraceID),
		nil,
	)
	if err != nil {
		t.Fatalf("create detail request failed: %v", err)
	}
	detailRequest.Header.Set("Authorization", token)

	detailResponse, err := client.Do(detailRequest)
	if err != nil {
		t.Fatalf("request trace detail failed: %v", err)
	}
	defer detailResponse.Body.Close()

	var detailBody struct {
		Code string `json:"code"`
		Data struct {
			Run struct {
				TraceID string `json:"traceId"`
				Status  string `json:"status"`
			} `json:"run"`
			Nodes []struct {
				TraceID    string `json:"traceId"`
				NodeID     string `json:"nodeId"`
				NodeName   string `json:"nodeName"`
				Status     string `json:"status"`
				DurationMs int64  `json:"durationMs"`
			} `json:"nodes"`
		} `json:"data"`
	}
	if err := json.NewDecoder(detailResponse.Body).Decode(&detailBody); err != nil {
		t.Fatalf("decode detail response failed: %v", err)
	}
	if detailBody.Code != "0" {
		t.Fatalf("expected detail code 0, got %s", detailBody.Code)
	}
	if detailBody.Data.Run.TraceID != run.TraceID {
		t.Fatalf("expected detail trace id %s, got %s", run.TraceID, detailBody.Data.Run.TraceID)
	}
	if len(detailBody.Data.Nodes) == 0 {
		t.Fatal("expected detail nodes")
	}

	nodesRequest, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("http://127.0.0.1:%d/rag/traces/runs/%s/nodes", server.GetListenedPort(), run.TraceID),
		nil,
	)
	if err != nil {
		t.Fatalf("create nodes request failed: %v", err)
	}
	nodesRequest.Header.Set("Authorization", token)

	nodesResponse, err := client.Do(nodesRequest)
	if err != nil {
		t.Fatalf("request trace nodes failed: %v", err)
	}
	defer nodesResponse.Body.Close()

	var nodesBody struct {
		Code string `json:"code"`
		Data []struct {
			TraceID string `json:"traceId"`
			NodeID  string `json:"nodeId"`
			Status  string `json:"status"`
		} `json:"data"`
	}
	if err := json.NewDecoder(nodesResponse.Body).Decode(&nodesBody); err != nil {
		t.Fatalf("decode nodes response failed: %v", err)
	}
	if nodesBody.Code != "0" {
		t.Fatalf("expected nodes code 0, got %s", nodesBody.Code)
	}
	if len(nodesBody.Data) != len(detailBody.Data.Nodes) {
		t.Fatalf("expected nodes length %d, got %d", len(detailBody.Data.Nodes), len(nodesBody.Data))
	}
}

func TestRagTracePersistsThroughConfiguredDatabase(t *testing.T) {
	dsn := filepath.Join(t.TempDir(), "agentrag.db")
	cfg := &appconfig.Config{
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

	server := newServer(cfg, guid.S())
	server.SetAddr("127.0.0.1:0")
	if err := server.Start(); err != nil {
		t.Fatalf("start server failed: %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	token := loginAndGetToken(t, server.GetListenedPort())
	client := &http.Client{Timeout: 5 * time.Second}
	chatRequest, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("http://127.0.0.1:%d/rag/v3/chat?question=%s", server.GetListenedPort(), url.QueryEscape("测试 Trace 数据库")),
		nil,
	)
	if err != nil {
		t.Fatalf("create chat request failed: %v", err)
	}
	chatRequest.Header.Set("Authorization", token)

	chatResponse, err := client.Do(chatRequest)
	if err != nil {
		t.Fatalf("chat request failed: %v", err)
	}
	if chatResponse.StatusCode != http.StatusOK {
		content, _ := io.ReadAll(chatResponse.Body)
		chatResponse.Body.Close()
		t.Fatalf("expected chat status 200, got %d body %s", chatResponse.StatusCode, string(content))
	}
	if !strings.Contains(chatResponse.Header.Get("Content-Type"), "text/event-stream") {
		content, _ := io.ReadAll(chatResponse.Body)
		chatResponse.Body.Close()
		t.Fatalf("expected sse response, got content-type %s body %s", chatResponse.Header.Get("Content-Type"), string(content))
	}
	events := readSSEEventsUntilDone(t, chatResponse.Body)
	chatResponse.Body.Close()
	if len(events) == 0 {
		t.Fatal("expected chat events")
	}

	var meta struct {
		TaskID string `json:"taskId"`
	}
	if err := json.Unmarshal(events[0].Data, &meta); err != nil {
		t.Fatalf("decode chat meta failed: %v", err)
	}
	if meta.TaskID == "" {
		t.Fatal("expected task id")
	}
	server.Shutdown()

	recreated := newServer(cfg, guid.S())
	recreated.SetAddr("127.0.0.1:0")
	if err := recreated.Start(); err != nil {
		t.Fatalf("restart server failed: %v", err)
	}
	defer recreated.Shutdown()
	time.Sleep(100 * time.Millisecond)

	recreatedToken := loginAndGetToken(t, recreated.GetListenedPort())
	runsRequest, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("http://127.0.0.1:%d/rag/traces/runs?taskId=%s", recreated.GetListenedPort(), meta.TaskID),
		nil,
	)
	if err != nil {
		t.Fatalf("create trace runs request failed: %v", err)
	}
	runsRequest.Header.Set("Authorization", recreatedToken)

	runsResponse, err := client.Do(runsRequest)
	if err != nil {
		t.Fatalf("request trace runs failed: %v", err)
	}
	defer runsResponse.Body.Close()

	var runsBody struct {
		Code string `json:"code"`
		Data struct {
			Records []struct {
				TraceID string `json:"traceId"`
				TaskID  string `json:"taskId"`
			} `json:"records"`
			Total int `json:"total"`
		} `json:"data"`
	}
	if err := json.NewDecoder(runsResponse.Body).Decode(&runsBody); err != nil {
		t.Fatalf("decode trace runs failed: %v", err)
	}
	if runsBody.Code != "0" {
		t.Fatalf("expected trace runs code 0, got %s", runsBody.Code)
	}
	if runsBody.Data.Total == 0 || len(runsBody.Data.Records) == 0 {
		t.Fatal("expected persisted trace run")
	}
	if runsBody.Data.Records[0].TaskID != meta.TaskID {
		t.Fatalf("expected task id %s, got %s", meta.TaskID, runsBody.Data.Records[0].TaskID)
	}
}
