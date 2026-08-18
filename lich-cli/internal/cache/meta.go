package cache

import (
	"database/sql"
	"time"
)

const (
	MetaKeyLastCursor   = "last_sync_cursor"
	MetaKeyLastSyncTime = "last_sync_time"
)

func GetSyncMeta(db *sql.DB, key string) (string, error) {
	var value string
	query := `SELECT value FROM sync_meta WHERE key = ?`
	err := db.QueryRow(query, key).Scan(&value)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", err
	}
	return value, nil
}

func SetSyncMeta(db *sql.DB, key, value string) error {
	query := `
	INSERT INTO sync_meta (key, value)
	VALUES (?, ?)
	ON CONFLICT(key) DO UPDATE SET value = excluded.value
	`
	_, err := db.Exec(query, key, value)
	return err
}

func GetLastSyncCursor(db *sql.DB) (string, error) {
	return GetSyncMeta(db, MetaKeyLastCursor)
}

func SetLastSyncCursor(db *sql.DB, cursor string) error {
	return SetSyncMeta(db, MetaKeyLastCursor, cursor)
}

func GetLastSyncTime(db *sql.DB) (*time.Time, error) {
	val, err := GetSyncMeta(db, MetaKeyLastSyncTime)
	if err != nil || val == "" {
		return nil, err
	}
	t, err := time.Parse(time.RFC3339, val)
	if err != nil {
		return nil, nil
	}
	return &t, nil
}

func SetLastSyncTime(db *sql.DB, t time.Time) error {
	return SetSyncMeta(db, MetaKeyLastSyncTime, t.UTC().Format(time.RFC3339))
}
