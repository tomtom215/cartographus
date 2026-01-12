// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package auth provides authentication functionality including OIDC support.
// ADR-0015: Zero Trust Authentication & Authorization
package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net"
	"net/http"
	"time"

	"github.com/goccy/go-json"

	"github.com/tomtom215/cartographus/internal/audit"
	"github.com/tomtom215/cartographus/internal/logging"
)

// OIDCAuditLogger provides structured audit logging for OIDC authentication events.
// It integrates with the audit subsystem to provide security-relevant event records
// for compliance and forensic analysis.
//
// ADR-0015: Zero Trust Authentication (Zitadel Amendment)
// All authentication events are logged with full context for security monitoring.
type OIDCAuditLogger struct {
	store    audit.Store
	provider string // IdP provider name for identification
}

// NewOIDCAuditLogger creates a new OIDC audit logger.
// The provider parameter identifies the IdP (e.g., "keycloak", "okta", "auth0").
func NewOIDCAuditLogger(store audit.Store, provider string) *OIDCAuditLogger {
	if provider == "" {
		provider = string(AuthModeOIDC)
	}
	return &OIDCAuditLogger{
		store:    store,
		provider: provider,
	}
}

// OIDCLoginMetadata contains additional details for login events.
type OIDCLoginMetadata struct {
	Provider    string `json:"provider"`
	AuthMethod  string `json:"auth_method"` // "authorization_code", "pkce"
	Scopes      string `json:"scopes,omitempty"`
	RedirectURI string `json:"redirect_uri,omitempty"`
	Nonce       bool   `json:"nonce_used"`
	Duration    string `json:"duration,omitempty"`
	ErrorCode   string `json:"error_code,omitempty"`
	ErrorDesc   string `json:"error_description,omitempty"`
}

// OIDCLogoutMetadata contains additional details for logout events.
type OIDCLogoutMetadata struct {
	Provider       string `json:"provider"`
	LogoutType     string `json:"logout_type"` // "rp_initiated", "back_channel"
	SessionID      string `json:"session_id,omitempty"`
	SessionsCount  int    `json:"sessions_terminated,omitempty"`
	HasIDTokenHint bool   `json:"has_id_token_hint"`
}

// OIDCRefreshMetadata contains additional details for token refresh events.
type OIDCRefreshMetadata struct {
	Provider  string `json:"provider"`
	Duration  string `json:"duration,omitempty"`
	NewExpiry string `json:"new_expiry,omitempty"`
}

// LogLoginSuccess logs a successful OIDC login.
func (l *OIDCAuditLogger) LogLoginSuccess(ctx context.Context, r *http.Request, subject *AuthSubject, duration time.Duration) {
	if l.store == nil {
		return
	}

	metadata := OIDCLoginMetadata{
		Provider:   l.provider,
		AuthMethod: "pkce", // Zitadel always uses PKCE
		Nonce:      true,
		Duration:   duration.String(),
	}

	//nolint:errcheck // OIDCLoginMetadata is a simple struct that always marshals successfully
	metadataJSON, _ := json.Marshal(metadata)

	event := &audit.Event{
		ID:        generateEventID(),
		Timestamp: time.Now().UTC(),
		Type:      audit.EventTypeAuthSuccess,
		Severity:  audit.SeverityInfo,
		Outcome:   audit.OutcomeSuccess,
		Actor: audit.Actor{
			ID:         subject.ID,
			Type:       "user",
			Name:       subject.Username,
			Roles:      subject.Roles,
			AuthMethod: "oidc",
		},
		Source:      extractSource(r),
		Action:      "oidc.login",
		Description: "OIDC login successful via " + l.provider,
		Metadata:    metadataJSON,
		RequestID:   getRequestID(r),
	}

	if err := l.store.Save(ctx, event); err != nil {
		logging.Error().Err(err).
			Str("event_type", string(event.Type)).
			Str("user_id", subject.ID).
			Msg("Failed to save audit event")
	}
}

// LogLoginFailure logs a failed OIDC login attempt.
func (l *OIDCAuditLogger) LogLoginFailure(ctx context.Context, r *http.Request, errorCode, errorDesc string) {
	if l.store == nil {
		return
	}

	metadata := OIDCLoginMetadata{
		Provider:   l.provider,
		AuthMethod: "pkce",
		ErrorCode:  errorCode,
		ErrorDesc:  errorDesc,
	}

	//nolint:errcheck // OIDCLoginMetadata is a simple struct that always marshals successfully
	metadataJSON, _ := json.Marshal(metadata)

	event := &audit.Event{
		ID:        generateEventID(),
		Timestamp: time.Now().UTC(),
		Type:      audit.EventTypeAuthFailure,
		Severity:  audit.SeverityWarning,
		Outcome:   audit.OutcomeFailure,
		Actor: audit.Actor{
			ID:         "anonymous",
			Type:       "user",
			AuthMethod: "oidc",
		},
		Source:      extractSource(r),
		Action:      "oidc.login",
		Description: "OIDC login failed: " + errorCode,
		Metadata:    metadataJSON,
		RequestID:   getRequestID(r),
	}

	if err := l.store.Save(ctx, event); err != nil {
		logging.Error().Err(err).
			Str("event_type", string(event.Type)).
			Msg("Failed to save audit event")
	}
}

// LogLogout logs an RP-initiated logout.
func (l *OIDCAuditLogger) LogLogout(ctx context.Context, r *http.Request, userID, username, sessionID string, hasIDTokenHint bool) {
	if l.store == nil {
		return
	}

	metadata := OIDCLogoutMetadata{
		Provider:       l.provider,
		LogoutType:     "rp_initiated",
		SessionID:      sessionID,
		HasIDTokenHint: hasIDTokenHint,
	}

	//nolint:errcheck // OIDCLogoutMetadata is a simple struct that always marshals successfully
	metadataJSON, _ := json.Marshal(metadata)

	event := &audit.Event{
		ID:        generateEventID(),
		Timestamp: time.Now().UTC(),
		Type:      audit.EventTypeLogout,
		Severity:  audit.SeverityInfo,
		Outcome:   audit.OutcomeSuccess,
		Actor: audit.Actor{
			ID:         userID,
			Type:       "user",
			Name:       username,
			SessionID:  sessionID,
			AuthMethod: "oidc",
		},
		Target: &audit.Target{
			ID:   sessionID,
			Type: "session",
		},
		Source:      extractSource(r),
		Action:      "oidc.logout",
		Description: "User initiated logout",
		Metadata:    metadataJSON,
		RequestID:   getRequestID(r),
	}

	if err := l.store.Save(ctx, event); err != nil {
		logging.Error().Err(err).
			Str("event_type", string(event.Type)).
			Str("user_id", userID).
			Msg("Failed to save audit event")
	}
}

// LogBackChannelLogout logs a back-channel logout from the IdP.
func (l *OIDCAuditLogger) LogBackChannelLogout(ctx context.Context, r *http.Request, subject, sessionID string, sessionsTerminated int) {
	if l.store == nil {
		return
	}

	metadata := OIDCLogoutMetadata{
		Provider:      l.provider,
		LogoutType:    "back_channel",
		SessionID:     sessionID,
		SessionsCount: sessionsTerminated,
	}

	//nolint:errcheck // OIDCLogoutMetadata is a simple struct that always marshals successfully
	metadataJSON, _ := json.Marshal(metadata)

	event := &audit.Event{
		ID:        generateEventID(),
		Timestamp: time.Now().UTC(),
		Type:      audit.EventTypeLogout,
		Severity:  audit.SeverityInfo,
		Outcome:   audit.OutcomeSuccess,
		Actor: audit.Actor{
			ID:         "idp:" + l.provider,
			Type:       "service",
			Name:       l.provider + " IdP",
			AuthMethod: "back_channel",
		},
		Target: &audit.Target{
			ID:   subject,
			Type: "user",
			Name: subject,
		},
		Source:      extractSource(r),
		Action:      "oidc.backchannel_logout",
		Description: "IdP initiated back-channel logout",
		Metadata:    metadataJSON,
		RequestID:   getRequestID(r),
	}

	if err := l.store.Save(ctx, event); err != nil {
		logging.Error().Err(err).
			Str("event_type", string(event.Type)).
			Str("subject", subject).
			Msg("Failed to save audit event")
	}
}

// LogBackChannelLogoutFailure logs a failed back-channel logout attempt.
func (l *OIDCAuditLogger) LogBackChannelLogoutFailure(ctx context.Context, r *http.Request, reason string) {
	if l.store == nil {
		return
	}

	metadata := map[string]string{
		"provider": l.provider,
		"reason":   reason,
	}

	//nolint:errcheck // Simple map[string]string always marshals successfully
	metadataJSON, _ := json.Marshal(metadata)

	event := &audit.Event{
		ID:        generateEventID(),
		Timestamp: time.Now().UTC(),
		Type:      audit.EventTypeAuthFailure,
		Severity:  audit.SeverityWarning,
		Outcome:   audit.OutcomeFailure,
		Actor: audit.Actor{
			ID:   "idp:" + l.provider,
			Type: "service",
			Name: l.provider + " IdP",
		},
		Source:      extractSource(r),
		Action:      "oidc.backchannel_logout",
		Description: "Back-channel logout failed: " + reason,
		Metadata:    metadataJSON,
		RequestID:   getRequestID(r),
	}

	if err := l.store.Save(ctx, event); err != nil {
		logging.Error().Err(err).
			Str("event_type", string(event.Type)).
			Msg("Failed to save audit event")
	}
}

// LogTokenRefresh logs a token refresh operation.
func (l *OIDCAuditLogger) LogTokenRefresh(ctx context.Context, r *http.Request, userID, username string, success bool, duration time.Duration, newExpiry time.Time) {
	if l.store == nil {
		return
	}

	outcome := audit.OutcomeSuccess
	severity := audit.SeverityInfo
	description := "Token refresh successful"
	if !success {
		outcome = audit.OutcomeFailure
		severity = audit.SeverityWarning
		description = "Token refresh failed"
	}

	metadata := OIDCRefreshMetadata{
		Provider: l.provider,
		Duration: duration.String(),
	}
	if success {
		metadata.NewExpiry = newExpiry.UTC().Format(time.RFC3339)
	}

	//nolint:errcheck // OIDCRefreshMetadata is a simple struct that always marshals successfully
	metadataJSON, _ := json.Marshal(metadata)

	event := &audit.Event{
		ID:        generateEventID(),
		Timestamp: time.Now().UTC(),
		Type:      audit.EventTypeSessionCreated, // Reusing as there's no specific refresh type
		Severity:  severity,
		Outcome:   outcome,
		Actor: audit.Actor{
			ID:         userID,
			Type:       "user",
			Name:       username,
			AuthMethod: "oidc",
		},
		Source:      extractSource(r),
		Action:      "oidc.token_refresh",
		Description: description,
		Metadata:    metadataJSON,
		RequestID:   getRequestID(r),
	}

	if err := l.store.Save(ctx, event); err != nil {
		logging.Error().Err(err).
			Str("event_type", string(event.Type)).
			Str("user_id", userID).
			Msg("Failed to save audit event")
	}
}

// LogSessionCreated logs a new session creation.
func (l *OIDCAuditLogger) LogSessionCreated(ctx context.Context, r *http.Request, session *Session) {
	if l.store == nil {
		return
	}

	metadata := map[string]interface{}{
		"provider":   l.provider,
		"session_id": session.ID,
		"expires_at": session.ExpiresAt.UTC().Format(time.RFC3339),
	}

	//nolint:errcheck // Simple map[string]interface{} with basic types always marshals successfully
	metadataJSON, _ := json.Marshal(metadata)

	event := &audit.Event{
		ID:        generateEventID(),
		Timestamp: time.Now().UTC(),
		Type:      audit.EventTypeSessionCreated,
		Severity:  audit.SeverityInfo,
		Outcome:   audit.OutcomeSuccess,
		Actor: audit.Actor{
			ID:         session.UserID,
			Type:       "user",
			Name:       session.Username,
			Roles:      session.Roles,
			SessionID:  session.ID,
			AuthMethod: "oidc",
		},
		Target: &audit.Target{
			ID:   session.ID,
			Type: "session",
		},
		Source:      extractSource(r),
		Action:      "session.create",
		Description: "OIDC session created",
		Metadata:    metadataJSON,
		RequestID:   getRequestID(r),
	}

	if err := l.store.Save(ctx, event); err != nil {
		logging.Error().Err(err).
			Str("event_type", string(event.Type)).
			Str("session_id", session.ID).
			Msg("Failed to save audit event")
	}
}

// extractSource extracts source information from an HTTP request.
func extractSource(r *http.Request) audit.Source {
	if r == nil {
		return audit.Source{IPAddress: "unknown"}
	}

	// Get IP address, preferring X-Forwarded-For for proxied requests
	ip := r.RemoteAddr
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		// Take the first IP in the chain (original client)
		if idx := len(forwarded); idx > 0 {
			ip = forwarded
			for i, c := range forwarded {
				if c == ',' {
					ip = forwarded[:i]
					break
				}
			}
		}
	}

	// Strip port from IP
	host, port, err := net.SplitHostPort(ip)
	if err == nil {
		ip = host
		_ = port // Could be used if needed
	}

	return audit.Source{
		IPAddress: ip,
		UserAgent: r.UserAgent(),
	}
}

// getRequestID extracts or generates a request ID.
func getRequestID(r *http.Request) string {
	if r == nil {
		return generateEventID()
	}

	// Check common request ID headers
	if id := r.Header.Get("X-Request-ID"); id != "" {
		return id
	}
	if id := r.Header.Get("X-Correlation-ID"); id != "" {
		return id
	}

	return ""
}

// generateEventID generates a unique event ID.
func generateEventID() string {
	bytes := make([]byte, 16)
	//nolint:errcheck // crypto/rand.Read error is extremely rare
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}
