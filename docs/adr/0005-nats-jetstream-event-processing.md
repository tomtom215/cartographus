# ADR-0005: NATS JetStream Event Processing

**Date**: 2025-12-01
**Status**: Accepted

---

## Context

Cartographus needs to process Plex media playback events from multiple sources:

1. **Tautulli Webhooks**: Primary event source from existing Tautulli installations
2. **Plex Webhooks**: Direct Plex server webhooks for standalone operation
3. **Manual Sync**: Periodic polling of Tautulli API
4. **Multi-Device Support**: Events from multiple Plex servers

### Requirements

- Event ordering guarantees (at-least-once delivery)
- Durable message storage for replay
- Dead letter queue for failed processing
- Backpressure handling for burst traffic
- Decoupled producers and consumers

### Alternatives Considered

| System | Pros | Cons |
|--------|------|------|
| **Direct Processing** | Simple, no infrastructure | No replay, no ordering |
| **Redis Streams** | Fast, mature | External dependency |
| **Apache Kafka** | Industry standard | Heavy, operational overhead |
| **NATS JetStream** | Embedded, lightweight | Newer than Kafka |
| **Watermill** | Go-native abstraction | Additional layer |

---

## Decision

Use **NATS JetStream** with **Watermill** for event processing:

- **NATS JetStream**: Embedded message broker with persistence
- **Watermill**: Go-native event processing framework with NATS adapter
- **Dual-Path Consumption**: WebSocket (real-time) + DuckDB (persistence)

### Architecture

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   Tautulli  │     │    Plex     │     │   Manual    │
│   Webhook   │     │   Webhook   │     │    Sync     │
└──────┬──────┘     └──────┬──────┘     └──────┬──────┘
       │                   │                   │
       └───────────────────┴───────────────────┘
                           │
                           ▼
                 ┌───────────────────┐
                 │  NATS JetStream   │
                 │   (Embedded)      │
                 │                   │
                 │ Stream:MEDIA_EVENTS│
                 │ Subject: playback.>│
                 └─────────┬─────────┘
                           │
            ┌──────────────┴──────────────┐
            ▼                             ▼
   ┌─────────────────┐           ┌─────────────────┐
   │ WebSocket       │           │ DuckDB          │
   │ Consumer        │           │ Consumer        │
   │ (Real-time)     │           │ (Persistence)   │
   └─────────────────┘           └─────────────────┘
```

### Key Factors

1. **Embedded Operation**: NATS server runs in-process, no external dependencies
2. **At-Least-Once Delivery**: JetStream guarantees message persistence
3. **Dual Consumers**: Real-time WebSocket + durable DuckDB storage
4. **Session Key Deduplication**: Prevent duplicate events from multiple sources
5. **Build Tag Isolation**: Optional feature via `nats` build tag

---

## Consequences

### Positive

- **Event Replay**: Historical events can be replayed from JetStream
- **Decoupled Processing**: Producers and consumers operate independently
- **Real-Time Updates**: WebSocket consumers receive immediate updates
- **Embedded Simplicity**: No external NATS server required
- **Graceful Degradation**: Application works without NATS (traditional sync)

### Negative

- **Memory Usage**: Embedded NATS server adds memory overhead
- **Complexity**: Additional abstraction layer (Watermill)
- **Build Configuration**: Requires `nats` build tag for full features

### Neutral

- **Optional Feature**: Can be disabled via `NATS_ENABLED=false`
- **Learning Curve**: Team needs to understand event-driven patterns

---

## Implementation

### Stream Configuration

```go
// internal/eventprocessor/config.go
func DefaultStreamConfig() StreamConfig {
    return StreamConfig{
        Name: "MEDIA_EVENTS",
        Subjects: []string{
            "playback.>",
            "plex.>",
            "jellyfin.>",
            "tautulli.>",
        },
        MaxAge:          7 * 24 * time.Hour,      // 7 days retention
        MaxBytes:        10 * 1024 * 1024 * 1024, // 10GB max
        MaxMsgs:         -1,                      // Unlimited
        DuplicateWindow: 2 * time.Minute,
        Replicas:        1,                       // Single node
    }
}
```

### Watermill Integration

```go
// internal/eventprocessor/publisher.go
func NewPublisher(cfg PublisherConfig, logger watermill.LoggerAdapter) (*Publisher, error) {
    // Watermill publisher configuration
    wmConfig := wmNats.PublisherConfig{
        URL:         cfg.URL,
        NatsOptions: natsOpts,
        Marshaler:   &wmNats.NATSMarshaler{},
        JetStream: wmNats.JetStreamConfig{
            Disabled:      false,
            AutoProvision: false,                // Stream is pre-created by StreamInitializer
            TrackMsgId:    cfg.EnableTrackMsgID, // Enable deduplication
            PublishOptions: []natsgo.PubOpt{
                natsgo.RetryAttempts(3),
                natsgo.RetryWait(100 * time.Millisecond),
            },
        },
    }

    pub, err := wmNats.NewPublisher(wmConfig, logger)
    // ...
}
```

### Event Types

```go
// internal/eventprocessor/events.go
type MediaEvent struct {
    EventID        string    `json:"event_id"`
    SessionKey     string    `json:"session_key,omitempty"`
    CorrelationKey string    `json:"correlation_key,omitempty"`
    Source         string    `json:"source"`        // plex, jellyfin, tautulli, emby
    ServerID       string    `json:"server_id,omitempty"`
    Timestamp      time.Time `json:"timestamp"`
    UserID         int       `json:"user_id"`
    Username       string    `json:"username"`
    Title          string    `json:"title"`
    MediaType      string    `json:"media_type"`   // movie, episode, track
    // ... additional fields
}

// Event type constants
const (
    EventTypePlaybackStart    = "start"
    EventTypePlaybackStop     = "stop"
    EventTypePlaybackPause    = "pause"
    EventTypePlaybackResume   = "resume"
    EventTypePlaybackProgress = "progress"
)

// Topic() returns the NATS subject: playback.<source>.<media_type>
```

### Dual-Path Consumption

The Router-based approach uses separate handlers for each consumption path:

```go
// internal/eventprocessor/handlers.go

// DuckDBHandler - persistence path with cross-source deduplication
type DuckDBHandler struct {
    appender   *Appender
    config     DuckDBHandlerConfig
    dedupCache cache.DeduplicationCache
    // ...
}

func (h *DuckDBHandler) Handle(msg *message.Message) error {
    var event MediaEvent
    if err := json.Unmarshal(msg.Payload, &event); err != nil {
        return NewPermanentError("JSON parse error", err)
    }

    // Cross-source deduplication (EventID, SessionKey, CorrelationKey)
    if h.config.EnableCrossSourceDedup && h.isDuplicateWithAudit(&event, msg.Payload) {
        return nil // Acknowledge duplicate
    }

    // Append to buffer for batch write
    if err := h.appender.Append(ctx, &event); err != nil {
        return NewRetryableError("append failed", err)
    }
    return nil
}

// WebSocketHandler - real-time broadcast path
type WebSocketHandler struct {
    hub WebSocketBroadcaster
    // ...
}

func (h *WebSocketHandler) Handle(msg *message.Message) error {
    h.hub.BroadcastRaw(msg.Payload)
    return nil
}
```

### Dead Letter Queue

```go
// internal/eventprocessor/dlq.go
type DLQHandler struct {
    config  DLQConfig
    entries *cache.MinHeap[*DLQEntry]
    // ...
}

func DefaultDLQConfig() DLQConfig {
    return DLQConfig{
        MaxRetries:        5,
        MaxEntries:        10000,
        RetentionTime:     7 * 24 * time.Hour, // 7 days
        InitialBackoff:    time.Second,
        MaxBackoff:        time.Minute,
        BackoffMultiplier: 2.0,
        JitterFraction:    0.1,
    }
}
```

The Router also supports poison queue routing via Watermill middleware.

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `NATS_ENABLED` | `false` | Enable NATS JetStream |
| `NATS_URL` | `nats://127.0.0.1:4222` | NATS server URL |
| `NATS_EMBEDDED` | `true` | Use embedded NATS server |
| `NATS_STORE_DIR` | `/data/nats/jetstream` | JetStream data directory |
| `NATS_RETENTION_DAYS` | `7` | Stream retention period |
| `NATS_BATCH_SIZE` | `1000` | Events to batch before DuckDB write |
| `NATS_FLUSH_INTERVAL` | `5s` | Maximum time between flushes |

### Code References

| Component | File | Notes |
|-----------|------|-------|
| NATS initialization | `cmd/server/nats_init.go` | Build tag: nats |
| Stream configuration | `internal/eventprocessor/config.go` | StreamConfig defaults |
| Stream initializer | `internal/eventprocessor/stream_init.go` | JetStream setup |
| Publisher | `internal/eventprocessor/publisher.go` | Watermill adapter |
| DuckDB handler | `internal/eventprocessor/handlers.go` | Persistence path (DuckDBHandler) |
| WebSocket handler | `internal/eventprocessor/handlers.go` | Real-time path (WebSocketHandler) |
| Router | `internal/eventprocessor/router.go` | Watermill Router wrapper |
| Events | `internal/eventprocessor/events.go` | MediaEvent struct |
| Dead Letter Queue | `internal/eventprocessor/dlq.go` | DLQHandler |

---

## Verification

### Verified Claims

| Claim | Source | Verified |
|-------|--------|----------|
| Watermill v1.5.1 | `go.mod` line 9 | Yes |
| watermill-nats v2.1.3 | `go.mod` line 10 | Yes |
| NATS Server v2.12.3 | `go.mod` line 19 | Yes |
| nats.go v1.48.0 | `go.mod` line 20 | Yes |
| Build tag: nats | `cmd/server/nats_init.go` line 1 | Yes |
| Stream name: MEDIA_EVENTS | `internal/eventprocessor/config.go` | Yes |
| MediaEvent struct | `internal/eventprocessor/events.go` | Yes |

### Test Coverage

- Event processor tests: `internal/eventprocessor/*_test.go`
- NATS integration tests: Run with `-tags nats`
- Coverage target: 80%+ for eventprocessor package

---

## Related ADRs

- [ADR-0004](0004-process-supervision-with-suture.md): NATS supervised by messaging layer
- [ADR-0006](0006-badgerdb-write-ahead-log.md): WAL for event durability
- [ADR-0007](0007-event-sourcing-architecture.md): Event sourcing mode
- [ADR-0009](0009-plex-direct-integration.md): Plex webhook events

---

## References

- [NATS JetStream Documentation](https://docs.nats.io/nats-concepts/jetstream)
- [Watermill Documentation](https://watermill.io/)
- [Watermill NATS Adapter](https://github.com/ThreeDotsLabs/watermill-nats)
- [WATERMILL_NATS_ARCHITECTURE.md](../WATERMILL_NATS_ARCHITECTURE.md)
