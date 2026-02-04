package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewConfig(t *testing.T) {
	cfg := NewConfig()

	if cfg.Defaults.Region != "us-east-1" {
		t.Errorf("expected default region us-east-1, got %s", cfg.Defaults.Region)
	}

	if cfg.Defaults.SessionDuration != 3600 {
		t.Errorf("expected default session duration 3600, got %d", cfg.Defaults.SessionDuration)
	}

	if cfg.Profiles == nil {
		t.Error("expected profiles map to be initialized")
	}
}

func TestSetAndGetProfile(t *testing.T) {
	cfg := NewConfig()

	profile := Profile{
		URL:      "https://myapps.microsoft.com",
		AppID:    "test-app-id",
		Username: "user@example.com",
	}

	cfg.SetProfile("production", profile)

	merged, err := cfg.GetProfile("production")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if merged.URL != profile.URL {
		t.Errorf("expected URL %s, got %s", profile.URL, merged.URL)
	}

	if merged.Region != cfg.Defaults.Region {
		t.Errorf("expected region %s (from defaults), got %s", cfg.Defaults.Region, merged.Region)
	}
}

func TestGetProfileNotFound(t *testing.T) {
	cfg := NewConfig()

	_, err := cfg.GetProfile("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent profile")
	}
}

func TestSaveAndLoadConfig(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "azure2aws-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	configPath := filepath.Join(tmpDir, "config.yaml")

	// Create and save config
	cfg := NewConfig()
	cfg.Defaults.Region = "ap-northeast-1"
	cfg.SetProfile("test", Profile{
		URL:      "https://test.example.com",
		AppID:    "app-123",
		Username: "test@example.com",
	})

	if err := SaveConfig(cfg, configPath); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Load and verify
	loaded, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if loaded.Defaults.Region != "ap-northeast-1" {
		t.Errorf("expected region ap-northeast-1, got %s", loaded.Defaults.Region)
	}

	if !loaded.HasProfile("test") {
		t.Error("expected profile 'test' to exist")
	}
}

func TestLoadConfigNotFound(t *testing.T) {
	_, err := LoadConfig("/nonexistent/path/config.yaml")
	if err != ErrConfigNotFound {
		t.Errorf("expected ErrConfigNotFound, got %v", err)
	}
}

func TestProfileOverridesDefaults(t *testing.T) {
	cfg := NewConfig()
	cfg.Defaults.Region = "us-east-1"
	cfg.Defaults.SessionDuration = 3600

	cfg.SetProfile("custom", Profile{
		URL:             "https://example.com",
		Region:          "eu-west-1",
		SessionDuration: 7200,
	})

	merged, err := cfg.GetProfile("custom")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if merged.Region != "eu-west-1" {
		t.Errorf("expected region eu-west-1, got %s", merged.Region)
	}

	if merged.SessionDuration != 7200 {
		t.Errorf("expected session duration 7200, got %d", merged.SessionDuration)
	}
}
