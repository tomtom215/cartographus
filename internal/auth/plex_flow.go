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
	"io"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/goccy/go-json"
)

// Plex Flow errors
var (
	// ErrPINNotFound indicates the PIN was not found or expired.
	ErrPINNotFound = errors.New("PIN not found or expired")

	// ErrPINNotAuthorized indicates the PIN has not been authorized yet.
	ErrPINNotAuthorized = errors.New("PIN not yet authorized")

	// ErrPlexAPIFailed indicates a Plex API call failed.
	ErrPlexAPIFailed = errors.New("Plex API request failed")
)

// Plex API endpoints
const (
	PlexPINEndpoint      = "https://plex.tv/api/v2/pins.json"
	PlexUserInfoEndpoint = "https://plex.tv/api/v2/user"
	PlexAuthBaseURL      = "https://app.plex.tv/auth#!"
)

// PlexFlowConfig holds configuration for the Plex authentication flow.
type PlexFlowConfig struct {
	// ClientID is the Plex application client identifier.
	// This should be a unique identifier for your application.
	ClientID string

	// Product is the product name shown to users.
	Product string

	// Version is the application version.
	Version string

	// Platform is the platform identifier (optional).
	Platform string

	// Device is the device name (optional).
	Device string

	// RedirectURI is the callback URL after Plex auth completes.
	RedirectURI string

	// ForwardURL is where Plex redirects after auth (for web flow).
	ForwardURL string

	// PINTimeout is how long PINs are valid.
	PINTimeout time.Duration

	// PollInterval is how often to poll for PIN authorization.
	PollInterval time.Duration

	// SessionDuration is how long sessions are valid.
	SessionDuration time.Duration

	// DefaultRoles are assigned to all Plex-authenticated users.
	DefaultRoles []string

	// PlexPassRole is assigned to Plex Pass subscribers.
	PlexPassRole string

	// HTTPClient for making requests (optional).
	HTTPClient *http.Client
}

// Validate checks the configuration for errors.
func (c *PlexFlowConfig) Validate() error {
	if c.ClientID == "" {
		return errors.New("plex: client ID is required")
	}
	if c.Product == "" {
		return errors.New("plex: product name is required")
	}
	return nil
}

// PlexPIN represents a Plex authentication PIN.
type PlexPIN struct {
	// ID is the PIN's unique identifier.
	ID int

	// Code is the user-visible PIN code.
	Code string

	// AuthURL is the URL where users authorize the PIN.
	AuthURL string

	// ExpiresAt is when the PIN expires.
	ExpiresAt time.Time
}

// PlexPINData holds internal PIN state data.
type PlexPINData struct {
	ID                int
	Code              string
	PostLoginRedirect string
	CreatedAt         time.Time
	ExpiresAt         time.Time
}

// IsExpired checks if the PIN has expired.
func (p *PlexPINData) IsExpired() bool {
	return time.Now().After(p.ExpiresAt)
}

// PlexPINStore defines the interface for storing Plex PIN data.
type PlexPINStore interface {
	// Store saves PIN data with the given ID.
	Store(ctx context.Context, pinID int, data *PlexPINData) error

	// Get retrieves PIN data by ID.
	Get(ctx context.Context, pinID int) (*PlexPINData, error)

	// Delete removes PIN data by ID.
	Delete(ctx context.Context, pinID int) error

	// CleanupExpired removes all expired PINs.
	CleanupExpired(ctx context.Context) (int, error)
}

// MemoryPlexPINStore is an in-memory implementation of PlexPINStore.
type MemoryPlexPINStore struct {
	mu   sync.RWMutex
	pins map[int]*PlexPINData
}

// NewMemoryPlexPINStore creates a new in-memory PIN store.
func NewMemoryPlexPINStore() *MemoryPlexPINStore {
	return &MemoryPlexPINStore{
		pins: make(map[int]*PlexPINData),
	}
}

// Store saves PIN data with the given ID.
func (s *MemoryPlexPINStore) Store(ctx context.Context, pinID int, data *PlexPINData) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.pins[pinID] = &PlexPINData{
		ID:                data.ID,
		Code:              data.Code,
		PostLoginRedirect: data.PostLoginRedirect,
		CreatedAt:         data.CreatedAt,
		ExpiresAt:         data.ExpiresAt,
	}
	return nil
}

// Get retrieves PIN data by ID.
func (s *MemoryPlexPINStore) Get(ctx context.Context, pinID int) (*PlexPINData, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, ok := s.pins[pinID]
	if !ok {
		return nil, ErrPINNotFound
	}

	if data.IsExpired() {
		return nil, ErrPINNotFound
	}

	return &PlexPINData{
		ID:                data.ID,
		Code:              data.Code,
		PostLoginRedirect: data.PostLoginRedirect,
		CreatedAt:         data.CreatedAt,
		ExpiresAt:         data.ExpiresAt,
	}, nil
}

// Delete removes PIN data by ID.
func (s *MemoryPlexPINStore) Delete(ctx context.Context, pinID int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.pins, pinID)
	return nil
}

// CleanupExpired removes all expired PINs.
func (s *MemoryPlexPINStore) CleanupExpired(ctx context.Context) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	count := 0
	for id, data := range s.pins {
		if data.IsExpired() {
			delete(s.pins, id)
			count++
		}
	}
	return count, nil
}

// PlexFlow manages the Plex PIN-based authentication flow.
type PlexFlow struct {
	config *PlexFlowConfig
	store  PlexPINStore
	client *http.Client

	// Endpoints (can be overridden for testing)
	pinEndpoint      string
	pinCheckEndpoint string
	userInfoEndpoint string
}

// NewPlexFlow creates a new Plex flow manager.
func NewPlexFlow(config *PlexFlowConfig, store PlexPINStore) *PlexFlow {
	client := config.HTTPClient
	if client == nil {
		client = &http.Client{
			Timeout: 30 * time.Second,
		}
	}

	// Set defaults
	if config.PINTimeout == 0 {
		config.PINTimeout = 5 * time.Minute
	}
	if config.PollInterval == 0 {
		config.PollInterval = 2 * time.Second
	}
	if config.Version == "" {
		config.Version = "1.0.0"
	}

	return &PlexFlow{
		config:           config,
		store:            store,
		client:           client,
		pinEndpoint:      PlexPINEndpoint,
		pinCheckEndpoint: "https://plex.tv/api/v2/pins",
		userInfoEndpoint: PlexUserInfoEndpoint,
	}
}

// SetPINEndpoint sets the PIN creation endpoint (for testing).
func (f *PlexFlow) SetPINEndpoint(url string) {
	f.pinEndpoint = url
}

// SetPINCheckEndpoint sets the PIN check endpoint (for testing).
func (f *PlexFlow) SetPINCheckEndpoint(url string) {
	f.pinCheckEndpoint = url
}

// SetUserInfoEndpoint sets the user info endpoint (for testing).
func (f *PlexFlow) SetUserInfoEndpoint(url string) {
	f.userInfoEndpoint = url
}

// RequestPIN requests a new authentication PIN from Plex.
func (f *PlexFlow) RequestPIN(ctx context.Context) (*PlexPIN, error) {
	// Build request
	data := url.Values{}
	data.Set("strong", "true")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, f.pinEndpoint, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("create PIN request: %w", err)
	}

	// Set required Plex headers
	f.setPlexHeaders(req)

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("PIN request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("%w: status %d", ErrPlexAPIFailed, resp.StatusCode)
		}
		return nil, fmt.Errorf("%w: status %d: %s", ErrPlexAPIFailed, resp.StatusCode, string(body))
	}

	var pinResp struct {
		ID        int    `json:"id"`
		Code      string `json:"code"`
		ExpiresAt string `json:"expires_at"`
		AuthToken string `json:"auth_token"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&pinResp); err != nil {
		return nil, fmt.Errorf("decode PIN response: %w", err)
	}

	// Parse expiration
	expiresAt, err := time.Parse(time.RFC3339, pinResp.ExpiresAt)
	if err != nil {
		// Default to configured timeout
		expiresAt = time.Now().Add(f.config.PINTimeout)
	}

	pin := &PlexPIN{
		ID:        pinResp.ID,
		Code:      pinResp.Code,
		ExpiresAt: expiresAt,
		AuthURL:   f.BuildAuthorizationURL(&PlexPIN{ID: pinResp.ID, Code: pinResp.Code}),
	}

	// Store PIN data
	pinData := &PlexPINData{
		ID:        pin.ID,
		Code:      pin.Code,
		CreatedAt: time.Now(),
		ExpiresAt: expiresAt,
	}
	if err := f.store.Store(ctx, pin.ID, pinData); err != nil {
		return nil, fmt.Errorf("store PIN: %w", err)
	}

	return pin, nil
}

// BuildAuthorizationURL builds the Plex authorization URL for a PIN.
func (f *PlexFlow) BuildAuthorizationURL(pin *PlexPIN) string {
	params := url.Values{}
	params.Set("clientID", f.config.ClientID)
	params.Set("code", pin.Code)
	params.Set("context[device][product]", f.config.Product)
	params.Set("context[device][version]", f.config.Version)

	if f.config.ForwardURL != "" {
		params.Set("forwardUrl", f.config.ForwardURL)
	}

	return PlexAuthBaseURL + "?" + params.Encode()
}

// PlexPINCheckResult holds the result of a PIN check.
type PlexPINCheckResult struct {
	// Authorized indicates if the PIN has been authorized.
	Authorized bool

	// AuthToken is the Plex auth token (if authorized).
	AuthToken string

	// ExpiresAt is when the PIN expires.
	ExpiresAt time.Time
}

// CheckPIN checks if a PIN has been authorized.
func (f *PlexFlow) CheckPIN(ctx context.Context, pinID int) (*PlexPINCheckResult, error) {
	checkURL := f.pinCheckEndpoint + "/" + strconv.Itoa(pinID) + ".json"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, checkURL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("create check request: %w", err)
	}

	f.setPlexHeaders(req)

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("check request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrPINNotFound
	}

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("%w: status %d", ErrPlexAPIFailed, resp.StatusCode)
		}
		return nil, fmt.Errorf("%w: status %d: %s", ErrPlexAPIFailed, resp.StatusCode, string(body))
	}

	var pinResp struct {
		ID        int     `json:"id"`
		Code      string  `json:"code"`
		ExpiresAt string  `json:"expires_at"`
		AuthToken *string `json:"auth_token"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&pinResp); err != nil {
		return nil, fmt.Errorf("decode check response: %w", err)
	}

	expiresAt, err := time.Parse(time.RFC3339, pinResp.ExpiresAt)
	if err != nil {
		expiresAt = time.Now().Add(5 * time.Minute) // Default expiration if parse fails
	}

	result := &PlexPINCheckResult{
		ExpiresAt: expiresAt,
	}

	if pinResp.AuthToken != nil && *pinResp.AuthToken != "" {
		result.Authorized = true
		result.AuthToken = *pinResp.AuthToken
	}

	return result, nil
}

// GetUserInfo fetches user information using the auth token.
func (f *PlexFlow) GetUserInfo(ctx context.Context, authToken string) (*AuthSubject, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, f.userInfoEndpoint, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("create user info request: %w", err)
	}

	f.setPlexHeaders(req)
	req.Header.Set("X-Plex-Token", authToken)

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("user info request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, ErrInvalidCredentials
	}

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("%w: status %d", ErrPlexAPIFailed, resp.StatusCode)
		}
		return nil, fmt.Errorf("%w: status %d: %s", ErrPlexAPIFailed, resp.StatusCode, string(body))
	}

	var userResp struct {
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

	if err := json.NewDecoder(resp.Body).Decode(&userResp); err != nil {
		return nil, fmt.Errorf("decode user info response: %w", err)
	}

	user := userResp.User

	subject := &AuthSubject{
		ID:         strconv.Itoa(user.ID),
		Username:   user.Username,
		Email:      user.Email,
		AuthMethod: AuthModePlex,
		Issuer:     "plex.tv",
		Provider:   "plex",
		RawClaims: map[string]interface{}{
			"plex_id":   user.ID,
			"plex_uuid": user.UUID,
			"thumb":     user.Thumb,
		},
		Metadata: map[string]string{
			"plex_uuid": user.UUID,
			"thumb":     user.Thumb,
		},
	}

	// Assign default roles
	if len(f.config.DefaultRoles) > 0 {
		subject.Roles = make([]string, len(f.config.DefaultRoles))
		copy(subject.Roles, f.config.DefaultRoles)
	}

	// Add Plex Pass role if applicable
	if user.Subscription.Active && f.config.PlexPassRole != "" {
		subject.Roles = append(subject.Roles, f.config.PlexPassRole)
	}

	return subject, nil
}

// PollForAuthorization polls for PIN authorization until authorized or timeout.
func (f *PlexFlow) PollForAuthorization(ctx context.Context, pinID int) (*PlexPINCheckResult, error) {
	ticker := time.NewTicker(f.config.PollInterval)
	defer ticker.Stop()

	timeout := time.After(f.config.PINTimeout)

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timeout:
			return nil, ErrPINNotFound
		case <-ticker.C:
			result, err := f.CheckPIN(ctx, pinID)
			if err != nil {
				if errors.Is(err, ErrPINNotFound) {
					return nil, err
				}
				// Continue polling on transient errors
				continue
			}

			if result.Authorized {
				return result, nil
			}
		}
	}
}

// setPlexHeaders sets the required Plex API headers.
func (f *PlexFlow) setPlexHeaders(req *http.Request) {
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Plex-Client-Identifier", f.config.ClientID)
	req.Header.Set("X-Plex-Product", f.config.Product)
	req.Header.Set("X-Plex-Version", f.config.Version)

	if f.config.Platform != "" {
		req.Header.Set("X-Plex-Platform", f.config.Platform)
	}
	if f.config.Device != "" {
		req.Header.Set("X-Plex-Device", f.config.Device)
	}
}
