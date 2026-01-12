// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

/*
Package services provides suture.Service wrappers for Cartographus components.

This package adapts existing application components to the suture v4 supervision
model, translating various lifecycle patterns (Start/Stop, Run, ListenAndServe)
into suture's context-aware Serve pattern.

# Overview

Each wrapper implements the suture.Service interface:

	type Service interface {
	    Serve(ctx context.Context) error
	}

The wrappers handle:
  - Lifecycle translation (Start/Stop to Serve pattern)
  - Graceful shutdown via context cancellation
  - Error propagation for supervisor restart decisions
  - Service identification via fmt.Stringer

# Available Services

HTTP Server (HTTPServerService):
  - Wraps *http.Server with graceful shutdown
  - Converts ListenAndServe pattern to Serve
  - Configurable shutdown timeout for draining connections

WebSocket Hub (WebSocketHubService):
  - Wraps websocket.Hub with context support
  - Handles client connection cleanup on shutdown
  - Broadcasts shutdown notification to connected clients

Sync Manager (SyncManagerService):
  - Wraps sync.Manager with Start/Stop lifecycle
  - Coordinates Tautulli/Plex synchronization
  - Handles reconnection on network failures

WAL Services (WALRetryLoopService, WALCompactorService):
  - Wraps wal.RetryLoop and wal.Compactor
  - Handles BadgerDB lifecycle management
  - Build tag: wal (disabled by default)

NATS Components (NATSComponentsService):
  - Wraps NATS JetStream consumer
  - Handles message processing and acknowledgment
  - Build tag: nats (disabled by default)

Detection Engine (DetectionEngineService):
  - Wraps detection.Engine for security monitoring
  - Processes events and generates alerts
  - Handles rule evaluation and notification

Import Service (ImportService):
  - Wraps database import operations
  - Handles long-running Tautulli database imports
  - Provides progress tracking via channels

Multi-Server Services (EmbyService, JellyfinService):
  - Wraps Emby/Jellyfin WebSocket clients
  - Handles real-time playback event streaming
  - Manages reconnection on connection loss

Recommendation Service (RecommendService):
  - Wraps recommendation engine training
  - Handles model training and persistence
  - Runs on configurable schedule

# Usage Example

Creating and registering services:

	import (
	    "net/http"
	    "time"

	    "github.com/tomtom215/cartographus/internal/supervisor"
	    "github.com/tomtom215/cartographus/internal/supervisor/services"
	)

	func setupSupervisor(server *http.Server, hub *websocket.Hub, syncMgr *sync.Manager) {
	    tree, _ := supervisor.NewSupervisorTree(logger, config)

	    // HTTP server with 30s shutdown timeout
	    httpSvc := services.NewHTTPServerService(server, 30*time.Second)
	    tree.AddAPIService(httpSvc)

	    // WebSocket hub
	    wsSvc := services.NewWebSocketHubService(hub)
	    tree.AddMessagingService(wsSvc)

	    // Sync manager
	    syncSvc := services.NewSyncManagerService(syncMgr)
	    tree.AddMessagingService(syncSvc)

	    // Start supervision
	    tree.Serve(ctx)
	}

# Lifecycle Patterns

The package handles three common lifecycle patterns:

Start/Stop Pattern:

	type StartStopper interface {
	    Start(ctx context.Context) error
	    Stop() error
	}

	// Wrapped as:
	func (s *Service) Serve(ctx context.Context) error {
	    if err := s.component.Start(ctx); err != nil {
	        return err
	    }
	    <-ctx.Done()
	    return s.component.Stop()
	}

Run Pattern:

	type Runner interface {
	    Run() error  // Blocks until complete
	}

	// Wrapped as:
	func (s *Service) Serve(ctx context.Context) error {
	    errCh := make(chan error, 1)
	    go func() { errCh <- s.component.Run() }()
	    select {
	    case err := <-errCh: return err
	    case <-ctx.Done(): s.component.Shutdown(); return nil
	    }
	}

ListenAndServe Pattern:

	type Listener interface {
	    ListenAndServe() error
	    Shutdown(ctx context.Context) error
	}

	// Wrapped as:
	func (s *Service) Serve(ctx context.Context) error {
	    go s.server.ListenAndServe()
	    <-ctx.Done()
	    return s.server.Shutdown(shutdownCtx)
	}

# Error Handling

Return values determine supervisor behavior:

	nil         -> Service stopped cleanly, will not restart
	error       -> Service crashed, supervisor will restart
	ctx.Err()   -> Shutdown requested, normal termination

Example error handling:

	func (s *SyncService) Serve(ctx context.Context) error {
	    if err := s.manager.Start(ctx); err != nil {
	        // Transient error - supervisor should restart
	        return fmt.Errorf("sync start failed: %w", err)
	    }

	    <-ctx.Done()

	    if err := s.manager.Stop(); err != nil {
	        // Log but don't propagate - shutdown is complete
	        log.Printf("sync stop warning: %v", err)
	    }

	    return nil  // Clean shutdown, no restart
	}

# Service Identification

All services implement fmt.Stringer for logging:

	func (s *HTTPServerService) String() string {
	    return "http-server"
	}

Suture uses this for log messages:

	INFO http-server: starting
	INFO http-server: stopped
	ERROR http-server: restarting after failure

# Testing

Services can be tested with mock components:

	type MockServer struct {
	    started  bool
	    shutdown bool
	}

	func (m *MockServer) ListenAndServe() error {
	    m.started = true
	    <-time.After(time.Hour) // Block until shutdown
	    return nil
	}

	func (m *MockServer) Shutdown(ctx context.Context) error {
	    m.shutdown = true
	    return nil
	}

	func TestHTTPService(t *testing.T) {
	    mock := &MockServer{}
	    svc := services.NewHTTPServerService(mock, time.Second)

	    ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	    defer cancel()

	    svc.Serve(ctx)

	    if !mock.started { t.Error("server not started") }
	    if !mock.shutdown { t.Error("server not shutdown") }
	}

# Thread Safety

All service wrappers are safe for concurrent use:
  - State is protected by mutexes where needed
  - Context cancellation is handled atomically
  - Multiple Serve calls are not supported (undefined behavior)

# See Also

  - internal/supervisor: SupervisorTree that manages these services
  - github.com/thejerf/suture/v4: Underlying supervision library
  - internal/websocket: WebSocket hub implementation
  - internal/sync: Sync manager implementation
*/
package services
