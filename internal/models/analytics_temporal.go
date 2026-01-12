// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package models

import (
	"time"
)

// TemporalHeatmapPoint represents a location with playback data for a specific time bucket
type TemporalHeatmapPoint struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Weight    int     `json:"weight"` // Playback count for intensity
}

// TemporalHeatmapBucket represents heatmap data for a specific time period
type TemporalHeatmapBucket struct {
	StartTime time.Time              `json:"start_time"`
	EndTime   time.Time              `json:"end_time"`
	Label     string                 `json:"label"` // Human-readable label (e.g., "Mon 2PM", "Jan 15")
	Points    []TemporalHeatmapPoint `json:"points"`
	Count     int                    `json:"count"` // Total playbacks in this bucket
}

// TemporalHeatmapResponse represents the complete temporal heatmap dataset
type TemporalHeatmapResponse struct {
	Interval   string                  `json:"interval"` // "hour", "day", "week", "month"
	Buckets    []TemporalHeatmapBucket `json:"buckets"`
	TotalCount int                     `json:"total_count"`
	StartDate  time.Time               `json:"start_date"`
	EndDate    time.Time               `json:"end_date"`
}

// ViewingHoursHeatmap represents playback activity by hour and day of week
type ViewingHoursHeatmap struct {
	DayOfWeek     int `json:"day_of_week"`
	Hour          int `json:"hour"`
	PlaybackCount int `json:"playback_count"`
}
