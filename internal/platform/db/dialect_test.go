package db

import (
	"database/sql"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

func TestSQLDriverNameNormalizesPostgresAliases(t *testing.T) {
	for _, driverName := range []string{"postgres", "postgresql", "pg"} {
		if got := sqlDriverName(driverName); got != "pgx" {
			t.Fatalf("expected %s to normalize to pgx, got %s", driverName, got)
		}
	}
	if got := sqlDriverName("sqlite"); got != "sqlite" {
		t.Fatalf("expected sqlite driver to stay unchanged, got %s", got)
	}
}

func TestBindPlaceholdersForPostgres(t *testing.T) {
	query := `
INSERT INTO demo (a, b, note)
VALUES (?, ?, '?')
-- comment ?
/* block ? */
RETURNING "weird?column"`
	got := bindPlaceholders(postgresDialect, query)
	want := `
INSERT INTO demo (a, b, note)
VALUES ($1, $2, '?')
-- comment ?
/* block ? */
RETURNING "weird?column"`
	if got != want {
		t.Fatalf("unexpected postgres placeholders:\n%s", got)
	}
}

func TestBindPlaceholdersLeavesSQLiteQueriesUntouched(t *testing.T) {
	query := "SELECT * FROM demo WHERE a = ? AND b = '?'"
	if got := bindPlaceholders(sqliteDialect, query); got != query {
		t.Fatalf("expected sqlite query unchanged, got %s", got)
	}
}

func TestDialectFromDriverNameRecognizesPostgresDrivers(t *testing.T) {
	for _, driverName := range []string{"pgx", "pgx/v5", "postgres", "postgresql", "pg"} {
		if got := dialectFromDriverName(driverName); got != postgresDialect {
			t.Fatalf("expected %s to use postgres dialect, got %s", driverName, got)
		}
	}
}

func TestRegisteredDatabaseDialectOverridesDriverInference(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite database failed: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	registerDatabaseDialect(database, "postgres")
	wrapped := newSQLDB(database)
	if !wrapped.isPostgres() {
		t.Fatal("expected registered postgres dialect to override driver inference")
	}
	if got := wrapped.bind("SELECT * FROM demo WHERE id = ?"); got != "SELECT * FROM demo WHERE id = $1" {
		t.Fatalf("unexpected registered postgres binding %s", got)
	}
}

func TestLegacyFeedbackInsertSQLUsesDialectSyntax(t *testing.T) {
	postgresSQL := legacyFeedbackInsertSQL(postgresDialect)
	if !strings.Contains(postgresSQL, "ON CONFLICT (message_id, user_id) DO NOTHING") {
		t.Fatalf("expected postgres conflict clause, got %s", postgresSQL)
	}
	if strings.Contains(postgresSQL, "INSERT OR IGNORE") {
		t.Fatalf("postgres legacy feedback sql should not use sqlite syntax: %s", postgresSQL)
	}

	sqliteSQL := legacyFeedbackInsertSQL(sqliteDialect)
	if !strings.Contains(sqliteSQL, "INSERT OR IGNORE") {
		t.Fatalf("expected sqlite INSERT OR IGNORE, got %s", sqliteSQL)
	}
}

func TestAddColumnSQLUsesPostgresIfNotExists(t *testing.T) {
	postgresSQL := addColumnSQL(postgresDialect, "demo", "name TEXT")
	if postgresSQL != "ALTER TABLE demo ADD COLUMN IF NOT EXISTS name TEXT" {
		t.Fatalf("unexpected postgres add column SQL %s", postgresSQL)
	}
	sqliteSQL := addColumnSQL(sqliteDialect, "demo", "name TEXT")
	if sqliteSQL != "ALTER TABLE demo ADD COLUMN name TEXT" {
		t.Fatalf("unexpected sqlite add column SQL %s", sqliteSQL)
	}
}
