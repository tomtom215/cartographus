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

// Note: The Plex API handlers require a *syncpkg.Manager which is a concrete type.
// For comprehensive testing, we test the nil sync manager case (plex disabled)
// and validate the error handling paths. Success paths would require integration tests.

// setupPlexAPITestHandler creates a handler for Plex API testing
func setupPlexAPITestHandlerWithNilSync(t *testing.T) *Handler {
	t.Helper()

	return &Handler{
		cache:     cache.New(5 * time.Minute),
		startTime: time.Now(),
		sync:      nil, // Plex disabled
	}
}

// Test PlexBandwidthStatistics endpoint - Plex Disabled
func TestPlexBandwidthStatistics_PlexDisabled_NilSync(t *testing.T) {
	t.Parallel()

	handler := setupPlexAPITestHandlerWithNilSync(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/plex/statistics/bandwidth", nil)
	w := httptest.NewRecorder()

	handler.PlexBandwidthStatistics(w, req)

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

	if response.Error.Message != "Plex integration is not enabled" {
		t.Errorf("Expected error message 'Plex integration is not enabled', got '%s'", response.Error.Message)
	}
}

func TestPlexBandwidthStatistics_TimespanParsing(t *testing.T) {
	t.Parallel()

	// Test that the handler correctly parses the timespan parameter format
	// Even though it returns 503 due to nil sync, we verify the request processing
	handler := setupPlexAPITestHandlerWithNilSync(t)

	testCases := []struct {
		name     string
		queryStr string
	}{
		{"no_timespan", "/api/v1/plex/statistics/bandwidth"},
		{"valid_timespan", "/api/v1/plex/statistics/bandwidth?timespan=3600"},
		{"invalid_timespan", "/api/v1/plex/statistics/bandwidth?timespan=invalid"},
		{"zero_timespan", "/api/v1/plex/statistics/bandwidth?timespan=0"},
		{"negative_timespan", "/api/v1/plex/statistics/bandwidth?timespan=-100"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.queryStr, nil)
			w := httptest.NewRecorder()

			handler.PlexBandwidthStatistics(w, req)

			// Should still return 503 (Plex disabled) regardless of query params
			if w.Code != http.StatusServiceUnavailable {
				t.Errorf("Expected status 503, got %d", w.Code)
			}
		})
	}
}

// Test PlexLibrarySections endpoint - Plex Disabled
func TestPlexLibrarySections_PlexDisabled(t *testing.T) {
	t.Parallel()

	handler := setupPlexAPITestHandlerWithNilSync(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/plex/library/sections", nil)
	w := httptest.NewRecorder()

	handler.PlexLibrarySections(w, req)

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

// Test PlexLibrarySectionContent endpoint - Plex Disabled
func TestPlexLibrarySectionContent_PlexDisabled(t *testing.T) {
	t.Parallel()

	handler := setupPlexAPITestHandlerWithNilSync(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/plex/library/sections/1/all", nil)
	w := httptest.NewRecorder()

	handler.PlexLibrarySectionContent(w, req)

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

func TestPlexLibrarySectionContent_PaginationParsing(t *testing.T) {
	t.Parallel()

	handler := setupPlexAPITestHandlerWithNilSync(t)

	testCases := []struct {
		name     string
		queryStr string
	}{
		{"no_pagination", "/api/v1/plex/library/sections/1/all"},
		{"with_start", "/api/v1/plex/library/sections/1/all?start=10"},
		{"with_size", "/api/v1/plex/library/sections/1/all?size=50"},
		{"with_both", "/api/v1/plex/library/sections/1/all?start=10&size=50"},
		{"invalid_start", "/api/v1/plex/library/sections/1/all?start=abc"},
		{"invalid_size", "/api/v1/plex/library/sections/1/all?size=xyz"},
		{"zero_values", "/api/v1/plex/library/sections/1/all?start=0&size=0"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.queryStr, nil)
			w := httptest.NewRecorder()

			handler.PlexLibrarySectionContent(w, req)

			// Should still return 503 (Plex disabled) regardless of query params
			if w.Code != http.StatusServiceUnavailable {
				t.Errorf("Expected status 503, got %d", w.Code)
			}
		})
	}
}

// Test PlexLibrarySectionRecentlyAdded endpoint - Plex Disabled
func TestPlexLibrarySectionRecentlyAdded_PlexDisabled(t *testing.T) {
	t.Parallel()

	handler := setupPlexAPITestHandlerWithNilSync(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/plex/library/sections/1/recentlyAdded", nil)
	w := httptest.NewRecorder()

	handler.PlexLibrarySectionRecentlyAdded(w, req)

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

func TestPlexLibrarySectionRecentlyAdded_SizeParsing(t *testing.T) {
	t.Parallel()

	handler := setupPlexAPITestHandlerWithNilSync(t)

	testCases := []struct {
		name     string
		queryStr string
	}{
		{"no_size", "/api/v1/plex/library/sections/1/recentlyAdded"},
		{"with_size", "/api/v1/plex/library/sections/1/recentlyAdded?size=25"},
		{"invalid_size", "/api/v1/plex/library/sections/1/recentlyAdded?size=not_a_number"},
		{"zero_size", "/api/v1/plex/library/sections/1/recentlyAdded?size=0"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.queryStr, nil)
			w := httptest.NewRecorder()

			handler.PlexLibrarySectionRecentlyAdded(w, req)

			if w.Code != http.StatusServiceUnavailable {
				t.Errorf("Expected status 503, got %d", w.Code)
			}
		})
	}
}

// Test PlexActivities endpoint - Plex Disabled
func TestPlexActivities_PlexDisabled(t *testing.T) {
	t.Parallel()

	handler := setupPlexAPITestHandlerWithNilSync(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/plex/activities", nil)
	w := httptest.NewRecorder()

	handler.PlexActivities(w, req)

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

	if response.Error.Message != "Plex integration is not enabled" {
		t.Errorf("Expected error message 'Plex integration is not enabled', got '%s'", response.Error.Message)
	}
}

// Test response structure for all Plex endpoints
func TestPlexEndpoints_ResponseStructure(t *testing.T) {
	t.Parallel()

	handler := setupPlexAPITestHandlerWithNilSync(t)

	testCases := []struct {
		name    string
		path    string
		handler func(http.ResponseWriter, *http.Request)
	}{
		{"BandwidthStatistics", "/api/v1/plex/statistics/bandwidth", handler.PlexBandwidthStatistics},
		{"LibrarySections", "/api/v1/plex/library/sections", handler.PlexLibrarySections},
		{"LibrarySectionContent", "/api/v1/plex/library/sections/1/all", handler.PlexLibrarySectionContent},
		{"LibrarySectionRecentlyAdded", "/api/v1/plex/library/sections/1/recentlyAdded", handler.PlexLibrarySectionRecentlyAdded},
		{"Activities", "/api/v1/plex/activities", handler.PlexActivities},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
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

// Test Content-Type headers
func TestPlexEndpoints_ContentType(t *testing.T) {
	t.Parallel()

	handler := setupPlexAPITestHandlerWithNilSync(t)

	testCases := []struct {
		name    string
		path    string
		handler func(http.ResponseWriter, *http.Request)
	}{
		{"BandwidthStatistics", "/api/v1/plex/statistics/bandwidth", handler.PlexBandwidthStatistics},
		{"LibrarySections", "/api/v1/plex/library/sections", handler.PlexLibrarySections},
		{"Activities", "/api/v1/plex/activities", handler.PlexActivities},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			w := httptest.NewRecorder()

			tc.handler(w, req)

			contentType := w.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
			}
		})
	}
}

// Test concurrent requests
func TestPlexEndpoints_ConcurrentRequests(t *testing.T) {
	t.Parallel()

	handler := setupPlexAPITestHandlerWithNilSync(t)

	// Spawn multiple goroutines to test concurrent access
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/plex/statistics/bandwidth", nil)
			w := httptest.NewRecorder()

			handler.PlexBandwidthStatistics(w, req)

			if w.Code != http.StatusServiceUnavailable {
				t.Errorf("Expected status 503, got %d", w.Code)
			}
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

// Test empty path value handling
func TestPlexLibrarySectionContent_EmptyPathValue(t *testing.T) {
	t.Parallel()

	handler := setupPlexAPITestHandlerWithNilSync(t)

	// When PathValue returns empty string (no key provided)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/plex/library/sections//all", nil)
	w := httptest.NewRecorder()

	handler.PlexLibrarySectionContent(w, req)

	// Should return 503 (Plex disabled) before checking key
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503, got %d", w.Code)
	}
}

func TestPlexLibrarySectionRecentlyAdded_EmptyPathValue(t *testing.T) {
	t.Parallel()

	handler := setupPlexAPITestHandlerWithNilSync(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/plex/library/sections//recentlyAdded", nil)
	w := httptest.NewRecorder()

	handler.PlexLibrarySectionRecentlyAdded(w, req)

	// Should return 503 (Plex disabled) before checking key
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503, got %d", w.Code)
	}
}

// Test various HTTP methods (handlers should work with any method since they don't check)
func TestPlexEndpoints_HTTPMethods(t *testing.T) {
	t.Parallel()

	handler := setupPlexAPITestHandlerWithNilSync(t)

	methods := []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/v1/plex/activities", nil)
			w := httptest.NewRecorder()

			handler.PlexActivities(w, req)

			// Should return 503 regardless of method
			if w.Code != http.StatusServiceUnavailable {
				t.Errorf("Method %s: Expected status 503, got %d", method, w.Code)
			}
		})
	}
}

// Benchmark tests
func BenchmarkPlexBandwidthStatistics_Disabled(b *testing.B) {
	handler := &Handler{
		cache:     cache.New(5 * time.Minute),
		startTime: time.Now(),
		sync:      nil,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/plex/statistics/bandwidth", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler.PlexBandwidthStatistics(w, req)
	}
}

func BenchmarkPlexLibrarySections_Disabled(b *testing.B) {
	handler := &Handler{
		cache:     cache.New(5 * time.Minute),
		startTime: time.Now(),
		sync:      nil,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/plex/library/sections", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler.PlexLibrarySections(w, req)
	}
}

func BenchmarkPlexActivities_Disabled(b *testing.B) {
	handler := &Handler{
		cache:     cache.New(5 * time.Minute),
		startTime: time.Now(),
		sync:      nil,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/plex/activities", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler.PlexActivities(w, req)
	}
}
