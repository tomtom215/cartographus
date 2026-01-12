// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build wal && nats

package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/goccy/go-json"
	"github.com/google/uuid"

	"github.com/tomtom215/cartographus/internal/database"
	"github.com/tomtom215/cartographus/internal/eventprocessor"
	"github.com/tomtom215/cartographus/internal/logging"
	"github.com/tomtom215/cartographus/internal/models"
	"github.com/tomtom215/cartographus/internal/wal"
)

// ConsumerWALComponents holds consumer-side WAL components for lifecycle management.
// This ensures exactly-once delivery between NATS and DuckDB.
type ConsumerWALComponents struct {
	wal       *wal.ConsumerWAL
	retryLoop *ConsumerWALRetryLoop
	db        *database.DB
}

// ConsumerWALRetryLoop handles background retry of pending consumer WAL entries.
type ConsumerWALRetryLoop struct {
	wal         *wal.ConsumerWAL
	callback    wal.RecoveryCallback
	interval    time.Duration
	leaseHolder string // Unique identifier for durable leasing
	stopCh      chan struct{}
	doneCh      chan struct{}
}

// InitConsumerWAL initializes the consumer-side Write-Ahead Log for exactly-once delivery.
// This ensures events survive the gap between NATS consumption and DuckDB persistence.
//
// The consumer WAL:
// 1. Writes events to BadgerDB before ACKing NATS
// 2. Confirms entries only after successful DuckDB insert
// 3. Recovers pending entries on startup
// 4. Moves failed events to failed_events table after max retries
func InitConsumerWAL(ctx context.Context, db *database.DB) (*ConsumerWALComponents, error) {
	cfg := wal.LoadConsumerWALConfig()

	if db == nil {
		logging.Info().Msg("Consumer WAL disabled (no database provided)")
		return nil, nil
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid consumer WAL config: %w", err)
	}

	logging.Info().
		Str("path", cfg.Path).
		Bool("sync_writes", cfg.SyncWrites).
		Int("max_retries", cfg.MaxRetries).
		Msg("Initializing Consumer WAL...")

	// Open Consumer WAL
	consumerWAL, err := wal.OpenConsumerWAL(&cfg)
	if err != nil {
		return nil, fmt.Errorf("open consumer WAL: %w", err)
	}

	components := &ConsumerWALComponents{
		wal: consumerWAL,
		db:  db,
	}

	// Create recovery callback
	callback := NewDuckDBRecoveryCallback(db)

	// Run recovery of pending entries from previous run
	logging.Info().Msg("Running Consumer WAL recovery for pending entries...")
	result, err := consumerWAL.RecoverOnStartup(ctx, callback)
	if err != nil {
		logging.Warn().Err(err).Msg("Consumer WAL recovery error")
		// Don't fail initialization - recovery is best-effort
	} else if result != nil && result.TotalPending > 0 {
		logging.Info().
			Int("total", result.TotalPending).
			Int("recovered", result.Recovered).
			Int("already_committed", result.AlreadyCommitted).
			Int("expired", result.Expired).
			Int("failed", result.Failed).
			Dur("duration", result.Duration).
			Msg("Consumer WAL recovery completed")
	} else {
		logging.Info().Msg("Consumer WAL recovery: no pending entries")
	}

	// Create and start background retry loop
	retryLoop := NewConsumerWALRetryLoop(consumerWAL, callback, cfg.RetryInterval)
	retryLoop.Start(ctx)
	// Note: Start() launches a goroutine and cannot fail
	components.retryLoop = retryLoop
	logging.Info().Msg("Consumer WAL retry loop started")

	logging.Info().Msg("Consumer WAL initialized successfully")
	return components, nil
}

// WAL returns the underlying ConsumerWAL instance.
func (c *ConsumerWALComponents) WAL() *wal.ConsumerWAL {
	if c == nil {
		return nil
	}
	return c.wal
}

// Shutdown gracefully stops all consumer WAL components.
func (c *ConsumerWALComponents) Shutdown() {
	if c == nil {
		return
	}

	logging.Info().Msg("Shutting down Consumer WAL components...")

	// Stop retry loop first
	if c.retryLoop != nil {
		c.retryLoop.Stop()
		logging.Info().Msg("Consumer WAL retry loop stopped")
	}

	// Close WAL
	if c.wal != nil {
		if err := c.wal.Close(); err != nil {
			logging.Error().Err(err).Msg("Error closing Consumer WAL")
		}
		logging.Info().Msg("Consumer WAL closed")
	}

	logging.Info().Msg("Consumer WAL shutdown complete")
}

// Stats returns current consumer WAL statistics.
func (c *ConsumerWALComponents) Stats() wal.ConsumerWALStats {
	if c == nil || c.wal == nil {
		return wal.ConsumerWALStats{}
	}
	return c.wal.Stats()
}

// DuckDBRecoveryCallback implements wal.RecoveryCallback for DuckDB operations.
// It provides the database operations needed for crash recovery:
// - Check if transaction ID already exists (idempotency)
// - Insert events with transaction ID
// - Move failed events to failed_events table
type DuckDBRecoveryCallback struct {
	db *database.DB
}

// NewDuckDBRecoveryCallback creates a new recovery callback for DuckDB.
func NewDuckDBRecoveryCallback(db *database.DB) *DuckDBRecoveryCallback {
	return &DuckDBRecoveryCallback{db: db}
}

// TransactionIDExists checks if a transaction ID is already in DuckDB.
// This is used to detect events that were committed but not confirmed in WAL.
func (c *DuckDBRecoveryCallback) TransactionIDExists(ctx context.Context, transactionID string) (bool, error) {
	return c.db.TransactionIDExists(ctx, transactionID)
}

// InsertEvent inserts the event payload into DuckDB with the transaction ID.
// The payload is the raw MediaEvent JSON.
func (c *DuckDBRecoveryCallback) InsertEvent(ctx context.Context, payload []byte, transactionID string) error {
	// Parse the MediaEvent from payload
	var event eventprocessor.MediaEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return fmt.Errorf("unmarshal event: %w", err)
	}

	// Set the transaction ID
	event.TransactionID = transactionID

	// Convert to PlaybackEvent and insert
	playback := mediaEventToPlaybackEvent(&event)
	return c.db.InsertPlaybackEvent(playback)
}

// InsertFailedEvent moves an event to the failed_events table.
// Called when max retries are exceeded.
func (c *DuckDBRecoveryCallback) InsertFailedEvent(ctx context.Context, entry *wal.ConsumerWALEntry, reason string) error {
	// Parse the MediaEvent from payload to extract metadata
	var event eventprocessor.MediaEvent
	if err := json.Unmarshal(entry.EventPayload, &event); err != nil {
		// If we can't parse, store what we can
		event.EventID = entry.ID
		event.Source = "unknown"
	}

	failedEvent := &models.FailedEvent{
		ID:             uuid.New(),
		TransactionID:  entry.TransactionID,
		EventID:        event.EventID,
		SessionKey:     event.SessionKey,
		CorrelationKey: event.CorrelationKey,
		Source:         event.Source,
		EventPayload:   entry.EventPayload,
		FailedAt:       timeNow(),
		FailureReason:  reason,
		FailureLayer:   "consumer_wal",
		LastError:      entry.LastError,
		RetryCount:     entry.Attempts,
		Status:         "pending",
	}

	return c.db.InsertFailedEventWithContext(ctx, failedEvent)
}

// mediaEventToPlaybackEvent converts a MediaEvent to a PlaybackEvent.
// This is a local copy to avoid circular dependencies.
func mediaEventToPlaybackEvent(event *eventprocessor.MediaEvent) *models.PlaybackEvent {
	// Generate session key if not set
	sessionKey := event.SessionKey
	if sessionKey == "" {
		sessionKey = fmt.Sprintf("%s:%s:%s", event.Source, event.EventID, event.MachineID)
	}

	// Ensure correlation key is set
	if event.CorrelationKey == "" {
		event.SetCorrelationKey()
	}

	return &models.PlaybackEvent{
		ID:               parseOrGenerateUUID(event.EventID),
		SessionKey:       sessionKey,
		StartedAt:        event.StartedAt,
		StoppedAt:        event.StoppedAt,
		Source:           event.Source,
		CreatedAt:        timeNow(),
		CorrelationKey:   &event.CorrelationKey,
		TransactionID:    ptrOrNil(event.TransactionID),
		UserID:           event.UserID,
		Username:         event.Username,
		IPAddress:        event.IPAddress,
		MediaType:        event.MediaType,
		Title:            event.Title,
		ParentTitle:      ptrOrNil(event.ParentTitle),
		GrandparentTitle: ptrOrNil(event.GrandparentTitle),
		RatingKey:        ptrOrNil(event.RatingKey),
		MachineID:        ptrOrNil(event.MachineID),
		Platform:         event.Platform,
		Player:           event.Player,
		ServerID:         ptrOrNil(event.ServerID),
	}
}

func parseOrGenerateUUID(s string) uuid.UUID {
	if id, err := uuid.Parse(s); err == nil {
		return id
	}
	return uuid.New()
}

func ptrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func timeNow() time.Time {
	return time.Now().UTC()
}

// NewConsumerWALRetryLoop creates a new retry loop for pending consumer WAL entries.
func NewConsumerWALRetryLoop(consumerWAL *wal.ConsumerWAL, callback wal.RecoveryCallback, interval time.Duration) *ConsumerWALRetryLoop {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	// Generate unique lease holder ID for durable leasing
	leaseHolder := fmt.Sprintf("retry-loop-%s", uuid.New().String()[:8])
	return &ConsumerWALRetryLoop{
		wal:         consumerWAL,
		callback:    callback,
		interval:    interval,
		leaseHolder: leaseHolder,
		stopCh:      make(chan struct{}),
		doneCh:      make(chan struct{}),
	}
}

// Start begins the background retry loop.
func (r *ConsumerWALRetryLoop) Start(ctx context.Context) {
	go r.run(ctx)
}

// Stop gracefully stops the retry loop.
func (r *ConsumerWALRetryLoop) Stop() {
	close(r.stopCh)
	<-r.doneCh
}

// run is the main loop that periodically checks for pending entries.
func (r *ConsumerWALRetryLoop) run(ctx context.Context) {
	defer close(r.doneCh)

	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-r.stopCh:
			return
		case <-ticker.C:
			r.processPending(ctx)
		}
	}
}

// processPending processes all pending entries that need retry.
func (r *ConsumerWALRetryLoop) processPending(ctx context.Context) {
	entries, err := r.wal.GetPending(ctx)
	if err != nil {
		logging.Warn().Err(err).Msg("Consumer WAL retry: error getting pending entries")
		return
	}

	if len(entries) == 0 {
		return
	}

	logging.Debug().Int("count", len(entries)).Msg("Consumer WAL retry: processing pending entries")

	cfg := r.wal.Config()
	for _, entry := range entries {
		select {
		case <-ctx.Done():
			return
		case <-r.stopCh:
			return
		default:
		}

		r.processEntry(ctx, entry, &cfg)
	}
}

// processEntry processes a single pending entry.
//
// DURABLE LEASING: Uses TryClaimEntryDurable to prevent concurrent processing.
// If this process crashes while holding the lease, it will automatically expire
// after LeaseDuration, allowing another process to claim and process the entry.
func (r *ConsumerWALRetryLoop) processEntry(ctx context.Context, entry *wal.ConsumerWALEntry, cfg *wal.ConsumerWALConfig) {
	// Skip recently attempted entries (exponential backoff)
	if r.shouldSkipDueToBackoff(entry, cfg) {
		return
	}

	// Try to claim the entry for processing
	if !r.tryClaimEntry(ctx, entry) {
		return
	}

	// Check if already committed (crash after insert, before confirm)
	exists, err := r.callback.TransactionIDExists(ctx, entry.TransactionID)
	if err != nil {
		logging.Warn().Err(err).Str("transaction_id", entry.TransactionID).Msg("Consumer WAL retry: error checking transaction")
		r.releaseLeaseSafely(ctx, entry.ID)
		return
	}

	if exists {
		r.confirmEntrySafely(ctx, entry.ID)
		return
	}

	// Check max retries
	if entry.Attempts >= cfg.MaxRetries {
		r.handleMaxRetriesExceeded(ctx, entry, cfg)
		return
	}

	// Attempt insert
	r.attemptInsert(ctx, entry)
}

// tryClaimEntry attempts to claim exclusive processing rights for an entry.
// Returns true if claimed, false if should skip (already processed or leased by another).
func (r *ConsumerWALRetryLoop) tryClaimEntry(ctx context.Context, entry *wal.ConsumerWALEntry) bool {
	claimed, err := r.wal.TryClaimEntryDurable(ctx, entry.ID, r.leaseHolder)
	if err != nil {
		// ErrEntryNotFound means entry was already processed and deleted - this is normal
		if errors.Is(err, wal.ErrEntryNotFound) {
			logging.Debug().Str("entry_id", entry.ID).Msg("Consumer WAL retry: entry already processed, skipping")
			return false
		}
		logging.Warn().Err(err).Str("entry_id", entry.ID).Msg("Consumer WAL retry: error claiming entry")
		return false
	}
	if !claimed {
		logging.Trace().Str("entry_id", entry.ID).Msg("Consumer WAL retry: entry leased by another processor, skipping")
		return false
	}
	return true
}

// releaseLeaseSafely releases a lease, handling ErrEntryNotFound gracefully.
func (r *ConsumerWALRetryLoop) releaseLeaseSafely(ctx context.Context, entryID string) {
	if releaseErr := r.wal.ReleaseLeaseDurable(ctx, entryID); releaseErr != nil {
		if !errors.Is(releaseErr, wal.ErrEntryNotFound) {
			logging.Warn().Err(releaseErr).Str("entry_id", entryID).Msg("Consumer WAL retry: error releasing lease")
		}
	}
}

// confirmEntrySafely confirms an entry, handling ErrEntryNotFound gracefully.
func (r *ConsumerWALRetryLoop) confirmEntrySafely(ctx context.Context, entryID string) {
	if err := r.wal.Confirm(ctx, entryID); err != nil {
		if !errors.Is(err, wal.ErrEntryNotFound) {
			logging.Warn().Err(err).Str("entry_id", entryID).Msg("Consumer WAL retry: error confirming")
		}
	}
}

// attemptInsert tries to insert the entry into DuckDB.
func (r *ConsumerWALRetryLoop) attemptInsert(ctx context.Context, entry *wal.ConsumerWALEntry) {
	if err := r.callback.InsertEvent(ctx, entry.EventPayload, entry.TransactionID); err != nil {
		logging.Warn().Err(err).Str("entry_id", entry.ID).Int("attempt", entry.Attempts+1).Msg("Consumer WAL retry: error inserting entry")
		if updateErr := r.wal.UpdateAttempt(ctx, entry.ID, err.Error()); updateErr != nil {
			if !errors.Is(updateErr, wal.ErrEntryNotFound) {
				logging.Error().Err(updateErr).Str("entry_id", entry.ID).Msg("Consumer WAL retry: error updating attempt")
			}
		}
		r.releaseLeaseSafely(ctx, entry.ID)
		return
	}

	// Success - confirm (this removes the entry, so lease is implicitly released)
	if err := r.wal.Confirm(ctx, entry.ID); err != nil {
		if !errors.Is(err, wal.ErrEntryNotFound) {
			logging.Error().Err(err).Str("entry_id", entry.ID).Msg("Consumer WAL retry: error confirming")
		}
	} else {
		logging.Debug().Str("entry_id", entry.ID).Msg("Consumer WAL retry: successfully recovered entry")
	}
}

// shouldSkipDueToBackoff checks if the entry should be skipped due to exponential backoff.
func (r *ConsumerWALRetryLoop) shouldSkipDueToBackoff(entry *wal.ConsumerWALEntry, cfg *wal.ConsumerWALConfig) bool {
	backoff := time.Duration(entry.Attempts) * cfg.RetryBackoff
	if backoff > 5*time.Minute {
		backoff = 5 * time.Minute // Cap at 5 minutes
	}
	return !entry.LastAttemptAt.IsZero() && time.Since(entry.LastAttemptAt) < backoff
}

// handleMaxRetriesExceeded handles an entry that has exceeded the maximum retry count.
func (r *ConsumerWALRetryLoop) handleMaxRetriesExceeded(ctx context.Context, entry *wal.ConsumerWALEntry, cfg *wal.ConsumerWALConfig) {
	logging.Warn().Str("entry_id", entry.ID).Int("max_retries", cfg.MaxRetries).Msg("Consumer WAL retry: entry exceeded max retries")
	if err := r.callback.InsertFailedEvent(ctx, entry, "max_retries_exceeded"); err != nil {
		logging.Error().Err(err).Str("entry_id", entry.ID).Msg("Consumer WAL retry: error inserting failed event")
	}
	if err := r.wal.MarkFailed(ctx, entry.ID, "exceeded max retries"); err != nil {
		logging.Error().Err(err).Str("entry_id", entry.ID).Msg("Consumer WAL retry: error marking failed")
	}
}
