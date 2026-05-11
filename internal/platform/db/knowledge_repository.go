package db

import (
	"database/sql"
	"encoding/json"

	domainknowledge "github.com/AmazingCYJ/AgentRAG/internal/domain/knowledge"
)

// SQLKnowledgeRepository 使用关系型数据库持久化知识库、文档、Chunk 和处理日志。
type SQLKnowledgeRepository struct {
	database *SQLDB
}

// NewSQLKnowledgeRepository 创建知识库 SQL 仓储。
func NewSQLKnowledgeRepository(database *sql.DB) *SQLKnowledgeRepository {
	return &SQLKnowledgeRepository{database: newSQLDB(database)}
}

// Bootstrap 初始化知识库相关表结构。
func (r *SQLKnowledgeRepository) Bootstrap() error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS agentrag_knowledge_bases (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    embedding_model TEXT NOT NULL,
    collection_name TEXT NOT NULL,
    created_by TEXT NOT NULL DEFAULT '',
    document_count INTEGER NOT NULL DEFAULT 0,
    create_time TIMESTAMP NOT NULL,
    update_time TIMESTAMP NOT NULL
)`,
		`CREATE TABLE IF NOT EXISTS agentrag_knowledge_documents (
    id TEXT PRIMARY KEY,
    kb_id TEXT NOT NULL,
    doc_name TEXT NOT NULL,
    source_type TEXT NOT NULL DEFAULT '',
    source_location TEXT NOT NULL DEFAULT '',
    text_content TEXT NOT NULL DEFAULT '',
    schedule_enabled INTEGER NOT NULL DEFAULT 0,
    schedule_cron TEXT NOT NULL DEFAULT '',
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    chunk_count INTEGER NOT NULL DEFAULT 0,
    file_url TEXT NOT NULL DEFAULT '',
    file_type TEXT NOT NULL DEFAULT '',
    file_size INTEGER NOT NULL DEFAULT 0,
    process_mode TEXT NOT NULL DEFAULT '',
    chunk_strategy TEXT NOT NULL DEFAULT '',
    chunk_config TEXT NOT NULL DEFAULT '',
    pipeline_id TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT '',
    created_by TEXT NOT NULL DEFAULT '',
    updated_by TEXT NOT NULL DEFAULT '',
    create_time TIMESTAMP NOT NULL,
    update_time TIMESTAMP NOT NULL
)`,
		`CREATE TABLE IF NOT EXISTS agentrag_knowledge_chunks (
    id TEXT PRIMARY KEY,
    kb_id TEXT NOT NULL DEFAULT '',
    doc_id TEXT NOT NULL,
    chunk_index INTEGER NOT NULL DEFAULT 0,
    content TEXT NOT NULL DEFAULT '',
    content_hash TEXT NOT NULL DEFAULT '',
    char_count INTEGER NOT NULL DEFAULT 0,
    token_count INTEGER NOT NULL DEFAULT 0,
    embedding_model TEXT NOT NULL DEFAULT '',
    embedding_vector TEXT NOT NULL DEFAULT '',
    enabled INTEGER NOT NULL DEFAULT 1,
    create_time TIMESTAMP NOT NULL,
    update_time TIMESTAMP NOT NULL
)`,
		`CREATE TABLE IF NOT EXISTS agentrag_knowledge_chunk_logs (
    id TEXT PRIMARY KEY,
    doc_id TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT '',
    process_mode TEXT NOT NULL DEFAULT '',
    chunk_strategy TEXT NOT NULL DEFAULT '',
    pipeline_id TEXT NOT NULL DEFAULT '',
    pipeline_name TEXT NOT NULL DEFAULT '',
    extract_duration INTEGER NOT NULL DEFAULT 0,
    chunk_duration INTEGER NOT NULL DEFAULT 0,
    embed_duration INTEGER NOT NULL DEFAULT 0,
    persist_duration INTEGER NOT NULL DEFAULT 0,
    other_duration INTEGER NOT NULL DEFAULT 0,
    total_duration INTEGER NOT NULL DEFAULT 0,
    chunk_count INTEGER NOT NULL DEFAULT 0,
    error_message TEXT NOT NULL DEFAULT '',
    start_time TIMESTAMP NOT NULL,
    end_time TIMESTAMP NOT NULL,
    create_time TIMESTAMP NOT NULL
)`,
	}
	for _, statement := range statements {
		if _, err := r.database.Exec(statement); err != nil {
			return err
		}
	}
	// 兼容已存在的早期本地库；列已存在时忽略错误。
	_, _ = r.database.Exec(addColumnSQL(r.database.dialect, "agentrag_knowledge_documents", `text_content TEXT NOT NULL DEFAULT ''`))
	_, _ = r.database.Exec(addColumnSQL(r.database.dialect, "agentrag_knowledge_chunks", `embedding_model TEXT NOT NULL DEFAULT ''`))
	_, _ = r.database.Exec(addColumnSQL(r.database.dialect, "agentrag_knowledge_chunks", `embedding_vector TEXT NOT NULL DEFAULT ''`))
	return nil
}

// LoadKnowledgeRecords 从数据库加载全部知识库数据。
func (r *SQLKnowledgeRepository) LoadKnowledgeRecords() ([]domainknowledge.KnowledgeBase, []domainknowledge.KnowledgeDocument, []domainknowledge.KnowledgeChunk, []domainknowledge.KnowledgeDocumentChunkLog, error) {
	bases, err := r.loadKnowledgeBases()
	if err != nil {
		return nil, nil, nil, nil, err
	}
	docs, err := r.loadKnowledgeDocuments()
	if err != nil {
		return nil, nil, nil, nil, err
	}
	chunks, err := r.loadKnowledgeChunks()
	if err != nil {
		return nil, nil, nil, nil, err
	}
	logs, err := r.loadKnowledgeLogs()
	if err != nil {
		return nil, nil, nil, nil, err
	}
	return bases, docs, chunks, logs, nil
}

func (r *SQLKnowledgeRepository) loadKnowledgeBases() ([]domainknowledge.KnowledgeBase, error) {
	rows, err := r.database.Query(`
SELECT id, name, embedding_model, collection_name, created_by, document_count, create_time, update_time
FROM agentrag_knowledge_bases
ORDER BY create_time DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []domainknowledge.KnowledgeBase{}
	for rows.Next() {
		var item domainknowledge.KnowledgeBase
		if err := rows.Scan(&item.ID, &item.Name, &item.EmbeddingModel, &item.CollectionName, &item.CreatedBy, &item.DocumentCount, &item.CreateTime, &item.UpdateTime); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *SQLKnowledgeRepository) loadKnowledgeDocuments() ([]domainknowledge.KnowledgeDocument, error) {
	rows, err := r.database.Query(`
SELECT id, kb_id, doc_name, source_type, source_location, text_content, schedule_enabled, schedule_cron,
       enabled, chunk_count, file_url, file_type, file_size, process_mode, chunk_strategy,
       chunk_config, pipeline_id, status, created_by, updated_by, create_time, update_time
FROM agentrag_knowledge_documents
ORDER BY create_time DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []domainknowledge.KnowledgeDocument{}
	for rows.Next() {
		var item domainknowledge.KnowledgeDocument
		if err := rows.Scan(
			&item.ID,
			&item.KBID,
			&item.DocName,
			&item.SourceType,
			&item.SourceLocation,
			&item.TextContent,
			&item.ScheduleEnabled,
			&item.ScheduleCron,
			&item.Enabled,
			&item.ChunkCount,
			&item.FileURL,
			&item.FileType,
			&item.FileSize,
			&item.ProcessMode,
			&item.ChunkStrategy,
			&item.ChunkConfig,
			&item.PipelineID,
			&item.Status,
			&item.CreatedBy,
			&item.UpdatedBy,
			&item.CreateTime,
			&item.UpdateTime,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *SQLKnowledgeRepository) loadKnowledgeChunks() ([]domainknowledge.KnowledgeChunk, error) {
	rows, err := r.database.Query(`
SELECT id, kb_id, doc_id, chunk_index, content, content_hash, char_count, token_count,
       embedding_model, embedding_vector, enabled, create_time, update_time
FROM agentrag_knowledge_chunks
ORDER BY doc_id ASC, chunk_index ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []domainknowledge.KnowledgeChunk{}
	for rows.Next() {
		var item domainknowledge.KnowledgeChunk
		var embeddingVector string
		if err := rows.Scan(&item.ID, &item.KBID, &item.DocID, &item.ChunkIndex, &item.Content, &item.ContentHash, &item.CharCount, &item.TokenCount, &item.EmbeddingModel, &embeddingVector, &item.Enabled, &item.CreateTime, &item.UpdateTime); err != nil {
			return nil, err
		}
		item.Embedding = decodeEmbeddingVector(embeddingVector)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *SQLKnowledgeRepository) loadKnowledgeLogs() ([]domainknowledge.KnowledgeDocumentChunkLog, error) {
	rows, err := r.database.Query(`
SELECT id, doc_id, status, process_mode, chunk_strategy, pipeline_id, pipeline_name,
       extract_duration, chunk_duration, embed_duration, persist_duration, other_duration,
       total_duration, chunk_count, error_message, start_time, end_time, create_time
FROM agentrag_knowledge_chunk_logs
ORDER BY doc_id ASC, create_time DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []domainknowledge.KnowledgeDocumentChunkLog{}
	for rows.Next() {
		var item domainknowledge.KnowledgeDocumentChunkLog
		if err := rows.Scan(
			&item.ID,
			&item.DocID,
			&item.Status,
			&item.ProcessMode,
			&item.ChunkStrategy,
			&item.PipelineID,
			&item.PipelineName,
			&item.ExtractDuration,
			&item.ChunkDuration,
			&item.EmbedDuration,
			&item.PersistDuration,
			&item.OtherDuration,
			&item.TotalDuration,
			&item.ChunkCount,
			&item.ErrorMessage,
			&item.StartTime,
			&item.EndTime,
			&item.CreateTime,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// SaveKnowledgeRecords 覆盖保存当前知识库数据。
func (r *SQLKnowledgeRepository) SaveKnowledgeRecords(bases []domainknowledge.KnowledgeBase, docs []domainknowledge.KnowledgeDocument, chunks []domainknowledge.KnowledgeChunk, logs []domainknowledge.KnowledgeDocumentChunkLog) error {
	baseIDs := make([]string, 0, len(bases))
	for _, item := range bases {
		baseIDs = append(baseIDs, item.ID)
	}
	if err := rejectDuplicateIDs(baseIDs); err != nil {
		return err
	}
	docIDs := make([]string, 0, len(docs))
	for _, item := range docs {
		docIDs = append(docIDs, item.ID)
	}
	if err := rejectDuplicateIDs(docIDs); err != nil {
		return err
	}
	chunkIDs := make([]string, 0, len(chunks))
	for _, item := range chunks {
		chunkIDs = append(chunkIDs, item.ID)
	}
	if err := rejectDuplicateIDs(chunkIDs); err != nil {
		return err
	}
	logIDs := make([]string, 0, len(logs))
	for _, item := range logs {
		logIDs = append(logIDs, item.ID)
	}
	if err := rejectDuplicateIDs(logIDs); err != nil {
		return err
	}

	tx, err := r.database.Begin()
	if err != nil {
		return err
	}
	deleteSteps := []struct {
		tableName string
		ids       []string
	}{
		{tableName: "agentrag_knowledge_chunk_logs", ids: logIDs},
		{tableName: "agentrag_knowledge_chunks", ids: chunkIDs},
		{tableName: "agentrag_knowledge_documents", ids: docIDs},
		{tableName: "agentrag_knowledge_bases", ids: baseIDs},
	}
	for _, deleteStep := range deleteSteps {
		if err := deleteMissingRows(tx, deleteStep.tableName, "id", deleteStep.ids); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	if err := saveKnowledgeBases(tx, bases); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := saveKnowledgeDocuments(tx, docs); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := saveKnowledgeChunks(tx, chunks); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := saveKnowledgeLogs(tx, logs); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func saveKnowledgeBases(tx *SQLTx, items []domainknowledge.KnowledgeBase) error {
	stmt, err := tx.Prepare(`
INSERT INTO agentrag_knowledge_bases
    (id, name, embedding_model, collection_name, created_by, document_count, create_time, update_time)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (id) DO UPDATE SET
    name = excluded.name,
    embedding_model = excluded.embedding_model,
    collection_name = excluded.collection_name,
    created_by = excluded.created_by,
    document_count = excluded.document_count,
    create_time = excluded.create_time,
    update_time = excluded.update_time`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, item := range items {
		if _, err := stmt.Exec(item.ID, item.Name, item.EmbeddingModel, item.CollectionName, item.CreatedBy, item.DocumentCount, normalizeTime(item.CreateTime), normalizeTime(item.UpdateTime)); err != nil {
			return err
		}
	}
	return nil
}

func saveKnowledgeDocuments(tx *SQLTx, items []domainknowledge.KnowledgeDocument) error {
	stmt, err := tx.Prepare(`
INSERT INTO agentrag_knowledge_documents
    (id, kb_id, doc_name, source_type, source_location, text_content, schedule_enabled, schedule_cron,
     enabled, chunk_count, file_url, file_type, file_size, process_mode, chunk_strategy,
     chunk_config, pipeline_id, status, created_by, updated_by, create_time, update_time)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (id) DO UPDATE SET
    kb_id = excluded.kb_id,
    doc_name = excluded.doc_name,
    source_type = excluded.source_type,
    source_location = excluded.source_location,
    text_content = excluded.text_content,
    schedule_enabled = excluded.schedule_enabled,
    schedule_cron = excluded.schedule_cron,
    enabled = excluded.enabled,
    chunk_count = excluded.chunk_count,
    file_url = excluded.file_url,
    file_type = excluded.file_type,
    file_size = excluded.file_size,
    process_mode = excluded.process_mode,
    chunk_strategy = excluded.chunk_strategy,
    chunk_config = excluded.chunk_config,
    pipeline_id = excluded.pipeline_id,
    status = excluded.status,
    created_by = excluded.created_by,
    updated_by = excluded.updated_by,
    create_time = excluded.create_time,
    update_time = excluded.update_time`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, item := range items {
		if _, err := stmt.Exec(
			item.ID,
			item.KBID,
			item.DocName,
			item.SourceType,
			item.SourceLocation,
			item.TextContent,
			item.ScheduleEnabled,
			item.ScheduleCron,
			item.Enabled,
			item.ChunkCount,
			item.FileURL,
			item.FileType,
			item.FileSize,
			item.ProcessMode,
			item.ChunkStrategy,
			item.ChunkConfig,
			item.PipelineID,
			item.Status,
			item.CreatedBy,
			item.UpdatedBy,
			normalizeTime(item.CreateTime),
			normalizeTime(item.UpdateTime),
		); err != nil {
			return err
		}
	}
	return nil
}

func saveKnowledgeChunks(tx *SQLTx, items []domainknowledge.KnowledgeChunk) error {
	stmt, err := tx.Prepare(`
INSERT INTO agentrag_knowledge_chunks
    (id, kb_id, doc_id, chunk_index, content, content_hash, char_count, token_count,
     embedding_model, embedding_vector, enabled, create_time, update_time)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (id) DO UPDATE SET
    kb_id = excluded.kb_id,
    doc_id = excluded.doc_id,
    chunk_index = excluded.chunk_index,
    content = excluded.content,
    content_hash = excluded.content_hash,
    char_count = excluded.char_count,
    token_count = excluded.token_count,
    embedding_model = excluded.embedding_model,
    embedding_vector = excluded.embedding_vector,
    enabled = excluded.enabled,
    create_time = excluded.create_time,
    update_time = excluded.update_time`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, item := range items {
		if _, err := stmt.Exec(item.ID, item.KBID, item.DocID, item.ChunkIndex, item.Content, item.ContentHash, item.CharCount, item.TokenCount, item.EmbeddingModel, encodeEmbeddingVector(item.Embedding), item.Enabled, normalizeTime(item.CreateTime), normalizeTime(item.UpdateTime)); err != nil {
			return err
		}
	}
	return nil
}

func encodeEmbeddingVector(values []float64) string {
	if len(values) == 0 {
		return ""
	}
	content, err := json.Marshal(values)
	if err != nil {
		return ""
	}
	return string(content)
}

func decodeEmbeddingVector(content string) []float64 {
	if content == "" {
		return nil
	}
	var values []float64
	if err := json.Unmarshal([]byte(content), &values); err != nil {
		return nil
	}
	return values
}

func saveKnowledgeLogs(tx *SQLTx, items []domainknowledge.KnowledgeDocumentChunkLog) error {
	stmt, err := tx.Prepare(`
INSERT INTO agentrag_knowledge_chunk_logs
    (id, doc_id, status, process_mode, chunk_strategy, pipeline_id, pipeline_name,
     extract_duration, chunk_duration, embed_duration, persist_duration, other_duration,
     total_duration, chunk_count, error_message, start_time, end_time, create_time)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (id) DO UPDATE SET
    doc_id = excluded.doc_id,
    status = excluded.status,
    process_mode = excluded.process_mode,
    chunk_strategy = excluded.chunk_strategy,
    pipeline_id = excluded.pipeline_id,
    pipeline_name = excluded.pipeline_name,
    extract_duration = excluded.extract_duration,
    chunk_duration = excluded.chunk_duration,
    embed_duration = excluded.embed_duration,
    persist_duration = excluded.persist_duration,
    other_duration = excluded.other_duration,
    total_duration = excluded.total_duration,
    chunk_count = excluded.chunk_count,
    error_message = excluded.error_message,
    start_time = excluded.start_time,
    end_time = excluded.end_time,
    create_time = excluded.create_time`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, item := range items {
		if _, err := stmt.Exec(
			item.ID,
			item.DocID,
			item.Status,
			item.ProcessMode,
			item.ChunkStrategy,
			item.PipelineID,
			item.PipelineName,
			item.ExtractDuration,
			item.ChunkDuration,
			item.EmbedDuration,
			item.PersistDuration,
			item.OtherDuration,
			item.TotalDuration,
			item.ChunkCount,
			item.ErrorMessage,
			normalizeTime(item.StartTime),
			normalizeTime(item.EndTime),
			normalizeTime(item.CreateTime),
		); err != nil {
			return err
		}
	}
	return nil
}
