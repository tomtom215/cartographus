// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package auth

import (
	"context"
	"errors"
	"testing"
	"time"
)

// =====================================================
// Session Management Tests
// ADR-0015: Zero Trust Authentication & Authorization
// =====================================================

func TestMemorySessionStore_CreateAndGet(t *testing.T) {
	store := NewMemorySessionStore()
	ctx := context.Background()

	// Create a session
	session := &Session{
		ID:        "session-123",
		UserID:    "user-abc",
		Username:  "testuser",
		Email:     "test@example.com",
		Roles:     []string{"viewer"},
		Groups:    []string{"users"},
		Provider:  "oidc",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
		Metadata: map[string]string{
			"issuer": "https://auth.example.com",
		},
	}

	// Store the session
	err := store.Create(ctx, session)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Retrieve the session
	retrieved, err := store.Get(ctx, session.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	// Verify the session data
	if retrieved.ID != session.ID {
		t.Errorf("ID = %v, want %v", retrieved.ID, session.ID)
	}
	if retrieved.UserID != session.UserID {
		t.Errorf("UserID = %v, want %v", retrieved.UserID, session.UserID)
	}
	if retrieved.Username != session.Username {
		t.Errorf("Username = %v, want %v", retrieved.Username, session.Username)
	}
	if retrieved.Provider != session.Provider {
		t.Errorf("Provider = %v, want %v", retrieved.Provider, session.Provider)
	}
	if len(retrieved.Roles) != len(session.Roles) {
		t.Errorf("Roles length = %v, want %v", len(retrieved.Roles), len(session.Roles))
	}
}

func TestMemorySessionStore_GetNonExistent(t *testing.T) {
	store := NewMemorySessionStore()
	ctx := context.Background()

	// Try to get a non-existent session
	_, err := store.Get(ctx, "non-existent-id")
	if !errors.Is(err, ErrSessionNotFound) {
		t.Errorf("Get() error = %v, want %v", err, ErrSessionNotFound)
	}
}

func TestMemorySessionStore_Update(t *testing.T) {
	store := NewMemorySessionStore()
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
	err := store.Create(ctx, session)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Update the session
	session.Roles = []string{"viewer", "editor"}
	session.LastAccessedAt = time.Now()
	err = store.Update(ctx, session)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	// Verify the update
	retrieved, err := store.Get(ctx, session.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if len(retrieved.Roles) != 2 {
		t.Errorf("Roles length = %v, want 2", len(retrieved.Roles))
	}
}

func TestMemorySessionStore_UpdateNonExistent(t *testing.T) {
	store := NewMemorySessionStore()
	ctx := context.Background()

	// Try to update a non-existent session
	session := &Session{
		ID:     "non-existent-id",
		UserID: "user-abc",
	}
	err := store.Update(ctx, session)
	if !errors.Is(err, ErrSessionNotFound) {
		t.Errorf("Update() error = %v, want %v", err, ErrSessionNotFound)
	}
}

func TestMemorySessionStore_Delete(t *testing.T) {
	store := NewMemorySessionStore()
	ctx := context.Background()

	// Create a session
	session := &Session{
		ID:        "session-123",
		UserID:    "user-abc",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	err := store.Create(ctx, session)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Delete the session
	err = store.Delete(ctx, session.ID)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify it's deleted
	_, err = store.Get(ctx, session.ID)
	if !errors.Is(err, ErrSessionNotFound) {
		t.Errorf("Get() after delete error = %v, want %v", err, ErrSessionNotFound)
	}
}

func TestMemorySessionStore_DeleteNonExistent(t *testing.T) {
	store := NewMemorySessionStore()
	ctx := context.Background()

	// Deleting a non-existent session should not error
	err := store.Delete(ctx, "non-existent-id")
	if err != nil {
		t.Errorf("Delete() error = %v, want nil", err)
	}
}

func TestMemorySessionStore_DeleteByUserID(t *testing.T) {
	store := NewMemorySessionStore()
	ctx := context.Background()

	// Create multiple sessions for the same user
	for i := 0; i < 3; i++ {
		session := &Session{
			ID:        "session-" + string(rune('a'+i)),
			UserID:    "user-abc",
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(24 * time.Hour),
		}
		err := store.Create(ctx, session)
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	// Create a session for another user
	otherSession := &Session{
		ID:        "session-other",
		UserID:    "user-def",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	err := store.Create(ctx, otherSession)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Delete all sessions for user-abc
	count, err := store.DeleteByUserID(ctx, "user-abc")
	if err != nil {
		t.Fatalf("DeleteByUserID() error = %v", err)
	}
	if count != 3 {
		t.Errorf("DeleteByUserID() count = %v, want 3", count)
	}

	// Verify user-abc sessions are deleted
	for i := 0; i < 3; i++ {
		_, err = store.Get(ctx, "session-"+string(rune('a'+i)))
		if !errors.Is(err, ErrSessionNotFound) {
			t.Errorf("Get() after DeleteByUserID error = %v, want %v", err, ErrSessionNotFound)
		}
	}

	// Verify other user's session still exists
	_, err = store.Get(ctx, "session-other")
	if err != nil {
		t.Errorf("Other user's session should still exist, got error = %v", err)
	}
}

func TestMemorySessionStore_GetByUserID(t *testing.T) {
	store := NewMemorySessionStore()
	ctx := context.Background()

	// Create multiple sessions for the same user
	for i := 0; i < 3; i++ {
		session := &Session{
			ID:        "session-" + string(rune('a'+i)),
			UserID:    "user-abc",
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(24 * time.Hour),
		}
		err := store.Create(ctx, session)
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	// Get all sessions for user-abc
	sessions, err := store.GetByUserID(ctx, "user-abc")
	if err != nil {
		t.Fatalf("GetByUserID() error = %v", err)
	}
	if len(sessions) != 3 {
		t.Errorf("GetByUserID() count = %v, want 3", len(sessions))
	}
}

func TestMemorySessionStore_ExpiredSessionsNotReturned(t *testing.T) {
	store := NewMemorySessionStore()
	ctx := context.Background()

	// Create an expired session
	session := &Session{
		ID:        "session-expired",
		UserID:    "user-abc",
		CreatedAt: time.Now().Add(-2 * time.Hour),
		ExpiresAt: time.Now().Add(-1 * time.Hour), // Expired 1 hour ago
	}
	err := store.Create(ctx, session)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Try to get the expired session
	_, err = store.Get(ctx, session.ID)
	if !errors.Is(err, ErrSessionExpired) {
		t.Errorf("Get() error = %v, want %v", err, ErrSessionExpired)
	}
}

func TestMemorySessionStore_CleanupExpired(t *testing.T) {
	store := NewMemorySessionStore()
	ctx := context.Background()

	// Create an expired session
	expiredSession := &Session{
		ID:        "session-expired",
		UserID:    "user-abc",
		CreatedAt: time.Now().Add(-2 * time.Hour),
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	}
	err := store.Create(ctx, expiredSession)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Create a valid session
	validSession := &Session{
		ID:        "session-valid",
		UserID:    "user-abc",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	err = store.Create(ctx, validSession)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Run cleanup
	count, err := store.CleanupExpired(ctx)
	if err != nil {
		t.Fatalf("CleanupExpired() error = %v", err)
	}
	if count != 1 {
		t.Errorf("CleanupExpired() count = %v, want 1", count)
	}

	// Verify expired session is gone
	_, err = store.Get(ctx, "session-expired")
	if !errors.Is(err, ErrSessionNotFound) {
		t.Errorf("Get() after cleanup error = %v, want %v", err, ErrSessionNotFound)
	}

	// Verify valid session still exists
	_, err = store.Get(ctx, "session-valid")
	if err != nil {
		t.Errorf("Valid session should still exist, got error = %v", err)
	}
}

func TestSessionStore_Touch(t *testing.T) {
	store := NewMemorySessionStore()
	ctx := context.Background()

	// Create a session
	session := &Session{
		ID:        "session-123",
		UserID:    "user-abc",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	err := store.Create(ctx, session)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Wait a moment
	time.Sleep(10 * time.Millisecond)

	// Touch the session to extend its expiry
	newExpiry := time.Now().Add(2 * time.Hour)
	err = store.Touch(ctx, session.ID, newExpiry)
	if err != nil {
		t.Fatalf("Touch() error = %v", err)
	}

	// Verify the expiry was updated
	retrieved, err := store.Get(ctx, session.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	// The new expiry should be close to what we set
	if retrieved.ExpiresAt.Before(time.Now().Add(1*time.Hour + 50*time.Minute)) {
		t.Errorf("ExpiresAt not properly extended")
	}
}

func TestSession_ToAuthSubject(t *testing.T) {
	session := &Session{
		ID:       "session-123",
		UserID:   "user-abc",
		Username: "testuser",
		Email:    "test@example.com",
		Roles:    []string{"viewer", "editor"},
		Groups:   []string{"users", "developers"},
		Provider: "oidc",
		Metadata: map[string]string{
			"issuer": "https://auth.example.com",
		},
	}

	subject := session.ToAuthSubject()

	if subject.ID != session.UserID {
		t.Errorf("ID = %v, want %v", subject.ID, session.UserID)
	}
	if subject.Username != session.Username {
		t.Errorf("Username = %v, want %v", subject.Username, session.Username)
	}
	if subject.Email != session.Email {
		t.Errorf("Email = %v, want %v", subject.Email, session.Email)
	}
	if len(subject.Roles) != len(session.Roles) {
		t.Errorf("Roles length = %v, want %v", len(subject.Roles), len(session.Roles))
	}
	if len(subject.Groups) != len(session.Groups) {
		t.Errorf("Groups length = %v, want %v", len(subject.Groups), len(session.Groups))
	}
	if subject.Provider != session.Provider {
		t.Errorf("Provider = %v, want %v", subject.Provider, session.Provider)
	}
}

func TestSession_IsExpired(t *testing.T) {
	tests := []struct {
		name      string
		expiresAt time.Time
		want      bool
	}{
		{
			name:      "not expired",
			expiresAt: time.Now().Add(1 * time.Hour),
			want:      false,
		},
		{
			name:      "expired",
			expiresAt: time.Now().Add(-1 * time.Hour),
			want:      true,
		},
		{
			name:      "just expired",
			expiresAt: time.Now().Add(-1 * time.Second),
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := &Session{
				ExpiresAt: tt.expiresAt,
			}
			if got := session.IsExpired(); got != tt.want {
				t.Errorf("IsExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewSession(t *testing.T) {
	subject := &AuthSubject{
		ID:       "user-abc",
		Username: "testuser",
		Email:    "test@example.com",
		Roles:    []string{"viewer"},
		Groups:   []string{"users"},
		Provider: "oidc",
	}

	duration := 24 * time.Hour
	session := NewSession(subject, duration)

	if session.ID == "" {
		t.Error("Session ID should be generated")
	}
	if session.UserID != subject.ID {
		t.Errorf("UserID = %v, want %v", session.UserID, subject.ID)
	}
	if session.Username != subject.Username {
		t.Errorf("Username = %v, want %v", session.Username, subject.Username)
	}
	if session.ExpiresAt.Before(time.Now().Add(duration - 1*time.Minute)) {
		t.Error("ExpiresAt should be approximately duration from now")
	}
}

func TestSessionStoreInterface(t *testing.T) {
	// Verify MemorySessionStore implements SessionStore interface
	var _ SessionStore = (*MemorySessionStore)(nil)
}
