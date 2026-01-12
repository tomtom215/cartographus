# ADR-0023: Consumer-Side WAL for Exactly-Once Delivery

**Status**: Accepted
**Date**: 2025-12-20
**Context**: Closing the critical durability gap between NATS and DuckDB to achieve exactly-once semantics.

## Context

Cartographus is designed as a **system of record** with perfect data quality that can always be trusted, traced, and audited. The current architecture has a critical durability gap:

### Current State (AT-LEAST-ONCE)

```
Sync → BadgerDB WAL → NATS JetStream → DuckDB Appender → DuckDB
      [DURABLE]        [DURABLE]        [IN-MEMORY]      [DURABLE]
                                        ↑
                                        CRITICAL GAP
```

**Gaps Identified:**
1. **NATS → DuckDB**: In-memory appender buffer, events lost on crash
2. **DLQ**: Failed events go to NATS topic only, not persisted to DuckDB
3. **No exactly-once**: Deduplication relies on in-memory cache + DB constraints

### Failure Scenarios (Current)

| Scenario | Current Behavior | Data Loss Risk |
|----------|------------------|----------------|
| App crash during appender flush | Buffer lost | HIGH |
| DuckDB unavailable | 5 retries → DLQ → NATS topic (7-day TTL) | MEDIUM |
| NATS down during sync | WAL retries (good) | LOW |
| Power failure | Appender buffer lost | HIGH |

## Decision

Implement a **Consumer-Side WAL** that mirrors the existing producer-side WAL pattern, ensuring exactly-once delivery from NATS to DuckDB.

### Architecture

```
                           ┌─────────────────────────────────────────────────────┐
                           │              EXACTLY-ONCE DELIVERY                  │
                           └─────────────────────────────────────────────────────┘

┌──────────────┐    ┌──────────────┐    ┌──────────────────────────────────────────┐
│ Event Source │───►│ Producer WAL │───►│              NATS JetStream              │
│ (Tautulli)   │    │ (BadgerDB)   │    │ (Persistent, 7-day retention)            │
└──────────────┘    └──────────────┘    └────────────────────┬─────────────────────┘
                                                             │
                                                             ▼
                                        ┌────────────────────────────────────────────┐
                                        │           Watermill Router                 │
                                        │    (Dedup middleware, Poison queue)        │
                                        └────────────────────┬───────────────────────┘
                                                             │
                                                             ▼
┌────────────────────────────────────────────────────────────────────────────────────┐
│                           CONSUMER WAL (NEW)                                       │
│  ┌─────────────────────────────────────────────────────────────────────────────┐  │
│  │ 1. NATS Message Received                                                     │  │
│  │    ↓                                                                         │  │
│  │ 2. Parse MediaEvent, generate transaction_id                                 │  │
│  │    ↓                                                                         │  │
│  │ 3. WAL.Write(event, transaction_id) → BadgerDB (ACID, fsync)                │  │
│  │    ↓                                                                         │  │
│  │ 4. ACK NATS message (event now durable in consumer WAL)                      │  │
│  │    ↓                                                                         │  │
│  │ 5. DuckDB.Insert(event) with transaction_id                                  │  │
│  │    ├─── SUCCESS → WAL.Confirm(transaction_id)                                │  │
│  │    │                                                                         │  │
│  │    └─── FAILURE → Entry stays in WAL for retry                               │  │
│  │                   Retry loop picks up on next interval                       │  │
│  │                   After max retries → Move to failed_events table            │  │
│  └─────────────────────────────────────────────────────────────────────────────┘  │
└────────────────────────────────────────────────────────────────────────────────────┘
                                                             │
                                                             ▼
                                        ┌────────────────────────────────────────────┐
                                        │              DuckDB                        │
                                        │  ┌──────────────────────────────────────┐  │
                                        │  │ playback_events (main table)         │  │
                                        │  │ + transaction_id column for idempot. │  │
                                        │  └──────────────────────────────────────┘  │
                                        │  ┌──────────────────────────────────────┐  │
                                        │  │ failed_events (persistent DLQ)       │  │
                                        │  │ + full event data for recovery       │  │
                                        │  │ + error details, retry count         │  │
                                        │  └──────────────────────────────────────┘  │
                                        └────────────────────────────────────────────┘
```

### Key Components

#### 1. Consumer WAL Entry Format

```go
// From internal/wal/consumer_wal.go
type ConsumerWALEntry struct {
    ID              string          `json:"id"`                           // UUID, unique identifier
    TransactionID   string          `json:"transaction_id"`               // Idempotency key for DuckDB
    NATSSubject     string          `json:"nats_subject,omitempty"`       // Original NATS subject
    NATSMessageID   string          `json:"nats_message_id,omitempty"`    // NATS message ID for correlation
    EventPayload    json.RawMessage `json:"event_payload"`                // Full MediaEvent payload

    // State tracking
    CreatedAt       time.Time       `json:"created_at"`
    Attempts        int             `json:"attempts"`
    LastAttemptAt   time.Time       `json:"last_attempt_at,omitempty"`
    LastError       string          `json:"last_error,omitempty"`

    // Confirmation
    Confirmed       bool            `json:"confirmed"`
    ConfirmedAt     *time.Time      `json:"confirmed_at,omitempty"`

    // Failure tracking
    FailedPermanent bool            `json:"failed_permanent"`             // True if moved to failed_events
    FailedAt        *time.Time      `json:"failed_at,omitempty"`
    FailureReason   string          `json:"failure_reason,omitempty"`

    // Durable leasing (v2.4 race condition fix)
    LeaseExpiry     time.Time       `json:"lease_expiry,omitempty"`       // When processing lease expires
    LeaseHolder     string          `json:"lease_holder,omitempty"`       // Processor holding the lease
}
```

#### 2. Transaction ID Generation

```go
// From internal/wal/consumer_wal.go
// Transaction ID format: {source}:{event_id}:{sequence_number}
// Uses an atomic counter instead of time.Now().UnixNano() to ensure:
// 1. Unique IDs even when multiple events are processed simultaneously
// 2. Reproducible deduplication behavior for testing and replay scenarios
// 3. Monotonically increasing sequence for ordering guarantees
// 4. No dependency on wall clock time which can drift or be non-monotonic

var transactionCounter atomic.Uint64

func GenerateTransactionID(source, eventID string) string {
    seq := transactionCounter.Add(1)
    return fmt.Sprintf("%s:%s:%d", source, eventID, seq)
}
```

#### 3. DuckDB Idempotency Check

```sql
-- From internal/database/database_schema.go
-- transaction_id column is defined in playback_events schema
-- Column: transaction_id TEXT

-- Unique index for transaction ID (no partial indexes in DuckDB)
CREATE UNIQUE INDEX IF NOT EXISTS idx_playback_transaction_id
    ON playback_events(transaction_id);

-- Insert with conflict handling via unique constraint
-- DuckDB uses ON CONFLICT DO NOTHING syntax
INSERT INTO playback_events (..., transaction_id)
VALUES (..., ?)
ON CONFLICT DO NOTHING;
```

Note: DuckDB does NOT support partial indexes (`WHERE transaction_id IS NOT NULL`). The unique index allows NULL values, which are not considered duplicates in DuckDB.

#### 4. Failed Events Table (Persistent DLQ)

```sql
-- From internal/database/database_schema.go
CREATE TABLE IF NOT EXISTS failed_events (
    id UUID PRIMARY KEY,

    -- Original event data
    transaction_id TEXT NOT NULL,
    event_id TEXT NOT NULL,
    session_key TEXT,
    correlation_key TEXT,
    source TEXT NOT NULL,
    event_payload JSON NOT NULL,

    -- Failure details
    failed_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    failure_reason TEXT NOT NULL,
    failure_layer TEXT NOT NULL,  -- 'consumer_wal', 'duckdb_insert', 'validation'
    last_error TEXT,

    -- Retry tracking
    retry_count INTEGER NOT NULL DEFAULT 0,
    last_retry_at TIMESTAMPTZ,
    max_retries_exceeded BOOLEAN DEFAULT FALSE,

    -- Resolution
    status TEXT NOT NULL DEFAULT 'pending',  -- 'pending', 'retrying', 'resolved', 'abandoned'
    resolved_at TIMESTAMPTZ,
    resolved_by TEXT,
    resolution_notes TEXT,

    -- Audit
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Indexes (DuckDB doesn't support inline index definitions)
CREATE INDEX IF NOT EXISTS idx_failed_events_status ON failed_events(status);
CREATE INDEX IF NOT EXISTS idx_failed_events_source ON failed_events(source);
CREATE INDEX IF NOT EXISTS idx_failed_events_failed_at ON failed_events(failed_at);
CREATE INDEX IF NOT EXISTS idx_failed_events_transaction_id ON failed_events(transaction_id);
```

#### 5. BadgerDB Key Prefixes

```go
// From internal/wal/consumer_wal.go
const (
    consumerPrefixPending   = "consumer:pending:"   // Active entries awaiting DuckDB insert
    consumerPrefixConfirmed = "consumer:confirmed:" // Successfully inserted, awaiting cleanup
    consumerPrefixFailed    = "consumer:failed:"    // Permanently failed, moved to failed_events
)
```

### Data Flow (Exactly-Once Guarantee)

```
NATS Message Arrives
        │
        ▼
┌───────────────────────────────────────────────────────────────┐
│ STEP 1: Parse and Validate                                    │
│   - Unmarshal JSON payload                                    │
│   - Validate required fields                                  │
│   - Generate transaction_id                                   │
│   - Check if transaction_id already exists in DuckDB          │
│     (If exists: ACK message, skip - already processed)        │
└───────────────────────────────────┬───────────────────────────┘
                                    │
                                    ▼
┌───────────────────────────────────────────────────────────────┐
│ STEP 2: Consumer WAL Write (ACID, fsync)                      │
│   - Write entry to BadgerDB with pending: prefix              │
│   - Entry includes: transaction_id, full event, metadata      │
│   - SyncWrites=true ensures durability                        │
│   - On failure: NACK message (NATS will retry)                │
└───────────────────────────────────┬───────────────────────────┘
                                    │
                                    ▼
┌───────────────────────────────────────────────────────────────┐
│ STEP 3: ACK NATS Message                                      │
│   - Event is now durable in Consumer WAL                      │
│   - NATS message can be safely acknowledged                   │
│   - From this point, Consumer WAL owns the event              │
└───────────────────────────────────┬───────────────────────────┘
                                    │
                                    ▼
┌───────────────────────────────────────────────────────────────┐
│ STEP 4: DuckDB Insert (with transaction_id)                   │
│   - Insert with ON CONFLICT DO NOTHING (DuckDB syntax)        │
│   - If success: WAL.Confirm(entry_id)                         │
│   - If failure: Entry stays in WAL for retry                  │
└───────────────────────────────────┬───────────────────────────┘
                                    │
        ┌───────────────────────────┴───────────────────────────┐
        │                                                       │
        ▼                                                       ▼
┌───────────────────┐                           ┌───────────────────────────┐
│ SUCCESS           │                           │ FAILURE                   │
│ - WAL.Confirm()   │                           │ - Update attempts count   │
│ - Entry moves to  │                           │ - Entry stays at          │
│   consumer:confirmed:                         │   consumer:pending:       │
│ - CleanupConfirmed() cleans up later          │ - Retry loop picks up     │
└───────────────────┘                           └───────────────────────────┘
                                                            │
                                                            ▼
                                    ┌───────────────────────────────────────┐
                                    │ RETRY LOOP (30s interval)             │
                                    │ - Exponential backoff: 5s → 5min cap  │
                                    │ - Max 100 retries (configurable)      │
                                    │ - After max: Move to failed_events    │
                                    └───────────────────────────────────────┘
                                                            │
                                                            ▼
                                    ┌───────────────────────────────────────┐
                                    │ PERMANENT FAILURE                     │
                                    │ - Insert into failed_events table     │
                                    │ - WAL.MarkFailed() moves entry to     │
                                    │   consumer:failed: prefix             │
                                    │ - Log with full context               │
                                    │ - Available for manual recovery       │
                                    └───────────────────────────────────────┘
```

### Recovery on Startup

```go
// From internal/wal/consumer_wal.go

// RecoveryCallback provides the database operations needed for crash recovery.
// This decouples the Consumer WAL from the database implementation.
type RecoveryCallback interface {
    // TransactionIDExists checks if a transaction ID is already in DuckDB.
    TransactionIDExists(ctx context.Context, transactionID string) (bool, error)

    // InsertEvent inserts the event payload into DuckDB with the transaction ID.
    InsertEvent(ctx context.Context, payload []byte, transactionID string) error

    // InsertFailedEvent moves an event to the failed_events table.
    InsertFailedEvent(ctx context.Context, entry *ConsumerWALEntry, reason string) error
}

// ConsumerRecoveryResult contains statistics from consumer WAL startup recovery.
type ConsumerRecoveryResult struct {
    TotalPending     int
    AlreadyCommitted int
    Recovered        int
    Expired          int
    Failed           int
    Skipped          int           // Entries skipped due to concurrent processing
    Duration         time.Duration
}

func (w *ConsumerWAL) RecoverOnStartup(ctx context.Context, callback RecoveryCallback) (*ConsumerRecoveryResult, error) {
    result := &ConsumerRecoveryResult{}

    // 1. Get all pending entries from previous run
    entries, err := w.GetPending(ctx)
    // ...

    for _, entry := range entries {
        // Durable leasing: claim exclusive processing rights
        leaseHolder := fmt.Sprintf("recovery-%s", uuid.New().String()[:8])
        claimed, err := w.TryClaimEntryDurable(ctx, entry.ID, leaseHolder)
        if !claimed {
            result.Skipped++
            continue
        }

        // 2. Check if already in DuckDB (crash after insert, before confirm)
        exists, err := callback.TransactionIDExists(ctx, entry.TransactionID)
        if exists {
            w.Confirm(ctx, entry.ID)
            result.AlreadyCommitted++
            continue
        }

        // 3. Check expiration
        if time.Since(entry.CreatedAt) > w.config.EntryTTL {
            callback.InsertFailedEvent(ctx, entry, "expired")
            w.MarkFailed(ctx, entry.ID, "expired after "+w.config.EntryTTL.String())
            result.Expired++
            continue
        }

        // 4. Check max retries
        if entry.Attempts >= w.config.MaxRetries {
            callback.InsertFailedEvent(ctx, entry, "max_retries_exceeded")
            w.MarkFailed(ctx, entry.ID, "exceeded max retries")
            result.Failed++
            continue
        }

        // 5. Retry DuckDB insert
        if err := callback.InsertEvent(ctx, entry.EventPayload, entry.TransactionID); err != nil {
            w.UpdateAttempt(ctx, entry.ID, err.Error())
            result.Failed++
            continue
        }

        // 6. Success
        w.Confirm(ctx, entry.ID)
        result.Recovered++
    }

    return result, nil
}
```

### Configuration

```yaml
# Environment variables (from internal/wal/consumer_wal.go)
CONSUMER_WAL_PATH: /data/wal-consumer       # Directory for BadgerDB storage
CONSUMER_WAL_SYNC_WRITES: true              # Force fsync after every write
CONSUMER_WAL_RETRY_INTERVAL: 30s            # Interval between retry loop iterations
CONSUMER_WAL_MAX_RETRIES: 100               # Maximum DuckDB insert attempts
CONSUMER_WAL_RETRY_BACKOFF: 5s              # Initial backoff duration (exponential)
CONSUMER_WAL_ENTRY_TTL: 168h                # 7 days TTL for unconfirmed entries
CONSUMER_WAL_MEMTABLE_SIZE: 16777216        # 16MB memtable size
CONSUMER_WAL_VLOG_SIZE: 67108864            # 64MB value log file size
CONSUMER_WAL_NUM_COMPACTORS: 2              # Number of compaction workers (min 2)
CONSUMER_WAL_COMPRESSION: true              # Enable Snappy compression
CONSUMER_WAL_CLOSE_TIMEOUT: 30s             # Maximum time for graceful shutdown
CONSUMER_WAL_LEASE_DURATION: 2m             # Durable lease duration for concurrent processing prevention
```

Note: Consumer WAL is enabled at compile time via the `wal` build tag. There is no runtime enable/disable flag.

### Metrics

```go
// From internal/wal/metrics.go
// Consumer WAL Prometheus metrics (ADR-0023: Exactly-Once Delivery)

consumer_wal_writes_total         // Total Consumer WAL write operations
consumer_wal_confirms_total       // Successful DuckDB inserts confirmed
consumer_wal_retries_total        // Retry attempts
consumer_wal_failures_total       // Permanently failed entries
consumer_wal_recoveries_total     // Entries recovered on startup
consumer_wal_pending_entries      // Current pending entries (gauge)
```

Note: DuckDB-specific transaction metrics are not implemented. Tracking is done via the Consumer WAL metrics above.

## Consequences

### Positive
- **Exactly-once semantics**: Transaction ID ensures no duplicates
- **Crash tolerance**: Events survive app crash, power failure
- **Full auditability**: Every event tracked from source to DuckDB
- **Recovery capability**: Pending events recovered on startup
- **Persistent DLQ**: Failed events stored in DuckDB for investigation
- **Manual recovery**: Users can retry/resolve failed events via UI

### Negative
- **Increased latency**: Additional WAL write before DuckDB
- **Storage overhead**: Two WALs (producer + consumer) + failed_events table
- **Complexity**: More components to monitor and maintain
- **Transaction ID overhead**: Extra column and index in playback_events

### Mitigations
- **Latency**: Use async WAL writes where safe, batch optimizations
- **Storage**: Aggressive compaction, configurable TTL
- **Complexity**: Comprehensive metrics, health checks, clear documentation

## Related ADRs

- ADR-0005: NATS JetStream Event Processing
- ADR-0006: BadgerDB Write-Ahead Log
- ADR-0007: Event Sourcing Architecture
- ADR-0022: Deduplication Audit and Management System

## Implementation Plan

1. **Phase 1**: Add transaction_id to DuckDB schema
2. **Phase 2**: Implement ConsumerWAL with BadgerDB
3. **Phase 3**: Create failed_events table and APIs
4. **Phase 4**: Integrate with DuckDBHandler
5. **Phase 5**: Add recovery logic
6. **Phase 6**: Metrics and health checks
7. **Phase 7**: Tests (unit, integration, chaos)

## Source Files

| Component | Path |
|-----------|------|
| Consumer WAL | `internal/wal/consumer_wal.go` |
| WAL Config | `internal/wal/config.go` |
| Recovery Logic | `internal/wal/recovery.go` |
| Retry Loop | `internal/wal/retry.go` |
| Metrics | `internal/wal/metrics.go` |
| Database Schema | `internal/database/database_schema.go` |
| TransactionIDExists | `internal/database/crud_playback.go` |
| InsertFailedEvent | `internal/database/crud_playback.go` |
| FailedEvent Model | `internal/models/playback.go` |
