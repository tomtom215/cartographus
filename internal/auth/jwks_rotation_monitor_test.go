// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package auth provides authentication functionality including OIDC support.
// ADR-0015: Zero Trust Authentication & Authorization
package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// testJWKS represents a JWKS response for testing.
type testJWKS struct {
	Keys []testJWK `json:"keys"`
}

// testJWK represents a JWK for testing.
type testJWK struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Alg string `json:"alg"`
	Use string `json:"use"`
	N   string `json:"n"`
	E   string `json:"e"`
}

// generateTestRSAKey generates a test RSA key pair.
func generateTestRSAKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}
	return key
}

// createTestJWKS creates a JWKS response with the given keys.
func createTestJWKS(keys map[string]*rsa.PublicKey) testJWKS {
	jwks := testJWKS{Keys: make([]testJWK, 0, len(keys))}
	for kid, key := range keys {
		jwks.Keys = append(jwks.Keys, testJWK{
			Kty: "RSA",
			Kid: kid,
			Alg: "RS256",
			Use: "sig",
			N:   base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
			E:   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes()),
		})
	}
	return jwks
}

func TestNewJWKSCacheWithRotationMonitor(t *testing.T) {
	t.Run("default_values", func(t *testing.T) {
		cache := NewJWKSCacheWithRotationMonitor("https://example.com/jwks", nil, 0, "")

		if cache.ttl != 15*time.Minute {
			t.Errorf("Expected default TTL of 15m, got %v", cache.ttl)
		}
		if cache.provider != "default" {
			t.Errorf("Expected default provider 'default', got %q", cache.provider)
		}
		if cache.httpClient == nil {
			t.Error("Expected default HTTP client to be set")
		}
	})

	t.Run("custom_values", func(t *testing.T) {
		client := &http.Client{Timeout: 10 * time.Second}
		cache := NewJWKSCacheWithRotationMonitor("https://example.com/jwks", client, 5*time.Minute, "my-idp")

		if cache.ttl != 5*time.Minute {
			t.Errorf("Expected TTL of 5m, got %v", cache.ttl)
		}
		if cache.provider != "my-idp" {
			t.Errorf("Expected provider 'my-idp', got %q", cache.provider)
		}
		if cache.httpClient != client {
			t.Error("Expected custom HTTP client to be used")
		}
	})
}

func TestJWKSCacheWithRotationMonitor_GetKey(t *testing.T) {
	key1 := generateTestRSAKey(t)
	key2 := generateTestRSAKey(t)

	keys := map[string]*rsa.PublicKey{
		"key-1": &key1.PublicKey,
		"key-2": &key2.PublicKey,
	}
	jwks := createTestJWKS(keys)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(jwks)
	}))
	defer server.Close()

	cache := NewJWKSCacheWithRotationMonitor(server.URL, nil, 15*time.Minute, "test")

	t.Run("get_existing_key", func(t *testing.T) {
		ctx := context.Background()
		pubKey, err := cache.GetKey(ctx, "key-1")
		if err != nil {
			t.Fatalf("GetKey failed: %v", err)
		}
		if pubKey == nil {
			t.Fatal("Expected public key, got nil")
		}
		if pubKey.N.Cmp(key1.N) != 0 {
			t.Error("Public key modulus doesn't match")
		}
	})

	t.Run("get_second_key", func(t *testing.T) {
		ctx := context.Background()
		pubKey, err := cache.GetKey(ctx, "key-2")
		if err != nil {
			t.Fatalf("GetKey failed: %v", err)
		}
		if pubKey == nil {
			t.Fatal("Expected public key, got nil")
		}
	})

	t.Run("get_nonexistent_key", func(t *testing.T) {
		ctx := context.Background()
		_, err := cache.GetKey(ctx, "nonexistent")
		if err == nil {
			t.Error("Expected error for nonexistent key")
		}
	})

	t.Run("cache_hit", func(t *testing.T) {
		ctx := context.Background()
		// First call populates cache
		_, _ = cache.GetKey(ctx, "key-1")
		// Second call should hit cache
		pubKey, err := cache.GetKey(ctx, "key-1")
		if err != nil {
			t.Fatalf("GetKey failed: %v", err)
		}
		if pubKey == nil {
			t.Fatal("Expected cached key")
		}
	})
}

func TestJWKSCacheWithRotationMonitor_KeyRotation(t *testing.T) {
	key1 := generateTestRSAKey(t)
	key2 := generateTestRSAKey(t)
	key3 := generateTestRSAKey(t)

	var mu sync.Mutex
	currentKeys := map[string]*rsa.PublicKey{
		"key-1": &key1.PublicKey,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		jwks := createTestJWKS(currentKeys)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(jwks)
	}))
	defer server.Close()

	// Use very short TTL for testing
	cache := NewJWKSCacheWithRotationMonitor(server.URL, nil, 50*time.Millisecond, "test")

	t.Run("initial_fetch", func(t *testing.T) {
		ctx := context.Background()
		_, err := cache.GetKey(ctx, "key-1")
		if err != nil {
			t.Fatalf("Initial fetch failed: %v", err)
		}

		if cache.KeyCount() != 1 {
			t.Errorf("Expected 1 key, got %d", cache.KeyCount())
		}
	})

	t.Run("detect_key_addition", func(t *testing.T) {
		// Add a new key
		mu.Lock()
		currentKeys["key-2"] = &key2.PublicKey
		mu.Unlock()

		// Wait for cache to expire
		time.Sleep(100 * time.Millisecond)

		ctx := context.Background()
		_, err := cache.GetKey(ctx, "key-2")
		if err != nil {
			t.Fatalf("GetKey after rotation failed: %v", err)
		}

		if cache.KeyCount() != 2 {
			t.Errorf("Expected 2 keys, got %d", cache.KeyCount())
		}
	})

	t.Run("detect_key_removal", func(t *testing.T) {
		// Remove key-1
		mu.Lock()
		delete(currentKeys, "key-1")
		mu.Unlock()

		// Wait for cache to expire
		time.Sleep(100 * time.Millisecond)

		ctx := context.Background()
		_, _ = cache.GetKey(ctx, "key-2") // Force refresh

		if cache.KeyCount() != 1 {
			t.Errorf("Expected 1 key after removal, got %d", cache.KeyCount())
		}

		// key-1 should no longer be available
		_, err := cache.GetKey(ctx, "key-1")
		if err == nil {
			t.Error("Expected error for removed key")
		}
	})

	t.Run("detect_full_rotation", func(t *testing.T) {
		// Replace all keys
		mu.Lock()
		currentKeys = map[string]*rsa.PublicKey{
			"key-3": &key3.PublicKey,
		}
		mu.Unlock()

		// Wait for cache to expire
		time.Sleep(100 * time.Millisecond)

		ctx := context.Background()
		_, err := cache.GetKey(ctx, "key-3")
		if err != nil {
			t.Fatalf("GetKey after full rotation failed: %v", err)
		}

		if cache.KeyCount() != 1 {
			t.Errorf("Expected 1 key after full rotation, got %d", cache.KeyCount())
		}
	})
}

func TestJWKSCacheWithRotationMonitor_GracefulDegradation(t *testing.T) {
	key1 := generateTestRSAKey(t)
	keys := map[string]*rsa.PublicKey{
		"key-1": &key1.PublicKey,
	}
	jwks := createTestJWKS(keys)

	requestCount := 0
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requestCount++
		count := requestCount
		mu.Unlock()

		// First request succeeds, subsequent requests fail
		if count == 1 {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(jwks)
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
	}))
	defer server.Close()

	cache := NewJWKSCacheWithRotationMonitor(server.URL, nil, 50*time.Millisecond, "test")

	// First fetch succeeds
	ctx := context.Background()
	pubKey, err := cache.GetKey(ctx, "key-1")
	if err != nil {
		t.Fatalf("Initial fetch failed: %v", err)
	}
	if pubKey == nil {
		t.Fatal("Expected public key")
	}

	// Wait for cache to expire
	time.Sleep(100 * time.Millisecond)

	// Second fetch fails but should return cached key
	pubKey2, err := cache.GetKey(ctx, "key-1")
	if err != nil {
		t.Fatalf("Should use cached key on fetch failure: %v", err)
	}
	if pubKey2 == nil {
		t.Fatal("Expected cached key on fetch failure")
	}
}

func TestJWKSCacheWithRotationMonitor_ConcurrentAccess(t *testing.T) {
	key1 := generateTestRSAKey(t)
	keys := map[string]*rsa.PublicKey{
		"key-1": &key1.PublicKey,
	}
	jwks := createTestJWKS(keys)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(jwks)
	}))
	defer server.Close()

	cache := NewJWKSCacheWithRotationMonitor(server.URL, nil, 15*time.Minute, "test")

	// Launch concurrent requests
	var wg sync.WaitGroup
	errors := make(chan error, 100)

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx := context.Background()
			_, err := cache.GetKey(ctx, "key-1")
			if err != nil {
				errors <- err
			}
		}()
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("Concurrent access error: %v", err)
	}
}

func TestJWKSCacheWithRotationMonitor_SetURI(t *testing.T) {
	key1 := generateTestRSAKey(t)
	keys := map[string]*rsa.PublicKey{
		"key-1": &key1.PublicKey,
	}
	jwks := createTestJWKS(keys)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(jwks)
	}))
	defer server.Close()

	cache := NewJWKSCacheWithRotationMonitor(server.URL, nil, 15*time.Minute, "test")

	// Populate cache
	ctx := context.Background()
	_, err := cache.GetKey(ctx, "key-1")
	if err != nil {
		t.Fatalf("Initial fetch failed: %v", err)
	}

	// Change URI
	cache.SetURI("https://new-endpoint.example.com/jwks")

	// Verify cache was cleared
	if cache.KeyCount() != 0 {
		t.Errorf("Expected cache to be cleared after SetURI")
	}

	// Verify URI was updated
	if cache.URI() != "https://new-endpoint.example.com/jwks" {
		t.Errorf("URI not updated")
	}
}

func TestJWKSCacheWithRotationMonitor_KeyIDs(t *testing.T) {
	key1 := generateTestRSAKey(t)
	key2 := generateTestRSAKey(t)

	keys := map[string]*rsa.PublicKey{
		"key-alpha": &key1.PublicKey,
		"key-beta":  &key2.PublicKey,
	}
	jwks := createTestJWKS(keys)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(jwks)
	}))
	defer server.Close()

	cache := NewJWKSCacheWithRotationMonitor(server.URL, nil, 15*time.Minute, "test")

	// Populate cache
	ctx := context.Background()
	_, _ = cache.GetKey(ctx, "key-alpha")

	// Get key IDs
	ids := cache.KeyIDs()
	if len(ids) != 2 {
		t.Errorf("Expected 2 key IDs, got %d", len(ids))
	}

	// Verify both IDs are present
	found := make(map[string]bool)
	for _, id := range ids {
		found[id] = true
	}
	if !found["key-alpha"] || !found["key-beta"] {
		t.Error("Expected both key IDs to be present")
	}
}

func TestJWKSCacheWithRotationMonitor_IsExpired(t *testing.T) {
	cache := NewJWKSCacheWithRotationMonitor("https://example.com/jwks", nil, 100*time.Millisecond, "test")

	// Initially should be expired (never fetched)
	if !cache.IsExpired() {
		t.Error("Cache should be expired initially")
	}

	// Simulate a fetch
	cache.mu.Lock()
	cache.fetched = time.Now()
	cache.mu.Unlock()

	// Should not be expired immediately after fetch
	if cache.IsExpired() {
		t.Error("Cache should not be expired right after fetch")
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Should be expired now
	if !cache.IsExpired() {
		t.Error("Cache should be expired after TTL")
	}
}

func TestJWKSCacheWithRotationMonitor_Provider(t *testing.T) {
	cache := NewJWKSCacheWithRotationMonitor("https://example.com/jwks", nil, 15*time.Minute, "my-custom-provider")

	if cache.Provider() != "my-custom-provider" {
		t.Errorf("Expected provider 'my-custom-provider', got %q", cache.Provider())
	}
}
