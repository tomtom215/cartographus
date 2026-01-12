// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package models

// BandwidthAnalytics represents overall bandwidth usage analytics
type BandwidthAnalytics struct {
	TotalBandwidthGB      float64                 `json:"total_bandwidth_gb"`
	DirectPlayBandwidthGB float64                 `json:"direct_play_bandwidth_gb"`
	TranscodeBandwidthGB  float64                 `json:"transcode_bandwidth_gb"`
	AvgBandwidthMbps      float64                 `json:"avg_bandwidth_mbps"`
	PeakBandwidthMbps     float64                 `json:"peak_bandwidth_mbps"`
	ByTranscode           []BandwidthByTranscode  `json:"by_transcode"`
	ByResolution          []BandwidthByResolution `json:"by_resolution"`
	ByCodec               []BandwidthByCodec      `json:"by_codec"`
	Trends                []BandwidthTrend        `json:"trends"`
	TopUsers              []BandwidthByUser       `json:"top_users"`
}

// BandwidthByTranscode represents bandwidth usage by transcode decision
type BandwidthByTranscode struct {
	TranscodeDecision string  `json:"transcode_decision"`
	BandwidthGB       float64 `json:"bandwidth_gb"`
	PlaybackCount     int     `json:"playback_count"`
	AvgBandwidthMbps  float64 `json:"avg_bandwidth_mbps"`
	Percentage        float64 `json:"percentage"`
}

// BandwidthByResolution represents bandwidth usage by video resolution
type BandwidthByResolution struct {
	Resolution       string  `json:"resolution"`
	BandwidthGB      float64 `json:"bandwidth_gb"`
	PlaybackCount    int     `json:"playback_count"`
	AvgBandwidthMbps float64 `json:"avg_bandwidth_mbps"`
	Percentage       float64 `json:"percentage"`
}

// BandwidthByCodec represents bandwidth usage by codec combination
type BandwidthByCodec struct {
	VideoCodec       string  `json:"video_codec"`
	AudioCodec       string  `json:"audio_codec"`
	BandwidthGB      float64 `json:"bandwidth_gb"`
	PlaybackCount    int     `json:"playback_count"`
	AvgBandwidthMbps float64 `json:"avg_bandwidth_mbps"`
}

// BandwidthTrend represents bandwidth usage over time
type BandwidthTrend struct {
	Date          string  `json:"date"`
	BandwidthGB   float64 `json:"bandwidth_gb"`
	PlaybackCount int     `json:"playback_count"`
	AvgMbps       float64 `json:"avg_mbps"`
}

// BandwidthByUser represents bandwidth usage by user
type BandwidthByUser struct {
	UserID           int     `json:"user_id"`
	Username         string  `json:"username"`
	BandwidthGB      float64 `json:"bandwidth_gb"`
	PlaybackCount    int     `json:"playback_count"`
	DirectPlayCount  int     `json:"direct_play_count"`
	TranscodeCount   int     `json:"transcode_count"`
	AvgBandwidthMbps float64 `json:"avg_bandwidth_mbps"`
}
