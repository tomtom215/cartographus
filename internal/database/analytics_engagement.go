// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package database provides data access and analytics functionality for the Cartographus application.
// This file contains user engagement analytics including popular content, watch parties, and viewing patterns.
package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/tomtom215/cartographus/internal/models"
)

// buildEngagementWhereClause builds a WHERE clause with optional table alias prefix for engagement queries.
// This consolidates the logic previously duplicated across buildFilterWhereClause, buildPopularContentWhereClause,
// buildWatchPartyWhereClause, and buildUserEngagementWhereClause.
//
// Parameters:
//   - filter: LocationStatsFilter with date/user/mediaType filters
//   - tableAlias: Optional table alias prefix (e.g., "", "p", "p1")
//   - includeMediaTypes: Whether to include media_type filter (some queries don't need it)
//
// Returns: WHERE clause string and query arguments
func buildEngagementWhereClause(filter LocationStatsFilter, tableAlias string, includeMediaTypes bool) (string, []interface{}) {
	whereClauses := []string{"1=1"}
	args := []interface{}{}

	// Helper to add prefix if provided
	col := func(name string) string {
		if tableAlias == "" {
			return name
		}
		return tableAlias + "." + name
	}

	// Add date filters
	if filter.StartDate != nil {
		whereClauses = append(whereClauses, col("started_at")+" >= ?")
		args = append(args, *filter.StartDate)
	}
	if filter.EndDate != nil {
		whereClauses = append(whereClauses, col("started_at")+" <= ?")
		args = append(args, *filter.EndDate)
	}

	// Add user filter
	if len(filter.Users) > 0 {
		placeholders := make([]string, len(filter.Users))
		for i, user := range filter.Users {
			placeholders[i] = "?"
			args = append(args, user)
		}
		whereClauses = append(whereClauses, fmt.Sprintf("%s IN (%s)", col("username"), join(placeholders, ", ")))
	}

	// Add media type filter (optional)
	if includeMediaTypes && len(filter.MediaTypes) > 0 {
		placeholders := make([]string, len(filter.MediaTypes))
		for i, mediaType := range filter.MediaTypes {
			placeholders[i] = "?"
			args = append(args, mediaType)
		}
		whereClauses = append(whereClauses, fmt.Sprintf("%s IN (%s)", col("media_type"), join(placeholders, ", ")))
	}

	return join(whereClauses, " AND "), args
}

// scanPopularContent scans a single row into PopularContent model
// Extracted to reduce duplication across queryTopMovies, queryTopShows, queryTopEpisodes
func scanPopularContent(rows *sql.Rows) (models.PopularContent, error) {
	var content models.PopularContent
	err := rows.Scan(
		&content.MediaType,
		&content.Title,
		&content.ParentTitle,
		&content.GrandparentTitle,
		&content.PlaybackCount,
		&content.UniqueUsers,
		&content.AvgCompletion,
		&content.FirstPlayed,
		&content.LastPlayed,
		&content.Year,
		&content.ContentRating,
		&content.TotalWatchTime,
	)
	return content, err
}

// queryPopularContentByType retrieves popular content by media type with custom query
// Reduces duplication across queryTopMovies, queryTopShows, queryTopEpisodes
func (db *DB) queryPopularContentByType(ctx context.Context, query string, args []interface{}, errorContext string) ([]models.PopularContent, error) {
	rows, err := db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query %s: %w", errorContext, err)
	}
	defer rows.Close()

	var contentList []models.PopularContent
	for rows.Next() {
		content, err := scanPopularContent(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan %s: %w", errorContext, err)
		}
		contentList = append(contentList, content)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating %s: %w", errorContext, err)
	}

	return contentList, nil
}

// queryTopMovies retrieves top movies by playback count
func (db *DB) queryTopMovies(ctx context.Context, whereClause string, args []interface{}, limit int) ([]models.PopularContent, error) {
	query := fmt.Sprintf(`
		SELECT
			media_type,
			title,
			NULL as parent_title,
			NULL as grandparent_title,
			COUNT(*) as playback_count,
			COUNT(DISTINCT user_id) as unique_users,
			AVG(percent_complete) as avg_completion,
			MIN(started_at) as first_played,
			MAX(started_at) as last_played,
			year,
			content_rating,
			CAST(COALESCE(SUM(play_duration), 0) / 60 AS INTEGER) as total_watch_time
		FROM playback_events
		WHERE %s AND media_type = 'movie'
		GROUP BY media_type, title, year, content_rating
		ORDER BY playback_count DESC
		LIMIT ?
	`, whereClause)

	queryArgs := append(args, limit)
	return db.queryPopularContentByType(ctx, query, queryArgs, "top movies")
}

// queryTopShows retrieves top TV shows by playback count (grouped by grandparent_title)
func (db *DB) queryTopShows(ctx context.Context, whereClause string, args []interface{}, limit int) ([]models.PopularContent, error) {
	query := fmt.Sprintf(`
		SELECT
			'show' as media_type,
			grandparent_title as title,
			NULL as parent_title,
			NULL as grandparent_title_dup,
			COUNT(*) as playback_count,
			COUNT(DISTINCT user_id) as unique_users,
			AVG(percent_complete) as avg_completion,
			MIN(started_at) as first_played,
			MAX(started_at) as last_played,
			NULL as year,
			NULL as content_rating,
			CAST(COALESCE(SUM(play_duration), 0) / 60 AS INTEGER) as total_watch_time
		FROM playback_events
		WHERE %s AND media_type = 'episode' AND grandparent_title IS NOT NULL AND grandparent_title != ''
		GROUP BY grandparent_title
		ORDER BY playback_count DESC
		LIMIT ?
	`, whereClause)

	queryArgs := append(args, limit)
	return db.queryPopularContentByType(ctx, query, queryArgs, "top shows")
}

// queryTopEpisodes retrieves top episodes by playback count
func (db *DB) queryTopEpisodes(ctx context.Context, whereClause string, args []interface{}, limit int) ([]models.PopularContent, error) {
	query := fmt.Sprintf(`
		SELECT
			media_type,
			title,
			parent_title,
			grandparent_title,
			COUNT(*) as playback_count,
			COUNT(DISTINCT user_id) as unique_users,
			AVG(percent_complete) as avg_completion,
			MIN(started_at) as first_played,
			MAX(started_at) as last_played,
			NULL as year,
			NULL as content_rating,
			CAST(COALESCE(SUM(play_duration), 0) / 60 AS INTEGER) as total_watch_time
		FROM playback_events
		WHERE %s AND media_type = 'episode'
		GROUP BY media_type, title, parent_title, grandparent_title
		ORDER BY playback_count DESC
		LIMIT ?
	`, whereClause)

	queryArgs := append(args, limit)
	return db.queryPopularContentByType(ctx, query, queryArgs, "top episodes")
}

// GetPopularContent retrieves popular content analytics (movies, shows, episodes)
func (db *DB) GetPopularContent(ctx context.Context, filter LocationStatsFilter, limit int) (*models.PopularAnalytics, error) {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	// Build WHERE clause once (no alias, exclude media types since queries filter explicitly)
	whereClause, args := buildEngagementWhereClause(filter, "", false)

	// Query all three content types using helper methods
	movies, err := db.queryTopMovies(ctx, whereClause, args, limit)
	if err != nil {
		return nil, err
	}

	shows, err := db.queryTopShows(ctx, whereClause, args, limit)
	if err != nil {
		return nil, err
	}

	episodes, err := db.queryTopEpisodes(ctx, whereClause, args, limit)
	if err != nil {
		return nil, err
	}

	return &models.PopularAnalytics{
		TopMovies:   movies,
		TopShows:    shows,
		TopEpisodes: episodes,
	}, nil
}

// watchPartyCTE returns the common CTE for detecting watch parties
// A watch party is 2+ users watching the same content within 15 minutes
// Extracted to reduce duplication across 5 watch party query functions
func watchPartyCTE() string {
	return `
		SELECT
			p1.id as anchor_id,
			p1.media_type,
			p1.title,
			p1.parent_title,
			p1.grandparent_title,
			p1.started_at as party_time,
			COUNT(DISTINCT p2.user_id) as participant_count,
			COUNT(DISTINCT CASE WHEN p2.ip_address != '' THEN p2.ip_address END) = 1 as same_ip,
			AVG(p2.percent_complete) as avg_completion,
			SUM(COALESCE(p2.play_duration, 0)) as total_duration
		FROM playback_events p1
		INNER JOIN playback_events p2 ON (
			p1.title = p2.title
			AND COALESCE(p1.parent_title, '') = COALESCE(p2.parent_title, '')
			AND COALESCE(p1.grandparent_title, '') = COALESCE(p2.grandparent_title, '')
			AND ABS(epoch(p2.started_at - p1.started_at)) <= 900
		)
		WHERE %s
		GROUP BY p1.id, p1.media_type, p1.title, p1.parent_title, p1.grandparent_title, p1.started_at
		HAVING COUNT(DISTINCT p2.user_id) >= 2`
}

// getWatchPartySummary retrieves overall watch party summary statistics
func (db *DB) getWatchPartySummary(ctx context.Context, whereClause string, args []interface{}) (int, int, float64, int, error) {
	cte := watchPartyCTE()
	query := fmt.Sprintf(`
		WITH watch_parties AS (`+cte+`)
		SELECT
			COUNT(*) as total_parties,
			COALESCE(SUM(participant_count), 0) as total_participants,
			COALESCE(AVG(participant_count), 0.0) as avg_participants,
			COALESCE(SUM(CASE WHEN same_ip THEN 1 ELSE 0 END), 0) as same_location_parties
		FROM watch_parties
	`, whereClause)

	var totalParties, totalParticipants, sameLocationParties int
	var avgParticipants float64
	err := db.conn.QueryRowContext(ctx, query, args...).Scan(
		&totalParties,
		&totalParticipants,
		&avgParticipants,
		&sameLocationParties,
	)

	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return 0, 0, 0, 0, fmt.Errorf("failed to query watch party summary: %w", err)
	}

	return totalParties, totalParticipants, avgParticipants, sameLocationParties, nil
}

// getRecentWatchParties retrieves the most recent watch parties (last 10)
func (db *DB) getRecentWatchParties(ctx context.Context, whereClause string, args []interface{}) ([]models.WatchParty, error) {
	cte := watchPartyCTE()
	query := fmt.Sprintf(`
		WITH watch_parties AS (`+cte+`)
		SELECT
			media_type,
			title,
			parent_title,
			grandparent_title,
			party_time,
			participant_count,
			avg_completion,
			total_duration
		FROM watch_parties
		ORDER BY party_time DESC
		LIMIT 10
	`, whereClause)

	rows, err := db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent watch parties: %w", err)
	}
	defer rows.Close()

	var parties []models.WatchParty
	for rows.Next() {
		var party models.WatchParty
		if err := rows.Scan(
			&party.MediaType,
			&party.Title,
			&party.ParentTitle,
			&party.GrandparentTitle,
			&party.PartyTime,
			&party.ParticipantCount,
			&party.AvgCompletion,
			&party.TotalDuration,
		); err != nil {
			return nil, fmt.Errorf("failed to scan watch party: %w", err)
		}
		// Note: Participants list would require another query per party, skipping for performance
		party.Participants = []models.WatchPartyParticipant{}
		parties = append(parties, party)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating watch parties: %w", err)
	}

	return parties, nil
}

// getTopWatchPartyContent retrieves content with the most watch parties
func (db *DB) getTopWatchPartyContent(ctx context.Context, whereClause string, args []interface{}) ([]models.WatchPartyContentStats, error) {
	// Simplified CTE without same_ip calculation (not needed for this query)
	query := fmt.Sprintf(`
		WITH watch_parties AS (
			SELECT
				p1.media_type,
				p1.title,
				p1.parent_title,
				p1.grandparent_title,
				COUNT(DISTINCT p2.user_id) as participant_count
			FROM playback_events p1
			INNER JOIN playback_events p2 ON (
				p1.title = p2.title
				AND COALESCE(p1.parent_title, '') = COALESCE(p2.parent_title, '')
				AND COALESCE(p1.grandparent_title, '') = COALESCE(p2.grandparent_title, '')
				AND ABS(epoch(p2.started_at - p1.started_at)) <= 900
			)
			WHERE %s
			GROUP BY p1.id, p1.media_type, p1.title, p1.parent_title, p1.grandparent_title
			HAVING COUNT(DISTINCT p2.user_id) >= 2
		)
		SELECT
			media_type,
			title,
			parent_title,
			grandparent_title,
			COUNT(*) as party_count,
			SUM(participant_count) as total_participants,
			AVG(participant_count) as avg_participants,
			COUNT(DISTINCT participant_count) as unique_users
		FROM watch_parties
		GROUP BY media_type, title, parent_title, grandparent_title
		ORDER BY party_count DESC
		LIMIT 10
	`, whereClause)

	rows, err := db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query top content: %w", err)
	}
	defer rows.Close()

	var contentList []models.WatchPartyContentStats
	for rows.Next() {
		var content models.WatchPartyContentStats
		if err := rows.Scan(
			&content.MediaType,
			&content.Title,
			&content.ParentTitle,
			&content.GrandparentTitle,
			&content.PartyCount,
			&content.TotalParticipants,
			&content.AvgParticipants,
			&content.UniqueUsers,
		); err != nil {
			return nil, fmt.Errorf("failed to scan content stats: %w", err)
		}
		contentList = append(contentList, content)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating content stats: %w", err)
	}

	return contentList, nil
}

// getTopSocialUsers retrieves users with the most watch party participations
// Takes two separate where clauses because the CTEs use different table aliases (p1 and p)
func (db *DB) getTopSocialUsers(ctx context.Context, whereClauseP1 string, argsP1 []interface{}, whereClauseP string, argsP []interface{}) ([]models.WatchPartyUserStats, error) {
	query := fmt.Sprintf(`
		WITH watch_parties AS (
			SELECT
				p1.id as anchor_id,
				p1.title,
				p1.grandparent_title,
				COUNT(DISTINCT p2.user_id) as participant_count,
				COUNT(DISTINCT CASE WHEN p2.ip_address != '' THEN p2.ip_address END) = 1 as same_location
			FROM playback_events p1
			INNER JOIN playback_events p2 ON (
				p1.title = p2.title
				AND COALESCE(p1.parent_title, '') = COALESCE(p2.parent_title, '')
				AND COALESCE(p1.grandparent_title, '') = COALESCE(p2.grandparent_title, '')
				AND ABS(epoch(p2.started_at - p1.started_at)) <= 900
			)
			WHERE %s
			GROUP BY p1.id, p1.title, p1.grandparent_title
			HAVING COUNT(DISTINCT p2.user_id) >= 2
		),
		user_parties AS (
			SELECT
				p.user_id,
				p.username,
				wp.anchor_id,
				wp.participant_count,
				wp.same_location,
				COALESCE(wp.grandparent_title, wp.title) as content_title
			FROM playback_events p
			INNER JOIN watch_parties wp ON p.id = wp.anchor_id OR (
				p.title = wp.title
				AND COALESCE(p.grandparent_title, '') = COALESCE(wp.grandparent_title, '')
			)
			WHERE %s
		)
		SELECT
			user_id,
			username,
			COUNT(DISTINCT anchor_id) as party_count,
			SUM(participant_count - 1) as total_co_watchers,
			AVG(participant_count) as avg_party_size,
			SUM(CASE WHEN same_location THEN 1 ELSE 0 END) as same_location_count,
			(SELECT content_title FROM user_parties up2
			 WHERE up2.user_id = user_parties.user_id
			 GROUP BY content_title
			 ORDER BY COUNT(*) DESC LIMIT 1) as favorite_content
		FROM user_parties
		GROUP BY user_id, username
		ORDER BY party_count DESC
		LIMIT 10
	`, whereClauseP1, whereClauseP)

	// Combine args for both WHERE clauses
	allArgs := make([]interface{}, 0, len(argsP1)+len(argsP))
	allArgs = append(allArgs, argsP1...) // for first WHERE clause (watch_parties CTE with p1 alias)
	allArgs = append(allArgs, argsP...)  // for second WHERE clause (user_parties CTE with p alias)
	rows, err := db.conn.QueryContext(ctx, query, allArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to query top social users: %w", err)
	}
	defer rows.Close()

	var userList []models.WatchPartyUserStats
	for rows.Next() {
		var user models.WatchPartyUserStats
		if err := rows.Scan(
			&user.UserID,
			&user.Username,
			&user.PartyCount,
			&user.TotalCoWatchers,
			&user.AvgPartySize,
			&user.SameLocationCount,
			&user.FavoriteContent,
		); err != nil {
			return nil, fmt.Errorf("failed to scan user stats: %w", err)
		}
		userList = append(userList, user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating user stats: %w", err)
	}

	return userList, nil
}

// getWatchPartiesByDay retrieves watch party distribution by day of week
func (db *DB) getWatchPartiesByDay(ctx context.Context, whereClause string, args []interface{}) ([]models.WatchPartyByDay, error) {
	// Simplified CTE without same_ip and other unused fields
	query := fmt.Sprintf(`
		WITH watch_parties AS (
			SELECT
				p1.started_at as party_time,
				COUNT(DISTINCT p2.user_id) as participant_count
			FROM playback_events p1
			INNER JOIN playback_events p2 ON (
				p1.title = p2.title
				AND COALESCE(p1.parent_title, '') = COALESCE(p2.parent_title, '')
				AND COALESCE(p1.grandparent_title, '') = COALESCE(p2.grandparent_title, '')
				AND ABS(epoch(p2.started_at - p1.started_at)) <= 900
			)
			WHERE %s
			GROUP BY p1.id, p1.started_at
			HAVING COUNT(DISTINCT p2.user_id) >= 2
		)
		SELECT
			DAYOFWEEK(party_time) as day_of_week,
			COUNT(*) as party_count,
			AVG(participant_count) as avg_participants
		FROM watch_parties
		GROUP BY day_of_week
		ORDER BY day_of_week
	`, whereClause)

	rows, err := db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query parties by day: %w", err)
	}
	defer rows.Close()

	dayStatsMap := make(map[int]models.WatchPartyByDay)
	for rows.Next() {
		var dayStats models.WatchPartyByDay
		if err := rows.Scan(&dayStats.DayOfWeek, &dayStats.PartyCount, &dayStats.AvgParticipants); err != nil {
			return nil, fmt.Errorf("failed to scan day stats: %w", err)
		}
		dayStatsMap[dayStats.DayOfWeek] = dayStats
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating day stats: %w", err)
	}

	// Fill in all 7 days (0-6, Sunday-Saturday)
	var partiesByDay []models.WatchPartyByDay
	for day := 0; day < 7; day++ {
		if stats, exists := dayStatsMap[day]; exists {
			partiesByDay = append(partiesByDay, stats)
		} else {
			partiesByDay = append(partiesByDay, models.WatchPartyByDay{
				DayOfWeek:       day,
				PartyCount:      0,
				AvgParticipants: 0,
			})
		}
	}

	return partiesByDay, nil
}

// GetWatchParties detects and analyzes watch parties (2+ users watching same content together)
// A watch party is detected when different users watch the same content within a 15-minute time window
func (db *DB) GetWatchParties(ctx context.Context, filter LocationStatsFilter) (*models.WatchPartyAnalytics, error) {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	// Build WHERE clause with p1 prefix for most queries (self-join on p1/p2)
	whereClauseP1, argsP1 := buildEngagementWhereClause(filter, "p1", true)

	// Build WHERE clause with p prefix for user_parties CTE in getTopSocialUsers
	whereClauseP, argsP := buildEngagementWhereClause(filter, "p", true)

	// Get summary statistics
	totalParties, totalParticipants, avgParticipants, sameLocationParties, err := db.getWatchPartySummary(ctx, whereClauseP1, argsP1)
	if err != nil {
		return nil, err
	}

	analytics := &models.WatchPartyAnalytics{
		TotalWatchParties:   totalParties,
		TotalParticipants:   totalParticipants,
		AvgParticipants:     avgParticipants,
		SameLocationParties: sameLocationParties,
		RecentWatchParties:  []models.WatchParty{},
		TopContent:          []models.WatchPartyContentStats{},
		TopSocialUsers:      []models.WatchPartyUserStats{},
		PartiesByDay:        []models.WatchPartyByDay{},
	}

	if totalParties == 0 {
		return analytics, nil
	}

	// Query all detailed statistics using helper methods
	recentParties, err := db.getRecentWatchParties(ctx, whereClauseP1, argsP1)
	if err != nil {
		return nil, err
	}
	analytics.RecentWatchParties = recentParties

	topContent, err := db.getTopWatchPartyContent(ctx, whereClauseP1, argsP1)
	if err != nil {
		return nil, err
	}
	analytics.TopContent = topContent

	// getTopSocialUsers needs two different WHERE clauses:
	// - First CTE (watch_parties) uses p1 alias
	// - Second CTE (user_parties) uses p alias
	topUsers, err := db.getTopSocialUsers(ctx, whereClauseP1, argsP1, whereClauseP, argsP)
	if err != nil {
		return nil, err
	}
	analytics.TopSocialUsers = topUsers

	partiesByDay, err := db.getWatchPartiesByDay(ctx, whereClauseP1, argsP1)
	if err != nil {
		return nil, err
	}
	analytics.PartiesByDay = partiesByDay

	return analytics, nil
}

// getDayName returns the name of the day for a given day of week (0-6)
func getDayName(dayOfWeek int) string {
	dayNames := []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}
	if dayOfWeek >= 0 && dayOfWeek < len(dayNames) {
		return dayNames[dayOfWeek]
	}
	return "Unknown"
}

// getTopEngagedUsers retrieves top engaged users with calculated activity scores
// Takes two WHERE clauses because:
// - First CTE (user_metrics) has no table alias
// - Second and third CTEs (user_media_types, user_titles) use 'playback_events p'
func (db *DB) getTopEngagedUsers(ctx context.Context, whereClauseNoAlias string, argsNoAlias []interface{}, whereClauseP string, argsP []interface{}, limit int) ([]models.UserEngagement, error) {
	userQuery := fmt.Sprintf(`
		WITH user_metrics AS (
			SELECT
				user_id,
				username,
				COALESCE(SUM(play_duration), 0) as total_watch_time_minutes,
				COUNT(DISTINCT session_key) as total_sessions,
				COALESCE(AVG(play_duration), 0) as avg_session_minutes,
				MIN(started_at) as first_seen,
				MAX(started_at) as last_seen,
				COUNT(*) as total_content_items,
				COUNT(DISTINCT title || COALESCE(parent_title, '') || COALESCE(grandparent_title, '')) as unique_content_items,
				COALESCE(AVG(percent_complete), 0) as avg_completion,
				SUM(CASE WHEN percent_complete >= 90 THEN 1 ELSE 0 END) as fully_watched_count,
				COUNT(DISTINCT ip_address) as unique_locations,
				COUNT(DISTINCT platform) as unique_platforms
			FROM playback_events
			WHERE %s AND play_duration IS NOT NULL AND play_duration > 0
			GROUP BY user_id, username
		),
		top_users_list AS (
			SELECT user_id, username
			FROM user_metrics
			ORDER BY total_watch_time_minutes DESC
			LIMIT ?
		),
		user_media_types AS (
			SELECT
				p.user_id,
				p.media_type
			FROM playback_events p
			INNER JOIN top_users_list t ON p.user_id = t.user_id
			WHERE %s
			GROUP BY p.user_id, p.media_type
			QUALIFY ROW_NUMBER() OVER (PARTITION BY p.user_id ORDER BY COUNT(*) DESC) = 1
		),
		user_titles AS (
			SELECT
				p.user_id,
				CASE
					WHEN p.grandparent_title IS NOT NULL THEN p.grandparent_title
					ELSE p.title
				END as display_title
			FROM playback_events p
			INNER JOIN top_users_list t ON p.user_id = t.user_id
			WHERE %s
			GROUP BY p.user_id, display_title
			QUALIFY ROW_NUMBER() OVER (PARTITION BY p.user_id ORDER BY COUNT(*) DESC) = 1
		)
		SELECT
			um.user_id,
			um.username,
			um.total_watch_time_minutes,
			um.total_sessions,
			um.avg_session_minutes,
			um.first_seen,
			um.last_seen,
			um.total_content_items,
			um.unique_content_items,
			um.avg_completion,
			um.fully_watched_count,
			um.unique_locations,
			um.unique_platforms,
			umt.media_type as most_watched_type,
			ut.display_title as most_watched_title
		FROM user_metrics um
		INNER JOIN top_users_list tul ON um.user_id = tul.user_id
		LEFT JOIN user_media_types umt ON um.user_id = umt.user_id
		LEFT JOIN user_titles ut ON um.user_id = ut.user_id
		ORDER BY um.total_watch_time_minutes DESC
	`, whereClauseNoAlias, whereClauseP, whereClauseP)

	// Build args: first WHERE (no alias) + limit + second WHERE (p alias) + third WHERE (p alias)
	allArgs := make([]interface{}, 0, len(argsNoAlias)+1+len(argsP)*2)
	allArgs = append(allArgs, argsNoAlias...) // for first WHERE clause (user_metrics, no alias)
	allArgs = append(allArgs, limit)          // for LIMIT in top_users_list CTE
	allArgs = append(allArgs, argsP...)       // for second WHERE clause (user_media_types, p alias)
	allArgs = append(allArgs, argsP...)       // for third WHERE clause (user_titles, p alias)
	rows, err := db.conn.QueryContext(ctx, userQuery, allArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to query user engagement: %w", err)
	}
	defer rows.Close()

	return db.scanEngagedUsers(rows)
}

// scanEngagedUsers scans rows into UserEngagement models with calculated fields
// Extracted to reduce complexity of getTopEngagedUsers
func (db *DB) scanEngagedUsers(rows *sql.Rows) ([]models.UserEngagement, error) {
	var topUsers []models.UserEngagement
	now := time.Now()

	for rows.Next() {
		user, err := db.scanSingleEngagedUser(rows, now)
		if err != nil {
			return nil, err
		}
		topUsers = append(topUsers, user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating user engagement: %w", err)
	}

	return topUsers, nil
}

// scanSingleEngagedUser scans a single row into UserEngagement model
// Extracted to reduce complexity and enable reuse
func (db *DB) scanSingleEngagedUser(rows *sql.Rows, now time.Time) (models.UserEngagement, error) {
	var user models.UserEngagement
	var firstSeen, lastSeen time.Time
	var mostWatchedType, mostWatchedTitle sql.NullString

	err := rows.Scan(
		&user.UserID,
		&user.Username,
		&user.TotalWatchTimeMinutes,
		&user.TotalSessions,
		&user.AverageSessionMinutes,
		&firstSeen,
		&lastSeen,
		&user.TotalContentItems,
		&user.UniqueContentItems,
		&user.AvgCompletionRate,
		&user.FullyWatchedCount,
		&user.UniqueLocations,
		&user.UniquePlatforms,
		&mostWatchedType,
		&mostWatchedTitle,
	)
	if err != nil {
		return user, fmt.Errorf("failed to scan user engagement: %w", err)
	}

	// Set time fields
	user.FirstSeenAt = firstSeen
	user.LastSeenAt = lastSeen
	user.DaysSinceFirstSeen = int(now.Sub(firstSeen).Hours() / 24)
	user.DaysSinceLastSeen = int(now.Sub(lastSeen).Hours() / 24)

	// Set most watched type and title if available
	if mostWatchedType.Valid {
		user.MostWatchedType = &mostWatchedType.String
	}
	if mostWatchedTitle.Valid {
		user.MostWatchedTitle = &mostWatchedTitle.String
	}

	// Calculate activity score and return visitor rate
	user.ActivityScore = calculateActivityScore(&user)
	user.ReturnVisitorRate = calculateReturnVisitorRate(user.TotalSessions)

	return user, nil
}

// calculateActivityScore computes weighted activity score for a user
// Components: watch time (40%), sessions (30%), completion rate (20%), unique content (10%)
func calculateActivityScore(user *models.UserEngagement) float64 {
	watchTimeScore := user.TotalWatchTimeMinutes / 60.0 // hours
	sessionScore := float64(user.TotalSessions)
	completionScore := user.AvgCompletionRate / 10.0
	contentScore := float64(user.UniqueContentItems)

	return (watchTimeScore * 0.4) + (sessionScore * 0.3) + (completionScore * 0.2) + (contentScore * 0.1)
}

// calculateReturnVisitorRate computes return visitor rate based on session count
func calculateReturnVisitorRate(totalSessions int) float64 {
	if totalSessions > 1 {
		return 100.0
	}
	return 0.0
}

// getUserEngagementSummary retrieves overall engagement summary statistics including return visitor rate
func (db *DB) getUserEngagementSummary(ctx context.Context, whereClause string, args []interface{}) (models.UserEngagementSummary, error) {
	summaryQuery := fmt.Sprintf(`
		SELECT
			COUNT(DISTINCT user_id) as total_users,
			COUNT(DISTINCT user_id) as active_users,
			COALESCE(SUM(play_duration), 0) as total_watch_time,
			COUNT(DISTINCT session_key) as total_sessions,
			COALESCE(AVG(play_duration), 0) as avg_session_minutes,
			COALESCE(AVG(percent_complete), 0) as avg_completion
		FROM playback_events
		WHERE %s AND play_duration IS NOT NULL AND play_duration > 0
	`, whereClause)

	var summary models.UserEngagementSummary
	var totalSessions int
	if err := db.conn.QueryRowContext(ctx, summaryQuery, args...).Scan(
		&summary.TotalUsers,
		&summary.ActiveUsers,
		&summary.TotalWatchTimeMinutes,
		&totalSessions,
		&summary.AvgSessionMinutes,
		&summary.AvgCompletionRate,
	); err != nil {
		return summary, fmt.Errorf("failed to query engagement summary: %w", err)
	}

	summary.TotalSessions = totalSessions
	if summary.TotalUsers > 0 {
		summary.AvgUserWatchTime = summary.TotalWatchTimeMinutes / float64(summary.TotalUsers)
	}

	// Calculate return visitor rate (users with 2+ sessions)
	returnVisitors, err := db.getReturnVisitorCount(ctx, whereClause, args)
	if err != nil {
		return summary, err
	}

	if summary.TotalUsers > 0 {
		summary.ReturnVisitorRate = (float64(returnVisitors) / float64(summary.TotalUsers)) * 100.0
	}

	return summary, nil
}

// getReturnVisitorCount retrieves count of users with 2+ sessions
// Extracted to reduce complexity of getUserEngagementSummary
func (db *DB) getReturnVisitorCount(ctx context.Context, whereClause string, args []interface{}) (int, error) {
	returnVisitorQuery := fmt.Sprintf(`
		SELECT COUNT(DISTINCT user_id)
		FROM (
			SELECT user_id, COUNT(DISTINCT session_key) as session_count
			FROM playback_events
			WHERE %s
			GROUP BY user_id
			HAVING COUNT(DISTINCT session_key) >= 2
		) subq
	`, whereClause)

	var returnVisitors int
	err := db.conn.QueryRowContext(ctx, returnVisitorQuery, args...).Scan(&returnVisitors)
	if err != nil {
		return 0, fmt.Errorf("failed to query return visitors: %w", err)
	}

	return returnVisitors, nil
}

// getViewingPatternsByHour retrieves hourly viewing patterns and identifies most active hour
func (db *DB) getViewingPatternsByHour(ctx context.Context, whereClause string, args []interface{}) ([]models.ViewingPatternByHour, *int, error) {
	hourQuery := fmt.Sprintf(`
		SELECT
			HOUR(started_at) as hour_of_day,
			COUNT(DISTINCT session_key) as session_count,
			COALESCE(SUM(play_duration), 0) as watch_time_minutes,
			COUNT(DISTINCT user_id) as unique_users,
			COALESCE(AVG(percent_complete), 0) as avg_completion
		FROM playback_events
		WHERE %s AND play_duration IS NOT NULL AND play_duration > 0
		GROUP BY hour_of_day
		ORDER BY hour_of_day
	`, whereClause)

	rows, err := db.conn.QueryContext(ctx, hourQuery, args...)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to query viewing patterns by hour: %w", err)
	}
	defer rows.Close()

	return scanHourlyPatterns(rows)
}

// scanHourlyPatterns scans rows into hourly patterns and finds most active hour
// Extracted to reduce complexity of getViewingPatternsByHour
func scanHourlyPatterns(rows *sql.Rows) ([]models.ViewingPatternByHour, *int, error) {
	var patternsByHour []models.ViewingPatternByHour
	var mostActiveHour *int
	maxHourSessions := 0

	for rows.Next() {
		var pattern models.ViewingPatternByHour
		if err := rows.Scan(
			&pattern.HourOfDay,
			&pattern.SessionCount,
			&pattern.WatchTimeMinutes,
			&pattern.UniqueUsers,
			&pattern.AvgCompletion,
		); err != nil {
			return nil, nil, fmt.Errorf("failed to scan hour pattern: %w", err)
		}

		if pattern.SessionCount > maxHourSessions {
			maxHourSessions = pattern.SessionCount
			hour := pattern.HourOfDay
			mostActiveHour = &hour
		}

		patternsByHour = append(patternsByHour, pattern)
	}

	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("error iterating hour patterns: %w", err)
	}

	return patternsByHour, mostActiveHour, nil
}

// getViewingPatternsByDay retrieves daily viewing patterns and identifies most active day
func (db *DB) getViewingPatternsByDay(ctx context.Context, whereClause string, args []interface{}) ([]models.ViewingPatternByDay, *int, error) {
	dayQuery := fmt.Sprintf(`
		SELECT
			DAYOFWEEK(started_at) as day_of_week,
			COUNT(DISTINCT session_key) as session_count,
			COALESCE(SUM(play_duration), 0) as watch_time_minutes,
			COUNT(DISTINCT user_id) as unique_users,
			COALESCE(AVG(percent_complete), 0) as avg_completion
		FROM playback_events
		WHERE %s AND play_duration IS NOT NULL AND play_duration > 0
		GROUP BY day_of_week
		ORDER BY day_of_week
	`, whereClause)

	rows, err := db.conn.QueryContext(ctx, dayQuery, args...)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to query viewing patterns by day: %w", err)
	}
	defer rows.Close()

	return scanDailyPatterns(rows)
}

// scanDailyPatterns scans rows into daily patterns and finds most active day
// Extracted to reduce complexity of getViewingPatternsByDay
func scanDailyPatterns(rows *sql.Rows) ([]models.ViewingPatternByDay, *int, error) {
	var patternsByDay []models.ViewingPatternByDay
	var mostActiveDay *int
	maxDaySessions := 0

	for rows.Next() {
		var pattern models.ViewingPatternByDay
		if err := rows.Scan(
			&pattern.DayOfWeek,
			&pattern.SessionCount,
			&pattern.WatchTimeMinutes,
			&pattern.UniqueUsers,
			&pattern.AvgCompletion,
		); err != nil {
			return nil, nil, fmt.Errorf("failed to scan day pattern: %w", err)
		}

		pattern.DayName = getDayName(pattern.DayOfWeek)

		if pattern.SessionCount > maxDaySessions {
			maxDaySessions = pattern.SessionCount
			day := pattern.DayOfWeek
			mostActiveDay = &day
		}

		patternsByDay = append(patternsByDay, pattern)
	}

	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("error iterating day patterns: %w", err)
	}

	return patternsByDay, mostActiveDay, nil
}

// GetUserEngagement retrieves comprehensive user engagement analytics
func (db *DB) GetUserEngagement(ctx context.Context, filter LocationStatsFilter, limit int) (*models.UserEngagementAnalytics, error) {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	// Validate and set default limit
	limit = clampLimit(limit, 1, 100, 10)

	// Build WHERE clause without alias for queries on bare playback_events table
	whereClauseNoAlias, argsNoAlias := buildEngagementWhereClause(filter, "", true)

	// Build WHERE clause with 'p' alias for queries that join playback_events p
	whereClauseP, argsP := buildEngagementWhereClause(filter, "p", true)

	// Execute all queries using helper methods
	// getTopEngagedUsers needs both WHERE clauses (first CTE has no alias, second/third have 'p' alias)
	topUsers, err := db.getTopEngagedUsers(ctx, whereClauseNoAlias, argsNoAlias, whereClauseP, argsP, limit)
	if err != nil {
		return nil, err
	}

	summary, err := db.getUserEngagementSummary(ctx, whereClauseNoAlias, argsNoAlias)
	if err != nil {
		return nil, err
	}

	patternsByHour, mostActiveHour, err := db.getViewingPatternsByHour(ctx, whereClauseNoAlias, argsNoAlias)
	if err != nil {
		return nil, err
	}

	patternsByDay, mostActiveDay, err := db.getViewingPatternsByDay(ctx, whereClauseNoAlias, argsNoAlias)
	if err != nil {
		return nil, err
	}

	return &models.UserEngagementAnalytics{
		Summary:               summary,
		TopUsers:              topUsers,
		ViewingPatternsByHour: patternsByHour,
		ViewingPatternsByDay:  patternsByDay,
		MostActiveHour:        mostActiveHour,
		MostActiveDay:         mostActiveDay,
	}, nil
}

// clampLimit ensures limit is within minValue/maxValue range, defaulting to defValue if invalid
func clampLimit(limit, minValue, maxValue, defValue int) int {
	if limit < minValue {
		return defValue
	}
	if limit > maxValue {
		return maxValue
	}
	return limit
}
