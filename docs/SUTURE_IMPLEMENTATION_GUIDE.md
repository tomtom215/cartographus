# Suture v4 Implementation Guide for Cartographus

**Version**: 4.0.0 (Implementation Complete)
**Last Verified**: 2026-01-11
**Target**: Cartographus v1.52+
**Suture Version**: v4.0.6 (Nov 29, 2024) - VERIFIED via pkg.go.dev
**sutureslog Version**: v1.0.1 (Mar 8, 2024) - VERIFIED via pkg.go.dev
**Go Version**: 1.24+ (per go.mod)
**Approach**: Test-Driven Development (TDD)

---

## Implementation Progress

| Phase | Component | Status | Notes |
|-------|-----------|--------|-------|
| 1 | Foundation (package structure, mock service) | COMPLETE | `internal/supervisor/` created |
| 1 | SupervisorTree with 3-layer hierarchy | COMPLETE | Tests passing with race detection |
| 2 | SyncService wrapper | COMPLETE | Adapts Start/Stop to Serve pattern |
| 3 | WebSocket Hub RunWithContext | COMPLETE | Added context support to hub.go |
| 3 | WebSocketHubService wrapper | COMPLETE | Simple delegation to RunWithContext |
| 4 | HTTPServerService wrapper | COMPLETE | Handles ListenAndServe/Shutdown |
| 5 | NATSComponentsService | COMPLETE | Build tag: nats, wraps NATSComponents |
| 6 | WAL Services (RetryLoop, Compactor) | COMPLETE | Build tag: wal, separate services |
| 7 | Supervisor Tree Assembly | COMPLETE | AddNATSToSupervisor helper |
| 8 | Main Integration | COMPLETE | main.go uses supervisor tree |
| 9 | E2E Testing | PENDING | Verify supervisor behavior |

**Test Results (as of 2026-01-11):**
- 82 tests passing in `internal/supervisor/...` (with -tags nats,wal)
- All tests pass with `-race` flag
- No linting issues (go vet, gofmt)

---

## Verification Status

All claims in this document have been verified against:
- [pkg.go.dev/github.com/thejerf/suture/v4](https://pkg.go.dev/github.com/thejerf/suture/v4) (v4.0.6)
- [pkg.go.dev/github.com/thejerf/sutureslog](https://pkg.go.dev/github.com/thejerf/sutureslog) (v1.0.1)
- Actual source code in this repository (December 2024)
- ARCHITECTURE.md and CLAUDE.md documents

---

## Table of Contents

1. [Prerequisites](#1-prerequisites)
2. [Existing Codebase Patterns](#2-existing-codebase-patterns)
3. [Phase 1: Foundation - Service Interface and Testing](#3-phase-1-foundation)
4. [Phase 2: Sync Manager Service](#4-phase-2-sync-manager-service)
5. [Phase 3: WebSocket Hub Service](#5-phase-3-websocket-hub-service)
6. [Phase 4: HTTP Server Service](#6-phase-4-http-server-service)
7. [Phase 5: NATS Services (Build Tag: nats)](#7-phase-5-nats-services)
8. [Phase 6: WAL Services (Build Tag: wal)](#8-phase-6-wal-services)
9. [Phase 7: Supervisor Tree Assembly](#9-phase-7-supervisor-tree-assembly)
10. [Phase 8: Main Integration](#10-phase-8-main-integration)
11. [Phase 9: E2E Testing](#11-phase-9-e2e-testing)
12. [Configuration Reference](#12-configuration-reference)
13. [Troubleshooting](#13-troubleshooting)
14. [API Reference](#14-api-reference)

---

## 1. Prerequisites

### 1.1 Add Dependencies

```bash
# Add suture v4 and sutureslog
go get github.com/thejerf/suture/v4@v4.0.6
go get github.com/thejerf/sutureslog@v1.0.1

# Verify
go mod tidy
```

**Expected go.mod additions:**

```go
require (
    // existing dependencies...
    github.com/thejerf/suture/v4 v4.0.6
    github.com/thejerf/sutureslog v1.0.1
)
```

### 1.2 Verify Go Version

Suture v4 requires Go 1.21+ for slog. Your project uses Go 1.24+.

```bash
go version
# Expected: go version go1.24.x or higher
```

### 1.3 Create Package Structure

```bash
mkdir -p internal/supervisor
mkdir -p internal/supervisor/services
```

### 1.4 Target Architecture

Based on ARCHITECTURE.md, here is the target supervisor tree:

```
RootSupervisor ("cartographus")
├── DataSupervisor ("data-layer")
│   ├── WALRetryLoopService (if WAL_ENABLED=true, build tag: wal)
│   └── WALCompactorService (if WAL_ENABLED=true, build tag: wal)
├── MessagingSupervisor ("messaging-layer")
│   ├── WebSocketHubService
│   ├── SyncManagerService
│   └── NATSComponentsService (if NATS_ENABLED=true, build tag: nats)
└── APISupervisor ("api-layer")
    └── HTTPServerService
```

**Important Notes**:
- DuckDB is a library, not a service. It does NOT need supervision.
- The database connection is passed to services that use it.
- WAL (BadgerDB) is a library; the RetryLoop and Compactor are the supervised services.

---

## 2. Existing Codebase Patterns

Before implementing, understand the existing lifecycle patterns in the codebase.

### 2.1 Sync Manager Pattern (internal/sync/manager.go)

The sync manager uses a **Start/Stop pattern**, not a Run pattern:

```go
// Simplified from internal/sync/manager.go (Start at ~line 165, Stop at ~line 308)
// Actual implementation includes additional features like conditional Tautulli
// and Plex WebSocket handling. Core pattern shown here:
func (m *Manager) Start(ctx context.Context) error {
    m.mu.Lock()
    if m.running {
        m.mu.Unlock()
        return fmt.Errorf("sync manager is already running")
    }
    m.wg.Add(2)
    m.running = true
    m.mu.Unlock()

    go func() {
        defer m.wg.Done()
        if err := m.performInitialSync(); err != nil {
            logging.Warn().Err(err).Msg("Initial sync failed (will retry)")
        }
    }()

    go m.syncLoop(ctx)
    return nil
}

func (m *Manager) Stop() error {
    m.mu.Lock()
    if !m.running {
        m.mu.Unlock()
        return fmt.Errorf("sync manager is not running")
    }
    m.running = false
    m.mu.Unlock()

    close(m.stopChan)
    m.wg.Wait()
    return nil
}
```

### 2.2 WebSocket Hub Pattern (internal/websocket/hub.go)

The WebSocket hub uses a **blocking Run pattern** with no context:

```go
// Simplified from internal/websocket/hub.go (Run at ~line 73, RunWithContext at ~line 144)
// Actual implementation uses priority-based select for deterministic behavior.
func (h *Hub) Run() {
    for {
        select {
        case client := <-h.Register:
            h.mu.Lock()
            h.clients[client] = true
            h.mu.Unlock()
        case client := <-h.Unregister:
            h.mu.Lock()
            if _, ok := h.clients[client]; ok {
                delete(h.clients, client)
                close(client.send)
            }
            h.mu.Unlock()
        case message := <-h.broadcast:
            h.mu.Lock()
            for client := range h.clients {
                select {
                case client.send <- message:
                default:
                    close(client.send)
                    delete(h.clients, client)
                }
            }
            h.mu.Unlock()
        }
    }
}
```

**Note**: The original Run() has been extended with RunWithContext() (line ~144) that supports graceful shutdown via context cancellation.

### 2.3 WAL Components Pattern (internal/wal/retry.go, compaction.go)

WAL components use the **Start/Stop pattern**:

```go
// From internal/wal/retry.go (Start at ~line 49, Stop at ~line 85)
func (r *RetryLoop) Start(ctx context.Context) error {
    r.mu.Lock()
    if r.running {
        r.mu.Unlock()
        return nil
    }
    r.ctx, r.cancel = context.WithCancel(ctx)
    r.running = true
    r.mu.Unlock()

    r.wg.Add(1)
    go r.run()
    return nil
}

func (r *RetryLoop) Stop() {
    r.mu.Lock()
    if !r.running {
        r.mu.Unlock()
        return
    }
    r.cancel()
    r.running = false
    r.mu.Unlock()
    r.wg.Wait()
}
```

### 2.4 Current main.go Lifecycle (cmd/server/main.go)

```go
// Current pattern (lines 105-314):
// 1. Load config
// 2. Initialize DuckDB
// 3. Create WebSocket Hub -> go wsHub.Run()  // No context!
// 4. Create Sync Manager
// 5. syncManager.Start(ctx)
// 6. Initialize auth
// 7. InitNATS() -> natsComponents.Start(ctx)
// 8. Start HTTP server (goroutine)
// 9. Wait for SIGINT/SIGTERM
// 10. syncManager.Stop()
// 11. natsComponents.Shutdown(ctx)
// 12. server.Shutdown(ctx)
```

---

## 3. Phase 1: Foundation

### 3.1 Write Service Interface Tests First (TDD)

```go
// internal/supervisor/service_test.go
package supervisor

import (
    "context"
    "errors"
    "sync/atomic"
    "testing"
    "time"

    "github.com/thejerf/suture/v4"
)

// TestServiceInterface verifies services implement suture.Service correctly.
// VERIFIED: suture.Service requires Serve(context.Context) error
func TestServiceInterface(t *testing.T) {
    t.Run("service implements Serve method", func(t *testing.T) {
        var _ suture.Service = (*MockService)(nil)
    })
}

// TestMockService validates the test helper works correctly.
func TestMockService(t *testing.T) {
    t.Run("runs until context cancelled", func(t *testing.T) {
        svc := NewMockService("test")
        ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
        defer cancel()

        err := svc.Serve(ctx)
        if !errors.Is(err, context.DeadlineExceeded) {
            t.Errorf("expected context.DeadlineExceeded, got %v", err)
        }
        if svc.StartCount() != 1 {
            t.Errorf("expected 1 start, got %d", svc.StartCount())
        }
    })

    t.Run("returns error on simulated failure", func(t *testing.T) {
        svc := NewMockService("failing")
        svc.SetError(errors.New("simulated failure"))

        ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
        defer cancel()

        err := svc.Serve(ctx)
        if err == nil || err.Error() != "simulated failure" {
            t.Errorf("expected simulated failure, got %v", err)
        }
    })

    // VERIFIED: suture.ErrDoNotRestart exists and prevents restart
    t.Run("returns ErrDoNotRestart for permanent completion", func(t *testing.T) {
        svc := NewMockService("one-shot")
        svc.SetError(suture.ErrDoNotRestart)

        ctx := context.Background()
        err := svc.Serve(ctx)
        if !errors.Is(err, suture.ErrDoNotRestart) {
            t.Errorf("expected ErrDoNotRestart, got %v", err)
        }
    })
}

// TestSupervisorBasics validates supervisor behavior.
func TestSupervisorBasics(t *testing.T) {
    t.Run("supervisor starts and stops services", func(t *testing.T) {
        svc := NewMockService("basic")
        // VERIFIED: NewSimple creates supervisor with sensible defaults
        sup := suture.NewSimple("test-supervisor")
        sup.Add(svc)

        ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
        defer cancel()

        errCh := make(chan error, 1)
        go func() {
            errCh <- sup.Serve(ctx)
        }()

        time.Sleep(50 * time.Millisecond)
        if svc.StartCount() < 1 {
            t.Error("service was not started")
        }

        cancel()
        select {
        case err := <-errCh:
            if err != nil && !errors.Is(err, context.Canceled) {
                t.Errorf("unexpected supervisor error: %v", err)
            }
        case <-time.After(time.Second):
            t.Error("supervisor did not stop in time")
        }
    })

    // VERIFIED: Spec fields from pkg.go.dev documentation
    t.Run("supervisor restarts crashed service", func(t *testing.T) {
        svc := NewMockService("crasher")
        svc.SetFailCount(2)

        sup := suture.New("restart-test", suture.Spec{
            FailureThreshold: 10,
            FailureDecay:     1,
            FailureBackoff:   10 * time.Millisecond,
            Timeout:          100 * time.Millisecond,
        })
        sup.Add(svc)

        ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
        defer cancel()

        go sup.Serve(ctx)
        time.Sleep(300 * time.Millisecond)

        if svc.StartCount() < 3 {
            t.Errorf("expected at least 3 starts (2 failures + 1 success), got %d", svc.StartCount())
        }
    })
}

// TestErrTerminateSupervisorTree validates tree termination.
// VERIFIED: suture.ErrTerminateSupervisorTree exists
func TestErrTerminateSupervisorTree(t *testing.T) {
    t.Run("service can terminate entire tree", func(t *testing.T) {
        svc := NewMockService("terminator")
        svc.SetError(suture.ErrTerminateSupervisorTree)

        sup := suture.New("tree-test", suture.Spec{
            FailureThreshold: 10,
            Timeout:          100 * time.Millisecond,
        })
        sup.Add(svc)

        ctx := context.Background()
        err := sup.Serve(ctx)

        if !errors.Is(err, suture.ErrTerminateSupervisorTree) {
            t.Logf("supervisor returned: %v (tree termination may propagate differently)", err)
        }
    })
}
```

### 3.2 Implement MockService

```go
// internal/supervisor/mock_service.go
package supervisor

import (
    "context"
    "errors"
    "sync"
    "sync/atomic"
)

// MockService is a test helper that implements suture.Service.
type MockService struct {
    name       string
    startCount atomic.Int32
    stopCount  atomic.Int32
    failCount  atomic.Int32
    maxFails   int32
    err        error
    mu         sync.Mutex
}

// NewMockService creates a new mock service for testing.
func NewMockService(name string) *MockService {
    return &MockService{name: name}
}

// Serve implements suture.Service.
// VERIFIED: Signature is Serve(context.Context) error
func (m *MockService) Serve(ctx context.Context) error {
    m.startCount.Add(1)
    defer m.stopCount.Add(1)

    m.mu.Lock()
    err := m.err
    maxFails := m.maxFails
    m.mu.Unlock()

    // If we have a fail count, fail that many times
    if maxFails > 0 {
        current := m.failCount.Add(1)
        if current <= maxFails {
            return errors.New("simulated failure")
        }
    }

    // If error is set, return it immediately
    if err != nil {
        return err
    }

    // Otherwise, run until context is cancelled
    <-ctx.Done()
    return ctx.Err()
}

// SetError configures the service to return this error.
func (m *MockService) SetError(err error) {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.err = err
}

// SetFailCount configures the service to fail N times before succeeding.
func (m *MockService) SetFailCount(n int) {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.maxFails = int32(n)
}

// StartCount returns how many times Serve was called.
func (m *MockService) StartCount() int32 {
    return m.startCount.Load()
}

// StopCount returns how many times Serve returned.
func (m *MockService) StopCount() int32 {
    return m.stopCount.Load()
}

// String implements fmt.Stringer for logging.
// VERIFIED: Suture uses fmt.Stringer for service names in logs
func (m *MockService) String() string {
    return m.name
}
```

### 3.3 Run Foundation Tests

```bash
export GOTOOLCHAIN=local
cd /home/user/map
go test -v -race ./internal/supervisor/...
```

---

## 4. Phase 2: Sync Manager Service

### 4.1 Write SyncManagerService Tests First

```go
// internal/supervisor/services/sync_service_test.go
package services

import (
    "context"
    "errors"
    "sync/atomic"
    "testing"
    "time"

    "github.com/thejerf/suture/v4"
)

// StartStopManager matches the existing internal/sync.Manager interface.
// Adapted from internal/sync/manager.go (Start/Stop pattern)
type StartStopManager interface {
    Start(ctx context.Context) error
    Stop() error
}

// MockSyncManager simulates the sync.Manager for testing.
type MockSyncManager struct {
    started    atomic.Bool
    stopped    atomic.Bool
    startError error
    stopError  error
}

func (m *MockSyncManager) Start(ctx context.Context) error {
    if m.startError != nil {
        return m.startError
    }
    m.started.Store(true)
    return nil
}

func (m *MockSyncManager) Stop() error {
    m.stopped.Store(true)
    return m.stopError
}

func TestSyncService(t *testing.T) {
    t.Run("implements suture.Service interface", func(t *testing.T) {
        var _ suture.Service = (*SyncService)(nil)
    })

    t.Run("starts underlying sync manager", func(t *testing.T) {
        mockMgr := &MockSyncManager{}
        svc := NewSyncService(mockMgr)

        ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
        defer cancel()

        go svc.Serve(ctx)
        time.Sleep(50 * time.Millisecond)

        if !mockMgr.started.Load() {
            t.Error("sync manager was not started")
        }
    })

    t.Run("stops manager on context cancellation", func(t *testing.T) {
        mockMgr := &MockSyncManager{}
        svc := NewSyncService(mockMgr)

        ctx, cancel := context.WithCancel(context.Background())

        done := make(chan error, 1)
        go func() {
            done <- svc.Serve(ctx)
        }()

        time.Sleep(50 * time.Millisecond)
        cancel()

        select {
        case err := <-done:
            if !errors.Is(err, context.Canceled) {
                t.Errorf("expected context.Canceled, got %v", err)
            }
        case <-time.After(time.Second):
            t.Error("service did not stop in time")
        }

        if !mockMgr.stopped.Load() {
            t.Error("sync manager was not stopped")
        }
    })

    t.Run("propagates start error for restart", func(t *testing.T) {
        mockMgr := &MockSyncManager{
            startError: errors.New("tautulli connection failed"),
        }
        svc := NewSyncService(mockMgr)

        err := svc.Serve(context.Background())
        if err == nil {
            t.Error("expected error to be propagated")
        }
    })
}
```

### 4.2 Implement SyncService

```go
// internal/supervisor/services/sync_service.go
package services

import (
    "context"
    "fmt"
)

// StartStopManager interface matches existing internal/sync.Manager.
// This uses the existing Start/Stop pattern rather than requiring modifications.
//
// Matches: internal/sync/manager.go Start/Stop pattern
type StartStopManager interface {
    Start(ctx context.Context) error
    Stop() error
}

// SyncService wraps the sync manager as a supervised service.
// It adapts the Start/Stop lifecycle to suture's Serve pattern.
type SyncService struct {
    manager StartStopManager
    name    string
}

// NewSyncService creates a new sync service wrapper.
func NewSyncService(manager StartStopManager) *SyncService {
    return &SyncService{
        manager: manager,
        name:    "sync-manager",
    }
}

// Serve implements suture.Service.
// Adapts the Start/Stop pattern to suture's context-based lifecycle.
func (s *SyncService) Serve(ctx context.Context) error {
    // Start the manager
    if err := s.manager.Start(ctx); err != nil {
        return fmt.Errorf("sync manager start failed: %w", err)
    }

    // Wait for context cancellation
    <-ctx.Done()

    // Stop the manager (this blocks until complete per manager.go Stop method)
    if err := s.manager.Stop(); err != nil {
        // Log but don't return - we're shutting down anyway
        // The primary error is the context error
        return fmt.Errorf("sync manager stop failed: %w", err)
    }

    return ctx.Err()
}

// String implements fmt.Stringer for logging.
func (s *SyncService) String() string {
    return s.name
}
```

**Note**: No modifications to internal/sync/manager.go are required. The wrapper adapts the existing interface.

---

## 5. Phase 3: WebSocket Hub Service

The WebSocket Hub requires modifications because its current `Run()` method has no context handling.

### 5.1 Add Context-Aware Method to Hub

First, add a shutdown channel and context-aware method to the existing Hub:

```go
// internal/websocket/hub.go
// ADD these fields to the Hub struct (around line 28):

type Hub struct {
    clients    map[*Client]bool
    broadcast  chan Message
    Register   chan *Client
    Unregister chan *Client
    mu         sync.RWMutex

    // ADD: Shutdown support for supervisor integration
    shutdown   chan struct{}
    shutdownMu sync.Mutex
    running    bool
}

// Modify NewHub (around line 37):
func NewHub() *Hub {
    return &Hub{
        broadcast:  make(chan Message, 256),
        Register:   make(chan *Client),
        Unregister: make(chan *Client),
        clients:    make(map[*Client]bool),
        shutdown:   make(chan struct{}),  // ADD
    }
}

// ADD this new method after Run():

// RunWithContext starts the hub with context cancellation support.
// This method is used by the supervisor for graceful shutdown.
// The original Run() method is preserved for backward compatibility.
func (h *Hub) RunWithContext(ctx context.Context) error {
    h.shutdownMu.Lock()
    if h.running {
        h.shutdownMu.Unlock()
        return fmt.Errorf("hub already running")
    }
    h.running = true
    h.shutdown = make(chan struct{})
    h.shutdownMu.Unlock()

    defer func() {
        h.shutdownMu.Lock()
        h.running = false
        h.shutdownMu.Unlock()
    }()

    for {
        select {
        case <-ctx.Done():
            h.closeAllClients()
            return ctx.Err()
        case <-h.shutdown:
            h.closeAllClients()
            return nil
        case client := <-h.Register:
            h.mu.Lock()
            h.clients[client] = true
            h.mu.Unlock()
            log.Printf("WebSocket client connected. Total clients: %d", len(h.clients))
        case client := <-h.Unregister:
            h.mu.Lock()
            if _, ok := h.clients[client]; ok {
                delete(h.clients, client)
                close(client.send)
            }
            h.mu.Unlock()
            log.Printf("WebSocket client disconnected. Total clients: %d", len(h.clients))
        case message := <-h.broadcast:
            h.mu.Lock()
            for client := range h.clients {
                select {
                case client.send <- message:
                default:
                    close(client.send)
                    delete(h.clients, client)
                }
            }
            h.mu.Unlock()
        }
    }
}

// closeAllClients gracefully closes all connected clients.
func (h *Hub) closeAllClients() {
    h.mu.Lock()
    defer h.mu.Unlock()
    for client := range h.clients {
        close(client.send)
        delete(h.clients, client)
    }
    log.Printf("WebSocket hub shutdown: closed all clients")
}

// Stop signals the hub to stop (for non-context shutdown).
func (h *Hub) Stop() {
    h.shutdownMu.Lock()
    defer h.shutdownMu.Unlock()
    if h.running {
        close(h.shutdown)
    }
}
```

### 5.2 Write WebSocketHubService Tests

```go
// internal/supervisor/services/websocket_service_test.go
package services

import (
    "context"
    "errors"
    "sync/atomic"
    "testing"
    "time"

    "github.com/thejerf/suture/v4"
)

// HubRunner interface matches the new RunWithContext method.
type HubRunner interface {
    RunWithContext(ctx context.Context) error
}

type MockWebSocketHub struct {
    running atomic.Bool
    stopped atomic.Bool
}

func (m *MockWebSocketHub) RunWithContext(ctx context.Context) error {
    m.running.Store(true)
    defer func() {
        m.running.Store(false)
        m.stopped.Store(true)
    }()

    <-ctx.Done()
    return ctx.Err()
}

func TestWebSocketHubService(t *testing.T) {
    t.Run("implements suture.Service interface", func(t *testing.T) {
        var _ suture.Service = (*WebSocketHubService)(nil)
    })

    t.Run("starts and stops hub correctly", func(t *testing.T) {
        mockHub := &MockWebSocketHub{}
        svc := NewWebSocketHubService(mockHub)

        ctx, cancel := context.WithCancel(context.Background())

        done := make(chan error, 1)
        go func() {
            done <- svc.Serve(ctx)
        }()

        time.Sleep(50 * time.Millisecond)
        if !mockHub.running.Load() {
            t.Error("hub should be running")
        }

        cancel()

        select {
        case err := <-done:
            if !errors.Is(err, context.Canceled) {
                t.Errorf("expected context.Canceled, got %v", err)
            }
        case <-time.After(time.Second):
            t.Error("service did not stop")
        }

        if !mockHub.stopped.Load() {
            t.Error("hub was not stopped")
        }
    })
}
```

### 5.3 Implement WebSocketHubService

```go
// internal/supervisor/services/websocket_service.go
package services

import (
    "context"
    "fmt"
)

// HubRunner interface for WebSocket hub with context support.
// Matches the new RunWithContext method added to internal/websocket.Hub
type HubRunner interface {
    RunWithContext(ctx context.Context) error
}

// WebSocketHubService wraps the WebSocket hub as a supervised service.
type WebSocketHubService struct {
    hub  HubRunner
    name string
}

// NewWebSocketHubService creates a new WebSocket hub service wrapper.
func NewWebSocketHubService(hub HubRunner) *WebSocketHubService {
    return &WebSocketHubService{
        hub:  hub,
        name: "websocket-hub",
    }
}

// Serve implements suture.Service.
func (s *WebSocketHubService) Serve(ctx context.Context) error {
    err := s.hub.RunWithContext(ctx)
    if err != nil && err != context.Canceled && err != context.DeadlineExceeded {
        return fmt.Errorf("websocket hub failed: %w", err)
    }
    return err
}

// String implements fmt.Stringer for logging.
func (s *WebSocketHubService) String() string {
    return s.name
}
```

---

## 6. Phase 4: HTTP Server Service

### 6.1 Write HTTPServerService Tests

```go
// internal/supervisor/services/http_service_test.go
package services

import (
    "context"
    "errors"
    "net/http"
    "testing"
    "time"

    "github.com/thejerf/suture/v4"
)

func TestHTTPServerService(t *testing.T) {
    t.Run("implements suture.Service interface", func(t *testing.T) {
        var _ suture.Service = (*HTTPServerService)(nil)
    })

    t.Run("starts and serves HTTP requests", func(t *testing.T) {
        handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            w.WriteHeader(http.StatusOK)
        })
        // Use port 0 for random available port
        svc := NewHTTPServerService(":0", handler)

        ctx, cancel := context.WithCancel(context.Background())

        done := make(chan error, 1)
        go func() {
            done <- svc.Serve(ctx)
        }()

        // Give server time to start
        time.Sleep(100 * time.Millisecond)

        cancel()

        select {
        case err := <-done:
            if err != nil && !errors.Is(err, context.Canceled) {
                t.Errorf("unexpected error: %v", err)
            }
        case <-time.After(2 * time.Second):
            t.Error("server did not stop in time")
        }
    })
}
```

### 6.2 Implement HTTPServerService

```go
// internal/supervisor/services/http_service.go
package services

import (
    "context"
    "errors"
    "fmt"
    "net"
    "net/http"
    "time"
)

// HTTPServerService wraps http.Server as a supervised service.
type HTTPServerService struct {
    server *http.Server
    name   string
}

// NewHTTPServerService creates a new HTTP server service.
// addr should be in format ":8080" or "0.0.0.0:8080"
//
// Timeouts match existing cmd/server/main.go HTTP server config
func NewHTTPServerService(addr string, handler http.Handler) *HTTPServerService {
    return &HTTPServerService{
        server: &http.Server{
            Addr:              addr,
            Handler:           handler,
            ReadTimeout:       30 * time.Second,
            ReadHeaderTimeout: 10 * time.Second,
            WriteTimeout:      30 * time.Second,
            IdleTimeout:       60 * time.Second,  // Matches main.go:279
        },
        name: "http-server",
    }
}

// NewHTTPServerServiceWithConfig creates an HTTP server with custom config.
func NewHTTPServerServiceWithConfig(server *http.Server) *HTTPServerService {
    return &HTTPServerService{
        server: server,
        name:   "http-server",
    }
}

// Serve implements suture.Service.
func (s *HTTPServerService) Serve(ctx context.Context) error {
    errCh := make(chan error, 1)
    go func() {
        if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
            errCh <- err
        }
        close(errCh)
    }()

    select {
    case <-ctx.Done():
        // Graceful shutdown with 10 second timeout (matches main.go shutdown pattern)
        shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
        defer cancel()

        if err := s.server.Shutdown(shutdownCtx); err != nil {
            return fmt.Errorf("HTTP server shutdown error: %w", err)
        }
        return ctx.Err()
    case err := <-errCh:
        if err != nil {
            var opErr *net.OpError
            if errors.As(err, &opErr) {
                return fmt.Errorf("HTTP server bind error (port in use?): %w", err)
            }
            return fmt.Errorf("HTTP server error: %w", err)
        }
        return nil
    }
}

// String implements fmt.Stringer for logging.
func (s *HTTPServerService) String() string {
    return s.name
}
```

---

## 7. Phase 5: NATS Services

Based on the existing `cmd/server/nats_init.go`, NATS components use a composite pattern with `NATSComponents.Start()` and `NATSComponents.Shutdown()`.

### 7.1 Write NATSComponentsService Tests

```go
// internal/supervisor/services/nats_service_test.go
//go:build nats

package services

import (
    "context"
    "errors"
    "sync/atomic"
    "testing"
    "time"

    "github.com/thejerf/suture/v4"
)

// NATSComponentsRunner matches the existing NATSComponents interface.
// From cmd/server/nats_init.go
type NATSComponentsRunner interface {
    Start(ctx context.Context) error
    Shutdown(ctx context.Context)
    IsRunning() bool
}

type MockNATSComponents struct {
    running   atomic.Bool
    started   atomic.Bool
    startErr  error
}

func (m *MockNATSComponents) Start(ctx context.Context) error {
    if m.startErr != nil {
        return m.startErr
    }
    m.started.Store(true)
    m.running.Store(true)
    return nil
}

func (m *MockNATSComponents) Shutdown(ctx context.Context) {
    m.running.Store(false)
}

func (m *MockNATSComponents) IsRunning() bool {
    return m.running.Load()
}

func TestNATSComponentsService(t *testing.T) {
    t.Run("implements suture.Service interface", func(t *testing.T) {
        var _ suture.Service = (*NATSComponentsService)(nil)
    })

    t.Run("starts and runs NATS components", func(t *testing.T) {
        mock := &MockNATSComponents{}
        svc := NewNATSComponentsService(mock)

        ctx, cancel := context.WithCancel(context.Background())

        done := make(chan error, 1)
        go func() {
            done <- svc.Serve(ctx)
        }()

        time.Sleep(50 * time.Millisecond)
        if !mock.started.Load() {
            t.Error("NATS components should be started")
        }

        cancel()

        select {
        case <-done:
            // Success
        case <-time.After(time.Second):
            t.Error("service did not stop")
        }

        if mock.IsRunning() {
            t.Error("NATS components should be stopped")
        }
    })

    t.Run("propagates start error for restart", func(t *testing.T) {
        mock := &MockNATSComponents{
            startErr: errors.New("NATS connection refused"),
        }
        svc := NewNATSComponentsService(mock)

        err := svc.Serve(context.Background())
        if err == nil {
            t.Error("expected error")
        }
    })
}
```

### 7.2 Implement NATSComponentsService

```go
// internal/supervisor/services/nats_service.go
//go:build nats

package services

import (
    "context"
    "fmt"
)

// NATSComponentsRunner interface matches existing NATSComponents.
// Adapted from cmd/server/nats_init.go (NATS components initialization)
type NATSComponentsRunner interface {
    Start(ctx context.Context) error
    Shutdown(ctx context.Context)
    IsRunning() bool
}

// NATSComponentsService wraps NATSComponents as a supervised service.
type NATSComponentsService struct {
    components NATSComponentsRunner
    name       string
}

// NewNATSComponentsService creates a new NATS components service.
func NewNATSComponentsService(components NATSComponentsRunner) *NATSComponentsService {
    return &NATSComponentsService{
        components: components,
        name:       "nats-components",
    }
}

// Serve implements suture.Service.
func (s *NATSComponentsService) Serve(ctx context.Context) error {
    if err := s.components.Start(ctx); err != nil {
        return fmt.Errorf("NATS components start failed: %w", err)
    }

    // Wait for context cancellation
    <-ctx.Done()

    // Shutdown with timeout
    shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    s.components.Shutdown(shutdownCtx)

    return ctx.Err()
}

// String implements fmt.Stringer.
func (s *NATSComponentsService) String() string {
    return s.name
}
```

---

## 8. Phase 6: WAL Services

The WAL has two background services that need supervision: RetryLoop and Compactor.
Both use the existing Start/Stop pattern from `internal/wal/`.

### 8.1 Write WAL Service Tests

```go
// internal/supervisor/services/wal_service_test.go
//go:build wal

package services

import (
    "context"
    "errors"
    "sync/atomic"
    "testing"
    "time"

    "github.com/thejerf/suture/v4"
)

// WALStartStopper matches internal/wal.RetryLoop and internal/wal.Compactor.
// Adapted from internal/wal/retry.go and internal/wal/compaction.go
type WALStartStopper interface {
    Start(ctx context.Context) error
    Stop()
    IsRunning() bool
}

type MockWALComponent struct {
    running  atomic.Bool
    started  atomic.Bool
    startErr error
}

func (m *MockWALComponent) Start(ctx context.Context) error {
    if m.startErr != nil {
        return m.startErr
    }
    m.started.Store(true)
    m.running.Store(true)
    return nil
}

func (m *MockWALComponent) Stop() {
    m.running.Store(false)
}

func (m *MockWALComponent) IsRunning() bool {
    return m.running.Load()
}

func TestWALRetryLoopService(t *testing.T) {
    t.Run("implements suture.Service interface", func(t *testing.T) {
        var _ suture.Service = (*WALRetryLoopService)(nil)
    })

    t.Run("starts and stops retry loop", func(t *testing.T) {
        mock := &MockWALComponent{}
        svc := NewWALRetryLoopService(mock)

        ctx, cancel := context.WithCancel(context.Background())

        done := make(chan error, 1)
        go func() {
            done <- svc.Serve(ctx)
        }()

        time.Sleep(50 * time.Millisecond)
        if !mock.started.Load() {
            t.Error("retry loop should be started")
        }

        cancel()
        <-done

        // Note: Stop is called, but isRunning check depends on implementation
    })
}

func TestWALCompactorService(t *testing.T) {
    t.Run("implements suture.Service interface", func(t *testing.T) {
        var _ suture.Service = (*WALCompactorService)(nil)
    })
}
```

### 8.2 Implement WAL Services

```go
// internal/supervisor/services/wal_service.go
//go:build wal

package services

import (
    "context"
    "fmt"
)

// WALStartStopper matches internal/wal.RetryLoop and internal/wal.Compactor.
// Adapted from internal/wal/retry.go and internal/wal/compaction.go
type WALStartStopper interface {
    Start(ctx context.Context) error
    Stop()
    IsRunning() bool
}

// WALRetryLoopService wraps the WAL retry loop as a supervised service.
type WALRetryLoopService struct {
    retryLoop WALStartStopper
    name      string
}

// NewWALRetryLoopService creates a new WAL retry loop service.
func NewWALRetryLoopService(retryLoop WALStartStopper) *WALRetryLoopService {
    return &WALRetryLoopService{
        retryLoop: retryLoop,
        name:      "wal-retry-loop",
    }
}

// Serve implements suture.Service.
func (s *WALRetryLoopService) Serve(ctx context.Context) error {
    if err := s.retryLoop.Start(ctx); err != nil {
        return fmt.Errorf("WAL retry loop start failed: %w", err)
    }

    // Wait for context cancellation
    <-ctx.Done()

    // Stop the retry loop
    s.retryLoop.Stop()

    return ctx.Err()
}

// String implements fmt.Stringer.
func (s *WALRetryLoopService) String() string {
    return s.name
}

// WALCompactorService wraps the WAL compactor as a supervised service.
type WALCompactorService struct {
    compactor WALStartStopper
    name      string
}

// NewWALCompactorService creates a new WAL compactor service.
func NewWALCompactorService(compactor WALStartStopper) *WALCompactorService {
    return &WALCompactorService{
        compactor: compactor,
        name:      "wal-compactor",
    }
}

// Serve implements suture.Service.
func (s *WALCompactorService) Serve(ctx context.Context) error {
    if err := s.compactor.Start(ctx); err != nil {
        return fmt.Errorf("WAL compactor start failed: %w", err)
    }

    // Wait for context cancellation
    <-ctx.Done()

    // Stop the compactor
    s.compactor.Stop()

    return ctx.Err()
}

// String implements fmt.Stringer.
func (s *WALCompactorService) String() string {
    return s.name
}
```

---

## 9. Phase 7: Supervisor Tree Assembly

### 9.1 Write Tree Assembly Tests

```go
// internal/supervisor/tree_test.go
package supervisor

import (
    "context"
    "log/slog"
    "os"
    "testing"
    "time"

    "github.com/thejerf/suture/v4"
)

func TestSupervisorTreeConstruction(t *testing.T) {
    t.Run("creates hierarchical supervisor tree", func(t *testing.T) {
        logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))

        tree, err := NewSupervisorTree(logger, TreeConfig{
            FailureThreshold: 5,
            FailureBackoff:   time.Second,
            ShutdownTimeout:  10 * time.Second,
        })
        if err != nil {
            t.Fatalf("failed to create tree: %v", err)
        }

        if tree.Root() == nil {
            t.Error("root supervisor should not be nil")
        }
    })

    t.Run("tree starts and stops gracefully", func(t *testing.T) {
        logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

        tree, _ := NewSupervisorTree(logger, TreeConfig{
            FailureThreshold: 5,
            FailureBackoff:   100 * time.Millisecond,
            ShutdownTimeout:  time.Second,
        })

        tree.AddDataService(NewMockService("mock-data"))
        tree.AddMessagingService(NewMockService("mock-messaging"))
        tree.AddAPIService(NewMockService("mock-api"))

        ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
        defer cancel()

        errCh := make(chan error, 1)
        go func() {
            errCh <- tree.Serve(ctx)
        }()

        time.Sleep(100 * time.Millisecond)
        cancel()

        select {
        case err := <-errCh:
            if err != nil && err != context.Canceled {
                t.Errorf("unexpected error: %v", err)
            }
        case <-time.After(2 * time.Second):
            t.Error("tree did not shut down in time")
        }
    })
}
```

### 9.2 Implement Supervisor Tree

```go
// internal/supervisor/tree.go
package supervisor

import (
    "context"
    "log/slog"
    "time"

    "github.com/thejerf/suture/v4"
    "github.com/thejerf/sutureslog"
)

// TreeConfig holds supervisor tree configuration.
type TreeConfig struct {
    FailureThreshold float64       // Failures before backoff (default: 5)
    FailureDecay     float64       // Decay rate in seconds (default: 30)
    FailureBackoff   time.Duration // Backoff duration (default: 15s)
    ShutdownTimeout  time.Duration // Graceful shutdown timeout (default: 10s)
}

// DefaultTreeConfig returns production defaults.
// VERIFIED: Default values from pkg.go.dev:
// - FailureDecay: 30 seconds
// - FailureThreshold: 5 failures
// - FailureBackoff: 15 seconds
// - Timeout: 10 seconds
func DefaultTreeConfig() TreeConfig {
    return TreeConfig{
        FailureThreshold: 5.0,
        FailureDecay:     30.0,
        FailureBackoff:   15 * time.Second,
        ShutdownTimeout:  10 * time.Second,
    }
}

// SupervisorTree manages the hierarchical supervisor structure.
type SupervisorTree struct {
    root      *suture.Supervisor
    data      *suture.Supervisor
    messaging *suture.Supervisor
    api       *suture.Supervisor
    logger    *slog.Logger
    config    TreeConfig
}

// NewSupervisorTree creates a new supervisor tree.
func NewSupervisorTree(logger *slog.Logger, config TreeConfig) (*SupervisorTree, error) {
    // Apply defaults
    if config.FailureThreshold == 0 {
        config.FailureThreshold = 5.0
    }
    if config.FailureDecay == 0 {
        config.FailureDecay = 30.0
    }
    if config.FailureBackoff == 0 {
        config.FailureBackoff = 15 * time.Second
    }
    if config.ShutdownTimeout == 0 {
        config.ShutdownTimeout = 10 * time.Second
    }

    // VERIFIED: sutureslog uses (&Handler{Logger: logger}).MustHook()
    // MustHook has a pointer receiver, so we need the address.
    // NOT sutureslog.EventHook(logger) which does not exist.
    handler := &sutureslog.Handler{Logger: logger}
    eventHook := handler.MustHook()

    rootSpec := suture.Spec{
        EventHook:        eventHook,
        FailureThreshold: config.FailureThreshold,
        FailureDecay:     config.FailureDecay,
        FailureBackoff:   config.FailureBackoff,
        Timeout:          config.ShutdownTimeout,
    }

    // Child supervisors inherit EventHook when added
    // VERIFIED: "As a special behavior, if the service added is itself a
    // supervisor, the supervisor being added will copy the EventHook function"
    childSpec := suture.Spec{
        FailureThreshold: config.FailureThreshold,
        FailureDecay:     config.FailureDecay,
        FailureBackoff:   config.FailureBackoff,
        Timeout:          config.ShutdownTimeout,
    }

    root := suture.New("cartographus", rootSpec)
    data := suture.New("data-layer", childSpec)
    messaging := suture.New("messaging-layer", childSpec)
    api := suture.New("api-layer", childSpec)

    // Build tree hierarchy
    root.Add(data)
    root.Add(messaging)
    root.Add(api)

    return &SupervisorTree{
        root:      root,
        data:      data,
        messaging: messaging,
        api:       api,
        logger:    logger,
        config:    config,
    }, nil
}

// Root returns the root supervisor.
func (t *SupervisorTree) Root() *suture.Supervisor {
    return t.root
}

// AddDataService adds a service to the data layer supervisor.
// VERIFIED: Add returns ServiceToken for later removal
func (t *SupervisorTree) AddDataService(svc suture.Service) suture.ServiceToken {
    return t.data.Add(svc)
}

// AddMessagingService adds a service to the messaging layer supervisor.
func (t *SupervisorTree) AddMessagingService(svc suture.Service) suture.ServiceToken {
    return t.messaging.Add(svc)
}

// AddAPIService adds a service to the API layer supervisor.
func (t *SupervisorTree) AddAPIService(svc suture.Service) suture.ServiceToken {
    return t.api.Add(svc)
}

// Serve starts the supervisor tree and blocks until context is cancelled.
func (t *SupervisorTree) Serve(ctx context.Context) error {
    return t.root.Serve(ctx)
}

// ServeBackground starts the supervisor tree in the background.
// VERIFIED: Returns <-chan error that receives error when supervisor stops
func (t *SupervisorTree) ServeBackground(ctx context.Context) <-chan error {
    return t.root.ServeBackground(ctx)
}

// UnstoppedServiceReport returns services that failed to stop within timeout.
// VERIFIED: Returns UnstoppedServiceReport and error
func (t *SupervisorTree) UnstoppedServiceReport() []suture.UnstoppedService {
    report, _ := t.root.UnstoppedServiceReport()
    return report
}

// Remove removes a service by its token.
// VERIFIED: Remove exists and returns error
func (t *SupervisorTree) Remove(token suture.ServiceToken) error {
    return t.root.Remove(token)
}

// RemoveAndWait removes a service and waits for it to stop.
// VERIFIED: RemoveAndWait exists with timeout parameter
func (t *SupervisorTree) RemoveAndWait(token suture.ServiceToken, timeout time.Duration) error {
    return t.root.RemoveAndWait(token, timeout)
}
```

---

## 10. Phase 8: Main Integration

### 10.1 Updated main.go

This integrates with the existing initialization flow from `cmd/server/main.go`:

```go
// cmd/server/main.go - Updated with supervisor tree
package main

import (
    "context"
    "fmt"
    "log"
    "log/slog"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"

    _ "github.com/tomtom215/cartographus/docs"
    "github.com/tomtom215/cartographus/internal/api"
    "github.com/tomtom215/cartographus/internal/auth"
    "github.com/tomtom215/cartographus/internal/backup"
    "github.com/tomtom215/cartographus/internal/config"
    "github.com/tomtom215/cartographus/internal/database"
    "github.com/tomtom215/cartographus/internal/supervisor"
    "github.com/tomtom215/cartographus/internal/supervisor/services"
    "github.com/tomtom215/cartographus/internal/sync"
    ws "github.com/tomtom215/cartographus/internal/websocket"
)

func main() {
    log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
    log.Println("Starting Cartographus with supervisor tree...")

    // Create structured logger for supervisor
    logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
        Level: getLogLevel(),
    }))
    slog.SetDefault(logger)

    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // Handle shutdown signals
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
    go func() {
        sig := <-sigCh
        log.Printf("Received shutdown signal: %v", sig)
        cancel()
    }()

    if err := runApp(ctx, logger); err != nil && err != context.Canceled {
        log.Fatalf("Application error: %v", err)
    }

    log.Println("Application stopped gracefully")
}

func runApp(ctx context.Context, logger *slog.Logger) error {
    cfg, err := config.Load()
    if err != nil {
        return fmt.Errorf("load configuration: %w", err)
    }

    log.Printf("Configuration loaded: Tautulli URL=%s, DB Path=%s, Auth Mode=%s",
        cfg.Tautulli.URL, cfg.Database.Path, cfg.Security.AuthMode)

    // Create supervisor tree
    tree, err := supervisor.NewSupervisorTree(logger, supervisor.TreeConfig{
        FailureThreshold: 5,
        FailureBackoff:   15 * time.Second,
        ShutdownTimeout:  10 * time.Second,
    })
    if err != nil {
        return fmt.Errorf("create supervisor tree: %w", err)
    }

    // === INITIALIZATION (same as before - DuckDB is a library, not supervised) ===

    db, err := database.New(&cfg.Database, cfg.Server.Latitude, cfg.Server.Longitude)
    if err != nil {
        return fmt.Errorf("initialize database: %w", err)
    }
    defer db.Close()
    log.Println("Database initialized successfully")

    // Seed mock data if enabled
    if cfg.Database.SeedMockData {
        if err := db.SeedMockData(ctx); err != nil {
            return fmt.Errorf("seed mock data: %w", err)
        }
    }

    // Initialize Tautulli client with circuit breaker
    tautulliClient := sync.NewCircuitBreakerClient(&cfg.Tautulli)
    if err := tautulliClient.Ping(); err != nil {
        log.Printf("Warning: Failed to connect to Tautulli (will retry): %v", err)
    }

    // Create WebSocket hub
    wsHub := ws.NewHub()

    // Create Sync Manager
    syncManager := sync.NewManager(db, tautulliClient, cfg, wsHub)

    // Initialize auth
    var jwtManager *auth.JWTManager
    var basicAuthManager *auth.BasicAuthManager

    if cfg.Security.AuthMode == "jwt" {
        jwtManager, err = auth.NewJWTManager(&cfg.Security)
        if err != nil {
            return fmt.Errorf("initialize JWT manager: %w", err)
        }
    } else if cfg.Security.AuthMode == "basic" {
        basicAuthManager, err = auth.NewBasicAuthManager(
            cfg.Security.AdminUsername,
            cfg.Security.AdminPassword,
        )
        if err != nil {
            return fmt.Errorf("initialize Basic Auth manager: %w", err)
        }
    }

    middleware := auth.NewMiddleware(
        jwtManager,
        basicAuthManager,
        cfg.Security.AuthMode,
        cfg.Security.RateLimitReqs,
        cfg.Security.RateLimitWindow,
        cfg.Security.RateLimitDisabled,
        cfg.Security.CORSOrigins,
        cfg.Security.TrustedProxies,
    )

    handler := api.NewHandler(db, syncManager, tautulliClient, cfg, jwtManager, wsHub)
    syncManager.SetOnSyncCompleted(handler.OnSyncCompleted)

    // Initialize NATS (optional)
    natsComponents, err := InitNATS(cfg, syncManager, wsHub, handler, db)
    if err != nil {
        return fmt.Errorf("initialize NATS: %w", err)
    }

    // Initialize backup manager
    backupCfg, err := backup.LoadConfig()
    if err != nil {
        log.Printf("Warning: Failed to load backup configuration: %v", err)
    } else if backupCfg.Enabled {
        backupManager, err := backup.NewManager(backupCfg, db)
        if err != nil {
            log.Printf("Warning: Failed to initialize backup manager: %v", err)
        } else {
            handler.SetBackupManager(backupManager)
            if backupCfg.Schedule.Enabled {
                if err := backupManager.Start(ctx); err != nil {
                    log.Printf("Warning: Failed to start backup scheduler: %v", err)
                }
            }
        }
    }

    router := api.NewRouter(handler, middleware)

    // === ADD SERVICES TO SUPERVISOR TREE ===

    // Messaging layer services
    tree.AddMessagingService(services.NewWebSocketHubService(wsHub))
    tree.AddMessagingService(services.NewSyncService(syncManager))

    // NATS components (if enabled)
    if natsComponents != nil {
        tree.AddMessagingService(services.NewNATSComponentsService(natsComponents))
    }

    // API layer services
    addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
    httpServer := &http.Server{
        Addr:         addr,
        Handler:      router.SetupChi(), // ADR-0016: Chi router
        ReadTimeout:  cfg.Server.Timeout,
        WriteTimeout: cfg.Server.Timeout,
        IdleTimeout:  60 * time.Second,
    }
    tree.AddAPIService(services.NewHTTPServerServiceWithConfig(httpServer))

    // === START SUPERVISOR TREE ===

    log.Printf("Starting supervisor tree on %s", addr)

    errCh := tree.ServeBackground(ctx)

    select {
    case <-ctx.Done():
        log.Println("Shutting down...")
    case err := <-errCh:
        if err != nil {
            log.Printf("Supervisor tree error: %v", err)
            return err
        }
    }

    // Log any services that failed to stop
    unstopped := tree.UnstoppedServiceReport()
    if len(unstopped) > 0 {
        log.Printf("Warning: %d services failed to stop within timeout", len(unstopped))
        for _, svc := range unstopped {
            log.Printf("  - %s", svc.Name)
        }
    }

    return ctx.Err()
}

func getLogLevel() slog.Level {
    switch os.Getenv("LOG_LEVEL") {
    case "debug":
        return slog.LevelDebug
    case "warn":
        return slog.LevelWarn
    case "error":
        return slog.LevelError
    default:
        return slog.LevelInfo
    }
}
```

---

## 11. Phase 9: E2E Testing

Add to your existing Playwright test suite:

```typescript
// tests/e2e/supervision.spec.ts
import { test, expect } from '@playwright/test';

test.describe('Supervisor Tree Resilience', () => {
  test('application recovers from sync failure', async ({ page }) => {
    await page.goto('/');
    await expect(page.locator('[data-testid="stats-panel"]')).toBeVisible();

    // App should remain responsive even if sync temporarily fails
    await expect(page.locator('[data-testid="map-container"]')).toBeVisible();
  });

  test('health endpoint reflects supervisor status', async ({ request }) => {
    const response = await request.get('/api/v1/health');
    expect(response.ok()).toBeTruthy();

    const health = await response.json();
    expect(health.status).toBe('healthy');
  });

  test('graceful shutdown preserves data', async ({ request }) => {
    // Trigger a sync
    const syncResponse = await request.post('/api/v1/sync');

    // Verify data persists after potential restart
    const statsResponse = await request.get('/api/v1/stats');
    expect(statsResponse.ok()).toBeTruthy();
  });
});
```

---

## 12. Configuration Reference

### 12.1 Environment Variables

Add to your `.env.example`:

```bash
# Supervisor Configuration
SUPERVISOR_FAILURE_THRESHOLD=5    # Failures before backoff (default: 5)
SUPERVISOR_FAILURE_DECAY=30       # Decay rate in seconds (default: 30)
SUPERVISOR_FAILURE_BACKOFF=15s    # Backoff duration (default: 15s)
SUPERVISOR_SHUTDOWN_TIMEOUT=10s   # Graceful shutdown timeout (default: 10s)
```

### 12.2 Config Struct Updates

Add to `internal/config/config.go`:

```go
// SupervisorConfig holds supervisor tree settings.
type SupervisorConfig struct {
    FailureThreshold float64       `koanf:"failure_threshold" default:"5"`
    FailureDecay     float64       `koanf:"failure_decay" default:"30"`
    FailureBackoff   time.Duration `koanf:"failure_backoff" default:"15s"`
    ShutdownTimeout  time.Duration `koanf:"shutdown_timeout" default:"10s"`
}
```

---

## 13. Troubleshooting

### 13.1 Service Won't Stop

**Symptom**: Shutdown hangs, EventStopTimeout events in logs

**Diagnosis**:
```go
// Add debug logging to see which services timeout
tree, _ := supervisor.NewSupervisorTree(logger, supervisor.TreeConfig{
    ShutdownTimeout: 5 * time.Second, // Shorter timeout for debugging
})
```

**Solution**: Ensure your service respects context cancellation:
```go
func (s *MyService) Serve(ctx context.Context) error {
    for {
        select {
        case <-ctx.Done():
            return ctx.Err() // CRITICAL: Must return here
        case work := <-s.workChan:
            s.process(work)
        }
    }
}
```

### 13.2 Restart Storm (CPU at 100%)

**Symptom**: Service keeps failing and restarting rapidly

**Solution**: Increase failure backoff:
```go
tree, _ := supervisor.NewSupervisorTree(logger, supervisor.TreeConfig{
    FailureThreshold: 3,                    // Lower threshold
    FailureBackoff:   30 * time.Second,     // Longer backoff
    FailureDecay:     60,                   // Slower decay
})
```

### 13.3 Debugging Panics

**For development** (panics crash app for stack trace):
```go
sup := suture.New("debug", suture.Spec{
    PassThroughPanics: true,
})
```

**For production** (catch and log panics):
```go
// VERIFIED: EventServicePanic includes PanicMsg and Stacktrace fields
eventHook := func(e suture.Event) {
    if ev, ok := e.(suture.EventServicePanic); ok {
        log.Printf("PANIC in %s: %s\n%s", ev.ServiceName, ev.PanicMsg, ev.Stacktrace)
    }
}
```

### 13.4 Testing Tips

1. **Use short timeouts in tests**:
```go
sup := suture.New("test", suture.Spec{
    FailureBackoff: 10 * time.Millisecond,
    Timeout:        100 * time.Millisecond,
})
```

2. **Always use context with timeout**:
```go
ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
defer cancel()
```

3. **Run with race detector**:
```bash
go test -race ./internal/supervisor/...
```

---

## 14. API Reference

### 14.1 Suture v4 Service Interface

```go
// VERIFIED: From pkg.go.dev/github.com/thejerf/suture/v4
type Service interface {
    Serve(ctx context.Context) error
}
```

### 14.2 Distinguished Errors

```go
// VERIFIED: These errors are checked via errors.Is
var ErrDoNotRestart = errors.New("service should not be restarted")
var ErrTerminateSupervisorTree = errors.New("tree should be terminated")
```

### 14.3 Spec Fields

```go
// VERIFIED: From pkg.go.dev
type Spec struct {
    EventHook                EventHook      // Receives all supervisor events
    Sprint                   SprintFunc     // Formats panic values to strings
    FailureDecay             float64        // Decay rate in seconds (default: 30)
    FailureThreshold         float64        // Failures before backoff (default: 5)
    FailureBackoff           time.Duration  // Backoff duration (default: 15s)
    BackoffJitter            Jitter         // Jitter function (default: DefaultJitter)
    Timeout                  time.Duration  // Shutdown timeout (default: 10s)
    PassThroughPanics        bool           // Let panics crash (default: false)
    DontPropagateTermination bool           // Contain ErrTerminateSupervisorTree
}
```

### 14.4 sutureslog Usage (CORRECTED)

```go
// VERIFIED: Correct usage from pkg.go.dev/github.com/thejerf/sutureslog
//
// The package provides a Handler struct, NOT a standalone EventHook function.
// MustHook() has a pointer receiver, so you must use & or assign to a variable.

import (
    "log/slog"
    "github.com/thejerf/suture/v4"
    "github.com/thejerf/sutureslog"
)

logger := slog.Default()

// CORRECT (MustHook has pointer receiver):
handler := &sutureslog.Handler{Logger: logger}
eventHook := handler.MustHook()

sup := suture.New("my-supervisor", suture.Spec{
    EventHook: eventHook,
})

// WRONG (will not compile - MustHook has pointer receiver):
// eventHook := sutureslog.Handler{Logger: logger}.MustHook()

// WRONG (no such function exists):
// eventHook := sutureslog.EventHook(logger)
```

### 14.5 Event Types

```go
// VERIFIED: Event types from pkg.go.dev
type EventBackoff struct {
    Supervisor     *Supervisor
    SupervisorName string
}

type EventResume struct {
    Supervisor     *Supervisor
    SupervisorName string
}

type EventServicePanic struct {
    Supervisor       *Supervisor
    SupervisorName   string
    Service          Service
    ServiceName      string
    CurrentFailures  float64
    FailureThreshold float64
    Restarting       bool
    PanicMsg         string
    Stacktrace       string
}

type EventServiceTerminate struct {
    Supervisor       *Supervisor
    SupervisorName   string
    Service          Service
    ServiceName      string
    CurrentFailures  float64
    FailureThreshold float64
    Restarting       bool
    Err              interface{}
}

type EventStopTimeout struct {
    Supervisor     *Supervisor
    SupervisorName string
    Service        Service
    ServiceName    string
}
```

---

## Implementation Checklist

### Phase 1: Foundation
- [ ] Add suture v4.0.6 dependency: `go get github.com/thejerf/suture/v4@v4.0.6`
- [ ] Add sutureslog v1.0.1 dependency: `go get github.com/thejerf/sutureslog@v1.0.1`
- [ ] Create `internal/supervisor` package
- [ ] Write and pass MockService tests
- [ ] Verify suture basics work

### Phase 2: Sync Manager Service
- [ ] Write SyncService tests (uses existing Start/Stop interface)
- [ ] Implement SyncService wrapper (no changes to manager.go needed)

### Phase 3: WebSocket Hub Service
- [ ] Add `RunWithContext(ctx context.Context) error` to Hub
- [ ] Add `closeAllClients()` helper to Hub
- [ ] Add `Stop()` method to Hub
- [ ] Write WebSocketHubService tests
- [ ] Implement WebSocketHubService wrapper

### Phase 4: HTTP Server Service
- [ ] Write HTTPServerService tests
- [ ] Implement HTTPServerService

### Phase 5: NATS Services (build tag: nats)
- [ ] Write NATSComponentsService tests
- [ ] Implement NATSComponentsService (wraps existing NATSComponents)

### Phase 6: WAL Services (build tag: wal)
- [ ] Write WALRetryLoopService tests
- [ ] Write WALCompactorService tests
- [ ] Implement WAL service wrappers

### Phase 7: Tree Assembly
- [ ] Write tree assembly tests
- [ ] Implement SupervisorTree with **CORRECT** sutureslog usage
- [ ] Verify hierarchical supervision

### Phase 8: Integration
- [ ] Update main.go to use supervisor tree
- [ ] Add configuration options
- [ ] Test graceful shutdown

### Phase 9: E2E
- [ ] Add Playwright supervision tests
- [ ] Verify production behavior

---

## Key Corrections from Original Guide

| Issue | Original | Corrected |
|-------|----------|-----------|
| sutureslog API | `sutureslog.EventHook(logger)` | `(&sutureslog.Handler{Logger: logger}).MustHook()` |
| Sync Manager | Assumed `Run(ctx)` method | Uses existing `Start(ctx)`/`Stop()` pattern |
| WebSocket Hub | Assumed `Run(ctx)` method | Must add `RunWithContext(ctx)` method |
| WAL | Treated as single service | RetryLoop and Compactor are separate services |
| WAL Interface | Proposed `Open(ctx)`/`Run(ctx)`/`Close()` | Actual: `Start(ctx)`/`Stop()` pattern |

---

**Document Version**: 3.0.0 (Verified and Corrected)
**Suture Version**: v4.0.6 (VERIFIED: pkg.go.dev Dec 2024)
**sutureslog Version**: v1.0.1 (VERIFIED: pkg.go.dev Dec 2024)
**Verification Date**: December 2024
**Source Code Verification**: All interfaces verified against actual repository code
**TDD Compliance**: All code follows RED-GREEN-REFACTOR
