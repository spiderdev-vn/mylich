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

func TestCLI_ParseFlexibleTime(t *testing.T) {
	tests := []struct {
		input    string
		wantHour int
		wantMin  int
		wantErr  bool
	}{
		{"10:00", 10, 0, false},
		{"22:33", 22, 33, false},
		{"00:00", 0, 0, false},
		{"23:59", 23, 59, false},
		{"10am", 10, 0, false},
		{"10:30am", 10, 30, false},
		{"11:30pm", 23, 30, false},
		{"12am", 0, 0, false},
		{"12pm", 12, 0, false},
		{"3am", 3, 0, false},
		{"3:00 am", 3, 0, false},
		{"3pm", 15, 0, false},
		{"25:00", 0, 0, true},
		{"13pm", 0, 0, true},
		{"invalid", 0, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			h, m, err := parseFlexibleTime(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseFlexibleTime(%q) err = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if !tt.wantErr {
				if h != tt.wantHour || m != tt.wantMin {
					t.Errorf("parseFlexibleTime(%q) = (%d, %d), want (%d, %d)", tt.input, h, m, tt.wantHour, tt.wantMin)
				}
			}
		})
	}
}

func TestCLI_ParseFlexibleDate(t *testing.T) {
	loc := time.UTC
	now := time.Now().In(loc)

	// Test today
	d1, err := parseFlexibleDate("today", loc)
	if err != nil || d1.Day() != now.Day() {
		t.Errorf("parseFlexibleDate('today') failed: %v", err)
	}

	// Test tomorrow
	d2, err := parseFlexibleDate("tomorrow", loc)
	if err != nil || d2.Day() != now.AddDate(0, 0, 1).Day() {
		t.Errorf("parseFlexibleDate('tomorrow') failed: %v", err)
	}

	// Test explicit date formats
	testCases := []struct {
		input       string
		expectedDay int
		expectedMon time.Month
		expectedYr  int
		expectErr   bool
	}{
		{"2026-12-25", 25, time.December, 2026, false},
		{"18-08-26", 18, time.August, 2026, false},
		{"5-8-26", 5, time.August, 2026, false},
		{"18-08-2026", 18, time.August, 2026, false},
		{"5-8-2026", 5, time.August, 2026, false},
		{"18/08/2026", 18, time.August, 2026, false},
		{"5/8/2026", 5, time.August, 2026, false},
		{"18/08/26", 18, time.August, 2026, false},
		{"5/8/26", 5, time.August, 2026, false},
		{"18/08", 18, time.August, now.Year(), false},
		{"5/8", 5, time.August, now.Year(), false},
		{"18-08", 18, time.August, now.Year(), false},
		{"5-8", 5, time.August, now.Year(), false},
		{"31/02/2026", 0, 0, 0, true}, // Invalid date
		{"invalid-date", 0, 0, 0, true},
	}

	for _, tc := range testCases {
		d, err := parseFlexibleDate(tc.input, loc)
		if tc.expectErr {
			if err == nil {
				t.Errorf("parseFlexibleDate('%s') expected error, got nil", tc.input)
			}
		} else {
			if err != nil {
				t.Errorf("parseFlexibleDate('%s') failed: %v", tc.input, err)
			} else if d.Day() != tc.expectedDay || d.Month() != tc.expectedMon || d.Year() != tc.expectedYr {
				t.Errorf("parseFlexibleDate('%s') = %v, expected %d/%d/%d", tc.input, d, tc.expectedDay, tc.expectedMon, tc.expectedYr)
			}
		}
	}
}

func TestCLI_ParseFlexibleTimeRange(t *testing.T) {
	loc := time.UTC

	// 1. Same-day with --at and --to
	start, end, overnight, err := parseFlexibleTimeRange("2026-08-18", "10:00", "22:33", "", false, loc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if overnight {
		t.Errorf("expected overnight == false for 10:00 -> 22:33")
	}
	if start.Hour() != 10 || start.Minute() != 0 || end.Hour() != 22 || end.Minute() != 33 {
		t.Errorf("unexpected range: %v -> %v", start, end)
	}
	if start.Day() != end.Day() {
		t.Errorf("expected start and end on same day")
	}

	// 2. Overnight cross-midnight: 23:30 -> 03:00
	start, end, overnight, err = parseFlexibleTimeRange("2026-08-18", "23:30", "03:00", "", false, loc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !overnight {
		t.Errorf("expected overnight == true for 23:30 -> 03:00")
	}
	if start.Day() != 18 || end.Day() != 19 {
		t.Errorf("expected end date to be 19th (+1 day), got start=%d, end=%d", start.Day(), end.Day())
	}
	if end.Hour() != 3 || end.Minute() != 0 {
		t.Errorf("expected end hour 03:00, got %02d:%02d", end.Hour(), end.Minute())
	}

	// 3. Overnight cross-midnight with 12h: 11:30pm -> 3:00am
	start, end, overnight, err = parseFlexibleTimeRange("2026-08-18", "11:30pm", "3:00am", "", false, loc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !overnight {
		t.Errorf("expected overnight == true for 11:30pm -> 3:00am")
	}
	if start.Hour() != 23 || start.Minute() != 30 || end.Hour() != 3 || end.Minute() != 0 {
		t.Errorf("unexpected time: %v -> %v", start, end)
	}

	// 4. Conflict: both --to and --duration passed
	_, _, _, err = parseFlexibleTimeRange("2026-08-18", "10:00", "12:00", "2h", true, loc)
	if err == nil {
		t.Fatalf("expected error when both --to and --duration are provided, got nil")
	}

	// 5. Overnight with explicit same-day end date (default same day): 2026-08-20 22:00 -> 2026-08-20 03:00
	start, end, overnight, err = parseFlexibleTimeRangeWithEndDate("2026-08-20", "2026-08-20", "22:00", "03:00", "", false, loc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !overnight {
		t.Errorf("expected overnight == true for 22:00 -> 03:00 with same end date")
	}
	if start.Day() != 20 || end.Day() != 21 {
		t.Errorf("expected start day 20 and end day 21, got %d and %d", start.Day(), end.Day())
	}
	if !end.After(start) {
		t.Errorf("end time (%v) must be after start time (%v)", end, start)
	}
}

func TestCLI_NukeDatabaseForce(t *testing.T) {
	err := RunNuke([]string{"--force", "--simple"})
	if err != nil {
		t.Fatalf("RunNuke --force failed: %v", err)
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
