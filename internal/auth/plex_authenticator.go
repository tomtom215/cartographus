// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package auth provides authentication functionality including Plex OAuth support.
// ADR-0015: Zero Trust Authentication & Authorization
package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/goccy/go-json"
)

// PlexAuthConfig holds configuration for Plex authentication.
type PlexAuthConfig struct {
	// ClientID is the Plex application client identifier.
	ClientID string

	// ClientSecret is the Plex application client secret (optional).
	ClientSecret string

	// RedirectURI is the OAuth callback URL.
	RedirectURI string

	// Timeout for Plex API requests.
	Timeout time.Duration

	// DefaultRoles are assigned to all Plex-authenticated users.
	DefaultRoles []string

	// PlexPassRole is assigned to Plex Pass subscribers.
	PlexPassRole string

	// ServerOwnerRole is assigned to Plex server owners (default: admin).
	ServerOwnerRole string

	// ServerAdminRole is assigned to Plex server admins who are not owners (default: editor).
	ServerAdminRole string

	// EnableServerDetection enables automatic server ownership detection.
	// When true, the authenticator queries Plex API for server resources
	// and assigns roles based on ownership status.
	EnableServerDetection bool

	// ServerMachineIdentifier limits server detection to a specific server.
	// If empty, all servers the user has access to are checked.
	ServerMachineIdentifier string
}

// Validate checks the configuration for errors.
func (c *PlexAuthConfig) Validate() error {
	if c.ClientID == "" {
		return errors.New("plex: client ID is required")
	}
	if c.RedirectURI == "" {
		return errors.New("plex: redirect URI is required")
	}
	return nil
}

// PlexAuthenticator implements the Authenticator interface for Plex OAuth.
type PlexAuthenticator struct {
	config      *PlexAuthConfig
	httpClient  *http.Client
	userInfoURL string
}

// NewPlexAuthenticator creates a new Plex authenticator.
func NewPlexAuthenticator(config *PlexAuthConfig) *PlexAuthenticator {
	timeout := config.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &PlexAuthenticator{
		config: config,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		userInfoURL: "https://plex.tv/users/account",
	}
}

// SetUserInfoURL overrides the user info endpoint (for testing).
func (a *PlexAuthenticator) SetUserInfoURL(url string) {
	a.userInfoURL = url
}

// Authenticate extracts and validates the Plex token from the request.
// When EnableServerDetection is true, also queries Plex for server ownership
// and assigns roles accordingly (owner -> admin, shared user -> viewer).
func (a *PlexAuthenticator) Authenticate(ctx context.Context, r *http.Request) (*AuthSubject, error) {
	// Extract token from various sources
	token := a.extractToken(r)
	if token == "" {
		return nil, ErrNoCredentials
	}

	// Validate token by fetching user info from Plex
	userInfo, err := a.fetchUserInfo(ctx, token)
	if err != nil {
		return nil, err
	}

	// Check server ownership (non-blocking, graceful degradation)
	// If this fails, user still authenticates but with default roles
	var serverStatus *PlexServerStatus
	if a.config.EnableServerDetection {
		serverStatus = a.checkServerOwnership(ctx, token)
	}

	// Build AuthSubject from user info and server status
	subject := a.buildAuthSubject(userInfo, serverStatus)
	return subject, nil
}

// Name returns the authenticator name.
func (a *PlexAuthenticator) Name() string {
	return string(AuthModePlex)
}

// Priority returns the authenticator priority (lower = higher priority).
func (a *PlexAuthenticator) Priority() int {
	return 15 // Between OIDC (10) and JWT (20)
}

// extractToken extracts the Plex token from headers or cookies.
func (a *PlexAuthenticator) extractToken(r *http.Request) string {
	// Priority 1: X-Plex-Token header
	if token := r.Header.Get("X-Plex-Token"); token != "" {
		return token
	}

	// Priority 2: Authorization Bearer token
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			return parts[1]
		}
	}

	// Priority 3: Cookie
	if cookie, err := r.Cookie("plex_token"); err == nil && cookie.Value != "" {
		return cookie.Value
	}

	return ""
}

// plexUserResponse represents the Plex user API response.
type plexUserResponse struct {
	User struct {
		ID           int    `json:"id"`
		UUID         string `json:"uuid"`
		Username     string `json:"username"`
		Email        string `json:"email"`
		Thumb        string `json:"thumb"`
		Subscription struct {
			Active bool   `json:"active"`
			Status string `json:"status"`
			Plan   string `json:"plan"`
		} `json:"subscription"`
	} `json:"user"`
}

// fetchUserInfo fetches user info from Plex API.
func (a *PlexAuthenticator) fetchUserInfo(ctx context.Context, token string) (*plexUserResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.userInfoURL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set required Plex headers
	req.Header.Set("X-Plex-Token", token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Plex-Client-Identifier", a.config.ClientID)
	req.Header.Set("X-Plex-Product", "Cartographus")
	req.Header.Set("X-Plex-Version", "1.0")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, errors.Join(ErrAuthenticatorUnavailable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, ErrInvalidCredentials
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("plex API returned status %d", resp.StatusCode)
	}

	var userResp plexUserResponse
	if err := json.NewDecoder(resp.Body).Decode(&userResp); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	return &userResp, nil
}

// PlexServerStatus represents the user's relationship to a Plex server.
type PlexServerStatus struct {
	// IsOwner indicates the user owns this server.
	IsOwner bool
	// IsAdmin indicates the user has admin privileges (owner or shared admin).
	IsAdmin bool
	// ServerName is the friendly name of the server.
	ServerName string
	// MachineIdentifier is the unique ID of the server.
	MachineIdentifier string
}

// plexResourcesResponse represents the Plex resources API response.
type plexResourcesResponse []plexResource

// plexResource represents a single Plex resource (server, player, etc).
type plexResource struct {
	Name             string `json:"name"`
	Product          string `json:"product"`
	ProductVersion   string `json:"productVersion"`
	Platform         string `json:"platform"`
	ClientIdentifier string `json:"clientIdentifier"`
	Owned            bool   `json:"owned"`
	Home             bool   `json:"home"`
	AccessToken      string `json:"accessToken"`
	SourceTitle      string `json:"sourceTitle"`
	Provides         string `json:"provides"` // "server", "player", etc.
}

// fetchServerResources fetches the user's Plex server resources.
func (a *PlexAuthenticator) fetchServerResources(ctx context.Context, token string) ([]plexResource, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://plex.tv/api/v2/resources", http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create resources request: %w", err)
	}

	// Set required Plex headers
	req.Header.Set("X-Plex-Token", token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Plex-Client-Identifier", a.config.ClientID)
	req.Header.Set("X-Plex-Product", "Cartographus")
	req.Header.Set("X-Plex-Version", "1.0")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch resources: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Non-critical: return empty list on error, don't fail auth
		return nil, nil
	}

	var resources plexResourcesResponse
	if err := json.NewDecoder(resp.Body).Decode(&resources); err != nil {
		// Non-critical: return empty list on parse error
		return nil, nil
	}

	return resources, nil
}

// checkServerOwnership determines the user's server ownership status.
// Returns the highest privilege level found across all accessible servers.
func (a *PlexAuthenticator) checkServerOwnership(ctx context.Context, token string) *PlexServerStatus {
	if !a.config.EnableServerDetection {
		return nil
	}

	resources, err := a.fetchServerResources(ctx, token)
	if err != nil || len(resources) == 0 {
		return nil
	}

	var status *PlexServerStatus

	for i := range resources {
		resource := &resources[i]
		// Only consider server resources
		if !strings.Contains(resource.Provides, "server") {
			continue
		}

		// If specific server is configured, only check that one
		if a.config.ServerMachineIdentifier != "" {
			if resource.ClientIdentifier != a.config.ServerMachineIdentifier {
				continue
			}
		}

		// Track the highest privilege level found
		if resource.Owned {
			// User owns this server - highest privilege
			status = &PlexServerStatus{
				IsOwner:           true,
				IsAdmin:           true,
				ServerName:        resource.Name,
				MachineIdentifier: resource.ClientIdentifier,
			}
			// Owner is the highest level, no need to check more
			break
		} else if resource.AccessToken != "" {
			// User has access to this server (shared with them)
			// They have at least viewer access; check if admin
			// Note: Plex API doesn't explicitly expose admin status for shared users,
			// but having a valid accessToken means they have some level of access.
			// For now, we treat shared users as viewers unless they own the server.
			if status == nil {
				status = &PlexServerStatus{
					IsOwner:           false,
					IsAdmin:           false,
					ServerName:        resource.Name,
					MachineIdentifier: resource.ClientIdentifier,
				}
			}
		}
	}

	return status
}

// buildAuthSubject creates an AuthSubject from Plex user info and server status.
func (a *PlexAuthenticator) buildAuthSubject(userInfo *plexUserResponse, serverStatus *PlexServerStatus) *AuthSubject {
	user := userInfo.User

	subject := &AuthSubject{
		ID:         strconv.Itoa(user.ID),
		Username:   user.Username,
		Email:      user.Email,
		AuthMethod: AuthModePlex,
		Issuer:     "plex.tv",
		RawClaims: map[string]interface{}{
			"plex_id":   user.ID,
			"plex_uuid": user.UUID,
			"thumb":     user.Thumb,
		},
	}

	// Track server info in raw claims if detected
	if serverStatus != nil {
		subject.RawClaims["server_owner"] = serverStatus.IsOwner
		subject.RawClaims["server_admin"] = serverStatus.IsAdmin
		subject.RawClaims["server_name"] = serverStatus.ServerName
		subject.RawClaims["server_machine_id"] = serverStatus.MachineIdentifier
	}

	// Assign roles based on server ownership (highest privilege first)
	if serverStatus != nil && serverStatus.IsOwner && a.config.ServerOwnerRole != "" {
		// Server owner gets the owner role (typically admin)
		subject.Roles = []string{a.config.ServerOwnerRole}
	} else if serverStatus != nil && serverStatus.IsAdmin && a.config.ServerAdminRole != "" {
		// Server admin gets the admin role (typically editor)
		subject.Roles = []string{a.config.ServerAdminRole}
	} else if len(a.config.DefaultRoles) > 0 {
		// Regular user gets default roles
		subject.Roles = make([]string, len(a.config.DefaultRoles))
		copy(subject.Roles, a.config.DefaultRoles)
	}

	// Add Plex Pass role if applicable (in addition to primary role)
	if user.Subscription.Active && a.config.PlexPassRole != "" {
		// Check if role already exists to avoid duplicates
		hasRole := false
		for _, role := range subject.Roles {
			if role == a.config.PlexPassRole {
				hasRole = true
				break
			}
		}
		if !hasRole {
			subject.Roles = append(subject.Roles, a.config.PlexPassRole)
		}
	}

	return subject
}
