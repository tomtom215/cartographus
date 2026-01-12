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

	"github.com/goccy/go-json"

	"github.com/tomtom215/cartographus/internal/cache"
	"github.com/tomtom215/cartographus/internal/models"
)

// Tests for Plex API handlers with 0% coverage

// TestPlexSearch_PlexDisabled tests the PlexSearch handler when Plex is disabled
func TestPlexSearch_PlexDisabled(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		cache:     cache.New(5 * time.Minute),
		startTime: time.Now(),
		sync:      nil, // Plex disabled
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/plex/library/sections/1/search?query=test", nil)
	w := httptest.NewRecorder()

	handler.PlexSearch(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503, got %d", w.Code)
	}

	var response models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Error == nil || response.Error.Code != "PLEX_DISABLED" {
		t.Errorf("Expected PLEX_DISABLED error, got %v", response.Error)
	}
}

// TestPlexSearch_QueryParams tests query parameter parsing
func TestPlexSearch_QueryParams(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		cache:     cache.New(5 * time.Minute),
		startTime: time.Now(),
		sync:      nil,
	}

	testCases := []struct {
		name     string
		queryStr string
	}{
		{"basic_query", "/api/v1/plex/library/sections/1/search?query=avatar"},
		{"with_type", "/api/v1/plex/library/sections/1/search?query=avatar&type=movie"},
		{"with_limit", "/api/v1/plex/library/sections/1/search?query=avatar&limit=20"},
		{"empty_query", "/api/v1/plex/library/sections/1/search?query="},
		{"no_query", "/api/v1/plex/library/sections/1/search"},
		{"special_chars", "/api/v1/plex/library/sections/1/search?query=the%20office"},
		{"with_all_params", "/api/v1/plex/library/sections/1/search?query=test&type=show&limit=10"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.queryStr, nil)
			w := httptest.NewRecorder()

			handler.PlexSearch(w, req)

			// Should return 503 (Plex disabled) regardless of query params
			if w.Code != http.StatusServiceUnavailable {
				t.Errorf("Expected status 503, got %d for %s", w.Code, tc.name)
			}
		})
	}
}

// TestPlexTranscodeSessions_PlexDisabled tests the PlexTranscodeSessions handler when Plex is disabled
func TestPlexTranscodeSessions_PlexDisabled(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		cache:     cache.New(5 * time.Minute),
		startTime: time.Now(),
		sync:      nil,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/plex/transcode/sessions", nil)
	w := httptest.NewRecorder()

	handler.PlexTranscodeSessions(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503, got %d", w.Code)
	}

	var response models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Error == nil || response.Error.Code != "PLEX_DISABLED" {
		t.Errorf("Expected PLEX_DISABLED error, got %v", response.Error)
	}
}

// TestPlexTranscodeSessions_ResponseStructure tests response structure
func TestPlexTranscodeSessions_ResponseStructure(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		cache:     cache.New(5 * time.Minute),
		startTime: time.Now(),
		sync:      nil,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/plex/transcode/sessions", nil)
	w := httptest.NewRecorder()

	handler.PlexTranscodeSessions(w, req)

	// Verify JSON response structure
	var response models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Status != "error" {
		t.Errorf("Expected status 'error', got '%s'", response.Status)
	}

	if response.Error == nil {
		t.Fatal("Expected error field in response")
	}

	if response.Error.Message == "" {
		t.Error("Expected non-empty error message")
	}
}

// TestPlexCancelTranscode_PlexDisabled tests the PlexCancelTranscode handler when Plex is disabled
func TestPlexCancelTranscode_PlexDisabled(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		cache:     cache.New(5 * time.Minute),
		startTime: time.Now(),
		sync:      nil,
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/plex/transcode/sessions/abc123", nil)
	w := httptest.NewRecorder()

	handler.PlexCancelTranscode(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503, got %d", w.Code)
	}

	var response models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Error == nil || response.Error.Code != "PLEX_DISABLED" {
		t.Errorf("Expected PLEX_DISABLED error, got %v", response.Error)
	}
}

// TestPlexCancelTranscode_SessionKeyVariations tests various session key formats
func TestPlexCancelTranscode_SessionKeyVariations(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		cache:     cache.New(5 * time.Minute),
		startTime: time.Now(),
		sync:      nil,
	}

	testCases := []struct {
		name       string
		sessionKey string
	}{
		{"alphanumeric", "/api/v1/plex/transcode/sessions/abc123def456"},
		{"numeric_only", "/api/v1/plex/transcode/sessions/12345678"},
		{"uuid_format", "/api/v1/plex/transcode/sessions/550e8400-e29b-41d4-a716-446655440000"},
		{"short_key", "/api/v1/plex/transcode/sessions/a"},
		{"long_key", "/api/v1/plex/transcode/sessions/abcdefghijklmnopqrstuvwxyz1234567890"},
		{"empty_key", "/api/v1/plex/transcode/sessions/"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodDelete, tc.sessionKey, nil)
			w := httptest.NewRecorder()

			handler.PlexCancelTranscode(w, req)

			// Should return 503 (Plex disabled) regardless of session key
			if w.Code != http.StatusServiceUnavailable {
				t.Errorf("Expected status 503, got %d for %s", w.Code, tc.name)
			}
		})
	}
}

// TestPlexServerCapabilities_PlexDisabled tests the PlexServerCapabilities handler when Plex is disabled
func TestPlexServerCapabilities_PlexDisabled(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		cache:     cache.New(5 * time.Minute),
		startTime: time.Now(),
		sync:      nil,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/plex/server/capabilities", nil)
	w := httptest.NewRecorder()

	handler.PlexServerCapabilities(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503, got %d", w.Code)
	}

	var response models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Error == nil || response.Error.Code != "PLEX_DISABLED" {
		t.Errorf("Expected PLEX_DISABLED error, got %v", response.Error)
	}
}

// TestPlexServerCapabilities_ContentType tests content type header
func TestPlexServerCapabilities_ContentType(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		cache:     cache.New(5 * time.Minute),
		startTime: time.Now(),
		sync:      nil,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/plex/server/capabilities", nil)
	w := httptest.NewRecorder()

	handler.PlexServerCapabilities(w, req)

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
	}
}

// TestPlexAPIEndpoints_ExtendedResponseStructure tests response structure for all new endpoints
func TestPlexAPIEndpoints_ExtendedResponseStructure(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		cache:     cache.New(5 * time.Minute),
		startTime: time.Now(),
		sync:      nil,
	}

	testCases := []struct {
		name    string
		method  string
		path    string
		handler func(http.ResponseWriter, *http.Request)
	}{
		{"PlexSearch", http.MethodGet, "/api/v1/plex/library/sections/1/search?query=test", handler.PlexSearch},
		{"PlexTranscodeSessions", http.MethodGet, "/api/v1/plex/transcode/sessions", handler.PlexTranscodeSessions},
		{"PlexCancelTranscode", http.MethodDelete, "/api/v1/plex/transcode/sessions/abc123", handler.PlexCancelTranscode},
		{"PlexServerCapabilities", http.MethodGet, "/api/v1/plex/server/capabilities", handler.PlexServerCapabilities},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			w := httptest.NewRecorder()

			tc.handler(w, req)

			if w.Code != http.StatusServiceUnavailable {
				t.Errorf("Expected status 503, got %d", w.Code)
			}

			var response models.APIResponse
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			// Verify error response structure
			if response.Status != "error" {
				t.Errorf("Expected status 'error', got '%s'", response.Status)
			}

			if response.Error == nil {
				t.Fatal("Expected error field in response")
			}

			if response.Error.Code == "" {
				t.Error("Expected error code to be non-empty")
			}

			if response.Error.Message == "" {
				t.Error("Expected error message to be non-empty")
			}
		})
	}
}

// TestPlexAPIEndpoints_ExtendedContentType tests content type for all new endpoints
func TestPlexAPIEndpoints_ExtendedContentType(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		cache:     cache.New(5 * time.Minute),
		startTime: time.Now(),
		sync:      nil,
	}

	testCases := []struct {
		name    string
		method  string
		path    string
		handler func(http.ResponseWriter, *http.Request)
	}{
		{"PlexSearch", http.MethodGet, "/api/v1/plex/library/sections/1/search?query=test", handler.PlexSearch},
		{"PlexTranscodeSessions", http.MethodGet, "/api/v1/plex/transcode/sessions", handler.PlexTranscodeSessions},
		{"PlexServerCapabilities", http.MethodGet, "/api/v1/plex/server/capabilities", handler.PlexServerCapabilities},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			w := httptest.NewRecorder()

			tc.handler(w, req)

			contentType := w.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
			}
		})
	}
}

// TestPlexAPIEndpoints_ExtendedConcurrent tests concurrent access to new endpoints
func TestPlexAPIEndpoints_ExtendedConcurrent(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		cache:     cache.New(5 * time.Minute),
		startTime: time.Now(),
		sync:      nil,
	}

	endpoints := []struct {
		method  string
		path    string
		handler func(http.ResponseWriter, *http.Request)
	}{
		{http.MethodGet, "/api/v1/plex/library/sections/1/search?query=test", handler.PlexSearch},
		{http.MethodGet, "/api/v1/plex/transcode/sessions", handler.PlexTranscodeSessions},
		{http.MethodGet, "/api/v1/plex/server/capabilities", handler.PlexServerCapabilities},
	}

	for _, ep := range endpoints {
		done := make(chan bool, 10)
		for i := 0; i < 10; i++ {
			go func(method, path string, h func(http.ResponseWriter, *http.Request)) {
				req := httptest.NewRequest(method, path, nil)
				w := httptest.NewRecorder()
				h(w, req)
				done <- true
			}(ep.method, ep.path, ep.handler)
		}

		for i := 0; i < 10; i++ {
			select {
			case <-done:
			case <-time.After(5 * time.Second):
				t.Errorf("Timeout waiting for concurrent request to %s", ep.path)
			}
		}
	}
}

// Benchmark tests for new Plex API endpoints
func BenchmarkPlexSearch_Disabled(b *testing.B) {
	handler := &Handler{
		cache:     cache.New(5 * time.Minute),
		startTime: time.Now(),
		sync:      nil,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/plex/library/sections/1/search?query=test", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler.PlexSearch(w, req)
	}
}

func BenchmarkPlexTranscodeSessions_Disabled(b *testing.B) {
	handler := &Handler{
		cache:     cache.New(5 * time.Minute),
		startTime: time.Now(),
		sync:      nil,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/plex/transcode/sessions", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler.PlexTranscodeSessions(w, req)
	}
}

func BenchmarkPlexCancelTranscode_Disabled(b *testing.B) {
	handler := &Handler{
		cache:     cache.New(5 * time.Minute),
		startTime: time.Now(),
		sync:      nil,
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/plex/transcode/sessions/abc123", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler.PlexCancelTranscode(w, req)
	}
}

func BenchmarkPlexServerCapabilities_Disabled(b *testing.B) {
	handler := &Handler{
		cache:     cache.New(5 * time.Minute),
		startTime: time.Now(),
		sync:      nil,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/plex/server/capabilities", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler.PlexServerCapabilities(w, req)
	}
}
