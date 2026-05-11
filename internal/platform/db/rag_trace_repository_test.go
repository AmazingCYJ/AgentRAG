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

func TestSQLRagTraceRepositoryReconcilesSavedRecords(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite database failed: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	repository := NewSQLRagTraceRepository(database)
	if err := repository.Bootstrap(); err != nil {
		t.Fatalf("bootstrap trace tables failed: %v", err)
	}
	startTime := time.Date(2026, 5, 10, 17, 0, 0, 0, time.UTC)
	if err := repository.SaveTraceRecords(
		[]domainragtrace.Run{
			{ID: "run_keep_pk", TraceID: "trace_keep", TraceName: "旧链路", EntryMethod: "chat", TaskID: "task_keep", Status: "SUCCESS", DurationMs: 80, ExtraData: `{"old":true}`, StartTime: startTime, EndTime: startTime.Add(80 * time.Millisecond)},
			{ID: "run_remove_pk", TraceID: "trace_remove", TraceName: "删除链路", EntryMethod: "chat", TaskID: "task_remove", Status: "SUCCESS", DurationMs: 60, StartTime: startTime.Add(time.Minute), EndTime: startTime.Add(time.Minute + 60*time.Millisecond)},
		},
		[]domainragtrace.Node{
			{ID: "node_keep_pk", TraceID: "trace_keep", NodeID: "node_keep", NodeType: "LLM", NodeName: "旧节点", Status: "SUCCESS", DurationMs: 70, ExtraData: `{"old":1}`, StartTime: startTime.Add(10 * time.Millisecond), EndTime: startTime.Add(80 * time.Millisecond)},
			{ID: "node_remove_pk", TraceID: "trace_remove", NodeID: "node_remove", NodeType: "TOOL", NodeName: "删除节点", Status: "SUCCESS", DurationMs: 50, StartTime: startTime.Add(time.Minute), EndTime: startTime.Add(time.Minute + 50*time.Millisecond)},
		},
	); err != nil {
		t.Fatalf("save initial trace snapshot failed: %v", err)
	}

	if err := repository.SaveTraceRecords(
		[]domainragtrace.Run{
			{ID: "run_keep_pk", TraceID: "trace_keep", TraceName: "新链路", EntryMethod: "chat", ConversationID: "conv_keep", TaskID: "task_keep_new", UserName: "admin", UserID: "u_admin", Status: "FAILED", ErrorMessage: "timeout", DurationMs: 120, ExtraData: `{"new":true}`, StartTime: startTime, EndTime: startTime.Add(120 * time.Millisecond)},
			{ID: "run_new_pk", TraceID: "trace_new", TraceName: "新增链路", EntryMethod: "chat", TaskID: "task_new", Status: "SUCCESS", DurationMs: 90, StartTime: startTime.Add(2 * time.Minute), EndTime: startTime.Add(2*time.Minute + 90*time.Millisecond)},
		},
		[]domainragtrace.Node{
			{ID: "node_keep_pk", TraceID: "trace_keep", NodeID: "node_keep", ParentNodeID: "node-entry", Depth: 1, NodeType: "RETRIEVER", NodeName: "新节点", ClassName: "Workflow", MethodName: "retrieve", Status: "FAILED", ErrorMessage: "timeout", DurationMs: 110, ExtraData: `{"new":1}`, StartTime: startTime.Add(10 * time.Millisecond), EndTime: startTime.Add(120 * time.Millisecond)},
			{ID: "node_new_pk", TraceID: "trace_new", NodeID: "node_new", NodeType: "LLM", NodeName: "新增节点", Status: "SUCCESS", DurationMs: 80, StartTime: startTime.Add(2 * time.Minute), EndTime: startTime.Add(2*time.Minute + 80*time.Millisecond)},
		},
	); err != nil {
		t.Fatalf("save reconciled trace snapshot failed: %v", err)
	}

	runs, nodes, err := repository.LoadTraceRecords()
	if err != nil {
		t.Fatalf("load reconciled trace records failed: %v", err)
	}
	if len(runs) != 2 || len(nodes) != 2 {
		t.Fatalf("expected 2 runs and nodes after reconcile, got runs=%#v nodes=%#v", runs, nodes)
	}
	runsByTraceID := map[string]domainragtrace.Run{}
	for _, run := range runs {
		runsByTraceID[run.TraceID] = run
	}
	if _, ok := runsByTraceID["trace_remove"]; ok {
		t.Fatal("expected missing trace run to be deleted")
	}
	if runsByTraceID["trace_keep"].TraceName != "新链路" || runsByTraceID["trace_keep"].Status != "FAILED" || runsByTraceID["trace_keep"].ExtraData != `{"new":true}` {
		t.Fatalf("expected existing trace run to be updated, got %#v", runsByTraceID["trace_keep"])
	}
	if runsByTraceID["trace_new"].TaskID != "task_new" {
		t.Fatalf("expected new trace run to be inserted, got %#v", runsByTraceID["trace_new"])
	}
	nodesByID := map[string]domainragtrace.Node{}
	for _, node := range nodes {
		nodesByID[node.ID] = node
	}
	if _, ok := nodesByID["node_remove_pk"]; ok {
		t.Fatal("expected missing trace node to be deleted")
	}
	if nodesByID["node_keep_pk"].NodeType != "RETRIEVER" || nodesByID["node_keep_pk"].ErrorMessage != "timeout" || nodesByID["node_keep_pk"].ExtraData != `{"new":1}` {
		t.Fatalf("expected existing trace node to be updated, got %#v", nodesByID["node_keep_pk"])
	}
	if nodesByID["node_new_pk"].NodeName != "新增节点" {
		t.Fatalf("expected new trace node to be inserted, got %#v", nodesByID["node_new_pk"])
	}

	if err := repository.SaveTraceRecords(nil, nil); err != nil {
		t.Fatalf("save empty trace snapshot failed: %v", err)
	}
	runs, nodes, err = repository.LoadTraceRecords()
	if err != nil {
		t.Fatalf("load empty trace snapshot failed: %v", err)
	}
	if len(runs) != 0 || len(nodes) != 0 {
		t.Fatalf("expected empty trace snapshot to clear records, got runs=%#v nodes=%#v", runs, nodes)
	}
}

func TestSQLRagTraceRepositoryAllowsUniqueKeySwaps(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite database failed: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	repository := NewSQLRagTraceRepository(database)
	if err := repository.Bootstrap(); err != nil {
		t.Fatalf("bootstrap trace tables failed: %v", err)
	}
	startTime := time.Date(2026, 5, 10, 18, 0, 0, 0, time.UTC)
	if err := repository.SaveTraceRecords(
		[]domainragtrace.Run{
			{ID: "run_a_pk", TraceID: "trace_a", TraceName: "A", Status: "SUCCESS", StartTime: startTime, EndTime: startTime},
			{ID: "run_b_pk", TraceID: "trace_b", TraceName: "B", Status: "SUCCESS", StartTime: startTime.Add(time.Minute), EndTime: startTime.Add(time.Minute)},
		},
		[]domainragtrace.Node{
			{ID: "node_a_pk", TraceID: "trace_a", NodeID: "node_a", NodeName: "A", Status: "SUCCESS", StartTime: startTime, EndTime: startTime},
			{ID: "node_b_pk", TraceID: "trace_a", NodeID: "node_b", NodeName: "B", Status: "SUCCESS", StartTime: startTime.Add(time.Millisecond), EndTime: startTime.Add(time.Millisecond)},
		},
	); err != nil {
		t.Fatalf("save initial trace records failed: %v", err)
	}
	if err := repository.SaveTraceRecords(
		[]domainragtrace.Run{
			{ID: "run_a_pk", TraceID: "trace_b", TraceName: "A swapped", Status: "SUCCESS", StartTime: startTime, EndTime: startTime},
			{ID: "run_b_pk", TraceID: "trace_a", TraceName: "B swapped", Status: "SUCCESS", StartTime: startTime.Add(time.Minute), EndTime: startTime.Add(time.Minute)},
		},
		[]domainragtrace.Node{
			{ID: "node_a_pk", TraceID: "trace_a", NodeID: "node_b", NodeName: "A swapped", Status: "SUCCESS", StartTime: startTime, EndTime: startTime},
			{ID: "node_b_pk", TraceID: "trace_a", NodeID: "node_a", NodeName: "B swapped", Status: "SUCCESS", StartTime: startTime.Add(time.Millisecond), EndTime: startTime.Add(time.Millisecond)},
		},
	); err != nil {
		t.Fatalf("save swapped trace records failed: %v", err)
	}

	runs, nodes, err := repository.LoadTraceRecords()
	if err != nil {
		t.Fatalf("load swapped trace records failed: %v", err)
	}
	runsByID := map[string]domainragtrace.Run{}
	for _, run := range runs {
		runsByID[run.ID] = run
	}
	if runsByID["run_a_pk"].TraceID != "trace_b" || runsByID["run_b_pk"].TraceID != "trace_a" {
		t.Fatalf("expected trace IDs to be swapped, got %#v", runsByID)
	}
	nodesByID := map[string]domainragtrace.Node{}
	for _, node := range nodes {
		nodesByID[node.ID] = node
	}
	if nodesByID["node_a_pk"].NodeID != "node_b" || nodesByID["node_b_pk"].NodeID != "node_a" {
		t.Fatalf("expected trace node IDs to be swapped, got %#v", nodesByID)
	}
}

func TestSQLRagTraceRepositoryRejectsDuplicateSnapshotIDs(t *testing.T) {
	startTime := time.Date(2026, 5, 10, 19, 0, 0, 0, time.UTC)
	testCases := []struct {
		name  string
		runs  []domainragtrace.Run
		nodes []domainragtrace.Node
	}{
		{
			name: "runs",
			runs: []domainragtrace.Run{
				{ID: "run_keep_pk", TraceID: "trace_keep", Status: "SUCCESS", StartTime: startTime, EndTime: startTime},
				{ID: "run_keep_pk", TraceID: "trace_duplicate", Status: "SUCCESS", StartTime: startTime, EndTime: startTime},
			},
		},
		{
			name: "nodes",
			runs: []domainragtrace.Run{{ID: "run_keep_pk", TraceID: "trace_keep", Status: "SUCCESS", StartTime: startTime, EndTime: startTime}},
			nodes: []domainragtrace.Node{
				{ID: "node_keep_pk", TraceID: "trace_keep", NodeID: "node_keep", Status: "SUCCESS", StartTime: startTime, EndTime: startTime},
				{ID: "node_keep_pk", TraceID: "trace_keep", NodeID: "node_duplicate", Status: "SUCCESS", StartTime: startTime, EndTime: startTime},
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

			repository := NewSQLRagTraceRepository(database)
			if err := repository.Bootstrap(); err != nil {
				t.Fatalf("bootstrap trace tables failed: %v", err)
			}
			if err := repository.SaveTraceRecords(
				[]domainragtrace.Run{{ID: "run_old_pk", TraceID: "trace_old", TraceName: "旧链路", Status: "SUCCESS", StartTime: startTime, EndTime: startTime}},
				[]domainragtrace.Node{{ID: "node_old_pk", TraceID: "trace_old", NodeID: "node_old", NodeName: "旧节点", Status: "SUCCESS", StartTime: startTime, EndTime: startTime}},
			); err != nil {
				t.Fatalf("save initial trace snapshot failed: %v", err)
			}

			err = repository.SaveTraceRecords(testCase.runs, testCase.nodes)
			if err == nil {
				t.Fatal("expected duplicate id error")
			}

			runs, nodes, err := repository.LoadTraceRecords()
			if err != nil {
				t.Fatalf("load trace records after duplicate id failure failed: %v", err)
			}
			if len(runs) != 1 || runs[0].ID != "run_old_pk" || len(nodes) != 1 || nodes[0].ID != "node_old_pk" {
				t.Fatalf("expected existing trace records to remain unchanged, got runs=%#v nodes=%#v", runs, nodes)
			}
		})
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
