# ADR-0027: WebSocket Real-Time Hub Architecture

**Date**: 2026-01-11
**Status**: Accepted
**Last Verified**: 2026-01-11

---

## Context

Cartographus displays live analytics dashboards with real-time updates for:
- Sync completion notifications (new records, duration)
- Statistics updates (total playbacks, unique users)
- Live playback activity from media servers
- Detection alerts for security events
- Sync progress for long-running operations

### Requirements

1. **Real-Time Updates**: Sub-second delivery of events to connected frontends
2. **Scalability**: Support 1000+ concurrent WebSocket connections
3. **NATS Integration**: Bridge NATS JetStream events to WebSocket clients
4. **Graceful Shutdown**: Clean client disconnection during server restart
5. **Deterministic Behavior**: Predictable message ordering for testing
6. **Suture Supervision**: Compatible with the process supervision tree

### Alternatives Considered

| Approach | Pros | Cons |
|----------|------|------|
| Server-Sent Events (SSE) | Simple, HTTP-based | Unidirectional, no ping/pong |
| Long Polling | Universal compatibility | High latency, inefficient |
| WebSocket (Hub-Spoke) | Bidirectional, efficient | More complex than SSE |
| gRPC Streaming | Type-safe, efficient | Requires client library |

---

## Decision

Implement a **hub-and-spoke WebSocket architecture** using `gorilla/websocket` with:

1. **Central Hub**: Manages all client connections and broadcasts messages
2. **Client Goroutines**: Each client runs read/write pumps in separate goroutines
3. **Priority-Based Selection**: Deterministic channel handling for predictable behavior
4. **NATS Bridge**: Optional subscriber that forwards NATS events to WebSocket

### Architecture

```
                    ┌──────────────────────┐
                    │   NATS JetStream     │
                    │   (playback.>)       │
                    └──────────┬───────────┘
                               │
                    ┌──────────▼───────────┐
                    │   NATSSubscriber     │
                    │  (nats build tag)    │
                    └──────────┬───────────┘
                               │
                    ┌──────────▼───────────┐
                    │        Hub           │
                    │  - clients map       │
                    │  - broadcast chan    │
                    │  - Register chan     │
                    │  - Unregister chan   │
                    └──────────┬───────────┘
                               │
        ┌──────────┬───────────┼───────────┬──────────┐
        │          │           │           │          │
   ┌────▼────┐ ┌───▼────┐ ┌────▼────┐ ┌────▼────┐ ┌───▼────┐
   │ Client1 │ │ Client2│ │ Client3 │ │ Client4 │ │Client N│
   │ readPump│ │readPump│ │readPump │ │readPump │ │readPump│
   │writePump│ │writePump│ │writePump│ │writePump│ │writePump│
   └─────────┘ └────────┘ └─────────┘ └─────────┘ └────────┘
```

---

## Consequences

### Positive

- **Efficient Broadcasting**: O(n) delivery to all clients via channel-based hub
- **Connection Management**: Automatic cleanup of dead connections via ping/pong
- **Deterministic Tests**: Priority-based selection enables reproducible behavior
- **Graceful Shutdown**: Context-aware RunWithContext for Suture integration
- **NATS Decoupling**: Optional NATS bridge via build tags

### Negative

- **Goroutine Overhead**: Each client requires 2 goroutines (read/write pumps)
- **Memory Pressure**: 256-message buffer per client can accumulate during slow clients
- **Single Hub**: No horizontal scaling without external message broker

### Neutral

- **gorilla/websocket**: Well-maintained library, no external dependencies
- **Build Tags**: NATS subscriber requires `nats` build tag

---

## Implementation

### Message Types

| Type | Description | Data Structure |
|------|-------------|----------------|
| `playback` | New playback event | `PlaybackEvent` |
| `sync_completed` | Sync operation finished | `SyncCompletedData` |
| `stats_update` | Statistics changed | `StatsUpdateData` |
| `detection_alert` | Security detection fired | Detection event |
| `sync_progress` | Long-running sync progress | `SyncProgressData` |
| `ping` / `pong` | Keep-alive heartbeat | null |

### Hub Structure

```go
// Hub maintains the set of active clients and broadcasts messages
// Location: internal/websocket/hub.go:47-53
type Hub struct {
    clients    map[*Client]bool
    broadcast  chan Message
    Register   chan *Client
    Unregister chan *Client
    mu         sync.RWMutex
}
```

### Client Structure

```go
// Client is a middleman between the websocket connection and the hub
// Location: internal/websocket/client.go:24-31
type Client struct {
    id   uint64              // Deterministic ordering via atomic counter
    hub  *Hub
    conn *websocket.Conn
    send chan Message
}
```

### Deterministic Broadcast

```go
// broadcastToClients sends a message to all clients in deterministic order
// Location: internal/websocket/hub.go:257-292
func (h *Hub) broadcastToClients(message Message) {
    h.mu.Lock()
    defer h.mu.Unlock()

    // Sort clients by ID for consistent ordering
    clients := make([]*Client, 0, len(h.clients))
    for client := range h.clients {
        clients = append(clients, client)
    }
    sort.Slice(clients, func(i, j int) bool {
        return clients[i].id < clients[j].id
    })

    for _, client := range clients {
        select {
        case client.send <- message:
        default:
            // Channel full, remove client
            close(client.send)
            delete(h.clients, client)
        }
    }
}
```

### NATS Bridge (Build Tag: `nats`)

```go
// NATSSubscriber bridges NATS events to WebSocket broadcasts
// Location: internal/websocket/nats_subscriber.go:87-106
type NATSSubscriber struct {
    hub     *Hub
    handler NATSMessageHandler
    mu      sync.Mutex
    running bool
    stopCh  chan struct{}
    doneCh  chan struct{}
}
```

### Configuration Constants

| Constant | Value | Description |
|----------|-------|-------------|
| `writeWait` | 10s | Time allowed to write a message |
| `pongWait` | 60s | Time allowed to read pong from client |
| `pingPeriod` | 54s | Ping interval (90% of pongWait) |
| `maxMessageSize` | 512KB | Maximum message size |
| Broadcast buffer | 256 | Hub broadcast channel capacity |
| Client buffer | 256 | Per-client send channel capacity |

---

## Code References

| Component | File | Notes |
|-----------|------|-------|
| Hub | `internal/websocket/hub.go` | Central message broker |
| Client | `internal/websocket/client.go` | Connection wrapper with pumps |
| NATS Bridge | `internal/websocket/nats_subscriber.go` | Requires `nats` build tag |
| Package Doc | `internal/websocket/doc.go` | Architecture documentation |
| Hub Tests | `internal/websocket/hub_test.go` | 851 lines of tests |
| Client Tests | `internal/websocket/client_test.go` | 502 lines of tests |
| NATS Tests | `internal/websocket/nats_subscriber_test.go` | 325 lines of tests |

---

## Verification

### Verified Claims

| Claim | Source | Verified |
|-------|--------|----------|
| gorilla/websocket v1.5.3 | `go.mod:18` | Yes |
| Hub-and-spoke pattern | `internal/websocket/doc.go:17-27` | Yes |
| Priority-based selection | `internal/websocket/hub.go:69-121` | Yes |
| Deterministic client ordering | `internal/websocket/hub.go:268-272` | Yes |
| NATS build tag | `internal/websocket/nats_subscriber.go:1` | Yes |
| 256-message buffers | `internal/websocket/hub.go:58`, `client.go:39` | Yes |
| RunWithContext for Suture | `internal/websocket/hub.go:124-204` | Yes |
| Graceful shutdown | `internal/websocket/hub.go:206-249` | Yes |

### Test Coverage

- `internal/websocket/hub_test.go` - Hub operations and broadcasting
- `internal/websocket/client_test.go` - Client lifecycle and message handling
- `internal/websocket/nats_subscriber_test.go` - NATS bridge integration

---

## Related ADRs

- [ADR-0004](0004-process-supervision-with-suture.md): Suture supervision for hub service
- [ADR-0005](0005-nats-jetstream-event-processing.md): NATS events that feed WebSocket
- [ADR-0007](0007-event-sourcing-architecture.md): Event sourcing integration

---

## References

- [gorilla/websocket](https://github.com/gorilla/websocket): WebSocket library
- [RFC 6455](https://datatracker.ietf.org/doc/html/rfc6455): WebSocket Protocol
- Internal: `internal/websocket/doc.go` - Package documentation
