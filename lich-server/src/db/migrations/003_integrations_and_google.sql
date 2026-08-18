-- Migration 003: Google Calendar and External Integrations Architecture

CREATE TABLE IF NOT EXISTS integrations (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    provider TEXT NOT NULL,
    status TEXT NOT NULL CHECK(status IN ('connected', 'disconnected', 'error', 'needs_reauth')),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_integrations_user_provider ON integrations(user_id, provider);

CREATE TABLE IF NOT EXISTS integration_credentials (
    integration_id TEXT PRIMARY KEY,
    access_token TEXT NOT NULL,
    refresh_token TEXT,
    token_type TEXT NOT NULL DEFAULT 'Bearer',
    expires_at TEXT,
    scope TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    FOREIGN KEY (integration_id) REFERENCES integrations(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS calendar_integrations (
    id TEXT PRIMARY KEY,
    calendar_id TEXT NOT NULL,
    integration_id TEXT NOT NULL,
    external_calendar_id TEXT NOT NULL,
    sync_direction TEXT NOT NULL DEFAULT 'bidirectional' CHECK(sync_direction IN ('push', 'pull', 'bidirectional')),
    enabled INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    FOREIGN KEY (calendar_id) REFERENCES calendars(id) ON DELETE CASCADE,
    FOREIGN KEY (integration_id) REFERENCES integrations(id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_calendar_integrations_cal ON calendar_integrations(calendar_id, integration_id);

CREATE TABLE IF NOT EXISTS event_integrations (
    id TEXT PRIMARY KEY,
    event_id TEXT NOT NULL,
    integration_id TEXT NOT NULL,
    external_id TEXT NOT NULL,
    external_updated_at TEXT,
    etag TEXT,
    metadata TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    FOREIGN KEY (event_id) REFERENCES events(id) ON DELETE CASCADE,
    FOREIGN KEY (integration_id) REFERENCES integrations(id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_event_integrations_event ON event_integrations(event_id, integration_id);
CREATE INDEX IF NOT EXISTS idx_event_integrations_ext ON event_integrations(integration_id, external_id);

CREATE TABLE IF NOT EXISTS integration_sync_state (
    id TEXT PRIMARY KEY,
    integration_id TEXT NOT NULL,
    resource TEXT NOT NULL,
    cursor TEXT NOT NULL,
    last_synced_at TEXT NOT NULL,
    FOREIGN KEY (integration_id) REFERENCES integrations(id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_sync_state_resource ON integration_sync_state(integration_id, resource);

CREATE TABLE IF NOT EXISTS conflicts (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    event_id TEXT,
    integration_id TEXT NOT NULL,
    local_snapshot TEXT NOT NULL,
    remote_snapshot TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'unresolved' CHECK(status IN ('unresolved', 'resolved_local', 'resolved_remote')),
    detected_at TEXT NOT NULL,
    resolved_at TEXT,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (integration_id) REFERENCES integrations(id) ON DELETE CASCADE
);
