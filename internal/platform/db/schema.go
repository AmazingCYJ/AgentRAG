package db

func columnExists(database *SQLDB, tableName, columnName string) bool {
	if database == nil {
		return false
	}
	if database.isPostgres() {
		return postgresColumnExists(database, tableName, columnName)
	}
	return sqliteColumnExists(database, tableName, columnName)
}

func sqliteColumnExists(database *SQLDB, tableName, columnName string) bool {
	rows, err := database.Query(`PRAGMA table_info(` + tableName + `)`)
	if err != nil {
		return false
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name, columnType string
		var notNull int
		var defaultValue any
		var pk int
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &pk); err != nil {
			return false
		}
		if name == columnName {
			return true
		}
	}
	return false
}

func postgresColumnExists(database *SQLDB, tableName, columnName string) bool {
	row := database.QueryRow(`
SELECT 1
FROM information_schema.columns
WHERE table_schema = current_schema()
  AND table_name = ?
  AND column_name = ?
LIMIT 1`, tableName, columnName)
	var exists int
	return row.Scan(&exists) == nil
}
