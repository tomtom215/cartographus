// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package models

// PlatformStats represents playback statistics by platform
type PlatformStats struct {
	Platform      string `json:"platform"`
	PlaybackCount int    `json:"playback_count"`
	UniqueUsers   int    `json:"unique_users"`
}

// PlayerStats represents playback statistics by player
type PlayerStats struct {
	Player        string `json:"player"`
	PlaybackCount int    `json:"playback_count"`
	UniqueUsers   int    `json:"unique_users"`
}

// CompletionBucket represents a completion rate bucket
type CompletionBucket struct {
	Bucket        string  `json:"bucket"`
	MinPercent    int     `json:"min_percent"`
	MaxPercent    int     `json:"max_percent"`
	PlaybackCount int     `json:"playback_count"`
	AvgCompletion float64 `json:"avg_completion"`
}

// ContentCompletionStats represents content completion analytics
type ContentCompletionStats struct {
	Buckets         []CompletionBucket `json:"buckets"`
	TotalPlaybacks  int                `json:"total_playbacks"`
	AvgCompletion   float64            `json:"avg_completion"`
	FullyWatched    int                `json:"fully_watched"`
	FullyWatchedPct float64            `json:"fully_watched_pct"`
}

// TranscodeStats represents playback statistics by transcode decision
type TranscodeStats struct {
	TranscodeDecision string  `json:"transcode_decision"`
	PlaybackCount     int     `json:"playback_count"`
	Percentage        float64 `json:"percentage"`
}

// ResolutionStats represents playback statistics by video resolution
type ResolutionStats struct {
	VideoResolution string  `json:"video_resolution"`
	PlaybackCount   int     `json:"playback_count"`
	Percentage      float64 `json:"percentage"`
}

// CodecStats represents playback statistics by codec combination
type CodecStats struct {
	VideoCodec    string  `json:"video_codec"`
	AudioCodec    string  `json:"audio_codec"`
	PlaybackCount int     `json:"playback_count"`
	Percentage    float64 `json:"percentage"`
}

// LibraryStats represents playback statistics by library
type LibraryStats struct {
	SectionID     int     `json:"section_id"`
	LibraryName   string  `json:"library_name"`
	PlaybackCount int     `json:"playback_count"`
	UniqueUsers   int     `json:"unique_users"`
	TotalDuration int     `json:"total_duration_minutes"`
	AvgCompletion float64 `json:"avg_completion"`
}

// RatingStats represents playback statistics by content rating
type RatingStats struct {
	ContentRating string  `json:"content_rating"`
	PlaybackCount int     `json:"playback_count"`
	Percentage    float64 `json:"percentage"`
}

// DurationStats represents watch duration analytics
type DurationStats struct {
	AvgDuration      int                   `json:"avg_duration_minutes"`
	MedianDuration   int                   `json:"median_duration_minutes"`
	TotalDuration    int                   `json:"total_duration_minutes"`
	AvgCompletion    float64               `json:"avg_completion_percent"`
	FullyWatched     int                   `json:"fully_watched_count"`
	FullyWatchedPct  float64               `json:"fully_watched_percent"`
	PartiallyWatched int                   `json:"partially_watched_count"`
	DurationByType   []DurationByMediaType `json:"duration_by_media_type"`
}

// DurationByMediaType represents duration breakdown by media type
type DurationByMediaType struct {
	MediaType     string  `json:"media_type"`
	AvgDuration   int     `json:"avg_duration_minutes"`
	TotalDuration int     `json:"total_duration_minutes"`
	PlaybackCount int     `json:"playback_count"`
	AvgCompletion float64 `json:"avg_completion"`
}

// YearStats represents playback statistics by release year
type YearStats struct {
	Year          int `json:"year"`
	PlaybackCount int `json:"playback_count"`
}
