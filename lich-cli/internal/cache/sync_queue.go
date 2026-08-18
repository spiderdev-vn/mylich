package cache

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"math"
	"time"
)

func randomID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func EnqueueSyncJob(db *sql.DB, entityType, entityID string, op SyncOperation, payload string) (*SyncJob, error) {
	job := SyncJob{
		ID:            randomID(),
		EntityType:    entityType,
		EntityID:      entityID,
		Operation:     op,
		Attempts:      0,
		NextAttemptAt: time.Now().UTC(),
		Payload:       payload,
		CreatedAt:     time.Now().UTC(),
	}

	query := `
	INSERT INTO sync_jobs (id, entity_type, entity_id, operation, attempts, next_attempt_at, payload, created_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := db.Exec(
		query,
		job.ID,
		job.EntityType,
		job.EntityID,
		string(job.Operation),
		job.Attempts,
		job.NextAttemptAt.Format(time.RFC3339),
		job.Payload,
		job.CreatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to enqueue sync job: %w", err)
	}

	return &job, nil
}

func GetPendingJobs(db *sql.DB, limit int) ([]SyncJob, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	query := `
	SELECT id, entity_type, entity_id, operation, attempts, next_attempt_at, last_error, payload, created_at
	FROM sync_jobs
	WHERE next_attempt_at <= ?
	ORDER BY created_at ASC
	LIMIT ?
	`

	rows, err := db.Query(query, now, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query pending sync jobs: %w", err)
	}
	defer rows.Close()

	var jobs []SyncJob
	for rows.Next() {
		var job SyncJob
		var opStr, nextStr, createdStr string
		var lastErr, payload sql.NullString

		if err := rows.Scan(
			&job.ID,
			&job.EntityType,
			&job.EntityID,
			&opStr,
			&job.Attempts,
			&nextStr,
			&lastErr,
			&payload,
			&createdStr,
		); err != nil {
			return nil, fmt.Errorf("failed to scan sync job: %w", err)
		}

		job.Operation = SyncOperation(opStr)
		if lastErr.Valid {
			job.LastError = lastErr.String
		}
		if payload.Valid {
			job.Payload = payload.String
		}
		if t, err := time.Parse(time.RFC3339, nextStr); err == nil {
			job.NextAttemptAt = t
		}
		if t, err := time.Parse(time.RFC3339, createdStr); err == nil {
			job.CreatedAt = t
		}

		jobs = append(jobs, job)
	}

	return jobs, nil
}

func RecordJobFailure(db *sql.DB, jobID string, errMsg string, currentAttempts int) error {
	newAttempts := currentAttempts + 1
	// Exponential backoff: 2s, 4s, 8s, 16s, 32s, 64s, max 300s
	delaySeconds := math.Min(300, math.Pow(2, float64(newAttempts)))
	nextAttempt := time.Now().UTC().Add(time.Duration(delaySeconds) * time.Second)

	query := `
	UPDATE sync_jobs
	SET attempts = ?, next_attempt_at = ?, last_error = ?
	WHERE id = ?
	`

	_, err := db.Exec(query, newAttempts, nextAttempt.Format(time.RFC3339), errMsg, jobID)
	return err
}

func DeleteJob(db *sql.DB, jobID string) error {
	query := `DELETE FROM sync_jobs WHERE id = ?`
	_, err := db.Exec(query, jobID)
	return err
}

func GetPendingJobCount(db *sql.DB) (int, error) {
	// Chỉ đếm các job thực sự sẵn sàng để chạy (next_attempt_at <= now)
	now := time.Now().UTC().Format(time.RFC3339)
	query := `SELECT COUNT(*) FROM sync_jobs WHERE next_attempt_at <= ?`
	var count int
	err := db.QueryRow(query, now).Scan(&count)
	return count, err
}

func GetTotalJobCount(db *sql.DB) (int, error) {
	// Tổng số tất cả job (kể cả failed đang backoff)
	query := `SELECT COUNT(*) FROM sync_jobs`
	var count int
	err := db.QueryRow(query).Scan(&count)
	return count, err
}
