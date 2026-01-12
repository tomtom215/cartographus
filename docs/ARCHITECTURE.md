# Architecture Design Document

**Last Updated**: 2026-01-11

This document describes the current production architecture of Cartographus. For detailed rationale behind architectural decisions, see the [Architecture Decision Records](./adr/) (29 ADRs).

---

## System Overview

Cartographus is a **standalone** self-hosted media server analytics platform that connects directly to Plex, Jellyfin, and Emby. It aggregates playback data and visualizes it on interactive maps with real-time security detection. Tautulli integration is optional and only needed for historical data migration.

### High-Level Architecture

```
┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐
│    Plex      │  │   Jellyfin   │  │    Emby      │  │  Tautulli    │
│  WebSocket   │  │  WebSocket   │  │  WebSocket   │  │ (Optional)   │
└──────┬───────┘  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘
       │                 │                 │                 │
       └─────────────────┴─────────────────┴─────────────────┘
                                   │
                                   ▼
                    ┌─────────────────────────────┐
                    │       Sync Manager          │
                    │  • Cross-source dedup       │
                    │  • User ID mapping          │
                    │  • CorrelationKey v2.0      │
                    │  • Circuit breaker          │
                    └──────────────┬──────────────┘
                                   │
              ┌────────────────────┼────────────────────┐
              ▼                    ▼                    ▼
    ┌────────────────┐    ┌────────────────┐    ┌────────────────┐
    │  BadgerDB WAL  │    │    DuckDB      │    │   Detection    │
    │  (Durability)  │    │  Analytics DB  │    │    Engine      │
    └────────────────┘    │  • Spatial     │    │  • 5 rules     │
                          │  • H3 hexagons │    │  • Trust scores│
                          │  • RapidFuzz   │    │  • Webhooks    │
                          └───────┬────────┘    └────────────────┘
                                  │
              ┌───────────────────┼───────────────────┐
              ▼                   ▼                   ▼
    ┌────────────────┐    ┌────────────────┐    ┌────────────────┐
    │ NATS JetStream │    │   Chi Router   │    │  WebSocket Hub │
    │ Event Sourcing │    │  302 endpoints │    │  Real-time     │
    └────────────────┘    └───────┬────────┘    └────────────────┘
                                  │
                                  ▼
                    ┌─────────────────────────────┐
                    │         Frontend            │
                    │  • TypeScript 5.9           │
                    │  • MapLibre GL JS 5.15      │
                    │  • deck.gl 9.2              │
                    │  • ECharts 6.0              │
                    └─────────────────────────────┘
```

---

## Technology Stack

| Layer | Technology | Version | ADR |
|-------|------------|---------|-----|
| **Database** | DuckDB | 1.4.3 | [ADR-0001](adr/0001-use-duckdb-for-analytics.md) |
| **Frontend** | TypeScript + MapLibre + ECharts | 5.9.3, 5.15.0, 6.0.0 | [ADR-0002](adr/0002-frontend-technology-stack.md) |
| **Authentication** | Zitadel OIDC + JWT + Casbin (Zero Trust) | v3.45.1 | [ADR-0003](adr/0003-authentication-architecture.md), [ADR-0015](adr/0015-zero-trust-authentication-authorization.md) |
| **Process Supervision** | Suture v4 | 4.0.6 | [ADR-0004](adr/0004-process-supervision-with-suture.md) |
| **Event Processing** | NATS JetStream + Watermill | - | [ADR-0005](adr/0005-nats-jetstream-event-processing.md), [ADR-0017](adr/0017-watermill-router-and-middleware.md) |
| **Write-Ahead Log** | BadgerDB | 4.9.0 | [ADR-0006](adr/0006-badgerdb-write-ahead-log.md) |
| **Event Sourcing** | NATS-First Architecture | - | [ADR-0007](adr/0007-event-sourcing-architecture.md) |
| **HTTP Router** | Chi | 5.2.3 | [ADR-0016](adr/0016-chi-router-adoption.md) |
| **Configuration** | Koanf v2 | - | [ADR-0012](adr/0012-configuration-management-koanf.md) |
| **Validation** | go-playground/validator | 10.30.1 | [ADR-0013](adr/0013-request-validation.md) |
| **Detection** | Custom Rules Engine | - | [ADR-0020](adr/0020-detection-rules-engine.md) |

### DuckDB Extensions

| Extension | Type | Purpose |
|-----------|------|---------|
| spatial | Core | GEOMETRY types, ST_* functions, R-tree indexes |
| h3 | Community | Hexagonal hierarchical geospatial indexing |
| inet | Core | IP address handling |
| icu | Core | Internationalization, timezone support |
| json | Core | JSON parsing and querying |
| sqlite_scanner | Core | Tautulli database import |
| rapidfuzz | Community | Fuzzy string matching ([ADR-0018](adr/0018-duckdb-community-extensions.md)) |
| datasketches | Community | Approximate analytics (HyperLogLog, KLL) |

---

## Architecture Layers

### Layer 1: Data Ingestion

Three primary data sources connect directly to the Sync Manager, with optional Tautulli support:

| Source | Client | Capabilities |
|--------|--------|--------------|
| Plex | `internal/sync/plex.go` | WebSocket real-time, webhooks, transcode monitoring, sessions |
| Jellyfin | `internal/sync/jellyfin_client.go` | WebSocket real-time, session polling, user mapping |
| Emby | `internal/sync/emby_client.go` | WebSocket real-time, session polling, user mapping |
| Tautulli (Optional) | `internal/sync/tautulli_client.go` | Historical data import, enhanced Plex metadata |

**Cross-Source Deduplication** (Event Sourcing Mode):
```
CorrelationKey v2.0 = {source}:{server_id}:{user_id}:{rating_key}:{machine_id}:{5min_time_bucket}
```

The v2.0 correlation key includes source and server_id to prevent cross-platform conflicts when users watch the same content on multiple platforms.

Deduplication layers:
1. **NATS MsgId**: Prevents redelivery of same message
2. **Consumer Cache**: 5-minute in-memory deduplication window
3. **Cross-Source Key**: `xsrc:{user_id}:{rating_key}:...` for detecting same content across platforms
4. **DuckDB Constraint**: Unique index on correlation_key

### Layer 2: Event Processing

**NATS JetStream Configuration:**
- Stream: `MEDIA_EVENTS`
- Subject: `playback.events.*`
- Retention: 7 days
- Storage: File-based

**BadgerDB WAL for Durability:**
- Write-before-publish pattern
- Automatic retry on NATS failures
- Crash recovery on startup

See [ADR-0005](adr/0005-nats-jetstream-event-processing.md) and [ADR-0006](adr/0006-badgerdb-write-ahead-log.md).

### Layer 3: Analytics Database

**DuckDB Schema Highlights:**

```sql
-- Main table: 117+ columns for playback events
CREATE TABLE playback_events (
    id UUID PRIMARY KEY,
    session_key TEXT NOT NULL,
    started_at TIMESTAMP NOT NULL,
    user_id INTEGER NOT NULL,
    ip_address TEXT NOT NULL,
    correlation_key TEXT UNIQUE,
    -- ... media metadata, streaming quality, technical details
    watched_at TIMESTAMP GENERATED ALWAYS AS (COALESCE(stopped_at, started_at)) STORED
);

-- Geographic data with H3 hexagonal indexes
CREATE TABLE geolocations (
    ip_address TEXT PRIMARY KEY,
    geom GEOMETRY NOT NULL,
    h3_index_6 UBIGINT,  -- ~36 km^2 hexagons
    h3_index_7 UBIGINT,  -- ~5 km^2 hexagons
    h3_index_8 UBIGINT,  -- ~0.74 km^2 hexagons
    -- ... location details
);

-- Spatial index for geographic queries
CREATE SPATIAL INDEX idx_geolocation_geom ON geolocations USING RTREE (geom);

-- Cross-platform content mapping (Phase 3)
CREATE TABLE content_mappings (
    id BIGINT PRIMARY KEY,
    imdb_id TEXT,           -- IMDb ID (tt1234567)
    tmdb_id INTEGER,        -- TMDB movie/show ID
    tvdb_id INTEGER,        -- TVDB series ID
    plex_rating_key TEXT,   -- Plex rating_key
    jellyfin_item_id TEXT,  -- Jellyfin Item.Id (UUID)
    emby_item_id TEXT,      -- Emby Item.Id
    title TEXT NOT NULL,
    media_type TEXT NOT NULL,  -- movie, show, episode
    -- ... indexes on external IDs for lookup
);

-- Cross-platform user linking (Phase 3)
CREATE TABLE user_links (
    id BIGINT PRIMARY KEY,
    primary_user_id INTEGER NOT NULL,
    linked_user_id INTEGER NOT NULL,
    link_type TEXT NOT NULL,  -- manual, email, plex_home
    confidence DOUBLE DEFAULT 1.0,
    UNIQUE(primary_user_id, linked_user_id)
);
```

**Performance Targets:**
- API Response: <30ms p95 (achieved)
- Query Parallelization: `runtime.NumCPU()` threads
- Cache TTL: 5 minutes for analytics

### Layer 4: Detection Engine

Five detection rules evaluate playback events in real-time:

| Rule | Detection Logic |
|------|-----------------|
| Impossible Travel | Haversine distance / time > 900 km/h |
| Concurrent Streams | Active sessions per user > limit |
| Device Velocity | Same device on >3 IPs within 5 minutes |
| Geo Restriction | Country in blocklist (or not in allowlist) |
| Simultaneous Locations | Active streams >100 km apart |

**Alert Flow:**
```
Event → Detection Engine → Alert Store (DuckDB)
                        → WebSocket Broadcast
                        → Discord/Webhook Notifiers
```

**Trust Scoring:**
- Default: 100
- Violation penalty: -10 (configurable)
- Daily recovery: +1
- Auto-restriction threshold: 50

See [ADR-0020](adr/0020-detection-rules-engine.md).

### Layer 5: API Service

**Chi Router Configuration** (302 endpoints):

| Group | Routes | Path Prefix |
|-------|--------|-------------|
| Health | 5 | `/api/v1/health` |
| Auth | 6 | `/api/v1/auth` |
| Core | 8 | `/api/v1` |
| Analytics | 28 | `/api/v1/analytics` |
| Spatial | 5 | `/api/v1/spatial` |
| Search | 2 | `/api/v1/search` |
| Plex | 18 | `/api/v1/plex` |
| Tautulli | 75 | `/api/v1/tautulli` |
| Export | 4 | `/api/v1/export` |
| Backup | 16 | `/api/v1/backup` |
| Detection | 11 | `/api/v1/detection` |
| Zero Trust | 12 | `/api/auth`, `/api/admin` |

**Middleware Stack:**
- CORS (configurable origins)
- Rate limiting (endpoint-specific: auth 5/min, analytics 1000/min, default 100/min)
- JWT authentication
- Request validation (go-playground/validator)
- Performance monitoring

### Layer 6: Process Supervision

**Suture v4 Supervisor Tree:**

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
    (Future)   WebSocket Sync  NATS      HTTP
               Hub     Mgr  Components  Server
```

**Service Restart Policy:**
- Failure threshold: 5 failures
- Failure decay: 30 seconds
- Backoff: 15 seconds initial, exponential increase
- Shutdown timeout: 10 seconds

See [ADR-0004](adr/0004-process-supervision-with-suture.md).

### Layer 7: Frontend

**Component Architecture:**

```
App
├── MapView (MapLibre GL JS 5.15)
│   ├── LocationLayer (GeoJSON clustering)
│   ├── ClusterManager
│   └── PopupManager
├── GlobeView (deck.gl 9.2)
│   ├── ScatterplotLayer
│   ├── H3HexagonLayer
│   └── ArcLayer (user-server connections)
├── AnalyticsDashboard (ECharts 6.0)
│   └── 47+ charts across 6 pages
├── FilterPanel
│   └── 14+ filter dimensions
├── DetectionAlertsPanel
│   ├── AlertsList
│   └── TrustScoreDisplay
└── LiveActivityDashboard
    ├── SessionCards
    └── TranscodeMonitor
```

**Performance Optimizations:**
- GeoJSON clustering (10x marker reduction)
- Debounced filter updates (300ms)
- Lazy chart loading (render on scroll)
- Parallel analytics API calls

---

## Security Architecture

### Authentication Modes

| Mode | Use Case | Implementation |
|------|----------|----------------|
| JWT (default) | Production | HTTP-only cookies, 24h sessions |
| Basic | Simple setups | bcrypt passwords (HTTPS required) |
| None | Development | No authentication |

### Zero Trust Authorization (Optional)

- OpenID Connect via Zitadel OIDC v3.45.1 (OpenID Foundation certified)
- PKCE (RFC 7636) and nonce validation for security
- Back-channel logout support (OIDC Back-Channel Logout 1.0)
- BadgerDB-backed durable state storage (ACID-compliant)
- Prometheus metrics for OIDC observability (15+ metrics)
- Audit logging for all authentication events
- Casbin RBAC policy enforcement
- Session management with revocation

See [ADR-0015](adr/0015-zero-trust-authentication-authorization.md).

### Security Headers

- Content-Security-Policy (no `unsafe-inline`)
- X-Frame-Options: DENY
- Strict-Transport-Security
- X-Content-Type-Options: nosniff

---

## Deployment Architecture

### Docker (Recommended)

```yaml
services:
  cartographus:
    image: ghcr.io/tomtom215/cartographus:latest
    ports:
      - "3857:3857"
    volumes:
      - ./data:/data
    environment:
      # Configure at least one media server
      - ENABLE_PLEX_SYNC=true
      - PLEX_URL=http://plex:32400
      - PLEX_TOKEN=${PLEX_TOKEN}
      # Or use Jellyfin/Emby - see README.md for examples
```

**Supported Platforms:**
- `linux/amd64`
- `linux/arm64`

**Not Supported:**
- `linux/arm/v7` (DuckDB limitation)

### Build Configuration

```bash
# Standard build
go build -o cartographus ./cmd/server

# With NATS + WAL support (recommended)
go build -tags "wal,nats" -o cartographus ./cmd/server
```

---

## Configuration

Configuration is loaded via Koanf v2 in order:
1. Built-in defaults
2. Config file (`config.yaml`)
3. Environment variables (highest priority)

Key environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `ENABLE_PLEX_SYNC` | `false` | Enable Plex direct integration |
| `PLEX_URL` | - | Plex server URL |
| `PLEX_TOKEN` | - | Plex authentication token |
| `JELLYFIN_ENABLED` | `false` | Enable Jellyfin integration |
| `EMBY_ENABLED` | `false` | Enable Emby integration |
| `TAUTULLI_ENABLED` | `false` | Enable Tautulli (optional, for historical data) |
| `DUCKDB_PATH` | `/data/cartographus.duckdb` | Database file |
| `DUCKDB_MAX_MEMORY` | `2GB` | Max query memory |
| `NATS_ENABLED` | `true` | Enable event processing |
| `WAL_ENABLED` | `true` | Enable durability layer |
| `DETECTION_ENABLED` | `true` | Enable security detection |

See [.env.example](.env.example) for complete reference.

---

## Monitoring

### Health Endpoints

| Endpoint | Purpose |
|----------|---------|
| `/api/v1/health` | Basic liveness check |
| `/api/v1/health/ready` | Readiness (DB + dependencies) |
| `/api/v1/health/nats` | NATS JetStream status |

### Metrics

- Prometheus endpoint: `/metrics`
- Request latency percentiles (p50, p95, p99)
- Detection engine metrics
- Cache hit rates

---

## Testing

| Type | Count | Location |
|------|-------|----------|
| Unit Tests | 8861 | `*_test.go` files |
| Fuzz Tests | 2 | `*_fuzz_test.go` files |
| E2E Tests | 1300+ | `tests/e2e/*.spec.ts` |
| Integration | Various | `-tags integration` |

**Coverage Targets:**
- API handlers: 90%+
- Database: 90%+
- Authentication: 100%
- Overall: 75.5%

---

## Related Documentation

| Document | Purpose |
|----------|---------|
| [README.md](README.md) | User documentation and quick start |
| [CLAUDE.md](CLAUDE.md) | AI assistant development guide |
| [adr/](adr/) | Architecture Decision Records (29 ADRs) |
| [docs/API-REFERENCE.md](docs/API-REFERENCE.md) | Complete API documentation |
| [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md) | Development workflow |
| [docs/TROUBLESHOOTING.md](docs/TROUBLESHOOTING.md) | Common issues |
| [CHANGELOG.md](CHANGELOG.md) | Version history |
