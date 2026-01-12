// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package auth provides authentication functionality including OIDC support.
// ADR-0015: Zero Trust Authentication & Authorization
package auth

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/goccy/go-json"
	"github.com/golang-jwt/jwt/v5"
)

// OIDCAuthenticator implements the Authenticator interface for OpenID Connect.
// It validates bearer tokens using JWKS (JSON Web Key Set) from the IdP.
type OIDCAuthenticator struct {
	config      *OIDCConfig
	issuer      string
	jwksURI     string
	jwksCache   *jwksCache
	httpClient  *http.Client
	tokenCookie string
}

// NewOIDCAuthenticator creates a new OIDC authenticator.
// It performs OIDC discovery to fetch the provider configuration.
func NewOIDCAuthenticator(ctx context.Context, config *OIDCConfig) (*OIDCAuthenticator, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid OIDC config: %w", err)
	}

	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Perform OIDC discovery
	discoveryURL := strings.TrimSuffix(config.IssuerURL, "/") + "/.well-known/openid-configuration"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, discoveryURL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create discovery request: %w", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("OIDC discovery failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return nil, fmt.Errorf("OIDC discovery returned status %d", resp.StatusCode)
		}
		return nil, fmt.Errorf("OIDC discovery returned status %d: %s", resp.StatusCode, string(body))
	}

	var discovery struct {
		Issuer  string `json:"issuer"`
		JWKSURI string `json:"jwks_uri"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&discovery); err != nil {
		return nil, fmt.Errorf("failed to parse discovery document: %w", err)
	}

	if discovery.JWKSURI == "" {
		return nil, errors.New("OIDC discovery missing jwks_uri")
	}

	// Set up JWKS cache TTL
	cacheTTL := config.JWKSCache.TTL
	if cacheTTL == 0 {
		cacheTTL = 15 * time.Minute // Default TTL
	}

	auth := &OIDCAuthenticator{
		config:      config,
		issuer:      discovery.Issuer,
		jwksURI:     discovery.JWKSURI,
		httpClient:  httpClient,
		tokenCookie: "access_token",
	}

	// Initialize JWKS cache
	auth.jwksCache = newJWKSCache(auth.jwksURI, httpClient, cacheTTL)

	// Pre-fetch JWKS
	if _, err := auth.jwksCache.getKeys(ctx); err != nil {
		return nil, fmt.Errorf("failed to fetch JWKS: %w", err)
	}

	return auth, nil
}

// Authenticate extracts and validates the bearer token from the request.
func (a *OIDCAuthenticator) Authenticate(ctx context.Context, r *http.Request) (*AuthSubject, error) {
	// Extract token from Authorization header or cookie
	tokenStr := a.extractToken(r)
	if tokenStr == "" {
		return nil, ErrNoCredentials
	}

	// Parse and validate the token
	token, err := a.parseAndValidateToken(ctx, tokenStr)
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredCredentials
		}
		return nil, ErrInvalidCredentials
	}

	// Extract claims and build AuthSubject
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, ErrInvalidCredentials
	}

	subject := a.buildAuthSubject(claims)
	return subject, nil
}

// Name returns the authenticator name.
func (a *OIDCAuthenticator) Name() string {
	return string(AuthModeOIDC)
}

// Priority returns the authenticator priority (lower = higher priority).
func (a *OIDCAuthenticator) Priority() int {
	return 10 // High priority for OIDC
}

// extractToken extracts the bearer token from Authorization header or cookie.
func (a *OIDCAuthenticator) extractToken(r *http.Request) string {
	// Check Authorization header first
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			return parts[1]
		}
	}

	// Fall back to cookie
	cookie, err := r.Cookie(a.tokenCookie)
	if err == nil && cookie.Value != "" {
		return cookie.Value
	}

	return ""
}

// parseAndValidateToken parses and validates the JWT token.
func (a *OIDCAuthenticator) parseAndValidateToken(ctx context.Context, tokenStr string) (*jwt.Token, error) {
	// Parse the token and validate signature
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		// Verify signing algorithm
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		// Get the key ID from header
		kidVal, ok := token.Header["kid"]
		if !ok {
			return nil, errors.New("token missing kid header")
		}
		kid, ok := kidVal.(string)
		if !ok {
			return nil, errors.New("token kid header is not a string")
		}

		// Get the public key from JWKS
		key, err := a.jwksCache.getKey(ctx, kid)
		if err != nil {
			return nil, fmt.Errorf("failed to get key for kid %s: %w", kid, err)
		}

		return key, nil
	})

	if err != nil {
		return nil, err
	}

	// Validate claims
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token claims")
	}

	// Validate issuer
	iss, ok := claims["iss"].(string)
	if !ok {
		return nil, errors.New("token missing iss claim")
	}
	if iss != a.issuer {
		return nil, fmt.Errorf("invalid issuer: got %s, want %s", iss, a.issuer)
	}

	// Validate audience
	if err := a.validateAudience(claims); err != nil {
		return nil, err
	}

	return token, nil
}

// validateAudience checks if the token audience includes our client ID.
func (a *OIDCAuthenticator) validateAudience(claims jwt.MapClaims) error {
	audClaim := claims["aud"]
	if audClaim == nil {
		return errors.New("token missing aud claim")
	}

	clientID := a.config.ClientID

	// Audience can be a string or array of strings
	switch aud := audClaim.(type) {
	case string:
		if aud != clientID {
			return fmt.Errorf("invalid audience: got %s, want %s", aud, clientID)
		}
	case []interface{}:
		found := false
		for _, a := range aud {
			if s, ok := a.(string); ok && s == clientID {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("client ID %s not in audience", clientID)
		}
	default:
		return fmt.Errorf("unexpected audience type: %T", audClaim)
	}

	return nil
}

// buildAuthSubject creates an AuthSubject from the validated claims.
func (a *OIDCAuthenticator) buildAuthSubject(claims jwt.MapClaims) *AuthSubject {
	subject := &AuthSubject{
		AuthMethod: AuthModeOIDC,
		Issuer:     a.issuer,
		RawClaims:  claims,
	}

	// Extract standard claims
	if sub, ok := claims["sub"].(string); ok {
		subject.ID = sub
	}

	// Username mapping (use configured claims or defaults)
	usernameClaims := a.config.ClaimsMapping.UsernameClaims
	if len(usernameClaims) == 0 {
		usernameClaims = []string{"preferred_username", "name", "email"}
	}
	for _, claim := range usernameClaims {
		if username, ok := claims[claim].(string); ok && username != "" {
			subject.Username = username
			break
		}
	}
	if subject.Username == "" {
		subject.Username = subject.ID // Fall back to sub
	}

	// Email
	if email, ok := claims["email"].(string); ok {
		subject.Email = email
	}
	if verified, ok := claims["email_verified"].(bool); ok {
		subject.EmailVerified = verified
	}

	// Roles mapping
	rolesClaim := a.config.ClaimsMapping.RolesClaim
	if rolesClaim == "" {
		rolesClaim = "roles"
	}
	subject.Roles = a.extractStringSlice(claims, rolesClaim)

	// Groups mapping
	groupsClaim := a.config.ClaimsMapping.GroupsClaim
	if groupsClaim == "" {
		groupsClaim = "groups"
	}
	subject.Groups = a.extractStringSlice(claims, groupsClaim)

	// Timestamps
	if iat, ok := claims["iat"].(float64); ok {
		subject.IssuedAt = int64(iat)
	}
	if exp, ok := claims["exp"].(float64); ok {
		subject.ExpiresAt = int64(exp)
	}

	return subject
}

// extractStringSlice extracts a string slice from claims.
func (a *OIDCAuthenticator) extractStringSlice(claims jwt.MapClaims, key string) []string {
	val, ok := claims[key]
	if !ok {
		return nil
	}

	switch v := val.(type) {
	case []string:
		return v
	case []interface{}:
		result := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	}

	return nil
}

// jwksCache caches JWKS keys with TTL support.
type jwksCache struct {
	uri        string
	httpClient *http.Client
	ttl        time.Duration

	mu      sync.RWMutex
	keys    map[string]*rsa.PublicKey
	fetched time.Time
}

func newJWKSCache(uri string, client *http.Client, ttl time.Duration) *jwksCache {
	return &jwksCache{
		uri:        uri,
		httpClient: client,
		ttl:        ttl,
		keys:       make(map[string]*rsa.PublicKey),
	}
}

// getKey retrieves a key by ID, refreshing the cache if needed.
func (c *jwksCache) getKey(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	c.mu.RLock()
	key, ok := c.keys[kid]
	expired := time.Since(c.fetched) > c.ttl
	c.mu.RUnlock()

	if ok && !expired {
		return key, nil
	}

	// Refresh the cache
	keys, err := c.getKeys(ctx)
	if err != nil {
		// If we have a cached key and refresh failed, use it
		if ok {
			return key, nil
		}
		return nil, err
	}

	key, ok = keys[kid]
	if !ok {
		return nil, fmt.Errorf("key not found: %s", kid)
	}

	return key, nil
}

// getKeys fetches and caches all keys from the JWKS endpoint.
func (c *jwksCache) getKeys(ctx context.Context) (map[string]*rsa.PublicKey, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if still valid (another goroutine might have refreshed)
	if time.Since(c.fetched) < c.ttl && len(c.keys) > 0 {
		return c.keys, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.uri, http.NoBody)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("JWKS fetch failed with status %d", resp.StatusCode)
	}

	var jwks struct {
		Keys []struct {
			Kty string `json:"kty"`
			Kid string `json:"kid"`
			Alg string `json:"alg"`
			Use string `json:"use"`
			N   string `json:"n"`
			E   string `json:"e"`
		} `json:"keys"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return nil, fmt.Errorf("failed to decode JWKS: %w", err)
	}

	c.keys = make(map[string]*rsa.PublicKey)

	for _, key := range jwks.Keys {
		if key.Kty != "RSA" {
			continue
		}

		// Decode modulus and exponent
		nBytes, err := base64URLDecode(key.N)
		if err != nil {
			continue
		}

		eBytes, err := base64URLDecode(key.E)
		if err != nil {
			continue
		}

		n := new(big.Int).SetBytes(nBytes)
		e := 0
		for _, b := range eBytes {
			e = e<<8 + int(b)
		}

		pubKey := &rsa.PublicKey{
			N: n,
			E: e,
		}

		c.keys[key.Kid] = pubKey
	}

	c.fetched = time.Now()
	return c.keys, nil
}

// base64URLDecode decodes a base64url encoded string.
func base64URLDecode(s string) ([]byte, error) {
	// Add padding if needed
	switch len(s) % 4 {
	case 2:
		s += "=="
	case 3:
		s += "="
	}

	return base64.URLEncoding.DecodeString(s)
}
