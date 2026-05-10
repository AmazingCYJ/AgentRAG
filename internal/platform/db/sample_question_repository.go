package db

import (
	"database/sql"

	domainsamplequestion "github.com/AmazingCYJ/AgentRAG/internal/domain/samplequestion"
)

// SQLSampleQuestionRepository 使用关系型数据库持久化示例问题。
type SQLSampleQuestionRepository struct {
	database *SQLDB
}

// NewSQLSampleQuestionRepository 创建示例问题 SQL 仓储。
func NewSQLSampleQuestionRepository(database *sql.DB) *SQLSampleQuestionRepository {
	return &SQLSampleQuestionRepository{database: newSQLDB(database)}
}

// Bootstrap 初始化示例问题表结构。
func (r *SQLSampleQuestionRepository) Bootstrap() error {
	_, err := r.database.Exec(`
CREATE TABLE IF NOT EXISTS agentrag_sample_questions (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    question TEXT NOT NULL,
    create_time TIMESTAMP NOT NULL,
    update_time TIMESTAMP NOT NULL
)`)
	return err
}

// LoadSampleQuestions 从数据库加载全部示例问题。
func (r *SQLSampleQuestionRepository) LoadSampleQuestions() ([]domainsamplequestion.Item, error) {
	rows, err := r.database.Query(`
SELECT id, title, description, question, create_time, update_time
FROM agentrag_sample_questions
ORDER BY update_time DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []domainsamplequestion.Item{}
	for rows.Next() {
		var item domainsamplequestion.Item
		if err := rows.Scan(&item.ID, &item.Title, &item.Description, &item.Question, &item.CreateTime, &item.UpdateTime); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// SaveSampleQuestions 覆盖保存当前示例问题集合。
func (r *SQLSampleQuestionRepository) SaveSampleQuestions(items []domainsamplequestion.Item) error {
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
	if err := deleteMissingRows(tx, "agentrag_sample_questions", "id", ids); err != nil {
		_ = tx.Rollback()
		return err
	}
	stmt, err := tx.Prepare(`
INSERT INTO agentrag_sample_questions (id, title, description, question, create_time, update_time)
VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT (id) DO UPDATE SET
    title = excluded.title,
    description = excluded.description,
    question = excluded.question,
    create_time = excluded.create_time,
    update_time = excluded.update_time`)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	defer stmt.Close()

	for _, item := range items {
		if _, err := stmt.Exec(item.ID, item.Title, item.Description, item.Question, normalizeTime(item.CreateTime), normalizeTime(item.UpdateTime)); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}
