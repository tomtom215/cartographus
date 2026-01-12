// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package tautulli_import provides direct Tautulli SQLite database import functionality.
//
// This package enables importing playback history directly from Tautulli's SQLite
// database files or backups, without requiring a live Tautulli API connection.
//
// # Use Cases
//
//   - Migration: Import existing Tautulli history when setting up Cartographus
//   - Testing: Use production-like data without affecting live Tautulli instances
//   - Backup Recovery: Restore data from Tautulli backup files
//   - Offline Analysis: Analyze historical data without network connectivity
//
// # Architecture Integration
//
// The importer integrates with the existing event-driven architecture:
//
//	Tautulli SQLite DB
//	       ↓
//	TautulliImporter (this package)
//	       ↓
//	NATS JetStream (internal/eventprocessor)
//	       ↓
//	BadgerDB WAL (internal/wal)
//	       ↓
//	DuckDBConsumer (internal/eventprocessor)
//	       ↓
//	DuckDB (internal/database)
//
// # Deduplication
//
// The importer leverages the existing triple-layer deduplication:
//
//  1. NATS Level: JetStream TrackMsgID with 2-minute duplicate window
//  2. Consumer Level: In-memory cache for EventID, SessionKey, CorrelationKey
//  3. Database Level: INSERT OR IGNORE on unique indexes
//
// This ensures that importing the same database multiple times or importing
// data that was already synced via the API will not create duplicates.
//
// # Tautulli Database Schema
//
// The importer reads from three related tables:
//
//   - session_history: Core playback session data (timing, user, progress)
//   - session_history_metadata: Media metadata (title, type, rating key)
//   - session_history_media_info: Stream quality data (resolution, codec, bitrate)
//
// These tables are joined on the session ID to create complete PlaybackEvent records.
//
// # Progress Tracking
//
// Import progress is tracked in BadgerDB for resumability:
//
//   - Last processed session ID
//   - Total records imported
//   - Error count and last error
//   - Import start/end times
//
// # Example Usage
//
//	cfg := &config.ImportConfig{
//	    Enabled:   true,
//	    DBPath:    "/path/to/tautulli.db",
//	    BatchSize: 1000,
//	}
//
//	importer, err := tautulli_import.NewImporter(cfg, publisher, progress)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	stats, err := importer.Import(ctx)
//	if err != nil {
//	    log.Printf("Import failed: %v", err)
//	}
//	log.Printf("Imported %d records", stats.Imported)
//
//nolint:revive // package name with underscore is intentional for clarity
package tautulli_import
