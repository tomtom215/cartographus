// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package auth provides authentication functionality including OIDC support.
// ADR-0015: Zero Trust Authentication & Authorization
//
// This file implements JTI (JWT ID) tracking for back-channel logout replay prevention.
// Per OIDC Back-Channel Logout specification, the logout token MUST contain a unique
// jti claim, and the RP SHOULD track used jti values to prevent replay attacks.
//
// Security considerations:
//   - JTIs are stored with TTL matching the token lifetime
//   - Replay attempts are logged and counted in metrics
//   - Storage is ACID-compliant when using BadgerDB backend
//   - Memory-based storage available for testing
//
// References:
//   - https://openid.net/specs/openid-connect-backchannel-1_0.html#Validation
//   - Section 2.6: "The RP SHOULD track the jti values it has received..."
package auth

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/goccy/go-json"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/tomtom215/cartographus/internal/logging"
)

// JTI Tracking Metrics
var (
	// OIDCJTIStoreOperationsTotal counts JTI store operations.
	OIDCJTIStoreOperationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "oidc_jti_store_operations_total",
			Help: "Total number of JTI store operations",
		},
		[]string{"operation", "outcome"}, // operation: check, store, cleanup; outcome: success, failure, replay_detected
	)

	// OIDCJTIReplayAttemptsTotal counts detected replay attempts.
	// This is a critical security metric - spikes indicate potential attack.
	OIDCJTIReplayAttemptsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "oidc_jti_replay_attempts_total",
			Help: "Total number of JTI replay attempts detected (potential attacks)",
		},
	)

	// OIDCJTIStoreSize tracks the current number of JTIs in the store.
	OIDCJTIStoreSize = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "oidc_jti_store_size",
			Help: "Current number of JTIs stored for replay prevention",
		},
	)

	// OIDCJTICleanedUpTotal counts JTIs removed during cleanup.
	OIDCJTICleanedUpTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "oidc_jti_cleaned_up_total",
			Help: "Total number of expired JTIs cleaned up",
		},
	)
)

// JTI-related errors
var (
	// ErrJTIAlreadyUsed indicates a replay attack attempt.
	ErrJTIAlreadyUsed = errors.New("JTI already used (replay attack prevented)")

	// ErrJTIStoreClosed indicates the store has been closed.
	ErrJTIStoreClosed = errors.New("JTI store is closed")
)

// JTIEntry represents a stored JTI record.
type JTIEntry struct {
	// JTI is the unique token identifier.
	JTI string `json:"jti"`

	// Issuer is the token issuer.
	Issuer string `json:"iss"`

	// Subject is the affected user.
	Subject string `json:"sub"`

	// SessionID is the affected session (if specified in token).
	SessionID string `json:"sid,omitempty"`

	// FirstSeen is when this JTI was first encountered.
	FirstSeen time.Time `json:"first_seen"`

	// ExpiresAt is when this entry expires (after which replay is irrelevant).
	ExpiresAt time.Time `json:"expires_at"`

	// SourceIP is the IP that submitted the logout request.
	SourceIP string `json:"source_ip,omitempty"`
}

// JTITracker defines the interface for JTI tracking stores.
type JTITracker interface {
	// CheckAndStore atomically checks if a JTI has been seen and stores it if not.
	// Returns ErrJTIAlreadyUsed if the JTI was previously used (replay attack).
	// The entry will expire after the given TTL.
	CheckAndStore(ctx context.Context, entry *JTIEntry, ttl time.Duration) error

	// IsUsed checks if a JTI has been used before without storing it.
	IsUsed(ctx context.Context, jti string) (bool, error)

	// CleanupExpired removes all expired JTI entries.
	// Returns the number of entries removed.
	CleanupExpired(ctx context.Context) (int, error)

	// Size returns the approximate number of entries in the store.
	Size(ctx context.Context) (int, error)

	// Close closes the tracker and releases resources.
	Close() error
}

// MemoryJTITracker is an in-memory JTI tracker for testing.
// Not recommended for production as entries are lost on restart.
type MemoryJTITracker struct {
	mu      sync.RWMutex
	entries map[string]*JTIEntry
	closed  bool
}

// NewMemoryJTITracker creates a new in-memory JTI tracker.
func NewMemoryJTITracker() *MemoryJTITracker {
	return &MemoryJTITracker{
		entries: make(map[string]*JTIEntry),
	}
}

// CheckAndStore atomically checks and stores a JTI.
func (t *MemoryJTITracker) CheckAndStore(ctx context.Context, entry *JTIEntry, ttl time.Duration) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		OIDCJTIStoreOperationsTotal.WithLabelValues("check", "failure").Inc()
		return ErrJTIStoreClosed
	}

	// Check if already exists and not expired
	if existing, ok := t.entries[entry.JTI]; ok {
		if time.Now().Before(existing.ExpiresAt) {
			OIDCJTIStoreOperationsTotal.WithLabelValues("check", "replay_detected").Inc()
			OIDCJTIReplayAttemptsTotal.Inc()
			logging.Warn().
				Str("jti", entry.JTI).
				Str("issuer", entry.Issuer).
				Str("subject", entry.Subject).
				Str("source_ip", entry.SourceIP).
				Time("first_seen", existing.FirstSeen).
				Msg("JTI replay attack detected")
			return ErrJTIAlreadyUsed
		}
		// Entry expired, can reuse
	}

	// Store new entry
	entry.FirstSeen = time.Now()
	entry.ExpiresAt = time.Now().Add(ttl)
	t.entries[entry.JTI] = entry

	OIDCJTIStoreOperationsTotal.WithLabelValues("store", "success").Inc()
	OIDCJTIStoreSize.Set(float64(len(t.entries)))

	return nil
}

// IsUsed checks if a JTI has been used.
func (t *MemoryJTITracker) IsUsed(ctx context.Context, jti string) (bool, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.closed {
		return false, ErrJTIStoreClosed
	}

	entry, ok := t.entries[jti]
	if !ok {
		return false, nil
	}

	// Check if expired
	if time.Now().After(entry.ExpiresAt) {
		return false, nil
	}

	return true, nil
}

// CleanupExpired removes expired entries.
func (t *MemoryJTITracker) CleanupExpired(ctx context.Context) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return 0, ErrJTIStoreClosed
	}

	count := 0
	now := time.Now()
	for jti, entry := range t.entries {
		if now.After(entry.ExpiresAt) {
			delete(t.entries, jti)
			count++
		}
	}

	OIDCJTIStoreOperationsTotal.WithLabelValues("cleanup", "success").Inc()
	OIDCJTICleanedUpTotal.Add(float64(count))
	OIDCJTIStoreSize.Set(float64(len(t.entries)))

	return count, nil
}

// Size returns the number of entries.
func (t *MemoryJTITracker) Size(ctx context.Context) (int, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.closed {
		return 0, ErrJTIStoreClosed
	}

	return len(t.entries), nil
}

// Close closes the tracker.
func (t *MemoryJTITracker) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.closed = true
	t.entries = nil
	return nil
}

// BadgerJTITracker is a BadgerDB-backed JTI tracker for production use.
// Provides ACID-compliant storage that survives restarts.
type BadgerJTITracker struct {
	db     *badger.DB
	prefix []byte
	closed bool
	mu     sync.RWMutex
}

// NewBadgerJTITracker creates a new BadgerDB-backed JTI tracker.
//
// Parameters:
//   - db: BadgerDB instance (shared with other components)
//   - prefix: Key prefix for JTI entries (default: "jti:")
func NewBadgerJTITracker(db *badger.DB, prefix string) *BadgerJTITracker {
	if prefix == "" {
		prefix = "jti:"
	}
	return &BadgerJTITracker{
		db:     db,
		prefix: []byte(prefix),
	}
}

// makeKey creates a BadgerDB key for a JTI.
func (t *BadgerJTITracker) makeKey(jti string) []byte {
	return append(t.prefix, []byte(jti)...)
}

// CheckAndStore atomically checks and stores a JTI.
func (t *BadgerJTITracker) CheckAndStore(ctx context.Context, entry *JTIEntry, ttl time.Duration) error {
	t.mu.RLock()
	if t.closed {
		t.mu.RUnlock()
		OIDCJTIStoreOperationsTotal.WithLabelValues("check", "failure").Inc()
		return ErrJTIStoreClosed
	}
	t.mu.RUnlock()

	key := t.makeKey(entry.JTI)

	err := t.db.Update(func(txn *badger.Txn) error {
		// Check if already exists
		item, err := txn.Get(key)
		if err == nil {
			// Entry exists, check if expired
			var existing JTIEntry
			if valErr := item.Value(func(val []byte) error {
				return json.Unmarshal(val, &existing)
			}); valErr == nil {
				if time.Now().Before(existing.ExpiresAt) {
					// Not expired - this is a replay attack
					OIDCJTIStoreOperationsTotal.WithLabelValues("check", "replay_detected").Inc()
					OIDCJTIReplayAttemptsTotal.Inc()
					logging.Warn().
						Str("jti", entry.JTI).
						Str("issuer", entry.Issuer).
						Str("subject", entry.Subject).
						Str("source_ip", entry.SourceIP).
						Time("first_seen", existing.FirstSeen).
						Msg("JTI replay attack detected")
					return ErrJTIAlreadyUsed
				}
			}
		} else if !errors.Is(err, badger.ErrKeyNotFound) {
			return err
		}

		// Store new entry
		entry.FirstSeen = time.Now()
		entry.ExpiresAt = time.Now().Add(ttl)

		data, err := json.Marshal(entry)
		if err != nil {
			return err
		}

		e := badger.NewEntry(key, data).WithTTL(ttl)
		return txn.SetEntry(e)
	})

	if err != nil {
		if errors.Is(err, ErrJTIAlreadyUsed) {
			return err
		}
		OIDCJTIStoreOperationsTotal.WithLabelValues("store", "failure").Inc()
		return err
	}

	OIDCJTIStoreOperationsTotal.WithLabelValues("store", "success").Inc()
	return nil
}

// IsUsed checks if a JTI has been used.
func (t *BadgerJTITracker) IsUsed(ctx context.Context, jti string) (bool, error) {
	t.mu.RLock()
	if t.closed {
		t.mu.RUnlock()
		return false, ErrJTIStoreClosed
	}
	t.mu.RUnlock()

	key := t.makeKey(jti)
	var used bool

	err := t.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if errors.Is(err, badger.ErrKeyNotFound) {
			used = false
			return nil
		}
		if err != nil {
			return err
		}

		// Check if expired
		var entry JTIEntry
		return item.Value(func(val []byte) error {
			if err := json.Unmarshal(val, &entry); err != nil {
				return err
			}
			used = time.Now().Before(entry.ExpiresAt)
			return nil
		})
	})

	return used, err
}

// CleanupExpired removes expired entries.
// Note: BadgerDB handles TTL expiration automatically, but this forces cleanup.
func (t *BadgerJTITracker) CleanupExpired(ctx context.Context) (int, error) {
	t.mu.RLock()
	if t.closed {
		t.mu.RUnlock()
		return 0, ErrJTIStoreClosed
	}
	t.mu.RUnlock()

	// BadgerDB automatically removes expired entries during compaction
	// We can force garbage collection to clean up
	count := 0
	now := time.Now()

	// Scan for expired entries (belt and suspenders approach)
	err := t.db.Update(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.Prefix = t.prefix
		it := txn.NewIterator(opts)
		defer it.Close()

		var keysToDelete [][]byte

		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			var entry JTIEntry
			err := item.Value(func(val []byte) error {
				return json.Unmarshal(val, &entry)
			})
			if err != nil {
				continue
			}

			if now.After(entry.ExpiresAt) {
				key := make([]byte, len(item.Key()))
				copy(key, item.Key())
				keysToDelete = append(keysToDelete, key)
			}
		}

		for _, key := range keysToDelete {
			if err := txn.Delete(key); err != nil {
				return err
			}
			count++
		}

		return nil
	})

	if err != nil {
		OIDCJTIStoreOperationsTotal.WithLabelValues("cleanup", "failure").Inc()
		return count, err
	}

	OIDCJTIStoreOperationsTotal.WithLabelValues("cleanup", "success").Inc()
	OIDCJTICleanedUpTotal.Add(float64(count))

	return count, nil
}

// Size returns the approximate number of entries.
func (t *BadgerJTITracker) Size(ctx context.Context) (int, error) {
	t.mu.RLock()
	if t.closed {
		t.mu.RUnlock()
		return 0, ErrJTIStoreClosed
	}
	t.mu.RUnlock()

	count := 0
	err := t.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.Prefix = t.prefix
		opts.PrefetchValues = false // We only need to count keys
		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			count++
		}
		return nil
	})

	OIDCJTIStoreSize.Set(float64(count))
	return count, err
}

// Close closes the tracker.
func (t *BadgerJTITracker) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.closed = true
	// Don't close the DB as it's shared
	return nil
}

// StartJTICleanupRoutine starts a background routine to periodically clean up expired JTIs.
// Returns a channel to stop the routine.
func StartJTICleanupRoutine(tracker JTITracker, interval time.Duration) chan struct{} {
	done := make(chan struct{})

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				count, err := tracker.CleanupExpired(ctx)
				cancel()

				if err != nil {
					logging.Error().Err(err).Msg("JTI cleanup failed")
				} else if count > 0 {
					logging.Debug().Int("count", count).Msg("JTI cleanup completed")
				}

			case <-done:
				return
			}
		}
	}()

	return done
}
