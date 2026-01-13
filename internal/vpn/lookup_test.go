// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package vpn

import (
	"testing"
)

func TestNewLookup(t *testing.T) {
	lookup := NewLookup()
	if lookup == nil {
		t.Fatal("expected non-nil lookup")
	}
	if lookup.Count() != 0 {
		t.Errorf("expected empty lookup, got %d entries", lookup.Count())
	}
}

func TestLookup_AddServer(t *testing.T) {
	lookup := NewLookup()

	server := &Server{
		Provider: "nordvpn",
		Country:  "United States",
		City:     "New York",
		Hostname: "us-nyc-001.nordvpn.com",
		IPs:      []string{"198.51.100.1", "198.51.100.2"},
	}

	err := lookup.AddServer(server)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if lookup.Count() != 2 {
		t.Errorf("expected 2 IPs, got %d", lookup.Count())
	}
}

func TestLookup_LookupIP_Found(t *testing.T) {
	lookup := NewLookup()

	server := &Server{
		Provider:   "expressvpn",
		Country:    "Germany",
		City:       "Frankfurt",
		Hostname:   "de-frankfurt-1.expressvpn.com",
		ServerName: "Frankfurt 1",
		IPs:        []string{"203.0.113.50"},
	}

	if err := lookup.AddServer(server); err != nil {
		t.Fatalf("AddServer() error = %v", err)
	}

	result := lookup.LookupIP("203.0.113.50")
	if !result.IsVPN {
		t.Error("expected IsVPN to be true")
	}
	if result.Provider != "expressvpn" {
		t.Errorf("expected provider expressvpn, got %s", result.Provider)
	}
	if result.ProviderDisplayName != "ExpressVPN" {
		t.Errorf("expected display name ExpressVPN, got %s", result.ProviderDisplayName)
	}
	if result.ServerCountry != "Germany" {
		t.Errorf("expected country Germany, got %s", result.ServerCountry)
	}
	if result.ServerCity != "Frankfurt" {
		t.Errorf("expected city Frankfurt, got %s", result.ServerCity)
	}
	if result.Confidence != 100 {
		t.Errorf("expected confidence 100, got %d", result.Confidence)
	}
}

func TestLookup_LookupIP_NotFound(t *testing.T) {
	lookup := NewLookup()

	server := &Server{
		Provider: "nordvpn",
		Country:  "Sweden",
		IPs:      []string{"192.0.2.1"},
	}

	_ = lookup.AddServer(server)

	// Lookup a different IP
	result := lookup.LookupIP("192.0.2.99")
	if result.IsVPN {
		t.Error("expected IsVPN to be false for unknown IP")
	}
	if result.Confidence != 100 {
		t.Errorf("expected confidence 100 for negative result, got %d", result.Confidence)
	}
}

func TestLookup_LookupIP_InvalidIP(t *testing.T) {
	lookup := NewLookup()

	result := lookup.LookupIP("not-an-ip")
	if result.IsVPN {
		t.Error("expected IsVPN to be false for invalid IP")
	}
	if result.Confidence != 0 {
		t.Errorf("expected confidence 0 for invalid IP, got %d", result.Confidence)
	}
}

func TestLookup_LookupIP_IPv6(t *testing.T) {
	lookup := NewLookup()

	server := &Server{
		Provider: "mullvad",
		Country:  "Netherlands",
		City:     "Amsterdam",
		IPs:      []string{"2001:db8::1", "2001:db8::2"},
	}

	_ = lookup.AddServer(server)

	result := lookup.LookupIP("2001:db8::1")
	if !result.IsVPN {
		t.Error("expected IsVPN to be true for IPv6")
	}
	if result.Provider != "mullvad" {
		t.Errorf("expected provider mullvad, got %s", result.Provider)
	}

	// Check stats
	stats := lookup.GetStats()
	if stats.IPv6Count != 2 {
		t.Errorf("expected 2 IPv6 addresses, got %d", stats.IPv6Count)
	}
}

func TestLookup_ContainsIP(t *testing.T) {
	lookup := NewLookup()

	server := &Server{
		Provider: "protonvpn",
		IPs:      []string{"198.51.100.10"},
	}

	_ = lookup.AddServer(server)

	if !lookup.ContainsIP("198.51.100.10") {
		t.Error("expected ContainsIP to return true for known IP")
	}
	if lookup.ContainsIP("198.51.100.99") {
		t.Error("expected ContainsIP to return false for unknown IP")
	}
	if lookup.ContainsIP("invalid") {
		t.Error("expected ContainsIP to return false for invalid IP")
	}
}

func TestLookup_AddProvider(t *testing.T) {
	lookup := NewLookup()

	provider := &Provider{
		Name:        "surfshark",
		DisplayName: "Surfshark",
		Website:     "https://surfshark.com",
		ServerCount: 100,
		IPCount:     500,
	}

	lookup.AddProvider(provider)

	retrieved := lookup.GetProvider("surfshark")
	if retrieved == nil {
		t.Fatal("expected to retrieve provider")
	}
	if retrieved.DisplayName != "Surfshark" {
		t.Errorf("expected display name Surfshark, got %s", retrieved.DisplayName)
	}

	stats := lookup.GetStats()
	if stats.TotalProviders != 1 {
		t.Errorf("expected 1 provider, got %d", stats.TotalProviders)
	}
}

func TestLookup_ListProviders(t *testing.T) {
	lookup := NewLookup()

	providers := []Provider{
		{Name: "nordvpn", DisplayName: "NordVPN"},
		{Name: "expressvpn", DisplayName: "ExpressVPN"},
		{Name: "surfshark", DisplayName: "Surfshark"},
	}

	for _, p := range providers {
		pCopy := p
		lookup.AddProvider(&pCopy)
	}

	list := lookup.ListProviders()
	if len(list) != 3 {
		t.Errorf("expected 3 providers, got %d", len(list))
	}
}

func TestLookup_Clear(t *testing.T) {
	lookup := NewLookup()

	server := &Server{
		Provider: "nordvpn",
		IPs:      []string{"192.0.2.1", "192.0.2.2"},
	}
	_ = lookup.AddServer(server)

	provider := &Provider{Name: "nordvpn", DisplayName: "NordVPN"}
	lookup.AddProvider(provider)

	if lookup.Count() != 2 {
		t.Errorf("expected 2 IPs before clear, got %d", lookup.Count())
	}

	lookup.Clear()

	if lookup.Count() != 0 {
		t.Errorf("expected 0 IPs after clear, got %d", lookup.Count())
	}
	if len(lookup.ListProviders()) != 0 {
		t.Error("expected no providers after clear")
	}
}

func TestLookup_GetStats(t *testing.T) {
	lookup := NewLookup()

	// Add some test data
	server1 := &Server{
		Provider: "nordvpn",
		IPs:      []string{"192.0.2.1", "192.0.2.2"},
	}
	server2 := &Server{
		Provider: "expressvpn",
		IPs:      []string{"2001:db8::1"},
	}

	_ = lookup.AddServer(server1)
	_ = lookup.AddServer(server2)

	lookup.AddProvider(&Provider{Name: "nordvpn", IPCount: 2})
	lookup.AddProvider(&Provider{Name: "expressvpn", IPCount: 1})

	stats := lookup.GetStats()

	if stats.TotalIPs != 3 {
		t.Errorf("expected 3 total IPs, got %d", stats.TotalIPs)
	}
	if stats.IPv4Count != 2 {
		t.Errorf("expected 2 IPv4 addresses, got %d", stats.IPv4Count)
	}
	if stats.IPv6Count != 1 {
		t.Errorf("expected 1 IPv6 address, got %d", stats.IPv6Count)
	}
	if stats.TotalServers != 2 {
		t.Errorf("expected 2 servers, got %d", stats.TotalServers)
	}
	if stats.TotalProviders != 2 {
		t.Errorf("expected 2 providers, got %d", stats.TotalProviders)
	}
}

func TestGetDisplayName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"nordvpn", "NordVPN"},
		{"expressvpn", "ExpressVPN"},
		{"mullvad", "Mullvad"},
		{"protonvpn", "ProtonVPN"},
		{"surfshark", "Surfshark"},
		{"unknown_provider", "unknown_provider"}, // Returns input if not found
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := GetDisplayName(tt.input)
			if result != tt.expected {
				t.Errorf("GetDisplayName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestLookup_GetProvider_NotFound(t *testing.T) {
	lookup := NewLookup()

	// Look up non-existent provider
	provider := lookup.GetProvider("nonexistent")
	if provider != nil {
		t.Error("expected nil for non-existent provider")
	}
}

func TestLookup_AddServer_EmptyIPs(t *testing.T) {
	lookup := NewLookup()

	server := &Server{
		Provider: "test",
		Country:  "Test",
		IPs:      []string{},
	}

	err := lookup.AddServer(server)
	if err != nil {
		t.Fatalf("unexpected error for server with no IPs: %v", err)
	}

	// Count should remain 0
	if lookup.Count() != 0 {
		t.Errorf("expected 0 IPs for empty server, got %d", lookup.Count())
	}
}

func TestLookup_AddServer_InvalidIP(t *testing.T) {
	lookup := NewLookup()

	server := &Server{
		Provider: "test",
		Country:  "Test",
		IPs:      []string{"not-a-valid-ip"},
	}

	// AddServer skips invalid IPs silently - no error returned
	err := lookup.AddServer(server)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// No IPs should be added
	if lookup.Count() != 0 {
		t.Errorf("expected 0 IPs (invalid IP skipped), got %d", lookup.Count())
	}
}

func TestLookup_AddServer_MixedValidInvalidIPs(t *testing.T) {
	lookup := NewLookup()

	server := &Server{
		Provider: "test",
		Country:  "Test",
		IPs:      []string{"192.0.2.1", "not-valid", "192.0.2.2"},
	}

	// AddServer skips invalid IPs but adds valid ones
	err := lookup.AddServer(server)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Only valid IPs should be added
	if lookup.Count() != 2 {
		t.Errorf("expected 2 IPs (skipping invalid), got %d", lookup.Count())
	}

	// Verify valid IPs are present
	if !lookup.ContainsIP("192.0.2.1") {
		t.Error("expected 192.0.2.1 to be in lookup")
	}
	if !lookup.ContainsIP("192.0.2.2") {
		t.Error("expected 192.0.2.2 to be in lookup")
	}
}

func TestLookup_AddProvider_Update(t *testing.T) {
	lookup := NewLookup()

	// Add initial provider
	provider1 := &Provider{
		Name:        "test",
		DisplayName: "Test Provider",
		ServerCount: 10,
		IPCount:     100,
	}
	lookup.AddProvider(provider1)

	// Update provider with new data
	provider2 := &Provider{
		Name:        "test",
		DisplayName: "Updated Provider",
		ServerCount: 20,
		IPCount:     200,
	}
	lookup.AddProvider(provider2)

	// Should have updated data
	retrieved := lookup.GetProvider("test")
	if retrieved == nil {
		t.Fatal("expected provider to exist")
	}
	if retrieved.DisplayName != "Updated Provider" {
		t.Errorf("expected updated display name, got %s", retrieved.DisplayName)
	}
	if retrieved.ServerCount != 20 {
		t.Errorf("expected updated server count 20, got %d", retrieved.ServerCount)
	}

	// Should still only have 1 provider
	stats := lookup.GetStats()
	if stats.TotalProviders != 1 {
		t.Errorf("expected 1 provider after update, got %d", stats.TotalProviders)
	}
}

func TestLookup_ContainsIP_Empty(t *testing.T) {
	lookup := NewLookup()

	// Empty string should return false
	if lookup.ContainsIP("") {
		t.Error("expected ContainsIP to return false for empty string")
	}
}

func TestLookup_LookupIP_Empty(t *testing.T) {
	lookup := NewLookup()

	result := lookup.LookupIP("")
	if result.IsVPN {
		t.Error("expected IsVPN to be false for empty IP")
	}
	if result.Confidence != 0 {
		t.Errorf("expected confidence 0 for empty IP, got %d", result.Confidence)
	}
}

func TestLookup_Stats_Empty(t *testing.T) {
	lookup := NewLookup()

	stats := lookup.GetStats()

	if stats.TotalProviders != 0 {
		t.Errorf("expected 0 providers, got %d", stats.TotalProviders)
	}
	if stats.TotalServers != 0 {
		t.Errorf("expected 0 servers, got %d", stats.TotalServers)
	}
	if stats.TotalIPs != 0 {
		t.Errorf("expected 0 IPs, got %d", stats.TotalIPs)
	}
	if stats.IPv4Count != 0 {
		t.Errorf("expected 0 IPv4, got %d", stats.IPv4Count)
	}
	if stats.IPv6Count != 0 {
		t.Errorf("expected 0 IPv6, got %d", stats.IPv6Count)
	}
}

func TestLookup_LookupIP_Hostname(t *testing.T) {
	lookup := NewLookup()

	server := &Server{
		Provider: "test",
		Country:  "Test Country",
		City:     "Test City",
		Hostname: "server.example.com",
		IPs:      []string{"192.0.2.1"},
	}
	_ = lookup.AddServer(server)

	result := lookup.LookupIP("192.0.2.1")
	if !result.IsVPN {
		t.Error("expected IsVPN to be true")
	}
	if result.ServerHostname != "server.example.com" {
		t.Errorf("expected hostname 'server.example.com', got %s", result.ServerHostname)
	}
}
