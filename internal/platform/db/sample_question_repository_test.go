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
