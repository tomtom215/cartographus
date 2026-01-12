# Cartographus Public Release Migration Plan

**From**: `github.com/tomtom215/cartographus` (private stealth repo)
**To**: `github.com/tomtom215/cartographus` (public release repo)
**Status**: Planning
**Created**: 2026-01-09
**Updated**: 2026-01-09

---

## Decisions Made

| Decision | Choice | Notes |
|----------|--------|-------|
| **License** | AGPL-3.0 | Network copyleft for SaaS protection |
| **Database path** | `/data/cartographus.duckdb` | Breaking change, migration guide needed |
| **Git history** | Fresh start | Clean slate, no legacy commits |
| **Old repo** | Keep, eventually archive | Redirect notice in README |

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [CRITICAL: Safe Replacement Patterns](#critical-safe-replacement-patterns)
3. [License Change Analysis](#3-license-change-analysis)
4. [Pre-Migration Checklist](#4-pre-migration-checklist)
5. [Phase 1: Code Changes](#5-phase-1-code-changes)
6. [Phase 2: Configuration Updates](#6-phase-2-configuration-updates)
7. [Phase 3: Documentation Updates](#7-phase-3-documentation-updates)
8. [Phase 4: License Implementation](#8-phase-4-license-implementation)
9. [Phase 5: GitHub Repository Setup](#9-phase-5-github-repository-setup)
10. [Phase 6: CI/CD Migration](#10-phase-6-cicd-migration)
11. [Phase 7: Final Verification](#11-phase-7-final-verification)
12. [Post-Migration Tasks](#12-post-migration-tasks)
13. [Rollback Plan](#13-rollback-plan)
14. [File Change Summary](#14-file-change-summary)
15. [CRITICAL: Self-Hosted Runner Security](#15-critical-self-hosted-runner-security)
16. [Appendices](#appendix-a-automated-migration-script)
    - [Appendix A: Automated Migration Script](#appendix-a-automated-migration-script)
    - [Appendix B: License Header Script](#appendix-b-license-header-script)
    - [Appendix C: User Upgrade Guide](#appendix-c-user-upgrade-guide)
    - [Appendix D: Go Module Deprecation](#appendix-d-go-module-deprecation)
    - [Appendix E: Migration Testing Procedure](#appendix-e-migration-testing-procedure)
    - [Appendix F: CHANGELOG Entry Template](#appendix-f-changelog-entry-template)

---

## 1. Executive Summary

### Scope

This migration involves:
- **468+ files** with repository URL references
- **461 Go files** with import path changes
- **License change** from MIT to AGPL-3.0 or GPL-3.0
- **Complete rebrand** of repository namespace from `map` to `cartographus`
- **Docker image** migration from `ghcr.io/tomtom215/cartographus` to `ghcr.io/tomtom215/cartographus`

### Key Decisions (RESOLVED)

| Decision | Current | **CHOSEN** | Impact |
|----------|---------|------------|--------|
| License | MIT | **AGPL-3.0** | Network copyleft, SaaS protection |
| Database path | `/data/cartographus.duckdb` | **`/data/cartographus.duckdb`** | Breaking change - migration guide needed |
| Binary name | `cartographus` | **Keep as-is** | No change needed |
| Git history | Full history | **Fresh start** | Clean slate, no legacy commits |
| Old repo | N/A | **Keep, archive later** | Redirect notice in README |

### Timeline Recommendation

This migration should be executed in a single coordinated release to avoid confusion. Do not publish partial changes.

### CRITICAL: Self-Hosted Runner Security

**Before making the repository public, the CI/CD configuration MUST be updated.**

The current configuration runs PR-triggered workflows on self-hosted runners, which allows arbitrary code execution by external contributors.

**See**: [SELF_HOSTED_RUNNER_SECURITY.md](./SELF_HOSTED_RUNNER_SECURITY.md) for detailed analysis and remediation steps.

**Summary of required changes:**
1. Move all `pull_request` triggered workflows to GitHub-hosted runners
2. Remove write permissions (packages, id-token) from PR workflows
3. Disable image push for PR builds
4. Keep self-hosted runners for main branch builds only

---

## CRITICAL: Safe Replacement Patterns

> ⚠️ **STOP AND READ THIS ENTIRE SECTION BEFORE RUNNING ANY MIGRATION COMMANDS** ⚠️
>
> The project name "map" is an **extremely common programming term**. A naive find-and-replace will cause **catastrophic, unrecoverable damage** to the codebase.

### The Problem

```
"map" appears in 5,800+ locations in this codebase:
  - 976 are SAFE project references (github.com/tomtom215/cartographus, etc.)
  - 4,800+ are DANGEROUS code patterns (map[string], .map(), MapLibre, etc.)

A blind replacement would DESTROY the entire codebase.
```

### ❌ DANGEROUS Patterns - NEVER Replace These

| Pattern | Count | Example | What Happens If Replaced |
|---------|-------|---------|--------------------------|
| `map[` | 1,760 | `map[string]bool`, `map[int]User` | **Go compilation fails** - breaks all Go map type declarations |
| `.map(` | 472 | `items.map(x => x.id)` | **JavaScript runtime crashes** - breaks Array.prototype.map |
| `*Map*` / `*map*` variable names | 3,340+ | `userMapping`, `MapManager`, `hashMap` | **Logic errors** - renames unrelated identifiers |
| MapLibre/Mapbox references | 258 | `MapLibreMap`, `mapbox-gl`, `maplibre-gl` | **UI breaks** - destroys map visualization library refs |
| Files with "map" in name | 19 | `map.ts`, `map.css`, `user_mapping.go` | **Path errors** - file imports break |
| **Total DANGEROUS** | **5,800+** | | **CATASTROPHIC** |

#### Verification Commands for Dangerous Patterns

```bash
# Count Go map type declarations (NEVER replace these)
grep -r "map\[" --include="*.go" . | wc -l
# Expected: ~1,760

# Count JavaScript/TypeScript .map() calls (NEVER replace these)
grep -r "\.map(" --include="*.ts" --include="*.js" . | wc -l
# Expected: ~472

# Count map-related variable/function names (NEVER replace these)
grep -riE "[A-Za-z_]map|map[A-Za-z_]" --include="*.go" --include="*.ts" . | grep -v "github.com/tomtom215/cartographus" | wc -l
# Expected: ~3,340

# Count MapLibre/Mapbox references (NEVER replace these)
grep -riE "maplibre|mapbox" --include="*.ts" --include="*.js" --include="*.go" --include="*.md" . | wc -l
# Expected: ~258

# Count files with "map" in filename (NEVER rename these)
find . -type f \( -name "*map*" -o -name "*Map*" \) | grep -v node_modules | grep -v .git | wc -l
# Expected: ~19
```

### ✅ SAFE Patterns - Only Replace These Exact Strings

| Pattern | Count | Replace With | File Types |
|---------|-------|--------------|------------|
| `github.com/tomtom215/cartographus` | 908 | `github.com/tomtom215/cartographus` | All (Go imports, docs, configs) |
| `ghcr.io/tomtom215/cartographus` | 51 | `ghcr.io/tomtom215/cartographus` | YAML, MD |
| `cartographus.duckdb` | 74 | `cartographus.duckdb` | Go, YAML, MD, env |
| **Total SAFE** | **1,033** | | |

#### Verification Commands for Safe Patterns

```bash
# Count github.com/tomtom215/cartographus (ALL occurrences - this is SAFE)
grep -r "github.com/tomtom215/cartographus" . 2>/dev/null | grep -v ".git/" | wc -l
# Expected: ~908

# Count ghcr.io/tomtom215/cartographus (SAFE)
grep -r "ghcr.io/tomtom215/cartographus" --include="*.yml" --include="*.yaml" --include="*.md" . | wc -l
# Expected: ~51

# Count cartographus.duckdb (SAFE)
grep -r "map\.duckdb" --include="*.go" --include="*.md" --include="*.yml" --include="*.yaml" --include="*.env*" . | wc -l
# Expected: ~74
```

### Safe Replacement Commands (COPY-PASTE THESE EXACTLY)

```bash
# ══════════════════════════════════════════════════════════════════════════════
# SAFE COMMANDS - These patterns are unambiguous project references
# ══════════════════════════════════════════════════════════════════════════════

# 1. Update go.mod module declaration (MUST be first)
sed -i 's|module github.com/tomtom215/cartographus|module github.com/tomtom215/cartographus|' go.mod

# 2. Update Go imports (safe - full URL is unique)
find . -type f -name "*.go" -exec sed -i 's|github.com/tomtom215/cartographus|github.com/tomtom215/cartographus|g' {} +

# 3. Update documentation and config files (safe - full URL is unique)
find . -type f \( -name "*.md" -o -name "*.yml" -o -name "*.yaml" -o -name "*.json" \
  -o -name "*.ts" -o -name "*.html" -o -name "*.xml" -o -name "*.txt" \) \
  -exec sed -i 's|github.com/tomtom215/cartographus|github.com/tomtom215/cartographus|g' {} +

# 4. Update Docker image references (safe - full registry URL is unique)
find . -type f \( -name "*.yml" -o -name "*.yaml" -o -name "*.md" -o -name "*.xml" \) \
  -exec sed -i 's|ghcr.io/tomtom215/cartographus|ghcr.io/tomtom215/cartographus|g' {} +

# 5. Update database filename (safe - includes file extension)
find . -type f \( -name "*.go" -o -name "*.md" -o -name "*.yml" -o -name "*.yaml" \
  -o -name "*.env*" -o -name "Dockerfile*" \) \
  -exec sed -i 's|map\.duckdb|cartographus.duckdb|g' {} +
```

### ❌ CATASTROPHIC Commands - NEVER RUN THESE

```bash
# ══════════════════════════════════════════════════════════════════════════════
# CATASTROPHIC - These commands will DESTROY the codebase
# ══════════════════════════════════════════════════════════════════════════════

# NEVER DO THIS - replaces ALL occurrences of "map" including Go keywords
sed -i 's/map/cartographus/g' **/*.go          # DESTROYS 1,760+ Go map declarations

# NEVER DO THIS - word boundary still matches Go map keyword
sed -i 's/\bmap\b/cartographus/g' **/*.go      # DESTROYS Go map keyword usage

# NEVER DO THIS - global replacement destroys .map() calls
sed -i 's/map/cartographus/g' **/*.ts          # DESTROYS 472+ Array.map() calls

# NEVER DO THIS - partial match breaks MapLibre
sed -i 's/map/cartographus/gi' **/*            # DESTROYS everything
```

### Pre-Migration Verification Checklist

Before running any migration commands, verify:

- [ ] Run `./scripts/migration-dry-run.sh` and review the "DANGEROUS PATTERNS" section
- [ ] Confirm dangerous pattern counts match expected values (see table above)
- [ ] Confirm safe pattern counts match expected values
- [ ] Back up the repository: `cp -r . ../map-backup-$(date +%Y%m%d)`
- [ ] Ensure you're on a clean git state: `git status` shows no uncommitted changes
- [ ] Create a migration branch: `git checkout -b migration/map-to-cartographus`

### Post-Migration Verification

After running safe replacement commands:

```bash
# 1. Verify NO dangerous patterns were touched
grep -r "cartographus\[" --include="*.go" .              # Should return 0 results
grep -r "\.cartographus(" --include="*.ts" .             # Should return 0 results

# 2. Verify safe patterns were updated
grep -r "github.com/tomtom215/cartographus" . | grep -v ".git/"   # Should return 0 results
grep -r "ghcr.io/tomtom215/cartographus" .                        # Should return 0 results
grep -r "map\.duckdb" .                                  # Should return 0 results

# 3. Verify build still works
go build -tags "wal,nats" -o cartographus ./cmd/server

# 4. Verify tests pass
go test -tags "wal,nats" -race ./...
```

---

## 3. License Change Analysis

### Current License: MIT

The MIT license is permissive and allows:
- Commercial use without disclosure
- Proprietary modifications
- No copyleft requirements

### Option A: AGPL-3.0 (Recommended for SaaS Protection)

**Pros:**
- Network copyleft: Users accessing over a network must receive source code
- Prevents SaaS providers from offering hosted versions without contributing back
- Strong copyleft protects against proprietary forks
- Widely respected in self-hosted software community (Nextcloud, Grafana OSS)

**Cons:**
- May discourage corporate adoption
- Some organizations have blanket AGPL bans
- Requires license header in all source files

**Best for:** Projects wanting to prevent commercial exploitation without contribution

### Option B: GPL-3.0 (Traditional Copyleft)

**Pros:**
- Well-understood copyleft license
- Requires derivative works to be GPL
- More corporate-friendly than AGPL
- No network clause (only applies to distributed software)

**Cons:**
- Does not cover SaaS use (no network copyleft)
- Hosted services can run modified versions without releasing source

**Best for:** Projects primarily distributed as binaries/containers

### Recommendation

**AGPL-3.0** is recommended for Cartographus because:
1. It's self-hosted software often deployed via Docker
2. Prevents commercial SaaS providers from offering hosted versions without contributing
3. Aligns with similar projects (Plausible Analytics, Chatwoot, etc.)
4. The "network copyleft" clause is specifically designed for server software

### License Compatibility Check

Current dependencies and their licenses:

| Dependency | License | AGPL Compatible |
|------------|---------|-----------------|
| DuckDB | MIT | Yes |
| Chi Router | MIT | Yes |
| Watermill | MIT | Yes |
| NATS | Apache-2.0 | Yes |
| BadgerDB | Apache-2.0 | Yes |
| Casbin | Apache-2.0 | Yes |
| zerolog | MIT | Yes |
| MapLibre GL JS | BSD-3-Clause | Yes |
| ECharts | Apache-2.0 | Yes |
| deck.gl | MIT | Yes |

**Result:** All dependencies are compatible with AGPL-3.0.

---

## 4. Pre-Migration Checklist

### Dry-Run Analysis (Run First)

Before making any changes, run the dry-run script to analyze the codebase:

```bash
# Basic analysis
./scripts/migration-dry-run.sh

# Verbose output (shows all files)
./scripts/migration-dry-run.sh --verbose

# Save report to file
./scripts/migration-dry-run.sh --output migration-report.txt

# Generate JSON output for automation
./scripts/migration-dry-run.sh --json

# Combined: verbose report + JSON
./scripts/migration-dry-run.sh --verbose --output migration-report.txt --json
```

**Dry-run script checks (9 sections):**

| Section | Description |
|---------|-------------|
| 0. Environment Snapshot | Git state, tool versions, current module |
| 1. Go Module | go.mod and import paths |
| 2. Docker Images | Container registry references |
| 3. GitHub URLs | Repository URLs in docs/config |
| 4. Database Paths | DuckDB file references |
| 5. License | LICENSE file and MIT references |
| 6. CI/CD Workflows | Self-hosted runner vulnerabilities |
| 7. Sensitive Data | Secrets, credentials, internal URLs |
| 8. Additional Checks | package.json, Makefile, GoReleaser, GitHub templates, large/binary files, Swagger/OpenAPI, license headers, CHANGELOG |

**Current dry-run results (2026-01-09):**

| Category | Files | Status |
|----------|-------|--------|
| go.mod | 1 | Needs update |
| Go imports | 427 | Needs update |
| Docker images | 21 | Needs update |
| GitHub URLs | 46 | Needs update |
| Database paths | 35 | Needs update |
| LICENSE file | 1 | Needs replacement |
| License refs | 6 | Needs update |
| Vulnerable workflows | 2 | **CRITICAL** |
| GoReleaser | 1 | Needs update |
| Swagger/OpenAPI | 2 | Needs regeneration |
| GitHub config | 3 | Missing (CODEOWNERS, templates) |
| **Total** | ~536 | 5 critical issues |

### Security Audit

- [ ] **Run dry-run script**: `./scripts/migration-dry-run.sh --output report.txt`
- [ ] **Secrets scan**: Run `gitleaks detect` on full git history
- [ ] **Environment files**: Verify no `.env` files with real secrets exist
- [ ] **API keys**: Search for hardcoded API keys, tokens, passwords
- [ ] **Personal data**: Remove any test data with real user information
- [ ] **Internal URLs**: Remove any references to internal infrastructure (60 files flagged)
- [ ] **Comments**: Review code comments for sensitive information

### Code Quality

- [ ] **All tests pass**: `go test -tags "wal,nats" -race ./...`
- [ ] **E2E tests pass**: `npm run test:e2e`
- [ ] **Linting clean**: `go vet -tags "wal,nats" ./...` and `npx tsc --noEmit`
- [ ] **No TODO comments** with internal references
- [ ] **Documentation current**: All docs reflect actual functionality

### Git History

- [ ] **Consider fresh history**: Start with clean history (squash or fresh init)
- [ ] **Remove large files**: Check for accidentally committed binaries
- [ ] **BFG Repo Cleaner**: Remove any secrets from history if needed

### Files to Potentially Remove/Clean

| File/Pattern | Action | Reason |
|--------------|--------|--------|
| ~~`docs/working/*`~~ | ✓ DONE | Archived to docs/archive/working/ |
| ~~`audit/*`~~ | ✓ DONE | Archived to docs/archive/audit/ |
| `scripts/fix_remaining_issues.py` | Remove | Development artifact |
| `scripts/setup-runner-host.sh` | Clean | Contains internal runner URLs |
| `.env*` (if any real ones) | Remove | Secrets |
| `testdata/` with real data | Clean | Privacy |

---

## 5. Phase 1: Code Changes

### 5.1 Go Module Rename (Critical - Do First)

**File:** `go.mod`

```diff
-module github.com/tomtom215/cartographus
+module github.com/tomtom215/cartographus
```

### 5.2 Go Import Path Updates (461 files)

All Go files must have their import paths updated:

```diff
-import "github.com/tomtom215/cartographus/internal/api"
+import "github.com/tomtom215/cartographus/internal/api"
```

**Automated command:**
```bash
# From repository root
find . -name "*.go" -type f -exec sed -i 's|github.com/tomtom215/cartographus|github.com/tomtom215/cartographus|g' {} +
```

**Packages affected:**
- `internal/api`
- `internal/auth`
- `internal/authz`
- `internal/audit`
- `internal/backup`
- `internal/bandwidth`
- `internal/cache`
- `internal/config`
- `internal/database`
- `internal/detection`
- `internal/eventprocessor`
- `internal/import`
- `internal/logging`
- `internal/metrics`
- `internal/middleware`
- `internal/models`
- `internal/newsletter`
- `internal/recommend`
- `internal/supervisor`
- `internal/sync`
- `internal/templates`
- `internal/vpn`
- `internal/wal`
- `internal/websocket`
- `cmd/server`

### 5.3 Swagger/OpenAPI Regeneration

After import path changes, regenerate Swagger docs:

```bash
swag init -g cmd/server/main.go -o docs/
```

**Files to regenerate:**
- `docs/docs.go`
- `docs/swagger.json`
- `docs/swagger.yaml`

### 5.4 Verify Build

```bash
# Clean module cache
go clean -modcache

# Download dependencies
go mod download

# Verify module
go mod verify

# Build
CGO_ENABLED=1 go build -tags "wal,nats" -v -o cartographus ./cmd/server

# Run tests
go test -tags "wal,nats" -race ./...
```

---

## 6. Phase 2: Configuration Updates

### 6.1 Docker Image References

**Files requiring Docker image updates:**

| File | Line(s) | Change |
|------|---------|--------|
| `docker-compose.yml` | 6, image | `ghcr.io/tomtom215/cartographus` -> `ghcr.io/tomtom215/cartographus` |
| `README.md` | 54 | Docker image in quick start |
| `wiki/Quick-Start.md` | Multiple | Docker compose examples |
| `wiki/Configuration.md` | Multiple | Docker compose examples |
| `wiki/Home.md` | Multiple | Docker examples |
| `deploy/kubernetes/deployment.yaml` | 28 | Image reference |
| `deploy/kubernetes/kustomization.yaml` | 21 | Image name |
| `deploy/unraid/cartographus.xml` | 4, 13 | Image and icon |
| `.goreleaser.yml` | Multiple | Release notes |
| `.github/workflows/_build.yml` | 17-18 | IMAGE_NAME env var |
| `.github/workflows/release.yml` | Multiple | Docker tags |

**Automated command:**
```bash
find . -type f \( -name "*.yml" -o -name "*.yaml" -o -name "*.md" -o -name "*.xml" \) \
  -exec sed -i 's|ghcr.io/tomtom215/cartographus|ghcr.io/tomtom215/cartographus|g' {} +
```

### 6.2 GitHub Repository URLs

**Pattern to replace:** `github.com/tomtom215/cartographus` -> `github.com/tomtom215/cartographus`

**Files affected (468 total):**

| Category | Count | Examples |
|----------|-------|----------|
| Go source files | 461 | All `.go` files |
| Documentation | 36+ | README, CONTRIBUTING, docs/*, wiki/* |
| CI/CD workflows | 15 | `.github/workflows/*.yml` |
| Deployment configs | 9 | Kubernetes, Prometheus, Unraid |
| Web frontend | 3 | index.html, HelpDocumentationManager.ts, security.txt |

**Automated command:**
```bash
# All text files
find . -type f \( -name "*.go" -o -name "*.md" -o -name "*.yml" -o -name "*.yaml" \
  -o -name "*.json" -o -name "*.ts" -o -name "*.html" -o -name "*.xml" -o -name "*.txt" \) \
  -exec sed -i 's|github.com/tomtom215/cartographus|github.com/tomtom215/cartographus|g' {} +
```

### 6.3 CI/CD Workflow Updates

**`.github/workflows/_build.yml`:**
```yaml
env:
  REGISTRY: ghcr.io
  IMAGE_NAME: ${{ github.repository }}  # Will auto-resolve to new repo
```

**`.github/workflows/build-binaries.yml`:**
```yaml
# Line 329 - owner field
owner: tomtom215
name: cartographus  # Change from 'map'
```

**`.github/workflows/release.yml`:**
- Update all Docker image tag references
- Update release notes template URLs

### 6.4 GoReleaser Configuration

**`.goreleaser.yml`:**
```yaml
project_name: cartographus  # Already correct

# Update any hardcoded URLs in release notes
release:
  github:
    owner: tomtom215
    name: cartographus  # Change from 'map'
```

### 6.5 Dependabot Configuration

**`.github/dependabot.yml`:**
- No URL changes needed (uses relative paths)
- Assignees remain `tomtom215`

### 6.6 Self-Hosted Runner Configuration

**`scripts/setup-runner-host.sh`:**
```bash
# Update service names
-actions.runner.tomtom215-map
+actions.runner.tomtom215-cartographus

# Update repository URLs
-https://github.com/tomtom215/cartographus
+https://github.com/tomtom215/cartographus
```

---

## 7. Phase 3: Documentation Updates

### 7.1 README.md Updates

| Section | Change |
|---------|--------|
| Badges | Update all badge URLs |
| Quick Start | Update Docker image |
| Clone URL | Update git clone command |
| License badge | Change from MIT to AGPL-3.0 |
| License section | Update license text |

**Badge changes:**
```markdown
[![Build Status](https://github.com/tomtom215/cartographus/actions/workflows/build-and-test.yml/badge.svg)](https://github.com/tomtom215/cartographus/actions)
[![License: AGPL-3.0](https://img.shields.io/badge/License-AGPL--3.0-blue.svg)](https://www.gnu.org/licenses/agpl-3.0)
[![GHCR](https://ghcr-badge.egpl.dev/tomtom215/cartographus/latest_tag?trim=major&label=GHCR)](https://github.com/tomtom215/cartographus/pkgs/container/cartographus)
[![Go Version](https://img.shields.io/github/go-mod-go-version/tomtom215/cartographus?label=Go)](https://github.com/tomtom215/cartographus/blob/main/go.mod)
```

### 7.2 CLAUDE.md Updates

- Update repository URL (line 7)
- Update any internal references

### 7.3 CONTRIBUTING.md Updates

| Line | Change |
|------|--------|
| 46 | Clone URL |
| 50 | Upstream remote URL |
| 261 | PR comparison URL |
| 407 | License reference (MIT -> AGPL) |

### 7.4 SECURITY.md Updates

| Line | Change |
|------|--------|
| 10 | Security advisories URL |
| 243-244 | Contact URLs |
| 267 | License reference |

### 7.5 Wiki Files (7 files)

All wiki files need URL updates:
- `wiki/_Sidebar.md`
- `wiki/Home.md`
- `wiki/Quick-Start.md`
- `wiki/README.md`
- `wiki/API-Reference.md`
- `wiki/Configuration.md`
- `wiki/FAQ.md`

**Key changes:**
- Docker image references
- GitHub URLs
- License mentions (MIT -> AGPL)

### 7.6 Deployment Documentation

| File | Updates Needed |
|------|----------------|
| `deploy/kubernetes/README.md` | GitHub URLs |
| `deploy/alertmanager/README.md` | Runbook URLs |
| `deploy/unraid/README.md` | GitHub URLs, Docker image |
| `deploy/prometheus/rules/cartographus.yml` | Runbook URLs |

### 7.7 Internal Documentation

Files in `docs/` directory:
- `docs/DEVELOPMENT.md`
- `docs/TROUBLESHOOTING.md`
- `docs/PRODUCTION_DEPLOYMENT.md`
- `docs/SELF_HOSTED_RUNNER.md`
- `docs/PATTERNS.md`
- And 20+ other docs

---

## 8. Phase 4: License Implementation

### 8.1 Replace LICENSE File

**Current:** MIT License
**New:** AGPL-3.0

```bash
# Download official AGPL-3.0 text
curl -o LICENSE https://www.gnu.org/licenses/agpl-3.0.txt
```

Or create `LICENSE` with:
- Full AGPL-3.0 text
- Copyright notice: `Copyright (c) 2025 Cartographus Contributors`

### 8.2 Add License Headers to Source Files

For AGPL-3.0 compliance, add headers to all source files:

**Go files (`.go`):**
```go
// Cartographus - Media Server Analytics Platform
// Copyright (C) 2025 Cartographus Contributors
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package ...
```

**TypeScript files (`.ts`):**
```typescript
/**
 * Cartographus - Media Server Analytics Platform
 * Copyright (C) 2025 Cartographus Contributors
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 */
```

**Automated header addition:**
```bash
# Use addlicense tool
go install github.com/google/addlicense@latest
addlicense -c "Cartographus Contributors" -l agpl -y 2025 .
```

### 8.3 Update License References

Update all documentation mentioning "MIT License":

| File | Line | Change |
|------|------|--------|
| `README.md` | 10, 38, 425-427 | MIT -> AGPL-3.0 |
| `CONTRIBUTING.md` | 407 | License reference |
| `SECURITY.md` | 267 | License reference |
| `wiki/FAQ.md` | 30 | License mention |
| `wiki/Home.md` | 191-193 | License section |

### 8.4 NOTICE File (Recommended for AGPL)

Create `NOTICE` file:

```text
Cartographus - Media Server Analytics Platform
Copyright 2025 Cartographus Contributors

This product includes software developed by:
- The DuckDB Authors (MIT License)
- The Go Authors (BSD License)
- The MapLibre Contributors (BSD-3-Clause License)
- And other open source contributors (see go.mod and package.json)

For a complete list of third-party licenses, see THIRD_PARTY_LICENSES.md
```

### 8.5 Third-Party Licenses File

Generate `THIRD_PARTY_LICENSES.md`:

```bash
# Go dependencies
go-licenses csv github.com/tomtom215/cartographus/... > go-licenses.csv

# npm dependencies
cd web && npx license-checker --csv > ../npm-licenses.csv
```

---

## 9. Phase 5: GitHub Repository Setup

### 9.1 Repository Creation

1. Create `github.com/tomtom215/cartographus` (if not already created)
2. Do NOT initialize with README, LICENSE, or .gitignore
3. Set visibility to **Private** initially (make public after verification)

### 9.2 Repository Settings

**General:**
- Description: "Media server analytics and geographic visualization platform"
- Website: (add if you have one)
- Topics: `media-server`, `analytics`, `plex`, `jellyfin`, `emby`, `self-hosted`, `duckdb`, `go`, `typescript`

**Features:**
- [x] Issues
- [x] Discussions (recommended for community)
- [x] Wiki
- [x] Projects (optional)
- [x] Preserve this repository (archive setting - leave unchecked)

### 9.3 Branch Protection Rules

**For `main` branch:**
- [x] Require pull request before merging
- [x] Require status checks (CI must pass)
- [x] Require conversation resolution
- [x] Require signed commits (recommended)
- [x] Include administrators

### 9.4 Security Settings

- [x] Enable Dependabot alerts
- [x] Enable Dependabot security updates
- [x] Enable secret scanning
- [x] Enable push protection

### 9.5 Issue Templates

Create `.github/ISSUE_TEMPLATE/`:

**bug_report.yml:**
```yaml
name: Bug Report
description: Report a bug in Cartographus
title: "[Bug]: "
labels: ["bug", "triage"]
body:
  - type: markdown
    attributes:
      value: |
        Thanks for taking the time to fill out this bug report!
  - type: textarea
    id: description
    attributes:
      label: Describe the bug
      description: A clear and concise description of what the bug is.
    validations:
      required: true
  - type: textarea
    id: reproduction
    attributes:
      label: Steps to reproduce
      description: Steps to reproduce the behavior
    validations:
      required: true
  - type: textarea
    id: expected
    attributes:
      label: Expected behavior
      description: What you expected to happen
    validations:
      required: true
  - type: dropdown
    id: deployment
    attributes:
      label: Deployment method
      options:
        - Docker Compose
        - Docker (standalone)
        - Kubernetes
        - Binary (direct)
        - Unraid
    validations:
      required: true
  - type: input
    id: version
    attributes:
      label: Cartographus version
      placeholder: "e.g., 1.0.0 or commit hash"
    validations:
      required: true
  - type: dropdown
    id: media-server
    attributes:
      label: Media server(s)
      multiple: true
      options:
        - Plex
        - Jellyfin
        - Emby
        - Tautulli
  - type: textarea
    id: logs
    attributes:
      label: Relevant logs
      description: Please copy and paste any relevant log output.
      render: shell
```

**feature_request.yml:**
```yaml
name: Feature Request
description: Suggest an idea for Cartographus
title: "[Feature]: "
labels: ["enhancement", "triage"]
body:
  - type: textarea
    id: problem
    attributes:
      label: Problem description
      description: What problem would this feature solve?
    validations:
      required: true
  - type: textarea
    id: solution
    attributes:
      label: Proposed solution
      description: How would you like to see this implemented?
    validations:
      required: true
  - type: textarea
    id: alternatives
    attributes:
      label: Alternatives considered
      description: Any alternative solutions you've considered?
```

### 9.6 Pull Request Template

Create `.github/PULL_REQUEST_TEMPLATE.md`:

```markdown
## Description
<!-- Describe your changes in detail -->

## Type of change
- [ ] Bug fix (non-breaking change that fixes an issue)
- [ ] New feature (non-breaking change that adds functionality)
- [ ] Breaking change (fix or feature that would cause existing functionality to change)
- [ ] Documentation update
- [ ] Refactoring (no functional changes)

## Related Issues
<!-- Link any related issues: Fixes #123, Relates to #456 -->

## Testing
<!-- Describe the tests you ran -->
- [ ] Unit tests pass (`go test -tags "wal,nats" -race ./...`)
- [ ] E2E tests pass (if applicable)
- [ ] Manual testing performed

## Checklist
- [ ] My code follows the project's style guidelines
- [ ] I have performed a self-review of my code
- [ ] I have commented my code where necessary
- [ ] I have updated documentation if needed
- [ ] My changes generate no new warnings
- [ ] I have added tests for my changes
```

### 9.7 Discussions Categories

Set up GitHub Discussions with categories:
- **Announcements** (maintainers only)
- **General** (Q&A)
- **Ideas** (feature suggestions)
- **Show and Tell** (user showcases)

---

## 10. Phase 6: CI/CD Migration

### 10.1 GitHub Actions Secrets

Transfer or recreate secrets in new repository:

| Secret | Purpose |
|--------|---------|
| `GITHUB_TOKEN` | Auto-provided |
| `GHCR_PAT` | Container registry push (if using PAT instead of GITHUB_TOKEN) |
| `CODECOV_TOKEN` | Code coverage uploads (if using Codecov) |

### 10.2 Self-Hosted Runner Migration

If using self-hosted runners:

1. Register new runner with `cartographus` repository
2. Update runner labels if needed
3. Update `scripts/setup-runner-host.sh` with new URLs
4. Test runner connectivity

### 10.3 Container Registry

**New registry path:** `ghcr.io/tomtom215/cartographus`

GitHub Container Registry will automatically work with the new repository name.

### 10.4 Workflow File Updates

All workflow files in `.github/workflows/` need:
- Repository URL updates (automatic via `${{ github.repository }}`)
- Any hardcoded `map` references changed to `cartographus`

### 10.5 Release Process

Update GoReleaser to publish to new repository:
- Releases will automatically go to the new repo
- Update changelog template URLs

---

## 11. Phase 7: Final Verification

### 11.1 Build Verification

```bash
# Clean build
rm -rf cartographus
go clean -modcache
go mod download
CGO_ENABLED=1 go build -tags "wal,nats" -v -o cartographus ./cmd/server

# Verify binary
./cartographus --version
```

### 11.2 Test Suite

```bash
# Unit tests
go test -tags "wal,nats" -race -cover ./...

# E2E tests
cd web && npm run test:e2e

# Integration tests
go test -tags "wal,nats,integration" -v ./internal/sync/... ./internal/import/...
```

### 11.3 Docker Build

```bash
# Build image with new tag
docker build -t ghcr.io/tomtom215/cartographus:test .

# Run container
docker run -d --name cartographus-test \
  -e AUTH_MODE=none \
  -p 3857:3857 \
  ghcr.io/tomtom215/cartographus:test

# Verify health
curl http://localhost:3857/api/v1/health

# Cleanup
docker stop cartographus-test && docker rm cartographus-test
```

### 11.4 Link Verification

Search for any remaining old references:

```bash
# Check for old repo URLs
grep -r "tomtom215/map" --include="*.go" --include="*.md" --include="*.yml" --include="*.yaml" --include="*.ts" --include="*.html" .

# Should return 0 results
```

### 11.5 License Verification

```bash
# Check license headers
grep -r "GNU Affero General Public License" --include="*.go" . | wc -l
# Should match number of Go files

# Check LICENSE file
head -20 LICENSE
# Should show AGPL-3.0 header
```

---

## 12. Post-Migration Tasks

### 12.1 Announcements

- [ ] Update any external links pointing to old repo
- [ ] Post announcement in Discussions (if enabled)
- [ ] Update any social media/forum profiles
- [ ] Notify any known users/contributors

### 12.2 Old Repository Handling

Options for `tomtom215/map`:
1. **Archive it** - Mark as read-only with redirect notice
2. **Delete it** - Remove entirely (loses stars/watchers)
3. **Keep as redirect** - Update README to point to new repo

**Recommended:** Archive with redirect notice in README:

```markdown
# This repository has moved

Cartographus has moved to a new home:

**https://github.com/tomtom215/cartographus**

Please update your bookmarks and remotes.
```

### 12.3 Documentation Site

If you have documentation hosted elsewhere:
- Update URLs to point to new repo
- Update any API documentation
- Update Docker Hub references (if applicable)

### 12.4 Container Registry Cleanup

- Delete old images from `ghcr.io/tomtom215/cartographus` (eventually)
- Set up redirect notice in old package description

---

## 13. Rollback Plan

If critical issues are discovered post-migration:

### Immediate Rollback (< 24 hours)

1. Revert code changes in new repo
2. Point documentation back to old repo
3. Update Docker compose examples

### Long-term Considerations

- Keep old repository archived (not deleted) for 6+ months
- Monitor for issues related to migration
- Have clear communication channel for migration issues

---

## 14. File Change Summary

### Files Requiring Changes

| Category | File Count | Change Type |
|----------|------------|-------------|
| Go files (imports) | 427 | Import paths |
| Documentation (MD) | 36 | URLs, license |
| Docker images | 21 | Image references |
| GitHub URLs | 46 | Non-Go files |
| Database paths | 35 | Go, YAML, MD, env |
| LICENSE | 1 | Complete replacement |
| License references | 6 | MIT -> AGPL |
| CI/CD workflows | 2 | Security fixes needed |
| **Total unique files** | ~536 | Multiple changes per file |

### Search/Replace Summary

| Find | Replace | File Types |
|------|---------|------------|
| `github.com/tomtom215/cartographus` | `github.com/tomtom215/cartographus` | All |
| `ghcr.io/tomtom215/cartographus` | `ghcr.io/tomtom215/cartographus` | YAML, MD |
| `tomtom215/map` | `tomtom215/cartographus` | Badges, URLs |
| `MIT License` | `AGPL-3.0 License` | MD, comments |
| `License-MIT` | `License-AGPL--3.0` | Badge URLs |
| `cartographus.duckdb` | `cartographus.duckdb` | Go, YAML, MD, env |
| `/data/cartographus.duckdb` | `/data/cartographus.duckdb` | Docker, docs |

### Database Path Migration

Users upgrading from the old database path will need to migrate:

```bash
# For existing users (add to release notes)
mv /data/cartographus.duckdb /data/cartographus.duckdb
```

**Files requiring database path updates:**
- `internal/config/config.go` (default path)
- `docker-compose.yml`
- `Dockerfile`
- `README.md`
- `wiki/Quick-Start.md`
- `wiki/Configuration.md`
- `deploy/kubernetes/deployment.yaml`
- `.env.example`

### Critical Path

Execute in this order to minimize issues:

1. **Dry-run analysis** - `./scripts/migration-dry-run.sh --output report.txt`
2. **go.mod** - Module declaration (must be first)
3. **Go imports** - All 427 .go files
4. **Regenerate Swagger** - After import changes
5. **Database paths** - Update 35 files with `cartographus.duckdb` references
6. **Documentation** - All .md files (36 files)
7. **CI/CD configs** - .github/workflows/ (security + URLs)
8. **LICENSE** - Replace file with AGPL-3.0
9. **Docker configs** - docker-compose.yml, Kubernetes, etc.
10. **Web frontend** - TypeScript files with URLs
11. **Final verification** - Build, test, lint, re-run dry-run (should show 0 issues)

---

## 15. CRITICAL: Self-Hosted Runner Security

**Risk Level: CRITICAL**

The current CI/CD configuration uses self-hosted GitHub Actions runners for workflows triggered by pull requests. **This is a severe security vulnerability for a public repository.**

### The Problem

```
Public Repository + Self-Hosted Runner + PR Trigger = Remote Code Execution
```

Anyone can:
1. Fork the repository
2. Submit a PR with malicious code
3. That code executes on YOUR infrastructure with Docker access

### Current Vulnerable Workflows

| Workflow | Trigger | Runner | Risk |
|----------|---------|--------|------|
| `_lint.yml` | push, **PR** | self-hosted | CRITICAL |
| `_test.yml` | push, **PR** | self-hosted | CRITICAL |
| `_build.yml` | push, **PR** | self-hosted | CRITICAL |
| `_e2e.yml` | push, **PR** | self-hosted | CRITICAL |

### Attack Vectors

1. **Supply Chain Poisoning**: PR code modifies build cache, poisons main branch builds
2. **Credential Theft**: Extract `GITHUB_TOKEN` with `packages:write` permission
3. **Container Registry Compromise**: Push malicious images to GHCR
4. **Infrastructure Pivot**: Gain shell access, enumerate network, establish persistence
5. **Cryptomining**: Run miners during long test suites

### Required Changes (BEFORE PUBLIC RELEASE)

#### 1. Split Workflows by Trust Level

```yaml
# Example: _lint.yml
jobs:
  lint-pr:
    if: github.event_name == 'pull_request'
    runs-on: ubuntu-latest  # GitHub-hosted for PRs (SAFE)

  lint-main:
    if: github.event_name == 'push'
    runs-on: [self-hosted, linux, x64]  # Self-hosted for main (TRUSTED)
```

#### 2. Remove Write Permissions from PR Workflows

```yaml
# For PR-triggered jobs
permissions:
  contents: read
  pull-requests: read
  # NO packages:write
  # NO id-token:write
```

#### 3. Disable Image Push for PRs

```yaml
jobs:
  build-docker:
    if: github.event_name == 'push' && github.ref == 'refs/heads/main'
    # Only push from main branch
```

### Recommended Hybrid Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    GitHub Repository                         │
└─────────────────────────────────────────────────────────────┘
                              │
        ┌─────────────────────┼─────────────────────┐
        │                     │                     │
        ▼                     ▼                     ▼
┌───────────────┐   ┌───────────────┐   ┌───────────────┐
│ Pull Request  │   │  Main Branch  │   │   Release     │
│  (Untrusted)  │   │   (Trusted)   │   │  (Critical)   │
└───────┬───────┘   └───────┬───────┘   └───────┬───────┘
        │                   │                   │
        ▼                   ▼                   ▼
┌───────────────┐   ┌───────────────┐   ┌───────────────┐
│ GitHub-Hosted │   │  Self-Hosted  │   │ GitHub-Hosted │
│   Runners     │   │   Runners     │   │   Runners     │
└───────────────┘   └───────────────┘   └───────────────┘
        │                   │                   │
        ▼                   ▼                   ▼
   - Lint              - Lint              - Build
   - Test              - Test              - Sign
   - Build (no push)   - Build + Push      - Release
   - Security scan     - E2E tests         - SBOM
```

### Trade-offs

| Aspect | Current | After Fix |
|--------|---------|-----------|
| PR build time | 3-5 min | 8-15 min |
| PR security | **CRITICAL** | Low |
| Main branch speed | Fast | Fast (unchanged) |
| Cost | Low | Low (same) |

### Implementation Checklist

- [ ] Update `_lint.yml` with conditional runner selection
- [ ] Update `_test.yml` with conditional runner selection
- [ ] Update `_build.yml` with conditional runner selection and PR push disable
- [ ] Update `_e2e.yml` to skip or use GitHub-hosted for PRs
- [ ] Remove `packages:write` from PR workflow permissions
- [ ] Remove `id-token:write` from PR workflow permissions
- [ ] Add branch protection rules requiring reviews
- [ ] Enable "Require approval for all outside collaborators" in Actions settings
- [ ] Test with a simulated external PR

### Full Analysis

See **[SELF_HOSTED_RUNNER_SECURITY.md](./SELF_HOSTED_RUNNER_SECURITY.md)** for:
- Detailed vulnerability analysis
- Attack scenario walkthroughs
- Complete workflow update examples
- SLSA compliance recommendations

---

## Appendix A: Automated Migration Script

```bash
#!/bin/bash
# migration-script.sh
# Run from repository root

set -e

OLD_REPO="github.com/tomtom215/cartographus"
NEW_REPO="github.com/tomtom215/cartographus"
OLD_IMAGE="ghcr.io/tomtom215/cartographus"
NEW_IMAGE="ghcr.io/tomtom215/cartographus"
OLD_DB="cartographus.duckdb"
NEW_DB="cartographus.duckdb"

echo "=== Cartographus Migration Script ==="

# Step 1: Update go.mod
echo "Updating go.mod..."
sed -i "s|module ${OLD_REPO}|module ${NEW_REPO}|g" go.mod

# Step 2: Update Go imports
echo "Updating Go imports..."
find . -name "*.go" -type f -exec sed -i "s|${OLD_REPO}|${NEW_REPO}|g" {} +

# Step 3: Update Docker images
echo "Updating Docker image references..."
find . -type f \( -name "*.yml" -o -name "*.yaml" -o -name "*.md" -o -name "*.xml" \) \
  -exec sed -i "s|${OLD_IMAGE}|${NEW_IMAGE}|g" {} +

# Step 4: Update GitHub URLs in all files
echo "Updating GitHub URLs..."
find . -type f \( -name "*.go" -o -name "*.md" -o -name "*.yml" -o -name "*.yaml" \
  -o -name "*.json" -o -name "*.ts" -o -name "*.html" -o -name "*.xml" -o -name "*.txt" \) \
  -exec sed -i "s|${OLD_REPO}|${NEW_REPO}|g" {} +

# Step 5: Update database path
echo "Updating database path references..."
find . -type f \( -name "*.go" -o -name "*.md" -o -name "*.yml" -o -name "*.yaml" \
  -o -name "*.env*" -o -name "Dockerfile" \) \
  -exec sed -i "s|${OLD_DB}|${NEW_DB}|g" {} +

# Step 6: Update license references
echo "Updating license references..."
find . -type f -name "*.md" -exec sed -i 's|MIT License|AGPL-3.0 License|g' {} +
find . -type f -name "*.md" -exec sed -i 's|License-MIT|License-AGPL--3.0|g' {} +

# Step 7: Verify
echo "Checking for remaining old references..."
REMAINING=$(grep -r "tomtom215/map" --include="*.go" --include="*.md" --include="*.yml" . 2>/dev/null | wc -l)
if [ "$REMAINING" -gt 0 ]; then
  echo "WARNING: Found $REMAINING remaining references to old repo"
  grep -r "tomtom215/map" --include="*.go" --include="*.md" --include="*.yml" . 2>/dev/null
else
  echo "SUCCESS: No remaining references found"
fi

# Check for old database path
DB_REMAINING=$(grep -r "map\.duckdb" --include="*.go" --include="*.yml" --include="*.md" . 2>/dev/null | wc -l)
if [ "$DB_REMAINING" -gt 0 ]; then
  echo "WARNING: Found $DB_REMAINING remaining database path references"
else
  echo "SUCCESS: No remaining database path references"
fi

# Step 8: Rebuild
echo "Rebuilding..."
go mod tidy
CGO_ENABLED=1 go build -tags "wal,nats" -v -o cartographus ./cmd/server

echo "=== Migration complete ==="
echo "Next steps:"
echo "1. Review changes with 'git diff'"
echo "2. Run tests: go test -tags 'wal,nats' -race ./..."
echo "3. Regenerate Swagger: swag init -g cmd/server/main.go -o docs/"
echo "4. Replace LICENSE file with AGPL-3.0"
echo "5. Add license headers to source files (see Appendix B)"
echo "6. Update CI/CD workflows for hybrid runner architecture (see Section 14)"
```

---

## Appendix B: License Header Script

```bash
#!/bin/bash
# add-license-headers.sh
# Adds AGPL-3.0 headers to source files

GO_HEADER='// Cartographus - Media Server Analytics Platform
// Copyright (C) 2025 Cartographus Contributors
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

'

# Add headers to Go files that don't have them
for file in $(find . -name "*.go" -type f); do
  if ! grep -q "GNU Affero General Public License" "$file"; then
    echo "Adding header to $file"
    echo "$GO_HEADER" | cat - "$file" > temp && mv temp "$file"
  fi
done
```

---

## Appendix C: User Upgrade Guide

This section documents what **existing users** need to do when upgrading from the `map` repository to `cartographus`.

### C.1 Database Migration Script

Users with existing data need to migrate their database files. Include this script in release notes:

```bash
#!/bin/bash
# upgrade-to-cartographus.sh
# Run this BEFORE starting Cartographus v2.0.0+

set -e

DATA_DIR="${DATA_DIR:-/data}"
OLD_DB="$DATA_DIR/cartographus.duckdb"
NEW_DB="$DATA_DIR/cartographus.duckdb"
OLD_WAL="$DATA_DIR/map.wal"
NEW_WAL="$DATA_DIR/cartographus.wal"

echo "=== Cartographus Database Migration ==="
echo "Data directory: $DATA_DIR"
echo ""

# Check if migration is needed
if [ -f "$NEW_DB" ]; then
    echo "✓ Already migrated: $NEW_DB exists"
    exit 0
fi

if [ ! -f "$OLD_DB" ]; then
    echo "✓ Fresh install: No existing database found"
    exit 0
fi

# Stop the service first
echo "⚠️  Please ensure Cartographus is stopped before proceeding!"
read -p "Press Enter to continue or Ctrl+C to cancel..."

# Migrate main database
echo "Migrating database: $OLD_DB -> $NEW_DB"
mv "$OLD_DB" "$NEW_DB"

# Migrate WAL directory if it exists
if [ -d "$OLD_WAL" ]; then
    echo "Migrating WAL: $OLD_WAL -> $NEW_WAL"
    mv "$OLD_WAL" "$NEW_WAL"
fi

# Migrate any .wal files
for wal_file in "$DATA_DIR"/cartographus.duckdb.wal*; do
    if [ -f "$wal_file" ]; then
        new_wal_file="${wal_file/cartographus.duckdb/cartographus.duckdb}"
        echo "Migrating: $wal_file -> $new_wal_file"
        mv "$wal_file" "$new_wal_file"
    fi
done

echo ""
echo "✓ Migration complete!"
echo ""
echo "You can now start Cartographus v2.0.0+"
```

### C.2 Docker Compose Migration

For users with existing Docker Compose deployments:

**Old configuration:**
```yaml
services:
  cartographus:
    image: ghcr.io/tomtom215/cartographus:latest
    volumes:
      - ./data:/data
      - map_data:/data  # Named volume example
```

**New configuration:**
```yaml
services:
  cartographus:
    image: ghcr.io/tomtom215/cartographus:latest
    volumes:
      - ./data:/data
      - cartographus_data:/data  # Renamed volume (optional)
```

**Named Volume Migration:**

If users have named volumes, they have two options:

1. **Option A: Rename the volume** (recommended for clean setups)
   ```bash
   # Stop containers
   docker-compose down

   # Create new volume and copy data
   docker volume create cartographus_data
   docker run --rm \
     -v map_data:/source:ro \
     -v cartographus_data:/dest \
     alpine cp -av /source/. /dest/

   # Update docker-compose.yml to use cartographus_data
   # Start with new config
   docker-compose up -d
   ```

2. **Option B: Keep existing volume** (simpler)
   ```yaml
   # Just update the image, keep the volume name
   services:
     cartographus:
       image: ghcr.io/tomtom215/cartographus:latest
       volumes:
         - map_data:/data  # Keep old volume name - it still works!
   ```

### C.3 Kubernetes Migration

For Kubernetes deployments, update the image reference:

```yaml
# Old
spec:
  containers:
    - name: cartographus
      image: ghcr.io/tomtom215/cartographus:v1.x.x

# New
spec:
  containers:
    - name: cartographus
      image: ghcr.io/tomtom215/cartographus:v2.0.0
```

**PersistentVolumeClaim data is preserved** - no action needed for existing PVCs.

### C.4 Environment Variable Changes

No environment variable changes are required. The `DATABASE_PATH` environment variable (if set) will continue to work. If not set, the new default is `/data/cartographus.duckdb`.

---

## Appendix D: Go Module Deprecation

If external projects import this module, add a deprecation notice to help them migrate.

### D.1 Old Module Deprecation Notice

After migration, update the old repository's `go.mod` (before archiving):

```go
// Deprecated: This module has moved to github.com/tomtom215/cartographus
//
// To migrate:
//   1. Update all imports from github.com/tomtom215/cartographus to github.com/tomtom215/cartographus
//   2. Run: go get github.com/tomtom215/cartographus@latest
//   3. Run: go mod tidy
//
// See https://github.com/tomtom215/cartographus for the new repository.
module github.com/tomtom215/cartographus

go 1.24.0

// Retract all versions to indicate this module is deprecated
retract (
    [v0.0.0, v1.99.99]
)
```

### D.2 Module Proxy Considerations

The Go module proxy (`proxy.golang.org`) will:
- Continue serving cached versions of `github.com/tomtom215/cartographus`
- Start serving new versions from `github.com/tomtom215/cartographus`
- Respect the `retract` directive to warn users

**Note:** Retracted versions are still installable but `go get` will show a warning.

### D.3 Consumer Migration Guide

For any projects that import this module, provide this migration guide:

```bash
# Step 1: Update go.mod
go get github.com/tomtom215/cartographus@latest

# Step 2: Update imports (use goimports or manually)
find . -name "*.go" -exec sed -i 's|github.com/tomtom215/cartographus|github.com/tomtom215/cartographus|g' {} +

# Step 3: Clean up
go mod tidy

# Step 4: Verify
go build ./...
```

---

## Appendix E: Migration Testing Procedure

Before running the migration on the actual repository, test it in isolation.

### E.1 Create Test Environment

```bash
# Clone to a test directory
git clone . ../map-migration-test
cd ../map-migration-test

# Create a test branch
git checkout -b test/migration-dry-run
```

### E.2 Run Dry-Run Analysis

```bash
# Verify current state
./scripts/migration-dry-run.sh --output pre-migration-report.txt --json

# Save the JSON for comparison
cp migration-dry-run.json pre-migration.json
```

### E.3 Execute Migration

```bash
# Run the safe replacement commands (from Section 2)
sed -i 's|module github.com/tomtom215/cartographus|module github.com/tomtom215/cartographus|' go.mod
find . -type f -name "*.go" -exec sed -i 's|github.com/tomtom215/cartographus|github.com/tomtom215/cartographus|g' {} +
# ... (run all safe commands)
```

### E.4 Verify Migration Success

```bash
# Re-run dry-run - should show 0 issues for safe patterns
./scripts/migration-dry-run.sh --output post-migration-report.txt --json

# Compare reports
diff pre-migration.json migration-dry-run.json

# Verify dangerous patterns were NOT touched
grep -r "cartographus\[" --include="*.go" . && echo "ERROR: Dangerous pattern replaced!" || echo "OK: No dangerous patterns touched"
grep -r "\.cartographus(" --include="*.ts" . && echo "ERROR: Dangerous pattern replaced!" || echo "OK: No dangerous patterns touched"

# Build test
source scripts/session-setup.sh
go build -tags "wal,nats" -o cartographus ./cmd/server

# Run tests
go test -tags "wal,nats" -race ./...

# E2E tests (if applicable)
cd web && npm run test:e2e
```

### E.5 Cleanup Test Environment

```bash
# After successful test, remove the test directory
cd ..
rm -rf map-migration-test
```

---

## Appendix F: CHANGELOG Entry Template

Add this to `CHANGELOG.md` for the release that includes the migration:

```markdown
## [2.0.0] - YYYY-MM-DD

### ⚠️ BREAKING CHANGES

This release migrates the project from `github.com/tomtom215/cartographus` to `github.com/tomtom215/cartographus`.

#### Repository Renamed

| Old | New |
|-----|-----|
| `github.com/tomtom215/cartographus` | `github.com/tomtom215/cartographus` |
| `ghcr.io/tomtom215/cartographus` | `ghcr.io/tomtom215/cartographus` |

#### Database Path Changed

| Old | New |
|-----|-----|
| `/data/cartographus.duckdb` | `/data/cartographus.duckdb` |

**Migration Required:** Run the migration script before starting v2.0.0:

```bash
# For Docker users
docker run --rm -v /path/to/data:/data ghcr.io/tomtom215/cartographus:2.0.0 /scripts/upgrade-to-cartographus.sh

# For binary users
mv /data/cartographus.duckdb /data/cartographus.duckdb
```

#### License Changed

The license has changed from MIT to **AGPL-3.0**. This means:
- You can still use Cartographus freely for personal/commercial use
- If you modify and distribute Cartographus, you must release your changes under AGPL-3.0
- If you run a modified version as a network service, you must provide source code to users

See [LICENSE](./LICENSE) for full terms.

### Changed
- Repository renamed from `map` to `cartographus`
- Docker image renamed from `ghcr.io/tomtom215/cartographus` to `ghcr.io/tomtom215/cartographus`
- Database default path changed from `/data/cartographus.duckdb` to `/data/cartographus.duckdb`
- License changed from MIT to AGPL-3.0

### Migration Guide
See [docs/MIGRATION_PLAN.md](./docs/MIGRATION_PLAN.md) for detailed migration instructions.
```

---

## Appendix G: Post-Migration Monitoring

After the migration goes live, monitor for issues.

### G.1 Monitoring Checklist

- [ ] **GitHub Issues**: Watch for issues mentioning "import", "module not found", or "map"
- [ ] **Docker Hub/GHCR**: Monitor download stats for old vs new image
- [ ] **Go Module Proxy**: Check `pkg.go.dev/github.com/tomtom215/cartographus` is indexed
- [ ] **Search Engines**: Verify new repo appears in search results
- [ ] **External Links**: Use a link checker on documentation

### G.2 Common Issues and Resolutions

| Issue | Resolution |
|-------|------------|
| "module not found" | User needs to update import paths |
| "database not found" | User needs to run migration script |
| "image not found" | Update Docker image reference |
| Old links in search results | Wait for re-indexing, add redirects |

### G.3 GitHub Repository Redirect

After archiving the old repository, update its README:

```markdown
# ⚠️ This repository has moved

**Cartographus** has moved to a new home:

## 👉 https://github.com/tomtom215/cartographus

### Why the move?

The project has been renamed from "map" to "cartographus" to avoid confusion with common programming terms.

### How to update

**For Git users:**
```bash
git remote set-url origin https://github.com/tomtom215/cartographus.git
```

**For Docker users:**
```bash
# Old
docker pull ghcr.io/tomtom215/cartographus:latest

# New
docker pull ghcr.io/tomtom215/cartographus:latest
```

**For Go module users:**
```bash
go get github.com/tomtom215/cartographus@latest
```

---

*This repository is archived and will not receive updates.*
```

---

**Document Version:** 1.5
**Last Updated:** 2026-01-12
**Author:** AI Assistant
**Review Status:** Decisions Finalized - Ready for Implementation

### Changelog

- **v1.5** (2026-01-12): Added Appendices C-G covering user upgrade guide, Go module deprecation, migration testing procedure, CHANGELOG template, and post-migration monitoring. Note: Since this is a pre-release stealth repo with no existing users, the user migration appendices serve as future reference documentation.
- **v1.4** (2026-01-12): Added CRITICAL "Safe Replacement Patterns" section with comprehensive analysis of dangerous vs safe replacement patterns, verification commands, and explicit warnings to prevent catastrophic codebase damage from naive find-and-replace operations
- **v1.3** (2026-01-09): Enhanced dry-run script with Section 8 (additional checks: package.json, Makefile, GoReleaser, GitHub templates, large files, binary files, Swagger, license headers, CHANGELOG), added JSON output option for automation, environment snapshot for reproducibility
- **v1.2** (2026-01-09): Added dry-run script documentation, updated file counts from actual analysis (~536 files), added dry-run to critical path
- **v1.1** (2026-01-09): Added decisions (AGPL-3.0, database path, fresh history), self-hosted runner security analysis, database path migration steps
- **v1.0** (2026-01-09): Initial comprehensive migration plan
