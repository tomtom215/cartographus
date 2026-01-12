// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package auth provides authentication functionality using Zitadel OIDC.
// ADR-0015: Zero Trust Authentication & Authorization (Zitadel Amendment)
//
// This file contains comprehensive tests for ZitadelOIDCFlow.
package auth

import (
	"context"
	"errors"
	"testing"
	"time"
)

// TestZitadelFlowConfig tests flow configuration.
func TestZitadelFlowConfig(t *testing.T) {
	t.Run("default_config", func(t *testing.T) {
		config := DefaultZitadelFlowConfig()

		if config.StateTTL != 10*time.Minute {
			t.Errorf("expected 10m StateTTL, got %v", config.StateTTL)
		}
		if config.SessionDuration != 24*time.Hour {
			t.Errorf("expected 24h SessionDuration, got %v", config.SessionDuration)
		}
		if config.DefaultPostLoginRedirect != "/" {
			t.Errorf("expected '/' DefaultPostLoginRedirect, got %q", config.DefaultPostLoginRedirect)
		}
		if config.ErrorRedirectURL != "/login?error=" {
			t.Errorf("expected '/login?error=' ErrorRedirectURL, got %q", config.ErrorRedirectURL)
		}
		if !config.NonceEnabled {
			t.Error("expected NonceEnabled to be true")
		}
	})
}

// TestZitadelStateData tests state data structure.
func TestZitadelStateData(t *testing.T) {
	t.Run("is_expired_false_for_future", func(t *testing.T) {
		state := &ZitadelStateData{
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(10 * time.Minute),
		}

		if state.IsExpired() {
			t.Error("state should not be expired")
		}
	})

	t.Run("is_expired_true_for_past", func(t *testing.T) {
		state := &ZitadelStateData{
			CreatedAt: time.Now().Add(-20 * time.Minute),
			ExpiresAt: time.Now().Add(-10 * time.Minute),
		}

		if !state.IsExpired() {
			t.Error("state should be expired")
		}
	})
}

// TestZitadelMemoryStateStore tests in-memory state store.
func TestZitadelMemoryStateStore(t *testing.T) {
	ctx := context.Background()

	t.Run("store_and_get", func(t *testing.T) {
		store := NewZitadelMemoryStateStore()

		state := &ZitadelStateData{
			CodeVerifier:      "verifier-123",
			PostLoginRedirect: "/dashboard",
			Nonce:             "nonce-abc",
			CreatedAt:         time.Now(),
			ExpiresAt:         time.Now().Add(10 * time.Minute),
		}

		err := store.Store(ctx, "key-1", state)
		if err != nil {
			t.Fatalf("Store failed: %v", err)
		}

		retrieved, err := store.Get(ctx, "key-1")
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}

		if retrieved.CodeVerifier != state.CodeVerifier {
			t.Errorf("CodeVerifier mismatch: got %q, want %q", retrieved.CodeVerifier, state.CodeVerifier)
		}
		if retrieved.PostLoginRedirect != state.PostLoginRedirect {
			t.Errorf("PostLoginRedirect mismatch")
		}
		if retrieved.Nonce != state.Nonce {
			t.Errorf("Nonce mismatch")
		}
	})

	t.Run("get_nonexistent_returns_error", func(t *testing.T) {
		store := NewZitadelMemoryStateStore()

		_, err := store.Get(ctx, "nonexistent")
		if !errors.Is(err, ErrStateNotFound) {
			t.Errorf("expected ErrStateNotFound, got %v", err)
		}
	})

	t.Run("get_expired_returns_error", func(t *testing.T) {
		store := NewZitadelMemoryStateStore()

		state := &ZitadelStateData{
			CreatedAt: time.Now().Add(-20 * time.Minute),
			ExpiresAt: time.Now().Add(-10 * time.Minute), // Already expired
		}

		err := store.Store(ctx, "expired-key", state)
		if err != nil {
			t.Fatalf("Store failed: %v", err)
		}

		_, err = store.Get(ctx, "expired-key")
		if !errors.Is(err, ErrStateExpired) {
			t.Errorf("expected ErrStateExpired, got %v", err)
		}
	})

	t.Run("delete_removes_state", func(t *testing.T) {
		store := NewZitadelMemoryStateStore()

		state := &ZitadelStateData{
			ExpiresAt: time.Now().Add(10 * time.Minute),
		}

		err := store.Store(ctx, "to-delete", state)
		if err != nil {
			t.Fatalf("Store failed: %v", err)
		}

		err = store.Delete(ctx, "to-delete")
		if err != nil {
			t.Fatalf("Delete failed: %v", err)
		}

		_, err = store.Get(ctx, "to-delete")
		if !errors.Is(err, ErrStateNotFound) {
			t.Errorf("expected ErrStateNotFound after delete, got %v", err)
		}
	})

	t.Run("delete_nonexistent_succeeds", func(t *testing.T) {
		store := NewZitadelMemoryStateStore()

		err := store.Delete(ctx, "nonexistent")
		if err != nil {
			t.Errorf("delete of nonexistent should succeed, got %v", err)
		}
	})

	t.Run("cleanup_expired_removes_old_states", func(t *testing.T) {
		store := NewZitadelMemoryStateStore()

		// Add some expired states
		expiredState := &ZitadelStateData{
			CreatedAt: time.Now().Add(-20 * time.Minute),
			ExpiresAt: time.Now().Add(-10 * time.Minute),
		}
		validState := &ZitadelStateData{
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(10 * time.Minute),
		}

		_ = store.Store(ctx, "expired-1", expiredState)
		_ = store.Store(ctx, "expired-2", expiredState)
		_ = store.Store(ctx, "valid-1", validState)

		count, err := store.CleanupExpired(ctx)
		if err != nil {
			t.Fatalf("CleanupExpired failed: %v", err)
		}
		if count != 2 {
			t.Errorf("expected 2 expired states cleaned, got %d", count)
		}

		// Valid state should still be there
		_, err = store.Get(ctx, "valid-1")
		if err != nil {
			t.Errorf("valid state should still exist: %v", err)
		}

		// Expired states should be gone
		_, err = store.Get(ctx, "expired-1")
		if !errors.Is(err, ErrStateNotFound) {
			t.Errorf("expired state should be cleaned up")
		}
	})

	t.Run("stored_state_is_deep_copy", func(t *testing.T) {
		store := NewZitadelMemoryStateStore()

		original := &ZitadelStateData{
			CodeVerifier: "original",
			ExpiresAt:    time.Now().Add(10 * time.Minute),
		}

		err := store.Store(ctx, "key", original)
		if err != nil {
			t.Fatalf("Store failed: %v", err)
		}

		// Modify original
		original.CodeVerifier = "modified"

		// Retrieve and check
		retrieved, _ := store.Get(ctx, "key")
		if retrieved.CodeVerifier != "original" {
			t.Error("stored state should be a deep copy")
		}
	})

	t.Run("retrieved_state_is_deep_copy", func(t *testing.T) {
		store := NewZitadelMemoryStateStore()

		state := &ZitadelStateData{
			CodeVerifier: "original",
			ExpiresAt:    time.Now().Add(10 * time.Minute),
		}

		_ = store.Store(ctx, "key", state)

		retrieved1, _ := store.Get(ctx, "key")
		retrieved1.CodeVerifier = "modified"

		retrieved2, _ := store.Get(ctx, "key")
		if retrieved2.CodeVerifier != "original" {
			t.Error("retrieved state should be a deep copy")
		}
	})

	t.Run("concurrent_access", func(t *testing.T) {
		store := NewZitadelMemoryStateStore()
		done := make(chan bool)

		// Concurrent writes
		for i := 0; i < 100; i++ {
			go func(idx int) {
				state := &ZitadelStateData{
					ExpiresAt: time.Now().Add(time.Hour),
				}
				_ = store.Store(ctx, "concurrent-key", state)
				done <- true
			}(i)
		}

		// Concurrent reads
		for i := 0; i < 100; i++ {
			go func() {
				_, _ = store.Get(ctx, "concurrent-key")
				done <- true
			}()
		}

		// Wait for all goroutines
		for i := 0; i < 200; i++ {
			<-done
		}
	})
}

// TestNewZitadelOIDCFlow tests flow creation.
func TestNewZitadelOIDCFlow(t *testing.T) {
	t.Run("nil_config_uses_defaults", func(t *testing.T) {
		// We can't create a real RelyingParty without an OIDC server,
		// but we can test that nil config is handled
		stateStore := NewZitadelMemoryStateStore()

		// This would panic if config wasn't handled
		flow := NewZitadelOIDCFlow(nil, stateStore, nil)

		if flow.config.StateTTL != 10*time.Minute {
			t.Error("default config not applied")
		}
	})

	t.Run("custom_config_used", func(t *testing.T) {
		stateStore := NewZitadelMemoryStateStore()
		customConfig := &ZitadelFlowConfig{
			StateTTL:        5 * time.Minute,
			SessionDuration: 12 * time.Hour,
		}

		flow := NewZitadelOIDCFlow(nil, stateStore, customConfig)

		if flow.config.StateTTL != 5*time.Minute {
			t.Error("custom config not used")
		}
	})
}

// TestGenerateSecureRandom tests random string generation.
func TestGenerateSecureRandom(t *testing.T) {
	t.Run("generates_expected_length", func(t *testing.T) {
		result, err := generateSecureRandom(32)
		if err != nil {
			t.Fatalf("generateSecureRandom failed: %v", err)
		}

		// 32 bytes in base64url without padding = 43 characters
		// Actually, base64url encoding of 32 bytes = ceil(32 * 4 / 3) = 43 chars
		if len(result) < 40 {
			t.Errorf("expected ~43 characters, got %d", len(result))
		}
	})

	t.Run("generates_unique_values", func(t *testing.T) {
		results := make(map[string]bool)

		for i := 0; i < 100; i++ {
			result, err := generateSecureRandom(32)
			if err != nil {
				t.Fatalf("generateSecureRandom failed: %v", err)
			}
			if results[result] {
				t.Errorf("duplicate value generated: %q", result)
			}
			results[result] = true
		}
	})

	t.Run("different_sizes", func(t *testing.T) {
		sizes := []int{8, 16, 32, 64}
		for _, size := range sizes {
			result, err := generateSecureRandom(size)
			if err != nil {
				t.Errorf("failed for size %d: %v", size, err)
			}
			if result == "" {
				t.Errorf("empty result for size %d", size)
			}
		}
	})
}

// TestBuiltinMin tests the built-in min function (Go 1.21+).
func TestBuiltinMin(t *testing.T) {
	tests := []struct {
		a, b, want int
	}{
		{1, 2, 1},
		{2, 1, 1},
		{5, 5, 5},
		{0, 1, 0},
		{-1, 0, -1},
		{-1, -2, -2},
	}

	for _, tt := range tests {
		got := min(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("min(%d, %d) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}
