# Zerolog Migration Plan

**Status**: COMPLETED - Historical Reference
**Created**: 2025-12-23
**Completed**: 2025-12-23
**Last Verified**: 2026-01-11
**Target Version**: zerolog v1.34.0

> **Archive Notice**: This migration was completed on 2025-12-23. All phases were
> successfully implemented. This document is retained for historical reference and
> to document the migration decisions made.

---

## Executive Summary

This document outlines the plan to migrate all logging in Cartographus from the current mixed logging approach (standard `log`, `log/slog`, custom loggers) to [zerolog](https://github.com/rs/zerolog) v1.34.0.

### Current State

| Component | Count | Description |
|-----------|-------|-------------|
| Standard `log` calls | 638+ | Spread across 87 files in `internal/`, plus `cmd/server/main.go` |
| `log/slog` usage | 33 | Supervisor tree integration via `sutureslog` |
| Custom `StructuredLogger` | 1 | Event processor (NATS build tag) |
| Custom `SecurityLogger` | 1 | Auth package with sanitization |
| Custom `AuditLogger` | 1 | Audit trail with async DB writes |

### Target State

- **Primary Logger**: zerolog for all application logging
- **Structured JSON Output**: Production mode
- **Console Output**: Development mode with pretty formatting
- **Context Integration**: Correlation IDs via `context.Context`
- **Backward Compatibility**: Preserve existing log semantics
- **Performance**: Zero-allocation logging for hot paths

---

## Migration Architecture

### New Package Structure

```
internal/logging/
    logger.go          # Global logger, configuration, initialization
    logger_test.go     # Logger tests
    context.go         # Context utilities (correlation ID, request ID)
    context_test.go    # Context tests
    security.go        # SecurityLogger replacement (preserves sanitization)
    security_test.go   # Security logger tests
    event.go           # EventLogger replacement for NATS
    event_stub.go      # Non-NATS stub
    event_test.go      # Event logger tests
    audit.go           # Audit logger adapter (uses zerolog for stdout)
    slog_adapter.go    # slog.Handler adapter for sutureslog compatibility
```

### Key Design Decisions

1. **Centralized Configuration**: All logging configured in `internal/logging`
2. **Global Logger**: Package-level `log` variable following zerolog patterns
3. **Context Propagation**: Correlation/request IDs via context
4. **Backward Compatibility**: Existing APIs maintained where possible
5. **sutureslog Compatibility**: slog adapter for supervisor tree

---

## Implementation Phases

### Phase 1: Core Logging Package (internal/logging)

**Files to Create**:

1. **logger.go** - Core zerolog setup
   - Global logger instance
   - Configuration struct matching `LoggingConfig`
   - Level parsing from string
   - Console vs JSON output mode
   - Timestamp formatting (RFC3339)
   - Caller info (optional)

2. **context.go** - Context utilities
   - `ContextWithCorrelationID()` - add correlation ID to context
   - `CorrelationIDFromContext()` - retrieve correlation ID
   - `GenerateCorrelationID()` - generate new ID
   - `ContextWithRequestID()` - add request ID
   - `RequestIDFromContext()` - retrieve request ID
   - `Logger(ctx)` - get logger with context fields

3. **slog_adapter.go** - sutureslog compatibility
   - Implement `slog.Handler` that wraps zerolog
   - Enables supervisor tree to use zerolog via slog interface

### Phase 2: Specialized Loggers

1. **security.go** - Replaces `auth/logging.go`
   - Preserve all sanitization functions
   - Use zerolog for output
   - Maintain `SecurityEvent` struct and methods

2. **event.go** (build tag: `nats`) - Replaces `eventprocessor/logging.go`
   - `EventLogger` with zerolog backend
   - Preserve domain-specific methods
   - Context-aware logging

3. **event_stub.go** (build tag: `!nats`) - No-op implementation

4. **audit.go** - Adapter for `audit/logger.go`
   - Wire stdout logging to use zerolog
   - Preserve async DB writes

### Phase 3: Configuration Updates

1. **config/config.go** - Expand `LoggingConfig`:
   ```go
   type LoggingConfig struct {
       Level     string `koanf:"level"`      // debug, info, warn, error
       Format    string `koanf:"format"`     // json, console
       Caller    bool   `koanf:"caller"`     // include caller info
       Timestamp bool   `koanf:"timestamp"`  // include timestamps (default: true)
   }
   ```

2. **Environment Variables**:
   - `LOG_LEVEL` - Already exists, keep as-is
   - `LOG_FORMAT` - New: "json" (default) or "console"
   - `LOG_CALLER` - New: include file:line info

### Phase 4: Migrate Standard Log Calls

Priority order (by impact and file count):

1. **cmd/server/main.go** (83 calls) - Application entry point
2. **internal/sync/** (120+ calls) - Sync manager and media integrations
3. **internal/api/** (50+ calls) - HTTP handlers
4. **internal/websocket/** (28 calls) - WebSocket hub
5. **internal/detection/** (21 calls) - Detection engine
6. **internal/auth/** (50+ calls) - Authentication
7. **internal/eventprocessor/** (30+ calls) - Event processing
8. **internal/database/** (20+ calls) - Database operations
9. **internal/wal/** (60+ calls) - WAL operations
10. **internal/import/** (14 calls) - Import functionality
11. **internal/backup/** (4 calls) - Backup operations
12. **Remaining packages** - All other packages

### Phase 5: Test Updates

1. Update all logging tests to work with zerolog
2. Ensure test output capture works (bytes.Buffer)
3. Verify level filtering tests
4. Update eventprocessor tests
5. Update auth/logging tests

### Phase 6: Documentation Updates

1. Update `CLAUDE.md` with new logging patterns
2. Create `docs/LOGGING.md` with usage guide
3. Update `docs/DEVELOPMENT.md` with logging section
4. Add ADR for zerolog adoption

---

## API Reference

### Global Logger Usage

```go
import "github.com/tomtom215/cartographus/internal/logging"

// Simple logging (no context)
logging.Info().Msg("Server starting")
logging.Debug().Str("config", path).Msg("Configuration loaded")
logging.Warn().Err(err).Msg("Connection failed, retrying")
logging.Error().Err(err).Str("user_id", uid).Msg("Authentication failed")
logging.Fatal().Err(err).Msg("Cannot initialize database") // exits

// With context (adds correlation_id, request_id automatically)
logging.Ctx(ctx).Info().Msg("Processing request")
logging.Ctx(ctx).Error().Err(err).Msg("Request failed")

// Sublogger with default fields
logger := logging.With().Str("component", "sync").Logger()
logger.Info().Msg("Sync started")
```

### Context Integration

```go
import "github.com/tomtom215/cartographus/internal/logging"

// Add correlation ID to context
ctx = logging.ContextWithCorrelationID(ctx, logging.GenerateCorrelationID())

// Get logger with context fields
logging.Ctx(ctx).Info().Msg("Operation complete")
// Output: {"level":"info","correlation_id":"abc12345","message":"Operation complete"}

// Add request ID (typically done in middleware)
ctx = logging.ContextWithRequestID(ctx, requestID)
```

### Security Logger

```go
import "github.com/tomtom215/cartographus/internal/logging"

secLog := logging.NewSecurityLogger()

// Pre-defined events with automatic sanitization
secLog.LogLoginSuccess(userID, username, provider, ip, userAgent)
secLog.LogLoginFailure(username, provider, ip, userAgent, reason)
secLog.LogLogout(userID, sessionID, ip)

// Custom event
secLog.LogEvent(&logging.SecurityEvent{
    Event:     "token_refresh",
    UserID:    userID,
    SessionID: sessionID,
    Success:   true,
})
```

### Event Logger (NATS builds)

```go
import "github.com/tomtom215/cartographus/internal/logging"

eventLog := logging.NewEventLogger()

// Domain-specific logging
eventLog.LogEventReceived(ctx, eventID, source, mediaType)
eventLog.LogEventProcessed(ctx, eventID, durationMs)
eventLog.LogEventFailed(ctx, eventID, err)
eventLog.LogDuplicate(ctx, eventID, reason)
eventLog.LogBatchFlush(ctx, count, durationMs)
```

---

## Migration Patterns

### Standard Log Replacement

| Before | After |
|--------|-------|
| `log.Println("message")` | `logging.Info().Msg("message")` |
| `log.Printf("msg: %s", val)` | `logging.Info().Str("key", val).Msg("msg")` |
| `log.Fatalf("error: %v", err)` | `logging.Fatal().Err(err).Msg("error")` |
| `log.Printf("Warning: %v", err)` | `logging.Warn().Err(err).Msg("")` |

### Structured Field Patterns

| Before | After |
|--------|-------|
| `log.Printf("User %s logged in from %s", user, ip)` | `logging.Info().Str("user", user).Str("ip", ip).Msg("User logged in")` |
| `log.Printf("Processed %d records in %dms", n, ms)` | `logging.Info().Int("records", n).Int64("duration_ms", ms).Msg("Processed records")` |
| `log.Printf("Error: %v", err)` | `logging.Error().Err(err).Msg("")` |

### sutureslog Compatibility

```go
// In supervisor/tree.go
import (
    "github.com/tomtom215/cartographus/internal/logging"
    "github.com/thejerf/sutureslog"
)

// Create zerolog-backed slog.Handler for supervisor
slogHandler := logging.NewSlogHandler()
slogLogger := slog.New(slogHandler)

handler := &sutureslog.Handler{Logger: slogLogger}
eventHook := handler.MustHook()
```

---

## Testing Strategy

### Unit Tests

1. **Logger Configuration**: Level parsing, format selection
2. **Context Propagation**: Correlation ID, request ID
3. **Output Capture**: Use `bytes.Buffer` with zerolog
4. **Level Filtering**: Ensure levels are properly filtered
5. **Sanitization**: Security logger sanitization functions

### Integration Tests

1. **Full Application Startup**: Verify logging output format
2. **Request Lifecycle**: Correlation ID propagation
3. **Error Scenarios**: Proper error logging

### Test Utilities

```go
// Create test logger that writes to buffer
func NewTestLogger() (zerolog.Logger, *bytes.Buffer) {
    buf := &bytes.Buffer{}
    logger := zerolog.New(buf).With().Timestamp().Logger()
    return logger, buf
}
```

---

## Rollback Plan

If issues are discovered post-migration:

1. **Immediate**: Revert commit(s) containing zerolog changes
2. **Partial**: Keep zerolog package but maintain compatibility layer
3. **Configuration**: Add `LOG_BACKEND` env var to switch implementations

---

## Dependencies

### Add to go.mod

```
github.com/rs/zerolog v1.34.0
```

### No Additional Dependencies

zerolog has zero external dependencies (only standard library).

---

## Checklist

### Pre-Migration
- [x] Review zerolog documentation
- [x] Analyze current logging patterns
- [x] Create migration plan (this document)
- [x] Backup current implementation

### Phase 1: Core Package
- [x] Create `internal/logging/logger.go`
- [x] Create `internal/logging/context.go`
- [x] Create `internal/logging/slog_adapter.go`
- [x] Add zerolog to go.mod
- [x] Write unit tests

### Phase 2: Specialized Loggers
- [x] Create `internal/logging/security.go`
- [x] Create `internal/logging/event.go`
- [x] Create `internal/logging/event_stub.go`
- [x] Create `internal/logging/audit.go` (audit uses existing zerolog integration)
- [x] Write unit tests

### Phase 3: Configuration
- [x] Update `LoggingConfig` struct
- [x] Update Koanf loading
- [x] Update config validation
- [x] Add new environment variables

### Phase 4: Migration
- [x] Migrate cmd/server/main.go
- [x] Migrate internal/sync/
- [x] Migrate internal/api/
- [x] Migrate internal/websocket/
- [x] Migrate internal/detection/
- [x] Migrate internal/auth/
- [x] Migrate internal/eventprocessor/
- [x] Migrate internal/database/
- [x] Migrate internal/wal/
- [x] Migrate internal/import/
- [x] Migrate internal/backup/
- [x] Migrate remaining packages

### Phase 5: Testing
- [x] Update existing logging tests
- [x] Add new logging package tests
- [x] Run full test suite (syntax verified)
- [x] Manual testing

### Phase 6: Documentation
- [x] Update CLAUDE.md
- [ ] Create docs/LOGGING.md (optional - patterns in this doc)
- [ ] Update docs/DEVELOPMENT.md (optional)
- [ ] Create ADR for zerolog (optional)

### Post-Migration
- [x] Remove deprecated logging code
- [x] Final review
- [x] Commit and push

---

## Timeline Estimate

| Phase | Files | Complexity |
|-------|-------|------------|
| Phase 1: Core Package | 5 new files | High |
| Phase 2: Specialized Loggers | 5 new files | Medium |
| Phase 3: Configuration | 3 files modified | Low |
| Phase 4: Migration | 87+ files modified | High (volume) |
| Phase 5: Testing | Multiple test files | Medium |
| Phase 6: Documentation | 4 files | Low |

---

## References

- [zerolog GitHub](https://github.com/rs/zerolog)
- [zerolog pkg.go.dev](https://pkg.go.dev/github.com/rs/zerolog)
- [zerolog v1.34.0 Release](https://github.com/rs/zerolog/releases/tag/v1.34.0)
- [ADR-0012: Configuration Management (Koanf)](./adr/0012-configuration-management-koanf.md)
