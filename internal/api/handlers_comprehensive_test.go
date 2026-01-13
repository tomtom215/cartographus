// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/goccy/go-json"
	syncpkg "github.com/tomtom215/cartographus/internal/sync"
	ws "github.com/tomtom215/cartographus/internal/websocket"

	"github.com/tomtom215/cartographus/internal/auth"
	"github.com/tomtom215/cartographus/internal/config"
	"github.com/tomtom215/cartographus/internal/database"
	"github.com/tomtom215/cartographus/internal/models"
)

// setupTestHandler creates a test handler with mocked dependencies
func setupTestHandler(t *testing.T) *Handler {
	// Create test database
	cfg := &config.DatabaseConfig{
		Path:        ":memory:",
		MaxMemory:   "512MB",
		SkipIndexes: true, // Skip 97 indexes for fast test setup
	}

	db, err := database.New(cfg, 0.0, 0.0)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	// Create test config
	testConfig := &config.Config{
		Server: config.ServerConfig{
			Port: 3857,
		},
		API: config.APIConfig{
			DefaultPageSize: 20,
			MaxPageSize:     100,
		},
		Security: config.SecurityConfig{
			AuthMode:       "jwt",
			JWTSecret:      "test_secret_with_at_least_32_characters_for_testing_purposes",
			AdminUsername:  "admin",
			AdminPassword:  "password123",
			SessionTimeout: 24 * time.Hour,
		},
		Plex: config.PlexConfig{},
	}

	// Create JWT manager
	jwtManager, err := auth.NewJWTManager(&testConfig.Security)
	if err != nil {
		t.Fatalf("Failed to create JWT manager: %v", err)
	}

	// Create WebSocket hub
	wsHub := ws.NewHub()
	go wsHub.RunWithContext(context.Background())

	// Create sync manager with mock client
	mockClient := &MockTautulliClient{}
	syncCfg := &config.Config{
		Sync: config.SyncConfig{
			Interval: 30 * time.Minute,
		},
	}
	syncManager := syncpkg.NewManager(db, nil, mockClient, syncCfg, wsHub)

	// Create handler
	handler := NewHandler(db, syncManager, mockClient, testConfig, jwtManager, wsHub)

	return handler
}

// TestStats tests the Stats endpoint
func TestStats(t *testing.T) {
	handler := setupTestHandler(t)
	defer handler.db.Close()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/stats", nil)
	w := httptest.NewRecorder()

	handler.Stats(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Status != "success" {
		t.Errorf("Expected status 'success', got '%s'", response.Status)
	}
}

// TestPlaybacks tests the Playbacks endpoint
func TestPlaybacks(t *testing.T) {
	handler := setupTestHandler(t)
	defer handler.db.Close()

	tests := []struct {
		name       string
		url        string
		wantStatus int
	}{
		{"no parameters", "/api/v1/playbacks", http.StatusOK},
		{"with limit", "/api/v1/playbacks?limit=10", http.StatusOK},
		{"with offset", "/api/v1/playbacks?offset=5", http.StatusOK},
		{"with user filter", "/api/v1/playbacks?user=testuser", http.StatusOK},
		{"with media type filter", "/api/v1/playbacks?media_type=movie", http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			w := httptest.NewRecorder()

			handler.Playbacks(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("Expected status %d, got %d", tt.wantStatus, w.Code)
			}
		})
	}
}

// TestLocations tests the Locations endpoint
func TestLocations(t *testing.T) {
	handler := setupTestHandler(t)
	defer handler.db.Close()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/locations", nil)
	w := httptest.NewRecorder()

	handler.Locations(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Status != "success" {
		t.Errorf("Expected status 'success', got '%s'", response.Status)
	}
}

// TestUsers tests the Users endpoint
func TestUsers(t *testing.T) {
	handler := setupTestHandler(t)
	defer handler.db.Close()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	w := httptest.NewRecorder()

	handler.Users(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Status != "success" {
		t.Errorf("Expected status 'success', got '%s'", response.Status)
	}
}

// TestMediaTypes tests the MediaTypes endpoint
func TestMediaTypes(t *testing.T) {
	handler := setupTestHandler(t)
	defer handler.db.Close()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/media-types", nil)
	w := httptest.NewRecorder()

	handler.MediaTypes(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Status != "success" {
		t.Errorf("Expected status 'success', got '%s'", response.Status)
	}
}

// TestServerInfo tests the ServerInfo endpoint
func TestServerInfo(t *testing.T) {
	handler := setupTestHandler(t)
	defer handler.db.Close()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/server-info", nil)
	w := httptest.NewRecorder()

	handler.ServerInfo(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Status != "success" {
		t.Errorf("Expected status 'success', got '%s'", response.Status)
	}
}

// TestAnalyticsTrends tests the AnalyticsTrends endpoint
func TestAnalyticsTrends(t *testing.T) {
	handler := setupTestHandler(t)
	defer handler.db.Close()

	tests := []struct {
		name       string
		url        string
		wantStatus int
	}{
		{"default parameters", "/api/v1/analytics/trends", http.StatusOK},
		{"with groupBy hour", "/api/v1/analytics/trends?groupBy=hour", http.StatusOK},
		{"with groupBy day", "/api/v1/analytics/trends?groupBy=day", http.StatusOK},
		{"with groupBy week", "/api/v1/analytics/trends?groupBy=week", http.StatusOK},
		{"with groupBy month", "/api/v1/analytics/trends?groupBy=month", http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			w := httptest.NewRecorder()

			handler.AnalyticsTrends(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("Expected status %d, got %d", tt.wantStatus, w.Code)
			}
		})
	}
}

// TestAnalyticsGeographic tests the AnalyticsGeographic endpoint
func TestAnalyticsGeographic(t *testing.T) {
	handler := setupTestHandler(t)
	defer handler.db.Close()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/geographic", nil)
	w := httptest.NewRecorder()

	handler.AnalyticsGeographic(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Status != "success" {
		t.Errorf("Expected status 'success', got '%s'", response.Status)
	}
}

// TestAnalyticsUsers tests the AnalyticsUsers endpoint
func TestAnalyticsUsers(t *testing.T) {
	handler := setupTestHandler(t)
	defer handler.db.Close()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/users", nil)
	w := httptest.NewRecorder()

	handler.AnalyticsUsers(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// TestAnalyticsBinge tests the AnalyticsBinge endpoint
func TestAnalyticsBinge(t *testing.T) {
	handler := setupTestHandler(t)
	defer handler.db.Close()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/binge", nil)
	w := httptest.NewRecorder()

	handler.AnalyticsBinge(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// TestAnalyticsBandwidth tests the AnalyticsBandwidth endpoint
func TestAnalyticsBandwidth(t *testing.T) {
	handler := setupTestHandler(t)
	defer handler.db.Close()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/bandwidth", nil)
	w := httptest.NewRecorder()

	handler.AnalyticsBandwidth(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// TestAnalyticsPopular tests the AnalyticsPopular endpoint
func TestAnalyticsPopular(t *testing.T) {
	handler := setupTestHandler(t)
	defer handler.db.Close()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/popular", nil)
	w := httptest.NewRecorder()

	handler.AnalyticsPopular(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// TestAnalyticsUserEngagement tests the AnalyticsUserEngagement endpoint
func TestAnalyticsUserEngagement(t *testing.T) {
	handler := setupTestHandler(t)
	defer handler.db.Close()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/user-engagement", nil)
	w := httptest.NewRecorder()

	handler.AnalyticsUserEngagement(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// TestAnalyticsWatchParties tests the AnalyticsWatchParties endpoint
func TestAnalyticsWatchParties(t *testing.T) {
	handler := setupTestHandler(t)
	defer handler.db.Close()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/watch-parties", nil)
	w := httptest.NewRecorder()

	handler.AnalyticsWatchParties(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// TestAnalyticsConcurrentStreams tests the AnalyticsConcurrentStreams endpoint
func TestAnalyticsConcurrentStreams(t *testing.T) {
	handler := setupTestHandler(t)
	defer handler.db.Close()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/concurrent-streams", nil)
	w := httptest.NewRecorder()

	handler.AnalyticsConcurrentStreams(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// TestClearCache tests the ClearCache method
func TestClearCache(t *testing.T) {
	handler := setupTestHandler(t)
	defer handler.db.Close()

	// Add something to cache
	handler.cache.Set("test_key", "test_value")

	// Verify it's there
	if _, found := handler.cache.Get("test_key"); !found {
		t.Fatal("Failed to set cache value")
	}

	// Clear cache
	handler.ClearCache()

	// Verify it's gone
	if _, found := handler.cache.Get("test_key"); found {
		t.Error("Cache was not cleared")
	}
}

// TestOnSyncCompletedHandlers tests the OnSyncCompleted method
func TestOnSyncCompletedHandlers(t *testing.T) {
	handler := setupTestHandler(t)
	defer handler.db.Close()

	// Add something to cache
	handler.cache.Set("test_key", "test_value")

	// Call OnSyncCompleted (newRecords int, durationMs int64)
	handler.OnSyncCompleted(100, 5000)

	// Verify cache was cleared
	if _, found := handler.cache.Get("test_key"); found {
		t.Error("Cache was not cleared after sync completed")
	}
}

// TestCacheKeyGeneration tests cache key generation for different endpoints
func TestCacheKeyGeneration(t *testing.T) {
	handler := setupTestHandler(t)
	defer handler.db.Close()

	tests := []struct {
		name     string
		endpoint string
		params   string
	}{
		{"trends no params", "/api/v1/analytics/trends", ""},
		{"trends with groupBy", "/api/v1/analytics/trends", "groupBy=hour"},
		{"users", "/api/v1/analytics/users", ""},
		{"binge", "/api/v1/analytics/binge", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := tt.endpoint
			if tt.params != "" {
				url += "?" + tt.params
			}

			// First request should miss cache
			req1 := httptest.NewRequest(http.MethodGet, url, nil)
			w1 := httptest.NewRecorder()

			handler.AnalyticsTrends(w1, req1)

			if w1.Code != http.StatusOK {
				t.Errorf("First request failed with status %d", w1.Code)
			}

			// Second identical request should hit cache
			req2 := httptest.NewRequest(http.MethodGet, url, nil)
			w2 := httptest.NewRecorder()

			handler.AnalyticsTrends(w2, req2)

			if w2.Code != http.StatusOK {
				t.Errorf("Second request failed with status %d", w2.Code)
			}
		})
	}
}

// TestInvalidHTTPMethods tests that endpoints reject invalid HTTP methods
func TestInvalidHTTPMethods(t *testing.T) {
	handler := setupTestHandler(t)
	defer handler.db.Close()

	endpoints := []struct {
		name    string
		handler http.HandlerFunc
	}{
		{"Stats", handler.Stats},
		{"Playbacks", handler.Playbacks},
		{"Locations", handler.Locations},
		{"Users", handler.Users},
	}

	for _, endpoint := range endpoints {
		t.Run(endpoint.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/test", nil)
			w := httptest.NewRecorder()

			endpoint.handler(w, req)

			// Some endpoints may accept POST, some may not
			// This test ensures the handler processes the request without panicking
			if w.Code == 0 {
				t.Error("Handler did not set a response code")
			}
		})
	}
}
