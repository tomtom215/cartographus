// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package sync

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/tomtom215/cartographus/internal/config"
	"github.com/tomtom215/cartographus/internal/models"
)

// ============================================================================
// Constructor Tests
// ============================================================================

func TestNewEmbyManager(t *testing.T) {
	t.Run("nil config", func(t *testing.T) {
		manager := NewEmbyManager(nil, nil, nil)
		if manager != nil {
			t.Error("expected nil manager for nil config")
		}
	})

	t.Run("disabled config", func(t *testing.T) {
		cfg := &config.EmbyConfig{
			Enabled: false,
		}
		manager := NewEmbyManager(cfg, nil, nil)
		if manager != nil {
			t.Error("expected nil manager for disabled config")
		}
	})

	t.Run("enabled config", func(t *testing.T) {
		cfg := &config.EmbyConfig{
			Enabled:  true,
			URL:      "http://localhost:8096",
			APIKey:   "test-key",
			ServerID: "server-1",
		}
		manager := NewEmbyManager(cfg, nil, nil)
		if manager == nil {
			t.Fatal("expected non-nil manager for enabled config")
		}
		if manager.client == nil {
			t.Error("expected client to be initialized")
		}
		if manager.cfg != cfg {
			t.Error("config not set correctly")
		}
	})

	t.Run("with websocket hub", func(t *testing.T) {
		cfg := &config.EmbyConfig{
			Enabled: true,
			URL:     "http://localhost:8096",
			APIKey:  "test-key",
		}
		hub := &managerMockWebSocketHub{}
		manager := NewEmbyManager(cfg, hub, nil)
		if manager == nil {
			t.Fatal("expected non-nil manager")
		}
		if manager.wsHub != hub {
			t.Error("wsHub not set correctly")
		}
	})

	t.Run("with user resolver", func(t *testing.T) {
		cfg := &config.EmbyConfig{
			Enabled: true,
			URL:     "http://localhost:8096",
			APIKey:  "test-key",
		}
		resolver := &managerMockUserResolver{resolvedUserID: 100}
		manager := NewEmbyManager(cfg, nil, resolver)
		if manager == nil {
			t.Fatal("expected non-nil manager")
		}
		if manager.userResolver != resolver {
			t.Error("userResolver not set correctly")
		}
	})
}

// ============================================================================
// SetEventPublisher Tests
// ============================================================================

func TestEmbyManager_SetEventPublisher(t *testing.T) {
	cfg := &config.EmbyConfig{
		Enabled: true,
		URL:     "http://localhost:8096",
		APIKey:  "test-key",
	}
	manager := NewEmbyManager(cfg, nil, nil)

	publisher := &mockEventPublisher{}
	manager.SetEventPublisher(publisher)

	if manager.eventPublisher != publisher {
		t.Error("eventPublisher not set correctly")
	}
}

// ============================================================================
// ServerID Tests
// ============================================================================

func TestEmbyManager_ServerID(t *testing.T) {
	t.Run("nil manager", func(t *testing.T) {
		var m *EmbyManager
		if m.ServerID() != "" {
			t.Error("expected empty string for nil manager")
		}
	})

	t.Run("manager with server ID", func(t *testing.T) {
		cfg := &config.EmbyConfig{
			Enabled:  true,
			URL:      "http://localhost:8096",
			APIKey:   "test-key",
			ServerID: "emby-main",
		}
		manager := NewEmbyManager(cfg, nil, nil)
		if manager.ServerID() != "emby-main" {
			t.Errorf("ServerID() = %s, want emby-main", manager.ServerID())
		}
	})

	t.Run("manager without server ID", func(t *testing.T) {
		cfg := &config.EmbyConfig{
			Enabled: true,
			URL:     "http://localhost:8096",
			APIKey:  "test-key",
		}
		manager := NewEmbyManager(cfg, nil, nil)
		if manager.ServerID() != "" {
			t.Errorf("ServerID() = %s, want empty string", manager.ServerID())
		}
	})

	t.Run("manager with nil config", func(t *testing.T) {
		cfg := &config.EmbyConfig{
			Enabled: true,
			URL:     "http://localhost:8096",
			APIKey:  "test-key",
		}
		manager := NewEmbyManager(cfg, nil, nil)
		// Manually nil out config to test defensive coding
		manager.cfg = nil
		if manager.ServerID() != "" {
			t.Errorf("ServerID() = %s, want empty string for nil config", manager.ServerID())
		}
	})
}

// ============================================================================
// Start/Stop Tests
// ============================================================================

func TestEmbyManager_Start(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/System/Ping":
			w.WriteHeader(http.StatusOK)
		case "/System/Info":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"ServerName":"Emby Test Server","Version":"4.7.0","Id":"emby-test-id"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	t.Run("nil manager", func(t *testing.T) {
		var m *EmbyManager
		err := m.Start(context.Background())
		if err != nil {
			t.Errorf("Start() on nil manager should return nil, got %v", err)
		}
	})

	t.Run("basic start", func(t *testing.T) {
		cfg := &config.EmbyConfig{
			Enabled:               true,
			URL:                   server.URL,
			APIKey:                "test-key",
			RealtimeEnabled:       false,
			SessionPollingEnabled: false,
		}
		manager := NewEmbyManager(cfg, nil, nil)

		ctx := context.Background()
		err := manager.Start(ctx)
		if err != nil {
			t.Errorf("Start() failed: %v", err)
		}

		// Stop cleanup
		err = manager.Stop()
		if err != nil {
			t.Errorf("Stop() failed: %v", err)
		}
	})

	t.Run("start with session polling", func(t *testing.T) {
		// Create mock server that handles session endpoint
		sessServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/System/Ping":
				w.WriteHeader(http.StatusOK)
			case "/System/Info":
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"ServerName":"Emby Test Server","Version":"4.7.0"}`))
			case "/Sessions":
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`[]`))
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer sessServer.Close()

		cfg := &config.EmbyConfig{
			Enabled:                true,
			URL:                    sessServer.URL,
			APIKey:                 "test-key",
			RealtimeEnabled:        false,
			SessionPollingEnabled:  true,
			SessionPollingInterval: 100 * time.Millisecond,
		}
		manager := NewEmbyManager(cfg, nil, nil)

		ctx := context.Background()
		err := manager.Start(ctx)
		if err != nil {
			t.Errorf("Start() failed: %v", err)
		}

		// Let poller run briefly
		time.Sleep(50 * time.Millisecond)

		// Stop cleanup
		err = manager.Stop()
		if err != nil {
			t.Errorf("Stop() failed: %v", err)
		}
	})

	t.Run("start with low polling interval", func(t *testing.T) {
		sessServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/System/Ping":
				w.WriteHeader(http.StatusOK)
			case "/System/Info":
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"ServerName":"Emby","Version":"4.7.0"}`))
			case "/Sessions":
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`[]`))
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer sessServer.Close()

		cfg := &config.EmbyConfig{
			Enabled:                true,
			URL:                    sessServer.URL,
			APIKey:                 "test-key",
			RealtimeEnabled:        false,
			SessionPollingEnabled:  true,
			SessionPollingInterval: 1 * time.Millisecond, // Too low - should be capped to 10s
		}
		manager := NewEmbyManager(cfg, nil, nil)

		ctx := context.Background()
		err := manager.Start(ctx)
		if err != nil {
			t.Errorf("Start() failed: %v", err)
		}

		// Check that poller was created with corrected interval
		if manager.poller == nil {
			t.Error("poller should be created")
		}

		err = manager.Stop()
		if err != nil {
			t.Errorf("Stop() failed: %v", err)
		}
	})

	t.Run("start with ping failure", func(t *testing.T) {
		// Create mock server that fails ping
		failServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer failServer.Close()

		cfg := &config.EmbyConfig{
			Enabled:               true,
			URL:                   failServer.URL,
			APIKey:                "test-key",
			RealtimeEnabled:       false,
			SessionPollingEnabled: false,
		}
		manager := NewEmbyManager(cfg, nil, nil)

		ctx := context.Background()
		// Start should succeed even if ping fails (server may become available later)
		err := manager.Start(ctx)
		if err != nil {
			t.Errorf("Start() should not fail on ping failure: %v", err)
		}

		err = manager.Stop()
		if err != nil {
			t.Errorf("Stop() failed: %v", err)
		}
	})
}

func TestEmbyManager_Stop(t *testing.T) {
	t.Run("nil manager", func(t *testing.T) {
		var m *EmbyManager
		err := m.Stop()
		if err != nil {
			t.Errorf("Stop() on nil manager should return nil, got %v", err)
		}
	})

	t.Run("stop without start", func(t *testing.T) {
		cfg := &config.EmbyConfig{
			Enabled: true,
			URL:     "http://localhost:8096",
			APIKey:  "test-key",
		}
		manager := NewEmbyManager(cfg, nil, nil)

		err := manager.Stop()
		if err != nil {
			t.Errorf("Stop() without Start() should not error: %v", err)
		}
	})
}

// ============================================================================
// Session Handling Tests
// ============================================================================

func TestEmbyManager_HandleSessionUpdate(t *testing.T) {
	cfg := &config.EmbyConfig{
		Enabled:  true,
		URL:      "http://localhost:8096",
		APIKey:   "test-key",
		ServerID: "emby-1",
	}

	hub := &managerMockWebSocketHub{}
	publisher := &mockEventPublisher{}

	manager := NewEmbyManager(cfg, hub, nil)
	manager.SetEventPublisher(publisher)

	// Create test sessions
	sessions := []models.EmbySession{
		{
			ID:       "emby-session-1",
			UserID:   "user-abc",
			UserName: "TestUser",
			NowPlayingItem: &models.EmbyNowPlayingItem{
				ID:   "item-123",
				Name: "The Matrix",
				Type: "Movie",
			},
			PlayState: &models.EmbyPlayState{
				IsPaused:   false,
				PlayMethod: "DirectPlay",
			},
		},
		{
			ID:       "emby-session-2",
			UserName: "IdleUser",
			// No NowPlayingItem - should be skipped
		},
	}

	manager.handleSessionUpdate(sessions)

	// Should have broadcast for active session
	if hub.broadcastCount != 1 {
		t.Errorf("broadcast count = %d, want 1", hub.broadcastCount)
	}

	// Should have published event for active session
	if publisher.publishCalls.Load() != 1 {
		t.Errorf("publish count = %d, want 1", publisher.publishCalls.Load())
	}
}

func TestEmbyManager_HandleSessionUpdateMultipleActive(t *testing.T) {
	cfg := &config.EmbyConfig{
		Enabled:  true,
		URL:      "http://localhost:8096",
		APIKey:   "test-key",
		ServerID: "emby-1",
	}

	hub := &managerMockWebSocketHub{}
	publisher := &mockEventPublisher{}

	manager := NewEmbyManager(cfg, hub, nil)
	manager.SetEventPublisher(publisher)

	// Create multiple active sessions
	sessions := []models.EmbySession{
		{
			ID:       "emby-session-1",
			UserID:   "user-abc",
			UserName: "User1",
			NowPlayingItem: &models.EmbyNowPlayingItem{
				ID:   "item-1",
				Name: "Movie 1",
				Type: "Movie",
			},
			PlayState: &models.EmbyPlayState{IsPaused: false},
		},
		{
			ID:       "emby-session-2",
			UserID:   "user-def",
			UserName: "User2",
			NowPlayingItem: &models.EmbyNowPlayingItem{
				ID:   "item-2",
				Name: "Movie 2",
				Type: "Movie",
			},
			PlayState: &models.EmbyPlayState{IsPaused: false},
		},
		{
			ID:       "emby-session-3",
			UserID:   "user-ghi",
			UserName: "User3",
			NowPlayingItem: &models.EmbyNowPlayingItem{
				ID:   "item-3",
				Name: "Episode 1",
				Type: "Episode",
			},
			PlayState: &models.EmbyPlayState{IsPaused: true},
		},
	}

	manager.handleSessionUpdate(sessions)

	// Should have broadcast for all 3 active sessions
	if hub.broadcastCount != 3 {
		t.Errorf("broadcast count = %d, want 3", hub.broadcastCount)
	}

	// Should have published event for all 3 sessions
	if publisher.publishCalls.Load() != 3 {
		t.Errorf("publish count = %d, want 3", publisher.publishCalls.Load())
	}
}

func TestEmbyManager_HandleNewSession(t *testing.T) {
	cfg := &config.EmbyConfig{
		Enabled:  true,
		URL:      "http://localhost:8096",
		APIKey:   "test-key",
		ServerID: "emby-1",
	}

	publisher := &mockEventPublisher{}
	manager := NewEmbyManager(cfg, nil, nil)
	manager.SetEventPublisher(publisher)

	session := &models.EmbySession{
		ID:       "emby-session-123",
		UserID:   "user-xyz",
		UserName: "MovieWatcher",
		NowPlayingItem: &models.EmbyNowPlayingItem{
			ID:           "item-456",
			Name:         "Inception",
			Type:         "Movie",
			RunTimeTicks: 88800000000,
		},
		PlayState: &models.EmbyPlayState{
			PositionTicks: 36000000000,
			IsPaused:      false,
			PlayMethod:    "DirectPlay",
		},
	}

	manager.handleNewSession(session)

	if publisher.publishCalls.Load() != 1 {
		t.Errorf("publish count = %d, want 1", publisher.publishCalls.Load())
	}

	event := publisher.getLastEvent()
	if event == nil {
		t.Fatal("expected event to be published")
	}

	if event.SessionKey != "emby-session-123" {
		t.Errorf("SessionKey = %s, want emby-session-123", event.SessionKey)
	}
	if *event.ServerID != "emby-1" {
		t.Errorf("ServerID = %s, want emby-1", *event.ServerID)
	}
}

func TestEmbyManager_PublishSessionWithUserResolver(t *testing.T) {
	cfg := &config.EmbyConfig{
		Enabled:  true,
		URL:      "http://localhost:8096",
		APIKey:   "test-key",
		ServerID: "emby-main",
	}

	publisher := &mockEventPublisher{}
	userResolver := &managerMockUserResolver{resolvedUserID: 42}
	manager := NewEmbyManager(cfg, nil, userResolver)
	manager.SetEventPublisher(publisher)

	session := &models.EmbySession{
		ID:       "emby-session-abc",
		UserID:   "user-uuid-123",
		UserName: "TestUser",
		NowPlayingItem: &models.EmbyNowPlayingItem{
			ID:           "item-xyz",
			Name:         "Movie",
			Type:         "Movie",
			RunTimeTicks: 1000000000,
		},
		PlayState: &models.EmbyPlayState{
			PlayMethod: "DirectPlay",
		},
	}

	manager.publishSession(session)

	if userResolver.resolvedCount != 1 {
		t.Errorf("user resolver called %d times, want 1", userResolver.resolvedCount)
	}

	event := publisher.getLastEvent()
	if event == nil {
		t.Fatal("expected event to be published")
	}

	if event.UserID != 42 {
		t.Errorf("UserID = %d, want 42", event.UserID)
	}
}

func TestEmbyManager_PublishSessionNoPublisher(t *testing.T) {
	cfg := &config.EmbyConfig{
		Enabled: true,
		URL:     "http://localhost:8096",
		APIKey:  "test-key",
	}

	manager := NewEmbyManager(cfg, nil, nil)
	// Don't set publisher

	session := &models.EmbySession{
		ID: "emby-session-123",
		NowPlayingItem: &models.EmbyNowPlayingItem{
			ID:           "item-456",
			Name:         "Test",
			Type:         "Movie",
			RunTimeTicks: 1000000000,
		},
		PlayState: &models.EmbyPlayState{
			PlayMethod: "DirectPlay",
		},
	}

	// Should not panic
	manager.publishSession(session)
}

func TestEmbyManager_PublishSessionNilEvent(t *testing.T) {
	cfg := &config.EmbyConfig{
		Enabled: true,
		URL:     "http://localhost:8096",
		APIKey:  "test-key",
	}

	publisher := &mockEventPublisher{}
	manager := NewEmbyManager(cfg, nil, nil)
	manager.SetEventPublisher(publisher)

	// Session without NowPlayingItem - will result in nil event
	session := &models.EmbySession{
		ID:       "emby-session-123",
		UserName: "Test",
	}

	manager.publishSession(session)

	// Should not publish nil event
	if publisher.publishCalls.Load() != 0 {
		t.Errorf("publish count = %d, want 0 (nil event)", publisher.publishCalls.Load())
	}
}

func TestEmbyManager_PublishSessionWithoutServerID(t *testing.T) {
	cfg := &config.EmbyConfig{
		Enabled: true,
		URL:     "http://localhost:8096",
		APIKey:  "test-key",
		// No ServerID
	}

	publisher := &mockEventPublisher{}
	manager := NewEmbyManager(cfg, nil, nil)
	manager.SetEventPublisher(publisher)

	session := &models.EmbySession{
		ID:     "emby-session-123",
		UserID: "user-1",
		NowPlayingItem: &models.EmbyNowPlayingItem{
			ID:           "item-456",
			Name:         "Test",
			Type:         "Movie",
			RunTimeTicks: 1000000000,
		},
		PlayState: &models.EmbyPlayState{
			PlayMethod: "DirectPlay",
		},
	}

	manager.publishSession(session)

	if publisher.publishCalls.Load() != 1 {
		t.Errorf("publish count = %d, want 1", publisher.publishCalls.Load())
	}

	// ServerID should not be set when config doesn't have it
	event := publisher.getLastEvent()
	if event.ServerID != nil {
		t.Error("expected ServerID to be nil when not configured")
	}
}

func TestEmbyManager_PublishSessionWithEmptyUserID(t *testing.T) {
	cfg := &config.EmbyConfig{
		Enabled:  true,
		URL:      "http://localhost:8096",
		APIKey:   "test-key",
		ServerID: "emby-1",
	}

	publisher := &mockEventPublisher{}
	userResolver := &managerMockUserResolver{resolvedUserID: 42}
	manager := NewEmbyManager(cfg, nil, userResolver)
	manager.SetEventPublisher(publisher)

	session := &models.EmbySession{
		ID:     "emby-session-123",
		UserID: "", // Empty user ID
		NowPlayingItem: &models.EmbyNowPlayingItem{
			ID:           "item-456",
			Name:         "Test",
			Type:         "Movie",
			RunTimeTicks: 1000000000,
		},
		PlayState: &models.EmbyPlayState{
			PlayMethod: "DirectPlay",
		},
	}

	manager.publishSession(session)

	// User resolver should NOT be called for empty user ID
	if userResolver.resolvedCount != 0 {
		t.Errorf("user resolver called %d times, want 0 for empty user ID", userResolver.resolvedCount)
	}

	if publisher.publishCalls.Load() != 1 {
		t.Errorf("publish count = %d, want 1", publisher.publishCalls.Load())
	}
}

// ============================================================================
// Playstate Handling Tests
// ============================================================================

func TestEmbyManager_HandlePlayStateChange(t *testing.T) {
	cfg := &config.EmbyConfig{
		Enabled: true,
		URL:     "http://localhost:8096",
		APIKey:  "test-key",
	}

	hub := &managerMockWebSocketHub{}
	manager := NewEmbyManager(cfg, hub, nil)

	manager.handlePlayStateChange("emby-session-123", "Pause")

	if hub.broadcastCount != 1 {
		t.Errorf("broadcast count = %d, want 1", hub.broadcastCount)
	}

	msg, ok := hub.lastMessage.(map[string]interface{})
	if !ok {
		t.Fatal("expected map message")
	}
	if msg["session_id"] != "emby-session-123" {
		t.Errorf("session_id = %v, want emby-session-123", msg["session_id"])
	}
	if msg["command"] != "Pause" {
		t.Errorf("command = %v, want Pause", msg["command"])
	}
}

func TestEmbyManager_HandlePlayStateChangeNoHub(t *testing.T) {
	cfg := &config.EmbyConfig{
		Enabled: true,
		URL:     "http://localhost:8096",
		APIKey:  "test-key",
	}

	manager := NewEmbyManager(cfg, nil, nil)

	// Should not panic without wsHub
	manager.handlePlayStateChange("emby-session-123", "Stop")
}

func TestEmbyManager_HandlePlayStateChangeCommands(t *testing.T) {
	cfg := &config.EmbyConfig{
		Enabled: true,
		URL:     "http://localhost:8096",
		APIKey:  "test-key",
	}

	commands := []string{"Play", "Pause", "Stop", "Seek", "Resume"}

	for _, cmd := range commands {
		t.Run(cmd, func(t *testing.T) {
			hub := &managerMockWebSocketHub{}
			manager := NewEmbyManager(cfg, hub, nil)

			manager.handlePlayStateChange("session-123", cmd)

			if hub.broadcastCount != 1 {
				t.Errorf("broadcast count = %d, want 1", hub.broadcastCount)
			}

			msg, ok := hub.lastMessage.(map[string]interface{})
			if !ok {
				t.Fatal("expected map message")
			}
			if msg["command"] != cmd {
				t.Errorf("command = %v, want %s", msg["command"], cmd)
			}
		})
	}
}

// ============================================================================
// User Data Changed Tests
// ============================================================================

func TestEmbyManager_HandleUserDataChanged(t *testing.T) {
	cfg := &config.EmbyConfig{
		Enabled: true,
		URL:     "http://localhost:8096",
		APIKey:  "test-key",
	}

	manager := NewEmbyManager(cfg, nil, nil)

	// Should not panic
	manager.handleUserDataChanged("user-123", map[string]interface{}{"foo": "bar"})
}

func TestEmbyManager_HandleUserDataChangedVariousTypes(t *testing.T) {
	cfg := &config.EmbyConfig{
		Enabled: true,
		URL:     "http://localhost:8096",
		APIKey:  "test-key",
	}

	manager := NewEmbyManager(cfg, nil, nil)

	// Test with various data types - none should panic
	testCases := []struct {
		name   string
		userID string
		data   any
	}{
		{"nil data", "user-1", nil},
		{"map data", "user-2", map[string]interface{}{"key": "value"}},
		{"string data", "user-3", "some string"},
		{"int data", "user-4", 42},
		{"slice data", "user-5", []string{"a", "b", "c"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Should not panic
			manager.handleUserDataChanged(tc.userID, tc.data)
		})
	}
}

// ============================================================================
// Session State Tests
// ============================================================================

func TestEmbyManager_GetSessionState(t *testing.T) {
	cfg := &config.EmbyConfig{
		Enabled: true,
		URL:     "http://localhost:8096",
		APIKey:  "test-key",
	}
	manager := NewEmbyManager(cfg, nil, nil)

	tests := []struct {
		name    string
		session *models.EmbySession
		want    string
	}{
		{
			name: "playing",
			session: &models.EmbySession{
				NowPlayingItem: &models.EmbyNowPlayingItem{Name: "Test"},
				PlayState:      &models.EmbyPlayState{IsPaused: false},
			},
			want: "playing",
		},
		{
			name: "paused",
			session: &models.EmbySession{
				NowPlayingItem: &models.EmbyNowPlayingItem{Name: "Test"},
				PlayState:      &models.EmbyPlayState{IsPaused: true},
			},
			want: "paused",
		},
		{
			name: "stopped (no playstate)",
			session: &models.EmbySession{
				NowPlayingItem: &models.EmbyNowPlayingItem{Name: "Test"},
			},
			want: "stopped",
		},
		{
			name:    "stopped (no item)",
			session: &models.EmbySession{},
			want:    "stopped",
		},
		{
			name: "stopped (nil playstate values)",
			session: &models.EmbySession{
				PlayState: &models.EmbyPlayState{},
			},
			want: "stopped",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := manager.getSessionState(tt.session)
			if got != tt.want {
				t.Errorf("getSessionState() = %s, want %s", got, tt.want)
			}
		})
	}
}

// ============================================================================
// Integration-Style Tests
// ============================================================================

func TestEmbyManager_FullLifecycle(t *testing.T) {
	var requestCount int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		switch r.URL.Path {
		case "/System/Ping":
			w.WriteHeader(http.StatusOK)
		case "/System/Info":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"ServerName":"Emby","Version":"4.7.0","Id":"test-id"}`))
		case "/Sessions":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[{
				"Id": "session-1",
				"UserName": "TestUser",
				"NowPlayingItem": {
					"Id": "item-1",
					"Name": "Test Movie",
					"Type": "Movie"
				},
				"PlayState": {
					"IsPaused": false
				}
			}]`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	hub := &mockWebSocketHub{}
	publisher := &mockEventPublisher{}

	cfg := &config.EmbyConfig{
		Enabled:                true,
		URL:                    server.URL,
		APIKey:                 "test-key",
		ServerID:               "emby-integration-test",
		RealtimeEnabled:        false,
		SessionPollingEnabled:  true,
		SessionPollingInterval: 100 * time.Millisecond,
	}

	manager := NewEmbyManager(cfg, hub, nil)
	manager.SetEventPublisher(publisher)

	ctx := context.Background()
	err := manager.Start(ctx)
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	// Let it run and poll a few times
	time.Sleep(250 * time.Millisecond)

	err = manager.Stop()
	if err != nil {
		t.Fatalf("Stop() failed: %v", err)
	}

	// Verify requests were made
	if atomic.LoadInt32(&requestCount) < 3 {
		t.Errorf("expected at least 3 requests (ping, info, sessions), got %d", requestCount)
	}
}

func TestEmbyManager_HandleSessionUpdateNoHub(t *testing.T) {
	cfg := &config.EmbyConfig{
		Enabled:  true,
		URL:      "http://localhost:8096",
		APIKey:   "test-key",
		ServerID: "emby-1",
	}

	publisher := &mockEventPublisher{}

	// Create manager without WebSocket hub
	manager := NewEmbyManager(cfg, nil, nil)
	manager.SetEventPublisher(publisher)

	sessions := []models.EmbySession{
		{
			ID:       "emby-session-1",
			UserID:   "user-abc",
			UserName: "TestUser",
			NowPlayingItem: &models.EmbyNowPlayingItem{
				ID:   "item-123",
				Name: "The Matrix",
				Type: "Movie",
			},
			PlayState: &models.EmbyPlayState{
				IsPaused:   false,
				PlayMethod: "DirectPlay",
			},
		},
	}

	// Should not panic without wsHub
	manager.handleSessionUpdate(sessions)

	// Should still publish events
	if publisher.publishCalls.Load() != 1 {
		t.Errorf("publish count = %d, want 1", publisher.publishCalls.Load())
	}
}

func TestEmbyManager_HandleSessionUpdateNoPublisher(t *testing.T) {
	cfg := &config.EmbyConfig{
		Enabled:  true,
		URL:      "http://localhost:8096",
		APIKey:   "test-key",
		ServerID: "emby-1",
	}

	hub := &managerMockWebSocketHub{}

	// Create manager without publisher
	manager := NewEmbyManager(cfg, hub, nil)
	// Don't set publisher

	sessions := []models.EmbySession{
		{
			ID:       "emby-session-1",
			UserID:   "user-abc",
			UserName: "TestUser",
			NowPlayingItem: &models.EmbyNowPlayingItem{
				ID:   "item-123",
				Name: "The Matrix",
				Type: "Movie",
			},
			PlayState: &models.EmbyPlayState{
				IsPaused:   false,
				PlayMethod: "DirectPlay",
			},
		},
	}

	// Should not panic without publisher
	manager.handleSessionUpdate(sessions)

	// Should still broadcast to hub
	if hub.broadcastCount != 1 {
		t.Errorf("broadcast count = %d, want 1", hub.broadcastCount)
	}
}
