// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package vpn

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
)

func setupTestDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("failed to open duckdb: %v", err)
	}
	return db, func() {
		db.Close()
	}
}

func TestNewDuckDBStore(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewDuckDBStore(db)
	if store == nil {
		t.Fatal("expected non-nil store")
	}
	if store.db != db {
		t.Error("expected store to use provided db")
	}
}

func TestDuckDBStore_InitSchema(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewDuckDBStore(db)
	ctx := context.Background()

	if err := store.InitSchema(ctx); err != nil {
		t.Fatalf("failed to init schema: %v", err)
	}

	// Verify tables exist
	tables := []string{"vpn_providers", "vpn_ips", "vpn_metadata"}
	for _, table := range tables {
		var count int
		err := db.QueryRowContext(ctx, `
			SELECT COUNT(*) FROM information_schema.tables
			WHERE table_name = ?
		`, table).Scan(&count)
		if err != nil {
			t.Errorf("failed to check table %s: %v", table, err)
		}
		if count != 1 {
			t.Errorf("expected table %s to exist", table)
		}
	}

	// Verify index exists
	var indexCount int
	err := db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM duckdb_indexes()
		WHERE index_name = 'idx_vpn_ips_provider'
	`).Scan(&indexCount)
	if err != nil {
		t.Errorf("failed to check index: %v", err)
	}
	if indexCount != 1 {
		t.Error("expected idx_vpn_ips_provider index to exist")
	}
}

func TestDuckDBStore_InitSchema_Idempotent(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewDuckDBStore(db)
	ctx := context.Background()

	// Initialize twice - should not error
	if err := store.InitSchema(ctx); err != nil {
		t.Fatalf("first init failed: %v", err)
	}
	if err := store.InitSchema(ctx); err != nil {
		t.Fatalf("second init should be idempotent: %v", err)
	}
}

func TestDuckDBStore_SaveProvider(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewDuckDBStore(db)
	ctx := context.Background()

	if err := store.InitSchema(ctx); err != nil {
		t.Fatalf("failed to init schema: %v", err)
	}

	now := time.Now()
	provider := &Provider{
		Name:        "nordvpn",
		DisplayName: "NordVPN",
		Website:     "https://nordvpn.com",
		ServerCount: 5000,
		IPCount:     10000,
		Version:     5,
		Timestamp:   now.Unix(),
		LastUpdated: now,
	}

	if err := store.SaveProvider(ctx, provider); err != nil {
		t.Fatalf("failed to save provider: %v", err)
	}

	// Verify saved
	retrieved, err := store.GetProvider(ctx, "nordvpn")
	if err != nil {
		t.Fatalf("failed to get provider: %v", err)
	}
	if retrieved == nil {
		t.Fatal("expected provider to be retrieved")
	}
	if retrieved.DisplayName != "NordVPN" {
		t.Errorf("DisplayName = %s, want NordVPN", retrieved.DisplayName)
	}
	if retrieved.ServerCount != 5000 {
		t.Errorf("ServerCount = %d, want 5000", retrieved.ServerCount)
	}
}

func TestDuckDBStore_SaveProvider_Upsert(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewDuckDBStore(db)
	ctx := context.Background()

	if err := store.InitSchema(ctx); err != nil {
		t.Fatalf("failed to init schema: %v", err)
	}

	// Save initial
	provider1 := &Provider{
		Name:        "test",
		DisplayName: "Test V1",
		ServerCount: 100,
	}
	if err := store.SaveProvider(ctx, provider1); err != nil {
		t.Fatalf("failed to save initial provider: %v", err)
	}

	// Update with same name
	provider2 := &Provider{
		Name:        "test",
		DisplayName: "Test V2",
		ServerCount: 200,
	}
	if err := store.SaveProvider(ctx, provider2); err != nil {
		t.Fatalf("failed to update provider: %v", err)
	}

	// Verify updated
	retrieved, _ := store.GetProvider(ctx, "test")
	if retrieved.DisplayName != "Test V2" {
		t.Errorf("DisplayName = %s, want Test V2", retrieved.DisplayName)
	}
	if retrieved.ServerCount != 200 {
		t.Errorf("ServerCount = %d, want 200", retrieved.ServerCount)
	}
}

func TestDuckDBStore_GetProvider_NotFound(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewDuckDBStore(db)
	ctx := context.Background()

	if err := store.InitSchema(ctx); err != nil {
		t.Fatalf("failed to init schema: %v", err)
	}

	provider, err := store.GetProvider(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if provider != nil {
		t.Error("expected nil for non-existent provider")
	}
}

func TestDuckDBStore_ListProviders(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewDuckDBStore(db)
	ctx := context.Background()

	if err := store.InitSchema(ctx); err != nil {
		t.Fatalf("failed to init schema: %v", err)
	}

	// Initially empty
	providers, err := store.ListProviders(ctx)
	if err != nil {
		t.Fatalf("failed to list providers: %v", err)
	}
	if len(providers) != 0 {
		t.Errorf("expected 0 providers, got %d", len(providers))
	}

	// Add providers
	store.SaveProvider(ctx, &Provider{Name: "a", DisplayName: "A", IPCount: 100})
	store.SaveProvider(ctx, &Provider{Name: "b", DisplayName: "B", IPCount: 300})
	store.SaveProvider(ctx, &Provider{Name: "c", DisplayName: "C", IPCount: 200})

	providers, err = store.ListProviders(ctx)
	if err != nil {
		t.Fatalf("failed to list providers: %v", err)
	}
	if len(providers) != 3 {
		t.Errorf("expected 3 providers, got %d", len(providers))
	}

	// Should be sorted by ip_count DESC
	if providers[0].Name != "b" {
		t.Errorf("expected first provider to be 'b' (highest ip_count), got %s", providers[0].Name)
	}
}

func TestDuckDBStore_SaveIP(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewDuckDBStore(db)
	ctx := context.Background()

	if err := store.InitSchema(ctx); err != nil {
		t.Fatalf("failed to init schema: %v", err)
	}

	err := store.SaveIP(ctx, "192.0.2.1", "nordvpn", "Sweden", "Stockholm", "se-sto-001.nordvpn.com")
	if err != nil {
		t.Fatalf("failed to save IP: %v", err)
	}

	// Verify
	isVPN, err := store.IsVPNIP(ctx, "192.0.2.1")
	if err != nil {
		t.Fatalf("failed to check VPN IP: %v", err)
	}
	if !isVPN {
		t.Error("expected IP to be VPN")
	}
}

func TestDuckDBStore_SaveIP_Upsert(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewDuckDBStore(db)
	ctx := context.Background()

	if err := store.InitSchema(ctx); err != nil {
		t.Fatalf("failed to init schema: %v", err)
	}

	// Save initial
	store.SaveIP(ctx, "192.0.2.1", "provider1", "Country1", "City1", "host1")

	// Update same IP with different data
	store.SaveIP(ctx, "192.0.2.1", "provider2", "Country2", "City2", "host2")

	// Verify updated
	result, err := store.GetVPNInfo(ctx, "192.0.2.1")
	if err != nil {
		t.Fatalf("failed to get VPN info: %v", err)
	}
	if result.Provider != "provider2" {
		t.Errorf("Provider = %s, want provider2", result.Provider)
	}
	if result.ServerCountry != "Country2" {
		t.Errorf("ServerCountry = %s, want Country2", result.ServerCountry)
	}
}

func TestDuckDBStore_SaveServer(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewDuckDBStore(db)
	ctx := context.Background()

	if err := store.InitSchema(ctx); err != nil {
		t.Fatalf("failed to init schema: %v", err)
	}

	server := &Server{
		Provider: "expressvpn",
		Country:  "Germany",
		City:     "Frankfurt",
		Hostname: "de-fra-001.expressvpn.com",
		IPs:      []string{"198.51.100.1", "198.51.100.2"},
	}

	if err := store.SaveServer(ctx, server); err != nil {
		t.Fatalf("failed to save server: %v", err)
	}

	// Both IPs should be VPN
	isVPN1, _ := store.IsVPNIP(ctx, "198.51.100.1")
	isVPN2, _ := store.IsVPNIP(ctx, "198.51.100.2")

	if !isVPN1 {
		t.Error("expected first IP to be VPN")
	}
	if !isVPN2 {
		t.Error("expected second IP to be VPN")
	}
}

func TestDuckDBStore_GetServers(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewDuckDBStore(db)
	ctx := context.Background()

	if err := store.InitSchema(ctx); err != nil {
		t.Fatalf("failed to init schema: %v", err)
	}

	// Add servers
	store.SaveServer(ctx, &Server{
		Provider: "mullvad",
		Country:  "Sweden",
		City:     "Stockholm",
		Hostname: "se-sto-001.mullvad.net",
		IPs:      []string{"192.0.2.1", "192.0.2.2"},
	})
	store.SaveServer(ctx, &Server{
		Provider: "mullvad",
		Country:  "Germany",
		City:     "Berlin",
		Hostname: "de-ber-001.mullvad.net",
		IPs:      []string{"192.0.2.3"},
	})
	store.SaveServer(ctx, &Server{
		Provider: "nordvpn", // Different provider
		Country:  "Norway",
		City:     "Oslo",
		Hostname: "no-osl-001.nordvpn.com",
		IPs:      []string{"192.0.2.4"},
	})

	servers, err := store.GetServers(ctx, "mullvad")
	if err != nil {
		t.Fatalf("failed to get servers: %v", err)
	}

	if len(servers) != 2 {
		t.Errorf("expected 2 mullvad servers, got %d", len(servers))
	}

	// Verify first server has grouped IPs
	found := false
	for _, s := range servers {
		if s.Hostname == "se-sto-001.mullvad.net" {
			found = true
			if len(s.IPs) != 2 {
				t.Errorf("expected 2 IPs for Stockholm server, got %d", len(s.IPs))
			}
		}
	}
	if !found {
		t.Error("expected to find Stockholm server")
	}
}

func TestDuckDBStore_IsVPNIP(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewDuckDBStore(db)
	ctx := context.Background()

	if err := store.InitSchema(ctx); err != nil {
		t.Fatalf("failed to init schema: %v", err)
	}

	store.SaveIP(ctx, "192.0.2.1", "test", "Test", "Test", "test.example.com")

	tests := []struct {
		ip       string
		expected bool
	}{
		{"192.0.2.1", true},
		{"192.0.2.2", false},
		{"8.8.8.8", false},
	}

	for _, tt := range tests {
		isVPN, err := store.IsVPNIP(ctx, tt.ip)
		if err != nil {
			t.Errorf("IsVPNIP(%s) error: %v", tt.ip, err)
		}
		if isVPN != tt.expected {
			t.Errorf("IsVPNIP(%s) = %v, want %v", tt.ip, isVPN, tt.expected)
		}
	}
}

func TestDuckDBStore_GetVPNInfo(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewDuckDBStore(db)
	ctx := context.Background()

	if err := store.InitSchema(ctx); err != nil {
		t.Fatalf("failed to init schema: %v", err)
	}

	store.SaveIP(ctx, "192.0.2.1", "protonvpn", "Switzerland", "Zurich", "ch-zrh-001.protonvpn.com")

	t.Run("found IP", func(t *testing.T) {
		result, err := store.GetVPNInfo(ctx, "192.0.2.1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !result.IsVPN {
			t.Error("expected IsVPN to be true")
		}
		if result.Provider != "protonvpn" {
			t.Errorf("Provider = %s, want protonvpn", result.Provider)
		}
		if result.ProviderDisplayName != "ProtonVPN" {
			t.Errorf("ProviderDisplayName = %s, want ProtonVPN", result.ProviderDisplayName)
		}
		if result.ServerCountry != "Switzerland" {
			t.Errorf("ServerCountry = %s, want Switzerland", result.ServerCountry)
		}
		if result.ServerCity != "Zurich" {
			t.Errorf("ServerCity = %s, want Zurich", result.ServerCity)
		}
		if result.ServerHostname != "ch-zrh-001.protonvpn.com" {
			t.Errorf("ServerHostname = %s, want ch-zrh-001.protonvpn.com", result.ServerHostname)
		}
		if result.Confidence != 100 {
			t.Errorf("Confidence = %d, want 100", result.Confidence)
		}
	})

	t.Run("not found IP", func(t *testing.T) {
		result, err := store.GetVPNInfo(ctx, "8.8.8.8")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.IsVPN {
			t.Error("expected IsVPN to be false")
		}
		if result.Confidence != 100 {
			t.Errorf("Confidence = %d, want 100 for negative match", result.Confidence)
		}
	})
}

func TestDuckDBStore_GetStats(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewDuckDBStore(db)
	ctx := context.Background()

	if err := store.InitSchema(ctx); err != nil {
		t.Fatalf("failed to init schema: %v", err)
	}

	// Empty stats
	stats, err := store.GetStats(ctx)
	if err != nil {
		t.Fatalf("failed to get stats: %v", err)
	}
	if stats.TotalIPs != 0 {
		t.Errorf("expected 0 IPs initially, got %d", stats.TotalIPs)
	}

	// Add data
	store.SaveIP(ctx, "192.0.2.1", "provider1", "Country", "City", "host1.example.com")
	store.SaveIP(ctx, "192.0.2.2", "provider1", "Country", "City", "host1.example.com")
	store.SaveIP(ctx, "2001:db8::1", "provider2", "Country", "City", "host2.example.com")
	store.SaveProvider(ctx, &Provider{Name: "provider1", DisplayName: "Provider 1"})
	store.SaveProvider(ctx, &Provider{Name: "provider2", DisplayName: "Provider 2"})

	stats, err = store.GetStats(ctx)
	if err != nil {
		t.Fatalf("failed to get stats: %v", err)
	}

	if stats.TotalIPs != 3 {
		t.Errorf("TotalIPs = %d, want 3", stats.TotalIPs)
	}
	if stats.IPv4Count != 2 {
		t.Errorf("IPv4Count = %d, want 2", stats.IPv4Count)
	}
	if stats.IPv6Count != 1 {
		t.Errorf("IPv6Count = %d, want 1", stats.IPv6Count)
	}
	if stats.TotalProviders != 2 {
		t.Errorf("TotalProviders = %d, want 2", stats.TotalProviders)
	}
	if stats.TotalServers != 2 {
		t.Errorf("TotalServers = %d, want 2 (unique hostnames)", stats.TotalServers)
	}
}

func TestDuckDBStore_Clear(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewDuckDBStore(db)
	ctx := context.Background()

	if err := store.InitSchema(ctx); err != nil {
		t.Fatalf("failed to init schema: %v", err)
	}

	// Add data
	store.SaveIP(ctx, "192.0.2.1", "test", "Country", "City", "host.example.com")
	store.SaveProvider(ctx, &Provider{Name: "test", DisplayName: "Test"})

	// Verify data exists
	isVPN, _ := store.IsVPNIP(ctx, "192.0.2.1")
	if !isVPN {
		t.Error("expected IP to exist before clear")
	}

	// Clear
	if err := store.Clear(ctx); err != nil {
		t.Fatalf("failed to clear: %v", err)
	}

	// Verify cleared
	isVPN, _ = store.IsVPNIP(ctx, "192.0.2.1")
	if isVPN {
		t.Error("expected IP to be cleared")
	}

	providers, _ := store.ListProviders(ctx)
	if len(providers) != 0 {
		t.Errorf("expected 0 providers after clear, got %d", len(providers))
	}
}

func TestDuckDBStore_LoadIntoLookup(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewDuckDBStore(db)
	ctx := context.Background()

	if err := store.InitSchema(ctx); err != nil {
		t.Fatalf("failed to init schema: %v", err)
	}

	// Add data to database
	store.SaveIP(ctx, "192.0.2.1", "nordvpn", "Sweden", "Stockholm", "se-sto-001.nordvpn.com")
	store.SaveIP(ctx, "192.0.2.2", "nordvpn", "Sweden", "Stockholm", "se-sto-001.nordvpn.com")
	store.SaveIP(ctx, "2001:db8::1", "mullvad", "Germany", "Berlin", "de-ber-001.mullvad.net")
	store.SaveProvider(ctx, &Provider{Name: "nordvpn", DisplayName: "NordVPN", IPCount: 2})
	store.SaveProvider(ctx, &Provider{Name: "mullvad", DisplayName: "Mullvad", IPCount: 1})

	// Create lookup and load data
	lookup := NewLookup()
	if err := store.LoadIntoLookup(ctx, lookup); err != nil {
		t.Fatalf("failed to load into lookup: %v", err)
	}

	// Verify IPs loaded
	if lookup.Count() != 3 {
		t.Errorf("expected 3 IPs in lookup, got %d", lookup.Count())
	}

	// Verify lookup works
	result := lookup.LookupIP("192.0.2.1")
	if !result.IsVPN {
		t.Error("expected IP to be VPN")
	}
	if result.Provider != "nordvpn" {
		t.Errorf("Provider = %s, want nordvpn", result.Provider)
	}
	if result.ServerCountry != "Sweden" {
		t.Errorf("ServerCountry = %s, want Sweden", result.ServerCountry)
	}

	// Verify providers loaded
	provider := lookup.GetProvider("nordvpn")
	if provider == nil {
		t.Error("expected nordvpn provider to be loaded")
	}

	provider = lookup.GetProvider("mullvad")
	if provider == nil {
		t.Error("expected mullvad provider to be loaded")
	}

	// Verify stats
	stats := lookup.GetStats()
	if stats.TotalProviders != 2 {
		t.Errorf("TotalProviders = %d, want 2", stats.TotalProviders)
	}
}

func TestDuckDBStore_SaveFromLookup(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewDuckDBStore(db)
	ctx := context.Background()

	if err := store.InitSchema(ctx); err != nil {
		t.Fatalf("failed to init schema: %v", err)
	}

	// Create lookup with data
	lookup := NewLookup()
	lookup.AddServer(&Server{
		Provider: "test",
		Country:  "Test Country",
		City:     "Test City",
		IPs:      []string{"192.0.2.1"},
	})
	lookup.AddProvider(&Provider{Name: "test", DisplayName: "Test Provider", IPCount: 1})

	// Save to database
	if err := store.SaveFromLookup(ctx, lookup); err != nil {
		t.Fatalf("failed to save from lookup: %v", err)
	}

	// Note: SaveFromLookup only saves providers, not IPs (by design)
	// because lookup's IP maps are private
	providers, _ := store.ListProviders(ctx)
	if len(providers) != 1 {
		t.Errorf("expected 1 provider saved, got %d", len(providers))
	}
}

func TestDuckDBStore_BulkSaveIPs(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewDuckDBStore(db)
	ctx := context.Background()

	if err := store.InitSchema(ctx); err != nil {
		t.Fatalf("failed to init schema: %v", err)
	}

	ips := []struct {
		IP       string
		Provider string
		Country  string
		City     string
		Hostname string
	}{
		{"192.0.2.1", "provider1", "Country1", "City1", "host1.example.com"},
		{"192.0.2.2", "provider1", "Country1", "City1", "host1.example.com"},
		{"192.0.2.3", "provider2", "Country2", "City2", "host2.example.com"},
	}

	if err := store.BulkSaveIPs(ctx, ips); err != nil {
		t.Fatalf("failed to bulk save IPs: %v", err)
	}

	// Verify all saved
	for _, ip := range ips {
		isVPN, err := store.IsVPNIP(ctx, ip.IP)
		if err != nil {
			t.Errorf("IsVPNIP(%s) error: %v", ip.IP, err)
		}
		if !isVPN {
			t.Errorf("expected %s to be VPN", ip.IP)
		}
	}
}

func TestDuckDBStore_BulkSaveIPs_Empty(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewDuckDBStore(db)
	ctx := context.Background()

	if err := store.InitSchema(ctx); err != nil {
		t.Fatalf("failed to init schema: %v", err)
	}

	// Empty slice should not error
	err := store.BulkSaveIPs(ctx, nil)
	if err != nil {
		t.Errorf("empty bulk save should not error: %v", err)
	}
}

func TestDuckDBStore_SaveUpdateStatus(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewDuckDBStore(db)
	ctx := context.Background()

	if err := store.InitSchema(ctx); err != nil {
		t.Fatalf("failed to init schema: %v", err)
	}

	now := time.Now()
	status := &UpdateStatus{
		LastUpdateAttempt:    now,
		LastSuccessfulUpdate: now.Add(-time.Hour),
		LastError:            "test error",
		SourceURL:            "https://example.com/servers.json",
		DataHash:             "abc123hash",
	}

	if err := store.SaveUpdateStatus(ctx, status); err != nil {
		t.Fatalf("failed to save update status: %v", err)
	}

	// Load and verify
	loaded, err := store.LoadUpdateStatus(ctx)
	if err != nil {
		t.Fatalf("failed to load update status: %v", err)
	}

	if loaded.LastError != "test error" {
		t.Errorf("LastError = %s, want 'test error'", loaded.LastError)
	}
	if loaded.SourceURL != "https://example.com/servers.json" {
		t.Errorf("SourceURL = %s, want https://example.com/servers.json", loaded.SourceURL)
	}
	if loaded.DataHash != "abc123hash" {
		t.Errorf("DataHash = %s, want abc123hash", loaded.DataHash)
	}
}

func TestDuckDBStore_LoadUpdateStatus_Empty(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewDuckDBStore(db)
	ctx := context.Background()

	if err := store.InitSchema(ctx); err != nil {
		t.Fatalf("failed to init schema: %v", err)
	}

	// Load from empty database
	status, err := store.LoadUpdateStatus(ctx)
	if err != nil {
		t.Fatalf("failed to load empty status: %v", err)
	}

	// Should return empty status, not nil
	if status == nil {
		t.Error("expected non-nil status even when empty")
	}
	if status.ProviderVersions == nil {
		t.Error("expected ProviderVersions to be initialized")
	}
}

func TestParseIPList(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "empty LIST format",
			input:    "[]",
			expected: nil,
		},
		{
			name:     "single IP LIST format",
			input:    "['192.0.2.1']",
			expected: []string{"192.0.2.1"},
		},
		{
			name:     "multiple IPs LIST format",
			input:    "['192.0.2.1', '192.0.2.2', '192.0.2.3']",
			expected: []string{"192.0.2.1", "192.0.2.2", "192.0.2.3"},
		},
		{
			name:     "empty string",
			input:    "",
			expected: nil,
		},
		{
			name:     "single IP STRING_AGG format",
			input:    "192.0.2.1",
			expected: []string{"192.0.2.1"},
		},
		{
			name:     "multiple IPs STRING_AGG format",
			input:    "192.0.2.1,192.0.2.2,192.0.2.3",
			expected: []string{"192.0.2.1", "192.0.2.2", "192.0.2.3"},
		},
		{
			name:     "STRING_AGG with spaces",
			input:    "192.0.2.1, 192.0.2.2, 192.0.2.3",
			expected: []string{"192.0.2.1", "192.0.2.2", "192.0.2.3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseIPList(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("len = %d, want %d", len(result), len(tt.expected))
				return
			}
			for i, ip := range result {
				if ip != tt.expected[i] {
					t.Errorf("result[%d] = %s, want %s", i, ip, tt.expected[i])
				}
			}
		})
	}
}

func TestDuckDBStore_GetVPNInfo_NullFields(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewDuckDBStore(db)
	ctx := context.Background()

	if err := store.InitSchema(ctx); err != nil {
		t.Fatalf("failed to init schema: %v", err)
	}

	// Insert with NULL optional fields
	_, err := db.ExecContext(ctx, `
		INSERT INTO vpn_ips (ip_address, provider, country, city, hostname)
		VALUES ('192.0.2.1', 'test', NULL, NULL, NULL)
	`)
	if err != nil {
		t.Fatalf("failed to insert: %v", err)
	}

	result, err := store.GetVPNInfo(ctx, "192.0.2.1")
	if err != nil {
		t.Fatalf("GetVPNInfo error: %v", err)
	}

	if !result.IsVPN {
		t.Error("expected IsVPN to be true")
	}
	if result.ServerCountry != "" {
		t.Errorf("ServerCountry = %s, want empty", result.ServerCountry)
	}
	if result.ServerCity != "" {
		t.Errorf("ServerCity = %s, want empty", result.ServerCity)
	}
	if result.ServerHostname != "" {
		t.Errorf("ServerHostname = %s, want empty", result.ServerHostname)
	}
}

func TestDuckDBStore_GetProvider_NullFields(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewDuckDBStore(db)
	ctx := context.Background()

	if err := store.InitSchema(ctx); err != nil {
		t.Fatalf("failed to init schema: %v", err)
	}

	// Insert with NULL optional fields
	_, err := db.ExecContext(ctx, `
		INSERT INTO vpn_providers (name, display_name, website, server_count, ip_count)
		VALUES ('test', 'Test', NULL, 0, 0)
	`)
	if err != nil {
		t.Fatalf("failed to insert: %v", err)
	}

	provider, err := store.GetProvider(ctx, "test")
	if err != nil {
		t.Fatalf("GetProvider error: %v", err)
	}

	if provider == nil {
		t.Fatal("expected provider, got nil")
	}
	if provider.Website != "" {
		t.Errorf("Website = %s, want empty", provider.Website)
	}
}
