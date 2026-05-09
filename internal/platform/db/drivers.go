package db

import (
	// 注册 SQLite 驱动，保证生产入口配置 database.driver=sqlite 时可以直接启用 SQL 持久化。
	_ "modernc.org/sqlite"
)
