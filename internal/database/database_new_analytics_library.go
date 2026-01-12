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

// GetLibraryAnalytics returns comprehensive library-specific analytics including total playbacks,
// unique users, watch time, most watched content, top users, quality distribution, and content health metrics.
func (db *DB) GetLibraryAnalytics(ctx context.Context, sectionID int, filter LocationStatsFilter) (*models.LibraryAnalytics, error) {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	whereClauses, args := buildFilterConditions(filter, false, 1)

	// Add section_id filter
	whereClauses = append(whereClauses, "section_id = ?")
	args = append(args, sectionID)

	whereClause := buildWhereClause(whereClauses)

	// Get library basic info
	analytics, err := db.getLibraryBasicInfo(ctx, whereClause, args)
	if err != nil {
		return nil, errorContext("get library basic info", err)
	}

	// Get top users
	topUsers, err := db.getLibraryTopUsers(ctx, whereClause, args)
	if err != nil {
		return nil, errorContext("get top users", err)
	}
	analytics.TopUsers = topUsers

	// Get quality distribution
	qualityDist, err := db.getLibraryQualityDistribution(ctx, whereClause, args)
	if err != nil {
		return nil, errorContext("get quality distribution", err)
	}
	analytics.QualityDistribution = qualityDist

	// Calculate health metrics
	if analytics.TotalPlaybacks > 0 {
		analytics.ContentHealth.PopularityScore = float64(analytics.TotalPlaybacks) / float64(analytics.UniqueUsers)
		analytics.ContentHealth.EngagementScore = analytics.AvgCompletion
	}

	return analytics, nil
}

// getLibraryBasicInfo retrieves library metadata and basic playback statistics
func (db *DB) getLibraryBasicInfo(ctx context.Context, whereClause string, args []interface{}) (*models.LibraryAnalytics, error) {
	query := fmt.Sprintf(`
		WITH title_counts AS (
			SELECT
				title,
				COUNT(*) as title_count,
				ROW_NUMBER() OVER (ORDER BY COUNT(*) DESC) as rn
			FROM playback_events
			%s
			GROUP BY title
		)
		SELECT
			section_id,
			library_name,
			media_type,
			COUNT(*) as total_playbacks,
			COUNT(DISTINCT username) as unique_users,
			COALESCE(SUM(COALESCE(play_duration, 0)), 0) as total_watch_time,
			COALESCE(AVG(percent_complete), 0) as avg_completion,
			(SELECT title FROM title_counts WHERE rn = 1) as most_watched
		FROM playback_events
		%s
		GROUP BY section_id, library_name, media_type
	`, whereClause, whereClause)

	// Duplicate args for the CTE query (used twice)
	queryArgs := append(args, args...)

	var analytics models.LibraryAnalytics
	err := db.conn.QueryRowContext(ctx, query, queryArgs...).Scan(
		&analytics.LibraryID,
		&analytics.LibraryName,
		&analytics.MediaType,
		&analytics.TotalPlaybacks,
		&analytics.UniqueUsers,
		&analytics.TotalWatchTime,
		&analytics.AvgCompletion,
		&analytics.MostWatchedItem,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			// No data in database, return empty analytics
			return &models.LibraryAnalytics{}, nil
		}
		return nil, fmt.Errorf("failed to get library info: %w", err)
	}

	return &analytics, nil
}

// getLibraryTopUsers retrieves top 10 users by playback count for the library
func (db *DB) getLibraryTopUsers(ctx context.Context, whereClause string, args []interface{}) ([]models.LibraryUserStats, error) {
	query := fmt.Sprintf(`
		SELECT
			username,
			COUNT(*) as plays,
			SUM(COALESCE(play_duration, 0)) as watch_time,
			AVG(percent_complete) as avg_completion
		FROM playback_events
		%s
		GROUP BY username
		ORDER BY plays DESC
		LIMIT 10
	`, whereClause)

	rows, err := db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query top users: %w", err)
	}
	defer rows.Close()

	var topUsers []models.LibraryUserStats
	for rows.Next() {
		var u models.LibraryUserStats
		err := rows.Scan(&u.Username, &u.Plays, &u.WatchTime, &u.Completion)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user row: %w", err)
		}
		topUsers = append(topUsers, u)
	}

	return topUsers, rows.Err()
}

// getLibraryQualityDistribution retrieves quality metrics (HDR, 4K, surround sound, bitrate)
func (db *DB) getLibraryQualityDistribution(ctx context.Context, whereClause string, args []interface{}) (models.LibraryQualityStats, error) {
	query := fmt.Sprintf(`
		SELECT
			COALESCE(SUM(CASE WHEN video_dynamic_range IS NOT NULL AND video_dynamic_range != 'SDR' THEN 1 ELSE 0 END), 0) as hdr_count,
			COALESCE(SUM(CASE WHEN video_resolution IN ('2160', '4k', 'UHD') THEN 1 ELSE 0 END), 0) as fourk_count,
			COALESCE(SUM(CASE WHEN audio_channels NOT IN ('1', '2') THEN 1 ELSE 0 END), 0) as surround_count,
			COALESCE(AVG(COALESCE(bitrate, 0)), 0) as avg_bitrate
		FROM playback_events
		%s
	`, whereClause)

	var qualityDist models.LibraryQualityStats
	err := db.conn.QueryRowContext(ctx, query, args...).Scan(
		&qualityDist.HDRContent,
		&qualityDist.FourKContent,
		&qualityDist.SurroundContent,
		&qualityDist.AvgBitrate,
	)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return models.LibraryQualityStats{}, fmt.Errorf("failed to get quality distribution: %w", err)
	}

	return qualityDist, nil
}
