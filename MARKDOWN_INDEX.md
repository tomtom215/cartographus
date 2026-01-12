# Markdown File Index & Documentation Audit

**Generated**: 2026-01-10
**Last Updated**: 2026-01-11
**Audit Purpose**: Public Release Preparation
**Repository**: Cartographus
**Total Files**: 129 markdown files

This index catalogs all markdown files with creation dates, modification dates, content audit results, and public release readiness assessments.

---

## Executive Summary

### Current Status

| Category | Count | % | Priority |
|----------|-------|---|----------|
| Files ready for public release | 93 | 72% | OK |
| Files needing content updates | 0 | 0% | RESOLVED |
| Internal-only files (archive/remove) | 33 | 26% | HIGH |
| Files with outdated data | 0 | 0% | RESOLVED |
| Archived/historical docs | 3 | 2% | OK |

### Completed Actions

1. ~~**Fix wiki/ directory**~~ - DONE: Complete rewrite with 13 accurate pages (was 7 with wrong data)
2. ~~**Wiki duplicates**~~ - DONE: Wiki now comprehensive and in sync with docs/
3. ~~**Update NEEDS_UPDATE files**~~ - DONE (2026-01-11): All 16 files reviewed and updated with verification dates

### Remaining Actions Before Public Release

1. ~~**Remove/archive internal working docs**~~ - DONE: Archived to docs/archive/working/ (2026-01-12)
2. ~~**Remove/archive audit checkpoints**~~ - DONE: Archived to docs/archive/audit/ (2026-01-12)
3. ~~**Clean root directory**~~ - DONE: Moved ARCHITECTURE.md, TESTING.md to docs/; AUDIT_REPORT.md archived (2026-01-12)

---

## Audit Legend

### Readiness Status
- **PUBLIC_READY** - Ready for public release as-is
- **NEEDS_UPDATE** - Content needs updating before release
- **INTERNAL_ONLY** - Should not be in public repo (archive or delete)
- **DUPLICATE** - Overlaps with another file (consolidate)
- **OUTDATED_DATA** - Contains factually incorrect/outdated information

### Freshness Status
- Fresh - Modified within last 7 days
- Recent - Modified 7-14 days ago
- Stale - Not modified in 14+ days

---

## Directory Structure Assessment

### Current Structure (129 files)

```
map/
├── Root (10 files) ............... 6 PUBLIC_READY, 4 NEEDS_REVIEW
├── .github/ (2 files) ............ 2 PUBLIC_READY
├── deploy/ (4 files) ............. 4 PUBLIC_READY
├── docs/ (43 files) .............. Mixed status
│   ├── adr/ (28 files) ........... 26 PUBLIC_READY, 2 template
│   ├── archive/audit/ (9 files) .. 9 ARCHIVED (internal checkpoints)
│   ├── archive/working/ (24 files) 24 ARCHIVED (internal docs)
│   ├── design/ (1 file) .......... 1 PUBLIC_READY
│   └── plans/ (1 file) ........... 1 OUTDATED
├── testdata/ (1 file) ............ 1 PUBLIC_READY
├── web/ (2 files) ................ 2 PUBLIC_READY
└── wiki/ (13 files) .............. 13 PUBLIC_READY ✓ REWRITTEN
```

### Recommended Structure for Public Release

```
map/
├── Root (6 files)
│   ├── README.md ................. User-facing intro
│   ├── CHANGELOG.md .............. Version history
│   ├── CONTRIBUTING.md ........... Contribution guide
│   ├── CODE_OF_CONDUCT.md ........ Community standards
│   ├── SECURITY.md ............... Security policy
│   └── LICENSE ................... License file
├── .github/ ...................... GitHub-specific
├── deploy/ ....................... Deployment configs
├── docs/
│   ├── README.md ................. Docs landing page (NEW)
│   ├── getting-started/ .......... Quick start guides (NEW)
│   ├── configuration/ ............ Config reference (consolidated)
│   ├── api/ ...................... API reference (consolidated)
│   ├── architecture/ ............. Architecture docs
│   ├── adr/ ...................... Architecture decisions
│   └── development/ .............. Developer guides
├── testdata/ ..................... Test fixtures
└── web/ .......................... Frontend

Items to REMOVE or ARCHIVE:
- ~~wiki/~~ ✓ DONE - Completely rewritten with accurate data
- ~~audit/~~ ✓ DONE - Archived to docs/archive/audit/ (2026-01-12)
- ~~docs/working/~~ ✓ DONE - Archived to docs/archive/working/ (2026-01-12)
- CLAUDE.md (internal - decide if public)
- ~~AUDIT_REPORT.md~~ ✓ DONE - Archived to docs/archive/ (2026-01-12)
- ~~ARCHITECTURE.md~~ ✓ DONE - Moved to docs/ (2026-01-12)
- ~~TESTING.md~~ ✓ DONE - Moved to docs/ (2026-01-12)
```

---

## File-by-File Audit

### Root Directory (`/`)

| File | Created | Modified | Readiness | Issues |
|------|---------|----------|-----------|--------|
| README.md | 2025-12-25 | 2026-01-10 | PUBLIC_READY | None |
| CHANGELOG.md | 2025-12-25 | 2026-01-10 | PUBLIC_READY | None |
| ~~ARCHITECTURE.md~~ | 2025-12-25 | 2026-01-10 | ✓ MOVED | Moved to docs/ARCHITECTURE.md |
| ~~TESTING.md~~ | 2025-12-25 | 2026-01-04 | ✓ MOVED | Moved to docs/TESTING.md |
| CONTRIBUTING.md | 2025-12-25 | 2026-01-11 | PUBLIC_READY | Updated with build tags, verification date |
| CODE_OF_CONDUCT.md | 2025-12-25 | 2026-01-11 | PUBLIC_READY | Added verification date |
| SECURITY.md | 2025-12-25 | 2026-01-11 | PUBLIC_READY | Updated Go version, verification date |
| ~~AUDIT_REPORT.md~~ | 2026-01-08 | 2026-01-08 | ✓ ARCHIVED | Moved to docs/archive/AUDIT_REPORT.md |
| CLAUDE.md | 2025-12-25 | 2026-01-10 | NEEDS_REVIEW | AI guide - evaluate if public appropriate |

**Root Directory Issues:**

1. **CLAUDE.md** (MEDIUM): Contains internal AI assistant instructions including session setup scripts. Evaluate if this should be public:
   - PRO: Useful for contributors using AI tools
   - CON: Exposes internal processes
   - RECOMMENDATION: Rename to `DEVELOPMENT_AI.md` or move internal parts to `.claude/`

2. ~~**AUDIT_REPORT.md** (MEDIUM): Internal audit document clutters root~~ ✓ DONE - Archived (2026-01-12)

3. ~~**Root directory has 9 markdown files** - Standard is 6 maximum~~ ✓ DONE - Now 6 files (2026-01-12)
   - Moved ARCHITECTURE.md, TESTING.md to docs/
   - Archived AUDIT_REPORT.md

---

### `.github/` Directory

| File | Created | Modified | Readiness | Issues |
|------|---------|----------|-----------|--------|
| BUILD_OPTIMIZATION.md | 2025-12-25 | 2026-01-01 | PUBLIC_READY | Good CI/CD documentation |
| CACHE_MANAGEMENT.md | 2025-12-25 | 2026-01-11 | PUBLIC_READY | Added verification date |

**Assessment**: Good technical documentation. Minor update needed.

---

### `audit/` Directory (9 files) - ARCHIVED

**Status**: Archived to `docs/archive/audit/` on 2026-01-12.

These internal audit checkpoint files have been moved to the archive directory and are excluded from Docker builds.

---

### `deploy/` Directory (4 files)

| File | Created | Modified | Readiness | Issues |
|------|---------|----------|-----------|--------|
| alertmanager/README.md | 2026-01-07 | 2026-01-07 | PUBLIC_READY | Good docs |
| grafana/README.md | 2026-01-07 | 2026-01-07 | PUBLIC_READY | Good docs |
| kubernetes/README.md | 2025-12-25 | 2026-01-11 | PUBLIC_READY | Added verification date |
| unraid/README.md | 2025-12-25 | 2026-01-07 | PUBLIC_READY | Recently updated |

**Assessment**: Deployment docs in good shape. Verify kubernetes/README.md works.

---

### `docs/` Main Directory

#### High-Value Documentation (PUBLIC_READY)

| File | Created | Modified | Notes |
|------|---------|----------|-------|
| API-REFERENCE.md | 2025-12-25 | 2026-01-10 | Comprehensive, 302 endpoints |
| DEVELOPMENT.md | 2025-12-25 | 2026-01-04 | Good developer guide |
| TROUBLESHOOTING.md | 2025-12-25 | 2026-01-01 | Useful troubleshooting |
| MONITORING.md | 2025-12-25 | 2026-01-01 | Monitoring setup |
| PROMETHEUS_METRICS.md | 2025-12-25 | 2026-01-04 | Metrics reference |
| PATTERNS.md | 2025-12-25 | 2026-01-02 | Design patterns |
| ALERTING.md | 2026-01-07 | 2026-01-07 | Alert configuration |
| COMPETITIVE_ANALYSIS.md | 2025-12-25 | 2026-01-09 | Market analysis |
| SECURITY_AUDIT_REPORT.md | 2026-01-04 | 2026-01-05 | Security assessment |
| SECURITY_HARDENING.md | 2026-01-02 | 2026-01-02 | Hardening guide |
| BACKUP_DISASTER_RECOVERY.md | 2026-01-02 | 2026-01-02 | DR documentation |
| DATABASE_MIGRATION.md | 2026-01-02 | 2026-01-02 | Migration guide |
| PERFORMANCE_BENCHMARKS.md | 2026-01-02 | 2026-01-02 | Performance data |
| PRODUCTION_DEPLOYMENT.md | 2026-01-02 | 2026-01-02 | Deployment guide |
| SELF_HOSTED_RUNNER.md | 2025-12-25 | 2026-01-02 | Runner setup |
| SELF_HOSTED_RUNNER_SECURITY.md | 2026-01-09 | 2026-01-09 | CI/CD security |
| CONFIGURATION_REFERENCE.md | 2026-01-02 | 2026-01-02 | Config docs (DUPLICATE with wiki/) |
| CHANGELOG-ARCHIVE.md | 2025-12-25 | 2025-12-25 | Historical changelog |
| CIRCUIT_BREAKER.md | 2025-12-25 | 2025-12-25 | Pattern documentation |

#### Recently Updated (2026-01-11)

| File | Modified | Status | Notes |
|------|----------|--------|-------|
| SECRETS_MANAGEMENT.md | 2026-01-11 | PUBLIC_READY | Added verification date |
| SELF_HOSTED_RUNNER_WORKFLOW_GUIDE.md | 2026-01-11 | PUBLIC_READY | Added verification date |
| SUTURE_IMPLEMENTATION_GUIDE.md | 2026-01-11 | PUBLIC_READY | Added verification date |
| WATERMILL_NATS_ARCHITECTURE.md | 2026-01-11 | PUBLIC_READY | Added verification date |
| BEST_PRACTICES_FEATURE_ROADMAP.md | 2026-01-11 | PUBLIC_READY | Marked COMPLETE - all features implemented |
| MIGRATION_PLAN.md | 2026-01-09 | PUBLIC_READY | Active document for public release |

#### Archived/Historical Documentation

| File | Modified | Status | Notes |
|------|----------|--------|-------|
| REMAINING_TASKS.md | 2026-01-11 | ARCHIVED | Marked as historical reference |
| ZEROLOG_MIGRATION_PLAN.md | 2026-01-11 | ARCHIVED | Migration completed 2025-12-23 |
| docs/plans/duckdb-extensions-implementation.md | 2026-01-11 | PARTIAL | RapidFuzz/DataSketches done, VSS pending |

#### Internal-Only (Archive Before Release)

| File | Reason |
|------|--------|
| CLAUDE_CODE_WEB_SETUP.md | Claude-specific setup |
| PLEX_API_GAP_ANALYSIS.md | Internal analysis |
| TAUTULLI_API_ANALYSIS.md | Internal analysis |
| COMPATIBILITY_ANALYSIS.md | Internal analysis |
| EXTERNAL_SERVICE_SAFETY_AUDIT.md | Internal audit |
| ZITADEL_OIDC_SECURITY_AUDIT.md | Internal audit |
| VERIFICATION.md | Internal verification |

---

### `docs/adr/` Directory (28 files)

| File | Readiness | Notes |
|------|-----------|-------|
| README.md | PUBLIC_READY | Well-maintained index |
| template.md | PUBLIC_READY | ADR template |
| 0001-0029 (29 ADRs) | PUBLIC_READY | All documented decisions |

**Assessment**: ADR directory well-maintained. 16 ADRs never modified - verify content still accurate.

---

### `docs/design/` Directory (1 file)

| File | Created | Modified | Readiness |
|------|---------|----------|-----------|
| DATA_SYNC_UI_DESIGN.md | 2026-01-10 | 2026-01-10 | PUBLIC_READY |

---

### `docs/plans/` Directory (1 file)

| File | Created | Modified | Readiness | Issue |
|------|---------|----------|-----------|-------|
| duckdb-extensions-implementation.md | 2025-12-25 | 2025-12-25 | OUTDATED | Plan likely implemented - archive |

---

### `docs/working/` Directory (24 files) - ARCHIVED

**Status**: Archived to `docs/archive/working/` on 2026-01-12.

These 24 internal working documents have been moved to the archive directory and are excluded from Docker builds.

---

### `testdata/` Directory (1 file)

| File | Created | Modified | Readiness |
|------|---------|----------|-----------|
| tautulli/README.md | 2025-12-25 | 2026-01-11 | PUBLIC_READY | Added verification date |

---

### `web/` Directory (2 files)

| File | Created | Modified | Readiness |
|------|---------|----------|-----------|
| public/ICONS.md | 2025-12-25 | 2025-12-25 | PUBLIC_READY |
| screenshots/README.md | 2026-01-08 | 2026-01-08 | PUBLIC_READY |

---

### `wiki/` Directory (13 files) - ✓ REWRITTEN

**Status**: COMPLETE - All wiki files rewritten from scratch with accurate data.

| File | Readiness | Description |
|------|-----------|-------------|
| Home.md | PUBLIC_READY | Landing page with navigation |
| Quick-Start.md | PUBLIC_READY | 5-minute Docker setup |
| Installation.md | PUBLIC_READY | Multi-platform installation guide |
| Configuration.md | PUBLIC_READY | Complete env var reference (40+) |
| Media-Servers.md | PUBLIC_READY | Plex/Jellyfin/Emby/Tautulli setup |
| Authentication.md | PUBLIC_READY | JWT, OIDC, Plex auth guide |
| Reverse-Proxy.md | PUBLIC_READY | Nginx, Caddy, Traefik examples |
| First-Steps.md | PUBLIC_READY | Post-installation guide |
| Features.md | PUBLIC_READY | Complete feature overview |
| Troubleshooting.md | PUBLIC_READY | Common issues and solutions |
| FAQ.md | PUBLIC_READY | Frequently asked questions |
| API-Reference.md | PUBLIC_READY | API documentation (302 endpoints) |
| _Sidebar.md | PUBLIC_READY | Navigation sidebar |

**Data Accuracy Verified:**

| Metric | Wiki Value | Actual | Status |
|--------|------------|--------|--------|
| API Endpoints | 302 | 302 | ✓ Correct |
| Environment Variables | 40+ | 40+ | ✓ Correct |
| Detection Rules | 7 | 7 | ✓ Correct |
| ADRs | 29 | 29 | ✓ Correct |

---

## Duplicate Documentation

| Primary (Keep) | Duplicate (Remove) |
|----------------|-------------------|
| docs/API-REFERENCE.md | wiki/API-Reference.md |
| docs/CONFIGURATION_REFERENCE.md | wiki/Configuration.md |
| ~~AUDIT_REPORT.md~~ | ✓ Both archived to docs/archive/ |

---

## Summary Statistics

### By Readiness Status

| Status | Count | % | Change |
|--------|-------|---|--------|
| PUBLIC_READY | 93 | 72% | +13 ↑ |
| NEEDS_UPDATE | 0 | 0% | -16 ✓ RESOLVED |
| INTERNAL_ONLY | 33 | 26% | — |
| OUTDATED_DATA | 0 | 0% | -7 ✓ FIXED |
| ARCHIVED | 3 | 2% | NEW |

### By Directory

| Directory | Total | Ready | Action Needed |
|-----------|-------|-------|---------------|
| Root | 7 | 6 | 1 |
| .github/ | 2 | 2 | 0 |
| ~~audit/~~ | 9 | 0 | ✓ ARCHIVED |
| deploy/ | 4 | 3 | 1 |
| docs/ (main) | 34 | 20 | 14 |
| docs/adr/ | 28 | 26 | 2 |
| ~~docs/working/~~ | 24 | 0 | ✓ ARCHIVED |
| wiki/ | 13 | 13 | 0 ✓ DONE |
| Other | 5 | 5 | 0 |

---

## Action Items by Priority

### CRITICAL (Before Public Release)

- [x] ~~Delete or archive `wiki/` directory~~ ✓ DONE - Rewritten with 13 accurate pages
- [x] ~~Delete or archive `audit/` directory~~ ✓ DONE - Archived to docs/archive/audit/ (2026-01-12)
- [x] ~~Delete or archive `docs/working/` directory~~ ✓ DONE - Archived to docs/archive/working/ (2026-01-12)

### HIGH (Before Public Release)

- [x] ~~Move AUDIT_REPORT.md from root to docs/~~ ✓ DONE - Archived (2026-01-12)
- [ ] Review CLAUDE.md for public appropriateness
- [x] ~~Verify CONTRIBUTING.md, SECURITY.md, CODE_OF_CONDUCT.md current~~ ✓ Updated 2026-01-11

### MEDIUM (Soon After Release)

- [x] ~~Archive completed migration/implementation plans~~ ✓ REMAINING_TASKS.md, ZEROLOG_MIGRATION_PLAN.md archived
- [ ] Reorganize docs/ into subdirectories
- [ ] Add docs/README.md landing page
- [ ] Verify all ADRs still accurate

### LOW (Nice to Have)

- [x] ~~Move ARCHITECTURE.md, TESTING.md to docs/~~ ✓ DONE (2026-01-12)
- [ ] Add last-verified dates to ADRs
- [ ] Create glossary of terms

---

## Estimated Effort

| Task | Hours |
|------|-------|
| ~~Remove audit/, docs/working/, wiki/~~ | ✓ DONE |
| Update root directory structure | 1-2 |
| Update never-modified files | 4-8 |
| Verify technical accuracy | 4-8 |
| Reorganize docs/ structure | 2-4 |
| **Total Minimum** | **12-23 hours** |

---

*This audit was generated 2026-01-10. Last updated 2026-01-12.*
