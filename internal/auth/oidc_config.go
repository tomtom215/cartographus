// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package auth

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// OIDCConfig holds the configuration for OIDC authentication.
type OIDCConfig struct {
	// IssuerURL is the OIDC provider's issuer URL.
	// Used for discovery of endpoints via .well-known/openid-configuration.
	IssuerURL string

	// ClientID is the OAuth2 client identifier.
	ClientID string

	// ClientSecret is the OAuth2 client secret.
	// Empty for public clients.
	ClientSecret string

	// RedirectURL is the callback URL for authorization code flow.
	RedirectURL string

	// Scopes to request from the OIDC provider.
	Scopes []string

	// PKCEEnabled enables Proof Key for Code Exchange.
	// REQUIRED for public clients, RECOMMENDED for all clients.
	PKCEEnabled bool

	// SkipIssuerCheck disables issuer validation.
	// WARNING: Only use for testing.
	SkipIssuerCheck bool

	// SkipExpiryCheck disables token expiry validation.
	// WARNING: Only use for testing.
	SkipExpiryCheck bool

	// HTTPClient is used for OIDC discovery and token exchange.
	// If nil, http.DefaultClient is used.
	HTTPClient *http.Client

	// JWKSCache holds JWKS caching configuration.
	JWKSCache JWKSCacheConfig

	// Session holds session management configuration.
	Session SessionConfig

	// ClaimsMapping defines how OIDC claims map to AuthSubject fields.
	ClaimsMapping ClaimsMappingConfig

	// RoleClaimPath is the JSON path to extract roles from ID token.
	// Examples: "roles", "realm_access.roles", "groups"
	RoleClaimPath string

	// DefaultRoles are assigned to all OIDC-authenticated users.
	DefaultRoles []string
}

// JWKSCacheConfig holds JWKS caching configuration.
type JWKSCacheConfig struct {
	// Enabled toggles JWKS caching.
	Enabled bool

	// TTL is how long to cache JWKS before refreshing.
	TTL time.Duration

	// Path is where to persist the JWKS cache for offline use.
	// Empty means in-memory only.
	Path string
}

// SessionConfig holds session management configuration.
type SessionConfig struct {
	// Secret is the encryption key for session data.
	// Must be exactly 32 bytes.
	Secret []byte

	// MaxAge is the session duration.
	MaxAge time.Duration

	// CookieName is the name of the session cookie.
	CookieName string

	// CookieSecure sets the Secure flag on cookies.
	CookieSecure bool

	// CookieHTTPOnly sets the HttpOnly flag on cookies.
	CookieHTTPOnly bool

	// CookieSameSite sets the SameSite attribute.
	// Valid values: "strict", "lax", "none"
	CookieSameSite string
}

// ClaimsMappingConfig defines how to extract user info from claims.
type ClaimsMappingConfig struct {
	// SubjectClaim is the claim to use as the subject ID.
	// Default: "sub"
	SubjectClaim string

	// UsernameClaims are claims to try for username, in order.
	// Default: ["preferred_username", "name", "email"]
	UsernameClaims []string

	// EmailClaim is the claim for email address.
	// Default: "email"
	EmailClaim string

	// GroupsClaim is the claim for group memberships.
	// Default: "groups"
	GroupsClaim string

	// RolesClaim is the claim for roles.
	// Default: "" (roles extracted via RoleClaimPath)
	RolesClaim string
}

// Validate checks the OIDC configuration for errors.
func (c *OIDCConfig) Validate() error {
	if c.IssuerURL == "" {
		return errors.New("oidc: issuer URL is required")
	}

	// Validate issuer URL format
	u, err := url.Parse(c.IssuerURL)
	if err != nil {
		return fmt.Errorf("oidc: invalid issuer URL: %w", err)
	}
	if u.Scheme != "https" && u.Scheme != "http" {
		return errors.New("oidc: issuer URL must use http or https scheme")
	}

	if c.ClientID == "" {
		return errors.New("oidc: client ID is required")
	}

	if c.RedirectURL == "" {
		return errors.New("oidc: redirect URL is required")
	}

	// Validate redirect URL format
	if _, err := url.Parse(c.RedirectURL); err != nil {
		return fmt.Errorf("oidc: invalid redirect URL: %w", err)
	}

	// Ensure openid scope is present
	hasOpenID := false
	for _, scope := range c.Scopes {
		if scope == "openid" {
			hasOpenID = true
			break
		}
	}
	if !hasOpenID {
		return errors.New("oidc: 'openid' scope is required")
	}

	// Validate session secret
	if len(c.Session.Secret) != 0 && len(c.Session.Secret) != 32 {
		return errors.New("oidc: session secret must be exactly 32 bytes")
	}

	return nil
}

// DefaultOIDCConfig returns a configuration with sensible defaults.
func DefaultOIDCConfig() *OIDCConfig {
	return &OIDCConfig{
		Scopes:      []string{"openid", "profile", "email"},
		PKCEEnabled: true,
		JWKSCache: JWKSCacheConfig{
			Enabled: true,
			TTL:     1 * time.Hour,
		},
		Session: SessionConfig{
			MaxAge:         24 * time.Hour,
			CookieName:     "tautulli_session",
			CookieSecure:   true,
			CookieHTTPOnly: true,
			CookieSameSite: "lax",
		},
		ClaimsMapping: ClaimsMappingConfig{
			SubjectClaim:   "sub",
			UsernameClaims: []string{"preferred_username", "name", "email"},
			EmailClaim:     "email",
			GroupsClaim:    "groups",
		},
	}
}

// GenerateSessionSecret generates a cryptographically secure session secret.
func GenerateSessionSecret() ([]byte, error) {
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return nil, fmt.Errorf("failed to generate session secret: %w", err)
	}
	return secret, nil
}

// EncodeSessionSecret encodes a session secret to base64 for configuration.
func EncodeSessionSecret(secret []byte) string {
	return base64.StdEncoding.EncodeToString(secret)
}

// DecodeSessionSecret decodes a base64-encoded session secret.
func DecodeSessionSecret(encoded string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(encoded)
}
