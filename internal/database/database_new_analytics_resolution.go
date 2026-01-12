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

// GetResolutionMismatchAnalytics analyzes resolution downgrade patterns including mismatch rates,
// detailed source→stream patterns, top downgrade users, and platform-specific mismatch statistics
// for quality optimization and bandwidth planning.
func (db *DB) GetResolutionMismatchAnalytics(ctx context.Context, filter LocationStatsFilter) (*models.ResolutionMismatchAnalytics, error) {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	// Get overall mismatch statistics
	total, mismatched, directPlay, transcode, mismatchRate, err := db.getResolutionMismatchStats(ctx, filter)
	if err != nil {
		return nil, errorContext("get mismatch stats", err)
	}

	// Get detailed mismatch patterns (source→stream)
	mismatches, err := db.getDetailedMismatchPatterns(ctx, filter, total)
	if err != nil {
		return nil, errorContext("get detailed patterns", err)
	}

	// Get top downgrade users
	topUsers, err := db.getTopDowngradeUsers(ctx, filter)
	if err != nil {
		return nil, errorContext("get top downgrade users", err)
	}

	// Get mismatch by platform
	platformMismatches, err := db.getMismatchByPlatform(ctx, filter)
	if err != nil {
		return nil, errorContext("get platform mismatches", err)
	}

	return &models.ResolutionMismatchAnalytics{
		TotalPlaybacks:      total,
		MismatchedPlaybacks: mismatched,
		MismatchRate:        mismatchRate,
		Mismatches:          mismatches,
		DirectPlayCount:     directPlay,
		TranscodeCount:      transcode,
		TopDowngradeUsers:   topUsers,
		MismatchByPlatform:  platformMismatches,
	}, nil
}

// getResolutionMismatchStats retrieves overall mismatch statistics including total playbacks,
// mismatch count, direct play/transcode counts, and calculates mismatch rate percentage.
func (db *DB) getResolutionMismatchStats(ctx context.Context, filter LocationStatsFilter) (int, int, int, int, float64, error) {
	whereClauses, args := buildFilterConditions(filter, false, 1)
	whereClause := buildWhereClause(whereClauses)

	query := fmt.Sprintf(`
		SELECT
			COUNT(*) as total_playbacks,
			COALESCE(SUM(CASE WHEN video_resolution IS NOT NULL AND stream_video_resolution IS NOT NULL
				AND video_resolution != stream_video_resolution THEN 1 ELSE 0 END), 0) as mismatched,
			COALESCE(SUM(CASE WHEN transcode_decision = 'direct play' THEN 1 ELSE 0 END), 0) as direct_play,
			COALESCE(SUM(CASE WHEN transcode_decision = 'transcode' THEN 1 ELSE 0 END), 0) as transcode
		FROM playback_events
		%s
	`, whereClause)

	var total, mismatched, directPlay, transcode int
	err := db.querySingleRow(ctx, query, args, &total, &mismatched, &directPlay, &transcode)
	if err != nil {
		return 0, 0, 0, 0, 0, fmt.Errorf("failed to get mismatch stats: %w", err)
	}

	mismatchRate := calculatePercentage(mismatched, total)

	return total, mismatched, directPlay, transcode, mismatchRate, nil
}

// getDetailedMismatchPatterns retrieves top 20 resolution mismatch patterns showing source→stream
// resolution pairs with affected users, platforms, and whether transcoding is required.
func (db *DB) getDetailedMismatchPatterns(ctx context.Context, filter LocationStatsFilter, total int) ([]models.ResolutionMismatch, error) {
	whereClauses, args := buildFilterConditions(filter, false, 1)
	baseWhere := buildWhereClause(whereClauses)

	// Build mismatch-specific WHERE clause
	mismatchCondition := "video_resolution IS NOT NULL AND stream_video_resolution IS NOT NULL AND video_resolution != stream_video_resolution"
	mismatchWhere := appendWhereCondition(baseWhere, mismatchCondition)

	query := fmt.Sprintf(`
		SELECT
			video_resolution as source,
			stream_video_resolution as stream,
			COUNT(*) as count,
			(COUNT(*) * 100.0 / %d) as percentage,
			string_agg(DISTINCT username, ',') as users,
			string_agg(DISTINCT platform, ',') as platforms,
			BOOL_OR(transcode_decision = 'transcode') as requires_transcode
		FROM playback_events
		%s
		GROUP BY video_resolution, stream_video_resolution
		ORDER BY count DESC
		LIMIT 20
	`, total, mismatchWhere)

	rows, err := db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query mismatches: %w", err)
	}
	defer rows.Close()

	var mismatches []models.ResolutionMismatch
	for rows.Next() {
		var m models.ResolutionMismatch
		var usersStr, platformsStr string
		err := rows.Scan(&m.SourceResolution, &m.StreamResolution, &m.PlaybackCount, &m.Percentage,
			&usersStr, &platformsStr, &m.TranscodeRequired)
		if err != nil {
			return nil, fmt.Errorf("failed to scan mismatch row: %w", err)
		}
		m.AffectedUsers = parseAggregatedList(usersStr)
		m.CommonPlatforms = parseAggregatedList(platformsStr)
		mismatches = append(mismatches, m)
	}

	return mismatches, rows.Err()
}

// getTopDowngradeUsers retrieves top 10 users experiencing the most resolution downgrades,
// including their total playbacks, downgrade count, rate, and common reason.
func (db *DB) getTopDowngradeUsers(ctx context.Context, filter LocationStatsFilter) ([]models.UserDowngradeStats, error) {
	whereClauses, args := buildFilterConditions(filter, false, 1)
	baseWhere := buildWhereClause(whereClauses)

	// Add resolution fields filter
	userCondition := "video_resolution IS NOT NULL AND stream_video_resolution IS NOT NULL"
	userWhere := appendWhereCondition(baseWhere, userCondition)

	query := fmt.Sprintf(`
		SELECT
			username,
			COUNT(*) as total_playbacks,
			SUM(CASE WHEN video_resolution != stream_video_resolution THEN 1 ELSE 0 END) as downgrades,
			(SUM(CASE WHEN video_resolution != stream_video_resolution THEN 1 ELSE 0 END) * 100.0 / COUNT(*)) as rate,
			'bandwidth_limited' as reason
		FROM playback_events
		%s
		GROUP BY username
		HAVING downgrades > 0
		ORDER BY downgrades DESC
		LIMIT 10
	`, userWhere)

	rows, err := db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query user downgrades: %w", err)
	}
	defer rows.Close()

	var topUsers []models.UserDowngradeStats
	for rows.Next() {
		var u models.UserDowngradeStats
		err := rows.Scan(&u.Username, &u.TotalPlaybacks, &u.DowngradeCount, &u.DowngradeRate, &u.CommonReason)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user row: %w", err)
		}
		topUsers = append(topUsers, u)
	}

	return topUsers, rows.Err()
}

// getMismatchByPlatform retrieves top 10 platforms with resolution mismatches,
// showing mismatch counts, total playbacks, and mismatch rate percentage.
func (db *DB) getMismatchByPlatform(ctx context.Context, filter LocationStatsFilter) ([]models.PlatformMismatch, error) {
	whereClauses, args := buildFilterConditions(filter, false, 1)
	baseWhere := buildWhereClause(whereClauses)

	// Add resolution fields filter
	platformCondition := "video_resolution IS NOT NULL AND stream_video_resolution IS NOT NULL"
	platformWhere := appendWhereCondition(baseWhere, platformCondition)

	query := fmt.Sprintf(`
		SELECT
			platform,
			SUM(CASE WHEN video_resolution != stream_video_resolution THEN 1 ELSE 0 END) as mismatch_count,
			COUNT(*) as total_playbacks,
			(SUM(CASE WHEN video_resolution != stream_video_resolution THEN 1 ELSE 0 END) * 100.0 / COUNT(*)) as rate
		FROM playback_events
		%s
		GROUP BY platform
		ORDER BY mismatch_count DESC
		LIMIT 10
	`, platformWhere)

	rows, err := db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query platform mismatches: %w", err)
	}
	defer rows.Close()

	var platformMismatches []models.PlatformMismatch
	for rows.Next() {
		var pm models.PlatformMismatch
		err := rows.Scan(&pm.Platform, &pm.MismatchCount, &pm.TotalPlaybacks, &pm.MismatchRate)
		if err != nil {
			return nil, fmt.Errorf("failed to scan platform row: %w", err)
		}
		platformMismatches = append(platformMismatches, pm)
	}

	return platformMismatches, rows.Err()
}
