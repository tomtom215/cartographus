# ADR-0014: Tautulli Database Import

**Date**: 2025-12-04
**Status**: Accepted

---

## Context

Users need a way to migrate historical data from existing Tautulli installations or restore from Tautulli database backups. The current API-based sync only retrieves recent data within the configured lookback period.

### Requirements

- Import historical playback data from Tautulli SQLite database files
- Resume capability for interrupted imports
- Deduplication to prevent duplicate events when data overlaps with API sync
- Progress tracking with real-time status updates
- Validation without import (dry run mode)
- Integration with existing NATS event processing pipeline

### Alternatives Considered

| Approach | Pros | Cons |
|----------|------|------|
| **Direct DuckDB Copy** | Fast, simple | No deduplication, no event processing |
| **ETL Tool (dbt)** | Mature tooling | External dependency, complex setup |
| **NATS Pipeline** | Deduplication built-in, reuses existing infrastructure | Requires NATS build tag |
| **Direct Database Insert** | Simple, no dependencies | Bypasses event processing, no deduplication |

---

## Decision

Use **DuckDB's sqlite_scanner extension** to read Tautulli databases directly and publish events through the **NATS JetStream pipeline** for triple-layer deduplication.

### Architecture

```
┌─────────────────┐     ┌──────────────────┐     ┌─────────────────┐
│  Tautulli.db    │────▶│  DuckDB Reader   │────▶│  Record Mapper  │
│  (SQLite)       │     │  sqlite_scanner  │     │                 │
└─────────────────┘     └──────────────────┘     └────────┬────────┘
                                                          │
                                                          ▼
                                                 ┌─────────────────┐
                                                 │  NATS Publisher │
                                                 │  (EventID +     │
                                                 │   CorrelationKey)│
                                                 └────────┬────────┘
                                                          │
                        ┌─────────────────────────────────┘
                        │
                        ▼
          ┌──────────────────────────┐
          │  NATS JetStream          │
          │  (Deduplication Layer 1) │
          │  MsgId = EventID         │
          └────────────┬─────────────┘
                       │
                       ▼
          ┌──────────────────────────┐
          │  DuckDB Consumer         │
          │  (Deduplication Layer 2) │
          │  Cache: EventID +        │
          │         SessionKey +     │
          │         CorrelationKey   │
          └────────────┬─────────────┘
                       │
                       ▼
          ┌──────────────────────────┐
          │  DuckDB Database         │
          │  (Deduplication Layer 3) │
          │  UNIQUE(correlation_key) │
          │  UNIQUE(rating_key,      │
          │    user_id, started_at)  │
          └──────────────────────────┘
```

### Triple-Layer Deduplication Strategy

The import system uses a three-layer deduplication strategy to handle all scenarios:

**Layer 1 - NATS JetStream MsgId (Same-Source Deduplication)**
- Uses deterministic EventID as NATS MsgId
- EventID = SHA256(`tautulli-import:{session_key}:{started_at}:{user_id}`)[:16]
- Prevents exact duplicate messages from same source
- JetStream dedup window configured per stream

**Layer 2 - Consumer Memory Cache (Cross-Source Deduplication)**
- Checks EventID, SessionKey, AND CorrelationKey
- CorrelationKey format: `{source}:{server_id}:{user_id}:{rating_key}:{machine_id}:{time_bucket}:{session_key}` (v2.3)
- Time bucket uses exact second precision (prevents false deduplication across time)
- 5-minute deduplication window, max 10,000 entries
- Cross-source dedup extracts content-based portion to match across Plex/Tautulli/Jellyfin

**Layer 3 - DuckDB Unique Indexes (Persistent Deduplication)**
- `idx_playback_correlation_key`: UNIQUE(correlation_key) - Primary cross-source dedup
- `idx_playback_dedup`: UNIQUE(rating_key, user_id, started_at) - Legacy dedup
- `ON CONFLICT DO NOTHING` silently skips duplicates (DuckDB-native syntax)
- Survives consumer restarts and cache clears

### Key Factors

1. **Triple-Layer Deduplication**: NATS MsgId (same-source), Consumer cache (cross-source), DuckDB indexes (persistent)
2. **Batch Processing**: Configurable batch size (1000-5000 recommended)
3. **Resumable**: Progress persisted to BadgerDB or in-memory
4. **Correlation Keys**: `{source}:{server_id}:{user_id}:{rating_key}:{machine_id}:{time_bucket}:{session_key}` (v2.3 format)
5. **Build Tag Isolation**: Requires `nats` build tag to use NATS infrastructure

---

## Consequences

### Positive

- **Historical Data**: Import years of Tautulli history in minutes
- **Deduplication**: Same event from API and import won't duplicate
- **Resume Capability**: Large imports can be interrupted and resumed
- **Validation**: Dry run mode validates before committing
- **Reuses Infrastructure**: Leverages existing NATS event pipeline

### Negative

- **NATS Required**: Cannot import without NATS build tag
- **Memory Usage**: Large batches require more memory
- **Processing Time**: Events go through full pipeline (not direct insert)

### Neutral

- **Geolocation**: Import skips geolocation (IP addresses preserved for later lookup)
- **Optional Feature**: Disabled by default (`IMPORT_ENABLED=false`)

---

## Implementation

### SQLite Reader

```go
// internal/import/sqlite_reader.go
type SQLiteReader struct {
    db     *sql.DB
    dbPath string
}

func NewSQLiteReader(dbPath string) (*SQLiteReader, error) {
    // Use DuckDB with sqlite_scanner extension
    db, err := sql.Open("duckdb", "")
    if err != nil {
        return nil, err
    }

    // Load sqlite_scanner and attach database
    db.Exec("INSTALL sqlite_scanner; LOAD sqlite_scanner;")
    db.Exec("CALL sqlite_attach(?)", dbPath)

    return &SQLiteReader{db: db, dbPath: dbPath}, nil
}

func (r *SQLiteReader) ReadBatch(ctx context.Context, sinceID int64, limit int) ([]TautulliRecord, error) {
    // JOIN session_history, session_history_metadata, session_history_media_info
    // Return records ordered by ID for resumability
}
```

### Record Mapper

```go
// internal/import/mapper.go
func (m *Mapper) ToPlaybackEvents(records []TautulliRecord) []*models.PlaybackEvent {
    // Map Tautulli record fields to PlaybackEvent
    // Generate correlation key for deduplication
    // Handle media type variations (movie, episode, track)
}

// generateCorrelationKey creates a correlation key for cross-source deduplication.
// v2.3 Format: {source}:{server_id}:{user_id}:{rating_key}:{machine_id}:{time_bucket}:{session_key}
func (m *Mapper) generateCorrelationKey(rec *TautulliRecord) string {
    timeBucket := rec.StartedAt.UTC().Format("2006-01-02T15:04:05")
    return fmt.Sprintf("%s:%s:%d:%s:%s:%s:%s",
        m.source,     // "tautulli-import"
        "default",    // serverID
        rec.UserID,
        ratingKey,    // rec.RatingKey or title hash
        machineID,    // rec.MachineID or "unknown"
        timeBucket,
        rec.SessionKey)
}
```

### Importer

```go
// internal/import/importer.go
type Importer struct {
    cfg       *config.ImportConfig
    publisher EventPublisher
    progress  ProgressTracker
}

func (i *Importer) Import(ctx context.Context) (*ImportStats, error) {
    reader, err := NewSQLiteReader(i.cfg.DBPath)
    if err != nil {
        return nil, err
    }
    defer reader.Close()

    // Resume from last progress or start fresh
    startID := i.getStartID(ctx)

    for {
        records, err := reader.ReadBatch(ctx, startID, i.cfg.BatchSize)
        if len(records) == 0 {
            break
        }

        // Convert and publish through NATS
        events := i.mapper.ToPlaybackEvents(records)
        for _, event := range events {
            i.publisher.PublishEvent(ctx, event)
        }

        // Update progress
        i.progress.Save(ctx, stats)
    }
}
```

### API Handlers

```go
// internal/api/handlers_import.go
func (h *ImportHandlers) HandleStartImport(w http.ResponseWriter, r *http.Request) {
    if h.importer.IsRunning() {
        // Return 409 Conflict with current progress
    }

    go h.importer.Import(context.Background())
    // Return 200 with import started message
}

func (h *ImportHandlers) HandleGetImportStatus(w http.ResponseWriter, r *http.Request) {
    stats := h.importer.GetStats()
    // Return progress percentage, records/sec, ETA
}
```

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `IMPORT_ENABLED` | `false` | Enable import functionality |
| `IMPORT_DB_PATH` | - | Path to Tautulli SQLite database |
| `IMPORT_BATCH_SIZE` | `1000` | Records per batch |
| `IMPORT_DRY_RUN` | `false` | Validate without importing |
| `IMPORT_AUTO_START` | `false` | Start import on application start |

### Code References

| Component | File | Notes |
|-----------|------|-------|
| SQLite Reader | `internal/import/sqlite_reader.go` | DuckDB sqlite_scanner (TautulliRecord struct included) |
| Record Mapper | `internal/import/mapper.go` | Tautulli to PlaybackEvent (v2.3 correlation key) |
| Core Importer | `internal/import/importer.go` | Orchestrates import (nats build tag) |
| Progress Tracker | `internal/import/progress.go` | BadgerDB/memory storage |
| Types | `internal/import/types.go` | ImportStats, ProgressSummary |
| API Handlers | `internal/api/handlers_import.go` | REST endpoints (nats build tag) |
| API Routes | `internal/api/router_import.go` | Route registration (nats build tag) |
| Service | `internal/supervisor/services/import_service.go` | Suture integration (nats build tag) |
| Config | `internal/config/config.go` | ImportConfig struct (lines 536-564) |
| Init | `cmd/server/import_init.go` | Application wiring (nats build tag) |

---

## Verification

### Verified Claims

| Claim | Source | Verified |
|-------|--------|----------|
| sqlite_scanner extension | `internal/import/sqlite_reader.go:131` | Yes |
| Deterministic EventID generation | `internal/import/mapper.go:87-110` | Yes |
| CorrelationKey format (v2.3) | `internal/import/mapper.go:121-152` | Yes |
| NATS MsgId = msg.UUID | `internal/eventprocessor/publisher.go:101-104` | Yes |
| Consumer cache checks CorrelationKey | `internal/eventprocessor/duckdb_consumer.go:386-390` | Yes |
| DuckDB UNIQUE(correlation_key) | `internal/database/database_schema.go:960` | Yes |
| DuckDB UNIQUE(rating_key, user_id, started_at) | `internal/database/database_schema.go:955` | Yes |
| ON CONFLICT DO NOTHING | `internal/database/crud_playback.go:35,120` | Yes |
| Build tag: nats | `internal/import/importer.go:1` | Yes |

### Test Coverage

- SQLite reader tests: `internal/import/sqlite_reader_test.go`
- Mapper tests: `internal/import/mapper_test.go`
- Importer tests: `internal/import/importer_test.go`
- Progress tests: `internal/import/progress_test.go`
- Testcontainer tests: `internal/import/tautulli_container_test.go`
- Handler tests: `internal/api/handlers_import_test.go`
- Service tests: `internal/supervisor/services/import_service_test.go`
- Integration tests: `internal/import/integration_test.go`
- Coverage target: 90%+ for import package

---

## Related ADRs

- [ADR-0001](0001-use-duckdb-for-analytics.md): DuckDB for data storage
- [ADR-0005](0005-nats-jetstream-event-processing.md): NATS event pipeline
- [ADR-0006](0006-badgerdb-write-ahead-log.md): Progress persistence
- [ADR-0007](0007-event-sourcing-architecture.md): Event deduplication strategy

---

## References

- [DuckDB SQLite Scanner](https://duckdb.org/docs/extensions/sqlite_scanner.html)
- [Tautulli Database Schema](https://github.com/Tautulli/Tautulli/wiki/Database-Tables)
- [NATS JetStream Deduplication](https://docs.nats.io/nats-concepts/jetstream/consumers#deduplication)
