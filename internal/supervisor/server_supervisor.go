// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package supervisor provides Suture-based process supervision for Cartographus.
// This file implements the ServerSupervisor for dynamic media server management.
//
// Architecture (ADR-0026):
//   - ServerSupervisor manages sync services for all configured media servers
//   - Services can be dynamically added, removed, and updated at runtime
//   - Each server gets its own Suture-supervised service for fault isolation
//   - Integrates with ServerConfigManager for unified configuration
//
// Example Usage:
//
//	supervisor, err := NewServerSupervisor(tree, configMgr, db, publisher, wsHub)
//	if err != nil {
//	    log.Fatal("Failed to create server supervisor:", err)
//	}
//
//	// Start all configured servers
//	if err := supervisor.StartAll(ctx); err != nil {
//	    log.Error().Err(err).Msg("Some servers failed to start")
//	}
//
//	// Add a new server dynamically
//	if err := supervisor.AddServer(ctx, config); err != nil {
//	    log.Error().Err(err).Msg("Failed to add server")
//	}
package supervisor

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/thejerf/suture/v4"
	"github.com/tomtom215/cartographus/internal/config"
	"github.com/tomtom215/cartographus/internal/logging"
)

// Errors for ServerSupervisor
var (
	ErrServerAlreadyExists = errors.New("server already exists in supervisor")
	ErrServerNotRunning    = errors.New("server is not running")
	ErrInvalidPlatform     = errors.New("invalid or unsupported platform")
	ErrNilSupervisorTree   = errors.New("supervisor tree cannot be nil")
	ErrNilConfigManager    = errors.New("config manager cannot be nil")
)

// ServerStatus represents the current status of a managed server.
type ServerStatus struct {
	ServerID       string     `json:"server_id"`
	Platform       string     `json:"platform"`
	Name           string     `json:"name"`
	Running        bool       `json:"running"`
	Status         string     `json:"status"` // connected, syncing, error, disabled
	LastSyncAt     *time.Time `json:"last_sync_at,omitempty"`
	LastSyncStatus string     `json:"last_sync_status,omitempty"`
	LastError      string     `json:"last_error,omitempty"`
	LastErrorAt    *time.Time `json:"last_error_at,omitempty"`
	StartedAt      *time.Time `json:"started_at,omitempty"`
}

// managedService holds metadata about a running service.
type managedService struct {
	token     suture.ServiceToken
	config    *config.UnifiedServerConfig
	service   suture.Service
	startedAt time.Time
}

// ServerSupervisor manages sync services for all configured media servers.
// It provides dynamic service lifecycle management with Suture supervision.
//
// Thread Safety:
//   - All operations are protected by a read-write mutex
//   - Services map is safe for concurrent access
//   - Individual services handle their own internal concurrency
type ServerSupervisor struct {
	tree      *SupervisorTree
	configMgr *config.ServerConfigManager
	services  map[string]*managedService // serverID -> managed service
	mu        sync.RWMutex

	// Dependencies for service creation
	db             ServerDB
	eventPublisher EventPublisher
	wsHub          WebSocketHub
	userResolver   UserResolver
}

// ServerDB defines the database interface needed by sync services.
type ServerDB interface {
	// Core interface - add methods as needed by sync services
}

// EventPublisher defines the interface for publishing events to NATS.
type EventPublisher interface {
	// Add methods as needed
}

// WebSocketHub defines the interface for broadcasting to WebSocket clients.
type WebSocketHub interface {
	BroadcastJSON(messageType string, data interface{})
}

// UserResolver resolves external user IDs to internal IDs.
type UserResolver interface {
	ResolveUserID(ctx context.Context, source, serverID, externalUserID string, username, friendlyName *string) (int, error)
}

// ServerSupervisorConfig holds configuration for the server supervisor.
type ServerSupervisorConfig struct {
	// StartupTimeout is the maximum time to wait for all servers to start.
	StartupTimeout time.Duration

	// ShutdownTimeout is the maximum time to wait for graceful shutdown.
	ShutdownTimeout time.Duration
}

// DefaultServerSupervisorConfig returns sensible defaults.
func DefaultServerSupervisorConfig() ServerSupervisorConfig {
	return ServerSupervisorConfig{
		StartupTimeout:  30 * time.Second,
		ShutdownTimeout: 10 * time.Second,
	}
}

// NewServerSupervisor creates a new server supervisor.
//
// Parameters:
//   - tree: The Suture supervisor tree to add services to
//   - configMgr: Configuration manager for loading server configs
//   - db: Database interface for sync services
//   - publisher: Event publisher for NATS integration
//   - wsHub: WebSocket hub for real-time updates
//   - userResolver: User ID resolver for cross-source tracking
//
// The tree and configMgr are required; other dependencies are optional.
func NewServerSupervisor(
	tree *SupervisorTree,
	configMgr *config.ServerConfigManager,
	db ServerDB,
	publisher EventPublisher,
	wsHub WebSocketHub,
	userResolver UserResolver,
) (*ServerSupervisor, error) {
	if tree == nil {
		return nil, ErrNilSupervisorTree
	}
	if configMgr == nil {
		return nil, ErrNilConfigManager
	}

	return &ServerSupervisor{
		tree:           tree,
		configMgr:      configMgr,
		services:       make(map[string]*managedService),
		db:             db,
		eventPublisher: publisher,
		wsHub:          wsHub,
		userResolver:   userResolver,
	}, nil
}

// StartAll starts sync services for all enabled servers from configuration.
// This should be called during application startup.
//
// Returns an error if loading configuration fails. Individual server failures
// are logged but don't prevent other servers from starting.
func (s *ServerSupervisor) StartAll(ctx context.Context) error {
	servers, err := s.configMgr.GetEnabledServers(ctx)
	if err != nil {
		return fmt.Errorf("failed to load server configurations: %w", err)
	}

	logging.Info().Int("count", len(servers)).Msg("Starting sync services for configured servers")

	var startErrors []error
	for i := range servers {
		cfg := &servers[i]
		if err := s.AddServer(ctx, cfg); err != nil {
			logging.Warn().
				Str("server_id", cfg.ServerID).
				Str("platform", cfg.Platform).
				Err(err).
				Msg("Failed to start server sync service")
			startErrors = append(startErrors, err)
		}
	}

	if len(startErrors) > 0 {
		return fmt.Errorf("failed to start %d servers", len(startErrors))
	}

	logging.Info().Int("count", len(servers)).Msg("All server sync services started")
	return nil
}

// AddServer adds a new server to the supervisor and starts its sync service.
//
// If a server with the same ID already exists, returns ErrServerAlreadyExists.
// The service is automatically restarted by Suture if it crashes.
func (s *ServerSupervisor) AddServer(ctx context.Context, cfg *config.UnifiedServerConfig) error {
	if cfg == nil {
		return errors.New("server configuration cannot be nil")
	}

	serverID := s.getServerKey(cfg)

	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if already running
	if _, exists := s.services[serverID]; exists {
		return ErrServerAlreadyExists
	}

	// Create the appropriate sync service
	svc, err := s.createService(cfg)
	if err != nil {
		return fmt.Errorf("failed to create sync service: %w", err)
	}

	// Add to supervisor tree (messaging layer)
	token := s.tree.AddMessagingService(svc)

	// Track the service
	now := time.Now()
	s.services[serverID] = &managedService{
		token:     token,
		config:    cfg,
		service:   svc,
		startedAt: now,
	}

	logging.Info().
		Str("server_id", serverID).
		Str("platform", cfg.Platform).
		Str("name", cfg.Name).
		Msg("Server sync service added to supervisor")

	return nil
}

// RemoveServer stops and removes a server's sync service.
//
// Returns ErrServerNotRunning if the server is not currently managed.
// The removal is graceful - Suture waits for the service to stop.
func (s *ServerSupervisor) RemoveServer(ctx context.Context, serverID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	managed, exists := s.services[serverID]
	if !exists {
		return ErrServerNotRunning
	}

	// Remove from messaging layer supervisor (triggers graceful shutdown)
	if err := s.tree.RemoveMessagingService(managed.token); err != nil {
		return fmt.Errorf("failed to remove service from supervisor: %w", err)
	}

	delete(s.services, serverID)

	logging.Info().
		Str("server_id", serverID).
		Str("platform", managed.config.Platform).
		Msg("Server sync service removed from supervisor")

	return nil
}

// UpdateServer updates a server's configuration by stopping the old service
// and starting a new one with the updated configuration.
//
// This is a stop-then-start operation, so there may be a brief period where
// the server is not syncing.
func (s *ServerSupervisor) UpdateServer(ctx context.Context, cfg *config.UnifiedServerConfig) error {
	if cfg == nil {
		return errors.New("server configuration cannot be nil")
	}

	serverID := s.getServerKey(cfg)

	// Check if exists
	s.mu.RLock()
	_, exists := s.services[serverID]
	s.mu.RUnlock()

	if !exists {
		// Server doesn't exist, just add it
		return s.AddServer(ctx, cfg)
	}

	// Remove old service
	if err := s.RemoveServer(ctx, serverID); err != nil {
		return fmt.Errorf("failed to remove old service: %w", err)
	}

	// Add new service with updated config
	if err := s.AddServer(ctx, cfg); err != nil {
		return fmt.Errorf("failed to add updated service: %w", err)
	}

	logging.Info().
		Str("server_id", serverID).
		Str("platform", cfg.Platform).
		Msg("Server sync service updated")

	return nil
}

// GetServerStatus returns the current status of a managed server.
func (s *ServerSupervisor) GetServerStatus(serverID string) (*ServerStatus, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	managed, exists := s.services[serverID]
	if !exists {
		return nil, ErrServerNotRunning
	}

	status := &ServerStatus{
		ServerID:       managed.config.ServerID,
		Platform:       managed.config.Platform,
		Name:           managed.config.Name,
		Running:        true,
		Status:         managed.config.Status,
		LastSyncAt:     managed.config.LastSyncAt,
		LastSyncStatus: managed.config.LastSyncStatus,
		LastError:      managed.config.LastError,
		LastErrorAt:    managed.config.LastErrorAt,
		StartedAt:      &managed.startedAt,
	}

	return status, nil
}

// GetAllServerStatuses returns status for all managed servers.
func (s *ServerSupervisor) GetAllServerStatuses() []ServerStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	statuses := make([]ServerStatus, 0, len(s.services))
	for _, managed := range s.services {
		status := ServerStatus{
			ServerID:       managed.config.ServerID,
			Platform:       managed.config.Platform,
			Name:           managed.config.Name,
			Running:        true,
			Status:         managed.config.Status,
			LastSyncAt:     managed.config.LastSyncAt,
			LastSyncStatus: managed.config.LastSyncStatus,
			LastError:      managed.config.LastError,
			LastErrorAt:    managed.config.LastErrorAt,
			StartedAt:      &managed.startedAt,
		}
		statuses = append(statuses, status)
	}

	return statuses
}

// IsServerRunning checks if a server's sync service is currently running.
func (s *ServerSupervisor) IsServerRunning(serverID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, exists := s.services[serverID]
	return exists
}

// StopAll stops all managed server sync services.
// This should be called during application shutdown.
func (s *ServerSupervisor) StopAll(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var stopErrors []error
	for serverID, managed := range s.services {
		if err := s.tree.RemoveMessagingService(managed.token); err != nil {
			logging.Warn().
				Str("server_id", serverID).
				Err(err).
				Msg("Failed to stop server sync service")
			stopErrors = append(stopErrors, err)
		}
	}

	// Clear the services map
	s.services = make(map[string]*managedService)

	if len(stopErrors) > 0 {
		return fmt.Errorf("failed to stop %d servers", len(stopErrors))
	}

	logging.Info().Msg("All server sync services stopped")
	return nil
}

// getServerKey returns the unique key for a server in the services map.
// Uses the server ID if available, otherwise falls back to the config ID.
func (s *ServerSupervisor) getServerKey(cfg *config.UnifiedServerConfig) string {
	if cfg.ServerID != "" {
		return cfg.ServerID
	}
	return cfg.ID
}

// createService creates the appropriate sync service for a platform.
// This is a factory method that creates platform-specific service wrappers.
func (s *ServerSupervisor) createService(cfg *config.UnifiedServerConfig) (suture.Service, error) {
	switch cfg.Platform {
	case "plex":
		return NewPlexSyncServiceWrapper(cfg, s.db, s.eventPublisher, s.wsHub, s.userResolver), nil
	case "jellyfin":
		return NewJellyfinSyncServiceWrapper(cfg, s.db, s.eventPublisher, s.wsHub, s.userResolver), nil
	case "emby":
		return NewEmbySyncServiceWrapper(cfg, s.db, s.eventPublisher, s.wsHub, s.userResolver), nil
	case "tautulli":
		return NewTautulliSyncServiceWrapper(cfg, s.db, s.eventPublisher, s.wsHub), nil
	default:
		return nil, fmt.Errorf("%w: %s", ErrInvalidPlatform, cfg.Platform)
	}
}
