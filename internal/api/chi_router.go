// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package api provides HTTP routing using Chi router (ADR-0016).
package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	httpSwagger "github.com/swaggo/http-swagger/v2"
	"github.com/tomtom215/cartographus/internal/middleware"
)

// chiMiddleware adapts http.HandlerFunc middleware to Chi's func(http.Handler) http.Handler.
// This allows our existing middleware to work with Chi's r.Use().
// Used for SecurityHeaders (CSP nonce), Authenticate (auth logic), and PrometheusMetrics.
func chiMiddleware(mw func(http.HandlerFunc) http.HandlerFunc) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return mw(next.ServeHTTP)
	}
}

// SetupChi configures all HTTP routes using Chi router.
// This replaces the http.ServeMux-based Setup() method.
func (router *Router) SetupChi() http.Handler {
	r := chi.NewRouter()

	// ========================
	// Global Middleware Stack
	// ========================
	// Applied to ALL routes in order
	r.Use(RequestIDWithLogging())      // Add X-Request-ID header with logging context
	r.Use(E2EDebugLogging())           // E2E diagnostic logging (enabled via E2E_DEBUG=true)
	r.Use(chimiddleware.RealIP)        // Extract real IP from X-Forwarded-For
	r.Use(chimiddleware.Recoverer)     // Recover from panics
	r.Use(router.chiMiddleware.CORS()) // CORS must be global to handle OPTIONS preflight

	// ========================
	// Health Endpoints
	// ========================
	// L-02 Security Fix: Permissive rate limiting (1000/min) for health endpoints
	// Allows frequent monitoring while preventing abuse
	r.Route("/api/v1/health", func(r chi.Router) {
		r.Use(router.chiMiddleware.RateLimitHealth())
		r.Use(APISecurityHeaders()) // L-01: Add security headers
		r.Get("/live", router.handler.HealthLive)
		r.Get("/ready", router.handler.HealthReady)
		r.Get("/", router.handler.Health)
		r.Get("/setup", router.handler.SetupStatus) // Setup wizard status for onboarding
		r.Get("/nats", router.handler.HealthNATS)
		r.Get("/nats/component", router.handler.HealthNATSComponent)
	})

	// ========================
	// Authentication Endpoints
	// ========================
	// Phase 3: Strict rate limiting for auth endpoints (brute force prevention)
	r.Route("/api/v1/auth", func(r chi.Router) {
		r.Use(router.chiMiddleware.RateLimitAuth())

		// Login has strictest rate limiting (5 attempts per 5 minutes)
		r.With(router.chiMiddleware.RateLimitLogin()).Post("/login", router.handler.Login)

		// Plex OAuth - standard auth rate limiting
		r.Get("/plex/start", router.handler.PlexOAuthStart)
		r.Get("/plex/callback", router.handler.PlexOAuthCallback)
		r.Post("/plex/refresh", router.handler.PlexOAuthRefresh)
		r.Post("/plex/revoke", router.handler.PlexOAuthRevoke)
	})

	// ========================
	// Core API Endpoints
	// ========================
	// SECURITY FIX: All data endpoints require authentication
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(router.chiMiddleware.RateLimit())
		r.Use(APISecurityHeaders()) // L-01: Add security headers to API endpoints
		r.Use(chiMiddleware(middleware.PrometheusMetrics))
		r.Use(chiMiddleware(router.middleware.Authenticate)) // SECURITY: Require auth for all data endpoints

		r.Get("/stats", router.handler.Stats)
		r.Get("/playbacks", router.handler.Playbacks)
		r.Get("/locations", router.handler.Locations)
		r.Get("/users", router.handler.Users)
		r.Get("/media-types", router.handler.MediaTypes)
		r.Get("/server-info", router.handler.ServerInfo)
		r.Get("/ws", router.handler.WebSocket)
	})

	// ========================
	// Analytics Endpoints
	// ========================
	// Phase 3: Permissive rate limiting for cached analytics (1000/min)
	// Read-only cached endpoints - allow smooth dashboard exploration
	// SECURITY FIX: All analytics endpoints require authentication
	r.Route("/api/v1/analytics", func(r chi.Router) {
		r.Use(router.chiMiddleware.RateLimitAnalytics())
		r.Use(APISecurityHeaders()) // L-01: Add security headers to API endpoints
		r.Use(chiMiddleware(middleware.PrometheusMetrics))
		r.Use(chiMiddleware(router.middleware.Authenticate)) // SECURITY: Require auth for analytics

		r.Get("/trends", router.handler.AnalyticsTrends)
		r.Get("/geographic", router.handler.AnalyticsGeographic)
		r.Get("/users", router.handler.AnalyticsUsers)
		r.Get("/binge", router.handler.AnalyticsBinge)
		r.Get("/bandwidth", router.handler.AnalyticsBandwidth)
		r.Get("/bitrate", router.handler.AnalyticsBitrate)
		r.Get("/popular", router.handler.AnalyticsPopular)
		r.Get("/watch-parties", router.handler.AnalyticsWatchParties)
		r.Get("/user-engagement", router.handler.AnalyticsUserEngagement)
		r.Get("/abandonment", router.handler.AnalyticsAbandonment)
		r.Get("/comparative", router.handler.AnalyticsComparative)
		r.Get("/temporal-heatmap", router.handler.AnalyticsTemporalHeatmap)
		r.Get("/resolution-mismatch", router.handler.AnalyticsResolutionMismatch)
		r.Get("/hdr", router.handler.AnalyticsHDR)
		r.Get("/audio", router.handler.AnalyticsAudio)
		r.Get("/subtitles", router.handler.AnalyticsSubtitles)
		r.Get("/frame-rate", router.handler.AnalyticsFrameRate)
		r.Get("/container", router.handler.AnalyticsContainer)
		r.Get("/connection-security", router.handler.AnalyticsConnectionSecurity)
		r.Get("/pause-patterns", router.handler.AnalyticsPausePatterns)
		r.Get("/concurrent-streams", router.handler.AnalyticsConcurrentStreams)
		r.Get("/library", router.handler.AnalyticsLibrary)
		r.Get("/hardware-transcode", router.handler.AnalyticsHardwareTranscode)
		r.Get("/hardware-transcode/trends", router.handler.AnalyticsHardwareTranscodeTrends)
		r.Get("/hdr-content", router.handler.AnalyticsHDRContent)

		// Enhanced analytics (production-grade insights)
		r.Get("/cohort-retention", router.handler.AnalyticsCohortRetention)   // Cohort retention analysis
		r.Get("/qoe", router.handler.AnalyticsQoE)                            // Quality of Experience dashboard
		r.Get("/data-quality", router.handler.AnalyticsDataQuality)           // Data quality monitoring
		r.Get("/user-network", router.handler.AnalyticsUserNetwork)           // User relationship network
		r.Get("/device-migration", router.handler.AnalyticsDeviceMigration)   // Device/platform migration tracking
		r.Get("/content-discovery", router.handler.AnalyticsContentDiscovery) // Content discovery & time-to-first-watch

		// Advanced chart visualizations (Sankey, Chord, Radar, Treemap)
		r.Get("/content-flow", router.handler.AnalyticsContentFlow)               // Sankey: Show->Season->Episode journeys
		r.Get("/user-overlap", router.handler.AnalyticsUserOverlap)               // Chord: User-user content similarity
		r.Get("/user-profile", router.handler.AnalyticsUserProfile)               // Radar: Multi-dimensional engagement
		r.Get("/library-utilization", router.handler.AnalyticsLibraryUtilization) // Treemap: Hierarchical library usage
		r.Get("/calendar-heatmap", router.handler.AnalyticsCalendarHeatmap)       // Calendar: Daily activity patterns
		r.Get("/bump-chart", router.handler.AnalyticsBumpChart)                   // Bump: Content ranking changes

		// Approximate analytics using DataSketches (HyperLogLog, KLL)
		r.Get("/approximate", router.handler.ApproximateStats)
		r.Get("/approximate/distinct", router.handler.ApproximateDistinctCount)
		r.Get("/approximate/percentile", router.handler.ApproximatePercentile)

		// Cross-platform analytics (Phase 3)
		r.Route("/cross-platform", func(r chi.Router) {
			r.Use(chiPathValue) // Bridge Chi URL params to r.PathValue()
			r.Get("/user/{id}", router.handler.CrossPlatformUserStats)
			r.Get("/content/{id}", router.handler.CrossPlatformContentStats)
			r.Get("/summary", router.handler.CrossPlatformSummary)
		})
	})

	// ========================
	// Search Endpoints
	// ========================
	// SECURITY FIX: Search endpoints require authentication
	r.Route("/api/v1/search", func(r chi.Router) {
		r.Use(router.chiMiddleware.RateLimit())
		r.Use(chiMiddleware(middleware.PrometheusMetrics))
		r.Use(chiMiddleware(router.middleware.Authenticate)) // SECURITY: Require auth for search

		r.Get("/fuzzy", router.handler.FuzzySearch)      // Fuzzy content search
		r.Get("/users", router.handler.FuzzySearchUsers) // Fuzzy user search
	})

	// ========================
	// Spatial Endpoints
	// ========================
	// SECURITY FIX: Spatial endpoints require authentication
	r.Route("/api/v1/spatial", func(r chi.Router) {
		r.Use(router.chiMiddleware.RateLimit())
		r.Use(chiMiddleware(middleware.PrometheusMetrics))
		r.Use(chiMiddleware(router.middleware.Authenticate)) // SECURITY: Require auth for spatial data

		r.Get("/hexagons", router.handler.SpatialHexagons)
		r.Get("/arcs", router.handler.SpatialArcs)
		r.Get("/viewport", router.handler.SpatialViewport)
		r.Get("/temporal-density", router.handler.SpatialTemporalDensity)
		r.Get("/nearby", router.handler.SpatialNearby)
	})

	// ========================
	// Cross-Platform Endpoints (Phase 3)
	// ========================
	// Content mapping and user linking for multi-server analytics
	// SECURITY FIX: Content data requires authentication
	r.Route("/api/v1/content", func(r chi.Router) {
		r.Use(router.chiMiddleware.RateLimit())
		r.Use(chiMiddleware(middleware.PrometheusMetrics))
		r.Use(chiPathValue)                                  // Bridge Chi URL params to r.PathValue()
		r.Use(chiMiddleware(router.middleware.Authenticate)) // SECURITY: Require auth for content data

		// Read operations
		r.Get("/lookup", router.handler.ContentMappingLookup)

		// Write operations
		r.Post("/link", router.handler.ContentMappingCreate)
		r.Post("/{id}/link/plex", router.handler.ContentMappingLinkPlex)
		r.Post("/{id}/link/jellyfin", router.handler.ContentMappingLinkJellyfin)
		r.Post("/{id}/link/emby", router.handler.ContentMappingLinkEmby)
	})

	// SECURITY FIX: User data requires authentication
	r.Route("/api/v1/users", func(r chi.Router) {
		r.Use(router.chiMiddleware.RateLimit())
		r.Use(chiMiddleware(middleware.PrometheusMetrics))
		r.Use(chiPathValue)                                  // Bridge Chi URL params to r.PathValue()
		r.Use(chiMiddleware(router.middleware.Authenticate)) // SECURITY: Require auth for user data

		// Read operations
		r.Get("/", router.handler.Users)
		r.Get("/suggest-links", router.handler.UserSuggestLinks)
		r.Get("/{id}/linked", router.handler.UserLinkedGet)

		// Write operations
		r.Post("/link", router.handler.UserLinkCreate)
		r.Delete("/link", router.handler.UserLinkDelete)
	})

	// ========================
	// Plex Direct Endpoints
	// ========================
	// SECURITY FIX (MEDIUM-001): Plex data requires authentication
	r.Route("/api/v1/plex", func(r chi.Router) {
		r.Use(router.chiMiddleware.RateLimit())
		r.Use(chiPathValue) // Bridge Chi URL params to r.PathValue()

		// Webhook endpoint - uses its own signature verification (HMAC-SHA256)
		// This MUST remain public for Plex to send webhooks
		r.Post("/webhook", router.handler.PlexWebhook)

		// All other Plex endpoints require authentication
		r.Group(func(r chi.Router) {
			r.Use(chiMiddleware(router.middleware.Authenticate)) // SECURITY: Require auth for Plex data

			// Read operations
			r.Get("/statistics/bandwidth", router.handler.PlexBandwidthStatistics)
			r.Get("/sessions", router.handler.PlexSessions)
			r.Get("/identity", router.handler.PlexIdentity)
			r.Get("/devices", router.handler.PlexDevices)
			r.Get("/accounts", router.handler.PlexAccounts)
			r.Get("/activities", router.handler.PlexActivities)
			r.Get("/capabilities", router.handler.PlexServerCapabilities)
			r.Get("/playlists", router.handler.PlexPlaylists)
			r.Get("/transcode/sessions", router.handler.PlexTranscodeSessions)

			// Library routes with path parameters
			r.Get("/library/sections", router.handler.PlexLibrarySections)
			r.Get("/library/sections/{key}/all", router.handler.PlexLibrarySectionContent)
			r.Get("/library/sections/{key}/recentlyAdded", router.handler.PlexLibrarySectionRecentlyAdded)
			r.Get("/library/sections/{key}/search", router.handler.PlexSearch)
			r.Get("/library/metadata/{ratingKey}", router.handler.PlexMetadata)
			r.Get("/library/onDeck", router.handler.PlexOnDeck)

			// Friends and Sharing Management (plex.tv API)
			r.Get("/friends", router.handler.PlexFriendsList)
			r.Post("/friends/invite", router.handler.PlexFriendsInvite)
			r.Delete("/friends/{id}", router.handler.PlexFriendsRemove)

			// Library Sharing
			r.Get("/sharing", router.handler.PlexSharingList)
			r.Post("/sharing", router.handler.PlexSharingCreate)
			r.Put("/sharing/{id}", router.handler.PlexSharingUpdate)
			r.Delete("/sharing/{id}", router.handler.PlexSharingRevoke)

			// Managed Users (Plex Home)
			r.Get("/home/users", router.handler.PlexManagedUsersList)
			r.Post("/home/users", router.handler.PlexManagedUsersCreate)
			r.Put("/home/users/{id}", router.handler.PlexManagedUsersUpdate)
			r.Delete("/home/users/{id}", router.handler.PlexManagedUsersDelete)

			// Library sections for sharing UI
			r.Get("/libraries", router.handler.PlexLibrariesList)

			// Write operations
			r.Delete("/transcode/sessions/{sessionKey}", router.handler.PlexCancelTranscode)
		})
	})

	// ========================
	// Tautulli Proxy Endpoints
	// ========================
	// SECURITY FIX (MEDIUM-001): Tautulli data requires authentication
	r.Route("/api/v1/tautulli", func(r chi.Router) {
		r.Use(router.chiMiddleware.RateLimit())
		r.Use(chiMiddleware(router.middleware.Authenticate)) // SECURITY: Require auth for all Tautulli data

		// Home & Activity
		r.Get("/home-stats", router.handler.TautulliHomeStats)
		r.Get("/activity", router.handler.TautulliActivity)

		// Plays Statistics
		r.Get("/plays-by-date", router.handler.TautulliPlaysByDate)
		r.Get("/plays-by-dayofweek", router.handler.TautulliPlaysByDayOfWeek)
		r.Get("/plays-by-hourofday", router.handler.TautulliPlaysByHourOfDay)
		r.Get("/plays-by-stream-type", router.handler.TautulliPlaysByStreamType)
		r.Get("/plays-by-source-resolution", router.handler.TautulliPlaysBySourceResolution)
		r.Get("/plays-by-stream-resolution", router.handler.TautulliPlaysByStreamResolution)
		r.Get("/plays-by-top-10-platforms", router.handler.TautulliPlaysByTop10Platforms)
		r.Get("/plays-by-top-10-users", router.handler.TautulliPlaysByTop10Users)
		r.Get("/plays-per-month", router.handler.TautulliPlaysPerMonth)
		r.Get("/concurrent-streams-by-stream-type", router.handler.TautulliConcurrentStreamsByStreamType)

		// User Management
		r.Get("/user", router.handler.TautulliUser)
		r.Get("/users", router.handler.TautulliUsers)
		r.Get("/users-table", router.handler.TautulliUsersTable)
		r.Get("/user-ips", router.handler.TautulliUserIPs)
		r.Get("/user-logins", router.handler.TautulliUserLogins)
		r.Get("/user-player-stats", router.handler.TautulliUserPlayerStats)
		r.Get("/user-watch-time-stats", router.handler.TautulliUserWatchTimeStats)

		// Library Management
		r.Get("/libraries", router.handler.TautulliLibraries)
		r.Get("/libraries-table", router.handler.TautulliLibrariesTable)
		r.Get("/library", router.handler.TautulliLibrary)
		r.Get("/library-media-info", router.handler.TautulliLibraryMediaInfo)
		r.Get("/library-names", router.handler.TautulliLibraryNames)
		r.Get("/library-user-stats", router.handler.TautulliLibraryUserStats)
		r.Get("/library-watch-time-stats", router.handler.TautulliLibraryWatchTimeStats)

		// Metadata
		r.Get("/metadata", router.handler.TautulliMetadata)
		r.Get("/children-metadata", router.handler.TautulliChildrenMetadata)
		r.Get("/item-user-stats", router.handler.TautulliItemUserStats)
		r.Get("/item-watch-time-stats", router.handler.TautulliItemWatchTimeStats)
		r.Get("/recently-added", router.handler.TautulliRecentlyAdded)

		// Stream Data
		r.Get("/stream-data", router.handler.TautulliStreamData)
		r.Get("/stream-type-by-top-10-users", router.handler.TautulliStreamTypeByTop10Users)
		r.Get("/stream-type-by-top-10-platforms", router.handler.TautulliStreamTypeByTop10Platforms)

		// Server Information
		r.Get("/server-info", router.handler.TautulliServerInfo)
		r.Get("/server-friendly-name", router.handler.TautulliServerFriendlyName)
		r.Get("/server-id", router.handler.TautulliServerID)
		r.Get("/server-identity", router.handler.TautulliServerIdentity)
		r.Get("/server-pref", router.handler.TautulliServerPref)
		r.Get("/server-list", router.handler.TautulliServerList)
		r.Get("/servers-info", router.handler.TautulliServersInfo)
		r.Get("/tautulli-info", router.handler.TautulliTautulliInfo)
		r.Get("/pms-update", router.handler.TautulliPMSUpdate)

		// Collections & Playlists
		r.Get("/collections-table", router.handler.TautulliCollectionsTable)
		r.Get("/playlists-table", router.handler.TautulliPlaylistsTable)
		r.Get("/synced-items", router.handler.TautulliSyncedItems)

		// Search & Rating Keys
		r.Get("/search", router.handler.TautulliSearch)
		r.Get("/new-rating-keys", router.handler.TautulliNewRatingKeys)
		r.Get("/old-rating-keys", router.handler.TautulliOldRatingKeys)

		// Export
		r.Get("/export-metadata", router.handler.TautulliExportMetadata)
		r.Get("/export-fields", router.handler.TautulliExportFields)
		r.Get("/exports-table", router.handler.TautulliExportsTable)
		r.Get("/download-export", router.handler.TautulliDownloadExport)
		r.Delete("/delete-export", router.handler.TautulliDeleteExport)
		r.Post("/terminate-session", router.handler.TautulliTerminateSession)
	})

	// ========================
	// Export Endpoints
	// ========================
	// Phase 3: Strict rate limiting for exports (10/min - resource intensive)
	// SECURITY FIX (HIGH-001): Export endpoints require authentication - prevents data exfiltration
	r.Route("/api/v1/export", func(r chi.Router) {
		r.Use(router.chiMiddleware.RateLimitExport())
		r.Use(chiMiddleware(router.middleware.Authenticate)) // SECURITY: Require auth for data exports

		r.Get("/geoparquet", router.handler.ExportGeoParquet)
		r.Get("/geojson", router.handler.ExportGeoJSON)
		r.Get("/playbacks/csv", router.handler.ExportPlaybacksCSV)
		r.Get("/locations/geojson", router.handler.ExportLocationsGeoJSON)
	})

	// ========================
	// Streaming Endpoints
	// ========================
	// Phase 3: Export rate limiting for streaming endpoints
	// SECURITY FIX: Streaming endpoints require authentication
	r.Route("/api/v1/stream", func(r chi.Router) {
		r.Use(router.chiMiddleware.RateLimitExport())
		r.Use(chiMiddleware(router.middleware.Authenticate)) // SECURITY: Require auth for streaming data

		r.Get("/locations-geojson", router.handler.StreamLocationsGeoJSON)
	})

	// ========================
	// Wrapped Reports (Annual Year-in-Review)
	// ========================
	// Spotify Wrapped-style annual analytics for users
	r.Route("/api/v1/wrapped", func(r chi.Router) {
		r.Use(router.chiMiddleware.RateLimitAnalytics())
		r.Use(APISecurityHeaders())
		r.Use(chiMiddleware(middleware.PrometheusMetrics))

		// Public share endpoint (no auth required)
		r.Get("/share/{token}", router.handler.WrappedShare)

		// Authenticated endpoints
		r.Group(func(r chi.Router) {
			r.Use(chiMiddleware(router.middleware.Authenticate))

			// Year-specific endpoints
			r.Route("/{year}", func(r chi.Router) {
				r.Get("/", router.handler.WrappedServerStats)
				r.Get("/user/{userID}", router.handler.WrappedUserReport)
				r.Get("/leaderboard", router.handler.WrappedLeaderboard)
				r.Post("/generate", router.handler.WrappedGenerate)
			})
		})
	})

	// ========================
	// Personal Access Tokens (PAT)
	// ========================
	// API token management for programmatic access
	r.Route("/api/v1/user/tokens", func(r chi.Router) {
		r.Use(router.chiMiddleware.RateLimit())
		r.Use(APISecurityHeaders())
		r.Use(chiMiddleware(middleware.PrometheusMetrics))
		r.Use(chiMiddleware(router.middleware.Authenticate))

		// Token CRUD
		r.Get("/", router.handler.PATList)
		r.Post("/", router.handler.PATCreate)
		r.Get("/stats", router.handler.PATStats)

		// Token-specific operations
		r.Route("/{id}", func(r chi.Router) {
			r.Use(chiPathValue)
			r.Get("/", router.handler.PATGet)
			r.Delete("/", router.handler.PATRevoke)
			r.Post("/regenerate", router.handler.PATRegenerate)
			r.Get("/logs", router.handler.PATUsageLogs)
		})
	})

	// ========================
	// Vector Tiles
	// ========================
	// No rate limit for tiles - they're cached
	r.Route("/api/v1/tiles", func(r chi.Router) {
		r.Get("/*", router.handler.GetVectorTile)
	})

	// ========================
	// Backup Endpoints
	// ========================
	r.Route("/api/v1/backup", func(r chi.Router) {
		r.Use(router.chiMiddleware.RateLimit())
		r.Use(chiMiddleware(router.middleware.Authenticate))

		r.Post("/", router.handler.HandleCreateBackup)
		r.Post("/quick", router.handler.HandleQuickBackup)
		r.Get("/stats", router.handler.HandleGetBackupStats)
		r.Get("/retention", router.handler.HandleGetRetentionPolicy)
		r.Put("/retention", router.handler.HandleSetRetentionPolicy)
		r.Get("/retention/preview", router.handler.HandleRetentionPreview)
		r.Post("/retention/apply", router.handler.HandleApplyRetention)
		r.Post("/cleanup", router.handler.HandleCleanupCorrupted)
		r.Get("/schedule", router.handler.HandleGetScheduleConfig)
		r.Put("/schedule", router.handler.HandleSetScheduleConfig)
		r.Post("/schedule/trigger", router.handler.HandleTriggerScheduledBackup)
	})

	r.Route("/api/v1/backups", func(r chi.Router) {
		r.Use(router.chiMiddleware.RateLimit())
		r.Use(chiMiddleware(router.middleware.Authenticate))

		r.Get("/", router.handler.HandleListBackups)
		r.Get("/get", router.handler.HandleGetBackup)
		r.Delete("/delete", router.handler.HandleDeleteBackup)
		r.Post("/validate", router.handler.HandleValidateBackup)
		r.Post("/restore", router.handler.HandleRestoreBackup)
		r.Get("/download", router.handler.HandleDownloadBackup)
		r.Post("/upload", router.handler.HandleUploadBackup)
	})

	// ========================
	// Newsletter Generator Endpoints
	// ========================
	// Newsletter templates, schedules, delivery, and user preferences
	router.registerChiNewsletterRoutes(r)

	// ========================
	// Detection Endpoints (ADR-0020)
	// ========================
	// Anomaly detection for media playback security
	if router.detectionHandlers != nil {
		router.registerChiDetectionRoutes(r)
	}

	// ========================
	// Audit Log Endpoints
	// ========================
	// Security audit trail for Data Governance
	if router.auditHandlers != nil {
		router.registerChiAuditRoutes(r)
	}

	// ========================
	// DLQ Management
	// ========================
	// Dead Letter Queue visibility and retry management
	if router.dlqHandlers != nil {
		router.registerChiDLQRoutes(r)
	}

	// ========================
	// WAL Stats
	// ========================
	// Write-Ahead Log statistics and health
	if router.walHandlers != nil {
		router.registerChiWALRoutes(r)
	}

	// ========================
	// Replay Management (CRITICAL-002)
	// ========================
	// Deterministic event replay for disaster recovery
	if router.replayHandlers != nil {
		router.registerChiReplayRoutes(r)
	}

	// ========================
	// Recommendation Engine (ADR-0024)
	// ========================
	// Hybrid recommendation engine for media content
	if router.recommendHandler != nil {
		router.registerChiRecommendRoutes(r)
	}

	// ========================
	// Deduplication Audit (ADR-0022)
	// ========================
	// Deduplication visibility, management, and recovery
	router.registerChiDedupeRoutes(r)

	// ========================
	// Import Routes (NATS)
	// ========================
	// Registered dynamically when NATS is enabled
	if router.importRouteRegistrar != nil {
		router.registerChiImportRoutes(r)
	}

	// ========================
	// Sync Routes (Data Sync UI)
	// ========================
	// Sync status and Plex historical sync endpoints
	router.registerChiSyncRoutes(r)

	// ========================
	// Zero Trust Auth Routes
	// ========================
	if router.flowHandlers != nil || router.policyHandlers != nil {
		router.registerChiZeroTrustRoutes(r)
	}

	// ========================
	// Observability
	// ========================
	r.Handle("/metrics", promhttp.Handler())
	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
		httpSwagger.DeepLinking(true),
		httpSwagger.DocExpansion("list"),
		httpSwagger.DomID("swagger-ui"),
	))

	// ========================
	// Static Files & SPA
	// ========================
	// Must be last - catches all unmatched routes
	// Wrap with security headers for CSP nonce injection
	r.Group(func(r chi.Router) {
		r.Use(chiMiddleware(router.middleware.SecurityHeaders))
		r.Get("/*", router.serveStaticOrIndex)
	})

	return r
}

// registerChiImportRoutes adds import routes using Chi router.
// This is called when NATS is enabled.
func (router *Router) registerChiImportRoutes(r chi.Router) {
	// Note: Import routes are registered via importRouteRegistrar
	// which still uses the old mux pattern. For full Chi integration,
	// import_handlers would need to be updated to work with Chi.
	// For now, we mount import routes at a sub-path.

	// The importRouteRegistrar expects *http.ServeMux, so we need
	// a different approach. We'll register the handlers directly here
	// if import handlers are available.
}

// registerChiSyncRoutes adds sync routes for the Data Sync UI.
// Provides endpoints for sync status monitoring and Plex historical sync.
func (router *Router) registerChiSyncRoutes(r chi.Router) {
	r.Route("/api/v1/sync", func(r chi.Router) {
		r.Use(router.chiMiddleware.RateLimit())
		r.Use(APISecurityHeaders())
		r.Use(chiMiddleware(middleware.PrometheusMetrics))
		r.Use(chiMiddleware(router.middleware.Authenticate))

		// Trigger sync - POST /api/v1/sync
		r.Post("/", router.handler.TriggerSync)

		// Sync status - returns combined status of all sync operations
		r.Get("/status", func(w http.ResponseWriter, req *http.Request) {
			if router.syncHandlers != nil {
				router.syncHandlers.HandleGetSyncStatus(w, req)
			} else {
				// Return empty status if no sync handlers configured
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				//nolint:errcheck // HTTP response write errors are not recoverable
				w.Write([]byte(`{"server_syncs":{}}`))
			}
		})

		// Plex historical sync - admin only
		r.Route("/plex/historical", func(r chi.Router) {
			r.Use(router.chiMiddleware.RateLimitAuth()) // Stricter rate limiting for sync triggers
			r.Post("/", func(w http.ResponseWriter, req *http.Request) {
				if router.syncHandlers != nil {
					router.syncHandlers.HandleStartPlexHistoricalSync(w, req)
				} else {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusServiceUnavailable)
					//nolint:errcheck // HTTP response write errors are not recoverable
					w.Write([]byte(`{"success":false,"error":"sync handlers not configured"}`))
				}
			})
		})
	})
}

// registerChiZeroTrustRoutes adds Zero Trust auth routes using Chi router.
func (router *Router) registerChiZeroTrustRoutes(r chi.Router) {
	// ========================
	// OIDC Authentication
	// ========================
	if router.flowHandlers != nil {
		r.Route("/api/auth/oidc", func(r chi.Router) {
			r.Use(router.chiMiddleware.RateLimit())

			r.Get("/login", router.flowHandlers.OIDCLogin)
			r.Get("/callback", router.flowHandlers.OIDCCallback)
			r.Post("/refresh", router.flowHandlers.OIDCRefresh)
			r.Post("/logout", router.sessionMiddleware.Authenticate(
				http.HandlerFunc(router.flowHandlers.OIDCLogout)).ServeHTTP)
			r.Post("/backchannel-logout", router.flowHandlers.BackChannelLogout)
		})

		// Plex Authentication
		r.Route("/api/auth/plex", func(r chi.Router) {
			r.Use(router.chiMiddleware.RateLimit())

			r.Get("/login", router.flowHandlers.PlexLogin)
			r.Get("/poll", router.flowHandlers.PlexPoll)
			r.Post("/callback", router.flowHandlers.PlexCallback)
		})

		// Session Management
		r.Route("/api/auth", func(r chi.Router) {
			r.Use(router.chiMiddleware.RateLimit())
			r.Use(chiPathValue) // Bridge Chi URL params to r.PathValue()

			r.Get("/userinfo", router.sessionMiddleware.Authenticate(
				http.HandlerFunc(router.flowHandlers.UserInfo)).ServeHTTP)
			r.Post("/logout", router.sessionMiddleware.Authenticate(
				http.HandlerFunc(router.flowHandlers.Logout)).ServeHTTP)
			r.Post("/logout/all", router.sessionMiddleware.RequireAuth(
				http.HandlerFunc(router.flowHandlers.LogoutAll)).ServeHTTP)
			r.Get("/sessions", router.sessionMiddleware.RequireAuth(
				http.HandlerFunc(router.flowHandlers.Sessions)).ServeHTTP)
			r.Delete("/sessions/{id}", router.sessionMiddleware.RequireAuth(
				http.HandlerFunc(router.handleChiRevokeSession)).ServeHTTP)
		})
	}

	// ========================
	// Authorization Routes
	// ========================
	// SECURITY FIX (CRITICAL-003): Admin role endpoints require authentication
	if router.policyHandlers != nil {
		r.Route("/api/admin/roles", func(r chi.Router) {
			r.Use(router.chiMiddleware.RateLimit())
			r.Use(chiPathValue) // Bridge Chi URL params to r.PathValue()

			// Read operations require authentication (CRITICAL-003 fix)
			r.Get("/", router.sessionMiddleware.RequireAuth(
				http.HandlerFunc(router.policyHandlers.ListRoles)).ServeHTTP)
			r.Get("/{role}/permissions", router.sessionMiddleware.RequireAuth(
				http.HandlerFunc(router.handleChiRolePermissions)).ServeHTTP)

			// Write operations require admin role
			r.Post("/assign", router.sessionMiddleware.RequireRole("admin",
				http.HandlerFunc(router.policyHandlers.AssignRole)).ServeHTTP)
			r.Post("/revoke", router.sessionMiddleware.RequireRole("admin",
				http.HandlerFunc(router.policyHandlers.RevokeRole)).ServeHTTP)
		})

		r.Route("/api/auth", func(r chi.Router) {
			r.Use(router.chiMiddleware.RateLimit())

			r.Post("/check", router.sessionMiddleware.RequireAuth(
				http.HandlerFunc(router.policyHandlers.CheckPermission)).ServeHTTP)
			r.Get("/roles", router.sessionMiddleware.RequireAuth(
				http.HandlerFunc(router.policyHandlers.GetUserRoles)).ServeHTTP)
		})

		// ADR-0016: Use Chi middleware for /api/admin/policies
		r.Route("/api/admin/policies", func(r chi.Router) {
			r.Use(router.chiMiddleware.RateLimit())
			r.Get("/", router.sessionMiddleware.RequireRole("admin",
				http.HandlerFunc(router.policyHandlers.GetPolicies)).ServeHTTP)
		})
	}

	// ========================
	// Server Management Routes
	// ========================
	// ADR-0026: Multi-Server Management UI
	r.Route("/api/v1/admin/servers", func(r chi.Router) {
		r.Use(router.chiMiddleware.RateLimit())
		r.Use(APISecurityHeaders())
		r.Use(chiMiddleware(middleware.PrometheusMetrics))
		r.Use(chiMiddleware(router.middleware.Authenticate))

		// Phase 1: Read-only view of configured media servers (env vars)
		r.Get("/", router.sessionMiddleware.RequireRole("admin",
			http.HandlerFunc(router.handler.ServerStatus)).ServeHTTP)

		// Phase 2: CRUD operations for database-stored servers
		r.Post("/", router.sessionMiddleware.RequireRole("admin",
			http.HandlerFunc(router.handler.CreateServer)).ServeHTTP)

		// Test server connectivity (without saving)
		r.Post("/test", router.sessionMiddleware.RequireRole("admin",
			http.HandlerFunc(router.handler.TestServerConnection)).ServeHTTP)

		// List only database servers (excludes env-var servers)
		r.Get("/db", router.sessionMiddleware.RequireRole("admin",
			http.HandlerFunc(router.handler.ListDBServers)).ServeHTTP)

		// Individual server operations
		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", router.sessionMiddleware.RequireRole("admin",
				http.HandlerFunc(router.handler.GetServer)).ServeHTTP)
			r.Put("/", router.sessionMiddleware.RequireRole("admin",
				http.HandlerFunc(router.handler.UpdateServer)).ServeHTTP)
			r.Delete("/", router.sessionMiddleware.RequireRole("admin",
				http.HandlerFunc(router.handler.DeleteServer)).ServeHTTP)
		})
	})

	// ========================
	// Mock Data Seeding (CI/Development only)
	// ========================
	// POST /api/v1/admin/seed - Seed database with mock data for screenshots
	// Protected by environment checks (CI, development) and optionally auth
	r.Route("/api/v1/admin/seed", func(r chi.Router) {
		r.Use(router.chiMiddleware.RateLimit())
		r.Use(APISecurityHeaders())
		r.Use(chiMiddleware(middleware.PrometheusMetrics))
		// Note: Authentication is optional for seeding in CI mode (AUTH_MODE=none)
		// The handler itself validates the environment
		r.Post("/", router.handler.SeedMockData)
	})
}

// handleChiRevokeSession extracts session ID from Chi URL param.
func (router *Router) handleChiRevokeSession(w http.ResponseWriter, req *http.Request) {
	sessionID := chi.URLParam(req, "id")
	if sessionID == "" {
		http.Error(w, "Session ID required", http.StatusBadRequest)
		return
	}
	router.flowHandlers.RevokeSession(w, req, sessionID)
}

// handleChiRolePermissions extracts role from Chi URL param.
func (router *Router) handleChiRolePermissions(w http.ResponseWriter, req *http.Request) {
	role := chi.URLParam(req, "role")
	if role == "" {
		http.Error(w, "Role required", http.StatusBadRequest)
		return
	}
	router.policyHandlers.GetRolePermissions(w, req, role)
}

// chiPathValue middleware injects Chi URL params into request so handlers
// using r.PathValue() continue to work. This bridges Chi's chi.URLParam()
// with Go 1.22+'s r.PathValue().
func chiPathValue(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Chi stores URL params in context via chi.RouteContext
		// Go 1.22's PathValue reads from request pattern match
		// We use SetPathValue to make Chi params available via PathValue
		rctx := chi.RouteContext(r.Context())
		if rctx != nil {
			for i, key := range rctx.URLParams.Keys {
				if i < len(rctx.URLParams.Values) {
					r.SetPathValue(key, rctx.URLParams.Values[i])
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}

// registerChiNewsletterRoutes adds Newsletter Generator routes using Chi router.
// Newsletter system with Tautulli parity and enhanced features.
// SECURITY: All endpoints require authentication with RBAC enforcement.
func (router *Router) registerChiNewsletterRoutes(r chi.Router) {
	// Newsletter Templates (RBAC: viewer for read, editor for write, admin for delete)
	r.Route("/api/v1/newsletter/templates", func(r chi.Router) {
		r.Use(router.chiMiddleware.RateLimit())
		r.Use(chiPathValue)
		r.Use(chiMiddleware(middleware.PrometheusMetrics))
		r.Use(chiMiddleware(router.middleware.Authenticate))

		r.Get("/", router.handler.NewsletterTemplateList)
		r.Post("/", router.handler.NewsletterTemplateCreate)
		r.Post("/preview", router.handler.NewsletterTemplatePreview)
		r.Get("/{id}", router.handler.NewsletterTemplateGet)
		r.Put("/{id}", router.handler.NewsletterTemplateUpdate)
		r.Delete("/{id}", router.handler.NewsletterTemplateDelete)
	})

	// Newsletter Schedules (RBAC: viewer for read, editor for write, admin for delete)
	r.Route("/api/v1/newsletter/schedules", func(r chi.Router) {
		r.Use(router.chiMiddleware.RateLimit())
		r.Use(chiPathValue)
		r.Use(chiMiddleware(middleware.PrometheusMetrics))
		r.Use(chiMiddleware(router.middleware.Authenticate))

		r.Get("/", router.handler.NewsletterScheduleList)
		r.Post("/", router.handler.NewsletterScheduleCreate)
		r.Get("/{id}", router.handler.NewsletterScheduleGet)
		r.Put("/{id}", router.handler.NewsletterScheduleUpdate)
		r.Delete("/{id}", router.handler.NewsletterScheduleDelete)
		r.Post("/{id}/trigger", router.handler.NewsletterScheduleTrigger)
	})

	// Newsletter Deliveries (read-only for viewers)
	r.Route("/api/v1/newsletter/deliveries", func(r chi.Router) {
		r.Use(router.chiMiddleware.RateLimit())
		r.Use(chiPathValue)
		r.Use(chiMiddleware(middleware.PrometheusMetrics))
		r.Use(chiMiddleware(router.middleware.Authenticate))

		r.Get("/", router.handler.NewsletterDeliveryList)
		r.Get("/{id}", router.handler.NewsletterDeliveryGet)
	})

	// Newsletter Stats and Audit (stats: viewer, audit: admin)
	r.Route("/api/v1/newsletter", func(r chi.Router) {
		r.Use(router.chiMiddleware.RateLimit())
		r.Use(chiMiddleware(middleware.PrometheusMetrics))
		r.Use(chiMiddleware(router.middleware.Authenticate))

		r.Get("/stats", router.handler.NewsletterStats)
		r.Get("/audit", router.handler.NewsletterAuditLog)
	})

	// User Newsletter Preferences (authenticated users can manage their own)
	r.Route("/api/v1/user/newsletter", func(r chi.Router) {
		r.Use(router.chiMiddleware.RateLimit())
		r.Use(chiMiddleware(middleware.PrometheusMetrics))
		r.Use(chiMiddleware(router.middleware.Authenticate))

		r.Get("/preferences", router.handler.NewsletterUserPreferencesGet)
		r.Put("/preferences", router.handler.NewsletterUserPreferencesUpdate)
		r.Post("/unsubscribe", router.handler.NewsletterUnsubscribe)
	})
}

// registerChiDedupeRoutes adds deduplication audit routes using Chi router.
// ADR-0022: Deduplication audit and management system.
// SECURITY FIX: Dedupe audit data requires authentication
func (router *Router) registerChiDedupeRoutes(r chi.Router) {
	r.Route("/api/v1/dedupe", func(r chi.Router) {
		r.Use(router.chiMiddleware.RateLimit())
		r.Use(chiPathValue) // Bridge Chi URL params to r.PathValue()
		r.Use(chiMiddleware(middleware.PrometheusMetrics))
		r.Use(chiMiddleware(router.middleware.Authenticate)) // SECURITY: Require auth for dedupe audit

		// Read operations
		r.Get("/audit", router.handler.DedupeAuditList)          // List all dedupe events (paginated)
		r.Get("/audit/stats", router.handler.DedupeAuditStats)   // Dedupe statistics dashboard
		r.Get("/audit/export", router.handler.DedupeAuditExport) // Export audit log to CSV
		r.Get("/audit/{id}", router.handler.DedupeAuditGet)      // Get specific dedupe event

		// Write operations
		r.Post("/audit/{id}/restore", router.handler.DedupeAuditRestore) // Restore a deduplicated event
		r.Post("/audit/{id}/confirm", router.handler.DedupeAuditConfirm) // Confirm dedup was correct
	})
}

// registerChiDetectionRoutes adds detection-related routes using Chi router.
// ADR-0020: Detection rules engine for media playback security monitoring.
// SECURITY FIX: Detection/security data requires authentication
func (router *Router) registerChiDetectionRoutes(r chi.Router) {
	r.Route("/api/v1/detection", func(r chi.Router) {
		r.Use(router.chiMiddleware.RateLimit())
		r.Use(chiPathValue) // Bridge Chi URL params to r.PathValue()
		r.Use(chiMiddleware(middleware.PrometheusMetrics))
		r.Use(chiMiddleware(router.middleware.Authenticate)) // SECURITY: Require auth for detection data

		// Read operations
		r.Get("/alerts", router.detectionHandlers.ListAlerts)
		r.Get("/alerts/{id}", router.detectionHandlers.GetAlert)
		r.Get("/rules", router.detectionHandlers.ListRules)
		r.Get("/rules/{type}", router.detectionHandlers.GetRule)
		r.Get("/users/{id}/trust", router.detectionHandlers.GetUserTrustScore)
		r.Get("/users/low-trust", router.detectionHandlers.ListLowTrustUsers)
		r.Get("/metrics", router.detectionHandlers.GetEngineMetrics)
		r.Get("/stats", router.detectionHandlers.GetAlertStats)

		// Write operations
		r.Post("/alerts/{id}/acknowledge", router.detectionHandlers.AcknowledgeAlert)
		r.Put("/rules/{type}", router.detectionHandlers.UpdateRule)
		r.Post("/rules/{type}/enable", router.detectionHandlers.SetRuleEnabled)
	})
}

// registerChiAuditRoutes adds security audit log routes using Chi router.
// SECURITY FIX: Audit log data requires authentication
func (router *Router) registerChiAuditRoutes(r chi.Router) {
	r.Route("/api/v1/audit", func(r chi.Router) {
		r.Use(router.chiMiddleware.RateLimit())
		r.Use(chiPathValue) // Bridge Chi URL params to r.PathValue()
		r.Use(chiMiddleware(middleware.PrometheusMetrics))
		r.Use(chiMiddleware(router.middleware.Authenticate)) // SECURITY: Require auth for audit logs

		// Event listing and querying
		r.Get("/events", router.auditHandlers.ListEvents)
		r.Get("/events/{id}", router.auditHandlers.GetEvent)

		// Statistics
		r.Get("/stats", router.auditHandlers.GetStats)

		// Metadata endpoints
		r.Get("/types", router.auditHandlers.GetTypes)
		r.Get("/severities", router.auditHandlers.GetSeverities)

		// Export
		r.Get("/export", router.auditHandlers.ExportEvents)
	})
}

// registerChiDLQRoutes adds DLQ management routes using Chi router.
// SECURITY FIX: DLQ data requires authentication
func (router *Router) registerChiDLQRoutes(r chi.Router) {
	r.Route("/api/v1/dlq", func(r chi.Router) {
		r.Use(router.chiMiddleware.RateLimit())
		r.Use(chiPathValue) // Bridge Chi URL params to r.PathValue()
		r.Use(chiMiddleware(middleware.PrometheusMetrics))
		r.Use(chiMiddleware(router.middleware.Authenticate)) // SECURITY: Require auth for DLQ data

		// Read operations
		r.Get("/entries", router.dlqHandlers.ListEntries)
		r.Get("/entries/{id}", router.dlqHandlers.GetEntry)
		r.Get("/stats", router.dlqHandlers.GetStats)
		r.Get("/categories", router.dlqHandlers.GetCategories)

		// Write operations
		r.Delete("/entries/{id}", router.dlqHandlers.DeleteEntry)
		r.Post("/entries/{id}/retry", router.dlqHandlers.RetryEntry)
		r.Post("/retry-all", router.dlqHandlers.RetryAllPending)
		r.Post("/cleanup", router.dlqHandlers.Cleanup)
	})
}

// registerChiWALRoutes adds WAL statistics routes using Chi router.
// SECURITY FIX: WAL data requires authentication
func (router *Router) registerChiWALRoutes(r chi.Router) {
	r.Route("/api/v1/wal", func(r chi.Router) {
		r.Use(router.chiMiddleware.RateLimit())
		r.Use(chiMiddleware(middleware.PrometheusMetrics))
		r.Use(chiMiddleware(router.middleware.Authenticate)) // SECURITY: Require auth for WAL data

		// Read operations
		r.Get("/stats", router.walHandlers.GetStats)
		r.Get("/health", router.walHandlers.GetHealth)

		// Write operations
		r.Post("/compact", router.walHandlers.TriggerCompaction)
	})
}

// registerChiReplayRoutes adds admin routes for event replay management.
// CRITICAL-002: Deterministic event replay for disaster recovery.
func (router *Router) registerChiReplayRoutes(r chi.Router) {
	r.Route("/api/v1/admin/replay", func(r chi.Router) {
		r.Use(router.chiMiddleware.RateLimit())
		r.Use(chiPathValue) // Bridge Chi URL params to r.PathValue()
		r.Use(chiMiddleware(middleware.PrometheusMetrics))

		// All replay operations require authentication (admin only)
		r.Use(chiMiddleware(router.middleware.Authenticate))

		// Checkpoint management
		r.Get("/checkpoints", router.replayHandlers.ListCheckpoints)
		r.Get("/checkpoints/last", router.replayHandlers.GetLastCheckpoint)
		r.Get("/checkpoints/{id}", router.replayHandlers.GetCheckpoint)
		r.Delete("/checkpoints/{id}", router.replayHandlers.DeleteCheckpoint)
		r.Post("/checkpoints/cleanup", router.replayHandlers.CleanupOldCheckpoints)

		// Replay operations
		r.Post("/start", router.replayHandlers.StartReplay)
	})
}

// registerChiRecommendRoutes adds recommendation engine routes using Chi router.
// ADR-0024: Hybrid recommendation engine for media content.
// SECURITY: All recommendation endpoints require authentication.
func (router *Router) registerChiRecommendRoutes(r chi.Router) {
	r.Route("/api/v1/recommendations", func(r chi.Router) {
		r.Use(router.chiMiddleware.RateLimit())
		r.Use(chiPathValue) // Bridge Chi URL params to r.PathValue()
		r.Use(chiMiddleware(middleware.PrometheusMetrics))
		r.Use(chiMiddleware(router.middleware.Authenticate)) // SECURITY: Require auth for recommendations

		// Status and configuration
		r.Get("/status", router.recommendHandler.GetRecommendationStatus)
		r.Get("/config", router.recommendHandler.GetRecommendationConfig)
		r.Put("/config", router.recommendHandler.UpdateRecommendationConfig)
		r.Post("/train", router.recommendHandler.TriggerTraining)

		// Algorithm information
		r.Get("/algorithms", router.recommendHandler.GetAlgorithms)
		r.Get("/algorithms/metrics", router.recommendHandler.GetAlgorithmMetrics)

		// User recommendations
		r.Get("/user/{userID}", router.recommendHandler.GetRecommendations)
		r.Get("/user/{userID}/continue", router.recommendHandler.GetContinueWatching)
		r.Get("/user/{userID}/explore", router.recommendHandler.GetExploreRecommendations)

		// Item-based recommendations
		r.Get("/similar/{itemID}", router.recommendHandler.GetSimilar)
		r.Get("/next/{itemID}", router.recommendHandler.GetWhatsNext)
	})
}
