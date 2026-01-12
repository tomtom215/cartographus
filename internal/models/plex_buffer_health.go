// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package models

import (
	"fmt"
	"time"
)

// ===================================================================================================
// Plex Buffer Health Monitoring Models (Phase 2.1: v1.41)
// ===================================================================================================
// These structures support real-time buffer health monitoring for predictive buffering detection.
// Polls Plex /status/sessions timeline endpoint to calculate buffer fill percentage and drain rate.
//
// Goal: Detect buffering risk 10-15 seconds before playback stalls
// Performance Target: <100ms overhead per session check
// Alert Target: Toast notification within 1 second of critical threshold
// Accuracy Target: Buffer drain rate ±5%
// ===================================================================================================

// PlexSessionTimelineResponse represents the response from /status/sessions/{sessionKey}/timeline
// Endpoint: GET /status/sessions/{sessionKey}?X-Plex-Token={token}
//
// This endpoint provides real-time playback timeline data including buffer information
// which is used to calculate buffer fill percentage and predict buffering events.
type PlexSessionTimelineResponse struct {
	MediaContainer PlexTimelineContainer `json:"MediaContainer"`
}

// PlexTimelineContainer wraps the timeline metadata
type PlexTimelineContainer struct {
	Size     int                    `json:"size"`
	Metadata []PlexTimelineMetadata `json:"Metadata"`
}

// PlexTimelineMetadata represents timeline data for a single session
// Contains critical buffer health information from TranscodeSession
type PlexTimelineMetadata struct {
	// Session identification
	SessionKey string `json:"sessionKey"`
	Key        string `json:"key"`

	// Content information
	Type  string `json:"type"`  // "movie", "episode", "track"
	Title string `json:"title"` // Content title

	// Playback state
	ViewOffset int64 `json:"viewOffset"` // Current playback position (milliseconds)
	Duration   int64 `json:"duration"`   // Total duration (milliseconds)

	// Transcode session with buffer information
	TranscodeSession *PlexTranscodeSessionTimeline `json:"TranscodeSession,omitempty"`
}

// PlexTranscodeSessionTimeline represents transcode session timeline data
// CRITICAL: Contains maxOffsetAvailable and minOffsetAvailable for buffer health calculation
type PlexTranscodeSessionTimeline struct {
	// Buffer information (CRITICAL for buffer health)
	MaxOffsetAvailable float64 `json:"maxOffsetAvailable"` // Maximum buffered offset (milliseconds)
	MinOffsetAvailable float64 `json:"minOffsetAvailable"` // Minimum buffered offset (milliseconds)

	// Transcode progress
	Progress float64 `json:"progress"` // Transcode progress (0-100)
	Speed    float64 `json:"speed"`    // Transcode speed (e.g., 1.5 = 1.5x realtime)

	// Transcode state
	Throttled bool   `json:"throttled"` // Throttled due to system load
	Complete  bool   `json:"complete"`  // Transcode completed
	Key       string `json:"key"`       // Transcode session key
}

// PlexBufferHealth represents calculated buffer health for a session
//
// Buffer Health Calculation:
//   - Buffer Fill % = (maxOffsetAvailable - viewOffset) / (bitrate * 5 seconds) * 100
//   - Drain Rate = buffer consumed per second (calculated from previous poll)
//   - Health Status = "healthy" (>50%), "risky" (20-50%), "critical" (<20%)
//
// Thresholds (configurable):
//   - Critical: <20% buffer remaining (high risk of stalling)
//   - Risky: 20-50% buffer remaining (moderate risk)
//   - Healthy: >50% buffer remaining (low risk)
type PlexBufferHealth struct {
	// Session identification
	SessionKey string `json:"sessionKey"` // Plex session key
	Title      string `json:"title"`      // Content title

	// User and player information
	Username     string `json:"username,omitempty"`      // User watching this session
	PlayerDevice string `json:"player_device,omitempty"` // Device name

	// Buffer metrics
	BufferFillPercent float64 `json:"bufferFillPercent"` // Buffer fill percentage (0-100)
	BufferDrainRate   float64 `json:"bufferDrainRate"`   // Buffer drain rate (seconds/second, e.g., 1.2 = draining 20% faster than playback)
	BufferSeconds     float64 `json:"bufferSeconds"`     // Seconds of buffered content available

	// Health status
	HealthStatus string `json:"healthStatus"` // "healthy", "risky", "critical"
	RiskLevel    int    `json:"riskLevel"`    // 0 = healthy, 1 = risky, 2 = critical (for sorting)

	// Raw metrics (from Plex API)
	MaxOffsetAvailable float64 `json:"maxOffsetAvailable"` // Maximum buffered offset (milliseconds)
	ViewOffset         int64   `json:"viewOffset"`         // Current playback position (milliseconds)
	TranscodeSpeed     float64 `json:"transcodeSpeed"`     // Transcode speed (1.5x realtime)

	// Metadata
	Timestamp time.Time `json:"timestamp"` // When this health check was performed
	AlertSent bool      `json:"alertSent"` // Whether an alert has been sent (to avoid spam)
}

// ===================================================================================================
// Helper Methods for Buffer Health
// ===================================================================================================

// IsCritical returns true if buffer health is critical (<20% by default)
func (bh *PlexBufferHealth) IsCritical() bool {
	return bh.HealthStatus == "critical"
}

// IsRisky returns true if buffer health is risky (20-50% by default)
func (bh *PlexBufferHealth) IsRisky() bool {
	return bh.HealthStatus == "risky"
}

// IsHealthy returns true if buffer health is healthy (>50% by default)
func (bh *PlexBufferHealth) IsHealthy() bool {
	return bh.HealthStatus == "healthy"
}

// GetHealthColor returns color code for UI display
// Returns: "#ef4444" (red) for critical, "#f59e0b" (amber) for risky, "#10b981" (green) for healthy
func (bh *PlexBufferHealth) GetHealthColor() string {
	switch bh.HealthStatus {
	case "critical":
		return "#ef4444" // Tailwind red-500
	case "risky":
		return "#f59e0b" // Tailwind amber-500
	case "healthy":
		return "#10b981" // Tailwind green-500
	default:
		return "#6b7280" // Tailwind gray-500 (unknown)
	}
}

// GetHealthEmoji returns emoji indicator for UI display
func (bh *PlexBufferHealth) GetHealthEmoji() string {
	switch bh.HealthStatus {
	case "critical":
		return "🔴" // Red circle
	case "risky":
		return "🟡" // Yellow circle
	case "healthy":
		return "🟢" // Green circle
	default:
		return "⚪" // White circle (unknown)
	}
}

// GetBufferFillString returns formatted buffer fill percentage
// Returns: "45.2%", "78.0%", etc.
func (bh *PlexBufferHealth) GetBufferFillString() string {
	return fmt.Sprintf("%.1f%%", bh.BufferFillPercent)
}

// GetBufferSecondsString returns formatted buffer seconds
// Returns: "12.5s", "45.0s", etc.
func (bh *PlexBufferHealth) GetBufferSecondsString() string {
	return fmt.Sprintf("%.1fs", bh.BufferSeconds)
}

// GetDrainRateString returns formatted drain rate
// Returns: "1.2x" (draining 20% faster), "0.9x" (draining 10% slower), "1.0x" (steady)
func (bh *PlexBufferHealth) GetDrainRateString() string {
	return fmt.Sprintf("%.2fx", bh.BufferDrainRate)
}

// ShouldAlert returns true if an alert should be sent based on health status and alert history
// Alerts are sent once per session when:
//   - Health status changes to critical
//   - Health status changes to risky (optional, can be disabled)
//   - AlertSent flag prevents duplicate alerts for the same issue
func (bh *PlexBufferHealth) ShouldAlert() bool {
	if bh.AlertSent {
		return false // Already alerted for this session
	}
	return bh.IsCritical() || bh.IsRisky()
}

// GetPredictedBufferingSeconds returns estimated seconds until buffering event
// Based on current drain rate and buffer seconds available
//
// Returns:
//   - Positive number: seconds until buffering (e.g., 15.5 = will buffer in 15.5 seconds)
//   - Zero: currently buffering or buffer exhausted
//   - Negative: buffer is increasing (transcode catching up)
//
// Calculation:
//   - If drain rate > 1.0 (buffer draining): secondsUntilBuffering = bufferSeconds / (drainRate - 1.0)
//   - If drain rate <= 1.0 (buffer stable/growing): return -1 (no buffering predicted)
func (bh *PlexBufferHealth) GetPredictedBufferingSeconds() float64 {
	if bh.BufferSeconds <= 0 {
		return 0 // Already buffering or buffer exhausted
	}

	if bh.BufferDrainRate > 1.0 {
		// Buffer is draining faster than playback
		// Calculate how long until buffer is exhausted
		drainExcess := bh.BufferDrainRate - 1.0
		return bh.BufferSeconds / drainExcess
	}

	// Buffer is stable or growing (transcode keeping up or catching up)
	return -1
}

// GetAlertMessage returns user-friendly alert message for toast notifications
//
// Examples:
//   - "Critical: Session X buffering in 12s (buffer: 45%)"
//   - "Warning: Session X low buffer (buffer: 35%)"
func (bh *PlexBufferHealth) GetAlertMessage() string {
	predictedSeconds := bh.GetPredictedBufferingSeconds()

	if bh.IsCritical() {
		if predictedSeconds > 0 && predictedSeconds < 30 {
			return fmt.Sprintf("Critical: %s buffering in %.0fs (buffer: %.0f%%)",
				bh.Title, predictedSeconds, bh.BufferFillPercent)
		}
		return fmt.Sprintf("Critical: %s low buffer (%.0f%%)",
			bh.Title, bh.BufferFillPercent)
	}

	if bh.IsRisky() {
		return fmt.Sprintf("Warning: %s buffer dropping (%.0f%%)",
			bh.Title, bh.BufferFillPercent)
	}

	return "" // No alert needed
}

// CalculateBufferHealth computes buffer health metrics from raw timeline data
//
// Parameters:
//   - maxOffsetAvailable: Maximum buffered offset in milliseconds (from Plex API)
//   - viewOffset: Current playback position in milliseconds (from Plex API)
//   - transcodeSpeed: Transcode speed multiplier (e.g., 1.5 = 1.5x realtime)
//   - criticalThreshold: Critical health threshold (default: 20%)
//   - riskyThreshold: Risky health threshold (default: 50%)
//
// Returns:
//   - bufferFillPercent: Buffer fill percentage (0-100)
//   - bufferSeconds: Seconds of buffered content available
//   - healthStatus: "healthy", "risky", or "critical"
//
// Buffer Calculation:
//  1. Buffer available (ms) = maxOffsetAvailable - viewOffset
//  2. Buffer seconds = buffer available / 1000
//  3. Assume 5 Mbps average bitrate → ~5 seconds of content per 3.125 MB
//  4. Buffer fill % = (buffer seconds / assumed buffer capacity) * 100
//  5. For simplicity: Assume 30 seconds is 100% buffer capacity
//
// Note: This is a simplified calculation. For production, consider:
//   - Using actual stream bitrate from session metadata
//   - Tracking historical drain rate with rolling average
//   - Adjusting thresholds based on content type (4K vs SD)
func CalculateBufferHealth(
	sessionKey string,
	title string,
	maxOffsetAvailable float64,
	viewOffset int64,
	transcodeSpeed float64,
	previousBufferSeconds float64,
	criticalThreshold float64,
	riskyThreshold float64,
) *PlexBufferHealth {
	// Calculate buffer available in milliseconds
	bufferAvailableMs := maxOffsetAvailable - float64(viewOffset)
	if bufferAvailableMs < 0 {
		bufferAvailableMs = 0
	}

	// Convert to seconds
	bufferSeconds := bufferAvailableMs / 1000.0

	// Calculate buffer fill percentage
	// Assume 30 seconds is 100% buffer capacity (configurable in production)
	const maxBufferCapacity = 30.0 // seconds
	bufferFillPercent := (bufferSeconds / maxBufferCapacity) * 100.0
	if bufferFillPercent > 100 {
		bufferFillPercent = 100
	}

	// Calculate buffer drain rate (if we have previous data)
	// drainRate > 1.0 = buffer draining faster than playback
	// drainRate < 1.0 = buffer growing (transcode catching up)
	// drainRate = 1.0 = buffer stable
	bufferDrainRate := 1.0 // Default: assume steady state
	if previousBufferSeconds > 0 {
		// Calculate drain rate based on buffer change
		// Positive change = buffer growing, negative change = buffer draining
		bufferChange := bufferSeconds - previousBufferSeconds

		// Assume 5 second poll interval (configurable)
		const pollIntervalSeconds = 5.0

		// Drain rate = 1.0 - (buffer change / poll interval)
		// If buffer dropped 2 seconds in 5 second poll: drain rate = 1.0 - (-2/5) = 1.4x
		// If buffer grew 2 seconds in 5 second poll: drain rate = 1.0 - (2/5) = 0.6x
		bufferDrainRate = 1.0 - (bufferChange / pollIntervalSeconds)

		// Clamp drain rate to reasonable bounds [0.1, 5.0]
		if bufferDrainRate < 0.1 {
			bufferDrainRate = 0.1
		} else if bufferDrainRate > 5.0 {
			bufferDrainRate = 5.0
		}
	}

	// Determine health status based on buffer fill percentage
	var healthStatus string
	var riskLevel int
	if bufferFillPercent < criticalThreshold {
		healthStatus = "critical"
		riskLevel = 2
	} else if bufferFillPercent < riskyThreshold {
		healthStatus = "risky"
		riskLevel = 1
	} else {
		healthStatus = "healthy"
		riskLevel = 0
	}

	return &PlexBufferHealth{
		SessionKey:         sessionKey,
		Title:              title,
		BufferFillPercent:  bufferFillPercent,
		BufferDrainRate:    bufferDrainRate,
		BufferSeconds:      bufferSeconds,
		HealthStatus:       healthStatus,
		RiskLevel:          riskLevel,
		MaxOffsetAvailable: maxOffsetAvailable,
		ViewOffset:         viewOffset,
		TranscodeSpeed:     transcodeSpeed,
		Timestamp:          time.Now(),
		AlertSent:          false,
	}
}
