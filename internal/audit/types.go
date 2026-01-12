// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package audit provides security audit logging functionality.
// It records security-relevant events for compliance and forensic analysis.
package audit

import (
	"context"
	"time"

	"github.com/goccy/go-json"
)

// EventType categorizes audit events.
type EventType string

const (
	// Authentication events
	EventTypeAuthSuccess    EventType = "auth.success"
	EventTypeAuthFailure    EventType = "auth.failure"
	EventTypeAuthLockout    EventType = "auth.lockout"
	EventTypeAuthUnlock     EventType = "auth.unlock"
	EventTypeLogout         EventType = "auth.logout"
	EventTypeLogoutAll      EventType = "auth.logout_all"
	EventTypeSessionCreated EventType = "auth.session_created"
	EventTypeSessionExpired EventType = "auth.session_expired"
	EventTypeTokenRevoked   EventType = "auth.token_revoked"

	// Authorization events
	EventTypeAuthzGranted EventType = "authz.granted"
	EventTypeAuthzDenied  EventType = "authz.denied"

	// Detection events
	EventTypeDetectionAlert        EventType = "detection.alert"
	EventTypeDetectionAcknowledged EventType = "detection.acknowledged"
	EventTypeDetectionRuleChanged  EventType = "detection.rule_changed"

	// User management events
	EventTypeUserCreated  EventType = "user.created"
	EventTypeUserModified EventType = "user.modified"
	EventTypeUserDeleted  EventType = "user.deleted"
	EventTypeRoleAssigned EventType = "user.role_assigned"
	EventTypeRoleRevoked  EventType = "user.role_revoked"

	// Configuration events
	EventTypeConfigChanged EventType = "config.changed"

	// Data access events
	EventTypeDataExport EventType = "data.export"
	EventTypeDataImport EventType = "data.import"
	EventTypeDataBackup EventType = "data.backup"

	// Administrative events
	EventTypeAdminAction EventType = "admin.action"
)

// Severity indicates the severity level of an audit event.
type Severity string

const (
	SeverityDebug    Severity = "debug"
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityError    Severity = "error"
	SeverityCritical Severity = "critical"
)

// Outcome indicates whether an action succeeded or failed.
type Outcome string

const (
	OutcomeSuccess Outcome = "success"
	OutcomeFailure Outcome = "failure"
	OutcomeUnknown Outcome = "unknown"
)

// Event represents a security audit event.
type Event struct {
	// ID is a unique identifier for this event.
	ID string `json:"id"`

	// Timestamp when the event occurred.
	Timestamp time.Time `json:"timestamp"`

	// Type categorizes the event.
	Type EventType `json:"type"`

	// Severity of the event.
	Severity Severity `json:"severity"`

	// Outcome indicates success or failure.
	Outcome Outcome `json:"outcome"`

	// Actor who performed the action.
	Actor Actor `json:"actor"`

	// Target of the action (optional).
	Target *Target `json:"target,omitempty"`

	// Source of the request.
	Source Source `json:"source"`

	// Action describes what was done.
	Action string `json:"action"`

	// Description provides human-readable details.
	Description string `json:"description"`

	// Metadata contains event-specific details.
	Metadata json.RawMessage `json:"metadata,omitempty"`

	// CorrelationID links related events.
	CorrelationID string `json:"correlation_id,omitempty"`

	// RequestID from the originating HTTP request.
	RequestID string `json:"request_id,omitempty"`
}

// Actor represents who performed an action.
type Actor struct {
	// ID is the unique identifier (user ID, service account, etc.).
	ID string `json:"id"`

	// Type of actor (user, service, system).
	Type string `json:"type"`

	// Username or service name.
	Name string `json:"name,omitempty"`

	// Roles assigned to the actor.
	Roles []string `json:"roles,omitempty"`

	// SessionID if authenticated via session.
	SessionID string `json:"session_id,omitempty"`

	// AuthMethod used (jwt, basic, oidc, etc.).
	AuthMethod string `json:"auth_method,omitempty"`
}

// Target represents the object of an action.
type Target struct {
	// ID of the target resource.
	ID string `json:"id"`

	// Type of target (user, config, session, etc.).
	Type string `json:"type"`

	// Name of the target.
	Name string `json:"name,omitempty"`
}

// Source represents where a request originated.
type Source struct {
	// IPAddress of the client.
	IPAddress string `json:"ip_address"`

	// UserAgent of the client.
	UserAgent string `json:"user_agent,omitempty"`

	// Hostname if available.
	Hostname string `json:"hostname,omitempty"`

	// Port of the client.
	Port int `json:"port,omitempty"`

	// Geo contains geolocation info if available.
	Geo *GeoLocation `json:"geo,omitempty"`
}

// GeoLocation contains geographic information.
type GeoLocation struct {
	City      string  `json:"city,omitempty"`
	Region    string  `json:"region,omitempty"`
	Country   string  `json:"country,omitempty"`
	Latitude  float64 `json:"latitude,omitempty"`
	Longitude float64 `json:"longitude,omitempty"`
}

// Store defines the interface for audit event persistence.
type Store interface {
	// Save persists an audit event.
	Save(ctx context.Context, event *Event) error

	// Get retrieves an event by ID.
	Get(ctx context.Context, id string) (*Event, error)

	// Query retrieves events matching the filter.
	Query(ctx context.Context, filter QueryFilter) ([]Event, error)

	// Count returns the number of events matching the filter.
	Count(ctx context.Context, filter QueryFilter) (int64, error)

	// Delete removes events older than the retention period.
	Delete(ctx context.Context, olderThan time.Time) (int64, error)
}

// QueryFilter defines filtering options for audit queries.
type QueryFilter struct {
	// Types filters by event types.
	Types []EventType `json:"types,omitempty"`

	// Severities filters by severity levels.
	Severities []Severity `json:"severities,omitempty"`

	// Outcomes filters by outcome.
	Outcomes []Outcome `json:"outcomes,omitempty"`

	// ActorID filters by actor ID.
	ActorID string `json:"actor_id,omitempty"`

	// ActorType filters by actor type.
	ActorType string `json:"actor_type,omitempty"`

	// TargetID filters by target ID.
	TargetID string `json:"target_id,omitempty"`

	// TargetType filters by target type.
	TargetType string `json:"target_type,omitempty"`

	// SourceIP filters by source IP.
	SourceIP string `json:"source_ip,omitempty"`

	// StartTime is the beginning of the time range.
	StartTime *time.Time `json:"start_time,omitempty"`

	// EndTime is the end of the time range.
	EndTime *time.Time `json:"end_time,omitempty"`

	// CorrelationID filters by correlation ID.
	CorrelationID string `json:"correlation_id,omitempty"`

	// RequestID filters by request ID.
	RequestID string `json:"request_id,omitempty"`

	// SearchText performs a text search on description and action.
	SearchText string `json:"search_text,omitempty"`

	// Limit is the maximum number of results.
	Limit int `json:"limit,omitempty"`

	// Offset for pagination.
	Offset int `json:"offset,omitempty"`

	// OrderBy specifies the sort field.
	OrderBy string `json:"order_by,omitempty"`

	// OrderDesc sorts in descending order.
	OrderDesc bool `json:"order_desc,omitempty"`
}

// DefaultQueryFilter returns a sensible default filter.
func DefaultQueryFilter() QueryFilter {
	return QueryFilter{
		Limit:     100,
		OrderBy:   "timestamp",
		OrderDesc: true,
	}
}
