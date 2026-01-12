// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package database provides data access and analytics functionality for the Cartographus application.
// This file contains approximate analytics methods using the DataSketches extension for O(1) space complexity
// on large datasets. Falls back to exact calculations when the extension is unavailable.
package database

import (
	"context"
	"fmt"
	"time"

	"github.com/tomtom215/cartographus/internal/logging"
)

// ApproximateStats contains approximate analytics metrics using DataSketches
type ApproximateStats struct {
	// Approximate distinct counts using HyperLogLog
	UniqueUsers     int64 `json:"unique_users"`
	UniqueTitles    int64 `json:"unique_titles"`
	UniqueIPs       int64 `json:"unique_ips"`
	UniquePlatforms int64 `json:"unique_platforms"`
	UniquePlayers   int64 `json:"unique_players"`

	// Approximate percentiles using KLL sketches
	WatchTimeP50 float64 `json:"watch_time_p50"` // Median watch time (minutes)
	WatchTimeP75 float64 `json:"watch_time_p75"` // 75th percentile
	WatchTimeP90 float64 `json:"watch_time_p90"` // 90th percentile
	WatchTimeP95 float64 `json:"watch_time_p95"` // 95th percentile
	WatchTimeP99 float64 `json:"watch_time_p99"` // 99th percentile

	// Total count (exact)
	TotalPlaybacks int64 `json:"total_playbacks"`

	// Metadata
	IsApproximate bool    `json:"is_approximate"`        // true if DataSketches was used
	ErrorBound    float64 `json:"error_bound,omitempty"` // Estimated error bound for HLL (typically <2%)
	QueryTimeMS   int64   `json:"query_time_ms"`
}

// ApproximateStatsFilter defines filters for approximate stats queries
type ApproximateStatsFilter struct {
	StartDate  *time.Time
	EndDate    *time.Time
	Users      []string
	MediaTypes []string
}

// GetApproximateStats returns approximate analytics using DataSketches when available,
// falling back to exact calculations otherwise. This is useful for dashboards where
// exact counts are not required but fast response times are essential.
func (db *DB) GetApproximateStats(ctx context.Context, filter ApproximateStatsFilter) (*ApproximateStats, error) {
	start := time.Now()

	if db.datasketchesAvailable {
		stats, err := db.getApproximateStatsWithSketches(ctx, filter)
		if err == nil {
			stats.QueryTimeMS = time.Since(start).Milliseconds()
			return stats, nil
		}
		// Fall through to exact calculation on error
		logging.Warn().Err(err).Msg("DataSketches query failed, falling back to exact")
	}

	// Fall back to exact calculation
	stats, err := db.getExactStats(ctx, filter)
	if err != nil {
		return nil, err
	}
	stats.QueryTimeMS = time.Since(start).Milliseconds()
	return stats, nil
}

// getApproximateStatsWithSketches uses DataSketches HLL and KLL for approximate metrics
func (db *DB) getApproximateStatsWithSketches(ctx context.Context, filter ApproximateStatsFilter) (*ApproximateStats, error) {
	whereClause, args := buildApproximateWhereClause(filter)

	// Query using DataSketches functions
	// Note: datasketch_hll(precision, value) is an aggregate that creates HLL sketch
	// Note: datasketch_hll_estimate extracts approximate distinct count from sketch
	query := fmt.Sprintf(`
		SELECT
			-- Approximate distinct counts using HyperLogLog
			COALESCE(datasketch_hll_estimate(user_hll), 0) as unique_users,
			COALESCE(datasketch_hll_estimate(title_hll), 0) as unique_titles,
			COALESCE(datasketch_hll_estimate(ip_hll), 0) as unique_ips,
			COALESCE(datasketch_hll_estimate(platform_hll), 0) as unique_platforms,
			COALESCE(datasketch_hll_estimate(player_hll), 0) as unique_players,
			-- Total count (exact)
			total_playbacks
		FROM (
			SELECT
				datasketch_hll(12, username) as user_hll,
				datasketch_hll(12, title) as title_hll,
				datasketch_hll(12, ip_address) as ip_hll,
				datasketch_hll(12, COALESCE(platform, 'Unknown')) as platform_hll,
				datasketch_hll(12, COALESCE(player, 'Unknown')) as player_hll,
				COUNT(*) as total_playbacks
			FROM playback_events
			WHERE %s
		) as sketches
	`, whereClause)

	var stats ApproximateStats
	stats.IsApproximate = true
	stats.ErrorBound = 0.02 // HyperLogLog typical error bound

	err := db.conn.QueryRowContext(ctx, query, args...).Scan(
		&stats.UniqueUsers,
		&stats.UniqueTitles,
		&stats.UniqueIPs,
		&stats.UniquePlatforms,
		&stats.UniquePlayers,
		&stats.TotalPlaybacks,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get approximate distinct counts: %w", err)
	}

	// Get percentiles using KLL sketches in a separate query
	// datasketch_kll(k, value) - k=200 gives ~1.65% error for PMF
	// datasketch_kll_quantile(sketch, rank, inclusive) - rank must be DOUBLE, inclusive=true
	percentileQuery := fmt.Sprintf(`
		SELECT
			COALESCE(datasketch_kll_quantile(watch_time_kll, 0.50::DOUBLE, true), 0) as p50,
			COALESCE(datasketch_kll_quantile(watch_time_kll, 0.75::DOUBLE, true), 0) as p75,
			COALESCE(datasketch_kll_quantile(watch_time_kll, 0.90::DOUBLE, true), 0) as p90,
			COALESCE(datasketch_kll_quantile(watch_time_kll, 0.95::DOUBLE, true), 0) as p95,
			COALESCE(datasketch_kll_quantile(watch_time_kll, 0.99::DOUBLE, true), 0) as p99
		FROM (
			SELECT datasketch_kll(200, CAST(play_duration / 60.0 AS DOUBLE)) as watch_time_kll
			FROM playback_events
			WHERE %s AND play_duration > 0
		) as kll_sketch
	`, whereClause)

	err = db.conn.QueryRowContext(ctx, percentileQuery, args...).Scan(
		&stats.WatchTimeP50,
		&stats.WatchTimeP75,
		&stats.WatchTimeP90,
		&stats.WatchTimeP95,
		&stats.WatchTimeP99,
	)
	if err != nil {
		// Percentile query failure is non-fatal, just set to 0
		logging.Warn().Err(err).Msg("Percentile query failed")
	}

	return &stats, nil
}

// getExactStats falls back to exact calculations when DataSketches is unavailable
func (db *DB) getExactStats(ctx context.Context, filter ApproximateStatsFilter) (*ApproximateStats, error) {
	whereClause, args := buildApproximateWhereClause(filter)

	query := fmt.Sprintf(`
		SELECT
			COUNT(DISTINCT username) as unique_users,
			COUNT(DISTINCT title) as unique_titles,
			COUNT(DISTINCT ip_address) as unique_ips,
			COUNT(DISTINCT COALESCE(platform, 'Unknown')) as unique_platforms,
			COUNT(DISTINCT COALESCE(player, 'Unknown')) as unique_players,
			COUNT(*) as total_playbacks
		FROM playback_events
		WHERE %s
	`, whereClause)

	var stats ApproximateStats
	stats.IsApproximate = false

	err := db.conn.QueryRowContext(ctx, query, args...).Scan(
		&stats.UniqueUsers,
		&stats.UniqueTitles,
		&stats.UniqueIPs,
		&stats.UniquePlatforms,
		&stats.UniquePlayers,
		&stats.TotalPlaybacks,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get exact distinct counts: %w", err)
	}

	// Get percentiles using PERCENTILE_CONT (DuckDB exact percentiles)
	percentileQuery := fmt.Sprintf(`
		SELECT
			COALESCE(PERCENTILE_CONT(0.50) WITHIN GROUP (ORDER BY play_duration / 60.0), 0) as p50,
			COALESCE(PERCENTILE_CONT(0.75) WITHIN GROUP (ORDER BY play_duration / 60.0), 0) as p75,
			COALESCE(PERCENTILE_CONT(0.90) WITHIN GROUP (ORDER BY play_duration / 60.0), 0) as p90,
			COALESCE(PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY play_duration / 60.0), 0) as p95,
			COALESCE(PERCENTILE_CONT(0.99) WITHIN GROUP (ORDER BY play_duration / 60.0), 0) as p99
		FROM playback_events
		WHERE %s AND play_duration > 0
	`, whereClause)

	err = db.conn.QueryRowContext(ctx, percentileQuery, args...).Scan(
		&stats.WatchTimeP50,
		&stats.WatchTimeP75,
		&stats.WatchTimeP90,
		&stats.WatchTimeP95,
		&stats.WatchTimeP99,
	)
	if err != nil {
		// Percentile query failure is non-fatal
		logging.Warn().Err(err).Msg("Exact percentile query failed")
	}

	return &stats, nil
}

// validDistinctColumns maps user-provided column names to safe SQL identifiers
// to prevent SQL injection. Only these predefined columns can be used.
var validDistinctColumns = map[string]string{
	"username":      "username",
	"title":         "title",
	"ip_address":    "ip_address",
	"platform":      "platform",
	"player":        "player",
	"rating_key":    "rating_key",
	"media_type":    "media_type",
	"location_type": "location_type",
}

// ApproximateDistinctCount returns approximate count of distinct values in a column
// using HyperLogLog. Falls back to exact COUNT(DISTINCT) when unavailable.
func (db *DB) ApproximateDistinctCount(ctx context.Context, column string, filter ApproximateStatsFilter) (int64, bool, error) {
	// Map user input to a safe SQL identifier to prevent SQL injection
	safeColumn, ok := validDistinctColumns[column]
	if !ok {
		return 0, false, fmt.Errorf("invalid column name: %s", column)
	}

	whereClause, args := buildApproximateWhereClause(filter)

	if db.datasketchesAvailable {
		// datasketch_hll(precision, value) is aggregate, datasketch_hll_estimate extracts count
		query := fmt.Sprintf(`
			SELECT COALESCE(datasketch_hll_estimate(datasketch_hll(12, %s)), 0)
			FROM playback_events
			WHERE %s
		`, safeColumn, whereClause)

		var count int64
		err := db.conn.QueryRowContext(ctx, query, args...).Scan(&count)
		if err == nil {
			return count, true, nil
		}
		// Fall through on error
	}

	// Exact fallback
	query := fmt.Sprintf(`
		SELECT COUNT(DISTINCT %s)
		FROM playback_events
		WHERE %s
	`, safeColumn, whereClause)

	var count int64
	err := db.conn.QueryRowContext(ctx, query, args...).Scan(&count)
	return count, false, err
}

// validPercentileColumns maps user-provided column names to safe SQL identifiers
// to prevent SQL injection. Only these predefined numeric columns can be used.
var validPercentileColumns = map[string]string{
	"play_duration":    "play_duration",
	"duration":         "play_duration", // alias for play_duration
	"percent_complete": "percent_complete",
	"paused_counter":   "paused_counter",
}

// ApproximatePercentile returns approximate percentile of a numeric column
// using KLL sketches. Falls back to exact PERCENTILE_CONT when unavailable.
func (db *DB) ApproximatePercentile(ctx context.Context, column string, percentile float64, filter ApproximateStatsFilter) (float64, bool, error) {
	// Map user input to a safe SQL identifier to prevent SQL injection
	safeColumn, ok := validPercentileColumns[column]
	if !ok {
		return 0, false, fmt.Errorf("invalid column name: %s", column)
	}

	if percentile < 0 || percentile > 1 {
		return 0, false, fmt.Errorf("percentile must be between 0 and 1")
	}

	whereClause, args := buildApproximateWhereClause(filter)

	if db.datasketchesAvailable {
		// datasketch_kll(k, value) - k=200 gives ~1.65% error for PMF
		// datasketch_kll_quantile(sketch, rank, inclusive) - rank must be DOUBLE, inclusive=true
		query := fmt.Sprintf(`
			SELECT COALESCE(datasketch_kll_quantile(datasketch_kll(200, CAST(%s AS DOUBLE)), CAST(? AS DOUBLE), true), 0)
			FROM playback_events
			WHERE %s AND %s > 0
		`, safeColumn, whereClause, safeColumn)

		// Add percentile as first arg
		queryArgs := append([]interface{}{percentile}, args...)

		var value float64
		err := db.conn.QueryRowContext(ctx, query, queryArgs...).Scan(&value)
		if err == nil {
			return value, true, nil
		}
		// Fall through on error
	}

	// Exact fallback
	query := fmt.Sprintf(`
		SELECT COALESCE(PERCENTILE_CONT(?) WITHIN GROUP (ORDER BY %s), 0)
		FROM playback_events
		WHERE %s AND %s > 0
	`, safeColumn, whereClause, safeColumn)

	queryArgs := append([]interface{}{percentile}, args...)

	var value float64
	err := db.conn.QueryRowContext(ctx, query, queryArgs...).Scan(&value)
	return value, false, err
}

// buildApproximateWhereClause builds WHERE clause for approximate stats queries
func buildApproximateWhereClause(filter ApproximateStatsFilter) (string, []interface{}) {
	whereClauses := []string{"1=1"}
	args := []interface{}{}

	if filter.StartDate != nil {
		whereClauses = append(whereClauses, "started_at >= ?")
		args = append(args, *filter.StartDate)
	}

	if filter.EndDate != nil {
		whereClauses = append(whereClauses, "started_at <= ?")
		args = append(args, *filter.EndDate)
	}

	if len(filter.Users) > 0 {
		placeholders := make([]string, len(filter.Users))
		for i, user := range filter.Users {
			placeholders[i] = "?"
			args = append(args, user)
		}
		whereClauses = append(whereClauses, fmt.Sprintf("username IN (%s)", join(placeholders, ", ")))
	}

	if len(filter.MediaTypes) > 0 {
		placeholders := make([]string, len(filter.MediaTypes))
		for i, mt := range filter.MediaTypes {
			placeholders[i] = "?"
			args = append(args, mt)
		}
		whereClauses = append(whereClauses, fmt.Sprintf("media_type IN (%s)", join(placeholders, ", ")))
	}

	return join(whereClauses, " AND "), args
}

// GetDistinctUsersApproximate returns approximate count of unique users
// This is a convenience wrapper around ApproximateDistinctCount
func (db *DB) GetDistinctUsersApproximate(ctx context.Context, filter ApproximateStatsFilter) (int64, bool, error) {
	return db.ApproximateDistinctCount(ctx, "username", filter)
}

// GetDistinctTitlesApproximate returns approximate count of unique titles
func (db *DB) GetDistinctTitlesApproximate(ctx context.Context, filter ApproximateStatsFilter) (int64, bool, error) {
	return db.ApproximateDistinctCount(ctx, "title", filter)
}

// GetMedianWatchTimeApproximate returns approximate median watch time in seconds
func (db *DB) GetMedianWatchTimeApproximate(ctx context.Context, filter ApproximateStatsFilter) (float64, bool, error) {
	return db.ApproximatePercentile(ctx, "play_duration", 0.50, filter)
}
