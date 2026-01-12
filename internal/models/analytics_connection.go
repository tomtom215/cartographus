// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package models

// ConnectionSecurityAnalytics represents connection security patterns
type ConnectionSecurityAnalytics struct {
	TotalPlaybacks      int                       `json:"total_playbacks"`
	SecureConnections   int                       `json:"secure_connections"`
	InsecureConnections int                       `json:"insecure_connections"`
	SecurePercent       float64                   `json:"secure_percent"`
	RelayedConnections  ConnectionRelayStats      `json:"relayed_connections"`
	LocalConnections    ConnectionLocalStats      `json:"local_connections"`
	ByUser              []UserConnectionStats     `json:"by_user"`
	ByPlatform          []PlatformConnectionStats `json:"by_platform"`
}

// ConnectionRelayStats represents relayed connection statistics
type ConnectionRelayStats struct {
	Count   int      `json:"count"`
	Percent float64  `json:"percent"`
	Users   []string `json:"users"`
	Reason  string   `json:"reason"`
}

// ConnectionLocalStats represents local connection statistics
type ConnectionLocalStats struct {
	Count   int     `json:"count"`
	Percent float64 `json:"percent"`
}

// UserConnectionStats represents connection statistics for a user
type UserConnectionStats struct {
	Username     string  `json:"username"`
	TotalStreams int     `json:"total_streams"`
	SecureRate   float64 `json:"secure_rate_percent"`
	RelayRate    float64 `json:"relay_rate_percent"`
	LocalRate    float64 `json:"local_rate_percent"`
}

// PlatformConnectionStats represents connection statistics by platform
type PlatformConnectionStats struct {
	Platform   string  `json:"platform"`
	SecureRate float64 `json:"secure_rate_percent"`
	RelayRate  float64 `json:"relay_rate_percent"`
}

// PausePatternAnalytics represents pause behavior and engagement analysis
type PausePatternAnalytics struct {
	TotalPlaybacks      int                 `json:"total_playbacks"`
	AvgPausesPerSession float64             `json:"avg_pauses_per_session"`
	HighPauseContent    []HighPauseContent  `json:"high_pause_content"`
	PauseDistribution   []PauseDistribution `json:"pause_distribution"`
	PauseTimingHeatmap  []PauseTimingBucket `json:"pause_timing_heatmap"`
	UserPausePatterns   []UserPauseStats    `json:"user_pause_patterns"`
	QualityIndicators   PauseQualityMetrics `json:"quality_indicators"`
}

// HighPauseContent represents content with high pause frequency
type HighPauseContent struct {
	Title                 string  `json:"title"`
	MediaType             string  `json:"media_type"`
	AveragePauses         float64 `json:"average_pauses"`
	CompletionRate        float64 `json:"completion_rate"`
	PotentialQualityIssue bool    `json:"potential_quality_issue"`
	PlaybackCount         int     `json:"playback_count"`
}

// PauseDistribution represents pause count distribution
type PauseDistribution struct {
	PauseBucket   string  `json:"pause_bucket"` // "0-2", "3-5", "6-10", "11+"
	PlaybackCount int     `json:"playback_count"`
	Percentage    float64 `json:"percentage"`
	AvgCompletion float64 `json:"avg_completion"`
}

// PauseTimingBucket represents pause frequency at content duration percentages
type PauseTimingBucket struct {
	DurationPercent int `json:"duration_percent"` // 0-100 in 10% increments
	PauseCount      int `json:"pause_count"`
}

// UserPauseStats represents pause statistics for a user
type UserPauseStats struct {
	Username      string  `json:"username"`
	AvgPauses     float64 `json:"avg_pauses"`
	BingeWatcher  bool    `json:"binge_watcher"` // Low pause rate indicates binging
	TotalSessions int     `json:"total_sessions"`
}

// PauseQualityMetrics represents quality indicators from pause patterns
type PauseQualityMetrics struct {
	HighEngagementThreshold float64 `json:"high_engagement_threshold_pauses"` // < 2 pauses = high engagement
	LowEngagementCount      int     `json:"low_engagement_count"`             // > 10 pauses
	PotentialIssuesDetected int     `json:"potential_issues_detected"`
}
