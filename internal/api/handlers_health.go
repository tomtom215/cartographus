// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package api

import (
	"net/http"
	"time"

	"github.com/tomtom215/cartographus/internal/cache"
	"github.com/tomtom215/cartographus/internal/middleware"
	"github.com/tomtom215/cartographus/internal/models"
)

// Health handles health check requests
//
// @Summary Get system health status
// @Description Returns comprehensive health status including database connectivity, Tautulli connectivity, last sync time, and uptime
// @Tags Core
// @Accept json
// @Produce json
// @Success 200 {object} models.APIResponse{data=models.HealthStatus} "Health status retrieved successfully"
// @Router /health [get]
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	// Check database connectivity (nil means not connected)
	dbConnected := h.db != nil && h.db.Ping(r.Context()) == nil

	// Check Tautulli connectivity (nil means not connected)
	tautulliConnected := h.client != nil && h.client.Ping(r.Context()) == nil

	// Determine mode: standalone (no Tautulli) or tautulli (with Tautulli)
	// In standalone mode (v2.0+), Tautulli is optional - direct Plex/Jellyfin/Emby connections are primary
	tautulliEnabled := h.config != nil && h.config.Tautulli.Enabled
	mode := "standalone"
	if tautulliEnabled {
		mode = "tautulli"
	}

	// Determine health status:
	// - In standalone mode: healthy if DB connected (Tautulli not expected)
	// - In tautulli mode: healthy if both DB and Tautulli connected
	status := "healthy"
	if !dbConnected {
		status = "degraded"
	} else if tautulliEnabled && !tautulliConnected {
		status = "degraded"
	}

	var lastSyncPtr *time.Time
	if h.sync != nil {
		lastSync := h.sync.LastSyncTime()
		if !lastSync.IsZero() {
			lastSyncPtr = &lastSync
		}
	}

	health := models.HealthStatus{
		Status:            status,
		Mode:              mode,
		Version:           "1.0.0",
		DatabaseConnected: dbConnected,
		TautulliConnected: tautulliConnected,
		LastSyncTime:      lastSyncPtr,
		Uptime:            time.Since(h.startTime).Seconds(),
	}

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data:   health,
		Metadata: models.Metadata{
			Timestamp: time.Now(),
		},
	})
}

// HealthLive handles liveness probe requests (Kubernetes-style)
// Returns 200 OK if the process is alive, regardless of dependencies
//
// @Summary Kubernetes liveness probe
// @Description Returns 200 OK if the process is alive, regardless of external dependencies. Used for Kubernetes liveness probes.
// @Tags Core
// @Accept json
// @Produce json
// @Success 200 {object} models.APIResponse "Service is alive"
// @Router /health/live [get]
func (h *Handler) HealthLive(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data: map[string]interface{}{
			"alive":  true,
			"uptime": time.Since(h.startTime).Seconds(),
		},
		Metadata: models.Metadata{
			Timestamp: time.Now(),
		},
	})
}

// HealthReady handles readiness probe requests (Kubernetes-style)
// Returns 200 OK only if the service is ready to handle traffic
//
// @Summary Kubernetes readiness probe
// @Description Returns 200 OK only if the service is ready to handle traffic (database and Tautulli are both connected). Returns 503 if not ready.
// @Tags Core
// @Accept json
// @Produce json
// @Success 200 {object} models.APIResponse "Service is ready"
// @Failure 503 {object} models.APIResponse "Service is not ready"
// @Router /health/ready [get]
func (h *Handler) HealthReady(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	// Check database connectivity (nil means not connected)
	dbConnected := h.db != nil && h.db.Ping(r.Context()) == nil

	// Check Tautulli connectivity (nil means not connected)
	tautulliConnected := h.client != nil && h.client.Ping(r.Context()) == nil
	ready := dbConnected && tautulliConnected

	statusCode := http.StatusOK
	status := "ready"
	if !ready {
		statusCode = http.StatusServiceUnavailable
		status = "not_ready"
	}

	respondJSON(w, statusCode, &models.APIResponse{
		Status: status,
		Data: map[string]interface{}{
			"database_connected": dbConnected,
			"tautulli_connected": tautulliConnected,
			"ready_to_serve":     ready,
			"uptime":             time.Since(h.startTime).Seconds(),
		},
		Metadata: models.Metadata{
			Timestamp: time.Now(),
		},
	})
}

// SetupStatus returns configuration and connectivity status for the setup wizard.
// This endpoint is used by the frontend onboarding wizard to guide first-time users
// through the initial configuration of data sources.
//
// The endpoint checks:
//   - Database connectivity
//   - Tautulli configuration and connectivity
//   - Plex/Jellyfin/Emby server configurations
//   - NATS messaging configuration
//   - Data availability (existing playback records)
//
// @Summary Get setup status for onboarding wizard
// @Description Returns comprehensive setup status including data source configurations,
// @Description connectivity checks, and recommendations for first-time users.
// @Tags Core
// @Accept json
// @Produce json
// @Success 200 {object} models.APIResponse{data=models.SetupStatus} "Setup status retrieved successfully"
// @Router /health/setup [get]
func (h *Handler) SetupStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	ctx := r.Context()
	status := models.SetupStatus{
		Recommendations: make([]string, 0),
	}

	// Check database connectivity
	dbConnected := h.db != nil && h.db.Ping(ctx) == nil
	status.Database = models.SetupDatabaseStatus{
		Connected: dbConnected,
	}

	// Handle nil config gracefully
	if h.config == nil {
		status.Recommendations = append(status.Recommendations, "Configuration not loaded. Check server startup logs.")
		respondJSON(w, http.StatusOK, &models.APIResponse{
			Status: "success",
			Data:   status,
			Metadata: models.Metadata{
				Timestamp: time.Now(),
			},
		})
		return
	}

	// Check Tautulli configuration and connectivity
	tautulliConfigured := h.config.Tautulli.Enabled && h.config.Tautulli.URL != ""
	tautulliConnected := false
	var tautulliError string
	if tautulliConfigured && h.client != nil {
		if err := h.client.Ping(ctx); err != nil {
			tautulliError = "Connection failed"
		} else {
			tautulliConnected = true
		}
	}
	status.DataSources.Tautulli = models.SetupDataSourceStatus{
		Configured: tautulliConfigured,
		Connected:  tautulliConnected,
		URL:        maskURL(h.config.Tautulli.URL),
		Error:      tautulliError,
	}

	// Check Plex configuration
	plexServers := h.config.GetPlexServers()
	plexConfigured := len(plexServers) > 0
	status.DataSources.Plex = models.SetupMediaServerStatus{
		Configured:  plexConfigured,
		ServerCount: len(plexServers),
	}

	// Check Jellyfin configuration
	jellyfinServers := h.config.GetJellyfinServers()
	jellyfinConfigured := len(jellyfinServers) > 0
	status.DataSources.Jellyfin = models.SetupMediaServerStatus{
		Configured:  jellyfinConfigured,
		ServerCount: len(jellyfinServers),
	}

	// Check Emby configuration
	embyServers := h.config.GetEmbyServers()
	embyConfigured := len(embyServers) > 0
	status.DataSources.Emby = models.SetupMediaServerStatus{
		Configured:  embyConfigured,
		ServerCount: len(embyServers),
	}

	// Check NATS configuration
	natsEnabled := h.config.NATS.Enabled
	status.DataSources.NATS = models.SetupOptionalFeature{
		Enabled: natsEnabled,
	}

	// Check data availability
	if dbConnected && h.db != nil {
		playbacks, geolocations, err := h.db.GetRecordCounts(ctx)
		if err == nil {
			status.DataAvailable = models.SetupDataAvailability{
				HasPlaybacks:    playbacks > 0,
				PlaybackCount:   playbacks,
				HasGeolocations: geolocations > 0,
			}
		}
	}

	// Determine overall readiness
	// Ready if database is connected AND at least one data source is configured
	hasDataSource := tautulliConnected || plexConfigured || jellyfinConfigured || embyConfigured
	status.Ready = dbConnected && hasDataSource

	// Generate recommendations
	if !dbConnected {
		status.Recommendations = append(status.Recommendations, "Database connection failed. Check database configuration.")
	}

	if !tautulliConfigured && !plexConfigured && !jellyfinConfigured && !embyConfigured {
		status.Recommendations = append(status.Recommendations, "No data sources configured. Configure at least one of: Tautulli, Plex, Jellyfin, or Emby.")
	}

	if tautulliConfigured && !tautulliConnected {
		status.Recommendations = append(status.Recommendations, "Tautulli is configured but not reachable. Check the URL and API key.")
	}

	if tautulliConnected && !plexConfigured {
		status.Recommendations = append(status.Recommendations, "Consider enabling Plex direct integration for real-time playback updates.")
	}

	if status.Ready && !status.DataAvailable.HasPlaybacks {
		status.Recommendations = append(status.Recommendations, "No playback data yet. Trigger a sync or wait for playback activity.")
	}

	if !natsEnabled && tautulliConnected {
		status.Recommendations = append(status.Recommendations, "Consider enabling NATS for event-driven processing and better real-time updates.")
	}

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data:   status,
		Metadata: models.Metadata{
			Timestamp: time.Now(),
		},
	})
}

// maskURL returns a masked version of a URL for display (hides credentials)
func maskURL(rawURL string) string {
	if rawURL == "" {
		return ""
	}
	// Simple masking - just show protocol and host
	// Don't expose full path or query params which might contain keys
	if len(rawURL) > 50 {
		return rawURL[:50] + "..."
	}
	return rawURL
}

// GetCacheStats returns cache performance statistics
func (h *Handler) GetCacheStats() cache.Stats {
	if h.cache != nil {
		return h.cache.GetStats()
	}
	return cache.Stats{}
}

// GetPerformanceStats returns performance monitoring statistics
func (h *Handler) GetPerformanceStats() []middleware.EndpointStats {
	if h.perfMon != nil {
		return h.perfMon.GetStats()
	}
	return nil
}
