package syncer

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"lich-cli/internal/api"
	"lich-cli/internal/cache"
)

func TestSyncer_PushAndPull(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "syncer_test.db")

	db, err := cache.OpenDatabase(dbPath)
	if err != nil {
		t.Fatalf("OpenDatabase failed: %v", err)
	}
	defer db.Close()

	// Create mock server
	var serverEvents []api.Event
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/events":
			var req api.CreateEventRequest
			_ = json.NewDecoder(r.Body).Decode(&req)
			ev := api.Event{
				ID:        req.ID,
				Title:     req.Title,
				StartAt:   req.StartAt,
				EndAt:     req.EndAt,
				CreatedAt: time.Now().UTC().Format(time.RFC3339),
				UpdatedAt: time.Now().UTC().Format(time.RFC3339),
			}
			serverEvents = append(serverEvents, ev)
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(ev)

		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/sync":
			changes := []api.SyncChangeItem{
				{
					ID:         "sync-change-1",
					EntityType: "event",
					EntityID:   "server-event-99",
					Action:     "create",
					Data: &api.Event{
						ID:        "server-event-99",
						Title:     "Remote Event From Web",
						StartAt:   "2026-08-18T14:00:00Z",
						EndAt:     "2026-08-18T15:00:00Z",
						Timezone:  "UTC",
						CreatedAt: time.Now().UTC().Format(time.RFC3339),
						UpdatedAt: time.Now().UTC().Format(time.RFC3339),
					},
					CreatedAt: time.Now().UTC().Format(time.RFC3339),
				},
			}
			res := api.SyncResponse{
				Cursor:  "new-cursor-123",
				Changes: changes,
			}
			_ = json.NewEncoder(w).Encode(res)

		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test-token")
	engine := NewSyncEngine(db, client)

	// 1. Create a local pending event
	localEv := cache.LocalEvent{
		ID:        "ev-local-1",
		Title:     "Local Event to Push",
		StartAt:   "2026-08-18T10:00:00Z",
		EndAt:     "2026-08-18T11:00:00Z",
		SyncState: cache.SyncStatePendingCreate,
	}
	_ = cache.UpsertEvent(db, localEv)
	_, _ = cache.EnqueueSyncJob(db, "event", localEv.ID, cache.SyncOpCreate, MarshalPayload(localEv))

	// 2. Test Push
	ctx := context.Background()
	pushed, err := engine.Push(ctx)
	if err != nil {
		t.Fatalf("engine.Push failed: %v", err)
	}
	if pushed != 1 {
		t.Errorf("expected 1 pushed event, got %d", pushed)
	}

	evAfterPush, _ := cache.GetEvent(db, localEv.ID)
	if evAfterPush.SyncState != cache.SyncStateSynced {
		t.Errorf("expected state synced, got %s", evAfterPush.SyncState)
	}

	// 3. Test Pull
	pulled, err := engine.Pull(ctx)
	if err != nil {
		t.Fatalf("engine.Pull failed: %v", err)
	}
	if pulled != 1 {
		t.Errorf("expected 1 pulled event, got %d", pulled)
	}

	remoteEv, err := cache.GetEvent(db, "server-event-99")
	if err != nil || remoteEv == nil {
		t.Fatalf("expected remote event to be saved in local cache, got %v", remoteEv)
	}
	if remoteEv.Title != "Remote Event From Web" {
		t.Errorf("unexpected title: %s", remoteEv.Title)
	}

	cursor, _ := cache.GetLastSyncCursor(db)
	if cursor != "new-cursor-123" {
		t.Errorf("expected cursor 'new-cursor-123', got '%s'", cursor)
	}

	// 4. Test PushWithProgress and PullWithProgress
	var eventsRecorded []ProgressEvent
	callback := func(ev ProgressEvent) {
		eventsRecorded = append(eventsRecorded, ev)
	}

	pushedProg, err := engine.PushWithProgress(ctx, callback)
	if err != nil {
		t.Fatalf("PushWithProgress failed: %v", err)
	}
	if pushedProg != 0 {
		t.Errorf("expected 0 pushed (empty queue), got %d", pushedProg)
	}

	pulledProg, err := engine.PullWithProgress(ctx, callback)
	if err != nil {
		t.Fatalf("PullWithProgress failed: %v", err)
	}
	if pulledProg != 1 {
		t.Errorf("expected 1 pulled, got %d", pulledProg)
	}
}
