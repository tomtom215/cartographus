// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package database

import (
	"context"
	"fmt"
	"time"

	"github.com/tomtom215/cartographus/internal/logging"
)

// initializeSpatialOptimizations adds H3 indexing, spatial indexes, and distance calculations
func (db *DB) initializeSpatialOptimizations(serverLat, serverLon float64) error {
	if !db.spatialAvailable {
		return nil // Skip if spatial extension not available
	}

	// Migration queries for spatial optimization columns
	spatialMigrations := []string{
		// H3 hexagon indexes at multiple resolutions for hierarchical aggregation
		// Resolution 6: ~36 km² hexagons (good for country-level visualization)
		`ALTER TABLE geolocations ADD COLUMN IF NOT EXISTS h3_index_6 UBIGINT;`,
		// Resolution 7: ~5.2 km² hexagons (good for city-level visualization)
		`ALTER TABLE geolocations ADD COLUMN IF NOT EXISTS h3_index_7 UBIGINT;`,
		// Resolution 8: ~0.74 km² hexagons (good for neighborhood-level visualization)
		`ALTER TABLE geolocations ADD COLUMN IF NOT EXISTS h3_index_8 UBIGINT;`,

		// Geodesic distance from server location (in kilometers)
		// Enables distance-based arc weighting and filtering
		`ALTER TABLE geolocations ADD COLUMN IF NOT EXISTS distance_from_server DOUBLE;`,

		// Pre-computed bounding box for faster spatial queries
		`ALTER TABLE geolocations ADD COLUMN IF NOT EXISTS bbox_xmin DOUBLE;`,
		`ALTER TABLE geolocations ADD COLUMN IF NOT EXISTS bbox_ymin DOUBLE;`,
		`ALTER TABLE geolocations ADD COLUMN IF NOT EXISTS bbox_xmax DOUBLE;`,
		`ALTER TABLE geolocations ADD COLUMN IF NOT EXISTS bbox_ymax DOUBLE;`,
	}

	for _, query := range spatialMigrations {
		if _, err := db.conn.Exec(query); err != nil {
			return fmt.Errorf("failed to execute spatial migration: %s: %w", query, err)
		}
	}

	// Create spatial indexes for fast geospatial queries
	// R-tree spatial index on geometry column (100x faster spatial queries)
	spatialIndexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_geolocation_spatial ON geolocations USING RTREE (geom);`,

		// Indexes on H3 columns for fast hexagon aggregation
		`CREATE INDEX IF NOT EXISTS idx_geolocation_h3_6 ON geolocations(h3_index_6);`,
		`CREATE INDEX IF NOT EXISTS idx_geolocation_h3_7 ON geolocations(h3_index_7);`,
		`CREATE INDEX IF NOT EXISTS idx_geolocation_h3_8 ON geolocations(h3_index_8);`,

		// Index on distance for distance-based filtering
		`CREATE INDEX IF NOT EXISTS idx_geolocation_distance ON geolocations(distance_from_server);`,

		// Bounding box indexes for viewport queries
		`CREATE INDEX IF NOT EXISTS idx_geolocation_bbox ON geolocations(bbox_xmin, bbox_ymin, bbox_xmax, bbox_ymax);`,
	}

	for _, query := range spatialIndexes {
		if _, err := db.conn.Exec(query); err != nil {
			// R-tree index may fail if geom column doesn't exist (test mode)
			// Continue with other indexes
			logging.Warn().Err(err).Msg("Failed to create spatial index (may not be supported)")
		}
	}

	// Populate H3 indexes and distance calculations for existing geolocations
	// This is idempotent - only updates NULL values
	if serverLat != 0.0 || serverLon != 0.0 {
		updateQuery := fmt.Sprintf(`
		UPDATE geolocations
		SET
			h3_index_6 = h3_latlng_to_cell(latitude, longitude, 6),
			h3_index_7 = h3_latlng_to_cell(latitude, longitude, 7),
			h3_index_8 = h3_latlng_to_cell(latitude, longitude, 8),
			distance_from_server = ST_Distance_Sphere(
				geom,
				ST_Point(%f, %f)
			) / 1000.0,  -- Convert meters to kilometers
			bbox_xmin = longitude - 0.01,
			bbox_ymin = latitude - 0.01,
			bbox_xmax = longitude + 0.01,
			bbox_ymax = latitude + 0.01
		WHERE h3_index_6 IS NULL OR distance_from_server IS NULL;
		`, serverLon, serverLat)

		if _, err := db.conn.Exec(updateQuery); err != nil {
			// This is not fatal - it means h3 functions aren't available
			logging.Warn().Err(err).Msg("Failed to populate H3 indexes (H3 extension may not be available)")
		}
	}

	return nil
}

// UpdateGeolocationSpatialData updates H3 indexes and distance for a specific geolocation
// Called after inserting/updating a geolocation record
func (db *DB) UpdateGeolocationSpatialData(ipAddress string, serverLat, serverLon float64) error {
	// Use a default timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return db.UpdateGeolocationSpatialDataCtx(ctx, ipAddress, serverLat, serverLon)
}

// UpdateGeolocationSpatialDataCtx updates H3 indexes and distance for a specific geolocation with context
// Called after inserting/updating a geolocation record
func (db *DB) UpdateGeolocationSpatialDataCtx(ctx context.Context, ipAddress string, serverLat, serverLon float64) error {
	if !db.spatialAvailable {
		return nil
	}

	if serverLat == 0.0 && serverLon == 0.0 {
		// No server location configured, skip distance calculation
		query := `
		UPDATE geolocations
		SET
			h3_index_6 = h3_latlng_to_cell(latitude, longitude, 6),
			h3_index_7 = h3_latlng_to_cell(latitude, longitude, 7),
			h3_index_8 = h3_latlng_to_cell(latitude, longitude, 8),
			bbox_xmin = longitude - 0.01,
			bbox_ymin = latitude - 0.01,
			bbox_xmax = longitude + 0.01,
			bbox_ymax = latitude + 0.01
		WHERE ip_address = ?;
		`
		_, err := db.conn.ExecContext(ctx, query, ipAddress)
		return err
	}

	query := fmt.Sprintf(`
	UPDATE geolocations
	SET
		h3_index_6 = h3_latlng_to_cell(latitude, longitude, 6),
		h3_index_7 = h3_latlng_to_cell(latitude, longitude, 7),
		h3_index_8 = h3_latlng_to_cell(latitude, longitude, 8),
		distance_from_server = ST_Distance_Sphere(
			geom,
			ST_Point(%f, %f)
		) / 1000.0,
		bbox_xmin = longitude - 0.01,
		bbox_ymin = latitude - 0.01,
		bbox_xmax = longitude + 0.01,
		bbox_ymax = latitude + 0.01
	WHERE ip_address = ?;
	`, serverLon, serverLat)

	_, err := db.conn.ExecContext(ctx, query, ipAddress)
	return err
}
