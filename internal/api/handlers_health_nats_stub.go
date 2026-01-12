// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build !nats

package api

import (
	"context"
	"net/http"
	"time"

	"github.com/tomtom215/cartographus/internal/models"
)

// HealthNATS handles NATS-specific health check requests.
// When NATS is not enabled, returns a message indicating NATS is disabled.
func (h *Handler) HealthNATS(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data: map[string]interface{}{
			"enabled": false,
			"message": "NATS support not compiled in. Build with -tags nats to enable.",
		},
		Metadata: models.Metadata{
			Timestamp: time.Now(),
		},
	})
}

// HealthNATSComponent handles health check for a specific NATS component.
// When NATS is not enabled, returns a message indicating NATS is disabled.
func (h *Handler) HealthNATSComponent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data: map[string]interface{}{
			"enabled": false,
			"message": "NATS support not compiled in. Build with -tags nats to enable.",
		},
		Metadata: models.Metadata{
			Timestamp: time.Now(),
		},
	})
}

// GetNATSHealth returns nil when NATS is not enabled.
func GetNATSHealth(ctx context.Context) interface{} {
	return nil
}

// IsNATSEnabled returns false when NATS is not compiled in.
func IsNATSEnabled() bool {
	return false
}
