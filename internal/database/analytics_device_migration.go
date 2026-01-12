// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package database provides data access and analytics functionality for the Cartographus application.
// This file contains device migration tracking analytics including platform adoption, user device
// profiles, and migration pattern detection.
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

// migrationWindowDays is the default window for detecting platform switches.
// A user switching to a new platform and not returning to the old one within this
// window is considered a "permanent" migration.
const migrationWindowDays = 30

// GetDeviceMigrationAnalytics retrieves comprehensive device migration analytics.
// This includes user device profiles, platform transitions, adoption trends, and
// migration detection using LAG() window functions.
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - filter: LocationStatsFilter with date range and user filters
//
// Returns: DeviceMigrationAnalytics with complete migration data, or error
func (db *DB) GetDeviceMigrationAnalytics(ctx context.Context, filter LocationStatsFilter) (*models.DeviceMigrationAnalytics, error) {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	startTime := time.Now()

	// Get summary statistics
	summary, err := db.getDeviceMigrationSummary(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get migration summary: %w", err)
	}

	// Get top user profiles (users with most interesting device patterns)
	topProfiles, err := db.getTopUserDeviceProfiles(ctx, filter, 20)
	if err != nil {
		return nil, fmt.Errorf("failed to get user device profiles: %w", err)
	}

	// Get recent migrations
	recentMigrations, err := db.getRecentMigrations(ctx, filter, 50)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent migrations: %w", err)
	}

	// Get platform adoption trends
	adoptionTrends, err := db.getPlatformAdoptionTrends(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get adoption trends: %w", err)
	}

	// Get common transitions
	commonTransitions, err := db.getCommonPlatformTransitions(ctx, filter, 10)
	if err != nil {
		return nil, fmt.Errorf("failed to get common transitions: %w", err)
	}

	// Get current platform distribution
	platformDistribution, err := db.GetPlatformDistribution(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get platform distribution: %w", err)
	}

	// Build metadata
	metadata := db.buildDeviceMigrationMetadata(ctx, filter, startTime)

	return &models.DeviceMigrationAnalytics{
		Summary:              *summary,
		TopUserProfiles:      topProfiles,
		RecentMigrations:     recentMigrations,
		AdoptionTrends:       adoptionTrends,
		CommonTransitions:    commonTransitions,
		PlatformDistribution: platformDistribution,
		Metadata:             metadata,
	}, nil
}

// getDeviceMigrationSummary retrieves aggregate statistics about device usage and migrations.
func (db *DB) getDeviceMigrationSummary(ctx context.Context, filter LocationStatsFilter) (*models.DeviceMigrationSummary, error) {
	whereClause, args := buildEngagementWhereClause(filter, "", false)

	query := fmt.Sprintf(`
		WITH user_platforms AS (
			SELECT
				user_id,
				COUNT(DISTINCT platform) as platform_count
			FROM playback_events
			WHERE %s
				AND platform IS NOT NULL
				AND platform != ''
			GROUP BY user_id
		),
		platform_usage AS (
			SELECT
				platform,
				COUNT(*) as session_count,
				COUNT(DISTINCT user_id) as user_count
			FROM playback_events
			WHERE %s
				AND platform IS NOT NULL
				AND platform != ''
			GROUP BY platform
		),
		primary_platforms AS (
			SELECT
				platform,
				COUNT(*) as primary_count
			FROM (
				SELECT
					user_id,
					platform,
					ROW_NUMBER() OVER (PARTITION BY user_id ORDER BY COUNT(*) DESC) as rn
				FROM playback_events
				WHERE %s
					AND platform IS NOT NULL
					AND platform != ''
				GROUP BY user_id, platform
			) ranked
			WHERE rn = 1
			GROUP BY platform
		),
		platform_growth AS (
			SELECT
				platform,
				COUNT(DISTINCT user_id) as recent_users
			FROM playback_events
			WHERE %s
				AND platform IS NOT NULL
				AND platform != ''
				AND started_at >= CURRENT_TIMESTAMP - INTERVAL 30 days
			GROUP BY platform
			ORDER BY recent_users DESC
			LIMIT 1
		)
		SELECT
			(SELECT COUNT(*) FROM user_platforms) as total_users,
			(SELECT COUNT(*) FROM user_platforms WHERE platform_count >= 2) as multi_device_users,
			(SELECT COALESCE(AVG(platform_count), 0) FROM user_platforms) as avg_platforms,
			(SELECT platform FROM primary_platforms ORDER BY primary_count DESC LIMIT 1) as most_common_primary,
			(SELECT platform FROM platform_growth LIMIT 1) as fastest_growing
	`, whereClause, whereClause, whereClause, whereClause)

	var summary models.DeviceMigrationSummary
	var mostCommonPrimary, fastestGrowing sql.NullString
	var avgPlatforms float64

	err := db.conn.QueryRowContext(ctx, query, append(append(append(args, args...), args...), args...)...).Scan(
		&summary.TotalUsers,
		&summary.MultiDeviceUsers,
		&avgPlatforms,
		&mostCommonPrimary,
		&fastestGrowing,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan summary: %w", err)
	}

	summary.AvgPlatformsPerUser = avgPlatforms
	if summary.TotalUsers > 0 {
		summary.MultiDevicePercentage = float64(summary.MultiDeviceUsers) / float64(summary.TotalUsers) * 100
	}
	if mostCommonPrimary.Valid {
		summary.MostCommonPrimaryPlatform = mostCommonPrimary.String
	}
	if fastestGrowing.Valid {
		summary.FastestGrowingPlatform = fastestGrowing.String
	}

	// Get total migrations count
	migrationCountQuery := fmt.Sprintf(`
		WITH user_sessions AS (
			SELECT
				user_id,
				platform,
				started_at,
				LAG(platform) OVER (PARTITION BY user_id ORDER BY started_at) as prev_platform
			FROM playback_events
			WHERE %s
				AND platform IS NOT NULL
				AND platform != ''
		)
		SELECT COUNT(*) FROM user_sessions
		WHERE prev_platform IS NOT NULL AND prev_platform != platform
	`, whereClause)

	err = db.conn.QueryRowContext(ctx, migrationCountQuery, args...).Scan(&summary.TotalMigrations)
	if err != nil {
		return nil, fmt.Errorf("failed to get migration count: %w", err)
	}

	return &summary, nil
}

// getTopUserDeviceProfiles retrieves detailed device profiles for users with notable patterns.
func (db *DB) getTopUserDeviceProfiles(ctx context.Context, filter LocationStatsFilter, limit int) ([]models.UserDeviceProfile, error) {
	whereClause, args := buildEngagementWhereClause(filter, "", false)

	// Get users with most platforms or migrations
	query := fmt.Sprintf(`
		WITH user_platform_stats AS (
			SELECT
				user_id,
				username,
				platform,
				COUNT(*) as session_count,
				CAST(COALESCE(SUM(play_duration), 0) / 60.0 AS DOUBLE) as watch_time_minutes,
				MIN(started_at) as first_used,
				MAX(started_at) as last_used
			FROM playback_events
			WHERE %s
				AND platform IS NOT NULL
				AND platform != ''
				AND username IS NOT NULL
			GROUP BY user_id, username, platform
		),
		user_summaries AS (
			SELECT
				user_id,
				username,
				COUNT(DISTINCT platform) as platform_count,
				SUM(session_count) as total_sessions,
				MIN(first_used) as first_seen,
				MAX(last_used) as last_seen
			FROM user_platform_stats
			GROUP BY user_id, username
			HAVING COUNT(DISTINCT platform) >= 1
		)
		SELECT
			u.user_id,
			u.username,
			u.platform_count,
			u.total_sessions,
			u.first_seen,
			u.last_seen
		FROM user_summaries u
		ORDER BY u.platform_count DESC, u.total_sessions DESC
		LIMIT ?
	`, whereClause)

	args = append(args, limit)
	rows, err := db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query user profiles: %w", err)
	}
	defer rows.Close()

	var profiles []models.UserDeviceProfile
	for rows.Next() {
		var profile models.UserDeviceProfile
		err := rows.Scan(
			&profile.UserID,
			&profile.Username,
			&profile.TotalPlatformsUsed,
			&profile.TotalSessions,
			&profile.FirstSeenAt,
			&profile.LastSeenAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user profile: %w", err)
		}

		profile.IsMultiDevice = profile.TotalPlatformsUsed >= 2
		profile.DaysSinceFirstSeen = int(time.Since(profile.FirstSeenAt).Hours() / 24)
		profile.DaysSinceLastSeen = int(time.Since(profile.LastSeenAt).Hours() / 24)

		profiles = append(profiles, profile)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating user profiles: %w", err)
	}

	// Populate platform history for each user
	for i := range profiles {
		history, err := db.getUserPlatformHistory(ctx, filter, profiles[i].UserID)
		if err != nil {
			return nil, fmt.Errorf("failed to get platform history for user %d: %w", profiles[i].UserID, err)
		}
		profiles[i].PlatformHistory = history

		// Determine primary platform
		if len(history) > 0 {
			profiles[i].PrimaryPlatform = history[0].Platform
			profiles[i].PrimaryPlatformPercentage = history[0].Percentage
		}

		// Count migrations for this user
		migrations, err := db.countUserMigrations(ctx, filter, profiles[i].UserID)
		if err != nil {
			return nil, fmt.Errorf("failed to count migrations for user %d: %w", profiles[i].UserID, err)
		}
		profiles[i].TotalMigrations = migrations
	}

	return profiles, nil
}

// getUserPlatformHistory retrieves a user's platform usage history sorted by session count.
func (db *DB) getUserPlatformHistory(ctx context.Context, filter LocationStatsFilter, userID int) ([]models.UserPlatformUsage, error) {
	whereClause, args := buildEngagementWhereClause(filter, "", false)

	query := fmt.Sprintf(`
		WITH user_total AS (
			SELECT COUNT(*) as total
			FROM playback_events
			WHERE %s
				AND user_id = ?
				AND platform IS NOT NULL
				AND platform != ''
		)
		SELECT
			platform,
			MIN(started_at) as first_used,
			MAX(started_at) as last_used,
			COUNT(*) as session_count,
			CAST(COALESCE(SUM(play_duration), 0) / 60.0 AS DOUBLE) as watch_time_minutes,
			CAST(COUNT(*) * 100.0 / NULLIF((SELECT total FROM user_total), 0) AS DOUBLE) as percentage,
			MAX(started_at) >= CURRENT_TIMESTAMP - INTERVAL 30 days as is_active
		FROM playback_events
		WHERE %s
			AND user_id = ?
			AND platform IS NOT NULL
			AND platform != ''
		GROUP BY platform
		ORDER BY session_count DESC
	`, whereClause, whereClause)

	args = append(args, userID)
	args = append(args, args[:len(args)-1]...)
	args = append(args, userID)

	rows, err := db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query platform history: %w", err)
	}
	defer rows.Close()

	var history []models.UserPlatformUsage
	isPrimarySet := false
	for rows.Next() {
		var usage models.UserPlatformUsage
		err := rows.Scan(
			&usage.Platform,
			&usage.FirstUsed,
			&usage.LastUsed,
			&usage.SessionCount,
			&usage.TotalWatchTimeMinutes,
			&usage.Percentage,
			&usage.IsActive,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan platform usage: %w", err)
		}

		if !isPrimarySet {
			usage.IsPrimary = true
			isPrimarySet = true
		}

		history = append(history, usage)
	}

	return history, rows.Err()
}

// countUserMigrations counts the number of platform switches for a user.
func (db *DB) countUserMigrations(ctx context.Context, filter LocationStatsFilter, userID int) (int, error) {
	whereClause, args := buildEngagementWhereClause(filter, "", false)

	query := fmt.Sprintf(`
		WITH user_sessions AS (
			SELECT
				platform,
				LAG(platform) OVER (ORDER BY started_at) as prev_platform
			FROM playback_events
			WHERE %s
				AND user_id = ?
				AND platform IS NOT NULL
				AND platform != ''
		)
		SELECT COUNT(*) FROM user_sessions
		WHERE prev_platform IS NOT NULL AND prev_platform != platform
	`, whereClause)

	args = append(args, userID)

	var count int
	err := db.conn.QueryRowContext(ctx, query, args...).Scan(&count)
	return count, err
}

// getRecentMigrations retrieves recent platform migration events.
func (db *DB) getRecentMigrations(ctx context.Context, filter LocationStatsFilter, limit int) ([]models.DeviceMigration, error) {
	whereClause, args := buildEngagementWhereClause(filter, "", false)

	query := fmt.Sprintf(`
		WITH user_sessions AS (
			SELECT
				user_id,
				username,
				platform,
				started_at,
				LAG(platform) OVER (PARTITION BY user_id ORDER BY started_at) as prev_platform,
				LAG(started_at) OVER (PARTITION BY user_id ORDER BY started_at) as prev_started_at
			FROM playback_events
			WHERE %s
				AND platform IS NOT NULL
				AND platform != ''
				AND username IS NOT NULL
		),
		migrations AS (
			SELECT
				user_id,
				username,
				prev_platform as from_platform,
				platform as to_platform,
				started_at as migration_date
			FROM user_sessions
			WHERE prev_platform IS NOT NULL
				AND prev_platform != platform
		),
		migration_stats AS (
			SELECT
				m.user_id,
				m.username,
				m.from_platform,
				m.to_platform,
				m.migration_date,
				(SELECT COUNT(*) FROM playback_events p
				 WHERE p.user_id = m.user_id
				   AND p.platform = m.from_platform
				   AND p.started_at < m.migration_date) as sessions_before,
				(SELECT COUNT(*) FROM playback_events p
				 WHERE p.user_id = m.user_id
				   AND p.platform = m.to_platform
				   AND p.started_at >= m.migration_date) as sessions_after,
				(SELECT COUNT(*) FROM playback_events p
				 WHERE p.user_id = m.user_id
				   AND p.platform = m.from_platform
				   AND p.started_at > m.migration_date) = 0 as is_permanent
			FROM migrations m
		)
		SELECT
			user_id,
			username,
			from_platform,
			to_platform,
			migration_date,
			sessions_before,
			sessions_after,
			is_permanent
		FROM migration_stats
		ORDER BY migration_date DESC
		LIMIT ?
	`, whereClause)

	args = append(args, limit)
	rows, err := db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query migrations: %w", err)
	}
	defer rows.Close()

	var migrations []models.DeviceMigration
	for rows.Next() {
		var m models.DeviceMigration
		err := rows.Scan(
			&m.UserID,
			&m.Username,
			&m.FromPlatform,
			&m.ToPlatform,
			&m.MigrationDate,
			&m.SessionsBeforeMigration,
			&m.SessionsAfterMigration,
			&m.IsPermanentSwitch,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan migration: %w", err)
		}
		migrations = append(migrations, m)
	}

	return migrations, rows.Err()
}

// getPlatformAdoptionTrends retrieves platform adoption trends over time.
func (db *DB) getPlatformAdoptionTrends(ctx context.Context, filter LocationStatsFilter) ([]models.PlatformAdoptionTrend, error) {
	whereClause, args := buildEngagementWhereClause(filter, "", false)

	// Determine appropriate interval based on date range
	interval := determineTrendInterval(filter)

	query := fmt.Sprintf(`
		WITH period_stats AS (
			SELECT
				DATE_TRUNC('%s', started_at) as period,
				platform,
				COUNT(*) as session_count,
				COUNT(DISTINCT user_id) as active_users
			FROM playback_events
			WHERE %s
				AND platform IS NOT NULL
				AND platform != ''
			GROUP BY DATE_TRUNC('%s', started_at), platform
		),
		first_usage AS (
			SELECT
				user_id,
				platform,
				DATE_TRUNC('%s', MIN(started_at)) as first_period
			FROM playback_events
			WHERE %s
				AND platform IS NOT NULL
				AND platform != ''
			GROUP BY user_id, platform
		),
		new_users_per_period AS (
			SELECT
				first_period as period,
				platform,
				COUNT(*) as new_users
			FROM first_usage
			GROUP BY first_period, platform
		),
		period_totals AS (
			SELECT
				period,
				SUM(session_count) as total_sessions
			FROM period_stats
			GROUP BY period
		)
		SELECT
			CAST(p.period AS VARCHAR) as date,
			p.platform,
			COALESCE(n.new_users, 0) as new_users,
			p.active_users,
			p.session_count,
			CAST(p.session_count * 100.0 / NULLIF(t.total_sessions, 0) AS DOUBLE) as market_share
		FROM period_stats p
		LEFT JOIN new_users_per_period n ON p.period = n.period AND p.platform = n.platform
		LEFT JOIN period_totals t ON p.period = t.period
		ORDER BY p.period DESC, p.session_count DESC
	`, interval, whereClause, interval, interval, whereClause)

	rows, err := db.conn.QueryContext(ctx, query, append(args, args...)...)
	if err != nil {
		return nil, fmt.Errorf("failed to query adoption trends: %w", err)
	}
	defer rows.Close()

	var trends []models.PlatformAdoptionTrend
	for rows.Next() {
		var t models.PlatformAdoptionTrend
		err := rows.Scan(
			&t.Date,
			&t.Platform,
			&t.NewUsers,
			&t.ActiveUsers,
			&t.SessionCount,
			&t.MarketShare,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan trend: %w", err)
		}
		trends = append(trends, t)
	}

	return trends, rows.Err()
}

// getCommonPlatformTransitions retrieves the most common platform transition paths.
func (db *DB) getCommonPlatformTransitions(ctx context.Context, filter LocationStatsFilter, limit int) ([]models.PlatformTransition, error) {
	whereClause, args := buildEngagementWhereClause(filter, "", false)

	query := fmt.Sprintf(`
		WITH user_sessions AS (
			SELECT
				user_id,
				platform,
				started_at,
				LAG(platform) OVER (PARTITION BY user_id ORDER BY started_at) as prev_platform,
				LAG(started_at) OVER (PARTITION BY user_id ORDER BY started_at) as prev_started_at
			FROM playback_events
			WHERE %s
				AND platform IS NOT NULL
				AND platform != ''
		),
		transitions AS (
			SELECT
				user_id,
				prev_platform as from_platform,
				platform as to_platform,
				EXTRACT(EPOCH FROM (started_at - prev_started_at)) / 86400.0 as days_gap
			FROM user_sessions
			WHERE prev_platform IS NOT NULL
				AND prev_platform != platform
		),
		return_checks AS (
			SELECT
				t.user_id,
				t.from_platform,
				t.to_platform,
				CASE WHEN EXISTS (
					SELECT 1 FROM user_sessions us
					WHERE us.user_id = t.user_id
						AND us.platform = t.from_platform
						AND us.prev_platform = t.to_platform
				) THEN 1 ELSE 0 END as returned
			FROM transitions t
		)
		SELECT
			t.from_platform,
			t.to_platform,
			COUNT(*) as transition_count,
			COUNT(DISTINCT t.user_id) as unique_users,
			AVG(t.days_gap) as avg_days_before_switch,
			CAST(SUM(COALESCE(r.returned, 0)) * 100.0 / COUNT(*) AS DOUBLE) as return_rate
		FROM transitions t
		LEFT JOIN return_checks r ON t.user_id = r.user_id AND t.from_platform = r.from_platform AND t.to_platform = r.to_platform
		GROUP BY t.from_platform, t.to_platform
		ORDER BY transition_count DESC
		LIMIT ?
	`, whereClause)

	args = append(args, limit)
	rows, err := db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query transitions: %w", err)
	}
	defer rows.Close()

	var transitions []models.PlatformTransition
	for rows.Next() {
		var t models.PlatformTransition
		var avgDays, returnRate sql.NullFloat64
		err := rows.Scan(
			&t.FromPlatform,
			&t.ToPlatform,
			&t.TransitionCount,
			&t.UniqueUsers,
			&avgDays,
			&returnRate,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan transition: %w", err)
		}
		if avgDays.Valid {
			t.AvgDaysBeforeSwitch = avgDays.Float64
		}
		if returnRate.Valid {
			t.ReturnRate = returnRate.Float64
		}
		transitions = append(transitions, t)
	}

	return transitions, rows.Err()
}

// determineTrendInterval selects the appropriate time interval based on filter date range.
func determineTrendInterval(filter LocationStatsFilter) string {
	if filter.StartDate == nil || filter.EndDate == nil {
		return "week" // Default to weekly
	}

	days := int(filter.EndDate.Sub(*filter.StartDate).Hours() / 24)

	switch {
	case days <= 7:
		return "day"
	case days <= 60:
		return "week"
	case days <= 365:
		return "month"
	default:
		return "quarter"
	}
}

// buildDeviceMigrationMetadata builds metadata for the device migration response.
func (db *DB) buildDeviceMigrationMetadata(ctx context.Context, filter LocationStatsFilter, startTime time.Time) models.DeviceMigrationMetadata {
	metadata := models.DeviceMigrationMetadata{
		ExecutionTimeMS:     time.Since(startTime).Milliseconds(),
		MigrationWindowDays: migrationWindowDays,
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
	hashInput := fmt.Sprintf("device_migration:%v:%v", metadata.DataRangeStart, metadata.DataRangeEnd)
	hash := sha256.Sum256([]byte(hashInput))
	metadata.QueryHash = hex.EncodeToString(hash[:8])

	// Get event count and platform count
	whereClause, args := buildEngagementWhereClause(filter, "", false)
	countQuery := fmt.Sprintf(`
		SELECT
			COUNT(*) as event_count,
			COUNT(DISTINCT platform) as platform_count
		FROM playback_events
		WHERE %s
			AND platform IS NOT NULL
			AND platform != ''
	`, whereClause)

	// Metadata counts are best-effort; errors don't affect the main response
	err := db.conn.QueryRowContext(ctx, countQuery, args...).Scan(
		&metadata.TotalEventsAnalyzed,
		&metadata.UniquePlatformsFound,
	)
	if err != nil {
		// Log but don't fail - metadata is supplementary
		metadata.TotalEventsAnalyzed = 0
		metadata.UniquePlatformsFound = 0
	}

	return metadata
}
