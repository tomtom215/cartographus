// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package database provides database operations for the Cartographus application.
//
// newsletter_content.go - Newsletter Content Store Implementation
//
// This file implements the ContentStore interface required by the newsletter
// content resolver. It provides methods to query playback events for:
//   - Recently added content (movies, shows, music)
//   - Top/popular content based on watch counts
//   - Period statistics (playbacks, watch time, users)
//   - User-specific statistics and recommendations
//   - Server health information
package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/tomtom215/cartographus/internal/models"
)

// GetRecentlyAddedMovies returns movies added since the given time.
// Results are ordered by added_at descending (most recent first).
func (db *DB) GetRecentlyAddedMovies(ctx context.Context, since time.Time, limit int) ([]models.NewsletterMediaItem, error) {
	if limit <= 0 {
		limit = 10
	}

	// Query for recently added movies from playback_events
	// We use added_at from the media metadata and group by rating_key
	query := `
		SELECT DISTINCT ON (rating_key)
			rating_key,
			title,
			year,
			media_type,
			summary,
			genres,
			content_rating,
			thumb,
			art,
			added_at,
			originally_available_at
		FROM playback_events
		WHERE media_type = 'movie'
			AND added_at IS NOT NULL
			AND CAST(added_at AS TIMESTAMP) >= ?
			AND rating_key IS NOT NULL
		ORDER BY rating_key, CAST(added_at AS TIMESTAMP) DESC
		LIMIT ?
	`

	rows, err := db.conn.QueryContext(ctx, query, since, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query recently added movies: %w", err)
	}
	defer rows.Close()

	return scanNewsletterMediaItems(rows)
}

// GetRecentlyAddedShows returns TV shows with new episodes added since the given time.
// Results are grouped by show with episode details.
func (db *DB) GetRecentlyAddedShows(ctx context.Context, since time.Time, limit int) ([]models.NewsletterShowItem, error) {
	if limit <= 0 {
		limit = 10
	}

	// First, get shows with recent episodes
	showsQuery := `
		SELECT
			grandparent_rating_key,
			grandparent_title,
			MIN(year) as year,
			MAX(summary) as summary,
			MAX(genres) as genres,
			MAX(content_rating) as content_rating,
			MAX(grandparent_thumb) as poster_url,
			COUNT(DISTINCT rating_key) as new_episodes_count
		FROM playback_events
		WHERE media_type = 'episode'
			AND added_at IS NOT NULL
			AND CAST(added_at AS TIMESTAMP) >= ?
			AND grandparent_rating_key IS NOT NULL
		GROUP BY grandparent_rating_key, grandparent_title
		ORDER BY MAX(CAST(added_at AS TIMESTAMP)) DESC
		LIMIT ?
	`

	rows, err := db.conn.QueryContext(ctx, showsQuery, since, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query recently added shows: %w", err)
	}
	defer rows.Close()

	var shows []models.NewsletterShowItem
	for rows.Next() {
		var show models.NewsletterShowItem
		var summary, genres, contentRating, posterURL sql.NullString
		var year sql.NullInt64

		err := rows.Scan(
			&show.RatingKey,
			&show.Title,
			&year,
			&summary,
			&genres,
			&contentRating,
			&posterURL,
			&show.NewEpisodesCount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan show: %w", err)
		}

		if year.Valid {
			show.Year = int(year.Int64)
		}
		if summary.Valid {
			show.Summary = summary.String
		}
		if genres.Valid && genres.String != "" {
			show.Genres = strings.Split(genres.String, ", ")
		}
		if contentRating.Valid {
			show.ContentRating = contentRating.String
		}
		if posterURL.Valid {
			show.PosterURL = posterURL.String
		}

		shows = append(shows, show)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating shows: %w", err)
	}

	return shows, nil
}

// GetRecentlyAddedMusic returns music tracks added since the given time.
func (db *DB) GetRecentlyAddedMusic(ctx context.Context, since time.Time, limit int) ([]models.NewsletterMediaItem, error) {
	if limit <= 0 {
		limit = 10
	}

	query := `
		SELECT DISTINCT ON (rating_key)
			rating_key,
			title,
			year,
			media_type,
			summary,
			genres,
			content_rating,
			thumb,
			art,
			added_at,
			originally_available_at
		FROM playback_events
		WHERE media_type = 'track'
			AND added_at IS NOT NULL
			AND CAST(added_at AS TIMESTAMP) >= ?
			AND rating_key IS NOT NULL
		ORDER BY rating_key, CAST(added_at AS TIMESTAMP) DESC
		LIMIT ?
	`

	rows, err := db.conn.QueryContext(ctx, query, since, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query recently added music: %w", err)
	}
	defer rows.Close()

	return scanNewsletterMediaItems(rows)
}

// GetTopMovies returns the most watched movies since the given time.
// Results are ordered by watch count descending.
func (db *DB) GetTopMovies(ctx context.Context, since time.Time, limit int) ([]models.NewsletterMediaItem, error) {
	if limit <= 0 {
		limit = 10
	}

	query := `
		SELECT
			rating_key,
			MAX(title) as title,
			MAX(year) as year,
			'movie' as media_type,
			MAX(summary) as summary,
			MAX(genres) as genres,
			MAX(content_rating) as content_rating,
			MAX(thumb) as thumb,
			MAX(art) as art,
			MAX(added_at) as added_at,
			MAX(originally_available_at) as released_at,
			COUNT(*) as watch_count,
			SUM(COALESCE(
				EXTRACT(EPOCH FROM (stopped_at - started_at)) / 3600.0,
				0
			)) as watch_time_hours
		FROM playback_events
		WHERE media_type = 'movie'
			AND started_at >= ?
			AND rating_key IS NOT NULL
		GROUP BY rating_key
		ORDER BY watch_count DESC, watch_time_hours DESC
		LIMIT ?
	`

	rows, err := db.conn.QueryContext(ctx, query, since, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query top movies: %w", err)
	}
	defer rows.Close()

	return scanNewsletterMediaItemsWithStats(rows)
}

// GetTopShows returns the most watched TV shows since the given time.
// Results are ordered by watch count descending.
func (db *DB) GetTopShows(ctx context.Context, since time.Time, limit int) ([]models.NewsletterShowItem, error) {
	if limit <= 0 {
		limit = 10
	}

	query := `
		SELECT
			grandparent_rating_key,
			MAX(grandparent_title) as title,
			MAX(year) as year,
			MAX(summary) as summary,
			MAX(genres) as genres,
			MAX(content_rating) as content_rating,
			MAX(grandparent_thumb) as poster_url,
			COUNT(*) as watch_count,
			SUM(COALESCE(
				EXTRACT(EPOCH FROM (stopped_at - started_at)) / 3600.0,
				0
			)) as watch_time_hours
		FROM playback_events
		WHERE media_type = 'episode'
			AND started_at >= ?
			AND grandparent_rating_key IS NOT NULL
		GROUP BY grandparent_rating_key
		ORDER BY watch_count DESC, watch_time_hours DESC
		LIMIT ?
	`

	rows, err := db.conn.QueryContext(ctx, query, since, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query top shows: %w", err)
	}
	defer rows.Close()

	var shows []models.NewsletterShowItem
	for rows.Next() {
		var show models.NewsletterShowItem
		var summary, genres, contentRating, posterURL sql.NullString
		var year sql.NullInt64
		var watchTimeHours sql.NullFloat64

		err := rows.Scan(
			&show.RatingKey,
			&show.Title,
			&year,
			&summary,
			&genres,
			&contentRating,
			&posterURL,
			&show.WatchCount,
			&watchTimeHours,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan top show: %w", err)
		}

		if year.Valid {
			show.Year = int(year.Int64)
		}
		if summary.Valid {
			show.Summary = summary.String
		}
		if genres.Valid && genres.String != "" {
			show.Genres = strings.Split(genres.String, ", ")
		}
		if contentRating.Valid {
			show.ContentRating = contentRating.String
		}
		if posterURL.Valid {
			show.PosterURL = posterURL.String
		}
		if watchTimeHours.Valid {
			show.WatchTime = watchTimeHours.Float64
		}

		shows = append(shows, show)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating top shows: %w", err)
	}

	return shows, nil
}

// GetPeriodStats returns aggregate statistics for the given time period.
//
//nolint:gocyclo // Statistics aggregation requires multiple query sections
func (db *DB) GetPeriodStats(ctx context.Context, start, end time.Time) (*models.NewsletterStats, error) {
	stats := &models.NewsletterStats{}

	// Get overall stats
	overallQuery := `
		SELECT
			COUNT(*) as total_playbacks,
			SUM(COALESCE(
				EXTRACT(EPOCH FROM (stopped_at - started_at)) / 3600.0,
				0
			)) as total_watch_time_hours,
			COUNT(DISTINCT user_id) as unique_users,
			COUNT(DISTINCT rating_key) as unique_content
		FROM playback_events
		WHERE started_at >= ? AND started_at < ?
	`

	var watchTimeHours sql.NullFloat64
	err := db.conn.QueryRowContext(ctx, overallQuery, start, end).Scan(
		&stats.TotalPlaybacks,
		&watchTimeHours,
		&stats.UniqueUsers,
		&stats.UniqueContent,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query overall stats: %w", err)
	}
	if watchTimeHours.Valid {
		stats.TotalWatchTimeHours = watchTimeHours.Float64
	}

	// Get content breakdown
	breakdownQuery := `
		SELECT
			COALESCE(SUM(CASE WHEN media_type = 'movie' THEN 1 ELSE 0 END), 0) as movies,
			COALESCE(SUM(CASE WHEN media_type = 'episode' THEN 1 ELSE 0 END), 0) as episodes,
			COALESCE(SUM(CASE WHEN media_type = 'track' THEN 1 ELSE 0 END), 0) as tracks
		FROM playback_events
		WHERE started_at >= ? AND started_at < ?
	`

	err = db.conn.QueryRowContext(ctx, breakdownQuery, start, end).Scan(
		&stats.MoviesWatched,
		&stats.EpisodesWatched,
		&stats.TracksPlayed,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query content breakdown: %w", err)
	}

	// Get top platforms
	platformQuery := `
		SELECT
			COALESCE(platform, 'Unknown') as platform,
			COUNT(*) as watch_count,
			SUM(COALESCE(
				EXTRACT(EPOCH FROM (stopped_at - started_at)) / 3600.0,
				0
			)) as watch_time_hours
		FROM playback_events
		WHERE started_at >= ? AND started_at < ?
		GROUP BY platform
		ORDER BY watch_count DESC
		LIMIT 5
	`

	platformRows, err := db.conn.QueryContext(ctx, platformQuery, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to query platform stats: %w", err)
	}
	defer platformRows.Close()

	totalPlatformWatches := 0
	var platforms []models.PlatformStat
	for platformRows.Next() {
		var p models.PlatformStat
		var watchTime sql.NullFloat64
		if err := platformRows.Scan(&p.Platform, &p.WatchCount, &watchTime); err != nil {
			return nil, fmt.Errorf("failed to scan platform stat: %w", err)
		}
		if watchTime.Valid {
			p.WatchTime = watchTime.Float64
		}
		totalPlatformWatches += p.WatchCount
		platforms = append(platforms, p)
	}
	if err := platformRows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating platform rows: %w", err)
	}

	// Calculate percentages
	for i := range platforms {
		if totalPlatformWatches > 0 {
			platforms[i].Percentage = float64(platforms[i].WatchCount) / float64(totalPlatformWatches) * 100
		}
	}
	stats.TopPlatforms = platforms

	// Get top users
	userQuery := `
		SELECT
			user_id,
			MAX(username) as username,
			COUNT(*) as watch_count,
			SUM(COALESCE(
				EXTRACT(EPOCH FROM (stopped_at - started_at)) / 3600.0,
				0
			)) as watch_time_hours
		FROM playback_events
		WHERE started_at >= ? AND started_at < ?
		GROUP BY user_id
		ORDER BY watch_count DESC
		LIMIT 5
	`

	userRows, err := db.conn.QueryContext(ctx, userQuery, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to query user stats: %w", err)
	}
	defer userRows.Close()

	var users []models.UserStat
	for userRows.Next() {
		var u models.UserStat
		var userID int
		var watchTime sql.NullFloat64
		if err := userRows.Scan(&userID, &u.Username, &u.WatchCount, &watchTime); err != nil {
			return nil, fmt.Errorf("failed to scan user stat: %w", err)
		}
		u.UserID = fmt.Sprintf("%d", userID)
		if watchTime.Valid {
			u.WatchTime = watchTime.Float64
		}
		users = append(users, u)
	}
	if err := userRows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating user rows: %w", err)
	}
	stats.TopUsers = users

	return stats, nil
}

// GetUserStats returns statistics for a specific user in the given time period.
//
//nolint:gocyclo // User statistics require multiple optional queries
func (db *DB) GetUserStats(ctx context.Context, userID string, start, end time.Time) (*models.NewsletterUserData, error) {
	userData := &models.NewsletterUserData{
		UserID: userID,
	}

	query := `
		SELECT
			MAX(username) as username,
			SUM(COALESCE(
				EXTRACT(EPOCH FROM (stopped_at - started_at)) / 3600.0,
				0
			)) as watch_time_hours,
			COUNT(*) as playback_count
		FROM playback_events
		WHERE user_id = CAST(? AS INTEGER)
			AND started_at >= ? AND started_at < ?
	`

	var username sql.NullString
	var watchTimeHours sql.NullFloat64
	err := db.conn.QueryRowContext(ctx, query, userID, start, end).Scan(
		&username,
		&watchTimeHours,
		&userData.PlaybackCount,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return userData, nil
		}
		return nil, fmt.Errorf("failed to query user stats: %w", err)
	}

	if username.Valid {
		userData.Username = username.String
	}
	if watchTimeHours.Valid {
		userData.WatchTimeHours = watchTimeHours.Float64
	}

	// Get top genre
	genreQuery := `
		SELECT genres
		FROM playback_events
		WHERE user_id = CAST(? AS INTEGER)
			AND started_at >= ? AND started_at < ?
			AND genres IS NOT NULL AND genres != ''
		GROUP BY genres
		ORDER BY COUNT(*) DESC
		LIMIT 1
	`

	var topGenre sql.NullString
	if err := db.conn.QueryRowContext(ctx, genreQuery, userID, start, end).Scan(&topGenre); err != nil && err != sql.ErrNoRows {
		// Log error but continue - this is optional data
		_ = err
	}
	if topGenre.Valid && topGenre.String != "" {
		// Take first genre if comma-separated
		genres := strings.Split(topGenre.String, ", ")
		if len(genres) > 0 {
			userData.TopGenre = genres[0]
		}
	}

	// Get top show
	showQuery := `
		SELECT grandparent_title
		FROM playback_events
		WHERE user_id = CAST(? AS INTEGER)
			AND started_at >= ? AND started_at < ?
			AND media_type = 'episode'
			AND grandparent_title IS NOT NULL
		GROUP BY grandparent_title
		ORDER BY COUNT(*) DESC
		LIMIT 1
	`

	var topShow sql.NullString
	if err := db.conn.QueryRowContext(ctx, showQuery, userID, start, end).Scan(&topShow); err != nil && err != sql.ErrNoRows {
		_ = err
	}
	if topShow.Valid {
		userData.TopShow = topShow.String
	}

	// Get top movie
	movieQuery := `
		SELECT title
		FROM playback_events
		WHERE user_id = CAST(? AS INTEGER)
			AND started_at >= ? AND started_at < ?
			AND media_type = 'movie'
		GROUP BY title
		ORDER BY COUNT(*) DESC
		LIMIT 1
	`

	var topMovie sql.NullString
	if err := db.conn.QueryRowContext(ctx, movieQuery, userID, start, end).Scan(&topMovie); err != nil && err != sql.ErrNoRows {
		_ = err
	}
	if topMovie.Valid {
		userData.TopMovie = topMovie.String
	}

	return userData, nil
}

// GetUserRecommendations returns content recommendations for a user.
// This uses a simple collaborative filtering approach based on similar users' watches.
func (db *DB) GetUserRecommendations(ctx context.Context, userID string, limit int) ([]models.NewsletterMediaItem, error) {
	if limit <= 0 {
		limit = 10
	}

	// Get content that similar users watched but this user hasn't
	// Similar users = users who watched the same content
	query := `
		WITH user_content AS (
			SELECT DISTINCT rating_key
			FROM playback_events
			WHERE user_id = CAST(? AS INTEGER)
		),
		similar_users AS (
			SELECT DISTINCT pe.user_id
			FROM playback_events pe
			INNER JOIN user_content uc ON pe.rating_key = uc.rating_key
			WHERE pe.user_id != CAST(? AS INTEGER)
		),
		recommended_content AS (
			SELECT
				pe.rating_key,
				MAX(pe.title) as title,
				MAX(pe.year) as year,
				MAX(pe.media_type) as media_type,
				MAX(pe.summary) as summary,
				MAX(pe.genres) as genres,
				MAX(pe.content_rating) as content_rating,
				MAX(pe.thumb) as thumb,
				MAX(pe.art) as art,
				COUNT(DISTINCT pe.user_id) as recommend_score
			FROM playback_events pe
			INNER JOIN similar_users su ON pe.user_id = su.user_id
			WHERE pe.rating_key NOT IN (SELECT rating_key FROM user_content)
				AND pe.media_type IN ('movie', 'episode')
				AND pe.rating_key IS NOT NULL
			GROUP BY pe.rating_key
		)
		SELECT
			rating_key,
			title,
			year,
			media_type,
			summary,
			genres,
			content_rating,
			thumb,
			art
		FROM recommended_content
		ORDER BY recommend_score DESC
		LIMIT ?
	`

	rows, err := db.conn.QueryContext(ctx, query, userID, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query user recommendations: %w", err)
	}
	defer rows.Close()

	return scanRecommendedItems(rows)
}

// GetServerHealth returns server health statistics for health newsletters.
func (db *DB) GetServerHealth(ctx context.Context) (*models.NewsletterHealthData, error) {
	health := &models.NewsletterHealthData{
		ServerStatus: "healthy",
	}

	// Get database statistics
	statsQuery := `
		SELECT
			COUNT(DISTINCT section_id) as total_libraries,
			COUNT(DISTINCT rating_key) as total_content
		FROM playback_events
		WHERE section_id IS NOT NULL
	`

	err := db.conn.QueryRowContext(ctx, statsQuery).Scan(
		&health.TotalLibraries,
		&health.TotalContent,
	)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to query server health: %w", err)
	}

	// Get last sync time from most recent playback
	syncQuery := `
		SELECT MAX(created_at)
		FROM playback_events
	`

	var lastSync sql.NullTime
	if err := db.conn.QueryRowContext(ctx, syncQuery).Scan(&lastSync); err != nil && err != sql.ErrNoRows {
		_ = err
	}
	if lastSync.Valid {
		health.LastSyncAt = &lastSync.Time
	}

	// Check for any warnings
	// Warning if no recent playbacks (last 24 hours)
	recentQuery := `
		SELECT COUNT(*)
		FROM playback_events
		WHERE started_at >= NOW() - INTERVAL '24 hours'
	`

	var recentCount int
	if err := db.conn.QueryRowContext(ctx, recentQuery).Scan(&recentCount); err != nil && err != sql.ErrNoRows {
		_ = err
	}
	if recentCount == 0 {
		health.Warnings = append(health.Warnings, "No playback activity in the last 24 hours")
	}

	// Set uptime to 100% for now (would need external monitoring for real uptime)
	health.UptimePercent = 100.0

	return health, nil
}

// scanNewsletterMediaItems scans rows into NewsletterMediaItem slice.
func scanNewsletterMediaItems(rows *sql.Rows) ([]models.NewsletterMediaItem, error) {
	var items []models.NewsletterMediaItem
	for rows.Next() {
		var item models.NewsletterMediaItem
		var summary, genres, contentRating, thumb, art, addedAt, releasedAt sql.NullString
		var year sql.NullInt64

		err := rows.Scan(
			&item.RatingKey,
			&item.Title,
			&year,
			&item.MediaType,
			&summary,
			&genres,
			&contentRating,
			&thumb,
			&art,
			&addedAt,
			&releasedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan media item: %w", err)
		}

		if year.Valid {
			item.Year = int(year.Int64)
		}
		if summary.Valid {
			item.Summary = summary.String
		}
		if genres.Valid && genres.String != "" {
			item.Genres = strings.Split(genres.String, ", ")
		}
		if contentRating.Valid {
			item.ContentRating = contentRating.String
		}
		if thumb.Valid {
			item.ThumbURL = thumb.String
		}
		if art.Valid {
			item.PosterURL = art.String
		}
		if addedAt.Valid {
			if t, err := time.Parse(time.RFC3339, addedAt.String); err == nil {
				item.AddedAt = &t
			}
		}
		if releasedAt.Valid {
			if t, err := time.Parse("2006-01-02", releasedAt.String); err == nil {
				item.ReleasedAt = &t
			}
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating media items: %w", err)
	}

	return items, nil
}

// scanNewsletterMediaItemsWithStats scans rows with watch statistics.
//
//nolint:gocyclo // Scanning nullable fields requires individual null checks
func scanNewsletterMediaItemsWithStats(rows *sql.Rows) ([]models.NewsletterMediaItem, error) {
	var items []models.NewsletterMediaItem
	for rows.Next() {
		var item models.NewsletterMediaItem
		var summary, genres, contentRating, thumb, art, addedAt, releasedAt sql.NullString
		var year sql.NullInt64
		var watchTimeHours sql.NullFloat64

		err := rows.Scan(
			&item.RatingKey,
			&item.Title,
			&year,
			&item.MediaType,
			&summary,
			&genres,
			&contentRating,
			&thumb,
			&art,
			&addedAt,
			&releasedAt,
			&item.WatchCount,
			&watchTimeHours,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan media item with stats: %w", err)
		}

		if year.Valid {
			item.Year = int(year.Int64)
		}
		if summary.Valid {
			item.Summary = summary.String
		}
		if genres.Valid && genres.String != "" {
			item.Genres = strings.Split(genres.String, ", ")
		}
		if contentRating.Valid {
			item.ContentRating = contentRating.String
		}
		if thumb.Valid {
			item.ThumbURL = thumb.String
		}
		if art.Valid {
			item.PosterURL = art.String
		}
		if addedAt.Valid {
			if t, err := time.Parse(time.RFC3339, addedAt.String); err == nil {
				item.AddedAt = &t
			}
		}
		if releasedAt.Valid {
			if t, err := time.Parse("2006-01-02", releasedAt.String); err == nil {
				item.ReleasedAt = &t
			}
		}
		if watchTimeHours.Valid {
			item.WatchTime = watchTimeHours.Float64
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating media items: %w", err)
	}

	return items, nil
}

// scanRecommendedItems scans recommendation query results.
func scanRecommendedItems(rows *sql.Rows) ([]models.NewsletterMediaItem, error) {
	var items []models.NewsletterMediaItem
	for rows.Next() {
		var item models.NewsletterMediaItem
		var summary, genres, contentRating, thumb, art sql.NullString
		var year sql.NullInt64

		err := rows.Scan(
			&item.RatingKey,
			&item.Title,
			&year,
			&item.MediaType,
			&summary,
			&genres,
			&contentRating,
			&thumb,
			&art,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan recommended item: %w", err)
		}

		if year.Valid {
			item.Year = int(year.Int64)
		}
		if summary.Valid {
			item.Summary = summary.String
		}
		if genres.Valid && genres.String != "" {
			item.Genres = strings.Split(genres.String, ", ")
		}
		if contentRating.Valid {
			item.ContentRating = contentRating.String
		}
		if thumb.Valid {
			item.ThumbURL = thumb.String
		}
		if art.Valid {
			item.PosterURL = art.String
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating recommended items: %w", err)
	}

	return items, nil
}
