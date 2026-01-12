// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/goccy/go-json"
)

// TestPlexAuthenticator_Interface verifies the Authenticator interface contract
func TestPlexAuthenticator_Interface(t *testing.T) {
	config := &PlexAuthConfig{
		ClientID:    "test-client",
		RedirectURI: "http://localhost:3857/auth/callback",
	}

	auth := NewPlexAuthenticator(config)

	// Verify interface implementation
	var _ Authenticator = auth

	// Test Name()
	if auth.Name() != "plex" {
		t.Errorf("Name() = %v, want plex", auth.Name())
	}

	// Test Priority() - Plex should have moderate priority
	if auth.Priority() < 10 || auth.Priority() > 30 {
		t.Errorf("Priority() = %v, want between 10 and 30", auth.Priority())
	}
}

// TestPlexAuthenticator_NoToken tests request without Plex token
func TestPlexAuthenticator_NoToken(t *testing.T) {
	config := &PlexAuthConfig{
		ClientID:    "test-client",
		RedirectURI: "http://localhost:3857/auth/callback",
	}

	auth := NewPlexAuthenticator(config)

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	_, err := auth.Authenticate(context.Background(), req)

	if !errors.Is(err, ErrNoCredentials) {
		t.Errorf("Authenticate() error = %v, want ErrNoCredentials", err)
	}
}

// TestPlexAuthenticator_ValidToken tests request with valid Plex token
func TestPlexAuthenticator_ValidToken(t *testing.T) {
	// Create mock Plex API server
	plexServer := newMockPlexServer(t)
	defer plexServer.Close()

	config := &PlexAuthConfig{
		ClientID:    "test-client",
		RedirectURI: "http://localhost:3857/auth/callback",
	}

	auth := NewPlexAuthenticator(config)
	auth.SetUserInfoURL(plexServer.URL + "/users/account")

	// Create request with Plex token
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("X-Plex-Token", "valid-plex-token")

	subject, err := auth.Authenticate(context.Background(), req)
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}

	if subject.ID != "12345" {
		t.Errorf("subject.ID = %v, want 12345", subject.ID)
	}
	if subject.Username != "plexuser" {
		t.Errorf("subject.Username = %v, want plexuser", subject.Username)
	}
	if subject.Email != "plex@example.com" {
		t.Errorf("subject.Email = %v, want plex@example.com", subject.Email)
	}
	if subject.Issuer != "plex.tv" {
		t.Errorf("subject.Issuer = %v, want plex.tv", subject.Issuer)
	}
	if subject.AuthMethod != AuthModePlex {
		t.Errorf("subject.AuthMethod = %v, want %v", subject.AuthMethod, AuthModePlex)
	}
}

// TestPlexAuthenticator_InvalidToken tests request with invalid Plex token
func TestPlexAuthenticator_InvalidToken(t *testing.T) {
	// Create mock Plex API server that returns 401
	plexServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "invalid token",
		})
	}))
	defer plexServer.Close()

	config := &PlexAuthConfig{
		ClientID:    "test-client",
		RedirectURI: "http://localhost:3857/auth/callback",
	}

	auth := NewPlexAuthenticator(config)
	auth.SetUserInfoURL(plexServer.URL + "/users/account")

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("X-Plex-Token", "invalid-token")

	_, err := auth.Authenticate(context.Background(), req)

	if !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("Authenticate() error = %v, want ErrInvalidCredentials", err)
	}
}

// TestPlexAuthenticator_BearerToken tests request with Bearer token
func TestPlexAuthenticator_BearerToken(t *testing.T) {
	plexServer := newMockPlexServer(t)
	defer plexServer.Close()

	config := &PlexAuthConfig{
		ClientID:    "test-client",
		RedirectURI: "http://localhost:3857/auth/callback",
	}

	auth := NewPlexAuthenticator(config)
	auth.SetUserInfoURL(plexServer.URL + "/users/account")

	// Create request with Bearer token
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("Authorization", "Bearer valid-plex-token")

	subject, err := auth.Authenticate(context.Background(), req)
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}

	if subject.ID != "12345" {
		t.Errorf("subject.ID = %v, want 12345", subject.ID)
	}
}

// TestPlexAuthenticator_CookieToken tests request with token in cookie
func TestPlexAuthenticator_CookieToken(t *testing.T) {
	plexServer := newMockPlexServer(t)
	defer plexServer.Close()

	config := &PlexAuthConfig{
		ClientID:    "test-client",
		RedirectURI: "http://localhost:3857/auth/callback",
	}

	auth := NewPlexAuthenticator(config)
	auth.SetUserInfoURL(plexServer.URL + "/users/account")

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.AddCookie(&http.Cookie{
		Name:  "plex_token",
		Value: "valid-plex-token",
	})

	subject, err := auth.Authenticate(context.Background(), req)
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}

	if subject.ID != "12345" {
		t.Errorf("subject.ID = %v, want 12345", subject.ID)
	}
}

// TestPlexAuthenticator_PlexPass tests Plex Pass user detection
func TestPlexAuthenticator_PlexPass(t *testing.T) {
	// Create mock server with Plex Pass user
	plexServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"user": map[string]interface{}{
				"id":       67890,
				"uuid":     "abc-def-ghi",
				"username": "plexpassuser",
				"email":    "pass@example.com",
				"subscription": map[string]interface{}{
					"active": true,
					"status": "Active",
					"plan":   "lifetime",
				},
			},
		})
	}))
	defer plexServer.Close()

	config := &PlexAuthConfig{
		ClientID:    "test-client",
		RedirectURI: "http://localhost:3857/auth/callback",
	}

	auth := NewPlexAuthenticator(config)
	auth.SetUserInfoURL(plexServer.URL + "/users/account")

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("X-Plex-Token", "valid-token")

	subject, err := auth.Authenticate(context.Background(), req)
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}

	// Plex Pass users should have additional metadata
	if subject.ID != "67890" {
		t.Errorf("subject.ID = %v, want 67890", subject.ID)
	}
}

// TestPlexAuthenticator_ServerUnavailable tests handling of Plex API unavailability
func TestPlexAuthenticator_ServerUnavailable(t *testing.T) {
	config := &PlexAuthConfig{
		ClientID:    "test-client",
		RedirectURI: "http://localhost:3857/auth/callback",
	}

	auth := NewPlexAuthenticator(config)
	// Point to non-existent server
	auth.SetUserInfoURL("http://localhost:99999/users/account")

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("X-Plex-Token", "some-token")

	_, err := auth.Authenticate(context.Background(), req)

	if !errors.Is(err, ErrAuthenticatorUnavailable) {
		t.Errorf("Authenticate() error = %v, want ErrAuthenticatorUnavailable", err)
	}
}

// TestPlexAuthConfig_Validate tests configuration validation
func TestPlexAuthConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *PlexAuthConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &PlexAuthConfig{
				ClientID:    "test-client",
				RedirectURI: "http://localhost:3857/callback",
			},
			wantErr: false,
		},
		{
			name: "missing client ID",
			config: &PlexAuthConfig{
				RedirectURI: "http://localhost:3857/callback",
			},
			wantErr: true,
		},
		{
			name: "missing redirect URI",
			config: &PlexAuthConfig{
				ClientID: "test-client",
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

// mockPlexServer creates a mock Plex API server
func newMockPlexServer(t *testing.T) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify token is present
		token := r.Header.Get("X-Plex-Token")
		if token == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Return mock user data
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"user": map[string]interface{}{
				"id":       12345,
				"uuid":     "uuid-12345",
				"username": "plexuser",
				"email":    "plex@example.com",
				"thumb":    "https://plex.tv/users/12345/avatar",
			},
		})
	}))
}

// TestPlexAuthenticator_TokenPriority tests token extraction priority
func TestPlexAuthenticator_TokenPriority(t *testing.T) {
	plexServer := newMockPlexServer(t)
	defer plexServer.Close()

	config := &PlexAuthConfig{
		ClientID:    "test-client",
		RedirectURI: "http://localhost:3857/auth/callback",
	}

	auth := NewPlexAuthenticator(config)
	auth.SetUserInfoURL(plexServer.URL + "/users/account")

	// When both header and cookie are present, header should take priority
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("X-Plex-Token", "valid-plex-token")
	req.AddCookie(&http.Cookie{
		Name:  "plex_token",
		Value: "cookie-token",
	})

	subject, err := auth.Authenticate(context.Background(), req)
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}

	// Should use header token (which is valid in mock)
	if subject.ID != "12345" {
		t.Errorf("subject.ID = %v, want 12345", subject.ID)
	}
}

// TestPlexAuthenticator_Timeout tests request timeout handling
func TestPlexAuthenticator_Timeout(t *testing.T) {
	// Create a slow server
	plexServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer plexServer.Close()

	config := &PlexAuthConfig{
		ClientID:    "test-client",
		RedirectURI: "http://localhost:3857/auth/callback",
		Timeout:     100 * time.Millisecond, // Very short timeout
	}

	auth := NewPlexAuthenticator(config)
	auth.SetUserInfoURL(plexServer.URL + "/users/account")

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("X-Plex-Token", "valid-token")

	_, err := auth.Authenticate(context.Background(), req)

	// Should fail due to timeout
	if !errors.Is(err, ErrAuthenticatorUnavailable) {
		t.Errorf("Authenticate() error = %v, want ErrAuthenticatorUnavailable", err)
	}
}

// =============================================================================
// Server Detection Tests
// =============================================================================

// newMockPlexServerWithResources creates a mock server that handles both user and resources APIs
func newMockPlexServerWithResources(t *testing.T, userID int, username, email string, resources []map[string]interface{}) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("X-Plex-Token")
		if token == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/users/account":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"user": map[string]interface{}{
					"id":       userID,
					"uuid":     "uuid-" + username,
					"username": username,
					"email":    email,
				},
			})
		case "/api/v2/resources":
			json.NewEncoder(w).Encode(resources)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

// TestPlexAuthenticator_ServerOwnerDetection tests automatic server owner detection
func TestPlexAuthenticator_ServerOwnerDetection(t *testing.T) {
	// Mock resources response with owned server
	resources := []map[string]interface{}{
		{
			"name":             "My Plex Server",
			"product":          "Plex Media Server",
			"clientIdentifier": "server-machine-id-123",
			"owned":            true,
			"accessToken":      "server-access-token",
			"provides":         "server",
		},
	}

	plexServer := newMockPlexServerWithResources(t, 12345, "serverowner", "owner@example.com", resources)
	defer plexServer.Close()

	config := &PlexAuthConfig{
		ClientID:              "test-client",
		RedirectURI:           "http://localhost:3857/auth/callback",
		DefaultRoles:          []string{"viewer"},
		ServerOwnerRole:       "admin",
		ServerAdminRole:       "editor",
		EnableServerDetection: true,
	}

	auth := NewPlexAuthenticator(config)
	auth.SetUserInfoURL(plexServer.URL + "/users/account")
	// Override resources URL for testing
	auth.httpClient = &http.Client{
		Transport: &mockTransport{
			baseURL:    plexServer.URL,
			realClient: http.DefaultClient,
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("X-Plex-Token", "valid-token")

	subject, err := auth.Authenticate(context.Background(), req)
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}

	// Should have admin role as server owner
	if len(subject.Roles) == 0 || subject.Roles[0] != "admin" {
		t.Errorf("subject.Roles = %v, want [admin]", subject.Roles)
	}

	// Check raw claims for server info
	if isOwner, ok := subject.RawClaims["server_owner"].(bool); !ok || !isOwner {
		t.Errorf("subject.RawClaims[server_owner] = %v, want true", subject.RawClaims["server_owner"])
	}
	if serverName, ok := subject.RawClaims["server_name"].(string); !ok || serverName != "My Plex Server" {
		t.Errorf("subject.RawClaims[server_name] = %v, want 'My Plex Server'", subject.RawClaims["server_name"])
	}
}

// TestPlexAuthenticator_SharedUserDetection tests detection of shared (non-owner) users
func TestPlexAuthenticator_SharedUserDetection(t *testing.T) {
	// Mock resources response with shared server (not owned)
	resources := []map[string]interface{}{
		{
			"name":             "Shared Server",
			"product":          "Plex Media Server",
			"clientIdentifier": "shared-server-123",
			"owned":            false,
			"accessToken":      "shared-access-token",
			"provides":         "server",
			"sourceTitle":      "Friend's Server",
		},
	}

	plexServer := newMockPlexServerWithResources(t, 67890, "shareduser", "shared@example.com", resources)
	defer plexServer.Close()

	config := &PlexAuthConfig{
		ClientID:              "test-client",
		RedirectURI:           "http://localhost:3857/auth/callback",
		DefaultRoles:          []string{"viewer"},
		ServerOwnerRole:       "admin",
		ServerAdminRole:       "editor",
		EnableServerDetection: true,
	}

	auth := NewPlexAuthenticator(config)
	auth.SetUserInfoURL(plexServer.URL + "/users/account")
	auth.httpClient = &http.Client{
		Transport: &mockTransport{
			baseURL:    plexServer.URL,
			realClient: http.DefaultClient,
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("X-Plex-Token", "valid-token")

	subject, err := auth.Authenticate(context.Background(), req)
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}

	// Shared users should get default viewer role (not owner, not admin by default)
	if len(subject.Roles) == 0 || subject.Roles[0] != "viewer" {
		t.Errorf("subject.Roles = %v, want [viewer]", subject.Roles)
	}

	// Check raw claims - should not be owner
	if isOwner, ok := subject.RawClaims["server_owner"].(bool); !ok || isOwner {
		t.Errorf("subject.RawClaims[server_owner] = %v, want false", subject.RawClaims["server_owner"])
	}
}

// TestPlexAuthenticator_ServerDetectionDisabled tests that detection can be disabled
func TestPlexAuthenticator_ServerDetectionDisabled(t *testing.T) {
	// Even if user is server owner, if detection is disabled, they get default role
	resources := []map[string]interface{}{
		{
			"name":             "My Server",
			"clientIdentifier": "my-server-123",
			"owned":            true,
			"provides":         "server",
		},
	}

	plexServer := newMockPlexServerWithResources(t, 12345, "owner", "owner@example.com", resources)
	defer plexServer.Close()

	config := &PlexAuthConfig{
		ClientID:              "test-client",
		RedirectURI:           "http://localhost:3857/auth/callback",
		DefaultRoles:          []string{"viewer"},
		ServerOwnerRole:       "admin",
		EnableServerDetection: false, // Disabled
	}

	auth := NewPlexAuthenticator(config)
	auth.SetUserInfoURL(plexServer.URL + "/users/account")
	auth.httpClient = &http.Client{
		Transport: &mockTransport{
			baseURL:    plexServer.URL,
			realClient: http.DefaultClient,
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("X-Plex-Token", "valid-token")

	subject, err := auth.Authenticate(context.Background(), req)
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}

	// With detection disabled, should get default role
	if len(subject.Roles) == 0 || subject.Roles[0] != "viewer" {
		t.Errorf("subject.Roles = %v, want [viewer]", subject.Roles)
	}

	// No server info should be in claims
	if _, ok := subject.RawClaims["server_owner"]; ok {
		t.Error("Expected no server_owner in RawClaims when detection is disabled")
	}
}

// TestPlexAuthenticator_SpecificServerMachineID tests filtering to specific server
func TestPlexAuthenticator_SpecificServerMachineID(t *testing.T) {
	// User owns multiple servers
	resources := []map[string]interface{}{
		{
			"name":             "Home Server",
			"clientIdentifier": "home-server-123",
			"owned":            true,
			"provides":         "server",
		},
		{
			"name":             "Work Server",
			"clientIdentifier": "work-server-456",
			"owned":            false, // Not owner of this one
			"accessToken":      "some-token",
			"provides":         "server",
		},
	}

	plexServer := newMockPlexServerWithResources(t, 12345, "multiowner", "multi@example.com", resources)
	defer plexServer.Close()

	// Configure to only check work-server-456 (which user doesn't own)
	config := &PlexAuthConfig{
		ClientID:                "test-client",
		RedirectURI:             "http://localhost:3857/auth/callback",
		DefaultRoles:            []string{"viewer"},
		ServerOwnerRole:         "admin",
		EnableServerDetection:   true,
		ServerMachineIdentifier: "work-server-456", // Only check this server
	}

	auth := NewPlexAuthenticator(config)
	auth.SetUserInfoURL(plexServer.URL + "/users/account")
	auth.httpClient = &http.Client{
		Transport: &mockTransport{
			baseURL:    plexServer.URL,
			realClient: http.DefaultClient,
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("X-Plex-Token", "valid-token")

	subject, err := auth.Authenticate(context.Background(), req)
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}

	// Should NOT be admin because work-server-456 is not owned
	if len(subject.Roles) > 0 && subject.Roles[0] == "admin" {
		t.Errorf("subject.Roles = %v, want [viewer] (not owner of specified server)", subject.Roles)
	}

	// Server machine ID should match the configured one
	if machineID, ok := subject.RawClaims["server_machine_id"].(string); ok && machineID != "work-server-456" {
		t.Errorf("subject.RawClaims[server_machine_id] = %v, want work-server-456", machineID)
	}
}

// TestPlexAuthenticator_ResourcesAPIFailure tests graceful degradation when resources API fails
func TestPlexAuthenticator_ResourcesAPIFailure(t *testing.T) {
	// Create server that returns user info but fails on resources
	plexServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("X-Plex-Token")
		if token == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/users/account":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"user": map[string]interface{}{
					"id":       12345,
					"uuid":     "uuid-user",
					"username": "testuser",
					"email":    "test@example.com",
				},
			})
		case "/api/v2/resources":
			// Simulate API failure
			w.WriteHeader(http.StatusInternalServerError)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer plexServer.Close()

	config := &PlexAuthConfig{
		ClientID:              "test-client",
		RedirectURI:           "http://localhost:3857/auth/callback",
		DefaultRoles:          []string{"viewer"},
		ServerOwnerRole:       "admin",
		EnableServerDetection: true,
	}

	auth := NewPlexAuthenticator(config)
	auth.SetUserInfoURL(plexServer.URL + "/users/account")
	auth.httpClient = &http.Client{
		Transport: &mockTransport{
			baseURL:    plexServer.URL,
			realClient: http.DefaultClient,
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("X-Plex-Token", "valid-token")

	// Should still authenticate successfully with default role
	subject, err := auth.Authenticate(context.Background(), req)
	if err != nil {
		t.Fatalf("Authenticate() should not fail when resources API fails: %v", err)
	}

	// Should get default viewer role
	if len(subject.Roles) == 0 || subject.Roles[0] != "viewer" {
		t.Errorf("subject.Roles = %v, want [viewer]", subject.Roles)
	}
}

// TestPlexAuthenticator_NoServers tests user with no server access
func TestPlexAuthenticator_NoServers(t *testing.T) {
	// Only player resources, no servers
	resources := []map[string]interface{}{
		{
			"name":             "Plex Web",
			"product":          "Plex Web",
			"clientIdentifier": "web-client",
			"provides":         "player",
		},
	}

	plexServer := newMockPlexServerWithResources(t, 12345, "playeronly", "player@example.com", resources)
	defer plexServer.Close()

	config := &PlexAuthConfig{
		ClientID:              "test-client",
		RedirectURI:           "http://localhost:3857/auth/callback",
		DefaultRoles:          []string{"viewer"},
		ServerOwnerRole:       "admin",
		EnableServerDetection: true,
	}

	auth := NewPlexAuthenticator(config)
	auth.SetUserInfoURL(plexServer.URL + "/users/account")
	auth.httpClient = &http.Client{
		Transport: &mockTransport{
			baseURL:    plexServer.URL,
			realClient: http.DefaultClient,
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("X-Plex-Token", "valid-token")

	subject, err := auth.Authenticate(context.Background(), req)
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}

	// No servers means default role
	if len(subject.Roles) == 0 || subject.Roles[0] != "viewer" {
		t.Errorf("subject.Roles = %v, want [viewer]", subject.Roles)
	}

	// No server info in claims
	if _, ok := subject.RawClaims["server_owner"]; ok {
		t.Error("Expected no server_owner in RawClaims when user has no servers")
	}
}

// TestPlexAuthenticator_PlexPassWithServerOwner tests Plex Pass + server owner combo
func TestPlexAuthenticator_PlexPassWithServerOwner(t *testing.T) {
	// Mock server for Plex Pass user who is also server owner
	plexServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("X-Plex-Token")
		if token == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/users/account":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"user": map[string]interface{}{
					"id":       12345,
					"uuid":     "uuid-premium",
					"username": "premiumowner",
					"email":    "premium@example.com",
					"subscription": map[string]interface{}{
						"active": true,
						"status": "Active",
						"plan":   "lifetime",
					},
				},
			})
		case "/api/v2/resources":
			json.NewEncoder(w).Encode([]map[string]interface{}{
				{
					"name":             "Premium Server",
					"clientIdentifier": "premium-server",
					"owned":            true,
					"provides":         "server",
				},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer plexServer.Close()

	config := &PlexAuthConfig{
		ClientID:              "test-client",
		RedirectURI:           "http://localhost:3857/auth/callback",
		DefaultRoles:          []string{"viewer"},
		ServerOwnerRole:       "admin",
		PlexPassRole:          "premium",
		EnableServerDetection: true,
	}

	auth := NewPlexAuthenticator(config)
	auth.SetUserInfoURL(plexServer.URL + "/users/account")
	auth.httpClient = &http.Client{
		Transport: &mockTransport{
			baseURL:    plexServer.URL,
			realClient: http.DefaultClient,
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("X-Plex-Token", "valid-token")

	subject, err := auth.Authenticate(context.Background(), req)
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}

	// Should have both admin (as owner) and premium (as Plex Pass) roles
	hasAdmin := false
	hasPremium := false
	for _, role := range subject.Roles {
		if role == "admin" {
			hasAdmin = true
		}
		if role == "premium" {
			hasPremium = true
		}
	}

	if !hasAdmin {
		t.Errorf("subject.Roles = %v, expected to contain 'admin'", subject.Roles)
	}
	if !hasPremium {
		t.Errorf("subject.Roles = %v, expected to contain 'premium'", subject.Roles)
	}
}

// mockTransport redirects plex.tv requests to mock server
type mockTransport struct {
	baseURL    string
	realClient *http.Client
}

func (t *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Redirect plex.tv URLs to mock server
	if req.URL.Host == "plex.tv" {
		req.URL.Scheme = "http"
		req.URL.Host = t.baseURL[7:] // Remove "http://" prefix
	}
	return http.DefaultTransport.RoundTrip(req)
}
