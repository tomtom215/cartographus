// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

/*
database_extensions_core.go - Core Extension Installation Logic

This file provides the core infrastructure for installing DuckDB extensions
with a table-driven approach to reduce code duplication.
*/

package database

import (
	"context"
	"fmt"
	"time"

	"github.com/tomtom215/cartographus/internal/logging"
)

// extensionContext returns a context with timeout for extension operations
func extensionContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 30*time.Second)
}

// extensionSpec defines the specification for installing a DuckDB extension
type extensionSpec struct {
	// Name is the extension name (e.g., "spatial", "h3")
	Name string
	// Community indicates if this is a community extension (requires "FROM community")
	Community bool
	// VerifyQuery is an optional SQL query to verify the extension is working
	VerifyQuery string
	// VerifyResultHandler processes the verify query result (returns true if valid)
	VerifyResultHandler func(interface{}) bool
	// DependsOnSpatial indicates if this extension requires spatial to be available
	DependsOnSpatial bool
	// AvailabilityField is a pointer to the DB field tracking availability
	AvailabilityField func(*DB) *bool
	// WarningMessage is shown when extension is unavailable (optional mode only)
	WarningMessage string
}

// installCoreExtension installs a core extension using the standard pattern
// Uses retry logic for INSTALL commands to handle transient network failures
func (db *DB) installCoreExtension(spec *extensionSpec, optional bool) error {
	// Skip if depends on spatial and spatial is not available
	if spec.DependsOnSpatial && !db.spatialAvailable {
		return nil
	}

	// Check if extension is installed locally first
	if isExtensionInstalledLocally(spec.Name) {
		logging.Debug().Str("extension", spec.Name).Msg("Extension found locally, skipping download")
	}

	var installErr error

	// Step 1: Try INSTALL with retry for transient failures
	if err := db.execWithRetry(fmt.Sprintf("INSTALL %s;", spec.Name), defaultRetryConfig); err != nil {
		installErr = err
		// Step 2: Try LOAD (may already be installed)
		if loadErr := db.execWithHardTimeout(fmt.Sprintf("LOAD %s;", spec.Name)); loadErr != nil {
			// Step 3: Try FORCE INSTALL with retry
			if forceErr := db.execWithRetry(fmt.Sprintf("FORCE INSTALL %s;", spec.Name), defaultRetryConfig); forceErr != nil {
				if optional {
					db.setExtensionUnavailable(spec)
					return nil
				}
				return fmt.Errorf("failed to install %s extension after retries: install error: %w, load error: %w, force install error: %w. "+
					"Pre-install extensions using: ./scripts/setup-duckdb-extensions.sh",
					spec.Name, installErr, loadErr, forceErr)
			}
		} else {
			// LOAD succeeded - extension is already loaded, verify and return
			if spec.VerifyQuery != "" {
				ctx, cancel := extensionContext()
				defer cancel()
				return db.verifyExtension(ctx, spec, optional)
			}
			// No verify query - mark as available
			db.setExtensionAvailable(spec)
			return nil
		}
	}

	// Step 4: Load the extension (only reached if INSTALL or FORCE INSTALL succeeded)
	if err := db.execWithHardTimeout(fmt.Sprintf("LOAD %s;", spec.Name)); err != nil {
		if optional {
			db.setExtensionUnavailable(spec)
			logging.Warn().Str("extension", spec.Name).Err(err).Msg("Failed to load extension")
			return nil
		}
		return fmt.Errorf("failed to load %s extension: %w", spec.Name, err)
	}

	// Step 5: Verify if verification query provided
	if spec.VerifyQuery != "" {
		ctx, cancel := extensionContext()
		defer cancel()
		return db.verifyExtension(ctx, spec, optional)
	}

	// No verify query - mark as available
	db.setExtensionAvailable(spec)
	return nil
}

// setExtensionUnavailable marks an extension as unavailable and logs warning
func (db *DB) setExtensionUnavailable(spec *extensionSpec) {
	if field := spec.AvailabilityField; field != nil {
		*field(db) = false
	}
	if spec.WarningMessage != "" {
		logging.Warn().Str("extension", spec.Name).Msg(spec.WarningMessage)
	}
}

// setExtensionAvailable marks an extension as available
func (db *DB) setExtensionAvailable(spec *extensionSpec) {
	if field := spec.AvailabilityField; field != nil {
		*field(db) = true
	}
}

// verifyExtension verifies an extension is working by running a test query
// Uses queryRowWithHardTimeout because CGO calls don't respect context cancellation
func (db *DB) verifyExtension(_ context.Context, spec *extensionSpec, optional bool) error {
	result, err := db.queryRowWithHardTimeout(spec.VerifyQuery)
	if err != nil {
		if optional {
			db.setExtensionUnavailable(spec)
			logging.Warn().Str("extension", spec.Name).Err(err).Msg("Extension functions unavailable")
			return nil
		}
		return fmt.Errorf("%s extension loaded but functions unavailable: %w", spec.Name, err)
	}

	if spec.VerifyResultHandler != nil && !spec.VerifyResultHandler(result) {
		if optional {
			db.setExtensionUnavailable(spec)
			logging.Warn().Str("extension", spec.Name).Msg("Extension verification failed")
			return nil
		}
		return fmt.Errorf("%s extension verification failed", spec.Name)
	}

	// Extension loaded and verified successfully
	db.setExtensionAvailable(spec)
	return nil
}

// installCommunityExtension installs a community extension with timeout handling
func (db *DB) installCommunityExtension(spec *extensionSpec, optional bool) error {
	// Skip if depends on spatial and spatial is not available
	if spec.DependsOnSpatial && !db.spatialAvailable {
		return nil
	}

	ctx, cancel := extensionContext()
	defer cancel()

	// Check if extension is already loaded and working
	if db.isCommunityExtensionReady(ctx, spec) {
		db.setExtensionAvailable(spec)
		return nil
	}

	// Install extension if not locally available
	if err := db.installCommunityIfNeeded(spec, optional); err != nil {
		return err
	}

	// Load and verify the extension
	return db.loadAndVerifyCommunityExtension(ctx, spec, optional)
}

// isCommunityExtensionReady checks if a community extension is already loaded and working
// Uses queryRowWithHardTimeout because CGO calls don't respect context cancellation
func (db *DB) isCommunityExtensionReady(_ context.Context, spec *extensionSpec) bool {
	if !db.isExtensionLoaded(spec.Name) {
		return false
	}
	// Verify it works if there's a verify query
	if spec.VerifyQuery == "" {
		return true
	}
	_, err := db.queryRowWithHardTimeout(spec.VerifyQuery)
	return err == nil
}

// installCommunityIfNeeded installs a community extension if not locally available
// Uses retry logic with exponential backoff for transient network failures
func (db *DB) installCommunityIfNeeded(spec *extensionSpec, optional bool) error {
	if isExtensionInstalledLocally(spec.Name) {
		logging.Debug().Str("extension", spec.Name).Msg("Extension found locally, skipping download")
		return nil
	}

	logging.Info().Str("extension", spec.Name).Msg("Extension not found locally, downloading from repository")

	// Try to install the extension with retry logic for transient failures
	installCmd := fmt.Sprintf("INSTALL %s FROM community;", spec.Name)
	if err := db.execWithRetry(installCmd, defaultRetryConfig); err != nil {
		// Try force install as fallback
		logging.Debug().Str("extension", spec.Name).Msg("Standard install failed, trying force install")
		forceCmd := fmt.Sprintf("FORCE INSTALL %s FROM community;", spec.Name)
		if err := db.execWithRetry(forceCmd, defaultRetryConfig); err != nil {
			if optional {
				db.setExtensionUnavailable(spec)
				return nil
			}
			// Provide actionable error message for required extensions
			return fmt.Errorf("failed to install %s extension after retries: %w. "+
				"Pre-install extensions using: ./scripts/setup-duckdb-extensions.sh "+
				"or increase timeout with DUCKDB_EXTENSION_TIMEOUT=60s", spec.Name, err)
		}
	}
	return nil
}

// loadAndVerifyCommunityExtension loads and verifies a community extension
// Uses queryRowWithHardTimeout because CGO calls don't respect context cancellation
func (db *DB) loadAndVerifyCommunityExtension(_ context.Context, spec *extensionSpec, optional bool) error {
	// Load the extension with hard timeout
	if err := db.execWithHardTimeout(fmt.Sprintf("LOAD %s;", spec.Name)); err != nil {
		if optional {
			db.setExtensionUnavailable(spec)
			logging.Warn().Str("extension", spec.Name).Err(err).Msg("Extension installed but failed to load")
			return nil
		}
		return fmt.Errorf("failed to load %s extension: %w", spec.Name, err)
	}

	// Verify extension functions are available with hard timeout
	if spec.VerifyQuery != "" {
		if _, err := db.queryRowWithHardTimeout(spec.VerifyQuery); err != nil {
			if optional {
				db.setExtensionUnavailable(spec)
				logging.Warn().Str("extension", spec.Name).Err(err).Msg("Extension functions unavailable")
				return nil
			}
			return fmt.Errorf("%s extension loaded but functions unavailable: %w", spec.Name, err)
		}
	}

	// Extension loaded and verified successfully
	db.setExtensionAvailable(spec)
	return nil
}

// isExtensionLoaded checks if an extension is already loaded
// Uses queryRowWithHardTimeout because CGO calls don't respect context cancellation
func (db *DB) isExtensionLoaded(name string) bool {
	query := fmt.Sprintf("SELECT loaded FROM duckdb_extensions() WHERE extension_name = '%s'", name)
	result, err := db.queryRowWithHardTimeout(query)
	if err != nil {
		return false
	}
	isLoaded, ok := result.(bool)
	return ok && isLoaded
}
