// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build nats

// Package eventprocessor provides DLQ persistence for production reliability.
// Phase 3: DLQ entries are persisted to DuckDB to survive restarts.
package eventprocessor

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/goccy/go-json"
	"github.com/tomtom215/cartographus/internal/logging"
)

// DLQStore defines the interface for DLQ persistence backends.
type DLQStore interface {
	// Save persists a DLQ entry.
	Save(ctx context.Context, entry *DLQEntry) error

	// Get retrieves an entry by event ID.
	Get(ctx context.Context, eventID string) (*DLQEntry, error)

	// Update modifies an existing entry (retry count, timestamps, etc.).
	Update(ctx context.Context, entry *DLQEntry) error

	// Delete removes an entry by event ID.
	Delete(ctx context.Context, eventID string) error

	// List returns all entries (for recovery on startup).
	List(ctx context.Context) ([]*DLQEntry, error)

	// DeleteExpired removes entries older than retention time.
	DeleteExpired(ctx context.Context, olderThan time.Time) (int64, error)

	// Count returns the total number of entries.
	Count(ctx context.Context) (int64, error)
}

// DuckDBDLQStore implements DLQStore using DuckDB for persistent storage.
// This enables DLQ entries to survive application restarts.
type DuckDBDLQStore struct {
	db *sql.DB
	mu sync.RWMutex
}

// NewDuckDBDLQStore creates a new DuckDB-backed DLQ store.
// The caller must call CreateTable() to ensure the schema exists.
func NewDuckDBDLQStore(db *sql.DB) *DuckDBDLQStore {
	return &DuckDBDLQStore{db: db}
}

// CreateTable creates the dlq_entries table if it doesn't exist.
// Should be called during application initialization.
func (s *DuckDBDLQStore) CreateTable(ctx context.Context) error {
	// Execute each statement separately (DuckDB doesn't support multi-statement execution)
	statements := []string{
		`CREATE TABLE IF NOT EXISTS dlq_entries (
			event_id TEXT PRIMARY KEY,
			message_id TEXT NOT NULL,
			event_data JSON NOT NULL,
			original_error TEXT NOT NULL,
			last_error TEXT NOT NULL,
			retry_count INTEGER NOT NULL DEFAULT 0,
			first_failure TIMESTAMPTZ NOT NULL,
			last_failure TIMESTAMPTZ NOT NULL,
			next_retry TIMESTAMPTZ NOT NULL,
			category INTEGER NOT NULL DEFAULT 0,
			created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_dlq_next_retry ON dlq_entries(next_retry)`,
		`CREATE INDEX IF NOT EXISTS idx_dlq_first_failure ON dlq_entries(first_failure)`,
		`CREATE INDEX IF NOT EXISTS idx_dlq_category ON dlq_entries(category)`,
	}

	for _, stmt := range statements {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("failed to execute DLQ schema statement: %w", err)
		}
	}

	// Force a checkpoint after creating the table to flush the WAL.
	// This prevents a DuckDB bug where WAL replay of CREATE TABLE statements
	// with TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP fails.
	if _, err := s.db.ExecContext(ctx, "CHECKPOINT"); err != nil {
		logging.Warn().Err(err).Msg("Failed to checkpoint after dlq_entries table creation")
	}

	logging.Info().Msg("DLQ entries table created/verified")
	return nil
}

// Save persists a DLQ entry to DuckDB.
func (s *DuckDBDLQStore) Save(ctx context.Context, entry *DLQEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if entry == nil || entry.Event == nil {
		return errors.New("entry and event cannot be nil")
	}

	// Serialize event data to JSON
	eventData, err := json.Marshal(entry.Event)
	if err != nil {
		return fmt.Errorf("failed to marshal event data: %w", err)
	}

	query := `
		INSERT INTO dlq_entries (
			event_id, message_id, event_data,
			original_error, last_error, retry_count,
			first_failure, last_failure, next_retry,
			category
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (event_id) DO UPDATE SET
			last_error = EXCLUDED.last_error,
			retry_count = EXCLUDED.retry_count,
			last_failure = EXCLUDED.last_failure,
			next_retry = EXCLUDED.next_retry
	`

	_, err = s.db.ExecContext(ctx, query,
		entry.Event.EventID,
		entry.MessageID,
		string(eventData),
		entry.OriginalError,
		entry.LastError,
		entry.RetryCount,
		entry.FirstFailure,
		entry.LastFailure,
		entry.NextRetry,
		int(entry.Category),
	)

	if err != nil {
		return fmt.Errorf("failed to save DLQ entry: %w", err)
	}

	return nil
}

// Get retrieves a DLQ entry by event ID.
func (s *DuckDBDLQStore) Get(ctx context.Context, eventID string) (*DLQEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `
		SELECT
			event_id, message_id, CAST(event_data AS VARCHAR),
			original_error, last_error, retry_count,
			first_failure, last_failure, next_retry,
			category
		FROM dlq_entries
		WHERE event_id = ?
	`

	row := s.db.QueryRowContext(ctx, query, eventID)
	return s.scanEntry(row)
}

// Update modifies an existing DLQ entry.
func (s *DuckDBDLQStore) Update(ctx context.Context, entry *DLQEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if entry == nil || entry.Event == nil {
		return errors.New("entry and event cannot be nil")
	}

	query := `
		UPDATE dlq_entries SET
			last_error = ?,
			retry_count = ?,
			last_failure = ?,
			next_retry = ?
		WHERE event_id = ?
	`

	result, err := s.db.ExecContext(ctx, query,
		entry.LastError,
		entry.RetryCount,
		entry.LastFailure,
		entry.NextRetry,
		entry.Event.EventID,
	)

	if err != nil {
		return fmt.Errorf("failed to update DLQ entry: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("DLQ entry not found: %s", entry.Event.EventID)
	}

	return nil
}

// Delete removes a DLQ entry by event ID.
func (s *DuckDBDLQStore) Delete(ctx context.Context, eventID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	query := `DELETE FROM dlq_entries WHERE event_id = ?`
	_, err := s.db.ExecContext(ctx, query, eventID)
	if err != nil {
		return fmt.Errorf("failed to delete DLQ entry: %w", err)
	}

	return nil
}

// List returns all DLQ entries (used for recovery on startup).
func (s *DuckDBDLQStore) List(ctx context.Context) ([]*DLQEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `
		SELECT
			event_id, message_id, CAST(event_data AS VARCHAR),
			original_error, last_error, retry_count,
			first_failure, last_failure, next_retry,
			category
		FROM dlq_entries
		ORDER BY first_failure ASC
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list DLQ entries: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var entries []*DLQEntry
	for rows.Next() {
		entry, err := s.scanEntryFromRows(rows)
		if err != nil {
			logging.Warn().Err(err).Msg("Failed to scan DLQ entry row")
			continue
		}
		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating DLQ entries: %w", err)
	}

	return entries, nil
}

// DeleteExpired removes entries older than the specified time.
func (s *DuckDBDLQStore) DeleteExpired(ctx context.Context, olderThan time.Time) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	query := `DELETE FROM dlq_entries WHERE first_failure < ?`

	result, err := s.db.ExecContext(ctx, query, olderThan)
	if err != nil {
		return 0, fmt.Errorf("failed to delete expired DLQ entries: %w", err)
	}

	count, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get deleted count: %w", err)
	}

	if count > 0 {
		logging.Info().Int64("deleted", count).Time("older_than", olderThan).
			Msg("Deleted expired DLQ entries")
	}

	return count, nil
}

// Count returns the total number of DLQ entries.
func (s *DuckDBDLQStore) Count(ctx context.Context) (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var count int64
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM dlq_entries").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count DLQ entries: %w", err)
	}

	return count, nil
}

// scanEntry scans a single row into a DLQEntry.
func (s *DuckDBDLQStore) scanEntry(row *sql.Row) (*DLQEntry, error) {
	var eventID, messageID, eventData string
	var originalError, lastError string
	var retryCount, category int
	var firstFailure, lastFailure, nextRetry time.Time

	err := row.Scan(
		&eventID,
		&messageID,
		&eventData,
		&originalError,
		&lastError,
		&retryCount,
		&firstFailure,
		&lastFailure,
		&nextRetry,
		&category,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	// Deserialize event data
	var event MediaEvent
	if err := json.Unmarshal([]byte(eventData), &event); err != nil {
		return nil, fmt.Errorf("failed to unmarshal event data: %w", err)
	}

	return &DLQEntry{
		Event:         &event,
		MessageID:     messageID,
		OriginalError: originalError,
		LastError:     lastError,
		RetryCount:    retryCount,
		FirstFailure:  firstFailure,
		LastFailure:   lastFailure,
		NextRetry:     nextRetry,
		Category:      ErrorCategory(category),
	}, nil
}

// scanEntryFromRows scans a row from sql.Rows into a DLQEntry.
func (s *DuckDBDLQStore) scanEntryFromRows(rows *sql.Rows) (*DLQEntry, error) {
	var eventID, messageID, eventData string
	var originalError, lastError string
	var retryCount, category int
	var firstFailure, lastFailure, nextRetry time.Time

	err := rows.Scan(
		&eventID,
		&messageID,
		&eventData,
		&originalError,
		&lastError,
		&retryCount,
		&firstFailure,
		&lastFailure,
		&nextRetry,
		&category,
	)
	if err != nil {
		return nil, err
	}

	// Deserialize event data
	var event MediaEvent
	if err := json.Unmarshal([]byte(eventData), &event); err != nil {
		return nil, fmt.Errorf("failed to unmarshal event data: %w", err)
	}

	return &DLQEntry{
		Event:         &event,
		MessageID:     messageID,
		OriginalError: originalError,
		LastError:     lastError,
		RetryCount:    retryCount,
		FirstFailure:  firstFailure,
		LastFailure:   lastFailure,
		NextRetry:     nextRetry,
		Category:      ErrorCategory(category),
	}, nil
}

// PersistentDLQHandler wraps DLQHandler with persistence support.
// It maintains in-memory state for performance while persisting to DuckDB.
type PersistentDLQHandler struct {
	*DLQHandler
	store DLQStore
}

// NewPersistentDLQHandler creates a DLQ handler with persistence.
// On startup, it loads entries from the persistent store.
func NewPersistentDLQHandler(cfg DLQConfig, store DLQStore) (*PersistentDLQHandler, error) {
	handler, err := NewDLQHandler(cfg)
	if err != nil {
		return nil, err
	}

	pHandler := &PersistentDLQHandler{
		DLQHandler: handler,
		store:      store,
	}

	// Load persisted entries on startup
	if err := pHandler.loadPersistedEntries(); err != nil {
		logging.Warn().Err(err).Msg("Failed to load persisted DLQ entries")
		// Continue anyway - entries can be recovered later
	}

	return pHandler, nil
}

// loadPersistedEntries recovers DLQ entries from persistent storage.
func (h *PersistentDLQHandler) loadPersistedEntries() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	entries, err := h.store.List(ctx)
	if err != nil {
		return fmt.Errorf("failed to list persisted entries: %w", err)
	}

	loaded := 0
	for _, entry := range entries {
		// Re-add to in-memory handler
		h.mu.Lock()
		h.entries.Push(entry.Event.EventID, entry, entry.FirstFailure)
		h.mu.Unlock()
		loaded++
	}

	if loaded > 0 {
		logging.Info().Int("count", loaded).Msg("Loaded DLQ entries from persistent storage")
	}

	return nil
}

// AddEntry adds a failed event to both memory and persistent storage.
func (h *PersistentDLQHandler) AddEntry(event *MediaEvent, err error, messageID string) *DLQEntry {
	// Add to in-memory handler first
	entry := h.DLQHandler.AddEntry(event, err, messageID)
	if entry == nil {
		return nil
	}

	// Persist asynchronously (fire and forget)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if saveErr := h.store.Save(ctx, entry); saveErr != nil {
			logging.Error().Err(saveErr).Str("event_id", event.EventID).
				Msg("Failed to persist DLQ entry")
		}
	}()

	return entry
}

// IncrementRetry updates retry count in both memory and persistent storage.
func (h *PersistentDLQHandler) IncrementRetry(eventID string, err error) bool {
	// Update in-memory first
	moreRetries := h.DLQHandler.IncrementRetry(eventID, err)

	// Get the updated entry and persist
	entry := h.GetEntry(eventID)
	if entry != nil {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			if updateErr := h.store.Update(ctx, entry); updateErr != nil {
				logging.Error().Err(updateErr).Str("event_id", eventID).
					Msg("Failed to persist DLQ retry update")
			}
		}()
	}

	return moreRetries
}

// RemoveEntry removes from both memory and persistent storage.
func (h *PersistentDLQHandler) RemoveEntry(eventID string) bool {
	// Remove from in-memory first
	removed := h.DLQHandler.RemoveEntry(eventID)

	// Delete from persistent storage
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if deleteErr := h.store.Delete(ctx, eventID); deleteErr != nil {
			logging.Error().Err(deleteErr).Str("event_id", eventID).
				Msg("Failed to delete persisted DLQ entry")
		}
	}()

	return removed
}

// Cleanup removes expired entries from both memory and persistent storage.
func (h *PersistentDLQHandler) Cleanup() int {
	// Cleanup in-memory first
	count := h.DLQHandler.Cleanup()

	// Cleanup persistent storage
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		cutoff := time.Now().Add(-h.config.RetentionTime)
		if _, deleteErr := h.store.DeleteExpired(ctx, cutoff); deleteErr != nil {
			logging.Error().Err(deleteErr).Msg("Failed to cleanup persisted DLQ entries")
		}
	}()

	return count
}
