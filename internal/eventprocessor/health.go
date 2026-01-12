// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build nats

package eventprocessor

import (
	"context"
	"sync"
	"time"
)

// HealthStatusType represents the overall health status.
type HealthStatusType string

const (
	// HealthStatusHealthy indicates all components are functioning normally.
	HealthStatusHealthy HealthStatusType = "healthy"
	// HealthStatusDegraded indicates some components are experiencing issues but still operational.
	HealthStatusDegraded HealthStatusType = "degraded"
	// HealthStatusUnhealthy indicates critical components are failing.
	HealthStatusUnhealthy HealthStatusType = "unhealthy"
)

// HealthConfig holds configuration for health checking.
type HealthConfig struct {
	// Timeout is the maximum time to wait for health checks.
	Timeout time.Duration
	// Interval is how often to run periodic health checks.
	Interval time.Duration
}

// DefaultHealthConfig returns sensible defaults for health checking.
func DefaultHealthConfig() HealthConfig {
	return HealthConfig{
		Timeout:  5 * time.Second,
		Interval: 30 * time.Second,
	}
}

// ComponentHealth represents the health status of a single component.
type ComponentHealth struct {
	// Healthy indicates whether the component is functioning.
	Healthy bool `json:"healthy"`
	// Degraded indicates the component is operational but experiencing issues.
	Degraded bool `json:"degraded,omitempty"`
	// Name is the component identifier.
	Name string `json:"name"`
	// Message provides additional context about the health status.
	Message string `json:"message,omitempty"`
	// Error contains error details if unhealthy.
	Error string `json:"error,omitempty"`
	// LastCheck is when the health check was performed.
	LastCheck time.Time `json:"last_check"`
	// Details contains component-specific health information.
	Details map[string]interface{} `json:"details,omitempty"`
}

// HealthCheckable is implemented by components that support health checking.
type HealthCheckable interface {
	// HealthCheck performs a health check and returns the result.
	HealthCheck(ctx context.Context) ComponentHealth
}

// OverallHealth represents the aggregated health status of all components.
type OverallHealth struct {
	// Healthy indicates whether all critical components are healthy.
	Healthy bool `json:"healthy"`
	// Status is the overall health status.
	Status HealthStatusType `json:"status"`
	// Timestamp is when this health check was performed.
	Timestamp time.Time `json:"timestamp"`
	// Components contains individual component health.
	Components map[string]ComponentHealth `json:"components"`
}

// HealthChecker manages health checks for multiple components.
type HealthChecker struct {
	config     HealthConfig
	mu         sync.RWMutex
	components map[string]HealthCheckable
}

// NewHealthChecker creates a new health checker.
func NewHealthChecker(cfg HealthConfig) *HealthChecker {
	return &HealthChecker{
		config:     cfg,
		components: make(map[string]HealthCheckable),
	}
}

// RegisterComponent registers a component for health checking.
func (h *HealthChecker) RegisterComponent(name string, component HealthCheckable) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.components[name] = component
}

// UnregisterComponent removes a component from health checking.
func (h *HealthChecker) UnregisterComponent(name string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.components, name)
}

// CheckAll performs health checks on all registered components.
func (h *HealthChecker) CheckAll(ctx context.Context) OverallHealth {
	h.mu.RLock()
	componentsCopy := make(map[string]HealthCheckable, len(h.components))
	for name, comp := range h.components {
		componentsCopy[name] = comp
	}
	h.mu.RUnlock()

	overall := OverallHealth{
		Healthy:    true,
		Status:     HealthStatusHealthy,
		Timestamp:  time.Now(),
		Components: make(map[string]ComponentHealth),
	}

	var wg sync.WaitGroup
	var mu sync.Mutex

	for name, component := range componentsCopy {
		wg.Add(1)
		go func(name string, comp HealthCheckable) {
			defer wg.Done()

			// Create timeout context for individual check
			checkCtx, cancel := context.WithTimeout(ctx, h.config.Timeout)
			defer cancel()

			// Channel to receive health check result
			resultCh := make(chan ComponentHealth, 1)
			go func() {
				result := comp.HealthCheck(checkCtx)
				result.Name = name
				result.LastCheck = time.Now()
				resultCh <- result
			}()

			var result ComponentHealth
			select {
			case result = <-resultCh:
			case <-checkCtx.Done():
				result = ComponentHealth{
					Name:      name,
					Healthy:   false,
					Error:     "health check timeout",
					LastCheck: time.Now(),
				}
			}

			mu.Lock()
			overall.Components[name] = result

			// Update overall status
			if !result.Healthy {
				overall.Healthy = false
				overall.Status = HealthStatusUnhealthy
			} else if result.Degraded && overall.Status == HealthStatusHealthy {
				overall.Status = HealthStatusDegraded
			}
			mu.Unlock()
		}(name, component)
	}

	wg.Wait()
	return overall
}

// CheckComponent performs a health check on a specific component.
func (h *HealthChecker) CheckComponent(ctx context.Context, name string) ComponentHealth {
	h.mu.RLock()
	component, exists := h.components[name]
	h.mu.RUnlock()

	if !exists {
		return ComponentHealth{
			Name:      name,
			Healthy:   false,
			Error:     "component not found",
			LastCheck: time.Now(),
		}
	}

	checkCtx, cancel := context.WithTimeout(ctx, h.config.Timeout)
	defer cancel()

	resultCh := make(chan ComponentHealth, 1)
	go func() {
		result := component.HealthCheck(checkCtx)
		result.Name = name
		result.LastCheck = time.Now()
		resultCh <- result
	}()

	select {
	case result := <-resultCh:
		return result
	case <-checkCtx.Done():
		return ComponentHealth{
			Name:      name,
			Healthy:   false,
			Error:     "health check timeout",
			LastCheck: time.Now(),
		}
	}
}

// HealthCheck implements HealthCheckable for DuckDBConsumer.
func (c *DuckDBConsumer) HealthCheck(ctx context.Context) ComponentHealth {
	stats := c.Stats()

	details := map[string]interface{}{
		"messages_received":    stats.MessagesReceived,
		"messages_processed":   stats.MessagesProcessed,
		"parse_errors":         stats.ParseErrors,
		"duplicates_skipped":   stats.DuplicatesSkipped,
		"messages_sent_to_dlq": stats.MessagesSentToDLQ,
	}

	if !stats.LastMessageTime.IsZero() {
		details["last_message_time"] = stats.LastMessageTime.Format(time.RFC3339)
		details["time_since_last_message"] = time.Since(stats.LastMessageTime).String()
	}

	if !c.running.Load() {
		return ComponentHealth{
			Healthy: false,
			Error:   "consumer is not running",
			Details: details,
		}
	}

	// Check for degraded state (high error rate)
	if stats.MessagesReceived > 100 {
		errorRate := float64(stats.ParseErrors) / float64(stats.MessagesReceived)
		if errorRate > 0.1 { // More than 10% parse errors
			return ComponentHealth{
				Healthy:  true,
				Degraded: true,
				Message:  "high parse error rate",
				Details:  details,
			}
		}
	}

	return ComponentHealth{
		Healthy: true,
		Message: "consumer is running",
		Details: details,
	}
}

// HealthCheck implements HealthCheckable for DLQHandler.
func (h *DLQHandler) HealthCheck(ctx context.Context) ComponentHealth {
	stats := h.Stats()

	details := map[string]interface{}{
		"entry_count":   stats.TotalEntries,
		"total_added":   stats.TotalAdded,
		"total_removed": stats.TotalRemoved,
		"total_retries": stats.TotalRetries,
		"total_expired": stats.TotalExpired,
	}

	if !stats.OldestEntry.IsZero() {
		details["oldest_entry"] = stats.OldestEntry.Format(time.RFC3339)
		details["oldest_entry_age"] = time.Since(stats.OldestEntry).String()
	}

	// Check for degraded state (too many entries)
	if stats.TotalEntries > int64(h.config.MaxEntries/2) {
		return ComponentHealth{
			Healthy:  true,
			Degraded: true,
			Message:  "DLQ is filling up",
			Details:  details,
		}
	}

	return ComponentHealth{
		Healthy: true,
		Message: "DLQ handler is operational",
		Details: details,
	}
}

// HealthCheck implements HealthCheckable for Publisher.
func (p *Publisher) HealthCheck(ctx context.Context) ComponentHealth {
	p.mu.RLock()
	closed := p.closed
	p.mu.RUnlock()

	if closed {
		return ComponentHealth{
			Healthy: false,
			Error:   "publisher is closed",
		}
	}

	details := map[string]interface{}{}

	// Check circuit breaker state if configured
	if p.circuitBreaker != nil {
		state := p.circuitBreaker.State()
		details["circuit_breaker_state"] = state.String()

		switch state {
		case 2: // Open
			return ComponentHealth{
				Healthy: false,
				Error:   "circuit breaker is open",
				Details: details,
			}
		case 1: // Half-Open
			return ComponentHealth{
				Healthy:  true,
				Degraded: true,
				Message:  "circuit breaker is half-open",
				Details:  details,
			}
		}
	}

	return ComponentHealth{
		Healthy: true,
		Message: "publisher is operational",
		Details: details,
	}
}

// HealthCheck implements HealthCheckable for Appender.
func (a *Appender) HealthCheck(ctx context.Context) ComponentHealth {
	stats := a.Stats()

	details := map[string]interface{}{
		"events_received": stats.EventsReceived,
		"events_flushed":  stats.EventsFlushed,
		"flush_count":     stats.FlushCount,
		"error_count":     stats.ErrorCount,
		"buffer_size":     stats.BufferSize,
		"avg_flush_time":  stats.AvgFlushTime.String(),
	}

	if !stats.LastFlushTime.IsZero() {
		details["last_flush_time"] = stats.LastFlushTime.Format(time.RFC3339)
	}

	if a.closed.Load() {
		return ComponentHealth{
			Healthy: false,
			Error:   "appender is closed",
			Details: details,
		}
	}

	// Check for degraded state (high error rate or buffer filling up)
	if stats.FlushCount > 10 && stats.ErrorCount > 0 {
		errorRate := float64(stats.ErrorCount) / float64(stats.FlushCount)
		if errorRate > 0.1 {
			return ComponentHealth{
				Healthy:  true,
				Degraded: true,
				Message:  "high flush error rate",
				Details:  details,
			}
		}
	}

	// Check buffer capacity
	if stats.BufferSize > a.config.BatchSize*2 {
		return ComponentHealth{
			Healthy:  true,
			Degraded: true,
			Message:  "buffer is filling up",
			Details:  details,
		}
	}

	if stats.LastError != "" {
		details["last_error"] = stats.LastError
	}

	return ComponentHealth{
		Healthy: true,
		Message: "appender is operational",
		Details: details,
	}
}
