// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package database provides data access and analytics functionality for the Cartographus application.
// This file contains comparative analytics and content abandonment analysis.
package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/tomtom215/cartographus/internal/models"
)

// periodBoundaries holds start and end times for current and previous periods
type periodBoundaries struct {
	currentStart  time.Time
	currentEnd    time.Time
	previousStart time.Time
	previousEnd   time.Time
}

// calculatePeriodBoundaries calculates time boundaries based on comparison type
func calculatePeriodBoundaries(now time.Time, comparisonType string, filter LocationStatsFilter) (periodBoundaries, string) {
	var bounds periodBoundaries

	switch comparisonType {
	case "week":
		bounds = weekPeriod(now)
	case "month":
		bounds = monthPeriod(now)
	case "quarter":
		bounds = quarterPeriod(now)
	case "year":
		bounds = yearPeriod(now)
	case "custom":
		bounds = customPeriod(now, filter)
	default:
		bounds = weekPeriod(now)
		comparisonType = "week"
	}

	return bounds, comparisonType
}

// weekPeriod calculates current week vs previous week
func weekPeriod(now time.Time) periodBoundaries {
	currentEnd := now
	currentStart := now.AddDate(0, 0, -7)
	previousEnd := currentStart
	previousStart := previousEnd.AddDate(0, 0, -7)
	return periodBoundaries{currentStart, currentEnd, previousStart, previousEnd}
}

// monthPeriod calculates current month vs previous month
func monthPeriod(now time.Time) periodBoundaries {
	currentEnd := now
	currentStart := now.AddDate(0, -1, 0)
	previousEnd := currentStart
	previousStart := previousEnd.AddDate(0, -1, 0)
	return periodBoundaries{currentStart, currentEnd, previousStart, previousEnd}
}

// quarterPeriod calculates current quarter vs previous quarter
func quarterPeriod(now time.Time) periodBoundaries {
	currentEnd := now
	currentStart := now.AddDate(0, -3, 0)
	previousEnd := currentStart
	previousStart := previousEnd.AddDate(0, -3, 0)
	return periodBoundaries{currentStart, currentEnd, previousStart, previousEnd}
}

// yearPeriod calculates current year vs previous year
func yearPeriod(now time.Time) periodBoundaries {
	currentEnd := now
	currentStart := now.AddDate(-1, 0, 0)
	previousEnd := currentStart
	previousStart := previousEnd.AddDate(-1, 0, 0)
	return periodBoundaries{currentStart, currentEnd, previousStart, previousEnd}
}

// customPeriod uses filter dates or defaults to 30 days
func customPeriod(now time.Time, filter LocationStatsFilter) periodBoundaries {
	if filter.StartDate != nil && filter.EndDate != nil {
		currentStart := *filter.StartDate
		currentEnd := *filter.EndDate
		duration := currentEnd.Sub(currentStart)
		previousEnd := currentStart
		previousStart := previousEnd.Add(-duration)
		return periodBoundaries{currentStart, currentEnd, previousStart, previousEnd}
	}

	// Default to last 30 days vs previous 30 days
	currentEnd := now
	currentStart := now.AddDate(0, 0, -30)
	previousEnd := currentStart
	previousStart := previousEnd.AddDate(0, 0, -30)
	return periodBoundaries{currentStart, currentEnd, previousStart, previousEnd}
}

func (db *DB) GetComparativeAnalytics(ctx context.Context, filter LocationStatsFilter, comparisonType string) (*models.ComparativeAnalytics, error) {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	// Calculate period boundaries
	bounds, comparisonType := calculatePeriodBoundaries(time.Now(), comparisonType, filter)

	// Get metrics for both periods
	currentMetrics, err := db.getPeriodMetrics(ctx, bounds.currentStart, bounds.currentEnd, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get current period metrics: %w", err)
	}

	previousMetrics, err := db.getPeriodMetrics(ctx, bounds.previousStart, bounds.previousEnd, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get previous period metrics: %w", err)
	}

	// Build metrics comparison
	metricsComparison := buildMetricsComparison(currentMetrics, previousMetrics)

	// Get top content and user comparisons
	topContentComparison, err := db.getTopContentComparison(ctx, bounds.currentStart, bounds.currentEnd, bounds.previousStart, bounds.previousEnd, filter, 10)
	if err != nil {
		return nil, fmt.Errorf("failed to get content comparison: %w", err)
	}

	topUserComparison, err := db.getTopUserComparison(ctx, bounds.currentStart, bounds.currentEnd, bounds.previousStart, bounds.previousEnd, filter, 10)
	if err != nil {
		return nil, fmt.Errorf("failed to get user comparison: %w", err)
	}

	// Determine overall trend and generate insights
	overallTrend := determineOverallTrend(currentMetrics, previousMetrics)
	insights := generateKeyInsights(currentMetrics, previousMetrics, metricsComparison)

	return &models.ComparativeAnalytics{
		CurrentPeriod:        *currentMetrics,
		PreviousPeriod:       *previousMetrics,
		ComparisonType:       comparisonType,
		MetricsComparison:    metricsComparison,
		TopContentComparison: topContentComparison,
		TopUserComparison:    topUserComparison,
		OverallTrend:         overallTrend,
		KeyInsights:          insights,
	}, nil
}

// buildMetricsComparison creates comparison metrics for all key metrics
func buildMetricsComparison(current, previous *models.PeriodMetrics) []models.ComparativeMetrics {
	return []models.ComparativeMetrics{
		compareMetric("Playback Count", float64(current.PlaybackCount), float64(previous.PlaybackCount), true),
		compareMetric("Unique Users", float64(current.UniqueUsers), float64(previous.UniqueUsers), true),
		compareMetric("Watch Time (Hours)", current.WatchTimeMinutes/60.0, previous.WatchTimeMinutes/60.0, true),
		compareMetric("Avg Session (Minutes)", current.AvgSessionMins, previous.AvgSessionMins, true),
		compareMetric("Completion Rate (%)", current.AvgCompletion, previous.AvgCompletion, true),
		compareMetric("Unique Content", float64(current.UniqueContent), float64(previous.UniqueContent), true),
		compareMetric("Unique Locations", float64(current.UniqueLocations), float64(previous.UniqueLocations), true),
	}
}

// determineOverallTrend determines if playback activity is growing, declining, or stable
func determineOverallTrend(current, previous *models.PeriodMetrics) string {
	if current.PlaybackCount > previous.PlaybackCount*11/10 {
		return "growing"
	}
	if current.PlaybackCount < previous.PlaybackCount*9/10 {
		return "declining"
	}
	return "stable"
}

// getPeriodMetrics retrieves aggregated metrics for a specific time period
func (db *DB) getPeriodMetrics(ctx context.Context, startDate, endDate time.Time, filter LocationStatsFilter) (*models.PeriodMetrics, error) {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	whereClauses, args := buildFilterConditions(filter, false, 1)
	whereClauses = append(whereClauses, "started_at >= ?", "started_at <= ?")
	args = append(args, startDate, endDate)
	whereClause := join(whereClauses, " AND ")

	query := fmt.Sprintf(`
		SELECT
			COUNT(*) as playback_count,
			COUNT(DISTINCT user_id) as unique_users,
			COALESCE(SUM(play_duration), 0) as watch_time_minutes,
			COALESCE(AVG(play_duration), 0) as avg_session_mins,
			COALESCE(AVG(percent_complete), 0) as avg_completion,
			COUNT(DISTINCT title || COALESCE(parent_title, '') || COALESCE(grandparent_title, '')) as unique_content,
			COUNT(DISTINCT ip_address) as unique_locations
		FROM playback_events
		WHERE %s AND play_duration IS NOT NULL AND play_duration > 0
	`, whereClause)

	var metrics models.PeriodMetrics
	err := db.conn.QueryRowContext(ctx, query, args...).Scan(
		&metrics.PlaybackCount,
		&metrics.UniqueUsers,
		&metrics.WatchTimeMinutes,
		&metrics.AvgSessionMins,
		&metrics.AvgCompletion,
		&metrics.UniqueContent,
		&metrics.UniqueLocations,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query period metrics: %w", err)
	}

	metrics.StartDate = startDate
	metrics.EndDate = endDate

	return &metrics, nil
}

// compareMetric creates a ComparativeMetrics comparison between two values
func compareMetric(name string, current, previous float64, higherIsBetter bool) models.ComparativeMetrics {
	absoluteChange := current - previous
	percentageChange := calculatePercentageChange(current, previous)
	direction := getChangeDirection(absoluteChange)
	isImprovement := determineImprovement(absoluteChange, higherIsBetter)

	return models.ComparativeMetrics{
		Metric:           name,
		CurrentValue:     current,
		PreviousValue:    previous,
		AbsoluteChange:   absoluteChange,
		PercentageChange: percentageChange,
		GrowthDirection:  direction,
		IsImprovement:    isImprovement,
	}
}

// calculatePercentageChange calculates the percentage change between two values
func calculatePercentageChange(current, previous float64) float64 {
	if previous != 0 {
		return ((current - previous) / previous) * 100.0
	}
	if current > 0 {
		return 100.0
	}
	return 0.0
}

// getChangeDirection determines if a value is going up, down, or stable
func getChangeDirection(change float64) string {
	if change > 0.01 {
		return "up"
	}
	if change < -0.01 {
		return "down"
	}
	return "stable"
}

// determineImprovement determines if a change is an improvement
func determineImprovement(change float64, higherIsBetter bool) bool {
	if higherIsBetter {
		return change > 0
	}
	return change < 0
}

// contentItem represents a content title and count for comparison
type contentItem struct {
	Title string
	Count int
}

// getTopContentComparison compares top content between two periods
func (db *DB) getTopContentComparison(ctx context.Context, currentStart, currentEnd, previousStart, previousEnd time.Time, filter LocationStatsFilter, limit int) ([]models.TopContentComparison, error) {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	// Get top content for both periods
	currentContent, err := db.getTopContentForPeriod(ctx, currentStart, currentEnd, filter, limit)
	if err != nil {
		return nil, err
	}

	previousContent, err := db.getTopContentForPeriod(ctx, previousStart, previousEnd, filter, limit*2)
	if err != nil {
		return nil, err
	}

	// Build comparison using generic helper
	return buildContentComparison(currentContent, previousContent), nil
}

// buildContentComparison builds a comparison between current and previous content
func buildContentComparison(current, previous []contentItem) []models.TopContentComparison {
	previousRanks := make(map[string]int, len(previous))
	previousCounts := make(map[string]int, len(previous))

	for i, item := range previous {
		previousRanks[item.Title] = i + 1
		previousCounts[item.Title] = item.Count
	}

	comparison := make([]models.TopContentComparison, 0, len(current))
	for i, item := range current {
		currentRank := i + 1
		previousRank := previousRanks[item.Title]
		previousCount := previousCounts[item.Title]
		existed := previousRank > 0

		rankChange := 0
		if existed {
			rankChange = previousRank - currentRank
		}

		countChange := item.Count - previousCount
		countChangePct := calculatePercentageChange(float64(item.Count), float64(previousCount))
		trending := determineTrending(existed, rankChange)

		comparison = append(comparison, models.TopContentComparison{
			Title:          item.Title,
			CurrentRank:    currentRank,
			PreviousRank:   previousRank,
			RankChange:     rankChange,
			CurrentCount:   item.Count,
			PreviousCount:  previousCount,
			CountChange:    countChange,
			CountChangePct: countChangePct,
			Trending:       trending,
		})
	}

	return comparison
}

// determineTrending determines the trending status of an item
func determineTrending(existed bool, rankChange int) string {
	if !existed {
		return "new"
	}
	if rankChange > 0 {
		return "rising"
	}
	if rankChange < 0 {
		return "falling"
	}
	return "stable"
}

// getTopContentForPeriod retrieves top content for a specific period
func (db *DB) getTopContentForPeriod(ctx context.Context, startDate, endDate time.Time, filter LocationStatsFilter, limit int) ([]contentItem, error) {
	whereClauses, args := buildFilterConditions(filter, false, 1)
	whereClauses = append(whereClauses, "started_at >= ?", "started_at <= ?")
	args = append(args, startDate, endDate)
	whereClause := join(whereClauses, " AND ")

	query := fmt.Sprintf(`
		SELECT
			CASE
				WHEN grandparent_title IS NOT NULL AND grandparent_title != '' THEN grandparent_title
				ELSE title
			END as display_title,
			COUNT(*) as count
		FROM playback_events
		WHERE %s
		GROUP BY display_title
		ORDER BY count DESC
		LIMIT ?
	`, whereClause)

	args = append(args, limit)
	var results []contentItem
	err := db.queryAndScan(ctx, query, args, func(rows *sql.Rows) error {
		var item contentItem
		if err := rows.Scan(&item.Title, &item.Count); err != nil {
			return err
		}
		results = append(results, item)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query top content: %w", err)
	}
	return results, nil
}

// userItem represents a user and watch time for comparison
type userItem struct {
	Username  string
	WatchTime float64
}

// getTopUserComparison compares top users between two periods
func (db *DB) getTopUserComparison(ctx context.Context, currentStart, currentEnd, previousStart, previousEnd time.Time, filter LocationStatsFilter, limit int) ([]models.TopUserComparison, error) {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	// Get top users for both periods
	currentUsers, err := db.getTopUsersForPeriod(ctx, currentStart, currentEnd, filter, limit)
	if err != nil {
		return nil, err
	}

	previousUsers, err := db.getTopUsersForPeriod(ctx, previousStart, previousEnd, filter, limit*2)
	if err != nil {
		return nil, err
	}

	// Build comparison using generic helper
	return buildUserComparison(currentUsers, previousUsers), nil
}

// buildUserComparison builds a comparison between current and previous users
func buildUserComparison(current, previous []userItem) []models.TopUserComparison {
	previousRanks := make(map[string]int, len(previous))
	previousWatchTimes := make(map[string]float64, len(previous))

	for i, user := range previous {
		previousRanks[user.Username] = i + 1
		previousWatchTimes[user.Username] = user.WatchTime
	}

	comparison := make([]models.TopUserComparison, 0, len(current))
	for i, user := range current {
		currentRank := i + 1
		previousRank := previousRanks[user.Username]
		previousWatchTime := previousWatchTimes[user.Username]
		existed := previousRank > 0

		rankChange := 0
		if existed {
			rankChange = previousRank - currentRank
		}

		watchTimeChange := user.WatchTime - previousWatchTime
		watchTimeChangePct := calculatePercentageChange(user.WatchTime, previousWatchTime)
		trending := determineTrending(existed, rankChange)

		comparison = append(comparison, models.TopUserComparison{
			Username:           user.Username,
			CurrentRank:        currentRank,
			PreviousRank:       previousRank,
			RankChange:         rankChange,
			CurrentWatchTime:   user.WatchTime,
			PreviousWatchTime:  previousWatchTime,
			WatchTimeChange:    watchTimeChange,
			WatchTimeChangePct: watchTimeChangePct,
			Trending:           trending,
		})
	}

	return comparison
}

// getTopUsersForPeriod retrieves top users for a specific period
func (db *DB) getTopUsersForPeriod(ctx context.Context, startDate, endDate time.Time, filter LocationStatsFilter, limit int) ([]userItem, error) {
	whereClauses, args := buildFilterConditions(filter, false, 1)
	whereClauses = append(whereClauses, "started_at >= ?", "started_at <= ?")
	args = append(args, startDate, endDate)
	whereClause := join(whereClauses, " AND ")

	query := fmt.Sprintf(`
		SELECT
			username,
			COALESCE(SUM(play_duration), 0) as watch_time
		FROM playback_events
		WHERE %s AND play_duration IS NOT NULL AND play_duration > 0
		GROUP BY username
		ORDER BY watch_time DESC
		LIMIT ?
	`, whereClause)

	args = append(args, limit)
	var results []userItem
	err := db.queryAndScan(ctx, query, args, func(rows *sql.Rows) error {
		var user userItem
		if err := rows.Scan(&user.Username, &user.WatchTime); err != nil {
			return err
		}
		results = append(results, user)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query top users: %w", err)
	}
	return results, nil
}

// buildAbandonmentWhereClause builds the WHERE clause and arguments for abandonment queries
// Uses shared helper from analytics_helpers.go to reduce code duplication
func buildAbandonmentWhereClause(filter LocationStatsFilter) (string, []interface{}) {
	return buildWhereClauseWithArgs(filter, "percent_complete IS NOT NULL")
}

// getAbandonmentSummary retrieves summary statistics for content abandonment
func (db *DB) getAbandonmentSummary(ctx context.Context, whereClause string, args []interface{}) (models.ContentAbandonmentSummary, error) {
	summaryQuery := fmt.Sprintf(`
		SELECT
			COUNT(*) as total_playbacks,
			COALESCE(SUM(CASE WHEN percent_complete >= 90 THEN 1 ELSE 0 END), 0) as completed_playbacks,
			COALESCE(SUM(CASE WHEN percent_complete < 90 THEN 1 ELSE 0 END), 0) as abandoned_playbacks,
			COALESCE(AVG(percent_complete), 0) as avg_completion,
			COALESCE(PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY percent_complete), 0) as median_drop_off_point
		FROM playback_events
		WHERE %s
	`, whereClause)

	summary := models.ContentAbandonmentSummary{}
	err := db.conn.QueryRowContext(ctx, summaryQuery, args...).Scan(
		&summary.TotalPlaybacks,
		&summary.CompletedPlaybacks,
		&summary.AbandonedPlaybacks,
		&summary.AverageCompletion,
		&summary.MedianDropOffPoint,
	)
	if err != nil {
		return summary, fmt.Errorf("failed to get abandonment summary: %w", err)
	}

	if summary.TotalPlaybacks > 0 {
		summary.CompletionRate = (float64(summary.CompletedPlaybacks) / float64(summary.TotalPlaybacks)) * 100.0
	}

	return summary, nil
}

// getTopAbandonedContent retrieves the top 20 most abandoned content items
func (db *DB) getTopAbandonedContent(ctx context.Context, whereClause string, args []interface{}) ([]models.AbandonedContent, error) {
	query := fmt.Sprintf(`
		SELECT
			title,
			media_type,
			COALESCE(grandparent_title, '') as grandparent_title,
			COUNT(*) as total_starts,
			SUM(CASE WHEN percent_complete >= 90 THEN 1 ELSE 0 END) as completions,
			SUM(CASE WHEN percent_complete < 90 THEN 1 ELSE 0 END) as abandonments,
			AVG(percent_complete) as avg_completion,
			PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY percent_complete) as median_drop_off_point,
			COALESCE(genres, '') as genres
		FROM playback_events
		WHERE %s
		GROUP BY title, media_type, grandparent_title, genres
		HAVING COUNT(*) >= 3
			AND SUM(CASE WHEN percent_complete < 90 THEN 1 ELSE 0 END) > 0
		ORDER BY (SUM(CASE WHEN percent_complete < 90 THEN 1 ELSE 0 END) * 1.0 / COUNT(*)) DESC, COUNT(*) DESC
		LIMIT 20
	`, whereClause)

	var results []models.AbandonedContent
	err := db.queryAndScan(ctx, query, args, func(rows *sql.Rows) error {
		var content models.AbandonedContent
		if err := rows.Scan(
			&content.Title,
			&content.MediaType,
			&content.GrandparentTitle,
			&content.TotalStarts,
			&content.Completions,
			&content.Abandonments,
			&content.AverageCompletion,
			&content.MedianDropOffPoint,
			&content.Genres,
		); err != nil {
			return err
		}
		if content.TotalStarts > 0 {
			content.CompletionRate = (float64(content.Completions) / float64(content.TotalStarts)) * 100.0
		}
		results = append(results, content)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get top abandoned content: %w", err)
	}
	return results, nil
}

// getCompletionByMediaType retrieves completion statistics by media type
func (db *DB) getCompletionByMediaType(ctx context.Context, whereClause string, args []interface{}) ([]models.MediaTypeCompletion, error) {
	query := fmt.Sprintf(`
		SELECT
			media_type,
			COUNT(*) as total_playbacks,
			SUM(CASE WHEN percent_complete >= 90 THEN 1 ELSE 0 END) as completed_playbacks,
			AVG(percent_complete) as avg_completion
		FROM playback_events
		WHERE %s
		GROUP BY media_type
		ORDER BY total_playbacks DESC
	`, whereClause)

	var results []models.MediaTypeCompletion
	err := db.queryAndScan(ctx, query, args, func(rows *sql.Rows) error {
		var mt models.MediaTypeCompletion
		if err := rows.Scan(
			&mt.MediaType,
			&mt.TotalPlaybacks,
			&mt.CompletedPlaybacks,
			&mt.AverageCompletion,
		); err != nil {
			return err
		}
		if mt.TotalPlaybacks > 0 {
			mt.CompletionRate = (float64(mt.CompletedPlaybacks) / float64(mt.TotalPlaybacks)) * 100.0
		}
		results = append(results, mt)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get media type completion: %w", err)
	}
	return results, nil
}

// getDropOffDistribution retrieves drop-off distribution by completion percentage buckets
func (db *DB) getDropOffDistribution(ctx context.Context, whereClause string, args []interface{}, totalPlaybacks int) ([]models.DropOffBucket, error) {
	query := fmt.Sprintf(`
		SELECT
			CASE
				WHEN percent_complete < 25 THEN '0-25%%'
				WHEN percent_complete < 50 THEN '25-50%%'
				WHEN percent_complete < 75 THEN '50-75%%'
				WHEN percent_complete < 90 THEN '75-90%%'
				ELSE '90-100%%'
			END as bucket,
			CASE
				WHEN percent_complete < 25 THEN 0
				WHEN percent_complete < 50 THEN 25
				WHEN percent_complete < 75 THEN 50
				WHEN percent_complete < 90 THEN 75
				ELSE 90
			END as min_percent,
			CASE
				WHEN percent_complete < 25 THEN 25
				WHEN percent_complete < 50 THEN 50
				WHEN percent_complete < 75 THEN 75
				WHEN percent_complete < 90 THEN 90
				ELSE 100
			END as max_percent,
			COUNT(*) as playback_count
		FROM playback_events
		WHERE %s
		GROUP BY bucket, min_percent, max_percent
		ORDER BY min_percent
	`, whereClause)

	var results []models.DropOffBucket
	err := db.queryAndScan(ctx, query, args, func(rows *sql.Rows) error {
		var bucket models.DropOffBucket
		if err := rows.Scan(
			&bucket.Bucket,
			&bucket.MinPercent,
			&bucket.MaxPercent,
			&bucket.PlaybackCount,
		); err != nil {
			return err
		}
		if totalPlaybacks > 0 {
			bucket.PercentageOfTotal = (float64(bucket.PlaybackCount) / float64(totalPlaybacks)) * 100.0
		}
		results = append(results, bucket)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get drop-off distribution: %w", err)
	}
	return results, nil
}

// getAbandonmentByGenre retrieves abandonment statistics by genre (optional, only for movies/episodes)
func (db *DB) getAbandonmentByGenre(ctx context.Context, whereClause string, args []interface{}, filter LocationStatsFilter) ([]models.GenreAbandonment, error) {
	// Only query genres for movies and episodes
	if len(filter.MediaTypes) > 0 && !contains(filter.MediaTypes, "movie") && !contains(filter.MediaTypes, "episode") {
		return []models.GenreAbandonment{}, nil
	}

	query := fmt.Sprintf(`
		WITH genre_split AS (
			SELECT
				TRIM(value) as genre,
				percent_complete
			FROM playback_events,
				json_each('["' || REPLACE(COALESCE(genres, ''), ', ', '","') || '"]')
			WHERE %s
				AND genres IS NOT NULL
				AND genres != ''
		)
		SELECT
			genre,
			COUNT(*) as total_playbacks,
			SUM(CASE WHEN percent_complete >= 90 THEN 1 ELSE 0 END) as completed_playbacks,
			AVG(percent_complete) as avg_completion
		FROM genre_split
		WHERE genre != ''
		GROUP BY genre
		HAVING COUNT(*) >= 5
		ORDER BY total_playbacks DESC
		LIMIT 15
	`, whereClause)

	var results []models.GenreAbandonment
	err := db.queryAndScan(ctx, query, args, func(rows *sql.Rows) error {
		var genre models.GenreAbandonment
		if err := rows.Scan(
			&genre.Genre,
			&genre.TotalPlaybacks,
			&genre.CompletedPlaybacks,
			&genre.AverageCompletion,
		); err != nil {
			return err
		}
		if genre.TotalPlaybacks > 0 {
			genre.CompletionRate = (float64(genre.CompletedPlaybacks) / float64(genre.TotalPlaybacks)) * 100.0
		}
		results = append(results, genre)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get genre abandonment: %w", err)
	}
	return results, nil
}

// getFirstEpisodeDropOff retrieves first episode abandonment statistics for TV shows
func (db *DB) getFirstEpisodeDropOff(ctx context.Context, whereClause string, args []interface{}, filter LocationStatsFilter) ([]models.FirstEpisodeDropOff, error) {
	// Only query for episodes
	if len(filter.MediaTypes) > 0 && !contains(filter.MediaTypes, "episode") {
		return []models.FirstEpisodeDropOff{}, nil
	}

	query := fmt.Sprintf(`
		WITH first_episodes AS (
			SELECT
				grandparent_title as show_name,
				title as first_episode_title,
				COUNT(DISTINCT user_id) as pilot_starts,
				SUM(CASE WHEN percent_complete >= 90 THEN 1 ELSE 0 END) as pilot_completions
			FROM playback_events
			WHERE %s
				AND media_type = 'episode'
				AND media_index = 1
				AND parent_media_index = 1
				AND grandparent_title IS NOT NULL
			GROUP BY grandparent_title, title
			HAVING COUNT(DISTINCT user_id) >= 3
		),
		series_continuations AS (
			SELECT
				grandparent_title as show_name,
				COUNT(DISTINCT user_id) as continuations
			FROM playback_events
			WHERE %s
				AND media_type = 'episode'
				AND (media_index > 1 OR parent_media_index > 1)
				AND grandparent_title IS NOT NULL
			GROUP BY grandparent_title
		)
		SELECT
			f.show_name,
			f.first_episode_title,
			f.pilot_starts,
			f.pilot_completions,
			COALESCE(s.continuations, 0) as series_continuations
		FROM first_episodes f
		LEFT JOIN series_continuations s ON f.show_name = s.show_name
		ORDER BY f.pilot_starts DESC
		LIMIT 20
	`, whereClause, whereClause)

	// Double the args for the two WHERE clauses
	doubledArgs := append(args, args...)

	var results []models.FirstEpisodeDropOff
	err := db.queryAndScan(ctx, query, doubledArgs, func(rows *sql.Rows) error {
		var dropOff models.FirstEpisodeDropOff
		if err := rows.Scan(
			&dropOff.ShowName,
			&dropOff.FirstEpisodeTitle,
			&dropOff.PilotStarts,
			&dropOff.PilotCompletions,
			&dropOff.SeriesContinuations,
		); err != nil {
			return err
		}
		if dropOff.PilotStarts > 0 {
			continuationRate := (float64(dropOff.SeriesContinuations) / float64(dropOff.PilotStarts)) * 100.0
			dropOff.DropOffRate = 100.0 - continuationRate
		}
		results = append(results, dropOff)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get first episode drop-off: %w", err)
	}
	return results, nil
}

// GetContentAbandonmentAnalytics analyzes content abandonment patterns (completion rates, drop-off points)
func (db *DB) GetContentAbandonmentAnalytics(ctx context.Context, filter LocationStatsFilter) (*models.ContentAbandonmentAnalytics, error) {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	// Build WHERE clause and arguments using shared helper
	whereClause, args := buildAbandonmentWhereClause(filter)

	// Execute all queries using helper methods with context
	summary, err := db.getAbandonmentSummary(ctx, whereClause, args)
	if err != nil {
		return nil, err
	}

	topAbandoned, err := db.getTopAbandonedContent(ctx, whereClause, args)
	if err != nil {
		return nil, err
	}

	completionByMediaType, err := db.getCompletionByMediaType(ctx, whereClause, args)
	if err != nil {
		return nil, err
	}

	dropOffDistribution, err := db.getDropOffDistribution(ctx, whereClause, args, summary.TotalPlaybacks)
	if err != nil {
		return nil, err
	}

	abandonmentByGenre, err := db.getAbandonmentByGenre(ctx, whereClause, args, filter)
	if err != nil {
		return nil, err
	}

	firstEpisodeAbandonment, err := db.getFirstEpisodeDropOff(ctx, whereClause, args, filter)
	if err != nil {
		return nil, err
	}

	return &models.ContentAbandonmentAnalytics{
		Summary:                 summary,
		TopAbandoned:            topAbandoned,
		CompletionByMediaType:   completionByMediaType,
		DropOffDistribution:     dropOffDistribution,
		AbandonmentByGenre:      abandonmentByGenre,
		FirstEpisodeAbandonment: firstEpisodeAbandonment,
	}, nil
}

// contains checks if a string slice contains a value
func contains(slice []string, value string) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}

// generateKeyInsights generates key insights from comparative metrics
func generateKeyInsights(current, previous *models.PeriodMetrics, metrics []models.ComparativeMetrics) []string {
	insights := []string{}

	insights = append(insights, getPlaybackInsights(current, previous)...)
	insights = append(insights, getUserInsights(current, previous)...)
	insights = append(insights, getWatchTimeInsights(current, previous)...)
	insights = append(insights, getSessionInsights(current, previous)...)
	insights = append(insights, getCompletionInsights(current, previous)...)
	insights = append(insights, getContentInsights(current, previous)...)

	// Add default insight if none generated
	if len(insights) == 0 {
		insights = append(insights, "Activity levels remain relatively stable compared to previous period")
	}

	return insights
}

// getPlaybackInsights generates insights about playback activity changes
func getPlaybackInsights(current, previous *models.PeriodMetrics) []string {
	if current.PlaybackCount == previous.PlaybackCount {
		return nil
	}

	pctChange := ((float64(current.PlaybackCount) - float64(previous.PlaybackCount)) / float64(previous.PlaybackCount)) * 100.0
	if current.PlaybackCount > previous.PlaybackCount {
		return []string{fmt.Sprintf("Playback activity increased by %.1f%% compared to previous period", pctChange)}
	}
	return []string{fmt.Sprintf("Playback activity decreased by %.1f%% compared to previous period", -pctChange)}
}

// getUserInsights generates insights about user engagement changes
func getUserInsights(current, previous *models.PeriodMetrics) []string {
	if current.UniqueUsers <= previous.UniqueUsers {
		return nil
	}

	newUsers := current.UniqueUsers - previous.UniqueUsers
	return []string{fmt.Sprintf("%d new active users joined in this period", newUsers)}
}

// getWatchTimeInsights generates insights about watch time changes
func getWatchTimeInsights(current, previous *models.PeriodMetrics) []string {
	if current.WatchTimeMinutes <= previous.WatchTimeMinutes*1.2 {
		return nil
	}

	return []string{"Significant increase in total watch time - user engagement is growing strongly"}
}

// getSessionInsights generates insights about session length changes
func getSessionInsights(current, previous *models.PeriodMetrics) []string {
	if current.AvgSessionMins <= previous.AvgSessionMins*1.1 {
		return nil
	}

	return []string{"Users are watching for longer sessions on average"}
}

// getCompletionInsights generates insights about completion rate changes
func getCompletionInsights(current, previous *models.PeriodMetrics) []string {
	diff := current.AvgCompletion - previous.AvgCompletion
	if diff > 5 {
		return []string{"Content completion rate improved - users are finishing more content"}
	}
	if diff < -5 {
		return []string{"Content completion rate declined - users may be browsing more"}
	}
	return nil
}

// getContentInsights generates insights about content diversity changes
func getContentInsights(current, previous *models.PeriodMetrics) []string {
	if current.UniqueContent <= int(float64(previous.UniqueContent)*1.2) {
		return nil
	}

	return []string{"Users are exploring more diverse content this period"}
}
