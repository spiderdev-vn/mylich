package cache

import (
	"path/filepath"
	"testing"
	"time"
)

func TestCache_DatabaseAndEvents(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_cache.db")

	db, err := OpenDatabase(dbPath)
	if err != nil {
		t.Fatalf("OpenDatabase failed: %v", err)
	}
	defer db.Close()

	// 1. Create local event
	ev := LocalEvent{
		ID:          "ev-1",
		CalendarID:  "cal-1",
		Title:       "Offline Team Meeting",
		Description: "Discuss sprint roadmap",
		StartAt:     "2026-08-18T10:00:00Z",
		EndAt:       "2026-08-18T11:00:00Z",
		Timezone:    "UTC",
		Location:    "Room A",
		SyncState:   SyncStatePendingCreate,
	}

	if err := UpsertEvent(db, ev); err != nil {
		t.Fatalf("UpsertEvent failed: %v", err)
	}

	// 2. Fetch event
	got, err := GetEvent(db, "ev-1")
	if err != nil || got == nil {
		t.Fatalf("GetEvent failed: %v", err)
	}
	if got.Title != "Offline Team Meeting" || got.SyncState != SyncStatePendingCreate {
		t.Errorf("unexpected event: %+v", got)
	}

	// 3. Search events
	results, err := SearchEvents(db, "roadmap")
	if err != nil || len(results) != 1 {
		t.Fatalf("SearchEvents failed, expected 1 result, got %d (err: %v)", len(results), err)
	}

	// 4. Range query
	inRange, err := GetEventsInRange(db, "2026-08-18T00:00:00Z", "2026-08-18T23:59:59Z", "")
	if err != nil || len(inRange) != 1 {
		t.Fatalf("GetEventsInRange failed, expected 1 result, got %d", len(inRange))
	}

	// 5. Update sync state
	if err := UpdateEventSyncState(db, "ev-1", SyncStateSynced); err != nil {
		t.Fatalf("UpdateEventSyncState failed: %v", err)
	}
	got, _ = GetEvent(db, "ev-1")
	if got.SyncState != SyncStateSynced {
		t.Errorf("expected state synced, got %s", got.SyncState)
	}

	// 6. Mark pending delete
	if err := MarkEventPendingDelete(db, "ev-1"); err != nil {
		t.Fatalf("MarkEventPendingDelete failed: %v", err)
	}
	deletedCheck, _ := GetEvent(db, "ev-1")
	if deletedCheck != nil {
		t.Errorf("expected GetEvent to return nil for pending_delete, got %+v", deletedCheck)
	}
}

func TestCache_SyncQueueAndMeta(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_queue.db")

	db, err := OpenDatabase(dbPath)
	if err != nil {
		t.Fatalf("OpenDatabase failed: %v", err)
	}
	defer db.Close()

	// 1. Enqueue job
	job, err := EnqueueSyncJob(db, "event", "ev-1", SyncOpCreate, `{"title":"Hello"}`)
	if err != nil {
		t.Fatalf("EnqueueSyncJob failed: %v", err)
	}

	count, _ := GetPendingJobCount(db)
	if count != 1 {
		t.Errorf("expected 1 job, got %d", count)
	}

	// 2. Fetch pending jobs
	jobs, err := GetPendingJobs(db, 10)
	if err != nil || len(jobs) != 1 {
		t.Fatalf("GetPendingJobs failed: %v", err)
	}
	if jobs[0].ID != job.ID || jobs[0].Operation != SyncOpCreate {
		t.Errorf("unexpected job: %+v", jobs[0])
	}

	// 3. Record failure and test backoff
	if err := RecordJobFailure(db, job.ID, "network error", 0); err != nil {
		t.Fatalf("RecordJobFailure failed: %v", err)
	}
	jobsAfterFail, _ := GetPendingJobs(db, 10)
	// Because next_attempt_at is in the future (+2s), it should not be returned right now
	if len(jobsAfterFail) != 0 {
		t.Errorf("expected 0 pending jobs immediately after backoff delay, got %d", len(jobsAfterFail))
	}

	// 4. Delete job
	if err := DeleteJob(db, job.ID); err != nil {
		t.Fatalf("DeleteJob failed: %v", err)
	}
	count, _ = GetPendingJobCount(db)
	if count != 0 {
		t.Errorf("expected 0 jobs, got %d", count)
	}

	// 5. Test Metadata and Cursor
	if err := SetLastSyncCursor(db, "cursor-2026-08-18"); err != nil {
		t.Fatalf("SetLastSyncCursor failed: %v", err)
	}
	cursor, _ := GetLastSyncCursor(db)
	if cursor != "cursor-2026-08-18" {
		t.Errorf("expected cursor 'cursor-2026-08-18', got '%s'", cursor)
	}

	syncTime := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	_ = SetLastSyncTime(db, syncTime)
	gotTime, _ := GetLastSyncTime(db)
	if gotTime == nil || !gotTime.Equal(syncTime) {
		t.Errorf("expected sync time %v, got %v", syncTime, gotTime)
	}
}
