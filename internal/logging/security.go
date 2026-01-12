// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package logging

import (
	"strings"

	"github.com/rs/zerolog"
)

// SecurityEvent represents a security-relevant event for audit logging.
type SecurityEvent struct {
	// Event is the type of event (e.g., "login_success", "logout", "token_refresh").
	Event string
	// UserID is the user's identifier (if known).
	UserID string
	// Username is the user's username (if known).
	Username string
	// SessionID is the session identifier (sanitized).
	SessionID string
	// Provider is the authentication provider (oidc, plex, basic, jwt).
	Provider string
	// IPAddress is the client's IP address.
	IPAddress string
	// UserAgent is the client's user agent (truncated).
	UserAgent string
	// Success indicates if the operation was successful.
	Success bool
	// Error is the error message if the operation failed.
	Error string
	// Details contains additional sanitized details.
	Details map[string]string
}

// SecurityLogger provides secure logging for authentication events.
// It automatically sanitizes sensitive data before logging.
type SecurityLogger struct {
	logger zerolog.Logger
}

// NewSecurityLogger creates a new security logger.
func NewSecurityLogger() *SecurityLogger {
	return &SecurityLogger{
		logger: With().Str("component", "auth").Logger(),
	}
}

// NewSecurityLoggerWithLogger creates a security logger with a custom zerolog logger.
//
//nolint:gocritic // zerolog.Logger is designed to be passed by value
func NewSecurityLoggerWithLogger(logger zerolog.Logger) *SecurityLogger {
	return &SecurityLogger{
		logger: logger.With().Str("component", "auth").Logger(),
	}
}

// LogEvent logs a security event with automatic sanitization.
func (l *SecurityLogger) LogEvent(event *SecurityEvent) {
	e := l.logger.Info().
		Str("event", event.Event)

	if event.Success {
		e = e.Str("status", "success")
	} else {
		e = e.Str("status", "failed")
	}

	if event.UserID != "" {
		e = e.Str("user_id", SanitizeUserID(event.UserID))
	}

	if event.Username != "" {
		e = e.Str("username", SanitizeUsername(event.Username))
	}

	if event.SessionID != "" {
		e = e.Str("session_id", SanitizeSessionID(event.SessionID))
	}

	if event.Provider != "" {
		e = e.Str("provider", event.Provider)
	}

	if event.IPAddress != "" {
		e = e.Str("ip", event.IPAddress)
	}

	if event.UserAgent != "" {
		e = e.Str("user_agent", truncateString(event.UserAgent, 100))
	}

	if event.Error != "" && !event.Success {
		e = e.Str("error", SanitizeError(event.Error))
	}

	// Add sanitized details
	for k, v := range event.Details {
		e = e.Str(k, SanitizeValue(k, v))
	}

	e.Msg("")
}

// Debug logs a debug-level message.
func (l *SecurityLogger) Debug(msg string, fields ...interface{}) {
	e := l.logger.Debug()
	e = addFieldPairs(e, fields)
	e.Msg(msg)
}

// Info logs an info-level message.
func (l *SecurityLogger) Info(msg string, fields ...interface{}) {
	e := l.logger.Info()
	e = addFieldPairs(e, fields)
	e.Msg(msg)
}

// Warn logs a warning-level message.
func (l *SecurityLogger) Warn(msg string, fields ...interface{}) {
	e := l.logger.Warn()
	e = addFieldPairs(e, fields)
	e.Msg(msg)
}

// Error logs an error-level message.
func (l *SecurityLogger) Error(msg string, fields ...interface{}) {
	e := l.logger.Error()
	e = addFieldPairs(e, fields)
	e.Msg(msg)
}

// addFieldPairs adds key-value pairs to a zerolog event.
func addFieldPairs(e *zerolog.Event, fields []interface{}) *zerolog.Event {
	for i := 0; i < len(fields); i += 2 {
		if i+1 < len(fields) {
			key, ok := fields[i].(string)
			if !ok {
				continue
			}
			e = e.Interface(key, fields[i+1])
		}
	}
	return e
}

// ============================================================
// Pre-defined Security Events
// ============================================================

// LogLoginSuccess logs a successful login event.
func (l *SecurityLogger) LogLoginSuccess(userID, username, provider, ip, userAgent string) {
	l.LogEvent(&SecurityEvent{
		Event:     "login_success",
		UserID:    userID,
		Username:  username,
		Provider:  provider,
		IPAddress: ip,
		UserAgent: userAgent,
		Success:   true,
	})
}

// LogLoginFailure logs a failed login event.
func (l *SecurityLogger) LogLoginFailure(username, provider, ip, userAgent, reason string) {
	l.LogEvent(&SecurityEvent{
		Event:     "login_failed",
		Username:  username,
		Provider:  provider,
		IPAddress: ip,
		UserAgent: userAgent,
		Success:   false,
		Error:     reason,
	})
}

// LogLogout logs a logout event.
func (l *SecurityLogger) LogLogout(userID, sessionID, ip string) {
	l.LogEvent(&SecurityEvent{
		Event:     "logout",
		UserID:    userID,
		SessionID: sessionID,
		IPAddress: ip,
		Success:   true,
	})
}

// LogLogoutAll logs a logout-all event.
func (l *SecurityLogger) LogLogoutAll(userID, ip string, sessionCount int) {
	l.LogEvent(&SecurityEvent{
		Event:     "logout_all",
		UserID:    userID,
		IPAddress: ip,
		Success:   true,
		Details: map[string]string{
			"sessions_revoked": string(rune('0' + sessionCount)),
		},
	})
}

// LogTokenRefresh logs a token refresh event.
func (l *SecurityLogger) LogTokenRefresh(userID, sessionID, provider string, success bool, errMsg string) {
	l.LogEvent(&SecurityEvent{
		Event:     "token_refresh",
		UserID:    userID,
		SessionID: sessionID,
		Provider:  provider,
		Success:   success,
		Error:     errMsg,
	})
}

// LogSessionCreated logs a session creation event.
func (l *SecurityLogger) LogSessionCreated(userID, sessionID, provider, ip string) {
	l.LogEvent(&SecurityEvent{
		Event:     "session_created",
		UserID:    userID,
		SessionID: sessionID,
		Provider:  provider,
		IPAddress: ip,
		Success:   true,
	})
}

// LogSessionRevoked logs a session revocation event.
func (l *SecurityLogger) LogSessionRevoked(userID, sessionID, revokedBy, ip string) {
	l.LogEvent(&SecurityEvent{
		Event:     "session_revoked",
		UserID:    userID,
		SessionID: sessionID,
		IPAddress: ip,
		Success:   true,
		Details: map[string]string{
			"revoked_by": SanitizeUserID(revokedBy),
		},
	})
}

// LogCSRFFailure logs a CSRF validation failure.
func (l *SecurityLogger) LogCSRFFailure(ip, userAgent, path string) {
	l.LogEvent(&SecurityEvent{
		Event:     "csrf_failed",
		IPAddress: ip,
		UserAgent: userAgent,
		Success:   false,
		Details: map[string]string{
			"path": path,
		},
	})
}

// LogBackChannelLogout logs a back-channel logout event.
func (l *SecurityLogger) LogBackChannelLogout(issuer, subject, sessionID string, success bool, errMsg string) {
	l.LogEvent(&SecurityEvent{
		Event:     "backchannel_logout",
		UserID:    subject,
		SessionID: sessionID,
		Success:   success,
		Error:     errMsg,
		Details: map[string]string{
			"issuer": issuer,
		},
	})
}

// ============================================================
// Sanitization Functions
// ============================================================

// SanitizeToken masks a token, showing only first and last 4 characters.
// Example: "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..." -> "eyJh...kpXV"
func SanitizeToken(token string) string {
	if token == "" {
		return ""
	}
	if len(token) <= 12 {
		return "***"
	}
	return token[:4] + "..." + token[len(token)-4:]
}

// SanitizeSessionID masks a session ID.
// Example: "abc123def456" -> "abc1...f456"
func SanitizeSessionID(sessionID string) string {
	if sessionID == "" {
		return ""
	}
	if len(sessionID) <= 12 {
		return "***"
	}
	return sessionID[:4] + "..." + sessionID[len(sessionID)-4:]
}

// SanitizeUserID masks a user ID for privacy.
// Example: "user-12345678" -> "user...5678"
func SanitizeUserID(userID string) string {
	if userID == "" {
		return ""
	}
	if len(userID) <= 8 {
		return "***"
	}
	return userID[:4] + "..." + userID[len(userID)-4:]
}

// SanitizeUsername masks a username, keeping first 2 characters.
// Example: "johndoe" -> "jo***"
func SanitizeUsername(username string) string {
	if username == "" {
		return ""
	}
	if len(username) <= 2 {
		return "***"
	}
	return username[:2] + "***"
}

// SanitizeEmail masks an email address.
// Example: "john.doe@example.com" -> "jo***@example.com"
func SanitizeEmail(email string) string {
	if email == "" {
		return ""
	}

	atIndex := strings.Index(email, "@")
	if atIndex <= 0 {
		return "***"
	}

	localPart := email[:atIndex]
	domain := email[atIndex:]

	if len(localPart) <= 2 {
		return "***" + domain
	}
	return localPart[:2] + "***" + domain
}

// SanitizeError removes potentially sensitive information from error messages.
func SanitizeError(err string) string {
	// Remove potential secrets from error messages
	sensitivePatterns := []string{
		"password",
		"secret",
		"token",
		"key",
		"bearer",
		"authorization",
		"cookie",
	}

	lowerErr := strings.ToLower(err)
	for _, pattern := range sensitivePatterns {
		if strings.Contains(lowerErr, pattern) {
			// Generic error message
			return "authentication error"
		}
	}

	// Truncate long errors
	return truncateString(err, 200)
}

// SanitizeValue sanitizes a value based on its key name.
func SanitizeValue(key, value string) string {
	lowerKey := strings.ToLower(key)

	// Check for sensitive key names
	sensitiveKeys := map[string]bool{
		"access_token":  true,
		"refresh_token": true,
		"id_token":      true,
		"token":         true,
		"password":      true,
		"secret":        true,
		"api_key":       true,
		"apikey":        true,
		"authorization": true,
		"bearer":        true,
		"cookie":        true,
		"session":       true,
		"session_id":    true,
		"sessionid":     true,
	}

	if sensitiveKeys[lowerKey] {
		return SanitizeToken(value)
	}

	// Check for email-like values
	if strings.Contains(value, "@") && strings.Contains(value, ".") {
		return SanitizeEmail(value)
	}

	return value
}

// truncateString truncates a string to a maximum length.
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
