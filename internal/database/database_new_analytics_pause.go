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

// GetPausePatternAnalytics analyzes playback pause behavior including pause frequency distribution,
// high-pause content identification, user pause patterns, and engagement quality indicators.
func (db *DB) GetPausePatternAnalytics(ctx context.Context, filter LocationStatsFilter) (*models.PausePatternAnalytics, error) {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	whereClauses, args := buildFilterConditions(filter, false, 1)
	whereClause := buildWhereClause(whereClauses)

	// Get overall stats
	total, avgPauses, err := db.getPauseOverallStats(ctx, whereClause, args)
	if err != nil {
		return nil, errorContext("get overall stats", err)
	}

	// Get high pause content
	highPauseContent, err := db.getHighPauseContent(ctx, whereClause, args)
	if err != nil {
		return nil, errorContext("get high pause content", err)
	}

	// Get pause distribution
	pauseDist, err := db.getPauseDistribution(ctx, whereClause, args, total)
	if err != nil {
		return nil, errorContext("get pause distribution", err)
	}

	// Get user pause patterns
	userPatterns, err := db.getUserPausePatterns(ctx, whereClause, args)
	if err != nil {
		return nil, errorContext("get user patterns", err)
	}

	// Calculate quality indicators
	qualityIndicators := calculatePauseQualityIndicators(highPauseContent, pauseDist)

	return &models.PausePatternAnalytics{
		TotalPlaybacks:      total,
		AvgPausesPerSession: avgPauses,
		HighPauseContent:    highPauseContent,
		PauseDistribution:   pauseDist,
		PauseTimingHeatmap:  []models.PauseTimingBucket{},
		UserPausePatterns:   userPatterns,
		QualityIndicators:   qualityIndicators,
	}, nil
}

// getPauseOverallStats queries total playback count and average pauses per session
// for baseline pause pattern metrics.
func (db *DB) getPauseOverallStats(ctx context.Context, whereClause string, args []interface{}) (int, float64, error) {
	query := fmt.Sprintf(`
		SELECT
			COUNT(*) as total_playbacks,
			COALESCE(AVG(paused_counter), 0) as avg_pauses
		FROM playback_events
		%s
	`, whereClause)

	var total int
	var avgPauses float64
	if err := db.conn.QueryRowContext(ctx, query, args...).Scan(&total, &avgPauses); err != nil {
		return 0, 0, fmt.Errorf("failed to get pause stats: %w", err)
	}

	return total, avgPauses, nil
}

// getHighPauseContent queries the top 20 content items with the highest average pause counts
// (minimum 3 playbacks) and identifies potential quality issues.
func (db *DB) getHighPauseContent(ctx context.Context, whereClause string, args []interface{}) ([]models.HighPauseContent, error) {
	query := fmt.Sprintf(`
		SELECT
			title,
			media_type,
			AVG(paused_counter) as avg_pauses,
			AVG(percent_complete) as avg_completion,
			COUNT(*) as playback_count
		FROM playback_events
		%s
		GROUP BY title, media_type
		HAVING playback_count >= 3
		ORDER BY avg_pauses DESC
		LIMIT 20
	`, whereClause)

	rows, err := db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query high pause content: %w", err)
	}
	defer rows.Close()

	var results []models.HighPauseContent
	for rows.Next() {
		var h models.HighPauseContent
		if err := rows.Scan(&h.Title, &h.MediaType, &h.AveragePauses, &h.CompletionRate, &h.PlaybackCount); err != nil {
			return nil, fmt.Errorf("failed to scan content row: %w", err)
		}
		h.PotentialQualityIssue = h.AveragePauses > 10 && h.CompletionRate < 50
		results = append(results, h)
	}

	return results, rows.Err()
}

// getPauseDistribution queries pause frequency distribution grouped into buckets
// (0-2, 3-5, 6-10, 11+) with playback counts, percentages, and average completion rates.
func (db *DB) getPauseDistribution(ctx context.Context, whereClause string, args []interface{}, total int) ([]models.PauseDistribution, error) {
	query := fmt.Sprintf(`
		SELECT
			CASE
				WHEN paused_counter <= 2 THEN '0-2'
				WHEN paused_counter <= 5 THEN '3-5'
				WHEN paused_counter <= 10 THEN '6-10'
				ELSE '11+'
			END as pause_bucket,
			COUNT(*) as playback_count,
			(COUNT(*) * 100.0 / %d) as percentage,
			AVG(percent_complete) as avg_completion
		FROM playback_events
		%s
		GROUP BY pause_bucket
		ORDER BY
			CASE pause_bucket
				WHEN '0-2' THEN 1
				WHEN '3-5' THEN 2
				WHEN '6-10' THEN 3
				ELSE 4
			END
	`, total, whereClause)

	rows, err := db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query pause distribution: %w", err)
	}
	defer rows.Close()

	var results []models.PauseDistribution
	for rows.Next() {
		var d models.PauseDistribution
		if err := rows.Scan(&d.PauseBucket, &d.PlaybackCount, &d.Percentage, &d.AvgCompletion); err != nil {
			return nil, fmt.Errorf("failed to scan distribution row: %w", err)
		}
		results = append(results, d)
	}

	return results, rows.Err()
}

// getUserPausePatterns queries the top 15 users with lowest average pause counts
// and identifies binge-watchers (users with <2 average pauses per session).
func (db *DB) getUserPausePatterns(ctx context.Context, whereClause string, args []interface{}) ([]models.UserPauseStats, error) {
	query := fmt.Sprintf(`
		SELECT
			username,
			AVG(paused_counter) as avg_pauses,
			COUNT(*) as total_sessions
		FROM playback_events
		%s
		GROUP BY username
		ORDER BY avg_pauses ASC
		LIMIT 15
	`, whereClause)

	rows, err := db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query user patterns: %w", err)
	}
	defer rows.Close()

	var results []models.UserPauseStats
	for rows.Next() {
		var u models.UserPauseStats
		if err := rows.Scan(&u.Username, &u.AvgPauses, &u.TotalSessions); err != nil {
			return nil, fmt.Errorf("failed to scan user row: %w", err)
		}
		u.BingeWatcher = u.AvgPauses < 2
		results = append(results, u)
	}

	return results, rows.Err()
}

// calculatePauseQualityIndicators computes engagement quality metrics based on pause patterns
// including low engagement count (11+ pauses) and potential quality issues detection.
func calculatePauseQualityIndicators(highPauseContent []models.HighPauseContent, pauseDist []models.PauseDistribution) models.PauseQualityMetrics {
	lowEngagementCount := 0
	potentialIssues := 0

	for _, content := range highPauseContent {
		if content.PotentialQualityIssue {
			potentialIssues++
		}
	}

	for _, dist := range pauseDist {
		if dist.PauseBucket == "11+" {
			lowEngagementCount = dist.PlaybackCount
		}
	}

	return models.PauseQualityMetrics{
		HighEngagementThreshold: 2.0,
		LowEngagementCount:      lowEngagementCount,
		PotentialIssuesDetected: potentialIssues,
	}
}
