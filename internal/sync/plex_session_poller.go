// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package sync

import (
	"context"
	"sync"
	"time"

	"github.com/tomtom215/cartographus/internal/logging"

	"github.com/tomtom215/cartographus/internal/cache"
)

// PlexSessionPoller periodically polls Plex for active sessions and publishes
// events to NATS for detection processing.
//
// Use Cases:
//   - Fallback when WebSocket is unavailable
//   - Periodic refresh of session state for detection rules
//   - Discovery of sessions that started before WebSocket connection
//
// Detection Integration (ADR-0020):
//   - All discovered sessions are published to NATS
//   - Detection engine processes events for anomaly detection
//
// Performance: Uses LRUCache for O(1) session tracking with automatic eviction.
type PlexSessionPoller struct {
	manager *Manager
	config  SessionPollerConfig

	// Runtime state
	mu       sync.RWMutex
	running  bool
	stopChan chan struct{}
	wg       sync.WaitGroup

	// LRU cache for session tracking - O(1) operations with automatic eviction
	seenSessions *cache.LRUCache
}

// SessionPollerConfig configures the session poller behavior.
type SessionPollerConfig struct {
	// Interval is how often to poll Plex for sessions.
	Interval time.Duration

	// PublishAll controls whether to publish all sessions or only new ones.
	// If true, publishes all sessions on each poll (for detection refresh).
	// If false, only publishes sessions not seen before.
	PublishAll bool

	// SeenSessionTTL is how long to remember seen sessions.
	SeenSessionTTL time.Duration
}

// DefaultSessionPollerConfig returns production defaults.
func DefaultSessionPollerConfig() SessionPollerConfig {
	return SessionPollerConfig{
		Interval:       30 * time.Second,
		PublishAll:     false, // Only new sessions by default
		SeenSessionTTL: 5 * time.Minute,
	}
}

// NewPlexSessionPoller creates a new session poller.
func NewPlexSessionPoller(manager *Manager, config SessionPollerConfig) *PlexSessionPoller {
	return &PlexSessionPoller{
		manager:  manager,
		config:   config,
		stopChan: make(chan struct{}),
		// LRU cache with 1000 capacity (typical max active sessions)
		seenSessions: cache.NewLRUCache(1000, config.SeenSessionTTL),
	}
}

// Start begins the polling loop.
func (p *PlexSessionPoller) Start(ctx context.Context) error {
	p.mu.Lock()
	if p.running {
		p.mu.Unlock()
		return nil
	}
	p.running = true
	p.stopChan = make(chan struct{})
	p.mu.Unlock()

	logging.Info().Dur("interval", p.config.Interval).Msg("Starting session poller")

	p.wg.Add(1)
	go p.pollLoop(ctx)

	return nil
}

// Serve implements suture.Service for supervisor integration.
func (p *PlexSessionPoller) Serve(ctx context.Context) error {
	if err := p.Start(ctx); err != nil {
		return err
	}

	// Wait for context cancellation
	<-ctx.Done()

	// Stop the poller
	p.Stop()

	return ctx.Err()
}

// Stop gracefully stops the polling loop.
func (p *PlexSessionPoller) Stop() {
	p.mu.Lock()
	if !p.running {
		p.mu.Unlock()
		return
	}
	p.running = false
	close(p.stopChan)
	p.mu.Unlock()

	p.wg.Wait()
	logging.Info().Msg("[plex-poller] Session poller stopped")
}

// IsRunning returns whether the poller is active.
func (p *PlexSessionPoller) IsRunning() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.running
}

// pollLoop is the main polling loop.
func (p *PlexSessionPoller) pollLoop(ctx context.Context) {
	defer p.wg.Done()

	// Do an initial poll immediately
	p.poll(ctx)

	ticker := time.NewTicker(p.config.Interval)
	defer ticker.Stop()

	cleanupTicker := time.NewTicker(p.config.SeenSessionTTL / 2)
	defer cleanupTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			logging.Info().Msg("[plex-poller] Context canceled, stopping")
			return
		case <-p.stopChan:
			logging.Info().Msg("[plex-poller] Stop signal received")
			return
		case <-ticker.C:
			p.poll(ctx)
		case <-cleanupTicker.C:
			p.cleanupSeenSessions()
		}
	}
}

// poll fetches sessions from Plex and publishes to NATS.
func (p *PlexSessionPoller) poll(ctx context.Context) {
	if p.manager == nil || p.manager.plexClient == nil {
		return
	}

	// Fetch active sessions from Plex
	sessions, err := p.manager.plexClient.GetTranscodeSessions(ctx)
	if err != nil {
		logging.Info().Err(err).Msg("Failed to fetch sessions")
		return
	}

	if len(sessions) == 0 {
		return
	}

	logging.Info().Int("count", len(sessions)).Msg("Found active sessions")

	// Process each session
	for i := range sessions {
		session := &sessions[i]

		// Skip if we've seen this session recently (unless PublishAll is true)
		// Use atomic IsDuplicate to prevent race conditions in concurrent polling
		if !p.config.PublishAll && p.seenSessions.IsDuplicate(session.SessionKey) {
			continue
		}

		// Convert to PlaybackEvent
		event := p.manager.plexSessionToPlaybackEvent(ctx, session)
		if event == nil {
			continue
		}

		// Publish to NATS for detection
		p.manager.publishEvent(ctx, event)

		// For PublishAll mode, still mark as seen (IsDuplicate already marked in non-PublishAll mode)
		if p.config.PublishAll {
			p.seenSessions.Add(session.SessionKey, time.Now())
		}

		logging.Info().Str("session", session.SessionKey).Str("user", session.User.Title).Msg("Published session for user")
	}
}

// hasSeenSession checks if a session has been seen recently.
// Uses LRUCache.Contains for O(1) lookup.
func (p *PlexSessionPoller) hasSeenSession(sessionKey string) bool {
	return p.seenSessions.Contains(sessionKey)
}

// markSessionSeen records that a session has been published.
// Uses LRUCache.Add for O(1) insertion with automatic LRU eviction.
func (p *PlexSessionPoller) markSessionSeen(sessionKey string) {
	p.seenSessions.Add(sessionKey, time.Now())
}

// cleanupSeenSessions removes expired session entries.
// LRUCache handles expiration automatically, but this provides explicit cleanup.
func (p *PlexSessionPoller) cleanupSeenSessions() {
	p.seenSessions.CleanupExpired()
}

// Stats returns current poller statistics.
func (p *PlexSessionPoller) Stats() SessionPollerStats {
	p.mu.RLock()
	running := p.running
	p.mu.RUnlock()

	return SessionPollerStats{
		Running:         running,
		TrackedSessions: p.seenSessions.Len(),
		PollInterval:    p.config.Interval,
		SeenSessionTTL:  p.config.SeenSessionTTL,
	}
}

// SessionPollerStats holds runtime statistics.
type SessionPollerStats struct {
	Running         bool
	TrackedSessions int
	PollInterval    time.Duration
	SeenSessionTTL  time.Duration
}
