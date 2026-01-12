// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package auth provides authentication functionality including OIDC support.
// ADR-0015: Zero Trust Authentication & Authorization
package auth

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"
)

// TestBadgerZitadelStateStore_BasicOperations tests basic store/get/delete operations.
func TestBadgerZitadelStateStore_BasicOperations(t *testing.T) {
	// Create temporary directory for test database
	tmpDir, err := os.MkdirTemp("", "badger_state_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create store
	store, err := NewBadgerZitadelStateStore(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	t.Run("Store and Get", func(t *testing.T) {
		stateKey := "test-state-123"
		state := &ZitadelStateData{
			CodeVerifier:      "test-code-verifier",
			Nonce:             "test-nonce",
			PostLoginRedirect: "/dashboard",
			ExpiresAt:         time.Now().Add(10 * time.Minute),
		}

		// Store
		err := store.Store(ctx, stateKey, state)
		if err != nil {
			t.Fatalf("Store failed: %v", err)
		}

		// Get
		retrieved, err := store.Get(ctx, stateKey)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}

		if retrieved.CodeVerifier != state.CodeVerifier {
			t.Errorf("CodeVerifier mismatch: got %s, want %s", retrieved.CodeVerifier, state.CodeVerifier)
		}
		if retrieved.Nonce != state.Nonce {
			t.Errorf("Nonce mismatch: got %s, want %s", retrieved.Nonce, state.Nonce)
		}
		if retrieved.PostLoginRedirect != state.PostLoginRedirect {
			t.Errorf("PostLoginRedirect mismatch: got %s, want %s", retrieved.PostLoginRedirect, state.PostLoginRedirect)
		}
	})

	t.Run("Get non-existent key", func(t *testing.T) {
		_, err := store.Get(ctx, "non-existent-key")
		if !errors.Is(err, ErrStateNotFound) {
			t.Errorf("Expected ErrStateNotFound, got: %v", err)
		}
	})

	t.Run("Delete", func(t *testing.T) {
		stateKey := "delete-test-state"
		state := &ZitadelStateData{
			ExpiresAt: time.Now().Add(10 * time.Minute),
		}

		// Store
		err := store.Store(ctx, stateKey, state)
		if err != nil {
			t.Fatalf("Store failed: %v", err)
		}

		// Verify it exists
		_, err = store.Get(ctx, stateKey)
		if err != nil {
			t.Fatalf("Get failed before delete: %v", err)
		}

		// Delete
		err = store.Delete(ctx, stateKey)
		if err != nil {
			t.Fatalf("Delete failed: %v", err)
		}

		// Verify it's gone
		_, err = store.Get(ctx, stateKey)
		if !errors.Is(err, ErrStateNotFound) {
			t.Errorf("Expected ErrStateNotFound after delete, got: %v", err)
		}
	})
}

// TestBadgerZitadelStateStore_Expiration tests that expired states are handled correctly.
func TestBadgerZitadelStateStore_Expiration(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "badger_state_exp_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := NewBadgerZitadelStateStore(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	t.Run("Get expired state returns ErrStateExpired", func(t *testing.T) {
		stateKey := "expired-state"
		state := &ZitadelStateData{
			ExpiresAt: time.Now().Add(-1 * time.Second), // Already expired
		}

		// Store the expired state
		err := store.Store(ctx, stateKey, state)
		if err != nil {
			t.Fatalf("Store failed: %v", err)
		}

		// Get should return ErrStateExpired
		_, err = store.Get(ctx, stateKey)
		if !errors.Is(err, ErrStateExpired) {
			t.Errorf("Expected ErrStateExpired for expired state, got: %v", err)
		}
	})
}

// TestBadgerZitadelStateStore_Count tests the count functionality.
func TestBadgerZitadelStateStore_Count(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "badger_state_count_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := NewBadgerZitadelStateStore(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Initial count should be 0
	count, err := store.Count(ctx)
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}
	if count != 0 {
		t.Errorf("Initial count should be 0, got: %d", count)
	}

	// Add some states
	for i := 0; i < 5; i++ {
		stateKey := "count-test-" + string(rune('a'+i))
		state := &ZitadelStateData{
			ExpiresAt: time.Now().Add(10 * time.Minute),
		}
		if err := store.Store(ctx, stateKey, state); err != nil {
			t.Fatalf("Store failed: %v", err)
		}
	}

	// Count should be 5
	count, err = store.Count(ctx)
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}
	if count != 5 {
		t.Errorf("Count should be 5, got: %d", count)
	}
}

// TestBadgerZitadelStateStore_CleanupExpired tests the cleanup functionality.
func TestBadgerZitadelStateStore_CleanupExpired(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "badger_state_cleanup_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := NewBadgerZitadelStateStore(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Add some valid states
	for i := 0; i < 3; i++ {
		stateKey := "valid-" + string(rune('a'+i))
		state := &ZitadelStateData{
			ExpiresAt: time.Now().Add(10 * time.Minute),
		}
		if err := store.Store(ctx, stateKey, state); err != nil {
			t.Fatalf("Store failed: %v", err)
		}
	}

	// Add some expired states
	for i := 0; i < 2; i++ {
		stateKey := "expired-" + string(rune('a'+i))
		state := &ZitadelStateData{
			ExpiresAt: time.Now().Add(-1 * time.Minute),
		}
		if err := store.Store(ctx, stateKey, state); err != nil {
			t.Fatalf("Store failed: %v", err)
		}
	}

	// Cleanup expired
	cleaned, err := store.CleanupExpired(ctx)
	if err != nil {
		t.Fatalf("CleanupExpired failed: %v", err)
	}

	if cleaned != 2 {
		t.Errorf("Expected 2 expired states to be cleaned, got: %d", cleaned)
	}

	// Count should be 3 (only valid states)
	count, err := store.Count(ctx)
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}
	if count != 3 {
		t.Errorf("Count should be 3 after cleanup, got: %d", count)
	}
}

// TestBadgerZitadelStateStore_Validation tests input validation.
func TestBadgerZitadelStateStore_Validation(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "badger_state_val_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := NewBadgerZitadelStateStore(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	t.Run("Empty key", func(t *testing.T) {
		state := &ZitadelStateData{
			ExpiresAt: time.Now().Add(10 * time.Minute),
		}
		err := store.Store(ctx, "", state)
		if err == nil {
			t.Error("Expected error for empty key")
		}
	})

	t.Run("Nil state", func(t *testing.T) {
		err := store.Store(ctx, "test-key", nil)
		if err == nil {
			t.Error("Expected error for nil state")
		}
	})

	t.Run("Get empty key", func(t *testing.T) {
		_, err := store.Get(ctx, "")
		if !errors.Is(err, ErrStateNotFound) {
			t.Errorf("Expected ErrStateNotFound for empty key, got: %v", err)
		}
	})
}

// TestBadgerZitadelStateStore_InterfaceCompliance verifies interface compliance.
func TestBadgerZitadelStateStore_InterfaceCompliance(t *testing.T) {
	var _ ZitadelStateStore = (*BadgerZitadelStateStore)(nil)
}
