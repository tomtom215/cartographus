// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package database provides versioned schema migration support.
//
// This file implements a production-ready migration system that:
// - Tracks applied migrations in schema_migrations table
// - Ensures migrations run exactly once
// - Provides migration version information for debugging
// - Supports both initial schema creation and incremental changes
//
// SCHEMA CONSOLIDATION (Pre-Release):
// As of 2026-01-04, all 187 column migrations have been consolidated into
// the initial CREATE TABLE statement in database_schema.go. This is appropriate
// because:
//   - The app has never been publicly released
//   - No existing databases need migration
//   - Single source of truth is cleaner and faster
//
// POST-RELEASE MIGRATION STRATEGY:
// After the first public release with real users, add new migrations here
// starting from version 1. The migration infrastructure is preserved and ready.
// See CLAUDE.md "Schema Consolidation" section for full details.
package database

import (
	"context"
	"fmt"
	"os"
	"time"
)

// Migration represents a versioned database migration.
type Migration struct {
	Version     int       // Unique version number (monotonically increasing)
	Name        string    // Human-readable migration name
	Description string    // Description of what this migration does
	SQL         string    // SQL statement to execute
	AppliedAt   time.Time // When the migration was applied (populated on query)
}

// schemaMigrationsTable creates the migration tracking table
const schemaMigrationsTable = `
CREATE TABLE IF NOT EXISTS schema_migrations (
	version INTEGER PRIMARY KEY,
	name TEXT NOT NULL,
	description TEXT,
	applied_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
`

// getMigrations returns all versioned migrations in order.
//
// PRE-RELEASE: This returns an empty slice because all columns are now
// defined in the initial CREATE TABLE statement in database_schema.go.
//
// POST-RELEASE: After the first public release, add new migrations here
// starting from version 1. Example:
//
//	{Version: 1, Name: "add_new_column", Description: "Add new_column for feature X",
//	 SQL: `ALTER TABLE playback_events ADD COLUMN IF NOT EXISTS new_column TEXT;`},
//
// Migrations MUST be append-only - never modify or remove existing migrations
// once users have databases with data.
func (db *DB) getMigrations() []Migration {
	// All 187 original column migrations have been consolidated into
	// database_schema.go as of 2026-01-04. This is appropriate because:
	//   - The app has never been publicly released
	//   - No existing databases need migration
	//   - Single source of truth is cleaner and faster
	//
	// The migration infrastructure is preserved for post-release schema changes.
	// When adding new migrations after release, start from version 1.
	return []Migration{
		// Post-release migrations will be added here.
		// Example:
		// {Version: 1, Name: "add_new_feature_column", Description: "Add column for new feature",
		//  SQL: `ALTER TABLE playback_events ADD COLUMN IF NOT EXISTS new_feature_column TEXT;`},
	}
}

// createMigrationsTable creates the schema_migrations table if it doesn't exist
func (db *DB) createMigrationsTable(ctx context.Context) error {
	_, err := db.conn.ExecContext(ctx, schemaMigrationsTable)
	return err
}

// getAppliedMigrations returns a map of version -> Migration for all applied migrations
func (db *DB) getAppliedMigrations(ctx context.Context) (map[int]Migration, error) {
	rows, err := db.conn.QueryContext(ctx, `SELECT version, name, description, applied_at FROM schema_migrations ORDER BY version`)
	if err != nil {
		return nil, fmt.Errorf("failed to query applied migrations: %w", err)
	}
	defer rows.Close()

	applied := make(map[int]Migration)
	for rows.Next() {
		var m Migration
		if err := rows.Scan(&m.Version, &m.Name, &m.Description, &m.AppliedAt); err != nil {
			return nil, fmt.Errorf("failed to scan migration row: %w", err)
		}
		applied[m.Version] = m
	}
	return applied, rows.Err()
}

// runVersionedMigrations executes only new migrations that haven't been applied yet.
// This replaces the old runMigrations() function with a proper versioned approach.
func (db *DB) runVersionedMigrations() error {
	ctx, cancel := schemaContext()
	defer cancel()

	// Ensure migrations table exists
	if err := db.createMigrationsTable(ctx); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Get already applied migrations
	applied, err := db.getAppliedMigrations(ctx)
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	// Get all migrations
	migrations := db.getMigrations()

	// Apply new migrations in order
	newMigrations := 0
	for _, m := range migrations {
		if _, exists := applied[m.Version]; exists {
			continue // Already applied
		}

		// Execute migration
		if _, err := db.conn.ExecContext(ctx, m.SQL); err != nil {
			return fmt.Errorf("failed to execute migration v%d (%s): %w", m.Version, m.Name, err)
		}

		// Record migration as applied
		_, err := db.conn.ExecContext(ctx,
			`INSERT INTO schema_migrations (version, name, description) VALUES (?, ?, ?)`,
			m.Version, m.Name, m.Description)
		if err != nil {
			return fmt.Errorf("failed to record migration v%d: %w", m.Version, err)
		}

		newMigrations++
	}

	if newMigrations > 0 {
		// Log migration count using the logging package
		// Note: This is called during initialization, so logging should be available
		// Suppress output during benchmarks to avoid polluting benchmark output
		if os.Getenv("BENCHMARK_MODE") != "1" {
			fmt.Printf("Applied %d new database migrations\n", newMigrations)
		}
	}

	return nil
}

// GetCurrentSchemaVersion returns the highest applied migration version
func (db *DB) GetCurrentSchemaVersion() (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var version int
	err := db.conn.QueryRowContext(ctx, `SELECT COALESCE(MAX(version), 0) FROM schema_migrations`).Scan(&version)
	if err != nil {
		return 0, fmt.Errorf("failed to get schema version: %w", err)
	}
	return version, nil
}

// GetMigrationHistory returns all applied migrations in order
func (db *DB) GetMigrationHistory() ([]Migration, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	rows, err := db.conn.QueryContext(ctx,
		`SELECT version, name, description, applied_at FROM schema_migrations ORDER BY version`)
	if err != nil {
		return nil, fmt.Errorf("failed to query migration history: %w", err)
	}
	defer rows.Close()

	var history []Migration
	for rows.Next() {
		var m Migration
		if err := rows.Scan(&m.Version, &m.Name, &m.Description, &m.AppliedAt); err != nil {
			return nil, fmt.Errorf("failed to scan migration: %w", err)
		}
		history = append(history, m)
	}
	return history, rows.Err()
}
