package db

import (
	"database/sql"
	"time"

	domainusermgmt "github.com/AmazingCYJ/AgentRAG/internal/domain/usermgmt"
)

// SQLUserRepository 使用关系型数据库持久化后台用户。
type SQLUserRepository struct {
	database *SQLDB
}

// NewSQLUserRepository 创建用户 SQL 仓储。
func NewSQLUserRepository(database *sql.DB) *SQLUserRepository {
	return &SQLUserRepository{database: newSQLDB(database)}
}

// Bootstrap 初始化用户表结构。
func (r *SQLUserRepository) Bootstrap() error {
	_, err := r.database.Exec(`
CREATE TABLE IF NOT EXISTS agentrag_users (
    id TEXT PRIMARY KEY,
    username TEXT NOT NULL UNIQUE,
    password TEXT NOT NULL,
    role TEXT NOT NULL,
    avatar TEXT NOT NULL DEFAULT '',
    create_time TIMESTAMP NOT NULL,
    update_time TIMESTAMP NOT NULL
)`)
	return err
}

// LoadUsers 从数据库加载全部后台用户。
func (r *SQLUserRepository) LoadUsers() ([]domainusermgmt.User, error) {
	rows, err := r.database.Query(`
SELECT id, username, password, role, avatar, create_time, update_time
FROM agentrag_users
ORDER BY create_time ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := []domainusermgmt.User{}
	for rows.Next() {
		var user domainusermgmt.User
		if err := rows.Scan(&user.ID, &user.Username, &user.Password, &user.Role, &user.Avatar, &user.CreateTime, &user.UpdateTime); err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, rows.Err()
}

// SaveUsers 覆盖保存当前内存中的后台用户集合。
func (r *SQLUserRepository) SaveUsers(users []domainusermgmt.User) error {
	ids := make([]string, 0, len(users))
	for _, user := range users {
		ids = append(ids, user.ID)
	}
	if err := rejectDuplicateIDs(ids); err != nil {
		return err
	}

	tx, err := r.database.Begin()
	if err != nil {
		return err
	}
	replaceAll, err := userSnapshotNeedsFullReplace(tx, users, ids)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	if replaceAll {
		_, err = tx.Exec(`DELETE FROM agentrag_users`)
	} else {
		err = deleteMissingRows(tx, "agentrag_users", "id", ids)
	}
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	stmt, err := tx.Prepare(`
INSERT INTO agentrag_users (id, username, password, role, avatar, create_time, update_time)
VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (id) DO UPDATE SET
    username = excluded.username,
    password = excluded.password,
    role = excluded.role,
    avatar = excluded.avatar,
    create_time = excluded.create_time,
    update_time = excluded.update_time`)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	defer stmt.Close()

	for _, user := range users {
		createTime := normalizeTime(user.CreateTime)
		updateTime := normalizeTime(user.UpdateTime)
		if _, err := stmt.Exec(user.ID, user.Username, user.Password, user.Role, user.Avatar, createTime, updateTime); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

func userSnapshotNeedsFullReplace(tx *SQLTx, users []domainusermgmt.User, ids []string) (bool, error) {
	if len(ids) == 0 {
		return false, nil
	}
	desiredOwnerByUsername := make(map[string]string, len(users))
	for _, user := range users {
		desiredOwnerByUsername[user.Username] = user.ID
	}

	rows, err := tx.Query("SELECT id, username FROM agentrag_users WHERE id IN ("+questionPlaceholders(len(ids))+")", stringArgs(ids)...)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			id       string
			username string
		)
		if err := rows.Scan(&id, &username); err != nil {
			return false, err
		}
		if ownerID, ok := desiredOwnerByUsername[username]; ok && ownerID != id {
			return true, nil
		}
	}
	return false, rows.Err()
}

func normalizeTime(value time.Time) time.Time {
	if value.IsZero() {
		return time.Now()
	}
	return value
}
