// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package api

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"github.com/tomtom215/cartographus/internal/models"
)

// ServerStatus returns the status of all configured media servers.
// This is a read-only view of servers configured via environment variables.
//
// @Summary Get status of all configured media servers
// @Description Returns status information for all media servers configured via environment variables.
// @Description Includes connection status, last sync time, and configuration details.
// @Tags Admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} models.APIResponse{data=models.MediaServerListResponse} "Server status retrieved successfully"
// @Failure 401 {object} models.APIResponse "Unauthorized"
// @Failure 403 {object} models.APIResponse "Forbidden - admin only"
// @Router /admin/servers [get]
func (h *Handler) ServerStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	ctx := r.Context()
	servers := make([]models.MediaServerStatus, 0)

	// Handle nil config gracefully
	if h.config == nil {
		respondJSON(w, http.StatusOK, &models.APIResponse{
			Status: "success",
			Data: models.MediaServerListResponse{
				Servers:     servers,
				TotalCount:  0,
				LastChecked: time.Now(),
			},
			Metadata: models.Metadata{
				Timestamp: time.Now(),
			},
		})
		return
	}

	// Collect Tautulli servers
	if h.config.Tautulli.Enabled {
		status := h.getTautulliStatus(ctx)
		servers = append(servers, status)
	}

	// Collect Plex servers - use index to avoid copying large struct
	plexServers := h.config.GetPlexServers()
	for i := range plexServers {
		status := h.getPlexStatus(plexServers[i])
		servers = append(servers, status)
	}

	// Collect Jellyfin servers - use index to avoid copying large struct
	jellyfinServers := h.config.GetJellyfinServers()
	for i := range jellyfinServers {
		status := h.getJellyfinStatus(jellyfinServers[i])
		servers = append(servers, status)
	}

	// Collect Emby servers - use index to avoid copying large struct
	embyServers := h.config.GetEmbyServers()
	for i := range embyServers {
		status := h.getEmbyStatus(embyServers[i])
		servers = append(servers, status)
	}

	// Count statuses - use index to avoid copying struct
	connected := 0
	syncing := 0
	errorCount := 0
	for i := range servers {
		switch servers[i].Status {
		case "connected":
			connected++
		case "syncing":
			syncing++
		case "error":
			errorCount++
		}
	}

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data: models.MediaServerListResponse{
			Servers:     servers,
			TotalCount:  len(servers),
			Connected:   connected,
			Syncing:     syncing,
			Error:       errorCount,
			LastChecked: time.Now(),
		},
		Metadata: models.Metadata{
			Timestamp: time.Now(),
		},
	})
}

// getTautulliStatus returns the status for the Tautulli server.
func (h *Handler) getTautulliStatus(ctx context.Context) models.MediaServerStatus {
	serverID := h.config.Tautulli.ServerID
	if serverID == "" {
		serverID = "tautulli-default"
	}

	status := models.MediaServerStatus{
		ID:              serverID,
		Platform:        "tautulli",
		Name:            "Tautulli",
		URL:             maskURLForDisplay(h.config.Tautulli.URL),
		Enabled:         h.config.Tautulli.Enabled,
		Source:          "env",
		Status:          "disconnected",
		RealtimeEnabled: false,
		WebhooksEnabled: false,
		SessionPolling:  false,
		Immutable:       true,
	}

	// Check connectivity
	if h.client != nil {
		if err := h.client.Ping(ctx); err != nil {
			status.Status = "error"
			status.LastError = "Connection failed"
			now := time.Now()
			status.LastErrorAt = &now
		} else {
			status.Status = "connected"
		}
	}

	// Get last sync time from sync manager
	if h.sync != nil {
		lastSync := h.sync.LastSyncTime()
		if !lastSync.IsZero() {
			status.LastSyncAt = &lastSync
			status.LastSyncStatus = "completed"
		}
	}

	return status
}

// getPlexStatus returns the status for a Plex server.
func (h *Handler) getPlexStatus(cfg interface{}) models.MediaServerStatus {
	// Use type assertion to get Plex config
	// Since we receive interface{} from GetPlexServers, we need to extract fields
	plexCfg, ok := cfg.(struct {
		URL                 string
		Token               string
		ServerID            string
		RealtimeEnabled     bool
		WebhooksEnabled     bool
		SyncInterval        time.Duration
		SyncDaysBack        int
		HistoricalSync      bool
		TranscodeMonitor    bool
		BufferHealthMonitor bool
	})

	if !ok {
		// Fallback: try to use reflection or return minimal info
		return models.MediaServerStatus{
			ID:        "plex-unknown",
			Platform:  "plex",
			Name:      "Plex Server",
			Status:    "unknown",
			Source:    "env",
			Immutable: true,
		}
	}

	serverID := plexCfg.ServerID
	if serverID == "" {
		serverID = "plex-default"
	}

	status := models.MediaServerStatus{
		ID:              serverID,
		Platform:        "plex",
		Name:            "Plex Server",
		URL:             maskURLForDisplay(plexCfg.URL),
		Enabled:         true,
		Source:          "env",
		Status:          "disconnected",
		RealtimeEnabled: plexCfg.RealtimeEnabled,
		WebhooksEnabled: plexCfg.WebhooksEnabled,
		SessionPolling:  false,
		Immutable:       true,
	}

	// We can't easily check Plex connectivity here without the Plex client
	// Mark as configured but unknown status
	if plexCfg.URL != "" && plexCfg.Token != "" {
		status.Status = "configured"
	}

	return status
}

// getJellyfinStatus returns the status for a Jellyfin server.
func (h *Handler) getJellyfinStatus(cfg interface{}) models.MediaServerStatus {
	// Similar pattern to Plex
	jellyfinCfg, ok := cfg.(struct {
		URL                    string
		APIKey                 string
		ServerID               string
		UserID                 string
		RealtimeEnabled        bool
		WebhooksEnabled        bool
		SessionPollingEnabled  bool
		SessionPollingInterval time.Duration
	})

	if !ok {
		return models.MediaServerStatus{
			ID:        "jellyfin-unknown",
			Platform:  "jellyfin",
			Name:      "Jellyfin Server",
			Status:    "unknown",
			Source:    "env",
			Immutable: true,
		}
	}

	serverID := jellyfinCfg.ServerID
	if serverID == "" {
		serverID = "jellyfin-default"
	}

	status := models.MediaServerStatus{
		ID:              serverID,
		Platform:        "jellyfin",
		Name:            "Jellyfin Server",
		URL:             maskURLForDisplay(jellyfinCfg.URL),
		Enabled:         true,
		Source:          "env",
		Status:          "disconnected",
		RealtimeEnabled: jellyfinCfg.RealtimeEnabled,
		WebhooksEnabled: jellyfinCfg.WebhooksEnabled,
		SessionPolling:  jellyfinCfg.SessionPollingEnabled,
		Immutable:       true,
	}

	if jellyfinCfg.URL != "" && jellyfinCfg.APIKey != "" {
		status.Status = "configured"
	}

	return status
}

// getEmbyStatus returns the status for an Emby server.
func (h *Handler) getEmbyStatus(cfg interface{}) models.MediaServerStatus {
	embyCfg, ok := cfg.(struct {
		URL                    string
		APIKey                 string
		ServerID               string
		UserID                 string
		RealtimeEnabled        bool
		WebhooksEnabled        bool
		SessionPollingEnabled  bool
		SessionPollingInterval time.Duration
	})

	if !ok {
		return models.MediaServerStatus{
			ID:        "emby-unknown",
			Platform:  "emby",
			Name:      "Emby Server",
			Status:    "unknown",
			Source:    "env",
			Immutable: true,
		}
	}

	serverID := embyCfg.ServerID
	if serverID == "" {
		serverID = "emby-default"
	}

	status := models.MediaServerStatus{
		ID:              serverID,
		Platform:        "emby",
		Name:            "Emby Server",
		URL:             maskURLForDisplay(embyCfg.URL),
		Enabled:         true,
		Source:          "env",
		Status:          "disconnected",
		RealtimeEnabled: embyCfg.RealtimeEnabled,
		WebhooksEnabled: embyCfg.WebhooksEnabled,
		SessionPolling:  embyCfg.SessionPollingEnabled,
		Immutable:       true,
	}

	if embyCfg.URL != "" && embyCfg.APIKey != "" {
		status.Status = "configured"
	}

	return status
}

// maskURLForDisplay returns a masked version of a URL safe for display.
// Shows protocol, host, and port but hides paths and query params.
func maskURLForDisplay(rawURL string) string {
	if rawURL == "" {
		return ""
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		// Can't parse, just truncate
		if len(rawURL) > 50 {
			return rawURL[:50] + "..."
		}
		return rawURL
	}

	// Return just scheme and host (includes port if present)
	return parsed.Scheme + "://" + parsed.Host
}
