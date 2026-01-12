// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

/*
database_cache.go - Caching Layer for Database Operations

This file implements caching strategies to improve database performance
and reduce redundant query execution.

Caching Systems:

1. Prepared Statement Cache:
  - Caches compiled SQL statements for reuse
  - Uses double-checked locking for thread-safe access
  - Statements are closed when DB.Close() is called
  - Reduces SQL parsing overhead for repeated queries

2. Vector Tile Cache:
  - Caches generated MVT (Mapbox Vector Tiles) for map visualization
  - TTL-based expiration (default 5 minutes)
  - Version-based invalidation when data changes
  - Prometheus metrics for cache hit/miss monitoring

3. Per-IP Locking:
  - Provides mutex locks per IP address for concurrent UPSERT operations
  - Prevents race conditions during geolocation updates
  - Uses sync.Map for lock-free concurrent access to lock registry

Cache Invalidation:
  - IncrementDataVersion(): Called after sync to invalidate stale tiles
  - InvalidateTileCache(): Clears all cached tiles immediately
*/

//nolint:staticcheck // File documentation, not package doc
package database

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/tomtom215/cartographus/internal/metrics"
)

// getTileCached retrieves a vector tile from cache if valid
func (db *DB) getTileCached(cacheKey string) ([]byte, bool) {
	db.tileCacheMu.RLock()
	tile, ok := db.tileCache[cacheKey]
	db.tileCacheMu.RUnlock()

	if !ok {
		metrics.TileCacheMisses.Inc()
		return nil, false
	}

	if time.Now().After(tile.Expires) {
		metrics.TileCacheMisses.Inc()
		return nil, false
	}

	db.dataVersionMu.RLock()
	currentVersion := db.dataVersion
	db.dataVersionMu.RUnlock()

	if tile.Version != currentVersion {
		metrics.TileCacheMisses.Inc()
		return nil, false
	}

	metrics.TileCacheHits.Inc()
	return tile.Data, true
}

// setTileCache stores a vector tile in cache
func (db *DB) setTileCache(cacheKey string, data []byte) {
	db.dataVersionMu.RLock()
	currentVersion := db.dataVersion
	db.dataVersionMu.RUnlock()

	db.tileCacheMu.Lock()
	db.tileCache[cacheKey] = CachedTile{
		Data:    data,
		Version: currentVersion,
		Expires: time.Now().Add(db.tileCacheTTL),
	}
	cacheSize := len(db.tileCache)
	db.tileCacheMu.Unlock()

	metrics.TileCacheSize.Set(float64(cacheSize))
}

// InvalidateTileCache clears all cached tiles
func (db *DB) InvalidateTileCache() {
	db.tileCacheMu.Lock()
	db.tileCache = make(map[string]CachedTile)
	db.tileCacheMu.Unlock()
}

// IncrementDataVersion increments the data version counter
func (db *DB) IncrementDataVersion() {
	db.dataVersionMu.Lock()
	db.dataVersion++
	newVersion := db.dataVersion
	db.dataVersionMu.Unlock()

	metrics.TileCacheDataVersion.Set(float64(newVersion))
}

// acquireIPLock acquires a per-IP mutex lock
func (db *DB) acquireIPLock(ipAddress string) *sync.Mutex {
	muInterface, _ := db.ipLocks.LoadOrStore(ipAddress, &sync.Mutex{})
	mu, ok := muInterface.(*sync.Mutex)
	if !ok {
		mu = &sync.Mutex{}
		db.ipLocks.Store(ipAddress, mu)
	}
	mu.Lock()
	return mu
}

// releaseIPLock releases the per-IP mutex lock
func (db *DB) releaseIPLock(ipAddress string, mu *sync.Mutex) {
	mu.Unlock()
}
