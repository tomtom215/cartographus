// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/tomtom215/cartographus/internal/cache"
	"github.com/tomtom215/cartographus/internal/config"
	"github.com/tomtom215/cartographus/internal/database"
	"github.com/tomtom215/cartographus/internal/middleware"
	"github.com/tomtom215/cartographus/internal/models"
)

// ============================================================================
// Test Setup Helpers
// ============================================================================

// setupTestHandlerForServerManagement creates a handler with config for server management tests
func setupTestHandlerForServerManagement(t *testing.T, db *database.DB) *Handler {
	t.Helper()
	cfg := &config.Config{
		API: config.APIConfig{
			DefaultPageSize: 100,
			MaxPageSize:     1000,
		},
		Server: config.ServerConfig{
			Latitude:  40.7128,
			Longitude: -74.0060,
		},
		Security: config.SecurityConfig{
			CORSOrigins: []string{"*"},
			JWTSecret:   "test-jwt-secret-for-encryption-32bytes!",
		},
	}

	return &Handler{
		db:        db,
		sync:      nil,
		client:    &MockTautulliClient{},
		config:    cfg,
		startTime: time.Now(),
		cache:     cache.New(5 * time.Minute),
		perfMon:   middleware.NewPerformanceMonitor(1000),
	}
}

// insertTestMediaServer inserts a test media server into the database
func insertTestMediaServer(t *testing.T, db *database.DB, handler *Handler, platform, name, source string) *models.MediaServer {
	t.Helper()

	encryptor, err := handler.getCredentialEncryptor()
	if err != nil {
		t.Fatalf("Failed to create encryptor: %v", err)
	}

	urlEncrypted, err := encryptor.Encrypt("http://test.local:32400")
	if err != nil {
		t.Fatalf("Failed to encrypt URL: %v", err)
	}

	tokenEncrypted, err := encryptor.Encrypt("test-token-12345678")
	if err != nil {
		t.Fatalf("Failed to encrypt token: %v", err)
	}

	server := &models.MediaServer{
		ID:                     uuid.New().String(),
		Platform:               platform,
		Name:                   name,
		URLEncrypted:           urlEncrypted,
		TokenEncrypted:         tokenEncrypted,
		ServerID:               config.GenerateServerID(platform, "http://test.local:32400"),
		Enabled:                true,
		Settings:               "{}",
		RealtimeEnabled:        false,
		WebhooksEnabled:        false,
		SessionPollingEnabled:  false,
		SessionPollingInterval: "30s",
		Source:                 source,
		CreatedBy:              "test-user",
		CreatedAt:              time.Now(),
		UpdatedAt:              time.Now(),
	}

	ctx := context.Background()
	if err := db.CreateMediaServer(ctx, server); err != nil {
		t.Fatalf("Failed to insert test server: %v", err)
	}

	return server
}

// ============================================================================
// CreateServer Tests
// ============================================================================

func TestCreateServer_Success(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()
	handler := setupTestHandlerForServerManagement(t, db)

	reqBody := models.CreateMediaServerRequest{
		Platform:               models.ServerPlatformPlex,
		Name:                   "Test Plex Server",
		URL:                    "http://plex.local:32400",
		Token:                  "test-token-12345678901234567890",
		RealtimeEnabled:        true,
		WebhooksEnabled:        false,
		SessionPollingEnabled:  false,
		SessionPollingInterval: "30s",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/servers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler.CreateServer(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusCreated, w.Code, w.Body.String())
	}

	var resp models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Status != "success" {
		t.Errorf("Expected status 'success', got %q", resp.Status)
	}

	// Verify server data in response
	data, ok := resp.Data.(map[string]any)
	if !ok {
		t.Fatal("Expected data to be a map")
	}

	if data["platform"] != "plex" {
		t.Errorf("Expected platform 'plex', got %v", data["platform"])
	}
	if data["name"] != "Test Plex Server" {
		t.Errorf("Expected name 'Test Plex Server', got %v", data["name"])
	}
	if data["source"] != "ui" {
		t.Errorf("Expected source 'ui', got %v", data["source"])
	}
}

func TestCreateServer_ValidationError(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()
	handler := setupTestHandlerForServerManagement(t, db)

	// Missing required fields
	reqBody := map[string]any{
		"platform": "plex",
		// Missing name, url, token
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/servers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler.CreateServer(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var resp models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Status != "error" {
		t.Errorf("Expected status 'error', got %q", resp.Status)
	}
}

func TestCreateServer_DuplicateURL(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()
	handler := setupTestHandlerForServerManagement(t, db)

	// Insert first server
	insertTestMediaServer(t, db, handler, models.ServerPlatformPlex, "First Server", models.ServerSourceUI)

	// Try to create duplicate
	reqBody := models.CreateMediaServerRequest{
		Platform:               models.ServerPlatformPlex,
		Name:                   "Duplicate Server",
		URL:                    "http://test.local:32400", // Same URL
		Token:                  "test-token-12345678901234567890",
		SessionPollingInterval: "30s",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/servers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler.CreateServer(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusConflict, w.Code, w.Body.String())
	}
}

func TestCreateServer_InvalidJSON(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()
	handler := setupTestHandlerForServerManagement(t, db)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/servers", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler.CreateServer(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

// ============================================================================
// GetServer Tests
// ============================================================================

func TestGetServer_Success(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()
	handler := setupTestHandlerForServerManagement(t, db)

	// Insert test server
	server := insertTestMediaServer(t, db, handler, models.ServerPlatformJellyfin, "Jellyfin Test", models.ServerSourceUI)

	// Create request with chi URL param
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/servers/"+server.ID, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", server.ID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	handler.GetServer(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var resp models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Status != "success" {
		t.Errorf("Expected status 'success', got %q", resp.Status)
	}

	data, ok := resp.Data.(map[string]any)
	if !ok {
		t.Fatal("Expected data to be a map")
	}

	if data["name"] != "Jellyfin Test" {
		t.Errorf("Expected name 'Jellyfin Test', got %v", data["name"])
	}
}

func TestGetServer_NotFound(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()
	handler := setupTestHandlerForServerManagement(t, db)

	fakeID := uuid.New().String()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/servers/"+fakeID, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", fakeID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	handler.GetServer(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestGetServer_MissingID(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()
	handler := setupTestHandlerForServerManagement(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/servers/", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	handler.GetServer(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

// ============================================================================
// UpdateServer Tests
// ============================================================================

func TestUpdateServer_Success(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()
	handler := setupTestHandlerForServerManagement(t, db)

	// Insert test server (UI source so it's mutable)
	server := insertTestMediaServer(t, db, handler, models.ServerPlatformEmby, "Emby Test", models.ServerSourceUI)

	// Update request
	newName := "Updated Emby Server"
	enabled := false
	updateReq := models.UpdateMediaServerRequest{
		Name:    &newName,
		Enabled: &enabled,
	}

	body, _ := json.Marshal(updateReq)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/servers/"+server.ID, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", server.ID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	handler.UpdateServer(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var resp models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	data, ok := resp.Data.(map[string]any)
	if !ok {
		t.Fatal("Expected data to be a map")
	}

	if data["name"] != newName {
		t.Errorf("Expected name %q, got %v", newName, data["name"])
	}
	if data["enabled"] != false {
		t.Errorf("Expected enabled to be false, got %v", data["enabled"])
	}
}

func TestUpdateServer_ImmutableEnvServer(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()
	handler := setupTestHandlerForServerManagement(t, db)

	// Insert env-sourced server (immutable)
	server := insertTestMediaServer(t, db, handler, models.ServerPlatformPlex, "Env Plex", models.ServerSourceEnv)

	// Try to update
	newName := "Tried to Update"
	updateReq := models.UpdateMediaServerRequest{
		Name: &newName,
	}

	body, _ := json.Marshal(updateReq)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/servers/"+server.ID, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", server.ID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	handler.UpdateServer(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusForbidden, w.Code, w.Body.String())
	}
}

func TestUpdateServer_NotFound(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()
	handler := setupTestHandlerForServerManagement(t, db)

	fakeID := uuid.New().String()
	newName := "New Name"
	updateReq := models.UpdateMediaServerRequest{
		Name: &newName,
	}

	body, _ := json.Marshal(updateReq)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/servers/"+fakeID, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", fakeID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	handler.UpdateServer(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

// ============================================================================
// DeleteServer Tests
// ============================================================================

func TestDeleteServer_Success(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()
	handler := setupTestHandlerForServerManagement(t, db)

	// Insert test server (UI source so it's deletable)
	server := insertTestMediaServer(t, db, handler, models.ServerPlatformTautulli, "Tautulli Test", models.ServerSourceUI)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/servers/"+server.ID, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", server.ID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	handler.DeleteServer(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	// Verify server is deleted
	_, err := db.GetMediaServer(context.Background(), server.ID)
	if err == nil {
		t.Error("Expected server to be deleted")
	}
}

func TestDeleteServer_ImmutableEnvServer(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()
	handler := setupTestHandlerForServerManagement(t, db)

	// Insert env-sourced server (immutable)
	server := insertTestMediaServer(t, db, handler, models.ServerPlatformPlex, "Env Plex", models.ServerSourceEnv)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/servers/"+server.ID, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", server.ID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	handler.DeleteServer(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusForbidden, w.Code, w.Body.String())
	}
}

func TestDeleteServer_NotFound(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()
	handler := setupTestHandlerForServerManagement(t, db)

	fakeID := uuid.New().String()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/servers/"+fakeID, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", fakeID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	handler.DeleteServer(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

// ============================================================================
// TestServerConnection Tests
// ============================================================================

func TestTestServerConnection_Success(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()
	handler := setupTestHandlerForServerManagement(t, db)

	testReq := models.MediaServerTestRequest{
		Platform: models.ServerPlatformPlex,
		URL:      "http://plex.local:32400",
		Token:    "test-token-12345678",
	}

	body, _ := json.Marshal(testReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/servers/test", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler.TestServerConnection(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var resp models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Status != "success" {
		t.Errorf("Expected status 'success', got %q", resp.Status)
	}

	data, ok := resp.Data.(map[string]any)
	if !ok {
		t.Fatal("Expected data to be a map")
	}

	if data["success"] != true {
		t.Errorf("Expected success to be true, got %v", data["success"])
	}
}

func TestTestServerConnection_ValidationError(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()
	handler := setupTestHandlerForServerManagement(t, db)

	// Missing required fields
	testReq := map[string]any{
		"platform": "plex",
		// Missing url and token
	}

	body, _ := json.Marshal(testReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/servers/test", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler.TestServerConnection(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestTestServerConnection_AllPlatforms(t *testing.T) {
	t.Parallel()

	platforms := []string{
		models.ServerPlatformPlex,
		models.ServerPlatformJellyfin,
		models.ServerPlatformEmby,
		models.ServerPlatformTautulli,
	}

	for _, platform := range platforms {
		t.Run(platform, func(t *testing.T) {
			db := setupTestDBForAPI(t)
			defer db.Close()
			handler := setupTestHandlerForServerManagement(t, db)

			testReq := models.MediaServerTestRequest{
				Platform: platform,
				URL:      "http://test.local:8080",
				Token:    "test-token-12345678",
			}

			body, _ := json.Marshal(testReq)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/servers/test", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			handler.TestServerConnection(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status %d for platform %s, got %d", http.StatusOK, platform, w.Code)
			}
		})
	}
}

// ============================================================================
// ListDBServers Tests
// ============================================================================

func TestListDBServers_Success(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()
	handler := setupTestHandlerForServerManagement(t, db)

	// Insert test servers
	insertTestMediaServer(t, db, handler, models.ServerPlatformPlex, "Plex 1", models.ServerSourceUI)
	insertTestMediaServer(t, db, handler, models.ServerPlatformJellyfin, "Jellyfin 1", models.ServerSourceUI)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/servers/db", nil)

	w := httptest.NewRecorder()
	handler.ListDBServers(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var resp models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Status != "success" {
		t.Errorf("Expected status 'success', got %q", resp.Status)
	}

	data, ok := resp.Data.([]any)
	if !ok {
		t.Fatal("Expected data to be an array")
	}

	if len(data) < 2 {
		t.Errorf("Expected at least 2 servers, got %d", len(data))
	}
}

func TestListDBServers_FilterByPlatform(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()
	handler := setupTestHandlerForServerManagement(t, db)

	// Insert test servers with different platforms
	insertTestMediaServer(t, db, handler, models.ServerPlatformPlex, "Plex 1", models.ServerSourceUI)
	insertTestMediaServer(t, db, handler, models.ServerPlatformJellyfin, "Jellyfin 1", models.ServerSourceUI)

	// Filter by platform
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/servers/db?platform=plex", nil)

	w := httptest.NewRecorder()
	handler.ListDBServers(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	data, ok := resp.Data.([]any)
	if !ok {
		t.Fatal("Expected data to be an array")
	}

	// Should only return plex servers
	for _, item := range data {
		server, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if server["platform"] != "plex" {
			t.Errorf("Expected platform 'plex', got %v", server["platform"])
		}
	}
}

func TestListDBServers_Empty(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()
	handler := setupTestHandlerForServerManagement(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/servers/db", nil)

	w := httptest.NewRecorder()
	handler.ListDBServers(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	data, ok := resp.Data.([]any)
	if !ok {
		t.Fatal("Expected data to be an array")
	}

	if len(data) != 0 {
		t.Errorf("Expected 0 servers, got %d", len(data))
	}
}

// ============================================================================
// CRUD Lifecycle Test
// ============================================================================

func TestServerManagement_CRUDLifecycle(t *testing.T) {
	t.Parallel()

	db := setupTestDBForAPI(t)
	defer db.Close()
	handler := setupTestHandlerForServerManagement(t, db)

	// 1. CREATE
	createReq := models.CreateMediaServerRequest{
		Platform:               models.ServerPlatformPlex,
		Name:                   "Lifecycle Test Server",
		URL:                    "http://lifecycle.local:32400",
		Token:                  "lifecycle-token-12345678901234567890",
		RealtimeEnabled:        true,
		SessionPollingInterval: "1m",
	}

	createBody, _ := json.Marshal(createReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/servers", bytes.NewReader(createBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler.CreateServer(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Create failed: %d - %s", w.Code, w.Body.String())
	}

	var createResp models.APIResponse
	json.NewDecoder(w.Body).Decode(&createResp)
	data := createResp.Data.(map[string]any)
	serverID := data["id"].(string)

	// 2. READ
	req = httptest.NewRequest(http.MethodGet, "/api/v1/admin/servers/"+serverID, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", serverID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w = httptest.NewRecorder()
	handler.GetServer(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Read failed: %d - %s", w.Code, w.Body.String())
	}

	// 3. UPDATE
	newName := "Updated Lifecycle Server"
	updateReq := models.UpdateMediaServerRequest{
		Name: &newName,
	}

	updateBody, _ := json.Marshal(updateReq)
	req = httptest.NewRequest(http.MethodPut, "/api/v1/admin/servers/"+serverID, bytes.NewReader(updateBody))
	req.Header.Set("Content-Type", "application/json")
	rctx = chi.NewRouteContext()
	rctx.URLParams.Add("id", serverID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w = httptest.NewRecorder()
	handler.UpdateServer(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Update failed: %d - %s", w.Code, w.Body.String())
	}

	// Verify update
	var updateResp models.APIResponse
	json.NewDecoder(w.Body).Decode(&updateResp)
	updateData := updateResp.Data.(map[string]any)
	if updateData["name"] != newName {
		t.Errorf("Expected name %q after update, got %v", newName, updateData["name"])
	}

	// 4. DELETE
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/admin/servers/"+serverID, nil)
	rctx = chi.NewRouteContext()
	rctx.URLParams.Add("id", serverID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w = httptest.NewRecorder()
	handler.DeleteServer(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Delete failed: %d - %s", w.Code, w.Body.String())
	}

	// 5. Verify DELETE - should get 404
	req = httptest.NewRequest(http.MethodGet, "/api/v1/admin/servers/"+serverID, nil)
	rctx = chi.NewRouteContext()
	rctx.URLParams.Add("id", serverID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w = httptest.NewRecorder()
	handler.GetServer(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected 404 after delete, got %d", w.Code)
	}
}

// ============================================================================
// Helper Function Tests
// ============================================================================

func TestGetServerStatus(t *testing.T) {
	tests := []struct {
		name     string
		server   *models.MediaServer
		expected string
	}{
		{
			name:     "disabled server",
			server:   &models.MediaServer{Enabled: false},
			expected: "disabled",
		},
		{
			name: "server with error",
			server: &models.MediaServer{
				Enabled:     true,
				LastError:   "Connection failed",
				LastErrorAt: func() *time.Time { t := time.Now(); return &t }(),
			},
			expected: "error",
		},
		{
			name: "connected server",
			server: &models.MediaServer{
				Enabled:    true,
				LastSyncAt: func() *time.Time { t := time.Now(); return &t }(),
			},
			expected: "connected",
		},
		{
			name: "configured but not synced",
			server: &models.MediaServer{
				Enabled: true,
			},
			expected: "configured",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getServerStatus(tt.server)
			if result != tt.expected {
				t.Errorf("getServerStatus() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestGetUserFromContext(t *testing.T) {
	t.Run("with user in context", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		ctx := context.WithValue(req.Context(), userIDContextKey, "user-123")
		ctx = context.WithValue(ctx, usernameContextKey, "testuser")
		req = req.WithContext(ctx)

		userID, username := getUserFromContext(req)
		if userID != "user-123" {
			t.Errorf("Expected user_id 'user-123', got %q", userID)
		}
		if username != "testuser" {
			t.Errorf("Expected username 'testuser', got %q", username)
		}
	})

	t.Run("without user in context", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)

		userID, username := getUserFromContext(req)
		if userID != "unknown" {
			t.Errorf("Expected user_id 'unknown', got %q", userID)
		}
		if username != "unknown" {
			t.Errorf("Expected username 'unknown', got %q", username)
		}
	})
}
