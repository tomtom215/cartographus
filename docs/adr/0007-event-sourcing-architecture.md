# ADR-0007: Event Sourcing with NATS-First Architecture

**Date**: 2025-12-02
**Status**: Accepted

---

## Context

With NATS JetStream (ADR-0005) and BadgerDB WAL (ADR-0006) in place, Cartographus can support an event sourcing architecture where:

1. **Events are the source of truth** (not database state)
2. **Materialized views** (DuckDB) derived from event stream
3. **Replay capability** for reprocessing or migration
4. **Temporal queries** possible from event history

### Traditional vs Event Sourcing

| Aspect | Traditional | Event Sourcing |
|--------|-------------|----------------|
| **Source of Truth** | Database state | Event log |
| **History** | Lost on update | Preserved forever |
| **Replay** | Not possible | Full capability |
| **Complexity** | Lower | Higher |
| **Storage** | Less | More |

### Use Cases

- **Multi-Device Aggregation**: Combine events from multiple Plex servers
- **Historical Analysis**: Reconstruct past state at any point
- **Migration**: Rebuild database from events
- **Debugging**: Trace exact sequence of changes

---

## Decision

Implement an **optional event sourcing mode** (NATS-First architecture):

- **Default Mode**: Event sourcing enabled (`NATS_EVENT_SOURCING=true`)
- **Legacy Mode**: Direct database writes with NATS for notifications only

### Architecture

```
                    EVENT SOURCING MODE
                    ═══════════════════

┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│  Tautulli   │     │    Plex     │     │  Jellyfin   │
│   Sync      │     │  Webhook    │     │  WebSocket  │
└──────┬──────┘     └──────┬──────┘     └──────┬──────┘
       │                   │                   │
       └───────────────────┴───────────────────┘
                           │
                           ▼
              ┌───────────────────────┐
              │    NATS JetStream     │ ◄─── Source of Truth
              │    (Event Store)      │
              │                       │
              │  Stream: MEDIA_EVENTS │
              │  Retention: 7 days    │
              └───────────┬───────────┘
                          │
                          ▼
              ┌───────────────────────┐
              │   Watermill Router    │
              │   (Middleware Stack)  │
              │   - Retry             │
              │   - Poison Queue      │
              │   - Deduplication     │
              └───────────┬───────────┘
                          │
         ┌────────────────┼────────────────┐
         ▼                ▼                ▼
┌─────────────┐   ┌─────────────┐   ┌─────────────┐
│ WebSocket   │   │  DuckDB     │   │  Detection  │
│  Handler    │   │  Handler    │   │  Handler    │
│ (Broadcast) │   │ (Appender)  │   │ (Anomaly)   │
└─────────────┘   └──────┬──────┘   └─────────────┘
                         │
                         ▼
                ┌─────────────────┐
                │    DuckDB       │ ◄─── Materialized View
                │  (Analytics)    │
                └─────────────────┘
```

### Key Factors

1. **Default Enabled**: Event sourcing is the default mode (`NATS_EVENT_SOURCING=true`)
2. **Gradual Adoption**: Users can opt-out to legacy mode if needed
3. **Event Immutability**: Events never modified, only appended
4. **Cross-Source Deduplication**: CorrelationKey prevents duplicate processing across sources
5. **Replayability**: Full event history available via JetStream

---

## Consequences

### Positive

- **Complete History**: All events preserved for analysis
- **Replay Capability**: Rebuild state from any point
- **Auditability**: Full trace of every change via dedupe audit log (ADR-0022)
- **Multi-Source Integration**: Events from Plex, Tautulli, Jellyfin, Emby unified
- **Future-Proof**: Easy to add new projections via Watermill handlers

### Negative

- **Storage Growth**: Events accumulate over time (mitigated by 7-day retention)
- **Eventual Consistency**: DuckDB may lag behind events (5-second flush interval)
- **Complexity**: Router-based middleware stack to understand
- **Operational Overhead**: Need to manage JetStream and WAL

### Neutral

- **Build Tags**: Requires `nats` and `wal` build tags for full functionality
- **Embedded NATS**: Default uses embedded server for simplicity

---

## Implementation

### Mode Detection

```go
// internal/config/config.go
type NATSConfig struct {
    // Enabled controls whether event processing is active.
    Enabled bool `koanf:"enabled"`

    // EventSourcing enables full event-sourcing mode where NATS JetStream
    // is the single source of truth.
    EventSourcing bool `koanf:"event_sourcing"`

    // Other fields...
}
```

Usage in sync manager:
```go
// internal/sync/tautulli_sync.go
eventSourcingMode := m.cfg.NATS.Enabled && m.cfg.NATS.EventSourcing
```

### Event Store Interface

```go
// internal/eventprocessor/appender.go
type EventStore interface {
    // InsertMediaEvents inserts a batch of media events.
    InsertMediaEvents(ctx context.Context, events []*MediaEvent) error
}
```

### DuckDB Handler (Projection)

```go
// internal/eventprocessor/handlers.go
type DuckDBHandler struct {
    appender   *Appender
    config     DuckDBHandlerConfig
    dedupCache cache.DeduplicationCache  // ExactLRU for zero false positives
    // ...
}

func (h *DuckDBHandler) Handle(msg *message.Message) error {
    // Deserialize, deduplicate, append to buffer
    // Returns nil on success, error triggers retry middleware
}
```

### Cross-Source Deduplication

```go
// internal/eventprocessor/events.go
// GenerateCorrelationKey creates a correlation key for cross-source deduplication.
// Format: {source}:{server_id}:{user_id}:{rating_key}:{machine_id}:{time_bucket}:{session_key}
func (e *MediaEvent) GenerateCorrelationKey() string {
    timeBucket := e.StartedAt.UTC().Format("2006-01-02T15:04:05")
    return formatCorrelationKey(source, serverID, e.UserID, ratingKey, machineID, timeBucket, sessionKey)
}
```

### Deduplication Strategy

The system uses three-tier deduplication:
1. **EventID**: Exact match prevents reprocessing same event
2. **SessionKey**: Source-specific session identifier
3. **CorrelationKey**: Cross-source matching (Plex + Tautulli for same playback)

```go
// internal/eventprocessor/handlers.go
func (h *DuckDBHandler) isDuplicateWithAudit(event *MediaEvent, rawPayload []byte) bool {
    // Check EventID (primary dedup key)
    if h.dedupCache.IsDuplicate(event.EventID) { return true }

    // Check SessionKey
    if h.dedupCache.Contains(event.SessionKey) { return true }

    // Check CorrelationKey for cross-source dedup
    crossSourceKey := GetCrossSourceKey(event.CorrelationKey)
    // ... check against other sources
}
```

### Database Deduplication

```sql
-- DuckDB upsert with correlation key for idempotent processing
INSERT INTO playback_events (...)
VALUES (?, ?, ?, ...)
ON CONFLICT (correlation_key) DO NOTHING
```

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `NATS_ENABLED` | `false` | Enable NATS event processing |
| `NATS_EVENT_SOURCING` | `true` | Enable NATS-first architecture |
| `NATS_RETENTION_DAYS` | `7` | JetStream event retention period |
| `NATS_BATCH_SIZE` | `1000` | Events to batch before DuckDB write |
| `NATS_FLUSH_INTERVAL` | `5s` | Maximum time between DuckDB flushes |
| `NATS_EMBEDDED` | `true` | Use embedded NATS server |
| `NATS_STORE_DIR` | `/data/nats/jetstream` | JetStream storage directory |

### Code References

| Component | File | Notes |
|-----------|------|-------|
| MediaEvent type | `internal/eventprocessor/events.go` | Canonical event format |
| DuckDB handler | `internal/eventprocessor/handlers.go` | DuckDBHandler with dedup |
| Appender | `internal/eventprocessor/appender.go` | Batch buffering to EventStore |
| DuckDB store | `internal/eventprocessor/duckdb_store.go` | EventStore implementation |
| Router | `internal/eventprocessor/router.go` | Watermill router with middleware |
| NATS config | `internal/config/config.go` | NATSConfig.EventSourcing field |
| NATS init | `cmd/server/nats_init.go` | Component initialization |

---

## Verification

### Verified Claims

| Claim | Source | Verified |
|-------|--------|----------|
| Event sourcing mode config | `internal/config/config.go:420-430` | Yes (`NATSConfig.EventSourcing`) |
| Cross-source deduplication | `internal/eventprocessor/handlers.go` | Yes (CorrelationKey + ExactLRU) |
| Idempotent batch insert | `internal/eventprocessor/duckdb_store.go` | Yes (InsertMediaEvents) |
| Watermill Router middleware | `internal/eventprocessor/router.go` | Yes (Retry, PoisonQueue) |
| DuckDB handler | `internal/eventprocessor/handlers.go` | Yes (DuckDBHandler) |

### Test Coverage

- Event processing tests: `internal/eventprocessor/*_test.go`
- Run with: `go test -tags "nats,wal" -race ./internal/eventprocessor/...`
- Coverage target: 80%+ for eventprocessor package

---

## Related ADRs

- [ADR-0005](0005-nats-jetstream-event-processing.md): NATS JetStream foundation
- [ADR-0006](0006-badgerdb-write-ahead-log.md): WAL for event durability
- [ADR-0009](0009-plex-direct-integration.md): Plex webhook events
- [ADR-0017](0017-watermill-router-and-middleware.md): Watermill router architecture
- [ADR-0022](0022-dedupe-audit-management.md): Deduplication audit logging
- [ADR-0023](0023-consumer-wal-exactly-once.md): Exactly-once delivery

---

## References

- [Event Sourcing Pattern](https://martinfowler.com/eaaDev/EventSourcing.html)
- [CQRS Pattern](https://martinfowler.com/bliki/CQRS.html)
- [NATS JetStream Documentation](https://docs.nats.io/nats-concepts/jetstream)
- [Watermill Documentation](https://watermill.io/)
