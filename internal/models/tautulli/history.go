// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package tautulli

// TautulliHistory represents the API response from Tautulli's get_history endpoint
type TautulliHistory struct {
	Response TautulliHistoryResponse `json:"response"`
}

type TautulliHistoryResponse struct {
	Result  string              `json:"result"`
	Message *string             `json:"message,omitempty"`
	Data    TautulliHistoryData `json:"data"`
}

type TautulliHistoryData struct {
	RecordsFiltered int                     `json:"recordsFiltered"`
	RecordsTotal    int                     `json:"recordsTotal"`
	Data            []TautulliHistoryRecord `json:"data"`
}

// TautulliHistoryRecord represents a single playback history record from Tautulli's get_history endpoint.
// This struct captures completed session data including playback metrics, quality info, and transcode status.
//
// Field Reference (Tautulli API v2):
//   - User fields: session_key, user, user_id, ip_address, ip_address_public
//   - Media fields: rating_key, parent_rating_key, grandparent_rating_key, title hierarchy
//   - Quality fields: video_codec, audio_codec, resolution, bitrate, HDR metadata
//   - Transcode fields: transcode_decision, video_decision, audio_decision, HW acceleration status
//   - Client fields: platform, player, machine_id, device, product
//   - Playback metrics: percent_complete, paused_counter, duration
//
// Note: Duration is in SECONDS (unlike get_activity which returns milliseconds)
type TautulliHistoryRecord struct {
	// Session identification
	// Pointer type allows distinguishing null from zero value in Tautulli API responses
	SessionKey *string `json:"session_key"` // Nullable - null when session ended
	Date       int64   `json:"date"`
	Started    int64   `json:"started"`
	Stopped    int64   `json:"stopped"`

	// Grouped playback fields (v1.46 - API Coverage Expansion)
	// Pointer type allows distinguishing null from zero value in Tautulli API responses
	GroupCount *int    `json:"group_count"` // Number of grouped entries (nullable when no grouping)
	GroupIDs   string  `json:"group_ids"`   // Comma-separated grouped IDs
	State      *string `json:"state"`       // Playback state at end of session (null for stopped)

	// User information
	// Pointer type allows distinguishing null from zero value in Tautulli API responses
	UserID          *int   `json:"user_id"` // Nullable - may be null in edge cases
	User            string `json:"user"`
	FriendlyName    string `json:"friendly_name"`
	UserThumb       string `json:"user_thumb"`
	Email           string `json:"email"`
	IPAddress       string `json:"ip_address"`
	IPAddressPublic string `json:"ip_address_public"` // Public IP for accurate geolocation

	// User permission/status fields (v1.44 - API Coverage Expansion)
	// Pointer type allows distinguishing null from zero value in Tautulli API responses
	IsAdmin      *int `json:"is_admin"`      // 0 or 1 - Admin user flag (nullable)
	IsHomeUser   *int `json:"is_home_user"`  // 0 or 1 - Home user flag (nullable)
	IsAllowSync  *int `json:"is_allow_sync"` // 0 or 1 - Sync permission flag (nullable)
	IsRestricted *int `json:"is_restricted"` // 0 or 1 - Restricted user flag (nullable)
	KeepHistory  *int `json:"keep_history"`  // 0 or 1 - History retention flag (nullable)
	DeletedUser  *int `json:"deleted_user"`  // 0 or 1 - Deleted user flag (nullable)
	DoNotify     *int `json:"do_notify"`     // 0 or 1 - Notification flag (nullable)
	AllowGuest   *int `json:"allow_guest"`   // 0 or 1 - Allow guest access flag (nullable)

	// User library access fields (v1.45 - API Coverage Expansion)
	SharedLibraries string `json:"shared_libraries"` // Semicolon-separated library IDs user can access

	// Media identification
	// Pointer type allows distinguishing null from zero value in Tautulli API responses
	MediaType        string  `json:"media_type"`
	Title            string  `json:"title"`
	ParentTitle      *string `json:"parent_title"`      // Nullable - null for movies
	GrandparentTitle *string `json:"grandparent_title"` // Nullable - null for movies
	SortTitle        string  `json:"sort_title"`        // Sort title for ordering (v1.45 - API Coverage Expansion)

	// Client/Player information
	Platform         string `json:"platform"`
	PlatformName     string `json:"platform_name"`
	PlatformVersion  string `json:"platform_version"`
	Player           string `json:"player"`
	Product          string `json:"product"`
	ProductVersion   string `json:"product_version"`
	Device           string `json:"device"`
	MachineID        string `json:"machine_id"` // Unique device identifier
	Location         string `json:"location"`
	QualityProfile   string `json:"quality_profile"`
	OptimizedVersion *int   `json:"optimized_version"` // Nullable - null if not optimized
	SyncedVersion    *int   `json:"synced_version"`    // Nullable - null if not synced

	// Playback metrics
	// Pointer type allows distinguishing null from zero value in Tautulli API responses
	PercentComplete *int `json:"percent_complete"` // Nullable
	PausedCounter   *int `json:"paused_counter"`   // Nullable
	Duration        *int `json:"duration"`         // In SECONDS (NOT milliseconds like get_activity), nullable for live
	Throttled       *int `json:"throttled"`        // 0 or 1 - Playback throttled status (nullable)

	// Live TV fields (v1.45 - API Coverage Expansion)
	Live              *int   `json:"live"`               // 0 or 1 - Live TV session flag (nullable)
	LiveUUID          string `json:"live_uuid"`          // Live TV session UUID
	ChannelStream     *int   `json:"channel_stream"`     // Live TV channel stream number (nullable for non-live)
	ChannelCallSign   string `json:"channel_call_sign"`  // Live TV channel call sign (e.g., "NBC")
	ChannelIdentifier string `json:"channel_identifier"` // Live TV channel identifier

	// Transcode decision fields
	TranscodeDecision string `json:"transcode_decision"`
	VideoDecision     string `json:"video_decision"`
	AudioDecision     string `json:"audio_decision"`
	SubtitleDecision  string `json:"subtitle_decision"`

	// Hardware transcode fields (CRITICAL for GPU utilization monitoring)
	TranscodeKey            string `json:"transcode_key"`
	TranscodeThrottled      *int   `json:"transcode_throttled"`        // Nullable - null when not transcoding
	TranscodeProgress       *int   `json:"transcode_progress"`         // 0-100 (nullable - null when not transcoding)
	TranscodeSpeed          string `json:"transcode_speed"`            // Speed multiplier (e.g., "2.5")
	TranscodeHWRequested    *int   `json:"transcode_hw_requested"`     // Nullable - null when not transcoding
	TranscodeHWDecoding     *int   `json:"transcode_hw_decoding"`      // Nullable - null when not transcoding
	TranscodeHWEncoding     *int   `json:"transcode_hw_encoding"`      // Nullable - null when not transcoding
	TranscodeHWFullPipeline *int   `json:"transcode_hw_full_pipeline"` // Nullable - null when not transcoding
	TranscodeHWDecode       string `json:"transcode_hw_decode"`        // HW decode codec used
	TranscodeHWDecodeTitle  string `json:"transcode_hw_decode_title"`  // HW decoder name (e.g., "Intel Quick Sync")
	TranscodeHWEncode       string `json:"transcode_hw_encode"`        // HW encode codec used
	TranscodeHWEncodeTitle  string `json:"transcode_hw_encode_title"`  // HW encoder name (e.g., "NVIDIA NVENC")
	TranscodeContainer      string `json:"transcode_container"`
	TranscodeVideoCodec     string `json:"transcode_video_codec"`
	TranscodeVideoWidth     *int   `json:"transcode_video_width"`  // Transcoded video width (nullable - null when not transcoding)
	TranscodeVideoHeight    *int   `json:"transcode_video_height"` // Transcoded video height (nullable - null when not transcoding)
	TranscodeAudioCodec     string `json:"transcode_audio_codec"`
	TranscodeAudioChannels  *int   `json:"transcode_audio_channels"` // Nullable - null when not transcoding

	// Source video fields
	VideoResolution     string `json:"video_resolution"`
	VideoFullResolution string `json:"video_full_resolution"` // Full resolution string (e.g., "1080p")
	VideoCodec          string `json:"video_codec"`
	VideoCodecLevel     string `json:"video_codec_level"`
	VideoProfile        string `json:"video_profile"`
	VideoScanType       string `json:"video_scan_type"`
	VideoLanguage       string `json:"video_language"`
	VideoLanguageCode   string `json:"video_language_code"` // 3-letter ISO code (eng, jpn, etc.)
	AspectRatio         string `json:"aspect_ratio"`

	// HDR/Color metadata (CRITICAL for HDR10/HDR10+/Dolby Vision detection)
	VideoDynamicRange      string `json:"video_dynamic_range"` // SDR, HDR, HDR10, HDR10+, DV
	VideoColorPrimaries    string `json:"video_color_primaries"`
	VideoColorRange        string `json:"video_color_range"`
	VideoColorSpace        string `json:"video_color_space"`
	VideoColorTrc          string `json:"video_color_trc"`
	VideoChromaSubsampling string `json:"video_chroma_subsampling"`

	// Source audio fields
	AudioCodec        string `json:"audio_codec"`
	AudioProfile      string `json:"audio_profile"`
	AudioLanguage     string `json:"audio_language"`
	AudioLanguageCode string `json:"audio_language_code"` // 3-letter ISO code (eng, jpn, etc.)

	// Library metadata
	// Pointer type allows distinguishing null from zero value in Tautulli API responses
	SectionID     *int   `json:"section_id"` // Nullable - may be null if library info unavailable
	LibraryName   string `json:"library_name"`
	ContentRating string `json:"content_rating"`
	Year          *int   `json:"year"`   // Nullable - null for media without year data
	Studio        string `json:"studio"` // Production studio

	// Media metadata fields (v1.44 - API Coverage Expansion)
	AddedAt        string `json:"added_at"`        // Unix timestamp when added to library
	UpdatedAt      string `json:"updated_at"`      // Unix timestamp of last metadata update
	LastViewedAt   string `json:"last_viewed_at"`  // Unix timestamp of last watch
	Summary        string `json:"summary"`         // Media description/synopsis
	Tagline        string `json:"tagline"`         // Movie tagline
	Rating         string `json:"rating"`          // Critic rating (0.0-10.0)
	AudienceRating string `json:"audience_rating"` // Audience rating (0.0-10.0)
	UserRating     string `json:"user_rating"`     // User's personal rating (0.0-10.0)
	Labels         string `json:"labels"`          // Labels (comma-separated)
	Collections    string `json:"collections"`     // Collections (comma-separated)
	Banner         string `json:"banner"`          // Banner image URL

	// Metadata enrichment fields (CRITICAL for analytics)
	// Pointer type allows distinguishing null from zero value in Tautulli API responses
	RatingKey             *int     `json:"rating_key"`              // Numeric ID (Tautulli returns as integer, can be null)
	ParentRatingKey       *int     `json:"parent_rating_key"`       // Numeric ID (nil when null)
	GrandparentRatingKey  *int     `json:"grandparent_rating_key"`  // Numeric ID (nil when null)
	MediaIndex            *int     `json:"media_index"`             // Episode number (nil for movies)
	ParentMediaIndex      *int     `json:"parent_media_index"`      // Season number (nil for movies)
	GUID                  string   `json:"guid"`                    // External IDs (IMDB, TVDB, etc.)
	OriginalTitle         *string  `json:"original_title"`          // Original non-localized title (nullable)
	FullTitle             string   `json:"full_title"`              // Formatted full title
	OriginallyAvailableAt *string  `json:"originally_available_at"` // Release date (nullable)
	WatchedStatus         *float64 `json:"watched_status"`          // Watch progress 0.0-1.0 (nullable)
	RowID                 *int     `json:"row_id"`                  // Tautulli database row ID (nullable)
	ReferenceID           *int     `json:"reference_id"`            // Reference to parent record (often null)
	Thumb                 string   `json:"thumb"`                   // Thumbnail URL

	// Cast and crew (comma-separated strings from Tautulli)
	Directors string `json:"directors"` // Director names, comma-separated
	Writers   string `json:"writers"`   // Writer names, comma-separated
	Actors    string `json:"actors"`    // Actor names, comma-separated
	Genres    string `json:"genres"`    // Genres, comma-separated

	// Stream output fields (transcoded output quality)
	// Pointer type allows distinguishing null from zero value in Tautulli API responses
	StreamContainer           string `json:"stream_container"`
	StreamBitrate             *int   `json:"stream_bitrate"` // Nullable - null when no stream data
	StreamVideoCodec          string `json:"stream_video_codec"`
	StreamVideoCodecLevel     string `json:"stream_video_codec_level"`
	StreamVideoResolution     string `json:"stream_video_resolution"`
	StreamVideoFullResolution string `json:"stream_video_full_resolution"` // e.g., "1080p"
	StreamVideoDecision       string `json:"stream_video_decision"`
	StreamVideoBitrate        *int   `json:"stream_video_bitrate"`   // Nullable - null when no stream data
	StreamVideoBitDepth       *int   `json:"stream_video_bit_depth"` // Nullable - null when no stream data
	StreamVideoWidth          *int   `json:"stream_video_width"`     // Nullable - null when no stream data
	StreamVideoHeight         *int   `json:"stream_video_height"`    // Nullable - null when no stream data
	StreamVideoFramerate      string `json:"stream_video_framerate"`
	StreamVideoProfile        string `json:"stream_video_profile"`
	StreamVideoScanType       string `json:"stream_video_scan_type"`
	StreamVideoLanguage       string `json:"stream_video_language"`
	StreamVideoLanguageCode   string `json:"stream_video_language_code"` // 3-letter ISO code
	StreamVideoDynamicRange   string `json:"stream_video_dynamic_range"`
	StreamAspectRatio         string `json:"stream_aspect_ratio"` // Output stream aspect ratio (v1.46 - API Coverage Expansion)
	StreamAudioCodec          string `json:"stream_audio_codec"`
	StreamAudioChannels       string `json:"stream_audio_channels"`
	StreamAudioChannelLayout  string `json:"stream_audio_channel_layout"`
	StreamAudioBitrate        *int   `json:"stream_audio_bitrate"`      // Nullable - null when no stream data
	StreamAudioBitrateMode    string `json:"stream_audio_bitrate_mode"` // cbr, vbr
	StreamAudioSampleRate     *int   `json:"stream_audio_sample_rate"`  // Hz (nullable)
	StreamAudioLanguage       string `json:"stream_audio_language"`
	StreamAudioLanguageCode   string `json:"stream_audio_language_code"` // 3-letter ISO code
	StreamAudioProfile        string `json:"stream_audio_profile"`
	StreamAudioDecision       string `json:"stream_audio_decision"`

	// Stream subtitle output fields (v1.44 - API Coverage Expansion)
	StreamSubtitleCodec        string `json:"stream_subtitle_codec"`
	StreamSubtitleContainer    string `json:"stream_subtitle_container"`
	StreamSubtitleFormat       string `json:"stream_subtitle_format"`
	StreamSubtitleForced       *int   `json:"stream_subtitle_forced"` // Nullable - null when no subtitles
	StreamSubtitleLocation     string `json:"stream_subtitle_location"`
	StreamSubtitleLanguage     string `json:"stream_subtitle_language"`
	StreamSubtitleLanguageCode string `json:"stream_subtitle_language_code"` // 3-letter ISO code
	StreamSubtitleDecision     string `json:"stream_subtitle_decision"`
	SubtitleContainerOld       string `json:"subtitle_container"` // Legacy field kept for compatibility

	// Source audio details
	// Pointer type allows distinguishing null from zero value in Tautulli API responses
	AudioChannels      string `json:"audio_channels"`
	AudioChannelLayout string `json:"audio_channel_layout"`
	AudioBitrate       *int   `json:"audio_bitrate"` // Nullable - null when metadata incomplete
	AudioBitrateMode   string `json:"audio_bitrate_mode"`
	AudioSampleRate    *int   `json:"audio_sample_rate"` // Nullable - null when metadata incomplete

	// Source video details
	// Pointer type allows distinguishing null from zero value in Tautulli API responses
	VideoFrameRate string `json:"video_framerate"`
	VideoBitrate   *int   `json:"video_bitrate"`    // Nullable - null when metadata incomplete
	VideoBitDepth  *int   `json:"video_bit_depth"`  // Nullable - null when metadata incomplete
	VideoRefFrames *int   `json:"video_ref_frames"` // Nullable - null when metadata incomplete
	VideoWidth     *int   `json:"video_width"`      // Nullable - null when metadata incomplete
	VideoHeight    *int   `json:"video_height"`     // Nullable - null when metadata incomplete

	// Container and subtitle
	// Pointer type allows distinguishing null from zero value in Tautulli API responses
	Container            string `json:"container"`
	SubtitleCodec        string `json:"subtitle_codec"`
	SubtitleContainer    string `json:"subtitle_container_fmt"` // Container format (using different tag to avoid conflict)
	SubtitleFormat       string `json:"subtitle_format"`        // Subtitle format (srt, ass, pgs)
	SubtitleLanguage     string `json:"subtitle_language"`
	SubtitleLanguageCode string `json:"subtitle_language_code"` // 3-letter ISO code (eng, spa, etc.)
	SubtitleForced       *int   `json:"subtitle_forced"`        // Nullable - null when no subtitle data
	SubtitleLocation     string `json:"subtitle_location"`      // embedded, external, agent
	Subtitles            *int   `json:"subtitles"`              // 0 or 1 (nullable)

	// Connection security and network
	// Pointer type allows distinguishing null from zero value in Tautulli API responses
	Secure    *int `json:"secure"`    // 0 or 1 - Secure connection flag (nullable)
	Relayed   *int `json:"relayed"`   // 0 or 1 - Relayed connection flag (nullable)
	Relay     *int `json:"relay"`     // 0 or 1 - Using relay flag (nullable)
	Local     *int `json:"local"`     // 0 or 1 - Local connection flag (nullable)
	Bandwidth *int `json:"bandwidth"` // Nullable - null if not measured

	// File metadata
	// Pointer type allows distinguishing null from zero value in Tautulli API responses
	FileSize *int64 `json:"file_size"` // Nullable - null when file data unavailable
	Bitrate  *int   `json:"bitrate"`   // Nullable - null when metadata incomplete
	File     string `json:"file"`      // Full file path

	// Thumbnails and art
	ParentThumb      string `json:"parent_thumb"`
	GrandparentThumb string `json:"grandparent_thumb"`
	Art              string `json:"art"`
	GrandparentArt   string `json:"grandparent_art"`

	// GUID for external ID matching
	ParentGUID      string `json:"parent_guid"`
	GrandparentGUID string `json:"grandparent_guid"`
}
