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

	"github.com/tomtom215/cartographus/internal/vpn"
)

// RuleTypeVPNUsage detects streaming from known VPN IP addresses.
const RuleTypeVPNUsage RuleType = "vpn_usage"

// VPNUsageConfig configures the VPN usage detector.
type VPNUsageConfig struct {
	// Severity for generated alerts.
	Severity Severity `json:"severity"`

	// AlertOnFirstUse generates an alert the first time a user streams via VPN.
	AlertOnFirstUse bool `json:"alert_on_first_use"`

	// AlertOnNewProvider generates an alert when a user switches VPN providers.
	AlertOnNewProvider bool `json:"alert_on_new_provider"`

	// AlertOnHighRisk generates alerts only for high-risk VPN usage patterns.
	// High-risk includes: VPN IP + impossible travel, VPN + multiple devices, etc.
	AlertOnHighRisk bool `json:"alert_on_high_risk"`

	// ExcludedProviders are VPN providers to ignore (e.g., corporate VPNs).
	ExcludedProviders []string `json:"excluded_providers,omitempty"`

	// ExcludedUsers are user IDs to exclude from VPN detection.
	ExcludedUsers []int `json:"excluded_users,omitempty"`

	// TrackVPNHistory maintains a history of VPN usage per user.
	TrackVPNHistory bool `json:"track_vpn_history"`
}

// DefaultVPNUsageConfig returns sensible defaults.
func DefaultVPNUsageConfig() VPNUsageConfig {
	return VPNUsageConfig{
		Severity:           SeverityInfo, // VPN usage is informational by default
		AlertOnFirstUse:    true,
		AlertOnNewProvider: true,
		AlertOnHighRisk:    true,
		ExcludedProviders:  []string{},
		ExcludedUsers:      []int{},
		TrackVPNHistory:    true,
	}
}

// VPNUsageMetadata contains details for VPN usage alerts.
type VPNUsageMetadata struct {
	// Provider is the VPN provider name.
	Provider string `json:"provider"`

	// ProviderDisplayName is the human-readable provider name.
	ProviderDisplayName string `json:"provider_display_name"`

	// VPNServerCountry is the VPN server's country.
	VPNServerCountry string `json:"vpn_server_country"`

	// VPNServerCity is the VPN server's city.
	VPNServerCity string `json:"vpn_server_city,omitempty"`

	// Confidence is the VPN detection confidence (0-100).
	Confidence int `json:"confidence"`

	// AlertReason explains why the alert was generated.
	AlertReason string `json:"alert_reason"`

	// IsFirstVPNUse indicates if this is the user's first VPN usage.
	IsFirstVPNUse bool `json:"is_first_vpn_use,omitempty"`

	// PreviousProvider is the user's previous VPN provider (if different).
	PreviousProvider string `json:"previous_provider,omitempty"`

	// GeolocationCountry is the geolocation country (from IP lookup, not VPN server).
	GeolocationCountry string `json:"geolocation_country,omitempty"`

	// UserHistory summarizes the user's VPN usage history.
	UserHistory *VPNUserHistory `json:"user_history,omitempty"`
}

// VPNUserHistory tracks a user's VPN usage patterns.
type VPNUserHistory struct {
	TotalVPNSessions int       `json:"total_vpn_sessions"`
	ProvidersUsed    []string  `json:"providers_used"`
	FirstVPNUsage    time.Time `json:"first_vpn_usage"`
	LastVPNUsage     time.Time `json:"last_vpn_usage"`
	VPNUsagePercent  float64   `json:"vpn_usage_percent"` // % of sessions via VPN
}

// VPNLookupService defines the interface for VPN IP lookup.
// This interface allows for mocking in tests.
type VPNLookupService interface {
	// LookupIP returns VPN information for an IP address.
	LookupIP(ip string) *vpn.LookupResult

	// Enabled returns whether VPN detection is enabled.
	Enabled() bool
}

// VPNUsageDetector detects streaming from known VPN IP addresses.
type VPNUsageDetector struct {
	config  VPNUsageConfig
	enabled bool
	vpnSvc  VPNLookupService

	// userVPNHistory tracks VPN usage per user (userID -> history)
	userVPNHistory map[int]*VPNUserHistory

	mu sync.RWMutex
}

// NewVPNUsageDetector creates a new VPN usage detector.
func NewVPNUsageDetector(vpnService VPNLookupService) *VPNUsageDetector {
	return &VPNUsageDetector{
		config:         DefaultVPNUsageConfig(),
		enabled:        true,
		vpnSvc:         vpnService,
		userVPNHistory: make(map[int]*VPNUserHistory),
	}
}

// Type returns the rule type.
func (d *VPNUsageDetector) Type() RuleType {
	return RuleTypeVPNUsage
}

// Check evaluates the event for VPN usage.
//
//nolint:gocyclo // complexity inherent to VPN detection logic
func (d *VPNUsageDetector) Check(ctx context.Context, event *DetectionEvent) (*Alert, error) {
	d.mu.RLock()
	if !d.enabled {
		d.mu.RUnlock()
		return nil, nil
	}
	config := d.config
	d.mu.RUnlock()

	// Check if user is excluded
	for _, excludedUser := range config.ExcludedUsers {
		if event.UserID == excludedUser {
			return nil, nil
		}
	}

	// Skip if no IP address
	if event.IPAddress == "" {
		return nil, nil
	}

	// Skip LAN connections (they won't be VPN exit IPs)
	if event.LocationType == "lan" {
		return nil, nil
	}

	// Check if VPN service is available
	if d.vpnSvc == nil || !d.vpnSvc.Enabled() {
		return nil, nil
	}

	// Lookup VPN information
	vpnResult := d.vpnSvc.LookupIP(event.IPAddress)
	if !vpnResult.IsVPN {
		return nil, nil
	}

	// Check if provider is excluded
	for _, excludedProvider := range config.ExcludedProviders {
		if vpnResult.Provider == excludedProvider {
			return nil, nil
		}
	}

	// Determine if we should generate an alert
	alertReason, shouldAlert := d.shouldAlert(event, vpnResult, &config)
	if !shouldAlert {
		// Still track VPN usage even if we don't alert
		d.updateUserHistory(event.UserID, vpnResult.Provider)
		return nil, nil
	}

	// Get user history for metadata
	d.mu.RLock()
	history := d.userVPNHistory[event.UserID]
	d.mu.RUnlock()

	// Build metadata
	metadata := VPNUsageMetadata{
		Provider:            vpnResult.Provider,
		ProviderDisplayName: vpnResult.ProviderDisplayName,
		VPNServerCountry:    vpnResult.ServerCountry,
		VPNServerCity:       vpnResult.ServerCity,
		Confidence:          vpnResult.Confidence,
		AlertReason:         alertReason,
		GeolocationCountry:  event.Country,
	}

	if history == nil {
		metadata.IsFirstVPNUse = true
	} else {
		metadata.UserHistory = history
		// Check for provider switch
		if len(history.ProvidersUsed) > 0 {
			lastProvider := history.ProvidersUsed[len(history.ProvidersUsed)-1]
			if lastProvider != vpnResult.Provider {
				metadata.PreviousProvider = lastProvider
			}
		}
	}

	// Update history after building metadata
	d.updateUserHistory(event.UserID, vpnResult.Provider)

	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Build alert message
	var message string
	if vpnResult.ServerCity != "" {
		message = fmt.Sprintf("User %s is streaming via %s VPN (server in %s, %s)",
			event.Username, vpnResult.ProviderDisplayName, vpnResult.ServerCity, vpnResult.ServerCountry)
	} else {
		message = fmt.Sprintf("User %s is streaming via %s VPN (server in %s)",
			event.Username, vpnResult.ProviderDisplayName, vpnResult.ServerCountry)
	}

	return &Alert{
		RuleType:  RuleTypeVPNUsage,
		UserID:    event.UserID,
		Username:  event.Username,
		ServerID:  event.ServerID, // v2.1: Multi-server support
		IPAddress: event.IPAddress,
		Severity:  config.Severity,
		Title:     "VPN Usage Detected",
		Message:   message,
		Metadata:  metadataJSON,
		CreatedAt: time.Now(),
	}, nil
}

// shouldAlert determines if an alert should be generated based on configuration.
func (d *VPNUsageDetector) shouldAlert(event *DetectionEvent, vpnResult *vpn.LookupResult, config *VPNUsageConfig) (string, bool) {
	d.mu.RLock()
	history := d.userVPNHistory[event.UserID]
	d.mu.RUnlock()

	// First VPN use
	if config.AlertOnFirstUse && history == nil {
		return "first_vpn_use", true
	}

	// New provider
	if config.AlertOnNewProvider && history != nil {
		isNewProvider := true
		for _, provider := range history.ProvidersUsed {
			if provider == vpnResult.Provider {
				isNewProvider = false
				break
			}
		}
		if isNewProvider {
			return "new_vpn_provider", true
		}
	}

	// High-risk scenarios are handled separately by combining with other detectors
	// For now, we alert on configured conditions

	return "", false
}

// updateUserHistory updates the VPN usage history for a user.
func (d *VPNUsageDetector) updateUserHistory(userID int, provider string) {
	if !d.config.TrackVPNHistory {
		return
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	history := d.userVPNHistory[userID]
	if history == nil {
		history = &VPNUserHistory{
			FirstVPNUsage: time.Now(),
			ProvidersUsed: []string{},
		}
		d.userVPNHistory[userID] = history
	}

	history.TotalVPNSessions++
	history.LastVPNUsage = time.Now()

	// Add provider if not already tracked
	found := false
	for _, p := range history.ProvidersUsed {
		if p == provider {
			found = true
			break
		}
	}
	if !found {
		history.ProvidersUsed = append(history.ProvidersUsed, provider)
	}
}

// Configure updates the detector configuration.
func (d *VPNUsageDetector) Configure(config json.RawMessage) error {
	var newConfig VPNUsageConfig
	if err := json.Unmarshal(config, &newConfig); err != nil {
		return fmt.Errorf("failed to parse VPN usage config: %w", err)
	}

	// Validate configuration
	if newConfig.Severity != "" &&
		newConfig.Severity != SeverityInfo &&
		newConfig.Severity != SeverityWarning &&
		newConfig.Severity != SeverityCritical {
		return fmt.Errorf("invalid severity: %s", newConfig.Severity)
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	// Merge with defaults for unset values
	if newConfig.Severity == "" {
		newConfig.Severity = d.config.Severity
	}

	d.config = newConfig
	return nil
}

// Enabled returns whether this detector is enabled.
func (d *VPNUsageDetector) Enabled() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.enabled
}

// SetEnabled enables or disables the detector.
func (d *VPNUsageDetector) SetEnabled(enabled bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.enabled = enabled
}

// Config returns the current configuration.
func (d *VPNUsageDetector) Config() VPNUsageConfig {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.config
}

// GetUserHistory returns the VPN usage history for a specific user.
func (d *VPNUsageDetector) GetUserHistory(userID int) *VPNUserHistory {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if history, ok := d.userVPNHistory[userID]; ok {
		// Return a copy
		historyCopy := *history
		historyCopy.ProvidersUsed = make([]string, len(history.ProvidersUsed))
		copy(historyCopy.ProvidersUsed, history.ProvidersUsed)
		return &historyCopy
	}
	return nil
}

// ClearUserHistory clears the VPN usage history for a specific user.
func (d *VPNUsageDetector) ClearUserHistory(userID int) {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.userVPNHistory, userID)
}

// ClearAllHistory clears all VPN usage history.
func (d *VPNUsageDetector) ClearAllHistory() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.userVPNHistory = make(map[int]*VPNUserHistory)
}
