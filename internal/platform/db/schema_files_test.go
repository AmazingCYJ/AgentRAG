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

func TestPostgresComposeRunsSchemaAndInitData(t *testing.T) {
	content, err := os.ReadFile("../../../resources/docker/postgres.compose.yaml")
	if err != nil {
		t.Fatalf("read postgres compose failed: %v", err)
	}
	compose := string(content)
	for _, expected := range []string{
		"name: agentrag",
		"postgres:16-alpine",
		"POSTGRES_DB: ragent",
		"../database/schema_pg.sql:/docker-entrypoint-initdb.d/01_schema_pg.sql:ro",
		"../database/init_data_pg.sql:/docker-entrypoint-initdb.d/02_init_data_pg.sql:ro",
		"pg_isready -U postgres -d ragent",
	} {
		if !strings.Contains(compose, expected) {
			t.Fatalf("postgres compose missing %q", expected)
		}
	}
}

func TestPostgresUpgradeScriptCoversRuntimeMigrations(t *testing.T) {
	content, err := os.ReadFile("../../../resources/database/upgrade_v1.0_to_v1.1.sql")
	if err != nil {
		t.Fatalf("read postgres upgrade script failed: %v", err)
	}
	upgrade := string(content)
	for _, expected := range []string{
		"ALTER TABLE agentrag_conversations ADD COLUMN IF NOT EXISTS id TEXT NOT NULL DEFAULT ''",
		"WHERE id IS NULL OR id = ''",
		"ALTER TABLE agentrag_conversation_messages ADD COLUMN IF NOT EXISTS thinking_content TEXT NOT NULL DEFAULT ''",
		"CREATE TABLE IF NOT EXISTS agentrag_message_feedback",
		"CREATE UNIQUE INDEX IF NOT EXISTS uq_agentrag_message_feedback_message_user",
		"column_name = 'vote'",
		"ON CONFLICT (message_id, user_id) DO NOTHING",
		"ALTER TABLE agentrag_trace_runs ADD COLUMN IF NOT EXISTS id TEXT NOT NULL DEFAULT ''",
		"ALTER TABLE agentrag_trace_nodes ADD COLUMN IF NOT EXISTS id TEXT NOT NULL DEFAULT ''",
		"ALTER TABLE agentrag_knowledge_documents ADD COLUMN IF NOT EXISTS text_content TEXT NOT NULL DEFAULT ''",
		"ALTER TABLE agentrag_knowledge_chunks ADD COLUMN IF NOT EXISTS embedding_model TEXT NOT NULL DEFAULT ''",
		"ALTER TABLE agentrag_knowledge_chunks ADD COLUMN IF NOT EXISTS embedding_vector TEXT NOT NULL DEFAULT ''",
		"RENAME COLUMN embedding_duration TO embed_duration",
		"ALTER TABLE agentrag_knowledge_chunk_logs ADD COLUMN IF NOT EXISTS persist_duration INTEGER NOT NULL DEFAULT 0",
		"ck_agentrag_intent_nodes_top_k_positive",
		"NOT VALID",
	} {
		if !strings.Contains(upgrade, expected) {
			t.Fatalf("postgres upgrade script missing %q", expected)
		}
	}
	if strings.Contains(upgrade, " t_") {
		t.Fatal("postgres upgrade script should use Go repository table names, not legacy t_* names")
	}
}
