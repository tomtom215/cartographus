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

// DeviceVelocityDetector flags devices appearing from multiple IPs rapidly.
// This can indicate VPN hopping, account sharing, or automated tooling.
type DeviceVelocityDetector struct {
	config       DeviceVelocityConfig
	eventHistory EventHistory
	enabled      bool
	mu           sync.RWMutex
}

// NewDeviceVelocityDetector creates a new device velocity detector.
func NewDeviceVelocityDetector(eventHistory EventHistory) *DeviceVelocityDetector {
	return &DeviceVelocityDetector{
		config:       DefaultDeviceVelocityConfig(),
		eventHistory: eventHistory,
		enabled:      true,
	}
}

// Type returns the rule type.
func (d *DeviceVelocityDetector) Type() RuleType {
	return RuleTypeDeviceVelocity
}

// Check evaluates the event against the device velocity rule.
func (d *DeviceVelocityDetector) Check(ctx context.Context, event *DetectionEvent) (*Alert, error) {
	d.mu.RLock()
	if !d.enabled {
		d.mu.RUnlock()
		return nil, nil
	}
	config := d.config
	d.mu.RUnlock()

	// Skip if no machine ID (can't track device)
	if event.MachineID == "" {
		return nil, nil
	}

	// Get recent IPs for this device on this server
	// v2.1: Pass serverID to scope detection to the same server instance
	window := time.Duration(config.WindowMinutes) * time.Minute
	recentIPs, err := d.eventHistory.GetRecentIPsForDevice(ctx, event.MachineID, event.ServerID, window)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent IPs: %w", err)
	}

	// Add current IP if not in list
	ipSet := make(map[string]bool)
	for _, ip := range recentIPs {
		ipSet[ip] = true
	}
	ipSet[event.IPAddress] = true

	// Count unique IPs
	uniqueIPCount := len(ipSet)

	// Check if threshold is exceeded
	if uniqueIPCount <= config.MaxUniqueIPs {
		return nil, nil
	}

	// Convert IP set to list for metadata
	ipList := make([]string, 0, len(ipSet))
	for ip := range ipSet {
		ipList = append(ipList, ip)
	}

	// Build metadata
	metadata := DeviceVelocityMetadata{
		MachineID:   event.MachineID,
		IPAddresses: ipList,
		WindowStart: event.Timestamp.Add(-window),
		WindowEnd:   event.Timestamp,
	}

	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	alert := &Alert{
		RuleType:  RuleTypeDeviceVelocity,
		UserID:    event.UserID,
		Username:  event.Username,
		ServerID:  event.ServerID, // v2.1: Multi-server support
		MachineID: event.MachineID,
		IPAddress: event.IPAddress,
		Severity:  config.Severity,
		Title:     "Device IP Velocity Alert",
		Message: fmt.Sprintf(
			"Device %s used %d unique IPs in %d minutes (limit: %d)",
			truncateMachineID(event.MachineID),
			uniqueIPCount,
			config.WindowMinutes,
			config.MaxUniqueIPs,
		),
		Metadata:  metadataJSON,
		CreatedAt: time.Now(),
	}

	return alert, nil
}

// Configure updates the detector configuration.
func (d *DeviceVelocityDetector) Configure(config json.RawMessage) error {
	var newConfig DeviceVelocityConfig
	if err := json.Unmarshal(config, &newConfig); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Validate configuration
	if newConfig.WindowMinutes <= 0 {
		return fmt.Errorf("window_minutes must be positive")
	}
	if newConfig.MaxUniqueIPs <= 0 {
		return fmt.Errorf("max_unique_ips must be positive")
	}

	d.mu.Lock()
	d.config = newConfig
	d.mu.Unlock()

	return nil
}

// Enabled returns whether this detector is enabled.
func (d *DeviceVelocityDetector) Enabled() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.enabled
}

// SetEnabled enables or disables the detector.
func (d *DeviceVelocityDetector) SetEnabled(enabled bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.enabled = enabled
}

// Config returns the current configuration.
func (d *DeviceVelocityDetector) Config() DeviceVelocityConfig {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.config
}

// truncateMachineID returns a shortened machine ID for display.
func truncateMachineID(machineID string) string {
	if len(machineID) <= 12 {
		return machineID
	}
	return machineID[:8] + "..."
}
