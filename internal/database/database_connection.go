// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

/*
database_connection.go - Connection Management and Recovery

This file provides connection pool configuration and automatic reconnection
capabilities for resilient database operations.

Connection Recovery:
The reconnect() function implements exponential backoff for connection recovery:
  - Detects connection errors (connection refused, broken pipe, bad connection)
  - Closes existing connection and clears prepared statement cache
  - Attempts reconnection with configurable max retries and delay
  - Re-initializes database schema and extensions after successful reconnect

Connection Pool Configuration:
  - MaxOpenConns: Based on CPU count for parallelism
  - MaxIdleConns: 2 for efficient connection reuse
  - ConnMaxLifetime: 1 hour to prevent stale connections
  - ConnMaxIdleTime: 5 minutes for idle connection cleanup

Error Detection:
The package identifies connection errors vs query errors to determine
when automatic reconnection should be attempted. Only true connection
failures trigger the reconnection logic.
*/

//nolint:staticcheck // File documentation, not package doc
package database

import (
	"context"
	"database/sql"
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/tomtom215/cartographus/internal/logging"
)

// reconnect attempts to re-establish database connection with exponential backoff
//
//nolint:unused // Infrastructure function for connection recovery
func (db *DB) reconnect() error {
	db.reconnectMu.Lock()
	defer db.reconnectMu.Unlock()

	// Check if connection is actually dead before reconnecting
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.Ping(ctx); err == nil {
		return nil // Connection is alive
	}

	// Close existing connection and prepared statements
	db.clearStatementCache()

	if db.conn != nil {
		closeWithLog(db.conn, nil, "database connection")
	}

	// Attempt reconnection with exponential backoff
	var lastErr error
	for attempt := 0; attempt < db.maxReconnectTries; attempt++ {
		if attempt > 0 {
			delay := db.reconnectDelay * time.Duration(1<<uint(attempt-1))
			select {
			case <-time.After(delay):
				// Continue with retry
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		if err := db.attemptReconnect(); err != nil {
			lastErr = fmt.Errorf("reconnect attempt %d failed: %w", attempt+1, err)
			continue
		}

		return nil
	}

	return fmt.Errorf("failed to reconnect after %d attempts: %w", db.maxReconnectTries, lastErr)
}

// attemptReconnect tries to establish a new database connection
//
//nolint:unused // Called by reconnect() for connection recovery
func (db *DB) attemptReconnect() error {
	numThreads := db.cfg.Threads
	if numThreads <= 0 {
		numThreads = runtime.NumCPU()
	}
	preserveOrder := "true"
	if !db.cfg.PreserveInsertionOrder {
		preserveOrder = "false"
	}
	connStr := fmt.Sprintf("%s?access_mode=read_write&threads=%d&max_memory=%s&preserve_insertion_order=%s",
		db.cfg.Path, numThreads, db.cfg.MaxMemory, preserveOrder)

	conn, err := sql.Open("duckdb", connStr)
	if err != nil {
		return fmt.Errorf("failed to open: %w", err)
	}

	// Verify connection
	pingCtx, pingCancel := context.WithTimeout(context.Background(), 5*time.Second)
	if err := conn.PingContext(pingCtx); err != nil {
		pingCancel()
		closeQuietly(conn)
		return fmt.Errorf("failed to ping: %w", err)
	}
	pingCancel()

	db.conn = conn

	if err := db.configureConnectionPool(); err != nil {
		closeQuietly(conn)
		return fmt.Errorf("failed to configure pool: %w", err)
	}

	if err := db.initialize(); err != nil {
		closeQuietly(conn)
		return fmt.Errorf("failed to initialize: %w", err)
	}

	// These are optional features - warnings are logged internally if they fail
	if err := db.enableProfiling(); err != nil {
		logging.Warn().Err(err).Msg("Query profiling not enabled")
	}
	if err := db.initializeSpatialOptimizations(db.serverLat, db.serverLon); err != nil {
		logging.Warn().Err(err).Msg("Spatial optimizations initialization had issues")
	}

	return nil
}

// clearStatementCache closes all cached prepared statements
//
//nolint:unused // Called by reconnect() for connection recovery
func (db *DB) clearStatementCache() {
	db.stmtCacheMu.Lock()
	for _, stmt := range db.stmtCache {
		if stmt != nil {
			closeWithLog(stmt, nil, "prepared statement")
		}
	}
	db.stmtCache = make(map[string]*sql.Stmt)
	db.stmtCacheMu.Unlock()
}

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
