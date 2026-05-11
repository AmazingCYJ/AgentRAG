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

func TestSQLIngestionRepositoryReconcilesSavedRecords(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite database failed: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	repository := NewSQLIngestionRepository(database)
	if err := repository.Bootstrap(); err != nil {
		t.Fatalf("bootstrap ingestion tables failed: %v", err)
	}
	now := time.Date(2026, 5, 11, 12, 0, 0, 0, time.UTC)
	if err := repository.SaveIngestionRecords(
		[]domainingestion.Pipeline{
			{
				ID: "pipe_keep", Name: "旧流水线", Description: "旧描述", CreatedBy: "admin", CreateTime: now, UpdateTime: now,
				Nodes: []domainingestion.PipelineNode{
					{ID: 1, NodeID: "fetcher", NodeType: "fetcher", Settings: map[string]any{"timeoutMs": float64(1000)}, NextNodeID: "parser"},
					{ID: 2, NodeID: "parser", NodeType: "parser", Settings: map[string]any{"format": "markdown"}},
				},
			},
			{
				ID: "pipe_remove", Name: "删除流水线", Description: "删除描述", CreatedBy: "admin", CreateTime: now.Add(time.Minute), UpdateTime: now.Add(time.Minute),
				Nodes: []domainingestion.PipelineNode{{ID: 1, NodeID: "remove", NodeType: "fetcher"}},
			},
		},
		[]domainingestion.Task{
			{ID: "task_keep", PipelineID: "pipe_keep", SourceType: "url", SourceLocation: "https://example.com/old", SourceFileName: "old.md", Status: "running", ChunkCount: 1, Logs: []domainingestion.TaskLog{{NodeID: "fetcher", NodeType: "fetcher", Message: "旧日志", DurationMs: 10, Success: true}}, Metadata: map[string]any{"version": float64(1)}, StartedAt: now, CompletedAt: now.Add(10 * time.Millisecond), CreatedBy: "admin", CreateTime: now, UpdateTime: now},
			{ID: "task_remove", PipelineID: "pipe_remove", SourceType: "file", SourceLocation: "remove.md", Status: "success", ChunkCount: 1, StartedAt: now.Add(time.Minute), CompletedAt: now.Add(time.Minute + 10*time.Millisecond), CreatedBy: "admin", CreateTime: now.Add(time.Minute), UpdateTime: now.Add(time.Minute)},
		},
		[]domainingestion.TaskNode{
			{ID: "tasknode_keep", TaskID: "task_keep", PipelineID: "pipe_keep", NodeID: "fetcher", NodeType: "fetcher", NodeOrder: 1, Status: "running", DurationMs: 10, Message: "旧节点", Output: map[string]any{"step": "old"}, CreateTime: now, UpdateTime: now},
			{ID: "tasknode_remove", TaskID: "task_remove", PipelineID: "pipe_remove", NodeID: "remove", NodeType: "fetcher", NodeOrder: 1, Status: "success", DurationMs: 10, Message: "删除节点", CreateTime: now.Add(time.Minute), UpdateTime: now.Add(time.Minute)},
		},
	); err != nil {
		t.Fatalf("save initial ingestion snapshot failed: %v", err)
	}

	if err := repository.SaveIngestionRecords(
		[]domainingestion.Pipeline{
			{
				ID: "pipe_keep", Name: "新流水线", Description: "新描述", CreatedBy: "owner", CreateTime: now, UpdateTime: now.Add(10 * time.Minute),
				Nodes: []domainingestion.PipelineNode{
					{ID: 2, NodeID: "parser", NodeType: "parser-v2", Settings: map[string]any{"format": "html"}, Condition: map[string]any{"mode": "always"}, NextNodeID: "embedder"},
					{ID: 3, NodeID: "embedder", NodeType: "embedder", Settings: map[string]any{"model": "embedding-v2"}},
				},
			},
			{
				ID: "pipe_new", Name: "新增流水线", Description: "新增描述", CreatedBy: "admin", CreateTime: now.Add(11 * time.Minute), UpdateTime: now.Add(11 * time.Minute),
				Nodes: []domainingestion.PipelineNode{{ID: 2, NodeID: "fetcher_new", NodeType: "fetcher"}},
			},
		},
		[]domainingestion.Task{
			{ID: "task_keep", PipelineID: "pipe_keep", SourceType: "url", SourceLocation: "https://example.com/new", SourceFileName: "new.md", Status: "failed", ChunkCount: 2, ErrorMessage: "timeout", Logs: []domainingestion.TaskLog{{NodeID: "parser", NodeType: "parser-v2", Message: "新日志", DurationMs: 20, Success: false, Error: "timeout"}}, Metadata: map[string]any{"version": float64(2)}, StartedAt: now, CompletedAt: now.Add(20 * time.Millisecond), CreatedBy: "owner", CreateTime: now, UpdateTime: now.Add(10 * time.Minute)},
			{ID: "task_new", PipelineID: "pipe_new", SourceType: "file", SourceLocation: "new.md", Status: "success", ChunkCount: 1, StartedAt: now.Add(11 * time.Minute), CompletedAt: now.Add(11*time.Minute + 20*time.Millisecond), CreatedBy: "admin", CreateTime: now.Add(11 * time.Minute), UpdateTime: now.Add(11 * time.Minute)},
		},
		[]domainingestion.TaskNode{
			{ID: "tasknode_keep", TaskID: "task_keep", PipelineID: "pipe_keep", NodeID: "parser", NodeType: "parser-v2", NodeOrder: 2, Status: "failed", DurationMs: 20, Message: "新节点", ErrorMessage: "timeout", Output: map[string]any{"step": "new"}, CreateTime: now, UpdateTime: now.Add(10 * time.Minute)},
			{ID: "tasknode_new", TaskID: "task_new", PipelineID: "pipe_new", NodeID: "fetcher_new", NodeType: "fetcher", NodeOrder: 1, Status: "success", DurationMs: 30, Message: "新增节点", Output: map[string]any{"step": "insert"}, CreateTime: now.Add(11 * time.Minute), UpdateTime: now.Add(11 * time.Minute)},
		},
	); err != nil {
		t.Fatalf("save reconciled ingestion snapshot failed: %v", err)
	}

	pipelines, tasks, taskNodes, err := repository.LoadIngestionRecords()
	if err != nil {
		t.Fatalf("load reconciled ingestion records failed: %v", err)
	}
	if len(pipelines) != 2 || len(tasks) != 2 || len(taskNodes) != 2 {
		t.Fatalf("expected 2 pipelines/tasks/taskNodes after reconcile, got pipelines=%#v tasks=%#v taskNodes=%#v", pipelines, tasks, taskNodes)
	}
	pipelinesByID := map[string]domainingestion.Pipeline{}
	for _, pipeline := range pipelines {
		pipelinesByID[pipeline.ID] = pipeline
	}
	if _, ok := pipelinesByID["pipe_remove"]; ok {
		t.Fatal("expected missing pipeline to be deleted")
	}
	if pipelinesByID["pipe_keep"].Name != "新流水线" || pipelinesByID["pipe_keep"].Description != "新描述" || len(pipelinesByID["pipe_keep"].Nodes) != 2 {
		t.Fatalf("expected existing pipeline to be updated, got %#v", pipelinesByID["pipe_keep"])
	}
	if pipelinesByID["pipe_keep"].Nodes[0].ID != 2 || pipelinesByID["pipe_keep"].Nodes[0].NodeType != "parser-v2" || pipelinesByID["pipe_keep"].Nodes[0].Settings["format"] != "html" {
		t.Fatalf("expected existing pipeline node to be updated and old node removed, got %#v", pipelinesByID["pipe_keep"].Nodes)
	}
	if pipelinesByID["pipe_new"].Nodes[0].ID != 2 {
		t.Fatalf("expected same pipeline node id to be allowed under a different pipeline, got %#v", pipelinesByID["pipe_new"].Nodes)
	}
	tasksByID := map[string]domainingestion.Task{}
	for _, task := range tasks {
		tasksByID[task.ID] = task
	}
	if _, ok := tasksByID["task_remove"]; ok {
		t.Fatal("expected missing task to be deleted")
	}
	if tasksByID["task_keep"].Status != "failed" || tasksByID["task_keep"].ErrorMessage != "timeout" || tasksByID["task_keep"].Metadata["version"] != float64(2) || len(tasksByID["task_keep"].Logs) != 1 || tasksByID["task_keep"].Logs[0].Error != "timeout" {
		t.Fatalf("expected existing task to be updated, got %#v", tasksByID["task_keep"])
	}
	taskNodesByID := map[string]domainingestion.TaskNode{}
	for _, node := range taskNodes {
		taskNodesByID[node.ID] = node
	}
	if _, ok := taskNodesByID["tasknode_remove"]; ok {
		t.Fatal("expected missing task node to be deleted")
	}
	if taskNodesByID["tasknode_keep"].Status != "failed" || taskNodesByID["tasknode_keep"].ErrorMessage != "timeout" || taskNodesByID["tasknode_keep"].Output["step"] != "new" {
		t.Fatalf("expected existing task node to be updated, got %#v", taskNodesByID["tasknode_keep"])
	}

	if err := repository.SaveIngestionRecords(nil, nil, nil); err != nil {
		t.Fatalf("save empty ingestion snapshot failed: %v", err)
	}
	pipelines, tasks, taskNodes, err = repository.LoadIngestionRecords()
	if err != nil {
		t.Fatalf("load empty ingestion snapshot failed: %v", err)
	}
	if len(pipelines) != 0 || len(tasks) != 0 || len(taskNodes) != 0 {
		t.Fatalf("expected empty ingestion snapshot to clear records, got pipelines=%#v tasks=%#v taskNodes=%#v", pipelines, tasks, taskNodes)
	}
}

func TestSQLIngestionRepositoryRejectsDuplicateSnapshotIDs(t *testing.T) {
	now := time.Date(2026, 5, 11, 13, 0, 0, 0, time.UTC)
	testCases := []struct {
		name      string
		pipelines []domainingestion.Pipeline
		tasks     []domainingestion.Task
		taskNodes []domainingestion.TaskNode
	}{
		{
			name: "pipelines",
			pipelines: []domainingestion.Pipeline{
				{ID: "pipe_keep", Name: "流水线 A", CreateTime: now, UpdateTime: now},
				{ID: "pipe_keep", Name: "流水线 B", CreateTime: now, UpdateTime: now},
			},
		},
		{
			name: "pipeline nodes",
			pipelines: []domainingestion.Pipeline{
				{
					ID: "pipe_keep", Name: "流水线", CreateTime: now, UpdateTime: now,
					Nodes: []domainingestion.PipelineNode{
						{ID: 1, NodeID: "fetcher", NodeType: "fetcher"},
						{ID: 1, NodeID: "parser", NodeType: "parser"},
					},
				},
			},
		},
		{
			name:      "tasks",
			pipelines: []domainingestion.Pipeline{{ID: "pipe_keep", Name: "流水线", CreateTime: now, UpdateTime: now}},
			tasks: []domainingestion.Task{
				{ID: "task_keep", PipelineID: "pipe_keep", Status: "success", StartedAt: now, CompletedAt: now, CreateTime: now, UpdateTime: now},
				{ID: "task_keep", PipelineID: "pipe_keep", Status: "failed", StartedAt: now, CompletedAt: now, CreateTime: now, UpdateTime: now},
			},
		},
		{
			name:      "task nodes",
			pipelines: []domainingestion.Pipeline{{ID: "pipe_keep", Name: "流水线", CreateTime: now, UpdateTime: now}},
			tasks:     []domainingestion.Task{{ID: "task_keep", PipelineID: "pipe_keep", Status: "success", StartedAt: now, CompletedAt: now, CreateTime: now, UpdateTime: now}},
			taskNodes: []domainingestion.TaskNode{
				{ID: "tasknode_keep", TaskID: "task_keep", PipelineID: "pipe_keep", NodeID: "fetcher", NodeType: "fetcher", Status: "success", CreateTime: now, UpdateTime: now},
				{ID: "tasknode_keep", TaskID: "task_keep", PipelineID: "pipe_keep", NodeID: "parser", NodeType: "parser", Status: "failed", CreateTime: now, UpdateTime: now},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			database, err := sql.Open("sqlite", ":memory:")
			if err != nil {
				t.Fatalf("open sqlite database failed: %v", err)
			}
			t.Cleanup(func() { _ = database.Close() })

			repository := NewSQLIngestionRepository(database)
			if err := repository.Bootstrap(); err != nil {
				t.Fatalf("bootstrap ingestion tables failed: %v", err)
			}
			if err := repository.SaveIngestionRecords(
				[]domainingestion.Pipeline{{ID: "pipe_old", Name: "旧流水线", CreateTime: now, UpdateTime: now, Nodes: []domainingestion.PipelineNode{{ID: 1, NodeID: "old", NodeType: "fetcher"}}}},
				[]domainingestion.Task{{ID: "task_old", PipelineID: "pipe_old", Status: "success", StartedAt: now, CompletedAt: now, CreateTime: now, UpdateTime: now}},
				[]domainingestion.TaskNode{{ID: "tasknode_old", TaskID: "task_old", PipelineID: "pipe_old", NodeID: "old", NodeType: "fetcher", Status: "success", CreateTime: now, UpdateTime: now}},
			); err != nil {
				t.Fatalf("save initial ingestion snapshot failed: %v", err)
			}

			err = repository.SaveIngestionRecords(testCase.pipelines, testCase.tasks, testCase.taskNodes)
			if err == nil {
				t.Fatal("expected duplicate id error")
			}

			pipelines, tasks, taskNodes, err := repository.LoadIngestionRecords()
			if err != nil {
				t.Fatalf("load ingestion records after duplicate id failure failed: %v", err)
			}
			if len(pipelines) != 1 || pipelines[0].ID != "pipe_old" || len(tasks) != 1 || tasks[0].ID != "task_old" || len(taskNodes) != 1 || taskNodes[0].ID != "tasknode_old" {
				t.Fatalf("expected existing ingestion records to remain unchanged, got pipelines=%#v tasks=%#v taskNodes=%#v", pipelines, tasks, taskNodes)
			}
		})
	}
}
