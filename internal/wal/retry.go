// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build wal

package wal

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/tomtom215/cartographus/internal/logging"
)

// RetryLoop handles background retry of failed WAL entries.
// It periodically checks for pending entries and attempts to publish them to NATS.
type RetryLoop struct {
	wal         *BadgerWAL
	publisher   Publisher
	config      Config
	leaseHolder string // Unique identifier for durable leasing

	// Control
	ctx    context.Context
	cancel context.CancelFunc

	// State - all protected by mu
	mu       sync.Mutex
	running  bool
	stopping bool          // true while Stop() is waiting for goroutine
	stopDone chan struct{} // closed when Stop() completes
}

// NewRetryLoop creates a new background retry loop.
func NewRetryLoop(wal *BadgerWAL, publisher Publisher) *RetryLoop {
	// Generate unique lease holder ID for durable leasing
	leaseHolder := fmt.Sprintf("retry-loop-%s", uuid.New().String()[:8])
	return &RetryLoop{
		wal:         wal,
		publisher:   publisher,
		config:      wal.GetConfig(),
		leaseHolder: leaseHolder,
	}
}

// Start begins the background retry loop.
// It will run until Stop is called or the context is canceled.
func (r *RetryLoop) Start(ctx context.Context) error {
	r.mu.Lock()

	// Wait for any in-progress Stop() to complete
	for r.stopping {
		stopDone := r.stopDone
		r.mu.Unlock()
		<-stopDone
		r.mu.Lock()
	}

	if r.running {
		r.mu.Unlock()
		return nil
	}

	r.ctx, r.cancel = context.WithCancel(ctx)
	r.running = true
	r.stopDone = make(chan struct{})

	// Capture context and done channel to avoid races
	loopCtx := r.ctx
	done := r.stopDone

	r.mu.Unlock()

	go r.runWithContext(loopCtx, done)

	logging.Info().
		Dur("interval", r.config.RetryInterval).
		Int("max_retries", r.config.MaxRetries).
		Msg("WAL retry loop started")
	return nil
}

// Stop gracefully stops the retry loop.
func (r *RetryLoop) Stop() {
	r.mu.Lock()
	if !r.running || r.stopping {
		r.mu.Unlock()
		return
	}

	r.cancel()
	r.running = false
	r.stopping = true
	stopDone := r.stopDone
	r.mu.Unlock()

	// Wait for the goroutine to signal completion
	<-stopDone

	r.mu.Lock()
	r.stopping = false
	r.mu.Unlock()

	logging.Info().Msg("WAL retry loop stopped")
}

// IsRunning returns whether the retry loop is active.
func (r *RetryLoop) IsRunning() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.running
}

// runWithContext is the main retry loop goroutine.
// The context is passed as a parameter to avoid race conditions with Stop().
// The done channel is closed when the goroutine exits to signal completion.
func (r *RetryLoop) runWithContext(ctx context.Context, done chan struct{}) {
	defer close(done)

	ticker := time.NewTicker(r.config.RetryInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.retryPendingWithContext(ctx)
		}
	}
}

// retryResult tracks the outcome of processing a single entry.
type retryResult int

const (
	retryResultSuccess retryResult = iota
	retryResultFailed
	retryResultExpired
	retryResultMaxRetried
	retryResultSkipped
	retryResultCanceled
)

// retryPendingWithContext attempts to publish all pending entries.
// The context is passed as a parameter to avoid race conditions.
func (r *RetryLoop) retryPendingWithContext(ctx context.Context) {
	entries, err := r.wal.GetPending(ctx)
	if err != nil {
		logging.Error().Err(err).Msg("WAL retry: failed to get pending entries")
		return
	}

	if len(entries) == 0 {
		return
	}

	logging.Info().Int("pending_entries", len(entries)).Msg("WAL retry: processing pending entries")

	var success, failed, expired, maxRetried int

	for _, entry := range entries {
		select {
		case <-ctx.Done():
			return
		default:
		}

		result := r.processEntryWithContext(ctx, entry)
		switch result {
		case retryResultSuccess:
			success++
		case retryResultFailed:
			failed++
		case retryResultExpired:
			expired++
		case retryResultMaxRetried:
			maxRetried++
		}
	}

	if success > 0 || failed > 0 || expired > 0 || maxRetried > 0 {
		logging.Info().
			Int("succeeded", success).
			Int("failed", failed).
			Int("expired", expired).
			Int("max_retried", maxRetried).
			Msg("WAL retry complete")
	}
}

// processEntryWithContext handles a single entry retry attempt.
// The context is passed as a parameter to avoid race conditions.
//
// DURABLE LEASING: Uses TryClaimEntryDurable for crash-safe concurrent processing prevention.
// If another processor has the lease, we skip it. If this process crashes while holding
// the lease, it will automatically expire after LeaseDuration.
func (r *RetryLoop) processEntryWithContext(ctx context.Context, entry *Entry) retryResult {
	// DURABLE LEASING: Try to claim exclusive processing rights using BadgerDB
	claimed, err := r.wal.TryClaimEntryDurable(ctx, entry.ID, r.leaseHolder)
	if err != nil {
		logging.Error().Err(err).Str("entry_id", entry.ID).Msg("WAL retry: error claiming entry")
		return retryResultFailed
	}
	if !claimed {
		// Another processor has the lease, skip this entry
		return retryResultSkipped
	}
	// Note: No defer release needed - DeleteEntry/Confirm removes entry, or lease expires naturally

	// Check if entry is expired
	if time.Since(entry.CreatedAt) > r.config.EntryTTL {
		return r.handleExpiredEntryWithContext(ctx, entry)
	}

	// Check if max retries exceeded
	if entry.Attempts >= r.config.MaxRetries {
		return r.handleMaxRetriedEntryWithContext(ctx, entry)
	}

	// Check backoff delay
	if !r.isReadyForRetry(entry) {
		// Release lease early to allow faster retry by another processor
		if releaseErr := r.wal.ReleaseLeaseDurable(ctx, entry.ID); releaseErr != nil {
			logging.Warn().Err(releaseErr).Str("entry_id", entry.ID).Msg("WAL retry: error releasing lease")
		}
		return retryResultSkipped
	}

	// Attempt to publish
	return r.attemptPublishWithContext(ctx, entry)
}

// handleExpiredEntryWithContext removes an expired entry.
// The context is passed as a parameter to avoid race conditions.
func (r *RetryLoop) handleExpiredEntryWithContext(ctx context.Context, entry *Entry) retryResult {
	logging.Info().Str("entry_id", entry.ID).Msg("WAL retry: entry expired, removing")
	if err := r.wal.DeleteEntry(ctx, entry.ID); err != nil {
		logging.Error().Err(err).Str("entry_id", entry.ID).Msg("WAL retry: failed to delete expired entry")
	}
	RecordWALExpiredEntry()
	return retryResultExpired
}

// handleMaxRetriedEntryWithContext removes an entry that exceeded max retries.
// The context is passed as a parameter to avoid race conditions.
func (r *RetryLoop) handleMaxRetriedEntryWithContext(ctx context.Context, entry *Entry) retryResult {
	logging.Info().
		Str("entry_id", entry.ID).
		Int("attempts", entry.Attempts).
		Int("max_retries", r.config.MaxRetries).
		Msg("WAL retry: entry exceeded max retries, removing")
	if err := r.wal.DeleteEntry(ctx, entry.ID); err != nil {
		logging.Error().Err(err).Str("entry_id", entry.ID).Msg("WAL retry: failed to delete max-retried entry")
	}
	RecordWALMaxRetriesExceeded()
	return retryResultMaxRetried
}

// isReadyForRetry checks if enough time has passed since last attempt.
func (r *RetryLoop) isReadyForRetry(entry *Entry) bool {
	if entry.LastAttemptAt.IsZero() {
		return true
	}
	backoff := r.calculateBackoff(entry.Attempts)
	return time.Since(entry.LastAttemptAt) >= backoff
}

// attemptPublishWithContext tries to publish an entry to NATS.
// The context is passed as a parameter to avoid race conditions.
func (r *RetryLoop) attemptPublishWithContext(ctx context.Context, entry *Entry) retryResult {
	pubCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	err := r.publisher.PublishEntry(pubCtx, entry)
	cancel()

	if err != nil {
		logging.Error().
			Err(err).
			Str("entry_id", entry.ID).
			Int("attempt", entry.Attempts+1).
			Msg("WAL retry: failed to publish entry")
		if updateErr := r.wal.UpdateAttempt(ctx, entry.ID, err.Error()); updateErr != nil {
			logging.Error().Err(updateErr).Str("entry_id", entry.ID).Msg("WAL retry: failed to update attempt")
		}
		RecordWALNATSPublishFailure()
		return retryResultFailed
	}

	// Confirm successful publish
	if err := r.wal.Confirm(ctx, entry.ID); err != nil {
		logging.Error().Err(err).Str("entry_id", entry.ID).Msg("WAL retry: failed to confirm entry")
		return retryResultFailed
	}

	return retryResultSuccess
}

// calculateBackoff calculates exponential backoff delay for retry attempts.
// Formula: base * 2^attempts, capped at 5 minutes.
func (r *RetryLoop) calculateBackoff(attempts int) time.Duration {
	base := r.config.RetryBackoff
	maxBackoff := 5 * time.Minute

	// Cap attempts to prevent overflow (2^63 is the max for time.Duration)
	// At ~52 attempts with 1s base, we'd exceed the cap anyway
	if attempts > 50 {
		return maxBackoff
	}

	// Exponential backoff: base * 2^attempts
	multiplier := math.Pow(2, float64(attempts))
	backoff := time.Duration(float64(base) * multiplier)

	// Handle overflow (negative duration means overflow occurred)
	if backoff < 0 || backoff > maxBackoff {
		backoff = maxBackoff
	}

	return backoff
}

// GetStats returns current retry loop statistics.
func (r *RetryLoop) GetStats() RetryStats {
	entries, err := r.wal.GetPending(context.Background())
	if err != nil {
		return RetryStats{}
	}

	stats := RetryStats{
		PendingCount: len(entries),
	}

	for _, entry := range entries {
		stats.TotalAttempts += entry.Attempts
		if entry.Attempts > stats.MaxAttempts {
			stats.MaxAttempts = entry.Attempts
		}
		if stats.OldestEntry.IsZero() || entry.CreatedAt.Before(stats.OldestEntry) {
			stats.OldestEntry = entry.CreatedAt
		}
	}

	return stats
}

// RetryStats contains statistics about the retry loop.
type RetryStats struct {
	PendingCount  int
	TotalAttempts int
	MaxAttempts   int
	OldestEntry   time.Time
}
