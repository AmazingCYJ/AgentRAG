package ingestion

import (
	"path/filepath"
	"testing"

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
