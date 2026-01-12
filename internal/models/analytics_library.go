// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package models

// LibraryAnalytics represents comprehensive library-specific analytics
type LibraryAnalytics struct {
	LibraryID           int                  `json:"library_id"`
	LibraryName         string               `json:"library_name"`
	MediaType           string               `json:"media_type"`
	TotalItems          int                  `json:"total_items"`
	WatchedItems        int                  `json:"watched_items"`
	UnwatchedItems      int                  `json:"unwatched_items"`
	WatchedPercentage   float64              `json:"watched_percentage"`
	TotalPlaybacks      int                  `json:"total_playbacks"`
	UniqueUsers         int                  `json:"unique_users"`
	TotalWatchTime      int                  `json:"total_watch_time_minutes"`
	AvgCompletion       float64              `json:"avg_completion"`
	MostWatchedItem     string               `json:"most_watched_item"`
	TopUsers            []LibraryUserStats   `json:"top_users"`
	PlaysByDay          []PlaybackTrend      `json:"plays_by_day"`
	QualityDistribution LibraryQualityStats  `json:"quality_distribution"`
	ContentHealth       LibraryHealthMetrics `json:"content_health"`
}

// LibraryUserStats represents user activity within a library
type LibraryUserStats struct {
	Username   string  `json:"username"`
	Plays      int     `json:"plays"`
	WatchTime  int     `json:"watch_time_minutes"`
	Completion float64 `json:"avg_completion"`
}

// LibraryQualityStats represents quality metrics for a library
type LibraryQualityStats struct {
	HDRContent      int     `json:"hdr_content_count"`
	FourKContent    int     `json:"4k_content_count"`
	SurroundContent int     `json:"surround_sound_count"`
	AvgBitrate      float64 `json:"avg_bitrate_kbps"`
}

// LibraryHealthMetrics represents health indicators for a library
type LibraryHealthMetrics struct {
	StaleContent    int     `json:"stale_content_count"` // Not watched in 90 days
	PopularityScore float64 `json:"popularity_score"`    // Plays per item
	EngagementScore float64 `json:"engagement_score"`    // Completion rate
	GrowthRate      float64 `json:"growth_rate_percent"` // Change vs previous period
}
