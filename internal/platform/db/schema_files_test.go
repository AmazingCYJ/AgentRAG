package db

import (
	"os"
	"strings"
	"testing"
)

func TestPostgresSchemaCoversRepositoryTables(t *testing.T) {
	content, err := os.ReadFile("../../../resources/database/schema_pg.sql")
	if err != nil {
		t.Fatalf("read postgres schema failed: %v", err)
	}
	schema := string(content)
	for _, tableName := range []string{
		"agentrag_users",
		"agentrag_sample_questions",
		"agentrag_query_mappings",
		"agentrag_intent_nodes",
		"agentrag_conversations",
		"agentrag_conversation_summaries",
		"agentrag_conversation_messages",
		"agentrag_message_feedback",
		"agentrag_trace_runs",
		"agentrag_trace_nodes",
		"agentrag_knowledge_bases",
		"agentrag_knowledge_documents",
		"agentrag_knowledge_chunks",
		"agentrag_knowledge_chunk_logs",
		"agentrag_ingestion_pipelines",
		"agentrag_ingestion_pipeline_nodes",
		"agentrag_ingestion_tasks",
		"agentrag_ingestion_task_nodes",
	} {
		if !strings.Contains(schema, "CREATE TABLE IF NOT EXISTS "+tableName) {
			t.Fatalf("postgres schema missing table %s", tableName)
		}
	}
	if strings.Contains(schema, "CREATE TABLE t_") {
		t.Fatal("postgres schema should use Go repository table names, not legacy t_* names")
	}
	if !strings.Contains(schema, "ck_agentrag_intent_nodes_top_k_positive") {
		t.Fatal("postgres schema should enforce positive intent top_k values")
	}
}

func TestPostgresInitDataSeedsBootstrapUser(t *testing.T) {
	content, err := os.ReadFile("../../../resources/database/init_data_pg.sql")
	if err != nil {
		t.Fatalf("read postgres init data failed: %v", err)
	}
	initData := string(content)
	for _, expected := range []string{
		"INSERT INTO agentrag_users",
		"'u_admin'",
		"'admin'",
		"'admin123'",
		"ON CONFLICT (id) DO UPDATE",
	} {
		if !strings.Contains(initData, expected) {
			t.Fatalf("postgres init data missing %q", expected)
		}
	}
}
