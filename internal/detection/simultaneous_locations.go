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

// SimultaneousLocationsMetadata contains details for simultaneous locations alerts.
type SimultaneousLocationsMetadata struct {
	Locations []LocationInfo `json:"locations"`
}

// LocationInfo represents a streaming location.
type LocationInfo struct {
	SessionKey string    `json:"session_key"`
	IPAddress  string    `json:"ip_address"`
	City       string    `json:"city,omitempty"`
	Country    string    `json:"country,omitempty"`
	Latitude   float64   `json:"latitude"`
	Longitude  float64   `json:"longitude"`
	Title      string    `json:"title,omitempty"`
	StartedAt  time.Time `json:"started_at"`
}

// SimultaneousLocationsDetector flags same account streaming from multiple cities.
// Unlike impossible travel (which looks at sequential events), this looks at
// concurrent active streams from geographically distant locations.
type SimultaneousLocationsDetector struct {
	config       SimultaneousLocationsConfig
	eventHistory EventHistory
	enabled      bool
	mu           sync.RWMutex
}

// NewSimultaneousLocationsDetector creates a new simultaneous locations detector.
func NewSimultaneousLocationsDetector(eventHistory EventHistory) *SimultaneousLocationsDetector {
	return &SimultaneousLocationsDetector{
		config:       DefaultSimultaneousLocationsConfig(),
		eventHistory: eventHistory,
		enabled:      true,
	}
}

// Type returns the rule type.
func (d *SimultaneousLocationsDetector) Type() RuleType {
	return RuleTypeSimultaneousLocations
}

// Check evaluates the event against the simultaneous locations rule.
func (d *SimultaneousLocationsDetector) Check(ctx context.Context, event *DetectionEvent) (*Alert, error) {
	d.mu.RLock()
	if !d.enabled {
		d.mu.RUnlock()
		return nil, nil
	}
	config := d.config
	d.mu.RUnlock()

	// Skip if no geolocation data
	// DETERMINISM: Use epsilon-based coordinate check instead of direct float equality
	if IsUnknownLocation(event.Latitude, event.Longitude) {
		return nil, nil
	}

	// Get concurrent sessions at different locations on this server
	// v2.1: Pass serverID to scope detection to the same server instance
	window := time.Duration(config.WindowMinutes) * time.Minute
	concurrentEvents, err := d.eventHistory.GetSimultaneousLocations(ctx, event.UserID, event.ServerID, window)
	if err != nil {
		return nil, fmt.Errorf("failed to get simultaneous locations: %w", err)
	}

	if len(concurrentEvents) == 0 {
		return nil, nil
	}

	// Check if any concurrent event is from a different location
	var distantLocations []LocationInfo

	// Add current event to locations
	currentLocation := LocationInfo{
		SessionKey: event.SessionKey,
		IPAddress:  event.IPAddress,
		City:       event.City,
		Country:    event.Country,
		Latitude:   event.Latitude,
		Longitude:  event.Longitude,
		Title:      event.Title,
		StartedAt:  event.Timestamp,
	}

	for i := range concurrentEvents {
		concurrent := &concurrentEvents[i]
		// Skip same session
		if concurrent.SessionKey == event.SessionKey {
			continue
		}

		// Skip if no geolocation
		// DETERMINISM: Use epsilon-based coordinate check instead of direct float equality
		if IsUnknownLocation(concurrent.Latitude, concurrent.Longitude) {
			continue
		}

		// Calculate distance
		distance := haversineDistance(
			event.Latitude, event.Longitude,
			concurrent.Latitude, concurrent.Longitude,
		)

		// Check if distance exceeds threshold
		if distance >= config.MinDistanceKm {
			distantLocations = append(distantLocations, LocationInfo{
				SessionKey: concurrent.SessionKey,
				IPAddress:  concurrent.IPAddress,
				City:       concurrent.City,
				Country:    concurrent.Country,
				Latitude:   concurrent.Latitude,
				Longitude:  concurrent.Longitude,
				Title:      concurrent.Title,
				StartedAt:  concurrent.Timestamp,
			})
		}
	}

	if len(distantLocations) == 0 {
		return nil, nil
	}

	// Include current location in metadata
	allLocations := append([]LocationInfo{currentLocation}, distantLocations...)

	// Build metadata
	metadata := SimultaneousLocationsMetadata{
		Locations: allLocations,
	}

	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Format location summary
	locationSummary := formatLocationSummary(allLocations)

	alert := &Alert{
		RuleType:  RuleTypeSimultaneousLocations,
		UserID:    event.UserID,
		Username:  event.Username,
		ServerID:  event.ServerID, // v2.1: Multi-server support
		MachineID: event.MachineID,
		IPAddress: event.IPAddress,
		Severity:  config.Severity,
		Title:     "Simultaneous Locations Detected",
		Message: fmt.Sprintf(
			"User %s is streaming from %d locations simultaneously: %s",
			event.Username,
			len(allLocations),
			locationSummary,
		),
		Metadata:  metadataJSON,
		CreatedAt: time.Now(),
	}

	return alert, nil
}

// Configure updates the detector configuration.
func (d *SimultaneousLocationsDetector) Configure(config json.RawMessage) error {
	var newConfig SimultaneousLocationsConfig
	if err := json.Unmarshal(config, &newConfig); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Validate configuration
	if newConfig.WindowMinutes <= 0 {
		return fmt.Errorf("window_minutes must be positive")
	}
	if newConfig.MinDistanceKm < 0 {
		return fmt.Errorf("min_distance_km cannot be negative")
	}

	d.mu.Lock()
	d.config = newConfig
	d.mu.Unlock()

	return nil
}

// Enabled returns whether this detector is enabled.
func (d *SimultaneousLocationsDetector) Enabled() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.enabled
}

// SetEnabled enables or disables the detector.
func (d *SimultaneousLocationsDetector) SetEnabled(enabled bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.enabled = enabled
}

// Config returns the current configuration.
func (d *SimultaneousLocationsDetector) Config() SimultaneousLocationsConfig {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.config
}

// formatLocationSummary creates a human-readable list of locations.
func formatLocationSummary(locations []LocationInfo) string {
	if len(locations) == 0 {
		return ""
	}

	var result string
	for i, loc := range locations {
		if i > 0 {
			result += ", "
		}
		result += formatLocation(loc.City, loc.Country)
	}
	return result
}
