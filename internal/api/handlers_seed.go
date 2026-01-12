// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package api

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/tomtom215/cartographus/internal/logging"
	"github.com/tomtom215/cartographus/internal/models"
)

// SeedMockData populates the database with mock data for screenshots and demos.
// This endpoint is only available in development/CI environments (non-production).
//
// @Summary Seed database with mock data
// @Description Seeds the database with realistic mock data for screenshots and demos
// @Tags admin
// @Produce json
// @Success 200 {object} models.APIResponse "Seeding successful"
// @Failure 403 {object} models.APIResponse "Forbidden in production environment"
// @Failure 500 {object} models.APIResponse "Seeding failed"
// @Router /api/v1/admin/seed [post]
func (h *Handler) SeedMockData(w http.ResponseWriter, r *http.Request) {
	// Security: Only allow seeding in non-production environments
	// Check multiple indicators for production environment
	env := os.Getenv("GO_ENV")
	nodeEnv := os.Getenv("NODE_ENV")
	authMode := h.config.Security.AuthMode

	// Explicitly check if running in CI or development
	isCI := os.Getenv("CI") == "true"
	isDev := env == "development" || env == "dev" || env == ""
	isTest := env == "test" || os.Getenv("TESTING") == "true"

	// Block if explicitly production or if auth mode suggests production without CI
	isProduction := env == "production" || nodeEnv == "production"
	if isProduction && !isCI {
		logging.Warn().
			Str("go_env", env).
			Str("node_env", nodeEnv).
			Str("auth_mode", authMode).
			Msg("Blocked seed attempt in production environment")
		respondError(w, http.StatusForbidden, "SEED_FORBIDDEN", "Seeding is not allowed in production environments", nil)
		return
	}

	// Additional safety: require explicit opt-in via environment variable or CI
	allowSeed := os.Getenv("ALLOW_SEED_DATA") == "true"
	if !isCI && !isDev && !isTest && !allowSeed {
		logging.Warn().
			Str("go_env", env).
			Bool("is_ci", isCI).
			Bool("allow_seed", allowSeed).
			Msg("Seed endpoint requires explicit opt-in outside CI/development")
		respondError(w, http.StatusForbidden, "SEED_NOT_ALLOWED", "Seeding requires ALLOW_SEED_DATA=true or CI environment", nil)
		return
	}

	logging.Info().
		Bool("is_ci", isCI).
		Bool("is_dev", isDev).
		Str("auth_mode", authMode).
		Msg("Starting mock data seeding")

	// Set a reasonable timeout for seeding
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()

	// Call the database seed method
	if err := h.db.SeedMockData(ctx); err != nil {
		logging.Error().Err(err).Msg("Failed to seed mock data")
		respondError(w, http.StatusInternalServerError, "SEED_FAILED", "Failed to seed mock data: "+err.Error(), err)
		return
	}

	// Clear analytics cache after seeding to ensure fresh data is served
	h.cache.Clear()

	logging.Info().Msg("Mock data seeded successfully")

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data: map[string]interface{}{
			"message": "Mock data seeded successfully",
			"counts": map[string]interface{}{
				"users":     15,
				"locations": 50,
				"playbacks": 250,
				"days":      30,
			},
		},
		Metadata: models.Metadata{
			Timestamp: time.Now(),
		},
	})
}
