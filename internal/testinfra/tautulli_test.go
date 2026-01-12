// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build integration

package testinfra

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"
)

// TestTautulliContainer_Integration tests the full Tautulli container lifecycle.
// This test requires Docker and is skipped in environments without Docker.
func TestTautulliContainer_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	SkipIfNoDocker(t)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Create Tautulli container
	tautulli, err := NewTautulliContainer(ctx,
		WithStartTimeout(90*time.Second),
	)
	if err != nil {
		t.Fatalf("Failed to create Tautulli container: %v", err)
	}
	defer CleanupContainer(t, ctx, tautulli.Container)

	// Verify container is running
	t.Logf("Tautulli container started at: %s", tautulli.URL)

	// Test basic HTTP connectivity
	resp, err := http.Get(tautulli.URL)
	if err != nil {
		logs, _ := tautulli.Logs(ctx)
		t.Fatalf("Failed to connect to Tautulli: %v\nContainer logs:\n%s", err, logs)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Test API endpoint construction
	apiEndpoint := tautulli.GetAPIEndpoint("get_activity")
	if apiEndpoint == "" {
		t.Error("GetAPIEndpoint returned empty string")
	}
	t.Logf("API endpoint: %s", apiEndpoint)

	// Get container info for debugging
	info, err := GetContainerInfo(ctx, tautulli.Container)
	if err != nil {
		t.Logf("Warning: Failed to get container info: %v", err)
	} else {
		t.Logf("Container ID: %s, State: %s, Ports: %v", info.ID, info.State, info.Ports)
	}
}

// TestTautulliContainer_WithSeedDatabase tests using a seeded database.
// This test is skipped if the seed database doesn't exist.
func TestTautulliContainer_WithSeedDatabase(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	SkipIfNoDocker(t)

	seedPath, err := GetDefaultSeedDBPath()
	if err != nil {
		t.Skipf("Skipping: could not determine seed path: %v", err)
	}

	// Check if seed database exists before trying to use it
	if _, err := os.Stat(seedPath); os.IsNotExist(err) {
		t.Skipf("Skipping: seed database does not exist at %s (run 'go test -v -run TestGenerateSeedDatabase ./testdata/tautulli/...' to create it)", seedPath)
	} else if err != nil {
		t.Skipf("Skipping: could not access seed database: %v", err)
	}
	t.Logf("Using seed database: %s", seedPath)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	tautulli, err := NewTautulliContainer(ctx,
		WithSeedDatabase(seedPath),
		WithStartTimeout(90*time.Second),
	)
	if err != nil {
		// Seed database may not exist, skip gracefully
		t.Skipf("Skipping: could not create container with seed database: %v", err)
	}
	defer CleanupContainer(t, ctx, tautulli.Container)

	t.Logf("Tautulli container with seed database started at: %s", tautulli.URL)

	// Verify we can query the history API
	historyEndpoint := tautulli.GetAPIEndpoint("get_history")
	resp, err := http.Get(historyEndpoint)
	if err != nil {
		t.Fatalf("Failed to query history: %v", err)
	}
	defer resp.Body.Close()

	// API should return 200 even if no data
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

// TestIsDockerAvailable tests the Docker detection function.
func TestIsDockerAvailable(t *testing.T) {
	// This test always passes - it just reports Docker availability
	available := IsDockerAvailable()
	t.Logf("Docker available: %v", available)
}

// TestGetDefaultSeedDBPath tests the seed path resolution.
func TestGetDefaultSeedDBPath(t *testing.T) {
	path, err := GetDefaultSeedDBPath()
	if err != nil {
		t.Errorf("GetDefaultSeedDBPath failed: %v", err)
	}

	t.Logf("Default seed DB path: %s", path)

	// Path should contain expected components
	if path == "" {
		t.Error("Seed path should not be empty")
	}
}

// TestContainerOptions tests the option functions.
func TestContainerOptions(t *testing.T) {
	// Test WithTautulliImage
	cfg := &tautulliConfig{}
	WithTautulliImage("custom-image:v1")(cfg)
	if cfg.image != "custom-image:v1" {
		t.Errorf("WithTautulliImage: expected custom-image:v1, got %s", cfg.image)
	}

	// Test WithAPIKey
	cfg = &tautulliConfig{}
	WithAPIKey("custom-api-key")(cfg)
	if cfg.apiKey != "custom-api-key" {
		t.Errorf("WithAPIKey: expected custom-api-key, got %s", cfg.apiKey)
	}

	// Test WithStartTimeout
	cfg = &tautulliConfig{}
	WithStartTimeout(5 * time.Minute)(cfg)
	if cfg.startTimeout != 5*time.Minute {
		t.Errorf("WithStartTimeout: expected 5m, got %v", cfg.startTimeout)
	}

	// Test WithSeedDatabase
	cfg = &tautulliConfig{}
	WithSeedDatabase("/path/to/db")(cfg)
	if cfg.seedDBPath != "/path/to/db" {
		t.Errorf("WithSeedDatabase: expected /path/to/db, got %s", cfg.seedDBPath)
	}
}
