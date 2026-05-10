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

func TestSQLQueryMappingRepositoryReconcilesSavedItems(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite database failed: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	repository := NewSQLQueryMappingRepository(database)
	if err := repository.Bootstrap(); err != nil {
		t.Fatalf("bootstrap query mapping table failed: %v", err)
	}
	now := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	if err := repository.SaveQueryMappings([]domainquerymapping.Item{
		{ID: "map_keep", SourceTerm: "oa", TargetTerm: "OA", MatchType: 1, Priority: 2, Enabled: true, CreateTime: now, UpdateTime: now},
		{ID: "map_remove", SourceTerm: "remove", TargetTerm: "REMOVE", MatchType: 1, Priority: 3, Enabled: true, CreateTime: now.Add(time.Minute), UpdateTime: now.Add(time.Minute)},
	}); err != nil {
		t.Fatalf("save initial query mappings failed: %v", err)
	}
	if err := repository.SaveQueryMappings([]domainquerymapping.Item{
		{ID: "map_keep", SourceTerm: "oa系统", TargetTerm: "办公自动化", MatchType: 2, Priority: 1, Enabled: false, Remark: "更新", CreateTime: now, UpdateTime: now.Add(2 * time.Minute)},
		{ID: "map_new", SourceTerm: "hr", TargetTerm: "人事", MatchType: 1, Priority: 4, Enabled: true, CreateTime: now.Add(3 * time.Minute), UpdateTime: now.Add(3 * time.Minute)},
	}); err != nil {
		t.Fatalf("save reconciled query mappings failed: %v", err)
	}

	loaded, err := repository.LoadQueryMappings()
	if err != nil {
		t.Fatalf("load reconciled query mappings failed: %v", err)
	}
	if len(loaded) != 2 {
		t.Fatalf("expected 2 query mappings after reconcile, got %d", len(loaded))
	}
	itemsByID := map[string]domainquerymapping.Item{}
	for _, item := range loaded {
		itemsByID[item.ID] = item
	}
	if _, ok := itemsByID["map_remove"]; ok {
		t.Fatal("expected missing query mapping to be deleted")
	}
	if itemsByID["map_keep"].TargetTerm != "办公自动化" || itemsByID["map_keep"].MatchType != 2 || itemsByID["map_keep"].Enabled {
		t.Fatalf("expected existing query mapping to be updated, got %#v", itemsByID["map_keep"])
	}
	if itemsByID["map_new"].TargetTerm != "人事" {
		t.Fatalf("expected new query mapping to be inserted, got %#v", itemsByID["map_new"])
	}

	if err := repository.SaveQueryMappings(nil); err != nil {
		t.Fatalf("save empty query mappings failed: %v", err)
	}
	loaded, err = repository.LoadQueryMappings()
	if err != nil {
		t.Fatalf("load empty query mappings failed: %v", err)
	}
	if len(loaded) != 0 {
		t.Fatalf("expected empty query mapping snapshot to clear table, got %#v", loaded)
	}
}

func TestSQLQueryMappingRepositoryRejectsDuplicateIDs(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite database failed: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	repository := NewSQLQueryMappingRepository(database)
	if err := repository.Bootstrap(); err != nil {
		t.Fatalf("bootstrap query mapping table failed: %v", err)
	}
	now := time.Date(2026, 5, 10, 13, 45, 0, 0, time.UTC)
	if err := repository.SaveQueryMappings([]domainquerymapping.Item{
		{ID: "map_keep", SourceTerm: "oa", TargetTerm: "OA", MatchType: 1, Priority: 1, Enabled: true, CreateTime: now, UpdateTime: now},
	}); err != nil {
		t.Fatalf("save initial query mapping failed: %v", err)
	}

	err = repository.SaveQueryMappings([]domainquerymapping.Item{
		{ID: "map_keep", SourceTerm: "oa", TargetTerm: "办公自动化", MatchType: 1, Priority: 1, Enabled: true, CreateTime: now, UpdateTime: now},
		{ID: "map_keep", SourceTerm: "oa2", TargetTerm: "重复", MatchType: 1, Priority: 2, Enabled: true, CreateTime: now, UpdateTime: now},
	})
	if err == nil {
		t.Fatal("expected duplicate id error")
	}

	loaded, err := repository.LoadQueryMappings()
	if err != nil {
		t.Fatalf("load query mappings after duplicate id failure failed: %v", err)
	}
	if len(loaded) != 1 || loaded[0].TargetTerm != "OA" {
		t.Fatalf("expected existing query mapping to remain unchanged, got %#v", loaded)
	}
}
