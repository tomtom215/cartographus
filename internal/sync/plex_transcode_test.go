// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package sync

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/goccy/go-json"

	"github.com/tomtom215/cartographus/internal/models"
)

// TestGetTranscodeSessions_Success tests successful transcode session retrieval
func TestGetTranscodeSessions_Success(t *testing.T) {
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

		// Return mock response with 2 sessions (1 transcoding, 1 direct play)
		response := models.PlexSessionsResponse{
			MediaContainer: models.PlexSessionsContainer{
				Size: 2,
				Metadata: []models.PlexSession{
					{
						SessionKey: "session-1",
						Title:      "Test Movie 4K",
						User:       &models.PlexSessionUser{ID: 1, Title: "TestUser"},
						Player: &models.PlexSessionPlayer{
							Title:   "Chrome",
							Product: "Plex Web",
							Address: "192.168.1.100",
						},
						TranscodeSession: &models.PlexTranscodeSession{
							Progress:             45.5,
							Speed:                1.5,
							VideoDecision:        "transcode",
							AudioDecision:        "copy",
							TranscodeHwDecoding:  "qsv",
							TranscodeHwEncoding:  "qsv",
							SourceVideoCodec:     "hevc",
							VideoCodec:           "h264",
							Width:                1920,
							Height:               1080,
							Throttled:            false,
							TranscodeHwFullPipe:  true,
							TranscodeHwRequested: true,
						},
						Media: []models.PlexMedia{
							{
								VideoResolution: "4k",
								VideoCodec:      "hevc",
							},
						},
					},
					{
						SessionKey: "session-2",
						Title:      "Test Movie 1080p",
						User:       &models.PlexSessionUser{ID: 2, Title: "User2"},
						Player: &models.PlexSessionPlayer{
							Title:   "Plex for iOS",
							Product: "Plex for iOS",
						},
						TranscodeSession: nil, // Direct play
						Media: []models.PlexMedia{
							{
								VideoResolution: "1080",
								VideoCodec:      "h264",
							},
						},
					},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	// Create client and fetch sessions
	client := NewPlexClient(mockServer.URL, "test-token")
	sessions, err := client.GetTranscodeSessions(context.Background())

	// Assertions
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(sessions) != 2 {
		t.Fatalf("Expected 2 sessions, got %d", len(sessions))
	}

	// Verify transcoding session
	if !sessions[0].IsTranscoding() {
		t.Error("Expected session 0 to be transcoding")
	}
	if sessions[0].GetHardwareAccelerationType() != "Quick Sync" {
		t.Errorf("Expected Quick Sync, got %s", sessions[0].GetHardwareAccelerationType())
	}
	if sessions[0].GetTranscodeProgress() != 45.5 {
		t.Errorf("Expected progress 45.5, got %f", sessions[0].GetTranscodeProgress())
	}
	if sessions[0].GetTranscodeSpeed() != "1.5x" {
		t.Errorf("Expected speed 1.5x, got %s", sessions[0].GetTranscodeSpeed())
	}
	if sessions[0].GetQualityTransition() != "4k → 1080p" {
		t.Errorf("Expected 4k → 1080p, got %s", sessions[0].GetQualityTransition())
	}
	if sessions[0].GetCodecTransition() != "HEVC → H.264" {
		t.Errorf("Expected HEVC → H.264, got %s", sessions[0].GetCodecTransition())
	}

	// Verify direct play session
	if sessions[1].IsTranscoding() {
		t.Error("Expected session 1 to be direct play")
	}
	if sessions[1].GetHardwareAccelerationType() != "Direct Play" {
		t.Errorf("Expected Direct Play, got %s", sessions[1].GetHardwareAccelerationType())
	}
}

// TestGetTranscodeSessions_EmptySessions tests empty sessions response
func TestGetTranscodeSessions_EmptySessions(t *testing.T) {
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
	sessions, err := client.GetTranscodeSessions(context.Background())

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(sessions) != 0 {
		t.Errorf("Expected empty sessions, got %d", len(sessions))
	}
}

// TestGetTranscodeSessions_HardwareAccelerationTypes tests all hardware types
func TestGetTranscodeSessions_HardwareAccelerationTypes(t *testing.T) {
	testCases := []struct {
		name          string
		hwDecoding    string
		hwEncoding    string
		videoDecision string
		expectedType  string
		expectedIsHW  bool
	}{
		{"Quick Sync", "qsv", "qsv", "transcode", "Quick Sync", true},
		{"NVENC", "nvenc", "nvenc", "transcode", "NVENC", true},
		{"VAAPI", "vaapi", "vaapi", "transcode", "VAAPI", true},
		{"VideoToolbox", "videotoolbox", "videotoolbox", "transcode", "VideoToolbox", true},
		{"MediaCodec", "mediacodec", "mediacodec", "transcode", "MediaCodec", true},
		{"MediaFoundation", "mf", "mf", "transcode", "MediaFoundation", true},
		{"Software", "", "", "transcode", "Software", false},
		{"Direct Play", "", "", "direct play", "Direct Play", false},
		{"Unknown", "unknown-hw", "", "transcode", "Unknown", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			session := models.PlexSession{
				TranscodeSession: &models.PlexTranscodeSession{
					VideoDecision:       tc.videoDecision,
					TranscodeHwDecoding: tc.hwDecoding,
					TranscodeHwEncoding: tc.hwEncoding,
				},
			}

			hwType := session.GetHardwareAccelerationType()
			if hwType != tc.expectedType {
				t.Errorf("Expected %s, got %s", tc.expectedType, hwType)
			}

			isHW := session.IsHardwareAccelerated()
			if isHW != tc.expectedIsHW {
				t.Errorf("Expected IsHardwareAccelerated=%v, got %v", tc.expectedIsHW, isHW)
			}
		})
	}
}

// TestGetTranscodeSessions_ErrorHandling tests various error scenarios
func TestGetTranscodeSessions_ErrorHandling(t *testing.T) {
	testCases := []struct {
		name       string
		statusCode int
		response   string
		expectErr  bool
	}{
		{"HTTP 404", http.StatusNotFound, "", true},
		{"HTTP 500", http.StatusInternalServerError, "", true},
		{"HTTP 401", http.StatusUnauthorized, "", true},
		{"Invalid JSON", http.StatusOK, "invalid json", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.statusCode)
				if tc.response != "" {
					w.Write([]byte(tc.response))
				}
			}))
			defer mockServer.Close()

			client := NewPlexClient(mockServer.URL, "test-token")
			_, err := client.GetTranscodeSessions(context.Background())

			if tc.expectErr && err == nil {
				t.Error("Expected error, got nil")
			}
			if !tc.expectErr && err != nil {
				t.Errorf("Expected no error, got %v", err)
			}
		})
	}
}

// TestGetTranscodeSessions_RateLimiting tests HTTP 429 retry logic
func TestGetTranscodeSessions_RateLimiting(t *testing.T) {
	attemptCount := 0
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		if attemptCount < 3 {
			// First 2 attempts: rate limited
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		// 3rd attempt: success
		response := models.PlexSessionsResponse{
			MediaContainer: models.PlexSessionsContainer{
				Size:     0,
				Metadata: []models.PlexSession{},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	client := NewPlexClient(mockServer.URL, "test-token")
	sessions, err := client.GetTranscodeSessions(context.Background())

	if err != nil {
		t.Fatalf("Expected no error after retries, got %v", err)
	}
	if attemptCount != 3 {
		t.Errorf("Expected 3 attempts, got %d", attemptCount)
	}
	if sessions == nil {
		t.Error("Expected empty sessions array, got nil")
	}
}

// TestGetTranscodeSessions_MinimalFields tests parsing with minimal fields
func TestGetTranscodeSessions_MinimalFields(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return session with only required fields
		response := models.PlexSessionsResponse{
			MediaContainer: models.PlexSessionsContainer{
				Size: 1,
				Metadata: []models.PlexSession{
					{
						SessionKey: "minimal-session",
						Title:      "Minimal Movie",
						// No User, Player, TranscodeSession, Media
					},
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	client := NewPlexClient(mockServer.URL, "test-token")
	sessions, err := client.GetTranscodeSessions(context.Background())

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("Expected 1 session, got %d", len(sessions))
	}

	// Test nil-safe helper methods
	if sessions[0].IsTranscoding() {
		t.Error("Expected false for nil TranscodeSession")
	}
	if sessions[0].GetTranscodeProgress() != 0 {
		t.Error("Expected 0 progress for nil TranscodeSession")
	}
	if sessions[0].IsHardwareAccelerated() {
		t.Error("Expected false for nil TranscodeSession")
	}
	if sessions[0].GetQualityTransition() != "Unknown" {
		t.Errorf("Expected Unknown for empty Media, got %s", sessions[0].GetQualityTransition())
	}
}

// TestGetTranscodeSessions_ThrottledDetection tests throttling detection
func TestGetTranscodeSessions_ThrottledDetection(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := models.PlexSessionsResponse{
			MediaContainer: models.PlexSessionsContainer{
				Size: 1,
				Metadata: []models.PlexSession{
					{
						SessionKey: "throttled-session",
						Title:      "Throttled Movie",
						TranscodeSession: &models.PlexTranscodeSession{
							VideoDecision: "transcode",
							Throttled:     true, // System load high
							Progress:      10.0,
							Speed:         0.5, // Slow transcode
						},
					},
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	client := NewPlexClient(mockServer.URL, "test-token")
	sessions, err := client.GetTranscodeSessions(context.Background())

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("Expected 1 session, got %d", len(sessions))
	}
	if !sessions[0].IsThrottled() {
		t.Error("Expected session to be throttled")
	}
}

// TestGetTranscodeSessions_ContextCancellation tests context cancellation
func TestGetTranscodeSessions_ContextCancellation(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Delay response to allow cancellation
		time.Sleep(100 * time.Millisecond)
		response := models.PlexSessionsResponse{
			MediaContainer: models.PlexSessionsContainer{Size: 0},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	client := NewPlexClient(mockServer.URL, "test-token")

	// Create context with immediate cancellation
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := client.GetTranscodeSessions(ctx)

	if err == nil {
		t.Error("Expected context cancellation error, got nil")
	}
}

// TestGetTranscodeSessions_QualityTransitions tests quality transition formatting
func TestGetTranscodeSessions_QualityTransitions(t *testing.T) {
	testCases := []struct {
		name             string
		sourceResolution string
		targetHeight     int
		expected         string
	}{
		{"4K to 1080p", "4k", 1080, "4k → 1080p"},
		{"4K to 720p", "4k", 720, "4k → 720p"},
		{"1080 to 720p", "1080", 720, "1080 → 720p"},
		{"1080 to 480p", "1080", 480, "1080 → 480p"},
		{"Same quality", "1080", 1080, "1080"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			session := models.PlexSession{
				Media: []models.PlexMedia{
					{VideoResolution: tc.sourceResolution},
				},
				TranscodeSession: &models.PlexTranscodeSession{
					Height: tc.targetHeight,
				},
			}

			result := session.GetQualityTransition()
			if result != tc.expected {
				t.Errorf("Expected %s, got %s", tc.expected, result)
			}
		})
	}
}

// TestGetTranscodeSessions_CodecTransitions tests codec transition formatting
func TestGetTranscodeSessions_CodecTransitions(t *testing.T) {
	testCases := []struct {
		name        string
		sourceCodec string
		targetCodec string
		expected    string
	}{
		{"HEVC to H.264", "hevc", "h264", "HEVC → H.264"},
		{"H265 to AVC", "h265", "avc", "HEVC → H.264"},
		{"VP9 to H.264", "vp9", "h264", "VP9 → H.264"},
		{"Same codec", "h264", "h264", "H.264"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			session := models.PlexSession{
				Media: []models.PlexMedia{
					{VideoCodec: tc.sourceCodec},
				},
				TranscodeSession: &models.PlexTranscodeSession{
					VideoCodec: tc.targetCodec,
				},
			}

			result := session.GetCodecTransition()
			if result != tc.expected {
				t.Errorf("Expected %s, got %s", tc.expected, result)
			}
		})
	}
}

// TestGetTranscodeSessions_HelperMethods tests helper method edge cases
func TestGetTranscodeSessions_HelperMethods(t *testing.T) {
	t.Run("getUsername with nil user", func(t *testing.T) {
		username := getUsername(nil)
		if username != "Unknown" {
			t.Errorf("Expected Unknown, got %s", username)
		}
	})

	t.Run("getUsername with valid user", func(t *testing.T) {
		user := &models.PlexSessionUser{Title: "TestUser"}
		username := getUsername(user)
		if username != "TestUser" {
			t.Errorf("Expected TestUser, got %s", username)
		}
	})

	t.Run("getPlayerName with nil player", func(t *testing.T) {
		playerName := getPlayerName(nil)
		if playerName != "Unknown" {
			t.Errorf("Expected Unknown, got %s", playerName)
		}
	})

	t.Run("getPlayerName with title", func(t *testing.T) {
		player := &models.PlexSessionPlayer{Title: "Chrome", Product: "Plex Web"}
		playerName := getPlayerName(player)
		if playerName != "Chrome" {
			t.Errorf("Expected Chrome, got %s", playerName)
		}
	})

	t.Run("getPlayerName without title", func(t *testing.T) {
		player := &models.PlexSessionPlayer{Title: "", Product: "Plex for iOS"}
		playerName := getPlayerName(player)
		if playerName != "Plex for iOS" {
			t.Errorf("Expected Plex for iOS, got %s", playerName)
		}
	})
}

// TestGetTranscodeSessions_ConcurrentCalls tests thread safety
func TestGetTranscodeSessions_ConcurrentCalls(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := models.PlexSessionsResponse{
			MediaContainer: models.PlexSessionsContainer{
				Size:     0,
				Metadata: []models.PlexSession{},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	client := NewPlexClient(mockServer.URL, "test-token")

	// Run 10 concurrent requests
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			_, err := client.GetTranscodeSessions(context.Background())
			if err != nil {
				t.Errorf("Concurrent request failed: %v", err)
			}
			done <- true
		}()
	}

	// Wait for all requests to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}
