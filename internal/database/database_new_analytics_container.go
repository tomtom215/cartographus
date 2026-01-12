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

	"github.com/tomtom215/cartographus/internal/models"
)

// GetContainerAnalytics analyzes media container formats including distribution, direct play rates,
// remuxing events, and platform compatibility for infrastructure optimization.
func (db *DB) GetContainerAnalytics(ctx context.Context, filter LocationStatsFilter) (*models.ContainerAnalytics, error) {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	whereClauses, args := buildFilterConditions(filter, false, 1)
	whereClause := buildWhereClause(whereClauses)

	// Get total playbacks
	total, err := db.getTotalPlaybacks(ctx, whereClause, args)
	if err != nil {
		return nil, errorContext("get total playbacks", err)
	}

	// Get format distribution with direct play rates
	formatDist, directPlayRates, err := db.getContainerFormatDistribution(ctx, whereClause, args, total)
	if err != nil {
		return nil, errorContext("get format distribution", err)
	}

	// Get remuxing events (container changes)
	remuxEvents, err := db.getContainerRemuxEvents(ctx, whereClause, args)
	if err != nil {
		return nil, errorContext("get remux events", err)
	}

	// Get platform compatibility
	platformCompat, err := db.getContainerPlatformCompatibility(ctx, whereClause, args)
	if err != nil {
		return nil, errorContext("get platform compatibility", err)
	}

	return &models.ContainerAnalytics{
		TotalPlaybacks:        total,
		FormatDistribution:    formatDist,
		DirectPlayRates:       directPlayRates,
		RemuxEvents:           remuxEvents,
		PlatformCompatibility: platformCompat,
	}, nil
}

// getContainerFormatDistribution queries media container format distribution with playback counts,
// percentages, and direct play rates for each container type.
func (db *DB) getContainerFormatDistribution(ctx context.Context, whereClause string, args []interface{}, total int) ([]models.ContainerDistribution, map[string]float64, error) {
	query := fmt.Sprintf(`
		SELECT
			COALESCE(LOWER(container), 'mp4') as container,
			COUNT(*) as playback_count,
			(COUNT(*) * 100.0 / %d) as percentage,
			(SUM(CASE WHEN transcode_decision = 'direct play' THEN 1 ELSE 0 END) * 100.0 / COUNT(*)) as direct_play_rate
		FROM playback_events
		%s
		GROUP BY container
		ORDER BY playback_count DESC
	`, total, whereClause)

	rows, err := db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to query container distribution: %w", err)
	}
	defer rows.Close()

	var formatDist []models.ContainerDistribution
	directPlayRates := make(map[string]float64)
	for rows.Next() {
		var d models.ContainerDistribution
		if err := rows.Scan(&d.Container, &d.PlaybackCount, &d.Percentage, &d.DirectPlayRate); err != nil {
			return nil, nil, fmt.Errorf("failed to scan container row: %w", err)
		}
		formatDist = append(formatDist, d)
		directPlayRates[d.Container] = d.DirectPlayRate
	}

	return formatDist, directPlayRates, rows.Err()
}

// getContainerRemuxEvents queries top 10 container remuxing events (source→stream container changes)
// with occurrence counts and affected platforms for transcoding optimization insights.
func (db *DB) getContainerRemuxEvents(ctx context.Context, whereClause string, args []interface{}) ([]models.ContainerRemux, error) {
	remuxCondition := "container IS NOT NULL AND stream_container IS NOT NULL AND container != stream_container"
	remuxWhere := appendWhereCondition(whereClause, remuxCondition)

	query := fmt.Sprintf(`
		SELECT
			container as source_container,
			stream_container,
			COUNT(*) as occurrence_count,
			string_agg(DISTINCT platform, ',') as platforms
		FROM playback_events
		%s
		GROUP BY container, stream_container
		ORDER BY occurrence_count DESC
		LIMIT 10
	`, remuxWhere)

	rows, err := db.conn.QueryContext(ctx, query, args...)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("failed to query remux events: %w", err)
	}

	var results []models.ContainerRemux
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var r models.ContainerRemux
			var platformsStr string
			if err := rows.Scan(&r.SourceContainer, &r.StreamContainer, &r.OccurrenceCount, &platformsStr); err != nil {
				return nil, fmt.Errorf("failed to scan remux row: %w", err)
			}
			r.Platforms = parseAggregatedList(platformsStr)
			results = append(results, r)
		}

		if err = rows.Err(); err != nil {
			return nil, fmt.Errorf("error iterating remux rows: %w", err)
		}
	}

	return results, nil
}

// getContainerPlatformCompatibility queries top 15 platforms with their supported container formats
// and direct play rates for client capability analysis.
func (db *DB) getContainerPlatformCompatibility(ctx context.Context, whereClause string, args []interface{}) ([]models.PlatformContainer, error) {
	platformCondition := "container IS NOT NULL"
	platformWhere := appendWhereCondition(whereClause, platformCondition)

	query := fmt.Sprintf(`
		SELECT
			platform,
			string_agg(DISTINCT container, ',') as containers,
			(SUM(CASE WHEN transcode_decision = 'direct play' THEN 1 ELSE 0 END) * 100.0 / COUNT(*)) as direct_play_rate
		FROM playback_events
		%s
		GROUP BY platform
		ORDER BY direct_play_rate DESC
		LIMIT 15
	`, platformWhere)

	rows, err := db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query platform compatibility: %w", err)
	}
	defer rows.Close()

	var results []models.PlatformContainer
	for rows.Next() {
		var p models.PlatformContainer
		var containersStr string
		if err := rows.Scan(&p.Platform, &containersStr, &p.DirectPlayRate); err != nil {
			return nil, fmt.Errorf("failed to scan platform row: %w", err)
		}
		p.SupportedFormats = parseAggregatedList(containersStr)
		results = append(results, p)
	}

	return results, rows.Err()
}
