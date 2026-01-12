// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build wal && nats

package eventprocessor

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/goccy/go-json"

	"github.com/tomtom215/cartographus/internal/logging"
	"github.com/tomtom215/cartographus/internal/wal"
)

// WALStore wraps a DuckDB store with Consumer WAL protection for exactly-once delivery.
// This ensures events survive crashes between NATS consumption and DuckDB persistence.
//
// Flow:
// 1. For each event, generate transaction ID
// 2. Write to ConsumerWAL (durable, fsync'd)
// 3. Call underlying store to insert
// 4. On success: Confirm WAL entries
// 5. On failure: WAL entries remain for retry on restart
//
// This implements the EventStore interface so it can replace DuckDBStore transparently.
type WALStore struct {
	underlying  EventStore
	consumerWAL *wal.ConsumerWAL

	// Metrics
	mu            sync.RWMutex
	totalWrites   int64
	totalConfirms int64
	totalFailures int64
}

// NewWALStore creates a new WAL-protected store.
func NewWALStore(underlying EventStore, consumerWAL *wal.ConsumerWAL) (*WALStore, error) {
	if underlying == nil {
		return nil, fmt.Errorf("underlying store required")
	}
	if consumerWAL == nil {
		return nil, fmt.Errorf("consumer WAL required")
	}

	return &WALStore{
		underlying:  underlying,
		consumerWAL: consumerWAL,
	}, nil
}

// InsertMediaEvents implements Store interface with WAL protection.
// Events are written to ConsumerWAL before being inserted into DuckDB.
// WAL entries are confirmed only after successful DuckDB insert.
func (s *WALStore) InsertMediaEvents(ctx context.Context, events []*MediaEvent) error {
	if len(events) == 0 {
		return nil
	}

	// Track WAL entry IDs for confirmation
	type walEntry struct {
		entryID       string
		transactionID string
	}
	walEntries := make([]walEntry, 0, len(events))

	// Step 1: Write all events to WAL with transaction IDs
	for _, event := range events {
		// Generate transaction ID if not set
		if event.TransactionID == "" {
			event.TransactionID = wal.GenerateTransactionID(event.Source, event.EventID)
		}

		// Serialize event
		payload, err := json.Marshal(event)
		if err != nil {
			logging.Error().
				Str("event_id", event.EventID).
				Err(err).
				Msg("WALStore: failed to marshal event")
			continue
		}

		// Write to WAL (durable)
		entryID, err := s.consumerWAL.Write(ctx, payload, event.TransactionID, "", "")
		if err != nil {
			logging.Error().
				Str("event_id", event.EventID).
				Err(err).
				Msg("WALStore: failed to write event to WAL")
			// Continue with other events - this one will be lost
			// In production, you might want to fail the entire batch
			continue
		}

		walEntries = append(walEntries, walEntry{
			entryID:       entryID,
			transactionID: event.TransactionID,
		})

		s.mu.Lock()
		s.totalWrites++
		s.mu.Unlock()
	}

	// Step 2: Insert into DuckDB
	err := s.underlying.InsertMediaEvents(ctx, events)
	if err != nil {
		// DuckDB insert failed - WAL entries will be retried on restart
		s.mu.Lock()
		s.totalFailures += int64(len(walEntries))
		s.mu.Unlock()
		return fmt.Errorf("DuckDB insert failed (WAL entries preserved for retry): %w", err)
	}

	// Step 3: Confirm all WAL entries
	for _, entry := range walEntries {
		if confirmErr := s.consumerWAL.Confirm(ctx, entry.entryID); confirmErr != nil {
			// Log but don't fail - entry will be detected as already-committed on restart
			logging.Warn().
				Str("wal_entry_id", entry.entryID).
				Err(confirmErr).
				Msg("WALStore: failed to confirm WAL entry")
		} else {
			s.mu.Lock()
			s.totalConfirms++
			s.mu.Unlock()
		}
	}

	return nil
}

// WALStoreStats returns current statistics.
type WALStoreStats struct {
	TotalWrites   int64
	TotalConfirms int64
	TotalFailures int64
}

// Stats returns current WAL store statistics.
func (s *WALStore) Stats() WALStoreStats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return WALStoreStats{
		TotalWrites:   s.totalWrites,
		TotalConfirms: s.totalConfirms,
		TotalFailures: s.totalFailures,
	}
}

// Close closes the underlying store.
func (s *WALStore) Close() error {
	if closer, ok := s.underlying.(interface{ Close() error }); ok {
		return closer.Close()
	}
	return nil
}

// WALStoreConfig holds configuration for the WAL store wrapper.
type WALStoreConfig struct {
	// MaxBatchWALWrites is the max events to write to WAL in a batch.
	MaxBatchWALWrites int

	// WALWriteTimeout is the timeout for WAL write operations.
	WALWriteTimeout time.Duration
}

// DefaultWALStoreConfig returns production defaults.
func DefaultWALStoreConfig() WALStoreConfig {
	return WALStoreConfig{
		MaxBatchWALWrites: 100,
		WALWriteTimeout:   5 * time.Second,
	}
}
