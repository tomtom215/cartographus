// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package models

import (
	"time"
)

// ConcurrentStreamsAnalytics represents concurrent stream analytics for infrastructure planning
type ConcurrentStreamsAnalytics struct {
	PeakConcurrent         int                            `json:"peak_concurrent"`
	PeakTime               time.Time                      `json:"peak_time"`
	AvgConcurrent          float64                        `json:"avg_concurrent"`
	TotalSessions          int                            `json:"total_sessions"`
	TimeSeriesData         []ConcurrentStreamsTimeBucket  `json:"time_series_data"`
	ByTranscodeDecision    []ConcurrentStreamsByType      `json:"by_transcode_decision"`
	ByDayOfWeek            []ConcurrentStreamsByDayOfWeek `json:"by_day_of_week"`
	ByHourOfDay            []ConcurrentStreamsByHour      `json:"by_hour_of_day"`
	CapacityRecommendation string                         `json:"capacity_recommendation"`
}

// ConcurrentStreamsTimeBucket represents concurrent streams at a specific time
type ConcurrentStreamsTimeBucket struct {
	Timestamp       time.Time `json:"timestamp"`
	ConcurrentCount int       `json:"concurrent_count"`
	DirectPlay      int       `json:"direct_play"`
	DirectStream    int       `json:"direct_stream"`
	Transcode       int       `json:"transcode"`
}

// ConcurrentStreamsByType represents concurrent streams by transcode decision
type ConcurrentStreamsByType struct {
	TranscodeDecision string  `json:"transcode_decision"`
	AvgConcurrent     float64 `json:"avg_concurrent"`
	MaxConcurrent     int     `json:"max_concurrent"`
	Percentage        float64 `json:"percentage"`
}

// ConcurrentStreamsByDayOfWeek represents concurrent stream patterns by day
type ConcurrentStreamsByDayOfWeek struct {
	DayOfWeek      int     `json:"day_of_week"` // 0=Sunday, 6=Saturday
	AvgConcurrent  float64 `json:"avg_concurrent"`
	PeakConcurrent int     `json:"peak_concurrent"`
}

// ConcurrentStreamsByHour represents concurrent stream patterns by hour
type ConcurrentStreamsByHour struct {
	Hour           int     `json:"hour"` // 0-23
	AvgConcurrent  float64 `json:"avg_concurrent"`
	PeakConcurrent int     `json:"peak_concurrent"`
}
