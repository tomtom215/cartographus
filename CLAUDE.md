# CLAUDE.md - AI Assistant Guide for Cartographus

This document provides guidance for AI assistants working on the Cartographus codebase.

**Last Updated**: 2026-01-11
**Project**: Cartographus - Media Server Analytics and Geographic Visualization Platform
**Repository**: https://github.com/tomtom215/cartographus

---

## !!! STOP - READ THIS FIRST !!!

```
╔══════════════════════════════════════════════════════════════════════════════╗
║                                                                              ║
║   MANDATORY: RUN THIS COMMAND AT THE START OF EVERY SESSION                  ║
║                                                                              ║
║   source scripts/session-setup.sh                                            ║
║                                                                              ║
║   WITHOUT THIS, ALL GO COMMANDS WILL FAIL WITH DNS/NETWORK ERRORS            ║
║                                                                              ║
╚══════════════════════════════════════════════════════════════════════════════╝
```

**Why?** Claude Code Web cannot reach external networks. The setup script configures:
- `GOTOOLCHAIN=local` - Prevents Go from trying to download toolchains
- `no_proxy/NO_PROXY` - Ensures localhost connections work
- `CGO_ENABLED=1` - Required for DuckDB

**Without setup, you will see:**
```
dial tcp: lookup storage.googleapis.com on [::1]:53: connection refused
```

---

## Session Setup Options

| Command | Use When |
|---------|----------|
| `source scripts/session-setup.sh` | Standard setup (env + extensions + build) |
| `source scripts/session-setup.sh --quick` | Repeat commands in same session (env only) |
| `source scripts/session-setup.sh --all` | First time setup (includes npm ci) |
| `source scripts/session-setup.sh --verify` | Check if setup is correct |

**IMPORTANT**: Always use `source` (not `./`) so environment variables persist.

### Manual Setup (if script unavailable)

```bash
export GOTOOLCHAIN=local
export CGO_ENABLED=1
export no_proxy="localhost,127.0.0.1"
export NO_PROXY="localhost,127.0.0.1"
```

### Build Tags (REQUIRED)

All Go commands MUST include `-tags "wal,nats"`:

```bash
go build -tags "wal,nats" -o cartographus ./cmd/server
go test -tags "wal,nats" -v -race ./...
go vet -tags "wal,nats" ./...
```

Without these tags, NATS and WAL features won't compile.

---

## Quick Reference

| Task | Command |
|------|---------|
| Build | `source scripts/session-setup.sh && go build -tags "wal,nats" -o cartographus ./cmd/server` |
| Test | `go test -tags "wal,nats" -v -race ./...` |
| Lint | `go vet -tags "wal,nats" ./...` |
| Frontend Build | `cd web && npm run build` |
| E2E Tests | `cd web && npm run test:e2e` |
| Route Count | `grep -c "r\.\(Get\|Post\|Put\|Delete\|Patch\)" internal/api/chi_router.go` (expect ~302) |

---

## Project Overview

Cartographus is a self-hosted media server analytics platform that visualizes playback activity on interactive maps.

### Key Numbers

| Metric | Value |
|--------|-------|
| API Endpoints | 302 |
| Go Test Files | 379 |
| E2E Test Suites | 75 (1300+ tests) |
| TypeScript Files | 229 |
| Internal Packages | 26 |
| ADRs | 29 |
| Detection Rules | 7 |

### Technology Stack

| Component | Technology | Version |
|-----------|------------|---------|
| Backend | Go | 1.24.0 |
| Router | Chi | 5.2.3 |
| Database | DuckDB | 1.4.3 (Go bindings 2.5.4) |
| WAL | BadgerDB | 4.9.0 |
| Logging | zerolog | 1.34.0 |
| Supervision | Suture | 4.0.6 |
| Validation | go-playground/validator | 10.30.1 |
| Frontend | TypeScript | 5.9.3 |
| Maps | MapLibre GL JS | 5.15.0 |
| Charts | ECharts | 6.0.0 |
| 3D Visualization | deck.gl | 9.2.5 |
| Tiles | PMTiles | 4.3.2 |
| Build | esbuild | 0.27.2 |

**Key Point**: DuckDB is NOT SQLite. Different syntax (PERCENTILE_CONT, spatial queries, etc.).

### DuckDB Limitations

- NO partial indexes (`CREATE INDEX ... WHERE ...`)
- NO `IDENTITY` with `PRIMARY KEY` - use manual ID generation
- Use `COALESCE(MAX(id), 0) + 1` pattern for auto-incrementing IDs

---

## Architecture

```
┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐
│   Tautulli   │  │    Plex      │  │   Jellyfin   │  │    Emby      │
│   API/DB     │  │  WebSocket   │  │  WebSocket   │  │  WebSocket   │
└──────┬───────┘  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘
       │                 │                 │                 │
       └─────────────────┴─────────────────┴─────────────────┘
                                   │
                                   ▼
                         ┌───────────────────┐
                         │   Sync Manager    │
                         │ (Cross-Source     │
                         │  Deduplication)   │
                         └─────────┬─────────┘
                                   │
                    ┌──────────────┼──────────────┐
                    ▼              ▼              ▼
            ┌────────────┐ ┌────────────┐ ┌────────────────┐
            │ BadgerDB   │ │   DuckDB   │ │   Detection    │
            │    WAL     │ │ + Spatial  │ │    Engine      │
            └────────────┘ │  + H3      │ └────────────────┘
                           └─────┬──────┘
                                 │
                    ┌────────────┼────────────┐
                    ▼            ▼            ▼
            ┌────────────┐ ┌──────────┐ ┌───────────┐
            │    NATS    │ │ REST API │ │ WebSocket │
            │ JetStream  │ │ (302     │ │    Hub    │
            └────────────┘ │ routes)  │ └───────────┘
                           └─────┬────┘
                                 ▼
                         ┌──────────────┐
                         │   Frontend   │
                         │ TypeScript   │
                         │ MapLibre GL  │
                         │ deck.gl      │
                         │ ECharts      │
                         └──────────────┘
```

### Supervisor Tree (Suture v4)

```
                    RootSupervisor
                   "cartographus"
                          │
        ┌─────────────────┼─────────────────┐
        │                 │                 │
        ▼                 ▼                 ▼
  DataSupervisor   MessagingSupervisor   APISupervisor
  "data-layer"     "messaging-layer"     "api-layer"
        │                 │                 │
        │           ┌─────┼─────┐           │
        ▼           ▼     ▼     ▼           ▼
   (Future)    WebSocket Sync  NATS      HTTP
               Hub     Mgr  Components  Server
```

---

## Directory Structure

```
map/
├── cmd/server/main.go           # Application entry point
├── internal/                    # Private application code (26 packages)
│   ├── api/                     # HTTP handlers (~51 source files)
│   │   ├── chi_router.go        # Chi router configuration (302 endpoints)
│   │   ├── chi_middleware.go    # Chi middleware (CORS, rate limiting)
│   │   ├── handlers_*.go        # Endpoint handlers
│   │   └── handlers_sync.go     # Data sync API handlers
│   ├── database/                # DuckDB abstraction (~62 source files)
│   │   ├── database.go          # Core DB lifecycle
│   │   ├── database_schema.go   # Complete schema (203 columns)
│   │   └── migrations.go        # Migration infrastructure
│   ├── supervisor/              # Suture v4 process supervision
│   ├── authz/                   # Zero Trust authorization (Casbin)
│   ├── eventprocessor/          # NATS/Watermill event processing
│   ├── detection/               # Security detection engine (5 rules)
│   ├── wal/                     # BadgerDB Write-Ahead Log
│   ├── backup/                  # Database backup/restore
│   ├── auth/                    # JWT/OIDC/Plex authentication
│   ├── config/                  # Configuration (Koanf v2)
│   ├── logging/                 # Structured logging (zerolog)
│   ├── sync/                    # Multi-server sync
│   ├── import/                  # Tautulli database import
│   └── websocket/               # Real-time WebSocket hub
├── web/                         # Frontend (~229 TypeScript files)
│   ├── src/index.ts             # Main entry
│   ├── src/app/                 # Application managers (~79 files)
│   ├── src/lib/api/             # API client modules
│   └── src/lib/types/           # TypeScript type definitions
├── tests/e2e/                   # Playwright E2E (75 suites)
├── docs/                        # Documentation
│   ├── adr/                     # 29 Architecture Decision Records
│   ├── API-REFERENCE.md         # Complete API documentation
│   └── design/                  # Design documents
├── scripts/
│   ├── session-setup.sh         # REQUIRED - Run at session start
│   ├── setup-duckdb-extensions.sh
│   └── sync-templates.sh
├── go.mod                       # Go 1.24.0 dependencies
└── Dockerfile                   # Multi-stage Docker build
```

---

## DuckDB Extensions

| Extension | Type | Purpose |
|-----------|------|---------|
| httpfs | Core | HTTPS downloads |
| spatial | Core | ST_* geometry functions |
| h3 | Community | Hexagonal geospatial indexing |
| inet | Core | IP address operations |
| icu | Core | Timezone-aware operations |
| json | Core | JSON parsing |
| sqlite_scanner | Core | Tautulli database import |
| rapidfuzz | Community | Fuzzy search |
| datasketches | Community | HLL/KLL approximate analytics (disabled by default) |

Enable datasketches: `DUCKDB_DATASKETCHES_ENABLED=true`

---

## AI Assistant Guidelines

### DO

- Run `source scripts/session-setup.sh` at session start
- Read files before modifying them
- Use `-tags "wal,nats"` for all Go commands
- Use parameterized queries (`?` placeholders)
- Add tests for new features
- Update CHANGELOG.md for new features
- Run pre-commit checks before committing

### DON'T

- Disable CGO (DuckDB requires it)
- Use SQLite syntax (this is DuckDB)
- Skip the session setup script
- Omit build tags from Go commands
- Ignore errors or skip tests
- Support ARMv7 (DuckDB limitation)

### TDD Requirements

All new code MUST have tests:

| Code Type | Coverage Target |
|-----------|-----------------|
| API Handlers | 90%+ |
| Database Methods | 90%+ |
| Authentication | 100% |

---

## Pre-Commit Checklist

```bash
# Option A: Automated (RECOMMENDED)
make pre-commit

# Option B: Manual
./scripts/sync-templates.sh --check     # Verify templates in sync
gofmt -s -w .                           # Format Go code
go mod tidy                             # Clean dependencies
go vet -tags "wal,nats" ./...           # Lint Go
go test -tags "wal,nats" -v -race ./... # Test
cd web && npm run build && cd ..        # Build frontend
grep -c "r\.\(Get\|Post\|Put\|Delete\|Patch\)" internal/api/chi_router.go  # Verify ~302
```

**Commit Types**: `feat`, `fix`, `docs`, `test`, `refactor`, `perf`, `chore`, `ci`

---

## Template Synchronization

| File | Purpose |
|------|---------|
| `web/public/index.html` | Development template (source of truth) |
| `internal/templates/index.html.tmpl` | Production template (Go server) |

```bash
./scripts/sync-templates.sh --check  # Verify in sync
./scripts/sync-templates.sh --sync   # Update production from dev
```

---

## Running the Server Locally

```bash
# Setup
source scripts/session-setup.sh
mkdir -p /data

# Build
go build -tags "wal,nats" -o cartographus ./cmd/server

# Run (minimal config)
AUTH_MODE=none PORT=3857 LOG_FORMAT=console ./cartographus

# Test endpoints
curl -s http://localhost:3857/api/v1/health | python3 -m json.tool
curl -s http://localhost:3857/api/v1/stats | python3 -m json.tool
```

---

## Documentation Links

| Document | Purpose |
|----------|---------|
| [docs/CLAUDE_CODE_WEB_SETUP.md](./docs/CLAUDE_CODE_WEB_SETUP.md) | Environment setup details |
| [docs/API-REFERENCE.md](./docs/API-REFERENCE.md) | Complete API documentation |
| [docs/adr/](./docs/adr/) | 29 Architecture Decision Records |
| [docs/DEVELOPMENT.md](./docs/DEVELOPMENT.md) | Development workflow |
| [docs/TROUBLESHOOTING.md](./docs/TROUBLESHOOTING.md) | Common issues |
| [docs/ARCHITECTURE.md](./docs/ARCHITECTURE.md) | System architecture |
| [CHANGELOG.md](./CHANGELOG.md) | Version history |

---

## Architecture Decision Records

| ADR | Decision |
|-----|----------|
| [0001](./docs/adr/0001-use-duckdb-for-analytics.md) | DuckDB for analytics |
| [0002](./docs/adr/0002-frontend-technology-stack.md) | TypeScript + MapLibre + ECharts + deck.gl |
| [0003](./docs/adr/0003-authentication-architecture.md) | Multi-mode auth |
| [0004](./docs/adr/0004-process-supervision-with-suture.md) | Suture v4 supervision |
| [0005](./docs/adr/0005-nats-jetstream-event-processing.md) | NATS JetStream |
| [0006](./docs/adr/0006-badgerdb-write-ahead-log.md) | BadgerDB WAL |
| [0007](./docs/adr/0007-event-sourcing-architecture.md) | Event sourcing |
| [0008](./docs/adr/0008-circuit-breaker-pattern.md) | Circuit breaker |
| [0009](./docs/adr/0009-plex-direct-integration.md) | Plex integration |
| [0010](./docs/adr/0010-cursor-based-pagination.md) | Cursor pagination |
| [0011](./docs/adr/0011-ci-cd-infrastructure.md) | Self-hosted CI/CD |
| [0012](./docs/adr/0012-configuration-management-koanf.md) | Koanf configuration |
| [0013](./docs/adr/0013-request-validation.md) | Request validation |
| [0014](./docs/adr/0014-tautulli-database-import.md) | Tautulli import |
| [0015](./docs/adr/0015-zero-trust-authentication-authorization.md) | Zero Trust auth |
| [0016](./docs/adr/0016-chi-router-adoption.md) | Chi router |
| [0017](./docs/adr/0017-watermill-router-and-middleware.md) | Watermill middleware |
| [0018](./docs/adr/0018-duckdb-community-extensions.md) | Community extensions |
| [0019](./docs/adr/0019-testcontainers-integration-testing.md) | Testcontainers |
| [0020](./docs/adr/0020-detection-rules-engine.md) | Detection rules |
| [0021](./docs/adr/0021-go-json-high-performance-json.md) | High-performance JSON |
| [0022](./docs/adr/0022-dedupe-audit-management.md) | Dedupe audit |
| [0023](./docs/adr/0023-consumer-wal-exactly-once.md) | Exactly-once delivery |
| [0024](./docs/adr/0024-recommendation-engine.md) | Recommendation engine |
| [0025](./docs/adr/0025-deterministic-e2e-mocking.md) | E2E mocking |
| [0026](./docs/adr/0026-multi-server-management-ui.md) | Multi-server UI |
| [0027](./docs/adr/0027-websocket-real-time-hub.md) | WebSocket real-time hub |
| [0028](./docs/adr/0028-jellyfin-emby-integration.md) | Jellyfin/Emby integration |
| [0029](./docs/adr/0029-backup-restore-gfs-retention.md) | Backup/Restore with GFS |

---

## Project-Specific Notes

- **Port 3857**: EPSG:3857 (Web Mercator projection)
- **Cache TTL**: 5 minutes for analytics
- **WebSocket ping**: 30 seconds
- **Prometheus metrics**: `/metrics` endpoint
- **ARMv7**: Not supported (DuckDB limitation)
- **Detection rules**: impossible_travel, concurrent_streams, device_velocity, geo_restriction, simultaneous_locations, user_agent_anomaly, vpn_usage
- **Password policy**: NIST SP 800-63B (12 char min, complexity)
- **Rate limiting**: auth 5/min, analytics 1000/min, default 100/min
- **Logging**: LOG_LEVEL (trace/debug/info/warn/error), LOG_FORMAT (json/console), LOG_CALLER (true/false)

---

## Troubleshooting

| Error | Cause | Solution |
|-------|-------|----------|
| `dial tcp: lookup storage.googleapis.com...` | Missing env setup | `source scripts/session-setup.sh` |
| `failed to install sqlite_scanner` | Missing extensions | `./scripts/setup-duckdb-extensions.sh` |
| `undefined: duckdb` | CGO disabled | Use `CGO_ENABLED=1` |
| `NATS support not compiled` | Missing tags | Use `-tags "wal,nats"` |
| Templates out of sync | E2E failures | `./scripts/sync-templates.sh --sync` |

---

## Getting Help

- **README.md**: User documentation
- **docs/ARCHITECTURE.md**: System design
- **docs/adr/**: Decision rationale
- **docs/TROUBLESHOOTING.md**: Common issues
- **GitHub Issues**: https://github.com/tomtom215/cartographus/issues
