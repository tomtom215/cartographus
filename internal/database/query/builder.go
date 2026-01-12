// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package query provides SQL query building utilities for the database package.
// It reduces code duplication and provides type-safe query construction.
package query

import (
	"fmt"
	"strings"
	"time"
)

// WhereBuilder constructs SQL WHERE clauses with parameterized arguments.
// It ensures consistent parameter handling and reduces SQL injection risks.
//
// Example usage:
//
//	wb := query.NewWhereBuilder()
//	wb.AddDateRange(startDate, endDate)
//	wb.AddUsers([]string{"user1", "user2"})
//	whereClause, args := wb.Build()
//	// WHERE started_at >= ? AND started_at <= ? AND username IN (?, ?)
type WhereBuilder struct {
	clauses []string
	args    []interface{}
}

// NewWhereBuilder creates a new WhereBuilder instance.
func NewWhereBuilder() *WhereBuilder {
	return &WhereBuilder{
		clauses: []string{},
		args:    []interface{}{},
	}
}

// AddClause adds a raw WHERE clause with its arguments.
// This is useful for custom conditions not covered by helper methods.
//
// Parameters:
//   - clause: SQL condition fragment (e.g., "media_type = ?")
//   - args: Arguments to bind to placeholders in the clause
func (wb *WhereBuilder) AddClause(clause string, args ...interface{}) *WhereBuilder {
	wb.clauses = append(wb.clauses, clause)
	wb.args = append(wb.args, args...)
	return wb
}

// AddDateRange adds start and/or end date filters to the WHERE clause.
// Nil dates are skipped, allowing flexible date range queries.
//
// Parameters:
//   - startDate: Optional start date (nil to skip)
//   - endDate: Optional end date (nil to skip)
//
// Generates:
//   - "started_at >= ?" if startDate is non-nil
//   - "started_at <= ?" if endDate is non-nil
func (wb *WhereBuilder) AddDateRange(startDate, endDate *time.Time) *WhereBuilder {
	if startDate != nil {
		wb.clauses = append(wb.clauses, "started_at >= ?")
		wb.args = append(wb.args, *startDate)
	}
	if endDate != nil {
		wb.clauses = append(wb.clauses, "started_at <= ?")
		wb.args = append(wb.args, *endDate)
	}
	return wb
}

// AddUsers adds a user filter using IN clause.
// Generates "username IN (?, ?, ...)" with proper parameterization.
//
// Parameters:
//   - users: List of usernames to filter (empty slice is skipped)
func (wb *WhereBuilder) AddUsers(users []string) *WhereBuilder {
	if len(users) > 0 {
		placeholders := make([]string, len(users))
		for i, user := range users {
			placeholders[i] = "?"
			wb.args = append(wb.args, user)
		}
		wb.clauses = append(wb.clauses, fmt.Sprintf("username IN (%s)", strings.Join(placeholders, ", ")))
	}
	return wb
}

// AddMediaTypes adds a media type filter using IN clause.
// Generates "media_type IN (?, ?, ...)" for filtering by content type.
//
// Parameters:
//   - mediaTypes: List of media types ("movie", "episode", "track")
func (wb *WhereBuilder) AddMediaTypes(mediaTypes []string) *WhereBuilder {
	if len(mediaTypes) > 0 {
		placeholders := make([]string, len(mediaTypes))
		for i, mediaType := range mediaTypes {
			placeholders[i] = "?"
			wb.args = append(wb.args, mediaType)
		}
		wb.clauses = append(wb.clauses, fmt.Sprintf("media_type IN (%s)", strings.Join(placeholders, ", ")))
	}
	return wb
}

// AddPlatforms adds a platform filter using IN clause.
// Generates "platform IN (?, ?, ...)" for filtering by client platform.
//
// Parameters:
//   - platforms: List of platform names ("Roku", "Apple TV", "Web", etc.)
func (wb *WhereBuilder) AddPlatforms(platforms []string) *WhereBuilder {
	if len(platforms) > 0 {
		placeholders := make([]string, len(platforms))
		for i, platform := range platforms {
			placeholders[i] = "?"
			wb.args = append(wb.args, platform)
		}
		wb.clauses = append(wb.clauses, fmt.Sprintf("platform IN (%s)", strings.Join(placeholders, ", ")))
	}
	return wb
}

// AddLibraries adds a library filter using IN clause.
// Generates "library_name IN (?, ?, ...)" for filtering by Plex library.
//
// Parameters:
//   - libraries: List of library names ("Movies", "TV Shows", etc.)
func (wb *WhereBuilder) AddLibraries(libraries []string) *WhereBuilder {
	if len(libraries) > 0 {
		placeholders := make([]string, len(libraries))
		for i, library := range libraries {
			placeholders[i] = "?"
			wb.args = append(wb.args, library)
		}
		wb.clauses = append(wb.clauses, fmt.Sprintf("library_name IN (%s)", strings.Join(placeholders, ", ")))
	}
	return wb
}

// Build constructs the final WHERE clause and returns it with arguments.
// Clauses are joined with "AND". Returns ("1=1", []) if no clauses were added.
//
// Returns:
//   - string: Complete WHERE clause (without "WHERE" keyword)
//   - []interface{}: Arguments to bind to placeholders
//
// Example:
//
//	whereClause, args := wb.Build()
//	query := fmt.Sprintf("SELECT * FROM table WHERE %s", whereClause)
//	db.Query(query, args...)
func (wb *WhereBuilder) Build() (string, []interface{}) {
	if len(wb.clauses) == 0 {
		return "1=1", []interface{}{}
	}
	return strings.Join(wb.clauses, " AND "), wb.args
}

// BuildWithPrefix returns the WHERE clause with "WHERE " prefix.
// Useful for direct SQL construction without manual prefix addition.
//
// Returns:
//   - string: Complete WHERE clause with "WHERE " prefix
//   - []interface{}: Arguments to bind to placeholders
func (wb *WhereBuilder) BuildWithPrefix() (string, []interface{}) {
	whereClause, args := wb.Build()
	return "WHERE " + whereClause, args
}

// Count returns the number of clauses added to the builder.
// Useful for conditional logic based on filter complexity.
func (wb *WhereBuilder) Count() int {
	return len(wb.clauses)
}

// IsEmpty returns true if no clauses have been added.
func (wb *WhereBuilder) IsEmpty() bool {
	return len(wb.clauses) == 0
}
