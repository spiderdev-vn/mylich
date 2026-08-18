package cli

import (
	"testing"
	"time"
)

func TestCLI_FormatTimeRange(t *testing.T) {
	loc := time.UTC
	startStr := "2026-08-18T10:00:00Z"
	endStr := "2026-08-18T11:30:00Z"

	formatted := formatTimeRange(startStr, endStr, loc)
	expected := "10:00 - 11:30"
	if formatted != expected {
		t.Errorf("expected '%s', got '%s'", expected, formatted)
	}
}

func TestCLI_ExecuteVersionAndHelp(t *testing.T) {
	if code := Execute([]string{"version"}); code != 0 {
		t.Errorf("expected exit code 0 for version, got %d", code)
	}
	if code := Execute([]string{"help"}); code != 0 {
		t.Errorf("expected exit code 0 for help, got %d", code)
	}
	if code := Execute([]string{"unknown-cmd"}); code != 1 {
		t.Errorf("expected exit code 1 for unknown command, got %d", code)
	}
}
