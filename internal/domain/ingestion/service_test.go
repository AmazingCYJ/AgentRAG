package ingestion

import (
	"path/filepath"
	"testing"
	"time"

	platformstate "github.com/AmazingCYJ/AgentRAG/internal/platform/state"
)

func TestIngestionStatePersistsAcrossServiceRecreation(t *testing.T) {
	store, err := platformstate.NewFileStore(filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatalf("create state store failed: %v", err)
	}

	service := NewService(store)
	pipeline, err := service.CreatePipeline(PipelineSaveRequest{
		Name:        "测试流水线",
		Description: "测试描述",
		CreatedBy:   "admin",
		Nodes: []PipelineNodeRequest{
			{
				NodeID:   "fetcher",
				NodeType: "fetcher",
				Settings: map[string]any{"timeoutMs": 1000},
			},
			{
				NodeID:     "parser",
				NodeType:   "parser",
				NextNodeID: "",
			},
		},
	})
	if err != nil {
		t.Fatalf("create pipeline failed: %v", err)
	}
	result, err := service.CreateTask(TaskCreateRequest{
		PipelineID: pipeline.ID,
		Source: TaskSource{
			Type:     "url",
			Location: "https://example.com",
			FileName: "demo.md",
		},
		CreatedBy: "admin",
	})
	if err != nil {
		t.Fatalf("create task failed: %v", err)
	}

	recreated := NewService(store)
	recreatedPipeline, err := recreated.GetPipeline(pipeline.ID)
	if err != nil {
		t.Fatalf("get recreated pipeline failed: %v", err)
	}
	if recreatedPipeline.Name != "测试流水线" {
		t.Fatalf("expected pipeline name 测试流水线, got %s", recreatedPipeline.Name)
	}
	task, err := recreated.GetTask(result.TaskID)
	if err != nil {
		t.Fatalf("get recreated task failed: %v", err)
	}
	if task.PipelineID != pipeline.ID {
		t.Fatalf("expected recreated task pipeline %s, got %s", pipeline.ID, task.PipelineID)
	}
	nodes, err := recreated.ListTaskNodes(result.TaskID)
	if err != nil {
		t.Fatalf("get recreated task nodes failed: %v", err)
	}
	if len(nodes) != 2 {
		t.Fatalf("expected 2 recreated task nodes, got %d", len(nodes))
	}
}

type memoryIngestionRepository struct {
	pipelines []Pipeline
	tasks     []Task
	taskNodes []TaskNode
}

func (r *memoryIngestionRepository) LoadIngestionRecords() ([]Pipeline, []Task, []TaskNode, error) {
	return r.pipelines, r.tasks, r.taskNodes, nil
}

func (r *memoryIngestionRepository) SaveIngestionRecords(pipelines []Pipeline, tasks []Task, taskNodes []TaskNode) error {
	r.pipelines = append([]Pipeline(nil), pipelines...)
	r.tasks = append([]Task(nil), tasks...)
	r.taskNodes = append([]TaskNode(nil), taskNodes...)
	return nil
}

func TestIngestionServiceLoadsAndPersistsThroughRepository(t *testing.T) {
	now := time.Date(2026, 5, 4, 10, 0, 0, 0, time.UTC)
	repository := &memoryIngestionRepository{
		pipelines: []Pipeline{
			{
				ID:          "pipe_existing",
				Name:        "已有流水线",
				Description: "已有描述",
				CreatedBy:   "admin",
				Nodes: []PipelineNode{
					{
						ID:       7,
						NodeID:   "fetcher",
						NodeType: "fetcher",
						Settings: map[string]any{
							"timeoutMs": float64(1000),
						},
					},
				},
				CreateTime: now,
				UpdateTime: now,
			},
		},
		tasks: []Task{
			{
				ID:             "task_existing",
				PipelineID:     "pipe_existing",
				SourceType:     "url",
				SourceLocation: "https://example.com",
				Status:         "success",
				ChunkCount:     1,
				Logs: []TaskLog{
					{NodeID: "fetcher", NodeType: "fetcher", Message: "节点执行完成", DurationMs: 120, Success: true},
				},
				Metadata:    map[string]any{"source": "test"},
				StartedAt:   now,
				CompletedAt: now.Add(time.Second),
				CreatedBy:   "admin",
				CreateTime:  now,
				UpdateTime:  now,
			},
		},
		taskNodes: []TaskNode{
			{
				ID:         "node_existing",
				TaskID:     "task_existing",
				PipelineID: "pipe_existing",
				NodeID:     "fetcher",
				NodeType:   "fetcher",
				NodeOrder:  1,
				Status:     "success",
				DurationMs: 120,
				Message:    "节点执行完成",
				Output:     map[string]any{"nextNodeId": ""},
				CreateTime: now,
				UpdateTime: now,
			},
		},
	}

	service := NewServiceWithRepository(repository)
	pipeline, err := service.GetPipeline("pipe_existing")
	if err != nil {
		t.Fatalf("load pipeline failed: %v", err)
	}
	if pipeline.Name != "已有流水线" || len(pipeline.Nodes) != 1 {
		t.Fatalf("unexpected loaded pipeline %#v", pipeline)
	}
	nodes, err := service.ListTaskNodes("task_existing")
	if err != nil {
		t.Fatalf("list loaded task nodes failed: %v", err)
	}
	if len(nodes) != 1 {
		t.Fatalf("expected 1 loaded task node, got %d", len(nodes))
	}

	created, err := service.CreatePipeline(PipelineSaveRequest{
		Name:      "新增流水线",
		CreatedBy: "admin",
		Nodes: []PipelineNodeRequest{
			{NodeID: "parser", NodeType: "parser"},
		},
	})
	if err != nil {
		t.Fatalf("create pipeline failed: %v", err)
	}
	if created.ID == "" {
		t.Fatal("expected created pipeline id")
	}
	if len(repository.pipelines) != 2 {
		t.Fatalf("expected repository to save 2 pipelines, got %d", len(repository.pipelines))
	}
}
