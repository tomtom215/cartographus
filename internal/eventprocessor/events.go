// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package eventprocessor

import (
	"time"

	"github.com/goccy/go-json"
	"github.com/google/uuid"
)

// SchemaVersion is the current event schema version.
// Increment this when making breaking changes to MediaEvent.
const SchemaVersion = 1

// MediaEvent represents a playback event from media servers.
// This is the canonical event format used across all sources (Plex, Tautulli, Jellyfin, Emby).
//
// Schema versioning (Phase 2.3):
// - SchemaVersion field tracks the event format version
// - Consumers should handle older schema versions for backward compatibility
// - Version 1: Initial schema with all current fields
type MediaEvent struct {
	// Schema version for forward/backward compatibility
	SchemaVersion int `json:"schema_version,omitempty"` // Event schema version (default: 1)

	// Identification
	EventID        string    `json:"event_id"`
	SessionKey     string    `json:"session_key,omitempty"`     // Source-specific session identifier
	CorrelationKey string    `json:"correlation_key,omitempty"` // Cross-source deduplication key
	TransactionID  string    `json:"transaction_id,omitempty"`  // v2.1: Idempotency key for exactly-once delivery
	Source         string    `json:"source"`                    // plex, jellyfin, tautulli, emby
	ServerID       string    `json:"server_id,omitempty"`       // Unique server identifier (v2.0: multi-server support)
	Timestamp      time.Time `json:"timestamp"`

	// User information
	UserID       int    `json:"user_id"`
	Username     string `json:"username"`
	FriendlyName string `json:"friendly_name,omitempty"` // Display name
	UserThumb    string `json:"user_thumb,omitempty"`    // User avatar URL
	Email        string `json:"email,omitempty"`         // User email

	// Media identification
	MediaType        string `json:"media_type"` // movie, episode, track
	Title            string `json:"title"`
	ParentTitle      string `json:"parent_title,omitempty"`      // Season name for episodes
	GrandparentTitle string `json:"grandparent_title,omitempty"` // Show name for episodes
	RatingKey        string `json:"rating_key,omitempty"`
	Year             int    `json:"year,omitempty"`           // Release year
	MediaDuration    int    `json:"media_duration,omitempty"` // Media duration in seconds

	// Playback timing
	StartedAt       time.Time  `json:"started_at"`
	StoppedAt       *time.Time `json:"stopped_at,omitempty"`
	PercentComplete int        `json:"percent_complete,omitempty"`
	PlayDuration    int        `json:"play_duration,omitempty"` // seconds
	PausedCounter   int        `json:"paused_counter,omitempty"`

	// Platform information
	Platform        string `json:"platform,omitempty"`
	PlatformName    string `json:"platform_name,omitempty"`
	PlatformVersion string `json:"platform_version,omitempty"`
	Player          string `json:"player,omitempty"`
	Product         string `json:"product,omitempty"`
	ProductVersion  string `json:"product_version,omitempty"`
	Device          string `json:"device,omitempty"`
	MachineID       string `json:"machine_id,omitempty"` // Unique device identifier
	IPAddress       string `json:"ip_address,omitempty"`
	LocationType    string `json:"location_type,omitempty"` // wan, lan

	// Streaming quality
	TranscodeDecision string `json:"transcode_decision,omitempty"`
	VideoResolution   string `json:"video_resolution,omitempty"`
	VideoCodec        string `json:"video_codec,omitempty"`
	VideoDynamicRange string `json:"video_dynamic_range,omitempty"`
	AudioCodec        string `json:"audio_codec,omitempty"`
	AudioChannels     int    `json:"audio_channels,omitempty"`
	StreamBitrate     int    `json:"stream_bitrate,omitempty"`
	Bandwidth         int    `json:"bandwidth,omitempty"` // Network bandwidth

	// Connection details
	Secure  bool `json:"secure,omitempty"`
	Local   bool `json:"local,omitempty"`
	Relayed bool `json:"relayed,omitempty"`

	// Raw payload for debugging and future fields
	RawPayload json.RawMessage `json:"raw_payload,omitempty"`
}

// NewMediaEvent creates an event with a unique ID, timestamp, and schema version.
func NewMediaEvent(source string) *MediaEvent {
	return &MediaEvent{
		SchemaVersion: SchemaVersion,
		EventID:       uuid.New().String(),
		Source:        source,
		Timestamp:     time.Now().UTC(),
	}
}

// GetSchemaVersion returns the schema version, defaulting to 1 for legacy events.
func (e *MediaEvent) GetSchemaVersion() int {
	if e.SchemaVersion == 0 {
		return 1 // Default for events without explicit version (backward compatibility)
	}
	return e.SchemaVersion
}

// EnsureSchemaVersion sets the schema version if not already set.
// Call this when processing events that may not have a version set.
func (e *MediaEvent) EnsureSchemaVersion() {
	if e.SchemaVersion == 0 {
		e.SchemaVersion = SchemaVersion
	}
}

// Validate checks required fields and returns an error if validation fails.
func (e *MediaEvent) Validate() error {
	if e.EventID == "" {
		return &ValidationError{Field: "event_id", Message: "required"}
	}
	if e.Source == "" {
		return &ValidationError{Field: "source", Message: "required"}
	}
	if e.UserID == 0 {
		return &ValidationError{Field: "user_id", Message: "required"}
	}
	if e.MediaType == "" {
		return &ValidationError{Field: "media_type", Message: "required"}
	}
	if e.Title == "" {
		return &ValidationError{Field: "title", Message: "required"}
	}
	return nil
}

// Topic returns the NATS subject for this event.
// Format: playback.<source>.<media_type>
// Example: playback.plex.movie
func (e *MediaEvent) Topic() string {
	return "playback." + e.Source + "." + e.MediaType
}

// IsComplete returns true if the playback has ended.
func (e *MediaEvent) IsComplete() bool {
	return e.StoppedAt != nil
}

// Duration returns the total playback duration in seconds.
func (e *MediaEvent) Duration() int {
	if e.PlayDuration > 0 {
		return e.PlayDuration
	}
	if e.StoppedAt != nil {
		return int(e.StoppedAt.Sub(e.StartedAt).Seconds())
	}
	return int(time.Since(e.StartedAt).Seconds())
}

// GenerateCorrelationKey creates a correlation key for cross-source deduplication.
// The key uniquely identifies a playback session for a specific source and server.
//
// Format (v2.3): {source}:{server_id}:{user_id}:{rating_key}:{machine_id}:{time_bucket}:{session_key}
//
// Components:
//   - source: Media server type (plex, jellyfin, emby, tautulli)
//   - server_id: Server instance identifier (for multi-server deployments)
//   - user_id: Internal user ID (mapped from external UUIDs for Jellyfin/Emby)
//   - rating_key: Media content identifier (or title as fallback)
//   - machine_id: Device identifier (for multi-device support)
//   - time_bucket: StartedAt with second precision (handles clock skew)
//   - session_key: Source-specific session identifier (GUARANTEES UNIQUENESS)
//
// CRITICAL (v2.3): The session_key is included to GUARANTEE uniqueness and prevent
// data loss from correlation key collisions. Previously, events with the same content
// at the same second would collide and be incorrectly deduplicated.
//
// Cross-source deduplication still works via getCrossSourceKey() in handlers.go,
// which extracts the content-based portion (parts 2-6) for matching.
//
// Example: "plex:server-abc:12345:54321:device123:2024-01-15T10:30:00:session-xyz"
func (e *MediaEvent) GenerateCorrelationKey() string {
	// Use exact timestamp (second precision) for correlation key
	// This prevents false deduplication of different sessions that happen within the same time window
	// For cross-source matching (Plex webhook + Tautulli sync), identical playbacks have identical started_at
	timeBucket := e.StartedAt.UTC().Format("2006-01-02T15:04:05")

	// Rating key is the primary content identifier
	ratingKey := e.RatingKey
	if ratingKey == "" {
		// Fallback to title-based key if no rating key
		ratingKey = e.Title
	}

	// MachineID identifies the device - critical for multi-device support
	// When empty, use "unknown" to ensure consistent key format
	machineID := e.MachineID
	if machineID == "" {
		machineID = "unknown"
	}

	// Source is required - default to "unknown" if not set
	source := e.Source
	if source == "" {
		source = "unknown"
	}

	// ServerID identifies the server instance - default to "default" if not set
	serverID := e.ServerID
	if serverID == "" {
		serverID = "default"
	}

	// SessionKey is the source-specific session identifier
	// CRITICAL: This guarantees uniqueness and prevents data loss from collisions
	sessionKey := e.SessionKey
	if sessionKey == "" {
		// Use EventID as fallback if no session key (should not happen normally)
		sessionKey = e.EventID
	}

	return formatCorrelationKey(source, serverID, e.UserID, ratingKey, machineID, timeBucket, sessionKey)
}

// SetCorrelationKey generates and sets the correlation key.
// Call this before publishing to NATS.
func (e *MediaEvent) SetCorrelationKey() {
	e.CorrelationKey = e.GenerateCorrelationKey()
}

// GetCrossSourceKey extracts the content-based portion of a correlation key
// for cross-source deduplication. This is the exported version for use in tests.
//
// v2.3 Format (7 parts): {source}:{server_id}:{user_id}:{rating_key}:{machine_id}:{time_bucket}:{session_key}
// Output: {server_id}:{user_id}:{rating_key}:{machine_id}:{time_bucket}
// (strips source prefix AND session_key suffix to get content-based portion)
//
// This enables cross-source deduplication: the same playback from different sources
// (e.g., Plex webhook + Tautulli import) will have the same cross-source key.
func GetCrossSourceKey(corrKey string) string {
	if corrKey == "" {
		return ""
	}

	// Count colons to determine format
	colonCount := 0
	for i := 0; i < len(corrKey); i++ {
		if corrKey[i] == ':' {
			colonCount++
		}
	}

	// Find the first colon position (to skip source)
	firstColon := -1
	for i := 0; i < len(corrKey); i++ {
		if corrKey[i] == ':' {
			firstColon = i
			break
		}
	}
	if firstColon < 0 || firstColon >= len(corrKey)-1 {
		return ""
	}

	// Legacy format (5 colons = 6 parts): skip source only
	if colonCount <= 5 {
		return corrKey[firstColon+1:]
	}

	// v2.3 format (6+ colons = 7+ parts): skip source AND session_key
	// Find the last colon position (to skip session_key)
	lastColon := -1
	for i := len(corrKey) - 1; i >= 0; i-- {
		if corrKey[i] == ':' {
			lastColon = i
			break
		}
	}

	// Return content between first colon and last colon
	if lastColon <= firstColon {
		return corrKey[firstColon+1:]
	}
	return corrKey[firstColon+1 : lastColon]
}

// formatCorrelationKey creates the correlation key string.
// Format (v2.3): {source}:{server_id}:{user_id}:{rating_key}:{machine_id}:{time_bucket}:{session_key}
func formatCorrelationKey(source, serverID string, userID int, ratingKey, machineID, timeBucket, sessionKey string) string {
	return source + ":" + serverID + ":" + formatInt(userID) + ":" + ratingKey + ":" + machineID + ":" + timeBucket + ":" + sessionKey
}

// formatInt converts an int to string without importing strconv.
func formatInt(n int) string {
	if n == 0 {
		return "0"
	}
	if n < 0 {
		return "-" + formatInt(-n)
	}

	// Simple digit extraction
	digits := make([]byte, 0, 20)
	for n > 0 {
		digits = append(digits, byte('0'+n%10))
		n /= 10
	}

	// Reverse
	for i, j := 0, len(digits)-1; i < j; i, j = i+1, j-1 {
		digits[i], digits[j] = digits[j], digits[i]
	}
	return string(digits)
}

// ValidationError represents a field validation error.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Field + ": " + e.Message
}

// EventType constants for NATS subjects.
const (
	// EventTypePlaybackStart indicates playback has started.
	EventTypePlaybackStart = "start"
	// EventTypePlaybackStop indicates playback has stopped.
	EventTypePlaybackStop = "stop"
	// EventTypePlaybackPause indicates playback was paused.
	EventTypePlaybackPause = "pause"
	// EventTypePlaybackResume indicates playback was resumed.
	EventTypePlaybackResume = "resume"
	// EventTypePlaybackProgress indicates a progress update.
	EventTypePlaybackProgress = "progress"
)

// SourceType constants for event sources.
const (
	// SourcePlex indicates the event came from Plex.
	SourcePlex = "plex"
	// SourceTautulli indicates the event came from Tautulli.
	SourceTautulli = "tautulli"
	// SourceJellyfin indicates the event came from Jellyfin.
	SourceJellyfin = "jellyfin"
	// SourceEmby indicates the event came from Emby.
	SourceEmby = "emby"
)

// MediaType constants for media types.
const (
	// MediaTypeMovie indicates a movie.
	MediaTypeMovie = "movie"
	// MediaTypeEpisode indicates a TV episode.
	MediaTypeEpisode = "episode"
	// MediaTypeTrack indicates a music track.
	MediaTypeTrack = "track"
)

// TranscodeDecision constants.
const (
	// TranscodeDecisionDirectPlay indicates direct playback without transcoding.
	TranscodeDecisionDirectPlay = "direct play"
	// TranscodeDecisionDirectStream indicates direct streaming without transcoding.
	TranscodeDecisionDirectStream = "direct stream"
	// TranscodeDecisionTranscode indicates the stream is being transcoded.
	TranscodeDecisionTranscode = "transcode"
)

// LocationType constants.
const (
	// LocationTypeWAN indicates remote/external connection.
	LocationTypeWAN = "wan"
	// LocationTypeLAN indicates local network connection.
	LocationTypeLAN = "lan"
)
