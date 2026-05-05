package db

import (
	"database/sql"
	"testing"
	"time"

	domainragtrace "github.com/AmazingCYJ/AgentRAG/internal/domain/ragtrace"
	_ "modernc.org/sqlite"
)

func TestSQLRagTraceRepositorySavesAndLoadsRecords(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite database failed: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	repository := NewSQLRagTraceRepository(database)
	if err := repository.Bootstrap(); err != nil {
		t.Fatalf("bootstrap trace tables failed: %v", err)
	}

	startTime := time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC)
	run := domainragtrace.Run{
		TraceID:        "trace_sql",
		TraceName:      "SQL 链路",
		EntryMethod:    "RAGChatController.chat",
		ConversationID: "conv_sql",
		TaskID:         "task_sql",
		UserName:       "admin",
		UserID:         "u_admin",
		Status:         "SUCCESS",
		DurationMs:     88,
		StartTime:      startTime,
		EndTime:        startTime.Add(88 * time.Millisecond),
	}
	node := domainragtrace.Node{
		TraceID:      run.TraceID,
		NodeID:       "node_sql",
		ParentNodeID: "node-entry",
		Depth:        1,
		NodeType:     "LLM",
		NodeName:     "生成回答",
		ClassName:    "EinoWorkflow",
		MethodName:   "stream",
		Status:       "SUCCESS",
		DurationMs:   80,
		StartTime:    startTime.Add(8 * time.Millisecond),
		EndTime:      run.EndTime,
	}
	if err := repository.SaveTraceRecords([]domainragtrace.Run{run}, []domainragtrace.Node{node}); err != nil {
		t.Fatalf("save trace records failed: %v", err)
	}

	runs, nodes, err := repository.LoadTraceRecords()
	if err != nil {
		t.Fatalf("load trace records failed: %v", err)
	}
	if len(runs) != 1 || runs[0].TaskID != "task_sql" {
		t.Fatalf("unexpected runs %#v", runs)
	}
	if len(nodes) != 1 || nodes[0].NodeID != "node_sql" {
		t.Fatalf("unexpected nodes %#v", nodes)
	}
	if nodes[0].ParentNodeID != "node-entry" || nodes[0].DurationMs != 80 {
		t.Fatalf("unexpected node detail %#v", nodes[0])
	}
}
