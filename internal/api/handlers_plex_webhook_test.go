// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package api

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
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

// setupWebhookTestHandler creates a handler for webhook testing
func setupWebhookTestHandler(t *testing.T, webhooksEnabled bool, webhookSecret string) *Handler {
	t.Helper()

	cfg := &config.Config{
		Plex: config.PlexConfig{
			WebhooksEnabled: webhooksEnabled,
			WebhookSecret:   webhookSecret,
		},
		API: config.APIConfig{
			DefaultPageSize: 100,
			MaxPageSize:     1000,
		},
	}

	wsHub := ws.NewHub()
	go wsHub.Run()

	return &Handler{
		cache:     cache.New(5 * time.Minute),
		config:    cfg,
		wsHub:     wsHub,
		startTime: time.Now(),
	}
}

// createPlexWebhookPayload creates a test Plex webhook payload
func createPlexWebhookPayload(event string) []byte {
	webhook := models.PlexWebhook{
		Event: event,
		Account: models.PlexWebhookAccount{
			ID:    12345,
			Title: "TestUser",
		},
		Server: models.PlexWebhookServer{
			Title: "TestServer",
			UUID:  "test-uuid-12345",
		},
		Player: models.PlexWebhookPlayer{
			Title:         "Test Player",
			PublicAddress: "192.168.1.100",
			Local:         true,
			UUID:          "player-uuid-12345",
		},
		Metadata: &models.PlexWebhookMetadata{
			Type:                "movie",
			Title:               "Test Movie",
			LibrarySectionTitle: "Movies",
			LibrarySectionType:  "movie",
		},
	}

	data, _ := json.Marshal(webhook)
	return data
}

// generateHMACSignature creates an HMAC-SHA256 signature for webhook verification
func generateHMACSignature(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

// TestPlexWebhook_Disabled tests webhook when webhooks are disabled
func TestPlexWebhook_Disabled(t *testing.T) {
	t.Parallel()

	handler := setupWebhookTestHandler(t, false, "")

	payload := createPlexWebhookPayload("media.play")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/plex/webhook", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.PlexWebhook(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404 when webhooks disabled, got %d", w.Code)
	}

	var response models.APIResponse
	json.NewDecoder(w.Body).Decode(&response)

	if response.Error == nil || response.Error.Code != "WEBHOOKS_DISABLED" {
		t.Error("Expected WEBHOOKS_DISABLED error")
	}
}

// TestPlexWebhook_MissingSignature tests webhook without signature when secret is configured
func TestPlexWebhook_MissingSignature(t *testing.T) {
	t.Parallel()

	handler := setupWebhookTestHandler(t, true, "test-secret-12345678901234567890")

	payload := createPlexWebhookPayload("media.play")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/plex/webhook", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	// Not setting X-Plex-Signature header
	w := httptest.NewRecorder()

	handler.PlexWebhook(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401 when signature missing, got %d", w.Code)
	}

	var response models.APIResponse
	json.NewDecoder(w.Body).Decode(&response)

	if response.Error == nil || response.Error.Code != "MISSING_SIGNATURE" {
		t.Error("Expected MISSING_SIGNATURE error")
	}
}

// TestPlexWebhook_InvalidSignature tests webhook with invalid signature
func TestPlexWebhook_InvalidSignature(t *testing.T) {
	t.Parallel()

	secret := "test-secret-12345678901234567890"
	handler := setupWebhookTestHandler(t, true, secret)

	payload := createPlexWebhookPayload("media.play")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/plex/webhook", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Plex-Signature", "invalid-signature")
	w := httptest.NewRecorder()

	handler.PlexWebhook(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401 for invalid signature, got %d", w.Code)
	}

	var response models.APIResponse
	json.NewDecoder(w.Body).Decode(&response)

	if response.Error == nil || response.Error.Code != "INVALID_SIGNATURE" {
		t.Error("Expected INVALID_SIGNATURE error")
	}
}

// TestPlexWebhook_ValidSignature tests webhook with valid signature
func TestPlexWebhook_ValidSignature(t *testing.T) {
	t.Parallel()

	secret := "test-secret-12345678901234567890"
	handler := setupWebhookTestHandler(t, true, secret)

	payload := createPlexWebhookPayload("media.play")
	signature := generateHMACSignature(payload, secret)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/plex/webhook", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Plex-Signature", signature)
	w := httptest.NewRecorder()

	handler.PlexWebhook(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response models.APIResponse
	json.NewDecoder(w.Body).Decode(&response)

	if response.Status != "success" {
		t.Errorf("Expected status 'success', got '%s'", response.Status)
	}

	data, ok := response.Data.(map[string]interface{})
	if !ok {
		t.Fatal("Response data is not a map")
	}

	if data["received"] != true {
		t.Error("Expected received=true")
	}

	if data["event"] != "media.play" {
		t.Errorf("Expected event='media.play', got '%v'", data["event"])
	}
}

// TestPlexWebhook_NoSecretRequired tests webhook when no secret is configured
func TestPlexWebhook_NoSecretRequired(t *testing.T) {
	t.Parallel()

	handler := setupWebhookTestHandler(t, true, "") // No secret

	payload := createPlexWebhookPayload("media.stop")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/plex/webhook", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.PlexWebhook(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 when no secret configured, got %d", w.Code)
	}
}

// TestPlexWebhook_InvalidJSON tests webhook with invalid JSON payload
func TestPlexWebhook_InvalidJSON(t *testing.T) {
	t.Parallel()

	handler := setupWebhookTestHandler(t, true, "")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/plex/webhook", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.PlexWebhook(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid JSON, got %d", w.Code)
	}

	var response models.APIResponse
	json.NewDecoder(w.Body).Decode(&response)

	if response.Error == nil || response.Error.Code != "INVALID_PAYLOAD" {
		t.Error("Expected INVALID_PAYLOAD error")
	}
}

// TestPlexWebhook_AllEventTypes tests that all event types are handled
func TestPlexWebhook_AllEventTypes(t *testing.T) {
	t.Parallel()

	eventTypes := []string{
		"media.play",
		"media.pause",
		"media.resume",
		"media.stop",
		"media.scrobble",
		"media.rate",
		"library.new",
		"library.on.deck",
		"admin.database.backup",
		"admin.database.corrupted",
		"device.new",
		"unknown.event", // Should also be handled gracefully
	}

	for _, eventType := range eventTypes {
		t.Run(eventType, func(t *testing.T) {
			handler := setupWebhookTestHandler(t, true, "")

			payload := createPlexWebhookPayload(eventType)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/plex/webhook", bytes.NewReader(payload))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.PlexWebhook(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Event type '%s' failed with status %d", eventType, w.Code)
			}

			var response models.APIResponse
			json.NewDecoder(w.Body).Decode(&response)

			if response.Status != "success" {
				t.Errorf("Event type '%s' expected status 'success', got '%s'", eventType, response.Status)
			}
		})
	}
}

// TestVerifyWebhookSignature tests the HMAC signature verification
func TestVerifyWebhookSignature(t *testing.T) {
	t.Parallel()

	handler := setupWebhookTestHandler(t, true, "")

	tests := []struct {
		name     string
		body     []byte
		secret   string
		getSign  func([]byte, string) string
		expected bool
	}{
		{
			name:   "valid signature",
			body:   []byte(`{"event":"media.play"}`),
			secret: "test-secret",
			getSign: func(body []byte, secret string) string {
				return generateHMACSignature(body, secret)
			},
			expected: true,
		},
		{
			name:   "invalid signature - wrong secret",
			body:   []byte(`{"event":"media.play"}`),
			secret: "test-secret",
			getSign: func(body []byte, secret string) string {
				return generateHMACSignature(body, "wrong-secret")
			},
			expected: false,
		},
		{
			name:   "invalid signature - tampered body",
			body:   []byte(`{"event":"media.play"}`),
			secret: "test-secret",
			getSign: func(body []byte, secret string) string {
				return generateHMACSignature([]byte(`{"event":"media.stop"}`), secret)
			},
			expected: false,
		},
		{
			name:   "empty signature",
			body:   []byte(`{"event":"media.play"}`),
			secret: "test-secret",
			getSign: func(body []byte, secret string) string {
				return ""
			},
			expected: false,
		},
		{
			name:   "malformed signature",
			body:   []byte(`{"event":"media.play"}`),
			secret: "test-secret",
			getSign: func(body []byte, secret string) string {
				return "not-a-valid-hex"
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			signature := tt.getSign(tt.body, tt.secret)
			result := handler.verifyWebhookSignature(tt.body, signature, tt.secret)

			if result != tt.expected {
				t.Errorf("verifyWebhookSignature() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestPlexWebhook_EmptyBody tests webhook with empty body
func TestPlexWebhook_EmptyBody(t *testing.T) {
	t.Parallel()

	handler := setupWebhookTestHandler(t, true, "")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/plex/webhook", bytes.NewReader([]byte{}))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.PlexWebhook(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for empty body, got %d", w.Code)
	}
}

// TestPlexWebhook_MinimalPayload tests webhook with minimal valid payload
func TestPlexWebhook_MinimalPayload(t *testing.T) {
	t.Parallel()

	handler := setupWebhookTestHandler(t, true, "")

	payload := []byte(`{"event":"media.play"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/plex/webhook", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.PlexWebhook(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for minimal payload, got %d", w.Code)
	}
}

// TestPlexWebhook_LibraryNewWithoutMetadata tests library.new event without metadata
func TestPlexWebhook_LibraryNewWithoutMetadata(t *testing.T) {
	t.Parallel()

	handler := setupWebhookTestHandler(t, true, "")

	// Webhook without metadata
	webhook := models.PlexWebhook{
		Event:    "library.new",
		Metadata: nil, // No metadata
	}
	payload, _ := json.Marshal(webhook)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/plex/webhook", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.PlexWebhook(w, req)

	// Should still succeed (graceful handling)
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for library.new without metadata, got %d", w.Code)
	}
}

// TestPlexWebhook_QueryTimeMetadata tests that query time is included in response
func TestPlexWebhook_QueryTimeMetadata(t *testing.T) {
	t.Parallel()

	handler := setupWebhookTestHandler(t, true, "")

	payload := createPlexWebhookPayload("media.play")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/plex/webhook", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.PlexWebhook(w, req)

	var response models.APIResponse
	json.NewDecoder(w.Body).Decode(&response)

	if response.Metadata.QueryTimeMS < 0 {
		t.Error("QueryTimeMS should be non-negative")
	}

	if response.Metadata.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}
}

// BenchmarkPlexWebhook benchmarks webhook processing
func BenchmarkPlexWebhook(b *testing.B) {
	cfg := &config.Config{
		Plex: config.PlexConfig{
			WebhooksEnabled: true,
			WebhookSecret:   "",
		},
	}

	wsHub := ws.NewHub()
	go wsHub.Run()

	handler := &Handler{
		cache:     cache.New(5 * time.Minute),
		config:    cfg,
		wsHub:     wsHub,
		startTime: time.Now(),
	}

	payload := createPlexWebhookPayload("media.play")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/plex/webhook", bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.PlexWebhook(w, req)
	}
}

// BenchmarkVerifyWebhookSignature benchmarks signature verification
func BenchmarkVerifyWebhookSignature(b *testing.B) {
	handler := &Handler{}
	body := []byte(`{"event":"media.play","account":{"id":12345,"title":"TestUser"}}`)
	secret := "test-secret-12345678901234567890"
	signature := generateHMACSignature(body, secret)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler.verifyWebhookSignature(body, signature, secret)
	}
}
