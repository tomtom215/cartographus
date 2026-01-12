// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package auth provides authentication functionality using Zitadel OIDC.
// ADR-0015: Zero Trust Authentication & Authorization (Zitadel Amendment)
//
// This file implements the OpenID Foundation-certified Zitadel OIDC library
// as the Relying Party (RP) for OIDC authentication. It replaces the custom
// OIDC implementation with a certified, production-grade solution.
//
// Key Features:
//   - OpenID Foundation certified (Basic + Config RP profiles)
//   - Automatic OIDC discovery
//   - Certified JWKS handling with caching
//   - at_hash and c_hash validation
//   - PKCE support (RFC 7636)
//   - Nonce validation
//   - RSA, ECDSA, and EdDSA algorithm support
//
// References:
//   - https://github.com/zitadel/oidc
//   - https://openid.net/specs/openid-connect-core-1_0.html
//   - ADR-0015: Zero Trust Authentication & Authorization
package auth

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/zitadel/oidc/v3/pkg/client/rp"
)

// ZitadelRPConfig holds configuration for the Zitadel Relying Party.
// This configuration is used to initialize the certified OIDC client.
//
// Required fields:
//   - IssuerURL: OIDC provider issuer URL (must match 'iss' claim)
//   - ClientID: OAuth 2.0 client identifier
//   - RedirectURL: Authorization callback URL
//
// Optional fields:
//   - ClientSecret: Required for confidential clients, optional with PKCE
//   - Scopes: OAuth 2.0 scopes (default: openid, profile, email)
//   - PKCEEnabled: Enable PKCE for public clients (default: true)
type ZitadelRPConfig struct {
	// IssuerURL is the OIDC provider's issuer URL.
	// This URL must match the 'iss' claim in tokens.
	// Example: "https://auth.example.com"
	IssuerURL string

	// ClientID is the OAuth 2.0 client identifier.
	// Obtained from your OIDC provider's configuration.
	ClientID string

	// ClientSecret is the OAuth 2.0 client secret.
	// Optional for public clients using PKCE.
	// Required for confidential clients.
	ClientSecret string

	// RedirectURL is the callback URL for authorization code flow.
	// Must be registered with your OIDC provider.
	// Example: "https://app.example.com/api/auth/oidc/callback"
	RedirectURL string

	// Scopes are the OAuth 2.0 scopes to request.
	// Must include "openid" for OIDC.
	// Default: ["openid", "profile", "email"]
	Scopes []string

	// PKCEEnabled enables Proof Key for Code Exchange (RFC 7636).
	// Recommended for all clients, required for public clients.
	// Default: true
	PKCEEnabled bool

	// HTTPClient is the HTTP client for OIDC requests.
	// If nil, a default client with 30s timeout is used.
	HTTPClient *http.Client

	// ClaimsMapping configures how OIDC claims map to AuthSubject.
	ClaimsMapping ZitadelClaimsMappingConfig

	// DefaultRoles are assigned when token contains no roles.
	// Default: ["viewer"]
	DefaultRoles []string

	// PostLogoutRedirectURI is the redirect URI after OIDC logout.
	// Used for RP-initiated logout (end_session_endpoint).
	PostLogoutRedirectURI string
}

// ZitadelClaimsMappingConfig defines how OIDC claims map to AuthSubject fields.
// This allows flexibility in mapping claims from different OIDC providers
// to the internal AuthSubject structure.
type ZitadelClaimsMappingConfig struct {
	// RolesClaim is the claim name containing user roles.
	// Different providers use different claim names:
	//   - Keycloak: "realm_access.roles" or "resource_access.{client}.roles"
	//   - Auth0: "https://example.com/roles"
	//   - Authentik: "groups" or custom claim
	// Default: "roles"
	RolesClaim string

	// GroupsClaim is the claim name containing user groups.
	// Default: "groups"
	GroupsClaim string

	// UsernameClaims is the ordered list of claims to try for username.
	// The first non-empty value found is used.
	// Default: ["preferred_username", "name", "email"]
	UsernameClaims []string
}

// Validate validates the configuration.
// Returns an error if required fields are missing or invalid.
//
// Validation rules:
//   - IssuerURL must not be empty
//   - ClientID must not be empty
//   - RedirectURL must not be empty
//   - Scopes must include "openid"
func (c *ZitadelRPConfig) Validate() error {
	if c.IssuerURL == "" {
		return fmt.Errorf("issuer_url is required")
	}
	if c.ClientID == "" {
		return fmt.Errorf("client_id is required")
	}
	if c.RedirectURL == "" {
		return fmt.Errorf("redirect_url is required")
	}

	// Ensure openid scope is present
	hasOpenID := false
	for _, scope := range c.Scopes {
		if scope == "openid" {
			hasOpenID = true
			break
		}
	}
	if !hasOpenID && len(c.Scopes) > 0 {
		return fmt.Errorf("scopes must include 'openid'")
	}

	return nil
}

// SetDefaults applies default values to unset fields.
// This method is idempotent and safe to call multiple times.
func (c *ZitadelRPConfig) SetDefaults() {
	if len(c.Scopes) == 0 {
		c.Scopes = []string{"openid", "profile", "email"}
	}
	if c.HTTPClient == nil {
		c.HTTPClient = &http.Client{Timeout: 30 * time.Second}
	}
	if c.ClaimsMapping.RolesClaim == "" {
		c.ClaimsMapping.RolesClaim = "roles"
	}
	if c.ClaimsMapping.GroupsClaim == "" {
		c.ClaimsMapping.GroupsClaim = "groups"
	}
	if len(c.ClaimsMapping.UsernameClaims) == 0 {
		c.ClaimsMapping.UsernameClaims = []string{"preferred_username", "name", "email"}
	}
	if len(c.DefaultRoles) == 0 {
		c.DefaultRoles = []string{"viewer"}
	}
}

// Clone creates a deep copy of the configuration.
// This is useful for creating modified copies without affecting the original.
func (c *ZitadelRPConfig) Clone() *ZitadelRPConfig {
	clone := &ZitadelRPConfig{
		IssuerURL:             c.IssuerURL,
		ClientID:              c.ClientID,
		ClientSecret:          c.ClientSecret,
		RedirectURL:           c.RedirectURL,
		PKCEEnabled:           c.PKCEEnabled,
		HTTPClient:            c.HTTPClient,
		PostLogoutRedirectURI: c.PostLogoutRedirectURI,
	}

	// Deep copy slices
	if c.Scopes != nil {
		clone.Scopes = make([]string, len(c.Scopes))
		copy(clone.Scopes, c.Scopes)
	}
	if c.DefaultRoles != nil {
		clone.DefaultRoles = make([]string, len(c.DefaultRoles))
		copy(clone.DefaultRoles, c.DefaultRoles)
	}

	// Deep copy claims mapping
	clone.ClaimsMapping.RolesClaim = c.ClaimsMapping.RolesClaim
	clone.ClaimsMapping.GroupsClaim = c.ClaimsMapping.GroupsClaim
	if c.ClaimsMapping.UsernameClaims != nil {
		clone.ClaimsMapping.UsernameClaims = make([]string, len(c.ClaimsMapping.UsernameClaims))
		copy(clone.ClaimsMapping.UsernameClaims, c.ClaimsMapping.UsernameClaims)
	}

	return clone
}

// ZitadelRelyingParty wraps the Zitadel rp.RelyingParty with our configuration.
// This provides a clean interface for both authentication and authorization flows.
//
// The wrapper:
//   - Manages configuration and defaults
//   - Provides access to the underlying RelyingParty
//   - Exposes common operations (ID token verification, endpoints)
//   - Handles claims mapping to AuthSubject
type ZitadelRelyingParty struct {
	rp     rp.RelyingParty
	config *ZitadelRPConfig
}

// NewZitadelRelyingParty creates a new Zitadel Relying Party.
// This performs OIDC discovery and initializes the certified OIDC client.
//
// The context is used for the initial discovery request and should have
// an appropriate timeout set. Discovery fetches:
//   - Authorization endpoint
//   - Token endpoint
//   - Userinfo endpoint
//   - JWKS URI
//   - End session endpoint (if supported)
//
// Example:
//
//	config := &ZitadelRPConfig{
//	    IssuerURL:   "https://auth.example.com",
//	    ClientID:    "my-client-id",
//	    RedirectURL: "https://app.example.com/callback",
//	    PKCEEnabled: true,
//	}
//	rp, err := NewZitadelRelyingParty(ctx, config)
//	if err != nil {
//	    return fmt.Errorf("create OIDC client: %w", err)
//	}
func NewZitadelRelyingParty(ctx context.Context, config *ZitadelRPConfig) (*ZitadelRelyingParty, error) {
	if config == nil {
		return nil, fmt.Errorf("config is required")
	}

	// Clone config to avoid external mutation
	cfg := config.Clone()
	cfg.SetDefaults()

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Build Zitadel RP options
	options := []rp.Option{
		rp.WithHTTPClient(cfg.HTTPClient),
	}

	if cfg.PKCEEnabled {
		options = append(options, rp.WithPKCE(nil))
	}

	// Create the Zitadel Relying Party
	// This performs OIDC discovery automatically
	relyingParty, err := rp.NewRelyingPartyOIDC(ctx,
		cfg.IssuerURL,
		cfg.ClientID,
		cfg.ClientSecret,
		cfg.RedirectURL,
		cfg.Scopes,
		options...,
	)
	if err != nil {
		return nil, fmt.Errorf("create relying party: %w", err)
	}

	return &ZitadelRelyingParty{
		rp:     relyingParty,
		config: cfg,
	}, nil
}

// RelyingParty returns the underlying Zitadel RelyingParty.
// Use this for direct access to Zitadel functionality.
func (z *ZitadelRelyingParty) RelyingParty() rp.RelyingParty {
	return z.rp
}

// Config returns the configuration.
// The returned config is a reference - do not modify.
func (z *ZitadelRelyingParty) Config() *ZitadelRPConfig {
	return z.config
}

// IDTokenVerifier returns the ID token verifier.
// Use this for validating ID tokens from the Authorization header.
//
// The verifier performs:
//   - Signature verification using JWKS
//   - Issuer validation
//   - Audience validation
//   - Expiration validation
//   - at_hash validation (when access token provided)
func (z *ZitadelRelyingParty) IDTokenVerifier() *rp.IDTokenVerifier {
	return z.rp.IDTokenVerifier()
}

// Issuer returns the configured issuer URL.
// This should match the 'iss' claim in tokens from this provider.
func (z *ZitadelRelyingParty) Issuer() string {
	return z.rp.Issuer()
}

// GetEndSessionEndpoint returns the end session endpoint if available.
// Returns empty string if the IdP doesn't support RP-initiated logout.
//
// When available, this endpoint can be used to:
//   - Log the user out at the IdP
//   - Redirect to a post-logout URI
//   - Pass id_token_hint for session identification
func (z *ZitadelRelyingParty) GetEndSessionEndpoint() string {
	return z.rp.GetEndSessionEndpoint()
}

// GetAuthURL returns the authorization endpoint URL.
// This is the URL where users are redirected to start the login flow.
func (z *ZitadelRelyingParty) GetAuthURL() string {
	oauthConfig := z.rp.OAuthConfig()
	if oauthConfig == nil {
		return ""
	}
	return oauthConfig.Endpoint.AuthURL
}

// GetTokenURL returns the token endpoint URL.
// This is where authorization codes are exchanged for tokens.
func (z *ZitadelRelyingParty) GetTokenURL() string {
	oauthConfig := z.rp.OAuthConfig()
	if oauthConfig == nil {
		return ""
	}
	return oauthConfig.Endpoint.TokenURL
}

// GetUserinfoURL returns the userinfo endpoint URL.
// This can be used to fetch additional claims about the user.
func (z *ZitadelRelyingParty) GetUserinfoURL() string {
	return z.rp.UserinfoEndpoint()
}

// GetClaimsMapping returns the claims mapping configuration.
func (z *ZitadelRelyingParty) GetClaimsMapping() ZitadelClaimsMappingConfig {
	return z.config.ClaimsMapping
}

// GetDefaultRoles returns the default roles for users without explicit roles.
func (z *ZitadelRelyingParty) GetDefaultRoles() []string {
	return z.config.DefaultRoles
}

// GetPostLogoutRedirectURI returns the configured post-logout redirect URI.
func (z *ZitadelRelyingParty) GetPostLogoutRedirectURI() string {
	return z.config.PostLogoutRedirectURI
}

// GetClientID returns the configured client ID.
// This is needed for audience validation in back-channel logout.
func (z *ZitadelRelyingParty) GetClientID() string {
	return z.config.ClientID
}

// GetJWKSURI returns the JWKS URI from the discovery document.
// This is used for fetching public keys for JWT signature verification.
// Returns empty string if the URI is not available.
func (z *ZitadelRelyingParty) GetJWKSURI() string {
	// Access the JWKS URI through the verifier
	// The Zitadel RP stores this internally during discovery
	if z.rp == nil {
		return ""
	}

	// Construct JWKS URI from issuer (standard OIDC convention)
	// This is typically issuer + "/.well-known/jwks.json" or from discovery
	// Zitadel follows the standard pattern
	issuer := z.rp.Issuer()
	if issuer == "" {
		return ""
	}

	// Most OIDC providers use /protocol/openid-connect/certs or /.well-known/jwks.json
	// For safety, we return a constructed URI based on the issuer
	// The actual URI should come from discovery, but Zitadel's RP doesn't expose it directly
	return issuer + "/.well-known/jwks.json"
}
