package httpx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	domainconversation "github.com/AmazingCYJ/AgentRAG/internal/domain/conversation"
	appconfig "github.com/AmazingCYJ/AgentRAG/internal/platform/config"
	"github.com/gogf/gf/v2/util/guid"
)

func TestListSessionsReturnsConversationsForAuthenticatedUser(t *testing.T) {
	conversationService := domainconversation.NewService()
	conversationService.UpsertConversation(domainconversation.Session{
		ConversationID: "c1",
		UserID:         "u_admin",
		Title:          "较早会话",
		LastTime:       time.Date(2026, 4, 14, 13, 0, 0, 0, time.UTC),
	})
	conversationService.UpsertConversation(domainconversation.Session{
		ConversationID: "c2",
		UserID:         "u_admin",
		Title:          "较新会话",
		LastTime:       time.Date(2026, 4, 14, 14, 0, 0, 0, time.UTC),
	})

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
	defer server.Shutdown()
	time.Sleep(100 * time.Millisecond)

	token := loginAndGetToken(t, server.GetListenedPort())

	request, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://127.0.0.1:%d/conversations", server.GetListenedPort()), nil)
	if err != nil {
		t.Fatalf("create request failed: %v", err)
	}
	request.Header.Set("Authorization", token)

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("request conversations failed: %v", err)
	}
	defer response.Body.Close()

	var body struct {
		Code string `json:"code"`
		Data []struct {
			ConversationID string `json:"conversationId"`
			Title          string `json:"title"`
			LastTime       string `json:"lastTime"`
		} `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if body.Code != "0" {
		t.Fatalf("expected code 0, got %s", body.Code)
	}
	if len(body.Data) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(body.Data))
	}
	if body.Data[0].ConversationID != "c2" {
		t.Fatalf("expected newest conversation first, got %s", body.Data[0].ConversationID)
	}
}

func TestListMessagesReturnsConversationMessagesForAuthenticatedUser(t *testing.T) {
	conversationService := domainconversation.NewService()
	conversationService.UpsertConversation(domainconversation.Session{
		ConversationID: "c1",
		UserID:         "u_admin",
		Title:          "测试会话",
		LastTime:       time.Date(2026, 4, 14, 14, 0, 0, 0, time.UTC),
	})
	thinkingDuration := 8
	vote := 1
	conversationService.AppendMessage(domainconversation.Message{
		ID:               "m1",
		ConversationID:   "c1",
		UserID:           "u_admin",
		Role:             "assistant",
		Content:          "你好",
		ThinkingContent:  "这是思考过程",
		ThinkingDuration: &thinkingDuration,
		Vote:             &vote,
		CreateTime:       time.Date(2026, 4, 14, 14, 1, 0, 0, time.UTC),
	})

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
	defer server.Shutdown()
	time.Sleep(100 * time.Millisecond)

	token := loginAndGetToken(t, server.GetListenedPort())

	request, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://127.0.0.1:%d/conversations/c1/messages", server.GetListenedPort()), nil)
	if err != nil {
		t.Fatalf("create request failed: %v", err)
	}
	request.Header.Set("Authorization", token)

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("request messages failed: %v", err)
	}
	defer response.Body.Close()

	var body struct {
		Code string `json:"code"`
		Data []struct {
			ID               string `json:"id"`
			ConversationID   string `json:"conversationId"`
			Role             string `json:"role"`
			Content          string `json:"content"`
			ThinkingContent  string `json:"thinkingContent"`
			ThinkingDuration int    `json:"thinkingDuration"`
			Vote             int    `json:"vote"`
		} `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if body.Code != "0" {
		t.Fatalf("expected code 0, got %s", body.Code)
	}
	if len(body.Data) != 1 {
		t.Fatalf("expected 1 message, got %d", len(body.Data))
	}
	if body.Data[0].ID != "m1" {
		t.Fatalf("expected message id m1, got %s", body.Data[0].ID)
	}
	if body.Data[0].ThinkingDuration != 8 {
		t.Fatalf("expected thinking duration 8, got %d", body.Data[0].ThinkingDuration)
	}
}

func TestRenameSessionUpdatesConversationTitle(t *testing.T) {
	conversationService := domainconversation.NewService()
	conversationService.UpsertConversation(domainconversation.Session{
		ConversationID: "c1",
		UserID:         "u_admin",
		Title:          "旧标题",
		LastTime:       time.Date(2026, 4, 14, 14, 0, 0, 0, time.UTC),
	})

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
	defer server.Shutdown()
	time.Sleep(100 * time.Millisecond)

	token := loginAndGetToken(t, server.GetListenedPort())
	payload, _ := json.Marshal(map[string]string{"title": "新标题"})
	request, err := http.NewRequest(http.MethodPut, fmt.Sprintf("http://127.0.0.1:%d/conversations/c1", server.GetListenedPort()), bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("create request failed: %v", err)
	}
	request.Header.Set("Authorization", token)
	request.Header.Set("Content-Type", "application/json")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("rename request failed: %v", err)
	}
	defer response.Body.Close()

	sessions := conversationService.ListByUserID("u_admin")
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}
	if sessions[0].Title != "新标题" {
		t.Fatalf("expected renamed title 新标题, got %s", sessions[0].Title)
	}
}

func TestDeleteSessionRemovesConversationAndMessages(t *testing.T) {
	conversationService := domainconversation.NewService()
	conversationService.UpsertConversation(domainconversation.Session{
		ConversationID: "c1",
		UserID:         "u_admin",
		Title:          "待删除会话",
		LastTime:       time.Date(2026, 4, 14, 14, 0, 0, 0, time.UTC),
	})
	conversationService.AppendMessage(domainconversation.Message{
		ID:             "m1",
		ConversationID: "c1",
		UserID:         "u_admin",
		Role:           "user",
		Content:        "hello",
		CreateTime:     time.Date(2026, 4, 14, 14, 1, 0, 0, time.UTC),
	})

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
	defer server.Shutdown()
	time.Sleep(100 * time.Millisecond)

	token := loginAndGetToken(t, server.GetListenedPort())
	request, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("http://127.0.0.1:%d/conversations/c1", server.GetListenedPort()), nil)
	if err != nil {
		t.Fatalf("create request failed: %v", err)
	}
	request.Header.Set("Authorization", token)

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("delete request failed: %v", err)
	}
	defer response.Body.Close()

	if len(conversationService.ListByUserID("u_admin")) != 0 {
		t.Fatal("expected session to be deleted")
	}
	if len(conversationService.ListMessages("c1", "u_admin")) != 0 {
		t.Fatal("expected conversation messages to be deleted")
	}
}

func loginAndGetToken(t *testing.T, port int) string {
	t.Helper()

	payload, _ := json.Marshal(map[string]string{
		"username": "admin",
		"password": "admin123",
	})
	response, err := http.Post(
		fmt.Sprintf("http://127.0.0.1:%d/auth/login", port),
		"application/json",
		bytes.NewReader(payload),
	)
	if err != nil {
		t.Fatalf("login request failed: %v", err)
	}
	defer response.Body.Close()

	var body struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode login response failed: %v", err)
	}
	if body.Data.Token == "" {
		t.Fatal("expected login token")
	}
	return body.Data.Token
}
