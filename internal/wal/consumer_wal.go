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

// transactionCounter is a global atomic counter for generating deterministic transaction IDs.
// This replaces time.Now().UnixNano() which was non-deterministic and could cause:
// 1. Collisions when two events are processed in the same nanosecond
// 2. Non-reproducible deduplication behavior across restarts
// 3. Ordering inconsistencies in the WAL
//
// The counter is initialized to 0 and monotonically increases. Combined with
// source and eventID, this guarantees unique, deterministic transaction IDs.
var transactionCounter atomic.Uint64

// ConsumerWAL provides durable write-ahead logging between NATS and DuckDB.
// This ensures exactly-once delivery semantics by:
// 1. Writing events to WAL before ACKing NATS
// 2. Confirming WAL entries only after successful DuckDB insert
// 3. Recovering pending entries on startup
//
// Unlike the producer WAL (which ensures events reach NATS), the ConsumerWAL
// ensures events survive the gap between NATS consumption and DuckDB persistence.
//
// RACE CONDITION PREVENTION (v2.4):
// The processingEntries map prevents concurrent processing of the same entry.
// This fixes the race condition where multiple goroutines (recovery, retry loop,
// normal operations) could process the same pending entry simultaneously, causing:
// - Duplicate key violations in DuckDB
// - ErrEntryNotFound when confirming (entry already confirmed by another goroutine)
type ConsumerWAL struct {
	db     *badger.DB
	config ConsumerWALConfig

	// Statistics
	totalWrites        atomic.Int64
	totalConfirms      atomic.Int64
	totalRetries       atomic.Int64
	totalFailures      atomic.Int64
	totalRecovered     atomic.Int64
	pendingCount       atomic.Int64
	lastRecoveryTime   atomic.Value // stores time.Time
	lastRecoveryResult atomic.Value // stores *RecoveryResult

	// State tracking
	mu     sync.RWMutex
	closed bool

	// RACE CONDITION FIX (v2.4): Tracks entries currently being processed.
	// Key: entry ID (string), Value: time.Time when claim was acquired.
	// This prevents multiple goroutines from processing the same entry.
	processingEntries sync.Map
}

// ConsumerWALConfig holds configuration for the consumer WAL.
type ConsumerWALConfig struct {
	// Path is the directory where BadgerDB stores its files.
	// Should be separate from the producer WAL path.
	Path string

	// SyncWrites forces fsync after every write for maximum durability.
	SyncWrites bool

	// EntryTTL is the time-to-live for unconfirmed entries.
	EntryTTL time.Duration

	// MaxRetries is the maximum DuckDB insert attempts before moving to failed_events.
	MaxRetries int

	// RetryInterval is the time between retry loop iterations.
	RetryInterval time.Duration

	// RetryBackoff is the initial backoff duration for exponential backoff.
	RetryBackoff time.Duration

	// MemTableSize is the size of each memtable in bytes.
	MemTableSize int64

	// ValueLogFileSize is the size of each value log file in bytes.
	ValueLogFileSize int64

	// NumCompactors is the number of compaction workers.
	NumCompactors int

	// Compression enables Snappy compression for WAL entries.
	Compression bool

	// CloseTimeout is the maximum time to wait for graceful shutdown.
	CloseTimeout time.Duration

	// LeaseDuration is how long a processing lease is held before expiring.
	// This enables durable leasing to prevent concurrent processing of the same entry.
	// When an entry is claimed, a lease expiry is set to now + LeaseDuration.
	// If the process crashes, the lease will naturally expire, allowing recovery.
	// Default: 2 minutes (should be longer than expected processing time)
	LeaseDuration time.Duration
}

// DefaultConsumerWALConfig returns sensible defaults for the consumer WAL.
func DefaultConsumerWALConfig() ConsumerWALConfig {
	return ConsumerWALConfig{
		Path:             "/data/wal-consumer",
		SyncWrites:       true,
		EntryTTL:         168 * time.Hour, // 7 days
		MaxRetries:       100,
		RetryInterval:    30 * time.Second,
		RetryBackoff:     5 * time.Second,
		MemTableSize:     16 * 1024 * 1024, // 16MB
		ValueLogFileSize: 64 * 1024 * 1024, // 64MB
		NumCompactors:    2,
		Compression:      true,
		CloseTimeout:     30 * time.Second,
		LeaseDuration:    2 * time.Minute, // Durable lease for concurrent processing prevention
	}
}

// LoadConsumerWALConfig loads consumer WAL configuration from environment variables.
func LoadConsumerWALConfig() ConsumerWALConfig {
	defaults := DefaultConsumerWALConfig()

	return ConsumerWALConfig{
		Path:             getEnv("CONSUMER_WAL_PATH", defaults.Path),
		SyncWrites:       getEnvBool("CONSUMER_WAL_SYNC_WRITES", defaults.SyncWrites),
		EntryTTL:         getEnvDuration("CONSUMER_WAL_ENTRY_TTL", defaults.EntryTTL),
		MaxRetries:       getEnvInt("CONSUMER_WAL_MAX_RETRIES", defaults.MaxRetries),
		RetryInterval:    getEnvDuration("CONSUMER_WAL_RETRY_INTERVAL", defaults.RetryInterval),
		RetryBackoff:     getEnvDuration("CONSUMER_WAL_RETRY_BACKOFF", defaults.RetryBackoff),
		MemTableSize:     getEnvInt64("CONSUMER_WAL_MEMTABLE_SIZE", defaults.MemTableSize),
		ValueLogFileSize: getEnvInt64("CONSUMER_WAL_VLOG_SIZE", defaults.ValueLogFileSize),
		NumCompactors:    getEnvInt("CONSUMER_WAL_NUM_COMPACTORS", defaults.NumCompactors),
		Compression:      getEnvBool("CONSUMER_WAL_COMPRESSION", defaults.Compression),
		CloseTimeout:     getEnvDuration("CONSUMER_WAL_CLOSE_TIMEOUT", defaults.CloseTimeout),
		LeaseDuration:    getEnvDuration("CONSUMER_WAL_LEASE_DURATION", defaults.LeaseDuration),
	}
}

// Validate checks that the consumer WAL configuration is valid.
func (c *ConsumerWALConfig) Validate() error {
	if c.Path == "" {
		return &ConfigError{Field: "Path", Message: "consumer WAL path is required"}
	}
	if c.MaxRetries < 1 {
		return &ConfigError{Field: "MaxRetries", Message: "must be at least 1"}
	}
	if c.NumCompactors < 2 {
		return &ConfigError{Field: "NumCompactors", Message: "must be at least 2 (BadgerDB requirement)"}
	}
	if c.LeaseDuration < 30*time.Second {
		return &ConfigError{Field: "LeaseDuration", Message: "must be at least 30 seconds"}
	}
	return nil
}

// ConsumerWALEntry represents a single entry in the consumer WAL.
type ConsumerWALEntry struct {
	// ID is the unique identifier for this WAL entry.
	ID string `json:"id"`

	// TransactionID is the idempotency key for DuckDB.
	// Format: {source}:{event_id}:{timestamp_nano}
	TransactionID string `json:"transaction_id"`

	// NATSSubject is the original NATS subject for correlation.
	NATSSubject string `json:"nats_subject,omitempty"`

	// NATSMessageID is the NATS message ID for correlation.
	NATSMessageID string `json:"nats_message_id,omitempty"`

	// EventPayload is the serialized MediaEvent.
	EventPayload json.RawMessage `json:"event_payload"`

	// CreatedAt is when the entry was written to the WAL.
	CreatedAt time.Time `json:"created_at"`

	// Attempts is the number of DuckDB insert attempts.
	Attempts int `json:"attempts"`

	// LastAttemptAt is the time of the last insert attempt.
	LastAttemptAt time.Time `json:"last_attempt_at,omitempty"`

	// LastError is the error message from the last failed attempt.
	LastError string `json:"last_error,omitempty"`

	// Confirmed indicates successful DuckDB insert.
	Confirmed bool `json:"confirmed"`

	// ConfirmedAt is when the entry was confirmed.
	ConfirmedAt *time.Time `json:"confirmed_at,omitempty"`

	// FailedPermanent indicates the entry was moved to failed_events.
	FailedPermanent bool `json:"failed_permanent"`

	// FailedAt is when the entry was marked as permanently failed.
	FailedAt *time.Time `json:"failed_at,omitempty"`

	// FailureReason is the reason for permanent failure.
	FailureReason string `json:"failure_reason,omitempty"`

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

// ConsumerRecoveryResult contains statistics from consumer WAL startup recovery.
type ConsumerRecoveryResult struct {
	// TotalPending is the number of pending entries found.
	TotalPending int

	// AlreadyCommitted is the number of entries already in DuckDB.
	AlreadyCommitted int

	// Recovered is the number of entries successfully recovered.
	Recovered int

	// Expired is the number of entries that exceeded TTL.
	Expired int

	// Failed is the number of entries that failed to recover.
	Failed int

	// Skipped is the number of entries skipped because they were already
	// being processed by another goroutine (RACE CONDITION FIX v2.4).
	Skipped int

	// Duration is how long recovery took.
	Duration time.Duration
}

// ConsumerWALStats contains consumer WAL metrics for monitoring.
type ConsumerWALStats struct {
	PendingCount   int64
	TotalWrites    int64
	TotalConfirms  int64
	TotalRetries   int64
	TotalFailures  int64
	TotalRecovered int64
	LastRecovery   time.Time
	RecoveryResult *ConsumerRecoveryResult
	DBSizeBytes    int64
}

// Consumer WAL key prefixes
const (
	consumerPrefixPending   = "consumer:pending:"
	consumerPrefixConfirmed = "consumer:confirmed:"
	consumerPrefixFailed    = "consumer:failed:"
)

// OpenConsumerWAL creates a new ConsumerWAL with the given configuration.
func OpenConsumerWAL(cfg *ConsumerWALConfig) (*ConsumerWAL, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid consumer WAL config: %w", err)
	}

	opts := badger.DefaultOptions(cfg.Path)
	opts.SyncWrites = cfg.SyncWrites
	opts.MemTableSize = cfg.MemTableSize
	opts.ValueLogFileSize = cfg.ValueLogFileSize
	opts.NumCompactors = cfg.NumCompactors

	if cfg.Compression {
		opts.Compression = options.Snappy
	}

	// Reduce logging verbosity
	opts.Logger = nil

	db, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("open BadgerDB: %w", err)
	}

	wal := &ConsumerWAL{
		db:     db,
		config: *cfg,
	}

	wal.lastRecoveryTime.Store(time.Time{})
	wal.lastRecoveryResult.Store((*ConsumerRecoveryResult)(nil))

	logging.Info().
		Str("path", cfg.Path).
		Bool("sync_writes", cfg.SyncWrites).
		Int("max_retries", cfg.MaxRetries).
		Msg("Consumer WAL opened")
	return wal, nil
}

// OpenConsumerWALForTesting opens a Consumer WAL without config validation.
// This allows tests to use short intervals that wouldn't be valid in production.
// DO NOT USE IN PRODUCTION CODE.
func OpenConsumerWALForTesting(cfg *ConsumerWALConfig) (*ConsumerWAL, error) {
	opts := badger.DefaultOptions(cfg.Path)
	opts.SyncWrites = cfg.SyncWrites
	opts.MemTableSize = cfg.MemTableSize
	opts.ValueLogFileSize = cfg.ValueLogFileSize
	opts.NumCompactors = cfg.NumCompactors

	if cfg.Compression {
		opts.Compression = options.Snappy
	}

	// Reduce logging verbosity
	opts.Logger = nil

	db, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("open BadgerDB: %w", err)
	}

	wal := &ConsumerWAL{
		db:     db,
		config: *cfg,
	}

	wal.lastRecoveryTime.Store(time.Time{})
	wal.lastRecoveryResult.Store((*ConsumerRecoveryResult)(nil))

	logging.Info().
		Str("path", cfg.Path).
		Bool("sync_writes", cfg.SyncWrites).
		Int("max_retries", cfg.MaxRetries).
		Msg("Consumer WAL opened (for testing)")
	return wal, nil
}

// GenerateTransactionID creates a unique, deterministic transaction ID for idempotency.
// Format: {source}:{event_id}:{sequence_number}
//
// DETERMINISM GUARANTEE: Uses an atomic counter instead of time.Now().UnixNano() to ensure:
// - Unique IDs even when multiple events are processed simultaneously
// - Reproducible deduplication behavior for testing and replay scenarios
// - Monotonically increasing sequence for ordering guarantees
// - No dependency on wall clock time which can drift or be non-monotonic
//
// The sequence number is process-local and resets on restart. Combined with
// source and eventID, this provides sufficient uniqueness for deduplication.
func GenerateTransactionID(source, eventID string) string {
	seq := transactionCounter.Add(1)
	return fmt.Sprintf("%s:%s:%d", source, eventID, seq)
}

// ResetTransactionCounter resets the transaction counter to zero.
// This should ONLY be used in tests to ensure deterministic test runs.
// Using this in production would break transaction ID uniqueness guarantees.
func ResetTransactionCounter() {
	transactionCounter.Store(0)
}

// GetTransactionCounter returns the current transaction counter value.
// This is useful for monitoring and debugging.
func GetTransactionCounter() uint64 {
	return transactionCounter.Load()
}

// Write persists an event to the consumer WAL before ACKing NATS.
// This operation is ACID-compliant with fsync when SyncWrites is enabled.
// Returns the entry ID and transaction ID for later confirmation.
func (w *ConsumerWAL) Write(ctx context.Context, eventPayload json.RawMessage, transactionID, natsSubject, natsMessageID string) (entryID string, err error) {
	w.mu.RLock()
	if w.closed {
		w.mu.RUnlock()
		return "", ErrWALClosed
	}
	w.mu.RUnlock()

	if eventPayload == nil {
		return "", ErrNilEvent
	}
	if transactionID == "" {
		return "", fmt.Errorf("transaction ID is required")
	}

	entryID = uuid.New().String()

	entry := &ConsumerWALEntry{
		ID:            entryID,
		TransactionID: transactionID,
		NATSSubject:   natsSubject,
		NATSMessageID: natsMessageID,
		EventPayload:  eventPayload,
		CreatedAt:     time.Now().UTC(),
		Attempts:      0,
		Confirmed:     false,
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return "", fmt.Errorf("marshal entry: %w", err)
	}

	key := []byte(consumerPrefixPending + entryID)
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
	w.pendingCount.Add(1)
	RecordConsumerWALWrite()

	return entryID, nil
}

// Confirm marks an entry as successfully inserted into DuckDB.
// The entry is moved from pending to confirmed state.
func (w *ConsumerWAL) Confirm(ctx context.Context, entryID string) error {
	if err := w.checkNotClosed(); err != nil {
		return err
	}
	if entryID == "" {
		return ErrEmptyEntryID
	}

	pendingKey := []byte(consumerPrefixPending + entryID)
	confirmedKey := []byte(consumerPrefixConfirmed + entryID)

	err := w.db.Update(func(txn *badger.Txn) error {
		entry, err := w.readEntryFromTxn(txn, pendingKey)
		if err != nil {
			return err
		}

		// Update to confirmed
		now := time.Now().UTC()
		entry.Confirmed = true
		entry.ConfirmedAt = &now

		if err := w.writeEntryToTxn(txn, confirmedKey, entry); err != nil {
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
	w.pendingCount.Add(-1)
	RecordConsumerWALConfirm()

	return nil
}

// MarkFailed marks an entry as permanently failed.
// This is called when max retries are exceeded.
func (w *ConsumerWAL) MarkFailed(ctx context.Context, entryID, failureReason string) error {
	if err := w.checkNotClosed(); err != nil {
		return err
	}
	if entryID == "" {
		return ErrEmptyEntryID
	}

	pendingKey := []byte(consumerPrefixPending + entryID)
	failedKey := []byte(consumerPrefixFailed + entryID)

	err := w.db.Update(func(txn *badger.Txn) error {
		entry, err := w.readEntryFromTxn(txn, pendingKey)
		if err != nil {
			return err
		}

		// Update to failed
		now := time.Now().UTC()
		entry.FailedPermanent = true
		entry.FailedAt = &now
		entry.FailureReason = failureReason

		if err := w.writeEntryToTxn(txn, failedKey, entry); err != nil {
			return fmt.Errorf("set failed entry: %w", err)
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

	w.totalFailures.Add(1)
	w.pendingCount.Add(-1)
	RecordConsumerWALFailure()

	return nil
}

// UpdateAttempt updates an entry's attempt count and last error.
func (w *ConsumerWAL) UpdateAttempt(ctx context.Context, entryID, lastError string) error {
	if err := w.checkNotClosed(); err != nil {
		return err
	}

	key := []byte(consumerPrefixPending + entryID)

	err := w.db.Update(func(txn *badger.Txn) error {
		entry, err := w.readEntryFromTxn(txn, key)
		if err != nil {
			return err
		}

		entry.Attempts++
		entry.LastAttemptAt = time.Now().UTC()
		entry.LastError = lastError

		return w.writeEntryToTxn(txn, key, entry)
	})

	if err != nil {
		return err
	}

	w.totalRetries.Add(1)
	RecordConsumerWALRetry()

	return nil
}

// GetPending returns all unconfirmed entries for retry or recovery.
func (w *ConsumerWAL) GetPending(ctx context.Context) ([]*ConsumerWALEntry, error) {
	if err := w.checkNotClosed(); err != nil {
		return nil, err
	}

	var entries []*ConsumerWALEntry

	err := w.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = true
		it := txn.NewIterator(opts)
		defer it.Close()

		prefix := []byte(consumerPrefixPending)
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			item := it.Item()
			var entry ConsumerWALEntry
			err := item.Value(func(val []byte) error {
				return json.Unmarshal(val, &entry)
			})
			if err != nil {
				logging.Warn().Err(err).Str("key", string(item.Key())).Msg("Consumer WAL failed to unmarshal entry")
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

// GetEntry returns a specific entry by ID.
func (w *ConsumerWAL) GetEntry(ctx context.Context, entryID string) (*ConsumerWALEntry, error) {
	if err := w.checkNotClosed(); err != nil {
		return nil, err
	}

	var entry *ConsumerWALEntry
	key := []byte(consumerPrefixPending + entryID)

	err := w.db.View(func(txn *badger.Txn) error {
		var err error
		entry, err = w.readEntryFromTxn(txn, key)
		return err
	})

	if err != nil {
		return nil, err
	}

	return entry, nil
}

// Stats returns current consumer WAL statistics.
func (w *ConsumerWAL) Stats() ConsumerWALStats {
	w.mu.RLock()
	closed := w.closed
	w.mu.RUnlock()

	if closed {
		return ConsumerWALStats{}
	}

	// Count pending entries
	var pendingCount int64
	if err := w.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()

		prefix := []byte(consumerPrefixPending)
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			pendingCount++
		}
		return nil
	}); err != nil {
		logging.Warn().Err(err).Msg("Consumer WAL Stats failed to count entries")
	}

	// Get DB size
	lsm, vlog := w.db.Size()
	dbSize := lsm + vlog

	var lastRecovery time.Time
	if t, ok := w.lastRecoveryTime.Load().(time.Time); ok {
		lastRecovery = t
	}

	var recoveryResult *ConsumerRecoveryResult
	if r, ok := w.lastRecoveryResult.Load().(*ConsumerRecoveryResult); ok {
		recoveryResult = r
	}

	return ConsumerWALStats{
		PendingCount:   pendingCount,
		TotalWrites:    w.totalWrites.Load(),
		TotalConfirms:  w.totalConfirms.Load(),
		TotalRetries:   w.totalRetries.Load(),
		TotalFailures:  w.totalFailures.Load(),
		TotalRecovered: w.totalRecovered.Load(),
		LastRecovery:   lastRecovery,
		RecoveryResult: recoveryResult,
		DBSizeBytes:    dbSize,
	}
}

// Config returns the consumer WAL configuration.
func (w *ConsumerWAL) Config() ConsumerWALConfig {
	return w.config
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
func (w *ConsumerWAL) TryClaimEntry(entryID string) bool {
	// LoadOrStore returns (existing value, true) if key exists,
	// or (new value, false) if we stored the new value
	_, alreadyClaimed := w.processingEntries.LoadOrStore(entryID, time.Now())
	if alreadyClaimed {
		logging.Trace().
			Str("entry_id", entryID).
			Msg("Consumer WAL: entry already being processed, skipping")
		return false
	}
	return true
}

// ReleaseEntry releases the processing claim on an entry.
// This MUST be called after TryClaimEntry returns true, typically via defer.
//
// RACE CONDITION FIX (v2.4): Releases the lock so other goroutines can process
// the entry if needed (e.g., if this processing attempt failed).
func (w *ConsumerWAL) ReleaseEntry(entryID string) {
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
//
// Usage:
//
//	claimed, err := wal.TryClaimEntryDurable(ctx, entryID, "worker-1")
//	if err != nil {
//	    return err // Database error
//	}
//	if !claimed {
//	    return nil // Another processor has the lease, skip this entry
//	}
//	defer wal.ReleaseLeaseDurable(ctx, entryID)
//	// ... process entry ...
func (w *ConsumerWAL) TryClaimEntryDurable(ctx context.Context, entryID, leaseHolder string) (bool, error) {
	if err := w.checkNotClosed(); err != nil {
		return false, err
	}

	now := time.Now()
	leaseExpiry := now.Add(w.config.LeaseDuration)

	var claimed bool
	err := w.db.Update(func(txn *badger.Txn) error {
		key := []byte(consumerPrefixPending + entryID)

		entry, err := w.readEntryFromTxn(txn, key)
		if err != nil {
			return err
		}

		// Check if already leased by another processor
		if !entry.LeaseExpiry.IsZero() && now.Before(entry.LeaseExpiry) {
			// Lease is still active - check if same processor
			if entry.LeaseHolder == leaseHolder {
				// Same processor can extend/reclaim their own lease
				entry.LeaseExpiry = leaseExpiry
				if err := w.writeEntryToTxn(txn, key, entry); err != nil {
					return err
				}
				claimed = true
				logging.Debug().
					Str("entry_id", entryID).
					Str("lease_holder", leaseHolder).
					Time("lease_expiry", leaseExpiry).
					Msg("Consumer WAL: extended durable lease")
				return nil
			}
			// Different processor, cannot claim
			logging.Trace().
				Str("entry_id", entryID).
				Str("lease_holder", entry.LeaseHolder).
				Time("lease_expiry", entry.LeaseExpiry).
				Msg("Consumer WAL: entry has active lease, skipping")
			claimed = false
			return nil
		}

		// Lease is expired or doesn't exist - claim it
		entry.LeaseExpiry = leaseExpiry
		entry.LeaseHolder = leaseHolder

		if err := w.writeEntryToTxn(txn, key, entry); err != nil {
			return err
		}

		claimed = true
		logging.Debug().
			Str("entry_id", entryID).
			Str("lease_holder", leaseHolder).
			Time("lease_expiry", leaseExpiry).
			Msg("Consumer WAL: acquired durable lease")
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
//
// This is typically called when processing fails and you want to allow
// immediate retry by another processor.
func (w *ConsumerWAL) ReleaseLeaseDurable(ctx context.Context, entryID string) error {
	if err := w.checkNotClosed(); err != nil {
		return err
	}

	return w.db.Update(func(txn *badger.Txn) error {
		key := []byte(consumerPrefixPending + entryID)

		entry, err := w.readEntryFromTxn(txn, key)
		if errors.Is(err, ErrEntryNotFound) {
			// Entry doesn't exist - nothing to release
			return nil
		}
		if err != nil {
			return err
		}

		// Clear the lease
		entry.LeaseExpiry = time.Time{}
		entry.LeaseHolder = ""

		if err := w.writeEntryToTxn(txn, key, entry); err != nil {
			return err
		}

		logging.Debug().
			Str("entry_id", entryID).
			Msg("Consumer WAL: released durable lease")
		return nil
	})
}

// ExtendLease extends an existing lease by LeaseDuration from now.
// This is useful for long-running operations that may exceed the initial lease duration.
// The caller must already hold the lease (same leaseHolder).
//
// Returns error if:
//   - Entry doesn't exist
//   - Lease is held by a different processor
//   - Database error
func (w *ConsumerWAL) ExtendLease(ctx context.Context, entryID, leaseHolder string) error {
	if err := w.checkNotClosed(); err != nil {
		return err
	}

	newExpiry := time.Now().Add(w.config.LeaseDuration)

	return w.db.Update(func(txn *badger.Txn) error {
		key := []byte(consumerPrefixPending + entryID)

		entry, err := w.readEntryFromTxn(txn, key)
		if err != nil {
			return err
		}

		// Verify we hold the lease
		if entry.LeaseHolder != leaseHolder {
			return fmt.Errorf("lease held by different processor: %s", entry.LeaseHolder)
		}

		// Extend the lease
		entry.LeaseExpiry = newExpiry

		if err := w.writeEntryToTxn(txn, key, entry); err != nil {
			return err
		}

		logging.Debug().
			Str("entry_id", entryID).
			Time("new_expiry", newExpiry).
			Msg("Consumer WAL: extended lease")
		return nil
	})
}

// GetProcessingCount returns the number of entries currently being processed.
// This is useful for monitoring and debugging race condition issues.
func (w *ConsumerWAL) GetProcessingCount() int {
	count := 0
	w.processingEntries.Range(func(_, _ interface{}) bool {
		count++
		return true
	})
	return count
}

// Close gracefully shuts down the consumer WAL.
func (w *ConsumerWAL) Close() error {
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

	logging.Info().Msg("Closing Consumer WAL")

	done := make(chan error, 1)
	go func() {
		done <- w.db.Close()
	}()

	select {
	case err := <-done:
		if err != nil {
			return fmt.Errorf("close BadgerDB: %w", err)
		}
		logging.Info().Msg("Consumer WAL closed")
		return nil
	case <-time.After(timeout):
		logging.Warn().Dur("timeout", timeout).Msg("Consumer WAL close timed out")
		return fmt.Errorf("consumer WAL close timeout after %v", timeout)
	}
}

// RunGC triggers BadgerDB garbage collection.
func (w *ConsumerWAL) RunGC() error {
	if err := w.checkNotClosed(); err != nil {
		return err
	}

	for {
		err := w.db.RunValueLogGC(0.5)
		if errors.Is(err, badger.ErrNoRewrite) {
			break
		}
		if err != nil {
			return fmt.Errorf("run GC: %w", err)
		}
	}

	return nil
}

// CleanupConfirmed removes all confirmed entries from the WAL.
// This is called periodically by the compaction loop.
func (w *ConsumerWAL) CleanupConfirmed(ctx context.Context) (int, error) {
	if err := w.checkNotClosed(); err != nil {
		return 0, err
	}

	var keysToDelete [][]byte

	// Collect keys to delete
	err := w.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()

		prefix := []byte(consumerPrefixConfirmed)
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
			key := it.Item().KeyCopy(nil)
			keysToDelete = append(keysToDelete, key)
		}
		return nil
	})

	if err != nil {
		return 0, fmt.Errorf("collect confirmed entries: %w", err)
	}

	if len(keysToDelete) == 0 {
		return 0, nil
	}

	// Delete in batches
	return w.deleteBatchedKeys(keysToDelete)
}

// RecoveryCallback provides the database operations needed for crash recovery.
// This decouples the Consumer WAL from the database implementation.
type RecoveryCallback interface {
	// TransactionIDExists checks if a transaction ID is already in DuckDB.
	// Used to detect events that were committed but not confirmed in WAL.
	TransactionIDExists(ctx context.Context, transactionID string) (bool, error)

	// InsertEvent inserts the event payload into DuckDB with the transaction ID.
	// The payload is the raw MediaEvent JSON.
	InsertEvent(ctx context.Context, payload []byte, transactionID string) error

	// InsertFailedEvent moves an event to the failed_events table.
	// Called when max retries are exceeded.
	InsertFailedEvent(ctx context.Context, entry *ConsumerWALEntry, reason string) error
}

// RecoverOnStartup processes all pending entries from a previous run.
// This is called during application startup to ensure no events are lost.
//
// For each pending entry:
// 1. Check if already in DuckDB (crash after insert, before confirm)
// 2. If found: Confirm WAL entry (skip duplicate)
// 3. If expired: Move to failed_events table
// 4. If not found: Retry DuckDB insert
//
// Returns statistics about the recovery process.
func (w *ConsumerWAL) RecoverOnStartup(ctx context.Context, callback RecoveryCallback) (*ConsumerRecoveryResult, error) {
	if callback == nil {
		return nil, fmt.Errorf("recovery callback is required")
	}

	start := time.Now()
	result := &ConsumerRecoveryResult{}

	entries, err := w.GetPending(ctx)
	if err != nil {
		return nil, fmt.Errorf("get pending entries: %w", err)
	}

	result.TotalPending = len(entries)
	if result.TotalPending == 0 {
		logging.Info().Msg("Consumer WAL recovery: no pending entries")
		result.Duration = time.Since(start)
		w.lastRecoveryTime.Store(time.Now())
		w.lastRecoveryResult.Store(result)
		return result, nil
	}

	logging.Info().Int("pending_entries", result.TotalPending).Msg("Consumer WAL recovery found pending entries")

	for _, entry := range entries {
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		default:
		}

		w.recoverEntry(ctx, entry, callback, result)
	}

	result.Duration = time.Since(start)
	w.lastRecoveryTime.Store(time.Now())
	w.lastRecoveryResult.Store(result)
	w.totalRecovered.Add(int64(result.Recovered))
	RecordConsumerWALRecovery(int64(result.Recovered))

	logging.Info().
		Int("recovered", result.Recovered).
		Int("already_committed", result.AlreadyCommitted).
		Int("expired", result.Expired).
		Int("failed", result.Failed).
		Int("skipped", result.Skipped).
		Dur("duration", result.Duration).
		Msg("Consumer WAL recovery complete")

	return result, nil
}

// recoverEntry processes a single pending entry during recovery.
//
// DURABLE LEASING: Uses TryClaimEntryDurable for crash-safe concurrent processing prevention.
// If another goroutine is already processing this entry, we skip it.
// If this process crashes while holding the lease, it will automatically expire
// after LeaseDuration, allowing another process to claim and process the entry.
func (w *ConsumerWAL) recoverEntry(ctx context.Context, entry *ConsumerWALEntry, callback RecoveryCallback, result *ConsumerRecoveryResult) {
	// DURABLE LEASING: Try to claim exclusive processing rights using BadgerDB
	// Generate a unique lease holder for this recovery operation
	leaseHolder := fmt.Sprintf("recovery-%s", uuid.New().String()[:8])
	claimed, err := w.TryClaimEntryDurable(ctx, entry.ID, leaseHolder)
	if err != nil {
		logging.Error().Err(err).Str("entry_id", entry.ID).Msg("Consumer WAL recovery: error claiming entry")
		result.Failed++
		return
	}
	if !claimed {
		// Another processor has the lease, skip this entry
		result.Skipped++
		return
	}
	// Note: We don't need defer release - Confirm/MarkFailed removes entry, or lease expires naturally

	// 1. Check if already in DuckDB (crash after insert, before confirm)
	exists, err := callback.TransactionIDExists(ctx, entry.TransactionID)
	if err != nil {
		logging.Error().Err(err).Str("transaction_id", entry.TransactionID).Msg("Consumer WAL recovery: error checking transaction")
		result.Failed++
		return
	}

	if exists {
		// Already committed to DuckDB, just confirm the WAL entry
		if err := w.Confirm(ctx, entry.ID); err != nil {
			// RACE CONDITION FIX: ErrEntryNotFound means another goroutine already confirmed it
			if errors.Is(err, ErrEntryNotFound) {
				logging.Debug().Str("entry_id", entry.ID).Msg("Consumer WAL recovery: entry already confirmed by another goroutine")
			} else {
				logging.Error().Err(err).Str("entry_id", entry.ID).Msg("Consumer WAL recovery: error confirming")
			}
		}
		logging.Info().Str("entry_id", entry.ID).Msg("Consumer WAL recovery: entry already in DuckDB, confirmed")
		result.AlreadyCommitted++
		return
	}

	// 2. Check expiration
	if time.Since(entry.CreatedAt) > w.config.EntryTTL {
		w.handleExpiredEntry(ctx, entry, callback, result)
		return
	}

	// 3. Check max retries
	if entry.Attempts >= w.config.MaxRetries {
		w.handleMaxRetriesEntry(ctx, entry, callback, result)
		return
	}

	// 4. Retry DuckDB insert
	w.retryEntryInsert(ctx, entry, callback, result)
}

// handleExpiredEntry moves an expired entry to the failed_events table.
func (w *ConsumerWAL) handleExpiredEntry(ctx context.Context, entry *ConsumerWALEntry, callback RecoveryCallback, result *ConsumerRecoveryResult) {
	if err := callback.InsertFailedEvent(ctx, entry, "expired"); err != nil {
		logging.Error().Err(err).Str("entry_id", entry.ID).Msg("Consumer WAL recovery: error moving expired entry to failed")
	}
	if err := w.MarkFailed(ctx, entry.ID, "expired after "+w.config.EntryTTL.String()); err != nil {
		logging.Error().Err(err).Str("entry_id", entry.ID).Msg("Consumer WAL recovery: error marking failed")
	}
	result.Expired++
}

// handleMaxRetriesEntry moves an entry that exceeded max retries to failed_events.
func (w *ConsumerWAL) handleMaxRetriesEntry(ctx context.Context, entry *ConsumerWALEntry, callback RecoveryCallback, result *ConsumerRecoveryResult) {
	if err := callback.InsertFailedEvent(ctx, entry, "max_retries_exceeded"); err != nil {
		logging.Error().Err(err).Str("entry_id", entry.ID).Msg("Consumer WAL recovery: error moving retry-exceeded entry to failed")
	}
	if err := w.MarkFailed(ctx, entry.ID, "exceeded max retries"); err != nil {
		logging.Error().Err(err).Str("entry_id", entry.ID).Msg("Consumer WAL recovery: error marking failed")
	}
	result.Failed++
}

// retryEntryInsert attempts to insert an entry into DuckDB.
//
// RACE CONDITION FIX (v2.4): Handles ErrEntryNotFound gracefully when another
// goroutine has already confirmed the entry.
func (w *ConsumerWAL) retryEntryInsert(ctx context.Context, entry *ConsumerWALEntry, callback RecoveryCallback, result *ConsumerRecoveryResult) {
	if err := callback.InsertEvent(ctx, entry.EventPayload, entry.TransactionID); err != nil {
		logging.Error().Err(err).Str("entry_id", entry.ID).Msg("Consumer WAL recovery: error inserting entry")
		if updateErr := w.UpdateAttempt(ctx, entry.ID, err.Error()); updateErr != nil {
			// RACE CONDITION FIX: ErrEntryNotFound means another goroutine already processed it
			if errors.Is(updateErr, ErrEntryNotFound) {
				logging.Debug().Str("entry_id", entry.ID).Msg("Consumer WAL recovery: entry already processed by another goroutine")
			} else {
				logging.Error().Err(updateErr).Str("entry_id", entry.ID).Msg("Consumer WAL recovery: error updating attempt")
			}
		}
		result.Failed++
		return
	}

	// Success - confirm the entry
	if err := w.Confirm(ctx, entry.ID); err != nil {
		// RACE CONDITION FIX: ErrEntryNotFound means another goroutine already confirmed it
		if errors.Is(err, ErrEntryNotFound) {
			logging.Debug().Str("entry_id", entry.ID).Msg("Consumer WAL recovery: entry already confirmed by another goroutine")
		} else {
			logging.Error().Err(err).Str("entry_id", entry.ID).Msg("Consumer WAL recovery: error confirming after insert")
		}
		// The insert succeeded but we couldn't confirm - this is okay,
		// the next recovery will find it's already in DuckDB
	}
	result.Recovered++
}

// ============================================================================
// Internal Helper Functions
// ============================================================================

// checkNotClosed returns ErrWALClosed if the WAL is closed.
// This reduces boilerplate in public methods.
func (w *ConsumerWAL) checkNotClosed() error {
	w.mu.RLock()
	defer w.mu.RUnlock()
	if w.closed {
		return ErrWALClosed
	}
	return nil
}

// readEntryFromTxn reads and unmarshals a ConsumerWALEntry from a BadgerDB transaction.
// Returns ErrEntryNotFound if the key doesn't exist.
func (w *ConsumerWAL) readEntryFromTxn(txn *badger.Txn, key []byte) (*ConsumerWALEntry, error) {
	item, err := txn.Get(key)
	if errors.Is(err, badger.ErrKeyNotFound) {
		return nil, ErrEntryNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get entry: %w", err)
	}

	var entry ConsumerWALEntry
	err = item.Value(func(val []byte) error {
		return json.Unmarshal(val, &entry)
	})
	if err != nil {
		return nil, fmt.Errorf("unmarshal entry: %w", err)
	}

	return &entry, nil
}

// writeEntryToTxn marshals and writes a ConsumerWALEntry to a BadgerDB transaction.
func (w *ConsumerWAL) writeEntryToTxn(txn *badger.Txn, key []byte, entry *ConsumerWALEntry) error {
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshal entry: %w", err)
	}
	return txn.Set(key, data)
}

// deleteBatchedKeys deletes keys in batches to avoid transaction size limits.
// Returns the number of deleted keys and any error encountered.
func (w *ConsumerWAL) deleteBatchedKeys(keys [][]byte) (int, error) {
	deleted := 0
	batchSize := 100

	for i := 0; i < len(keys); i += batchSize {
		end := i + batchSize
		if end > len(keys) {
			end = len(keys)
		}
		batch := keys[i:end]

		err := w.db.Update(func(txn *badger.Txn) error {
			for _, key := range batch {
				if err := txn.Delete(key); err != nil && !errors.Is(err, badger.ErrKeyNotFound) {
					return err
				}
				deleted++
			}
			return nil
		})
		if err != nil {
			return deleted, fmt.Errorf("delete batch: %w", err)
		}
	}

	return deleted, nil
}
