// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package models provides data models for the application.
package models

import "time"

// MediaServerStatus represents the status of a configured media server.
// This is a read-only view of servers configured via environment variables.
type MediaServerStatus struct {
	ID              string     `json:"id"`
	Platform        string     `json:"platform"` // plex, jellyfin, emby, tautulli
	Name            string     `json:"name"`
	URL             string     `json:"url"` // Masked for display
	Enabled         bool       `json:"enabled"`
	Source          string     `json:"source"` // Always "env" for now
	Status          string     `json:"status"` // connected, disconnected, syncing, error, disabled
	RealtimeEnabled bool       `json:"realtime_enabled"`
	WebhooksEnabled bool       `json:"webhooks_enabled"`
	SessionPolling  bool       `json:"session_polling_enabled"`
	LastSyncAt      *time.Time `json:"last_sync_at,omitempty"`
	LastSyncStatus  string     `json:"last_sync_status,omitempty"`
	LastError       string     `json:"last_error,omitempty"`
	LastErrorAt     *time.Time `json:"last_error_at,omitempty"`
	ServerVersion   string     `json:"server_version,omitempty"`
	Immutable       bool       `json:"immutable"` // true for env-var servers (cannot be edited in UI)
}

// MediaServerListResponse represents the response for listing all configured servers.
type MediaServerListResponse struct {
	Servers     []MediaServerStatus `json:"servers"`
	TotalCount  int                 `json:"total_count"`
	Connected   int                 `json:"connected_count"`
	Syncing     int                 `json:"syncing_count"`
	Error       int                 `json:"error_count"`
	LastChecked time.Time           `json:"last_checked"`
}

// MediaServerTestRequest represents a request to test server connectivity.
type MediaServerTestRequest struct {
	Platform string `json:"platform" validate:"required,oneof=plex jellyfin emby tautulli"`
	URL      string `json:"url" validate:"required,url"`
	Token    string `json:"token" validate:"required"`
}

// MediaServerTestResponse represents the response from a connectivity test.
type MediaServerTestResponse struct {
	Success    bool   `json:"success"`
	LatencyMs  int64  `json:"latency_ms"`
	ServerName string `json:"server_name,omitempty"`
	Version    string `json:"version,omitempty"`
	Error      string `json:"error,omitempty"`
	ErrorCode  string `json:"error_code,omitempty"`
}

// SyncTriggerRequest represents a request to trigger sync for a specific server.
type SyncTriggerRequest struct {
	ServerID string `json:"server_id" validate:"required"`
	FullSync bool   `json:"full_sync"` // Whether to do a full historical sync
}

// SyncTriggerResponse represents the response from a sync trigger.
type SyncTriggerResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	SyncID  string `json:"sync_id,omitempty"`
}

// MediaServer represents a media server configuration stored in the database.
// Credentials (URL, token) are stored encrypted using AES-256-GCM.
type MediaServer struct {
	ID                     string     `json:"id" db:"id"`
	Platform               string     `json:"platform" db:"platform"`
	Name                   string     `json:"name" db:"name"`
	URLEncrypted           string     `json:"-" db:"url_encrypted"`     // Never expose encrypted value
	TokenEncrypted         string     `json:"-" db:"token_encrypted"`   // Never expose encrypted value
	ServerID               string     `json:"server_id" db:"server_id"` // Unique identifier for deduplication
	Enabled                bool       `json:"enabled" db:"enabled"`
	Settings               string     `json:"settings" db:"settings"` // JSON blob
	RealtimeEnabled        bool       `json:"realtime_enabled" db:"realtime_enabled"`
	WebhooksEnabled        bool       `json:"webhooks_enabled" db:"webhooks_enabled"`
	SessionPollingEnabled  bool       `json:"session_polling_enabled" db:"session_polling_enabled"`
	SessionPollingInterval string     `json:"session_polling_interval" db:"session_polling_interval"`
	Source                 string     `json:"source" db:"source"` // "env", "ui", or "import"
	CreatedBy              string     `json:"created_by" db:"created_by"`
	CreatedAt              time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at" db:"updated_at"`
	LastSyncAt             *time.Time `json:"last_sync_at,omitempty" db:"last_sync_at"`
	LastSyncStatus         string     `json:"last_sync_status,omitempty" db:"last_sync_status"`
	LastError              string     `json:"last_error,omitempty" db:"last_error"`
	LastErrorAt            *time.Time `json:"last_error_at,omitempty" db:"last_error_at"`
}

// MediaServerAudit represents an audit log entry for server configuration changes.
type MediaServerAudit struct {
	ID        string    `json:"id" db:"id"`
	ServerID  string    `json:"server_id" db:"server_id"`
	Action    string    `json:"action" db:"action"` // create, update, delete, enable, disable, test, sync
	UserID    string    `json:"user_id" db:"user_id"`
	Username  string    `json:"username" db:"username"`
	Changes   string    `json:"changes" db:"changes"` // JSON with changed fields (credentials redacted)
	IPAddress string    `json:"ip_address" db:"ip_address"`
	UserAgent string    `json:"user_agent" db:"user_agent"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// CreateMediaServerRequest represents a request to add a new media server.
type CreateMediaServerRequest struct {
	Platform               string         `json:"platform" validate:"required,oneof=plex jellyfin emby tautulli"`
	Name                   string         `json:"name" validate:"required,min=1,max=100"`
	URL                    string         `json:"url" validate:"required,url"`
	Token                  string         `json:"token" validate:"required,min=8"`
	RealtimeEnabled        bool           `json:"realtime_enabled"`
	WebhooksEnabled        bool           `json:"webhooks_enabled"`
	SessionPollingEnabled  bool           `json:"session_polling_enabled"`
	SessionPollingInterval string         `json:"session_polling_interval" validate:"omitempty"`
	Settings               map[string]any `json:"settings"`
}

// UpdateMediaServerRequest represents a request to update an existing media server.
type UpdateMediaServerRequest struct {
	Name                   *string        `json:"name,omitempty" validate:"omitempty,min=1,max=100"`
	URL                    *string        `json:"url,omitempty" validate:"omitempty,url"`
	Token                  *string        `json:"token,omitempty" validate:"omitempty,min=8"`
	Enabled                *bool          `json:"enabled,omitempty"`
	RealtimeEnabled        *bool          `json:"realtime_enabled,omitempty"`
	WebhooksEnabled        *bool          `json:"webhooks_enabled,omitempty"`
	SessionPollingEnabled  *bool          `json:"session_polling_enabled,omitempty"`
	SessionPollingInterval *string        `json:"session_polling_interval,omitempty"`
	Settings               map[string]any `json:"settings,omitempty"`
}

// MediaServerResponse represents a server in API responses.
// Credentials are masked for security.
type MediaServerResponse struct {
	ID                     string     `json:"id"`
	Platform               string     `json:"platform"`
	Name                   string     `json:"name"`
	URL                    string     `json:"url"`          // Decrypted and shown
	TokenMasked            string     `json:"token_masked"` // "****...last4"
	ServerID               string     `json:"server_id"`
	Enabled                bool       `json:"enabled"`
	Source                 string     `json:"source"`
	RealtimeEnabled        bool       `json:"realtime_enabled"`
	WebhooksEnabled        bool       `json:"webhooks_enabled"`
	SessionPollingEnabled  bool       `json:"session_polling_enabled"`
	SessionPollingInterval string     `json:"session_polling_interval"`
	Status                 string     `json:"status"` // connected, syncing, error, disabled
	LastSyncAt             *time.Time `json:"last_sync_at,omitempty"`
	LastSyncStatus         string     `json:"last_sync_status,omitempty"`
	LastError              string     `json:"last_error,omitempty"`
	LastErrorAt            *time.Time `json:"last_error_at,omitempty"`
	CreatedAt              time.Time  `json:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at"`
	Immutable              bool       `json:"immutable"` // true for env-var servers
}

// MediaServerAuditResponse represents an audit log entry in API responses.
type MediaServerAuditResponse struct {
	ID        string    `json:"id"`
	ServerID  string    `json:"server_id"`
	Action    string    `json:"action"`
	UserID    string    `json:"user_id"`
	Username  string    `json:"username"`
	Changes   any       `json:"changes"` // Parsed JSON
	IPAddress string    `json:"ip_address"`
	UserAgent string    `json:"user_agent"`
	CreatedAt time.Time `json:"created_at"`
}

// ServerAuditAction constants for audit logging.
const (
	ServerAuditActionCreate  = "create"
	ServerAuditActionUpdate  = "update"
	ServerAuditActionDelete  = "delete"
	ServerAuditActionEnable  = "enable"
	ServerAuditActionDisable = "disable"
	ServerAuditActionTest    = "test"
	ServerAuditActionSync    = "sync"
)

// ServerPlatform constants for supported platforms.
const (
	ServerPlatformPlex     = "plex"
	ServerPlatformJellyfin = "jellyfin"
	ServerPlatformEmby     = "emby"
	ServerPlatformTautulli = "tautulli"
)

// ServerSource constants for server configuration sources.
const (
	ServerSourceEnv    = "env"
	ServerSourceUI     = "ui"
	ServerSourceImport = "import"
)

// IsValidPlatform checks if a platform is supported.
func IsValidPlatform(platform string) bool {
	switch platform {
	case ServerPlatformPlex, ServerPlatformJellyfin, ServerPlatformEmby, ServerPlatformTautulli:
		return true
	default:
		return false
	}
}

// IsValidAuditAction checks if an audit action is valid.
func IsValidAuditAction(action string) bool {
	switch action {
	case ServerAuditActionCreate, ServerAuditActionUpdate, ServerAuditActionDelete,
		ServerAuditActionEnable, ServerAuditActionDisable, ServerAuditActionTest, ServerAuditActionSync:
		return true
	default:
		return false
	}
}
