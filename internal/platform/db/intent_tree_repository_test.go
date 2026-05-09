package db

import (
	"database/sql"
	"testing"
	"time"

	domainintenttree "github.com/AmazingCYJ/AgentRAG/internal/domain/intenttree"
)

func TestSQLIntentTreeRepositorySavesAndLoadsNodes(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite database failed: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	repository := NewSQLIntentTreeRepository(database)
	if err := repository.Bootstrap(); err != nil {
		t.Fatalf("bootstrap intent tree table failed: %v", err)
	}

	now := time.Date(2026, 5, 9, 10, 0, 0, 0, time.UTC)
	topK := 6
	if err := repository.SaveNodes([]domainintenttree.Node{
		{
			ID:             "node_kb",
			KBID:           "kb_finance",
			IntentCode:     "finance_policy",
			Name:           "财务制度",
			Level:          0,
			Description:    "查询财务制度知识库",
			Examples:       []string{"报销规则", "差旅标准"},
			CollectionName: "finance_docs",
			TopK:           &topK,
			Kind:           1,
			SortOrder:      10,
			Enabled:        1,
			PromptSnippet:  "优先引用制度原文",
			PromptTemplate: "你是财务制度助手",
			CreateTime:     now,
			UpdateTime:     now.Add(time.Minute),
		},
		{
			ID:                  "node_tool",
			IntentCode:          "ticket_status",
			Name:                "工单状态",
			Level:               1,
			ParentCode:          "finance_policy",
			Description:         "查询工单状态",
			Examples:            []string{"查询工单", "我的工单进度"},
			MCPToolID:           "ticket_query",
			Kind:                2,
			SortOrder:           20,
			Enabled:             1,
			ParamPromptTemplate: "抽取工单查询参数",
			CreateTime:          now.Add(time.Second),
			UpdateTime:          now.Add(2 * time.Minute),
		},
	}); err != nil {
		t.Fatalf("save intent nodes failed: %v", err)
	}

	loaded, err := repository.LoadNodes()
	if err != nil {
		t.Fatalf("load intent nodes failed: %v", err)
	}
	if len(loaded) != 2 {
		t.Fatalf("expected 2 intent nodes, got %d", len(loaded))
	}

	root := loaded[0]
	if root.ID != "node_kb" || root.KBID != "kb_finance" || root.IntentCode != "finance_policy" {
		t.Fatalf("unexpected root node %#v", root)
	}
	if root.CollectionName != "finance_docs" || root.TopK == nil || *root.TopK != topK {
		t.Fatalf("unexpected root retrieval fields %#v", root)
	}
	if len(root.Examples) != 2 || root.Examples[0] != "报销规则" || root.Examples[1] != "差旅标准" {
		t.Fatalf("unexpected root examples %#v", root.Examples)
	}
	if root.PromptSnippet != "优先引用制度原文" || root.PromptTemplate != "你是财务制度助手" {
		t.Fatalf("unexpected root prompts %#v", root)
	}

	child := loaded[1]
	if child.ID != "node_tool" || child.ParentCode != "finance_policy" || child.MCPToolID != "ticket_query" {
		t.Fatalf("unexpected child node %#v", child)
	}
	if child.TopK != nil {
		t.Fatalf("expected nil child topK, got %#v", child.TopK)
	}
	if child.ParamPromptTemplate != "抽取工单查询参数" || child.Kind != 2 {
		t.Fatalf("unexpected child tool fields %#v", child)
	}
}
