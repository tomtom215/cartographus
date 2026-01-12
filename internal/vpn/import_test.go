// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package vpn

import (
	"os"
	"strings"
	"testing"
)

func TestImporter_ImportFromReader(t *testing.T) {
	lookup := NewLookup()
	importer := NewImporter(lookup)

	// Sample gluetun format data
	jsonData := `{
		"airvpn": {
			"version": 1,
			"timestamp": 1721997873,
			"servers": [
				{
					"vpn": "wireguard",
					"country": "Austria",
					"region": "Europe",
					"city": "Vienna",
					"server_name": "Alderamin",
					"hostname": "at.vpn.airdns.org",
					"wgpubkey": "test-key",
					"ips": ["203.0.113.1"]
				},
				{
					"vpn": "openvpn",
					"country": "Germany",
					"region": "Europe",
					"city": "Frankfurt",
					"server_name": "Bellatrix",
					"hostname": "de.vpn.airdns.org",
					"tcp": true,
					"udp": true,
					"ips": ["203.0.113.2", "203.0.113.3"]
				}
			]
		}
	}`

	result, err := importer.ImportFromReader(strings.NewReader(jsonData))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ProvidersImported != 1 {
		t.Errorf("expected 1 provider, got %d", result.ProvidersImported)
	}
	if result.ServersImported != 2 {
		t.Errorf("expected 2 servers, got %d", result.ServersImported)
	}
	if result.IPsImported != 3 {
		t.Errorf("expected 3 IPs, got %d", result.IPsImported)
	}
	if result.Duration <= 0 {
		t.Error("expected positive duration")
	}

	// Verify lookup works
	vpnResult := lookup.LookupIP("203.0.113.1")
	if !vpnResult.IsVPN {
		t.Error("expected IP to be detected as VPN")
	}
	if vpnResult.Provider != "airvpn" {
		t.Errorf("expected provider airvpn, got %s", vpnResult.Provider)
	}
	if vpnResult.ServerCountry != "Austria" {
		t.Errorf("expected country Austria, got %s", vpnResult.ServerCountry)
	}
	if vpnResult.ServerCity != "Vienna" {
		t.Errorf("expected city Vienna, got %s", vpnResult.ServerCity)
	}

	// Verify provider metadata
	provider := lookup.GetProvider("airvpn")
	if provider == nil {
		t.Fatal("expected provider to exist")
	}
	if provider.Version != 1 {
		t.Errorf("expected version 1, got %d", provider.Version)
	}
	if provider.Timestamp != 1721997873 {
		t.Errorf("expected timestamp 1721997873, got %d", provider.Timestamp)
	}
}

func TestImporter_ImportFromBytes(t *testing.T) {
	lookup := NewLookup()
	importer := NewImporter(lookup)

	jsonData := []byte(`{
		"nordvpn": {
			"version": 2,
			"timestamp": 1700000000,
			"servers": [
				{
					"country": "United States",
					"city": "New York",
					"hostname": "us-nyc1.nordvpn.com",
					"ips": ["198.51.100.1", "198.51.100.2"]
				}
			]
		}
	}`)

	result, err := importer.ImportFromBytes(jsonData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ProvidersImported != 1 {
		t.Errorf("expected 1 provider, got %d", result.ProvidersImported)
	}
	if result.IPsImported != 2 {
		t.Errorf("expected 2 IPs, got %d", result.IPsImported)
	}

	// Verify lookup
	if !lookup.ContainsIP("198.51.100.1") {
		t.Error("expected IP to be in lookup")
	}
}

func TestImporter_ImportMultipleProviders(t *testing.T) {
	lookup := NewLookup()
	importer := NewImporter(lookup)

	jsonData := `{
		"nordvpn": {
			"version": 1,
			"timestamp": 1700000000,
			"servers": [
				{
					"country": "Sweden",
					"hostname": "se1.nordvpn.com",
					"ips": ["192.0.2.1"]
				}
			]
		},
		"expressvpn": {
			"version": 1,
			"timestamp": 1700000001,
			"servers": [
				{
					"country": "Netherlands",
					"hostname": "nl1.expressvpn.com",
					"ips": ["192.0.2.2"]
				}
			]
		},
		"mullvad": {
			"version": 1,
			"timestamp": 1700000002,
			"servers": [
				{
					"country": "Germany",
					"hostname": "de1.mullvad.net",
					"ips": ["192.0.2.3"]
				}
			]
		}
	}`

	result, err := importer.ImportFromReader(strings.NewReader(jsonData))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ProvidersImported != 3 {
		t.Errorf("expected 3 providers, got %d", result.ProvidersImported)
	}
	if result.IPsImported != 3 {
		t.Errorf("expected 3 IPs, got %d", result.IPsImported)
	}

	// Verify each provider's IP
	tests := []struct {
		ip       string
		provider string
	}{
		{"192.0.2.1", "nordvpn"},
		{"192.0.2.2", "expressvpn"},
		{"192.0.2.3", "mullvad"},
	}

	for _, tt := range tests {
		result := lookup.LookupIP(tt.ip)
		if !result.IsVPN {
			t.Errorf("expected %s to be VPN", tt.ip)
		}
		if result.Provider != tt.provider {
			t.Errorf("expected provider %s for %s, got %s", tt.provider, tt.ip, result.Provider)
		}
	}
}

func TestImporter_ImportClearsExistingData(t *testing.T) {
	lookup := NewLookup()
	importer := NewImporter(lookup)

	// Add initial data
	initialServer := &Server{
		Provider: "old_provider",
		IPs:      []string{"10.0.0.1"},
	}
	lookup.AddServer(initialServer)

	if lookup.Count() != 1 {
		t.Fatalf("expected 1 IP before import, got %d", lookup.Count())
	}

	// Import new data
	jsonData := `{
		"new_provider": {
			"version": 1,
			"timestamp": 1700000000,
			"servers": [
				{"country": "Test", "ips": ["10.0.0.2"]}
			]
		}
	}`

	_, err := importer.ImportFromReader(strings.NewReader(jsonData))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Old IP should be gone
	if lookup.ContainsIP("10.0.0.1") {
		t.Error("expected old IP to be cleared after import")
	}

	// New IP should be present
	if !lookup.ContainsIP("10.0.0.2") {
		t.Error("expected new IP to be present after import")
	}
}

func TestImporter_MergeFromReader(t *testing.T) {
	lookup := NewLookup()
	importer := NewImporter(lookup)

	// Add initial data directly
	initialServer := &Server{
		Provider: "initial_provider",
		Country:  "Initial",
		IPs:      []string{"10.0.0.1"},
	}
	lookup.AddServer(initialServer)

	// Merge new data (should NOT clear existing)
	jsonData := `{
		"merged_provider": {
			"version": 1,
			"timestamp": 1700000000,
			"servers": [
				{"country": "Merged", "ips": ["10.0.0.2"]}
			]
		}
	}`

	_, err := importer.MergeFromReader(strings.NewReader(jsonData))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Both IPs should be present
	if !lookup.ContainsIP("10.0.0.1") {
		t.Error("expected initial IP to still be present after merge")
	}
	if !lookup.ContainsIP("10.0.0.2") {
		t.Error("expected merged IP to be present after merge")
	}
}

func TestImporter_InvalidJSON(t *testing.T) {
	lookup := NewLookup()
	importer := NewImporter(lookup)

	invalidJSON := `{not valid json`

	_, err := importer.ImportFromReader(strings.NewReader(invalidJSON))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestImporter_EmptyData(t *testing.T) {
	lookup := NewLookup()
	importer := NewImporter(lookup)

	emptyJSON := `{}`

	result, err := importer.ImportFromReader(strings.NewReader(emptyJSON))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ProvidersImported != 0 {
		t.Errorf("expected 0 providers, got %d", result.ProvidersImported)
	}
	if result.IPsImported != 0 {
		t.Errorf("expected 0 IPs, got %d", result.IPsImported)
	}
}

func TestImporter_SkipsVersionField(t *testing.T) {
	lookup := NewLookup()
	importer := NewImporter(lookup)

	// Root "version" field should be skipped
	jsonData := `{
		"version": 1,
		"real_provider": {
			"version": 1,
			"timestamp": 1700000000,
			"servers": [
				{"country": "Test", "ips": ["10.0.0.1"]}
			]
		}
	}`

	result, err := importer.ImportFromReader(strings.NewReader(jsonData))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ProvidersImported != 1 {
		t.Errorf("expected 1 provider (skipping version), got %d", result.ProvidersImported)
	}
}

func TestImporter_IPv6Support(t *testing.T) {
	lookup := NewLookup()
	importer := NewImporter(lookup)

	jsonData := `{
		"ipv6_provider": {
			"version": 1,
			"timestamp": 1700000000,
			"servers": [
				{
					"country": "Test",
					"ips": ["2001:db8::1", "2001:db8::2", "192.0.2.1"]
				}
			]
		}
	}`

	result, err := importer.ImportFromReader(strings.NewReader(jsonData))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.IPsImported != 3 {
		t.Errorf("expected 3 IPs, got %d", result.IPsImported)
	}

	// Check IPv6 lookup
	v6Result := lookup.LookupIP("2001:db8::1")
	if !v6Result.IsVPN {
		t.Error("expected IPv6 address to be detected as VPN")
	}

	// Check stats
	stats := lookup.GetStats()
	if stats.IPv6Count != 2 {
		t.Errorf("expected 2 IPv6 addresses, got %d", stats.IPv6Count)
	}
	if stats.IPv4Count != 1 {
		t.Errorf("expected 1 IPv4 address, got %d", stats.IPv4Count)
	}
}

func TestImporter_ImportFromFile(t *testing.T) {
	lookup := NewLookup()
	importer := NewImporter(lookup)

	// Create a temporary file with test data
	tmpFile, err := os.CreateTemp("", "vpn_test_*.json")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	jsonData := `{
		"testprovider": {
			"version": 1,
			"timestamp": 1700000000,
			"servers": [
				{
					"country": "Test Country",
					"city": "Test City",
					"hostname": "test.example.com",
					"ips": ["192.0.2.100", "192.0.2.101"]
				}
			]
		}
	}`

	if _, err := tmpFile.WriteString(jsonData); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	tmpFile.Close()

	result, err := importer.ImportFromFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ProvidersImported != 1 {
		t.Errorf("expected 1 provider, got %d", result.ProvidersImported)
	}
	if result.IPsImported != 2 {
		t.Errorf("expected 2 IPs, got %d", result.IPsImported)
	}

	// Verify lookup works
	lookupResult := lookup.LookupIP("192.0.2.100")
	if !lookupResult.IsVPN {
		t.Error("expected IP to be detected as VPN")
	}
	if lookupResult.ServerCountry != "Test Country" {
		t.Errorf("expected country 'Test Country', got %s", lookupResult.ServerCountry)
	}
}

func TestImporter_ImportFromFile_NotFound(t *testing.T) {
	lookup := NewLookup()
	importer := NewImporter(lookup)

	_, err := importer.ImportFromFile("/nonexistent/path/servers.json")
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestImporter_ImportFromBytes_InvalidJSON(t *testing.T) {
	lookup := NewLookup()
	importer := NewImporter(lookup)

	_, err := importer.ImportFromBytes([]byte("not valid json"))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestImporter_MergeFromReader_InvalidJSON(t *testing.T) {
	lookup := NewLookup()
	importer := NewImporter(lookup)

	_, err := importer.MergeFromReader(strings.NewReader("not valid json"))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestImporter_ServerWithAllFields(t *testing.T) {
	lookup := NewLookup()
	importer := NewImporter(lookup)

	jsonData := `{
		"fullprovider": {
			"version": 5,
			"timestamp": 1700000000,
			"servers": [
				{
					"vpn": "wireguard",
					"country": "Germany",
					"region": "Europe",
					"city": "Berlin",
					"isp": "Example ISP",
					"owned": true,
					"number": 42,
					"server_name": "Berlin-42",
					"hostname": "de-ber-042.example.com",
					"tcp": true,
					"udp": true,
					"wgpubkey": "test-pubkey",
					"ips": ["10.0.0.42"],
					"categories": ["P2P", "Streaming"]
				}
			]
		}
	}`

	result, err := importer.ImportFromReader(strings.NewReader(jsonData))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ProvidersImported != 1 {
		t.Errorf("expected 1 provider, got %d", result.ProvidersImported)
	}

	// Verify server details through lookup
	lookupResult := lookup.LookupIP("10.0.0.42")
	if !lookupResult.IsVPN {
		t.Error("expected IP to be VPN")
	}
	if lookupResult.ServerCountry != "Germany" {
		t.Errorf("expected country Germany, got %s", lookupResult.ServerCountry)
	}
	if lookupResult.ServerCity != "Berlin" {
		t.Errorf("expected city Berlin, got %s", lookupResult.ServerCity)
	}
}

func TestImporter_ProviderDisplayName(t *testing.T) {
	lookup := NewLookup()
	importer := NewImporter(lookup)

	jsonData := `{
		"nordvpn": {
			"version": 1,
			"timestamp": 1700000000,
			"servers": [
				{"country": "Test", "ips": ["10.0.0.1"]}
			]
		}
	}`

	_, err := importer.ImportFromReader(strings.NewReader(jsonData))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify provider display name
	provider := lookup.GetProvider("nordvpn")
	if provider == nil {
		t.Fatal("expected provider to exist")
	}
	if provider.DisplayName != "NordVPN" {
		t.Errorf("expected display name 'NordVPN', got %s", provider.DisplayName)
	}
}

func TestImporter_SkipsInvalidProviderEntries(t *testing.T) {
	lookup := NewLookup()
	importer := NewImporter(lookup)

	// Mix of valid and invalid provider entries
	jsonData := `{
		"version": 1,
		"valid_provider": {
			"version": 1,
			"timestamp": 1700000000,
			"servers": [
				{"country": "Test", "ips": ["10.0.0.1"]}
			]
		},
		"invalid_entry": "this is not a provider object"
	}`

	result, err := importer.ImportFromReader(strings.NewReader(jsonData))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should only import the valid provider
	if result.ProvidersImported != 1 {
		t.Errorf("expected 1 provider (skipping invalid), got %d", result.ProvidersImported)
	}
}
