package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfig_SaveAndLoad(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("APPDATA", tempDir)        // Windows
	t.Setenv("XDG_CONFIG_HOME", tempDir) // Linux
	t.Setenv("HOME", tempDir)           // macOS

	cfg := &Config{
		ServerURL: "http://localhost:3000",
		Token:     "test-token-123",
		Username:  "alice",
	}

	if err := SaveConfig(cfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	path, err := GetConfigPath()
	if err != nil {
		t.Fatalf("failed to get config path: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("config file was not created: %v", err)
	}

	loaded, err := LoadConfig()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if loaded.ServerURL != cfg.ServerURL || loaded.Token != cfg.Token || loaded.Username != cfg.Username {
		t.Errorf("loaded config does not match saved config: %+v vs %+v", loaded, cfg)
	}
}

func TestConfig_DefaultValues(t *testing.T) {
	tempDir := t.TempDir()
	emptyDir := filepath.Join(tempDir, "empty")
	t.Setenv("APPDATA", emptyDir)
	t.Setenv("XDG_CONFIG_HOME", emptyDir)
	t.Setenv("HOME", emptyDir)

	loaded, err := LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error on missing config: %v", err)
	}
	if loaded.ServerURL != DefaultServerURL {
		t.Errorf("expected default server url %s, got %s", DefaultServerURL, loaded.ServerURL)
	}
	if loaded.Token != "" {
		t.Errorf("expected empty token, got %s", loaded.Token)
	}
}
