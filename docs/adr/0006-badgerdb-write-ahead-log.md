# ADR-0006: BadgerDB Write-Ahead Log for Event Durability

**Date**: 2025-12-02
**Updated**: 2026-01-11
**Status**: Accepted

---

## Context

When using NATS JetStream for event processing (ADR-0005), there's a potential gap:

1. **DuckDB Transaction Failures**: If DuckDB write fails, the event is lost
2. **Application Crashes**: In-flight events may not persist
3. **Retry Logic**: Failed events need retry with backoff

### Requirements

- Durable event storage before DuckDB commit
- Automatic retry for failed writes
- Compaction to prevent unbounded growth
- Low latency for event acceptance
- No additional external dependencies

### Alternatives Considered

| Approach | Pros | Cons |
|----------|------|------|
| **Rely on JetStream** | Already in use | Separate data path |
| **SQLite WAL** | Proven, simple | Not designed for key-value |
| **BoltDB** | Pure Go, embedded | Single writer, no compaction |
| **BadgerDB** | LSM, compaction, pure Go | More complex API |
| **LevelDB** | Fast, proven | CGO required |

---

## Decision

Use **BadgerDB v4** as a write-ahead log (WAL) for event durability:

- **Pre-Commit Buffer**: Events written to BadgerDB before DuckDB commit
- **Retry Loop**: Failed events automatically retried with exponential backoff
- **Compaction**: Background compaction prevents storage growth
- **Build Tag Isolation**: Optional feature via `wal` build tag

### Architecture

```
┌─────────────┐
│   Event     │
│   Producer  │
└──────┬──────┘
       │
       ▼
┌─────────────────┐
│   BadgerDB WAL  │ ◄─── Pre-commit buffer
│   (LSM Tree)    │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│   Retry Loop    │ ◄─── Background goroutine
│   (Exponential) │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│   DuckDB        │ ◄─── Final destination
│   (Analytics)   │
└─────────────────┘
         │
         ▼
┌─────────────────┐
│   Delete from   │ ◄─── Cleanup on success
│   BadgerDB      │
└─────────────────┘
```

### Key Factors

1. **Pure Go**: No CGO required, simplifies cross-compilation
2. **LSM Architecture**: Optimized for write-heavy workloads
3. **Automatic Compaction**: Built-in garbage collection
4. **Key-Value Simplicity**: Natural fit for event buffering
5. **Transactions**: ACID guarantees for event writes

---

## Consequences

### Positive

- **Event Durability**: No event loss on DuckDB failures
- **Automatic Retry**: Failed writes retried with backoff
- **Bounded Storage**: Compaction prevents unbounded growth
- **Low Latency**: LSM tree optimized for fast writes
- **No External Dependencies**: Embedded operation

### Negative

- **Memory Usage**: LSM memtables consume memory
- **Disk Space**: WAL + DuckDB doubles storage temporarily
- **Complexity**: Additional component to manage

### Neutral

- **Optional Feature**: Disabled by default, enabled via build tag
- **Background Goroutines**: Retry loop and compactor run concurrently

---

## Implementation

### WAL Configuration

```go
// internal/wal/config.go
type Config struct {
    Enabled          bool          // Enable WAL durability (default: true)
    Path             string        // /data/wal
    SyncWrites       bool          // true for durability
    RetryInterval    time.Duration // 30s - interval between retry loop iterations
    MaxRetries       int           // 100 - maximum retry attempts
    RetryBackoff     time.Duration // 5s - initial exponential backoff
    CompactInterval  time.Duration // 1h - compaction frequency
    EntryTTL         time.Duration // 168h (7 days) - entry time-to-live
    MemTableSize     int64         // 16MB - BadgerDB memtable size
    ValueLogFileSize int64         // 64MB - value log file size
    NumCompactors    int           // 2 - compaction workers
    Compression      bool          // true - Snappy compression
    GCRatio          float64       // 0.5 - value log GC aggressiveness
    CloseTimeout     time.Duration // 30s - graceful shutdown timeout
    NumMemtables     int           // 5 - memtables in memory
    BlockCacheSize   int64         // 256MB - block cache size
    IndexCacheSize   int64         // 0 (disabled) - index cache size
    LeaseDuration    time.Duration // 2m - durable lease duration
}
```

### WAL Operations

```go
// internal/wal/wal.go

// WAL interface defines the contract for write-ahead logging
type WAL interface {
    Write(ctx context.Context, event interface{}) (entryID string, err error)
    Confirm(ctx context.Context, entryID string) error
    GetPending(ctx context.Context) ([]*Entry, error)
    Stats() Stats
    Close() error
}

// BadgerWAL implements WAL using BadgerDB for durable storage
type BadgerWAL struct {
    db                *badger.DB
    config            Config
    totalWrites       atomic.Int64
    totalConfirms     atomic.Int64
    totalRetries      atomic.Int64
    lastCompaction    time.Time
    mu                sync.RWMutex
    closed            bool
    processingEntries sync.Map  // Race condition prevention (v2.4)
}

// Write persists an event to the WAL with native TTL support
func (w *BadgerWAL) Write(ctx context.Context, event interface{}) (string, error) {
    // Serialize event to JSON, generate UUID entry ID
    // Write to BadgerDB with optional TTL
    // Returns entry ID for later confirmation
}

// Confirm marks an entry as successfully published
func (w *BadgerWAL) Confirm(ctx context.Context, entryID string) error {
    // Move entry from pending to confirmed state
}

// GetPending returns all unconfirmed entries for retry
func (w *BadgerWAL) GetPending(ctx context.Context) ([]*Entry, error) {
    // Iterate pending prefix and return entries
}

// GetPendingStream returns a channel for memory-efficient streaming
func (w *BadgerWAL) GetPendingStream(ctx context.Context) (<-chan *Entry, <-chan error) {
    // Stream entries via channel for large WALs
}
```

### Retry Loop

```go
// internal/wal/retry.go
type RetryLoop struct {
    wal         *BadgerWAL
    publisher   Publisher       // Interface for publishing entries
    config      Config
    leaseHolder string          // Unique ID for durable leasing
    ctx         context.Context
    cancel      context.CancelFunc
    mu          sync.Mutex
    running     bool
    stopping    bool
    stopDone    chan struct{}
}

func (r *RetryLoop) Start(ctx context.Context) error {
    r.mu.Lock()
    // Wait for any in-progress Stop() to complete
    for r.stopping {
        stopDone := r.stopDone
        r.mu.Unlock()
        <-stopDone
        r.mu.Lock()
    }
    r.ctx, r.cancel = context.WithCancel(ctx)
    r.running = true
    r.stopDone = make(chan struct{})
    r.mu.Unlock()
    go r.runWithContext(r.ctx, r.stopDone)
    return nil
}

func (r *RetryLoop) runWithContext(ctx context.Context, done chan struct{}) {
    defer close(done)
    ticker := time.NewTicker(r.config.RetryInterval)
    defer ticker.Stop()
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            r.retryPendingWithContext(ctx)
        }
    }
}

func (r *RetryLoop) retryPendingWithContext(ctx context.Context) {
    entries, err := r.wal.GetPending(ctx)
    // Process each entry with durable leasing via TryClaimEntryDurable
    // Uses exponential backoff: base * 2^attempts, capped at 5 minutes
}
```

### Compactor

```go
// internal/wal/compaction.go
type Compactor struct {
    wal              *BadgerWAL
    config           Config
    ctx              context.Context
    cancel           context.CancelFunc
    wg               sync.WaitGroup
    mu               sync.Mutex
    running          bool
    lastRun          time.Time
    lastEntriesCount int64
}

func (c *Compactor) run() {
    defer c.wg.Done()
    ticker := time.NewTicker(c.config.CompactInterval)
    defer ticker.Stop()
    for {
        select {
        case <-c.ctx.Done():
            return
        case <-ticker.C:
            c.compact()
        }
    }
}

func (c *Compactor) compact() {
    // Delete confirmed entries
    // Delete expired pending entries (older than EntryTTL)
    // Run BadgerDB GC via c.wal.RunGC()
    // Record metrics: RecordWALCompaction(), RecordWALCompactionLatency()
}
```

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `WAL_ENABLED` | `true` | Enable BadgerDB WAL |
| `WAL_PATH` | `/data/wal` | WAL storage directory |
| `WAL_SYNC_WRITES` | `true` | Sync writes to disk (fsync) |
| `WAL_MAX_RETRIES` | `100` | Maximum retry attempts |
| `WAL_RETRY_INTERVAL` | `30s` | Retry loop interval |
| `WAL_RETRY_BACKOFF` | `5s` | Initial exponential backoff |
| `WAL_COMPACT_INTERVAL` | `1h` | Compaction frequency |
| `WAL_ENTRY_TTL` | `168h` | Entry time-to-live (7 days) |
| `WAL_MEMTABLE_SIZE` | `16MB` | BadgerDB memtable size |
| `WAL_VLOG_SIZE` | `64MB` | Value log file size |
| `WAL_NUM_COMPACTORS` | `2` | Compaction workers |
| `WAL_COMPRESSION` | `true` | Enable Snappy compression |
| `WAL_GC_RATIO` | `0.5` | Value log GC aggressiveness (0.0-1.0) |
| `WAL_CLOSE_TIMEOUT` | `30s` | Graceful shutdown timeout |
| `WAL_NUM_MEMTABLES` | `5` | Number of memtables in memory |
| `WAL_BLOCK_CACHE_SIZE` | `256MB` | Block cache size |
| `WAL_INDEX_CACHE_SIZE` | `0` | Index cache size (0=disabled) |

### Production Features (Added 2025-12-14)

#### Native TTL Support
Entries are written with BadgerDB's native TTL support, automatically expiring after `WAL_ENTRY_TTL`. This ensures entries are cleaned up even without explicit compaction, preventing unbounded growth.

#### Snappy Compression
Enabled by default (`WAL_COMPRESSION=true`), Snappy compression reduces disk usage by 40-60% for JSON payloads with minimal CPU overhead.

#### Graceful Shutdown Timeout
The `Close()` method includes a configurable timeout (`WAL_CLOSE_TIMEOUT`) to prevent indefinite hangs during shutdown. If the database doesn't close within the timeout, an error is returned.

#### Streaming Recovery
For large WALs, streaming recovery (`RecoverPendingStream`) processes entries one at a time instead of loading all entries into memory, improving memory efficiency during crash recovery.

#### Comprehensive Metrics
Prometheus metrics for monitoring:
- `wal_write_latency_seconds` - Write operation latency histogram
- `wal_write_failures_total` - Failed write operations counter
- `wal_compaction_latency_seconds` - Compaction duration histogram
- `wal_gc_latency_seconds` - GC duration histogram
- `wal_gc_runs_total` - GC run counter
- Plus existing metrics for writes, confirms, retries, pending entries, etc.

### Code References

| Component | File | Notes |
|-----------|------|-------|
| WAL implementation | `internal/wal/wal.go` | BadgerDB wrapper, native TTL, streaming |
| Retry loop | `internal/wal/retry.go` | Background retry service |
| Compactor | `internal/wal/compaction.go` | GC service with latency metrics |
| Configuration | `internal/wal/config.go` | WAL settings, env var loading |
| Metrics | `internal/wal/metrics.go` | Prometheus metrics |
| Recovery | `internal/wal/recovery.go` | Crash recovery, streaming recovery |
| Supervisor service | `internal/supervisor/services/wal_service.go` | Build tag: wal |
| WAL publisher | `internal/eventprocessor/wal_publisher.go` | NATS integration, Build tag: wal && nats |
| Initialization | `cmd/server/wal_init.go` | WAL lifecycle management |

---

## Verification

### Verified Claims

| Claim | Source | Verified |
|-------|--------|----------|
| BadgerDB v4.9.0 | `go.mod:32` | Yes |
| Build tag: wal | `internal/wal/wal.go:1` | Yes |
| Build tag: wal && nats | `internal/eventprocessor/wal_publisher.go:1` | Yes |
| LSM architecture | BadgerDB documentation | Yes |
| Automatic GC | `internal/wal/compaction.go` | Yes |
| Native TTL support | `internal/wal/wal.go:296` | Yes |
| Snappy compression | `internal/wal/wal.go:167` | Yes |
| Durable leasing | `internal/wal/wal.go:700` | Yes |

### Test Coverage

- WAL tests: `internal/wal/*_test.go`
- Run with: `go test -tags wal -race ./internal/wal/...`
- Coverage target: 80%+ for wal package

---

## Related ADRs

- [ADR-0004](0004-process-supervision-with-suture.md): WAL services in data layer
- [ADR-0005](0005-nats-jetstream-event-processing.md): Event processing integration
- [ADR-0007](0007-event-sourcing-architecture.md): Event sourcing mode

---

## References

- [BadgerDB Documentation](https://dgraph.io/docs/badger/)
- [BadgerDB GitHub](https://github.com/dgraph-io/badger)
- [LSM Tree Architecture](https://en.wikipedia.org/wiki/Log-structured_merge-tree)
- [Write-Ahead Logging](https://en.wikipedia.org/wiki/Write-ahead_logging)
