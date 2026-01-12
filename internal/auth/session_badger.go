// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build wal

// Package auth provides authentication functionality including session management.
// ADR-0015: Zero Trust Authentication & Authorization
package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/goccy/go-json"
)

// Key prefixes for BadgerDB storage
const (
	sessionKeyPrefix     = "session:"
	sessionUserKeyPrefix = "session_user:"
)

// BadgerSessionStore implements SessionStore using BadgerDB for durable storage.
// This is suitable for production use with persistence across restarts.
type BadgerSessionStore struct {
	db *badger.DB
}

// NewBadgerSessionStore creates a new BadgerDB-backed session store.
func NewBadgerSessionStore(db *badger.DB) *BadgerSessionStore {
	return &BadgerSessionStore{db: db}
}

// Create stores a new session.
func (s *BadgerSessionStore) Create(ctx context.Context, session *Session) error {
	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}

	return s.db.Update(func(txn *badger.Txn) error {
		// Store session by ID
		sessionKey := []byte(sessionKeyPrefix + session.ID)
		if err := txn.Set(sessionKey, data); err != nil {
			return fmt.Errorf("set session: %w", err)
		}

		// Store user-to-session mapping for efficient lookup
		userKey := []byte(sessionUserKeyPrefix + session.UserID + ":" + session.ID)
		if err := txn.Set(userKey, []byte(session.ID)); err != nil {
			return fmt.Errorf("set user mapping: %w", err)
		}

		return nil
	})
}

// Get retrieves a session by ID.
func (s *BadgerSessionStore) Get(ctx context.Context, id string) (*Session, error) {
	var session Session

	err := s.db.View(func(txn *badger.Txn) error {
		key := []byte(sessionKeyPrefix + id)
		item, err := txn.Get(key)
		if errors.Is(err, badger.ErrKeyNotFound) {
			return ErrSessionNotFound
		}
		if err != nil {
			return fmt.Errorf("get session: %w", err)
		}

		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &session)
		})
	})

	if err != nil {
		return nil, err
	}

	// Check expiration
	if session.IsExpired() {
		return nil, ErrSessionExpired
	}

	return &session, nil
}

// Update updates an existing session.
func (s *BadgerSessionStore) Update(ctx context.Context, session *Session) error {
	// First check if session exists
	_, err := s.Get(ctx, session.ID)
	if err != nil {
		if errors.Is(err, ErrSessionExpired) {
			return ErrSessionNotFound
		}
		return err
	}

	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}

	return s.db.Update(func(txn *badger.Txn) error {
		key := []byte(sessionKeyPrefix + session.ID)
		return txn.Set(key, data)
	})
}

// Delete removes a session by ID.
func (s *BadgerSessionStore) Delete(ctx context.Context, id string) error {
	// Get session first to find user ID for cleanup
	var session Session
	err := s.db.View(func(txn *badger.Txn) error {
		key := []byte(sessionKeyPrefix + id)
		item, err := txn.Get(key)
		if errors.Is(err, badger.ErrKeyNotFound) {
			return nil // Already deleted
		}
		if err != nil {
			return err
		}

		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &session)
		})
	})

	if err != nil {
		return err
	}

	return s.db.Update(func(txn *badger.Txn) error {
		// Delete session
		sessionKey := []byte(sessionKeyPrefix + id)
		if err := txn.Delete(sessionKey); err != nil && !errors.Is(err, badger.ErrKeyNotFound) {
			return fmt.Errorf("delete session: %w", err)
		}

		// Delete user mapping if session was found
		if session.UserID != "" {
			userKey := []byte(sessionUserKeyPrefix + session.UserID + ":" + id)
			if err := txn.Delete(userKey); err != nil && !errors.Is(err, badger.ErrKeyNotFound) {
				return fmt.Errorf("delete user mapping: %w", err)
			}
		}

		return nil
	})
}

// DeleteByUserID removes all sessions for a user.
func (s *BadgerSessionStore) DeleteByUserID(ctx context.Context, userID string) (int, error) {
	// First, get all session IDs for this user
	var sessionIDs []string

	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = true
		it := txn.NewIterator(opts)
		defer it.Close()

		prefix := []byte(sessionUserKeyPrefix + userID + ":")
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			err := item.Value(func(val []byte) error {
				sessionIDs = append(sessionIDs, string(val))
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return 0, fmt.Errorf("list user sessions: %w", err)
	}

	// Delete all sessions
	count := 0
	for _, sessionID := range sessionIDs {
		if err := s.Delete(ctx, sessionID); err != nil {
			continue // Log but continue deleting others
		}
		count++
	}

	return count, nil
}

// GetByUserID returns all sessions for a user.
func (s *BadgerSessionStore) GetByUserID(ctx context.Context, userID string) ([]*Session, error) {
	var sessions []*Session

	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = true
		it := txn.NewIterator(opts)
		defer it.Close()

		prefix := []byte(sessionUserKeyPrefix + userID + ":")
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()

			var sessionID string
			err := item.Value(func(val []byte) error {
				sessionID = string(val)
				return nil
			})
			if err != nil {
				continue
			}

			// Get the actual session
			sessionKey := []byte(sessionKeyPrefix + sessionID)
			sessionItem, err := txn.Get(sessionKey)
			if err != nil {
				continue // Session may have been deleted
			}

			var session Session
			err = sessionItem.Value(func(val []byte) error {
				return json.Unmarshal(val, &session)
			})
			if err != nil {
				continue
			}

			// Skip expired sessions
			if !session.IsExpired() {
				sessions = append(sessions, &session)
			}
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("list user sessions: %w", err)
	}

	return sessions, nil
}

// Touch updates the session's last accessed time and extends expiry.
func (s *BadgerSessionStore) Touch(ctx context.Context, id string, newExpiry time.Time) error {
	return s.db.Update(func(txn *badger.Txn) error {
		key := []byte(sessionKeyPrefix + id)
		item, err := txn.Get(key)
		if errors.Is(err, badger.ErrKeyNotFound) {
			return ErrSessionNotFound
		}
		if err != nil {
			return fmt.Errorf("get session: %w", err)
		}

		var session Session
		err = item.Value(func(val []byte) error {
			return json.Unmarshal(val, &session)
		})
		if err != nil {
			return fmt.Errorf("unmarshal session: %w", err)
		}

		// Update timestamps
		session.LastAccessedAt = time.Now()
		session.ExpiresAt = newExpiry

		data, err := json.Marshal(&session)
		if err != nil {
			return fmt.Errorf("marshal session: %w", err)
		}

		return txn.Set(key, data)
	})
}

// CleanupExpired removes all expired sessions.
func (s *BadgerSessionStore) CleanupExpired(ctx context.Context) (int, error) {
	var expiredIDs []string

	// Find all expired sessions
	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = true
		it := txn.NewIterator(opts)
		defer it.Close()

		prefix := []byte(sessionKeyPrefix)
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()

			var session Session
			err := item.Value(func(val []byte) error {
				return json.Unmarshal(val, &session)
			})
			if err != nil {
				continue
			}

			if session.IsExpired() {
				expiredIDs = append(expiredIDs, session.ID)
			}
		}
		return nil
	})

	if err != nil {
		return 0, fmt.Errorf("scan sessions: %w", err)
	}

	// Delete expired sessions
	count := 0
	for _, id := range expiredIDs {
		if err := s.Delete(ctx, id); err != nil {
			continue
		}
		count++
	}

	return count, nil
}

// StartCleanupRoutine starts a background routine to clean up expired sessions.
func (s *BadgerSessionStore) StartCleanupRoutine(ctx context.Context, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				//nolint:errcheck // Background cleanup - errors are logged but non-fatal
				s.CleanupExpired(ctx)
			}
		}
	}()
}

// Count returns the total number of sessions in the store.
func (s *BadgerSessionStore) Count(ctx context.Context) (int, error) {
	count := 0

	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()

		prefix := []byte(sessionKeyPrefix)
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			count++
		}
		return nil
	})

	return count, err
}
