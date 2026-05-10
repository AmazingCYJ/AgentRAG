package db

import (
	// 注册 PostgreSQL 驱动，配置 database.driver=postgres/postgresql/pgx 时使用。
	_ "github.com/jackc/pgx/v5/stdlib"
	// 注册 SQLite 驱动，保证生产入口配置 database.driver=sqlite 时可以直接启用 SQL 持久化。
	_ "modernc.org/sqlite"
)
