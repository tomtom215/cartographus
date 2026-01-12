# ADR-0019: Testcontainers for Integration Testing

**Date**: 2025-12-16
**Status**: Accepted

---

## Context

The Cartographus project has accumulated significant mock-based test infrastructure:

- **220+ test files** with mock implementations
- **4+ categories** of mocks (Tautulli API, DuckDB, NATS, HTTP handlers)
- **Significant maintenance burden** debugging mock servers that drift from real API behavior

Problems with the mock-based approach:

1. **Mock drift**: Mocks get out of sync with real API responses, causing false positives
2. **Incomplete coverage**: Mocks don't test edge cases that occur in real systems
3. **Maintenance cost**: Time spent debugging mock setup vs actual functionality
4. **False confidence**: Tests pass against mocks but fail against real services

The team evaluated three containerization candidates:

| Service | Recommendation | Rationale |
|---------|----------------|-----------|
| **Tautulli** | Highly Recommended | Official Docker images exist, schema is known, API is stable |
| **DuckDB** | Unnecessary | Embedded database, `:memory:` mode is correct approach |
| **Plex** | Not Recommended | Claim token expires every 4 minutes, complex auth flow |

---

## Decision

Adopt **testcontainers-go** for integration testing with real Tautulli containers.

Key design choices:

1. **Use build tags** (`//go:build integration`) to separate unit and integration tests
2. **Graceful degradation**: Tests skip when Docker unavailable
3. **Seed database**: Pre-populated SQLite database for deterministic testing
4. **Container reuse**: Containers are started once per test file, not per test

Why testcontainers-go over alternatives:

| Alternative | Rejected Because |
|-------------|------------------|
| Docker Compose | More complex setup, less Go-native |
| Custom Docker scripts | Harder to maintain, less test integration |
| Remote test environments | Network latency, shared state issues |
| Pure mocks (status quo) | Mock drift, maintenance burden |

---

## Consequences

### Positive

- Tests validate actual API contracts, not assumptions
- No mock drift (mocks getting out of sync with real API)
- Production-equivalent test environments
- Reduced maintenance burden (one seed database vs many mock functions)
- Catches API compatibility issues early
- More realistic integration testing

### Negative

- Slower test execution (container startup ~10-30 seconds)
- Requires Docker daemon access
- Cannot run integration tests in sandboxed/restricted environments
- First-time container pull requires network access
- More complex CI setup

### Neutral

- Unit tests unchanged (still use mocks for fast feedback)
- Hybrid approach: mocks for unit tests, containers for integration tests
- Test data generation moved from code to database seed files

---

## Implementation

### Technical Details

**Package Structure:**

```
internal/testinfra/           # Container management
  doc.go                      # Package documentation
  tautulli.go                 # Tautulli container implementation
  containers.go               # Shared container utilities
  tautulli_test.go            # Infrastructure tests

testdata/tautulli/            # Test data
  seed.sql                    # Schema documentation
  static_seed_data.sql        # Static deterministic SQL data (550 records)
  generate_seed_test.go       # Seed database generator (loads static_seed_data.sql)
  seed.db                     # Pre-populated SQLite (generated in CI)
  README.md                   # Usage documentation
```

**Key Components:**

1. **TautulliContainer**: Manages Tautulli Docker container lifecycle
2. **Seed Database Generator**: Creates 550+ sessions, 10 users, 6 months of data
3. **Integration Test Examples**: Demonstrates migration from mocks

**Container Configuration:**

```go
tautulli, err := testinfra.NewTautulliContainer(ctx,
    testinfra.WithSeedDatabase("/path/to/seed.db"),
    testinfra.WithAPIKey("custom-api-key"),
    testinfra.WithStartTimeout(90*time.Second),
)
```

### Code References

| Component | File | Notes |
|-----------|------|-------|
| Tautulli Container | `internal/testinfra/tautulli.go` | Container lifecycle management |
| Container Utilities | `internal/testinfra/containers.go` | SkipIfNoDocker, CleanupContainer |
| Seed Generator | `testdata/tautulli/generate_seed_test.go` | 550+ sessions, deterministic |
| Sync Integration Tests | `internal/sync/tautulli_integration_test.go` | API client tests |
| Import Integration Tests | `internal/import/tautulli_container_test.go` | Import pipeline tests |

### Dependencies

```go
// go.mod
require (
    github.com/testcontainers/testcontainers-go v0.40.0
)
```

### Build Tags

Integration tests use `//go:build integration` tag:

```bash
# Run unit tests only (default)
go test ./...

# Run integration tests
go test -tags integration ./...

# Run specific integration test
go test -tags integration -run TestTautulliClient_Integration ./internal/sync/
```

---

## Verification

### Verified Claims

| Claim | Source | Verified |
|-------|--------|----------|
| testcontainers-go v0.40.0 supports Tautulli image | `go.mod` | Yes |
| LinuxServer Tautulli image exists | Docker Hub | Yes |
| Tautulli SQLite schema has 3 main tables | Tautulli source code | Yes |
| Build tags exclude integration tests from unit runs | Go build system | Yes |

### Test Coverage

- **Unit tests**: Continue running without Docker (existing 8861+ tests)
- **Integration tests**: Require Docker, tagged with `//go:build integration`
- **Coverage target**: Integration tests supplement, not replace, unit test coverage

### CI Verification

Integration tests will run in CI when:
1. Docker daemon is available (self-hosted runner)
2. Network access allows container image pull
3. `go test -tags integration` is invoked

---

## Self-Hosted Runner Requirements

**No additional setup required.** The existing self-hosted runner already has:

- Docker Engine 24.0+ with daemon access
- `github-runner` user in `docker` group
- Network access for pulling images

Testcontainers-go will work out of the box because:
1. It uses the Docker daemon (not Docker Compose)
2. Container images are cached between runs
3. Tests gracefully skip if Docker unavailable

---

## Related ADRs

- [ADR-0011](0011-ci-cd-infrastructure.md): Self-hosted CI/CD runners (provides Docker infrastructure)
- [ADR-0014](0014-tautulli-database-import.md): Tautulli database import (uses same SQLite schema)
- [ADR-0001](0001-use-duckdb-for-analytics.md): DuckDB for analytics (sqlite_scanner reads Tautulli DB)

---

## References

- [testcontainers-go Documentation](https://golang.testcontainers.org/)
- [LinuxServer Tautulli Image](https://hub.docker.com/r/linuxserver/tautulli)
- [Tautulli GitHub Repository](https://github.com/Tautulli/Tautulli)
- Internal: `testdata/tautulli/README.md` (usage documentation)
- Internal: `docs/SELF_HOSTED_RUNNER.md` (runner setup)
