// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package models provides data structures for analytics and API responses.
// This file contains models for content discovery analytics.
package models

import "time"

// ContentDiscoveryItem represents discovery metrics for a single content item.
type ContentDiscoveryItem struct {
	// RatingKey is the unique identifier for the content
	RatingKey string `json:"rating_key"`
	// Title is the content title
	Title string `json:"title"`
	// MediaType is the type of content (movie, episode, track)
	MediaType string `json:"media_type"`
	// LibraryName is the library containing this content
	LibraryName string `json:"library_name"`
	// AddedAt is when the content was added to the library
	AddedAt time.Time `json:"added_at"`
	// FirstWatchedAt is when the content was first watched (nil if never watched)
	FirstWatchedAt *time.Time `json:"first_watched_at,omitempty"`
	// TimeToFirstWatchHours is the hours between added and first watch (nil if never watched)
	TimeToFirstWatchHours *float64 `json:"time_to_first_watch_hours,omitempty"`
	// TotalPlaybacks is the number of times this content has been played
	TotalPlaybacks int `json:"total_playbacks"`
	// UniqueViewers is the number of unique users who watched this content
	UniqueViewers int `json:"unique_viewers"`
	// AvgCompletion is the average completion rate
	AvgCompletion float64 `json:"avg_completion"`
	// DiscoveryVelocity indicates how quickly the content was discovered (fast/medium/slow/not_discovered)
	DiscoveryVelocity string `json:"discovery_velocity"`
	// Year is the release year of the content
	Year *int `json:"year,omitempty"`
	// Genres are the content genres
	Genres *string `json:"genres,omitempty"`
}

// DiscoveryTimeBucket represents discovery rate within a time bucket after content addition.
type DiscoveryTimeBucket struct {
	// Bucket is the time bucket label (e.g., "0-24h", "1-7d", "7-30d", "30d+")
	Bucket string `json:"bucket"`
	// BucketMinHours is the minimum hours in this bucket
	BucketMinHours int `json:"bucket_min_hours"`
	// BucketMaxHours is the maximum hours in this bucket
	BucketMaxHours int `json:"bucket_max_hours"`
	// ContentCount is the number of content items discovered in this time window
	ContentCount int `json:"content_count"`
	// Percentage is this bucket's share of all discovered content
	Percentage float64 `json:"percentage"`
}

// EarlyAdopter represents a user who consistently discovers new content quickly.
type EarlyAdopter struct {
	// UserID is the internal identifier for the user
	UserID int `json:"user_id"`
	// Username is the display name of the user
	Username string `json:"username"`
	// EarlyDiscoveryCount is how many times they watched content within 24h of addition
	EarlyDiscoveryCount int `json:"early_discovery_count"`
	// TotalDiscoveries is total content items they've watched
	TotalDiscoveries int `json:"total_discoveries"`
	// EarlyDiscoveryRate is the percentage of their watches that are early discoveries
	EarlyDiscoveryRate float64 `json:"early_discovery_rate"`
	// AvgTimeToDiscoveryHours is their average time from content addition to first watch
	AvgTimeToDiscoveryHours float64 `json:"avg_time_to_discovery_hours"`
	// FirstSeenAt is when this user was first seen
	FirstSeenAt time.Time `json:"first_seen_at"`
	// FavoriteLibrary is the library they discover from most
	FavoriteLibrary *string `json:"favorite_library,omitempty"`
}

// StaleContent represents content that was added but never watched.
type StaleContent struct {
	// RatingKey is the unique identifier for the content
	RatingKey string `json:"rating_key"`
	// Title is the content title
	Title string `json:"title"`
	// MediaType is the type of content
	MediaType string `json:"media_type"`
	// LibraryName is the library containing this content
	LibraryName string `json:"library_name"`
	// AddedAt is when the content was added
	AddedAt time.Time `json:"added_at"`
	// DaysSinceAdded is the number of days since the content was added
	DaysSinceAdded int `json:"days_since_added"`
	// Year is the release year
	Year *int `json:"year,omitempty"`
	// Genres are the content genres
	Genres *string `json:"genres,omitempty"`
	// ContentRating is the content's rating (G, PG, R, etc.)
	ContentRating *string `json:"content_rating,omitempty"`
}

// LibraryDiscoveryStats represents discovery statistics for a library.
type LibraryDiscoveryStats struct {
	// LibraryName is the name of the library
	LibraryName string `json:"library_name"`
	// TotalItems is the total content items in the library with added_at data
	TotalItems int `json:"total_items"`
	// WatchedItems is the number of items that have been watched at least once
	WatchedItems int `json:"watched_items"`
	// UnwatchedItems is the number of items never watched
	UnwatchedItems int `json:"unwatched_items"`
	// DiscoveryRate is the percentage of items watched
	DiscoveryRate float64 `json:"discovery_rate"`
	// AvgTimeToDiscoveryHours is the average time from add to first watch
	AvgTimeToDiscoveryHours float64 `json:"avg_time_to_discovery_hours"`
	// MedianTimeToDiscoveryHours is the median time from add to first watch
	MedianTimeToDiscoveryHours float64 `json:"median_time_to_discovery_hours"`
	// EarlyDiscoveryRate is percentage discovered within 24 hours
	EarlyDiscoveryRate float64 `json:"early_discovery_rate"`
}

// DiscoveryTrend represents discovery metrics over time.
type DiscoveryTrend struct {
	// Date is the date bucket for this trend
	Date string `json:"date"`
	// ContentAdded is the number of content items added in this period
	ContentAdded int `json:"content_added"`
	// ContentDiscovered is the number of newly added items watched for the first time
	ContentDiscovered int `json:"content_discovered"`
	// DiscoveryRate is the percentage of new content watched in this period
	DiscoveryRate float64 `json:"discovery_rate"`
	// AvgTimeToDiscoveryHours is the average discovery time for this period
	AvgTimeToDiscoveryHours float64 `json:"avg_time_to_discovery_hours"`
}

// ContentDiscoverySummary represents aggregate discovery statistics.
type ContentDiscoverySummary struct {
	// TotalContentWithAddedAt is content items with valid added_at data
	TotalContentWithAddedAt int `json:"total_content_with_added_at"`
	// TotalDiscovered is content items that have been watched at least once
	TotalDiscovered int `json:"total_discovered"`
	// TotalNeverWatched is content items never watched
	TotalNeverWatched int `json:"total_never_watched"`
	// OverallDiscoveryRate is the percentage of content that has been watched
	OverallDiscoveryRate float64 `json:"overall_discovery_rate"`
	// AvgTimeToDiscoveryHours is the overall average time to first watch
	AvgTimeToDiscoveryHours float64 `json:"avg_time_to_discovery_hours"`
	// MedianTimeToDiscoveryHours is the overall median time to first watch
	MedianTimeToDiscoveryHours float64 `json:"median_time_to_discovery_hours"`
	// EarlyDiscoveryRate is percentage discovered within 24 hours of addition
	EarlyDiscoveryRate float64 `json:"early_discovery_rate"`
	// FastestDiscoveryHours is the quickest time to first watch
	FastestDiscoveryHours float64 `json:"fastest_discovery_hours"`
	// SlowestDiscoveryDays is the longest time to first watch (in days)
	SlowestDiscoveryDays int `json:"slowest_discovery_days"`
	// RecentAdditionsCount is content added in last 30 days
	RecentAdditionsCount int `json:"recent_additions_count"`
	// RecentDiscoveredCount is recent additions that have been watched
	RecentDiscoveredCount int `json:"recent_discovered_count"`
}

// ContentDiscoveryAnalytics represents the complete content discovery analytics response.
type ContentDiscoveryAnalytics struct {
	// Summary contains aggregate discovery statistics
	Summary ContentDiscoverySummary `json:"summary"`
	// TimeBuckets contains discovery distribution by time to first watch
	TimeBuckets []DiscoveryTimeBucket `json:"time_buckets"`
	// EarlyAdopters contains users who discover content quickly
	EarlyAdopters []EarlyAdopter `json:"early_adopters"`
	// RecentlyDiscovered contains recently watched new content
	RecentlyDiscovered []ContentDiscoveryItem `json:"recently_discovered"`
	// StaleContent contains content added but never watched
	StaleContent []StaleContent `json:"stale_content"`
	// LibraryStats contains per-library discovery statistics
	LibraryStats []LibraryDiscoveryStats `json:"library_stats"`
	// Trends contains discovery patterns over time
	Trends []DiscoveryTrend `json:"trends"`
	// Metadata contains query execution information for observability
	Metadata ContentDiscoveryMetadata `json:"metadata"`
}

// ContentDiscoveryMetadata contains metadata for content discovery analytics queries.
// This supports the deterministic, auditable, traceable, and observable requirements.
type ContentDiscoveryMetadata struct {
	// QueryHash is a deterministic hash of the query parameters for caching
	QueryHash string `json:"query_hash"`
	// ExecutionTimeMS is the query execution time in milliseconds
	ExecutionTimeMS int64 `json:"execution_time_ms"`
	// DataRangeStart is the start of the analyzed date range
	DataRangeStart time.Time `json:"data_range_start"`
	// DataRangeEnd is the end of the analyzed date range
	DataRangeEnd time.Time `json:"data_range_end"`
	// TotalEventsAnalyzed is the number of playback events analyzed
	TotalEventsAnalyzed int `json:"total_events_analyzed"`
	// UniqueContentAnalyzed is the number of unique content items analyzed
	UniqueContentAnalyzed int `json:"unique_content_analyzed"`
	// EarlyDiscoveryThresholdHours is the hours threshold for "early" discovery
	EarlyDiscoveryThresholdHours int `json:"early_discovery_threshold_hours"`
	// StaleContentThresholdDays is the days threshold for "stale" content
	StaleContentThresholdDays int `json:"stale_content_threshold_days"`
}
