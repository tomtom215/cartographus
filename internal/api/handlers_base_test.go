// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/tomtom215/cartographus/internal/cache"
	"github.com/tomtom215/cartographus/internal/config"
	"github.com/tomtom215/cartographus/internal/middleware"
	ws "github.com/tomtom215/cartographus/internal/websocket"
)

// TestNewHandler tests the NewHandler constructor
func TestNewHandler(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Server: config.ServerConfig{
			Latitude:  40.7128,
			Longitude: -74.0060,
		},
		API: config.APIConfig{
			DefaultPageSize: 100,
			MaxPageSize:     1000,
		},
		Security: config.SecurityConfig{
			CORSOrigins: []string{"*"},
		},
		Plex: config.PlexConfig{
			OAuthClientID:    "test-client-id",
			OAuthRedirectURI: "http://localhost:3857/callback",
		},
	}

	wsHub := ws.NewHub()
	go wsHub.Run()

	mockClient := &MockTautulliClient{}

	handler := NewHandler(nil, nil, mockClient, cfg, nil, wsHub)

	if handler == nil {
		t.Fatal("NewHandler returned nil")
	}

	if handler.cache == nil {
		t.Error("Expected cache to be initialized")
	}

	if handler.perfMon == nil {
		t.Error("Expected performance monitor to be initialized")
	}

	if handler.plexOAuthClient == nil {
		t.Error("Expected Plex OAuth client to be initialized")
	}

	if handler.startTime.IsZero() {
		t.Error("Expected start time to be set")
	}
}

// TestNewHandler_WithoutPlexOAuth tests NewHandler without OAuth config
func TestNewHandler_WithoutPlexOAuth(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		API: config.APIConfig{
			DefaultPageSize: 100,
			MaxPageSize:     1000,
		},
		Plex: config.PlexConfig{}, // Empty - no OAuth
	}

	wsHub := ws.NewHub()
	go wsHub.Run()

	handler := NewHandler(nil, nil, nil, cfg, nil, wsHub)

	if handler == nil {
		t.Fatal("NewHandler returned nil")
	}

	if handler.plexOAuthClient != nil {
		t.Error("Expected Plex OAuth client to be nil when not configured")
	}
}

// TestCheckWebSocketOrigin tests the WebSocket origin validation
func TestCheckWebSocketOrigin(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		corsOrigins    []string
		requestOrigin  string
		expectedResult bool
	}{
		{
			name:           "no origin header - SECURITY: must reject",
			corsOrigins:    []string{"http://localhost:3857"},
			requestOrigin:  "",
			expectedResult: false, // REJECT: prevents CORS bypass from non-browser clients
		},
		{
			name:           "wildcard origin - allow any",
			corsOrigins:    []string{"*"},
			requestOrigin:  "http://example.com",
			expectedResult: true,
		},
		{
			name:           "exact match - allow",
			corsOrigins:    []string{"http://localhost:3857"},
			requestOrigin:  "http://localhost:3857",
			expectedResult: true,
		},
		{
			name:           "multiple origins - match first",
			corsOrigins:    []string{"http://localhost:3857", "http://example.com"},
			requestOrigin:  "http://localhost:3857",
			expectedResult: true,
		},
		{
			name:           "multiple origins - match second",
			corsOrigins:    []string{"http://localhost:3857", "http://example.com"},
			requestOrigin:  "http://example.com",
			expectedResult: true,
		},
		{
			name:           "origin not in list - reject",
			corsOrigins:    []string{"http://localhost:3857"},
			requestOrigin:  "http://evil.com",
			expectedResult: false,
		},
		{
			name:           "empty allowed origins - reject",
			corsOrigins:    []string{},
			requestOrigin:  "http://example.com",
			expectedResult: false,
		},
		{
			name:           "origin with different port - reject",
			corsOrigins:    []string{"http://localhost:3857"},
			requestOrigin:  "http://localhost:8080",
			expectedResult: false,
		},
		{
			name:           "origin with different protocol - reject",
			corsOrigins:    []string{"http://localhost:3857"},
			requestOrigin:  "https://localhost:3857",
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Security: config.SecurityConfig{
					CORSOrigins: tt.corsOrigins,
				},
			}

			handler := &Handler{
				config: cfg,
			}

			req := httptest.NewRequest(http.MethodGet, "/ws", nil)
			if tt.requestOrigin != "" {
				req.Header.Set("Origin", tt.requestOrigin)
			}

			result := handler.checkWebSocketOrigin(req)

			if result != tt.expectedResult {
				t.Errorf("checkWebSocketOrigin() = %v, want %v", result, tt.expectedResult)
			}
		})
	}
}

// TestGetUpgrader tests the WebSocket upgrader configuration
func TestGetUpgrader(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Security: config.SecurityConfig{
			CORSOrigins: []string{"*"},
		},
	}

	handler := &Handler{
		config: cfg,
	}

	upgrader := handler.getUpgrader()

	if upgrader.ReadBufferSize != 1024 {
		t.Errorf("ReadBufferSize = %d, want 1024", upgrader.ReadBufferSize)
	}

	if upgrader.WriteBufferSize != 1024 {
		t.Errorf("WriteBufferSize = %d, want 1024", upgrader.WriteBufferSize)
	}

	if upgrader.CheckOrigin == nil {
		t.Error("CheckOrigin function should be set")
	}
}

// TestClearCache_LogsMessage tests that ClearCache logs appropriately
func TestClearCache_LogsMessage(t *testing.T) {
	t.Parallel()

	c := cache.New(5 * time.Minute)
	c.Set("test", "value")

	handler := &Handler{
		cache: c,
	}

	// Should not panic
	handler.ClearCache()

	// Verify cache is cleared
	if _, found := c.Get("test"); found {
		t.Error("Cache should be cleared")
	}
}

// TestOnSyncCompleted_NilWSHub tests OnSyncCompleted when wsHub is nil
func TestOnSyncCompleted_NilWSHub(t *testing.T) {
	t.Parallel()

	c := cache.New(5 * time.Minute)
	c.Set("test", "value")

	handler := &Handler{
		cache: c,
		wsHub: nil,
		db:    nil,
	}

	// Should not panic
	handler.OnSyncCompleted(10, 100)

	// Cache should still be cleared
	if _, found := c.Get("test"); found {
		t.Error("Cache should be cleared")
	}
}

// TestOnSyncCompleted_WithWSHub tests OnSyncCompleted with active WebSocket hub
func TestOnSyncCompleted_WithWSHub(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()

	c := cache.New(5 * time.Minute)
	c.Set("analytics", "cached_data")

	wsHub := ws.NewHub()
	go wsHub.Run()

	handler := &Handler{
		cache:     c,
		wsHub:     wsHub,
		db:        db,
		startTime: time.Now(),
		config: &config.Config{
			API: config.APIConfig{
				DefaultPageSize: 100,
				MaxPageSize:     1000,
			},
		},
	}

	// Should not panic
	handler.OnSyncCompleted(100, 500)

	// Cache should be cleared
	if _, found := c.Get("analytics"); found {
		t.Error("Cache should be cleared after sync completed")
	}
}

// TestGetCacheStats_Comprehensive tests GetCacheStats in various states
func TestGetCacheStats_Comprehensive(t *testing.T) {
	t.Parallel()

	t.Run("with active cache", func(t *testing.T) {
		c := cache.New(5 * time.Minute)

		// Add items and access
		c.Set("key1", "value1")
		c.Set("key2", "value2")
		c.Get("key1") // Hit
		c.Get("key3") // Miss

		handler := &Handler{cache: c}
		stats := handler.GetCacheStats()

		if stats.Hits < 1 {
			t.Errorf("Expected at least 1 hit, got %d", stats.Hits)
		}
		if stats.Misses < 1 {
			t.Errorf("Expected at least 1 miss, got %d", stats.Misses)
		}
	})

	t.Run("with empty cache", func(t *testing.T) {
		c := cache.New(5 * time.Minute)
		handler := &Handler{cache: c}
		stats := handler.GetCacheStats()

		// Should return valid stats even with empty cache
		if stats.Hits != 0 {
			t.Errorf("Expected 0 hits, got %d", stats.Hits)
		}
	})
}

// TestGetPerformanceStats_Comprehensive tests GetPerformanceStats in various states
func TestGetPerformanceStats_Comprehensive(t *testing.T) {
	t.Parallel()

	t.Run("with active monitor", func(t *testing.T) {
		perfMon := middleware.NewPerformanceMonitor(100)
		handler := &Handler{perfMon: perfMon}

		stats := handler.GetPerformanceStats()

		// Should return valid (possibly empty) stats
		if stats == nil {
			// Empty stats is valid for new handler
			t.Log("Performance stats are nil (expected for new handler)")
		}
	})

	t.Run("nil monitor returns nil", func(t *testing.T) {
		handler := &Handler{perfMon: nil}
		stats := handler.GetPerformanceStats()

		if stats != nil {
			t.Error("Expected nil stats for nil monitor")
		}
	})
}

// BenchmarkCheckWebSocketOrigin benchmarks the origin checking function
func BenchmarkCheckWebSocketOrigin(b *testing.B) {
	cfg := &config.Config{
		Security: config.SecurityConfig{
			CORSOrigins: []string{
				"http://localhost:3857",
				"http://example.com",
				"https://app.example.com",
			},
		},
	}

	handler := &Handler{config: cfg}
	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	req.Header.Set("Origin", "http://example.com")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler.checkWebSocketOrigin(req)
	}
}

// BenchmarkClearCache benchmarks cache clearing
func BenchmarkClearCache(b *testing.B) {
	c := cache.New(5 * time.Minute)

	// Pre-populate cache
	for i := 0; i < 1000; i++ {
		c.Set(string(rune('A'+i%26)), "value")
	}

	handler := &Handler{cache: c}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler.ClearCache()
	}
}
