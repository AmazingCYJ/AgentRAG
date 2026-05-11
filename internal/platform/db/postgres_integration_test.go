package db

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	domainconversation "github.com/AmazingCYJ/AgentRAG/internal/domain/conversation"
	domainknowledge "github.com/AmazingCYJ/AgentRAG/internal/domain/knowledge"
	domainragtrace "github.com/AmazingCYJ/AgentRAG/internal/domain/ragtrace"
	domainusermgmt "github.com/AmazingCYJ/AgentRAG/internal/domain/usermgmt"
)

func TestPostgresRepositoryBootstrapSmoke(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("AGENTRAG_POSTGRES_DSN"))
	if dsn == "" {
		t.Skip("set AGENTRAG_POSTGRES_DSN to run PostgreSQL smoke tests")
	}

	database, err := OpenDatabase(Config{
		Driver: "postgres",
		DSN:    dsn,
	})
	if err != nil {
		t.Fatalf("open postgres database failed: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	database.SetMaxOpenConns(1)
	database.SetMaxIdleConns(1)

	schemaName := fmt.Sprintf("agentrag_test_%d", time.Now().UnixNano())
	quotedSchemaName := quotePostgresIdentifier(schemaName)
	if _, err := database.Exec(`CREATE SCHEMA ` + quotedSchemaName); err != nil {
		t.Fatalf("create postgres test schema failed: %v", err)
	}
	t.Cleanup(func() {
		_, _ = database.Exec(`DROP SCHEMA ` + quotedSchemaName + ` CASCADE`)
	})
	if _, err := database.Exec(`SET search_path TO ` + quotedSchemaName); err != nil {
		t.Fatalf("set postgres search_path failed: %v", err)
	}

	repositories := []struct {
		name      string
		bootstrap func() error
	}{
		{name: "users", bootstrap: NewSQLUserRepository(database).Bootstrap},
		{name: "sample questions", bootstrap: NewSQLSampleQuestionRepository(database).Bootstrap},
		{name: "query mappings", bootstrap: NewSQLQueryMappingRepository(database).Bootstrap},
		{name: "intent tree", bootstrap: NewSQLIntentTreeRepository(database).Bootstrap},
		{name: "conversation", bootstrap: NewSQLConversationRepository(database).Bootstrap},
		{name: "rag trace", bootstrap: NewSQLRagTraceRepository(database).Bootstrap},
		{name: "knowledge", bootstrap: NewSQLKnowledgeRepository(database).Bootstrap},
		{name: "ingestion", bootstrap: NewSQLIngestionRepository(database).Bootstrap},
	}
	for _, repository := range repositories {
		if err := repository.bootstrap(); err != nil {
			t.Fatalf("bootstrap %s repository on postgres failed: %v", repository.name, err)
		}
	}

	userRepository := NewSQLUserRepository(database)
	now := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	if err := userRepository.SaveUsers([]domainusermgmt.User{
		{
			ID:         "u_pg",
			Username:   "pg_admin",
			Password:   "admin123",
			Role:       "admin",
			CreateTime: now,
			UpdateTime: now,
		},
	}); err != nil {
		t.Fatalf("save postgres users failed: %v", err)
	}
	users, err := userRepository.LoadUsers()
	if err != nil {
		t.Fatalf("load postgres users failed: %v", err)
	}
	if len(users) != 1 || users[0].ID != "u_pg" || users[0].Username != "pg_admin" {
		t.Fatalf("unexpected postgres users %#v", users)
	}

	conversationRepository := NewSQLConversationRepository(database)
	if err := conversationRepository.SaveConversations(
		[]domainconversation.Session{
			{ConversationID: "conv_pg", UserID: "u_pg", Title: "PostgreSQL 会话", LastTime: now},
		},
		[]domainconversation.Message{
			{ID: "msg_pg", ConversationID: "conv_pg", UserID: "u_pg", Role: "assistant", Content: "PostgreSQL 消息", CreateTime: now.Add(time.Second)},
		},
		nil,
	); err != nil {
		t.Fatalf("save postgres conversations failed: %v", err)
	}
	if err := conversationRepository.SaveConversations(
		[]domainconversation.Session{
			{ConversationID: "conv_pg", UserID: "u_pg", Title: "PostgreSQL 会话更新", LastTime: now.Add(time.Minute)},
		},
		[]domainconversation.Message{
			{ID: "msg_pg", ConversationID: "conv_pg", UserID: "u_pg", Role: "assistant", Content: "PostgreSQL 消息更新", CreateTime: now.Add(time.Second)},
		},
		nil,
	); err != nil {
		t.Fatalf("update postgres conversations failed: %v", err)
	}
	sessions, messages, feedbacks, err := conversationRepository.LoadConversations()
	if err != nil {
		t.Fatalf("load postgres conversations failed: %v", err)
	}
	if len(sessions) != 1 || sessions[0].Title != "PostgreSQL 会话更新" || len(messages) != 1 || messages[0].Content != "PostgreSQL 消息更新" || len(feedbacks) != 0 {
		t.Fatalf("unexpected postgres conversations sessions=%#v messages=%#v feedbacks=%#v", sessions, messages, feedbacks)
	}

	traceRepository := NewSQLRagTraceRepository(database)
	if err := traceRepository.SaveTraceRecords(
		[]domainragtrace.Run{
			{ID: "run_pg", TraceID: "trace_pg", TraceName: "PostgreSQL Trace", EntryMethod: "chat", TaskID: "task_pg", Status: "SUCCESS", DurationMs: 80, StartTime: now, EndTime: now.Add(80 * time.Millisecond)},
		},
		[]domainragtrace.Node{
			{ID: "node_pg", TraceID: "trace_pg", NodeID: "node_pg", NodeType: "LLM", NodeName: "PostgreSQL Node", Status: "SUCCESS", DurationMs: 70, StartTime: now.Add(time.Millisecond), EndTime: now.Add(80 * time.Millisecond)},
		},
	); err != nil {
		t.Fatalf("save postgres trace records failed: %v", err)
	}
	if err := traceRepository.SaveTraceRecords(
		[]domainragtrace.Run{
			{ID: "run_pg", TraceID: "trace_pg", TraceName: "PostgreSQL Trace Updated", EntryMethod: "chat", TaskID: "task_pg", Status: "FAILED", ErrorMessage: "timeout", DurationMs: 120, StartTime: now, EndTime: now.Add(120 * time.Millisecond)},
		},
		[]domainragtrace.Node{
			{ID: "node_pg", TraceID: "trace_pg", NodeID: "node_pg", NodeType: "RETRIEVER", NodeName: "PostgreSQL Node Updated", Status: "FAILED", ErrorMessage: "timeout", DurationMs: 110, StartTime: now.Add(time.Millisecond), EndTime: now.Add(120 * time.Millisecond)},
		},
	); err != nil {
		t.Fatalf("update postgres trace records failed: %v", err)
	}
	runs, nodes, err := traceRepository.LoadTraceRecords()
	if err != nil {
		t.Fatalf("load postgres trace records failed: %v", err)
	}
	if len(runs) != 1 || runs[0].TraceName != "PostgreSQL Trace Updated" || runs[0].Status != "FAILED" || len(nodes) != 1 || nodes[0].NodeType != "RETRIEVER" || nodes[0].Status != "FAILED" {
		t.Fatalf("unexpected postgres trace records runs=%#v nodes=%#v", runs, nodes)
	}

	knowledgeRepository := NewSQLKnowledgeRepository(database)
	if err := knowledgeRepository.SaveKnowledgeRecords(
		[]domainknowledge.KnowledgeBase{
			{ID: "kb_pg", Name: "PostgreSQL 知识库", EmbeddingModel: "embedding", CollectionName: "pg_collection", CreatedBy: "admin", DocumentCount: 1, CreateTime: now, UpdateTime: now},
		},
		[]domainknowledge.KnowledgeDocument{
			{ID: "doc_pg", KBID: "kb_pg", DocName: "PostgreSQL 文档", SourceType: "file", SourceLocation: "pg.md", TextContent: "PostgreSQL 文档正文", Enabled: true, ChunkCount: 1, FileURL: "memory://pg", FileType: "md", FileSize: 100, ProcessMode: "chunk", ChunkStrategy: "fixed_size", ChunkConfig: "{}", PipelineID: "pipe_pg", Status: "pending", CreatedBy: "admin", UpdatedBy: "admin", CreateTime: now, UpdateTime: now},
		},
		[]domainknowledge.KnowledgeChunk{
			{ID: "chunk_pg", KBID: "kb_pg", DocID: "doc_pg", ChunkIndex: 0, Content: "PostgreSQL Chunk", ContentHash: "pg-hash", CharCount: 16, TokenCount: 3, Enabled: 1, CreateTime: now, UpdateTime: now},
		},
		[]domainknowledge.KnowledgeDocumentChunkLog{
			{ID: "log_pg", DocID: "doc_pg", Status: "pending", ProcessMode: "chunk", ChunkStrategy: "fixed_size", PipelineID: "pipe_pg", PipelineName: "pipe_pg", TotalDuration: 80, ChunkCount: 1, StartTime: now, EndTime: now.Add(80 * time.Millisecond), CreateTime: now},
		},
	); err != nil {
		t.Fatalf("save postgres knowledge records failed: %v", err)
	}
	if err := knowledgeRepository.SaveKnowledgeRecords(
		[]domainknowledge.KnowledgeBase{
			{ID: "kb_pg", Name: "PostgreSQL 知识库更新", EmbeddingModel: "embedding-v2", CollectionName: "pg_collection", CreatedBy: "admin", DocumentCount: 1, CreateTime: now, UpdateTime: now.Add(time.Minute)},
		},
		[]domainknowledge.KnowledgeDocument{
			{ID: "doc_pg", KBID: "kb_pg", DocName: "PostgreSQL 文档更新", SourceType: "file", SourceLocation: "pg.md", TextContent: "PostgreSQL 文档正文更新", Enabled: false, ChunkCount: 1, FileURL: "memory://pg", FileType: "md", FileSize: 120, ProcessMode: "chunk", ChunkStrategy: "structure_aware", ChunkConfig: "{}", PipelineID: "pipe_pg", Status: "success", CreatedBy: "admin", UpdatedBy: "admin", CreateTime: now, UpdateTime: now.Add(time.Minute)},
		},
		[]domainknowledge.KnowledgeChunk{
			{ID: "chunk_pg", KBID: "kb_pg", DocID: "doc_pg", ChunkIndex: 1, Content: "PostgreSQL Chunk 更新", ContentHash: "pg-hash-new", CharCount: 19, TokenCount: 4, Enabled: 0, CreateTime: now, UpdateTime: now.Add(time.Minute)},
		},
		[]domainknowledge.KnowledgeDocumentChunkLog{
			{ID: "log_pg", DocID: "doc_pg", Status: "success", ProcessMode: "chunk", ChunkStrategy: "structure_aware", PipelineID: "pipe_pg", PipelineName: "pipe_pg", TotalDuration: 120, ChunkCount: 1, StartTime: now, EndTime: now.Add(120 * time.Millisecond), CreateTime: now},
		},
	); err != nil {
		t.Fatalf("update postgres knowledge records failed: %v", err)
	}
	bases, docs, chunks, logs, err := knowledgeRepository.LoadKnowledgeRecords()
	if err != nil {
		t.Fatalf("load postgres knowledge records failed: %v", err)
	}
	if len(bases) != 1 || bases[0].Name != "PostgreSQL 知识库更新" || len(docs) != 1 || docs[0].DocName != "PostgreSQL 文档更新" || docs[0].Enabled || len(chunks) != 1 || chunks[0].Enabled != 0 || len(logs) != 1 || logs[0].Status != "success" {
		t.Fatalf("unexpected postgres knowledge records bases=%#v docs=%#v chunks=%#v logs=%#v", bases, docs, chunks, logs)
	}
}

func quotePostgresIdentifier(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}
