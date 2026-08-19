package cache

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

func UpsertEvent(db *sql.DB, event LocalEvent) error {
	now := time.Now().UTC().Format(time.RFC3339)
	if event.CreatedAt == "" {
		event.CreatedAt = now
	}
	if event.UpdatedAt == "" {
		event.UpdatedAt = now
	}
	if event.SyncState == "" {
		event.SyncState = SyncStateSynced
	}

	query := `
	INSERT INTO local_events (id, calendar_id, title, description, start_at, end_at, timezone, location, created_at, updated_at, sync_state)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(id) DO UPDATE SET
		calendar_id = excluded.calendar_id,
		title = excluded.title,
		description = excluded.description,
		start_at = excluded.start_at,
		end_at = excluded.end_at,
		timezone = excluded.timezone,
		location = excluded.location,
		updated_at = excluded.updated_at,
		sync_state = excluded.sync_state
	`

	_, err := db.Exec(
		query,
		event.ID,
		event.CalendarID,
		event.Title,
		event.Description,
		event.StartAt,
		event.EndAt,
		event.Timezone,
		event.Location,
		event.CreatedAt,
		event.UpdatedAt,
		string(event.SyncState),
	)
	if err != nil {
		return fmt.Errorf("failed to upsert local event: %w", err)
	}

	return nil
}

func GetEvent(db *sql.DB, id string) (*LocalEvent, error) {
	query := `
	SELECT id, calendar_id, title, description, start_at, end_at, timezone, location, created_at, updated_at, sync_state
	FROM local_events
	WHERE id = ? AND sync_state != 'pending_delete'
	`

	row := db.QueryRow(query, id)

	var e LocalEvent
	var desc, loc sql.NullString
	var syncStateStr string

	err := row.Scan(
		&e.ID,
		&e.CalendarID,
		&e.Title,
		&desc,
		&e.StartAt,
		&e.EndAt,
		&e.Timezone,
		&loc,
		&e.CreatedAt,
		&e.UpdatedAt,
		&syncStateStr,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get local event: %w", err)
	}

	if desc.Valid {
		e.Description = desc.String
	}
	if loc.Valid {
		e.Location = loc.String
	}
	e.SyncState = SyncState(syncStateStr)

	return &e, nil
}

// FindEventsByPrefix searches for events whose ID starts with the given prefix.
func FindEventsByPrefix(db *sql.DB, prefix string) ([]LocalEvent, error) {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		return nil, nil
	}

	query := `
	SELECT id, calendar_id, title, description, start_at, end_at, timezone, location, created_at, updated_at, sync_state
	FROM local_events
	WHERE id LIKE ? AND sync_state != 'pending_delete'
	ORDER BY start_at DESC
	`
	rows, err := db.Query(query, prefix+"%")
	if err != nil {
		return nil, fmt.Errorf("failed to search events by prefix: %w", err)
	}
	defer rows.Close()

	var events []LocalEvent
	for rows.Next() {
		var e LocalEvent
		var desc, loc sql.NullString
		var syncStateStr string

		err := rows.Scan(
			&e.ID,
			&e.CalendarID,
			&e.Title,
			&desc,
			&e.StartAt,
			&e.EndAt,
			&e.Timezone,
			&loc,
			&e.CreatedAt,
			&e.UpdatedAt,
			&syncStateStr,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}

		if desc.Valid {
			e.Description = desc.String
		}
		if loc.Valid {
			e.Location = loc.String
		}
		e.SyncState = SyncState(syncStateStr)
		events = append(events, e)
	}

	return events, nil
}

// ResolveEventByPrefix finds a single event by exact ID or unique short ID prefix.
// If multiple events match the prefix, it returns an error with the list of conflicting events.
func ResolveEventByPrefix(db *sql.DB, prefix string) (*LocalEvent, error) {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		return nil, fmt.Errorf("ID sự kiện không được để trống")
	}

	// 1. Check exact match first
	if exact, err := GetEvent(db, prefix); err == nil && exact != nil {
		return exact, nil
	}

	// 2. Prefix search
	matches, err := FindEventsByPrefix(db, prefix)
	if err != nil {
		return nil, err
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf("không tìm thấy sự kiện nào khớp với ID '%s'", prefix)
	}

	if len(matches) == 1 {
		return &matches[0], nil
	}

	// Conflict / Ambiguous short ID matches
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Xung đột tiền tố: Tìm thấy %d sự kiện trùng khớp với '%s':\n", len(matches), prefix))
	loc := time.Now().Location()
	for _, m := range matches {
		tStart, _ := time.Parse(time.RFC3339, m.StartAt)
		timeStr := tStart.In(loc).Format("02/01 15:04")
		b.WriteString(fmt.Sprintf("  • [%s] %s (%s)\n", m.ID, m.Title, timeStr))
	}
	b.WriteString("\nVui lòng nhập thêm ký tự để phân biệt chính xác.")
	return nil, errors.New(b.String())
}

func GetEventsInRange(db *sql.DB, from, to string, calendarID string) ([]LocalEvent, error) {
	conditions := []string{"sync_state != 'pending_delete'"}
	var args []any

	if calendarID != "" {
		conditions = append(conditions, "calendar_id = ?")
		args = append(args, calendarID)
	}

	if from != "" {
		conditions = append(conditions, "end_at > ?")
		args = append(args, from)
	}

	if to != "" {
		conditions = append(conditions, "start_at < ?")
		args = append(args, to)
	}

	query := fmt.Sprintf(`
	SELECT id, calendar_id, title, description, start_at, end_at, timezone, location, created_at, updated_at, sync_state
	FROM local_events
	WHERE %s
	ORDER BY start_at ASC
	`, strings.Join(conditions, " AND "))

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query local events: %w", err)
	}
	defer rows.Close()

	var events []LocalEvent
	for rows.Next() {
		var e LocalEvent
		var desc, loc sql.NullString
		var syncStateStr string

		if err := rows.Scan(
			&e.ID,
			&e.CalendarID,
			&e.Title,
			&desc,
			&e.StartAt,
			&e.EndAt,
			&e.Timezone,
			&loc,
			&e.CreatedAt,
			&e.UpdatedAt,
			&syncStateStr,
		); err != nil {
			return nil, fmt.Errorf("failed to scan local event: %w", err)
		}

		if desc.Valid {
			e.Description = desc.String
		}
		if loc.Valid {
			e.Location = loc.String
		}
		e.SyncState = SyncState(syncStateStr)
		events = append(events, e)
	}

	return events, nil
}

func SearchEvents(db *sql.DB, keyword string) ([]LocalEvent, error) {
	pattern := "%" + strings.ToLower(keyword) + "%"
	query := `
	SELECT id, calendar_id, title, description, start_at, end_at, timezone, location, created_at, updated_at, sync_state
	FROM local_events
	WHERE sync_state != 'pending_delete'
	  AND (LOWER(title) LIKE ? OR LOWER(description) LIKE ? OR LOWER(location) LIKE ?)
	ORDER BY start_at ASC
	LIMIT 50
	`

	rows, err := db.Query(query, pattern, pattern, pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to search local events: %w", err)
	}
	defer rows.Close()

	var events []LocalEvent
	for rows.Next() {
		var e LocalEvent
		var desc, loc sql.NullString
		var syncStateStr string

		if err := rows.Scan(
			&e.ID,
			&e.CalendarID,
			&e.Title,
			&desc,
			&e.StartAt,
			&e.EndAt,
			&e.Timezone,
			&loc,
			&e.CreatedAt,
			&e.UpdatedAt,
			&syncStateStr,
		); err != nil {
			return nil, fmt.Errorf("failed to scan local event: %w", err)
		}

		if desc.Valid {
			e.Description = desc.String
		}
		if loc.Valid {
			e.Location = loc.String
		}
		e.SyncState = SyncState(syncStateStr)
		events = append(events, e)
	}

	return events, nil
}

func MarkEventPendingDelete(db *sql.DB, id string) error {
	query := `UPDATE local_events SET sync_state = 'pending_delete', updated_at = ? WHERE id = ?`
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := db.Exec(query, now, id)
	return err
}

func DeleteEventPermanently(db *sql.DB, id string) error {
	query := `DELETE FROM local_events WHERE id = ?`
	_, err := db.Exec(query, id)
	return err
}

func UpdateEventSyncState(db *sql.DB, id string, state SyncState) error {
	query := `UPDATE local_events SET sync_state = ? WHERE id = ?`
	_, err := db.Exec(query, string(state), id)
	return err
}
