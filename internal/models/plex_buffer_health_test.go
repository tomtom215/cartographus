// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package models

import (
	"testing"
	"time"
)

// ===================================================================================================
// PlexBufferHealth Tests
// ===================================================================================================

func TestPlexBufferHealth_IsCritical(t *testing.T) {
	tests := []struct {
		name         string
		healthStatus string
		expected     bool
	}{
		{"critical status", "critical", true},
		{"risky status", "risky", false},
		{"healthy status", "healthy", false},
		{"empty status", "", false},
		{"unknown status", "unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bh := &PlexBufferHealth{HealthStatus: tt.healthStatus}
			if got := bh.IsCritical(); got != tt.expected {
				t.Errorf("IsCritical() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestPlexBufferHealth_IsRisky(t *testing.T) {
	tests := []struct {
		name         string
		healthStatus string
		expected     bool
	}{
		{"risky status", "risky", true},
		{"critical status", "critical", false},
		{"healthy status", "healthy", false},
		{"empty status", "", false},
		{"unknown status", "unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bh := &PlexBufferHealth{HealthStatus: tt.healthStatus}
			if got := bh.IsRisky(); got != tt.expected {
				t.Errorf("IsRisky() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestPlexBufferHealth_IsHealthy(t *testing.T) {
	tests := []struct {
		name         string
		healthStatus string
		expected     bool
	}{
		{"healthy status", "healthy", true},
		{"critical status", "critical", false},
		{"risky status", "risky", false},
		{"empty status", "", false},
		{"unknown status", "unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bh := &PlexBufferHealth{HealthStatus: tt.healthStatus}
			if got := bh.IsHealthy(); got != tt.expected {
				t.Errorf("IsHealthy() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestPlexBufferHealth_GetHealthColor(t *testing.T) {
	tests := []struct {
		name         string
		healthStatus string
		expected     string
	}{
		{"critical returns red", "critical", "#ef4444"},
		{"risky returns amber", "risky", "#f59e0b"},
		{"healthy returns green", "healthy", "#10b981"},
		{"empty returns gray", "", "#6b7280"},
		{"unknown returns gray", "unknown", "#6b7280"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bh := &PlexBufferHealth{HealthStatus: tt.healthStatus}
			if got := bh.GetHealthColor(); got != tt.expected {
				t.Errorf("GetHealthColor() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestPlexBufferHealth_GetHealthEmoji(t *testing.T) {
	tests := []struct {
		name         string
		healthStatus string
		expected     string
	}{
		{"critical returns red circle", "critical", "🔴"},
		{"risky returns yellow circle", "risky", "🟡"},
		{"healthy returns green circle", "healthy", "🟢"},
		{"empty returns white circle", "", "⚪"},
		{"unknown returns white circle", "unknown", "⚪"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bh := &PlexBufferHealth{HealthStatus: tt.healthStatus}
			if got := bh.GetHealthEmoji(); got != tt.expected {
				t.Errorf("GetHealthEmoji() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestPlexBufferHealth_GetBufferFillString(t *testing.T) {
	tests := []struct {
		name              string
		bufferFillPercent float64
		expected          string
	}{
		{"zero percent", 0.0, "0.0%"},
		{"50 percent", 50.0, "50.0%"},
		{"100 percent", 100.0, "100.0%"},
		{"fractional percent", 45.5, "45.5%"},
		{"high precision", 33.333333, "33.3%"},
		{"negative value", -5.0, "-5.0%"}, // Edge case
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bh := &PlexBufferHealth{BufferFillPercent: tt.bufferFillPercent}
			if got := bh.GetBufferFillString(); got != tt.expected {
				t.Errorf("GetBufferFillString() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestPlexBufferHealth_GetBufferSecondsString(t *testing.T) {
	tests := []struct {
		name          string
		bufferSeconds float64
		expected      string
	}{
		{"zero seconds", 0.0, "0.0s"},
		{"10 seconds", 10.0, "10.0s"},
		{"fractional seconds", 12.5, "12.5s"},
		{"high precision", 45.6789, "45.7s"},
		{"negative value", -2.0, "-2.0s"}, // Edge case
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bh := &PlexBufferHealth{BufferSeconds: tt.bufferSeconds}
			if got := bh.GetBufferSecondsString(); got != tt.expected {
				t.Errorf("GetBufferSecondsString() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestPlexBufferHealth_GetDrainRateString(t *testing.T) {
	tests := []struct {
		name            string
		bufferDrainRate float64
		expected        string
	}{
		{"1x drain rate", 1.0, "1.00x"},
		{"1.2x drain rate", 1.2, "1.20x"},
		{"0.9x drain rate (buffer growing)", 0.9, "0.90x"},
		{"high precision", 1.234567, "1.23x"},
		{"zero drain rate", 0.0, "0.00x"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bh := &PlexBufferHealth{BufferDrainRate: tt.bufferDrainRate}
			if got := bh.GetDrainRateString(); got != tt.expected {
				t.Errorf("GetDrainRateString() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestPlexBufferHealth_ShouldAlert(t *testing.T) {
	tests := []struct {
		name         string
		healthStatus string
		alertSent    bool
		expected     bool
	}{
		{"critical not sent", "critical", false, true},
		{"critical already sent", "critical", true, false},
		{"risky not sent", "risky", false, true},
		{"risky already sent", "risky", true, false},
		{"healthy not sent", "healthy", false, false},
		{"healthy already sent", "healthy", true, false},
		{"unknown not sent", "unknown", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bh := &PlexBufferHealth{
				HealthStatus: tt.healthStatus,
				AlertSent:    tt.alertSent,
			}
			if got := bh.ShouldAlert(); got != tt.expected {
				t.Errorf("ShouldAlert() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestPlexBufferHealth_GetPredictedBufferingSeconds(t *testing.T) {
	tests := []struct {
		name            string
		bufferSeconds   float64
		bufferDrainRate float64
		expected        float64
	}{
		{"zero buffer", 0.0, 1.5, 0},
		{"negative buffer", -5.0, 1.5, 0},
		{"stable drain rate", 10.0, 1.0, -1},
		{"buffer growing", 10.0, 0.8, -1},
		{"buffer draining", 10.0, 1.5, 20.0},        // 10 / (1.5 - 1.0) = 20
		{"fast draining", 10.0, 2.0, 10.0},          // 10 / (2.0 - 1.0) = 10
		{"slow draining", 30.0, 1.2, 150.0},         // 30 / (1.2 - 1.0) = 150
		{"very fast draining", 5.0, 5.0, 1.25},      // 5 / (5.0 - 1.0) = 1.25
		{"barely draining", 10.0, 1.01, 1000.0},     // 10 / (1.01 - 1.0) = 1000
		{"exactly at threshold", 15.0, 1.0, -1},     // At threshold = stable
		{"fractional drain rate", 20.0, 1.25, 80.0}, // 20 / 0.25 = 80
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bh := &PlexBufferHealth{
				BufferSeconds:   tt.bufferSeconds,
				BufferDrainRate: tt.bufferDrainRate,
			}
			got := bh.GetPredictedBufferingSeconds()
			// Use approximate comparison for floating point
			if (tt.expected == -1 && got != -1) || (tt.expected == 0 && got != 0) {
				t.Errorf("GetPredictedBufferingSeconds() = %v, want %v", got, tt.expected)
			} else if tt.expected > 0 {
				diff := got - tt.expected
				if diff < -0.01 || diff > 0.01 {
					t.Errorf("GetPredictedBufferingSeconds() = %v, want %v (diff: %v)", got, tt.expected, diff)
				}
			}
		})
	}
}

func TestPlexBufferHealth_GetAlertMessage(t *testing.T) {
	tests := []struct {
		name              string
		healthStatus      string
		title             string
		bufferFillPercent float64
		bufferSeconds     float64
		bufferDrainRate   float64
		expectEmpty       bool
		contains          []string
	}{
		{
			name:              "critical with predicted buffering",
			healthStatus:      "critical",
			title:             "Test Movie",
			bufferFillPercent: 15.0,
			bufferSeconds:     5.0,
			bufferDrainRate:   1.5, // 5 / 0.5 = 10 seconds until buffering
			expectEmpty:       false,
			contains:          []string{"Critical:", "Test Movie", "buffering in", "15%"},
		},
		{
			name:              "critical without predicted buffering soon",
			healthStatus:      "critical",
			title:             "Test Movie",
			bufferFillPercent: 15.0,
			bufferSeconds:     60.0,
			bufferDrainRate:   1.1, // 60 / 0.1 = 600 seconds (> 30)
			expectEmpty:       false,
			contains:          []string{"Critical:", "Test Movie", "low buffer", "15%"},
		},
		{
			name:              "risky status",
			healthStatus:      "risky",
			title:             "Show Episode",
			bufferFillPercent: 35.0,
			bufferSeconds:     15.0,
			bufferDrainRate:   1.2,
			expectEmpty:       false,
			contains:          []string{"Warning:", "Show Episode", "buffer dropping", "35%"},
		},
		{
			name:              "healthy status returns empty",
			healthStatus:      "healthy",
			title:             "Test Content",
			bufferFillPercent: 75.0,
			bufferSeconds:     30.0,
			bufferDrainRate:   0.9,
			expectEmpty:       true,
			contains:          nil,
		},
		{
			name:              "unknown status returns empty",
			healthStatus:      "unknown",
			title:             "Test Content",
			bufferFillPercent: 50.0,
			bufferSeconds:     20.0,
			bufferDrainRate:   1.0,
			expectEmpty:       true,
			contains:          nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bh := &PlexBufferHealth{
				HealthStatus:      tt.healthStatus,
				Title:             tt.title,
				BufferFillPercent: tt.bufferFillPercent,
				BufferSeconds:     tt.bufferSeconds,
				BufferDrainRate:   tt.bufferDrainRate,
			}
			got := bh.GetAlertMessage()
			if tt.expectEmpty {
				if got != "" {
					t.Errorf("GetAlertMessage() = %q, want empty string", got)
				}
			} else {
				if got == "" {
					t.Error("GetAlertMessage() returned empty string, expected non-empty")
				}
				for _, substr := range tt.contains {
					if !containsString(got, substr) {
						t.Errorf("GetAlertMessage() = %q, expected to contain %q", got, substr)
					}
				}
			}
		})
	}
}

// Helper function to check if string contains substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && (containsAt(s, substr, 0) || containsString(s[1:], substr)))
}

func containsAt(s, substr string, pos int) bool {
	if pos+len(substr) > len(s) {
		return false
	}
	for i := 0; i < len(substr); i++ {
		if s[pos+i] != substr[i] {
			return false
		}
	}
	return true
}

// ===================================================================================================
// CalculateBufferHealth Tests
// ===================================================================================================

func TestCalculateBufferHealth(t *testing.T) {
	tests := []struct {
		name                  string
		sessionKey            string
		title                 string
		maxOffsetAvailable    float64
		viewOffset            int64
		transcodeSpeed        float64
		previousBufferSeconds float64
		criticalThreshold     float64
		riskyThreshold        float64
		expectedHealthStatus  string
		expectedRiskLevel     int
		minBufferFill         float64
		maxBufferFill         float64
	}{
		{
			name:                  "healthy buffer",
			sessionKey:            "session1",
			title:                 "Test Movie",
			maxOffsetAvailable:    60000.0, // 60 seconds buffered ahead
			viewOffset:            30000,   // 30 seconds into playback
			transcodeSpeed:        1.5,
			previousBufferSeconds: 0,
			criticalThreshold:     20.0,
			riskyThreshold:        50.0,
			expectedHealthStatus:  "healthy",
			expectedRiskLevel:     0,
			minBufferFill:         90.0, // 30 seconds buffer / 30 max = 100%
			maxBufferFill:         100.0,
		},
		{
			name:                  "risky buffer",
			sessionKey:            "session2",
			title:                 "Test Show",
			maxOffsetAvailable:    40000.0, // 40 seconds
			viewOffset:            30000,   // 30 seconds into playback
			transcodeSpeed:        1.0,
			previousBufferSeconds: 0,
			criticalThreshold:     20.0,
			riskyThreshold:        50.0,
			expectedHealthStatus:  "risky",
			expectedRiskLevel:     1,
			minBufferFill:         30.0, // 10 seconds buffer / 30 max = 33.3%
			maxBufferFill:         40.0,
		},
		{
			name:                  "critical buffer",
			sessionKey:            "session3",
			title:                 "Test Episode",
			maxOffsetAvailable:    32000.0, // 32 seconds
			viewOffset:            30000,   // 30 seconds into playback
			transcodeSpeed:        0.5,
			previousBufferSeconds: 0,
			criticalThreshold:     20.0,
			riskyThreshold:        50.0,
			expectedHealthStatus:  "critical",
			expectedRiskLevel:     2,
			minBufferFill:         0.0, // 2 seconds buffer / 30 max = 6.7%
			maxBufferFill:         10.0,
		},
		{
			name:                  "negative buffer (view ahead of transcode)",
			sessionKey:            "session4",
			title:                 "Test Content",
			maxOffsetAvailable:    25000.0, // 25 seconds
			viewOffset:            30000,   // 30 seconds into playback (ahead of buffer)
			transcodeSpeed:        0.5,
			previousBufferSeconds: 0,
			criticalThreshold:     20.0,
			riskyThreshold:        50.0,
			expectedHealthStatus:  "critical",
			expectedRiskLevel:     2,
			minBufferFill:         0.0,
			maxBufferFill:         1.0,
		},
		{
			name:                  "exact threshold - risky boundary",
			sessionKey:            "session5",
			title:                 "Test Movie 2",
			maxOffsetAvailable:    45000.0, // 45 seconds
			viewOffset:            30000,   // 15 seconds buffer = 50% = boundary of risky/healthy
			transcodeSpeed:        1.0,
			previousBufferSeconds: 0,
			criticalThreshold:     20.0,
			riskyThreshold:        50.0,
			expectedHealthStatus:  "healthy", // 50% is the threshold, so >= 50 is healthy
			expectedRiskLevel:     0,
			minBufferFill:         49.0,
			maxBufferFill:         51.0,
		},
		{
			name:                  "custom thresholds",
			sessionKey:            "session6",
			title:                 "Test Content",
			maxOffsetAvailable:    35000.0, // 35 seconds
			viewOffset:            30000,   // 5 seconds buffer
			transcodeSpeed:        1.0,
			previousBufferSeconds: 0,
			criticalThreshold:     10.0, // More lenient
			riskyThreshold:        30.0,
			expectedHealthStatus:  "risky",
			expectedRiskLevel:     1,
			minBufferFill:         15.0,
			maxBufferFill:         20.0,
		},
		{
			name:                  "over 100% buffer fill",
			sessionKey:            "session7",
			title:                 "Full Buffer",
			maxOffsetAvailable:    100000.0, // 100 seconds
			viewOffset:            0,        // Start of playback
			transcodeSpeed:        2.0,
			previousBufferSeconds: 0,
			criticalThreshold:     20.0,
			riskyThreshold:        50.0,
			expectedHealthStatus:  "healthy",
			expectedRiskLevel:     0,
			minBufferFill:         100.0, // Capped at 100%
			maxBufferFill:         100.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateBufferHealth(
				tt.sessionKey,
				tt.title,
				tt.maxOffsetAvailable,
				tt.viewOffset,
				tt.transcodeSpeed,
				tt.previousBufferSeconds,
				tt.criticalThreshold,
				tt.riskyThreshold,
			)

			if got == nil {
				t.Fatal("CalculateBufferHealth() returned nil")
			}

			if got.SessionKey != tt.sessionKey {
				t.Errorf("SessionKey = %q, want %q", got.SessionKey, tt.sessionKey)
			}

			if got.Title != tt.title {
				t.Errorf("Title = %q, want %q", got.Title, tt.title)
			}

			if got.HealthStatus != tt.expectedHealthStatus {
				t.Errorf("HealthStatus = %q, want %q", got.HealthStatus, tt.expectedHealthStatus)
			}

			if got.RiskLevel != tt.expectedRiskLevel {
				t.Errorf("RiskLevel = %d, want %d", got.RiskLevel, tt.expectedRiskLevel)
			}

			if got.BufferFillPercent < tt.minBufferFill || got.BufferFillPercent > tt.maxBufferFill {
				t.Errorf("BufferFillPercent = %v, want between %v and %v", got.BufferFillPercent, tt.minBufferFill, tt.maxBufferFill)
			}

			if got.TranscodeSpeed != tt.transcodeSpeed {
				t.Errorf("TranscodeSpeed = %v, want %v", got.TranscodeSpeed, tt.transcodeSpeed)
			}

			if got.AlertSent != false {
				t.Errorf("AlertSent = %v, want false", got.AlertSent)
			}

			if got.Timestamp.IsZero() {
				t.Error("Timestamp should not be zero")
			}
		})
	}
}

func TestCalculateBufferHealth_DrainRate(t *testing.T) {
	// Test drain rate calculation with previous buffer seconds
	tests := []struct {
		name                  string
		previousBufferSeconds float64
		currentBufferSeconds  float64 // Calculated from maxOffset - viewOffset
		minDrainRate          float64
		maxDrainRate          float64
	}{
		{
			name:                  "no previous data",
			previousBufferSeconds: 0,
			currentBufferSeconds:  15.0, // maxOffset=45000, viewOffset=30000
			minDrainRate:          0.9,  // Default drain rate is 1.0
			maxDrainRate:          1.1,
		},
		{
			name:                  "buffer draining",
			previousBufferSeconds: 20.0,
			currentBufferSeconds:  15.0, // Lost 5 seconds in 5 second poll
			minDrainRate:          1.9,  // 1.0 - (-5/5) = 2.0
			maxDrainRate:          2.1,
		},
		{
			name:                  "buffer growing",
			previousBufferSeconds: 10.0,
			currentBufferSeconds:  15.0, // Gained 5 seconds in 5 second poll
			minDrainRate:          0.0,  // 1.0 - (5/5) = 0.0 but clamped to 0.1
			maxDrainRate:          0.2,
		},
		{
			name:                  "buffer stable",
			previousBufferSeconds: 15.0,
			currentBufferSeconds:  15.0, // No change
			minDrainRate:          0.9,  // 1.0 - (0/5) = 1.0
			maxDrainRate:          1.1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Calculate buffer in ms based on desired seconds
			maxOffset := float64(30000) + (tt.currentBufferSeconds * 1000)

			got := CalculateBufferHealth(
				"session",
				"Test",
				maxOffset,
				30000, // viewOffset
				1.5,
				tt.previousBufferSeconds,
				20.0,
				50.0,
			)

			if got.BufferDrainRate < tt.minDrainRate || got.BufferDrainRate > tt.maxDrainRate {
				t.Errorf("BufferDrainRate = %v, want between %v and %v", got.BufferDrainRate, tt.minDrainRate, tt.maxDrainRate)
			}
		})
	}
}

func TestCalculateBufferHealth_FieldsSet(t *testing.T) {
	// Verify all fields are properly set
	got := CalculateBufferHealth(
		"test-session",
		"Test Title",
		45000.0,
		30000,
		1.5,
		0,
		20.0,
		50.0,
	)

	// Check all fields are set
	if got.SessionKey != "test-session" {
		t.Errorf("SessionKey not set correctly")
	}
	if got.Title != "Test Title" {
		t.Errorf("Title not set correctly")
	}
	if got.MaxOffsetAvailable != 45000.0 {
		t.Errorf("MaxOffsetAvailable not set correctly")
	}
	if got.ViewOffset != 30000 {
		t.Errorf("ViewOffset not set correctly")
	}
	if got.TranscodeSpeed != 1.5 {
		t.Errorf("TranscodeSpeed not set correctly")
	}
	if got.Timestamp.Before(time.Now().Add(-time.Second)) {
		t.Errorf("Timestamp should be recent")
	}
	if got.AlertSent != false {
		t.Errorf("AlertSent should be false")
	}
	if got.BufferSeconds < 0 {
		t.Errorf("BufferSeconds should not be negative")
	}
	if got.BufferFillPercent < 0 || got.BufferFillPercent > 100 {
		t.Errorf("BufferFillPercent should be between 0 and 100")
	}
	if got.BufferDrainRate < 0.1 || got.BufferDrainRate > 5.0 {
		t.Errorf("BufferDrainRate should be clamped between 0.1 and 5.0")
	}
	if got.RiskLevel < 0 || got.RiskLevel > 2 {
		t.Errorf("RiskLevel should be 0, 1, or 2")
	}
	validStatuses := map[string]bool{"healthy": true, "risky": true, "critical": true}
	if !validStatuses[got.HealthStatus] {
		t.Errorf("HealthStatus should be healthy, risky, or critical")
	}
}
