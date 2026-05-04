package db

import (
	"database/sql"
	"encoding/json"

	domainintenttree "github.com/AmazingCYJ/AgentRAG/internal/domain/intenttree"
)

// SQLIntentTreeRepository 使用关系型数据库持久化意图节点。
type SQLIntentTreeRepository struct {
	database *sql.DB
}

// NewSQLIntentTreeRepository 创建意图节点 SQL 仓储。
func NewSQLIntentTreeRepository(database *sql.DB) *SQLIntentTreeRepository {
	return &SQLIntentTreeRepository{database: database}
}

// Bootstrap 初始化意图节点表结构。
func (r *SQLIntentTreeRepository) Bootstrap() error {
	_, err := r.database.Exec(`
CREATE TABLE IF NOT EXISTS agentrag_intent_nodes (
    id                    TEXT         NOT NULL PRIMARY KEY,
    kb_id                 TEXT,
    intent_code           TEXT         NOT NULL,
    name                  TEXT         NOT NULL,
    level                 INTEGER      NOT NULL,
    parent_code           TEXT,
    description           TEXT,
    examples              TEXT,
    collection_name       TEXT,
    top_k                 INTEGER,
    mcp_tool_id           TEXT,
    kind                  INTEGER      NOT NULL DEFAULT 0,
    prompt_snippet        TEXT,
    prompt_template       TEXT,
    param_prompt_template TEXT,
    sort_order            INTEGER      NOT NULL DEFAULT 0,
    enabled               INTEGER      NOT NULL DEFAULT 1,
    create_time           TIMESTAMP    NOT NULL,
    update_time           TIMESTAMP    NOT NULL,
    deleted               INTEGER      NOT NULL DEFAULT 0
)`)
	return err
}

// LoadNodes 从数据库加载全部意图节点（过滤已删除）。
func (r *SQLIntentTreeRepository) LoadNodes() ([]domainintenttree.Node, error) {
	rows, err := r.database.Query(`
SELECT id, kb_id, intent_code, name, level, parent_code, description, examples,
       collection_name, top_k, mcp_tool_id, kind, prompt_snippet, prompt_template,
       param_prompt_template, sort_order, enabled, create_time, update_time
FROM agentrag_intent_nodes
WHERE deleted = 0
ORDER BY create_time ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nodes []domainintenttree.Node
	for rows.Next() {
		var node domainintenttree.Node
		var kbID, parentCode, description, examples, collectionName, mcpToolID sql.NullString
		var topK sql.NullInt64
		var promptSnippet, promptTemplate, paramPromptTemplate sql.NullString

		if err := rows.Scan(
			&node.ID, &kbID, &node.IntentCode, &node.Name, &node.Level,
			&parentCode, &description, &examples,
			&collectionName, &topK, &mcpToolID,
			&node.Kind, &promptSnippet, &promptTemplate, &paramPromptTemplate,
			&node.SortOrder, &node.Enabled,
			&node.CreateTime, &node.UpdateTime,
		); err != nil {
			return nil, err
		}
		node.KBID = kbID.String
		node.ParentCode = parentCode.String
		node.Description = description.String
		node.CollectionName = collectionName.String
		node.MCPToolID = mcpToolID.String
		node.PromptSnippet = promptSnippet.String
		node.PromptTemplate = promptTemplate.String
		node.ParamPromptTemplate = paramPromptTemplate.String
		if topK.Valid {
			v := int(topK.Int64)
			node.TopK = &v
		}
		if examples.Valid && examples.String != "" {
			var exList []string
			if err := json.Unmarshal([]byte(examples.String), &exList); err == nil {
				node.Examples = exList
			}
		}
		nodes = append(nodes, node)
	}
	return nodes, rows.Err()
}

// SaveNodes 覆盖保存当前意图节点集合。
func (r *SQLIntentTreeRepository) SaveNodes(nodes []domainintenttree.Node) error {
	tx, err := r.database.Begin()
	if err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM agentrag_intent_nodes`); err != nil {
		_ = tx.Rollback()
		return err
	}
	stmt, err := tx.Prepare(`
INSERT INTO agentrag_intent_nodes
    (id, kb_id, intent_code, name, level, parent_code, description, examples,
     collection_name, top_k, mcp_tool_id, kind, prompt_snippet, prompt_template,
     param_prompt_template, sort_order, enabled, create_time, update_time, deleted)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 0)`)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	defer stmt.Close()

	for _, node := range nodes {
		var examplesJSON *string
		if len(node.Examples) > 0 {
			b, _ := json.Marshal(node.Examples)
			s := string(b)
			examplesJSON = &s
		}
		var topK *int
		if node.TopK != nil {
			v := *node.TopK
			topK = &v
		}
		if _, err := stmt.Exec(
			node.ID, nullableString(node.KBID), node.IntentCode, node.Name, node.Level,
			nullableString(node.ParentCode), nullableString(node.Description), examplesJSON,
			nullableString(node.CollectionName), topK, nullableString(node.MCPToolID),
			node.Kind, nullableString(node.PromptSnippet), nullableString(node.PromptTemplate),
			nullableString(node.ParamPromptTemplate), node.SortOrder, node.Enabled,
			normalizeTime(node.CreateTime), normalizeTime(node.UpdateTime),
		); err != nil {
			_ = tx.Rollback()
						return err
		}
	}
	return tx.Commit()
}

func nullableString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
