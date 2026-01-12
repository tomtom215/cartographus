// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package database

import (
	"fmt"
	"strings"
)

// buildInClause creates a parameterized IN clause for SQL queries.
// Returns the placeholder string and the arguments slice.
//
// Example:
//
//	placeholders, args := buildInClause([]string{"user1", "user2", "user3"})
//	// placeholders = "?,?,?"
//	// args = []interface{}{"user1", "user2", "user3"}
func buildInClause(items []string) (string, []interface{}) {
	placeholders := make([]string, len(items))
	args := make([]interface{}, len(items))
	for i, item := range items {
		placeholders[i] = "?"
		args[i] = item
	}
	return strings.Join(placeholders, ","), args
}

// buildFilterConditions extracts common filter logic used across multiple queries.
// Builds WHERE clause conditions for LocationStatsFilter including:
// - Date range filtering (StartDate, EndDate)
// - User filtering (Users IN clause)
// - Media type filtering (MediaTypes IN clause)
//
// Returns SQL conditions (without WHERE keyword) and corresponding arguments.
// The base query should already have "WHERE 1=1" to which these conditions are appended.
func (f *LocationStatsFilter) buildFilterConditions() (string, []interface{}) {
	var conditions []string
	var args []interface{}

	if f.StartDate != nil {
		conditions = append(conditions, "p.started_at >= ?")
		args = append(args, *f.StartDate)
	}

	if f.EndDate != nil {
		conditions = append(conditions, "p.started_at <= ?")
		args = append(args, *f.EndDate)
	}

	if len(f.Users) > 0 {
		placeholders, userArgs := buildInClause(f.Users)
		conditions = append(conditions, fmt.Sprintf("p.username IN (%s)", placeholders))
		args = append(args, userArgs...)
	}

	if len(f.MediaTypes) > 0 {
		placeholders, typeArgs := buildInClause(f.MediaTypes)
		conditions = append(conditions, fmt.Sprintf("p.media_type IN (%s)", placeholders))
		args = append(args, typeArgs...)
	}

	// Join all conditions with AND
	if len(conditions) > 0 {
		return " AND " + strings.Join(conditions, " AND "), args
	}

	return "", args
}
