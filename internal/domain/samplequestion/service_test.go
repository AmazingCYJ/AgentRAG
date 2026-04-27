package samplequestion

import (
	"path/filepath"
	"testing"
	"time"

	platformstate "github.com/AmazingCYJ/AgentRAG/internal/platform/state"
)

func TestSampleQuestionsPersistAcrossServiceRecreation(t *testing.T) {
	store, err := platformstate.NewFileStore(filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatalf("create state store failed: %v", err)
	}

	service := NewService(store)
	id, err := service.Create(SaveRequest{
		Title:       "测试标题",
		Description: "测试描述",
		Question:    "测试问题",
	})
	if err != nil {
		t.Fatalf("create sample question failed: %v", err)
	}

	recreated := NewService(store)
	item, err := recreated.GetByID(id)
	if err != nil {
		t.Fatalf("get recreated sample question failed: %v", err)
	}
	if item.Question != "测试问题" {
		t.Fatalf("expected recreated question 测试问题, got %s", item.Question)
	}
}

type memorySampleQuestionRepository struct {
	items []Item
}

func (r *memorySampleQuestionRepository) LoadSampleQuestions() ([]Item, error) {
	result := make([]Item, len(r.items))
	copy(result, r.items)
	return result, nil
}

func (r *memorySampleQuestionRepository) SaveSampleQuestions(items []Item) error {
	r.items = make([]Item, len(items))
	copy(r.items, items)
	return nil
}

func TestServiceLoadsAndPersistsThroughRepository(t *testing.T) {
	now := time.Date(2026, 4, 27, 14, 0, 0, 0, time.UTC)
	repo := &memorySampleQuestionRepository{
		items: []Item{
			{
				ID:          "sq_db",
				Title:       "数据库标题",
				Description: "数据库描述",
				Question:    "数据库问题",
				CreateTime:  now,
				UpdateTime:  now,
			},
		},
	}
	service := NewServiceWithRepository(repo)

	loaded, err := service.GetByID("sq_db")
	if err != nil {
		t.Fatalf("get repository sample question failed: %v", err)
	}
	if loaded.Question != "数据库问题" {
		t.Fatalf("expected repository question 数据库问题, got %s", loaded.Question)
	}
	service.newID = func() string { return "sq_new" }
	service.now = func() time.Time { return now.Add(time.Hour) }
	if _, err := service.Create(SaveRequest{Title: "新增标题", Question: "新增问题"}); err != nil {
		t.Fatalf("create sample question failed: %v", err)
	}
	if _, ok := findSampleQuestion(repo.items, "sq_new"); !ok {
		t.Fatal("expected created sample question to be saved through repository")
	}
}

func findSampleQuestion(items []Item, id string) (Item, bool) {
	for _, item := range items {
		if item.ID == id {
			return item, true
		}
	}
	return Item{}, false
}
