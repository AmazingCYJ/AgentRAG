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

type memoryUserRepository struct {
	users []User
}

func (r *memoryUserRepository) LoadUsers() ([]User, error) {
	result := make([]User, len(r.users))
	copy(result, r.users)
	return result, nil
}

func (r *memoryUserRepository) SaveUsers(users []User) error {
	r.users = make([]User, len(users))
	copy(r.users, users)
	return nil
}

func TestServiceLoadsAndPersistsThroughRepository(t *testing.T) {
	createdAt := time.Date(2026, 4, 27, 10, 0, 0, 0, time.UTC)
	repo := &memoryUserRepository{
		users: []User{
			{
				ID:         "u_db",
				Username:   "db_user",
				Password:   "pass123",
				Role:       "admin",
				CreateTime: createdAt,
				UpdateTime: createdAt,
			},
		},
	}
	service := NewServiceWithRepository(appconfig.AuthConfig{}, repo)

	if _, ok := service.Authenticate("db_user", "pass123"); !ok {
		t.Fatal("expected service to load user from repository")
	}
	service.newID = func() string { return "u_new" }
	service.now = func() time.Time { return createdAt.Add(time.Hour) }
	if _, err := service.Create(CreateRequest{Username: "new_user", Password: "new123", Role: "user"}); err != nil {
		t.Fatalf("create user failed: %v", err)
	}

	if len(repo.users) != 3 {
		t.Fatalf("expected bootstrap, loaded and created users persisted, got %d", len(repo.users))
	}
	if _, ok := findUser(repo.users, "u_new"); !ok {
		t.Fatal("expected created user to be saved through repository")
	}
}

func findUser(users []User, id string) (User, bool) {
	for _, user := range users {
		if user.ID == id {
			return user, true
		}
	}
	return User{}, false
}
