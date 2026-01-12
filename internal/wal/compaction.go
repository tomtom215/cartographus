// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build wal

package wal

import (
	"context"
	"sync"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/goccy/go-json"
	"github.com/tomtom215/cartographus/internal/logging"
)

// Compactor handles periodic cleanup of confirmed WAL entries.
// It removes entries that have been successfully published to NATS
// and triggers BadgerDB garbage collection.
type Compactor struct {
	wal    *BadgerWAL
	config Config

	// Control
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// State
	mu      sync.Mutex
	running bool

	// Stats
	lastRun          time.Time
	lastEntriesCount int64
}

// NewCompactor creates a new compaction manager.
func NewCompactor(wal *BadgerWAL) *Compactor {
	return &Compactor{
		wal:    wal,
		config: wal.GetConfig(),
	}
}

// Start begins the background compaction loop.
func (c *Compactor) Start(ctx context.Context) error {
	c.mu.Lock()
	if c.running {
		c.mu.Unlock()
		return nil
	}

	c.ctx, c.cancel = context.WithCancel(ctx)
	c.running = true
	c.mu.Unlock()

	c.wg.Add(1)
	go c.run()

	logging.Info().Dur("interval", c.config.CompactInterval).Msg("WAL compactor started")
	return nil
}

// Stop gracefully stops the compaction loop.
func (c *Compactor) Stop() {
	c.mu.Lock()
	if !c.running {
		c.mu.Unlock()
		return
	}
	c.cancel()
	c.running = false
	c.mu.Unlock()

	c.wg.Wait()
	logging.Info().Msg("WAL compactor stopped")
}

// IsRunning returns whether the compactor is active.
func (c *Compactor) IsRunning() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.running
}

// run is the main compaction loop goroutine.
func (c *Compactor) run() {
	defer c.wg.Done()

	ticker := time.NewTicker(c.config.CompactInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.compact()
		}
	}
}

// compact removes confirmed entries and runs garbage collection.
func (c *Compactor) compact() {
	start := time.Now()

	// Count and delete confirmed entries
	deletedCount, err := c.deleteConfirmedEntries()
	if err != nil {
		logging.Error().Err(err).Msg("WAL compaction failed to delete confirmed entries")
	}

	// Delete expired pending entries
	expiredCount, err := c.deleteExpiredEntries()
	if err != nil {
		logging.Error().Err(err).Msg("WAL compaction failed to delete expired entries")
	}

	totalDeleted := deletedCount + expiredCount

	// Run BadgerDB garbage collection
	if err := c.wal.RunGC(); err != nil {
		logging.Error().Err(err).Msg("WAL compaction GC error")
	}

	// Update stats
	c.mu.Lock()
	c.lastRun = time.Now()
	c.lastEntriesCount = totalDeleted
	c.mu.Unlock()

	// Update metrics
	duration := time.Since(start)
	RecordWALCompaction()
	RecordWALCompactionLatency(duration.Seconds())
	if totalDeleted > 0 {
		RecordWALEntriesCompacted(totalDeleted)
	}

	if totalDeleted > 0 {
		logging.Info().
			Int64("total_deleted", totalDeleted).
			Int64("confirmed", deletedCount).
			Int64("expired", expiredCount).
			Dur("duration", duration).
			Msg("WAL compaction removed entries")
	}
}

// deleteConfirmedEntries removes all entries in the confirmed prefix.
func (c *Compactor) deleteConfirmedEntries() (int64, error) {
	var count int64

	err := c.wal.db.Update(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()

		// Collect keys to delete (can't delete while iterating)
		var keysToDelete [][]byte
		prefix := []byte(prefixConfirmed)
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			keyCopy := make([]byte, len(it.Item().Key()))
			copy(keyCopy, it.Item().Key())
			keysToDelete = append(keysToDelete, keyCopy)
		}

		// Delete collected keys
		for _, key := range keysToDelete {
			if err := txn.Delete(key); err != nil {
				return err
			}
			count++
		}

		return nil
	})

	return count, err
}

// deleteExpiredEntries removes pending entries older than EntryTTL.
func (c *Compactor) deleteExpiredEntries() (int64, error) {
	var count int64
	cutoff := time.Now().Add(-c.config.EntryTTL)

	err := c.wal.db.Update(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = true
		it := txn.NewIterator(opts)
		defer it.Close()

		// Collect expired keys to delete
		var keysToDelete [][]byte
		prefix := []byte(prefixPending)
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()

			var entry Entry
			err := item.Value(func(val []byte) error {
				return json.Unmarshal(val, &entry)
			})
			if err != nil {
				// Skip entries we can't parse
				continue
			}

			if entry.CreatedAt.Before(cutoff) {
				keyCopy := make([]byte, len(item.Key()))
				copy(keyCopy, item.Key())
				keysToDelete = append(keysToDelete, keyCopy)
			}
		}

		// Delete expired keys
		for _, key := range keysToDelete {
			if err := txn.Delete(key); err != nil {
				return err
			}
			count++
			RecordWALExpiredEntry()
		}

		return nil
	})

	return count, err
}

// RunNow triggers an immediate compaction run.
func (c *Compactor) RunNow() error {
	c.compact()
	return nil
}

// GetStats returns compaction statistics.
func (c *Compactor) GetStats() CompactorStats {
	c.mu.Lock()
	defer c.mu.Unlock()

	return CompactorStats{
		LastRun:          c.lastRun,
		LastEntriesCount: c.lastEntriesCount,
	}
}

// CompactorStats contains statistics about compaction.
type CompactorStats struct {
	LastRun          time.Time
	LastEntriesCount int64
}
