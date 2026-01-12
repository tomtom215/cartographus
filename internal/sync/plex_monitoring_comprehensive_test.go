// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package sync

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/tomtom215/cartographus/internal/config"
	"github.com/tomtom215/cartographus/internal/models"
)

// mockWebSocketHub implements WebSocketHub for testing
type mockWebSocketHub struct {
	mu            sync.Mutex
	broadcasts    []mockBroadcast
	broadcastChan chan struct{}
}

type mockBroadcast struct {
	messageType string
	data        interface{}
}

func newMockWebSocketHub() *mockWebSocketHub {
	return &mockWebSocketHub{
		broadcasts:    make([]mockBroadcast, 0),
		broadcastChan: make(chan struct{}, 100),
	}
}

func (m *mockWebSocketHub) BroadcastJSON(messageType string, data interface{}) {
	m.mu.Lock()
	m.broadcasts = append(m.broadcasts, mockBroadcast{messageType: messageType, data: data})
	m.mu.Unlock()
	select {
	case m.broadcastChan <- struct{}{}:
	default:
	}
}

func (m *mockWebSocketHub) getBroadcasts() []mockBroadcast {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]mockBroadcast, len(m.broadcasts))
	copy(result, m.broadcasts)
	return result
}

// TestStartTranscodeMonitoringNilClient tests error when Plex client is nil
func TestStartTranscodeMonitoringNilClient(t *testing.T) {
	cfg := &config.Config{
		Plex: config.PlexConfig{
			TranscodeMonitoring: true,
		},
	}

	manager := &Manager{
		cfg:        cfg,
		plexClient: nil,
	}

	err := manager.StartTranscodeMonitoring(context.Background())

	if err == nil {
		t.Error("expected error for nil Plex client")
	}

	if err.Error() != "plex client not initialized" {
		t.Errorf("unexpected error message: %v", err)
	}
}

// TestStartTranscodeMonitoringDisabled tests error when monitoring is disabled
func TestStartTranscodeMonitoringDisabled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := &config.Config{
		Plex: config.PlexConfig{
			Enabled:             true,
			TranscodeMonitoring: false, // Disabled
		},
	}

	plexClient := NewPlexClient(server.URL, "test-token")
	manager := &Manager{
		cfg:        cfg,
		plexClient: plexClient,
	}

	err := manager.StartTranscodeMonitoring(context.Background())

	if err == nil {
		t.Error("expected error when monitoring is disabled")
	}

	if err.Error() != "transcode monitoring disabled in config" {
		t.Errorf("unexpected error message: %v", err)
	}
}

// TestStartBufferHealthMonitoringNilClient tests error when Plex client is nil
func TestStartBufferHealthMonitoringNilClient(t *testing.T) {
	cfg := &config.Config{
		Plex: config.PlexConfig{
			BufferHealthMonitoring: true,
		},
	}

	manager := &Manager{
		cfg:        cfg,
		plexClient: nil,
	}

	err := manager.StartBufferHealthMonitoring(context.Background())

	if err == nil {
		t.Error("expected error for nil Plex client")
	}

	if err.Error() != "plex client not initialized" {
		t.Errorf("unexpected error message: %v", err)
	}
}

// TestStartBufferHealthMonitoringDisabled tests error when monitoring is disabled
func TestStartBufferHealthMonitoringDisabled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := &config.Config{
		Plex: config.PlexConfig{
			Enabled:                true,
			BufferHealthMonitoring: false, // Disabled
		},
	}

	plexClient := NewPlexClient(server.URL, "test-token")
	manager := &Manager{
		cfg:        cfg,
		plexClient: plexClient,
	}

	err := manager.StartBufferHealthMonitoring(context.Background())

	if err == nil {
		t.Error("expected error when monitoring is disabled")
	}

	if err.Error() != "buffer health monitoring disabled in config" {
		t.Errorf("unexpected error message: %v", err)
	}
}

// TestPollTranscodeSessionsBroadcasts tests that transcode sessions are broadcast
func TestPollTranscodeSessionsBroadcasts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"MediaContainer": {
				"size": 2,
				"Metadata": [
					{
						"sessionKey": "session1",
						"title": "Movie 1",
						"TranscodeSession": {
							"key": "trans1",
							"progress": 50.0,
							"speed": 2.5,
							"videoDecision": "transcode"
						},
						"User": {"id": 1, "title": "user1"},
						"Player": {"title": "Player 1"}
					},
					{
						"sessionKey": "session2",
						"title": "Movie 2",
						"TranscodeSession": null,
						"User": {"id": 2, "title": "user2"}
					}
				]
			}
		}`))
	}))
	defer server.Close()

	cfg := &config.Config{
		Plex: config.PlexConfig{
			Enabled:                     true,
			TranscodeMonitoring:         true,
			TranscodeMonitoringInterval: 10 * time.Second,
		},
	}

	plexClient := NewPlexClient(server.URL, "test-token")
	wsHub := newMockWebSocketHub()

	manager := &Manager{
		cfg:        cfg,
		plexClient: plexClient,
		wsHub:      wsHub,
	}

	manager.pollTranscodeSessions(context.Background())

	broadcasts := wsHub.getBroadcasts()
	if len(broadcasts) == 0 {
		t.Fatal("expected at least one broadcast")
	}

	// Verify broadcast type
	if broadcasts[0].messageType != "plex_transcode_sessions" {
		t.Errorf("expected messageType 'plex_transcode_sessions', got %q", broadcasts[0].messageType)
	}

	// Verify broadcast data structure
	data, ok := broadcasts[0].data.(map[string]interface{})
	if !ok {
		t.Fatal("broadcast data should be a map")
	}

	if _, ok := data["sessions"]; !ok {
		t.Error("broadcast should contain 'sessions'")
	}
	if _, ok := data["timestamp"]; !ok {
		t.Error("broadcast should contain 'timestamp'")
	}
	if _, ok := data["count"]; !ok {
		t.Error("broadcast should contain 'count'")
	}
}

// TestPollTranscodeSessionsError tests error handling
func TestPollTranscodeSessionsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	cfg := &config.Config{
		Plex: config.PlexConfig{
			Enabled:                     true,
			TranscodeMonitoring:         true,
			TranscodeMonitoringInterval: 10 * time.Second,
		},
	}

	plexClient := NewPlexClient(server.URL, "test-token")
	wsHub := newMockWebSocketHub()

	manager := &Manager{
		cfg:        cfg,
		plexClient: plexClient,
		wsHub:      wsHub,
	}

	// Should not panic, just log error
	manager.pollTranscodeSessions(context.Background())

	// No broadcasts should be made on error
	broadcasts := wsHub.getBroadcasts()
	if len(broadcasts) != 0 {
		t.Error("should not broadcast on error")
	}
}

// TestPollBufferHealthBroadcasts tests buffer health broadcast
func TestPollBufferHealthBroadcasts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"MediaContainer": {
				"size": 1,
				"Metadata": [
					{
						"sessionKey": "session1",
						"title": "Movie 1",
						"viewOffset": 3600000,
						"duration": 7200000,
						"TranscodeSession": {
							"key": "trans1",
							"progress": 50.0,
							"speed": 1.5,
							"maxOffsetAvailable": 5000000,
							"minOffsetAvailable": 0,
							"throttled": false,
							"complete": false
						},
						"User": {"id": 1, "title": "user1"},
						"Player": {"title": "Player 1"}
					}
				]
			}
		}`))
	}))
	defer server.Close()

	cfg := &config.Config{
		Plex: config.PlexConfig{
			Enabled:                       true,
			BufferHealthMonitoring:        true,
			BufferHealthPollInterval:      5 * time.Second,
			BufferHealthCriticalThreshold: 20.0,
			BufferHealthRiskyThreshold:    50.0,
		},
	}

	plexClient := NewPlexClient(server.URL, "test-token")
	wsHub := newMockWebSocketHub()

	manager := &Manager{
		cfg:               cfg,
		plexClient:        plexClient,
		wsHub:             wsHub,
		bufferHealthCache: make(map[string]*models.PlexBufferHealth),
	}

	manager.pollBufferHealth(context.Background())

	broadcasts := wsHub.getBroadcasts()
	if len(broadcasts) == 0 {
		t.Fatal("expected at least one broadcast")
	}

	// Verify broadcast type
	if broadcasts[0].messageType != "buffer_health_update" {
		t.Errorf("expected messageType 'buffer_health_update', got %q", broadcasts[0].messageType)
	}

	// Verify broadcast data structure
	data, ok := broadcasts[0].data.(map[string]interface{})
	if !ok {
		t.Fatal("broadcast data should be a map")
	}

	if _, ok := data["sessions"]; !ok {
		t.Error("broadcast should contain 'sessions'")
	}
	if _, ok := data["timestamp"]; !ok {
		t.Error("broadcast should contain 'timestamp'")
	}
	if _, ok := data["critical_count"]; !ok {
		t.Error("broadcast should contain 'critical_count'")
	}
	if _, ok := data["risky_count"]; !ok {
		t.Error("broadcast should contain 'risky_count'")
	}
}

// TestPollBufferHealthError tests error handling
func TestPollBufferHealthError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	cfg := &config.Config{
		Plex: config.PlexConfig{
			Enabled:                       true,
			BufferHealthMonitoring:        true,
			BufferHealthPollInterval:      5 * time.Second,
			BufferHealthCriticalThreshold: 20.0,
			BufferHealthRiskyThreshold:    50.0,
		},
	}

	plexClient := NewPlexClient(server.URL, "test-token")
	wsHub := newMockWebSocketHub()

	manager := &Manager{
		cfg:               cfg,
		plexClient:        plexClient,
		wsHub:             wsHub,
		bufferHealthCache: make(map[string]*models.PlexBufferHealth),
	}

	// Should not panic
	manager.pollBufferHealth(context.Background())

	// No broadcasts should be made on error
	broadcasts := wsHub.getBroadcasts()
	if len(broadcasts) != 0 {
		t.Error("should not broadcast on error")
	}
}

// TestProcessSessionForBufferHealthNilTranscode tests skipping direct play sessions
func TestProcessSessionForBufferHealthNilTranscode(t *testing.T) {
	cfg := &config.Config{
		Plex: config.PlexConfig{
			BufferHealthCriticalThreshold: 20.0,
			BufferHealthRiskyThreshold:    50.0,
		},
	}

	manager := &Manager{
		cfg:               cfg,
		bufferHealthCache: make(map[string]*models.PlexBufferHealth),
	}

	// Session without TranscodeSession (direct play)
	session := &models.PlexSession{
		SessionKey:       "session1",
		Title:            "Direct Play Movie",
		TranscodeSession: nil,
	}

	result := manager.processSessionForBufferHealth(context.Background(), session)

	// Should return nil for direct play sessions
	if result != nil {
		t.Error("expected nil for direct play session")
	}
}

// TestCleanupInactiveBufferHealthCache tests cache cleanup
func TestCleanupInactiveBufferHealthCache(t *testing.T) {
	manager := &Manager{
		bufferHealthCache: make(map[string]*models.PlexBufferHealth),
	}

	// Pre-populate cache with some entries
	manager.bufferHealthCache["session1"] = &models.PlexBufferHealth{SessionKey: "session1"}
	manager.bufferHealthCache["session2"] = &models.PlexBufferHealth{SessionKey: "session2"}
	manager.bufferHealthCache["session3"] = &models.PlexBufferHealth{SessionKey: "session3"}

	// Only session1 and session3 are still active
	activeSessions := []*models.PlexBufferHealth{
		{SessionKey: "session1"},
		{SessionKey: "session3"},
	}

	manager.cleanupInactiveBufferHealthCache(activeSessions)

	// session2 should be removed
	if len(manager.bufferHealthCache) != 2 {
		t.Errorf("expected 2 cached sessions, got %d", len(manager.bufferHealthCache))
	}

	if _, exists := manager.bufferHealthCache["session2"]; exists {
		t.Error("session2 should have been removed from cache")
	}

	if _, exists := manager.bufferHealthCache["session1"]; !exists {
		t.Error("session1 should still be in cache")
	}

	if _, exists := manager.bufferHealthCache["session3"]; !exists {
		t.Error("session3 should still be in cache")
	}
}

// TestLogBufferHealthDetailsHealthy tests that healthy sessions don't log
func TestLogBufferHealthDetailsHealthy(t *testing.T) {
	// Healthy buffer health (above risky threshold)
	bufferHealth := &models.PlexBufferHealth{
		SessionKey:        "session1",
		Title:             "Movie",
		BufferFillPercent: 75.0, // Healthy (above 50%)
		HealthStatus:      "healthy",
		BufferSeconds:     30.0,
	}

	// Should not panic or log (healthy sessions are skipped)
	logBufferHealthDetails(bufferHealth)
}

// TestLogBufferHealthDetailsCritical tests logging for critical sessions
func TestLogBufferHealthDetailsCritical(t *testing.T) {
	// Critical buffer health (below 20%)
	bufferHealth := &models.PlexBufferHealth{
		SessionKey:        "session1",
		Title:             "Movie",
		BufferFillPercent: 15.0, // Critical
		HealthStatus:      "critical",
		BufferSeconds:     5.0,
		BufferDrainRate:   1.5,
		Username:          "testuser",
		PlayerDevice:      "Test Player",
	}

	// Should not panic (logs to console)
	logBufferHealthDetails(bufferHealth)
}

// TestLogBufferHealthDetailsRisky tests logging for risky sessions
func TestLogBufferHealthDetailsRisky(t *testing.T) {
	// Risky buffer health (20-50%)
	bufferHealth := &models.PlexBufferHealth{
		SessionKey:        "session1",
		Title:             "Movie",
		BufferFillPercent: 35.0, // Risky
		HealthStatus:      "risky",
		BufferSeconds:     15.0,
		BufferDrainRate:   1.2,
		Username:          "testuser",
		PlayerDevice:      "Test Player",
	}

	// Should not panic (logs to console)
	logBufferHealthDetails(bufferHealth)
}

// TestRunTranscodeMonitoringLoopCancel tests loop cancellation
func TestRunTranscodeMonitoringLoopCancel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"MediaContainer": {"size": 0}}`))
	}))
	defer server.Close()

	cfg := &config.Config{
		Plex: config.PlexConfig{
			Enabled:                     true,
			TranscodeMonitoring:         true,
			TranscodeMonitoringInterval: 50 * time.Millisecond,
		},
	}

	plexClient := NewPlexClient(server.URL, "test-token")
	wsHub := newMockWebSocketHub()

	manager := &Manager{
		cfg:        cfg,
		plexClient: plexClient,
		wsHub:      wsHub,
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Start loop in goroutine
	done := make(chan bool)
	go func() {
		manager.runTranscodeMonitoringLoop(ctx)
		done <- true
	}()

	// Wait for a couple of ticks
	time.Sleep(100 * time.Millisecond)

	// Cancel context
	cancel()

	// Wait for loop to stop
	select {
	case <-done:
		// Success
	case <-time.After(500 * time.Millisecond):
		t.Error("loop did not stop after context cancellation")
	}
}

// TestRunBufferHealthMonitoringLoopCancel tests loop cancellation
func TestRunBufferHealthMonitoringLoopCancel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"MediaContainer": {"size": 0}}`))
	}))
	defer server.Close()

	cfg := &config.Config{
		Plex: config.PlexConfig{
			Enabled:                       true,
			BufferHealthMonitoring:        true,
			BufferHealthPollInterval:      50 * time.Millisecond,
			BufferHealthCriticalThreshold: 20.0,
			BufferHealthRiskyThreshold:    50.0,
		},
	}

	plexClient := NewPlexClient(server.URL, "test-token")
	wsHub := newMockWebSocketHub()

	manager := &Manager{
		cfg:               cfg,
		plexClient:        plexClient,
		wsHub:             wsHub,
		bufferHealthCache: make(map[string]*models.PlexBufferHealth),
		wg:                sync.WaitGroup{},
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Simulate what StartBufferHealthMonitoring does
	manager.wg.Add(1)

	// Start loop in goroutine
	go manager.runBufferHealthMonitoringLoop(ctx)

	// Wait for a couple of ticks
	time.Sleep(100 * time.Millisecond)

	// Cancel context
	cancel()

	// Wait for loop to stop
	done := make(chan bool)
	go func() {
		manager.wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		// Success
	case <-time.After(500 * time.Millisecond):
		t.Error("loop did not stop after context cancellation")
	}
}

// TestPollTranscodeSessionsNoHub tests polling without WebSocket hub
func TestPollTranscodeSessionsNoHub(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"MediaContainer": {
				"size": 1,
				"Metadata": [
					{
						"sessionKey": "session1",
						"title": "Movie 1",
						"TranscodeSession": {
							"videoDecision": "transcode"
						}
					}
				]
			}
		}`))
	}))
	defer server.Close()

	cfg := &config.Config{
		Plex: config.PlexConfig{
			Enabled:                     true,
			TranscodeMonitoring:         true,
			TranscodeMonitoringInterval: 10 * time.Second,
		},
	}

	plexClient := NewPlexClient(server.URL, "test-token")

	manager := &Manager{
		cfg:        cfg,
		plexClient: plexClient,
		wsHub:      nil, // No WebSocket hub
	}

	// Should not panic even without wsHub
	manager.pollTranscodeSessions(context.Background())
}

// TestPollBufferHealthNoHub tests polling without WebSocket hub
func TestPollBufferHealthNoHub(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"MediaContainer": {"size": 0}}`))
	}))
	defer server.Close()

	cfg := &config.Config{
		Plex: config.PlexConfig{
			Enabled:                       true,
			BufferHealthMonitoring:        true,
			BufferHealthPollInterval:      5 * time.Second,
			BufferHealthCriticalThreshold: 20.0,
			BufferHealthRiskyThreshold:    50.0,
		},
	}

	plexClient := NewPlexClient(server.URL, "test-token")

	manager := &Manager{
		cfg:               cfg,
		plexClient:        plexClient,
		wsHub:             nil, // No WebSocket hub
		bufferHealthCache: make(map[string]*models.PlexBufferHealth),
	}

	// Should not panic even without wsHub
	manager.pollBufferHealth(context.Background())
}

// TestBufferHealthCacheUpdates tests that cache is updated on poll
func TestBufferHealthCacheUpdates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// Both sessions and timeline endpoints return similar response format
		w.Write([]byte(`{
			"MediaContainer": {
				"size": 1,
				"Metadata": [
					{
						"sessionKey": "session1",
						"title": "Movie 1",
						"viewOffset": 3600000,
						"duration": 7200000,
						"TranscodeSession": {
							"key": "trans1",
							"progress": 50.0,
							"speed": 1.5,
							"maxOffsetAvailable": 5000000,
							"throttled": false,
							"videoDecision": "transcode"
						},
						"User": {"id": 1, "title": "user1"},
						"Player": {"title": "Player 1"}
					}
				]
			}
		}`))
	}))
	defer server.Close()

	cfg := &config.Config{
		Plex: config.PlexConfig{
			Enabled:                       true,
			BufferHealthMonitoring:        true,
			BufferHealthPollInterval:      5 * time.Second,
			BufferHealthCriticalThreshold: 20.0,
			BufferHealthRiskyThreshold:    50.0,
		},
	}

	plexClient := NewPlexClient(server.URL, "test-token")

	manager := &Manager{
		cfg:               cfg,
		plexClient:        plexClient,
		wsHub:             nil,
		bufferHealthCache: make(map[string]*models.PlexBufferHealth),
	}

	// Initial poll
	manager.pollBufferHealth(context.Background())

	// Verify cache was updated
	if len(manager.bufferHealthCache) != 1 {
		t.Errorf("expected 1 cached session, got %d", len(manager.bufferHealthCache))
	}

	cached, exists := manager.bufferHealthCache["session1"]
	if !exists {
		t.Error("session1 should be in cache")
	}

	if cached == nil {
		t.Error("cached value should not be nil")
	}
}

// TestBufferHealthDrainRateCalculation tests drain rate calculation with previous state
func TestBufferHealthDrainRateCalculation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"MediaContainer": {
				"size": 1,
				"Metadata": [
					{
						"sessionKey": "session1",
						"title": "Movie 1",
						"viewOffset": 3600000,
						"duration": 7200000,
						"TranscodeSession": {
							"key": "trans1",
							"progress": 50.0,
							"speed": 1.5,
							"maxOffsetAvailable": 5000000,
							"throttled": false,
							"videoDecision": "transcode"
						},
						"User": {"id": 1, "title": "user1"},
						"Player": {"title": "Player 1"}
					}
				]
			}
		}`))
	}))
	defer server.Close()

	cfg := &config.Config{
		Plex: config.PlexConfig{
			Enabled:                       true,
			BufferHealthMonitoring:        true,
			BufferHealthPollInterval:      5 * time.Second,
			BufferHealthCriticalThreshold: 20.0,
			BufferHealthRiskyThreshold:    50.0,
		},
	}

	plexClient := NewPlexClient(server.URL, "test-token")

	// Pre-populate cache with previous state
	previousBufferSeconds := 30.0
	manager := &Manager{
		cfg:        cfg,
		plexClient: plexClient,
		wsHub:      nil,
		bufferHealthCache: map[string]*models.PlexBufferHealth{
			"session1": {
				SessionKey:    "session1",
				BufferSeconds: previousBufferSeconds,
			},
		},
	}

	// Poll should use previous state for drain rate calculation
	manager.pollBufferHealth(context.Background())

	// Verify cache was updated (drain rate should be calculated)
	cached := manager.bufferHealthCache["session1"]
	if cached == nil {
		t.Error("session1 should be in cache")
	}
}
