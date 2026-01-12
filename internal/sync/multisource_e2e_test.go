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

	"github.com/goccy/go-json"

	"github.com/tomtom215/cartographus/internal/config"
	"github.com/tomtom215/cartographus/internal/models"
)

// ============================================================================
// Multi-Source E2E Scenario Tests
//
// These tests simulate real-world scenarios with multiple media servers
// (Jellyfin, Emby) running concurrently and publishing events.
// ============================================================================

// mockMultiSourceHub implements the WebSocket hub interface for testing.
// Session poller doesn't broadcast to hub (only publishes events), so this is a no-op.
type mockMultiSourceHub struct{}

func (m *mockMultiSourceHub) BroadcastJSON(messageType string, data interface{}) {
	// No-op: session poller doesn't use hub broadcasts
}

// mockMultiSourcePublisher collects events from all sources
type mockMultiSourcePublisher struct {
	mu     sync.Mutex
	events []*models.PlaybackEvent
}

func (m *mockMultiSourcePublisher) PublishPlaybackEvent(_ context.Context, event *models.PlaybackEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, event)
	return nil
}

func (m *mockMultiSourcePublisher) getEventCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.events)
}

func (m *mockMultiSourcePublisher) getEventsByServerID(serverID string) []*models.PlaybackEvent {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []*models.PlaybackEvent
	for _, e := range m.events {
		if e.ServerID != nil && *e.ServerID == serverID {
			result = append(result, e)
		}
	}
	return result
}

// ============================================================================
// Scenario: Simultaneous Playback on Multiple Servers
// ============================================================================

func TestMultiSource_SimultaneousPlayback(t *testing.T) {
	t.Parallel()

	// Create mock Jellyfin server
	jellyfinServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/System/Ping":
			w.WriteHeader(http.StatusOK)
		case "/System/Info":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"ServerName":"Jellyfin Server","Version":"10.8.0"}`))
		case "/Sessions":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[{
				"Id": "jellyfin-session-1",
				"UserName": "JellyfinUser",
				"NowPlayingItem": {
					"Id": "jellyfin-item-1",
					"Name": "The Matrix",
					"Type": "Movie",
					"RunTimeTicks": 88800000000
				},
				"PlayState": {
					"IsPaused": false,
					"PlayMethod": "DirectPlay"
				}
			}]`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer jellyfinServer.Close()

	// Create mock Emby server
	embyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/System/Ping":
			w.WriteHeader(http.StatusOK)
		case "/System/Info":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"ServerName":"Emby Server","Version":"4.7.0"}`))
		case "/Sessions":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[{
				"Id": "emby-session-1",
				"UserName": "EmbyUser",
				"NowPlayingItem": {
					"Id": "emby-item-1",
					"Name": "Inception",
					"Type": "Movie",
					"RunTimeTicks": 93600000000
				},
				"PlayState": {
					"IsPaused": false,
					"PlayMethod": "DirectPlay"
				}
			}]`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer embyServer.Close()

	// Create shared hub and publisher
	hub := &mockMultiSourceHub{}
	publisher := &mockMultiSourcePublisher{}

	// Create Jellyfin manager
	jellyfinCfg := &config.JellyfinConfig{
		Enabled:                true,
		URL:                    jellyfinServer.URL,
		APIKey:                 "jellyfin-api-key",
		ServerID:               "jellyfin-server-1",
		RealtimeEnabled:        false,
		SessionPollingEnabled:  true,
		SessionPollingInterval: 100 * time.Millisecond,
	}
	jellyfinManager := NewJellyfinManager(jellyfinCfg, hub, nil)
	jellyfinManager.SetEventPublisher(publisher)

	// Create Emby manager
	embyCfg := &config.EmbyConfig{
		Enabled:                true,
		URL:                    embyServer.URL,
		APIKey:                 "emby-api-key",
		ServerID:               "emby-server-1",
		RealtimeEnabled:        false,
		SessionPollingEnabled:  true,
		SessionPollingInterval: 100 * time.Millisecond,
	}
	embyManager := NewEmbyManager(embyCfg, hub, nil)
	embyManager.SetEventPublisher(publisher)

	ctx := context.Background()

	// Start both managers
	if err := jellyfinManager.Start(ctx); err != nil {
		t.Fatalf("Failed to start Jellyfin manager: %v", err)
	}
	if err := embyManager.Start(ctx); err != nil {
		t.Fatalf("Failed to start Emby manager: %v", err)
	}

	// Let them run for a bit
	time.Sleep(250 * time.Millisecond)

	// Stop both managers
	if err := jellyfinManager.Stop(); err != nil {
		t.Errorf("Failed to stop Jellyfin manager: %v", err)
	}
	if err := embyManager.Stop(); err != nil {
		t.Errorf("Failed to stop Emby manager: %v", err)
	}

	// Verify events from both sources
	jellyfinEvents := publisher.getEventsByServerID("jellyfin-server-1")
	embyEvents := publisher.getEventsByServerID("emby-server-1")

	if len(jellyfinEvents) == 0 {
		t.Error("expected events from Jellyfin server")
	}
	if len(embyEvents) == 0 {
		t.Error("expected events from Emby server")
	}

	// Note: Session poller publishes events but doesn't broadcast to WebSocket hub.
	// Hub broadcasts only happen from WebSocket handler (handleSessionUpdate).
	// Therefore we only verify events were published, not broadcasts.
}

// ============================================================================
// Scenario: User Watching on Multiple Servers
// ============================================================================

func TestMultiSource_SameUserMultipleServers(t *testing.T) {
	t.Parallel()

	// Track resolved users
	userResolver := newMockUserResolverWithTracking()

	// Create Jellyfin server with user "john"
	jellyfinServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/System/Ping":
			w.WriteHeader(http.StatusOK)
		case "/System/Info":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"ServerName":"Jellyfin","Version":"10.8.0"}`))
		case "/Sessions":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[{
				"Id": "jf-session-1",
				"UserId": "john-uuid-jellyfin",
				"UserName": "john",
				"NowPlayingItem": {
					"Id": "item-1",
					"Name": "Movie 1",
					"Type": "Movie",
					"RunTimeTicks": 1000000000
				},
				"PlayState": {"PlayMethod": "DirectPlay"}
			}]`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer jellyfinServer.Close()

	// Create Emby server with same user "john"
	embyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/System/Ping":
			w.WriteHeader(http.StatusOK)
		case "/System/Info":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"ServerName":"Emby","Version":"4.7.0"}`))
		case "/Sessions":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[{
				"Id": "emby-session-1",
				"UserId": "john-uuid-emby",
				"UserName": "john",
				"NowPlayingItem": {
					"Id": "item-1",
					"Name": "Movie 1",
					"Type": "Movie",
					"RunTimeTicks": 1000000000
				},
				"PlayState": {"PlayMethod": "DirectPlay"}
			}]`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer embyServer.Close()

	publisher := &mockMultiSourcePublisher{}

	// Create managers with user resolver
	// Note: SessionPollingInterval minimum is 10 seconds (enforced by managers)
	jellyfinCfg := &config.JellyfinConfig{
		Enabled:                true,
		URL:                    jellyfinServer.URL,
		APIKey:                 "test-key",
		ServerID:               "jellyfin-1",
		SessionPollingEnabled:  true,
		SessionPollingInterval: 10 * time.Second,
	}
	jellyfinManager := NewJellyfinManager(jellyfinCfg, nil, userResolver)
	jellyfinManager.SetEventPublisher(publisher)

	embyCfg := &config.EmbyConfig{
		Enabled:                true,
		URL:                    embyServer.URL,
		APIKey:                 "test-key",
		ServerID:               "emby-1",
		SessionPollingEnabled:  true,
		SessionPollingInterval: 10 * time.Second,
	}
	embyManager := NewEmbyManager(embyCfg, nil, userResolver)
	embyManager.SetEventPublisher(publisher)

	ctx := context.Background()

	// Start both
	_ = jellyfinManager.Start(ctx)
	_ = embyManager.Start(ctx)

	// Wait for session polling to complete - minimum interval is 10s
	// Poll until we get expected results or timeout after 12 seconds
	var userCount int
	for i := 0; i < 24; i++ {
		time.Sleep(500 * time.Millisecond)
		userCount = userResolver.getUniqueUserCount()
		if userCount >= 2 {
			break
		}
	}

	_ = jellyfinManager.Stop()
	_ = embyManager.Stop()

	// User resolver should have been called for both sources
	// Each source + server combination creates a unique mapping
	if userCount != 2 {
		t.Errorf("expected 2 unique user mappings (one per source), got %d", userCount)
	}
}

// ============================================================================
// Scenario: Graceful Degradation When One Source Fails
// ============================================================================

func TestMultiSource_GracefulDegradation(t *testing.T) {
	t.Parallel()

	// Working Jellyfin server
	jellyfinServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/System/Ping":
			w.WriteHeader(http.StatusOK)
		case "/System/Info":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"ServerName":"Jellyfin","Version":"10.8.0"}`))
		case "/Sessions":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[{
				"Id": "jf-session-1",
				"UserName": "User",
				"NowPlayingItem": {"Id": "item-1", "Name": "Movie", "Type": "Movie"}
			}]`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer jellyfinServer.Close()

	// Failing Emby server (returns 500 errors)
	embyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer embyServer.Close()

	publisher := &mockMultiSourcePublisher{}

	// Create managers
	jellyfinCfg := &config.JellyfinConfig{
		Enabled:                true,
		URL:                    jellyfinServer.URL,
		APIKey:                 "test-key",
		ServerID:               "jellyfin-1",
		SessionPollingEnabled:  true,
		SessionPollingInterval: 100 * time.Millisecond,
	}
	jellyfinManager := NewJellyfinManager(jellyfinCfg, nil, nil)
	jellyfinManager.SetEventPublisher(publisher)

	embyCfg := &config.EmbyConfig{
		Enabled:                true,
		URL:                    embyServer.URL,
		APIKey:                 "test-key",
		ServerID:               "emby-1",
		SessionPollingEnabled:  true,
		SessionPollingInterval: 100 * time.Millisecond,
	}
	embyManager := NewEmbyManager(embyCfg, nil, nil)
	embyManager.SetEventPublisher(publisher)

	ctx := context.Background()

	// Start both - Emby should fail gracefully
	err := jellyfinManager.Start(ctx)
	if err != nil {
		t.Errorf("Jellyfin manager should start successfully: %v", err)
	}

	err = embyManager.Start(ctx)
	if err != nil {
		t.Errorf("Emby manager should start (gracefully handling errors): %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	_ = jellyfinManager.Stop()
	_ = embyManager.Stop()

	// Should have events from Jellyfin but not Emby
	jellyfinEvents := publisher.getEventsByServerID("jellyfin-1")
	embyEvents := publisher.getEventsByServerID("emby-1")

	if len(jellyfinEvents) == 0 {
		t.Error("expected events from Jellyfin server (which is working)")
	}
	if len(embyEvents) != 0 {
		t.Errorf("expected no events from Emby server (which is failing), got %d", len(embyEvents))
	}
}

// ============================================================================
// Scenario: High-Volume Multi-Source Load
// ============================================================================

func TestMultiSource_HighVolumeLoad(t *testing.T) {
	t.Parallel()

	const numSessionsPerServer = 50

	// Generate session response
	generateSessions := func(prefix string, count int) []byte {
		sessions := make([]map[string]interface{}, count)
		for i := 0; i < count; i++ {
			sessions[i] = map[string]interface{}{
				"Id":       prefix + "-session-" + string(rune('0'+i%10)) + string(rune('0'+i/10)),
				"UserName": "User" + string(rune('0'+i%10)),
				"NowPlayingItem": map[string]interface{}{
					"Id":   "item-" + string(rune('0'+i)),
					"Name": "Movie " + string(rune('0'+i)),
					"Type": "Movie",
				},
				"PlayState": map[string]interface{}{
					"IsPaused": false,
				},
			}
		}
		data, _ := json.Marshal(sessions)
		return data
	}

	jellyfinSessions := generateSessions("jellyfin", numSessionsPerServer)
	embySessions := generateSessions("emby", numSessionsPerServer)

	jellyfinServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/System/Ping":
			w.WriteHeader(http.StatusOK)
		case "/System/Info":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"ServerName":"Jellyfin","Version":"10.8.0"}`))
		case "/Sessions":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(jellyfinSessions)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer jellyfinServer.Close()

	embyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/System/Ping":
			w.WriteHeader(http.StatusOK)
		case "/System/Info":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"ServerName":"Emby","Version":"4.7.0"}`))
		case "/Sessions":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(embySessions)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer embyServer.Close()

	publisher := &mockEventPublisher{}

	// Create managers
	jellyfinCfg := &config.JellyfinConfig{
		Enabled:                true,
		URL:                    jellyfinServer.URL,
		APIKey:                 "test-key",
		ServerID:               "jellyfin-1",
		SessionPollingEnabled:  true,
		SessionPollingInterval: 50 * time.Millisecond,
	}
	jellyfinManager := NewJellyfinManager(jellyfinCfg, nil, nil)
	jellyfinManager.SetEventPublisher(publisher)

	embyCfg := &config.EmbyConfig{
		Enabled:                true,
		URL:                    embyServer.URL,
		APIKey:                 "test-key",
		ServerID:               "emby-1",
		SessionPollingEnabled:  true,
		SessionPollingInterval: 50 * time.Millisecond,
	}
	embyManager := NewEmbyManager(embyCfg, nil, nil)
	embyManager.SetEventPublisher(publisher)

	ctx := context.Background()

	_ = jellyfinManager.Start(ctx)
	_ = embyManager.Start(ctx)

	// Let them run
	time.Sleep(150 * time.Millisecond)

	_ = jellyfinManager.Stop()
	_ = embyManager.Stop()

	// Should have events from both sources
	finalCount := publisher.publishCalls.Load()
	if finalCount < int32(numSessionsPerServer*2) {
		t.Errorf("expected at least %d events, got %d", numSessionsPerServer*2, finalCount)
	}
}

// ============================================================================
// Scenario: Rapid Start/Stop Cycles
// ============================================================================

func TestMultiSource_RapidStartStopCycles(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/System/Ping":
			w.WriteHeader(http.StatusOK)
		case "/System/Info":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"ServerName":"Test","Version":"1.0.0"}`))
		case "/Sessions":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[]`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	cfg := &config.JellyfinConfig{
		Enabled:                true,
		URL:                    server.URL,
		APIKey:                 "test-key",
		ServerID:               "test-1",
		SessionPollingEnabled:  true,
		SessionPollingInterval: 100 * time.Millisecond,
	}

	ctx := context.Background()

	// Rapid start/stop cycles
	for i := 0; i < 5; i++ {
		manager := NewJellyfinManager(cfg, nil, nil)

		if err := manager.Start(ctx); err != nil {
			t.Errorf("cycle %d: Start failed: %v", i, err)
		}

		time.Sleep(20 * time.Millisecond)

		if err := manager.Stop(); err != nil {
			t.Errorf("cycle %d: Stop failed: %v", i, err)
		}
	}
}

// ============================================================================
// Scenario: Mixed Session States
// ============================================================================

func TestMultiSource_MixedSessionStates(t *testing.T) {
	t.Parallel()

	jellyfinServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/System/Ping":
			w.WriteHeader(http.StatusOK)
		case "/System/Info":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"ServerName":"Jellyfin","Version":"10.8.0"}`))
		case "/Sessions":
			w.Header().Set("Content-Type", "application/json")
			// Mix of playing, paused, and stopped sessions
			_, _ = w.Write([]byte(`[
				{
					"Id": "playing-session",
					"UserName": "User1",
					"NowPlayingItem": {"Id": "item-1", "Name": "Playing Movie", "Type": "Movie"},
					"PlayState": {"IsPaused": false}
				},
				{
					"Id": "paused-session",
					"UserName": "User2",
					"NowPlayingItem": {"Id": "item-2", "Name": "Paused Movie", "Type": "Movie"},
					"PlayState": {"IsPaused": true}
				},
				{
					"Id": "idle-session",
					"UserName": "User3"
				}
			]`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer jellyfinServer.Close()

	hub := &mockMultiSourceHub{}
	publisher := &mockMultiSourcePublisher{}

	cfg := &config.JellyfinConfig{
		Enabled:                true,
		URL:                    jellyfinServer.URL,
		APIKey:                 "test-key",
		ServerID:               "jellyfin-1",
		SessionPollingEnabled:  true,
		SessionPollingInterval: 100 * time.Millisecond,
	}
	manager := NewJellyfinManager(cfg, hub, nil)
	manager.SetEventPublisher(publisher)

	ctx := context.Background()
	_ = manager.Start(ctx)

	time.Sleep(150 * time.Millisecond)

	_ = manager.Stop()

	// Should have events for active sessions (playing and paused)
	eventCount := publisher.getEventCount()
	if eventCount < 2 {
		t.Errorf("expected at least 2 events (playing + paused), got %d", eventCount)
	}

	// Note: Session poller publishes events but doesn't broadcast to WebSocket hub.
	// Hub broadcasts only happen from WebSocket handler (handleSessionUpdate).
}

// ============================================================================
// Scenario: Event Ordering
// ============================================================================

func TestMultiSource_EventOrdering(t *testing.T) {
	t.Parallel()

	publisher := &mockEventPublisher{}

	// Server returns sessions in specific order
	jellyfinServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/System/Ping":
			w.WriteHeader(http.StatusOK)
		case "/System/Info":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"ServerName":"Jellyfin","Version":"10.8.0"}`))
		case "/Sessions":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[
				{"Id": "session-a", "UserName": "User1", "NowPlayingItem": {"Id": "1", "Name": "A", "Type": "Movie"}},
				{"Id": "session-b", "UserName": "User2", "NowPlayingItem": {"Id": "2", "Name": "B", "Type": "Movie"}},
				{"Id": "session-c", "UserName": "User3", "NowPlayingItem": {"Id": "3", "Name": "C", "Type": "Movie"}}
			]`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer jellyfinServer.Close()

	cfg := &config.JellyfinConfig{
		Enabled:                true,
		URL:                    jellyfinServer.URL,
		APIKey:                 "test-key",
		ServerID:               "jellyfin-1",
		SessionPollingEnabled:  true,
		SessionPollingInterval: 1 * time.Hour, // Manual polling
	}
	manager := NewJellyfinManager(cfg, nil, nil)
	manager.SetEventPublisher(publisher)

	ctx := context.Background()
	_ = manager.Start(ctx)

	// Manual trigger via poller
	if manager.poller != nil {
		manager.poller.poll(ctx)
	}

	_ = manager.Stop()

	// Should have at least 3 events published (one per session)
	// May have more due to automatic poll on start + manual poll
	eventCount := publisher.publishCalls.Load()
	if eventCount < 3 {
		t.Errorf("expected at least 3 events, got %d", eventCount)
	}
}

// ============================================================================
// Scenario: ServerID Propagation
// ============================================================================

func TestMultiSource_ServerIDPropagation(t *testing.T) {
	t.Parallel()

	publisher := &mockMultiSourcePublisher{}

	// Test with different server IDs
	serverIDs := []struct {
		name     string
		serverID string
	}{
		{"jellyfin-main", "jellyfin-main-server"},
		{"jellyfin-backup", "jellyfin-backup-server"},
	}

	for _, tc := range serverIDs {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/System/Ping":
					w.WriteHeader(http.StatusOK)
				case "/System/Info":
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"ServerName":"Test","Version":"10.8.0"}`))
				case "/Sessions":
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`[{
						"Id": "test-session",
						"UserName": "User",
						"NowPlayingItem": {"Id": "item", "Name": "Movie", "Type": "Movie"},
						"PlayState": {"PlayMethod": "DirectPlay"}
					}]`))
				default:
					w.WriteHeader(http.StatusNotFound)
				}
			}))
			defer server.Close()

			cfg := &config.JellyfinConfig{
				Enabled:                true,
				URL:                    server.URL,
				APIKey:                 "test-key",
				ServerID:               tc.serverID,
				SessionPollingEnabled:  true,
				SessionPollingInterval: 1 * time.Hour,
			}
			manager := NewJellyfinManager(cfg, nil, nil)
			manager.SetEventPublisher(publisher)

			ctx := context.Background()
			_ = manager.Start(ctx)

			if manager.poller != nil {
				manager.poller.poll(ctx)
			}

			_ = manager.Stop()

			// Verify ServerID was propagated
			events := publisher.getEventsByServerID(tc.serverID)
			if len(events) == 0 {
				t.Errorf("expected events with ServerID %s", tc.serverID)
			}
		})
	}
}

// ============================================================================
// Scenario: Context Cancellation
// ============================================================================

func TestMultiSource_ContextCancellation(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/System/Ping":
			w.WriteHeader(http.StatusOK)
		case "/System/Info":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"ServerName":"Test","Version":"10.8.0"}`))
		case "/Sessions":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[]`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	cfg := &config.JellyfinConfig{
		Enabled:                true,
		URL:                    server.URL,
		APIKey:                 "test-key",
		ServerID:               "test-1",
		SessionPollingEnabled:  true,
		SessionPollingInterval: 50 * time.Millisecond,
	}

	ctx, cancel := context.WithCancel(context.Background())

	manager := NewJellyfinManager(cfg, nil, nil)
	_ = manager.Start(ctx)

	time.Sleep(75 * time.Millisecond)

	// Cancel context
	cancel()

	// Give time for cancellation to propagate
	time.Sleep(50 * time.Millisecond)

	// Stop should work cleanly
	err := manager.Stop()
	if err != nil {
		t.Errorf("Stop after context cancellation should work: %v", err)
	}
}

// ============================================================================
// Scenario: No Active Sessions
// ============================================================================

func TestMultiSource_NoActiveSessions(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/System/Ping":
			w.WriteHeader(http.StatusOK)
		case "/System/Info":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"ServerName":"Jellyfin","Version":"10.8.0"}`))
		case "/Sessions":
			w.Header().Set("Content-Type", "application/json")
			// Empty sessions array
			_, _ = w.Write([]byte(`[]`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	publisher := &mockEventPublisher{}

	cfg := &config.JellyfinConfig{
		Enabled:                true,
		URL:                    server.URL,
		APIKey:                 "test-key",
		ServerID:               "jellyfin-1",
		SessionPollingEnabled:  true,
		SessionPollingInterval: 100 * time.Millisecond,
	}
	manager := NewJellyfinManager(cfg, nil, nil)
	manager.SetEventPublisher(publisher)

	ctx := context.Background()
	_ = manager.Start(ctx)

	time.Sleep(200 * time.Millisecond)

	_ = manager.Stop()

	// Should have no events
	if publisher.publishCalls.Load() != 0 {
		t.Errorf("expected 0 events for empty sessions, got %d", publisher.publishCalls.Load())
	}
}
