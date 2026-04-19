package samplequestion

import (
	"path/filepath"
	"testing"

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
