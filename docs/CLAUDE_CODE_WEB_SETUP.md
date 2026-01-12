# Claude Code Web Environment Setup

**CRITICAL: This setup is REQUIRED at the start of EVERY Claude Code Web session.**

This document provides the complete environment setup procedure for developing Cartographus in Claude Code Web. Without this setup, Go builds, tests, and DuckDB operations will fail.

**Last Updated**: 2026-01-05
**Environment**: Claude Code Web (Ubuntu 24.04.3 LTS, Linux 4.4.0)
**Related**: [DIAGNOSTIC_REPORT.md](./working/DIAGNOSTIC_REPORT.md) - Full technical analysis

---

## Quick Start (Copy-Paste Ready)

```bash
# REQUIRED: Run this at the START of EVERY session
export GOTOOLCHAIN=local
export no_proxy="localhost,127.0.0.1"
export NO_PROXY="localhost,127.0.0.1"

# Setup DuckDB extensions (creates ~/.duckdb/extensions/v1.4.3/linux_amd64/)
./scripts/setup-duckdb-extensions.sh

# Build the project (MUST include -tags "wal,nats" for full functionality)
CGO_ENABLED=1 go build -tags "wal,nats" -v -o cartographus ./cmd/server

# Verify build
ls -lh cartographus  # Should be ~102MB
```

**IMPORTANT**: The `-tags "wal,nats"` flag is REQUIRED for all Go commands. Without it:
- NATS event processing will not be compiled
- WAL (Write-Ahead Log) support will not be compiled
- Runtime errors like `NATS_ENABLED=true but NATS support not compiled` will occur

---

## Why This Setup is Required

Claude Code Web runs in a sandboxed container with specific network restrictions:

| Issue | Impact | Workaround |
|-------|--------|------------|
| DNS Resolution Broken | Go toolchain downloads fail | `GOTOOLCHAIN=local` |
| no_proxy blocks googleapis.com | Module downloads fail | Override `no_proxy` |
| DuckDB uses HTTP by default | Extension downloads return 403 | Use setup script (HTTPS) |

Without the environment variables, you'll see errors like:
```
dial tcp: lookup storage.googleapis.com on [::1]:53: connection refused
```

---

## Detailed Setup Steps

### Step 1: Set Environment Variables (CRITICAL)

These variables MUST be set before any Go commands:

```bash
# Prevent Go from trying to download newer toolchains
export GOTOOLCHAIN=local

# Override no_proxy to allow proxy for googleapis.com
# (Default no_proxy blocks *.googleapis.com which breaks Go modules)
export no_proxy="localhost,127.0.0.1"
export NO_PROXY="localhost,127.0.0.1"
```

**Verification:**
```bash
go version                    # Should show: go version go1.24.7 linux/amd64
go env GOTOOLCHAIN           # Should show: local
```

### Step 2: Setup DuckDB Extensions

DuckDB extensions must be downloaded before tests/builds that use spatial queries:

```bash
./scripts/setup-duckdb-extensions.sh
```

**Expected Output:**
```
Setting up DuckDB extensions for Claude Code Web...
DuckDB Version: v1.4.3
Platform: linux_amd64
Target directory: /root/.duckdb/extensions/v1.4.3/linux_amd64
Extensions directory ready: /root/.duckdb/extensions/v1.4.3/linux_amd64
Downloading httpfs extension...
✓ httpfs installed (24M)
Downloading spatial extension...
✓ spatial installed (84M)
...
All DuckDB extensions installed successfully!
```

**If extensions already exist**, the script will skip them:
```
✓ spatial already exists (84M)
```

### Step 3: Build the Project

```bash
# MUST include -tags "wal,nats" for full functionality
CGO_ENABLED=1 go build -tags "wal,nats" -v -o cartographus ./cmd/server
```

**Expected Result:**
- Binary: `./cartographus` (approximately 102MB)
- Build time: 1-3 minutes (first build with CGO compilation)

**Verification:**
```bash
ls -lh cartographus     # Should be ~102MB
file cartographus       # Should show: ELF 64-bit LSB executable, x86-64
```

### Step 4: Frontend Setup (Optional)

Only needed if working on frontend code:

```bash
cd web
npm ci                # Install dependencies (first time only)
npm run build         # Build TypeScript bundle
npx tsc --noEmit      # Type checking
cd ..
```

---

## Complete Extension List

The setup script downloads these extensions to `~/.duckdb/extensions/v1.4.3/linux_amd64/`:

### Core Extensions (from extensions.duckdb.org)

| Extension | Size | Purpose |
|-----------|------|---------|
| httpfs | 24MB | HTTP file system access |
| spatial | 84MB | Geographic/spatial queries (ST_* functions) |
| icu | 34MB | Unicode and timezone support |
| inet | 22MB | IP address type and functions |
| json | 48MB | JSON parsing and querying |
| fts | 24MB | Full-text search |
| sqlite_scanner | 48MB | SQLite database import (Tautulli) |

### Community Extensions (from community-extensions.duckdb.org)

| Extension | Size | Purpose |
|-----------|------|---------|
| h3 | 7.5MB | H3 hexagonal geospatial indexing |
| rapidfuzz | 7.4MB | Fuzzy string matching |
| datasketches | 14MB | Approximate analytics (HyperLogLog, etc.) |

**Total Size**: ~313MB for all extensions

---

## Running Tests

**IMPORTANT**: ALWAYS include `-tags "wal,nats"` in all Go test commands.

### Test Specific Packages

```bash
# These packages don't require DuckDB extensions:
go test -tags "wal,nats" -v -race ./internal/auth/...
go test -tags "wal,nats" -v -race ./internal/config/...
go test -tags "wal,nats" -v -race ./internal/cache/...
go test -tags "wal,nats" -v -race ./internal/middleware/...

# These packages REQUIRE DuckDB extensions to be set up first:
go test -tags "wal,nats" -v -race ./internal/database/...
go test -tags "wal,nats" -v -race ./internal/api/...
```

### Run All Tests

```bash
go test -tags "wal,nats" -v -race ./...
```

### Linting

```bash
# Go linting (MUST include build tags)
go vet -tags "wal,nats" ./...

# golangci-lint is pre-installed
golangci-lint run ./...

# TypeScript checking
cd web && npx tsc --noEmit
```

---

## Troubleshooting

### Error: "dial tcp: lookup storage.googleapis.com on [::1]:53: connection refused"

**Cause**: DNS resolution is broken and Go is trying to download the toolchain.

**Fix**: Set environment variables:
```bash
export GOTOOLCHAIN=local
export no_proxy="localhost,127.0.0.1"
export NO_PROXY="localhost,127.0.0.1"
```

### Error: "failed to install sqlite_scanner extension"

**Cause**: Extension not downloaded or version mismatch.

**Fix**: Run the setup script or download manually:
```bash
./scripts/setup-duckdb-extensions.sh

# Or manually:
cd ~/.duckdb/extensions/v1.4.3/linux_amd64
curl -sf --retry 3 -o sqlite_scanner.duckdb_extension.gz \
  "https://extensions.duckdb.org/v1.4.3/linux_amd64/sqlite_scanner.duckdb_extension.gz"
gunzip -f sqlite_scanner.duckdb_extension.gz
```

### Error: "503 Service Unavailable" on extension download

**Cause**: Transient error from DuckDB extension server.

**Fix**: The setup script uses `--retry 3`. If it still fails, wait a moment and retry:
```bash
./scripts/setup-duckdb-extensions.sh
```

### Error: "module lookup disabled by GOPROXY=off"

**Cause**: GOPROXY was set to off but modules aren't cached.

**Fix**: Unset GOPROXY to allow downloads:
```bash
unset GOPROXY
go mod download
```

### Frontend build fails with missing dependencies

**Fix**: Install npm dependencies first:
```bash
cd web
npm ci
npm run build
```

### Error: "NATS_ENABLED=true but NATS support not compiled"

**Cause**: Binary was built without `-tags "wal,nats"`.

**Fix**: Rebuild with build tags:
```bash
CGO_ENABLED=1 go build -tags "wal,nats" -v -o cartographus ./cmd/server
```

### Tests fail with "undefined" errors for NATS/WAL types

**Cause**: Tests run without `-tags "wal,nats"`.

**Fix**: Always include build tags:
```bash
go test -tags "wal,nats" -v -race ./...
```

### Error: "converting NULL to string is unsupported" in database tests

**Cause**: DuckDB JSON column handling issue. Some columns return NULL that must be handled with `sql.NullString`.

**Fix**: This is a code issue, not environment. Check that nullable JSON columns use `sql.NullString` instead of `string` in scan targets.

---

## Available Tools

The Claude Code Web environment includes:

| Tool | Version | Notes |
|------|---------|-------|
| Go | 1.24.7 | 1.25.5 downloadable with proxy override |
| GCC | 13.3.0 | Required for CGO |
| Clang | Available | Alternative C compiler |
| Node.js | 22.21.1 | LTS version |
| npm | 10.9.4 | Package manager |
| golangci-lint | 2.5.0 | Pre-installed |
| Git | 2.43.0 | Version control |
| Make | 4.3 | Build automation |
| Python | 3.x | With requests module |
| TypeScript | 5.9.3 | Via npm |

**NOT Available**: Docker, staticcheck (use golangci-lint), DNS tools (host, dig, nslookup)

---

## Offline Capabilities

After initial setup, these work without network access:

| Command | Network Required |
|---------|------------------|
| `go build ./...` | No (if modules cached) |
| `go test ./...` | No (if modules/extensions cached) |
| `go mod verify` | No |
| `gofmt -e ./...` | No |
| `go vet ./...` | No |
| `golangci-lint run ./...` | No |
| `npm run build` | No (after npm ci) |
| `npx tsc --noEmit` | No |

---

## Session Setup Script

For convenience, you can create a setup script:

```bash
#!/bin/bash
# session-setup.sh - Run at the start of each Claude Code Web session

set -e

echo "=== Claude Code Web Session Setup ==="

# Step 1: Environment variables (CRITICAL - prevents toolchain download errors)
export GOTOOLCHAIN=local
export no_proxy="localhost,127.0.0.1"
export NO_PROXY="localhost,127.0.0.1"
echo "Environment variables set."

# Step 2: Verify Go
echo "Go version: $(go version)"
echo "GOTOOLCHAIN: $(go env GOTOOLCHAIN)"

# Step 3: Setup DuckDB extensions (always run to ensure all extensions present)
echo "Setting up DuckDB extensions..."
./scripts/setup-duckdb-extensions.sh

# Step 4: Build with REQUIRED build tags
echo "Building project with -tags 'wal,nats'..."
CGO_ENABLED=1 go build -tags "wal,nats" -v -o cartographus ./cmd/server

# Step 5: Verify build
echo ""
echo "=== Setup Complete ==="
ls -lh cartographus
file cartographus
echo ""
echo "Ready to run: ./cartographus"
echo "Ready to test: go test -tags 'wal,nats' -v -race ./..."
```

**Key Points**:
1. Environment variables MUST be set before any Go commands
2. The `-tags "wal,nats"` flag is REQUIRED for all Go commands
3. DuckDB extensions should be installed before running database-dependent tests
4. Binary should be ~102MB when built correctly

---

## Running the Binary Locally

After building, you can run the server for live API testing:

### Start the Server

```bash
# Create data directory (required for DuckDB database)
mkdir -p /data

# Set minimal configuration for testing
export AUTH_MODE=none
export PORT=3857
export LOG_LEVEL=info
export LOG_FORMAT=console
export PLEX_ENABLED=false
export JELLYFIN_ENABLED=false
export EMBY_ENABLED=false
export TAUTULLI_ENABLED=false

# Run in background
nohup ./cartographus > /tmp/server.log 2>&1 &
sleep 5

# View startup logs
cat /tmp/server.log
```

### Verify Server is Running

```bash
# Health check
curl -s http://localhost:3857/api/v1/health | python3 -m json.tool

# Expected output:
# {
#     "status": "success",
#     "data": {
#         "status": "degraded",  # Expected without media servers
#         "database_connected": true,
#         ...
#     }
# }
```

### Test API Endpoints

```bash
# Core endpoints
curl -s http://localhost:3857/api/v1/stats
curl -s http://localhost:3857/api/v1/playbacks
curl -s http://localhost:3857/api/v1/users

# Analytics
curl -s http://localhost:3857/api/v1/analytics/trends
curl -s http://localhost:3857/api/v1/analytics/geographic
curl -s http://localhost:3857/api/v1/spatial/hexagons

# Export
curl -s http://localhost:3857/api/v1/export/geojson
curl -s http://localhost:3857/api/v1/export/playbacks/csv | head -5

# Swagger UI
curl -s -o /dev/null -w "HTTP %{http_code}\n" http://localhost:3857/swagger/index.html

# Prometheus metrics
curl -s http://localhost:3857/metrics | head -10
```

### Stop the Server

```bash
pkill -f cartographus
```

### Expected Startup Output

A successful startup shows:

```
INF Starting Cartographus with supervisor tree
INF Configuration loaded (standalone mode) auth_mode=none db_path=/data/cartographus.duckdb
WRN DataSketches extension unavailable, approximate analytics will use exact calculations
INF Database initialized successfully
INF HTTP server service added addr=0.0.0.0:3857
INF Starting supervisor tree...
```

**Notes**:
- **DataSketches warning is expected** (optional extension, graceful fallback to exact calculations)
- **RapidFuzz warning should NOT appear** if you ran `./scripts/setup-duckdb-extensions.sh` - this extension IS installed by the script
- No migration messages expected (schema is consolidated pre-release)
- Status "degraded" is expected when no media servers are configured

**If you see RapidFuzz warning**: Re-run the extension setup script:
```bash
./scripts/setup-duckdb-extensions.sh
```

---

## Version Compatibility

| Component | Current Version | Notes |
|-----------|-----------------|-------|
| Go (installed) | 1.24.7 | Works with project |
| Go (go.mod) | 1.24.0 | Minimum required |
| Go toolchain | 1.25.5 | Specified in go.mod, downloadable |
| DuckDB | v1.4.3 | Must match duckdb-go-bindings |
| duckdb-go/v2 | v2.5.4 | Go wrapper |
| duckdb-go-bindings | v0.1.24 | Platform-specific bindings |

**IMPORTANT**: DuckDB extension version (v1.4.3) MUST match the version used by duckdb-go-bindings. Using wrong versions will cause runtime errors.

---

## References

- [DIAGNOSTIC_REPORT.md](./working/DIAGNOSTIC_REPORT.md) - Full technical analysis of Claude Code Web environment
- [setup-duckdb-extensions.sh](../scripts/setup-duckdb-extensions.sh) - Extension download script
- [DEVELOPMENT.md](./DEVELOPMENT.md) - General development workflow
- [TROUBLESHOOTING.md](./TROUBLESHOOTING.md) - Common issues and solutions
