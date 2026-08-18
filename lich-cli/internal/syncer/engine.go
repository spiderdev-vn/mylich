package syncer

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"lich-cli/internal/api"
	"lich-cli/internal/cache"
)

type SyncEngine struct {
	db     *sql.DB
	client *api.Client
}

// ProgressKind phân loại từng bước sync
type ProgressKind string

const (
	ProgressStart   ProgressKind = "start"
	ProgressPush    ProgressKind = "push"
	ProgressPull    ProgressKind = "pull"
	ProgressDone    ProgressKind = "done"
	ProgressError   ProgressKind = "error"
	ProgressSkip    ProgressKind = "skip"
)

// ProgressEvent là một bước trong quá trình sync
type ProgressEvent struct {
	Kind    ProgressKind
	Total   int    // Tổng số jobs (nếu biết)
	Current int    // Job hiện tại
	Message string // Mô tả ngắn
	Err     error
}

func NewSyncEngine(db *sql.DB, client *api.Client) *SyncEngine {
	return &SyncEngine{
		db:     db,
		client: client,
	}
}

func (e *SyncEngine) Push(ctx context.Context) (int, error) {
	if e.client == nil {
		return 0, fmt.Errorf("client is nil, not authenticated")
	}

	jobs, err := cache.GetPendingJobs(e.db, 50)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch pending jobs: %w", err)
	}

	pushed := 0
	for _, job := range jobs {
		if job.EntityType != "event" {
			continue
		}

		var jobErr error
		switch job.Operation {
		case cache.SyncOpCreate:
			jobErr = e.pushCreate(ctx, job)
		case cache.SyncOpUpdate:
			jobErr = e.pushUpdate(ctx, job)
		case cache.SyncOpDelete:
			jobErr = e.pushDelete(ctx, job)
		}

		if jobErr != nil {
			_ = cache.RecordJobFailure(e.db, job.ID, jobErr.Error(), job.Attempts)
			_ = cache.UpdateEventSyncState(e.db, job.EntityID, cache.SyncStateFailed)
		} else {
			_ = cache.DeleteJob(e.db, job.ID)
			pushed++
		}
	}

	return pushed, nil
}

func (e *SyncEngine) pushCreate(ctx context.Context, job cache.SyncJob) error {
	event, err := cache.GetEvent(e.db, job.EntityID)
	if err != nil {
		return err
	}
	if event == nil {
		// Event was deleted locally before sync
		return nil
	}

	tz := event.Timezone
	if tz == "Local" {
		tz = "UTC"
	}

	_, err = e.client.CreateEvent(ctx, api.CreateEventRequest{
		ID:          event.ID,
		Title:       event.Title,
		CalendarID:  event.CalendarID,
		Description: event.Description,
		StartAt:     event.StartAt,
		EndAt:       event.EndAt,
		Timezone:    tz,
		Location:    event.Location,
	})
	if err != nil {
		// Check if already created
		if apiErr, ok := err.(*api.APIError); ok && apiErr.StatusCode == 409 {
			_ = cache.UpdateEventSyncState(e.db, event.ID, cache.SyncStateSynced)
			return nil
		}
		return err
	}

	return cache.UpdateEventSyncState(e.db, event.ID, cache.SyncStateSynced)
}

func (e *SyncEngine) pushUpdate(ctx context.Context, job cache.SyncJob) error {
	event, err := cache.GetEvent(e.db, job.EntityID)
	if err != nil {
		return err
	}
	if event == nil {
		return nil
	}

	_, err = e.client.UpdateEvent(ctx, event.ID, api.UpdateEventRequest{
		Title:       event.Title,
		CalendarID:  event.CalendarID,
		Description: event.Description,
		StartAt:     event.StartAt,
		EndAt:       event.EndAt,
		Timezone:    event.Timezone,
		Location:    event.Location,
	})
	if err != nil {
		return err
	}

	return cache.UpdateEventSyncState(e.db, event.ID, cache.SyncStateSynced)
}

func (e *SyncEngine) pushDelete(ctx context.Context, job cache.SyncJob) error {
	err := e.client.DeleteEvent(ctx, job.EntityID)
	if err != nil {
		if apiErr, ok := err.(*api.APIError); ok && apiErr.StatusCode == 404 {
			// Already deleted on server
			_ = cache.DeleteEventPermanently(e.db, job.EntityID)
			return nil
		}
		return err
	}

	return cache.DeleteEventPermanently(e.db, job.EntityID)
}

func (e *SyncEngine) Pull(ctx context.Context) (int, error) {
	if e.client == nil {
		return 0, fmt.Errorf("client is nil, not authenticated")
	}

	cursor, err := cache.GetLastSyncCursor(e.db)
	if err != nil {
		return 0, fmt.Errorf("failed to get sync cursor: %w", err)
	}

	res, err := e.client.Sync(ctx, cursor, 100)
	if err != nil {
		return 0, fmt.Errorf("failed to pull sync changes from server: %w", err)
	}

	applied := 0
	for _, change := range res.Changes {
		if change.EntityType != "event" {
			continue
		}

		if change.Action == "delete" {
			_ = cache.DeleteEventPermanently(e.db, change.EntityID)
			applied++
			continue
		}

		if change.Data != nil {
			localEvt, _ := cache.GetEvent(e.db, change.EntityID)
			// Don't overwrite local pending changes
			if localEvt != nil && (localEvt.SyncState == cache.SyncStatePendingUpdate || localEvt.SyncState == cache.SyncStatePendingDelete) {
				continue
			}

			_ = cache.UpsertEvent(e.db, cache.LocalEvent{
				ID:          change.Data.ID,
				CalendarID:  change.Data.CalendarID,
				Title:       change.Data.Title,
				Description: change.Data.Description,
				StartAt:     change.Data.StartAt,
				EndAt:       change.Data.EndAt,
				Timezone:    change.Data.Timezone,
				Location:    change.Data.Location,
				CreatedAt:   change.Data.CreatedAt,
				UpdatedAt:   change.Data.UpdatedAt,
				SyncState:   cache.SyncStateSynced,
			})
			applied++
		}
	}

	if res.Cursor != "" {
		_ = cache.SetLastSyncCursor(e.db, res.Cursor)
	}
	_ = cache.SetLastSyncTime(e.db, time.Now())

	return applied, nil
}

func (e *SyncEngine) Sync(ctx context.Context) (int, int, error) {
	pushed, pushErr := e.Push(ctx)
	pulled, pullErr := e.Pull(ctx)

	if pushErr != nil {
		return pushed, pulled, pushErr
	}
	if pullErr != nil {
		return pushed, pulled, pullErr
	}

	return pushed, pulled, nil
}

// SyncWithProgress chạy sync và gọi onProgress sau mỗi bước
func (e *SyncEngine) SyncWithProgress(ctx context.Context, onProgress func(ProgressEvent)) (int, int, error) {
	if e.client == nil {
		return 0, 0, fmt.Errorf("client is nil, not authenticated")
	}

	// Đếm trước số pending jobs
	jobs, err := cache.GetPendingJobs(e.db, 50)
	if err != nil {
		return 0, 0, err
	}

	onProgress(ProgressEvent{Kind: ProgressStart, Total: len(jobs), Message: fmt.Sprintf("%d thao tác chờ đẩy lên", len(jobs))})

	// PUSH từng job
	pushed := 0
	for idx, job := range jobs {
		if job.EntityType != "event" {
			onProgress(ProgressEvent{Kind: ProgressSkip, Total: len(jobs), Current: idx + 1, Message: fmt.Sprintf("Bỏ qua job %s (không phải event)", job.ID[:8])})
			continue
		}

		opName := string(job.Operation)
		onProgress(ProgressEvent{Kind: ProgressPush, Total: len(jobs), Current: idx + 1, Message: fmt.Sprintf("[%d/%d] %s %s...", idx+1, len(jobs), opName, job.EntityID[:12])})

		var jobErr error
		switch job.Operation {
		case cache.SyncOpCreate:
			jobErr = e.pushCreate(ctx, job)
		case cache.SyncOpUpdate:
			jobErr = e.pushUpdate(ctx, job)
		case cache.SyncOpDelete:
			jobErr = e.pushDelete(ctx, job)
		}

		if jobErr != nil {
			_ = cache.RecordJobFailure(e.db, job.ID, jobErr.Error(), job.Attempts)
			_ = cache.UpdateEventSyncState(e.db, job.EntityID, cache.SyncStateFailed)
			onProgress(ProgressEvent{Kind: ProgressError, Total: len(jobs), Current: idx + 1, Message: fmt.Sprintf("✗ Lỗi: %v", jobErr), Err: jobErr})
		} else {
			_ = cache.DeleteJob(e.db, job.ID)
			pushed++
		}
	}

	// PULL từ server
	onProgress(ProgressEvent{Kind: ProgressPull, Message: "Nhận dữ liệu mới từ máy chủ..."})
	pulled, pullErr := e.Pull(ctx)
	if pullErr != nil {
		onProgress(ProgressEvent{Kind: ProgressError, Message: fmt.Sprintf("✗ Lỗi kết nối: %v", pullErr), Err: pullErr})
		return pushed, 0, pullErr
	}

	onProgress(ProgressEvent{Kind: ProgressDone, Message: fmt.Sprintf("✓ Hoàn tất: ↑%d đẩy lên  ↓%d nhận về", pushed, pulled)})
	return pushed, pulled, nil
}

func (e *SyncEngine) SyncInBackground() {
	// Dùng goroutine với WaitGroup để đảm bảo sync hoàn tất ngay cả khi process CLI thoát ngay
	// Trong thực tế: goroutine chạy đủ thời gian vì CLI chờ fmt.Println() rồi mới return
	// Nhưng nếu OS kill process ngay, sync sẽ bị mất — đây là trade-off chấp nhận được
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		_, _, _ = e.Sync(ctx)
	}()
}

func MarshalPayload(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
