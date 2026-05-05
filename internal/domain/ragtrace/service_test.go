package ragtrace

import (
	"path/filepath"
	"testing"
	"time"

	platformstate "github.com/AmazingCYJ/AgentRAG/internal/platform/state"
)

func TestRagTraceStatePersistsAcrossServiceRecreation(t *testing.T) {
	store, err := platformstate.NewFileStore(filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatalf("create state store failed: %v", err)
	}

	service := NewService(store)
	startTime := time.Date(2026, 4, 19, 10, 0, 0, 0, time.UTC)
	traceID := service.RecordChatTrace(ChatTraceRecord{
		TraceName:      "测试链路",
		ConversationID: "conv_1",
		TaskID:         "task_1",
		UserName:       "admin",
		UserID:         "u_admin",
		Status:         "SUCCESS",
		DurationMs:     1200,
		StartTime:      startTime,
		EndTime:        startTime.Add(1200 * time.Millisecond),
		DeepThinking:   true,
		Steps: []ChatTraceStep{
			{
				NodeID:     "route",
				NodeType:   "ROUTER",
				NodeName:   "Route Intent",
				Status:     "SUCCESS",
				DurationMs: 50,
			},
			{
				NodeID:     "tool",
				NodeType:   "TOOL",
				NodeName:   "Call MCP Tool",
				Status:     "SUCCESS",
				DurationMs: 180,
			},
		},
	})

	recreated := NewService(store)
	detail, err := recreated.Detail(traceID)
	if err != nil {
		t.Fatalf("get recreated trace detail failed: %v", err)
	}
	if detail.Run.TaskID != "task_1" {
		t.Fatalf("expected task id task_1, got %s", detail.Run.TaskID)
	}
	if len(detail.Nodes) == 0 {
		t.Fatal("expected recreated trace nodes")
	}
	foundRouter := false
	foundTool := false
	for _, node := range detail.Nodes {
		if node.NodeType == "ROUTER" {
			foundRouter = true
		}
		if node.NodeType == "TOOL" {
			foundTool = true
		}
	}
	if !foundRouter || !foundTool {
		t.Fatalf("expected router/tool nodes after recreation, router=%v tool=%v", foundRouter, foundTool)
	}
}

type memoryTraceRepository struct {
	runs  []Run
	nodes []Node
}

func (r *memoryTraceRepository) LoadTraceRecords() ([]Run, []Node, error) {
	return r.runs, r.nodes, nil
}

func (r *memoryTraceRepository) SaveTraceRecords(runs []Run, nodes []Node) error {
	r.runs = append([]Run(nil), runs...)
	r.nodes = append([]Node(nil), nodes...)
	return nil
}

func TestRagTraceServiceLoadsAndPersistsThroughRepository(t *testing.T) {
	startTime := time.Date(2026, 5, 2, 11, 0, 0, 0, time.UTC)
	repository := &memoryTraceRepository{
		runs: []Run{
			{
				TraceID:        "trace_existing",
				TraceName:      "已有链路",
				EntryMethod:    "RAGChatController.chat",
				ConversationID: "conv_existing",
				TaskID:         "task_existing",
				UserID:         "u_admin",
				Status:         "SUCCESS",
				DurationMs:     20,
				StartTime:      startTime,
				EndTime:        startTime.Add(20 * time.Millisecond),
			},
		},
		nodes: []Node{
			{
				TraceID:    "trace_existing",
				NodeID:     "node-existing",
				NodeType:   "ENTRY",
				NodeName:   "Chat Entry",
				Status:     "SUCCESS",
				DurationMs: 20,
				StartTime:  startTime,
				EndTime:    startTime.Add(20 * time.Millisecond),
			},
		},
	}

	service := NewServiceWithRepository(repository)
	detail, err := service.Detail("trace_existing")
	if err != nil {
		t.Fatalf("load existing trace failed: %v", err)
	}
	if detail.Run.TaskID != "task_existing" || len(detail.Nodes) != 1 {
		t.Fatalf("unexpected loaded trace detail %#v", detail)
	}

	traceID := service.RecordChatTrace(ChatTraceRecord{
		TraceName:      "新增链路",
		ConversationID: "conv_new",
		TaskID:         "task_new",
		UserID:         "u_admin",
		Status:         "SUCCESS",
		DurationMs:     30,
		StartTime:      startTime.Add(time.Minute),
		EndTime:        startTime.Add(time.Minute + 30*time.Millisecond),
		Steps: []ChatTraceStep{
			{NodeID: "router", NodeType: "ROUTER", NodeName: "路由", Status: "SUCCESS", DurationMs: 5},
		},
	})
	if traceID == "" {
		t.Fatal("expected generated trace id")
	}
	if len(repository.runs) != 2 {
		t.Fatalf("expected repository to save 2 runs, got %d", len(repository.runs))
	}
	if len(repository.nodes) == 0 {
		t.Fatal("expected repository to save nodes")
	}
}
