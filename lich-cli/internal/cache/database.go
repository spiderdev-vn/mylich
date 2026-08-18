package cache

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

func GetCachePath() (string, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get user cache/home directory: %w", err)
		}
		cacheDir = filepath.Join(homeDir, ".cache")
	}

	dir := filepath.Join(cacheDir, "lich")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("failed to create cache directory '%s': %w", dir, err)
	}

	return filepath.Join(dir, "cache.db"), nil
}

func OpenDatabase(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open local sqlite database: %w", err)
	}

	// PRAGMA configuration for speed, concurrency, and safety
	pragmas := []string{
		"PRAGMA journal_mode = WAL;",
		"PRAGMA busy_timeout = 5000;",
		"PRAGMA synchronous = NORMAL;",
		"PRAGMA foreign_keys = ON;",
	}

	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			db.Close()
			return nil, fmt.Errorf("failed to set pragma '%s': %w", p, err)
		}
	}

	if err := initSchema(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize local database schema: %w", err)
	}

	return db, nil
}

func initSchema(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS local_calendars (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		name TEXT NOT NULL,
		description TEXT,
		timezone TEXT NOT NULL,
		is_default INTEGER NOT NULL DEFAULT 0,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS local_events (
		id TEXT PRIMARY KEY,
		calendar_id TEXT NOT NULL,
		title TEXT NOT NULL,
		description TEXT,
		start_at TEXT NOT NULL,
		end_at TEXT NOT NULL,
		timezone TEXT NOT NULL,
		location TEXT,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL,
		sync_state TEXT NOT NULL DEFAULT 'synced'
	);

	CREATE INDEX IF NOT EXISTS idx_local_events_start_at ON local_events(start_at);
	CREATE INDEX IF NOT EXISTS idx_local_events_end_at ON local_events(end_at);
	CREATE INDEX IF NOT EXISTS idx_local_events_sync_state ON local_events(sync_state);

	CREATE TABLE IF NOT EXISTS sync_jobs (
		id TEXT PRIMARY KEY,
		entity_type TEXT NOT NULL,
		entity_id TEXT NOT NULL,
		operation TEXT NOT NULL,
		attempts INTEGER NOT NULL DEFAULT 0,
		next_attempt_at TEXT NOT NULL,
		last_error TEXT,
		payload TEXT,
		created_at TEXT NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_sync_jobs_next_attempt ON sync_jobs(next_attempt_at);

	CREATE TABLE IF NOT EXISTS sync_meta (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL
	);
	`

	_, err := db.Exec(schema)
	return err
}
