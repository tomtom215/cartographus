// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package vpn

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestUpdater_UpdateFromURL(t *testing.T) {
	// Create a test server with sample VPN data
	sampleData := `{
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

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(sampleData))
	}))
	defer server.Close()

	// Create service with lookup only (no DB)
	lookup := NewLookup()
	service := &Service{
		config:   DefaultConfig(),
		lookup:   lookup,
		importer: NewImporter(lookup),
	}

	config := DefaultUpdaterConfig()
	config.SourceURL = server.URL
	updater := NewUpdater(service, config)

	ctx := context.Background()
	err := updater.UpdateFromURL(ctx, server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify data was imported
	if !lookup.ContainsIP("198.51.100.1") {
		t.Error("expected IP to be imported")
	}

	// Check status
	status := updater.GetStatus()
	if status.LastSuccessfulUpdate.IsZero() {
		t.Error("expected LastSuccessfulUpdate to be set")
	}
	if status.DataHash == "" {
		t.Error("expected DataHash to be set")
	}
	if status.LastError != "" {
		t.Errorf("unexpected error in status: %s", status.LastError)
	}
	if status.LastImportResult == nil {
		t.Error("expected LastImportResult to be set")
	}
	if status.LastImportResult.IPsImported != 2 {
		t.Errorf("expected 2 IPs imported, got %d", status.LastImportResult.IPsImported)
	}
}

func TestUpdater_UpdateFromURL_NoChangeIfSameHash(t *testing.T) {
	sampleData := `{
		"testprovider": {
			"version": 1,
			"timestamp": 1700000000,
			"servers": [{"country": "Test", "ips": ["192.0.2.1"]}]
		}
	}`

	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(sampleData))
	}))
	defer server.Close()

	lookup := NewLookup()
	service := &Service{
		config:   DefaultConfig(),
		lookup:   lookup,
		importer: NewImporter(lookup),
	}

	config := DefaultUpdaterConfig()
	updater := NewUpdater(service, config)

	ctx := context.Background()

	// First update
	err := updater.UpdateFromURL(ctx, server.URL)
	if err != nil {
		t.Fatalf("first update failed: %v", err)
	}

	firstHash := updater.GetStatus().DataHash

	// Second update with same data
	err = updater.UpdateFromURL(ctx, server.URL)
	if err != nil {
		t.Fatalf("second update failed: %v", err)
	}

	secondHash := updater.GetStatus().DataHash

	// Hashes should match
	if firstHash != secondHash {
		t.Error("expected hashes to match for same data")
	}

	// Should have made 2 requests (both fetched, but second detected no change)
	if requestCount != 2 {
		t.Errorf("expected 2 requests, got %d", requestCount)
	}
}

func TestUpdater_UpdateFromURL_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	lookup := NewLookup()
	service := &Service{
		config:   DefaultConfig(),
		lookup:   lookup,
		importer: NewImporter(lookup),
	}

	config := DefaultUpdaterConfig()
	config.RetryAttempts = 0 // No retries for faster test
	updater := NewUpdater(service, config)

	ctx := context.Background()
	err := updater.UpdateFromURL(ctx, server.URL)
	if err == nil {
		t.Error("expected error for HTTP 500")
	}

	status := updater.GetStatus()
	if status.LastError == "" {
		t.Error("expected LastError to be set")
	}
}

func TestUpdater_UpdateFromURL_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("not valid json"))
	}))
	defer server.Close()

	lookup := NewLookup()
	service := &Service{
		config:   DefaultConfig(),
		lookup:   lookup,
		importer: NewImporter(lookup),
	}

	config := DefaultUpdaterConfig()
	config.RetryAttempts = 0
	updater := NewUpdater(service, config)

	ctx := context.Background()
	err := updater.UpdateFromURL(ctx, server.URL)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestUpdater_RetryLogic(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		// Succeed on third attempt
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"testprovider":{"version":1,"timestamp":1,"servers":[{"country":"Test","ips":["192.0.2.1"]}]}}`))
	}))
	defer server.Close()

	lookup := NewLookup()
	service := &Service{
		config:   DefaultConfig(),
		lookup:   lookup,
		importer: NewImporter(lookup),
	}

	config := DefaultUpdaterConfig()
	config.RetryAttempts = 3
	config.RetryDelay = 10 * time.Millisecond // Fast retries for test
	updater := NewUpdater(service, config)

	ctx := context.Background()
	err := updater.UpdateFromURL(ctx, server.URL)
	if err != nil {
		t.Fatalf("unexpected error after retries: %v", err)
	}

	if requestCount != 3 {
		t.Errorf("expected 3 requests (2 failures + 1 success), got %d", requestCount)
	}
}

func TestUpdater_ConcurrentUpdatePrevention(t *testing.T) {
	// Server that takes a while to respond
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"testprovider":{"version":1,"timestamp":1,"servers":[{"country":"Test","ips":["192.0.2.1"]}]}}`))
	}))
	defer server.Close()

	lookup := NewLookup()
	service := &Service{
		config:   DefaultConfig(),
		lookup:   lookup,
		importer: NewImporter(lookup),
	}

	config := DefaultUpdaterConfig()
	updater := NewUpdater(service, config)

	ctx := context.Background()

	// Start first update in goroutine
	done := make(chan error)
	go func() {
		done <- updater.UpdateFromURL(ctx, server.URL)
	}()

	// Wait a bit for first update to start
	time.Sleep(20 * time.Millisecond)

	// Try second update - should fail immediately
	err := updater.UpdateFromURL(ctx, server.URL)
	if err == nil {
		t.Error("expected error for concurrent update")
	}

	// Wait for first update to complete
	if firstErr := <-done; firstErr != nil {
		t.Fatalf("first update failed: %v", firstErr)
	}
}

func TestUpdater_ExtractProviderVersions(t *testing.T) {
	lookup := NewLookup()
	service := &Service{
		config:   DefaultConfig(),
		lookup:   lookup,
		importer: NewImporter(lookup),
	}

	updater := NewUpdater(service, nil)

	data := []byte(`{
		"version": 1,
		"nordvpn": {"version": 5, "timestamp": 1700000000, "servers": []},
		"expressvpn": {"version": 3, "timestamp": 1700000001, "servers": []},
		"mullvad": {"version": 7, "timestamp": 1700000002, "servers": []}
	}`)

	versions := updater.extractProviderVersions(data)

	if versions["nordvpn"] != 5 {
		t.Errorf("expected nordvpn version 5, got %d", versions["nordvpn"])
	}
	if versions["expressvpn"] != 3 {
		t.Errorf("expected expressvpn version 3, got %d", versions["expressvpn"])
	}
	if versions["mullvad"] != 7 {
		t.Errorf("expected mullvad version 7, got %d", versions["mullvad"])
	}
	if _, ok := versions["version"]; ok {
		t.Error("should skip root 'version' field")
	}
}

func TestUpdater_GetStatus(t *testing.T) {
	lookup := NewLookup()
	service := &Service{
		config:   DefaultConfig(),
		lookup:   lookup,
		importer: NewImporter(lookup),
	}

	config := DefaultUpdaterConfig()
	config.SourceURL = "https://example.com/servers.json"
	updater := NewUpdater(service, config)

	status := updater.GetStatus()

	if status.SourceURL != "https://example.com/servers.json" {
		t.Errorf("expected source URL to match config, got %s", status.SourceURL)
	}
	if status.IsUpdating {
		t.Error("should not be updating initially")
	}
}

func TestUpdater_EnableDisable(t *testing.T) {
	lookup := NewLookup()
	service := &Service{
		config:   DefaultConfig(),
		lookup:   lookup,
		importer: NewImporter(lookup),
	}

	config := DefaultUpdaterConfig()
	config.Enabled = true
	updater := NewUpdater(service, config)

	if !updater.IsEnabled() {
		t.Error("expected updater to be enabled")
	}

	updater.SetEnabled(false)
	if updater.IsEnabled() {
		t.Error("expected updater to be disabled")
	}

	updater.SetEnabled(true)
	if !updater.IsEnabled() {
		t.Error("expected updater to be re-enabled")
	}
}

func TestUpdater_HashCalculation(t *testing.T) {
	data := []byte(`{"test": "data"}`)
	expectedHash := sha256.Sum256(data)
	expectedHashStr := hex.EncodeToString(expectedHash[:])

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(data)
	}))
	defer server.Close()

	lookup := NewLookup()
	service := &Service{
		config:   DefaultConfig(),
		lookup:   lookup,
		importer: NewImporter(lookup),
	}

	config := DefaultUpdaterConfig()
	config.RetryAttempts = 0
	updater := NewUpdater(service, config)

	ctx := context.Background()
	// This will fail to import (invalid format) but hash should still be calculated
	updater.UpdateFromURL(ctx, server.URL)

	// The hash is only set on successful import, so let's use valid data
	validServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		validData := `{"provider":{"version":1,"timestamp":1,"servers":[{"country":"Test","ips":["192.0.2.1"]}]}}`
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(validData))
	}))
	defer validServer.Close()

	err := updater.UpdateFromURL(ctx, validServer.URL)
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}

	status := updater.GetStatus()
	if status.DataHash == "" {
		t.Error("expected hash to be set")
	}

	// Calculate expected hash for valid data
	validData := `{"provider":{"version":1,"timestamp":1,"servers":[{"country":"Test","ips":["192.0.2.1"]}]}}`
	validHash := sha256.Sum256([]byte(validData))
	validHashStr := hex.EncodeToString(validHash[:])

	if status.DataHash != validHashStr {
		t.Errorf("hash mismatch: expected %s, got %s", validHashStr, status.DataHash)
	}

	// Verify expectedHashStr is different (different data)
	if expectedHashStr == validHashStr {
		t.Error("test data hashes should be different")
	}
}

func TestUpdater_SetUpdateInterval(t *testing.T) {
	lookup := NewLookup()
	service := &Service{
		config:   DefaultConfig(),
		lookup:   lookup,
		importer: NewImporter(lookup),
	}

	config := DefaultUpdaterConfig()
	updater := NewUpdater(service, config)

	// Check initial interval
	initialInterval := updater.GetConfig().UpdateInterval
	if initialInterval != DefaultUpdateInterval {
		t.Errorf("expected initial interval %v, got %v", DefaultUpdateInterval, initialInterval)
	}

	// Change interval
	newInterval := 6 * time.Hour
	updater.SetUpdateInterval(newInterval)

	// Verify change
	updatedInterval := updater.GetConfig().UpdateInterval
	if updatedInterval != newInterval {
		t.Errorf("expected updated interval %v, got %v", newInterval, updatedInterval)
	}
}

func TestUpdater_GetConfig(t *testing.T) {
	lookup := NewLookup()
	service := &Service{
		config:   DefaultConfig(),
		lookup:   lookup,
		importer: NewImporter(lookup),
	}

	config := &UpdaterConfig{
		Enabled:        true,
		SourceURL:      "https://custom.example.com/servers.json",
		UpdateInterval: 12 * time.Hour,
		HTTPTimeout:    30 * time.Second,
		RetryAttempts:  5,
		RetryDelay:     10 * time.Second,
	}
	updater := NewUpdater(service, config)

	retrievedConfig := updater.GetConfig()

	if !retrievedConfig.Enabled {
		t.Error("expected Enabled to be true")
	}
	if retrievedConfig.SourceURL != "https://custom.example.com/servers.json" {
		t.Errorf("expected custom SourceURL, got %s", retrievedConfig.SourceURL)
	}
	if retrievedConfig.UpdateInterval != 12*time.Hour {
		t.Errorf("expected 12h interval, got %v", retrievedConfig.UpdateInterval)
	}
	if retrievedConfig.HTTPTimeout != 30*time.Second {
		t.Errorf("expected 30s timeout, got %v", retrievedConfig.HTTPTimeout)
	}
	if retrievedConfig.RetryAttempts != 5 {
		t.Errorf("expected 5 retry attempts, got %d", retrievedConfig.RetryAttempts)
	}
	if retrievedConfig.RetryDelay != 10*time.Second {
		t.Errorf("expected 10s retry delay, got %v", retrievedConfig.RetryDelay)
	}
}

func TestDefaultUpdaterConfig(t *testing.T) {
	config := DefaultUpdaterConfig()

	if config.Enabled {
		t.Error("expected Enabled to be false by default")
	}
	if config.SourceURL != DefaultGluetunURL {
		t.Errorf("expected default gluetun URL, got %s", config.SourceURL)
	}
	if config.UpdateInterval != DefaultUpdateInterval {
		t.Errorf("expected %v interval, got %v", DefaultUpdateInterval, config.UpdateInterval)
	}
	if config.HTTPTimeout != DefaultHTTPTimeout {
		t.Errorf("expected %v timeout, got %v", DefaultHTTPTimeout, config.HTTPTimeout)
	}
	if config.RetryAttempts != 3 {
		t.Errorf("expected 3 retry attempts, got %d", config.RetryAttempts)
	}
	if config.RetryDelay != 5*time.Second {
		t.Errorf("expected 5s retry delay, got %v", config.RetryDelay)
	}
}

func TestUpdater_NewUpdaterWithNilConfig(t *testing.T) {
	lookup := NewLookup()
	service := &Service{
		config:   DefaultConfig(),
		lookup:   lookup,
		importer: NewImporter(lookup),
	}

	updater := NewUpdater(service, nil)

	config := updater.GetConfig()
	if config.SourceURL != DefaultGluetunURL {
		t.Errorf("expected default config to be used, got SourceURL: %s", config.SourceURL)
	}
}

func TestUpdater_FetchContextCancellation(t *testing.T) {
	// Server that takes a while to respond
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"provider":{"version":1,"timestamp":1,"servers":[{"country":"Test","ips":["192.0.2.1"]}]}}`))
	}))
	defer server.Close()

	lookup := NewLookup()
	service := &Service{
		config:   DefaultConfig(),
		lookup:   lookup,
		importer: NewImporter(lookup),
	}

	config := DefaultUpdaterConfig()
	config.RetryAttempts = 0
	config.RetryDelay = 100 * time.Millisecond
	updater := NewUpdater(service, config)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := updater.UpdateFromURL(ctx, server.URL)
	if err == nil {
		t.Error("expected error from context cancellation")
	}
}

func TestUpdater_ExtractProviderVersions_InvalidJSON(t *testing.T) {
	lookup := NewLookup()
	service := &Service{
		config:   DefaultConfig(),
		lookup:   lookup,
		importer: NewImporter(lookup),
	}

	updater := NewUpdater(service, nil)

	versions := updater.extractProviderVersions([]byte("not valid json"))
	if versions != nil {
		t.Error("expected nil for invalid JSON")
	}
}

func TestUpdater_ExtractProviderVersions_InvalidProviderFormat(t *testing.T) {
	lookup := NewLookup()
	service := &Service{
		config:   DefaultConfig(),
		lookup:   lookup,
		importer: NewImporter(lookup),
	}

	updater := NewUpdater(service, nil)

	// Provider data without version field
	data := []byte(`{
		"validprovider": {"version": 5, "servers": []},
		"invalidprovider": "not an object"
	}`)

	versions := updater.extractProviderVersions(data)
	if versions["validprovider"] != 5 {
		t.Errorf("expected validprovider version 5, got %d", versions["validprovider"])
	}
	// invalidprovider should be skipped (unmarshal fails)
}
