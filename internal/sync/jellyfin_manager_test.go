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

// managerMockWebSocketHub is a mock implementation of WebSocketHub for manager testing
type managerMockWebSocketHub struct {
	broadcastCount int32
	lastMessage    interface{}
}

func (m *managerMockWebSocketHub) BroadcastJSON(_ string, data interface{}) {
	atomic.AddInt32(&m.broadcastCount, 1)
	m.lastMessage = data
}

// managerMockUserResolver is a mock implementation of UserResolver for manager testing
type managerMockUserResolver struct {
	resolvedCount  int32
	resolvedUserID int
}

func (m *managerMockUserResolver) ResolveUserID(_ context.Context, _, _, _ string, _, _ *string) (int, error) {
	atomic.AddInt32(&m.resolvedCount, 1)
	return m.resolvedUserID, nil
}

// ============================================================================
// Constructor Tests
// ============================================================================

func TestNewJellyfinManager(t *testing.T) {
	t.Run("nil config", func(t *testing.T) {
		manager := NewJellyfinManager(nil, nil, nil)
		if manager != nil {
			t.Error("expected nil manager for nil config")
		}
	})

	t.Run("disabled config", func(t *testing.T) {
		cfg := &config.JellyfinConfig{
			Enabled: false,
		}
		manager := NewJellyfinManager(cfg, nil, nil)
		if manager != nil {
			t.Error("expected nil manager for disabled config")
		}
	})

	t.Run("enabled config", func(t *testing.T) {
		cfg := &config.JellyfinConfig{
			Enabled:  true,
			URL:      "http://localhost:8096",
			APIKey:   "test-key",
			ServerID: "server-1",
		}
		manager := NewJellyfinManager(cfg, nil, nil)
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
		cfg := &config.JellyfinConfig{
			Enabled: true,
			URL:     "http://localhost:8096",
			APIKey:  "test-key",
		}
		hub := &managerMockWebSocketHub{}
		manager := NewJellyfinManager(cfg, hub, nil)
		if manager == nil {
			t.Fatal("expected non-nil manager")
		}
		if manager.wsHub != hub {
			t.Error("wsHub not set correctly")
		}
	})
}

// ============================================================================
// SetEventPublisher Tests
// ============================================================================

func TestJellyfinManager_SetEventPublisher(t *testing.T) {
	cfg := &config.JellyfinConfig{
		Enabled: true,
		URL:     "http://localhost:8096",
		APIKey:  "test-key",
	}
	manager := NewJellyfinManager(cfg, nil, nil)

	publisher := &mockEventPublisher{}
	manager.SetEventPublisher(publisher)

	if manager.eventPublisher != publisher {
		t.Error("eventPublisher not set correctly")
	}
}

// ============================================================================
// ServerID Tests
// ============================================================================

func TestJellyfinManager_ServerID(t *testing.T) {
	t.Run("nil manager", func(t *testing.T) {
		var m *JellyfinManager
		if m.ServerID() != "" {
			t.Error("expected empty string for nil manager")
		}
	})

	t.Run("manager with server ID", func(t *testing.T) {
		cfg := &config.JellyfinConfig{
			Enabled:  true,
			URL:      "http://localhost:8096",
			APIKey:   "test-key",
			ServerID: "jellyfin-main",
		}
		manager := NewJellyfinManager(cfg, nil, nil)
		if manager.ServerID() != "jellyfin-main" {
			t.Errorf("ServerID() = %s, want jellyfin-main", manager.ServerID())
		}
	})

	t.Run("manager without server ID", func(t *testing.T) {
		cfg := &config.JellyfinConfig{
			Enabled: true,
			URL:     "http://localhost:8096",
			APIKey:  "test-key",
		}
		manager := NewJellyfinManager(cfg, nil, nil)
		if manager.ServerID() != "" {
			t.Errorf("ServerID() = %s, want empty string", manager.ServerID())
		}
	})
}

// ============================================================================
// Start/Stop Tests
// ============================================================================

func TestJellyfinManager_Start(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/System/Ping":
			w.WriteHeader(http.StatusOK)
		case "/System/Info":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"ServerName":"Test Server","Version":"10.8.0","Id":"test-id"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	t.Run("nil manager", func(t *testing.T) {
		var m *JellyfinManager
		err := m.Start(context.Background())
		if err != nil {
			t.Errorf("Start() on nil manager should return nil, got %v", err)
		}
	})

	t.Run("basic start", func(t *testing.T) {
		cfg := &config.JellyfinConfig{
			Enabled:               true,
			URL:                   server.URL,
			APIKey:                "test-key",
			RealtimeEnabled:       false,
			SessionPollingEnabled: false,
		}
		manager := NewJellyfinManager(cfg, nil, nil)

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
				_, _ = w.Write([]byte(`{"ServerName":"Test Server","Version":"10.8.0"}`))
			case "/Sessions":
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`[]`))
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer sessServer.Close()

		cfg := &config.JellyfinConfig{
			Enabled:                true,
			URL:                    sessServer.URL,
			APIKey:                 "test-key",
			RealtimeEnabled:        false,
			SessionPollingEnabled:  true,
			SessionPollingInterval: 100 * time.Millisecond,
		}
		manager := NewJellyfinManager(cfg, nil, nil)

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
				_, _ = w.Write([]byte(`{"ServerName":"Test","Version":"10.8.0"}`))
			case "/Sessions":
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`[]`))
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer sessServer.Close()

		cfg := &config.JellyfinConfig{
			Enabled:                true,
			URL:                    sessServer.URL,
			APIKey:                 "test-key",
			RealtimeEnabled:        false,
			SessionPollingEnabled:  true,
			SessionPollingInterval: 1 * time.Millisecond, // Too low - should be capped to 10s
		}
		manager := NewJellyfinManager(cfg, nil, nil)

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
}

func TestJellyfinManager_Stop(t *testing.T) {
	t.Run("nil manager", func(t *testing.T) {
		var m *JellyfinManager
		err := m.Stop()
		if err != nil {
			t.Errorf("Stop() on nil manager should return nil, got %v", err)
		}
	})

	t.Run("stop without start", func(t *testing.T) {
		cfg := &config.JellyfinConfig{
			Enabled: true,
			URL:     "http://localhost:8096",
			APIKey:  "test-key",
		}
		manager := NewJellyfinManager(cfg, nil, nil)

		err := manager.Stop()
		if err != nil {
			t.Errorf("Stop() without Start() should not error: %v", err)
		}
	})
}

// ============================================================================
// Session Handling Tests
// ============================================================================

func TestJellyfinManager_HandleSessionUpdate(t *testing.T) {
	cfg := &config.JellyfinConfig{
		Enabled:  true,
		URL:      "http://localhost:8096",
		APIKey:   "test-key",
		ServerID: "jellyfin-1",
	}

	hub := &managerMockWebSocketHub{}
	publisher := &mockEventPublisher{}

	manager := NewJellyfinManager(cfg, hub, nil)
	manager.SetEventPublisher(publisher)

	// Create test sessions
	sessions := []models.JellyfinSession{
		{
			ID:       "session-1",
			UserID:   "user-abc",
			UserName: "TestUser",
			NowPlayingItem: &models.JellyfinNowPlayingItem{
				ID:   "item-123",
				Name: "Test Movie",
				Type: "Movie",
			},
			PlayState: &models.JellyfinPlayState{
				IsPaused:   false,
				PlayMethod: "DirectPlay",
			},
		},
		{
			ID:       "session-2",
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

func TestJellyfinManager_HandleNewSession(t *testing.T) {
	cfg := &config.JellyfinConfig{
		Enabled:  true,
		URL:      "http://localhost:8096",
		APIKey:   "test-key",
		ServerID: "jellyfin-1",
	}

	publisher := &mockEventPublisher{}
	manager := NewJellyfinManager(cfg, nil, nil)
	manager.SetEventPublisher(publisher)

	session := &models.JellyfinSession{
		ID:       "session-123",
		UserID:   "user-xyz",
		UserName: "MovieWatcher",
		NowPlayingItem: &models.JellyfinNowPlayingItem{
			ID:           "item-456",
			Name:         "Inception",
			Type:         "Movie",
			RunTimeTicks: 88800000000,
		},
		PlayState: &models.JellyfinPlayState{
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

	if event.SessionKey != "session-123" {
		t.Errorf("SessionKey = %s, want session-123", event.SessionKey)
	}
	if *event.ServerID != "jellyfin-1" {
		t.Errorf("ServerID = %s, want jellyfin-1", *event.ServerID)
	}
}

func TestJellyfinManager_PublishSessionWithUserResolver(t *testing.T) {
	cfg := &config.JellyfinConfig{
		Enabled:  true,
		URL:      "http://localhost:8096",
		APIKey:   "test-key",
		ServerID: "jellyfin-main",
	}

	publisher := &mockEventPublisher{}
	userResolver := &managerMockUserResolver{resolvedUserID: 42}
	manager := NewJellyfinManager(cfg, nil, userResolver)
	manager.SetEventPublisher(publisher)

	session := &models.JellyfinSession{
		ID:       "session-abc",
		UserID:   "user-uuid-123",
		UserName: "TestUser",
		NowPlayingItem: &models.JellyfinNowPlayingItem{
			ID:           "item-xyz",
			Name:         "Movie",
			Type:         "Movie",
			RunTimeTicks: 1000000000,
		},
		PlayState: &models.JellyfinPlayState{
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

func TestJellyfinManager_PublishSessionNoPublisher(t *testing.T) {
	cfg := &config.JellyfinConfig{
		Enabled: true,
		URL:     "http://localhost:8096",
		APIKey:  "test-key",
	}

	manager := NewJellyfinManager(cfg, nil, nil)
	// Don't set publisher

	session := &models.JellyfinSession{
		ID: "session-123",
		NowPlayingItem: &models.JellyfinNowPlayingItem{
			ID:           "item-456",
			Name:         "Test",
			Type:         "Movie",
			RunTimeTicks: 1000000000,
		},
		PlayState: &models.JellyfinPlayState{
			PlayMethod: "DirectPlay",
		},
	}

	// Should not panic
	manager.publishSession(session)
}

func TestJellyfinManager_PublishSessionNilEvent(t *testing.T) {
	cfg := &config.JellyfinConfig{
		Enabled: true,
		URL:     "http://localhost:8096",
		APIKey:  "test-key",
	}

	publisher := &mockEventPublisher{}
	manager := NewJellyfinManager(cfg, nil, nil)
	manager.SetEventPublisher(publisher)

	// Session without NowPlayingItem - will result in nil event
	session := &models.JellyfinSession{
		ID:       "session-123",
		UserName: "Test",
	}

	manager.publishSession(session)

	// Should not publish nil event
	if publisher.publishCalls.Load() != 0 {
		t.Errorf("publish count = %d, want 0 (nil event)", publisher.publishCalls.Load())
	}
}

// ============================================================================
// Playstate Handling Tests
// ============================================================================

func TestJellyfinManager_HandlePlayStateChange(t *testing.T) {
	cfg := &config.JellyfinConfig{
		Enabled: true,
		URL:     "http://localhost:8096",
		APIKey:  "test-key",
	}

	hub := &managerMockWebSocketHub{}
	manager := NewJellyfinManager(cfg, hub, nil)

	manager.handlePlayStateChange("session-123", "Pause")

	if hub.broadcastCount != 1 {
		t.Errorf("broadcast count = %d, want 1", hub.broadcastCount)
	}

	msg, ok := hub.lastMessage.(map[string]interface{})
	if !ok {
		t.Fatal("expected map message")
	}
	if msg["session_id"] != "session-123" {
		t.Errorf("session_id = %v, want session-123", msg["session_id"])
	}
	if msg["command"] != "Pause" {
		t.Errorf("command = %v, want Pause", msg["command"])
	}
}

func TestJellyfinManager_HandlePlayStateChangeNoHub(t *testing.T) {
	cfg := &config.JellyfinConfig{
		Enabled: true,
		URL:     "http://localhost:8096",
		APIKey:  "test-key",
	}

	manager := NewJellyfinManager(cfg, nil, nil)

	// Should not panic without wsHub
	manager.handlePlayStateChange("session-123", "Stop")
}

// ============================================================================
// User Data Changed Tests
// ============================================================================

func TestJellyfinManager_HandleUserDataChanged(t *testing.T) {
	cfg := &config.JellyfinConfig{
		Enabled: true,
		URL:     "http://localhost:8096",
		APIKey:  "test-key",
	}

	manager := NewJellyfinManager(cfg, nil, nil)

	// Should not panic
	manager.handleUserDataChanged("user-123", map[string]interface{}{"foo": "bar"})
}

// ============================================================================
// Session State Tests
// ============================================================================

func TestJellyfinManager_GetSessionState(t *testing.T) {
	cfg := &config.JellyfinConfig{
		Enabled: true,
		URL:     "http://localhost:8096",
		APIKey:  "test-key",
	}
	manager := NewJellyfinManager(cfg, nil, nil)

	tests := []struct {
		name    string
		session *models.JellyfinSession
		want    string
	}{
		{
			name: "playing",
			session: &models.JellyfinSession{
				NowPlayingItem: &models.JellyfinNowPlayingItem{Name: "Test"},
				PlayState:      &models.JellyfinPlayState{IsPaused: false},
			},
			want: "playing",
		},
		{
			name: "paused",
			session: &models.JellyfinSession{
				NowPlayingItem: &models.JellyfinNowPlayingItem{Name: "Test"},
				PlayState:      &models.JellyfinPlayState{IsPaused: true},
			},
			want: "paused",
		},
		{
			name: "stopped (no playstate)",
			session: &models.JellyfinSession{
				NowPlayingItem: &models.JellyfinNowPlayingItem{Name: "Test"},
			},
			want: "stopped",
		},
		{
			name:    "stopped (no item)",
			session: &models.JellyfinSession{},
			want:    "stopped",
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
