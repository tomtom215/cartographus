# Code Patterns

This document describes the key architectural patterns used in Cartographus.

**Related Documentation**:
- [DEVELOPMENT.md](./DEVELOPMENT.md) - Development workflow
- [ARCHITECTURE.md](./ARCHITECTURE.md) - System architecture

---

## Handler Pattern

All API handlers follow this structure:

```go
func (h *Handler) EndpointName(w http.ResponseWriter, r *http.Request) {
    // 1. Parse query parameters
    filter := parseFilterFromRequest(r)

    // 2. Validate input
    if err := validateFilter(filter); err != nil {
        h.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
        return
    }

    // 3. Check cache
    cacheKey := fmt.Sprintf("endpoint:%v", filter)
    if cached, ok := h.cache.Get(cacheKey); ok {
        h.writeJSON(w, http.StatusOK, cached)
        return
    }

    // 4. Query database
    data, err := h.db.GetData(filter)
    if err != nil {
        h.writeError(w, http.StatusInternalServerError, "DATABASE_ERROR", err.Error())
        return
    }

    // 5. Cache result
    h.cache.Set(cacheKey, data)

    // 6. Return JSON
    h.writeJSON(w, http.StatusOK, data)
}
```

---

## Cache-First Executor Pattern

Extracted in `spatial_executor.go` and `analytics_executor.go`:

```go
// SpatialQueryExecutor simplifies handler code
type SpatialQueryExecutor struct {
    handler *Handler
}

func (e *SpatialQueryExecutor) ExecuteWithCache(
    w http.ResponseWriter,
    r *http.Request,
    cacheKey string,
    queryFunc func() (interface{}, error),
) {
    // Check cache
    if cached, ok := e.handler.cache.Get(cacheKey); ok {
        e.handler.writeJSON(w, http.StatusOK, cached)
        return
    }

    // Execute query
    data, err := queryFunc()
    if err != nil {
        e.handler.writeError(w, http.StatusInternalServerError, "QUERY_ERROR", err.Error())
        return
    }

    // Cache and respond
    e.handler.cache.Set(cacheKey, data)
    e.handler.writeJSON(w, http.StatusOK, data)
}
```

Usage:

```go
func (h *Handler) SpatialHexagons(w http.ResponseWriter, r *http.Request) {
    executor := &SpatialQueryExecutor{handler: h}
    filter := parseFilter(r)

    executor.ExecuteWithCache(w, r, "hexagons:"+filter.Hash(), func() (interface{}, error) {
        return h.db.GetHexagons(filter)
    })
}
```

---

## Request Validation Pattern

Use go-playground/validator v10 for struct-based request validation:

```go
// 1. Define request struct with validation tags in requests.go
type LoginRequestValidation struct {
    Username   string `validate:"required,min=1"`
    Password   string `validate:"required,min=1"`
    RememberMe bool
}

// 2. Use validateRequest() helper in handler
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
    var req models.LoginRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body", err)
        return
    }

    // Validate using struct tags
    validationReq := LoginRequestValidation{
        Username:   req.Username,
        Password:   req.Password,
        RememberMe: req.RememberMe,
    }
    if apiErr := validateRequest(&validationReq); apiErr != nil {
        respondError(w, http.StatusBadRequest, apiErr.Code, apiErr.Message, nil)
        return
    }

    // Continue with business logic...
}
```

Common validation tags:
- `required` - field must be present and non-zero
- `min=N,max=M` - numeric or string length bounds
- `oneof=a b c` - value must be one of the options
- `datetime=2006-01-02T15:04:05Z07:00` - RFC3339 date format
- `base64url` - URL-safe base64 encoding
- `latitude,longitude` - geographic coordinates
- `omitempty` - skip validation if empty

Files:
- `internal/validation/validator.go` - Singleton validator with error translation
- `internal/api/requests.go` - Request structs with validation tags
- `internal/api/handlers_helpers.go` - `validateRequest()` helper function

---

## Cache Invalidation

```go
// Sync completion callback (cmd/server/main.go)
syncManager.SetOnSyncCompleted(handler.OnSyncCompleted)

// Handler method
func (h *Handler) OnSyncCompleted(newPlaybacks int, duration time.Duration) {
    // Clear cache
    h.ClearCache()

    // Broadcast WebSocket update
    h.wsHub.BroadcastSyncCompleted(newPlaybacks, duration)
}
```

---

## Parallel Query Execution

```go
// Execute multiple queries concurrently
var wg sync.WaitGroup
var mu sync.Mutex
var firstErr error

wg.Add(3)

go func() {
    defer wg.Done()
    data1, err := db.Query1()
    if err != nil {
        mu.Lock()
        if firstErr == nil {
            firstErr = err
        }
        mu.Unlock()
    }
}()

// ... more goroutines

wg.Wait()
if firstErr != nil {
    return nil, firstErr
}
```

---

## WebSocket Broadcast

```go
// Hub pattern for WebSocket management
type Hub struct {
    clients    map[*Client]bool
    broadcast  chan []byte
    register   chan *Client
    unregister chan *Client
}

// Broadcast to all connected clients
hub.Broadcast(Message{
    Type: "sync_completed",
    Data: map[string]interface{}{
        "new_playbacks": count,
        "duration_ms":   duration.Milliseconds(),
    },
})
```

---

## Circuit Breaker Pattern

```go
import "github.com/sony/gobreaker/v2"

// Circuit breaker for external service calls
cb := gobreaker.NewCircuitBreaker[any](gobreaker.Settings{
    Name:        "tautulli-api",
    MaxRequests: 3,
    Interval:    10 * time.Second,
    Timeout:     30 * time.Second,
    ReadyToTrip: func(counts gobreaker.Counts) bool {
        return counts.ConsecutiveFailures > 5
    },
})

// Usage
result, err := cb.Execute(func() (interface{}, error) {
    return tautulliClient.GetHistory()
})
```

---

## Event-Driven Pattern (Watermill/NATS)

```go
import "github.com/tomtom215/cartographus/internal/eventprocessor"

// Create embedded NATS JetStream server
server, err := eventprocessor.NewEmbeddedServer(eventprocessor.DefaultServerConfig())
defer server.Shutdown(ctx)

// Create publisher with circuit breaker
pub, err := eventprocessor.NewPublisher(
    eventprocessor.DefaultPublisherConfig(server.ClientURL()),
    nil,
)
defer pub.Close()

// Publish event
event := eventprocessor.NewMediaEvent("plex")
event.UserID = 1
event.Username = "user"
msg, _ := eventprocessor.SerializeEvent(event)
pub.Publish(ctx, event.Topic(), msg)

// Subscribe with exactly-once delivery
sub, _ := eventprocessor.NewSubscriber(cfg)
messages, _ := sub.Subscribe(ctx, "MEDIA_EVENTS")
for msg := range messages {
    // Process event
    msg.Ack()
}
```

---

## Write-Ahead Log (WAL) Pattern

```go
import "github.com/tomtom215/cartographus/internal/wal"

// Load configuration
cfg := wal.LoadConfig()

// Open BadgerDB-backed WAL
w, err := wal.Open(&cfg)
defer w.Close()

// Write event before NATS publish (ACID, fsync)
entryID, err := w.Write(ctx, &mediaEvent)

// Attempt publish
if err := publisher.Publish(ctx, topic, msg); err != nil {
    // Entry remains in WAL for retry
    return nil
}

// Confirm on success
w.Confirm(ctx, entryID)

// Background retry and compaction
retryLoop := wal.NewRetryLoop(w, publisher)
retryLoop.Start(ctx)

compactor := wal.NewCompactor(w)
compactor.Start(ctx)
```

---

## Debounced Filter Updates (Frontend)

```typescript
let filterTimeout: number;

function applyFilters(filters: FilterState): void {
    clearTimeout(filterTimeout);
    filterTimeout = window.setTimeout(() => {
        updateURLParams(filters);
        refreshAllData(filters);
    }, 300);
}
```

---

## Prometheus Metrics

```go
import "github.com/prometheus/client_golang/prometheus"

// Counter for total requests
var httpRequestsTotal = prometheus.NewCounterVec(
    prometheus.CounterOpts{
        Name: "http_requests_total",
        Help: "Total number of HTTP requests",
    },
    []string{"method", "endpoint", "status"},
)

// Histogram for latency
var httpRequestDuration = prometheus.NewHistogramVec(
    prometheus.HistogramOpts{
        Name:    "http_request_duration_seconds",
        Help:    "HTTP request latency",
        Buckets: prometheus.DefBuckets,
    },
    []string{"method", "endpoint"},
)
```

---

## API Response Format

```go
// Standard success response
{
    "status": "success",
    "data": { /* payload */ },
    "metadata": {
        "timestamp": "2025-11-18T12:34:56Z",
        "query_time_ms": 23
    }
}

// Standard error response
{
    "status": "error",
    "data": null,
    "error": {
        "code": "VALIDATION_ERROR",
        "message": "Invalid date range",
        "details": { "field": "start_date" }
    }
}
```

---

## Backup/Restore Pattern

```go
import "github.com/tomtom215/cartographus/internal/backup"

mgr, err := backup.NewManager(backup.Config{
    BackupDir:     "/data/backups",
    MaxBackups:    10,
    RetentionDays: 30,
})

// Create backup
result, err := mgr.CreateBackup(ctx, backup.Options{
    Compress:    true,
    IncludeBlob: true,
})

// List and restore
backups, err := mgr.ListBackups(ctx)
err = mgr.Restore(ctx, backupID, backup.RestoreOptions{
    TargetPath: "/data/cartographus.duckdb",
})
```

---

## Concurrency Patterns

### Mutex Lock Ordering

When multiple mutexes protect different resources in the same struct, follow these rules:

1. **Single-purpose mutexes**: Each mutex protects a specific resource
2. **Document the protected resource**: Use comments to clarify

```go
type Manager struct {
    // mu protects general state (lastSync, running)
    mu     sync.RWMutex

    // syncMu serializes sync operations (prevents concurrent syncs)
    syncMu sync.Mutex

    // bufferHealthMu protects bufferHealthCache map
    bufferHealthMu sync.RWMutex
    bufferHealthCache map[string]*BufferHealth
}
```

**Lock ordering rules:**
- Never hold multiple mutexes simultaneously unless absolutely necessary
- If you must hold multiple, always acquire in alphabetical order by field name
- Prefer RWMutex for read-heavy data, Mutex for write-heavy
- Release locks as soon as possible (defer for simple cases)

```go
// Good: Single lock, released via defer
func (m *Manager) GetState() State {
    m.mu.RLock()
    defer m.mu.RUnlock()
    return m.state
}

// Good: Non-overlapping locks
func (m *Manager) UpdateBufferHealth(id string, health *BufferHealth) {
    m.bufferHealthMu.Lock()
    defer m.bufferHealthMu.Unlock()
    m.bufferHealthCache[id] = health
}

// Avoid: Holding multiple locks (if needed, use consistent order)
func (m *Manager) ComplexOperation() {
    m.bufferHealthMu.Lock()  // alphabetically first
    defer m.bufferHealthMu.Unlock()
    m.mu.Lock()              // alphabetically second
    defer m.mu.Unlock()
    // ...
}
```

### Channel Buffering Guidelines

| Channel Type | Buffer Size | Use Case |
|-------------|-------------|----------|
| Signal channels | 0 (unbuffered) | `stopChan`, `doneChan` - synchronization |
| Event channels | 100+ | High-throughput event processing |
| Error channels | 1 | Single error result from goroutine |
| Result channels | Configurable | Based on expected throughput |

```go
// Signal channel (unbuffered) - blocks until receiver ready
stopChan := make(chan struct{})

// Event channel (buffered) - allows producer to continue
eventChan := make(chan *Event, config.BufferSize) // default: 1000

// Error channel (single result)
errCh := make(chan error, 1)

// High-throughput processing
violationChan := make(chan *Alert, 100)
```

**Buffering rules:**
- Unbuffered for synchronization signals
- Buffer size >= expected burst size to prevent blocking
- For pipelines, match buffer to batch size
- Monitor channel capacity in production (metrics)
