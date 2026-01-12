// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package database

import (
	"context"
	"fmt"
)

// calculatePercentage computes percentage with zero-division safety
func calculatePercentage(numerator, denominator int) float64 {
	if denominator == 0 {
		return 0.0
	}
	return (float64(numerator) / float64(denominator)) * 100.0
}

// buildWhereClause constructs WHERE clause from filter conditions
// Returns empty string if no conditions, otherwise "WHERE condition1 AND condition2..."
func buildWhereClause(whereClauses []string) string {
	if len(whereClauses) == 0 {
		return ""
	}
	return "WHERE " + joinWithAnd(whereClauses)
}

// buildAndWhereClause constructs AND-prefixed WHERE clause for appending to existing WHERE
// Returns empty string if no conditions, otherwise "condition1 AND condition2..."
func buildAndWhereClause(whereClauses []string) string {
	if len(whereClauses) == 0 {
		return ""
	}
	return joinWithAnd(whereClauses)
}

// joinWithAnd joins string slices with " AND "
func joinWithAnd(clauses []string) string {
	result := ""
	for i, clause := range clauses {
		if i > 0 {
			result += " AND "
		}
		result += clause
	}
	return result
}

// appendWhereCondition appends a condition to WHERE clause with proper AND prefix
// Example: appendWhereCondition(baseWhere, "column IS NOT NULL")
func appendWhereCondition(baseWhere, condition string) string {
	if baseWhere == "" {
		return "WHERE " + condition
	}
	return baseWhere + " AND " + condition
}

// querySingleRow executes a query and scans a single row into dest variables
func (db *DB) querySingleRow(
	ctx context.Context,
	query string,
	args []interface{},
	dest ...interface{},
) error {
	return db.conn.QueryRowContext(ctx, query, args...).Scan(dest...)
}

// getTotalPlaybacks queries the total playback count for baseline metrics
func (db *DB) getTotalPlaybacks(ctx context.Context, whereClause string, args []interface{}) (int, error) {
	query := fmt.Sprintf("SELECT COUNT(*) FROM playback_events %s", whereClause)
	var total int
	if err := db.conn.QueryRowContext(ctx, query, args...).Scan(&total); err != nil {
		return 0, fmt.Errorf("failed to get total playbacks: %w", err)
	}
	return total, nil
}

// parseAggregatedList parses DuckDB string_agg() output into slice
// Handles both comma-separated and DuckDB LIST format []
// Filters out NULL and empty strings
func parseAggregatedList(listStr string) []string {
	// Reuse existing parseList function from database_new_analytics.go
	return parseList(listStr)
}

// calculateAdoptionRate calculates adoption/usage rate as percentage
func calculateAdoptionRate(adoptedCount, total int) float64 {
	return calculatePercentage(adoptedCount, total)
}

// errorContext wraps an error with contextual information
func errorContext(operation string, err error) error {
	if err != nil {
		return fmt.Errorf("%s: %w", operation, err)
	}
	return nil
}
