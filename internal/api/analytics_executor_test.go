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

// setupAnalyticsExecutorHandler creates a handler for analytics executor testing
func setupAnalyticsExecutorHandler(t *testing.T) *Handler {
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

	return &Handler{
		db:        &database.DB{}, // Empty db struct to pass nil check (mock queryFunc doesn't use it)
		cache:     cache.New(5 * time.Minute),
		config:    cfg,
		startTime: time.Now(),
	}
}

// TestNewAnalyticsQueryExecutor tests the executor constructor
func TestNewAnalyticsQueryExecutor(t *testing.T) {
	t.Parallel()

	handler := setupAnalyticsExecutorHandler(t)
	executor := NewAnalyticsQueryExecutor(handler)

	if executor == nil {
		t.Fatal("NewAnalyticsQueryExecutor returned nil")
	}

	if executor.handler != handler {
		t.Error("Handler not set correctly")
	}
}

// TestExecuteSimple tests the ExecuteSimple method
func TestExecuteSimple(t *testing.T) {
	t.Parallel()

	t.Run("successful query", func(t *testing.T) {
		handler := setupAnalyticsExecutorHandler(t)
		executor := NewAnalyticsQueryExecutor(handler)

		expectedData := map[string]interface{}{
			"total": 100,
			"items": []string{"a", "b", "c"},
		}

		queryFunc := func(ctx context.Context, filter database.LocationStatsFilter) (interface{}, error) {
			return expectedData, nil
		}

		req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/test", nil)
		w := httptest.NewRecorder()

		executor.ExecuteSimple(w, req, "TestQuery", queryFunc)

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
		handler := setupAnalyticsExecutorHandler(t)
		executor := NewAnalyticsQueryExecutor(handler)

		expectedData := map[string]interface{}{"cached": true}

		queryFunc := func(ctx context.Context, filter database.LocationStatsFilter) (interface{}, error) {
			return expectedData, nil
		}

		// First request - should execute query
		req1 := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/test", nil)
		w1 := httptest.NewRecorder()
		executor.ExecuteSimple(w1, req1, "CacheTest", queryFunc)

		if w1.Code != http.StatusOK {
			t.Fatalf("First request failed: %d", w1.Code)
		}

		// Second request - should be cached
		req2 := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/test", nil)
		w2 := httptest.NewRecorder()
		executor.ExecuteSimple(w2, req2, "CacheTest", queryFunc)

		if w2.Code != http.StatusOK {
			t.Errorf("Second request failed: %d", w2.Code)
		}

		var response models.APIResponse
		if err := json.NewDecoder(w2.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if !response.Metadata.Cached {
			t.Error("Second query should be cached")
		}

		if response.Metadata.QueryTimeMS != 0 {
			t.Errorf("Cached query should have 0 query time, got %d", response.Metadata.QueryTimeMS)
		}
	})

	t.Run("query error", func(t *testing.T) {
		handler := setupAnalyticsExecutorHandler(t)
		executor := NewAnalyticsQueryExecutor(handler)

		queryFunc := func(ctx context.Context, filter database.LocationStatsFilter) (interface{}, error) {
			return nil, errors.New("database error")
		}

		req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/test", nil)
		w := httptest.NewRecorder()

		executor.ExecuteSimple(w, req, "ErrorTest", queryFunc)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 500, got %d", w.Code)
		}

		var response models.APIResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response.Status != "error" {
			t.Errorf("Expected status 'error', got '%s'", response.Status)
		}
	})

	t.Run("with query parameters", func(t *testing.T) {
		handler := setupAnalyticsExecutorHandler(t)
		executor := NewAnalyticsQueryExecutor(handler)

		queryFunc := func(ctx context.Context, filter database.LocationStatsFilter) (interface{}, error) {
			return map[string]interface{}{
				"users":       filter.Users,
				"media_types": filter.MediaTypes,
			}, nil
		}

		req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/test?users=alice,bob&media_types=movie,episode", nil)
		w := httptest.NewRecorder()

		executor.ExecuteSimple(w, req, "ParamsTest", queryFunc)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})
}

// TestExecuteWithParam tests the ExecuteWithParam method
func TestExecuteWithParam(t *testing.T) {
	t.Parallel()

	t.Run("successful query with int param", func(t *testing.T) {
		handler := setupAnalyticsExecutorHandler(t)
		executor := NewAnalyticsQueryExecutor(handler)

		expectedData := []string{"item1", "item2", "item3"}

		queryFunc := func(ctx context.Context, filter database.LocationStatsFilter, param interface{}) (interface{}, error) {
			limit := param.(int)
			return map[string]interface{}{
				"limit": limit,
				"items": expectedData,
			}, nil
		}

		req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/users", nil)
		w := httptest.NewRecorder()

		executor.ExecuteWithParam(w, req, "UsersQuery", queryFunc, 10)

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
	})

	t.Run("successful query with string param", func(t *testing.T) {
		handler := setupAnalyticsExecutorHandler(t)
		executor := NewAnalyticsQueryExecutor(handler)

		queryFunc := func(ctx context.Context, filter database.LocationStatsFilter, param interface{}) (interface{}, error) {
			interval := param.(string)
			return map[string]interface{}{
				"interval": interval,
			}, nil
		}

		req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/temporal", nil)
		w := httptest.NewRecorder()

		executor.ExecuteWithParam(w, req, "TemporalQuery", queryFunc, "hour")

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("cached response with param", func(t *testing.T) {
		handler := setupAnalyticsExecutorHandler(t)
		executor := NewAnalyticsQueryExecutor(handler)

		callCount := 0
		queryFunc := func(ctx context.Context, filter database.LocationStatsFilter, param interface{}) (interface{}, error) {
			callCount++
			return map[string]interface{}{"count": callCount}, nil
		}

		// First request
		req1 := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/test", nil)
		w1 := httptest.NewRecorder()
		executor.ExecuteWithParam(w1, req1, "CacheParamTest", queryFunc, 20)

		// Second request with same param - should be cached
		req2 := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/test", nil)
		w2 := httptest.NewRecorder()
		executor.ExecuteWithParam(w2, req2, "CacheParamTest", queryFunc, 20)

		if callCount != 1 {
			t.Errorf("Query should only be executed once, was called %d times", callCount)
		}

		var response models.APIResponse
		json.NewDecoder(w2.Body).Decode(&response)
		if !response.Metadata.Cached {
			t.Error("Second request should be cached")
		}
	})

	t.Run("different params use different cache keys", func(t *testing.T) {
		handler := setupAnalyticsExecutorHandler(t)
		executor := NewAnalyticsQueryExecutor(handler)

		callCount := 0
		queryFunc := func(ctx context.Context, filter database.LocationStatsFilter, param interface{}) (interface{}, error) {
			callCount++
			limit := param.(int)
			return map[string]interface{}{"limit": limit, "call": callCount}, nil
		}

		// First request with limit=10
		req1 := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/test", nil)
		w1 := httptest.NewRecorder()
		executor.ExecuteWithParam(w1, req1, "DiffParamTest", queryFunc, 10)

		// Second request with limit=20 - should NOT be cached
		req2 := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/test", nil)
		w2 := httptest.NewRecorder()
		executor.ExecuteWithParam(w2, req2, "DiffParamTest", queryFunc, 20)

		if callCount != 2 {
			t.Errorf("Different params should trigger separate queries, got %d calls", callCount)
		}
	})

	t.Run("query error with param", func(t *testing.T) {
		handler := setupAnalyticsExecutorHandler(t)
		executor := NewAnalyticsQueryExecutor(handler)

		queryFunc := func(ctx context.Context, filter database.LocationStatsFilter, param interface{}) (interface{}, error) {
			return nil, errors.New("query failed")
		}

		req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/test", nil)
		w := httptest.NewRecorder()

		executor.ExecuteWithParam(w, req, "ErrorParamTest", queryFunc, "invalid")

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 500, got %d", w.Code)
		}
	})
}

// TestExecuteSimple_WithFilters tests ExecuteSimple with various filter parameters
func TestExecuteSimple_WithFilters(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		queryParams string
	}{
		{"with users filter", "users=alice,bob,charlie"},
		{"with media_types filter", "media_types=movie,episode"},
		{"with platforms filter", "platforms=Android,iOS"},
		{"with players filter", "players=Plex+for+Android"},
		{"with days filter", "days=7"},
		{"with start and end date", "start_date=2025-01-01T00:00:00Z&end_date=2025-12-31T23:59:59Z"},
		{"with multiple filters", "users=alice&media_types=movie&days=30"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := setupAnalyticsExecutorHandler(t)
			executor := NewAnalyticsQueryExecutor(handler)

			queryFunc := func(ctx context.Context, filter database.LocationStatsFilter) (interface{}, error) {
				return map[string]interface{}{"filter_applied": true}, nil
			}

			req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/test?"+tt.queryParams, nil)
			w := httptest.NewRecorder()

			executor.ExecuteSimple(w, req, tt.name, queryFunc)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", w.Code)
			}
		})
	}
}

// TestExecuteSimple_QueryTimeMeasurement tests that query time is measured correctly
func TestExecuteSimple_QueryTimeMeasurement(t *testing.T) {
	t.Parallel()

	handler := setupAnalyticsExecutorHandler(t)
	executor := NewAnalyticsQueryExecutor(handler)

	queryFunc := func(ctx context.Context, filter database.LocationStatsFilter) (interface{}, error) {
		// Simulate some processing time
		time.Sleep(10 * time.Millisecond)
		return map[string]interface{}{"data": "test"}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/test", nil)
	w := httptest.NewRecorder()

	executor.ExecuteSimple(w, req, "TimeTest", queryFunc)

	var response models.APIResponse
	json.NewDecoder(w.Body).Decode(&response)

	if response.Metadata.QueryTimeMS < 10 {
		t.Errorf("Query time should be at least 10ms, got %d", response.Metadata.QueryTimeMS)
	}
}

// BenchmarkExecuteSimple benchmarks the ExecuteSimple method
func BenchmarkExecuteSimple(b *testing.B) {
	handler := &Handler{
		db:    &database.DB{}, // Empty db struct to pass nil check
		cache: cache.New(5 * time.Minute),
		config: &config.Config{
			API: config.APIConfig{
				DefaultPageSize: 100,
				MaxPageSize:     1000,
			},
		},
	}
	executor := NewAnalyticsQueryExecutor(handler)

	queryFunc := func(ctx context.Context, filter database.LocationStatsFilter) (interface{}, error) {
		return map[string]interface{}{"data": "test"}, nil
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/test", nil)
		w := httptest.NewRecorder()
		executor.ExecuteSimple(w, req, "BenchTest", queryFunc)
	}
}

// BenchmarkExecuteSimple_CacheHit benchmarks cache hit performance
func BenchmarkExecuteSimple_CacheHit(b *testing.B) {
	handler := &Handler{
		db:    &database.DB{}, // Empty db struct to pass nil check
		cache: cache.New(5 * time.Minute),
		config: &config.Config{
			API: config.APIConfig{
				DefaultPageSize: 100,
				MaxPageSize:     1000,
			},
		},
	}
	executor := NewAnalyticsQueryExecutor(handler)

	queryFunc := func(ctx context.Context, filter database.LocationStatsFilter) (interface{}, error) {
		return map[string]interface{}{"data": "test"}, nil
	}

	// Prime the cache
	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/test", nil)
	w := httptest.NewRecorder()
	executor.ExecuteSimple(w, req, "CacheHitBench", queryFunc)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/test", nil)
		w := httptest.NewRecorder()
		executor.ExecuteSimple(w, req, "CacheHitBench", queryFunc)
	}
}

// BenchmarkExecuteWithParam benchmarks the ExecuteWithParam method
func BenchmarkExecuteWithParam(b *testing.B) {
	handler := &Handler{
		db:    &database.DB{}, // Empty db struct to pass nil check
		cache: cache.New(5 * time.Minute),
		config: &config.Config{
			API: config.APIConfig{
				DefaultPageSize: 100,
				MaxPageSize:     1000,
			},
		},
	}
	executor := NewAnalyticsQueryExecutor(handler)

	queryFunc := func(ctx context.Context, filter database.LocationStatsFilter, param interface{}) (interface{}, error) {
		return map[string]interface{}{"limit": param}, nil
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/test", nil)
		w := httptest.NewRecorder()
		executor.ExecuteWithParam(w, req, "BenchParamTest", queryFunc, 10)
	}
}
