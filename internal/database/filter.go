// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package database

import (
	"fmt"
	"time"
)

// LocationStatsFilter contains filter parameters for location statistics and analytics queries.
// Provides comprehensive multi-dimensional filtering with 14+ filter dimensions, supporting
// complex analytics queries across temporal, user, content, technical, and geographic axes.
//
// All filter fields are optional and combine using AND logic. Multi-select fields (slices)
// use OR logic within the field (e.g., Users: ["alice", "bob"] matches alice OR bob).
//
// Filter Dimensions:
//
//  1. Temporal Filtering:
//     - StartDate: Filter events on or after this timestamp (nil = no start limit)
//     - EndDate: Filter events on or before this timestamp (nil = no end limit)
//     - Years: Filter by content release year (supports multiple years)
//
//  2. User Filtering:
//     - Users: Filter by usernames (multi-select OR)
//
//  3. Content Filtering:
//     - MediaTypes: Filter by media type ("movie", "episode", "track", "photo")
//     - Libraries: Filter by library name (multi-select OR)
//     - ContentRatings: Filter by content rating ("G", "PG", "PG-13", "R", etc.)
//
//  4. Technical Filtering:
//     - Platforms: Filter by platform/OS ("iOS", "Android", "Web", etc.)
//     - Players: Filter by player app ("Plex Web", "Plex for iOS", etc.)
//     - TranscodeDecisions: Filter by transcode decision ("direct play", "transcode", "copy")
//     - VideoResolutions: Filter by video resolution ("4k", "1080p", "720p", etc.)
//     - VideoCodecs: Filter by video codec ("h264", "hevc", "vp9", etc.)
//     - AudioCodecs: Filter by audio codec ("aac", "ac3", "dts", etc.)
//
//  5. Geographic Filtering:
//     - LocationTypes: Filter by location type ("country", "city", "isp")
//
//  6. Server Filtering (v2.1 Multi-Server Support):
//     - ServerIDs: Filter by server ID ("plex-home", "jellyfin-abc123", etc.)
//
//  7. Result Limiting:
//     - Limit: Maximum number of results to return (0 = no limit)
//
// Example - Basic temporal filter:
//
//	filter := LocationStatsFilter{
//	    StartDate: &time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
//	    EndDate:   &time.Date(2025, 1, 31, 23, 59, 59, 0, time.UTC),
//	    Limit:     100,
//	}
//
// Example - Multi-dimensional analytics filter:
//
//	now := time.Now()
//	thirtyDaysAgo := now.AddDate(0, 0, -30)
//	filter := LocationStatsFilter{
//	    StartDate:        &thirtyDaysAgo,
//	    EndDate:          &now,
//	    Users:            []string{"alice", "bob"},            // alice OR bob
//	    MediaTypes:       []string{"movie", "episode"},        // movies OR episodes
//	    TranscodeDecisions: []string{"transcode"},             // only transcoded streams
//	    VideoResolutions: []string{"4k", "1080p"},             // 4K OR 1080p
//	    Platforms:        []string{"iOS", "Android"},          // mobile only
//	    Limit:            50,
//	}
//
// SQL Generation:
// This filter generates parameterized SQL WHERE clauses via buildFilterConditions():
//
//	// Example generated SQL:
//	WHERE started_at >= ? AND started_at <= ?
//	  AND username IN (?, ?)
//	  AND media_type IN (?, ?)
//	  AND transcode_decision = ?
//	  AND video_resolution IN (?, ?)
//	  AND platform IN (?, ?)
//	LIMIT ?
//
// Performance Notes:
//   - All fields use indexed columns for efficient filtering
//   - Multi-select filters use IN clauses (optimized by DuckDB query planner)
//   - Date range filters leverage composite index on (started_at DESC, id)
//   - Typical query time: 5-50ms with proper indexing
//
// Thread Safety:
// LocationStatsFilter is immutable after creation and safe for concurrent read access.
// Multiple goroutines can safely pass the same filter to different query methods.
type LocationStatsFilter struct {
	StartDate          *time.Time
	EndDate            *time.Time
	Users              []string
	MediaTypes         []string
	Platforms          []string
	Players            []string
	TranscodeDecisions []string
	VideoResolutions   []string
	VideoCodecs        []string
	AudioCodecs        []string
	Libraries          []string
	ContentRatings     []string
	Years              []int
	LocationTypes      []string
	ServerIDs          []string // v2.1: Multi-server support - filter by server ID
	Limit              int
}

// appendInClause is a generic helper for building SQL IN clauses
// Eliminates code duplication across 12+ filter dimensions
//
// Parameters:
//   - columnName: SQL column name (e.g., "username", "media_type")
//   - values: Slice of values to filter by (string or int)
//   - whereClauses: Accumulator for WHERE clause conditions
//   - args: Accumulator for query parameters
//   - argPos: Current parameter position (for positional params)
//   - usePositionalParams: Whether to use $N syntax (PostgreSQL) vs ? (SQLite/DuckDB)
//
// Example:
//
//	appendInClause("username", []interface{}{"alice", "bob"}, &clauses, &args, &pos, false)
//	// Adds: "username IN (?, ?)" to clauses
//	// Adds: ["alice", "bob"] to args
func appendInClause(columnName string, values interface{}, whereClauses *[]string, args *[]interface{}, argPos *int, usePositionalParams bool) {
	// Handle different slice types using type assertion
	var length int
	var getValue func(int) interface{}

	switch v := values.(type) {
	case []string:
		length = len(v)
		getValue = func(i int) interface{} { return v[i] }
	case []int:
		length = len(v)
		getValue = func(i int) interface{} { return v[i] }
	default:
		return // Unknown type, skip
	}

	if length == 0 {
		return
	}

	// Build placeholders and collect values
	placeholders := make([]string, length)
	for i := 0; i < length; i++ {
		if usePositionalParams {
			placeholders[i] = fmt.Sprintf("$%d", *argPos)
		} else {
			placeholders[i] = "?"
		}
		*args = append(*args, getValue(i))
		*argPos++
	}

	// Add IN clause
	*whereClauses = append(*whereClauses, fmt.Sprintf("%s IN (%s)", columnName, join(placeholders, ", ")))
}

// buildFilterConditions builds WHERE clause conditions and args from a LocationStatsFilter
// Returns (whereClauses, args) that can be used to build parameterized queries
//
// Refactored to use appendInClause helper for DRY principle (41 → 8 complexity)
//
//nolint:gocyclo // Acceptable complexity for centralized filter building with 14+ dimensions
func buildFilterConditions(filter LocationStatsFilter, usePositionalParams bool, startArgPos int) ([]string, []interface{}) {
	whereClauses := []string{}
	args := []interface{}{}
	argPos := startArgPos

	// Date range filters (special handling needed)
	if filter.StartDate != nil {
		if usePositionalParams {
			whereClauses = append(whereClauses, fmt.Sprintf("started_at >= $%d", argPos))
		} else {
			whereClauses = append(whereClauses, "started_at >= ?")
		}
		args = append(args, *filter.StartDate)
		argPos++
	}

	if filter.EndDate != nil {
		if usePositionalParams {
			whereClauses = append(whereClauses, fmt.Sprintf("started_at <= $%d", argPos))
		} else {
			whereClauses = append(whereClauses, "started_at <= ?")
		}
		args = append(args, *filter.EndDate)
		argPos++
	}

	// Multi-value filters using generic helper (13 filter dimensions)
	appendInClause("username", filter.Users, &whereClauses, &args, &argPos, usePositionalParams)
	appendInClause("media_type", filter.MediaTypes, &whereClauses, &args, &argPos, usePositionalParams)
	appendInClause("platform", filter.Platforms, &whereClauses, &args, &argPos, usePositionalParams)
	appendInClause("player", filter.Players, &whereClauses, &args, &argPos, usePositionalParams)
	appendInClause("transcode_decision", filter.TranscodeDecisions, &whereClauses, &args, &argPos, usePositionalParams)
	appendInClause("video_resolution", filter.VideoResolutions, &whereClauses, &args, &argPos, usePositionalParams)
	appendInClause("video_codec", filter.VideoCodecs, &whereClauses, &args, &argPos, usePositionalParams)
	appendInClause("audio_codec", filter.AudioCodecs, &whereClauses, &args, &argPos, usePositionalParams)
	appendInClause("library_name", filter.Libraries, &whereClauses, &args, &argPos, usePositionalParams)
	appendInClause("content_rating", filter.ContentRatings, &whereClauses, &args, &argPos, usePositionalParams)
	appendInClause("year", filter.Years, &whereClauses, &args, &argPos, usePositionalParams)
	appendInClause("location_type", filter.LocationTypes, &whereClauses, &args, &argPos, usePositionalParams)
	appendInClause("server_id", filter.ServerIDs, &whereClauses, &args, &argPos, usePositionalParams) // v2.1: Multi-server support

	return whereClauses, args
}

// join is a helper function to join strings with a separator
func join(strs []string, sep string) string {
	result := ""
	for i, s := range strs {
		if i > 0 {
			result += sep
		}
		result += s
	}
	return result
}

// buildFilterWhereClause builds a WHERE clause string with "1=1" base for safe concatenation.
// This helper wraps buildFilterConditions to return a single WHERE clause string and args.
//
// Returns: WHERE clause string (e.g., "1=1 AND started_at >= ? AND username IN (?, ?)") and query arguments
//
// Example:
//
//	whereClause, args := buildFilterWhereClause(filter)
//	query := fmt.Sprintf("SELECT * FROM playback_events WHERE %s", whereClause)
//	rows, err := db.Query(query, args...)
func buildFilterWhereClause(filter LocationStatsFilter) (string, []interface{}) {
	clauses, args := buildFilterConditions(filter, false, 1)

	// Start with "1=1" for safe AND concatenation
	if len(clauses) == 0 {
		return "1=1", args
	}

	return "1=1 AND " + join(clauses, " AND "), args
}
