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

// ============================================================================
// Tests for Plex Sessions Endpoint - GET /api/v1/plex/sessions
// TDD: Tests written FIRST before implementation
// ============================================================================

// TestPlexSessions_PlexDisabled tests that the endpoint returns 503 when Plex is not enabled
func TestPlexSessions_PlexDisabled(t *testing.T) {
	t.Parallel()

	handler := setupPlexAPITestHandlerWithNilSync(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/plex/sessions", nil)
	w := httptest.NewRecorder()

	handler.PlexSessions(w, req)

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

// TestPlexSessions_ResponseContentType tests that the response has correct Content-Type
func TestPlexSessions_ResponseContentType(t *testing.T) {
	t.Parallel()

	handler := setupPlexAPITestHandlerWithNilSync(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/plex/sessions", nil)
	w := httptest.NewRecorder()

	handler.PlexSessions(w, req)

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
	}
}

// TestPlexSessions_ResponseStructure tests error response structure
func TestPlexSessions_ResponseStructure(t *testing.T) {
	t.Parallel()

	handler := setupPlexAPITestHandlerWithNilSync(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/plex/sessions", nil)
	w := httptest.NewRecorder()

	handler.PlexSessions(w, req)

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
}

// TestPlexSessions_HTTPMethods tests that the handler accepts various HTTP methods
func TestPlexSessions_HTTPMethods(t *testing.T) {
	t.Parallel()

	handler := setupPlexAPITestHandlerWithNilSync(t)

	methods := []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/v1/plex/sessions", nil)
			w := httptest.NewRecorder()

			handler.PlexSessions(w, req)

			// Should return 503 regardless of method (Plex disabled check happens first)
			if w.Code != http.StatusServiceUnavailable {
				t.Errorf("Method %s: Expected status 503, got %d", method, w.Code)
			}
		})
	}
}

// TestPlexSessions_ConcurrentRequests tests thread safety
func TestPlexSessions_ConcurrentRequests(t *testing.T) {
	t.Parallel()

	handler := setupPlexAPITestHandlerWithNilSync(t)

	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/plex/sessions", nil)
			w := httptest.NewRecorder()

			handler.PlexSessions(w, req)

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

// ============================================================================
// Tests for Plex Identity Endpoint - GET /api/v1/plex/identity
// ============================================================================

// TestPlexIdentity_PlexDisabled tests that the endpoint returns 503 when Plex is not enabled
func TestPlexIdentity_PlexDisabled(t *testing.T) {
	t.Parallel()

	handler := setupPlexAPITestHandlerWithNilSync(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/plex/identity", nil)
	w := httptest.NewRecorder()

	handler.PlexIdentity(w, req)

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

// TestPlexIdentity_ResponseContentType tests correct Content-Type header
func TestPlexIdentity_ResponseContentType(t *testing.T) {
	t.Parallel()

	handler := setupPlexAPITestHandlerWithNilSync(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/plex/identity", nil)
	w := httptest.NewRecorder()

	handler.PlexIdentity(w, req)

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
	}
}

// ============================================================================
// Tests for Plex Metadata Endpoint - GET /api/v1/plex/library/metadata/{ratingKey}
// ============================================================================

// TestPlexMetadata_PlexDisabled tests that the endpoint returns 503 when Plex is not enabled
func TestPlexMetadata_PlexDisabled(t *testing.T) {
	t.Parallel()

	handler := setupPlexAPITestHandlerWithNilSync(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/plex/library/metadata/12345", nil)
	w := httptest.NewRecorder()

	handler.PlexMetadata(w, req)

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

// TestPlexMetadata_ResponseContentType tests correct Content-Type header
func TestPlexMetadata_ResponseContentType(t *testing.T) {
	t.Parallel()

	handler := setupPlexAPITestHandlerWithNilSync(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/plex/library/metadata/12345", nil)
	w := httptest.NewRecorder()

	handler.PlexMetadata(w, req)

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
	}
}

// TestPlexMetadata_ResponseStructure tests error response structure
func TestPlexMetadata_ResponseStructure(t *testing.T) {
	t.Parallel()

	handler := setupPlexAPITestHandlerWithNilSync(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/plex/library/metadata/12345", nil)
	w := httptest.NewRecorder()

	handler.PlexMetadata(w, req)

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
}

// TestPlexMetadata_VariousRatingKeys tests various rating key formats
func TestPlexMetadata_VariousRatingKeys(t *testing.T) {
	t.Parallel()

	handler := setupPlexAPITestHandlerWithNilSync(t)

	testCases := []struct {
		name      string
		ratingKey string
	}{
		{"numeric key", "/api/v1/plex/library/metadata/12345"},
		{"large key", "/api/v1/plex/library/metadata/999999999"},
		{"zero key", "/api/v1/plex/library/metadata/0"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.ratingKey, nil)
			w := httptest.NewRecorder()

			handler.PlexMetadata(w, req)

			// Should return 503 (Plex disabled) regardless of rating key
			if w.Code != http.StatusServiceUnavailable {
				t.Errorf("Expected status 503, got %d", w.Code)
			}
		})
	}
}

// ============================================================================
// Tests for Plex Devices Endpoint - GET /api/v1/plex/devices
// ============================================================================

// TestPlexDevices_PlexDisabled tests that the endpoint returns 503 when Plex is not enabled
func TestPlexDevices_PlexDisabled(t *testing.T) {
	t.Parallel()

	handler := setupPlexAPITestHandlerWithNilSync(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/plex/devices", nil)
	w := httptest.NewRecorder()

	handler.PlexDevices(w, req)

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

// TestPlexDevices_ResponseContentType tests correct Content-Type header
func TestPlexDevices_ResponseContentType(t *testing.T) {
	t.Parallel()

	handler := setupPlexAPITestHandlerWithNilSync(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/plex/devices", nil)
	w := httptest.NewRecorder()

	handler.PlexDevices(w, req)

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
	}
}

// ============================================================================
// Tests for Plex Accounts Endpoint - GET /api/v1/plex/accounts
// ============================================================================

// TestPlexAccounts_PlexDisabled tests that the endpoint returns 503 when Plex is not enabled
func TestPlexAccounts_PlexDisabled(t *testing.T) {
	t.Parallel()

	handler := setupPlexAPITestHandlerWithNilSync(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/plex/accounts", nil)
	w := httptest.NewRecorder()

	handler.PlexAccounts(w, req)

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

// TestPlexAccounts_ResponseContentType tests correct Content-Type header
func TestPlexAccounts_ResponseContentType(t *testing.T) {
	t.Parallel()

	handler := setupPlexAPITestHandlerWithNilSync(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/plex/accounts", nil)
	w := httptest.NewRecorder()

	handler.PlexAccounts(w, req)

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
	}
}

// ============================================================================
// Tests for Plex On-Deck Endpoint - GET /api/v1/plex/library/onDeck
// ============================================================================

// TestPlexOnDeck_PlexDisabled tests that the endpoint returns 503 when Plex is not enabled
func TestPlexOnDeck_PlexDisabled(t *testing.T) {
	t.Parallel()

	handler := setupPlexAPITestHandlerWithNilSync(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/plex/library/onDeck", nil)
	w := httptest.NewRecorder()

	handler.PlexOnDeck(w, req)

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

// TestPlexOnDeck_ResponseContentType tests correct Content-Type header
func TestPlexOnDeck_ResponseContentType(t *testing.T) {
	t.Parallel()

	handler := setupPlexAPITestHandlerWithNilSync(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/plex/library/onDeck", nil)
	w := httptest.NewRecorder()

	handler.PlexOnDeck(w, req)

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
	}
}

// ============================================================================
// Tests for Plex Playlists Endpoint - GET /api/v1/plex/playlists
// ============================================================================

// TestPlexPlaylists_PlexDisabled tests that the endpoint returns 503 when Plex is not enabled
func TestPlexPlaylists_PlexDisabled(t *testing.T) {
	t.Parallel()

	handler := setupPlexAPITestHandlerWithNilSync(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/plex/playlists", nil)
	w := httptest.NewRecorder()

	handler.PlexPlaylists(w, req)

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

// ============================================================================
// Benchmark Tests for New Endpoints
// ============================================================================

func BenchmarkPlexSessions_Disabled(b *testing.B) {
	handler := &Handler{
		cache:     cache.New(5 * time.Minute),
		startTime: time.Now(),
		sync:      nil,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/plex/sessions", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler.PlexSessions(w, req)
	}
}

func BenchmarkPlexIdentity_Disabled(b *testing.B) {
	handler := &Handler{
		cache:     cache.New(5 * time.Minute),
		startTime: time.Now(),
		sync:      nil,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/plex/identity", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler.PlexIdentity(w, req)
	}
}

func BenchmarkPlexMetadata_Disabled(b *testing.B) {
	handler := &Handler{
		cache:     cache.New(5 * time.Minute),
		startTime: time.Now(),
		sync:      nil,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/plex/library/metadata/12345", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler.PlexMetadata(w, req)
	}
}
