package cache

import "time"

type SyncState string

const (
	SyncStateSynced        SyncState = "synced"
	SyncStatePendingCreate SyncState = "pending_create"
	SyncStatePendingUpdate SyncState = "pending_update"
	SyncStatePendingDelete SyncState = "pending_delete"
	SyncStateFailed        SyncState = "failed"
)

type LocalEvent struct {
	ID          string    `json:"id"`
	CalendarID  string    `json:"calendar_id"`
	Title       string    `json:"title"`
	Description string    `json:"description,omitempty"`
	StartAt     string    `json:"start_at"`
	EndAt       string    `json:"end_at"`
	Timezone    string    `json:"timezone"`
	Location    string    `json:"location,omitempty"`
	CreatedAt   string    `json:"created_at"`
	UpdatedAt   string    `json:"updated_at"`
	SyncState   SyncState `json:"sync_state"`
}

type LocalCalendar struct {
	ID          string `json:"id"`
	UserID      string `json:"user_id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Timezone    string `json:"timezone"`
	IsDefault   bool   `json:"is_default"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type SyncOperation string

const (
	SyncOpCreate SyncOperation = "CREATE"
	SyncOpUpdate SyncOperation = "UPDATE"
	SyncOpDelete SyncOperation = "DELETE"
)

type SyncJob struct {
	ID            string        `json:"id"`
	EntityType    string        `json:"entity_type"`
	EntityID      string        `json:"entity_id"`
	Operation     SyncOperation `json:"operation"`
	Attempts      int           `json:"attempts"`
	NextAttemptAt time.Time     `json:"next_attempt_at"`
	LastError     string        `json:"last_error,omitempty"`
	Payload       string        `json:"payload,omitempty"`
	CreatedAt     time.Time     `json:"created_at"`
}
