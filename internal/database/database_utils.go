// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

/*
database_utils.go - Database Utility Functions

This file provides utility functions for database operations including
profiling, context management, and backup support.

Profiling:
  - enableProfiling(): Enables DuckDB query profiling when ENABLE_QUERY_PROFILING=true
  - logQueryPlan(): Logs EXPLAIN ANALYZE output when LOG_QUERY_PLANS=true
  - Useful for debugging slow queries and understanding execution plans

Context Management:
  - ensureContext(): Creates a context with 30-second timeout if none provided
  - Ensures all database operations have a timeout to prevent hanging queries

Backup Support:
  - Checkpoint(): Forces a WAL checkpoint for consistent backup state
  - GetDatabasePath(): Returns the database file path for backup operations
  - GetRecordCounts(): Returns row counts for backup verification

Environment Variables:
  - ENABLE_QUERY_PROFILING=true: Enable DuckDB profiling
  - LOG_QUERY_PLANS=true: Log EXPLAIN ANALYZE output for queries
*/

//nolint:staticcheck // File documentation, not package doc
package database

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/tomtom215/cartographus/internal/logging"
)

// enableProfiling enables DuckDB query profiling for performance debugging
func (db *DB) enableProfiling() error {
	if os.Getenv("ENABLE_QUERY_PROFILING") != "true" {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if _, err := db.conn.ExecContext(ctx, "PRAGMA enable_profiling"); err != nil {
		return fmt.Errorf("failed to enable profiling: %w", err)
	}

	if _, err := db.conn.ExecContext(ctx, "PRAGMA profiling_mode = 'detailed'"); err != nil {
		return fmt.Errorf("failed to set profiling mode: %w", err)
	}

	logging.Info().Msg("Query profiling enabled (detailed mode)")
	return nil
}

// ensureContext creates a context with 30-second timeout if none provided
func (db *DB) ensureContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if ctx == nil {
		return context.WithTimeout(context.Background(), 30*time.Second)
	}

	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		return context.WithTimeout(ctx, 30*time.Second)
	}

	return ctx, func() {}
}


// Checkpoint forces a WAL checkpoint
func (db *DB) Checkpoint(ctx context.Context) error {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	_, err := db.conn.ExecContext(ctx, "CHECKPOINT")
	if err != nil {
		return fmt.Errorf("checkpoint failed: %w", err)
	}
	return nil
}

// GetDatabasePath returns the path to the database file
func (db *DB) GetDatabasePath() string {
	return db.cfg.Path
}

// GetRecordCounts returns the count of records in main tables
func (db *DB) GetRecordCounts(ctx context.Context) (playbacks int64, geolocations int64, err error) {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	err = db.conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM playback_events").Scan(&playbacks)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to count playbacks: %w", err)
	}

	err = db.conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM geolocations").Scan(&geolocations)
	if err != nil {
		return playbacks, 0, fmt.Errorf("failed to count geolocations: %w", err)
	}

	return playbacks, geolocations, nil
}
