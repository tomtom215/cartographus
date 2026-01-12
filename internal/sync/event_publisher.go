// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package sync

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/tomtom215/cartographus/internal/logging"
	"github.com/tomtom215/cartographus/internal/models"
)

// asyncPublishErrors tracks the number of async publish failures.
// DETERMINISM: This counter provides visibility into non-propagated errors
// from the deprecated publishEvent method for monitoring purposes.
var asyncPublishErrors atomic.Int64

// GetAsyncPublishErrors returns the total count of async publish errors.
// This is useful for monitoring and alerting on publish failures that would
// otherwise be silently logged.
func GetAsyncPublishErrors() int64 {
	return asyncPublishErrors.Load()
}

// ResetAsyncPublishErrors resets the async publish error counter.
// This should ONLY be used in tests to ensure deterministic test runs.
func ResetAsyncPublishErrors() {
	asyncPublishErrors.Store(0)
}

// EventPublisher defines the interface for publishing playback events.
// This abstraction allows optional NATS integration without requiring
// the nats build tag for the sync package.
type EventPublisher interface {
	// PublishPlaybackEvent publishes a playback event to the event bus.
	// Returns nil if publishing is not enabled or succeeds.
	// Errors should be logged but not block sync operations.
	PublishPlaybackEvent(ctx context.Context, event *models.PlaybackEvent) error
}

// EventFlusher is an optional interface for publishers that support flushing.
// When implemented, Flush() will be called after sync completion to ensure
// all events are written to the database before reporting sync complete.
type EventFlusher interface {
	// Flush waits for all pending events to be written to the database.
	// This is called after sync completion in event sourcing mode to ensure
	// deterministic database state for testing and monitoring.
	Flush(ctx context.Context) error
}

// EventFlusherWithStats extends EventFlusher with statistics for count-based verification.
// This enables deterministic sync completion by verifying events actually reached the database.
type EventFlusherWithStats interface {
	EventFlusher

	// EventsReceived returns the total number of events received by the appender.
	EventsReceived() int64

	// EventsFlushed returns the total number of events successfully flushed to database.
	EventsFlushed() int64

	// ErrorCount returns the number of flush errors encountered.
	ErrorCount() int64

	// LastError returns the last error message from a flush attempt.
	LastError() string

	// BufferSize returns the current number of events in the buffer.
	BufferSize() int
}

// FlushConfig controls the flush verification behavior.
type FlushConfig struct {
	// VerificationTimeout is the maximum time to wait for all events to be flushed.
	// Default: 30 seconds
	VerificationTimeout time.Duration

	// PollInterval is how often to check for new events.
	// Default: 50ms
	PollInterval time.Duration

	// StabilityThreshold is how many consecutive polls with no change before considering stable.
	// Default: 5 (250ms at 50ms poll interval)
	StabilityThreshold int

	// MaxFlushAttempts is the maximum number of flush cycles before giving up.
	// Default: 10
	MaxFlushAttempts int
}

// DefaultFlushConfig returns production-safe defaults.
// MaxFlushAttempts is set high to allow for slow database inserts.
// Each attempt waits 500ms between flushes, so 30 attempts gives ~15 seconds
// for store inserts to complete. This is necessary because the store insert
// can take 5+ seconds for large batches with individual INSERT statements.
func DefaultFlushConfig() FlushConfig {
	return FlushConfig{
		VerificationTimeout: 30 * time.Second,
		PollInterval:        50 * time.Millisecond,
		StabilityThreshold:  5,
		MaxFlushAttempts:    30, // Increased from 10 to allow for slow DB inserts
	}
}

// SetEventPublisher sets the optional event publisher for NATS integration.
// When set, playback events will be published after successful database insertion.
// The publisher is optional - passing nil disables event publishing.
func (m *Manager) SetEventPublisher(publisher EventPublisher) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.eventPublisher = publisher
}

// publishEvent publishes a playback event if a publisher is configured.
// This method still uses async publishing for backwards compatibility in notification mode.
// Uses Manager's publishWg to track in-flight goroutines for deterministic flush.
//
// Deprecated: Use publishEventSync for event-sourcing mode to ensure zero data loss.
//
// DETERMINISM: Errors are tracked via asyncPublishErrors counter for monitoring.
// Use GetAsyncPublishErrors() to check for publish failures that occurred in the background.
// While this method doesn't propagate errors (by design for notification mode), the counter
// provides visibility into publish health for observability.
func (m *Manager) publishEvent(ctx context.Context, event *models.PlaybackEvent) {
	m.mu.RLock()
	publisher := m.eventPublisher
	m.mu.RUnlock()

	if publisher == nil {
		return
	}

	// Track this goroutine so flushPublisher can wait for it
	m.publishWg.Add(1)

	// Publish asynchronously to avoid blocking sync
	go func() {
		defer m.publishWg.Done()
		if err := publisher.PublishPlaybackEvent(ctx, event); err != nil {
			// DETERMINISM: Track error count for monitoring even though we don't propagate.
			// In notification mode, the event is already safely in DuckDB, so this is informational.
			asyncPublishErrors.Add(1)
			logging.Warn().Err(err).Str("event_id", event.ID.String()).Int64("total_errors", asyncPublishErrors.Load()).Msg("Async publish failed")
		}
	}()
}

// publishEventSync publishes a playback event SYNCHRONOUSLY with proper error handling.
// CRITICAL (v2.3): This is the required method for Event Sourcing Mode to ensure zero data loss.
//
// Returns:
//   - nil: Event was successfully published to NATS
//   - error: Publishing failed (caller MUST handle by falling back to direct DB insert)
//
// This method differs from publishEvent:
//   - Synchronous (blocks until publish completes or fails)
//   - Returns errors instead of silently ignoring them
//   - Does NOT track in publishWg (synchronous completion is implicit)
func (m *Manager) publishEventSync(ctx context.Context, event *models.PlaybackEvent) error {
	m.mu.RLock()
	publisher := m.eventPublisher
	m.mu.RUnlock()

	if publisher == nil {
		return fmt.Errorf("publisher not configured")
	}

	// Publish synchronously and return error
	if err := publisher.PublishPlaybackEvent(ctx, event); err != nil {
		return fmt.Errorf("NATS publish failed for event %s: %w", event.ID.String(), err)
	}

	return nil
}

// publishEventWithFallback publishes an event to NATS with automatic DB fallback on failure.
// CRITICAL (v2.3): This ensures ZERO data loss in Event Sourcing Mode.
//
// Flow:
//  1. Attempt synchronous publish to NATS
//  2. If publish succeeds, DuckDBConsumer will handle DB write
//  3. If publish fails, immediately fallback to direct DB insert
//
// This guarantees the event is persisted either through NATS→DuckDBConsumer or directly to DB.
func (m *Manager) publishEventWithFallback(ctx context.Context, event *models.PlaybackEvent) error {
	// Generate correlation key before attempting publish
	// This ensures consistent dedup behavior regardless of which path is taken
	if event.CorrelationKey == nil {
		correlationKey := generatePlaybackEventCorrelationKey(event)
		event.CorrelationKey = &correlationKey
	}

	// TRACING: Log publish attempt
	logging.Trace().Str("session", event.SessionKey).Str("event_id", event.ID.String()).Str("correlation", *event.CorrelationKey).Msg("attempting NATS publish")

	// Attempt synchronous NATS publish
	err := m.publishEventSync(ctx, event)
	if err == nil {
		// Successfully published to NATS - DuckDBConsumer will handle DB write
		logging.Trace().Str("session", event.SessionKey).Msg("NATS publish success (DuckDBConsumer will persist)")
		return nil
	}

	// NATS publish failed - fallback to direct DB insert
	logging.Warn().Err(err).Msg("NATS publish failed, using direct DB insert")
	logging.Trace().Str("event_id", event.ID.String()).Str("session", event.SessionKey).Str("correlation", *event.CorrelationKey).Msg("fallback event details")

	// Insert directly to database as fallback
	if dbErr := m.db.InsertPlaybackEvent(event); dbErr != nil {
		// CRITICAL: Both NATS and DB failed - this should NEVER happen
		logging.Error().Err(dbErr).Str("event_id", event.ID.String()).Str("nats_error", err.Error()).Msg("CRITICAL DATA LOSS: Both NATS and DB insert failed")
		return fmt.Errorf("CRITICAL: both NATS and DB failed for event %s: %w", event.ID.String(), dbErr)
	}

	logging.Info().Str("event_id", event.ID.String()).Msg("event saved via direct DB insert (NATS unavailable)")
	return nil
}

// flushPublisherWithVerification waits for events to be received and flushed to database.
// This provides deterministic sync completion by ensuring all events are written to DB.
//
// The verification process:
//  1. Wait for all publish goroutines to complete (NATS accepted messages)
//  2. Wait for event flow to stabilize (no new events arriving to Appender)
//  3. Loop flushing until received == flushed (all buffered events written)
//
// This handles the async nature of NATS delivery gracefully.
func (m *Manager) flushPublisherWithVerification(ctx context.Context, expectedEvents int) {
	cfg := DefaultFlushConfig()

	logging.Trace().Int("expected_events", expectedEvents).Msg("flush verification starting")

	// Step 1: Wait for all publish goroutines to complete
	logging.Trace().Msg("waiting for publish goroutines to complete")
	publishStart := time.Now()
	done := make(chan struct{})
	go func() {
		m.publishWg.Wait()
		close(done)
	}()

	select {
	case <-done:
		logging.Trace().Dur("duration", time.Since(publishStart)).Msg("all publish goroutines complete")
	case <-ctx.Done():
		logging.Trace().Msg("context canceled while waiting for publish goroutines")
		return
	}

	// Step 2: Get publisher and check for stats support
	m.mu.RLock()
	publisher := m.eventPublisher
	m.mu.RUnlock()

	if publisher == nil {
		logging.Trace().Msg("no publisher configured, skipping flush")
		return
	}

	// Check if publisher supports stats (for count-based verification)
	flusherWithStats, hasStats := publisher.(EventFlusherWithStats)
	flusher, hasFlusher := publisher.(EventFlusher)

	if !hasFlusher {
		logging.Trace().Msg("publisher doesn't support flushing, skipping")
		return
	}

	// Step 3: Wait for events and flush until complete
	if hasStats {
		m.flushUntilComplete(ctx, flusherWithStats, cfg)
	} else {
		// Fallback: simple delay-based approach (less reliable)
		logging.Trace().Msg("publisher doesn't support stats, using delay-based fallback")
		select {
		case <-time.After(500 * time.Millisecond):
		case <-ctx.Done():
			return
		}
		if err := flusher.Flush(ctx); err != nil {
			logging.Trace().Err(err).Msg("flush error (non-fatal)")
		}
	}
}

// flushUntilComplete waits for event flow to stabilize, then flushes in a loop
// until all received events have been written to the database.
func (m *Manager) flushUntilComplete(ctx context.Context, flusher EventFlusherWithStats, cfg FlushConfig) {
	startTime := time.Now()

	// Log initial stats including error info
	initialReceived := flusher.EventsReceived()
	initialFlushed := flusher.EventsFlushed()
	bufferSize := flusher.BufferSize()
	errorCount := flusher.ErrorCount()
	lastError := flusher.LastError()
	logging.Trace().Int64("received", initialReceived).Int64("flushed", initialFlushed).Int("buffer", bufferSize).Int64("errors", errorCount).Msg("flush initial stats")
	if lastError != "" {
		logging.Trace().Str("last_error", lastError).Msg("flush last error")
	}

	// Wait for event flow to stabilize (no new events arriving)
	m.waitForEventStability(ctx, flusher, cfg)

	// Loop: flush until received == flushed
	for attempt := 1; attempt <= cfg.MaxFlushAttempts; attempt++ {
		received := flusher.EventsReceived()
		flushed := flusher.EventsFlushed()
		bufSize := flusher.BufferSize()
		errCount := flusher.ErrorCount()
		pending := received - flushed

		if pending <= 0 {
			// All events flushed
			logging.Trace().Int64("received", received).Int64("flushed", flushed).Dur("duration", time.Since(startTime)).Msg("all events flushed")
			return
		}

		logging.Trace().Int("attempt", attempt).Int64("pending", pending).Int64("received", received).Int64("flushed", flushed).Int("buffer", bufSize).Int64("errors", errCount).Msg("flush attempt")

		// Flush buffered events
		flushStart := time.Now()
		if err := flusher.Flush(ctx); err != nil {
			logging.Trace().Err(err).Msg("flush error (non-fatal)")
		}
		flushDuration := time.Since(flushStart)

		// Get stats IMMEDIATELY after flush completes to capture the state
		postReceived := flusher.EventsReceived()
		postFlushed := flusher.EventsFlushed()
		postBuffer := flusher.BufferSize()
		postErrors := flusher.ErrorCount()
		logging.Trace().Dur("duration", flushDuration).Int64("received", postReceived).Int64("flushed", postFlushed).Int("buffer", postBuffer).Int64("errors", postErrors).Msg("flush complete - post-flush stats")

		// Wait longer between attempts to allow store insert to complete.
		// The store insert can take 5+ seconds for large batches.
		select {
		case <-time.After(500 * time.Millisecond):
		case <-ctx.Done():
			logging.Trace().Msg("context canceled during flush loop")
			return
		}
	}

	// Max attempts reached - log detailed diagnostics
	received := flusher.EventsReceived()
	flushed := flusher.EventsFlushed()
	bufSize := flusher.BufferSize()
	errCount := flusher.ErrorCount()
	lastErr := flusher.LastError()
	if received > flushed {
		logging.Warn().Int64("received", received).Int64("flushed", flushed).Int("buffer", bufSize).Int64("errors", errCount).Msg("Max flush attempts reached")
		if lastErr != "" {
			logging.Warn().Str("last_error", lastErr).Msg("Last flush error")
		}
	}
}

// waitForEventStability polls until no new events arrive for consecutive polls.
// This ensures all NATS messages have been delivered to the Appender.
func (m *Manager) waitForEventStability(ctx context.Context, flusher EventFlusherWithStats, cfg FlushConfig) {
	timeout := time.After(cfg.VerificationTimeout)
	ticker := time.NewTicker(cfg.PollInterval)
	defer ticker.Stop()

	var lastReceived int64
	stableCount := 0
	pollCount := 0
	startTime := time.Now()

	for {
		select {
		case <-timeout:
			logging.Trace().Dur("duration", time.Since(startTime)).Msg("stability timeout - proceeding with flush")
			return

		case <-ticker.C:
			pollCount++
			currentReceived := flusher.EventsReceived()

			// Log progress every 20 polls (1 second at 50ms interval)
			if pollCount%20 == 0 {
				logging.Trace().Int("poll", pollCount).Int64("received", currentReceived).Int("stable", stableCount).Int("threshold", cfg.StabilityThreshold).Msg("poll progress")
			}

			// Check for stability (no new events)
			if currentReceived == lastReceived {
				stableCount++
				if stableCount >= cfg.StabilityThreshold {
					logging.Trace().Dur("duration", time.Since(startTime)).Int64("received", currentReceived).Msg("event flow stabilized")
					return
				}
			} else {
				stableCount = 0
				lastReceived = currentReceived
			}

		case <-ctx.Done():
			logging.Trace().Msg("context canceled during stability wait")
			return
		}
	}
}
