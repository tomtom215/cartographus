// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// queryBuilder helps construct SQL queries with filters
type queryBuilder struct {
	baseQuery string
	args      []interface{}
	filters   []string
}

// newQueryBuilder creates a new query builder with a base query.
func newQueryBuilder(baseQuery string) *queryBuilder {
	return &queryBuilder{
		baseQuery: baseQuery,
		args:      make([]interface{}, 0, 8),
		filters:   make([]string, 0, 4),
	}
}

// addDateRangeFilter adds date range filtering to the query
func (qb *queryBuilder) addDateRangeFilter(filter LocationStatsFilter) *queryBuilder {
	if filter.StartDate != nil {
		qb.filters = append(qb.filters, "started_at >= ?")
		qb.args = append(qb.args, *filter.StartDate)
	}
	if filter.EndDate != nil {
		qb.filters = append(qb.filters, "started_at <= ?")
		qb.args = append(qb.args, *filter.EndDate)
	}
	return qb
}

// addUsersFilter adds user filtering to the query
func (qb *queryBuilder) addUsersFilter(users []string) *queryBuilder {
	if len(users) > 0 {
		placeholders := make([]string, len(users))
		for i, user := range users {
			placeholders[i] = "?"
			qb.args = append(qb.args, user)
		}
		qb.filters = append(qb.filters, fmt.Sprintf("username IN (%s)", strings.Join(placeholders, ",")))
	}
	return qb
}

// addMediaTypesFilter adds media type filtering to the query
func (qb *queryBuilder) addMediaTypesFilter(mediaTypes []string) *queryBuilder {
	if len(mediaTypes) > 0 {
		placeholders := make([]string, len(mediaTypes))
		for i, mediaType := range mediaTypes {
			placeholders[i] = "?"
			qb.args = append(qb.args, mediaType)
		}
		qb.filters = append(qb.filters, fmt.Sprintf("media_type IN (%s)", strings.Join(placeholders, ",")))
	}
	return qb
}

// addStandardFilters applies all standard filters (date range, users, media types)
func (qb *queryBuilder) addStandardFilters(filter LocationStatsFilter) *queryBuilder {
	return qb.addDateRangeFilter(filter).
		addUsersFilter(filter.Users).
		addMediaTypesFilter(filter.MediaTypes)
}

// addFilter adds a custom filter condition
func (qb *queryBuilder) addFilter(condition string, args ...interface{}) {
	qb.filters = append(qb.filters, condition)
	qb.args = append(qb.args, args...)
}

// addLimit adds a LIMIT clause (does not use filters slice)
func (qb *queryBuilder) addLimit(limit int) *queryBuilder {
	qb.args = append(qb.args, limit)
	return qb
}

// build constructs the final query and returns it with args
func (qb *queryBuilder) build(suffix string) (string, []interface{}) {
	query := qb.baseQuery
	if len(qb.filters) > 0 {
		query += " AND " + strings.Join(qb.filters, " AND ")
	}
	if suffix != "" {
		query += " " + suffix
	}
	return query, qb.args
}

// scanFunc is a function that scans a single row into a result type
type scanFunc[T any] func(*sql.Rows) (T, error)

// queryAndScan executes a query and scans all rows using the provided scan function
func queryAndScan[T any](ctx context.Context, db *sql.DB, query string, args []interface{}, scan scanFunc[T]) ([]T, error) {
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []T
	for rows.Next() {
		item, err := scan(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}
