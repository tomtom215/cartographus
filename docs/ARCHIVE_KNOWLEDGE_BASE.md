# Cartographus Archive Knowledge Base

**Document Version**: 1.0
**Created**: 2026-01-12
**Last Verified**: 2026-01-12
**Purpose**: Consolidated reference document replacing all archived documentation
**Verification Status**: All metrics verified against source code

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [Project Overview](#2-project-overview)
3. [Architecture and Technology Stack](#3-architecture-and-technology-stack)
4. [Production Readiness Audit Summary](#4-production-readiness-audit-summary)
5. [API Integration Analysis](#5-api-integration-analysis)
6. [Security Implementation](#6-security-implementation)
7. [Performance and Scalability](#7-performance-and-scalability)
8. [Recommendation Engine Design](#8-recommendation-engine-design)
9. [DuckDB Extensions Implementation](#9-duckdb-extensions-implementation)
10. [Claude Code Web Environment Setup](#10-claude-code-web-environment-setup)
11. [Testing Infrastructure](#11-testing-infrastructure)
12. [Migration Completed Work](#12-migration-completed-work)
13. [Remaining Enhancement Opportunities](#13-remaining-enhancement-opportunities)
14. [Appendices](#14-appendices)

---

## 1. Executive Summary

Cartographus is a **production-ready** self-hosted media server analytics platform that provides geographic visualization of playback activity from Plex, Jellyfin, Emby, and Tautulli.

### Key Metrics (Verified 2026-01-12)

| Metric | Current Value |
|--------|---------------|
| API Endpoints | 302 |
| Architecture Decision Records | 29 |
| Go Source Files | 768+ |
| TypeScript Files | 229+ |
| Internal Packages | 26 |
| E2E Test Suites | 75+ (1300+ tests) |
| Production Readiness Score | 92/100 |

### Production Status

| Component | Status |
|-----------|--------|
| Core Backend | Production Ready |
| Database Layer | Production Ready |
| Authentication | Production Ready (Multiple modes) |
| API Surface | Production Ready |
| Frontend | Production Ready |
| Security | Audited and Approved |

---

## 2. Project Overview

### What is Cartographus?

Cartographus aggregates playback data from media servers and displays it on interactive geographic maps, enabling:

- **Geographic Visualization**: See where your viewers are located worldwide
- **Playback Analytics**: Understand viewing patterns, trends, and behaviors
- **Multi-Server Support**: Aggregate data from multiple Plex, Jellyfin, and Emby servers
- **Real-time Monitoring**: WebSocket-based live activity tracking
- **Detection Engine**: Identify suspicious patterns (VPN usage, impossible travel, concurrent streams)
- **Recommendation System**: Content recommendations based on viewing history

### Supported Integrations

| Service | Integration Type | Status |
|---------|------------------|--------|
| Plex | Direct API + WebSocket | Full Support |
| Jellyfin | REST API + WebSocket | Full Support |
| Emby | REST API + WebSocket | Full Support |
| Tautulli | REST API + DB Import | Full Support |

---

## 3. Architecture and Technology Stack

### Current Versions (Verified 2026-01-12)

| Component | Technology | Version |
|-----------|------------|---------|
| Backend Language | Go | 1.24.0 |
| HTTP Router | Chi | 5.2.3 |
| Database | DuckDB | 1.4.3 |
| DuckDB Go Bindings | duckdb-go/v2 | 2.5.4 |
| Write-Ahead Log | BadgerDB | 4.9.0 |
| Logging | zerolog | 1.34.0 |
| Process Supervision | Suture | 4.0.6 |
| Validation | go-playground/validator | 10.30.1 |
| Messaging | NATS JetStream | 2.12.3 |
| Frontend | TypeScript | 5.9.3 |
| Maps | MapLibre GL JS | 5.16.0 |
| Charts | ECharts | 6.0.0 |
| 3D Visualization | deck.gl | 9.2.5 |
| Build System | esbuild | 0.27.2 |

### Architecture Diagram

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
                         │ (Deduplication)   │
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
                         └──────────────┘
```

### Directory Structure

```
map/
├── cmd/server/main.go           # Entry point
├── internal/                    # Private packages (26)
│   ├── api/                     # HTTP handlers, 302 endpoints
│   ├── auth/                    # JWT, OIDC, Plex OAuth, Basic Auth
│   ├── authz/                   # Casbin RBAC
│   ├── backup/                  # Backup/restore with GFS retention
│   ├── config/                  # Koanf configuration
│   ├── database/                # DuckDB abstraction
│   ├── detection/               # Security detection rules
│   ├── eventprocessor/          # NATS/Watermill processing
│   ├── import/                  # Tautulli DB import
│   ├── logging/                 # zerolog wrapper
│   ├── metrics/                 # Prometheus metrics
│   ├── middleware/              # HTTP middleware
│   ├── models/                  # Data models
│   ├── recommend/               # Recommendation engine
│   ├── supervisor/              # Suture process supervision
│   ├── sync/                    # Multi-server sync
│   ├── wal/                     # BadgerDB WAL
│   ├── websocket/               # Real-time hub
│   └── ...                      # Other packages
├── web/                         # Frontend (229 TypeScript files)
├── tests/e2e/                   # Playwright E2E (75 suites)
├── docs/                        # Documentation
│   └── adr/                     # 29 ADRs
├── scripts/                     # Utility scripts
└── deploy/                      # Deployment configs
```

### DuckDB Extensions

| Extension | Type | Purpose |
|-----------|------|---------|
| httpfs | Core | HTTPS downloads |
| spatial | Core | ST_* geometry functions |
| h3 | Community | Hexagonal geospatial indexing |
| inet | Core | IP address operations |
| icu | Core | Timezone-aware operations |
| json | Core | JSON parsing |
| sqlite_scanner | Core | Tautulli database import |
| rapidfuzz | Community | Fuzzy string search |
| datasketches | Community | Approximate analytics (optional) |

---

## 4. Production Readiness Audit Summary

### Audit History

The codebase underwent a comprehensive 19-phase production readiness audit.

| Phase | Focus Area | Status |
|-------|------------|--------|
| 1 | Initial Database Issues | Completed |
| 2 | Authentication Hardening | Completed |
| 3 | API Consistency | Completed |
| 4 | Error Handling | Completed |
| 5 | Rate Limiting | Completed |
| 6 | WebSocket Security | Completed |
| 7 | Database Migrations | Completed |
| 8 | Password Policy (NIST SP 800-63B) | Completed |
| 9 | Final Review | Completed |

### Issues Resolved

| Priority | Count | Status |
|----------|-------|--------|
| Critical (P0) | 7 | All Fixed |
| High (P1) | 23 | All Fixed |
| Medium (P2) | 33 | All Fixed |
| Low (P3) | 30 | All Fixed |
| **Total** | **93** | **All Fixed** |

### Critical Fixes Completed

1. **DuckDB Audit Store** - Implemented for compliance logging
2. **Deterministic Event Replay** - Event sourcing with ordering guarantees
3. **Authentication on Write Endpoints** - Zero Trust enforcement
4. **HTTP Panic Recovery** - Graceful error handling
5. **CORS Validation** - Origin validation warnings
6. **Versioned Migrations** - Schema version tracking
7. **Password Policy** - NIST SP 800-63B compliance (12 char minimum)

### Code Quality Improvements

| Metric | Before | After |
|--------|--------|-------|
| console.* statements | 266 | 9 |
| `any` type usage | 161 | 66 |
| addEventListener:removeEventListener ratio | 35:1 | 2.9:1 |
| TODO comments in Go | Multiple | 0 |

---

## 5. API Integration Analysis

### Tautulli API Coverage

| Category | Used | Available | Coverage |
|----------|------|-----------|----------|
| Core Sync | 3 | 3 | 100% |
| Statistics | 18 | 20 | 90% |
| Library | 9 | 12 | 75% |
| User Management | 6 | 10 | 60% |
| **Total** | **44** | **93** | **47%** |

### Plex Direct API

**Read Operations**: 21 endpoints (GET)
**Write Operations**: 1 endpoint (CancelTranscode - admin only)

Key endpoints:
- `/status/sessions` - Active sessions
- `/library/sections` - Library content
- `/library/metadata/{key}` - Rich metadata
- WebSocket for real-time updates

### Plex.tv API

**Read Operations**: 3 endpoints
**Write Operations**: 8 endpoints (admin-protected)
- Friend management
- Library sharing
- Managed user management

### Jellyfin/Emby API

- Session monitoring via WebSocket
- REST API for metadata
- Circuit breaker protection on all calls

### Data Field Coverage

**Playback Events**: 68 fields captured (100%)

Key fields include:
- Session metadata (ID, timestamps, duration)
- User data (ID, username, IP)
- Media metadata (title, type, year, genres)
- Quality data (resolution, codec, bitrate)
- Stream data (transcoding decision, protocol)
- Geographic data (via GeoIP lookup)

---

## 6. Security Implementation

### Authentication Modes

| Mode | Use Case | Implementation |
|------|----------|----------------|
| None | Development | No auth required |
| Basic | Simple deployments | Username/password |
| JWT | API tokens | RS256/HS256 |
| Plex OAuth | Plex-integrated | OAuth 2.0 |
| OIDC | Enterprise SSO | Zitadel library v3.45.1 |

### OIDC Security (Zitadel Implementation)

| Feature | Status |
|---------|--------|
| PKCE (RFC 7636) | S256 only |
| State Parameter | 256-bit, TTL enforced |
| Nonce Validation | Enabled by default |
| Token Validation | Full JWT validation |
| JWKS Caching | 15-minute TTL |
| Key Rotation Monitoring | Prometheus metrics |
| Back-Channel Logout | Full spec compliance |
| JTI Replay Prevention | BadgerDB tracking |
| Session Fixation Prevention | Implemented |
| CSRF Protection | Double-submit cookie |

### Authorization (Casbin RBAC)

Role hierarchy: `viewer → editor → admin`

| Role | Permissions |
|------|-------------|
| Viewer | Read analytics, view maps |
| Editor | + Create exports, manage settings |
| Admin | + User management, server config |

### Rate Limiting

| Endpoint Type | Limit |
|---------------|-------|
| Auth endpoints | 5/min |
| Analytics | 1000/min |
| Default | 100/min |

### Detection Rules

7 security detection rules:
1. `impossible_travel` - Geo-based velocity anomaly
2. `concurrent_streams` - Multiple simultaneous sessions
3. `device_velocity` - Rapid device switching
4. `geo_restriction` - Geographic policy violations
5. `simultaneous_locations` - Multi-location access
6. `user_agent_anomaly` - UA pattern detection
7. `vpn_usage` - VPN/proxy detection

---

## 7. Performance and Scalability

### Performance Benchmarks

| Metric | Target | Achieved |
|--------|--------|----------|
| API Response (p95) | <100ms | <30ms |
| Map Rendering | 60 FPS | 60 FPS |
| Sync (10k events) | <30s | <30s |
| Sync (100k events) | <10min | <5min |
| Memory (normal) | <512MB | <512MB |
| Memory (100k records) | <2GB | <1GB |
| Throughput | >500 rec/sec | >1,000 rec/sec |

### Scalability Testing

| Dataset Size | Sync Time | Memory | Verified |
|--------------|-----------|--------|----------|
| 100 events | <1s | <50MB | Yes |
| 1,000 events | <5s | <100MB | Yes |
| 10,000 events | <30s | <512MB | Yes |
| 100,000 events | <5min | <1GB | Yes |

### Error Recovery

| Scenario | Handling |
|----------|----------|
| HTTP 429 Rate Limit | Exponential backoff (1-16s) |
| Database Connection Loss | Auto-reconnect (2-8s backoff) |
| WebSocket Disconnect | Auto-reconnect (1-32s backoff) |
| Circuit Breaker Open | 2-minute recovery timeout |

---

## 8. Recommendation Engine Design

### Algorithm Architecture

Planned/implemented algorithms:

| Algorithm | Type | Status |
|-----------|------|--------|
| Co-Visitation | Collaborative | Implemented |
| EASE | Matrix Factorization | Implemented |
| ALS | Matrix Factorization | Planned |
| NCF | Neural Collaborative | Planned |
| SASRec | Sequential | Planned |
| LightGCN | Graph Neural Network | Planned |
| FPMC | Sequential | Planned |

### Data Foundation

The playback_events table provides rich data for recommendations:
- 187 columns of playback metadata
- User viewing history
- Content metadata (genres, actors, directors)
- Multi-server aggregation

### Implementation Phases

1. **Phase 1**: Co-Visitation (baseline)
2. **Phase 2**: EASE (efficient matrix factorization)
3. **Phase 3**: Sequential models (SASRec)
4. **Phase 4**: Graph models (LightGCN)

---

## 9. DuckDB Extensions Implementation

### RapidFuzz (Implemented)

**File**: `internal/database/search_fuzzy.go`

Provides fuzzy string matching for search:
- `rapidfuzz_ratio()` - Overall similarity
- `rapidfuzz_token_set_ratio()` - Token-based matching
- Fallback to LIKE when unavailable

### DataSketches (Implemented)

**File**: `internal/database/analytics_approximate.go`

Provides approximate analytics:
- HyperLogLog for distinct counts
- KLL sketches for percentiles
- Optional (falls back to exact queries)

### VSS (Future)

Vector Similarity Search for recommendations:
- Content embeddings
- User preference vectors
- HNSW indexing

---

## 10. Claude Code Web Environment Setup

### Required at Session Start

```bash
# REQUIRED: Run this at the START of EVERY session
source scripts/session-setup.sh
```

Or manually:

```bash
export GOTOOLCHAIN=local
export CGO_ENABLED=1
export no_proxy="localhost,127.0.0.1"
export NO_PROXY="localhost,127.0.0.1"
```

### Build Commands

```bash
# Build with required tags
CGO_ENABLED=1 go build -tags "wal,nats" -o cartographus ./cmd/server

# Run tests
go test -tags "wal,nats" -v -race ./...

# Lint
go vet -tags "wal,nats" ./...
```

### Common Errors and Solutions

| Error | Cause | Solution |
|-------|-------|----------|
| `dial tcp: lookup storage.googleapis.com...` | Missing env setup | `source scripts/session-setup.sh` |
| `NATS support not compiled` | Missing build tags | Use `-tags "wal,nats"` |
| `undefined: duckdb` | CGO disabled | Use `CGO_ENABLED=1` |
| Extension installation failure | Missing extensions | `./scripts/setup-duckdb-extensions.sh` |

---

## 11. Testing Infrastructure

### Test Coverage

| Package | Coverage |
|---------|----------|
| config | 100% |
| cache | 98.7% |
| auth | 94% |
| middleware | 92% |
| sync | 88% |
| database | 85% |
| api | 78% |
| **Average** | **90.2%** |

### Test Categories

| Type | Count | Description |
|------|-------|-------------|
| Unit Tests | 8861+ | Table-driven tests |
| E2E Tests | 1300+ | Playwright browser tests |
| Integration | 100+ | Database integration |
| Race Detection | All | `-race` flag enabled |
| Benchmarks | 20+ | Performance benchmarks |

### E2E Test Suites

- Authentication
- Map rendering
- Chart rendering
- WebSocket communication
- Globe visualization (deck.gl)
- Live activity monitoring
- Data export
- Mobile/responsive
- Accessibility

---

## 12. Migration Completed Work

### Zerolog Migration (Completed 2025-12-23)

Migrated all logging from mixed approaches to zerolog v1.34.0:
- 638+ `log` calls migrated
- Structured JSON output in production
- Console output in development
- Context-based correlation IDs
- slog adapter for Suture compatibility

### Documentation Version Fixes

All version numbers verified and corrected:
- DuckDB: 1.4.3 (was 1.4.2 in old docs)
- MapLibre: 5.16.0 (was 5.15.0 in old docs)
- ADR count: 29 (was 25 in old docs)
- Route count: 302 (was 227 in old docs)
- BadgerDB: 4.9.0 (was 4.8.0 in old docs)

### External Service Safety

All external service integrations audited:
- 15 write methods identified (all admin-protected)
- Circuit breakers on all clients
- Rate limiting with exponential backoff
- 30-second HTTP timeouts
- Graceful shutdown on all WebSockets
- No credential logging

---

## 13. Remaining Enhancement Opportunities

### High Priority (Optional)

| Enhancement | Effort | Impact |
|-------------|--------|--------|
| Chaos Engineering Tests | 24-40h | System resilience |
| Performance Regression CI | 16-24h | Prevent regressions |
| OpenTelemetry Tracing | 16-24h | Distributed tracing |
| Grafana Dashboards | 8-16h | Observability |

### Medium Priority (Optional)

| Enhancement | Effort | Impact |
|-------------|--------|--------|
| AlertManager Rules | 8-16h | Proactive alerting |
| Secret Rotation Automation | 8-16h | Security hardening |
| Kubernetes HA Testing | 24-40h | Production validation |

### Tautulli API Expansion

53% of Tautulli API still available:
- Collections/playlists analytics
- Notification system integration
- Newsletter automation
- Advanced metadata search

### Plex API Expansion

Many read endpoints available:
- `/library/metadata/{id}/similar` - Recommendations
- `/statistics/resources` - System health
- `/butler` - Scheduled tasks
- Hub search and browse endpoints

---

## 14. Appendices

### A. Architecture Decision Records Index

| ADR | Title |
|-----|-------|
| 0001 | DuckDB for Analytics |
| 0002 | TypeScript + MapLibre + ECharts + deck.gl |
| 0003 | Multi-mode Authentication |
| 0004 | Suture v4 Process Supervision |
| 0005 | NATS JetStream Event Processing |
| 0006 | BadgerDB Write-Ahead Log |
| 0007 | Event Sourcing Architecture |
| 0008 | Circuit Breaker Pattern |
| 0009 | Plex Direct Integration |
| 0010 | Cursor-based Pagination |
| 0011 | Self-hosted CI/CD |
| 0012 | Koanf Configuration Management |
| 0013 | Request Validation |
| 0014 | Tautulli Database Import |
| 0015 | Zero Trust Authentication |
| 0016 | Chi Router Adoption |
| 0017 | Watermill Middleware |
| 0018 | DuckDB Community Extensions |
| 0019 | Testcontainers Integration |
| 0020 | Detection Rules Engine |
| 0021 | High-Performance JSON |
| 0022 | Dedupe Audit Management |
| 0023 | Exactly-Once Delivery |
| 0024 | Recommendation Engine |
| 0025 | Deterministic E2E Mocking |
| 0026 | Multi-Server Management UI |
| 0027 | WebSocket Real-Time Hub |
| 0028 | Jellyfin/Emby Integration |
| 0029 | Backup/Restore with GFS |

### B. External Documentation References

| Document | Purpose |
|----------|---------|
| CLAUDE.md | AI assistant development guide |
| README.md | User documentation |
| ARCHITECTURE.md | System architecture |
| CHANGELOG.md | Version history |
| SECURITY.md | Security policy |
| docs/API-REFERENCE.md | API documentation |
| docs/DEVELOPMENT.md | Development workflow |
| docs/TROUBLESHOOTING.md | Common issues |

### C. Archived Files Consolidated

This document replaces the following archived files:

**docs/archive/audit/**
- PHASE_1_CHECKPOINT.md through PHASE_9_CHECKPOINT.md

**docs/archive/working/**
- AUDIT_COMPLETED.md
- AUDIT_LEARNINGS.md
- AUDIT_REMAINING.md
- AUDIT_REPORT.md
- COMPREHENSIVE_BINARY_TESTING_REPORT.md
- DATA_OPPORTUNITIES_WOW_FACTOR.md
- DIAGNOSTIC_REPORT.md
- E2E_PLAYWRIGHT_INVESTIGATION.md
- FRONTEND_AUDIT_REPORT.md
- FRONTEND_INVENTORY.md
- OIDC_MIGRATION_ANALYSIS.md
- ONBOARDING_REVIEW.md
- PRE_RELEASE_AUDIT.md
- PRODUCTION_READINESS_AUDIT.md
- RBAC_IMPLEMENTATION_PLAN.md
- RECOMMENDATION_FEASIBILITY_ANALYSIS.md
- REVIEW_ACTION_PLAN.md
- REVIEW_COMMENTS.md
- REVIEW_DOCS.md
- REVIEW_INVENTORY.md
- WAL_VERIFICATION_REPORT.md

**docs/ (internal-only)**
- REMAINING_TASKS.md
- ZEROLOG_MIGRATION_PLAN.md
- CLAUDE_CODE_WEB_SETUP.md
- PLEX_API_GAP_ANALYSIS.md
- TAUTULLI_API_ANALYSIS.md
- COMPATIBILITY_ANALYSIS.md
- EXTERNAL_SERVICE_SAFETY_AUDIT.md
- ZITADEL_OIDC_SECURITY_AUDIT.md
- plans/duckdb-extensions-implementation.md

---

**Document Maintenance**

This document should be updated when:
1. Major version releases occur
2. New ADRs are added
3. Architecture changes are implemented
4. Security audits are completed
5. Technology stack versions change significantly

**Last Verification**: 2026-01-12 - All metrics verified against source code.
