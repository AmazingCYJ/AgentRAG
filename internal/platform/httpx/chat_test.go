package httpx

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	domainchat "github.com/AmazingCYJ/AgentRAG/internal/domain/chat"
	domainconversation "github.com/AmazingCYJ/AgentRAG/internal/domain/conversation"
	domainintenttree "github.com/AmazingCYJ/AgentRAG/internal/domain/intenttree"
	domainknowledge "github.com/AmazingCYJ/AgentRAG/internal/domain/knowledge"
	domainragtrace "github.com/AmazingCYJ/AgentRAG/internal/domain/ragtrace"
	appconfig "github.com/AmazingCYJ/AgentRAG/internal/platform/config"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/util/guid"
)

type sseEvent struct {
	Name string
	Data json.RawMessage
}

type fixedChatGenerator struct {
	answer string
}

func (g fixedChatGenerator) Generate(_ context.Context, _ domainchat.GenerateRequest) (domainchat.GenerateResult, error) {
	return domainchat.GenerateResult{Answer: g.answer}, nil
}

func TestChatStreamCreatesConversationAndEmitsFinishEvents(t *testing.T) {
	conversationService := domainconversation.NewService(nil)
	server := startChatTestServer(t, conversationService)
	defer server.Shutdown()

	token := loginAndGetToken(t, server.GetListenedPort())
	request, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("http://127.0.0.1:%d/rag/v3/chat?question=%s&deepThinking=true", server.GetListenedPort(), "请介绍一下当前系统"),
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

	if !strings.HasPrefix(response.Header.Get("Content-Type"), "text/event-stream") {
		t.Fatalf("expected text/event-stream content type, got %s", response.Header.Get("Content-Type"))
	}

	events := readSSEEventsUntilDone(t, response.Body)
	if len(events) < 4 {
		t.Fatalf("expected at least 4 sse events, got %d", len(events))
	}
	if events[0].Name != "meta" {
		t.Fatalf("expected first event meta, got %s", events[0].Name)
	}

	var meta struct {
		ConversationID string `json:"conversationId"`
		TaskID         string `json:"taskId"`
	}
	if err := json.Unmarshal(events[0].Data, &meta); err != nil {
		t.Fatalf("decode meta payload failed: %v", err)
	}
	if meta.ConversationID == "" {
		t.Fatal("expected conversation id in meta payload")
	}
	if meta.TaskID == "" {
		t.Fatal("expected task id in meta payload")
	}

	foundThinking := false
	foundResponse := false
	var finish struct {
		MessageID string `json:"messageId"`
		Title     string `json:"title"`
	}
	for _, event := range events[1:] {
		switch event.Name {
		case "message":
			var payload struct {
				Type  string `json:"type"`
				Delta string `json:"delta"`
			}
			if err := json.Unmarshal(event.Data, &payload); err != nil {
				t.Fatalf("decode message payload failed: %v", err)
			}
			if payload.Type == "think" && payload.Delta != "" {
				foundThinking = true
			}
			if payload.Type == "response" && payload.Delta != "" {
				foundResponse = true
			}
		case "finish":
			if err := json.Unmarshal(event.Data, &finish); err != nil {
				t.Fatalf("decode finish payload failed: %v", err)
			}
		}
	}

	if !foundThinking {
		t.Fatal("expected at least one thinking message chunk")
	}
	if !foundResponse {
		t.Fatal("expected at least one response message chunk")
	}
	if finish.MessageID == "" {
		t.Fatal("expected finish payload to contain message id")
	}
	if finish.Title == "" {
		t.Fatal("expected finish payload to contain title")
	}
	if events[len(events)-1].Name != "done" {
		t.Fatalf("expected last event done, got %s", events[len(events)-1].Name)
	}

	sessions := conversationService.ListByUserID("u_admin")
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}
	if sessions[0].ConversationID != meta.ConversationID {
		t.Fatalf("expected session id %s, got %s", meta.ConversationID, sessions[0].ConversationID)
	}
	if sessions[0].Title != finish.Title {
		t.Fatalf("expected title %s, got %s", finish.Title, sessions[0].Title)
	}

	messages := conversationService.ListMessages(meta.ConversationID, "u_admin")
	if len(messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(messages))
	}
	if messages[0].Role != "user" {
		t.Fatalf("expected first message role user, got %s", messages[0].Role)
	}
	if messages[1].ID != finish.MessageID {
		t.Fatalf("expected assistant message id %s, got %s", finish.MessageID, messages[1].ID)
	}
	if messages[1].Content == "" {
		t.Fatal("expected assistant message content")
	}
	if messages[1].ThinkingContent == "" {
		t.Fatal("expected assistant thinking content")
	}
}

func TestStopTaskCancelsStreamingResponse(t *testing.T) {
	conversationService := domainconversation.NewService(nil)
	server := startChatTestServer(t, conversationService)
	defer server.Shutdown()

	token := loginAndGetToken(t, server.GetListenedPort())
	request, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("http://127.0.0.1:%d/rag/v3/chat?question=%s&deepThinking=true", server.GetListenedPort(), "请在输出前等待"),
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

	reader := bufio.NewReader(response.Body)
	metaEvent := readSingleSSEEvent(t, reader)
	if metaEvent.Name != "meta" {
		t.Fatalf("expected first event meta, got %s", metaEvent.Name)
	}

	var meta struct {
		ConversationID string `json:"conversationId"`
		TaskID         string `json:"taskId"`
	}
	if err := json.Unmarshal(metaEvent.Data, &meta); err != nil {
		t.Fatalf("decode meta payload failed: %v", err)
	}
	if meta.TaskID == "" {
		t.Fatal("expected task id in meta payload")
	}

	stopRequest, err := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("http://127.0.0.1:%d/rag/v3/stop?taskId=%s", server.GetListenedPort(), meta.TaskID),
		nil,
	)
	if err != nil {
		t.Fatalf("create stop request failed: %v", err)
	}
	stopRequest.Header.Set("Authorization", token)
	stopResponse, err := client.Do(stopRequest)
	if err != nil {
		t.Fatalf("request stop endpoint failed: %v", err)
	}
	stopResponse.Body.Close()

	events := append([]sseEvent{metaEvent}, readRemainingSSEEventsUntilDone(t, reader)...)
	foundCancel := false
	for _, event := range events {
		if event.Name == "cancel" {
			foundCancel = true
			var payload struct {
				MessageID string `json:"messageId"`
				Title     string `json:"title"`
			}
			if err := json.Unmarshal(event.Data, &payload); err != nil {
				t.Fatalf("decode cancel payload failed: %v", err)
			}
			if payload.MessageID == "" {
				t.Fatal("expected cancel payload to contain message id")
			}
			if payload.Title == "" {
				t.Fatal("expected cancel payload to contain title")
			}
		}
	}
	if !foundCancel {
		t.Fatal("expected cancel event after calling stop")
	}
	if events[len(events)-1].Name != "done" {
		t.Fatalf("expected last event done, got %s", events[len(events)-1].Name)
	}

	messages := conversationService.ListMessages(meta.ConversationID, "u_admin")
	if len(messages) != 2 {
		t.Fatalf("expected 2 stored messages after cancel, got %d", len(messages))
	}
	if messages[1].Role != "assistant" {
		t.Fatalf("expected second message role assistant, got %s", messages[1].Role)
	}
}

func TestChatStreamRejectsWhenConcurrencyLimitReached(t *testing.T) {
	conversationService := domainconversation.NewService(nil)
	chatService := domainchat.NewService(
		conversationService,
		nil,
		fixedChatGenerator{answer: strings.Repeat("持续输出。", 200)},
	)
	chatService.SetMaxConcurrent(1)
	server := newServerWithDeps(&appconfig.Config{
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
	}, guid.S(), serverDeps{
		conversationService: conversationService,
		chatService:         chatService,
	})
	server.SetAddr("127.0.0.1:0")
	if err := server.Start(); err != nil {
		t.Fatalf("start server failed: %v", err)
	}
	defer server.Shutdown()
	time.Sleep(100 * time.Millisecond)

	token := loginAndGetToken(t, server.GetListenedPort())
	client := &http.Client{Timeout: 5 * time.Second}
	firstRequest, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("http://127.0.0.1:%d/rag/v3/chat?question=%s", server.GetListenedPort(), "第一路请求"),
		nil,
	)
	if err != nil {
		t.Fatalf("create first chat request failed: %v", err)
	}
	firstRequest.Header.Set("Authorization", token)
	firstResponse, err := client.Do(firstRequest)
	if err != nil {
		t.Fatalf("request first chat stream failed: %v", err)
	}
	defer firstResponse.Body.Close()
	firstMeta := readSingleSSEEvent(t, bufio.NewReader(firstResponse.Body))
	if firstMeta.Name != "meta" {
		t.Fatalf("expected first stream meta event, got %s", firstMeta.Name)
	}

	secondRequest, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("http://127.0.0.1:%d/rag/v3/chat?question=%s", server.GetListenedPort(), "第二路请求"),
		nil,
	)
	if err != nil {
		t.Fatalf("create second chat request failed: %v", err)
	}
	secondRequest.Header.Set("Authorization", token)
	secondResponse, err := client.Do(secondRequest)
	if err != nil {
		t.Fatalf("request second chat stream failed: %v", err)
	}
	defer secondResponse.Body.Close()

	events := readSSEEventsUntilDone(t, secondResponse.Body)
	foundReject := false
	for _, event := range events {
		if event.Name != "reject" {
			continue
		}
		foundReject = true
		var payload struct {
			Type  string `json:"type"`
			Delta string `json:"delta"`
		}
		if err := json.Unmarshal(event.Data, &payload); err != nil {
			t.Fatalf("decode reject payload failed: %v", err)
		}
		if payload.Type != "response" || !strings.Contains(payload.Delta, "请求较多") {
			t.Fatalf("unexpected reject payload %#v", payload)
		}
	}
	if !foundReject {
		t.Fatalf("expected reject event, got %#v", events)
	}
}

func TestChatRouteUsesMCPToolContextWhenIntentMatches(t *testing.T) {
	conversationService := domainconversation.NewService(nil)
	intentTreeService := domainintenttree.NewService(nil)
	_, err := intentTreeService.CreateNode(domainintenttree.CreateRequest{
		IntentCode: "weather_intent",
		Name:       "天气查询",
		Level:      0,
		Kind:       2,
		MCPToolID:  "weather_query",
		Enabled:    1,
		Examples:   []string{"北京天气怎么样", "查询天气"},
	})
	if err != nil {
		t.Fatalf("create weather intent failed: %v", err)
	}

	server := newServerWithDeps(&appconfig.Config{
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
	}, guid.S(), serverDeps{
		conversationService: conversationService,
		intentTreeService:   intentTreeService,
	})
	server.SetAddr("127.0.0.1:0")
	if err := server.Start(); err != nil {
		t.Fatalf("start server failed: %v", err)
	}
	defer server.Shutdown()
	time.Sleep(100 * time.Millisecond)

	token := loginAndGetToken(t, server.GetListenedPort())
	request, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("http://127.0.0.1:%d/rag/v3/chat?question=%s", server.GetListenedPort(), "北京天气怎么样"),
		nil,
	)
	if err != nil {
		t.Fatalf("create chat request failed: %v", err)
	}
	request.Header.Set("Authorization", token)

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("request chat stream failed: %v", err)
	}
	defer response.Body.Close()

	events := readSSEEventsUntilDone(t, response.Body)
	var answer strings.Builder
	for _, event := range events {
		if event.Name != "message" {
			continue
		}
		var payload struct {
			Type  string `json:"type"`
			Delta string `json:"delta"`
		}
		if err := json.Unmarshal(event.Data, &payload); err != nil {
			t.Fatalf("decode message payload failed: %v", err)
		}
		if payload.Type == "response" {
			answer.WriteString(payload.Delta)
		}
	}
	if !strings.Contains(answer.String(), "晴") {
		t.Fatalf("expected response to contain weather tool context, got %s", answer.String())
	}
	if strings.Contains(answer.String(), "占位答案") || strings.Contains(answer.String(), "最小可用回答") {
		t.Fatalf("expected tool context answer instead of placeholder, got %s", answer.String())
	}
}

func TestChatUsesUploadedKnowledgeDocumentAndRecordsRetrieverTrace(t *testing.T) {
	conversationService := domainconversation.NewService(nil)
	knowledgeService := domainknowledge.NewService(nil)
	traceService := domainragtrace.NewService(nil)
	server := newServerWithDeps(&appconfig.Config{
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
	}, guid.S(), serverDeps{
		conversationService: conversationService,
		knowledgeService:    knowledgeService,
		ragTraceService:     traceService,
	})
	server.SetAddr("127.0.0.1:0")
	if err := server.Start(); err != nil {
		t.Fatalf("start server failed: %v", err)
	}
	defer server.Shutdown()
	time.Sleep(100 * time.Millisecond)

	token := loginAndGetToken(t, server.GetListenedPort())
	kbID := createKnowledgeBaseForChatTest(t, server.GetListenedPort(), token)
	docID := uploadKnowledgeDocumentForChatTest(t, server.GetListenedPort(), token, kbID)
	startKnowledgeDocumentChunkForChatTest(t, server.GetListenedPort(), token, docID)

	request, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("http://127.0.0.1:%d/rag/v3/chat?question=%s", server.GetListenedPort(), url.QueryEscape("报销流程需要准备什么材料")),
		nil,
	)
	if err != nil {
		t.Fatalf("create chat request failed: %v", err)
	}
	request.Header.Set("Authorization", token)

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("request chat stream failed: %v", err)
	}
	defer response.Body.Close()

	events := readSSEEventsUntilDone(t, response.Body)
	var meta struct {
		TaskID string `json:"taskId"`
	}
	var answer strings.Builder
	for _, event := range events {
		switch event.Name {
		case "meta":
			if err := json.Unmarshal(event.Data, &meta); err != nil {
				t.Fatalf("decode meta failed: %v", err)
			}
		case "message":
			var payload struct {
				Type  string `json:"type"`
				Delta string `json:"delta"`
			}
			if err := json.Unmarshal(event.Data, &payload); err != nil {
				t.Fatalf("decode message failed: %v", err)
			}
			if payload.Type == "response" {
				answer.WriteString(payload.Delta)
			}
		}
	}
	if meta.TaskID == "" {
		t.Fatal("expected chat task id")
	}
	answerText := answer.String()
	for _, expected := range []string{"发票", "报销制度.md", "知识库：财务知识库", "Chunk：0"} {
		if !strings.Contains(answerText, expected) {
			t.Fatalf("expected answer to contain %q, got %s", expected, answerText)
		}
	}

	runs := traceService.PageRuns(domainragtrace.RunQuery{TaskID: meta.TaskID, Size: 10})
	if runs.Total == 0 || len(runs.Records) == 0 {
		t.Fatal("expected trace run for knowledge chat")
	}
	nodes, err := traceService.ListNodes(runs.Records[0].TraceID)
	if err != nil {
		t.Fatalf("list trace nodes failed: %v", err)
	}
	for _, node := range nodes {
		if node.NodeType == "RETRIEVER" {
			return
		}
	}
	t.Fatalf("expected retriever node in trace, got %#v", nodes)
}

func startChatTestServer(t *testing.T, conversationService *domainconversation.Service) *ghttp.Server {
	t.Helper()

	server := newServerWithDeps(&appconfig.Config{
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
	}, guid.S(), serverDeps{
		conversationService: conversationService,
	})
	server.SetAddr("127.0.0.1:0")
	if err := server.Start(); err != nil {
		t.Fatalf("start server failed: %v", err)
	}
	time.Sleep(100 * time.Millisecond)
	return server
}

func createKnowledgeBaseForChatTest(t *testing.T, port int, token string) string {
	t.Helper()

	body := strings.NewReader(`{"name":"财务知识库","embeddingModel":"embedding-openai-large","collectionName":"fin_docs"}`)
	request, err := http.NewRequest(http.MethodPost, fmt.Sprintf("http://127.0.0.1:%d/knowledge-base", port), body)
	if err != nil {
		t.Fatalf("create knowledge base request failed: %v", err)
	}
	request.Header.Set("Authorization", token)
	request.Header.Set("Content-Type", "application/json")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("request create knowledge base failed: %v", err)
	}
	defer response.Body.Close()

	var responseBody struct {
		Code string `json:"code"`
		Data string `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&responseBody); err != nil {
		t.Fatalf("decode create knowledge base response failed: %v", err)
	}
	if responseBody.Code != "0" || responseBody.Data == "" {
		t.Fatalf("expected created knowledge base id, got code=%s data=%s", responseBody.Code, responseBody.Data)
	}
	return responseBody.Data
}

func uploadKnowledgeDocumentForChatTest(t *testing.T, port int, token, kbID string) string {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	_ = writer.WriteField("sourceType", "file")
	_ = writer.WriteField("processMode", "chunk")
	_ = writer.WriteField("chunkStrategy", "structure_aware")
	fileWriter, err := writer.CreateFormFile("file", "报销制度.md")
	if err != nil {
		t.Fatalf("create multipart file failed: %v", err)
	}
	if _, err := fileWriter.Write([]byte("报销流程需要准备发票、审批单和付款账号。")); err != nil {
		t.Fatalf("write multipart file failed: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer failed: %v", err)
	}

	request, err := http.NewRequest(http.MethodPost, fmt.Sprintf("http://127.0.0.1:%d/knowledge-base/%s/docs/upload", port, kbID), &body)
	if err != nil {
		t.Fatalf("create upload request failed: %v", err)
	}
	request.Header.Set("Authorization", token)
	request.Header.Set("Content-Type", writer.FormDataContentType())

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("request upload document failed: %v", err)
	}
	defer response.Body.Close()

	var responseBody struct {
		Code string `json:"code"`
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&responseBody); err != nil {
		t.Fatalf("decode upload document response failed: %v", err)
	}
	if responseBody.Code != "0" || responseBody.Data.ID == "" {
		t.Fatalf("expected uploaded document id, got code=%s id=%s", responseBody.Code, responseBody.Data.ID)
	}
	return responseBody.Data.ID
}

func startKnowledgeDocumentChunkForChatTest(t *testing.T, port int, token, docID string) {
	t.Helper()

	request, err := http.NewRequest(http.MethodPost, fmt.Sprintf("http://127.0.0.1:%d/knowledge-base/docs/%s/chunk", port, docID), nil)
	if err != nil {
		t.Fatalf("create start chunk request failed: %v", err)
	}
	request.Header.Set("Authorization", token)

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("request start chunk failed: %v", err)
	}
	defer response.Body.Close()

	var responseBody struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(response.Body).Decode(&responseBody); err != nil {
		t.Fatalf("decode start chunk response failed: %v", err)
	}
	if responseBody.Code != "0" {
		t.Fatalf("expected start chunk code 0, got %s", responseBody.Code)
	}
}

func readSSEEventsUntilDone(t *testing.T, body io.Reader) []sseEvent {
	t.Helper()

	reader := bufio.NewReader(body)
	return readRemainingSSEEventsUntilDone(t, reader)
}

func readRemainingSSEEventsUntilDone(t *testing.T, reader *bufio.Reader) []sseEvent {
	t.Helper()

	events := make([]sseEvent, 0, 8)
	for {
		event := readSingleSSEEvent(t, reader)
		events = append(events, event)
		if event.Name == "done" {
			return events
		}
	}
}

func readSingleSSEEvent(t *testing.T, reader *bufio.Reader) sseEvent {
	t.Helper()

	event := sseEvent{Name: "message"}
	dataLines := make([]string, 0, 2)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("read sse line failed: %v", err)
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			if len(dataLines) == 0 {
				continue
			}
			event.Data = json.RawMessage(strings.Join(dataLines, "\n"))
			return event
		}
		if strings.HasPrefix(line, "event:") {
			event.Name = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
			continue
		}
		if strings.HasPrefix(line, "data:") {
			dataLines = append(dataLines, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
		}
	}
}
