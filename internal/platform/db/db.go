package db

import (
	"database/sql"
	"strings"
)

// Config 定义数据库连接配置。
type Config struct {
	Driver string
	DSN    string
}

// OpenDatabase 根据配置打开数据库连接；未配置时返回 nil。
func OpenDatabase(cfg Config) (*sql.DB, error) {
	driver := strings.TrimSpace(cfg.Driver)
	dsn := strings.TrimSpace(cfg.DSN)
	if driver == "" || dsn == "" {
		return nil, nil
	}
	database, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, err
	}
	if err := database.Ping(); err != nil {
		_ = database.Close()
		return nil, err
	}
	return database, nil
}
