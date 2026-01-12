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
// Emby WebSocket Notification Models
// ============================================================================
// These structures represent real-time notifications from Emby's WebSocket endpoint
// Endpoint: ws://{emby_url}/embywebsocket?api_key={api_key}&deviceId={deviceId}
// Documentation: https://dev.emby.media/doc/restapi/Web-Socket.html

// EmbyWebSocketMessage represents a WebSocket message from Emby
type EmbyWebSocketMessage struct {
	MessageType string `json:"MessageType"` // Message type: "Sessions", "UserDataChanged", "Play", "Playstate", etc.
	Data        any    `json:"Data"`        // Message payload (type varies by MessageType)
}

// EmbySessionsMessage represents the Sessions subscription response
type EmbySessionsMessage struct {
	MessageType string        `json:"MessageType"` // "Sessions"
	Data        []EmbySession `json:"Data"`        // Array of active sessions
}

// EmbySession represents an active playback session from Emby
type EmbySession struct {
	// Session identification
	ID                 string `json:"Id"`                 // Unique session identifier
	Client             string `json:"Client"`             // Client application name
	DeviceID           string `json:"DeviceId"`           // Unique device identifier
	DeviceName         string `json:"DeviceName"`         // Device friendly name
	DeviceType         string `json:"DeviceType"`         // Device type
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
	NowPlayingItem  *EmbyNowPlayingItem  `json:"NowPlayingItem,omitempty"`  // Currently playing content
	PlayState       *EmbyPlayState       `json:"PlayState,omitempty"`       // Playback state details
	TranscodingInfo *EmbyTranscodingInfo `json:"TranscodingInfo,omitempty"` // Transcode session details

	// Capabilities
	Capabilities *EmbyCapabilities `json:"Capabilities,omitempty"`

	// Additional users (shared watching)
	AdditionalUsers []EmbyAdditionalUser `json:"AdditionalUsers,omitempty"`

	// Server metadata
	ServerID string `json:"ServerId,omitempty"` // Server identifier
}

// EmbyNowPlayingItem represents the currently playing content
type EmbyNowPlayingItem struct {
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
	MediaStreams []EmbyMediaStream `json:"MediaStreams,omitempty"`

	// Images
	PrimaryImageTag   string   `json:"PrimaryImageTag,omitempty"`
	BackdropImageTags []string `json:"BackdropImageTags,omitempty"`

	// External IDs
	ProviderIDs map[string]string `json:"ProviderIds,omitempty"` // IMDB, TVDB, TMDB, etc.
}

// EmbyPlayState represents playback state details
type EmbyPlayState struct {
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

// EmbyTranscodingInfo represents transcode session details
type EmbyTranscodingInfo struct {
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

// EmbyMediaStream represents a media stream (video/audio/subtitle)
type EmbyMediaStream struct {
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

// EmbyCapabilities represents client capabilities
type EmbyCapabilities struct {
	PlayableMediaTypes           []string `json:"PlayableMediaTypes,omitempty"`
	SupportedCommands            []string `json:"SupportedCommands,omitempty"`
	SupportsMediaControl         bool     `json:"SupportsMediaControl"`
	SupportsPersistentIdentifier bool     `json:"SupportsPersistentIdentifier"`
}

// EmbyAdditionalUser represents additional users in a shared session
type EmbyAdditionalUser struct {
	UserID   string `json:"UserId"`
	UserName string `json:"UserName"`
}

// ============================================================================
// Emby Webhook Models (requires Emby webhook plugin)
// ============================================================================

// EmbyWebhook represents a webhook payload from Emby
type EmbyWebhook struct {
	// Event information
	Event     string    `json:"Event"`     // "playback.start", "playback.stop", "playback.pause", "playback.unpause"
	Timestamp time.Time `json:"Timestamp"` // Event timestamp

	// Server information
	ServerID      string `json:"Server.Id,omitempty"`
	ServerName    string `json:"Server.Name,omitempty"`
	ServerVersion string `json:"Server.Version,omitempty"`

	// User information
	UserID   string `json:"User.Id,omitempty"`
	UserName string `json:"User.Name,omitempty"`

	// Item information
	ItemID   string `json:"Item.Id,omitempty"`
	ItemName string `json:"Item.Name,omitempty"`
	ItemType string `json:"Item.Type,omitempty"` // "Movie", "Episode", "Audio"

	// TV Show specific
	SeriesID   string `json:"Item.SeriesId,omitempty"`
	SeriesName string `json:"Item.SeriesName,omitempty"`
	SeasonID   string `json:"Item.SeasonId,omitempty"`
	SeasonName string `json:"Item.SeasonName,omitempty"`

	// Playback information
	PlaybackPositionTicks int64  `json:"PlaybackInfo.PositionTicks,omitempty"`
	RunTimeTicks          int64  `json:"Item.RunTimeTicks,omitempty"`
	IsPaused              bool   `json:"PlaybackInfo.IsPaused,omitempty"`
	PlayMethod            string `json:"PlaybackInfo.PlayMethod,omitempty"`

	// Device information
	DeviceID   string `json:"Device.Id,omitempty"`
	DeviceName string `json:"Device.Name,omitempty"`
	ClientName string `json:"Device.AppName,omitempty"`

	// Connection information
	RemoteEndPoint string `json:"Session.RemoteEndPoint,omitempty"` // Client IP address

	// Provider IDs
	ProviderImdb string `json:"Item.ProviderIds.Imdb,omitempty"`
	ProviderTmdb string `json:"Item.ProviderIds.Tmdb,omitempty"`
	ProviderTvdb string `json:"Item.ProviderIds.Tvdb,omitempty"`
}

// ============================================================================
// Helper Methods for Emby Sessions
// ============================================================================

// IsPlaying returns true if the session is actively playing content
func (s *EmbySession) IsPlaying() bool {
	return s.NowPlayingItem != nil && s.PlayState != nil && !s.PlayState.IsPaused
}

// IsPaused returns true if the session has content paused
func (s *EmbySession) IsPaused() bool {
	return s.NowPlayingItem != nil && s.PlayState != nil && s.PlayState.IsPaused
}

// IsActive returns true if the session has active content (playing or paused)
func (s *EmbySession) IsActive() bool {
	return s.NowPlayingItem != nil
}

// GetIPAddress returns the client IP address
func (s *EmbySession) GetIPAddress() string {
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
func (s *EmbySession) GetPositionSeconds() int64 {
	if s.PlayState == nil {
		return 0
	}
	// Emby uses ticks (100ns units)
	return s.PlayState.PositionTicks / 10000000
}

// GetDurationSeconds returns the content duration in seconds
func (s *EmbySession) GetDurationSeconds() int64 {
	if s.NowPlayingItem == nil {
		return 0
	}
	return s.NowPlayingItem.RunTimeTicks / 10000000
}

// GetPercentComplete returns the playback progress percentage
func (s *EmbySession) GetPercentComplete() int {
	duration := s.GetDurationSeconds()
	if duration == 0 {
		return 0
	}
	return int((s.GetPositionSeconds() * 100) / duration)
}

// GetTranscodeDecision returns the transcode decision string
func (s *EmbySession) GetTranscodeDecision() string {
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
func (n *EmbyNowPlayingItem) GetMediaType() string {
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
func (s *EmbySession) GetContentTitle() string {
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
func (s *EmbySession) IsTranscoding() bool {
	return s.PlayState != nil && s.PlayState.PlayMethod == "Transcode"
}

// GetHardwareAccelerationType returns the hardware acceleration type if transcoding
func (s *EmbySession) GetHardwareAccelerationType() string {
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
// Helper Methods for Emby Webhooks
// ============================================================================

// IsPlaybackEvent returns true if this is a playback-related event
func (w *EmbyWebhook) IsPlaybackEvent() bool {
	return strings.HasPrefix(w.Event, "playback.")
}

// GetMediaType returns the normalized media type
func (w *EmbyWebhook) GetMediaType() string {
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
func (w *EmbyWebhook) GetContentTitle() string {
	if w.SeriesName != "" {
		return fmt.Sprintf("%s - %s - %s", w.SeriesName, w.SeasonName, w.ItemName)
	}
	return w.ItemName
}

// GetPercentComplete returns the playback progress percentage
func (w *EmbyWebhook) GetPercentComplete() int {
	if w.RunTimeTicks == 0 {
		return 0
	}
	return int((w.PlaybackPositionTicks * 100) / w.RunTimeTicks)
}

// GetIPAddress returns the client IP address from the webhook
func (w *EmbyWebhook) GetIPAddress() string {
	if w.RemoteEndPoint == "" {
		return ""
	}
	// Handle IPv6 addresses in brackets
	if strings.HasPrefix(w.RemoteEndPoint, "[") {
		if idx := strings.LastIndex(w.RemoteEndPoint, "]:"); idx != -1 {
			return w.RemoteEndPoint[1:idx]
		}
		return strings.Trim(w.RemoteEndPoint, "[]")
	}
	// Handle IPv4 addresses with port
	if idx := strings.LastIndex(w.RemoteEndPoint, ":"); idx != -1 {
		return w.RemoteEndPoint[:idx]
	}
	return w.RemoteEndPoint
}
