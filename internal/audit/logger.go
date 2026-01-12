// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package audit

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"sync"
	"time"

	"github.com/goccy/go-json"
	"github.com/tomtom215/cartographus/internal/logging"
)

// Config holds configuration for the audit logger.
type Config struct {
	// Enabled controls whether audit logging is active.
	Enabled bool `json:"enabled"`

	// LogLevel filters events by minimum severity.
	LogLevel Severity `json:"log_level"`

	// RetentionDays is how long to keep audit logs.
	RetentionDays int `json:"retention_days"`

	// CleanupInterval is how often to run retention cleanup.
	CleanupInterval time.Duration `json:"cleanup_interval"`

	// BufferSize is the size of the async write buffer.
	BufferSize int `json:"buffer_size"`

	// LogToStdout also writes events to stdout.
	LogToStdout bool `json:"log_to_stdout"`

	// IncludeDebug includes debug-level events.
	IncludeDebug bool `json:"include_debug"`
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		Enabled:         true,
		LogLevel:        SeverityInfo,
		RetentionDays:   90,
		CleanupInterval: 24 * time.Hour,
		BufferSize:      1000,
		LogToStdout:     false,
		IncludeDebug:    false,
	}
}

// Logger is the main audit logging service.
type Logger struct {
	config    *Config
	store     Store
	eventChan chan *Event
	mu        sync.RWMutex
	stopChan  chan struct{}
	wg        sync.WaitGroup
}

// NewLogger creates a new audit logger.
func NewLogger(store Store, config *Config) *Logger {
	if config == nil {
		config = DefaultConfig()
	}

	l := &Logger{
		config:    config,
		store:     store,
		eventChan: make(chan *Event, config.BufferSize),
		stopChan:  make(chan struct{}),
	}

	// Start async writer
	l.wg.Add(1)
	go l.asyncWriter()

	return l
}

// asyncWriter processes events from the buffer.
func (l *Logger) asyncWriter() {
	defer l.wg.Done()

	for {
		select {
		case <-l.stopChan:
			// Drain remaining events
			for {
				select {
				case event := <-l.eventChan:
					l.writeEvent(event)
				default:
					return
				}
			}
		case event := <-l.eventChan:
			l.writeEvent(event)
		}
	}
}

// writeEvent persists an event to the store.
func (l *Logger) writeEvent(event *Event) {
	l.mu.RLock()
	config := l.config
	l.mu.RUnlock()

	if config.LogToStdout {
		l.logToStdout(event)
	}

	if l.store != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := l.store.Save(ctx, event); err != nil {
			logging.Error().Err(err).Msg("Failed to save audit event")
		}
	}
}

// logToStdout writes an event to stdout in JSON format.
func (l *Logger) logToStdout(event *Event) {
	data, err := json.Marshal(event)
	if err != nil {
		logging.Error().Err(err).Msg("Failed to marshal audit event")
		return
	}
	logging.Info().RawJSON("event", data).Msg("Audit event")
}

// Log records an audit event.
func (l *Logger) Log(event *Event) {
	l.mu.RLock()
	config := l.config
	l.mu.RUnlock()

	if !config.Enabled {
		return
	}

	// Filter by severity
	if !l.shouldLog(event.Severity, config) {
		return
	}

	// Generate ID if not set
	if event.ID == "" {
		event.ID = generateEventID()
	}

	// Set timestamp if not set
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// Send to async writer
	select {
	case l.eventChan <- event:
	default:
		logging.Warn().Str("event_id", event.ID).Msg("Audit event buffer full, dropping event")
	}
}

// shouldLog returns true if the event severity meets the minimum level.
func (l *Logger) shouldLog(severity Severity, config *Config) bool {
	if severity == SeverityDebug && !config.IncludeDebug {
		return false
	}

	severityOrder := map[Severity]int{
		SeverityDebug:    0,
		SeverityInfo:     1,
		SeverityWarning:  2,
		SeverityError:    3,
		SeverityCritical: 4,
	}

	return severityOrder[severity] >= severityOrder[config.LogLevel]
}

// Close shuts down the logger gracefully.
func (l *Logger) Close() error {
	close(l.stopChan)
	l.wg.Wait()
	return nil
}

// StartCleanupRoutine starts the retention cleanup routine.
func (l *Logger) StartCleanupRoutine(ctx context.Context) {
	l.mu.RLock()
	interval := l.config.CleanupInterval
	retention := l.config.RetentionDays
	l.mu.RUnlock()

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				cutoff := time.Now().AddDate(0, 0, -retention)
				count, err := l.store.Delete(ctx, cutoff)
				if err != nil {
					logging.Error().Err(err).Msg("Audit cleanup error")
				} else if count > 0 {
					logging.Info().Int64("count", count).Msg("Cleaned up old audit events")
				}
			}
		}
	}()
}

// Query retrieves events matching the filter.
func (l *Logger) Query(ctx context.Context, filter QueryFilter) ([]Event, error) {
	return l.store.Query(ctx, filter)
}

// Count returns the number of events matching the filter.
func (l *Logger) Count(ctx context.Context, filter QueryFilter) (int64, error) {
	return l.store.Count(ctx, filter)
}

// SetEnabled enables or disables audit logging.
func (l *Logger) SetEnabled(enabled bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.config.Enabled = enabled
}

// Enabled returns whether audit logging is enabled.
func (l *Logger) Enabled() bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.config.Enabled
}

// generateEventID generates a unique event ID.
func generateEventID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return time.Now().Format("20060102150405.000000000")
	}
	return hex.EncodeToString(b)
}

// Helper methods for common audit events

// LogAuthSuccess logs a successful authentication.
//
//nolint:gocritic // hugeParam: Actor passed by value for API simplicity
func (l *Logger) LogAuthSuccess(ctx context.Context, actor Actor, source Source, authMethod string) {
	l.Log(&Event{
		Type:        EventTypeAuthSuccess,
		Severity:    SeverityInfo,
		Outcome:     OutcomeSuccess,
		Actor:       actor,
		Source:      source,
		Action:      "authenticate",
		Description: "User authenticated successfully",
		Metadata:    mustJSON(map[string]string{"method": authMethod}),
		RequestID:   getRequestID(ctx),
	})
}

// LogAuthFailure logs a failed authentication attempt.
func (l *Logger) LogAuthFailure(ctx context.Context, actorID, actorName string, source Source, reason string) {
	l.Log(&Event{
		Type:     EventTypeAuthFailure,
		Severity: SeverityWarning,
		Outcome:  OutcomeFailure,
		Actor: Actor{
			ID:   actorID,
			Type: "user",
			Name: actorName,
		},
		Source:      source,
		Action:      "authenticate",
		Description: "Authentication failed: " + reason,
		Metadata:    mustJSON(map[string]string{"reason": reason}),
		RequestID:   getRequestID(ctx),
	})
}

// LogAuthLockout logs an account lockout.
func (l *Logger) LogAuthLockout(ctx context.Context, actorID, actorName string, source Source, duration time.Duration, attempts int) {
	l.Log(&Event{
		Type:     EventTypeAuthLockout,
		Severity: SeverityCritical,
		Outcome:  OutcomeSuccess,
		Actor: Actor{
			ID:   actorID,
			Type: "user",
			Name: actorName,
		},
		Source:      source,
		Action:      "lockout",
		Description: "Account locked due to too many failed attempts",
		Metadata: mustJSON(map[string]interface{}{
			"duration_seconds": duration.Seconds(),
			"failed_attempts":  attempts,
		}),
		RequestID: getRequestID(ctx),
	})
}

// LogLogout logs a logout event.
//
//nolint:gocritic // hugeParam: Actor passed by value for API simplicity
func (l *Logger) LogLogout(ctx context.Context, actor Actor, source Source, sessionID string) {
	l.Log(&Event{
		Type:     EventTypeLogout,
		Severity: SeverityInfo,
		Outcome:  OutcomeSuccess,
		Actor:    actor,
		Source:   source,
		Action:   "logout",
		Target: &Target{
			ID:   sessionID,
			Type: "session",
		},
		Description: "User logged out",
		RequestID:   getRequestID(ctx),
	})
}

// LogAuthzDenied logs an authorization denial.
//
//nolint:gocritic // hugeParam: Actor passed by value for API simplicity
func (l *Logger) LogAuthzDenied(ctx context.Context, actor Actor, source Source, resource, action string) {
	l.Log(&Event{
		Type:     EventTypeAuthzDenied,
		Severity: SeverityWarning,
		Outcome:  OutcomeFailure,
		Actor:    actor,
		Source:   source,
		Action:   "authorize",
		Target: &Target{
			ID:   resource,
			Type: "resource",
		},
		Description: "Authorization denied for " + action + " on " + resource,
		Metadata: mustJSON(map[string]string{
			"resource":         resource,
			"requested_action": action,
		}),
		RequestID: getRequestID(ctx),
	})
}

// LogDetectionAlert logs a security detection alert.
func (l *Logger) LogDetectionAlert(ctx context.Context, ruleType, alertTitle string, userID int, username string, severity Severity) {
	l.Log(&Event{
		Type:     EventTypeDetectionAlert,
		Severity: severity,
		Outcome:  OutcomeSuccess,
		Actor: Actor{
			ID:   "detection_engine",
			Type: "system",
			Name: "Detection Engine",
		},
		Target: &Target{
			ID:   username,
			Type: "user",
			Name: username,
		},
		Action:      "detect",
		Description: alertTitle,
		Metadata: mustJSON(map[string]interface{}{
			"rule_type": ruleType,
			"user_id":   userID,
		}),
	})
}

// LogConfigChange logs a configuration change.
//
//nolint:gocritic // hugeParam: Actor passed by value for API simplicity
func (l *Logger) LogConfigChange(ctx context.Context, actor Actor, source Source, configKey, oldValue, newValue string) {
	l.Log(&Event{
		Type:     EventTypeConfigChanged,
		Severity: SeverityWarning,
		Outcome:  OutcomeSuccess,
		Actor:    actor,
		Source:   source,
		Action:   "update",
		Target: &Target{
			ID:   configKey,
			Type: "config",
		},
		Description: "Configuration changed: " + configKey,
		Metadata: mustJSON(map[string]string{
			"key":       configKey,
			"old_value": oldValue,
			"new_value": newValue,
		}),
		RequestID: getRequestID(ctx),
	})
}

// LogDataExport logs a data export event.
//
//nolint:gocritic // hugeParam: Actor passed by value for API simplicity
func (l *Logger) LogDataExport(ctx context.Context, actor Actor, source Source, format string, recordCount int) {
	l.Log(&Event{
		Type:        EventTypeDataExport,
		Severity:    SeverityInfo,
		Outcome:     OutcomeSuccess,
		Actor:       actor,
		Source:      source,
		Action:      "export",
		Description: "Data exported",
		Metadata: mustJSON(map[string]interface{}{
			"format":       format,
			"record_count": recordCount,
		}),
		RequestID: getRequestID(ctx),
	})
}

// LogAdminAction logs an administrative action.
//
//nolint:gocritic // hugeParam: Actor passed by value for API simplicity
func (l *Logger) LogAdminAction(ctx context.Context, actor Actor, source Source, action, description string, metadata map[string]interface{}) {
	l.Log(&Event{
		Type:        EventTypeAdminAction,
		Severity:    SeverityWarning,
		Outcome:     OutcomeSuccess,
		Actor:       actor,
		Source:      source,
		Action:      action,
		Description: description,
		Metadata:    mustJSON(metadata),
		RequestID:   getRequestID(ctx),
	})
}

// mustJSON converts a value to JSON, returning empty object on error.
func mustJSON(v interface{}) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		return json.RawMessage("{}")
	}
	return data
}

// getRequestID extracts the request ID from context.
func getRequestID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if reqID, ok := ctx.Value(RequestIDKey).(string); ok {
		return reqID
	}
	return ""
}

// Context keys
type contextKey string

// RequestIDKey is the context key for request ID.
const RequestIDKey contextKey = "request_id"

// SourceFromRequest creates a Source from an HTTP request.
func SourceFromRequest(r *http.Request) Source {
	ip := r.RemoteAddr
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ip = xff
	} else if xri := r.Header.Get("X-Real-IP"); xri != "" {
		ip = xri
	}

	return Source{
		IPAddress: ip,
		UserAgent: r.UserAgent(),
		Hostname:  r.Host,
	}
}

// ActorFromUser creates an Actor from user information.
func ActorFromUser(id, name string, roles []string, authMethod, sessionID string) Actor {
	return Actor{
		ID:         id,
		Type:       "user",
		Name:       name,
		Roles:      roles,
		AuthMethod: authMethod,
		SessionID:  sessionID,
	}
}

// SystemActor returns an Actor representing the system.
func SystemActor() Actor {
	return Actor{
		ID:   "system",
		Type: "system",
		Name: "Cartographus",
	}
}
