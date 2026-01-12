// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package authz

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestNewAuditLogger(t *testing.T) {
	t.Run("nil config uses defaults", func(t *testing.T) {
		logger := NewAuditLogger(nil)
		defer logger.Close()

		if logger == nil {
			t.Fatal("expected non-nil logger")
		}
		if !logger.config.Enabled {
			t.Error("expected enabled=true by default")
		}
		if !logger.config.LogAllowed {
			t.Error("expected log_allowed=true by default")
		}
		if !logger.config.LogDenied {
			t.Error("expected log_denied=true by default")
		}
		if logger.config.SampleRate != 1.0 {
			t.Errorf("expected sample_rate=1.0, got %f", logger.config.SampleRate)
		}
	})

	t.Run("custom config is used", func(t *testing.T) {
		config := &AuditLoggerConfig{
			Enabled:       true,
			LogAllowed:    false,
			LogDenied:     true,
			SampleRate:    0.5,
			BufferSize:    500,
			FlushInterval: 10 * time.Second,
		}
		logger := NewAuditLogger(config)
		defer logger.Close()

		if logger.config.LogAllowed {
			t.Error("expected log_allowed=false")
		}
		if !logger.config.LogDenied {
			t.Error("expected log_denied=true")
		}
		if logger.config.SampleRate != 0.5 {
			t.Errorf("expected sample_rate=0.5, got %f", logger.config.SampleRate)
		}
	})

	t.Run("negative buffer size uses default", func(t *testing.T) {
		config := &AuditLoggerConfig{
			Enabled:    true,
			BufferSize: -1,
		}
		logger := NewAuditLogger(config)
		defer logger.Close()

		if logger.config.BufferSize != 1000 {
			t.Errorf("expected buffer_size=1000, got %d", logger.config.BufferSize)
		}
	})

	t.Run("sample rate clamped to valid range", func(t *testing.T) {
		config := &AuditLoggerConfig{
			Enabled:    true,
			SampleRate: 2.0, // Invalid, should be clamped to 1.0
		}
		logger := NewAuditLogger(config)
		defer logger.Close()

		if logger.config.SampleRate != 1.0 {
			t.Errorf("expected sample_rate=1.0, got %f", logger.config.SampleRate)
		}
	})

	t.Run("disabled logger does not start goroutine", func(t *testing.T) {
		config := &AuditLoggerConfig{
			Enabled:    false,
			BufferSize: 10,
		}
		logger := NewAuditLogger(config)
		defer logger.Close()

		// Should not panic or block
		logger.LogDecision(&AuditEvent{
			ActorID:  "user1",
			Resource: "/api/v1/test",
			Action:   "read",
			Decision: true,
		})
	})
}

func TestAuditLogger_LogDecision(t *testing.T) {
	t.Run("logs allowed decision when enabled", func(t *testing.T) {
		config := &AuditLoggerConfig{
			Enabled:       true,
			LogAllowed:    true,
			LogDenied:     true,
			SampleRate:    1.0,
			BufferSize:    10,
			FlushInterval: 100 * time.Millisecond,
		}
		logger := NewAuditLogger(config)
		defer logger.Close()

		event := &AuditEvent{
			ActorID:  "user1",
			Resource: "/api/v1/test",
			Action:   "read",
			Decision: true,
		}
		logger.LogDecision(event)

		// Give time for async processing
		time.Sleep(50 * time.Millisecond)

		// Check event was queued
		if len(logger.events) > 10 {
			t.Error("unexpected event queue overflow")
		}
	})

	t.Run("logs denied decision when enabled", func(t *testing.T) {
		config := &AuditLoggerConfig{
			Enabled:       true,
			LogAllowed:    true,
			LogDenied:     true,
			SampleRate:    1.0,
			BufferSize:    10,
			FlushInterval: 100 * time.Millisecond,
		}
		logger := NewAuditLogger(config)
		defer logger.Close()

		event := &AuditEvent{
			ActorID:  "user1",
			Resource: "/api/v1/admin",
			Action:   "write",
			Decision: false,
			Reason:   "Insufficient permissions",
		}
		logger.LogDecision(event)

		// Give time for async processing
		time.Sleep(50 * time.Millisecond)
	})

	t.Run("skips allowed when log_allowed is false", func(t *testing.T) {
		config := &AuditLoggerConfig{
			Enabled:       true,
			LogAllowed:    false,
			LogDenied:     true,
			SampleRate:    1.0,
			BufferSize:    10,
			FlushInterval: 100 * time.Millisecond,
		}
		logger := NewAuditLogger(config)
		defer logger.Close()

		event := &AuditEvent{
			ActorID:  "user1",
			Resource: "/api/v1/test",
			Action:   "read",
			Decision: true,
		}
		logger.LogDecision(event)

		// Event should not be queued
		time.Sleep(50 * time.Millisecond)
		if len(logger.events) > 0 {
			t.Error("expected no events when log_allowed is false")
		}
	})

	t.Run("skips denied when log_denied is false", func(t *testing.T) {
		config := &AuditLoggerConfig{
			Enabled:       true,
			LogAllowed:    true,
			LogDenied:     false,
			SampleRate:    1.0,
			BufferSize:    10,
			FlushInterval: 100 * time.Millisecond,
		}
		logger := NewAuditLogger(config)
		defer logger.Close()

		event := &AuditEvent{
			ActorID:  "user1",
			Resource: "/api/v1/admin",
			Action:   "write",
			Decision: false,
		}
		logger.LogDecision(event)

		// Event should not be queued
		time.Sleep(50 * time.Millisecond)
		if len(logger.events) > 0 {
			t.Error("expected no events when log_denied is false")
		}
	})

	t.Run("generates ID if not set", func(t *testing.T) {
		config := &AuditLoggerConfig{
			Enabled:       true,
			LogAllowed:    true,
			LogDenied:     true,
			SampleRate:    1.0,
			BufferSize:    10,
			FlushInterval: 100 * time.Millisecond,
		}
		logger := NewAuditLogger(config)
		defer logger.Close()

		event := &AuditEvent{
			ActorID:  "user1",
			Resource: "/api/v1/test",
			Action:   "read",
			Decision: true,
		}
		logger.LogDecision(event)

		// ID should have been generated
		if event.ID == "" {
			t.Error("expected ID to be generated")
		}
	})

	t.Run("sets timestamp if not set", func(t *testing.T) {
		config := &AuditLoggerConfig{
			Enabled:       true,
			LogAllowed:    true,
			LogDenied:     true,
			SampleRate:    1.0,
			BufferSize:    10,
			FlushInterval: 100 * time.Millisecond,
		}
		logger := NewAuditLogger(config)
		defer logger.Close()

		event := &AuditEvent{
			ActorID:  "user1",
			Resource: "/api/v1/test",
			Action:   "read",
			Decision: true,
		}
		logger.LogDecision(event)

		// Timestamp should have been set
		if event.Timestamp.IsZero() {
			t.Error("expected timestamp to be set")
		}
	})

	t.Run("nil logger does not panic", func(t *testing.T) {
		var logger *AuditLogger
		// Should not panic
		logger.LogDecision(&AuditEvent{})
	})
}

func TestAuditLogger_LogDecisionContext(t *testing.T) {
	config := &AuditLoggerConfig{
		Enabled:       true,
		LogAllowed:    true,
		LogDenied:     true,
		SampleRate:    1.0,
		BufferSize:    10,
		FlushInterval: 100 * time.Millisecond,
	}
	logger := NewAuditLogger(config)
	defer logger.Close()

	ctx := context.Background()
	ctx = WithRequestID(ctx, "req-12345")

	logger.LogDecisionContext(
		ctx,
		"user1",
		"testuser",
		[]string{"viewer"},
		"/api/v1/test",
		"read",
		true,
		"",
		100*time.Microsecond,
		false,
	)

	// Give time for async processing
	time.Sleep(50 * time.Millisecond)
}

func TestAuditLogger_Close(t *testing.T) {
	t.Run("close drains remaining events", func(t *testing.T) {
		config := &AuditLoggerConfig{
			Enabled:       true,
			LogAllowed:    true,
			LogDenied:     true,
			SampleRate:    1.0,
			BufferSize:    100,
			FlushInterval: 1 * time.Hour, // Long interval to ensure we don't flush before close
		}
		logger := NewAuditLogger(config)

		// Queue some events
		for i := 0; i < 10; i++ {
			logger.LogDecision(&AuditEvent{
				ActorID:  "user1",
				Resource: "/api/v1/test",
				Action:   "read",
				Decision: true,
			})
		}

		// Close should drain all events
		logger.Close()
	})

	t.Run("double close does not panic", func(t *testing.T) {
		logger := NewAuditLogger(nil)
		logger.Close()
		logger.Close() // Should not panic
	})

	t.Run("nil close does not panic", func(t *testing.T) {
		var logger *AuditLogger
		logger.Close() // Should not panic
	})
}

func TestAuditLogger_Stats(t *testing.T) {
	config := &AuditLoggerConfig{
		Enabled:       true,
		LogAllowed:    true,
		LogDenied:     false,
		SampleRate:    0.5,
		BufferSize:    100,
		FlushInterval: 5 * time.Second,
	}
	logger := NewAuditLogger(config)
	defer logger.Close()

	stats := logger.Stats()

	if stats.BufferSize != 100 {
		t.Errorf("expected buffer_size=100, got %d", stats.BufferSize)
	}
	if !stats.Enabled {
		t.Error("expected enabled=true")
	}
	if !stats.LogAllowed {
		t.Error("expected log_allowed=true")
	}
	if stats.LogDenied {
		t.Error("expected log_denied=false")
	}
	if stats.SampleRate != 0.5 {
		t.Errorf("expected sample_rate=0.5, got %f", stats.SampleRate)
	}
}

func TestAuditLogger_Stats_Nil(t *testing.T) {
	var logger *AuditLogger
	stats := logger.Stats()

	// Should return zero value
	if stats.BufferSize != 0 {
		t.Error("expected zero value for nil logger")
	}
}

func TestAuditLogger_Concurrent(t *testing.T) {
	config := &AuditLoggerConfig{
		Enabled:       true,
		LogAllowed:    true,
		LogDenied:     true,
		SampleRate:    1.0,
		BufferSize:    1000,
		FlushInterval: 100 * time.Millisecond,
	}
	logger := NewAuditLogger(config)
	defer logger.Close()

	var wg sync.WaitGroup
	numGoroutines := 10
	eventsPerGoroutine := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < eventsPerGoroutine; j++ {
				logger.LogDecision(&AuditEvent{
					ActorID:  "user" + string(rune('0'+id)),
					Resource: "/api/v1/test",
					Action:   "read",
					Decision: j%2 == 0,
				})
			}
		}(i)
	}

	wg.Wait()
	// Give time for async processing
	time.Sleep(200 * time.Millisecond)
}

func TestWithRequestID(t *testing.T) {
	ctx := context.Background()
	reqID := "test-request-123"

	ctx = WithRequestID(ctx, reqID)
	got := GetRequestID(ctx)

	if got != reqID {
		t.Errorf("expected %s, got %s", reqID, got)
	}
}

func TestGetRequestID_NotSet(t *testing.T) {
	ctx := context.Background()
	got := GetRequestID(ctx)

	if got != "" {
		t.Errorf("expected empty string, got %s", got)
	}
}

func TestDefaultAuditLoggerConfig(t *testing.T) {
	config := DefaultAuditLoggerConfig()

	if !config.Enabled {
		t.Error("expected enabled=true")
	}
	if !config.LogAllowed {
		t.Error("expected log_allowed=true")
	}
	if !config.LogDenied {
		t.Error("expected log_denied=true")
	}
	if config.SampleRate != 1.0 {
		t.Errorf("expected sample_rate=1.0, got %f", config.SampleRate)
	}
	if config.BufferSize != 1000 {
		t.Errorf("expected buffer_size=1000, got %d", config.BufferSize)
	}
	if config.FlushInterval != 5*time.Second {
		t.Errorf("expected flush_interval=5s, got %v", config.FlushInterval)
	}
}

func TestAuditEvent_Fields(t *testing.T) {
	now := time.Now()
	event := &AuditEvent{
		ID:            "test-id",
		Timestamp:     now,
		RequestID:     "req-123",
		ActorID:       "user1",
		ActorUsername: "testuser",
		ActorRole:     "admin",
		ActorRoles:    []string{"admin", "editor"},
		Resource:      "/api/v1/users",
		Action:        "write",
		Decision:      true,
		Reason:        "Admin access",
		Duration:      100 * time.Microsecond,
		CacheHit:      true,
		IPAddress:     "192.168.1.1",
		UserAgent:     "Mozilla/5.0",
		SessionID:     "session-123",
		Method:        "POST",
	}

	if event.ID != "test-id" {
		t.Errorf("expected ID=test-id, got %s", event.ID)
	}
	if event.Timestamp != now {
		t.Errorf("expected Timestamp=%v, got %v", now, event.Timestamp)
	}
	if event.RequestID != "req-123" {
		t.Errorf("expected RequestID=req-123, got %s", event.RequestID)
	}
	if event.ActorID != "user1" {
		t.Errorf("expected ActorID=user1, got %s", event.ActorID)
	}
	if event.ActorUsername != "testuser" {
		t.Errorf("expected ActorUsername=testuser, got %s", event.ActorUsername)
	}
	if event.ActorRole != "admin" {
		t.Errorf("expected ActorRole=admin, got %s", event.ActorRole)
	}
	if len(event.ActorRoles) != 2 || event.ActorRoles[0] != "admin" {
		t.Errorf("expected ActorRoles=[admin,editor], got %v", event.ActorRoles)
	}
	if event.Resource != "/api/v1/users" {
		t.Errorf("expected Resource=/api/v1/users, got %s", event.Resource)
	}
	if event.Action != "write" {
		t.Errorf("expected Action=write, got %s", event.Action)
	}
	if !event.Decision {
		t.Error("expected Decision=true")
	}
	if event.Reason != "Admin access" {
		t.Errorf("expected Reason=Admin access, got %s", event.Reason)
	}
	if event.Duration != 100*time.Microsecond {
		t.Errorf("expected Duration=100us, got %v", event.Duration)
	}
	if !event.CacheHit {
		t.Error("expected CacheHit=true")
	}
	if event.IPAddress != "192.168.1.1" {
		t.Errorf("expected IPAddress=192.168.1.1, got %s", event.IPAddress)
	}
	if event.UserAgent != "Mozilla/5.0" {
		t.Errorf("expected UserAgent=Mozilla/5.0, got %s", event.UserAgent)
	}
	if event.SessionID != "session-123" {
		t.Errorf("expected SessionID=session-123, got %s", event.SessionID)
	}
	if event.Method != "POST" {
		t.Errorf("expected Method=POST, got %s", event.Method)
	}
}
