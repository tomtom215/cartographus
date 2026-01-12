# ADR-0011: CI/CD Infrastructure with Self-Hosted Runners

**Date**: 2025-12-02
**Status**: Accepted

---

## Context

Cartographus CI/CD requires:

1. **Multi-Architecture Builds**: linux/amd64, linux/arm64, darwin, windows
2. **DuckDB Extensions**: Pre-installation for spatial, H3, INET, ICU
3. **WebGL Testing**: Browser screenshots for E2E tests
4. **Large Test Suite**: 379 Go test files, 1300+ E2E tests (75 spec files)
5. **Code Analysis**: Coverage, benchmarks, complexity metrics

### GitHub-Hosted Runner Limitations

| Limitation | Impact |
|------------|--------|
| 2 vCPU, 7GB RAM | Slow builds, DuckDB memory issues |
| No persistent cache | Extension downloads every run |
| No GPU/WebGL | Screenshot tests fail |
| Rate limited | Slow for many parallel jobs |

---

## Decision

Implement a **hybrid CI/CD architecture**:

- **Self-Hosted Runners**: Build, test, E2E (Ubuntu 24.04, 8+ cores, 32GB RAM)
- **GitHub-Hosted Runners**: Security scanning (Trivy, CodeQL, Gitleaks)

### Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                  GitHub Actions Workflows                    │
└────────────────────────────┬────────────────────────────────┘
                             │
         ┌───────────────────┴───────────────────┐
         ▼                                       ▼
┌─────────────────────┐               ┌─────────────────────┐
│  Self-Hosted Runner │               │ GitHub-Hosted Runner│
│  (ubuntu-24.04)     │               │  (ubuntu-latest)    │
├─────────────────────┤               ├─────────────────────┤
│ - Go 1.24+          │               │ - Trivy Scanner     │
│ - Node.js 20.x      │               │ - CodeQL Analysis   │
│ - Docker Buildx     │               │ - Gitleaks Scan     │
│ - DuckDB Extensions │               │ - License Check     │
│ - WebGL Libraries   │               │                     │
│ - gotestsum, scc    │               │                     │
│ - ARM64 Toolchain   │               │                     │
└─────────────────────┘               └─────────────────────┘
         │                                       │
         ▼                                       ▼
┌─────────────────────┐               ┌─────────────────────┐
│ Workflows:          │               │ Workflows:          │
│ - _lint.yml         │               │ - _security.yml     │
│ - _test.yml         │               │ - _codeql.yml       │
│ - _build.yml        │               │                     │
│ - _analysis.yml     │               │                     │
│ - _e2e.yml          │               │                     │
└─────────────────────┘               └─────────────────────┘
```

### Key Factors

1. **Performance**: 50%+ faster builds with persistent cache
2. **Reliability**: Pre-installed DuckDB extensions never fail
3. **WebGL Support**: Mesa libraries for screenshot tests
4. **Security Isolation**: Sensitive scans on GitHub infrastructure
5. **Cost Control**: Self-hosted for compute-heavy, hosted for scanning

---

## Consequences

### Positive

- **Faster CI**: ~3 minutes vs ~8 minutes for full build
- **Reliable Tests**: No extension download failures
- **E2E Screenshots**: WebGL working for Playwright
- **Full Coverage**: All test types can run
- **Cache Persistence**: Tools and dependencies cached

### Negative

- **Infrastructure Management**: Must maintain runner host
- **Security Responsibility**: Runner host security is our burden
- **Availability**: Single point of failure if runner is down

### Neutral

- **Hybrid Complexity**: Two runner types to configure
- **Workflow Conditions**: Need `runs-on` logic for runner selection

---

## Implementation

### Workflow Structure

```yaml
# .github/workflows/_test.yml
name: Test

on:
  workflow_call:

jobs:
  test:
    runs-on: self-hosted
    steps:
      - uses: actions/checkout@v4

      # Skip setup on self-hosted (pre-installed)
      - name: Setup Go
        if: runner.environment != 'self-hosted'
        uses: actions/setup-go@v5
        with:
          go-version: '1.24'

      - name: Run Tests
        run: |
          gotestsum --format pkgname -- -race -coverprofile=coverage.out ./...

      - name: Upload Coverage
        uses: codecov/codecov-action@v5
```

### Self-Hosted Runner Setup

```bash
#!/bin/bash
# scripts/setup-runner-host.sh

# Go 1.24+
wget https://go.dev/dl/go1.24.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.24.linux-amd64.tar.gz

# Go tools
go install gotest.tools/gotestsum@latest
go install github.com/wadey/gocovmerge@latest
go install golang.org/x/perf/cmd/benchstat@latest

# Node.js 20.x
curl -fsSL https://deb.nodesource.com/setup_20.x | sudo -E bash -
sudo apt-get install -y nodejs

# Docker with Buildx
sudo apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin

# WebGL dependencies
sudo apt-get install -y libegl1 libgles2 libgl1-mesa-dri xvfb

# DuckDB extensions (pre-download)
./scripts/setup-duckdb-extensions.sh

# scc (code complexity)
wget https://github.com/boyter/scc/releases/download/v3.5.0/scc_Linux_x86_64.tar.gz
sudo tar -xzf scc_Linux_x86_64.tar.gz -C /usr/local/bin

# GitHub Actions Runner
mkdir actions-runner && cd actions-runner
curl -o actions-runner-linux-x64-2.321.0.tar.gz -L https://github.com/actions/runner/releases/download/v2.321.0/actions-runner-linux-x64-2.321.0.tar.gz
tar xzf ./actions-runner-linux-x64-2.321.0.tar.gz
./config.sh --url https://github.com/OWNER/REPO --token TOKEN
sudo ./svc.sh install
sudo ./svc.sh start
```

### E2E Test Configuration

```yaml
# .github/workflows/_e2e.yml
name: E2E Tests

on:
  workflow_call:

jobs:
  e2e:
    # Only run on main branch merges (skip on PRs)
    if: github.ref == 'refs/heads/main' && github.event_name == 'push'
    runs-on: self-hosted
    steps:
      - uses: actions/checkout@v4

      - name: Build Application
        run: |
          CGO_ENABLED=1 go build -o cartographus ./cmd/server
          cd web && npm ci && npm run build

      - name: Start Application
        run: |
          ./cartographus &
          sleep 5

      - name: Run Playwright Tests
        run: |
          cd web
          npx playwright test --reporter=html

      - name: Upload Report
        uses: actions/upload-artifact@v4
        with:
          name: playwright-report
          path: web/playwright-report/
```

### Security Workflow (GitHub-Hosted)

```yaml
# .github/workflows/_security.yml
name: Security

on:
  workflow_call:

jobs:
  security:
    runs-on: ubuntu-latest  # GitHub-hosted for isolation
    steps:
      - uses: actions/checkout@v4

      - name: Trivy Vulnerability Scan
        uses: aquasecurity/trivy-action@master
        with:
          scan-type: 'fs'
          severity: 'HIGH,CRITICAL'
          exit-code: '1'  # Fail on vulnerabilities

      - name: Gitleaks Secret Scan
        uses: gitleaks/gitleaks-action@v2
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}

      - name: License Compliance
        run: |
          go install github.com/google/go-licenses@latest
          go-licenses check ./...
```

### Docker Cache Strategy

```yaml
# Self-hosted uses local cache (faster)
- name: Build Docker Image
  uses: docker/build-push-action@v6
  with:
    push: false
    cache-from: type=local,src=/tmp/.buildx-cache
    cache-to: type=local,dest=/tmp/.buildx-cache-new,mode=max

# GitHub-hosted uses GHA cache
- name: Build Docker Image
  uses: docker/build-push-action@v6
  with:
    push: false
    cache-from: type=gha
    cache-to: type=gha,mode=max
```

### Code References

| Component | File | Notes |
|-----------|------|-------|
| Test workflow | `.github/workflows/_test.yml` | Unit tests |
| Build workflow | `.github/workflows/_build.yml` | Docker, binaries |
| E2E workflow | `.github/workflows/_e2e.yml` | Playwright |
| Security workflow | `.github/workflows/_security.yml` | Trivy, CodeQL |
| Runner setup | `scripts/setup-runner-host.sh` | Host provisioning |
| DuckDB setup | `scripts/setup-duckdb-extensions.sh` | Extensions |

---

## Verification

### Verified Claims

| Claim | Source | Verified |
|-------|--------|----------|
| 379 Go test files | `**/*_test.go` | Yes |
| 1300+ E2E tests (75 spec files) | `tests/e2e/` directory | Yes |
| golangci-lint v2.6.2 | `.github/workflows/_lint.yml` | Yes |
| WebGL dependencies | `scripts/setup-runner-host.sh` | Yes |
| scc v3.5.0 | `scripts/setup-runner-host.sh` | Yes |

### Test Coverage

- CI workflow tests: Manual verification
- E2E in CI: 1300+ Playwright tests
- Coverage target: 75.5% overall

---

## Related ADRs

- [ADR-0001](0001-use-duckdb-for-analytics.md): DuckDB extension requirements
- [ADR-0002](0002-frontend-technology-stack.md): Playwright E2E

---

## References

- [GitHub Actions Self-Hosted Runners](https://docs.github.com/en/actions/hosting-your-own-runners)
- [Docker Buildx Cache](https://docs.docker.com/build/cache/)
- [Playwright CI Configuration](https://playwright.dev/docs/ci)
- [docs/SELF_HOSTED_RUNNER.md](../SELF_HOSTED_RUNNER.md)
