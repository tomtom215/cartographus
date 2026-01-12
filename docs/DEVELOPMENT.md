# Development Guide

This document covers development workflow, code conventions, testing strategy, and build procedures for Cartographus.

**Related Documentation**:
- [CLAUDE.md](../CLAUDE.md) - AI assistant guide and project overview
- [PATTERNS.md](./PATTERNS.md) - Code patterns and examples
- [TROUBLESHOOTING.md](./TROUBLESHOOTING.md) - Common issues and solutions

---

## Table of Contents

1. [Local Development Setup](#local-development-setup)
2. [Development Environment](#development-environment)
3. [Making Changes](#making-changes)
4. [Code Conventions](#code-conventions)
5. [Testing Strategy](#testing-strategy)
6. [Build and Deployment](#build-and-deployment)
7. [Pre-Commit Checklist](#pre-commit-checklist)
8. [HTML Template Synchronization](#html-template-synchronization)

---

## Local Development Setup

### Prerequisites

- Go 1.24+ with CGO enabled (DuckDB requirement)
- Node.js 18+ and npm
- Git
- Docker (optional, for container builds)

### Quick Setup

```bash
# 1. Clone repository
git clone https://github.com/tomtom215/cartographus.git
cd map

# 2. Install dependencies
make install-deps

# 3. Copy environment template
cp .env.example .env
# Edit .env with your Tautulli URL and API key

# 4. Build frontend
make build-frontend

# 5. Build and run backend (CGO required)
CGO_ENABLED=1 go build -o cartographus ./cmd/server
./cartographus

# Alternative: Use Docker Compose
docker-compose up -d
```

---

## Development Environment

### Claude Code Web / Sandboxed Environments

**CRITICAL: Complete this setup before ANY Go commands.**

#### Step 1: Environment Variables (Every Session)

```bash
export GOTOOLCHAIN=local
export no_proxy="localhost,127.0.0.1"
export NO_PROXY="localhost,127.0.0.1"
```

#### Step 2: DuckDB Extensions (Once Per Container)

```bash
# Automated setup (recommended):
./scripts/setup-duckdb-extensions.sh

# Or manual setup - see CLAUDE.md for details
```

#### Step 3: Build and Test

```bash
# Build backend
CGO_ENABLED=1 go build -v -o cartographus ./cmd/server

# Run all tests
go test -v -race ./...

# Build with WAL and NATS support
CGO_ENABLED=1 go build -tags "wal,nats" -o cartographus ./cmd/server
```

---

## Configuration

Cartographus uses [Koanf v2](https://github.com/knadh/koanf) for flexible, layered configuration management.

### Configuration Sources (Priority Order)

1. **Built-in Defaults** - Sensible defaults for all optional settings
2. **Config File** - Optional YAML configuration file
3. **Environment Variables** - Highest priority, overrides all other sources

### Using Environment Variables (Recommended for Containers)

Copy and customize the `.env.example` file:

```bash
cp .env.example .env
# Edit .env with your settings
```

Environment variables work exactly as before - full backward compatibility is maintained.

### Using a Config File (New in v1.50)

For persistent configuration, use a YAML file:

```bash
# Copy the example config
cp config.yaml.example config.yaml

# Edit with your settings
vim config.yaml
```

**Config File Search Order:**
1. `CONFIG_PATH` environment variable (if set)
2. `./config.yaml`
3. `./config.yml`
4. `/etc/cartographus/config.yaml`
5. `/etc/cartographus/config.yml`

### Mixing Configuration Sources

Environment variables always take precedence over config file settings:

```yaml
# config.yaml
server:
  port: 3857
  host: "0.0.0.0"
```

```bash
# Override just the port via environment
export HTTP_PORT=8080
./cartographus  # Uses port 8080, host 0.0.0.0
```

### Configuration Struct Tags

The config package uses `koanf` struct tags for type-safe unmarshaling:

```go
type ServerConfig struct {
    Port      int           `koanf:"port"`
    Host      string        `koanf:"host"`
    Timeout   time.Duration `koanf:"timeout"`
}
```

---

## Making Changes

### Backend Changes (Go)

1. Modify code in `internal/` or `cmd/`
2. Format: `gofmt -s -w .`
3. Lint: `go vet ./...`
4. Test: `go test -v -race ./...`
5. Build: `CGO_ENABLED=1 go build -o cartographus ./cmd/server`

### Frontend Changes (TypeScript)

1. Modify code in `web/src/`
2. Type check: `cd web && npx tsc --noEmit`
3. Build: `cd web && npm run build`
4. Watch mode: `cd web && npm run watch`

### Database Changes

**Important**: DuckDB is NOT SQLite. Key differences:
- Use `PERCENTILE_CONT(0.5)` for median (not `MEDIAN()`)
- Spatial queries: `ST_MakePoint()`, `ST_Distance()`, etc.
- No `PRAGMA` statements (use `SET` for configuration)

**Extensions**: Core extensions (spatial, inet, icu) are installed automatically. Community extensions (h3, rapidfuzz, datasketches) require `INSTALL FROM community`:
- **RapidFuzz**: Fuzzy string matching (`rapidfuzz_distance`, `jaro_winkler_similarity`)
- **DataSketches**: Approximate analytics (`datasketch_hll`, `datasketch_kll_quantile`)

### Git Workflow

```bash
# Create feature branch
git checkout -b feature/your-feature-name

# Make commits (conventional commits)
git commit -m "feat: add new analytics endpoint"
git commit -m "fix: resolve null pointer in sync manager"

# Push and create PR
git push origin feature/your-feature-name
```

**Commit Types**: `feat`, `fix`, `docs`, `test`, `refactor`, `perf`, `chore`, `ci`

---

## Code Conventions

### Go Conventions

#### Error Handling

```go
// Good: Explicit error handling
result, err := db.Query(ctx, sql, args...)
if err != nil {
    return fmt.Errorf("query failed: %w", err)
}
defer result.Close()

// Bad: Ignoring errors
result, _ := db.Query(ctx, sql, args...)
```

#### Naming

- **Exported functions**: `PascalCase` (e.g., `GetPlaybacks`)
- **Unexported functions**: `camelCase` (e.g., `parseQueryParams`)
- **Constants**: `PascalCase` (e.g., `DefaultPageSize`)
- **Interfaces**: `-er` suffix (e.g., `Syncer`, `Cacher`)

#### Table-Driven Tests

```go
func TestParseFilter(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    Filter
        wantErr bool
    }{
        {"valid date", "2025-01-01", Filter{...}, false},
        {"invalid date", "invalid", Filter{}, true},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := ParseFilter(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

### TypeScript Conventions

#### Explicit Types and Error Handling

```typescript
// Good
async function fetchPlaybacks(filter: Filter): Promise<Playback[]> {
    try {
        const response = await fetch(`/api/v1/playbacks?${buildQueryString(filter)}`);
        if (!response.ok) {
            throw new Error(`HTTP ${response.status}`);
        }
        return (await response.json()).data as Playback[];
    } catch (error) {
        console.error('Failed to fetch:', error);
        throw error;
    }
}
```

#### Configuration

- **Strict mode**: `"strict": true` in `tsconfig.json`
- **No implicit any**: All types explicit
- **Null checks**: `strictNullChecks` enabled

### Database Query Patterns

```go
// Good: Parameterized queries
query := "SELECT * FROM playback_events WHERE username = ?"
rows, err := db.Query(query, username)

// Bad: SQL injection vulnerability
query := fmt.Sprintf("SELECT * FROM playback_events WHERE username = '%s'", username)
```

---

## Testing Strategy

### Backend Testing

| Test Type | Location | Command |
|-----------|----------|---------|
| Unit Tests | `*_test.go` files | `go test -v ./...` |
| Race Detection | - | `go test -v -race ./...` |
| Integration Tests | `*_integration_test.go`, `*_container_test.go` | `go test -tags integration ./...` |
| Benchmarks | `*_bench_test.go` | `go test -bench=. ./internal/database` |

**Coverage Target**: 78%+ per package (currently 88% average)

### Integration Testing with Testcontainers

Integration tests use real Docker containers instead of mocks. See [ADR-0019](./adr/0019-testcontainers-integration-testing.md).

```bash
# Run all integration tests (requires Docker)
go test -tags integration -v ./internal/sync/... ./internal/import/...

# Run specific integration test
go test -tags integration -v -run TestTautulliClient_Integration ./internal/sync/
```

**Key files:**
- `internal/testinfra/` - Container management infrastructure
- `testdata/tautulli/` - Seed database and documentation
- `internal/sync/tautulli_integration_test.go` - API client integration tests
- `internal/import/tautulli_container_test.go` - Import pipeline integration tests

**Requirements:**
- Docker daemon running
- First run downloads container images (~500MB)
- Tests skip gracefully if Docker unavailable

### Frontend Testing

| Test Type | Location | Command |
|-----------|----------|---------|
| Type Check | - | `cd web && npx tsc --noEmit` |
| E2E Tests | `tests/e2e/` | `npx playwright test` |

**E2E Coverage**: 22 test suites, 337+ test cases across Chromium, Firefox, WebKit

### Test-Driven Development (TDD)

**This project enforces TDD. All new code MUST have tests.**

#### TDD Workflow

1. **RED**: Write failing test first
2. **GREEN**: Write minimal code to pass
3. **REFACTOR**: Clean up while keeping tests green

#### Coverage Requirements

| Code Type | Requirement |
|-----------|-------------|
| API Handlers | 90%+ |
| Database Methods | 90%+ |
| Authentication | 100% |
| Sync Operations | 80%+ |
| Models | 80%+ |

---

## Build and Deployment

### Local Build

```bash
make build              # Frontend + backend
make build-frontend     # Frontend only
make build-backend      # Backend only
make clean              # Clean artifacts
```

### Multi-Platform Binary Builds

```bash
make release-snapshot   # Snapshot for testing
make release            # Official release (requires git tag)
```

**Supported Platforms**: linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64

### Docker Build

```bash
make docker-build       # Build local image

# Multi-arch build
docker buildx build \
    --platform linux/amd64,linux/arm64 \
    -t ghcr.io/tomtom215/cartographus:latest \
    --push .
```

### CI/CD Pipeline

**Build and Test** (`.github/workflows/build-and-test.yml`):
1. Lint: Go vet + gofmt + TypeScript check
2. Test: Unit tests with coverage
3. Build: Frontend + multi-arch Docker images
4. Security Scan: Trivy vulnerability scanning
5. Integration Tests: Container smoke tests

---

## Pre-Commit Checklist

### Automated Pre-Commit (Recommended)

```bash
# Run all pre-commit checks automatically
make pre-commit

# Or install git hooks for automatic checking
./scripts/install-hooks.sh
```

### Manual Pre-Commit Sequence

**Run this sequence before EVERY commit:**

```bash
# 1. CRITICAL: Verify HTML templates are in sync
./scripts/sync-templates.sh --check
# If out of sync, run: ./scripts/sync-templates.sh --sync

# 2. Format and cleanup
gofmt -s -w .
go mod tidy
cd web && npm run build && cd ..

# 3. Lint and verify
go vet ./...
cd web && npx tsc --noEmit && cd ..

# 4. Test
go test -v -race ./...

# 5. Verify counts match documentation
grep -c "r\.\(Get\|Post\|Put\|Delete\|Patch\)" internal/api/chi_router.go  # Should match API docs

# 6. Stage and commit
git add .
git diff --cached  # Review changes
git commit -m "type(scope): description"
```

### Verification Checklist

- [ ] **HTML templates in sync** (prevents E2E failures)
- [ ] All tests pass (0 failures)
- [ ] No race conditions detected
- [ ] Code formatted with gofmt
- [ ] TypeScript compiles without errors
- [ ] Documentation updated if needed
- [ ] No secrets in code

---

## HTML Template Synchronization

### Overview

Cartographus uses two HTML templates:

| File | Purpose | Managed By |
|------|---------|------------|
| `web/public/index.html` | Development template | Frontend developers |
| `internal/templates/index.html.tmpl` | Production template | Sync script |

The production template is a copy of development with Go template syntax added for CSP nonce injection.

### Why This Matters

If the production template drifts from development:
- E2E tests will fail (missing DOM elements)
- Features implemented in development won't work in production
- Debugging becomes extremely difficult

### Commands

```bash
# Check sync status (use in CI/pre-commit)
./scripts/sync-templates.sh --check
make verify-templates

# Update production from development
./scripts/sync-templates.sh --sync
make sync-templates

# View differences
./scripts/sync-templates.sh --diff
```

### Workflow After UI Changes

1. Modify `web/public/index.html`
2. Run `./scripts/sync-templates.sh --sync`
3. Commit both files together:
   ```bash
   git add web/public/index.html internal/templates/index.html.tmpl
   git commit -m "feat(ui): add new feature"
   ```

### Git Hooks

Install the pre-commit hook for automatic validation:

```bash
./scripts/install-hooks.sh
```

This prevents commits when templates are out of sync.

### CI/CD Integration

The CI pipeline includes template validation in the lint workflow:
- Job: `verify-templates`
- Runs on: Every PR and push to main
- Fails if: Templates are out of sync
- Fix instructions: Provided in workflow output

---

## Useful Commands

```bash
# Development
make install-deps          # Install dependencies
make build                 # Build all
make test                  # Unit tests
make test-e2e              # E2E tests
make lint                  # Run linters

# Template Sync (CRITICAL)
make verify-templates      # Check if templates are in sync
make sync-templates        # Update production template
make pre-commit            # Run all pre-commit checks

# Docker
make docker-build          # Build image
make docker-run            # Run with Docker Compose
make docker-logs           # View logs

# Debugging
LOG_LEVEL=debug ./cartographus
go test -v -race ./...
go test -bench=. ./internal/database
```
