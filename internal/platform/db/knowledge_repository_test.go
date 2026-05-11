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

func TestSQLKnowledgeRepositoryReconcilesSavedRecords(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite database failed: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	repository := NewSQLKnowledgeRepository(database)
	if err := repository.Bootstrap(); err != nil {
		t.Fatalf("bootstrap knowledge tables failed: %v", err)
	}
	now := time.Date(2026, 5, 11, 10, 0, 0, 0, time.UTC)
	if err := repository.SaveKnowledgeRecords(
		[]domainknowledge.KnowledgeBase{
			{ID: "kb_keep", Name: "旧知识库", EmbeddingModel: "old-embedding", CollectionName: "old_collection", CreatedBy: "admin", DocumentCount: 1, CreateTime: now, UpdateTime: now},
			{ID: "kb_remove", Name: "删除知识库", EmbeddingModel: "old-embedding", CollectionName: "remove_collection", CreatedBy: "admin", DocumentCount: 1, CreateTime: now.Add(time.Minute), UpdateTime: now.Add(time.Minute)},
		},
		[]domainknowledge.KnowledgeDocument{
			{ID: "doc_keep", KBID: "kb_keep", DocName: "旧文档.md", SourceType: "file", SourceLocation: "old.md", TextContent: "旧正文", Enabled: true, ChunkCount: 1, FileURL: "memory://old", FileType: "md", FileSize: 100, ProcessMode: "chunk", ChunkStrategy: "fixed_size", ChunkConfig: `{"chunkSize":512}`, PipelineID: "pipe_old", Status: "pending", CreatedBy: "admin", UpdatedBy: "admin", CreateTime: now, UpdateTime: now},
			{ID: "doc_remove", KBID: "kb_remove", DocName: "删除文档.md", SourceType: "file", SourceLocation: "remove.md", TextContent: "删除正文", Enabled: true, ChunkCount: 1, CreateTime: now.Add(time.Minute), UpdateTime: now.Add(time.Minute)},
		},
		[]domainknowledge.KnowledgeChunk{
			{ID: "chunk_keep", KBID: "kb_keep", DocID: "doc_keep", ChunkIndex: 0, Content: "旧 Chunk", ContentHash: "old-hash", CharCount: 8, TokenCount: 2, Enabled: 1, CreateTime: now, UpdateTime: now},
			{ID: "chunk_remove", KBID: "kb_remove", DocID: "doc_remove", ChunkIndex: 0, Content: "删除 Chunk", ContentHash: "remove-hash", CharCount: 8, TokenCount: 2, Enabled: 1, CreateTime: now.Add(time.Minute), UpdateTime: now.Add(time.Minute)},
		},
		[]domainknowledge.KnowledgeDocumentChunkLog{
			{ID: "log_keep", DocID: "doc_keep", Status: "pending", ProcessMode: "chunk", ChunkStrategy: "fixed_size", PipelineID: "pipe_old", PipelineName: "旧流水线", ExtractDuration: 1, ChunkDuration: 2, EmbedDuration: 3, PersistDuration: 4, OtherDuration: 5, TotalDuration: 15, ChunkCount: 1, StartTime: now, EndTime: now.Add(15 * time.Millisecond), CreateTime: now},
			{ID: "log_remove", DocID: "doc_remove", Status: "success", ProcessMode: "chunk", ChunkStrategy: "fixed_size", PipelineID: "pipe_remove", PipelineName: "删除流水线", TotalDuration: 20, ChunkCount: 1, StartTime: now.Add(time.Minute), EndTime: now.Add(time.Minute + 20*time.Millisecond), CreateTime: now.Add(time.Minute)},
		},
	); err != nil {
		t.Fatalf("save initial knowledge snapshot failed: %v", err)
	}

	if err := repository.SaveKnowledgeRecords(
		[]domainknowledge.KnowledgeBase{
			{ID: "kb_keep", Name: "新知识库", EmbeddingModel: "new-embedding", CollectionName: "new_collection", CreatedBy: "owner", DocumentCount: 2, CreateTime: now, UpdateTime: now.Add(10 * time.Minute)},
			{ID: "kb_new", Name: "新增知识库", EmbeddingModel: "new-embedding", CollectionName: "insert_collection", CreatedBy: "admin", DocumentCount: 1, CreateTime: now.Add(11 * time.Minute), UpdateTime: now.Add(11 * time.Minute)},
		},
		[]domainknowledge.KnowledgeDocument{
			{ID: "doc_keep", KBID: "kb_keep", DocName: "新文档.md", SourceType: "url", SourceLocation: "https://example.com/new.md", TextContent: "新正文", ScheduleEnabled: 1, ScheduleCron: "0 0 * * *", Enabled: false, ChunkCount: 2, FileURL: "memory://new", FileType: "md", FileSize: 200, ProcessMode: "chunk", ChunkStrategy: "structure_aware", ChunkConfig: `{"targetChars":1400}`, PipelineID: "pipe_new", Status: "success", CreatedBy: "admin", UpdatedBy: "owner", CreateTime: now, UpdateTime: now.Add(10 * time.Minute)},
			{ID: "doc_new", KBID: "kb_new", DocName: "新增文档.md", SourceType: "file", SourceLocation: "new.md", TextContent: "新增正文", Enabled: true, ChunkCount: 1, CreateTime: now.Add(11 * time.Minute), UpdateTime: now.Add(11 * time.Minute)},
		},
		[]domainknowledge.KnowledgeChunk{
			{ID: "chunk_keep", KBID: "kb_keep", DocID: "doc_keep", ChunkIndex: 1, Content: "新 Chunk", ContentHash: "new-hash", CharCount: 9, TokenCount: 3, Enabled: 0, CreateTime: now, UpdateTime: now.Add(10 * time.Minute)},
			{ID: "chunk_new", KBID: "kb_new", DocID: "doc_new", ChunkIndex: 0, Content: "新增 Chunk", ContentHash: "insert-hash", CharCount: 10, TokenCount: 3, Enabled: 1, CreateTime: now.Add(11 * time.Minute), UpdateTime: now.Add(11 * time.Minute)},
		},
		[]domainknowledge.KnowledgeDocumentChunkLog{
			{ID: "log_keep", DocID: "doc_keep", Status: "success", ProcessMode: "chunk", ChunkStrategy: "structure_aware", PipelineID: "pipe_new", PipelineName: "新流水线", ExtractDuration: 10, ChunkDuration: 20, EmbedDuration: 30, PersistDuration: 40, OtherDuration: 50, TotalDuration: 150, ChunkCount: 2, ErrorMessage: "updated", StartTime: now, EndTime: now.Add(150 * time.Millisecond), CreateTime: now},
			{ID: "log_new", DocID: "doc_new", Status: "success", ProcessMode: "chunk", ChunkStrategy: "fixed_size", PipelineID: "pipe_insert", PipelineName: "新增流水线", TotalDuration: 25, ChunkCount: 1, StartTime: now.Add(11 * time.Minute), EndTime: now.Add(11*time.Minute + 25*time.Millisecond), CreateTime: now.Add(11 * time.Minute)},
		},
	); err != nil {
		t.Fatalf("save reconciled knowledge snapshot failed: %v", err)
	}

	bases, docs, chunks, logs, err := repository.LoadKnowledgeRecords()
	if err != nil {
		t.Fatalf("load reconciled knowledge records failed: %v", err)
	}
	if len(bases) != 2 || len(docs) != 2 || len(chunks) != 2 || len(logs) != 2 {
		t.Fatalf("expected 2 records per table after reconcile, got bases=%#v docs=%#v chunks=%#v logs=%#v", bases, docs, chunks, logs)
	}
	basesByID := map[string]domainknowledge.KnowledgeBase{}
	for _, item := range bases {
		basesByID[item.ID] = item
	}
	if _, ok := basesByID["kb_remove"]; ok {
		t.Fatal("expected missing knowledge base to be deleted")
	}
	if basesByID["kb_keep"].Name != "新知识库" || basesByID["kb_keep"].EmbeddingModel != "new-embedding" || basesByID["kb_keep"].DocumentCount != 2 {
		t.Fatalf("expected existing knowledge base to be updated, got %#v", basesByID["kb_keep"])
	}
	docsByID := map[string]domainknowledge.KnowledgeDocument{}
	for _, item := range docs {
		docsByID[item.ID] = item
	}
	if _, ok := docsByID["doc_remove"]; ok {
		t.Fatal("expected missing knowledge document to be deleted")
	}
	if docsByID["doc_keep"].DocName != "新文档.md" || docsByID["doc_keep"].TextContent != "新正文" || docsByID["doc_keep"].Enabled || docsByID["doc_keep"].ChunkStrategy != "structure_aware" {
		t.Fatalf("expected existing knowledge document to be updated, got %#v", docsByID["doc_keep"])
	}
	chunksByID := map[string]domainknowledge.KnowledgeChunk{}
	for _, item := range chunks {
		chunksByID[item.ID] = item
	}
	if _, ok := chunksByID["chunk_remove"]; ok {
		t.Fatal("expected missing knowledge chunk to be deleted")
	}
	if chunksByID["chunk_keep"].Content != "新 Chunk" || chunksByID["chunk_keep"].Enabled != 0 || chunksByID["chunk_keep"].ChunkIndex != 1 {
		t.Fatalf("expected existing knowledge chunk to be updated, got %#v", chunksByID["chunk_keep"])
	}
	logsByID := map[string]domainknowledge.KnowledgeDocumentChunkLog{}
	for _, item := range logs {
		logsByID[item.ID] = item
	}
	if _, ok := logsByID["log_remove"]; ok {
		t.Fatal("expected missing knowledge chunk log to be deleted")
	}
	if logsByID["log_keep"].Status != "success" || logsByID["log_keep"].PersistDuration != 40 || logsByID["log_keep"].ErrorMessage != "updated" {
		t.Fatalf("expected existing knowledge chunk log to be updated, got %#v", logsByID["log_keep"])
	}

	if err := repository.SaveKnowledgeRecords(nil, nil, nil, nil); err != nil {
		t.Fatalf("save empty knowledge snapshot failed: %v", err)
	}
	bases, docs, chunks, logs, err = repository.LoadKnowledgeRecords()
	if err != nil {
		t.Fatalf("load empty knowledge snapshot failed: %v", err)
	}
	if len(bases) != 0 || len(docs) != 0 || len(chunks) != 0 || len(logs) != 0 {
		t.Fatalf("expected empty knowledge snapshot to clear records, got bases=%#v docs=%#v chunks=%#v logs=%#v", bases, docs, chunks, logs)
	}
}

func TestSQLKnowledgeRepositoryRejectsDuplicateSnapshotIDs(t *testing.T) {
	now := time.Date(2026, 5, 11, 11, 0, 0, 0, time.UTC)
	testCases := []struct {
		name   string
		bases  []domainknowledge.KnowledgeBase
		docs   []domainknowledge.KnowledgeDocument
		chunks []domainknowledge.KnowledgeChunk
		logs   []domainknowledge.KnowledgeDocumentChunkLog
	}{
		{
			name: "bases",
			bases: []domainknowledge.KnowledgeBase{
				{ID: "kb_keep", Name: "知识库 A", EmbeddingModel: "embedding", CollectionName: "collection", CreateTime: now, UpdateTime: now},
				{ID: "kb_keep", Name: "知识库 B", EmbeddingModel: "embedding", CollectionName: "collection", CreateTime: now, UpdateTime: now},
			},
		},
		{
			name:  "documents",
			bases: []domainknowledge.KnowledgeBase{{ID: "kb_keep", Name: "知识库", EmbeddingModel: "embedding", CollectionName: "collection", CreateTime: now, UpdateTime: now}},
			docs: []domainknowledge.KnowledgeDocument{
				{ID: "doc_keep", KBID: "kb_keep", DocName: "文档 A", Enabled: true, CreateTime: now, UpdateTime: now},
				{ID: "doc_keep", KBID: "kb_keep", DocName: "文档 B", Enabled: true, CreateTime: now, UpdateTime: now},
			},
		},
		{
			name:  "chunks",
			bases: []domainknowledge.KnowledgeBase{{ID: "kb_keep", Name: "知识库", EmbeddingModel: "embedding", CollectionName: "collection", CreateTime: now, UpdateTime: now}},
			docs:  []domainknowledge.KnowledgeDocument{{ID: "doc_keep", KBID: "kb_keep", DocName: "文档", Enabled: true, CreateTime: now, UpdateTime: now}},
			chunks: []domainknowledge.KnowledgeChunk{
				{ID: "chunk_keep", KBID: "kb_keep", DocID: "doc_keep", Content: "Chunk A", Enabled: 1, CreateTime: now, UpdateTime: now},
				{ID: "chunk_keep", KBID: "kb_keep", DocID: "doc_keep", Content: "Chunk B", Enabled: 1, CreateTime: now, UpdateTime: now},
			},
		},
		{
			name:  "logs",
			bases: []domainknowledge.KnowledgeBase{{ID: "kb_keep", Name: "知识库", EmbeddingModel: "embedding", CollectionName: "collection", CreateTime: now, UpdateTime: now}},
			docs:  []domainknowledge.KnowledgeDocument{{ID: "doc_keep", KBID: "kb_keep", DocName: "文档", Enabled: true, CreateTime: now, UpdateTime: now}},
			chunks: []domainknowledge.KnowledgeChunk{
				{ID: "chunk_keep", KBID: "kb_keep", DocID: "doc_keep", Content: "Chunk", Enabled: 1, CreateTime: now, UpdateTime: now},
			},
			logs: []domainknowledge.KnowledgeDocumentChunkLog{
				{ID: "log_keep", DocID: "doc_keep", Status: "success", StartTime: now, EndTime: now, CreateTime: now},
				{ID: "log_keep", DocID: "doc_keep", Status: "failed", StartTime: now, EndTime: now, CreateTime: now},
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

			repository := NewSQLKnowledgeRepository(database)
			if err := repository.Bootstrap(); err != nil {
				t.Fatalf("bootstrap knowledge tables failed: %v", err)
			}
			if err := repository.SaveKnowledgeRecords(
				[]domainknowledge.KnowledgeBase{{ID: "kb_old", Name: "旧知识库", EmbeddingModel: "embedding", CollectionName: "collection", CreateTime: now, UpdateTime: now}},
				[]domainknowledge.KnowledgeDocument{{ID: "doc_old", KBID: "kb_old", DocName: "旧文档", Enabled: true, CreateTime: now, UpdateTime: now}},
				[]domainknowledge.KnowledgeChunk{{ID: "chunk_old", KBID: "kb_old", DocID: "doc_old", Content: "旧 Chunk", Enabled: 1, CreateTime: now, UpdateTime: now}},
				[]domainknowledge.KnowledgeDocumentChunkLog{{ID: "log_old", DocID: "doc_old", Status: "success", StartTime: now, EndTime: now, CreateTime: now}},
			); err != nil {
				t.Fatalf("save initial knowledge snapshot failed: %v", err)
			}

			err = repository.SaveKnowledgeRecords(testCase.bases, testCase.docs, testCase.chunks, testCase.logs)
			if err == nil {
				t.Fatal("expected duplicate id error")
			}

			bases, docs, chunks, logs, err := repository.LoadKnowledgeRecords()
			if err != nil {
				t.Fatalf("load knowledge records after duplicate id failure failed: %v", err)
			}
			if len(bases) != 1 || bases[0].ID != "kb_old" || len(docs) != 1 || docs[0].ID != "doc_old" || len(chunks) != 1 || chunks[0].ID != "chunk_old" || len(logs) != 1 || logs[0].ID != "log_old" {
				t.Fatalf("expected existing knowledge records to remain unchanged, got bases=%#v docs=%#v chunks=%#v logs=%#v", bases, docs, chunks, logs)
			}
		})
	}
}
