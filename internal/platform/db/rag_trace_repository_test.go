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
		ID:             "run_sql",
		TraceID:        "trace_sql",
		TraceName:      "SQL 链路",
		EntryMethod:    "RAGChatController.chat",
		ConversationID: "conv_sql",
		TaskID:         "task_sql",
		UserName:       "admin",
		UserID:         "u_admin",
		Status:         "SUCCESS",
		DurationMs:     88,
		ExtraData:      `{"source":"test"}`,
		StartTime:      startTime,
		EndTime:        startTime.Add(88 * time.Millisecond),
	}
	node := domainragtrace.Node{
		ID:           "node_sql_pk",
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
		ExtraData:    `{"step":1}`,
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
	if runs[0].ID != "run_sql" || runs[0].ExtraData != `{"source":"test"}` {
		t.Fatalf("unexpected run detail %#v", runs[0])
	}
	if len(nodes) != 1 || nodes[0].NodeID != "node_sql" {
		t.Fatalf("unexpected nodes %#v", nodes)
	}
	if nodes[0].ID != "node_sql_pk" || nodes[0].ExtraData != `{"step":1}` {
		t.Fatalf("unexpected node metadata %#v", nodes[0])
	}
	if nodes[0].ParentNodeID != "node-entry" || nodes[0].DurationMs != 80 {
		t.Fatalf("unexpected node detail %#v", nodes[0])
	}
}

func TestSQLRagTraceRepositoryBootstrapsLegacyTables(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite database failed: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	startTime := time.Date(2026, 5, 6, 10, 30, 0, 0, time.UTC)
	endTime := startTime.Add(120 * time.Millisecond)
	if _, err := database.Exec(`
CREATE TABLE agentrag_trace_runs (
    trace_id TEXT PRIMARY KEY,
    trace_name TEXT NOT NULL DEFAULT '',
    entry_method TEXT NOT NULL DEFAULT '',
    conversation_id TEXT NOT NULL DEFAULT '',
    task_id TEXT NOT NULL DEFAULT '',
    user_name TEXT NOT NULL DEFAULT '',
    user_id TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT '',
    error_message TEXT NOT NULL DEFAULT '',
    duration_ms INTEGER NOT NULL DEFAULT 0,
    start_time TIMESTAMP NOT NULL,
    end_time TIMESTAMP NOT NULL
)`); err != nil {
		t.Fatalf("create legacy trace runs table failed: %v", err)
	}
	if _, err := database.Exec(`
CREATE TABLE agentrag_trace_nodes (
    trace_id TEXT NOT NULL,
    node_id TEXT NOT NULL,
    parent_node_id TEXT NOT NULL DEFAULT '',
    depth INTEGER NOT NULL DEFAULT 0,
    node_type TEXT NOT NULL DEFAULT '',
    node_name TEXT NOT NULL DEFAULT '',
    class_name TEXT NOT NULL DEFAULT '',
    method_name TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT '',
    error_message TEXT NOT NULL DEFAULT '',
    duration_ms INTEGER NOT NULL DEFAULT 0,
    start_time TIMESTAMP NOT NULL,
    end_time TIMESTAMP NOT NULL,
    PRIMARY KEY (trace_id, node_id)
)`); err != nil {
		t.Fatalf("create legacy trace nodes table failed: %v", err)
	}
	if _, err := database.Exec(`INSERT INTO agentrag_trace_runs (trace_id, trace_name, entry_method, status, duration_ms, start_time, end_time) VALUES (?, ?, ?, ?, ?, ?, ?)`, "trace_legacy", "旧链路", "chat", "SUCCESS", 120, startTime, endTime); err != nil {
		t.Fatalf("insert legacy trace run failed: %v", err)
	}
	if _, err := database.Exec(`INSERT INTO agentrag_trace_nodes (trace_id, node_id, node_type, node_name, status, duration_ms, start_time, end_time) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, "trace_legacy", "node_legacy", "LLM", "旧节点", "SUCCESS", 100, startTime, endTime); err != nil {
		t.Fatalf("insert legacy trace node failed: %v", err)
	}

	repository := NewSQLRagTraceRepository(database)
	if err := repository.Bootstrap(); err != nil {
		t.Fatalf("bootstrap legacy trace tables failed: %v", err)
	}
	runs, nodes, err := repository.LoadTraceRecords()
	if err != nil {
		t.Fatalf("load migrated traces failed: %v", err)
	}
	if len(runs) != 1 || runs[0].ID != "run_trace_legacy" || runs[0].TraceName != "旧链路" {
		t.Fatalf("unexpected migrated runs %#v", runs)
	}
	if len(nodes) != 1 || nodes[0].ID != "trace_legacy_node_legacy" || nodes[0].NodeName != "旧节点" {
		t.Fatalf("unexpected migrated nodes %#v", nodes)
	}
}
