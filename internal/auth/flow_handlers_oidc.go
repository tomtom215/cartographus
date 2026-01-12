// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package auth provides authentication functionality including OAuth flow handlers.
// ADR-0015: Zero Trust Authentication & Authorization
package auth

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/goccy/go-json"
	"github.com/tomtom215/cartographus/internal/logging"
)

// validateRedirectURI validates that a redirect URI is safe to use.
// It only allows relative paths starting with "/" to prevent open redirect attacks.
// Returns the validated URI or an empty string if invalid.
//
// Security: This function prevents open redirect vulnerabilities by ensuring
// that redirect URIs cannot point to external domains. Only relative paths
// are allowed, and paths starting with "//" (protocol-relative URLs) are rejected.
//
// Valid examples:
//   - "/" → "/"
//   - "/dashboard" → "/dashboard"
//   - "/app/settings" → "/app/settings"
//
// Invalid examples (return ""):
//   - "https://evil.com" → ""
//   - "//evil.com/path" → ""
//   - "http://example.com/path" → ""
//   - "javascript:alert(1)" → ""
func validateRedirectURI(uri string) string {
	// Empty string is valid (will use default)
	if uri == "" {
		return uri
	}

	// Must start with "/" for relative path
	if !strings.HasPrefix(uri, "/") {
		logging.Warn().Str("redirect_uri", uri).Msg("Rejected redirect URI: must be relative path starting with /")
		return ""
	}

	// Reject protocol-relative URLs (//example.com)
	if strings.HasPrefix(uri, "//") {
		logging.Warn().Str("redirect_uri", uri).Msg("Rejected redirect URI: protocol-relative URLs not allowed")
		return ""
	}

	// Additional validation: parse as URL to ensure it's well-formed
	// Using ParseRequestURI to be strict about validation
	if _, err := url.ParseRequestURI(uri); err != nil {
		logging.Warn().Str("redirect_uri", uri).Err(err).Msg("Rejected redirect URI: malformed URL")
		return ""
	}

	return uri
}

// OIDCLogin initiates the OIDC authorization flow.
// GET /api/auth/oidc/login
func (h *FlowHandlers) OIDCLogin(w http.ResponseWriter, r *http.Request) {
	if h.oidcFlow == nil {
		http.Error(w, "OIDC authentication not configured", http.StatusServiceUnavailable)
		return
	}

	// Get optional redirect_uri query parameter and validate it
	postLoginRedirect := r.URL.Query().Get("redirect_uri")
	postLoginRedirect = validateRedirectURI(postLoginRedirect)
	if postLoginRedirect == "" {
		postLoginRedirect = h.config.DefaultPostLoginRedirect
	}

	// Generate authorization URL with state and PKCE
	authURL, err := h.oidcFlow.AuthorizationURL(r.Context(), postLoginRedirect)
	if err != nil {
		logging.Error().Err(err).Msg("Failed to generate OIDC auth URL")
		http.Error(w, "Failed to initiate login", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, authURL, http.StatusFound)
}

// OIDCCallback handles the OIDC authorization callback.
// GET /api/auth/oidc/callback
func (h *FlowHandlers) OIDCCallback(w http.ResponseWriter, r *http.Request) {
	callbackStart := time.Now()

	if h.oidcFlow == nil {
		http.Error(w, "OIDC authentication not configured", http.StatusServiceUnavailable)
		return
	}

	// Check for error from IdP
	if errParam := r.URL.Query().Get("error"); errParam != "" {
		errDesc := r.URL.Query().Get("error_description")
		logging.Error().Str("error", errParam).Str("error_description", errDesc).Msg("OIDC error from provider")

		// Record metrics and audit
		RecordOIDCLogin("oidc", "failure", time.Since(callbackStart))
		if h.auditLogger != nil {
			h.auditLogger.LogLoginFailure(r.Context(), r, errParam, errDesc)
		}

		http.Redirect(w, r, h.config.ErrorRedirectURL+"oidc_error", http.StatusFound)
		return
	}

	// Get code and state from query parameters
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	if code == "" || state == "" {
		logging.Error().Msg("OIDC callback missing code or state")

		// Record metrics and audit
		RecordOIDCLogin("oidc", "failure", time.Since(callbackStart))
		if h.auditLogger != nil {
			h.auditLogger.LogLoginFailure(r.Context(), r, "missing_params", "Code or state parameter missing")
		}

		http.Redirect(w, r, h.config.ErrorRedirectURL+"missing_params", http.StatusFound)
		return
	}

	// Handle callback (validates state, exchanges code for tokens)
	exchangeStart := time.Now()
	result, err := h.oidcFlow.HandleCallback(r.Context(), code, state)
	RecordOIDCTokenExchange("oidc", time.Since(exchangeStart))

	if err != nil {
		logging.Error().Err(err).Msg("OIDC callback failed")

		// Record metrics and audit
		RecordOIDCLogin("oidc", "failure", time.Since(callbackStart))
		if h.auditLogger != nil {
			h.auditLogger.LogLoginFailure(r.Context(), r, "callback_failed", err.Error())
		}

		http.Redirect(w, r, h.config.ErrorRedirectURL+"callback_failed", http.StatusFound)
		return
	}

	// Create session and redirect (metrics recorded in completeOIDCCallback)
	h.completeOIDCCallback(w, r, result, callbackStart)
}

// completeOIDCCallback creates a session and redirects after successful OIDC callback.
// ADR-0015: Zero Trust Authentication & Authorization (Zitadel Amendment)
// Uses ZitadelTokenResult from the certified Zitadel OIDC library.
func (h *FlowHandlers) completeOIDCCallback(w http.ResponseWriter, r *http.Request, result *ZitadelTokenResult, callbackStart time.Time) {
	// Create AuthSubject from result
	subject := &AuthSubject{
		AuthMethod: AuthModeOIDC,
		Provider:   "oidc",
		Metadata: map[string]string{
			"access_token":  result.AccessToken,
			"refresh_token": result.RefreshToken,
		},
	}

	// If result includes subject info, use it
	if result.Subject != nil {
		subject = result.Subject
		subject.Metadata = map[string]string{
			"access_token":  result.AccessToken,
			"refresh_token": result.RefreshToken,
		}
	}

	// Extract old session ID from cookie for session fixation protection (Phase 4A.3)
	oldSessionID := h.extractSessionIDFromCookie(r)

	// Create session with session fixation protection
	session, err := h.sessionMiddleware.CreateSessionWithOldID(r.Context(), w, subject, oldSessionID)
	if err != nil {
		logging.Error().Err(err).Msg("Failed to create session")

		// Record metrics and audit for session creation failure
		RecordOIDCLogin("oidc", "error", time.Since(callbackStart))
		if h.auditLogger != nil {
			h.auditLogger.LogLoginFailure(r.Context(), r, "session_error", err.Error())
		}

		http.Redirect(w, r, h.config.ErrorRedirectURL+"session_error", http.StatusFound)
		return
	}

	// Record successful login metrics and audit
	duration := time.Since(callbackStart)
	RecordOIDCLogin("oidc", "success", duration)
	RecordOIDCSessionCreated("oidc")

	if h.auditLogger != nil {
		h.auditLogger.LogLoginSuccess(r.Context(), r, subject, duration)
		h.auditLogger.LogSessionCreated(r.Context(), r, session)
	}

	logging.Info().Str("user", subject.Username).Str("session_id", session.ID).Str("old_session_id", oldSessionID).Msg("OIDC login successful")

	// Redirect to post-login URL with validation
	redirectURL := validateRedirectURI(result.PostLoginRedirect)
	if redirectURL == "" {
		redirectURL = h.config.DefaultPostLoginRedirect
	}

	http.Redirect(w, r, redirectURL, http.StatusFound)
}

// OIDCRefreshRequest is the request body for token refresh.
type OIDCRefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// OIDCRefreshResponse is the response body for token refresh.
type OIDCRefreshResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

// OIDCRefresh exchanges a refresh token for new tokens.
// POST /api/auth/oidc/refresh
func (h *FlowHandlers) OIDCRefresh(w http.ResponseWriter, r *http.Request) {
	refreshStart := time.Now()

	if h.oidcFlow == nil {
		http.Error(w, "OIDC authentication not configured", http.StatusServiceUnavailable)
		return
	}

	// Parse request body
	var req OIDCRefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid_request","error_description":"Invalid JSON body"}`, http.StatusBadRequest)
		return
	}

	if req.RefreshToken == "" {
		http.Error(w, `{"error":"invalid_request","error_description":"refresh_token is required"}`, http.StatusBadRequest)
		return
	}

	// Exchange refresh token for new tokens
	result, err := h.oidcFlow.RefreshToken(r.Context(), req.RefreshToken)
	duration := time.Since(refreshStart)

	if err != nil {
		logging.Error().Err(err).Msg("OIDC token refresh failed")

		// Record metrics and audit for failed refresh
		RecordOIDCTokenRefresh("oidc", "failure", duration)
		subject := GetAuthSubject(r.Context())
		if h.auditLogger != nil && subject != nil {
			h.auditLogger.LogTokenRefresh(r.Context(), r, subject.ID, subject.Username, false, duration, time.Time{})
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		//nolint:errcheck // error response
		w.Write([]byte(`{"error":"invalid_grant","error_description":"Refresh token expired or invalid"}`))
		return
	}

	// Update session with new tokens if user is authenticated
	h.updateSessionTokens(r, result)

	// Record metrics and audit for successful refresh
	RecordOIDCTokenRefresh("oidc", "success", duration)
	subject := GetAuthSubject(r.Context())
	if h.auditLogger != nil && subject != nil {
		newExpiry := time.Now().Add(time.Duration(result.ExpiresIn) * time.Second)
		h.auditLogger.LogTokenRefresh(r.Context(), r, subject.ID, subject.Username, true, duration, newExpiry)
	}

	// Return new tokens
	resp := OIDCRefreshResponse{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		ExpiresIn:    result.ExpiresIn,
		TokenType:    result.TokenType,
	}
	if resp.TokenType == "" {
		resp.TokenType = "Bearer"
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		logging.Error().Err(err).Msg("Failed to encode refresh response")
	}
}

// updateSessionTokens updates the session with new tokens after refresh.
// ADR-0015: Zero Trust Authentication & Authorization (Zitadel Amendment)
// Uses ZitadelTokenResult from the certified Zitadel OIDC library.
func (h *FlowHandlers) updateSessionTokens(r *http.Request, result *ZitadelTokenResult) {
	subject := GetAuthSubject(r.Context())
	if subject == nil || subject.SessionID == "" {
		return
	}

	session, err := h.sessionStore.Get(r.Context(), subject.SessionID)
	if err != nil || session == nil {
		return
	}

	if session.Metadata == nil {
		session.Metadata = make(map[string]string)
	}
	session.Metadata["access_token"] = result.AccessToken
	if result.RefreshToken != "" {
		session.Metadata["refresh_token"] = result.RefreshToken
	}
	//nolint:errcheck // best effort session update
	h.sessionStore.Update(r.Context(), session)
}

// OIDCLogoutRequest is the optional request body for logout.
type OIDCLogoutRequest struct {
	// PostLogoutRedirectURI overrides the default redirect URI.
	PostLogoutRedirectURI string `json:"post_logout_redirect_uri,omitempty"`
}

// OIDCLogoutResponse is the response for logout operations.
type OIDCLogoutResponse struct {
	// Message describes the logout result.
	Message string `json:"message"`
	// RedirectURL is the IdP logout URL (if RP-initiated logout is supported).
	RedirectURL string `json:"redirect_url,omitempty"`
}

// OIDCLogout initiates RP-initiated logout with the OIDC provider.
// POST /api/auth/oidc/logout
// ADR-0015 Phase 4B.2: RP-Initiated Logout
//
// If the IdP supports RP-initiated logout (end_session_endpoint), this handler:
//  1. Destroys the local session
//  2. Redirects to the IdP's end_session_endpoint with:
//     - id_token_hint: The user's ID token (if available)
//     - post_logout_redirect_uri: Where to redirect after IdP logout
//     - state: Optional state for security
//
// If the IdP does not support RP-initiated logout, only local session is destroyed.
func (h *FlowHandlers) OIDCLogout(w http.ResponseWriter, r *http.Request) {
	subject := GetAuthSubject(r.Context())
	postLogoutRedirectURI := h.getPostLogoutRedirectURI(r)
	idTokenHint := h.getIDTokenHint(r.Context(), subject)

	// Record metrics and audit before destroying session
	RecordOIDCLogout("rp_initiated", "success")
	RecordOIDCSessionTerminated("logout")

	if h.auditLogger != nil && subject != nil {
		h.auditLogger.LogLogout(r.Context(), r, subject.ID, subject.Username, subject.SessionID, idTokenHint != "")
	}

	// Destroy local session
	h.destroyLocalSession(r.Context(), w, subject)

	// Build and send logout response
	h.sendLogoutResponse(w, r, idTokenHint, postLogoutRedirectURI)
}

// getPostLogoutRedirectURI determines the post-logout redirect URI from request or config.
// It validates the URI to prevent open redirect attacks.
func (h *FlowHandlers) getPostLogoutRedirectURI(r *http.Request) string {
	var req OIDCLogoutRequest
	if r.Body != nil && r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			// Log but don't fail - body is optional, use default redirect
			logging.Debug().Err(err).Msg("failed to parse optional logout request body, using defaults")
		}
	}

	// Validate user-provided redirect URI
	validatedURI := validateRedirectURI(req.PostLogoutRedirectURI)
	if validatedURI != "" {
		return validatedURI
	}
	if h.config.PostLogoutRedirectURI != "" {
		return h.config.PostLogoutRedirectURI
	}
	return "/"
}

// sendLogoutResponse sends the appropriate logout response based on IdP configuration.
func (h *FlowHandlers) sendLogoutResponse(w http.ResponseWriter, r *http.Request, idTokenHint, postLogoutRedirectURI string) {
	// Check if IdP supports RP-initiated logout
	if h.oidcFlow == nil {
		h.writeLogoutJSON(w, "Logged out successfully (local only)", "")
		return
	}

	endSessionEndpoint := h.oidcFlow.GetEndSessionEndpoint()
	if endSessionEndpoint == "" {
		h.writeLogoutJSON(w, "Logged out successfully (IdP logout not supported)", "")
		return
	}

	// Build IdP logout URL
	logoutURLStr, err := h.buildLogoutURL(endSessionEndpoint, idTokenHint, postLogoutRedirectURI)
	if err != nil {
		logging.Error().Err(err).Msg("Failed to build logout URL")
		h.writeLogoutJSON(w, "Logged out successfully (IdP logout URL invalid)", "")
		return
	}

	// Return redirect URL for client-side redirect or redirect directly
	if h.isAJAXRequest(r) {
		h.writeLogoutJSON(w, "Logged out successfully", logoutURLStr)
		return
	}

	http.Redirect(w, r, logoutURLStr, http.StatusFound)
}

// buildLogoutURL constructs the IdP logout URL with query parameters.
func (h *FlowHandlers) buildLogoutURL(endSessionEndpoint, idTokenHint, postLogoutRedirectURI string) (string, error) {
	logoutURL, err := url.Parse(endSessionEndpoint)
	if err != nil {
		return "", err
	}

	params := logoutURL.Query()
	if idTokenHint != "" {
		params.Set("id_token_hint", idTokenHint)
	}
	if postLogoutRedirectURI != "" {
		params.Set("post_logout_redirect_uri", postLogoutRedirectURI)
	}
	if state, err := GenerateStateParameter(); err == nil {
		params.Set("state", state)
	}
	logoutURL.RawQuery = params.Encode()

	return logoutURL.String(), nil
}

// writeLogoutJSON writes a JSON logout response.
func (h *FlowHandlers) writeLogoutJSON(w http.ResponseWriter, message, redirectURL string) {
	w.Header().Set("Content-Type", "application/json")
	resp := OIDCLogoutResponse{Message: message, RedirectURL: redirectURL}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		logging.Error().Err(err).Msg("Failed to encode logout response")
	}
}
