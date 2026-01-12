// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package database provides database operations for the Cartographus application.
//
// analytics_wrapped.go - Annual Wrapped Report Analytics
//
// This file contains the analytics queries for generating "Spotify Wrapped" style
// annual reports for users. Reports include:
//   - Total watch time, playbacks, and unique content
//   - Top movies, shows, genres, actors, and directors
//   - Viewing patterns by hour, day of week, and month
//   - Binge-watching statistics
//   - Quality metrics (bitrate, direct play rate, HDR/4K usage)
//   - Discovery metrics (new content watched)
//   - Achievements and percentiles compared to other users
//
// Performance:
//   - Report generation: ~100-500ms depending on data volume
//   - Queries use indexes on started_at, user_id for efficient filtering
//   - Results are cached in wrapped_reports table for subsequent requests
//
// Mathematical Correctness:
//   - Completion rate: AVG(CASE WHEN percent_complete >= 90 THEN 1.0 ELSE 0.0 END) * 100
//   - Discovery rate: new_content_count / total_playbacks * 100
//   - Percentiles: Uses PERCENT_RANK() for accurate ranking against all users
//   - Watch time: SUM(COALESCE(play_duration, 0)) / 3600.0 hours
package database

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/tomtom215/cartographus/internal/models"
)

// WrappedYearFilter specifies the year for wrapped report generation.
type WrappedYearFilter struct {
	Year   int
	UserID *int // If nil, generate for all users
}

// GetWrappedReport retrieves a cached wrapped report for a user and year.
// Returns nil, nil if no report exists (not an error condition).
//
//nolint:gocyclo // Complex JSON parsing required for wrapped report with many fields
func (db *DB) GetWrappedReport(ctx context.Context, userID int, year int) (*models.WrappedReport, error) {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	query := `
		SELECT
			id, user_id, username, year,
			total_watch_time_hours, total_playbacks, unique_content_count,
			completion_rate, days_active, longest_streak_days, avg_daily_watch_minutes,
			binge_sessions, total_binge_hours, favorite_binge_show, avg_binge_episodes,
			longest_binge_json::VARCHAR,
			avg_bitrate_mbps, direct_play_rate, hdr_viewing_percent, four_k_viewing_percent,
			preferred_platform, preferred_player,
			new_content_count, discovery_rate, first_watch_of_year, last_watch_of_year,
			peak_hour, peak_day, peak_month,
			viewing_by_hour::VARCHAR, viewing_by_day::VARCHAR, viewing_by_month::VARCHAR, monthly_trends::VARCHAR,
			top_movies::VARCHAR, top_shows::VARCHAR, top_episodes::VARCHAR, top_genres::VARCHAR, top_actors::VARCHAR, top_directors::VARCHAR,
			achievements::VARCHAR, percentiles::VARCHAR,
			share_token, shareable_text, generated_at
		FROM wrapped_reports
		WHERE user_id = ? AND year = ?
	`

	row := db.conn.QueryRowContext(ctx, query, userID, year)

	var report models.WrappedReport
	var longestBingeJSON, viewingByHourJSON, viewingByDayJSON, viewingByMonthJSON sql.NullString
	var monthlyTrendsJSON, topMoviesJSON, topShowsJSON, topEpisodesJSON sql.NullString
	var topGenresJSON, topActorsJSON, topDirectorsJSON sql.NullString
	var achievementsJSON, percentilesJSON string
	var favoriteBingeShow, preferredPlatform, preferredPlayer sql.NullString
	var firstWatch, lastWatch, shareToken, shareableText sql.NullString

	err := row.Scan(
		&report.ID, &report.UserID, &report.Username, &report.Year,
		&report.TotalWatchTimeHours, &report.TotalPlaybacks, &report.UniqueContentCount,
		&report.CompletionRate, &report.DaysActive, &report.LongestStreakDays, &report.AvgDailyWatchMinutes,
		&report.BingeSessions, &report.TotalBingeHours, &favoriteBingeShow, &report.AvgBingeEpisodes,
		&longestBingeJSON,
		&report.AvgBitrateMbps, &report.DirectPlayRate, &report.HDRViewingPercent, &report.FourKViewingPercent,
		&preferredPlatform, &preferredPlayer,
		&report.NewContentCount, &report.DiscoveryRate, &firstWatch, &lastWatch,
		&report.PeakHour, &report.PeakDay, &report.PeakMonth,
		&viewingByHourJSON, &viewingByDayJSON, &viewingByMonthJSON, &monthlyTrendsJSON,
		&topMoviesJSON, &topShowsJSON, &topEpisodesJSON, &topGenresJSON, &topActorsJSON, &topDirectorsJSON,
		&achievementsJSON, &percentilesJSON,
		&shareToken, &shareableText, &report.GeneratedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil // No report exists - not an error
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan wrapped report: %w", err)
	}

	// Parse nullable strings
	report.FavoriteBingeShow = favoriteBingeShow.String
	report.PreferredPlatform = preferredPlatform.String
	report.PreferredPlayer = preferredPlayer.String
	report.FirstWatchOfYear = firstWatch.String
	report.LastWatchOfYear = lastWatch.String
	report.ShareToken = shareToken.String
	report.ShareableText = shareableText.String

	// Parse JSON fields using helpers - reduces repetitive error handling
	jsonFields := []struct {
		field     sql.NullString
		dest      interface{}
		fieldName string
	}{
		{longestBingeJSON, &report.LongestBinge, "longest_binge"},
		{viewingByHourJSON, &report.ViewingByHour, "viewing_by_hour"},
		{viewingByDayJSON, &report.ViewingByDay, "viewing_by_day"},
		{viewingByMonthJSON, &report.ViewingByMonth, "viewing_by_month"},
		{monthlyTrendsJSON, &report.MonthlyTrends, "monthly_trends"},
		{topMoviesJSON, &report.TopMovies, "top_movies"},
		{topShowsJSON, &report.TopShows, "top_shows"},
		{topEpisodesJSON, &report.TopEpisodes, "top_episodes"},
		{topGenresJSON, &report.TopGenres, "top_genres"},
		{topActorsJSON, &report.TopActors, "top_actors"},
		{topDirectorsJSON, &report.TopDirectors, "top_directors"},
	}

	for _, jf := range jsonFields {
		if err := parseJSONFieldInto(jf.field, jf.dest, jf.fieldName); err != nil {
			return nil, err
		}
	}

	// Parse required JSON fields (not nullable)
	if err := json.Unmarshal([]byte(achievementsJSON), &report.Achievements); err != nil {
		return nil, fmt.Errorf("failed to parse achievements: %w", err)
	}
	if err := json.Unmarshal([]byte(percentilesJSON), &report.Percentiles); err != nil {
		return nil, fmt.Errorf("failed to parse percentiles: %w", err)
	}

	return &report, nil
}

// GetWrappedReportByShareToken retrieves a wrapped report by its share token.
// This allows anonymous access to shared reports without authentication.
func (db *DB) GetWrappedReportByShareToken(ctx context.Context, shareToken string) (*models.WrappedReport, error) {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	// First get user_id and year from the share token
	var userID, year int
	query := `SELECT user_id, year FROM wrapped_reports WHERE share_token = ?`
	err := db.conn.QueryRowContext(ctx, query, shareToken).Scan(&userID, &year)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find report by share token: %w", err)
	}

	return db.GetWrappedReport(ctx, userID, year)
}

// GenerateWrappedReport generates a new wrapped report for a user and year.
// This computes all statistics from the playback_events table.
func (db *DB) GenerateWrappedReport(ctx context.Context, userID int, year int) (*models.WrappedReport, error) {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	// Define date range for the year
	startDate := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(year+1, 1, 1, 0, 0, 0, 0, time.UTC)

	// Get username
	username, err := db.getWrappedUsername(ctx, userID, startDate, endDate)
	if err != nil {
		return nil, err
	}
	if username == "" {
		return nil, fmt.Errorf("no playback data found for user %d in year %d", userID, year)
	}

	report := &models.WrappedReport{
		ID:          generateWrappedID(),
		Year:        year,
		UserID:      userID,
		Username:    username,
		GeneratedAt: time.Now(),
		ShareToken:  generateShareToken(),
	}

	// Execute all wrapped queries
	if err := db.populateWrappedCoreStats(ctx, report, userID, startDate, endDate); err != nil {
		return nil, fmt.Errorf("failed to get core stats: %w", err)
	}

	if err := db.populateWrappedTopContent(ctx, report, userID, startDate, endDate); err != nil {
		return nil, fmt.Errorf("failed to get top content: %w", err)
	}

	if err := db.populateWrappedViewingPatterns(ctx, report, userID, startDate, endDate); err != nil {
		return nil, fmt.Errorf("failed to get viewing patterns: %w", err)
	}

	if err := db.populateWrappedBingeStats(ctx, report, userID, startDate, endDate); err != nil {
		return nil, fmt.Errorf("failed to get binge stats: %w", err)
	}

	if err := db.populateWrappedQualityMetrics(ctx, report, userID, startDate, endDate); err != nil {
		return nil, fmt.Errorf("failed to get quality metrics: %w", err)
	}

	if err := db.populateWrappedDiscoveryMetrics(ctx, report, userID, startDate, endDate); err != nil {
		return nil, fmt.Errorf("failed to get discovery metrics: %w", err)
	}

	// Calculate achievements based on collected stats
	report.Achievements = db.calculateWrappedAchievements(report)

	// Calculate percentiles compared to other users
	if err := db.populateWrappedPercentiles(ctx, report, year); err != nil {
		return nil, fmt.Errorf("failed to get percentiles: %w", err)
	}

	// Generate shareable text
	report.ShareableText = generateShareableText(report)

	// Save to database
	if err := db.saveWrappedReport(ctx, report); err != nil {
		return nil, fmt.Errorf("failed to save report: %w", err)
	}

	return report, nil
}

// getWrappedUsername retrieves the username for a user ID from playback data.
func (db *DB) getWrappedUsername(ctx context.Context, userID int, startDate, endDate time.Time) (string, error) {
	query := `
		SELECT username
		FROM playback_events
		WHERE user_id = ? AND started_at >= ? AND started_at < ?
		LIMIT 1
	`
	var username string
	err := db.conn.QueryRowContext(ctx, query, userID, startDate, endDate).Scan(&username)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to get username: %w", err)
	}
	return username, nil
}

// populateWrappedCoreStats fills in the core statistics for a wrapped report.
func (db *DB) populateWrappedCoreStats(ctx context.Context, report *models.WrappedReport, userID int, startDate, endDate time.Time) error {
	query := `
		WITH user_stats AS (
			SELECT
				SUM(COALESCE(play_duration, 0)) / 3600.0 AS total_hours,
				COUNT(*) AS total_playbacks,
				COUNT(DISTINCT rating_key) AS unique_content,
				AVG(CASE WHEN percent_complete >= 90 THEN 1.0 ELSE 0.0 END) * 100 AS completion_rate,
				COUNT(DISTINCT DATE_TRUNC('day', started_at)) AS days_active
			FROM playback_events
			WHERE user_id = ? AND started_at >= ? AND started_at < ?
		),
		daily_activity AS (
			SELECT DATE_TRUNC('day', started_at) AS day
			FROM playback_events
			WHERE user_id = ? AND started_at >= ? AND started_at < ?
			GROUP BY DATE_TRUNC('day', started_at)
			ORDER BY day
		),
		streaks AS (
			SELECT
				day,
				day - INTERVAL (ROW_NUMBER() OVER (ORDER BY day)) DAY AS streak_group
			FROM daily_activity
		),
		streak_lengths AS (
			SELECT COUNT(*) AS streak_length
			FROM streaks
			GROUP BY streak_group
		)
		SELECT
			us.total_hours,
			us.total_playbacks,
			us.unique_content,
			us.completion_rate,
			us.days_active,
			COALESCE((SELECT MAX(streak_length) FROM streak_lengths), 0) AS longest_streak,
			CASE WHEN us.days_active > 0
				THEN (us.total_hours * 60) / us.days_active
				ELSE 0
			END AS avg_daily_minutes
		FROM user_stats us
	`

	row := db.conn.QueryRowContext(ctx, query, userID, startDate, endDate, userID, startDate, endDate)

	var totalHours, completionRate, avgDailyMinutes sql.NullFloat64
	var totalPlaybacks, uniqueContent, daysActive, longestStreak sql.NullInt64

	err := row.Scan(
		&totalHours,
		&totalPlaybacks,
		&uniqueContent,
		&completionRate,
		&daysActive,
		&longestStreak,
		&avgDailyMinutes,
	)
	if err != nil {
		return fmt.Errorf("failed to scan core stats: %w", err)
	}

	report.TotalWatchTimeHours = totalHours.Float64
	report.TotalPlaybacks = int(totalPlaybacks.Int64)
	report.UniqueContentCount = int(uniqueContent.Int64)
	report.CompletionRate = completionRate.Float64
	report.DaysActive = int(daysActive.Int64)
	report.LongestStreakDays = int(longestStreak.Int64)
	report.AvgDailyWatchMinutes = avgDailyMinutes.Float64

	return nil
}

// populateWrappedTopContent fills in the top movies, shows, genres, etc.
func (db *DB) populateWrappedTopContent(ctx context.Context, report *models.WrappedReport, userID int, startDate, endDate time.Time) error {
	// Top Movies
	moviesQuery := `
		SELECT
			ROW_NUMBER() OVER (ORDER BY SUM(COALESCE(play_duration, 0)) DESC) AS rank,
			title,
			rating_key,
			thumb,
			COUNT(*) AS watch_count,
			SUM(COALESCE(play_duration, 0)) / 3600.0 AS watch_hours
		FROM playback_events
		WHERE user_id = ? AND started_at >= ? AND started_at < ?
			AND media_type = 'movie'
		GROUP BY title, rating_key, thumb
		ORDER BY watch_hours DESC
		LIMIT 10
	`
	movies, err := db.queryWrappedContentRanks(ctx, moviesQuery, userID, startDate, endDate, "movie")
	if err != nil {
		return fmt.Errorf("failed to get top movies: %w", err)
	}
	report.TopMovies = movies

	// Top Shows (by grandparent_title for episodes)
	showsQuery := `
		SELECT
			ROW_NUMBER() OVER (ORDER BY SUM(COALESCE(play_duration, 0)) DESC) AS rank,
			grandparent_title AS title,
			grandparent_rating_key AS rating_key,
			grandparent_thumb AS thumb,
			COUNT(*) AS watch_count,
			SUM(COALESCE(play_duration, 0)) / 3600.0 AS watch_hours
		FROM playback_events
		WHERE user_id = ? AND started_at >= ? AND started_at < ?
			AND media_type = 'episode'
			AND grandparent_title IS NOT NULL AND grandparent_title != ''
		GROUP BY grandparent_title, grandparent_rating_key, grandparent_thumb
		ORDER BY watch_hours DESC
		LIMIT 10
	`
	shows, err := db.queryWrappedContentRanks(ctx, showsQuery, userID, startDate, endDate, "show")
	if err != nil {
		return fmt.Errorf("failed to get top shows: %w", err)
	}
	report.TopShows = shows

	// Top Genres
	genresQuery := `
		WITH genre_split AS (
			SELECT
				TRIM(UNNEST(STRING_SPLIT(genres, ','))) AS genre,
				COALESCE(play_duration, 0) AS duration
			FROM playback_events
			WHERE user_id = ? AND started_at >= ? AND started_at < ?
				AND genres IS NOT NULL AND genres != ''
		),
		genre_stats AS (
			SELECT
				genre,
				COUNT(*) AS watch_count,
				SUM(duration) / 3600.0 AS watch_hours
			FROM genre_split
			WHERE genre != ''
			GROUP BY genre
		),
		total_hours AS (
			SELECT SUM(watch_hours) AS total FROM genre_stats
		)
		SELECT
			ROW_NUMBER() OVER (ORDER BY watch_hours DESC) AS rank,
			gs.genre,
			gs.watch_count,
			gs.watch_hours,
			CASE WHEN th.total > 0 THEN (gs.watch_hours / th.total) * 100 ELSE 0 END AS percentage
		FROM genre_stats gs, total_hours th
		ORDER BY watch_hours DESC
		LIMIT 10
	`
	genres, err := db.queryWrappedGenreRanks(ctx, genresQuery, userID, startDate, endDate)
	if err != nil {
		return fmt.Errorf("failed to get top genres: %w", err)
	}
	report.TopGenres = genres

	return nil
}

// queryWrappedContentRanks executes a content ranking query and returns results.
func (db *DB) queryWrappedContentRanks(ctx context.Context, query string, userID int, startDate, endDate time.Time, mediaType string) ([]models.WrappedContentRank, error) {
	rows, err := db.conn.QueryContext(ctx, query, userID, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []models.WrappedContentRank
	for rows.Next() {
		var rank models.WrappedContentRank
		var ratingKey, thumb sql.NullString
		if err := rows.Scan(&rank.Rank, &rank.Title, &ratingKey, &thumb, &rank.WatchCount, &rank.WatchTime); err != nil {
			return nil, err
		}
		rank.RatingKey = ratingKey.String
		rank.Thumb = thumb.String
		rank.MediaType = mediaType
		results = append(results, rank)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if results == nil {
		results = []models.WrappedContentRank{}
	}
	return results, nil
}

// queryWrappedGenreRanks executes a genre ranking query and returns results.
func (db *DB) queryWrappedGenreRanks(ctx context.Context, query string, userID int, startDate, endDate time.Time) ([]models.WrappedGenreRank, error) {
	rows, err := db.conn.QueryContext(ctx, query, userID, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []models.WrappedGenreRank
	for rows.Next() {
		var rank models.WrappedGenreRank
		if err := rows.Scan(&rank.Rank, &rank.Genre, &rank.WatchCount, &rank.WatchTime, &rank.Percentage); err != nil {
			return nil, err
		}
		results = append(results, rank)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if results == nil {
		results = []models.WrappedGenreRank{}
	}
	return results, nil
}

// populateViewingByHour fills in viewing patterns by hour.
func (db *DB) populateViewingByHour(ctx context.Context, report *models.WrappedReport, userID int, startDate, endDate time.Time) error {
	hourQuery := `
		SELECT
			EXTRACT(HOUR FROM started_at)::INTEGER AS hour,
			COUNT(*) AS count
		FROM playback_events
		WHERE user_id = ? AND started_at >= ? AND started_at < ?
		GROUP BY hour
		ORDER BY hour
	`
	rows, err := db.conn.QueryContext(ctx, hourQuery, userID, startDate, endDate)
	if err != nil {
		return fmt.Errorf("failed to query viewing by hour: %w", err)
	}
	defer rows.Close()

	maxHourCount := 0
	for rows.Next() {
		var hour, count int
		if err := rows.Scan(&hour, &count); err != nil {
			return err
		}
		if hour >= 0 && hour < 24 {
			report.ViewingByHour[hour] = count
			if count > maxHourCount {
				maxHourCount = count
				report.PeakHour = hour
			}
		}
	}
	return rows.Err()
}

// populateViewingByDay fills in viewing patterns by day of week.
func (db *DB) populateViewingByDay(ctx context.Context, report *models.WrappedReport, userID int, startDate, endDate time.Time) error {
	dayQuery := `
		SELECT
			DAYOFWEEK(started_at) - 1 AS day_of_week,
			COUNT(*) AS count
		FROM playback_events
		WHERE user_id = ? AND started_at >= ? AND started_at < ?
		GROUP BY day_of_week
		ORDER BY day_of_week
	`
	rows, err := db.conn.QueryContext(ctx, dayQuery, userID, startDate, endDate)
	if err != nil {
		return fmt.Errorf("failed to query viewing by day: %w", err)
	}
	defer rows.Close()

	maxDayCount := 0
	for rows.Next() {
		var day, count int
		if err := rows.Scan(&day, &count); err != nil {
			return err
		}
		if day >= 0 && day < 7 {
			report.ViewingByDay[day] = count
			if count > maxDayCount {
				maxDayCount = count
				report.PeakDay = models.DayNames[day]
			}
		}
	}
	return rows.Err()
}

// populateViewingByMonth fills in viewing patterns by month.
func (db *DB) populateViewingByMonth(ctx context.Context, report *models.WrappedReport, userID int, startDate, endDate time.Time) error {
	monthQuery := `
		SELECT
			EXTRACT(MONTH FROM started_at)::INTEGER AS month,
			COUNT(*) AS count
		FROM playback_events
		WHERE user_id = ? AND started_at >= ? AND started_at < ?
		GROUP BY month
		ORDER BY month
	`
	rows, err := db.conn.QueryContext(ctx, monthQuery, userID, startDate, endDate)
	if err != nil {
		return fmt.Errorf("failed to query viewing by month: %w", err)
	}
	defer rows.Close()

	maxMonthCount := 0
	for rows.Next() {
		var month, count int
		if err := rows.Scan(&month, &count); err != nil {
			return err
		}
		if month >= 1 && month <= 12 {
			report.ViewingByMonth[month-1] = count
			if count > maxMonthCount {
				maxMonthCount = count
				report.PeakMonth = models.MonthNames[month]
			}
		}
	}
	return rows.Err()
}

// populateWrappedViewingPatterns fills in viewing patterns by hour, day, month.
//
//nolint:gocyclo // Multiple similar queries for different time dimensions
func (db *DB) populateWrappedViewingPatterns(ctx context.Context, report *models.WrappedReport, userID int, startDate, endDate time.Time) error {
	// Viewing by hour
	if err := db.populateViewingByHour(ctx, report, userID, startDate, endDate); err != nil {
		return err
	}

	// Viewing by day of week
	if err := db.populateViewingByDay(ctx, report, userID, startDate, endDate); err != nil {
		return err
	}

	// Viewing by month
	if err := db.populateViewingByMonth(ctx, report, userID, startDate, endDate); err != nil {
		return err
	}

	// Monthly trends with more detail
	trendsQuery := `
		SELECT
			EXTRACT(MONTH FROM started_at)::INTEGER AS month,
			COUNT(*) AS playback_count,
			SUM(COALESCE(play_duration, 0)) / 3600.0 AS watch_hours,
			COUNT(DISTINCT rating_key) AS unique_content
		FROM playback_events
		WHERE user_id = ? AND started_at >= ? AND started_at < ?
		GROUP BY month
		ORDER BY month
	`
	rows, err := db.conn.QueryContext(ctx, trendsQuery, userID, startDate, endDate)
	if err != nil {
		return fmt.Errorf("failed to query monthly trends: %w", err)
	}
	defer rows.Close()

	// Initialize all months
	report.MonthlyTrends = make([]models.WrappedMonthly, 12)
	for i := 0; i < 12; i++ {
		report.MonthlyTrends[i] = models.WrappedMonthly{
			Month:     i + 1,
			MonthName: models.MonthNames[i+1],
		}
	}

	for rows.Next() {
		var month, playbackCount, uniqueContent int
		var watchHours float64
		if err := rows.Scan(&month, &playbackCount, &watchHours, &uniqueContent); err != nil {
			return err
		}
		if month >= 1 && month <= 12 {
			report.MonthlyTrends[month-1].PlaybackCount = playbackCount
			report.MonthlyTrends[month-1].WatchTimeHours = watchHours
			report.MonthlyTrends[month-1].UniqueContent = uniqueContent
		}
	}

	return rows.Err()
}

// populateWrappedBingeStats fills in binge-watching statistics.
func (db *DB) populateWrappedBingeStats(ctx context.Context, report *models.WrappedReport, userID int, startDate, endDate time.Time) error {
	// Use existing binge detection logic adapted for wrapped reports
	query := `
		WITH episode_sessions AS (
			SELECT
				grandparent_title AS show_name,
				started_at,
				COALESCE(play_duration, 0) AS duration,
				LAG(started_at) OVER (PARTITION BY grandparent_title ORDER BY started_at) AS prev_started_at
			FROM playback_events
			WHERE user_id = ?
				AND started_at >= ? AND started_at < ?
				AND media_type = 'episode'
				AND grandparent_title IS NOT NULL AND grandparent_title != ''
		),
		session_markers AS (
			SELECT *,
				CASE
					WHEN prev_started_at IS NULL OR epoch(started_at - prev_started_at) > 21600
					THEN 1
					ELSE 0
				END AS is_new_session
			FROM episode_sessions
		),
		session_groups AS (
			SELECT *,
				SUM(is_new_session) OVER (PARTITION BY show_name ORDER BY started_at ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW) AS session_id
			FROM session_markers
		),
		binge_sessions AS (
			SELECT
				show_name,
				session_id,
				COUNT(*) AS episode_count,
				SUM(duration) / 3600.0 AS duration_hours,
				MIN(started_at) AS session_start
			FROM session_groups
			GROUP BY show_name, session_id
			HAVING COUNT(*) >= 3
		)
		SELECT
			COUNT(*) AS binge_count,
			COALESCE(SUM(duration_hours), 0) AS total_binge_hours,
			COALESCE(AVG(episode_count), 0) AS avg_episodes,
			(SELECT show_name FROM binge_sessions GROUP BY show_name ORDER BY COUNT(*) DESC LIMIT 1) AS favorite_show,
			(SELECT show_name FROM binge_sessions ORDER BY episode_count DESC LIMIT 1) AS longest_binge_show,
			(SELECT episode_count FROM binge_sessions ORDER BY episode_count DESC LIMIT 1) AS longest_binge_count,
			(SELECT duration_hours FROM binge_sessions ORDER BY episode_count DESC LIMIT 1) AS longest_binge_hours,
			(SELECT session_start FROM binge_sessions ORDER BY episode_count DESC LIMIT 1) AS longest_binge_date
		FROM binge_sessions
	`

	var bingeCount int
	var totalBingeHours, avgEpisodes sql.NullFloat64
	var favoriteShow, longestBingeShow sql.NullString
	var longestBingeCount sql.NullInt64
	var longestBingeHours sql.NullFloat64
	var longestBingeDate sql.NullTime

	err := db.conn.QueryRowContext(ctx, query, userID, startDate, endDate).Scan(
		&bingeCount,
		&totalBingeHours,
		&avgEpisodes,
		&favoriteShow,
		&longestBingeShow,
		&longestBingeCount,
		&longestBingeHours,
		&longestBingeDate,
	)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("failed to query binge stats: %w", err)
	}

	report.BingeSessions = bingeCount
	report.TotalBingeHours = totalBingeHours.Float64
	report.AvgBingeEpisodes = avgEpisodes.Float64
	report.FavoriteBingeShow = favoriteShow.String

	if longestBingeShow.Valid && longestBingeCount.Int64 > 0 {
		report.LongestBinge = &models.WrappedBingeInfo{
			ShowName:      longestBingeShow.String,
			EpisodeCount:  int(longestBingeCount.Int64),
			DurationHours: longestBingeHours.Float64,
		}
		if longestBingeDate.Valid {
			report.LongestBinge.Date = longestBingeDate.Time
		}
	}

	return nil
}

// populateWrappedQualityMetrics fills in streaming quality metrics.
func (db *DB) populateWrappedQualityMetrics(ctx context.Context, report *models.WrappedReport, userID int, startDate, endDate time.Time) error {
	query := `
		SELECT
			COALESCE(AVG(NULLIF(stream_bitrate, 0)) / 1000.0, 0) AS avg_bitrate_mbps,
			COALESCE(AVG(CASE WHEN transcode_decision = 'direct play' THEN 1.0 ELSE 0.0 END) * 100, 0) AS direct_play_rate,
			COALESCE(AVG(CASE WHEN video_dynamic_range IN ('HDR', 'HDR10', 'Dolby Vision') THEN 1.0 ELSE 0.0 END) * 100, 0) AS hdr_percent,
			COALESCE(AVG(CASE WHEN video_full_resolution LIKE '%4K%' OR video_resolution = '4k' OR video_width >= 3840 THEN 1.0 ELSE 0.0 END) * 100, 0) AS four_k_percent,
			(SELECT platform FROM playback_events WHERE user_id = ? AND started_at >= ? AND started_at < ? GROUP BY platform ORDER BY COUNT(*) DESC LIMIT 1) AS preferred_platform,
			(SELECT player FROM playback_events WHERE user_id = ? AND started_at >= ? AND started_at < ? GROUP BY player ORDER BY COUNT(*) DESC LIMIT 1) AS preferred_player
		FROM playback_events
		WHERE user_id = ? AND started_at >= ? AND started_at < ?
	`

	var avgBitrate, directPlayRate, hdrPercent, fourKPercent sql.NullFloat64
	var preferredPlatform, preferredPlayer sql.NullString

	err := db.conn.QueryRowContext(ctx, query,
		userID, startDate, endDate,
		userID, startDate, endDate,
		userID, startDate, endDate,
	).Scan(
		&avgBitrate,
		&directPlayRate,
		&hdrPercent,
		&fourKPercent,
		&preferredPlatform,
		&preferredPlayer,
	)
	if err != nil {
		return fmt.Errorf("failed to query quality metrics: %w", err)
	}

	report.AvgBitrateMbps = avgBitrate.Float64
	report.DirectPlayRate = directPlayRate.Float64
	report.HDRViewingPercent = hdrPercent.Float64
	report.FourKViewingPercent = fourKPercent.Float64
	report.PreferredPlatform = preferredPlatform.String
	report.PreferredPlayer = preferredPlayer.String

	return nil
}

// populateWrappedDiscoveryMetrics fills in content discovery metrics.
func (db *DB) populateWrappedDiscoveryMetrics(ctx context.Context, report *models.WrappedReport, userID int, startDate, endDate time.Time) error {
	// Count content watched for the first time this year
	query := `
		WITH first_watches AS (
			SELECT rating_key, MIN(started_at) AS first_watch
			FROM playback_events
			WHERE user_id = ?
			GROUP BY rating_key
		)
		SELECT COUNT(*)
		FROM first_watches
		WHERE first_watch >= ? AND first_watch < ?
	`
	var newContentCount int
	err := db.conn.QueryRowContext(ctx, query, userID, startDate, endDate).Scan(&newContentCount)
	if err != nil {
		return fmt.Errorf("failed to query new content count: %w", err)
	}
	report.NewContentCount = newContentCount

	// Calculate discovery rate
	if report.TotalPlaybacks > 0 {
		report.DiscoveryRate = float64(newContentCount) / float64(report.TotalPlaybacks) * 100
	}

	// First and last watch of the year
	firstLastQuery := `
		SELECT
			(SELECT title FROM playback_events WHERE user_id = ? AND started_at >= ? AND started_at < ? ORDER BY started_at ASC LIMIT 1) AS first_watch,
			(SELECT title FROM playback_events WHERE user_id = ? AND started_at >= ? AND started_at < ? ORDER BY started_at DESC LIMIT 1) AS last_watch
	`
	var firstWatch, lastWatch sql.NullString
	err = db.conn.QueryRowContext(ctx, firstLastQuery,
		userID, startDate, endDate,
		userID, startDate, endDate,
	).Scan(&firstWatch, &lastWatch)
	if err != nil {
		return fmt.Errorf("failed to query first/last watch: %w", err)
	}

	report.FirstWatchOfYear = firstWatch.String
	report.LastWatchOfYear = lastWatch.String

	return nil
}

// calculateWrappedAchievements determines which achievements a user has earned.
func (db *DB) calculateWrappedAchievements(report *models.WrappedReport) []models.WrappedAchievement {
	// Pre-calculate viewing totals for percentage-based achievements
	totalHourViews := sumIntSlice(report.ViewingByHour[:])
	totalDayViews := sumIntSlice(report.ViewingByDay[:])
	nightHours := report.ViewingByHour[22] + report.ViewingByHour[23] + report.ViewingByHour[0] + report.ViewingByHour[1] + report.ViewingByHour[2]
	morningHours := report.ViewingByHour[5] + report.ViewingByHour[6] + report.ViewingByHour[7] + report.ViewingByHour[8]
	weekendViews := report.ViewingByDay[0] + report.ViewingByDay[6]

	// Achievement definitions with conditions - table-driven for maintainability
	achievementChecks := []struct {
		condition   bool
		achievement models.WrappedAchievement
	}{
		{
			condition: report.BingeSessions >= 10,
			achievement: models.WrappedAchievement{
				ID:          models.AchievementBingemaster,
				Name:        "Bingemaster",
				Description: fmt.Sprintf("Completed %d binge sessions", report.BingeSessions),
				Icon:        "tv",
				Tier:        getTier(report.BingeSessions, 10, 25, 50),
			},
		},
		{
			condition: totalHourViews > 0 && float64(nightHours)/float64(totalHourViews) >= 0.5,
			achievement: models.WrappedAchievement{
				ID:          models.AchievementNightOwl,
				Name:        "Night Owl",
				Description: "50%+ of viewing after 10 PM",
				Icon:        "moon",
			},
		},
		{
			condition: totalHourViews > 0 && float64(morningHours)/float64(totalHourViews) >= 0.5,
			achievement: models.WrappedAchievement{
				ID:          models.AchievementEarlyBird,
				Name:        "Early Bird",
				Description: "50%+ of viewing before 9 AM",
				Icon:        "sun",
			},
		},
		{
			condition: totalDayViews > 0 && float64(weekendViews)/float64(totalDayViews) >= 0.6,
			achievement: models.WrappedAchievement{
				ID:          models.AchievementWeekendWarrior,
				Name:        "Weekend Warrior",
				Description: "60%+ of viewing on weekends",
				Icon:        "calendar",
			},
		},
		{
			condition: report.UniqueContentCount >= 50,
			achievement: models.WrappedAchievement{
				ID:          models.AchievementMovieBuff,
				Name:        "Movie Buff",
				Description: fmt.Sprintf("Watched %d unique pieces of content", report.UniqueContentCount),
				Icon:        "film",
				Tier:        getTier(report.UniqueContentCount, 50, 100, 200),
			},
		},
		{
			condition: report.DirectPlayRate >= 80,
			achievement: models.WrappedAchievement{
				ID:          models.AchievementQualityEnthusiast,
				Name:        "Quality Enthusiast",
				Description: fmt.Sprintf("%.0f%% direct play rate", report.DirectPlayRate),
				Icon:        "award",
			},
		},
		{
			condition: len(report.TopGenres) >= 10,
			achievement: models.WrappedAchievement{
				ID:          models.AchievementExplorer,
				Name:        "Explorer",
				Description: fmt.Sprintf("Watched content in %d genres", len(report.TopGenres)),
				Icon:        "compass",
			},
		},
		{
			condition: report.TotalWatchTimeHours >= 500,
			achievement: models.WrappedAchievement{
				ID:          models.AchievementMarathoner,
				Name:        "Marathoner",
				Description: fmt.Sprintf("Watched %.0f hours", report.TotalWatchTimeHours),
				Icon:        "clock",
				Tier:        getTier(int(report.TotalWatchTimeHours), 500, 1000, 2000),
			},
		},
		{
			condition: report.LongestStreakDays >= 30,
			achievement: models.WrappedAchievement{
				ID:          models.AchievementConsistent,
				Name:        "Consistent",
				Description: fmt.Sprintf("%d day viewing streak", report.LongestStreakDays),
				Icon:        "trending-up",
				Tier:        getTier(report.LongestStreakDays, 30, 60, 100),
			},
		},
	}

	// Collect earned achievements
	achievements := make([]models.WrappedAchievement, 0, len(achievementChecks))
	for _, check := range achievementChecks {
		if check.condition {
			achievements = append(achievements, check.achievement)
		}
	}

	return achievements
}

// sumIntSlice sums all values in an int slice.
func sumIntSlice(slice []int) int {
	total := 0
	for _, v := range slice {
		total += v
	}
	return total
}

// getTier returns a tier based on value thresholds.
// The bronze threshold parameter is kept for API consistency but not used
// since bronze is the default tier when value doesn't meet silver/gold thresholds.
func getTier(value, _ /* bronze */, silver, gold int) string {
	if value >= gold {
		return "gold"
	}
	if value >= silver {
		return "silver"
	}
	return "bronze"
}

// populateWrappedPercentiles calculates how the user compares to all users.
func (db *DB) populateWrappedPercentiles(ctx context.Context, report *models.WrappedReport, year int) error {
	startDate := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(year+1, 1, 1, 0, 0, 0, 0, time.UTC)

	query := `
		WITH user_stats AS (
			SELECT
				user_id,
				SUM(COALESCE(play_duration, 0)) / 3600.0 AS total_hours,
				COUNT(DISTINCT rating_key) AS unique_content,
				AVG(CASE WHEN percent_complete >= 90 THEN 1.0 ELSE 0.0 END) * 100 AS completion_rate
			FROM playback_events
			WHERE started_at >= ? AND started_at < ?
			GROUP BY user_id
		),
		percentiles AS (
			SELECT
				user_id,
				PERCENT_RANK() OVER (ORDER BY total_hours) * 100 AS watch_time_pct,
				PERCENT_RANK() OVER (ORDER BY unique_content) * 100 AS content_pct,
				PERCENT_RANK() OVER (ORDER BY completion_rate) * 100 AS completion_pct
			FROM user_stats
		)
		SELECT
			COALESCE(watch_time_pct, 0)::INTEGER,
			COALESCE(content_pct, 0)::INTEGER,
			COALESCE(completion_pct, 0)::INTEGER
		FROM percentiles
		WHERE user_id = ?
	`

	var watchTimePct, contentPct, completionPct int
	err := db.conn.QueryRowContext(ctx, query, startDate, endDate, report.UserID).Scan(
		&watchTimePct,
		&contentPct,
		&completionPct,
	)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("failed to query percentiles: %w", err)
	}

	report.Percentiles = models.WrappedPercentiles{
		WatchTime:      watchTimePct,
		ContentCount:   contentPct,
		CompletionRate: completionPct,
		BingeCount:     0, // Would need separate binge query
		EarlyAdopter:   0, // Would need separate early adopter query
	}

	return nil
}

// saveWrappedReport saves a wrapped report to the database.
func (db *DB) saveWrappedReport(ctx context.Context, report *models.WrappedReport) error {
	// Marshal JSON fields using table-driven approach to reduce repetition
	type jsonField struct {
		value     interface{}
		fieldName string
	}

	fields := []jsonField{
		{report.LongestBinge, "longest_binge"},
		{report.ViewingByHour, "viewing_by_hour"},
		{report.ViewingByDay, "viewing_by_day"},
		{report.ViewingByMonth, "viewing_by_month"},
		{report.MonthlyTrends, "monthly_trends"},
		{report.TopMovies, "top_movies"},
		{report.TopShows, "top_shows"},
		{report.TopEpisodes, "top_episodes"},
		{report.TopGenres, "top_genres"},
		{report.TopActors, "top_actors"},
		{report.TopDirectors, "top_directors"},
		{report.Achievements, "achievements"},
		{report.Percentiles, "percentiles"},
	}

	jsonData := make([][]byte, len(fields))
	for i, f := range fields {
		data, err := marshalJSONField(f.value, f.fieldName)
		if err != nil {
			return err
		}
		jsonData[i] = data
	}

	// Extract individual JSON byte slices for clarity in the query
	longestBingeJSON := jsonData[0]
	viewingByHourJSON := jsonData[1]
	viewingByDayJSON := jsonData[2]
	viewingByMonthJSON := jsonData[3]
	monthlyTrendsJSON := jsonData[4]
	topMoviesJSON := jsonData[5]
	topShowsJSON := jsonData[6]
	topEpisodesJSON := jsonData[7]
	topGenresJSON := jsonData[8]
	topActorsJSON := jsonData[9]
	topDirectorsJSON := jsonData[10]
	achievementsJSON := jsonData[11]
	percentilesJSON := jsonData[12]

	query := `
		INSERT INTO wrapped_reports (
			id, user_id, username, year,
			total_watch_time_hours, total_playbacks, unique_content_count,
			completion_rate, days_active, longest_streak_days, avg_daily_watch_minutes,
			binge_sessions, total_binge_hours, favorite_binge_show, avg_binge_episodes,
			longest_binge_json,
			avg_bitrate_mbps, direct_play_rate, hdr_viewing_percent, four_k_viewing_percent,
			preferred_platform, preferred_player,
			new_content_count, discovery_rate, first_watch_of_year, last_watch_of_year,
			peak_hour, peak_day, peak_month,
			viewing_by_hour, viewing_by_day, viewing_by_month, monthly_trends,
			top_movies, top_shows, top_episodes, top_genres, top_actors, top_directors,
			achievements, percentiles,
			share_token, shareable_text, generated_at
		) VALUES (
			?, ?, ?, ?,
			?, ?, ?,
			?, ?, ?, ?,
			?, ?, ?, ?,
			?,
			?, ?, ?, ?,
			?, ?,
			?, ?, ?, ?,
			?, ?, ?,
			?, ?, ?, ?,
			?, ?, ?, ?, ?, ?,
			?, ?,
			?, ?, ?
		)
		ON CONFLICT (user_id, year) DO UPDATE SET
			total_watch_time_hours = EXCLUDED.total_watch_time_hours,
			total_playbacks = EXCLUDED.total_playbacks,
			unique_content_count = EXCLUDED.unique_content_count,
			completion_rate = EXCLUDED.completion_rate,
			days_active = EXCLUDED.days_active,
			longest_streak_days = EXCLUDED.longest_streak_days,
			avg_daily_watch_minutes = EXCLUDED.avg_daily_watch_minutes,
			binge_sessions = EXCLUDED.binge_sessions,
			total_binge_hours = EXCLUDED.total_binge_hours,
			favorite_binge_show = EXCLUDED.favorite_binge_show,
			avg_binge_episodes = EXCLUDED.avg_binge_episodes,
			longest_binge_json = EXCLUDED.longest_binge_json,
			avg_bitrate_mbps = EXCLUDED.avg_bitrate_mbps,
			direct_play_rate = EXCLUDED.direct_play_rate,
			hdr_viewing_percent = EXCLUDED.hdr_viewing_percent,
			four_k_viewing_percent = EXCLUDED.four_k_viewing_percent,
			preferred_platform = EXCLUDED.preferred_platform,
			preferred_player = EXCLUDED.preferred_player,
			new_content_count = EXCLUDED.new_content_count,
			discovery_rate = EXCLUDED.discovery_rate,
			first_watch_of_year = EXCLUDED.first_watch_of_year,
			last_watch_of_year = EXCLUDED.last_watch_of_year,
			peak_hour = EXCLUDED.peak_hour,
			peak_day = EXCLUDED.peak_day,
			peak_month = EXCLUDED.peak_month,
			viewing_by_hour = EXCLUDED.viewing_by_hour,
			viewing_by_day = EXCLUDED.viewing_by_day,
			viewing_by_month = EXCLUDED.viewing_by_month,
			monthly_trends = EXCLUDED.monthly_trends,
			top_movies = EXCLUDED.top_movies,
			top_shows = EXCLUDED.top_shows,
			top_episodes = EXCLUDED.top_episodes,
			top_genres = EXCLUDED.top_genres,
			top_actors = EXCLUDED.top_actors,
			top_directors = EXCLUDED.top_directors,
			achievements = EXCLUDED.achievements,
			percentiles = EXCLUDED.percentiles,
			shareable_text = EXCLUDED.shareable_text,
			generated_at = EXCLUDED.generated_at
	`

	_, err := db.conn.ExecContext(ctx, query,
		report.ID, report.UserID, report.Username, report.Year,
		report.TotalWatchTimeHours, report.TotalPlaybacks, report.UniqueContentCount,
		report.CompletionRate, report.DaysActive, report.LongestStreakDays, report.AvgDailyWatchMinutes,
		report.BingeSessions, report.TotalBingeHours, report.FavoriteBingeShow, report.AvgBingeEpisodes,
		string(longestBingeJSON),
		report.AvgBitrateMbps, report.DirectPlayRate, report.HDRViewingPercent, report.FourKViewingPercent,
		report.PreferredPlatform, report.PreferredPlayer,
		report.NewContentCount, report.DiscoveryRate, report.FirstWatchOfYear, report.LastWatchOfYear,
		report.PeakHour, report.PeakDay, report.PeakMonth,
		string(viewingByHourJSON), string(viewingByDayJSON), string(viewingByMonthJSON), string(monthlyTrendsJSON),
		string(topMoviesJSON), string(topShowsJSON), string(topEpisodesJSON), string(topGenresJSON), string(topActorsJSON), string(topDirectorsJSON),
		string(achievementsJSON), string(percentilesJSON),
		report.ShareToken, report.ShareableText, report.GeneratedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to insert wrapped report: %w", err)
	}

	return nil
}

// GetWrappedLeaderboard retrieves the wrapped leaderboard for a year.
func (db *DB) GetWrappedLeaderboard(ctx context.Context, year int, limit int) ([]models.WrappedLeaderboardEntry, error) {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	if limit <= 0 {
		limit = 10
	}

	query := `
		SELECT
			ROW_NUMBER() OVER (ORDER BY total_watch_time_hours DESC) AS rank,
			user_id,
			username,
			total_watch_time_hours,
			total_playbacks,
			unique_content_count,
			completion_rate
		FROM wrapped_reports
		WHERE year = ?
		ORDER BY total_watch_time_hours DESC
		LIMIT ?
	`

	rows, err := db.conn.QueryContext(ctx, query, year, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query leaderboard: %w", err)
	}
	defer rows.Close()

	var results []models.WrappedLeaderboardEntry
	for rows.Next() {
		var entry models.WrappedLeaderboardEntry
		if err := rows.Scan(
			&entry.Rank,
			&entry.UserID,
			&entry.Username,
			&entry.TotalWatchTimeHours,
			&entry.TotalPlaybacks,
			&entry.UniqueContent,
			&entry.CompletionRate,
		); err != nil {
			return nil, fmt.Errorf("failed to scan leaderboard entry: %w", err)
		}
		results = append(results, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if results == nil {
		results = []models.WrappedLeaderboardEntry{}
	}

	return results, nil
}

// GetWrappedServerStats retrieves server-wide wrapped statistics for a year.
func (db *DB) GetWrappedServerStats(ctx context.Context, year int) (*models.WrappedServerStats, error) {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	startDate := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(year+1, 1, 1, 0, 0, 0, 0, time.UTC)

	query := `
		SELECT
			COALESCE(COUNT(DISTINCT user_id), 0) AS total_users,
			COALESCE(SUM(COALESCE(play_duration, 0)) / 3600.0, 0) AS total_hours,
			COALESCE(COUNT(*), 0) AS total_playbacks,
			COALESCE(COUNT(DISTINCT rating_key), 0) AS unique_content,
			COALESCE(AVG(CASE WHEN percent_complete >= 90 THEN 1.0 ELSE 0.0 END) * 100, 0) AS avg_completion
		FROM playback_events
		WHERE started_at >= ? AND started_at < ?
	`

	stats := &models.WrappedServerStats{
		Year:        year,
		GeneratedAt: time.Now(),
	}

	var totalUsers, totalPlaybacks, uniqueContent int
	var totalHours, avgCompletion float64

	err := db.conn.QueryRowContext(ctx, query, startDate, endDate).Scan(
		&totalUsers,
		&totalHours,
		&totalPlaybacks,
		&uniqueContent,
		&avgCompletion,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query server stats: %w", err)
	}

	stats.TotalUsers = totalUsers
	stats.TotalWatchTimeHours = totalHours
	stats.TotalPlaybacks = totalPlaybacks
	stats.UniqueContentWatched = uniqueContent
	stats.AvgCompletionRate = avgCompletion

	// Get top movie - error ignored as empty result is acceptable
	movieQuery := `
		SELECT title
		FROM playback_events
		WHERE started_at >= ? AND started_at < ? AND media_type = 'movie'
		GROUP BY title
		ORDER BY COUNT(*) DESC
		LIMIT 1
	`
	var topMovie sql.NullString
	//nolint:errcheck // No error if no data - empty string is acceptable
	db.conn.QueryRowContext(ctx, movieQuery, startDate, endDate).Scan(&topMovie)
	stats.TopMovie = topMovie.String

	// Get top show - error ignored as empty result is acceptable
	showQuery := `
		SELECT grandparent_title
		FROM playback_events
		WHERE started_at >= ? AND started_at < ? AND media_type = 'episode' AND grandparent_title IS NOT NULL
		GROUP BY grandparent_title
		ORDER BY COUNT(*) DESC
		LIMIT 1
	`
	var topShow sql.NullString
	//nolint:errcheck // No error if no data - empty string is acceptable
	db.conn.QueryRowContext(ctx, showQuery, startDate, endDate).Scan(&topShow)
	stats.TopShow = topShow.String

	// Get top genre - error ignored as empty result is acceptable
	genreQuery := `
		SELECT TRIM(UNNEST(STRING_SPLIT(genres, ','))) AS genre
		FROM playback_events
		WHERE started_at >= ? AND started_at < ? AND genres IS NOT NULL
		GROUP BY genre
		ORDER BY COUNT(*) DESC
		LIMIT 1
	`
	var topGenre sql.NullString
	//nolint:errcheck // No error if no data - empty string is acceptable
	db.conn.QueryRowContext(ctx, genreQuery, startDate, endDate).Scan(&topGenre)
	stats.TopGenre = topGenre.String

	// Get peak month - error ignored as empty result is acceptable
	monthQuery := `
		SELECT EXTRACT(MONTH FROM started_at)::INTEGER AS month
		FROM playback_events
		WHERE started_at >= ? AND started_at < ?
		GROUP BY month
		ORDER BY COUNT(*) DESC
		LIMIT 1
	`
	var peakMonth int
	//nolint:errcheck // No error if no data - zero is acceptable
	db.conn.QueryRowContext(ctx, monthQuery, startDate, endDate).Scan(&peakMonth)
	if peakMonth >= 1 && peakMonth <= 12 {
		stats.PeakMonth = models.MonthNames[peakMonth]
	}

	return stats, nil
}

// GetUsersWithPlaybacksInYear returns user IDs that have playbacks in the given year.
func (db *DB) GetUsersWithPlaybacksInYear(ctx context.Context, year int) ([]int, error) {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	startDate := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(year+1, 1, 1, 0, 0, 0, 0, time.UTC)

	query := `
		SELECT DISTINCT user_id
		FROM playback_events
		WHERE started_at >= ? AND started_at < ?
		ORDER BY user_id
	`

	rows, err := db.conn.QueryContext(ctx, query, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()

	var userIDs []int
	for rows.Next() {
		var userID int
		if err := rows.Scan(&userID); err != nil {
			return nil, err
		}
		userIDs = append(userIDs, userID)
	}

	return userIDs, rows.Err()
}

// Helper functions

func generateWrappedID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Crypto random failure is a critical system error
		panic("failed to generate random bytes: " + err.Error())
	}
	return fmt.Sprintf("wrapped_%s", hex.EncodeToString(b))
}

func generateShareToken() string {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		// Crypto random failure is a critical system error
		panic("failed to generate random bytes: " + err.Error())
	}
	return hex.EncodeToString(b)
}

func generateShareableText(report *models.WrappedReport) string {
	text := fmt.Sprintf("My %d Wrapped:\n", report.Year)
	text += fmt.Sprintf("- %.0f hours watched\n", report.TotalWatchTimeHours)
	text += fmt.Sprintf("- %d playbacks\n", report.TotalPlaybacks)
	text += fmt.Sprintf("- %d unique titles\n", report.UniqueContentCount)

	if len(report.TopGenres) > 0 {
		text += fmt.Sprintf("- Top genre: %s\n", report.TopGenres[0].Genre)
	}
	if len(report.TopShows) > 0 {
		text += fmt.Sprintf("- Top show: %s\n", report.TopShows[0].Title)
	}
	if len(report.TopMovies) > 0 {
		text += fmt.Sprintf("- Top movie: %s\n", report.TopMovies[0].Title)
	}

	return text
}
