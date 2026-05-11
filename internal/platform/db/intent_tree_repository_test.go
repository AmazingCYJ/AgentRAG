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

func TestSQLIntentTreeRepositoryReconcilesSavedNodes(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite database failed: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	repository := NewSQLIntentTreeRepository(database)
	if err := repository.Bootstrap(); err != nil {
		t.Fatalf("bootstrap intent tree table failed: %v", err)
	}

	now := time.Date(2026, 5, 11, 10, 0, 0, 0, time.UTC)
	oldTopK := 5
	removeTopK := 3
	if err := repository.SaveNodes([]domainintenttree.Node{
		{
			ID:                  "node_keep",
			KBID:                "kb_old",
			IntentCode:          "keep",
			Name:                "旧节点",
			Level:               0,
			Description:         "旧描述",
			Examples:            []string{"旧例子"},
			CollectionName:      "collection_old",
			MCPToolID:           "tool_old",
			TopK:                &oldTopK,
			Kind:                1,
			SortOrder:           20,
			Enabled:             1,
			PromptSnippet:       "旧片段",
			PromptTemplate:      "旧模板",
			ParamPromptTemplate: "旧参数模板",
			CreateTime:          now,
			UpdateTime:          now,
		},
		{
			ID:             "node_remove",
			IntentCode:     "remove",
			Name:           "待删除节点",
			Level:          0,
			CollectionName: "collection_remove",
			TopK:           &removeTopK,
			Kind:           1,
			SortOrder:      30,
			Enabled:        1,
			CreateTime:     now.Add(time.Minute),
			UpdateTime:     now.Add(time.Minute),
		},
	}); err != nil {
		t.Fatalf("save initial intent nodes failed: %v", err)
	}
	if _, err := database.Exec(`UPDATE agentrag_intent_nodes SET deleted = 1 WHERE id = ?`, "node_keep"); err != nil {
		t.Fatalf("mark stale intent node deleted failed: %v", err)
	}

	if err := repository.SaveNodes([]domainintenttree.Node{
		{
			ID:          "node_new",
			IntentCode:  "parent",
			Name:        "新增父节点",
			Level:       0,
			Description: "新增描述",
			Examples:    []string{"新增例子"},
			Kind:        0,
			SortOrder:   1,
			Enabled:     1,
			CreateTime:  now.Add(2 * time.Minute),
			UpdateTime:  now.Add(2 * time.Minute),
		},
		{
			ID:             "node_keep",
			IntentCode:     "keep_updated",
			Name:           "新节点",
			Level:          1,
			ParentCode:     "parent",
			Kind:           2,
			SortOrder:      5,
			Enabled:        0,
			PromptTemplate: "新模板",
			CreateTime:     now.Add(3 * time.Minute),
			UpdateTime:     now.Add(4 * time.Minute),
		},
	}); err != nil {
		t.Fatalf("save reconciled intent nodes failed: %v", err)
	}

	loaded, err := repository.LoadNodes()
	if err != nil {
		t.Fatalf("load reconciled intent nodes failed: %v", err)
	}
	if len(loaded) != 2 {
		t.Fatalf("expected 2 intent nodes after reconcile, got %d", len(loaded))
	}
	nodesByID := map[string]domainintenttree.Node{}
	for _, node := range loaded {
		nodesByID[node.ID] = node
	}
	if _, ok := nodesByID["node_remove"]; ok {
		t.Fatal("expected missing intent node to be deleted")
	}
	keep, ok := nodesByID["node_keep"]
	if !ok {
		t.Fatalf("expected existing intent node to be restored and updated, got %#v", loaded)
	}
	if keep.IntentCode != "keep_updated" || keep.Name != "新节点" || keep.Level != 1 || keep.ParentCode != "parent" || keep.Kind != 2 || keep.Enabled != 0 {
		t.Fatalf("expected existing intent node fields to be updated, got %#v", keep)
	}
	if keep.KBID != "" || keep.Description != "" || keep.CollectionName != "" || keep.MCPToolID != "" || keep.TopK != nil || len(keep.Examples) != 0 {
		t.Fatalf("expected nullable intent node fields to be cleared, got %#v", keep)
	}
	if keep.PromptSnippet != "" || keep.PromptTemplate != "新模板" || keep.ParamPromptTemplate != "" {
		t.Fatalf("expected prompt fields to be reconciled, got %#v", keep)
	}
	if nodesByID["node_new"].Name != "新增父节点" {
		t.Fatalf("expected new intent node to be inserted, got %#v", nodesByID["node_new"])
	}

	if err := repository.SaveNodes(nil); err != nil {
		t.Fatalf("save empty intent nodes failed: %v", err)
	}
	loaded, err = repository.LoadNodes()
	if err != nil {
		t.Fatalf("load empty intent nodes failed: %v", err)
	}
	if len(loaded) != 0 {
		t.Fatalf("expected empty intent node snapshot to clear table, got %#v", loaded)
	}
}

func TestSQLIntentTreeRepositoryRejectsDuplicateIDs(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite database failed: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	repository := NewSQLIntentTreeRepository(database)
	if err := repository.Bootstrap(); err != nil {
		t.Fatalf("bootstrap intent tree table failed: %v", err)
	}

	now := time.Date(2026, 5, 11, 11, 0, 0, 0, time.UTC)
	if err := repository.SaveNodes([]domainintenttree.Node{
		{
			ID:         "node_keep",
			IntentCode: "keep",
			Name:       "旧节点",
			Level:      0,
			Enabled:    1,
			CreateTime: now,
			UpdateTime: now,
		},
	}); err != nil {
		t.Fatalf("save initial intent node failed: %v", err)
	}

	err = repository.SaveNodes([]domainintenttree.Node{
		{
			ID:         "node_keep",
			IntentCode: "keep_updated",
			Name:       "新节点",
			Level:      0,
			Enabled:    1,
			CreateTime: now,
			UpdateTime: now,
		},
		{
			ID:         "node_keep",
			IntentCode: "duplicated",
			Name:       "重复节点",
			Level:      0,
			Enabled:    1,
			CreateTime: now,
			UpdateTime: now,
		},
	})
	if err == nil {
		t.Fatal("expected duplicate id error")
	}

	loaded, err := repository.LoadNodes()
	if err != nil {
		t.Fatalf("load intent nodes after duplicate id failure failed: %v", err)
	}
	if len(loaded) != 1 || loaded[0].Name != "旧节点" || loaded[0].IntentCode != "keep" {
		t.Fatalf("expected existing intent node to remain unchanged, got %#v", loaded)
	}
}
