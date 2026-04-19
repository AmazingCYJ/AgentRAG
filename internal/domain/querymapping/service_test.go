package querymapping

import (
	"path/filepath"
	"testing"

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
