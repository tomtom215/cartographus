// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package api

import (
	"context"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/tomtom215/cartographus/internal/auth"
	"github.com/tomtom215/cartographus/internal/cache"
	"github.com/tomtom215/cartographus/internal/config"
	"github.com/tomtom215/cartographus/internal/database"
	"github.com/tomtom215/cartographus/internal/logging"
	"github.com/tomtom215/cartographus/internal/middleware"
	syncpkg "github.com/tomtom215/cartographus/internal/sync"
	ws "github.com/tomtom215/cartographus/internal/websocket"
)

// Handler contains dependencies for API handlers
//
// Handler methods are now split across multiple files:
//   - handlers.go: Handler struct, constructor, utility methods (this file)
//   - handlers_helpers.go: Shared helper functions (11 functions)
//   - handlers_health.go: Health/monitoring endpoints (5 methods)
//   - handlers_core.go: Core API endpoints (10 methods)
//   - handlers_analytics.go: Analytics endpoints (11 methods)
//   - handlers_tautulli.go: Tautulli API proxy endpoints (54 methods)
//   - handlers_spatial.go: Spatial analytics & export endpoints (11 methods)
//   - handlers_backup.go: Backup and restore endpoints (17 methods)
type Handler struct {
	db              *database.DB
	sync            *syncpkg.Manager
	client          syncpkg.TautulliClientInterface
	config          *config.Config
	jwtManager      *auth.JWTManager
	plexOAuthClient *auth.PlexOAuthClient // OAuth 2.0 PKCE client for Plex authentication
	wsHub           *ws.Hub
	startTime       time.Time
	cache           *cache.Cache
	perfMon         *middleware.PerformanceMonitor
	backupManager   BackupManager  // Backup manager for backup/restore operations (optional)
	eventPublisher  EventPublisher // NATS event publisher for webhook events (optional)
}

// NewHandler creates a new API handler with all required dependencies.
//
// The handler manages HTTP request processing for 100+ API endpoints across
// five categories: Core, Analytics, Spatial, Tautulli Proxy, and WebSocket.
//
// Dependencies:
//   - db: Database connection for data access
//   - syncMgr: Synchronization manager for manual sync triggers
//   - client: Tautulli API client for proxy endpoints
//   - cfg: Application configuration
//   - jwtManager: JWT token manager for authentication
//   - wsHub: WebSocket hub for real-time broadcasts
//
// The handler initializes with:
//   - 5-minute TTL cache for analytics endpoints
//   - Performance monitor tracking last 1000 requests
//   - Start time for uptime calculations
//
// Example:
//
//	handler := api.NewHandler(db, syncMgr, client, cfg, jwtManager, wsHub)
//	router := api.NewRouter(handler, middleware)
//	http.ListenAndServe(":3857", router.SetupChi()) // ADR-0016: Chi router
func NewHandler(db *database.DB, syncMgr *syncpkg.Manager, client syncpkg.TautulliClientInterface, cfg *config.Config, jwtManager *auth.JWTManager, wsHub *ws.Hub) *Handler {
	// Initialize Plex OAuth client if configured
	var plexOAuthClient *auth.PlexOAuthClient
	if cfg.Plex.OAuthClientID != "" && cfg.Plex.OAuthRedirectURI != "" {
		plexOAuthClient = auth.NewPlexOAuthClient(
			cfg.Plex.OAuthClientID,
			cfg.Plex.OAuthClientSecret,
			cfg.Plex.OAuthRedirectURI,
		)
	}

	return &Handler{
		db:              db,
		sync:            syncMgr,
		client:          client,
		config:          cfg,
		jwtManager:      jwtManager,
		plexOAuthClient: plexOAuthClient,
		wsHub:           wsHub,
		startTime:       time.Now(),
		cache:           cache.New(5 * time.Minute),             // 5 minute TTL for analytics cache
		perfMon:         middleware.NewPerformanceMonitor(1000), // Keep last 1000 requests
	}
}

// ClearCache invalidates all cached analytics data.
//
// This method is called automatically after each successful sync to ensure
// clients receive fresh data. It can also be called manually to force cache
// invalidation without waiting for a sync.
//
// The cache stores analytics query results with a 5-minute TTL. Clearing it
// ensures the next request will query the database directly.
//
// Thread Safety: Safe for concurrent access.
func (h *Handler) ClearCache() {
	if h.cache != nil {
		h.cache.Clear()
		logging.Info().Msg("Analytics cache cleared")
	}
}

// SetBackupManager sets the backup manager for backup/restore operations.
//
// This method allows late initialization of the backup manager after the handler
// is created, which is useful when the backup manager requires the database to
// be fully initialized first.
//
// Thread Safety: Safe for concurrent access but should be called once during startup.
func (h *Handler) SetBackupManager(bm BackupManager) {
	h.backupManager = bm
}

// OnSyncCompleted is the callback invoked after each successful sync operation.
//
// This method handles post-sync tasks:
//  1. Clears the analytics cache to serve fresh data
//  2. Broadcasts sync completion to WebSocket clients
//  3. Fetches and broadcasts updated statistics
//
// Parameters:
//   - newRecords: Number of playback events added during sync
//   - durationMs: Sync operation duration in milliseconds
//
// WebSocket clients receive two messages:
//   - sync_completed: With newRecords and durationMs
//   - stats_update: With current total playbacks and last playback time
//
// The callback is registered via syncManager.SetOnSyncCompleted() during startup.
//
// Thread Safety: Safe for concurrent access.
func (h *Handler) OnSyncCompleted(newRecords int, durationMs int64) {
	// Clear analytics cache
	h.ClearCache()

	// Broadcast sync_completed message to all WebSocket clients
	if h.wsHub != nil {
		h.wsHub.BroadcastSyncCompleted(newRecords, durationMs)

		// Also broadcast stats update with current total count (if db is available)
		if h.db == nil {
			return
		}
		stats, err := h.db.GetStats(context.Background())
		if err == nil {
			// Get last playback time
			lastPlaybackTime, err := h.db.GetLastPlaybackTime(context.Background())
			lastPlayback := ""
			if err == nil && lastPlaybackTime != nil {
				lastPlayback = lastPlaybackTime.Format(time.RFC3339)
			}
			h.wsHub.BroadcastStatsUpdate(stats.TotalPlaybacks, lastPlayback)
		}
	}
}

// getUpgrader creates a WebSocket upgrader with proper origin checking and timeouts.
// Phase 2.4: Added HandshakeTimeout for protection against slow client attacks.
func (h *Handler) getUpgrader() websocket.Upgrader {
	return websocket.Upgrader{
		ReadBufferSize:   1024,
		WriteBufferSize:  1024,
		CheckOrigin:      h.checkWebSocketOrigin,
		HandshakeTimeout: 10 * time.Second, // Phase 2.4: Timeout for handshake completion
	}
}

// checkWebSocketOrigin validates WebSocket connection origins
func (h *Handler) checkWebSocketOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")

	// If no origin header, REJECT - legitimate browser WebSockets ALWAYS include Origin
	// Only non-browser clients (curl, scripts, mobile apps) omit Origin header
	// Allowing empty Origin bypasses CORS entirely - security vulnerability
	if origin == "" {
		logging.Warn().Msg("WebSocket connection rejected: missing Origin header")
		return false
	}

	// If config is nil, allow by default (fail open for tests/development)
	if h.config == nil {
		return true
	}

	// Check against allowed origins from config
	for _, allowedOrigin := range h.config.Security.CORSOrigins {
		if allowedOrigin == "*" || allowedOrigin == origin {
			return true
		}
	}

	// Origin not in allowed list - sanitize origin to prevent log injection
	logging.Warn().Str("origin", sanitizeLogValue(origin)).Msg("WebSocket connection rejected from unauthorized origin")
	return false
}
