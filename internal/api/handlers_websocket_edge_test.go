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
	"github.com/tomtom215/cartographus/internal/models"
	ws "github.com/tomtom215/cartographus/internal/websocket"
)

// TestWebSocket_MethodNotAllowed tests WebSocket with invalid HTTP methods
func TestWebSocket_MethodNotAllowed(t *testing.T) {
	t.Parallel()

	wsHub := ws.NewHub()
	go wsHub.Run()

	handler := &Handler{
		wsHub: wsHub,
		config: &config.Config{
			Security: config.SecurityConfig{
				CORSOrigins: []string{"*"},
			},
		},
		cache:     cache.New(5 * time.Minute),
		startTime: time.Now(),
	}

	// WebSocket upgrade requires GET method
	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/ws", nil)
			req.Header.Set("Connection", "Upgrade")
			req.Header.Set("Upgrade", "websocket")
			req.Header.Set("Sec-WebSocket-Version", "13")
			req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
			w := httptest.NewRecorder()

			handler.WebSocket(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status 405 for %s, got %d", method, w.Code)
			}
		})
	}
}

// TestWebSocket_MissingUpgradeHeaders tests WebSocket without required headers
func TestWebSocket_MissingUpgradeHeaders(t *testing.T) {
	t.Parallel()

	wsHub := ws.NewHub()
	go wsHub.Run()

	handler := &Handler{
		wsHub: wsHub,
		config: &config.Config{
			Security: config.SecurityConfig{
				CORSOrigins: []string{"*"},
			},
		},
		cache:     cache.New(5 * time.Minute),
		startTime: time.Now(),
	}

	tests := []struct {
		name    string
		headers map[string]string
	}{
		{
			name:    "no headers",
			headers: map[string]string{},
		},
		{
			name: "missing Connection header",
			headers: map[string]string{
				"Upgrade":               "websocket",
				"Sec-WebSocket-Version": "13",
				"Sec-WebSocket-Key":     "dGhlIHNhbXBsZSBub25jZQ==",
			},
		},
		{
			name: "missing Upgrade header",
			headers: map[string]string{
				"Connection":            "Upgrade",
				"Sec-WebSocket-Version": "13",
				"Sec-WebSocket-Key":     "dGhlIHNhbXBsZSBub25jZQ==",
			},
		},
		{
			name: "missing Sec-WebSocket-Key",
			headers: map[string]string{
				"Connection":            "Upgrade",
				"Upgrade":               "websocket",
				"Sec-WebSocket-Version": "13",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/ws", nil)
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}
			w := httptest.NewRecorder()

			handler.WebSocket(w, req)

			// Should fail to upgrade (400 Bad Request from gorilla/websocket)
			if w.Code == http.StatusSwitchingProtocols {
				t.Error("Should not successfully upgrade with missing headers")
			}
		})
	}
}

// TestCheckWebSocketOrigin_NilConfig tests checkWebSocketOrigin when config is nil
func TestCheckWebSocketOrigin_NilConfig(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		config: nil, // nil config
	}

	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	req.Header.Set("Origin", "http://malicious.com")

	// Should handle nil config gracefully (will panic if not handled)
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("checkWebSocketOrigin panicked with nil config: %v", r)
		}
	}()

	// This will panic if Security.CORSOrigins is accessed without nil check
	// The current implementation doesn't have a nil check, so this documents the issue
	result := handler.checkWebSocketOrigin(req)

	// If we get here without panic, the nil check exists
	t.Logf("Result with nil config: %v", result)
}

// TestCheckWebSocketOrigin_EmptyCORSOrigins tests with empty CORS origins list
func TestCheckWebSocketOrigin_EmptyCORSOrigins(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		config: &config.Config{
			Security: config.SecurityConfig{
				CORSOrigins: []string{}, // empty list
			},
		},
	}

	tests := []struct {
		name   string
		origin string
		want   bool
	}{
		{
			name:   "no origin header - SECURITY: must reject",
			origin: "",
			want:   false, // REJECT: legitimate browsers ALWAYS include Origin header
		},
		{
			name:   "with origin",
			origin: "http://example.com",
			want:   false, // not in allowed list
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/ws", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}

			result := handler.checkWebSocketOrigin(req)
			if result != tt.want {
				t.Errorf("checkWebSocketOrigin() = %v, want %v", result, tt.want)
			}
		})
	}
}

// TestCheckWebSocketOrigin_MissingOriginRejected tests the security fix for AUDIT-1.2
// This test verifies that WebSocket connections without Origin header are rejected
// to prevent CORS bypass attacks from non-browser clients.
func TestCheckWebSocketOrigin_MissingOriginRejected(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		config: &config.Config{
			Security: config.SecurityConfig{
				CORSOrigins: []string{"https://trusted.com"},
			},
		},
	}

	tests := []struct {
		name        string
		setOrigin   bool
		originValue string
		want        bool
		description string
	}{
		{
			name:        "missing Origin header - must reject",
			setOrigin:   false,
			originValue: "",
			want:        false,
			description: "Non-browser clients (curl, scripts) omit Origin - SECURITY RISK",
		},
		{
			name:        "empty Origin header - must reject",
			setOrigin:   true,
			originValue: "",
			want:        false,
			description: "Explicitly empty Origin header should also be rejected",
		},
		{
			name:        "null Origin header - must reject",
			setOrigin:   true,
			originValue: "null",
			want:        false,
			description: "Browser sends 'null' for file:// origins and sandboxed iframes",
		},
		{
			name:        "trusted origin - must allow",
			setOrigin:   true,
			originValue: "https://trusted.com",
			want:        true,
			description: "Legitimate browser request from allowed origin",
		},
		{
			name:        "untrusted origin - must reject",
			setOrigin:   true,
			originValue: "https://evil.com",
			want:        false,
			description: "Legitimate browser request but from disallowed origin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/ws", nil)
			if tt.setOrigin {
				req.Header.Set("Origin", tt.originValue)
			}

			result := handler.checkWebSocketOrigin(req)
			if result != tt.want {
				t.Errorf("checkWebSocketOrigin() = %v, want %v\nDescription: %s", result, tt.want, tt.description)
			}
		})
	}
}

// TestGetUpgrader_BufferSizes tests that upgrader has correct buffer sizes
func TestGetUpgrader_BufferSizes(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		config: &config.Config{
			Security: config.SecurityConfig{
				CORSOrigins: []string{"*"},
			},
		},
	}

	upgrader := handler.getUpgrader()

	if upgrader.ReadBufferSize != 1024 {
		t.Errorf("Expected ReadBufferSize 1024, got %d", upgrader.ReadBufferSize)
	}

	if upgrader.WriteBufferSize != 1024 {
		t.Errorf("Expected WriteBufferSize 1024, got %d", upgrader.WriteBufferSize)
	}

	if upgrader.CheckOrigin == nil {
		t.Error("CheckOrigin function should not be nil")
	}
}

// TestPlaybacks_InvalidCursor tests Playbacks with malformed cursor
func TestPlaybacks_InvalidCursor(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()

	handler := &Handler{
		db: db,
		config: &config.Config{
			API: config.APIConfig{
				DefaultPageSize: 100,
				MaxPageSize:     1000,
			},
		},
	}

	tests := []struct {
		name   string
		cursor string
	}{
		{
			name:   "invalid base64",
			cursor: "not-valid-base64!!!",
		},
		{
			name:   "valid base64 but invalid JSON",
			cursor: "aGVsbG8gd29ybGQ=", // "hello world" in base64
		},
		{
			name:   "empty cursor",
			cursor: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/api/v1/playbacks"
			if tt.cursor != "" {
				url += "?cursor=" + tt.cursor
			}

			req := httptest.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()

			handler.Playbacks(w, req)

			// Empty cursor should succeed (no cursor used)
			// Invalid cursor should return 400
			if tt.cursor != "" {
				if w.Code != http.StatusBadRequest && w.Code != http.StatusOK {
					t.Errorf("Expected status 400 or 200, got %d", w.Code)
				}
			}
		})
	}
}

// TestPlaybacks_OffsetBoundaries tests Playbacks with offset boundary values
func TestPlaybacks_OffsetBoundaries(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()

	handler := &Handler{
		db: db,
		config: &config.Config{
			API: config.APIConfig{
				DefaultPageSize: 100,
				MaxPageSize:     1000,
			},
		},
	}

	tests := []struct {
		name       string
		offset     string
		wantStatus int
	}{
		{"offset -1 - invalid", "-1", http.StatusBadRequest},
		{"offset 0 - valid", "0", http.StatusOK},
		{"offset 1000000 - valid max", "1000000", http.StatusOK},
		{"offset 1000001 - exceeds max", "1000001", http.StatusBadRequest},
		{"offset 999999 - valid", "999999", http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/playbacks?offset="+tt.offset, nil)
			w := httptest.NewRecorder()

			handler.Playbacks(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("Expected status %d, got %d", tt.wantStatus, w.Code)
			}
		})
	}
}

// TestPlaybacks_LimitBoundaries tests Playbacks with limit boundary values
func TestPlaybacks_LimitBoundaries(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()

	handler := &Handler{
		db: db,
		config: &config.Config{
			API: config.APIConfig{
				DefaultPageSize: 100,
				MaxPageSize:     1000,
			},
		},
	}

	tests := []struct {
		name       string
		limit      string
		wantStatus int
	}{
		{"limit 0 - invalid", "0", http.StatusBadRequest},
		{"limit -1 - invalid", "-1", http.StatusBadRequest},
		{"limit 1 - valid min", "1", http.StatusOK},
		{"limit 1000 - valid max", "1000", http.StatusOK},
		{"limit 1001 - exceeds max", "1001", http.StatusBadRequest},
		{"limit 500 - valid", "500", http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/playbacks?limit="+tt.limit, nil)
			w := httptest.NewRecorder()

			handler.Playbacks(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("Expected status %d, got %d", tt.wantStatus, w.Code)
			}
		})
	}
}

// TestPlaybacks_NilConfig tests Playbacks with nil config
func TestPlaybacks_NilConfig(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()

	handler := &Handler{
		db:     db,
		config: nil, // nil config - should use safe defaults
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/playbacks", nil)
	w := httptest.NewRecorder()

	handler.Playbacks(w, req)

	// Should succeed with default values
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 with nil config, got %d", w.Code)
	}
}

// TestLocations_NilConfig tests Locations with nil config
func TestLocations_NilConfig(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()

	handler := &Handler{
		db:     db,
		config: nil, // nil config - should use safe defaults
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/locations", nil)
	w := httptest.NewRecorder()

	handler.Locations(w, req)

	// Should succeed with default values
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 with nil config, got %d", w.Code)
	}
}

// TestBuildPlaybacksResponse_NilEvents tests buildPlaybacksResponse with nil events
func TestBuildPlaybacksResponse_NilEvents(t *testing.T) {
	t.Parallel()

	response := buildPlaybacksResponse(nil, 100, false, nil)

	// Should return empty array, not nil
	if response.Events == nil {
		t.Error("Expected empty array, got nil")
	}

	if len(response.Events) != 0 {
		t.Errorf("Expected empty array, got %d items", len(response.Events))
	}

	if response.Pagination.Limit != 100 {
		t.Errorf("Expected limit 100, got %d", response.Pagination.Limit)
	}
}

// TestEncodeCursor_WithEmptyFields tests encodeCursor with zero-value fields
func TestEncodeCursor_WithEmptyFields(t *testing.T) {
	t.Parallel()

	cursor := &models.PlaybackCursor{
		ID:        "",
		StartedAt: time.Time{},
	}

	encoded := encodeCursor(cursor)

	// Should produce valid base64 even with empty fields
	if encoded == "" {
		t.Error("encodeCursor returned empty string for cursor with empty fields")
	}

	// Should be decodable
	decoded, err := decodeCursor(encoded)
	if err != nil {
		t.Errorf("Failed to decode cursor with empty fields: %v", err)
	}

	if decoded.ID != cursor.ID {
		t.Errorf("Decoded ID mismatch: got %q, want %q", decoded.ID, cursor.ID)
	}
}
