// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package auth provides authentication functionality using Zitadel OIDC.
// ADR-0015: Zero Trust Authentication & Authorization (Zitadel Amendment)
//
// This file implements the Authenticator interface using the OpenID Foundation-
// certified Zitadel OIDC library. It validates bearer tokens from the
// Authorization header or cookies.
//
// Security Features:
//   - Certified token signature verification
//   - Issuer validation
//   - Audience validation
//   - Expiration validation
//   - at_hash validation
//   - Algorithm confusion prevention
//
// References:
//   - https://github.com/zitadel/oidc
//   - https://openid.net/specs/openid-connect-core-1_0.html
package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/zitadel/oidc/v3/pkg/client/rp"
	"github.com/zitadel/oidc/v3/pkg/oidc"

	"github.com/tomtom215/cartographus/internal/logging"
)

// ZitadelOIDCAuthenticator implements the Authenticator interface for OpenID Connect
// using the certified Zitadel OIDC library.
//
// This authenticator:
//   - Validates bearer tokens from Authorization header or cookies
//   - Uses Zitadel's certified ID token verifier
//   - Maps OIDC claims to AuthSubject
//   - Supports configurable claim mapping
//
// Usage:
//
//	rp, err := NewZitadelRelyingParty(ctx, config)
//	if err != nil {
//	    return err
//	}
//	auth := NewZitadelOIDCAuthenticator(rp)
//	subject, err := auth.Authenticate(ctx, request)
type ZitadelOIDCAuthenticator struct {
	rp          *ZitadelRelyingParty
	tokenCookie string
}

// NewZitadelOIDCAuthenticator creates a new OIDC authenticator using Zitadel.
//
// Parameters:
//   - relyingParty: The Zitadel Relying Party (must be initialized)
//
// Returns:
//   - Authenticator that validates tokens using Zitadel's certified verifier
//
// Example:
//
//	config := &ZitadelRPConfig{...}
//	rp, _ := NewZitadelRelyingParty(ctx, config)
//	auth := NewZitadelOIDCAuthenticator(rp)
func NewZitadelOIDCAuthenticator(relyingParty *ZitadelRelyingParty) *ZitadelOIDCAuthenticator {
	return &ZitadelOIDCAuthenticator{
		rp:          relyingParty,
		tokenCookie: MetadataKeyAccessToken,
	}
}

// NewZitadelOIDCAuthenticatorWithCookie creates an authenticator with a custom cookie name.
//
// Parameters:
//   - relyingParty: The Zitadel Relying Party (must be initialized)
//   - cookieName: Name of the cookie containing the access token
//
// Returns:
//   - Authenticator configured with the specified cookie name
func NewZitadelOIDCAuthenticatorWithCookie(relyingParty *ZitadelRelyingParty, cookieName string) *ZitadelOIDCAuthenticator {
	if cookieName == "" {
		cookieName = MetadataKeyAccessToken
	}
	return &ZitadelOIDCAuthenticator{
		rp:          relyingParty,
		tokenCookie: cookieName,
	}
}

// Authenticate extracts and validates the bearer token from the request.
// This method implements the Authenticator interface.
//
// Token extraction priority:
//  1. Authorization header (Bearer token)
//  2. Cookie (configured cookie name)
//
// Validation using Zitadel's certified verifier:
//  1. Signature verification using JWKS
//  2. Issuer claim validation
//  3. Audience claim validation
//  4. Expiration validation
//  5. at_hash validation (if access token provided)
//
// Returns:
//   - AuthSubject on successful validation
//   - ErrNoCredentials if no token found
//   - ErrExpiredCredentials if token is expired
//   - ErrInvalidCredentials if token is invalid
func (a *ZitadelOIDCAuthenticator) Authenticate(ctx context.Context, r *http.Request) (*AuthSubject, error) {
	// Extract token from Authorization header or cookie
	tokenStr := a.extractToken(r)
	if tokenStr == "" {
		return nil, ErrNoCredentials
	}

	// Validate using Zitadel's certified verifier
	claims, err := a.validateToken(ctx, tokenStr)
	if err != nil {
		return nil, err
	}

	// Map claims to AuthSubject
	config := a.rp.GetClaimsMapping()
	subject := MapZitadelClaimsToAuthSubject(claims, &config, a.rp.GetDefaultRoles())

	logging.Debug().
		Str("user", subject.Username).
		Str("issuer", subject.Issuer).
		Int("roles", len(subject.Roles)).
		Msg("OIDC authentication successful")

	return subject, nil
}

// Name returns the authenticator's name for logging.
// Implements the Authenticator interface.
func (a *ZitadelOIDCAuthenticator) Name() string {
	return string(AuthModeOIDC)
}

// Priority returns the authenticator's priority for multi-mode.
// Lower values are tried first.
// Implements the Authenticator interface.
//
// OIDC has priority 10 (high priority) as it's the preferred
// enterprise authentication method.
func (a *ZitadelOIDCAuthenticator) Priority() int {
	return 10
}

// extractToken extracts the bearer token from Authorization header or cookie.
// Returns empty string if no token is found.
//
// Extraction priority:
//  1. Authorization header with "Bearer " prefix
//  2. Cookie with configured name
func (a *ZitadelOIDCAuthenticator) extractToken(r *http.Request) string {
	// Check Authorization header first
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			return strings.TrimSpace(parts[1])
		}
	}

	// Fall back to cookie
	cookie, err := r.Cookie(a.tokenCookie)
	if err == nil && cookie.Value != "" {
		return cookie.Value
	}

	return ""
}

// validateToken validates the token using Zitadel's certified verifier.
// Returns the validated claims or an error.
func (a *ZitadelOIDCAuthenticator) validateToken(ctx context.Context, tokenStr string) (*oidc.IDTokenClaims, error) {
	// Use Zitadel's certified ID token verifier
	verifier := a.rp.IDTokenVerifier()
	if verifier == nil {
		return nil, fmt.Errorf("%w: verifier not initialized", ErrAuthenticatorUnavailable)
	}

	// Verify the token
	// Zitadel's VerifyIDToken performs:
	// - Signature verification using JWKS
	// - Issuer validation
	// - Audience validation (client ID)
	// - Expiration validation
	// - Not before validation
	// - Algorithm validation (prevents confusion attacks)
	claims, err := rp.VerifyIDToken[*oidc.IDTokenClaims](ctx, tokenStr, verifier)
	if err != nil {
		// Map Zitadel errors to our error types
		return nil, a.mapVerificationError(err)
	}

	return claims, nil
}

// mapVerificationError maps Zitadel verification errors to our error types.
// This provides consistent error handling across the application.
func (a *ZitadelOIDCAuthenticator) mapVerificationError(err error) error {
	if err == nil {
		return nil
	}

	errStr := err.Error()

	// Check for expiration
	if strings.Contains(errStr, "expired") ||
		strings.Contains(errStr, "token is expired") {
		return ErrExpiredCredentials
	}

	// Check for signature/validation errors
	if strings.Contains(errStr, "signature") ||
		strings.Contains(errStr, "invalid token") ||
		strings.Contains(errStr, "verification failed") {
		logging.Debug().Err(err).Msg("Token verification failed")
		return fmt.Errorf("%w: %s", ErrInvalidCredentials, err.Error())
	}

	// Check for issuer mismatch
	if strings.Contains(errStr, "issuer") {
		logging.Warn().Err(err).Msg("Token issuer mismatch")
		return fmt.Errorf("%w: issuer mismatch", ErrInvalidCredentials)
	}

	// Check for audience mismatch
	if strings.Contains(errStr, "audience") {
		logging.Warn().Err(err).Msg("Token audience mismatch")
		return fmt.Errorf("%w: audience mismatch", ErrInvalidCredentials)
	}

	// Default to invalid credentials
	logging.Debug().Err(err).Msg("Token validation failed")
	return fmt.Errorf("%w: %s", ErrInvalidCredentials, err.Error())
}

// GetRelyingParty returns the underlying Zitadel Relying Party.
// Use this for direct access to Zitadel functionality.
func (a *ZitadelOIDCAuthenticator) GetRelyingParty() *ZitadelRelyingParty {
	return a.rp
}

// Issuer returns the OIDC issuer URL.
func (a *ZitadelOIDCAuthenticator) Issuer() string {
	return a.rp.Issuer()
}

// Compile-time interface assertion
var _ Authenticator = (*ZitadelOIDCAuthenticator)(nil)

// ZitadelOIDCAuthenticatorFromConfig creates an authenticator from OIDCConfig.
// This is a convenience function for integration with the existing configuration.
//
// Parameters:
//   - ctx: Context for OIDC discovery
//   - cfg: Existing OIDCConfig from the application configuration
//
// Returns:
//   - Configured authenticator or error
//
// Example:
//
//	auth, err := ZitadelOIDCAuthenticatorFromConfig(ctx, cfg.Security.OIDC)
//	if err != nil {
//	    return fmt.Errorf("create OIDC authenticator: %w", err)
//	}
func ZitadelOIDCAuthenticatorFromConfig(ctx context.Context, cfg *OIDCConfig) (*ZitadelOIDCAuthenticator, error) {
	// Map OIDCConfig to ZitadelRPConfig
	// OIDCConfig uses nested structures (ClaimsMapping, Session)
	rpConfig := &ZitadelRPConfig{
		IssuerURL:    cfg.IssuerURL,
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  cfg.RedirectURL,
		Scopes:       cfg.Scopes,
		PKCEEnabled:  cfg.PKCEEnabled,
		HTTPClient:   cfg.HTTPClient,
		ClaimsMapping: ZitadelClaimsMappingConfig{
			RolesClaim:     cfg.ClaimsMapping.RolesClaim,
			GroupsClaim:    cfg.ClaimsMapping.GroupsClaim,
			UsernameClaims: cfg.ClaimsMapping.UsernameClaims,
		},
		DefaultRoles: cfg.DefaultRoles,
	}

	// Create the Relying Party (performs OIDC discovery)
	relyingParty, err := NewZitadelRelyingParty(ctx, rpConfig)
	if err != nil {
		return nil, fmt.Errorf("create OIDC client: %w", err)
	}

	// Create authenticator with cookie from config (from Session.CookieName)
	cookieName := cfg.Session.CookieName
	if cookieName == "" {
		cookieName = MetadataKeyAccessToken
	}

	return NewZitadelOIDCAuthenticatorWithCookie(relyingParty, cookieName), nil
}

// Note: OIDCConfig is defined in oidc_config.go
