// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package models

import (
	"time"
)

// PeriodMetrics represents metrics for a single time period
type PeriodMetrics struct {
	StartDate        time.Time `json:"start_date"`
	EndDate          time.Time `json:"end_date"`
	PlaybackCount    int       `json:"playback_count"`
	UniqueUsers      int       `json:"unique_users"`
	WatchTimeMinutes float64   `json:"watch_time_minutes"`
	AvgSessionMins   float64   `json:"avg_session_minutes"`
	AvgCompletion    float64   `json:"avg_completion"`
	UniqueContent    int       `json:"unique_content"`
	UniqueLocations  int       `json:"unique_locations"`
}

// ComparativeMetrics represents a comparison between two time periods
type ComparativeMetrics struct {
	Metric                 string  `json:"metric"`
	CurrentValue           float64 `json:"current_value"`
	PreviousValue          float64 `json:"previous_value"`
	AbsoluteChange         float64 `json:"absolute_change"`
	PercentageChange       float64 `json:"percentage_change"`
	GrowthDirection        string  `json:"growth_direction"` // "up", "down", "stable"
	IsImprovement          bool    `json:"is_improvement"`
	CurrentValueFormatted  string  `json:"current_value_formatted,omitempty"`
	PreviousValueFormatted string  `json:"previous_value_formatted,omitempty"`
}

// TopContentComparison represents content ranking comparison between periods
type TopContentComparison struct {
	Title          string  `json:"title"`
	CurrentRank    int     `json:"current_rank"`
	PreviousRank   int     `json:"previous_rank"`
	RankChange     int     `json:"rank_change"`
	CurrentCount   int     `json:"current_count"`
	PreviousCount  int     `json:"previous_count"`
	CountChange    int     `json:"count_change"`
	CountChangePct float64 `json:"count_change_pct"`
	Trending       string  `json:"trending"` // "rising", "falling", "stable", "new"
}

// TopUserComparison represents user ranking comparison between periods
type TopUserComparison struct {
	Username           string  `json:"username"`
	CurrentRank        int     `json:"current_rank"`
	PreviousRank       int     `json:"previous_rank"`
	RankChange         int     `json:"rank_change"`
	CurrentWatchTime   float64 `json:"current_watch_time"`
	PreviousWatchTime  float64 `json:"previous_watch_time"`
	WatchTimeChange    float64 `json:"watch_time_change"`
	WatchTimeChangePct float64 `json:"watch_time_change_pct"`
	Trending           string  `json:"trending"` // "rising", "falling", "stable", "new"
}

// ComparativeAnalytics represents comprehensive period-over-period comparison analytics
type ComparativeAnalytics struct {
	CurrentPeriod        PeriodMetrics          `json:"current_period"`
	PreviousPeriod       PeriodMetrics          `json:"previous_period"`
	ComparisonType       string                 `json:"comparison_type"` // "week", "month", "quarter", "year", "custom"
	MetricsComparison    []ComparativeMetrics   `json:"metrics_comparison"`
	TopContentComparison []TopContentComparison `json:"top_content_comparison"`
	TopUserComparison    []TopUserComparison    `json:"top_user_comparison"`
	OverallTrend         string                 `json:"overall_trend"` // "growing", "declining", "stable"
	KeyInsights          []string               `json:"key_insights"`
}
