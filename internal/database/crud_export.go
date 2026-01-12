// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package database

import (
	"context"
	"fmt"

	"github.com/tomtom215/cartographus/internal/logging"
)

// ExportGeoParquet exports location data to GeoParquet format with optimized compression
// This addresses Medium Priority Issue M2 from the production audit
// GeoParquet provides 20% smaller files than GeoJSON and 10x faster export
// Requires spatial extension to be available
func (db *DB) ExportGeoParquet(ctx context.Context, outputPath string, filter LocationStatsFilter) error {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	// Verify spatial extension is available
	if !db.spatialAvailable {
		return fmt.Errorf("spatial extension not available - GeoParquet export requires spatial extension")
	}

	// Build filtered query
	query := `
		SELECT
			g.ip_address,
			g.latitude,
			g.longitude,
			g.geom,
			g.city,
			g.region,
			g.country,
			g.postal_code,
			g.timezone,
			COUNT(p.id) as playback_count
		FROM geolocations g
		LEFT JOIN playback_events p ON g.ip_address = p.ip_address
		WHERE 1=1`

	// Apply filters using extracted helper
	conditions, args := filter.buildFilterConditions()
	query += conditions

	query += `
		GROUP BY g.ip_address, g.latitude, g.longitude, g.geom, g.city, g.region, g.country, g.postal_code, g.timezone
		ORDER BY playback_count DESC`

	// Create temporary table with filtered data, sorted by spatial Hilbert curve for better compression
	// Hilbert curve ordering groups spatially close points together, improving compression ratio
	createTempQuery := fmt.Sprintf(`
		CREATE TEMPORARY TABLE IF NOT EXISTS temp_export_locations AS
		SELECT * FROM (%s) t
		ORDER BY ST_Hilbert(geom)`, query)

	if _, err := db.conn.ExecContext(ctx, createTempQuery, args...); err != nil {
		return fmt.Errorf("failed to create temporary export table: %w", err)
	}

	// Export to GeoParquet with ZSTD compression (better compression than GZIP for spatial data)
	exportQuery := `
		COPY temp_export_locations TO ? (
			FORMAT PARQUET,
			COMPRESSION 'ZSTD',
			ROW_GROUP_SIZE 100000
		)`

	if _, err := db.conn.ExecContext(ctx, exportQuery, outputPath); err != nil {
		return fmt.Errorf("failed to export GeoParquet: %w", err)
	}

	// Clean up temporary table
	if _, err := db.conn.ExecContext(ctx, "DROP TABLE IF EXISTS temp_export_locations"); err != nil {
		// Non-fatal error, just log
		logging.Warn().Err(err).Msg("Failed to drop temporary export table")
	}

	return nil
}

// ExportGeoJSON exports location data to GeoJSON format
// Provided as an alternative to GeoParquet for broader compatibility
func (db *DB) ExportGeoJSON(ctx context.Context, outputPath string, filter LocationStatsFilter) error {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	// Build filtered query
	query := `
		SELECT
			g.ip_address,
			g.latitude,
			g.longitude,
			g.city,
			g.region,
			g.country,
			g.postal_code,
			g.timezone,
			COUNT(p.id) as playback_count
		FROM geolocations g
		LEFT JOIN playback_events p ON g.ip_address = p.ip_address
		WHERE 1=1`

	// Apply filters using extracted helper
	conditions, args := filter.buildFilterConditions()
	query += conditions

	query += `
		GROUP BY g.ip_address, g.latitude, g.longitude, g.city, g.region, g.country, g.postal_code, g.timezone
		ORDER BY playback_count DESC`

	// Use DuckDB's built-in GeoJSON export if spatial extension is available
	if db.spatialAvailable {
		// Create GeoJSON with geometry column
		geoJSONQuery := fmt.Sprintf(`
			COPY (
				SELECT
					ST_AsGeoJSON(ST_MakePoint(longitude, latitude)) as geometry,
					ip_address,
					latitude,
					longitude,
					city,
					region,
					country,
					postal_code,
					timezone,
					playback_count
				FROM (%s) t
			) TO ? (FORMAT JSON, ARRAY true)`, query)

		// Append outputPath to args
		geoJSONArgs := append(args, outputPath)
		if _, err := db.conn.ExecContext(ctx, geoJSONQuery, geoJSONArgs...); err != nil {
			return fmt.Errorf("failed to export GeoJSON: %w", err)
		}
	} else {
		// Fallback: Export as plain JSON without geometry
		exportQuery := fmt.Sprintf(`
			COPY (%s) TO ? (FORMAT JSON, ARRAY true)`, query)

		// Append outputPath to args
		jsonArgs := append(args, outputPath)
		if _, err := db.conn.ExecContext(ctx, exportQuery, jsonArgs...); err != nil {
			return fmt.Errorf("failed to export JSON: %w", err)
		}
	}

	return nil
}
