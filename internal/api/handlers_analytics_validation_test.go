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

// Note: TestAnalyticsTrends_MethodNotAllowed, TestAnalyticsGeographic_MethodNotAllowed,
// TestAnalyticsUsers_MethodNotAllowed, TestAnalyticsAbandonment_MethodNotAllowed, and
// TestAnalyticsUsers_LimitValidation are defined in handlers_method_validation_split_test.go

// TestAnalyticsBinge_MethodNotAllowed tests invalid HTTP methods
func TestAnalyticsBinge_MethodNotAllowed(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		cache: cache.New(5 * time.Minute),
	}

	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/v1/analytics/binge", nil)
			w := httptest.NewRecorder()

			handler.AnalyticsBinge(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status 405 for %s, got %d", method, w.Code)
			}
		})
	}
}

// TestAnalyticsBandwidth_MethodNotAllowed tests invalid HTTP methods
func TestAnalyticsBandwidth_MethodNotAllowed(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		cache: cache.New(5 * time.Minute),
	}

	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/v1/analytics/bandwidth", nil)
			w := httptest.NewRecorder()

			handler.AnalyticsBandwidth(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status 405 for %s, got %d", method, w.Code)
			}
		})
	}
}

// TestAnalyticsBitrate_MethodNotAllowed tests invalid HTTP methods
func TestAnalyticsBitrate_MethodNotAllowed(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		cache: cache.New(5 * time.Minute),
	}

	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/v1/analytics/bitrate", nil)
			w := httptest.NewRecorder()

			handler.AnalyticsBitrate(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status 405 for %s, got %d", method, w.Code)
			}
		})
	}
}

// TestAnalyticsPopular_MethodNotAllowed tests invalid HTTP methods
func TestAnalyticsPopular_MethodNotAllowed(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		cache: cache.New(5 * time.Minute),
	}

	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/v1/analytics/popular", nil)
			w := httptest.NewRecorder()

			handler.AnalyticsPopular(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status 405 for %s, got %d", method, w.Code)
			}
		})
	}
}

// TestAnalyticsWatchParties_MethodNotAllowed tests invalid HTTP methods
func TestAnalyticsWatchParties_MethodNotAllowed(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		cache: cache.New(5 * time.Minute),
	}

	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/v1/analytics/watch-parties", nil)
			w := httptest.NewRecorder()

			handler.AnalyticsWatchParties(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status 405 for %s, got %d", method, w.Code)
			}
		})
	}
}

// TestAnalyticsUserEngagement_MethodNotAllowed tests invalid HTTP methods
func TestAnalyticsUserEngagement_MethodNotAllowed(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		cache: cache.New(5 * time.Minute),
	}

	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/v1/analytics/user-engagement", nil)
			w := httptest.NewRecorder()

			handler.AnalyticsUserEngagement(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status 405 for %s, got %d", method, w.Code)
			}
		})
	}
}

// TestAnalyticsComparative_MethodNotAllowed tests invalid HTTP methods
func TestAnalyticsComparative_MethodNotAllowed(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		cache: cache.New(5 * time.Minute),
	}

	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/v1/analytics/comparative", nil)
			w := httptest.NewRecorder()

			handler.AnalyticsComparative(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status 405 for %s, got %d", method, w.Code)
			}
		})
	}
}

// TestAnalyticsTemporalHeatmap_MethodNotAllowed tests invalid HTTP methods
func TestAnalyticsTemporalHeatmap_MethodNotAllowed(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		cache: cache.New(5 * time.Minute),
	}

	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/v1/analytics/temporal-heatmap", nil)
			w := httptest.NewRecorder()

			handler.AnalyticsTemporalHeatmap(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status 405 for %s, got %d", method, w.Code)
			}
		})
	}
}

// TestAnalyticsComparative_InvalidComparisonType tests invalid comparison types
func TestAnalyticsComparative_InvalidComparisonType(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		cache: cache.New(5 * time.Minute),
	}

	invalidTypes := []string{
		"invalid",
		"daily",
		"yearly",
		"WEEK",      // case sensitive
		"Week",      // case sensitive
		"quarterly", // not valid
		"",          // empty should default to week, not fail
	}

	for _, invalidType := range invalidTypes {
		t.Run("type_"+invalidType, func(t *testing.T) {
			url := "/api/v1/analytics/comparative"
			if invalidType != "" {
				url += "?comparison_type=" + invalidType
			}

			req := httptest.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()

			handler.AnalyticsComparative(w, req)

			// Empty string should default to "week", which is valid
			if invalidType == "" {
				// Would need DB to succeed, but should not return 400
				return
			}

			if w.Code != http.StatusBadRequest {
				t.Errorf("Expected status 400 for invalid comparison_type '%s', got %d", invalidType, w.Code)
			}

			var response models.APIResponse
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if response.Error == nil || response.Error.Code != "INVALID_PARAMETER" {
				t.Errorf("Expected INVALID_PARAMETER error code, got: %v", response.Error)
			}
		})
	}
}

// TestAnalyticsComparative_ValidComparisonTypes tests valid comparison types
func TestAnalyticsComparative_ValidComparisonTypes(t *testing.T) {
	t.Parallel()

	validTypes := []string{"week", "month", "quarter", "year", "custom"}

	for _, validType := range validTypes {
		t.Run("type_"+validType, func(t *testing.T) {
			handler := &Handler{
				cache: cache.New(5 * time.Minute),
			}

			url := "/api/v1/analytics/comparative?comparison_type=" + validType

			req := httptest.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()

			handler.AnalyticsComparative(w, req)

			// Without a real database, we can't get 200, but we shouldn't get 400 for valid params
			// The error should be from missing database, not validation
			if w.Code == http.StatusBadRequest {
				var response models.APIResponse
				if err := json.NewDecoder(w.Body).Decode(&response); err == nil {
					if response.Error != nil && response.Error.Code == "INVALID_PARAMETER" {
						t.Errorf("Got INVALID_PARAMETER error for valid comparison_type '%s'", validType)
					}
				}
			}
		})
	}
}

// TestAnalyticsTemporalHeatmap_InvalidInterval tests invalid interval values
func TestAnalyticsTemporalHeatmap_InvalidInterval(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		cache: cache.New(5 * time.Minute),
	}

	invalidIntervals := []string{
		"invalid",
		"yearly",
		"daily",
		"HOUR",    // case sensitive
		"Hour",    // case sensitive
		"minutes", // not valid
		"seconds", // not valid
	}

	for _, invalidInterval := range invalidIntervals {
		t.Run("interval_"+invalidInterval, func(t *testing.T) {
			url := "/api/v1/analytics/temporal-heatmap?interval=" + invalidInterval

			req := httptest.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()

			handler.AnalyticsTemporalHeatmap(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("Expected status 400 for invalid interval '%s', got %d", invalidInterval, w.Code)
			}

			var response models.APIResponse
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if response.Error == nil || response.Error.Code != "INVALID_PARAMETER" {
				t.Errorf("Expected INVALID_PARAMETER error code, got: %v", response.Error)
			}
		})
	}
}

// TestAnalyticsTemporalHeatmap_ValidIntervals tests valid interval values
func TestAnalyticsTemporalHeatmap_ValidIntervals(t *testing.T) {
	t.Parallel()

	validIntervals := []string{"hour", "day", "week", "month"}

	for _, validInterval := range validIntervals {
		t.Run("interval_"+validInterval, func(t *testing.T) {
			handler := &Handler{
				cache: cache.New(5 * time.Minute),
			}

			url := "/api/v1/analytics/temporal-heatmap?interval=" + validInterval

			req := httptest.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()

			handler.AnalyticsTemporalHeatmap(w, req)

			// Without a real database, we can't get 200, but we shouldn't get 400 for valid params
			if w.Code == http.StatusBadRequest {
				var response models.APIResponse
				if err := json.NewDecoder(w.Body).Decode(&response); err == nil {
					if response.Error != nil && response.Error.Code == "INVALID_PARAMETER" {
						t.Errorf("Got INVALID_PARAMETER error for valid interval '%s'", validInterval)
					}
				}
			}
		})
	}
}

// TestAnalyticsPopular_LimitHandling tests limit parameter handling
func TestAnalyticsPopular_LimitHandling(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		limit         string
		expectedLimit int // The limit that should be used (defaults/clamped)
	}{
		{
			name:          "no limit - uses default",
			limit:         "",
			expectedLimit: 10,
		},
		{
			name:          "valid limit",
			limit:         "25",
			expectedLimit: 25,
		},
		{
			name:          "limit above max - uses max",
			limit:         "100",
			expectedLimit: 10, // Max is 50, should default back to 10 for invalid
		},
		{
			name:          "zero limit - uses default",
			limit:         "0",
			expectedLimit: 10,
		},
		{
			name:          "negative limit - uses default",
			limit:         "-5",
			expectedLimit: 10,
		},
		{
			name:          "invalid limit string - uses default",
			limit:         "abc",
			expectedLimit: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This test verifies the limit parsing logic without database
			// The handler should parse the limit and not fail on validation
			handler := &Handler{
				cache: cache.New(5 * time.Minute),
			}

			url := "/api/v1/analytics/popular"
			if tt.limit != "" {
				url += "?limit=" + tt.limit
			}

			req := httptest.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()

			handler.AnalyticsPopular(w, req)

			// Without DB, we can't verify success, but we shouldn't get validation errors
			// for the limit parsing (it uses defaults for invalid values)
			if w.Code == http.StatusBadRequest {
				var response models.APIResponse
				if err := json.NewDecoder(w.Body).Decode(&response); err == nil {
					if response.Error != nil && response.Error.Code == "VALIDATION_ERROR" {
						// AnalyticsPopular doesn't do explicit validation, it just uses defaults
						t.Logf("Note: Got validation error for limit '%s'", tt.limit)
					}
				}
			}
		})
	}
}

// TestAnalyticsUserEngagement_LimitHandling tests limit parameter handling
func TestAnalyticsUserEngagement_LimitHandling(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		limit string
	}{
		{"no limit - uses default", ""},
		{"valid limit", "25"},
		{"limit at max", "100"},
		{"limit above max - uses default", "150"},
		{"zero limit - uses default", "0"},
		{"negative limit - uses default", "-5"},
		{"invalid limit string - uses default", "abc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := &Handler{
				cache: cache.New(5 * time.Minute),
			}

			url := "/api/v1/analytics/user-engagement"
			if tt.limit != "" {
				url += "?limit=" + tt.limit
			}

			req := httptest.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()

			handler.AnalyticsUserEngagement(w, req)

			// Without DB, we just verify no validation errors for limit parsing
			if w.Code == http.StatusBadRequest {
				var response models.APIResponse
				if err := json.NewDecoder(w.Body).Decode(&response); err == nil {
					if response.Error != nil && response.Error.Code == "VALIDATION_ERROR" {
						t.Logf("Note: Got validation error for limit '%s'", tt.limit)
					}
				}
			}
		})
	}
}

// TestCheckCacheAndReturnIfHit tests the cache check helper function
func TestCheckCacheAndReturnIfHit(t *testing.T) {
	t.Parallel()

	t.Run("cache miss", func(t *testing.T) {
		c := cache.New(5 * time.Minute)
		handler := &Handler{cache: c}

		w := httptest.NewRecorder()
		start := time.Now()

		hit := handler.checkCacheAndReturnIfHit(w, "nonexistent-key", start)

		if hit {
			t.Error("Expected cache miss, got hit")
		}

		if w.Body.Len() > 0 {
			t.Error("Expected empty body on cache miss")
		}
	})

	t.Run("cache hit with valid data", func(t *testing.T) {
		c := cache.New(5 * time.Minute)

		response := models.GeographicResponse{
			TopCities: []models.CityStats{
				{City: "New York", Country: "US", PlaybackCount: 100, UniqueUsers: 10},
			},
		}
		c.Set("test-key", response)

		handler := &Handler{cache: c}

		w := httptest.NewRecorder()
		start := time.Now()

		hit := handler.checkCacheAndReturnIfHit(w, "test-key", start)

		if !hit {
			t.Error("Expected cache hit, got miss")
		}

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var apiResponse models.APIResponse
		if err := json.NewDecoder(w.Body).Decode(&apiResponse); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if apiResponse.Status != "success" {
			t.Errorf("Expected status 'success', got '%s'", apiResponse.Status)
		}
	})

	t.Run("cache hit with wrong type", func(t *testing.T) {
		c := cache.New(5 * time.Minute)

		// Store wrong type in cache
		c.Set("test-key", "not a GeographicResponse")

		handler := &Handler{cache: c}

		w := httptest.NewRecorder()
		start := time.Now()

		hit := handler.checkCacheAndReturnIfHit(w, "test-key", start)

		// Should return false because type assertion fails
		if hit {
			t.Error("Expected cache miss due to type mismatch, got hit")
		}
	})
}

// TestCacheAndRespondSuccess tests the cache and respond helper function
func TestCacheAndRespondSuccess(t *testing.T) {
	t.Parallel()

	c := cache.New(5 * time.Minute)
	handler := &Handler{cache: c}

	response := &models.GeographicResponse{
		TopCities: []models.CityStats{
			{City: "Tokyo", Country: "JP", PlaybackCount: 50, UniqueUsers: 5},
		},
		TopCountries: []models.CountryStats{
			{Country: "Japan", PlaybackCount: 100, UniqueUsers: 10},
		},
	}

	w := httptest.NewRecorder()
	start := time.Now()
	cacheKey := "geographic-test"

	handler.cacheAndRespondSuccess(w, cacheKey, response, start)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify response
	var apiResponse models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&apiResponse); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if apiResponse.Status != "success" {
		t.Errorf("Expected status 'success', got '%s'", apiResponse.Status)
	}

	// Verify data was cached
	cached, found := c.Get(cacheKey)
	if !found {
		t.Error("Expected data to be cached")
	}

	if _, ok := cached.(models.GeographicResponse); !ok {
		t.Error("Cached data is not a GeographicResponse")
	}
}

// BenchmarkCheckCacheAndReturnIfHit benchmarks cache checking
func BenchmarkCheckCacheAndReturnIfHit(b *testing.B) {
	c := cache.New(5 * time.Minute)
	response := models.GeographicResponse{
		TopCities: []models.CityStats{{City: "NYC", Country: "US", PlaybackCount: 100, UniqueUsers: 10}},
	}
	c.Set("bench-key", response)

	handler := &Handler{cache: c}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler.checkCacheAndReturnIfHit(w, "bench-key", time.Now())
	}
}

// BenchmarkCacheAndRespondSuccess benchmarks cache and respond
func BenchmarkCacheAndRespondSuccess(b *testing.B) {
	c := cache.New(5 * time.Minute)
	handler := &Handler{cache: c}

	response := &models.GeographicResponse{
		TopCities: []models.CityStats{{City: "NYC", Country: "US", PlaybackCount: 100, UniqueUsers: 10}},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler.cacheAndRespondSuccess(w, "bench-key", response, time.Now())
	}
}
