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
		ServerURL:  "http://127.0.0.1:3000",
		Icons:      "unicode",
		AgendaMode: "list",
	}
	_ = config.SaveConfig(cfg)

	// 1. Test get icons
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

	// 3. Test set valid agenda_mode
	err = RunConfig([]string{"set", "agenda_mode", "gantt", "--simple"})
	if err != nil {
		t.Fatalf("RunConfig set agenda_mode gantt failed: %v", err)
	}

	updated, err = config.LoadConfig()
	if err != nil || updated.AgendaMode != "gantt" {
		t.Errorf("expected updated agenda_mode to be 'gantt', got '%s'", updated.AgendaMode)
	}

	// 4. Test set invalid agenda_mode
	err = RunConfig([]string{"set", "agenda_mode", "invalid_mode"})
	if err == nil {
		t.Errorf("expected error when setting invalid agenda_mode, got nil")
	}

	// 5. Test set invalid icon preset
	err = RunConfig([]string{"set", "icons", "invalid_theme"})
	if err == nil {
		t.Errorf("expected error when setting invalid icon theme, got nil")
	}

	// 6. Test list
	err = RunConfig([]string{"list", "--simple"})
	if err != nil {
		t.Fatalf("RunConfig list failed: %v", err)
	}

	// 7. Test JSON output
	err = RunConfig([]string{"list", "--json"})
	if err != nil {
		t.Fatalf("RunConfig list --json failed: %v", err)
	}
	_ = configPath
}
