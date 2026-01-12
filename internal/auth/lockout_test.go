// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestLockoutManager_RecordFailedAttempt(t *testing.T) {
	store := NewMemoryLockoutStore()
	config := &LockoutConfig{
		MaxAttempts:              3,
		LockoutDuration:          5 * time.Minute,
		EnableExponentialBackoff: false,
		Enabled:                  true,
		TrackByIP:                false,
	}
	manager := NewLockoutManager(store, config)

	ctx := context.Background()
	username := "testuser"
	ip := "192.168.1.1"
	ua := "TestBrowser/1.0"

	// First attempt - should not lock
	locked, _, err := manager.RecordFailedAttempt(ctx, username, ip, ua)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if locked {
		t.Error("should not be locked after first attempt")
	}

	// Second attempt - should not lock
	locked, _, err = manager.RecordFailedAttempt(ctx, username, ip, ua)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if locked {
		t.Error("should not be locked after second attempt")
	}

	// Third attempt - should lock
	locked, remaining, err := manager.RecordFailedAttempt(ctx, username, ip, ua)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !locked {
		t.Error("should be locked after third attempt")
	}
	if remaining <= 0 {
		t.Error("remaining time should be positive")
	}
}

func TestLockoutManager_CheckLocked(t *testing.T) {
	store := NewMemoryLockoutStore()
	config := &LockoutConfig{
		MaxAttempts:     2,
		LockoutDuration: 1 * time.Hour,
		Enabled:         true,
		TrackByIP:       false,
	}
	manager := NewLockoutManager(store, config)

	ctx := context.Background()
	username := "testuser"

	// Initially not locked
	locked, _, err := manager.CheckLocked(ctx, username)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if locked {
		t.Error("should not be locked initially")
	}

	// Lock the account
	manager.RecordFailedAttempt(ctx, username, "", "")
	manager.RecordFailedAttempt(ctx, username, "", "")

	// Now should be locked
	locked, remaining, err := manager.CheckLocked(ctx, username)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !locked {
		t.Error("should be locked")
	}
	if remaining <= 0 {
		t.Error("remaining should be positive")
	}
}

func TestLockoutManager_RecordSuccessfulLogin(t *testing.T) {
	store := NewMemoryLockoutStore()
	config := &LockoutConfig{
		MaxAttempts:     3,
		LockoutDuration: 1 * time.Hour,
		Enabled:         true,
		TrackByIP:       false,
	}
	manager := NewLockoutManager(store, config)

	ctx := context.Background()
	username := "testuser"

	// Record some failed attempts (but not enough to lock)
	manager.RecordFailedAttempt(ctx, username, "", "")
	manager.RecordFailedAttempt(ctx, username, "", "")

	// Successful login should clear the state
	err := manager.RecordSuccessfulLogin(ctx, username)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should be able to fail again from scratch
	locked, _, _ := manager.RecordFailedAttempt(ctx, username, "", "")
	if locked {
		t.Error("should not be locked after successful login cleared state")
	}
}

func TestLockoutManager_ExponentialBackoff(t *testing.T) {
	store := NewMemoryLockoutStore()
	config := &LockoutConfig{
		MaxAttempts:              2,
		LockoutDuration:          1 * time.Minute,
		EnableExponentialBackoff: true,
		MaxLockoutDuration:       1 * time.Hour,
		Enabled:                  true,
		TrackByIP:                false,
	}
	manager := NewLockoutManager(store, config)

	ctx := context.Background()
	username := "testuser"

	// First lockout (1 minute)
	manager.RecordFailedAttempt(ctx, username, "", "")
	_, duration1, _ := manager.RecordFailedAttempt(ctx, username, "", "")

	// Wait for lockout to expire (simulate by updating the entry)
	entry, _ := store.GetEntry(ctx, username)
	entry.LockedUntil = time.Now().Add(-1 * time.Second)
	store.SaveEntry(ctx, entry)

	// Second lockout (should be 2 minutes due to exponential backoff)
	manager.RecordFailedAttempt(ctx, username, "", "")
	_, duration2, _ := manager.RecordFailedAttempt(ctx, username, "", "")

	// Second duration should be approximately double (within tolerance for test timing)
	if duration2 <= duration1 {
		t.Errorf("expected exponential backoff: duration2 (%v) should be > duration1 (%v)", duration2, duration1)
	}
}

func TestLockoutManager_Disabled(t *testing.T) {
	store := NewMemoryLockoutStore()
	config := &LockoutConfig{
		MaxAttempts:     1,
		LockoutDuration: 1 * time.Hour,
		Enabled:         false, // Disabled
		TrackByIP:       false,
	}
	manager := NewLockoutManager(store, config)

	ctx := context.Background()
	username := "testuser"

	// Even after exceeding max attempts, should not lock when disabled
	locked, _, err := manager.RecordFailedAttempt(ctx, username, "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if locked {
		t.Error("should not lock when disabled")
	}

	// Check should also return not locked
	locked, _, err = manager.CheckLocked(ctx, username)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if locked {
		t.Error("should not be locked when disabled")
	}
}

func TestLockoutManager_ClearLockout(t *testing.T) {
	store := NewMemoryLockoutStore()
	config := &LockoutConfig{
		MaxAttempts:     2,
		LockoutDuration: 1 * time.Hour,
		Enabled:         true,
		TrackByIP:       false,
	}
	manager := NewLockoutManager(store, config)

	ctx := context.Background()
	username := "testuser"

	// Lock the account
	manager.RecordFailedAttempt(ctx, username, "", "")
	manager.RecordFailedAttempt(ctx, username, "", "")

	// Verify locked
	locked, _, _ := manager.CheckLocked(ctx, username)
	if !locked {
		t.Error("should be locked")
	}

	// Clear the lockout
	err := manager.ClearLockout(ctx, username)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should no longer be locked
	locked, _, _ = manager.CheckLocked(ctx, username)
	if locked {
		t.Error("should not be locked after clear")
	}
}

func TestLockoutManager_TrackByIP(t *testing.T) {
	store := NewMemoryLockoutStore()
	config := &LockoutConfig{
		MaxAttempts:     2,
		LockoutDuration: 1 * time.Hour,
		Enabled:         true,
		TrackByIP:       true,
	}
	manager := NewLockoutManager(store, config)

	ctx := context.Background()
	ip := "192.168.1.100"

	// Different usernames from same IP
	manager.RecordFailedAttempt(ctx, "user1", ip, "")
	manager.RecordFailedAttempt(ctx, "user2", ip, "")

	// IP should be locked
	locked, _, _ := manager.CheckLocked(ctx, "ip:"+ip)
	if !locked {
		t.Error("IP should be locked after max attempts from different users")
	}
}

func TestLockoutEntry_IsLocked(t *testing.T) {
	// Locked entry
	lockedEntry := &LockoutEntry{
		Subject:     "test",
		LockedUntil: time.Now().Add(1 * time.Hour),
	}
	if !lockedEntry.IsLocked() {
		t.Error("entry with future LockedUntil should be locked")
	}

	// Unlocked entry (expired)
	unlockedEntry := &LockoutEntry{
		Subject:     "test",
		LockedUntil: time.Now().Add(-1 * time.Hour),
	}
	if unlockedEntry.IsLocked() {
		t.Error("entry with past LockedUntil should not be locked")
	}

	// Never locked entry
	neverLockedEntry := &LockoutEntry{
		Subject: "test",
	}
	if neverLockedEntry.IsLocked() {
		t.Error("entry with zero LockedUntil should not be locked")
	}
}

func TestMemoryLockoutStore_CleanupExpired(t *testing.T) {
	store := NewMemoryLockoutStore()
	ctx := context.Background()

	// Add an old entry
	oldEntry := &LockoutEntry{
		Subject:     "olduser",
		LastAttempt: time.Now().Add(-48 * time.Hour),
		LockedUntil: time.Now().Add(-47 * time.Hour),
	}
	store.SaveEntry(ctx, oldEntry)

	// Add a recent entry
	recentEntry := &LockoutEntry{
		Subject:     "recentuser",
		LastAttempt: time.Now().Add(-1 * time.Hour),
		LockedUntil: time.Now().Add(-30 * time.Minute),
	}
	store.SaveEntry(ctx, recentEntry)

	// Cleanup
	count, err := store.CleanupExpired(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Old entry should be removed
	if count != 1 {
		t.Errorf("expected 1 entry removed, got %d", count)
	}

	// Recent entry should still exist
	_, err = store.GetEntry(ctx, "recentuser")
	if err != nil {
		t.Error("recent entry should still exist")
	}
}

func TestLockoutManager_Callbacks(t *testing.T) {
	store := NewMemoryLockoutStore()
	config := &LockoutConfig{
		MaxAttempts:     2,
		LockoutDuration: 1 * time.Hour,
		Enabled:         true,
		TrackByIP:       false,
	}
	manager := NewLockoutManager(store, config)

	ctx := context.Background()

	var lockoutCalled atomic.Bool
	var failedLoginCalled atomic.Bool
	var clearCalled atomic.Bool

	manager.SetOnLockout(func(entry *LockoutEntry) {
		lockoutCalled.Store(true)
	})
	manager.SetOnFailedLogin(func(subject, ip, userAgent string) {
		failedLoginCalled.Store(true)
	})
	manager.SetOnLockoutClear(func(subject string) {
		clearCalled.Store(true)
	})

	// Trigger callbacks
	manager.RecordFailedAttempt(ctx, "testuser", "", "")
	manager.RecordFailedAttempt(ctx, "testuser", "", "")
	manager.ClearLockout(ctx, "testuser")

	// Give callbacks time to execute (they run in goroutines)
	time.Sleep(100 * time.Millisecond)

	if !failedLoginCalled.Load() {
		t.Error("onFailedLogin callback should have been called")
	}
	if !lockoutCalled.Load() {
		t.Error("onLockout callback should have been called")
	}
	if !clearCalled.Load() {
		t.Error("onLockoutClear callback should have been called")
	}
}

func TestDefaultLockoutConfig(t *testing.T) {
	config := DefaultLockoutConfig()

	if config == nil {
		t.Fatal("DefaultLockoutConfig should not return nil")
	}
	if config.MaxAttempts != 5 {
		t.Errorf("expected MaxAttempts=5, got %d", config.MaxAttempts)
	}
	if config.LockoutDuration != 15*time.Minute {
		t.Errorf("expected LockoutDuration=15m, got %v", config.LockoutDuration)
	}
	if !config.EnableExponentialBackoff {
		t.Error("expected EnableExponentialBackoff=true")
	}
	if config.MaxLockoutDuration != 24*time.Hour {
		t.Errorf("expected MaxLockoutDuration=24h, got %v", config.MaxLockoutDuration)
	}
	if config.CleanupInterval != 5*time.Minute {
		t.Errorf("expected CleanupInterval=5m, got %v", config.CleanupInterval)
	}
	if !config.TrackByIP {
		t.Error("expected TrackByIP=true")
	}
	if !config.Enabled {
		t.Error("expected Enabled=true")
	}
}

func TestLockoutManager_GetLockedAccounts(t *testing.T) {
	store := NewMemoryLockoutStore()
	config := &LockoutConfig{
		MaxAttempts:     2,
		LockoutDuration: 1 * time.Hour,
		Enabled:         true,
		TrackByIP:       false,
	}
	manager := NewLockoutManager(store, config)
	ctx := context.Background()

	// Initially no locked accounts
	locked, err := manager.GetLockedAccounts(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(locked) != 0 {
		t.Errorf("expected 0 locked accounts, got %d", len(locked))
	}

	// Lock user1
	manager.RecordFailedAttempt(ctx, "user1", "", "")
	manager.RecordFailedAttempt(ctx, "user1", "", "")

	// Lock user2
	manager.RecordFailedAttempt(ctx, "user2", "", "")
	manager.RecordFailedAttempt(ctx, "user2", "", "")

	// Should have 2 locked accounts
	locked, err = manager.GetLockedAccounts(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(locked) != 2 {
		t.Errorf("expected 2 locked accounts, got %d", len(locked))
	}
}

func TestLockoutManager_Config(t *testing.T) {
	store := NewMemoryLockoutStore()
	config := &LockoutConfig{
		MaxAttempts:     7,
		LockoutDuration: 30 * time.Minute,
		Enabled:         true,
		TrackByIP:       true,
	}
	manager := NewLockoutManager(store, config)

	got := manager.Config()
	if got.MaxAttempts != 7 {
		t.Errorf("expected MaxAttempts=7, got %d", got.MaxAttempts)
	}
	if got.LockoutDuration != 30*time.Minute {
		t.Errorf("expected LockoutDuration=30m, got %v", got.LockoutDuration)
	}
	if !got.Enabled {
		t.Error("expected Enabled=true")
	}
	if !got.TrackByIP {
		t.Error("expected TrackByIP=true")
	}
}

func TestLockoutManager_SetEnabled(t *testing.T) {
	store := NewMemoryLockoutStore()
	config := &LockoutConfig{
		MaxAttempts:     2,
		LockoutDuration: 1 * time.Hour,
		Enabled:         true,
		TrackByIP:       false,
	}
	manager := NewLockoutManager(store, config)
	ctx := context.Background()

	// Verify enabled
	if !manager.Config().Enabled {
		t.Error("expected Enabled=true initially")
	}

	// Disable
	manager.SetEnabled(false)
	if manager.Config().Enabled {
		t.Error("expected Enabled=false after SetEnabled(false)")
	}

	// Should not lock when disabled
	manager.RecordFailedAttempt(ctx, "testuser", "", "")
	manager.RecordFailedAttempt(ctx, "testuser", "", "")
	locked, _, _ := manager.CheckLocked(ctx, "testuser")
	if locked {
		t.Error("should not lock when disabled")
	}

	// Re-enable
	manager.SetEnabled(true)
	if !manager.Config().Enabled {
		t.Error("expected Enabled=true after SetEnabled(true)")
	}
}

func TestLockoutManager_StartCleanupRoutine(t *testing.T) {
	store := NewMemoryLockoutStore()
	config := &LockoutConfig{
		MaxAttempts:     2,
		LockoutDuration: 1 * time.Hour,
		Enabled:         true,
		TrackByIP:       false,
		CleanupInterval: 50 * time.Millisecond, // Short interval for testing
	}
	manager := NewLockoutManager(store, config)
	ctx, cancel := context.WithCancel(context.Background())

	// Add an expired entry directly to the store
	expiredEntry := &LockoutEntry{
		Subject:     "expireduser",
		LastAttempt: time.Now().Add(-48 * time.Hour),
		LockedUntil: time.Now().Add(-47 * time.Hour),
	}
	store.SaveEntry(ctx, expiredEntry)

	// Verify entry exists
	_, err := store.GetEntry(ctx, "expireduser")
	if err != nil {
		t.Fatalf("entry should exist before cleanup")
	}

	// Start cleanup routine
	manager.StartCleanupRoutine(ctx)

	// Wait for cleanup to run
	time.Sleep(150 * time.Millisecond)

	// Cancel to stop the routine
	cancel()

	// Verify entry was cleaned up
	_, err = store.GetEntry(ctx, "expireduser")
	if !errors.Is(err, ErrLockoutNotFound) {
		t.Error("expired entry should have been cleaned up")
	}
}

func TestMemoryLockoutStore_ListLockedEntries(t *testing.T) {
	store := NewMemoryLockoutStore()
	ctx := context.Background()

	// Add some entries
	lockedEntry := &LockoutEntry{
		Subject:     "lockeduser",
		LockedUntil: time.Now().Add(1 * time.Hour),
		LastAttempt: time.Now(),
	}
	store.SaveEntry(ctx, lockedEntry)

	unlockedEntry := &LockoutEntry{
		Subject:     "unlockeduser",
		LockedUntil: time.Now().Add(-1 * time.Hour), // Expired
		LastAttempt: time.Now(),
	}
	store.SaveEntry(ctx, unlockedEntry)

	// List locked entries
	locked, err := store.ListLockedEntries(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(locked) != 1 {
		t.Errorf("expected 1 locked entry, got %d", len(locked))
	}
	if len(locked) > 0 && locked[0].Subject != "lockeduser" {
		t.Errorf("expected lockeduser, got %s", locked[0].Subject)
	}
}

func TestLockoutManager_NilConfig(t *testing.T) {
	store := NewMemoryLockoutStore()
	// Pass nil config - should use defaults
	manager := NewLockoutManager(store, nil)

	config := manager.Config()
	if config.MaxAttempts != 5 {
		t.Errorf("expected default MaxAttempts=5, got %d", config.MaxAttempts)
	}
}

func TestLockoutMiddleware(t *testing.T) {
	store := NewMemoryLockoutStore()
	config := &LockoutConfig{
		MaxAttempts:     2,
		LockoutDuration: 1 * time.Hour,
		Enabled:         true,
		TrackByIP:       false,
	}
	manager := NewLockoutManager(store, config)
	ctx := context.Background()

	// Subject extractor that returns username from header
	getSubject := func(r *http.Request) string {
		return r.Header.Get("X-Username")
	}

	// Create a test handler that always returns OK
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Wrap with lockout middleware
	handler := LockoutMiddleware(manager, getSubject)(testHandler)

	t.Run("allows request when not locked", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Username", "alloweduser")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rr.Code)
		}
	})

	t.Run("allows request when no subject", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		// No X-Username header
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rr.Code)
		}
	})

	t.Run("blocks request when locked", func(t *testing.T) {
		// Lock the account
		manager.RecordFailedAttempt(ctx, "lockeduser", "", "")
		manager.RecordFailedAttempt(ctx, "lockeduser", "", "")

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Username", "lockeduser")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusTooManyRequests {
			t.Errorf("expected status 429, got %d", rr.Code)
		}

		// Should have Retry-After header
		retryAfter := rr.Header().Get("Retry-After")
		if retryAfter == "" {
			t.Error("expected Retry-After header")
		}

		// Should have JSON response
		contentType := rr.Header().Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", contentType)
		}
	})
}
