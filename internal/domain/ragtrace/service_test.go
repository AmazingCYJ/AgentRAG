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
		Username:       "admin",
		UserID:         "u_admin",
		Status:         "success",
		DurationMs:     1200,
		StartTime:      startTime,
		EndTime:        startTime.Add(1200 * time.Millisecond),
		DeepThinking:   true,
		Steps: []ChatTraceStep{
			{
				NodeID:     "route",
				NodeType:   "ROUTER",
				NodeName:   "Route Intent",
				Status:     "success",
				DurationMs: 50,
			},
			{
				NodeID:     "tool",
				NodeType:   "TOOL",
				NodeName:   "Call MCP Tool",
				Status:     "success",
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
