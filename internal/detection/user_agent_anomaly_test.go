// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package detection

import (
	"context"
	"testing"
	"time"

	"github.com/goccy/go-json"
)

// mockEventHistoryForUA implements EventHistory for user agent testing.
// v2.1: Updated interface to include serverID parameter for multi-server support
type mockEventHistoryForUA struct {
	lastEvent *DetectionEvent
	events    []DetectionEvent
}

func (m *mockEventHistoryForUA) GetLastEventForUser(ctx context.Context, userID int, serverID string) (*DetectionEvent, error) {
	return m.lastEvent, nil
}

func (m *mockEventHistoryForUA) GetActiveStreamsForUser(ctx context.Context, userID int, serverID string) ([]DetectionEvent, error) {
	return m.events, nil
}

func (m *mockEventHistoryForUA) GetRecentIPsForDevice(ctx context.Context, machineID string, serverID string, window time.Duration) ([]string, error) {
	return nil, nil
}

func (m *mockEventHistoryForUA) GetSimultaneousLocations(ctx context.Context, userID int, serverID string, window time.Duration) ([]DetectionEvent, error) {
	return nil, nil
}

func (m *mockEventHistoryForUA) GetGeolocation(ctx context.Context, ipAddress string) (*Geolocation, error) {
	return nil, nil
}

func TestUserAgentAnomalyDetector_Type(t *testing.T) {
	detector := NewUserAgentAnomalyDetector(nil)
	if detector.Type() != RuleTypeUserAgentAnomaly {
		t.Errorf("expected type %s, got %s", RuleTypeUserAgentAnomaly, detector.Type())
	}
}

func TestUserAgentAnomalyDetector_SuspiciousPattern(t *testing.T) {
	history := &mockEventHistoryForUA{}
	detector := NewUserAgentAnomalyDetector(history)

	ctx := context.Background()

	tests := []struct {
		name      string
		platform  string
		player    string
		device    string
		wantAlert bool
	}{
		{"Normal iOS client", "iOS", "Plex for iOS", "iPhone", false},
		{"Normal Android client", "Android", "Plex for Android", "Pixel 5", false},
		{"Suspicious curl", "curl/7.68.0", "", "", true},
		{"Suspicious wget", "wget/1.21", "", "", true},
		{"Suspicious bot", "SomeBot/1.0", "", "", true},
		{"Suspicious python", "Python-urllib/3.9", "", "", true},
		{"Suspicious headless", "HeadlessChrome", "Browser", "Unknown", true},
		{"Suspicious selenium", "Selenium WebDriver", "Browser", "Unknown", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := &DetectionEvent{
				UserID:    1,
				Username:  "testuser",
				Platform:  tt.platform,
				Player:    tt.player,
				Device:    tt.device,
				Timestamp: time.Now(),
			}

			alert, err := detector.Check(ctx, event)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantAlert && alert == nil {
				t.Error("expected alert for suspicious pattern, got nil")
			}
			if !tt.wantAlert && alert != nil {
				t.Errorf("unexpected alert for normal pattern: %s", alert.Title)
			}

			if alert != nil {
				if alert.Severity != SeverityCritical {
					t.Errorf("suspicious pattern alerts should be critical, got %s", alert.Severity)
				}

				var metadata UserAgentAnomalyMetadata
				if err := json.Unmarshal(alert.Metadata, &metadata); err != nil {
					t.Fatalf("failed to unmarshal metadata: %v", err)
				}
				if metadata.AnomalyType != "suspicious_pattern" {
					t.Errorf("expected anomaly_type suspicious_pattern, got %s", metadata.AnomalyType)
				}
			}
		})
	}
}

func TestUserAgentAnomalyDetector_PlatformSwitch(t *testing.T) {
	now := time.Now()
	lastEvent := &DetectionEvent{
		UserID:    1,
		Username:  "testuser",
		Platform:  "iOS",
		Player:    "Plex for iOS",
		Device:    "iPhone",
		Timestamp: now.Add(-10 * time.Minute),
	}

	history := &mockEventHistoryForUA{lastEvent: lastEvent}
	detector := NewUserAgentAnomalyDetector(history)

	ctx := context.Background()

	tests := []struct {
		name        string
		newPlatform string
		wantAlert   bool
	}{
		{"Same platform family", "iPhone 14", false}, // Still iOS
		{"Different platform", "Android", true},      // iOS to Android
		{"Windows", "Windows", true},                 // iOS to Windows
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := &DetectionEvent{
				UserID:    1,
				Username:  "testuser",
				Platform:  tt.newPlatform,
				Player:    "Plex",
				Device:    "Test Device",
				Timestamp: now,
			}

			alert, err := detector.Check(ctx, event)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantAlert && alert == nil {
				t.Error("expected alert for platform switch, got nil")
			}
			if !tt.wantAlert && alert != nil {
				// Make sure it's not a platform_switch alert
				var metadata UserAgentAnomalyMetadata
				json.Unmarshal(alert.Metadata, &metadata)
				if metadata.AnomalyType == "platform_switch" {
					t.Errorf("unexpected platform switch alert")
				}
			}
		})
	}
}

func TestUserAgentAnomalyDetector_NewUserAgent(t *testing.T) {
	now := time.Now()
	lastEvent := &DetectionEvent{
		UserID:    1,
		Username:  "testuser",
		Platform:  "iOS",
		Player:    "Plex for iOS",
		Device:    "iPhone 12",
		Timestamp: now.Add(-5 * time.Minute),
	}

	history := &mockEventHistoryForUA{lastEvent: lastEvent}
	detector := NewUserAgentAnomalyDetector(history)

	ctx := context.Background()

	// New device on same platform
	event := &DetectionEvent{
		UserID:    1,
		Username:  "testuser",
		Platform:  "iOS",
		Player:    "Plex for iOS",
		Device:    "iPad Pro", // Different device
		Timestamp: now,
	}

	alert, err := detector.Check(ctx, event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if alert == nil {
		t.Error("expected alert for new user agent")
		return
	}

	var metadata UserAgentAnomalyMetadata
	if err := json.Unmarshal(alert.Metadata, &metadata); err != nil {
		t.Fatalf("failed to unmarshal metadata: %v", err)
	}

	if metadata.AnomalyType != "new_agent" {
		t.Errorf("expected anomaly_type new_agent, got %s", metadata.AnomalyType)
	}
	if metadata.PreviousDevice != "iPhone 12" {
		t.Errorf("expected previous device iPhone 12, got %s", metadata.PreviousDevice)
	}
	if metadata.CurrentDevice != "iPad Pro" {
		t.Errorf("expected current device iPad Pro, got %s", metadata.CurrentDevice)
	}
}

func TestUserAgentAnomalyDetector_OutsideWindow(t *testing.T) {
	now := time.Now()
	lastEvent := &DetectionEvent{
		UserID:    1,
		Username:  "testuser",
		Platform:  "iOS",
		Player:    "Plex for iOS",
		Device:    "iPhone",
		Timestamp: now.Add(-2 * time.Hour), // 2 hours ago, outside 30 min window
	}

	history := &mockEventHistoryForUA{lastEvent: lastEvent}
	detector := NewUserAgentAnomalyDetector(history)

	ctx := context.Background()

	// Switch platform but outside time window
	event := &DetectionEvent{
		UserID:    1,
		Username:  "testuser",
		Platform:  "Android",
		Player:    "Plex for Android",
		Device:    "Pixel 5",
		Timestamp: now,
	}

	alert, err := detector.Check(ctx, event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should not alert because it's outside the window
	if alert != nil {
		var metadata UserAgentAnomalyMetadata
		json.Unmarshal(alert.Metadata, &metadata)
		if metadata.AnomalyType == "platform_switch" || metadata.AnomalyType == "new_agent" {
			t.Errorf("should not alert for events outside time window")
		}
	}
}

func TestUserAgentAnomalyDetector_Disabled(t *testing.T) {
	history := &mockEventHistoryForUA{}
	detector := NewUserAgentAnomalyDetector(history)
	detector.SetEnabled(false)

	ctx := context.Background()

	// Suspicious pattern that would normally alert
	event := &DetectionEvent{
		UserID:    1,
		Username:  "testuser",
		Platform:  "curl/7.68.0",
		Timestamp: time.Now(),
	}

	alert, err := detector.Check(ctx, event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if alert != nil {
		t.Error("disabled detector should not generate alerts")
	}
}

func TestUserAgentAnomalyDetector_Configure(t *testing.T) {
	detector := NewUserAgentAnomalyDetector(nil)

	// Valid configuration
	validConfig := `{
		"window_minutes": 60,
		"alert_on_new_user_agent": false,
		"alert_on_platform_switch": true,
		"min_history_for_anomaly": 5,
		"suspicious_patterns": ["bot", "crawler"],
		"severity": "critical"
	}`

	err := detector.Configure([]byte(validConfig))
	if err != nil {
		t.Fatalf("unexpected error for valid config: %v", err)
	}

	config := detector.Config()
	if config.WindowMinutes != 60 {
		t.Errorf("expected window_minutes 60, got %d", config.WindowMinutes)
	}
	if config.AlertOnNewUserAgent {
		t.Error("expected alert_on_new_user_agent to be false")
	}
	if len(config.SuspiciousPatterns) != 2 {
		t.Errorf("expected 2 suspicious patterns, got %d", len(config.SuspiciousPatterns))
	}

	// Invalid configuration
	invalidConfig := `{"window_minutes": -5}`
	err = detector.Configure([]byte(invalidConfig))
	if err == nil {
		t.Error("expected error for invalid config")
	}
}

func TestUserAgentAnomalyDetector_NoPlatformInfo(t *testing.T) {
	history := &mockEventHistoryForUA{}
	detector := NewUserAgentAnomalyDetector(history)

	ctx := context.Background()

	// Event with no platform/player info
	event := &DetectionEvent{
		UserID:    1,
		Username:  "testuser",
		Timestamp: time.Now(),
	}

	alert, err := detector.Check(ctx, event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if alert != nil {
		t.Error("should not alert when no platform info available")
	}
}

func TestNormalizePlatform(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"iOS", "iOS"},
		{"iPhone", "iOS"},
		{"iPad", "iOS"},
		{"Android", "Android"},
		{"Android TV", "Android"},
		{"macOS", "macOS"},
		{"Mac OS X", "macOS"},
		{"Windows 10", "Windows"},
		{"Linux", "Linux"},
		{"Ubuntu", "Linux"},
		{"Chrome", "Web"},
		{"Firefox", "Web"},
		{"Safari", "Web"},
		{"Roku", "SmartTV"},
		{"Fire TV", "SmartTV"},
		{"Samsung Tizen", "SmartTV"},
		{"Xbox One", "Console"},
		{"PlayStation 5", "Console"},
		{"PS4", "Console"},
		{"Unknown Platform", "Unknown Platform"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizePlatform(tt.input)
			if result != tt.expected {
				t.Errorf("normalizePlatform(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestUserAgentAnomalyDetector_EnableDisable(t *testing.T) {
	detector := NewUserAgentAnomalyDetector(nil)

	// Initially enabled
	if !detector.Enabled() {
		t.Error("detector should be enabled by default")
	}

	// Disable
	detector.SetEnabled(false)
	if detector.Enabled() {
		t.Error("detector should be disabled after SetEnabled(false)")
	}

	// Re-enable
	detector.SetEnabled(true)
	if !detector.Enabled() {
		t.Error("detector should be enabled after SetEnabled(true)")
	}
}
