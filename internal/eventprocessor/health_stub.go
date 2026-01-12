// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build !nats

package eventprocessor

import (
	"context"
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
	Timeout  time.Duration
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
	Healthy   bool                   `json:"healthy"`
	Degraded  bool                   `json:"degraded,omitempty"`
	Name      string                 `json:"name"`
	Message   string                 `json:"message,omitempty"`
	Error     string                 `json:"error,omitempty"`
	LastCheck time.Time              `json:"last_check"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

// HealthCheckable is implemented by components that support health checking.
type HealthCheckable interface {
	HealthCheck(ctx context.Context) ComponentHealth
}

// OverallHealth represents the aggregated health status of all components.
type OverallHealth struct {
	Healthy    bool                       `json:"healthy"`
	Status     HealthStatusType           `json:"status"`
	Timestamp  time.Time                  `json:"timestamp"`
	Components map[string]ComponentHealth `json:"components"`
}

// HealthChecker manages health checks for multiple components.
type HealthChecker struct {
	config HealthConfig
}

// NewHealthChecker creates a new health checker.
func NewHealthChecker(cfg HealthConfig) *HealthChecker {
	return &HealthChecker{config: cfg}
}

// RegisterComponent registers a component for health checking.
func (h *HealthChecker) RegisterComponent(name string, component HealthCheckable) {}

// UnregisterComponent removes a component from health checking.
func (h *HealthChecker) UnregisterComponent(name string) {}

// CheckAll performs health checks on all registered components.
func (h *HealthChecker) CheckAll(ctx context.Context) OverallHealth {
	return OverallHealth{
		Healthy:    false,
		Status:     HealthStatusUnhealthy,
		Timestamp:  time.Now(),
		Components: make(map[string]ComponentHealth),
	}
}

// CheckComponent performs a health check on a specific component.
func (h *HealthChecker) CheckComponent(ctx context.Context, name string) ComponentHealth {
	return ComponentHealth{
		Name:      name,
		Healthy:   false,
		Error:     "NATS not enabled",
		LastCheck: time.Now(),
	}
}
