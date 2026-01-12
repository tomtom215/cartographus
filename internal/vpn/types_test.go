// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package vpn

import (
	"encoding/json"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if !config.Enabled {
		t.Error("expected Enabled to be true by default")
	}
	if config.CacheSize != 10000 {
		t.Errorf("CacheSize = %d, want 10000", config.CacheSize)
	}
	if config.AutoUpdate {
		t.Error("expected AutoUpdate to be false by default")
	}
	if config.UpdateInterval != 24*time.Hour {
		t.Errorf("UpdateInterval = %v, want 24h", config.UpdateInterval)
	}
}

// TestGetDisplayName is in lookup_test.go

func TestConfig_MarshalJSON(t *testing.T) {
	config := Config{
		Enabled:        true,
		CacheSize:      5000,
		DataFile:       "/path/to/data.json",
		AutoUpdate:     true,
		UpdateInterval: 12 * time.Hour,
	}

	data, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Unmarshal to verify structure
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	if result["enabled"] != true {
		t.Errorf("enabled = %v, want true", result["enabled"])
	}
	if result["cache_size"].(float64) != 5000 {
		t.Errorf("cache_size = %v, want 5000", result["cache_size"])
	}
	if result["update_interval"] != "12h0m0s" {
		t.Errorf("update_interval = %v, want 12h0m0s", result["update_interval"])
	}
	if result["auto_update"] != true {
		t.Errorf("auto_update = %v, want true", result["auto_update"])
	}
}

func TestConfig_MarshalJSON_ZeroDuration(t *testing.T) {
	config := Config{
		Enabled:        false,
		UpdateInterval: 0,
	}

	data, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	// Zero duration should marshal as "0s"
	if result["update_interval"] != "0s" {
		t.Errorf("update_interval = %v, want 0s", result["update_interval"])
	}
}

func TestServer_Fields(t *testing.T) {
	server := Server{
		Provider:   "nordvpn",
		Country:    "United States",
		Region:     "North America",
		City:       "New York",
		Hostname:   "us-ny-001.nordvpn.com",
		ServerName: "Server 001",
		IPs:        []string{"1.2.3.4", "5.6.7.8"},
		VPNType:    "openvpn",
		ISP:        "Example ISP",
		Categories: []string{"P2P", "Double VPN"},
		TCP:        true,
		UDP:        true,
		Number:     1,
	}

	if server.Provider != "nordvpn" {
		t.Errorf("Provider = %q, want nordvpn", server.Provider)
	}
	if server.Country != "United States" {
		t.Errorf("Country = %q, want United States", server.Country)
	}
	if server.Region != "North America" {
		t.Errorf("Region = %q, want North America", server.Region)
	}
	if server.City != "New York" {
		t.Errorf("City = %q, want New York", server.City)
	}
	if server.Hostname != "us-ny-001.nordvpn.com" {
		t.Errorf("Hostname = %q, want us-ny-001.nordvpn.com", server.Hostname)
	}
	if server.ServerName != "Server 001" {
		t.Errorf("ServerName = %q, want Server 001", server.ServerName)
	}
	if server.ISP != "Example ISP" {
		t.Errorf("ISP = %q, want Example ISP", server.ISP)
	}
	if len(server.IPs) != 2 {
		t.Errorf("IPs length = %d, want 2", len(server.IPs))
	}
	if !server.TCP || !server.UDP {
		t.Error("expected TCP and UDP to be true")
	}
	if len(server.Categories) != 2 {
		t.Errorf("Categories length = %d, want 2", len(server.Categories))
	}
	if server.VPNType != "openvpn" {
		t.Errorf("VPNType = %q, want openvpn", server.VPNType)
	}
	if server.Number != 1 {
		t.Errorf("Number = %d, want 1", server.Number)
	}
}

func TestLookupResult_Fields(t *testing.T) {
	result := LookupResult{
		IsVPN:               true,
		Provider:            "expressvpn",
		ProviderDisplayName: "ExpressVPN",
		ServerHostname:      "us-ny-001.expressvpn.com",
		ServerCountry:       "United States",
		ServerCity:          "New York",
		Confidence:          100,
	}

	if !result.IsVPN {
		t.Error("expected IsVPN to be true")
	}
	if result.Provider != "expressvpn" {
		t.Errorf("Provider = %q, want expressvpn", result.Provider)
	}
	if result.ProviderDisplayName != "ExpressVPN" {
		t.Errorf("ProviderDisplayName = %q, want ExpressVPN", result.ProviderDisplayName)
	}
	if result.ServerHostname != "us-ny-001.expressvpn.com" {
		t.Errorf("ServerHostname = %q, want us-ny-001.expressvpn.com", result.ServerHostname)
	}
	if result.ServerCountry != "United States" {
		t.Errorf("ServerCountry = %q, want United States", result.ServerCountry)
	}
	if result.ServerCity != "New York" {
		t.Errorf("ServerCity = %q, want New York", result.ServerCity)
	}
	if result.Confidence != 100 {
		t.Errorf("Confidence = %d, want 100", result.Confidence)
	}
}

func TestProvider_Fields(t *testing.T) {
	now := time.Now()
	provider := Provider{
		Name:        "mullvad",
		DisplayName: "Mullvad",
		Website:     "https://mullvad.net",
		Version:     5,
		ServerCount: 500,
		IPCount:     1000,
		LastUpdated: now,
		Timestamp:   now.Unix(),
	}

	if provider.Name != "mullvad" {
		t.Errorf("Name = %q, want mullvad", provider.Name)
	}
	if provider.DisplayName != "Mullvad" {
		t.Errorf("DisplayName = %q, want Mullvad", provider.DisplayName)
	}
	if provider.Website != "https://mullvad.net" {
		t.Errorf("Website = %q, want https://mullvad.net", provider.Website)
	}
	if provider.Version != 5 {
		t.Errorf("Version = %d, want 5", provider.Version)
	}
	if provider.ServerCount != 500 {
		t.Errorf("ServerCount = %d, want 500", provider.ServerCount)
	}
	if provider.IPCount != 1000 {
		t.Errorf("IPCount = %d, want 1000", provider.IPCount)
	}
	if provider.Timestamp != now.Unix() {
		t.Errorf("Timestamp = %d, want %d", provider.Timestamp, now.Unix())
	}
	if provider.LastUpdated != now {
		t.Errorf("LastUpdated = %v, want %v", provider.LastUpdated, now)
	}
}

func TestStats_Fields(t *testing.T) {
	now := time.Now()
	stats := Stats{
		TotalProviders: 10,
		TotalServers:   5000,
		TotalIPs:       20000,
		IPv4Count:      15000,
		IPv6Count:      5000,
		LastUpdated:    now,
		ProviderStats: []Provider{
			{Name: "nordvpn", IPCount: 5000},
			{Name: "mullvad", IPCount: 3000},
		},
	}

	if stats.TotalProviders != 10 {
		t.Errorf("TotalProviders = %d, want 10", stats.TotalProviders)
	}
	if stats.TotalServers != 5000 {
		t.Errorf("TotalServers = %d, want 5000", stats.TotalServers)
	}
	if stats.TotalIPs != 20000 {
		t.Errorf("TotalIPs = %d, want 20000", stats.TotalIPs)
	}
	if stats.IPv4Count != 15000 {
		t.Errorf("IPv4Count = %d, want 15000", stats.IPv4Count)
	}
	if stats.IPv6Count != 5000 {
		t.Errorf("IPv6Count = %d, want 5000", stats.IPv6Count)
	}
	if stats.LastUpdated != now {
		t.Errorf("LastUpdated = %v, want %v", stats.LastUpdated, now)
	}
	if len(stats.ProviderStats) != 2 {
		t.Errorf("ProviderStats length = %d, want 2", len(stats.ProviderStats))
	}
}

func TestImportResult_Fields(t *testing.T) {
	result := ImportResult{
		ProvidersImported: 5,
		ServersImported:   1000,
		IPsImported:       5000,
		Errors:            []string{"warning: skipped invalid IP"},
		Duration:          500 * time.Millisecond,
	}

	if result.ProvidersImported != 5 {
		t.Errorf("ProvidersImported = %d, want 5", result.ProvidersImported)
	}
	if result.ServersImported != 1000 {
		t.Errorf("ServersImported = %d, want 1000", result.ServersImported)
	}
	if result.IPsImported != 5000 {
		t.Errorf("IPsImported = %d, want 5000", result.IPsImported)
	}
	if len(result.Errors) != 1 {
		t.Errorf("Errors length = %d, want 1", len(result.Errors))
	}
	if result.Duration != 500*time.Millisecond {
		t.Errorf("Duration = %v, want 500ms", result.Duration)
	}
}

func TestGluetunProvider_Fields(t *testing.T) {
	provider := GluetunProvider{
		Version:   3,
		Timestamp: 1234567890,
		Servers: []GluetunServer{
			{
				Country:  "United States",
				City:     "New York",
				Hostname: "us-ny-001.example.com",
				IPs:      []string{"1.2.3.4"},
				TCP:      true,
				UDP:      true,
			},
		},
	}

	if provider.Version != 3 {
		t.Errorf("Version = %d, want 3", provider.Version)
	}
	if provider.Timestamp != 1234567890 {
		t.Errorf("Timestamp = %d, want 1234567890", provider.Timestamp)
	}
	if len(provider.Servers) != 1 {
		t.Errorf("Servers length = %d, want 1", len(provider.Servers))
	}
}

func TestGluetunServer_Fields(t *testing.T) {
	server := GluetunServer{
		VPN:        "openvpn",
		Country:    "Germany",
		Region:     "Europe",
		City:       "Berlin",
		ISP:        "Example ISP",
		Owned:      true,
		Number:     42,
		ServerName: "Berlin-42",
		Hostname:   "de-ber-042.example.com",
		TCP:        true,
		UDP:        true,
		WgPubKey:   "abc123pubkey",
		IPs:        []string{"10.0.0.1", "10.0.0.2"},
		Categories: []string{"P2P"},
	}

	if server.VPN != "openvpn" {
		t.Errorf("VPN = %q, want openvpn", server.VPN)
	}
	if server.Country != "Germany" {
		t.Errorf("Country = %q, want Germany", server.Country)
	}
	if server.Region != "Europe" {
		t.Errorf("Region = %q, want Europe", server.Region)
	}
	if server.City != "Berlin" {
		t.Errorf("City = %q, want Berlin", server.City)
	}
	if server.ISP != "Example ISP" {
		t.Errorf("ISP = %q, want Example ISP", server.ISP)
	}
	if server.ServerName != "Berlin-42" {
		t.Errorf("ServerName = %q, want Berlin-42", server.ServerName)
	}
	if server.Hostname != "de-ber-042.example.com" {
		t.Errorf("Hostname = %q, want de-ber-042.example.com", server.Hostname)
	}
	if !server.Owned {
		t.Error("expected Owned to be true")
	}
	if server.Number != 42 {
		t.Errorf("Number = %d, want 42", server.Number)
	}
	if !server.TCP || !server.UDP {
		t.Error("expected TCP and UDP to be true")
	}
	if len(server.IPs) != 2 {
		t.Errorf("IPs length = %d, want 2", len(server.IPs))
	}
	if len(server.Categories) != 1 {
		t.Errorf("Categories length = %d, want 1", len(server.Categories))
	}
	if server.WgPubKey != "abc123pubkey" {
		t.Errorf("WgPubKey = %q, want abc123pubkey", server.WgPubKey)
	}
}

func TestGluetunData_Map(t *testing.T) {
	data := GluetunData{
		"nordvpn": {
			Version:   1,
			Timestamp: 1234567890,
			Servers: []GluetunServer{
				{Country: "USA", IPs: []string{"1.1.1.1"}},
			},
		},
		"mullvad": {
			Version:   2,
			Timestamp: 1234567891,
			Servers: []GluetunServer{
				{Country: "Sweden", IPs: []string{"2.2.2.2"}},
			},
		},
	}

	if len(data) != 2 {
		t.Errorf("GluetunData length = %d, want 2", len(data))
	}

	if nord, ok := data["nordvpn"]; !ok {
		t.Error("expected nordvpn in GluetunData")
	} else {
		if nord.Version != 1 {
			t.Errorf("nordvpn Version = %d, want 1", nord.Version)
		}
		if len(nord.Servers) != 1 {
			t.Errorf("nordvpn Servers length = %d, want 1", len(nord.Servers))
		}
	}

	if mull, ok := data["mullvad"]; !ok {
		t.Error("expected mullvad in GluetunData")
	} else {
		if mull.Version != 2 {
			t.Errorf("mullvad Version = %d, want 2", mull.Version)
		}
	}
}

func TestConfig_DefaultValues(t *testing.T) {
	// Test zero-value config
	config := Config{}

	if config.Enabled {
		t.Error("zero-value Enabled should be false")
	}
	if config.CacheSize != 0 {
		t.Errorf("zero-value CacheSize = %d, want 0", config.CacheSize)
	}
	if config.AutoUpdate {
		t.Error("zero-value AutoUpdate should be false")
	}
	if config.UpdateInterval != 0 {
		t.Errorf("zero-value UpdateInterval = %v, want 0", config.UpdateInterval)
	}
	if config.DataFile != "" {
		t.Errorf("zero-value DataFile = %q, want empty", config.DataFile)
	}
}

func TestLookupResult_NotVPN(t *testing.T) {
	result := LookupResult{
		IsVPN:      false,
		Confidence: 0,
	}

	if result.IsVPN {
		t.Error("expected IsVPN to be false")
	}
	if result.Provider != "" {
		t.Errorf("Provider = %q, want empty", result.Provider)
	}
	if result.ProviderDisplayName != "" {
		t.Errorf("ProviderDisplayName = %q, want empty", result.ProviderDisplayName)
	}
	if result.Confidence != 0 {
		t.Errorf("Confidence = %d, want 0", result.Confidence)
	}
}
