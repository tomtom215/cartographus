// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package tautulliimport

import (
	"context"
	"errors"
	"fmt"

	"github.com/dgraph-io/badger/v4"
	"github.com/goccy/go-json"
)

const (
	// progressKey is the BadgerDB key for storing import progress.
	progressKey = "import:tautulli:progress"
)

// BadgerProgress implements ProgressTracker using BadgerDB for persistence.
// This enables resumable imports across application restarts.
type BadgerProgress struct {
	db *badger.DB
}

// NewBadgerProgress creates a new progress tracker using the provided BadgerDB instance.
func NewBadgerProgress(db *badger.DB) *BadgerProgress {
	return &BadgerProgress{db: db}
}

// Save persists the current import progress to BadgerDB.
func (p *BadgerProgress) Save(ctx context.Context, stats *ImportStats) error {
	data, err := json.Marshal(stats)
	if err != nil {
		return fmt.Errorf("marshal stats: %w", err)
	}

	return p.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(progressKey), data)
	})
}

// Load retrieves the last saved import progress from BadgerDB.
// Returns nil, nil if no progress has been saved.
func (p *BadgerProgress) Load(ctx context.Context) (*ImportStats, error) {
	var stats ImportStats

	err := p.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(progressKey))
		if errors.Is(err, badger.ErrKeyNotFound) {
			return nil
		}
		if err != nil {
			return err
		}

		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &stats)
		})
	})

	if err != nil {
		return nil, fmt.Errorf("load progress: %w", err)
	}

	// Return nil if no progress was found
	if stats.StartTime.IsZero() {
		return nil, nil
	}

	return &stats, nil
}

// Clear removes saved progress from BadgerDB.
// Use this to start a fresh import.
func (p *BadgerProgress) Clear(ctx context.Context) error {
	return p.db.Update(func(txn *badger.Txn) error {
		err := txn.Delete([]byte(progressKey))
		if errors.Is(err, badger.ErrKeyNotFound) {
			return nil // Already cleared
		}
		return err
	})
}

// InMemoryProgress implements ProgressTracker using in-memory storage.
// This is useful for testing or when persistence is not required.
type InMemoryProgress struct {
	stats *ImportStats
}

// NewInMemoryProgress creates a new in-memory progress tracker.
func NewInMemoryProgress() *InMemoryProgress {
	return &InMemoryProgress{}
}

// Save stores the progress in memory.
func (p *InMemoryProgress) Save(_ context.Context, stats *ImportStats) error {
	// Deep copy to prevent external modifications
	statsCopy := *stats
	p.stats = &statsCopy
	return nil
}

// Load retrieves the progress from memory.
func (p *InMemoryProgress) Load(_ context.Context) (*ImportStats, error) {
	if p.stats == nil {
		return nil, nil
	}
	// Return a copy
	statsCopy := *p.stats
	return &statsCopy, nil
}

// Clear removes the stored progress.
func (p *InMemoryProgress) Clear(_ context.Context) error {
	p.stats = nil
	return nil
}
