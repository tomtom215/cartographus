// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package detection

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/goccy/go-json"
)

// ConcurrentStreamsDetector enforces per-user stream limits.
// It flags users who exceed their allowed number of simultaneous streams.
type ConcurrentStreamsDetector struct {
	config       ConcurrentStreamsConfig
	eventHistory EventHistory
	enabled      bool
	mu           sync.RWMutex
}

// NewConcurrentStreamsDetector creates a new concurrent streams detector.
func NewConcurrentStreamsDetector(eventHistory EventHistory) *ConcurrentStreamsDetector {
	return &ConcurrentStreamsDetector{
		config:       DefaultConcurrentStreamsConfig(),
		eventHistory: eventHistory,
		enabled:      true,
	}
}

// Type returns the rule type.
func (d *ConcurrentStreamsDetector) Type() RuleType {
	return RuleTypeConcurrentStreams
}

// Check evaluates the event against the concurrent streams rule.
func (d *ConcurrentStreamsDetector) Check(ctx context.Context, event *DetectionEvent) (*Alert, error) {
	d.mu.RLock()
	if !d.enabled {
		d.mu.RUnlock()
		return nil, nil
	}
	config := d.config
	d.mu.RUnlock()

	// Only check on stream start events
	if event.EventType != "start" && event.EventType != "" {
		// Also check empty event type (for initial processing)
		// This handles both explicit "start" and undefined event types
		if event.EventType != "" {
			return nil, nil
		}
	}

	// Get active streams for this user on this server
	// v2.1: Pass serverID to scope detection to the same server instance
	activeStreams, err := d.eventHistory.GetActiveStreamsForUser(ctx, event.UserID, event.ServerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get active streams: %w", err)
	}

	// Get user's stream limit (per-user override or default)
	limit := config.DefaultLimit
	if userLimit, ok := config.UserLimits[event.UserID]; ok {
		limit = userLimit
	}

	// Count current active streams (including this new one)
	activeCount := len(activeStreams)

	// Check if limit is exceeded
	// Note: We check >= limit because the current event is a new stream starting
	if activeCount < limit {
		return nil, nil
	}

	// Collect session keys for metadata
	sessionKeys := make([]string, 0, len(activeStreams))
	for i := range activeStreams {
		sessionKeys = append(sessionKeys, activeStreams[i].SessionKey)
	}
	// Include current session
	if event.SessionKey != "" {
		sessionKeys = append(sessionKeys, event.SessionKey)
	}

	// Build metadata
	metadata := ConcurrentStreamsMetadata{
		ActiveStreams: activeCount + 1, // +1 for current event
		StreamLimit:   limit,
		SessionKeys:   sessionKeys,
	}

	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	alert := &Alert{
		RuleType:  RuleTypeConcurrentStreams,
		UserID:    event.UserID,
		Username:  event.Username,
		ServerID:  event.ServerID, // v2.1: Multi-server support
		MachineID: event.MachineID,
		IPAddress: event.IPAddress,
		Severity:  config.Severity,
		Title:     "Concurrent Stream Limit Exceeded",
		Message: fmt.Sprintf(
			"User %s has %d active streams (limit: %d)",
			event.Username,
			activeCount+1,
			limit,
		),
		Metadata:  metadataJSON,
		CreatedAt: time.Now(),
	}

	return alert, nil
}

// Configure updates the detector configuration.
func (d *ConcurrentStreamsDetector) Configure(config json.RawMessage) error {
	var newConfig ConcurrentStreamsConfig
	if err := json.Unmarshal(config, &newConfig); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Validate configuration
	if newConfig.DefaultLimit <= 0 {
		return fmt.Errorf("default_limit must be positive")
	}

	// Validate per-user limits
	for userID, limit := range newConfig.UserLimits {
		if limit <= 0 {
			return fmt.Errorf("user_limit for user %d must be positive", userID)
		}
	}

	d.mu.Lock()
	d.config = newConfig
	d.mu.Unlock()

	return nil
}

// Enabled returns whether this detector is enabled.
func (d *ConcurrentStreamsDetector) Enabled() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.enabled
}

// SetEnabled enables or disables the detector.
func (d *ConcurrentStreamsDetector) SetEnabled(enabled bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.enabled = enabled
}

// Config returns the current configuration.
func (d *ConcurrentStreamsDetector) Config() ConcurrentStreamsConfig {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.config
}

// SetUserLimit sets a specific stream limit for a user.
func (d *ConcurrentStreamsDetector) SetUserLimit(userID int, limit int) error {
	if limit <= 0 {
		return fmt.Errorf("limit must be positive")
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	if d.config.UserLimits == nil {
		d.config.UserLimits = make(map[int]int)
	}
	d.config.UserLimits[userID] = limit

	return nil
}

// RemoveUserLimit removes a user-specific limit, reverting to default.
func (d *ConcurrentStreamsDetector) RemoveUserLimit(userID int) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.config.UserLimits != nil {
		delete(d.config.UserLimits, userID)
	}
}

// GetUserLimit returns the effective limit for a user.
func (d *ConcurrentStreamsDetector) GetUserLimit(userID int) int {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if limit, ok := d.config.UserLimits[userID]; ok {
		return limit
	}
	return d.config.DefaultLimit
}
