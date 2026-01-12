// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package authz

import (
	"testing"
	"time"
)

// =====================================================
// Cache Unit Tests
// ADR-0015: Zero Trust Authentication & Authorization
// =====================================================

func TestNewEnforcementCache(t *testing.T) {
	// Test with positive TTL
	cache := newEnforcementCache(5 * time.Minute)
	if cache == nil {
		t.Fatal("newEnforcementCache() returned nil")
	}
	defer cache.stop()

	if cache.ttl != 5*time.Minute {
		t.Errorf("cache.ttl = %v, want 5m", cache.ttl)
	}
}

func TestNewEnforcementCache_ZeroTTL(t *testing.T) {
	// Zero TTL should use default
	cache := newEnforcementCache(0)
	if cache == nil {
		t.Fatal("newEnforcementCache() returned nil")
	}
	defer cache.stop()

	if cache.ttl != 5*time.Minute {
		t.Errorf("cache.ttl = %v, want 5m (default)", cache.ttl)
	}
}

func TestNewEnforcementCache_NegativeTTL(t *testing.T) {
	// Negative TTL should use default
	cache := newEnforcementCache(-1 * time.Second)
	if cache == nil {
		t.Fatal("newEnforcementCache() returned nil")
	}
	defer cache.stop()

	if cache.ttl != 5*time.Minute {
		t.Errorf("cache.ttl = %v, want 5m (default)", cache.ttl)
	}
}

func TestEnforcementCache_Key(t *testing.T) {
	cache := newEnforcementCache(5 * time.Minute)
	defer cache.stop()

	key := cache.key("user1", "/api/resource", "GET")
	expected := "user1:/api/resource:GET"

	if key != expected {
		t.Errorf("key() = %q, want %q", key, expected)
	}
}

func TestEnforcementCache_SetAndGet(t *testing.T) {
	cache := newEnforcementCache(5 * time.Minute)
	defer cache.stop()

	// Set a value
	cache.set("user1", "/api/resource", "GET", true)

	// Get it back
	allowed, found := cache.get("user1", "/api/resource", "GET")
	if !found {
		t.Error("Expected to find cached value")
	}
	if !allowed {
		t.Error("Expected allowed to be true")
	}

	// Set a denied value
	cache.set("user2", "/api/admin", "DELETE", false)

	// Get it back
	allowed, found = cache.get("user2", "/api/admin", "DELETE")
	if !found {
		t.Error("Expected to find cached value")
	}
	if allowed {
		t.Error("Expected allowed to be false")
	}
}

func TestEnforcementCache_Get_NotFound(t *testing.T) {
	cache := newEnforcementCache(5 * time.Minute)
	defer cache.stop()

	// Get non-existent key
	allowed, found := cache.get("nonexistent", "/api/resource", "GET")
	if found {
		t.Error("Expected not to find non-existent key")
	}
	if allowed {
		t.Error("Expected allowed to be false for not found")
	}
}

func TestEnforcementCache_Get_Expired(t *testing.T) {
	// Use a very short TTL
	cache := newEnforcementCache(1 * time.Millisecond)
	defer cache.stop()

	// Set a value
	cache.set("user1", "/api/resource", "GET", true)

	// Wait for expiration
	time.Sleep(10 * time.Millisecond)

	// Should not be found (expired)
	_, found := cache.get("user1", "/api/resource", "GET")
	if found {
		t.Error("Expected expired item to not be found")
	}
}

func TestEnforcementCache_InvalidateUser(t *testing.T) {
	cache := newEnforcementCache(5 * time.Minute)
	defer cache.stop()

	// Set multiple values for the same user
	cache.set("user1", "/api/resource1", "GET", true)
	cache.set("user1", "/api/resource2", "POST", true)
	cache.set("user2", "/api/resource1", "GET", true)

	// Invalidate user1
	cache.invalidateUser("user1")

	// user1's entries should be gone
	_, found := cache.get("user1", "/api/resource1", "GET")
	if found {
		t.Error("user1's entry should be invalidated")
	}

	_, found = cache.get("user1", "/api/resource2", "POST")
	if found {
		t.Error("user1's other entry should be invalidated")
	}

	// user2's entry should still exist
	_, found = cache.get("user2", "/api/resource1", "GET")
	if !found {
		t.Error("user2's entry should not be affected")
	}
}

func TestEnforcementCache_Clear(t *testing.T) {
	cache := newEnforcementCache(5 * time.Minute)
	defer cache.stop()

	// Set multiple values
	cache.set("user1", "/api/resource1", "GET", true)
	cache.set("user2", "/api/resource2", "POST", true)

	// Clear all
	cache.clear()

	// All entries should be gone
	_, found1 := cache.get("user1", "/api/resource1", "GET")
	_, found2 := cache.get("user2", "/api/resource2", "POST")

	if found1 || found2 {
		t.Error("All entries should be cleared")
	}
}

func TestEnforcementCache_Stop(t *testing.T) {
	cache := newEnforcementCache(5 * time.Minute)

	// Stop should not panic
	cache.stop()

	// Stopping again should not panic (idempotent - uses sync.Once)
	// ADR-0015 Phase 4 Continuation: Verify Close() is idempotent
	cache.stop()
	cache.stop() // Third call should also be safe
}

func TestEnforcementCache_StopIdempotent(t *testing.T) {
	// Specifically test that multiple calls to stop() don't panic
	cache := newEnforcementCache(5 * time.Minute)

	// Run multiple concurrent stops - none should panic
	done := make(chan bool, 5)
	for i := 0; i < 5; i++ {
		go func() {
			cache.stop()
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 5; i++ {
		<-done
	}
}

func TestEnforcementCache_ConcurrentAccess(t *testing.T) {
	cache := newEnforcementCache(5 * time.Minute)
	defer cache.stop()

	// Run concurrent operations
	done := make(chan bool, 3)

	// Writer 1
	go func() {
		for i := 0; i < 100; i++ {
			cache.set("user1", "/api/resource", "GET", true)
		}
		done <- true
	}()

	// Writer 2
	go func() {
		for i := 0; i < 100; i++ {
			cache.set("user2", "/api/resource", "POST", false)
		}
		done <- true
	}()

	// Reader
	go func() {
		for i := 0; i < 100; i++ {
			cache.get("user1", "/api/resource", "GET")
			cache.get("user2", "/api/resource", "POST")
		}
		done <- true
	}()

	// Wait for all goroutines
	for i := 0; i < 3; i++ {
		<-done
	}
}

func TestEnforcementCache_InvalidateUserEdgeCases(t *testing.T) {
	cache := newEnforcementCache(5 * time.Minute)
	defer cache.stop()

	// Invalidate non-existent user (should not panic)
	cache.invalidateUser("nonexistent")

	// Invalidate empty user (should not panic)
	cache.invalidateUser("")

	// Set a value with empty user
	cache.set("", "/api/resource", "GET", true)

	// Should be able to get it
	_, found := cache.get("", "/api/resource", "GET")
	if !found {
		t.Error("Should find entry with empty user")
	}

	// Invalidate empty user
	cache.invalidateUser("")

	// Should not find it anymore
	_, found = cache.get("", "/api/resource", "GET")
	if found {
		t.Error("Entry with empty user should be invalidated")
	}
}

// Benchmark tests
func BenchmarkCache_Set(b *testing.B) {
	cache := newEnforcementCache(5 * time.Minute)
	defer cache.stop()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.set("user1", "/api/resource", "GET", true)
	}
}

func BenchmarkCache_Get(b *testing.B) {
	cache := newEnforcementCache(5 * time.Minute)
	defer cache.stop()

	cache.set("user1", "/api/resource", "GET", true)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.get("user1", "/api/resource", "GET")
	}
}

func BenchmarkCache_Key(b *testing.B) {
	cache := newEnforcementCache(5 * time.Minute)
	defer cache.stop()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.key("user1", "/api/resource", "GET")
	}
}
