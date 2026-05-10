package db

import (
	"database/sql"
	"strings"
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

func TestSQLUserRepositoryReconcilesSavedUsers(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite database failed: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	repository := NewSQLUserRepository(database)
	if err := repository.Bootstrap(); err != nil {
		t.Fatalf("bootstrap user table failed: %v", err)
	}
	now := time.Date(2026, 5, 10, 10, 0, 0, 0, time.UTC)
	if err := repository.SaveUsers([]domainusermgmt.User{
		{ID: "u_keep", Username: "admin", Password: "old", Role: "admin", CreateTime: now, UpdateTime: now},
		{ID: "u_remove", Username: "remove", Password: "old", Role: "user", CreateTime: now.Add(time.Minute), UpdateTime: now.Add(time.Minute)},
	}); err != nil {
		t.Fatalf("save initial users failed: %v", err)
	}
	if err := repository.SaveUsers([]domainusermgmt.User{
		{ID: "u_keep", Username: "admin", Password: "new", Role: "owner", Avatar: "avatar.png", CreateTime: now, UpdateTime: now.Add(2 * time.Minute)},
		{ID: "u_new", Username: "alice", Password: "alice123", Role: "user", CreateTime: now.Add(3 * time.Minute), UpdateTime: now.Add(3 * time.Minute)},
	}); err != nil {
		t.Fatalf("save reconciled users failed: %v", err)
	}

	loaded, err := repository.LoadUsers()
	if err != nil {
		t.Fatalf("load reconciled users failed: %v", err)
	}
	if len(loaded) != 2 {
		t.Fatalf("expected 2 users after reconcile, got %d", len(loaded))
	}
	usersByID := map[string]domainusermgmt.User{}
	for _, user := range loaded {
		usersByID[user.ID] = user
	}
	if _, ok := usersByID["u_remove"]; ok {
		t.Fatal("expected missing user to be deleted")
	}
	if usersByID["u_keep"].Password != "new" || usersByID["u_keep"].Role != "owner" || usersByID["u_keep"].Avatar != "avatar.png" {
		t.Fatalf("expected existing user to be updated, got %#v", usersByID["u_keep"])
	}
	if usersByID["u_new"].Username != "alice" {
		t.Fatalf("expected new user to be inserted, got %#v", usersByID["u_new"])
	}

	if err := repository.SaveUsers(nil); err != nil {
		t.Fatalf("save empty users failed: %v", err)
	}
	loaded, err = repository.LoadUsers()
	if err != nil {
		t.Fatalf("load empty users failed: %v", err)
	}
	if len(loaded) != 0 {
		t.Fatalf("expected empty user snapshot to clear table, got %#v", loaded)
	}
}

func TestSQLUserRepositoryAllowsUsernameSwap(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite database failed: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	repository := NewSQLUserRepository(database)
	if err := repository.Bootstrap(); err != nil {
		t.Fatalf("bootstrap user table failed: %v", err)
	}
	now := time.Date(2026, 5, 10, 14, 0, 0, 0, time.UTC)
	if err := repository.SaveUsers([]domainusermgmt.User{
		{ID: "u_1", Username: "alice", Password: "p1", Role: "admin", CreateTime: now, UpdateTime: now},
		{ID: "u_2", Username: "bob", Password: "p2", Role: "user", CreateTime: now.Add(time.Minute), UpdateTime: now.Add(time.Minute)},
	}); err != nil {
		t.Fatalf("save initial users failed: %v", err)
	}
	if err := repository.SaveUsers([]domainusermgmt.User{
		{ID: "u_1", Username: "bob", Password: "p1", Role: "admin", CreateTime: now, UpdateTime: now.Add(2 * time.Minute)},
		{ID: "u_2", Username: "alice", Password: "p2", Role: "user", CreateTime: now.Add(time.Minute), UpdateTime: now.Add(2 * time.Minute)},
	}); err != nil {
		t.Fatalf("save username swap failed: %v", err)
	}

	loaded, err := repository.LoadUsers()
	if err != nil {
		t.Fatalf("load swapped users failed: %v", err)
	}
	usersByID := map[string]domainusermgmt.User{}
	for _, user := range loaded {
		usersByID[user.ID] = user
	}
	if usersByID["u_1"].Username != "bob" || usersByID["u_2"].Username != "alice" {
		t.Fatalf("expected usernames to be swapped, got %#v", usersByID)
	}
}

func TestSQLUserRepositoryRejectsDuplicateIDs(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite database failed: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	repository := NewSQLUserRepository(database)
	if err := repository.Bootstrap(); err != nil {
		t.Fatalf("bootstrap user table failed: %v", err)
	}
	now := time.Date(2026, 5, 10, 13, 0, 0, 0, time.UTC)
	if err := repository.SaveUsers([]domainusermgmt.User{
		{ID: "u_keep", Username: "admin", Password: "old", Role: "admin", CreateTime: now, UpdateTime: now},
	}); err != nil {
		t.Fatalf("save initial user failed: %v", err)
	}

	err = repository.SaveUsers([]domainusermgmt.User{
		{ID: "u_keep", Username: "admin", Password: "new", Role: "admin", CreateTime: now, UpdateTime: now},
		{ID: "u_keep", Username: "duplicate", Password: "new", Role: "user", CreateTime: now, UpdateTime: now},
	})
	if err == nil || !strings.Contains(err.Error(), `duplicate id "u_keep"`) {
		t.Fatalf("expected duplicate id error, got %v", err)
	}

	loaded, err := repository.LoadUsers()
	if err != nil {
		t.Fatalf("load users after duplicate id failure failed: %v", err)
	}
	if len(loaded) != 1 || loaded[0].Password != "old" || loaded[0].Username != "admin" {
		t.Fatalf("expected existing user to remain unchanged, got %#v", loaded)
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
