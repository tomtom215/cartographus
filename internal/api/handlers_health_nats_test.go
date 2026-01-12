// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build nats

package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tomtom215/cartographus/internal/eventprocessor"
)

// mockHealthChecker implements NATSHealthChecker for testing.
type mockNATSHealthChecker struct {
	overallHealth   eventprocessor.OverallHealth
	componentHealth map[string]eventprocessor.ComponentHealth
}

func (m *mockNATSHealthChecker) CheckAll(ctx context.Context) eventprocessor.OverallHealth {
	return m.overallHealth
}

func (m *mockNATSHealthChecker) CheckComponent(ctx context.Context, name string) eventprocessor.ComponentHealth {
	if h, ok := m.componentHealth[name]; ok {
		return h
	}
	return eventprocessor.ComponentHealth{
		Name:    name,
		Healthy: false,
		Error:   "component not found",
	}
}

func TestHealthNATS_NoChecker(t *testing.T) {
	// Reset the global checker
	oldChecker := natsHealthChecker
	natsHealthChecker = nil
	defer func() { natsHealthChecker = oldChecker }()

	handler := &Handler{}

	req := httptest.NewRequest(http.MethodGet, "/health/nats", nil)
	rr := httptest.NewRecorder()

	handler.HealthNATS(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	// Should indicate NATS is not enabled
	body := rr.Body.String()
	if body == "" {
		t.Error("expected non-empty response body")
	}
}

func TestHealthNATS_Healthy(t *testing.T) {
	// Setup mock checker
	oldChecker := natsHealthChecker
	mockChecker := &mockNATSHealthChecker{
		overallHealth: eventprocessor.OverallHealth{
			Healthy: true,
			Status:  eventprocessor.HealthStatusHealthy,
			Components: map[string]eventprocessor.ComponentHealth{
				"consumer": {Healthy: true, Name: "consumer"},
				"dlq":      {Healthy: true, Name: "dlq"},
			},
		},
	}
	natsHealthChecker = mockChecker
	defer func() { natsHealthChecker = oldChecker }()

	handler := &Handler{}

	req := httptest.NewRequest(http.MethodGet, "/health/nats", nil)
	rr := httptest.NewRecorder()

	handler.HealthNATS(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
}

func TestHealthNATS_Unhealthy(t *testing.T) {
	// Setup mock checker with unhealthy status
	oldChecker := natsHealthChecker
	mockChecker := &mockNATSHealthChecker{
		overallHealth: eventprocessor.OverallHealth{
			Healthy: false,
			Status:  eventprocessor.HealthStatusUnhealthy,
			Components: map[string]eventprocessor.ComponentHealth{
				"consumer": {Healthy: false, Name: "consumer", Error: "connection lost"},
			},
		},
	}
	natsHealthChecker = mockChecker
	defer func() { natsHealthChecker = oldChecker }()

	handler := &Handler{}

	req := httptest.NewRequest(http.MethodGet, "/health/nats", nil)
	rr := httptest.NewRecorder()

	handler.HealthNATS(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status %d, got %d", http.StatusServiceUnavailable, rr.Code)
	}
}

func TestHealthNATS_MethodNotAllowed(t *testing.T) {
	handler := &Handler{}

	req := httptest.NewRequest(http.MethodPost, "/health/nats", nil)
	rr := httptest.NewRecorder()

	handler.HealthNATS(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, rr.Code)
	}
}

func TestHealthNATSComponent_NoChecker(t *testing.T) {
	// Reset the global checker
	oldChecker := natsHealthChecker
	natsHealthChecker = nil
	defer func() { natsHealthChecker = oldChecker }()

	handler := &Handler{}

	req := httptest.NewRequest(http.MethodGet, "/health/nats/component?component=consumer", nil)
	rr := httptest.NewRecorder()

	handler.HealthNATSComponent(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
}

func TestHealthNATSComponent_MissingParam(t *testing.T) {
	handler := &Handler{}

	req := httptest.NewRequest(http.MethodGet, "/health/nats/component", nil)
	rr := httptest.NewRecorder()

	handler.HealthNATSComponent(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestHealthNATSComponent_Found(t *testing.T) {
	// Setup mock checker
	oldChecker := natsHealthChecker
	mockChecker := &mockNATSHealthChecker{
		componentHealth: map[string]eventprocessor.ComponentHealth{
			"consumer": {Healthy: true, Name: "consumer", Message: "running"},
		},
	}
	natsHealthChecker = mockChecker
	defer func() { natsHealthChecker = oldChecker }()

	handler := &Handler{}

	req := httptest.NewRequest(http.MethodGet, "/health/nats/component?component=consumer", nil)
	rr := httptest.NewRecorder()

	handler.HealthNATSComponent(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
}

func TestHealthNATSComponent_NotFound(t *testing.T) {
	// Setup mock checker
	oldChecker := natsHealthChecker
	mockChecker := &mockNATSHealthChecker{
		componentHealth: map[string]eventprocessor.ComponentHealth{},
	}
	natsHealthChecker = mockChecker
	defer func() { natsHealthChecker = oldChecker }()

	handler := &Handler{}

	req := httptest.NewRequest(http.MethodGet, "/health/nats/component?component=unknown", nil)
	rr := httptest.NewRecorder()

	handler.HealthNATSComponent(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status %d, got %d", http.StatusServiceUnavailable, rr.Code)
	}
}

func TestIsNATSEnabled(t *testing.T) {
	oldChecker := natsHealthChecker

	// When nil
	natsHealthChecker = nil
	if IsNATSEnabled() {
		t.Error("expected IsNATSEnabled to return false when checker is nil")
	}

	// When set
	natsHealthChecker = &mockNATSHealthChecker{}
	if !IsNATSEnabled() {
		t.Error("expected IsNATSEnabled to return true when checker is set")
	}

	natsHealthChecker = oldChecker
}

func TestSetNATSHealthChecker(t *testing.T) {
	oldChecker := natsHealthChecker
	defer func() { natsHealthChecker = oldChecker }()

	mockChecker := &mockNATSHealthChecker{}
	SetNATSHealthChecker(mockChecker)

	if natsHealthChecker != mockChecker {
		t.Error("expected SetNATSHealthChecker to set the checker")
	}
}
