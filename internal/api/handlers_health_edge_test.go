// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/goccy/go-json"
	ws "github.com/tomtom215/cartographus/internal/websocket"

	"github.com/tomtom215/cartographus/internal/cache"
	"github.com/tomtom215/cartographus/internal/config"
	"github.com/tomtom215/cartographus/internal/models"
)

// TestHealthLive_MethodNotAllowed tests HealthLive with invalid HTTP methods
func TestHealthLive_MethodNotAllowed(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		startTime: time.Now(),
	}

	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/v1/health/live", nil)
			w := httptest.NewRecorder()

			handler.HealthLive(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status 405 for %s, got %d", method, w.Code)
			}
		})
	}
}

// TestHealthReady_MethodNotAllowed tests HealthReady with invalid HTTP methods
func TestHealthReady_MethodNotAllowed(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		startTime: time.Now(),
	}

	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/v1/health/ready", nil)
			w := httptest.NewRecorder()

			handler.HealthReady(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status 405 for %s, got %d", method, w.Code)
			}
		})
	}
}

// TestHealthLive_Success tests successful liveness check
func TestHealthLive_Success(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		startTime: time.Now().Add(-1 * time.Hour),
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health/live", nil)
	w := httptest.NewRecorder()

	handler.HealthLive(w, req)

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

// TestHealthLive_Uptime verifies uptime is returned
func TestHealthLive_Uptime(t *testing.T) {
	t.Parallel()

	startTime := time.Now().Add(-2 * time.Hour)
	handler := &Handler{
		startTime: startTime,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health/live", nil)
	w := httptest.NewRecorder()

	handler.HealthLive(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	data, ok := response.Data.(map[string]interface{})
	if !ok {
		t.Fatal("Response data is not a map")
	}

	uptime, ok := data["uptime"].(float64)
	if !ok {
		t.Fatal("Uptime is not a number")
	}

	// With a startTime 2 hours ago, uptime should be > 7200 seconds
	if uptime < 7200 {
		t.Errorf("Expected uptime > 7200 seconds (2 hours), got %f", uptime)
	}
}

// TestHealth_WithCache tests health endpoint with cache
func TestHealth_WithCache(t *testing.T) {
	t.Parallel()

	c := cache.New(5 * time.Minute)
	handler := &Handler{
		startTime: time.Now(),
		cache:     c,
	}

	// Add some data to cache to verify it's working
	c.Set("test-key", "test-value")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health/live", nil)
	w := httptest.NewRecorder()

	handler.HealthLive(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// TestHealth_WithWebSocket tests health with WebSocket hub
func TestHealth_WithWebSocket(t *testing.T) {
	t.Parallel()

	wsHub := ws.NewHub()
	go wsHub.Run()

	handler := &Handler{
		startTime: time.Now(),
		wsHub:     wsHub,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health/live", nil)
	w := httptest.NewRecorder()

	handler.HealthLive(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// TestHealth_WithConfig tests health with config
func TestHealth_WithConfig(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		API: config.APIConfig{
			DefaultPageSize: 100,
			MaxPageSize:     1000,
		},
	}

	handler := &Handler{
		startTime: time.Now(),
		config:    cfg,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health/live", nil)
	w := httptest.NewRecorder()

	handler.HealthLive(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// TestHealth_TautulliConnectivity tests health when Tautulli client is available
func TestHealth_TautulliConnectivity(t *testing.T) {
	t.Parallel()

	mockClient := &MockTautulliClient{}

	handler := &Handler{
		startTime: time.Now(),
		client:    mockClient,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health/live", nil)
	w := httptest.NewRecorder()

	handler.HealthLive(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// TestHealth_TautulliFails tests health when Tautulli connection fails
func TestHealth_TautulliFails(t *testing.T) {
	t.Parallel()

	mockClient := &MockTautulliClient{
		PingFunc: func(ctx context.Context) error {
			return errors.New("tautulli connection failed")
		},
	}

	handler := &Handler{
		startTime: time.Now(),
		client:    mockClient,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health/live", nil)
	w := httptest.NewRecorder()

	handler.HealthLive(w, req)

	// HealthLive should still return 200 (just reports status)
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// TestHealth_ResponseFormat tests response format
func TestHealth_ResponseFormat(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		startTime: time.Now(),
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health/live", nil)
	w := httptest.NewRecorder()

	handler.HealthLive(w, req)

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
	}
}

// BenchmarkHealthLive benchmarks the liveness endpoint
func BenchmarkHealthLive(b *testing.B) {
	handler := &Handler{
		startTime: time.Now(),
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health/live", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler.HealthLive(w, req)
	}
}

// BenchmarkHealthReady benchmarks the readiness endpoint
func BenchmarkHealthReady(b *testing.B) {
	handler := &Handler{
		startTime: time.Now(),
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health/ready", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler.HealthReady(w, req)
	}
}
