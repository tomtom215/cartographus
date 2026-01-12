# Tautulli Test Data

**Last Verified**: 2026-01-11

This directory contains test infrastructure for generating realistic Tautulli databases used in integration testing with testcontainers.

## Files

| File | Description |
|------|-------------|
| `seed.sql` | Complete Tautulli SQLite schema with documentation |
| `generate_seed_test.go` | Test-based seed generator (runs in CI with network access) |
| `seed.db` | Generated SQLite database with 550+ sessions (generated in CI) |

## Generating the Seed Database

The seed database is generated automatically in CI when tests run. To regenerate locally:

```bash
# Ensure DuckDB extensions are installed
./scripts/setup-duckdb-extensions.sh

# Run the generator test
go test -v -run TestGenerateSeedDatabase ./testdata/tautulli/
```

## Seed Data Contents

The generated database contains:

- **10 users** with different viewing habits:
  - admin (balanced viewer)
  - moviebuff (80% movies)
  - tvaddict (85% TV)
  - musiclover (80% music)
  - familyaccount (mixed content)
  - nightowl (late night viewer)
  - casualviewer (balanced)
  - mobilewatcher (mobile device)
  - weekendwarrior (weekend viewer)
  - bingewatcher (75% TV binge)

- **550+ playback sessions** across 6 months

- **Media types**:
  - 50 movies
  - 20 TV shows with multiple seasons/episodes
  - 15 music artists with albums/tracks

- **Platforms**: Chrome, Roku, Android TV, iOS, Samsung TV, Apple TV, Firefox, PlayStation, Xbox

- **Geographic locations**: New York, Los Angeles, Chicago, London, Berlin, Tokyo, Sydney, Toronto, Paris, Singapore

## Schema Reference

The database uses Tautulli's standard schema with three main tables:

1. **session_history** - Core playback session data (user, timestamps, device info)
2. **session_history_metadata** - Media metadata (title, year, genres, etc.)
3. **session_history_media_info** - Stream quality and transcoding information

See `seed.sql` for complete schema documentation.

## Usage in Tests

The seed database is used by testcontainers to spin up real Tautulli instances:

```go
import "github.com/tomtom215/cartographus/internal/testinfra"

func TestWithTautulli(t *testing.T) {
    ctx := context.Background()

    // Start Tautulli container with seeded database
    tautulli, err := testinfra.NewTautulliContainer(ctx)
    if err != nil {
        t.Fatal(err)
    }
    defer tautulli.Terminate(ctx)

    // Use the container's URL for API calls
    client := sync.NewTautulliClient(&config.TautulliConfig{
        URL:    tautulli.URL,
        APIKey: tautulli.APIKey,
    })

    // Test against real Tautulli API
    history, err := client.GetHistory(10)
    // ...
}
```

## Deterministic Generation

The seed generator uses a fixed random seed (42) to ensure reproducible data across runs. This means:

- Same session IDs
- Same timestamps (relative to generation time)
- Same user distribution
- Same media selection

This reproducibility is important for test stability.

## Running Integration Tests

Integration tests using testcontainers are tagged with `//go:build integration`. To run them:

```bash
# Run all integration tests
go test -tags integration -v ./internal/sync/... ./internal/import/...

# Run specific integration test
go test -tags integration -v -run TestTautulliClient_Integration ./internal/sync/

# Skip short (unit) tests, run only integration
go test -tags integration -v -short=false ./...
```

### Example Test Files

- `internal/sync/tautulli_integration_test.go` - API client integration tests
- `internal/import/tautulli_container_test.go` - Import pipeline integration tests
- `internal/testinfra/tautulli_test.go` - Container infrastructure tests

### Migration from Mocks

The integration tests demonstrate migrating from mock-based tests to testcontainers:

**Before (mock-based):**
```go
mock := setupMockServer(jsonHandler(tautulli.TautulliServerInfo{...}))
defer mock.close()
info, err := mock.client.GetServerInfo()
```

**After (testcontainers):**
```go
tautulli, err := testinfra.NewTautulliContainer(ctx)
defer testinfra.CleanupContainer(t, ctx, tautulli.Container)
client := NewTautulliClient(&config.TautulliConfig{URL: tautulli.URL, APIKey: tautulli.APIKey})
info, err := client.GetServerInfo()
```

## CI Integration

Integration tests run on the self-hosted runner which has Docker pre-installed. The workflow:

1. Generates `seed.db` if missing (via `TestGenerateSeedDatabase`)
2. Pulls Tautulli container image (cached between runs)
3. Runs integration tests with real containers
4. Cleans up containers after tests complete
