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
	"fmt"
	"io"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/goccy/go-json"
)

// SessionStoreType defines the type of session storage backend.
type SessionStoreType string

const (
	// SessionStoreMemory uses in-memory storage (default, not persistent).
	SessionStoreMemory SessionStoreType = "memory"

	// SessionStoreBadger uses BadgerDB for persistent session storage.
	SessionStoreBadger SessionStoreType = "badger"
)

// SessionStoreFactory creates session stores based on configuration.
type SessionStoreFactory struct {
	db *badger.DB
}

// NewSessionStoreFactory creates a new session store factory.
// If storeType is "badger", it opens a BadgerDB at the given path.
// If storeType is "memory" or empty, no database is opened.
func NewSessionStoreFactory(storeType SessionStoreType, path string) (*SessionStoreFactory, error) {
	factory := &SessionStoreFactory{}

	if storeType == SessionStoreBadger {
		opts := badger.DefaultOptions(path)
		opts.Logger = nil // Suppress BadgerDB logs

		db, err := badger.Open(opts)
		if err != nil {
			return nil, fmt.Errorf("open badger db for sessions: %w", err)
		}
		factory.db = db
	}

	return factory, nil
}

// CreateStore creates a SessionStore based on the factory's configuration.
func (f *SessionStoreFactory) CreateStore() SessionStore {
	if f.db != nil {
		return NewBadgerSessionStoreFromDB(f.db)
	}
	return NewMemorySessionStore()
}

// Close closes the underlying BadgerDB if one was opened.
func (f *SessionStoreFactory) Close() error {
	if f.db != nil {
		return f.db.Close()
	}
	return nil
}

// GetDB returns the underlying BadgerDB, or nil if using memory store.
func (f *SessionStoreFactory) GetDB() *badger.DB {
	return f.db
}

// BadgerSessionStoreImpl is a BadgerDB-backed session store implementation.
// This implementation doesn't require the wal build tag.
type BadgerSessionStoreImpl struct {
	db *badger.DB
}

// Session storage key prefixes
const (
	badgerSessionKeyPrefix     = "session:"
	badgerSessionUserKeyPrefix = "session_user:"
)

// NewBadgerSessionStoreFromDB creates a BadgerSessionStore from an existing DB connection.
func NewBadgerSessionStoreFromDB(db *badger.DB) *BadgerSessionStoreImpl {
	return &BadgerSessionStoreImpl{db: db}
}

// Create stores a new session.
func (s *BadgerSessionStoreImpl) Create(ctx context.Context, session *Session) error {
	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}

	return s.db.Update(func(txn *badger.Txn) error {
		// Store session by ID
		sessionKey := []byte(badgerSessionKeyPrefix + session.ID)
		if err := txn.Set(sessionKey, data); err != nil {
			return fmt.Errorf("set session: %w", err)
		}

		// Store user-to-session mapping for efficient lookup
		userKey := []byte(badgerSessionUserKeyPrefix + session.UserID + ":" + session.ID)
		if err := txn.Set(userKey, []byte(session.ID)); err != nil {
			return fmt.Errorf("set user mapping: %w", err)
		}

		return nil
	})
}

// Get retrieves a session by ID.
func (s *BadgerSessionStoreImpl) Get(ctx context.Context, id string) (*Session, error) {
	var session Session

	err := s.db.View(func(txn *badger.Txn) error {
		key := []byte(badgerSessionKeyPrefix + id)
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
func (s *BadgerSessionStoreImpl) Update(ctx context.Context, session *Session) error {
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
		key := []byte(badgerSessionKeyPrefix + session.ID)
		return txn.Set(key, data)
	})
}

// Delete removes a session by ID.
func (s *BadgerSessionStoreImpl) Delete(ctx context.Context, id string) error {
	// Get session first to find user ID for cleanup
	var session Session
	err := s.db.View(func(txn *badger.Txn) error {
		key := []byte(badgerSessionKeyPrefix + id)
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
		sessionKey := []byte(badgerSessionKeyPrefix + id)
		if err := txn.Delete(sessionKey); err != nil && !errors.Is(err, badger.ErrKeyNotFound) {
			return fmt.Errorf("delete session: %w", err)
		}

		// Delete user mapping if session was found
		if session.UserID != "" {
			userKey := []byte(badgerSessionUserKeyPrefix + session.UserID + ":" + id)
			if err := txn.Delete(userKey); err != nil && !errors.Is(err, badger.ErrKeyNotFound) {
				return fmt.Errorf("delete user mapping: %w", err)
			}
		}

		return nil
	})
}

// DeleteByUserID removes all sessions for a user.
func (s *BadgerSessionStoreImpl) DeleteByUserID(ctx context.Context, userID string) (int, error) {
	// First, get all session IDs for this user
	var sessionIDs []string

	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = true
		it := txn.NewIterator(opts)
		defer it.Close()

		prefix := []byte(badgerSessionUserKeyPrefix + userID + ":")
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
func (s *BadgerSessionStoreImpl) GetByUserID(ctx context.Context, userID string) ([]*Session, error) {
	var sessions []*Session

	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = true
		it := txn.NewIterator(opts)
		defer it.Close()

		prefix := []byte(badgerSessionUserKeyPrefix + userID + ":")
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
			sessionKey := []byte(badgerSessionKeyPrefix + sessionID)
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
func (s *BadgerSessionStoreImpl) Touch(ctx context.Context, id string, newExpiry time.Time) error {
	return s.db.Update(func(txn *badger.Txn) error {
		key := []byte(badgerSessionKeyPrefix + id)
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
func (s *BadgerSessionStoreImpl) CleanupExpired(ctx context.Context) (int, error) {
	var expiredIDs []string

	// Find all expired sessions
	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = true
		it := txn.NewIterator(opts)
		defer it.Close()

		prefix := []byte(badgerSessionKeyPrefix)
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
func (s *BadgerSessionStoreImpl) StartCleanupRoutine(ctx context.Context, interval time.Duration) {
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
func (s *BadgerSessionStoreImpl) Count(ctx context.Context) (int, error) {
	count := 0

	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()

		prefix := []byte(badgerSessionKeyPrefix)
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			count++
		}
		return nil
	})

	return count, err
}

// Closeable is an interface for stores that can be closed.
type Closeable interface {
	io.Closer
}
