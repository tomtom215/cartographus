// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package query provides SQL query building utilities for the database package.
//
// This package reduces code duplication and provides type-safe query construction
// for parameterized SQL WHERE clauses. It ensures consistent parameter handling
// and prevents SQL injection vulnerabilities.
//
// # Overview
//
// The WhereBuilder is the primary component, providing a fluent interface for
// constructing WHERE clauses with properly parameterized queries:
//
//	wb := query.NewWhereBuilder()
//	wb.AddDateRange(startDate, endDate)
//	wb.AddUsers([]string{"alice", "bob"})
//	wb.AddMediaTypes([]string{"movie", "episode"})
//	whereClause, args := wb.Build()
//	// Result: "started_at >= ? AND started_at <= ? AND username IN (?, ?) AND media_type IN (?, ?)"
//	// Args: [startDate, endDate, "alice", "bob", "movie", "episode"]
//
// # Usage Example
//
// Building a query with multiple filters:
//
//	func GetFilteredPlaybacks(ctx context.Context, filter Filter) ([]Playback, error) {
//	    wb := query.NewWhereBuilder()
//	    wb.AddDateRange(filter.StartDate, filter.EndDate)
//	    wb.AddUsers(filter.Users)
//	    wb.AddMediaTypes(filter.MediaTypes)
//	    wb.AddPlatforms(filter.Platforms)
//	    wb.AddLibraries(filter.Libraries)
//
//	    whereClause, args := wb.Build()
//
//	    sql := fmt.Sprintf(`
//	        SELECT * FROM playback_events
//	        WHERE %s
//	        ORDER BY started_at DESC
//	        LIMIT ?
//	    `, whereClause)
//	    args = append(args, filter.Limit)
//
//	    rows, err := db.QueryContext(ctx, sql, args...)
//	    // ...
//	}
//
// Adding custom clauses:
//
//	wb := query.NewWhereBuilder()
//	wb.AddClause("bitrate >= ?", 5000)
//	wb.AddClause("duration > ?", 3600)
//	wb.AddClause("transcode_decision = ?", "transcode")
//
// # Available Filter Methods
//
// The WhereBuilder provides methods for common filter types:
//
//   - AddDateRange: Filters by started_at date range
//   - AddUsers: Filters by username list (IN clause)
//   - AddMediaTypes: Filters by media type (movie, episode, track)
//   - AddPlatforms: Filters by client platform (Roku, Web, iOS, etc.)
//   - AddLibraries: Filters by Plex library name
//   - AddClause: Adds custom WHERE clause with parameters
//
// # SQL Injection Prevention
//
// All methods use parameterized queries with ? placeholders:
//
//	// Safe - parameters are properly escaped by the database driver
//	wb.AddUsers(userInput)  // Generates: "username IN (?, ?)"
//
//	// The generated SQL is safe regardless of input content
//	// Never concatenate user input directly into SQL strings
//
// # Thread Safety
//
// WhereBuilder instances are not thread-safe. Create a new instance per query
// or protect concurrent access with appropriate synchronization.
//
// # Performance
//
//   - Zero allocations for empty builders (returns "1=1")
//   - Efficient string building using slices
//   - No reflection or dynamic SQL parsing
//
// # See Also
//
//   - internal/database: Main database package using this builder
//   - internal/models: Filter types used with the builder
package query
