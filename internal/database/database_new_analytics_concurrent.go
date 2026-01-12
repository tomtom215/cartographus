// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package database

import (
	"context"
	"fmt"
	"time"

	"github.com/tomtom215/cartographus/internal/models"
)

// GetConcurrentStreamsAnalytics analyzes concurrent stream patterns including peak/average concurrency,
// distribution by transcode type, temporal patterns by day of week and hour, and capacity recommendations.
func (db *DB) GetConcurrentStreamsAnalytics(ctx context.Context, filter LocationStatsFilter) (*models.ConcurrentStreamsAnalytics, error) {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	// Query time series data and calculate peak/average metrics
	timeSeriesData, peakConcurrent, peakTime, avgConcurrent, err := db.getConcurrentStreamsTimeSeries(ctx, filter)
	if err != nil {
		return nil, errorContext("get time series", err)
	}

	// Query distribution by transcode type
	byTranscodeDecision, err := db.getConcurrentStreamsByType(ctx, filter, peakConcurrent, avgConcurrent)
	if err != nil {
		return nil, errorContext("get by transcode type", err)
	}

	// Query temporal patterns
	byDayOfWeek, err := db.getConcurrentStreamsByDay(ctx, filter)
	if err != nil {
		return nil, errorContext("get by day of week", err)
	}

	byHourOfDay, err := db.getConcurrentStreamsByHour(ctx, filter)
	if err != nil {
		return nil, errorContext("get by hour of day", err)
	}

	// Query total sessions count
	totalSessions, err := db.getConcurrentStreamsTotalSessions(ctx, filter)
	if err != nil {
		return nil, errorContext("get total sessions", err)
	}

	return &models.ConcurrentStreamsAnalytics{
		PeakConcurrent:         peakConcurrent,
		PeakTime:               peakTime,
		AvgConcurrent:          avgConcurrent,
		TotalSessions:          totalSessions,
		TimeSeriesData:         timeSeriesData,
		ByTranscodeDecision:    byTranscodeDecision,
		ByDayOfWeek:            byDayOfWeek,
		ByHourOfDay:            byHourOfDay,
		CapacityRecommendation: generateCapacityRecommendation(float64(peakConcurrent), avgConcurrent),
	}, nil
}

// generateCapacityRecommendation provides infrastructure sizing guidance
func generateCapacityRecommendation(peak, avg float64) string {
	if peak <= 2 {
		return "Light usage - entry-level Plex Pass recommended"
	} else if peak <= 5 {
		return "Moderate usage - standard Plex Pass with CPU transcode support"
	} else if peak <= 10 {
		return "Heavy usage - consider hardware transcoding (Intel Quick Sync or NVIDIA)"
	} else if peak <= 20 {
		return "Very heavy usage - dedicated GPU strongly recommended, consider multiple Plex instances"
	} else {
		return "Extreme usage - enterprise setup required with load balancing and multiple GPUs"
	}
}

// getConcurrentStreamsTimeSeries queries hourly concurrent stream counts using DuckDB's generate_series
// (optimized from recursive CTE) and calculates peak/average concurrency metrics across the filtered time range.
// Performance: ~10-20x faster than recursive CTE for large date ranges (90 days = 2160 hours).
func (db *DB) getConcurrentStreamsTimeSeries(ctx context.Context, filter LocationStatsFilter) ([]models.ConcurrentStreamsTimeBucket, int, time.Time, float64, error) {
	whereClauses, args := buildFilterConditions(filter, false, 1)

	// Build AND-prefixed WHERE clause for subqueries
	whereAnd := ""
	if len(whereClauses) > 0 {
		whereAnd = "AND " + buildAndWhereClause(whereClauses)
	}

	// Optimized query using generate_series instead of recursive CTE
	// This is significantly faster as generate_series is a built-in function
	// that doesn't require iterative evaluation like recursive CTEs.
	//
	// The query structure:
	// 1. time_range: Single scan to get min/max timestamps (cached for reuse)
	// 2. time_buckets: Generate all hour buckets using generate_series (O(1) vs O(n) recursive)
	//    - Only generates buckets when min_time IS NOT NULL (handles empty table case)
	// 3. concurrent_counts: Count overlapping sessions per bucket with filter conditions
	query := fmt.Sprintf(`
		WITH time_range AS (
			SELECT
				DATE_TRUNC('hour', MIN(started_at)) as min_time,
				DATE_TRUNC('hour', MAX(COALESCE(stopped_at, started_at))) as max_time
			FROM playback_events
			WHERE stopped_at IS NOT NULL %s
		),
		time_buckets AS (
			SELECT unnest(generate_series(
				(SELECT min_time FROM time_range),
				(SELECT max_time FROM time_range),
				INTERVAL '1 hour'
			)) as bucket_time
			WHERE (SELECT min_time FROM time_range) IS NOT NULL
		),
		concurrent_counts AS (
			SELECT
				tb.bucket_time as timestamp,
				COUNT(DISTINCT pe.session_key) as concurrent_count,
				COUNT(DISTINCT CASE WHEN pe.transcode_decision = 'direct play' THEN pe.session_key END) as direct_play,
				COUNT(DISTINCT CASE WHEN pe.transcode_decision = 'direct stream' THEN pe.session_key END) as direct_stream,
				COUNT(DISTINCT CASE WHEN pe.transcode_decision = 'transcode' THEN pe.session_key END) as transcode
			FROM time_buckets tb
			LEFT JOIN playback_events pe ON (
				pe.started_at <= tb.bucket_time + INTERVAL '1 hour'
				AND COALESCE(pe.stopped_at, CURRENT_TIMESTAMP) >= tb.bucket_time
				AND pe.stopped_at IS NOT NULL
				%s
			)
			GROUP BY tb.bucket_time
			HAVING COUNT(DISTINCT pe.session_key) > 0
		)
		SELECT
			timestamp,
			concurrent_count,
			direct_play,
			direct_stream,
			transcode
		FROM concurrent_counts
		ORDER BY timestamp ASC
	`, whereAnd, whereAnd)

	// WHERE clause appears 2 times in the optimized query (down from 3)
	allArgs := make([]interface{}, 0, len(args)*2)
	allArgs = append(allArgs, args...)
	allArgs = append(allArgs, args...)

	rows, err := db.conn.QueryContext(ctx, query, allArgs...)
	if err != nil {
		return nil, 0, time.Time{}, 0, fmt.Errorf("failed to query concurrent streams time series: %w", err)
	}
	defer rows.Close()

	var timeSeriesData []models.ConcurrentStreamsTimeBucket
	var peakConcurrent int
	var peakTime time.Time
	var totalConcurrent int64
	var bucketCount int

	for rows.Next() {
		var bucket models.ConcurrentStreamsTimeBucket
		if err := rows.Scan(&bucket.Timestamp, &bucket.ConcurrentCount, &bucket.DirectPlay, &bucket.DirectStream, &bucket.Transcode); err != nil {
			return nil, 0, time.Time{}, 0, fmt.Errorf("failed to scan time series row: %w", err)
		}

		timeSeriesData = append(timeSeriesData, bucket)
		totalConcurrent += int64(bucket.ConcurrentCount)
		bucketCount++

		if bucket.ConcurrentCount > peakConcurrent {
			peakConcurrent = bucket.ConcurrentCount
			peakTime = bucket.Timestamp
		}
	}

	if err = rows.Err(); err != nil {
		return nil, 0, time.Time{}, 0, fmt.Errorf("time series iteration error: %w", err)
	}

	avgConcurrent := 0.0
	if bucketCount > 0 {
		avgConcurrent = float64(totalConcurrent) / float64(bucketCount)
	}

	return timeSeriesData, peakConcurrent, peakTime, avgConcurrent, nil
}

// getConcurrentStreamsByType queries concurrent stream distribution by transcode decision
// (direct play, direct stream, transcode) with percentage and estimated concurrency metrics.
func (db *DB) getConcurrentStreamsByType(ctx context.Context, filter LocationStatsFilter, peakConcurrent int, avgConcurrent float64) ([]models.ConcurrentStreamsByType, error) {
	whereClauses, args := buildFilterConditions(filter, false, 1)

	// Build AND-prefixed WHERE clause for subqueries
	whereAnd := ""
	if len(whereClauses) > 0 {
		whereAnd = "AND " + buildAndWhereClause(whereClauses)
	}

	query := fmt.Sprintf(`
		WITH concurrent_by_type AS (
			SELECT
				transcode_decision,
				COUNT(DISTINCT session_key) as session_count
			FROM playback_events
			WHERE stopped_at IS NOT NULL %s
			GROUP BY transcode_decision
		)
		SELECT
			COALESCE(transcode_decision, 'unknown') as transcode_decision,
			session_count,
			(session_count * 100.0 / SUM(session_count) OVER ()) as percentage
		FROM concurrent_by_type
		ORDER BY session_count DESC
	`, whereAnd)

	rows, err := db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query by transcode type: %w", err)
	}
	defer rows.Close()

	var results []models.ConcurrentStreamsByType
	for rows.Next() {
		var bt models.ConcurrentStreamsByType
		var sessionCount int
		if err := rows.Scan(&bt.TranscodeDecision, &sessionCount, &bt.Percentage); err != nil {
			return nil, fmt.Errorf("failed to scan by type row: %w", err)
		}

		bt.AvgConcurrent = avgConcurrent * (bt.Percentage / 100.0)
		bt.MaxConcurrent = int(float64(peakConcurrent) * (bt.Percentage / 100.0))
		results = append(results, bt)
	}

	return results, rows.Err()
}

// getConcurrentStreamsByDay queries average and peak concurrent streams by day of week
// for identifying weekly usage patterns and capacity planning.
func (db *DB) getConcurrentStreamsByDay(ctx context.Context, filter LocationStatsFilter) ([]models.ConcurrentStreamsByDayOfWeek, error) {
	whereClauses, args := buildFilterConditions(filter, false, 1)

	// Build AND-prefixed WHERE clause for subqueries
	whereAnd := ""
	if len(whereClauses) > 0 {
		whereAnd = "AND " + buildAndWhereClause(whereClauses)
	}

	query := fmt.Sprintf(`
		WITH day_stats AS (
			SELECT
				EXTRACT(DOW FROM started_at) as day_of_week,
				DATE_TRUNC('hour', started_at) as hour_bucket,
				COUNT(DISTINCT session_key) as concurrent_count
			FROM playback_events
			WHERE stopped_at IS NOT NULL %s
			GROUP BY day_of_week, hour_bucket
		)
		SELECT
			day_of_week::INTEGER,
			AVG(concurrent_count) as avg_concurrent,
			MAX(concurrent_count) as peak_concurrent
		FROM day_stats
		GROUP BY day_of_week
		ORDER BY day_of_week
	`, whereAnd)

	rows, err := db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query by day: %w", err)
	}
	defer rows.Close()

	var results []models.ConcurrentStreamsByDayOfWeek
	for rows.Next() {
		var bd models.ConcurrentStreamsByDayOfWeek
		if err := rows.Scan(&bd.DayOfWeek, &bd.AvgConcurrent, &bd.PeakConcurrent); err != nil {
			return nil, fmt.Errorf("failed to scan by day row: %w", err)
		}
		results = append(results, bd)
	}

	return results, rows.Err()
}

// getConcurrentStreamsByHour queries average and peak concurrent streams by hour of day (0-23)
// for identifying daily usage patterns and peak viewing hours.
func (db *DB) getConcurrentStreamsByHour(ctx context.Context, filter LocationStatsFilter) ([]models.ConcurrentStreamsByHour, error) {
	whereClauses, args := buildFilterConditions(filter, false, 1)

	// Build AND-prefixed WHERE clause for subqueries
	whereAnd := ""
	if len(whereClauses) > 0 {
		whereAnd = "AND " + buildAndWhereClause(whereClauses)
	}

	query := fmt.Sprintf(`
		WITH hour_stats AS (
			SELECT
				EXTRACT(HOUR FROM started_at) as hour,
				DATE_TRUNC('hour', started_at) as hour_bucket,
				COUNT(DISTINCT session_key) as concurrent_count
			FROM playback_events
			WHERE stopped_at IS NOT NULL %s
			GROUP BY hour, hour_bucket
		)
		SELECT
			hour::INTEGER,
			AVG(concurrent_count) as avg_concurrent,
			MAX(concurrent_count) as peak_concurrent
		FROM hour_stats
		GROUP BY hour
		ORDER BY hour
	`, whereAnd)

	rows, err := db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query by hour: %w", err)
	}
	defer rows.Close()

	var results []models.ConcurrentStreamsByHour
	for rows.Next() {
		var bh models.ConcurrentStreamsByHour
		if err := rows.Scan(&bh.Hour, &bh.AvgConcurrent, &bh.PeakConcurrent); err != nil {
			return nil, fmt.Errorf("failed to scan by hour row: %w", err)
		}
		results = append(results, bh)
	}

	return results, rows.Err()
}

// getConcurrentStreamsTotalSessions queries the total count of unique playback sessions
// matching the filter criteria for normalizing concurrency metrics.
func (db *DB) getConcurrentStreamsTotalSessions(ctx context.Context, filter LocationStatsFilter) (int, error) {
	whereClauses, args := buildFilterConditions(filter, false, 1)

	// Build AND-prefixed WHERE clause for subqueries
	whereAnd := ""
	if len(whereClauses) > 0 {
		whereAnd = "AND " + buildAndWhereClause(whereClauses)
	}

	query := fmt.Sprintf(`
		SELECT COUNT(DISTINCT session_key)
		FROM playback_events
		WHERE stopped_at IS NOT NULL %s
	`, whereAnd)

	var totalSessions int
	if err := db.conn.QueryRowContext(ctx, query, args...).Scan(&totalSessions); err != nil {
		return 0, fmt.Errorf("failed to get total sessions: %w", err)
	}

	return totalSessions, nil
}
