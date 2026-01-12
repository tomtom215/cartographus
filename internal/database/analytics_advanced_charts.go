// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package database provides data access and analytics functionality for the Cartographus application.
// This file contains analytics methods for advanced chart visualizations:
//   - Content Flow (Sankey): Show -> Season -> Episode viewing journeys
//   - User Content Overlap (Chord): User-user content similarity matrix
//   - User Profile (Radar): Multi-dimensional user engagement scores
//   - Library Utilization (Treemap): Hierarchical library usage
//   - Calendar Heatmap: Daily activity patterns
//   - Bump Chart: Content ranking changes over time
package database

import (
	"context"
	"fmt"

	"github.com/tomtom215/cartographus/internal/models"
)

// ============================================================================
// Content Flow Analytics (Sankey Diagram)
// ============================================================================

// GetContentFlowAnalytics retrieves content flow data for Sankey diagram visualization.
// This shows viewing journeys from shows to seasons to episodes with drop-off analysis.
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - filter: LocationStatsFilter with date range and filters
//
// Returns: ContentFlowAnalytics with nodes, links, and journey data
func (db *DB) GetContentFlowAnalytics(ctx context.Context, filter LocationStatsFilter) (*models.ContentFlowAnalytics, error) {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	// Query to get show -> season -> episode flow
	query := `
		WITH episode_views AS (
			SELECT
				grandparent_title AS show_title,
				parent_index AS season_number,
				media_index AS episode_number,
				title AS episode_title,
				COUNT(*) AS watch_count,
				AVG(COALESCE(CAST(progress_percent AS DOUBLE), 0)) AS avg_completion
			FROM playback_events
			WHERE media_type = 'episode'
			AND grandparent_title IS NOT NULL
			AND grandparent_title != ''
			AND started_at >= ?
			AND started_at <= ?
			GROUP BY grandparent_title, parent_index, media_index, title
		),
		show_totals AS (
			SELECT
				show_title,
				SUM(watch_count) AS total_watches
			FROM episode_views
			GROUP BY show_title
			ORDER BY total_watches DESC
			LIMIT 20
		),
		filtered_views AS (
			SELECT ev.*
			FROM episode_views ev
			INNER JOIN show_totals st ON ev.show_title = st.show_title
		)
		SELECT
			show_title,
			season_number,
			episode_number,
			episode_title,
			watch_count,
			avg_completion
		FROM filtered_views
		ORDER BY show_title, season_number, episode_number
	`

	rows, err := db.conn.QueryContext(ctx, query, filter.StartDate, filter.EndDate)
	if err != nil {
		return nil, fmt.Errorf("query content flow: %w", err)
	}
	defer rows.Close()

	// Build nodes and links from query results
	nodes := make(map[string]*models.SankeyNode)
	links := make(map[string]*models.SankeyLink)
	journeys := []models.ContentFlowJourney{}

	var totalFlows int64
	showSet := make(map[string]bool)

	for rows.Next() {
		var journey models.ContentFlowJourney
		if err := rows.Scan(
			&journey.ShowTitle,
			&journey.SeasonNumber,
			&journey.EpisodeNumber,
			&journey.EpisodeTitle,
			&journey.WatchCount,
			&journey.AvgCompletion,
		); err != nil {
			return nil, fmt.Errorf("scan content flow row: %w", err)
		}

		journeys = append(journeys, journey)
		totalFlows += journey.WatchCount
		showSet[journey.ShowTitle] = true

		// Create show node
		showID := fmt.Sprintf("show_%s", journey.ShowTitle)
		if _, exists := nodes[showID]; !exists {
			nodes[showID] = &models.SankeyNode{
				ID:    showID,
				Name:  journey.ShowTitle,
				Depth: 0,
				Value: 0,
			}
		}
		nodes[showID].Value += journey.WatchCount

		// Create season node
		seasonID := fmt.Sprintf("season_%s_S%d", journey.ShowTitle, journey.SeasonNumber)
		seasonName := fmt.Sprintf("S%d", journey.SeasonNumber)
		if _, exists := nodes[seasonID]; !exists {
			nodes[seasonID] = &models.SankeyNode{
				ID:    seasonID,
				Name:  seasonName,
				Depth: 1,
				Value: 0,
			}
		}
		nodes[seasonID].Value += journey.WatchCount

		// Create episode node
		episodeID := fmt.Sprintf("episode_%s_S%dE%d", journey.ShowTitle, journey.SeasonNumber, journey.EpisodeNumber)
		episodeName := fmt.Sprintf("E%d: %s", journey.EpisodeNumber, journey.EpisodeTitle)
		if len(episodeName) > 30 {
			episodeName = episodeName[:27] + "..."
		}
		if _, exists := nodes[episodeID]; !exists {
			nodes[episodeID] = &models.SankeyNode{
				ID:    episodeID,
				Name:  episodeName,
				Depth: 2,
				Value: 0,
			}
		}
		nodes[episodeID].Value += journey.WatchCount

		// Create show -> season link
		showSeasonLink := fmt.Sprintf("%s->%s", showID, seasonID)
		if _, exists := links[showSeasonLink]; !exists {
			links[showSeasonLink] = &models.SankeyLink{
				Source: showID,
				Target: seasonID,
				Value:  0,
			}
		}
		links[showSeasonLink].Value += journey.WatchCount

		// Create season -> episode link
		seasonEpisodeLink := fmt.Sprintf("%s->%s", seasonID, episodeID)
		if _, exists := links[seasonEpisodeLink]; !exists {
			links[seasonEpisodeLink] = &models.SankeyLink{
				Source: seasonID,
				Target: episodeID,
				Value:  0,
			}
		}
		links[seasonEpisodeLink].Value += journey.WatchCount
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate content flow rows: %w", err)
	}

	// Convert maps to slices
	nodeSlice := make([]models.SankeyNode, 0, len(nodes))
	for _, node := range nodes {
		nodeSlice = append(nodeSlice, *node)
	}

	linkSlice := make([]models.SankeyLink, 0, len(links))
	for _, link := range links {
		linkSlice = append(linkSlice, *link)
	}

	// Calculate drop-off rate (simplified: compare first episode watches to last)
	var dropOffRate float64
	if len(journeys) > 0 && totalFlows > 0 {
		// This is a simplified calculation
		dropOffRate = 0.0 // Would need more complex logic for accurate drop-off
	}

	return &models.ContentFlowAnalytics{
		Nodes:       nodeSlice,
		Links:       linkSlice,
		Journeys:    journeys,
		TotalShows:  len(showSet),
		TotalFlows:  totalFlows,
		DropOffRate: dropOffRate,
	}, nil
}

// ============================================================================
// User Content Overlap Analytics (Chord Diagram)
// ============================================================================

// GetUserContentOverlapAnalytics retrieves user-user content overlap for Chord diagram.
// This calculates Jaccard similarity between users based on shared content consumption.
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - filter: LocationStatsFilter with date range and filters
//
// Returns: UserContentOverlapAnalytics with matrix and top pairs
func (db *DB) GetUserContentOverlapAnalytics(ctx context.Context, filter LocationStatsFilter) (*models.UserContentOverlapAnalytics, error) {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	// Get active users and their watched content
	query := `
		WITH user_content AS (
			SELECT DISTINCT
				user_id,
				user_name,
				COALESCE(rating_key, title) AS content_key
			FROM playback_events
			WHERE started_at >= ?
			AND started_at <= ?
			AND user_id IS NOT NULL
			AND user_id != ''
		),
		user_content_counts AS (
			SELECT user_id, user_name, COUNT(DISTINCT content_key) AS content_count
			FROM user_content
			GROUP BY user_id, user_name
			HAVING content_count >= 5  -- Only users with meaningful activity
		),
		top_users AS (
			SELECT user_id, user_name
			FROM user_content_counts
			ORDER BY content_count DESC
			LIMIT 20
		),
		user_pairs AS (
			SELECT
				u1.user_id AS user1_id,
				u1.user_name AS user1_name,
				u2.user_id AS user2_id,
				u2.user_name AS user2_name
			FROM top_users u1
			CROSS JOIN top_users u2
			WHERE u1.user_id < u2.user_id
		),
		overlap_calc AS (
			SELECT
				up.user1_id,
				up.user1_name,
				up.user2_id,
				up.user2_name,
				COUNT(DISTINCT CASE WHEN uc1.content_key IS NOT NULL AND uc2.content_key IS NOT NULL THEN uc1.content_key END) AS shared_items,
				COUNT(DISTINCT uc1.content_key) AS user1_items,
				COUNT(DISTINCT uc2.content_key) AS user2_items
			FROM user_pairs up
			LEFT JOIN user_content uc1 ON up.user1_id = uc1.user_id
			LEFT JOIN user_content uc2 ON up.user2_id = uc2.user_id AND uc1.content_key = uc2.content_key
			GROUP BY up.user1_id, up.user1_name, up.user2_id, up.user2_name
		)
		SELECT
			user1_id,
			user1_name,
			user2_id,
			user2_name,
			shared_items,
			CASE
				WHEN (user1_items + user2_items - shared_items) > 0
				THEN CAST(shared_items AS DOUBLE) / (user1_items + user2_items - shared_items)
				ELSE 0
			END AS jaccard_similarity
		FROM overlap_calc
		WHERE shared_items > 0
		ORDER BY jaccard_similarity DESC
		LIMIT 100
	`

	rows, err := db.conn.QueryContext(ctx, query, filter.StartDate, filter.EndDate)
	if err != nil {
		return nil, fmt.Errorf("query user overlap: %w", err)
	}
	defer rows.Close()

	userSet := make(map[string]bool)
	pairs := []models.UserOverlapPair{}
	var totalOverlap float64
	var pairCount int

	for rows.Next() {
		var pair models.UserOverlapPair
		if err := rows.Scan(
			&pair.User1ID,
			&pair.User1Name,
			&pair.User2ID,
			&pair.User2Name,
			&pair.SharedItems,
			&pair.OverlapPercent,
		); err != nil {
			return nil, fmt.Errorf("scan user overlap row: %w", err)
		}

		userSet[pair.User1Name] = true
		userSet[pair.User2Name] = true
		pairs = append(pairs, pair)
		totalOverlap += pair.OverlapPercent
		pairCount++
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate user overlap rows: %w", err)
	}

	// Build users list
	users := make([]string, 0, len(userSet))
	for user := range userSet {
		users = append(users, user)
	}

	// Build matrix (simplified - would need full user list for complete matrix)
	n := len(users)
	matrix := make([][]float64, n)
	for i := range matrix {
		matrix[i] = make([]float64, n)
		matrix[i][i] = 1.0 // Self-similarity is 1
	}

	// Fill matrix from pairs
	userIndex := make(map[string]int)
	for i, user := range users {
		userIndex[user] = i
	}

	for _, pair := range pairs {
		i, ok1 := userIndex[pair.User1Name]
		j, ok2 := userIndex[pair.User2Name]
		if ok1 && ok2 {
			matrix[i][j] = pair.OverlapPercent
			matrix[j][i] = pair.OverlapPercent
		}
	}

	avgOverlap := 0.0
	if pairCount > 0 {
		avgOverlap = totalOverlap / float64(pairCount)
	}

	return &models.UserContentOverlapAnalytics{
		Matrix: models.UserOverlapMatrix{
			Users:  users,
			Matrix: matrix,
		},
		TopPairs:         pairs[:min(10, len(pairs))],
		AvgOverlap:       avgOverlap,
		TotalUsers:       len(users),
		TotalConnections: pairCount,
		ClusterCount:     1, // Simplified - would need clustering algorithm
	}, nil
}

// ============================================================================
// User Profile Analytics (Radar Chart)
// ============================================================================

// GetUserProfileAnalytics retrieves multi-dimensional user engagement scores for Radar chart.
// Dimensions: Watch Time, Completion, Diversity, Quality, Discovery, Social
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - filter: LocationStatsFilter with date range and filters
//
// Returns: UserProfileAnalytics with user scores on each dimension
func (db *DB) GetUserProfileAnalytics(ctx context.Context, filter LocationStatsFilter) (*models.UserProfileAnalytics, error) {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	query := `
		WITH user_metrics AS (
			SELECT
				user_id,
				user_name,
				SUM(COALESCE(duration, 0)) AS total_watch_time,
				AVG(COALESCE(CAST(progress_percent AS DOUBLE), 0)) AS avg_completion,
				COUNT(DISTINCT COALESCE(NULLIF(genre, ''), 'Unknown')) AS genre_count,
				AVG(CASE
					WHEN quality_profile LIKE '%4K%' OR quality_profile LIKE '%2160%' THEN 100
					WHEN quality_profile LIKE '%1080%' THEN 75
					WHEN quality_profile LIKE '%720%' THEN 50
					ELSE 25
				END) AS quality_score,
				COUNT(DISTINCT CASE
					WHEN started_at >= CURRENT_DATE - INTERVAL 30 DAY
					AND added_at IS NOT NULL
					AND started_at <= added_at + INTERVAL 7 DAY
					THEN rating_key
				END) AS new_discoveries,
				COUNT(DISTINCT CASE
					WHEN session_id IN (
						SELECT session_id FROM playback_events
						GROUP BY session_id HAVING COUNT(DISTINCT user_id) > 1
					)
					THEN session_id
				END) AS social_sessions,
				COUNT(*) AS total_plays
			FROM playback_events
			WHERE started_at >= ?
			AND started_at <= ?
			AND user_id IS NOT NULL
			AND user_id != ''
			GROUP BY user_id, user_name
			HAVING total_plays >= 5
		),
		max_values AS (
			SELECT
				MAX(total_watch_time) AS max_watch_time,
				MAX(genre_count) AS max_genres,
				MAX(new_discoveries) AS max_discoveries,
				MAX(social_sessions) AS max_social
			FROM user_metrics
		)
		SELECT
			um.user_id,
			um.user_name,
			-- Normalize each dimension to 0-100
			LEAST(100, CAST(um.total_watch_time AS DOUBLE) / NULLIF(mv.max_watch_time, 0) * 100) AS watch_time_score,
			um.avg_completion AS completion_score,
			LEAST(100, CAST(um.genre_count AS DOUBLE) / NULLIF(mv.max_genres, 0) * 100) AS diversity_score,
			um.quality_score,
			LEAST(100, CAST(um.new_discoveries AS DOUBLE) / NULLIF(GREATEST(mv.max_discoveries, 1), 0) * 100) AS discovery_score,
			LEAST(100, CAST(um.social_sessions AS DOUBLE) / NULLIF(GREATEST(mv.max_social, 1), 0) * 100) AS social_score
		FROM user_metrics um
		CROSS JOIN max_values mv
		ORDER BY (um.total_watch_time + um.avg_completion + um.genre_count) DESC
		LIMIT 50
	`

	rows, err := db.conn.QueryContext(ctx, query, filter.StartDate, filter.EndDate)
	if err != nil {
		return nil, fmt.Errorf("query user profiles: %w", err)
	}
	defer rows.Close()

	profiles := []models.UserProfileScore{}
	avgScores := make([]float64, 6)
	var profileCount int

	for rows.Next() {
		var userID, userName string
		var scores [6]float64

		if err := rows.Scan(
			&userID,
			&userName,
			&scores[0], // Watch Time
			&scores[1], // Completion
			&scores[2], // Diversity
			&scores[3], // Quality
			&scores[4], // Discovery
			&scores[5], // Social
		); err != nil {
			return nil, fmt.Errorf("scan user profile row: %w", err)
		}

		profile := models.UserProfileScore{
			UserID:   userID,
			Username: userName,
			Scores:   scores[:],
			Rank:     profileCount + 1,
		}
		profiles = append(profiles, profile)

		// Accumulate for average
		for i, score := range scores {
			avgScores[i] += score
		}
		profileCount++
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate user profile rows: %w", err)
	}

	// Calculate averages
	if profileCount > 0 {
		for i := range avgScores {
			avgScores[i] /= float64(profileCount)
		}
	}

	// Get top performers (first 5)
	topPerformers := profiles
	if len(topPerformers) > 5 {
		topPerformers = topPerformers[:5]
	}

	return &models.UserProfileAnalytics{
		Axes:           models.UserProfileAxes,
		Profiles:       profiles,
		AverageProfile: avgScores,
		TopPerformers:  topPerformers,
		TotalUsers:     profileCount,
	}, nil
}

// ============================================================================
// Library Utilization Analytics (Treemap)
// ============================================================================

// GetLibraryUtilizationAnalytics retrieves hierarchical library usage for Treemap.
// Hierarchy: Library -> Section Type -> Genre -> Content
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - filter: LocationStatsFilter with date range and filters
//
// Returns: LibraryUtilizationAnalytics with hierarchical tree data
//
//nolint:gocyclo // Complexity from building hierarchical tree structure is acceptable
func (db *DB) GetLibraryUtilizationAnalytics(ctx context.Context, filter LocationStatsFilter) (*models.LibraryUtilizationAnalytics, error) {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	// Query for library section statistics
	query := `
		WITH library_stats AS (
			SELECT
				COALESCE(NULLIF(library_name, ''), 'Unknown Library') AS library_name,
				COALESCE(NULLIF(media_type, ''), 'unknown') AS section_type,
				COALESCE(NULLIF(genre, ''), 'Unknown') AS genre,
				COALESCE(title, 'Unknown') AS content_title,
				COUNT(*) AS play_count,
				SUM(COALESCE(duration, 0)) AS total_watch_time,
				COUNT(DISTINCT user_id) AS unique_viewers,
				AVG(COALESCE(CAST(progress_percent AS DOUBLE), 0)) AS avg_completion
			FROM playback_events
			WHERE started_at >= ?
			AND started_at <= ?
			GROUP BY library_name, section_type, genre, content_title
		)
		SELECT
			library_name,
			section_type,
			genre,
			content_title,
			play_count,
			total_watch_time,
			unique_viewers,
			avg_completion
		FROM library_stats
		ORDER BY library_name, section_type, genre, play_count DESC
	`

	rows, err := db.conn.QueryContext(ctx, query, filter.StartDate, filter.EndDate)
	if err != nil {
		return nil, fmt.Errorf("query library utilization: %w", err)
	}
	defer rows.Close()

	// Build hierarchical tree
	libraryNodes := make(map[string]*models.TreemapNode)
	sectionNodes := make(map[string]*models.TreemapNode)
	genreNodes := make(map[string]*models.TreemapNode)

	var totalWatchTime int64
	var totalContent int

	for rows.Next() {
		var libraryName, sectionType, genre, contentTitle string
		var playCount, watchTime int64
		var uniqueViewers int
		var avgCompletion float64

		if err := rows.Scan(
			&libraryName,
			&sectionType,
			&genre,
			&contentTitle,
			&playCount,
			&watchTime,
			&uniqueViewers,
			&avgCompletion,
		); err != nil {
			return nil, fmt.Errorf("scan library utilization row: %w", err)
		}

		totalWatchTime += watchTime
		totalContent++

		// Library node
		libraryKey := libraryName
		if _, exists := libraryNodes[libraryKey]; !exists {
			libraryNodes[libraryKey] = &models.TreemapNode{
				ID:       libraryKey,
				Name:     libraryName,
				ItemType: "library",
				Children: []models.TreemapNode{},
			}
		}
		libraryNodes[libraryKey].Value += watchTime

		// Section node
		sectionKey := fmt.Sprintf("%s/%s", libraryName, sectionType)
		if _, exists := sectionNodes[sectionKey]; !exists {
			sectionNodes[sectionKey] = &models.TreemapNode{
				ID:       sectionKey,
				Name:     sectionType,
				ItemType: "section",
				Children: []models.TreemapNode{},
			}
		}
		sectionNodes[sectionKey].Value += watchTime

		// Genre node
		genreKey := fmt.Sprintf("%s/%s/%s", libraryName, sectionType, genre)
		if _, exists := genreNodes[genreKey]; !exists {
			genreNodes[genreKey] = &models.TreemapNode{
				ID:       genreKey,
				Name:     genre,
				ItemType: "genre",
				Children: []models.TreemapNode{},
				Metrics: models.TreemapMetrics{
					PlayCount:     playCount,
					UniqueViewers: uniqueViewers,
					AvgCompletion: avgCompletion,
				},
			}
		}
		genreNodes[genreKey].Value += watchTime
		genreNodes[genreKey].Metrics.PlayCount += playCount
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate library utilization rows: %w", err)
	}

	// Build tree structure (simplified - just show libraries with sections)
	rootChildren := []models.TreemapNode{}
	for _, libraryNode := range libraryNodes {
		// Find sections for this library
		for sectionKey, sectionNode := range sectionNodes {
			if len(sectionKey) > len(libraryNode.Name) && sectionKey[:len(libraryNode.Name)] == libraryNode.Name {
				// Find genres for this section
				for genreKey, genreNode := range genreNodes {
					if len(genreKey) > len(sectionKey) && genreKey[:len(sectionKey)] == sectionKey {
						sectionNode.Children = append(sectionNode.Children, *genreNode)
					}
				}
				libraryNode.Children = append(libraryNode.Children, *sectionNode)
			}
		}
		rootChildren = append(rootChildren, *libraryNode)
	}

	root := models.TreemapNode{
		ID:       "root",
		Name:     "All Libraries",
		ItemType: "root",
		Value:    totalWatchTime,
		Children: rootChildren,
	}

	utilizationRate := 0.0
	if totalContent > 0 {
		// Calculate utilization based on content with plays
		utilizationRate = float64(len(genreNodes)) / float64(totalContent) * 100
	}

	return &models.LibraryUtilizationAnalytics{
		Root:            root,
		TotalWatchTime:  totalWatchTime,
		TotalContent:    totalContent,
		UtilizedContent: len(genreNodes),
		UtilizationRate: utilizationRate,
		TopUnwatched:    []string{}, // Would need separate query
		MostPopularPath: "",         // Would need to calculate
	}, nil
}

// ============================================================================
// Calendar Heatmap Analytics
// ============================================================================

// GetCalendarHeatmapAnalytics retrieves daily activity for calendar heatmap.
// Returns 365 days of activity data with normalized intensity values.
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - filter: LocationStatsFilter with date range
//
// Returns: CalendarHeatmapAnalytics with daily activity data
func (db *DB) GetCalendarHeatmapAnalytics(ctx context.Context, filter LocationStatsFilter) (*models.CalendarHeatmapAnalytics, error) {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	query := `
		WITH daily_stats AS (
			SELECT
				CAST(started_at AS DATE) AS day_date,
				SUM(COALESCE(duration, 0)) AS watch_time,
				COUNT(*) AS play_count
			FROM playback_events
			WHERE started_at >= ?
			AND started_at <= ?
			GROUP BY day_date
		)
		SELECT
			CAST(day_date AS VARCHAR) AS date_str,
			watch_time,
			play_count
		FROM daily_stats
		ORDER BY day_date
	`

	rows, err := db.conn.QueryContext(ctx, query, filter.StartDate, filter.EndDate)
	if err != nil {
		return nil, fmt.Errorf("query calendar heatmap: %w", err)
	}
	defer rows.Close()

	days := []models.CalendarDayActivity{}
	var totalWatchTime int64
	var totalPlayCount int
	var maxDaily int64
	var activeDays int

	for rows.Next() {
		var day models.CalendarDayActivity
		if err := rows.Scan(&day.Date, &day.WatchTime, &day.PlayCount); err != nil {
			return nil, fmt.Errorf("scan calendar row: %w", err)
		}

		days = append(days, day)
		totalWatchTime += day.WatchTime
		totalPlayCount += day.PlayCount

		if day.WatchTime > maxDaily {
			maxDaily = day.WatchTime
		}
		if day.WatchTime > 0 {
			activeDays++
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate calendar rows: %w", err)
	}

	// Calculate intensity (0-1 normalized) for each day
	for i := range days {
		if maxDaily > 0 {
			days[i].Intensity = float64(days[i].WatchTime) / float64(maxDaily)
		}
	}

	avgDaily := 0.0
	if len(days) > 0 {
		avgDaily = float64(totalWatchTime) / float64(len(days))
	}

	// Calculate streaks (simplified)
	longestStreak := 0
	currentStreak := 0
	for i := len(days) - 1; i >= 0; i-- {
		if days[i].WatchTime > 0 {
			currentStreak++
			if currentStreak > longestStreak {
				longestStreak = currentStreak
			}
		} else if i == len(days)-1 {
			currentStreak = 0 // Reset if most recent day has no activity
		}
	}

	return &models.CalendarHeatmapAnalytics{
		Days:           days,
		TotalWatchTime: totalWatchTime,
		TotalPlayCount: totalPlayCount,
		MaxDaily:       maxDaily,
		AvgDaily:       avgDaily,
		ActiveDays:     activeDays,
		LongestStreak:  longestStreak,
		CurrentStreak:  currentStreak,
	}, nil
}

// ============================================================================
// Bump Chart Analytics (Ranking Changes)
// ============================================================================

// GetBumpChartAnalytics retrieves content ranking changes over time for Bump chart.
// Shows how top 10 content ranking evolves week-by-week.
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - filter: LocationStatsFilter with date range
//
// Returns: BumpChartAnalytics with ranking data per period
func (db *DB) GetBumpChartAnalytics(ctx context.Context, filter LocationStatsFilter) (*models.BumpChartAnalytics, error) {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	query := `
		WITH weekly_plays AS (
			SELECT
				DATE_TRUNC('week', started_at) AS week_start,
				COALESCE(rating_key, title) AS content_id,
				COALESCE(title, 'Unknown') AS content_title,
				COUNT(*) AS play_count
			FROM playback_events
			WHERE started_at >= ?
			AND started_at <= ?
			GROUP BY week_start, content_id, content_title
		),
		weekly_rankings AS (
			SELECT
				week_start,
				content_id,
				content_title,
				play_count,
				ROW_NUMBER() OVER (PARTITION BY week_start ORDER BY play_count DESC) AS rank
			FROM weekly_plays
		)
		SELECT
			CAST(week_start AS VARCHAR) AS period,
			content_id,
			content_title,
			rank,
			play_count
		FROM weekly_rankings
		WHERE rank <= 10
		ORDER BY week_start, rank
	`

	rows, err := db.conn.QueryContext(ctx, query, filter.StartDate, filter.EndDate)
	if err != nil {
		return nil, fmt.Errorf("query bump chart: %w", err)
	}
	defer rows.Close()

	periodMap := make(map[string][]models.RankingEntry)
	periods := []string{}
	seenPeriods := make(map[string]bool)

	for rows.Next() {
		var entry models.RankingEntry
		if err := rows.Scan(
			&entry.Period,
			&entry.ContentID,
			&entry.ContentTitle,
			&entry.Rank,
			&entry.PlayCount,
		); err != nil {
			return nil, fmt.Errorf("scan bump chart row: %w", err)
		}

		if !seenPeriods[entry.Period] {
			seenPeriods[entry.Period] = true
			periods = append(periods, entry.Period)
		}
		periodMap[entry.Period] = append(periodMap[entry.Period], entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate bump chart rows: %w", err)
	}

	// Convert to array of arrays
	rankings := make([][]models.RankingEntry, len(periods))
	for i, period := range periods {
		rankings[i] = periodMap[period]
	}

	return &models.BumpChartAnalytics{
		Periods:      periods,
		Rankings:     rankings,
		TopMovers:    []models.ContentMover{}, // Would need comparison logic
		NewEntries:   []models.RankingEntry{}, // Would need comparison logic
		Exits:        []models.RankingEntry{}, // Would need comparison logic
		TotalPeriods: len(periods),
	}, nil
}
