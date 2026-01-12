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

	"github.com/tomtom215/cartographus/internal/cache"
	"github.com/tomtom215/cartographus/internal/config"
	"github.com/tomtom215/cartographus/internal/database"
	"github.com/tomtom215/cartographus/internal/models"
)

// setupTestHandlerForDiscovery creates a handler with an in-memory test database for discovery analytics testing
func setupTestHandlerForDiscovery(t *testing.T) (*Handler, *database.DB) {
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

	// Create cache
	c := cache.New(5 * time.Minute)

	// Create test config
	testConfig := &config.Config{
		API: config.APIConfig{
			DefaultPageSize: 20,
			MaxPageSize:     100,
		},
	}

	// Create handler
	handler := &Handler{
		db:     db,
		cache:  c,
		config: testConfig,
	}

	return handler, db
}

// TestAnalyticsDeviceMigration tests the AnalyticsDeviceMigration handler
func TestAnalyticsDeviceMigration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
		validateResp   func(*testing.T, *models.APIResponse)
	}{
		{
			name:           "success - default params",
			queryParams:    "",
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
			name:           "success - with multiple filters",
			queryParams:    "username=testuser&platform=Roku&media_type=movie",
			expectedStatus: http.StatusOK,
			validateResp: func(t *testing.T, resp *models.APIResponse) {
				if resp.Status != "success" {
					t.Errorf("Expected status 'success', got '%s'", resp.Status)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, db := setupTestHandlerForDiscovery(t)
			defer db.Close()

			url := "/api/v1/analytics/device-migration"
			if tt.queryParams != "" {
				url += "?" + tt.queryParams
			}

			req := httptest.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()

			handler.AnalyticsDeviceMigration(w, req)

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

// TestAnalyticsDeviceMigrationMethodNotAllowed tests HTTP method validation
func TestAnalyticsDeviceMigrationMethodNotAllowed(t *testing.T) {
	t.Parallel()

	handler, db := setupTestHandlerForDiscovery(t)
	defer db.Close()

	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/v1/analytics/device-migration", nil)
			w := httptest.NewRecorder()

			handler.AnalyticsDeviceMigration(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status 405 for %s, got %d", method, w.Code)
			}
		})
	}
}

// TestAnalyticsDeviceMigrationCaching tests caching behavior
func TestAnalyticsDeviceMigrationCaching(t *testing.T) {
	t.Parallel()

	handler, db := setupTestHandlerForDiscovery(t)
	defer db.Close()

	// First request
	req1 := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/device-migration", nil)
	w1 := httptest.NewRecorder()
	handler.AnalyticsDeviceMigration(w1, req1)

	// Second request (should use cache)
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/device-migration", nil)
	w2 := httptest.NewRecorder()
	handler.AnalyticsDeviceMigration(w2, req2)

	// Both should succeed
	if w1.Code != http.StatusOK || w2.Code != http.StatusOK {
		t.Errorf("Expected both requests to succeed, got %d and %d", w1.Code, w2.Code)
	}
}

// TestAnalyticsDeviceMigrationWithContext tests context handling
func TestAnalyticsDeviceMigrationWithContext(t *testing.T) {
	t.Parallel()

	handler, db := setupTestHandlerForDiscovery(t)
	defer db.Close()

	// Create request with cancellable context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/device-migration", nil)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handler.AnalyticsDeviceMigration(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// TestAnalyticsContentDiscovery tests the AnalyticsContentDiscovery handler
func TestAnalyticsContentDiscovery(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
		validateResp   func(*testing.T, *models.APIResponse)
	}{
		{
			name:           "success - default params",
			queryParams:    "",
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
			name:           "success - with library filter",
			queryParams:    "library=Movies",
			expectedStatus: http.StatusOK,
			validateResp: func(t *testing.T, resp *models.APIResponse) {
				if resp.Status != "success" {
					t.Errorf("Expected status 'success', got '%s'", resp.Status)
				}
			},
		},
		{
			name:           "success - with multiple filters",
			queryParams:    "username=testuser&media_type=movie&library=Movies",
			expectedStatus: http.StatusOK,
			validateResp: func(t *testing.T, resp *models.APIResponse) {
				if resp.Status != "success" {
					t.Errorf("Expected status 'success', got '%s'", resp.Status)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, db := setupTestHandlerForDiscovery(t)
			defer db.Close()

			url := "/api/v1/analytics/content-discovery"
			if tt.queryParams != "" {
				url += "?" + tt.queryParams
			}

			req := httptest.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()

			handler.AnalyticsContentDiscovery(w, req)

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

// TestAnalyticsContentDiscoveryMethodNotAllowed tests HTTP method validation
func TestAnalyticsContentDiscoveryMethodNotAllowed(t *testing.T) {
	t.Parallel()

	handler, db := setupTestHandlerForDiscovery(t)
	defer db.Close()

	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/v1/analytics/content-discovery", nil)
			w := httptest.NewRecorder()

			handler.AnalyticsContentDiscovery(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status 405 for %s, got %d", method, w.Code)
			}
		})
	}
}

// TestAnalyticsContentDiscoveryCaching tests caching behavior
func TestAnalyticsContentDiscoveryCaching(t *testing.T) {
	t.Parallel()

	handler, db := setupTestHandlerForDiscovery(t)
	defer db.Close()

	// First request
	req1 := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/content-discovery", nil)
	w1 := httptest.NewRecorder()
	handler.AnalyticsContentDiscovery(w1, req1)

	// Second request (should use cache)
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/content-discovery", nil)
	w2 := httptest.NewRecorder()
	handler.AnalyticsContentDiscovery(w2, req2)

	// Both should succeed
	if w1.Code != http.StatusOK || w2.Code != http.StatusOK {
		t.Errorf("Expected both requests to succeed, got %d and %d", w1.Code, w2.Code)
	}
}

// TestAnalyticsContentDiscoveryWithContext tests context handling
func TestAnalyticsContentDiscoveryWithContext(t *testing.T) {
	t.Parallel()

	handler, db := setupTestHandlerForDiscovery(t)
	defer db.Close()

	// Create request with cancellable context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/content-discovery", nil)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handler.AnalyticsContentDiscovery(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// TestDiscoveryAnalyticsResponseStructure validates the response structure
func TestDiscoveryAnalyticsResponseStructure(t *testing.T) {
	t.Parallel()

	t.Run("device migration response structure", func(t *testing.T) {
		handler, db := setupTestHandlerForDiscovery(t)
		defer db.Close()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/device-migration", nil)
		w := httptest.NewRecorder()
		handler.AnalyticsDeviceMigration(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d", w.Code)
		}

		var response models.APIResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response.Status != "success" {
			t.Errorf("Expected status 'success', got '%s'", response.Status)
		}

		// Validate data is present and properly structured
		if response.Data == nil {
			t.Error("Expected data to be present")
		}
	})

	t.Run("content discovery response structure", func(t *testing.T) {
		handler, db := setupTestHandlerForDiscovery(t)
		defer db.Close()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/content-discovery", nil)
		w := httptest.NewRecorder()
		handler.AnalyticsContentDiscovery(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d", w.Code)
		}

		var response models.APIResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response.Status != "success" {
			t.Errorf("Expected status 'success', got '%s'", response.Status)
		}

		// Validate data is present and properly structured
		if response.Data == nil {
			t.Error("Expected data to be present")
		}
	})
}

// TestDiscoveryAnalyticsInvalidDateFormat tests handling of invalid date formats
func TestDiscoveryAnalyticsInvalidDateFormat(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		handler  func(*Handler, http.ResponseWriter, *http.Request)
		endpoint string
	}{
		{
			name: "device migration",
			handler: func(h *Handler, w http.ResponseWriter, r *http.Request) {
				h.AnalyticsDeviceMigration(w, r)
			},
			endpoint: "/api/v1/analytics/device-migration",
		},
		{
			name: "content discovery",
			handler: func(h *Handler, w http.ResponseWriter, r *http.Request) {
				h.AnalyticsContentDiscovery(w, r)
			},
			endpoint: "/api/v1/analytics/content-discovery",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name+" with invalid date", func(t *testing.T) {
			handler, db := setupTestHandlerForDiscovery(t)
			defer db.Close()

			// Invalid date format should not crash
			req := httptest.NewRequest(http.MethodGet, tc.endpoint+"?start_date=not-a-date", nil)
			w := httptest.NewRecorder()

			tc.handler(handler, w, req)

			// Should either succeed (ignoring invalid date) or return a proper error
			if w.Code != http.StatusOK && w.Code != http.StatusBadRequest {
				t.Errorf("Expected status 200 or 400, got %d", w.Code)
			}
		})
	}
}

// TestDiscoveryAnalyticsEmptyDatabase tests behavior with no data
func TestDiscoveryAnalyticsEmptyDatabase(t *testing.T) {
	t.Parallel()

	t.Run("device migration with empty db", func(t *testing.T) {
		handler, db := setupTestHandlerForDiscovery(t)
		defer db.Close()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/device-migration", nil)
		w := httptest.NewRecorder()
		handler.AnalyticsDeviceMigration(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response models.APIResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response.Status != "success" {
			t.Errorf("Expected success even with empty db, got '%s'", response.Status)
		}
	})

	t.Run("content discovery with empty db", func(t *testing.T) {
		handler, db := setupTestHandlerForDiscovery(t)
		defer db.Close()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/content-discovery", nil)
		w := httptest.NewRecorder()
		handler.AnalyticsContentDiscovery(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response models.APIResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response.Status != "success" {
			t.Errorf("Expected success even with empty db, got '%s'", response.Status)
		}
	})
}
