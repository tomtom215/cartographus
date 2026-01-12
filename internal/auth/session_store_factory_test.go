// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package auth provides authentication functionality including session management.
// ADR-0015: Zero Trust Authentication & Authorization
package auth

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"
)

// TestNewSessionStoreFactory_Memory tests creating a memory session store factory.
func TestNewSessionStoreFactory_Memory(t *testing.T) {
	factory, err := NewSessionStoreFactory(SessionStoreMemory, "")
	if err != nil {
		t.Fatalf("NewSessionStoreFactory(memory) error: %v", err)
	}
	defer func() {
		_ = factory.Close()
	}()

	if factory.GetDB() != nil {
		t.Error("Memory store factory should have nil DB")
	}

	store := factory.CreateStore()
	if store == nil {
		t.Fatal("CreateStore returned nil")
	}

	// Verify it's a MemorySessionStore by testing basic operations
	ctx := context.Background()
	session := &Session{
		ID:        "test-session-1",
		UserID:    "user1",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(time.Hour),
	}

	if err := store.Create(ctx, session); err != nil {
		t.Errorf("Create session error: %v", err)
	}

	got, err := store.Get(ctx, "test-session-1")
	if err != nil {
		t.Errorf("Get session error: %v", err)
	}
	if got.UserID != "user1" {
		t.Errorf("UserID = %s, want user1", got.UserID)
	}
}

// TestNewSessionStoreFactory_Badger tests creating a BadgerDB session store factory.
func TestNewSessionStoreFactory_Badger(t *testing.T) {
	// Create temp directory for BadgerDB
	tempDir, err := os.MkdirTemp("", "session-test-*")
	if err != nil {
		t.Fatalf("Create temp dir error: %v", err)
	}
	defer os.RemoveAll(tempDir)

	factory, err := NewSessionStoreFactory(SessionStoreBadger, tempDir)
	if err != nil {
		t.Fatalf("NewSessionStoreFactory(badger) error: %v", err)
	}
	defer func() {
		_ = factory.Close()
	}()

	if factory.GetDB() == nil {
		t.Error("Badger store factory should have non-nil DB")
	}

	store := factory.CreateStore()
	if store == nil {
		t.Fatal("CreateStore returned nil")
	}

	// Verify it's a BadgerSessionStore by testing basic operations
	ctx := context.Background()
	session := &Session{
		ID:        "test-session-2",
		UserID:    "user2",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(time.Hour),
	}

	if err := store.Create(ctx, session); err != nil {
		t.Errorf("Create session error: %v", err)
	}

	got, err := store.Get(ctx, "test-session-2")
	if err != nil {
		t.Errorf("Get session error: %v", err)
	}
	if got.UserID != "user2" {
		t.Errorf("UserID = %s, want user2", got.UserID)
	}
}

// TestNewSessionStoreFactory_BadgerInvalidPath tests creating a BadgerDB factory with invalid path.
func TestNewSessionStoreFactory_BadgerInvalidPath(t *testing.T) {
	// Use a path that can't be created (root-owned directory without write permission)
	// On Linux, /proc is a pseudo-filesystem that doesn't allow creating directories
	_, err := NewSessionStoreFactory(SessionStoreBadger, "/proc/1/badger-test")
	if err == nil {
		t.Error("NewSessionStoreFactory should fail with invalid path")
	}
}

// TestNewSessionStoreFactory_Close tests closing the factory.
func TestNewSessionStoreFactory_Close(t *testing.T) {
	// Test memory store close (should be no-op)
	memFactory, err := NewSessionStoreFactory(SessionStoreMemory, "")
	if err != nil {
		t.Fatalf("NewSessionStoreFactory(memory) error: %v", err)
	}
	if err := memFactory.Close(); err != nil {
		t.Errorf("Close memory factory error: %v", err)
	}

	// Test badger store close
	tempDir, err := os.MkdirTemp("", "session-close-test-*")
	if err != nil {
		t.Fatalf("Create temp dir error: %v", err)
	}
	defer os.RemoveAll(tempDir)

	badgerFactory, err := NewSessionStoreFactory(SessionStoreBadger, tempDir)
	if err != nil {
		t.Fatalf("NewSessionStoreFactory(badger) error: %v", err)
	}
	if err := badgerFactory.Close(); err != nil {
		t.Errorf("Close badger factory error: %v", err)
	}
}

// TestBadgerSessionStoreImpl_CRUD tests full CRUD operations on BadgerSessionStore.
func TestBadgerSessionStoreImpl_CRUD(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "session-crud-test-*")
	if err != nil {
		t.Fatalf("Create temp dir error: %v", err)
	}
	defer os.RemoveAll(tempDir)

	factory, err := NewSessionStoreFactory(SessionStoreBadger, tempDir)
	if err != nil {
		t.Fatalf("NewSessionStoreFactory error: %v", err)
	}
	defer func() {
		_ = factory.Close()
	}()

	store := factory.CreateStore()
	ctx := context.Background()

	// Create
	session := &Session{
		ID:             "crud-session",
		UserID:         "crud-user",
		Username:       "testuser",
		Provider:       "oidc",
		Roles:          []string{"viewer"},
		Groups:         []string{"users"},
		CreatedAt:      time.Now(),
		LastAccessedAt: time.Now(),
		ExpiresAt:      time.Now().Add(time.Hour),
		Metadata:       map[string]string{"test": "value"},
	}

	if err := store.Create(ctx, session); err != nil {
		t.Fatalf("Create error: %v", err)
	}

	// Read
	got, err := store.Get(ctx, "crud-session")
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if got.Username != "testuser" {
		t.Errorf("Username = %s, want testuser", got.Username)
	}
	if got.Metadata["test"] != "value" {
		t.Errorf("Metadata[test] = %s, want value", got.Metadata["test"])
	}

	// Update
	session.Username = "updated-user"
	if err := store.Update(ctx, session); err != nil {
		t.Fatalf("Update error: %v", err)
	}

	got, err = store.Get(ctx, "crud-session")
	if err != nil {
		t.Fatalf("Get after update error: %v", err)
	}
	if got.Username != "updated-user" {
		t.Errorf("Updated Username = %s, want updated-user", got.Username)
	}

	// Delete
	if err := store.Delete(ctx, "crud-session"); err != nil {
		t.Fatalf("Delete error: %v", err)
	}

	_, err = store.Get(ctx, "crud-session")
	if !errors.Is(err, ErrSessionNotFound) {
		t.Errorf("Get after delete error = %v, want ErrSessionNotFound", err)
	}
}

// TestBadgerSessionStoreImpl_GetByUserID tests getting sessions by user ID.
func TestBadgerSessionStoreImpl_GetByUserID(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "session-getbyuser-test-*")
	if err != nil {
		t.Fatalf("Create temp dir error: %v", err)
	}
	defer os.RemoveAll(tempDir)

	factory, err := NewSessionStoreFactory(SessionStoreBadger, tempDir)
	if err != nil {
		t.Fatalf("NewSessionStoreFactory error: %v", err)
	}
	defer func() {
		_ = factory.Close()
	}()

	store := factory.CreateStore()
	ctx := context.Background()

	// Create multiple sessions for same user
	for i := 0; i < 3; i++ {
		session := &Session{
			ID:        "multi-session-" + string(rune('a'+i)),
			UserID:    "multi-user",
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(time.Hour),
		}
		if err := store.Create(ctx, session); err != nil {
			t.Fatalf("Create session %d error: %v", i, err)
		}
	}

	// Create session for different user
	otherSession := &Session{
		ID:        "other-session",
		UserID:    "other-user",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(time.Hour),
	}
	if err := store.Create(ctx, otherSession); err != nil {
		t.Fatalf("Create other session error: %v", err)
	}

	// Get sessions for multi-user
	sessions, err := store.GetByUserID(ctx, "multi-user")
	if err != nil {
		t.Fatalf("GetByUserID error: %v", err)
	}
	if len(sessions) != 3 {
		t.Errorf("GetByUserID returned %d sessions, want 3", len(sessions))
	}

	// Get sessions for other-user
	sessions, err = store.GetByUserID(ctx, "other-user")
	if err != nil {
		t.Fatalf("GetByUserID other error: %v", err)
	}
	if len(sessions) != 1 {
		t.Errorf("GetByUserID other returned %d sessions, want 1", len(sessions))
	}
}

// TestBadgerSessionStoreImpl_DeleteByUserID tests deleting all sessions for a user.
func TestBadgerSessionStoreImpl_DeleteByUserID(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "session-deletebyuser-test-*")
	if err != nil {
		t.Fatalf("Create temp dir error: %v", err)
	}
	defer os.RemoveAll(tempDir)

	factory, err := NewSessionStoreFactory(SessionStoreBadger, tempDir)
	if err != nil {
		t.Fatalf("NewSessionStoreFactory error: %v", err)
	}
	defer func() {
		_ = factory.Close()
	}()

	store := factory.CreateStore()
	ctx := context.Background()

	// Create multiple sessions for same user
	for i := 0; i < 3; i++ {
		session := &Session{
			ID:        "del-session-" + string(rune('a'+i)),
			UserID:    "del-user",
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(time.Hour),
		}
		if err := store.Create(ctx, session); err != nil {
			t.Fatalf("Create session %d error: %v", i, err)
		}
	}

	// Delete all sessions for user
	count, err := store.DeleteByUserID(ctx, "del-user")
	if err != nil {
		t.Fatalf("DeleteByUserID error: %v", err)
	}
	if count != 3 {
		t.Errorf("DeleteByUserID deleted %d sessions, want 3", count)
	}

	// Verify sessions are deleted
	sessions, err := store.GetByUserID(ctx, "del-user")
	if err != nil {
		t.Fatalf("GetByUserID after delete error: %v", err)
	}
	if len(sessions) != 0 {
		t.Errorf("GetByUserID after delete returned %d sessions, want 0", len(sessions))
	}
}

// TestBadgerSessionStoreImpl_Touch tests touching a session to extend expiry.
func TestBadgerSessionStoreImpl_Touch(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "session-touch-test-*")
	if err != nil {
		t.Fatalf("Create temp dir error: %v", err)
	}
	defer os.RemoveAll(tempDir)

	factory, err := NewSessionStoreFactory(SessionStoreBadger, tempDir)
	if err != nil {
		t.Fatalf("NewSessionStoreFactory error: %v", err)
	}
	defer func() {
		_ = factory.Close()
	}()

	store := factory.CreateStore()
	ctx := context.Background()

	originalExpiry := time.Now().Add(time.Hour)
	session := &Session{
		ID:             "touch-session",
		UserID:         "touch-user",
		CreatedAt:      time.Now(),
		LastAccessedAt: time.Now(),
		ExpiresAt:      originalExpiry,
	}
	if err := store.Create(ctx, session); err != nil {
		t.Fatalf("Create error: %v", err)
	}

	// Touch with new expiry
	newExpiry := time.Now().Add(2 * time.Hour)
	if err := store.Touch(ctx, "touch-session", newExpiry); err != nil {
		t.Fatalf("Touch error: %v", err)
	}

	// Verify expiry was extended
	got, err := store.Get(ctx, "touch-session")
	if err != nil {
		t.Fatalf("Get after touch error: %v", err)
	}
	if got.ExpiresAt.Before(originalExpiry.Add(30 * time.Minute)) {
		t.Errorf("ExpiresAt should be extended, got %v", got.ExpiresAt)
	}
}

// TestBadgerSessionStoreImpl_ExpiredSession tests that expired sessions return error.
func TestBadgerSessionStoreImpl_ExpiredSession(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "session-expired-test-*")
	if err != nil {
		t.Fatalf("Create temp dir error: %v", err)
	}
	defer os.RemoveAll(tempDir)

	factory, err := NewSessionStoreFactory(SessionStoreBadger, tempDir)
	if err != nil {
		t.Fatalf("NewSessionStoreFactory error: %v", err)
	}
	defer func() {
		_ = factory.Close()
	}()

	store := factory.CreateStore()
	ctx := context.Background()

	// Create expired session
	session := &Session{
		ID:        "expired-session",
		UserID:    "expired-user",
		CreatedAt: time.Now().Add(-2 * time.Hour),
		ExpiresAt: time.Now().Add(-time.Hour), // Already expired
	}
	if err := store.Create(ctx, session); err != nil {
		t.Fatalf("Create error: %v", err)
	}

	// Get should return expired error
	_, err = store.Get(ctx, "expired-session")
	if !errors.Is(err, ErrSessionExpired) {
		t.Errorf("Get expired session error = %v, want ErrSessionExpired", err)
	}
}

// TestBadgerSessionStoreImpl_CleanupExpired tests cleanup of expired sessions.
func TestBadgerSessionStoreImpl_CleanupExpired(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "session-cleanup-test-*")
	if err != nil {
		t.Fatalf("Create temp dir error: %v", err)
	}
	defer os.RemoveAll(tempDir)

	factory, err := NewSessionStoreFactory(SessionStoreBadger, tempDir)
	if err != nil {
		t.Fatalf("NewSessionStoreFactory error: %v", err)
	}
	defer func() {
		_ = factory.Close()
	}()

	store := factory.CreateStore().(*BadgerSessionStoreImpl)
	ctx := context.Background()

	// Create expired sessions
	for i := 0; i < 3; i++ {
		session := &Session{
			ID:        "cleanup-expired-" + string(rune('a'+i)),
			UserID:    "cleanup-user",
			CreatedAt: time.Now().Add(-2 * time.Hour),
			ExpiresAt: time.Now().Add(-time.Hour),
		}
		if err := store.Create(ctx, session); err != nil {
			t.Fatalf("Create expired session %d error: %v", i, err)
		}
	}

	// Create valid sessions
	for i := 0; i < 2; i++ {
		session := &Session{
			ID:        "cleanup-valid-" + string(rune('a'+i)),
			UserID:    "cleanup-user",
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(time.Hour),
		}
		if err := store.Create(ctx, session); err != nil {
			t.Fatalf("Create valid session %d error: %v", i, err)
		}
	}

	// Run cleanup
	count, err := store.CleanupExpired(ctx)
	if err != nil {
		t.Fatalf("CleanupExpired error: %v", err)
	}
	if count != 3 {
		t.Errorf("CleanupExpired removed %d sessions, want 3", count)
	}

	// Verify total count
	total, err := store.Count(ctx)
	if err != nil {
		t.Fatalf("Count error: %v", err)
	}
	if total != 2 {
		t.Errorf("Count after cleanup = %d, want 2", total)
	}
}

// TestBadgerSessionStoreImpl_Count tests counting sessions.
func TestBadgerSessionStoreImpl_Count(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "session-count-test-*")
	if err != nil {
		t.Fatalf("Create temp dir error: %v", err)
	}
	defer os.RemoveAll(tempDir)

	factory, err := NewSessionStoreFactory(SessionStoreBadger, tempDir)
	if err != nil {
		t.Fatalf("NewSessionStoreFactory error: %v", err)
	}
	defer func() {
		_ = factory.Close()
	}()

	store := factory.CreateStore().(*BadgerSessionStoreImpl)
	ctx := context.Background()

	// Initially empty
	count, err := store.Count(ctx)
	if err != nil {
		t.Fatalf("Initial count error: %v", err)
	}
	if count != 0 {
		t.Errorf("Initial count = %d, want 0", count)
	}

	// Add sessions
	for i := 0; i < 5; i++ {
		session := &Session{
			ID:        "count-session-" + string(rune('a'+i)),
			UserID:    "count-user",
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(time.Hour),
		}
		if err := store.Create(ctx, session); err != nil {
			t.Fatalf("Create session %d error: %v", i, err)
		}
	}

	// Verify count
	count, err = store.Count(ctx)
	if err != nil {
		t.Fatalf("Count after create error: %v", err)
	}
	if count != 5 {
		t.Errorf("Count after create = %d, want 5", count)
	}
}

// TestSessionStoreType_Constants tests the session store type constants.
func TestSessionStoreType_Constants(t *testing.T) {
	if SessionStoreMemory != "memory" {
		t.Errorf("SessionStoreMemory = %s, want memory", SessionStoreMemory)
	}
	if SessionStoreBadger != "badger" {
		t.Errorf("SessionStoreBadger = %s, want badger", SessionStoreBadger)
	}
}
