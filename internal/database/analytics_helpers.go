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
)

// analytics_helpers.go - Shared helper functions for analytics queries
// This file reduces complexity in analytics_advanced.go by extracting common patterns

// buildWhereClauseWithArgs builds a WHERE clause and args from a filter
// Reduces code duplication across all analytics functions
func buildWhereClauseWithArgs(filter LocationStatsFilter, baseConditions ...string) (string, []interface{}) {
	whereClauses := make([]string, 0, len(baseConditions)+10)
	whereClauses = append(whereClauses, baseConditions...)
	args := []interface{}{}

	// Date range filters
	if filter.StartDate != nil {
		whereClauses = append(whereClauses, "started_at >= ?")
		args = append(args, *filter.StartDate)
	}

	if filter.EndDate != nil {
		whereClauses = append(whereClauses, "started_at <= ?")
		args = append(args, *filter.EndDate)
	}

	// User filters
	if len(filter.Users) > 0 {
		placeholders := make([]string, len(filter.Users))
		for i, user := range filter.Users {
			placeholders[i] = "?"
			args = append(args, user)
		}
		whereClauses = append(whereClauses, fmt.Sprintf("username IN (%s)", join(placeholders, ", ")))
	}

	// Media type filters
	if len(filter.MediaTypes) > 0 {
		placeholders := make([]string, len(filter.MediaTypes))
		for i, mediaType := range filter.MediaTypes {
			placeholders[i] = "?"
			args = append(args, mediaType)
		}
		whereClauses = append(whereClauses, fmt.Sprintf("media_type IN (%s)", join(placeholders, ", ")))
	}

	// Platform filters
	if len(filter.Platforms) > 0 {
		placeholders := make([]string, len(filter.Platforms))
		for i, platform := range filter.Platforms {
			placeholders[i] = "?"
			args = append(args, platform)
		}
		whereClauses = append(whereClauses, fmt.Sprintf("platform IN (%s)", join(placeholders, ", ")))
	}

	// Player filters
	if len(filter.Players) > 0 {
		placeholders := make([]string, len(filter.Players))
		for i, player := range filter.Players {
			placeholders[i] = "?"
			args = append(args, player)
		}
		whereClauses = append(whereClauses, fmt.Sprintf("player IN (%s)", join(placeholders, ", ")))
	}

	whereClause := join(whereClauses, " AND ")
	return whereClause, args
}

// queryRowWithContext executes a query expecting a single row and scans into dest
// Reduces error handling boilerplate in analytics functions
func (db *DB) queryRowWithContext(ctx context.Context, query string, args []interface{}, dest ...interface{}) error {
	row := db.conn.QueryRowContext(ctx, query, args...)
	if err := row.Scan(dest...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Return zero values for aggregations when no data
			return nil
		}
		return fmt.Errorf("scan row: %w", err)
	}
	return nil
}

// queryAndScan executes a query and scans all rows using the provided scanner function
// Reduces repetitive query-scan-collect patterns
func (db *DB) queryAndScan(ctx context.Context, query string, args []interface{}, scanner func(*sql.Rows) error) error {
	rows, err := db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		if err := scanner(rows); err != nil {
			return fmt.Errorf("scan row: %w", err)
		}
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("rows iteration: %w", err)
	}

	return nil
}

// buildBingeSQLWithWhere constructs the binge detection SQL with a WHERE clause placeholder
// Extracted from GetBingeAnalytics to reduce function length
func buildBingeSQLWithWhere() string {
	return `
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
		ORDER BY total_duration DESC
		LIMIT 50`
}

// buildBingeSummarySQL constructs summary statistics query for binge analytics
// Extracted from GetBingeAnalytics to reduce complexity
func buildBingeSummarySQL() string {
	return `
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
			COUNT(*) as total_binge_sessions,
			SUM(episode_count) as total_episodes_binged,
			SUM(total_duration) as total_watch_time,
			AVG(episode_count) as avg_episodes_per_session,
			PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY episode_count) as median_episodes_per_session
		FROM binge_sessions`
}

// buildTopShowsSQL constructs the top binge-watched shows query
// Extracted from GetBingeAnalytics to reduce complexity
func buildTopShowsSQL() string {
	return `
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
			show_name,
			COUNT(*) as session_count,
			COUNT(DISTINCT user_id) as unique_bingers,
			SUM(episode_count) as total_episodes,
			SUM(total_duration) as total_watch_time
		FROM binge_sessions
		GROUP BY show_name
		ORDER BY session_count DESC, total_watch_time DESC
		LIMIT 20`
}
