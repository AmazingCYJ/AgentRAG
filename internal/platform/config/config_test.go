package config

import "testing"

func TestLoadParsesHTTPPort(t *testing.T) {
	cfg, err := Load("testdata/config.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.HTTP.Port != 8080 {
		t.Fatalf("expected 8080, got %d", cfg.HTTP.Port)
	}
}
