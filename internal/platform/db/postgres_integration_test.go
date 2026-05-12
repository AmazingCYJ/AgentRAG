package db

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	domainconversation "github.com/AmazingCYJ/AgentRAG/internal/domain/conversation"
	domainingestion "github.com/AmazingCYJ/AgentRAG/internal/domain/ingestion"
	domainintenttree "github.com/AmazingCYJ/AgentRAG/internal/domain/intenttree"
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
		{name: "knowledge vector", bootstrap: NewSQLKnowledgeVectorStore(database).Bootstrap},
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

	intentTreeRepository := NewSQLIntentTreeRepository(database)
	intentTopK := 6
	if err := intentTreeRepository.SaveNodes([]domainintenttree.Node{
		{ID: "intent_pg", KBID: "kb_pg", IntentCode: "policy", Name: "PostgreSQL 意图", Level: 0, Description: "制度查询", Examples: []string{"报销规则"}, CollectionName: "pg_collection", TopK: &intentTopK, Kind: 1, SortOrder: 1, Enabled: 1, PromptSnippet: "引用原文", CreateTime: now, UpdateTime: now},
		{ID: "intent_remove_pg", IntentCode: "remove", Name: "待删除意图", Level: 0, Kind: 0, SortOrder: 2, Enabled: 1, CreateTime: now.Add(time.Second), UpdateTime: now.Add(time.Second)},
	}); err != nil {
		t.Fatalf("save postgres intent nodes failed: %v", err)
	}
	if err := intentTreeRepository.SaveNodes([]domainintenttree.Node{
		{ID: "intent_pg", IntentCode: "policy_updated", Name: "PostgreSQL 意图更新", Level: 1, ParentCode: "root", Kind: 2, SortOrder: 3, Enabled: 0, PromptTemplate: "更新模板", CreateTime: now, UpdateTime: now.Add(time.Minute)},
		{ID: "intent_new_pg", IntentCode: "root", Name: "PostgreSQL 新意图", Level: 0, Kind: 0, SortOrder: 1, Enabled: 1, CreateTime: now.Add(2 * time.Second), UpdateTime: now.Add(2 * time.Second)},
	}); err != nil {
		t.Fatalf("update postgres intent nodes failed: %v", err)
	}
	intentNodes, err := intentTreeRepository.LoadNodes()
	if err != nil {
		t.Fatalf("load postgres intent nodes failed: %v", err)
	}
	intentNodesByID := map[string]domainintenttree.Node{}
	for _, node := range intentNodes {
		intentNodesByID[node.ID] = node
	}
	if len(intentNodes) != 2 || intentNodesByID["intent_pg"].IntentCode != "policy_updated" || intentNodesByID["intent_pg"].Enabled != 0 || intentNodesByID["intent_pg"].TopK != nil || intentNodesByID["intent_new_pg"].Name != "PostgreSQL 新意图" {
		t.Fatalf("unexpected postgres intent nodes %#v", intentNodes)
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
			{ID: "chunk_pg", KBID: "kb_pg", DocID: "doc_pg", ChunkIndex: 0, Content: "PostgreSQL Chunk", ContentHash: "pg-hash", CharCount: 16, TokenCount: 3, EmbeddingModel: "embedding", Embedding: []float64{0.1, 0.2}, Enabled: 1, CreateTime: now, UpdateTime: now},
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
			{ID: "chunk_pg", KBID: "kb_pg", DocID: "doc_pg", ChunkIndex: 1, Content: "PostgreSQL Chunk 更新", ContentHash: "pg-hash-new", CharCount: 19, TokenCount: 4, EmbeddingModel: "embedding-v2", Embedding: []float64{0.3, 0.4, 0.5}, Enabled: 0, CreateTime: now, UpdateTime: now.Add(time.Minute)},
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
	if len(bases) != 1 || bases[0].Name != "PostgreSQL 知识库更新" || len(docs) != 1 || docs[0].DocName != "PostgreSQL 文档更新" || docs[0].Enabled || len(chunks) != 1 || chunks[0].Enabled != 0 || chunks[0].EmbeddingModel != "embedding-v2" || len(chunks[0].Embedding) != 3 || chunks[0].Embedding[2] != 0.5 || len(logs) != 1 || logs[0].Status != "success" {
		t.Fatalf("unexpected postgres knowledge records bases=%#v docs=%#v chunks=%#v logs=%#v", bases, docs, chunks, logs)
	}
	vectorStore := NewSQLKnowledgeVectorStore(database)
	if err := vectorStore.Upsert(context.Background(), []domainknowledge.KnowledgeVector{
		{ID: "vec_pg", KBID: "kb_pg", DocID: "doc_pg", CollectionName: "pg_collection", ChunkID: "chunk_pg", ChunkIndex: 1, Content: "PostgreSQL Chunk 更新", EmbeddingModel: "embedding-v2", Embedding: []float64{0.3, 0.4, 0.5}},
	}); err != nil {
		t.Fatalf("upsert postgres knowledge vector failed: %v", err)
	}
	vectorResults, err := vectorStore.Search(context.Background(), domainknowledge.VectorSearchRequest{
		CollectionName: "pg_collection",
		EmbeddingModel: "embedding-v2",
		Embedding:      []float64{0.3, 0.4, 0.5},
		Limit:          1,
	})
	if err != nil {
		t.Fatalf("search postgres knowledge vector failed: %v", err)
	}
	if len(vectorResults) != 1 || vectorResults[0].Vector.ID != "vec_pg" {
		t.Fatalf("unexpected postgres knowledge vector results %#v", vectorResults)
	}

	ingestionRepository := NewSQLIngestionRepository(database)
	if err := ingestionRepository.SaveIngestionRecords(
		[]domainingestion.Pipeline{
			{
				ID: "pipe_pg", Name: "PostgreSQL Pipeline", Description: "postgres smoke", CreatedBy: "admin", CreateTime: now, UpdateTime: now,
				Nodes: []domainingestion.PipelineNode{
					{ID: 1, NodeID: "fetcher", NodeType: "fetcher", Settings: map[string]any{"timeoutMs": float64(1000)}, NextNodeID: "parser"},
					{ID: 2, NodeID: "parser", NodeType: "parser"},
				},
			},
		},
		[]domainingestion.Task{
			{ID: "task_pg", PipelineID: "pipe_pg", SourceType: "url", SourceLocation: "https://example.com/pg", SourceFileName: "pg.md", Status: "running", ChunkCount: 1, Logs: []domainingestion.TaskLog{{NodeID: "fetcher", NodeType: "fetcher", Message: "fetched", DurationMs: 10, Success: true}}, Metadata: map[string]any{"version": float64(1)}, StartedAt: now, CompletedAt: now.Add(10 * time.Millisecond), CreatedBy: "admin", CreateTime: now, UpdateTime: now},
		},
		[]domainingestion.TaskNode{
			{ID: "tasknode_pg", TaskID: "task_pg", PipelineID: "pipe_pg", NodeID: "fetcher", NodeType: "fetcher", NodeOrder: 1, Status: "running", DurationMs: 10, Message: "fetching", Output: map[string]any{"nextNodeId": "parser"}, CreateTime: now, UpdateTime: now},
		},
	); err != nil {
		t.Fatalf("save postgres ingestion records failed: %v", err)
	}
	if err := ingestionRepository.SaveIngestionRecords(
		[]domainingestion.Pipeline{
			{
				ID: "pipe_pg", Name: "PostgreSQL Pipeline Updated", Description: "postgres smoke updated", CreatedBy: "admin", CreateTime: now, UpdateTime: now.Add(time.Minute),
				Nodes: []domainingestion.PipelineNode{
					{ID: 2, NodeID: "parser", NodeType: "parser-v2", Settings: map[string]any{"format": "markdown"}},
					{ID: 3, NodeID: "embedder", NodeType: "embedder"},
				},
			},
		},
		[]domainingestion.Task{
			{ID: "task_pg", PipelineID: "pipe_pg", SourceType: "url", SourceLocation: "https://example.com/pg-updated", SourceFileName: "pg.md", Status: "failed", ChunkCount: 2, ErrorMessage: "timeout", Logs: []domainingestion.TaskLog{{NodeID: "parser", NodeType: "parser-v2", Message: "parsed", DurationMs: 20, Success: false, Error: "timeout"}}, Metadata: map[string]any{"version": float64(2)}, StartedAt: now, CompletedAt: now.Add(20 * time.Millisecond), CreatedBy: "admin", CreateTime: now, UpdateTime: now.Add(time.Minute)},
		},
		[]domainingestion.TaskNode{
			{ID: "tasknode_pg", TaskID: "task_pg", PipelineID: "pipe_pg", NodeID: "parser", NodeType: "parser-v2", NodeOrder: 2, Status: "failed", DurationMs: 20, Message: "parsed", ErrorMessage: "timeout", Output: map[string]any{"nextNodeId": "embedder"}, CreateTime: now, UpdateTime: now.Add(time.Minute)},
		},
	); err != nil {
		t.Fatalf("update postgres ingestion records failed: %v", err)
	}
	pipelines, ingestionTasks, ingestionTaskNodes, err := ingestionRepository.LoadIngestionRecords()
	if err != nil {
		t.Fatalf("load postgres ingestion records failed: %v", err)
	}
	if len(pipelines) != 1 || pipelines[0].Name != "PostgreSQL Pipeline Updated" || len(pipelines[0].Nodes) != 2 || pipelines[0].Nodes[0].ID != 2 || pipelines[0].Nodes[0].NodeType != "parser-v2" || len(ingestionTasks) != 1 || ingestionTasks[0].Status != "failed" || ingestionTasks[0].Metadata["version"] != float64(2) || len(ingestionTaskNodes) != 1 || ingestionTaskNodes[0].Status != "failed" || ingestionTaskNodes[0].Output["nextNodeId"] != "embedder" {
		t.Fatalf("unexpected postgres ingestion records pipelines=%#v tasks=%#v taskNodes=%#v", pipelines, ingestionTasks, ingestionTaskNodes)
	}
}

func quotePostgresIdentifier(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}
