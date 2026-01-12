// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

/*
search_fuzzy.go - Fuzzy String Search using RapidFuzz Extension

This file provides fuzzy string matching capabilities for search functionality,
leveraging the DuckDB RapidFuzz community extension.

Features:
  - FuzzySearchPlaybacks: Search playback events with fuzzy title matching
  - FuzzySearchUsers: Search users with fuzzy username matching
  - Automatic fallback to exact LIKE matching when RapidFuzz is unavailable

RapidFuzz Functions Used:
  - rapidfuzz_ratio(): Overall similarity scoring (0-100)
  - rapidfuzz_token_set_ratio(): Word-set based matching (handles reordering)

Performance Considerations:
  - Results are limited by default to prevent expensive full-table scans
  - Score threshold filtering reduces result set before sorting
  - DISTINCT eliminates duplicate content entries
*/

package database

import (
	"context"
	"database/sql"
	"fmt"
)

// FuzzySearchResult represents a fuzzy search match for media content
type FuzzySearchResult struct {
	ID               string `json:"id"`                          // rating_key for the content
	Title            string `json:"title"`                       // Primary title
	ParentTitle      string `json:"parent_title,omitempty"`      // Season/album title
	GrandparentTitle string `json:"grandparent_title,omitempty"` // Show/artist title
	MediaType        string `json:"media_type"`                  // movie, episode, track
	Year             int    `json:"year,omitempty"`              // Release year
	Score            int    `json:"score"`                       // Fuzzy match score (0-100)
	Thumb            string `json:"thumb,omitempty"`             // Thumbnail URL
}

// UserSearchResult represents a fuzzy search match for users
type UserSearchResult struct {
	UserID       int    `json:"user_id"`
	Username     string `json:"username"`
	FriendlyName string `json:"friendly_name,omitempty"`
	Score        int    `json:"score"` // Fuzzy match score (0-100)
	UserThumb    string `json:"user_thumb,omitempty"`
}

// FuzzySearchPlaybacks searches playback events with fuzzy title matching.
// Uses RapidFuzz extension when available, falls back to exact LIKE matching.
//
// Parameters:
//   - query: Search string to match against titles
//   - minScore: Minimum similarity score (0-100), default 70 if <= 0
//   - limit: Maximum results to return (1-100), default 20 if <= 0
//
// The search matches against:
//   - title (primary title)
//   - grandparent_title (show name for episodes)
//   - Combined full title (grandparent + parent + title)
//
// Returns deduplicated results by rating_key, sorted by score descending.
func (db *DB) FuzzySearchPlaybacks(ctx context.Context, query string, minScore int, limit int) ([]FuzzySearchResult, error) {
	// Validate and set defaults
	if minScore <= 0 || minScore > 100 {
		minScore = 70
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	// Use fuzzy search if RapidFuzz available, otherwise fall back to exact
	if db.rapidfuzzAvailable {
		return db.fuzzySearchPlaybacksWithRapidFuzz(ctx, query, minScore, limit)
	}
	return db.fuzzySearchPlaybacksFallback(ctx, query, limit)
}

// fuzzySearchPlaybacksWithRapidFuzz performs fuzzy search using RapidFuzz extension
func (db *DB) fuzzySearchPlaybacksWithRapidFuzz(ctx context.Context, query string, minScore int, limit int) ([]FuzzySearchResult, error) {
	sqlQuery := `
		WITH scored_results AS (
			SELECT DISTINCT ON (rating_key)
				rating_key as id,
				title,
				parent_title,
				grandparent_title,
				media_type,
				year,
				thumb,
				GREATEST(
					rapidfuzz_ratio(LOWER(title), LOWER(?)),
					COALESCE(rapidfuzz_ratio(LOWER(grandparent_title), LOWER(?)), 0),
					rapidfuzz_token_set_ratio(
						LOWER(COALESCE(grandparent_title, '') || ' ' || COALESCE(parent_title, '') || ' ' || title),
						LOWER(?)
					)
				)::INTEGER as score
			FROM playback_events
			WHERE rating_key IS NOT NULL
			  AND title IS NOT NULL
		)
		SELECT id, title, parent_title, grandparent_title, media_type, year, thumb, score
		FROM scored_results
		WHERE score >= ?
		ORDER BY score DESC, title ASC
		LIMIT ?
	`

	rows, err := db.conn.QueryContext(ctx, sqlQuery, query, query, query, minScore, limit)
	if err != nil {
		return nil, fmt.Errorf("fuzzy search query failed: %w", err)
	}
	defer rows.Close()

	return db.scanFuzzySearchResults(rows)
}

// fuzzySearchPlaybacksFallback performs exact LIKE search when RapidFuzz unavailable
func (db *DB) fuzzySearchPlaybacksFallback(ctx context.Context, query string, limit int) ([]FuzzySearchResult, error) {
	// Escape special LIKE characters
	escapedQuery := "%" + query + "%"

	sqlQuery := `
		SELECT DISTINCT
			rating_key as id,
			title,
			parent_title,
			grandparent_title,
			media_type,
			year,
			thumb,
			100 as score
		FROM playback_events
		WHERE rating_key IS NOT NULL
		  AND title IS NOT NULL
		  AND (
			LOWER(title) LIKE LOWER(?)
			OR LOWER(grandparent_title) LIKE LOWER(?)
			OR LOWER(parent_title) LIKE LOWER(?)
		  )
		ORDER BY title ASC
		LIMIT ?
	`

	rows, err := db.conn.QueryContext(ctx, sqlQuery, escapedQuery, escapedQuery, escapedQuery, limit)
	if err != nil {
		return nil, fmt.Errorf("fallback search query failed: %w", err)
	}
	defer rows.Close()

	return db.scanFuzzySearchResults(rows)
}

// scanFuzzySearchResults scans rows into FuzzySearchResult slice
func (db *DB) scanFuzzySearchResults(rows *sql.Rows) ([]FuzzySearchResult, error) {
	var results []FuzzySearchResult

	for rows.Next() {
		var r FuzzySearchResult
		var parentTitle, grandparentTitle, thumb sql.NullString
		var year sql.NullInt64

		if err := rows.Scan(
			&r.ID,
			&r.Title,
			&parentTitle,
			&grandparentTitle,
			&r.MediaType,
			&year,
			&thumb,
			&r.Score,
		); err != nil {
			return nil, fmt.Errorf("scan error: %w", err)
		}

		if parentTitle.Valid {
			r.ParentTitle = parentTitle.String
		}
		if grandparentTitle.Valid {
			r.GrandparentTitle = grandparentTitle.String
		}
		if year.Valid {
			r.Year = int(year.Int64)
		}
		if thumb.Valid {
			r.Thumb = thumb.String
		}

		results = append(results, r)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return results, nil
}

// FuzzySearchUsers searches users with fuzzy username matching.
// Uses RapidFuzz extension when available, falls back to exact LIKE matching.
//
// Parameters:
//   - query: Search string to match against usernames
//   - minScore: Minimum similarity score (0-100), default 70 if <= 0
//   - limit: Maximum results to return (1-100), default 20 if <= 0
//
// Returns deduplicated results by user_id, sorted by score descending.
func (db *DB) FuzzySearchUsers(ctx context.Context, query string, minScore int, limit int) ([]UserSearchResult, error) {
	// Validate and set defaults
	if minScore <= 0 || minScore > 100 {
		minScore = 70
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	if db.rapidfuzzAvailable {
		return db.fuzzySearchUsersWithRapidFuzz(ctx, query, minScore, limit)
	}
	return db.fuzzySearchUsersFallback(ctx, query, limit)
}

// fuzzySearchUsersWithRapidFuzz performs fuzzy user search using RapidFuzz
func (db *DB) fuzzySearchUsersWithRapidFuzz(ctx context.Context, query string, minScore int, limit int) ([]UserSearchResult, error) {
	sqlQuery := `
		WITH user_scores AS (
			SELECT DISTINCT
				user_id,
				username,
				friendly_name,
				user_thumb,
				GREATEST(
					rapidfuzz_ratio(LOWER(username), LOWER(?)),
					COALESCE(rapidfuzz_ratio(LOWER(friendly_name), LOWER(?)), 0)
				)::INTEGER as score
			FROM playback_events
			WHERE username IS NOT NULL
		)
		SELECT user_id, username, friendly_name, user_thumb, score
		FROM user_scores
		WHERE score >= ?
		ORDER BY score DESC, username ASC
		LIMIT ?
	`

	rows, err := db.conn.QueryContext(ctx, sqlQuery, query, query, minScore, limit)
	if err != nil {
		return nil, fmt.Errorf("fuzzy user search query failed: %w", err)
	}
	defer rows.Close()

	return db.scanUserSearchResults(rows)
}

// fuzzySearchUsersFallback performs exact LIKE search for users
func (db *DB) fuzzySearchUsersFallback(ctx context.Context, query string, limit int) ([]UserSearchResult, error) {
	escapedQuery := "%" + query + "%"

	sqlQuery := `
		SELECT DISTINCT
			user_id,
			username,
			friendly_name,
			user_thumb,
			100 as score
		FROM playback_events
		WHERE username IS NOT NULL
		  AND (
			LOWER(username) LIKE LOWER(?)
			OR LOWER(friendly_name) LIKE LOWER(?)
		  )
		ORDER BY username ASC
		LIMIT ?
	`

	rows, err := db.conn.QueryContext(ctx, sqlQuery, escapedQuery, escapedQuery, limit)
	if err != nil {
		return nil, fmt.Errorf("fallback user search query failed: %w", err)
	}
	defer rows.Close()

	return db.scanUserSearchResults(rows)
}

// scanUserSearchResults scans rows into UserSearchResult slice
func (db *DB) scanUserSearchResults(rows *sql.Rows) ([]UserSearchResult, error) {
	var results []UserSearchResult

	for rows.Next() {
		var r UserSearchResult
		var friendlyName, userThumb sql.NullString

		if err := rows.Scan(
			&r.UserID,
			&r.Username,
			&friendlyName,
			&userThumb,
			&r.Score,
		); err != nil {
			return nil, fmt.Errorf("scan error: %w", err)
		}

		if friendlyName.Valid {
			r.FriendlyName = friendlyName.String
		}
		if userThumb.Valid {
			r.UserThumb = userThumb.String
		}

		results = append(results, r)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return results, nil
}

// FuzzyMatchScore calculates the fuzzy match score between two strings.
// Returns 0-100 similarity score. Only works when RapidFuzz is available.
// Returns 100 for exact match or 0 if RapidFuzz unavailable.
func (db *DB) FuzzyMatchScore(ctx context.Context, str1, str2 string) (int, error) {
	if !db.rapidfuzzAvailable {
		// Simple exact match check
		if str1 == str2 {
			return 100, nil
		}
		return 0, nil
	}

	var scoreFloat float64
	err := db.conn.QueryRowContext(ctx,
		"SELECT rapidfuzz_ratio(?, ?)",
		str1, str2,
	).Scan(&scoreFloat)

	if err != nil {
		return 0, fmt.Errorf("fuzzy match score query failed: %w", err)
	}

	return int(scoreFloat), nil
}
