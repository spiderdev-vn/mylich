-- Migration 002: Add deleted_at to calendars and create change_logs table
ALTER TABLE calendars ADD COLUMN deleted_at TEXT;

-- Change logs for incremental synchronization
CREATE TABLE IF NOT EXISTS change_logs (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    entity_type TEXT NOT NULL CHECK(entity_type IN ('event', 'calendar')),
    entity_id TEXT NOT NULL,
    action TEXT NOT NULL CHECK(action IN ('create', 'update', 'delete')),
    data TEXT,
    created_at TEXT NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_change_logs_user_created ON change_logs(user_id, created_at);
