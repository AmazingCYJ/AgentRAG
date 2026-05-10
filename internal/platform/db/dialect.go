package db

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"strconv"
	"strings"
	"sync"
)

type sqlDialect string

const (
	sqliteDialect   sqlDialect = "sqlite"
	postgresDialect sqlDialect = "postgres"
)

var databaseDialects sync.Map

// SQLDB 在 database/sql 之上补充少量方言适配能力。
type SQLDB struct {
	raw     *sql.DB
	dialect sqlDialect
}

type SQLTx struct {
	raw     *sql.Tx
	dialect sqlDialect
}

func newSQLDB(database *sql.DB) *SQLDB {
	return newSQLDBWithDriver(database, "")
}

func newSQLDBWithDriver(database *sql.DB, driverName string) *SQLDB {
	if database == nil {
		return nil
	}
	dialect := dialectFromDriverName(driverName)
	if dialect == "" {
		if value, ok := databaseDialects.Load(database); ok {
			if registered, ok := value.(sqlDialect); ok {
				dialect = registered
			}
		}
	}
	if dialect == "" {
		dialect = dialectFromDriver(database.Driver())
	}
	return &SQLDB{
		raw:     database,
		dialect: dialect,
	}
}

func registerDatabaseDialect(database *sql.DB, driverName string) {
	if database == nil {
		return
	}
	if dialect := dialectFromDriverName(driverName); dialect != "" {
		databaseDialects.Store(database, dialect)
	}
}

func (db *SQLDB) Exec(query string, args ...any) (sql.Result, error) {
	return db.raw.Exec(db.bind(query), args...)
}

func (db *SQLDB) Query(query string, args ...any) (*sql.Rows, error) {
	return db.raw.Query(db.bind(query), args...)
}

func (db *SQLDB) QueryRow(query string, args ...any) *sql.Row {
	return db.raw.QueryRow(db.bind(query), args...)
}

func (db *SQLDB) Prepare(query string) (*sql.Stmt, error) {
	return db.raw.Prepare(db.bind(query))
}

func (db *SQLDB) Begin() (*SQLTx, error) {
	tx, err := db.raw.Begin()
	if err != nil {
		return nil, err
	}
	return &SQLTx{
		raw:     tx,
		dialect: db.dialect,
	}, nil
}

func (db *SQLDB) bind(query string) string {
	return bindPlaceholders(db.dialect, query)
}

func (db *SQLDB) isPostgres() bool {
	return db != nil && db.dialect == postgresDialect
}

func (tx *SQLTx) Exec(query string, args ...any) (sql.Result, error) {
	return tx.raw.Exec(bindPlaceholders(tx.dialect, query), args...)
}

func (tx *SQLTx) Query(query string, args ...any) (*sql.Rows, error) {
	return tx.raw.Query(bindPlaceholders(tx.dialect, query), args...)
}

func (tx *SQLTx) Prepare(query string) (*sql.Stmt, error) {
	return tx.raw.Prepare(bindPlaceholders(tx.dialect, query))
}

func (tx *SQLTx) Commit() error {
	return tx.raw.Commit()
}

func (tx *SQLTx) Rollback() error {
	return tx.raw.Rollback()
}

func normalizeDriverName(driver string) string {
	return strings.ToLower(strings.TrimSpace(driver))
}

func sqlDriverName(driver string) string {
	switch normalizeDriverName(driver) {
	case "postgres", "postgresql", "pg":
		return "pgx"
	default:
		return strings.TrimSpace(driver)
	}
}

func dialectFromDriverName(driver string) sqlDialect {
	switch normalizeDriverName(driver) {
	case "pgx", "pgx/v5", "postgres", "postgresql", "pg":
		return postgresDialect
	case "sqlite", "sqlite3":
		return sqliteDialect
	default:
		return ""
	}
}

func dialectFromDriver(driver driver.Driver) sqlDialect {
	typeName := strings.ToLower(fmt.Sprintf("%T", driver))
	if strings.Contains(typeName, "pgx") || strings.Contains(typeName, "postgres") {
		return postgresDialect
	}
	return sqliteDialect
}

func bindPlaceholders(dialect sqlDialect, query string) string {
	if dialect != postgresDialect || !strings.Contains(query, "?") {
		return query
	}
	return rebindQuestionPlaceholders(query)
}

func addColumnSQL(dialect sqlDialect, tableName, columnDefinition string) string {
	if dialect == postgresDialect {
		return "ALTER TABLE " + tableName + " ADD COLUMN IF NOT EXISTS " + columnDefinition
	}
	return "ALTER TABLE " + tableName + " ADD COLUMN " + columnDefinition
}

func rebindQuestionPlaceholders(query string) string {
	var builder strings.Builder
	builder.Grow(len(query) + 8)

	index := 1
	inSingleQuote := false
	inDoubleQuote := false
	inLineComment := false
	inBlockComment := false

	for i := 0; i < len(query); i++ {
		current := query[i]
		var next byte
		if i+1 < len(query) {
			next = query[i+1]
		}

		switch {
		case inLineComment:
			builder.WriteByte(current)
			if current == '\n' {
				inLineComment = false
			}
			continue
		case inBlockComment:
			builder.WriteByte(current)
			if current == '*' && next == '/' {
				builder.WriteByte(next)
				i++
				inBlockComment = false
			}
			continue
		case inSingleQuote:
			builder.WriteByte(current)
			if current == '\'' {
				if next == '\'' {
					builder.WriteByte(next)
					i++
				} else {
					inSingleQuote = false
				}
			}
			continue
		case inDoubleQuote:
			builder.WriteByte(current)
			if current == '"' {
				inDoubleQuote = false
			}
			continue
		}

		if current == '-' && next == '-' {
			builder.WriteByte(current)
			builder.WriteByte(next)
			i++
			inLineComment = true
			continue
		}
		if current == '/' && next == '*' {
			builder.WriteByte(current)
			builder.WriteByte(next)
			i++
			inBlockComment = true
			continue
		}
		if current == '\'' {
			builder.WriteByte(current)
			inSingleQuote = true
			continue
		}
		if current == '"' {
			builder.WriteByte(current)
			inDoubleQuote = true
			continue
		}
		if current == '?' {
			builder.WriteByte('$')
			builder.WriteString(strconv.Itoa(index))
			index++
			continue
		}
		builder.WriteByte(current)
	}
	return builder.String()
}
