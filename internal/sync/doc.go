// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

/*
Package sync orchestrates data synchronization from Tautulli to the database.

This package implements the core business logic for fetching playback history from
Tautulli, enriching it with geolocation data, and storing it in the database. It
provides automatic periodic synchronization, manual sync triggers, and circuit
breaker protection for fault tolerance.

Key Components:

  - Manager: Orchestrates periodic synchronization with configurable intervals
  - TautulliClient: HTTP client for Tautulli API with 60+ methods (split across 6 files)
  - Circuit Breaker: Automatic failure detection and recovery (v1.35)
  - Rate Limiting: HTTP 429 handling with exponential backoff
  - Geolocation Caching: Fetches and caches geolocation data from Tautulli's GeoIP API

Architecture:

The sync manager implements a producer-consumer pattern:

1. Fetch: Retrieves playback history from Tautulli in batches (default: 1000 records)
2. Enrich: Adds geolocation data for each unique IP address
3. Store: Inserts playback events into DuckDB database
4. Notify: Broadcasts sync completion via WebSocket to connected clients

Tautulli Client Organization:

The TautulliClient is split into 6 domain-specific files (v1.31 refactoring):
  - tautulli_client.go: Core HTTP client with circuit breaker and rate limiting
  - tautulli_history.go: Playback history and activity methods
  - tautulli_library.go: Library, collection, and playlist methods
  - tautulli_users.go: User management and statistics
  - tautulli_server.go: Server info, exports, and admin operations
  - tautulli_analytics.go: Analytics and reporting endpoints

Plex Client Organization (v1.37+):

The PlexClient provides hybrid data architecture with real-time capabilities:
  - plex.go: Core PlexClient struct, types, and rate limiting
  - plex_request.go: HTTP request helpers and JSON convenience methods
  - plex_sessions.go: Session monitoring, transcode tracking, buffer health (v1.39-v1.41)
  - plex_library.go: Library sections, content, playlists, and search
  - plex_server.go: Server identity, capabilities, devices, and bandwidth
  - plex_sync.go: Historical sync and ongoing data backfill
  - plex_websocket.go: Real-time playback notifications via WebSocket (v1.39)
  - plex_monitoring.go: Transcode and buffer health monitoring (v1.40-v1.41)

Usage Example:

	import (
	    "context"
	    "github.com/tomtom215/cartographus/internal/sync"
	    "github.com/tomtom215/cartographus/internal/config"
	)

	// Create sync manager
	client := sync.NewTautulliClient(cfg.TautulliURL, cfg.TautulliAPIKey)
	manager := sync.NewManager(db, client, &cfg.SyncConfig)

	// Set callback for sync completion
	manager.SetOnSyncCompleted(func(newRecords int, durationMs int64) {
	    log.Printf("Sync completed: %d new records in %dms", newRecords, durationMs)
	})

	// Start periodic sync (runs every cfg.SyncInterval)
	ctx := context.Background()
	if err := manager.Start(ctx); err != nil {
	    log.Fatal(err)
	}

	// Trigger manual sync
	if err := manager.TriggerSync(); err != nil {
	    log.Printf("Manual sync failed: %v", err)
	}

Performance Characteristics:

  - Batch size: 1000 records per fetch (configurable)
  - Sync interval: 15 minutes (default, configurable)
  - Geolocation caching: Reduces redundant API calls for repeated IPs
  - Concurrent safety: Mutex-protected sync execution (only one sync at a time)
  - Large dataset handling: Successfully tested with 100k+ records (v1.11)

Fault Tolerance:

  - Circuit Breaker: Automatic failure detection (60% threshold) with 2-minute open state (v1.35)
  - Rate Limiting: Exponential backoff for HTTP 429 (1s, 2s, 4s, 8s, 16s, max 5 retries)
  - Database Reconnection: Automatic reconnection with exponential backoff (v1.10)
  - Graceful Degradation: Uses "Unknown" location if geolocation fails (v1.9)

Thread Safety:

The Manager is fully thread-safe:
  - Mutex protects concurrent sync execution (syncMu)
  - RWMutex protects shared state (mu)
  - All TautulliClient methods are goroutine-safe

Metrics:

Prometheus metrics are exported for observability:
  - sync_duration_seconds: Sync operation latency
  - sync_records_total: Number of records processed
  - circuit_breaker_state: Circuit breaker state (closed/open/half-open)
  - circuit_breaker_failures_total: Failure counts by state

See Also:

  - internal/database: Data persistence layer
  - internal/config: Configuration management
  - internal/models: Data structures for Tautulli API responses
  - internal/metrics: Prometheus metrics
*/
package sync
