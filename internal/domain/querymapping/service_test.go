package querymapping

import (
	"path/filepath"
	"testing"
	"time"

	platformstate "github.com/AmazingCYJ/AgentRAG/internal/platform/state"
)

func TestQueryMappingsPersistAcrossServiceRecreation(t *testing.T) {
	store, err := platformstate.NewFileStore(filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatalf("create state store failed: %v", err)
	}

	service := NewService(store)
	id, err := service.Create(SaveRequest{
		SourceTerm: "oa",
		TargetTerm: "OA",
		MatchType:  1,
		Priority:   1,
		Enabled:    true,
		Remark:     "test",
	})
	if err != nil {
		t.Fatalf("create mapping failed: %v", err)
	}

	recreated := NewService(store)
	item, err := recreated.GetByID(id)
	if err != nil {
		t.Fatalf("get recreated mapping failed: %v", err)
	}
	if item.TargetTerm != "OA" {
		t.Fatalf("expected recreated target term OA, got %s", item.TargetTerm)
	}
}

type memoryQueryMappingRepository struct {
	items []Item
}

func (r *memoryQueryMappingRepository) LoadQueryMappings() ([]Item, error) {
	result := make([]Item, len(r.items))
	copy(result, r.items)
	return result, nil
}

func (r *memoryQueryMappingRepository) SaveQueryMappings(items []Item) error {
	r.items = make([]Item, len(items))
	copy(r.items, items)
	return nil
}

func TestServiceLoadsAndPersistsThroughRepository(t *testing.T) {
	now := time.Date(2026, 4, 27, 15, 0, 0, 0, time.UTC)
	repo := &memoryQueryMappingRepository{
		items: []Item{
			{
				ID:         "map_db",
				SourceTerm: "oa系统",
				TargetTerm: "OA",
				MatchType:  1,
				Priority:   1,
				Enabled:    true,
				Remark:     "数据库记录",
				CreateTime: now,
				UpdateTime: now,
			},
		},
	}
	service := NewServiceWithRepository(repo)

	loaded, err := service.GetByID("map_db")
	if err != nil {
		t.Fatalf("get repository mapping failed: %v", err)
	}
	if loaded.TargetTerm != "OA" {
		t.Fatalf("expected repository target term OA, got %s", loaded.TargetTerm)
	}
	service.newID = func() string { return "map_new" }
	service.now = func() time.Time { return now.Add(time.Hour) }
	if _, err := service.Create(SaveRequest{SourceTerm: "hr", TargetTerm: "HR", Enabled: true}); err != nil {
		t.Fatalf("create mapping failed: %v", err)
	}
	if _, ok := findQueryMapping(repo.items, "map_new"); !ok {
		t.Fatal("expected created mapping to be saved through repository")
	}
}

func findQueryMapping(items []Item, id string) (Item, bool) {
	for _, item := range items {
		if item.ID == id {
			return item, true
		}
	}
	return Item{}, false
}
