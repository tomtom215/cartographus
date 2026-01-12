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
	"github.com/tomtom215/cartographus/internal/config"
	"github.com/tomtom215/cartographus/internal/models"
)

// TestAnalyticsResolutionMismatch_MethodNotAllowed tests invalid HTTP methods
func TestAnalyticsResolutionMismatch_MethodNotAllowed(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		cache: cache.New(5 * time.Minute),
	}

	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/v1/analytics/resolution-mismatch", nil)
			w := httptest.NewRecorder()

			handler.AnalyticsResolutionMismatch(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status 405 for %s, got %d", method, w.Code)
			}
		})
	}
}

// TestAnalyticsHDR_MethodNotAllowed tests invalid HTTP methods
func TestAnalyticsHDR_MethodNotAllowed(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		cache: cache.New(5 * time.Minute),
	}

	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/v1/analytics/hdr", nil)
			w := httptest.NewRecorder()

			handler.AnalyticsHDR(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status 405 for %s, got %d", method, w.Code)
			}
		})
	}
}

// TestAnalyticsAudio_MethodNotAllowed tests invalid HTTP methods
func TestAnalyticsAudio_MethodNotAllowed(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		cache: cache.New(5 * time.Minute),
	}

	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/v1/analytics/audio", nil)
			w := httptest.NewRecorder()

			handler.AnalyticsAudio(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status 405 for %s, got %d", method, w.Code)
			}
		})
	}
}

// TestAnalyticsSubtitles_MethodNotAllowed tests invalid HTTP methods
func TestAnalyticsSubtitles_MethodNotAllowed(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		cache: cache.New(5 * time.Minute),
	}

	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/v1/analytics/subtitles", nil)
			w := httptest.NewRecorder()

			handler.AnalyticsSubtitles(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status 405 for %s, got %d", method, w.Code)
			}
		})
	}
}

// TestAnalyticsFrameRate_MethodNotAllowed tests invalid HTTP methods
func TestAnalyticsFrameRate_MethodNotAllowed(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		cache: cache.New(5 * time.Minute),
	}

	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/v1/analytics/frame-rate", nil)
			w := httptest.NewRecorder()

			handler.AnalyticsFrameRate(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status 405 for %s, got %d", method, w.Code)
			}
		})
	}
}

// TestAnalyticsContainer_MethodNotAllowed tests invalid HTTP methods
func TestAnalyticsContainer_MethodNotAllowed(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		cache: cache.New(5 * time.Minute),
	}

	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/v1/analytics/container", nil)
			w := httptest.NewRecorder()

			handler.AnalyticsContainer(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status 405 for %s, got %d", method, w.Code)
			}
		})
	}
}

// TestAnalyticsConnectionSecurity_MethodNotAllowed tests invalid HTTP methods
func TestAnalyticsConnectionSecurity_MethodNotAllowed(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		cache: cache.New(5 * time.Minute),
	}

	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/v1/analytics/connection-security", nil)
			w := httptest.NewRecorder()

			handler.AnalyticsConnectionSecurity(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status 405 for %s, got %d", method, w.Code)
			}
		})
	}
}

// TestAnalyticsPausePatterns_MethodNotAllowed tests invalid HTTP methods
func TestAnalyticsPausePatterns_MethodNotAllowed(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		cache: cache.New(5 * time.Minute),
	}

	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/v1/analytics/pause-patterns", nil)
			w := httptest.NewRecorder()

			handler.AnalyticsPausePatterns(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status 405 for %s, got %d", method, w.Code)
			}
		})
	}
}

// TestAnalyticsLibrary_MethodNotAllowed tests invalid HTTP methods
func TestAnalyticsLibrary_MethodNotAllowed(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		cache: cache.New(5 * time.Minute),
	}

	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/v1/analytics/library", nil)
			w := httptest.NewRecorder()

			handler.AnalyticsLibrary(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status 405 for %s, got %d", method, w.Code)
			}
		})
	}
}

// TestAnalyticsConcurrentStreams_MethodNotAllowed tests invalid HTTP methods
func TestAnalyticsConcurrentStreams_MethodNotAllowed(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		cache: cache.New(5 * time.Minute),
	}

	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/v1/analytics/concurrent-streams", nil)
			w := httptest.NewRecorder()

			handler.AnalyticsConcurrentStreams(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status 405 for %s, got %d", method, w.Code)
			}
		})
	}
}

// TestAnalyticsLibrary_InvalidLibraryID tests invalid library_id parameter
func TestAnalyticsLibrary_InvalidLibraryID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		libraryID  string
		wantStatus int
		wantErr    string
	}{
		{
			name:       "non-numeric library_id",
			libraryID:  "abc",
			wantStatus: http.StatusBadRequest,
			wantErr:    "VALIDATION_ERROR",
		},
		{
			name:       "negative library_id",
			libraryID:  "-5",
			wantStatus: http.StatusBadRequest,
			wantErr:    "VALIDATION_ERROR",
		},
		{
			name:       "float library_id",
			libraryID:  "1.5",
			wantStatus: http.StatusBadRequest,
			wantErr:    "VALIDATION_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := &Handler{
				cache: cache.New(5 * time.Minute),
			}

			url := "/api/v1/analytics/library?library_id=" + tt.libraryID
			req := httptest.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()

			handler.AnalyticsLibrary(w, req)

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

// TestAnalyticsConcurrentStreams_InvalidInterval tests invalid interval parameter
func TestAnalyticsConcurrentStreams_InvalidInterval(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		interval   string
		wantStatus int
		wantErr    string
	}{
		{
			name:       "invalid interval",
			interval:   "invalid",
			wantStatus: http.StatusBadRequest,
			wantErr:    "VALIDATION_ERROR",
		},
		{
			name:       "case sensitive - HOUR",
			interval:   "HOUR",
			wantStatus: http.StatusBadRequest,
			wantErr:    "VALIDATION_ERROR",
		},
		{
			name:       "not valid - minute",
			interval:   "minute",
			wantStatus: http.StatusBadRequest,
			wantErr:    "VALIDATION_ERROR",
		},
		{
			name:       "not valid - second",
			interval:   "second",
			wantStatus: http.StatusBadRequest,
			wantErr:    "VALIDATION_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := &Handler{
				cache: cache.New(5 * time.Minute),
			}

			url := "/api/v1/analytics/concurrent-streams?interval=" + tt.interval
			req := httptest.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()

			handler.AnalyticsConcurrentStreams(w, req)

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

// TestAnalyticsConcurrentStreams_ValidIntervals tests valid interval values
func TestAnalyticsConcurrentStreams_ValidIntervals(t *testing.T) {
	t.Parallel()

	validIntervals := []string{"hour", "day", "week", "month"}

	for _, interval := range validIntervals {
		t.Run("interval_"+interval, func(t *testing.T) {
			handler := &Handler{
				cache: cache.New(5 * time.Minute),
			}

			url := "/api/v1/analytics/concurrent-streams?interval=" + interval
			req := httptest.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()

			handler.AnalyticsConcurrentStreams(w, req)

			// Without DB, we can't get 200, but we shouldn't get 400 for valid params
			if w.Code == http.StatusBadRequest {
				var response models.APIResponse
				if err := json.NewDecoder(w.Body).Decode(&response); err == nil {
					if response.Error != nil && response.Error.Code == "VALIDATION_ERROR" {
						t.Errorf("Got VALIDATION_ERROR for valid interval '%s'", interval)
					}
				}
			}
		})
	}
}

// TestNewAnalyticsHandlers_CacheHit tests caching behavior for new analytics handlers
func TestNewAnalyticsHandlers_CacheHit(t *testing.T) {
	t.Parallel()

	// Pre-populate cache with different types
	testCases := []struct {
		name      string
		cacheKey  string
		cacheData interface{}
		handler   func(h *Handler) func(w http.ResponseWriter, r *http.Request)
		endpoint  string
	}{
		{
			name:     "resolution mismatch cache hit",
			cacheKey: "AnalyticsResolutionMismatch",
			cacheData: &models.ResolutionMismatchAnalytics{
				TotalPlaybacks:      40,
				MismatchedPlaybacks: 10,
				MismatchRate:        25.0,
			},
			handler: func(h *Handler) func(w http.ResponseWriter, r *http.Request) {
				return h.AnalyticsResolutionMismatch
			},
			endpoint: "/api/v1/analytics/resolution-mismatch",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &config.Config{
				API: config.APIConfig{
					DefaultPageSize: 100,
					MaxPageSize:     1000,
				},
			}

			// Create new cache for this test
			testCache := cache.New(5 * time.Minute)

			handler := &Handler{
				config: cfg,
				cache:  testCache,
			}

			// Request without DB should return 503 Service Unavailable
			req1 := httptest.NewRequest(http.MethodGet, tc.endpoint, nil)
			w1 := httptest.NewRecorder()

			handlerFunc := tc.handler(handler)
			handlerFunc(w1, req1)

			// Verify handler returns 503 when database is not available
			if w1.Code != http.StatusServiceUnavailable {
				t.Errorf("Expected status %d for nil db, got %d", http.StatusServiceUnavailable, w1.Code)
			}

			// Now test cache hit by pre-populating the cache with the correct key
			filter := handler.buildFilter(req1)
			cacheKey := cache.GenerateKey(tc.cacheKey, filter)
			testCache.Set(cacheKey, tc.cacheData)

			// Second request should hit cache
			req2 := httptest.NewRequest(http.MethodGet, tc.endpoint, nil)
			w2 := httptest.NewRecorder()

			handlerFunc(w2, req2)

			// Should return 200 OK from cache
			if w2.Code != http.StatusOK {
				t.Errorf("Expected status %d for cache hit, got %d", http.StatusOK, w2.Code)
			}
		})
	}
}

// TestNewAnalyticsHandlers_QueryParams tests filter parsing from query parameters
func TestNewAnalyticsHandlers_QueryParams(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		queryParam string
		handler    func(h *Handler) func(w http.ResponseWriter, r *http.Request)
		endpoint   string
	}{
		{
			name:       "resolution mismatch with date filter",
			queryParam: "start_date=2025-01-01T00:00:00Z&end_date=2025-01-31T23:59:59Z",
			handler: func(h *Handler) func(w http.ResponseWriter, r *http.Request) {
				return h.AnalyticsResolutionMismatch
			},
			endpoint: "/api/v1/analytics/resolution-mismatch",
		},
		{
			name:       "hdr with user filter",
			queryParam: "users=user1,user2",
			handler: func(h *Handler) func(w http.ResponseWriter, r *http.Request) {
				return h.AnalyticsHDR
			},
			endpoint: "/api/v1/analytics/hdr",
		},
		{
			name:       "audio with media type filter",
			queryParam: "media_types=movie,episode",
			handler: func(h *Handler) func(w http.ResponseWriter, r *http.Request) {
				return h.AnalyticsAudio
			},
			endpoint: "/api/v1/analytics/audio",
		},
		{
			name:       "subtitles with days filter",
			queryParam: "days=30",
			handler: func(h *Handler) func(w http.ResponseWriter, r *http.Request) {
				return h.AnalyticsSubtitles
			},
			endpoint: "/api/v1/analytics/subtitles",
		},
		{
			name:       "frame rate with platform filter",
			queryParam: "platforms=Android,iOS",
			handler: func(h *Handler) func(w http.ResponseWriter, r *http.Request) {
				return h.AnalyticsFrameRate
			},
			endpoint: "/api/v1/analytics/frame-rate",
		},
		{
			name:       "container with library filter",
			queryParam: "libraries=Movies",
			handler: func(h *Handler) func(w http.ResponseWriter, r *http.Request) {
				return h.AnalyticsContainer
			},
			endpoint: "/api/v1/analytics/container",
		},
		{
			name:       "connection security with player filter",
			queryParam: "players=Plex%20for%20Android",
			handler: func(h *Handler) func(w http.ResponseWriter, r *http.Request) {
				return h.AnalyticsConnectionSecurity
			},
			endpoint: "/api/v1/analytics/connection-security",
		},
		{
			name:       "pause patterns with transcode filter",
			queryParam: "transcode_decisions=transcode",
			handler: func(h *Handler) func(w http.ResponseWriter, r *http.Request) {
				return h.AnalyticsPausePatterns
			},
			endpoint: "/api/v1/analytics/pause-patterns",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
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

			url := tc.endpoint
			if tc.queryParam != "" {
				url += "?" + tc.queryParam
			}

			req := httptest.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()

			handlerFunc := tc.handler(handler)
			handlerFunc(w, req)

			// We should not get a validation error (400) for valid filter parameters
			// The error should be from missing database (500), not parameter parsing
			if w.Code == http.StatusBadRequest {
				var response models.APIResponse
				if err := json.NewDecoder(w.Body).Decode(&response); err == nil {
					if response.Error != nil && response.Error.Code == "VALIDATION_ERROR" {
						t.Errorf("Got VALIDATION_ERROR for valid query params: %s", tc.queryParam)
					}
				}
			}
		})
	}
}

// BenchmarkNewAnalyticsHandlers benchmarks method validation overhead
func BenchmarkNewAnalyticsHandlers_MethodValidation(b *testing.B) {
	handler := &Handler{
		cache: cache.New(5 * time.Minute),
	}

	endpoints := []struct {
		name    string
		handler func(w http.ResponseWriter, r *http.Request)
	}{
		{"ResolutionMismatch", handler.AnalyticsResolutionMismatch},
		{"HDR", handler.AnalyticsHDR},
		{"Audio", handler.AnalyticsAudio},
		{"Subtitles", handler.AnalyticsSubtitles},
		{"FrameRate", handler.AnalyticsFrameRate},
		{"Container", handler.AnalyticsContainer},
		{"ConnectionSecurity", handler.AnalyticsConnectionSecurity},
		{"PausePatterns", handler.AnalyticsPausePatterns},
	}

	for _, ep := range endpoints {
		b.Run(ep.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				req := httptest.NewRequest(http.MethodPost, "/test", nil)
				w := httptest.NewRecorder()
				ep.handler(w, req)
			}
		})
	}
}
