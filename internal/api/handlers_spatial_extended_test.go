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

// Tests for spatial handlers with success path coverage

// TestSpatialViewport_MethodNotAllowed tests invalid HTTP methods
func TestSpatialViewport_MethodNotAllowed(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		cache:     cache.New(5 * time.Minute),
		startTime: time.Now(),
	}

	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/v1/spatial/viewport?west=-74&south=40&east=-73&north=41", nil)
			w := httptest.NewRecorder()

			handler.SpatialViewport(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status 405 for %s, got %d", method, w.Code)
			}
		})
	}
}

// TestSpatialViewport_DBUnavailable tests when database is nil
func TestSpatialViewport_DBUnavailable(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		db:        nil,
		cache:     cache.New(5 * time.Minute),
		startTime: time.Now(),
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/spatial/viewport?west=-74&south=40&east=-73&north=41", nil)
	w := httptest.NewRecorder()

	handler.SpatialViewport(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503 for nil db, got %d", w.Code)
	}
}

// TestSpatialViewport_Success tests successful request with valid bounding box
func TestSpatialViewport_Success(t *testing.T) {
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
		db:        db,
		cache:     cache.New(5 * time.Minute),
		startTime: time.Now(),
		config: &config.Config{
			API: config.APIConfig{
				DefaultPageSize: 20,
				MaxPageSize:     100,
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/spatial/viewport?west=-74&south=40&east=-73&north=41", nil)
	w := httptest.NewRecorder()

	handler.SpatialViewport(w, req)

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

// TestSpatialViewport_WithFilters tests viewport with additional filter parameters
func TestSpatialViewport_WithFilters(t *testing.T) {
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
		db:        db,
		cache:     cache.New(5 * time.Minute),
		startTime: time.Now(),
		config: &config.Config{
			API: config.APIConfig{
				DefaultPageSize: 20,
				MaxPageSize:     100,
			},
		},
	}

	testCases := []struct {
		name   string
		params string
	}{
		{"basic_bbox", "west=-74&south=40&east=-73&north=41"},
		{"with_users", "west=-74&south=40&east=-73&north=41&users=alice,bob"},
		{"with_media_types", "west=-74&south=40&east=-73&north=41&media_types=movie,episode"},
		{"with_platforms", "west=-74&south=40&east=-73&north=41&platforms=Chrome,Firefox"},
		{"with_date_range", "west=-74&south=40&east=-73&north=41&start_date=2025-01-01&end_date=2025-12-31"},
		{"full_params", "west=-74&south=40&east=-73&north=41&users=alice&media_types=movie&platforms=Chrome&days=30"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/spatial/viewport?"+tc.params, nil)
			w := httptest.NewRecorder()

			handler.SpatialViewport(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200 for %s, got %d", tc.name, w.Code)
			}
		})
	}
}

// TestSpatialHexagons_MethodNotAllowed tests invalid HTTP methods
func TestSpatialHexagons_MethodNotAllowed(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		cache:     cache.New(5 * time.Minute),
		startTime: time.Now(),
	}

	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/v1/spatial/hexagons", nil)
			w := httptest.NewRecorder()

			handler.SpatialHexagons(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status 405 for %s, got %d", method, w.Code)
			}
		})
	}
}

// TestSpatialHexagons_DBUnavailable tests when database is nil
func TestSpatialHexagons_DBUnavailable(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		db:        nil,
		cache:     cache.New(5 * time.Minute),
		startTime: time.Now(),
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/spatial/hexagons?resolution=7", nil)
	w := httptest.NewRecorder()

	handler.SpatialHexagons(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503 for nil db, got %d", w.Code)
	}
}

// TestSpatialHexagons_Success tests successful hexagon aggregation
func TestSpatialHexagons_Success(t *testing.T) {
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
		db:        db,
		cache:     cache.New(5 * time.Minute),
		startTime: time.Now(),
		config: &config.Config{
			API: config.APIConfig{
				DefaultPageSize: 20,
				MaxPageSize:     100,
			},
		},
	}

	testCases := []struct {
		name   string
		params string
	}{
		{"default_resolution", ""},
		{"resolution_6", "resolution=6"},
		{"resolution_7", "resolution=7"},
		{"resolution_8", "resolution=8"},
		{"with_filters", "resolution=7&users=alice&media_types=movie"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			path := "/api/v1/spatial/hexagons"
			if tc.params != "" {
				path += "?" + tc.params
			}
			req := httptest.NewRequest(http.MethodGet, path, nil)
			w := httptest.NewRecorder()

			handler.SpatialHexagons(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200 for %s, got %d", tc.name, w.Code)
			}
		})
	}
}

// TestSpatialArcs_MethodNotAllowed tests invalid HTTP methods
func TestSpatialArcs_MethodNotAllowed(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		cache:     cache.New(5 * time.Minute),
		startTime: time.Now(),
		config: &config.Config{
			Server: config.ServerConfig{
				Latitude:  40.7128,
				Longitude: -74.0060,
			},
		},
	}

	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/v1/spatial/arcs", nil)
			w := httptest.NewRecorder()

			handler.SpatialArcs(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status 405 for %s, got %d", method, w.Code)
			}
		})
	}
}

// TestSpatialArcs_ServerNotConfigured tests when server location is not configured
func TestSpatialArcs_ServerNotConfigured(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		cache:     cache.New(5 * time.Minute),
		startTime: time.Now(),
		config: &config.Config{
			Server: config.ServerConfig{
				Latitude:  0.0,
				Longitude: 0.0,
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/spatial/arcs", nil)
	w := httptest.NewRecorder()

	handler.SpatialArcs(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for unconfigured server location, got %d", w.Code)
	}

	var response models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Error == nil || response.Error.Code != "CONFIGURATION_ERROR" {
		t.Errorf("Expected CONFIGURATION_ERROR, got %v", response.Error)
	}
}

// TestSpatialArcs_DBUnavailable tests when database is nil
func TestSpatialArcs_DBUnavailable(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		db:        nil,
		cache:     cache.New(5 * time.Minute),
		startTime: time.Now(),
		config: &config.Config{
			Server: config.ServerConfig{
				Latitude:  40.7128,
				Longitude: -74.0060,
			},
			API: config.APIConfig{
				DefaultPageSize: 20,
				MaxPageSize:     100,
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/spatial/arcs", nil)
	w := httptest.NewRecorder()

	handler.SpatialArcs(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503 for nil db, got %d", w.Code)
	}
}

// TestSpatialArcs_Success tests successful arc retrieval
func TestSpatialArcs_Success(t *testing.T) {
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
		db:        db,
		cache:     cache.New(5 * time.Minute),
		startTime: time.Now(),
		config: &config.Config{
			Server: config.ServerConfig{
				Latitude:  40.7128,
				Longitude: -74.0060,
			},
			API: config.APIConfig{
				DefaultPageSize: 20,
				MaxPageSize:     100,
			},
		},
	}

	testCases := []struct {
		name   string
		params string
	}{
		{"no_params", ""},
		{"with_limit", "limit=50"},
		{"with_users", "users=alice,bob"},
		{"with_media_types", "media_types=movie,episode"},
		{"with_date_range", "start_date=2025-01-01&end_date=2025-12-31"},
		{"full_params", "users=alice&media_types=movie&platforms=Chrome&days=30&limit=100"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			path := "/api/v1/spatial/arcs"
			if tc.params != "" {
				path += "?" + tc.params
			}
			req := httptest.NewRequest(http.MethodGet, path, nil)
			w := httptest.NewRecorder()

			handler.SpatialArcs(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200 for %s, got %d", tc.name, w.Code)
			}
		})
	}
}

// TestSpatialTemporalDensity_MethodNotAllowed tests invalid HTTP methods
func TestSpatialTemporalDensity_MethodNotAllowed(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		cache:     cache.New(5 * time.Minute),
		startTime: time.Now(),
	}

	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/v1/spatial/temporal-density", nil)
			w := httptest.NewRecorder()

			handler.SpatialTemporalDensity(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status 405 for %s, got %d", method, w.Code)
			}
		})
	}
}

// TestSpatialTemporalDensity_DBUnavailable tests when database is nil
func TestSpatialTemporalDensity_DBUnavailable(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		db:        nil,
		cache:     cache.New(5 * time.Minute),
		startTime: time.Now(),
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/spatial/temporal-density", nil)
	w := httptest.NewRecorder()

	handler.SpatialTemporalDensity(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503 for nil db, got %d", w.Code)
	}
}

// TestSpatialTemporalDensity_Success tests successful temporal density query
func TestSpatialTemporalDensity_Success(t *testing.T) {
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
		db:        db,
		cache:     cache.New(5 * time.Minute),
		startTime: time.Now(),
		config: &config.Config{
			API: config.APIConfig{
				DefaultPageSize: 20,
				MaxPageSize:     100,
			},
		},
	}

	testCases := []struct {
		name   string
		params string
	}{
		{"default_params", ""},
		{"interval_hour", "interval=hour"},
		{"interval_day", "interval=day"},
		{"interval_week", "interval=week"},
		{"interval_month", "interval=month"},
		{"with_resolution", "interval=day&resolution=7"},
		{"with_filters", "interval=hour&resolution=8&users=alice&media_types=movie"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			path := "/api/v1/spatial/temporal-density"
			if tc.params != "" {
				path += "?" + tc.params
			}
			req := httptest.NewRequest(http.MethodGet, path, nil)
			w := httptest.NewRecorder()

			handler.SpatialTemporalDensity(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200 for %s, got %d", tc.name, w.Code)
			}
		})
	}
}

// TestExportPlaybacksCSV_MethodNotAllowed tests invalid HTTP methods
func TestExportPlaybacksCSV_MethodNotAllowed(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		cache:     cache.New(5 * time.Minute),
		startTime: time.Now(),
	}

	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/v1/export/playbacks.csv", nil)
			w := httptest.NewRecorder()

			handler.ExportPlaybacksCSV(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status 405 for %s, got %d", method, w.Code)
			}
		})
	}
}

// TestExportPlaybacksCSV_DBUnavailable tests when database is nil
func TestExportPlaybacksCSV_DBUnavailable(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		db:        nil,
		cache:     cache.New(5 * time.Minute),
		startTime: time.Now(),
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/export/playbacks.csv", nil)
	w := httptest.NewRecorder()

	handler.ExportPlaybacksCSV(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503 for nil db, got %d", w.Code)
	}
}

// TestExportPlaybacksCSV_Success tests successful CSV export
func TestExportPlaybacksCSV_Success(t *testing.T) {
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
		db:        db,
		cache:     cache.New(5 * time.Minute),
		startTime: time.Now(),
		config: &config.Config{
			API: config.APIConfig{
				DefaultPageSize: 20,
				MaxPageSize:     100,
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/export/playbacks.csv", nil)
	w := httptest.NewRecorder()

	handler.ExportPlaybacksCSV(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify Content-Type header
	contentType := w.Header().Get("Content-Type")
	if contentType != "text/csv" {
		t.Errorf("Expected Content-Type 'text/csv', got '%s'", contentType)
	}

	// Verify Content-Disposition header
	contentDisposition := w.Header().Get("Content-Disposition")
	if contentDisposition == "" {
		t.Error("Expected Content-Disposition header to be set")
	}
}

// TestExportPlaybacksCSV_WithFilters tests CSV export with filters
func TestExportPlaybacksCSV_WithFilters(t *testing.T) {
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
		db:        db,
		cache:     cache.New(5 * time.Minute),
		startTime: time.Now(),
		config: &config.Config{
			API: config.APIConfig{
				DefaultPageSize: 20,
				MaxPageSize:     100,
			},
		},
	}

	testCases := []struct {
		name   string
		params string
	}{
		{"with_users", "users=alice,bob"},
		{"with_media_types", "media_types=movie,episode"},
		{"with_date_range", "start_date=2025-01-01&end_date=2025-12-31"},
		{"with_limit", "limit=500"},
		{"full_params", "users=alice&media_types=movie&platforms=Chrome&days=30&limit=100"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/export/playbacks.csv?"+tc.params, nil)
			w := httptest.NewRecorder()

			handler.ExportPlaybacksCSV(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200 for %s, got %d", tc.name, w.Code)
			}
		})
	}
}

// TestExportLocationsGeoJSON_MethodNotAllowed tests invalid HTTP methods
func TestExportLocationsGeoJSON_MethodNotAllowed(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		cache:     cache.New(5 * time.Minute),
		startTime: time.Now(),
	}

	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/v1/export/locations.geojson", nil)
			w := httptest.NewRecorder()

			handler.ExportLocationsGeoJSON(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status 405 for %s, got %d", method, w.Code)
			}
		})
	}
}

// TestExportLocationsGeoJSON_DBUnavailable tests when database is nil
func TestExportLocationsGeoJSON_DBUnavailable(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		db:        nil,
		cache:     cache.New(5 * time.Minute),
		startTime: time.Now(),
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/export/locations.geojson", nil)
	w := httptest.NewRecorder()

	handler.ExportLocationsGeoJSON(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503 for nil db, got %d", w.Code)
	}
}

// TestExportLocationsGeoJSON_Success tests successful GeoJSON export
func TestExportLocationsGeoJSON_Success(t *testing.T) {
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
		db:        db,
		cache:     cache.New(5 * time.Minute),
		startTime: time.Now(),
		config: &config.Config{
			API: config.APIConfig{
				DefaultPageSize: 20,
				MaxPageSize:     100,
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/export/locations.geojson", nil)
	w := httptest.NewRecorder()

	handler.ExportLocationsGeoJSON(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify Content-Type header
	contentType := w.Header().Get("Content-Type")
	if contentType != "application/geo+json" {
		t.Errorf("Expected Content-Type 'application/geo+json', got '%s'", contentType)
	}

	// Verify Content-Disposition header
	contentDisposition := w.Header().Get("Content-Disposition")
	if contentDisposition == "" {
		t.Error("Expected Content-Disposition header to be set")
	}
}

// TestExportLocationsGeoJSON_WithFilters tests GeoJSON export with filters
func TestExportLocationsGeoJSON_WithFilters(t *testing.T) {
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
		db:        db,
		cache:     cache.New(5 * time.Minute),
		startTime: time.Now(),
		config: &config.Config{
			API: config.APIConfig{
				DefaultPageSize: 20,
				MaxPageSize:     100,
			},
		},
	}

	testCases := []struct {
		name   string
		params string
	}{
		{"with_users", "users=alice,bob"},
		{"with_media_types", "media_types=movie,episode"},
		{"with_date_range", "start_date=2025-01-01&end_date=2025-12-31"},
		{"full_params", "users=alice&media_types=movie&platforms=Chrome&days=30"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/export/locations.geojson?"+tc.params, nil)
			w := httptest.NewRecorder()

			handler.ExportLocationsGeoJSON(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200 for %s, got %d", tc.name, w.Code)
			}
		})
	}
}

// TestSpatialNearby_MethodNotAllowed tests invalid HTTP methods
func TestSpatialNearby_MethodNotAllowed(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		cache:     cache.New(5 * time.Minute),
		startTime: time.Now(),
	}

	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/v1/spatial/nearby?lat=40&lon=-74", nil)
			w := httptest.NewRecorder()

			handler.SpatialNearby(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status 405 for %s, got %d", method, w.Code)
			}
		})
	}
}

// TestSpatialNearby_DBUnavailable tests when database is nil
func TestSpatialNearby_DBUnavailable(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		db:        nil,
		cache:     cache.New(5 * time.Minute),
		startTime: time.Now(),
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/spatial/nearby?lat=40&lon=-74", nil)
	w := httptest.NewRecorder()

	handler.SpatialNearby(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503 for nil db, got %d", w.Code)
	}
}

// TestSpatialNearby_Success tests successful nearby location query
func TestSpatialNearby_Success(t *testing.T) {
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
		db:        db,
		cache:     cache.New(5 * time.Minute),
		startTime: time.Now(),
		config: &config.Config{
			API: config.APIConfig{
				DefaultPageSize: 20,
				MaxPageSize:     100,
			},
		},
	}

	testCases := []struct {
		name   string
		params string
	}{
		{"basic_query", "lat=40.7128&lon=-74.0060"},
		{"with_radius", "lat=40.7128&lon=-74.0060&radius=1000"},
		{"with_limit", "lat=40.7128&lon=-74.0060&radius=5000&limit=50"},
		{"with_filters", "lat=40.7128&lon=-74.0060&users=alice&media_types=movie"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/spatial/nearby?"+tc.params, nil)
			w := httptest.NewRecorder()

			handler.SpatialNearby(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200 for %s, got %d", tc.name, w.Code)
			}
		})
	}
}

// Benchmark tests for spatial handlers
func BenchmarkSpatialViewport(b *testing.B) {
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
		db:        db,
		cache:     cache.New(5 * time.Minute),
		startTime: time.Now(),
		config: &config.Config{
			API: config.APIConfig{
				DefaultPageSize: 20,
				MaxPageSize:     100,
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/spatial/viewport?west=-74&south=40&east=-73&north=41", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler.SpatialViewport(w, req)
	}
}

func BenchmarkSpatialHexagons(b *testing.B) {
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
		db:        db,
		cache:     cache.New(5 * time.Minute),
		startTime: time.Now(),
		config: &config.Config{
			API: config.APIConfig{
				DefaultPageSize: 20,
				MaxPageSize:     100,
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/spatial/hexagons?resolution=7", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler.SpatialHexagons(w, req)
	}
}

func BenchmarkSpatialArcs(b *testing.B) {
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
		db:        db,
		cache:     cache.New(5 * time.Minute),
		startTime: time.Now(),
		config: &config.Config{
			Server: config.ServerConfig{
				Latitude:  40.7128,
				Longitude: -74.0060,
			},
			API: config.APIConfig{
				DefaultPageSize: 20,
				MaxPageSize:     100,
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/spatial/arcs", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler.SpatialArcs(w, req)
	}
}
