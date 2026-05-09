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
)

func TestIntentTreeCRUDAndBatchActions(t *testing.T) {
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

	assertIntentNodeInvalidTopKForTest(t, server.GetListenedPort(), token)

	rootID := createIntentNodeForTest(t, server.GetListenedPort(), token, map[string]any{
		"intentCode":     "biz_oa",
		"name":           "OA系统",
		"level":          0,
		"kind":           0,
		"enabled":        1,
		"sortOrder":      10,
		"description":    "办公系统",
		"examples":       []string{"如何发起请假", "如何查看审批"},
		"promptSnippet":  "优先回答OA相关问题",
		"promptTemplate": "你是OA助手",
	})
	childID := createIntentNodeForTest(t, server.GetListenedPort(), token, map[string]any{
		"intentCode":          "biz_oa_leave",
		"name":                "请假流程",
		"level":               1,
		"parentCode":          "biz_oa",
		"kind":                2,
		"enabled":             1,
		"sortOrder":           20,
		"mcpToolId":           "leave-tool",
		"paramPromptTemplate": "抽取请假参数",
	})

	tree := getIntentTreeForTest(t, server.GetListenedPort(), token)
	if len(tree) == 0 {
		t.Fatal("expected non-empty intent tree")
	}
	if tree[0].IntentCode != "biz_oa" {
		t.Fatalf("expected root intentCode biz_oa, got %s", tree[0].IntentCode)
	}
	if len(tree[0].Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(tree[0].Children))
	}
	if tree[0].Children[0].IntentCode != "biz_oa_leave" {
		t.Fatalf("expected child intentCode biz_oa_leave, got %s", tree[0].Children[0].IntentCode)
	}

	updatePayload, _ := json.Marshal(map[string]any{
		"name":          "请假审批",
		"level":         1,
		"parentCode":    "biz_oa",
		"kind":          2,
		"topK":          4,
		"enabled":       1,
		"sortOrder":     30,
		"mcpToolId":     "leave-tool-v2",
		"description":   "处理请假审批",
		"examples":      []string{"请假多久需要审批"},
		"promptSnippet": "请先识别请假场景",
	})
	updateRequest, err := http.NewRequest(
		http.MethodPut,
		fmt.Sprintf("http://127.0.0.1:%d/intent-tree/%s", server.GetListenedPort(), childID),
		bytes.NewReader(updatePayload),
	)
	if err != nil {
		t.Fatalf("create update request failed: %v", err)
	}
	updateRequest.Header.Set("Authorization", token)
	updateRequest.Header.Set("Content-Type", "application/json")

	updateResponse, err := http.DefaultClient.Do(updateRequest)
	if err != nil {
		t.Fatalf("request update intent node failed: %v", err)
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

	postUpdateTree := getIntentTreeForTest(t, server.GetListenedPort(), token)
	if postUpdateTree[0].Children[0].Name != "请假审批" {
		t.Fatalf("expected updated child name 请假审批, got %s", postUpdateTree[0].Children[0].Name)
	}
	if postUpdateTree[0].Children[0].TopK == nil || *postUpdateTree[0].Children[0].TopK != 4 {
		t.Fatalf("expected updated child topK 4, got %#v", postUpdateTree[0].Children[0].TopK)
	}

	assertIntentNodeInvalidUpdateTopKForTest(t, server.GetListenedPort(), token, childID)
	afterInvalidUpdateTree := getIntentTreeForTest(t, server.GetListenedPort(), token)
	if afterInvalidUpdateTree[0].Children[0].Name != "请假审批" {
		t.Fatalf("expected child name unchanged after invalid topK, got %s", afterInvalidUpdateTree[0].Children[0].Name)
	}
	if afterInvalidUpdateTree[0].Children[0].TopK == nil || *afterInvalidUpdateTree[0].Children[0].TopK != 4 {
		t.Fatalf("expected child topK unchanged after invalid update, got %#v", afterInvalidUpdateTree[0].Children[0].TopK)
	}

	runIntentBatchForTest(t, server.GetListenedPort(), token, "/intent-tree/batch/disable", []string{rootID, childID})
	disabledTree := getIntentTreeForTest(t, server.GetListenedPort(), token)
	if disabledTree[0].Enabled != 0 || disabledTree[0].Children[0].Enabled != 0 {
		t.Fatal("expected nodes to be disabled")
	}

	runIntentBatchForTest(t, server.GetListenedPort(), token, "/intent-tree/batch/enable", []string{rootID, childID})
	enabledTree := getIntentTreeForTest(t, server.GetListenedPort(), token)
	if enabledTree[0].Enabled == 0 || enabledTree[0].Children[0].Enabled == 0 {
		t.Fatal("expected nodes to be enabled")
	}

	runIntentBatchForTest(t, server.GetListenedPort(), token, "/intent-tree/batch/delete", []string{rootID})
	emptyTree := getIntentTreeForTest(t, server.GetListenedPort(), token)
	if len(emptyTree) != 0 {
		t.Fatalf("expected tree to be empty after delete, got %d roots", len(emptyTree))
	}
}

type intentTreeNodeForTest struct {
	ID                  string                  `json:"id"`
	IntentCode          string                  `json:"intentCode"`
	Name                string                  `json:"name"`
	Level               int                     `json:"level"`
	ParentCode          string                  `json:"parentCode"`
	Examples            string                  `json:"examples"`
	CollectionName      string                  `json:"collectionName"`
	MCPToolID           string                  `json:"mcpToolId"`
	TopK                *int                    `json:"topK"`
	Kind                int                     `json:"kind"`
	Enabled             int                     `json:"enabled"`
	PromptSnippet       string                  `json:"promptSnippet"`
	PromptTemplate      string                  `json:"promptTemplate"`
	ParamPromptTemplate string                  `json:"paramPromptTemplate"`
	Children            []intentTreeNodeForTest `json:"children"`
}

func assertIntentNodeInvalidTopKForTest(t *testing.T, port int, token string) {
	t.Helper()

	payload, _ := json.Marshal(map[string]any{
		"intentCode": "invalid_topk",
		"name":       "无效召回",
		"level":      0,
		"topK":       0,
		"enabled":    1,
	})
	request, err := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("http://127.0.0.1:%d/intent-tree", port),
		bytes.NewReader(payload),
	)
	if err != nil {
		t.Fatalf("create invalid topK request failed: %v", err)
	}
	request.Header.Set("Authorization", token)
	request.Header.Set("Content-Type", "application/json")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("request invalid topK failed: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected invalid topK status 400, got %d", response.StatusCode)
	}

	var body struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode invalid topK response failed: %v", err)
	}
	if body.Code != "400" || body.Message != "召回数量必须大于 0" {
		t.Fatalf("unexpected invalid topK response %#v", body)
	}
}

func assertIntentNodeInvalidUpdateTopKForTest(t *testing.T, port int, token, nodeID string) {
	t.Helper()

	payload, _ := json.Marshal(map[string]any{
		"name":       "无效更新",
		"level":      1,
		"parentCode": "biz_oa",
		"topK":       0,
		"kind":       2,
		"enabled":    1,
	})
	request, err := http.NewRequest(
		http.MethodPut,
		fmt.Sprintf("http://127.0.0.1:%d/intent-tree/%s", port, nodeID),
		bytes.NewReader(payload),
	)
	if err != nil {
		t.Fatalf("create invalid topK update request failed: %v", err)
	}
	request.Header.Set("Authorization", token)
	request.Header.Set("Content-Type", "application/json")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("request invalid topK update failed: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected invalid update topK status 400, got %d", response.StatusCode)
	}

	var body struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode invalid update topK response failed: %v", err)
	}
	if body.Code != "400" || body.Message != "召回数量必须大于 0" {
		t.Fatalf("unexpected invalid update topK response %#v", body)
	}
}

func TestIntentTreePersistsThroughConfiguredDatabase(t *testing.T) {
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
	topK := 5
	rootID := createIntentNodeForTest(t, server.GetListenedPort(), token, map[string]any{
		"kbId":           "kb_sql",
		"intentCode":     "sql_policy",
		"name":           "SQL 制度问答",
		"level":          0,
		"kind":           1,
		"enabled":        1,
		"sortOrder":      10,
		"description":    "SQL 持久化意图",
		"examples":       []string{"SQL 意图路由"},
		"topK":           topK,
		"promptSnippet":  "优先使用 SQL 节点",
		"promptTemplate": "你是 SQL 意图助手",
	})
	createIntentNodeForTest(t, server.GetListenedPort(), token, map[string]any{
		"intentCode":          "sql_ticket",
		"name":                "SQL 工单工具",
		"level":               1,
		"parentCode":          "sql_policy",
		"kind":                2,
		"enabled":             1,
		"sortOrder":           20,
		"mcpToolId":           "ticket_query",
		"examples":            []string{"查工单"},
		"paramPromptTemplate": "抽取 SQL 工单参数",
	})
	server.Shutdown()

	recreated := newServer(&cfg, guid.S())
	recreated.SetAddr("127.0.0.1:0")
	if err := recreated.Start(); err != nil {
		t.Fatalf("start recreated server failed: %v", err)
	}
	defer recreated.Shutdown()
	time.Sleep(100 * time.Millisecond)

	recreatedToken := loginAndGetToken(t, recreated.GetListenedPort())
	tree := getIntentTreeForTest(t, recreated.GetListenedPort(), recreatedToken)
	if len(tree) != 1 {
		t.Fatalf("expected 1 root after restart, got %d", len(tree))
	}
	root := tree[0]
	if root.ID != rootID || root.IntentCode != "sql_policy" || root.CollectionName != "kb_sql" {
		t.Fatalf("unexpected persisted root %#v", root)
	}
	if root.TopK == nil || *root.TopK != topK || root.Examples != `["SQL 意图路由"]` {
		t.Fatalf("unexpected persisted root retrieval fields %#v", root)
	}
	if root.PromptSnippet != "优先使用 SQL 节点" || root.PromptTemplate != "你是 SQL 意图助手" {
		t.Fatalf("unexpected persisted root prompts %#v", root)
	}
	if len(root.Children) != 1 {
		t.Fatalf("expected 1 persisted child, got %d", len(root.Children))
	}
	child := root.Children[0]
	if child.IntentCode != "sql_ticket" || child.ParentCode != "sql_policy" || child.MCPToolID != "ticket_query" {
		t.Fatalf("unexpected persisted child %#v", child)
	}
	if child.ParamPromptTemplate != "抽取 SQL 工单参数" || child.Kind != 2 {
		t.Fatalf("unexpected persisted child tool fields %#v", child)
	}
}

func createIntentNodeForTest(t *testing.T, port int, token string, payload map[string]any) string {
	t.Helper()

	body, _ := json.Marshal(payload)
	request, err := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("http://127.0.0.1:%d/intent-tree", port),
		bytes.NewReader(body),
	)
	if err != nil {
		t.Fatalf("create intent node request failed: %v", err)
	}
	request.Header.Set("Authorization", token)
	request.Header.Set("Content-Type", "application/json")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("request create intent node failed: %v", err)
	}
	defer response.Body.Close()

	var responseBody struct {
		Code string `json:"code"`
		Data string `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&responseBody); err != nil {
		t.Fatalf("decode create intent node response failed: %v", err)
	}
	if responseBody.Code != "0" {
		t.Fatalf("expected create code 0, got %s", responseBody.Code)
	}
	if responseBody.Data == "" {
		t.Fatal("expected created intent node id")
	}
	return responseBody.Data
}

func getIntentTreeForTest(t *testing.T, port int, token string) []intentTreeNodeForTest {
	t.Helper()

	request, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://127.0.0.1:%d/intent-tree/trees", port), nil)
	if err != nil {
		t.Fatalf("create intent tree request failed: %v", err)
	}
	request.Header.Set("Authorization", token)

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("request intent tree failed: %v", err)
	}
	defer response.Body.Close()

	var body struct {
		Code string                  `json:"code"`
		Data []intentTreeNodeForTest `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode intent tree response failed: %v", err)
	}
	if body.Code != "0" {
		t.Fatalf("expected tree code 0, got %s", body.Code)
	}
	return body.Data
}

func runIntentBatchForTest(t *testing.T, port int, token, path string, ids []string) {
	t.Helper()

	body, _ := json.Marshal(map[string]any{"ids": ids})
	request, err := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("http://127.0.0.1:%d%s", port, path),
		bytes.NewReader(body),
	)
	if err != nil {
		t.Fatalf("create batch request failed: %v", err)
	}
	request.Header.Set("Authorization", token)
	request.Header.Set("Content-Type", "application/json")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("request batch action failed: %v", err)
	}
	defer response.Body.Close()

	var responseBody struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(response.Body).Decode(&responseBody); err != nil {
		t.Fatalf("decode batch response failed: %v", err)
	}
	if responseBody.Code != "0" {
		t.Fatalf("expected batch code 0, got %s", responseBody.Code)
	}
}
