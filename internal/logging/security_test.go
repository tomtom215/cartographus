// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package logging

import (
	"bytes"
	"strings"
	"testing"

	"github.com/rs/zerolog"
)

func TestSanitizeToken(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"short", "***"},
		{"exactlytwelv", "***"},
		{"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9", "eyJh...VCJ9"},
		{"1234567890123456", "1234...3456"},
	}

	for _, tt := range tests {
		result := SanitizeToken(tt.input)
		if result != tt.expected {
			t.Errorf("SanitizeToken(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestSanitizeSessionID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"short123", "***"},
		{"exactlytwelv", "***"},
		{"abc123def456789", "abc1...6789"},
		{"session-id-12345678", "sess...5678"},
	}

	for _, tt := range tests {
		result := SanitizeSessionID(tt.input)
		if result != tt.expected {
			t.Errorf("SanitizeSessionID(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestSanitizeUserID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"short", "***"},
		{"12345678", "***"},
		{"user-12345678", "user...5678"},
		{"a-very-long-user-id", "a-ve...r-id"},
	}

	for _, tt := range tests {
		result := SanitizeUserID(tt.input)
		if result != tt.expected {
			t.Errorf("SanitizeUserID(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestSanitizeUsername(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"a", "***"},
		{"ab", "***"},
		{"johndoe", "jo***"},
		{"administrator", "ad***"},
	}

	for _, tt := range tests {
		result := SanitizeUsername(tt.input)
		if result != tt.expected {
			t.Errorf("SanitizeUsername(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestSanitizeEmail(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"invalid", "***"},
		{"a@b.com", "***@b.com"},
		{"ab@example.com", "***@example.com"},
		{"john.doe@example.com", "jo***@example.com"},
	}

	for _, tt := range tests {
		result := SanitizeEmail(tt.input)
		if result != tt.expected {
			t.Errorf("SanitizeEmail(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestSanitizeError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"regular error", "regular error"},
		{"invalid password", "authentication error"},
		{"token expired", "authentication error"},
		{"secret key invalid", "authentication error"},
		{"Bearer token missing", "authentication error"},
		{"authorization failed", "authentication error"},
		{"cookie missing", "authentication error"},
	}

	for _, tt := range tests {
		result := SanitizeError(tt.input)
		if result != tt.expected {
			t.Errorf("SanitizeError(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestSanitizeError_LongError(t *testing.T) {
	t.Parallel()

	longErr := strings.Repeat("a", 250)
	result := SanitizeError(longErr)

	if len(result) > 210 { // 200 + "..."
		t.Errorf("expected truncated error, got length %d", len(result))
	}
	if !strings.HasSuffix(result, "...") {
		t.Error("expected truncation suffix")
	}
}

func TestSanitizeValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		key      string
		value    string
		expected string
	}{
		{"name", "John", "John"},
		{"token", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9", "eyJh...VCJ9"},
		{"password", "secret123", "***"},                         // <= 12 chars, fully masked
		{"access_token", "token-value-12345", "toke...2345"},     // > 12 chars, partial mask
		{"email_field", "john@example.com", "jo***@example.com"}, // email sanitization
		{"api_key", "key-12345678901234", "key-...1234"},         // > 12 chars, partial mask
	}

	for _, tt := range tests {
		result := SanitizeValue(tt.key, tt.value)
		if result != tt.expected {
			t.Errorf("SanitizeValue(%q, %q) = %q, want %q", tt.key, tt.value, result, tt.expected)
		}
	}
}

func TestSecurityLogger_LogEvent(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	secLog := NewSecurityLoggerWithLogger(logger)

	secLog.LogEvent(&SecurityEvent{
		Event:     "test_event",
		UserID:    "user-12345678",
		Username:  "testuser",
		SessionID: "session-id-123456",
		Provider:  "jwt",
		IPAddress: "192.168.1.1",
		UserAgent: "TestBrowser/1.0",
		Success:   true,
	})

	output := buf.String()
	if !strings.Contains(output, "test_event") {
		t.Errorf("expected event in output: %s", output)
	}
	if !strings.Contains(output, "success") {
		t.Errorf("expected status in output: %s", output)
	}
	if !strings.Contains(output, "user...5678") {
		t.Errorf("expected sanitized user_id in output: %s", output)
	}
	if !strings.Contains(output, "te***") {
		t.Errorf("expected sanitized username in output: %s", output)
	}
}

func TestSecurityLogger_LogEvent_Failed(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	secLog := NewSecurityLoggerWithLogger(logger)

	secLog.LogEvent(&SecurityEvent{
		Event:   "login_failed",
		Success: false,
		Error:   "invalid credentials",
	})

	output := buf.String()
	if !strings.Contains(output, "failed") {
		t.Errorf("expected failed status in output: %s", output)
	}
}

func TestSecurityLogger_LogLoginSuccess(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	secLog := NewSecurityLoggerWithLogger(logger)

	secLog.LogLoginSuccess("user-123456789", "johndoe", "jwt", "192.168.1.1", "Mozilla/5.0")

	output := buf.String()
	if !strings.Contains(output, "login_success") {
		t.Errorf("expected login_success event: %s", output)
	}
	if !strings.Contains(output, "success") {
		t.Errorf("expected success status: %s", output)
	}
}

func TestSecurityLogger_LogLoginFailure(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	secLog := NewSecurityLoggerWithLogger(logger)

	secLog.LogLoginFailure("johndoe", "basic", "192.168.1.1", "Mozilla/5.0", "invalid password")

	output := buf.String()
	if !strings.Contains(output, "login_failed") {
		t.Errorf("expected login_failed event: %s", output)
	}
	if !strings.Contains(output, "failed") {
		t.Errorf("expected failed status: %s", output)
	}
}

func TestSecurityLogger_LogLogout(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	secLog := NewSecurityLoggerWithLogger(logger)

	secLog.LogLogout("user-123456789", "session-abc123def456", "192.168.1.1")

	output := buf.String()
	if !strings.Contains(output, "logout") {
		t.Errorf("expected logout event: %s", output)
	}
}

func TestSecurityLogger_LogLevels(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	secLog := NewSecurityLoggerWithLogger(logger)

	tests := []struct {
		name    string
		logFunc func()
		level   string
	}{
		{"Debug", func() { secLog.Debug("debug msg") }, "debug"},
		{"Info", func() { secLog.Info("info msg") }, "info"},
		{"Warn", func() { secLog.Warn("warn msg") }, "warn"},
		{"Error", func() { secLog.Error("error msg") }, "error"},
	}

	for _, tt := range tests {
		buf.Reset()
		tt.logFunc()
		output := buf.String()
		if !strings.Contains(output, tt.level) {
			t.Errorf("%s: expected level '%s' in output: %s", tt.name, tt.level, output)
		}
	}
}

func TestSecurityLogger_WithFields(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	secLog := NewSecurityLoggerWithLogger(logger)

	secLog.Info("test", "key1", "value1", "key2", 42)

	output := buf.String()
	if !strings.Contains(output, "key1") {
		t.Errorf("expected key1 in output: %s", output)
	}
	if !strings.Contains(output, "value1") {
		t.Errorf("expected value1 in output: %s", output)
	}
}

func TestNewSecurityLogger(t *testing.T) {
	// Should not panic
	secLog := NewSecurityLogger()
	if secLog == nil {
		t.Error("expected non-nil security logger")
	}
}

func TestTruncateString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"exactly10!", 10, "exactly10!"},
		{"this is a longer string", 10, "this is a ..."},
	}

	for _, tt := range tests {
		result := truncateString(tt.input, tt.maxLen)
		if result != tt.expected {
			t.Errorf("truncateString(%q, %d) = %q, want %q", tt.input, tt.maxLen, result, tt.expected)
		}
	}
}
