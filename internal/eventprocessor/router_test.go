// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build nats

package eventprocessor

import (
	"context"
	"testing"
	"time"
)

func TestDefaultRouterConfig(t *testing.T) {
	t.Parallel()

	cfg := DefaultRouterConfig()

	if cfg.CloseTimeout != 30*time.Second {
		t.Errorf("CloseTimeout = %v, want %v", cfg.CloseTimeout, 30*time.Second)
	}
	if cfg.RetryMaxRetries != 5 {
		t.Errorf("RetryMaxRetries = %d, want 5", cfg.RetryMaxRetries)
	}
	if cfg.RetryInitialInterval != time.Second {
		t.Errorf("RetryInitialInterval = %v, want %v", cfg.RetryInitialInterval, time.Second)
	}
	if cfg.RetryMaxInterval != time.Minute {
		t.Errorf("RetryMaxInterval = %v, want %v", cfg.RetryMaxInterval, time.Minute)
	}
	if cfg.RetryMultiplier != 2.0 {
		t.Errorf("RetryMultiplier = %f, want 2.0", cfg.RetryMultiplier)
	}
	if cfg.ThrottlePerSecond != 0 {
		t.Errorf("ThrottlePerSecond = %d, want 0", cfg.ThrottlePerSecond)
	}
	if cfg.PoisonQueueTopic != "dlq.playback" {
		t.Errorf("PoisonQueueTopic = %q, want %q", cfg.PoisonQueueTopic, "dlq.playback")
	}
	if cfg.DeduplicationEnabled {
		t.Error("DeduplicationEnabled should be false by default (uses msg.UUID which may be regenerated, causing data loss)")
	}
	if cfg.DeduplicationTTL != 5*time.Minute {
		t.Errorf("DeduplicationTTL = %v, want %v", cfg.DeduplicationTTL, 5*time.Minute)
	}
}

func TestInMemoryDeduplicator(t *testing.T) {
	t.Parallel()

	ttl := 100 * time.Millisecond
	dedup := NewInMemoryDeduplicator(ttl)
	ctx := context.Background()

	// First call should not be duplicate
	isDup, err := dedup.IsDuplicate(ctx, "key1")
	if err != nil {
		t.Fatalf("IsDuplicate error: %v", err)
	}
	if isDup {
		t.Error("First call should not be duplicate")
	}

	// Second call with same key should be duplicate
	isDup, err = dedup.IsDuplicate(ctx, "key1")
	if err != nil {
		t.Fatalf("IsDuplicate error: %v", err)
	}
	if !isDup {
		t.Error("Second call with same key should be duplicate")
	}

	// Different key should not be duplicate
	isDup, err = dedup.IsDuplicate(ctx, "key2")
	if err != nil {
		t.Fatalf("IsDuplicate error: %v", err)
	}
	if isDup {
		t.Error("Different key should not be duplicate")
	}

	// After TTL expires, key should not be duplicate
	time.Sleep(ttl + 10*time.Millisecond)
	isDup, err = dedup.IsDuplicate(ctx, "key1")
	if err != nil {
		t.Fatalf("IsDuplicate error: %v", err)
	}
	if isDup {
		t.Error("After TTL, key should not be duplicate")
	}
}

func TestRouterMetrics(t *testing.T) {
	t.Parallel()

	metrics := &RouterMetrics{
		MessagesReceived:  100,
		MessagesProcessed: 95,
		MessagesFailed:    3,
	}

	if metrics.MessagesReceived != 100 {
		t.Errorf("MessagesReceived = %d, want 100", metrics.MessagesReceived)
	}
	if metrics.MessagesProcessed != 95 {
		t.Errorf("MessagesProcessed = %d, want 95", metrics.MessagesProcessed)
	}
	if metrics.MessagesFailed != 3 {
		t.Errorf("MessagesFailed = %d, want 3", metrics.MessagesFailed)
	}
}

// TestNewRouter_NilLogger verifies router creation with nil logger uses default.
func TestNewRouter_NilLogger(t *testing.T) {
	t.Parallel()

	cfg := DefaultRouterConfig()
	cfg.PoisonQueueTopic = "" // Disable poison queue for this test

	router, err := NewRouter(&cfg, nil, nil)
	if err != nil {
		t.Fatalf("NewRouter error: %v", err)
	}
	if router == nil {
		t.Fatal("NewRouter returned nil")
	}
	defer router.Close()

	if router.config.CloseTimeout != cfg.CloseTimeout {
		t.Error("Router config not set correctly")
	}
}

// TestNewRouter_WithDeduplication verifies deduplicator middleware is added.
func TestNewRouter_WithDeduplication(t *testing.T) {
	t.Parallel()

	cfg := DefaultRouterConfig()
	cfg.PoisonQueueTopic = ""
	cfg.DeduplicationEnabled = true
	cfg.DeduplicationTTL = time.Minute

	router, err := NewRouter(&cfg, nil, nil)
	if err != nil {
		t.Fatalf("NewRouter error: %v", err)
	}
	defer router.Close()

	if router.dedupRepo == nil {
		t.Error("Deduplicator repository should be created when enabled")
	}
}

// TestNewRouter_WithThrottle verifies throttle middleware configuration.
func TestNewRouter_WithThrottle(t *testing.T) {
	t.Parallel()

	cfg := DefaultRouterConfig()
	cfg.PoisonQueueTopic = ""
	cfg.ThrottlePerSecond = 100

	router, err := NewRouter(&cfg, nil, nil)
	if err != nil {
		t.Fatalf("NewRouter error: %v", err)
	}
	defer router.Close()

	// Router should be created successfully with throttle
	if router.config.ThrottlePerSecond != 100 {
		t.Errorf("ThrottlePerSecond = %d, want 100", router.config.ThrottlePerSecond)
	}
}

// TestRouter_IsRunning verifies running state tracking.
func TestRouter_IsRunning(t *testing.T) {
	t.Parallel()

	cfg := DefaultRouterConfig()
	cfg.PoisonQueueTopic = ""

	router, err := NewRouter(&cfg, nil, nil)
	if err != nil {
		t.Fatalf("NewRouter error: %v", err)
	}

	// Should not be running initially
	if router.IsRunning() {
		t.Error("Router should not be running before Run()")
	}

	// Close without running
	if err := router.Close(); err != nil {
		t.Errorf("Close error: %v", err)
	}
}

// TestRouter_Metrics verifies metrics are accessible.
func TestRouter_Metrics(t *testing.T) {
	t.Parallel()

	cfg := DefaultRouterConfig()
	cfg.PoisonQueueTopic = ""

	router, err := NewRouter(&cfg, nil, nil)
	if err != nil {
		t.Fatalf("NewRouter error: %v", err)
	}
	defer router.Close()

	metrics := router.Metrics()
	if metrics == nil {
		t.Fatal("Metrics() returned nil")
	}

	// Initial metrics should be zero
	if metrics.MessagesReceived != 0 {
		t.Errorf("Initial MessagesReceived = %d, want 0", metrics.MessagesReceived)
	}
}

// TestRouter_RunAsync verifies async run returns running channel.
func TestRouter_RunAsync(t *testing.T) {
	t.Parallel()

	cfg := DefaultRouterConfig()
	cfg.PoisonQueueTopic = ""
	cfg.CloseTimeout = 100 * time.Millisecond

	router, err := NewRouter(&cfg, nil, nil)
	if err != nil {
		t.Fatalf("NewRouter error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	running := router.RunAsync(ctx)

	select {
	case <-running:
		// Router started
	case <-time.After(time.Second):
		t.Error("Router did not start within timeout")
	}

	// Allow router to run briefly
	time.Sleep(50 * time.Millisecond)

	if err := router.Close(); err != nil {
		t.Errorf("Close error: %v", err)
	}
}

// TestRouter_AddHandlerMiddleware_NotFound verifies error for unknown handler.
func TestRouter_AddHandlerMiddleware_NotFound(t *testing.T) {
	t.Parallel()

	cfg := DefaultRouterConfig()
	cfg.PoisonQueueTopic = ""

	router, err := NewRouter(&cfg, nil, nil)
	if err != nil {
		t.Fatalf("NewRouter error: %v", err)
	}
	defer router.Close()

	err = router.AddHandlerMiddleware("nonexistent", nil)
	if err == nil {
		t.Error("AddHandlerMiddleware should error for unknown handler")
	}
}
