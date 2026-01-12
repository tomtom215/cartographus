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

	"github.com/tomtom215/cartographus/internal/cache"
)

// TestAnalyticsTrends_NilDatabase tests AnalyticsTrends when database is nil
func TestAnalyticsTrends_NilDatabase(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		db:    nil,
		cache: cache.New(5 * time.Minute),
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/trends", nil)
	w := httptest.NewRecorder()

	handler.AnalyticsTrends(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503 for nil database, got %d", w.Code)
	}
}

// TestAnalyticsBandwidth_NilDatabase tests AnalyticsBandwidth when database is nil
func TestAnalyticsBandwidth_NilDatabase(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		db:    nil,
		cache: cache.New(5 * time.Minute),
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/bandwidth", nil)
	w := httptest.NewRecorder()

	handler.AnalyticsBandwidth(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503 for nil database, got %d", w.Code)
	}
}

// TestAnalyticsGeographic_NilDatabase tests AnalyticsGeographic when database is nil
func TestAnalyticsGeographic_NilDatabase(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		db:    nil,
		cache: cache.New(5 * time.Minute),
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/geographic", nil)
	w := httptest.NewRecorder()

	handler.AnalyticsGeographic(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503 for nil database, got %d", w.Code)
	}
}

// TestAnalyticsTrends_WithCaching tests that analytics endpoints use caching
func TestAnalyticsTrends_WithCaching(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()

	handler := &Handler{
		db:    db,
		cache: cache.New(5 * time.Minute),
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/trends", nil)

	// First request - cache miss
	w1 := httptest.NewRecorder()
	handler.AnalyticsTrends(w1, req)

	if w1.Code != http.StatusOK {
		t.Errorf("First request failed with status %d", w1.Code)
	}

	// Second request - should hit cache
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/trends", nil)
	handler.AnalyticsTrends(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("Second request failed with status %d", w2.Code)
	}

	// Both should return same response
	// Note: We can't easily verify cache hit without instrumenting the handler
	// This test verifies caching doesn't break functionality
}
