// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package models provides data structures for the Cartographus application.
// This file contains Quality of Experience (QoE) analytics models following Netflix
// and industry standard metrics for streaming video quality measurement.
//
// Key metrics based on industry standards:
// - EBVS (Exit Before Video Starts) - Sessions abandoned before playback
// - VST (Video Start Time) - Time to first frame (approximated)
// - Rebuffering Ratio - Buffering time vs playback time
// - Quality Degradation - Source vs stream quality mismatches
// - Bitrate Stability - Consistency of streaming quality
//
// References:
// - Netflix QoE: https://netflixtechblog.com/streaming-video-experimentation-at-netflix
// - Conviva SPI: Stream Performance Index methodology
// - Mux Data: Industry standard video analytics
package models

import "time"

// QoEDashboard represents comprehensive Quality of Experience metrics
type QoEDashboard struct {
	// Summary provides high-level QoE health indicators
	Summary QoESummary `json:"summary"`

	// Trends provides time-series QoE metrics
	Trends []QoETrendPoint `json:"trends"`

	// ByPlatform breaks down QoE by device/platform
	ByPlatform []QoEByPlatform `json:"by_platform"`

	// ByTranscodeDecision shows QoE differences between direct play and transcode
	ByTranscodeDecision []QoEByTranscode `json:"by_transcode_decision"`

	// TopIssues lists the most impactful quality issues
	TopIssues []QoEIssue `json:"top_issues"`

	// Metadata provides query provenance
	Metadata QoEQueryMetadata `json:"metadata"`
}

// QoESummary provides aggregate QoE health metrics
type QoESummary struct {
	// TotalSessions is the count of playback sessions analyzed
	TotalSessions int64 `json:"total_sessions"`

	// EBVSRate is Exit Before Video Starts rate (percent_complete = 0 AND play_duration < 10s)
	// Industry benchmark: <2% is excellent, >5% indicates issues
	EBVSRate float64 `json:"ebvs_rate"`

	// EBVSCount is the absolute count of EBVS events
	EBVSCount int64 `json:"ebvs_count"`

	// QualityDegradeRate is the percentage of sessions where stream quality < source quality
	// Indicates transcoding or bandwidth limitations
	QualityDegradeRate float64 `json:"quality_degrade_rate"`

	// QualityDegradeCount is the absolute count of degraded sessions
	QualityDegradeCount int64 `json:"quality_degrade_count"`

	// TranscodeRate is the percentage of sessions requiring transcoding
	TranscodeRate float64 `json:"transcode_rate"`

	// TranscodeCount is the absolute count of transcoded sessions
	TranscodeCount int64 `json:"transcode_count"`

	// DirectPlayRate is the percentage of sessions playing directly without transcoding
	DirectPlayRate float64 `json:"direct_play_rate"`

	// DirectPlayCount is the absolute count of direct play sessions
	DirectPlayCount int64 `json:"direct_play_count"`

	// AvgCompletion is the average completion percentage for all sessions
	AvgCompletion float64 `json:"avg_completion"`

	// HighCompletionRate is the percentage of sessions with >80% completion
	HighCompletionRate float64 `json:"high_completion_rate"`

	// PauseRate is the percentage of sessions with pause events (paused_counter > 0)
	// High pause rates may indicate buffering or quality issues
	PauseRate float64 `json:"pause_rate"`

	// AvgPauseCount is the average number of pauses per session
	AvgPauseCount float64 `json:"avg_pause_count"`

	// RelayedRate is the percentage of sessions using relay connections
	// Relay indicates inability to establish direct connection
	RelayedRate float64 `json:"relayed_rate"`

	// SecureConnectionRate is the percentage of sessions using secure connections
	SecureConnectionRate float64 `json:"secure_connection_rate"`

	// AvgBitrateMbps is the average streaming bitrate in Mbps
	AvgBitrateMbps float64 `json:"avg_bitrate_mbps"`

	// BitrateP50Mbps is the median bitrate (50th percentile)
	BitrateP50Mbps float64 `json:"bitrate_p50_mbps"`

	// BitrateP95Mbps is the 95th percentile bitrate
	BitrateP95Mbps float64 `json:"bitrate_p95_mbps"`

	// QoEScore is a composite score (0-100) based on weighted metrics
	// Formula: 100 - (EBVS_penalty + QualityDegrade_penalty + Pause_penalty)
	QoEScore float64 `json:"qoe_score"`

	// QoEGrade is a letter grade (A, B, C, D, F) based on QoEScore
	QoEGrade string `json:"qoe_grade"`
}

// QoETrendPoint represents QoE metrics at a specific point in time
type QoETrendPoint struct {
	// Timestamp is the time bucket for these metrics
	Timestamp time.Time `json:"timestamp"`

	// SessionCount is the number of sessions in this time bucket
	SessionCount int64 `json:"session_count"`

	// EBVSRate for this time period
	EBVSRate float64 `json:"ebvs_rate"`

	// QualityDegradeRate for this time period
	QualityDegradeRate float64 `json:"quality_degrade_rate"`

	// TranscodeRate for this time period
	TranscodeRate float64 `json:"transcode_rate"`

	// AvgCompletion for this time period
	AvgCompletion float64 `json:"avg_completion"`

	// AvgBitrateMbps for this time period
	AvgBitrateMbps float64 `json:"avg_bitrate_mbps"`

	// QoEScore for this time period
	QoEScore float64 `json:"qoe_score"`
}

// QoEByPlatform provides QoE breakdown by platform/device
type QoEByPlatform struct {
	// Platform name (e.g., "iOS", "Android", "Roku", "Web")
	Platform string `json:"platform"`

	// SessionCount for this platform
	SessionCount int64 `json:"session_count"`

	// SessionPercentage is this platform's share of total sessions
	SessionPercentage float64 `json:"session_percentage"`

	// EBVSRate for this platform
	EBVSRate float64 `json:"ebvs_rate"`

	// QualityDegradeRate for this platform
	QualityDegradeRate float64 `json:"quality_degrade_rate"`

	// TranscodeRate for this platform
	TranscodeRate float64 `json:"transcode_rate"`

	// DirectPlayRate for this platform
	DirectPlayRate float64 `json:"direct_play_rate"`

	// AvgCompletion for this platform
	AvgCompletion float64 `json:"avg_completion"`

	// AvgBitrateMbps for this platform
	AvgBitrateMbps float64 `json:"avg_bitrate_mbps"`

	// QoEScore for this platform
	QoEScore float64 `json:"qoe_score"`

	// QoEGrade for this platform
	QoEGrade string `json:"qoe_grade"`
}

// QoEByTranscode provides QoE comparison between transcode decisions
type QoEByTranscode struct {
	// TranscodeDecision is the decision type ("direct play", "transcode", "copy")
	TranscodeDecision string `json:"transcode_decision"`

	// SessionCount for this decision type
	SessionCount int64 `json:"session_count"`

	// SessionPercentage is this decision's share of total sessions
	SessionPercentage float64 `json:"session_percentage"`

	// EBVSRate for this decision type
	EBVSRate float64 `json:"ebvs_rate"`

	// AvgCompletion for this decision type
	AvgCompletion float64 `json:"avg_completion"`

	// AvgBitrateMbps for this decision type
	AvgBitrateMbps float64 `json:"avg_bitrate_mbps"`

	// QoEScore for this decision type
	QoEScore float64 `json:"qoe_score"`
}

// QoEIssue represents a specific quality issue with impact assessment
type QoEIssue struct {
	// IssueType categorizes the issue
	// Types: "high_ebvs", "quality_degradation", "high_transcode", "high_pause", "low_completion"
	IssueType string `json:"issue_type"`

	// Severity is "critical", "warning", or "info"
	Severity string `json:"severity"`

	// Title is a human-readable issue title
	Title string `json:"title"`

	// Description provides details about the issue
	Description string `json:"description"`

	// AffectedSessions is the count of sessions impacted
	AffectedSessions int64 `json:"affected_sessions"`

	// ImpactPercentage is the percentage of total sessions affected
	ImpactPercentage float64 `json:"impact_percentage"`

	// Recommendation provides actionable guidance
	Recommendation string `json:"recommendation"`

	// RelatedDimension is the dimension most correlated with this issue (e.g., "platform:Roku")
	RelatedDimension string `json:"related_dimension,omitempty"`
}

// QoEQueryMetadata provides provenance and auditability
type QoEQueryMetadata struct {
	// QueryHash for reproducibility
	QueryHash string `json:"query_hash"`

	// DataRangeStart is the earliest data analyzed
	DataRangeStart time.Time `json:"data_range_start"`

	// DataRangeEnd is the latest data analyzed
	DataRangeEnd time.Time `json:"data_range_end"`

	// TrendInterval is the granularity of trend data ("hour", "day")
	TrendInterval string `json:"trend_interval"`

	// EventCount is total events analyzed
	EventCount int64 `json:"event_count"`

	// GeneratedAt is when this was generated
	GeneratedAt time.Time `json:"generated_at"`

	// QueryTimeMs is execution time
	QueryTimeMs int64 `json:"query_time_ms"`

	// Cached indicates if from cache
	Cached bool `json:"cached"`
}
