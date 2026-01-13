// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build nats

// Package eventprocessor provides checkpoint tracking for deterministic event replay.
// Checkpoints enable resuming replay from the last processed position after restarts.
package eventprocessor

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/tomtom215/cartographus/internal/logging"
)

// Checkpoint represents a saved replay position.
type Checkpoint struct {
	ID             int64     // Auto-generated ID
	ConsumerName   string    // Name of the consumer (durable name)
	StreamName     string    // Name of the NATS stream
	LastSequence   uint64    // Last processed sequence number
	LastTimestamp  time.Time // Timestamp of last processed message
	ProcessedCount int64     // Total messages processed
	ErrorCount     int64     // Total errors encountered
	Status         string    // Current status (running, paused, completed, error)
	ReplayMode     string    // Mode used for this replay
	StartSequence  uint64    // Starting sequence for this replay
	StartTime      time.Time // Starting timestamp for this replay
	CreatedAt      time.Time // When checkpoint was created
	UpdatedAt      time.Time // When checkpoint was last updated
}

// CheckpointStore persists checkpoint data to DuckDB.
type CheckpointStore struct {
	db *sql.DB
	mu sync.RWMutex
}

// NewCheckpointStore creates a new checkpoint store.
func NewCheckpointStore(db *sql.DB) *CheckpointStore {
	return &CheckpointStore{db: db}
}

// CreateTable creates the replay_checkpoints table if it doesn't exist.
func (s *CheckpointStore) CreateTable(ctx context.Context) error {
	// Execute statements separately (DuckDB doesn't support multi-statement)
	statements := []string{
		`CREATE TABLE IF NOT EXISTS replay_checkpoints (
			id INTEGER PRIMARY KEY,
			consumer_name TEXT NOT NULL,
			stream_name TEXT NOT NULL,
			last_sequence BIGINT NOT NULL DEFAULT 0,
			last_timestamp TIMESTAMPTZ,
			processed_count BIGINT NOT NULL DEFAULT 0,
			error_count BIGINT NOT NULL DEFAULT 0,
			status TEXT NOT NULL DEFAULT 'initialized',
			replay_mode TEXT,
			start_sequence BIGINT,
			start_time TIMESTAMPTZ,
			created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(consumer_name, stream_name)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_checkpoint_consumer ON replay_checkpoints(consumer_name)`,
		`CREATE INDEX IF NOT EXISTS idx_checkpoint_stream ON replay_checkpoints(stream_name)`,
		`CREATE INDEX IF NOT EXISTS idx_checkpoint_status ON replay_checkpoints(status)`,
		`CREATE INDEX IF NOT EXISTS idx_checkpoint_updated ON replay_checkpoints(updated_at DESC)`,
	}

	for _, stmt := range statements {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("failed to execute checkpoint schema: %w", err)
		}
	}

	// Force a checkpoint after creating the table to flush the WAL.
	// This prevents a DuckDB bug where WAL replay of CREATE TABLE statements
	// with TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP fails.
	if _, err := s.db.ExecContext(ctx, "CHECKPOINT"); err != nil {
		logging.Warn().Err(err).Msg("Failed to checkpoint after replay_checkpoints table creation")
	}

	logging.Info().Msg("Replay checkpoints table created/verified")
	return nil
}

// Save persists or updates a checkpoint.
func (s *CheckpointStore) Save(ctx context.Context, cp *Checkpoint) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()

	query := `
		INSERT INTO replay_checkpoints (
			id, consumer_name, stream_name, last_sequence, last_timestamp,
			processed_count, error_count, status, replay_mode,
			start_sequence, start_time, created_at, updated_at
		) VALUES (
			(SELECT COALESCE(MAX(id), 0) + 1 FROM replay_checkpoints),
			?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
		)
		ON CONFLICT (consumer_name, stream_name) DO UPDATE SET
			last_sequence = EXCLUDED.last_sequence,
			last_timestamp = EXCLUDED.last_timestamp,
			processed_count = EXCLUDED.processed_count,
			error_count = EXCLUDED.error_count,
			status = EXCLUDED.status,
			updated_at = EXCLUDED.updated_at
	`

	_, err := s.db.ExecContext(ctx, query,
		cp.ConsumerName,
		cp.StreamName,
		cp.LastSequence,
		cp.LastTimestamp,
		cp.ProcessedCount,
		cp.ErrorCount,
		cp.Status,
		cp.ReplayMode,
		cp.StartSequence,
		cp.StartTime,
		now, // created_at
		now, // updated_at
	)

	if err != nil {
		return fmt.Errorf("failed to save checkpoint: %w", err)
	}

	return nil
}

// Get retrieves a checkpoint by consumer and stream name.
func (s *CheckpointStore) Get(ctx context.Context, consumerName, streamName string) (*Checkpoint, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `
		SELECT id, consumer_name, stream_name, last_sequence, last_timestamp,
			processed_count, error_count, status, replay_mode,
			start_sequence, start_time, created_at, updated_at
		FROM replay_checkpoints
		WHERE consumer_name = ? AND stream_name = ?
	`

	row := s.db.QueryRowContext(ctx, query, consumerName, streamName)
	return s.scanCheckpoint(row)
}

// GetByID retrieves a checkpoint by ID.
func (s *CheckpointStore) GetByID(ctx context.Context, id int64) (*Checkpoint, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `
		SELECT id, consumer_name, stream_name, last_sequence, last_timestamp,
			processed_count, error_count, status, replay_mode,
			start_sequence, start_time, created_at, updated_at
		FROM replay_checkpoints
		WHERE id = ?
	`

	row := s.db.QueryRowContext(ctx, query, id)
	return s.scanCheckpoint(row)
}

// List returns all checkpoints, optionally filtered by status.
func (s *CheckpointStore) List(ctx context.Context, status string) ([]*Checkpoint, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var query string
	var args []interface{}

	if status != "" {
		query = `
			SELECT id, consumer_name, stream_name, last_sequence, last_timestamp,
				processed_count, error_count, status, replay_mode,
				start_sequence, start_time, created_at, updated_at
			FROM replay_checkpoints
			WHERE status = ?
			ORDER BY updated_at DESC
		`
		args = append(args, status)
	} else {
		query = `
			SELECT id, consumer_name, stream_name, last_sequence, last_timestamp,
				processed_count, error_count, status, replay_mode,
				start_sequence, start_time, created_at, updated_at
			FROM replay_checkpoints
			ORDER BY updated_at DESC
		`
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list checkpoints: %w", err)
	}
	defer rows.Close()

	var checkpoints []*Checkpoint
	for rows.Next() {
		cp, err := s.scanCheckpointFromRows(rows)
		if err != nil {
			logging.Warn().Err(err).Msg("Failed to scan checkpoint row")
			continue
		}
		checkpoints = append(checkpoints, cp)
	}

	return checkpoints, rows.Err()
}

// Delete removes a checkpoint by ID.
func (s *CheckpointStore) Delete(ctx context.Context, id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	query := `DELETE FROM replay_checkpoints WHERE id = ?`
	result, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete checkpoint: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get deleted count: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("checkpoint not found: %d", id)
	}

	return nil
}

// DeleteOld removes checkpoints older than the specified duration.
func (s *CheckpointStore) DeleteOld(ctx context.Context, olderThan time.Duration) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().Add(-olderThan)
	query := `DELETE FROM replay_checkpoints WHERE updated_at < ? AND status IN ('completed', 'error', 'canceled')`

	result, err := s.db.ExecContext(ctx, query, cutoff)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old checkpoints: %w", err)
	}

	count, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get deleted count: %w", err)
	}

	if count > 0 {
		logging.Info().Int64("deleted", count).Dur("older_than", olderThan).
			Msg("Deleted old replay checkpoints")
	}

	return count, nil
}

// GetLastForStream returns the most recent checkpoint for a stream.
func (s *CheckpointStore) GetLastForStream(ctx context.Context, streamName string) (*Checkpoint, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `
		SELECT id, consumer_name, stream_name, last_sequence, last_timestamp,
			processed_count, error_count, status, replay_mode,
			start_sequence, start_time, created_at, updated_at
		FROM replay_checkpoints
		WHERE stream_name = ?
		ORDER BY last_sequence DESC
		LIMIT 1
	`

	row := s.db.QueryRowContext(ctx, query, streamName)
	return s.scanCheckpoint(row)
}

// scanCheckpoint scans a single row into a Checkpoint.
func (s *CheckpointStore) scanCheckpoint(row *sql.Row) (*Checkpoint, error) {
	var cp Checkpoint
	var lastTimestamp, startTime sql.NullTime
	var replayMode sql.NullString
	var startSequence sql.NullInt64

	err := row.Scan(
		&cp.ID,
		&cp.ConsumerName,
		&cp.StreamName,
		&cp.LastSequence,
		&lastTimestamp,
		&cp.ProcessedCount,
		&cp.ErrorCount,
		&cp.Status,
		&replayMode,
		&startSequence,
		&startTime,
		&cp.CreatedAt,
		&cp.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	if lastTimestamp.Valid {
		cp.LastTimestamp = lastTimestamp.Time
	}
	if replayMode.Valid {
		cp.ReplayMode = replayMode.String
	}
	if startSequence.Valid {
		cp.StartSequence = uint64(startSequence.Int64)
	}
	if startTime.Valid {
		cp.StartTime = startTime.Time
	}

	return &cp, nil
}

// scanCheckpointFromRows scans a row from sql.Rows into a Checkpoint.
func (s *CheckpointStore) scanCheckpointFromRows(rows *sql.Rows) (*Checkpoint, error) {
	var cp Checkpoint
	var lastTimestamp, startTime sql.NullTime
	var replayMode sql.NullString
	var startSequence sql.NullInt64

	err := rows.Scan(
		&cp.ID,
		&cp.ConsumerName,
		&cp.StreamName,
		&cp.LastSequence,
		&lastTimestamp,
		&cp.ProcessedCount,
		&cp.ErrorCount,
		&cp.Status,
		&replayMode,
		&startSequence,
		&startTime,
		&cp.CreatedAt,
		&cp.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if lastTimestamp.Valid {
		cp.LastTimestamp = lastTimestamp.Time
	}
	if replayMode.Valid {
		cp.ReplayMode = replayMode.String
	}
	if startSequence.Valid {
		cp.StartSequence = uint64(startSequence.Int64)
	}
	if startTime.Valid {
		cp.StartTime = startTime.Time
	}

	return &cp, nil
}
