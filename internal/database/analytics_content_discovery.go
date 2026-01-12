// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package database provides data access and analytics functionality for the Cartographus application.
// This file contains content discovery analytics including time-to-first-watch tracking,
// discovery rate calculations, early adopter identification, and stale content detection.
package database

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/tomtom215/cartographus/internal/models"
)

// Default thresholds for content discovery analytics
const (
	// earlyDiscoveryThresholdHours is the hours threshold to count as "early" discovery
	earlyDiscoveryThresholdHours = 24
	// staleContentThresholdDays is days without any watch to count as "stale"
	staleContentThresholdDays = 90
)

// GetContentDiscoveryAnalytics retrieves comprehensive content discovery analytics.
// This includes discovery rates, time-to-first-watch metrics, early adopters,
// stale content identification, and library-level discovery statistics.
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - filter: LocationStatsFilter with date range and user filters
//
// Returns: ContentDiscoveryAnalytics with complete discovery data, or error
func (db *DB) GetContentDiscoveryAnalytics(ctx context.Context, filter LocationStatsFilter) (*models.ContentDiscoveryAnalytics, error) {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	startTime := time.Now()

	// Get summary statistics
	summary, err := db.getContentDiscoverySummary(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get discovery summary: %w", err)
	}

	// Get time bucket distribution
	timeBuckets, err := db.getDiscoveryTimeBuckets(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get time buckets: %w", err)
	}

	// Get early adopters
	earlyAdopters, err := db.getEarlyAdopters(ctx, filter, 20)
	if err != nil {
		return nil, fmt.Errorf("failed to get early adopters: %w", err)
	}

	// Get recently discovered content
	recentlyDiscovered, err := db.getRecentlyDiscoveredContent(ctx, filter, 20)
	if err != nil {
		return nil, fmt.Errorf("failed to get recently discovered: %w", err)
	}

	// Get stale content
	staleContent, err := db.getStaleContent(ctx, filter, 50)
	if err != nil {
		return nil, fmt.Errorf("failed to get stale content: %w", err)
	}

	// Get library statistics
	libraryStats, err := db.getLibraryDiscoveryStats(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get library stats: %w", err)
	}

	// Get discovery trends
	trends, err := db.getDiscoveryTrends(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get discovery trends: %w", err)
	}

	// Build metadata
	metadata := db.buildContentDiscoveryMetadata(ctx, filter, startTime)

	return &models.ContentDiscoveryAnalytics{
		Summary:            *summary,
		TimeBuckets:        timeBuckets,
		EarlyAdopters:      earlyAdopters,
		RecentlyDiscovered: recentlyDiscovered,
		StaleContent:       staleContent,
		LibraryStats:       libraryStats,
		Trends:             trends,
		Metadata:           metadata,
	}, nil
}

// getContentDiscoverySummary retrieves aggregate content discovery statistics.
func (db *DB) getContentDiscoverySummary(ctx context.Context, filter LocationStatsFilter) (*models.ContentDiscoverySummary, error) {
	whereClause, args := buildEngagementWhereClause(filter, "", false)

	query := fmt.Sprintf(`
		WITH content_discovery AS (
			SELECT
				rating_key,
				title,
				added_at,
				MIN(started_at) as first_watched_at,
				COUNT(*) as playback_count
			FROM playback_events
			WHERE %s
				AND rating_key IS NOT NULL
				AND added_at IS NOT NULL
				AND added_at != ''
			GROUP BY rating_key, title, added_at
		),
		discovery_times AS (
			SELECT
				rating_key,
				CASE
					WHEN first_watched_at IS NOT NULL THEN
						EXTRACT(EPOCH FROM (first_watched_at - TRY_CAST(added_at AS TIMESTAMP))) / 3600.0
					ELSE NULL
				END as hours_to_discovery
			FROM content_discovery
		),
		recent_content AS (
			SELECT COUNT(DISTINCT rating_key) as recent_count
			FROM playback_events
			WHERE %s
				AND rating_key IS NOT NULL
				AND added_at IS NOT NULL
				AND TRY_CAST(added_at AS TIMESTAMP) >= CURRENT_TIMESTAMP - INTERVAL 30 days
		),
		recent_discovered AS (
			SELECT COUNT(DISTINCT rating_key) as discovered_count
			FROM playback_events
			WHERE %s
				AND rating_key IS NOT NULL
				AND added_at IS NOT NULL
				AND TRY_CAST(added_at AS TIMESTAMP) >= CURRENT_TIMESTAMP - INTERVAL 30 days
				AND started_at IS NOT NULL
		)
		SELECT
			(SELECT COUNT(*) FROM content_discovery) as total_with_added_at,
			(SELECT COUNT(*) FROM content_discovery WHERE first_watched_at IS NOT NULL) as total_discovered,
			(SELECT COUNT(*) FROM content_discovery WHERE first_watched_at IS NULL) as total_never_watched,
			COALESCE((SELECT AVG(hours_to_discovery) FROM discovery_times WHERE hours_to_discovery IS NOT NULL), 0) as avg_hours,
			COALESCE((SELECT PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY hours_to_discovery)
			          FROM discovery_times WHERE hours_to_discovery IS NOT NULL), 0) as median_hours,
			COALESCE((SELECT COUNT(*) FROM discovery_times
			          WHERE hours_to_discovery IS NOT NULL AND hours_to_discovery <= %d) * 100.0 /
			         NULLIF((SELECT COUNT(*) FROM discovery_times WHERE hours_to_discovery IS NOT NULL), 0), 0) as early_rate,
			COALESCE((SELECT MIN(hours_to_discovery) FROM discovery_times WHERE hours_to_discovery IS NOT NULL AND hours_to_discovery > 0), 0) as fastest_hours,
			COALESCE((SELECT MAX(hours_to_discovery) FROM discovery_times WHERE hours_to_discovery IS NOT NULL) / 24.0, 0) as slowest_days,
			COALESCE((SELECT recent_count FROM recent_content), 0) as recent_additions,
			COALESCE((SELECT discovered_count FROM recent_discovered), 0) as recent_discovered
	`, whereClause, whereClause, whereClause, earlyDiscoveryThresholdHours)

	var summary models.ContentDiscoverySummary
	var avgHours, medianHours, earlyRate, fastestHours, slowestDays sql.NullFloat64

	err := db.conn.QueryRowContext(ctx, query, append(append(args, args...), args...)...).Scan(
		&summary.TotalContentWithAddedAt,
		&summary.TotalDiscovered,
		&summary.TotalNeverWatched,
		&avgHours,
		&medianHours,
		&earlyRate,
		&fastestHours,
		&slowestDays,
		&summary.RecentAdditionsCount,
		&summary.RecentDiscoveredCount,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan summary: %w", err)
	}

	if avgHours.Valid {
		summary.AvgTimeToDiscoveryHours = avgHours.Float64
	}
	if medianHours.Valid {
		summary.MedianTimeToDiscoveryHours = medianHours.Float64
	}
	if earlyRate.Valid {
		summary.EarlyDiscoveryRate = earlyRate.Float64
	}
	if fastestHours.Valid {
		summary.FastestDiscoveryHours = fastestHours.Float64
	}
	if slowestDays.Valid {
		summary.SlowestDiscoveryDays = int(slowestDays.Float64)
	}

	if summary.TotalContentWithAddedAt > 0 {
		summary.OverallDiscoveryRate = float64(summary.TotalDiscovered) / float64(summary.TotalContentWithAddedAt) * 100
	}

	return &summary, nil
}

// getDiscoveryTimeBuckets calculates discovery distribution by time buckets.
func (db *DB) getDiscoveryTimeBuckets(ctx context.Context, filter LocationStatsFilter) ([]models.DiscoveryTimeBucket, error) {
	whereClause, args := buildEngagementWhereClause(filter, "", false)

	query := fmt.Sprintf(`
		WITH content_discovery AS (
			SELECT
				rating_key,
				added_at,
				MIN(started_at) as first_watched_at
			FROM playback_events
			WHERE %s
				AND rating_key IS NOT NULL
				AND added_at IS NOT NULL
				AND added_at != ''
			GROUP BY rating_key, added_at
			HAVING MIN(started_at) IS NOT NULL
		),
		discovery_times AS (
			SELECT
				rating_key,
				EXTRACT(EPOCH FROM (first_watched_at - TRY_CAST(added_at AS TIMESTAMP))) / 3600.0 as hours_to_discovery
			FROM content_discovery
			WHERE TRY_CAST(added_at AS TIMESTAMP) IS NOT NULL
		),
		total_discovered AS (
			SELECT COUNT(*) as total FROM discovery_times WHERE hours_to_discovery IS NOT NULL
		)
		SELECT
			bucket,
			min_hours,
			max_hours,
			content_count,
			CAST(content_count * 100.0 / NULLIF((SELECT total FROM total_discovered), 0) AS DOUBLE) as percentage
		FROM (
			SELECT '0-24h' as bucket, 0 as min_hours, 24 as max_hours,
			       COUNT(*) as content_count
			FROM discovery_times WHERE hours_to_discovery >= 0 AND hours_to_discovery < 24
			UNION ALL
			SELECT '1-7d' as bucket, 24 as min_hours, 168 as max_hours,
			       COUNT(*) as content_count
			FROM discovery_times WHERE hours_to_discovery >= 24 AND hours_to_discovery < 168
			UNION ALL
			SELECT '7-30d' as bucket, 168 as min_hours, 720 as max_hours,
			       COUNT(*) as content_count
			FROM discovery_times WHERE hours_to_discovery >= 168 AND hours_to_discovery < 720
			UNION ALL
			SELECT '30-90d' as bucket, 720 as min_hours, 2160 as max_hours,
			       COUNT(*) as content_count
			FROM discovery_times WHERE hours_to_discovery >= 720 AND hours_to_discovery < 2160
			UNION ALL
			SELECT '90d+' as bucket, 2160 as min_hours, 999999 as max_hours,
			       COUNT(*) as content_count
			FROM discovery_times WHERE hours_to_discovery >= 2160
		) buckets
		ORDER BY min_hours
	`, whereClause)

	rows, err := db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query time buckets: %w", err)
	}
	defer rows.Close()

	var buckets []models.DiscoveryTimeBucket
	for rows.Next() {
		var b models.DiscoveryTimeBucket
		var percentage sql.NullFloat64
		err := rows.Scan(
			&b.Bucket,
			&b.BucketMinHours,
			&b.BucketMaxHours,
			&b.ContentCount,
			&percentage,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan bucket: %w", err)
		}
		if percentage.Valid {
			b.Percentage = percentage.Float64
		}
		buckets = append(buckets, b)
	}

	return buckets, rows.Err()
}

// getEarlyAdopters identifies users who consistently discover new content quickly.
func (db *DB) getEarlyAdopters(ctx context.Context, filter LocationStatsFilter, limit int) ([]models.EarlyAdopter, error) {
	whereClause, args := buildEngagementWhereClause(filter, "", false)

	query := fmt.Sprintf(`
		WITH user_discoveries AS (
			SELECT
				user_id,
				username,
				rating_key,
				library_name,
				added_at,
				MIN(started_at) as first_watched_at,
				EXTRACT(EPOCH FROM (MIN(started_at) - TRY_CAST(added_at AS TIMESTAMP))) / 3600.0 as hours_to_discovery
			FROM playback_events
			WHERE %s
				AND user_id IS NOT NULL
				AND username IS NOT NULL
				AND rating_key IS NOT NULL
				AND added_at IS NOT NULL
				AND added_at != ''
			GROUP BY user_id, username, rating_key, library_name, added_at
		),
		user_stats AS (
			SELECT
				user_id,
				username,
				COUNT(DISTINCT rating_key) as total_discoveries,
				COUNT(DISTINCT CASE WHEN hours_to_discovery <= %d THEN rating_key END) as early_discoveries,
				AVG(hours_to_discovery) as avg_hours_to_discovery,
				MIN(first_watched_at) as first_seen_at,
				MODE() WITHIN GROUP (ORDER BY library_name) as favorite_library
			FROM user_discoveries
			WHERE hours_to_discovery IS NOT NULL
			GROUP BY user_id, username
			HAVING COUNT(DISTINCT rating_key) >= 3
		)
		SELECT
			user_id,
			username,
			early_discoveries,
			total_discoveries,
			CAST(early_discoveries * 100.0 / NULLIF(total_discoveries, 0) AS DOUBLE) as early_rate,
			COALESCE(avg_hours_to_discovery, 0) as avg_hours,
			first_seen_at,
			favorite_library
		FROM user_stats
		ORDER BY early_discoveries DESC, early_rate DESC
		LIMIT ?
	`, whereClause, earlyDiscoveryThresholdHours)

	args = append(args, limit)
	rows, err := db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query early adopters: %w", err)
	}
	defer rows.Close()

	var adopters []models.EarlyAdopter
	for rows.Next() {
		var a models.EarlyAdopter
		var earlyRate, avgHours sql.NullFloat64
		var favoriteLibrary sql.NullString
		err := rows.Scan(
			&a.UserID,
			&a.Username,
			&a.EarlyDiscoveryCount,
			&a.TotalDiscoveries,
			&earlyRate,
			&avgHours,
			&a.FirstSeenAt,
			&favoriteLibrary,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan early adopter: %w", err)
		}
		if earlyRate.Valid {
			a.EarlyDiscoveryRate = earlyRate.Float64
		}
		if avgHours.Valid {
			a.AvgTimeToDiscoveryHours = avgHours.Float64
		}
		if favoriteLibrary.Valid {
			a.FavoriteLibrary = &favoriteLibrary.String
		}
		adopters = append(adopters, a)
	}

	return adopters, rows.Err()
}

// getRecentlyDiscoveredContent retrieves content that was recently watched for the first time.
func (db *DB) getRecentlyDiscoveredContent(ctx context.Context, filter LocationStatsFilter, limit int) ([]models.ContentDiscoveryItem, error) {
	whereClause, args := buildEngagementWhereClause(filter, "", false)

	query := fmt.Sprintf(`
		WITH content_stats AS (
			SELECT
				rating_key,
				title,
				media_type,
				library_name,
				added_at,
				year,
				genres,
				MIN(started_at) as first_watched_at,
				COUNT(*) as playback_count,
				COUNT(DISTINCT user_id) as unique_viewers,
				AVG(percent_complete) as avg_completion
			FROM playback_events
			WHERE %s
				AND rating_key IS NOT NULL
				AND added_at IS NOT NULL
				AND added_at != ''
			GROUP BY rating_key, title, media_type, library_name, added_at, year, genres
			HAVING MIN(started_at) IS NOT NULL
		)
		SELECT
			rating_key,
			title,
			media_type,
			library_name,
			TRY_CAST(added_at AS TIMESTAMP) as added_at,
			first_watched_at,
			EXTRACT(EPOCH FROM (first_watched_at - TRY_CAST(added_at AS TIMESTAMP))) / 3600.0 as hours_to_discovery,
			playback_count,
			unique_viewers,
			COALESCE(avg_completion, 0) as avg_completion,
			year,
			genres,
			CASE
				WHEN EXTRACT(EPOCH FROM (first_watched_at - TRY_CAST(added_at AS TIMESTAMP))) / 3600.0 <= 24 THEN 'fast'
				WHEN EXTRACT(EPOCH FROM (first_watched_at - TRY_CAST(added_at AS TIMESTAMP))) / 3600.0 <= 168 THEN 'medium'
				ELSE 'slow'
			END as discovery_velocity
		FROM content_stats
		WHERE TRY_CAST(added_at AS TIMESTAMP) IS NOT NULL
		ORDER BY first_watched_at DESC
		LIMIT ?
	`, whereClause)

	args = append(args, limit)
	rows, err := db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query recently discovered: %w", err)
	}
	defer rows.Close()

	var items []models.ContentDiscoveryItem
	for rows.Next() {
		var item models.ContentDiscoveryItem
		var addedAt, firstWatched sql.NullTime
		var hoursToDiscovery sql.NullFloat64
		var year sql.NullInt64
		var genres sql.NullString

		err := rows.Scan(
			&item.RatingKey,
			&item.Title,
			&item.MediaType,
			&item.LibraryName,
			&addedAt,
			&firstWatched,
			&hoursToDiscovery,
			&item.TotalPlaybacks,
			&item.UniqueViewers,
			&item.AvgCompletion,
			&year,
			&genres,
			&item.DiscoveryVelocity,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan content item: %w", err)
		}

		if addedAt.Valid {
			item.AddedAt = addedAt.Time
		}
		if firstWatched.Valid {
			item.FirstWatchedAt = &firstWatched.Time
		}
		if hoursToDiscovery.Valid {
			item.TimeToFirstWatchHours = &hoursToDiscovery.Float64
		}
		if year.Valid {
			y := int(year.Int64)
			item.Year = &y
		}
		if genres.Valid {
			item.Genres = &genres.String
		}

		items = append(items, item)
	}

	return items, rows.Err()
}

// getStaleContent retrieves content that was added but never watched.
func (db *DB) getStaleContent(ctx context.Context, filter LocationStatsFilter, limit int) ([]models.StaleContent, error) {
	whereClause, args := buildEngagementWhereClause(filter, "", false)

	// Find content with added_at but no playback records or only partial records
	query := fmt.Sprintf(`
		WITH all_content AS (
			SELECT DISTINCT
				rating_key,
				title,
				media_type,
				library_name,
				added_at,
				year,
				genres,
				content_rating
			FROM playback_events
			WHERE %s
				AND rating_key IS NOT NULL
				AND added_at IS NOT NULL
				AND added_at != ''
				AND TRY_CAST(added_at AS TIMESTAMP) IS NOT NULL
		),
		watched_content AS (
			SELECT DISTINCT rating_key
			FROM playback_events
			WHERE %s
				AND rating_key IS NOT NULL
				AND started_at IS NOT NULL
				AND percent_complete > 5
		)
		SELECT
			a.rating_key,
			a.title,
			a.media_type,
			a.library_name,
			TRY_CAST(a.added_at AS TIMESTAMP) as added_at,
			EXTRACT(DAY FROM (CURRENT_TIMESTAMP - TRY_CAST(a.added_at AS TIMESTAMP))) as days_since_added,
			a.year,
			a.genres,
			a.content_rating
		FROM all_content a
		LEFT JOIN watched_content w ON a.rating_key = w.rating_key
		WHERE w.rating_key IS NULL
			AND TRY_CAST(a.added_at AS TIMESTAMP) <= CURRENT_TIMESTAMP - INTERVAL %d days
		ORDER BY days_since_added DESC
		LIMIT ?
	`, whereClause, whereClause, staleContentThresholdDays)

	args = append(args, args...)
	args = append(args, limit)
	rows, err := db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query stale content: %w", err)
	}
	defer rows.Close()

	var content []models.StaleContent
	for rows.Next() {
		var item models.StaleContent
		var addedAt sql.NullTime
		var daysSinceAdded sql.NullFloat64
		var year sql.NullInt64
		var genres, contentRating sql.NullString

		err := rows.Scan(
			&item.RatingKey,
			&item.Title,
			&item.MediaType,
			&item.LibraryName,
			&addedAt,
			&daysSinceAdded,
			&year,
			&genres,
			&contentRating,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan stale content: %w", err)
		}

		if addedAt.Valid {
			item.AddedAt = addedAt.Time
		}
		if daysSinceAdded.Valid {
			item.DaysSinceAdded = int(daysSinceAdded.Float64)
		}
		if year.Valid {
			y := int(year.Int64)
			item.Year = &y
		}
		if genres.Valid {
			item.Genres = &genres.String
		}
		if contentRating.Valid {
			item.ContentRating = &contentRating.String
		}

		content = append(content, item)
	}

	return content, rows.Err()
}

// getLibraryDiscoveryStats retrieves discovery statistics per library.
func (db *DB) getLibraryDiscoveryStats(ctx context.Context, filter LocationStatsFilter) ([]models.LibraryDiscoveryStats, error) {
	whereClause, args := buildEngagementWhereClause(filter, "", false)

	query := fmt.Sprintf(`
		WITH library_content AS (
			SELECT
				library_name,
				rating_key,
				added_at,
				MIN(started_at) as first_watched_at
			FROM playback_events
			WHERE %s
				AND library_name IS NOT NULL
				AND library_name != ''
				AND rating_key IS NOT NULL
				AND added_at IS NOT NULL
				AND added_at != ''
			GROUP BY library_name, rating_key, added_at
		),
		library_stats AS (
			SELECT
				library_name,
				COUNT(DISTINCT rating_key) as total_items,
				COUNT(DISTINCT CASE WHEN first_watched_at IS NOT NULL THEN rating_key END) as watched_items,
				AVG(CASE
					WHEN first_watched_at IS NOT NULL AND TRY_CAST(added_at AS TIMESTAMP) IS NOT NULL
					THEN EXTRACT(EPOCH FROM (first_watched_at - TRY_CAST(added_at AS TIMESTAMP))) / 3600.0
					END) as avg_hours,
				PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY
					CASE
						WHEN first_watched_at IS NOT NULL AND TRY_CAST(added_at AS TIMESTAMP) IS NOT NULL
						THEN EXTRACT(EPOCH FROM (first_watched_at - TRY_CAST(added_at AS TIMESTAMP))) / 3600.0
						END
				) as median_hours,
				COUNT(DISTINCT CASE
					WHEN first_watched_at IS NOT NULL
						AND TRY_CAST(added_at AS TIMESTAMP) IS NOT NULL
						AND EXTRACT(EPOCH FROM (first_watched_at - TRY_CAST(added_at AS TIMESTAMP))) / 3600.0 <= %d
					THEN rating_key END) as early_items
			FROM library_content
			GROUP BY library_name
		)
		SELECT
			library_name,
			total_items,
			watched_items,
			total_items - watched_items as unwatched_items,
			CAST(watched_items * 100.0 / NULLIF(total_items, 0) AS DOUBLE) as discovery_rate,
			COALESCE(avg_hours, 0) as avg_hours,
			COALESCE(median_hours, 0) as median_hours,
			CAST(early_items * 100.0 / NULLIF(watched_items, 0) AS DOUBLE) as early_rate
		FROM library_stats
		ORDER BY total_items DESC
	`, whereClause, earlyDiscoveryThresholdHours)

	rows, err := db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query library stats: %w", err)
	}
	defer rows.Close()

	var stats []models.LibraryDiscoveryStats
	for rows.Next() {
		var s models.LibraryDiscoveryStats
		var discoveryRate, avgHours, medianHours, earlyRate sql.NullFloat64

		err := rows.Scan(
			&s.LibraryName,
			&s.TotalItems,
			&s.WatchedItems,
			&s.UnwatchedItems,
			&discoveryRate,
			&avgHours,
			&medianHours,
			&earlyRate,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan library stat: %w", err)
		}

		if discoveryRate.Valid {
			s.DiscoveryRate = discoveryRate.Float64
		}
		if avgHours.Valid {
			s.AvgTimeToDiscoveryHours = avgHours.Float64
		}
		if medianHours.Valid {
			s.MedianTimeToDiscoveryHours = medianHours.Float64
		}
		if earlyRate.Valid {
			s.EarlyDiscoveryRate = earlyRate.Float64
		}

		stats = append(stats, s)
	}

	return stats, rows.Err()
}

// getDiscoveryTrends retrieves content discovery trends over time.
func (db *DB) getDiscoveryTrends(ctx context.Context, filter LocationStatsFilter) ([]models.DiscoveryTrend, error) {
	whereClause, args := buildEngagementWhereClause(filter, "", false)

	// Determine appropriate interval
	interval := determineTrendInterval(filter)

	query := fmt.Sprintf(`
		WITH content_per_period AS (
			SELECT
				DATE_TRUNC('%s', TRY_CAST(added_at AS TIMESTAMP)) as period,
				rating_key,
				added_at,
				MIN(started_at) as first_watched_at
			FROM playback_events
			WHERE %s
				AND rating_key IS NOT NULL
				AND added_at IS NOT NULL
				AND added_at != ''
				AND TRY_CAST(added_at AS TIMESTAMP) IS NOT NULL
			GROUP BY DATE_TRUNC('%s', TRY_CAST(added_at AS TIMESTAMP)), rating_key, added_at
		)
		SELECT
			CAST(period AS VARCHAR) as date,
			COUNT(DISTINCT rating_key) as content_added,
			COUNT(DISTINCT CASE WHEN first_watched_at IS NOT NULL THEN rating_key END) as content_discovered,
			CAST(COUNT(DISTINCT CASE WHEN first_watched_at IS NOT NULL THEN rating_key END) * 100.0 /
				NULLIF(COUNT(DISTINCT rating_key), 0) AS DOUBLE) as discovery_rate,
			COALESCE(AVG(
				CASE WHEN first_watched_at IS NOT NULL AND TRY_CAST(added_at AS TIMESTAMP) IS NOT NULL
				THEN EXTRACT(EPOCH FROM (first_watched_at - TRY_CAST(added_at AS TIMESTAMP))) / 3600.0
				END
			), 0) as avg_hours_to_discovery
		FROM content_per_period
		WHERE period IS NOT NULL
		GROUP BY period
		ORDER BY period DESC
		LIMIT 50
	`, interval, whereClause, interval)

	rows, err := db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query discovery trends: %w", err)
	}
	defer rows.Close()

	var trends []models.DiscoveryTrend
	for rows.Next() {
		var t models.DiscoveryTrend
		var discoveryRate, avgHours sql.NullFloat64

		err := rows.Scan(
			&t.Date,
			&t.ContentAdded,
			&t.ContentDiscovered,
			&discoveryRate,
			&avgHours,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan trend: %w", err)
		}

		if discoveryRate.Valid {
			t.DiscoveryRate = discoveryRate.Float64
		}
		if avgHours.Valid {
			t.AvgTimeToDiscoveryHours = avgHours.Float64
		}

		trends = append(trends, t)
	}

	return trends, rows.Err()
}

// buildContentDiscoveryMetadata builds metadata for the content discovery response.
func (db *DB) buildContentDiscoveryMetadata(ctx context.Context, filter LocationStatsFilter, startTime time.Time) models.ContentDiscoveryMetadata {
	metadata := models.ContentDiscoveryMetadata{
		ExecutionTimeMS:              time.Since(startTime).Milliseconds(),
		EarlyDiscoveryThresholdHours: earlyDiscoveryThresholdHours,
		StaleContentThresholdDays:    staleContentThresholdDays,
	}

	// Set date range
	if filter.StartDate != nil {
		metadata.DataRangeStart = *filter.StartDate
	}
	if filter.EndDate != nil {
		metadata.DataRangeEnd = *filter.EndDate
	} else {
		metadata.DataRangeEnd = time.Now()
	}

	// Generate query hash for caching/auditing
	hashInput := fmt.Sprintf("content_discovery:%v:%v", metadata.DataRangeStart, metadata.DataRangeEnd)
	hash := sha256.Sum256([]byte(hashInput))
	metadata.QueryHash = hex.EncodeToString(hash[:8])

	// Get event and content counts
	whereClause, args := buildEngagementWhereClause(filter, "", false)
	countQuery := fmt.Sprintf(`
		SELECT
			COUNT(*) as event_count,
			COUNT(DISTINCT rating_key) as content_count
		FROM playback_events
		WHERE %s
			AND rating_key IS NOT NULL
	`, whereClause)

	// Metadata counts are best-effort; errors don't affect the main response
	err := db.conn.QueryRowContext(ctx, countQuery, args...).Scan(
		&metadata.TotalEventsAnalyzed,
		&metadata.UniqueContentAnalyzed,
	)
	if err != nil {
		// Log but don't fail - metadata is supplementary
		metadata.TotalEventsAnalyzed = 0
		metadata.UniqueContentAnalyzed = 0
	}

	return metadata
}
