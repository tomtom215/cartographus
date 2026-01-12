// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package authz provides authorization decision audit logging for security
// monitoring, compliance reporting, and forensic analysis.
//
// Key Features:
//   - Records all authorization decisions (allow/deny)
//   - Captures context: actor, resource, action, decision, reason
//   - Request metadata: IP address, user agent, request ID
//   - Performance tracking: decision latency
//   - Configurable log levels and sampling
//
// ADR-0015: Zero Trust Authentication & Authorization
//
// Usage:
//
//	logger := NewAuditLogger(config)
//	defer logger.Close()
//
//	// Record a decision
//	logger.LogDecision(&AuditEvent{
//	    ActorID:  subject.ID,
//	    Resource: "/api/v1/users",
//	    Action:   "read",
//	    Decision: true,
//	})
package authz

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/tomtom215/cartographus/internal/logging"
)

// AuditEvent represents an authorization decision for audit purposes.
// Each event captures the complete context of an authorization check.
type AuditEvent struct {
	// ID is a unique identifier for this audit event
	ID string `json:"id"`

	// Timestamp is when the authorization decision was made
	Timestamp time.Time `json:"timestamp"`

	// RequestID links this event to an HTTP request (if applicable)
	RequestID string `json:"request_id,omitempty"`

	// ActorID is the subject requesting access
	ActorID string `json:"actor_id"`

	// ActorUsername is the display name of the actor
	ActorUsername string `json:"actor_username,omitempty"`

	// ActorRole is the effective role used for the decision
	ActorRole string `json:"actor_role,omitempty"`

	// ActorRoles is the list of all roles the actor has
	ActorRoles []string `json:"actor_roles,omitempty"`

	// Resource is the object being accessed (e.g., "/api/v1/users")
	Resource string `json:"resource"`

	// Action is the operation being performed (e.g., "read", "write", "delete")
	Action string `json:"action"`

	// Decision is true if access was allowed, false if denied
	Decision bool `json:"decision"`

	// Reason provides context for the decision (especially useful for denials)
	Reason string `json:"reason,omitempty"`

	// Duration is how long the authorization check took
	Duration time.Duration `json:"duration_ns"`

	// CacheHit indicates if the decision came from cache
	CacheHit bool `json:"cache_hit"`

	// IPAddress is the client's IP address
	IPAddress string `json:"ip_address,omitempty"`

	// UserAgent is the client's user agent string
	UserAgent string `json:"user_agent,omitempty"`

	// SessionID is the session identifier (if applicable)
	SessionID string `json:"session_id,omitempty"`

	// Method is the HTTP method (if applicable)
	Method string `json:"method,omitempty"`
}

// AuditLoggerConfig configures the audit logger behavior.
type AuditLoggerConfig struct {
	// Enabled controls whether audit logging is active
	Enabled bool

	// LogAllowed controls whether to log allowed decisions
	// Set to false to only log denials (reduces log volume)
	LogAllowed bool

	// LogDenied controls whether to log denied decisions
	LogDenied bool

	// SampleRate is the fraction of allowed decisions to log (0.0 to 1.0)
	// Only applies when LogAllowed is true. 1.0 means log all.
	// Denials are always logged at full rate when LogDenied is true.
	SampleRate float64

	// BufferSize is the size of the async log buffer
	// Events are dropped if buffer is full (non-blocking)
	BufferSize int

	// FlushInterval is how often to flush buffered events
	FlushInterval time.Duration
}

// DefaultAuditLoggerConfig returns sensible defaults for production.
func DefaultAuditLoggerConfig() *AuditLoggerConfig {
	return &AuditLoggerConfig{
		Enabled:       true,
		LogAllowed:    true,
		LogDenied:     true,
		SampleRate:    1.0, // Log all events by default
		BufferSize:    1000,
		FlushInterval: 5 * time.Second,
	}
}

// AuditLogger handles async logging of authorization decisions.
type AuditLogger struct {
	config   *AuditLoggerConfig
	events   chan *AuditEvent
	stopChan chan struct{}
	stopOnce sync.Once
	wg       sync.WaitGroup
}

// NewAuditLogger creates a new audit logger with the given configuration.
func NewAuditLogger(config *AuditLoggerConfig) *AuditLogger {
	if config == nil {
		config = DefaultAuditLoggerConfig()
	}

	if config.BufferSize <= 0 {
		config.BufferSize = 1000
	}

	if config.FlushInterval <= 0 {
		config.FlushInterval = 5 * time.Second
	}

	if config.SampleRate <= 0 {
		config.SampleRate = 1.0
	}
	if config.SampleRate > 1.0 {
		config.SampleRate = 1.0
	}

	al := &AuditLogger{
		config:   config,
		events:   make(chan *AuditEvent, config.BufferSize),
		stopChan: make(chan struct{}),
	}

	if config.Enabled {
		al.wg.Add(1)
		go al.processEvents()
	}

	return al
}

// LogDecision records an authorization decision asynchronously.
// This method is non-blocking; events are dropped if the buffer is full.
func (al *AuditLogger) LogDecision(event *AuditEvent) {
	if al == nil || !al.config.Enabled {
		return
	}

	// Check if we should log this event
	if event.Decision {
		if !al.config.LogAllowed {
			return
		}
		// Apply sampling for allowed decisions
		if al.config.SampleRate < 1.0 {
			// Simple deterministic sampling based on event ID
			if len(event.ID) > 0 && (int(event.ID[0])%100) >= int(al.config.SampleRate*100) {
				return
			}
		}
	} else if !al.config.LogDenied {
		return
	}

	// Generate ID if not set
	if event.ID == "" {
		event.ID = uuid.New().String()
	}

	// Set timestamp if not set
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// Non-blocking send
	select {
	case al.events <- event:
		// Event queued
	default:
		// Buffer full, log warning and drop
		logging.Warn().
			Str("actor_id", event.ActorID).
			Str("resource", event.Resource).
			Msg("Audit log buffer full, event dropped")
	}
}

// LogDecisionContext is a convenience method that creates an event from context.
func (al *AuditLogger) LogDecisionContext(
	ctx context.Context,
	actorID, actorUsername string,
	actorRoles []string,
	resource, action string,
	decision bool,
	reason string,
	duration time.Duration,
	cacheHit bool,
) {
	event := &AuditEvent{
		ID:            uuid.New().String(),
		Timestamp:     time.Now(),
		ActorID:       actorID,
		ActorUsername: actorUsername,
		ActorRoles:    actorRoles,
		Resource:      resource,
		Action:        action,
		Decision:      decision,
		Reason:        reason,
		Duration:      duration,
		CacheHit:      cacheHit,
	}

	// Extract request ID from context if available
	if reqID, ok := ctx.Value(requestIDKey).(string); ok {
		event.RequestID = reqID
	}

	al.LogDecision(event)
}

// processEvents handles the async event processing.
func (al *AuditLogger) processEvents() {
	defer al.wg.Done()

	for {
		select {
		case <-al.stopChan:
			// Drain remaining events
			al.drainEvents()
			return
		case event := <-al.events:
			al.writeEvent(event)
		}
	}
}

// drainEvents processes any remaining events in the buffer.
func (al *AuditLogger) drainEvents() {
	for {
		select {
		case event := <-al.events:
			al.writeEvent(event)
		default:
			return
		}
	}
}

// writeEvent outputs the event to the log.
func (al *AuditLogger) writeEvent(event *AuditEvent) {
	logEvent := logging.Info()

	// Always include these fields
	logEvent = logEvent.
		Str("event_type", "authz_decision").
		Str("audit_id", event.ID).
		Time("audit_timestamp", event.Timestamp).
		Str("actor_id", event.ActorID).
		Str("resource", event.Resource).
		Str("action", event.Action).
		Bool("decision", event.Decision).
		Dur("duration", event.Duration).
		Bool("cache_hit", event.CacheHit)

	// Optional fields
	if event.ActorUsername != "" {
		logEvent = logEvent.Str("actor_username", event.ActorUsername)
	}
	if event.ActorRole != "" {
		logEvent = logEvent.Str("actor_role", event.ActorRole)
	}
	if len(event.ActorRoles) > 0 {
		logEvent = logEvent.Strs("actor_roles", event.ActorRoles)
	}
	if event.RequestID != "" {
		logEvent = logEvent.Str("request_id", event.RequestID)
	}
	if event.Reason != "" {
		logEvent = logEvent.Str("reason", event.Reason)
	}
	if event.IPAddress != "" {
		logEvent = logEvent.Str("ip_address", event.IPAddress)
	}
	if event.UserAgent != "" {
		logEvent = logEvent.Str("user_agent", event.UserAgent)
	}
	if event.SessionID != "" {
		logEvent = logEvent.Str("session_id", event.SessionID)
	}
	if event.Method != "" {
		logEvent = logEvent.Str("method", event.Method)
	}

	// Use appropriate level based on decision
	if event.Decision {
		logEvent.Msg("Authorization allowed")
	} else {
		// Log denials as warnings for visibility
		logging.Warn().
			Str("event_type", "authz_decision").
			Str("audit_id", event.ID).
			Time("audit_timestamp", event.Timestamp).
			Str("actor_id", event.ActorID).
			Str("actor_username", event.ActorUsername).
			Str("resource", event.Resource).
			Str("action", event.Action).
			Bool("decision", event.Decision).
			Str("reason", event.Reason).
			Dur("duration", event.Duration).
			Str("ip_address", event.IPAddress).
			Msg("Authorization denied")
	}
}

// Close stops the audit logger and flushes remaining events.
func (al *AuditLogger) Close() {
	if al == nil {
		return
	}

	al.stopOnce.Do(func() {
		close(al.stopChan)
	})
	al.wg.Wait()
}

// Stats returns current audit logger statistics.
func (al *AuditLogger) Stats() AuditLoggerStats {
	if al == nil {
		return AuditLoggerStats{}
	}

	return AuditLoggerStats{
		BufferSize:    al.config.BufferSize,
		BufferUsed:    len(al.events),
		Enabled:       al.config.Enabled,
		LogAllowed:    al.config.LogAllowed,
		LogDenied:     al.config.LogDenied,
		SampleRate:    al.config.SampleRate,
		FlushInterval: al.config.FlushInterval,
	}
}

// AuditLoggerStats provides statistics about the audit logger.
type AuditLoggerStats struct {
	BufferSize    int           `json:"buffer_size"`
	BufferUsed    int           `json:"buffer_used"`
	Enabled       bool          `json:"enabled"`
	LogAllowed    bool          `json:"log_allowed"`
	LogDenied     bool          `json:"log_denied"`
	SampleRate    float64       `json:"sample_rate"`
	FlushInterval time.Duration `json:"flush_interval"`
}

// Context key for request ID
type contextKey string

const requestIDKey contextKey = "request_id"

// WithRequestID adds a request ID to the context for audit correlation.
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

// GetRequestID retrieves the request ID from context.
func GetRequestID(ctx context.Context) string {
	if reqID, ok := ctx.Value(requestIDKey).(string); ok {
		return reqID
	}
	return ""
}
