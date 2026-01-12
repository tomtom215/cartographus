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
	"github.com/tomtom215/cartographus/internal/database"
	"github.com/tomtom215/cartographus/internal/models"
)

// TestAnalyticsHardwareTranscode_MethodNotAllowed tests invalid HTTP methods
func TestAnalyticsHardwareTranscode_MethodNotAllowed(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		cache: cache.New(5 * time.Minute),
	}

	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/v1/analytics/hardware-transcode", nil)
			w := httptest.NewRecorder()

			handler.AnalyticsHardwareTranscode(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status 405 for %s, got %d", method, w.Code)
			}
		})
	}
}

// TestAnalyticsHardwareTranscode_DBUnavailable tests when database is nil
func TestAnalyticsHardwareTranscode_DBUnavailable(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		db:    nil,
		cache: cache.New(5 * time.Minute),
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/hardware-transcode", nil)
	w := httptest.NewRecorder()

	handler.AnalyticsHardwareTranscode(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503 for nil db, got %d", w.Code)
	}

	var response models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Error == nil || response.Error.Code != "SERVICE_ERROR" {
		t.Errorf("Expected SERVICE_ERROR, got %v", response.Error)
	}
}

// TestAnalyticsHardwareTranscode_Success tests successful request
func TestAnalyticsHardwareTranscode_Success(t *testing.T) {
	t.Parallel()

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
	defer db.Close()

	handler := &Handler{
		db:    db,
		cache: cache.New(5 * time.Minute),
		config: &config.Config{
			API: config.APIConfig{
				DefaultPageSize: 20,
				MaxPageSize:     100,
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/hardware-transcode", nil)
	w := httptest.NewRecorder()

	handler.AnalyticsHardwareTranscode(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var response models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Status != "success" {
		t.Errorf("Expected status 'success', got '%s'", response.Status)
	}
}

// TestAnalyticsHardwareTranscode_WithFilters tests with various filter parameters
func TestAnalyticsHardwareTranscode_WithFilters(t *testing.T) {
	t.Parallel()

	cfg := &config.DatabaseConfig{
		Path:        ":memory:",
		MaxMemory:   "512MB",
		SkipIndexes: true, // Skip 97 indexes for fast test setup
	}

	db, err := database.New(cfg, 0.0, 0.0)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	handler := &Handler{
		db:    db,
		cache: cache.New(5 * time.Minute),
		config: &config.Config{
			API: config.APIConfig{
				DefaultPageSize: 20,
				MaxPageSize:     100,
			},
		},
	}

	tests := []struct {
		name        string
		queryParams string
	}{
		{"no_filters", ""},
		{"with_date_range", "start_date=2025-01-01T00:00:00Z&end_date=2025-01-31T23:59:59Z"},
		{"with_users", "users=user1,user2"},
		{"with_media_types", "media_types=movie,episode"},
		{"with_platforms", "platforms=Roku,Android"},
		{"with_days", "days=30"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/api/v1/analytics/hardware-transcode"
			if tt.queryParams != "" {
				url += "?" + tt.queryParams
			}

			req := httptest.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()

			handler.AnalyticsHardwareTranscode(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200 for %s, got %d. Body: %s", tt.name, w.Code, w.Body.String())
			}
		})
	}
}

// TestAnalyticsHDRContent_MethodNotAllowed tests invalid HTTP methods
func TestAnalyticsHDRContent_MethodNotAllowed(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		cache: cache.New(5 * time.Minute),
	}

	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/v1/analytics/hdr-content", nil)
			w := httptest.NewRecorder()

			handler.AnalyticsHDRContent(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status 405 for %s, got %d", method, w.Code)
			}
		})
	}
}

// TestAnalyticsHDRContent_DBUnavailable tests when database is nil
func TestAnalyticsHDRContent_DBUnavailable(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		db:    nil,
		cache: cache.New(5 * time.Minute),
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/hdr-content", nil)
	w := httptest.NewRecorder()

	handler.AnalyticsHDRContent(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503 for nil db, got %d", w.Code)
	}
}

// TestAnalyticsHDRContent_Success tests successful request
func TestAnalyticsHDRContent_Success(t *testing.T) {
	t.Parallel()

	cfg := &config.DatabaseConfig{
		Path:        ":memory:",
		MaxMemory:   "512MB",
		SkipIndexes: true, // Skip 97 indexes for fast test setup
	}

	db, err := database.New(cfg, 0.0, 0.0)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	handler := &Handler{
		db:    db,
		cache: cache.New(5 * time.Minute),
		config: &config.Config{
			API: config.APIConfig{
				DefaultPageSize: 20,
				MaxPageSize:     100,
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/hdr-content", nil)
	w := httptest.NewRecorder()

	handler.AnalyticsHDRContent(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var response models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Status != "success" {
		t.Errorf("Expected status 'success', got '%s'", response.Status)
	}
}

// TestAnalyticsHDRContent_WithFilters tests with various filter parameters
func TestAnalyticsHDRContent_WithFilters(t *testing.T) {
	t.Parallel()

	cfg := &config.DatabaseConfig{
		Path:        ":memory:",
		MaxMemory:   "512MB",
		SkipIndexes: true, // Skip 97 indexes for fast test setup
	}

	db, err := database.New(cfg, 0.0, 0.0)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	handler := &Handler{
		db:    db,
		cache: cache.New(5 * time.Minute),
		config: &config.Config{
			API: config.APIConfig{
				DefaultPageSize: 20,
				MaxPageSize:     100,
			},
		},
	}

	tests := []struct {
		name        string
		queryParams string
	}{
		{"no_filters", ""},
		{"with_date_range", "start_date=2025-01-01T00:00:00Z&end_date=2025-01-31T23:59:59Z"},
		{"with_users", "users=user1,user2"},
		{"with_media_types", "media_types=movie"},
		{"with_days", "days=7"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/api/v1/analytics/hdr-content"
			if tt.queryParams != "" {
				url += "?" + tt.queryParams
			}

			req := httptest.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()

			handler.AnalyticsHDRContent(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200 for %s, got %d", tt.name, w.Code)
			}
		})
	}
}

// TestAnalyticsHardwareTranscodeTrends_MethodNotAllowed tests invalid HTTP methods
func TestAnalyticsHardwareTranscodeTrends_MethodNotAllowed(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		cache: cache.New(5 * time.Minute),
	}

	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/v1/analytics/hardware-transcode/trends", nil)
			w := httptest.NewRecorder()

			handler.AnalyticsHardwareTranscodeTrends(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status 405 for %s, got %d", method, w.Code)
			}
		})
	}
}

// TestAnalyticsHardwareTranscodeTrends_DBUnavailable tests when database is nil
func TestAnalyticsHardwareTranscodeTrends_DBUnavailable(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		db:    nil,
		cache: cache.New(5 * time.Minute),
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/hardware-transcode/trends", nil)
	w := httptest.NewRecorder()

	handler.AnalyticsHardwareTranscodeTrends(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503 for nil db, got %d", w.Code)
	}
}

// TestAnalyticsHardwareTranscodeTrends_Success tests successful request
func TestAnalyticsHardwareTranscodeTrends_Success(t *testing.T) {
	t.Parallel()

	cfg := &config.DatabaseConfig{
		Path:        ":memory:",
		MaxMemory:   "512MB",
		SkipIndexes: true, // Skip 97 indexes for fast test setup
	}

	db, err := database.New(cfg, 0.0, 0.0)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	handler := &Handler{
		db:    db,
		cache: cache.New(5 * time.Minute),
		config: &config.Config{
			API: config.APIConfig{
				DefaultPageSize: 20,
				MaxPageSize:     100,
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/hardware-transcode/trends", nil)
	w := httptest.NewRecorder()

	handler.AnalyticsHardwareTranscodeTrends(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var response models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Status != "success" {
		t.Errorf("Expected status 'success', got '%s'", response.Status)
	}

	// Data should be an array (even if empty)
	if response.Data == nil {
		t.Error("Expected data field to be present")
	}
}

// TestAnalyticsHardwareTranscodeTrends_WithFilters tests with various filter parameters
func TestAnalyticsHardwareTranscodeTrends_WithFilters(t *testing.T) {
	t.Parallel()

	cfg := &config.DatabaseConfig{
		Path:        ":memory:",
		MaxMemory:   "512MB",
		SkipIndexes: true, // Skip 97 indexes for fast test setup
	}

	db, err := database.New(cfg, 0.0, 0.0)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	handler := &Handler{
		db:    db,
		cache: cache.New(5 * time.Minute),
		config: &config.Config{
			API: config.APIConfig{
				DefaultPageSize: 20,
				MaxPageSize:     100,
			},
		},
	}

	tests := []struct {
		name        string
		queryParams string
	}{
		{"no_filters", ""},
		{"with_date_range", "start_date=2025-01-01T00:00:00Z&end_date=2025-01-31T23:59:59Z"},
		{"with_users", "users=user1,user2,user3"},
		{"with_platforms", "platforms=Roku,Plex%20Web"},
		{"with_days", "days=14"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/api/v1/analytics/hardware-transcode/trends"
			if tt.queryParams != "" {
				url += "?" + tt.queryParams
			}

			req := httptest.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()

			handler.AnalyticsHardwareTranscodeTrends(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200 for %s, got %d", tt.name, w.Code)
			}
		})
	}
}

// TestAnalyticsHardware_CacheHit tests cache functionality
func TestAnalyticsHardware_CacheHit(t *testing.T) {
	t.Parallel()

	cfg := &config.DatabaseConfig{
		Path:        ":memory:",
		MaxMemory:   "512MB",
		SkipIndexes: true, // Skip 97 indexes for fast test setup
	}

	db, err := database.New(cfg, 0.0, 0.0)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	c := cache.New(5 * time.Minute)
	handler := &Handler{
		db:    db,
		cache: c,
		config: &config.Config{
			API: config.APIConfig{
				DefaultPageSize: 20,
				MaxPageSize:     100,
			},
		},
	}

	// First request (cache miss)
	req1 := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/hardware-transcode", nil)
	w1 := httptest.NewRecorder()
	handler.AnalyticsHardwareTranscode(w1, req1)

	if w1.Code != http.StatusOK {
		t.Fatalf("First request failed: status %d", w1.Code)
	}

	// Second request (cache hit)
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/hardware-transcode", nil)
	w2 := httptest.NewRecorder()
	handler.AnalyticsHardwareTranscode(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("Second request (cache hit) failed: status %d", w2.Code)
	}
}

// Benchmark tests
func BenchmarkAnalyticsHardwareTranscode(b *testing.B) {
	cfg := &config.DatabaseConfig{
		Path:        ":memory:",
		MaxMemory:   "512MB",
		SkipIndexes: true, // Skip 97 indexes for fast test setup
	}

	db, err := database.New(cfg, 0.0, 0.0)
	if err != nil {
		b.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	handler := &Handler{
		db:    db,
		cache: cache.New(5 * time.Minute),
		config: &config.Config{
			API: config.APIConfig{
				DefaultPageSize: 20,
				MaxPageSize:     100,
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/hardware-transcode", nil)
		w := httptest.NewRecorder()
		handler.AnalyticsHardwareTranscode(w, req)
	}
}

func BenchmarkAnalyticsHDRContent(b *testing.B) {
	cfg := &config.DatabaseConfig{
		Path:        ":memory:",
		MaxMemory:   "512MB",
		SkipIndexes: true, // Skip 97 indexes for fast test setup
	}

	db, err := database.New(cfg, 0.0, 0.0)
	if err != nil {
		b.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	handler := &Handler{
		db:    db,
		cache: cache.New(5 * time.Minute),
		config: &config.Config{
			API: config.APIConfig{
				DefaultPageSize: 20,
				MaxPageSize:     100,
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/hdr-content", nil)
		w := httptest.NewRecorder()
		handler.AnalyticsHDRContent(w, req)
	}
}

func BenchmarkAnalyticsHardwareTranscodeTrends(b *testing.B) {
	cfg := &config.DatabaseConfig{
		Path:        ":memory:",
		MaxMemory:   "512MB",
		SkipIndexes: true, // Skip 97 indexes for fast test setup
	}

	db, err := database.New(cfg, 0.0, 0.0)
	if err != nil {
		b.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	handler := &Handler{
		db:    db,
		cache: cache.New(5 * time.Minute),
		config: &config.Config{
			API: config.APIConfig{
				DefaultPageSize: 20,
				MaxPageSize:     100,
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/hardware-transcode/trends", nil)
		w := httptest.NewRecorder()
		handler.AnalyticsHardwareTranscodeTrends(w, req)
	}
}
