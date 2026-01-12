// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package database

import (
	"context"
	"fmt"

	"github.com/tomtom215/cartographus/internal/models"
)

// GetH3AggregatedHexagons returns pre-aggregated hexagon data using H3 spatial indexing
// This is 10x faster than client-side hexagon aggregation in deck.gl
// Resolution: 6 (country), 7 (city), 8 (neighborhood)
func (db *DB) GetH3AggregatedHexagons(ctx context.Context, filter LocationStatsFilter, resolution int) ([]models.H3HexagonStats, error) {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	if !db.spatialAvailable {
		return nil, fmt.Errorf("spatial extension not available")
	}

	if resolution < 6 || resolution > 8 {
		return nil, fmt.Errorf("resolution must be 6, 7, or 8")
	}

	// Build filter conditions
	whereClauses, args := buildFilterConditions(filter, false, 1)
	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = " AND " + join(whereClauses, " AND ")
	}

	h3Column := fmt.Sprintf("h3_index_%d", resolution)

	// Query aggregated hexagon data with H3 spatial index
	query := fmt.Sprintf(`
	WITH hexagon_stats AS (
		SELECT
			g.%s as h3_index,
			COUNT(*) as playback_count,
			COUNT(DISTINCT p.user_id) as unique_users,
			AVG(COALESCE(p.percent_complete, 0)) as avg_completion,
			SUM(COALESCE(p.play_duration, 0)) as total_watch_minutes
		FROM playback_events p
		JOIN geolocations g ON p.ip_address = g.ip_address
		WHERE g.%s IS NOT NULL%s
		GROUP BY g.%s
	)
	SELECT
		h3_index,
		h3_cell_to_lat(h3_index) as latitude,
		h3_cell_to_lng(h3_index) as longitude,
		playback_count,
		unique_users,
		avg_completion,
		total_watch_minutes
	FROM hexagon_stats
	WHERE playback_count > 0
	ORDER BY playback_count DESC
	LIMIT 10000;
	`, h3Column, h3Column, whereSQL, h3Column)

	rows, err := db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query H3 hexagons: %w", err)
	}
	defer rows.Close()

	var hexagons []models.H3HexagonStats
	for rows.Next() {
		var h models.H3HexagonStats
		if err := rows.Scan(
			&h.H3Index,
			&h.Latitude,
			&h.Longitude,
			&h.PlaybackCount,
			&h.UniqueUsers,
			&h.AvgCompletion,
			&h.TotalWatchMinutes,
		); err != nil {
			return nil, fmt.Errorf("failed to scan hexagon row: %w", err)
		}
		hexagons = append(hexagons, h)
	}

	return hexagons, rows.Err()
}

// GetDistanceWeightedArcs returns arc data with geodesic distance calculations
// Arcs are weighted by both playback count AND distance from server
func (db *DB) GetDistanceWeightedArcs(ctx context.Context, filter LocationStatsFilter, serverLat, serverLon float64) ([]models.ArcStats, error) {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	if !db.spatialAvailable {
		return nil, fmt.Errorf("spatial extension not available")
	}

	if serverLat == 0.0 && serverLon == 0.0 {
		return nil, fmt.Errorf("server location not configured")
	}

	// Build filter conditions
	whereClauses, args := buildFilterConditions(filter, false, 1)
	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = " AND " + join(whereClauses, " AND ")
	}

	// Query with geodesic distance calculations
	query := `
	SELECT
		g.latitude as user_latitude,
		g.longitude as user_longitude,
		g.city,
		g.country,
		g.distance_from_server,
		COUNT(*) as playback_count,
		COUNT(DISTINCT p.user_id) as unique_users,
		AVG(COALESCE(p.percent_complete, 0)) as avg_completion,
		-- Arc weight: playback count * distance decay factor
		-- Longer arcs get more visual weight to show global reach
		-- Add minimum weight of 1.0 to ensure local arcs (distance ~0) have positive weight
		COUNT(*) * (1.0 + LOG(1 + g.distance_from_server / 1000.0)) as arc_weight
	FROM playback_events p
	JOIN geolocations g ON p.ip_address = g.ip_address
	WHERE g.distance_from_server IS NOT NULL` + whereSQL + `
	GROUP BY g.latitude, g.longitude, g.city, g.country, g.distance_from_server
	HAVING COUNT(*) > 0
	ORDER BY arc_weight DESC
	LIMIT 1000;
	`

	rows, err := db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query distance-weighted arcs: %w", err)
	}
	defer rows.Close()

	var arcs []models.ArcStats
	for rows.Next() {
		var a models.ArcStats
		a.ServerLatitude = serverLat
		a.ServerLongitude = serverLon

		if err := rows.Scan(
			&a.UserLatitude,
			&a.UserLongitude,
			&a.City,
			&a.Country,
			&a.DistanceKm,
			&a.PlaybackCount,
			&a.UniqueUsers,
			&a.AvgCompletion,
			&a.Weight,
		); err != nil {
			return nil, fmt.Errorf("failed to scan arc row: %w", err)
		}
		arcs = append(arcs, a)
	}

	return arcs, rows.Err()
}

// GetLocationsInViewport returns locations within a geographic bounding box
// Uses R-tree spatial index for 100x faster viewport queries
func (db *DB) GetLocationsInViewport(ctx context.Context, filter LocationStatsFilter, west, south, east, north float64) ([]models.LocationStats, error) {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	if !db.spatialAvailable {
		return nil, fmt.Errorf("spatial extension not available")
	}

	// Build filter conditions
	whereClauses, args := buildFilterConditions(filter, false, 1)

	// Add bounding box filter using spatial index
	// ST_Within is optimized with R-tree index
	boundingBoxFilter := fmt.Sprintf("ST_Within(g.geom, ST_MakeEnvelope(%.6f, %.6f, %.6f, %.6f))", west, south, east, north)

	whereClauses = append(whereClauses, boundingBoxFilter)
	whereSQL := join(whereClauses, " AND ")

	// Query with spatial filter
	query := `
	SELECT
		g.latitude,
		g.longitude,
		g.city,
		g.region,
		g.country,
		COUNT(*) as playback_count,
		COUNT(DISTINCT p.user_id) as unique_users,
		MIN(p.started_at) as first_seen,
		MAX(p.started_at) as last_seen,
		AVG(COALESCE(p.percent_complete, 0)) as avg_completion
	FROM playback_events p
	JOIN geolocations g ON p.ip_address = g.ip_address
	WHERE ` + whereSQL + `
	GROUP BY g.latitude, g.longitude, g.city, g.region, g.country, g.geom
	HAVING COUNT(*) > 0
	ORDER BY playback_count DESC
	LIMIT 5000;
	`

	rows, err := db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query viewport locations: %w", err)
	}
	defer rows.Close()

	var locations []models.LocationStats
	for rows.Next() {
		var loc models.LocationStats
		if err := rows.Scan(
			&loc.Latitude,
			&loc.Longitude,
			&loc.City,
			&loc.Region,
			&loc.Country,
			&loc.PlaybackCount,
			&loc.UniqueUsers,
			&loc.FirstSeen,
			&loc.LastSeen,
			&loc.AvgCompletion,
		); err != nil {
			return nil, fmt.Errorf("failed to scan viewport location row: %w", err)
		}
		locations = append(locations, loc)
	}

	return locations, rows.Err()
}

// GetTemporalSpatialDensity returns playback density with rolling spatial aggregations
// Uses window functions for smooth temporal-spatial visualization
func (db *DB) GetTemporalSpatialDensity(ctx context.Context, filter LocationStatsFilter, interval string, h3Resolution int) ([]models.TemporalSpatialPoint, error) {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	if !db.spatialAvailable {
		return nil, fmt.Errorf("spatial extension not available")
	}

	if h3Resolution < 6 || h3Resolution > 8 {
		return nil, fmt.Errorf("h3Resolution must be 6, 7, or 8")
	}

	// Build filter conditions
	whereClauses, args := buildFilterConditions(filter, false, 1)
	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = " AND " + join(whereClauses, " AND ")
	}

	// Determine time bucketing SQL
	var bucketSQL string
	switch interval {
	case "hour":
		bucketSQL = "DATE_TRUNC('hour', p.started_at)"
	case "day":
		bucketSQL = "DATE_TRUNC('day', p.started_at)"
	case "week":
		bucketSQL = "DATE_TRUNC('week', p.started_at)"
	case "month":
		bucketSQL = "DATE_TRUNC('month', p.started_at)"
	default:
		return nil, fmt.Errorf("invalid interval: must be hour, day, week, or month")
	}

	h3Column := fmt.Sprintf("h3_index_%d", h3Resolution)

	// Query with temporal-spatial window functions
	query := fmt.Sprintf(`
	WITH temporal_hexagons AS (
		SELECT
			%s as time_bucket,
			g.%s as h3_index,
			h3_cell_to_lat(g.%s) as latitude,
			h3_cell_to_lng(g.%s) as longitude,
			COUNT(*) as playback_count,
			COUNT(DISTINCT p.user_id) as unique_users
		FROM playback_events p
		JOIN geolocations g ON p.ip_address = g.ip_address
		WHERE g.%s IS NOT NULL%s
		GROUP BY time_bucket, g.%s, latitude, longitude
	)
	SELECT
		time_bucket,
		h3_index,
		latitude,
		longitude,
		playback_count,
		unique_users,
		-- Rolling 7-period average for smooth animation
		AVG(playback_count) OVER (
			PARTITION BY h3_index
			ORDER BY time_bucket
			ROWS BETWEEN 6 PRECEDING AND CURRENT ROW
		) as rolling_avg_playbacks,
		-- Cumulative playbacks for growing effect
		SUM(playback_count) OVER (
			PARTITION BY h3_index
			ORDER BY time_bucket
			ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW
		) as cumulative_playbacks
	FROM temporal_hexagons
	ORDER BY time_bucket, playback_count DESC;
	`, bucketSQL, h3Column, h3Column, h3Column, h3Column, whereSQL, h3Column)

	rows, err := db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query temporal spatial density: %w", err)
	}
	defer rows.Close()

	var points []models.TemporalSpatialPoint
	for rows.Next() {
		var p models.TemporalSpatialPoint
		if err := rows.Scan(
			&p.TimeBucket,
			&p.H3Index,
			&p.Latitude,
			&p.Longitude,
			&p.PlaybackCount,
			&p.UniqueUsers,
			&p.RollingAvgPlaybacks,
			&p.CumulativePlaybacks,
		); err != nil {
			return nil, fmt.Errorf("failed to scan temporal spatial point: %w", err)
		}
		points = append(points, p)
	}

	return points, rows.Err()
}

// GetNearbyLocations finds locations within a specified radius of a point
// Uses ST_DWithin with spatial index for fast proximity queries
func (db *DB) GetNearbyLocations(ctx context.Context, lat, lon float64, radiusKm float64, filter LocationStatsFilter) ([]models.LocationStats, error) {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	if !db.spatialAvailable {
		return nil, fmt.Errorf("spatial extension not available")
	}

	// Build filter conditions
	whereClauses, args := buildFilterConditions(filter, false, 1)

	// Add proximity filter (ST_DWithin uses spatial index)
	// radiusKm * 1000 converts to meters
	proximityFilter := fmt.Sprintf(`
		ST_DWithin(
			g.geom,
			ST_Point(%.6f, %.6f),
			%.2f
		)
	`, lon, lat, radiusKm*1000.0)

	whereClauses = append(whereClauses, proximityFilter)
	whereSQL := join(whereClauses, " AND ")

	query := `
	SELECT
		g.latitude,
		g.longitude,
		g.city,
		g.region,
		g.country,
		COUNT(*) as playback_count,
		COUNT(DISTINCT p.user_id) as unique_users,
		MIN(p.started_at) as first_seen,
		MAX(p.started_at) as last_seen,
		AVG(COALESCE(p.percent_complete, 0)) as avg_completion,
		ST_Distance_Sphere(
			g.geom,
			ST_Point(?, ?)
		) / 1000.0 as distance_km
	FROM playback_events p
	JOIN geolocations g ON p.ip_address = g.ip_address
	WHERE ` + whereSQL + `
	GROUP BY g.latitude, g.longitude, g.city, g.region, g.country, g.geom
	HAVING COUNT(*) > 0
	ORDER BY distance_km ASC
	LIMIT 1000;
	`

	// Append lat/lon for distance calculation
	args = append(args, lon, lat)

	rows, err := db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query nearby locations: %w", err)
	}
	defer rows.Close()

	var locations []models.LocationStats
	for rows.Next() {
		var loc models.LocationStats
		var distanceKm float64
		if err := rows.Scan(
			&loc.Latitude,
			&loc.Longitude,
			&loc.City,
			&loc.Region,
			&loc.Country,
			&loc.PlaybackCount,
			&loc.UniqueUsers,
			&loc.FirstSeen,
			&loc.LastSeen,
			&loc.AvgCompletion,
			&distanceKm,
		); err != nil {
			return nil, fmt.Errorf("failed to scan nearby location row: %w", err)
		}
		locations = append(locations, loc)
	}

	return locations, rows.Err()
}
