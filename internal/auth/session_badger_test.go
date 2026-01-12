// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build wal

package auth

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/dgraph-io/badger/v4"
)

// =====================================================
// BadgerDB Session Store Tests
// ADR-0015: Zero Trust Authentication & Authorization
// =====================================================

// Helper function to create a test BadgerDB instance
func createTestBadgerDB(t *testing.T) (*badger.DB, func()) {
	t.Helper()

	// Create temp directory
	dir, err := os.MkdirTemp("", "badger-session-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Open BadgerDB
	opts := badger.DefaultOptions(dir)
	opts.Logger = nil // Disable logging for tests
	db, err := badger.Open(opts)
	if err != nil {
		os.RemoveAll(dir)
		t.Fatalf("Failed to open BadgerDB: %v", err)
	}

	cleanup := func() {
		db.Close()
		os.RemoveAll(dir)
	}

	return db, cleanup
}

func TestBadgerSessionStore_Create(t *testing.T) {
	db, cleanup := createTestBadgerDB(t)
	defer cleanup()

	store := NewBadgerSessionStore(db)
	ctx := context.Background()

	session := &Session{
		ID:             "session-123",
		UserID:         "user-abc",
		Username:       "testuser",
		Email:          "test@example.com",
		Roles:          []string{"viewer", "editor"},
		Groups:         []string{"users"},
		Provider:       "oidc",
		CreatedAt:      time.Now(),
		ExpiresAt:      time.Now().Add(24 * time.Hour),
		LastAccessedAt: time.Now(),
		Metadata: map[string]string{
			"key": "value",
		},
	}

	err := store.Create(ctx, session)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Verify session was stored
	retrieved, err := store.Get(ctx, session.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if retrieved.UserID != session.UserID {
		t.Errorf("UserID = %s, want %s", retrieved.UserID, session.UserID)
	}
	if retrieved.Username != session.Username {
		t.Errorf("Username = %s, want %s", retrieved.Username, session.Username)
	}
}

func TestBadgerSessionStore_Get(t *testing.T) {
	db, cleanup := createTestBadgerDB(t)
	defer cleanup()

	store := NewBadgerSessionStore(db)
	ctx := context.Background()

	// Create a session
	session := &Session{
		ID:        "session-123",
		UserID:    "user-abc",
		Username:  "testuser",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	if err := store.Create(ctx, session); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Get the session
	retrieved, err := store.Get(ctx, session.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if retrieved.ID != session.ID {
		t.Errorf("ID = %s, want %s", retrieved.ID, session.ID)
	}
}

func TestBadgerSessionStore_Get_NotFound(t *testing.T) {
	db, cleanup := createTestBadgerDB(t)
	defer cleanup()

	store := NewBadgerSessionStore(db)
	ctx := context.Background()

	_, err := store.Get(ctx, "nonexistent-session")
	if !errors.Is(err, ErrSessionNotFound) {
		t.Errorf("Get() error = %v, want ErrSessionNotFound", err)
	}
}

func TestBadgerSessionStore_Get_Expired(t *testing.T) {
	db, cleanup := createTestBadgerDB(t)
	defer cleanup()

	store := NewBadgerSessionStore(db)
	ctx := context.Background()

	// Create an expired session
	session := &Session{
		ID:        "expired-session",
		UserID:    "user-abc",
		Username:  "testuser",
		CreatedAt: time.Now().Add(-2 * time.Hour),
		ExpiresAt: time.Now().Add(-1 * time.Hour), // Expired
	}
	if err := store.Create(ctx, session); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	_, err := store.Get(ctx, session.ID)
	if !errors.Is(err, ErrSessionExpired) {
		t.Errorf("Get() error = %v, want ErrSessionExpired", err)
	}
}

func TestBadgerSessionStore_Update(t *testing.T) {
	db, cleanup := createTestBadgerDB(t)
	defer cleanup()

	store := NewBadgerSessionStore(db)
	ctx := context.Background()

	// Create a session
	session := &Session{
		ID:        "session-123",
		UserID:    "user-abc",
		Username:  "testuser",
		Roles:     []string{"viewer"},
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	if err := store.Create(ctx, session); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Update the session
	session.Roles = []string{"viewer", "admin"}
	session.LastAccessedAt = time.Now()
	if err := store.Update(ctx, session); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	// Verify update
	retrieved, err := store.Get(ctx, session.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if len(retrieved.Roles) != 2 {
		t.Errorf("Roles length = %d, want 2", len(retrieved.Roles))
	}
}

func TestBadgerSessionStore_Update_NotFound(t *testing.T) {
	db, cleanup := createTestBadgerDB(t)
	defer cleanup()

	store := NewBadgerSessionStore(db)
	ctx := context.Background()

	session := &Session{
		ID:        "nonexistent-session",
		UserID:    "user-abc",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	err := store.Update(ctx, session)
	if !errors.Is(err, ErrSessionNotFound) {
		t.Errorf("Update() error = %v, want ErrSessionNotFound", err)
	}
}

func TestBadgerSessionStore_Delete(t *testing.T) {
	db, cleanup := createTestBadgerDB(t)
	defer cleanup()

	store := NewBadgerSessionStore(db)
	ctx := context.Background()

	// Create a session
	session := &Session{
		ID:        "session-to-delete",
		UserID:    "user-abc",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	if err := store.Create(ctx, session); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Delete the session
	if err := store.Delete(ctx, session.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify deletion
	_, err := store.Get(ctx, session.ID)
	if !errors.Is(err, ErrSessionNotFound) {
		t.Errorf("Get() after Delete() error = %v, want ErrSessionNotFound", err)
	}
}

func TestBadgerSessionStore_DeleteByUserID(t *testing.T) {
	db, cleanup := createTestBadgerDB(t)
	defer cleanup()

	store := NewBadgerSessionStore(db)
	ctx := context.Background()

	userID := "user-abc"

	// Create multiple sessions for the same user
	for i := 0; i < 3; i++ {
		session := &Session{
			ID:        "session-" + string(rune('a'+i)),
			UserID:    userID,
			Username:  "testuser",
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(24 * time.Hour),
		}
		if err := store.Create(ctx, session); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	// Create a session for a different user
	otherSession := &Session{
		ID:        "other-session",
		UserID:    "other-user",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	if err := store.Create(ctx, otherSession); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Delete all sessions for userID
	count, err := store.DeleteByUserID(ctx, userID)
	if err != nil {
		t.Fatalf("DeleteByUserID() error = %v", err)
	}

	if count != 3 {
		t.Errorf("DeleteByUserID() count = %d, want 3", count)
	}

	// Verify user's sessions are deleted
	sessions, err := store.GetByUserID(ctx, userID)
	if err != nil {
		t.Fatalf("GetByUserID() error = %v", err)
	}
	if len(sessions) != 0 {
		t.Errorf("GetByUserID() length = %d, want 0", len(sessions))
	}

	// Verify other user's session still exists
	_, err = store.Get(ctx, otherSession.ID)
	if err != nil {
		t.Errorf("Other user's session should still exist: %v", err)
	}
}

func TestBadgerSessionStore_GetByUserID(t *testing.T) {
	db, cleanup := createTestBadgerDB(t)
	defer cleanup()

	store := NewBadgerSessionStore(db)
	ctx := context.Background()

	userID := "user-abc"

	// Create multiple sessions
	for i := 0; i < 3; i++ {
		session := &Session{
			ID:        "session-" + string(rune('a'+i)),
			UserID:    userID,
			Username:  "testuser",
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(24 * time.Hour),
		}
		if err := store.Create(ctx, session); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	// Get all sessions for user
	sessions, err := store.GetByUserID(ctx, userID)
	if err != nil {
		t.Fatalf("GetByUserID() error = %v", err)
	}

	if len(sessions) != 3 {
		t.Errorf("GetByUserID() length = %d, want 3", len(sessions))
	}
}

func TestBadgerSessionStore_GetByUserID_ExcludesExpired(t *testing.T) {
	db, cleanup := createTestBadgerDB(t)
	defer cleanup()

	store := NewBadgerSessionStore(db)
	ctx := context.Background()

	userID := "user-abc"

	// Create a valid session
	validSession := &Session{
		ID:        "valid-session",
		UserID:    userID,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	if err := store.Create(ctx, validSession); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Create an expired session
	expiredSession := &Session{
		ID:        "expired-session",
		UserID:    userID,
		CreatedAt: time.Now().Add(-2 * time.Hour),
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	}
	if err := store.Create(ctx, expiredSession); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Get sessions - should exclude expired
	sessions, err := store.GetByUserID(ctx, userID)
	if err != nil {
		t.Fatalf("GetByUserID() error = %v", err)
	}

	if len(sessions) != 1 {
		t.Errorf("GetByUserID() length = %d, want 1 (excluding expired)", len(sessions))
	}
}

func TestBadgerSessionStore_Touch(t *testing.T) {
	db, cleanup := createTestBadgerDB(t)
	defer cleanup()

	store := NewBadgerSessionStore(db)
	ctx := context.Background()

	// Create a session
	originalExpiry := time.Now().Add(1 * time.Hour)
	session := &Session{
		ID:             "session-123",
		UserID:         "user-abc",
		CreatedAt:      time.Now(),
		ExpiresAt:      originalExpiry,
		LastAccessedAt: time.Now().Add(-10 * time.Minute),
	}
	if err := store.Create(ctx, session); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Touch the session with new expiry
	newExpiry := time.Now().Add(24 * time.Hour)
	if err := store.Touch(ctx, session.ID, newExpiry); err != nil {
		t.Fatalf("Touch() error = %v", err)
	}

	// Verify update
	retrieved, err := store.Get(ctx, session.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	// LastAccessedAt should be updated
	if retrieved.LastAccessedAt.Before(session.LastAccessedAt) {
		t.Error("LastAccessedAt should be updated")
	}

	// ExpiresAt should be extended
	if retrieved.ExpiresAt.Before(originalExpiry) {
		t.Error("ExpiresAt should be extended")
	}
}

func TestBadgerSessionStore_Touch_NotFound(t *testing.T) {
	db, cleanup := createTestBadgerDB(t)
	defer cleanup()

	store := NewBadgerSessionStore(db)
	ctx := context.Background()

	err := store.Touch(ctx, "nonexistent-session", time.Now().Add(24*time.Hour))
	if !errors.Is(err, ErrSessionNotFound) {
		t.Errorf("Touch() error = %v, want ErrSessionNotFound", err)
	}
}

func TestBadgerSessionStore_CleanupExpired(t *testing.T) {
	db, cleanup := createTestBadgerDB(t)
	defer cleanup()

	store := NewBadgerSessionStore(db)
	ctx := context.Background()

	// Create expired sessions
	for i := 0; i < 3; i++ {
		session := &Session{
			ID:        "expired-" + string(rune('a'+i)),
			UserID:    "user-abc",
			CreatedAt: time.Now().Add(-2 * time.Hour),
			ExpiresAt: time.Now().Add(-1 * time.Hour),
		}
		if err := store.Create(ctx, session); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	// Create valid sessions
	for i := 0; i < 2; i++ {
		session := &Session{
			ID:        "valid-" + string(rune('a'+i)),
			UserID:    "user-abc",
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(24 * time.Hour),
		}
		if err := store.Create(ctx, session); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	// Cleanup
	count, err := store.CleanupExpired(ctx)
	if err != nil {
		t.Fatalf("CleanupExpired() error = %v", err)
	}

	if count != 3 {
		t.Errorf("CleanupExpired() count = %d, want 3", count)
	}

	// Verify expired sessions are gone
	for i := 0; i < 3; i++ {
		_, err := store.Get(ctx, "expired-"+string(rune('a'+i)))
		if !errors.Is(err, ErrSessionNotFound) {
			t.Errorf("Expired session should be deleted")
		}
	}

	// Verify valid sessions still exist
	for i := 0; i < 2; i++ {
		_, err := store.Get(ctx, "valid-"+string(rune('a'+i)))
		if err != nil {
			t.Errorf("Valid session should still exist: %v", err)
		}
	}
}

func TestBadgerSessionStore_DataIntegrity(t *testing.T) {
	db, cleanup := createTestBadgerDB(t)
	defer cleanup()

	store := NewBadgerSessionStore(db)
	ctx := context.Background()

	// Create a session with all fields populated
	original := &Session{
		ID:             "session-123",
		UserID:         "user-abc",
		Username:       "testuser",
		Email:          "test@example.com",
		Roles:          []string{"viewer", "editor", "admin"},
		Groups:         []string{"users", "devs"},
		Provider:       "oidc",
		CreatedAt:      time.Now().Truncate(time.Second), // Truncate for comparison
		ExpiresAt:      time.Now().Add(24 * time.Hour).Truncate(time.Second),
		LastAccessedAt: time.Now().Truncate(time.Second),
		Metadata: map[string]string{
			"key1": "value1",
			"key2": "value2",
		},
	}

	if err := store.Create(ctx, original); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Retrieve and verify all fields
	retrieved, err := store.Get(ctx, original.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	// Verify all fields
	if retrieved.ID != original.ID {
		t.Errorf("ID mismatch")
	}
	if retrieved.UserID != original.UserID {
		t.Errorf("UserID mismatch")
	}
	if retrieved.Username != original.Username {
		t.Errorf("Username mismatch")
	}
	if retrieved.Email != original.Email {
		t.Errorf("Email mismatch")
	}
	if retrieved.Provider != original.Provider {
		t.Errorf("Provider mismatch")
	}

	// Verify slices
	if len(retrieved.Roles) != len(original.Roles) {
		t.Errorf("Roles length mismatch: got %d, want %d", len(retrieved.Roles), len(original.Roles))
	}
	if len(retrieved.Groups) != len(original.Groups) {
		t.Errorf("Groups length mismatch: got %d, want %d", len(retrieved.Groups), len(original.Groups))
	}

	// Verify metadata
	if len(retrieved.Metadata) != len(original.Metadata) {
		t.Errorf("Metadata length mismatch: got %d, want %d", len(retrieved.Metadata), len(original.Metadata))
	}
	for k, v := range original.Metadata {
		if retrieved.Metadata[k] != v {
			t.Errorf("Metadata[%s] = %s, want %s", k, retrieved.Metadata[k], v)
		}
	}
}

func TestBadgerSessionStore_Count(t *testing.T) {
	db, dbCleanup := createTestBadgerDB(t)
	defer dbCleanup()

	store := NewBadgerSessionStore(db)
	ctx := context.Background()

	// Initially empty
	count, err := store.Count(ctx)
	if err != nil {
		t.Fatalf("Count() error = %v", err)
	}
	if count != 0 {
		t.Errorf("Count() = %d, want 0", count)
	}

	// Add some sessions
	for i := 0; i < 5; i++ {
		session := &Session{
			ID:        fmt.Sprintf("session-%d", i),
			UserID:    "test-user",
			Username:  "testuser",
			Provider:  "test",
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(24 * time.Hour),
		}
		if err := store.Create(ctx, session); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	// Verify count
	count, err = store.Count(ctx)
	if err != nil {
		t.Fatalf("Count() error = %v", err)
	}
	if count != 5 {
		t.Errorf("Count() = %d, want 5", count)
	}

	// Delete one
	if err := store.Delete(ctx, "session-0"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	count, err = store.Count(ctx)
	if err != nil {
		t.Fatalf("Count() error = %v", err)
	}
	if count != 4 {
		t.Errorf("Count() = %d, want 4", count)
	}
}

func TestBadgerSessionStore_StartCleanupRoutine(t *testing.T) {
	db, dbCleanup := createTestBadgerDB(t)
	defer dbCleanup()

	store := NewBadgerSessionStore(db)
	ctx, cancel := context.WithCancel(context.Background())

	// Create an expired session
	expiredSession := &Session{
		ID:        "expired-session",
		UserID:    "test-user",
		Username:  "testuser",
		Provider:  "test",
		CreatedAt: time.Now().Add(-2 * time.Hour),
		ExpiresAt: time.Now().Add(-1 * time.Hour), // Already expired
	}
	if err := store.Create(ctx, expiredSession); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Create a valid session
	validSession := &Session{
		ID:        "valid-session",
		UserID:    "test-user",
		Username:  "testuser",
		Provider:  "test",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	if err := store.Create(ctx, validSession); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Verify both sessions exist
	count, _ := store.Count(ctx)
	if count != 2 {
		t.Fatalf("Count() = %d, want 2", count)
	}

	// Start cleanup routine with short interval
	store.StartCleanupRoutine(ctx, 50*time.Millisecond)

	// Wait for cleanup to run
	time.Sleep(150 * time.Millisecond)

	// Cancel context to stop cleanup routine
	cancel()

	// Verify expired session was cleaned up
	_, err := store.Get(ctx, "expired-session")
	if !errors.Is(err, ErrSessionExpired) && !errors.Is(err, ErrSessionNotFound) {
		t.Errorf("Expected ErrSessionExpired or ErrSessionNotFound for expired session, got: %v", err)
	}

	// Verify valid session still exists
	_, err = store.Get(ctx, "valid-session")
	if err != nil {
		t.Errorf("Valid session should still exist, got error: %v", err)
	}
}
