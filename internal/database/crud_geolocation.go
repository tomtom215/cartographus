// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/tomtom215/cartographus/internal/logging"
	"github.com/tomtom215/cartographus/internal/models"
)

// UpsertGeolocationWithServer inserts or updates a geolocation record with server location
// Automatically populates H3 spatial indexes and distance calculations
// Uses per-IP locking to prevent DuckDB INTERNAL errors while allowing concurrent writes to different IPs
// Implements retry logic for transaction conflicts with exponential backoff
func (db *DB) UpsertGeolocationWithServer(geo *models.Geolocation, serverLat, serverLon float64) error {
	// Acquire per-IP lock to prevent concurrent UPSERTs on same IP (prevents INTERNAL errors)
	mu := db.acquireIPLock(geo.IPAddress)
	defer db.releaseIPLock(geo.IPAddress, mu)

	if geo.LastUpdated.IsZero() {
		geo.LastUpdated = time.Now()
	}

	// Create context with timeout to prevent indefinite hangs
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Retry logic for transaction conflicts
	const maxRetries = 3
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		err := db.doUpsertGeolocation(ctx, geo, serverLat, serverLon)
		if err == nil {
			// MEDIUM-1: Increment data version to invalidate tile cache
			db.IncrementDataVersion()
			return nil // Success
		}

		lastErr = err

		// Check for context timeout/cancellation
		if ctx.Err() != nil {
			return fmt.Errorf("operation timed out or canceled: %w", ctx.Err())
		}

		// Check error type
		if isInternalError(err) {
			// INTERNAL errors are fatal bugs - don't retry, fail immediately
			return fmt.Errorf("FATAL: DuckDB internal error (this should not happen with per-IP locking): %w", err)
		}

		if isTransactionConflict(err) {
			// Expected transaction conflict - retry with exponential backoff
			if attempt < maxRetries-1 {
				backoff := time.Millisecond * time.Duration(1<<uint(attempt)) // 1ms, 2ms, 4ms
				// Use cancellable wait instead of time.Sleep
				select {
				case <-time.After(backoff):
					continue
				case <-ctx.Done():
					return ctx.Err()
				}
			}
		}

		// Other errors (network, database closed, etc.) - don't retry
		return err
	}

	return fmt.Errorf("max retries exceeded: %w", lastErr)
}

// doUpsertGeolocation performs the actual upsert operation (internal helper)
func (db *DB) doUpsertGeolocation(ctx context.Context, geo *models.Geolocation, serverLat, serverLon float64) error {
	var query string
	var args []interface{}

	if db.spatialAvailable {
		// With spatial extension: include geom column with ST_Point
		query = `INSERT INTO geolocations (
			ip_address, latitude, longitude, geom, city, region, country,
			postal_code, timezone, accuracy_radius, last_updated
		) VALUES (?, ?, ?, ST_Point(?, ?), ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (ip_address) DO UPDATE SET
			latitude = EXCLUDED.latitude,
			longitude = EXCLUDED.longitude,
			geom = EXCLUDED.geom,
			city = EXCLUDED.city,
			region = EXCLUDED.region,
			country = EXCLUDED.country,
			postal_code = EXCLUDED.postal_code,
			timezone = EXCLUDED.timezone,
			accuracy_radius = EXCLUDED.accuracy_radius,
			last_updated = EXCLUDED.last_updated`

		args = []interface{}{
			geo.IPAddress, geo.Latitude, geo.Longitude, geo.Longitude, geo.Latitude,
			geo.City, geo.Region, geo.Country, geo.PostalCode, geo.Timezone,
			geo.AccuracyRadius, geo.LastUpdated,
		}
	} else {
		// Without spatial extension: omit geom column
		query = `INSERT INTO geolocations (
			ip_address, latitude, longitude, city, region, country,
			postal_code, timezone, accuracy_radius, last_updated
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (ip_address) DO UPDATE SET
			latitude = EXCLUDED.latitude,
			longitude = EXCLUDED.longitude,
			city = EXCLUDED.city,
			region = EXCLUDED.region,
			country = EXCLUDED.country,
			postal_code = EXCLUDED.postal_code,
			timezone = EXCLUDED.timezone,
			accuracy_radius = EXCLUDED.accuracy_radius,
			last_updated = EXCLUDED.last_updated`

		args = []interface{}{
			geo.IPAddress, geo.Latitude, geo.Longitude,
			geo.City, geo.Region, geo.Country, geo.PostalCode, geo.Timezone,
			geo.AccuracyRadius, geo.LastUpdated,
		}
	}

	_, err := db.conn.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to upsert geolocation: %w", err)
	}

	// Update spatial optimizations (H3 indexes, distance from server)
	// Only attempt if spatial extension is available
	if db.spatialAvailable {
		if err := db.UpdateGeolocationSpatialDataCtx(ctx, geo.IPAddress, serverLat, serverLon); err != nil {
			// Log warning but don't fail the operation
			logging.Warn().Str("ip_address", geo.IPAddress).Err(err).Msg("Failed to update spatial data")
		}
	}

	return nil
}

// GetGeolocations retrieves multiple geolocation records in a single query
// Returns a map of IP address -> geolocation for efficient lookups
// MEDIUM-2: Batch geolocation lookups for 10-20x performance improvement
func (db *DB) GetGeolocations(ctx context.Context, ipAddresses []string) (map[string]*models.Geolocation, error) {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	if len(ipAddresses) == 0 {
		return make(map[string]*models.Geolocation), nil
	}

	// Build parameterized IN clause using helper
	placeholders, args := buildInClause(ipAddresses)

	query := fmt.Sprintf(`
		SELECT ip_address, latitude, longitude, city, region, country,
		       postal_code, timezone, accuracy_radius, last_updated
		FROM geolocations
		WHERE ip_address IN (%s)
	`, placeholders)

	rows, err := db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query geolocations: %w", err)
	}
	defer rows.Close()

	result := make(map[string]*models.Geolocation, len(ipAddresses))
	for rows.Next() {
		geo := &models.Geolocation{}
		err := rows.Scan(
			&geo.IPAddress, &geo.Latitude, &geo.Longitude,
			&geo.City, &geo.Region, &geo.Country,
			&geo.PostalCode, &geo.Timezone,
			&geo.AccuracyRadius, &geo.LastUpdated,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan geolocation: %w", err)
		}
		result[geo.IPAddress] = geo
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating geolocations: %w", err)
	}

	return result, nil
}

// GetGeolocation retrieves geolocation data for a single IP address.
//
// This method performs a simple lookup by IP address primary key. It's used during
// sync operations to check if an IP address has already been geolocated, avoiding
// unnecessary external API calls to the geolocation service.
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - ipAddress: IP address to lookup (IPv4 or IPv6)
//
// Returns:
//   - Pointer to Geolocation struct if found
//   - nil if IP address not found in database (no error)
//   - error only if query execution fails
//
// Performance: ~0.5-1ms with primary key index on ip_address.
//
// For batch lookups of multiple IPs, use GetGeolocations() instead for 10-20x
// better performance (single query vs N queries).
//
// Example:
//
//	geo, err := db.GetGeolocation(ctx, "1.2.3.4")
//	if err != nil {
//	    return err
//	}
//	if geo == nil {
//	    // IP not geolocated yet, need to call external API
//	}
func (db *DB) GetGeolocation(ctx context.Context, ipAddress string) (*models.Geolocation, error) {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	query := `
	SELECT ip_address, latitude, longitude, city, region, country,
		postal_code, timezone, accuracy_radius, last_updated
	FROM geolocations
	WHERE ip_address = ?`

	var geo models.Geolocation
	err := db.conn.QueryRowContext(ctx, query, ipAddress).Scan(
		&geo.IPAddress, &geo.Latitude, &geo.Longitude, &geo.City, &geo.Region,
		&geo.Country, &geo.PostalCode, &geo.Timezone, &geo.AccuracyRadius,
		&geo.LastUpdated,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get geolocation: %w", err)
	}

	return &geo, nil
}

// UpsertGeolocation is a backward-compatible wrapper for tests
// Use UpsertGeolocationWithServer in production code
func (db *DB) UpsertGeolocation(geo *models.Geolocation) error {
	return db.UpsertGeolocationWithServer(geo, 0.0, 0.0)
}
