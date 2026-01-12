// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/tomtom215/cartographus/internal/models"
)

// GetConnectionSecurityAnalytics analyzes connection security patterns including secure vs insecure
// connections, relayed connection statistics, user-specific security rates, and platform security profiles
// for network security optimization and CGNAT detection.
func (db *DB) GetConnectionSecurityAnalytics(ctx context.Context, filter LocationStatsFilter) (*models.ConnectionSecurityAnalytics, error) {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	whereClauses, args := buildFilterConditions(filter, false, 1)
	whereClause := buildWhereClause(whereClauses)

	// Get overall security statistics
	total, secureCount, insecureCount, relayedCount, localCount, securePercent, relayedPercent, localPercent, err := db.getConnectionSecurityStats(ctx, whereClause, args)
	if err != nil {
		return nil, errorContext("get security stats", err)
	}

	// Get list of relayed users
	relayedUsers, err := db.getRelayedConnectionUsers(ctx, whereClause, args)
	if err != nil {
		return nil, errorContext("get relayed users", err)
	}

	// Get user connection statistics
	userStats, err := db.getUserConnectionStats(ctx, whereClause, args)
	if err != nil {
		return nil, errorContext("get user stats", err)
	}

	// Get platform connection statistics
	platformStats, err := db.getPlatformConnectionStats(ctx, whereClause, args)
	if err != nil {
		return nil, errorContext("get platform stats", err)
	}

	return &models.ConnectionSecurityAnalytics{
		TotalPlaybacks:      total,
		SecureConnections:   secureCount,
		InsecureConnections: insecureCount,
		SecurePercent:       securePercent,
		RelayedConnections: models.ConnectionRelayStats{
			Count:   relayedCount,
			Percent: relayedPercent,
			Users:   relayedUsers,
			Reason:  "Port forwarding or CGNAT issues",
		},
		LocalConnections: models.ConnectionLocalStats{
			Count:   localCount,
			Percent: localPercent,
		},
		ByUser:     userStats,
		ByPlatform: platformStats,
	}, nil
}

// getConnectionSecurityStats retrieves overall connection security statistics including total playbacks,
// secure/insecure counts, relayed/local counts, and calculates percentage rates for each category.
func (db *DB) getConnectionSecurityStats(ctx context.Context, whereClause string, args []interface{}) (int, int, int, int, int, float64, float64, float64, error) {
	query := fmt.Sprintf(`
		SELECT
			COUNT(*) as total,
			COALESCE(SUM(CASE WHEN secure = 1 THEN 1 ELSE 0 END), 0) as secure_count,
			COALESCE(SUM(CASE WHEN secure = 0 OR secure IS NULL THEN 1 ELSE 0 END), 0) as insecure_count,
			COALESCE(SUM(CASE WHEN relayed = 1 THEN 1 ELSE 0 END), 0) as relayed_count,
			COALESCE(SUM(CASE WHEN local = 1 THEN 1 ELSE 0 END), 0) as local_count
		FROM playback_events
		%s
	`, whereClause)

	var total, secureCount, insecureCount, relayedCount, localCount int
	err := db.conn.QueryRowContext(ctx, query, args...).Scan(&total, &secureCount, &insecureCount, &relayedCount, &localCount)
	if err != nil {
		return 0, 0, 0, 0, 0, 0, 0, 0, fmt.Errorf("failed to get security stats: %w", err)
	}

	securePercent := calculatePercentage(secureCount, total)
	relayedPercent := calculatePercentage(relayedCount, total)
	localPercent := calculatePercentage(localCount, total)

	return total, secureCount, insecureCount, relayedCount, localCount, securePercent, relayedPercent, localPercent, nil
}

// getRelayedConnectionUsers retrieves list of users experiencing relayed connections
// (typically due to port forwarding or CGNAT issues).
func (db *DB) getRelayedConnectionUsers(ctx context.Context, whereClause string, args []interface{}) ([]string, error) {
	relayedCondition := "relayed = 1"
	relayedWhere := appendWhereCondition(whereClause, relayedCondition)

	query := fmt.Sprintf(`
		SELECT string_agg(DISTINCT username, ',')
		FROM playback_events
		%s
	`, relayedWhere)

	var relayedUsersStr sql.NullString
	err := db.conn.QueryRowContext(ctx, query, args...).Scan(&relayedUsersStr)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("failed to get relayed users: %w", err)
	}

	var relayedUsers []string
	if relayedUsersStr.Valid {
		relayedUsers = parseAggregatedList(relayedUsersStr.String)
	}

	return relayedUsers, nil
}

// getUserConnectionStats retrieves top 15 users with their connection security statistics
// including total streams, secure rate, relay rate, and local connection rate.
func (db *DB) getUserConnectionStats(ctx context.Context, whereClause string, args []interface{}) ([]models.UserConnectionStats, error) {
	query := fmt.Sprintf(`
		SELECT
			username,
			COUNT(*) as total_streams,
			(SUM(CASE WHEN secure = 1 THEN 1 ELSE 0 END) * 100.0 / COUNT(*)) as secure_rate,
			(SUM(CASE WHEN relayed = 1 THEN 1 ELSE 0 END) * 100.0 / COUNT(*)) as relay_rate,
			(SUM(CASE WHEN local = 1 THEN 1 ELSE 0 END) * 100.0 / COUNT(*)) as local_rate
		FROM playback_events
		%s
		GROUP BY username
		ORDER BY total_streams DESC
		LIMIT 15
	`, whereClause)

	rows, err := db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query user stats: %w", err)
	}
	defer rows.Close()

	var userStats []models.UserConnectionStats
	for rows.Next() {
		var u models.UserConnectionStats
		err := rows.Scan(&u.Username, &u.TotalStreams, &u.SecureRate, &u.RelayRate, &u.LocalRate)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user row: %w", err)
		}
		userStats = append(userStats, u)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating user rows: %w", err)
	}

	return userStats, nil
}

// getPlatformConnectionStats retrieves top 15 platforms with their security profiles
// including secure connection rate and relay rate for platform capability analysis.
func (db *DB) getPlatformConnectionStats(ctx context.Context, whereClause string, args []interface{}) ([]models.PlatformConnectionStats, error) {
	query := fmt.Sprintf(`
		SELECT
			platform,
			(SUM(CASE WHEN secure = 1 THEN 1 ELSE 0 END) * 100.0 / COUNT(*)) as secure_rate,
			(SUM(CASE WHEN relayed = 1 THEN 1 ELSE 0 END) * 100.0 / COUNT(*)) as relay_rate
		FROM playback_events
		%s
		GROUP BY platform
		ORDER BY secure_rate DESC
		LIMIT 15
	`, whereClause)

	rows, err := db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query platform stats: %w", err)
	}
	defer rows.Close()

	var platformStats []models.PlatformConnectionStats
	for rows.Next() {
		var p models.PlatformConnectionStats
		err := rows.Scan(&p.Platform, &p.SecureRate, &p.RelayRate)
		if err != nil {
			return nil, fmt.Errorf("failed to scan platform row: %w", err)
		}
		platformStats = append(platformStats, p)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating platform rows: %w", err)
	}

	return platformStats, nil
}
