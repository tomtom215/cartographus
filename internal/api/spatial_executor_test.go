// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package api

import (
	"context"
	"errors"
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

// setupSpatialExecutorHandler creates a handler for spatial executor testing
func setupSpatialExecutorHandler(t *testing.T) *Handler {
	t.Helper()

	cfg := &config.Config{
		API: config.APIConfig{
			DefaultPageSize: 100,
			MaxPageSize:     1000,
		},
		Security: config.SecurityConfig{
			CORSOrigins: []string{"*"},
		},
	}

	// Create a DB struct and set spatial as available for testing
	db := &database.DB{}
	db.SetSpatialAvailableForTesting(true)

	return &Handler{
		db:        db,
		cache:     cache.New(5 * time.Minute),
		config:    cfg,
		startTime: time.Now(),
	}
}

// TestNewSpatialQueryExecutor tests the executor constructor
func TestNewSpatialQueryExecutor(t *testing.T) {
	t.Parallel()

	handler := setupSpatialExecutorHandler(t)
	executor := NewSpatialQueryExecutor(handler)

	if executor == nil {
		t.Fatal("NewSpatialQueryExecutor returned nil")
	}

	if executor.handler != handler {
		t.Error("Handler not set correctly")
	}
}

// TestSpatialExecuteWithCache tests the ExecuteWithCache method
func TestSpatialExecuteWithCache(t *testing.T) {
	t.Parallel()

	t.Run("successful query", func(t *testing.T) {
		handler := setupSpatialExecutorHandler(t)
		executor := NewSpatialQueryExecutor(handler)

		expectedData := []map[string]interface{}{
			{"lat": 40.7128, "lon": -74.0060},
		}

		queryFunc := func(ctx context.Context, filter database.LocationStatsFilter, params interface{}) (interface{}, error) {
			return expectedData, nil
		}

		req := httptest.NewRequest(http.MethodGet, "/api/v1/spatial/test", nil)
		w := httptest.NewRecorder()

		bbox := &BoundingBoxParams{West: -74.0, South: 40.0, East: -73.0, North: 41.0}
		executor.ExecuteWithCache(w, req, "SpatialTest", queryFunc, bbox, bbox)

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

		if response.Metadata.Cached {
			t.Error("First query should not be cached")
		}
	})

	t.Run("cached response", func(t *testing.T) {
		handler := setupSpatialExecutorHandler(t)
		executor := NewSpatialQueryExecutor(handler)

		callCount := 0
		queryFunc := func(ctx context.Context, filter database.LocationStatsFilter, params interface{}) (interface{}, error) {
			callCount++
			return map[string]interface{}{"count": callCount}, nil
		}

		bbox := &BoundingBoxParams{West: -74.0, South: 40.0, East: -73.0, North: 41.0}

		// First request
		req1 := httptest.NewRequest(http.MethodGet, "/api/v1/spatial/test", nil)
		w1 := httptest.NewRecorder()
		executor.ExecuteWithCache(w1, req1, "CacheSpatialTest", queryFunc, bbox, bbox)

		// Second request - should be cached
		req2 := httptest.NewRequest(http.MethodGet, "/api/v1/spatial/test", nil)
		w2 := httptest.NewRecorder()
		executor.ExecuteWithCache(w2, req2, "CacheSpatialTest", queryFunc, bbox, bbox)

		if callCount != 1 {
			t.Errorf("Query should only be executed once, was called %d times", callCount)
		}

		var response models.APIResponse
		json.NewDecoder(w2.Body).Decode(&response)
		if !response.Metadata.Cached {
			t.Error("Second request should be cached")
		}

		if response.Metadata.QueryTimeMS != 0 {
			t.Errorf("Cached query should have 0 query time, got %d", response.Metadata.QueryTimeMS)
		}
	})

	t.Run("query error", func(t *testing.T) {
		handler := setupSpatialExecutorHandler(t)
		executor := NewSpatialQueryExecutor(handler)

		queryFunc := func(ctx context.Context, filter database.LocationStatsFilter, params interface{}) (interface{}, error) {
			return nil, errors.New("spatial query failed")
		}

		req := httptest.NewRequest(http.MethodGet, "/api/v1/spatial/test", nil)
		w := httptest.NewRecorder()

		bbox := &BoundingBoxParams{West: -74.0, South: 40.0, East: -73.0, North: 41.0}
		executor.ExecuteWithCache(w, req, "ErrorSpatialTest", queryFunc, bbox, bbox)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 500, got %d", w.Code)
		}
	})

	t.Run("different params use different cache keys", func(t *testing.T) {
		handler := setupSpatialExecutorHandler(t)
		executor := NewSpatialQueryExecutor(handler)

		callCount := 0
		queryFunc := func(ctx context.Context, filter database.LocationStatsFilter, params interface{}) (interface{}, error) {
			callCount++
			return params, nil
		}

		bbox1 := &BoundingBoxParams{West: -74.0, South: 40.0, East: -73.0, North: 41.0}
		bbox2 := &BoundingBoxParams{West: -75.0, South: 39.0, East: -74.0, North: 40.0}

		req1 := httptest.NewRequest(http.MethodGet, "/api/v1/spatial/test", nil)
		w1 := httptest.NewRecorder()
		executor.ExecuteWithCache(w1, req1, "DiffParamsSpatial", queryFunc, bbox1, bbox1)

		req2 := httptest.NewRequest(http.MethodGet, "/api/v1/spatial/test", nil)
		w2 := httptest.NewRecorder()
		executor.ExecuteWithCache(w2, req2, "DiffParamsSpatial", queryFunc, bbox2, bbox2)

		if callCount != 2 {
			t.Errorf("Different params should trigger separate queries, got %d calls", callCount)
		}
	})
}

// TestValidateFloatParam tests the validateFloatParam helper function
func TestValidateFloatParam(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		value     string
		paramName string
		minVal    float64
		maxVal    float64
		expected  float64
		wantErr   bool
	}{
		{"valid positive", "40.7128", "lat", -90, 90, 40.7128, false},
		{"valid negative", "-74.0060", "lon", -180, 180, -74.0060, false},
		{"valid zero", "0", "lat", -90, 90, 0, false},
		{"at min boundary", "-90", "lat", -90, 90, -90, false},
		{"at max boundary", "90", "lat", -90, 90, 90, false},
		{"below min", "-91", "lat", -90, 90, 0, true},
		{"above max", "91", "lat", -90, 90, 0, true},
		{"invalid format", "abc", "lat", -90, 90, 0, true},
		{"empty value", "", "lat", -90, 90, 0, true},
		{"valid decimal", "40.123456", "lat", -90, 90, 40.123456, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := validateFloatParam(tt.value, tt.paramName, tt.minVal, tt.maxVal)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("Expected %f, got %f", tt.expected, result)
				}
			}
		})
	}
}

// TestValidateIntParam tests the validateIntParam helper function
func TestValidateIntParam(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		value     string
		paramName string
		minVal    int
		maxVal    int
		expected  int
		wantErr   bool
	}{
		{"valid positive", "7", "resolution", 6, 8, 7, false},
		{"at min boundary", "6", "resolution", 6, 8, 6, false},
		{"at max boundary", "8", "resolution", 6, 8, 8, false},
		{"below min", "5", "resolution", 6, 8, 0, true},
		{"above max", "9", "resolution", 6, 8, 0, true},
		{"invalid format", "abc", "resolution", 6, 8, 0, true},
		{"empty value", "", "resolution", 6, 8, 0, true},
		{"valid zero", "0", "offset", 0, 100, 0, false},
		{"large valid", "1000", "limit", 1, 10000, 1000, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := validateIntParam(tt.value, tt.paramName, tt.minVal, tt.maxVal)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("Expected %d, got %d", tt.expected, result)
				}
			}
		})
	}
}

// TestValidateBoundingBox tests the ValidateBoundingBox function
func TestValidateBoundingBox_Comprehensive(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		params  map[string]string
		wantErr bool
		wantBox *BoundingBoxParams
	}{
		{
			name: "valid NYC bounding box",
			params: map[string]string{
				"west": "-74.0", "south": "40.0", "east": "-73.0", "north": "41.0",
			},
			wantErr: false,
			wantBox: &BoundingBoxParams{West: -74.0, South: 40.0, East: -73.0, North: 41.0},
		},
		{
			name: "valid global bounding box",
			params: map[string]string{
				"west": "-180", "south": "-90", "east": "180", "north": "90",
			},
			wantErr: false,
			wantBox: &BoundingBoxParams{West: -180, South: -90, East: 180, North: 90},
		},
		{
			name: "valid zero point",
			params: map[string]string{
				"west": "0", "south": "0", "east": "0", "north": "0",
			},
			wantErr: false,
			wantBox: &BoundingBoxParams{West: 0, South: 0, East: 0, North: 0},
		},
		{
			name:    "missing all params",
			params:  map[string]string{},
			wantErr: true,
		},
		{
			name: "missing west",
			params: map[string]string{
				"south": "40.0", "east": "-73.0", "north": "41.0",
			},
			wantErr: true,
		},
		{
			name: "missing south",
			params: map[string]string{
				"west": "-74.0", "east": "-73.0", "north": "41.0",
			},
			wantErr: true,
		},
		{
			name: "missing east",
			params: map[string]string{
				"west": "-74.0", "south": "40.0", "north": "41.0",
			},
			wantErr: true,
		},
		{
			name: "missing north",
			params: map[string]string{
				"west": "-74.0", "south": "40.0", "east": "-73.0",
			},
			wantErr: true,
		},
		{
			name: "west out of range low",
			params: map[string]string{
				"west": "-181", "south": "40.0", "east": "-73.0", "north": "41.0",
			},
			wantErr: true,
		},
		{
			name: "west out of range high",
			params: map[string]string{
				"west": "181", "south": "40.0", "east": "-73.0", "north": "41.0",
			},
			wantErr: true,
		},
		{
			name: "south out of range",
			params: map[string]string{
				"west": "-74.0", "south": "-91", "east": "-73.0", "north": "41.0",
			},
			wantErr: true,
		},
		{
			name: "north out of range",
			params: map[string]string{
				"west": "-74.0", "south": "40.0", "east": "-73.0", "north": "91",
			},
			wantErr: true,
		},
		{
			name: "invalid west format",
			params: map[string]string{
				"west": "abc", "south": "40.0", "east": "-73.0", "north": "41.0",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/test?"
			for k, v := range tt.params {
				url += k + "=" + v + "&"
			}
			req := httptest.NewRequest(http.MethodGet, url, nil)

			box, err := ValidateBoundingBox(req)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if box == nil {
					t.Fatal("Expected box, got nil")
				}
				if box.West != tt.wantBox.West || box.South != tt.wantBox.South ||
					box.East != tt.wantBox.East || box.North != tt.wantBox.North {
					t.Errorf("Box mismatch: got %+v, want %+v", box, tt.wantBox)
				}
			}
		})
	}
}

// TestValidateCoordinates tests the ValidateCoordinates function
func TestValidateCoordinates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		params        map[string]string
		requireRadius bool
		wantErr       bool
		wantLat       float64
		wantLon       float64
		wantRadius    float64
	}{
		{
			name:          "valid coordinates without radius",
			params:        map[string]string{"lat": "40.7128", "lon": "-74.0060"},
			requireRadius: false,
			wantErr:       false,
			wantLat:       40.7128,
			wantLon:       -74.0060,
			wantRadius:    100.0, // default
		},
		{
			name:          "valid coordinates with radius",
			params:        map[string]string{"lat": "40.7128", "lon": "-74.0060", "radius": "50"},
			requireRadius: true,
			wantErr:       false,
			wantLat:       40.7128,
			wantLon:       -74.0060,
			wantRadius:    50,
		},
		{
			name:          "valid extreme coordinates",
			params:        map[string]string{"lat": "-90", "lon": "180"},
			requireRadius: false,
			wantErr:       false,
			wantLat:       -90,
			wantLon:       180,
			wantRadius:    100.0,
		},
		{
			name:          "valid large radius",
			params:        map[string]string{"lat": "0", "lon": "0", "radius": "20000"},
			requireRadius: true,
			wantErr:       false,
			wantLat:       0,
			wantLon:       0,
			wantRadius:    20000,
		},
		{
			name:          "missing lat",
			params:        map[string]string{"lon": "-74.0060"},
			requireRadius: false,
			wantErr:       true,
		},
		{
			name:          "missing lon",
			params:        map[string]string{"lat": "40.7128"},
			requireRadius: false,
			wantErr:       true,
		},
		{
			name:          "lat out of range",
			params:        map[string]string{"lat": "91", "lon": "-74.0060"},
			requireRadius: false,
			wantErr:       true,
		},
		{
			name:          "lon out of range",
			params:        map[string]string{"lat": "40.7128", "lon": "181"},
			requireRadius: false,
			wantErr:       true,
		},
		{
			name:          "invalid radius too small",
			params:        map[string]string{"lat": "40.7128", "lon": "-74.0060", "radius": "0"},
			requireRadius: true,
			wantErr:       true,
		},
		{
			name:          "invalid radius too large",
			params:        map[string]string{"lat": "40.7128", "lon": "-74.0060", "radius": "20001"},
			requireRadius: true,
			wantErr:       true,
		},
		{
			name:          "invalid lat format",
			params:        map[string]string{"lat": "abc", "lon": "-74.0060"},
			requireRadius: false,
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/test?"
			for k, v := range tt.params {
				url += k + "=" + v + "&"
			}
			req := httptest.NewRequest(http.MethodGet, url, nil)

			coords, err := ValidateCoordinates(req, tt.requireRadius)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if coords == nil {
					t.Fatal("Expected coords, got nil")
				}
				if coords.Lat != tt.wantLat {
					t.Errorf("Lat = %f, want %f", coords.Lat, tt.wantLat)
				}
				if coords.Lon != tt.wantLon {
					t.Errorf("Lon = %f, want %f", coords.Lon, tt.wantLon)
				}
				if coords.Radius != tt.wantRadius {
					t.Errorf("Radius = %f, want %f", coords.Radius, tt.wantRadius)
				}
			}
		})
	}
}

// TestValidateResolution tests the ValidateResolution function
func TestValidateResolution(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		resolution     string
		defaultVal     int
		wantErr        bool
		wantResolution int
	}{
		{"default when empty", "", 7, false, 7},
		{"valid resolution 6", "6", 7, false, 6},
		{"valid resolution 7", "7", 7, false, 7},
		{"valid resolution 8", "8", 7, false, 8},
		{"resolution too low", "5", 7, true, 0},
		{"resolution too high", "9", 7, true, 0},
		{"invalid format", "abc", 7, true, 0},
		{"negative resolution", "-1", 7, true, 0},
		{"float resolution", "7.5", 7, true, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/test?"
			if tt.resolution != "" {
				url += "resolution=" + tt.resolution
			}
			req := httptest.NewRequest(http.MethodGet, url, nil)

			resParams, err := ValidateResolution(req, tt.defaultVal)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if resParams == nil {
					t.Fatal("Expected resParams, got nil")
				}
				if resParams.Resolution != tt.wantResolution {
					t.Errorf("Resolution = %d, want %d", resParams.Resolution, tt.wantResolution)
				}
			}
		})
	}
}

// TestValidateInterval tests the ValidateInterval function
func TestValidateInterval(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		interval string
		wantErr  bool
	}{
		{"valid hour", "hour", false},
		{"valid day", "day", false},
		{"valid week", "week", false},
		{"valid month", "month", false},
		{"invalid year", "year", true},
		{"invalid minute", "minute", true},
		{"invalid empty", "", true},
		{"invalid random", "abc", true},
		{"invalid case", "HOUR", true},
		{"invalid with space", "hour ", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateInterval(tt.interval)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error for interval '%s', got nil", tt.interval)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for interval '%s': %v", tt.interval, err)
				}
			}
		})
	}
}

// BenchmarkValidateBoundingBox_Comprehensive benchmarks the ValidateBoundingBox function with various inputs
func BenchmarkValidateBoundingBox_Comprehensive(b *testing.B) {
	req := httptest.NewRequest(http.MethodGet, "/test?west=-74.0&south=40.0&east=-73.0&north=41.0", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ValidateBoundingBox(req)
	}
}

// BenchmarkValidateCoordinates benchmarks the ValidateCoordinates function
func BenchmarkValidateCoordinates(b *testing.B) {
	req := httptest.NewRequest(http.MethodGet, "/test?lat=40.7128&lon=-74.0060&radius=100", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ValidateCoordinates(req, true)
	}
}

// BenchmarkSpatialExecuteWithCache benchmarks the ExecuteWithCache method
func BenchmarkSpatialExecuteWithCache(b *testing.B) {
	db := &database.DB{}
	db.SetSpatialAvailableForTesting(true)
	handler := &Handler{
		db:    db,
		cache: cache.New(5 * time.Minute),
		config: &config.Config{
			API: config.APIConfig{
				DefaultPageSize: 100,
				MaxPageSize:     1000,
			},
		},
	}
	executor := NewSpatialQueryExecutor(handler)

	queryFunc := func(ctx context.Context, filter database.LocationStatsFilter, params interface{}) (interface{}, error) {
		return []map[string]interface{}{{"lat": 40.7, "lon": -74.0}}, nil
	}

	bbox := &BoundingBoxParams{West: -74.0, South: 40.0, East: -73.0, North: 41.0}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/spatial/test", nil)
		w := httptest.NewRecorder()
		executor.ExecuteWithCache(w, req, "BenchSpatial", queryFunc, bbox, bbox)
	}
}

// BenchmarkSpatialExecuteWithCache_CacheHit benchmarks cache hit performance
func BenchmarkSpatialExecuteWithCache_CacheHit(b *testing.B) {
	db := &database.DB{}
	db.SetSpatialAvailableForTesting(true)
	handler := &Handler{
		db:    db,
		cache: cache.New(5 * time.Minute),
		config: &config.Config{
			API: config.APIConfig{
				DefaultPageSize: 100,
				MaxPageSize:     1000,
			},
		},
	}
	executor := NewSpatialQueryExecutor(handler)

	queryFunc := func(ctx context.Context, filter database.LocationStatsFilter, params interface{}) (interface{}, error) {
		return []map[string]interface{}{{"lat": 40.7, "lon": -74.0}}, nil
	}

	bbox := &BoundingBoxParams{West: -74.0, South: 40.0, East: -73.0, North: 41.0}

	// Prime the cache
	req := httptest.NewRequest(http.MethodGet, "/api/v1/spatial/test", nil)
	w := httptest.NewRecorder()
	executor.ExecuteWithCache(w, req, "CacheHitSpatialBench", queryFunc, bbox, bbox)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/spatial/test", nil)
		w := httptest.NewRecorder()
		executor.ExecuteWithCache(w, req, "CacheHitSpatialBench", queryFunc, bbox, bbox)
	}
}
