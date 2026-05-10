package db

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

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
}

func quotePostgresIdentifier(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}
