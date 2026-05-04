package intenttree

import (
	"path/filepath"
	"testing"

	platformstate "github.com/AmazingCYJ/AgentRAG/internal/platform/state"
)

func TestIntentTreePersistsAcrossServiceRecreation(t *testing.T) {
	store, err := platformstate.NewFileStore(filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatalf("create state store failed: %v", err)
	}

	service := NewService(store)
	rootID, err := service.CreateNode(CreateRequest{
		IntentCode: "biz_oa",
		Name:       "OA系统",
		Level:      0,
		Kind:       0,
		Enabled:    1,
	})
	if err != nil {
		t.Fatalf("create root node failed: %v", err)
	}
	_, err = service.CreateNode(CreateRequest{
		IntentCode: "biz_oa_leave",
		Name:       "请假",
		Level:      1,
		ParentCode: "biz_oa",
		Kind:       2,
		Enabled:    1,
	})
	if err != nil {
		t.Fatalf("create child node failed: %v", err)
	}

	recreated := NewService(store)
	tree := recreated.GetFullTree()
	if len(tree) != 1 {
		t.Fatalf("expected 1 root node, got %d", len(tree))
	}
	if tree[0].ID != rootID {
		t.Fatalf("expected recreated root id %s, got %s", rootID, tree[0].ID)
	}
	if len(tree[0].Children) != 1 {
		t.Fatalf("expected 1 child node, got %d", len(tree[0].Children))
	}
}
