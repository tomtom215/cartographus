// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package models defines data structures used throughout the Cartographus application.
// These models represent playback events, geolocation data, analytics results, and API responses.

package models

import (
	"time"

	"github.com/google/uuid"
)

// PlaybackEvent represents a single media playback session from Tautulli or Plex.
//
// This is the core data model storing all information about a user's playback activity.
// Events are created during sync from Tautulli history and enriched with geolocation data.
//
// Key Fields:
//   - ID: Unique UUID for each playback event (not session_key, which can repeat)
//   - SessionKey: Tautulli session identifier (used for deduplication)
//   - Source: Data source ("tautulli" for complete data, "plex" for historical)
//   - UserID/Username: User identification
//   - IPAddress/IPAddressPublic: Client IPs for geolocation lookup
//   - MediaType: "movie", "episode", or "track"
//   - Title/ParentTitle/GrandparentTitle: Media hierarchy (episode -> season -> show)
//   - PercentComplete: Playback progress (0-100)
//   - TranscodeDecision: "direct play", "direct stream", or "transcode"
//
// Metadata Fields (for binge detection):
//   - RatingKey: Plex media identifier for content correlation
//   - MediaIndex: Episode number within season
//   - ParentMediaIndex: Season number
//
// Hardware Transcode Fields (CRITICAL for GPU utilization monitoring):
//   - TranscodeHWRequested/Decoding/Encoding: Hardware acceleration flags
//   - TranscodeHWDecode/Encode: Codec names (e.g., "h264_nvdec")
//   - TranscodeHWDecodeTitle/EncodeTitle: Human-readable names (e.g., "NVIDIA NVENC")
//
// Streaming Quality Fields:
//   - StreamVideoResolution, StreamVideoDecision: Actual streamed quality
//   - StreamBitrate: Effective streaming bitrate
//   - SourceBitrate, TranscodeBitrate, NetworkBandwidth: Bandwidth metrics
//
// JSON serialization uses omitempty for optional pointer fields to minimize
// response payload size.
type PlaybackEvent struct {
	ID uuid.UUID `json:"id"`

	// Data source tracking (v1.37) - Enables hybrid Plex + Tautulli architecture
	Source  string  `json:"source"`             // 'tautulli', 'plex', 'jellyfin', or 'emby'
	PlexKey *string `json:"plex_key,omitempty"` // Plex metadata rating key for correlation

	// Multi-server support (v2.0 - Phase 0.6)
	// ServerID uniquely identifies the source server instance, enabling:
	// - Multiple servers of the same type (e.g., multiple Plex servers)
	// - Cross-server deduplication
	// - Per-server analytics and filtering
	ServerID *string `json:"server_id,omitempty"` // Unique identifier for source server

	// Cross-source deduplication (v1.47 - NATS JetStream Phase 2, enhanced v2.0)
	// Format: {source}:{server_id}:{user_id}:{rating_key}:{machine_id}:{time_bucket}
	// Enables deduplication across Plex webhook, Tautulli sync, Jellyfin, and Emby events
	CorrelationKey *string `json:"correlation_key,omitempty"`

	// Exactly-once delivery (v2.1 - ADR-0023: Consumer-Side WAL)
	// Format: {source}:{event_id}:{timestamp_nano}
	// Used by Consumer WAL for idempotent DuckDB inserts during crash recovery
	TransactionID *string `json:"transaction_id,omitempty"`

	// Session identification
	SessionKey string     `json:"session_key"`
	StartedAt  time.Time  `json:"started_at"`
	StoppedAt  *time.Time `json:"stopped_at,omitempty"`

	// Grouped playback fields (v1.46 - API Coverage Expansion)
	GroupCount *int    `json:"group_count,omitempty"` // Number of grouped entries (for grouped view)
	GroupIDs   *string `json:"group_ids,omitempty"`   // Comma-separated grouped IDs
	State      *string `json:"state,omitempty"`       // Playback state at end of session (null for stopped)

	// User information
	UserID       int     `json:"user_id"`
	Username     string  `json:"username"`
	FriendlyName *string `json:"friendly_name,omitempty"` // Display name
	UserThumb    *string `json:"user_thumb,omitempty"`    // User avatar URL
	Email        *string `json:"email,omitempty"`         // User email

	// User permission/status fields (v1.44 - API Coverage Expansion)
	IsAdmin      *int `json:"is_admin,omitempty"`      // 0 or 1 - Admin user flag
	IsHomeUser   *int `json:"is_home_user,omitempty"`  // 0 or 1 - Home user flag
	IsAllowSync  *int `json:"is_allow_sync,omitempty"` // 0 or 1 - Sync permission flag
	IsRestricted *int `json:"is_restricted,omitempty"` // 0 or 1 - Restricted user flag
	KeepHistory  *int `json:"keep_history,omitempty"`  // 0 or 1 - History retention flag
	DeletedUser  *int `json:"deleted_user,omitempty"`  // 0 or 1 - Deleted user flag
	DoNotify     *int `json:"do_notify,omitempty"`     // 0 or 1 - Notification flag
	AllowGuest   *int `json:"allow_guest,omitempty"`   // 0 or 1 - Allow guest access flag (v1.46 - API Coverage Expansion)

	// User library access fields (v1.45 - API Coverage Expansion)
	SharedLibraries *string `json:"shared_libraries,omitempty"` // Semicolon-separated library IDs user can access

	// Network/IP information
	IPAddress       string  `json:"ip_address"`
	IPAddressPublic *string `json:"ip_address_public,omitempty"` // Public IP for accurate geolocation

	// Media identification
	MediaType        string  `json:"media_type"`
	Title            string  `json:"title"`
	ParentTitle      *string `json:"parent_title,omitempty"`
	GrandparentTitle *string `json:"grandparent_title,omitempty"`
	SortTitle        *string `json:"sort_title,omitempty"` // Sort title for ordering (v1.45 - API Coverage Expansion)

	// Client/Player information
	Platform         string  `json:"platform"`
	PlatformName     *string `json:"platform_name,omitempty"`
	PlatformVersion  *string `json:"platform_version,omitempty"`
	Player           string  `json:"player"`
	Product          *string `json:"product,omitempty"`
	ProductVersion   *string `json:"product_version,omitempty"`
	Device           *string `json:"device,omitempty"`
	MachineID        *string `json:"machine_id,omitempty"` // Unique device identifier
	LocationType     string  `json:"location_type"`
	QualityProfile   *string `json:"quality_profile,omitempty"`
	OptimizedVersion *int    `json:"optimized_version,omitempty"`
	SyncedVersion    *int    `json:"synced_version,omitempty"`

	// Playback metrics
	PercentComplete int  `json:"percent_complete"`
	PausedCounter   int  `json:"paused_counter"`
	PlayDuration    *int `json:"play_duration,omitempty"`
	Throttled       *int `json:"throttled,omitempty"` // 0 or 1 - Playback throttled status (v1.45 - API Coverage Expansion)

	// Live TV fields (v1.45 - API Coverage Expansion)
	Live              *int    `json:"live,omitempty"`               // 0 or 1 - Live TV session flag
	LiveUUID          *string `json:"live_uuid,omitempty"`          // Live TV session UUID
	ChannelStream     *int    `json:"channel_stream,omitempty"`     // Live TV channel stream number
	ChannelCallSign   *string `json:"channel_call_sign,omitempty"`  // Live TV channel call sign (e.g., "NBC")
	ChannelIdentifier *string `json:"channel_identifier,omitempty"` // Live TV channel identifier

	// Transcode decision fields
	TranscodeDecision *string `json:"transcode_decision,omitempty"`
	VideoDecision     *string `json:"video_decision,omitempty"`
	AudioDecision     *string `json:"audio_decision,omitempty"`
	SubtitleDecision  *string `json:"subtitle_decision,omitempty"`

	// Hardware transcode fields (CRITICAL for GPU utilization monitoring)
	TranscodeKey            *string `json:"transcode_key,omitempty"`
	TranscodeThrottled      *int    `json:"transcode_throttled,omitempty"`
	TranscodeProgress       *int    `json:"transcode_progress,omitempty"` // 0-100
	TranscodeSpeed          *string `json:"transcode_speed,omitempty"`    // Speed multiplier (e.g., "2.5")
	TranscodeHWRequested    *int    `json:"transcode_hw_requested,omitempty"`
	TranscodeHWDecoding     *int    `json:"transcode_hw_decoding,omitempty"`
	TranscodeHWEncoding     *int    `json:"transcode_hw_encoding,omitempty"`
	TranscodeHWFullPipeline *int    `json:"transcode_hw_full_pipeline,omitempty"`
	TranscodeHWDecode       *string `json:"transcode_hw_decode,omitempty"`       // HW decode codec used
	TranscodeHWDecodeTitle  *string `json:"transcode_hw_decode_title,omitempty"` // HW decoder name (e.g., "Intel Quick Sync")
	TranscodeHWEncode       *string `json:"transcode_hw_encode,omitempty"`       // HW encode codec used
	TranscodeHWEncodeTitle  *string `json:"transcode_hw_encode_title,omitempty"` // HW encoder name (e.g., "NVIDIA NVENC")
	TranscodeContainer      *string `json:"transcode_container,omitempty"`
	TranscodeVideoCodec     *string `json:"transcode_video_codec,omitempty"`
	TranscodeVideoWidth     *int    `json:"transcode_video_width,omitempty"`  // Transcoded video width (v1.46 - API Coverage Expansion)
	TranscodeVideoHeight    *int    `json:"transcode_video_height,omitempty"` // Transcoded video height (v1.46 - API Coverage Expansion)
	TranscodeAudioCodec     *string `json:"transcode_audio_codec,omitempty"`
	TranscodeAudioChannels  *int    `json:"transcode_audio_channels,omitempty"`

	// Source video fields
	VideoResolution     *string `json:"video_resolution,omitempty"`
	VideoFullResolution *string `json:"video_full_resolution,omitempty"` // Full resolution string (e.g., "1080p")
	VideoCodec          *string `json:"video_codec,omitempty"`
	VideoCodecLevel     *string `json:"video_codec_level,omitempty"`
	VideoProfile        *string `json:"video_profile,omitempty"`
	VideoScanType       *string `json:"video_scan_type,omitempty"`
	VideoLanguage       *string `json:"video_language,omitempty"`
	VideoLanguageCode   *string `json:"video_language_code,omitempty"` // 3-letter ISO code (eng, jpn, etc.)
	AspectRatio         *string `json:"aspect_ratio,omitempty"`

	// HDR/Color metadata (CRITICAL for HDR10/HDR10+/Dolby Vision detection)
	VideoColorPrimaries    *string `json:"video_color_primaries,omitempty"`
	VideoColorRange        *string `json:"video_color_range,omitempty"`
	VideoColorSpace        *string `json:"video_color_space,omitempty"`
	VideoColorTrc          *string `json:"video_color_trc,omitempty"`
	VideoChromaSubsampling *string `json:"video_chroma_subsampling,omitempty"`

	// Source audio fields
	AudioCodec   *string `json:"audio_codec,omitempty"`
	AudioProfile *string `json:"audio_profile,omitempty"`

	// Library metadata
	SectionID     *int    `json:"section_id,omitempty"`
	LibraryName   *string `json:"library_name,omitempty"`
	ContentRating *string `json:"content_rating,omitempty"`
	Year          *int    `json:"year,omitempty"`
	Studio        *string `json:"studio,omitempty"` // Production studio

	// Media metadata fields (v1.44 - API Coverage Expansion)
	AddedAt        *string `json:"added_at,omitempty"`        // Unix timestamp when added to library
	UpdatedAt      *string `json:"updated_at,omitempty"`      // Unix timestamp of last metadata update
	LastViewedAt   *string `json:"last_viewed_at,omitempty"`  // Unix timestamp of last watch
	Summary        *string `json:"summary,omitempty"`         // Media description/synopsis
	Tagline        *string `json:"tagline,omitempty"`         // Movie tagline
	Rating         *string `json:"rating,omitempty"`          // Critic rating (0.0-10.0)
	AudienceRating *string `json:"audience_rating,omitempty"` // Audience rating (0.0-10.0)
	UserRating     *string `json:"user_rating,omitempty"`     // User's personal rating (0.0-10.0)
	Labels         *string `json:"labels,omitempty"`          // Labels (comma-separated)
	Collections    *string `json:"collections,omitempty"`     // Collections (comma-separated)
	Banner         *string `json:"banner,omitempty"`          // Banner image URL

	// Metadata enrichment fields (CRITICAL for binge detection and analytics)
	RatingKey             *string `json:"rating_key,omitempty"`
	ParentRatingKey       *string `json:"parent_rating_key,omitempty"`
	GrandparentRatingKey  *string `json:"grandparent_rating_key,omitempty"`
	MediaIndex            *int    `json:"media_index,omitempty"`             // Episode number (for binge detection)
	ParentMediaIndex      *int    `json:"parent_media_index,omitempty"`      // Season number (for binge detection)
	Guid                  *string `json:"guid,omitempty"`                    // External IDs (IMDB, TVDB, TMDB)
	OriginalTitle         *string `json:"original_title,omitempty"`          // Original non-localized title
	FullTitle             *string `json:"full_title,omitempty"`              // Formatted full title
	OriginallyAvailableAt *string `json:"originally_available_at,omitempty"` // Release date
	WatchedStatus         *int    `json:"watched_status,omitempty"`          // 0 = unwatched, 1 = watched
	Thumb                 *string `json:"thumb,omitempty"`                   // Thumbnail URL

	// Cast and crew (comma-separated strings)
	Directors *string `json:"directors,omitempty"` // Director names
	Writers   *string `json:"writers,omitempty"`   // Writer names
	Actors    *string `json:"actors,omitempty"`    // Actor names
	Genres    *string `json:"genres,omitempty"`    // Genres

	// Stream output fields (transcoded output quality)
	StreamContainer           *string `json:"stream_container,omitempty"`
	StreamBitrate             *int    `json:"stream_bitrate,omitempty"`
	StreamVideoCodec          *string `json:"stream_video_codec,omitempty"`
	StreamVideoCodecLevel     *string `json:"stream_video_codec_level,omitempty"`
	StreamVideoResolution     *string `json:"stream_video_resolution,omitempty"`
	StreamVideoFullResolution *string `json:"stream_video_full_resolution,omitempty"` // e.g., "1080p"
	StreamVideoDecision       *string `json:"stream_video_decision,omitempty"`
	StreamVideoBitrate        *int    `json:"stream_video_bitrate,omitempty"`
	StreamVideoBitDepth       *int    `json:"stream_video_bit_depth,omitempty"`
	StreamVideoWidth          *int    `json:"stream_video_width,omitempty"`
	StreamVideoHeight         *int    `json:"stream_video_height,omitempty"`
	StreamVideoFramerate      *string `json:"stream_video_framerate,omitempty"`
	StreamVideoProfile        *string `json:"stream_video_profile,omitempty"`
	StreamVideoScanType       *string `json:"stream_video_scan_type,omitempty"`
	StreamVideoLanguage       *string `json:"stream_video_language,omitempty"`
	StreamVideoLanguageCode   *string `json:"stream_video_language_code,omitempty"` // 3-letter ISO code
	StreamVideoDynamicRange   *string `json:"stream_video_dynamic_range,omitempty"`
	StreamAspectRatio         *string `json:"stream_aspect_ratio,omitempty"` // Output stream aspect ratio (v1.46 - API Coverage Expansion)
	StreamAudioCodec          *string `json:"stream_audio_codec,omitempty"`
	StreamAudioChannels       *string `json:"stream_audio_channels,omitempty"`
	StreamAudioChannelLayout  *string `json:"stream_audio_channel_layout,omitempty"`
	StreamAudioBitrate        *int    `json:"stream_audio_bitrate,omitempty"`
	StreamAudioBitrateMode    *string `json:"stream_audio_bitrate_mode,omitempty"` // cbr, vbr
	StreamAudioSampleRate     *int    `json:"stream_audio_sample_rate,omitempty"`  // Hz
	StreamAudioLanguage       *string `json:"stream_audio_language,omitempty"`
	StreamAudioLanguageCode   *string `json:"stream_audio_language_code,omitempty"` // 3-letter ISO code
	StreamAudioProfile        *string `json:"stream_audio_profile,omitempty"`
	StreamAudioDecision       *string `json:"stream_audio_decision,omitempty"`

	// Stream subtitle output fields (v1.44 - API Coverage Expansion)
	StreamSubtitleCodec        *string `json:"stream_subtitle_codec,omitempty"`
	StreamSubtitleContainer    *string `json:"stream_subtitle_container,omitempty"`
	StreamSubtitleFormat       *string `json:"stream_subtitle_format,omitempty"`
	StreamSubtitleForced       *int    `json:"stream_subtitle_forced,omitempty"`
	StreamSubtitleLocation     *string `json:"stream_subtitle_location,omitempty"`
	StreamSubtitleLanguage     *string `json:"stream_subtitle_language,omitempty"`
	StreamSubtitleLanguageCode *string `json:"stream_subtitle_language_code,omitempty"` // 3-letter ISO code
	StreamSubtitleDecision     *string `json:"stream_subtitle_decision,omitempty"`
	SubtitleContainer          *string `json:"subtitle_container,omitempty"` // Legacy field

	// Source audio details
	AudioChannels      *string `json:"audio_channels,omitempty"`
	AudioChannelLayout *string `json:"audio_channel_layout,omitempty"`
	AudioBitrate       *int    `json:"audio_bitrate,omitempty"`
	AudioBitrateMode   *string `json:"audio_bitrate_mode,omitempty"`
	AudioSampleRate    *int    `json:"audio_sample_rate,omitempty"`
	AudioLanguage      *string `json:"audio_language,omitempty"`
	AudioLanguageCode  *string `json:"audio_language_code,omitempty"` // 3-letter ISO code (eng, jpn, etc.)

	// Source video details
	VideoDynamicRange *string `json:"video_dynamic_range,omitempty"`
	VideoFrameRate    *string `json:"video_framerate,omitempty"`
	VideoBitrate      *int    `json:"video_bitrate,omitempty"`
	VideoBitDepth     *int    `json:"video_bit_depth,omitempty"`
	VideoRefFrames    *int    `json:"video_ref_frames,omitempty"`
	VideoWidth        *int    `json:"video_width,omitempty"`
	VideoHeight       *int    `json:"video_height,omitempty"`

	// Container and subtitle
	Container            *string `json:"container,omitempty"`
	SubtitleCodec        *string `json:"subtitle_codec,omitempty"`
	SubtitleContainerFmt *string `json:"subtitle_container_fmt,omitempty"` // Container format
	SubtitleFormat       *string `json:"subtitle_format,omitempty"`        // Subtitle format (srt, ass, pgs)
	SubtitleLanguage     *string `json:"subtitle_language,omitempty"`
	SubtitleLanguageCode *string `json:"subtitle_language_code,omitempty"` // 3-letter ISO code (eng, spa, etc.)
	SubtitleForced       *int    `json:"subtitle_forced,omitempty"`
	SubtitleLocation     *string `json:"subtitle_location,omitempty"` // embedded, external, agent
	Subtitles            *int    `json:"subtitles,omitempty"`         // 0 or 1 flag

	// Connection security and network
	Secure       *int    `json:"secure,omitempty"`        // 0 or 1 flag
	Relayed      *int    `json:"relayed,omitempty"`       // 0 or 1 flag
	Relay        *int    `json:"relay,omitempty"`         // 0 or 1 flag
	Local        *int    `json:"local,omitempty"`         // 0 or 1 flag
	Bandwidth    *int    `json:"bandwidth,omitempty"`     // Overall bandwidth
	Location     *string `json:"location,omitempty"`      // "lan" or "wan"
	BandwidthLAN *int    `json:"bandwidth_lan,omitempty"` // LAN bandwidth limit (kbps)
	BandwidthWAN *int    `json:"bandwidth_wan,omitempty"` // WAN bandwidth limit (kbps)

	// File metadata
	FileSize *int64  `json:"file_size,omitempty"`
	Bitrate  *int    `json:"bitrate,omitempty"`
	File     *string `json:"file,omitempty"` // Full file path

	// Bitrate analytics fields (v1.42) - 3-level tracking for network bottleneck identification
	SourceBitrate    *int `json:"source_bitrate,omitempty"`    // Kbps - original file bitrate
	TranscodeBitrate *int `json:"transcode_bitrate,omitempty"` // Kbps - transcoded stream bitrate
	NetworkBandwidth *int `json:"network_bandwidth,omitempty"` // Kbps - measured network bandwidth

	// Thumbnails and art
	ParentThumb      *string `json:"parent_thumb,omitempty"`
	GrandparentThumb *string `json:"grandparent_thumb,omitempty"`
	Art              *string `json:"art,omitempty"`
	GrandparentArt   *string `json:"grandparent_art,omitempty"`

	// Additional GUID fields for external ID matching
	ParentGuid      *string `json:"parent_guid,omitempty"`
	GrandparentGuid *string `json:"grandparent_guid,omitempty"`

	CreatedAt time.Time `json:"created_at"`
}

// FailedEvent represents an event that failed to be inserted into DuckDB
// after exceeding the maximum retry count in the Consumer WAL.
// This provides a persistent DLQ (Dead Letter Queue) for investigation and recovery.
type FailedEvent struct {
	ID uuid.UUID `json:"id"`

	// Original event data
	TransactionID  string `json:"transaction_id"`
	EventID        string `json:"event_id"`
	SessionKey     string `json:"session_key,omitempty"`
	CorrelationKey string `json:"correlation_key,omitempty"`
	Source         string `json:"source"`
	EventPayload   []byte `json:"event_payload"` // Full event JSON for recovery

	// Failure details
	FailedAt      time.Time `json:"failed_at"`
	FailureReason string    `json:"failure_reason"`
	FailureLayer  string    `json:"failure_layer"` // 'consumer_wal', 'duckdb_insert', 'validation'
	LastError     string    `json:"last_error,omitempty"`

	// Retry tracking
	RetryCount         int        `json:"retry_count"`
	LastRetryAt        *time.Time `json:"last_retry_at,omitempty"`
	MaxRetriesExceeded bool       `json:"max_retries_exceeded"`

	// Resolution
	Status          string     `json:"status"` // 'pending', 'retrying', 'resolved', 'abandoned'
	ResolvedAt      *time.Time `json:"resolved_at,omitempty"`
	ResolvedBy      string     `json:"resolved_by,omitempty"`
	ResolutionNotes string     `json:"resolution_notes,omitempty"`

	// Audit
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// DedupeAuditEntry represents a deduplication decision for audit and recovery.
// Created when an event is deduplicated (discarded as duplicate) by the event processor.
// Stores full event data to enable manual restoration if the dedup was incorrect.
//
// Deduplication reasons (dedupe_reason):
//   - event_id: Exact EventID match
//   - session_key: Same SessionKey from same source
//   - correlation_key: Same CorrelationKey from same source
//   - cross_source_key: Same content from different sources (Plex+Tautulli)
//   - db_constraint: Unique constraint violation in DuckDB
//
// Deduplication layers (dedupe_layer):
//   - bloom_cache: In-memory cache (v2.3: ExactLRU with ZERO false positives, previously BloomLRU)
//   - nats_dedup: NATS JetStream message deduplication
//   - db_unique: DuckDB unique index constraint
//
// Status workflow:
//   - auto_dedupe: Automatically deduplicated (default)
//   - user_confirmed: User confirmed dedup was correct
//   - user_restored: User restored the discarded event
type DedupeAuditEntry struct {
	ID        uuid.UUID `json:"id"`
	Timestamp time.Time `json:"timestamp"`

	// The event that was deduplicated (discarded)
	DiscardedEventID        string     `json:"discarded_event_id"`
	DiscardedSessionKey     string     `json:"discarded_session_key,omitempty"`
	DiscardedCorrelationKey string     `json:"discarded_correlation_key,omitempty"`
	DiscardedSource         string     `json:"discarded_source"`
	DiscardedStartedAt      *time.Time `json:"discarded_started_at,omitempty"`
	DiscardedRawPayload     []byte     `json:"discarded_raw_payload,omitempty"` // Full event JSON for recovery

	// The event that it was matched against (kept)
	MatchedEventID        string `json:"matched_event_id,omitempty"`
	MatchedSessionKey     string `json:"matched_session_key,omitempty"`
	MatchedCorrelationKey string `json:"matched_correlation_key,omitempty"`
	MatchedSource         string `json:"matched_source,omitempty"`

	// Deduplication details
	DedupeReason    string   `json:"dedupe_reason"`              // 'event_id', 'session_key', 'correlation_key', 'cross_source_key', 'db_constraint'
	DedupeLayer     string   `json:"dedupe_layer"`               // 'bloom_cache', 'nats_dedup', 'db_unique'
	SimilarityScore *float64 `json:"similarity_score,omitempty"` // For fuzzy matching scenarios

	// User information
	UserID   int    `json:"user_id"`
	Username string `json:"username,omitempty"`

	// Media information
	MediaType string `json:"media_type,omitempty"`
	Title     string `json:"title,omitempty"`
	RatingKey string `json:"rating_key,omitempty"`

	// Resolution status
	Status          string     `json:"status"` // 'auto_dedupe', 'user_confirmed', 'user_restored'
	ResolvedBy      string     `json:"resolved_by,omitempty"`
	ResolvedAt      *time.Time `json:"resolved_at,omitempty"`
	ResolutionNotes string     `json:"resolution_notes,omitempty"`

	// Audit
	CreatedAt time.Time `json:"created_at"`
}

// DedupeAuditStats holds aggregate statistics for the dedupe audit dashboard.
type DedupeAuditStats struct {
	TotalDeduped   int64            `json:"total_deduped"`    // Total deduplicated events
	PendingReview  int64            `json:"pending_review"`   // Events with status 'auto_dedupe'
	UserRestored   int64            `json:"user_restored"`    // Events restored by users
	UserConfirmed  int64            `json:"user_confirmed"`   // Events confirmed correct by users
	AccuracyRate   float64          `json:"accuracy_rate"`    // Percentage confirmed as correct
	DedupeByReason map[string]int64 `json:"dedupe_by_reason"` // Breakdown by reason
	DedupeByLayer  map[string]int64 `json:"dedupe_by_layer"`  // Breakdown by layer
	DedupeBySource map[string]int64 `json:"dedupe_by_source"` // Breakdown by source
	Last24Hours    int64            `json:"last_24_hours"`    // Dedupes in last 24 hours
	Last7Days      int64            `json:"last_7_days"`      // Dedupes in last 7 days
	Last30Days     int64            `json:"last_30_days"`     // Dedupes in last 30 days
}

// Geolocation represents geographic data for an IP address
type Geolocation struct {
	IPAddress      string    `json:"ip_address"`
	Latitude       float64   `json:"latitude"`
	Longitude      float64   `json:"longitude"`
	City           *string   `json:"city,omitempty"`
	Region         *string   `json:"region,omitempty"`
	Country        string    `json:"country"`
	PostalCode     *string   `json:"postal_code,omitempty"`
	Timezone       *string   `json:"timezone,omitempty"`
	AccuracyRadius *int      `json:"accuracy_radius,omitempty"`
	LastUpdated    time.Time `json:"last_updated"`
}

// LocationStats represents aggregated playback statistics for a geographic location
type LocationStats struct {
	Country       string    `json:"country"`
	Region        *string   `json:"region,omitempty"`
	City          *string   `json:"city,omitempty"`
	Latitude      float64   `json:"latitude"`
	Longitude     float64   `json:"longitude"`
	PlaybackCount int       `json:"playback_count"`
	UniqueUsers   int       `json:"unique_users"`
	FirstSeen     time.Time `json:"first_seen"`
	LastSeen      time.Time `json:"last_seen"`
	AvgCompletion float64   `json:"avg_completion"`
}
