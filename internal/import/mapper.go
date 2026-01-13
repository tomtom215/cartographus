// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package tautulliimport

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/tomtom215/cartographus/internal/models"
)

// Mapper converts TautulliRecord to PlaybackEvent.
type Mapper struct {
	// source is the data source identifier for imported events
	source string
}

// NewMapper creates a new field mapper.
func NewMapper() *Mapper {
	return &Mapper{
		source: "tautulli-import",
	}
}

// ToPlaybackEvent converts a TautulliRecord to a PlaybackEvent.
// It generates a deterministic UUID based on the session key and started time
// to ensure the same record always produces the same event ID.
func (m *Mapper) ToPlaybackEvent(rec *TautulliRecord) *models.PlaybackEvent {
	event := &models.PlaybackEvent{
		ID:        m.generateDeterministicID(rec),
		Source:    m.source,
		CreatedAt: time.Now(),

		// Core session fields
		SessionKey: rec.SessionKey,
		StartedAt:  rec.StartedAt,

		// User information
		UserID:       rec.UserID,
		Username:     rec.Username,
		FriendlyName: rec.FriendlyName,

		// Network/IP information
		IPAddress:    rec.IPAddress,
		LocationType: rec.LocationType,

		// Media identification
		MediaType:        rec.MediaType,
		Title:            rec.Title,
		ParentTitle:      rec.ParentTitle,
		GrandparentTitle: rec.GrandparentTitle,

		// Client/Player information
		Platform:  rec.Platform,
		Player:    rec.Player,
		Product:   rec.Product,
		MachineID: rec.MachineID,

		// Playback metrics
		PercentComplete: rec.PercentComplete,
		PausedCounter:   rec.PausedCounter,
	}

	// Set stopped time if available
	if !rec.StoppedAt.IsZero() {
		event.StoppedAt = &rec.StoppedAt
	}

	// Set correlation key for deduplication
	correlationKey := m.generateCorrelationKey(rec)
	event.CorrelationKey = &correlationKey

	// Map media metadata fields
	m.mapMediaMetadata(event, rec)

	// Map stream quality fields
	m.mapStreamQuality(event, rec)

	return event
}

// generateDeterministicID creates a deterministic UUID based on session data.
// This ensures that re-importing the same record produces the same event ID,
// which helps with deduplication at the NATS/consumer level.
func (m *Mapper) generateDeterministicID(rec *TautulliRecord) uuid.UUID {
	// Create a unique string from session key and start time
	input := fmt.Sprintf("tautulli-import:%s:%d:%d",
		rec.SessionKey,
		rec.StartedAt.Unix(),
		rec.UserID,
	)

	// Hash to create deterministic bytes
	hash := sha256.Sum256([]byte(input))

	// Use first 16 bytes as UUID - this cannot fail with 16 bytes input
	id, err := uuid.FromBytes(hash[:16])
	if err != nil {
		// Fallback to random UUID if hash conversion fails (should never happen)
		return uuid.New()
	}

	// Set version 5 (SHA-1 based) and variant bits
	id[6] = (id[6] & 0x0f) | 0x50 // Version 5
	id[8] = (id[8] & 0x3f) | 0x80 // Variant 10

	return id
}

// generateCorrelationKey creates a correlation key for cross-source deduplication.
//
// v2.3 Format: {source}:{server_id}:{user_id}:{rating_key}:{machine_id}:{time_bucket}:{session_key}
//
// This matches the format used by MediaEvent.GenerateCorrelationKey() in eventprocessor.
// Cross-source deduplication works via getCrossSourceKey() in handlers.go,
// which extracts the content-based portion (parts 2-6) for matching across sources.
//
// Example: "tautulli-import:default:42:12345:device123:2024-01-15T10:00:00:session-abc"
func (m *Mapper) generateCorrelationKey(rec *TautulliRecord) string {
	// Use exact timestamp (second precision) for correlation key
	// For cross-source matching, identical playbacks have identical started_at
	timeBucket := rec.StartedAt.UTC().Format("2006-01-02T15:04:05")

	// Use rating key if available, otherwise hash the title
	ratingKey := ""
	if rec.RatingKey != nil && *rec.RatingKey != "" {
		ratingKey = *rec.RatingKey
	} else {
		// Fall back to title hash for records without rating key
		hash := sha256.Sum256([]byte(rec.Title))
		ratingKey = hex.EncodeToString(hash[:8])
	}

	// Use machine ID if available
	machineID := "unknown"
	if rec.MachineID != nil && *rec.MachineID != "" {
		machineID = *rec.MachineID
	}

	// ServerID defaults to "default" for imported data
	serverID := "default"

	// SessionKey for uniqueness
	sessionKey := rec.SessionKey
	if sessionKey == "" {
		sessionKey = "unknown"
	}

	return fmt.Sprintf("%s:%s:%d:%s:%s:%s:%s", m.source, serverID, rec.UserID, ratingKey, machineID, timeBucket, sessionKey)
}

// mapMediaMetadata maps media-related fields from the record.
func (m *Mapper) mapMediaMetadata(event *models.PlaybackEvent, rec *TautulliRecord) {
	// Rating keys for binge detection
	event.RatingKey = rec.RatingKey
	event.ParentRatingKey = rec.ParentRatingKey
	event.GrandparentRatingKey = rec.GrandparentRKey

	// Episode/Season numbers for binge detection
	event.MediaIndex = rec.MediaIndex
	event.ParentMediaIndex = rec.ParentMediaIndex

	// Library metadata
	event.SectionID = rec.SectionID
	event.LibraryName = rec.LibraryName
	event.ContentRating = rec.ContentRating
	event.Year = rec.Year

	// External IDs
	event.GUID = rec.GUID

	// Thumbnails
	event.Thumb = rec.Thumb
	event.ParentThumb = rec.ParentThumb
	event.GrandparentThumb = rec.GrandparentThumb

	// Cast and crew
	event.Directors = rec.Directors
	event.Writers = rec.Writers
	event.Actors = rec.Actors
	event.Genres = rec.Genres
	event.Studio = rec.Studio

	// Title variations
	event.FullTitle = rec.FullTitle
	event.OriginalTitle = rec.OriginalTitle
	event.OriginallyAvailableAt = rec.OriginallyAvail
}

// mapStreamQuality maps stream quality fields from the record.
func (m *Mapper) mapStreamQuality(event *models.PlaybackEvent, rec *TautulliRecord) {
	// Source video fields
	event.VideoResolution = rec.VideoResolution
	event.VideoFullResolution = rec.VideoFullResolution
	event.VideoCodec = rec.VideoCodec

	// Source audio fields
	event.AudioCodec = rec.AudioCodec
	event.AudioChannels = rec.AudioChannels

	// Container
	event.Container = rec.Container

	// Bitrate
	event.Bitrate = rec.Bitrate
	event.StreamBitrate = rec.StreamBitrate

	// Transcode decisions
	event.TranscodeDecision = rec.TranscodeDecision
	event.VideoDecision = rec.VideoDecision
	event.AudioDecision = rec.AudioDecision
	event.SubtitleDecision = rec.SubtitleDecision

	// Stream output fields
	event.StreamVideoCodec = rec.StreamVideoCodec
	event.StreamVideoResolution = rec.StreamVideoRes
	event.StreamAudioCodec = rec.StreamAudioCodec
	event.StreamAudioChannels = rec.StreamAudioChannels
}

// ToPlaybackEvents converts a batch of TautulliRecords to PlaybackEvents.
func (m *Mapper) ToPlaybackEvents(records []TautulliRecord) []*models.PlaybackEvent {
	events := make([]*models.PlaybackEvent, len(records))
	for i := range records {
		events[i] = m.ToPlaybackEvent(&records[i])
	}
	return events
}

// ValidateRecord checks if a record has required fields for import.
// Returns an error describing any validation failures.
func (m *Mapper) ValidateRecord(rec *TautulliRecord) error {
	if rec.SessionKey == "" {
		return fmt.Errorf("missing session_key")
	}
	if rec.StartedAt.IsZero() {
		return fmt.Errorf("missing or invalid started timestamp")
	}
	if rec.UserID <= 0 {
		return fmt.Errorf("invalid user_id: %d", rec.UserID)
	}
	if rec.Username == "" {
		return fmt.Errorf("missing username")
	}
	if rec.IPAddress == "" || rec.IPAddress == "N/A" {
		return fmt.Errorf("missing or invalid ip_address")
	}
	if rec.MediaType == "" {
		return fmt.Errorf("missing media_type")
	}
	if rec.Title == "" {
		return fmt.Errorf("missing title")
	}
	return nil
}

// FilterValidRecords filters out invalid records and returns valid ones.
// Also returns a count of skipped records.
func (m *Mapper) FilterValidRecords(records []TautulliRecord) (valid []TautulliRecord, skipped int) {
	for i := range records {
		if err := m.ValidateRecord(&records[i]); err == nil {
			valid = append(valid, records[i])
		} else {
			skipped++
		}
	}
	return valid, skipped
}
