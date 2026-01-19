// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package database

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"github.com/tomtom215/cartographus/internal/config"
	"github.com/tomtom215/cartographus/internal/logging"
)

// CachedTile represents a cached vector tile with versioning and TTL
type CachedTile struct {
	Data    []byte
	Version int64
	Expires time.Time
}

// DB wraps the DuckDB connection and provides data access methods
type DB struct {
	conn                  *sql.DB
	cfg                   *config.DatabaseConfig
	spatialAvailable      bool // Tracks whether spatial extension is loaded
	inetAvailable         bool // Tracks whether inet extension is loaded
	icuAvailable          bool // Tracks whether icu extension is loaded
	jsonAvailable         bool // Tracks whether json extension is loaded
	sqliteAvailable       bool // Tracks whether sqlite extension is loaded (for Tautulli import)
	rapidfuzzAvailable    bool // Tracks whether rapidfuzz extension is loaded (for fuzzy search)
	datasketchesAvailable bool // Tracks whether datasketches extension is loaded (for approximate analytics)

	// Prepared statement caching
	stmtCache   map[string]*sql.Stmt
	stmtCacheMu sync.RWMutex

	// Vector tile caching
	tileCache     map[string]CachedTile
	tileCacheMu   sync.RWMutex
	dataVersion   int64
	dataVersionMu sync.RWMutex
	tileCacheTTL  time.Duration

	// Per-row write locks for concurrent UPSERTs
	ipLocks sync.Map

	// Connection recovery fields
	serverLat         float64
	serverLon         float64
	maxReconnectTries int
	reconnectDelay    time.Duration
}

// New creates a new database connection and initializes the schema
func New(cfg *config.DatabaseConfig, serverLat, serverLon float64) (*DB, error) {
	numThreads := cfg.Threads
	if numThreads <= 0 {
		numThreads = runtime.NumCPU()
	}

	// Ensure parent directory exists for database file
	// This prevents "No such file or directory" errors when the data directory doesn't exist
	// Use 0750 permissions (owner: rwx, group: rx, other: none) per gosec G301
	dbDir := filepath.Dir(cfg.Path)
	if dbDir != "" && dbDir != "." {
		if err := os.MkdirAll(dbDir, 0o750); err != nil {
			return nil, fmt.Errorf("failed to create database directory %s: %w", dbDir, err)
		}
	}

	// CRITICAL: Preload extensions BEFORE opening the main database.
	// When DuckDB opens a database file, it immediately replays the WAL (Write-Ahead Log).
	// If the WAL contains ALTER TABLE statements that use extension functions (e.g.,
	// TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP from the ICU extension), WAL replay will fail
	// with "GetDefaultDatabase with no default database set" if extensions aren't loaded.
	//
	// By loading extensions in an in-memory database first, DuckDB caches them per-process,
	// making them available when we open the main database file for WAL replay.
	if err := preloadExtensions(); err != nil {
		logging.Warn().Err(err).Msg("Failed to preload extensions, WAL replay may fail if database has pending changes")
	}

	// Build connection string with tuning options
	// preserve_insertion_order=false reduces memory usage but may change result order
	preserveOrder := "true"
	if !cfg.PreserveInsertionOrder {
		preserveOrder = "false"
	}

	// Disable auto-install/auto-load to prevent hangs in restricted network environments
	// Extensions are explicitly loaded by installExtensions() with proper timeout handling
	connStr := fmt.Sprintf("%s?access_mode=read_write&threads=%d&max_memory=%s&preserve_insertion_order=%s&autoinstall_known_extensions=false&autoload_known_extensions=false",
		cfg.Path, numThreads, cfg.MaxMemory, preserveOrder)

	conn, err := sql.Open("duckdb", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db := &DB{
		conn:                  conn,
		cfg:                   cfg,
		spatialAvailable:      true,
		inetAvailable:         true,
		icuAvailable:          true,
		jsonAvailable:         true,
		sqliteAvailable:       true,
		rapidfuzzAvailable:    true,
		datasketchesAvailable: true,
		stmtCache:             make(map[string]*sql.Stmt),
		tileCache:             make(map[string]CachedTile),
		dataVersion:           0,
		tileCacheTTL:          5 * time.Minute,
		serverLat:             serverLat,
		serverLon:             serverLon,
		maxReconnectTries:     3,
		reconnectDelay:        2 * time.Second,
	}

	if err := db.configureConnectionPool(); err != nil {
		closeQuietly(conn)
		return nil, fmt.Errorf("failed to configure connection pool: %w", err)
	}

	if err := db.initialize(); err != nil {
		closeQuietly(conn)
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	if err := db.enableProfiling(); err != nil {
		logging.Warn().Err(err).Msg("Query profiling not enabled")
	}

	if err := db.initializeSpatialOptimizations(serverLat, serverLon); err != nil {
		logging.Warn().Err(err).Msg("Spatial optimizations initialization had issues")
	}

	return db, nil
}

// IsSpatialAvailable returns whether the spatial extension is available
func (db *DB) IsSpatialAvailable() bool {
	return db.spatialAvailable
}

// IsInetAvailable returns whether the inet extension is available
func (db *DB) IsInetAvailable() bool {
	return db.inetAvailable
}

// IsIcuAvailable returns whether the icu extension is available
func (db *DB) IsIcuAvailable() bool {
	return db.icuAvailable
}

// IsJSONAvailable returns whether the json extension is available
func (db *DB) IsJSONAvailable() bool {
	return db.jsonAvailable
}

// IsSQLiteAvailable returns whether the sqlite extension is available (for Tautulli import)
func (db *DB) IsSQLiteAvailable() bool {
	return db.sqliteAvailable
}

// IsRapidFuzzAvailable returns whether the rapidfuzz extension is available (for fuzzy search)
func (db *DB) IsRapidFuzzAvailable() bool {
	return db.rapidfuzzAvailable
}

// IsDataSketchesAvailable returns whether the datasketches extension is available (for approximate analytics)
func (db *DB) IsDataSketchesAvailable() bool {
	return db.datasketchesAvailable
}

// SetSpatialAvailableForTesting sets the spatial extension availability flag.
// This method is intended for testing purposes only to allow unit tests to
// mock the spatial extension availability without requiring actual DuckDB extensions.
func (db *DB) SetSpatialAvailableForTesting(available bool) {
	db.spatialAvailable = available
}

// Conn returns the underlying SQL database connection.
// This is used by packages that need direct database access, such as the
// detection package for storing alerts and trust scores.
func (db *DB) Conn() *sql.DB {
	return db.conn
}

// preloadExtensions loads DuckDB extensions in an in-memory database before opening
// the main database file. This ensures extensions are available during WAL replay.
//
// DuckDB caches loaded extensions per-process, so once loaded in any database
// connection (even in-memory), they become available for all subsequent connections.
// This prevents "GetDefaultDatabase with no default database set" errors during
// WAL replay when the WAL contains ALTER TABLE statements with extension functions.
//
// This function is skipped in CI/test environments where extensions may not be
// installed and tests use DUCKDB_SPATIAL_OPTIONAL=true anyway.
func preloadExtensions() error {
	// Skip in CI environments - tests use DUCKDB_SPATIAL_OPTIONAL=true and
	// don't need extension preloading. This also prevents potential resource
	// contention issues when running many parallel tests.
	if os.Getenv("CI") != "" || os.Getenv("GITHUB_ACTIONS") != "" {
		logging.Debug().Msg("Skipping extension preload in CI environment")
		return nil
	}

	logging.Debug().Msg("Preloading DuckDB extensions for WAL replay compatibility")

	// Open an in-memory database with autoload disabled (we'll load explicitly)
	conn, err := sql.Open("duckdb", ":memory:?autoinstall_known_extensions=false&autoload_known_extensions=false")
	if err != nil {
		return fmt.Errorf("failed to open in-memory database for extension preload: %w", err)
	}

	// Ensure proper cleanup: disable connection pooling before close
	// This prevents resource leaks that could affect the main database connection
	defer func() {
		conn.SetConnMaxLifetime(0)
		conn.SetMaxIdleConns(0)
		conn.SetMaxOpenConns(0)
		closeQuietly(conn)
	}()

	// List of core extensions that might be used in table defaults
	// ICU is critical - it provides TIMESTAMPTZ and timezone functions
	extensions := []string{"icu", "json", "inet", "spatial"}

	for _, ext := range extensions {
		// Check if extension is installed locally
		if !isExtensionInstalledLocally(ext) {
			logging.Debug().Str("extension", ext).Msg("Extension not installed locally, skipping preload")
			continue
		}

		// Load the extension
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		_, err := conn.ExecContext(ctx, fmt.Sprintf("LOAD %s;", ext))
		cancel()

		if err != nil {
			logging.Debug().Str("extension", ext).Err(err).Msg("Failed to preload extension")
			// Continue with other extensions - non-fatal
		} else {
			logging.Debug().Str("extension", ext).Msg("Extension preloaded successfully")
		}
	}

	return nil
}

// Close closes the database connection and all prepared statements.
// It performs a CHECKPOINT before closing to flush the WAL to the main database file.
// This prevents WAL replay issues on next startup caused by a DuckDB bug where
// replaying CREATE TABLE statements with TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
// can fail with "GetDefaultDatabase with no default database set" errors.
func (db *DB) Close() error {
	db.stmtCacheMu.Lock()
	for _, stmt := range db.stmtCache {
		if stmt != nil {
			closeWithLog(stmt, nil, "prepared statement")
		}
	}
	db.stmtCache = make(map[string]*sql.Stmt)
	db.stmtCacheMu.Unlock()

	if db.conn != nil {
		// Force a checkpoint to flush WAL before closing.
		// This prevents WAL replay issues on next startup.
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		if err := db.Checkpoint(ctx); err != nil {
			// Log warning but don't fail - best effort checkpoint
			logging.Warn().Err(err).Msg("Failed to checkpoint database before close")
		}
		cancel()

		return db.conn.Close()
	}
	return nil
}

// Ping checks if the database connection is alive
func (db *DB) Ping(ctx context.Context) error {
	if db.conn == nil {
		return fmt.Errorf("database connection is nil")
	}
	return db.conn.PingContext(ctx)
}

// initialize creates tables and installs required extensions
func (db *DB) initialize() error {
	// Install all extensions
	if err := db.installExtensions(); err != nil {
		return err
	}

	// Create tables
	if err := db.createTables(); err != nil {
		return err
	}

	// Run versioned migrations (CRITICAL-006 fix: tracks applied migrations)
	if err := db.runVersionedMigrations(); err != nil {
		return err
	}

	// Initialize Phase 3 cross-platform schema (content mapping, user linking)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := db.InitCrossPlatformSchema(ctx); err != nil {
		return fmt.Errorf("failed to initialize cross-platform schema: %w", err)
	}

	// Create indexes
	if err := db.createIndexes(); err != nil {
		return err
	}

	// Force a checkpoint after schema initialization to flush the WAL.
	// This prevents a DuckDB bug where WAL replay of CREATE TABLE statements
	// with TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP fails with
	// "GetDefaultDatabase with no default database set" errors.
	// By checkpointing here, we ensure the WAL is flushed before normal operations.
	checkpointCtx, checkpointCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer checkpointCancel()
	if err := db.Checkpoint(checkpointCtx); err != nil {
		// Log warning but don't fail initialization - the issue only affects restart
		logging.Warn().Err(err).Msg("Failed to checkpoint after schema initialization")
	}

	return nil
}
