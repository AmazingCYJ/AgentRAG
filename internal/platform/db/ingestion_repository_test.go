package db

import (
	"database/sql"
	"testing"
	"time"

	domainingestion "github.com/AmazingCYJ/AgentRAG/internal/domain/ingestion"
	_ "modernc.org/sqlite"
)

func TestSQLIngestionRepositorySavesAndLoadsRecords(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite database failed: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	repository := NewSQLIngestionRepository(database)
	if err := repository.Bootstrap(); err != nil {
		t.Fatalf("bootstrap ingestion tables failed: %v", err)
	}

	now := time.Date(2026, 5, 5, 9, 30, 0, 0, time.UTC)
	pipeline := domainingestion.Pipeline{
		ID:          "pipe_sql",
		Name:        "SQL 采集流水线",
		Description: "用于验证 SQL 持久化",
		CreatedBy:   "admin",
		Nodes: []domainingestion.PipelineNode{
			{
				ID:         11,
				NodeID:     "fetcher",
				NodeType:   "fetcher",
				Settings:   map[string]any{"timeoutMs": float64(15000)},
				Condition:  map[string]any{"mode": "always"},
				NextNodeID: "parser",
			},
			{
				ID:       12,
				NodeID:   "parser",
				NodeType: "parser",
				Settings: map[string]any{
					"format": "markdown",
				},
			},
		},
		CreateTime: now,
		UpdateTime: now,
	}
	task := domainingestion.Task{
		ID:             "task_sql",
		PipelineID:     pipeline.ID,
		SourceType:     "url",
		SourceLocation: "https://example.com/docs",
		SourceFileName: "docs.md",
		Status:         "success",
		ChunkCount:     2,
		Logs: []domainingestion.TaskLog{
			{NodeID: "fetcher", NodeType: "fetcher", Message: "抓取完成", DurationMs: 120, Success: true},
			{NodeID: "parser", NodeType: "parser", Message: "解析完成", DurationMs: 160, Success: true},
		},
		Metadata:    map[string]any{"scene": "knowledge"},
		StartedAt:   now,
		CompletedAt: now.Add(280 * time.Millisecond),
		CreatedBy:   "admin",
		CreateTime:  now,
		UpdateTime:  now,
	}
	taskNode := domainingestion.TaskNode{
		ID:         "task_node_sql",
		TaskID:     task.ID,
		PipelineID: pipeline.ID,
		NodeID:     "fetcher",
		NodeType:   "fetcher",
		NodeOrder:  1,
		Status:     "success",
		DurationMs: 120,
		Message:    "节点执行完成",
		Output:     map[string]any{"nextNodeId": "parser"},
		CreateTime: now,
		UpdateTime: now,
	}

	if err := repository.SaveIngestionRecords(
		[]domainingestion.Pipeline{pipeline},
		[]domainingestion.Task{task},
		[]domainingestion.TaskNode{taskNode},
	); err != nil {
		t.Fatalf("save ingestion records failed: %v", err)
	}

	pipelines, tasks, taskNodes, err := repository.LoadIngestionRecords()
	if err != nil {
		t.Fatalf("load ingestion records failed: %v", err)
	}
	if len(pipelines) != 1 || pipelines[0].Name != "SQL 采集流水线" || len(pipelines[0].Nodes) != 2 {
		t.Fatalf("unexpected pipelines %#v", pipelines)
	}
	if pipelines[0].Nodes[0].Settings["timeoutMs"] != float64(15000) {
		t.Fatalf("unexpected node settings %#v", pipelines[0].Nodes[0].Settings)
	}
	if len(tasks) != 1 || tasks[0].SourceLocation != "https://example.com/docs" || len(tasks[0].Logs) != 2 {
		t.Fatalf("unexpected tasks %#v", tasks)
	}
	if tasks[0].Metadata["scene"] != "knowledge" {
		t.Fatalf("unexpected task metadata %#v", tasks[0].Metadata)
	}
	if len(taskNodes) != 1 || taskNodes[0].Output["nextNodeId"] != "parser" {
		t.Fatalf("unexpected task nodes %#v", taskNodes)
	}
}
