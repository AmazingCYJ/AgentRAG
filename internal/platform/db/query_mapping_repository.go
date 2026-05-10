package db

import (
	"database/sql"

	domainquerymapping "github.com/AmazingCYJ/AgentRAG/internal/domain/querymapping"
)

// SQLQueryMappingRepository 使用关系型数据库持久化关键词映射。
type SQLQueryMappingRepository struct {
	database *SQLDB
}

// NewSQLQueryMappingRepository 创建关键词映射 SQL 仓储。
func NewSQLQueryMappingRepository(database *sql.DB) *SQLQueryMappingRepository {
	return &SQLQueryMappingRepository{database: newSQLDB(database)}
}

// Bootstrap 初始化关键词映射表结构。
func (r *SQLQueryMappingRepository) Bootstrap() error {
	_, err := r.database.Exec(`
CREATE TABLE IF NOT EXISTS agentrag_query_mappings (
    id TEXT PRIMARY KEY,
    source_term TEXT NOT NULL,
    target_term TEXT NOT NULL,
    match_type INTEGER NOT NULL,
    priority INTEGER NOT NULL,
    enabled BOOLEAN NOT NULL,
    remark TEXT NOT NULL DEFAULT '',
    create_time TIMESTAMP NOT NULL,
    update_time TIMESTAMP NOT NULL
)`)
	return err
}

// LoadQueryMappings 从数据库加载全部关键词映射。
func (r *SQLQueryMappingRepository) LoadQueryMappings() ([]domainquerymapping.Item, error) {
	rows, err := r.database.Query(`
SELECT id, source_term, target_term, match_type, priority, enabled, remark, create_time, update_time
FROM agentrag_query_mappings
ORDER BY priority ASC, create_time ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []domainquerymapping.Item{}
	for rows.Next() {
		var item domainquerymapping.Item
		if err := rows.Scan(&item.ID, &item.SourceTerm, &item.TargetTerm, &item.MatchType, &item.Priority, &item.Enabled, &item.Remark, &item.CreateTime, &item.UpdateTime); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// SaveQueryMappings 覆盖保存当前关键词映射集合。
func (r *SQLQueryMappingRepository) SaveQueryMappings(items []domainquerymapping.Item) error {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	if err := rejectDuplicateIDs(ids); err != nil {
		return err
	}

	tx, err := r.database.Begin()
	if err != nil {
		return err
	}
	if err := deleteMissingRows(tx, "agentrag_query_mappings", "id", ids); err != nil {
		_ = tx.Rollback()
		return err
	}
	stmt, err := tx.Prepare(`
INSERT INTO agentrag_query_mappings (id, source_term, target_term, match_type, priority, enabled, remark, create_time, update_time)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (id) DO UPDATE SET
    source_term = excluded.source_term,
    target_term = excluded.target_term,
    match_type = excluded.match_type,
    priority = excluded.priority,
    enabled = excluded.enabled,
    remark = excluded.remark,
    create_time = excluded.create_time,
    update_time = excluded.update_time`)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	defer stmt.Close()

	for _, item := range items {
		if _, err := stmt.Exec(item.ID, item.SourceTerm, item.TargetTerm, item.MatchType, item.Priority, item.Enabled, item.Remark, normalizeTime(item.CreateTime), normalizeTime(item.UpdateTime)); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}
