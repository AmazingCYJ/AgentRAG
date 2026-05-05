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
		return err
	}
	_, err := r.database.Exec(`
CREATE TABLE IF NOT EXISTS agentrag_trace_nodes (
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
SELECT trace_id, trace_name, entry_method, conversation_id, task_id, user_name, user_id,
       status, error_message, duration_ms, start_time, end_time
FROM agentrag_trace_runs
ORDER BY start_time DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	runs := []domainragtrace.Run{}
	for rows.Next() {
		var run domainragtrace.Run
		if err := rows.Scan(
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
		); err != nil {
			return nil, err
		}
		runs = append(runs, run)
	}
	return runs, rows.Err()
}

func (r *SQLRagTraceRepository) loadNodes() ([]domainragtrace.Node, error) {
	rows, err := r.database.Query(`
SELECT trace_id, node_id, parent_node_id, depth, node_type, node_name,
       class_name, method_name, status, error_message, duration_ms, start_time, end_time
FROM agentrag_trace_nodes
ORDER BY trace_id ASC, start_time ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	nodes := []domainragtrace.Node{}
	for rows.Next() {
		var node domainragtrace.Node
		if err := rows.Scan(
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
    (trace_id, trace_name, entry_method, conversation_id, task_id, user_name, user_id,
     status, error_message, duration_ms, start_time, end_time)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, run := range runs {
		if _, err := stmt.Exec(
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
		); err != nil {
			return err
		}
	}
	return nil
}

func saveTraceNodes(tx *sql.Tx, nodes []domainragtrace.Node) error {
	stmt, err := tx.Prepare(`
INSERT INTO agentrag_trace_nodes
    (trace_id, node_id, parent_node_id, depth, node_type, node_name,
     class_name, method_name, status, error_message, duration_ms, start_time, end_time)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, node := range nodes {
		if _, err := stmt.Exec(
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
		); err != nil {
			return err
		}
	}
	return nil
}
