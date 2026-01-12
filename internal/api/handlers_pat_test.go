// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package api provides HTTP handlers for the Cartographus application.
//
// handlers_pat_test.go - Tests for Personal Access Token API handlers.
//
// These tests verify:
//   - Authentication requirements
//   - Input validation
//   - Authorization (users can only access their own tokens)
//   - CRUD operations work correctly
//   - Error handling
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/tomtom215/cartographus/internal/auth"
	"github.com/tomtom215/cartographus/internal/cache"
	"github.com/tomtom215/cartographus/internal/config"
	"github.com/tomtom215/cartographus/internal/database"
	"github.com/tomtom215/cartographus/internal/middleware"
	"github.com/tomtom215/cartographus/internal/models"
	ws "github.com/tomtom215/cartographus/internal/websocket"
)

// setupPATTestHandler creates a handler with database for PAT testing.
func setupPATTestHandler(t *testing.T) (*Handler, *database.DB, func()) {
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
			CORSOrigins: []string{"*"},
		},
	}

	wsHub := ws.NewHub()
	go wsHub.RunWithContext(context.Background())

	handler := &Handler{
		db:        db,
		cache:     cache.New(5 * time.Minute),
		client:    &MockTautulliClient{},
		config:    testConfig,
		wsHub:     wsHub,
		startTime: time.Now(),
		perfMon:   middleware.NewPerformanceMonitor(100),
	}

	cleanup := func() {
		db.Close()
	}

	return handler, db, cleanup
}

// requestWithAuth creates a request with authentication context.
func requestWithAuth(method, path string, body []byte, userID, username string, isAdmin bool) *http.Request {
	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(method, path, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}

	roles := []string{models.RoleViewer}
	if isAdmin {
		roles = []string{models.RoleAdmin}
	}

	subject := &auth.AuthSubject{
		ID:       userID,
		Username: username,
		Roles:    roles,
	}

	ctx := context.WithValue(req.Context(), auth.AuthSubjectContextKey, subject)
	return req.WithContext(ctx)
}

// requestWithChiParam adds chi URL params to a request.
func requestWithChiParam(req *http.Request, key, value string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, value)
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	return req.WithContext(ctx)
}

// TestPATList tests the PAT listing endpoint.
func TestPATList(t *testing.T) {
	handler, db, cleanup := setupPATTestHandler(t)
	defer cleanup()

	t.Run("unauthenticated request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/user/tokens", nil)
		w := httptest.NewRecorder()

		handler.PATList(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
		}
	})

	t.Run("method not allowed", func(t *testing.T) {
		req := requestWithAuth(http.MethodPost, "/api/v1/user/tokens", nil, "user123", "testuser", false)
		w := httptest.NewRecorder()

		handler.PATList(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
		}
	})

	t.Run("empty list for new user", func(t *testing.T) {
		req := requestWithAuth(http.MethodGet, "/api/v1/user/tokens", nil, "user123", "testuser", false)
		w := httptest.NewRecorder()

		handler.PATList(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}

		var resp models.APIResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if resp.Status != "success" {
			t.Errorf("expected status 'success', got %q", resp.Status)
		}

		data, ok := resp.Data.(map[string]interface{})
		if !ok {
			t.Fatalf("expected Data to be map, got %T", resp.Data)
		}

		count := int(data["total_count"].(float64))
		if count != 0 {
			t.Errorf("expected 0 tokens, got %d", count)
		}
	})

	t.Run("list user tokens", func(t *testing.T) {
		ctx := context.Background()

		// Create a token for user123
		token := &models.PersonalAccessToken{
			ID:          "test-token-id",
			UserID:      "user123",
			Username:    "testuser",
			Name:        "Test Token",
			TokenPrefix: "carto_pat_test1234",
			TokenHash:   "$2a$12$testhashhashhashhashhashhashhashhashhashhashhashhash",
			Scopes:      []models.TokenScope{models.ScopeReadAnalytics},
			CreatedAt:   time.Now(),
		}
		if err := db.CreatePAT(ctx, token); err != nil {
			t.Fatalf("failed to create token: %v", err)
		}

		req := requestWithAuth(http.MethodGet, "/api/v1/user/tokens", nil, "user123", "testuser", false)
		w := httptest.NewRecorder()

		handler.PATList(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}

		var resp models.APIResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		data, ok := resp.Data.(map[string]interface{})
		if !ok {
			t.Fatalf("expected Data to be map, got %T", resp.Data)
		}

		count := int(data["total_count"].(float64))
		if count != 1 {
			t.Errorf("expected 1 token, got %d", count)
		}
	})
}

// TestPATCreate tests the PAT creation endpoint.
func TestPATCreate(t *testing.T) {
	handler, _, cleanup := setupPATTestHandler(t)
	defer cleanup()

	t.Run("unauthenticated request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/user/tokens", nil)
		w := httptest.NewRecorder()

		handler.PATCreate(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
		}
	})

	t.Run("method not allowed", func(t *testing.T) {
		req := requestWithAuth(http.MethodGet, "/api/v1/user/tokens", nil, "user123", "testuser", false)
		w := httptest.NewRecorder()

		handler.PATCreate(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		req := requestWithAuth(http.MethodPost, "/api/v1/user/tokens", []byte("{invalid}"), "user123", "testuser", false)
		w := httptest.NewRecorder()

		handler.PATCreate(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("missing name", func(t *testing.T) {
		body := `{"scopes": ["read:analytics"]}`
		req := requestWithAuth(http.MethodPost, "/api/v1/user/tokens", []byte(body), "user123", "testuser", false)
		w := httptest.NewRecorder()

		handler.PATCreate(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
		}

		var resp models.APIResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if resp.Error == nil {
			t.Fatal("expected error in response")
		}
		if !strings.Contains(resp.Error.Message, "name") {
			t.Errorf("expected error message about name, got %q", resp.Error.Message)
		}
	})

	t.Run("missing scopes", func(t *testing.T) {
		body := `{"name": "Test Token"}`
		req := requestWithAuth(http.MethodPost, "/api/v1/user/tokens", []byte(body), "user123", "testuser", false)
		w := httptest.NewRecorder()

		handler.PATCreate(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("non-admin cannot create admin token", func(t *testing.T) {
		body := `{"name": "Admin Token", "scopes": ["admin"]}`
		req := requestWithAuth(http.MethodPost, "/api/v1/user/tokens", []byte(body), "user123", "testuser", false)
		w := httptest.NewRecorder()

		handler.PATCreate(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("expected status %d, got %d", http.StatusForbidden, w.Code)
		}
	})

	t.Run("admin can create admin token", func(t *testing.T) {
		body := `{"name": "Admin Token", "scopes": ["admin"]}`
		req := requestWithAuth(http.MethodPost, "/api/v1/user/tokens", []byte(body), "admin123", "admin", true)
		w := httptest.NewRecorder()

		handler.PATCreate(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("expected status %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
		}
	})

	t.Run("successful creation", func(t *testing.T) {
		body := `{"name": "My API Token", "scopes": ["read:analytics", "read:users"], "description": "For automation"}`
		req := requestWithAuth(http.MethodPost, "/api/v1/user/tokens", []byte(body), "user456", "testuser2", false)
		w := httptest.NewRecorder()

		handler.PATCreate(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("expected status %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
		}

		var resp models.APIResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if resp.Status != "success" {
			t.Errorf("expected status 'success', got %q", resp.Status)
		}

		data, ok := resp.Data.(map[string]interface{})
		if !ok {
			t.Fatalf("expected Data to be map, got %T", resp.Data)
		}

		// Verify plaintext token is returned
		plaintextToken, ok := data["plaintext_token"].(string)
		if !ok || plaintextToken == "" {
			t.Error("expected plaintext_token to be returned")
		}

		if !strings.HasPrefix(plaintextToken, "carto_pat_") {
			t.Errorf("expected token to start with 'carto_pat_', got %q", plaintextToken[:20])
		}
	})

	t.Run("with expiration", func(t *testing.T) {
		body := `{"name": "Expiring Token", "scopes": ["read:analytics"], "expires_in_days": 30}`
		req := requestWithAuth(http.MethodPost, "/api/v1/user/tokens", []byte(body), "user789", "testuser3", false)
		w := httptest.NewRecorder()

		handler.PATCreate(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("expected status %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
		}
	})
}

// TestPATGet tests the PAT get endpoint.
func TestPATGet(t *testing.T) {
	handler, db, cleanup := setupPATTestHandler(t)
	defer cleanup()

	ctx := context.Background()

	// Create a test token
	token := &models.PersonalAccessToken{
		ID:          "get-test-token",
		UserID:      "user123",
		Username:    "testuser",
		Name:        "Get Test Token",
		TokenPrefix: "carto_pat_gettest",
		TokenHash:   "$2a$12$testhashhashhashhashhashhashhashhashhashhashhashhash",
		Scopes:      []models.TokenScope{models.ScopeReadAnalytics},
		CreatedAt:   time.Now(),
	}
	if err := db.CreatePAT(ctx, token); err != nil {
		t.Fatalf("failed to create token: %v", err)
	}

	t.Run("unauthenticated request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/user/tokens/get-test-token", nil)
		req = requestWithChiParam(req, "id", "get-test-token")
		w := httptest.NewRecorder()

		handler.PATGet(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
		}
	})

	t.Run("method not allowed", func(t *testing.T) {
		req := requestWithAuth(http.MethodPost, "/api/v1/user/tokens/get-test-token", nil, "user123", "testuser", false)
		req = requestWithChiParam(req, "id", "get-test-token")
		w := httptest.NewRecorder()

		handler.PATGet(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
		}
	})

	t.Run("missing id", func(t *testing.T) {
		req := requestWithAuth(http.MethodGet, "/api/v1/user/tokens/", nil, "user123", "testuser", false)
		// Don't add chi param
		w := httptest.NewRecorder()

		handler.PATGet(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("token not found", func(t *testing.T) {
		req := requestWithAuth(http.MethodGet, "/api/v1/user/tokens/nonexistent", nil, "user123", "testuser", false)
		req = requestWithChiParam(req, "id", "nonexistent")
		w := httptest.NewRecorder()

		handler.PATGet(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
		}
	})

	t.Run("access denied for other user", func(t *testing.T) {
		req := requestWithAuth(http.MethodGet, "/api/v1/user/tokens/get-test-token", nil, "otheruser", "other", false)
		req = requestWithChiParam(req, "id", "get-test-token")
		w := httptest.NewRecorder()

		handler.PATGet(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("expected status %d, got %d", http.StatusForbidden, w.Code)
		}
	})

	t.Run("successful get", func(t *testing.T) {
		req := requestWithAuth(http.MethodGet, "/api/v1/user/tokens/get-test-token", nil, "user123", "testuser", false)
		req = requestWithChiParam(req, "id", "get-test-token")
		w := httptest.NewRecorder()

		handler.PATGet(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
		}

		var resp models.APIResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if resp.Status != "success" {
			t.Errorf("expected status 'success', got %q", resp.Status)
		}
	})
}

// TestPATRevoke tests the PAT revocation endpoint.
func TestPATRevoke(t *testing.T) {
	handler, db, cleanup := setupPATTestHandler(t)
	defer cleanup()

	ctx := context.Background()

	// Create a test token
	token := &models.PersonalAccessToken{
		ID:          "revoke-test-token",
		UserID:      "user123",
		Username:    "testuser",
		Name:        "Revoke Test Token",
		TokenPrefix: "carto_pat_revoke1",
		TokenHash:   "$2a$12$testhashhashhashhashhashhashhashhashhashhashhashhash",
		Scopes:      []models.TokenScope{models.ScopeReadAnalytics},
		CreatedAt:   time.Now(),
	}
	if err := db.CreatePAT(ctx, token); err != nil {
		t.Fatalf("failed to create token: %v", err)
	}

	t.Run("unauthenticated request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/user/tokens/revoke-test-token", nil)
		req = requestWithChiParam(req, "id", "revoke-test-token")
		w := httptest.NewRecorder()

		handler.PATRevoke(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
		}
	})

	t.Run("method not allowed", func(t *testing.T) {
		req := requestWithAuth(http.MethodGet, "/api/v1/user/tokens/revoke-test-token", nil, "user123", "testuser", false)
		req = requestWithChiParam(req, "id", "revoke-test-token")
		w := httptest.NewRecorder()

		handler.PATRevoke(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
		}
	})

	t.Run("access denied for other user", func(t *testing.T) {
		req := requestWithAuth(http.MethodDelete, "/api/v1/user/tokens/revoke-test-token", nil, "otheruser", "other", false)
		req = requestWithChiParam(req, "id", "revoke-test-token")
		w := httptest.NewRecorder()

		handler.PATRevoke(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("expected status %d, got %d", http.StatusForbidden, w.Code)
		}
	})

	t.Run("successful revocation", func(t *testing.T) {
		req := requestWithAuth(http.MethodDelete, "/api/v1/user/tokens/revoke-test-token", nil, "user123", "testuser", false)
		req = requestWithChiParam(req, "id", "revoke-test-token")
		w := httptest.NewRecorder()

		handler.PATRevoke(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
		}

		// Verify token is revoked in database
		revokedToken, err := db.GetPATByID(ctx, "revoke-test-token")
		if err != nil {
			t.Fatalf("failed to get token: %v", err)
		}
		if revokedToken == nil {
			t.Fatal("token should still exist")
		}
		if revokedToken.RevokedAt == nil {
			t.Error("token should be revoked")
		}
	})

	t.Run("revocation with reason", func(t *testing.T) {
		// Create another token
		token2 := &models.PersonalAccessToken{
			ID:          "revoke-test-token-2",
			UserID:      "user123",
			Username:    "testuser",
			Name:        "Revoke Test Token 2",
			TokenPrefix: "carto_pat_revoke2",
			TokenHash:   "$2a$12$testhashhashhashhashhashhashhashhashhashhashhashhash",
			Scopes:      []models.TokenScope{models.ScopeReadAnalytics},
			CreatedAt:   time.Now(),
		}
		if err := db.CreatePAT(ctx, token2); err != nil {
			t.Fatalf("failed to create token: %v", err)
		}

		body := `{"reason": "Security concern"}`
		req := requestWithAuth(http.MethodDelete, "/api/v1/user/tokens/revoke-test-token-2", []byte(body), "user123", "testuser", false)
		req = requestWithChiParam(req, "id", "revoke-test-token-2")
		w := httptest.NewRecorder()

		handler.PATRevoke(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
		}
	})
}

// TestPATRegenerate tests the PAT regeneration endpoint.
func TestPATRegenerate(t *testing.T) {
	handler, db, cleanup := setupPATTestHandler(t)
	defer cleanup()

	ctx := context.Background()

	// Create a test token
	token := &models.PersonalAccessToken{
		ID:          "regen-test-token",
		UserID:      "user123",
		Username:    "testuser",
		Name:        "Regen Test Token",
		TokenPrefix: "carto_pat_regen12",
		TokenHash:   "$2a$12$testhashhashhashhashhashhashhashhashhashhashhashhash",
		Scopes:      []models.TokenScope{models.ScopeReadAnalytics},
		CreatedAt:   time.Now(),
	}
	if err := db.CreatePAT(ctx, token); err != nil {
		t.Fatalf("failed to create token: %v", err)
	}

	t.Run("unauthenticated request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/user/tokens/regen-test-token/regenerate", nil)
		req = requestWithChiParam(req, "id", "regen-test-token")
		w := httptest.NewRecorder()

		handler.PATRegenerate(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
		}
	})

	t.Run("method not allowed", func(t *testing.T) {
		req := requestWithAuth(http.MethodGet, "/api/v1/user/tokens/regen-test-token/regenerate", nil, "user123", "testuser", false)
		req = requestWithChiParam(req, "id", "regen-test-token")
		w := httptest.NewRecorder()

		handler.PATRegenerate(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
		}
	})

	t.Run("access denied for other user", func(t *testing.T) {
		req := requestWithAuth(http.MethodPost, "/api/v1/user/tokens/regen-test-token/regenerate", nil, "otheruser", "other", false)
		req = requestWithChiParam(req, "id", "regen-test-token")
		w := httptest.NewRecorder()

		handler.PATRegenerate(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("expected status %d, got %d", http.StatusForbidden, w.Code)
		}
	})

	t.Run("token not found", func(t *testing.T) {
		req := requestWithAuth(http.MethodPost, "/api/v1/user/tokens/nonexistent/regenerate", nil, "user123", "testuser", false)
		req = requestWithChiParam(req, "id", "nonexistent")
		w := httptest.NewRecorder()

		handler.PATRegenerate(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
		}
	})

	t.Run("successful regeneration", func(t *testing.T) {
		req := requestWithAuth(http.MethodPost, "/api/v1/user/tokens/regen-test-token/regenerate", nil, "user123", "testuser", false)
		req = requestWithChiParam(req, "id", "regen-test-token")
		w := httptest.NewRecorder()

		handler.PATRegenerate(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
		}

		var resp models.APIResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if resp.Status != "success" {
			t.Errorf("expected status 'success', got %q", resp.Status)
		}

		// Verify new plaintext token is returned
		data, ok := resp.Data.(map[string]interface{})
		if !ok {
			t.Fatalf("expected Data to be map, got %T", resp.Data)
		}

		plaintextToken, ok := data["plaintext_token"].(string)
		if !ok || plaintextToken == "" {
			t.Error("expected plaintext_token to be returned")
		}
	})
}

// TestPATStats tests the PAT statistics endpoint.
func TestPATStats(t *testing.T) {
	handler, db, cleanup := setupPATTestHandler(t)
	defer cleanup()

	ctx := context.Background()

	// Create some test tokens
	for i, name := range []string{"Token1", "Token2", "Token3"} {
		token := &models.PersonalAccessToken{
			ID:          "stats-token-" + string(rune('a'+i)),
			UserID:      "user123",
			Username:    "testuser",
			Name:        name,
			TokenPrefix: "carto_pat_stat" + string(rune('1'+i)),
			TokenHash:   "$2a$12$testhashhashhashhashhashhashhashhashhashhashhashhash",
			Scopes:      []models.TokenScope{models.ScopeReadAnalytics},
			UseCount:    i * 10,
			CreatedAt:   time.Now(),
		}
		if err := db.CreatePAT(ctx, token); err != nil {
			t.Fatalf("failed to create token: %v", err)
		}
	}

	t.Run("unauthenticated request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/user/tokens/stats", nil)
		w := httptest.NewRecorder()

		handler.PATStats(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
		}
	})

	t.Run("method not allowed", func(t *testing.T) {
		req := requestWithAuth(http.MethodPost, "/api/v1/user/tokens/stats", nil, "user123", "testuser", false)
		w := httptest.NewRecorder()

		handler.PATStats(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
		}
	})

	t.Run("successful stats retrieval", func(t *testing.T) {
		req := requestWithAuth(http.MethodGet, "/api/v1/user/tokens/stats", nil, "user123", "testuser", false)
		w := httptest.NewRecorder()

		handler.PATStats(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
		}

		var resp models.APIResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if resp.Status != "success" {
			t.Errorf("expected status 'success', got %q", resp.Status)
		}

		data, ok := resp.Data.(map[string]interface{})
		if !ok {
			t.Fatalf("expected Data to be map, got %T", resp.Data)
		}

		totalTokens := int(data["total_tokens"].(float64))
		if totalTokens != 3 {
			t.Errorf("expected 3 total tokens, got %d", totalTokens)
		}
	})
}

// TestPATUsageLogs tests the PAT usage logs endpoint.
func TestPATUsageLogs(t *testing.T) {
	handler, db, cleanup := setupPATTestHandler(t)
	defer cleanup()

	ctx := context.Background()

	// Create a test token
	token := &models.PersonalAccessToken{
		ID:          "logs-test-token",
		UserID:      "user123",
		Username:    "testuser",
		Name:        "Logs Test Token",
		TokenPrefix: "carto_pat_logs123",
		TokenHash:   "$2a$12$testhashhashhashhashhashhashhashhashhashhashhashhash",
		Scopes:      []models.TokenScope{models.ScopeReadAnalytics},
		CreatedAt:   time.Now(),
	}
	if err := db.CreatePAT(ctx, token); err != nil {
		t.Fatalf("failed to create token: %v", err)
	}

	// Add some usage logs
	for i := 0; i < 5; i++ {
		log := &models.PATUsageLog{
			TokenID:   "logs-test-token",
			UserID:    "user123",
			Action:    "authenticate",
			Timestamp: time.Now(),
			Success:   true,
			IPAddress: "192.168.1.1",
		}
		if err := db.LogPATUsage(ctx, log); err != nil {
			t.Fatalf("failed to log usage: %v", err)
		}
	}

	t.Run("unauthenticated request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/user/tokens/logs-test-token/logs", nil)
		req = requestWithChiParam(req, "id", "logs-test-token")
		w := httptest.NewRecorder()

		handler.PATUsageLogs(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
		}
	})

	t.Run("access denied for other user", func(t *testing.T) {
		req := requestWithAuth(http.MethodGet, "/api/v1/user/tokens/logs-test-token/logs", nil, "otheruser", "other", false)
		req = requestWithChiParam(req, "id", "logs-test-token")
		w := httptest.NewRecorder()

		handler.PATUsageLogs(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("expected status %d, got %d", http.StatusForbidden, w.Code)
		}
	})

	t.Run("successful logs retrieval", func(t *testing.T) {
		req := requestWithAuth(http.MethodGet, "/api/v1/user/tokens/logs-test-token/logs", nil, "user123", "testuser", false)
		req = requestWithChiParam(req, "id", "logs-test-token")
		w := httptest.NewRecorder()

		handler.PATUsageLogs(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
		}

		var resp models.APIResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if resp.Status != "success" {
			t.Errorf("expected status 'success', got %q", resp.Status)
		}

		logs, ok := resp.Data.([]interface{})
		if !ok {
			t.Fatalf("expected Data to be array, got %T", resp.Data)
		}

		if len(logs) != 5 {
			t.Errorf("expected 5 logs, got %d", len(logs))
		}
	})

	t.Run("with limit parameter", func(t *testing.T) {
		req := requestWithAuth(http.MethodGet, "/api/v1/user/tokens/logs-test-token/logs?limit=2", nil, "user123", "testuser", false)
		req = requestWithChiParam(req, "id", "logs-test-token")
		w := httptest.NewRecorder()

		handler.PATUsageLogs(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}

		var resp models.APIResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		logs, ok := resp.Data.([]interface{})
		if !ok {
			t.Fatalf("expected Data to be array, got %T", resp.Data)
		}

		if len(logs) != 2 {
			t.Errorf("expected 2 logs with limit, got %d", len(logs))
		}
	})
}

// TestPATValidScopes tests that only valid scopes are accepted.
func TestPATValidScopes(t *testing.T) {
	handler, _, cleanup := setupPATTestHandler(t)
	defer cleanup()

	tests := []struct {
		name       string
		scopes     string
		wantStatus int
	}{
		{"valid read scope", `["read:analytics"]`, http.StatusCreated},
		{"valid write scope", `["write:playbacks"]`, http.StatusCreated},
		{"multiple valid scopes", `["read:analytics", "read:users"]`, http.StatusCreated},
		{"invalid scope", `["invalid:scope"]`, http.StatusInternalServerError}, // PAT manager validates
		{"empty scopes", `[]`, http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := `{"name": "Test Token", "scopes": ` + tt.scopes + `}`
			req := requestWithAuth(http.MethodPost, "/api/v1/user/tokens", []byte(body), "scope-test-user", "testuser", false)
			w := httptest.NewRecorder()

			handler.PATCreate(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d: %s", tt.wantStatus, w.Code, w.Body.String())
			}
		})
	}
}
