package db

import (
	"database/sql"
	"testing"
	"time"

	domainsamplequestion "github.com/AmazingCYJ/AgentRAG/internal/domain/samplequestion"
	_ "modernc.org/sqlite"
)

func TestSQLSampleQuestionRepositorySavesAndLoadsItems(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite database failed: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	repository := NewSQLSampleQuestionRepository(database)
	if err := repository.Bootstrap(); err != nil {
		t.Fatalf("bootstrap sample question table failed: %v", err)
	}
	now := time.Date(2026, 4, 27, 14, 30, 0, 0, time.UTC)
	if err := repository.SaveSampleQuestions([]domainsamplequestion.Item{
		{
			ID:          "sq_1",
			Title:       "标题",
			Description: "描述",
			Question:    "问题",
			CreateTime:  now,
			UpdateTime:  now.Add(time.Minute),
		},
	}); err != nil {
		t.Fatalf("save sample questions failed: %v", err)
	}

	loaded, err := repository.LoadSampleQuestions()
	if err != nil {
		t.Fatalf("load sample questions failed: %v", err)
	}
	if len(loaded) != 1 {
		t.Fatalf("expected 1 sample question, got %d", len(loaded))
	}
	if loaded[0].ID != "sq_1" || loaded[0].Question != "问题" {
		t.Fatalf("unexpected loaded sample question %#v", loaded[0])
	}
}

func TestSQLSampleQuestionRepositoryReconcilesSavedItems(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite database failed: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	repository := NewSQLSampleQuestionRepository(database)
	if err := repository.Bootstrap(); err != nil {
		t.Fatalf("bootstrap sample question table failed: %v", err)
	}
	now := time.Date(2026, 5, 10, 11, 0, 0, 0, time.UTC)
	if err := repository.SaveSampleQuestions([]domainsamplequestion.Item{
		{ID: "sq_keep", Title: "旧标题", Description: "旧描述", Question: "旧问题", CreateTime: now, UpdateTime: now},
		{ID: "sq_remove", Title: "删除", Question: "删除问题", CreateTime: now.Add(time.Minute), UpdateTime: now.Add(time.Minute)},
	}); err != nil {
		t.Fatalf("save initial sample questions failed: %v", err)
	}
	if err := repository.SaveSampleQuestions([]domainsamplequestion.Item{
		{ID: "sq_keep", Title: "新标题", Description: "新描述", Question: "新问题", CreateTime: now, UpdateTime: now.Add(2 * time.Minute)},
		{ID: "sq_new", Title: "新增", Question: "新增问题", CreateTime: now.Add(3 * time.Minute), UpdateTime: now.Add(3 * time.Minute)},
	}); err != nil {
		t.Fatalf("save reconciled sample questions failed: %v", err)
	}

	loaded, err := repository.LoadSampleQuestions()
	if err != nil {
		t.Fatalf("load reconciled sample questions failed: %v", err)
	}
	if len(loaded) != 2 {
		t.Fatalf("expected 2 sample questions after reconcile, got %d", len(loaded))
	}
	itemsByID := map[string]domainsamplequestion.Item{}
	for _, item := range loaded {
		itemsByID[item.ID] = item
	}
	if _, ok := itemsByID["sq_remove"]; ok {
		t.Fatal("expected missing sample question to be deleted")
	}
	if itemsByID["sq_keep"].Title != "新标题" || itemsByID["sq_keep"].Question != "新问题" {
		t.Fatalf("expected existing sample question to be updated, got %#v", itemsByID["sq_keep"])
	}
	if itemsByID["sq_new"].Question != "新增问题" {
		t.Fatalf("expected new sample question to be inserted, got %#v", itemsByID["sq_new"])
	}

	if err := repository.SaveSampleQuestions(nil); err != nil {
		t.Fatalf("save empty sample questions failed: %v", err)
	}
	loaded, err = repository.LoadSampleQuestions()
	if err != nil {
		t.Fatalf("load empty sample questions failed: %v", err)
	}
	if len(loaded) != 0 {
		t.Fatalf("expected empty sample question snapshot to clear table, got %#v", loaded)
	}
}

func TestSQLSampleQuestionRepositoryRejectsDuplicateIDs(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite database failed: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	repository := NewSQLSampleQuestionRepository(database)
	if err := repository.Bootstrap(); err != nil {
		t.Fatalf("bootstrap sample question table failed: %v", err)
	}
	now := time.Date(2026, 5, 10, 13, 30, 0, 0, time.UTC)
	if err := repository.SaveSampleQuestions([]domainsamplequestion.Item{
		{ID: "sq_keep", Title: "旧标题", Question: "旧问题", CreateTime: now, UpdateTime: now},
	}); err != nil {
		t.Fatalf("save initial sample question failed: %v", err)
	}

	err = repository.SaveSampleQuestions([]domainsamplequestion.Item{
		{ID: "sq_keep", Title: "新标题", Question: "新问题", CreateTime: now, UpdateTime: now},
		{ID: "sq_keep", Title: "重复", Question: "重复问题", CreateTime: now, UpdateTime: now},
	})
	if err == nil {
		t.Fatal("expected duplicate id error")
	}

	loaded, err := repository.LoadSampleQuestions()
	if err != nil {
		t.Fatalf("load sample questions after duplicate id failure failed: %v", err)
	}
	if len(loaded) != 1 || loaded[0].Title != "旧标题" {
		t.Fatalf("expected existing sample question to remain unchanged, got %#v", loaded)
	}
}
