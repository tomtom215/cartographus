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

// GetSubtitleAnalytics returns comprehensive subtitle usage analytics including overall usage rates,
// language distribution, codec distribution, and user-specific subtitle preferences.
func (db *DB) GetSubtitleAnalytics(ctx context.Context, filter LocationStatsFilter) (*models.SubtitleAnalytics, error) {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	whereClauses, args := buildFilterConditions(filter, false, 1)
	whereClause := buildWhereClause(whereClauses)

	// Get overall subtitle usage stats
	total, withSubs, withoutSubs, usageRate, err := db.getSubtitleSummary(ctx, whereClause, args)
	if err != nil {
		return nil, errorContext("get subtitle summary", err)
	}

	// Get language distribution
	langDist, err := db.getSubtitleLanguageDistribution(ctx, whereClause, args, withSubs)
	if err != nil {
		return nil, errorContext("get language distribution", err)
	}

	// Get codec distribution
	codecDist, err := db.getSubtitleCodecDistribution(ctx, whereClause, args, withSubs)
	if err != nil {
		return nil, errorContext("get codec distribution", err)
	}

	// Get user preferences
	userPrefs, err := db.getSubtitleUserPreferences(ctx, whereClause, args)
	if err != nil {
		return nil, errorContext("get user preferences", err)
	}

	return &models.SubtitleAnalytics{
		TotalPlaybacks:       total,
		SubtitleUsageRate:    usageRate,
		PlaybacksWithSubs:    withSubs,
		PlaybacksWithoutSubs: withoutSubs,
		LanguageDistribution: langDist,
		CodecDistribution:    codecDist,
		UserPreferences:      userPrefs,
	}, nil
}

// getSubtitleSummary retrieves overall subtitle usage statistics
func (db *DB) getSubtitleSummary(ctx context.Context, whereClause string, args []interface{}) (int, int, int, float64, error) {
	query := fmt.Sprintf(`
		SELECT
			COUNT(*) as total,
			COALESCE(SUM(CASE WHEN subtitles = 1 THEN 1 ELSE 0 END), 0) as with_subs,
			COALESCE(SUM(CASE WHEN subtitles = 0 OR subtitles IS NULL THEN 1 ELSE 0 END), 0) as without_subs
		FROM playback_events
		%s
	`, whereClause)

	var total, withSubs, withoutSubs int
	err := db.querySingleRow(ctx, query, args, &total, &withSubs, &withoutSubs)
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("failed to get subtitle stats: %w", err)
	}

	usageRate := calculatePercentage(withSubs, total)

	return total, withSubs, withoutSubs, usageRate, nil
}

// getSubtitleLanguageDistribution retrieves subtitle language distribution statistics
func (db *DB) getSubtitleLanguageDistribution(ctx context.Context, whereClause string, args []interface{}, withSubs int) ([]models.SubtitleLanguageStats, error) {
	langCondition := "subtitle_language IS NOT NULL AND subtitles = 1"
	langWhere := appendWhereCondition(whereClause, langCondition)

	query := fmt.Sprintf(`
		SELECT
			subtitle_language,
			COUNT(*) as playback_count,
			(COUNT(*) * 100.0 / %d) as percentage,
			COUNT(DISTINCT username) as unique_users
		FROM playback_events
		%s
		GROUP BY subtitle_language
		ORDER BY playback_count DESC
		LIMIT 15
	`, withSubs, langWhere)

	rows, err := db.conn.QueryContext(ctx, query, args...)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("failed to query language distribution: %w", err)
	}

	var langDist []models.SubtitleLanguageStats
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var l models.SubtitleLanguageStats
			if err := rows.Scan(&l.Language, &l.PlaybackCount, &l.Percentage, &l.UniqueUsers); err != nil {
				return nil, fmt.Errorf("failed to scan language row: %w", err)
			}
			langDist = append(langDist, l)
		}

		if err = rows.Err(); err != nil {
			return nil, fmt.Errorf("error iterating lang rows: %w", err)
		}
	}

	return langDist, nil
}

// getSubtitleCodecDistribution retrieves subtitle codec distribution statistics
func (db *DB) getSubtitleCodecDistribution(ctx context.Context, whereClause string, args []interface{}, withSubs int) ([]models.SubtitleCodecStats, error) {
	codecCondition := "subtitle_codec IS NOT NULL AND subtitles = 1"
	codecWhere := appendWhereCondition(whereClause, codecCondition)

	query := fmt.Sprintf(`
		SELECT
			subtitle_codec,
			COUNT(*) as playback_count,
			(COUNT(*) * 100.0 / %d) as percentage
		FROM playback_events
		%s
		GROUP BY subtitle_codec
		ORDER BY playback_count DESC
	`, withSubs, codecWhere)

	rows, err := db.conn.QueryContext(ctx, query, args...)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("failed to query codec distribution: %w", err)
	}

	var codecDist []models.SubtitleCodecStats
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var c models.SubtitleCodecStats
			if err := rows.Scan(&c.Codec, &c.PlaybackCount, &c.Percentage); err != nil {
				return nil, fmt.Errorf("failed to scan codec row: %w", err)
			}
			codecDist = append(codecDist, c)
		}

		if err = rows.Err(); err != nil {
			return nil, fmt.Errorf("error iterating codec rows: %w", err)
		}
	}

	return codecDist, nil
}

// getSubtitleUserPreferences retrieves user subtitle usage preferences
func (db *DB) getSubtitleUserPreferences(ctx context.Context, whereClause string, args []interface{}) ([]models.UserSubtitlePreference, error) {
	query := fmt.Sprintf(`
		SELECT
			username,
			COUNT(*) as total_playbacks,
			SUM(CASE WHEN subtitles = 1 THEN 1 ELSE 0 END) as subtitle_usage_count,
			(SUM(CASE WHEN subtitles = 1 THEN 1 ELSE 0 END) * 100.0 / COUNT(*)) as usage_rate,
			string_agg(DISTINCT subtitle_language, ',') as preferred_languages
		FROM playback_events
		%s
		GROUP BY username
		HAVING subtitle_usage_count > 0
		ORDER BY subtitle_usage_count DESC
		LIMIT 15
	`, whereClause)

	rows, err := db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query user preferences: %w", err)
	}
	defer rows.Close()

	var userPrefs []models.UserSubtitlePreference
	for rows.Next() {
		var u models.UserSubtitlePreference
		var langsStr sql.NullString
		if err := rows.Scan(&u.Username, &u.TotalPlaybacks, &u.SubtitleUsageCount, &u.SubtitleUsageRate, &langsStr); err != nil {
			return nil, fmt.Errorf("failed to scan user row: %w", err)
		}
		if langsStr.Valid {
			u.PreferredLanguages = parseList(langsStr.String)
		}
		userPrefs = append(userPrefs, u)
	}

	return userPrefs, rows.Err()
}
