// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package database

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/tomtom215/cartographus/internal/recommend"
)

// GetRecommendationInteractions returns user-item interactions for training recommendation models.
// It aggregates playback events into interactions with confidence scores.
func (db *DB) GetRecommendationInteractions(ctx context.Context, since time.Time) ([]recommend.Interaction, error) {
	query := `
		WITH playback_aggregates AS (
			SELECT
				user_id,
				rating_key AS item_id,
				MAX(percent_complete) AS max_percent,
				SUM(COALESCE(play_duration, 0)) AS total_duration,
				COUNT(*) AS play_count,
				MAX(started_at) AS last_played,
				session_key
			FROM playbacks
			WHERE started_at >= ?
			  AND user_id IS NOT NULL
			  AND rating_key IS NOT NULL
			GROUP BY user_id, rating_key, session_key
		)
		SELECT
			user_id,
			item_id,
			max_percent,
			total_duration,
			play_count,
			last_played,
			session_key
		FROM playback_aggregates
		ORDER BY last_played DESC
	`

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	rows, err := db.conn.QueryContext(ctx, query, since)
	if err != nil {
		return nil, fmt.Errorf("query interactions: %w", err)
	}
	defer rows.Close()

	var interactions []recommend.Interaction
	for rows.Next() {
		var (
			userID        int
			itemID        int
			maxPercent    int
			totalDuration int
			playCount     int
			lastPlayed    time.Time
			sessionKey    string
		)

		if err := rows.Scan(&userID, &itemID, &maxPercent, &totalDuration, &playCount, &lastPlayed, &sessionKey); err != nil {
			return nil, fmt.Errorf("scan interaction: %w", err)
		}

		interactionType := recommend.ClassifyInteraction(maxPercent)
		confidence := recommend.ComputeConfidence(maxPercent, totalDuration)

		// Boost confidence for rewatches
		if playCount > 1 {
			confidence *= 1.0 + 0.1*float64(playCount-1)
		}

		interactions = append(interactions, recommend.Interaction{
			UserID:          userID,
			ItemID:          itemID,
			Type:            interactionType,
			Confidence:      confidence,
			PercentComplete: maxPercent,
			PlayDuration:    totalDuration,
			Timestamp:       lastPlayed,
			SessionID:       sessionKey,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate interactions: %w", err)
	}

	return interactions, nil
}

// GetRecommendationItems returns item metadata for recommendation training.
func (db *DB) GetRecommendationItems(ctx context.Context) ([]recommend.Item, error) {
	query := `
		SELECT DISTINCT ON (rating_key)
			rating_key AS id,
			COALESCE(title, '') AS title,
			COALESCE(media_type, 'unknown') AS media_type,
			COALESCE(genres, '') AS genres,
			COALESCE(directors, '') AS directors,
			COALESCE(actors, '') AS actors,
			COALESCE(year, 0) AS year,
			COALESCE(studio, '') AS studio,
			COALESCE(content_rating, '') AS content_rating,
			COALESCE(rating, 0) AS rating,
			COALESCE(audience_rating, 0) AS audience_rating,
			COALESCE(parent_rating_key, 0) AS parent_id,
			COALESCE(grandparent_rating_key, 0) AS grandparent_id
		FROM playbacks
		WHERE rating_key IS NOT NULL
		ORDER BY rating_key, started_at DESC
	`

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	rows, err := db.conn.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query items: %w", err)
	}
	defer rows.Close()

	var items []recommend.Item
	for rows.Next() {
		var (
			id             int
			title          string
			mediaType      string
			genresStr      string
			directorsStr   string
			actorsStr      string
			year           int
			studio         string
			contentRating  string
			rating         float64
			audienceRating float64
			parentID       int
			grandparentID  int
		)

		if err := rows.Scan(&id, &title, &mediaType, &genresStr, &directorsStr, &actorsStr,
			&year, &studio, &contentRating, &rating, &audienceRating, &parentID, &grandparentID); err != nil {
			return nil, fmt.Errorf("scan item: %w", err)
		}

		items = append(items, recommend.Item{
			ID:             id,
			Title:          title,
			MediaType:      mediaType,
			Genres:         splitAndTrim(genresStr),
			Directors:      splitAndTrim(directorsStr),
			Actors:         splitAndTrim(actorsStr),
			Year:           year,
			Studio:         studio,
			ContentRating:  contentRating,
			Rating:         rating,
			AudienceRating: audienceRating,
			ParentID:       parentID,
			GrandparentID:  grandparentID,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate items: %w", err)
	}

	return items, nil
}

// GetUserWatchHistory returns item IDs that a user has interacted with.
func (db *DB) GetUserWatchHistory(ctx context.Context, userID int) ([]int, error) {
	query := `
		SELECT DISTINCT rating_key
		FROM playbacks
		WHERE user_id = ?
		  AND rating_key IS NOT NULL
	`

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	rows, err := db.conn.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("query user history: %w", err)
	}
	defer rows.Close()

	var history []int
	for rows.Next() {
		var itemID int
		if err := rows.Scan(&itemID); err != nil {
			return nil, fmt.Errorf("scan item id: %w", err)
		}
		history = append(history, itemID)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate history: %w", err)
	}

	return history, nil
}

// GetRecommendationCandidates returns candidate item IDs for recommendations.
// It excludes items the user has already watched.
func (db *DB) GetRecommendationCandidates(ctx context.Context, userID int, limit int) ([]int, error) {
	query := `
		WITH user_watched AS (
			SELECT DISTINCT rating_key
			FROM playbacks
			WHERE user_id = ?
		),
		all_items AS (
			SELECT DISTINCT rating_key
			FROM playbacks
			WHERE rating_key IS NOT NULL
		)
		SELECT rating_key
		FROM all_items
		WHERE rating_key NOT IN (SELECT rating_key FROM user_watched WHERE rating_key IS NOT NULL)
		LIMIT ?
	`

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	rows, err := db.conn.QueryContext(ctx, query, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("query candidates: %w", err)
	}
	defer rows.Close()

	var candidates []int
	for rows.Next() {
		var itemID int
		if err := rows.Scan(&itemID); err != nil {
			return nil, fmt.Errorf("scan candidate: %w", err)
		}
		candidates = append(candidates, itemID)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate candidates: %w", err)
	}

	return candidates, nil
}

// GetContinueWatchingItems returns in-progress items for a user.
func (db *DB) GetContinueWatchingItems(ctx context.Context, userID int, limit int) ([]recommend.ScoredItem, error) {
	query := `
		WITH latest_playback AS (
			SELECT
				rating_key,
				title,
				media_type,
				genres,
				percent_complete,
				started_at,
				parent_title,
				grandparent_title,
				media_index,
				parent_media_index,
				ROW_NUMBER() OVER (
					PARTITION BY COALESCE(grandparent_rating_key, parent_rating_key, rating_key)
					ORDER BY started_at DESC
				) AS rn
			FROM playbacks
			WHERE user_id = ?
			  AND percent_complete < 90
			  AND percent_complete > 5
		)
		SELECT
			rating_key,
			title,
			media_type,
			COALESCE(genres, '') AS genres,
			percent_complete,
			started_at,
			COALESCE(parent_title, '') AS parent_title,
			COALESCE(grandparent_title, '') AS grandparent_title,
			COALESCE(media_index, 0) AS media_index,
			COALESCE(parent_media_index, 0) AS parent_media_index
		FROM latest_playback
		WHERE rn = 1
		ORDER BY started_at DESC
		LIMIT ?
	`

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	rows, err := db.conn.QueryContext(ctx, query, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("query continue watching: %w", err)
	}
	defer rows.Close()

	var items []recommend.ScoredItem
	now := time.Now()

	for rows.Next() {
		var (
			id               int
			title            string
			mediaType        string
			genresStr        string
			percentComplete  int
			startedAt        time.Time
			parentTitle      string
			grandparentTitle string
			mediaIndex       int
			parentMediaIndex int
		)

		if err := rows.Scan(&id, &title, &mediaType, &genresStr, &percentComplete, &startedAt,
			&parentTitle, &grandparentTitle, &mediaIndex, &parentMediaIndex); err != nil {
			return nil, fmt.Errorf("scan continue watching: %w", err)
		}

		// Score based on recency and progress
		// More recent = higher score, higher progress = higher score
		daysSince := now.Sub(startedAt).Hours() / 24
		recencyScore := 1.0 / (1.0 + daysSince/7.0) // Half-life of 7 days
		progressScore := float64(percentComplete) / 100.0
		score := 0.6*recencyScore + 0.4*progressScore

		// Build display title
		displayTitle := title
		if grandparentTitle != "" {
			displayTitle = fmt.Sprintf("%s - S%02dE%02d", grandparentTitle, parentMediaIndex, mediaIndex)
		} else if parentTitle != "" {
			displayTitle = fmt.Sprintf("%s - %s", parentTitle, title)
		}

		items = append(items, recommend.ScoredItem{
			Item: recommend.Item{
				ID:        id,
				Title:     displayTitle,
				MediaType: mediaType,
				Genres:    splitAndTrim(genresStr),
			},
			Score:  score,
			Reason: fmt.Sprintf("Continue watching (%d%% complete)", percentComplete),
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate continue watching: %w", err)
	}

	return items, nil
}

// splitAndTrim splits a comma-separated string and trims whitespace.
func splitAndTrim(s string) []string {
	if s == "" {
		return nil
	}

	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// RecommendationDataProvider implements recommend.DataProvider using the database.
type RecommendationDataProvider struct {
	db *DB
}

// GetMediaItemByID returns a single media item by its ID (rating key).
func (db *DB) GetMediaItemByID(ctx context.Context, itemID int) (*recommend.Item, error) {
	query := `
		SELECT DISTINCT ON (rating_key)
			rating_key AS id,
			COALESCE(title, '') AS title,
			COALESCE(media_type, 'unknown') AS media_type,
			COALESCE(genres, '') AS genres,
			COALESCE(year, 0) AS year
		FROM playbacks
		WHERE rating_key = ?
		ORDER BY rating_key, started_at DESC
		LIMIT 1
	`

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var (
		id        int
		title     string
		mediaType string
		genresStr string
		year      int
	)

	err := db.conn.QueryRowContext(ctx, query, itemID).Scan(&id, &title, &mediaType, &genresStr, &year)
	if err != nil {
		return nil, fmt.Errorf("query item by ID: %w", err)
	}

	// Parse genres
	var genres []string
	if genresStr != "" {
		genres = strings.Split(genresStr, ",")
		for i := range genres {
			genres[i] = strings.TrimSpace(genres[i])
		}
	}

	return &recommend.Item{
		ID:        id,
		Title:     title,
		MediaType: mediaType,
		Genres:    genres,
		Year:      year,
	}, nil
}

// NewRecommendationDataProvider creates a new data provider.
func NewRecommendationDataProvider(db *DB) *RecommendationDataProvider {
	return &RecommendationDataProvider{db: db}
}

// GetInteractions implements recommend.DataProvider.
func (p *RecommendationDataProvider) GetInteractions(ctx context.Context, since time.Time) ([]recommend.Interaction, error) {
	return p.db.GetRecommendationInteractions(ctx, since)
}

// GetItems implements recommend.DataProvider.
func (p *RecommendationDataProvider) GetItems(ctx context.Context) ([]recommend.Item, error) {
	return p.db.GetRecommendationItems(ctx)
}

// GetUserHistory implements recommend.DataProvider.
func (p *RecommendationDataProvider) GetUserHistory(ctx context.Context, userID int) ([]int, error) {
	return p.db.GetUserWatchHistory(ctx, userID)
}

// GetCandidates implements recommend.DataProvider.
func (p *RecommendationDataProvider) GetCandidates(ctx context.Context, userID int, limit int) ([]int, error) {
	return p.db.GetRecommendationCandidates(ctx, userID, limit)
}

// Ensure interface compliance.
var _ recommend.DataProvider = (*RecommendationDataProvider)(nil)
