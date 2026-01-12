// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package cache provides high-performance data structures for caching and deduplication.
package cache

import (
	"math"
	"sync"
	"time"
)

// SpatialHashGrid divides geographic space into cells for fast proximity queries.
// Instead of O(n) comparisons to find nearby events, we only check cells near
// the query point, reducing to O(k) where k = number of nearby entries.
//
// Use cases:
//   - Impossible travel detection: Only check events in nearby cells
//   - Geo-clustering: Group events by location
//   - Proximity alerts: Find users near a location
//
// Time Complexity:
//   - Insert: O(1)
//   - Query nearby: O(k) where k = entries in nearby cells (vs O(n) for linear scan)
//   - Remove: O(1)
type SpatialHashGrid struct {
	mu       sync.RWMutex
	cells    map[CellKey]*Cell        // Grid cells containing entries
	cellSize float64                  // Cell size in degrees (default: ~100km at equator)
	entries  map[string]*SpatialEntry // Index by ID for fast lookup/removal
}

// CellKey represents a grid cell coordinate.
type CellKey struct {
	X, Y int
}

// Cell contains all entries in a grid cell.
type Cell struct {
	entries []*SpatialEntry
}

// SpatialEntry represents an entry in the spatial grid.
type SpatialEntry struct {
	ID        string
	Lat       float64
	Lon       float64
	Timestamp time.Time
	Data      any
	cellKey   CellKey // Cached cell key for fast removal
}

// NewSpatialHashGrid creates a new spatial hash grid.
// cellSizeKm specifies the approximate cell size in kilometers.
// Smaller cells = more precise but more cells to check.
// Default: 100km which is good for impossible travel detection.
func NewSpatialHashGrid(cellSizeKm float64) *SpatialHashGrid {
	if cellSizeKm <= 0 {
		cellSizeKm = 100 // Default 100km
	}

	// Convert km to degrees (approximate: 1 degree ≈ 111km at equator)
	cellSizeDeg := cellSizeKm / 111.0

	return &SpatialHashGrid{
		cells:    make(map[CellKey]*Cell),
		cellSize: cellSizeDeg,
		entries:  make(map[string]*SpatialEntry),
	}
}

// getCellKey returns the cell key for a lat/lon coordinate.
func (g *SpatialHashGrid) getCellKey(lat, lon float64) CellKey {
	// Normalize longitude to [-180, 180]
	for lon > 180 {
		lon -= 360
	}
	for lon < -180 {
		lon += 360
	}

	// Calculate cell coordinates
	x := int(math.Floor(lon / g.cellSize))
	y := int(math.Floor(lat / g.cellSize))

	return CellKey{X: x, Y: y}
}

// Insert adds an entry to the grid.
// If an entry with the same ID exists, it's updated.
func (g *SpatialHashGrid) Insert(id string, lat, lon float64, timestamp time.Time, data any) {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Remove existing entry if present
	if existing, ok := g.entries[id]; ok {
		g.removeFromCellUnlocked(existing)
	}

	cellKey := g.getCellKey(lat, lon)

	entry := &SpatialEntry{
		ID:        id,
		Lat:       lat,
		Lon:       lon,
		Timestamp: timestamp,
		Data:      data,
		cellKey:   cellKey,
	}

	// Add to cell
	cell, exists := g.cells[cellKey]
	if !exists {
		cell = &Cell{entries: make([]*SpatialEntry, 0, 4)}
		g.cells[cellKey] = cell
	}
	cell.entries = append(cell.entries, entry)

	// Add to index
	g.entries[id] = entry
}

// Remove removes an entry by ID.
func (g *SpatialHashGrid) Remove(id string) bool {
	g.mu.Lock()
	defer g.mu.Unlock()

	entry, exists := g.entries[id]
	if !exists {
		return false
	}

	g.removeFromCellUnlocked(entry)
	delete(g.entries, id)
	return true
}

// removeFromCellUnlocked removes an entry from its cell (caller must hold lock).
func (g *SpatialHashGrid) removeFromCellUnlocked(entry *SpatialEntry) {
	cell, exists := g.cells[entry.cellKey]
	if !exists {
		return
	}

	// Find and remove entry from cell
	for i, e := range cell.entries {
		if e.ID == entry.ID {
			// Swap with last and truncate
			cell.entries[i] = cell.entries[len(cell.entries)-1]
			cell.entries = cell.entries[:len(cell.entries)-1]
			break
		}
	}

	// Remove empty cell
	if len(cell.entries) == 0 {
		delete(g.cells, entry.cellKey)
	}
}

// Get returns an entry by ID.
func (g *SpatialHashGrid) Get(id string) (*SpatialEntry, bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	entry, exists := g.entries[id]
	if !exists {
		return nil, false
	}

	// Return a copy to prevent modification
	entryCopy := *entry
	return &entryCopy, true
}

// QueryNearby returns all entries within a radius of the given point.
// radiusKm specifies the search radius in kilometers.
func (g *SpatialHashGrid) QueryNearby(lat, lon, radiusKm float64) []*SpatialEntry {
	g.mu.RLock()
	defer g.mu.RUnlock()

	// Calculate how many cells to check in each direction
	cellsToCheck := int(math.Ceil(radiusKm/111.0/g.cellSize)) + 1
	centerCell := g.getCellKey(lat, lon)

	var results []*SpatialEntry

	// Check all cells in the bounding box
	for dx := -cellsToCheck; dx <= cellsToCheck; dx++ {
		for dy := -cellsToCheck; dy <= cellsToCheck; dy++ {
			cellKey := CellKey{X: centerCell.X + dx, Y: centerCell.Y + dy}
			cell, exists := g.cells[cellKey]
			if !exists {
				continue
			}

			for _, entry := range cell.entries {
				// Calculate actual distance
				dist := haversineDistance(lat, lon, entry.Lat, entry.Lon)
				if dist <= radiusKm {
					entryCopy := *entry
					results = append(results, &entryCopy)
				}
			}
		}
	}

	return results
}

// QueryNearbyWithinTime returns entries within radius AND within a time window.
// This is useful for impossible travel detection.
func (g *SpatialHashGrid) QueryNearbyWithinTime(lat, lon, radiusKm float64, since time.Time) []*SpatialEntry {
	g.mu.RLock()
	defer g.mu.RUnlock()

	cellsToCheck := int(math.Ceil(radiusKm/111.0/g.cellSize)) + 1
	centerCell := g.getCellKey(lat, lon)

	var results []*SpatialEntry

	for dx := -cellsToCheck; dx <= cellsToCheck; dx++ {
		for dy := -cellsToCheck; dy <= cellsToCheck; dy++ {
			cellKey := CellKey{X: centerCell.X + dx, Y: centerCell.Y + dy}
			cell, exists := g.cells[cellKey]
			if !exists {
				continue
			}

			for _, entry := range cell.entries {
				// Check time first (fast)
				if entry.Timestamp.Before(since) {
					continue
				}

				// Then check distance (slower)
				dist := haversineDistance(lat, lon, entry.Lat, entry.Lon)
				if dist <= radiusKm {
					entryCopy := *entry
					results = append(results, &entryCopy)
				}
			}
		}
	}

	return results
}

// QueryCell returns all entries in the cell containing the given point.
func (g *SpatialHashGrid) QueryCell(lat, lon float64) []*SpatialEntry {
	g.mu.RLock()
	defer g.mu.RUnlock()

	cellKey := g.getCellKey(lat, lon)
	cell, exists := g.cells[cellKey]
	if !exists {
		return nil
	}

	results := make([]*SpatialEntry, len(cell.entries))
	for i, entry := range cell.entries {
		entryCopy := *entry
		results[i] = &entryCopy
	}
	return results
}

// Size returns the total number of entries.
func (g *SpatialHashGrid) Size() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return len(g.entries)
}

// NumCells returns the number of non-empty cells.
func (g *SpatialHashGrid) NumCells() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return len(g.cells)
}

// Clear removes all entries.
func (g *SpatialHashGrid) Clear() {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.cells = make(map[CellKey]*Cell)
	g.entries = make(map[string]*SpatialEntry)
}

// CleanupBefore removes all entries older than the given time.
// Returns the number of entries removed.
func (g *SpatialHashGrid) CleanupBefore(before time.Time) int {
	g.mu.Lock()
	defer g.mu.Unlock()

	removed := 0
	for id, entry := range g.entries {
		if entry.Timestamp.Before(before) {
			g.removeFromCellUnlocked(entry)
			delete(g.entries, id)
			removed++
		}
	}

	return removed
}

// haversineDistance calculates the distance between two lat/lon points in km.
// Uses the Haversine formula for accurate spherical distance.
func haversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadiusKm = 6371.0

	// Convert to radians
	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	deltaLat := (lat2 - lat1) * math.Pi / 180
	deltaLon := (lon2 - lon1) * math.Pi / 180

	// Haversine formula
	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(deltaLon/2)*math.Sin(deltaLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadiusKm * c
}

// UserLocationTracker wraps SpatialHashGrid for tracking user locations.
// It maintains the most recent location per user.
type UserLocationTracker struct {
	grid *SpatialHashGrid
	mu   sync.RWMutex
	// userLastLocation maps userID to their most recent entry ID
	userLastLocation map[string]string
}

// NewUserLocationTracker creates a tracker with the specified cell size.
func NewUserLocationTracker(cellSizeKm float64) *UserLocationTracker {
	return &UserLocationTracker{
		grid:             NewSpatialHashGrid(cellSizeKm),
		userLastLocation: make(map[string]string),
	}
}

// RecordLocation records a user's location.
// Returns the previous location if available, for travel velocity checks.
func (t *UserLocationTracker) RecordLocation(userID string, lat, lon float64, timestamp time.Time, data any) *SpatialEntry {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Generate unique entry ID
	entryID := userID + "-" + timestamp.Format(time.RFC3339Nano)

	// Get previous location before inserting new one
	var previousEntry *SpatialEntry
	if prevID, exists := t.userLastLocation[userID]; exists {
		prev, found := t.grid.Get(prevID)
		if found {
			previousEntry = prev
		}
	}

	// Insert new location
	t.grid.Insert(entryID, lat, lon, timestamp, data)
	t.userLastLocation[userID] = entryID

	return previousEntry
}

// GetLastLocation returns the most recent location for a user.
func (t *UserLocationTracker) GetLastLocation(userID string) (*SpatialEntry, bool) {
	t.mu.RLock()
	entryID, exists := t.userLastLocation[userID]
	t.mu.RUnlock()

	if !exists {
		return nil, false
	}

	return t.grid.Get(entryID)
}

// GetNearbyUsers returns users near a given location.
func (t *UserLocationTracker) GetNearbyUsers(lat, lon, radiusKm float64) []*SpatialEntry {
	return t.grid.QueryNearby(lat, lon, radiusKm)
}

// CleanupOldLocations removes locations older than the given duration.
func (t *UserLocationTracker) CleanupOldLocations(maxAge time.Duration) int {
	cutoff := time.Now().Add(-maxAge)
	return t.grid.CleanupBefore(cutoff)
}

// Clear removes all tracked locations.
func (t *UserLocationTracker) Clear() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.grid.Clear()
	t.userLastLocation = make(map[string]string)
}

// Size returns the total number of tracked locations.
func (t *UserLocationTracker) Size() int {
	return t.grid.Size()
}

// NumUsers returns the number of tracked users.
func (t *UserLocationTracker) NumUsers() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.userLastLocation)
}
