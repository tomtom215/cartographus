// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package models provides data structures for analytics and API responses.
// This file contains models for device migration tracking analytics.
package models

import "time"

// DeviceMigration represents a single device platform migration event for a user.
// A migration occurs when a user switches from one platform to another.
type DeviceMigration struct {
	// UserID is the internal identifier for the user
	UserID int `json:"user_id"`
	// Username is the display name of the user
	Username string `json:"username"`
	// FromPlatform is the platform the user was previously using
	FromPlatform string `json:"from_platform"`
	// ToPlatform is the platform the user switched to
	ToPlatform string `json:"to_platform"`
	// MigrationDate is when the switch was first detected
	MigrationDate time.Time `json:"migration_date"`
	// SessionsBeforeMigration is how many sessions the user had on the previous platform
	SessionsBeforeMigration int `json:"sessions_before_migration"`
	// SessionsAfterMigration is how many sessions the user has had on the new platform
	SessionsAfterMigration int `json:"sessions_after_migration"`
	// IsPermanentSwitch indicates if the user abandoned the previous platform
	IsPermanentSwitch bool `json:"is_permanent_switch"`
}

// UserDeviceProfile represents a user's complete device usage history and patterns.
type UserDeviceProfile struct {
	// UserID is the internal identifier for the user
	UserID int `json:"user_id"`
	// Username is the display name of the user
	Username string `json:"username"`
	// TotalPlatformsUsed is the count of unique platforms this user has ever used
	TotalPlatformsUsed int `json:"total_platforms_used"`
	// PrimaryPlatform is the platform with the most sessions
	PrimaryPlatform string `json:"primary_platform"`
	// PrimaryPlatformPercentage is the percentage of sessions on the primary platform
	PrimaryPlatformPercentage float64 `json:"primary_platform_percentage"`
	// FirstSeenAt is when the user was first seen on any platform
	FirstSeenAt time.Time `json:"first_seen_at"`
	// LastSeenAt is when the user was last seen on any platform
	LastSeenAt time.Time `json:"last_seen_at"`
	// DaysSinceFirstSeen is the number of days since the user was first seen
	DaysSinceFirstSeen int `json:"days_since_first_seen"`
	// DaysSinceLastSeen is the number of days since the user was last seen
	DaysSinceLastSeen int `json:"days_since_last_seen"`
	// TotalSessions is the total number of playback sessions across all platforms
	TotalSessions int `json:"total_sessions"`
	// TotalMigrations is the number of platform switches
	TotalMigrations int `json:"total_migrations"`
	// PlatformHistory contains the user's usage of each platform
	PlatformHistory []UserPlatformUsage `json:"platform_history"`
	// IsMultiDevice indicates if the user uses multiple platforms
	IsMultiDevice bool `json:"is_multi_device"`
}

// UserPlatformUsage represents a user's usage of a specific platform.
type UserPlatformUsage struct {
	// Platform is the name of the platform (e.g., "Android", "iOS", "Web")
	Platform string `json:"platform"`
	// FirstUsed is when the user first used this platform
	FirstUsed time.Time `json:"first_used"`
	// LastUsed is when the user last used this platform
	LastUsed time.Time `json:"last_used"`
	// SessionCount is the total number of sessions on this platform
	SessionCount int `json:"session_count"`
	// TotalWatchTimeMinutes is the total watch time on this platform
	TotalWatchTimeMinutes float64 `json:"total_watch_time_minutes"`
	// Percentage is this platform's share of the user's total sessions
	Percentage float64 `json:"percentage"`
	// IsActive indicates if the platform was used in the last 30 days
	IsActive bool `json:"is_active"`
	// IsPrimary indicates if this is the user's primary platform
	IsPrimary bool `json:"is_primary"`
}

// PlatformAdoptionTrend represents platform adoption metrics over a time period.
type PlatformAdoptionTrend struct {
	// Date is the date bucket for this trend data
	Date string `json:"date"`
	// Platform is the name of the platform
	Platform string `json:"platform"`
	// NewUsers is the count of users who first used this platform in this period
	NewUsers int `json:"new_users"`
	// ActiveUsers is the count of users who used this platform in this period
	ActiveUsers int `json:"active_users"`
	// ReturningUsers is the count of users who returned to this platform
	ReturningUsers int `json:"returning_users"`
	// SessionCount is the total sessions on this platform in this period
	SessionCount int `json:"session_count"`
	// MarketShare is this platform's share of total sessions in this period
	MarketShare float64 `json:"market_share"`
}

// PlatformTransition represents a common transition path between platforms.
type PlatformTransition struct {
	// FromPlatform is the source platform
	FromPlatform string `json:"from_platform"`
	// ToPlatform is the destination platform
	ToPlatform string `json:"to_platform"`
	// TransitionCount is how many times this transition has occurred
	TransitionCount int `json:"transition_count"`
	// UniqueUsers is how many unique users made this transition
	UniqueUsers int `json:"unique_users"`
	// AvgDaysBeforeSwitch is the average days on source platform before switching
	AvgDaysBeforeSwitch float64 `json:"avg_days_before_switch"`
	// ReturnRate is the percentage of users who returned to the source platform
	ReturnRate float64 `json:"return_rate"`
}

// DeviceMigrationSummary represents aggregate statistics about device migrations.
type DeviceMigrationSummary struct {
	// TotalUsers is the total number of users analyzed
	TotalUsers int `json:"total_users"`
	// MultiDeviceUsers is the count of users using 2+ platforms
	MultiDeviceUsers int `json:"multi_device_users"`
	// MultiDevicePercentage is the percentage of users using multiple platforms
	MultiDevicePercentage float64 `json:"multi_device_percentage"`
	// TotalMigrations is the total number of platform switches detected
	TotalMigrations int `json:"total_migrations"`
	// AvgPlatformsPerUser is the average number of platforms per user
	AvgPlatformsPerUser float64 `json:"avg_platforms_per_user"`
	// MostCommonPrimaryPlatform is the platform most often used as primary
	MostCommonPrimaryPlatform string `json:"most_common_primary_platform"`
	// FastestGrowingPlatform is the platform with highest recent adoption rate
	FastestGrowingPlatform string `json:"fastest_growing_platform"`
	// DecliningSplatforms are platforms with decreasing usage
	DecliningPlatforms []string `json:"declining_platforms,omitempty"`
}

// DeviceMigrationAnalytics represents the complete device migration analytics response.
type DeviceMigrationAnalytics struct {
	// Summary contains aggregate device migration statistics
	Summary DeviceMigrationSummary `json:"summary"`
	// TopUserProfiles contains detailed profiles for users with notable migration patterns
	TopUserProfiles []UserDeviceProfile `json:"top_user_profiles"`
	// RecentMigrations contains the most recent platform migration events
	RecentMigrations []DeviceMigration `json:"recent_migrations"`
	// AdoptionTrends contains platform adoption trends over time
	AdoptionTrends []PlatformAdoptionTrend `json:"adoption_trends"`
	// CommonTransitions contains the most common platform transition paths
	CommonTransitions []PlatformTransition `json:"common_transitions"`
	// PlatformDistribution contains current platform usage distribution
	PlatformDistribution []PlatformStats `json:"platform_distribution"`
	// Metadata contains query execution information for observability
	Metadata DeviceMigrationMetadata `json:"metadata"`
}

// DeviceMigrationMetadata contains metadata for device migration analytics queries.
// This supports the deterministic, auditable, traceable, and observable requirements.
type DeviceMigrationMetadata struct {
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
	// UniquePlatformsFound is the number of distinct platforms in the data
	UniquePlatformsFound int `json:"unique_platforms_found"`
	// MigrationWindowDays is the threshold used for detecting migrations
	MigrationWindowDays int `json:"migration_window_days"`
}
