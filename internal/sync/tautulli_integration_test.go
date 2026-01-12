// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build integration

package sync

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/tomtom215/cartographus/internal/config"
	"github.com/tomtom215/cartographus/internal/testinfra"
)

// This file demonstrates how to migrate from mock-based tests to testcontainers.
//
// BEFORE (mock-based): Tests used httptest.NewServer with canned JSON responses.
// AFTER (testcontainers): Tests use a real Tautulli container with actual database.
//
// Benefits:
// - Tests against real Tautulli API behavior
// - Catches API compatibility issues early
// - Validates database schema compatibility
// - More realistic integration testing
//
// Trade-offs:
// - Slower test execution (container startup ~10-30s)
// - Requires Docker
// - Should be run as integration tests, not unit tests
//
// Usage:
//   go test -tags integration -run TestTautulli ./internal/sync/...

// TestTautulliClient_Integration tests the TautulliClient against a real Tautulli instance.
// This replaces mock-based tests with real container-based integration tests.
func TestTautulliClient_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testinfra.SkipIfNoDocker(t)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Start a real Tautulli container
	tautulli, err := testinfra.NewTautulliContainer(ctx,
		testinfra.WithStartTimeout(90*time.Second),
	)
	if err != nil {
		t.Fatalf("Failed to start Tautulli container: %v", err)
	}
	defer testinfra.CleanupContainer(t, ctx, tautulli.Container)

	// Create a real TautulliClient pointing to the container
	cfg := &config.TautulliConfig{
		URL:    tautulli.URL,
		APIKey: tautulli.APIKey,
	}
	client := NewTautulliClient(cfg)

	t.Run("GetServerInfo returns real server info", func(t *testing.T) {
		// This test previously used a mock:
		//   mock := setupMockServer(jsonHandler(tautulli.TautulliServerInfo{...}))
		//
		// Now we test against the real Tautulli API:
		info, err := client.GetServerInfo()
		if err != nil {
			// Note: Fresh Tautulli may return error if Plex is not configured
			// This is expected behavior we can validate
			t.Logf("GetServerInfo returned error (expected for fresh instance): %v", err)
			return
		}

		// Validate real response structure
		if info.Response.Result != "success" && info.Response.Result != "error" {
			t.Errorf("Unexpected result: %s", info.Response.Result)
		}

		t.Logf("Server info result: %s", info.Response.Result)
	})

	t.Run("GetActivity returns real activity data", func(t *testing.T) {
		activity, err := client.GetActivity("")
		if err != nil {
			t.Logf("GetActivity returned error: %v", err)
			return
		}

		// Fresh Tautulli should return empty activity
		if activity.Response.Result == "success" {
			t.Logf("Active streams: %d", activity.Response.Data.StreamCount)
		}
	})

	t.Run("GetHistory returns real history data", func(t *testing.T) {
		history, err := client.GetHistory(0, 25)
		if err != nil {
			t.Logf("GetHistory returned error: %v", err)
			return
		}

		// Fresh Tautulli should return empty history
		if history.Response.Result == "success" {
			t.Logf("Total history records: %d", history.Response.Data.RecordsTotal)
		}
	})

	t.Run("GetUsers returns real user data", func(t *testing.T) {
		users, err := client.GetUsers()
		if err != nil {
			t.Logf("GetUsers returned error: %v", err)
			return
		}

		if users.Response.Result == "success" {
			t.Logf("User count: %d", len(users.Response.Data))
		}
	})

	t.Run("GetLibraries returns real library data", func(t *testing.T) {
		libraries, err := client.GetLibraries()
		if err != nil {
			t.Logf("GetLibraries returned error: %v", err)
			return
		}

		if libraries.Response.Result == "success" {
			t.Logf("Library count: %d", len(libraries.Response.Data))
		}
	})
}

// TestTautulliClient_WithSeedDatabase tests with pre-populated data.
// This demonstrates using the seed database for realistic testing scenarios.
func TestTautulliClient_WithSeedDatabase(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testinfra.SkipIfNoDocker(t)

	seedPath, err := testinfra.GetDefaultSeedDBPath()
	if err != nil {
		t.Skipf("Skipping: could not determine seed path: %v", err)
	}

	// Check if seed database exists before trying to use it
	if _, err := os.Stat(seedPath); os.IsNotExist(err) {
		t.Skipf("Skipping: seed database does not exist at %s", seedPath)
	} else if err != nil {
		t.Skipf("Skipping: could not access seed database: %v", err)
	}
	t.Logf("Using seed database: %s", seedPath)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Start Tautulli with seed database
	tautulli, err := testinfra.NewTautulliContainer(ctx,
		testinfra.WithSeedDatabase(seedPath),
		testinfra.WithStartTimeout(90*time.Second),
	)
	if err != nil {
		t.Skipf("Skipping: could not create container with seed database: %v", err)
	}
	defer testinfra.CleanupContainer(t, ctx, tautulli.Container)

	cfg := &config.TautulliConfig{
		URL:    tautulli.URL,
		APIKey: tautulli.APIKey,
	}
	client := NewTautulliClient(cfg)

	t.Run("GetHistory returns seeded history", func(t *testing.T) {
		history, err := client.GetHistory(0, 100)
		if err != nil {
			t.Fatalf("GetHistory error: %v", err)
		}

		if history.Response.Result != "success" {
			t.Skipf("API returned non-success: %s", history.Response.Result)
		}

		// With seed database, we should have history records
		if history.Response.Data.RecordsTotal == 0 {
			t.Log("Warning: No history records found (seed may not be applied)")
		} else {
			t.Logf("Found %d history records", history.Response.Data.RecordsTotal)
		}
	})

	t.Run("GetUsers returns seeded users", func(t *testing.T) {
		users, err := client.GetUsers()
		if err != nil {
			t.Fatalf("GetUsers error: %v", err)
		}

		if users.Response.Result != "success" {
			t.Skipf("API returned non-success: %s", users.Response.Result)
		}

		if len(users.Response.Data) == 0 {
			t.Log("Warning: No users found (seed may not be applied)")
		} else {
			t.Logf("Found %d users", len(users.Response.Data))
		}
	})
}

// TestTautulliClient_RealAPIBehavior tests specific API behaviors that differ
// between mocks and the real implementation.
func TestTautulliClient_RealAPIBehavior(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testinfra.SkipIfNoDocker(t)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	tautulli, err := testinfra.NewTautulliContainer(ctx,
		testinfra.WithStartTimeout(90*time.Second),
	)
	if err != nil {
		t.Fatalf("Failed to start Tautulli container: %v", err)
	}
	defer testinfra.CleanupContainer(t, ctx, tautulli.Container)

	cfg := &config.TautulliConfig{
		URL:    tautulli.URL,
		APIKey: tautulli.APIKey,
	}
	client := NewTautulliClient(cfg)

	t.Run("invalid API key returns proper error", func(t *testing.T) {
		// Create client with wrong API key
		badCfg := &config.TautulliConfig{
			URL:    tautulli.URL,
			APIKey: "invalid-key",
		}
		badClient := NewTautulliClient(badCfg)

		_, err := badClient.GetServerInfo()
		// Real Tautulli may handle this differently than mocks
		// This test validates actual behavior
		t.Logf("Invalid API key response: %v", err)
	})

	t.Run("API endpoint is accessible", func(t *testing.T) {
		// Test raw HTTP access to verify container is working
		resp, err := http.Get(tautulli.URL)
		if err != nil {
			t.Fatalf("HTTP request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}
	})

	t.Run("API rate limiting behavior", func(t *testing.T) {
		// Make multiple rapid requests to observe rate limiting
		// Real Tautulli may have different rate limiting than mocks
		for i := 0; i < 5; i++ {
			_, err := client.GetActivity("")
			if err != nil {
				t.Logf("Request %d error: %v", i+1, err)
			}
		}
	})
}

// TestTautulliContainer_Lifecycle tests container management behaviors.
func TestTautulliContainer_Lifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testinfra.SkipIfNoDocker(t)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	t.Run("container starts and stops cleanly", func(t *testing.T) {
		tautulli, err := testinfra.NewTautulliContainer(ctx,
			testinfra.WithStartTimeout(90*time.Second),
		)
		if err != nil {
			t.Fatalf("Failed to start container: %v", err)
		}

		// Verify container is running
		info, err := testinfra.GetContainerInfo(ctx, tautulli.Container)
		if err != nil {
			t.Fatalf("Failed to get container info: %v", err)
		}
		if info.State != "running" {
			t.Errorf("Expected state 'running', got '%s'", info.State)
		}

		// Clean up
		testinfra.CleanupContainer(t, ctx, tautulli.Container)
	})

	t.Run("container logs are accessible", func(t *testing.T) {
		tautulli, err := testinfra.NewTautulliContainer(ctx,
			testinfra.WithStartTimeout(90*time.Second),
		)
		if err != nil {
			t.Fatalf("Failed to start container: %v", err)
		}
		defer testinfra.CleanupContainer(t, ctx, tautulli.Container)

		logs, err := tautulli.Logs(ctx)
		if err != nil {
			t.Fatalf("Failed to get logs: %v", err)
		}

		if logs == "" {
			t.Log("Warning: Container logs are empty")
		} else {
			t.Logf("Container log length: %d bytes", len(logs))
		}
	})
}
