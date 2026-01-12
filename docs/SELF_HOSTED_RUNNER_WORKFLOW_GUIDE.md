# Self-Hosted Runner Workflow Configuration Guide

This document provides specific recommendations for which workflows should use self-hosted runners and how to configure them.

**Last Verified**: 2026-01-11
**Related**: [SELF_HOSTED_RUNNER.md](./SELF_HOSTED_RUNNER.md)

---

## Overview

Not all workflows benefit equally from self-hosted runners. This guide analyzes each workflow and provides recommendations based on:

- Build time and resource usage
- Frequency of execution
- Security requirements
- Caching benefits
- Cost-effectiveness

---

## Workflow Analysis

### High Priority for Self-Hosted

These workflows will see the most significant improvements on self-hosted runners:

#### 1. `_build.yml` - Docker Multi-Platform Builds

**Current**: `runs-on: ubuntu-latest`
**Recommendation**: `runs-on: self-hosted`

**Why**:
- Multi-platform Docker builds (amd64, arm64)
- Heavy use of Docker layer caching
- Long build times (5-10 minutes per platform)
- Runs on every PR and push to main

**Benefits**:
- 50-70% faster builds with persistent Docker cache
- No cold starts
- Reduced GitHub Actions minutes consumption

**Configuration**:

```yaml
# .github/workflows/_build.yml
jobs:
  build-docker:
    name: Build ${{ matrix.platform }} Docker Image
    runs-on: self-hosted  # Changed from ubuntu-latest
    needs: build-frontend
    strategy:
      fail-fast: false
      matrix:
        platform:
          - linux/amd64
          - linux/arm64
```

**Estimated Impact**: Very High (10+ minute savings per build)

---

#### 2. `_test.yml` - Unit and Integration Tests

**Current**: `runs-on: ubuntu-latest`
**Recommendation**: `runs-on: self-hosted`

**Why**:
- Runs on every PR and push (highest frequency)
- DuckDB extension setup takes time on cold runners
- Go module downloads benefit from persistent cache
- Tests with CGO require build tools

**Benefits**:
- Faster Go module resolution (cached)
- DuckDB extensions pre-installed
- Faster test execution with warm CPU cache

**Configuration**:

```yaml
# .github/workflows/_test.yml
jobs:
  test-backend:
    name: Test Go Code
    runs-on: self-hosted  # Changed from ubuntu-latest
    env:
      GOTOOLCHAIN: local
      GODEBUG: netdns=cgo
      CGO_ENABLED: "1"
```

**Estimated Impact**: High (2-3 minute savings per test run)

---

#### 3. `_e2e.yml` - Playwright E2E Tests

**Current**: `runs-on: ubuntu-latest`
**Recommendation**: `runs-on: self-hosted`

**Why**:
- 338 E2E tests with Playwright
- Docker container orchestration
- Playwright browser downloads
- High CPU/memory usage

**Benefits**:
- Playwright browsers pre-installed
- Docker image layer caching
- Faster browser startup times
- Better parallelization

**Configuration**:

```yaml
# .github/workflows/_e2e.yml
jobs:
  integration-test:
    name: Integration Tests
    runs-on: self-hosted  # Changed from ubuntu-latest

  e2e-tests:
    name: E2E Tests (Playwright)
    runs-on: self-hosted  # Changed from ubuntu-latest

  ui-screenshots:
    name: Capture UI Screenshots
    runs-on: self-hosted  # Changed from ubuntu-latest
```

**Estimated Impact**: Very High (5-8 minute savings, especially for screenshot capture)

---

#### 4. `build-binaries.yml` - Cross-Platform Binary Builds

**Current**: `runs-on: ubuntu-latest`
**Recommendation**: `runs-on: self-hosted`

**Why**:
- OSXCross compilation (very resource-intensive)
- Cross-compilation for multiple platforms
- Long build times (10-15 minutes)
- Heavy CPU and disk usage

**Benefits**:
- OSXCross can be pre-built and cached
- Cross-compilation toolchains pre-installed
- Faster Go builds with cached modules
- Persistent build artifacts

**Configuration**:

```yaml
# .github/workflows/build-binaries.yml
jobs:
  build-binaries:
    name: Build Multi-Platform Binaries
    runs-on: self-hosted  # Changed from ubuntu-latest
    permissions:
      contents: write
      packages: read
```

**Note**: The OSXCross setup (60+ seconds) can be done once and cached on the self-hosted runner.

**Estimated Impact**: Very High (5-7 minute savings per build)

---

#### 5. `_analysis.yml` - Code Coverage and Profiling

**Current**: `runs-on: ubuntu-latest`
**Recommendation**: `runs-on: self-hosted`

**Why**:
- Heavy computation (coverage, profiling, benchmarks)
- Only runs on main branch pushes
- Long execution time (15-20 minutes)
- Benefits from faster CPU

**Benefits**:
- Faster benchmark execution
- Better profiling accuracy
- Cached Go modules and tools

**Configuration**:

```yaml
# .github/workflows/_analysis.yml
jobs:
  coverage-analysis:
    name: Code Coverage Analysis
    runs-on: self-hosted  # Changed from ubuntu-latest

  code-metrics:
    name: Code Metrics Analysis
    runs-on: self-hosted  # Changed from ubuntu-latest

  profile-backend:
    name: Profile Go Code
    runs-on: self-hosted  # Changed from ubuntu-latest
```

**Estimated Impact**: High (3-5 minute savings)

---

### Keep on GitHub-Hosted

These workflows should remain on GitHub-hosted runners:

#### 1. `_security.yml` - Security Scanning

**Current**: `runs-on: ubuntu-latest`
**Recommendation**: Keep `runs-on: ubuntu-latest`

**Why**:
- Security scanning should use trusted infrastructure
- SARIF uploads work better on GitHub-hosted
- Dependency scanning requires GitHub API access
- Trust and audit trail considerations

**Keep as-is**: No changes needed.

---

#### 2. `_codeql.yml` - CodeQL Analysis

**Current**: `runs-on: ubuntu-latest`
**Recommendation**: Keep `runs-on: ubuntu-latest`

**Why**:
- CodeQL requires GitHub's infrastructure
- SARIF results upload to Security tab
- OIDC authentication works best on GitHub-hosted
- Audit and compliance requirements

**Keep as-is**: No changes needed.

---

#### 3. `release.yml` - Release Publishing

**Current**: `runs-on: ubuntu-latest`
**Recommendation**: Keep `runs-on: ubuntu-latest`

**Why**:
- Cosign keyless signing requires GitHub OIDC
- Security and trust considerations
- Release artifacts should come from trusted runners
- Audit trail for releases

**Keep as-is**: No changes needed.

---

### Hybrid Approach (Advanced)

For workflows with multiple jobs, consider a hybrid approach:

#### `_lint.yml` - Linting

**Current**: `runs-on: ubuntu-latest`
**Recommendation**: Hybrid (backend on self-hosted, frontend on GitHub)

**Why**:
- Backend linting (golangci-lint) is CPU-intensive
- Frontend linting (TypeScript) is fast and lightweight
- Split based on resource requirements

**Configuration**:

```yaml
# .github/workflows/_lint.yml
jobs:
  lint-backend:
    name: Lint Go Code
    runs-on: self-hosted  # Changed from ubuntu-latest for Go linting
    env:
      GOTOOLCHAIN: local
      CGO_ENABLED: "1"

  lint-frontend:
    name: Lint Frontend Code
    runs-on: ubuntu-latest  # Keep GitHub-hosted for TypeScript (fast)
```

**Estimated Impact**: Moderate (1-2 minute savings on backend linting)

---

## Migration Strategy

### Phase 1: High-Value Workflows (Week 1)

Migrate workflows with highest impact first:

1. `_build.yml` - Docker builds
2. `_test.yml` - Unit tests
3. `build-binaries.yml` - Binary builds

**Expected Savings**: 15-20 minutes per CI run

### Phase 2: E2E and Analysis (Week 2)

After verifying Phase 1 stability:

4. `_e2e.yml` - E2E tests
5. `_analysis.yml` - Coverage and profiling

**Expected Savings**: Additional 8-12 minutes per CI run

### Phase 3: Optimization (Week 3+)

Fine-tune and optimize:

6. `_lint.yml` - Hybrid approach
7. Monitor and adjust concurrency limits
8. Optimize cache strategies

---

## Example Workflow Modifications

### Before: GitHub-Hosted

```yaml
name: Build

on:
  workflow_call:
    inputs:
      push-images:
        type: boolean
        default: false

jobs:
  build-docker:
    name: Build Docker Image
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v6

      - name: Build image
        uses: docker/build-push-action@v6
        with:
          context: .
          push: ${{ inputs.push-images }}
```

### After: Self-Hosted

```yaml
name: Build

on:
  workflow_call:
    inputs:
      push-images:
        type: boolean
        default: false

jobs:
  build-docker:
    name: Build Docker Image
    runs-on: self-hosted  # <-- Changed
    steps:
      - name: Checkout
        uses: actions/checkout@v6

      - name: Build image
        uses: docker/build-push-action@v6
        with:
          context: .
          push: ${{ inputs.push-images }}
          cache-from: type=local,src=/tmp/.buildx-cache  # Optional: local cache
          cache-to: type=local,dest=/tmp/.buildx-cache-new  # Optional: local cache
```

---

## Performance Expectations

### Build Times: Before vs After

| Workflow | GitHub-Hosted | Self-Hosted | Savings |
|----------|---------------|-------------|---------|
| `_build.yml` (both platforms) | 12-15 min | 5-7 min | 7-8 min |
| `_test.yml` (all tests) | 8-10 min | 5-6 min | 3-4 min |
| `_e2e.yml` (all tests) | 15-18 min | 8-10 min | 7-8 min |
| `build-binaries.yml` | 20-25 min | 13-15 min | 7-10 min |
| `_analysis.yml` (profiling) | 18-22 min | 12-15 min | 6-7 min |
| **Total per full CI run** | **73-90 min** | **43-53 min** | **30-37 min** |

**Note**: Actual times depend on hardware specs. These estimates assume:
- Self-hosted: 8-core CPU, 32GB RAM, SSD
- GitHub-hosted: 2-core CPU, 7GB RAM, SSD

---

## Concurrency Considerations

### Limiting Concurrent Jobs

To prevent resource exhaustion on a single runner:

```yaml
# In each workflow file
concurrency:
  group: ${{ github.workflow }}-${{ github.ref }}
  cancel-in-progress: true  # Cancel older runs
```

### Runner Groups (Enterprise/Organizations)

For multiple runners, use runner groups:

```yaml
jobs:
  build-docker:
    runs-on: [self-hosted, heavy-build]  # Use 'heavy-build' runner group

  lint-frontend:
    runs-on: [self-hosted, light-compute]  # Use 'light-compute' runner group
```

---

## Monitoring and Alerting

### Key Metrics to Track

1. **Job Duration**: Compare before/after migration
2. **Queue Time**: How long jobs wait for available runners
3. **Failure Rate**: Track if self-hosted runners have higher failures
4. **Disk Usage**: Monitor `/home/github-runner/_work`
5. **Docker Cache Size**: Keep under 50GB

### Monitoring Commands

```bash
# Check runner status
sudo systemctl status actions.runner.tomtom215-map.*.service

# Monitor job execution
sudo journalctl -u actions.runner.tomtom215-map.*.service -f

# Check disk usage
df -h
docker system df

# Monitor resource usage during builds
htop  # or top
```

---

## Rollback Plan

If self-hosted runners cause issues:

1. **Immediate**: Change `runs-on: self-hosted` back to `runs-on: ubuntu-latest`
2. **Commit and push** to restore GitHub-hosted execution
3. **Investigate** logs and troubleshoot
4. **Re-enable** once issues are resolved

**Rollback time**: < 5 minutes (just change YAML and commit)

---

## Cost Analysis

### GitHub Actions Minutes

Assuming 100 CI runs per month:

**Before (GitHub-hosted)**:
- Average run time: 80 minutes
- Monthly minutes: 100 runs × 80 min = 8,000 minutes
- Cost: $0.008/min for private repos = $64/month

**After (self-hosted)**:
- Average run time: 48 minutes (40% faster)
- GitHub minutes: 0 (self-hosted)
- Cost: $0/month for GitHub Actions

**Infrastructure costs** (self-hosted runner):
- VPS/Cloud VM: $40-80/month (8-core, 32GB RAM)
- Or: Existing hardware (no additional cost)

**Net savings**: $0-24/month + 40% faster CI

---

## Frequently Asked Questions

### Q: Can I use both self-hosted and GitHub-hosted runners?

Yes! Use a matrix strategy:

```yaml
jobs:
  test:
    runs-on: ${{ matrix.runner }}
    strategy:
      matrix:
        runner: [self-hosted, ubuntu-latest]
```

### Q: What if my self-hosted runner goes offline?

Jobs will queue until the runner comes back online or timeout (default: 6 hours). Have a monitoring system to alert when the runner is offline.

### Q: Do I need multiple runners?

For this repository, one runner is sufficient for most workflows. Consider adding a second runner if:
- You have > 10 PRs per day
- Jobs frequently queue
- You want redundancy

### Q: Can self-hosted runners use GitHub Actions cache?

Yes! The `actions/cache` action works on self-hosted runners. You can also use local filesystem caching for even better performance.

### Q: How do I update the runner software?

See [SELF_HOSTED_RUNNER.md](./SELF_HOSTED_RUNNER.md#updating-the-runner) for update instructions. GitHub will automatically prompt for updates in the Actions UI.

---

## Next Steps

1. **Set up runner**: Follow [SELF_HOSTED_RUNNER.md](./SELF_HOSTED_RUNNER.md)
2. **Test with one workflow**: Start with `_test.yml` (lowest risk)
3. **Monitor performance**: Track build times and resource usage
4. **Migrate remaining workflows**: Follow the phased approach above
5. **Optimize**: Fine-tune caching and concurrency

---

## Additional Resources

- [Self-Hosted Runner Setup Guide](./SELF_HOSTED_RUNNER.md)
- [GitHub Actions Self-Hosted Runners Documentation](https://docs.github.com/en/actions/hosting-your-own-runners)
- [Cartographus Development Guide](./DEVELOPMENT.md)

---

## Summary

**Recommended changes**:

1. Migrate `_build.yml`, `_test.yml`, `_e2e.yml`, `build-binaries.yml`, `_analysis.yml` to `runs-on: self-hosted`
2. Keep `_security.yml`, `_codeql.yml`, `release.yml` on `runs-on: ubuntu-latest`
3. Use hybrid approach for `_lint.yml` (backend on self-hosted, frontend on GitHub-hosted)

**Expected outcomes**:

- 40-50% faster CI runs
- Zero GitHub Actions minutes consumption
- Better resource utilization
- Persistent caching benefits

For setup instructions, see [SELF_HOSTED_RUNNER.md](./SELF_HOSTED_RUNNER.md).
