package httpx

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	domainconversation "github.com/AmazingCYJ/AgentRAG/internal/domain/conversation"
	domainintenttree "github.com/AmazingCYJ/AgentRAG/internal/domain/intenttree"
	appconfig "github.com/AmazingCYJ/AgentRAG/internal/platform/config"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/util/guid"
)

type sseEvent struct {
	Name string
	Data json.RawMessage
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
	foundWeather := false
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
		if payload.Type == "response" && strings.Contains(payload.Delta, "北京") {
			foundWeather = true
		}
	}
	if !foundWeather {
		t.Fatal("expected response to contain weather tool context")
	}
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
