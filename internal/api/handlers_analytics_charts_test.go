// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package api provides HTTP handlers for the Cartographus application.
//
// handlers_analytics_charts_test.go - Tests for Advanced Chart Analytics handlers.
//
// These tests verify:
//   - HTTP method validation (only GET allowed)
//   - Database unavailability handling (503 when db is nil)
//   - Response format correctness
//   - Cache key prefix usage
//   - RBAC enforcement (admin-only vs user-scoped endpoints)
package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/goccy/go-json"

	"github.com/tomtom215/cartographus/internal/auth"
	"github.com/tomtom215/cartographus/internal/cache"
	"github.com/tomtom215/cartographus/internal/models"
)

// ============================================================================
// RBAC Test Helpers
// ============================================================================

// requestWithUserAuth creates a request with a regular (non-admin) user context.
func requestWithUserAuth(method, path string) *http.Request {
	req := httptest.NewRequest(method, path, nil)
	subject := &auth.AuthSubject{
		ID:       "user-123",
		Username: "testuser",
		Roles:    []string{models.RoleViewer},
	}
	ctx := context.WithValue(req.Context(), auth.AuthSubjectContextKey, subject)
	return req.WithContext(ctx)
}

// requestWithAdminAuth creates a request with an admin user context.
func requestWithAdminAuthCharts(method, path string) *http.Request {
	req := httptest.NewRequest(method, path, nil)
	subject := &auth.AuthSubject{
		ID:       "admin-123",
		Username: "admin",
		Roles:    []string{models.RoleAdmin},
	}
	ctx := context.WithValue(req.Context(), auth.AuthSubjectContextKey, subject)
	return req.WithContext(ctx)
}

// ============================================================================
// RBAC Tests for Admin-Only Endpoints (UserOverlap, BumpChart)
// ============================================================================

// TestAnalyticsUserOverlap_RBAC_Unauthenticated tests 401 for unauthenticated requests.
func TestAnalyticsUserOverlap_RBAC_Unauthenticated(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		cache: cache.New(5 * time.Minute),
		db:    nil,
	}

	// No auth context - should return 401
	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/user-overlap", nil)
	w := httptest.NewRecorder()

	handler.AnalyticsUserOverlap(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401 for unauthenticated request, got %d", w.Code)
	}

	var resp models.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}
	if resp.Error == nil || resp.Error.Code != "AUTH_REQUIRED" {
		t.Errorf("Expected error code AUTH_REQUIRED, got %v", resp.Error)
	}
}

// TestAnalyticsUserOverlap_RBAC_NonAdmin tests 403 for non-admin authenticated users.
func TestAnalyticsUserOverlap_RBAC_NonAdmin(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		cache: cache.New(5 * time.Minute),
		db:    nil,
	}

	// Regular user auth - should return 403
	req := requestWithUserAuth(http.MethodGet, "/api/v1/analytics/user-overlap")
	w := httptest.NewRecorder()

	handler.AnalyticsUserOverlap(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status 403 for non-admin request, got %d", w.Code)
	}

	var resp models.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}
	if resp.Error == nil || resp.Error.Code != "ADMIN_REQUIRED" {
		t.Errorf("Expected error code ADMIN_REQUIRED, got %v", resp.Error)
	}
}

// TestAnalyticsUserOverlap_RBAC_Admin tests that admin users can access the endpoint.
func TestAnalyticsUserOverlap_RBAC_Admin(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		cache: cache.New(5 * time.Minute),
		db:    nil, // Will return 503, but NOT 401 or 403
	}

	// Admin auth - should proceed (will get 503 due to nil db, not 401/403)
	req := requestWithAdminAuthCharts(http.MethodGet, "/api/v1/analytics/user-overlap")
	w := httptest.NewRecorder()

	handler.AnalyticsUserOverlap(w, req)

	// Should NOT be 401 or 403
	if w.Code == http.StatusUnauthorized || w.Code == http.StatusForbidden {
		t.Errorf("Admin request should not return 401/403, got %d", w.Code)
	}
	// Should be 503 (db unavailable) which means RBAC passed
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503 (db unavailable) for admin request, got %d", w.Code)
	}
}

// TestAnalyticsBumpChart_RBAC_Unauthenticated tests 401 for unauthenticated requests.
func TestAnalyticsBumpChart_RBAC_Unauthenticated(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		cache: cache.New(5 * time.Minute),
		db:    nil,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/bump-chart", nil)
	w := httptest.NewRecorder()

	handler.AnalyticsBumpChart(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401 for unauthenticated request, got %d", w.Code)
	}

	var resp models.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}
	if resp.Error == nil || resp.Error.Code != "AUTH_REQUIRED" {
		t.Errorf("Expected error code AUTH_REQUIRED, got %v", resp.Error)
	}
}

// TestAnalyticsBumpChart_RBAC_NonAdmin tests 403 for non-admin authenticated users.
func TestAnalyticsBumpChart_RBAC_NonAdmin(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		cache: cache.New(5 * time.Minute),
		db:    nil,
	}

	req := requestWithUserAuth(http.MethodGet, "/api/v1/analytics/bump-chart")
	w := httptest.NewRecorder()

	handler.AnalyticsBumpChart(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status 403 for non-admin request, got %d", w.Code)
	}

	var resp models.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}
	if resp.Error == nil || resp.Error.Code != "ADMIN_REQUIRED" {
		t.Errorf("Expected error code ADMIN_REQUIRED, got %v", resp.Error)
	}
}

// TestAnalyticsBumpChart_RBAC_Admin tests that admin users can access the endpoint.
func TestAnalyticsBumpChart_RBAC_Admin(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		cache: cache.New(5 * time.Minute),
		db:    nil,
	}

	req := requestWithAdminAuthCharts(http.MethodGet, "/api/v1/analytics/bump-chart")
	w := httptest.NewRecorder()

	handler.AnalyticsBumpChart(w, req)

	if w.Code == http.StatusUnauthorized || w.Code == http.StatusForbidden {
		t.Errorf("Admin request should not return 401/403, got %d", w.Code)
	}
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503 (db unavailable) for admin request, got %d", w.Code)
	}
}

// ============================================================================
// Table-Driven RBAC Tests
// ============================================================================

// TestAdminOnlyEndpoints_RBAC tests that admin-only endpoints enforce RBAC correctly.
func TestAdminOnlyEndpoints_RBAC(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		cache: cache.New(5 * time.Minute),
		db:    nil,
	}

	adminOnlyEndpoints := []struct {
		name      string
		path      string
		handlerFn func(http.ResponseWriter, *http.Request)
	}{
		{"UserOverlap", "/api/v1/analytics/user-overlap", handler.AnalyticsUserOverlap},
		{"BumpChart", "/api/v1/analytics/bump-chart", handler.AnalyticsBumpChart},
	}

	for _, tc := range adminOnlyEndpoints {
		t.Run(tc.name+"_Unauthenticated", func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			w := httptest.NewRecorder()
			tc.handlerFn(w, req)
			if w.Code != http.StatusUnauthorized {
				t.Errorf("Expected 401 for unauthenticated, got %d", w.Code)
			}
		})

		t.Run(tc.name+"_NonAdmin", func(t *testing.T) {
			t.Parallel()
			req := requestWithUserAuth(http.MethodGet, tc.path)
			w := httptest.NewRecorder()
			tc.handlerFn(w, req)
			if w.Code != http.StatusForbidden {
				t.Errorf("Expected 403 for non-admin, got %d", w.Code)
			}
		})

		t.Run(tc.name+"_Admin", func(t *testing.T) {
			t.Parallel()
			req := requestWithAdminAuthCharts(http.MethodGet, tc.path)
			w := httptest.NewRecorder()
			tc.handlerFn(w, req)
			// Admin should pass RBAC (get 503 for nil db, not 401/403)
			if w.Code == http.StatusUnauthorized || w.Code == http.StatusForbidden {
				t.Errorf("Admin should pass RBAC, got %d", w.Code)
			}
		})
	}
}

// TestUserScopedEndpoints_AllowAuthenticatedUsers tests that user-scoped endpoints allow all authenticated users.
func TestUserScopedEndpoints_AllowAuthenticatedUsers(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		cache: cache.New(5 * time.Minute),
		db:    nil,
	}

	userScopedEndpoints := []struct {
		name      string
		path      string
		handlerFn func(http.ResponseWriter, *http.Request)
	}{
		{"ContentFlow", "/api/v1/analytics/content-flow", handler.AnalyticsContentFlow},
		{"UserProfile", "/api/v1/analytics/user-profile", handler.AnalyticsUserProfile},
		{"LibraryUtilization", "/api/v1/analytics/library-utilization", handler.AnalyticsLibraryUtilization},
		{"CalendarHeatmap", "/api/v1/analytics/calendar-heatmap", handler.AnalyticsCalendarHeatmap},
	}

	for _, tc := range userScopedEndpoints {
		t.Run(tc.name+"_RegularUser", func(t *testing.T) {
			t.Parallel()
			req := requestWithUserAuth(http.MethodGet, tc.path)
			w := httptest.NewRecorder()
			tc.handlerFn(w, req)
			// Regular users should pass (get 503 for nil db, not 401/403)
			if w.Code == http.StatusUnauthorized || w.Code == http.StatusForbidden {
				t.Errorf("Regular user should be allowed, got %d", w.Code)
			}
		})

		t.Run(tc.name+"_AdminUser", func(t *testing.T) {
			t.Parallel()
			req := requestWithAdminAuthCharts(http.MethodGet, tc.path)
			w := httptest.NewRecorder()
			tc.handlerFn(w, req)
			// Admin should also pass
			if w.Code == http.StatusUnauthorized || w.Code == http.StatusForbidden {
				t.Errorf("Admin user should be allowed, got %d", w.Code)
			}
		})
	}
}

// ============================================================================
// Content Flow Analytics (Sankey Diagram) Tests
// ============================================================================

// TestAnalyticsContentFlow_MethodNotAllowed tests that only GET is allowed.
func TestAnalyticsContentFlow_MethodNotAllowed(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		cache: cache.New(5 * time.Minute),
	}

	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/v1/analytics/content-flow", nil)
			w := httptest.NewRecorder()

			handler.AnalyticsContentFlow(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status 405 for %s, got %d", method, w.Code)
			}

			var resp models.APIResponse
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("Failed to unmarshal response: %v", err)
			}
			if resp.Error == nil || resp.Error.Code != "METHOD_NOT_ALLOWED" {
				t.Errorf("Expected error code METHOD_NOT_ALLOWED, got %v", resp.Error)
			}
		})
	}
}

// TestAnalyticsContentFlow_DatabaseUnavailable tests 503 when database is nil.
func TestAnalyticsContentFlow_DatabaseUnavailable(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		cache: cache.New(5 * time.Minute),
		db:    nil, // Database not configured
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/content-flow", nil)
	w := httptest.NewRecorder()

	handler.AnalyticsContentFlow(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503, got %d", w.Code)
	}

	var resp models.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}
	if resp.Error == nil || resp.Error.Code != "SERVICE_ERROR" {
		t.Errorf("Expected error code SERVICE_ERROR, got %v", resp.Error)
	}
}

// ============================================================================
// User Overlap Analytics (Chord Diagram) Tests
// ============================================================================

// TestAnalyticsUserOverlap_MethodNotAllowed tests that only GET is allowed.
func TestAnalyticsUserOverlap_MethodNotAllowed(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		cache: cache.New(5 * time.Minute),
	}

	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/v1/analytics/user-overlap", nil)
			w := httptest.NewRecorder()

			handler.AnalyticsUserOverlap(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status 405 for %s, got %d", method, w.Code)
			}

			var resp models.APIResponse
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("Failed to unmarshal response: %v", err)
			}
			if resp.Error == nil || resp.Error.Code != "METHOD_NOT_ALLOWED" {
				t.Errorf("Expected error code METHOD_NOT_ALLOWED, got %v", resp.Error)
			}
		})
	}
}

// TestAnalyticsUserOverlap_DatabaseUnavailable tests 503 when database is nil.
// Note: This endpoint is admin-only, so we need admin auth to reach the db check.
func TestAnalyticsUserOverlap_DatabaseUnavailable(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		cache: cache.New(5 * time.Minute),
		db:    nil,
	}

	// Must use admin auth to pass RBAC check and reach database check
	req := requestWithAdminAuthCharts(http.MethodGet, "/api/v1/analytics/user-overlap")
	w := httptest.NewRecorder()

	handler.AnalyticsUserOverlap(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503, got %d", w.Code)
	}

	var resp models.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}
	if resp.Error == nil || resp.Error.Code != "SERVICE_ERROR" {
		t.Errorf("Expected error code SERVICE_ERROR, got %v", resp.Error)
	}
}

// ============================================================================
// User Profile Analytics (Radar Chart) Tests
// ============================================================================

// TestAnalyticsUserProfile_MethodNotAllowed tests that only GET is allowed.
func TestAnalyticsUserProfile_MethodNotAllowed(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		cache: cache.New(5 * time.Minute),
	}

	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/v1/analytics/user-profile", nil)
			w := httptest.NewRecorder()

			handler.AnalyticsUserProfile(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status 405 for %s, got %d", method, w.Code)
			}

			var resp models.APIResponse
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("Failed to unmarshal response: %v", err)
			}
			if resp.Error == nil || resp.Error.Code != "METHOD_NOT_ALLOWED" {
				t.Errorf("Expected error code METHOD_NOT_ALLOWED, got %v", resp.Error)
			}
		})
	}
}

// TestAnalyticsUserProfile_DatabaseUnavailable tests 503 when database is nil.
func TestAnalyticsUserProfile_DatabaseUnavailable(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		cache: cache.New(5 * time.Minute),
		db:    nil,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/user-profile", nil)
	w := httptest.NewRecorder()

	handler.AnalyticsUserProfile(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503, got %d", w.Code)
	}

	var resp models.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}
	if resp.Error == nil || resp.Error.Code != "SERVICE_ERROR" {
		t.Errorf("Expected error code SERVICE_ERROR, got %v", resp.Error)
	}
}

// ============================================================================
// Library Utilization Analytics (Treemap) Tests
// ============================================================================

// TestAnalyticsLibraryUtilization_MethodNotAllowed tests that only GET is allowed.
func TestAnalyticsLibraryUtilization_MethodNotAllowed(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		cache: cache.New(5 * time.Minute),
	}

	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/v1/analytics/library-utilization", nil)
			w := httptest.NewRecorder()

			handler.AnalyticsLibraryUtilization(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status 405 for %s, got %d", method, w.Code)
			}

			var resp models.APIResponse
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("Failed to unmarshal response: %v", err)
			}
			if resp.Error == nil || resp.Error.Code != "METHOD_NOT_ALLOWED" {
				t.Errorf("Expected error code METHOD_NOT_ALLOWED, got %v", resp.Error)
			}
		})
	}
}

// TestAnalyticsLibraryUtilization_DatabaseUnavailable tests 503 when database is nil.
func TestAnalyticsLibraryUtilization_DatabaseUnavailable(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		cache: cache.New(5 * time.Minute),
		db:    nil,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/library-utilization", nil)
	w := httptest.NewRecorder()

	handler.AnalyticsLibraryUtilization(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503, got %d", w.Code)
	}

	var resp models.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}
	if resp.Error == nil || resp.Error.Code != "SERVICE_ERROR" {
		t.Errorf("Expected error code SERVICE_ERROR, got %v", resp.Error)
	}
}

// ============================================================================
// Calendar Heatmap Analytics Tests
// ============================================================================

// TestAnalyticsCalendarHeatmap_MethodNotAllowed tests that only GET is allowed.
func TestAnalyticsCalendarHeatmap_MethodNotAllowed(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		cache: cache.New(5 * time.Minute),
	}

	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/v1/analytics/calendar-heatmap", nil)
			w := httptest.NewRecorder()

			handler.AnalyticsCalendarHeatmap(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status 405 for %s, got %d", method, w.Code)
			}

			var resp models.APIResponse
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("Failed to unmarshal response: %v", err)
			}
			if resp.Error == nil || resp.Error.Code != "METHOD_NOT_ALLOWED" {
				t.Errorf("Expected error code METHOD_NOT_ALLOWED, got %v", resp.Error)
			}
		})
	}
}

// TestAnalyticsCalendarHeatmap_DatabaseUnavailable tests 503 when database is nil.
func TestAnalyticsCalendarHeatmap_DatabaseUnavailable(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		cache: cache.New(5 * time.Minute),
		db:    nil,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/calendar-heatmap", nil)
	w := httptest.NewRecorder()

	handler.AnalyticsCalendarHeatmap(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503, got %d", w.Code)
	}

	var resp models.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}
	if resp.Error == nil || resp.Error.Code != "SERVICE_ERROR" {
		t.Errorf("Expected error code SERVICE_ERROR, got %v", resp.Error)
	}
}

// ============================================================================
// Bump Chart Analytics Tests
// ============================================================================

// TestAnalyticsBumpChart_MethodNotAllowed tests that only GET is allowed.
func TestAnalyticsBumpChart_MethodNotAllowed(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		cache: cache.New(5 * time.Minute),
	}

	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/v1/analytics/bump-chart", nil)
			w := httptest.NewRecorder()

			handler.AnalyticsBumpChart(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status 405 for %s, got %d", method, w.Code)
			}

			var resp models.APIResponse
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("Failed to unmarshal response: %v", err)
			}
			if resp.Error == nil || resp.Error.Code != "METHOD_NOT_ALLOWED" {
				t.Errorf("Expected error code METHOD_NOT_ALLOWED, got %v", resp.Error)
			}
		})
	}
}

// TestAnalyticsBumpChart_DatabaseUnavailable tests 503 when database is nil.
// Note: This endpoint is admin-only, so we need admin auth to reach the db check.
func TestAnalyticsBumpChart_DatabaseUnavailable(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		cache: cache.New(5 * time.Minute),
		db:    nil,
	}

	// Must use admin auth to pass RBAC check and reach database check
	req := requestWithAdminAuthCharts(http.MethodGet, "/api/v1/analytics/bump-chart")
	w := httptest.NewRecorder()

	handler.AnalyticsBumpChart(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503, got %d", w.Code)
	}

	var resp models.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}
	if resp.Error == nil || resp.Error.Code != "SERVICE_ERROR" {
		t.Errorf("Expected error code SERVICE_ERROR, got %v", resp.Error)
	}
}

// ============================================================================
// Table-Driven Tests for All Advanced Chart Endpoints
// ============================================================================

// TestAdvancedChartEndpoints_ContentTypeJSON verifies JSON content-type for errors.
func TestAdvancedChartEndpoints_ContentTypeJSON(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		cache: cache.New(5 * time.Minute),
	}

	endpoints := []struct {
		name      string
		path      string
		handlerFn func(http.ResponseWriter, *http.Request)
	}{
		{"ContentFlow", "/api/v1/analytics/content-flow", handler.AnalyticsContentFlow},
		{"UserOverlap", "/api/v1/analytics/user-overlap", handler.AnalyticsUserOverlap},
		{"UserProfile", "/api/v1/analytics/user-profile", handler.AnalyticsUserProfile},
		{"LibraryUtilization", "/api/v1/analytics/library-utilization", handler.AnalyticsLibraryUtilization},
		{"CalendarHeatmap", "/api/v1/analytics/calendar-heatmap", handler.AnalyticsCalendarHeatmap},
		{"BumpChart", "/api/v1/analytics/bump-chart", handler.AnalyticsBumpChart},
	}

	for _, tc := range endpoints {
		t.Run(tc.name, func(t *testing.T) {
			// Test method not allowed response has JSON content type
			req := httptest.NewRequest(http.MethodPost, tc.path, nil)
			w := httptest.NewRecorder()

			tc.handlerFn(w, req)

			contentType := w.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("Expected Content-Type application/json, got %s", contentType)
			}
		})
	}
}

// TestAdvancedChartEndpoints_AllowedMethod verifies GET requests don't return 405.
// Note: Admin-only endpoints need admin auth, user-scoped endpoints allow any auth.
func TestAdvancedChartEndpoints_AllowedMethod(t *testing.T) {
	t.Parallel()

	handler := &Handler{
		cache: cache.New(5 * time.Minute),
		db:    nil, // Will trigger 503, but not 405
	}

	// User-scoped endpoints (allow any authenticated user)
	userScopedEndpoints := []struct {
		name      string
		path      string
		handlerFn func(http.ResponseWriter, *http.Request)
	}{
		{"ContentFlow", "/api/v1/analytics/content-flow", handler.AnalyticsContentFlow},
		{"UserProfile", "/api/v1/analytics/user-profile", handler.AnalyticsUserProfile},
		{"LibraryUtilization", "/api/v1/analytics/library-utilization", handler.AnalyticsLibraryUtilization},
		{"CalendarHeatmap", "/api/v1/analytics/calendar-heatmap", handler.AnalyticsCalendarHeatmap},
	}

	// Admin-only endpoints
	adminOnlyEndpoints := []struct {
		name      string
		path      string
		handlerFn func(http.ResponseWriter, *http.Request)
	}{
		{"UserOverlap", "/api/v1/analytics/user-overlap", handler.AnalyticsUserOverlap},
		{"BumpChart", "/api/v1/analytics/bump-chart", handler.AnalyticsBumpChart},
	}

	// Test user-scoped endpoints with regular user auth
	for _, tc := range userScopedEndpoints {
		t.Run(tc.name, func(t *testing.T) {
			req := requestWithUserAuth(http.MethodGet, tc.path)
			w := httptest.NewRecorder()

			tc.handlerFn(w, req)

			if w.Code == http.StatusMethodNotAllowed {
				t.Errorf("GET request should not return 405, got %d", w.Code)
			}
		})
	}

	// Test admin-only endpoints with admin auth
	for _, tc := range adminOnlyEndpoints {
		t.Run(tc.name, func(t *testing.T) {
			req := requestWithAdminAuthCharts(http.MethodGet, tc.path)
			w := httptest.NewRecorder()

			tc.handlerFn(w, req)

			if w.Code == http.StatusMethodNotAllowed {
				t.Errorf("GET request should not return 405, got %d", w.Code)
			}
		})
	}
}
