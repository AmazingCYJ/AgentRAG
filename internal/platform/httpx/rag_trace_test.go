package httpx

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	domainconversation "github.com/AmazingCYJ/AgentRAG/internal/domain/conversation"
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
