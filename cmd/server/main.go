// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package main is the entry point for the Cartographus server application.
//
// Cartographus is a self-hosted media server analytics platform that visualizes
// playback activity from Plex, Jellyfin, and Emby on interactive maps. It provides
// geographic visualization of streaming sessions, analytics dashboards, and
// real-time monitoring capabilities.
//
// # Application Architecture
//
// The server initializes components in the following order:
//
//  1. Configuration: Load settings from environment variables and config files (Koanf v2)
//  2. Database: Initialize DuckDB with spatial extensions for geographic queries
//  3. Sync Manager: Connect to enabled media sources (Tautulli, Plex, Jellyfin, Emby)
//  4. WebSocket Hub: Enable real-time updates to connected clients
//  5. Authentication: Configure JWT, Basic Auth, or no-auth mode
//  6. NATS (optional): Event-driven processing with JetStream persistence
//  7. Backup Manager: Scheduled backups with retention policies
//  8. HTTP Server: REST API with 182 endpoints and Swagger documentation
//
// # Configuration
//
// Configuration is loaded via Koanf v2 with layered sources (highest priority wins):
//   - Environment variables (see .env.example)
//   - Config file (config.yaml)
//   - Built-in defaults
//
// # Standalone Mode (v2.0+)
//
// Cartographus can run WITHOUT Tautulli in standalone mode, connecting directly to:
//   - Plex: PLEX_ENABLED=true, PLEX_URL, PLEX_TOKEN
//   - Jellyfin: JELLYFIN_ENABLED=true, JELLYFIN_URL, JELLYFIN_API_KEY
//   - Emby: EMBY_ENABLED=true, EMBY_URL, EMBY_API_KEY
//
// Optional Tautulli integration (for migration or enhanced metadata):
//   - TAUTULLI_ENABLED=true (default: false)
//   - TAUTULLI_URL: Tautulli server URL (e.g., http://localhost:8181)
//   - TAUTULLI_API_KEY: API key from Tautulli Settings > Web Interface
//
// For JWT authentication (default):
//   - JWT_SECRET: 32+ character secret for token signing
//   - ADMIN_USERNAME: Admin username
//   - ADMIN_PASSWORD: Admin password (8+ characters)
//
// # Build Tags
//
// Optional build tags enable additional functionality:
//
//	go build -tags "nats" ./cmd/server      # Enable NATS JetStream
//	go build -tags "wal" ./cmd/server       # Enable BadgerDB WAL
//	go build -tags "nats,wal" ./cmd/server  # Enable both
//
// # Signal Handling
//
// The server handles graceful shutdown on SIGINT and SIGTERM:
//   - Stops accepting new connections
//   - Waits for in-flight requests to complete (10s timeout)
//   - Closes sync manager and database connections
//   - Shuts down NATS components if enabled
//
// # Example Usage
//
// Standalone mode with Plex (no Tautulli):
//
//	export PLEX_ENABLED=true
//	export PLEX_URL=http://localhost:32400
//	export PLEX_TOKEN=your-plex-token
//	export AUTH_MODE=none  # For development
//	./cartographus
//
// Standalone mode with Jellyfin:
//
//	export JELLYFIN_ENABLED=true
//	export JELLYFIN_URL=http://localhost:8096
//	export JELLYFIN_API_KEY=your-jellyfin-api-key
//	./cartographus
//
// With Tautulli (optional, for migration or enhanced metadata):
//
//	export TAUTULLI_ENABLED=true
//	export TAUTULLI_URL=http://localhost:8181
//	export TAUTULLI_API_KEY=your-api-key
//	export AUTH_MODE=none  # For development
//	./cartographus
//
// Production with JWT and Plex:
//
//	export PLEX_ENABLED=true
//	export PLEX_URL=http://plex:32400
//	export PLEX_TOKEN=your-plex-token
//	export JWT_SECRET=$(openssl rand -base64 32)
//	export ADMIN_USERNAME=admin
//	export ADMIN_PASSWORD=secure-password
//	./cartographus
//
// Docker (standalone with Plex):
//
//	docker run -d \
//	  -e PLEX_ENABLED=true \
//	  -e PLEX_URL=http://plex:32400 \
//	  -e PLEX_TOKEN=your-plex-token \
//	  -p 3857:3857 \
//	  ghcr.io/tomtom215/cartographus
//
// # Port 3857
//
// The default port 3857 references EPSG:3857 (Web Mercator projection),
// the coordinate system used by web mapping libraries.
package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	_ "github.com/tomtom215/cartographus/docs" // Import generated swagger docs
	"github.com/tomtom215/cartographus/internal/api"
	"github.com/tomtom215/cartographus/internal/audit"
	"github.com/tomtom215/cartographus/internal/auth"
	"github.com/tomtom215/cartographus/internal/backup"
	"github.com/tomtom215/cartographus/internal/config"
	"github.com/tomtom215/cartographus/internal/database"
	"github.com/tomtom215/cartographus/internal/detection"
	"github.com/tomtom215/cartographus/internal/logging"
	"github.com/tomtom215/cartographus/internal/supervisor"
	"github.com/tomtom215/cartographus/internal/supervisor/services"
	"github.com/tomtom215/cartographus/internal/sync"
	"github.com/tomtom215/cartographus/internal/vpn"
	ws "github.com/tomtom215/cartographus/internal/websocket"
)

//nolint:gocyclo // Main initialization function with sequential setup steps
func main() {
	// Load configuration first to get logging settings
	cfg, err := config.Load()
	if err != nil {
		// Use default logger for config errors (config not yet available)
		logging.Fatal().Err(err).Msg("Failed to load configuration")
	}

	// Initialize zerolog with configuration
	logging.Init(logging.Config{
		Level:  cfg.Logging.Level,
		Format: cfg.Logging.Format,
		Caller: cfg.Logging.Caller,
	})

	logging.Info().Msg("Starting Cartographus with supervisor tree")

	// Log configuration status - show Tautulli status based on Enabled flag
	if cfg.Tautulli.Enabled {
		logging.Info().
			Str("tautulli_url", cfg.Tautulli.URL).
			Str("db_path", cfg.Database.Path).
			Str("auth_mode", cfg.Security.AuthMode).
			Msg("Configuration loaded")
	} else {
		logging.Info().
			Bool("tautulli_enabled", false).
			Str("db_path", cfg.Database.Path).
			Str("auth_mode", cfg.Security.AuthMode).
			Msg("Configuration loaded (standalone mode)")
	}

	// Initialize database with server location for spatial optimizations
	db, err := database.New(&cfg.Database, cfg.Server.Latitude, cfg.Server.Longitude)
	if err != nil {
		logging.Fatal().Err(err).Msg("Failed to initialize database")
	}
	defer func() {
		if err := db.Close(); err != nil {
			logging.Error().Err(err).Msg("Error closing database")
		}
	}()
	logging.Info().Msg("Database initialized successfully")

	// Seed mock data if enabled (for CI/CD screenshot tests)
	if cfg.Database.SeedMockData {
		logging.Info().Msg("Mock data seeding enabled (SEED_MOCK_DATA=true)")
		if err := db.SeedMockData(context.Background()); err != nil {
			// Close database before fatal exit to ensure defer runs
			if closeErr := db.Close(); closeErr != nil {
				logging.Error().Err(closeErr).Msg("Error closing database")
			}
			logging.Fatal().Err(err).Msg("Failed to seed mock data")
		}
	}

	// Log spatial optimization status
	if cfg.Server.Latitude != 0.0 || cfg.Server.Longitude != 0.0 {
		logging.Info().
			Float64("latitude", cfg.Server.Latitude).
			Float64("longitude", cfg.Server.Longitude).
			Msg("Spatial optimizations enabled with server location")
	} else {
		logging.Info().Msg("Spatial optimizations available without distance calculations (server location not configured)")
	}

	// Initialize Tautulli client with circuit breaker for fault tolerance (v2.0: optional)
	// Circuit breaker prevents cascading failures when Tautulli API is unavailable
	// As of v2.0, Tautulli is OPTIONAL - Cartographus can work standalone with direct
	// Plex, Jellyfin, and/or Emby integrations without requiring Tautulli.
	var tautulliClient sync.TautulliClientInterface
	if cfg.Tautulli.Enabled {
		tautulliClient = sync.NewCircuitBreakerClient(&cfg.Tautulli)
		if err := tautulliClient.Ping(context.Background()); err != nil {
			logging.Warn().Err(err).Msg("Failed to connect to Tautulli (will retry)")
		} else {
			logging.Info().Msg("Connected to Tautulli successfully")
		}
	} else {
		logging.Info().Msg("Tautulli integration disabled - running in standalone mode")
	}

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create structured logger for supervisor using our slog adapter
	// This bridges zerolog to slog for sutureslog compatibility
	slogLogger := logging.NewSlogLogger()

	// Create supervisor tree
	tree, err := supervisor.NewSupervisorTree(slogLogger, supervisor.TreeConfig{
		FailureThreshold: 5,
		FailureBackoff:   15 * time.Second,
		ShutdownTimeout:  10 * time.Second,
	})
	if err != nil {
		logging.Fatal().Err(err).Msg("Failed to create supervisor tree")
	}

	// Create WebSocket hub for real-time updates (before sync manager)
	// This must be created early so the sync manager can use it for Plex WebSocket broadcasts (v1.39)
	wsHub := ws.NewHub()

	// Create sync manager (no longer started here - supervisor will start it)
	// The database implements UserResolver for mapping external user IDs to internal IDs (v2.0)
	syncManager := sync.NewManager(db, db, tautulliClient, cfg, wsHub)

	// Create Jellyfin managers (v2.1: multi-server support)
	// The database is passed as UserResolver for mapping Jellyfin UUIDs to internal user IDs
	var jellyfinManagers []*sync.JellyfinManager
	for _, jfCfg := range cfg.GetJellyfinServers() {
		// Create a copy to avoid pointer reuse issues in loop
		serverCfg := jfCfg
		jfManager := sync.NewJellyfinManager(&serverCfg, wsHub, db)
		if jfManager != nil {
			jellyfinManagers = append(jellyfinManagers, jfManager)
			logging.Info().
				Str("url", serverCfg.URL).
				Bool("realtime", serverCfg.RealtimeEnabled).
				Bool("polling", serverCfg.SessionPollingEnabled).
				Str("server_id", serverCfg.ServerID).
				Msg("Jellyfin integration enabled")
		}
	}
	if len(jellyfinManagers) > 0 {
		logging.Info().Int("count", len(jellyfinManagers)).Msg("Total Jellyfin servers configured")
	}

	// Create Emby managers (v2.1: multi-server support)
	// The database is passed as UserResolver for mapping Emby UUIDs to internal user IDs
	var embyManagers []*sync.EmbyManager
	for _, embyCfg := range cfg.GetEmbyServers() {
		// Create a copy to avoid pointer reuse issues in loop
		serverCfg := embyCfg
		embyMgr := sync.NewEmbyManager(&serverCfg, wsHub, db)
		if embyMgr != nil {
			embyManagers = append(embyManagers, embyMgr)
			logging.Info().
				Str("url", serverCfg.URL).
				Bool("realtime", serverCfg.RealtimeEnabled).
				Bool("polling", serverCfg.SessionPollingEnabled).
				Str("server_id", serverCfg.ServerID).
				Msg("Emby integration enabled")
		}
	}
	if len(embyManagers) > 0 {
		logging.Info().Int("count", len(embyManagers)).Msg("Total Emby servers configured")
	}

	var jwtManager *auth.JWTManager
	var basicAuthManager *auth.BasicAuthManager

	if cfg.Security.AuthMode == "jwt" {
		var err error
		jwtManager, err = auth.NewJWTManager(&cfg.Security)
		if err != nil {
			logging.Fatal().Err(err).Msg("Failed to initialize JWT manager")
		}
		logging.Info().Msg("JWT authentication enabled")
	} else if cfg.Security.AuthMode == "basic" {
		var err error
		basicAuthManager, err = auth.NewBasicAuthManager(
			cfg.Security.AdminUsername,
			cfg.Security.AdminPassword,
		)
		if err != nil {
			logging.Fatal().Err(err).Msg("Failed to initialize Basic Auth manager")
		}
		logging.Info().Msg("Basic authentication enabled")
		logging.Warn().Msg("Basic Auth transmits credentials with each request. Use HTTPS in production!")
	} else if cfg.Security.AuthMode == "none" {
		logging.Warn().Msg("============================================================")
		logging.Warn().Msg("  SECURITY WARNING: Authentication is DISABLED (AUTH_MODE=none)")
		logging.Warn().Msg("  ")
		logging.Warn().Msg("  All endpoints are publicly accessible without authentication!")
		logging.Warn().Msg("  This mode should ONLY be used for:")
		logging.Warn().Msg("    - Local development")
		logging.Warn().Msg("    - Completely isolated private networks")
		logging.Warn().Msg("    - CI/CD testing environments")
		logging.Warn().Msg("  ")
		logging.Warn().Msg("  NEVER use AUTH_MODE=none in production or on public networks!")
		logging.Warn().Msg("============================================================")
	}

	middleware := auth.NewMiddleware(
		jwtManager,
		basicAuthManager,
		cfg.Security.AuthMode,
		cfg.Security.RateLimitReqs,
		cfg.Security.RateLimitWindow,
		cfg.Security.RateLimitDisabled,
		cfg.Security.CORSOrigins,
		cfg.Security.TrustedProxies,
		cfg.Security.BasicAuthDefaultRole,
		cfg.Security.AdminUsername, // Admin username gets admin role for RBAC
	)

	if cfg.Security.RateLimitDisabled {
		logging.Warn().Msg("Rate limiting is DISABLED (DISABLE_RATE_LIMIT=true)")
		logging.Warn().Msg("This should only be used for CI/CD screenshot tests!")
	}

	// CRITICAL-005: Warn about wildcard CORS when authentication is enabled
	if cfg.ShouldWarnAboutCORS() {
		logging.Warn().Msg("============================================================")
		logging.Warn().Msg("  SECURITY WARNING: CORS is configured with wildcard origin (CORS_ORIGINS=*)")
		logging.Warn().Msg("  ")
		logging.Warn().Msg("  This allows ANY website to make cross-origin requests to your API.")
		logging.Warn().Msg("  With authentication enabled, this creates a security vulnerability:")
		logging.Warn().Msg("  attackers can steal credentials via malicious websites.")
		logging.Warn().Msg("  ")
		logging.Warn().Msg("  RECOMMENDED: Set specific origins in production:")
		logging.Warn().Msg("    CORS_ORIGINS=https://yourdomain.com,https://app.yourdomain.com")
		logging.Warn().Msg("============================================================")
	}

	// L-03 Security Fix: Warn about in-memory session store in non-development environments
	if cfg.Security.SessionStore == "memory" && !cfg.IsDevelopment() {
		logging.Warn().Msg("============================================================")
		logging.Warn().Msg("  NOTICE: Session store is set to 'memory' (SESSION_STORE=memory)")
		logging.Warn().Msg("  ")
		logging.Warn().Msg("  Sessions will be lost when the server restarts!")
		logging.Warn().Msg("  This is fine for development, but for production consider:")
		logging.Warn().Msg("    SESSION_STORE=badger")
		logging.Warn().Msg("    SESSION_STORE_PATH=/data/sessions")
		logging.Warn().Msg("  ")
		logging.Warn().Msg("  This provides persistent sessions across restarts.")
		logging.Warn().Msg("============================================================")
	}

	logging.Info().Msg("WebSocket hub started")

	handler := api.NewHandler(db, syncManager, tautulliClient, cfg, jwtManager, wsHub)

	// Register sync completion callback to clear cache and broadcast updates after each sync
	syncManager.SetOnSyncCompleted(handler.OnSyncCompleted)

	// === DETECTION ENGINE INITIALIZATION (ADR-0020) ===
	// Initialize detection system for anomaly detection and security monitoring
	// Must be initialized before NATS so detection handler can subscribe to events
	detectionEngine, detectionHandlers := initDetection(ctx, db, wsHub, cfg)
	if detectionEngine != nil {
		logging.Info().Msg("Detection engine initialized successfully")

		// Start trust score recovery scheduler (runs daily)
		// Uses configuration values for recovery amount
		recoveryAmount := cfg.Detection.TrustScoreRecovery
		if recoveryAmount <= 0 {
			recoveryAmount = 1 // Default to 1 point per day
		}
		detectionEngine.StartTrustScoreRecovery(ctx, recoveryAmount, 24*time.Hour)
	}

	// Initialize NATS event processing (optional - requires build with -tags nats)
	// Wires event publisher to both sync manager and handler for webhooks
	// Provides dual-path consumption: WebSocket (real-time) + DuckDB (persistence)
	// Also registers detection handler for anomaly detection on playback events
	natsComponents, err := InitNATS(cfg, syncManager, wsHub, handler, db, detectionEngine)
	if err != nil {
		logging.Fatal().Err(err).Msg("Failed to initialize NATS")
	}

	// Add NATS to supervisor tree (if enabled)
	// Note: NATS components are started/managed by supervisor, not manually
	AddNATSToSupervisor(tree, natsComponents)

	// Wire event publisher to all Jellyfin/Emby managers (if NATS is enabled)
	// v2.1: Multi-server support - wire to all managers
	if natsComponents != nil {
		if eventPub := natsComponents.EventPublisher(); eventPub != nil {
			for i, jfMgr := range jellyfinManagers {
				jfMgr.SetEventPublisher(eventPub)
				logging.Info().Int("index", i+1).Msg("Event publisher wired to Jellyfin manager")
			}
			for i, embyMgr := range embyManagers {
				embyMgr.SetEventPublisher(eventPub)
				logging.Info().Int("index", i+1).Msg("Event publisher wired to Emby manager")
			}
		}
	}

	// Initialize backup manager for backup/restore functionality
	backupCfg, err := backup.LoadConfig()
	if err != nil {
		logging.Warn().Err(err).Msg("Failed to load backup configuration, backups disabled")
	} else if backupCfg.Enabled {
		backupManager, err := backup.NewManager(backupCfg, db)
		if err != nil {
			logging.Warn().Err(err).Msg("Failed to initialize backup manager")
		} else {
			handler.SetBackupManager(backupManager)
			logging.Info().
				Str("dir", backupCfg.BackupDir).
				Bool("schedule_enabled", backupCfg.Schedule.Enabled).
				Msg("Backup manager initialized")

			// Start backup scheduler if enabled
			if backupCfg.Schedule.Enabled {
				if err := backupManager.Start(ctx); err != nil {
					logging.Warn().Err(err).Msg("Failed to start backup scheduler")
				} else {
					logging.Info().
						Dur("interval", backupCfg.Schedule.Interval).
						Int("preferred_hour", backupCfg.Schedule.PreferredHour).
						Msg("Backup scheduler started")
				}

				// Register pre-sync backup callback if enabled
				if backupCfg.Schedule.PreSyncBackup {
					originalCallback := handler.OnSyncCompleted
					syncManager.SetOnSyncCompleted(func(newRecords int, durationMs int64) {
						// Create pre-sync snapshot before processing
						if _, err := backupManager.CreatePreSyncBackup(context.Background()); err != nil {
							logging.Warn().Err(err).Msg("Pre-sync backup failed")
						}
						// Call original callback
						originalCallback(newRecords, durationMs)
					})
					logging.Info().Msg("Pre-sync backups enabled")
				}
			}
		}
	} else {
		logging.Info().Msg("Backup functionality disabled (BACKUP_ENABLED=false)")
	}

	router := api.NewRouter(handler, middleware)

	// Configure detection handlers if initialized (detection engine was created earlier)
	if detectionHandlers != nil {
		router.ConfigureDetection(detectionHandlers)
		logging.Info().Msg("Detection routes configured")
	}

	// === AUDIT LOGGING SYSTEM INITIALIZATION ===
	// Initialize DuckDB-backed audit store for persistent security audit trail.
	// This addresses CRITICAL-001: Audit events not persisted to database.
	auditStore := audit.NewDuckDBStore(db.Conn())
	if err := auditStore.CreateTable(ctx); err != nil {
		logging.Warn().Err(err).Msg("Failed to create audit events table - audit logging disabled")
	} else {
		// Create audit logger with default config
		auditConfig := audit.DefaultConfig()
		auditLogger := audit.NewLogger(auditStore, auditConfig)
		defer func() {
			if err := auditLogger.Close(); err != nil {
				logging.Error().Err(err).Msg("Error closing audit logger")
			}
		}()

		// Start cleanup routine for retention policy
		auditLogger.StartCleanupRoutine(ctx)

		// Configure audit handlers for the router
		auditHandlers := api.NewAuditHandlers(auditLogger, auditStore)
		router.ConfigureAudit(auditHandlers)
		logging.Info().Msg("Audit logging initialized with DuckDB persistence")
	}

	// Initialize Tautulli database import (optional - requires build with -tags nats)
	// This must be called before router.Setup() to register import routes
	_, err = InitImport(cfg, natsComponents, tree, router)
	if err != nil {
		logging.Fatal().Err(err).Msg("Failed to initialize import")
	}

	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      router.SetupChi(), // ADR-0016: Chi router for route grouping
		ReadTimeout:  cfg.Server.Timeout,
		WriteTimeout: cfg.Server.Timeout,
		IdleTimeout:  60 * time.Second,
	}

	// === ADD SERVICES TO SUPERVISOR TREE ===

	// Messaging layer services
	tree.AddMessagingService(services.NewWebSocketHubService(wsHub))
	tree.AddMessagingService(services.NewSyncService(syncManager))
	if detectionEngine != nil {
		tree.AddMessagingService(services.NewDetectionService(detectionEngine))
		logging.Info().Msg("Detection engine added to supervisor tree")
	}
	logging.Info().Msg("WebSocket hub and sync manager added to supervisor tree")

	// Initialize recommendation engine (if enabled)
	_ = initRecommend(cfg, zerolog.Nop(), tree)

	// Initialize newsletter scheduler (if enabled)
	// Provides cron-based automatic newsletter delivery
	nopLogger := zerolog.Nop()
	_ = initNewsletter(cfg, db, &nopLogger, tree)

	// Add all Jellyfin/Emby managers to supervisor tree (v2.1: multi-server support)
	for _, jfMgr := range jellyfinManagers {
		// Use the server ID as the service name for supervisor logging
		serviceName := fmt.Sprintf("jellyfin-%s", jfMgr.ServerID())
		tree.AddMessagingService(services.NewJellyfinServiceWithName(jfMgr, serviceName))
		logging.Info().Str("service", serviceName).Msg("Jellyfin manager added to supervisor tree")
	}
	for _, embyMgr := range embyManagers {
		// Use the server ID as the service name for supervisor logging
		serviceName := fmt.Sprintf("emby-%s", embyMgr.ServerID())
		tree.AddMessagingService(services.NewEmbyServiceWithName(embyMgr, serviceName))
		logging.Info().Str("service", serviceName).Msg("Emby manager added to supervisor tree")
	}

	// API layer services
	tree.AddAPIService(services.NewHTTPServerService(server, 10*time.Second))
	logging.Info().Str("addr", server.Addr).Msg("HTTP server service added")

	// === START SUPERVISOR TREE ===

	// Setup signal handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		logging.Info().Str("signal", sig.String()).Msg("Received shutdown signal")
		cancel()
	}()

	logging.Info().Msg("Starting supervisor tree...")
	errCh := tree.ServeBackground(ctx)

	// Wait for supervisor to finish (either from signal or error)
	select {
	case <-ctx.Done():
		logging.Info().Msg("Context canceled, waiting for supervisor to finish...")
	case err := <-errCh:
		if err != nil && !errors.Is(err, context.Canceled) {
			logging.Error().Err(err).Msg("Supervisor tree error")
		}
	}

	// Wait for the error channel to close (supervisor finished)
	for err := range errCh {
		if err != nil && !errors.Is(err, context.Canceled) {
			logging.Error().Err(err).Msg("Supervisor shutdown error")
		}
	}

	// Report any services that failed to stop within timeout
	unstopped, _ := tree.UnstoppedServiceReport()
	if len(unstopped) > 0 {
		logging.Warn().Int("count", len(unstopped)).Msg("Services failed to stop within timeout")
		for _, svc := range unstopped {
			logging.Warn().Str("service", svc.Name).Msg("Service failed to stop")
		}
	}

	logging.Info().Msg("Application stopped gracefully")
}

// initDetection initializes the detection engine and handlers.
// ADR-0020: Detection rules engine for media playback security monitoring.
//
// The detection system provides:
//   - Impossible travel detection (geographic anomalies)
//   - Concurrent stream limits (per-user stream caps)
//   - Device IP velocity (rapid IP changes)
//   - Simultaneous locations (active streams from different locations)
//   - Geographic restrictions (country blocklist/allowlist)
//
// Returns nil, nil if initialization fails (non-fatal - app continues without detection).
func initDetection(ctx context.Context, db *database.DB, broadcaster detection.AlertBroadcaster, cfg *config.Config) (*detection.Engine, *api.DetectionHandlers) {
	// Check if detection is disabled
	if !cfg.Detection.Enabled {
		logging.Info().Msg("Detection engine disabled (DETECTION_ENABLED=false)")
		return nil, nil
	}

	// Create the DuckDB store for detection data
	store := detection.NewDuckDBStore(db.Conn())

	// Initialize schema (creates tables if not exist)
	if err := store.InitSchema(ctx); err != nil {
		logging.Warn().Err(err).Msg("Failed to initialize detection schema")
		logging.Info().Msg("Detection system disabled - continuing without anomaly detection")
		return nil, nil
	}

	// Create detection engine
	engine := detection.NewEngine(store, store, store, broadcaster)

	// Register all detectors
	engine.RegisterDetector(detection.NewImpossibleTravelDetector(store))
	engine.RegisterDetector(detection.NewConcurrentStreamsDetector(store))
	engine.RegisterDetector(detection.NewDeviceVelocityDetector(store))
	engine.RegisterDetector(detection.NewSimultaneousLocationsDetector(store))
	engine.RegisterDetector(detection.NewGeoRestrictionDetector(store))
	engine.RegisterDetector(detection.NewUserAgentAnomalyDetector(store))

	// Initialize VPN service for VPN usage detection
	if cfg.VPN.Enabled {
		vpnConfig := &vpn.Config{
			Enabled:        cfg.VPN.Enabled,
			CacheSize:      cfg.VPN.CacheSize,
			DataFile:       cfg.VPN.DataFile,
			AutoUpdate:     cfg.VPN.AutoUpdate,
			UpdateInterval: cfg.VPN.UpdateInterval,
		}
		vpnSvc, err := vpn.NewService(db.Conn(), vpnConfig)
		if err != nil {
			logging.Warn().Err(err).Msg("Failed to create VPN service, VPN detection disabled")
		} else {
			if err := vpnSvc.Initialize(ctx); err != nil {
				logging.Warn().Err(err).Msg("Failed to initialize VPN service, VPN detection disabled")
			} else {
				// Import VPN data from file if configured
				if cfg.VPN.DataFile != "" {
					if _, err := vpnSvc.ImportFromFile(ctx, cfg.VPN.DataFile); err != nil {
						logging.Warn().Err(err).Str("file", cfg.VPN.DataFile).Msg("Failed to import VPN data file")
					}
				}
				// Register VPN usage detector
				engine.RegisterDetector(detection.NewVPNUsageDetector(vpnSvc))
				stats := vpnSvc.GetStats()
				logging.Info().
					Int("providers", stats.TotalProviders).
					Int("ips", stats.TotalIPs).
					Msg("VPN detection service initialized")
			}
		}
	} else {
		logging.Info().Msg("VPN detection disabled (VPN_ENABLED=false)")
	}

	// Register Discord notifier if configured
	if cfg.Detection.Discord.Enabled && cfg.Detection.Discord.WebhookURL != "" {
		discordNotifier := detection.NewDiscordNotifier(detection.DiscordConfig{
			WebhookURL:  cfg.Detection.Discord.WebhookURL,
			Enabled:     cfg.Detection.Discord.Enabled,
			RateLimitMs: cfg.Detection.Discord.RateLimitMs,
		})
		engine.RegisterNotifier(discordNotifier)
		logging.Info().Int("rate_limit_ms", cfg.Detection.Discord.RateLimitMs).Msg("Discord notifier registered")
	}

	// Register generic webhook notifier if configured
	if cfg.Detection.Webhook.Enabled && cfg.Detection.Webhook.WebhookURL != "" {
		webhookNotifier := detection.NewWebhookNotifier(detection.WebhookConfig{
			WebhookURL:  cfg.Detection.Webhook.WebhookURL,
			Headers:     cfg.Detection.Webhook.Headers,
			Enabled:     cfg.Detection.Webhook.Enabled,
			RateLimitMs: cfg.Detection.Webhook.RateLimitMs,
		})
		engine.RegisterNotifier(webhookNotifier)
		logging.Info().
			Str("url", cfg.Detection.Webhook.WebhookURL).
			Int("rate_limit_ms", cfg.Detection.Webhook.RateLimitMs).
			Msg("Webhook notifier registered")
	}

	// Load detector configurations from database
	rules, err := store.ListRules(ctx)
	if err != nil {
		logging.Warn().Err(err).Msg("Failed to load detection rules")
	} else {
		for _, rule := range rules {
			// Configure detector with stored settings
			if len(rule.Config) > 0 {
				if err := engine.ConfigureDetector(rule.RuleType, rule.Config); err != nil {
					logging.Warn().Err(err).Str("detector", string(rule.RuleType)).Msg("Failed to configure detector")
				}
			}
			// Set enabled state
			if err := engine.SetDetectorEnabled(rule.RuleType, rule.Enabled); err != nil {
				logging.Warn().Err(err).Str("detector", string(rule.RuleType)).Msg("Failed to set detector enabled state")
			}
		}
		logging.Info().Int("count", len(rules)).Msg("Loaded detection rules from database")
	}

	// Create API handlers
	handlers := api.NewDetectionHandlers(store, store, store, engine)

	return engine, handlers
}
