package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewAppReturnsConfigErrorWhenConfigPathMissing(t *testing.T) {
	_, err := NewApp("/tmp/not-exists.yaml")
	if err == nil {
		t.Fatal("expected config error")
	}
}

func TestNewAppCreatesHTTPServer(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte("http:\n  port: 8080\n"), 0o644); err != nil {
		t.Fatalf("write config failed: %v", err)
	}

	instance, err := NewApp(configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if instance.HTTPServer == nil {
		t.Fatal("expected HTTP server to be initialized")
	}
}
