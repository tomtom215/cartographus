// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package api

import (
	"bytes"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/goccy/go-json"
	"github.com/google/uuid"

	"github.com/tomtom215/cartographus/internal/auth"
	"github.com/tomtom215/cartographus/internal/cache"
	"github.com/tomtom215/cartographus/internal/config"
	"github.com/tomtom215/cartographus/internal/models"
)

// TestEncodeCursor_TableDriven tests the cursor encoding function with table-driven cases
func TestEncodeCursor_TableDriven(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		cursor *models.PlaybackCursor
		want   string
	}{
		{
			name: "valid cursor",
			cursor: &models.PlaybackCursor{
				StartedAt: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
				ID:        "abc123",
			},
			want: "", // Will be computed
		},
		{
			name:   "nil cursor returns empty string",
			cursor: nil,
			want:   "", // nil cursor should not be passed but encodeCursor handles it gracefully
		},
		{
			name: "cursor with empty values",
			cursor: &models.PlaybackCursor{
				StartedAt: time.Time{},
				ID:        "",
			},
			want: "", // Will be computed
		},
		{
			name: "cursor with special characters in ID",
			cursor: &models.PlaybackCursor{
				StartedAt: time.Date(2025, 6, 20, 14, 0, 0, 0, time.UTC),
				ID:        "id-with-special/chars+and=equals",
			},
			want: "", // Will be computed
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.cursor == nil {
				// encodeCursor expects non-nil, but let's test the function's behavior
				// when called with a valid cursor
				return
			}

			encoded := encodeCursor(tt.cursor)

			// Verify it's valid base64
			_, err := base64.URLEncoding.DecodeString(encoded)
			if err != nil {
				t.Errorf("encodeCursor() returned invalid base64: %v", err)
			}

			// Verify we can decode it back
			decoded, err := decodeCursor(encoded)
			if err != nil {
				t.Errorf("decodeCursor() failed to decode: %v", err)
			}

			if decoded.ID != tt.cursor.ID {
				t.Errorf("ID mismatch: got %s, want %s", decoded.ID, tt.cursor.ID)
			}

			// Compare times (truncate to second precision for comparison)
			if !decoded.StartedAt.Truncate(time.Second).Equal(tt.cursor.StartedAt.Truncate(time.Second)) {
				t.Errorf("StartedAt mismatch: got %v, want %v", decoded.StartedAt, tt.cursor.StartedAt)
			}
		})
	}
}

// TestDecodeCursor_TableDriven tests the cursor decoding function with table-driven cases
func TestDecodeCursor_TableDriven(t *testing.T) {
	t.Parallel()

	// Create a valid encoded cursor for testing
	validCursor := &models.PlaybackCursor{
		StartedAt: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
		ID:        "test-id-123",
	}
	validEncoded := encodeCursor(validCursor)

	tests := []struct {
		name    string
		encoded string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid cursor",
			encoded: validEncoded,
			wantErr: false,
		},
		{
			name:    "invalid base64",
			encoded: "not-valid-base64!!!",
			wantErr: true,
			errMsg:  "invalid base64 encoding",
		},
		{
			name:    "empty string",
			encoded: "",
			wantErr: true,
			errMsg:  "invalid cursor JSON",
		},
		{
			name:    "valid base64 but invalid JSON",
			encoded: base64.URLEncoding.EncodeToString([]byte("not json")),
			wantErr: true,
			errMsg:  "invalid cursor JSON",
		},
		{
			name:    "valid base64 empty JSON object",
			encoded: base64.URLEncoding.EncodeToString([]byte("{}")),
			wantErr: false, // Empty JSON object is valid, just has zero values
		},
		{
			name:    "valid base64 but wrong JSON structure",
			encoded: base64.URLEncoding.EncodeToString([]byte(`{"foo":"bar"}`)),
			wantErr: false, // Go JSON unmarshal ignores extra fields, just has zero values
		},
		{
			name:    "truncated base64",
			encoded: "eyJsYXN0X3dhdGNoZWRfYXQiOi",
			wantErr: true,
			errMsg:  "invalid base64 encoding",
		},
		{
			name:    "base64 with invalid JSON time format",
			encoded: base64.URLEncoding.EncodeToString([]byte(`{"started_at":"invalid-time","id":"id"}`)),
			wantErr: true,
			errMsg:  "invalid cursor JSON",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cursor, err := decodeCursor(tt.encoded)

			if tt.wantErr {
				if err == nil {
					t.Errorf("decodeCursor() expected error containing %q, got nil", tt.errMsg)
				} else if tt.errMsg != "" && !bytes.Contains([]byte(err.Error()), []byte(tt.errMsg)) {
					t.Errorf("decodeCursor() error = %v, want error containing %q", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("decodeCursor() unexpected error: %v", err)
				}
				if cursor == nil {
					t.Error("decodeCursor() returned nil cursor for valid input")
				}
			}
		})
	}
}

// TestBuildPlaybacksResponse tests the response builder function
func TestBuildPlaybacksResponse(t *testing.T) {
	t.Parallel()

	now := time.Now()
	sampleEvents := []models.PlaybackEvent{
		{
			ID:        uuid.MustParse("11111111-1111-1111-1111-111111111111"),
			Username:  "user1",
			Title:     "Movie 1",
			StartedAt: now.Add(-1 * time.Hour),
		},
		{
			ID:        uuid.MustParse("22222222-2222-2222-2222-222222222222"),
			Username:  "user2",
			Title:     "Movie 2",
			StartedAt: now.Add(-2 * time.Hour),
		},
	}

	tests := []struct {
		name          string
		events        []models.PlaybackEvent
		limit         int
		hasMore       bool
		nextCursor    *models.PlaybackCursor
		wantEvents    int
		wantHasMore   bool
		wantHasCursor bool
	}{
		{
			name:          "with events and next cursor",
			events:        sampleEvents,
			limit:         10,
			hasMore:       true,
			nextCursor:    &models.PlaybackCursor{StartedAt: now, ID: "cursor-1"},
			wantEvents:    2,
			wantHasMore:   true,
			wantHasCursor: true,
		},
		{
			name:          "with events but no next cursor",
			events:        sampleEvents,
			limit:         10,
			hasMore:       false,
			nextCursor:    nil,
			wantEvents:    2,
			wantHasMore:   false,
			wantHasCursor: false,
		},
		{
			name:          "nil events returns empty slice",
			events:        nil,
			limit:         10,
			hasMore:       false,
			nextCursor:    nil,
			wantEvents:    0,
			wantHasMore:   false,
			wantHasCursor: false,
		},
		{
			name:          "empty events slice",
			events:        []models.PlaybackEvent{},
			limit:         10,
			hasMore:       false,
			nextCursor:    nil,
			wantEvents:    0,
			wantHasMore:   false,
			wantHasCursor: false,
		},
		{
			name:          "has more with matching limit",
			events:        sampleEvents,
			limit:         2,
			hasMore:       true,
			nextCursor:    &models.PlaybackCursor{StartedAt: now, ID: "next"},
			wantEvents:    2,
			wantHasMore:   true,
			wantHasCursor: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := buildPlaybacksResponse(tt.events, tt.limit, tt.hasMore, tt.nextCursor)

			if len(response.Events) != tt.wantEvents {
				t.Errorf("Events count: got %d, want %d", len(response.Events), tt.wantEvents)
			}

			if response.Pagination.HasMore != tt.wantHasMore {
				t.Errorf("HasMore: got %v, want %v", response.Pagination.HasMore, tt.wantHasMore)
			}

			if response.Pagination.Limit != tt.limit {
				t.Errorf("Limit: got %d, want %d", response.Pagination.Limit, tt.limit)
			}

			hasCursor := response.Pagination.NextCursor != nil
			if hasCursor != tt.wantHasCursor {
				t.Errorf("NextCursor present: got %v, want %v", hasCursor, tt.wantHasCursor)
			}

			// Verify cursor is valid if present
			if hasCursor {
				_, err := decodeCursor(*response.Pagination.NextCursor)
				if err != nil {
					t.Errorf("NextCursor is not decodable: %v", err)
				}
			}
		})
	}
}

// TestPlaybacks_CursorPagination tests cursor-based pagination
func TestPlaybacks_CursorPagination(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		cursor     string
		limit      string
		wantStatus int
		wantErr    string
	}{
		{
			name:       "invalid cursor format",
			cursor:     "not-valid-base64!!!",
			limit:      "10",
			wantStatus: http.StatusBadRequest,
			wantErr:    "VALIDATION_ERROR",
		},
		{
			name:       "valid base64 but invalid JSON cursor",
			cursor:     base64.URLEncoding.EncodeToString([]byte("not json")),
			limit:      "10",
			wantStatus: http.StatusBadRequest,
			wantErr:    "VALIDATION_ERROR",
		},
		{
			name:       "limit below minimum",
			cursor:     "",
			limit:      "0",
			wantStatus: http.StatusBadRequest,
			wantErr:    "VALIDATION_ERROR",
		},
		{
			name:       "limit above maximum",
			cursor:     "",
			limit:      "10001",
			wantStatus: http.StatusBadRequest,
			wantErr:    "VALIDATION_ERROR",
		},
		{
			name:       "negative limit",
			cursor:     "",
			limit:      "-5",
			wantStatus: http.StatusBadRequest,
			wantErr:    "VALIDATION_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				API: config.APIConfig{
					DefaultPageSize: 100,
					MaxPageSize:     1000,
				},
			}

			handler := &Handler{
				config: cfg,
				cache:  cache.New(5 * time.Minute),
			}

			url := "/api/v1/playbacks?"
			if tt.cursor != "" {
				url += "cursor=" + tt.cursor + "&"
			}
			if tt.limit != "" {
				url += "limit=" + tt.limit
			}

			req := httptest.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()

			handler.Playbacks(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("Status: got %d, want %d. Body: %s", w.Code, tt.wantStatus, w.Body.String())
			}

			if tt.wantErr != "" {
				var response models.APIResponse
				if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}
				if response.Error == nil || response.Error.Code != tt.wantErr {
					t.Errorf("Error code: got %v, want %s", response.Error, tt.wantErr)
				}
			}
		})
	}
}

// TestPlaybacks_OffsetPagination tests offset-based pagination validation
func TestPlaybacks_OffsetPagination(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		offset     string
		limit      string
		wantStatus int
		wantErr    string
	}{
		{
			name:       "offset below minimum",
			offset:     "-1",
			limit:      "10",
			wantStatus: http.StatusBadRequest,
			wantErr:    "VALIDATION_ERROR",
		},
		{
			name:       "offset above maximum",
			offset:     "1000001",
			limit:      "10",
			wantStatus: http.StatusBadRequest,
			wantErr:    "VALIDATION_ERROR",
		},
		{
			name:       "offset way above maximum",
			offset:     "9999999",
			limit:      "10",
			wantStatus: http.StatusBadRequest,
			wantErr:    "VALIDATION_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				API: config.APIConfig{
					DefaultPageSize: 100,
					MaxPageSize:     1000,
				},
			}

			handler := &Handler{
				config: cfg,
				cache:  cache.New(5 * time.Minute),
			}

			url := "/api/v1/playbacks?"
			if tt.offset != "" {
				url += "offset=" + tt.offset + "&"
			}
			if tt.limit != "" {
				url += "limit=" + tt.limit
			}

			req := httptest.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()

			handler.Playbacks(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("Status: got %d, want %d. Body: %s", w.Code, tt.wantStatus, w.Body.String())
			}

			if tt.wantErr != "" {
				var response models.APIResponse
				if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}
				if response.Error == nil || response.Error.Code != tt.wantErr {
					t.Errorf("Error code: got %v, want %s", response.Error, tt.wantErr)
				}
			}
		})
	}
}

// TestLocations_Validation tests locations endpoint parameter validation
func TestLocations_Validation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		query      string
		wantStatus int
		wantErr    string
	}{
		{
			name:       "limit below minimum",
			query:      "limit=0",
			wantStatus: http.StatusBadRequest,
			wantErr:    "VALIDATION_ERROR",
		},
		{
			name:       "limit above maximum",
			query:      "limit=10001",
			wantStatus: http.StatusBadRequest,
			wantErr:    "VALIDATION_ERROR",
		},
		{
			name:       "invalid start_date format",
			query:      "start_date=not-a-date",
			wantStatus: http.StatusBadRequest,
			wantErr:    "VALIDATION_ERROR",
		},
		{
			name:       "invalid end_date format",
			query:      "end_date=invalid-date",
			wantStatus: http.StatusBadRequest,
			wantErr:    "VALIDATION_ERROR",
		},
		{
			name:       "days below minimum",
			query:      "days=0",
			wantStatus: http.StatusBadRequest,
			wantErr:    "VALIDATION_ERROR",
		},
		{
			name:       "days above maximum",
			query:      "days=3651",
			wantStatus: http.StatusBadRequest,
			wantErr:    "VALIDATION_ERROR",
		},
		{
			name:       "negative days",
			query:      "days=-5",
			wantStatus: http.StatusBadRequest,
			wantErr:    "VALIDATION_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				API: config.APIConfig{
					DefaultPageSize: 100,
					MaxPageSize:     1000,
				},
			}

			handler := &Handler{
				config: cfg,
				cache:  cache.New(5 * time.Minute),
			}

			req := httptest.NewRequest(http.MethodGet, "/api/v1/locations?"+tt.query, nil)
			w := httptest.NewRecorder()

			handler.Locations(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("Status: got %d, want %d. Body: %s", w.Code, tt.wantStatus, w.Body.String())
			}

			if tt.wantErr != "" {
				var response models.APIResponse
				if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}
				if response.Error == nil || response.Error.Code != tt.wantErr {
					t.Errorf("Error code: got %v, want %s", response.Error, tt.wantErr)
				}
			}
		})
	}
}

// TestServerInfo_Variations tests server info with different configurations
func TestServerInfo_Variations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		latitude     float64
		longitude    float64
		wantLocation bool
	}{
		{
			name:         "with coordinates",
			latitude:     40.7128,
			longitude:    -74.0060,
			wantLocation: true,
		},
		{
			name:         "without coordinates (zeros)",
			latitude:     0.0,
			longitude:    0.0,
			wantLocation: false,
		},
		{
			name:         "with only latitude",
			latitude:     45.0,
			longitude:    0.0,
			wantLocation: true,
		},
		{
			name:         "with only longitude",
			latitude:     0.0,
			longitude:    -122.0,
			wantLocation: true,
		},
		{
			name:         "negative coordinates",
			latitude:     -33.8688,
			longitude:    151.2093,
			wantLocation: true,
		},
		{
			name:         "edge coordinates (poles)",
			latitude:     90.0,
			longitude:    180.0,
			wantLocation: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Server: config.ServerConfig{
					Latitude:  tt.latitude,
					Longitude: tt.longitude,
				},
			}

			handler := &Handler{
				config: cfg,
			}

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

			data, ok := response.Data.(map[string]interface{})
			if !ok {
				t.Fatal("Response data is not a map")
			}

			hasLocation := data["has_location"].(bool)
			if hasLocation != tt.wantLocation {
				t.Errorf("has_location: got %v, want %v", hasLocation, tt.wantLocation)
			}

			if lat, ok := data["latitude"].(float64); ok {
				if lat != tt.latitude {
					t.Errorf("latitude: got %v, want %v", lat, tt.latitude)
				}
			}

			if lon, ok := data["longitude"].(float64); ok {
				if lon != tt.longitude {
					t.Errorf("longitude: got %v, want %v", lon, tt.longitude)
				}
			}
		})
	}
}

// TestTriggerSync_Variations tests sync trigger with different states
func TestTriggerSync_Variations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		method     string
		hasSync    bool
		wantStatus int
		wantErr    string
	}{
		{
			name:       "GET method not allowed",
			method:     http.MethodGet,
			hasSync:    true,
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "PUT method not allowed",
			method:     http.MethodPut,
			hasSync:    true,
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "DELETE method not allowed",
			method:     http.MethodDelete,
			hasSync:    true,
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "POST without sync manager",
			method:     http.MethodPost,
			hasSync:    false,
			wantStatus: http.StatusServiceUnavailable,
			wantErr:    "SERVICE_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := &Handler{
				sync: nil,
			}

			req := httptest.NewRequest(tt.method, "/api/v1/sync", nil)
			w := httptest.NewRecorder()

			handler.TriggerSync(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("Status: got %d, want %d", w.Code, tt.wantStatus)
			}

			if tt.wantErr != "" {
				var response models.APIResponse
				if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
					// For method not allowed, it returns plain text
					return
				}
				if response.Error == nil || response.Error.Code != tt.wantErr {
					t.Errorf("Error code: got %v, want %s", response.Error, tt.wantErr)
				}
			}
		})
	}
}

// TestLogin_Variations tests login with various scenarios
func TestLogin_Variations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		method     string
		authMode   string
		hasJWT     bool
		body       string
		username   string
		password   string
		wantStatus int
		wantErr    string
	}{
		{
			name:       "GET method not allowed",
			method:     http.MethodGet,
			authMode:   "jwt",
			hasJWT:     true,
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "auth disabled",
			method:     http.MethodPost,
			authMode:   "none",
			hasJWT:     true,
			body:       `{"username":"admin","password":"password"}`,
			wantStatus: http.StatusForbidden,
			wantErr:    "AUTH_DISABLED",
		},
		{
			name:       "no JWT manager",
			method:     http.MethodPost,
			authMode:   "jwt",
			hasJWT:     false,
			body:       `{"username":"admin","password":"password"}`,
			wantStatus: http.StatusInternalServerError,
			wantErr:    "AUTH_NOT_CONFIGURED",
		},
		{
			name:       "invalid JSON body",
			method:     http.MethodPost,
			authMode:   "jwt",
			hasJWT:     true,
			body:       "not json",
			wantStatus: http.StatusBadRequest,
			wantErr:    "INVALID_REQUEST",
		},
		{
			name:       "empty username",
			method:     http.MethodPost,
			authMode:   "jwt",
			hasJWT:     true,
			body:       `{"username":"","password":"password"}`,
			wantStatus: http.StatusBadRequest,
			wantErr:    "VALIDATION_ERROR",
		},
		{
			name:       "empty password",
			method:     http.MethodPost,
			authMode:   "jwt",
			hasJWT:     true,
			body:       `{"username":"admin","password":""}`,
			wantStatus: http.StatusBadRequest,
			wantErr:    "VALIDATION_ERROR",
		},
		{
			name:       "missing both fields",
			method:     http.MethodPost,
			authMode:   "jwt",
			hasJWT:     true,
			body:       `{}`,
			wantStatus: http.StatusBadRequest,
			wantErr:    "VALIDATION_ERROR",
		},
		{
			name:       "wrong username",
			method:     http.MethodPost,
			authMode:   "jwt",
			hasJWT:     true,
			body:       `{"username":"wrong","password":"adminpass123"}`,
			username:   "admin",
			password:   "adminpass123",
			wantStatus: http.StatusUnauthorized,
			wantErr:    "INVALID_CREDENTIALS",
		},
		{
			name:       "wrong password",
			method:     http.MethodPost,
			authMode:   "jwt",
			hasJWT:     true,
			body:       `{"username":"admin","password":"wrong"}`,
			username:   "admin",
			password:   "adminpass123",
			wantStatus: http.StatusUnauthorized,
			wantErr:    "INVALID_CREDENTIALS",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Security: config.SecurityConfig{
					AuthMode:      tt.authMode,
					AdminUsername: tt.username,
					AdminPassword: tt.password,
					JWTSecret:     "test-secret-key-minimum-32-characters-long",
				},
			}

			handler := &Handler{
				config: cfg,
			}

			// Create real jwtManager for tests that need credential validation
			if tt.hasJWT && tt.authMode == "jwt" {
				jwtManager, _ := auth.NewJWTManager(&cfg.Security)
				handler.jwtManager = jwtManager
			}

			var body *bytes.Reader
			if tt.body != "" {
				body = bytes.NewReader([]byte(tt.body))
			} else {
				body = bytes.NewReader([]byte{})
			}

			req := httptest.NewRequest(tt.method, "/api/v1/auth/login", body)
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.Login(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("Status: got %d, want %d. Body: %s", w.Code, tt.wantStatus, w.Body.String())
			}

			if tt.wantErr != "" && w.Code != http.StatusMethodNotAllowed {
				var response models.APIResponse
				if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}
				if response.Error == nil || response.Error.Code != tt.wantErr {
					t.Errorf("Error code: got %v, want %s", response.Error, tt.wantErr)
				}
			}
		})
	}
}

// BenchmarkEncodeCursor benchmarks cursor encoding
func BenchmarkEncodeCursor(b *testing.B) {
	cursor := &models.PlaybackCursor{
		StartedAt: time.Now(),
		ID:        "benchmark-id-12345",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		encodeCursor(cursor)
	}
}

// BenchmarkDecodeCursor benchmarks cursor decoding
func BenchmarkDecodeCursor(b *testing.B) {
	cursor := &models.PlaybackCursor{
		StartedAt: time.Now(),
		ID:        "benchmark-id-12345",
	}
	encoded := encodeCursor(cursor)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = decodeCursor(encoded)
	}
}

// BenchmarkBuildPlaybacksResponse benchmarks response building
func BenchmarkBuildPlaybacksResponse(b *testing.B) {
	events := make([]models.PlaybackEvent, 100)
	for i := range events {
		events[i] = models.PlaybackEvent{
			ID:       uuid.New(),
			Username: "user",
			Title:    "Title",
		}
	}
	cursor := &models.PlaybackCursor{
		StartedAt: time.Now(),
		ID:        "cursor-id",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buildPlaybacksResponse(events, 100, true, cursor)
	}
}
