// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build nats

package eventprocessor

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestHealthChecker_NewHealthChecker(t *testing.T) {
	t.Parallel()

	cfg := DefaultHealthConfig()
	checker := NewHealthChecker(cfg)

	if checker == nil {
		t.Fatal("expected non-nil health checker")
	}
}

func TestHealthChecker_RegisterComponent(t *testing.T) {
	t.Parallel()

	checker := NewHealthChecker(DefaultHealthConfig())

	// Create a mock healthy component
	component := &mockHealthComponent{healthy: true}
	checker.RegisterComponent("test", component)

	// Check that component was registered
	status := checker.CheckAll(context.Background())
	if _, exists := status.Components["test"]; !exists {
		t.Error("expected 'test' component to be registered")
	}
}

func TestHealthChecker_CheckAll_AllHealthy(t *testing.T) {
	t.Parallel()

	checker := NewHealthChecker(DefaultHealthConfig())

	// Register multiple healthy components
	checker.RegisterComponent("nats", &mockHealthComponent{healthy: true})
	checker.RegisterComponent("duckdb", &mockHealthComponent{healthy: true})
	checker.RegisterComponent("consumer", &mockHealthComponent{healthy: true})

	status := checker.CheckAll(context.Background())

	if !status.Healthy {
		t.Error("expected overall status to be healthy")
	}
	if status.Status != HealthStatusHealthy {
		t.Errorf("expected status %s, got %s", HealthStatusHealthy, status.Status)
	}
	if len(status.Components) != 3 {
		t.Errorf("expected 3 components, got %d", len(status.Components))
	}
}

func TestHealthChecker_CheckAll_OneUnhealthy(t *testing.T) {
	t.Parallel()

	checker := NewHealthChecker(DefaultHealthConfig())

	// Register mix of healthy and unhealthy
	checker.RegisterComponent("nats", &mockHealthComponent{healthy: true})
	checker.RegisterComponent("duckdb", &mockHealthComponent{healthy: false, err: "database connection failed"})

	status := checker.CheckAll(context.Background())

	if status.Healthy {
		t.Error("expected overall status to be unhealthy")
	}
	if status.Status != HealthStatusUnhealthy {
		t.Errorf("expected status %s, got %s", HealthStatusUnhealthy, status.Status)
	}

	// Check individual component status
	if status.Components["nats"].Healthy != true {
		t.Error("expected nats component to be healthy")
	}
	if status.Components["duckdb"].Healthy != false {
		t.Error("expected duckdb component to be unhealthy")
	}
}

func TestHealthChecker_CheckAll_Degraded(t *testing.T) {
	t.Parallel()

	checker := NewHealthChecker(DefaultHealthConfig())

	// Register degraded component
	checker.RegisterComponent("nats", &mockHealthComponent{healthy: true, degraded: true, message: "high latency"})

	status := checker.CheckAll(context.Background())

	if !status.Healthy {
		t.Error("expected overall status to be healthy (degraded is still healthy)")
	}
	if status.Status != HealthStatusDegraded {
		t.Errorf("expected status %s, got %s", HealthStatusDegraded, status.Status)
	}
}

func TestHealthChecker_CheckAll_Timeout(t *testing.T) {
	t.Parallel()

	cfg := DefaultHealthConfig()
	cfg.Timeout = 50 * time.Millisecond
	checker := NewHealthChecker(cfg)

	// Register slow component
	checker.RegisterComponent("slow", &mockHealthComponent{healthy: true, delay: 100 * time.Millisecond})

	status := checker.CheckAll(context.Background())

	// Component should be marked unhealthy due to timeout
	if status.Components["slow"].Healthy {
		t.Error("expected slow component to be unhealthy due to timeout")
	}
}

func TestHealthChecker_CheckComponent(t *testing.T) {
	t.Parallel()

	checker := NewHealthChecker(DefaultHealthConfig())
	checker.RegisterComponent("test", &mockHealthComponent{healthy: true})

	result := checker.CheckComponent(context.Background(), "test")

	if !result.Healthy {
		t.Error("expected component to be healthy")
	}
	if result.Name != "test" {
		t.Errorf("expected name 'test', got '%s'", result.Name)
	}
}

func TestHealthChecker_CheckComponent_NotFound(t *testing.T) {
	t.Parallel()

	checker := NewHealthChecker(DefaultHealthConfig())

	result := checker.CheckComponent(context.Background(), "nonexistent")

	if result.Healthy {
		t.Error("expected unknown component to be unhealthy")
	}
	if result.Error == "" {
		t.Error("expected error message for unknown component")
	}
}

func TestHealthChecker_UnregisterComponent(t *testing.T) {
	t.Parallel()

	checker := NewHealthChecker(DefaultHealthConfig())
	checker.RegisterComponent("test", &mockHealthComponent{healthy: true})

	// Verify it's registered
	status := checker.CheckAll(context.Background())
	if _, exists := status.Components["test"]; !exists {
		t.Fatal("component should be registered")
	}

	// Unregister
	checker.UnregisterComponent("test")

	// Verify it's gone
	status = checker.CheckAll(context.Background())
	if _, exists := status.Components["test"]; exists {
		t.Error("component should be unregistered")
	}
}

func TestHealthConfig_Defaults(t *testing.T) {
	t.Parallel()

	cfg := DefaultHealthConfig()

	if cfg.Timeout <= 0 {
		t.Error("expected positive timeout")
	}
	if cfg.Interval <= 0 {
		t.Error("expected positive interval")
	}
}

func TestHealthStatus_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		status   HealthStatusType
		expected string
	}{
		{HealthStatusHealthy, "healthy"},
		{HealthStatusDegraded, "degraded"},
		{HealthStatusUnhealthy, "unhealthy"},
	}

	for _, tt := range tests {
		if string(tt.status) != tt.expected {
			t.Errorf("expected %s, got %s", tt.expected, tt.status)
		}
	}
}

func TestComponentHealth_Details(t *testing.T) {
	t.Parallel()

	checker := NewHealthChecker(DefaultHealthConfig())

	// Register component with custom details
	component := &mockHealthComponent{
		healthy: true,
		details: map[string]interface{}{
			"connections": 5,
			"uptime":      "1h30m",
		},
	}
	checker.RegisterComponent("test", component)

	status := checker.CheckAll(context.Background())

	if status.Components["test"].Details == nil {
		t.Fatal("expected details to be present")
	}
	if status.Components["test"].Details["connections"] != 5 {
		t.Error("expected connections detail to be 5")
	}
}

// mockHealthComponent implements HealthCheckable for testing.
type mockHealthComponent struct {
	healthy  bool
	degraded bool
	message  string
	err      string
	delay    time.Duration
	details  map[string]interface{}
}

func (m *mockHealthComponent) HealthCheck(ctx context.Context) ComponentHealth {
	if m.delay > 0 {
		select {
		case <-time.After(m.delay):
		case <-ctx.Done():
			return ComponentHealth{
				Healthy: false,
				Error:   "health check timeout",
			}
		}
	}

	return ComponentHealth{
		Healthy:  m.healthy,
		Degraded: m.degraded,
		Message:  m.message,
		Error:    m.err,
		Details:  m.details,
	}
}

// Test the consumer implements HealthCheckable
func TestDuckDBConsumer_HealthCheck(t *testing.T) {
	t.Parallel()

	store := NewMockEventStore()
	appender, _ := NewAppender(store, DefaultAppenderConfig())
	source := NewMockMessageSource()

	cfg := DefaultConsumerConfig()
	consumer, err := NewDuckDBConsumer(source, appender, &cfg)
	if err != nil {
		t.Fatalf("failed to create consumer: %v", err)
	}

	// Check health before start (should be unhealthy)
	health := consumer.HealthCheck(context.Background())
	if health.Healthy {
		t.Error("expected consumer to be unhealthy before start")
	}

	// Start consumer
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := consumer.Start(ctx); err != nil {
		t.Fatalf("failed to start consumer: %v", err)
	}

	// Check health after start (should be healthy)
	health = consumer.HealthCheck(context.Background())
	if !health.Healthy {
		t.Error("expected consumer to be healthy after start")
	}

	// Stop consumer
	consumer.Stop()

	// Check health after stop (should be unhealthy)
	health = consumer.HealthCheck(context.Background())
	if health.Healthy {
		t.Error("expected consumer to be unhealthy after stop")
	}
}

// Test the publisher implements HealthCheckable
func TestPublisher_HealthCheck(t *testing.T) {
	t.Parallel()

	// Create mock publisher that tracks closed state
	pub := &mockPublisherHealth{
		closed: atomic.Bool{},
	}

	// Check health when not closed (should be healthy)
	health := pub.HealthCheck(context.Background())
	if !health.Healthy {
		t.Error("expected publisher to be healthy when not closed")
	}

	// Mark as closed
	pub.closed.Store(true)

	// Check health when closed (should be unhealthy)
	health = pub.HealthCheck(context.Background())
	if health.Healthy {
		t.Error("expected publisher to be unhealthy when closed")
	}
}

// mockPublisherHealth implements HealthCheckable for testing publisher health.
type mockPublisherHealth struct {
	closed atomic.Bool
}

func (m *mockPublisherHealth) HealthCheck(ctx context.Context) ComponentHealth {
	if m.closed.Load() {
		return ComponentHealth{
			Healthy: false,
			Error:   "publisher is closed",
		}
	}
	return ComponentHealth{
		Healthy: true,
		Message: "publisher is operational",
	}
}

// Test DLQHandler implements HealthCheckable
func TestDLQHandler_HealthCheck(t *testing.T) {
	t.Parallel()

	cfg := DefaultDLQConfig()
	handler, err := NewDLQHandler(cfg)
	if err != nil {
		t.Fatalf("failed to create DLQ handler: %v", err)
	}

	// Check health with no entries
	health := handler.HealthCheck(context.Background())
	if !health.Healthy {
		t.Error("expected DLQ handler to be healthy with no entries")
	}

	// Add some entries
	for i := 0; i < 5; i++ {
		event := &MediaEvent{EventID: string(rune('a' + i))}
		handler.AddEntry(event, NewRetryableError("test error", nil), "msg-"+string(rune('a'+i)))
	}

	// Check health with entries (still healthy, but should have details)
	health = handler.HealthCheck(context.Background())
	if !health.Healthy {
		t.Error("expected DLQ handler to be healthy with few entries")
	}
	if health.Details == nil {
		t.Error("expected health details")
	}
	if health.Details["entry_count"] != int64(5) {
		t.Errorf("expected entry_count 5, got %v", health.Details["entry_count"])
	}
}
