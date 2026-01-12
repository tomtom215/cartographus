# Architecture Decision Records

This directory contains Architecture Decision Records (ADRs) for the Cartographus project. ADRs capture significant architectural decisions along with their context and consequences.

## What is an ADR?

An Architecture Decision Record (ADR) documents a single architectural decision and its justification. ADRs help understand the reasons for chosen architectural decisions, along with their trade-offs and consequences.

**Format**: This project follows the [Michael Nygard ADR template](https://github.com/joelparkerhenderson/architecture-decision-record/blob/main/locales/en/templates/decision-record-template-by-michael-nygard/index.md) with additional sections for implementation details and verification status.

**Naming Convention**: `NNNN-lowercase-with-dashes.md` (e.g., `0001-use-duckdb-for-analytics.md`)

---

## ADR Index

### Accepted

| ADR | Title | Date | Status |
|-----|-------|------|--------|
| [ADR-0001](0001-use-duckdb-for-analytics.md) | Use DuckDB for Analytics Database | 2025-11-16 | Accepted |
| [ADR-0002](0002-frontend-technology-stack.md) | Frontend Technology Stack Selection | 2025-11-16 | Accepted |
| [ADR-0003](0003-authentication-architecture.md) | Multi-Mode Authentication Architecture | 2025-11-16 | Accepted |
| [ADR-0004](0004-process-supervision-with-suture.md) | Process Supervision with Suture v4 | 2025-12-03 | Accepted |
| [ADR-0005](0005-nats-jetstream-event-processing.md) | NATS JetStream Event Processing | 2025-12-01 | Accepted |
| [ADR-0006](0006-badgerdb-write-ahead-log.md) | BadgerDB Write-Ahead Log for Event Durability | 2025-12-02 | Accepted |
| [ADR-0007](0007-event-sourcing-architecture.md) | Event Sourcing with NATS-First Architecture | 2025-12-02 | Accepted |
| [ADR-0008](0008-circuit-breaker-pattern.md) | Circuit Breaker for Tautulli API | 2025-11-23 | Accepted |
| [ADR-0009](0009-plex-direct-integration.md) | Plex Direct Integration for Standalone Operation | 2025-11-24 | Accepted |
| [ADR-0010](0010-cursor-based-pagination.md) | Cursor-Based Pagination for API | 2025-11-25 | Accepted |
| [ADR-0011](0011-ci-cd-infrastructure.md) | CI/CD Infrastructure with Self-Hosted Runners | 2025-12-02 | Accepted |
| [ADR-0012](0012-configuration-management-koanf.md) | Configuration Management with Koanf v2 | 2025-12-03 | Accepted |
| [ADR-0013](0013-request-validation.md) | Request Validation with go-playground/validator | 2025-12-04 | Accepted |
| [ADR-0014](0014-tautulli-database-import.md) | Tautulli Database Import | 2025-12-04 | Accepted |
| [ADR-0015](0015-zero-trust-authentication-authorization.md) | Zero Trust Authentication and Authorization | 2025-12-05 | Accepted |
| [ADR-0016](0016-chi-router-adoption.md) | Chi Router Adoption | 2025-12-08 | Accepted |
| [ADR-0017](0017-watermill-router-and-middleware.md) | Watermill Router and Middleware | 2025-12-08 | Accepted |
| [ADR-0018](0018-duckdb-community-extensions.md) | DuckDB Community Extensions (RapidFuzz + DataSketches) | 2025-12-13 | Accepted |
| [ADR-0019](0019-testcontainers-integration-testing.md) | Testcontainers Integration Testing | 2025-12-14 | Accepted |
| [ADR-0020](0020-detection-rules-engine.md) | Detection Rules Engine for Media Playback Security | 2025-12-16 | Accepted |
| [ADR-0021](0021-go-json-high-performance-json.md) | High-Performance JSON with goccy/go-json | 2025-12-18 | Accepted |
| [ADR-0022](0022-dedupe-audit-management.md) | Dedupe Audit and Data Management | 2025-12-20 | Accepted |
| [ADR-0023](0023-consumer-wal-exactly-once.md) | Consumer WAL for Exactly-Once Delivery | 2025-12-22 | Accepted |
| [ADR-0024](0024-recommendation-engine.md) | Hybrid Recommendation Engine for Media Content | 2025-12-29 | Accepted |
| [ADR-0025](0025-deterministic-e2e-mocking.md) | Deterministic E2E Test Mocking | 2026-01-01 | Accepted |
| [ADR-0026](0026-multi-server-management-ui.md) | Multi-Server Management UI | 2026-01-07 | Accepted |
| [ADR-0027](0027-websocket-real-time-hub.md) | WebSocket Real-Time Hub Architecture | 2026-01-11 | Accepted |
| [ADR-0028](0028-jellyfin-emby-integration.md) | Jellyfin and Emby Media Server Integration | 2026-01-11 | Accepted |
| [ADR-0029](0029-backup-restore-gfs-retention.md) | Backup and Restore with GFS Retention Strategy | 2026-01-11 | Accepted |

---

## ADR Lifecycle

ADRs in this project follow these statuses:

- **Proposed**: Under discussion, not yet decided
- **Accepted**: Decision made and implemented
- **Deprecated**: Decision no longer applies but kept for historical context
- **Superseded**: Replaced by another ADR (linked in the document)

---

## Creating a New ADR

1. Copy the template: `cp docs/adr/template.md docs/adr/NNNN-short-title.md`
2. Fill in all sections with factual, verified information
3. Add the ADR to this index
4. Submit for review via pull request

### Template Sections

- **Title**: Clear, descriptive name
- **Status**: Current lifecycle state
- **Context**: Problem or situation driving the decision
- **Decision**: The architectural choice made
- **Consequences**: Effects and trade-offs
- **Implementation**: Technical details (optional)
- **Verification**: Source code references (required)

---

## Related Documentation

| Document | Description |
|----------|-------------|
| [ARCHITECTURE.md](../ARCHITECTURE.md) | High-level system architecture |
| [DEVELOPMENT.md](../DEVELOPMENT.md) | Development workflow and conventions |
| [TROUBLESHOOTING.md](../TROUBLESHOOTING.md) | Common issues and solutions |
| [CLAUDE.md](../../CLAUDE.md) | AI assistant guide |

---

## Principles

These ADRs follow the principles outlined in the project's CLAUDE.md:

1. **Never Assume, Never Guess** - All claims verified against source code
2. **Base Everything on Facts** - Source code is the single source of truth
3. **Test-Driven Development** - Decisions align with TDD approach
4. **Professional Documentation** - No emojis, factual content only

---

**Last Updated**: 2026-01-11
**Last Audit**: 2026-01-11 (All 29 ADRs verified against source code)
**Total ADRs**: 29 (29 Accepted, 0 Proposed)
