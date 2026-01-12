// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build wal

package wal

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/dgraph-io/badger/v4/options"
	"github.com/goccy/go-json"
	"github.com/google/uuid"
	"github.com/tomtom215/cartographus/internal/logging"
)

// WAL provides durable write-ahead logging before NATS publish.
// Events are persisted to BadgerDB (ACID, fsync) before NATS publishing,
// ensuring no event loss in case of NATS failures or process crashes.
//
// The WAL stores events as raw JSON bytes, making it agnostic to the
// specific event type. This allows reuse across different event formats.
type WAL interface {
	// Write persists an event before NATS publish (ACID, durable).
	// The event is serialized to JSON and stored. Returns an entry ID
	// for later confirmation.
	Write(ctx context.Context, event interface{}) (entryID string, err error)

	// Confirm marks an entry as successfully published to NATS.
	// The entry will be cleaned up during the next compaction.
	Confirm(ctx context.Context, entryID string) error

	// GetPending returns all unconfirmed entries for retry.
	// Used on startup recovery and by the retry loop.
	GetPending(ctx context.Context) ([]*Entry, error)

	// Stats returns WAL metrics.
	Stats() Stats

	// Close gracefully shuts down the WAL.
	Close() error
}

// Entry represents a single WAL entry containing an event and metadata.
type Entry struct {
	// ID is the unique identifier for this WAL entry.
	ID string `json:"id"`

	// Payload is the serialized event data (JSON).
	// Use UnmarshalPayload to deserialize into a specific type.
	Payload json.RawMessage `json:"payload"`

	// CreatedAt is when the entry was written to the WAL.
	CreatedAt time.Time `json:"created_at"`

	// Attempts is the number of NATS publish attempts.
	Attempts int `json:"attempts"`

	// LastAttemptAt is the time of the last publish attempt.
	LastAttemptAt time.Time `json:"last_attempt_at,omitempty"`

	// LastError is the error message from the last failed attempt.
	LastError string `json:"last_error,omitempty"`

	// Confirmed indicates if the entry was successfully published.
	Confirmed bool `json:"confirmed"`

	// ConfirmedAt is when the entry was confirmed.
	ConfirmedAt *time.Time `json:"confirmed_at,omitempty"`

	// LeaseExpiry is when the current processing lease expires.
	// Zero value means no active lease. Entries can be claimed when:
	// 1. LeaseExpiry.IsZero() - no active lease
	// 2. time.Now().After(LeaseExpiry) - lease expired (crash recovery)
	// This provides durable, crash-safe concurrent processing prevention.
	LeaseExpiry time.Time `json:"lease_expiry,omitempty"`

	// LeaseHolder identifies the processor holding the lease.
	// Format: UUID or instance identifier. Used for debugging and auditing.
	LeaseHolder string `json:"lease_holder,omitempty"`
}

// UnmarshalPayload deserializes the payload into the given type.
func (e *Entry) UnmarshalPayload(v interface{}) error {
	return json.Unmarshal(e.Payload, v)
}

// Stats contains WAL metrics for monitoring.
type Stats struct {
	// PendingCount is the number of unconfirmed entries.
	PendingCount int64

	// ConfirmedCount is the number of confirmed entries awaiting compaction.
	ConfirmedCount int64

	// TotalWrites is the total number of Write operations.
	TotalWrites int64

	// TotalConfirms is the total number of Confirm operations.
	TotalConfirms int64

	// TotalRetries is the total number of retry attempts.
	TotalRetries int64

	// LastCompaction is the time of the last compaction.
	LastCompaction time.Time

	// DBSizeBytes is the estimated database size.
	DBSizeBytes int64
}

// BadgerWAL implements WAL using BadgerDB for durable storage.
//
// RACE CONDITION PREVENTION (v2.4):
// The processingEntries map prevents concurrent processing of the same entry.
// This fixes the race condition where multiple goroutines (recovery, retry loop)
// could process the same pending entry simultaneously, causing errors like:
// - Duplicate NATS publishes
// - ErrEntryNotFound when confirming (entry already confirmed by another goroutine)
type BadgerWAL struct {
	db     *badger.DB
	config Config

	// Statistics
	totalWrites   atomic.Int64
	totalConfirms atomic.Int64
	totalRetries  atomic.Int64

	// State tracking
	lastCompaction time.Time
	mu             sync.RWMutex
	closed         bool

	// Metrics callback (optional)
	metricsCallback func(Stats)

	// RACE CONDITION FIX (v2.4): Tracks entries currently being processed.
	// Key: entry ID (string), Value: time.Time when claim was acquired.
	// This prevents multiple goroutines from processing the same entry.
	processingEntries sync.Map
}

// Prefix keys for different entry types
const (
	prefixPending   = "pending:"
	prefixConfirmed = "confirmed:"
)

// Open creates a new BadgerWAL with the given configuration.
// The BadgerDB database is opened (or created) at the configured path.
func Open(cfg *Config) (*BadgerWAL, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid WAL config: %w", err)
	}

	opts := badger.DefaultOptions(cfg.Path)
	opts.SyncWrites = cfg.SyncWrites
	opts.MemTableSize = cfg.MemTableSize
	opts.ValueLogFileSize = cfg.ValueLogFileSize
	opts.NumCompactors = cfg.NumCompactors

	// Apply compression if enabled
	if cfg.Compression {
		opts.Compression = options.Snappy
	}

	// Apply additional tuning options
	if cfg.NumMemtables > 0 {
		opts.NumMemtables = cfg.NumMemtables
	}
	if cfg.BlockCacheSize > 0 {
		opts.BlockCacheSize = cfg.BlockCacheSize
	}
	if cfg.IndexCacheSize > 0 {
		opts.IndexCacheSize = cfg.IndexCacheSize
	}

	// Reduce logging verbosity
	opts.Logger = nil

	db, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("open BadgerDB: %w", err)
	}

	wal := &BadgerWAL{
		db:             db,
		config:         *cfg,
		lastCompaction: time.Now(),
	}

	logging.Info().
		Str("path", cfg.Path).
		Bool("sync_writes", cfg.SyncWrites).
		Bool("compression", cfg.Compression).
		Msg("WAL opened")
	return wal, nil
}

// OpenForTesting creates a BadgerWAL without configuration validation.
// This is intended for unit tests that need faster intervals than production minimums.
// WARNING: Do not use in production code.
func OpenForTesting(cfg *Config) (*BadgerWAL, error) {
	// Ensure minimum BadgerDB requirements even for tests
	if cfg.NumCompactors < 2 {
		cfg.NumCompactors = 2
	}

	// Set defaults for new fields if not set
	if cfg.GCRatio == 0 {
		cfg.GCRatio = 0.5
	}
	if cfg.CloseTimeout == 0 {
		cfg.CloseTimeout = 30 * time.Second
	}

	opts := badger.DefaultOptions(cfg.Path)
	opts.SyncWrites = cfg.SyncWrites
	opts.MemTableSize = cfg.MemTableSize
	opts.ValueLogFileSize = cfg.ValueLogFileSize
	opts.NumCompactors = cfg.NumCompactors

	// Apply compression if enabled
	if cfg.Compression {
		opts.Compression = options.Snappy
	}

	// Reduce logging verbosity
	opts.Logger = nil

	db, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("open BadgerDB: %w", err)
	}

	wal := &BadgerWAL{
		db:             db,
		config:         *cfg,
		lastCompaction: time.Now(),
	}

	return wal, nil
}

// Write persists an event to the WAL before NATS publishing.
// This operation is ACID-compliant with fsync when SyncWrites is enabled.
// The event can be any JSON-serializable type.
func (w *BadgerWAL) Write(ctx context.Context, event interface{}) (string, error) {
	start := time.Now()
	defer func() {
		RecordWALWriteLatency(time.Since(start).Seconds())
	}()

	w.mu.RLock()
	if w.closed {
		w.mu.RUnlock()
		return "", ErrWALClosed
	}
	w.mu.RUnlock()

	if event == nil {
		return "", ErrNilEvent
	}

	// Serialize event to JSON for storage
	payload, err := json.Marshal(event)
	if err != nil {
		return "", fmt.Errorf("marshal event: %w", err)
	}

	// Generate unique entry ID
	entryID := uuid.New().String()

	entry := &Entry{
		ID:        entryID,
		Payload:   payload,
		CreatedAt: time.Now().UTC(),
		Attempts:  0,
		Confirmed: false,
	}

	// Serialize entry
	data, err := json.Marshal(entry)
	if err != nil {
		return "", fmt.Errorf("marshal entry: %w", err)
	}

	// Write to BadgerDB with native TTL
	key := []byte(prefixPending + entryID)
	err = w.db.Update(func(txn *badger.Txn) error {
		e := badger.NewEntry(key, data)
		if w.config.EntryTTL > 0 {
			e = e.WithTTL(w.config.EntryTTL)
		}
		return txn.SetEntry(e)
	})
	if err != nil {
		return "", fmt.Errorf("write to BadgerDB: %w", err)
	}

	w.totalWrites.Add(1)
	RecordWALWrite()

	return entryID, nil
}

// Confirm marks an entry as successfully published to NATS.
// The entry is moved from pending to confirmed state.
func (w *BadgerWAL) Confirm(ctx context.Context, entryID string) error {
	w.mu.RLock()
	if w.closed {
		w.mu.RUnlock()
		return ErrWALClosed
	}
	w.mu.RUnlock()

	if entryID == "" {
		return ErrEmptyEntryID
	}

	pendingKey := []byte(prefixPending + entryID)
	confirmedKey := []byte(prefixConfirmed + entryID)

	err := w.db.Update(func(txn *badger.Txn) error {
		// Get the pending entry
		item, err := txn.Get(pendingKey)
		if errors.Is(err, badger.ErrKeyNotFound) {
			return ErrEntryNotFound
		}
		if err != nil {
			return fmt.Errorf("get pending entry: %w", err)
		}

		// Deserialize
		var entry Entry
		err = item.Value(func(val []byte) error {
			return json.Unmarshal(val, &entry)
		})
		if err != nil {
			return fmt.Errorf("unmarshal entry: %w", err)
		}

		// Update to confirmed
		now := time.Now().UTC()
		entry.Confirmed = true
		entry.ConfirmedAt = &now

		// Serialize updated entry
		data, err := json.Marshal(&entry)
		if err != nil {
			return fmt.Errorf("marshal confirmed entry: %w", err)
		}

		// Write confirmed entry
		if err := txn.Set(confirmedKey, data); err != nil {
			return fmt.Errorf("set confirmed entry: %w", err)
		}

		// Delete pending entry
		if err := txn.Delete(pendingKey); err != nil {
			return fmt.Errorf("delete pending entry: %w", err)
		}

		return nil
	})

	if err != nil {
		return err
	}

	w.totalConfirms.Add(1)
	RecordWALConfirm()

	return nil
}

// GetPending returns all unconfirmed entries.
// Used for recovery on startup and by the retry loop.
// Note: For large WALs, consider using GetPendingStream for memory efficiency.
//
// DETERMINISM: Uses BadgerDB's View() transaction which provides snapshot isolation.
// All entries returned are from a consistent point-in-time snapshot, ensuring
// no partial reads or phantom entries during concurrent writes.
func (w *BadgerWAL) GetPending(ctx context.Context) ([]*Entry, error) {
	w.mu.RLock()
	if w.closed {
		w.mu.RUnlock()
		return nil, ErrWALClosed
	}
	w.mu.RUnlock()

	var entries []*Entry

	err := w.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = true
		it := txn.NewIterator(opts)
		defer it.Close()

		prefix := []byte(prefixPending)
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			// Check for context cancellation
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			item := it.Item()

			var entry Entry
			err := item.Value(func(val []byte) error {
				return json.Unmarshal(val, &entry)
			})
			if err != nil {
				logging.Warn().Err(err).Str("key", string(item.Key())).Msg("WAL failed to unmarshal entry")
				continue
			}

			entries = append(entries, &entry)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("iterate pending entries: %w", err)
	}

	return entries, nil
}

// GetPendingStream returns a channel of pending entries for memory-efficient processing.
// This is useful for large WALs where loading all entries into memory at once
// would be prohibitive. The caller should read from the entry channel until it's closed,
// then check the error channel for any errors that occurred during iteration.
func (w *BadgerWAL) GetPendingStream(ctx context.Context) (<-chan *Entry, <-chan error) {
	entryCh := make(chan *Entry, 100)
	errCh := make(chan error, 1)

	go func() {
		defer close(entryCh)
		defer close(errCh)

		w.mu.RLock()
		if w.closed {
			w.mu.RUnlock()
			errCh <- ErrWALClosed
			return
		}
		w.mu.RUnlock()

		err := w.db.View(func(txn *badger.Txn) error {
			opts := badger.DefaultIteratorOptions
			opts.PrefetchValues = true
			it := txn.NewIterator(opts)
			defer it.Close()

			prefix := []byte(prefixPending)
			for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
				// Check for context cancellation
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
				}

				item := it.Item()

				var entry Entry
				err := item.Value(func(val []byte) error {
					return json.Unmarshal(val, &entry)
				})
				if err != nil {
					logging.Warn().Err(err).Str("key", string(item.Key())).Msg("WAL failed to unmarshal entry")
					continue // Skip malformed entries
				}

				// Send entry to channel
				select {
				case entryCh <- &entry:
				case <-ctx.Done():
					return ctx.Err()
				}
			}
			return nil
		})

		if err != nil {
			errCh <- fmt.Errorf("iterate pending entries: %w", err)
		}
	}()

	return entryCh, errCh
}

// UpdateAttempt updates an entry's attempt count and last error.
// Called after a failed NATS publish attempt.
func (w *BadgerWAL) UpdateAttempt(ctx context.Context, entryID string, lastError string) error {
	w.mu.RLock()
	if w.closed {
		w.mu.RUnlock()
		return ErrWALClosed
	}
	w.mu.RUnlock()

	key := []byte(prefixPending + entryID)

	err := w.db.Update(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if errors.Is(err, badger.ErrKeyNotFound) {
			return ErrEntryNotFound
		}
		if err != nil {
			return fmt.Errorf("get entry: %w", err)
		}

		var entry Entry
		err = item.Value(func(val []byte) error {
			return json.Unmarshal(val, &entry)
		})
		if err != nil {
			return fmt.Errorf("unmarshal entry: %w", err)
		}

		entry.Attempts++
		entry.LastAttemptAt = time.Now().UTC()
		entry.LastError = lastError

		data, err := json.Marshal(&entry)
		if err != nil {
			return fmt.Errorf("marshal entry: %w", err)
		}

		return txn.Set(key, data)
	})

	if err != nil {
		return err
	}

	w.totalRetries.Add(1)
	RecordWALRetry()

	return nil
}

// DeleteEntry permanently removes an entry from the WAL.
// Used when an entry exceeds max retries or is expired.
func (w *BadgerWAL) DeleteEntry(ctx context.Context, entryID string) error {
	w.mu.RLock()
	if w.closed {
		w.mu.RUnlock()
		return ErrWALClosed
	}
	w.mu.RUnlock()

	// Try both pending and confirmed prefixes
	pendingKey := []byte(prefixPending + entryID)
	confirmedKey := []byte(prefixConfirmed + entryID)

	return w.db.Update(func(txn *badger.Txn) error {
		// Try to delete pending
		err := txn.Delete(pendingKey)
		if err == nil {
			return nil
		}
		if !errors.Is(err, badger.ErrKeyNotFound) {
			return fmt.Errorf("delete pending entry: %w", err)
		}

		// Try to delete confirmed
		err = txn.Delete(confirmedKey)
		if errors.Is(err, badger.ErrKeyNotFound) {
			return ErrEntryNotFound
		}
		return err
	})
}

// Stats returns current WAL statistics.
func (w *BadgerWAL) Stats() Stats {
	w.mu.RLock()
	closed := w.closed
	lastCompaction := w.lastCompaction
	w.mu.RUnlock()

	if closed {
		return Stats{}
	}

	var pendingCount, confirmedCount int64

	// Count entries by prefix
	if err := w.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()

		// Count pending
		pendingPrefix := []byte(prefixPending)
		for it.Seek(pendingPrefix); it.ValidForPrefix(pendingPrefix); it.Next() {
			pendingCount++
		}

		// Count confirmed
		confirmedPrefix := []byte(prefixConfirmed)
		for it.Seek(confirmedPrefix); it.ValidForPrefix(confirmedPrefix); it.Next() {
			confirmedCount++
		}

		return nil
	}); err != nil {
		logging.Warn().Err(err).Msg("WAL Stats failed to count entries")
		// Continue with zero counts
	}

	// Get DB size
	lsm, vlog := w.db.Size()
	dbSize := lsm + vlog

	stats := Stats{
		PendingCount:   pendingCount,
		ConfirmedCount: confirmedCount,
		TotalWrites:    w.totalWrites.Load(),
		TotalConfirms:  w.totalConfirms.Load(),
		TotalRetries:   w.totalRetries.Load(),
		LastCompaction: lastCompaction,
		DBSizeBytes:    dbSize,
	}

	// Update metrics
	UpdateWALPendingEntries(pendingCount)
	UpdateWALDBSize(dbSize)

	return stats
}

// DB returns the underlying BadgerDB instance.
// This can be used by other components that need access to BadgerDB storage,
// such as import progress tracking. The returned DB should not be closed
// directly; use the WAL's Close method instead.
func (w *BadgerWAL) DB() *badger.DB {
	return w.db
}

// TryClaimEntry attempts to claim exclusive processing rights for an entry.
// Returns true if the claim was successful, false if another goroutine is
// already processing this entry.
//
// RACE CONDITION FIX (v2.4): This prevents concurrent processing of the same entry.
// The caller MUST call ReleaseEntry when done, regardless of success or failure.
//
// Usage:
//
//	if !wal.TryClaimEntry(entryID) {
//	    // Another goroutine is processing this entry, skip it
//	    return
//	}
//	defer wal.ReleaseEntry(entryID)
//	// ... process entry ...
func (w *BadgerWAL) TryClaimEntry(entryID string) bool {
	// LoadOrStore returns (existing value, true) if key exists,
	// or (new value, false) if we stored the new value
	_, alreadyClaimed := w.processingEntries.LoadOrStore(entryID, time.Now())
	if alreadyClaimed {
		logging.Trace().
			Str("entry_id", entryID).
			Msg("WAL: entry already being processed, skipping")
		return false
	}
	return true
}

// ReleaseEntry releases the processing claim on an entry.
// This MUST be called after TryClaimEntry returns true, typically via defer.
//
// RACE CONDITION FIX (v2.4): Releases the lock so other goroutines can process
// the entry if needed (e.g., if this processing attempt failed).
func (w *BadgerWAL) ReleaseEntry(entryID string) {
	w.processingEntries.Delete(entryID)
}

// TryClaimEntryDurable attempts to claim exclusive processing rights for an entry
// using durable BadgerDB storage. This is crash-safe: if the process crashes while
// holding a lease, the lease will naturally expire after LeaseDuration, allowing
// another process to claim it.
//
// Returns:
//   - (true, nil): Lease acquired successfully
//   - (false, nil): Entry is already leased by another processor (not an error)
//   - (false, error): Database error occurred
//
// The leaseHolder parameter identifies this processor for debugging/auditing.
// Use a UUID or unique instance identifier.
func (w *BadgerWAL) TryClaimEntryDurable(ctx context.Context, entryID, leaseHolder string) (bool, error) {
	w.mu.RLock()
	if w.closed {
		w.mu.RUnlock()
		return false, ErrWALClosed
	}
	w.mu.RUnlock()

	now := time.Now()
	leaseExpiry := now.Add(w.config.LeaseDuration)

	var claimed bool
	err := w.db.Update(func(txn *badger.Txn) error {
		key := []byte(prefixPending + entryID)

		// Read current entry
		item, err := txn.Get(key)
		if err != nil {
			if errors.Is(err, badger.ErrKeyNotFound) {
				return ErrEntryNotFound
			}
			return fmt.Errorf("get entry: %w", err)
		}

		var entry Entry
		err = item.Value(func(val []byte) error {
			return json.Unmarshal(val, &entry)
		})
		if err != nil {
			return fmt.Errorf("unmarshal entry: %w", err)
		}

		// Check if already leased by another processor
		if !entry.LeaseExpiry.IsZero() && now.Before(entry.LeaseExpiry) {
			// Lease is still active - check if same processor
			if entry.LeaseHolder == leaseHolder {
				// Same processor can extend/reclaim their own lease
				entry.LeaseExpiry = leaseExpiry
				data, err := json.Marshal(&entry)
				if err != nil {
					return fmt.Errorf("marshal entry: %w", err)
				}
				if err := txn.Set(key, data); err != nil {
					return fmt.Errorf("set entry: %w", err)
				}
				claimed = true
				logging.Debug().
					Str("entry_id", entryID).
					Str("lease_holder", leaseHolder).
					Time("lease_expiry", leaseExpiry).
					Msg("WAL: extended durable lease")
				return nil
			}
			// Different processor, cannot claim
			logging.Trace().
				Str("entry_id", entryID).
				Str("lease_holder", entry.LeaseHolder).
				Time("lease_expiry", entry.LeaseExpiry).
				Msg("WAL: entry has active lease, skipping")
			claimed = false
			return nil
		}

		// Lease is expired or doesn't exist - claim it
		entry.LeaseExpiry = leaseExpiry
		entry.LeaseHolder = leaseHolder

		// Write updated entry
		data, err := json.Marshal(&entry)
		if err != nil {
			return fmt.Errorf("marshal entry: %w", err)
		}

		if err := txn.Set(key, data); err != nil {
			return fmt.Errorf("set entry: %w", err)
		}

		claimed = true
		logging.Debug().
			Str("entry_id", entryID).
			Str("lease_holder", leaseHolder).
			Time("lease_expiry", leaseExpiry).
			Msg("WAL: acquired durable lease")
		return nil
	})

	if err != nil {
		return false, err
	}

	return claimed, nil
}

// ReleaseLeaseDurable explicitly releases a durable lease.
// This is optional - the lease will naturally expire after LeaseDuration.
// Calling this allows other processors to immediately claim the entry.
func (w *BadgerWAL) ReleaseLeaseDurable(ctx context.Context, entryID string) error {
	w.mu.RLock()
	if w.closed {
		w.mu.RUnlock()
		return ErrWALClosed
	}
	w.mu.RUnlock()

	return w.db.Update(func(txn *badger.Txn) error {
		key := []byte(prefixPending + entryID)

		// Read current entry
		item, err := txn.Get(key)
		if err != nil {
			if errors.Is(err, badger.ErrKeyNotFound) {
				return nil // Entry doesn't exist - nothing to release
			}
			return fmt.Errorf("get entry: %w", err)
		}

		var entry Entry
		err = item.Value(func(val []byte) error {
			return json.Unmarshal(val, &entry)
		})
		if err != nil {
			return fmt.Errorf("unmarshal entry: %w", err)
		}

		// Clear the lease
		entry.LeaseExpiry = time.Time{}
		entry.LeaseHolder = ""

		// Write updated entry
		data, err := json.Marshal(&entry)
		if err != nil {
			return fmt.Errorf("marshal entry: %w", err)
		}

		if err := txn.Set(key, data); err != nil {
			return fmt.Errorf("set entry: %w", err)
		}

		logging.Debug().
			Str("entry_id", entryID).
			Msg("WAL: released durable lease")
		return nil
	})
}

// GetProcessingCount returns the number of entries currently being processed.
// This is useful for monitoring and debugging race condition issues.
func (w *BadgerWAL) GetProcessingCount() int {
	count := 0
	w.processingEntries.Range(func(_, _ interface{}) bool {
		count++
		return true
	})
	return count
}

// Close gracefully shuts down the WAL with a configurable timeout.
// If the database doesn't close within the configured CloseTimeout,
// Close() returns with an error to prevent indefinite hangs.
func (w *BadgerWAL) Close() error {
	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return nil
	}
	w.closed = true
	timeout := w.config.CloseTimeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	w.mu.Unlock()

	logging.Info().Msg("Closing WAL")

	// Use a channel to implement timeout
	done := make(chan error, 1)
	go func() {
		done <- w.db.Close()
	}()

	select {
	case err := <-done:
		if err != nil {
			return fmt.Errorf("close BadgerDB: %w", err)
		}
		logging.Info().Msg("WAL closed")
		return nil
	case <-time.After(timeout):
		logging.Warn().Dur("timeout", timeout).Msg("BadgerDB close timed out")
		return fmt.Errorf("badgerdb close timeout after %v", timeout)
	}
}

// GetConfig returns the WAL configuration.
func (w *BadgerWAL) GetConfig() Config {
	return w.config
}

// SetMetricsCallback sets a callback for periodic metrics updates.
func (w *BadgerWAL) SetMetricsCallback(cb func(Stats)) {
	w.mu.Lock()
	w.metricsCallback = cb
	w.mu.Unlock()
}

// RunGC triggers BadgerDB garbage collection.
// This should be called periodically to reclaim space.
func (w *BadgerWAL) RunGC() error {
	w.mu.RLock()
	if w.closed {
		w.mu.RUnlock()
		return ErrWALClosed
	}
	w.mu.RUnlock()

	start := time.Now()
	defer func() {
		RecordWALGCLatency(time.Since(start).Seconds())
		RecordWALGCRun()
	}()

	// Run GC until no more cleanup is possible
	for {
		err := w.db.RunValueLogGC(w.config.GCRatio)
		if errors.Is(err, badger.ErrNoRewrite) {
			break
		}
		if err != nil {
			return fmt.Errorf("run GC: %w", err)
		}
	}

	return nil
}

// Errors
var (
	// ErrWALClosed is returned when the WAL is closed.
	ErrWALClosed = fmt.Errorf("WAL is closed")

	// ErrNilEvent is returned when a nil event is passed to Write.
	ErrNilEvent = fmt.Errorf("event cannot be nil")

	// ErrEmptyEntryID is returned when an empty entry ID is provided.
	ErrEmptyEntryID = fmt.Errorf("entry ID cannot be empty")

	// ErrEntryNotFound is returned when an entry doesn't exist.
	ErrEntryNotFound = fmt.Errorf("entry not found")
)
