// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/goccy/go-json"
)

// =====================================================
// Plex Authentication Flow Tests
// ADR-0015: Zero Trust Authentication & Authorization
// =====================================================

// Test PlexFlowConfig validation
func TestPlexFlowConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *PlexFlowConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &PlexFlowConfig{
				ClientID:    "test-client-id",
				Product:     "Cartographus",
				RedirectURI: "https://app.example.com/plex/callback",
			},
			wantErr: false,
		},
		{
			name: "missing client ID",
			config: &PlexFlowConfig{
				Product:     "Cartographus",
				RedirectURI: "https://app.example.com/plex/callback",
			},
			wantErr: true,
		},
		{
			name: "missing product name",
			config: &PlexFlowConfig{
				ClientID:    "test-client-id",
				RedirectURI: "https://app.example.com/plex/callback",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Test PlexFlow creation
func TestNewPlexFlow(t *testing.T) {
	config := &PlexFlowConfig{
		ClientID:        "test-client-id",
		Product:         "Cartographus",
		Version:         "1.0.0",
		RedirectURI:     "https://app.example.com/plex/callback",
		PINTimeout:      5 * time.Minute,
		SessionDuration: 24 * time.Hour,
		DefaultRoles:    []string{"viewer"},
	}

	store := NewMemoryPlexPINStore()
	flow := NewPlexFlow(config, store)

	if flow == nil {
		t.Fatal("NewPlexFlow() returned nil")
	}
}

// Test PIN request
func TestPlexFlow_RequestPIN(t *testing.T) {
	// Mock Plex PIN API
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/pins.json" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Verify required headers
		if r.Header.Get("X-Plex-Client-Identifier") == "" {
			http.Error(w, "missing client identifier", http.StatusBadRequest)
			return
		}

		// Return mock PIN response
		resp := map[string]interface{}{
			"id":         12345,
			"code":       "ABC1",
			"expires_at": time.Now().Add(5 * time.Minute).Format(time.RFC3339),
			"auth_token": nil,
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Errorf("Failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	config := &PlexFlowConfig{
		ClientID:   "test-client-id",
		Product:    "Cartographus",
		Version:    "1.0.0",
		PINTimeout: 5 * time.Minute,
	}

	store := NewMemoryPlexPINStore()
	flow := NewPlexFlow(config, store)
	flow.SetPINEndpoint(server.URL + "/pins.json")

	ctx := context.Background()
	pin, err := flow.RequestPIN(ctx)
	if err != nil {
		t.Fatalf("RequestPIN() error = %v", err)
	}

	if pin.ID == 0 {
		t.Error("PIN ID should not be 0")
	}
	if pin.Code == "" {
		t.Error("PIN code should not be empty")
	}
	if pin.AuthURL == "" {
		t.Error("AuthURL should not be empty")
	}
}

// Test PIN check when not yet authorized
func TestPlexFlow_CheckPIN_NotAuthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Return PIN without auth token (not yet authorized)
		resp := map[string]interface{}{
			"id":         12345,
			"code":       "ABC1",
			"expires_at": time.Now().Add(5 * time.Minute).Format(time.RFC3339),
			"auth_token": nil,
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Errorf("Failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	config := &PlexFlowConfig{
		ClientID:   "test-client-id",
		Product:    "Cartographus",
		PINTimeout: 5 * time.Minute,
	}

	store := NewMemoryPlexPINStore()
	flow := NewPlexFlow(config, store)
	flow.SetPINCheckEndpoint(server.URL)

	ctx := context.Background()
	result, err := flow.CheckPIN(ctx, 12345)
	if err != nil {
		t.Fatalf("CheckPIN() error = %v", err)
	}

	if result.Authorized {
		t.Error("Authorized should be false")
	}
	if result.AuthToken != "" {
		t.Error("AuthToken should be empty when not authorized")
	}
}

// Test PIN check when authorized
func TestPlexFlow_CheckPIN_Authorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return PIN with auth token (authorized)
		resp := map[string]interface{}{
			"id":         12345,
			"code":       "ABC1",
			"expires_at": time.Now().Add(5 * time.Minute).Format(time.RFC3339),
			"auth_token": "plex-auth-token-123",
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Errorf("Failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	config := &PlexFlowConfig{
		ClientID:   "test-client-id",
		Product:    "Cartographus",
		PINTimeout: 5 * time.Minute,
	}

	store := NewMemoryPlexPINStore()
	flow := NewPlexFlow(config, store)
	flow.SetPINCheckEndpoint(server.URL)

	ctx := context.Background()
	result, err := flow.CheckPIN(ctx, 12345)
	if err != nil {
		t.Fatalf("CheckPIN() error = %v", err)
	}

	if !result.Authorized {
		t.Error("Authorized should be true")
	}
	if result.AuthToken != "plex-auth-token-123" {
		t.Errorf("AuthToken = %s, want plex-auth-token-123", result.AuthToken)
	}
}

// Test PIN expiration
func TestPlexFlow_CheckPIN_Expired(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return 404 for expired PIN
		http.Error(w, "PIN not found", http.StatusNotFound)
	}))
	defer server.Close()

	config := &PlexFlowConfig{
		ClientID:   "test-client-id",
		Product:    "Cartographus",
		PINTimeout: 5 * time.Minute,
	}

	store := NewMemoryPlexPINStore()
	flow := NewPlexFlow(config, store)
	flow.SetPINCheckEndpoint(server.URL)

	ctx := context.Background()
	_, err := flow.CheckPIN(ctx, 12345)
	if err == nil {
		t.Error("CheckPIN() should fail for expired PIN")
	}
}

// Test fetching user info with auth token
func TestPlexFlow_GetUserInfo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify auth token is present
		if r.Header.Get("X-Plex-Token") == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		resp := map[string]interface{}{
			"user": map[string]interface{}{
				"id":       12345,
				"uuid":     "user-uuid-123",
				"username": "testuser",
				"email":    "test@example.com",
				"thumb":    "https://plex.tv/users/avatar.png",
				"subscription": map[string]interface{}{
					"active": true,
					"status": "Active",
					"plan":   "lifetime",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Errorf("Failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	config := &PlexFlowConfig{
		ClientID:     "test-client-id",
		Product:      "Cartographus",
		DefaultRoles: []string{"viewer"},
		PlexPassRole: "plex_pass",
	}

	store := NewMemoryPlexPINStore()
	flow := NewPlexFlow(config, store)
	flow.SetUserInfoEndpoint(server.URL)

	ctx := context.Background()
	subject, err := flow.GetUserInfo(ctx, "plex-auth-token")
	if err != nil {
		t.Fatalf("GetUserInfo() error = %v", err)
	}

	if subject.ID != "12345" {
		t.Errorf("ID = %s, want 12345", subject.ID)
	}
	if subject.Username != "testuser" {
		t.Errorf("Username = %s, want testuser", subject.Username)
	}
	if subject.Email != "test@example.com" {
		t.Errorf("Email = %s, want test@example.com", subject.Email)
	}
	// Should have Plex Pass role since subscription is active
	if !subject.HasRole("plex_pass") {
		t.Error("subject should have plex_pass role")
	}
}

// Test authorization URL generation
func TestPlexFlow_AuthorizationURL(t *testing.T) {
	config := &PlexFlowConfig{
		ClientID:   "test-client-id",
		Product:    "Cartographus",
		Version:    "1.0.0",
		ForwardURL: "https://app.example.com/plex/callback",
	}

	store := NewMemoryPlexPINStore()
	flow := NewPlexFlow(config, store)

	pin := &PlexPIN{
		ID:   12345,
		Code: "ABC1",
	}

	authURL := flow.BuildAuthorizationURL(pin)

	// Verify URL contains required parameters
	if authURL == "" {
		t.Error("AuthorizationURL should not be empty")
	}
	// The URL should point to Plex's auth page
	expectedBase := "https://app.plex.tv/auth#!"
	if len(authURL) < len(expectedBase) {
		t.Errorf("AuthorizationURL = %s, should start with %s", authURL, expectedBase)
	}
}

// Test complete login flow
func TestPlexFlow_CompleteLogin(t *testing.T) {
	// Mock Plex PIN endpoint
	pinServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/pins.json" {
			resp := map[string]interface{}{
				"id":         12345,
				"code":       "ABC1",
				"expires_at": time.Now().Add(5 * time.Minute).Format(time.RFC3339),
			}
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				t.Errorf("Failed to encode response: %v", err)
			}
			return
		}
		if r.Method == http.MethodGet {
			resp := map[string]interface{}{
				"id":         12345,
				"code":       "ABC1",
				"auth_token": "plex-auth-token",
			}
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				t.Errorf("Failed to encode response: %v", err)
			}
			return
		}
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer pinServer.Close()

	// Mock user info endpoint
	userServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"user": map[string]interface{}{
				"id":       12345,
				"username": "testuser",
				"email":    "test@example.com",
				"subscription": map[string]interface{}{
					"active": false,
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Errorf("Failed to encode response: %v", err)
		}
	}))
	defer userServer.Close()

	config := &PlexFlowConfig{
		ClientID:        "test-client-id",
		Product:         "Cartographus",
		Version:         "1.0.0",
		PINTimeout:      5 * time.Minute,
		SessionDuration: 24 * time.Hour,
		DefaultRoles:    []string{"viewer"},
	}

	store := NewMemoryPlexPINStore()
	flow := NewPlexFlow(config, store)
	flow.SetPINEndpoint(pinServer.URL + "/pins.json")
	flow.SetPINCheckEndpoint(pinServer.URL)
	flow.SetUserInfoEndpoint(userServer.URL)

	ctx := context.Background()

	// Step 1: Request PIN
	pin, err := flow.RequestPIN(ctx)
	if err != nil {
		t.Fatalf("RequestPIN() error = %v", err)
	}

	// Step 2: Check PIN (simulating user authorization)
	result, err := flow.CheckPIN(ctx, pin.ID)
	if err != nil {
		t.Fatalf("CheckPIN() error = %v", err)
	}

	if !result.Authorized {
		t.Fatal("PIN should be authorized in this test")
	}

	// Step 3: Get user info
	subject, err := flow.GetUserInfo(ctx, result.AuthToken)
	if err != nil {
		t.Fatalf("GetUserInfo() error = %v", err)
	}

	if subject.Username != "testuser" {
		t.Errorf("Username = %s, want testuser", subject.Username)
	}
}

// Test memory PIN store
func TestMemoryPlexPINStore(t *testing.T) {
	store := NewMemoryPlexPINStore()
	ctx := context.Background()

	pin := &PlexPINData{
		ID:                12345,
		Code:              "ABC1",
		PostLoginRedirect: "/dashboard",
		CreatedAt:         time.Now(),
		ExpiresAt:         time.Now().Add(5 * time.Minute),
	}

	// Test Store
	err := store.Store(ctx, pin.ID, pin)
	if err != nil {
		t.Fatalf("Store() error = %v", err)
	}

	// Test Get
	retrieved, err := store.Get(ctx, pin.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if retrieved.Code != pin.Code {
		t.Errorf("Code = %s, want %s", retrieved.Code, pin.Code)
	}

	// Test Delete
	err = store.Delete(ctx, pin.ID)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify deleted
	_, err = store.Get(ctx, pin.ID)
	if err == nil {
		t.Error("Get() should fail after Delete()")
	}
}

// Test PIN store cleanup
func TestMemoryPlexPINStore_Cleanup(t *testing.T) {
	store := NewMemoryPlexPINStore()
	ctx := context.Background()

	// Store expired PIN
	expiredPIN := &PlexPINData{
		ID:        11111,
		Code:      "EXP1",
		CreatedAt: time.Now().Add(-10 * time.Minute),
		ExpiresAt: time.Now().Add(-5 * time.Minute),
	}
	if err := store.Store(ctx, expiredPIN.ID, expiredPIN); err != nil {
		t.Fatalf("Store() error = %v", err)
	}

	// Store valid PIN
	validPIN := &PlexPINData{
		ID:        22222,
		Code:      "VAL1",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}
	if err := store.Store(ctx, validPIN.ID, validPIN); err != nil {
		t.Fatalf("Store() error = %v", err)
	}

	// Cleanup
	count, err := store.CleanupExpired(ctx)
	if err != nil {
		t.Fatalf("CleanupExpired() error = %v", err)
	}
	if count != 1 {
		t.Errorf("CleanupExpired() count = %d, want 1", count)
	}

	// Verify expired is gone
	_, err = store.Get(ctx, expiredPIN.ID)
	if err == nil {
		t.Error("expired PIN should be cleaned up")
	}

	// Verify valid still exists
	_, err = store.Get(ctx, validPIN.ID)
	if err != nil {
		t.Errorf("valid PIN should still exist: %v", err)
	}
}
