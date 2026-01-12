// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package database

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/tomtom215/cartographus/internal/logging"
	"github.com/tomtom215/cartographus/internal/models"
)

// InsertPlaybackEvent inserts a new playback event into the database with duplicate handling.
//
// Deduplication Strategy (v1.47 - NATS JetStream Phase 2):
//   - Uses INSERT ... ON CONFLICT DO NOTHING (DuckDB-native syntax) to gracefully handle duplicate events
//   - Two unique constraints prevent duplicates:
//     1. correlation_key (primary, for NATS event sourcing cross-source dedup)
//     2. rating_key + user_id + started_at (legacy, for pre-v1.47 events)
//   - If either constraint is violated, the insert is silently ignored (no error)
//   - This allows idempotent event processing after cache clear or restart
//
// Note: Uses UUID primary key so no same-row conflicts possible (no locking needed)
// Updated v1.43: Added 70+ new fields for comprehensive Plex/Tautulli API coverage
// Updated v1.47: Added correlation_key for cross-source deduplication
// Updated v2.0: Migrated from INSERT OR IGNORE (SQLite) to ON CONFLICT DO NOTHING (DuckDB-native)
func (db *DB) InsertPlaybackEvent(event *models.PlaybackEvent) error {
	if event.ID == uuid.Nil {
		event.ID = uuid.New()
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now()
	}

	// DuckDB-native: ON CONFLICT DO NOTHING handles both unique constraint violations
	query := `INSERT INTO playback_events (
		-- Core identification
		id, session_key, started_at, stopped_at, user_id, username, ip_address,
		source, plex_key,
		-- Cross-source deduplication (v1.47)
		correlation_key,
		-- Exactly-once delivery (v2.1 - ADR-0023)
		transaction_id,
		-- User extended fields (v1.43)
		friendly_name, user_thumb, email, ip_address_public,
		-- Media identification
		media_type, title, parent_title, grandparent_title,
		-- Client/Player information
		platform, player, platform_name, platform_version,
		product, product_version, device, machine_id,
		location_type, quality_profile, optimized_version, synced_version,
		-- Playback metrics
		percent_complete, paused_counter, play_duration,
		-- Transcode decisions (v1.43)
		transcode_decision, video_decision, audio_decision, subtitle_decision,
		-- Hardware transcode fields (v1.43 - CRITICAL for GPU monitoring)
		transcode_key, transcode_throttled, transcode_progress, transcode_speed,
		transcode_hw_requested, transcode_hw_decoding, transcode_hw_encoding, transcode_hw_full_pipeline,
		transcode_hw_decode, transcode_hw_decode_title, transcode_hw_encode, transcode_hw_encode_title,
		transcode_container, transcode_video_codec, transcode_audio_codec, transcode_audio_channels,
		-- Source video fields
		video_resolution, video_codec, video_codec_level, video_profile,
		video_scan_type, video_language, aspect_ratio,
		-- HDR/Color metadata (v1.43 - CRITICAL for HDR detection)
		video_color_primaries, video_color_range, video_color_space, video_color_trc, video_chroma_subsampling,
		-- Source audio fields
		audio_codec, audio_profile,
		-- Library metadata
		section_id, library_name, content_rating, year,
		-- Metadata enrichment fields
		rating_key, parent_rating_key, grandparent_rating_key,
		media_index, parent_media_index,
		guid, original_title, full_title, originally_available_at,
		watched_status, thumb,
		-- Cast and crew
		directors, writers, actors, genres,
		-- Stream output fields (v1.43 extended)
		stream_container, stream_bitrate,
		stream_video_codec, stream_video_resolution, stream_video_decision,
		stream_video_bitrate, stream_video_width, stream_video_height, stream_video_dynamic_range,
		stream_audio_codec, stream_audio_channels, stream_audio_bitrate, stream_audio_decision,
		stream_audio_channel_layout,
		-- Subtitle fields (v1.43 extended)
		stream_subtitle_codec, stream_subtitle_language,
		subtitle_container, subtitle_forced, subtitle_location,
		-- Source audio details
		audio_channels, audio_channel_layout, audio_bitrate, audio_bitrate_mode, audio_sample_rate, audio_language,
		-- Source video details
		video_dynamic_range, video_framerate, video_bitrate, video_bit_depth, video_ref_frames, video_width, video_height,
		-- Container and subtitle legacy
		container, subtitle_codec, subtitle_language, subtitles,
		-- Connection and network (v1.43 extended)
		secure, relayed, relay, local, bandwidth, location, bandwidth_lan, bandwidth_wan,
		-- File metadata (v1.43 extended)
		file_size, bitrate, file,
		-- Bitrate analytics
		source_bitrate, transcode_bitrate, network_bandwidth,
		-- Thumbnails and art (v1.43)
		parent_thumb, grandparent_thumb, art, grandparent_art,
		-- Additional GUIDs (v1.43)
		parent_guid, grandparent_guid,
		-- Timestamp
		created_at
	) VALUES (
		?, ?, ?, ?, ?, ?, ?, ?, ?, ?,
		?, ?, ?, ?, ?, ?, ?, ?, ?, ?,
		?, ?, ?, ?, ?, ?, ?, ?, ?, ?,
		?, ?, ?, ?, ?, ?, ?, ?, ?, ?,
		?, ?, ?, ?, ?, ?, ?, ?, ?, ?,
		?, ?, ?, ?, ?, ?, ?, ?, ?, ?,
		?, ?, ?, ?, ?, ?, ?, ?, ?, ?,
		?, ?, ?, ?, ?, ?, ?, ?, ?, ?,
		?, ?, ?, ?, ?, ?, ?, ?, ?, ?,
		?, ?, ?, ?, ?, ?, ?, ?, ?, ?,
		?, ?, ?, ?, ?, ?, ?, ?, ?, ?,
		?, ?, ?, ?, ?, ?, ?, ?, ?, ?,
		?, ?, ?, ?, ?, ?, ?, ?, ?, ?,
		?, ?, ?, ?, ?, ?, ?, ?, ?, ?,
		?, ?, ?, ?
	) ON CONFLICT DO NOTHING`

	result, err := db.conn.ExecContext(context.Background(), query,
		// Core identification
		event.ID, event.SessionKey, event.StartedAt, event.StoppedAt,
		event.UserID, event.Username, event.IPAddress,
		event.Source, event.PlexKey,
		// Cross-source deduplication (v1.47)
		event.CorrelationKey,
		// Exactly-once delivery (v2.1 - ADR-0023)
		event.TransactionID,
		// User extended fields
		event.FriendlyName, event.UserThumb, event.Email, event.IPAddressPublic,
		// Media identification
		event.MediaType, event.Title, event.ParentTitle, event.GrandparentTitle,
		// Client/Player information
		event.Platform, event.Player, event.PlatformName, event.PlatformVersion,
		event.Product, event.ProductVersion, event.Device, event.MachineID,
		event.LocationType, event.QualityProfile, event.OptimizedVersion, event.SyncedVersion,
		// Playback metrics
		event.PercentComplete, event.PausedCounter, event.PlayDuration,
		// Transcode decisions
		event.TranscodeDecision, event.VideoDecision, event.AudioDecision, event.SubtitleDecision,
		// Hardware transcode fields
		event.TranscodeKey, event.TranscodeThrottled, event.TranscodeProgress, event.TranscodeSpeed,
		event.TranscodeHWRequested, event.TranscodeHWDecoding, event.TranscodeHWEncoding, event.TranscodeHWFullPipeline,
		event.TranscodeHWDecode, event.TranscodeHWDecodeTitle, event.TranscodeHWEncode, event.TranscodeHWEncodeTitle,
		event.TranscodeContainer, event.TranscodeVideoCodec, event.TranscodeAudioCodec, event.TranscodeAudioChannels,
		// Source video fields
		event.VideoResolution, event.VideoCodec, event.VideoCodecLevel, event.VideoProfile,
		event.VideoScanType, event.VideoLanguage, event.AspectRatio,
		// HDR/Color metadata
		event.VideoColorPrimaries, event.VideoColorRange, event.VideoColorSpace, event.VideoColorTrc, event.VideoChromaSubsampling,
		// Source audio fields
		event.AudioCodec, event.AudioProfile,
		// Library metadata
		event.SectionID, event.LibraryName, event.ContentRating, event.Year,
		// Metadata enrichment fields
		event.RatingKey, event.ParentRatingKey, event.GrandparentRatingKey,
		event.MediaIndex, event.ParentMediaIndex,
		event.Guid, event.OriginalTitle, event.FullTitle, event.OriginallyAvailableAt,
		event.WatchedStatus, event.Thumb,
		// Cast and crew
		event.Directors, event.Writers, event.Actors, event.Genres,
		// Stream output fields
		event.StreamContainer, event.StreamBitrate,
		event.StreamVideoCodec, event.StreamVideoResolution, event.StreamVideoDecision,
		event.StreamVideoBitrate, event.StreamVideoWidth, event.StreamVideoHeight, event.StreamVideoDynamicRange,
		event.StreamAudioCodec, event.StreamAudioChannels, event.StreamAudioBitrate, event.StreamAudioDecision,
		event.StreamAudioChannelLayout,
		// Subtitle fields
		event.StreamSubtitleCodec, event.StreamSubtitleLanguage,
		event.SubtitleContainer, event.SubtitleForced, event.SubtitleLocation,
		// Source audio details
		event.AudioChannels, event.AudioChannelLayout, event.AudioBitrate, event.AudioBitrateMode, event.AudioSampleRate, event.AudioLanguage,
		// Source video details
		event.VideoDynamicRange, event.VideoFrameRate, event.VideoBitrate, event.VideoBitDepth, event.VideoRefFrames, event.VideoWidth, event.VideoHeight,
		// Container and subtitle legacy
		event.Container, event.SubtitleCodec, event.SubtitleLanguage, event.Subtitles,
		// Connection and network
		event.Secure, event.Relayed, event.Relay, event.Local, event.Bandwidth, event.Location, event.BandwidthLAN, event.BandwidthWAN,
		// File metadata
		event.FileSize, event.Bitrate, event.File,
		// Bitrate analytics
		event.SourceBitrate, event.TranscodeBitrate, event.NetworkBandwidth,
		// Thumbnails and art
		event.ParentThumb, event.GrandparentThumb, event.Art, event.GrandparentArt,
		// Additional GUIDs
		event.ParentGuid, event.GrandparentGuid,
		// Timestamp
		event.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to insert playback event: %w", err)
	}

	// Check if row was actually inserted (affected rows > 0)
	// With ON CONFLICT DO NOTHING, no error is returned for duplicates
	rowsAffected, err := result.RowsAffected()
	if err == nil && rowsAffected == 0 {
		// Log duplicate detection for debugging
		ratingKey := "<nil>"
		if event.RatingKey != nil {
			ratingKey = *event.RatingKey
		}
		logging.Debug().
			Int("user_id", event.UserID).
			Str("rating_key", ratingKey).
			Str("started_at", event.StartedAt.Format("2006-01-02T15:04:05")).
			Msg("Duplicate detected")
	}

	// MEDIUM-1: Increment data version to invalidate tile cache
	db.IncrementDataVersion()

	return nil
}

// InsertPlaybackEventsBatch atomically inserts a batch of playback events.
// Uses a database transaction to ensure all-or-nothing semantics.
//
// Returns:
//   - inserted: number of events successfully inserted
//   - duplicates: number of events skipped due to unique constraint violations
//   - err: error if the transaction failed (all events are rolled back)
//
// CRITICAL (v2.3): This method guarantees ACID compliance:
//   - Atomicity: All inserts succeed or all are rolled back
//   - Consistency: Unique constraints are enforced
//   - Isolation: Transaction provides snapshot isolation
//   - Durability: Committed changes are persistent
//
// Performance: Uses prepared statements within the transaction for efficiency.
//
//nolint:gocyclo // Complexity is inherent to ACID-compliant batch insert with field mapping
func (db *DB) InsertPlaybackEventsBatch(ctx context.Context, events []*models.PlaybackEvent) (inserted int, duplicates int, err error) {
	if len(events) == 0 {
		return 0, 0, nil
	}

	// Start transaction
	tx, err := db.conn.BeginTx(ctx, nil)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Ensure transaction is finalized
	defer func() {
		if err != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				logging.Error().
					Err(rbErr).
					AnErr("original_error", err).
					Msg("Transaction rollback failed")
			}
		}
	}()

	// Prepare statement within transaction for efficiency
	// DuckDB-native: ON CONFLICT DO NOTHING handles both unique constraint violations
	query := `INSERT INTO playback_events (
		-- Core identification
		id, session_key, started_at, stopped_at, user_id, username, ip_address,
		source, plex_key,
		-- Cross-source deduplication (v1.47)
		correlation_key,
		-- Exactly-once delivery (v2.1 - ADR-0023)
		transaction_id,
		-- User extended fields (v1.43)
		friendly_name, user_thumb, email, ip_address_public,
		-- Media identification
		media_type, title, parent_title, grandparent_title,
		-- Client/Player information
		platform, player, platform_name, platform_version,
		product, product_version, device, machine_id,
		location_type, quality_profile, optimized_version, synced_version,
		-- Playback metrics
		percent_complete, paused_counter, play_duration,
		-- Transcode decisions (v1.43)
		transcode_decision, video_decision, audio_decision, subtitle_decision,
		-- Hardware transcode fields (v1.43)
		transcode_key, transcode_throttled, transcode_progress, transcode_speed,
		transcode_hw_requested, transcode_hw_decoding, transcode_hw_encoding, transcode_hw_full_pipeline,
		transcode_hw_decode, transcode_hw_decode_title, transcode_hw_encode, transcode_hw_encode_title,
		transcode_container, transcode_video_codec, transcode_audio_codec, transcode_audio_channels,
		-- Source video fields
		video_resolution, video_codec, video_codec_level, video_profile,
		video_scan_type, video_language, aspect_ratio,
		-- HDR/Color metadata
		video_color_primaries, video_color_range, video_color_space, video_color_trc, video_chroma_subsampling,
		-- Source audio fields
		audio_codec, audio_profile,
		-- Library metadata
		section_id, library_name, content_rating, year,
		-- Metadata enrichment fields
		rating_key, parent_rating_key, grandparent_rating_key,
		media_index, parent_media_index,
		guid, original_title, full_title, originally_available_at,
		watched_status, thumb,
		-- Cast and crew
		directors, writers, actors, genres,
		-- Stream output fields
		stream_container, stream_bitrate,
		stream_video_codec, stream_video_resolution, stream_video_decision,
		stream_video_bitrate, stream_video_width, stream_video_height, stream_video_dynamic_range,
		stream_audio_codec, stream_audio_channels, stream_audio_bitrate, stream_audio_decision,
		stream_audio_channel_layout,
		-- Subtitle fields
		stream_subtitle_codec, stream_subtitle_language,
		subtitle_container, subtitle_forced, subtitle_location,
		-- Source audio details
		audio_channels, audio_channel_layout, audio_bitrate, audio_bitrate_mode, audio_sample_rate, audio_language,
		-- Source video details
		video_dynamic_range, video_framerate, video_bitrate, video_bit_depth, video_ref_frames, video_width, video_height,
		-- Container and subtitle legacy
		container, subtitle_codec, subtitle_language, subtitles,
		-- Connection and network
		secure, relayed, relay, local, bandwidth, location, bandwidth_lan, bandwidth_wan,
		-- File metadata
		file_size, bitrate, file,
		-- Bitrate analytics
		source_bitrate, transcode_bitrate, network_bandwidth,
		-- Thumbnails and art
		parent_thumb, grandparent_thumb, art, grandparent_art,
		-- Additional GUIDs
		parent_guid, grandparent_guid,
		-- Timestamp
		created_at
	) VALUES (
		?, ?, ?, ?, ?, ?, ?, ?, ?, ?,
		?, ?, ?, ?, ?, ?, ?, ?, ?, ?,
		?, ?, ?, ?, ?, ?, ?, ?, ?, ?,
		?, ?, ?, ?, ?, ?, ?, ?, ?, ?,
		?, ?, ?, ?, ?, ?, ?, ?, ?, ?,
		?, ?, ?, ?, ?, ?, ?, ?, ?, ?,
		?, ?, ?, ?, ?, ?, ?, ?, ?, ?,
		?, ?, ?, ?, ?, ?, ?, ?, ?, ?,
		?, ?, ?, ?, ?, ?, ?, ?, ?, ?,
		?, ?, ?, ?, ?, ?, ?, ?, ?, ?,
		?, ?, ?, ?, ?, ?, ?, ?, ?, ?,
		?, ?, ?, ?, ?, ?, ?, ?, ?, ?,
		?, ?, ?, ?, ?, ?, ?, ?, ?, ?,
		?, ?, ?, ?, ?, ?, ?, ?, ?, ?,
		?, ?, ?, ?
	) ON CONFLICT DO NOTHING`

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer func() {
		if closeErr := stmt.Close(); closeErr != nil {
			logging.Warn().Err(closeErr).Msg("Failed to close prepared statement")
		}
	}()

	// Insert each event and track results
	inserted = 0
	duplicates = 0

	for i, event := range events {
		// Ensure ID and CreatedAt are set
		if event.ID == uuid.Nil {
			event.ID = uuid.New()
		}
		if event.CreatedAt.IsZero() {
			event.CreatedAt = time.Now()
		}

		result, execErr := stmt.ExecContext(ctx,
			// Core identification
			event.ID, event.SessionKey, event.StartedAt, event.StoppedAt,
			event.UserID, event.Username, event.IPAddress,
			event.Source, event.PlexKey,
			// Cross-source deduplication
			event.CorrelationKey,
			// Exactly-once delivery
			event.TransactionID,
			// User extended fields
			event.FriendlyName, event.UserThumb, event.Email, event.IPAddressPublic,
			// Media identification
			event.MediaType, event.Title, event.ParentTitle, event.GrandparentTitle,
			// Client/Player information
			event.Platform, event.Player, event.PlatformName, event.PlatformVersion,
			event.Product, event.ProductVersion, event.Device, event.MachineID,
			event.LocationType, event.QualityProfile, event.OptimizedVersion, event.SyncedVersion,
			// Playback metrics
			event.PercentComplete, event.PausedCounter, event.PlayDuration,
			// Transcode decisions
			event.TranscodeDecision, event.VideoDecision, event.AudioDecision, event.SubtitleDecision,
			// Hardware transcode fields
			event.TranscodeKey, event.TranscodeThrottled, event.TranscodeProgress, event.TranscodeSpeed,
			event.TranscodeHWRequested, event.TranscodeHWDecoding, event.TranscodeHWEncoding, event.TranscodeHWFullPipeline,
			event.TranscodeHWDecode, event.TranscodeHWDecodeTitle, event.TranscodeHWEncode, event.TranscodeHWEncodeTitle,
			event.TranscodeContainer, event.TranscodeVideoCodec, event.TranscodeAudioCodec, event.TranscodeAudioChannels,
			// Source video fields
			event.VideoResolution, event.VideoCodec, event.VideoCodecLevel, event.VideoProfile,
			event.VideoScanType, event.VideoLanguage, event.AspectRatio,
			// HDR/Color metadata
			event.VideoColorPrimaries, event.VideoColorRange, event.VideoColorSpace, event.VideoColorTrc, event.VideoChromaSubsampling,
			// Source audio fields
			event.AudioCodec, event.AudioProfile,
			// Library metadata
			event.SectionID, event.LibraryName, event.ContentRating, event.Year,
			// Metadata enrichment fields
			event.RatingKey, event.ParentRatingKey, event.GrandparentRatingKey,
			event.MediaIndex, event.ParentMediaIndex,
			event.Guid, event.OriginalTitle, event.FullTitle, event.OriginallyAvailableAt,
			event.WatchedStatus, event.Thumb,
			// Cast and crew
			event.Directors, event.Writers, event.Actors, event.Genres,
			// Stream output fields
			event.StreamContainer, event.StreamBitrate,
			event.StreamVideoCodec, event.StreamVideoResolution, event.StreamVideoDecision,
			event.StreamVideoBitrate, event.StreamVideoWidth, event.StreamVideoHeight, event.StreamVideoDynamicRange,
			event.StreamAudioCodec, event.StreamAudioChannels, event.StreamAudioBitrate, event.StreamAudioDecision,
			event.StreamAudioChannelLayout,
			// Subtitle fields
			event.StreamSubtitleCodec, event.StreamSubtitleLanguage,
			event.SubtitleContainer, event.SubtitleForced, event.SubtitleLocation,
			// Source audio details
			event.AudioChannels, event.AudioChannelLayout, event.AudioBitrate, event.AudioBitrateMode, event.AudioSampleRate, event.AudioLanguage,
			// Source video details
			event.VideoDynamicRange, event.VideoFrameRate, event.VideoBitrate, event.VideoBitDepth, event.VideoRefFrames, event.VideoWidth, event.VideoHeight,
			// Container and subtitle legacy
			event.Container, event.SubtitleCodec, event.SubtitleLanguage, event.Subtitles,
			// Connection and network
			event.Secure, event.Relayed, event.Relay, event.Local, event.Bandwidth, event.Location, event.BandwidthLAN, event.BandwidthWAN,
			// File metadata
			event.FileSize, event.Bitrate, event.File,
			// Bitrate analytics
			event.SourceBitrate, event.TranscodeBitrate, event.NetworkBandwidth,
			// Thumbnails and art
			event.ParentThumb, event.GrandparentThumb, event.Art, event.GrandparentArt,
			// Additional GUIDs
			event.ParentGuid, event.GrandparentGuid,
			// Timestamp
			event.CreatedAt,
		)

		if execErr != nil {
			err = fmt.Errorf("failed to insert event %d (session=%s): %w", i, event.SessionKey, execErr)
			return 0, 0, err
		}

		// Check if row was actually inserted (affected rows > 0)
		rowsAffected, rowsErr := result.RowsAffected()
		if rowsErr != nil {
			err = fmt.Errorf("failed to get rows affected for event %d: %w", i, rowsErr)
			return 0, 0, err
		}

		if rowsAffected > 0 {
			inserted++
		} else {
			duplicates++
			// Log duplicate for debugging
			ratingKey := "<nil>"
			if event.RatingKey != nil {
				ratingKey = *event.RatingKey
			}
			correlationKey := "<nil>"
			if event.CorrelationKey != nil {
				correlationKey = *event.CorrelationKey
			}
			logging.Debug().
				Str("session", event.SessionKey).
				Int("user_id", event.UserID).
				Str("rating_key", ratingKey).
				Str("correlation_key", correlationKey).
				Msg("Batch duplicate detected")
		}
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return 0, 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// MEDIUM-1: Increment data version to invalidate tile cache (only if any inserts)
	if inserted > 0 {
		db.IncrementDataVersion()
	}

	logging.Debug().
		Int("inserted", inserted).
		Int("duplicates", duplicates).
		Int("total", len(events)).
		Msg("Batch transaction committed")

	return inserted, duplicates, nil
}

// GetLastPlaybackTime retrieves the timestamp of the most recent playback event.
//
// This method is used for dashboard "last activity" displays and sync status indicators.
// It returns the MAX(started_at) value across all playback events.
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//
// Returns:
//   - Pointer to time.Time with most recent playback timestamp
//   - nil if no playback events exist in database
//   - error if query fails
//
// Performance: ~1-2ms with idx_playback_started_at index.
//
// Called frequently by WebSocket broadcasts after sync completion.
func (db *DB) GetLastPlaybackTime(ctx context.Context) (*time.Time, error) {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	var lastPlayback *time.Time
	err := db.conn.QueryRowContext(ctx, `SELECT MAX(started_at) FROM playback_events`).Scan(&lastPlayback)
	if err != nil {
		return nil, fmt.Errorf("failed to get last playback time: %w", err)
	}
	return lastPlayback, nil
}

// GetPlaybackEvents retrieves playback events with pagination and sorting.
//
// This method returns playback events ordered by started_at DESC (most recent first)
// with support for offset-based pagination. It returns a subset of fields optimized
// for list views (excludes less frequently needed metadata fields).
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - limit: Maximum number of events to return (typical values: 50-100)
//   - offset: Number of events to skip for pagination (offset = page * limit)
//
// Returns:
//   - Array of PlaybackEvent structs with 26 core fields populated
//   - Empty array if no events match pagination window
//   - error if query fails
//
// Performance: ~5-10ms for 100 events with idx_playback_started_at index.
//
// Pagination Example:
//
//	// Page 1 (events 0-99)
//	events, err := db.GetPlaybackEvents(ctx, 100, 0)
//	// Page 2 (events 100-199)
//	events, err := db.GetPlaybackEvents(ctx, 100, 100)
//
// Note: For large offsets (>10,000), consider using cursor-based pagination
// with WHERE started_at < ? for better performance.
func (db *DB) GetPlaybackEvents(ctx context.Context, limit, offset int) ([]models.PlaybackEvent, error) {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	query := `
	SELECT id, session_key, started_at, stopped_at, user_id, username, ip_address,
		media_type, title, parent_title, grandparent_title, platform, player,
		location_type, percent_complete, paused_counter, created_at,
		transcode_decision, video_resolution, video_codec, audio_codec,
		section_id, library_name, content_rating, play_duration, year
	FROM playback_events
	ORDER BY started_at DESC
	LIMIT ? OFFSET ?`

	rows, err := db.conn.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query playback events: %w", err)
	}
	defer rows.Close()

	var events []models.PlaybackEvent
	for rows.Next() {
		var e models.PlaybackEvent
		err := rows.Scan(
			&e.ID, &e.SessionKey, &e.StartedAt, &e.StoppedAt, &e.UserID,
			&e.Username, &e.IPAddress, &e.MediaType, &e.Title, &e.ParentTitle,
			&e.GrandparentTitle, &e.Platform, &e.Player, &e.LocationType,
			&e.PercentComplete, &e.PausedCounter, &e.CreatedAt,
			&e.TranscodeDecision, &e.VideoResolution, &e.VideoCodec, &e.AudioCodec,
			&e.SectionID, &e.LibraryName, &e.ContentRating, &e.PlayDuration, &e.Year,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan playback event: %w", err)
		}
		events = append(events, e)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating playback events: %w", err)
	}

	return events, nil
}

// GetPlaybackEventsWithCursor retrieves playback events using cursor-based pagination.
//
// Cursor-based pagination is more efficient than offset-based for large datasets because
// it uses an index seek instead of scanning and skipping rows. The cursor encodes the
// (started_at, id) pair of the last item, allowing direct seeking to the next page.
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - limit: Maximum number of events to return (1-1000)
//   - cursor: Optional cursor from previous response (nil for first page)
//
// Returns:
//   - events: Slice of playback events for this page
//   - nextCursor: Cursor for next page (nil if no more results)
//   - hasMore: True if there are more results beyond this page
//   - error: Database or scan error
//
// Performance: O(limit) with index on (started_at DESC, id DESC) vs O(offset+limit) for offset pagination.
// For page 1000 with 100 items per page, cursor is ~1000x faster than offset.
func (db *DB) GetPlaybackEventsWithCursor(ctx context.Context, limit int, cursor *models.PlaybackCursor) ([]models.PlaybackEvent, *models.PlaybackCursor, bool, error) {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	var query string
	var args []interface{}

	// Fetch one extra to determine if there are more results
	fetchLimit := limit + 1

	if cursor == nil {
		// First page - no cursor
		query = `
		SELECT id, session_key, started_at, stopped_at, user_id, username, ip_address,
			media_type, title, parent_title, grandparent_title, platform, player,
			location_type, percent_complete, paused_counter, created_at,
			transcode_decision, video_resolution, video_codec, audio_codec,
			section_id, library_name, content_rating, play_duration, year
		FROM playback_events
		ORDER BY started_at DESC, id DESC
		LIMIT ?`
		args = []interface{}{fetchLimit}
	} else {
		// Subsequent page - use cursor for efficient seeking
		// Uses (started_at, id) composite for deterministic ordering
		// Validate the cursor ID is a valid UUID format
		if _, err := uuid.Parse(cursor.ID); err != nil {
			return nil, nil, false, fmt.Errorf("invalid cursor ID format: %w", err)
		}

		// Use explicit CAST to UUID in SQL because DuckDB's Go driver passes
		// uuid.UUID as VARCHAR in tuple comparisons, causing type mismatch errors
		query = `
		SELECT id, session_key, started_at, stopped_at, user_id, username, ip_address,
			media_type, title, parent_title, grandparent_title, platform, player,
			location_type, percent_complete, paused_counter, created_at,
			transcode_decision, video_resolution, video_codec, audio_codec,
			section_id, library_name, content_rating, play_duration, year
		FROM playback_events
		WHERE (started_at, id) < (?, CAST(? AS UUID))
		ORDER BY started_at DESC, id DESC
		LIMIT ?`
		args = []interface{}{cursor.StartedAt, cursor.ID, fetchLimit}
	}

	rows, err := db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, nil, false, fmt.Errorf("failed to query playback events with cursor: %w", err)
	}
	defer rows.Close()

	var events []models.PlaybackEvent
	for rows.Next() {
		var e models.PlaybackEvent
		err := rows.Scan(
			&e.ID, &e.SessionKey, &e.StartedAt, &e.StoppedAt, &e.UserID,
			&e.Username, &e.IPAddress, &e.MediaType, &e.Title, &e.ParentTitle,
			&e.GrandparentTitle, &e.Platform, &e.Player, &e.LocationType,
			&e.PercentComplete, &e.PausedCounter, &e.CreatedAt,
			&e.TranscodeDecision, &e.VideoResolution, &e.VideoCodec, &e.AudioCodec,
			&e.SectionID, &e.LibraryName, &e.ContentRating, &e.PlayDuration, &e.Year,
		)
		if err != nil {
			return nil, nil, false, fmt.Errorf("failed to scan playback event: %w", err)
		}
		events = append(events, e)
	}

	if err = rows.Err(); err != nil {
		return nil, nil, false, fmt.Errorf("error iterating playback events: %w", err)
	}

	// Determine if there are more results
	hasMore := len(events) > limit
	if hasMore {
		// Remove the extra item we fetched
		events = events[:limit]
	}

	// Build next cursor from last item
	var nextCursor *models.PlaybackCursor
	if hasMore && len(events) > 0 {
		lastEvent := events[len(events)-1]
		nextCursor = &models.PlaybackCursor{
			StartedAt: lastEvent.StartedAt,
			ID:        lastEvent.ID.String(),
		}
	}

	return events, nextCursor, hasMore, nil
}

// TransactionIDExists checks if a playback event with the given transaction ID already exists.
// This is used by the Consumer WAL crash recovery to detect events that were already
// committed to DuckDB before confirming the WAL entry.
//
// Parameters:
//   - ctx: Context for cancellation
//   - transactionID: Consumer WAL transaction ID to check
//
// Returns:
//   - true if the transaction ID exists in playback_events table
//   - false if not found
//   - error if query fails
//
// Performance: ~0.5-1ms with idx_playback_transaction_id index.
func (db *DB) TransactionIDExists(ctx context.Context, transactionID string) (bool, error) {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM playback_events WHERE transaction_id = ?)`
	err := db.conn.QueryRowContext(ctx, query, transactionID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check transaction ID existence: %w", err)
	}
	return exists, nil
}

// InsertFailedEvent inserts a failed event into the persistent DLQ.
// This is called when an event exceeds the maximum retry count in the Consumer WAL.
//
// Parameters:
//   - event: FailedEvent struct with all failure details
//
// Returns:
//   - error if insert fails
//
// The failed_events table provides:
//   - Persistent storage for events that couldn't be inserted into playback_events
//   - Full event payload for manual recovery
//   - Failure reason and layer for debugging
//   - Status tracking for resolution workflow
func (db *DB) InsertFailedEvent(event *models.FailedEvent) error {
	return db.InsertFailedEventWithContext(context.Background(), event)
}

// InsertFailedEventWithContext inserts a failed event into the persistent DLQ with context support.
func (db *DB) InsertFailedEventWithContext(ctx context.Context, event *models.FailedEvent) error {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	if event.ID == uuid.Nil {
		event.ID = uuid.New()
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now()
	}
	if event.UpdatedAt.IsZero() {
		event.UpdatedAt = time.Now()
	}
	if event.FailedAt.IsZero() {
		event.FailedAt = time.Now()
	}
	if event.Status == "" {
		event.Status = "pending"
	}

	query := `INSERT INTO failed_events (
		id, transaction_id, event_id, session_key, correlation_key, source,
		event_payload, failed_at, failure_reason, failure_layer, last_error,
		retry_count, last_retry_at, max_retries_exceeded, status,
		resolved_at, resolved_by, resolution_notes, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := db.conn.ExecContext(ctx, query,
		event.ID, event.TransactionID, event.EventID, event.SessionKey, event.CorrelationKey, event.Source,
		event.EventPayload, event.FailedAt, event.FailureReason, event.FailureLayer, event.LastError,
		event.RetryCount, event.LastRetryAt, event.MaxRetriesExceeded, event.Status,
		event.ResolvedAt, event.ResolvedBy, event.ResolutionNotes, event.CreatedAt, event.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert failed event: %w", err)
	}
	return nil
}

// SessionKeyExists checks if a playback event with the given session key already exists.
//
// This method is used during sync operations to prevent duplicate insertions of the
// same playback event. Tautulli session keys are unique identifiers for each playback session.
//
// Parameters:
//   - ctx: Context for cancellation
//   - sessionKey: Tautulli session key to check (e.g., "abc123xyz")
//
// Returns:
//   - true if the session key exists in playback_events table
//   - false if not found
//   - error if query fails
//
// Performance: ~0.5-1ms with idx_playback_session_key index.
//
// This is called for every record during sync (potentially 1000s of times),
// so performance is critical.
func (db *DB) SessionKeyExists(ctx context.Context, sessionKey string) (bool, error) {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM playback_events WHERE session_key = ?)`
	err := db.conn.QueryRowContext(ctx, query, sessionKey).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check session key existence: %w", err)
	}
	return exists, nil
}
