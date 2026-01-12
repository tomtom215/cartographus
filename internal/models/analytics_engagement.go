// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package models

import (
	"time"
)

// UserEngagement represents comprehensive engagement metrics for a single user
type UserEngagement struct {
	UserID                int       `json:"user_id"`
	Username              string    `json:"username"`
	TotalWatchTimeMinutes float64   `json:"total_watch_time_minutes"`
	TotalSessions         int       `json:"total_sessions"`
	AverageSessionMinutes float64   `json:"average_session_minutes"`
	FirstSeenAt           time.Time `json:"first_seen_at"`
	LastSeenAt            time.Time `json:"last_seen_at"`
	DaysSinceFirstSeen    int       `json:"days_since_first_seen"`
	DaysSinceLastSeen     int       `json:"days_since_last_seen"`
	TotalContentItems     int       `json:"total_content_items"`
	UniqueContentItems    int       `json:"unique_content_items"`
	MostWatchedType       *string   `json:"most_watched_type,omitempty"`
	MostWatchedTitle      *string   `json:"most_watched_title,omitempty"`
	ActivityScore         float64   `json:"activity_score"`
	AvgCompletionRate     float64   `json:"avg_completion_rate"`
	FullyWatchedCount     int       `json:"fully_watched_count"`
	ReturnVisitorRate     float64   `json:"return_visitor_rate"`
	UniqueLocations       int       `json:"unique_locations"`
	UniquePlatforms       int       `json:"unique_platforms"`
}

// ViewingPatternByHour represents viewing activity aggregated by hour of day
type ViewingPatternByHour struct {
	HourOfDay        int     `json:"hour_of_day"` // 0-23
	SessionCount     int     `json:"session_count"`
	WatchTimeMinutes float64 `json:"watch_time_minutes"`
	UniqueUsers      int     `json:"unique_users"`
	AvgCompletion    float64 `json:"avg_completion"`
}

// ViewingPatternByDay represents viewing activity aggregated by day of week
type ViewingPatternByDay struct {
	DayOfWeek        int     `json:"day_of_week"` // 0 = Sunday, 6 = Saturday
	DayName          string  `json:"day_name"`
	SessionCount     int     `json:"session_count"`
	WatchTimeMinutes float64 `json:"watch_time_minutes"`
	UniqueUsers      int     `json:"unique_users"`
	AvgCompletion    float64 `json:"avg_completion"`
}

// UserEngagementSummary represents aggregated engagement metrics across all users
type UserEngagementSummary struct {
	TotalUsers            int     `json:"total_users"`
	ActiveUsers           int     `json:"active_users"`
	TotalWatchTimeMinutes float64 `json:"total_watch_time_minutes"`
	TotalSessions         int     `json:"total_sessions"`
	AvgSessionMinutes     float64 `json:"avg_session_minutes"`
	AvgUserWatchTime      float64 `json:"avg_user_watch_time_minutes"`
	AvgCompletionRate     float64 `json:"avg_completion_rate"`
	ReturnVisitorRate     float64 `json:"return_visitor_rate"`
}

// UserEngagementAnalytics represents comprehensive user engagement analytics
type UserEngagementAnalytics struct {
	Summary               UserEngagementSummary  `json:"summary"`
	TopUsers              []UserEngagement       `json:"top_users"`
	ViewingPatternsByHour []ViewingPatternByHour `json:"viewing_patterns_by_hour"`
	ViewingPatternsByDay  []ViewingPatternByDay  `json:"viewing_patterns_by_day"`
	MostActiveHour        *int                   `json:"most_active_hour,omitempty"`
	MostActiveDay         *int                   `json:"most_active_day,omitempty"`
}
