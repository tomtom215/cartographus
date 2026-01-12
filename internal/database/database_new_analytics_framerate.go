// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package database

import (
	"context"
	"fmt"
	"strings"

	"github.com/tomtom215/cartographus/internal/models"
)

// GetFrameRateAnalytics returns comprehensive frame rate analytics including distribution,
// high FPS adoption rate, and breakdown by media type.
func (db *DB) GetFrameRateAnalytics(ctx context.Context, filter LocationStatsFilter) (*models.FrameRateAnalytics, error) {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	whereClauses, args := buildFilterConditions(filter, false, 1)
	whereClause := buildWhereClause(whereClauses)

	// Get total playbacks
	total, err := db.getTotalPlaybacks(ctx, whereClause, args)
	if err != nil {
		return nil, errorContext("get total playbacks", err)
	}

	// Get frame rate distribution with high FPS adoption
	frameDist, highFPSAdoption, err := db.getFrameRateDistribution(ctx, whereClause, args, total)
	if err != nil {
		return nil, errorContext("get frame rate distribution", err)
	}

	// Get breakdown by media type
	byMediaType, err := db.getFrameRateByMediaType(ctx, whereClause, args)
	if err != nil {
		return nil, errorContext("get frame rate by media type", err)
	}

	return &models.FrameRateAnalytics{
		TotalPlaybacks:        total,
		FrameRateDistribution: frameDist,
		ByMediaType:           byMediaType,
		HighFrameRateAdoption: highFPSAdoption,
		ConversionEvents:      []models.FrameRateConversion{}, // Placeholder for future enhancement
	}, nil
}

// getFrameRateDistribution retrieves frame rate distribution and calculates high FPS adoption rate
func (db *DB) getFrameRateDistribution(ctx context.Context, whereClause string, args []interface{}, total int) ([]models.FrameRateDistribution, float64, error) {
	query := fmt.Sprintf(`
		SELECT
			COALESCE(video_framerate, '24p') as framerate,
			COUNT(*) as playback_count,
			(COUNT(*) * 100.0 / %d) as percentage,
			AVG(percent_complete) as avg_completion
		FROM playback_events
		%s
		GROUP BY video_framerate
		ORDER BY playback_count DESC
	`, total, whereClause)

	rows, err := db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query frame rate distribution: %w", err)
	}
	defer rows.Close()

	var frameDist []models.FrameRateDistribution
	highFPSCount := 0
	for rows.Next() {
		var d models.FrameRateDistribution
		err := rows.Scan(&d.FrameRate, &d.PlaybackCount, &d.Percentage, &d.AvgCompletion)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan frame rate row: %w", err)
		}
		frameDist = append(frameDist, d)

		// Count 60fps+ as high frame rate
		if strings.Contains(strings.ToLower(d.FrameRate), "60") || strings.Contains(strings.ToLower(d.FrameRate), "120") {
			highFPSCount += d.PlaybackCount
		}
	}

	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating rows: %w", err)
	}

	highFPSAdoption := calculateAdoptionRate(highFPSCount, total)

	return frameDist, highFPSAdoption, nil
}

// getFrameRateByMediaType retrieves frame rate distribution grouped by media type
func (db *DB) getFrameRateByMediaType(ctx context.Context, whereClause string, args []interface{}) (map[string][]models.FrameRateDistribution, error) {
	query := fmt.Sprintf(`
		SELECT
			media_type,
			COALESCE(video_framerate, '24p') as framerate,
			COUNT(*) as playback_count,
			AVG(percent_complete) as avg_completion
		FROM playback_events
		%s
		GROUP BY media_type, video_framerate
		ORDER BY media_type, playback_count DESC
	`, whereClause)

	rows, err := db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query by media type: %w", err)
	}
	defer rows.Close()

	byMediaType := make(map[string][]models.FrameRateDistribution)
	for rows.Next() {
		var mediaType string
		var d models.FrameRateDistribution
		err := rows.Scan(&mediaType, &d.FrameRate, &d.PlaybackCount, &d.AvgCompletion)
		if err != nil {
			return nil, fmt.Errorf("failed to scan media type row: %w", err)
		}
		byMediaType[mediaType] = append(byMediaType[mediaType], d)
	}

	return byMediaType, rows.Err()
}
