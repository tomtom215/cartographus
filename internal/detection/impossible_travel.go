// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package detection

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/goccy/go-json"
)

// ImpossibleTravelDetector detects implausible geographic transitions.
// It flags users who appear to stream from locations that are too far apart
// given the time between events (e.g., NYC to London in 30 minutes).
type ImpossibleTravelDetector struct {
	config       ImpossibleTravelConfig
	eventHistory EventHistory
	enabled      bool
	mu           sync.RWMutex
}

// NewImpossibleTravelDetector creates a new impossible travel detector.
func NewImpossibleTravelDetector(eventHistory EventHistory) *ImpossibleTravelDetector {
	return &ImpossibleTravelDetector{
		config:       DefaultImpossibleTravelConfig(),
		eventHistory: eventHistory,
		enabled:      true,
	}
}

// Type returns the rule type.
func (d *ImpossibleTravelDetector) Type() RuleType {
	return RuleTypeImpossibleTravel
}

// Check evaluates the event against the impossible travel rule.
func (d *ImpossibleTravelDetector) Check(ctx context.Context, event *DetectionEvent) (*Alert, error) {
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

	// Get the user's last event on this server
	// v2.1: Pass serverID to scope detection to the same server instance
	lastEvent, err := d.eventHistory.GetLastEventForUser(ctx, event.UserID, event.ServerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get last event: %w", err)
	}
	if lastEvent == nil {
		// No previous event, can't detect impossible travel
		return nil, nil
	}

	// Skip if last event has no geolocation
	// DETERMINISM: Use epsilon-based coordinate check instead of direct float equality
	if IsUnknownLocation(lastEvent.Latitude, lastEvent.Longitude) {
		return nil, nil
	}

	// Calculate time delta
	timeDelta := event.Timestamp.Sub(lastEvent.Timestamp)
	if timeDelta < 0 {
		// Event is older than last event (out of order), skip
		return nil, nil
	}

	// Skip if time delta is too small (likely same session or duplicate)
	minTimeDelta := time.Duration(config.MinTimeDeltaMinutes) * time.Minute
	if timeDelta < minTimeDelta {
		return nil, nil
	}

	// Calculate distance using Haversine formula
	distanceKm := haversineDistance(
		lastEvent.Latitude, lastEvent.Longitude,
		event.Latitude, event.Longitude,
	)

	// Skip if distance is below threshold
	if distanceKm < config.MinDistanceKm {
		return nil, nil
	}

	// Calculate required speed
	// DETERMINISM: Use epsilon comparison instead of direct float equality.
	// IEEE 754 floats can have precision issues, and direct equality comparison
	// with 0 is unreliable due to floating-point representation errors.
	// An epsilon of 1e-9 hours (3.6 microseconds) is well below any meaningful
	// time delta for travel detection while avoiding floating-point edge cases.
	const floatEpsilon = 1e-9
	timeDeltaHours := timeDelta.Hours()
	if math.Abs(timeDeltaHours) < floatEpsilon {
		timeDeltaHours = 0.001 // Prevent division by zero
	}
	requiredSpeedKmH := distanceKm / timeDeltaHours

	// Check if travel is impossible
	if requiredSpeedKmH <= config.MaxSpeedKmH {
		return nil, nil
	}

	// Build metadata
	metadata := ImpossibleTravelMetadata{
		FromCity:       lastEvent.City,
		FromCountry:    lastEvent.Country,
		FromLatitude:   lastEvent.Latitude,
		FromLongitude:  lastEvent.Longitude,
		FromTimestamp:  lastEvent.Timestamp,
		ToCity:         event.City,
		ToCountry:      event.Country,
		ToLatitude:     event.Latitude,
		ToLongitude:    event.Longitude,
		ToTimestamp:    event.Timestamp,
		DistanceKm:     roundTo2Decimals(distanceKm),
		TimeDeltaMins:  roundTo2Decimals(timeDelta.Minutes()),
		RequiredSpeedK: roundTo2Decimals(requiredSpeedKmH),
	}

	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Build alert message
	fromLocation := formatLocation(lastEvent.City, lastEvent.Country)
	toLocation := formatLocation(event.City, event.Country)

	alert := &Alert{
		RuleType:  RuleTypeImpossibleTravel,
		UserID:    event.UserID,
		Username:  event.Username,
		ServerID:  event.ServerID, // v2.1: Multi-server support
		MachineID: event.MachineID,
		IPAddress: event.IPAddress,
		Severity:  config.Severity,
		Title:     "Impossible Travel Detected",
		Message: fmt.Sprintf(
			"User %s traveled %.0f km from %s to %s in %.0f minutes (would require %.0f km/h)",
			event.Username,
			distanceKm,
			fromLocation,
			toLocation,
			timeDelta.Minutes(),
			requiredSpeedKmH,
		),
		Metadata:  metadataJSON,
		CreatedAt: time.Now(),
	}

	return alert, nil
}

// Configure updates the detector configuration.
func (d *ImpossibleTravelDetector) Configure(config json.RawMessage) error {
	var newConfig ImpossibleTravelConfig
	if err := json.Unmarshal(config, &newConfig); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Validate configuration
	if newConfig.MaxSpeedKmH <= 0 {
		return fmt.Errorf("max_speed_kmh must be positive")
	}
	if newConfig.MinDistanceKm < 0 {
		return fmt.Errorf("min_distance_km cannot be negative")
	}
	if newConfig.MinTimeDeltaMinutes < 0 {
		return fmt.Errorf("min_time_delta_minutes cannot be negative")
	}

	d.mu.Lock()
	d.config = newConfig
	d.mu.Unlock()

	return nil
}

// Enabled returns whether this detector is enabled.
func (d *ImpossibleTravelDetector) Enabled() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.enabled
}

// SetEnabled enables or disables the detector.
func (d *ImpossibleTravelDetector) SetEnabled(enabled bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.enabled = enabled
}

// Config returns the current configuration.
func (d *ImpossibleTravelDetector) Config() ImpossibleTravelConfig {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.config
}

// haversineDistance calculates the great-circle distance between two points
// on Earth using the Haversine formula. Returns distance in kilometers.
func haversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadiusKm = 6371.0

	// Convert to radians
	lat1Rad := lat1 * math.Pi / 180.0
	lon1Rad := lon1 * math.Pi / 180.0
	lat2Rad := lat2 * math.Pi / 180.0
	lon2Rad := lon2 * math.Pi / 180.0

	// Haversine formula
	dLat := lat2Rad - lat1Rad
	dLon := lon2Rad - lon1Rad

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(dLon/2)*math.Sin(dLon/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadiusKm * c
}

// formatLocation returns a human-readable location string.
func formatLocation(city, country string) string {
	if city != "" && country != "" {
		return city + ", " + country
	}
	if country != "" {
		return country
	}
	if city != "" {
		return city
	}
	return "Unknown"
}

// roundTo2Decimals rounds a float64 to 2 decimal places.
func roundTo2Decimals(f float64) float64 {
	return math.Round(f*100) / 100
}
