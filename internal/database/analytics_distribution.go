// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package database

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/tomtom215/cartographus/internal/models"
)

func (db *DB) GetTopUsers(ctx context.Context, filter LocationStatsFilter, limit int) ([]models.UserActivity, error) {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	baseQuery := `
	SELECT
		username,
		COUNT(*) as playback_count,
		CAST(SUM(COALESCE(play_duration, 0)) AS INTEGER) as total_duration_minutes,
		AVG(percent_complete) as avg_completion,
		COUNT(DISTINCT title) as unique_media
	FROM playback_events
	WHERE 1=1`

	query, args := newQueryBuilder(baseQuery).
		addStandardFilters(filter).
		addLimit(limit).
		build("GROUP BY username ORDER BY playback_count DESC LIMIT ?")

	scanUser := func(rows *sql.Rows) (models.UserActivity, error) {
		var u models.UserActivity
		err := rows.Scan(&u.Username, &u.PlaybackCount, &u.TotalDuration, &u.AvgCompletion, &u.UniqueMedia)
		return u, err
	}

	users, err := queryAndScan(ctx, db.conn, query, args, scanUser)
	if err != nil {
		return nil, fmt.Errorf("failed to query top users: %w", err)
	}

	return users, nil
}

// GetMediaTypeDistribution retrieves playback statistics by media type
func (db *DB) GetMediaTypeDistribution(ctx context.Context, filter LocationStatsFilter) ([]models.MediaTypeStats, error) {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	baseQuery := `
	SELECT
		media_type,
		COUNT(*) as playback_count,
		COUNT(DISTINCT user_id) as unique_users
	FROM playback_events
	WHERE media_type IS NOT NULL AND media_type != ''`

	query, args := newQueryBuilder(baseQuery).
		addStandardFilters(filter).
		build("GROUP BY media_type ORDER BY playback_count DESC")

	scanMediaType := func(rows *sql.Rows) (models.MediaTypeStats, error) {
		var m models.MediaTypeStats
		err := rows.Scan(&m.MediaType, &m.PlaybackCount, &m.UniqueUsers)
		return m, err
	}

	mediaTypes, err := queryAndScan(ctx, db.conn, query, args, scanMediaType)
	if err != nil {
		return nil, fmt.Errorf("failed to query media type distribution: %w", err)
	}

	return mediaTypes, nil
}

// addJoinedTableFilters adds date, user, and media type filters for queries with playback_events aliased as 'p'
func addJoinedTableFilters(qb *queryBuilder, filter LocationStatsFilter) *queryBuilder {
	if filter.StartDate != nil {
		qb.addFilter("p.started_at >= ?", *filter.StartDate)
	}
	if filter.EndDate != nil {
		qb.addFilter("p.started_at <= ?", *filter.EndDate)
	}
	qb.addUsersFilter(filter.Users) // Uses username which exists in both tables
	if len(filter.MediaTypes) > 0 {
		placeholders := make([]string, len(filter.MediaTypes))
		mediaArgs := make([]interface{}, len(filter.MediaTypes))
		for i, mt := range filter.MediaTypes {
			placeholders[i] = "?"
			mediaArgs[i] = mt
		}
		qb.addFilter(fmt.Sprintf("p.media_type IN (%s)", join(placeholders, ",")), mediaArgs...)
	}
	return qb
}

// GetTopCities retrieves top cities by playback count
func (db *DB) GetTopCities(ctx context.Context, filter LocationStatsFilter, limit int) ([]models.CityStats, error) {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	baseQuery := `
	SELECT
		g.city,
		g.country,
		COUNT(*) as playback_count,
		COUNT(DISTINCT p.user_id) as unique_users
	FROM playback_events p
	JOIN geolocations g ON p.ip_address = g.ip_address
	WHERE g.city IS NOT NULL AND g.city != ''`

	qb := addJoinedTableFilters(newQueryBuilder(baseQuery), filter)
	query, args := qb.addLimit(limit).
		build("GROUP BY g.city, g.country ORDER BY playback_count DESC LIMIT ?")

	scanCity := func(rows *sql.Rows) (models.CityStats, error) {
		var c models.CityStats
		err := rows.Scan(&c.City, &c.Country, &c.PlaybackCount, &c.UniqueUsers)
		return c, err
	}

	cities, err := queryAndScan(ctx, db.conn, query, args, scanCity)
	if err != nil {
		return nil, fmt.Errorf("failed to query top cities: %w", err)
	}

	return cities, nil
}

// GetTopCountries retrieves top countries by playback count (filtered version)
func (db *DB) GetTopCountries(ctx context.Context, filter LocationStatsFilter, limit int) ([]models.CountryStats, error) {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	baseQuery := `
	SELECT
		g.country,
		COUNT(*) as playback_count,
		COUNT(DISTINCT p.user_id) as unique_users
	FROM playback_events p
	JOIN geolocations g ON p.ip_address = g.ip_address
	WHERE 1=1`

	qb := addJoinedTableFilters(newQueryBuilder(baseQuery), filter)
	query, args := qb.addLimit(limit).
		build("GROUP BY g.country ORDER BY playback_count DESC LIMIT ?")

	scanCountry := func(rows *sql.Rows) (models.CountryStats, error) {
		var c models.CountryStats
		err := rows.Scan(&c.Country, &c.PlaybackCount, &c.UniqueUsers)
		return c, err
	}

	countries, err := queryAndScan(ctx, db.conn, query, args, scanCountry)
	if err != nil {
		return nil, fmt.Errorf("failed to query top countries: %w", err)
	}

	return countries, nil
}

// GetPlatformDistribution retrieves playback statistics by platform
func (db *DB) GetPlatformDistribution(ctx context.Context, filter LocationStatsFilter) ([]models.PlatformStats, error) {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	baseQuery := `
	SELECT
		platform,
		COUNT(*) as playback_count,
		COUNT(DISTINCT user_id) as unique_users
	FROM playback_events
	WHERE platform IS NOT NULL AND platform != ''`

	query, args := newQueryBuilder(baseQuery).
		addStandardFilters(filter).
		build("GROUP BY platform ORDER BY playback_count DESC")

	scanPlatform := func(rows *sql.Rows) (models.PlatformStats, error) {
		var p models.PlatformStats
		err := rows.Scan(&p.Platform, &p.PlaybackCount, &p.UniqueUsers)
		return p, err
	}

	platforms, err := queryAndScan(ctx, db.conn, query, args, scanPlatform)
	if err != nil {
		return nil, fmt.Errorf("failed to query platform distribution: %w", err)
	}

	return platforms, nil
}

// GetPlayerDistribution retrieves playback statistics by player
func (db *DB) GetPlayerDistribution(ctx context.Context, filter LocationStatsFilter) ([]models.PlayerStats, error) {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	baseQuery := `
	SELECT
		player,
		COUNT(*) as playback_count,
		COUNT(DISTINCT user_id) as unique_users
	FROM playback_events
	WHERE player IS NOT NULL AND player != ''`

	query, args := newQueryBuilder(baseQuery).
		addStandardFilters(filter).
		build("GROUP BY player ORDER BY playback_count DESC")

	scanPlayer := func(rows *sql.Rows) (models.PlayerStats, error) {
		var p models.PlayerStats
		err := rows.Scan(&p.Player, &p.PlaybackCount, &p.UniqueUsers)
		return p, err
	}

	players, err := queryAndScan(ctx, db.conn, query, args, scanPlayer)
	if err != nil {
		return nil, fmt.Errorf("failed to query player distribution: %w", err)
	}

	return players, nil
}

// GetContentCompletionStats analyzes content completion patterns including completion distribution
// across 5 buckets (0-25%, 25-50%, 50-75%, 75-99%, 100%), average completion rates, and fully watched
// content percentage for engagement optimization.
func (db *DB) GetContentCompletionStats(ctx context.Context, filter LocationStatsFilter) (models.ContentCompletionStats, error) {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	// Define completion buckets
	buckets := initializeCompletionBuckets()

	// Query and scan completion distribution
	buckets, totalPlaybacks, totalCompletion, fullyWatched, err := db.queryCompletionDistribution(ctx, filter, buckets)
	if err != nil {
		return models.ContentCompletionStats{}, err
	}

	// Calculate overall statistics
	avgCompletion, fullyWatchedPct := calculateCompletionStats(totalPlaybacks, totalCompletion, fullyWatched)

	return models.ContentCompletionStats{
		Buckets:         buckets,
		TotalPlaybacks:  totalPlaybacks,
		AvgCompletion:   avgCompletion,
		FullyWatched:    fullyWatched,
		FullyWatchedPct: fullyWatchedPct,
	}, nil
}

// initializeCompletionBuckets creates the standard 5-bucket completion distribution structure.
func initializeCompletionBuckets() []models.CompletionBucket {
	return []models.CompletionBucket{
		{Bucket: "0-25%", MinPercent: 0, MaxPercent: 25},
		{Bucket: "25-50%", MinPercent: 25, MaxPercent: 50},
		{Bucket: "50-75%", MinPercent: 50, MaxPercent: 75},
		{Bucket: "75-99%", MinPercent: 75, MaxPercent: 99},
		{Bucket: "100%", MinPercent: 100, MaxPercent: 100},
	}
}

// queryCompletionDistribution queries completion bucket distribution with filter support,
// scans results, and populates bucket statistics including counts and averages.
func (db *DB) queryCompletionDistribution(ctx context.Context, filter LocationStatsFilter, buckets []models.CompletionBucket) ([]models.CompletionBucket, int, float64, int, error) {
	query := `
	SELECT
		CASE
			WHEN percent_complete = 100 THEN '100%'
			WHEN percent_complete >= 75 THEN '75-99%'
			WHEN percent_complete >= 50 THEN '50-75%'
			WHEN percent_complete >= 25 THEN '25-50%'
			ELSE '0-25%'
		END as bucket,
		COUNT(*) as playback_count,
		AVG(percent_complete) as avg_completion
	FROM playback_events
	WHERE percent_complete IS NOT NULL`

	args := []interface{}{}

	if filter.StartDate != nil {
		query += " AND started_at >= ?"
		args = append(args, *filter.StartDate)
	}
	if filter.EndDate != nil {
		query += " AND started_at <= ?"
		args = append(args, *filter.EndDate)
	}
	if len(filter.Users) > 0 {
		placeholders := make([]string, len(filter.Users))
		for i, user := range filter.Users {
			placeholders[i] = "?"
			args = append(args, user)
		}
		query += fmt.Sprintf(" AND username IN (%s)", join(placeholders, ","))
	}
	if len(filter.MediaTypes) > 0 {
		placeholders := make([]string, len(filter.MediaTypes))
		for i, mediaType := range filter.MediaTypes {
			placeholders[i] = "?"
			args = append(args, mediaType)
		}
		query += fmt.Sprintf(" AND media_type IN (%s)", join(placeholders, ","))
	}

	query += " GROUP BY bucket ORDER BY MIN(percent_complete)"

	rows, err := db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, 0, 0, fmt.Errorf("failed to query completion stats: %w", err)
	}
	defer rows.Close()

	totalPlaybacks := 0
	totalCompletion := 0.0
	fullyWatched := 0

	for rows.Next() {
		var bucketName string
		var count int
		var avgCompletion float64

		if err := rows.Scan(&bucketName, &count, &avgCompletion); err != nil {
			return nil, 0, 0, 0, fmt.Errorf("failed to scan completion bucket: %w", err)
		}

		for i, b := range buckets {
			if b.Bucket == bucketName {
				buckets[i].PlaybackCount = count
				buckets[i].AvgCompletion = avgCompletion
				break
			}
		}

		totalPlaybacks += count
		totalCompletion += float64(count) * avgCompletion

		if bucketName == "100%" {
			fullyWatched = count
		}
	}

	if err = rows.Err(); err != nil {
		return nil, 0, 0, 0, fmt.Errorf("error iterating completion buckets: %w", err)
	}

	return buckets, totalPlaybacks, totalCompletion, fullyWatched, nil
}

// calculateCompletionStats computes overall average completion rate and fully watched percentage
// from aggregated bucket statistics.
func calculateCompletionStats(totalPlaybacks int, totalCompletion float64, fullyWatched int) (float64, float64) {
	avgCompletion := 0.0
	if totalPlaybacks > 0 {
		avgCompletion = totalCompletion / float64(totalPlaybacks)
	}

	fullyWatchedPct := 0.0
	if totalPlaybacks > 0 {
		fullyWatchedPct = float64(fullyWatched) / float64(totalPlaybacks) * 100
	}

	return avgCompletion, fullyWatchedPct
}

// GetTranscodeDistribution retrieves playback statistics by transcode decision
func (db *DB) GetTranscodeDistribution(ctx context.Context, filter LocationStatsFilter) ([]models.TranscodeStats, error) {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	baseQuery := `
	SELECT
		COALESCE(transcode_decision, 'Unknown') as transcode_decision,
		COUNT(*) as playback_count
	FROM playback_events
	WHERE 1=1`

	query, args := newQueryBuilder(baseQuery).
		addStandardFilters(filter).
		build("GROUP BY transcode_decision ORDER BY playback_count DESC")

	scanTranscode := func(rows *sql.Rows) (models.TranscodeStats, error) {
		var ts models.TranscodeStats
		err := rows.Scan(&ts.TranscodeDecision, &ts.PlaybackCount)
		return ts, err
	}

	results, err := queryAndScan(ctx, db.conn, query, args, scanTranscode)
	if err != nil {
		return nil, fmt.Errorf("failed to query transcode distribution: %w", err)
	}

	// Calculate percentages
	total := 0
	for _, r := range results {
		total += r.PlaybackCount
	}
	if total > 0 {
		for i := range results {
			results[i].Percentage = float64(results[i].PlaybackCount) / float64(total) * 100
		}
	}

	return results, nil
}

// GetResolutionDistribution retrieves playback statistics by video resolution
func (db *DB) GetResolutionDistribution(ctx context.Context, filter LocationStatsFilter) ([]models.ResolutionStats, error) {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	baseQuery := `
	SELECT
		COALESCE(video_resolution, 'Unknown') as video_resolution,
		COUNT(*) as playback_count
	FROM playback_events
	WHERE 1=1`

	query, args := newQueryBuilder(baseQuery).
		addStandardFilters(filter).
		build("GROUP BY video_resolution ORDER BY playback_count DESC")

	scanResolution := func(rows *sql.Rows) (models.ResolutionStats, error) {
		var rs models.ResolutionStats
		err := rows.Scan(&rs.VideoResolution, &rs.PlaybackCount)
		return rs, err
	}

	results, err := queryAndScan(ctx, db.conn, query, args, scanResolution)
	if err != nil {
		return nil, fmt.Errorf("failed to query resolution distribution: %w", err)
	}

	// Calculate percentages
	total := 0
	for _, r := range results {
		total += r.PlaybackCount
	}
	if total > 0 {
		for i := range results {
			results[i].Percentage = float64(results[i].PlaybackCount) / float64(total) * 100
		}
	}

	return results, nil
}

// GetCodecDistribution retrieves playback statistics by codec combination
func (db *DB) GetCodecDistribution(ctx context.Context, filter LocationStatsFilter) ([]models.CodecStats, error) {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	baseQuery := `
	SELECT
		COALESCE(video_codec, 'Unknown') as video_codec,
		COALESCE(audio_codec, 'Unknown') as audio_codec,
		COUNT(*) as playback_count
	FROM playback_events
	WHERE 1=1`

	query, args := newQueryBuilder(baseQuery).
		addStandardFilters(filter).
		build("GROUP BY video_codec, audio_codec ORDER BY playback_count DESC LIMIT 10")

	scanCodec := func(rows *sql.Rows) (models.CodecStats, error) {
		var cs models.CodecStats
		err := rows.Scan(&cs.VideoCodec, &cs.AudioCodec, &cs.PlaybackCount)
		return cs, err
	}

	results, err := queryAndScan(ctx, db.conn, query, args, scanCodec)
	if err != nil {
		return nil, fmt.Errorf("failed to query codec distribution: %w", err)
	}

	// Calculate percentages
	total := 0
	for _, r := range results {
		total += r.PlaybackCount
	}
	if total > 0 {
		for i := range results {
			results[i].Percentage = float64(results[i].PlaybackCount) / float64(total) * 100
		}
	}

	return results, nil
}

// GetLibraryStats retrieves playback statistics by library
func (db *DB) GetLibraryStats(ctx context.Context, filter LocationStatsFilter) ([]models.LibraryStats, error) {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	baseQuery := `
	SELECT
		COALESCE(section_id, 0) as section_id,
		COALESCE(library_name, 'Unknown') as library_name,
		COUNT(*) as playback_count,
		COUNT(DISTINCT user_id) as unique_users,
		CAST(SUM(COALESCE(play_duration, 0)) AS INTEGER) as total_duration,
		AVG(percent_complete) as avg_completion
	FROM playback_events
	WHERE library_name IS NOT NULL AND library_name != ''`

	query, args := newQueryBuilder(baseQuery).
		addStandardFilters(filter).
		build("GROUP BY section_id, library_name ORDER BY playback_count DESC")

	scanLibrary := func(rows *sql.Rows) (models.LibraryStats, error) {
		var ls models.LibraryStats
		err := rows.Scan(&ls.SectionID, &ls.LibraryName, &ls.PlaybackCount, &ls.UniqueUsers, &ls.TotalDuration, &ls.AvgCompletion)
		return ls, err
	}

	libraries, err := queryAndScan(ctx, db.conn, query, args, scanLibrary)
	if err != nil {
		return nil, fmt.Errorf("failed to query library stats: %w", err)
	}

	return libraries, nil
}

// GetRatingDistribution retrieves playback statistics by content rating
func (db *DB) GetRatingDistribution(ctx context.Context, filter LocationStatsFilter) ([]models.RatingStats, error) {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	baseQuery := `
	SELECT
		COALESCE(content_rating, 'Not Rated') as content_rating,
		COUNT(*) as playback_count
	FROM playback_events
	WHERE 1=1`

	query, args := newQueryBuilder(baseQuery).
		addStandardFilters(filter).
		build("GROUP BY content_rating ORDER BY playback_count DESC")

	scanRating := func(rows *sql.Rows) (models.RatingStats, error) {
		var rs models.RatingStats
		err := rows.Scan(&rs.ContentRating, &rs.PlaybackCount)
		return rs, err
	}

	results, err := queryAndScan(ctx, db.conn, query, args, scanRating)
	if err != nil {
		return nil, fmt.Errorf("failed to query rating distribution: %w", err)
	}

	// Calculate percentages
	total := 0
	for _, r := range results {
		total += r.PlaybackCount
	}
	if total > 0 {
		for i := range results {
			results[i].Percentage = float64(results[i].PlaybackCount) / float64(total) * 100
		}
	}

	return results, nil
}

// GetDurationStats retrieves watch duration analytics
// buildDurationWhereClause builds the WHERE clause and arguments for duration queries
func buildDurationWhereClause(filter LocationStatsFilter) (string, []interface{}) {
	whereClause := " AND play_duration IS NOT NULL AND play_duration > 0"
	args := []interface{}{}

	if filter.StartDate != nil {
		whereClause += " AND started_at >= ?"
		args = append(args, *filter.StartDate)
	}
	if filter.EndDate != nil {
		whereClause += " AND started_at <= ?"
		args = append(args, *filter.EndDate)
	}
	if len(filter.Users) > 0 {
		placeholders := make([]string, len(filter.Users))
		for i, user := range filter.Users {
			placeholders[i] = "?"
			args = append(args, user)
		}
		whereClause += fmt.Sprintf(" AND username IN (%s)", join(placeholders, ","))
	}
	if len(filter.MediaTypes) > 0 {
		placeholders := make([]string, len(filter.MediaTypes))
		for i, mediaType := range filter.MediaTypes {
			placeholders[i] = "?"
			args = append(args, mediaType)
		}
		whereClause += fmt.Sprintf(" AND media_type IN (%s)", join(placeholders, ","))
	}

	return whereClause, args
}

// getDurationSummary retrieves overall duration statistics with NULL handling
func (db *DB) getDurationSummary(ctx context.Context, whereClause string, args []interface{}) (models.DurationStats, int, error) {
	query := `
	SELECT
		CAST(AVG(COALESCE(play_duration, 0)) AS INTEGER) as avg_duration,
		CAST(PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY COALESCE(play_duration, 0)) AS INTEGER) as median_duration,
		CAST(SUM(COALESCE(play_duration, 0)) AS INTEGER) as total_duration,
		AVG(percent_complete) as avg_completion,
		SUM(CASE WHEN percent_complete = 100 THEN 1 ELSE 0 END) as fully_watched,
		COUNT(*) as total_playbacks
	FROM playback_events
	WHERE 1=1` + whereClause

	var stats models.DurationStats
	var totalPlaybacks int
	var avgDuration, medianDuration, totalDuration, fullyWatched sql.NullInt64
	var avgCompletion sql.NullFloat64

	err := db.conn.QueryRowContext(ctx, query, args...).Scan(
		&avgDuration,
		&medianDuration,
		&totalDuration,
		&avgCompletion,
		&fullyWatched,
		&totalPlaybacks,
	)
	if err != nil {
		return models.DurationStats{}, 0, fmt.Errorf("failed to query duration stats: %w", err)
	}

	// Handle NULL values by converting to 0
	if avgDuration.Valid {
		stats.AvgDuration = int(avgDuration.Int64)
	}
	if medianDuration.Valid {
		stats.MedianDuration = int(medianDuration.Int64)
	}
	if totalDuration.Valid {
		stats.TotalDuration = int(totalDuration.Int64)
	}
	if avgCompletion.Valid {
		stats.AvgCompletion = avgCompletion.Float64
	}
	if fullyWatched.Valid {
		stats.FullyWatched = int(fullyWatched.Int64)
	}

	// Calculate fully watched percentage
	if totalPlaybacks > 0 {
		stats.FullyWatchedPct = float64(stats.FullyWatched) / float64(totalPlaybacks) * 100
		stats.PartiallyWatched = totalPlaybacks - stats.FullyWatched
	}

	return stats, totalPlaybacks, nil
}

// getDurationByMediaType retrieves duration breakdown by media type
func (db *DB) getDurationByMediaType(ctx context.Context, whereClause string, args []interface{}) ([]models.DurationByMediaType, error) {
	query := `
	SELECT
		media_type,
		CAST(AVG(COALESCE(play_duration, 0)) AS INTEGER) as avg_duration,
		CAST(SUM(COALESCE(play_duration, 0)) AS INTEGER) as total_duration,
		COUNT(*) as playback_count,
		AVG(percent_complete) as avg_completion
	FROM playback_events
	WHERE 1=1` + whereClause + `
	GROUP BY media_type
	ORDER BY total_duration DESC`

	rows, err := db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query duration by type: %w", err)
	}
	defer rows.Close()

	var durationByType []models.DurationByMediaType
	for rows.Next() {
		var dt models.DurationByMediaType
		if err := rows.Scan(&dt.MediaType, &dt.AvgDuration, &dt.TotalDuration, &dt.PlaybackCount, &dt.AvgCompletion); err != nil {
			return nil, fmt.Errorf("failed to scan duration by type: %w", err)
		}
		durationByType = append(durationByType, dt)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating duration by type: %w", err)
	}

	return durationByType, nil
}

func (db *DB) GetDurationStats(ctx context.Context, filter LocationStatsFilter) (models.DurationStats, error) {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	// Build WHERE clause once for both queries
	whereClause, args := buildDurationWhereClause(filter)

	// Get overall duration statistics
	stats, _, err := db.getDurationSummary(ctx, whereClause, args)
	if err != nil {
		return models.DurationStats{}, err
	}

	// Get duration breakdown by media type
	durationByType, err := db.getDurationByMediaType(ctx, whereClause, args)
	if err != nil {
		return stats, err
	}

	stats.DurationByType = durationByType

	return stats, nil
}

// GetYearDistribution retrieves playback statistics by release year
func (db *DB) GetYearDistribution(ctx context.Context, filter LocationStatsFilter, limit int) ([]models.YearStats, error) {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	baseQuery := `
	SELECT
		year,
		COUNT(*) as playback_count
	FROM playback_events
	WHERE year IS NOT NULL AND year > 0`

	query, args := newQueryBuilder(baseQuery).
		addStandardFilters(filter).
		addLimit(limit).
		build("GROUP BY year ORDER BY playback_count DESC LIMIT ?")

	scanYear := func(rows *sql.Rows) (models.YearStats, error) {
		var ys models.YearStats
		err := rows.Scan(&ys.Year, &ys.PlaybackCount)
		return ys, err
	}

	years, err := queryAndScan(ctx, db.conn, query, args, scanYear)
	if err != nil {
		return nil, fmt.Errorf("failed to query year distribution: %w", err)
	}

	return years, nil
}

// GetBingeAnalytics retrieves binge-watching analytics with optional filters
// A binge session is defined as 3+ consecutive episodes of the same show watched within 6 hours between episodes
