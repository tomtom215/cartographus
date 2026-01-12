// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package audit provides security audit logging for compliance and forensic analysis.
//
// This package implements a comprehensive security audit trail for the Cartographus
// application, recording security-relevant events such as authentication attempts,
// authorization decisions, detection alerts, and administrative actions.
//
// # Overview
//
// The audit system provides:
//   - Structured event logging with typed event categories
//   - DuckDB persistence for durable audit trail storage
//   - Asynchronous buffered writes for minimal latency impact
//   - Automatic retention policy enforcement with configurable cleanup
//   - SIEM integration via Common Event Format (CEF) export
//   - Flexible querying with multi-dimensional filters
//
// # Event Types
//
// Events are categorized into the following groups:
//
// Authentication Events:
//   - auth.success: Successful login attempts
//   - auth.failure: Failed login attempts
//   - auth.lockout: Account lockouts due to failed attempts
//   - auth.logout: User logout events
//   - auth.session_created: New session creation
//   - auth.session_expired: Session expiration
//   - auth.token_revoked: Token revocation events
//
// Authorization Events:
//   - authz.granted: Access granted decisions
//   - authz.denied: Access denied decisions
//
// Detection Events:
//   - detection.alert: Security anomaly alerts
//   - detection.acknowledged: Alert acknowledgment
//   - detection.rule_changed: Detection rule configuration changes
//
// Administrative Events:
//   - user.created, user.modified, user.deleted: User lifecycle
//   - config.changed: Configuration modifications
//   - data.export, data.import, data.backup: Data operations
//   - admin.action: General administrative actions
//
// # Architecture
//
// The audit system uses a producer-consumer pattern:
//
//	Logger.Log() -> Event Buffer (chan) -> Async Writer -> Store
//	                     |                      |
//	                 Non-blocking           Background goroutine
//
// Events are buffered in a channel to avoid blocking the caller. A background
// goroutine drains the buffer and persists events to the store.
//
// # Usage Example
//
// Basic audit logging:
//
//	// Initialize store and logger
//	store := audit.NewDuckDBStore(db.Conn())
//	logger := audit.NewLogger(store, audit.DefaultConfig())
//	defer logger.Close()
//
//	// Log authentication success
//	logger.LogAuthSuccess(ctx, audit.Actor{
//	    ID:   userID,
//	    Type: "user",
//	    Name: username,
//	}, audit.SourceFromRequest(r), "jwt")
//
//	// Log authentication failure
//	logger.LogAuthFailure(ctx, userID, username,
//	    audit.SourceFromRequest(r), "invalid_password")
//
//	// Log authorization denial
//	logger.LogAuthzDenied(ctx, actor, source, "/api/admin", "write")
//
// Querying audit logs:
//
//	filter := audit.QueryFilter{
//	    Types:      []audit.EventType{audit.EventTypeAuthFailure},
//	    StartTime:  &startTime,
//	    EndTime:    &endTime,
//	    ActorID:    "user123",
//	    Limit:      100,
//	    OrderDesc:  true,
//	}
//	events, err := logger.Query(ctx, filter)
//
// # Configuration
//
// The logger supports the following configuration options:
//
//	cfg := audit.Config{
//	    Enabled:         true,           // Enable audit logging
//	    LogLevel:        audit.SeverityInfo, // Minimum severity level
//	    RetentionDays:   90,             // Keep logs for 90 days
//	    CleanupInterval: 24 * time.Hour, // Run cleanup daily
//	    BufferSize:      1000,           // Event buffer size
//	    LogToStdout:     false,          // Also log to stdout
//	    IncludeDebug:    false,          // Include debug events
//	}
//
// # SIEM Integration
//
// Export events in Common Event Format (CEF) for SIEM integration:
//
//	exporter := audit.NewCEFExporter()
//	events, _ := logger.Query(ctx, filter)
//	cefData, _ := exporter.Export(events)
//
// # Retention Policy
//
// Automatic retention cleanup runs at the configured interval:
//
//	logger.StartCleanupRoutine(ctx)
//	// Events older than RetentionDays are automatically deleted
//
// # Thread Safety
//
// All exported functions are safe for concurrent use:
//   - Logger uses buffered channel for non-blocking writes
//   - Store implementations use appropriate synchronization
//   - Query operations use read locks for concurrent access
//
// # Performance Characteristics
//
//   - Log operation: <1ms (non-blocking, channel send)
//   - Query operation: 1-100ms depending on filter complexity
//   - Buffer overflow: Events dropped with warning log
//   - Memory overhead: ~100 bytes per buffered event
//
// # See Also
//
//   - internal/auth: Authentication events source
//   - internal/authz: Authorization events source
//   - internal/detection: Detection alerts source
//   - internal/api: Audit handlers for API access
package audit
