// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package api

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/goccy/go-json"

	"github.com/tomtom215/cartographus/internal/models"
)

// plexUserInfo holds user information fetched from Plex API.
type plexUserInfo struct {
	ID       int    `json:"id"`
	UUID     string `json:"uuid"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Thumb    string `json:"thumb"`
	PlexPass bool   `json:"plexPass"`
}

// fetchPlexUserInfo fetches user information from Plex API using the access token.
func (h *Handler) fetchPlexUserInfo(ctx context.Context, accessToken string) (*plexUserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://plex.tv/users/account", http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set required Plex headers
	req.Header.Set("X-Plex-Token", accessToken)
	req.Header.Set("Accept", "application/json")
	// Set client identifier from config if available
	if h.config != nil && h.config.Plex.OAuthClientID != "" {
		req.Header.Set("X-Plex-Client-Identifier", h.config.Plex.OAuthClientID)
	}
	req.Header.Set("X-Plex-Product", "Cartographus")
	req.Header.Set("X-Plex-Version", "1.0")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("plex API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("plex token is invalid or expired")
	}

	if resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return nil, fmt.Errorf("plex API returned status %d (failed to read body: %w)", resp.StatusCode, readErr)
		}
		return nil, fmt.Errorf("plex API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse Plex user response
	var userResp struct {
		User struct {
			ID           int    `json:"id"`
			UUID         string `json:"uuid"`
			Username     string `json:"username"`
			Email        string `json:"email"`
			Thumb        string `json:"thumb"`
			Subscription struct {
				Active bool `json:"active"`
			} `json:"subscription"`
		} `json:"user"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&userResp); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	return &plexUserInfo{
		ID:       userResp.User.ID,
		UUID:     userResp.User.UUID,
		Username: userResp.User.Username,
		Email:    userResp.User.Email,
		Thumb:    userResp.User.Thumb,
		PlexPass: userResp.User.Subscription.Active,
	}, nil
}

// PlexOAuthStart initiates the Plex OAuth 2.0 PKCE flow.
//
// Endpoint: GET /api/v1/auth/plex/start
//
// Response:
//
//	{
//	  "status": "success",
//	  "data": {
//	    "authorization_url": "https://app.plex.tv/auth#?clientID=...",
//	    "state": "random-csrf-token"
//	  }
//	}
//
// Workflow:
//  1. Generate PKCE code_verifier and code_challenge
//  2. Generate random CSRF state token
//  3. Store code_verifier and state in session (httpOnly cookie)
//  4. Build Plex authorization URL with code_challenge
//  5. Return authorization URL to frontend
//  6. Frontend redirects user to authorization URL
//
// Security: PKCE prevents authorization code interception, state prevents CSRF.
func (h *Handler) PlexOAuthStart(w http.ResponseWriter, r *http.Request) {
	// Check if OAuth client is configured
	if h.plexOAuthClient == nil {
		respondError(w, http.StatusServiceUnavailable, "OAUTH_NOT_CONFIGURED", "Plex OAuth is not configured. Set PLEX_OAUTH_CLIENT_ID and PLEX_OAUTH_REDIRECT_URI environment variables.", nil)
		return
	}

	// Generate PKCE challenge
	pkce, err := h.plexOAuthClient.GeneratePKCE()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "OAUTH_ERROR", "Failed to generate PKCE challenge", err)
		return
	}

	// Generate random state token for CSRF protection
	stateBytes := make([]byte, 32)
	if _, err := rand.Read(stateBytes); err != nil {
		respondError(w, http.StatusInternalServerError, "OAUTH_ERROR", "Failed to generate state token", err)
		return
	}
	state := base64.RawURLEncoding.EncodeToString(stateBytes)

	// Store PKCE verifier and state in HTTP-only cookie (valid for 10 minutes)
	oauthData := map[string]string{
		"code_verifier": pkce.CodeVerifier,
		"state":         state,
	}
	oauthDataJSON, err := json.Marshal(oauthData)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "OAUTH_ERROR", "Failed to encode OAuth state", err)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "plex_oauth_state",
		Value:    base64.StdEncoding.EncodeToString(oauthDataJSON),
		Path:     "/",
		MaxAge:   600, // 10 minutes
		HttpOnly: true,
		Secure:   r.TLS != nil, // Only set Secure flag if HTTPS
		SameSite: http.SameSiteLaxMode,
	})

	// Build authorization URL
	authURL := h.plexOAuthClient.BuildAuthorizationURL(pkce.CodeChallenge, state)

	// Return authorization URL
	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data: map[string]interface{}{
			"authorization_url": authURL,
			"state":             state, // Frontend can verify this matches callback
		},
		Metadata: models.Metadata{
			Timestamp: time.Now().UTC(),
		},
	})
}

// PlexOAuthCallback handles the OAuth callback from Plex.
//
// Endpoint: GET /api/v1/auth/plex/callback?code=...&state=...
//
// Query Parameters:
//   - code: Authorization code from Plex
//   - state: CSRF token (must match state from /start)
//
// Response:
//
//	{
//	  "status": "success",
//	  "data": {
//	    "token": "jwt-session-token",
//	    "expires_at": "2025-03-13T12:00:00Z",
//	    "user": {
//	      "id": 12345,
//	      "uuid": "abc-123",
//	      "username": "plexuser",
//	      "email": "user@example.com",
//	      "thumb": "https://plex.tv/users/abc/avatar",
//	      "plex_pass": true,
//	      "role": "plex_pass"
//	    },
//	    "plex_token": {
//	      "expires_at": 1765746983,
//	      "expires_in": 7776000
//	    }
//	  }
//	}
//
// Workflow:
//  1. Validate state parameter matches cookie (CSRF check)
//  2. Retrieve code_verifier from cookie
//  3. Exchange authorization code + code_verifier for access token
//  4. Fetch Plex user information using access token
//  5. Generate JWT session token for our application
//  6. Set JWT session cookie and Plex token cookie
//  7. Return user information and session token
//
// Security: Validates state token, uses PKCE code_verifier, sets HTTP-only cookies.
func (h *Handler) PlexOAuthCallback(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	if code == "" {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Missing authorization code", nil)
		return
	}

	// Retrieve OAuth state from cookie
	cookie, err := r.Cookie("plex_oauth_state")
	if err != nil {
		respondError(w, http.StatusUnauthorized, "INVALID_STATE", "OAuth state cookie not found", err)
		return
	}

	// Decode cookie data
	oauthDataJSON, err := base64.StdEncoding.DecodeString(cookie.Value)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "INVALID_STATE", "Failed to decode OAuth state", err)
		return
	}

	var oauthData map[string]string
	if err := json.Unmarshal(oauthDataJSON, &oauthData); err != nil {
		respondError(w, http.StatusUnauthorized, "INVALID_STATE", "Failed to parse OAuth state", err)
		return
	}

	// Validate state parameter (CSRF protection)
	if state != oauthData["state"] {
		respondError(w, http.StatusUnauthorized, "INVALID_STATE", "State parameter mismatch", nil)
		return
	}

	// Check if OAuth client is configured
	if h.plexOAuthClient == nil {
		respondError(w, http.StatusServiceUnavailable, "OAUTH_NOT_CONFIGURED", "Plex OAuth is not configured", nil)
		return
	}

	// Exchange authorization code for access token
	plexToken, err := h.plexOAuthClient.ExchangeCodeForToken(r.Context(), code, oauthData["code_verifier"])
	if err != nil {
		respondError(w, http.StatusUnauthorized, "TOKEN_EXCHANGE_FAILED", fmt.Sprintf("Failed to exchange code: %v", err), err)
		return
	}

	// Fetch Plex user information with access token
	userInfo, err := h.fetchPlexUserInfo(r.Context(), plexToken.AccessToken)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "USER_INFO_FAILED", fmt.Sprintf("Failed to fetch user info: %v", err), err)
		return
	}

	// Determine user role based on Plex Pass status
	role := "plex_user"
	if userInfo.PlexPass {
		role = "plex_pass"
	}

	// Generate JWT session token for our application
	jwtToken, err := h.jwtManager.GenerateToken(userInfo.Username, role)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "TOKEN_GENERATION_FAILED", "Failed to generate session token", err)
		return
	}

	// Calculate session expiration
	sessionExpiry := time.Now().Add(h.config.Security.SessionTimeout)

	// Clear OAuth state cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "plex_oauth_state",
		Value:    "",
		Path:     "/",
		MaxAge:   -1, // Delete cookie
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
	})

	// Set JWT session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    jwtToken,
		Path:     "/",
		Expires:  sessionExpiry,
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteStrictMode,
	})

	// Store Plex token in separate HTTP-only cookie for Plex API calls
	// This allows the frontend to make authenticated Plex API requests
	http.SetCookie(w, &http.Cookie{
		Name:     "plex_token",
		Value:    plexToken.AccessToken,
		Path:     "/",
		Expires:  time.Unix(plexToken.ExpiresAt, 0),
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteStrictMode,
	})

	// Return success with user information
	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data: map[string]interface{}{
			"token":      jwtToken,
			"expires_at": sessionExpiry,
			"user": map[string]interface{}{
				"id":        userInfo.ID,
				"uuid":      userInfo.UUID,
				"username":  userInfo.Username,
				"email":     userInfo.Email,
				"thumb":     userInfo.Thumb,
				"plex_pass": userInfo.PlexPass,
				"role":      role,
			},
			"plex_token": map[string]interface{}{
				"expires_at": plexToken.ExpiresAt,
				"expires_in": plexToken.ExpiresIn,
			},
		},
		Metadata: models.Metadata{
			Timestamp: time.Now().UTC(),
		},
	})
}

// PlexOAuthRefresh refreshes a Plex OAuth access token.
//
// Endpoint: POST /api/v1/auth/plex/refresh
//
// Request Body:
//
//	{
//	  "refresh_token": "..."
//	}
//
// Response:
//
//	{
//	  "status": "success",
//	  "data": {
//	    "access_token": "...",
//	    "expires_in": 7776000,
//	    "token_type": "Bearer"
//	  }
//	}
//
// Workflow:
//  1. Parse refresh_token from request body
//  2. Call Plex token refresh endpoint
//  3. Update stored tokens in database
//  4. Return new access_token
//
// Use Case: Called automatically when access token expires (typically after 90 days).
func (h *Handler) PlexOAuthRefresh(w http.ResponseWriter, r *http.Request) {
	// Parse request body
	var requestBody struct {
		RefreshToken string `json:"refresh_token"`
	}

	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid JSON body", err)
		return
	}

	if requestBody.RefreshToken == "" {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Missing refresh_token", nil)
		return
	}

	// Check if OAuth client is configured
	if h.plexOAuthClient == nil {
		respondError(w, http.StatusServiceUnavailable, "OAUTH_NOT_CONFIGURED", "Plex OAuth is not configured", nil)
		return
	}

	// Refresh access token
	plexToken, err := h.plexOAuthClient.RefreshToken(r.Context(), requestBody.RefreshToken)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "REFRESH_FAILED", fmt.Sprintf("Failed to refresh token: %v", err), err)
		return
	}

	// Update stored Plex token cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "plex_token",
		Value:    plexToken.AccessToken,
		Path:     "/",
		Expires:  time.Unix(plexToken.ExpiresAt, 0),
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteStrictMode,
	})

	// Return new token information
	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data: map[string]interface{}{
			"access_token":  plexToken.AccessToken,
			"refresh_token": plexToken.RefreshToken,
			"expires_in":    plexToken.ExpiresIn,
			"token_type":    plexToken.TokenType,
			"expires_at":    plexToken.ExpiresAt,
		},
		Metadata: models.Metadata{
			Timestamp: time.Now().UTC(),
		},
	})
}

// PlexOAuthRevoke revokes a Plex OAuth access token.
//
// Endpoint: POST /api/v1/auth/plex/revoke
//
// Request Body:
//
//	{
//	  "access_token": "..."
//	}
//
// Response:
//
//	{
//	  "status": "success",
//	  "data": {
//	    "revoked": true
//	  }
//	}
//
// Workflow:
//  1. Parse access_token from request body
//  2. TODO: Call Plex token revocation endpoint (if available)
//  3. Delete token from database
//  4. Clear session cookies
//
// Use Case: User logs out or disconnects Plex account.
func (h *Handler) PlexOAuthRevoke(w http.ResponseWriter, r *http.Request) {
	// Parse request body
	var requestBody struct {
		AccessToken string `json:"access_token"`
	}

	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid JSON body", err)
		return
	}

	if requestBody.AccessToken == "" {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Missing access_token", nil)
		return
	}

	// Note: Plex does not have a public token revocation endpoint.
	// The token will naturally expire after its lifetime (~90 days).
	// We clear all local session state to log the user out.

	// Clear JWT session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteStrictMode,
	})

	// Clear Plex token cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "plex_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteStrictMode,
	})

	// Return success
	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data: map[string]interface{}{
			"revoked": true,
			"message": "Session cleared. Plex token will expire naturally.",
		},
		Metadata: models.Metadata{
			Timestamp: time.Now().UTC(),
		},
	})
}
