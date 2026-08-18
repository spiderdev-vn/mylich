package cli

import (
	"strings"
	"testing"
	"time"

	"lich-cli/internal/cache"
	"lich-cli/internal/config"
)

func TestEdit_Commands(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("LOCALAPPDATA", tempDir)
	t.Setenv("APPDATA", tempDir)
	t.Setenv("XDG_DATA_HOME", tempDir)
	t.Setenv("XDG_CACHE_HOME", tempDir)
	t.Setenv("XDG_CONFIG_HOME", tempDir)

	_ = config.SaveConfig(&config.Config{
		ServerURL: "http://127.0.0.1:3000",
		Token:     "",
		Icons:     "unicode",
	})

	cachePath, err := cache.GetCachePath()
	if err != nil {
		t.Fatalf("GetCachePath failed: %v", err)
	}

	db, err := cache.OpenDatabase(cachePath)
	if err != nil {
		t.Fatalf("OpenDatabase failed: %v", err)
	}

	eventID := "edit-test-event-123"
	now := time.Now().UTC()
	startRFC := now.Format(time.RFC3339)
	endRFC := now.Add(1 * time.Hour).Format(time.RFC3339)

	_ = cache.UpsertEvent(db, cache.LocalEvent{
		ID:          eventID,
		CalendarID:  "default",
		Title:       "Ban đầu",
		Description: "Mô tả cũ",
		StartAt:     startRFC,
		EndAt:       endRFC,
		Timezone:    "UTC",
		Location:    "Phòng họp A",
		SyncState:   cache.SyncStateSynced,
	})
	db.Close()

	// 1. Test edit without args
	err = RunEdit([]string{})
	if err == nil {
		t.Errorf("expected error when running edit without args, got nil")
	}

	// 2. Test edit non-existent event
	err = RunEdit([]string{"non-existent-id", "--title", "Test"})
	if err == nil {
		t.Errorf("expected error when editing non-existent event, got nil")
	}

	// 3. Test edit with flags (change date to 2026-12-25, verify start and end date are both same day: 2026-12-25)
	err = RunEdit([]string{
		eventID,
		"--title", "Đã đổi tiêu đề",
		"--date", "25/12/2026",
		"--location", "Phòng họp VIP",
		"--at", "15:00",
		"--to", "16:30",
		"--simple",
	})
	if err != nil {
		t.Fatalf("RunEdit with flags failed: %v", err)
	}

	// Verify update in database
	db, _ = cache.OpenDatabase(cachePath)
	defer db.Close()

	updated, err := cache.GetEvent(db, eventID)
	if err != nil || updated == nil {
		t.Fatalf("GetEvent failed: %v", err)
	}

	if updated.Title != "Đã đổi tiêu đề" {
		t.Errorf("expected updated title 'Đã đổi tiêu đề', got '%s'", updated.Title)
	}
	if updated.Location != "Phòng họp VIP" {
		t.Errorf("expected updated location 'Phòng họp VIP', got '%s'", updated.Location)
	}
	if !strings.HasPrefix(updated.StartAt, "2026-12-25") {
		t.Errorf("expected start date 2026-12-25, got %s", updated.StartAt)
	}
	if !strings.HasPrefix(updated.EndAt, "2026-12-25") {
		t.Errorf("expected end date to default to SAME DAY 2026-12-25, got %s", updated.EndAt)
	}
	if updated.SyncState != cache.SyncStatePendingUpdate {
		t.Errorf("expected SyncStatePendingUpdate, got '%s'", updated.SyncState)
	}

	// Verify sync job enqueued
	jobs, err := cache.GetPendingJobs(db, 10)
	if err != nil || len(jobs) == 0 {
		t.Fatalf("expected pending sync job, found %d", len(jobs))
	}
	if jobs[0].Operation != cache.SyncOpUpdate || jobs[0].EntityID != eventID {
		t.Errorf("unexpected sync job: %+v", jobs[0])
	}
}
