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

// TestAnalyticsExecutor_NilDatabase tests ExecuteSimple with nil database
func TestAnalyticsExecutor_NilDatabase(t *testing.T) {
	t.Parallel()

	// Create handler with nil database
	handler := &Handler{
		db:        nil, // Nil database
		cache:     cache.New(5 * time.Minute),
		config:    &config.Config{},
		startTime: time.Now(),
	}

	executor := NewAnalyticsQueryExecutor(handler)

	queryFunc := func(ctx context.Context, filter database.LocationStatsFilter) (interface{}, error) {
		t.Error("Query function should not be called when database is nil")
		return nil, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/test", nil)
	w := httptest.NewRecorder()

	executor.ExecuteSimple(w, req, "NilDBTest", queryFunc)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503 (Service Unavailable), got %d", w.Code)
	}

	var response models.APIResponse
	json.NewDecoder(w.Body).Decode(&response)

	if response.Status != "error" {
		t.Errorf("Expected status 'error', got '%s'", response.Status)
	}

	if response.Error == nil || response.Error.Code != "SERVICE_ERROR" {
		t.Error("Expected SERVICE_ERROR error code")
	}
}

// TestAnalyticsExecutor_NilDatabaseWithParam tests ExecuteWithParam with nil database
func TestAnalyticsExecutor_NilDatabaseWithParam(t *testing.T) {
	t.Parallel()

	// Create handler with nil database
	handler := &Handler{
		db:        nil, // Nil database
		cache:     cache.New(5 * time.Minute),
		config:    &config.Config{},
		startTime: time.Now(),
	}

	executor := NewAnalyticsQueryExecutor(handler)

	queryFunc := func(ctx context.Context, filter database.LocationStatsFilter, param interface{}) (interface{}, error) {
		t.Error("Query function should not be called when database is nil")
		return nil, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/test", nil)
	w := httptest.NewRecorder()

	executor.ExecuteWithParam(w, req, "NilDBWithParamTest", queryFunc, 10)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503 (Service Unavailable), got %d", w.Code)
	}
}

// TestAnalyticsExecutor_NilCache tests ExecuteSimple with nil cache
func TestAnalyticsExecutor_NilCache(t *testing.T) {
	t.Parallel()

	// Create handler with nil cache but valid DB
	handler := &Handler{
		db:        &database.DB{}, // Non-nil DB
		cache:     nil,            // Nil cache - should still work
		config:    &config.Config{},
		startTime: time.Now(),
	}

	executor := NewAnalyticsQueryExecutor(handler)

	callCount := 0
	queryFunc := func(ctx context.Context, filter database.LocationStatsFilter) (interface{}, error) {
		callCount++
		return map[string]interface{}{"call": callCount}, nil
	}

	// First request
	req1 := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/test", nil)
	w1 := httptest.NewRecorder()
	executor.ExecuteSimple(w1, req1, "NilCacheTest", queryFunc)

	if w1.Code != http.StatusOK {
		t.Errorf("First request failed: %d", w1.Code)
	}

	// Second request - should execute query again since cache is nil
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/test", nil)
	w2 := httptest.NewRecorder()
	executor.ExecuteSimple(w2, req2, "NilCacheTest", queryFunc)

	if w2.Code != http.StatusOK {
		t.Errorf("Second request failed: %d", w2.Code)
	}

	// Without cache, query should be executed twice
	if callCount != 2 {
		t.Errorf("Without cache, query should be executed twice, but was called %d times", callCount)
	}

	// Verify response doesn't indicate caching
	var response models.APIResponse
	json.NewDecoder(w2.Body).Decode(&response)
	if response.Metadata.Cached {
		t.Error("Response should not be marked as cached when cache is nil")
	}
}

// TestAnalyticsExecutor_NilCacheWithParam tests ExecuteWithParam with nil cache
func TestAnalyticsExecutor_NilCacheWithParam(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		db:        &database.DB{},
		cache:     nil, // Nil cache
		config:    &config.Config{},
		startTime: time.Now(),
	}

	executor := NewAnalyticsQueryExecutor(handler)

	callCount := 0
	queryFunc := func(ctx context.Context, filter database.LocationStatsFilter, param interface{}) (interface{}, error) {
		callCount++
		return map[string]interface{}{"param": param, "call": callCount}, nil
	}

	// Execute twice
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/test", nil)
		w := httptest.NewRecorder()
		executor.ExecuteWithParam(w, req, "NilCacheParamTest", queryFunc, 50)

		if w.Code != http.StatusOK {
			t.Errorf("Request %d failed: %d", i+1, w.Code)
		}
	}

	if callCount != 2 {
		t.Errorf("Without cache, query should be executed twice, but was called %d times", callCount)
	}
}

// TestSpatialExecutor_NilDatabase tests ExecuteWithCache with nil database
func TestSpatialExecutor_NilDatabase(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		db:        nil, // Nil database
		cache:     cache.New(5 * time.Minute),
		config:    &config.Config{},
		startTime: time.Now(),
	}

	executor := NewSpatialQueryExecutor(handler)

	queryFunc := func(ctx context.Context, filter database.LocationStatsFilter, params interface{}) (interface{}, error) {
		t.Error("Query function should not be called when database is nil")
		return nil, nil
	}

	bbox := &BoundingBoxParams{West: -74.0, South: 40.0, East: -73.0, North: 41.0}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/spatial/test", nil)
	w := httptest.NewRecorder()

	executor.ExecuteWithCache(w, req, "NilDBSpatialTest", queryFunc, bbox, bbox)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503, got %d", w.Code)
	}
}

// TestSpatialExecutor_NilCache tests ExecuteWithCache with nil cache
func TestSpatialExecutor_NilCache(t *testing.T) {
	t.Parallel()

	// Create a DB struct and set spatial as available for testing
	db := &database.DB{}
	db.SetSpatialAvailableForTesting(true)

	handler := &Handler{
		db:        db,
		cache:     nil, // Nil cache
		config:    &config.Config{},
		startTime: time.Now(),
	}

	executor := NewSpatialQueryExecutor(handler)

	callCount := 0
	queryFunc := func(ctx context.Context, filter database.LocationStatsFilter, params interface{}) (interface{}, error) {
		callCount++
		return map[string]interface{}{"call": callCount}, nil
	}

	bbox := &BoundingBoxParams{West: -74.0, South: 40.0, East: -73.0, North: 41.0}

	// Execute twice
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/spatial/test", nil)
		w := httptest.NewRecorder()
		executor.ExecuteWithCache(w, req, "NilCacheSpatialTest", queryFunc, bbox, bbox)

		if w.Code != http.StatusOK {
			t.Errorf("Request %d failed: %d", i+1, w.Code)
		}
	}

	if callCount != 2 {
		t.Errorf("Without cache, query should be executed twice, but was called %d times", callCount)
	}
}

// TestAnalyticsExecutor_ContextCancellation tests query cancellation via context
func TestAnalyticsExecutor_ContextCancellation(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		db:        &database.DB{},
		cache:     cache.New(5 * time.Minute),
		config:    &config.Config{},
		startTime: time.Now(),
	}

	executor := NewAnalyticsQueryExecutor(handler)

	queryFunc := func(ctx context.Context, filter database.LocationStatsFilter) (interface{}, error) {
		// Check if context is passed correctly
		if ctx == nil {
			t.Error("Context should not be nil")
		}
		return map[string]interface{}{"context_present": true}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/test", nil)
	w := httptest.NewRecorder()

	executor.ExecuteSimple(w, req, "ContextTest", queryFunc)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// TestValidateFloatParam_EdgeCases tests edge cases for float validation
func TestValidateFloatParam_EdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		value     string
		paramName string
		minVal    float64
		maxVal    float64
		wantErr   bool
	}{
		{"very small positive", "0.000001", "val", 0, 1, false},
		{"very small negative", "-0.000001", "val", -1, 0, false},
		{"scientific notation", "1e10", "val", 0, 1e11, false},
		{"negative scientific", "-1e-5", "val", -1, 0, false},
		{"whitespace only", "   ", "val", 0, 1, true},
		{"newline", "\n", "val", 0, 1, true},
		{"tab", "\t", "val", 0, 1, true},
		{"special chars", "@#$", "val", 0, 1, true},
		{"unicode number", "123", "val", 0, 200, false},
		{"inf", "inf", "val", 0, 1e308, true},
		{"nan", "nan", "val", 0, 1, false}, // Go's Sscanf parses "nan" as NaN, which passes range checks,
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := validateFloatParam(tt.value, tt.paramName, tt.minVal, tt.maxVal)
			gotErr := err != nil

			if gotErr != tt.wantErr {
				t.Errorf("validateFloatParam(%q) error = %v, wantErr %v", tt.value, err, tt.wantErr)
			}
		})
	}
}

// TestValidateIntParam_EdgeCases tests edge cases for integer validation
func TestValidateIntParam_EdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		value     string
		paramName string
		minVal    int
		maxVal    int
		wantErr   bool
	}{
		{"leading zeros", "007", "val", 0, 10, false},
		{"with plus sign", "+5", "val", 0, 10, false},
		{"with minus sign", "-5", "val", -10, 10, false},
		{"whitespace around", " 5 ", "val", 0, 10, true}, // strict parsing
		{"decimal - should fail", "5.5", "val", 0, 10, true},
		{"overflow", "99999999999999999999", "val", 0, 100, true},
		{"binary format", "0b101", "val", 0, 10, true},
		{"octal format", "0o10", "val", 0, 10, true},
		{"hex format", "0x10", "val", 0, 20, true},
		{"empty spaces", "  ", "val", 0, 10, true},
		{"mixed chars and numbers", "5a", "val", 0, 10, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := validateIntParam(tt.value, tt.paramName, tt.minVal, tt.maxVal)
			gotErr := err != nil

			if gotErr != tt.wantErr {
				t.Errorf("validateIntParam(%q) error = %v, wantErr %v", tt.value, err, tt.wantErr)
			}
		})
	}
}

// TestValidateBoundingBox_AntiMeridian tests bounding box crossing the anti-meridian
func TestValidateBoundingBox_AntiMeridian(t *testing.T) {
	t.Parallel()

	// Test bounding box that crosses the anti-meridian (international date line)
	// This is valid - west > east when crossing the date line
	url := "/test?west=170&south=-10&east=-170&north=10"
	req := httptest.NewRequest(http.MethodGet, url, nil)

	box, err := ValidateBoundingBox(req)
	if err != nil {
		t.Errorf("Valid anti-meridian crossing bbox should not error: %v", err)
	}

	if box == nil {
		t.Fatal("Box should not be nil")
	}

	// Verify the values are correctly parsed even when west > east
	if box.West != 170 {
		t.Errorf("West = %f, want 170", box.West)
	}
	if box.East != -170 {
		t.Errorf("East = %f, want -170", box.East)
	}
}

// TestValidateCoordinates_SpecialLocations tests coordinates for special geographic locations
func TestValidateCoordinates_SpecialLocations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		lat    string
		lon    string
		radius string
	}{
		{"North Pole", "90", "0", "100"},
		{"South Pole", "-90", "0", "100"},
		{"Prime Meridian at Equator", "0", "0", "50"},
		{"Date Line at Equator", "0", "180", "50"},
		{"Date Line (negative)", "0", "-180", "50"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/test?lat=" + tt.lat + "&lon=" + tt.lon + "&radius=" + tt.radius
			req := httptest.NewRequest(http.MethodGet, url, nil)

			coords, err := ValidateCoordinates(req, true)
			if err != nil {
				t.Errorf("Coordinates for %s should be valid: %v", tt.name, err)
			}

			if coords == nil {
				t.Fatal("Coords should not be nil")
			}
		})
	}
}

// TestExecuteSimple_LargeResponseData tests handling of large response data
func TestExecuteSimple_LargeResponseData(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		db:        &database.DB{},
		cache:     cache.New(5 * time.Minute),
		config:    &config.Config{},
		startTime: time.Now(),
	}

	executor := NewAnalyticsQueryExecutor(handler)

	// Generate large response data
	largeData := make([]map[string]interface{}, 1000)
	for i := 0; i < 1000; i++ {
		largeData[i] = map[string]interface{}{
			"id":    i,
			"value": "test data entry",
			"nested": map[string]interface{}{
				"key": "value",
			},
		}
	}

	queryFunc := func(ctx context.Context, filter database.LocationStatsFilter) (interface{}, error) {
		return largeData, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/test", nil)
	w := httptest.NewRecorder()

	executor.ExecuteSimple(w, req, "LargeDataTest", queryFunc)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify response can be decoded
	var response models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Errorf("Failed to decode large response: %v", err)
	}

	if response.Status != "success" {
		t.Errorf("Expected status 'success', got '%s'", response.Status)
	}
}

// TestExecuteWithParam_StructParam tests ExecuteWithParam with struct parameter
func TestExecuteWithParam_StructParam(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		db:        &database.DB{},
		cache:     cache.New(5 * time.Minute),
		config:    &config.Config{},
		startTime: time.Now(),
	}

	executor := NewAnalyticsQueryExecutor(handler)

	type CustomParam struct {
		Limit    int
		Interval string
		Grouped  bool
	}

	param := CustomParam{
		Limit:    25,
		Interval: "hour",
		Grouped:  true,
	}

	queryFunc := func(ctx context.Context, filter database.LocationStatsFilter, p interface{}) (interface{}, error) {
		cp := p.(CustomParam)
		return map[string]interface{}{
			"limit":    cp.Limit,
			"interval": cp.Interval,
			"grouped":  cp.Grouped,
		}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/test", nil)
	w := httptest.NewRecorder()

	executor.ExecuteWithParam(w, req, "StructParamTest", queryFunc, param)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response models.APIResponse
	json.NewDecoder(w.Body).Decode(&response)

	if response.Status != "success" {
		t.Errorf("Expected status 'success', got '%s'", response.Status)
	}
}

// Benchmark tests for edge cases

// BenchmarkValidateFloatParam benchmarks float parameter validation
func BenchmarkValidateFloatParam(b *testing.B) {
	b.Run("valid", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			validateFloatParam("40.7128", "lat", -90, 90)
		}
	})

	b.Run("invalid", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			validateFloatParam("invalid", "lat", -90, 90)
		}
	})
}

// BenchmarkValidateIntParam benchmarks integer parameter validation
func BenchmarkValidateIntParam(b *testing.B) {
	b.Run("valid", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			validateIntParam("7", "resolution", 6, 8)
		}
	})

	b.Run("invalid", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			validateIntParam("invalid", "resolution", 6, 8)
		}
	})
}
