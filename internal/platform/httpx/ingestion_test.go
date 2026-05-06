package httpx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	appconfig "github.com/AmazingCYJ/AgentRAG/internal/platform/config"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/util/guid"
	_ "modernc.org/sqlite"
)

func TestIngestionPipelineAndTaskLifecycle(t *testing.T) {
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

	pipelinePayload, _ := json.Marshal(map[string]any{
		"name":        "默认采集流水线",
		"description": "用于导入知识库文档",
		"nodes": []map[string]any{
			{
				"nodeId":     "fetcher",
				"nodeType":   "fetcher",
				"settings":   map[string]any{"timeoutMs": 15000},
				"nextNodeId": "parser",
			},
			{
				"nodeId":   "parser",
				"nodeType": "parser",
				"settings": map[string]any{"format": "markdown"},
			},
		},
	})
	createPipelineRequest, err := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("http://127.0.0.1:%d/ingestion/pipelines", server.GetListenedPort()),
		bytes.NewReader(pipelinePayload),
	)
	if err != nil {
		t.Fatalf("create pipeline request failed: %v", err)
	}
	createPipelineRequest.Header.Set("Authorization", token)
	createPipelineRequest.Header.Set("Content-Type", "application/json")

	createPipelineResponse, err := http.DefaultClient.Do(createPipelineRequest)
	if err != nil {
		t.Fatalf("request create pipeline failed: %v", err)
	}
	defer createPipelineResponse.Body.Close()

	var createPipelineBody struct {
		Code string `json:"code"`
		Data struct {
			ID    string `json:"id"`
			Name  string `json:"name"`
			Nodes []struct {
				ID       int    `json:"id"`
				NodeID   string `json:"nodeId"`
				NodeType string `json:"nodeType"`
			} `json:"nodes"`
		} `json:"data"`
	}
	if err := json.NewDecoder(createPipelineResponse.Body).Decode(&createPipelineBody); err != nil {
		t.Fatalf("decode create pipeline response failed: %v", err)
	}
	if createPipelineBody.Code != "0" {
		t.Fatalf("expected create pipeline code 0, got %s", createPipelineBody.Code)
	}
	if createPipelineBody.Data.ID == "" {
		t.Fatal("expected pipeline id")
	}
	if len(createPipelineBody.Data.Nodes) != 2 {
		t.Fatalf("expected 2 pipeline nodes, got %d", len(createPipelineBody.Data.Nodes))
	}
	pipelineID := createPipelineBody.Data.ID

	pagePipelineRequest, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("http://127.0.0.1:%d/ingestion/pipelines?pageNo=1&pageSize=10&keyword=%s", server.GetListenedPort(), "默认"),
		nil,
	)
	if err != nil {
		t.Fatalf("create page pipeline request failed: %v", err)
	}
	pagePipelineRequest.Header.Set("Authorization", token)

	pagePipelineResponse, err := http.DefaultClient.Do(pagePipelineRequest)
	if err != nil {
		t.Fatalf("request page pipeline failed: %v", err)
	}
	defer pagePipelineResponse.Body.Close()

	var pagePipelineBody struct {
		Code string `json:"code"`
		Data struct {
			Records []struct {
				ID          string `json:"id"`
				Name        string `json:"name"`
				Description string `json:"description"`
			} `json:"records"`
			Total int `json:"total"`
		} `json:"data"`
	}
	if err := json.NewDecoder(pagePipelineResponse.Body).Decode(&pagePipelineBody); err != nil {
		t.Fatalf("decode page pipeline response failed: %v", err)
	}
	if pagePipelineBody.Code != "0" {
		t.Fatalf("expected page pipeline code 0, got %s", pagePipelineBody.Code)
	}
	if pagePipelineBody.Data.Total == 0 || len(pagePipelineBody.Data.Records) == 0 {
		t.Fatal("expected pipeline page records")
	}

	getPipelineRequest, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("http://127.0.0.1:%d/ingestion/pipelines/%s", server.GetListenedPort(), pipelineID),
		nil,
	)
	if err != nil {
		t.Fatalf("create get pipeline request failed: %v", err)
	}
	getPipelineRequest.Header.Set("Authorization", token)

	getPipelineResponse, err := http.DefaultClient.Do(getPipelineRequest)
	if err != nil {
		t.Fatalf("request get pipeline failed: %v", err)
	}
	defer getPipelineResponse.Body.Close()

	var getPipelineBody struct {
		Code string `json:"code"`
		Data struct {
			ID    string `json:"id"`
			Name  string `json:"name"`
			Nodes []struct {
				NodeID string `json:"nodeId"`
			} `json:"nodes"`
		} `json:"data"`
	}
	if err := json.NewDecoder(getPipelineResponse.Body).Decode(&getPipelineBody); err != nil {
		t.Fatalf("decode get pipeline response failed: %v", err)
	}
	if getPipelineBody.Code != "0" {
		t.Fatalf("expected get pipeline code 0, got %s", getPipelineBody.Code)
	}
	if getPipelineBody.Data.ID != pipelineID {
		t.Fatalf("expected pipeline id %s, got %s", pipelineID, getPipelineBody.Data.ID)
	}

	updatePipelinePayload, _ := json.Marshal(map[string]any{
		"name":        "默认采集流水线-更新",
		"description": "更新后的描述",
		"nodes": []map[string]any{
			{
				"nodeId":     "fetcher",
				"nodeType":   "fetcher",
				"settings":   map[string]any{"timeoutMs": 20000},
				"nextNodeId": "chunker",
			},
			{
				"nodeId":   "chunker",
				"nodeType": "chunker",
				"settings": map[string]any{"chunkSize": 512},
			},
		},
	})
	updatePipelineRequest, err := http.NewRequest(
		http.MethodPut,
		fmt.Sprintf("http://127.0.0.1:%d/ingestion/pipelines/%s", server.GetListenedPort(), pipelineID),
		bytes.NewReader(updatePipelinePayload),
	)
	if err != nil {
		t.Fatalf("create update pipeline request failed: %v", err)
	}
	updatePipelineRequest.Header.Set("Authorization", token)
	updatePipelineRequest.Header.Set("Content-Type", "application/json")

	updatePipelineResponse, err := http.DefaultClient.Do(updatePipelineRequest)
	if err != nil {
		t.Fatalf("request update pipeline failed: %v", err)
	}
	defer updatePipelineResponse.Body.Close()

	var updatePipelineBody struct {
		Code string `json:"code"`
		Data struct {
			Name string `json:"name"`
		} `json:"data"`
	}
	if err := json.NewDecoder(updatePipelineResponse.Body).Decode(&updatePipelineBody); err != nil {
		t.Fatalf("decode update pipeline response failed: %v", err)
	}
	if updatePipelineBody.Code != "0" {
		t.Fatalf("expected update pipeline code 0, got %s", updatePipelineBody.Code)
	}

	createTaskPayload, _ := json.Marshal(map[string]any{
		"pipelineId": pipelineID,
		"source": map[string]any{
			"type":     "url",
			"location": "https://example.com/docs/ops",
			"fileName": "ops.md",
			"credentials": map[string]any{
				"token": "demo-token",
			},
		},
		"metadata": map[string]any{
			"scene": "knowledge",
		},
		"vectorSpaceId": map[string]any{
			"kbId": "kb_demo",
		},
	})
	createTaskRequest, err := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("http://127.0.0.1:%d/ingestion/tasks", server.GetListenedPort()),
		bytes.NewReader(createTaskPayload),
	)
	if err != nil {
		t.Fatalf("create task request failed: %v", err)
	}
	createTaskRequest.Header.Set("Authorization", token)
	createTaskRequest.Header.Set("Content-Type", "application/json")

	createTaskResponse, err := http.DefaultClient.Do(createTaskRequest)
	if err != nil {
		t.Fatalf("request create task failed: %v", err)
	}
	defer createTaskResponse.Body.Close()

	var createTaskBody struct {
		Code string `json:"code"`
		Data struct {
			TaskID     string `json:"taskId"`
			PipelineID string `json:"pipelineId"`
			Status     string `json:"status"`
			ChunkCount int    `json:"chunkCount"`
		} `json:"data"`
	}
	if err := json.NewDecoder(createTaskResponse.Body).Decode(&createTaskBody); err != nil {
		t.Fatalf("decode create task response failed: %v", err)
	}
	if createTaskBody.Code != "0" {
		t.Fatalf("expected create task code 0, got %s", createTaskBody.Code)
	}
	if createTaskBody.Data.TaskID == "" {
		t.Fatal("expected task id")
	}
	taskID := createTaskBody.Data.TaskID

	pageTaskRequest, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("http://127.0.0.1:%d/ingestion/tasks?pageNo=1&pageSize=10&status=%s", server.GetListenedPort(), "success"),
		nil,
	)
	if err != nil {
		t.Fatalf("create page task request failed: %v", err)
	}
	pageTaskRequest.Header.Set("Authorization", token)

	pageTaskResponse, err := http.DefaultClient.Do(pageTaskRequest)
	if err != nil {
		t.Fatalf("request page task failed: %v", err)
	}
	defer pageTaskResponse.Body.Close()

	var pageTaskBody struct {
		Code string `json:"code"`
		Data struct {
			Records []struct {
				ID         string `json:"id"`
				PipelineID string `json:"pipelineId"`
				SourceType string `json:"sourceType"`
				Status     string `json:"status"`
				ChunkCount int    `json:"chunkCount"`
			} `json:"records"`
			Total int `json:"total"`
		} `json:"data"`
	}
	if err := json.NewDecoder(pageTaskResponse.Body).Decode(&pageTaskBody); err != nil {
		t.Fatalf("decode page task response failed: %v", err)
	}
	if pageTaskBody.Code != "0" {
		t.Fatalf("expected page task code 0, got %s", pageTaskBody.Code)
	}
	if pageTaskBody.Data.Total == 0 || len(pageTaskBody.Data.Records) == 0 {
		t.Fatal("expected task page records")
	}

	getTaskRequest, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("http://127.0.0.1:%d/ingestion/tasks/%s", server.GetListenedPort(), taskID),
		nil,
	)
	if err != nil {
		t.Fatalf("create get task request failed: %v", err)
	}
	getTaskRequest.Header.Set("Authorization", token)

	getTaskResponse, err := http.DefaultClient.Do(getTaskRequest)
	if err != nil {
		t.Fatalf("request get task failed: %v", err)
	}
	defer getTaskResponse.Body.Close()

	var getTaskBody struct {
		Code string `json:"code"`
		Data struct {
			ID         string `json:"id"`
			PipelineID string `json:"pipelineId"`
			Status     string `json:"status"`
			Logs       []struct {
				NodeID   string `json:"nodeId"`
				NodeType string `json:"nodeType"`
				Success  bool   `json:"success"`
			} `json:"logs"`
		} `json:"data"`
	}
	if err := json.NewDecoder(getTaskResponse.Body).Decode(&getTaskBody); err != nil {
		t.Fatalf("decode get task response failed: %v", err)
	}
	if getTaskBody.Code != "0" {
		t.Fatalf("expected get task code 0, got %s", getTaskBody.Code)
	}
	if getTaskBody.Data.ID != taskID {
		t.Fatalf("expected task id %s, got %s", taskID, getTaskBody.Data.ID)
	}
	if len(getTaskBody.Data.Logs) == 0 {
		t.Fatal("expected task logs")
	}

	nodesRequest, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("http://127.0.0.1:%d/ingestion/tasks/%s/nodes", server.GetListenedPort(), taskID),
		nil,
	)
	if err != nil {
		t.Fatalf("create task nodes request failed: %v", err)
	}
	nodesRequest.Header.Set("Authorization", token)

	nodesResponse, err := http.DefaultClient.Do(nodesRequest)
	if err != nil {
		t.Fatalf("request task nodes failed: %v", err)
	}
	defer nodesResponse.Body.Close()

	var nodesBody struct {
		Code string `json:"code"`
		Data []struct {
			ID        string `json:"id"`
			TaskID    string `json:"taskId"`
			NodeID    string `json:"nodeId"`
			NodeType  string `json:"nodeType"`
			NodeOrder int    `json:"nodeOrder"`
			Status    string `json:"status"`
		} `json:"data"`
	}
	if err := json.NewDecoder(nodesResponse.Body).Decode(&nodesBody); err != nil {
		t.Fatalf("decode task nodes response failed: %v", err)
	}
	if nodesBody.Code != "0" {
		t.Fatalf("expected task nodes code 0, got %s", nodesBody.Code)
	}
	if len(nodesBody.Data) == 0 {
		t.Fatal("expected task node records")
	}

	uploadTaskID := uploadIngestionTaskForTest(t, server.GetListenedPort(), token, pipelineID)
	if uploadTaskID == "" {
		t.Fatal("expected uploaded task id")
	}
	uploadedTask := getIngestionTaskForTest(t, server.GetListenedPort(), token, uploadTaskID)
	if uploadedTask.SourceFileName != "ingestion-demo.md" {
		t.Fatalf("expected uploaded source file name ingestion-demo.md, got %s", uploadedTask.SourceFileName)
	}
	if uploadedTask.Metadata["fileSize"] == nil {
		t.Fatalf("expected uploaded task metadata to contain fileSize, got %#v", uploadedTask.Metadata)
	}
	preview, _ := uploadedTask.Metadata["contentPreview"].(string)
	if preview != "# Upload Task\n\nThis is a demo." {
		t.Fatalf("expected uploaded content preview, got %#v", uploadedTask.Metadata)
	}
	mimeType, _ := uploadedTask.Metadata["mimeType"].(string)
	if mimeType == "" {
		t.Fatalf("expected uploaded task metadata to contain mimeType, got %#v", uploadedTask.Metadata)
	}

	deletePipelineRequest, err := http.NewRequest(
		http.MethodDelete,
		fmt.Sprintf("http://127.0.0.1:%d/ingestion/pipelines/%s", server.GetListenedPort(), pipelineID),
		nil,
	)
	if err != nil {
		t.Fatalf("create delete pipeline request failed: %v", err)
	}
	deletePipelineRequest.Header.Set("Authorization", token)

	deletePipelineResponse, err := http.DefaultClient.Do(deletePipelineRequest)
	if err != nil {
		t.Fatalf("request delete pipeline failed: %v", err)
	}
	defer deletePipelineResponse.Body.Close()

	var deletePipelineBody struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(deletePipelineResponse.Body).Decode(&deletePipelineBody); err != nil {
		t.Fatalf("decode delete pipeline response failed: %v", err)
	}
	if deletePipelineBody.Code != "0" {
		t.Fatalf("expected delete pipeline code 0, got %s", deletePipelineBody.Code)
	}
}

func uploadIngestionTaskForTest(t *testing.T, port int, token, pipelineID string) string {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	fileWriter, err := writer.CreateFormFile("file", "ingestion-demo.md")
	if err != nil {
		t.Fatalf("create upload task file failed: %v", err)
	}
	if _, err := fileWriter.Write([]byte("# Upload Task\n\nThis is a demo.")); err != nil {
		t.Fatalf("write upload task file failed: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close upload task multipart failed: %v", err)
	}

	request, err := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("http://127.0.0.1:%d/ingestion/tasks/upload?pipelineId=%s", port, pipelineID),
		&body,
	)
	if err != nil {
		t.Fatalf("create upload task request failed: %v", err)
	}
	request.Header.Set("Authorization", token)
	request.Header.Set("Content-Type", writer.FormDataContentType())

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("request upload task failed: %v", err)
	}
	defer response.Body.Close()

	var responseBody struct {
		Code string `json:"code"`
		Data struct {
			TaskID string `json:"taskId"`
		} `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&responseBody); err != nil {
		t.Fatalf("decode upload task response failed: %v", err)
	}
	if responseBody.Code != "0" {
		t.Fatalf("expected upload task code 0, got %s", responseBody.Code)
	}
	return responseBody.Data.TaskID
}

func TestIngestionRecordsPersistAcrossServerRestartWithDatabase(t *testing.T) {
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
			DSN:    filepath.Join(t.TempDir(), "agentrag.db"),
		},
	}

	firstServer := startIngestionTestServer(t, cfg)
	firstToken := loginAndGetToken(t, firstServer.GetListenedPort())
	pipelineID := createIngestionPipelineForTest(t, firstServer.GetListenedPort(), firstToken, "SQL 重启流水线")
	taskID := createIngestionTaskForTest(t, firstServer.GetListenedPort(), firstToken, pipelineID)
	firstServer.Shutdown()

	secondServer := startIngestionTestServer(t, cfg)
	defer secondServer.Shutdown()
	secondToken := loginAndGetToken(t, secondServer.GetListenedPort())

	pipeline := getIngestionPipelineForTest(t, secondServer.GetListenedPort(), secondToken, pipelineID)
	if pipeline.Name != "SQL 重启流水线" || len(pipeline.Nodes) != 2 {
		t.Fatalf("unexpected persisted pipeline %#v", pipeline)
	}
	task := getIngestionTaskForTest(t, secondServer.GetListenedPort(), secondToken, taskID)
	if task.PipelineID != pipelineID || task.Status != "success" {
		t.Fatalf("unexpected persisted task %#v", task)
	}
	nodes := getIngestionTaskNodesForTest(t, secondServer.GetListenedPort(), secondToken, taskID)
	if len(nodes) != 2 || nodes[0].NodeID != "fetcher" {
		t.Fatalf("unexpected persisted task nodes %#v", nodes)
	}
}

func startIngestionTestServer(t *testing.T, cfg *appconfig.Config) *ghttp.Server {
	t.Helper()

	server := newServer(cfg, guid.S())
	server.SetAddr("127.0.0.1:0")
	if err := server.Start(); err != nil {
		t.Fatalf("start server failed: %v", err)
	}
	time.Sleep(100 * time.Millisecond)
	return server
}

type ingestionPipelineTestResponse struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Nodes []struct {
		NodeID string `json:"nodeId"`
	} `json:"nodes"`
}

type ingestionTaskTestResponse struct {
	ID             string         `json:"id"`
	PipelineID     string         `json:"pipelineId"`
	SourceFileName string         `json:"sourceFileName"`
	Status         string         `json:"status"`
	Metadata       map[string]any `json:"metadata"`
}

type ingestionTaskNodeTestResponse struct {
	ID     string `json:"id"`
	NodeID string `json:"nodeId"`
}

func createIngestionPipelineForTest(t *testing.T, port int, token, name string) string {
	t.Helper()

	payload, _ := json.Marshal(map[string]any{
		"name":        name,
		"description": "数据库持久化测试",
		"nodes": []map[string]any{
			{
				"nodeId":     "fetcher",
				"nodeType":   "fetcher",
				"settings":   map[string]any{"timeoutMs": 15000},
				"nextNodeId": "parser",
			},
			{
				"nodeId":   "parser",
				"nodeType": "parser",
				"settings": map[string]any{"format": "markdown"},
			},
		},
	})
	request, err := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("http://127.0.0.1:%d/ingestion/pipelines", port),
		bytes.NewReader(payload),
	)
	if err != nil {
		t.Fatalf("create pipeline request failed: %v", err)
	}
	request.Header.Set("Authorization", token)
	request.Header.Set("Content-Type", "application/json")

	var body struct {
		Code string                        `json:"code"`
		Data ingestionPipelineTestResponse `json:"data"`
	}
	doJSONRequestForTest(t, request, &body)
	if body.Code != "0" || body.Data.ID == "" {
		t.Fatalf("unexpected create pipeline response %#v", body)
	}
	return body.Data.ID
}

func createIngestionTaskForTest(t *testing.T, port int, token, pipelineID string) string {
	t.Helper()

	payload, _ := json.Marshal(map[string]any{
		"pipelineId": pipelineID,
		"source": map[string]any{
			"type":     "url",
			"location": "https://example.com/restart",
			"fileName": "restart.md",
		},
		"metadata": map[string]any{
			"scene": "restart",
		},
	})
	request, err := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("http://127.0.0.1:%d/ingestion/tasks", port),
		bytes.NewReader(payload),
	)
	if err != nil {
		t.Fatalf("create task request failed: %v", err)
	}
	request.Header.Set("Authorization", token)
	request.Header.Set("Content-Type", "application/json")

	var body struct {
		Code string `json:"code"`
		Data struct {
			TaskID string `json:"taskId"`
		} `json:"data"`
	}
	doJSONRequestForTest(t, request, &body)
	if body.Code != "0" || body.Data.TaskID == "" {
		t.Fatalf("unexpected create task response %#v", body)
	}
	return body.Data.TaskID
}

func getIngestionPipelineForTest(t *testing.T, port int, token, pipelineID string) ingestionPipelineTestResponse {
	t.Helper()

	request, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("http://127.0.0.1:%d/ingestion/pipelines/%s", port, pipelineID),
		nil,
	)
	if err != nil {
		t.Fatalf("create get pipeline request failed: %v", err)
	}
	request.Header.Set("Authorization", token)

	var body struct {
		Code string                        `json:"code"`
		Data ingestionPipelineTestResponse `json:"data"`
	}
	doJSONRequestForTest(t, request, &body)
	if body.Code != "0" {
		t.Fatalf("unexpected get pipeline response %#v", body)
	}
	return body.Data
}

func getIngestionTaskForTest(t *testing.T, port int, token, taskID string) ingestionTaskTestResponse {
	t.Helper()

	request, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("http://127.0.0.1:%d/ingestion/tasks/%s", port, taskID),
		nil,
	)
	if err != nil {
		t.Fatalf("create get task request failed: %v", err)
	}
	request.Header.Set("Authorization", token)

	var body struct {
		Code string                    `json:"code"`
		Data ingestionTaskTestResponse `json:"data"`
	}
	doJSONRequestForTest(t, request, &body)
	if body.Code != "0" {
		t.Fatalf("unexpected get task response %#v", body)
	}
	return body.Data
}

func getIngestionTaskNodesForTest(t *testing.T, port int, token, taskID string) []ingestionTaskNodeTestResponse {
	t.Helper()

	request, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("http://127.0.0.1:%d/ingestion/tasks/%s/nodes", port, taskID),
		nil,
	)
	if err != nil {
		t.Fatalf("create get task nodes request failed: %v", err)
	}
	request.Header.Set("Authorization", token)

	var body struct {
		Code string                          `json:"code"`
		Data []ingestionTaskNodeTestResponse `json:"data"`
	}
	doJSONRequestForTest(t, request, &body)
	if body.Code != "0" {
		t.Fatalf("unexpected get task nodes response %#v", body)
	}
	return body.Data
}

func doJSONRequestForTest(t *testing.T, request *http.Request, target any) {
	t.Helper()

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer response.Body.Close()
	if err := json.NewDecoder(response.Body).Decode(target); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
}
