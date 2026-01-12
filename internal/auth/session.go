// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package auth provides authentication middleware with support for multiple auth modes.
// ADR-0015: Zero Trust Authentication & Authorization
package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"
	"time"
)

// Session-related errors
var (
	// ErrSessionNotFound is returned when a session is not found in the store.
	ErrSessionNotFound = errors.New("session not found")

	// ErrSessionExpired is returned when trying to access an expired session.
	ErrSessionExpired = errors.New("session expired")
)

// Session represents an authenticated user session.
// Sessions are created after successful authentication and used to track
// user state across requests.
type Session struct {
	// ID is the unique session identifier (opaque token).
	ID string

	// UserID is the authenticated user's unique identifier.
	UserID string

	// Username is the authenticated user's username.
	Username string

	// Email is the authenticated user's email address.
	Email string

	// Roles are the user's assigned roles for authorization.
	Roles []string

	// Groups are the user's group memberships.
	Groups []string

	// Provider is the authentication provider that created this session.
	// Values: "oidc", "plex", "jwt", "basic"
	Provider string

	// CreatedAt is when the session was created.
	CreatedAt time.Time

	// ExpiresAt is when the session expires.
	ExpiresAt time.Time

	// LastAccessedAt is when the session was last accessed.
	LastAccessedAt time.Time

	// Metadata holds additional session-specific data.
	Metadata map[string]string
}

// IsExpired returns true if the session has expired.
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// ToAuthSubject converts the session to an AuthSubject for use in authorization.
func (s *Session) ToAuthSubject() *AuthSubject {
	return &AuthSubject{
		ID:       s.UserID,
		Username: s.Username,
		Email:    s.Email,
		Roles:    s.Roles,
		Groups:   s.Groups,
		Provider: s.Provider,
		Metadata: s.Metadata,
	}
}

// NewSession creates a new session from an AuthSubject with the given duration.
func NewSession(subject *AuthSubject, duration time.Duration) *Session {
	now := time.Now()
	return &Session{
		ID:             generateSessionID(),
		UserID:         subject.ID,
		Username:       subject.Username,
		Email:          subject.Email,
		Roles:          subject.Roles,
		Groups:         subject.Groups,
		Provider:       subject.Provider,
		CreatedAt:      now,
		ExpiresAt:      now.Add(duration),
		LastAccessedAt: now,
		Metadata:       subject.Metadata,
	}
}

// generateSessionID generates a cryptographically secure session ID.
func generateSessionID() string {
	bytes := make([]byte, 32)
	_, err := rand.Read(bytes)
	if err != nil {
		// Fallback to less secure but still random ID
		return hex.EncodeToString([]byte(time.Now().String()))
	}
	return hex.EncodeToString(bytes)
}

// SessionStore defines the interface for session storage backends.
type SessionStore interface {
	// Create stores a new session.
	Create(ctx context.Context, session *Session) error

	// Get retrieves a session by ID.
	// Returns ErrSessionNotFound if not found.
	// Returns ErrSessionExpired if the session exists but is expired.
	Get(ctx context.Context, id string) (*Session, error)

	// Update updates an existing session.
	// Returns ErrSessionNotFound if not found.
	Update(ctx context.Context, session *Session) error

	// Delete removes a session by ID.
	// Does not return error if session doesn't exist.
	Delete(ctx context.Context, id string) error

	// DeleteByUserID removes all sessions for a user.
	// Returns the count of deleted sessions.
	DeleteByUserID(ctx context.Context, userID string) (int, error)

	// GetByUserID returns all sessions for a user.
	GetByUserID(ctx context.Context, userID string) ([]*Session, error)

	// Touch updates the session's last accessed time and optionally extends expiry.
	Touch(ctx context.Context, id string, newExpiry time.Time) error

	// CleanupExpired removes all expired sessions.
	// Returns the count of deleted sessions.
	CleanupExpired(ctx context.Context) (int, error)
}

// MemorySessionStore is an in-memory implementation of SessionStore.
// Suitable for development and testing. For production, use BadgerDBSessionStore.
type MemorySessionStore struct {
	mu       sync.RWMutex
	sessions map[string]*Session
}

// NewMemorySessionStore creates a new in-memory session store.
func NewMemorySessionStore() *MemorySessionStore {
	return &MemorySessionStore{
		sessions: make(map[string]*Session),
	}
}

// Create stores a new session.
func (s *MemorySessionStore) Create(ctx context.Context, session *Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Deep copy the session to prevent external modifications
	stored := &Session{
		ID:             session.ID,
		UserID:         session.UserID,
		Username:       session.Username,
		Email:          session.Email,
		Provider:       session.Provider,
		CreatedAt:      session.CreatedAt,
		ExpiresAt:      session.ExpiresAt,
		LastAccessedAt: session.LastAccessedAt,
	}

	// Copy slices
	if session.Roles != nil {
		stored.Roles = make([]string, len(session.Roles))
		copy(stored.Roles, session.Roles)
	}
	if session.Groups != nil {
		stored.Groups = make([]string, len(session.Groups))
		copy(stored.Groups, session.Groups)
	}
	if session.Metadata != nil {
		stored.Metadata = make(map[string]string)
		for k, v := range session.Metadata {
			stored.Metadata[k] = v
		}
	}

	s.sessions[session.ID] = stored
	return nil
}

// Get retrieves a session by ID.
func (s *MemorySessionStore) Get(ctx context.Context, id string) (*Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, ok := s.sessions[id]
	if !ok {
		return nil, ErrSessionNotFound
	}

	if session.IsExpired() {
		return nil, ErrSessionExpired
	}

	// Return a copy to prevent external modifications
	return s.copySession(session), nil
}

// Update updates an existing session.
func (s *MemorySessionStore) Update(ctx context.Context, session *Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.sessions[session.ID]; !ok {
		return ErrSessionNotFound
	}

	// Deep copy the session
	stored := &Session{
		ID:             session.ID,
		UserID:         session.UserID,
		Username:       session.Username,
		Email:          session.Email,
		Provider:       session.Provider,
		CreatedAt:      session.CreatedAt,
		ExpiresAt:      session.ExpiresAt,
		LastAccessedAt: session.LastAccessedAt,
	}

	if session.Roles != nil {
		stored.Roles = make([]string, len(session.Roles))
		copy(stored.Roles, session.Roles)
	}
	if session.Groups != nil {
		stored.Groups = make([]string, len(session.Groups))
		copy(stored.Groups, session.Groups)
	}
	if session.Metadata != nil {
		stored.Metadata = make(map[string]string)
		for k, v := range session.Metadata {
			stored.Metadata[k] = v
		}
	}

	s.sessions[session.ID] = stored
	return nil
}

// Delete removes a session by ID.
func (s *MemorySessionStore) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.sessions, id)
	return nil
}

// DeleteByUserID removes all sessions for a user.
func (s *MemorySessionStore) DeleteByUserID(ctx context.Context, userID string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	count := 0
	for id, session := range s.sessions {
		if session.UserID == userID {
			delete(s.sessions, id)
			count++
		}
	}
	return count, nil
}

// GetByUserID returns all sessions for a user.
func (s *MemorySessionStore) GetByUserID(ctx context.Context, userID string) ([]*Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var sessions []*Session
	for _, session := range s.sessions {
		if session.UserID == userID && !session.IsExpired() {
			sessions = append(sessions, s.copySession(session))
		}
	}
	return sessions, nil
}

// Touch updates the session's last accessed time and extends expiry.
func (s *MemorySessionStore) Touch(ctx context.Context, id string, newExpiry time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[id]
	if !ok {
		return ErrSessionNotFound
	}

	session.LastAccessedAt = time.Now()
	session.ExpiresAt = newExpiry
	return nil
}

// CleanupExpired removes all expired sessions.
func (s *MemorySessionStore) CleanupExpired(ctx context.Context) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	count := 0
	for id, session := range s.sessions {
		if session.IsExpired() {
			delete(s.sessions, id)
			count++
		}
	}
	return count, nil
}

// copySession creates a deep copy of a session.
func (s *MemorySessionStore) copySession(session *Session) *Session {
	copied := &Session{
		ID:             session.ID,
		UserID:         session.UserID,
		Username:       session.Username,
		Email:          session.Email,
		Provider:       session.Provider,
		CreatedAt:      session.CreatedAt,
		ExpiresAt:      session.ExpiresAt,
		LastAccessedAt: session.LastAccessedAt,
	}

	if session.Roles != nil {
		copied.Roles = make([]string, len(session.Roles))
		copy(copied.Roles, session.Roles)
	}
	if session.Groups != nil {
		copied.Groups = make([]string, len(session.Groups))
		copy(copied.Groups, session.Groups)
	}
	if session.Metadata != nil {
		copied.Metadata = make(map[string]string)
		for k, v := range session.Metadata {
			copied.Metadata[k] = v
		}
	}

	return copied
}

// StartCleanupRoutine starts a goroutine that periodically cleans up expired sessions.
// Returns a channel that should be closed to stop the routine.
func (s *MemorySessionStore) StartCleanupRoutine(interval time.Duration) chan struct{} {
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				//nolint:errcheck // Background cleanup - errors are non-critical
				s.CleanupExpired(context.Background())
			case <-done:
				return
			}
		}
	}()
	return done
}
