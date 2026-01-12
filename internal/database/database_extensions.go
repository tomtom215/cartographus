// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

/*
database_extensions.go - DuckDB Extension Installation

This file handles the installation and loading of DuckDB extensions required for
full functionality of the Cartographus application.

Required Extensions (installed in every build):
  - httpfs: Enables HTTPS downloads for extension installation
  - spatial: Provides GEOMETRY types, ST_* functions, and R-tree spatial indexes
  - h3: Hexagonal hierarchical geospatial indexing for efficient aggregation
  - inet: Native IP address type with validation and network operations
  - icu: Timezone-aware timestamp operations and internationalized collations
  - json: JSON data processing and path-based extraction
  - sqlite_scanner: Direct SQLite database file access for Tautulli import
  - rapidfuzz: High-performance fuzzy string matching for search functionality
  - datasketches: Approximate analytics with HyperLogLog (distinct counts) and KLL (percentiles)

All extensions are pre-installed in Docker images and should be installed locally
using ./scripts/setup-duckdb-extensions.sh. The datasketches extension is installed
but disabled by default - enable it via DUCKDB_DATASKETCHES_ENABLED=true.

Installation Strategy:
Each extension follows a fallback installation pattern:
 1. Try INSTALL <extension>
 2. If install fails, try LOAD <extension> (may already be installed)
 3. If load fails, try FORCE INSTALL <extension>
 4. If optional=true and all fail, disable feature gracefully

Environment Variables:
  - DUCKDB_SPATIAL_OPTIONAL=true: Allow startup without spatial extensions (testing only)
  - DUCKDB_DATASKETCHES_ENABLED=true: Enable datasketches extension (disabled by default)
*/

//nolint:staticcheck // File documentation, not package doc
package database

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/tomtom215/cartographus/internal/logging"
)

// Note: Removed sync.Once caching for community extensions.
// CGO calls cannot be interrupted, so the only safe approach is to skip
// loading community extensions that aren't already locally installed.
// The new installXxxIfLocal() functions handle this deterministically.

// communityExtensionTimeout is the hard timeout for community extension operations
// CGO calls don't respect context cancellation, so we need goroutine-based timeouts
// Can be overridden via DUCKDB_EXTENSION_TIMEOUT environment variable (e.g., "30s", "1m")
var communityExtensionTimeout = getExtensionTimeout()

// extensionRetryConfig controls retry behavior for extension operations
type extensionRetryConfig struct {
	MaxRetries  int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
	BackoffMult float64
}

// defaultRetryConfig provides sensible defaults for extension loading retries
var defaultRetryConfig = extensionRetryConfig{
	MaxRetries:  3,
	BaseDelay:   2 * time.Second,
	MaxDelay:    30 * time.Second,
	BackoffMult: 2.0,
}

// getExtensionTimeout returns the timeout for extension operations
// Configurable via DUCKDB_EXTENSION_TIMEOUT environment variable
func getExtensionTimeout() time.Duration {
	if timeoutStr := os.Getenv("DUCKDB_EXTENSION_TIMEOUT"); timeoutStr != "" {
		if d, err := time.ParseDuration(timeoutStr); err == nil && d > 0 {
			return d
		}
	}
	return 30 * time.Second // Increased from 15s to 30s for better reliability
}

// duckdbVersion is the DuckDB version used for extension paths
// Single source of truth is scripts/duckdb-version.sh - keep in sync when updating
// This must also match the duckdb-go-bindings version in go.mod
const duckdbVersion = "v1.4.3"

// isExtensionInstalledLocally checks if an extension file exists in the local DuckDB
// extension directory. This is used to skip network INSTALL commands when extensions
// are pre-installed (e.g., by setup-duckdb-extensions.sh in CI).
func isExtensionInstalledLocally(extensionName string) bool {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return false
	}

	// DuckDB extension path: ~/.duckdb/extensions/{version}/{platform}/{name}.duckdb_extension
	platform := runtime.GOOS + "_" + runtime.GOARCH
	extPath := filepath.Join(homeDir, ".duckdb", "extensions", duckdbVersion, platform, extensionName+".duckdb_extension")

	_, err = os.Stat(extPath)
	return err == nil
}

// execResult holds the result of an async exec operation
type execResult struct {
	err error
}

// queryResult holds the result of an async query operation
type queryResult struct {
	value interface{}
	err   error
}

// execWithHardTimeout executes a SQL statement with a goroutine-based hard timeout
// This is necessary because DuckDB CGO calls don't respect context cancellation.
// We still use ExecContext for proper resource cleanup, but enforce timeout via select.
func (db *DB) execWithHardTimeout(query string) error {
	resultCh := make(chan execResult, 1)

	// Create context with same timeout - CGO may ignore it, but it helps with cleanup
	ctx, cancel := extensionContext()
	defer cancel()

	go func() {
		_, err := db.conn.ExecContext(ctx, query)
		resultCh <- execResult{err: err}
	}()

	select {
	case result := <-resultCh:
		return result.err
	case <-time.After(communityExtensionTimeout):
		return fmt.Errorf("operation timed out after %v", communityExtensionTimeout)
	}
}

// queryRowWithHardTimeout executes a query and scans a single value with a hard timeout
// This is necessary because DuckDB CGO calls don't respect context cancellation.
func (db *DB) queryRowWithHardTimeout(query string) (interface{}, error) {
	resultCh := make(chan queryResult, 1)

	// Create context with same timeout - CGO may ignore it, but it helps with cleanup
	ctx, cancel := extensionContext()
	defer cancel()

	go func() {
		var result interface{}
		err := db.conn.QueryRowContext(ctx, query).Scan(&result)
		resultCh <- queryResult{value: result, err: err}
	}()

	select {
	case result := <-resultCh:
		return result.value, result.err
	case <-time.After(communityExtensionTimeout):
		return nil, fmt.Errorf("query timed out after %v", communityExtensionTimeout)
	}
}

// execWithRetry executes a SQL statement with retry logic and exponential backoff
// This handles transient network failures when downloading extensions
func (db *DB) execWithRetry(query string, config extensionRetryConfig) error {
	var lastErr error
	delay := config.BaseDelay

	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		if attempt > 0 {
			logging.Debug().
				Int("attempt", attempt).
				Dur("delay", delay).
				Str("query", query).
				Msg("Retrying extension operation")
			time.Sleep(delay)
			// Exponential backoff with cap
			delay = time.Duration(float64(delay) * config.BackoffMult)
			if delay > config.MaxDelay {
				delay = config.MaxDelay
			}
		}

		err := db.execWithHardTimeout(query)
		if err == nil {
			return nil
		}
		lastErr = err

		// Check if error is retryable (timeout or transient network error)
		errStr := err.Error()
		isRetryable := strings.Contains(errStr, "timed out") ||
			strings.Contains(errStr, "timeout") ||
			strings.Contains(errStr, "connection refused") ||
			strings.Contains(errStr, "503") ||
			strings.Contains(errStr, "temporary failure")

		if !isRetryable {
			// Non-retryable error, fail immediately
			return err
		}

		logging.Warn().
			Err(err).
			Int("attempt", attempt+1).
			Int("max_attempts", config.MaxRetries+1).
			Msg("Extension operation failed, will retry")
	}

	return fmt.Errorf("extension operation failed after %d attempts: %w", config.MaxRetries+1, lastErr)
}

// extensionInstaller is a function type for installing an extension
type extensionInstaller func(optional bool) error

// installExtension installs an extension and returns error only if not optional
func installExtension(installer extensionInstaller, optional bool) error {
	if err := installer(optional); err != nil && !optional {
		return err
	}
	return nil
}

// installExtensions installs and loads all required DuckDB extensions
// Returns error if required extensions fail to load (unless DUCKDB_SPATIAL_OPTIONAL=true)
//
// Extension behavior:
//   - All extensions are pre-installed in Docker images and via setup-duckdb-extensions.sh
//   - Core extensions (spatial, h3, inet, icu, json) are always required
//   - sqlite_scanner and rapidfuzz are required for full functionality
//   - datasketches is installed but disabled by default (enable via DUCKDB_DATASKETCHES_ENABLED=true)
func (db *DB) installExtensions() error {
	spatialOptional := os.Getenv("DUCKDB_SPATIAL_OPTIONAL") == "true"
	datasketchesEnabled := os.Getenv("DUCKDB_DATASKETCHES_ENABLED") == "true"
	isCI := os.Getenv("CI") != "" || os.Getenv("GITHUB_ACTIONS") != ""

	// Configure HTTPS for extension repository
	if err := db.configureExtensionRepository(); err != nil {
		logging.Warn().Err(err).Msg("Failed to set custom extension repository, will use default")
	}

	// Install httpfs first (dependency for other extensions)
	if err := db.installHttpfs(); err != nil {
		logging.Warn().Err(err).Msg("Failed to install/load httpfs extension, spatial extension may fail")
	}

	// Install core extensions (required unless spatialOptional=true)
	// Note: Community extensions (sqlite_scanner, rapidfuzz, datasketches) are handled
	// separately below with CI-specific logic to prevent CGO hangs.
	coreExtensions := []extensionInstaller{
		db.installSpatial,
		db.installH3,
		db.installInet,
		db.installICU,
		db.installJSON,
	}
	for _, installer := range coreExtensions {
		if err := installExtension(installer, spatialOptional); err != nil {
			return err
		}
	}

	// In CI environments, skip ALL community extensions that use CGO LOAD calls.
	// CGO calls cannot be interrupted by Go context cancellation or timeouts -
	// once a CGO call starts, it blocks until completion or process termination.
	// This is a fundamental limitation that cannot be worked around with goroutines.
	//
	// Extensions skipped in CI:
	// - sqlite_scanner: Only needed for Tautulli import tests (run separately)
	// - rapidfuzz: Only needed for fuzzy search (exact matching fallback)
	// - datasketches: Only needed for approximate analytics (exact calculation fallback)
	if isCI {
		db.sqliteAvailable = false
		db.rapidfuzzAvailable = false
		db.datasketchesAvailable = false
		return nil
	}

	// Install sqlite_scanner extension (for Tautulli database import)
	// Required - all extensions are pre-installed via setup script or Docker image
	if err := db.installSQLiteIfLocal(spatialOptional); err != nil {
		return err
	}

	// Install rapidfuzz extension (for fuzzy search)
	// Required - all extensions are pre-installed via setup script or Docker image
	if err := db.installRapidFuzzIfLocal(spatialOptional); err != nil {
		return err
	}

	// Install datasketches extension (for approximate analytics)
	// Installed but disabled by default - enable via DUCKDB_DATASKETCHES_ENABLED=true
	// This allows users to enable approximate analytics as a config change without rebuilding
	if datasketchesEnabled {
		logging.Info().Msg("DataSketches extension enabled via DUCKDB_DATASKETCHES_ENABLED")
		if err := db.installDataSketchesIfLocal(spatialOptional); err != nil {
			return err
		}
	} else {
		// Mark as unavailable without attempting to load
		// The extension is installed but not loaded - available via config change
		db.datasketchesAvailable = false
		logging.Info().Msg("DataSketches extension installed but disabled (enable via DUCKDB_DATASKETCHES_ENABLED=true)")
	}

	return nil
}

// configureExtensionRepository sets HTTPS for extension downloads
// Uses execWithHardTimeout because CGO calls don't respect context cancellation
func (db *DB) configureExtensionRepository() error {
	return db.execWithHardTimeout("SET custom_extension_repository = 'https://extensions.duckdb.org';")
}

// installHttpfs installs the httpfs extension for HTTPS downloads
// Uses retry logic because this is critical for downloading other extensions
func (db *DB) installHttpfs() error {
	// Check if already installed locally
	if isExtensionInstalledLocally("httpfs") {
		logging.Debug().Msg("httpfs extension found locally")
	}

	if err := db.execWithRetry("INSTALL httpfs;", defaultRetryConfig); err != nil {
		if loadErr := db.execWithHardTimeout("LOAD httpfs;"); loadErr != nil {
			return fmt.Errorf("httpfs install error: %w, load error: %w. "+
				"Pre-install extensions using: ./scripts/setup-duckdb-extensions.sh", err, loadErr)
		}
		return nil
	}
	return db.execWithHardTimeout("LOAD httpfs;")
}

// installSpatial installs the spatial extension
func (db *DB) installSpatial(optional bool) error {
	spec := &extensionSpec{
		Name:              "spatial",
		AvailabilityField: func(db *DB) *bool { return &db.spatialAvailable },
		WarningMessage:    "Spatial extension unavailable (DUCKDB_SPATIAL_OPTIONAL=true), creating tables without GEOMETRY columns",
	}
	return db.installCoreExtension(spec, optional)
}

// installH3 installs the H3 community extension for hexagonal indexing
func (db *DB) installH3(optional bool) error {
	spec := &extensionSpec{
		Name:             "h3",
		Community:        true,
		DependsOnSpatial: true,
		VerifyQuery:      "SELECT h3_latlng_to_cell(0.0, 0.0, 0)",
		WarningMessage:   "H3 extension unavailable, H3 indexing will be disabled",
	}
	return db.installCommunityExtension(spec, optional)
}

// installInet installs the INET extension for IP address operations
func (db *DB) installInet(optional bool) error {
	spec := &extensionSpec{
		Name:              "inet",
		VerifyQuery:       "SELECT host('192.168.1.1'::INET)",
		AvailabilityField: func(db *DB) *bool { return &db.inetAvailable },
		WarningMessage:    "INET extension unavailable (DUCKDB_SPATIAL_OPTIONAL=true), IP addresses will use TEXT type",
	}
	return db.installCoreExtension(spec, optional)
}

// installICU installs the ICU extension for timezone support
func (db *DB) installICU(optional bool) error {
	spec := &extensionSpec{
		Name:              "icu",
		VerifyQuery:       "SELECT timezone('America/New_York', TIMESTAMP '2024-01-01 12:00:00')::VARCHAR",
		AvailabilityField: func(db *DB) *bool { return &db.icuAvailable },
		WarningMessage:    "ICU extension unavailable (DUCKDB_SPATIAL_OPTIONAL=true), timezone operations will be limited",
	}
	return db.installCoreExtension(spec, optional)
}

// installJSON installs the JSON extension for JSON operations
func (db *DB) installJSON(optional bool) error {
	spec := &extensionSpec{
		Name:              "json",
		VerifyQuery:       "SELECT json_extract('{\"name\":\"test\"}', '$.name')::VARCHAR",
		AvailabilityField: func(db *DB) *bool { return &db.jsonAvailable },
		WarningMessage:    "JSON extension unavailable (DUCKDB_SPATIAL_OPTIONAL=true), JSON operations will be limited",
	}
	return db.installCoreExtension(spec, optional)
}

// installSQLite installs the sqlite_scanner extension for reading Tautulli database files
func (db *DB) installSQLite(optional bool) error {
	ctx, cancel := extensionContext()
	defer cancel()

	spec := &extensionSpec{
		Name:              "sqlite_scanner",
		AvailabilityField: func(db *DB) *bool { return &db.sqliteAvailable },
		WarningMessage:    "sqlite_scanner extension unavailable (DUCKDB_SPATIAL_OPTIONAL=true), Tautulli import will be disabled",
	}

	// Use standard core installation
	if err := db.installCoreExtension(spec, optional); err != nil {
		return err
	}

	// Additional verification: check sqlite_attach function exists
	if db.sqliteAvailable {
		var functionExists int
		if err := db.conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM duckdb_functions() WHERE function_name = 'sqlite_attach'").Scan(&functionExists); err != nil || functionExists == 0 {
			if optional {
				db.sqliteAvailable = false
				logging.Warn().Msg("SQLite attach function unavailable, Tautulli import will be disabled")
				return nil
			}
			return fmt.Errorf("sqlite_scanner extension loaded but sqlite_attach function unavailable")
		}
	}

	return nil
}

// installRapidFuzz installs the RapidFuzz community extension for fuzzy string matching
// This enables fuzzy search, data deduplication, and autocomplete functionality
func (db *DB) installRapidFuzz(optional bool) error {
	spec := &extensionSpec{
		Name:              "rapidfuzz",
		Community:         true,
		VerifyQuery:       "SELECT rapidfuzz_ratio('hello', 'helo')",
		AvailabilityField: func(db *DB) *bool { return &db.rapidfuzzAvailable },
		WarningMessage:    "RapidFuzz extension unavailable, fuzzy search will use exact matching",
	}
	return db.installCommunityExtension(spec, optional)
}

// installDataSketches installs the DataSketches community extension for approximate analytics
// This enables HyperLogLog (approximate distinct counts) and KLL (approximate percentiles)
func (db *DB) installDataSketches(optional bool) error {
	ctx, cancel := extensionContext()
	defer cancel()

	spec := &extensionSpec{
		Name:              "datasketches",
		Community:         true,
		AvailabilityField: func(db *DB) *bool { return &db.datasketchesAvailable },
		WarningMessage:    "DataSketches extension unavailable, approximate analytics will use exact calculations",
	}

	// Use community installation
	if err := db.installCommunityExtension(spec, optional); err != nil {
		return err
	}

	// Additional verification: check HLL function exists
	if db.datasketchesAvailable {
		var functionExists int
		if err := db.conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM duckdb_functions() WHERE function_name = 'datasketch_hll'").Scan(&functionExists); err != nil || functionExists == 0 {
			if optional {
				db.datasketchesAvailable = false
				logging.Warn().Msg("DataSketches HLL functions unavailable, approximate analytics will use exact calculations")
				return nil
			}
			return fmt.Errorf("datasketches extension loaded but HLL functions unavailable")
		}
	}

	return nil
}

// installSQLiteIfLocal installs sqlite_scanner ONLY if it's already locally installed.
// This prevents CGO hangs from network downloads. If not local, marks as unavailable.
func (db *DB) installSQLiteIfLocal(optional bool) error {
	if !isExtensionInstalledLocally("sqlite_scanner") {
		db.sqliteAvailable = false
		logging.Info().Msg("sqlite_scanner extension not found locally, Tautulli import will be disabled")
		return nil
	}
	return db.installSQLite(optional)
}

// installRapidFuzzIfLocal installs rapidfuzz ONLY if it's already locally installed.
// This prevents CGO hangs from network downloads. If not local, marks as unavailable.
func (db *DB) installRapidFuzzIfLocal(optional bool) error {
	if !isExtensionInstalledLocally("rapidfuzz") {
		db.rapidfuzzAvailable = false
		logging.Info().Msg("rapidfuzz extension not found locally, fuzzy search will use exact matching")
		return nil
	}
	return db.installRapidFuzz(optional)
}

// installDataSketchesIfLocal installs datasketches ONLY if it's already locally installed.
// This prevents CGO hangs from network downloads. If not local, marks as unavailable.
func (db *DB) installDataSketchesIfLocal(optional bool) error {
	if !isExtensionInstalledLocally("datasketches") {
		db.datasketchesAvailable = false
		logging.Info().Msg("datasketches extension not found locally, approximate analytics will use exact calculations")
		return nil
	}
	return db.installDataSketches(optional)
}
