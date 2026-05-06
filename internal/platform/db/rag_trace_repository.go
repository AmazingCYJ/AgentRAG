package db

import (
	"database/sql"

	domainragtrace "github.com/AmazingCYJ/AgentRAG/internal/domain/ragtrace"
)

// SQLRagTraceRepository 使用关系型数据库持久化 RAG Trace。
type SQLRagTraceRepository struct {
	database *sql.DB
}

// NewSQLRagTraceRepository 创建 RAG Trace SQL 仓储。
func NewSQLRagTraceRepository(database *sql.DB) *SQLRagTraceRepository {
	return &SQLRagTraceRepository{database: database}
}

// Bootstrap 初始化 Trace 运行记录和节点表结构。
func (r *SQLRagTraceRepository) Bootstrap() error {
	if _, err := r.database.Exec(`
CREATE TABLE IF NOT EXISTS agentrag_trace_runs (
    id TEXT PRIMARY KEY,
    trace_id TEXT NOT NULL UNIQUE,
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
    end_time TIMESTAMP NOT NULL,
    extra_data TEXT NOT NULL DEFAULT '',
    create_time TIMESTAMP NOT NULL,
    update_time TIMESTAMP NOT NULL,
    deleted INTEGER NOT NULL DEFAULT 0
)`); err != nil {
		return err
	}
	r.migrateLegacyTraceTables()
	if _, err := r.database.Exec(`
CREATE INDEX IF NOT EXISTS idx_agentrag_trace_runs_task_id ON agentrag_trace_runs (task_id)`); err != nil {
		return err
	}
	if _, err := r.database.Exec(`
CREATE INDEX IF NOT EXISTS idx_agentrag_trace_runs_user_id ON agentrag_trace_runs (user_id)`); err != nil {
		return err
	}
	_, err := r.database.Exec(`
CREATE TABLE IF NOT EXISTS agentrag_trace_nodes (
    id TEXT PRIMARY KEY,
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
    extra_data TEXT NOT NULL DEFAULT '',
    create_time TIMESTAMP NOT NULL,
    update_time TIMESTAMP NOT NULL,
    deleted INTEGER NOT NULL DEFAULT 0,
    UNIQUE (trace_id, node_id)
)`)
	return err
}

// LoadTraceRecords 从数据库加载全部 Trace 运行记录和节点。
func (r *SQLRagTraceRepository) LoadTraceRecords() ([]domainragtrace.Run, []domainragtrace.Node, error) {
	runs, err := r.loadRuns()
	if err != nil {
		return nil, nil, err
	}
	nodes, err := r.loadNodes()
	if err != nil {
		return nil, nil, err
	}
	return runs, nodes, nil
}

func (r *SQLRagTraceRepository) loadRuns() ([]domainragtrace.Run, error) {
	rows, err := r.database.Query(`
SELECT id, trace_id, trace_name, entry_method, conversation_id, task_id, user_name, user_id,
       status, error_message, duration_ms, start_time, end_time, extra_data
FROM agentrag_trace_runs
WHERE deleted = 0
ORDER BY start_time DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	runs := []domainragtrace.Run{}
	for rows.Next() {
		var run domainragtrace.Run
		if err := rows.Scan(
			&run.ID,
			&run.TraceID,
			&run.TraceName,
			&run.EntryMethod,
			&run.ConversationID,
			&run.TaskID,
			&run.UserName,
			&run.UserID,
			&run.Status,
			&run.ErrorMessage,
			&run.DurationMs,
			&run.StartTime,
			&run.EndTime,
			&run.ExtraData,
		); err != nil {
			return nil, err
		}
		runs = append(runs, run)
	}
	return runs, rows.Err()
}

func (r *SQLRagTraceRepository) loadNodes() ([]domainragtrace.Node, error) {
	rows, err := r.database.Query(`
SELECT id, trace_id, node_id, parent_node_id, depth, node_type, node_name,
       class_name, method_name, status, error_message, duration_ms, start_time, end_time, extra_data
FROM agentrag_trace_nodes
WHERE deleted = 0
ORDER BY trace_id ASC, start_time ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	nodes := []domainragtrace.Node{}
	for rows.Next() {
		var node domainragtrace.Node
		if err := rows.Scan(
			&node.ID,
			&node.TraceID,
			&node.NodeID,
			&node.ParentNodeID,
			&node.Depth,
			&node.NodeType,
			&node.NodeName,
			&node.ClassName,
			&node.MethodName,
			&node.Status,
			&node.ErrorMessage,
			&node.DurationMs,
			&node.StartTime,
			&node.EndTime,
			&node.ExtraData,
		); err != nil {
			return nil, err
		}
		nodes = append(nodes, node)
	}
	return nodes, rows.Err()
}

// SaveTraceRecords 覆盖保存当前 Trace 运行记录和节点。
func (r *SQLRagTraceRepository) SaveTraceRecords(runs []domainragtrace.Run, nodes []domainragtrace.Node) error {
	tx, err := r.database.Begin()
	if err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM agentrag_trace_nodes`); err != nil {
		_ = tx.Rollback()
		return err
	}
	if _, err := tx.Exec(`DELETE FROM agentrag_trace_runs`); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := saveTraceRuns(tx, runs); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := saveTraceNodes(tx, nodes); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func saveTraceRuns(tx *sql.Tx, runs []domainragtrace.Run) error {
	stmt, err := tx.Prepare(`
INSERT INTO agentrag_trace_runs
    (id, trace_id, trace_name, entry_method, conversation_id, task_id, user_name, user_id,
     status, error_message, duration_ms, start_time, end_time, extra_data, create_time, update_time, deleted)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 0)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, run := range runs {
		if _, err := stmt.Exec(
			defaultTraceRunID(run),
			run.TraceID,
			run.TraceName,
			run.EntryMethod,
			run.ConversationID,
			run.TaskID,
			run.UserName,
			run.UserID,
			run.Status,
			run.ErrorMessage,
			run.DurationMs,
			normalizeTime(run.StartTime),
			normalizeTime(run.EndTime),
			run.ExtraData,
			normalizeTime(run.StartTime),
			normalizeTime(run.EndTime),
		); err != nil {
			return err
		}
	}
	return nil
}

func saveTraceNodes(tx *sql.Tx, nodes []domainragtrace.Node) error {
	stmt, err := tx.Prepare(`
INSERT INTO agentrag_trace_nodes
    (id, trace_id, node_id, parent_node_id, depth, node_type, node_name,
     class_name, method_name, status, error_message, duration_ms, start_time, end_time, extra_data, create_time, update_time, deleted)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 0)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, node := range nodes {
		if _, err := stmt.Exec(
			defaultTraceNodeID(node),
			node.TraceID,
			node.NodeID,
			node.ParentNodeID,
			node.Depth,
			node.NodeType,
			node.NodeName,
			node.ClassName,
			node.MethodName,
			node.Status,
			node.ErrorMessage,
			node.DurationMs,
			normalizeTime(node.StartTime),
			normalizeTime(node.EndTime),
			node.ExtraData,
			normalizeTime(node.StartTime),
			normalizeTime(node.EndTime),
		); err != nil {
			return err
		}
	}
	return nil
}

func defaultTraceRunID(run domainragtrace.Run) string {
	if run.ID != "" {
		return run.ID
	}
	return "run_" + run.TraceID
}

func defaultTraceNodeID(node domainragtrace.Node) string {
	if node.ID != "" {
		return node.ID
	}
	return node.TraceID + "_" + node.NodeID
}

func (r *SQLRagTraceRepository) migrateLegacyTraceTables() {
	for _, statement := range []string{
		`ALTER TABLE agentrag_trace_runs ADD COLUMN id TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE agentrag_trace_runs ADD COLUMN extra_data TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE agentrag_trace_runs ADD COLUMN create_time TIMESTAMP`,
		`ALTER TABLE agentrag_trace_runs ADD COLUMN update_time TIMESTAMP`,
		`ALTER TABLE agentrag_trace_runs ADD COLUMN deleted INTEGER NOT NULL DEFAULT 0`,
		`UPDATE agentrag_trace_runs SET id = 'run_' || trace_id WHERE id = ''`,
		`UPDATE agentrag_trace_runs SET create_time = start_time WHERE create_time IS NULL`,
		`UPDATE agentrag_trace_runs SET update_time = end_time WHERE update_time IS NULL`,
		`ALTER TABLE agentrag_trace_nodes ADD COLUMN id TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE agentrag_trace_nodes ADD COLUMN extra_data TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE agentrag_trace_nodes ADD COLUMN create_time TIMESTAMP`,
		`ALTER TABLE agentrag_trace_nodes ADD COLUMN update_time TIMESTAMP`,
		`ALTER TABLE agentrag_trace_nodes ADD COLUMN deleted INTEGER NOT NULL DEFAULT 0`,
		`UPDATE agentrag_trace_nodes SET id = trace_id || '_' || node_id WHERE id = ''`,
		`UPDATE agentrag_trace_nodes SET create_time = start_time WHERE create_time IS NULL`,
		`UPDATE agentrag_trace_nodes SET update_time = end_time WHERE update_time IS NULL`,
	} {
		_, _ = r.database.Exec(statement)
	}
}
