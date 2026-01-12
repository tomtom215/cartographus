# Self-Hosted Runner Security Analysis for Public Repository

**Status**: CRITICAL - Must Address Before Public Release
**Created**: 2026-01-09
**Risk Level**: CRITICAL

---

## Executive Summary

The current CI/CD configuration uses self-hosted GitHub Actions runners for the majority of workflows, including those triggered by pull requests. **This is incompatible with a public repository** because external contributors can submit PRs that execute arbitrary code on the self-hosted infrastructure.

**Bottom Line**: Before making the repository public, all PR-triggered workflows MUST be moved to GitHub-hosted runners OR the repository must not accept external PRs.

---

## Table of Contents

1. [Current Configuration](#1-current-configuration)
2. [Critical Vulnerabilities](#2-critical-vulnerabilities)
3. [Attack Scenarios](#3-attack-scenarios)
4. [Recommendations](#4-recommendations)
5. [Implementation Plan](#5-implementation-plan)
6. [Hybrid Architecture](#6-hybrid-architecture)

---

## 1. Current Configuration

### Workflows Using Self-Hosted Runners

| Workflow | Trigger | Runner | Risk |
|----------|---------|--------|------|
| `_lint.yml` | push, PR | `[self-hosted, linux, x64]` | **CRITICAL** |
| `_test.yml` | push, PR | `[self-hosted, linux, x64, big-builder]` | **CRITICAL** |
| `_build.yml` | push, PR | `[self-hosted, linux, x64]` | **CRITICAL** |
| `_e2e.yml` | push, PR | `[self-hosted, linux, x64, big-builder]` | **CRITICAL** |
| `build-and-test.yml` | push, PR | `[self-hosted, linux, x64]` | **CRITICAL** |

### Workflows Using GitHub-Hosted Runners (Safe)

| Workflow | Trigger | Runner |
|----------|---------|--------|
| `_security.yml` | push, PR | `ubuntu-latest` |
| `_codeql.yml` | push, PR | `ubuntu-latest` |
| `release.yml` | tags | `ubuntu-latest` |

### Why This Matters

```
Public Repository + Self-Hosted Runner + PR Trigger = Remote Code Execution
```

Anyone can:
1. Fork the repository
2. Submit a PR with malicious code
3. That code executes on YOUR infrastructure with:
   - Docker daemon access
   - Registry credentials
   - Persistent cache access
   - Network access to your infrastructure

---

## 2. Critical Vulnerabilities

### 2.1 Arbitrary Code Execution

**Current state**: PR workflows run on self-hosted runners

```yaml
# build-and-test.yml (VULNERABLE)
on:
  pull_request:
    branches: [main, develop]

jobs:
  lint:
    runs-on: [self-hosted, linux, x64]  # Attacker code runs here
```

**Impact**: Attacker submits PR, their code runs on your server with full user privileges.

### 2.2 Docker Daemon Access

**Current state**: Runner user is in the docker group

```bash
# From setup-runner-host.sh
usermod -aG docker "$RUNNER_USER"
```

**Impact**: Malicious PR can:
- Run arbitrary containers with `--privileged`
- Mount host filesystem (`-v /:/host`)
- Access Docker socket
- Push to container registries

### 2.3 Persistent Cache Poisoning

**Current state**: Caches persist between runs

| Cache | Location | Risk |
|-------|----------|------|
| Go modules | `~/.go/pkg/mod` | Supply chain injection |
| Docker layers | `/tmp/.buildx-cache` | Malicious base images |
| DuckDB extensions | `~/.duckdb/extensions/` | Binary injection |
| npm cache | `~/.npm` | Dependency confusion |

**Attack**: PR poisons cache, main branch build uses poisoned cache, release contains malware.

### 2.4 Credential Exposure

**Current state**: Workflows have elevated permissions

```yaml
# _build.yml
permissions:
  packages: write      # Push to GHCR
  id-token: write      # OIDC signing
  security-events: write
```

**Impact**: Malicious PR can extract tokens and:
- Push malicious images to `ghcr.io/tomtom215/cartographus`
- Sign containers with your OIDC identity
- Publish fake security advisories

### 2.5 Container Signing Compromise

**Current state**: Cosign signing happens on self-hosted runners

```yaml
# _build.yml - security-sbom job
- name: Sign container image
  uses: sigstore/cosign-installer@v3
```

**Impact**: If PR code runs on the same infrastructure, signatures can be forged.

---

## 3. Attack Scenarios

### Scenario A: Supply Chain Attack

```
1. Attacker forks repository
2. Submits PR modifying Dockerfile or build scripts
3. PR workflow runs on self-hosted runner
4. Malicious code injects into Docker layer cache
5. Maintainer merges unrelated PR to main
6. Main branch build uses poisoned cache
7. Release contains backdoored binary
8. Users install compromised software
```

**Difficulty**: Low (requires only a GitHub account)
**Impact**: Critical (all users compromised)

### Scenario B: GHCR Credential Theft

```
1. Attacker submits PR with code that extracts GITHUB_TOKEN
2. Token has packages:write permission
3. Attacker pushes malicious image to ghcr.io/tomtom215/cartographus:latest
4. Image is signed with legitimate OIDC identity
5. Users pull "latest" tag, get malicious image
```

**Difficulty**: Medium (requires token extraction technique)
**Impact**: Critical (users get malicious images)

### Scenario C: Infrastructure Pivot

```
1. Attacker submits PR with reverse shell payload
2. Gains shell access to self-hosted runner
3. Enumerates network (what else is on this server/network?)
4. Pivots to other systems accessible from runner
5. Establishes persistence
```

**Difficulty**: Low (standard pentesting techniques)
**Impact**: Critical (infrastructure compromise)

### Scenario D: Cryptomining/Resource Abuse

```
1. Attacker submits PR with hidden cryptominer
2. Miner runs during long test suite
3. Your electricity/compute costs increase
4. May go unnoticed for extended periods
```

**Difficulty**: Very Low
**Impact**: Medium (financial, reputational)

---

## 4. Recommendations

### Option 1: Hybrid Architecture (RECOMMENDED)

Use different runners for different trust levels:

| Trust Level | Trigger | Runner | Use Cases |
|-------------|---------|--------|-----------|
| Untrusted | `pull_request` | GitHub-hosted | Lint, test, build (no push) |
| Trusted | `push` to main | Self-hosted | Full build, cache, push images |
| Release | `tag` | GitHub-hosted | Signing, release artifacts |

**Pros**: Maintains performance for trusted builds, secure for PRs
**Cons**: PR feedback is slower, some cache benefits lost

### Option 2: All GitHub-Hosted

Move everything to GitHub-hosted runners.

**Pros**: Simplest security model, no infrastructure to maintain
**Cons**:
- Slower builds (no persistent cache)
- GitHub Actions minutes cost
- Less control over environment

### Option 3: Contributor-Only Model

Don't accept external PRs - require contributors to be added to the organization first.

**Pros**: Self-hosted security maintained
**Cons**:
- Higher barrier to contribution
- Slower community growth
- Still vulnerable to compromised contributor accounts

### Option 4: Ephemeral Runners

Use ephemeral/disposable runners that are destroyed after each job.

**Pros**: No persistent state to poison
**Cons**:
- Complex infrastructure (Kubernetes, terraform)
- Slower (no cache)
- Still has runtime risks

---

## 5. Implementation Plan

### Phase 1: Immediate (Before Public Release)

#### 1.1 Split Workflow Triggers

Create separate jobs for PRs vs main branch:

```yaml
# _lint.yml - UPDATED
name: Lint

on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main, develop]

jobs:
  lint-pr:
    if: github.event_name == 'pull_request'
    runs-on: ubuntu-latest  # GitHub-hosted for PRs
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.24'
      # ... lint steps

  lint-main:
    if: github.event_name == 'push'
    runs-on: [self-hosted, linux, x64]  # Self-hosted for main
    steps:
      # ... same lint steps with caching benefits
```

#### 1.2 Remove Write Permissions from PR Workflows

```yaml
# For PR-triggered workflows
permissions:
  contents: read
  pull-requests: read
  # NO packages:write
  # NO id-token:write
```

#### 1.3 Disable PR Builds That Push Images

```yaml
# _build.yml
jobs:
  build-docker:
    # Only push images from main branch
    if: github.event_name == 'push' && github.ref == 'refs/heads/main'
```

### Phase 2: Before First Public Release

#### 2.1 Implement Cache Isolation

```bash
# Clear caches at start of PR workflows
- name: Clear caches (PR security)
  if: github.event_name == 'pull_request'
  run: |
    rm -rf ~/.cache/go-build
    rm -rf ~/go/pkg/mod
    docker system prune -af
```

#### 2.2 Add Required Reviewers for PRs

Repository Settings > Branches > main:
- [x] Require pull request reviews (1+)
- [x] Dismiss stale reviews on new commits
- [x] Require review from code owners

#### 2.3 Enable GitHub Security Features

- [x] Dependabot alerts
- [x] Secret scanning
- [x] Push protection
- [x] Code scanning (CodeQL)

### Phase 3: Post-Release Hardening

#### 3.1 Implement SLSA Level 3

- Use organization-owned runners
- Implement build provenance
- Cryptographic verification of artifacts

#### 3.2 Container Hardening

- Rootless Docker on runners
- Network isolation
- seccomp/AppArmor profiles

---

## 6. Hybrid Architecture

### Recommended Final State

```
                     ┌─────────────────────────────────────────┐
                     │           GitHub Repository             │
                     │        (tomtom215/cartographus)         │
                     └─────────────────────────────────────────┘
                                        │
              ┌─────────────────────────┼─────────────────────────┐
              │                         │                         │
              ▼                         ▼                         ▼
    ┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
    │  Pull Request   │     │   Main Branch   │     │    Release      │
    │   (Untrusted)   │     │   (Trusted)     │     │   (Critical)    │
    └────────┬────────┘     └────────┬────────┘     └────────┬────────┘
             │                       │                       │
             ▼                       ▼                       ▼
    ┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
    │ GitHub-Hosted   │     │  Self-Hosted    │     │ GitHub-Hosted   │
    │    Runners      │     │    Runners      │     │    Runners      │
    └────────┬────────┘     └────────┬────────┘     └────────┬────────┘
             │                       │                       │
             ▼                       ▼                       ▼
    ┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
    │  - Lint         │     │  - Lint         │     │  - Build        │
    │  - Test         │     │  - Test         │     │  - Sign         │
    │  - Build (no    │     │  - Build        │     │  - Release      │
    │    push)        │     │  - Push images  │     │  - SBOM         │
    │  - Security     │     │  - E2E tests    │     │  - Attestation  │
    │    scan         │     │  - Cached       │     │                 │
    └─────────────────┘     └─────────────────┘     └─────────────────┘
             │                       │                       │
             │                       │                       │
             ▼                       ▼                       ▼
         No artifacts           Push to GHCR          Release to
         produced               (dev tags)            GitHub/GHCR
```

### Workflow Configuration Summary

| Workflow | PR Trigger | Main Trigger | Release Trigger |
|----------|------------|--------------|-----------------|
| `_lint.yml` | GH-hosted | Self-hosted | N/A |
| `_test.yml` | GH-hosted | Self-hosted | N/A |
| `_build.yml` | GH-hosted (no push) | Self-hosted (push) | GH-hosted |
| `_e2e.yml` | Skip or GH-hosted | Self-hosted | N/A |
| `_security.yml` | GH-hosted | GH-hosted | GH-hosted |
| `release.yml` | N/A | N/A | GH-hosted |

### Trade-offs

| Aspect | Current | Proposed Hybrid |
|--------|---------|-----------------|
| PR build time | ~3-5 min | ~8-15 min |
| PR security | CRITICAL risk | LOW risk |
| Main branch speed | Fast (cached) | Fast (cached) |
| Release security | MEDIUM risk | LOW risk |
| Infrastructure cost | Low | Low (same runners) |
| Complexity | Simple | Moderate |

---

## 7. Required Workflow Changes

### Files to Modify

| File | Changes Required |
|------|-----------------|
| `.github/workflows/_lint.yml` | Split into PR/main jobs |
| `.github/workflows/_test.yml` | Split into PR/main jobs |
| `.github/workflows/_build.yml` | Conditional runner selection, disable PR push |
| `.github/workflows/_e2e.yml` | Skip for PRs or use GH-hosted |
| `.github/workflows/build-and-test.yml` | Update job dependencies |

### Example: Updated `_lint.yml`

```yaml
name: Lint

on:
  workflow_call:
    inputs:
      is_pr:
        description: 'Whether this is a PR build'
        required: false
        type: boolean
        default: false

jobs:
  lint:
    name: Go and TypeScript Lint
    runs-on: ${{ inputs.is_pr && 'ubuntu-latest' || fromJSON('["self-hosted", "linux", "x64"]') }}

    permissions:
      contents: read
      # No write permissions for PRs

    steps:
      - uses: actions/checkout@v4

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.24'
          cache: ${{ !inputs.is_pr }}  # Only cache for main branch

      - name: Setup DuckDB extensions (PR only)
        if: inputs.is_pr
        run: |
          # Download extensions fresh for PRs (no cached extensions)
          ./scripts/setup-duckdb-extensions.sh

      - name: Run golangci-lint
        uses: golangci/golangci-lint-action@v6
        with:
          version: v1.64.8
```

---

## 8. Checklist for Public Release

### Security Requirements (MUST)

- [ ] All `pull_request` triggered workflows use GitHub-hosted runners
- [ ] No write permissions (packages, security-events) for PR workflows
- [ ] Image push disabled for PR builds
- [ ] Container signing only on GitHub-hosted runners
- [ ] Branch protection rules enabled (required reviews)
- [ ] Secret scanning enabled
- [ ] Dependabot enabled

### Recommended (SHOULD)

- [ ] CODEOWNERS file configured
- [ ] Security policy (SECURITY.md) updated
- [ ] Signed commits required for main branch
- [ ] Audit logging enabled
- [ ] Runner hardening (AppArmor, seccomp)

### Nice to Have (MAY)

- [ ] SLSA Level 3 compliance
- [ ] Hermetic builds
- [ ] Ephemeral runners
- [ ] Network segmentation for runners

---

## 9. Conclusion

**The current configuration is secure for a private repository but becomes critically vulnerable when made public.**

The recommended approach is the **Hybrid Architecture**:
1. GitHub-hosted runners for all PR (untrusted) workflows
2. Self-hosted runners for main branch (trusted) workflows
3. GitHub-hosted runners for release (critical) workflows

This maintains the performance benefits of self-hosted runners for trusted code while protecting against supply chain attacks from external contributors.

**Action Required**: Implement Phase 1 changes BEFORE making the repository public.

---

## References

- [GitHub Docs: Self-hosted runner security](https://docs.github.com/en/actions/hosting-your-own-runners/managing-self-hosted-runners/about-self-hosted-runners#self-hosted-runner-security)
- [SLSA Framework](https://slsa.dev/)
- [Sigstore/Cosign](https://docs.sigstore.dev/)
- [GitHub Actions Security Hardening](https://docs.github.com/en/actions/security-guides/security-hardening-for-github-actions)
