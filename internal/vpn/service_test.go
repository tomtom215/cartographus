// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package vpn

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
)

func TestNewService(t *testing.T) {
	t.Run("with nil db", func(t *testing.T) {
		service, err := NewService(nil, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if service == nil {
			t.Fatal("expected non-nil service")
		}
		if service.lookup == nil {
			t.Error("expected non-nil lookup")
		}
		if service.importer == nil {
			t.Error("expected non-nil importer")
		}
		if service.config == nil {
			t.Error("expected non-nil config")
		}
	})

	t.Run("with default config", func(t *testing.T) {
		service, err := NewService(nil, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !service.config.Enabled {
			t.Error("expected service to be enabled by default")
		}
		if service.config.CacheSize != 10000 {
			t.Errorf("expected cache size 10000, got %d", service.config.CacheSize)
		}
	})

	t.Run("with custom config", func(t *testing.T) {
		customConfig := &Config{
			Enabled:   false,
			CacheSize: 5000,
		}
		service, err := NewService(nil, customConfig)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if service.config.Enabled {
			t.Error("expected service to be disabled")
		}
		if service.config.CacheSize != 5000 {
			t.Errorf("expected cache size 5000, got %d", service.config.CacheSize)
		}
	})
}

func TestService_IsVPN(t *testing.T) {
	t.Run("returns false when disabled", func(t *testing.T) {
		service, _ := NewService(nil, &Config{Enabled: false})

		// Add a VPN IP directly to lookup
		if err := service.lookup.AddServer(&Server{
			Provider: "test",
			IPs:      []string{"192.0.2.1"},
		}); err != nil {
			t.Fatalf("AddServer() error = %v", err)
		}

		// Should return false because service is disabled
		if service.IsVPN("192.0.2.1") {
			t.Error("expected IsVPN to return false when service is disabled")
		}
	})

	t.Run("returns true for VPN IP when enabled", func(t *testing.T) {
		service, _ := NewService(nil, &Config{Enabled: true})

		if err := service.lookup.AddServer(&Server{
			Provider: "nordvpn",
			IPs:      []string{"198.51.100.1"},
		}); err != nil {
			t.Fatalf("AddServer() error = %v", err)
		}

		if !service.IsVPN("198.51.100.1") {
			t.Error("expected IsVPN to return true for known VPN IP")
		}
	})

	t.Run("returns false for non-VPN IP", func(t *testing.T) {
		service, _ := NewService(nil, &Config{Enabled: true})

		if service.IsVPN("8.8.8.8") {
			t.Error("expected IsVPN to return false for non-VPN IP")
		}
	})

	t.Run("returns false for invalid IP", func(t *testing.T) {
		service, _ := NewService(nil, &Config{Enabled: true})

		if service.IsVPN("not-an-ip") {
			t.Error("expected IsVPN to return false for invalid IP")
		}
	})
}

func TestService_LookupIP(t *testing.T) {
	t.Run("returns no VPN result when disabled", func(t *testing.T) {
		service, _ := NewService(nil, &Config{Enabled: false})

		if err := service.lookup.AddServer(&Server{
			Provider: "test",
			Country:  "Test",
			IPs:      []string{"192.0.2.1"},
		}); err != nil {
			t.Fatalf("AddServer() error = %v", err)
		}

		result := service.LookupIP("192.0.2.1")
		if result.IsVPN {
			t.Error("expected IsVPN to be false when service is disabled")
		}
		if result.Confidence != 0 {
			t.Errorf("expected confidence 0 when disabled, got %d", result.Confidence)
		}
	})

	t.Run("returns VPN details when found", func(t *testing.T) {
		service, _ := NewService(nil, &Config{Enabled: true})

		service.lookup.AddServer(&Server{
			Provider: "expressvpn",
			Country:  "Germany",
			City:     "Frankfurt",
			Hostname: "de-fra-001.expressvpn.com",
			IPs:      []string{"203.0.113.50"},
		})

		result := service.LookupIP("203.0.113.50")
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
	})

	t.Run("returns not VPN for unknown IP", func(t *testing.T) {
		service, _ := NewService(nil, &Config{Enabled: true})

		result := service.LookupIP("8.8.8.8")
		if result.IsVPN {
			t.Error("expected IsVPN to be false for unknown IP")
		}
	})
}

func TestService_ImportFromReader(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("failed to open duckdb: %v", err)
	}
	defer db.Close()

	service, err := NewService(db, nil)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	ctx := context.Background()
	if err := service.Initialize(ctx); err != nil {
		t.Fatalf("failed to initialize: %v", err)
	}

	jsonData := `{
		"testprovider": {
			"version": 1,
			"timestamp": 1700000000,
			"servers": [
				{
					"country": "United States",
					"city": "New York",
					"hostname": "us-nyc-001.test.com",
					"ips": ["198.51.100.1", "198.51.100.2"]
				}
			]
		}
	}`

	result, err := service.ImportFromReader(ctx, strings.NewReader(jsonData))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ProvidersImported != 1 {
		t.Errorf("expected 1 provider imported, got %d", result.ProvidersImported)
	}
	if result.IPsImported != 2 {
		t.Errorf("expected 2 IPs imported, got %d", result.IPsImported)
	}

	// Verify lookup works
	if !service.IsVPN("198.51.100.1") {
		t.Error("expected imported IP to be detected as VPN")
	}
}

func TestService_ImportFromBytes(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("failed to open duckdb: %v", err)
	}
	defer db.Close()

	service, err := NewService(db, nil)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	ctx := context.Background()
	if err := service.Initialize(ctx); err != nil {
		t.Fatalf("failed to initialize: %v", err)
	}

	jsonData := []byte(`{
		"mullvad": {
			"version": 2,
			"timestamp": 1700000000,
			"servers": [
				{
					"country": "Sweden",
					"city": "Stockholm",
					"hostname": "se-sto-001.mullvad.net",
					"ips": ["192.0.2.1"]
				}
			]
		}
	}`)

	result, err := service.ImportFromBytes(ctx, jsonData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ProvidersImported != 1 {
		t.Errorf("expected 1 provider, got %d", result.ProvidersImported)
	}

	if !service.IsVPN("192.0.2.1") {
		t.Error("expected imported IP to be detected as VPN")
	}
}

func TestService_ImportFromFile(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("failed to open duckdb: %v", err)
	}
	defer db.Close()

	service, err := NewService(db, nil)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	ctx := context.Background()
	if err := service.Initialize(ctx); err != nil {
		t.Fatalf("failed to initialize: %v", err)
	}

	// Create a temporary file with test data
	tmpFile, err := os.CreateTemp("", "vpn_service_test_*.json")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	jsonData := `{
		"surfshark": {
			"version": 1,
			"timestamp": 1700000000,
			"servers": [
				{
					"country": "Netherlands",
					"city": "Amsterdam",
					"hostname": "nl-ams-001.surfshark.com",
					"ips": ["203.0.113.10"]
				}
			]
		}
	}`

	if _, err := tmpFile.WriteString(jsonData); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	tmpFile.Close()

	result, err := service.ImportFromFile(ctx, tmpFile.Name())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ProvidersImported != 1 {
		t.Errorf("expected 1 provider, got %d", result.ProvidersImported)
	}

	if !service.IsVPN("203.0.113.10") {
		t.Error("expected imported IP to be detected as VPN")
	}
}

func TestService_ImportFromFile_NotFound(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("failed to open duckdb: %v", err)
	}
	defer db.Close()

	service, err := NewService(db, nil)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	ctx := context.Background()
	if err := service.Initialize(ctx); err != nil {
		t.Fatalf("failed to initialize: %v", err)
	}

	_, err = service.ImportFromFile(ctx, "/nonexistent/path/servers.json")
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestService_GetStats(t *testing.T) {
	service, _ := NewService(nil, nil)

	// Initially empty
	stats := service.GetStats()
	if stats.TotalIPs != 0 {
		t.Errorf("expected 0 IPs initially, got %d", stats.TotalIPs)
	}

	// Add some data
	service.lookup.AddServer(&Server{
		Provider: "test",
		IPs:      []string{"192.0.2.1", "192.0.2.2"},
	})
	service.lookup.AddServer(&Server{
		Provider: "test",
		IPs:      []string{"2001:db8::1"},
	})

	stats = service.GetStats()
	if stats.TotalIPs != 3 {
		t.Errorf("expected 3 IPs, got %d", stats.TotalIPs)
	}
	if stats.IPv4Count != 2 {
		t.Errorf("expected 2 IPv4, got %d", stats.IPv4Count)
	}
	if stats.IPv6Count != 1 {
		t.Errorf("expected 1 IPv6, got %d", stats.IPv6Count)
	}
}

func TestService_GetProvider(t *testing.T) {
	service, _ := NewService(nil, nil)

	// Add provider
	service.lookup.AddProvider(&Provider{
		Name:        "protonvpn",
		DisplayName: "ProtonVPN",
		Website:     "https://protonvpn.com",
		ServerCount: 100,
		IPCount:     500,
	})

	provider := service.GetProvider("protonvpn")
	if provider == nil {
		t.Fatal("expected provider to exist")
	}
	if provider.DisplayName != "ProtonVPN" {
		t.Errorf("expected ProtonVPN, got %s", provider.DisplayName)
	}

	// Non-existent provider
	provider = service.GetProvider("nonexistent")
	if provider != nil {
		t.Error("expected nil for non-existent provider")
	}
}

func TestService_ListProviders(t *testing.T) {
	service, _ := NewService(nil, nil)

	// Initially empty
	providers := service.ListProviders()
	if len(providers) != 0 {
		t.Errorf("expected 0 providers initially, got %d", len(providers))
	}

	// Add providers
	service.lookup.AddProvider(&Provider{Name: "nordvpn", DisplayName: "NordVPN"})
	service.lookup.AddProvider(&Provider{Name: "expressvpn", DisplayName: "ExpressVPN"})

	providers = service.ListProviders()
	if len(providers) != 2 {
		t.Errorf("expected 2 providers, got %d", len(providers))
	}
}

func TestService_SetEnabled(t *testing.T) {
	service, _ := NewService(nil, &Config{Enabled: true})

	if !service.Enabled() {
		t.Error("expected service to be enabled initially")
	}

	service.SetEnabled(false)
	if service.Enabled() {
		t.Error("expected service to be disabled")
	}

	service.SetEnabled(true)
	if !service.Enabled() {
		t.Error("expected service to be re-enabled")
	}
}

func TestService_Reload(t *testing.T) {
	// Create in-memory DuckDB
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("failed to open duckdb: %v", err)
	}
	defer db.Close()

	service, err := NewService(db, nil)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	ctx := context.Background()

	// Initialize schema
	if err := service.Initialize(ctx); err != nil {
		t.Fatalf("failed to initialize: %v", err)
	}

	// Add data directly to lookup
	service.lookup.AddServer(&Server{
		Provider: "test",
		IPs:      []string{"192.0.2.1"},
	})

	if service.lookup.Count() != 1 {
		t.Errorf("expected 1 IP before reload, got %d", service.lookup.Count())
	}

	// Reload should clear lookup and reload from (empty) database
	if err := service.Reload(ctx); err != nil {
		t.Fatalf("reload failed: %v", err)
	}

	// After reload from empty database, count should be 0
	if service.lookup.Count() != 0 {
		t.Errorf("expected 0 IPs after reload, got %d", service.lookup.Count())
	}
}

func TestService_EnrichWithVPNInfo(t *testing.T) {
	service, _ := NewService(nil, &Config{Enabled: true})

	// Add VPN data
	service.lookup.AddServer(&Server{
		Provider: "mullvad",
		Country:  "Netherlands",
		City:     "Amsterdam",
		Hostname: "nl-ams-001.mullvad.net",
		IPs:      []string{"198.51.100.50"},
	})

	t.Run("enriches VPN IP", func(t *testing.T) {
		result := service.EnrichWithVPNInfo(
			"198.51.100.50",
			52.3676, 4.9041,
			"Amsterdam", "North Holland", "NL",
		)

		if result.IPAddress != "198.51.100.50" {
			t.Errorf("expected IP 198.51.100.50, got %s", result.IPAddress)
		}
		if result.Latitude != 52.3676 {
			t.Errorf("expected latitude 52.3676, got %f", result.Latitude)
		}
		if !result.IsVPN {
			t.Error("expected IsVPN to be true")
		}
		if result.VPNProvider != "mullvad" {
			t.Errorf("expected provider mullvad, got %s", result.VPNProvider)
		}
		if result.VPNProviderDisplay != "Mullvad" {
			t.Errorf("expected display name Mullvad, got %s", result.VPNProviderDisplay)
		}
		if result.VPNServerCountry != "Netherlands" {
			t.Errorf("expected Netherlands, got %s", result.VPNServerCountry)
		}
		if result.VPNServerCity != "Amsterdam" {
			t.Errorf("expected Amsterdam, got %s", result.VPNServerCity)
		}
		if result.VPNConfidence != 100 {
			t.Errorf("expected confidence 100, got %d", result.VPNConfidence)
		}
		if result.LastUpdated.IsZero() {
			t.Error("expected LastUpdated to be set")
		}
	})

	t.Run("enriches non-VPN IP", func(t *testing.T) {
		result := service.EnrichWithVPNInfo(
			"8.8.8.8",
			37.7749, -122.4194,
			"San Francisco", "California", "US",
		)

		if result.IPAddress != "8.8.8.8" {
			t.Errorf("expected IP 8.8.8.8, got %s", result.IPAddress)
		}
		if result.IsVPN {
			t.Error("expected IsVPN to be false")
		}
		if result.VPNProvider != "" {
			t.Errorf("expected empty provider, got %s", result.VPNProvider)
		}
	})
}

func TestService_Initialize(t *testing.T) {
	t.Run("initializes with database", func(t *testing.T) {
		db, err := sql.Open("duckdb", ":memory:")
		if err != nil {
			t.Fatalf("failed to open duckdb: %v", err)
		}
		defer db.Close()

		service, err := NewService(db, nil)
		if err != nil {
			t.Fatalf("failed to create service: %v", err)
		}

		ctx := context.Background()
		if err := service.Initialize(ctx); err != nil {
			t.Fatalf("failed to initialize: %v", err)
		}

		// Verify tables exist by inserting data through store
		_, err = db.ExecContext(ctx, `
			INSERT INTO vpn_providers (name, display_name)
			VALUES ('test', 'Test Provider')
		`)
		if err != nil {
			t.Errorf("expected vpn_providers table to exist: %v", err)
		}
	})

	t.Run("loads existing data from database", func(t *testing.T) {
		db, err := sql.Open("duckdb", ":memory:")
		if err != nil {
			t.Fatalf("failed to open duckdb: %v", err)
		}
		defer db.Close()

		ctx := context.Background()

		// First, create a service and add data
		service1, _ := NewService(db, nil)
		if err := service1.Initialize(ctx); err != nil {
			t.Fatalf("failed to initialize first service: %v", err)
		}

		// Add data directly to database
		_, err = db.ExecContext(ctx, `
			INSERT INTO vpn_ips (ip_address, provider, country, city, hostname)
			VALUES ('192.0.2.100', 'testprovider', 'Test Country', 'Test City', 'test.example.com')
		`)
		if err != nil {
			t.Fatalf("failed to insert test data: %v", err)
		}

		// Create a new service and initialize (should load data)
		service2, _ := NewService(db, nil)
		if err := service2.Initialize(ctx); err != nil {
			t.Fatalf("failed to initialize second service: %v", err)
		}

		// Verify data was loaded
		if !service2.IsVPN("192.0.2.100") {
			t.Error("expected IP to be loaded from database")
		}

		result := service2.LookupIP("192.0.2.100")
		if result.ServerCountry != "Test Country" {
			t.Errorf("expected country 'Test Country', got %s", result.ServerCountry)
		}
	})
}

func TestService_ConcurrentAccess(t *testing.T) {
	service, _ := NewService(nil, nil)

	// Add some data
	service.lookup.AddServer(&Server{
		Provider: "test",
		IPs:      []string{"192.0.2.1"},
	})

	// Run concurrent operations
	done := make(chan bool)

	// Concurrent IsVPN calls
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				service.IsVPN("192.0.2.1")
				service.IsVPN("8.8.8.8")
			}
			done <- true
		}()
	}

	// Concurrent LookupIP calls
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				service.LookupIP("192.0.2.1")
				service.LookupIP("8.8.8.8")
			}
			done <- true
		}()
	}

	// Concurrent GetStats calls
	for i := 0; i < 5; i++ {
		go func() {
			for j := 0; j < 50; j++ {
				service.GetStats()
			}
			done <- true
		}()
	}

	// Concurrent SetEnabled/Enabled calls
	for i := 0; i < 5; i++ {
		go func() {
			for j := 0; j < 50; j++ {
				service.SetEnabled(true)
				service.Enabled()
				service.SetEnabled(false)
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 30; i++ {
		<-done
	}
}

func TestService_PersistLookupData(t *testing.T) {
	t.Run("persists data successfully", func(t *testing.T) {
		db, err := sql.Open("duckdb", ":memory:")
		if err != nil {
			t.Fatalf("failed to open duckdb: %v", err)
		}
		defer db.Close()

		service, err := NewService(db, nil)
		if err != nil {
			t.Fatalf("failed to create service: %v", err)
		}

		ctx := context.Background()
		if err := service.Initialize(ctx); err != nil {
			t.Fatalf("failed to initialize: %v", err)
		}

		jsonData := `{
			"testprovider": {
				"version": 1,
				"timestamp": 1700000000,
				"servers": [{"country": "Test", "ips": ["192.0.2.1"]}]
			}
		}`

		_, err = service.ImportFromBytes(ctx, []byte(jsonData))
		if err != nil {
			t.Errorf("import should succeed: %v", err)
		}

		// Verify data was persisted
		if !service.IsVPN("192.0.2.1") {
			t.Error("expected IP to be detected as VPN after import")
		}
	})
}

func TestEnrichedGeolocation_Fields(t *testing.T) {
	now := time.Now()
	eg := EnrichedGeolocation{
		IPAddress:          "192.0.2.1",
		Latitude:           51.5074,
		Longitude:          -0.1278,
		City:               "London",
		Region:             "England",
		Country:            "GB",
		LastUpdated:        now,
		IsVPN:              true,
		VPNProvider:        "nordvpn",
		VPNProviderDisplay: "NordVPN",
		VPNServerCountry:   "United Kingdom",
		VPNServerCity:      "London",
		VPNConfidence:      100,
	}

	if eg.IPAddress != "192.0.2.1" {
		t.Errorf("IPAddress = %s, want 192.0.2.1", eg.IPAddress)
	}
	if eg.Latitude != 51.5074 {
		t.Errorf("Latitude = %f, want 51.5074", eg.Latitude)
	}
	if eg.Longitude != -0.1278 {
		t.Errorf("Longitude = %f, want -0.1278", eg.Longitude)
	}
	if eg.City != "London" {
		t.Errorf("City = %s, want London", eg.City)
	}
	if eg.Region != "England" {
		t.Errorf("Region = %s, want England", eg.Region)
	}
	if eg.Country != "GB" {
		t.Errorf("Country = %s, want GB", eg.Country)
	}
	if eg.LastUpdated != now {
		t.Errorf("LastUpdated = %v, want %v", eg.LastUpdated, now)
	}
	if !eg.IsVPN {
		t.Error("expected IsVPN to be true")
	}
	if eg.VPNProvider != "nordvpn" {
		t.Errorf("VPNProvider = %s, want nordvpn", eg.VPNProvider)
	}
	if eg.VPNProviderDisplay != "NordVPN" {
		t.Errorf("VPNProviderDisplay = %s, want NordVPN", eg.VPNProviderDisplay)
	}
	if eg.VPNServerCountry != "United Kingdom" {
		t.Errorf("VPNServerCountry = %s, want United Kingdom", eg.VPNServerCountry)
	}
	if eg.VPNServerCity != "London" {
		t.Errorf("VPNServerCity = %s, want London", eg.VPNServerCity)
	}
	if eg.VPNConfidence != 100 {
		t.Errorf("VPNConfidence = %d, want 100", eg.VPNConfidence)
	}
}
