// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

/*
database_connection.go - Connection Management

This file provides connection pool configuration and error detection utilities.

Connection Pool Configuration:
  - MaxOpenConns: Based on CPU count for parallelism
  - MaxIdleConns: 2 for efficient connection reuse
  - ConnMaxLifetime: 1 hour to prevent stale connections
  - ConnMaxIdleTime: 5 minutes for idle connection cleanup

Error Detection:
The package identifies connection errors vs query errors to determine
appropriate error handling and recovery strategies.
*/

//nolint:staticcheck // File documentation, not package doc
package database

import (
	"runtime"
	"strings"
	"time"
)

// isConnectionError checks if an error indicates database connection loss
func isConnectionError(err error) bool {
	if err == nil {
		return false
	}
	errMsg := err.Error()
	return stringContains(errMsg, "connection refused") ||
		stringContains(errMsg, "connection reset") ||
		stringContains(errMsg, "broken pipe") ||
		stringContains(errMsg, "bad connection") ||
		stringContains(errMsg, "driver: bad connection") ||
		stringContains(errMsg, "database is closed") ||
		stringContains(errMsg, "sql: database is closed")
}

// configureConnectionPool sets connection pool parameters
func (db *DB) configureConnectionPool() error {
	db.conn.SetMaxOpenConns(runtime.NumCPU())
	db.conn.SetMaxIdleConns(2)
	db.conn.SetConnMaxLifetime(time.Hour)
	db.conn.SetConnMaxIdleTime(5 * time.Minute)

	// Note: Connection pool settings:
	// - max_open: NumCPU() for parallelism
	// - max_idle: 2 for connection reuse
	// - max_lifetime: 1h to prevent stale connections
	// - max_idle_time: 5m for idle connection cleanup

	return nil
}

// isTransactionConflict checks if an error is a DuckDB transaction conflict
func isTransactionConflict(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "Transaction conflict") ||
		strings.Contains(errStr, "Conflict on update") ||
		strings.Contains(errStr, "cannot update a table that has been altered")
}

// isInternalError checks if an error is a DuckDB INTERNAL error
func isInternalError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "INTERNAL Error")
}

// Helper string functions
func stringContains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || stringIndexOf(s, substr) >= 0)
}

func stringIndexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
