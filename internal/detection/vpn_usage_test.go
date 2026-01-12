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

	"github.com/tomtom215/cartographus/internal/vpn"
)

// mockVPNService implements VPNLookupService for testing.
type mockVPNService struct {
	lookup  *vpn.Lookup
	enabled bool
}

func (s *mockVPNService) LookupIP(ip string) *vpn.LookupResult {
	if !s.enabled {
		return &vpn.LookupResult{IsVPN: false, Confidence: 0}
	}
	return s.lookup.LookupIP(ip)
}

func (s *mockVPNService) Enabled() bool {
	return s.enabled
}

// NewVPNUsageDetectorForTest creates a detector with test VPN lookup.
func NewVPNUsageDetectorForTest(lookup *vpn.Lookup) *VPNUsageDetector {
	return &VPNUsageDetector{
		config:         DefaultVPNUsageConfig(),
		enabled:        true,
		vpnSvc:         &mockVPNService{lookup: lookup, enabled: true},
		userVPNHistory: make(map[int]*VPNUserHistory),
	}
}

func TestVPNUsageDetector_Type(t *testing.T) {
	detector := NewVPNUsageDetector(nil)
	if detector.Type() != RuleTypeVPNUsage {
		t.Errorf("expected type %s, got %s", RuleTypeVPNUsage, detector.Type())
	}
}

func TestVPNUsageDetector_DetectsVPNIP(t *testing.T) {
	lookup := vpn.NewLookup()
	lookup.AddServer(&vpn.Server{
		Provider: "nordvpn",
		Country:  "United States",
		City:     "New York",
		Hostname: "us-nyc-001.nordvpn.com",
		IPs:      []string{"198.51.100.1"},
	})
	lookup.AddProvider(&vpn.Provider{Name: "nordvpn", DisplayName: "NordVPN"})

	detector := NewVPNUsageDetectorForTest(lookup)

	ctx := context.Background()
	event := &DetectionEvent{
		UserID:       1,
		Username:     "testuser",
		IPAddress:    "198.51.100.1",
		LocationType: "wan",
		Timestamp:    time.Now(),
	}

	alert, err := detector.Check(ctx, event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if alert == nil {
		t.Fatal("expected alert for VPN IP")
	}

	if alert.Title != "VPN Usage Detected" {
		t.Errorf("unexpected alert title: %s", alert.Title)
	}

	var metadata VPNUsageMetadata
	if err := json.Unmarshal(alert.Metadata, &metadata); err != nil {
		t.Fatalf("failed to unmarshal metadata: %v", err)
	}

	if metadata.Provider != "nordvpn" {
		t.Errorf("expected provider nordvpn, got %s", metadata.Provider)
	}
	if metadata.ProviderDisplayName != "NordVPN" {
		t.Errorf("expected display name NordVPN, got %s", metadata.ProviderDisplayName)
	}
	if metadata.VPNServerCountry != "United States" {
		t.Errorf("expected country United States, got %s", metadata.VPNServerCountry)
	}
	if metadata.VPNServerCity != "New York" {
		t.Errorf("expected city New York, got %s", metadata.VPNServerCity)
	}
	if !metadata.IsFirstVPNUse {
		t.Error("expected IsFirstVPNUse to be true")
	}
}

func TestVPNUsageDetector_NoAlertForNonVPN(t *testing.T) {
	lookup := vpn.NewLookup()
	lookup.AddServer(&vpn.Server{
		Provider: "nordvpn",
		IPs:      []string{"198.51.100.1"},
	})

	detector := NewVPNUsageDetectorForTest(lookup)

	ctx := context.Background()
	event := &DetectionEvent{
		UserID:       1,
		Username:     "testuser",
		IPAddress:    "8.8.8.8", // Not a VPN IP
		LocationType: "wan",
		Timestamp:    time.Now(),
	}

	alert, err := detector.Check(ctx, event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if alert != nil {
		t.Error("expected no alert for non-VPN IP")
	}
}

func TestVPNUsageDetector_SkipsLANConnections(t *testing.T) {
	lookup := vpn.NewLookup()
	lookup.AddServer(&vpn.Server{
		Provider: "nordvpn",
		IPs:      []string{"192.168.1.100"}, // Unlikely but testing LAN skip
	})

	detector := NewVPNUsageDetectorForTest(lookup)

	ctx := context.Background()
	event := &DetectionEvent{
		UserID:       1,
		Username:     "testuser",
		IPAddress:    "192.168.1.100",
		LocationType: "lan", // LAN connection
		Timestamp:    time.Now(),
	}

	alert, err := detector.Check(ctx, event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if alert != nil {
		t.Error("expected no alert for LAN connection")
	}
}

func TestVPNUsageDetector_ExcludedUsers(t *testing.T) {
	lookup := vpn.NewLookup()
	lookup.AddServer(&vpn.Server{
		Provider: "nordvpn",
		IPs:      []string{"198.51.100.1"},
	})

	detector := NewVPNUsageDetectorForTest(lookup)

	// Exclude user 1
	config := DefaultVPNUsageConfig()
	config.ExcludedUsers = []int{1}
	configJSON, _ := json.Marshal(config)
	detector.Configure(configJSON)

	ctx := context.Background()
	event := &DetectionEvent{
		UserID:       1, // Excluded user
		Username:     "excludeduser",
		IPAddress:    "198.51.100.1",
		LocationType: "wan",
		Timestamp:    time.Now(),
	}

	alert, err := detector.Check(ctx, event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if alert != nil {
		t.Error("expected no alert for excluded user")
	}
}

func TestVPNUsageDetector_ExcludedProviders(t *testing.T) {
	lookup := vpn.NewLookup()
	lookup.AddServer(&vpn.Server{
		Provider: "corporate_vpn",
		IPs:      []string{"198.51.100.1"},
	})
	lookup.AddProvider(&vpn.Provider{Name: "corporate_vpn", DisplayName: "Corporate VPN"})

	detector := NewVPNUsageDetectorForTest(lookup)

	// Exclude corporate VPN
	config := DefaultVPNUsageConfig()
	config.ExcludedProviders = []string{"corporate_vpn"}
	configJSON, _ := json.Marshal(config)
	detector.Configure(configJSON)

	ctx := context.Background()
	event := &DetectionEvent{
		UserID:       1,
		Username:     "testuser",
		IPAddress:    "198.51.100.1",
		LocationType: "wan",
		Timestamp:    time.Now(),
	}

	alert, err := detector.Check(ctx, event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if alert != nil {
		t.Error("expected no alert for excluded provider")
	}
}

func TestVPNUsageDetector_TracksUserHistory(t *testing.T) {
	lookup := vpn.NewLookup()
	lookup.AddServer(&vpn.Server{
		Provider: "nordvpn",
		IPs:      []string{"198.51.100.1"},
	})
	lookup.AddServer(&vpn.Server{
		Provider: "expressvpn",
		IPs:      []string{"203.0.113.50"},
	})
	lookup.AddProvider(&vpn.Provider{Name: "nordvpn", DisplayName: "NordVPN"})
	lookup.AddProvider(&vpn.Provider{Name: "expressvpn", DisplayName: "ExpressVPN"})

	detector := NewVPNUsageDetectorForTest(lookup)

	ctx := context.Background()

	// First VPN use
	event1 := &DetectionEvent{
		UserID:       1,
		Username:     "testuser",
		IPAddress:    "198.51.100.1",
		LocationType: "wan",
		Timestamp:    time.Now(),
	}

	alert1, _ := detector.Check(ctx, event1)
	if alert1 == nil {
		t.Fatal("expected alert for first VPN use")
	}

	// Check history was updated
	history := detector.GetUserHistory(1)
	if history == nil {
		t.Fatal("expected user history to exist")
	}
	if history.TotalVPNSessions != 1 {
		t.Errorf("expected 1 session, got %d", history.TotalVPNSessions)
	}
	if len(history.ProvidersUsed) != 1 || history.ProvidersUsed[0] != "nordvpn" {
		t.Error("expected nordvpn in providers used")
	}

	// Use different provider
	event2 := &DetectionEvent{
		UserID:       1,
		Username:     "testuser",
		IPAddress:    "203.0.113.50",
		LocationType: "wan",
		Timestamp:    time.Now(),
	}

	alert2, _ := detector.Check(ctx, event2)
	if alert2 == nil {
		t.Fatal("expected alert for new provider")
	}

	var metadata2 VPNUsageMetadata
	json.Unmarshal(alert2.Metadata, &metadata2)
	if metadata2.PreviousProvider != "nordvpn" {
		t.Errorf("expected previous provider nordvpn, got %s", metadata2.PreviousProvider)
	}

	// Verify updated history
	history = detector.GetUserHistory(1)
	if history.TotalVPNSessions != 2 {
		t.Errorf("expected 2 sessions, got %d", history.TotalVPNSessions)
	}
	if len(history.ProvidersUsed) != 2 {
		t.Errorf("expected 2 providers, got %d", len(history.ProvidersUsed))
	}
}

func TestVPNUsageDetector_NoAlertOnSameProvider(t *testing.T) {
	lookup := vpn.NewLookup()
	lookup.AddServer(&vpn.Server{
		Provider: "nordvpn",
		IPs:      []string{"198.51.100.1", "198.51.100.2"},
	})
	lookup.AddProvider(&vpn.Provider{Name: "nordvpn", DisplayName: "NordVPN"})

	detector := NewVPNUsageDetectorForTest(lookup)

	// Disable first use alerts, enable new provider alerts
	config := DefaultVPNUsageConfig()
	config.AlertOnFirstUse = true
	config.AlertOnNewProvider = true
	configJSON, _ := json.Marshal(config)
	detector.Configure(configJSON)

	ctx := context.Background()

	// First use
	event1 := &DetectionEvent{
		UserID:       1,
		Username:     "testuser",
		IPAddress:    "198.51.100.1",
		LocationType: "wan",
		Timestamp:    time.Now(),
	}
	detector.Check(ctx, event1)

	// Same provider, different IP
	event2 := &DetectionEvent{
		UserID:       1,
		Username:     "testuser",
		IPAddress:    "198.51.100.2",
		LocationType: "wan",
		Timestamp:    time.Now(),
	}

	alert, _ := detector.Check(ctx, event2)
	if alert != nil {
		t.Error("expected no alert for same provider")
	}
}

func TestVPNUsageDetector_Disabled(t *testing.T) {
	lookup := vpn.NewLookup()
	lookup.AddServer(&vpn.Server{
		Provider: "nordvpn",
		IPs:      []string{"198.51.100.1"},
	})

	detector := NewVPNUsageDetectorForTest(lookup)
	detector.SetEnabled(false)

	ctx := context.Background()
	event := &DetectionEvent{
		UserID:       1,
		Username:     "testuser",
		IPAddress:    "198.51.100.1",
		LocationType: "wan",
		Timestamp:    time.Now(),
	}

	alert, err := detector.Check(ctx, event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if alert != nil {
		t.Error("expected no alert when detector is disabled")
	}
}

func TestVPNUsageDetector_Configure(t *testing.T) {
	detector := NewVPNUsageDetector(nil)

	// Valid configuration
	validConfig := `{
		"severity": "warning",
		"alert_on_first_use": false,
		"alert_on_new_provider": true,
		"excluded_providers": ["corporate_vpn"],
		"excluded_users": [100, 200]
	}`

	err := detector.Configure([]byte(validConfig))
	if err != nil {
		t.Fatalf("unexpected error for valid config: %v", err)
	}

	config := detector.Config()
	if config.Severity != SeverityWarning {
		t.Errorf("expected severity warning, got %s", config.Severity)
	}
	if config.AlertOnFirstUse {
		t.Error("expected AlertOnFirstUse to be false")
	}
	if !config.AlertOnNewProvider {
		t.Error("expected AlertOnNewProvider to be true")
	}
	if len(config.ExcludedProviders) != 1 || config.ExcludedProviders[0] != "corporate_vpn" {
		t.Error("expected excluded provider corporate_vpn")
	}
	if len(config.ExcludedUsers) != 2 {
		t.Errorf("expected 2 excluded users, got %d", len(config.ExcludedUsers))
	}

	// Invalid configuration
	invalidConfig := `{"severity": "invalid_level"}`
	err = detector.Configure([]byte(invalidConfig))
	if err == nil {
		t.Error("expected error for invalid severity")
	}
}

func TestVPNUsageDetector_ClearHistory(t *testing.T) {
	lookup := vpn.NewLookup()
	lookup.AddServer(&vpn.Server{
		Provider: "nordvpn",
		IPs:      []string{"198.51.100.1"},
	})
	lookup.AddProvider(&vpn.Provider{Name: "nordvpn", DisplayName: "NordVPN"})

	detector := NewVPNUsageDetectorForTest(lookup)

	ctx := context.Background()
	event := &DetectionEvent{
		UserID:       1,
		Username:     "testuser",
		IPAddress:    "198.51.100.1",
		LocationType: "wan",
		Timestamp:    time.Now(),
	}

	detector.Check(ctx, event)

	// Verify history exists
	if detector.GetUserHistory(1) == nil {
		t.Fatal("expected history to exist")
	}

	// Clear user history
	detector.ClearUserHistory(1)
	if detector.GetUserHistory(1) != nil {
		t.Error("expected history to be cleared")
	}

	// Add history again
	detector.Check(ctx, event)
	detector.Check(ctx, &DetectionEvent{
		UserID:       2,
		Username:     "user2",
		IPAddress:    "198.51.100.1",
		LocationType: "wan",
		Timestamp:    time.Now(),
	})

	// Clear all history
	detector.ClearAllHistory()
	if detector.GetUserHistory(1) != nil || detector.GetUserHistory(2) != nil {
		t.Error("expected all history to be cleared")
	}
}

func TestVPNUsageDetector_EnableDisable(t *testing.T) {
	detector := NewVPNUsageDetector(nil)

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

func TestVPNUsageDetector_NoIPAddress(t *testing.T) {
	lookup := vpn.NewLookup()
	detector := NewVPNUsageDetectorForTest(lookup)

	ctx := context.Background()
	event := &DetectionEvent{
		UserID:       1,
		Username:     "testuser",
		IPAddress:    "", // Empty IP
		LocationType: "wan",
		Timestamp:    time.Now(),
	}

	alert, err := detector.Check(ctx, event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if alert != nil {
		t.Error("expected no alert for empty IP")
	}
}
