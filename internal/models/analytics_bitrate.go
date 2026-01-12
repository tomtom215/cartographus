// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package models

// BitrateAnalytics represents comprehensive bandwidth and bitrate analysis
// Tracks bitrate at 3 levels (source, transcode, network) for bottleneck identification
type BitrateAnalytics struct {
	// Overall statistics
	AverageSourceBitrate    int `json:"avg_source_bitrate"`    // Kbps - average original file bitrate
	AverageTranscodeBitrate int `json:"avg_transcode_bitrate"` // Kbps - average transcoded stream bitrate
	PeakBitrate             int `json:"peak_bitrate"`          // Kbps - highest bitrate observed
	MedianBitrate           int `json:"median_bitrate"`        // Kbps - median bitrate (50th percentile)

	// Bandwidth analysis
	BandwidthUtilization float64 `json:"bandwidth_utilization"` // % (0-100) - network utilization percentage
	ConstrainedSessions  int     `json:"constrained_sessions"`  // Count of sessions where bitrate > bandwidth

	// Breakdown by resolution
	BitrateByResolution []BitrateByResolutionItem `json:"bitrate_by_resolution"`

	// Time series (for charts with LTTB downsampling)
	BitrateTimeSeries []BitrateTimeSeriesItem `json:"bitrate_timeseries"`
}

// BitrateByResolutionItem represents bitrate statistics for a specific resolution tier
type BitrateByResolutionItem struct {
	Resolution     string  `json:"resolution"`     // "4K", "1080p", "720p", "SD", "Unknown"
	AverageBitrate int     `json:"avg_bitrate"`    // Kbps - average bitrate for this resolution
	SessionCount   int     `json:"session_count"`  // Number of sessions at this resolution
	TranscodeRate  float64 `json:"transcode_rate"` // % (0-100) - percentage of sessions transcoded
}

// BitrateTimeSeriesItem represents bitrate metrics at a specific point in time
// Used for charting bitrate trends and identifying peak usage periods
type BitrateTimeSeriesItem struct {
	Timestamp      string `json:"timestamp"`       // RFC3339 timestamp
	AverageBitrate int    `json:"avg_bitrate"`     // Kbps - average bitrate during this period
	PeakBitrate    int    `json:"peak_bitrate"`    // Kbps - peak bitrate during this period
	ActiveSessions int    `json:"active_sessions"` // Number of concurrent sessions
}
