// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

/*
Package supervisor provides process supervision for Cartographus using suture v4.

This package implements a hierarchical supervisor tree that manages the lifecycle
of all long-running services in the application. It provides Erlang/OTP-style
supervision with automatic restart, failure isolation, and graceful shutdown.

# Overview

The supervisor tree organizes services into three layers for failure isolation:

	RootSupervisor ("cartographus")
	├── DataSupervisor ("data-layer")
	│   ├── WALRetryLoopService (if WAL_ENABLED, build tag: wal)
	│   └── WALCompactorService (if WAL_ENABLED, build tag: wal)
	├── MessagingSupervisor ("messaging-layer")
	│   ├── WebSocketHubService
	│   ├── SyncManagerService
	│   └── NATSComponentsService (if NATS_ENABLED, build tag: nats)
	└── APISupervisor ("api-layer")
	    └── HTTPServerService

This hierarchy ensures that:
  - A crash in sync doesn't affect WebSocket connections
  - WAL failures don't impact API availability
  - Each layer can restart independently

# Key Features

Automatic Restart:
  - Crashed services are automatically restarted
  - Exponential backoff prevents restart storms
  - Configurable failure thresholds and decay rates

Failure Isolation:
  - Services are organized into logical groups
  - Child supervisor failures don't propagate upward
  - Each layer has independent failure counting

Graceful Shutdown:
  - Context cancellation triggers orderly shutdown
  - Configurable shutdown timeout per service
  - UnstoppedServiceReport for debugging hangs

Structured Logging:
  - Integration with slog for structured events
  - Logs service starts, stops, failures, and restarts
  - Event hooks via sutureslog adapter

# Usage Example

Basic setup in main.go:

	import (
	    "log/slog"
	    "github.com/tomtom215/cartographus/internal/supervisor"
	    "github.com/tomtom215/cartographus/internal/supervisor/services"
	)

	func main() {
	    logger := slog.Default()
	    config := supervisor.DefaultTreeConfig()

	    tree, err := supervisor.NewSupervisorTree(logger, config)
	    if err != nil {
	        log.Fatal(err)
	    }

	    // Add services to appropriate layers
	    tree.AddAPIService(services.NewHTTPServerService(server))
	    tree.AddMessagingService(services.NewWebSocketHubService(hub))
	    tree.AddMessagingService(services.NewSyncManagerService(syncMgr))

	    // Start the tree (blocks until context canceled)
	    ctx := context.Background()
	    if err := tree.Serve(ctx); err != nil {
	        log.Printf("Supervisor stopped: %v", err)
	    }
	}

Background operation:

	// Start in background
	errChan := tree.ServeBackground(ctx)

	// Do other setup...

	// Wait for shutdown
	if err := <-errChan; err != nil {
	    log.Printf("Supervisor error: %v", err)
	}

# Configuration

The TreeConfig controls restart behavior:

	config := supervisor.TreeConfig{
	    FailureThreshold: 5.0,          // Failures before backoff
	    FailureDecay:     30.0,         // Seconds for failures to decay
	    FailureBackoff:   15 * time.Second, // Backoff duration
	    ShutdownTimeout:  10 * time.Second, // Per-service shutdown timeout
	}

Default values match suture's production-ready defaults:
  - FailureThreshold: 5 failures
  - FailureDecay: 30 seconds
  - FailureBackoff: 15 seconds
  - ShutdownTimeout: 10 seconds

# Failure Handling

The supervisor uses a failure counter with exponential decay:

1. Each service failure increments the counter
2. Counter decays exponentially over time (FailureDecay seconds)
3. When counter exceeds FailureThreshold, supervisor enters backoff
4. During backoff, restarts are delayed by FailureBackoff duration
5. If failures continue, the child supervisor may be restarted by parent

Example failure scenarios:

	# Single crash - immediate restart
	Service crashes -> Counter: 1 -> Restart immediately

	# Rapid crashes - backoff triggered
	Service crashes 5x in 10s -> Counter: 5+ -> Wait 15s before restart

	# Isolated failures - counter decays
	Service crashes once, stable for 60s -> Counter: ~0.13 -> Normal restart

# Service Interface

All services must implement suture.Service:

	type Service interface {
	    Serve(ctx context.Context) error
	}

Return behavior:
  - Return nil: Service stopped cleanly, will not be restarted
  - Return error: Service crashed, will be restarted
  - Context canceled: Shutdown requested, return promptly

# Build Tags

Optional components are controlled by build tags:

	-tags wal    # Enable WAL services (BadgerDB)
	-tags nats   # Enable NATS/JetStream services

Without these tags, the corresponding service wrappers are no-ops.

# What Is NOT Supervised

DuckDB is intentionally not supervised:
  - It's an embedded library, not a long-running service
  - Connections are managed by the database package
  - Crashes in DuckDB would require process restart anyway

The Plex/Tautulli connections are supervised via SyncManagerService:
  - WebSocket reconnection is handled within the sync manager
  - Circuit breaker provides failure isolation for API calls

# Debugging Shutdown Issues

If services don't stop within the timeout:

	// Get report of unstopped services
	report, err := tree.UnstoppedServiceReport()
	for _, svc := range report {
	    log.Printf("Service didn't stop: %v", svc)
	}

Common causes:
  - Goroutines not respecting context cancellation
  - Blocked network I/O without deadlines
  - Mutex deadlocks during shutdown

# Performance Characteristics

The supervisor tree has minimal overhead:
  - Service check: <1us per iteration
  - Restart: ~1ms (goroutine spawn)
  - Memory: ~1KB per supervised service
  - No polling (event-driven via channels)

# Thread Safety

The SupervisorTree is safe for concurrent use:
  - Services can be added from any goroutine
  - Remove operations are synchronized
  - Multiple services can crash simultaneously

# See Also

  - internal/supervisor/services: Service wrappers
  - github.com/thejerf/suture/v4: Underlying library
  - docs/adr/0004-process-supervision-with-suture.md: ADR
*/
package supervisor
