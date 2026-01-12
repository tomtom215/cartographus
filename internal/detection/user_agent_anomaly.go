// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package detection

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/goccy/go-json"
)

// RuleTypeUserAgentAnomaly detects unusual user agent patterns.
const RuleTypeUserAgentAnomaly RuleType = "user_agent_anomaly"

// UserAgentAnomalyConfig configures the user agent anomaly detector.
type UserAgentAnomalyConfig struct {
	// WindowMinutes is the time window for tracking user agent changes.
	WindowMinutes int `json:"window_minutes"`

	// AlertOnNewUserAgent triggers an alert when a user uses a new user agent.
	AlertOnNewUserAgent bool `json:"alert_on_new_user_agent"`

	// AlertOnPlatformSwitch triggers when a user switches platforms rapidly (e.g., iOS to Android).
	AlertOnPlatformSwitch bool `json:"alert_on_platform_switch"`

	// MinHistoryForAnomaly requires this many previous events before flagging anomalies.
	MinHistoryForAnomaly int `json:"min_history_for_anomaly"`

	// SuspiciousPatterns are regex patterns for suspicious user agents (bots, automation).
	SuspiciousPatterns []string `json:"suspicious_patterns,omitempty"`

	// Severity for generated alerts.
	Severity Severity `json:"severity"`
}

// DefaultUserAgentAnomalyConfig returns sensible defaults.
func DefaultUserAgentAnomalyConfig() UserAgentAnomalyConfig {
	return UserAgentAnomalyConfig{
		WindowMinutes:         30,
		AlertOnNewUserAgent:   true,
		AlertOnPlatformSwitch: true,
		MinHistoryForAnomaly:  3,
		SuspiciousPatterns: []string{
			"curl", "wget", "python", "bot", "crawler", "spider",
			"headless", "phantom", "selenium", "puppeteer",
		},
		Severity: SeverityWarning,
	}
}

// UserAgentAnomalyMetadata contains details for user agent anomaly alerts.
type UserAgentAnomalyMetadata struct {
	AnomalyType       string    `json:"anomaly_type"` // new_agent, platform_switch, suspicious_pattern
	CurrentPlatform   string    `json:"current_platform"`
	CurrentPlayer     string    `json:"current_player"`
	CurrentDevice     string    `json:"current_device"`
	PreviousPlatform  string    `json:"previous_platform,omitempty"`
	PreviousPlayer    string    `json:"previous_player,omitempty"`
	PreviousDevice    string    `json:"previous_device,omitempty"`
	PreviousTimestamp time.Time `json:"previous_timestamp,omitempty"`
	MatchedPattern    string    `json:"matched_pattern,omitempty"`
	TimeSinceLastMins float64   `json:"time_since_last_mins,omitempty"`
}

// UserAgentHistory provides access to user agent history for detection.
type UserAgentHistory interface {
	// GetRecentUserAgentsForUser retrieves recent user agents used by a user.
	GetRecentUserAgentsForUser(ctx context.Context, userID int, window time.Duration) ([]UserAgentEvent, error)

	// GetLastUserAgentForUser retrieves the most recent user agent for a user.
	GetLastUserAgentForUser(ctx context.Context, userID int) (*UserAgentEvent, error)
}

// UserAgentEvent represents a user agent usage event.
type UserAgentEvent struct {
	UserID    int       `json:"user_id"`
	Platform  string    `json:"platform"`
	Player    string    `json:"player"`
	Device    string    `json:"device"`
	IPAddress string    `json:"ip_address"`
	Timestamp time.Time `json:"timestamp"`
}

// UserAgentAnomalyDetector detects unusual user agent patterns.
type UserAgentAnomalyDetector struct {
	config       UserAgentAnomalyConfig
	eventHistory EventHistory
	enabled      bool
	mu           sync.RWMutex
}

// NewUserAgentAnomalyDetector creates a new user agent anomaly detector.
func NewUserAgentAnomalyDetector(eventHistory EventHistory) *UserAgentAnomalyDetector {
	return &UserAgentAnomalyDetector{
		config:       DefaultUserAgentAnomalyConfig(),
		eventHistory: eventHistory,
		enabled:      true,
	}
}

// Type returns the rule type.
func (d *UserAgentAnomalyDetector) Type() RuleType {
	return RuleTypeUserAgentAnomaly
}

// Check evaluates the event against the user agent anomaly rule.
func (d *UserAgentAnomalyDetector) Check(ctx context.Context, event *DetectionEvent) (*Alert, error) {
	d.mu.RLock()
	if !d.enabled {
		d.mu.RUnlock()
		return nil, nil
	}
	config := d.config
	d.mu.RUnlock()

	// Skip if no platform info
	if event.Platform == "" && event.Player == "" {
		return nil, nil
	}

	// Check for suspicious patterns first
	if alert := d.checkSuspiciousPatterns(event, &config); alert != nil {
		return alert, nil
	}

	// Get last event for user to compare on this server
	// v2.1: Pass serverID to scope detection to the same server instance
	lastEvent, err := d.eventHistory.GetLastEventForUser(ctx, event.UserID, event.ServerID)
	if err != nil {
		// No previous event - this is the first one, nothing to compare
		// Intentionally ignoring error since "not found" is expected for first-time users
		return nil, nil //nolint:nilerr // intentional: treat any error as "no history"
	}

	// Check for platform switch (e.g., iOS to Android in short time)
	if config.AlertOnPlatformSwitch {
		if alert := d.checkPlatformSwitch(event, lastEvent, &config); alert != nil {
			return alert, nil
		}
	}

	// Check for new/unusual user agent
	if config.AlertOnNewUserAgent {
		if alert := d.checkNewUserAgent(event, lastEvent, &config); alert != nil {
			return alert, nil
		}
	}

	return nil, nil
}

// checkSuspiciousPatterns checks for known suspicious user agent patterns.
func (d *UserAgentAnomalyDetector) checkSuspiciousPatterns(event *DetectionEvent, config *UserAgentAnomalyConfig) *Alert {
	// Combine platform, player, and device for checking
	agentString := strings.ToLower(fmt.Sprintf("%s %s %s", event.Platform, event.Player, event.Device))

	for _, pattern := range config.SuspiciousPatterns {
		if strings.Contains(agentString, strings.ToLower(pattern)) {
			metadata := UserAgentAnomalyMetadata{
				AnomalyType:     "suspicious_pattern",
				CurrentPlatform: event.Platform,
				CurrentPlayer:   event.Player,
				CurrentDevice:   event.Device,
				MatchedPattern:  pattern,
			}

			metadataJSON, err := json.Marshal(metadata)
			if err != nil {
				return nil
			}

			return &Alert{
				RuleType:  RuleTypeUserAgentAnomaly,
				UserID:    event.UserID,
				Username:  event.Username,
				ServerID:  event.ServerID, // v2.1: Multi-server support
				MachineID: event.MachineID,
				IPAddress: event.IPAddress,
				Severity:  SeverityCritical, // Suspicious patterns are critical
				Title:     "Suspicious User Agent Detected",
				Message: fmt.Sprintf(
					"User %s is using a potentially suspicious client matching pattern '%s': %s/%s",
					event.Username,
					pattern,
					event.Platform,
					event.Player,
				),
				Metadata:  metadataJSON,
				CreatedAt: time.Now(),
			}
		}
	}

	return nil
}

// checkPlatformSwitch checks for rapid platform switches.
func (d *UserAgentAnomalyDetector) checkPlatformSwitch(event, lastEvent *DetectionEvent, config *UserAgentAnomalyConfig) *Alert {
	if lastEvent == nil || lastEvent.Platform == "" {
		return nil
	}

	// Normalize platforms for comparison
	currentPlatform := normalizePlatform(event.Platform)
	previousPlatform := normalizePlatform(lastEvent.Platform)

	// Skip if same platform family
	if currentPlatform == previousPlatform {
		return nil
	}

	// Check time window
	timeDelta := event.Timestamp.Sub(lastEvent.Timestamp)
	windowDuration := time.Duration(config.WindowMinutes) * time.Minute

	if timeDelta > windowDuration {
		return nil // Not within suspicious window
	}

	// This is a rapid platform switch
	metadata := UserAgentAnomalyMetadata{
		AnomalyType:       "platform_switch",
		CurrentPlatform:   event.Platform,
		CurrentPlayer:     event.Player,
		CurrentDevice:     event.Device,
		PreviousPlatform:  lastEvent.Platform,
		PreviousPlayer:    lastEvent.Player,
		PreviousDevice:    lastEvent.Device,
		PreviousTimestamp: lastEvent.Timestamp,
		TimeSinceLastMins: timeDelta.Minutes(),
	}

	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return nil
	}

	return &Alert{
		RuleType:  RuleTypeUserAgentAnomaly,
		UserID:    event.UserID,
		Username:  event.Username,
		ServerID:  event.ServerID, // v2.1: Multi-server support
		MachineID: event.MachineID,
		IPAddress: event.IPAddress,
		Severity:  config.Severity,
		Title:     "Rapid Platform Switch Detected",
		Message: fmt.Sprintf(
			"User %s switched from %s to %s in %.1f minutes",
			event.Username,
			previousPlatform,
			currentPlatform,
			timeDelta.Minutes(),
		),
		Metadata:  metadataJSON,
		CreatedAt: time.Now(),
	}
}

// checkNewUserAgent checks for completely new user agents.
func (d *UserAgentAnomalyDetector) checkNewUserAgent(event, lastEvent *DetectionEvent, config *UserAgentAnomalyConfig) *Alert {
	if lastEvent == nil {
		return nil // First event, nothing to compare
	}

	// Check if this is an entirely new device/player combination
	currentCombo := fmt.Sprintf("%s|%s|%s", event.Platform, event.Player, event.Device)
	previousCombo := fmt.Sprintf("%s|%s|%s", lastEvent.Platform, lastEvent.Player, lastEvent.Device)

	if currentCombo == previousCombo {
		return nil // Same user agent
	}

	// Check time window
	timeDelta := event.Timestamp.Sub(lastEvent.Timestamp)
	windowDuration := time.Duration(config.WindowMinutes) * time.Minute

	if timeDelta > windowDuration {
		return nil // Not suspicious if outside window
	}

	metadata := UserAgentAnomalyMetadata{
		AnomalyType:       "new_agent",
		CurrentPlatform:   event.Platform,
		CurrentPlayer:     event.Player,
		CurrentDevice:     event.Device,
		PreviousPlatform:  lastEvent.Platform,
		PreviousPlayer:    lastEvent.Player,
		PreviousDevice:    lastEvent.Device,
		PreviousTimestamp: lastEvent.Timestamp,
		TimeSinceLastMins: timeDelta.Minutes(),
	}

	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return nil
	}

	return &Alert{
		RuleType:  RuleTypeUserAgentAnomaly,
		UserID:    event.UserID,
		Username:  event.Username,
		ServerID:  event.ServerID, // v2.1: Multi-server support
		MachineID: event.MachineID,
		IPAddress: event.IPAddress,
		Severity:  SeverityInfo, // New agent is informational
		Title:     "New Device/Client Detected",
		Message: fmt.Sprintf(
			"User %s started using a new client (%s/%s) within %.1f minutes of using %s/%s",
			event.Username,
			event.Platform,
			event.Player,
			timeDelta.Minutes(),
			lastEvent.Platform,
			lastEvent.Player,
		),
		Metadata:  metadataJSON,
		CreatedAt: time.Now(),
	}
}

// normalizePlatform groups platforms into families for comparison.
//
//nolint:gocyclo // complexity inherent to platform family classification
func normalizePlatform(platform string) string {
	p := strings.ToLower(platform)

	// iOS family
	if strings.Contains(p, "ios") || strings.Contains(p, "iphone") || strings.Contains(p, "ipad") {
		return "iOS"
	}

	// Android family
	if strings.Contains(p, "android") {
		return "Android"
	}

	// macOS family
	if strings.Contains(p, "macos") || strings.Contains(p, "osx") || strings.Contains(p, "mac os") {
		return "macOS"
	}

	// Windows family
	if strings.Contains(p, "windows") {
		return "Windows"
	}

	// Linux family
	if strings.Contains(p, "linux") || strings.Contains(p, "ubuntu") || strings.Contains(p, "debian") || strings.Contains(p, "fedora") || strings.Contains(p, "centos") {
		return "Linux"
	}

	// Web/Browser family
	if strings.Contains(p, "chrome") || strings.Contains(p, "firefox") || strings.Contains(p, "safari") || strings.Contains(p, "edge") || strings.Contains(p, "web") {
		return "Web"
	}

	// Smart TV family
	if strings.Contains(p, "tv") || strings.Contains(p, "roku") || strings.Contains(p, "fire") || strings.Contains(p, "tizen") || strings.Contains(p, "webos") {
		return "SmartTV"
	}

	// Gaming consoles
	if strings.Contains(p, "xbox") || strings.Contains(p, "playstation") || strings.Contains(p, "ps4") || strings.Contains(p, "ps5") {
		return "Console"
	}

	return platform
}

// Configure updates the detector configuration.
func (d *UserAgentAnomalyDetector) Configure(config json.RawMessage) error {
	var newConfig UserAgentAnomalyConfig
	if err := json.Unmarshal(config, &newConfig); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Validate configuration
	if newConfig.WindowMinutes <= 0 {
		return fmt.Errorf("window_minutes must be positive")
	}
	if newConfig.MinHistoryForAnomaly < 0 {
		return fmt.Errorf("min_history_for_anomaly must be non-negative")
	}

	d.mu.Lock()
	d.config = newConfig
	d.mu.Unlock()

	return nil
}

// Enabled returns whether this detector is enabled.
func (d *UserAgentAnomalyDetector) Enabled() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.enabled
}

// SetEnabled enables or disables the detector.
func (d *UserAgentAnomalyDetector) SetEnabled(enabled bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.enabled = enabled
}

// Config returns the current configuration.
func (d *UserAgentAnomalyDetector) Config() UserAgentAnomalyConfig {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.config
}
