package auth

import (
	"testing"
	"time"

	appconfig "github.com/AmazingCYJ/AgentRAG/internal/platform/config"
)

func TestLoginRejectsUnknownUser(t *testing.T) {
	service := NewService(appconfig.AuthConfig{
		JWTSecret: "test-secret",
		TokenTTL:  time.Hour,
		Bootstrap: appconfig.BootstrapUserConfig{
			UserID:   "u_admin",
			Username: "admin",
			Password: "admin123",
			Role:     "admin",
		},
	})

	_, err := service.Login("nobody", "bad-password")
	if err == nil {
		t.Fatal("expected auth error")
	}
}

func TestLoginReturnsTokenForBootstrapUser(t *testing.T) {
	service := NewService(appconfig.AuthConfig{
		JWTSecret: "test-secret",
		TokenTTL:  time.Hour,
		Bootstrap: appconfig.BootstrapUserConfig{
			UserID:   "u_admin",
			Username: "admin",
			Password: "admin123",
			Role:     "admin",
			Avatar:   "https://example.com/avatar.png",
		},
	})

	result, err := service.Login("admin", "admin123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Token == "" {
		t.Fatal("expected token to be generated")
	}
	if result.UserID != "u_admin" {
		t.Fatalf("expected user id u_admin, got %s", result.UserID)
	}
}
