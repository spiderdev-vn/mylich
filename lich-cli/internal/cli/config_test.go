package cli

import (
	"path/filepath"
	"testing"

	"lich-cli/internal/config"
)

func TestConfig_Commands(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")
	t.Setenv("APPDATA", tempDir)
	t.Setenv("XDG_CONFIG_HOME", tempDir)

	cfg := &config.Config{
		ServerURL: "http://127.0.0.1:3000",
		Icons:     "unicode",
	}
	_ = config.SaveConfig(cfg)

	// 1. Test get
	err := RunConfig([]string{"get", "icons"})
	if err != nil {
		t.Fatalf("RunConfig get icons failed: %v", err)
	}

	// 2. Test set valid icon preset
	err = RunConfig([]string{"set", "icons", "nerd", "--simple"})
	if err != nil {
		t.Fatalf("RunConfig set icons nerd failed: %v", err)
	}

	updated, err := config.LoadConfig()
	if err != nil || updated.Icons != "nerd" {
		t.Errorf("expected updated icons to be 'nerd', got '%s'", updated.Icons)
	}

	// 3. Test set invalid icon preset
	err = RunConfig([]string{"set", "icons", "invalid_theme"})
	if err == nil {
		t.Errorf("expected error when setting invalid icon theme, got nil")
	}

	// 4. Test list
	err = RunConfig([]string{"list", "--simple"})
	if err != nil {
		t.Fatalf("RunConfig list failed: %v", err)
	}

	// 5. Test JSON output
	err = RunConfig([]string{"list", "--json"})
	if err != nil {
		t.Fatalf("RunConfig list --json failed: %v", err)
	}
	_ = configPath
}
