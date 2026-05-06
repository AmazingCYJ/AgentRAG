package db

import (
	"database/sql"
	"testing"
	"time"

	domainknowledge "github.com/AmazingCYJ/AgentRAG/internal/domain/knowledge"
	_ "modernc.org/sqlite"
)

func TestSQLKnowledgeRepositorySavesAndLoadsRecords(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite database failed: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	repository := NewSQLKnowledgeRepository(database)
	if err := repository.Bootstrap(); err != nil {
		t.Fatalf("bootstrap knowledge tables failed: %v", err)
	}

	now := time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC)
	base := domainknowledge.KnowledgeBase{
		ID:             "kb_sql",
		Name:           "SQL 知识库",
		EmbeddingModel: "embedding-openai-large",
		CollectionName: "sqldocs",
		CreatedBy:      "admin",
		DocumentCount:  1,
		CreateTime:     now,
		UpdateTime:     now,
	}
	doc := domainknowledge.KnowledgeDocument{
		ID:              "doc_sql",
		KBID:            base.ID,
		DocName:         "SQL 文档.md",
		SourceType:      "file",
		SourceLocation:  "SQL 文档.md",
		TextContent:     "SQL 文档正文内容",
		ScheduleEnabled: 1,
		ScheduleCron:    "0 0 * * *",
		Enabled:         true,
		ChunkCount:      1,
		FileURL:         "memory://doc_sql/SQL 文档.md",
		FileType:        "md",
		FileSize:        1024,
		ProcessMode:     "chunk",
		ChunkStrategy:   "structure_aware",
		ChunkConfig:     `{"targetChars":1400}`,
		PipelineID:      "pipe_sql",
		Status:          "success",
		CreatedBy:       "admin",
		UpdatedBy:       "admin",
		CreateTime:      now,
		UpdateTime:      now,
	}
	chunk := domainknowledge.KnowledgeChunk{
		ID:          "chunk_sql",
		KBID:        base.ID,
		DocID:       doc.ID,
		ChunkIndex:  0,
		Content:     "SQL Chunk 内容",
		ContentHash: "SQL Chunk 内容",
		CharCount:   12,
		TokenCount:  3,
		Enabled:     1,
		CreateTime:  now,
		UpdateTime:  now,
	}
	log := domainknowledge.KnowledgeDocumentChunkLog{
		ID:              "log_sql",
		DocID:           doc.ID,
		Status:          "success",
		ProcessMode:     "chunk",
		ChunkStrategy:   "structure_aware",
		PipelineID:      "pipe_sql",
		PipelineName:    "默认流水线",
		ExtractDuration: 10,
		ChunkDuration:   20,
		EmbedDuration:   30,
		PersistDuration: 40,
		OtherDuration:   5,
		TotalDuration:   105,
		ChunkCount:      1,
		StartTime:       now,
		EndTime:         now.Add(105 * time.Millisecond),
		CreateTime:      now,
	}
	if err := repository.SaveKnowledgeRecords(
		[]domainknowledge.KnowledgeBase{base},
		[]domainknowledge.KnowledgeDocument{doc},
		[]domainknowledge.KnowledgeChunk{chunk},
		[]domainknowledge.KnowledgeDocumentChunkLog{log},
	); err != nil {
		t.Fatalf("save knowledge records failed: %v", err)
	}

	bases, docs, chunks, logs, err := repository.LoadKnowledgeRecords()
	if err != nil {
		t.Fatalf("load knowledge records failed: %v", err)
	}
	if len(bases) != 1 || bases[0].Name != "SQL 知识库" {
		t.Fatalf("unexpected bases %#v", bases)
	}
	if len(docs) != 1 || docs[0].DocName != "SQL 文档.md" || docs[0].TextContent != "SQL 文档正文内容" || !docs[0].Enabled {
		t.Fatalf("unexpected docs %#v", docs)
	}
	if len(chunks) != 1 || chunks[0].Content != "SQL Chunk 内容" {
		t.Fatalf("unexpected chunks %#v", chunks)
	}
	if len(logs) != 1 || logs[0].TotalDuration != 105 {
		t.Fatalf("unexpected logs %#v", logs)
	}
}
