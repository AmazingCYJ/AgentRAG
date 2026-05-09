package httpx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	appconfig "github.com/AmazingCYJ/AgentRAG/internal/platform/config"
	"github.com/gogf/gf/v2/util/guid"
	_ "modernc.org/sqlite"
)

func TestKnowledgeBaseCRUDAndChunkStrategies(t *testing.T) {
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
	kbID := createKnowledgeBaseForTest(t, server.GetListenedPort(), token, map[string]any{
		"name":           "产品文档库",
		"embeddingModel": "embedding-openai-large",
		"collectionName": "productdocs",
	})

	strategyRequest, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("http://127.0.0.1:%d/knowledge-base/chunk-strategies", server.GetListenedPort()),
		nil,
	)
	if err != nil {
		t.Fatalf("create strategy request failed: %v", err)
	}
	strategyRequest.Header.Set("Authorization", token)

	strategyResponse, err := http.DefaultClient.Do(strategyRequest)
	if err != nil {
		t.Fatalf("request chunk strategies failed: %v", err)
	}
	defer strategyResponse.Body.Close()

	var strategyBody struct {
		Code string `json:"code"`
		Data []struct {
			Value         string         `json:"value"`
			Label         string         `json:"label"`
			DefaultConfig map[string]int `json:"defaultConfig"`
		} `json:"data"`
	}
	if err := json.NewDecoder(strategyResponse.Body).Decode(&strategyBody); err != nil {
		t.Fatalf("decode chunk strategies failed: %v", err)
	}
	if strategyBody.Code != "0" {
		t.Fatalf("expected strategy code 0, got %s", strategyBody.Code)
	}
	if len(strategyBody.Data) == 0 {
		t.Fatal("expected chunk strategy options")
	}

	pageRequest, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("http://127.0.0.1:%d/knowledge-base?current=1&size=10&name=%s", server.GetListenedPort(), "产品"),
		nil,
	)
	if err != nil {
		t.Fatalf("create knowledge-base page request failed: %v", err)
	}
	pageRequest.Header.Set("Authorization", token)

	pageResponse, err := http.DefaultClient.Do(pageRequest)
	if err != nil {
		t.Fatalf("request knowledge-base page failed: %v", err)
	}
	defer pageResponse.Body.Close()

	var pageBody struct {
		Code string `json:"code"`
		Data struct {
			Records []struct {
				ID             string `json:"id"`
				Name           string `json:"name"`
				EmbeddingModel string `json:"embeddingModel"`
				CollectionName string `json:"collectionName"`
			} `json:"records"`
			Total int `json:"total"`
		} `json:"data"`
	}
	if err := json.NewDecoder(pageResponse.Body).Decode(&pageBody); err != nil {
		t.Fatalf("decode knowledge-base page failed: %v", err)
	}
	if pageBody.Code != "0" {
		t.Fatalf("expected page code 0, got %s", pageBody.Code)
	}
	if pageBody.Data.Total == 0 || len(pageBody.Data.Records) == 0 {
		t.Fatal("expected knowledge-base page records")
	}

	detailRequest, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("http://127.0.0.1:%d/knowledge-base/%s", server.GetListenedPort(), kbID),
		nil,
	)
	if err != nil {
		t.Fatalf("create knowledge-base detail request failed: %v", err)
	}
	detailRequest.Header.Set("Authorization", token)

	detailResponse, err := http.DefaultClient.Do(detailRequest)
	if err != nil {
		t.Fatalf("request knowledge-base detail failed: %v", err)
	}
	defer detailResponse.Body.Close()

	var detailBody struct {
		Code string `json:"code"`
		Data struct {
			ID             string `json:"id"`
			Name           string `json:"name"`
			EmbeddingModel string `json:"embeddingModel"`
			CollectionName string `json:"collectionName"`
		} `json:"data"`
	}
	if err := json.NewDecoder(detailResponse.Body).Decode(&detailBody); err != nil {
		t.Fatalf("decode knowledge-base detail failed: %v", err)
	}
	if detailBody.Code != "0" {
		t.Fatalf("expected detail code 0, got %s", detailBody.Code)
	}
	if detailBody.Data.ID != kbID {
		t.Fatalf("expected knowledge-base id %s, got %s", kbID, detailBody.Data.ID)
	}

	updatePayload, _ := json.Marshal(map[string]any{
		"name":           "产品知识库",
		"embeddingModel": "embedding-local-bge",
	})
	updateRequest, err := http.NewRequest(
		http.MethodPut,
		fmt.Sprintf("http://127.0.0.1:%d/knowledge-base/%s", server.GetListenedPort(), kbID),
		bytes.NewReader(updatePayload),
	)
	if err != nil {
		t.Fatalf("create update request failed: %v", err)
	}
	updateRequest.Header.Set("Authorization", token)
	updateRequest.Header.Set("Content-Type", "application/json")

	updateResponse, err := http.DefaultClient.Do(updateRequest)
	if err != nil {
		t.Fatalf("request update knowledge-base failed: %v", err)
	}
	defer updateResponse.Body.Close()

	var updateBody struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(updateResponse.Body).Decode(&updateBody); err != nil {
		t.Fatalf("decode update knowledge-base response failed: %v", err)
	}
	if updateBody.Code != "0" {
		t.Fatalf("expected update code 0, got %s", updateBody.Code)
	}

	deleteRequest, err := http.NewRequest(
		http.MethodDelete,
		fmt.Sprintf("http://127.0.0.1:%d/knowledge-base/%s", server.GetListenedPort(), kbID),
		nil,
	)
	if err != nil {
		t.Fatalf("create delete request failed: %v", err)
	}
	deleteRequest.Header.Set("Authorization", token)

	deleteResponse, err := http.DefaultClient.Do(deleteRequest)
	if err != nil {
		t.Fatalf("request delete knowledge-base failed: %v", err)
	}
	defer deleteResponse.Body.Close()

	var deleteBody struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(deleteResponse.Body).Decode(&deleteBody); err != nil {
		t.Fatalf("decode delete knowledge-base response failed: %v", err)
	}
	if deleteBody.Code != "0" {
		t.Fatalf("expected delete code 0, got %s", deleteBody.Code)
	}
}

func TestKnowledgeDocumentAndChunkLifecycle(t *testing.T) {
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
	kbID := createKnowledgeBaseForTest(t, server.GetListenedPort(), token, map[string]any{
		"name":           "运维文档库",
		"embeddingModel": "embedding-openai-large",
		"collectionName": "opsdocs",
	})
	docID := uploadKnowledgeDocumentForTest(t, server.GetListenedPort(), token, kbID)

	docsRequest, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("http://127.0.0.1:%d/knowledge-base/%s/docs?current=1&size=10&keyword=%s", server.GetListenedPort(), kbID, "手册"),
		nil,
	)
	if err != nil {
		t.Fatalf("create docs page request failed: %v", err)
	}
	docsRequest.Header.Set("Authorization", token)

	docsResponse, err := http.DefaultClient.Do(docsRequest)
	if err != nil {
		t.Fatalf("request docs page failed: %v", err)
	}
	defer docsResponse.Body.Close()

	var docsBody struct {
		Code string `json:"code"`
		Data struct {
			Records []struct {
				ID          string `json:"id"`
				KBID        string `json:"kbId"`
				DocName     string `json:"docName"`
				ChunkCount  int    `json:"chunkCount"`
				ProcessMode string `json:"processMode"`
			} `json:"records"`
			Total int `json:"total"`
		} `json:"data"`
	}
	if err := json.NewDecoder(docsResponse.Body).Decode(&docsBody); err != nil {
		t.Fatalf("decode docs page failed: %v", err)
	}
	if docsBody.Code != "0" {
		t.Fatalf("expected docs page code 0, got %s", docsBody.Code)
	}
	if docsBody.Data.Total == 0 || len(docsBody.Data.Records) == 0 {
		t.Fatal("expected docs page records")
	}

	searchRequest, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("http://127.0.0.1:%d/knowledge-base/docs/search?keyword=%s&limit=6", server.GetListenedPort(), "运维"),
		nil,
	)
	if err != nil {
		t.Fatalf("create doc search request failed: %v", err)
	}
	searchRequest.Header.Set("Authorization", token)

	searchResponse, err := http.DefaultClient.Do(searchRequest)
	if err != nil {
		t.Fatalf("request doc search failed: %v", err)
	}
	defer searchResponse.Body.Close()

	var searchBody struct {
		Code string `json:"code"`
		Data []struct {
			ID      string `json:"id"`
			KBID    string `json:"kbId"`
			DocName string `json:"docName"`
			KBName  string `json:"kbName"`
		} `json:"data"`
	}
	if err := json.NewDecoder(searchResponse.Body).Decode(&searchBody); err != nil {
		t.Fatalf("decode doc search failed: %v", err)
	}
	if searchBody.Code != "0" {
		t.Fatalf("expected doc search code 0, got %s", searchBody.Code)
	}

	detailRequest, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("http://127.0.0.1:%d/knowledge-base/docs/%s", server.GetListenedPort(), docID),
		nil,
	)
	if err != nil {
		t.Fatalf("create doc detail request failed: %v", err)
	}
	detailRequest.Header.Set("Authorization", token)

	detailResponse, err := http.DefaultClient.Do(detailRequest)
	if err != nil {
		t.Fatalf("request doc detail failed: %v", err)
	}
	defer detailResponse.Body.Close()

	var detailBody struct {
		Code string `json:"code"`
		Data struct {
			ID            string `json:"id"`
			DocName       string `json:"docName"`
			ProcessMode   string `json:"processMode"`
			ChunkStrategy string `json:"chunkStrategy"`
			Status        string `json:"status"`
		} `json:"data"`
	}
	if err := json.NewDecoder(detailResponse.Body).Decode(&detailBody); err != nil {
		t.Fatalf("decode doc detail failed: %v", err)
	}
	if detailBody.Code != "0" {
		t.Fatalf("expected doc detail code 0, got %s", detailBody.Code)
	}

	updatePayload, _ := json.Marshal(map[string]any{
		"docName":         "运维手册-更新",
		"processMode":     "chunk",
		"chunkStrategy":   "fixed_size",
		"chunkConfig":     "{\"maxTokens\":256,\"overlap\":32}",
		"scheduleEnabled": 1,
		"scheduleCron":    "0 0 * * *",
	})
	updateRequest, err := http.NewRequest(
		http.MethodPut,
		fmt.Sprintf("http://127.0.0.1:%d/knowledge-base/docs/%s", server.GetListenedPort(), docID),
		bytes.NewReader(updatePayload),
	)
	if err != nil {
		t.Fatalf("create doc update request failed: %v", err)
	}
	updateRequest.Header.Set("Authorization", token)
	updateRequest.Header.Set("Content-Type", "application/json")

	updateResponse, err := http.DefaultClient.Do(updateRequest)
	if err != nil {
		t.Fatalf("request doc update failed: %v", err)
	}
	defer updateResponse.Body.Close()

	var updateBody struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(updateResponse.Body).Decode(&updateBody); err != nil {
		t.Fatalf("decode doc update failed: %v", err)
	}
	if updateBody.Code != "0" {
		t.Fatalf("expected doc update code 0, got %s", updateBody.Code)
	}

	chunkRequest, err := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("http://127.0.0.1:%d/knowledge-base/docs/%s/chunk", server.GetListenedPort(), docID),
		nil,
	)
	if err != nil {
		t.Fatalf("create start chunk request failed: %v", err)
	}
	chunkRequest.Header.Set("Authorization", token)

	chunkResponse, err := http.DefaultClient.Do(chunkRequest)
	if err != nil {
		t.Fatalf("request start chunk failed: %v", err)
	}
	defer chunkResponse.Body.Close()

	var chunkBody struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(chunkResponse.Body).Decode(&chunkBody); err != nil {
		t.Fatalf("decode start chunk failed: %v", err)
	}
	if chunkBody.Code != "0" {
		t.Fatalf("expected start chunk code 0, got %s", chunkBody.Code)
	}

	enableRequest, err := http.NewRequest(
		http.MethodPatch,
		fmt.Sprintf("http://127.0.0.1:%d/knowledge-base/docs/%s/enable?value=false", server.GetListenedPort(), docID),
		nil,
	)
	if err != nil {
		t.Fatalf("create enable request failed: %v", err)
	}
	enableRequest.Header.Set("Authorization", token)

	enableResponse, err := http.DefaultClient.Do(enableRequest)
	if err != nil {
		t.Fatalf("request enable doc failed: %v", err)
	}
	defer enableResponse.Body.Close()

	var enableBody struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(enableResponse.Body).Decode(&enableBody); err != nil {
		t.Fatalf("decode enable doc failed: %v", err)
	}
	if enableBody.Code != "0" {
		t.Fatalf("expected enable doc code 0, got %s", enableBody.Code)
	}

	logRequest, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("http://127.0.0.1:%d/knowledge-base/docs/%s/chunk-logs?current=1&size=10", server.GetListenedPort(), docID),
		nil,
	)
	if err != nil {
		t.Fatalf("create chunk logs request failed: %v", err)
	}
	logRequest.Header.Set("Authorization", token)

	logResponse, err := http.DefaultClient.Do(logRequest)
	if err != nil {
		t.Fatalf("request chunk logs failed: %v", err)
	}
	defer logResponse.Body.Close()

	var logBody struct {
		Code string `json:"code"`
		Data struct {
			Records []struct {
				ID         string `json:"id"`
				DocID      string `json:"docId"`
				Status     string `json:"status"`
				ChunkCount int    `json:"chunkCount"`
			} `json:"records"`
			Total int `json:"total"`
		} `json:"data"`
	}
	if err := json.NewDecoder(logResponse.Body).Decode(&logBody); err != nil {
		t.Fatalf("decode chunk logs failed: %v", err)
	}
	if logBody.Code != "0" {
		t.Fatalf("expected chunk logs code 0, got %s", logBody.Code)
	}
	if logBody.Data.Total == 0 || len(logBody.Data.Records) == 0 {
		t.Fatal("expected chunk log records")
	}

	chunksRequest, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("http://127.0.0.1:%d/knowledge-base/docs/%s/chunks?current=1&size=20", server.GetListenedPort(), docID),
		nil,
	)
	if err != nil {
		t.Fatalf("create chunks page request failed: %v", err)
	}
	chunksRequest.Header.Set("Authorization", token)

	chunksResponse, err := http.DefaultClient.Do(chunksRequest)
	if err != nil {
		t.Fatalf("request chunks page failed: %v", err)
	}
	defer chunksResponse.Body.Close()

	var chunksBody struct {
		Code string `json:"code"`
		Data struct {
			Records []struct {
				ID         string `json:"id"`
				DocID      string `json:"docId"`
				Content    string `json:"content"`
				Enabled    int    `json:"enabled"`
				ChunkIndex int    `json:"chunkIndex"`
			} `json:"records"`
			Total int `json:"total"`
		} `json:"data"`
	}
	if err := json.NewDecoder(chunksResponse.Body).Decode(&chunksBody); err != nil {
		t.Fatalf("decode chunks page failed: %v", err)
	}
	if chunksBody.Code != "0" {
		t.Fatalf("expected chunks page code 0, got %s", chunksBody.Code)
	}
	if chunksBody.Data.Total == 0 || len(chunksBody.Data.Records) == 0 {
		t.Fatal("expected chunk records after start chunk")
	}
	combinedChunkContent := ""
	for _, chunk := range chunksBody.Data.Records {
		combinedChunkContent += chunk.Content
	}
	if !strings.Contains(combinedChunkContent, "这是测试文档内容") {
		t.Fatalf("expected chunk content from uploaded file, got %s", combinedChunkContent)
	}
	seedChunkID := chunksBody.Data.Records[0].ID

	createChunkPayload, _ := json.Marshal(map[string]any{
		"content": "这是手工新增的知识块内容",
		"index":   99,
	})
	createChunkRequest, err := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("http://127.0.0.1:%d/knowledge-base/docs/%s/chunks", server.GetListenedPort(), docID),
		bytes.NewReader(createChunkPayload),
	)
	if err != nil {
		t.Fatalf("create chunk create request failed: %v", err)
	}
	createChunkRequest.Header.Set("Authorization", token)
	createChunkRequest.Header.Set("Content-Type", "application/json")

	createChunkResponse, err := http.DefaultClient.Do(createChunkRequest)
	if err != nil {
		t.Fatalf("request chunk create failed: %v", err)
	}
	defer createChunkResponse.Body.Close()

	var createChunkBody struct {
		Code string `json:"code"`
		Data struct {
			ID      string `json:"id"`
			Content string `json:"content"`
		} `json:"data"`
	}
	if err := json.NewDecoder(createChunkResponse.Body).Decode(&createChunkBody); err != nil {
		t.Fatalf("decode chunk create response failed: %v", err)
	}
	if createChunkBody.Code != "0" {
		t.Fatalf("expected chunk create code 0, got %s", createChunkBody.Code)
	}
	if createChunkBody.Data.ID == "" {
		t.Fatal("expected created chunk id")
	}

	updateChunkPayload, _ := json.Marshal(map[string]any{
		"content": "这是更新后的知识块内容",
	})
	updateChunkRequest, err := http.NewRequest(
		http.MethodPut,
		fmt.Sprintf("http://127.0.0.1:%d/knowledge-base/docs/%s/chunks/%s", server.GetListenedPort(), docID, createChunkBody.Data.ID),
		bytes.NewReader(updateChunkPayload),
	)
	if err != nil {
		t.Fatalf("create chunk update request failed: %v", err)
	}
	updateChunkRequest.Header.Set("Authorization", token)
	updateChunkRequest.Header.Set("Content-Type", "application/json")

	updateChunkResponse, err := http.DefaultClient.Do(updateChunkRequest)
	if err != nil {
		t.Fatalf("request chunk update failed: %v", err)
	}
	defer updateChunkResponse.Body.Close()

	var updateChunkBody struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(updateChunkResponse.Body).Decode(&updateChunkBody); err != nil {
		t.Fatalf("decode chunk update failed: %v", err)
	}
	if updateChunkBody.Code != "0" {
		t.Fatalf("expected chunk update code 0, got %s", updateChunkBody.Code)
	}

	toggleChunkRequest, err := http.NewRequest(
		http.MethodPatch,
		fmt.Sprintf("http://127.0.0.1:%d/knowledge-base/docs/%s/chunks/%s/enable?value=false", server.GetListenedPort(), docID, seedChunkID),
		nil,
	)
	if err != nil {
		t.Fatalf("create chunk toggle request failed: %v", err)
	}
	toggleChunkRequest.Header.Set("Authorization", token)

	toggleChunkResponse, err := http.DefaultClient.Do(toggleChunkRequest)
	if err != nil {
		t.Fatalf("request chunk toggle failed: %v", err)
	}
	defer toggleChunkResponse.Body.Close()

	var toggleChunkBody struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(toggleChunkResponse.Body).Decode(&toggleChunkBody); err != nil {
		t.Fatalf("decode chunk toggle failed: %v", err)
	}
	if toggleChunkBody.Code != "0" {
		t.Fatalf("expected chunk toggle code 0, got %s", toggleChunkBody.Code)
	}

	batchTogglePayload, _ := json.Marshal(map[string]any{
		"chunkIds": []string{seedChunkID, createChunkBody.Data.ID},
	})
	batchToggleRequest, err := http.NewRequest(
		http.MethodPatch,
		fmt.Sprintf("http://127.0.0.1:%d/knowledge-base/docs/%s/chunks/batch-enable?value=true", server.GetListenedPort(), docID),
		bytes.NewReader(batchTogglePayload),
	)
	if err != nil {
		t.Fatalf("create chunk batch toggle request failed: %v", err)
	}
	batchToggleRequest.Header.Set("Authorization", token)
	batchToggleRequest.Header.Set("Content-Type", "application/json")

	batchToggleResponse, err := http.DefaultClient.Do(batchToggleRequest)
	if err != nil {
		t.Fatalf("request chunk batch toggle failed: %v", err)
	}
	defer batchToggleResponse.Body.Close()

	var batchToggleBody struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(batchToggleResponse.Body).Decode(&batchToggleBody); err != nil {
		t.Fatalf("decode chunk batch toggle failed: %v", err)
	}
	if batchToggleBody.Code != "0" {
		t.Fatalf("expected chunk batch toggle code 0, got %s", batchToggleBody.Code)
	}

	deleteChunkRequest, err := http.NewRequest(
		http.MethodDelete,
		fmt.Sprintf("http://127.0.0.1:%d/knowledge-base/docs/%s/chunks/%s", server.GetListenedPort(), docID, createChunkBody.Data.ID),
		nil,
	)
	if err != nil {
		t.Fatalf("create chunk delete request failed: %v", err)
	}
	deleteChunkRequest.Header.Set("Authorization", token)

	deleteChunkResponse, err := http.DefaultClient.Do(deleteChunkRequest)
	if err != nil {
		t.Fatalf("request chunk delete failed: %v", err)
	}
	defer deleteChunkResponse.Body.Close()

	var deleteChunkBody struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(deleteChunkResponse.Body).Decode(&deleteChunkBody); err != nil {
		t.Fatalf("decode chunk delete failed: %v", err)
	}
	if deleteChunkBody.Code != "0" {
		t.Fatalf("expected chunk delete code 0, got %s", deleteChunkBody.Code)
	}

	deleteDocRequest, err := http.NewRequest(
		http.MethodDelete,
		fmt.Sprintf("http://127.0.0.1:%d/knowledge-base/docs/%s", server.GetListenedPort(), docID),
		nil,
	)
	if err != nil {
		t.Fatalf("create delete doc request failed: %v", err)
	}
	deleteDocRequest.Header.Set("Authorization", token)

	deleteDocResponse, err := http.DefaultClient.Do(deleteDocRequest)
	if err != nil {
		t.Fatalf("request delete doc failed: %v", err)
	}
	defer deleteDocResponse.Body.Close()

	var deleteDocBody struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(deleteDocResponse.Body).Decode(&deleteDocBody); err != nil {
		t.Fatalf("decode delete doc failed: %v", err)
	}
	if deleteDocBody.Code != "0" {
		t.Fatalf("expected delete doc code 0, got %s", deleteDocBody.Code)
	}
}

func TestKnowledgeDocumentSearchMatchesBusinessSynonyms(t *testing.T) {
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
	kbID := createKnowledgeBaseForTest(t, server.GetListenedPort(), token, map[string]any{
		"name":           "财务文档库",
		"embeddingModel": "embedding-openai-large",
		"collectionName": "finance_docs",
	})
	docID := uploadNamedKnowledgeDocumentForTest(t, server.GetListenedPort(), token, kbID, "报销制度.md", "报销流程需要准备发票。")

	request, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("http://127.0.0.1:%d/knowledge-base/docs/search?keyword=%s&limit=6", server.GetListenedPort(), "报账"),
		nil,
	)
	if err != nil {
		t.Fatalf("create doc search request failed: %v", err)
	}
	request.Header.Set("Authorization", token)

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("request doc search failed: %v", err)
	}
	defer response.Body.Close()

	var body struct {
		Code string `json:"code"`
		Data []struct {
			ID      string `json:"id"`
			DocName string `json:"docName"`
			KBName  string `json:"kbName"`
		} `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode doc search failed: %v", err)
	}
	if body.Code != "0" || len(body.Data) != 1 {
		t.Fatalf("expected one synonym search result, got code=%s data=%#v", body.Code, body.Data)
	}
	if body.Data[0].ID != docID || body.Data[0].DocName != "报销制度.md" || body.Data[0].KBName != "财务文档库" {
		t.Fatalf("unexpected synonym search result %#v", body.Data[0])
	}
}

func TestKnowledgeDataPersistsThroughConfiguredDatabase(t *testing.T) {
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
	kbID := createKnowledgeBaseForTest(t, server.GetListenedPort(), token, map[string]any{
		"name":           "SQL 产品文档库",
		"embeddingModel": "embedding-openai-large",
		"collectionName": "sqlproductdocs",
	})
	docID := uploadKnowledgeDocumentForTest(t, server.GetListenedPort(), token, kbID)
	chunkRequest, err := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("http://127.0.0.1:%d/knowledge-base/docs/%s/chunk", server.GetListenedPort(), docID),
		nil,
	)
	if err != nil {
		t.Fatalf("create chunk request failed: %v", err)
	}
	chunkRequest.Header.Set("Authorization", token)
	chunkResponse, err := http.DefaultClient.Do(chunkRequest)
	if err != nil {
		t.Fatalf("request chunk failed: %v", err)
	}
	chunkResponse.Body.Close()
	server.Shutdown()

	recreated := newServer(cfg, guid.S())
	recreated.SetAddr("127.0.0.1:0")
	if err := recreated.Start(); err != nil {
		t.Fatalf("restart server failed: %v", err)
	}
	defer recreated.Shutdown()
	time.Sleep(100 * time.Millisecond)

	recreatedToken := loginAndGetToken(t, recreated.GetListenedPort())
	request, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("http://127.0.0.1:%d/knowledge-base/%s", recreated.GetListenedPort(), kbID),
		nil,
	)
	if err != nil {
		t.Fatalf("create knowledge-base detail request failed: %v", err)
	}
	request.Header.Set("Authorization", recreatedToken)

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("request recreated knowledge-base failed: %v", err)
	}
	defer response.Body.Close()

	var body struct {
		Code string `json:"code"`
		Data struct {
			ID            string `json:"id"`
			Name          string `json:"name"`
			DocumentCount int    `json:"documentCount"`
		} `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode recreated knowledge-base failed: %v", err)
	}
	if body.Code != "0" {
		t.Fatalf("expected code 0, got %s", body.Code)
	}
	if body.Data.ID != kbID || body.Data.Name != "SQL 产品文档库" {
		t.Fatalf("unexpected recreated knowledge-base %#v", body.Data)
	}
	if body.Data.DocumentCount != 1 {
		t.Fatalf("expected 1 document after restart, got %d", body.Data.DocumentCount)
	}
}

func createKnowledgeBaseForTest(t *testing.T, port int, token string, payload map[string]any) string {
	t.Helper()

	body, _ := json.Marshal(payload)
	request, err := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("http://127.0.0.1:%d/knowledge-base", port),
		bytes.NewReader(body),
	)
	if err != nil {
		t.Fatalf("create knowledge-base request failed: %v", err)
	}
	request.Header.Set("Authorization", token)
	request.Header.Set("Content-Type", "application/json")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("request create knowledge-base failed: %v", err)
	}
	defer response.Body.Close()

	var responseBody struct {
		Code string `json:"code"`
		Data string `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&responseBody); err != nil {
		t.Fatalf("decode create knowledge-base response failed: %v", err)
	}
	if responseBody.Code != "0" {
		t.Fatalf("expected create knowledge-base code 0, got %s", responseBody.Code)
	}
	if responseBody.Data == "" {
		t.Fatal("expected created knowledge-base id")
	}
	return responseBody.Data
}

func uploadKnowledgeDocumentForTest(t *testing.T, port int, token, kbID string) string {
	t.Helper()

	return uploadNamedKnowledgeDocumentForTest(t, port, token, kbID, "运维手册.md", "# 运维手册\n\n这是测试文档内容。")
}

func uploadNamedKnowledgeDocumentForTest(t *testing.T, port int, token, kbID, fileName, content string) string {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	_ = writer.WriteField("sourceType", "file")
	_ = writer.WriteField("processMode", "chunk")
	_ = writer.WriteField("chunkStrategy", "structure_aware")
	_ = writer.WriteField("chunkConfig", `{"maxTokens":512,"overlap":64}`)
	fileWriter, err := writer.CreateFormFile("file", fileName)
	if err != nil {
		t.Fatalf("create multipart file failed: %v", err)
	}
	if _, err := fileWriter.Write([]byte(content)); err != nil {
		t.Fatalf("write multipart file failed: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer failed: %v", err)
	}

	request, err := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("http://127.0.0.1:%d/knowledge-base/%s/docs/upload", port, kbID),
		&body,
	)
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
	if responseBody.Code != "0" {
		t.Fatalf("expected upload document code 0, got %s", responseBody.Code)
	}
	if responseBody.Data.ID == "" {
		t.Fatal("expected uploaded document id")
	}
	return responseBody.Data.ID
}
