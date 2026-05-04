package httpx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
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
	ID         string                 `json:"id"`
	IntentCode string                 `json:"intentCode"`
	Name       string                 `json:"name"`
	Enabled    int                    `json:"enabled"`
	Children   []intentTreeNodeForTest `json:"children"`
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
		Code string                 `json:"code"`
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
