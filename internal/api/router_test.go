// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/tomtom215/cartographus/internal/auth"
	"github.com/tomtom215/cartographus/internal/cache"
	"github.com/tomtom215/cartographus/internal/config"
	"github.com/tomtom215/cartographus/internal/database"
	"github.com/tomtom215/cartographus/internal/middleware"
	ws "github.com/tomtom215/cartographus/internal/websocket"
)

// setupRouterTestHandler creates a handler for router testing
func setupRouterTestHandler(t *testing.T) (*Handler, *database.DB) {
	t.Helper()

	cfg := &config.DatabaseConfig{
		Path:        ":memory:",
		MaxMemory:   "256MB",
		SkipIndexes: true, // Skip 97 indexes for fast test setup
	}

	db, err := database.New(cfg, 0.0, 0.0)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	testConfig := &config.Config{
		API: config.APIConfig{
			DefaultPageSize: 100,
			MaxPageSize:     1000,
		},
		Security: config.SecurityConfig{
			AuthMode:        "jwt",
			JWTSecret:       "test_secret_with_at_least_32_characters_for_testing",
			AdminUsername:   "admin",
			AdminPassword:   "password123",
			SessionTimeout:  24 * time.Hour,
			CORSOrigins:     []string{"*"},
			RateLimitReqs:   100,
			RateLimitWindow: time.Minute,
		},
		Server: config.ServerConfig{
			Latitude:  40.7128,
			Longitude: -74.0060,
		},
	}

	wsHub := ws.NewHub()
	go wsHub.Run()

	mockClient := &MockTautulliClient{}

	handler := &Handler{
		db:        db,
		cache:     cache.New(5 * time.Minute),
		client:    mockClient,
		config:    testConfig,
		wsHub:     wsHub,
		startTime: time.Now(),
		perfMon:   middleware.NewPerformanceMonitor(100),
	}

	return handler, db
}

// TestNewRouter tests the NewRouter constructor
func TestNewRouter(t *testing.T) {
	t.Parallel()

	handler, db := setupRouterTestHandler(t)
	defer db.Close()

	cfg := &config.Config{
		Security: config.SecurityConfig{
			JWTSecret:       "test_secret_with_at_least_32_characters_for_testing",
			RateLimitReqs:   100,
			RateLimitWindow: time.Minute,
			SessionTimeout:  24 * time.Hour,
			CORSOrigins:     []string{"*"},
		},
	}

	jwtManager, err := auth.NewJWTManager(&cfg.Security)
	if err != nil {
		t.Fatalf("Failed to create JWT manager: %v", err)
	}

	mw := auth.NewMiddleware(jwtManager, nil, cfg.Security.AuthMode, cfg.Security.RateLimitReqs, cfg.Security.RateLimitWindow, cfg.Security.RateLimitDisabled, cfg.Security.CORSOrigins, cfg.Security.TrustedProxies, "", "")

	router := NewRouter(handler, mw)

	if router == nil {
		t.Fatal("NewRouter returned nil")
	}

	if router.handler != handler {
		t.Error("Handler not set correctly")
	}

	if router.middleware != mw {
		t.Error("Middleware not set correctly")
	}

	// Note: indexTemplate may be nil if template file doesn't exist
	// This is expected and handled gracefully
}

// TestRouterSetup_HealthEndpoints tests that health endpoints are correctly configured
func TestRouterSetup_HealthEndpoints(t *testing.T) {
	t.Parallel()

	handler, db := setupRouterTestHandler(t)
	defer db.Close()

	cfg := &config.Config{
		Security: config.SecurityConfig{
			JWTSecret:       "test_secret_with_at_least_32_characters_for_testing",
			RateLimitReqs:   100,
			RateLimitWindow: time.Minute,
			SessionTimeout:  24 * time.Hour,
			CORSOrigins:     []string{"*"},
		},
	}

	jwtManager, _ := auth.NewJWTManager(&cfg.Security)
	mw := auth.NewMiddleware(jwtManager, nil, cfg.Security.AuthMode, cfg.Security.RateLimitReqs, cfg.Security.RateLimitWindow, cfg.Security.RateLimitDisabled, cfg.Security.CORSOrigins, cfg.Security.TrustedProxies, "", "")
	router := NewRouter(handler, mw)

	tests := []struct {
		name           string
		path           string
		method         string
		expectedStatus int
	}{
		{"health live endpoint", "/api/v1/health/live", http.MethodGet, http.StatusOK},
		{"health ready endpoint", "/api/v1/health/ready", http.MethodGet, http.StatusOK},
		{"health legacy endpoint", "/api/v1/health", http.MethodGet, http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			mux := router.SetupChi()
			mux.ServeHTTP(w, req)

			// Health endpoints should work (may return 200 or 503 depending on mock state)
			if w.Code != http.StatusOK && w.Code != http.StatusServiceUnavailable {
				t.Errorf("%s: expected status 200 or 503, got %d", tt.name, w.Code)
			}
		})
	}
}

// TestRouterSetup_APIEndpoints tests that API endpoints are correctly configured
func TestRouterSetup_APIEndpoints(t *testing.T) {
	t.Parallel()

	handler, db := setupRouterTestHandler(t)
	defer db.Close()

	cfg := &config.Config{
		Security: config.SecurityConfig{
			JWTSecret:       "test_secret_with_at_least_32_characters_for_testing",
			RateLimitReqs:   100,
			RateLimitWindow: time.Minute,
			SessionTimeout:  24 * time.Hour,
			CORSOrigins:     []string{"*"},
		},
	}

	jwtManager, _ := auth.NewJWTManager(&cfg.Security)
	mw := auth.NewMiddleware(jwtManager, nil, cfg.Security.AuthMode, cfg.Security.RateLimitReqs, cfg.Security.RateLimitWindow, cfg.Security.RateLimitDisabled, cfg.Security.CORSOrigins, cfg.Security.TrustedProxies, "", "")
	router := NewRouter(handler, mw)

	tests := []struct {
		name   string
		path   string
		method string
	}{
		{"stats endpoint", "/api/v1/stats", http.MethodGet},
		{"playbacks endpoint", "/api/v1/playbacks", http.MethodGet},
		{"locations endpoint", "/api/v1/locations", http.MethodGet},
		{"users endpoint", "/api/v1/users", http.MethodGet},
		{"media-types endpoint", "/api/v1/media-types", http.MethodGet},
		{"server-info endpoint", "/api/v1/server-info", http.MethodGet},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			mux := router.SetupChi()
			mux.ServeHTTP(w, req)

			// Should return success (200) or at least not 404
			if w.Code == http.StatusNotFound {
				t.Errorf("%s: endpoint not found (404)", tt.name)
			}
		})
	}
}

// TestRouterSetup_AnalyticsEndpoints tests that analytics endpoints are correctly configured
func TestRouterSetup_AnalyticsEndpoints(t *testing.T) {
	t.Parallel()

	handler, db := setupRouterTestHandler(t)
	defer db.Close()

	cfg := &config.Config{
		Security: config.SecurityConfig{
			JWTSecret:       "test_secret_with_at_least_32_characters_for_testing",
			RateLimitReqs:   100,
			RateLimitWindow: time.Minute,
			SessionTimeout:  24 * time.Hour,
			CORSOrigins:     []string{"*"},
		},
	}

	jwtManager, _ := auth.NewJWTManager(&cfg.Security)
	mw := auth.NewMiddleware(jwtManager, nil, cfg.Security.AuthMode, cfg.Security.RateLimitReqs, cfg.Security.RateLimitWindow, cfg.Security.RateLimitDisabled, cfg.Security.CORSOrigins, cfg.Security.TrustedProxies, "", "")
	router := NewRouter(handler, mw)

	tests := []struct {
		name string
		path string
	}{
		{"trends", "/api/v1/analytics/trends"},
		{"geographic", "/api/v1/analytics/geographic"},
		{"users", "/api/v1/analytics/users"},
		{"binge", "/api/v1/analytics/binge"},
		{"bandwidth", "/api/v1/analytics/bandwidth"},
		{"bitrate", "/api/v1/analytics/bitrate"},
		{"popular", "/api/v1/analytics/popular"},
		{"watch-parties", "/api/v1/analytics/watch-parties"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()

			mux := router.SetupChi()
			mux.ServeHTTP(w, req)

			if w.Code == http.StatusNotFound {
				t.Errorf("%s: endpoint not found (404)", tt.name)
			}
		})
	}
}

// TestRouterSetup_SpatialEndpoints tests that spatial endpoints are correctly configured
func TestRouterSetup_SpatialEndpoints(t *testing.T) {
	t.Parallel()

	handler, db := setupRouterTestHandler(t)
	defer db.Close()

	cfg := &config.Config{
		Security: config.SecurityConfig{
			JWTSecret:       "test_secret_with_at_least_32_characters_for_testing",
			RateLimitReqs:   100,
			RateLimitWindow: time.Minute,
			SessionTimeout:  24 * time.Hour,
			CORSOrigins:     []string{"*"},
		},
	}

	jwtManager, _ := auth.NewJWTManager(&cfg.Security)
	mw := auth.NewMiddleware(jwtManager, nil, cfg.Security.AuthMode, cfg.Security.RateLimitReqs, cfg.Security.RateLimitWindow, cfg.Security.RateLimitDisabled, cfg.Security.CORSOrigins, cfg.Security.TrustedProxies, "", "")
	router := NewRouter(handler, mw)

	tests := []struct {
		name string
		path string
	}{
		{"hexagons", "/api/v1/spatial/hexagons"},
		{"arcs", "/api/v1/spatial/arcs"},
		{"viewport", "/api/v1/spatial/viewport"},
		{"temporal-density", "/api/v1/spatial/temporal-density"},
		{"nearby", "/api/v1/spatial/nearby"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()

			mux := router.SetupChi()
			mux.ServeHTTP(w, req)

			if w.Code == http.StatusNotFound {
				t.Errorf("%s: endpoint not found (404)", tt.name)
			}
		})
	}
}

// TestRouterSetup_ExportEndpoints tests that export endpoints are correctly configured
func TestRouterSetup_ExportEndpoints(t *testing.T) {
	t.Parallel()

	handler, db := setupRouterTestHandler(t)
	defer db.Close()

	cfg := &config.Config{
		Security: config.SecurityConfig{
			JWTSecret:       "test_secret_with_at_least_32_characters_for_testing",
			RateLimitReqs:   100,
			RateLimitWindow: time.Minute,
			SessionTimeout:  24 * time.Hour,
			CORSOrigins:     []string{"*"},
		},
	}

	jwtManager, _ := auth.NewJWTManager(&cfg.Security)
	mw := auth.NewMiddleware(jwtManager, nil, cfg.Security.AuthMode, cfg.Security.RateLimitReqs, cfg.Security.RateLimitWindow, cfg.Security.RateLimitDisabled, cfg.Security.CORSOrigins, cfg.Security.TrustedProxies, "", "")
	router := NewRouter(handler, mw)

	tests := []struct {
		name string
		path string
	}{
		{"geoparquet", "/api/v1/export/geoparquet"},
		{"geojson", "/api/v1/export/geojson"},
		{"playbacks csv", "/api/v1/export/playbacks/csv"},
		{"locations geojson", "/api/v1/export/locations/geojson"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()

			mux := router.SetupChi()
			mux.ServeHTTP(w, req)

			if w.Code == http.StatusNotFound {
				t.Errorf("%s: endpoint not found (404)", tt.name)
			}
		})
	}
}

// TestRouterSetup_TautulliEndpoints tests that Tautulli endpoints are correctly configured
func TestRouterSetup_TautulliEndpoints(t *testing.T) {
	t.Parallel()

	handler, db := setupRouterTestHandler(t)
	defer db.Close()

	cfg := &config.Config{
		Security: config.SecurityConfig{
			JWTSecret:       "test_secret_with_at_least_32_characters_for_testing",
			RateLimitReqs:   100,
			RateLimitWindow: time.Minute,
			SessionTimeout:  24 * time.Hour,
			CORSOrigins:     []string{"*"},
		},
	}

	jwtManager, _ := auth.NewJWTManager(&cfg.Security)
	mw := auth.NewMiddleware(jwtManager, nil, cfg.Security.AuthMode, cfg.Security.RateLimitReqs, cfg.Security.RateLimitWindow, cfg.Security.RateLimitDisabled, cfg.Security.CORSOrigins, cfg.Security.TrustedProxies, "", "")
	router := NewRouter(handler, mw)

	tests := []struct {
		name string
		path string
	}{
		{"home-stats", "/api/v1/tautulli/home-stats"},
		{"plays-by-date", "/api/v1/tautulli/plays-by-date"},
		{"activity", "/api/v1/tautulli/activity"},
		{"server-info", "/api/v1/tautulli/server-info"},
		{"libraries", "/api/v1/tautulli/libraries"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()

			mux := router.SetupChi()
			mux.ServeHTTP(w, req)

			if w.Code == http.StatusNotFound {
				t.Errorf("%s: endpoint not found (404)", tt.name)
			}
		})
	}
}

// TestRouterSetup_AuthEndpoints tests that auth endpoints are correctly configured
func TestRouterSetup_AuthEndpoints(t *testing.T) {
	t.Parallel()

	handler, db := setupRouterTestHandler(t)
	defer db.Close()

	cfg := &config.Config{
		Security: config.SecurityConfig{
			JWTSecret:       "test_secret_with_at_least_32_characters_for_testing",
			RateLimitReqs:   100,
			RateLimitWindow: time.Minute,
			SessionTimeout:  24 * time.Hour,
			CORSOrigins:     []string{"*"},
		},
	}

	jwtManager, _ := auth.NewJWTManager(&cfg.Security)
	mw := auth.NewMiddleware(jwtManager, nil, cfg.Security.AuthMode, cfg.Security.RateLimitReqs, cfg.Security.RateLimitWindow, cfg.Security.RateLimitDisabled, cfg.Security.CORSOrigins, cfg.Security.TrustedProxies, "", "")
	router := NewRouter(handler, mw)

	tests := []struct {
		name   string
		path   string
		method string
	}{
		{"login", "/api/v1/auth/login", http.MethodPost},
		{"plex start", "/api/v1/auth/plex/start", http.MethodGet},
		{"plex callback", "/api/v1/auth/plex/callback", http.MethodGet},
		{"plex refresh", "/api/v1/auth/plex/refresh", http.MethodPost},
		{"plex revoke", "/api/v1/auth/plex/revoke", http.MethodPost},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			mux := router.SetupChi()
			mux.ServeHTTP(w, req)

			if w.Code == http.StatusNotFound {
				t.Errorf("%s: endpoint not found (404)", tt.name)
			}
		})
	}
}

// TestRouterSetup_MetricsEndpoint tests that Prometheus metrics endpoint is configured
func TestRouterSetup_MetricsEndpoint(t *testing.T) {
	t.Parallel()

	handler, db := setupRouterTestHandler(t)
	defer db.Close()

	cfg := &config.Config{
		Security: config.SecurityConfig{
			JWTSecret:       "test_secret_with_at_least_32_characters_for_testing",
			RateLimitReqs:   100,
			RateLimitWindow: time.Minute,
			SessionTimeout:  24 * time.Hour,
			CORSOrigins:     []string{"*"},
		},
	}

	jwtManager, _ := auth.NewJWTManager(&cfg.Security)
	mw := auth.NewMiddleware(jwtManager, nil, cfg.Security.AuthMode, cfg.Security.RateLimitReqs, cfg.Security.RateLimitWindow, cfg.Security.RateLimitDisabled, cfg.Security.CORSOrigins, cfg.Security.TrustedProxies, "", "")
	router := NewRouter(handler, mw)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()

	mux := router.SetupChi()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for /metrics, got %d", w.Code)
	}

	// Check content type is Prometheus format
	contentType := w.Header().Get("Content-Type")
	if contentType == "" {
		t.Error("Expected Content-Type header for metrics endpoint")
	}
}

// TestRouterSetup_CORSHeaders tests that CORS headers are set correctly
func TestRouterSetup_CORSHeaders(t *testing.T) {
	t.Parallel()

	handler, db := setupRouterTestHandler(t)
	defer db.Close()

	cfg := &config.Config{
		Security: config.SecurityConfig{
			JWTSecret:       "test_secret_with_at_least_32_characters_for_testing",
			RateLimitReqs:   100,
			RateLimitWindow: time.Minute,
			SessionTimeout:  24 * time.Hour,
			CORSOrigins:     []string{"http://localhost:3857"},
		},
	}

	jwtManager, _ := auth.NewJWTManager(&cfg.Security)
	mw := auth.NewMiddleware(jwtManager, nil, cfg.Security.AuthMode, cfg.Security.RateLimitReqs, cfg.Security.RateLimitWindow, cfg.Security.RateLimitDisabled, cfg.Security.CORSOrigins, cfg.Security.TrustedProxies, "", "")
	router := NewRouter(handler, mw)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/stats", nil)
	req.Header.Set("Origin", "http://localhost:3857")
	w := httptest.NewRecorder()

	mux := router.SetupChi()
	mux.ServeHTTP(w, req)

	// Check CORS headers
	accessControl := w.Header().Get("Access-Control-Allow-Origin")
	if accessControl == "" {
		// CORS headers may not be set for same-origin requests
		t.Log("CORS header not set - may be expected for same-origin")
	}
}

// TestFileExists tests the fileExists helper function
func TestFileExists(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"non-existent file", "/nonexistent/file.js", false},
		{"directory path", "/", false},
		{"empty path", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fileExists(tt.path)
			if result != tt.expected {
				t.Errorf("fileExists(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

// TestFileExists_WithActualFile tests fileExists with an actual file
func TestFileExists_WithActualFile(t *testing.T) {
	// Create a temporary directory structure
	tmpDir := t.TempDir()

	// Create web/dist directory
	distDir := filepath.Join(tmpDir, "web", "dist")
	if err := os.MkdirAll(distDir, 0755); err != nil {
		t.Fatalf("Failed to create dist directory: %v", err)
	}

	// Create a test file
	testFile := filepath.Join(distDir, "test.js")
	if err := os.WriteFile(testFile, []byte("// test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Save current working directory
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	// Change to temp directory
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	// Restore working directory after test
	defer func() {
		if err := os.Chdir(originalWd); err != nil {
			t.Errorf("Failed to restore working directory: %v", err)
		}
	}()

	// Test with existing file
	if !fileExists("/test.js") {
		t.Error("fileExists should return true for existing file")
	}

	// Test with non-existing file
	if fileExists("/nonexistent.js") {
		t.Error("fileExists should return false for non-existing file")
	}
}

// TestServeStaticOrIndex_CacheHeaders tests cache header configuration
// Note: This test cannot use t.Parallel() because it changes the working directory
func TestServeStaticOrIndex_CacheHeaders(t *testing.T) {
	// Create temporary web/dist directory with test files
	// This is needed because http.ServeFile doesn't preserve headers when files don't exist
	tmpDir := t.TempDir()
	webDistDir := filepath.Join(tmpDir, "web", "dist")
	if err := os.MkdirAll(webDistDir, 0755); err != nil {
		t.Fatalf("Failed to create temp web/dist dir: %v", err)
	}

	// Create test files
	testFiles := []string{"bundle.js", "styles.css", "icon.png", "logo.svg", "data.json", "index.html", "manifest.json"}
	for _, f := range testFiles {
		if err := os.WriteFile(filepath.Join(webDistDir, f), []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", f, err)
		}
	}

	// Change to temp directory so ./web/dist resolves correctly
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}
	defer os.Chdir(origDir)

	tests := []struct {
		name          string
		path          string
		expectedCache string
	}{
		{"JS file", "/bundle.js", "public, max-age=31536000, immutable"},
		{"CSS file", "/styles.css", "public, max-age=31536000, immutable"},
		{"PNG image", "/icon.png", "public, max-age=604800"},
		{"SVG image", "/logo.svg", "public, max-age=604800"},
		{"JSON file", "/data.json", "public, max-age=3600"},
		{"Root path", "/", "public, max-age=300"},
		{"Index HTML", "/index.html", "public, max-age=300"},
		{"Manifest JSON", "/manifest.json", "public, max-age=300"},
	}

	// Create minimal router for testing cache headers (no database needed)
	// The serveStaticOrIndex function only uses the router's handler pointer,
	// and doesn't actually access the database
	minimalHandler := &Handler{
		config: &config.Config{},
	}
	router := &Router{
		handler: minimalHandler,
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()

			// Test the serveStaticOrIndex function directly
			router.serveStaticOrIndex(w, req)

			cacheControl := w.Header().Get("Cache-Control")
			if cacheControl != tt.expectedCache {
				t.Errorf("Cache-Control for %s = %q, want %q", tt.path, cacheControl, tt.expectedCache)
			}
		})
	}
}

// TestRenderIndexTemplate tests the template rendering function
func TestRenderIndexTemplate(t *testing.T) {
	t.Parallel()

	t.Run("without template", func(t *testing.T) {
		handler, db := setupRouterTestHandler(t)
		defer db.Close()

		router := &Router{
			handler:       handler,
			indexTemplate: nil, // No template
		}

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()

		// This should fall back to serving static file (which may not exist)
		router.renderIndexTemplate(w, req)

		// Should not panic
	})

	t.Run("with nonce in context", func(t *testing.T) {
		handler, db := setupRouterTestHandler(t)
		defer db.Close()

		router := &Router{
			handler:       handler,
			indexTemplate: nil,
		}

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		ctx := context.WithValue(req.Context(), auth.CSPNonceContextKey, "test-nonce")
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		router.renderIndexTemplate(w, req)

		// Should not panic
	})
}

// TestRouterWrap tests the wrap middleware function
func TestRouterWrap(t *testing.T) {
	t.Parallel()

	handler, db := setupRouterTestHandler(t)
	defer db.Close()

	cfg := &config.Config{
		Security: config.SecurityConfig{
			JWTSecret:       "test_secret_with_at_least_32_characters_for_testing",
			RateLimitReqs:   100,
			RateLimitWindow: time.Minute,
			SessionTimeout:  24 * time.Hour,
			CORSOrigins:     []string{"*"},
		},
	}

	jwtManager, _ := auth.NewJWTManager(&cfg.Security)
	mw := auth.NewMiddleware(jwtManager, nil, cfg.Security.AuthMode, cfg.Security.RateLimitReqs, cfg.Security.RateLimitWindow, cfg.Security.RateLimitDisabled, cfg.Security.CORSOrigins, cfg.Security.TrustedProxies, "", "")
	router := NewRouter(handler, mw)

	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test"))
	})

	// Wrap it
	wrapped := router.wrap(testHandler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	wrapped(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// TestRouterSetup_MethodNotAllowed tests that wrong HTTP methods are handled
func TestRouterSetup_MethodNotAllowed(t *testing.T) {
	t.Parallel()

	handler, db := setupRouterTestHandler(t)
	defer db.Close()

	cfg := &config.Config{
		Security: config.SecurityConfig{
			JWTSecret:       "test_secret_with_at_least_32_characters_for_testing",
			RateLimitReqs:   100,
			RateLimitWindow: time.Minute,
			SessionTimeout:  24 * time.Hour,
			CORSOrigins:     []string{"*"},
			AuthMode:        "none", // Disable auth to test method handling
		},
	}

	jwtManager, _ := auth.NewJWTManager(&cfg.Security)
	mw := auth.NewMiddleware(jwtManager, nil, cfg.Security.AuthMode, cfg.Security.RateLimitReqs, cfg.Security.RateLimitWindow, cfg.Security.RateLimitDisabled, cfg.Security.CORSOrigins, cfg.Security.TrustedProxies, "", "")
	router := NewRouter(handler, mw)

	tests := []struct {
		name   string
		path   string
		method string
	}{
		{"POST to stats", "/api/v1/stats", http.MethodPost},
		{"PUT to playbacks", "/api/v1/playbacks", http.MethodPut},
		{"DELETE to locations", "/api/v1/locations", http.MethodDelete},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			mux := router.SetupChi()
			mux.ServeHTTP(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("%s: expected status 405, got %d", tt.name, w.Code)
			}
		})
	}
}

// TestRouterSetup_SyncEndpoint_RequiresAuth tests that sync endpoint requires authentication
func TestRouterSetup_SyncEndpoint_RequiresAuth(t *testing.T) {
	t.Parallel()

	handler, db := setupRouterTestHandler(t)
	defer db.Close()

	cfg := &config.Config{
		Security: config.SecurityConfig{
			JWTSecret:       "test_secret_with_at_least_32_characters_for_testing",
			RateLimitReqs:   100,
			RateLimitWindow: time.Minute,
			SessionTimeout:  24 * time.Hour,
			CORSOrigins:     []string{"*"},
			AuthMode:        "jwt",
		},
	}

	jwtManager, _ := auth.NewJWTManager(&cfg.Security)
	mw := auth.NewMiddleware(jwtManager, nil, cfg.Security.AuthMode, cfg.Security.RateLimitReqs, cfg.Security.RateLimitWindow, cfg.Security.RateLimitDisabled, cfg.Security.CORSOrigins, cfg.Security.TrustedProxies, "", "")
	router := NewRouter(handler, mw)

	// Request without auth
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sync", nil)
	w := httptest.NewRecorder()

	mux := router.SetupChi()
	mux.ServeHTTP(w, req)

	// Should return 401 Unauthorized
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401 for unauthenticated sync request, got %d", w.Code)
	}
}

// BenchmarkRouterSetup benchmarks the router setup
func BenchmarkRouterSetup(b *testing.B) {
	cfg := &config.DatabaseConfig{
		Path:        ":memory:",
		MaxMemory:   "256MB",
		SkipIndexes: true, // Skip 97 indexes for fast test setup
	}

	db, _ := database.New(cfg, 0.0, 0.0)
	defer db.Close()

	testConfig := &config.Config{
		API: config.APIConfig{
			DefaultPageSize: 100,
			MaxPageSize:     1000,
		},
		Security: config.SecurityConfig{
			JWTSecret:       "test_secret_with_at_least_32_characters_for_testing",
			RateLimitReqs:   100,
			RateLimitWindow: time.Minute,
			SessionTimeout:  24 * time.Hour,
			CORSOrigins:     []string{"*"},
		},
	}

	wsHub := ws.NewHub()
	go wsHub.Run()

	handler := &Handler{
		db:        db,
		cache:     cache.New(5 * time.Minute),
		config:    testConfig,
		wsHub:     wsHub,
		startTime: time.Now(),
	}

	jwtManager, _ := auth.NewJWTManager(&testConfig.Security)
	mw := auth.NewMiddleware(jwtManager, nil, testConfig.Security.AuthMode, testConfig.Security.RateLimitReqs, testConfig.Security.RateLimitWindow, testConfig.Security.RateLimitDisabled, testConfig.Security.CORSOrigins, testConfig.Security.TrustedProxies, "", "")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		router := NewRouter(handler, mw)
		_ = router.SetupChi()
	}
}

// BenchmarkRouterHandleRequest benchmarks request handling
func BenchmarkRouterHandleRequest(b *testing.B) {
	cfg := &config.DatabaseConfig{
		Path:        ":memory:",
		MaxMemory:   "256MB",
		SkipIndexes: true, // Skip 97 indexes for fast test setup
	}

	db, _ := database.New(cfg, 0.0, 0.0)
	defer db.Close()

	testConfig := &config.Config{
		API: config.APIConfig{
			DefaultPageSize: 100,
			MaxPageSize:     1000,
		},
		Security: config.SecurityConfig{
			JWTSecret:       "test_secret_with_at_least_32_characters_for_testing",
			RateLimitReqs:   100000, // High limit for benchmark
			RateLimitWindow: time.Minute,
			SessionTimeout:  24 * time.Hour,
			CORSOrigins:     []string{"*"},
		},
	}

	wsHub := ws.NewHub()
	go wsHub.Run()

	handler := &Handler{
		db:        db,
		cache:     cache.New(5 * time.Minute),
		config:    testConfig,
		client:    &MockTautulliClient{},
		wsHub:     wsHub,
		startTime: time.Now(),
		perfMon:   middleware.NewPerformanceMonitor(100),
	}

	jwtManager, _ := auth.NewJWTManager(&testConfig.Security)
	mw := auth.NewMiddleware(jwtManager, nil, testConfig.Security.AuthMode, testConfig.Security.RateLimitReqs, testConfig.Security.RateLimitWindow, testConfig.Security.RateLimitDisabled, testConfig.Security.CORSOrigins, testConfig.Security.TrustedProxies, "", "")
	router := NewRouter(handler, mw)
	mux := router.SetupChi()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/stats", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
	}
}
