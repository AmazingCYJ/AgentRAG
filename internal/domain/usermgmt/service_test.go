package usermgmt

import (
	"path/filepath"
	"testing"
	"time"

	appconfig "github.com/AmazingCYJ/AgentRAG/internal/platform/config"
	platformstate "github.com/AmazingCYJ/AgentRAG/internal/platform/state"
)

func TestCreatedUserPersistsAcrossServiceRecreation(t *testing.T) {
	store, err := platformstate.NewFileStore(filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatalf("create state store failed: %v", err)
	}

	cfg := appconfig.AuthConfig{
		JWTSecret: "test-secret",
		TokenTTL:  time.Hour,
		Bootstrap: appconfig.BootstrapUserConfig{
			UserID:   "u_admin",
			Username: "admin",
			Password: "admin123",
			Role:     "admin",
		},
	}

	service := NewService(cfg, store)
	userID, err := service.Create(CreateRequest{
		Username: "alice",
		Password: "alice123",
		Role:     "user",
		Avatar:   "https://example.com/avatar.png",
	})
	if err != nil {
		t.Fatalf("create user failed: %v", err)
	}
	if _, ok := service.Authenticate("alice", "alice123"); !ok {
		t.Fatal("expected created user to authenticate before recreation")
	}

	recreated := NewService(cfg, store)
	user, ok := recreated.Authenticate("alice", "alice123")
	if !ok {
		t.Fatal("expected created user to authenticate after recreation")
	}
	if user.ID != userID {
		t.Fatalf("expected recreated user id %s, got %s", userID, user.ID)
	}
}

func TestChangedBootstrapPasswordPersistsAcrossServiceRecreation(t *testing.T) {
	store, err := platformstate.NewFileStore(filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatalf("create state store failed: %v", err)
	}

	cfg := appconfig.AuthConfig{
		JWTSecret: "test-secret",
		TokenTTL:  time.Hour,
		Bootstrap: appconfig.BootstrapUserConfig{
			UserID:   "u_admin",
			Username: "admin",
			Password: "admin123",
			Role:     "admin",
		},
	}

	service := NewService(cfg, store)
	if err := service.ChangePassword("u_admin", "admin123", "admin456"); err != nil {
		t.Fatalf("change bootstrap password failed: %v", err)
	}

	recreated := NewService(cfg, store)
	if _, ok := recreated.Authenticate("admin", "admin456"); !ok {
		t.Fatal("expected updated bootstrap password to persist after recreation")
	}
	if _, ok := recreated.Authenticate("admin", "admin123"); ok {
		t.Fatal("expected old bootstrap password to be invalid after recreation")
	}
}
