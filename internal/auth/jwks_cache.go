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
	"fmt"
	"math/big"
	"net/http"
	"sync"
	"time"

	"github.com/goccy/go-json"
)

// JWKSCache caches JWKS keys with TTL support.
// It is thread-safe and can be shared between OIDCAuthenticator and OIDCFlow.
type JWKSCache struct {
	uri        string
	httpClient *http.Client
	ttl        time.Duration

	mu      sync.RWMutex
	keys    map[string]*rsa.PublicKey
	fetched time.Time
}

// NewJWKSCache creates a new JWKS cache.
func NewJWKSCache(uri string, client *http.Client, ttl time.Duration) *JWKSCache {
	if client == nil {
		client = &http.Client{
			Timeout: 30 * time.Second,
		}
	}
	if ttl == 0 {
		ttl = 15 * time.Minute
	}
	return &JWKSCache{
		uri:        uri,
		httpClient: client,
		ttl:        ttl,
		keys:       make(map[string]*rsa.PublicKey),
	}
}

// GetKey retrieves a key by ID, refreshing the cache if needed.
func (c *JWKSCache) GetKey(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	c.mu.RLock()
	key, ok := c.keys[kid]
	expired := time.Since(c.fetched) > c.ttl
	c.mu.RUnlock()

	if ok && !expired {
		return key, nil
	}

	// Refresh the cache
	keys, err := c.refreshKeys(ctx)
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

// refreshKeys fetches and caches all keys from the JWKS endpoint.
func (c *JWKSCache) refreshKeys(ctx context.Context) (map[string]*rsa.PublicKey, error) {
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
	defer func() { _ = resp.Body.Close() }()

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
		nBytes, err := base64URLDecodeJWKS(key.N)
		if err != nil {
			continue
		}

		eBytes, err := base64URLDecodeJWKS(key.E)
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

// base64URLDecodeJWKS decodes a base64url encoded string.
func base64URLDecodeJWKS(s string) ([]byte, error) {
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
func (c *JWKSCache) URI() string {
	return c.uri
}

// SetURI sets the JWKS endpoint URI and clears the cache.
func (c *JWKSCache) SetURI(uri string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.uri = uri
	c.keys = make(map[string]*rsa.PublicKey)
	c.fetched = time.Time{}
}
