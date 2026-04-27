package db

import (
	"database/sql"
	"testing"
	"time"

	domainquerymapping "github.com/AmazingCYJ/AgentRAG/internal/domain/querymapping"
	_ "modernc.org/sqlite"
)

func TestSQLQueryMappingRepositorySavesAndLoadsItems(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite database failed: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	repository := NewSQLQueryMappingRepository(database)
	if err := repository.Bootstrap(); err != nil {
		t.Fatalf("bootstrap query mapping table failed: %v", err)
	}
	now := time.Date(2026, 4, 27, 15, 30, 0, 0, time.UTC)
	if err := repository.SaveQueryMappings([]domainquerymapping.Item{
		{
			ID:         "map_1",
			SourceTerm: "oa系统",
			TargetTerm: "OA",
			MatchType:  2,
			Priority:   5,
			Enabled:    true,
			Remark:     "办公系统",
			CreateTime: now,
			UpdateTime: now.Add(time.Minute),
		},
	}); err != nil {
		t.Fatalf("save query mappings failed: %v", err)
	}

	loaded, err := repository.LoadQueryMappings()
	if err != nil {
		t.Fatalf("load query mappings failed: %v", err)
	}
	if len(loaded) != 1 {
		t.Fatalf("expected 1 query mapping, got %d", len(loaded))
	}
	if loaded[0].ID != "map_1" || loaded[0].TargetTerm != "OA" || loaded[0].MatchType != 2 {
		t.Fatalf("unexpected loaded query mapping %#v", loaded[0])
	}
	if !loaded[0].Enabled || loaded[0].Remark != "办公系统" {
		t.Fatalf("unexpected loaded query mapping flags %#v", loaded[0])
	}
}
