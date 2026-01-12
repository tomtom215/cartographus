# ADR-0004: Process Supervision with Suture v4

**Date**: 2025-12-03
**Status**: Accepted

---

## Context

Cartographus operates as a long-running service with multiple concurrent components:

1. **HTTP Server**: API endpoints and static file serving
2. **WebSocket Hub**: Real-time client connections
3. **Sync Manager**: Periodic Tautulli synchronization
4. **NATS Components**: Event processing (optional)
5. **WAL Services**: Write-ahead log retry and compaction (optional)

### Requirements

- Automatic restart of crashed components
- Graceful shutdown with timeout
- Hierarchical supervision (parent controls children)
- Integration with Go's `context.Context` for cancellation
- Structured logging of supervisor events

### Alternatives Considered

| Approach | Pros | Cons |
|----------|------|------|
| **Manual goroutine management** | Simple, no dependencies | Error-prone, no restart logic |
| **oklog/run** | Simple group management | No hierarchical supervision |
| **errgroup** | stdlib, simple | No restart, no hierarchy |
| **Suture v4** | Full OTP-style supervision | Additional dependency |

---

## Decision

Use **Suture v4** for Erlang/OTP-style process supervision with a three-layer supervisor tree.

### Architecture

```
                    RootSupervisor
                   "cartographus"
                          │
        ┌─────────────────┼─────────────────┐
        │                 │                 │
        ▼                 ▼                 ▼
  DataSupervisor   MessagingSupervisor   APISupervisor
  "data-layer"     "messaging-layer"     "api-layer"
        │                 │                 │
        │           ┌─────┼─────┐           │
        ▼           ▼     ▼     ▼           ▼
   (Future)    WebSocket Sync  NATS      HTTP
   Services    Hub     Mgr  Components  Server
```

### Key Factors

1. **Proven Pattern**: Erlang OTP supervision has 30+ years of production use
2. **Graceful Degradation**: Failed services restart without affecting others
3. **Context Integration**: Native `context.Context` support in v4
4. **Structured Logging**: `sutureslog` adapter for `slog` integration
5. **Go Idiomatic**: Clean Go API, not a port of Erlang syntax

---

## Consequences

### Positive

- **Automatic Recovery**: Crashed services restart with exponential backoff
- **Graceful Shutdown**: 10-second timeout for clean service termination
- **Hierarchical Control**: Parent supervisors control child lifecycles
- **Event Visibility**: Structured logging of all supervisor events
- **Isolated Failures**: Service crash doesn't bring down entire application

### Negative

- **Additional Dependency**: `github.com/thejerf/suture/v4` added to go.mod
- **Service Interface Adaptation**: Existing components need `Serve(ctx)` wrappers
- **Learning Curve**: Understanding supervision tree behavior

### Neutral

- **Configuration Tuning**: Failure thresholds need production tuning
- **Test Complexity**: Supervisor behavior adds test scenarios

---

## Implementation

### Supervisor Tree Configuration

```go
// internal/supervisor/tree.go
type TreeConfig struct {
    FailureThreshold float64       // Failures before backoff (default: 5)
    FailureDecay     float64       // Decay rate in seconds (default: 30)
    FailureBackoff   time.Duration // Backoff duration (default: 15s)
    ShutdownTimeout  time.Duration // Graceful shutdown timeout (default: 10s)
}

func NewSupervisorTree(logger *slog.Logger, config TreeConfig) (*SupervisorTree, error) {
    // Create slog event hook
    handler := &sutureslog.Handler{Logger: logger}
    eventHook := handler.MustHook()

    rootSpec := suture.Spec{
        EventHook:        eventHook,
        FailureThreshold: config.FailureThreshold,
        FailureDecay:     config.FailureDecay,
        FailureBackoff:   config.FailureBackoff,
        Timeout:          config.ShutdownTimeout,
    }

    root := suture.New("cartographus", rootSpec)
    data := suture.New("data-layer", childSpec)
    messaging := suture.New("messaging-layer", childSpec)
    api := suture.New("api-layer", childSpec)

    // Build hierarchy
    root.Add(data)
    root.Add(messaging)
    root.Add(api)

    return &SupervisorTree{root, data, messaging, api, logger, config}, nil
}
```

### Service Wrapper Pattern

```go
// internal/supervisor/services/sync_service.go
type SyncService struct {
    manager StartStopManager
    name    string
}

// Serve implements suture.Service
func (s *SyncService) Serve(ctx context.Context) error {
    // Start the manager
    if err := s.manager.Start(ctx); err != nil {
        return fmt.Errorf("sync manager start failed: %w", err)
    }

    // Wait for context cancellation
    <-ctx.Done()

    // Stop the manager
    if err := s.manager.Stop(); err != nil {
        return fmt.Errorf("sync manager stop failed: %w", err)
    }

    return ctx.Err()
}

// String implements fmt.Stringer for logging
func (s *SyncService) String() string {
    return s.name
}
```

### Main Integration

```go
// cmd/server/main.go
func runApp(ctx context.Context, logger *slog.Logger) error {
    // Create supervisor tree
    tree, err := supervisor.NewSupervisorTree(logger, supervisor.TreeConfig{
        FailureThreshold: 5,
        FailureBackoff:   15 * time.Second,
        ShutdownTimeout:  10 * time.Second,
    })

    // Add services to appropriate layers
    tree.AddMessagingService(services.NewWebSocketHubService(wsHub))
    tree.AddMessagingService(services.NewSyncService(syncManager))
    tree.AddAPIService(services.NewHTTPServerService(httpServer, 10*time.Second))

    // Start supervisor tree
    errCh := tree.ServeBackground(ctx)

    select {
    case <-ctx.Done():
        log.Println("Shutting down...")
    case err := <-errCh:
        return err
    }

    return nil
}
```

### Service Interfaces

```go
// suture.Service interface (from pkg.go.dev)
type Service interface {
    Serve(ctx context.Context) error
}

// StartStopManager for adapting existing components
type StartStopManager interface {
    Start(ctx context.Context) error
    Stop() error
}

// ContextHub for WebSocket hub
type ContextHub interface {
    RunWithContext(ctx context.Context) error
}
```

### Code References

| Component | File | Notes |
|-----------|------|-------|
| Supervisor tree | `internal/supervisor/tree.go` | 3-layer hierarchy |
| Service interface | `internal/supervisor/mock_service.go` | Test implementation |
| Sync service wrapper | `internal/supervisor/services/sync_service.go` | Adapts Start/Stop to Serve |
| WebSocket service | `internal/supervisor/services/websocket_service.go` | Hub wrapper |
| HTTP service | `internal/supervisor/services/http_service.go` | Server wrapper |
| NATS service | `internal/supervisor/services/nats_service.go` | Build tag: nats |
| WAL services | `internal/supervisor/services/wal_service.go` | Build tag: wal |

---

## Verification

### Verified Claims

| Claim | Source | Verified |
|-------|--------|----------|
| Suture v4.0.6 | `go.mod:38` | Yes |
| sutureslog v1.0.1 | `go.mod:39` | Yes |
| 3-layer supervisor tree | `internal/supervisor/tree.go` | Yes |
| 82 tests in supervisor package | `internal/supervisor/*_test.go` | Yes |
| Race-safe implementation | Tests pass with `-race` flag | Yes |

### Test Coverage

- Supervisor tests: `internal/supervisor/*_test.go`
- Service tests: `internal/supervisor/services/*_test.go`
- 82 tests with race detection
- Coverage target: 80%+ for supervisor package

---

## Related ADRs

- [ADR-0005](0005-nats-jetstream-event-processing.md): NATS service supervised by messaging layer
- [ADR-0006](0006-badgerdb-write-ahead-log.md): WAL services supervised by data layer

---

## References

- [Suture v4 Documentation](https://pkg.go.dev/github.com/thejerf/suture/v4)
- [sutureslog Documentation](https://pkg.go.dev/github.com/thejerf/sutureslog)
- [Erlang OTP Supervision Principles](https://www.erlang.org/doc/design_principles/sup_princ.html)
- [SUTURE_IMPLEMENTATION_GUIDE.md](../SUTURE_IMPLEMENTATION_GUIDE.md)
