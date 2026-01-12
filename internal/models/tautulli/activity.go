// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package tautulli

// TautulliActivity represents the API response from Tautulli's get_activity endpoint
type TautulliActivity struct {
	Response TautulliActivityResponse `json:"response"`
}

type TautulliActivityResponse struct {
	Result  string               `json:"result"`
	Message *string              `json:"message,omitempty"`
	Data    TautulliActivityData `json:"data"`
}

type TautulliActivityData struct {
	LANBandwidth            int                       `json:"lan_bandwidth"`
	WANBandwidth            int                       `json:"wan_bandwidth"`
	TotalBandwidth          int                       `json:"total_bandwidth"`
	StreamCount             int                       `json:"stream_count"`
	StreamCountDirectPlay   int                       `json:"stream_count_direct_play"`
	StreamCountDirectStream int                       `json:"stream_count_direct_stream"`
	StreamCountTranscode    int                       `json:"stream_count_transcode"`
	Sessions                []TautulliActivitySession `json:"sessions"`
}

// TautulliActivitySession represents a single active streaming session from Tautulli's get_activity endpoint.
// This struct captures real-time session data including playback state, quality metrics, and transcode status.
//
// Field Reference (Tautulli API v2):
//   - User fields: session_key, session_id, user, user_id, friendly_name, user_thumb
//   - Media fields: rating_key, parent_rating_key, grandparent_rating_key, title hierarchy
//   - Quality fields: video_codec, audio_codec, resolution, bitrate, HDR metadata
//   - Transcode fields: transcode_decision, video_decision, audio_decision, HW acceleration status
//   - Client fields: ip_address, ip_address_public, platform, player, machine_id
//   - Playback state: state, view_offset, duration, progress_percent, live status
type TautulliActivitySession struct {
	// Session identification
	SessionKey string `json:"session_key"`
	SessionID  string `json:"session_id"`

	// Media identification
	MediaType            string `json:"media_type"`
	RatingKey            string `json:"rating_key"`
	ParentRatingKey      string `json:"parent_rating_key"`
	GrandparentRatingKey string `json:"grandparent_rating_key"`
	Title                string `json:"title"`
	ParentTitle          string `json:"parent_title"`
	GrandparentTitle     string `json:"grandparent_title"`
	FullTitle            string `json:"full_title"`
	OriginalTitle        string `json:"original_title"`
	SortTitle            string `json:"sort_title"`
	MediaIndex           string `json:"media_index"`        // Episode number
	ParentMediaIndex     string `json:"parent_media_index"` // Season number
	Year                 int    `json:"year"`

	// Thumbnails and art
	Thumb            string `json:"thumb"`
	ParentThumb      string `json:"parent_thumb"`
	GrandparentThumb string `json:"grandparent_thumb"`
	Art              string `json:"art"`
	GrandparentArt   string `json:"grandparent_art"`

	// User information
	User         string `json:"user"`
	UserID       int    `json:"user_id"`
	FriendlyName string `json:"friendly_name"`
	UserThumb    string `json:"user_thumb"`
	Email        string `json:"email"`

	// User permission/status fields (v1.44 - API Coverage Expansion)
	IsAdmin      int `json:"is_admin"`      // 0 or 1 - Admin user flag
	IsHomeUser   int `json:"is_home_user"`  // 0 or 1 - Home user flag
	IsAllowSync  int `json:"is_allow_sync"` // 0 or 1 - Sync permission flag
	IsRestricted int `json:"is_restricted"` // 0 or 1 - Restricted user flag
	KeepHistory  int `json:"keep_history"`  // 0 or 1 - History retention flag
	DeletedUser  int `json:"deleted_user"`  // 0 or 1 - Deleted user flag
	DoNotify     int `json:"do_notify"`     // 0 or 1 - Notification flag

	// Client/Player information
	IPAddress        string `json:"ip_address"`
	IPAddressPublic  string `json:"ip_address_public"` // Public IP for accurate geolocation
	Player           string `json:"player"`
	Platform         string `json:"platform"`
	PlatformName     string `json:"platform_name"`
	PlatformVersion  string `json:"platform_version"`
	Product          string `json:"product"`
	ProductVersion   string `json:"product_version"`
	Device           string `json:"device"`
	MachineID        string `json:"machine_id"` // Unique device identifier
	Local            int    `json:"local"`      // 0 or 1
	QualityProfile   string `json:"quality_profile"`
	OptimizedVersion int    `json:"optimized_version"`
	SyncedVersion    int    `json:"synced_version"`

	// Playback state
	State           string `json:"state"` // playing, paused, buffering
	ViewOffset      int    `json:"view_offset"`
	Duration        int    `json:"duration"`
	ProgressPercent int    `json:"progress_percent"`
	Throttled       int    `json:"throttled"`

	// Live TV fields
	Live              int    `json:"live"`
	LiveUUID          string `json:"live_uuid"`
	ChannelStream     int    `json:"channel_stream"`
	ChannelCallSign   string `json:"channel_call_sign"`
	ChannelIdentifier string `json:"channel_identifier"`

	// Transcode decision fields
	TranscodeDecision string `json:"transcode_decision"`
	VideoDecision     string `json:"video_decision"`
	AudioDecision     string `json:"audio_decision"`
	SubtitleDecision  string `json:"subtitle_decision"`

	// Hardware transcode fields (CRITICAL for GPU utilization monitoring)
	TranscodeKey            string `json:"transcode_key"`
	TranscodeThrottled      int    `json:"transcode_throttled"`
	TranscodeProgress       int    `json:"transcode_progress"` // 0-100
	TranscodeSpeed          string `json:"transcode_speed"`    // Speed multiplier (e.g., "2.5")
	TranscodeHWRequested    int    `json:"transcode_hw_requested"`
	TranscodeHWDecoding     int    `json:"transcode_hw_decoding"`
	TranscodeHWEncoding     int    `json:"transcode_hw_encoding"`
	TranscodeHWFullPipeline int    `json:"transcode_hw_full_pipeline"`
	TranscodeHWDecode       string `json:"transcode_hw_decode"`       // HW decode codec used
	TranscodeHWDecodeTitle  string `json:"transcode_hw_decode_title"` // HW decoder name (e.g., "Intel Quick Sync")
	TranscodeHWEncode       string `json:"transcode_hw_encode"`       // HW encode codec used
	TranscodeHWEncodeTitle  string `json:"transcode_hw_encode_title"` // HW encoder name (e.g., "NVIDIA NVENC")
	TranscodeContainer      string `json:"transcode_container"`
	TranscodeVideoCodec     string `json:"transcode_video_codec"`
	TranscodeAudioCodec     string `json:"transcode_audio_codec"`
	TranscodeAudioChannels  int    `json:"transcode_audio_channels"`

	// Source video fields
	Width               int    `json:"width"`
	Height              int    `json:"height"`
	Container           string `json:"container"`
	VideoCodec          string `json:"video_codec"`
	VideoCodecLevel     string `json:"video_codec_level"`
	VideoBitrate        int    `json:"video_bitrate"`
	VideoBitDepth       int    `json:"video_bit_depth"`
	VideoFramerate      string `json:"video_framerate"`
	VideoRefFrames      int    `json:"video_ref_frames"`
	VideoResolution     string `json:"video_resolution"`
	VideoFullResolution string `json:"video_full_resolution"` // Full resolution string (e.g., "1080p")
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
	AudioCodec         string `json:"audio_codec"`
	AudioBitrate       int    `json:"audio_bitrate"`
	AudioBitrateMode   string `json:"audio_bitrate_mode"`
	AudioChannels      int    `json:"audio_channels"`
	AudioChannelLayout string `json:"audio_channel_layout"`
	AudioSampleRate    int    `json:"audio_sample_rate"`
	AudioLanguage      string `json:"audio_language"`
	AudioLanguageCode  string `json:"audio_language_code"` // 3-letter ISO code (eng, jpn, etc.)
	AudioProfile       string `json:"audio_profile"`

	// Subtitle fields
	SubtitleCodec        string `json:"subtitle_codec"`
	SubtitleContainer    string `json:"subtitle_container"` // Container format
	SubtitleFormat       string `json:"subtitle_format"`    // Subtitle format (srt, ass, pgs)
	SubtitleLanguage     string `json:"subtitle_language"`
	SubtitleLanguageCode string `json:"subtitle_language_code"` // 3-letter ISO code (eng, spa, etc.)
	SubtitleForced       int    `json:"subtitle_forced"`
	SubtitleLocation     string `json:"subtitle_location"` // embedded, external, agent
	Subtitles            int    `json:"subtitles"`         // 0 or 1

	// Stream output fields (transcoded output quality)
	StreamContainer           string `json:"stream_container"`
	StreamBitrate             int    `json:"stream_bitrate"`
	StreamVideoCodec          string `json:"stream_video_codec"`
	StreamVideoCodecLevel     string `json:"stream_video_codec_level"`
	StreamVideoResolution     string `json:"stream_video_resolution"`
	StreamVideoFullResolution string `json:"stream_video_full_resolution"` // e.g., "1080p"
	StreamVideoDecision       string `json:"stream_video_decision"`
	StreamVideoBitrate        int    `json:"stream_video_bitrate"`
	StreamVideoBitDepth       int    `json:"stream_video_bit_depth"`
	StreamVideoWidth          int    `json:"stream_video_width"`
	StreamVideoHeight         int    `json:"stream_video_height"`
	StreamVideoFramerate      string `json:"stream_video_framerate"`
	StreamVideoProfile        string `json:"stream_video_profile"`
	StreamVideoScanType       string `json:"stream_video_scan_type"`
	StreamVideoLanguage       string `json:"stream_video_language"`
	StreamVideoLanguageCode   string `json:"stream_video_language_code"` // 3-letter ISO code
	StreamVideoDynamicRange   string `json:"stream_video_dynamic_range"`
	StreamAudioCodec          string `json:"stream_audio_codec"`
	StreamAudioChannels       int    `json:"stream_audio_channels"`
	StreamAudioChannelLayout  string `json:"stream_audio_channel_layout"`
	StreamAudioBitrate        int    `json:"stream_audio_bitrate"`
	StreamAudioBitrateMode    string `json:"stream_audio_bitrate_mode"` // cbr, vbr
	StreamAudioSampleRate     int    `json:"stream_audio_sample_rate"`  // Hz
	StreamAudioLanguage       string `json:"stream_audio_language"`
	StreamAudioLanguageCode   string `json:"stream_audio_language_code"` // 3-letter ISO code
	StreamAudioProfile        string `json:"stream_audio_profile"`
	StreamAudioDecision       string `json:"stream_audio_decision"`

	// Stream subtitle output fields (v1.44 - API Coverage Expansion)
	StreamSubtitleCodec        string `json:"stream_subtitle_codec"`
	StreamSubtitleContainer    string `json:"stream_subtitle_container"`
	StreamSubtitleFormat       string `json:"stream_subtitle_format"`
	StreamSubtitleForced       int    `json:"stream_subtitle_forced"`
	StreamSubtitleLocation     string `json:"stream_subtitle_location"`
	StreamSubtitleLanguage     string `json:"stream_subtitle_language"`
	StreamSubtitleLanguageCode string `json:"stream_subtitle_language_code"` // 3-letter ISO code
	StreamSubtitleDecision     string `json:"stream_subtitle_decision"`

	// Overall quality metrics
	Bitrate   int    `json:"bitrate"`
	Bandwidth int    `json:"bandwidth"`
	Location  string `json:"location"`
	Secure    int    `json:"secure"`
	Relayed   int    `json:"relayed"`
	Relay     int    `json:"relay"`

	// Library metadata
	SectionID     string `json:"section_id"`
	LibraryName   string `json:"library_name"`
	ContentRating string `json:"content_rating"` // Age rating (PG, R, TV-MA)
	Studio        string `json:"studio"`         // Production studio

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

	// File information
	File     string `json:"file"`
	FileSize int64  `json:"file_size"`

	// GUID for external ID matching
	Guid            string `json:"guid"`
	ParentGuid      string `json:"parent_guid"`
	GrandparentGuid string `json:"grandparent_guid"`
}
