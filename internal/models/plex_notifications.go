// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package models

import (
	"fmt"
	"strings"
)

// Plex WebSocket Notification Models
// These structures represent real-time notifications from Plex Media Server's WebSocket endpoint
// Endpoint: ws://{plex_url}/:/websockets/notifications
// Documentation: https://forums.plex.tv/t/about-websocket-notifications/79679

// PlexNotificationWrapper wraps the top-level notification container
type PlexNotificationWrapper struct {
	NotificationContainer PlexNotificationContainer `json:"NotificationContainer"`
}

// PlexNotificationContainer wraps all notification types from Plex WebSocket
type PlexNotificationContainer struct {
	Type                         string                         `json:"type"`                                   // Notification type: "playing", "timeline", "activity", "status", "reachability"
	Size                         int                            `json:"size,omitempty"`                         // Number of notifications in this message
	PlaySessionStateNotification []PlexPlayingNotification      `json:"PlaySessionStateNotification,omitempty"` // Real-time playback state changes
	TimelineEntry                []PlexTimelineNotification     `json:"TimelineEntry,omitempty"`                // Content metadata changes
	ActivityNotification         []PlexActivityNotification     `json:"ActivityNotification,omitempty"`         // Background task updates
	StatusNotification           []PlexStatusNotification       `json:"StatusNotification,omitempty"`           // Server status changes
	ReachabilityNotification     []PlexReachabilityNotification `json:"ReachabilityNotification,omitempty"`     // Network connectivity changes
}

// PlexPlayingNotification represents real-time playback state changes
// CRITICAL: Provides instant updates (< 1 second) vs Tautulli's 30-60 second polling
//
// Use Cases:
//   - Instant map marker updates when playback starts
//   - Buffering detection for performance monitoring
//   - Pause/resume notifications for engagement analytics
//   - Session state tracking for live activity dashboard
type PlexPlayingNotification struct {
	// Session identification
	SessionKey       string `json:"sessionKey"`       // Unique session identifier (links to Tautulli data)
	ClientIdentifier string `json:"clientIdentifier"` // Unique device identifier

	// Playback state (CRITICAL for real-time updates)
	State      string `json:"state"`      // "playing", "paused", "stopped", "buffering" - INSTANT STATE CHANGES
	RatingKey  string `json:"ratingKey"`  // Plex content identifier
	ViewOffset int64  `json:"viewOffset"` // Current playback position (milliseconds)

	// Content identification
	Key  string `json:"key,omitempty"`  // Metadata key path
	Guid string `json:"guid,omitempty"` // External GUID (imdb://, tvdb://, etc.)
	URL  string `json:"url,omitempty"`  // Content URL

	// Transcode session (CRITICAL: Real-time transcode monitoring)
	TranscodeSession string `json:"transcodeSession,omitempty"` // Links to /video/:/transcode/sessions endpoint
}

// PlexTimelineNotification represents content metadata changes
//
// Use Cases:
//   - Library scan progress monitoring
//   - New content detection
//   - Metadata update tracking
//   - Delete operations detection
type PlexTimelineNotification struct {
	// Content identification
	Identifier string `json:"identifier"`          // Provider identifier (tv.plex.provider.vod, com.plexapp.plugins.library)
	ItemID     int    `json:"itemID"`              // Plex item ID
	Type       int    `json:"type"`                // Media type: 1=movie, 2=show, 3=season, 4=episode, 8=artist, 9=album, 10=track
	Title      string `json:"title,omitempty"`     // Content title
	SectionID  int    `json:"sectionID,omitempty"` // Library section ID

	// State tracking
	State         int    `json:"state"`                   // State code: 0=created, 2=matching, 3=downloading, 4=loading, 5=finished, 6=analyzing, 9=deleted
	MetadataState string `json:"metadataState,omitempty"` // Text state: "created", "matched", "downloaded", "analyzed"
	UpdatedAt     int64  `json:"updatedAt"`               // Unix timestamp (seconds)
}

// PlexActivityNotification represents background task updates
//
// Use Cases:
//   - Library scan progress (0-100%)
//   - Optimize database progress
//   - Backup operations monitoring
//   - Scheduled task tracking
type PlexActivityNotification struct {
	Event    string           `json:"event"`    // Event type: "started", "ended", "progress"
	UUID     string           `json:"uuid"`     // Unique task identifier
	Activity PlexActivityData `json:"Activity"` // Task details and progress
}

// PlexActivityData represents detailed background task information
type PlexActivityData struct {
	UUID     string `json:"uuid"`     // Task UUID (same as parent)
	Type     string `json:"type"`     // Task type: "library.refresh.items", "library.scan", "library.analyze"
	Title    string `json:"title"`    // User-friendly task name
	Subtitle string `json:"subtitle"` // Additional context
	Progress int    `json:"progress"` // Progress percentage (0-100)

	// Task context (library operations)
	Context PlexActivityContext `json:"Context,omitempty"`
}

// PlexActivityContext provides task context (library section details)
type PlexActivityContext struct {
	LibrarySectionID    string `json:"librarySectionID"`    // Library section ID
	LibrarySectionTitle string `json:"librarySectionTitle"` // Library name (e.g., "Movies", "TV Shows")
	LibrarySectionType  string `json:"librarySectionType"`  // Library type (movie, show, artist, photo)
}

// PlexStatusNotification represents server status changes
//
// Use Cases:
//   - Server shutdown alerts
//   - Server restart detection
//   - Maintenance window tracking
type PlexStatusNotification struct {
	Title            string `json:"title"`            // Status title
	Description      string `json:"description"`      // Detailed description
	NotificationName string `json:"notificationName"` // Notification type: "SERVER_SHUTDOWN", "SERVER_STARTED"
}

// PlexReachabilityNotification represents network connectivity changes
//
// Use Cases:
//   - Server connectivity monitoring
//   - Network status alerts
//   - Uptime tracking
type PlexReachabilityNotification struct {
	Reachable bool `json:"reachable"` // Server reachability status
}

// GetPlaybackState returns user-friendly playback state description
func (p *PlexPlayingNotification) GetPlaybackState() string {
	switch p.State {
	case "playing":
		return "Playing"
	case "paused":
		return "Paused"
	case "stopped":
		return "Stopped"
	case "buffering":
		return "Buffering" // CRITICAL: Detect performance issues in real-time
	default:
		return "Unknown"
	}
}

// IsBuffering returns true if the session is currently buffering
// CRITICAL: Use for performance monitoring and server health alerts
func (p *PlexPlayingNotification) IsBuffering() bool {
	return p.State == "buffering"
}

// IsActive returns true if the session is actively playing or paused
func (p *PlexPlayingNotification) IsActive() bool {
	return p.State == "playing" || p.State == "paused" || p.State == "buffering"
}

// GetActivityProgress returns progress percentage for background tasks
func (a *PlexActivityNotification) GetActivityProgress() int {
	return a.Activity.Progress
}

// IsComplete returns true if the activity has finished
func (a *PlexActivityNotification) IsComplete() bool {
	return a.Event == "ended" || a.Activity.Progress == 100
}

// GetLibraryContext returns library context if available
func (a *PlexActivityNotification) GetLibraryContext() *PlexActivityContext {
	if a.Activity.Context.LibrarySectionID != "" {
		return &a.Activity.Context
	}
	return nil
}

// ===================================================================================================
// Plex Active Sessions Models (Phase 1.2: Active Transcode Monitoring)
// ===================================================================================================
// These structures represent active playback sessions from /status/sessions endpoint
// Used for real-time transcode monitoring, quality tracking, and hardware acceleration detection
//
// Endpoint: GET /status/sessions?X-Plex-Token={token}
// Returns: Currently active playback sessions with transcode details
// ===================================================================================================

// PlexSessionsResponse represents the top-level response from /status/sessions
type PlexSessionsResponse struct {
	MediaContainer PlexSessionsContainer `json:"MediaContainer"`
}

// PlexSessionsContainer wraps the active sessions array
type PlexSessionsContainer struct {
	Size     int           `json:"size"`     // Number of active sessions
	Metadata []PlexSession `json:"Metadata"` // Array of active session metadata
}

// PlexSession represents a single active playback session
//
// Contains complete information about:
//   - User and device details
//   - Content being played
//   - Playback progress and state
//   - Transcode session details (if transcoding)
//   - Quality metrics (source vs delivered)
type PlexSession struct {
	// Session identification
	SessionKey string `json:"sessionKey"` // Unique session identifier (matches WebSocket notifications)
	Key        string `json:"key"`        // Metadata key path

	// Content information
	RatingKey            string `json:"ratingKey"`                      // Plex content identifier
	ParentRatingKey      string `json:"parentRatingKey,omitempty"`      // Season/Album rating key
	GrandparentRatingKey string `json:"grandparentRatingKey,omitempty"` // Show/Artist rating key
	Type                 string `json:"type"`                           // "movie", "episode", "track"
	Title                string `json:"title"`                          // Content title
	GrandparentTitle     string `json:"grandparentTitle,omitempty"`     // Show/Artist name
	ParentTitle          string `json:"parentTitle,omitempty"`          // Season/Album name

	// User information
	User *PlexSessionUser `json:"User,omitempty"` // User watching this session

	// Player information
	Player *PlexSessionPlayer `json:"Player,omitempty"` // Device/client information

	// Playback state
	ViewOffset int64  `json:"viewOffset"` // Current playback position (milliseconds)
	Duration   int64  `json:"duration"`   // Total duration (milliseconds)
	PlayState  string `json:"playState"`  // "playing", "paused", "buffering"

	// Transcode session (CRITICAL: Real-time transcode monitoring)
	TranscodeSession *PlexTranscodeSession `json:"TranscodeSession,omitempty"` // Transcode details (nil if direct play)

	// Media information (source quality)
	Media []PlexMedia `json:"Media,omitempty"` // Media streams and quality info
}

// PlexSessionUser represents user information in active sessions
type PlexSessionUser struct {
	ID    int    `json:"id"`
	Title string `json:"title"` // Username
	Thumb string `json:"thumb"` // Avatar URL
}

// PlexSessionPlayer represents device/client information
type PlexSessionPlayer struct {
	Address         string `json:"address"` // Client IP address
	Device          string `json:"device"`  // Device name (e.g., "Xbox One", "Chrome")
	MachineID       string `json:"machineIdentifier"`
	Model           string `json:"model"`    // Device model
	Platform        string `json:"platform"` // Platform (e.g., "Windows", "iOS", "tvOS")
	PlatformVersion string `json:"platformVersion"`
	Product         string `json:"product"` // Client app (e.g., "Plex Web", "Plex for iOS")
	Profile         string `json:"profile"` // Client profile
	State           string `json:"state"`   // Player state ("playing", "paused")
	Title           string `json:"title"`   // Device friendly name
	Version         string `json:"version"` // Client version
	Local           bool   `json:"local"`   // Local network connection
	Relayed         bool   `json:"relayed"` // Using Plex Relay
	Secure          bool   `json:"secure"`  // HTTPS connection
}

// PlexTranscodeSession represents active transcode session details
//
// CRITICAL: This is the core data structure for Phase 1.2 transcode monitoring
// Provides real-time insights into:
//   - Transcode progress and speed
//   - Hardware acceleration status
//   - Quality decisions and reasoning
//   - Resource utilization
type PlexTranscodeSession struct {
	// Transcode identification
	Key                  string  `json:"key"`                     // Transcode session key (/transcode/sessions/{uuid})
	Throttled            bool    `json:"throttled"`               // Transcode throttled due to system load
	Complete             bool    `json:"complete"`                // Transcode completed
	Progress             float64 `json:"progress"`                // Transcode progress (0-100 percentage)
	Size                 int64   `json:"size"`                    // Transcoded data size (bytes)
	Speed                float64 `json:"speed"`                   // Transcode speed (e.g., 1.5 = 1.5x realtime)
	Duration             int64   `json:"duration"`                // Total duration being transcoded (milliseconds)
	Remaining            int     `json:"remaining"`               // Estimated time remaining (seconds)
	Context              string  `json:"context"`                 // Transcode context (e.g., "streaming")
	SourceVideoCodec     string  `json:"sourceVideoCodec"`        // Original video codec (e.g., "hevc", "h264")
	SourceAudioCodec     string  `json:"sourceAudioCodec"`        // Original audio codec
	VideoDecision        string  `json:"videoDecision"`           // "transcode", "copy", "direct play"
	AudioDecision        string  `json:"audioDecision"`           // "transcode", "copy", "direct play"
	SubtitleDecision     string  `json:"subtitleDecision"`        // "transcode", "copy", "burn"
	Protocol             string  `json:"protocol"`                // Streaming protocol (e.g., "hls", "dash")
	Container            string  `json:"container"`               // Output container format
	VideoCodec           string  `json:"videoCodec"`              // Target video codec (transcoded)
	AudioCodec           string  `json:"audioCodec"`              // Target audio codec (transcoded)
	AudioChannels        int     `json:"audioChannels"`           // Target audio channels
	Width                int     `json:"width"`                   // Target video width (pixels)
	Height               int     `json:"height"`                  // Target video height (pixels)
	TranscodeHwRequested bool    `json:"transcodeHwRequested"`    // Hardware acceleration requested
	TranscodeHwDecoding  string  `json:"transcodeHwDecoding"`     // Hardware decoder (e.g., "videotoolbox", "qsv", "nvenc", "vaapi")
	TranscodeHwEncoding  string  `json:"transcodeHwEncoding"`     // Hardware encoder (e.g., "qsv", "nvenc", "vaapi", "videotoolbox")
	TranscodeHwFullPipe  bool    `json:"transcodeHwFullPipeline"` // Full hardware pipeline (decode + encode)
	TimeStamp            float64 `json:"timeStamp"`               // Server timestamp
	MaxOffsetAvailable   float64 `json:"maxOffsetAvailable"`      // Maximum buffered offset available
	MinOffsetAvailable   float64 `json:"minOffsetAvailable"`      // Minimum buffered offset available
}

// PlexMedia represents media information (quality, codecs, streams)
type PlexMedia struct {
	ID                    int             `json:"id"`
	Duration              int64           `json:"duration"`
	Bitrate               int             `json:"bitrate"` // Kbps
	Width                 int             `json:"width"`   // Video width
	Height                int             `json:"height"`  // Video height
	AspectRatio           float64         `json:"aspectRatio"`
	AudioChannels         int             `json:"audioChannels"`
	AudioCodec            string          `json:"audioCodec"`
	VideoCodec            string          `json:"videoCodec"`
	VideoResolution       string          `json:"videoResolution"` // "4k", "1080", "720", "sd"
	Container             string          `json:"container"`
	VideoFrameRate        string          `json:"videoFrameRate"`
	AudioProfile          string          `json:"audioProfile"`
	VideoProfile          string          `json:"videoProfile"`
	Part                  []PlexMediaPart `json:"Part,omitempty"`
	OptimizedForStreaming bool            `json:"optimizedForStreaming"`
	Has64bitOffsets       bool            `json:"has64bitOffsets"`
}

// PlexMediaPart represents a media part with streams
type PlexMediaPart struct {
	ID                    int          `json:"id"`
	Key                   string       `json:"key"`
	Duration              int64        `json:"duration"`
	File                  string       `json:"file"`
	Size                  int64        `json:"size"`
	Container             string       `json:"container"`
	VideoProfile          string       `json:"videoProfile"`
	Has64bitOffsets       bool         `json:"has64bitOffsets"`
	OptimizedForStreaming bool         `json:"optimizedForStreaming"`
	Stream                []PlexStream `json:"Stream,omitempty"`
}

// PlexStream represents a single stream (video/audio/subtitle)
type PlexStream struct {
	ID                   int     `json:"id"`
	StreamType           int     `json:"streamType"` // 1=video, 2=audio, 3=subtitle
	Default              bool    `json:"default"`
	Codec                string  `json:"codec"`
	Index                int     `json:"index"`
	Bitrate              int     `json:"bitrate"`
	Language             string  `json:"language,omitempty"`
	LanguageCode         string  `json:"languageCode,omitempty"`
	BitDepth             int     `json:"bitDepth,omitempty"`
	ChromaLocation       string  `json:"chromaLocation,omitempty"`
	ChromaSubsampling    string  `json:"chromaSubsampling,omitempty"`
	ColorPrimaries       string  `json:"colorPrimaries,omitempty"`
	ColorRange           string  `json:"colorRange,omitempty"`
	ColorSpace           string  `json:"colorSpace,omitempty"`
	ColorTrc             string  `json:"colorTrc,omitempty"`
	FrameRate            float64 `json:"frameRate,omitempty"`
	Height               int     `json:"height,omitempty"`
	Width                int     `json:"width,omitempty"`
	DisplayTitle         string  `json:"displayTitle"`
	ExtendedDisplayTitle string  `json:"extendedDisplayTitle"`
	Selected             bool    `json:"selected,omitempty"`
	Channels             int     `json:"channels,omitempty"`
	Profile              string  `json:"profile,omitempty"`
	SamplingRate         int     `json:"samplingRate,omitempty"`
	Title                string  `json:"title,omitempty"`
}

// ===================================================================================================
// Helper Methods for Transcode Monitoring
// ===================================================================================================

// IsTranscoding returns true if the session is actively transcoding
func (s *PlexSession) IsTranscoding() bool {
	return s.TranscodeSession != nil && s.TranscodeSession.VideoDecision == "transcode"
}

// GetTranscodeProgress returns transcode progress percentage (0-100)
func (s *PlexSession) GetTranscodeProgress() float64 {
	if s.TranscodeSession != nil {
		return s.TranscodeSession.Progress
	}
	return 0
}

// IsHardwareAccelerated returns true if hardware acceleration is being used
func (s *PlexSession) IsHardwareAccelerated() bool {
	if s.TranscodeSession == nil {
		return false
	}
	return s.TranscodeSession.TranscodeHwDecoding != "" || s.TranscodeSession.TranscodeHwEncoding != ""
}

// GetHardwareAccelerationType returns the hardware acceleration type
// Returns: "Quick Sync", "NVENC", "VAAPI", "VideoToolbox", "Software", or "Unknown"
//
//nolint:gocyclo // Hardware acceleration type mapping requires multiple conditions
func (s *PlexSession) GetHardwareAccelerationType() string {
	if s.TranscodeSession == nil {
		return "Direct Play"
	}

	// Check decoder first (if present, usually matches encoder)
	hwDecoder := s.TranscodeSession.TranscodeHwDecoding
	hwEncoder := s.TranscodeSession.TranscodeHwEncoding

	if hwDecoder == "" && hwEncoder == "" {
		if s.TranscodeSession.VideoDecision == "transcode" {
			return "Software"
		}
		return "Direct Play"
	}

	// Map Plex hardware acceleration types to user-friendly names
	switch {
	case hwDecoder == "qsv" || hwEncoder == "qsv":
		return "Quick Sync" // Intel Quick Sync Video
	case hwDecoder == "nvenc" || hwEncoder == "nvenc":
		return "NVENC" // NVIDIA NVENC
	case hwDecoder == "vaapi" || hwEncoder == "vaapi":
		return "VAAPI" // Video Acceleration API (Linux)
	case hwDecoder == "videotoolbox" || hwEncoder == "videotoolbox":
		return "VideoToolbox" // Apple VideoToolbox
	case hwDecoder == "mediacodec" || hwEncoder == "mediacodec":
		return "MediaCodec" // Android MediaCodec
	case hwDecoder == "mf" || hwEncoder == "mf":
		return "MediaFoundation" // Windows Media Foundation
	default:
		return "Unknown"
	}
}

// GetQualityTransition returns source -> target quality string (e.g., "4K → 1080p")
func (s *PlexSession) GetQualityTransition() string {
	if len(s.Media) == 0 {
		return "Unknown"
	}

	sourceRes := s.Media[0].VideoResolution
	if s.TranscodeSession == nil {
		return sourceRes // Direct play
	}

	targetHeight := s.TranscodeSession.Height
	targetRes := ""
	switch {
	case targetHeight >= 2160:
		targetRes = "4K"
	case targetHeight >= 1080:
		targetRes = "1080p"
	case targetHeight >= 720:
		targetRes = "720p"
	case targetHeight >= 480:
		targetRes = "480p"
	default:
		targetRes = "SD"
	}

	// Normalize source resolution for comparison (remove 'p' suffix if present)
	normalizedSource := strings.TrimSuffix(sourceRes, "p")
	normalizedTarget := strings.TrimSuffix(targetRes, "p")

	if normalizedSource == normalizedTarget {
		return sourceRes
	}

	return fmt.Sprintf("%s → %s", sourceRes, targetRes)
}

// GetCodecTransition returns source -> target codec string (e.g., "HEVC → H.264")
func (s *PlexSession) GetCodecTransition() string {
	if len(s.Media) == 0 {
		return "Unknown"
	}

	sourceCodec := s.Media[0].VideoCodec
	if s.TranscodeSession == nil {
		return codecToFriendlyName(sourceCodec) // Direct play (formatted)
	}

	targetCodec := s.TranscodeSession.VideoCodec

	// Map codec names to user-friendly format
	sourceName := codecToFriendlyName(sourceCodec)
	targetName := codecToFriendlyName(targetCodec)

	if sourceCodec == targetCodec {
		return sourceName // Return formatted name when same codec
	}

	return fmt.Sprintf("%s → %s", sourceName, targetName)
}

// codecToFriendlyName converts codec identifiers to user-friendly names
func codecToFriendlyName(codec string) string {
	switch codec {
	case "hevc", "h265":
		return "HEVC"
	case "h264", "avc":
		return "H.264"
	case "vp9":
		return "VP9"
	case "av1":
		return "AV1"
	case "mpeg4":
		return "MPEG-4"
	case "mpeg2video":
		return "MPEG-2"
	default:
		return codec
	}
}

// GetTranscodeSpeed returns transcode speed in human-readable format (e.g., "1.5x")
func (s *PlexSession) GetTranscodeSpeed() string {
	if s.TranscodeSession == nil {
		return "N/A"
	}
	return fmt.Sprintf("%.1fx", s.TranscodeSession.Speed)
}

// IsThrottled returns true if transcode is throttled due to system load
func (s *PlexSession) IsThrottled() bool {
	if s.TranscodeSession == nil {
		return false
	}
	return s.TranscodeSession.Throttled
}

// ============================================================================
// Plex Webhook Models
// ============================================================================
// These structures represent HTTP webhook payloads from Plex Media Server
// Webhooks are push-based notifications complementing the WebSocket real-time stream
// Setup: Plex Settings → Webhooks → Add webhook URL
// Events: media.play, media.pause, media.resume, media.stop, media.scrobble,
//         media.rate, library.on.deck, library.new, admin.database.backup,
//         admin.database.corrupted, device.new, playback.started

// PlexWebhook represents a Plex webhook HTTP POST payload
// Documentation: https://support.plex.tv/articles/115002267687-webhooks/
type PlexWebhook struct {
	Event    string               `json:"event"`              // Webhook event type (e.g., "media.play", "media.stop")
	User     bool                 `json:"user"`               // True if user-initiated action
	Owner    bool                 `json:"owner"`              // True if server owner triggered event
	Account  PlexWebhookAccount   `json:"Account"`            // User account information
	Server   PlexWebhookServer    `json:"Server"`             // Plex server information
	Player   PlexWebhookPlayer    `json:"Player"`             // Client/device information
	Metadata *PlexWebhookMetadata `json:"Metadata,omitempty"` // Content metadata (present for media events)
}

// PlexWebhookAccount represents the user account in webhook payload
type PlexWebhookAccount struct {
	ID    int    `json:"id"`    // Plex account ID
	Thumb string `json:"thumb"` // Profile picture URL
	Title string `json:"title"` // Username/display name
}

// PlexWebhookServer represents the Plex server in webhook payload
type PlexWebhookServer struct {
	Title string `json:"title"` // Server name
	UUID  string `json:"uuid"`  // Server machine identifier
}

// PlexWebhookPlayer represents the client/device in webhook payload
type PlexWebhookPlayer struct {
	Local         bool   `json:"local"`         // True if on local network
	PublicAddress string `json:"publicAddress"` // Client IP address (for geo-location)
	Title         string `json:"title"`         // Device name
	UUID          string `json:"uuid"`          // Device unique identifier
}

// PlexWebhookMetadata represents content metadata in webhook payload
type PlexWebhookMetadata struct {
	LibrarySectionType   string `json:"librarySectionType"`   // "movie", "show", "artist"
	RatingKey            string `json:"ratingKey"`            // Content unique identifier
	Key                  string `json:"key"`                  // Metadata API path
	ParentRatingKey      string `json:"parentRatingKey"`      // Parent (season/album) key
	GrandparentRatingKey string `json:"grandparentRatingKey"` // Grandparent (show/artist) key
	GUID                 string `json:"guid"`                 // External GUID (imdb://, tvdb://)
	LibrarySectionTitle  string `json:"librarySectionTitle"`  // Library name
	LibrarySectionID     int    `json:"librarySectionID"`     // Library section ID
	Type                 string `json:"type"`                 // Content type: "movie", "episode", "track"
	Title                string `json:"title"`                // Content title
	GrandparentTitle     string `json:"grandparentTitle"`     // Show/Artist title
	ParentTitle          string `json:"parentTitle"`          // Season/Album title
	ContentRating        string `json:"contentRating"`        // Rating (e.g., "PG-13", "TV-MA")
	Summary              string `json:"summary"`              // Description/synopsis
	Index                int    `json:"index"`                // Episode/track number
	ParentIndex          int    `json:"parentIndex"`          // Season/disc number
	Year                 int    `json:"year"`                 // Release year
	Thumb                string `json:"thumb"`                // Thumbnail URL
	Art                  string `json:"art"`                  // Background art URL
	ParentThumb          string `json:"parentThumb"`          // Parent thumbnail
	GrandparentThumb     string `json:"grandparentThumb"`     // Grandparent thumbnail
	GrandparentArt       string `json:"grandparentArt"`       // Grandparent art
	AddedAt              int64  `json:"addedAt"`              // Unix timestamp when added
	UpdatedAt            int64  `json:"updatedAt"`            // Unix timestamp last updated
}

// IsMediaEvent returns true if this is a media playback event
func (w *PlexWebhook) IsMediaEvent() bool {
	return strings.HasPrefix(w.Event, "media.")
}

// IsLibraryEvent returns true if this is a library-related event
func (w *PlexWebhook) IsLibraryEvent() bool {
	return strings.HasPrefix(w.Event, "library.")
}

// IsAdminEvent returns true if this is an admin/system event
func (w *PlexWebhook) IsAdminEvent() bool {
	return strings.HasPrefix(w.Event, "admin.") || strings.HasPrefix(w.Event, "device.")
}

// GetUsername returns the username from the webhook account
func (w *PlexWebhook) GetUsername() string {
	return w.Account.Title
}

// GetPlayerIP returns the client IP address for geo-location
func (w *PlexWebhook) GetPlayerIP() string {
	return w.Player.PublicAddress
}

// GetContentTitle returns a formatted content title
func (w *PlexWebhook) GetContentTitle() string {
	if w.Metadata == nil {
		return ""
	}
	if w.Metadata.GrandparentTitle != "" {
		// TV Show episode: "Show Name - S01E05 - Episode Title"
		return fmt.Sprintf("%s - S%02dE%02d - %s",
			w.Metadata.GrandparentTitle,
			w.Metadata.ParentIndex,
			w.Metadata.Index,
			w.Metadata.Title)
	}
	// Movie or standalone content
	return w.Metadata.Title
}
