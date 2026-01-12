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
	"fmt"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/goccy/go-json"
)

// BadgerZitadelStateStore implements ZitadelStateStore using BadgerDB.
// This provides ACID-compliant, durable storage for OIDC state parameters.
// State data is persisted to disk and survives server restarts.
//
// ADR-0015: Zero Trust Authentication (Zitadel Amendment)
// Production-grade state storage for OIDC authorization flows.
type BadgerZitadelStateStore struct {
	db *badger.DB
}

// State storage key prefix for namespacing in BadgerDB.
const badgerStateKeyPrefix = "oidc_state:"

// NewBadgerZitadelStateStore creates a new BadgerDB-backed state store.
// The path specifies the directory where BadgerDB will store its data.
//
// Example:
//
//	store, err := NewBadgerZitadelStateStore("/data/sessions")
//	if err != nil {
//	    return err
//	}
//	defer store.Close()
func NewBadgerZitadelStateStore(path string) (*BadgerZitadelStateStore, error) {
	opts := badger.DefaultOptions(path)
	opts.Logger = nil // Suppress BadgerDB internal logs
	// Use value log file size appropriate for small state data
	opts.ValueLogFileSize = 16 << 20 // 16MB (smaller than default 1GB)
	// Enable sync writes for durability (ACID compliance)
	opts.SyncWrites = true

	db, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("open badger db for OIDC state: %w", err)
	}

	return &BadgerZitadelStateStore{db: db}, nil
}

// NewBadgerZitadelStateStoreFromDB creates a state store from an existing BadgerDB connection.
// This is useful when sharing a BadgerDB instance across multiple stores.
func NewBadgerZitadelStateStoreFromDB(db *badger.DB) *BadgerZitadelStateStore {
	return &BadgerZitadelStateStore{db: db}
}

// Store saves state data with the given key.
// The state is stored with a TTL based on ExpiresAt to enable automatic cleanup.
// This operation is ACID-compliant through BadgerDB transactions.
func (s *BadgerZitadelStateStore) Store(ctx context.Context, key string, state *ZitadelStateData) error {
	if key == "" {
		return errors.New("state key cannot be empty")
	}
	if state == nil {
		return errors.New("state data cannot be nil")
	}

	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}

	return s.db.Update(func(txn *badger.Txn) error {
		stateKey := []byte(badgerStateKeyPrefix + key)

		// Create entry with TTL for automatic expiration
		entry := badger.NewEntry(stateKey, data)

		// Set TTL based on state expiration
		ttl := time.Until(state.ExpiresAt)
		if ttl > 0 {
			entry = entry.WithTTL(ttl)
		}

		return txn.SetEntry(entry)
	})
}

// Get retrieves state data by key.
// Returns ErrStateNotFound if the key doesn't exist.
// Returns ErrStateExpired if the state has expired.
func (s *BadgerZitadelStateStore) Get(ctx context.Context, key string) (*ZitadelStateData, error) {
	if key == "" {
		return nil, ErrStateNotFound
	}

	var state ZitadelStateData

	err := s.db.View(func(txn *badger.Txn) error {
		stateKey := []byte(badgerStateKeyPrefix + key)
		item, err := txn.Get(stateKey)
		if errors.Is(err, badger.ErrKeyNotFound) {
			return ErrStateNotFound
		}
		if err != nil {
			return fmt.Errorf("get state: %w", err)
		}

		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &state)
		})
	})

	if err != nil {
		return nil, err
	}

	// Check expiration (belt and suspenders - TTL should handle this)
	if state.IsExpired() {
		// Clean up the expired entry - ignore error as it's best-effort cleanup
		//nolint:errcheck // Best-effort cleanup, state is already expired
		s.Delete(ctx, key)
		return nil, ErrStateExpired
	}

	return &state, nil
}

// Delete removes state data by key.
// This is called after a state is consumed to prevent replay attacks.
func (s *BadgerZitadelStateStore) Delete(ctx context.Context, key string) error {
	if key == "" {
		return nil
	}

	return s.db.Update(func(txn *badger.Txn) error {
		stateKey := []byte(badgerStateKeyPrefix + key)
		err := txn.Delete(stateKey)
		if errors.Is(err, badger.ErrKeyNotFound) {
			return nil // Already deleted
		}
		return err
	})
}

// CleanupExpired removes all expired states.
// Note: BadgerDB TTL handles most expiration automatically, but this
// provides an explicit cleanup mechanism for edge cases.
// Returns the number of states cleaned up.
func (s *BadgerZitadelStateStore) CleanupExpired(ctx context.Context) (int, error) {
	var expiredKeys [][]byte
	now := time.Now()

	// Find expired keys
	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = true
		it := txn.NewIterator(opts)
		defer it.Close()

		prefix := []byte(badgerStateKeyPrefix)
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()

			var state ZitadelStateData
			err := item.Value(func(val []byte) error {
				return json.Unmarshal(val, &state)
			})
			if err != nil {
				// Corrupted entry - mark for deletion
				expiredKeys = append(expiredKeys, append([]byte{}, item.Key()...))
				continue
			}

			if state.ExpiresAt.Before(now) {
				expiredKeys = append(expiredKeys, append([]byte{}, item.Key()...))
			}
		}
		return nil
	})

	if err != nil {
		return 0, fmt.Errorf("scan for expired states: %w", err)
	}

	// Delete expired keys
	count := 0
	for _, key := range expiredKeys {
		err := s.db.Update(func(txn *badger.Txn) error {
			return txn.Delete(key)
		})
		if err == nil {
			count++
		}
	}

	return count, nil
}

// Count returns the number of active (non-expired) states.
func (s *BadgerZitadelStateStore) Count(ctx context.Context) (int, error) {
	count := 0
	now := time.Now()

	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = true
		it := txn.NewIterator(opts)
		defer it.Close()

		prefix := []byte(badgerStateKeyPrefix)
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()

			var state ZitadelStateData
			err := item.Value(func(val []byte) error {
				return json.Unmarshal(val, &state)
			})
			if err != nil {
				continue
			}

			if !state.ExpiresAt.Before(now) {
				count++
			}
		}
		return nil
	})

	return count, err
}

// Close closes the underlying BadgerDB connection.
// Call this when the store is no longer needed.
func (s *BadgerZitadelStateStore) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// GetDB returns the underlying BadgerDB connection.
// This is useful for advanced operations or sharing the connection.
func (s *BadgerZitadelStateStore) GetDB() *badger.DB {
	return s.db
}

// StartCleanupRoutine starts a background goroutine to periodically
// clean up expired states. The cleanup runs at the specified interval.
// The routine stops when the context is canceled.
func (s *BadgerZitadelStateStore) StartCleanupRoutine(ctx context.Context, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				//nolint:errcheck // Background cleanup - errors logged but non-fatal
				s.CleanupExpired(ctx)
				// Also run BadgerDB garbage collection
				//nolint:errcheck // GC errors are non-fatal
				s.db.RunValueLogGC(0.5)
			}
		}
	}()
}

// RunGC runs BadgerDB garbage collection to reclaim space from deleted entries.
// This should be called periodically in production to prevent disk space growth.
func (s *BadgerZitadelStateStore) RunGC() error {
	return s.db.RunValueLogGC(0.5)
}

// Compile-time interface assertion
var _ ZitadelStateStore = (*BadgerZitadelStateStore)(nil)
