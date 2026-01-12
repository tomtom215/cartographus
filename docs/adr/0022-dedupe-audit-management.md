# ADR-0022: Deduplication Audit and Management System

**Status**: Accepted
**Date**: 2025-12-20
**Context**: Users need visibility into automated deduplication decisions and the ability to undo incorrect deduplication.

## Context

The event processing pipeline includes multiple deduplication layers:
1. ExactLRU in-memory cache (EventID, SessionKey, CorrelationKey) - O(1) operations with zero false positives
2. NATS JetStream message deduplication (2-minute window)
3. DuckDB unique constraints (correlation_key, rating_key+user_id+started_at)

Currently, when an event is deduplicated:
- No record is kept of the decision
- The original event data is discarded
- Users have no visibility into why data might be "missing"
- There's no way to recover incorrectly deduplicated events

This is similar to Plex's metadata matching problem, where automated agents match files to IMDB/TVDB records but sometimes get it wrong. Plex provides an "Unmatch" feature to manually correct mistakes.

## Decision

Implement a comprehensive deduplication audit and management system with:

### 1. Dedupe Audit Log Table

**Location**: `internal/database/database_schema.go`

```sql
CREATE TABLE IF NOT EXISTS dedupe_audit_log (
    id UUID PRIMARY KEY,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    -- The event that was deduplicated (discarded)
    discarded_event_id TEXT NOT NULL,
    discarded_session_key TEXT,
    discarded_correlation_key TEXT,
    discarded_source TEXT NOT NULL,
    discarded_started_at TIMESTAMPTZ,
    discarded_raw_payload JSON,  -- Full event data for recovery

    -- The event that it was matched against (kept)
    matched_event_id TEXT,
    matched_session_key TEXT,
    matched_correlation_key TEXT,
    matched_source TEXT,

    -- Deduplication details
    dedupe_reason TEXT NOT NULL,   -- 'event_id', 'session_key', 'correlation_key', 'cross_source_key', 'db_constraint'
    dedupe_layer TEXT NOT NULL,    -- 'bloom_cache', 'nats_dedup', 'db_unique'
    similarity_score DOUBLE,       -- For fuzzy matching scenarios

    -- User information
    user_id INTEGER NOT NULL,
    username TEXT,

    -- Media information
    media_type TEXT,
    title TEXT,
    rating_key TEXT,

    -- Resolution status
    status TEXT NOT NULL DEFAULT 'auto_dedupe',  -- 'auto_dedupe', 'user_confirmed', 'user_restored'
    resolved_by TEXT,              -- Username who resolved
    resolved_at TIMESTAMPTZ,
    resolution_notes TEXT,

    -- Audit timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Indexes (created separately in database_schema.go)
CREATE INDEX IF NOT EXISTS idx_dedupe_audit_timestamp ON dedupe_audit_log(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_dedupe_audit_user_id ON dedupe_audit_log(user_id);
CREATE INDEX IF NOT EXISTS idx_dedupe_audit_status ON dedupe_audit_log(status);
CREATE INDEX IF NOT EXISTS idx_dedupe_audit_discarded ON dedupe_audit_log(discarded_event_id);
CREATE INDEX IF NOT EXISTS idx_dedupe_audit_source ON dedupe_audit_log(discarded_source);
CREATE INDEX IF NOT EXISTS idx_dedupe_audit_reason ON dedupe_audit_log(dedupe_reason);
```

### 2. API Endpoints

**Location**: `internal/api/chi_router.go` (registerChiDedupeRoutes)

**Handlers**: `internal/api/handlers_dedupe.go`

```
GET  /api/v1/dedupe/audit                    # List all dedupe events (paginated)
GET  /api/v1/dedupe/audit/{id}               # Get specific dedupe event details
GET  /api/v1/dedupe/audit/stats              # Dedupe statistics dashboard
POST /api/v1/dedupe/audit/{id}/restore       # Restore a deduplicated event
POST /api/v1/dedupe/audit/{id}/confirm       # Confirm dedup was correct
GET  /api/v1/dedupe/audit/export             # Export audit log to CSV

# Filtering (query parameters)
GET  /api/v1/dedupe/audit?user_id=123        # Filter by user
GET  /api/v1/dedupe/audit?source=tautulli    # Filter by source
GET  /api/v1/dedupe/audit?status=auto_dedupe # Filter by status
GET  /api/v1/dedupe/audit?reason=cross_source_key # Filter by reason
GET  /api/v1/dedupe/audit?layer=bloom_cache  # Filter by dedup layer
GET  /api/v1/dedupe/audit?from=2025-01-01T00:00:00Z&to=2025-12-31T23:59:59Z # Date range (RFC3339)
GET  /api/v1/dedupe/audit?limit=100&offset=0 # Pagination
```

All endpoints require authentication and are rate-limited.

### 3. UI Components

**Location**: `web/src/app/DedupeAuditManager.ts`

**Types**: `web/src/lib/types/dedupe.ts`

#### Dedupe Dashboard (`/settings/deduplication`)

```
+------------------------------------------------------------------+
|  Deduplication Management                                         |
+------------------------------------------------------------------+
|                                                                   |
|  Summary (Last 30 days)                                          |
|  +------------+  +------------+  +------------+  +------------+   |
|  | 1,234      |  | 45         |  | 12         |  | 98.5%      |   |
|  | Total      |  | Pending    |  | Restored   |  | Accuracy   |   |
|  | Deduped    |  | Review     |  | by User    |  | Rate       |   |
|  +------------+  +------------+  +------------+  +------------+   |
|                                                                   |
|  Audit Log                                           [Export CSV] |
|  +--------------------------------------------------------------+|
|  | Time    | User    | Title        | Reason      | Status  |   ||
|  |---------|---------|--------------|-------------|---------|---||
|  | 2m ago  | john    | Inception    | cross_src   | ⚠ Review|[↩]||
|  | 5m ago  | sarah   | Breaking Bad | session_key | ✓ Auto  |[↩]||
|  | 1h ago  | mike    | Frozen II    | correlation | ✓ User  |   ||
|  +--------------------------------------------------------------+|
|                                                                   |
+------------------------------------------------------------------+
```

#### Dedupe Detail Modal

```
+------------------------------------------------------------------+
|  Deduplication Details                                      [X]   |
+------------------------------------------------------------------+
|                                                                   |
|  Discarded Event                    Matched Against               |
|  +----------------------------+    +----------------------------+ |
|  | Event ID: abc-123          |    | Event ID: xyz-789          | |
|  | Source: tautulli           |    | Source: plex               | |
|  | Session: tautulli-456      |    | Session: plex-ws-123       | |
|  | Started: 2025-12-20 10:30  |    | Started: 2025-12-20 10:30  | |
|  | User: john (ID: 42)        |    | User: john (ID: 42)        | |
|  | Title: Inception           |    | Title: Inception           | |
|  | Device: Living Room Roku   |    | Device: Living Room Roku   | |
|  +----------------------------+    +----------------------------+ |
|                                                                   |
|  Deduplication Reason: cross_source_key                          |
|  Deduplication Layer: bloom_cache                                 |
|  Correlation Key Match: default:42:1018:abc123:2025-12-20T10:30  |
|                                                                   |
|  [View Raw Payload]                                               |
|                                                                   |
|  Actions:                                                         |
|  +----------------------------+    +----------------------------+ |
|  | [✓ Confirm Correct]        |    | [↩ Restore Event]          | |
|  +----------------------------+    +----------------------------+ |
|                                                                   |
|  Resolution Notes (optional):                                     |
|  +--------------------------------------------------------------+|
|  |                                                               ||
|  +--------------------------------------------------------------+|
|                                                                   |
+------------------------------------------------------------------+
```

### 4. Handler Integration

**Location**: `internal/eventprocessor/handlers.go`

The `DuckDBHandler` includes dedupe audit logging via the `isDuplicateWithAudit()` method:

```go
// DuckDBHandler processes media events for DuckDB persistence.
// It handles deserialization, cross-source deduplication, and batch appending.
type DuckDBHandler struct {
    appender   *Appender
    config     DuckDBHandlerConfig
    logger     watermill.LoggerAdapter
    auditStore DedupeAuditStore // Optional: for logging dedupe decisions (ADR-0022)
    dedupCache cache.DeduplicationCache // ExactLRU with zero false positives
    // ... metrics fields
}

// isDuplicateWithAudit checks if an event has been seen recently and logs audit entries.
// Checks EventID, SessionKey, and CorrelationKey for cross-source deduplication.
func (h *DuckDBHandler) isDuplicateWithAudit(event *MediaEvent, rawPayload []byte) bool {
    // Check EventID (primary dedup key)
    if h.dedupCache.IsDuplicate(event.EventID) {
        h.logDedupeDecision(event, rawPayload, "event_id")
        return true
    }

    // Check SessionKey if different from EventID
    if event.SessionKey != "" && event.SessionKey != event.EventID {
        if h.dedupCache.Contains(event.SessionKey) {
            h.logDedupeDecision(event, rawPayload, "session_key")
            return true
        }
    }

    // Check CorrelationKey for same-source deduplication
    if event.CorrelationKey != "" {
        if h.dedupCache.Contains("corr:" + event.CorrelationKey) {
            h.logDedupeDecision(event, rawPayload, "correlation_key")
            return true
        }

        // Check cross-source key for cross-source deduplication
        crossSourceKey := getCrossSourceKey(event.CorrelationKey)
        if crossSourceKey != "" {
            // ... check against other sources
            h.logDedupeDecision(event, rawPayload, "cross_source_key")
            return true
        }
    }

    return false
}

// logDedupeDecision logs a deduplication decision to the audit store asynchronously.
func (h *DuckDBHandler) logDedupeDecision(event *MediaEvent, rawPayload []byte, reason string) {
    if !h.config.EnableDedupeAudit || h.auditStore == nil {
        return
    }

    entry := &models.DedupeAuditEntry{
        ID:                      uuid.New(),
        Timestamp:               time.Now(),
        DiscardedEventID:        event.EventID,
        DiscardedSessionKey:     event.SessionKey,
        DiscardedCorrelationKey: event.CorrelationKey,
        DiscardedSource:         event.Source,
        DedupeReason:            reason,
        DedupeLayer:             "bloom_cache",  // Always bloom_cache from handler
        UserID:                  event.UserID,
        Username:                event.Username,
        MediaType:               event.MediaType,
        Title:                   event.Title,
        RatingKey:               event.RatingKey,
        Status:                  "auto_dedupe",
        CreatedAt:               time.Now(),
    }

    if h.config.StoreRawPayload && len(rawPayload) > 0 {
        entry.DiscardedRawPayload = rawPayload
    }

    // Insert asynchronously to avoid blocking message processing
    go func() {
        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()
        h.auditStore.InsertDedupeAuditEntry(ctx, entry)
    }()
}
```

The `DedupeAuditStore` interface allows the handler to work with any storage implementation:

```go
type DedupeAuditStore interface {
    InsertDedupeAuditEntry(ctx context.Context, entry *models.DedupeAuditEntry) error
}
```

### 5. Restore Functionality

**Location**: `internal/api/handlers_dedupe.go` (DedupeAuditRestore)

When a user restores a deduplicated event via POST `/api/v1/dedupe/audit/{id}/restore`:

```go
// DedupeAuditRestore handles POST /api/v1/dedupe/audit/{id}/restore
// Restores a deduplicated event (inserts it back into playback_events).
func (h *Handler) DedupeAuditRestore(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    id, err := uuid.Parse(r.PathValue("id"))
    if err != nil {
        respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid entry ID format", err)
        return
    }

    // 1. Get the audit entry to retrieve the raw payload
    entry, err := h.db.GetDedupeAuditEntry(ctx, id)
    if err != nil {
        respondError(w, http.StatusNotFound, "NOT_FOUND", "Dedupe audit entry not found", err)
        return
    }

    // 2. Check if already restored
    if entry.Status == "user_restored" {
        respondError(w, http.StatusConflict, "CONFLICT", "Event has already been restored", nil)
        return
    }

    // 3. Verify raw payload exists
    if len(entry.DiscardedRawPayload) == 0 {
        respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", "No raw payload available for restoration", nil)
        return
    }

    // 4. Unmarshal the raw payload into a PlaybackEvent
    var event models.PlaybackEvent
    if err := json.Unmarshal(entry.DiscardedRawPayload, &event); err != nil {
        respondError(w, http.StatusInternalServerError, "PARSE_ERROR", "Failed to parse stored event data", err)
        return
    }

    // 5. Generate new unique ID and correlation key to avoid re-deduplication
    event.ID = uuid.New()
    restoredFromAudit := id.String()
    event.CorrelationKey = &restoredFromAudit

    // 6. Insert the restored event
    if err := h.db.InsertPlaybackEvent(&event); err != nil {
        respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to restore event", err)
        return
    }

    // 7. Update the audit entry status
    h.db.UpdateDedupeAuditStatus(ctx, id, "user_restored", req.ResolvedBy, req.Notes)

    // Return success response with restored event ID
}
```

### 6. Configuration Options

**Location**: `internal/eventprocessor/handlers.go` (DuckDBHandlerConfig)

```go
// DuckDBHandlerConfig holds configuration for the DuckDB handler.
type DuckDBHandlerConfig struct {
    // EnableCrossSourceDedup enables correlation key deduplication
    // for detecting duplicate events from different sources (Plex, Tautulli, Jellyfin).
    EnableCrossSourceDedup bool

    // DeduplicationWindow is how long to remember correlation keys.
    DeduplicationWindow time.Duration

    // MaxDeduplicationEntries is the maximum cache size.
    MaxDeduplicationEntries int

    // EnableDedupeAudit enables logging of deduplication decisions (ADR-0022).
    // When enabled, each dedupe decision is recorded for visibility and recovery.
    EnableDedupeAudit bool

    // StoreRawPayload enables storing the full event payload in audit entries.
    // This allows restoration of incorrectly deduplicated events.
    // Uses more storage but enables full recovery capability.
    StoreRawPayload bool

    // SyncFlush forces synchronous flush after each append.
    SyncFlush bool
}

// DefaultDuckDBHandlerConfig returns production defaults.
func DefaultDuckDBHandlerConfig() DuckDBHandlerConfig {
    return DuckDBHandlerConfig{
        EnableCrossSourceDedup:  true,
        DeduplicationWindow:     5 * time.Minute,
        MaxDeduplicationEntries: 10000,
        EnableDedupeAudit:       true,  // Enable audit logging by default
        StoreRawPayload:         true,  // Store full payload for recovery
    }
}
```

**Database CRUD**: `internal/database/crud_dedupe.go`

Cleanup of resolved entries is available via `CleanupDedupeAuditEntries(ctx, retentionDays)` which removes entries with status `user_confirmed` or `user_restored` older than the specified retention period (default: 90 days).

## Consequences

### Positive
- Complete visibility into deduplication decisions
- Users can recover incorrectly deduplicated events
- Troubleshooting becomes precise and auditable
- Similar UX to Plex's "Unmatch" feature users are familiar with
- Helps identify and fix overly aggressive deduplication rules

### Negative
- Increased storage requirements (raw payload storage)
- Slight performance overhead for audit logging
- Additional database table and API complexity
- Need to handle audit log retention/cleanup

### Implementation Status

All phases have been implemented:

1. **Phase 1**: Audit log table + basic logging - `internal/database/database_schema.go`, `internal/eventprocessor/handlers.go`
2. **Phase 2**: API endpoints for query/restore - `internal/api/handlers_dedupe.go`
3. **Phase 3**: Settings page UI with dashboard - `web/src/app/DedupeAuditManager.ts`
4. **Phase 4**: Real-time notifications - Toast notifications integrated in UI

## Source Files

| Component | File Path |
|-----------|-----------|
| Database Schema | `internal/database/database_schema.go` |
| Database CRUD | `internal/database/crud_dedupe.go` |
| API Handlers | `internal/api/handlers_dedupe.go` |
| Router Registration | `internal/api/chi_router.go` (registerChiDedupeRoutes) |
| Event Handler | `internal/eventprocessor/handlers.go` |
| Go Models | `internal/models/playback.go` (DedupeAuditEntry, DedupeAuditStats) |
| Frontend Manager | `web/src/app/DedupeAuditManager.ts` |
| Frontend Types | `web/src/lib/types/dedupe.ts` |
| API Client | `web/src/lib/api/dedupe.ts` |

## Related ADRs
- ADR-0005: NATS JetStream Event Processing
- ADR-0006: BadgerDB Write-Ahead Log
- ADR-0007: Event Sourcing Architecture
- ADR-0023: Consumer-Side WAL for Exactly-Once Delivery
