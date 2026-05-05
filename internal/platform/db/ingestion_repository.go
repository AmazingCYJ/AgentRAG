package db

import (
	"database/sql"
	"encoding/json"

	domainingestion "github.com/AmazingCYJ/AgentRAG/internal/domain/ingestion"
)

// SQLIngestionRepository 使用关系型数据库持久化采集流水线、任务和任务节点。
type SQLIngestionRepository struct {
	database *sql.DB
}

// NewSQLIngestionRepository 创建采集 SQL 仓储。
func NewSQLIngestionRepository(database *sql.DB) *SQLIngestionRepository {
	return &SQLIngestionRepository{database: database}
}

// Bootstrap 初始化采集相关表结构。
func (r *SQLIngestionRepository) Bootstrap() error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS agentrag_ingestion_pipelines (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_by TEXT NOT NULL DEFAULT '',
    create_time TIMESTAMP NOT NULL,
    update_time TIMESTAMP NOT NULL
)`,
		`CREATE TABLE IF NOT EXISTS agentrag_ingestion_pipeline_nodes (
    id INTEGER NOT NULL,
    pipeline_id TEXT NOT NULL,
    node_id TEXT NOT NULL,
    node_type TEXT NOT NULL,
    settings TEXT NOT NULL DEFAULT '',
    condition_json TEXT NOT NULL DEFAULT '',
    next_node_id TEXT NOT NULL DEFAULT '',
    node_order INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (pipeline_id, id)
)`,
		`CREATE TABLE IF NOT EXISTS agentrag_ingestion_tasks (
    id TEXT PRIMARY KEY,
    pipeline_id TEXT NOT NULL,
    source_type TEXT NOT NULL DEFAULT '',
    source_location TEXT NOT NULL DEFAULT '',
    source_file_name TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT '',
    chunk_count INTEGER NOT NULL DEFAULT 0,
    error_message TEXT NOT NULL DEFAULT '',
    logs TEXT NOT NULL DEFAULT '',
    metadata TEXT NOT NULL DEFAULT '',
    started_at TIMESTAMP NOT NULL,
    completed_at TIMESTAMP NOT NULL,
    created_by TEXT NOT NULL DEFAULT '',
    create_time TIMESTAMP NOT NULL,
    update_time TIMESTAMP NOT NULL
)`,
		`CREATE TABLE IF NOT EXISTS agentrag_ingestion_task_nodes (
    id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL,
    pipeline_id TEXT NOT NULL,
    node_id TEXT NOT NULL,
    node_type TEXT NOT NULL,
    node_order INTEGER NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT '',
    duration_ms INTEGER NOT NULL DEFAULT 0,
    message TEXT NOT NULL DEFAULT '',
    error_message TEXT NOT NULL DEFAULT '',
    output TEXT NOT NULL DEFAULT '',
    create_time TIMESTAMP NOT NULL,
    update_time TIMESTAMP NOT NULL
)`,
	}
	for _, statement := range statements {
		if _, err := r.database.Exec(statement); err != nil {
			return err
		}
	}
	return nil
}

// LoadIngestionRecords 从数据库加载全部采集数据。
func (r *SQLIngestionRepository) LoadIngestionRecords() ([]domainingestion.Pipeline, []domainingestion.Task, []domainingestion.TaskNode, error) {
	pipelines, err := r.loadIngestionPipelines()
	if err != nil {
		return nil, nil, nil, err
	}
	tasks, err := r.loadIngestionTasks()
	if err != nil {
		return nil, nil, nil, err
	}
	taskNodes, err := r.loadIngestionTaskNodes()
	if err != nil {
		return nil, nil, nil, err
	}
	return pipelines, tasks, taskNodes, nil
}

func (r *SQLIngestionRepository) loadIngestionPipelines() ([]domainingestion.Pipeline, error) {
	rows, err := r.database.Query(`
SELECT id, name, description, created_by, create_time, update_time
FROM agentrag_ingestion_pipelines
ORDER BY create_time ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	pipelines := []domainingestion.Pipeline{}
	for rows.Next() {
		var pipeline domainingestion.Pipeline
		if err := rows.Scan(&pipeline.ID, &pipeline.Name, &pipeline.Description, &pipeline.CreatedBy, &pipeline.CreateTime, &pipeline.UpdateTime); err != nil {
			return nil, err
		}
		pipelines = append(pipelines, pipeline)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	nodesByPipelineID, err := r.loadIngestionPipelineNodes()
	if err != nil {
		return nil, err
	}
	for index := range pipelines {
		pipelines[index].Nodes = nodesByPipelineID[pipelines[index].ID]
	}
	return pipelines, nil
}

func (r *SQLIngestionRepository) loadIngestionPipelineNodes() (map[string][]domainingestion.PipelineNode, error) {
	rows, err := r.database.Query(`
SELECT pipeline_id, id, node_id, node_type, settings, condition_json, next_node_id
FROM agentrag_ingestion_pipeline_nodes
ORDER BY pipeline_id ASC, node_order ASC, id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	nodesByPipelineID := map[string][]domainingestion.PipelineNode{}
	for rows.Next() {
		var node domainingestion.PipelineNode
		var pipelineID string
		var settingsJSON, conditionJSON string
		if err := rows.Scan(&pipelineID, &node.ID, &node.NodeID, &node.NodeType, &settingsJSON, &conditionJSON, &node.NextNodeID); err != nil {
			return nil, err
		}
		node.Settings = decodeJSONObject(settingsJSON)
		node.Condition = decodeJSONObject(conditionJSON)
		nodesByPipelineID[pipelineID] = append(nodesByPipelineID[pipelineID], node)
	}
	return nodesByPipelineID, rows.Err()
}

func (r *SQLIngestionRepository) loadIngestionTasks() ([]domainingestion.Task, error) {
	rows, err := r.database.Query(`
SELECT id, pipeline_id, source_type, source_location, source_file_name, status, chunk_count,
       error_message, logs, metadata, started_at, completed_at, created_by, create_time, update_time
FROM agentrag_ingestion_tasks
ORDER BY create_time DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tasks := []domainingestion.Task{}
	for rows.Next() {
		var task domainingestion.Task
		var logsJSON, metadataJSON string
		if err := rows.Scan(
			&task.ID,
			&task.PipelineID,
			&task.SourceType,
			&task.SourceLocation,
			&task.SourceFileName,
			&task.Status,
			&task.ChunkCount,
			&task.ErrorMessage,
			&logsJSON,
			&metadataJSON,
			&task.StartedAt,
			&task.CompletedAt,
			&task.CreatedBy,
			&task.CreateTime,
			&task.UpdateTime,
		); err != nil {
			return nil, err
		}
		task.Logs = decodeTaskLogs(logsJSON)
		task.Metadata = decodeJSONObject(metadataJSON)
		tasks = append(tasks, task)
	}
	return tasks, rows.Err()
}

func (r *SQLIngestionRepository) loadIngestionTaskNodes() ([]domainingestion.TaskNode, error) {
	rows, err := r.database.Query(`
SELECT id, task_id, pipeline_id, node_id, node_type, node_order, status, duration_ms,
       message, error_message, output, create_time, update_time
FROM agentrag_ingestion_task_nodes
ORDER BY task_id ASC, node_order ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	nodes := []domainingestion.TaskNode{}
	for rows.Next() {
		var node domainingestion.TaskNode
		var outputJSON string
		if err := rows.Scan(
			&node.ID,
			&node.TaskID,
			&node.PipelineID,
			&node.NodeID,
			&node.NodeType,
			&node.NodeOrder,
			&node.Status,
			&node.DurationMs,
			&node.Message,
			&node.ErrorMessage,
			&outputJSON,
			&node.CreateTime,
			&node.UpdateTime,
		); err != nil {
			return nil, err
		}
		node.Output = decodeJSONObject(outputJSON)
		nodes = append(nodes, node)
	}
	return nodes, rows.Err()
}

// SaveIngestionRecords 覆盖保存当前采集数据。
func (r *SQLIngestionRepository) SaveIngestionRecords(pipelines []domainingestion.Pipeline, tasks []domainingestion.Task, taskNodes []domainingestion.TaskNode) error {
	tx, err := r.database.Begin()
	if err != nil {
		return err
	}
	for _, statement := range []string{
		`DELETE FROM agentrag_ingestion_task_nodes`,
		`DELETE FROM agentrag_ingestion_tasks`,
		`DELETE FROM agentrag_ingestion_pipeline_nodes`,
		`DELETE FROM agentrag_ingestion_pipelines`,
	} {
		if _, err := tx.Exec(statement); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	if err := saveIngestionPipelines(tx, pipelines); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := saveIngestionTasks(tx, tasks); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := saveIngestionTaskNodes(tx, taskNodes); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func saveIngestionPipelines(tx *sql.Tx, pipelines []domainingestion.Pipeline) error {
	pipelineStmt, err := tx.Prepare(`
INSERT INTO agentrag_ingestion_pipelines
    (id, name, description, created_by, create_time, update_time)
VALUES (?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer pipelineStmt.Close()

	nodeStmt, err := tx.Prepare(`
INSERT INTO agentrag_ingestion_pipeline_nodes
    (id, pipeline_id, node_id, node_type, settings, condition_json, next_node_id, node_order)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer nodeStmt.Close()

	for _, pipeline := range pipelines {
		if _, err := pipelineStmt.Exec(
			pipeline.ID,
			pipeline.Name,
			pipeline.Description,
			pipeline.CreatedBy,
			normalizeTime(pipeline.CreateTime),
			normalizeTime(pipeline.UpdateTime),
		); err != nil {
			return err
		}
		for index, node := range pipeline.Nodes {
			if _, err := nodeStmt.Exec(
				node.ID,
				pipeline.ID,
				node.NodeID,
				node.NodeType,
				encodeJSON(node.Settings),
				encodeJSON(node.Condition),
				node.NextNodeID,
				index+1,
			); err != nil {
				return err
			}
		}
	}
	return nil
}

func saveIngestionTasks(tx *sql.Tx, tasks []domainingestion.Task) error {
	stmt, err := tx.Prepare(`
INSERT INTO agentrag_ingestion_tasks
    (id, pipeline_id, source_type, source_location, source_file_name, status, chunk_count,
     error_message, logs, metadata, started_at, completed_at, created_by, create_time, update_time)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, task := range tasks {
		if _, err := stmt.Exec(
			task.ID,
			task.PipelineID,
			task.SourceType,
			task.SourceLocation,
			task.SourceFileName,
			task.Status,
			task.ChunkCount,
			task.ErrorMessage,
			encodeJSON(task.Logs),
			encodeJSON(task.Metadata),
			normalizeTime(task.StartedAt),
			normalizeTime(task.CompletedAt),
			task.CreatedBy,
			normalizeTime(task.CreateTime),
			normalizeTime(task.UpdateTime),
		); err != nil {
			return err
		}
	}
	return nil
}

func saveIngestionTaskNodes(tx *sql.Tx, nodes []domainingestion.TaskNode) error {
	stmt, err := tx.Prepare(`
INSERT INTO agentrag_ingestion_task_nodes
    (id, task_id, pipeline_id, node_id, node_type, node_order, status, duration_ms,
     message, error_message, output, create_time, update_time)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, node := range nodes {
		if _, err := stmt.Exec(
			node.ID,
			node.TaskID,
			node.PipelineID,
			node.NodeID,
			node.NodeType,
			node.NodeOrder,
			node.Status,
			node.DurationMs,
			node.Message,
			node.ErrorMessage,
			encodeJSON(node.Output),
			normalizeTime(node.CreateTime),
			normalizeTime(node.UpdateTime),
		); err != nil {
			return err
		}
	}
	return nil
}

func encodeJSON(value any) string {
	if value == nil {
		return ""
	}
	content, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return string(content)
}

func decodeJSONObject(raw string) map[string]any {
	if raw == "" {
		return nil
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return nil
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func decodeTaskLogs(raw string) []domainingestion.TaskLog {
	if raw == "" {
		return nil
	}
	var result []domainingestion.TaskLog
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return nil
	}
	return result
}
