# Self-Hosted GitHub Actions Runner Setup

This guide explains how to set up a self-hosted GitHub Actions runner for the Cartographus repository using native host installation (not Docker Compose).

**Last Updated**: 2025-12-13
**Runner Version**: 2.329.0
**Architecture**: Native Linux host with Docker daemon access

---

## Table of Contents

- [Overview](#overview)
- [Prerequisites](#prerequisites)
- [Installation](#installation)
- [Running as a Service](#running-as-a-service)
- [Workflow Configuration](#workflow-configuration)
- [Maintenance](#maintenance)
  - [Dependency Management (Critical)](#dependency-management-critical)
- [Troubleshooting](#troubleshooting)
- [Security Considerations](#security-considerations)

---

## Overview

A self-hosted runner provides:

- **Faster build times**: No cold starts, persistent caches
- **Custom hardware**: More CPU/RAM than GitHub-hosted runners
- **Persistent Docker layers**: Faster image builds with layer caching
- **Cost savings**: No GitHub Actions minutes consumed
- **Local debugging**: Easier to troubleshoot build failures

This setup uses a native Linux installation where the runner process runs directly on the host with access to the host's Docker daemon.

---

## Prerequisites

### Hardware Requirements

**Minimum**:
- 4 CPU cores
- 16GB RAM
- 100GB SSD storage
- x86_64 (amd64) architecture

**Recommended**:
- 8+ CPU cores
- 32GB+ RAM
- 250GB+ SSD storage
- Fast network connection

### Software Requirements

The self-hosted runner needs the following tools installed on the host:

#### Core Tools

```bash
# System info
OS: Ubuntu 22.04 LTS or Ubuntu 24.04 LTS (REQUIRED)
    IMPORTANT: Ubuntu 25.x is NOT supported by Playwright
    If using Ubuntu 25.x, Playwright E2E tests will fail
Kernel: 5.15+ with namespace/cgroup support for Docker

# Required packages
- git (2.40+)
- curl/wget
- tar, gzip, xz-utils, bzip2
- jq (JSON processor)
- bc (calculator for shell scripts)
```

#### Build Toolchains

```bash
# Go compiler
- Go 1.24+ (for building backend)

# Node.js and npm
- Node.js 20.x LTS (for frontend builds)
- npm 10.x (comes with Node.js)

# C/C++ compilers (required for CGO/DuckDB)
- gcc (11.0+)
- g++
- make
- build-essential
```

#### Docker

```bash
# Docker Engine
- Docker Engine 24.0+ (NOT Docker Desktop)
- Docker Buildx plugin (multi-platform builds)
- QEMU user-mode emulation (for ARM64 builds)

# User permissions
- Runner user must be in 'docker' group
- Non-root Docker socket access
```

#### Cross-Compilation Toolchains (for build-binaries.yml)

```bash
# Linux ARM64 cross-compilation
- gcc-aarch64-linux-gnu
- g++-aarch64-linux-gnu

# Windows cross-compilation
- gcc-mingw-w64-x86-64
- g++-mingw-w64-x86-64

# macOS cross-compilation (OSXCross)
- clang
- llvm
- libxml2-dev
- uuid-dev
- libssl-dev
- libbz2-dev
- zlib1g-dev
```

#### Playwright Browser Dependencies

Playwright's Chromium browser requires these system libraries to be pre-installed:

```bash
# Core ATK/accessibility libraries (the most common missing ones)
- libatk1.0-0
- libatk-bridge2.0-0
- libatspi2.0-0

# X11 libraries
- libxcomposite1
- libxdamage1
- libxfixes3
- libxrandr2
- libxkbcommon0
- libxshmfence1

# Graphics libraries
- libdrm2
- libgbm1
- libcups2

# Text rendering
- libpango-1.0-0
- libcairo2

# Network/security
- libnss3
- libnspr4

# Audio/system
- libasound2
- libdbus-1-3
```

**These are installed automatically by `scripts/setup-runner-host.sh install`**

#### Additional Tools (installed by workflows)

These are installed automatically during workflow runs:

- `golangci-lint` (Go linting)
- `gotestsum` (Go test runner)
- `scc` (code complexity)
- `Playwright browsers` (E2E testing - browsers only, system deps must be pre-installed)
- `Trivy` (security scanning)
- `Cosign` (image signing)

---

## Installation

### Step 1: Prepare Host System

Run the automated setup script (recommended):

```bash
# Clone the repository
git clone https://github.com/tomtom215/cartographus.git
cd cartographus

# Run the setup script (requires sudo)
sudo bash scripts/setup-runner-host.sh install
```

Or manually install prerequisites:

```bash
# Update system
sudo apt-get update
sudo apt-get upgrade -y

# Install core dependencies
sudo apt-get install -y \
  git curl wget \
  tar gzip xz-utils bzip2 \
  jq bc \
  build-essential gcc g++ make \
  gcc-aarch64-linux-gnu g++-aarch64-linux-gnu \
  gcc-mingw-w64-x86-64 g++-mingw-w64-x86-64 \
  clang llvm \
  libxml2-dev uuid-dev libssl-dev libbz2-dev zlib1g-dev \
  libegl1 libgles2 libgl1-mesa-dri  # WebGL for Playwright screenshots

# Install Docker (if not already installed)
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh

# Install Docker Buildx
docker buildx create --use --name multiarch --driver docker-container

# Install Go 1.24+
wget https://go.dev/dl/go1.24.0.linux-amd64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.24.0.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc

# Install Node.js 20.x
curl -fsSL https://deb.nodesource.com/setup_20.x | sudo -E bash -
sudo apt-get install -y nodejs

# Verify installations
go version       # Should be 1.24.0 or higher
node --version   # Should be v20.x.x
npm --version    # Should be 10.x.x
docker --version # Should be 24.0.0 or higher
```

### Step 2: Create Runner User

```bash
# Create dedicated user for the runner
sudo useradd -m -s /bin/bash github-runner
sudo usermod -aG docker github-runner

# Verify docker group membership
groups github-runner
```

### Step 3: Download and Configure GitHub Actions Runner

**IMPORTANT**: These commands run as the `github-runner` user:

```bash
# Switch to runner user
sudo su - github-runner

# Create runner directory
mkdir actions-runner && cd actions-runner

# Download runner (check for latest version at https://github.com/actions/runner/releases)
curl -o actions-runner-linux-x64-2.329.0.tar.gz -L \
  https://github.com/actions/runner/releases/download/v2.329.0/actions-runner-linux-x64-2.329.0.tar.gz

# Verify hash (optional but recommended)
echo "d802e3d34f71b8fb6be0779e1f9ccae6e46ed8e0d1e94e9cb95fb6b91a653dc7  actions-runner-linux-x64-2.329.0.tar.gz" | shasum -a 256 -c

# Extract runner
tar xzf ./actions-runner-linux-x64-2.329.0.tar.gz
```

### Step 4: Configure Runner

You need a registration token from GitHub:

1. Go to: `https://github.com/tomtom215/cartographus/settings/actions/runners/new`
2. Copy the registration token
3. Run configuration (still as `github-runner` user):

```bash
# Configure runner (still as github-runner user)
./config.sh \
  --url https://github.com/tomtom215/cartographus \
  --token YOUR_REGISTRATION_TOKEN \
  --name "self-hosted-$(hostname)" \
  --labels "self-hosted,Linux,X64,docker,native" \
  --work _work
```

**Configuration options:**

- `--name`: Unique name for this runner
- `--labels`: Labels to identify runner capabilities
- `--work`: Working directory for job execution

### Step 5: Test Runner (Optional)

Before setting up as a service, you can test the runner interactively:

```bash
# Start runner interactively (still as github-runner user, press Ctrl+C to stop)
./run.sh
```

You should see:

```
Connected to GitHub
Listening for Jobs
```

Press Ctrl+C to stop the test run.

### Step 6: Make Directory Accessible

**IMPORTANT**: Before exiting the `github-runner` user session, you must make the `actions-runner` directory accessible to your admin user. By default, the directory has `700` permissions (owner-only access), which will prevent your admin user from accessing it for service installation.

```bash
# Still as github-runner user, make directory accessible
chmod 755 /home/github-runner/actions-runner
```

This allows your admin user to `cd` into the directory and run `sudo ./svc.sh install`.

---

## Running as a Service

### Install systemd Service

**CRITICAL**: You MUST exit from the `github-runner` user session before running these commands. Running `sudo` from within `sudo su - github-runner` will cause the service installation to fail with permission errors.

```bash
# 1. Exit from github-runner user back to your admin user (CRITICAL STEP!)
exit

# 2. Verify you're back to your original user (should NOT be github-runner)
whoami

# 3. Change to the runner directory (as your admin user)
cd /home/github-runner/actions-runner

# 4. Install service (requires sudo, run from your admin user, NOT from github-runner)
sudo ./svc.sh install github-runner

# 5. Start service
sudo ./svc.sh start

# 6. Check status (should show "active (running)")
sudo ./svc.sh status

# 7. View logs
sudo journalctl -u actions.runner.tomtom215-map.*.service -f
```

**Common Mistakes to Avoid**:

- ❌ DO NOT skip the `chmod 755` step before exiting (you'll get "Permission denied")
- ❌ DO NOT run `sudo ./svc.sh install` while still in `sudo su - github-runner` session
- ❌ DO NOT skip the `exit` step
- ❌ DO NOT run the service commands as the `github-runner` user
- ✅ DO run `chmod 755 /home/github-runner/actions-runner` before exiting
- ✅ DO exit back to your original admin user first
- ✅ DO run `sudo ./svc.sh install github-runner` from your admin user
- ✅ DO verify with `whoami` that you're not the github-runner user

**Why This Matters**:

1. **Permissions**: By default, `mkdir actions-runner` creates a directory with `700` permissions (owner-only access). Your admin user cannot `cd` into it without execute permission. Running `chmod 755` makes it accessible.

2. **Sudo Context**: The `svc.sh` script creates systemd service files and sets up permissions. This requires proper sudo context which doesn't work correctly when you're already in a `sudo su - username` session. The service must be installed by a user with direct sudo privileges, not from within a nested sudo context.

### Service Management Commands

```bash
# Start runner service
sudo systemctl start actions.runner.tomtom215-map.*.service

# Stop runner service
sudo systemctl stop actions.runner.tomtom215-map.*.service

# Restart runner service
sudo systemctl restart actions.runner.tomtom215-map.*.service

# Enable on boot
sudo systemctl enable actions.runner.tomtom215-map.*.service

# Disable on boot
sudo systemctl disable actions.runner.tomtom215-map.*.service

# View status
sudo systemctl status actions.runner.tomtom215-map.*.service

# View logs (live)
sudo journalctl -u actions.runner.tomtom215-map.*.service -f

# View logs (last 100 lines)
sudo journalctl -u actions.runner.tomtom215-map.*.service -n 100
```

---

## Workflow Configuration

### Which Workflows Should Use Self-Hosted Runners?

Not all workflows benefit from self-hosted runners. Here's the recommended configuration:

#### Recommended for Self-Hosted

1. **Build workflows** (`_build.yml`)
   - Multi-platform Docker builds (amd64, arm64)
   - Benefits from persistent Docker layer cache
   - Heavy CPU usage

2. **Test workflows** (`_test.yml`)
   - Unit tests with DuckDB extensions
   - Frequent runs on every PR
   - Benefits from cached Go modules

3. **E2E workflows** (`_e2e.yml`)
   - Playwright tests (338 tests)
   - Docker container management
   - Heavy resource usage

4. **Binary builds** (`build-binaries.yml`)
   - Cross-compilation with OSXCross
   - Long build times (5-10 minutes)

5. **Analysis workflows** (`_analysis.yml`)
   - Code coverage and profiling
   - Benefits from faster CPU

#### Keep on GitHub-Hosted

1. **Release workflows** (`release.yml`)
   - Keyless signing with Cosign requires GitHub OIDC
   - Security/trust considerations
   - Should use GitHub-hosted for audit trail

2. **Security workflows** (`_security.yml`, `_codeql.yml`)
   - Security scanning should use trusted infrastructure
   - SARIF uploads work better on GitHub-hosted

3. **Lint workflows** (`_lint.yml`)
   - Fast, lightweight
   - No benefit from self-hosted

### Hybrid Approach (Recommended)

Use a matrix strategy to run some jobs on self-hosted, others on GitHub-hosted:

```yaml
jobs:
  build-docker:
    name: Build ${{ matrix.platform }} Docker Image
    runs-on: ${{ matrix.runner }}
    strategy:
      matrix:
        include:
          - platform: linux/amd64
            runner: self-hosted  # Fast local builds
          - platform: linux/arm64
            runner: ubuntu-latest  # Use GitHub for ARM emulation
```

### Updating Workflows

To use the self-hosted runner, modify the `runs-on` field:

```yaml
# Before (GitHub-hosted)
jobs:
  build:
    runs-on: ubuntu-latest

# After (self-hosted)
jobs:
  build:
    runs-on: self-hosted

# Or with labels
jobs:
  build:
    runs-on: [self-hosted, linux, x64, docker]
```

**Example: Update _build.yml to use self-hosted runner**

```yaml
# In .github/workflows/_build.yml
jobs:
  build-docker:
    name: Build ${{ matrix.platform }} Docker Image
    runs-on: self-hosted  # Changed from ubuntu-latest
    needs: build-frontend
    # ... rest of configuration
```

---

## Maintenance

### Updating the Runner

```bash
# Stop service
sudo systemctl stop actions.runner.tomtom215-map.*.service

# Switch to runner user
sudo su - github-runner
cd actions-runner

# Download new version
curl -o actions-runner-linux-x64-NEW_VERSION.tar.gz -L \
  https://github.com/actions/runner/releases/download/vNEW_VERSION/actions-runner-linux-x64-NEW_VERSION.tar.gz

# Extract (replaces old files)
tar xzf ./actions-runner-linux-x64-NEW_VERSION.tar.gz

# Exit and restart service
exit
sudo systemctl start actions.runner.tomtom215-map.*.service
```

### Clearing Cache

```bash
# Docker layer cache
docker system prune -a -f

# Go module cache
sudo su - github-runner
rm -rf ~/actions-runner/_work/_tool/go
rm -rf ~/go/pkg/mod

# Node.js cache
rm -rf ~/actions-runner/_work/_tool/node
rm -rf ~/.npm
```

### Monitoring Disk Space

```bash
# Check overall disk usage
df -h

# Check runner work directory
du -sh /home/github-runner/actions-runner/_work

# Check Docker usage
docker system df

# Clean up Docker
docker system prune -a -f --volumes
```

### Log Rotation

The systemd service automatically rotates logs via journald. To configure retention:

```bash
# Edit journald config
sudo nano /etc/systemd/journald.conf

# Set retention (example: 1 week)
SystemMaxUse=1G
MaxRetentionSec=1week

# Restart journald
sudo systemctl restart systemd-journald
```

### Dependency Management (Critical)

Self-hosted runners have persistent caches that can cause issues when dependencies are updated. This is especially critical for DuckDB, which has complex version coupling.

#### The Problem

DuckDB has **three separate version components** that must be kept in sync:

1. **Go bindings** (`github.com/duckdb/duckdb-go-bindings`)
2. **Native libraries** (`github.com/duckdb/duckdb-go-bindings/lib/*`)
3. **Extensions** (`~/.duckdb/extensions/vX.Y.Z/`)

When Dependabot updates only part of this chain (e.g., lib packages but not bindings), tests can hang or fail with cryptic errors because the cached extensions are incompatible with the new binaries.

#### Automatic Protection

The CI workflow (`_test.yml`) has two layers of protection:

1. **Cache Invalidation**: Detects changes to `go.sum` and clears stale caches:
   ```bash
   # Triggered automatically when go.sum changes
   go clean -modcache
   rm -rf ~/.duckdb/extensions
   ```

2. **Smoke Test**: A quick database initialization test runs before the full test suite:
   ```bash
   # Runs in <30 seconds, catches version mismatches early
   go test -run "TestGetPlaybackTrends_EmptyData" ./internal/database/...
   ```

#### Dependencies Excluded from Dependabot

These dependencies are **ignored by Dependabot** because they require manual testing:

| Dependency | Reason |
|------------|--------|
| `github.com/duckdb/duckdb-go*` | Version coupling with extensions |
| `github.com/duckdb/duckdb-go-bindings*` | Version coupling with lib packages |
| `github.com/dgraph-io/badger*` | WAL storage, requires migration testing |
| `github.com/nats-io/*` | Event streaming, protocol compatibility |

#### Manual Dependency Update Process

When updating critical dependencies manually:

```bash
# 1. Update all DuckDB packages together
go get github.com/duckdb/duckdb-go/v2@latest
go get github.com/duckdb/duckdb-go-bindings@latest
go get github.com/duckdb/duckdb-go-bindings/lib/linux-amd64@latest
# ... (repeat for all platforms)
go mod tidy

# 2. Clear caches on runner
ssh github-runner@<host>
go clean -modcache
rm -rf ~/.duckdb/extensions

# 3. Test locally before pushing
go test -v -race ./internal/database/...

# 4. Push and verify CI passes
git push
```

#### Troubleshooting Version Mismatches

**Symptom**: Tests hang with "TestTileCache_SetAndGet" or similar database tests timing out.

**Solution**:

```bash
# SSH to runner
ssh github-runner@<host>

# Clear all caches
go clean -modcache
rm -rf ~/.duckdb/extensions
rm -rf ~/.cache/cartographus-deps-hash

# Re-trigger CI
# The workflow will download fresh dependencies
```

---

## Removing a Runner

### Option 1: Automated Removal (Recommended)

Use the setup script's uninstall mode:

```bash
sudo bash scripts/setup-runner-host.sh uninstall
```

This interactive script will:
1. Stop and uninstall the systemd service
2. Prompt for a removal token to unregister from GitHub
3. Ask if you want to delete the `actions-runner` directory
4. Ask if you want to remove the `github-runner` user
5. Preserve Go, Node.js, and Docker (may be used elsewhere)

**Note**: You'll need a removal token from GitHub. Get it from:
- `https://github.com/tomtom215/cartographus/settings/actions/runners`
- Click on your runner → Click "Remove" → Copy the token

### Option 2: Manual Removal

If you prefer to remove the runner manually, follow these steps:

#### Step 1: Stop the Service

```bash
# Find the service name
sudo systemctl list-units --type=service --all | grep actions.runner

# Stop the service
sudo systemctl stop actions.runner.tomtom215-map.*.service
```

#### Step 2: Uninstall the Service

```bash
cd /home/github-runner/actions-runner
sudo ./svc.sh uninstall
```

#### Step 3: Remove from GitHub

Get a removal token from the GitHub UI:
1. Go to: `https://github.com/tomtom215/cartographus/settings/actions/runners`
2. Click on your runner
3. Click "Remove" button
4. Copy the removal token

Then run:

```bash
cd /home/github-runner/actions-runner
./config.sh remove --token YOUR_REMOVAL_TOKEN
```

#### Step 4: Delete Runner Files

```bash
# Delete the runner directory
sudo rm -rf /home/github-runner/actions-runner

# Optionally remove the github-runner user (WARNING: deletes home directory)
sudo userdel -r github-runner
```

#### Step 5: Clean Up Docker Cache (Optional)

```bash
# Remove buildx cache
sudo rm -rf /tmp/.buildx-cache
sudo rm -rf /tmp/.buildx-cache-new

# Clean up Docker system
docker system prune -a -f --volumes
```

### What Gets Removed vs. Preserved

**Removed by uninstall:**
- Runner systemd service
- Runner registration from GitHub
- `/home/github-runner/actions-runner` directory
- Optionally: `github-runner` user and home directory

**Preserved (may be used by other applications):**
- Go installation (`/usr/local/go`)
- Node.js installation
- Docker Engine
- System dependencies (gcc, make, toolchains)

**To manually remove preserved software:**

```bash
# Remove Go
sudo rm -rf /usr/local/go
sudo sed -i '/\/usr\/local\/go\/bin/d' /etc/profile.d/go.sh
sudo rm -f /etc/profile.d/go.sh

# Remove Node.js (via NodeSource)
sudo apt-get remove -y nodejs
sudo rm -f /etc/apt/sources.list.d/nodesource.list

# Remove Docker
sudo apt-get remove -y docker-ce docker-ce-cli containerd.io \
  docker-buildx-plugin docker-compose-plugin
sudo rm -rf /var/lib/docker
sudo rm -rf /var/lib/containerd
```

### Removing Multiple Runners

If you have multiple runners on the same host (different runner names), remove them one at a time:

```bash
# List all runner services
systemctl list-units --type=service --all | grep actions.runner

# Remove each runner service individually
sudo systemctl stop actions.runner.tomtom215-map.runner1.service
cd /home/github-runner/actions-runner-1
sudo ./svc.sh uninstall
./config.sh remove --token TOKEN1

sudo systemctl stop actions.runner.tomtom215-map.runner2.service
cd /home/github-runner/actions-runner-2
sudo ./svc.sh uninstall
./config.sh remove --token TOKEN2
```

---

## Troubleshooting

### Runner Not Connecting

**Symptom**: Runner shows "Offline" in GitHub UI

**Solutions**:

```bash
# Check service status
sudo systemctl status actions.runner.tomtom215-map.*.service

# Check logs
sudo journalctl -u actions.runner.tomtom215-map.*.service -n 50

# Test network connectivity
curl -I https://github.com
curl -I https://api.github.com

# Verify token is valid (re-register if needed)
cd /home/github-runner/actions-runner
sudo su - github-runner
./config.sh remove --token YOUR_REMOVAL_TOKEN
./config.sh --url https://github.com/tomtom215/cartographus --token YOUR_NEW_TOKEN
exit
sudo systemctl restart actions.runner.tomtom215-map.*.service
```

### Docker Permission Denied

**Symptom**: Jobs fail with "permission denied while trying to connect to Docker daemon"

**Solution**:

```bash
# Verify github-runner is in docker group
groups github-runner

# Add to docker group if missing
sudo usermod -aG docker github-runner

# Restart service
sudo systemctl restart actions.runner.tomtom215-map.*.service

# Verify docker access
sudo su - github-runner
docker ps  # Should work without sudo
```

### Out of Disk Space

**Symptom**: Jobs fail with "no space left on device"

**Solution**:

```bash
# Check disk usage
df -h
docker system df

# Clean up Docker
docker system prune -a -f --volumes

# Clean up old workflow runs
cd /home/github-runner/actions-runner/_work
sudo rm -rf */*/  # Be careful with this command

# Clean up Go cache
sudo su - github-runner
go clean -cache -modcache -testcache
```

### DuckDB Extension Failures

**Symptom**: Tests fail with "extension not available" or "extension not loading"

**Solution**:

```bash
# Install DuckDB extensions manually
sudo su - github-runner
cd /home/github-runner/map
./scripts/setup-duckdb-extensions.sh

# Verify extensions
ls -lh ~/.duckdb/extensions/v1.4.2/linux_amd64/

# Expected extensions:
# - spatial.duckdb_extension (core - geometry, spatial queries)
# - h3.duckdb_extension (community - hexagonal indexing)
# - inet.duckdb_extension (core - IP address handling)
# - icu.duckdb_extension (core - timezone operations)
# - json.duckdb_extension (core - JSON processing)
# - sqlite_scanner.duckdb_extension (core - Tautulli import)
# - rapidfuzz.duckdb_extension (community - fuzzy string matching)
# - datasketches.duckdb_extension (community - approximate analytics)
```

### Go Module Download Failures

**Symptom**: "failed to download module" errors

**Solution**:

```bash
# Clear Go module cache
sudo su - github-runner
go clean -modcache

# Set proxy (if behind corporate firewall)
export GOPROXY=https://proxy.golang.org,direct
export GOSUMDB=sum.golang.org

# Retry download
cd /home/github-runner/map
go mod download
```

### Playwright Browser Installation Failures

**Symptom**: "Executable doesn't exist at /home/github-runner/.cache/ms-playwright/chromium-XXX/chrome-linux/chrome"

**Solution**:

```bash
# Install browsers manually as the runner user
sudo su - github-runner
cd /home/github-runner/map/web
npx playwright install chromium

# Verify installation
npx playwright --version
```

### Playwright Missing Libraries (libatk, libnss3, etc.)

**Symptom**: E2E tests fail with errors like:
```
error while loading shared libraries: libatk-1.0.so.0: cannot open shared object file
```

**Cause**: Playwright's Chromium browser requires system libraries that aren't installed
on the runner host. The workflow verifies these dependencies are pre-installed.

**Solution**: Re-run the setup script on your runner host:

```bash
# SSH into your runner host and run:
cd /path/to/map
sudo bash scripts/setup-runner-host.sh install

# This will install all required Playwright dependencies including:
# - libatk1.0-0, libatk-bridge2.0-0, libatspi2.0-0
# - libcups2, libdrm2, libgbm1, libnss3, libnspr4
# - libxcomposite1, libxdamage1, libxfixes3, libxrandr2
# - And more (see install_playwright_deps function in the script)
```

**Manual installation** (if you can't run the setup script):

```bash
sudo apt-get update
sudo apt-get install -y \
  libatk1.0-0 libatk-bridge2.0-0 libatspi2.0-0 \
  libxcomposite1 libxdamage1 libxfixes3 libxrandr2 libxkbcommon0 libxshmfence1 \
  libdrm2 libgbm1 libcups2 \
  libpango-1.0-0 libcairo2 \
  libnss3 libnspr4 \
  libasound2 libdbus-1-3
```

**Note**: On Ubuntu 24.04+, some packages may have `t64` suffix (e.g., `libcups2t64`).
The setup script handles this automatically.

### OSXCross Build Failures (build-binaries.yml)

**Symptom**: "osxcross: command not found" or macOS SDK errors

**Solution**:

```bash
# OSXCross is built during workflow run
# Ensure these packages are installed:
sudo apt-get install -y \
  clang llvm \
  libxml2-dev uuid-dev libssl-dev \
  libbz2-dev zlib1g-dev \
  bash patch make tar xz-utils bzip2 gzip sed cpio

# Check available disk space (OSXCross needs ~2GB)
df -h

# If builds consistently fail, consider disabling macOS builds
# or using GitHub-hosted runners for that matrix job
```

### Job Timeout Issues

**Symptom**: Jobs time out after 6 hours (default GitHub Actions limit)

**Solution**:

```yaml
# In workflow file, increase timeout for slow jobs
jobs:
  build-binaries:
    timeout-minutes: 120  # 2 hours instead of default 360 minutes
```

### Runner Using Too Much CPU/RAM

**Symptom**: System becomes unresponsive during builds

**Solution**:

```bash
# Limit concurrent jobs (edit runner config)
cd /home/github-runner/actions-runner
./config.sh remove --token YOUR_REMOVAL_TOKEN
./config.sh \
  --url https://github.com/tomtom215/cartographus \
  --token YOUR_TOKEN \
  --name "self-hosted-$(hostname)" \
  --labels "self-hosted,Linux,X64,docker,native" \
  --work _work \
  --runnergroup default \
  --disableupdate  # Optional: disable auto-updates

# Or use systemd resource limits
sudo systemctl edit actions.runner.tomtom215-map.*.service

# Add these lines:
[Service]
CPUQuota=400%  # Limit to 4 cores
MemoryMax=8G   # Limit to 8GB RAM
```

---

## Security Considerations

### Runner Security

1. **Dedicated User**: Always run the runner as a non-root user (`github-runner`)
2. **Firewall**: Restrict outbound connections to GitHub API endpoints only
3. **Secrets**: Never log secrets; workflows should use GitHub Secrets
4. **Network**: Use a dedicated VLAN or subnet for runners
5. **Updates**: Keep runner, Docker, and host OS up to date

### Docker Security

```bash
# Enable Docker content trust
export DOCKER_CONTENT_TRUST=1

# Scan images before pushing
docker scan map:test

# Use read-only root filesystem where possible
docker run --read-only ...
```

### Access Control

```bash
# Restrict who can create self-hosted runners
# GitHub Settings > Actions > Runners > Runner groups > Permissions

# Use separate runners for:
# - Public repositories (untrusted code)
# - Private repositories (trusted code)

# For this private repo, configure:
# Settings > Actions > Runners > Default runner group > Selected repositories only
```

### Monitoring

```bash
# Monitor runner logs for suspicious activity
sudo journalctl -u actions.runner.tomtom215-map.*.service | grep -i "error\|fail\|suspicious"

# Set up alerts for:
# - Failed login attempts
# - Unusual Docker container activity
# - High resource usage
# - Job failures
```

### CI/CD Security Policies

#### Action Version Pinning

This project uses **semantic version pinning** (e.g., `@v6.0.1`) for GitHub Actions rather than full SHA pinning. This approach balances security with maintainability:

| Approach | Example | Security | Maintenance |
|----------|---------|----------|-------------|
| Major version | `@v6` | Lower | Easy |
| **Semantic version** | `@v6.0.1` | Medium | Moderate |
| Full SHA | `@abc123...` | Higher | Difficult |

**Rationale:**
- Semantic versions are immutable after creation (cannot be changed)
- SHA pinning breaks Dependabot automatic updates
- Major version tags can be moved (security risk)
- Our approach: Pin to specific semantic versions, update regularly

**When to use SHA pinning:**
- Actions from untrusted sources
- Security-critical actions (signing, secrets access)
- When required by compliance

#### Node.js Version Management

Node.js version is pinned to **LTS version 20** across all workflows:
- `.nvmrc` file in repository root for local development
- `node-version: '20'` in all workflow `setup-node` steps
- Uses Node.js LTS for stability and long-term support

To update Node.js version:
1. Update `.nvmrc` files (root and `web/`)
2. Update all workflow `node-version` values
3. Test locally and in CI before merging

#### Secrets Management

Test secrets (JWT, API keys) in E2E workflows are generated at runtime:
- Uses `openssl rand -base64 48` for cryptographically random values
- Not hardcoded in workflow files (audit trail clean)
- Each job generates its own secret (isolation)

Production secrets must:
- Be stored in GitHub Secrets (encrypted at rest)
- Never appear in workflow files or logs
- Use environment variable redaction (`add-mask`)

---

## Additional Resources

- [GitHub Actions Self-Hosted Runners Documentation](https://docs.github.com/en/actions/hosting-your-own-runners)
- [Docker Multi-Platform Builds](https://docs.docker.com/build/building/multi-platform/)
- [Cartographus Development Guide](./DEVELOPMENT.md)
- [Cartographus Architecture](./ARCHITECTURE.md)

---

## Summary

To set up a self-hosted runner:

1. Install prerequisites (Go, Node.js, Docker, build tools)
2. Create `github-runner` user
3. Download and configure GitHub Actions runner
4. Install as systemd service
5. Update workflows to use `runs-on: self-hosted`
6. Monitor logs and disk space regularly

The runner will significantly speed up CI/CD pipelines, especially for Docker builds and E2E tests.

For automated setup, use:

```bash
sudo bash scripts/setup-runner-host.sh install
```

To verify the installation without reinstalling:

```bash
bash scripts/setup-runner-host.sh verify
```

For manual setup, follow the step-by-step instructions above.
