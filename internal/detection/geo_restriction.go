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

// GeoRestrictionMetadata contains details for geo restriction alerts.
type GeoRestrictionMetadata struct {
	Country          string   `json:"country"`
	City             string   `json:"city,omitempty"`
	IPAddress        string   `json:"ip_address"`
	RestrictionMode  string   `json:"restriction_mode"` // blocklist, allowlist
	BlockedCountries []string `json:"blocked_countries,omitempty"`
	AllowedCountries []string `json:"allowed_countries,omitempty"`
}

// GeoRestrictionDetector blocks streaming from specified countries.
// Can operate in blocklist mode (block specific countries) or
// allowlist mode (only allow specific countries).
type GeoRestrictionDetector struct {
	config       GeoRestrictionConfig
	eventHistory EventHistory
	enabled      bool
	mu           sync.RWMutex
}

// NewGeoRestrictionDetector creates a new geo restriction detector.
func NewGeoRestrictionDetector(eventHistory EventHistory) *GeoRestrictionDetector {
	return &GeoRestrictionDetector{
		config:       DefaultGeoRestrictionConfig(),
		eventHistory: eventHistory,
		enabled:      false, // Disabled by default - requires explicit configuration
	}
}

// Type returns the rule type.
func (d *GeoRestrictionDetector) Type() RuleType {
	return RuleTypeGeoRestriction
}

// Check evaluates the event against the geo restriction rule.
func (d *GeoRestrictionDetector) Check(ctx context.Context, event *DetectionEvent) (*Alert, error) {
	d.mu.RLock()
	if !d.enabled {
		d.mu.RUnlock()
		return nil, nil
	}
	config := d.config
	d.mu.RUnlock()

	// Skip if no country data
	if event.Country == "" {
		return nil, nil
	}

	var isViolation bool
	var restrictionMode string

	// Check allowlist mode first (if configured)
	if len(config.AllowedCountries) > 0 {
		restrictionMode = "allowlist"
		isViolation = true // Assume violation unless found in allowlist
		for _, allowed := range config.AllowedCountries {
			if event.Country == allowed {
				isViolation = false
				break
			}
		}
	} else if len(config.BlockedCountries) > 0 {
		// Blocklist mode
		restrictionMode = "blocklist"
		for _, blocked := range config.BlockedCountries {
			if event.Country == blocked {
				isViolation = true
				break
			}
		}
	}

	if !isViolation {
		return nil, nil
	}

	// Build metadata
	metadata := GeoRestrictionMetadata{
		Country:         event.Country,
		City:            event.City,
		IPAddress:       event.IPAddress,
		RestrictionMode: restrictionMode,
	}

	if restrictionMode == "blocklist" {
		metadata.BlockedCountries = config.BlockedCountries
	} else {
		metadata.AllowedCountries = config.AllowedCountries
	}

	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	var message string
	if restrictionMode == "blocklist" {
		message = fmt.Sprintf(
			"User %s attempted to stream from blocked country: %s",
			event.Username,
			event.Country,
		)
	} else {
		message = fmt.Sprintf(
			"User %s attempted to stream from unauthorized country: %s",
			event.Username,
			event.Country,
		)
	}

	alert := &Alert{
		RuleType:  RuleTypeGeoRestriction,
		UserID:    event.UserID,
		Username:  event.Username,
		ServerID:  event.ServerID, // v2.1: Multi-server support
		MachineID: event.MachineID,
		IPAddress: event.IPAddress,
		Severity:  config.Severity,
		Title:     "Geographic Restriction Violation",
		Message:   message,
		Metadata:  metadataJSON,
		CreatedAt: time.Now(),
	}

	return alert, nil
}

// Configure updates the detector configuration.
func (d *GeoRestrictionDetector) Configure(config json.RawMessage) error {
	var newConfig GeoRestrictionConfig
	if err := json.Unmarshal(config, &newConfig); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Validate - at least one list must be configured
	if len(newConfig.BlockedCountries) == 0 && len(newConfig.AllowedCountries) == 0 {
		return fmt.Errorf("either blocked_countries or allowed_countries must be configured")
	}

	// Cannot use both modes
	if len(newConfig.BlockedCountries) > 0 && len(newConfig.AllowedCountries) > 0 {
		return fmt.Errorf("cannot use both blocked_countries and allowed_countries")
	}

	d.mu.Lock()
	d.config = newConfig
	d.mu.Unlock()

	return nil
}

// Enabled returns whether this detector is enabled.
func (d *GeoRestrictionDetector) Enabled() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.enabled
}

// SetEnabled enables or disables the detector.
func (d *GeoRestrictionDetector) SetEnabled(enabled bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.enabled = enabled
}

// Config returns the current configuration.
func (d *GeoRestrictionDetector) Config() GeoRestrictionConfig {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.config
}

// AddBlockedCountry adds a country to the blocklist.
func (d *GeoRestrictionDetector) AddBlockedCountry(countryCode string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Can't add to blocklist if allowlist is active
	if len(d.config.AllowedCountries) > 0 {
		return fmt.Errorf("cannot add to blocklist while allowlist is active")
	}

	// Check if already blocked
	for _, c := range d.config.BlockedCountries {
		if c == countryCode {
			return nil // Already blocked
		}
	}

	d.config.BlockedCountries = append(d.config.BlockedCountries, countryCode)
	return nil
}

// RemoveBlockedCountry removes a country from the blocklist.
func (d *GeoRestrictionDetector) RemoveBlockedCountry(countryCode string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	newList := make([]string, 0, len(d.config.BlockedCountries))
	for _, c := range d.config.BlockedCountries {
		if c != countryCode {
			newList = append(newList, c)
		}
	}
	d.config.BlockedCountries = newList
}

// IsCountryBlocked checks if a country is blocked.
func (d *GeoRestrictionDetector) IsCountryBlocked(countryCode string) bool {
	d.mu.RLock()
	defer d.mu.RUnlock()

	// Allowlist mode: blocked if not in list
	if len(d.config.AllowedCountries) > 0 {
		for _, c := range d.config.AllowedCountries {
			if c == countryCode {
				return false
			}
		}
		return true
	}

	// Blocklist mode: blocked if in list
	for _, c := range d.config.BlockedCountries {
		if c == countryCode {
			return true
		}
	}
	return false
}
