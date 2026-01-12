// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package models

// Plex REST API Expanded Models
// These structures support additional Plex API endpoints for standalone operation
// Documentation: https://plexapi.dev and https://www.plexopedia.com/plex-media-server/api/
//
// NOTE: Basic Plex types (PlexLibrarySectionsResponse, PlexActivitiesResponse, etc.)
// are defined in plex_api.go. Session types (PlexSessionsResponse, PlexSession, etc.)
// are defined in plex_notifications.go.

// ============================================================================
// Server Capabilities Models - GET /
// ============================================================================

// PlexServerCapabilitiesResponse represents the response from GET /
// This endpoint provides comprehensive server capabilities and feature flags
type PlexServerCapabilitiesResponse struct {
	MediaContainer PlexServerCapabilitiesContainer `json:"MediaContainer"`
}

// PlexServerCapabilitiesContainer wraps server capabilities information
type PlexServerCapabilitiesContainer struct {
	Size              int    `json:"size"`
	FriendlyName      string `json:"friendlyName"`
	MachineIdentifier string `json:"machineIdentifier"`
	Version           string `json:"version"`
	Platform          string `json:"platform"`
	PlatformVersion   string `json:"platformVersion"`

	Claimed            bool   `json:"claimed,omitempty"`
	MyPlex             bool   `json:"myPlex,omitempty"`
	MyPlexMappingState string `json:"myPlexMappingState,omitempty"`
	MyPlexSigninState  string `json:"myPlexSigninState,omitempty"`
	MyPlexSubscription bool   `json:"myPlexSubscription,omitempty"`
	MyPlexUsername     string `json:"myPlexUsername,omitempty"`
	OwnerFeatures      string `json:"ownerFeatures,omitempty"`

	AllowCameraUpload    bool   `json:"allowCameraUpload,omitempty"`
	AllowChannelAccess   bool   `json:"allowChannelAccess,omitempty"`
	AllowMediaDeletion   bool   `json:"allowMediaDeletion,omitempty"`
	AllowSharing         bool   `json:"allowSharing,omitempty"`
	AllowSync            bool   `json:"allowSync,omitempty"`
	AllowTuners          bool   `json:"allowTuners,omitempty"`
	BackgroundProcessing bool   `json:"backgroundProcessing,omitempty"`
	Certificate          bool   `json:"certificate,omitempty"`
	CompanionProxy       bool   `json:"companionProxy,omitempty"`
	CountryCode          string `json:"countryCode,omitempty"`
	Diagnostics          string `json:"diagnostics,omitempty"`
	EventStream          bool   `json:"eventStream,omitempty"`
	HubSearch            bool   `json:"hubSearch,omitempty"`
	ItemClusters         bool   `json:"itemClusters,omitempty"`

	LiveTV            bool `json:"livetv,omitempty"`
	MediaProviders    bool `json:"mediaProviders,omitempty"`
	Multiuser         bool `json:"multiuser,omitempty"`
	PhotoAutoTag      bool `json:"photoAutoTag,omitempty"`
	PluginHost        bool `json:"pluginHost,omitempty"`
	PushNotifications bool `json:"pushNotifications,omitempty"`
	ReadOnlyLibraries bool `json:"readOnlyLibraries,omitempty"`

	StreamingBrainABRVersion int `json:"streamingBrainABRVersion,omitempty"`
	StreamingBrainVersion    int `json:"streamingBrainVersion,omitempty"`

	Sync bool `json:"sync,omitempty"`

	TranscoderActiveVideoSessions int    `json:"transcoderActiveVideoSessions,omitempty"`
	TranscoderAudio               bool   `json:"transcoderAudio,omitempty"`
	TranscoderLyrics              bool   `json:"transcoderLyrics,omitempty"`
	TranscoderPhoto               bool   `json:"transcoderPhoto,omitempty"`
	TranscoderSubtitles           bool   `json:"transcoderSubtitles,omitempty"`
	TranscoderVideo               bool   `json:"transcoderVideo,omitempty"`
	TranscoderVideoBitrates       string `json:"transcoderVideoBitrates,omitempty"`
	TranscoderVideoQualities      string `json:"transcoderVideoQualities,omitempty"`
	TranscoderVideoResolutions    string `json:"transcoderVideoResolutions,omitempty"`

	UpdatedAt   int64 `json:"updatedAt,omitempty"`
	Updater     bool  `json:"updater,omitempty"`
	VoiceSearch bool  `json:"voiceSearch,omitempty"`

	Directory []PlexServerDirectory `json:"Directory,omitempty"`
}

// PlexServerDirectory represents a server endpoint directory
type PlexServerDirectory struct {
	Count int    `json:"count,omitempty"`
	Key   string `json:"key"`
	Title string `json:"title"`
}

// ============================================================================
// Server Identity Models - GET /identity
// ============================================================================

// PlexIdentityResponse represents the response from GET /identity
// This endpoint provides server identification information
type PlexIdentityResponse struct {
	MediaContainer PlexIdentityContainer `json:"MediaContainer"`
}

// PlexIdentityContainer wraps server identity information
type PlexIdentityContainer struct {
	Size              int    `json:"size"`
	Claimed           bool   `json:"claimed"`
	MachineIdentifier string `json:"machineIdentifier"`
	Version           string `json:"version"`
}

// ============================================================================
// Transcode Sessions Models - GET /transcode/sessions
// ============================================================================

// PlexTranscodeSessionsResponse represents the response from GET /transcode/sessions
type PlexTranscodeSessionsResponse struct {
	MediaContainer PlexTranscodeSessionsContainer `json:"MediaContainer"`
}

// PlexTranscodeSessionsContainer wraps transcode session list
type PlexTranscodeSessionsContainer struct {
	Size             int                        `json:"size"`
	TranscodeSession []PlexTranscodeSessionInfo `json:"TranscodeSession,omitempty"`
}

// PlexTranscodeSessionInfo represents a transcode session from /transcode/sessions
type PlexTranscodeSessionInfo struct {
	Key                      string  `json:"key,omitempty"`
	Throttled                bool    `json:"throttled,omitempty"`
	Complete                 bool    `json:"complete,omitempty"`
	Progress                 float64 `json:"progress,omitempty"`
	Size                     int64   `json:"size,omitempty"`
	Speed                    float64 `json:"speed,omitempty"`
	Error                    bool    `json:"error,omitempty"`
	Duration                 int64   `json:"duration,omitempty"`
	Remaining                int     `json:"remaining,omitempty"`
	Context                  string  `json:"context,omitempty"`
	SourceVideoCodec         string  `json:"sourceVideoCodec,omitempty"`
	SourceAudioCodec         string  `json:"sourceAudioCodec,omitempty"`
	VideoDecision            string  `json:"videoDecision,omitempty"`
	AudioDecision            string  `json:"audioDecision,omitempty"`
	SubtitleDecision         string  `json:"subtitleDecision,omitempty"`
	Protocol                 string  `json:"protocol,omitempty"`
	Container                string  `json:"container,omitempty"`
	VideoCodec               string  `json:"videoCodec,omitempty"`
	AudioCodec               string  `json:"audioCodec,omitempty"`
	AudioChannels            int     `json:"audioChannels,omitempty"`
	TranscodeHwRequested     bool    `json:"transcodeHwRequested,omitempty"`
	TranscodeHwDecoding      string  `json:"transcodeHwDecoding,omitempty"`
	TranscodeHwDecodingTitle string  `json:"transcodeHwDecodingTitle,omitempty"`
	TranscodeHwEncoding      string  `json:"transcodeHwEncoding,omitempty"`
	TranscodeHwEncodingTitle string  `json:"transcodeHwEncodingTitle,omitempty"`
	TranscodeHwFullPipeline  bool    `json:"transcodeHwFullPipeline,omitempty"`
	TimeStamp                float64 `json:"timeStamp,omitempty"`
	MaxOffsetAvailable       float64 `json:"maxOffsetAvailable,omitempty"`
	MinOffsetAvailable       float64 `json:"minOffsetAvailable,omitempty"`
}

// ============================================================================
// Metadata Response Models - GET /library/metadata/{ratingKey}
// ============================================================================

// PlexMetadataResponse represents the response from GET /library/metadata/{id}
type PlexMetadataResponse struct {
	MediaContainer PlexMetadataContainer `json:"MediaContainer"`
}

// PlexMetadataContainer wraps metadata response
type PlexMetadataContainer struct {
	Size                int                   `json:"size"`
	AllowSync           bool                  `json:"allowSync,omitempty"`
	Identifier          string                `json:"identifier,omitempty"`
	LibrarySectionID    int                   `json:"librarySectionID,omitempty"`
	LibrarySectionTitle string                `json:"librarySectionTitle,omitempty"`
	LibrarySectionUUID  string                `json:"librarySectionUUID,omitempty"`
	MediaTagPrefix      string                `json:"mediaTagPrefix,omitempty"`
	MediaTagVersion     int                   `json:"mediaTagVersion,omitempty"`
	Metadata            []PlexMetadataDetails `json:"Metadata,omitempty"`
}

// PlexMetadataDetails represents detailed metadata for a media item
type PlexMetadataDetails struct {
	RatingKey             string            `json:"ratingKey,omitempty"`
	Key                   string            `json:"key,omitempty"`
	GUID                  string            `json:"guid,omitempty"`
	Studio                string            `json:"studio,omitempty"`
	Type                  string            `json:"type,omitempty"`
	Title                 string            `json:"title,omitempty"`
	TitleSort             string            `json:"titleSort,omitempty"`
	ContentRating         string            `json:"contentRating,omitempty"`
	Summary               string            `json:"summary,omitempty"`
	Rating                float64           `json:"rating,omitempty"`
	AudienceRating        float64           `json:"audienceRating,omitempty"`
	ViewCount             int               `json:"viewCount,omitempty"`
	LastViewedAt          int64             `json:"lastViewedAt,omitempty"`
	Year                  int               `json:"year,omitempty"`
	Tagline               string            `json:"tagline,omitempty"`
	Thumb                 string            `json:"thumb,omitempty"`
	Art                   string            `json:"art,omitempty"`
	Duration              int64             `json:"duration,omitempty"`
	OriginallyAvailableAt string            `json:"originallyAvailableAt,omitempty"`
	AddedAt               int64             `json:"addedAt,omitempty"`
	UpdatedAt             int64             `json:"updatedAt,omitempty"`
	AudienceRatingImage   string            `json:"audienceRatingImage,omitempty"`
	PrimaryExtraKey       string            `json:"primaryExtraKey,omitempty"`
	RatingImage           string            `json:"ratingImage,omitempty"`
	Media                 []PlexMediaInfo   `json:"Media,omitempty"`
	Genre                 []PlexMetadataTag `json:"Genre,omitempty"`
	Director              []PlexMetadataTag `json:"Director,omitempty"`
	Writer                []PlexMetadataTag `json:"Writer,omitempty"`
	Country               []PlexMetadataTag `json:"Country,omitempty"`
	Role                  []PlexMetadataTag `json:"Role,omitempty"`
}

// PlexMediaInfo represents media information for metadata endpoint
type PlexMediaInfo struct {
	AspectRatio           float64                   `json:"aspectRatio,omitempty"`
	AudioChannels         int                       `json:"audioChannels,omitempty"`
	AudioCodec            string                    `json:"audioCodec,omitempty"`
	AudioProfile          string                    `json:"audioProfile,omitempty"`
	Bitrate               int                       `json:"bitrate,omitempty"`
	Container             string                    `json:"container,omitempty"`
	Duration              int64                     `json:"duration,omitempty"`
	Has64bitOffsets       bool                      `json:"has64bitOffsets,omitempty"`
	Height                int                       `json:"height,omitempty"`
	ID                    int                       `json:"id,omitempty"`
	OptimizedForStreaming int                       `json:"optimizedForStreaming,omitempty"`
	VideoCodec            string                    `json:"videoCodec,omitempty"`
	VideoFrameRate        string                    `json:"videoFrameRate,omitempty"`
	VideoProfile          string                    `json:"videoProfile,omitempty"`
	VideoResolution       string                    `json:"videoResolution,omitempty"`
	Width                 int                       `json:"width,omitempty"`
	Part                  []PlexMetadataMediaPart   `json:"Part,omitempty"`
	Stream                []PlexMetadataMediaStream `json:"Stream,omitempty"`
}

// PlexMetadataMediaPart represents a media part (file) for metadata endpoint
type PlexMetadataMediaPart struct {
	AudioProfile          string                    `json:"audioProfile,omitempty"`
	Container             string                    `json:"container,omitempty"`
	Decision              string                    `json:"decision,omitempty"`
	Duration              int64                     `json:"duration,omitempty"`
	File                  string                    `json:"file,omitempty"`
	Has64bitOffsets       bool                      `json:"has64bitOffsets,omitempty"`
	HasThumbnail          string                    `json:"hasThumbnail,omitempty"`
	ID                    int                       `json:"id,omitempty"`
	Indexes               string                    `json:"indexes,omitempty"`
	Key                   string                    `json:"key,omitempty"`
	OptimizedForStreaming bool                      `json:"optimizedForStreaming,omitempty"`
	Selected              bool                      `json:"selected,omitempty"`
	Size                  int64                     `json:"size,omitempty"`
	VideoProfile          string                    `json:"videoProfile,omitempty"`
	Stream                []PlexMetadataMediaStream `json:"Stream,omitempty"`
}

// PlexMetadataMediaStream represents a media stream (video, audio, subtitle)
type PlexMetadataMediaStream struct {
	BitDepth             int     `json:"bitDepth,omitempty"`
	Bitrate              int     `json:"bitrate,omitempty"`
	Cabac                int     `json:"cabac,omitempty"`
	Channels             int     `json:"channels,omitempty"`
	ChromaLocation       string  `json:"chromaLocation,omitempty"`
	ChromaSubsampling    string  `json:"chromaSubsampling,omitempty"`
	Codec                string  `json:"codec,omitempty"`
	CodecID              string  `json:"codecID,omitempty"`
	ColorPrimaries       string  `json:"colorPrimaries,omitempty"`
	ColorRange           string  `json:"colorRange,omitempty"`
	ColorSpace           string  `json:"colorSpace,omitempty"`
	ColorTrc             string  `json:"colorTrc,omitempty"`
	Decision             string  `json:"decision,omitempty"`
	Default              bool    `json:"default,omitempty"`
	DisplayTitle         string  `json:"displayTitle,omitempty"`
	DOVIBLCompatID       int     `json:"DOVIBLCompatID,omitempty"`
	DOVIBLPresent        bool    `json:"DOVIBLPresent,omitempty"`
	DOVIELPresent        bool    `json:"DOVIELPresent,omitempty"`
	DOVILevel            int     `json:"DOVILevel,omitempty"`
	DOVIPresent          bool    `json:"DOVIPresent,omitempty"`
	DOVIProfile          int     `json:"DOVIProfile,omitempty"`
	DOVIRPUPresent       bool    `json:"DOVIRPUPresent,omitempty"`
	DOVIVersion          string  `json:"DOVIVersion,omitempty"`
	Duration             int64   `json:"duration,omitempty"`
	ExtendedDisplayTitle string  `json:"extendedDisplayTitle,omitempty"`
	FrameRate            float64 `json:"frameRate,omitempty"`
	HasScalingMatrix     bool    `json:"hasScalingMatrix,omitempty"`
	Height               int     `json:"height,omitempty"`
	ID                   int     `json:"id,omitempty"`
	Index                int     `json:"index,omitempty"`
	Language             string  `json:"language,omitempty"`
	LanguageCode         string  `json:"languageCode,omitempty"`
	LanguageTag          string  `json:"languageTag,omitempty"`
	Level                int     `json:"level,omitempty"`
	Location             string  `json:"location,omitempty"`
	Profile              string  `json:"profile,omitempty"`
	RefFrames            int     `json:"refFrames,omitempty"`
	SamplingRate         int     `json:"samplingRate,omitempty"`
	ScanType             string  `json:"scanType,omitempty"`
	Selected             bool    `json:"selected,omitempty"`
	StreamIdentifier     string  `json:"streamIdentifier,omitempty"`
	StreamType           int     `json:"streamType,omitempty"`
	Title                string  `json:"title,omitempty"`
	Width                int     `json:"width,omitempty"`
}

// PlexMetadataTag represents a metadata tag (genre, director, etc.)
type PlexMetadataTag struct {
	ID     int    `json:"id,omitempty"`
	Filter string `json:"filter,omitempty"`
	Tag    string `json:"tag,omitempty"`
	Role   string `json:"role,omitempty"`
	Thumb  string `json:"thumb,omitempty"`
}

// ============================================================================
// Devices Models - GET /devices
// ============================================================================

// PlexDevicesResponse represents the response from GET /devices
type PlexDevicesResponse struct {
	MediaContainer PlexDevicesContainer `json:"MediaContainer"`
}

// PlexDevicesContainer wraps devices list
type PlexDevicesContainer struct {
	Size   int          `json:"size"`
	Device []PlexDevice `json:"Device,omitempty"`
}

// PlexDevice represents a Plex device
type PlexDevice struct {
	ID                     int              `json:"id,omitempty"`
	Name                   string           `json:"name,omitempty"`
	Platform               string           `json:"platform,omitempty"`
	ClientIdentifier       string           `json:"clientIdentifier,omitempty"`
	CreatedAt              int64            `json:"createdAt,omitempty"`
	LastSeenAt             int64            `json:"lastSeenAt,omitempty"`
	Provides               string           `json:"provides,omitempty"`
	Owned                  bool             `json:"owned,omitempty"`
	AccessToken            string           `json:"accessToken,omitempty"`
	PublicAddress          string           `json:"publicAddress,omitempty"`
	HTTPSRequired          bool             `json:"httpsRequired,omitempty"`
	Synced                 bool             `json:"synced,omitempty"`
	Relay                  bool             `json:"relay,omitempty"`
	DNSRebindingProtection bool             `json:"dnsRebindingProtection,omitempty"`
	NATLoopbackSupported   bool             `json:"natLoopbackSupported,omitempty"`
	PublicAddressMatches   bool             `json:"publicAddressMatches,omitempty"`
	Presence               bool             `json:"presence,omitempty"`
	Connection             []PlexConnection `json:"Connection,omitempty"`
}

// PlexConnection represents a device connection
type PlexConnection struct {
	Protocol string `json:"protocol,omitempty"`
	Address  string `json:"address,omitempty"`
	Port     int    `json:"port,omitempty"`
	URI      string `json:"uri,omitempty"`
	Local    bool   `json:"local,omitempty"`
}

// ============================================================================
// Accounts Models - GET /accounts
// ============================================================================

// PlexAccountsResponse represents the response from GET /accounts
type PlexAccountsResponse struct {
	MediaContainer PlexAccountsContainer `json:"MediaContainer"`
}

// PlexAccountsContainer wraps accounts list
type PlexAccountsContainer struct {
	Size    int                  `json:"size"`
	Account []PlexAccountDetails `json:"Account,omitempty"`
}

// PlexAccountDetails represents a Plex account
type PlexAccountDetails struct {
	ID                      int    `json:"id,omitempty"`
	Key                     string `json:"key,omitempty"`
	Name                    string `json:"name,omitempty"`
	DefaultAudioLanguage    string `json:"defaultAudioLanguage,omitempty"`
	AutoSelectAudio         bool   `json:"autoSelectAudio,omitempty"`
	DefaultSubtitleLanguage string `json:"defaultSubtitleLanguage,omitempty"`
	SubtitleMode            int    `json:"subtitleMode,omitempty"`
	Thumb                   string `json:"thumb,omitempty"`
}

// ============================================================================
// On Deck Models - GET /library/onDeck
// ============================================================================

// PlexOnDeckResponse represents the response from GET /library/onDeck
type PlexOnDeckResponse struct {
	MediaContainer PlexOnDeckContainer `json:"MediaContainer"`
}

// PlexOnDeckContainer wraps on-deck items
type PlexOnDeckContainer struct {
	Size            int                   `json:"size"`
	AllowSync       bool                  `json:"allowSync,omitempty"`
	Identifier      string                `json:"identifier,omitempty"`
	MediaTagPrefix  string                `json:"mediaTagPrefix,omitempty"`
	MediaTagVersion int                   `json:"mediaTagVersion,omitempty"`
	MixedParents    bool                  `json:"mixedParents,omitempty"`
	Metadata        []PlexMetadataDetails `json:"Metadata,omitempty"`
}

// ============================================================================
// Playlists Models - GET /playlists
// ============================================================================

// PlexPlaylistsResponse represents the response from GET /playlists
type PlexPlaylistsResponse struct {
	MediaContainer PlexPlaylistsContainer `json:"MediaContainer"`
}

// PlexPlaylistsContainer wraps playlists
type PlexPlaylistsContainer struct {
	Size     int            `json:"size"`
	Metadata []PlexPlaylist `json:"Metadata,omitempty"`
}

// PlexPlaylist represents a playlist
type PlexPlaylist struct {
	RatingKey    string `json:"ratingKey,omitempty"`
	Key          string `json:"key,omitempty"`
	GUID         string `json:"guid,omitempty"`
	Type         string `json:"type,omitempty"`
	Title        string `json:"title,omitempty"`
	Summary      string `json:"summary,omitempty"`
	Smart        bool   `json:"smart,omitempty"`
	PlaylistType string `json:"playlistType,omitempty"`
	Composite    string `json:"composite,omitempty"`
	ViewCount    int    `json:"viewCount,omitempty"`
	LastViewedAt int64  `json:"lastViewedAt,omitempty"`
	Duration     int64  `json:"duration,omitempty"`
	LeafCount    int    `json:"leafCount,omitempty"`
	AddedAt      int64  `json:"addedAt,omitempty"`
	UpdatedAt    int64  `json:"updatedAt,omitempty"`
}

// ============================================================================
// Search Models - GET /library/sections/{key}/search
// ============================================================================

// PlexSearchResponse represents the response from search endpoint
type PlexSearchResponse struct {
	MediaContainer PlexSearchContainer `json:"MediaContainer"`
}

// PlexSearchContainer wraps search results
type PlexSearchContainer struct {
	Size                int                   `json:"size"`
	AllowSync           bool                  `json:"allowSync,omitempty"`
	Art                 string                `json:"art,omitempty"`
	Identifier          string                `json:"identifier,omitempty"`
	LibrarySectionID    int                   `json:"librarySectionID,omitempty"`
	LibrarySectionTitle string                `json:"librarySectionTitle,omitempty"`
	LibrarySectionUUID  string                `json:"librarySectionUUID,omitempty"`
	MediaTagPrefix      string                `json:"mediaTagPrefix,omitempty"`
	MediaTagVersion     int                   `json:"mediaTagVersion,omitempty"`
	Thumb               string                `json:"thumb,omitempty"`
	Title1              string                `json:"title1,omitempty"`
	Title2              string                `json:"title2,omitempty"`
	Metadata            []PlexMetadataDetails `json:"Metadata,omitempty"`
}

// ============================================================================
// Helper Methods
// ============================================================================

// GetHardwareAccelerationName returns a human-readable name for hardware acceleration
func (t *PlexTranscodeSessionInfo) GetHardwareAccelerationName() string {
	if t.TranscodeHwDecodingTitle != "" {
		return t.TranscodeHwDecodingTitle
	}
	if t.TranscodeHwEncodingTitle != "" {
		return t.TranscodeHwEncodingTitle
	}
	hw := t.TranscodeHwDecoding
	if hw == "" {
		hw = t.TranscodeHwEncoding
	}
	switch hw {
	case "":
		return "Software"
	case "qsv":
		return "Intel QuickSync"
	case "nvenc", "nvdec":
		return "NVIDIA NVENC"
	case "vaapi":
		return "VAAPI"
	case "videotoolbox":
		return "VideoToolbox"
	case "mediacodec":
		return "MediaCodec"
	case "mf":
		return "MediaFoundation"
	default:
		return hw
	}
}

// HasPlexPass returns true if the server owner has a Plex Pass subscription
func (c *PlexServerCapabilitiesContainer) HasPlexPass() bool {
	return c.MyPlexSubscription
}

// SupportsHardwareTranscoding returns true if the server supports video transcoding
func (c *PlexServerCapabilitiesContainer) SupportsHardwareTranscoding() bool {
	return c.TranscoderVideo
}

// SupportsLiveTV returns true if the server supports Live TV/DVR
func (c *PlexServerCapabilitiesContainer) SupportsLiveTV() bool {
	return c.LiveTV
}

// SupportsSync returns true if the server supports sync/download
func (c *PlexServerCapabilitiesContainer) SupportsSync() bool {
	return c.Sync && c.AllowSync
}

// IsClaimedAndConnected returns true if the server is claimed and connected to Plex.tv
func (c *PlexServerCapabilitiesContainer) IsClaimedAndConnected() bool {
	return c.Claimed && c.MyPlex && c.MyPlexSigninState == "ok"
}

// GetActiveTranscodeSessions returns the number of active video transcode sessions
func (c *PlexServerCapabilitiesContainer) GetActiveTranscodeSessions() int {
	return c.TranscoderActiveVideoSessions
}
