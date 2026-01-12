// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package database provides data access and analytics functionality for the Cartographus application.
// This file contains binge-watching analytics implementation.
package database

import (
	"context"
	"fmt"

	"github.com/tomtom215/cartographus/internal/models"
)

// GetBingeAnalytics analyzes binge-watching patterns from playback data.
//
// Binge-watching is defined as 3 or more episodes of the same TV show watched within
// a 6-hour window. This function identifies binge sessions, calculates statistics,
// and ranks shows and users by binge activity.
//
// The analysis includes:
//   - Total binge sessions and episodes binged
//   - Average episodes per binge session
//   - Average binge session duration
//   - Recent binge sessions (last 10)
//   - Top 10 shows by binge count
//   - Top 10 users by binge frequency
//   - Binge distribution by day of week
//
// Parameters:
//   - ctx: Context for query cancellation and timeout control
//   - filter: Filters for date range, users, and other criteria
//
// Returns:
//   - *models.BingeAnalytics: Comprehensive binge-watching statistics
//   - error: Any error encountered during database queries
//
// SQL Implementation:
// Uses window functions (LAG) to detect episode gaps, session markers to group
// consecutive episodes, and CTEs to calculate aggregate statistics. The 6-hour
// threshold (21600 seconds) is based on typical binge-watching research.
//
// Performance:
// Query complexity: O(n log n) due to window function sorting
// Typical execution time: <50ms for 10k playback events
// buildBingeWhereClause builds the WHERE clause and arguments for binge analytics queries
func buildBingeWhereClause(filter LocationStatsFilter) (string, []interface{}) {
	whereClauses := []string{"media_type = 'episode'"}
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

	whereClause := join(whereClauses, " AND ")
	return whereClause, args
}

// queryBingeSessions retrieves all binge sessions with totals for statistics calculation
func (db *DB) queryBingeSessions(ctx context.Context, whereClause string, args []interface{}) ([]models.BingeSession, int, int, error) {
	query := fmt.Sprintf(`
		WITH episode_sessions AS (
			SELECT
				user_id,
				username,
				grandparent_title as show_name,
				started_at,
				percent_complete,
				COALESCE(play_duration, 0) as play_duration_min,
				LAG(started_at) OVER (
					PARTITION BY user_id, grandparent_title
					ORDER BY started_at
				) as prev_started_at
			FROM playback_events
			WHERE %s
				AND grandparent_title IS NOT NULL
				AND grandparent_title != ''
		),
		session_markers AS (
			SELECT *,
				CASE
					WHEN prev_started_at IS NULL
						OR epoch(started_at - prev_started_at) > 21600
					THEN 1
					ELSE 0
				END as is_new_session
			FROM episode_sessions
		),
		session_groups AS (
			SELECT *,
				SUM(is_new_session) OVER (
					PARTITION BY user_id, show_name
					ORDER BY started_at
					ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW
				) as session_id
			FROM session_markers
		),
		binge_sessions AS (
			SELECT
				user_id,
				username,
				show_name,
				session_id,
				COUNT(*) as episode_count,
				MIN(started_at) as first_episode_time,
				MAX(started_at) as last_episode_time,
				SUM(play_duration_min) as total_duration,
				AVG(percent_complete) as avg_completion
			FROM session_groups
			GROUP BY user_id, username, show_name, session_id
			HAVING COUNT(*) >= 3
		)
		SELECT
			user_id,
			username,
			show_name,
			episode_count,
			first_episode_time,
			last_episode_time,
			total_duration,
			avg_completion
		FROM binge_sessions
		ORDER BY first_episode_time DESC
	`, whereClause)

	rows, err := db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to query binge sessions: %w", err)
	}
	defer rows.Close()

	var bingeSessions []models.BingeSession
	totalEpisodes := 0
	totalDuration := 0

	for rows.Next() {
		var session models.BingeSession
		if err := rows.Scan(
			&session.UserID,
			&session.Username,
			&session.ShowName,
			&session.EpisodeCount,
			&session.FirstEpisodeTime,
			&session.LastEpisodeTime,
			&session.TotalDuration,
			&session.AvgCompletion,
		); err != nil {
			return nil, 0, 0, fmt.Errorf("failed to scan binge session: %w", err)
		}
		bingeSessions = append(bingeSessions, session)
		totalEpisodes += session.EpisodeCount
		totalDuration += session.TotalDuration
	}

	if err = rows.Err(); err != nil {
		return nil, 0, 0, fmt.Errorf("error iterating binge sessions: %w", err)
	}

	return bingeSessions, totalEpisodes, totalDuration, nil
}

// queryTopBingeShows retrieves shows with the most binge sessions
func (db *DB) queryTopBingeShows(ctx context.Context, whereClause string, args []interface{}) ([]models.BingeShowStats, error) {
	query := fmt.Sprintf(`
		WITH episode_sessions AS (
			SELECT
				user_id,
				grandparent_title as show_name,
				started_at,
				LAG(started_at) OVER (
					PARTITION BY user_id, grandparent_title
					ORDER BY started_at
				) as prev_started_at
			FROM playback_events
			WHERE %s
				AND grandparent_title IS NOT NULL
				AND grandparent_title != ''
		),
		session_markers AS (
			SELECT *,
				CASE
					WHEN prev_started_at IS NULL
						OR epoch(started_at - prev_started_at) > 21600
					THEN 1
					ELSE 0
				END as is_new_session
			FROM episode_sessions
		),
		session_groups AS (
			SELECT *,
				SUM(is_new_session) OVER (
					PARTITION BY user_id, show_name
					ORDER BY started_at
					ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW
				) as session_id
			FROM session_markers
		),
		binge_sessions AS (
			SELECT
				user_id,
				show_name,
				session_id,
				COUNT(*) as episode_count
			FROM session_groups
			GROUP BY user_id, show_name, session_id
			HAVING COUNT(*) >= 3
		)
		SELECT
			show_name,
			COUNT(*) as binge_count,
			SUM(episode_count) as total_episodes,
			COUNT(DISTINCT user_id) as unique_watchers,
			AVG(episode_count) as avg_episodes
		FROM binge_sessions
		GROUP BY show_name
		ORDER BY binge_count DESC
		LIMIT 10
	`, whereClause)

	rows, err := db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query top binge shows: %w", err)
	}
	defer rows.Close()

	var showStats []models.BingeShowStats
	for rows.Next() {
		var stats models.BingeShowStats
		if err := rows.Scan(
			&stats.ShowName,
			&stats.BingeCount,
			&stats.TotalEpisodes,
			&stats.UniqueWatchers,
			&stats.AvgEpisodes,
		); err != nil {
			return nil, fmt.Errorf("failed to scan show stats: %w", err)
		}
		showStats = append(showStats, stats)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating show stats: %w", err)
	}

	return showStats, nil
}

// queryTopBingeWatchers retrieves users with the most binge sessions
func (db *DB) queryTopBingeWatchers(ctx context.Context, whereClause string, args []interface{}) ([]models.BingeUserStats, error) {
	query := fmt.Sprintf(`
		WITH episode_sessions AS (
			SELECT
				user_id,
				username,
				grandparent_title as show_name,
				started_at,
				LAG(started_at) OVER (
					PARTITION BY user_id, grandparent_title
					ORDER BY started_at
				) as prev_started_at
			FROM playback_events
			WHERE %s
				AND grandparent_title IS NOT NULL
				AND grandparent_title != ''
		),
		session_markers AS (
			SELECT *,
				CASE
					WHEN prev_started_at IS NULL
						OR epoch(started_at - prev_started_at) > 21600
					THEN 1
					ELSE 0
				END as is_new_session
			FROM episode_sessions
		),
		session_groups AS (
			SELECT *,
				SUM(is_new_session) OVER (
					PARTITION BY user_id, show_name
					ORDER BY started_at
					ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW
				) as session_id
			FROM session_markers
		),
		binge_sessions AS (
			SELECT
				user_id,
				username,
				show_name,
				session_id,
				COUNT(*) as episode_count
			FROM session_groups
			GROUP BY user_id, username, show_name, session_id
			HAVING COUNT(*) >= 3
		),
		user_show_counts AS (
			SELECT
				user_id,
				show_name,
				COUNT(*) as show_binge_count
			FROM binge_sessions
			GROUP BY user_id, show_name
		)
		SELECT
			bs.user_id,
			bs.username,
			COUNT(*) as binge_count,
			SUM(bs.episode_count) as total_episodes,
			AVG(bs.episode_count) as avg_episodes,
			(SELECT show_name FROM user_show_counts usc
			 WHERE usc.user_id = bs.user_id
			 ORDER BY show_binge_count DESC LIMIT 1) as favorite_show
		FROM binge_sessions bs
		GROUP BY bs.user_id, bs.username
		ORDER BY binge_count DESC
		LIMIT 10
	`, whereClause)

	rows, err := db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query top binge watchers: %w", err)
	}
	defer rows.Close()

	var userStats []models.BingeUserStats
	for rows.Next() {
		var stats models.BingeUserStats
		if err := rows.Scan(
			&stats.UserID,
			&stats.Username,
			&stats.BingeCount,
			&stats.TotalEpisodes,
			&stats.AvgEpisodes,
			&stats.FavoriteShow,
		); err != nil {
			return nil, fmt.Errorf("failed to scan user stats: %w", err)
		}
		userStats = append(userStats, stats)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating user stats: %w", err)
	}

	return userStats, nil
}

// queryBingesByDay retrieves binge session distribution by day of week
func (db *DB) queryBingesByDay(ctx context.Context, whereClause string, args []interface{}) ([]models.BingesByDayOfWeek, error) {
	query := fmt.Sprintf(`
		WITH episode_sessions AS (
			SELECT
				user_id,
				grandparent_title as show_name,
				started_at,
				LAG(started_at) OVER (
					PARTITION BY user_id, grandparent_title
					ORDER BY started_at
				) as prev_started_at
			FROM playback_events
			WHERE %s
				AND grandparent_title IS NOT NULL
				AND grandparent_title != ''
		),
		session_markers AS (
			SELECT *,
				CASE
					WHEN prev_started_at IS NULL
						OR epoch(started_at - prev_started_at) > 21600
					THEN 1
					ELSE 0
				END as is_new_session
			FROM episode_sessions
		),
		session_groups AS (
			SELECT *,
				SUM(is_new_session) OVER (
					PARTITION BY user_id, show_name
					ORDER BY started_at
					ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW
				) as session_id
			FROM session_markers
		),
		binge_sessions AS (
			SELECT
				MIN(started_at) as first_episode_time,
				COUNT(*) as episode_count
			FROM session_groups
			GROUP BY user_id, show_name, session_id
			HAVING COUNT(*) >= 3
		)
		SELECT
			DAYOFWEEK(first_episode_time) - 1 as day_of_week,
			COUNT(*) as binge_count,
			CAST(AVG(episode_count) AS INTEGER) as avg_episodes
		FROM binge_sessions
		GROUP BY day_of_week
		ORDER BY day_of_week
	`, whereClause)

	rows, err := db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query binges by day: %w", err)
	}
	defer rows.Close()

	// Build map to fill in missing days with zero counts
	dayStatsMap := make(map[int]models.BingesByDayOfWeek)
	for rows.Next() {
		var dayStats models.BingesByDayOfWeek
		if err := rows.Scan(&dayStats.DayOfWeek, &dayStats.BingeCount, &dayStats.AvgEpisodes); err != nil {
			return nil, fmt.Errorf("failed to scan day stats: %w", err)
		}
		dayStatsMap[dayStats.DayOfWeek] = dayStats
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating day stats: %w", err)
	}

	// Fill in all 7 days (0-6, Sunday-Saturday) for complete week visualization
	var bingesByDay []models.BingesByDayOfWeek
	for day := 0; day < 7; day++ {
		if stats, exists := dayStatsMap[day]; exists {
			bingesByDay = append(bingesByDay, stats)
		} else {
			bingesByDay = append(bingesByDay, models.BingesByDayOfWeek{
				DayOfWeek:   day,
				BingeCount:  0,
				AvgEpisodes: 0,
			})
		}
	}

	return bingesByDay, nil
}

// GetBingeAnalytics analyzes binge-watching patterns from playback data.
//
// Binge-watching is defined as 3 or more episodes of the same TV show watched
// within a 6-hour window (21600 seconds). This function identifies binge sessions,
// calculates statistics, and ranks shows and users by binge activity.
//
// The analysis includes:
//   - Total binge sessions and episodes binged
//   - Average episodes per binge session
//   - Average binge session duration
//   - Recent binge sessions (last 10)
//   - Top 10 shows by binge count
//   - Top 10 users by binge frequency
//   - Binge distribution by day of week (0=Sunday through 6=Saturday)
//
// Parameters:
//   - ctx: Context for query cancellation and timeout control
//   - filter: Filters for date range, users, and other criteria
//
// Returns:
//   - *models.BingeAnalytics: Comprehensive binge-watching statistics
//   - error: Any error encountered during database queries
//
// SQL Implementation:
// Uses window functions (LAG) to detect episode gaps, session markers to group
// consecutive episodes, and CTEs to calculate aggregate statistics.
//
// Performance:
// Query complexity: O(n log n) due to window function sorting
// Typical execution time: <50ms for 10k playback events
func (db *DB) GetBingeAnalytics(ctx context.Context, filter LocationStatsFilter) (*models.BingeAnalytics, error) {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	// Build WHERE clause once for all queries
	whereClause, args := buildBingeWhereClause(filter)

	// Query all binge sessions with totals
	bingeSessions, totalEpisodes, totalDuration, err := db.queryBingeSessions(ctx, whereClause, args)
	if err != nil {
		return nil, err
	}

	// Initialize analytics with empty slices to avoid null in JSON responses
	analytics := &models.BingeAnalytics{
		TotalBingeSessions:  len(bingeSessions),
		TotalEpisodesBinged: totalEpisodes,
		RecentBingeSessions: []models.BingeSession{},
		TopBingeShows:       []models.BingeShowStats{},
		TopBingeWatchers:    []models.BingeUserStats{},
		BingesByDay:         []models.BingesByDayOfWeek{},
	}

	if len(bingeSessions) == 0 {
		// Even with no data, return all 7 days with zero counts for consistent chart rendering
		for day := 0; day < 7; day++ {
			analytics.BingesByDay = append(analytics.BingesByDay, models.BingesByDayOfWeek{
				DayOfWeek:   day,
				BingeCount:  0,
				AvgEpisodes: 0,
			})
		}
		return analytics, nil
	}

	analytics.AvgEpisodesPerBinge = float64(totalEpisodes) / float64(len(bingeSessions))
	analytics.AvgBingeDuration = float64(totalDuration) / float64(len(bingeSessions))

	// Get recent binge sessions (last 10)
	recentCount := 10
	if len(bingeSessions) < recentCount {
		recentCount = len(bingeSessions)
	}
	analytics.RecentBingeSessions = bingeSessions[:recentCount]

	// Query top binge shows
	topShows, err := db.queryTopBingeShows(ctx, whereClause, args)
	if err != nil {
		return nil, err
	}
	analytics.TopBingeShows = topShows

	// Query top binge watchers
	topWatchers, err := db.queryTopBingeWatchers(ctx, whereClause, args)
	if err != nil {
		return nil, err
	}
	analytics.TopBingeWatchers = topWatchers

	// Query binges by day
	bingesByDay, err := db.queryBingesByDay(ctx, whereClause, args)
	if err != nil {
		return nil, err
	}
	analytics.BingesByDay = bingesByDay

	return analytics, nil
}
