// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

/*
manager.go - Sync Manager Lifecycle and Orchestration

This file contains the core sync manager struct, initialization, and lifecycle
methods for orchestrating data synchronization from Tautulli and Plex.

Manager Components:
  - DBInterface: Database operations interface for playback event storage
  - TautulliClient: Primary data source for playback history and geolocation
  - PlexClient: Optional secondary data source for hybrid architecture (v1.37)
  - PlexWebSocketClient: Real-time playback notifications (v1.39)
  - WebSocketHub: Frontend broadcast interface for live updates

Lifecycle Methods:
  - NewManager(): Initialize manager with configuration and dependencies
  - Start(): Begin periodic sync and start all background services
  - Stop(): Gracefully shutdown all services and wait for completion
  - TriggerSync(): Manual sync execution (mutex-protected)
  - LastSyncTime(): Query last successful sync timestamp

Background Services Started:
  - Tautulli sync loop (periodic, configurable interval)
  - Plex historical sync (one-time backfill if enabled)
  - Plex periodic sync (alternative to historical)
  - Plex WebSocket (real-time notifications)
  - Transcode monitoring (active session tracking)
  - Buffer health monitoring (playback quality tracking)

Thread Safety:
  - syncMu: Prevents concurrent sync execution
  - mu: Protects shared state (running, lastSync)
  - bufferHealthMu: Protects buffer health cache
  - All services use WaitGroup for coordinated shutdown
*/

//nolint:staticcheck // File documentation, not package doc
package sync

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/tomtom215/cartographus/internal/logging"

	"github.com/tomtom215/cartographus/internal/config"
	"github.com/tomtom215/cartographus/internal/models"
)

// DBInterface defines the interface for database operations
type DBInterface interface {
	SessionKeyExists(ctx context.Context, sessionKey string) (bool, error)
	GetGeolocation(ctx context.Context, ipAddress string) (*models.Geolocation, error)
	GetGeolocations(ctx context.Context, ipAddresses []string) (map[string]*models.Geolocation, error) // MEDIUM-2: Batch geolocation lookups
	UpsertGeolocation(geo *models.Geolocation) error
	InsertPlaybackEvent(event *models.PlaybackEvent) error
}

// UserResolver resolves external user IDs to internal integer user IDs.
// This interface enables cross-source user tracking by mapping:
//   - Jellyfin/Emby UUIDs to internal integers
//   - Plex user IDs to internal integers (may differ across servers)
//   - Tautulli user IDs to internal integers
//
// The mapping is stored in the user_mappings table and enables:
//   - Consistent user tracking across all media server sources
//   - Cross-platform user correlation (same person on Plex + Jellyfin)
//   - Multi-server deduplication (same user on multiple Plex servers)
type UserResolver interface {
	// ResolveUserID maps an external user ID to an internal integer user ID.
	// Creates a new mapping if one doesn't exist, returning a consistent internal ID.
	//
	// Parameters:
	//   - ctx: Context for cancellation
	//   - source: Source system (plex, jellyfin, emby, tautulli)
	//   - serverID: Server instance identifier (from config)
	//   - externalUserID: External user ID (UUID for Jellyfin/Emby, int string for Plex)
	//   - username: Optional username for display (may be nil)
	//   - friendlyName: Optional friendly name (may be nil)
	//
	// Returns:
	//   - Internal user ID (auto-assigned on first encounter)
	//   - Error if resolution fails
	ResolveUserID(ctx context.Context, source, serverID, externalUserID string, username, friendlyName *string) (int, error)
}

// Manager orchestrates data synchronization from Tautulli to database
type Manager struct {
	db                DBInterface
	userResolver      UserResolver // For resolving external user IDs to internal IDs (v2.0)
	client            TautulliClientInterface
	plexClient        *PlexClient          // Optional: Hybrid Plex + Tautulli data architecture (v1.37)
	plexWSClient      *PlexWebSocketClient // Optional: Real-time Plex WebSocket for instant updates (v1.39)
	cfg               *config.Config       // Full config (changed from *config.SyncConfig for Plex access)
	lastSync          time.Time
	running           bool
	mu                sync.RWMutex
	syncMu            sync.Mutex // Protects concurrent sync execution
	stopChan          chan struct{}
	wg                sync.WaitGroup
	plexSyncTicker    *time.Ticker                           // Periodic Plex sync ticker (v1.37)
	onSyncCompleted   func(newRecords int, durationMs int64) // Callback invoked after successful sync with stats
	wsHub             WebSocketHub                           // WebSocket hub for broadcasting real-time updates to frontend (v1.39)
	bufferHealthMu    sync.RWMutex                           // Protects bufferHealthCache map (v1.41)
	bufferHealthCache map[string]*models.PlexBufferHealth    // Previous buffer health states for drain rate calculation (v1.41)
	eventPublisher    EventPublisher                         // Optional: NATS event publisher for event-driven architecture (v1.47)
	publishWg         sync.WaitGroup                         // Tracks in-flight publish goroutines for deterministic flush (v2.1)
	sessionPoller     *PlexSessionPoller                     // Optional: Backup session polling when WebSocket is insufficient (v1.50)
}

// WebSocketHub interface for broadcasting messages to frontend clients
// Implemented by internal/websocket/Hub
type WebSocketHub interface {
	BroadcastJSON(messageType string, data interface{})
}

// NewManager creates a new sync manager with optional Plex integration (v1.37)
//
// Parameters:
//   - db: Database interface for playback event storage
//   - userResolver: For resolving external user IDs to internal IDs (optional, can be nil)
//   - client: Tautulli API client (primary data source, can be nil if Tautulli disabled)
//   - cfg: Full application configuration (includes Sync and Plex configs)
//   - wsHub: WebSocket hub for broadcasting real-time updates (optional, can be nil)
//
// If cfg.Plex.Enabled is true, initializes PlexClient for hybrid data architecture.
// The userResolver enables proper user tracking across multiple Plex servers.
func NewManager(db DBInterface, userResolver UserResolver, client TautulliClientInterface, cfg *config.Config, wsHub WebSocketHub) *Manager {
	m := &Manager{
		db:                db,
		userResolver:      userResolver,
		client:            client,
		cfg:               cfg,
		wsHub:             wsHub,
		stopChan:          make(chan struct{}),
		bufferHealthCache: make(map[string]*models.PlexBufferHealth), // v1.41: Initialize buffer health cache
	}

	// Log sync configuration for debugging
	logging.Info().
		Bool("sync_all", cfg.Sync.SyncAll).
		Dur("lookback", cfg.Sync.Lookback).
		Dur("interval", cfg.Sync.Interval).
		Int("batch_size", cfg.Sync.BatchSize).
		Msg("Sync manager config loaded")

	// Initialize Plex client if enabled (v1.37 hybrid architecture)
	if cfg.Plex.Enabled {
		m.plexClient = NewPlexClient(cfg.Plex.URL, cfg.Plex.Token)
		logging.Info().Bool("historical", cfg.Plex.HistoricalSync).Int("days_back", cfg.Plex.SyncDaysBack).Dur("interval", cfg.Plex.SyncInterval).Msg("Plex sync enabled")
	}

	return m
}

// SetOnSyncCompleted sets the callback to be invoked after each successful sync
func (m *Manager) SetOnSyncCompleted(callback func(newRecords int, durationMs int64)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onSyncCompleted = callback
}

// Start begins the periodic synchronization process
func (m *Manager) Start(ctx context.Context) error {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return fmt.Errorf("sync manager is already running")
	}

	logging.Info().Msg("Starting sync manager...")

	m.running = true
	m.mu.Unlock()

	// Start Tautulli sync only if enabled (v2.0: Tautulli is now optional)
	if m.cfg.Tautulli.Enabled && m.client != nil {
		// Add all goroutines to WaitGroup BEFORE starting them
		// This prevents Stop() from calling Wait() before all Add() calls complete
		m.wg.Add(2) // One for initial sync, one for sync loop

		// Perform initial sync in background to avoid blocking server startup
		go func() {
			defer m.wg.Done()
			if err := m.performInitialSync(); err != nil {
				logging.Warn().Err(err).Msg("Initial sync failed (will retry)")
			}
		}()

		go m.syncLoop(ctx)
		logging.Info().Msg("Tautulli sync enabled and started")
	} else {
		logging.Info().Msg("Tautulli sync disabled (TAUTULLI_ENABLED=false) - running in standalone mode")
	}

	// Start all Plex-related services
	m.startPlexServices(ctx)

	return nil
}

// startPlexServices starts all Plex-related background services (sync, websocket, monitoring).
// All services are non-critical - failures are logged but don't block startup.
func (m *Manager) startPlexServices(ctx context.Context) {
	m.startPlexSyncService(ctx)
	m.startPlexWebSocketService(ctx)
	m.startPlexMonitoringServices(ctx)
	m.startPlexSessionPollingService(ctx)
}

// startPlexSyncService starts historical or periodic Plex sync.
func (m *Manager) startPlexSyncService(ctx context.Context) {
	if m.plexClient == nil {
		return
	}

	if m.cfg.Plex.HistoricalSync {
		logging.Info().Msg("Starting Plex historical sync (one-time backfill)...")
		m.wg.Add(1)
		go func() {
			defer m.wg.Done()
			if err := m.syncPlexHistorical(ctx); err != nil {
				logging.Warn().Err(err).Msg("Plex historical sync failed")
			} else {
				logging.Info().Msg("Plex historical sync completed successfully")
			}
		}()
	} else {
		logging.Info().Dur("interval", m.cfg.Plex.SyncInterval).Msg("Starting Plex periodic sync...")
		m.wg.Add(1)
		go m.runPlexSyncLoop(ctx)
	}
}

// startPlexWebSocketService starts real-time WebSocket if enabled.
func (m *Manager) startPlexWebSocketService(ctx context.Context) {
	if !m.cfg.Plex.Enabled || !m.cfg.Plex.RealtimeEnabled {
		return
	}

	logging.Info().Msg("Starting Plex WebSocket for real-time updates...")
	if err := m.StartPlexWebSocket(ctx); err != nil {
		logging.Warn().Err(err).Msg("Failed to start Plex WebSocket")
	}
}

// startPlexMonitoringServices starts transcode and buffer health monitoring.
func (m *Manager) startPlexMonitoringServices(ctx context.Context) {
	if !m.cfg.Plex.Enabled {
		return
	}

	if m.cfg.Plex.TranscodeMonitoring {
		logging.Info().Msg("Starting Plex transcode monitoring...")
		if err := m.StartTranscodeMonitoring(ctx); err != nil {
			logging.Warn().Err(err).Msg("Failed to start transcode monitoring")
		}
	}

	if m.cfg.Plex.BufferHealthMonitoring {
		logging.Info().Msg("Starting Plex buffer health monitoring...")
		if err := m.StartBufferHealthMonitoring(ctx); err != nil {
			logging.Warn().Err(err).Msg("Failed to start buffer health monitoring")
		}
	}
}

// startPlexSessionPollingService starts the optional session polling backup mechanism.
//
// This is a BACKUP mechanism for environments where WebSocket is unreliable.
// In most cases, you should use PLEX_REALTIME_ENABLED instead.
//
// Session polling is only needed if:
// - Your network blocks WebSocket connections
// - Plex WebSocket has reliability issues in your environment
// - You want extra redundancy for mission-critical deployments
func (m *Manager) startPlexSessionPollingService(ctx context.Context) {
	if !m.cfg.Plex.Enabled || !m.cfg.Plex.SessionPollingEnabled {
		return
	}

	logging.Info().Dur("interval", m.cfg.Plex.SessionPollingInterval).Msg("Starting Plex session polling...")
	logging.Info().Msg("NOTE: Session polling is a backup mechanism. PLEX_REALTIME_ENABLED (WebSocket) is recommended.")

	pollerConfig := SessionPollerConfig{
		Interval:       m.cfg.Plex.SessionPollingInterval,
		PublishAll:     false,
		SeenSessionTTL: 1 * time.Hour,
	}

	// Enforce minimum poll interval to protect Plex server
	if pollerConfig.Interval < 10*time.Second {
		logging.Warn().Dur("interval", pollerConfig.Interval).Msg("SessionPollingInterval too low, using minimum 10s")
		pollerConfig.Interval = 10 * time.Second
	}

	m.sessionPoller = NewPlexSessionPoller(m, pollerConfig)
	if err := m.sessionPoller.Start(ctx); err != nil {
		logging.Warn().Err(err).Msg("Failed to start session polling")
		m.sessionPoller = nil
	} else {
		logging.Info().Msg("Plex session polling started successfully")
	}
}

// Stop gracefully stops the synchronization process
func (m *Manager) Stop() error {
	m.mu.Lock()
	if !m.running {
		m.mu.Unlock()
		return fmt.Errorf("sync manager is not running")
	}
	m.running = false
	m.mu.Unlock()

	logging.Info().Msg("Stopping sync manager...")

	// Stop Plex WebSocket if running (v1.39)
	if m.plexWSClient != nil {
		logging.Info().Msg("Stopping Plex WebSocket...")
		if err := m.plexWSClient.Close(); err != nil {
			logging.Error().Err(err).Msg("Failed to close Plex WebSocket")
		}
	}

	// Stop session poller if running (v1.50)
	if m.sessionPoller != nil {
		logging.Info().Msg("Stopping Plex session polling...")
		m.sessionPoller.Stop()
	}

	close(m.stopChan)
	m.wg.Wait()
	logging.Info().Msg("Sync manager stopped")

	return nil
}

// LastSyncTime returns the timestamp of the last successful sync
func (m *Manager) LastSyncTime() time.Time {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastSync
}

// TriggerSync manually triggers a synchronization
func (m *Manager) TriggerSync() error {
	// Prevent concurrent sync execution
	m.syncMu.Lock()
	defer m.syncMu.Unlock()

	return m.syncData()
}
