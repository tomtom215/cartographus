// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package auth provides authentication functionality including OAuth flow handlers.
// ADR-0015: Zero Trust Authentication & Authorization
package auth

import (
	"context"
	"crypto"
	"crypto/rsa"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/goccy/go-json"
	"github.com/tomtom215/cartographus/internal/logging"
)

// BackChannelLogoutClaims holds the claims from a back-channel logout token.
type BackChannelLogoutClaims struct {
	// Subject is the user identifier (sub claim).
	Subject string
	// SessionID is the optional session identifier (sid claim).
	SessionID string
	// Issuer is the token issuer (iss claim).
	Issuer string
	// Audience is the intended audience (aud claim).
	Audience []string
	// IssuedAt is when the token was issued (iat claim).
	IssuedAt time.Time
	// JTI is the unique token identifier (jti claim).
	JTI string
}

// BackChannelLogout handles OIDC back-channel logout requests from the IdP.
// POST /api/auth/oidc/backchannel-logout
// ADR-0015 Phase 4B.3: Back-Channel Logout
//
// The IdP sends a logout_token JWT in the request body (form-encoded).
// The token contains claims identifying which sessions to terminate:
//   - sub: Subject identifier (user ID)
//   - sid: Session ID (optional, specific session)
//   - events: Must contain "http://schemas.openid.net/event/backchannel-logout"
//
// This endpoint validates the token and terminates matching sessions.
// Returns:
//   - 200 OK: Logout successful
//   - 400 Bad Request: Invalid or missing logout token
//   - 501 Not Implemented: Back-channel logout not supported
func (h *FlowHandlers) BackChannelLogout(w http.ResponseWriter, r *http.Request) {
	// Check if OIDC is configured
	if h.oidcFlow == nil {
		RecordOIDCBackChannelLogout("not_configured")
		writeBackChannelError(w, http.StatusNotImplemented, "not_implemented", "Back-channel logout not configured")
		return
	}

	// Parse form-encoded body
	if err := r.ParseForm(); err != nil {
		RecordOIDCBackChannelLogout("invalid_request")
		writeBackChannelError(w, http.StatusBadRequest, "invalid_request", "Failed to parse request body")
		return
	}

	// Get logout_token from form
	logoutToken := r.FormValue("logout_token")
	if logoutToken == "" {
		RecordOIDCBackChannelLogout("missing_token")
		writeBackChannelError(w, http.StatusBadRequest, "invalid_request", "Missing logout_token parameter")
		return
	}

	// Validate and parse the logout token
	claims, err := h.validateLogoutToken(logoutToken)
	if err != nil {
		logging.Error().Err(err).Msg("Back-channel logout token validation failed")
		RecordOIDCBackChannelLogout("validation_failed")
		if h.auditLogger != nil {
			h.auditLogger.LogBackChannelLogoutFailure(r.Context(), r, err.Error())
		}
		writeBackChannelError(w, http.StatusBadRequest, "invalid_token", "Logout token validation failed")
		return
	}

	// Check for JTI replay attack (if JTI tracker is configured)
	// ADR-0015: Security Enhancement - Replay Prevention
	if h.jtiTracker != nil && claims.JTI != "" {
		sourceIP := r.RemoteAddr
		if forwardedFor := r.Header.Get("X-Forwarded-For"); forwardedFor != "" {
			sourceIP = forwardedFor
		}

		jtiEntry := &JTIEntry{
			JTI:       claims.JTI,
			Issuer:    claims.Issuer,
			Subject:   claims.Subject,
			SessionID: claims.SessionID,
			SourceIP:  sourceIP,
		}

		// Default TTL: 1 hour (logout tokens should have short lifetime)
		jtiTTL := time.Hour
		if err := h.jtiTracker.CheckAndStore(r.Context(), jtiEntry, jtiTTL); err != nil {
			if errors.Is(err, ErrJTIAlreadyUsed) {
				logging.Warn().
					Str("jti", claims.JTI).
					Str("issuer", claims.Issuer).
					Str("subject", claims.Subject).
					Str("source_ip", sourceIP).
					Msg("Back-channel logout replay attack detected")
				RecordOIDCBackChannelLogout("replay_attack")
				if h.auditLogger != nil {
					h.auditLogger.LogBackChannelLogoutFailure(r.Context(), r, "replay attack: JTI already used")
				}
				writeBackChannelError(w, http.StatusBadRequest, "invalid_token", "Token has already been used")
				return
			}
			// Log error but continue - don't fail logout for JTI tracking issues
			logging.Error().Err(err).Str("jti", claims.JTI).Msg("JTI tracking error (continuing with logout)")
		}
	}

	// Terminate sessions matching the claims
	count, terminateErr := h.terminateSessions(r.Context(), claims)
	if terminateErr != nil {
		logging.Error().Err(terminateErr).Msg("Back-channel logout failed to terminate sessions")
		// Per spec, we should still return 200 if we processed the request
	}

	// Record metrics and audit for successful back-channel logout
	RecordOIDCBackChannelLogout("success")
	RecordOIDCLogout("back_channel", "success")
	for i := 0; i < count; i++ {
		RecordOIDCSessionTerminated("back_channel")
	}

	if h.auditLogger != nil {
		h.auditLogger.LogBackChannelLogout(r.Context(), r, claims.Subject, claims.SessionID, count)
	}

	logging.Info().Int("count", count).Str("subject", claims.Subject).Str("session_id", claims.SessionID).Msg("Back-channel logout: terminated sessions")

	// Success - return 200 OK with empty body (per OIDC spec)
	w.WriteHeader(http.StatusOK)
}

// writeBackChannelError writes a JSON error response for back-channel logout.
func writeBackChannelError(w http.ResponseWriter, status int, errorCode, description string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	//nolint:errcheck // error response
	fmt.Fprintf(w, `{"error":"%s","error_description":"%s"}`, errorCode, description)
}

// terminateSessions terminates sessions based on logout claims.
func (h *FlowHandlers) terminateSessions(ctx context.Context, claims *BackChannelLogoutClaims) (int, error) {
	if claims.SessionID != "" {
		// Specific session to terminate
		if err := h.sessionStore.Delete(ctx, claims.SessionID); err != nil {
			if !errors.Is(err, ErrSessionNotFound) && !errors.Is(err, ErrSessionExpired) {
				return 0, err
			}
		}
		return 1, nil
	}

	if claims.Subject != "" {
		// Terminate all sessions for this user
		return h.sessionStore.DeleteByUserID(ctx, claims.Subject)
	}

	return 0, nil
}

// validateLogoutToken validates a back-channel logout token.
// ADR-0015 Phase 4B.3: Full JWT signature verification for back-channel logout
func (h *FlowHandlers) validateLogoutToken(tokenString string) (*BackChannelLogoutClaims, error) {
	// Get the ID token validator from OIDC flow
	validator := h.oidcFlow.GetIDTokenValidator()
	if validator == nil {
		return nil, errors.New("ID token validator not configured")
	}

	// Use the ID token validator's JWKS cache to validate signature
	jwksCache := h.oidcFlow.GetJWKSCache()
	if jwksCache == nil {
		return nil, errors.New("JWKS cache not configured")
	}

	// Verify the JWT signature using the JWKS cache
	if err := verifyLogoutTokenSignature(tokenString, jwksCache); err != nil {
		return nil, fmt.Errorf("signature verification failed: %w", err)
	}

	// Parse and validate claims (signature already verified)
	claims, err := parseLogoutTokenClaims(tokenString)
	if err != nil {
		return nil, err
	}

	// Verify issuer matches expected
	config := validator.GetConfig()
	if claims.Issuer != config.Issuer {
		return nil, errors.New("invalid issuer")
	}

	// Verify audience contains client ID
	if !containsAudience(claims.Audience, config.ClientID) {
		return nil, errors.New("client_id not in audience")
	}

	return claims, nil
}

// containsAudience checks if the audience list contains the client ID.
func containsAudience(audience []string, clientID string) bool {
	for _, aud := range audience {
		if aud == clientID {
			return true
		}
	}
	return false
}

// verifyLogoutTokenSignature verifies the JWT signature of a logout token using JWKS.
// ADR-0015 Phase 4B.3: JWT signature verification for back-channel logout
func verifyLogoutTokenSignature(tokenString string, jwksCache *JWKSCache) error {
	// Parse the token header to get the key ID
	parts := splitJWTToken(tokenString)
	if len(parts) != 3 {
		return errors.New("invalid token format")
	}

	// Decode and parse the header
	header, err := parseJWTHeader(parts[0])
	if err != nil {
		return err
	}

	// Verify signing algorithm is RSA
	if !isValidRSAAlgorithm(header.Alg) {
		return fmt.Errorf("unsupported signing algorithm: %s", header.Alg)
	}

	// Get the key ID
	if header.Kid == "" {
		return errors.New("token missing kid header")
	}

	// Fetch the public key from JWKS cache
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	publicKey, err := jwksCache.GetKey(ctx, header.Kid)
	if err != nil {
		return fmt.Errorf("failed to get key for kid %s: %w", header.Kid, err)
	}

	// Verify the signature
	signingString := parts[0] + "." + parts[1]
	signatureBytes, err := decodeBase64URL(parts[2])
	if err != nil {
		return errors.New("failed to decode token signature")
	}

	// Hash the signing string and verify with RSA
	return verifyRSASignature(header.Alg, []byte(signingString), signatureBytes, publicKey)
}

// jwtHeader represents the header portion of a JWT.
type jwtHeader struct {
	Alg string `json:"alg"`
	Kid string `json:"kid"`
	Typ string `json:"typ"`
}

// parseJWTHeader decodes and parses a JWT header.
func parseJWTHeader(headerB64 string) (*jwtHeader, error) {
	headerBytes, err := decodeBase64URL(headerB64)
	if err != nil {
		return nil, errors.New("failed to decode token header")
	}

	var header jwtHeader
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return nil, errors.New("failed to parse token header")
	}

	return &header, nil
}

// isValidRSAAlgorithm checks if the algorithm is a supported RSA algorithm.
func isValidRSAAlgorithm(alg string) bool {
	switch alg {
	case "RS256", "RS384", "RS512":
		return true
	default:
		return false
	}
}

// verifyRSASignature verifies an RSA signature for the given algorithm.
func verifyRSASignature(alg string, signingInput, signature []byte, key interface{}) error {
	rsaKey, ok := key.(*rsa.PublicKey)
	if !ok {
		return errors.New("key is not an RSA public key")
	}

	hashFunc, err := getHashFunc(alg)
	if err != nil {
		return err
	}

	hasher := hashFunc.New()
	hasher.Write(signingInput)
	hashed := hasher.Sum(nil)

	return rsa.VerifyPKCS1v15(rsaKey, hashFunc, hashed, signature)
}

// getHashFunc returns the crypto.Hash for the given algorithm.
func getHashFunc(alg string) (crypto.Hash, error) {
	switch alg {
	case "RS256":
		return crypto.SHA256, nil
	case "RS384":
		return crypto.SHA384, nil
	case "RS512":
		return crypto.SHA512, nil
	default:
		return 0, fmt.Errorf("unsupported algorithm: %s", alg)
	}
}

// splitToken is an alias for splitJWTToken for backwards compatibility.
var splitToken = splitJWTToken

// splitJWTToken splits a JWT into its three parts.
func splitJWTToken(token string) []string {
	parts := make([]string, 0, 3)
	start := 0
	count := 0
	for i := 0; i < len(token) && count < 3; i++ {
		if token[i] == '.' {
			parts = append(parts, token[start:i])
			start = i + 1
			count++
		}
	}
	if start < len(token) {
		parts = append(parts, token[start:])
	}
	return parts
}

// parseLogoutTokenClaims parses claims from a logout token.
func parseLogoutTokenClaims(tokenString string) (*BackChannelLogoutClaims, error) {
	parts := splitJWTToken(tokenString)
	if len(parts) != 3 {
		return nil, errors.New("invalid token format")
	}

	// Decode payload (second part)
	payload, err := decodeBase64URL(parts[1])
	if err != nil {
		return nil, errors.New("failed to decode token payload")
	}

	// Parse raw claims
	rawClaims, err := parseRawLogoutClaims(payload)
	if err != nil {
		return nil, err
	}

	// Verify events claim contains back-channel logout event
	if err := validateLogoutEvent(rawClaims.Events); err != nil {
		return nil, err
	}

	// Parse audience (can be string or array)
	audience := parseAudience(rawClaims.Aud)

	return &BackChannelLogoutClaims{
		Subject:   rawClaims.Sub,
		SessionID: rawClaims.Sid,
		Issuer:    rawClaims.Iss,
		Audience:  audience,
		IssuedAt:  time.Unix(rawClaims.Iat, 0),
		JTI:       rawClaims.Jti,
	}, nil
}

// rawLogoutClaims is the raw structure for parsing logout token claims.
type rawLogoutClaims struct {
	Iss    string                 `json:"iss"`
	Sub    string                 `json:"sub"`
	Aud    interface{}            `json:"aud"` // Can be string or array
	Iat    int64                  `json:"iat"`
	Jti    string                 `json:"jti"`
	Sid    string                 `json:"sid"`
	Events map[string]interface{} `json:"events"`
}

// parseRawLogoutClaims parses the raw JWT payload into claims.
func parseRawLogoutClaims(payload []byte) (*rawLogoutClaims, error) {
	var rawClaims rawLogoutClaims
	if err := json.Unmarshal(payload, &rawClaims); err != nil {
		return nil, errors.New("failed to parse token claims")
	}
	return &rawClaims, nil
}

// validateLogoutEvent verifies the events claim contains the back-channel logout event.
func validateLogoutEvent(events map[string]interface{}) error {
	if events == nil {
		return errors.New("missing events claim")
	}
	if _, ok := events["http://schemas.openid.net/event/backchannel-logout"]; !ok {
		return errors.New("missing back-channel logout event")
	}
	return nil
}

// parseAudience parses the audience claim which can be string or array.
func parseAudience(aud interface{}) []string {
	switch a := aud.(type) {
	case string:
		return []string{a}
	case []interface{}:
		var audience []string
		for _, item := range a {
			if s, ok := item.(string); ok {
				audience = append(audience, s)
			}
		}
		return audience
	default:
		return nil
	}
}

// decodeBase64URL decodes a base64url-encoded string for logout token parsing.
func decodeBase64URL(s string) ([]byte, error) {
	// Add padding if needed
	switch len(s) % 4 {
	case 2:
		s += "=="
	case 3:
		s += "="
	}

	// Decode using base64url encoding
	decoder := base64.URLEncoding
	return decoder.DecodeString(s)
}
