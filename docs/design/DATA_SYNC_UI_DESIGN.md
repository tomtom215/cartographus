# Data Sync UI Design Document

**Version**: 1.0
**Date**: 2025-01-09
**Author**: Claude
**Status**: Implementation

## Overview

This document describes the design for the Data Sync UI feature, enabling users to initiate and monitor data synchronization from the web interface.

## Architecture Decisions

### 1. UI Location

**Decision**: Dedicated "Data Sync" section within the Settings panel.

**Rationale**:
- Settings panel already has modular sections (Theme, Visualization, etc.)
- Data sync is an admin operation that fits naturally with settings
- Avoids cluttering the main navigation
- Consistent with existing patterns (MultiServerManager lives in Settings area)

### 2. Real-Time Progress Communication

**Decision**: WebSocket with polling fallback.

**Design**:
- Add new WebSocket message type: `sync_progress`
- Push updates every 1-2 seconds during active operations
- Fallback to 3-second polling if WebSocket disconnected
- Reuse existing WebSocket hub infrastructure

**Message Format**:
```typescript
interface SyncProgressMessage {
    type: 'sync_progress';
    data: {
        operation: 'tautulli_import' | 'plex_historical' | 'server_sync';
        status: 'running' | 'completed' | 'error' | 'cancelled';
        server_id?: string;  // For server_sync operations
        progress: {
            total_records: number;
            processed_records: number;
            imported_records: number;
            skipped_records: number;
            error_count: number;
            progress_percent: number;
            records_per_second: number;
            elapsed_seconds: number;
            estimated_remaining_seconds: number;
        };
        message?: string;
        error?: string;
        correlation_id: string;
    };
}
```

### 3. State Management

**Decision**: `SyncManager` class with `SafeSessionStorage` for durability.

**Design**:
- Class-based with dependency injection (follows PlexPINAuthenticator pattern)
- State persisted to session storage for page refresh recovery
- Automatic cleanup of stale state (>5 minutes old)
- Correlation IDs for audit tracing

### 4. API Design

**New Backend Endpoints**:

```
POST /api/v1/sync/plex/historical
    Request: { days_back?: number, library_ids?: string[] }
    Response: { success: boolean, message: string, correlation_id: string }

GET /api/v1/sync/status
    Response: {
        tautulli_import?: SyncProgress,
        plex_historical?: SyncProgress,
        server_syncs?: Record<string, SyncProgress>
    }
```

**Existing Endpoints (already implemented)**:
```
POST /api/v1/import/tautulli       - Start Tautulli import
GET  /api/v1/import/status         - Get import status
DELETE /api/v1/import              - Stop import
POST /api/v1/import/validate       - Validate database file
DELETE /api/v1/import/progress     - Clear saved progress
```

## Component Hierarchy

```
SettingsManager
└── DataSyncSettingsSection
    ├── TautulliImportSection
    │   ├── SyncProgressComponent (reusable)
    │   └── ErrorLogComponent
    ├── PlexHistoricalSection
    │   ├── SyncProgressComponent (reusable)
    │   └── LibrarySelectorComponent
    └── ServerSyncStatusSection
        └── ServerSyncCard[] (per server)
```

## Implementation Files

### Backend (Go)

| File | Purpose |
|------|---------|
| `internal/api/handlers_sync.go` | New sync API handlers |
| `internal/websocket/hub.go` | Add sync_progress message type |
| `internal/sync/manager.go` | Add progress broadcasting |
| `internal/import/importer.go` | Add progress broadcasting |

### Frontend (TypeScript)

| File | Purpose |
|------|---------|
| `web/src/app/SyncManager.ts` | Core sync state management |
| `web/src/app/SyncProgressComponent.ts` | Reusable progress UI |
| `web/src/app/DataSyncSettingsSection.ts` | Settings integration |
| `web/src/lib/api/sync.ts` | Sync API client |
| `web/src/lib/types/sync.ts` | TypeScript interfaces |

## TypeScript Interfaces

```typescript
// web/src/lib/types/sync.ts

export interface SyncProgress {
    status: 'idle' | 'running' | 'completed' | 'error' | 'cancelled';
    total_records: number;
    processed_records: number;
    imported_records: number;
    skipped_records: number;
    error_count: number;
    progress_percent: number;
    records_per_second: number;
    elapsed_seconds: number;
    estimated_remaining_seconds: number;
    start_time?: string;
    last_processed_id?: number;
    dry_run?: boolean;
    errors?: SyncError[];
}

export interface SyncError {
    timestamp: string;
    record_id?: number;
    message: string;
    recoverable: boolean;
}

export interface TautulliImportRequest {
    db_path?: string;
    resume?: boolean;
    dry_run?: boolean;
}

export interface PlexHistoricalRequest {
    days_back?: number;
    library_ids?: string[];
}

export interface SyncStatusResponse {
    tautulli_import?: SyncProgress;
    plex_historical?: SyncProgress;
    server_syncs?: Record<string, SyncProgress>;
}
```

## UI Wireframes (Text-Based)

### Data Sync Section (Collapsed)

```
┌──────────────────────────────────────────────────────────┐
│ Data Sync                                            [v] │
│ ─────────────────────────────────────────────────────── │
│ Tautulli Import: Ready to import                        │
│ Plex Historical: Sync completed (2 days ago)            │
│ Server Sync: 3 servers connected                        │
└──────────────────────────────────────────────────────────┘
```

### Tautulli Import Section (Expanded, Idle)

```
┌──────────────────────────────────────────────────────────┐
│ Tautulli Database Import                                 │
│ ─────────────────────────────────────────────────────── │
│                                                          │
│ Database Path:                                           │
│ ┌──────────────────────────────────────────────────────┐ │
│ │ /config/tautulli.db                                  │ │
│ └──────────────────────────────────────────────────────┘ │
│                                                          │
│ Options:                                                 │
│ ☐ Resume from last position                             │
│ ☐ Dry run (validate without importing)                  │
│                                                          │
│ [Validate Database]  [Start Import]                      │
│                                                          │
│ Last Import: 2025-01-08 14:30 (50,000 records)          │
└──────────────────────────────────────────────────────────┘
```

### Import In Progress

```
┌──────────────────────────────────────────────────────────┐
│ Tautulli Database Import                                 │
│ ─────────────────────────────────────────────────────── │
│                                                          │
│ ████████████████████░░░░░░░░░  67.5%                    │
│                                                          │
│ Status: Importing records...                             │
│ Progress: 33,750 / 50,000 records                        │
│ Speed: 125 records/sec                                   │
│ Elapsed: 4m 30s                                          │
│ Remaining: ~2m 10s                                       │
│                                                          │
│ Imported: 33,500  Skipped: 200  Errors: 50              │
│                                                          │
│ [Stop Import]                                            │
│                                                          │
│ ▼ Errors (50)                                            │
│ ├─ Record 12345: Invalid timestamp format                │
│ ├─ Record 12380: Missing required field 'ip_address'    │
│ └─ ... 48 more                                           │
└──────────────────────────────────────────────────────────┘
```

### Plex Historical Sync Section

```
┌──────────────────────────────────────────────────────────┐
│ Plex Historical Sync                                     │
│ ─────────────────────────────────────────────────────── │
│                                                          │
│ Sync playback history from Plex directly.                │
│                                                          │
│ Days to sync back: [30 ▼]                                │
│                                                          │
│ Libraries (optional):                                    │
│ ☑ Movies                                                 │
│ ☑ TV Shows                                               │
│ ☐ Music                                                  │
│                                                          │
│ [Start Historical Sync]                                  │
│                                                          │
│ Note: Cannot run while Tautulli import is active.        │
└──────────────────────────────────────────────────────────┘
```

## Security Considerations

1. **RBAC Enforcement**: All sync operations require admin role
2. **Path Validation**: Database paths validated against allowed directories
3. **Rate Limiting**: Sync triggers rate-limited to prevent abuse
4. **CSRF Protection**: All POST requests include CSRF tokens
5. **Audit Logging**: All sync operations logged with correlation IDs

## Accessibility (WCAG 2.1 AA)

1. **Progress Bar**:
   - `role="progressbar"` with `aria-valuenow`, `aria-valuemin`, `aria-valuemax`
   - Screen reader announcements on status changes

2. **Controls**:
   - All buttons have accessible names
   - Focus management when modals open/close
   - Keyboard navigable (Tab, Enter, Escape)

3. **Status Updates**:
   - `aria-live="polite"` region for progress updates
   - `aria-live="assertive"` for errors

## Error Handling

| Error Type | User Message | Recovery Action |
|------------|--------------|-----------------|
| Network failure | "Connection lost. Retrying..." | Auto-retry with backoff |
| Import conflict | "Another import is already running" | Show current progress |
| Invalid path | "Database file not found" | Allow path correction |
| Permission denied | "Access denied to database file" | Show required permissions |
| Timeout | "Operation timed out" | Offer resume option |

## Testing Strategy

### Unit Tests
- SyncManager state transitions
- Progress calculation (ETA, rate)
- Error handling and recovery
- State persistence/recovery

### Integration Tests
- WebSocket message handling
- API endpoint responses
- State sync across tabs

### E2E Tests
- Full import flow (start → progress → complete)
- Stop and resume import
- Error display and recovery
- Page refresh during import

## Implementation Order

1. **Phase 1**: Backend enhancements
   - Add `sync_progress` WebSocket message type
   - Add progress broadcasting to importer
   - Add `POST /api/v1/sync/plex/historical` endpoint

2. **Phase 2**: Core frontend
   - Create TypeScript interfaces
   - Implement SyncManager class
   - Implement SyncProgressComponent

3. **Phase 3**: Settings integration
   - Create DataSyncSettingsSection
   - Wire up WebSocket listener
   - Add to SettingsManager

4. **Phase 4**: Testing
   - Unit tests
   - E2E tests
   - Documentation updates
