// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

/*
emby_session_poller.go - Emby Session Poller

This file implements a backup session polling mechanism for Emby.
It periodically fetches active sessions from the Emby API and publishes
new sessions to NATS for event processing.

Why this exists:
- WebSocket may be unreliable in some environments
- Provides redundancy for mission-critical deployments
- Useful for debugging and testing
*/

package sync

import (
	"context"
	"sync"
	"time"

	"github.com/tomtom215/cartographus/internal/logging"

	"github.com/tomtom215/cartographus/internal/cache"
	"github.com/tomtom215/cartographus/internal/models"
)

// EmbySessionPoller periodically polls Emby for active sessions
//
// Performance: Uses LRUCache for O(1) session tracking with automatic eviction.
type EmbySessionPoller struct {
	client EmbyClientInterface
	config SessionPollerConfig

	mu       sync.RWMutex
	running  bool
	stopChan chan struct{}
	wg       sync.WaitGroup

	// LRU cache for session tracking - O(1) operations with automatic eviction
	seenSessions *cache.LRUCache

	// Callbacks
	onSession func(*models.EmbySession)
}

// NewEmbySessionPoller creates a new Emby session poller
func NewEmbySessionPoller(client EmbyClientInterface, config SessionPollerConfig) *EmbySessionPoller {
	return &EmbySessionPoller{
		client: client,
		config: config,
		// LRU cache with 1000 capacity (typical max active sessions)
		seenSessions: cache.NewLRUCache(1000, config.SeenSessionTTL),
	}
}

// SetOnSession sets the callback for new sessions
func (p *EmbySessionPoller) SetOnSession(callback func(*models.EmbySession)) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.onSession = callback
}

// Start begins the polling loop
func (p *EmbySessionPoller) Start(ctx context.Context) error {
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

// Stop stops the polling loop
func (p *EmbySessionPoller) Stop() {
	p.mu.Lock()
	if !p.running {
		p.mu.Unlock()
		return
	}
	p.running = false
	close(p.stopChan)
	p.mu.Unlock()

	p.wg.Wait()
	logging.Info().Msg("[emby-poller] Session poller stopped")
}

// pollLoop runs the periodic polling
func (p *EmbySessionPoller) pollLoop(ctx context.Context) {
	defer p.wg.Done()

	// Initial poll
	p.poll(ctx)

	ticker := time.NewTicker(p.config.Interval)
	defer ticker.Stop()

	cleanupTicker := time.NewTicker(p.config.SeenSessionTTL / 2)
	defer cleanupTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-p.stopChan:
			return
		case <-ticker.C:
			p.poll(ctx)
		case <-cleanupTicker.C:
			p.cleanupSeenSessions()
		}
	}
}

// poll fetches sessions and processes new ones
func (p *EmbySessionPoller) poll(ctx context.Context) {
	sessions, err := p.client.GetActiveSessions(ctx)
	if err != nil {
		logging.Info().Err(err).Msg("Failed to fetch sessions")
		return
	}

	p.mu.RLock()
	callback := p.onSession
	p.mu.RUnlock()

	for i := range sessions {
		session := &sessions[i]

		// Skip if we've seen this session recently (unless PublishAll is true)
		// Use atomic IsDuplicate to prevent race conditions in concurrent polling
		if !p.config.PublishAll && p.seenSessions.IsDuplicate(session.ID) {
			continue
		}

		if callback != nil {
			callback(session)
		}

		// For PublishAll mode, still mark as seen (IsDuplicate already marked in non-PublishAll mode)
		if p.config.PublishAll {
			p.seenSessions.Add(session.ID, time.Now())
		}
	}
}

// hasSeenSession checks if a session was recently processed
// Uses LRUCache.Contains for O(1) lookup.
func (p *EmbySessionPoller) hasSeenSession(sessionID string) bool {
	return p.seenSessions.Contains(sessionID)
}

// markSessionSeen records that a session was processed
// Uses LRUCache.Add for O(1) insertion with automatic LRU eviction.
func (p *EmbySessionPoller) markSessionSeen(sessionID string) {
	p.seenSessions.Add(sessionID, time.Now())
}

// cleanupSeenSessions removes expired entries from the seen sessions map
// LRUCache handles expiration automatically, but this provides explicit cleanup.
func (p *EmbySessionPoller) cleanupSeenSessions() {
	p.seenSessions.CleanupExpired()
}
