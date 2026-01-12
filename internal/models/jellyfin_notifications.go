// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package models

import (
	"fmt"
	"strings"
	"time"
)

// ============================================================================
// Jellyfin WebSocket Notification Models
// ============================================================================
// These structures represent real-time notifications from Jellyfin's WebSocket endpoint
// Endpoint: ws://{jellyfin_url}/socket?api_key={api_key}
// Documentation: https://api.jellyfin.org/

// JellyfinWebSocketMessage represents a WebSocket message from Jellyfin
type JellyfinWebSocketMessage struct {
	MessageType string `json:"MessageType"` // Message type: "Sessions", "UserDataChanged", "Play", "Playstate", etc.
	MessageID   string `json:"MessageId"`   // Unique message identifier
	Data        any    `json:"Data"`        // Message payload (type varies by MessageType)
}

// JellyfinSessionsMessage represents the Sessions subscription response
type JellyfinSessionsMessage struct {
	MessageType string            `json:"MessageType"` // "Sessions"
	Data        []JellyfinSession `json:"Data"`        // Array of active sessions
}

// JellyfinSession represents an active playback session from Jellyfin
type JellyfinSession struct {
	// Session identification
	ID                 string `json:"Id"`                 // Unique session identifier
	Client             string `json:"Client"`             // Client application name
	DeviceID           string `json:"DeviceId"`           // Unique device identifier
	DeviceName         string `json:"DeviceName"`         // Device friendly name
	DeviceType         string `json:"DeviceType"`         // Device type (e.g., "Browser", "AndroidTV")
	ApplicationVersion string `json:"ApplicationVersion"` // Client version

	// User information
	UserID              string `json:"UserId"`                        // User ID
	UserName            string `json:"UserName"`                      // Username
	UserPrimaryImageTag string `json:"UserPrimaryImageTag,omitempty"` // User avatar tag

	// Connection details
	RemoteEndPoint      string `json:"RemoteEndPoint"`                // Client IP address
	LastActivityDate    string `json:"LastActivityDate"`              // ISO timestamp of last activity
	LastPlaybackCheckIn string `json:"LastPlaybackCheckIn,omitempty"` // ISO timestamp of last playback check-in

	// Playback state
	NowPlayingItem  *JellyfinNowPlayingItem  `json:"NowPlayingItem,omitempty"`  // Currently playing content
	PlayState       *JellyfinPlayState       `json:"PlayState,omitempty"`       // Playback state details
	TranscodingInfo *JellyfinTranscodingInfo `json:"TranscodingInfo,omitempty"` // Transcode session details

	// Capabilities
	Capabilities *JellyfinCapabilities `json:"Capabilities,omitempty"`

	// Additional users (shared watching)
	AdditionalUsers []JellyfinAdditionalUser `json:"AdditionalUsers,omitempty"`

	// Server metadata
	ServerID string `json:"ServerId,omitempty"` // Server identifier
}

// JellyfinNowPlayingItem represents the currently playing content
type JellyfinNowPlayingItem struct {
	// Content identification
	ID        string `json:"Id"`        // Item ID (rating key equivalent)
	Name      string `json:"Name"`      // Content title
	Type      string `json:"Type"`      // "Movie", "Episode", "MusicAlbum", "Audio"
	MediaType string `json:"MediaType"` // "Video", "Audio"

	// TV Show specific
	SeriesID          string `json:"SeriesId,omitempty"`          // Series ID for episodes
	SeriesName        string `json:"SeriesName,omitempty"`        // Series name
	SeasonID          string `json:"SeasonId,omitempty"`          // Season ID for episodes
	SeasonName        string `json:"SeasonName,omitempty"`        // Season name
	IndexNumber       int    `json:"IndexNumber,omitempty"`       // Episode number
	ParentIndexNumber int    `json:"ParentIndexNumber,omitempty"` // Season number

	// Music specific
	AlbumID        string   `json:"AlbumId,omitempty"`        // Album ID for tracks
	AlbumArtist    string   `json:"AlbumArtist,omitempty"`    // Album artist
	Album          string   `json:"Album,omitempty"`          // Album name
	Artists        []string `json:"Artists,omitempty"`        // Artists
	IndexNumberEnd int      `json:"IndexNumberEnd,omitempty"` // Track number

	// Media information
	RunTimeTicks   int64  `json:"RunTimeTicks"`             // Duration in ticks (100ns units)
	ProductionYear int    `json:"ProductionYear,omitempty"` // Release year
	Container      string `json:"Container,omitempty"`      // Container format

	// Media streams (video/audio/subtitle)
	MediaStreams []JellyfinMediaStream `json:"MediaStreams,omitempty"`

	// Images
	PrimaryImageTag   string   `json:"PrimaryImageTag,omitempty"`
	BackdropImageTags []string `json:"BackdropImageTags,omitempty"`

	// External IDs
	ProviderIDs map[string]string `json:"ProviderIds,omitempty"` // IMDB, TVDB, TMDB, etc.
}

// JellyfinPlayState represents playback state details
type JellyfinPlayState struct {
	PositionTicks       int64  `json:"PositionTicks"`                 // Current position in ticks
	CanSeek             bool   `json:"CanSeek"`                       // Can seek
	IsPaused            bool   `json:"IsPaused"`                      // Is paused
	IsMuted             bool   `json:"IsMuted"`                       // Is muted
	VolumeLevel         int    `json:"VolumeLevel,omitempty"`         // Volume level (0-100)
	AudioStreamIndex    int    `json:"AudioStreamIndex,omitempty"`    // Selected audio stream
	SubtitleStreamIndex int    `json:"SubtitleStreamIndex,omitempty"` // Selected subtitle stream (-1 = none)
	MediaSourceID       string `json:"MediaSourceId,omitempty"`       // Media source ID
	PlayMethod          string `json:"PlayMethod,omitempty"`          // "DirectPlay", "DirectStream", "Transcode"
	RepeatMode          string `json:"RepeatMode,omitempty"`          // Repeat mode
}

// JellyfinTranscodingInfo represents transcode session details
type JellyfinTranscodingInfo struct {
	AudioCodec               string   `json:"AudioCodec,omitempty"`
	VideoCodec               string   `json:"VideoCodec,omitempty"`
	Container                string   `json:"Container,omitempty"`
	IsVideoDirect            bool     `json:"IsVideoDirect"`
	IsAudioDirect            bool     `json:"IsAudioDirect"`
	Bitrate                  int      `json:"Bitrate,omitempty"`
	Framerate                float64  `json:"Framerate,omitempty"`
	CompletionPercentage     float64  `json:"CompletionPercentage,omitempty"`
	Width                    int      `json:"Width,omitempty"`
	Height                   int      `json:"Height,omitempty"`
	AudioChannels            int      `json:"AudioChannels,omitempty"`
	HardwareAccelerationType string   `json:"HardwareAccelerationType,omitempty"` // "vaapi", "nvenc", "qsv", etc.
	TranscodeReasons         []string `json:"TranscodeReasons,omitempty"`
}

// JellyfinMediaStream represents a media stream (video/audio/subtitle)
type JellyfinMediaStream struct {
	Codec            string  `json:"Codec"`
	CodecTag         string  `json:"CodecTag,omitempty"`
	Language         string  `json:"Language,omitempty"`
	DisplayTitle     string  `json:"DisplayTitle,omitempty"`
	Type             string  `json:"Type"` // "Video", "Audio", "Subtitle"
	Index            int     `json:"Index"`
	IsDefault        bool    `json:"IsDefault"`
	IsForced         bool    `json:"IsForced"`
	IsExternal       bool    `json:"IsExternal"`
	Height           int     `json:"Height,omitempty"`
	Width            int     `json:"Width,omitempty"`
	BitRate          int     `json:"BitRate,omitempty"`
	Channels         int     `json:"Channels,omitempty"`
	SampleRate       int     `json:"SampleRate,omitempty"`
	BitDepth         int     `json:"BitDepth,omitempty"`
	AspectRatio      string  `json:"AspectRatio,omitempty"`
	PixelFormat      string  `json:"PixelFormat,omitempty"`
	AverageFrameRate float64 `json:"AverageFrameRate,omitempty"`
}

// JellyfinCapabilities represents client capabilities
type JellyfinCapabilities struct {
	PlayableMediaTypes           []string `json:"PlayableMediaTypes,omitempty"`
	SupportedCommands            []string `json:"SupportedCommands,omitempty"`
	SupportsMediaControl         bool     `json:"SupportsMediaControl"`
	SupportsPersistentIdentifier bool     `json:"SupportsPersistentIdentifier"`
}

// JellyfinAdditionalUser represents additional users in a shared session
type JellyfinAdditionalUser struct {
	UserID   string `json:"UserId"`
	UserName string `json:"UserName"`
}

// ============================================================================
// Jellyfin Webhook Models (requires jellyfin-plugin-webhook)
// ============================================================================

// JellyfinWebhook represents a webhook payload from Jellyfin
// Requires: https://github.com/jellyfin/jellyfin-plugin-webhook
type JellyfinWebhook struct {
	// Event information
	NotificationType string    `json:"NotificationType"` // "PlaybackStart", "PlaybackStop", "PlaybackProgress", etc.
	Timestamp        time.Time `json:"Timestamp"`        // Event timestamp

	// Server information
	ServerID      string `json:"ServerId"`
	ServerName    string `json:"ServerName"`
	ServerVersion string `json:"ServerVersion"`
	ServerURL     string `json:"ServerUrl,omitempty"`

	// User information
	UserID   string `json:"UserId"`
	UserName string `json:"UserName,omitempty"`

	// Item information
	ItemID   string `json:"ItemId,omitempty"`
	ItemName string `json:"ItemName,omitempty"`
	ItemType string `json:"ItemType,omitempty"` // "Movie", "Episode", "Audio"

	// TV Show specific
	SeriesID   string `json:"SeriesId,omitempty"`
	SeriesName string `json:"SeriesName,omitempty"`
	SeasonID   string `json:"SeasonId,omitempty"`
	SeasonName string `json:"SeasonName,omitempty"`

	// Playback information
	PlaybackPositionTicks int64  `json:"PlaybackPositionTicks,omitempty"`
	RunTimeTicks          int64  `json:"RunTimeTicks,omitempty"`
	IsPaused              bool   `json:"IsPaused,omitempty"`
	PlayMethod            string `json:"PlayMethod,omitempty"` // "DirectPlay", "Transcode"

	// Device information
	DeviceID   string `json:"DeviceId,omitempty"`
	DeviceName string `json:"DeviceName,omitempty"`
	ClientName string `json:"ClientName,omitempty"`

	// Connection information
	RemoteEndPoint string `json:"RemoteEndPoint,omitempty"` // Client IP address

	// Provider IDs (external identifiers)
	ProviderImdb string `json:"Provider_imdb,omitempty"`
	ProviderTmdb string `json:"Provider_tmdb,omitempty"`
	ProviderTvdb string `json:"Provider_tvdb,omitempty"`
}

// ============================================================================
// Helper Methods for Jellyfin Sessions
// ============================================================================

// IsPlaying returns true if the session is actively playing content
func (s *JellyfinSession) IsPlaying() bool {
	return s.NowPlayingItem != nil && s.PlayState != nil && !s.PlayState.IsPaused
}

// IsPaused returns true if the session has content paused
func (s *JellyfinSession) IsPaused() bool {
	return s.NowPlayingItem != nil && s.PlayState != nil && s.PlayState.IsPaused
}

// IsActive returns true if the session has active content (playing or paused)
func (s *JellyfinSession) IsActive() bool {
	return s.NowPlayingItem != nil
}

// GetIPAddress returns the client IP address
func (s *JellyfinSession) GetIPAddress() string {
	// RemoteEndPoint may include port, extract IP only
	if s.RemoteEndPoint == "" {
		return ""
	}
	// Handle IPv6 addresses in brackets
	if strings.HasPrefix(s.RemoteEndPoint, "[") {
		if idx := strings.LastIndex(s.RemoteEndPoint, "]:"); idx != -1 {
			return s.RemoteEndPoint[1:idx]
		}
		return strings.Trim(s.RemoteEndPoint, "[]")
	}
	// Handle IPv4 addresses
	if idx := strings.LastIndex(s.RemoteEndPoint, ":"); idx != -1 {
		return s.RemoteEndPoint[:idx]
	}
	return s.RemoteEndPoint
}

// GetPositionSeconds returns the current playback position in seconds
func (s *JellyfinSession) GetPositionSeconds() int64 {
	if s.PlayState == nil {
		return 0
	}
	// Jellyfin uses ticks (100ns units)
	return s.PlayState.PositionTicks / 10000000
}

// GetDurationSeconds returns the content duration in seconds
func (s *JellyfinSession) GetDurationSeconds() int64 {
	if s.NowPlayingItem == nil {
		return 0
	}
	return s.NowPlayingItem.RunTimeTicks / 10000000
}

// GetPercentComplete returns the playback progress percentage
func (s *JellyfinSession) GetPercentComplete() int {
	duration := s.GetDurationSeconds()
	if duration == 0 {
		return 0
	}
	return int((s.GetPositionSeconds() * 100) / duration)
}

// GetTranscodeDecision returns the transcode decision string
func (s *JellyfinSession) GetTranscodeDecision() string {
	if s.PlayState == nil {
		return ""
	}
	switch s.PlayState.PlayMethod {
	case "DirectPlay":
		return "direct play"
	case "DirectStream":
		return "direct stream"
	case "Transcode":
		return "transcode"
	default:
		return s.PlayState.PlayMethod
	}
}

// GetMediaType returns the normalized media type (movie, episode, track)
func (n *JellyfinNowPlayingItem) GetMediaType() string {
	switch strings.ToLower(n.Type) {
	case "movie":
		return "movie"
	case "episode":
		return "episode"
	case "audio", "musicvideo":
		return "track"
	default:
		return strings.ToLower(n.Type)
	}
}

// GetContentTitle returns a formatted content title
func (s *JellyfinSession) GetContentTitle() string {
	if s.NowPlayingItem == nil {
		return ""
	}
	item := s.NowPlayingItem

	// TV Show episode
	if item.SeriesName != "" {
		return fmt.Sprintf("%s - S%02dE%02d - %s",
			item.SeriesName,
			item.ParentIndexNumber,
			item.IndexNumber,
			item.Name)
	}

	// Music track
	if item.Album != "" {
		artists := strings.Join(item.Artists, ", ")
		if artists == "" {
			artists = item.AlbumArtist
		}
		return fmt.Sprintf("%s - %s", artists, item.Name)
	}

	// Movie or other content
	return item.Name
}

// IsTranscoding returns true if the session is transcoding
func (s *JellyfinSession) IsTranscoding() bool {
	return s.PlayState != nil && s.PlayState.PlayMethod == "Transcode"
}

// GetHardwareAccelerationType returns the hardware acceleration type if transcoding
func (s *JellyfinSession) GetHardwareAccelerationType() string {
	if s.TranscodingInfo == nil {
		if s.PlayState != nil && s.PlayState.PlayMethod == "DirectPlay" {
			return "Direct Play"
		}
		return ""
	}

	switch s.TranscodingInfo.HardwareAccelerationType {
	case "vaapi":
		return "VAAPI"
	case "nvenc":
		return "NVENC"
	case "qsv":
		return "Quick Sync"
	case "videotoolbox":
		return "VideoToolbox"
	case "":
		if s.TranscodingInfo.VideoCodec != "" {
			return "Software"
		}
		return ""
	default:
		return s.TranscodingInfo.HardwareAccelerationType
	}
}

// ============================================================================
// Helper Methods for Jellyfin Webhooks
// ============================================================================

// IsPlaybackEvent returns true if this is a playback-related event
func (w *JellyfinWebhook) IsPlaybackEvent() bool {
	return strings.HasPrefix(w.NotificationType, "Playback")
}

// GetMediaType returns the normalized media type
func (w *JellyfinWebhook) GetMediaType() string {
	switch strings.ToLower(w.ItemType) {
	case "movie":
		return "movie"
	case "episode":
		return "episode"
	case "audio":
		return "track"
	default:
		return strings.ToLower(w.ItemType)
	}
}

// GetContentTitle returns a formatted content title
func (w *JellyfinWebhook) GetContentTitle() string {
	if w.SeriesName != "" {
		return fmt.Sprintf("%s - %s - %s", w.SeriesName, w.SeasonName, w.ItemName)
	}
	return w.ItemName
}

// GetPercentComplete returns the playback progress percentage
func (w *JellyfinWebhook) GetPercentComplete() int {
	if w.RunTimeTicks == 0 {
		return 0
	}
	return int((w.PlaybackPositionTicks * 100) / w.RunTimeTicks)
}
