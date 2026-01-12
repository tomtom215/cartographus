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

	"github.com/tomtom215/cartographus/internal/auth"
)

// =====================================================
// ChiMiddleware Configuration Tests
// =====================================================

func TestNewChiMiddleware_DefaultConfig(t *testing.T) {
	m := NewChiMiddleware(nil)

	if m == nil {
		t.Fatal("NewChiMiddleware returned nil")
	}
	if m.config == nil {
		t.Fatal("config is nil")
	}
	// Default should be empty (secure by default - requires explicit configuration)
	if len(m.config.CORSAllowedOrigins) != 0 {
		t.Errorf("CORSAllowedOrigins = %v, want []", m.config.CORSAllowedOrigins)
	}
	if m.config.CORSMaxAge != 86400 {
		t.Errorf("CORSMaxAge = %d, want 86400", m.config.CORSMaxAge)
	}
}

func TestNewChiMiddleware_CustomConfig(t *testing.T) {
	config := &ChiMiddlewareConfig{
		CORSAllowedOrigins: []string{"https://example.com"},
		CORSAllowedMethods: []string{"GET", "POST"},
		CORSAllowedHeaders: []string{"Content-Type"},
		CORSMaxAge:         3600,
		RateLimitRequests:  50,
		RateLimitWindow:    time.Second * 30,
		RateLimitDisabled:  true,
	}

	m := NewChiMiddleware(config)

	if m.config.CORSAllowedOrigins[0] != "https://example.com" {
		t.Errorf("CORSAllowedOrigins = %v, want [https://example.com]", m.config.CORSAllowedOrigins)
	}
	if m.config.RateLimitRequests != 50 {
		t.Errorf("RateLimitRequests = %d, want 50", m.config.RateLimitRequests)
	}
	if !m.config.RateLimitDisabled {
		t.Error("RateLimitDisabled should be true")
	}
}

func TestNewChiMiddlewareFromAuth(t *testing.T) {
	corsOrigins := []string{"https://example.com", "https://other.com"}
	m := NewChiMiddlewareFromAuth(corsOrigins, 200, time.Minute*2, false)

	if len(m.config.CORSAllowedOrigins) != 2 {
		t.Errorf("CORSAllowedOrigins length = %d, want 2", len(m.config.CORSAllowedOrigins))
	}
	if m.config.RateLimitRequests != 200 {
		t.Errorf("RateLimitRequests = %d, want 200", m.config.RateLimitRequests)
	}
	if m.config.RateLimitWindow != time.Minute*2 {
		t.Errorf("RateLimitWindow = %v, want 2m", m.config.RateLimitWindow)
	}
}

// =====================================================
// CORS Middleware Tests
// =====================================================

func TestChiMiddleware_CORS_WildcardOrigin(t *testing.T) {
	config := &ChiMiddlewareConfig{
		CORSAllowedOrigins: []string{"*"},
		CORSAllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		CORSAllowedHeaders: []string{"Content-Type", "Authorization"},
		CORSMaxAge:         86400,
	}
	m := NewChiMiddleware(config)

	handlerCalled := false
	handler := m.CORS()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "https://example.com")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if !handlerCalled {
		t.Error("Handler should be called")
	}

	// Check CORS headers
	allowOrigin := w.Header().Get("Access-Control-Allow-Origin")
	if allowOrigin != "*" {
		t.Errorf("Access-Control-Allow-Origin = %q, want *", allowOrigin)
	}
}

func TestChiMiddleware_CORS_SpecificOrigin(t *testing.T) {
	config := &ChiMiddlewareConfig{
		CORSAllowedOrigins: []string{"https://allowed.com"},
		CORSAllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		CORSAllowedHeaders: []string{"Content-Type", "Authorization"},
		CORSMaxAge:         86400,
	}
	m := NewChiMiddleware(config)

	handlerCalled := false
	handler := m.CORS()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "https://allowed.com")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if !handlerCalled {
		t.Error("Handler should be called")
	}

	// Check CORS headers - go-chi/cors reflects the specific origin
	allowOrigin := w.Header().Get("Access-Control-Allow-Origin")
	if allowOrigin != "https://allowed.com" {
		t.Errorf("Access-Control-Allow-Origin = %q, want https://allowed.com", allowOrigin)
	}

	// Vary header should be set for specific origins
	vary := w.Header().Get("Vary")
	if vary == "" {
		t.Error("Vary header should be set for specific origins")
	}
}

func TestChiMiddleware_CORS_PreflightRequest(t *testing.T) {
	config := &ChiMiddlewareConfig{
		CORSAllowedOrigins: []string{"*"},
		CORSAllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		CORSAllowedHeaders: []string{"Content-Type", "Authorization"},
		CORSMaxAge:         86400,
	}
	m := NewChiMiddleware(config)

	handlerCalled := false
	handler := m.CORS()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	}))

	// OPTIONS preflight request
	req := httptest.NewRequest("OPTIONS", "/", nil)
	req.Header.Set("Origin", "https://example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// Preflight should return 200 or 204 without calling handler
	if w.Code != http.StatusOK && w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want 200 or 204", w.Code)
	}
	if handlerCalled {
		t.Error("Handler should not be called for OPTIONS preflight")
	}

	// Check preflight response headers
	if w.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Error("Access-Control-Allow-Methods should be set")
	}
}

func TestChiMiddleware_CORS_DisallowedOrigin(t *testing.T) {
	config := &ChiMiddlewareConfig{
		CORSAllowedOrigins: []string{"https://allowed.com"},
		CORSAllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		CORSAllowedHeaders: []string{"Content-Type", "Authorization"},
		CORSMaxAge:         86400,
	}
	m := NewChiMiddleware(config)

	handler := m.CORS()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Request from disallowed origin
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "https://not-allowed.com")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// go-chi/cors doesn't block the request, but doesn't set CORS headers
	// The browser will block the response due to CORS policy
	allowOrigin := w.Header().Get("Access-Control-Allow-Origin")
	if allowOrigin != "" {
		t.Errorf("Access-Control-Allow-Origin should not be set for disallowed origin, got %q", allowOrigin)
	}
}

func TestChiMiddleware_CORS_PreflightDisallowedOrigin(t *testing.T) {
	config := &ChiMiddlewareConfig{
		CORSAllowedOrigins: []string{"https://allowed.com"},
		CORSAllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		CORSAllowedHeaders: []string{"Content-Type", "Authorization"},
		CORSMaxAge:         86400,
	}
	m := NewChiMiddleware(config)

	handlerCalled := false
	handler := m.CORS()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	}))

	// OPTIONS preflight from disallowed origin
	req := httptest.NewRequest("OPTIONS", "/", nil)
	req.Header.Set("Origin", "https://not-allowed.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// go-chi/cors returns 200 but without CORS headers for disallowed origins
	// The browser will block based on missing CORS headers
	allowOrigin := w.Header().Get("Access-Control-Allow-Origin")
	if allowOrigin != "" {
		t.Errorf("Access-Control-Allow-Origin should not be set for disallowed origin, got %q", allowOrigin)
	}

	// Handler should not be called for OPTIONS
	if handlerCalled {
		t.Error("Handler should not be called for OPTIONS preflight")
	}
}

func TestChiMiddleware_CORS_NoOriginHeader(t *testing.T) {
	config := &ChiMiddlewareConfig{
		CORSAllowedOrigins: []string{"https://allowed.com"},
		CORSAllowedMethods: []string{"GET"},
		CORSAllowedHeaders: []string{"Content-Type"},
		CORSMaxAge:         86400,
	}
	m := NewChiMiddleware(config)

	handlerCalled := false
	handler := m.CORS()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	}))

	// Request without Origin header (same-origin request)
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// Should allow same-origin requests
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if !handlerCalled {
		t.Error("Handler should be called for same-origin requests")
	}
}

// =====================================================
// Rate Limiting Middleware Tests
// =====================================================

func TestChiMiddleware_RateLimit_Disabled(t *testing.T) {
	config := &ChiMiddlewareConfig{
		RateLimitDisabled: true,
		RateLimitRequests: 3,
		RateLimitWindow:   time.Second,
	}
	m := NewChiMiddleware(config)

	callCount := 0
	handler := m.RateLimit()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
	}))

	// Should allow unlimited requests when disabled
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Request %d: status = %d, want %d", i, w.Code, http.StatusOK)
		}
	}

	if callCount != 10 {
		t.Errorf("callCount = %d, want 10", callCount)
	}
}

func TestChiMiddleware_RateLimit_Enabled(t *testing.T) {
	config := &ChiMiddlewareConfig{
		RateLimitDisabled: false,
		RateLimitRequests: 3,
		RateLimitWindow:   time.Minute, // Use a longer window for test stability
	}
	m := NewChiMiddleware(config)

	handler := m.RateLimit()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	successCount := 0
	limitedCount := 0

	// Make more requests than the limit allows
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code == http.StatusOK {
			successCount++
		} else if w.Code == http.StatusTooManyRequests {
			limitedCount++
		}
	}

	// First 3 requests should succeed
	if successCount != 3 {
		t.Errorf("successCount = %d, want 3", successCount)
	}

	// Remaining requests should be rate limited
	if limitedCount != 2 {
		t.Errorf("limitedCount = %d, want 2", limitedCount)
	}
}

func TestChiMiddleware_RateLimit_DifferentIPs(t *testing.T) {
	config := &ChiMiddlewareConfig{
		RateLimitDisabled: false,
		RateLimitRequests: 2,
		RateLimitWindow:   time.Minute,
	}
	m := NewChiMiddleware(config)

	handler := m.RateLimit()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Different IPs should have separate rate limits
	ips := []string{"192.168.1.1:12345", "192.168.1.2:12345", "192.168.1.3:12345"}

	for _, ip := range ips {
		// Each IP should be able to make 2 requests
		for i := 0; i < 2; i++ {
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = ip
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("IP %s request %d: status = %d, want %d", ip, i, w.Code, http.StatusOK)
			}
		}
	}
}

func TestChiMiddleware_RateLimitByIP(t *testing.T) {
	config := &ChiMiddlewareConfig{
		RateLimitDisabled: false,
		RateLimitRequests: 2,
		RateLimitWindow:   time.Minute,
	}
	m := NewChiMiddleware(config)

	handler := m.RateLimitByIP()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// First 2 requests should succeed
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "192.168.1.100:12345"
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Request %d: status = %d, want %d", i, w.Code, http.StatusOK)
		}
	}

	// 3rd request should be rate limited
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.100:12345"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("3rd request: status = %d, want %d", w.Code, http.StatusTooManyRequests)
	}
}

func TestChiMiddleware_RateLimitByIP_Disabled(t *testing.T) {
	config := &ChiMiddlewareConfig{
		RateLimitDisabled: true,
	}
	m := NewChiMiddleware(config)

	handler := m.RateLimitByIP()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// All requests should succeed when disabled
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "192.168.1.100:12345"
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Request %d: status = %d, want %d", i, w.Code, http.StatusOK)
		}
	}
}

func TestChiMiddleware_RateLimitByRealIP_Disabled(t *testing.T) {
	config := &ChiMiddlewareConfig{
		RateLimitDisabled: true,
	}
	m := NewChiMiddleware(config)

	handler := m.RateLimitByRealIP()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// All requests should succeed when disabled
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "192.168.1.100:12345"
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Request %d: status = %d, want %d", i, w.Code, http.StatusOK)
		}
	}
}

// =====================================================
// Integration Tests
// =====================================================

func TestChiMiddleware_CORSAndRateLimit_Combined(t *testing.T) {
	config := &ChiMiddlewareConfig{
		CORSAllowedOrigins: []string{"*"},
		CORSAllowedMethods: []string{"GET"},
		CORSAllowedHeaders: []string{"Content-Type"},
		CORSMaxAge:         86400,
		RateLimitRequests:  2,
		RateLimitWindow:    time.Minute,
		RateLimitDisabled:  false,
	}
	m := NewChiMiddleware(config)

	// Chain CORS and RateLimit middleware
	handler := m.CORS()(m.RateLimit()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})))

	// First request - should succeed with CORS headers
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "https://example.com")
	req.RemoteAddr = "10.0.0.1:12345"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("First request: status = %d, want %d", w.Code, http.StatusOK)
	}
	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("First request should have CORS headers")
	}

	// Second request - should also succeed
	req = httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "https://example.com")
	req.RemoteAddr = "10.0.0.1:12345"
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Second request: status = %d, want %d", w.Code, http.StatusOK)
	}

	// Third request - should be rate limited
	req = httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "https://example.com")
	req.RemoteAddr = "10.0.0.1:12345"
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("Third request: status = %d, want %d", w.Code, http.StatusTooManyRequests)
	}
}

func TestDefaultChiMiddlewareConfig(t *testing.T) {
	config := DefaultChiMiddlewareConfig()

	// Verify defaults are secure (empty CORS - requires explicit configuration)
	if len(config.CORSAllowedOrigins) != 0 {
		t.Errorf("CORSAllowedOrigins = %v, want []", config.CORSAllowedOrigins)
	}

	expectedMethods := []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	if len(config.CORSAllowedMethods) != len(expectedMethods) {
		t.Errorf("CORSAllowedMethods length = %d, want %d", len(config.CORSAllowedMethods), len(expectedMethods))
	}

	expectedHeaders := []string{"Content-Type", "Authorization", "X-Plex-Token"}
	if len(config.CORSAllowedHeaders) != len(expectedHeaders) {
		t.Errorf("CORSAllowedHeaders length = %d, want %d", len(config.CORSAllowedHeaders), len(expectedHeaders))
	}

	if config.CORSMaxAge != 86400 {
		t.Errorf("CORSMaxAge = %d, want 86400", config.CORSMaxAge)
	}

	if config.RateLimitRequests != 100 {
		t.Errorf("RateLimitRequests = %d, want 100", config.RateLimitRequests)
	}

	if config.RateLimitWindow != time.Minute {
		t.Errorf("RateLimitWindow = %v, want 1m", config.RateLimitWindow)
	}

	if config.RateLimitDisabled {
		t.Error("RateLimitDisabled should be false by default")
	}
}

// =====================================================
// RBAC Phase 3: Authorization Middleware Tests
// =====================================================

func TestRequireAdminMiddleware_NotAuthenticated(t *testing.T) {
	handler := RequireAdminMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/admin/endpoint", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestRequireAdminMiddleware_NotAdmin(t *testing.T) {
	handler := RequireAdminMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Create request with viewer role in context
	req := httptest.NewRequest(http.MethodGet, "/admin/endpoint", nil)
	req = addAuthSubjectToRequest(req, "viewer-user", "viewer", false, false)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestRequireAdminMiddleware_Admin(t *testing.T) {
	handler := RequireAdminMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Create request with admin role in context
	req := httptest.NewRequest(http.MethodGet, "/admin/endpoint", nil)
	req = addAuthSubjectToRequest(req, "admin-user", "admin", true, true)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestRequireEditorMiddleware_NotAuthenticated(t *testing.T) {
	handler := RequireEditorMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/editor/endpoint", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestRequireEditorMiddleware_Viewer(t *testing.T) {
	handler := RequireEditorMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Create request with viewer role
	req := httptest.NewRequest(http.MethodGet, "/editor/endpoint", nil)
	req = addAuthSubjectToRequest(req, "viewer-user", "viewer", false, false)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestRequireEditorMiddleware_Editor(t *testing.T) {
	handler := RequireEditorMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Create request with editor role
	req := httptest.NewRequest(http.MethodGet, "/editor/endpoint", nil)
	req = addAuthSubjectToRequest(req, "editor-user", "editor", false, true)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestRequireEditorMiddleware_Admin(t *testing.T) {
	handler := RequireEditorMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Admin should also pass editor middleware
	req := httptest.NewRequest(http.MethodGet, "/editor/endpoint", nil)
	req = addAuthSubjectToRequest(req, "admin-user", "admin", true, true)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestRequireAuthMiddleware_NotAuthenticated(t *testing.T) {
	handler := RequireAuthMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/protected/endpoint", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestRequireAuthMiddleware_Authenticated(t *testing.T) {
	handler := RequireAuthMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Any authenticated user should pass
	req := httptest.NewRequest(http.MethodGet, "/protected/endpoint", nil)
	req = addAuthSubjectToRequest(req, "any-user", "viewer", false, false)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

// addAuthSubjectToRequest adds an AuthSubject to the request context for testing.
// This helper creates a properly configured request with authentication context.
func addAuthSubjectToRequest(r *http.Request, userID, role string, isAdmin, isEditor bool) *http.Request {
	subject := &auth.AuthSubject{
		ID:       userID,
		Username: userID,
		Roles:    []string{role},
	}
	ctx := context.WithValue(r.Context(), auth.AuthSubjectContextKey, subject)
	return r.WithContext(ctx)
}
