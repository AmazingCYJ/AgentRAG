package httpx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"testing"
	"time"

	appconfig "github.com/AmazingCYJ/AgentRAG/internal/platform/config"
	"github.com/gogf/gf/v2/util/guid"
)

func TestFrontendAPIPrefixedSmokeFlow(t *testing.T) {
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

	baseURL := fmt.Sprintf("http://127.0.0.1:%d/api/ragent", server.GetListenedPort())
	token := loginThroughFrontendPrefix(t, baseURL)

	assertCurrentUserContract(t, baseURL, token)
	assertSettingsContract(t, baseURL, token)
	assertDashboardContract(t, baseURL, token)
	assertSampleQuestionContract(t, baseURL, token)
	assertUserPageContract(t, baseURL, token)

	kbID := createKnowledgeBaseThroughFrontendPrefix(t, baseURL, token)
	docID := uploadKnowledgeDocumentThroughFrontendPrefix(t, baseURL, token, kbID)
	startDocumentChunkThroughFrontendPrefix(t, baseURL, token, docID)
	assertKnowledgeSearchContract(t, baseURL, token)

	pipelineID := createIngestionPipelineThroughFrontendPrefix(t, baseURL, token)
	createIngestionTaskThroughFrontendPrefix(t, baseURL, token, pipelineID)
	createIntentNodeThroughFrontendPrefix(t, baseURL, token)
	createMappingThroughFrontendPrefix(t, baseURL, token)

	conversationID, messageID := streamChatThroughFrontendPrefix(t, baseURL, token)
	assertConversationContract(t, baseURL, token, conversationID, messageID)
	assertTraceContract(t, baseURL, token, conversationID)
}

func loginThroughFrontendPrefix(t *testing.T, baseURL string) string {
	t.Helper()

	var body struct {
		Code string `json:"code"`
		Data struct {
			UserID   string `json:"userId"`
			Username string `json:"username"`
			Role     string `json:"role"`
			Token    string `json:"token"`
		} `json:"data"`
	}
	postJSONThroughFrontendPrefix(t, baseURL+"/auth/login", "", map[string]any{
		"username": "admin",
		"password": "admin123",
	}, &body)
	if body.Code != "0" || body.Data.Token == "" {
		t.Fatalf("unexpected frontend login response %#v", body)
	}
	if body.Data.UserID == "" || body.Data.Username != "admin" || body.Data.Role != "admin" {
		t.Fatalf("unexpected frontend user payload %#v", body.Data)
	}
	return body.Data.Token
}

func assertCurrentUserContract(t *testing.T, baseURL, token string) {
	t.Helper()

	var body struct {
		Code string `json:"code"`
		Data struct {
			UserID   string `json:"userId"`
			Username string `json:"username"`
			Role     string `json:"role"`
		} `json:"data"`
	}
	getJSONThroughFrontendPrefix(t, baseURL+"/user/me", token, &body)
	if body.Code != "0" || body.Data.UserID == "" || body.Data.Username != "admin" {
		t.Fatalf("unexpected current user response %#v", body)
	}
}

func assertSettingsContract(t *testing.T, baseURL, token string) {
	t.Helper()

	var body struct {
		Code string `json:"code"`
		Data struct {
			Upload struct {
				MaxFileSize int64 `json:"maxFileSize"`
			} `json:"upload"`
			RAG struct {
				RateLimit struct {
					Global struct {
						MaxConcurrent int `json:"maxConcurrent"`
					} `json:"global"`
				} `json:"rateLimit"`
			} `json:"rag"`
			AI struct {
				Chat struct {
					DefaultModel string `json:"defaultModel"`
				} `json:"chat"`
			} `json:"ai"`
		} `json:"data"`
	}
	getJSONThroughFrontendPrefix(t, baseURL+"/rag/settings", token, &body)
	if body.Code != "0" || body.Data.Upload.MaxFileSize <= 0 || body.Data.RAG.RateLimit.Global.MaxConcurrent <= 0 || body.Data.AI.Chat.DefaultModel == "" {
		t.Fatalf("unexpected settings response %#v", body)
	}
}

func assertDashboardContract(t *testing.T, baseURL, token string) {
	t.Helper()

	var overview struct {
		Code string `json:"code"`
		Data struct {
			Window string `json:"window"`
			KPIs   struct {
				TotalUsers struct {
					Value int `json:"value"`
				} `json:"totalUsers"`
			} `json:"kpis"`
		} `json:"data"`
	}
	getJSONThroughFrontendPrefix(t, baseURL+"/admin/dashboard/overview?window=24h", token, &overview)
	if overview.Code != "0" || overview.Data.Window == "" || overview.Data.KPIs.TotalUsers.Value <= 0 {
		t.Fatalf("unexpected dashboard overview %#v", overview)
	}

	var performance struct {
		Code string `json:"code"`
		Data struct {
			SuccessRate float64 `json:"successRate"`
		} `json:"data"`
	}
	getJSONThroughFrontendPrefix(t, baseURL+"/admin/dashboard/performance?window=24h", token, &performance)
	if performance.Code != "0" {
		t.Fatalf("unexpected dashboard performance %#v", performance)
	}

	var trends struct {
		Code string `json:"code"`
		Data struct {
			Series []struct {
				Name string `json:"name"`
			} `json:"series"`
		} `json:"data"`
	}
	getJSONThroughFrontendPrefix(t, baseURL+"/admin/dashboard/trends?metric=sessions&window=7d&granularity=day", token, &trends)
	if trends.Code != "0" || len(trends.Data.Series) == 0 {
		t.Fatalf("unexpected dashboard trends %#v", trends)
	}
}

func assertSampleQuestionContract(t *testing.T, baseURL, token string) {
	t.Helper()

	var body struct {
		Code string `json:"code"`
		Data []struct {
			ID       string `json:"id"`
			Question string `json:"question"`
		} `json:"data"`
	}
	getJSONThroughFrontendPrefix(t, baseURL+"/rag/sample-questions", token, &body)
	if body.Code != "0" || len(body.Data) == 0 || body.Data[0].Question == "" {
		t.Fatalf("unexpected sample questions %#v", body)
	}
}

func assertUserPageContract(t *testing.T, baseURL, token string) {
	t.Helper()

	var body struct {
		Code string `json:"code"`
		Data struct {
			Records []struct {
				ID       string `json:"id"`
				Username string `json:"username"`
			} `json:"records"`
			Total int `json:"total"`
		} `json:"data"`
	}
	getJSONThroughFrontendPrefix(t, baseURL+"/users?current=1&size=10", token, &body)
	if body.Code != "0" || body.Data.Total <= 0 || len(body.Data.Records) == 0 {
		t.Fatalf("unexpected users page %#v", body)
	}
}

func createKnowledgeBaseThroughFrontendPrefix(t *testing.T, baseURL, token string) string {
	t.Helper()

	var body struct {
		Code string `json:"code"`
		Data string `json:"data"`
	}
	postJSONThroughFrontendPrefix(t, baseURL+"/knowledge-base", token, map[string]any{
		"name":           "前端契约知识库",
		"embeddingModel": "embedding-openai-large",
		"collectionName": "frontend_contract_docs",
	}, &body)
	if body.Code != "0" || body.Data == "" {
		t.Fatalf("unexpected create knowledge base response %#v", body)
	}

	var page struct {
		Code string `json:"code"`
		Data struct {
			Records []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"records"`
			Total int `json:"total"`
		} `json:"data"`
	}
	getJSONThroughFrontendPrefix(t, baseURL+"/knowledge-base?current=1&size=10&name=前端契约", token, &page)
	if page.Code != "0" || page.Data.Total <= 0 || page.Data.Records[0].ID == "" {
		t.Fatalf("unexpected knowledge base page %#v", page)
	}
	return body.Data
}

func uploadKnowledgeDocumentThroughFrontendPrefix(t *testing.T, baseURL, token, kbID string) string {
	t.Helper()

	var payload bytes.Buffer
	writer := multipart.NewWriter(&payload)
	_ = writer.WriteField("sourceType", "file")
	_ = writer.WriteField("processMode", "chunk")
	_ = writer.WriteField("chunkStrategy", "structure_aware")
	fileWriter, err := writer.CreateFormFile("file", "前端契约.md")
	if err != nil {
		t.Fatalf("create multipart file failed: %v", err)
	}
	if _, err := fileWriter.Write([]byte("前端契约测试用于验证登录、知识库、对话和追踪接口。")); err != nil {
		t.Fatalf("write multipart file failed: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer failed: %v", err)
	}

	request, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/knowledge-base/%s/docs/upload", baseURL, kbID), &payload)
	if err != nil {
		t.Fatalf("create upload request failed: %v", err)
	}
	request.Header.Set("Authorization", token)
	request.Header.Set("Content-Type", writer.FormDataContentType())

	var body struct {
		Code string `json:"code"`
		Data struct {
			ID      string `json:"id"`
			DocName string `json:"docName"`
		} `json:"data"`
	}
	doFrontendJSONRequest(t, request, &body)
	if body.Code != "0" || body.Data.ID == "" || body.Data.DocName == "" {
		t.Fatalf("unexpected document upload response %#v", body)
	}
	return body.Data.ID
}

func startDocumentChunkThroughFrontendPrefix(t *testing.T, baseURL, token, docID string) {
	t.Helper()

	var body struct {
		Code string `json:"code"`
	}
	postJSONThroughFrontendPrefix(t, fmt.Sprintf("%s/knowledge-base/docs/%s/chunk", baseURL, docID), token, nil, &body)
	if body.Code != "0" {
		t.Fatalf("unexpected start chunk response %#v", body)
	}

	var chunks struct {
		Code string `json:"code"`
		Data struct {
			Records []struct {
				ID      string `json:"id"`
				Content string `json:"content"`
			} `json:"records"`
		} `json:"data"`
	}
	getJSONThroughFrontendPrefix(t, fmt.Sprintf("%s/knowledge-base/docs/%s/chunks?current=1&size=10", baseURL, docID), token, &chunks)
	if chunks.Code != "0" || len(chunks.Data.Records) == 0 || chunks.Data.Records[0].Content == "" {
		t.Fatalf("unexpected chunk page response %#v", chunks)
	}
}

func assertKnowledgeSearchContract(t *testing.T, baseURL, token string) {
	t.Helper()

	var body struct {
		Code string `json:"code"`
		Data []struct {
			ID      string `json:"id"`
			DocName string `json:"docName"`
		} `json:"data"`
	}
	getJSONThroughFrontendPrefix(t, baseURL+"/knowledge-base/docs/search?keyword=前端契约&limit=8", token, &body)
	if body.Code != "0" || len(body.Data) == 0 || body.Data[0].DocName == "" {
		t.Fatalf("unexpected document search response %#v", body)
	}
}

func createIngestionPipelineThroughFrontendPrefix(t *testing.T, baseURL, token string) string {
	t.Helper()

	var body struct {
		Code string `json:"code"`
		Data struct {
			ID    string `json:"id"`
			Nodes []struct {
				NodeID string `json:"nodeId"`
			} `json:"nodes"`
		} `json:"data"`
	}
	postJSONThroughFrontendPrefix(t, baseURL+"/ingestion/pipelines", token, map[string]any{
		"name":        "前端采集流水线",
		"description": "前端契约测试",
		"nodes": []map[string]any{
			{"nodeId": "fetcher", "nodeType": "fetcher", "nextNodeId": "parser"},
			{"nodeId": "parser", "nodeType": "parser"},
		},
	}, &body)
	if body.Code != "0" || body.Data.ID == "" || len(body.Data.Nodes) != 2 {
		t.Fatalf("unexpected ingestion pipeline response %#v", body)
	}
	return body.Data.ID
}

func createIngestionTaskThroughFrontendPrefix(t *testing.T, baseURL, token, pipelineID string) {
	t.Helper()

	var body struct {
		Code string `json:"code"`
		Data struct {
			TaskID     string `json:"taskId"`
			PipelineID string `json:"pipelineId"`
			Status     string `json:"status"`
		} `json:"data"`
	}
	postJSONThroughFrontendPrefix(t, baseURL+"/ingestion/tasks", token, map[string]any{
		"pipelineId": pipelineID,
		"source": map[string]any{
			"type":     "url",
			"location": "https://example.com/frontend-contract",
			"fileName": "frontend-contract.md",
		},
		"metadata": map[string]any{"scene": "frontend-contract"},
	}, &body)
	if body.Code != "0" || body.Data.TaskID == "" || body.Data.PipelineID != pipelineID {
		t.Fatalf("unexpected ingestion task response %#v", body)
	}
}

func createIntentNodeThroughFrontendPrefix(t *testing.T, baseURL, token string) {
	t.Helper()

	var createBody struct {
		Code string `json:"code"`
		Data string `json:"data"`
	}
	postJSONThroughFrontendPrefix(t, baseURL+"/intent-tree", token, map[string]any{
		"intentCode": "frontend_contract",
		"name":       "前端契约",
		"level":      0,
		"kind":       1,
		"enabled":    1,
		"examples":   []string{"前端如何调用"},
	}, &createBody)
	if createBody.Code != "0" || createBody.Data == "" {
		t.Fatalf("unexpected intent create response %#v", createBody)
	}

	var treeBody struct {
		Code string `json:"code"`
		Data []struct {
			ID         string `json:"id"`
			IntentCode string `json:"intentCode"`
		} `json:"data"`
	}
	getJSONThroughFrontendPrefix(t, baseURL+"/intent-tree/trees", token, &treeBody)
	if treeBody.Code != "0" || len(treeBody.Data) == 0 {
		t.Fatalf("unexpected intent tree response %#v", treeBody)
	}
}

func createMappingThroughFrontendPrefix(t *testing.T, baseURL, token string) {
	t.Helper()

	var createBody struct {
		Code string `json:"code"`
		Data string `json:"data"`
	}
	postJSONThroughFrontendPrefix(t, baseURL+"/mappings", token, map[string]any{
		"sourceTerm": "前端联调",
		"targetTerm": "前端契约测试",
		"matchType":  1,
		"priority":   1,
		"enabled":    true,
	}, &createBody)
	if createBody.Code != "0" || createBody.Data == "" {
		t.Fatalf("unexpected mapping create response %#v", createBody)
	}

	var pageBody struct {
		Code string `json:"code"`
		Data struct {
			Records []struct {
				ID         string `json:"id"`
				SourceTerm string `json:"sourceTerm"`
			} `json:"records"`
		} `json:"data"`
	}
	getJSONThroughFrontendPrefix(t, baseURL+"/mappings?current=1&size=10&keyword=前端", token, &pageBody)
	if pageBody.Code != "0" || len(pageBody.Data.Records) == 0 {
		t.Fatalf("unexpected mapping page response %#v", pageBody)
	}
}

func streamChatThroughFrontendPrefix(t *testing.T, baseURL, token string) (string, string) {
	t.Helper()

	request, err := http.NewRequest(http.MethodGet, baseURL+"/rag/v3/chat?question=前端契约测试能否回答&deepThinking=true", nil)
	if err != nil {
		t.Fatalf("create chat request failed: %v", err)
	}
	request.Header.Set("Authorization", token)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("request chat stream failed: %v", err)
	}
	defer response.Body.Close()
	if !strings.HasPrefix(response.Header.Get("Content-Type"), "text/event-stream") {
		t.Fatalf("expected sse content type, got %s", response.Header.Get("Content-Type"))
	}

	events := readSSEEventsUntilDone(t, response.Body)
	var (
		conversationID string
		messageID      string
		foundMessage   bool
	)
	for _, event := range events {
		switch event.Name {
		case "meta":
			var payload struct {
				ConversationID string `json:"conversationId"`
			}
			if err := json.Unmarshal(event.Data, &payload); err != nil {
				t.Fatalf("decode chat meta failed: %v", err)
			}
			conversationID = payload.ConversationID
		case "message":
			foundMessage = true
		case "finish":
			var payload struct {
				MessageID string `json:"messageId"`
			}
			if err := json.Unmarshal(event.Data, &payload); err != nil {
				t.Fatalf("decode chat finish failed: %v", err)
			}
			messageID = payload.MessageID
		}
	}
	if conversationID == "" || messageID == "" || !foundMessage {
		t.Fatalf("unexpected chat sse events %#v", events)
	}
	return conversationID, messageID
}

func assertConversationContract(t *testing.T, baseURL, token, conversationID, messageID string) {
	t.Helper()

	var messages struct {
		Code string `json:"code"`
		Data []struct {
			ID             string `json:"id"`
			ConversationID string `json:"conversationId"`
			Role           string `json:"role"`
			Content        string `json:"content"`
		} `json:"data"`
	}
	getJSONThroughFrontendPrefix(t, fmt.Sprintf("%s/conversations/%s/messages", baseURL, conversationID), token, &messages)
	if messages.Code != "0" || len(messages.Data) < 2 {
		t.Fatalf("unexpected conversation messages %#v", messages)
	}

	var feedback struct {
		Code string `json:"code"`
	}
	postJSONThroughFrontendPrefix(t, fmt.Sprintf("%s/conversations/messages/%s/feedback", baseURL, messageID), token, map[string]any{
		"vote": 1,
	}, &feedback)
	if feedback.Code != "0" {
		t.Fatalf("unexpected feedback response %#v", feedback)
	}

	var sessions struct {
		Code string `json:"code"`
		Data []struct {
			ConversationID string `json:"conversationId"`
			Title          string `json:"title"`
		} `json:"data"`
	}
	getJSONThroughFrontendPrefix(t, baseURL+"/conversations", token, &sessions)
	if sessions.Code != "0" || len(sessions.Data) == 0 {
		t.Fatalf("unexpected conversation list %#v", sessions)
	}
}

func assertTraceContract(t *testing.T, baseURL, token, conversationID string) {
	t.Helper()

	var runs struct {
		Code string `json:"code"`
		Data struct {
			Records []struct {
				TraceID        string `json:"traceId"`
				ConversationID string `json:"conversationId"`
			} `json:"records"`
		} `json:"data"`
	}
	getJSONThroughFrontendPrefix(t, fmt.Sprintf("%s/rag/traces/runs?current=1&size=10&conversationId=%s", baseURL, conversationID), token, &runs)
	if runs.Code != "0" || len(runs.Data.Records) == 0 || runs.Data.Records[0].TraceID == "" {
		t.Fatalf("unexpected trace runs %#v", runs)
	}

	var detail struct {
		Code string `json:"code"`
		Data struct {
			Run struct {
				TraceID string `json:"traceId"`
			} `json:"run"`
			Nodes []struct {
				NodeID string `json:"nodeId"`
			} `json:"nodes"`
		} `json:"data"`
	}
	getJSONThroughFrontendPrefix(t, fmt.Sprintf("%s/rag/traces/runs/%s", baseURL, runs.Data.Records[0].TraceID), token, &detail)
	if detail.Code != "0" || detail.Data.Run.TraceID == "" || len(detail.Data.Nodes) == 0 {
		t.Fatalf("unexpected trace detail %#v", detail)
	}
}

func getJSONThroughFrontendPrefix(t *testing.T, endpoint, token string, target any) {
	t.Helper()

	request, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		t.Fatalf("create get request failed: %v", err)
	}
	if token != "" {
		request.Header.Set("Authorization", token)
	}
	doFrontendJSONRequest(t, request, target)
}

func postJSONThroughFrontendPrefix(t *testing.T, endpoint, token string, payload any, target any) {
	t.Helper()

	var reader io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal payload failed: %v", err)
		}
		reader = bytes.NewReader(data)
	}
	request, err := http.NewRequest(http.MethodPost, endpoint, reader)
	if err != nil {
		t.Fatalf("create post request failed: %v", err)
	}
	if token != "" {
		request.Header.Set("Authorization", token)
	}
	request.Header.Set("Content-Type", "application/json")
	doFrontendJSONRequest(t, request, target)
}

func doFrontendJSONRequest(t *testing.T, request *http.Request, target any) {
	t.Helper()

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("request %s %s failed: %v", request.Method, request.URL.String(), err)
	}
	defer response.Body.Close()
	if response.StatusCode >= http.StatusBadRequest {
		t.Fatalf("request %s %s returned status %d", request.Method, request.URL.String(), response.StatusCode)
	}
	if err := json.NewDecoder(response.Body).Decode(target); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
}
