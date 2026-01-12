// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build nats

package eventprocessor

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/tomtom215/cartographus/internal/logging"
	"github.com/tomtom215/cartographus/internal/models"
)

// SyncEventPublisher implements sync.EventPublisher for NATS integration.
// It converts PlaybackEvent to MediaEvent and publishes to the event bus.
// Optionally implements sync.EventFlusher when an Appender is provided.
type SyncEventPublisher struct {
	publisher *Publisher
	appender  *Appender // Optional: for flushing events to database
}

// NewSyncEventPublisher creates a new publisher for sync manager integration.
func NewSyncEventPublisher(pub *Publisher) (*SyncEventPublisher, error) {
	if pub == nil {
		return nil, fmt.Errorf("publisher required")
	}
	return &SyncEventPublisher{publisher: pub}, nil
}

// SetAppender sets the optional appender for flushing events to database.
// When set, the publisher implements sync.EventFlusher interface.
func (p *SyncEventPublisher) SetAppender(appender *Appender) {
	p.appender = appender
}

// Flush waits for all pending events to be written to the database.
// Implements sync.EventFlusher interface.
// Returns nil if no appender is configured or flush succeeds.
func (p *SyncEventPublisher) Flush(ctx context.Context) error {
	if p.appender == nil {
		return nil
	}
	logging.Info().Msg("Flushing pending events to database")
	if err := p.appender.Flush(ctx); err != nil {
		return fmt.Errorf("flush appender: %w", err)
	}
	logging.Info().Msg("Flush complete")
	return nil
}

// EventsReceived returns the total number of events received by the appender.
// Implements sync.EventFlusherWithStats interface.
// Returns 0 if no appender is configured.
func (p *SyncEventPublisher) EventsReceived() int64 {
	if p.appender == nil {
		return 0
	}
	return p.appender.Stats().EventsReceived
}

// EventsFlushed returns the total number of events successfully flushed to database.
// Implements sync.EventFlusherWithStats interface.
// Returns 0 if no appender is configured.
func (p *SyncEventPublisher) EventsFlushed() int64 {
	if p.appender == nil {
		return 0
	}
	return p.appender.Stats().EventsFlushed
}

// ErrorCount returns the number of flush errors encountered.
// Returns 0 if no appender is configured.
func (p *SyncEventPublisher) ErrorCount() int64 {
	if p.appender == nil {
		return 0
	}
	return p.appender.Stats().ErrorCount
}

// LastError returns the last error message from a flush attempt.
// Returns empty string if no appender is configured or no errors.
func (p *SyncEventPublisher) LastError() string {
	if p.appender == nil {
		return ""
	}
	return p.appender.Stats().LastError
}

// BufferSize returns the current number of events in the buffer.
// Returns 0 if no appender is configured.
func (p *SyncEventPublisher) BufferSize() int {
	if p.appender == nil {
		return 0
	}
	return p.appender.Stats().BufferSize
}

// PublishPlaybackEvent converts a PlaybackEvent to MediaEvent and publishes it.
// This method implements sync.EventPublisher interface.
//
// The correlation key is automatically generated for cross-source deduplication.
// This enables detecting duplicate events from different sources (Tautulli, Plex webhook,
// Jellyfin) that represent the same playback session.
func (p *SyncEventPublisher) PublishPlaybackEvent(ctx context.Context, event *models.PlaybackEvent) error {
	if event == nil {
		return nil
	}

	mediaEvent := p.playbackEventToMediaEvent(event)

	// Set correlation key for cross-source deduplication
	// This is critical for event sourcing mode where events from multiple sources
	// (Tautulli sync, Plex webhooks, Jellyfin) need to be deduplicated
	mediaEvent.SetCorrelationKey()

	return p.publisher.PublishEvent(ctx, mediaEvent)
}

// playbackEventToMediaEvent converts a PlaybackEvent to MediaEvent.
// This is the inverse of DuckDBStore.mediaEventToPlaybackEvent.
//
//nolint:gocyclo // Data mapping function with many optional fields requires conditional checks
func (p *SyncEventPublisher) playbackEventToMediaEvent(event *models.PlaybackEvent) *MediaEvent {
	mediaEvent := &MediaEvent{
		EventID:    event.ID.String(),
		SessionKey: event.SessionKey, // Critical for deduplication
		Source:     event.Source,
		Timestamp:  time.Now(),

		// User information
		UserID:   event.UserID,
		Username: event.Username,

		// Media identification
		MediaType: event.MediaType,
		Title:     event.Title,

		// Timing
		StartedAt:       event.StartedAt,
		StoppedAt:       event.StoppedAt,
		PercentComplete: event.PercentComplete,
		PausedCounter:   event.PausedCounter,

		// Platform
		Platform:     event.Platform,
		Player:       event.Player,
		IPAddress:    event.IPAddress,
		LocationType: event.LocationType,
	}

	// User optional string fields
	if event.FriendlyName != nil {
		mediaEvent.FriendlyName = *event.FriendlyName
	}
	if event.UserThumb != nil {
		mediaEvent.UserThumb = *event.UserThumb
	}
	if event.Email != nil {
		mediaEvent.Email = *event.Email
	}

	// Media optional string fields
	if event.ParentTitle != nil {
		mediaEvent.ParentTitle = *event.ParentTitle
	}
	if event.GrandparentTitle != nil {
		mediaEvent.GrandparentTitle = *event.GrandparentTitle
	}
	if event.RatingKey != nil {
		mediaEvent.RatingKey = *event.RatingKey
	}

	// Media optional integer fields
	if event.Year != nil {
		mediaEvent.Year = *event.Year
	}
	// Note: MediaDuration not mapped - PlaybackEvent doesn't have media duration field

	// Platform optional fields
	if event.PlatformName != nil {
		mediaEvent.PlatformName = *event.PlatformName
	}
	if event.PlatformVersion != nil {
		mediaEvent.PlatformVersion = *event.PlatformVersion
	}
	if event.Product != nil {
		mediaEvent.Product = *event.Product
	}
	if event.ProductVersion != nil {
		mediaEvent.ProductVersion = *event.ProductVersion
	}
	if event.Device != nil {
		mediaEvent.Device = *event.Device
	}
	if event.MachineID != nil {
		mediaEvent.MachineID = *event.MachineID
	}

	// Streaming quality optional fields
	if event.TranscodeDecision != nil {
		mediaEvent.TranscodeDecision = *event.TranscodeDecision
	}
	if event.VideoResolution != nil {
		mediaEvent.VideoResolution = *event.VideoResolution
	}
	if event.VideoCodec != nil {
		mediaEvent.VideoCodec = *event.VideoCodec
	}
	if event.StreamVideoDynamicRange != nil {
		mediaEvent.VideoDynamicRange = *event.StreamVideoDynamicRange
	}
	if event.AudioCodec != nil {
		mediaEvent.AudioCodec = *event.AudioCodec
	}

	// Optional integer fields
	if event.PlayDuration != nil {
		mediaEvent.PlayDuration = *event.PlayDuration
	}
	if event.AudioChannels != nil {
		if channels, err := strconv.Atoi(*event.AudioChannels); err == nil {
			mediaEvent.AudioChannels = channels
		}
	}
	if event.StreamBitrate != nil {
		mediaEvent.StreamBitrate = *event.StreamBitrate
	}
	if event.Bandwidth != nil {
		mediaEvent.Bandwidth = *event.Bandwidth
	}

	// Boolean fields from int
	if event.Secure != nil && *event.Secure == 1 {
		mediaEvent.Secure = true
	}
	if event.Local != nil && *event.Local == 1 {
		mediaEvent.Local = true
	}
	if event.Relayed != nil && *event.Relayed == 1 {
		mediaEvent.Relayed = true
	}

	return mediaEvent
}
