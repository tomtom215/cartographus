// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package auth provides authentication functionality including OIDC support.
// ADR-0015: Zero Trust Authentication & Authorization
//
// This file implements JWKS key rotation monitoring with Prometheus metrics.
// Key rotation monitoring helps detect:
//   - When the IdP rotates signing keys
//   - Potential key compromise (unexpected key changes)
//   - JWKS endpoint availability issues
//
// Metrics emitted:
//   - oidc_jwks_keys_total: Current number of keys in cache
//   - oidc_jwks_key_rotations_total: Count of key rotation events
//   - oidc_jwks_keys_added_total: Count of keys added
//   - oidc_jwks_keys_removed_total: Count of keys removed
//   - oidc_jwks_last_rotation_timestamp: Unix timestamp of last rotation
package auth

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"fmt"
	"math/big"
	"net/http"
	"sync"
	"time"

	"github.com/goccy/go-json"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/tomtom215/cartographus/internal/logging"
)

// JWKS Key Rotation Monitoring Metrics
var (
	// OIDCJWKSKeysTotal tracks the current number of keys in the JWKS cache.
	OIDCJWKSKeysTotal = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "oidc_jwks_keys_total",
			Help: "Current number of keys in the JWKS cache",
		},
		[]string{"provider"},
	)

	// OIDCJWKSKeyRotationsTotal counts the number of key rotation events detected.
	// A rotation event is when the set of key IDs changes between fetches.
	OIDCJWKSKeyRotationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "oidc_jwks_key_rotations_total",
			Help: "Total number of JWKS key rotation events detected",
		},
		[]string{"provider"},
	)

	// OIDCJWKSKeysAddedTotal counts the number of keys added during rotations.
	OIDCJWKSKeysAddedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "oidc_jwks_keys_added_total",
			Help: "Total number of JWKS keys added during rotations",
		},
		[]string{"provider"},
	)

	// OIDCJWKSKeysRemovedTotal counts the number of keys removed during rotations.
	OIDCJWKSKeysRemovedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "oidc_jwks_keys_removed_total",
			Help: "Total number of JWKS keys removed during rotations",
		},
		[]string{"provider"},
	)

	// OIDCJWKSLastRotationTimestamp records the Unix timestamp of the last rotation.
	OIDCJWKSLastRotationTimestamp = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "oidc_jwks_last_rotation_timestamp",
			Help: "Unix timestamp of the last JWKS key rotation",
		},
		[]string{"provider"},
	)

	// OIDCJWKSFetchErrorsTotal counts JWKS fetch errors.
	OIDCJWKSFetchErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "oidc_jwks_fetch_errors_total",
			Help: "Total number of JWKS fetch errors",
		},
		[]string{"provider", "error_type"},
	)
)

// JWKSCacheWithRotationMonitor extends JWKSCache with key rotation monitoring.
// It tracks key changes between fetches and emits Prometheus metrics for
// observability and security monitoring.
//
// Usage:
//
//	cache := NewJWKSCacheWithRotationMonitor("https://idp.example.com/jwks", nil, 15*time.Minute, "my-idp")
//	key, err := cache.GetKey(ctx, "key-id-123")
type JWKSCacheWithRotationMonitor struct {
	uri        string
	httpClient *http.Client
	ttl        time.Duration
	provider   string

	mu          sync.RWMutex
	keys        map[string]*rsa.PublicKey
	keyIDs      map[string]struct{} // Set of current key IDs for rotation detection
	fetched     time.Time
	initialized bool
}

// NewJWKSCacheWithRotationMonitor creates a new JWKS cache with rotation monitoring.
//
// Parameters:
//   - uri: The JWKS endpoint URL
//   - client: HTTP client (nil uses default with 30s timeout)
//   - ttl: Cache TTL (0 uses 15 minute default)
//   - provider: Provider name for metrics labels
func NewJWKSCacheWithRotationMonitor(uri string, client *http.Client, ttl time.Duration, provider string) *JWKSCacheWithRotationMonitor {
	if client == nil {
		client = &http.Client{
			Timeout: 30 * time.Second,
		}
	}
	if ttl == 0 {
		ttl = 15 * time.Minute
	}
	if provider == "" {
		provider = "default"
	}

	return &JWKSCacheWithRotationMonitor{
		uri:        uri,
		httpClient: client,
		ttl:        ttl,
		provider:   provider,
		keys:       make(map[string]*rsa.PublicKey),
		keyIDs:     make(map[string]struct{}),
	}
}

// GetKey retrieves a key by ID, refreshing the cache if needed.
// Records metrics for cache hits/misses and key rotation events.
func (c *JWKSCacheWithRotationMonitor) GetKey(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	start := time.Now()

	c.mu.RLock()
	key, ok := c.keys[kid]
	expired := time.Since(c.fetched) > c.ttl
	c.mu.RUnlock()

	if ok && !expired {
		OIDCJWKSCacheHits.Inc()
		return key, nil
	}

	// Cache miss - need to refresh
	OIDCJWKSCacheMisses.Inc()

	keys, err := c.refreshKeysWithMonitoring(ctx)
	if err != nil {
		// If we have a cached key and refresh failed, use it (graceful degradation)
		if ok {
			logging.Warn().
				Err(err).
				Str("kid", kid).
				Str("provider", c.provider).
				Msg("JWKS refresh failed, using cached key")
			return key, nil
		}
		return nil, err
	}

	// Record fetch duration
	RecordOIDCJWKSFetch(c.provider, time.Since(start), false)

	key, ok = keys[kid]
	if !ok {
		return nil, fmt.Errorf("key not found: %s", kid)
	}

	return key, nil
}

// refreshKeysWithMonitoring fetches keys and monitors for rotation events.
func (c *JWKSCacheWithRotationMonitor) refreshKeysWithMonitoring(ctx context.Context) (map[string]*rsa.PublicKey, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check if still needs refresh (another goroutine might have done it)
	if time.Since(c.fetched) < c.ttl && len(c.keys) > 0 {
		return c.keys, nil
	}

	// Fetch JWKS from endpoint
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.uri, http.NoBody)
	if err != nil {
		OIDCJWKSFetchErrorsTotal.WithLabelValues(c.provider, "request_creation").Inc()
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		OIDCJWKSFetchErrorsTotal.WithLabelValues(c.provider, "network").Inc()
		return nil, fmt.Errorf("fetch JWKS: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		OIDCJWKSFetchErrorsTotal.WithLabelValues(c.provider, "http_status").Inc()
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
		OIDCJWKSFetchErrorsTotal.WithLabelValues(c.provider, "decode").Inc()
		return nil, fmt.Errorf("decode JWKS: %w", err)
	}

	// Parse keys and build new key sets
	newKeys := make(map[string]*rsa.PublicKey)
	newKeyIDs := make(map[string]struct{})

	for _, key := range jwks.Keys {
		if key.Kty != "RSA" {
			continue
		}
		if key.Kid == "" {
			continue
		}

		pubKey, err := parseRSAPublicKey(key.N, key.E)
		if err != nil {
			logging.Warn().
				Err(err).
				Str("kid", key.Kid).
				Str("provider", c.provider).
				Msg("Failed to parse RSA key, skipping")
			continue
		}

		newKeys[key.Kid] = pubKey
		newKeyIDs[key.Kid] = struct{}{}
	}

	// Detect key rotation
	c.detectRotation(newKeyIDs)

	// Update cache
	c.keys = newKeys
	c.keyIDs = newKeyIDs
	c.fetched = time.Now()
	c.initialized = true

	// Update key count metric
	OIDCJWKSKeysTotal.WithLabelValues(c.provider).Set(float64(len(newKeys)))

	return c.keys, nil
}

// detectRotation compares old and new key sets to detect rotation events.
func (c *JWKSCacheWithRotationMonitor) detectRotation(newKeyIDs map[string]struct{}) {
	// Skip on first fetch (initialization)
	if !c.initialized {
		return
	}

	// Find added keys
	added := 0
	for kid := range newKeyIDs {
		if _, exists := c.keyIDs[kid]; !exists {
			added++
			logging.Info().
				Str("kid", kid).
				Str("provider", c.provider).
				Msg("JWKS key added")
		}
	}

	// Find removed keys
	removed := 0
	for kid := range c.keyIDs {
		if _, exists := newKeyIDs[kid]; !exists {
			removed++
			logging.Info().
				Str("kid", kid).
				Str("provider", c.provider).
				Msg("JWKS key removed")
		}
	}

	// Record rotation if any changes detected
	if added > 0 || removed > 0 {
		OIDCJWKSKeyRotationsTotal.WithLabelValues(c.provider).Inc()
		OIDCJWKSKeysAddedTotal.WithLabelValues(c.provider).Add(float64(added))
		OIDCJWKSKeysRemovedTotal.WithLabelValues(c.provider).Add(float64(removed))
		OIDCJWKSLastRotationTimestamp.WithLabelValues(c.provider).Set(float64(time.Now().Unix()))

		logging.Info().
			Int("keys_added", added).
			Int("keys_removed", removed).
			Int("total_keys", len(newKeyIDs)).
			Str("provider", c.provider).
			Msg("JWKS key rotation detected")
	}
}

// parseRSAPublicKey parses RSA modulus and exponent into a public key.
func parseRSAPublicKey(nBase64, eBase64 string) (*rsa.PublicKey, error) {
	nBytes, err := base64URLDecodeJWKSRotation(nBase64)
	if err != nil {
		return nil, fmt.Errorf("decode modulus: %w", err)
	}

	eBytes, err := base64URLDecodeJWKSRotation(eBase64)
	if err != nil {
		return nil, fmt.Errorf("decode exponent: %w", err)
	}

	n := new(big.Int).SetBytes(nBytes)
	e := 0
	for _, b := range eBytes {
		e = e<<8 + int(b)
	}

	return &rsa.PublicKey{N: n, E: e}, nil
}

// base64URLDecodeJWKSRotation decodes a base64url encoded string.
func base64URLDecodeJWKSRotation(s string) ([]byte, error) {
	// Add padding if needed
	switch len(s) % 4 {
	case 2:
		s += "=="
	case 3:
		s += "="
	}
	return base64.URLEncoding.DecodeString(s)
}

// URI returns the JWKS endpoint URI.
func (c *JWKSCacheWithRotationMonitor) URI() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.uri
}

// SetURI sets the JWKS endpoint URI and clears the cache.
func (c *JWKSCacheWithRotationMonitor) SetURI(uri string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.uri = uri
	c.keys = make(map[string]*rsa.PublicKey)
	c.keyIDs = make(map[string]struct{})
	c.fetched = time.Time{}
	c.initialized = false
}

// KeyCount returns the current number of cached keys.
func (c *JWKSCacheWithRotationMonitor) KeyCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.keys)
}

// LastFetched returns the timestamp of the last successful fetch.
func (c *JWKSCacheWithRotationMonitor) LastFetched() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.fetched
}

// IsExpired returns true if the cache has expired.
func (c *JWKSCacheWithRotationMonitor) IsExpired() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return time.Since(c.fetched) > c.ttl
}

// Provider returns the provider name for this cache.
func (c *JWKSCacheWithRotationMonitor) Provider() string {
	return c.provider
}

// KeyIDs returns a copy of the current key IDs in the cache.
// Useful for debugging and monitoring.
func (c *JWKSCacheWithRotationMonitor) KeyIDs() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	ids := make([]string, 0, len(c.keyIDs))
	for kid := range c.keyIDs {
		ids = append(ids, kid)
	}
	return ids
}
