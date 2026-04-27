package db

import (
	"database/sql"
	"testing"
	"time"

	domainusermgmt "github.com/AmazingCYJ/AgentRAG/internal/domain/usermgmt"
	_ "modernc.org/sqlite"
)

func TestSQLUserRepositorySavesAndLoadsUsers(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite database failed: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	repository := NewSQLUserRepository(database)
	if err := repository.Bootstrap(); err != nil {
		t.Fatalf("bootstrap user table failed: %v", err)
	}
	now := time.Date(2026, 4, 27, 10, 0, 0, 0, time.UTC)
	if err := repository.SaveUsers([]domainusermgmt.User{
		{
			ID:         "u_1",
			Username:   "admin",
			Password:   "admin123",
			Role:       "admin",
			Avatar:     "https://example.com/avatar.png",
			CreateTime: now,
			UpdateTime: now.Add(time.Minute),
		},
	}); err != nil {
		t.Fatalf("save users failed: %v", err)
	}

	loaded, err := repository.LoadUsers()
	if err != nil {
		t.Fatalf("load users failed: %v", err)
	}
	if len(loaded) != 1 {
		t.Fatalf("expected 1 user, got %d", len(loaded))
	}
	if loaded[0].ID != "u_1" || loaded[0].Username != "admin" || loaded[0].Password != "admin123" {
		t.Fatalf("unexpected loaded user %#v", loaded[0])
	}
	if loaded[0].Role != "admin" || loaded[0].Avatar != "https://example.com/avatar.png" {
		t.Fatalf("unexpected loaded profile %#v", loaded[0])
	}
}

func TestOpenDatabaseReturnsNilWhenDisabled(t *testing.T) {
	database, err := OpenDatabase(Config{})
	if err != nil {
		t.Fatalf("open disabled database failed: %v", err)
	}
	if database != nil {
		t.Fatal("expected nil database when config is disabled")
	}
}
