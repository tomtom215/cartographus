// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build wal && nats

package eventprocessor

import (
	"context"

	"github.com/tomtom215/cartographus/internal/logging"
	"github.com/tomtom215/cartographus/internal/models"
	"github.com/tomtom215/cartographus/internal/wal"
)

// WALEnabledPublisher wraps a SyncEventPublisher with WAL durability.
// Events are persisted to the WAL before NATS publishing, ensuring no event loss
// in case of NATS failures, process crashes, or power loss.
//
// The flow is:
//  1. Convert PlaybackEvent to MediaEvent
//  2. Write to WAL (ACID, durable)
//  3. Attempt NATS publish
//  4. On success: Confirm WAL entry
//  5. On failure: Entry remains in WAL for retry by background RetryLoop
type WALEnabledPublisher struct {
	inner *SyncEventPublisher
	wal   *wal.BadgerWAL
}

// NewWALEnabledPublisher creates a WAL-enabled event publisher.
// The inner publisher handles the actual NATS publishing.
// The WAL ensures durability before publishing.
func NewWALEnabledPublisher(inner *SyncEventPublisher, w *wal.BadgerWAL) (*WALEnabledPublisher, error) {
	if inner == nil {
		return nil, &ValidationError{Field: "inner", Message: "inner publisher required"}
	}
	if w == nil {
		return nil, &ValidationError{Field: "wal", Message: "WAL required"}
	}
	return &WALEnabledPublisher{
		inner: inner,
		wal:   w,
	}, nil
}

// PublishPlaybackEvent implements sync.EventPublisher with WAL durability.
// The event is first persisted to the WAL, then published to NATS.
// On successful publish, the WAL entry is confirmed. On failure, the entry
// remains in the WAL for later retry by the background RetryLoop.
func (p *WALEnabledPublisher) PublishPlaybackEvent(ctx context.Context, event *models.PlaybackEvent) error {
	if event == nil {
		return nil
	}

	// Convert to MediaEvent for WAL storage
	mediaEvent := p.inner.playbackEventToMediaEvent(event)

	// Set correlation key for cross-source deduplication
	mediaEvent.SetCorrelationKey()

	// Write to WAL first (ACID, durable)
	entryID, err := p.wal.Write(ctx, mediaEvent)
	if err != nil {
		logging.Error().
			Str("event_id", event.ID.String()).
			Err(err).
			Msg("WAL write failed for event")
		wal.RecordWALWriteFailure()
		// Fall through to try NATS anyway - better to attempt than lose the event
		return p.inner.publisher.PublishEvent(ctx, mediaEvent)
	}

	// Attempt NATS publish
	if err := p.inner.publisher.PublishEvent(ctx, mediaEvent); err != nil {
		logging.Warn().
			Str("event_id", event.ID.String()).
			Str("wal_entry_id", entryID).
			Err(err).
			Msg("NATS publish failed, entry will be retried")
		// Return nil - entry is safe in WAL and will be retried by RetryLoop
		wal.RecordWALNATSPublishFailure()
		return nil
	}

	// NATS publish succeeded - confirm WAL entry
	if err := p.wal.Confirm(ctx, entryID); err != nil {
		logging.Warn().
			Str("wal_entry_id", entryID).
			Err(err).
			Msg("WAL confirm failed")
		// Event was published, confirm failure is non-fatal (entry will be cleaned up eventually)
	}

	return nil
}

// WAL returns the underlying WAL for background processing.
func (p *WALEnabledPublisher) WAL() *wal.BadgerWAL {
	return p.wal
}

// Inner returns the underlying SyncEventPublisher.
// This is useful for recovery operations that need to publish directly.
func (p *WALEnabledPublisher) Inner() *SyncEventPublisher {
	return p.inner
}

// CreateWALPublisher creates a wal.Publisher that publishes MediaEvents to NATS.
// This is used by the WAL recovery and retry loops.
func (p *WALEnabledPublisher) CreateWALPublisher() wal.Publisher {
	return wal.PublisherFunc(func(ctx context.Context, entry *wal.Entry) error {
		// Unmarshal the payload to get the MediaEvent
		var event MediaEvent
		if err := entry.UnmarshalPayload(&event); err != nil {
			return err
		}
		return p.inner.publisher.PublishEvent(ctx, &event)
	})
}

// Flush implements sync.EventFlusher by delegating to the inner publisher.
// This ensures the WAL wrapper is transparent for flush operations.
func (p *WALEnabledPublisher) Flush(ctx context.Context) error {
	return p.inner.Flush(ctx)
}

// EventsReceived implements sync.EventFlusherWithStats by delegating to inner.
// Returns the total events received by the underlying appender.
func (p *WALEnabledPublisher) EventsReceived() int64 {
	return p.inner.EventsReceived()
}

// EventsFlushed implements sync.EventFlusherWithStats by delegating to inner.
// Returns the total events flushed by the underlying appender.
func (p *WALEnabledPublisher) EventsFlushed() int64 {
	return p.inner.EventsFlushed()
}

// ErrorCount returns the number of flush errors encountered.
func (p *WALEnabledPublisher) ErrorCount() int64 {
	return p.inner.ErrorCount()
}

// LastError returns the last error message from a flush attempt.
func (p *WALEnabledPublisher) LastError() string {
	return p.inner.LastError()
}

// BufferSize returns the current number of events in the buffer.
func (p *WALEnabledPublisher) BufferSize() int {
	return p.inner.BufferSize()
}
