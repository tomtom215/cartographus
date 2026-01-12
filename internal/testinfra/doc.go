// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package testinfra provides test infrastructure for integration testing with containers.
//
// This package uses testcontainers-go to manage Docker containers for integration tests,
// providing realistic testing environments that closely match production.
//
// # Tautulli Container
//
// The TautulliContainer provides a real Tautulli instance for testing API integration:
//
//	func TestTautulliSync(t *testing.T) {
//	    ctx := context.Background()
//	    tautulli, err := testinfra.NewTautulliContainer(ctx,
//	        testinfra.WithSeedDatabase("/path/to/seed.db"),
//	    )
//	    if err != nil {
//	        t.Fatal(err)
//	    }
//	    defer tautulli.Terminate(ctx)
//
//	    client := sync.NewTautulliClient(&config.TautulliConfig{
//	        URL:    tautulli.URL,
//	        APIKey: tautulli.APIKey,
//	    })
//
//	    // Test with real Tautulli API
//	    history, err := client.GetHistory(10)
//	    // ...
//	}
//
// # Benefits Over Mocks
//
// Using real containers provides several advantages:
//   - Tests validate actual API contracts
//   - No mock drift (mocks getting out of sync with real API)
//   - Tests run against production-equivalent services
//   - Reduces maintenance burden (one seed database vs many mock functions)
//
// # CI Considerations
//
// These tests require Docker and network access. In CI:
//   - Self-hosted runners have Docker pre-installed
//   - Container images are cached between runs
//   - Tests are skipped gracefully if Docker is unavailable
//
// # Network Requirements
//
// First run may need to download container images. Subsequent runs use cached images.
package testinfra
