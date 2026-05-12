package db

import (
	"context"
	"database/sql"
	"sort"
	"strings"
	"time"

	domainknowledge "github.com/AmazingCYJ/AgentRAG/internal/domain/knowledge"
)

// SQLKnowledgeVectorStore 使用关系型数据库保存知识向量。
type SQLKnowledgeVectorStore struct {
	database *SQLDB
	now      func() time.Time
}

// NewSQLKnowledgeVectorStore 创建知识向量 SQL 存储。
func NewSQLKnowledgeVectorStore(database *sql.DB) *SQLKnowledgeVectorStore {
	return &SQLKnowledgeVectorStore{
		database: newSQLDB(database),
		now:      time.Now,
	}
}

// Bootstrap 初始化知识向量表结构。
func (s *SQLKnowledgeVectorStore) Bootstrap() error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS agentrag_knowledge_vectors (
    id TEXT PRIMARY KEY,
    kb_id TEXT NOT NULL DEFAULT '',
    doc_id TEXT NOT NULL DEFAULT '',
    collection_name TEXT NOT NULL DEFAULT '',
    chunk_id TEXT NOT NULL DEFAULT '',
    chunk_index INTEGER NOT NULL DEFAULT 0,
    content TEXT NOT NULL DEFAULT '',
    embedding_model TEXT NOT NULL DEFAULT '',
    embedding_vector TEXT NOT NULL DEFAULT '',
    create_time TIMESTAMP NOT NULL,
    update_time TIMESTAMP NOT NULL
)`,
		`CREATE INDEX IF NOT EXISTS idx_agentrag_knowledge_vectors_collection
ON agentrag_knowledge_vectors (collection_name, embedding_model)`,
		`CREATE INDEX IF NOT EXISTS idx_agentrag_knowledge_vectors_chunk
ON agentrag_knowledge_vectors (chunk_id)`,
	}
	for _, statement := range statements {
		if _, err := s.database.Exec(statement); err != nil {
			return err
		}
	}
	return nil
}

// Upsert 写入或更新知识向量。
func (s *SQLKnowledgeVectorStore) Upsert(ctx context.Context, vectors []domainknowledge.KnowledgeVector) error {
	ctx = defaultContext(ctx)
	if err := ctx.Err(); err != nil {
		return err
	}
	tx, err := s.database.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare(`
INSERT INTO agentrag_knowledge_vectors
    (id, kb_id, doc_id, collection_name, chunk_id, chunk_index, content,
     embedding_model, embedding_vector, create_time, update_time)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (id) DO UPDATE SET
    kb_id = excluded.kb_id,
    doc_id = excluded.doc_id,
    collection_name = excluded.collection_name,
    chunk_id = excluded.chunk_id,
    chunk_index = excluded.chunk_index,
    content = excluded.content,
    embedding_model = excluded.embedding_model,
    embedding_vector = excluded.embedding_vector,
    update_time = excluded.update_time`)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	defer stmt.Close()

	now := s.now()
	for _, vector := range vectors {
		if err := ctx.Err(); err != nil {
			_ = tx.Rollback()
			return err
		}
		if _, err := stmt.Exec(
			vector.ID,
			vector.KBID,
			vector.DocID,
			vector.CollectionName,
			vector.ChunkID,
			vector.ChunkIndex,
			vector.Content,
			vector.EmbeddingModel,
			encodeEmbeddingVector(vector.Embedding),
			normalizeTime(now),
			normalizeTime(now),
		); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

// Delete 删除指定知识向量。
func (s *SQLKnowledgeVectorStore) Delete(ctx context.Context, ids []string) error {
	ctx = defaultContext(ctx)
	if err := ctx.Err(); err != nil {
		return err
	}
	if len(ids) == 0 {
		return nil
	}
	_, err := s.database.Exec(
		"DELETE FROM agentrag_knowledge_vectors WHERE id IN ("+questionPlaceholders(len(ids))+")",
		stringArgs(ids)...,
	)
	return err
}

// Search 搜索相似知识向量。
func (s *SQLKnowledgeVectorStore) Search(ctx context.Context, request domainknowledge.VectorSearchRequest) ([]domainknowledge.VectorSearchResult, error) {
	ctx = defaultContext(ctx)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(request.Embedding) == 0 {
		return nil, nil
	}

	query := `
SELECT id, kb_id, doc_id, collection_name, chunk_id, chunk_index, content, embedding_model, embedding_vector
FROM agentrag_knowledge_vectors`
	args := []any{}
	conditions := []string{}
	if strings.TrimSpace(request.CollectionName) != "" {
		conditions = append(conditions, "collection_name = ?")
		args = append(args, strings.TrimSpace(request.CollectionName))
	}
	if strings.TrimSpace(request.EmbeddingModel) != "" {
		conditions = append(conditions, "embedding_model = ?")
		args = append(args, strings.TrimSpace(request.EmbeddingModel))
	}
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY chunk_id ASC"

	rows, err := s.database.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := []domainknowledge.VectorSearchResult{}
	for rows.Next() {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		var vector domainknowledge.KnowledgeVector
		var embeddingVector string
		if err := rows.Scan(
			&vector.ID,
			&vector.KBID,
			&vector.DocID,
			&vector.CollectionName,
			&vector.ChunkID,
			&vector.ChunkIndex,
			&vector.Content,
			&vector.EmbeddingModel,
			&embeddingVector,
		); err != nil {
			return nil, err
		}
		vector.Embedding = decodeEmbeddingVector(embeddingVector)
		if len(vector.Embedding) != len(request.Embedding) {
			continue
		}
		results = append(results, domainknowledge.VectorSearchResult{
			Vector: vector,
			Score:  cosineScore(request.Embedding, vector.Embedding),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].Score == results[j].Score {
			return results[i].Vector.ChunkID < results[j].Vector.ChunkID
		}
		return results[i].Score > results[j].Score
	})
	limit := request.Limit
	if limit <= 0 {
		limit = 4
	}
	if len(results) > limit {
		results = results[:limit]
	}
	return results, nil
}

func cosineScore(left, right []float64) float64 {
	if len(left) == 0 || len(left) != len(right) {
		return 0
	}
	var score float64
	for index := range left {
		score += left[index] * right[index]
	}
	return score
}

func defaultContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}
