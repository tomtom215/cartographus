# ADR-0017: Watermill Router and Middleware Adoption

**Date**: 2025-12-12
**Status**: Accepted

---

## Context

The existing Watermill implementation (ADR-0005) used only basic Publisher/Subscriber interfaces with manual message handling loops. While functional, this approach:

1. Required manual Ack/Nack handling in every handler
2. Duplicated retry logic across handlers
3. Implemented custom DLQ handling instead of using Watermill's PoisonQueue
4. Wrapped circuit breaker externally instead of using middleware
5. Lacked centralized panic recovery
6. Had no rate limiting capability

Watermill v1.5.1 provides advanced features that were not being utilized:

- **Router**: High-level message handling with automatic Ack/Nack
- **Middleware**: Composable cross-cutting concerns
- **CQRS Component**: Type-safe event/command handling
- **Forwarder**: Transactional outbox pattern

### Alternatives Considered

| Approach | Pros | Cons |
|----------|------|------|
| **Keep Manual Loops** | Simple, working | Duplicated code, no middleware |
| **Adopt Router Only** | Better structure | Still custom error handling |
| **Full Watermill Stack** | All features, less code | Migration effort |
| **Red Panda Connect** | Declarative pipelines | Different paradigm, operational overhead |

---

## Decision

Adopt **Watermill Router with full middleware stack** for event processing:

1. **Router**: Replace manual subscribe loops with Router-based handlers
2. **Middleware Stack**: Use built-in middleware for cross-cutting concerns
3. **CQRS Component**: Type-safe event publishing and handling
4. **Forwarder**: Optional transactional outbox for critical paths

### Architecture

```
                    Watermill Router Architecture
                    ═══════════════════════════════

┌─────────────────────────────────────────────────────────────────┐
│                         Router                                   │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │                    Middleware Stack                         ││
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────────┐   ││
│  │  │Recoverer │→│  Retry   │→│ Throttle │→│ PoisonQueue  │   ││
│  │  │ (panics) │ │(backoff) │ │ (rate)   │ │   (DLQ)      │   ││
│  │  └──────────┘ └──────────┘ └──────────┘ └──────────────┘   ││
│  └─────────────────────────────────────────────────────────────┘│
│                              │                                   │
│         ┌────────────────────┴────────────────────┐             │
│         ▼                                         ▼             │
│  ┌──────────────┐                         ┌──────────────┐      │
│  │ DuckDB       │                         │ WebSocket    │      │
│  │ Handler      │                         │ Handler      │      │
│  │ (persistence)│                         │ (real-time)  │      │
│  └──────────────┘                         └──────────────┘      │
└─────────────────────────────────────────────────────────────────┘
```

### Middleware Stack (Applied in Order)

| Middleware | Purpose | Configuration |
|------------|---------|---------------|
| **Recoverer** | Catch panics, convert to errors | Always enabled |
| **Retry** | Exponential backoff on errors | 5 retries, 1s-1m backoff |
| **Throttle** | Rate limiting | Optional, disabled by default |
| **Deduplicator** | Simple message ID dedup | Optional, for exact matches |
| **PoisonQueue** | Route failed messages to DLQ | Routes to `dlq.playback` |

### Key Components

**Router** (`router.go`):
- Wraps Watermill Router with pre-configured middleware
- Automatic Ack/Nack based on handler return value
- Graceful shutdown with configurable timeout

**Handlers** (`handlers.go`):
- `DuckDBHandler`: Persistence with cross-source deduplication
- `WebSocketHandler`: Real-time broadcasting

**CQRS** (`cqrs.go`):
- `EventBus`: Type-safe event publishing
- `MediaEventHandler`: Type-safe event handling
- `EventHandlerGroup`: Multiple handlers per event type

**Forwarder** (`forwarder.go`):
- `TransactionalPublisher`: Writes to outbox store
- `Forwarder`: Polls and forwards to NATS
- `OutboxStore`: Interface for persistent storage

---

## Consequences

### Positive

- **Less Code**: Removed ~400 lines of custom retry/DLQ/error handling
- **Consistent Error Handling**: All handlers use same middleware stack
- **Type Safety**: CQRS prevents runtime type errors
- **Panic Safety**: Recoverer prevents handler crashes from affecting system
- **Rate Limiting**: Can throttle high-volume streams
- **Transactional Safety**: Forwarder ensures no message loss

### Negative

- **Learning Curve**: Team needs to understand Router/middleware patterns
- **Migration Effort**: Existing code needs updating (backward compatible)
- **Slight Overhead**: Middleware chain adds minimal latency

### Neutral

- **Deprecated Legacy**: DuckDBConsumer is deprecated but still functional
- **Optional Features**: Forwarder/CQRS are opt-in
- **Build Tags**: Still requires `-tags nats`

---

## Implementation

### File Structure

```
internal/eventprocessor/
├── router.go           # Watermill Router wrapper
├── router_stub.go      # Non-NATS stub
├── router_test.go      # Router tests
├── handlers.go         # DuckDB and WebSocket handlers
├── handlers_stub.go    # Non-NATS stub
├── handlers_test.go    # Handler tests
├── cqrs.go             # CQRS components
├── cqrs_stub.go        # Non-NATS stub
├── cqrs_test.go        # CQRS tests
├── forwarder.go        # Transactional outbox
├── forwarder_stub.go   # Non-NATS stub
├── forwarder_test.go   # Forwarder tests
├── router_init.go      # Initialization helpers
└── router_init_stub.go # Non-NATS stub
```

### Usage Example

```go
// Create components
cfg := eventprocessor.DefaultRouterComponentsConfig()
cfg.WebSocketBroadcaster = wsHub

components, err := eventprocessor.NewRouterComponents(
    &cfg, // Pass pointer to config
    appender,
    publisher,
    duckdbSubscriber,
    websocketSubscriber,
    logger,
)
if err != nil {
    return err
}

// Start processing
ctx := context.Background()
if err := components.Start(ctx); err != nil {
    return err
}

// Cleanup
defer components.Stop()
```

### Migration Path

The migration has been completed. The Router-based approach is now the default in `cmd/server/nats_init.go`.

**Phase 1** (Completed): Added Router-based components alongside existing code
**Phase 2** (Completed): Migrated consumers to Router-based handlers in nats_init.go
**Phase 3** (Completed): Legacy DuckDBConsumer marked as deprecated

The `DuckDBConsumer` type is now deprecated and will be removed in a future release.
New code should use `Router.AddConsumerHandler` with `DuckDBHandler` instead.

---

## Verification

### Verified Claims

| Claim | Source | Verified |
|-------|--------|----------|
| Watermill v1.5.1 | `go.mod:9` | Yes |
| Router auto Ack/Nack | `router.go` | Yes |
| Middleware chain | `router.go:148-193` | Yes |
| CQRS type safety | `cqrs.go` | Yes |
| Forwarder outbox | `forwarder.go` | Yes |

### Test Coverage

- `router_test.go`: Router configuration and lifecycle
- `handlers_test.go`: Handler message processing
- `cqrs_test.go`: CQRS marshaling and handlers
- `forwarder_test.go`: Outbox store and forwarding

Run tests:
```bash
go test -tags nats -v ./internal/eventprocessor/...
```

---

## Related ADRs

- [ADR-0005](0005-nats-jetstream-event-processing.md): NATS JetStream foundation
- [ADR-0006](0006-badgerdb-write-ahead-log.md): WAL for event durability
- [ADR-0007](0007-event-sourcing-architecture.md): Event sourcing mode

---

## References

- [Watermill Router Documentation](https://watermill.io/docs/messages-router/)
- [Watermill Middleware Documentation](https://watermill.io/docs/middlewares/)
- [Watermill CQRS Documentation](https://watermill.io/docs/cqrs/)
- [Watermill Forwarder Documentation](https://watermill.io/docs/forwarder/)
