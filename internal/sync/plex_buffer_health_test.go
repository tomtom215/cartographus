// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package sync

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/goccy/go-json"

	"github.com/tomtom215/cartographus/internal/models"
)

// assertBufferHealth verifies common buffer health properties
func assertBufferHealth(t *testing.T, health *models.PlexBufferHealth, expectedStatus string, expectedRisk int) {
	t.Helper()
	if health.HealthStatus != expectedStatus {
		t.Errorf("Expected health status '%s', got '%s'", expectedStatus, health.HealthStatus)
	}
	if health.RiskLevel != expectedRisk {
		t.Errorf("Expected risk level %d, got %d", expectedRisk, health.RiskLevel)
	}
}

// TestGetSessionTimeline_Success tests successful session timeline retrieval with buffer data
func TestGetSessionTimeline_Success(t *testing.T) {
	// Create mock HTTP server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.URL.Path != "/status/sessions" {
			t.Errorf("Expected path /status/sessions, got %s", r.URL.Path)
		}
		if r.Header.Get("X-Plex-Token") != "test-token" {
			t.Errorf("Expected X-Plex-Token header, got %s", r.Header.Get("X-Plex-Token"))
		}
		if r.Header.Get("Accept") != "application/json" {
			t.Errorf("Expected Accept: application/json header")
		}

		// Return mock response with session that has buffer data
		response := models.PlexSessionsResponse{
			MediaContainer: models.PlexSessionsContainer{
				Size: 2,
				Metadata: []models.PlexSession{
					{
						SessionKey: "test-session-123",
						Key:        "/library/metadata/12345",
						Type:       "movie",
						Title:      "Test Movie 4K",
						ViewOffset: 120000,  // 120 seconds (2 minutes) into playback
						Duration:   7200000, // 2 hour movie
						TranscodeSession: &models.PlexTranscodeSession{
							MaxOffsetAvailable: 150000, // 150 seconds buffered (30 seconds ahead)
							MinOffsetAvailable: 0,
							Progress:           45.5,
							Speed:              1.2, // Transcode speed 1.2x realtime
							Throttled:          false,
							Complete:           false,
							Key:                "transcode-key-123",
						},
					},
					{
						SessionKey: "other-session-456",
						Title:      "Different Movie",
						ViewOffset: 60000,
						Duration:   5400000,
					},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	// Create client and fetch timeline
	client := NewPlexClient(mockServer.URL, "test-token")
	timeline, err := client.GetSessionTimeline(context.Background(), "test-session-123")

	// Assertions
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if timeline == nil {
		t.Fatal("Expected timeline response, got nil")
	}
	if timeline.MediaContainer.Size != 1 {
		t.Fatalf("Expected 1 matching session, got %d", timeline.MediaContainer.Size)
	}

	// Verify timeline metadata
	session := timeline.MediaContainer.Metadata[0]
	if session.SessionKey != "test-session-123" {
		t.Errorf("Expected session key test-session-123, got %s", session.SessionKey)
	}
	if session.Title != "Test Movie 4K" {
		t.Errorf("Expected title 'Test Movie 4K', got %s", session.Title)
	}
	if session.ViewOffset != 120000 {
		t.Errorf("Expected viewOffset 120000, got %d", session.ViewOffset)
	}
	if session.Duration != 7200000 {
		t.Errorf("Expected duration 7200000, got %d", session.Duration)
	}

	// Verify transcode session with buffer data
	if session.TranscodeSession == nil {
		t.Fatal("Expected TranscodeSession, got nil")
	}
	ts := session.TranscodeSession
	if ts.MaxOffsetAvailable != 150000 {
		t.Errorf("Expected maxOffsetAvailable 150000, got %f", ts.MaxOffsetAvailable)
	}
	if ts.MinOffsetAvailable != 0 {
		t.Errorf("Expected minOffsetAvailable 0, got %f", ts.MinOffsetAvailable)
	}
	if ts.Progress != 45.5 {
		t.Errorf("Expected progress 45.5, got %f", ts.Progress)
	}
	if ts.Speed != 1.2 {
		t.Errorf("Expected speed 1.2, got %f", ts.Speed)
	}

	// Calculate buffer available for validation
	bufferAvailable := ts.MaxOffsetAvailable - float64(session.ViewOffset)
	expectedBuffer := 150000.0 - 120000.0 // 30 seconds ahead
	if bufferAvailable != expectedBuffer {
		t.Errorf("Expected buffer available %f, got %f", expectedBuffer, bufferAvailable)
	}
}

// TestGetSessionTimeline_SessionNotFound tests when requested session doesn't exist
func TestGetSessionTimeline_SessionNotFound(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return sessions without the requested session key
		response := models.PlexSessionsResponse{
			MediaContainer: models.PlexSessionsContainer{
				Size: 1,
				Metadata: []models.PlexSession{
					{
						SessionKey: "other-session-999",
						Title:      "Some Movie",
						ViewOffset: 60000,
						Duration:   5400000,
					},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	client := NewPlexClient(mockServer.URL, "test-token")
	timeline, err := client.GetSessionTimeline(context.Background(), "nonexistent-session")

	// Should succeed but return empty metadata
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if timeline == nil {
		t.Fatal("Expected timeline response, got nil")
	}
	if timeline.MediaContainer.Size != 0 {
		t.Errorf("Expected 0 matching sessions, got %d", timeline.MediaContainer.Size)
	}
	if len(timeline.MediaContainer.Metadata) != 0 {
		t.Errorf("Expected empty metadata array, got %d items", len(timeline.MediaContainer.Metadata))
	}
}

// TestGetSessionTimeline_EmptySessions tests empty sessions response
func TestGetSessionTimeline_EmptySessions(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return empty sessions
		response := models.PlexSessionsResponse{
			MediaContainer: models.PlexSessionsContainer{
				Size:     0,
				Metadata: nil, // nil metadata array
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	client := NewPlexClient(mockServer.URL, "test-token")
	timeline, err := client.GetSessionTimeline(context.Background(), "any-session")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if timeline.MediaContainer.Size != 0 {
		t.Errorf("Expected 0 sessions, got %d", timeline.MediaContainer.Size)
	}
	if len(timeline.MediaContainer.Metadata) != 0 {
		t.Errorf("Expected empty metadata, got %d items", len(timeline.MediaContainer.Metadata))
	}
}

// TestGetSessionTimeline_NoTranscodeSession tests session without transcode data
func TestGetSessionTimeline_NoTranscodeSession(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return session without TranscodeSession (direct play)
		response := models.PlexSessionsResponse{
			MediaContainer: models.PlexSessionsContainer{
				Size: 1,
				Metadata: []models.PlexSession{
					{
						SessionKey:       "direct-play-session",
						Title:            "Direct Play Movie",
						ViewOffset:       30000,
						Duration:         5400000,
						TranscodeSession: nil, // No transcode session for direct play
					},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	client := NewPlexClient(mockServer.URL, "test-token")
	timeline, err := client.GetSessionTimeline(context.Background(), "direct-play-session")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if timeline.MediaContainer.Size != 1 {
		t.Fatalf("Expected 1 session, got %d", timeline.MediaContainer.Size)
	}

	session := timeline.MediaContainer.Metadata[0]
	if session.TranscodeSession != nil {
		t.Error("Expected nil TranscodeSession for direct play, got non-nil")
	}
}

// TestGetSessionTimeline_HTTPError tests handling of HTTP errors
func TestGetSessionTimeline_HTTPError(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantErr    bool
	}{
		{"404 Not Found", http.StatusNotFound, true},
		{"500 Server Error", http.StatusInternalServerError, true},
		{"401 Unauthorized", http.StatusUnauthorized, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "error", tt.statusCode)
			}))
			defer mockServer.Close()

			client := NewPlexClient(mockServer.URL, "test-token")
			_, err := client.GetSessionTimeline(context.Background(), "any-session")

			if (err != nil) != tt.wantErr {
				t.Errorf("Expected error = %v, got %v", tt.wantErr, err)
			}
		})
	}
}

// TestCalculateBufferHealth_StatusLevels tests critical, risky, and healthy buffer health detection
func TestCalculateBufferHealth_StatusLevels(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		bufferAheadMs  float64 // Milliseconds ahead of playback
		transcodeSpeed float64
		expectedStatus string
		expectedRisk   int
		expectedColor  string
		expectedEmoji  string
		isCritical     bool
		isRisky        bool
		isHealthy      bool
	}{
		{
			name:           "critical - 5 seconds ahead (16.7%)",
			bufferAheadMs:  5000,
			transcodeSpeed: 1.0,
			expectedStatus: "critical", expectedRisk: 2,
			expectedColor: "#ef4444", expectedEmoji: "🔴",
			isCritical: true, isRisky: false, isHealthy: false,
		},
		{
			name:           "risky - 10 seconds ahead (33.3%)",
			bufferAheadMs:  10000,
			transcodeSpeed: 1.0,
			expectedStatus: "risky", expectedRisk: 1,
			expectedColor: "#f59e0b", expectedEmoji: "🟡",
			isCritical: false, isRisky: true, isHealthy: false,
		},
		{
			name:           "healthy - 20 seconds ahead (66.7%)",
			bufferAheadMs:  20000,
			transcodeSpeed: 1.5,
			expectedStatus: "healthy", expectedRisk: 0,
			expectedColor: "#10b981", expectedEmoji: "🟢",
			isCritical: false, isRisky: false, isHealthy: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			viewOffset := int64(100000) // 100 seconds into playback
			maxOffsetAvailable := float64(viewOffset) + tt.bufferAheadMs

			health := models.CalculateBufferHealth(
				"test-session", "Test Movie",
				maxOffsetAvailable, viewOffset,
				tt.transcodeSpeed, 0.0, // previous buffer
				20.0, 50.0, // thresholds
			)

			// Verify status and risk level
			assertBufferHealth(t, health, tt.expectedStatus, tt.expectedRisk)

			// Verify buffer seconds
			expectedBuffer := tt.bufferAheadMs / 1000
			if health.BufferSeconds != expectedBuffer {
				t.Errorf("Expected bufferSeconds %.1f, got %.1f", expectedBuffer, health.BufferSeconds)
			}

			// Verify helper methods
			if health.IsCritical() != tt.isCritical {
				t.Errorf("IsCritical() = %v, want %v", health.IsCritical(), tt.isCritical)
			}
			if health.IsRisky() != tt.isRisky {
				t.Errorf("IsRisky() = %v, want %v", health.IsRisky(), tt.isRisky)
			}
			if health.IsHealthy() != tt.isHealthy {
				t.Errorf("IsHealthy() = %v, want %v", health.IsHealthy(), tt.isHealthy)
			}

			// Verify visual indicators
			if health.GetHealthColor() != tt.expectedColor {
				t.Errorf("GetHealthColor() = %s, want %s", health.GetHealthColor(), tt.expectedColor)
			}
			if health.GetHealthEmoji() != tt.expectedEmoji {
				t.Errorf("GetHealthEmoji() = %s, want %s", health.GetHealthEmoji(), tt.expectedEmoji)
			}
		})
	}
}

// TestCalculateBufferHealth_DrainRateCalculation tests buffer drain rate with previous data
func TestCalculateBufferHealth_DrainRateCalculation(t *testing.T) {
	tests := []struct {
		name                  string
		currentBufferSeconds  float64
		previousBufferSeconds float64
		expectedDrainRate     float64
		description           string
	}{
		{
			name:                  "Buffer draining fast",
			currentBufferSeconds:  8.0,  // Down from 10 seconds
			previousBufferSeconds: 10.0, // Was 10 seconds
			expectedDrainRate:     1.4,  // 1.0 - ((8-10)/5) = 1.0 - (-0.4) = 1.4x (draining 40% faster)
			description:           "Buffer dropped 2 seconds in 5 second poll",
		},
		{
			name:                  "Buffer stable",
			currentBufferSeconds:  10.0,
			previousBufferSeconds: 10.0,
			expectedDrainRate:     1.0, // No change = 1.0x (steady)
			description:           "Buffer unchanged",
		},
		{
			name:                  "Buffer growing",
			currentBufferSeconds:  12.0,
			previousBufferSeconds: 10.0,
			expectedDrainRate:     0.6, // 1.0 - ((12-10)/5) = 1.0 - 0.4 = 0.6x (growing)
			description:           "Buffer increased 2 seconds (transcode catching up)",
		},
		{
			name:                  "No previous data",
			currentBufferSeconds:  15.0,
			previousBufferSeconds: 0.0, // First poll
			expectedDrainRate:     1.0, // Default to 1.0x
			description:           "First poll, no previous data to compare",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sessionKey := "test-session"
			title := "Test Movie"
			viewOffset := int64(100000)
			// Calculate maxOffsetAvailable from desired buffer seconds
			maxOffsetAvailable := float64(viewOffset) + (tt.currentBufferSeconds * 1000)

			health := models.CalculateBufferHealth(
				sessionKey,
				title,
				maxOffsetAvailable,
				viewOffset,
				1.0, // transcode speed
				tt.previousBufferSeconds,
				20.0, // critical threshold
				50.0, // risky threshold
			)

			// Verify buffer seconds calculated correctly
			if health.BufferSeconds != tt.currentBufferSeconds {
				t.Errorf("Expected bufferSeconds %.1f, got %.1f", tt.currentBufferSeconds, health.BufferSeconds)
			}

			// Verify drain rate (allow small floating point tolerance)
			tolerance := 0.01
			if health.BufferDrainRate < tt.expectedDrainRate-tolerance || health.BufferDrainRate > tt.expectedDrainRate+tolerance {
				t.Errorf("%s: Expected drain rate %.2fx, got %.2fx", tt.description, tt.expectedDrainRate, health.BufferDrainRate)
			}
		})
	}
}

// TestPlexBufferHealth_GetPredictedBufferingSeconds tests buffering prediction algorithm
func TestPlexBufferHealth_GetPredictedBufferingSeconds(t *testing.T) {
	tests := []struct {
		name               string
		bufferSeconds      float64
		bufferDrainRate    float64
		expectedPrediction float64
		description        string
	}{
		{
			name:               "Fast drain - 10s buffer, 1.5x drain",
			bufferSeconds:      10.0,
			bufferDrainRate:    1.5,  // Draining 50% faster
			expectedPrediction: 20.0, // 10 / (1.5 - 1.0) = 10 / 0.5 = 20 seconds until buffering
			description:        "10 second buffer draining at 1.5x = 20 seconds until stall",
		},
		{
			name:               "Critical drain - 5s buffer, 1.2x drain",
			bufferSeconds:      5.0,
			bufferDrainRate:    1.2,  // Draining 20% faster
			expectedPrediction: 25.0, // 5 / (1.2 - 1.0) = 5 / 0.2 = 25 seconds
			description:        "5 second buffer draining at 1.2x = 25 seconds until stall",
		},
		{
			name:               "Steady drain - 15s buffer, 1.1x drain",
			bufferSeconds:      15.0,
			bufferDrainRate:    1.1,   // Draining 10% faster
			expectedPrediction: 150.0, // 15 / 0.1 = 150 seconds
			description:        "15 second buffer draining slowly",
		},
		{
			name:               "No drain - buffer stable",
			bufferSeconds:      10.0,
			bufferDrainRate:    1.0,  // Steady state
			expectedPrediction: -1.0, // No buffering predicted
			description:        "Buffer stable at 1.0x",
		},
		{
			name:               "Buffer growing",
			bufferSeconds:      10.0,
			bufferDrainRate:    0.8,  // Growing (transcode catching up)
			expectedPrediction: -1.0, // No buffering predicted
			description:        "Buffer growing (transcode faster than playback)",
		},
		{
			name:               "Already buffering",
			bufferSeconds:      0.0, // No buffer
			bufferDrainRate:    1.5,
			expectedPrediction: 0.0, // Currently buffering
			description:        "Buffer exhausted",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			health := &models.PlexBufferHealth{
				BufferSeconds:   tt.bufferSeconds,
				BufferDrainRate: tt.bufferDrainRate,
			}

			predicted := health.GetPredictedBufferingSeconds()

			// Verify prediction (allow small floating point tolerance for division)
			tolerance := 0.01
			if predicted >= 0 && tt.expectedPrediction >= 0 {
				if predicted < tt.expectedPrediction-tolerance || predicted > tt.expectedPrediction+tolerance {
					t.Errorf("%s: Expected prediction %.2fs, got %.2fs", tt.description, tt.expectedPrediction, predicted)
				}
			} else if predicted != tt.expectedPrediction {
				t.Errorf("%s: Expected prediction %.2fs, got %.2fs", tt.description, tt.expectedPrediction, predicted)
			}
		})
	}
}

// TestPlexBufferHealth_GetAlertMessage tests alert message generation
func TestPlexBufferHealth_GetAlertMessage(t *testing.T) {
	tests := []struct {
		name             string
		health           models.PlexBufferHealth
		expectedContains []string
	}{
		{
			"Critical with imminent buffering",
			models.PlexBufferHealth{Title: "Test Movie", HealthStatus: "critical", BufferSeconds: 10, BufferDrainRate: 1.5, BufferFillPercent: 15},
			[]string{"Critical", "Test Movie", "buffering in", "15%"},
		},
		{
			"Critical without imminent buffering",
			models.PlexBufferHealth{Title: "Another Movie", HealthStatus: "critical", BufferSeconds: 5, BufferDrainRate: 1.05, BufferFillPercent: 18},
			[]string{"Critical", "Another Movie", "low buffer", "18%"},
		},
		{
			"Risky buffer",
			models.PlexBufferHealth{Title: "Some Show", HealthStatus: "risky", BufferSeconds: 12, BufferDrainRate: 1.2, BufferFillPercent: 35},
			[]string{"Warning", "Some Show", "buffer dropping", "35%"},
		},
		{
			"Healthy buffer (no alert)",
			models.PlexBufferHealth{Title: "Happy Movie", HealthStatus: "healthy", BufferSeconds: 25, BufferDrainRate: 0.9, BufferFillPercent: 80},
			nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			message := tt.health.GetAlertMessage()
			if tt.expectedContains == nil {
				if message != "" {
					t.Errorf("Expected empty message for healthy status, got '%s'", message)
				}
				return
			}
			for _, expected := range tt.expectedContains {
				if !strings.Contains(message, expected) {
					t.Errorf("Expected alert message to contain '%s', got '%s'", expected, message)
				}
			}
		})
	}
}

// TestPlexBufferHealth_ShouldAlert tests alert logic
func TestPlexBufferHealth_ShouldAlert(t *testing.T) {
	tests := []struct {
		name         string
		healthStatus string
		alertSent    bool
		shouldAlert  bool
	}{
		{"Critical not sent", "critical", false, true},
		{"Critical already sent", "critical", true, false},
		{"Risky not sent", "risky", false, true},
		{"Risky already sent", "risky", true, false},
		{"Healthy not sent", "healthy", false, false},
		{"Healthy already sent", "healthy", true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			health := &models.PlexBufferHealth{
				HealthStatus: tt.healthStatus,
				AlertSent:    tt.alertSent,
			}

			result := health.ShouldAlert()
			if result != tt.shouldAlert {
				t.Errorf("Expected ShouldAlert() = %v, got %v", tt.shouldAlert, result)
			}
		})
	}
}
