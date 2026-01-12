// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/goccy/go-json"
)

// PlexOAuthClient implements RFC 7636 PKCE (Proof Key for Code Exchange)
// for secure OAuth 2.0 authorization with Plex Media Server.
//
// OAuth Flow:
//  1. Generate code_verifier and code_challenge (PKCE)
//  2. Redirect user to Plex authorization URL with code_challenge
//  3. User authorizes and Plex redirects back with authorization code
//  4. Exchange authorization code + code_verifier for access token
//  5. Use access token for Plex API calls
//
// Security: PKCE prevents authorization code interception attacks by requiring
// the original code_verifier to exchange the authorization code for tokens.
type PlexOAuthClient struct {
	ClientID     string       // Plex application client ID
	ClientSecret string       // Plex application client secret (optional for public clients)
	RedirectURI  string       // OAuth callback URL (must match Plex app config)
	httpClient   *http.Client // HTTP client for API calls

	// Plex OAuth endpoints
	authURL  string // Default: https://app.plex.tv/auth#?
	tokenURL string // Default: https://plex.tv/api/v2/oauth/token
}

// PlexOAuthToken represents the OAuth 2.0 access token response from Plex.
type PlexOAuthToken struct {
	AccessToken  string `json:"access_token"`  // Bearer token for API calls
	TokenType    string `json:"token_type"`    // Always "Bearer"
	ExpiresIn    int    `json:"expires_in"`    // Token lifetime in seconds (typically 90 days)
	RefreshToken string `json:"refresh_token"` // Token for obtaining new access tokens
	Scope        string `json:"scope"`         // Granted scopes (e.g., "all")
	ExpiresAt    int64  `json:"expires_at"`    // Unix timestamp when token expires
}

// PKCEChallenge contains the code verifier and challenge for PKCE flow.
type PKCEChallenge struct {
	CodeVerifier  string // Random 43-128 character string (RFC 7636)
	CodeChallenge string // Base64URL(SHA256(code_verifier))
}

// NewPlexOAuthClient creates a new Plex OAuth client with default endpoints.
//
// Parameters:
//   - clientID: Your Plex application's client identifier
//   - clientSecret: Your Plex application's client secret (can be empty for public clients)
//   - redirectURI: The callback URL where Plex redirects after authorization
//
// Returns a configured PlexOAuthClient ready to initiate OAuth flows.
func NewPlexOAuthClient(clientID, clientSecret, redirectURI string) *PlexOAuthClient {
	return &PlexOAuthClient{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURI:  redirectURI,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		authURL:  "https://app.plex.tv/auth#?",
		tokenURL: "https://plex.tv/api/v2/oauth/token",
	}
}

// GeneratePKCE generates a PKCE code verifier and challenge pair.
//
// Implementation:
//  1. Generate 32 random bytes (256 bits of entropy)
//  2. Base64URL encode as code_verifier (43 characters)
//  3. SHA256 hash the code_verifier
//  4. Base64URL encode the hash as code_challenge
//
// Returns:
//   - PKCEChallenge with verifier and challenge
//   - error if random number generation fails
//
// Security: Uses crypto/rand for cryptographically secure random generation.
// The code_verifier is 43 characters (minimum per RFC 7636 is 43, maximum 128).
func (c *PlexOAuthClient) GeneratePKCE() (*PKCEChallenge, error) {
	// Generate 32 random bytes (256 bits)
	verifierBytes := make([]byte, 32)
	if _, err := rand.Read(verifierBytes); err != nil {
		return nil, fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// Base64URL encode the verifier (43 characters without padding)
	codeVerifier := base64.RawURLEncoding.EncodeToString(verifierBytes)

	// SHA256 hash the verifier
	hash := sha256.Sum256([]byte(codeVerifier))

	// Base64URL encode the hash as the challenge
	codeChallenge := base64.RawURLEncoding.EncodeToString(hash[:])

	return &PKCEChallenge{
		CodeVerifier:  codeVerifier,
		CodeChallenge: codeChallenge,
	}, nil
}

// BuildAuthorizationURL constructs the Plex authorization URL with PKCE parameters.
//
// Parameters:
//   - codeChallenge: The PKCE code_challenge from GeneratePKCE()
//   - state: A random string to prevent CSRF attacks (should be stored in session)
//
// Returns the full authorization URL to redirect the user to.
//
// Example URL:
//
//	https://app.plex.tv/auth#?clientID=abc123&code=xyz789&
//	  code_challenge=E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM&
//	  code_challenge_method=S256&
//	  state=random-state-string
//
// The user should be redirected to this URL to authorize the application.
func (c *PlexOAuthClient) BuildAuthorizationURL(codeChallenge, state string) string {
	params := url.Values{}
	params.Set("clientID", c.ClientID)
	params.Set("code_challenge", codeChallenge)
	params.Set("code_challenge_method", "S256") // SHA256 method
	params.Set("state", state)
	params.Set("forwardUrl", c.RedirectURI)
	params.Set("context[device][product]", "Cartographus")
	params.Set("context[device][version]", "1.42")
	params.Set("context[device][platform]", "Web")

	return c.authURL + params.Encode()
}

// ExchangeCodeForToken exchanges an authorization code for access and refresh tokens.
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - code: The authorization code from Plex callback
//   - codeVerifier: The original PKCE code_verifier from GeneratePKCE()
//
// Returns:
//   - PlexOAuthToken with access_token, refresh_token, and expiration
//   - error if the exchange fails (invalid code, expired code, or network error)
//
// Implementation:
//  1. POST to https://plex.tv/api/v2/oauth/token
//  2. Include code, code_verifier, client_id, redirect_uri
//  3. Parse JSON response with access_token and refresh_token
//
// Security: The code_verifier proves that the client requesting the token
// is the same client that initiated the authorization request.
func (c *PlexOAuthClient) ExchangeCodeForToken(ctx context.Context, code, codeVerifier string) (*PlexOAuthToken, error) {
	// Build token request parameters
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("code_verifier", codeVerifier)
	data.Set("client_id", c.ClientID)
	data.Set("redirect_uri", c.RedirectURI)

	// Add client_secret if provided (for confidential clients)
	if c.ClientSecret != "" {
		data.Set("client_secret", c.ClientSecret)
	}

	// POST to token endpoint
	req, err := http.NewRequestWithContext(ctx, "POST", c.tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create token request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code for token: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read token response: %w", err)
	}

	// Check for HTTP errors
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse JSON response
	var token PlexOAuthToken
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	// Calculate expiration timestamp
	token.ExpiresAt = time.Now().Unix() + int64(token.ExpiresIn)

	return &token, nil
}

// RefreshToken uses a refresh token to obtain a new access token.
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - refreshToken: The refresh_token from a previous PlexOAuthToken
//
// Returns:
//   - PlexOAuthToken with new access_token (and possibly new refresh_token)
//   - error if refresh fails (invalid/expired refresh token)
//
// Implementation:
//  1. POST to https://plex.tv/api/v2/oauth/token
//  2. Include grant_type=refresh_token and refresh_token
//  3. Parse JSON response with new tokens
//
// Best Practice: Refresh tokens before access token expires (typically 90 days).
// Store the new refresh_token as it may rotate.
func (c *PlexOAuthClient) RefreshToken(ctx context.Context, refreshToken string) (*PlexOAuthToken, error) {
	// Build refresh request parameters
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", refreshToken)
	data.Set("client_id", c.ClientID)

	// Add client_secret if provided
	if c.ClientSecret != "" {
		data.Set("client_secret", c.ClientSecret)
	}

	// POST to token endpoint
	req, err := http.NewRequestWithContext(ctx, "POST", c.tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create refresh request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read refresh response: %w", err)
	}

	// Check for HTTP errors
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token refresh failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse JSON response
	var token PlexOAuthToken
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, fmt.Errorf("failed to parse refresh response: %w", err)
	}

	// Calculate expiration timestamp
	token.ExpiresAt = time.Now().Unix() + int64(token.ExpiresIn)

	return &token, nil
}

// ValidateToken checks if an access token is still valid.
//
// Parameters:
//   - token: The PlexOAuthToken to validate
//
// Returns:
//   - true if token is valid and not expired
//   - false if token is expired or invalid
//
// Implementation: Checks if current time < expires_at timestamp.
// Add 5-minute buffer to refresh before actual expiration.
func (c *PlexOAuthClient) ValidateToken(token *PlexOAuthToken) bool {
	if token == nil || token.AccessToken == "" {
		return false
	}

	// Check expiration with 5-minute buffer
	expirationBuffer := int64(5 * 60) // 5 minutes
	return time.Now().Unix() < (token.ExpiresAt - expirationBuffer)
}

// SetTokenURL overrides the token endpoint URL.
// This is primarily used for testing with mock servers.
func (c *PlexOAuthClient) SetTokenURL(url string) {
	c.tokenURL = url
}
