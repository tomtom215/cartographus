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

	"github.com/google/uuid"
	"github.com/tomtom215/cartographus/internal/logging"
	"github.com/tomtom215/cartographus/internal/models"
)

// PlaybackEventInserter defines the interface for inserting playback events.
// This abstraction allows the DuckDBStore to work with the database package
// without importing it directly, avoiding circular dependencies.
type PlaybackEventInserter interface {
	InsertPlaybackEvent(event *models.PlaybackEvent) error
}

// BatchPlaybackEventInserter extends PlaybackEventInserter with atomic batch operations.
// Implementations must guarantee all-or-nothing semantics using database transactions.
type BatchPlaybackEventInserter interface {
	PlaybackEventInserter

	// InsertPlaybackEventsBatch atomically inserts a batch of playback events.
	// Returns:
	//   - inserted: number of events successfully inserted
	//   - duplicates: number of events skipped due to unique constraint violations
	//   - err: error if the transaction failed (all events are rolled back)
	//
	// CRITICAL: This method MUST use a database transaction to ensure atomicity.
	// If ANY insert fails (other than duplicate), ALL inserts are rolled back.
	InsertPlaybackEventsBatch(ctx context.Context, events []*models.PlaybackEvent) (inserted int, duplicates int, err error)
}

// DuckDBStore implements EventStore using the DuckDB database.
// It converts MediaEvent to PlaybackEvent and delegates to the database layer.
//
// CRITICAL (v2.3): Supports atomic batch inserts via BatchPlaybackEventInserter.
// When the underlying db implements BatchPlaybackEventInserter, all inserts
// are wrapped in a transaction for all-or-nothing semantics.
type DuckDBStore struct {
	db      PlaybackEventInserter
	batchDB BatchPlaybackEventInserter // nil if db doesn't support batch ops
}

// NewDuckDBStore creates a new DuckDBStore with the given database.
// If the database implements BatchPlaybackEventInserter, atomic batch operations are enabled.
func NewDuckDBStore(db PlaybackEventInserter) (*DuckDBStore, error) {
	if db == nil {
		return nil, fmt.Errorf("database required")
	}

	store := &DuckDBStore{db: db}

	// Check if the database supports atomic batch operations
	if batchDB, ok := db.(BatchPlaybackEventInserter); ok {
		store.batchDB = batchDB
		logging.Info().Msg("STORE: Atomic batch insert support enabled")
	} else {
		logging.Warn().Msg("STORE: Database does not support atomic batch inserts, using individual inserts")
	}

	return store, nil
}

// InsertMediaEvents converts and inserts a batch of media events.
// Uses atomic batch insert when available (all-or-nothing semantics).
// Falls back to individual inserts for backward compatibility.
//
// Returns error if:
//   - Atomic mode: Transaction fails (all events rolled back)
//   - Individual mode: Any single insert fails (partial state possible)
//
// Note: With ON CONFLICT DO NOTHING, duplicate events are silently skipped.
// The batch method returns counts of inserted vs duplicates for auditability.
func (s *DuckDBStore) InsertMediaEvents(ctx context.Context, events []*MediaEvent) error {
	if len(events) == 0 {
		return nil
	}

	startTime := time.Now()
	logging.Trace().Int("count", len(events)).Msg("STORE: Inserting batch of events")

	// TRACING: Log each event being inserted
	for i, event := range events {
		if i < 5 || i >= len(events)-3 {
			logging.Trace().
				Int("index", i+1).
				Int("total", len(events)).
				Str("session_key", event.SessionKey).
				Str("event_id", event.EventID).
				Str("username", event.Username).
				Msg("STORE: Event")
		} else if i == 5 {
			logging.Trace().Int("omitted", len(events)-8).Msg("STORE: Events omitted")
		}
	}

	// Convert all events to PlaybackEvents first
	playbackEvents := make([]*models.PlaybackEvent, len(events))
	for i, event := range events {
		playbackEvents[i] = s.mediaEventToPlaybackEvent(event)
	}

	// Use atomic batch insert if available
	if s.batchDB != nil {
		inserted, duplicates, err := s.batchDB.InsertPlaybackEventsBatch(ctx, playbackEvents)
		if err != nil {
			logging.Trace().
				Dur("elapsed", time.Since(startTime)).
				Err(err).
				Int("rolled_back", len(events)).
				Msg("STORE: Atomic batch FAILED")
			return fmt.Errorf("atomic batch insert failed: %w", err)
		}

		logging.Trace().
			Int("inserted", inserted).
			Int("duplicates", duplicates).
			Int("total", len(events)).
			Dur("elapsed", time.Since(startTime)).
			Msg("STORE: Atomic batch SUCCESS")
		return nil
	}

	// Fallback to individual inserts (non-atomic, for backward compatibility)
	logging.Warn().Msg("STORE: Using non-atomic individual inserts (partial state possible on failure)")

	for i, playback := range playbackEvents {
		if err := s.db.InsertPlaybackEvent(playback); err != nil {
			logging.Error().
				Int("index", i).
				Int("total", len(events)).
				Dur("elapsed", time.Since(startTime)).
				Err(err).
				Int("committed", i-1).
				Msg("STORE: Non-atomic batch failed")
			return fmt.Errorf("insert event %d (%s): %w", i, events[i].EventID, err)
		}
	}

	logging.Info().
		Int("count", len(events)).
		Dur("elapsed", time.Since(startTime)).
		Msg("STORE: Non-atomic batch complete")
	return nil
}

// mediaEventToPlaybackEvent converts a MediaEvent to a PlaybackEvent.
// This mapping handles the transformation from the NATS event format
// to the database schema format.
//
// CRITICAL (v1.47): The CorrelationKey is mapped for cross-source deduplication.
// If not set on the event, generate it here to ensure database-level dedup works.
//
//nolint:gocyclo // Data mapping function with many optional fields requires conditional checks
func (s *DuckDBStore) mediaEventToPlaybackEvent(event *MediaEvent) *models.PlaybackEvent {
	// Use SessionKey if available, fallback to EventID
	sessionKey := event.SessionKey
	if sessionKey == "" {
		sessionKey = event.EventID
	}

	// Ensure CorrelationKey is set for database-level deduplication (v1.47)
	// This is critical for cross-source dedup after cache clear or restart
	if event.CorrelationKey == "" {
		event.SetCorrelationKey()
	}

	playback := &models.PlaybackEvent{
		// Core identification
		ID:         parseOrGenerateUUID(event.EventID),
		SessionKey: sessionKey,
		StartedAt:  event.StartedAt,
		StoppedAt:  event.StoppedAt,
		Source:     event.Source,
		CreatedAt:  time.Now(),

		// Cross-source deduplication (v1.47)
		// Format: {user_id}:{rating_key}:{machine_id}:{time_bucket}
		CorrelationKey: &event.CorrelationKey,

		// User information
		UserID:   event.UserID,
		Username: event.Username,

		// Exactly-once delivery (v2.1 - ADR-0023)
		// Transaction ID for idempotent inserts during crash recovery
		TransactionID: ptrOrNil(event.TransactionID),

		// Media identification
		MediaType: event.MediaType,
		Title:     event.Title,

		// Network
		IPAddress:    event.IPAddress,
		LocationType: event.LocationType,

		// Client/Player
		Platform: event.Platform,
		Player:   event.Player,

		// Playback metrics
		PercentComplete: event.PercentComplete,
		PausedCounter:   event.PausedCounter,
	}

	// User optional string fields
	if event.FriendlyName != "" {
		playback.FriendlyName = &event.FriendlyName
	}
	if event.UserThumb != "" {
		playback.UserThumb = &event.UserThumb
	}
	if event.Email != "" {
		playback.Email = &event.Email
	}

	// Media optional string fields
	if event.ParentTitle != "" {
		playback.ParentTitle = &event.ParentTitle
	}
	if event.GrandparentTitle != "" {
		playback.GrandparentTitle = &event.GrandparentTitle
	}
	if event.RatingKey != "" {
		playback.RatingKey = &event.RatingKey
	}

	// Media optional integer fields
	if event.Year > 0 {
		playback.Year = &event.Year
	}
	// Note: MediaDuration not mapped - PlaybackEvent doesn't have media duration field

	// Platform optional fields
	if event.PlatformName != "" {
		playback.PlatformName = &event.PlatformName
	}
	if event.PlatformVersion != "" {
		playback.PlatformVersion = &event.PlatformVersion
	}
	if event.Product != "" {
		playback.Product = &event.Product
	}
	if event.ProductVersion != "" {
		playback.ProductVersion = &event.ProductVersion
	}
	if event.Device != "" {
		playback.Device = &event.Device
	}
	if event.MachineID != "" {
		playback.MachineID = &event.MachineID
	}

	// Streaming quality optional fields
	if event.TranscodeDecision != "" {
		playback.TranscodeDecision = &event.TranscodeDecision
	}
	if event.VideoResolution != "" {
		playback.VideoResolution = &event.VideoResolution
	}
	if event.VideoCodec != "" {
		playback.VideoCodec = &event.VideoCodec
	}
	if event.VideoDynamicRange != "" {
		playback.StreamVideoDynamicRange = &event.VideoDynamicRange
	}
	if event.AudioCodec != "" {
		playback.AudioCodec = &event.AudioCodec
	}

	// Optional integer fields
	if event.PlayDuration > 0 {
		playback.PlayDuration = &event.PlayDuration
	}
	if event.AudioChannels > 0 {
		audioChannels := strconv.Itoa(event.AudioChannels)
		playback.AudioChannels = &audioChannels
	}
	if event.StreamBitrate > 0 {
		playback.StreamBitrate = &event.StreamBitrate
	}
	if event.Bandwidth > 0 {
		playback.Bandwidth = &event.Bandwidth
	}

	// Boolean to int conversions for database schema
	if event.Secure {
		secure := 1
		playback.Secure = &secure
	}
	if event.Local {
		local := 1
		playback.Local = &local
	}
	if event.Relayed {
		relayed := 1
		playback.Relayed = &relayed
	}

	return playback
}

// parseOrGenerateUUID attempts to parse the string as a UUID.
// If parsing fails, generates a new UUID.
func parseOrGenerateUUID(s string) uuid.UUID {
	if id, err := uuid.Parse(s); err == nil {
		return id
	}
	return uuid.New()
}

// ptrOrNil returns a pointer to s if non-empty, otherwise nil.
func ptrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
