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
}

func TestCLI_FormatTimeRange_EdgeCases(t *testing.T) {
	loc := time.UTC

	// Same day
	t1 := formatTimeRange("2026-08-18T10:00:00Z", "2026-08-18T11:30:00Z", loc)
	if t1 != "10:00 - 11:30" {
		t.Errorf("expected '10:00 - 11:30', got '%s'", t1)
	}

	// Overnight cross midnight (20/08 22:00 -> 21/08 03:00)
	t2 := formatTimeRange("2026-08-20T22:00:00Z", "2026-08-21T03:00:00Z", loc)
	if t2 != "22:00 20/08 - 03:00 21/08" {
		t.Errorf("expected '22:00 20/08 - 03:00 21/08', got '%s'", t2)
	}

	// Multi-day spanning 3 days (20/08 08:00 -> 23/08 17:00)
	t3 := formatTimeRange("2026-08-20T08:00:00Z", "2026-08-23T17:00:00Z", loc)
	if t3 != "08:00 20/08 - 17:00 23/08" {
		t.Errorf("expected '08:00 20/08 - 17:00 23/08', got '%s'", t3)
	}

	// Invalid ISO format fallback
	t4 := formatTimeRange("invalid-start", "invalid-end", loc)
	if t4 != "invalid-start - invalid-end" {
		t.Errorf("expected fallback 'invalid-start - invalid-end', got '%s'", t4)
	}
}

func TestCLI_ParseFlexibleTimeRange_EdgeCases(t *testing.T) {
	loc := time.UTC

	// 1. Cross Month Boundary: 31/08/2026 23:00 -> 01/09/2026 04:00
	start, end, overnight, err := parseFlexibleTimeRangeWithEndDate("31/08/2026", "01/09/2026", "23:00", "04:00", "", false, loc)
	if err != nil {
		t.Fatalf("cross-month error: %v", err)
	}
	if !overnight {
		t.Errorf("expected overnight == true for cross-month")
	}
	if start.Month() != time.August || start.Day() != 31 || end.Month() != time.September || end.Day() != 1 {
		t.Errorf("unexpected cross-month dates: %v -> %v", start, end)
	}

	// 2. Cross Year Boundary: 31/12/2026 22:00 -> 01/01/2027 02:00
	start, end, overnight, err = parseFlexibleTimeRangeWithEndDate("31/12/2026", "01/01/2027", "22:00", "02:00", "", false, loc)
	if err != nil {
		t.Fatalf("cross-year error: %v", err)
	}
	if !overnight {
		t.Errorf("expected overnight == true for cross-year")
	}
	if start.Year() != 2026 || end.Year() != 2027 {
		t.Errorf("unexpected cross-year: %v -> %v", start, end)
	}

	// 3. Multi-day duration (48h spanning 2 full days)
	start, end, overnight, err = parseFlexibleTimeRange("2026-08-18", "10:00", "", "48h", true, loc)
	if err != nil {
		t.Fatalf("duration 48h error: %v", err)
	}
	if !overnight {
		t.Errorf("expected overnight == true for 48h duration")
	}
	if end.Day() != 20 || end.Hour() != 10 {
		t.Errorf("expected end date 2026-08-20 10:00, got %v", end)
	}

	// 4. 12-hour AM/PM edge cases: 12:00am (00:00) and 12:00pm (12:00)
	start, end, overnight, err = parseFlexibleTimeRange("2026-08-18", "12:00am", "12:00pm", "", false, loc)
	if err != nil {
		t.Fatalf("12am->12pm error: %v", err)
	}
	if overnight {
		t.Errorf("expected overnight == false for 12am -> 12pm on same day")
	}
	if start.Hour() != 0 || end.Hour() != 12 {
		t.Errorf("expected start=00:00 end=12:00, got start=%d end=%d", start.Hour(), end.Hour())
	}

	// 5. 12-hour PM to AM overnight: 12:00pm -> 12:00am (12:00 -> 00:00 next day)
	start, end, overnight, err = parseFlexibleTimeRange("2026-08-18", "12:00pm", "12:00am", "", false, loc)
	if err != nil {
		t.Fatalf("12pm->12am error: %v", err)
	}
	if !overnight {
		t.Errorf("expected overnight == true for 12pm -> 12am")
	}
	if start.Hour() != 12 || end.Hour() != 0 || end.Day() != 19 {
		t.Errorf("expected start=18th 12:00 end=19th 00:00, got start=%v end=%v", start, end)
	}

	// 6. Invalid: end date strictly before start date
	_, _, _, err = parseFlexibleTimeRangeWithEndDate("2026-08-20", "2026-08-19", "10:00", "11:00", "", false, loc)
	if err == nil {
		t.Fatalf("expected error when end date is before start date, got nil")
	}

	// 7. Same start and end time on same day (advances to 24h next day)
	start, end, overnight, err = parseFlexibleTimeRange("2026-08-18", "10:00", "10:00", "", false, loc)
	if err != nil {
		t.Fatalf("same time error: %v", err)
	}
	if !overnight {
		t.Errorf("expected overnight == true for 10:00 -> 10:00 same time")
	}
	if end.Day() != 19 || end.Hour() != 10 {
		t.Errorf("expected end date next day 10:00, got %v", end)
	}
}

func TestCLI_NukeDatabaseForce(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("LOCALAPPDATA", tempDir)
	t.Setenv("XDG_CACHE_HOME", tempDir)
	t.Setenv("HOME", tempDir)

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
