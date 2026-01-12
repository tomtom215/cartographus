// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/goccy/go-json"
	syncpkg "github.com/tomtom215/cartographus/internal/sync"
	ws "github.com/tomtom215/cartographus/internal/websocket"

	"github.com/tomtom215/cartographus/internal/auth"
	"github.com/tomtom215/cartographus/internal/cache"
	"github.com/tomtom215/cartographus/internal/config"
	"github.com/tomtom215/cartographus/internal/database"
	"github.com/tomtom215/cartographus/internal/models"
)

// setupCoreTestHandler creates a handler with full dependencies for core endpoint testing
func setupCoreTestHandler(t *testing.T) (*Handler, *database.DB) {
	t.Helper()

	// Create in-memory database
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
	}

	// Create JWT manager
	jwtManager, err := auth.NewJWTManager(&testConfig.Security)
	if err != nil {
		t.Fatalf("Failed to create JWT manager: %v", err)
	}

	// Create WebSocket hub
	wsHub := ws.NewHub()
	go wsHub.Run()

	// Create mock client
	mockClient := &MockTautulliClient{}

	// Create sync manager
	syncCfg := &config.Config{
		Sync: config.SyncConfig{
			Interval: 30 * time.Minute,
		},
	}
	syncManager := syncpkg.NewManager(db, nil, mockClient, syncCfg, wsHub)

	// Create cache
	c := cache.New(5 * time.Minute)

	// Create handler
	handler := &Handler{
		db:         db,
		sync:       syncManager,
		cache:      c,
		client:     mockClient,
		config:     testConfig,
		jwtManager: jwtManager,
		wsHub:      wsHub,
	}

	return handler, db
}

// TestStatsEnhanced tests the Stats handler with enhanced coverage
func TestStatsEnhanced(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		expectedStatus int
		validateResp   func(*testing.T, *models.APIResponse)
	}{
		{
			name:           "success - retrieve stats",
			expectedStatus: http.StatusOK,
			validateResp: func(t *testing.T, resp *models.APIResponse) {
				if resp.Status != "success" {
					t.Errorf("Expected status 'success', got '%s'", resp.Status)
				}
				if resp.Data == nil {
					t.Error("Expected data to be present")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, db := setupCoreTestHandler(t)
			defer db.Close()

			req := httptest.NewRequest(http.MethodGet, "/api/v1/stats", nil)
			w := httptest.NewRecorder()

			handler.Stats(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
				t.Logf("Response body: %s", w.Body.String())
			}

			var response models.APIResponse
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if tt.validateResp != nil {
				tt.validateResp(t, &response)
			}
		})
	}
}

// TestPlaybacksEnhanced tests the Playbacks handler with enhanced coverage
func TestPlaybacksEnhanced(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
		validateResp   func(*testing.T, *models.APIResponse)
	}{
		{
			name:           "success - default pagination",
			queryParams:    "",
			expectedStatus: http.StatusOK,
			validateResp: func(t *testing.T, resp *models.APIResponse) {
				if resp.Status != "success" {
					t.Errorf("Expected status 'success', got '%s'", resp.Status)
				}
			},
		},
		{
			name:           "success - with limit and offset",
			queryParams:    "limit=50&offset=0",
			expectedStatus: http.StatusOK,
			validateResp: func(t *testing.T, resp *models.APIResponse) {
				if resp.Status != "success" {
					t.Errorf("Expected status 'success', got '%s'", resp.Status)
				}
			},
		},
		{
			name:           "success - with date range",
			queryParams:    "start_date=2025-01-01T00:00:00Z&end_date=2025-01-31T23:59:59Z",
			expectedStatus: http.StatusOK,
			validateResp: func(t *testing.T, resp *models.APIResponse) {
				if resp.Status != "success" {
					t.Errorf("Expected status 'success', got '%s'", resp.Status)
				}
			},
		},
		{
			name:           "success - with username filter",
			queryParams:    "username=testuser",
			expectedStatus: http.StatusOK,
			validateResp: func(t *testing.T, resp *models.APIResponse) {
				if resp.Status != "success" {
					t.Errorf("Expected status 'success', got '%s'", resp.Status)
				}
			},
		},
		{
			name:           "success - with media type filter",
			queryParams:    "media_type=movie",
			expectedStatus: http.StatusOK,
			validateResp: func(t *testing.T, resp *models.APIResponse) {
				if resp.Status != "success" {
					t.Errorf("Expected status 'success', got '%s'", resp.Status)
				}
			},
		},
		{
			name:           "success - with platform filter",
			queryParams:    "platform=Roku",
			expectedStatus: http.StatusOK,
			validateResp: func(t *testing.T, resp *models.APIResponse) {
				if resp.Status != "success" {
					t.Errorf("Expected status 'success', got '%s'", resp.Status)
				}
			},
		},
		{
			name:           "error - invalid limit (too high)",
			queryParams:    "limit=10000",
			expectedStatus: http.StatusBadRequest,
			validateResp: func(t *testing.T, resp *models.APIResponse) {
				if resp.Status != "error" {
					t.Errorf("Expected status 'error', got '%s'", resp.Status)
				}
			},
		},
		{
			name:           "error - invalid offset (negative)",
			queryParams:    "offset=-1",
			expectedStatus: http.StatusBadRequest,
			validateResp: func(t *testing.T, resp *models.APIResponse) {
				if resp.Status != "error" {
					t.Errorf("Expected status 'error', got '%s'", resp.Status)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, db := setupCoreTestHandler(t)
			defer db.Close()

			url := "/api/v1/playbacks"
			if tt.queryParams != "" {
				url += "?" + tt.queryParams
			}

			req := httptest.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()

			handler.Playbacks(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
				t.Logf("Response body: %s", w.Body.String())
			}

			var response models.APIResponse
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if tt.validateResp != nil {
				tt.validateResp(t, &response)
			}
		})
	}
}

// TestLocationsEnhanced tests the Locations handler with enhanced coverage
func TestLocationsEnhanced(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
	}{
		{
			name:           "success - all locations",
			queryParams:    "",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "success - with date range",
			queryParams:    "start_date=2025-01-01T00:00:00Z&end_date=2025-01-31T23:59:59Z",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "success - with country filter",
			queryParams:    "country=US",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "success - with city filter",
			queryParams:    "city=New+York",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, db := setupCoreTestHandler(t)
			defer db.Close()

			url := "/api/v1/locations"
			if tt.queryParams != "" {
				url += "?" + tt.queryParams
			}

			req := httptest.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()

			handler.Locations(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
				t.Logf("Response body: %s", w.Body.String())
			}
		})
	}
}

// TestTriggerSync tests the TriggerSync handler
func TestTriggerSync(t *testing.T) {
	t.Parallel()

	handler, db := setupCoreTestHandler(t)
	defer db.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sync/trigger", nil)
	w := httptest.NewRecorder()

	handler.TriggerSync(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("Expected status 202 (Accepted), got %d", w.Code)
		t.Logf("Response body: %s", w.Body.String())
	}

	var response models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Status != "success" {
		t.Errorf("Expected status 'success', got '%s'", response.Status)
	}
}

// TestUsersEnhanced tests the Users handler with enhanced coverage
func TestUsersEnhanced(t *testing.T) {
	t.Parallel()

	handler, db := setupCoreTestHandler(t)
	defer db.Close()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	w := httptest.NewRecorder()

	handler.Users(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
		t.Logf("Response body: %s", w.Body.String())
	}

	var response models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Status != "success" {
		t.Errorf("Expected status 'success', got '%s'", response.Status)
	}
}

// TestServerInfoEnhanced tests the ServerInfo handler with enhanced coverage
func TestServerInfoEnhanced(t *testing.T) {
	t.Parallel()

	handler, db := setupCoreTestHandler(t)
	defer db.Close()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/server/info", nil)
	w := httptest.NewRecorder()

	handler.ServerInfo(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
		t.Logf("Response body: %s", w.Body.String())
	}

	var response models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Status != "success" {
		t.Errorf("Expected status 'success', got '%s'", response.Status)
	}
}

// TestMediaTypesEnhanced tests the MediaTypes handler with enhanced coverage
func TestMediaTypesEnhanced(t *testing.T) {
	t.Parallel()

	handler, db := setupCoreTestHandler(t)
	defer db.Close()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/media-types", nil)
	w := httptest.NewRecorder()

	handler.MediaTypes(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
		t.Logf("Response body: %s", w.Body.String())
	}

	var response models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Status != "success" {
		t.Errorf("Expected status 'success', got '%s'", response.Status)
	}
}

// TestLogin tests the Login handler
func TestLogin(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		username       string
		password       string
		expectedStatus int
		validateResp   func(*testing.T, *models.APIResponse)
	}{
		{
			name:           "success - valid credentials",
			username:       "admin",
			password:       "password123",
			expectedStatus: http.StatusOK,
			validateResp: func(t *testing.T, resp *models.APIResponse) {
				if resp.Status != "success" {
					t.Errorf("Expected status 'success', got '%s'", resp.Status)
				}
				// Check that token is present in response
				if resp.Data == nil {
					t.Error("Expected data to be present")
				}
			},
		},
		{
			name:           "error - invalid credentials",
			username:       "admin",
			password:       "wrongpassword",
			expectedStatus: http.StatusUnauthorized,
			validateResp: func(t *testing.T, resp *models.APIResponse) {
				if resp.Status != "error" {
					t.Errorf("Expected status 'error', got '%s'", resp.Status)
				}
			},
		},
		{
			name:           "error - missing username",
			username:       "",
			password:       "password123",
			expectedStatus: http.StatusBadRequest,
			validateResp: func(t *testing.T, resp *models.APIResponse) {
				if resp.Status != "error" {
					t.Errorf("Expected status 'error', got '%s'", resp.Status)
				}
			},
		},
		{
			name:           "error - missing password",
			username:       "admin",
			password:       "",
			expectedStatus: http.StatusBadRequest,
			validateResp: func(t *testing.T, resp *models.APIResponse) {
				if resp.Status != "error" {
					t.Errorf("Expected status 'error', got '%s'", resp.Status)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, db := setupCoreTestHandler(t)
			defer db.Close()

			// Create login request
			loginReq := map[string]string{
				"username": tt.username,
				"password": tt.password,
			}
			body, _ := json.Marshal(loginReq)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.Login(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
				t.Logf("Response body: %s", w.Body.String())
			}

			var response models.APIResponse
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if tt.validateResp != nil {
				tt.validateResp(t, &response)
			}
		})
	}
}

// TestPlaybacksCaching tests caching behavior for Playbacks handler
func TestPlaybacksCaching(t *testing.T) {
	t.Parallel()

	handler, db := setupCoreTestHandler(t)
	defer db.Close()

	// First request
	req1 := httptest.NewRequest(http.MethodGet, "/api/v1/playbacks", nil)
	w1 := httptest.NewRecorder()
	handler.Playbacks(w1, req1)

	// Second request (should potentially use cache)
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/playbacks", nil)
	w2 := httptest.NewRecorder()
	handler.Playbacks(w2, req2)

	// Both should succeed
	if w1.Code != http.StatusOK || w2.Code != http.StatusOK {
		t.Errorf("Expected both requests to succeed, got %d and %d", w1.Code, w2.Code)
	}
}

// TestPlaybacksPagination tests pagination logic
func TestPlaybacksPagination(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		limit          string
		offset         string
		expectedStatus int
	}{
		{
			name:           "valid pagination - page 1",
			limit:          "20",
			offset:         "0",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "valid pagination - page 2",
			limit:          "20",
			offset:         "20",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "valid pagination - large limit",
			limit:          "100",
			offset:         "0",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid - limit exceeds max",
			limit:          "1000",
			offset:         "0",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid - negative offset",
			limit:          "20",
			offset:         "-10",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid - zero limit",
			limit:          "0",
			offset:         "0",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, db := setupCoreTestHandler(t)
			defer db.Close()

			url := "/api/v1/playbacks?limit=" + tt.limit + "&offset=" + tt.offset
			req := httptest.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()

			handler.Playbacks(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
				t.Logf("Response body: %s", w.Body.String())
			}
		})
	}
}

// TestLoginInvalidJSON tests Login handler with malformed JSON
func TestLoginInvalidJSON(t *testing.T) {
	t.Parallel()

	handler, db := setupCoreTestHandler(t)
	defer db.Close()

	// Send malformed JSON
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader("{invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Login(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

// TestTriggerSyncMethodNotAllowed tests TriggerSync with wrong HTTP method
func TestTriggerSyncMethodNotAllowed(t *testing.T) {
	t.Parallel()

	handler, db := setupCoreTestHandler(t)
	defer db.Close()

	// Send GET instead of POST
	req := httptest.NewRequest(http.MethodGet, "/api/v1/sync/trigger", nil)
	w := httptest.NewRecorder()

	handler.TriggerSync(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}
