// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build nats

package api

import (
	"context"
	"net/http"
	"time"

	"github.com/tomtom215/cartographus/internal/eventprocessor"
	"github.com/tomtom215/cartographus/internal/models"
)

// NATSHealthChecker defines the interface for NATS health checking.
type NATSHealthChecker interface {
	CheckAll(ctx context.Context) eventprocessor.OverallHealth
	CheckComponent(ctx context.Context, name string) eventprocessor.ComponentHealth
}

// natsHealthChecker holds the optional NATS health checker.
// This is set via SetNATSHealthChecker when NATS is enabled.
var natsHealthChecker NATSHealthChecker

// SetNATSHealthChecker sets the NATS health checker for health endpoints.
// This is called during application initialization when NATS is enabled.
func SetNATSHealthChecker(checker NATSHealthChecker) {
	natsHealthChecker = checker
}

// HealthNATS handles NATS-specific health check requests.
//
// @Summary Get NATS health status
// @Description Returns comprehensive health status of all NATS components including consumer, publisher, DLQ handler, and appender
// @Tags Core
// @Accept json
// @Produce json
// @Success 200 {object} models.APIResponse{data=eventprocessor.OverallHealth} "NATS health status"
// @Failure 503 {object} models.APIResponse "NATS is unhealthy"
// @Router /health/nats [get]
func (h *Handler) HealthNATS(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	if natsHealthChecker == nil {
		respondJSON(w, http.StatusOK, &models.APIResponse{
			Status: "success",
			Data: map[string]interface{}{
				"enabled": false,
				"message": "NATS is not enabled",
			},
			Metadata: models.Metadata{
				Timestamp: time.Now(),
			},
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	health := natsHealthChecker.CheckAll(ctx)

	statusCode := http.StatusOK
	if !health.Healthy {
		statusCode = http.StatusServiceUnavailable
	}

	respondJSON(w, statusCode, &models.APIResponse{
		Status: "success",
		Data:   health,
		Metadata: models.Metadata{
			Timestamp: time.Now(),
		},
	})
}

// HealthNATSComponent handles health check for a specific NATS component.
//
// @Summary Get specific NATS component health
// @Description Returns health status of a specific NATS component (consumer, publisher, dlq, appender)
// @Tags Core
// @Accept json
// @Produce json
// @Param component query string true "Component name (consumer, publisher, dlq, appender)"
// @Success 200 {object} models.APIResponse{data=eventprocessor.ComponentHealth} "Component health status"
// @Failure 400 {object} models.APIResponse "Missing component parameter"
// @Router /health/nats/component [get]
func (h *Handler) HealthNATSComponent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	component := r.URL.Query().Get("component")
	if component == "" {
		respondJSON(w, http.StatusBadRequest, &models.APIResponse{
			Status: "error",
			Error: &models.APIError{
				Code:    "MISSING_PARAMETER",
				Message: "component parameter is required",
			},
			Metadata: models.Metadata{
				Timestamp: time.Now(),
			},
		})
		return
	}

	if natsHealthChecker == nil {
		respondJSON(w, http.StatusOK, &models.APIResponse{
			Status: "success",
			Data: eventprocessor.ComponentHealth{
				Name:      component,
				Healthy:   false,
				Error:     "NATS is not enabled",
				LastCheck: time.Now(),
			},
			Metadata: models.Metadata{
				Timestamp: time.Now(),
			},
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	health := natsHealthChecker.CheckComponent(ctx, component)

	statusCode := http.StatusOK
	if !health.Healthy {
		statusCode = http.StatusServiceUnavailable
	}

	respondJSON(w, statusCode, &models.APIResponse{
		Status: "success",
		Data:   health,
		Metadata: models.Metadata{
			Timestamp: time.Now(),
		},
	})
}

// GetNATSHealth returns NATS health for inclusion in main health endpoint.
// Returns nil if NATS is not enabled.
func GetNATSHealth(ctx context.Context) *eventprocessor.OverallHealth {
	if natsHealthChecker == nil {
		return nil
	}
	health := natsHealthChecker.CheckAll(ctx)
	return &health
}

// IsNATSEnabled returns true if NATS health checking is enabled.
func IsNATSEnabled() bool {
	return natsHealthChecker != nil
}
