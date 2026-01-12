// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package supervisor provides Suture-based process supervision for Cartographus.
// This file contains service wrappers that adapt sync managers to the Suture interface.
//
// The wrappers implement suture.Service and provide lifecycle management for:
//   - Plex sync (via Manager)
//   - Jellyfin sync (via JellyfinManager)
//   - Emby sync (via EmbyManager)
//   - Tautulli sync (via Manager)
//
// Each wrapper:
//   - Creates the appropriate sync manager from UnifiedServerConfig
//   - Implements the Serve(context.Context) error method
//   - Handles graceful shutdown on context cancellation
package supervisor

import (
	"context"
	"time"

	"github.com/tomtom215/cartographus/internal/config"
	"github.com/tomtom215/cartographus/internal/logging"
	"github.com/tomtom215/cartographus/internal/models"
	"github.com/tomtom215/cartographus/internal/sync"
)

// SyncDBInterface defines the database interface needed by sync managers.
// This matches sync.DBInterface to allow type-safe dependencies.
type SyncDBInterface interface {
	SessionKeyExists(ctx context.Context, sessionKey string) (bool, error)
	GetGeolocation(ctx context.Context, ipAddress string) (*models.Geolocation, error)
	GetGeolocations(ctx context.Context, ipAddresses []string) (map[string]*models.Geolocation, error)
	UpsertGeolocation(geo *models.Geolocation) error
	InsertPlaybackEvent(event *models.PlaybackEvent) error
}

// SyncEventPublisher defines the interface for publishing events.
// This matches sync.EventPublisher.
type SyncEventPublisher interface {
	PublishPlaybackEvent(ctx context.Context, event *models.PlaybackEvent) error
}

// SyncWebSocketHub defines the interface for WebSocket broadcasting.
// This matches sync.WebSocketHub.
type SyncWebSocketHub interface {
	BroadcastJSON(messageType string, data interface{})
}

// SyncUserResolver defines the interface for user ID resolution.
// This matches sync.UserResolver.
type SyncUserResolver interface {
	ResolveUserID(ctx context.Context, source, serverID, externalUserID string, username, friendlyName *string) (int, error)
}

// PlexSyncServiceWrapper wraps the Plex sync manager as a Suture service.
type PlexSyncServiceWrapper struct {
	cfg          *config.UnifiedServerConfig
	db           SyncDBInterface
	publisher    SyncEventPublisher
	wsHub        SyncWebSocketHub
	userResolver SyncUserResolver
	manager      *sync.Manager
}

// NewPlexSyncServiceWrapper creates a new Plex sync service wrapper.
func NewPlexSyncServiceWrapper(
	cfg *config.UnifiedServerConfig,
	db ServerDB,
	publisher EventPublisher,
	wsHub WebSocketHub,
	userResolver UserResolver,
) *PlexSyncServiceWrapper {
	// Convert interfaces to the sync-compatible types using type assertions
	var syncDB SyncDBInterface
	if v, ok := db.(SyncDBInterface); ok {
		syncDB = v
	}
	var syncPub SyncEventPublisher
	if v, ok := publisher.(SyncEventPublisher); ok {
		syncPub = v
	}
	var syncWS SyncWebSocketHub
	if v, ok := wsHub.(SyncWebSocketHub); ok {
		syncWS = v
	}
	var syncResolver SyncUserResolver
	if v, ok := userResolver.(SyncUserResolver); ok {
		syncResolver = v
	}

	return &PlexSyncServiceWrapper{
		cfg:          cfg,
		db:           syncDB,
		publisher:    syncPub,
		wsHub:        syncWS,
		userResolver: syncResolver,
	}
}

// Serve implements suture.Service interface.
// Starts the Plex sync manager and blocks until context is canceled.
func (s *PlexSyncServiceWrapper) Serve(ctx context.Context) error {
	logging.Info().
		Str("server_id", s.cfg.ServerID).
		Str("name", s.cfg.Name).
		Msg("Starting Plex sync service")

	// Convert unified config to platform-specific config
	plexCfg := s.cfg.ToPlexConfig()
	if plexCfg == nil {
		logging.Error().Msg("Failed to convert config to Plex config")
		return nil
	}

	// Ensure valid sync interval (default 5 minutes if not set)
	if plexCfg.SyncInterval == 0 {
		plexCfg.SyncInterval = 5 * time.Minute
	}

	// Create a full config with just Plex enabled
	fullCfg := &config.Config{
		Plex: *plexCfg,
		Sync: config.SyncConfig{
			Interval:  5 * time.Minute,
			BatchSize: 100,
		},
	}

	// Create sync manager
	// Note: We pass nil for TautulliClient since this is a Plex-only service
	s.manager = sync.NewManager(s.db, s.userResolver, nil, fullCfg, s.wsHub)

	if s.publisher != nil {
		s.manager.SetEventPublisher(s.publisher)
	}

	// Start the manager
	if err := s.manager.Start(ctx); err != nil {
		logging.Error().Err(err).Msg("Failed to start Plex sync manager")
		return err
	}

	// Wait for context cancellation
	<-ctx.Done()

	// Stop the manager
	if err := s.manager.Stop(); err != nil {
		logging.Warn().Err(err).Msg("Error stopping Plex sync manager")
	}

	logging.Info().
		Str("server_id", s.cfg.ServerID).
		Msg("Plex sync service stopped")

	return nil
}

// JellyfinSyncServiceWrapper wraps the Jellyfin manager as a Suture service.
type JellyfinSyncServiceWrapper struct {
	cfg          *config.UnifiedServerConfig
	db           SyncDBInterface
	publisher    SyncEventPublisher
	wsHub        SyncWebSocketHub
	userResolver SyncUserResolver
	manager      *sync.JellyfinManager
}

// NewJellyfinSyncServiceWrapper creates a new Jellyfin sync service wrapper.
func NewJellyfinSyncServiceWrapper(
	cfg *config.UnifiedServerConfig,
	db ServerDB,
	publisher EventPublisher,
	wsHub WebSocketHub,
	userResolver UserResolver,
) *JellyfinSyncServiceWrapper {
	// Convert interfaces using type assertions
	var syncDB SyncDBInterface
	if v, ok := db.(SyncDBInterface); ok {
		syncDB = v
	}
	var syncPub SyncEventPublisher
	if v, ok := publisher.(SyncEventPublisher); ok {
		syncPub = v
	}
	var syncWS SyncWebSocketHub
	if v, ok := wsHub.(SyncWebSocketHub); ok {
		syncWS = v
	}
	var syncResolver SyncUserResolver
	if v, ok := userResolver.(SyncUserResolver); ok {
		syncResolver = v
	}

	return &JellyfinSyncServiceWrapper{
		cfg:          cfg,
		db:           syncDB,
		publisher:    syncPub,
		wsHub:        syncWS,
		userResolver: syncResolver,
	}
}

// Serve implements suture.Service interface.
func (s *JellyfinSyncServiceWrapper) Serve(ctx context.Context) error {
	logging.Info().
		Str("server_id", s.cfg.ServerID).
		Str("name", s.cfg.Name).
		Msg("Starting Jellyfin sync service")

	// Convert unified config to platform-specific config
	jellyCfg := s.cfg.ToJellyfinConfig()
	if jellyCfg == nil {
		logging.Error().Msg("Failed to convert config to Jellyfin config")
		return nil
	}

	// Create Jellyfin manager
	s.manager = sync.NewJellyfinManager(jellyCfg, s.wsHub, s.userResolver)
	if s.manager == nil {
		logging.Warn().Msg("Jellyfin manager not created (disabled)")
		return nil
	}

	if s.publisher != nil {
		s.manager.SetEventPublisher(s.publisher)
	}

	// Start the manager
	if err := s.manager.Start(ctx); err != nil {
		logging.Error().Err(err).Msg("Failed to start Jellyfin manager")
		return err
	}

	// Wait for context cancellation
	<-ctx.Done()

	// Stop the manager
	if err := s.manager.Stop(); err != nil {
		logging.Warn().Err(err).Msg("Error stopping Jellyfin manager")
	}

	logging.Info().
		Str("server_id", s.cfg.ServerID).
		Msg("Jellyfin sync service stopped")

	return nil
}

// EmbySyncServiceWrapper wraps the Emby manager as a Suture service.
type EmbySyncServiceWrapper struct {
	cfg          *config.UnifiedServerConfig
	db           SyncDBInterface
	publisher    SyncEventPublisher
	wsHub        SyncWebSocketHub
	userResolver SyncUserResolver
	manager      *sync.EmbyManager
}

// NewEmbySyncServiceWrapper creates a new Emby sync service wrapper.
func NewEmbySyncServiceWrapper(
	cfg *config.UnifiedServerConfig,
	db ServerDB,
	publisher EventPublisher,
	wsHub WebSocketHub,
	userResolver UserResolver,
) *EmbySyncServiceWrapper {
	// Convert interfaces using type assertions
	var syncDB SyncDBInterface
	if v, ok := db.(SyncDBInterface); ok {
		syncDB = v
	}
	var syncPub SyncEventPublisher
	if v, ok := publisher.(SyncEventPublisher); ok {
		syncPub = v
	}
	var syncWS SyncWebSocketHub
	if v, ok := wsHub.(SyncWebSocketHub); ok {
		syncWS = v
	}
	var syncResolver SyncUserResolver
	if v, ok := userResolver.(SyncUserResolver); ok {
		syncResolver = v
	}

	return &EmbySyncServiceWrapper{
		cfg:          cfg,
		db:           syncDB,
		publisher:    syncPub,
		wsHub:        syncWS,
		userResolver: syncResolver,
	}
}

// Serve implements suture.Service interface.
func (s *EmbySyncServiceWrapper) Serve(ctx context.Context) error {
	logging.Info().
		Str("server_id", s.cfg.ServerID).
		Str("name", s.cfg.Name).
		Msg("Starting Emby sync service")

	// Convert unified config to platform-specific config
	embyCfg := s.cfg.ToEmbyConfig()
	if embyCfg == nil {
		logging.Error().Msg("Failed to convert config to Emby config")
		return nil
	}

	// Create Emby manager
	s.manager = sync.NewEmbyManager(embyCfg, s.wsHub, s.userResolver)
	if s.manager == nil {
		logging.Warn().Msg("Emby manager not created (disabled)")
		return nil
	}

	if s.publisher != nil {
		s.manager.SetEventPublisher(s.publisher)
	}

	// Start the manager
	if err := s.manager.Start(ctx); err != nil {
		logging.Error().Err(err).Msg("Failed to start Emby manager")
		return err
	}

	// Wait for context cancellation
	<-ctx.Done()

	// Stop the manager
	if err := s.manager.Stop(); err != nil {
		logging.Warn().Err(err).Msg("Error stopping Emby manager")
	}

	logging.Info().
		Str("server_id", s.cfg.ServerID).
		Msg("Emby sync service stopped")

	return nil
}

// TautulliSyncServiceWrapper wraps the Tautulli sync as a Suture service.
type TautulliSyncServiceWrapper struct {
	cfg       *config.UnifiedServerConfig
	db        SyncDBInterface
	publisher SyncEventPublisher
	wsHub     SyncWebSocketHub
	manager   *sync.Manager
}

// NewTautulliSyncServiceWrapper creates a new Tautulli sync service wrapper.
func NewTautulliSyncServiceWrapper(
	cfg *config.UnifiedServerConfig,
	db ServerDB,
	publisher EventPublisher,
	wsHub WebSocketHub,
) *TautulliSyncServiceWrapper {
	// Convert interfaces using type assertions
	var syncDB SyncDBInterface
	if v, ok := db.(SyncDBInterface); ok {
		syncDB = v
	}
	var syncPub SyncEventPublisher
	if v, ok := publisher.(SyncEventPublisher); ok {
		syncPub = v
	}
	var syncWS SyncWebSocketHub
	if v, ok := wsHub.(SyncWebSocketHub); ok {
		syncWS = v
	}

	return &TautulliSyncServiceWrapper{
		cfg:       cfg,
		db:        syncDB,
		publisher: syncPub,
		wsHub:     syncWS,
	}
}

// Serve implements suture.Service interface.
func (s *TautulliSyncServiceWrapper) Serve(ctx context.Context) error {
	logging.Info().
		Str("server_id", s.cfg.ServerID).
		Str("name", s.cfg.Name).
		Msg("Starting Tautulli sync service")

	// Convert unified config to platform-specific config
	tautCfg := s.cfg.ToTautulliConfig()
	if tautCfg == nil {
		logging.Error().Msg("Failed to convert config to Tautulli config")
		return nil
	}

	// Create Tautulli client
	client := sync.NewTautulliClient(tautCfg)

	// Create a config for the manager with Tautulli enabled
	fullCfg := &config.Config{
		Tautulli: *tautCfg,
		Sync: config.SyncConfig{
			Interval:  5 * time.Minute,
			BatchSize: 100,
		},
	}

	// Create sync manager
	s.manager = sync.NewManager(s.db, nil, client, fullCfg, s.wsHub)

	if s.publisher != nil {
		s.manager.SetEventPublisher(s.publisher)
	}

	// Start the manager
	if err := s.manager.Start(ctx); err != nil {
		logging.Error().Err(err).Msg("Failed to start Tautulli sync manager")
		return err
	}

	// Wait for context cancellation
	<-ctx.Done()

	// Stop the manager
	if err := s.manager.Stop(); err != nil {
		logging.Warn().Err(err).Msg("Error stopping Tautulli sync manager")
	}

	logging.Info().
		Str("server_id", s.cfg.ServerID).
		Msg("Tautulli sync service stopped")

	return nil
}
