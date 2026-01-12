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

// setupTestHandlerForAnalytics creates a handler with an in-memory test database for analytics testing
func setupTestHandlerForAnalytics(t *testing.T) (*Handler, *database.DB) {
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

// TestAnalyticsTrendsEnhanced tests the AnalyticsTrends handler with enhanced coverage
func TestAnalyticsTrendsEnhanced(t *testing.T) {
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, db := setupTestHandlerForAnalytics(t)
			defer db.Close()

			url := "/api/v1/analytics/trends"
			if tt.queryParams != "" {
				url += "?" + tt.queryParams
			}

			req := httptest.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()

			handler.AnalyticsTrends(w, req)

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

// TestAnalyticsGeographicEnhanced tests the AnalyticsGeographic handler with enhanced coverage
func TestAnalyticsGeographicEnhanced(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
	}{
		{
			name:           "success - default params",
			queryParams:    "",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "success - with date range",
			queryParams:    "start_date=2025-01-01T00:00:00Z&end_date=2025-01-31T23:59:59Z",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "success - with location filters",
			queryParams:    "city=New+York&country=US",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, db := setupTestHandlerForAnalytics(t)
			defer db.Close()

			url := "/api/v1/analytics/geographic"
			if tt.queryParams != "" {
				url += "?" + tt.queryParams
			}

			req := httptest.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()

			handler.AnalyticsGeographic(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
				t.Logf("Response body: %s", w.Body.String())
			}
		})
	}
}

// TestAnalyticsUsersEnhanced tests the AnalyticsUsers handler with enhanced coverage
func TestAnalyticsUsersEnhanced(t *testing.T) {
	t.Parallel()

	handler, db := setupTestHandlerForAnalytics(t)
	defer db.Close()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/users", nil)
	w := httptest.NewRecorder()

	handler.AnalyticsUsers(w, req)

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

// TestAnalyticsBingeEnhanced tests the AnalyticsBinge handler with enhanced coverage
func TestAnalyticsBingeEnhanced(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
	}{
		{
			name:           "success - default params",
			queryParams:    "",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "success - with min duration",
			queryParams:    "min_duration=3600",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "success - with date range",
			queryParams:    "start_date=2025-01-01T00:00:00Z&end_date=2025-01-31T23:59:59Z",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, db := setupTestHandlerForAnalytics(t)
			defer db.Close()

			url := "/api/v1/analytics/binge"
			if tt.queryParams != "" {
				url += "?" + tt.queryParams
			}

			req := httptest.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()

			handler.AnalyticsBinge(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

// TestAnalyticsBandwidthEnhanced tests the AnalyticsBandwidth handler with enhanced coverage
func TestAnalyticsBandwidthEnhanced(t *testing.T) {
	t.Parallel()

	handler, db := setupTestHandlerForAnalytics(t)
	defer db.Close()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/bandwidth", nil)
	w := httptest.NewRecorder()

	handler.AnalyticsBandwidth(w, req)

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

// TestAnalyticsBitrateEnhanced tests the AnalyticsBitrate handler with enhanced coverage
func TestAnalyticsBitrateEnhanced(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
	}{
		{
			name:           "success - default params",
			queryParams:    "",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "success - with date range",
			queryParams:    "start_date=2025-01-01T00:00:00Z&end_date=2025-01-31T23:59:59Z",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, db := setupTestHandlerForAnalytics(t)
			defer db.Close()

			url := "/api/v1/analytics/bitrate"
			if tt.queryParams != "" {
				url += "?" + tt.queryParams
			}

			req := httptest.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()

			handler.AnalyticsBitrate(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

// TestAnalyticsPopularEnhanced tests the AnalyticsPopular handler with enhanced coverage
func TestAnalyticsPopularEnhanced(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
	}{
		{
			name:           "success - default params",
			queryParams:    "",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "success - with limit",
			queryParams:    "limit=50",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "success - with media type filter",
			queryParams:    "media_type=movie",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, db := setupTestHandlerForAnalytics(t)
			defer db.Close()

			url := "/api/v1/analytics/popular"
			if tt.queryParams != "" {
				url += "?" + tt.queryParams
			}

			req := httptest.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()

			handler.AnalyticsPopular(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

// TestAnalyticsWatchPartiesEnhanced tests the AnalyticsWatchParties handler with enhanced coverage
func TestAnalyticsWatchPartiesEnhanced(t *testing.T) {
	t.Parallel()

	handler, db := setupTestHandlerForAnalytics(t)
	defer db.Close()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/watch-parties", nil)
	w := httptest.NewRecorder()

	handler.AnalyticsWatchParties(w, req)

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

// TestAnalyticsUserEngagementEnhanced tests the AnalyticsUserEngagement handler with enhanced coverage
func TestAnalyticsUserEngagementEnhanced(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
	}{
		{
			name:           "success - default params",
			queryParams:    "",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "success - with date range",
			queryParams:    "start_date=2025-01-01T00:00:00Z&end_date=2025-01-31T23:59:59Z",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "success - with username filter",
			queryParams:    "username=testuser",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, db := setupTestHandlerForAnalytics(t)
			defer db.Close()

			url := "/api/v1/analytics/user-engagement"
			if tt.queryParams != "" {
				url += "?" + tt.queryParams
			}

			req := httptest.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()

			handler.AnalyticsUserEngagement(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

// TestAnalyticsAbandonment tests the AnalyticsAbandonment handler
func TestAnalyticsAbandonment(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
	}{
		{
			name:           "success - default params",
			queryParams:    "",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "success - with threshold",
			queryParams:    "threshold=0.5",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "success - with date range",
			queryParams:    "start_date=2025-01-01T00:00:00Z&end_date=2025-01-31T23:59:59Z",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, db := setupTestHandlerForAnalytics(t)
			defer db.Close()

			url := "/api/v1/analytics/abandonment"
			if tt.queryParams != "" {
				url += "?" + tt.queryParams
			}

			req := httptest.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()

			handler.AnalyticsAbandonment(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

// TestAnalyticsComparative tests the AnalyticsComparative handler
func TestAnalyticsComparative(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
	}{
		{
			name:           "success - default params",
			queryParams:    "",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "success - with comparison type",
			queryParams:    "comparison_type=week",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "success - with date range",
			queryParams:    "start_date=2025-01-01T00:00:00Z&end_date=2025-01-31T23:59:59Z",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, db := setupTestHandlerForAnalytics(t)
			defer db.Close()

			url := "/api/v1/analytics/comparative"
			if tt.queryParams != "" {
				url += "?" + tt.queryParams
			}

			req := httptest.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()

			handler.AnalyticsComparative(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

// TestAnalyticsTemporalHeatmap tests the AnalyticsTemporalHeatmap handler
func TestAnalyticsTemporalHeatmap(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
	}{
		{
			name:           "success - default params",
			queryParams:    "",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "success - with date range",
			queryParams:    "start_date=2025-01-01T00:00:00Z&end_date=2025-01-31T23:59:59Z",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "success - with granularity",
			queryParams:    "granularity=hour",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, db := setupTestHandlerForAnalytics(t)
			defer db.Close()

			url := "/api/v1/analytics/temporal-heatmap"
			if tt.queryParams != "" {
				url += "?" + tt.queryParams
			}

			req := httptest.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()

			handler.AnalyticsTemporalHeatmap(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

// TestAnalyticsCaching tests caching behavior for analytics handlers
func TestAnalyticsCaching(t *testing.T) {
	t.Parallel()

	handler, db := setupTestHandlerForAnalytics(t)
	defer db.Close()

	// First request
	req1 := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/trends", nil)
	w1 := httptest.NewRecorder()
	handler.AnalyticsTrends(w1, req1)

	// Second request (should use cache)
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/trends", nil)
	w2 := httptest.NewRecorder()
	handler.AnalyticsTrends(w2, req2)

	// Both should succeed
	if w1.Code != http.StatusOK || w2.Code != http.StatusOK {
		t.Errorf("Expected both requests to succeed, got %d and %d", w1.Code, w2.Code)
	}
}

// TestAnalyticsWithContext tests context handling
func TestAnalyticsWithContext(t *testing.T) {
	t.Parallel()

	handler, db := setupTestHandlerForAnalytics(t)
	defer db.Close()

	// Create request with cancellable context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/trends", nil)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handler.AnalyticsTrends(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// TestAnalyticsMethodNotAllowed tests HTTP method validation
func TestAnalyticsMethodNotAllowed(t *testing.T) {
	t.Parallel()

	handler, db := setupTestHandlerForAnalytics(t)
	defer db.Close()

	// Test POST method (should fail)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/analytics/trends", nil)
	w := httptest.NewRecorder()

	handler.AnalyticsTrends(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}
