// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package models

// ResolutionMismatch represents a single resolution downgrade pattern
type ResolutionMismatch struct {
	SourceResolution  string   `json:"source_resolution"`
	StreamResolution  string   `json:"stream_resolution"`
	PlaybackCount     int      `json:"playback_count"`
	Percentage        float64  `json:"percentage"`
	AffectedUsers     []string `json:"affected_users"`
	CommonPlatforms   []string `json:"common_platforms"`
	TranscodeRequired bool     `json:"transcode_required"`
}

// ResolutionMismatchAnalytics represents comprehensive resolution mismatch analytics
type ResolutionMismatchAnalytics struct {
	TotalPlaybacks      int                  `json:"total_playbacks"`
	MismatchedPlaybacks int                  `json:"mismatched_playbacks"`
	MismatchRate        float64              `json:"mismatch_rate_percent"`
	Mismatches          []ResolutionMismatch `json:"mismatches"`
	DirectPlayCount     int                  `json:"direct_play_count"`
	TranscodeCount      int                  `json:"transcode_count"`
	TopDowngradeUsers   []UserDowngradeStats `json:"top_downgrade_users"`
	MismatchByPlatform  []PlatformMismatch   `json:"mismatch_by_platform"`
}

// UserDowngradeStats represents downgrade statistics for a user
type UserDowngradeStats struct {
	Username       string  `json:"username"`
	TotalPlaybacks int     `json:"total_playbacks"`
	DowngradeCount int     `json:"downgrade_count"`
	DowngradeRate  float64 `json:"downgrade_rate_percent"`
	CommonReason   string  `json:"common_reason"`
}

// PlatformMismatch represents resolution mismatches by platform
type PlatformMismatch struct {
	Platform       string  `json:"platform"`
	MismatchCount  int     `json:"mismatch_count"`
	TotalPlaybacks int     `json:"total_playbacks"`
	MismatchRate   float64 `json:"mismatch_rate_percent"`
}

// HDRAnalytics represents HDR and dynamic range analytics
type HDRAnalytics struct {
	TotalPlaybacks     int                        `json:"total_playbacks"`
	FormatDistribution []DynamicRangeDistribution `json:"format_distribution"`
	HDRAdoptionRate    float64                    `json:"hdr_adoption_rate_percent"`
	ToneMappingEvents  []ToneMappingEvent         `json:"tone_mapping_events"`
	HDRCapableDevices  []HDRDeviceStats           `json:"hdr_capable_devices"`
	ContentByFormat    []ContentFormatStats       `json:"content_by_format"`
}

// DynamicRangeDistribution represents playback distribution by dynamic range
type DynamicRangeDistribution struct {
	DynamicRange  string  `json:"dynamic_range"`
	PlaybackCount int     `json:"playback_count"`
	Percentage    float64 `json:"percentage"`
	UniqueUsers   int     `json:"unique_users"`
}

// ToneMappingEvent represents HDR to SDR tone mapping occurrences
type ToneMappingEvent struct {
	SourceFormat    string   `json:"source_format"`
	StreamFormat    string   `json:"stream_format"`
	OccurrenceCount int      `json:"occurrence_count"`
	AffectedUsers   []string `json:"affected_users"`
	Platforms       []string `json:"platforms"`
}

// HDRDeviceStats represents HDR capability statistics by device
type HDRDeviceStats struct {
	Platform     string `json:"platform"`
	HDRPlaybacks int    `json:"hdr_playbacks"`
	SDRPlaybacks int    `json:"sdr_playbacks"`
	HDRCapable   bool   `json:"hdr_capable"`
}

// ContentFormatStats represents content statistics by dynamic range
type ContentFormatStats struct {
	DynamicRange     string  `json:"dynamic_range"`
	UniqueContent    int     `json:"unique_content"`
	AvgCompletion    float64 `json:"avg_completion"`
	MostWatchedTitle string  `json:"most_watched_title"`
}

// AudioAnalytics represents comprehensive audio quality analytics
type AudioAnalytics struct {
	TotalPlaybacks        int                        `json:"total_playbacks"`
	ChannelDistribution   []AudioChannelDistribution `json:"channel_distribution"`
	CodecDistribution     []AudioCodecDistribution   `json:"codec_distribution"`
	DownmixEvents         []AudioDownmixEvent        `json:"downmix_events"`
	SurroundSoundAdoption float64                    `json:"surround_sound_adoption_percent"`
	LosslessAdoption      float64                    `json:"lossless_adoption_percent"`
	AvgBitrate            float64                    `json:"avg_bitrate_kbps"`
	AtmosPlaybacks        int                        `json:"atmos_playbacks"`
}

// AudioChannelDistribution represents playback distribution by audio channels
type AudioChannelDistribution struct {
	Channels      string  `json:"channels"`
	Layout        string  `json:"layout,omitempty"`
	PlaybackCount int     `json:"playback_count"`
	Percentage    float64 `json:"percentage"`
	AvgCompletion float64 `json:"avg_completion"`
}

// AudioCodecDistribution represents playback distribution by audio codec
type AudioCodecDistribution struct {
	Codec         string  `json:"codec"`
	PlaybackCount int     `json:"playback_count"`
	Percentage    float64 `json:"percentage"`
	IsLossless    bool    `json:"is_lossless"`
	AvgBitrate    float64 `json:"avg_bitrate_kbps"`
}

// AudioDownmixEvent represents audio channel downmixing occurrences
type AudioDownmixEvent struct {
	SourceChannels  string   `json:"source_channels"`
	StreamChannels  string   `json:"stream_channels"`
	OccurrenceCount int      `json:"occurrence_count"`
	Platforms       []string `json:"platforms"`
	Users           []string `json:"users"`
}

// SubtitleAnalytics represents comprehensive subtitle usage analytics
type SubtitleAnalytics struct {
	TotalPlaybacks       int                      `json:"total_playbacks"`
	SubtitleUsageRate    float64                  `json:"subtitle_usage_rate_percent"`
	PlaybacksWithSubs    int                      `json:"playbacks_with_subtitles"`
	PlaybacksWithoutSubs int                      `json:"playbacks_without_subtitles"`
	LanguageDistribution []SubtitleLanguageStats  `json:"language_distribution"`
	CodecDistribution    []SubtitleCodecStats     `json:"codec_distribution"`
	UserPreferences      []UserSubtitlePreference `json:"user_preferences"`
}

// SubtitleLanguageStats represents subtitle usage by language
type SubtitleLanguageStats struct {
	Language      string  `json:"language"`
	PlaybackCount int     `json:"playback_count"`
	Percentage    float64 `json:"percentage"`
	UniqueUsers   int     `json:"unique_users"`
}

// SubtitleCodecStats represents subtitle usage by codec
type SubtitleCodecStats struct {
	Codec         string  `json:"codec"`
	PlaybackCount int     `json:"playback_count"`
	Percentage    float64 `json:"percentage"`
}

// UserSubtitlePreference represents subtitle preferences for a user
type UserSubtitlePreference struct {
	Username           string   `json:"username"`
	TotalPlaybacks     int      `json:"total_playbacks"`
	SubtitleUsageCount int      `json:"subtitle_usage_count"`
	SubtitleUsageRate  float64  `json:"subtitle_usage_rate_percent"`
	PreferredLanguages []string `json:"preferred_languages"`
}

// FrameRateAnalytics represents frame rate distribution and analysis
type FrameRateAnalytics struct {
	TotalPlaybacks        int                                `json:"total_playbacks"`
	FrameRateDistribution []FrameRateDistribution            `json:"frame_rate_distribution"`
	ByMediaType           map[string][]FrameRateDistribution `json:"by_media_type"`
	HighFrameRateAdoption float64                            `json:"high_frame_rate_adoption_percent"` // 60fps+
	ConversionEvents      []FrameRateConversion              `json:"conversion_events"`
}

// FrameRateDistribution represents playback distribution by frame rate
type FrameRateDistribution struct {
	FrameRate     string  `json:"frame_rate"`
	PlaybackCount int     `json:"playback_count"`
	Percentage    float64 `json:"percentage"`
	AvgCompletion float64 `json:"avg_completion"`
}

// FrameRateConversion represents frame rate conversion occurrences
type FrameRateConversion struct {
	SourceFPS       string   `json:"source_fps"`
	StreamFPS       string   `json:"stream_fps"`
	OccurrenceCount int      `json:"occurrence_count"`
	Platforms       []string `json:"platforms"`
}

// ContainerAnalytics represents container format distribution and compatibility
type ContainerAnalytics struct {
	TotalPlaybacks        int                     `json:"total_playbacks"`
	FormatDistribution    []ContainerDistribution `json:"format_distribution"`
	DirectPlayRates       map[string]float64      `json:"direct_play_rates"`
	RemuxEvents           []ContainerRemux        `json:"remux_events"`
	PlatformCompatibility []PlatformContainer     `json:"platform_compatibility"`
}

// ContainerDistribution represents playback distribution by container format
type ContainerDistribution struct {
	Container      string  `json:"container"`
	PlaybackCount  int     `json:"playback_count"`
	Percentage     float64 `json:"percentage"`
	DirectPlayRate float64 `json:"direct_play_rate_percent"`
}

// ContainerRemux represents container remuxing occurrences
type ContainerRemux struct {
	SourceContainer string   `json:"source_container"`
	StreamContainer string   `json:"stream_container"`
	OccurrenceCount int      `json:"occurrence_count"`
	Platforms       []string `json:"platforms"`
}

// PlatformContainer represents container compatibility by platform
type PlatformContainer struct {
	Platform          string   `json:"platform"`
	SupportedFormats  []string `json:"supported_formats"`
	TranscodeRequired []string `json:"transcode_required"`
	DirectPlayRate    float64  `json:"direct_play_rate_percent"`
}
