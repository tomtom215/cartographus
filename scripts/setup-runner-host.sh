#!/usr/bin/env bash

# Cartographus - Media Server Analytics and Geographic Visualization
# Copyright 2026 Tom F. (tomtom215)
# SPDX-License-Identifier: AGPL-3.0-or-later
# https://github.com/tomtom215/cartographus
set -euo pipefail

# setup-runner-host.sh
# Automated setup/update/removal script for GitHub Actions self-hosted runner host
# Installs all prerequisites for running Cartographus CI/CD workflows
#
# Usage:
#   sudo bash scripts/setup-runner-host.sh install    # Install all prerequisites
#   sudo bash scripts/setup-runner-host.sh update     # Update all dependencies in-place
#   bash scripts/setup-runner-host.sh verify          # Check installation (no sudo needed)
#   sudo bash scripts/setup-runner-host.sh uninstall  # Remove runner
#
# Install mode will:
# - Install system dependencies (gcc, make, cross-compilation toolchains)
# - Install Go 1.24+
# - Install Node.js 20.x
# - Install Docker Engine (if not present)
# - Set up Docker Buildx
# - Create github-runner user
# - Configure permissions
#
# Update mode will:
# - Check and update Go to latest stable version
# - Update Node.js to latest LTS within major version
# - Update Go tools (gotestsum, gocovmerge, benchstat)
# - Refresh DuckDB extensions
# - Verify all permissions are correct
# - Does NOT break running services
#
# Uninstall mode will:
# - Stop and uninstall runner service
# - Remove runner from GitHub (requires token)
# - Delete actions-runner directory
# - Optionally remove github-runner user
# - Does NOT remove Go/Node.js/Docker (may be used elsewhere)

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
# GO_VERSION: Set to "auto" to detect latest stable, or specify exact version
# Can be overridden via environment: GO_VERSION=1.23.4 sudo bash setup-runner-host.sh install
GO_VERSION="${GO_VERSION:-auto}"
# NOTE: SHA256 is fetched dynamically from go.dev to ensure correctness
NODE_VERSION="20"
DOCKER_MIN_VERSION="24.0"
RUNNER_USER="github-runner"
LOCK_FILE="/var/run/setup-runner-host.lock"
LOG_FILE="/var/log/setup-runner-host.log"
RUNNER_VERSION="2.330.0"  # GitHub Actions runner version (check https://github.com/actions/runner/releases for latest)

# Pinned Go tool versions for reproducibility (NO @latest - must be explicit)
GOTESTSUM_VERSION="v1.13.0"
# gocovmerge has no tags, use commit hash for reproducibility
GOCOVMERGE_VERSION="b5bfa59ec0adc420475f97f89b58045c721d761c"
# benchstat uses pseudo-version format: v0.0.0-YYYYMMDDHHMMSS-HASH12
BENCHSTAT_VERSION="v0.0.0-20251208221838-04cf7a2dca90"
SCC_VERSION="3.5.0"

# Network retry configuration
MAX_RETRIES=4
RETRY_DELAYS=(2 4 8 16)  # Exponential backoff in seconds

# CI cache invalidation marker path (matches _test.yml)
# Used in clear_caches() and health_check() functions
readonly CI_CACHE_MARKER_NAME="cartographus-deps-hash"

# Mode flags
DRY_RUN=false
VERBOSE=false

# Logging functions - output to stderr (so they don't pollute function return values)
# Also writes to log file for audit trail
_log() {
    local level="$1"
    local color="$2"
    local message="$3"
    local timestamp
    timestamp=$(date '+%Y-%m-%d %H:%M:%S')

    # Console output with color (to stderr, so $(function) captures don't include log messages)
    echo -e "${color}[${level}]${NC} ${message}" >&2

    # Log file output (plain text, with timestamp)
    if [ -n "$LOG_FILE" ] && [ "$DRY_RUN" = false ]; then
        echo "[${timestamp}] [${level}] ${message}" >> "$LOG_FILE" 2>/dev/null || true
    fi
}

log_info() {
    _log "INFO" "${BLUE}" "$1"
}

log_success() {
    _log "SUCCESS" "${GREEN}" "$1"
}

log_warn() {
    _log "WARN" "${YELLOW}" "$1"
}

log_error() {
    _log "ERROR" "${RED}" "$1"
}

log_debug() {
    if [ "$VERBOSE" = true ]; then
        _log "DEBUG" "${BLUE}" "$1"
    fi
}

# Network download with retry and exponential backoff
download_with_retry() {
    local url="$1"
    local output="$2"
    local description="${3:-file}"
    local attempt=1

    while [ $attempt -le $MAX_RETRIES ]; do
        log_info "Downloading ${description} (attempt ${attempt}/${MAX_RETRIES})..."

        if wget -q --timeout=30 --tries=1 "$url" -O "$output" 2>/dev/null; then
            log_success "Downloaded ${description}"
            return 0
        fi

        if [ $attempt -lt $MAX_RETRIES ]; then
            local delay=${RETRY_DELAYS[$((attempt-1))]}
            log_warn "Download failed, retrying in ${delay}s..."
            sleep "$delay"
        fi

        ((attempt++))
    done

    log_error "Failed to download ${description} after ${MAX_RETRIES} attempts"
    return 1
}

# Dry-run wrapper - skip actual commands in dry-run mode
run_cmd() {
    if [ "$DRY_RUN" = true ]; then
        log_info "[DRY-RUN] Would execute: $*"
        return 0
    fi
    "$@"
}

# Safe directory creation with proper error handling
# Usage: ensure_directory "/path/to/dir" [owner] [mode]
# Returns: 0 on success, 1 on failure
ensure_directory() {
    local dir="$1"
    local owner="${2:-}"
    local mode="${3:-755}"

    # Check if directory already exists
    if [ -d "$dir" ]; then
        log_debug "Directory already exists: $dir"
        # Fix ownership if specified
        if [ -n "$owner" ]; then
            if ! chown "$owner:$owner" "$dir" 2>/dev/null; then
                log_error "Failed to set ownership on $dir"
                return 1
            fi
        fi
        # Fix permissions
        if ! chmod "$mode" "$dir" 2>/dev/null; then
            log_error "Failed to set permissions on $dir"
            return 1
        fi
        return 0
    fi

    # Create parent directories first
    local parent
    parent=$(dirname "$dir")
    if [ ! -d "$parent" ]; then
        if ! ensure_directory "$parent" "$owner" "$mode"; then
            return 1
        fi
    fi

    # Create the directory
    if [ -n "$owner" ]; then
        # Create as specific user
        if ! sudo -u "$owner" mkdir -p "$dir" 2>/dev/null; then
            # Fallback: create as root then chown
            if ! mkdir -p "$dir" 2>/dev/null; then
                log_error "Failed to create directory: $dir"
                return 1
            fi
            if ! chown "$owner:$owner" "$dir" 2>/dev/null; then
                log_error "Failed to set ownership on $dir"
                return 1
            fi
        fi
    else
        if ! mkdir -p "$dir" 2>/dev/null; then
            log_error "Failed to create directory: $dir"
            return 1
        fi
    fi

    # Set permissions
    if ! chmod "$mode" "$dir" 2>/dev/null; then
        log_error "Failed to set permissions on $dir"
        return 1
    fi

    log_debug "Created directory: $dir (owner=$owner, mode=$mode)"
    return 0
}

# Verify file/directory permissions
# Usage: verify_permissions "/path" "expected_owner" "expected_mode"
verify_permissions() {
    local path="$1"
    local expected_owner="$2"
    local expected_mode="$3"

    if [ ! -e "$path" ]; then
        log_error "Path does not exist: $path"
        return 1
    fi

    local actual_owner
    actual_owner=$(stat -c '%U' "$path" 2>/dev/null)
    if [ "$actual_owner" != "$expected_owner" ]; then
        log_warn "Ownership mismatch on $path: expected $expected_owner, got $actual_owner"
        return 1
    fi

    local actual_mode
    actual_mode=$(stat -c '%a' "$path" 2>/dev/null)
    if [ "$actual_mode" != "$expected_mode" ]; then
        log_warn "Permission mismatch on $path: expected $expected_mode, got $actual_mode"
        return 1
    fi

    return 0
}

# Check if running as root
check_root() {
    if [ "$EUID" -ne 0 ]; then
        log_error "This script must be run as root (use sudo)"
        exit 1
    fi
}

# Acquire lock to prevent concurrent runs
acquire_lock() {
    # Ensure lock directory exists
    local lock_dir
    lock_dir=$(dirname "$LOCK_FILE")
    if [ ! -d "$lock_dir" ]; then
        if ! mkdir -p "$lock_dir" 2>/dev/null; then
            log_warn "Cannot create lock directory $lock_dir, skipping lock"
            return 0
        fi
    fi

    if [ -f "$LOCK_FILE" ]; then
        local pid
        pid=$(cat "$LOCK_FILE" 2>/dev/null)
        if [ -n "$pid" ] && kill -0 "$pid" 2>/dev/null; then
            log_error "Another instance is running (PID: $pid)"
            log_error "If this is incorrect, remove $LOCK_FILE"
            exit 1
        else
            log_warn "Stale lock file found, removing..."
            rm -f "$LOCK_FILE"
        fi
    fi

    # Write lock file with error handling
    if ! echo $$ > "$LOCK_FILE" 2>/dev/null; then
        log_warn "Cannot create lock file, proceeding without lock"
        return 0
    fi

    # Ensure lock is removed on exit
    trap 'rm -f "$LOCK_FILE" 2>/dev/null' EXIT
    log_info "Acquired lock (PID: $$)"
}

# Release lock (called automatically via trap)
release_lock() {
    rm -f "$LOCK_FILE"
}

# Check network connectivity
check_network() {
    log_info "Checking network connectivity..."

    # Test connectivity using URLs that reliably return 200 OK
    # CRITICAL: Use -L to follow redirects
    # NOTE: Don't use -I (HEAD request) - some servers/CDNs don't handle it properly
    local -A test_urls=(
        ["github.com"]="https://github.com"
        ["go.dev"]="https://go.dev/dl/"
        ["deb.nodesource.com"]="https://deb.nodesource.com/gpgkey/nodesource-repo.gpg.key"
        ["download.docker.com"]="https://download.docker.com/linux/ubuntu/gpg"
    )
    local failed=false

    for host in "${!test_urls[@]}"; do
        local url="${test_urls[$host]}"
        # -s: silent, -f: fail on HTTP errors, -L: follow redirects
        # -o /dev/null: discard output (we only care about success/failure)
        # --max-time: timeout for entire operation
        if ! curl -sfL --max-time 10 "$url" -o /dev/null 2>/dev/null; then
            log_error "Cannot reach ${host} (${url})"
            failed=true
        else
            log_debug "Reached ${host}"
        fi
    done

    if [ "$failed" = true ]; then
        log_error "Network connectivity check failed"
        log_error "Ensure the host can reach GitHub, Go, NodeSource, and Docker"
        log_error "Debug: Try running 'curl -vL https://go.dev/dl/' to see detailed errors"
        exit 1
    fi

    log_success "Network connectivity OK"
}

# Check available disk space
check_disk_space() {
    log_info "Checking available disk space..."

    # Need at least 5GB free for all installations
    local required_mb=5120
    local available_mb
    available_mb=$(df / --output=avail -BM | tail -1 | tr -d 'M ')

    if [ "$available_mb" -lt "$required_mb" ]; then
        log_error "Insufficient disk space: ${available_mb}MB available, ${required_mb}MB required"
        log_error "Free up space and try again"
        exit 1
    fi

    log_success "Disk space OK: ${available_mb}MB available"
}

# Check OS compatibility
check_os() {
    log_info "Checking OS compatibility..."

    if [ ! -f /etc/os-release ]; then
        log_error "Cannot detect OS. /etc/os-release not found."
        exit 1
    fi

    . /etc/os-release

    case "$ID" in
        ubuntu|debian)
            log_success "Detected compatible OS: $PRETTY_NAME"
            ;;
        *)
            log_error "Unsupported OS: $PRETTY_NAME"
            log_error "This script supports Ubuntu and Debian only."
            exit 1
            ;;
    esac

    # Check architecture
    ARCH=$(uname -m)
    if [ "$ARCH" != "x86_64" ]; then
        log_error "Unsupported architecture: $ARCH"
        log_error "This script supports x86_64 (amd64) only."
        exit 1
    fi

    log_success "Architecture: $ARCH"
}

# Check required commands are available
check_required_commands() {
    log_info "Checking required commands..."

    local missing=()

    # curl is required for downloads and network checks
    if ! command -v curl &> /dev/null; then
        missing+=("curl")
    fi

    # wget is used as fallback for some downloads
    if ! command -v wget &> /dev/null; then
        missing+=("wget")
    fi

    # tar is required to extract archives
    if ! command -v tar &> /dev/null; then
        missing+=("tar")
    fi

    # sha256sum is required for checksum verification
    if ! command -v sha256sum &> /dev/null; then
        missing+=("coreutils (for sha256sum)")
    fi

    if [ ${#missing[@]} -gt 0 ]; then
        log_error "Missing required commands: ${missing[*]}"
        log_error "Install them first: sudo apt-get install -y curl wget tar coreutils"
        exit 1
    fi

    log_success "Required commands available (curl, wget, tar, sha256sum)"
}

# Update system packages
update_system() {
    log_info "Updating system packages..."
    apt-get update -qq
    apt-get upgrade -y -qq
    log_success "System packages updated"
}

# Optional: Enable automatic security updates only
# WARNING: For CI runners, automatic updates can cause unexpected build failures.
# Only enable this if you prefer security over strict reproducibility.
# To enable: uncomment the setup_auto_security_updates call in main_install()
setup_auto_security_updates() {
    log_info "Setting up automatic security updates..."

    # Install unattended-upgrades
    apt-get install -y -qq unattended-upgrades apt-listchanges

    # Configure for security updates only
    cat > /etc/apt/apt.conf.d/50unattended-upgrades << 'EOF'
Unattended-Upgrade::Allowed-Origins {
    "${distro_id}:${distro_codename}-security";
    // "${distro_id}:${distro_codename}-updates";  // Disabled for CI stability
};

// Do not automatically reboot
Unattended-Upgrade::Automatic-Reboot "false";

// Remove unused dependencies
Unattended-Upgrade::Remove-Unused-Dependencies "true";

// Log to syslog
Unattended-Upgrade::SyslogEnable "true";
EOF

    # Enable automatic updates
    cat > /etc/apt/apt.conf.d/20auto-upgrades << 'EOF'
APT::Periodic::Update-Package-Lists "1";
APT::Periodic::Unattended-Upgrade "1";
APT::Periodic::AutocleanInterval "7";
EOF

    # Enable and start the service
    systemctl enable unattended-upgrades
    systemctl start unattended-upgrades

    log_success "Automatic security updates enabled (security patches only)"
    log_warn "Note: This may cause occasional CI instability if security updates break compatibility"
}

# Install core dependencies
install_core_deps() {
    log_info "Installing core dependencies..."

    apt-get install -y -qq \
        git \
        curl \
        wget \
        tar \
        gzip \
        xz-utils \
        bzip2 \
        jq \
        bc \
        build-essential \
        gcc \
        g++ \
        make \
        ca-certificates \
        gnupg \
        lsb-release

    log_success "Core dependencies installed"
}

# Install scc (code complexity analyzer)
install_scc() {
    log_info "Installing scc (code complexity analyzer)..."

    # Check if scc is already installed
    if command -v scc &> /dev/null; then
        SCC_VERSION=$(scc --version 2>/dev/null | head -1 || echo "unknown")
        log_success "scc already installed: ${SCC_VERSION}"
        return
    fi

    # Download and install scc
    SCC_VERSION="3.5.0"
    wget -q --timeout=30 --tries=3 "https://github.com/boyter/scc/releases/download/v${SCC_VERSION}/scc_Linux_x86_64.tar.gz" -O /tmp/scc.tar.gz
    tar -xzf /tmp/scc.tar.gz -C /tmp
    mv /tmp/scc /usr/local/bin/
    chmod +x /usr/local/bin/scc
    rm /tmp/scc.tar.gz

    # Verify installation
    if command -v scc &> /dev/null; then
        SCC_VERSION=$(scc --version 2>/dev/null | head -1 || echo "installed")
        log_success "scc installed: ${SCC_VERSION}"
    else
        log_error "scc installation failed"
        exit 1
    fi
}

# Install WebGL dependencies for Playwright screenshots
install_webgl_deps() {
    log_info "Installing WebGL dependencies for Playwright screenshots..."

    # Check if already installed using dpkg-query (more reliable than grep)
    local all_installed=true
    for pkg in libegl1 libgles2 libgl1-mesa-dri; do
        if ! dpkg-query -W -f='${Status}' "$pkg" 2>/dev/null | grep -q "install ok installed"; then
            all_installed=false
            break
        fi
    done

    if [ "$all_installed" = true ]; then
        log_success "WebGL dependencies already installed"
        return
    fi

    # Install packages (no -qq to show errors)
    log_info "Installing libegl1, libgles2, libgl1-mesa-dri..."
    if ! apt-get install -y --no-install-recommends \
        libegl1 \
        libgles2 \
        libgl1-mesa-dri 2>&1 | tee /tmp/webgl-install.log; then
        log_error "apt-get install failed. Check /tmp/webgl-install.log for details"
        log_error "Last 10 lines of error:"
        tail -10 /tmp/webgl-install.log
        exit 1
    fi

    # Verify installations using dpkg-query
    local verify_failed=false
    for pkg in libegl1 libgles2 libgl1-mesa-dri; do
        if ! dpkg-query -W -f='${Status}' "$pkg" 2>/dev/null | grep -q "install ok installed"; then
            log_error "Package $pkg not properly installed"
            verify_failed=true
        fi
    done

    if [ "$verify_failed" = true ]; then
        log_error "WebGL dependencies installation verification failed"
        log_info "Installed packages matching pattern:"
        dpkg -l | grep -E "libegl|libgles|libgl1-mesa" || echo "No matching packages found"
        exit 1
    fi

    log_success "WebGL dependencies installed (libegl1, libgles2, libgl1-mesa-dri)"
}

# Install Playwright browser system dependencies
# These are required for Chromium to run in E2E tests
# Full list derived from: npx playwright install-deps chromium --dry-run
install_playwright_deps() {
    log_info "Installing Playwright browser system dependencies..."

    # Playwright Chromium requires these system libraries
    # Package names may vary between Ubuntu versions (some have t64 suffix on 24.04+)
    # We try the standard name first, then the t64 variant if needed

    # Core packages that should exist on all Ubuntu versions
    # This is the COMPLETE list needed for Chromium headless
    local CORE_PACKAGES=(
        # ATK (Accessibility Toolkit)
        "libatk1.0-0"
        "libatk-bridge2.0-0"
        "libatspi2.0-0"
        # X11 core libraries
        "libx11-6"
        "libx11-xcb1"
        "libxcb1"
        "libxcomposite1"
        "libxdamage1"
        "libxext6"
        "libxfixes3"
        "libxrandr2"
        "libxkbcommon0"
        "libxshmfence1"
        # Graphics / rendering
        "libdrm2"
        "libgbm1"
        "libgl1"
        "libglib2.0-0"
        # Text rendering
        "libpango-1.0-0"
        "libpangocairo-1.0-0"
        "libcairo2"
        "libcairo-gobject2"
        "libfontconfig1"
        "libfreetype6"
        "libharfbuzz0b"
        # GTK (for some Chromium features)
        "libgtk-3-0"
        "libgdk-pixbuf-2.0-0"
        # Network/security
        "libnss3"
        "libnspr4"
        "libsecret-1-0"
        # System
        "libdbus-1-3"
        "libexpat1"
        "libffi8"
        "libuuid1"
        "zlib1g"
        # Fonts (required for text rendering)
        "fonts-liberation"
        "fonts-noto-color-emoji"
    )

    # Packages that may have t64 suffix on Ubuntu 24.04+
    local T64_CANDIDATES=(
        "libcups2"
        "libasound2"
    )

    # First, install core packages
    log_info "Installing core Playwright dependencies..."
    if ! apt-get install -y --no-install-recommends "${CORE_PACKAGES[@]}" 2>&1 | tee /tmp/playwright-deps.log; then
        log_error "Failed to install core Playwright dependencies"
        log_error "Last 20 lines of log:"
        tail -20 /tmp/playwright-deps.log
        exit 1
    fi

    # Handle packages that may have t64 suffix
    log_info "Installing additional Playwright dependencies (handling Ubuntu version differences)..."
    for pkg in "${T64_CANDIDATES[@]}"; do
        # Try standard package name first
        if apt-get install -y --no-install-recommends "$pkg" 2>/dev/null; then
            log_success "  Installed: $pkg"
        # Try t64 variant (Ubuntu 24.04+)
        elif apt-get install -y --no-install-recommends "${pkg}t64" 2>/dev/null; then
            log_success "  Installed: ${pkg}t64"
        else
            log_warn "  Could not install $pkg or ${pkg}t64 - Playwright may still work"
        fi
    done

    # Verify critical libraries exist
    log_info "Verifying critical Playwright libraries..."
    local verify_failed=false
    local critical_libs=(
        "/usr/lib/x86_64-linux-gnu/libatk-1.0.so.0"
        "/usr/lib/x86_64-linux-gnu/libatk-bridge-2.0.so.0"
        "/usr/lib/x86_64-linux-gnu/libcups.so.2"
        "/usr/lib/x86_64-linux-gnu/libdrm.so.2"
        "/usr/lib/x86_64-linux-gnu/libgbm.so.1"
        "/usr/lib/x86_64-linux-gnu/libnss3.so"
        "/usr/lib/x86_64-linux-gnu/libpango-1.0.so.0"
        "/usr/lib/x86_64-linux-gnu/libcairo.so.2"
        "/usr/lib/x86_64-linux-gnu/libX11.so.6"
        "/usr/lib/x86_64-linux-gnu/libxcb.so.1"
        "/usr/lib/x86_64-linux-gnu/libgtk-3.so.0"
    )

    for lib in "${critical_libs[@]}"; do
        if [ -f "$lib" ]; then
            log_success "  Found: $(basename "$lib")"
        else
            log_error "  Missing: $lib"
            verify_failed=true
        fi
    done

    if [ "$verify_failed" = true ]; then
        log_error "Some critical Playwright libraries are missing"
        log_error "Try running: npx playwright install-deps chromium"
        exit 1
    fi

    log_success "Playwright browser dependencies installed successfully"
}

# Install cross-compilation toolchains
install_cross_toolchains() {
    log_info "Installing cross-compilation toolchains..."

    apt-get install -y -qq \
        gcc-aarch64-linux-gnu \
        g++-aarch64-linux-gnu \
        gcc-mingw-w64-x86-64 \
        g++-mingw-w64-x86-64

    log_success "Cross-compilation toolchains installed"

    # Verify installations
    log_info "Verifying toolchains..."
    which aarch64-linux-gnu-gcc >/dev/null 2>&1 && log_success "  ARM64: aarch64-linux-gnu-gcc"
    which x86_64-w64-mingw32-gcc >/dev/null 2>&1 && log_success "  Windows: x86_64-w64-mingw32-gcc"
}

# Install OSXCross dependencies (OSXCross itself is built during workflow)
install_osxcross_deps() {
    log_info "Installing OSXCross dependencies..."

    apt-get install -y -qq \
        clang \
        llvm \
        libxml2-dev \
        uuid-dev \
        libssl-dev \
        bash \
        patch \
        make \
        tar \
        xz-utils \
        bzip2 \
        gzip \
        sed \
        cpio \
        libbz2-dev \
        zlib1g-dev

    log_success "OSXCross dependencies installed"
}

# Fetch latest stable Go version from go.dev API
# Returns version number without "go" prefix (e.g., "1.23.4")
fetch_latest_go_version() {
    log_info "Detecting latest stable Go version from go.dev..."

    local version=""
    local json_data

    # Fetch JSON data once (cache for reuse)
    json_data=$(curl -sfL --max-time 15 "https://go.dev/dl/?mode=json" 2>/dev/null)

    if [ -z "$json_data" ]; then
        log_error "Failed to fetch Go version data from go.dev"
        return 1
    fi

    if command -v jq &> /dev/null; then
        # Use jq for reliable JSON parsing
        # Get the first stable version from the list
        version=$(echo "$json_data" | \
            jq -r '[.[] | select(.stable == true)][0].version' 2>/dev/null | \
            sed 's/^go//' | tr -d '[:space:]')
    fi

    # Fallback: parse with grep/sed if jq not available or failed
    if [ -z "$version" ]; then
        log_debug "jq not available or failed, using grep fallback..."
        # Find first "version":"goX.Y.Z" where stable:true follows
        version=$(echo "$json_data" | \
            grep -oP '"version":"go[0-9]+\.[0-9]+\.[0-9]+"[^}]*"stable":true' | \
            head -1 | \
            grep -oP '(?<="version":"go)[0-9]+\.[0-9]+\.[0-9]+')
    fi

    if [ -z "$version" ]; then
        log_error "Failed to detect latest Go version"
        return 1
    fi

    # Validate version format (X.Y.Z)
    if ! [[ "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
        log_error "Invalid Go version format detected: ${version}"
        return 1
    fi

    log_success "Latest stable Go version: ${version}"
    echo "$version"
}

# Fetch Go checksum from official go.dev API
# Returns the SHA256 checksum for a specific Go version and platform
fetch_go_checksum() {
    local version="$1"
    local filename="go${version}.linux-amd64.tar.gz"

    log_info "Fetching checksum for ${filename} from go.dev..."

    # go.dev provides a JSON API with all download info
    # We use jq to parse it, but fall back to grep if jq isn't available
    local checksum=""
    local json_data

    # Fetch JSON data once
    json_data=$(curl -sfL --max-time 15 "https://go.dev/dl/?mode=json" 2>/dev/null)

    if [ -z "$json_data" ]; then
        log_error "Failed to fetch data from go.dev"
        return 1
    fi

    if command -v jq &> /dev/null; then
        # Use jq for reliable JSON parsing
        # Strip whitespace to ensure clean output
        checksum=$(echo "$json_data" | \
            jq -r --arg fn "$filename" '.[] | .files[] | select(.filename == $fn) | .sha256' 2>/dev/null | \
            head -1 | tr -d '[:space:]')
    fi

    # Fallback: parse with grep/sed if jq not available or failed
    if [ -z "$checksum" ]; then
        log_debug "jq not available or failed, using grep fallback..."
        checksum=$(echo "$json_data" | \
            grep -oP "\"filename\":\"${filename}\"[^}]*\"sha256\":\"[a-f0-9]+\"" | \
            grep -oP '(?<="sha256":")[a-f0-9]+' | head -1 | tr -d '[:space:]')
    fi

    # Validate checksum format (exactly 64 hex characters)
    if [ -z "$checksum" ]; then
        log_error "Checksum not found for Go ${version}"
        log_error "The version may not exist. Check available versions at: https://go.dev/dl/"
        return 1
    fi

    if [ ${#checksum} -ne 64 ]; then
        log_error "Invalid checksum format for Go ${version}"
        log_error "Expected 64 characters, got ${#checksum}: '${checksum}'"
        return 1
    fi

    # Verify checksum contains only hex characters
    if ! [[ "$checksum" =~ ^[a-f0-9]+$ ]]; then
        log_error "Checksum contains invalid characters: '${checksum}'"
        return 1
    fi

    echo "$checksum"
}

# Install Go with checksum verification
install_go() {
    local target_version="$GO_VERSION"

    # Handle "auto" - detect latest stable version
    if [ "$target_version" = "auto" ]; then
        log_info "GO_VERSION is 'auto', detecting latest stable version..."
        if ! target_version=$(fetch_latest_go_version) || [ -z "$target_version" ]; then
            log_error "Failed to detect latest Go version"
            log_error "Specify an explicit version: GO_VERSION=1.23.4 sudo bash $0 install"
            exit 1
        fi
    fi

    log_info "Installing Go ${target_version}..."

    # Check if Go is already installed with correct version
    if command -v go &> /dev/null; then
        CURRENT_GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
        if [ "$CURRENT_GO_VERSION" = "$target_version" ]; then
            log_success "Go ${target_version} already installed"
            return
        else
            log_warn "Found Go ${CURRENT_GO_VERSION}, upgrading to ${target_version}..."
        fi
    fi

    if [ "$DRY_RUN" = true ]; then
        log_info "[DRY-RUN] Would download and install Go ${target_version}"
        return
    fi

    # Fetch expected checksum from go.dev (ensures we have correct value)
    local expected_sha256
    if ! expected_sha256=$(fetch_go_checksum "$target_version") || [ -z "$expected_sha256" ]; then
        log_error "Cannot proceed without valid checksum"
        exit 1
    fi
    log_success "Fetched expected checksum: ${expected_sha256:0:16}..."

    # Download Go with retry
    local go_tarball="go${target_version}.linux-amd64.tar.gz"
    local go_url="https://go.dev/dl/${go_tarball}"

    if ! download_with_retry "$go_url" "/tmp/${go_tarball}" "Go ${target_version}"; then
        log_error "Failed to download Go"
        exit 1
    fi

    # Verify SHA256 checksum (CRITICAL for security and reproducibility)
    log_info "Verifying Go tarball checksum..."
    local actual_sha256
    actual_sha256=$(sha256sum "/tmp/${go_tarball}" | awk '{print $1}' | tr -d '[:space:]')

    # Strip any whitespace from expected checksum too (defensive)
    expected_sha256=$(echo "$expected_sha256" | tr -d '[:space:]')

    if [ "$actual_sha256" != "$expected_sha256" ]; then
        log_error "Checksum verification FAILED!"
        log_error "Expected: ${expected_sha256}"
        log_error "Actual:   ${actual_sha256}"
        log_error "Expected length: ${#expected_sha256}, Actual length: ${#actual_sha256}"
        log_error "This may indicate a corrupted download or supply chain attack."
        log_error "Verify the correct checksum at: https://go.dev/dl/"
        rm -f "/tmp/${go_tarball}"
        exit 1
    fi
    log_success "Checksum verified: ${actual_sha256:0:16}..."

    # Remove old installation
    rm -rf /usr/local/go

    # Extract new version
    tar -C /usr/local -xzf "/tmp/${go_tarball}"
    rm "/tmp/${go_tarball}"

    # Add to PATH for all users
    if ! grep -q '/usr/local/go/bin' /etc/profile.d/go.sh 2>/dev/null; then
        # shellcheck disable=SC2016 # Intentional: $PATH should expand at shell startup, not now
        echo 'export PATH=$PATH:/usr/local/go/bin' > /etc/profile.d/go.sh
        chmod +x /etc/profile.d/go.sh
    fi

    # Verify installation
    export PATH=$PATH:/usr/local/go/bin
    local go_installed_version
    go_installed_version=$(/usr/local/go/bin/go version | awk '{print $3}' | sed 's/go//')

    if [ "$go_installed_version" = "$target_version" ]; then
        log_success "Go ${target_version} installed successfully"
    else
        log_error "Go installation failed"
        log_error "Expected version: ${target_version}, got: ${go_installed_version}"
        exit 1
    fi
}

# Install Go tools
# Supports both install and update modes via optional parameter
install_go_tools() {
    local mode="${1:-install}"  # "install" or "update"
    log_info "Installing/updating Go tools (gotestsum, gocovmerge, benchstat)..."

    # Ensure Go is in PATH
    export PATH=$PATH:/usr/local/go/bin

    # Verify Go is available
    if ! command -v go &>/dev/null && [ ! -x /usr/local/go/bin/go ]; then
        log_error "Go is not installed. Run 'install' first."
        exit 1
    fi

    # Create GOPATH directory with proper permissions
    export GOPATH=/usr/local/go-tools
    if ! ensure_directory "$GOPATH" "" "755"; then
        log_error "Failed to create GOPATH directory"
        exit 1
    fi

    # Helper function to install a Go tool
    install_go_tool() {
        local name="$1"
        local package="$2"
        local version="$3"

        # In update mode, always reinstall
        if [ "$mode" = "update" ]; then
            log_info "Updating $name@$version..."
            rm -f "/usr/local/bin/$name"
        elif command -v "$name" &> /dev/null; then
            log_success "$name already installed"
            return 0
        fi

        log_info "Installing $name@$version..."
        if ! GOBIN=/usr/local/bin /usr/local/go/bin/go install "${package}@${version}" 2>&1; then
            log_error "Failed to install $name"
            return 1
        fi

        # Verify installation
        if ! command -v "$name" &>/dev/null; then
            log_error "$name not found after installation"
            return 1
        fi

        log_success "$name installed"
        return 0
    }

    local failed_tools=()

    # Install gotestsum (pinned version for reproducibility)
    if ! install_go_tool "gotestsum" "gotest.tools/gotestsum" "$GOTESTSUM_VERSION"; then
        failed_tools+=("gotestsum")
    fi

    # Install gocovmerge
    if ! install_go_tool "gocovmerge" "github.com/wadey/gocovmerge" "$GOCOVMERGE_VERSION"; then
        failed_tools+=("gocovmerge")
    fi

    # Install benchstat
    if ! install_go_tool "benchstat" "golang.org/x/perf/cmd/benchstat" "$BENCHSTAT_VERSION"; then
        failed_tools+=("benchstat")
    fi

    # Report results
    if [ ${#failed_tools[@]} -gt 0 ]; then
        log_error "Failed to install Go tools: ${failed_tools[*]}"
        exit 1
    fi

    log_success "Go tools installed/updated successfully"
}

# Install Node.js
install_nodejs() {
    log_info "Installing Node.js ${NODE_VERSION}.x..."

    # Check if Node.js is already installed with correct major version
    if command -v node &> /dev/null; then
        CURRENT_NODE_VERSION=$(node --version | sed 's/v//' | cut -d. -f1)
        if [ "$CURRENT_NODE_VERSION" = "$NODE_VERSION" ]; then
            log_success "Node.js ${NODE_VERSION}.x already installed"
            return
        else
            log_warn "Found Node.js ${CURRENT_NODE_VERSION}.x, upgrading to ${NODE_VERSION}.x..."
        fi
    fi

    # Add NodeSource repository
    curl -fsSL https://deb.nodesource.com/setup_${NODE_VERSION}.x | bash -

    # Install Node.js
    apt-get install -y -qq nodejs

    # Verify installation
    NODE_INSTALLED_VERSION=$(node --version | sed 's/v//' | cut -d. -f1)
    NPM_VERSION=$(npm --version)

    if [ "$NODE_INSTALLED_VERSION" = "$NODE_VERSION" ]; then
        log_success "Node.js ${NODE_VERSION}.x installed successfully"
        log_success "npm ${NPM_VERSION} installed"
    else
        log_error "Node.js installation failed"
        exit 1
    fi
}

# Install Docker
install_docker() {
    log_info "Checking Docker installation..."

    # Check if Docker is already installed
    if command -v docker &> /dev/null; then
        DOCKER_VERSION=$(docker --version | awk '{print $3}' | sed 's/,//')
        log_success "Docker ${DOCKER_VERSION} already installed"

        # Check if version is recent enough
        DOCKER_MAJOR=$(echo "$DOCKER_VERSION" | cut -d. -f1)
        if [ "$DOCKER_MAJOR" -lt 24 ]; then
            log_warn "Docker version ${DOCKER_VERSION} is older than recommended ${DOCKER_MIN_VERSION}"
            log_warn "Consider upgrading Docker manually"
        fi
    else
        log_info "Installing Docker Engine..."

        # Detect OS for Docker repository
        . /etc/os-release
        DOCKER_OS="$ID"  # Will be 'ubuntu' or 'debian'

        # Add Docker's official GPG key
        install -m 0755 -d /etc/apt/keyrings
        curl -fsSL "https://download.docker.com/linux/${DOCKER_OS}/gpg" | gpg --dearmor -o /etc/apt/keyrings/docker.gpg
        chmod a+r /etc/apt/keyrings/docker.gpg

        # Add Docker repository
        echo \
          "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/${DOCKER_OS} \
          $(lsb_release -cs) stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null

        # Install Docker
        apt-get update -qq
        apt-get install -y -qq \
            docker-ce \
            docker-ce-cli \
            containerd.io \
            docker-buildx-plugin \
            docker-compose-plugin

        # Verify installation
        DOCKER_VERSION=$(docker --version | awk '{print $3}' | sed 's/,//')
        log_success "Docker ${DOCKER_VERSION} installed successfully"
    fi

    # Start and enable Docker service
    systemctl enable docker
    systemctl start docker

    log_success "Docker service enabled and started"
}

# Setup Docker Buildx
setup_docker_buildx() {
    log_info "Setting up Docker Buildx..."

    # Check if buildx builder already exists
    if docker buildx ls | grep -q "multiarch"; then
        log_success "Docker Buildx 'multiarch' builder already exists"
    else
        # Create multiarch builder
        docker buildx create --use --name multiarch --driver docker-container
        log_success "Docker Buildx 'multiarch' builder created"
    fi

    # Install QEMU for multi-platform builds (only if not already set up)
    log_info "Checking QEMU for ARM64 emulation..."
    if [ -f /proc/sys/fs/binfmt_misc/qemu-aarch64 ]; then
        log_success "QEMU ARM64 emulation already configured"
    else
        log_info "Installing QEMU for ARM64 emulation..."
        docker run --privileged --rm tonistiigi/binfmt --install all
        log_success "QEMU installed for multi-platform builds"
    fi
}

# Create runner user
create_runner_user() {
    log_info "Creating runner user '${RUNNER_USER}'..."

    local runner_home="/home/${RUNNER_USER}"

    # Check if user already exists
    if id "$RUNNER_USER" &>/dev/null; then
        log_success "User '${RUNNER_USER}' already exists"
    else
        # Create user with error handling
        if ! useradd -m -s /bin/bash "$RUNNER_USER" 2>/dev/null; then
            log_error "Failed to create user '${RUNNER_USER}'"
            exit 1
        fi
        log_success "User '${RUNNER_USER}' created"
    fi

    # Verify home directory exists
    if [ ! -d "$runner_home" ]; then
        log_warn "Home directory does not exist, creating..."
        if ! mkdir -p "$runner_home"; then
            log_error "Failed to create home directory: $runner_home"
            exit 1
        fi
        chown "${RUNNER_USER}:${RUNNER_USER}" "$runner_home"
    fi

    # Set proper permissions on home directory
    # 755 allows admin users to traverse and access actions-runner directory
    if ! chmod 755 "$runner_home" 2>/dev/null; then
        log_error "Failed to set permissions on home directory"
        exit 1
    fi
    log_success "Home directory permissions set (755)"

    # Verify home directory ownership
    local owner
    owner=$(stat -c '%U' "$runner_home" 2>/dev/null)
    if [ "$owner" != "$RUNNER_USER" ]; then
        log_warn "Fixing home directory ownership..."
        chown "${RUNNER_USER}:${RUNNER_USER}" "$runner_home"
    fi

    # Check if docker group exists before adding user
    if ! getent group docker &>/dev/null; then
        log_warn "Docker group does not exist, creating..."
        groupadd docker
    fi

    # Add user to docker group
    if ! usermod -aG docker "$RUNNER_USER" 2>/dev/null; then
        log_error "Failed to add user to docker group"
        exit 1
    fi
    log_success "User '${RUNNER_USER}' added to 'docker' group"

    # Verify group membership
    if groups "$RUNNER_USER" | grep -q docker; then
        log_success "Docker group membership verified"
    else
        log_error "Failed to verify docker group membership"
        exit 1
    fi

    # Create essential directories for the runner user
    local essential_dirs=(
        "$runner_home/.cache"
        "$runner_home/.config"
        "$runner_home/go"
    )

    for dir in "${essential_dirs[@]}"; do
        if ! ensure_directory "$dir" "$RUNNER_USER" "755"; then
            log_warn "Failed to create $dir (non-critical)"
        fi
    done

    log_success "Runner user setup complete"
}

# Setup DuckDB extensions for runner user
# Self-contained: does not require external scripts
# Deterministic: always creates same directory structure with proper permissions
# Robust: exponential backoff with jitter, atomic downloads, file validation
setup_duckdb_extensions() {
    log_info "Setting up DuckDB extensions for runner user..."

    # DuckDB version must match duckdb-go-bindings in go.mod
    # Single source of truth is scripts/duckdb-version.sh - keep in sync when updating
    local DUCKDB_VERSION="${DUCKDB_VERSION:-v1.4.3}"
    local PLATFORM="linux_amd64"
    local RUNNER_HOME="/home/${RUNNER_USER}"
    local DUCKDB_BASE="${RUNNER_HOME}/.duckdb"
    local EXTENSIONS_DIR="${DUCKDB_BASE}/extensions/${DUCKDB_VERSION}/${PLATFORM}"
    local MAIN_REPO="https://extensions.duckdb.org/${DUCKDB_VERSION}/${PLATFORM}"
    local COMMUNITY_REPO="https://community-extensions.duckdb.org/${DUCKDB_VERSION}/${PLATFORM}"

    # Retry configuration
    local EXT_MAX_RETRIES=5
    local EXT_RETRY_DELAY=2
    local EXT_DOWNLOAD_TIMEOUT=120
    local EXT_MIN_FILE_SIZE=10240  # Extensions should be at least 10KB

    # ALL extensions - all are now required for full functionality
    # Core extensions from main repository
    local -a CORE_EXTENSIONS=("httpfs" "spatial" "inet" "icu" "json" "sqlite_scanner")
    # Community extensions
    local -a COMMUNITY_EXTENSIONS=("h3" "rapidfuzz" "datasketches")

    log_info "DuckDB Version: ${DUCKDB_VERSION}, Platform: ${PLATFORM}"

    # Verify runner user exists
    if ! id "$RUNNER_USER" &>/dev/null; then
        log_error "Runner user '$RUNNER_USER' does not exist. Run 'install' first."
        exit 1
    fi

    # Verify runner home directory exists
    if [ ! -d "$RUNNER_HOME" ]; then
        log_error "Runner home directory does not exist: $RUNNER_HOME"
        exit 1
    fi

    # Create directory hierarchy with proper permissions
    log_info "Creating DuckDB extensions directory hierarchy..."

    # Create base .duckdb directory
    if ! ensure_directory "$DUCKDB_BASE" "$RUNNER_USER" "755"; then
        log_error "Failed to create DuckDB base directory: $DUCKDB_BASE"
        exit 1
    fi

    # Create extensions subdirectory
    if ! ensure_directory "${DUCKDB_BASE}/extensions" "$RUNNER_USER" "755"; then
        log_error "Failed to create DuckDB extensions directory"
        exit 1
    fi

    # Create version subdirectory
    if ! ensure_directory "${DUCKDB_BASE}/extensions/${DUCKDB_VERSION}" "$RUNNER_USER" "755"; then
        log_error "Failed to create DuckDB version directory"
        exit 1
    fi

    # Create platform subdirectory (final target)
    if ! ensure_directory "$EXTENSIONS_DIR" "$RUNNER_USER" "755"; then
        log_error "Failed to create DuckDB platform directory: $EXTENSIONS_DIR"
        exit 1
    fi

    # Verify directory was created with correct ownership
    if ! verify_permissions "$EXTENSIONS_DIR" "$RUNNER_USER" "755"; then
        log_error "DuckDB extensions directory has incorrect permissions"
        log_info "Attempting to fix permissions..."
        chown -R "${RUNNER_USER}:${RUNNER_USER}" "$DUCKDB_BASE"
        chmod -R 755 "$DUCKDB_BASE"
    fi

    log_success "DuckDB extensions directory created: $EXTENSIONS_DIR"

    # Track installation results
    local installed_count=0
    local failed_extensions=()

    # Calculate exponential backoff with jitter
    # Usage: calculate_ext_backoff attempt_number
    calculate_ext_backoff() {
        local attempt=$1
        local base_delay=$EXT_RETRY_DELAY
        # Exponential backoff: base * 2^attempt
        local delay=$((base_delay * (1 << attempt)))
        # Add jitter (0-25% of delay)
        local jitter=$((RANDOM % (delay / 4 + 1)))
        echo $((delay + jitter))
    }

    # Validate extension file
    # Returns: 0 if valid, 1 if invalid
    validate_extension_file() {
        local file=$1
        # File must exist
        [ -f "$file" ] || return 1
        # File must be non-empty
        [ -s "$file" ] || return 1
        # File must be readable
        [ -r "$file" ] || return 1
        # DuckDB extensions should be at least 10KB
        local size
        size=$(stat -c%s "$file" 2>/dev/null || echo 0)
        [ "$size" -gt "$EXT_MIN_FILE_SIZE" ] || return 1
        return 0
    }

    # Download and extract a single extension with exponential backoff
    # Returns: 0 on success, 1 on failure
    download_duckdb_extension() {
        local name=$1
        local repo=$2
        local url="${repo}/${name}.duckdb_extension.gz"
        local output="${EXTENSIONS_DIR}/${name}.duckdb_extension"
        local temp_gz="${EXTENSIONS_DIR}/.${name}.duckdb_extension.gz.tmp"
        local temp_ext="${EXTENSIONS_DIR}/.${name}.duckdb_extension.tmp"

        # Check if already installed and valid (skip if so)
        if validate_extension_file "$output"; then
            local size
            size=$(stat -c%s "$output" 2>/dev/null || echo "?")
            log_success "  ${name} already installed (${size} bytes)"
            ((installed_count++))
            return 0
        fi

        # Remove invalid existing file
        if [ -f "$output" ]; then
            log_warn "  ${name} exists but is invalid, re-downloading..."
            rm -f "$output"
        fi

        log_info "  Downloading ${name}..."
        log_debug "    URL: $url"

        # Remove any stale temp files
        rm -f "$temp_gz" "$temp_ext" 2>/dev/null || true

        # Retry loop with exponential backoff and jitter
        local attempt=0
        local success=false
        local last_error=""

        while [ $attempt -lt $EXT_MAX_RETRIES ]; do
            attempt=$((attempt + 1))
            log_debug "    Attempt $attempt of $EXT_MAX_RETRIES"

            # Download with curl
            local curl_exit=0
            local http_code
            http_code=$(sudo -u "$RUNNER_USER" curl -w "%{http_code}" \
                --silent \
                --show-error \
                --fail \
                --location \
                --max-time "$EXT_DOWNLOAD_TIMEOUT" \
                --output "$temp_gz" \
                "$url" 2>&1) || curl_exit=$?

            if [ $curl_exit -eq 0 ] && [ "$http_code" = "200" ]; then
                # Verify downloaded file is not empty
                if [ ! -s "$temp_gz" ]; then
                    last_error="Downloaded file is empty"
                    log_debug "    Failed: $last_error"
                else
                    # Decompress to temp file
                    if gunzip -c "$temp_gz" > "$temp_ext" 2>/dev/null; then
                        # Validate decompressed file
                        if validate_extension_file "$temp_ext"; then
                            # Atomic move to final location
                            if mv -f "$temp_ext" "$output"; then
                                rm -f "$temp_gz" 2>/dev/null || true
                                # Set correct permissions
                                chmod 644 "$output"
                                chown "${RUNNER_USER}:${RUNNER_USER}" "$output"
                                success=true
                                break
                            else
                                last_error="Failed to move file to final location"
                            fi
                        else
                            last_error="Decompressed file appears invalid (too small)"
                        fi
                    else
                        last_error="Failed to decompress (file may be corrupted)"
                    fi
                fi
            else
                # Determine error type for better messaging
                case $curl_exit in
                    6)  last_error="Could not resolve host" ;;
                    7)  last_error="Failed to connect to host" ;;
                    22) last_error="HTTP error (status: $http_code)" ;;
                    28) last_error="Operation timed out after ${EXT_DOWNLOAD_TIMEOUT}s" ;;
                    35) last_error="SSL/TLS handshake failed" ;;
                    56) last_error="Network data receive error" ;;
                    *)  last_error="curl failed (exit: $curl_exit, http: $http_code)" ;;
                esac
                log_debug "    Failed: $last_error"
            fi

            # Clean up failed attempt
            rm -f "$temp_gz" "$temp_ext" 2>/dev/null || true

            # Calculate backoff for next attempt
            if [ $attempt -lt $EXT_MAX_RETRIES ]; then
                local backoff
                backoff=$(calculate_ext_backoff $attempt)
                log_debug "    Retrying in ${backoff}s..."
                sleep "$backoff"
            fi
        done

        if [ "$success" = true ]; then
            local size
            size=$(stat -c%s "$output" 2>/dev/null || echo "?")
            log_success "  ${name} installed (${size} bytes)"
            ((installed_count++))
            return 0
        else
            log_error "  Failed to install ${name} after $EXT_MAX_RETRIES attempts"
            log_error "    Last error: $last_error"
            log_error "    URL: $url"
            return 1
        fi
    }

    # Download core extensions from main repository
    log_info "Installing core DuckDB extensions from main repository..."
    for ext in "${CORE_EXTENSIONS[@]}"; do
        if ! download_duckdb_extension "$ext" "$MAIN_REPO"; then
            failed_extensions+=("$ext")
        fi
    done

    # Download community extensions
    log_info "Installing community DuckDB extensions..."
    for ext in "${COMMUNITY_EXTENSIONS[@]}"; do
        if ! download_duckdb_extension "$ext" "$COMMUNITY_REPO"; then
            failed_extensions+=("$ext")
        fi
    done

    # Write version file with error handling
    log_info "Writing DuckDB version file..."
    if ! sudo -u "$RUNNER_USER" bash -c "echo '${DUCKDB_VERSION}' > '${DUCKDB_BASE}/version'" 2>/dev/null; then
        log_warn "Failed to write version file (non-critical)"
    fi

    # Final verification
    local ext_count
    ext_count=$(find "$EXTENSIONS_DIR" -name "*.duckdb_extension" -type f 2>/dev/null | wc -l)

    echo ""
    log_info "DuckDB Extensions Summary:"
    log_info "  Directory: $EXTENSIONS_DIR"
    log_info "  Total installed: $ext_count"

    # List installed extensions
    for ext_file in "$EXTENSIONS_DIR"/*.duckdb_extension; do
        if [ -f "$ext_file" ]; then
            local name size
            name=$(basename "$ext_file" .duckdb_extension)
            size=$(stat -c%s "$ext_file" 2>/dev/null || echo "?")
            log_success "    $name ($size bytes)"
        fi
    done

    if [ ${#failed_extensions[@]} -gt 0 ]; then
        log_error "  FAILED extensions: ${failed_extensions[*]}"
        log_error ""
        log_error "DuckDB setup FAILED - some extensions could not be installed"
        log_error ""
        log_error "Remediation steps:"
        log_error "  1. Check network connectivity:"
        log_error "     curl -I https://extensions.duckdb.org/"
        log_error ""
        log_error "  2. Manual download (as ${RUNNER_USER}):"
        for ext in "${failed_extensions[@]}"; do
            local repo="$MAIN_REPO"
            case "$ext" in
                h3|rapidfuzz|datasketches) repo="$COMMUNITY_REPO" ;;
            esac
            log_error "     curl -sSfL '${repo}/${ext}.duckdb_extension.gz' | gunzip > '${EXTENSIONS_DIR}/${ext}.duckdb_extension'"
        done
        exit 1
    fi

    # All extensions are required
    local total_expected=$(( ${#CORE_EXTENSIONS[@]} + ${#COMMUNITY_EXTENSIONS[@]} ))
    if [ "$ext_count" -lt "$total_expected" ]; then
        log_error "Expected $total_expected extensions, found $ext_count"
        exit 1
    fi

    log_success "DuckDB extensions setup complete (${ext_count} extensions)"
}

# Print summary
print_summary() {
    echo ""
    echo "========================================"
    log_success "Self-Hosted Runner Host Setup Complete!"
    echo "========================================"
    echo ""
    echo "Installed versions:"
    echo "  - Go: $(/usr/local/go/bin/go version | awk '{print $3}')"
    echo "  - gotestsum: $(gotestsum --version 2>/dev/null || echo 'installed')"
    echo "  - gocovmerge: $(command -v gocovmerge &>/dev/null && echo 'installed' || echo 'not found')"
    echo "  - benchstat: $(command -v benchstat &>/dev/null && echo 'installed' || echo 'not found')"
    echo "  - Node.js: $(node --version)"
    echo "  - npm: $(npm --version)"
    echo "  - Docker: $(docker --version | awk '{print $3}' | sed 's/,//')"
    echo "  - scc: $(scc --version 2>/dev/null | head -1 || echo 'installed')"

    # Check WebGL deps using same method as install function
    webgl_status="installed"
    for pkg in libegl1 libgles2 libgl1-mesa-dri; do
        if ! dpkg-query -W -f='${Status}' "$pkg" 2>/dev/null | grep -q "install ok installed"; then
            webgl_status="not found"
            break
        fi
    done
    echo "  - WebGL deps: $webgl_status"

    # Check Playwright deps
    playwright_status="installed"
    if [ ! -f "/usr/lib/x86_64-linux-gnu/libatk-1.0.so.0" ]; then
        playwright_status="not found (libatk missing)"
    elif [ ! -f "/usr/lib/x86_64-linux-gnu/libnss3.so" ]; then
        playwright_status="not found (libnss3 missing)"
    fi
    echo "  - Playwright deps: $playwright_status"
    echo ""
    echo "Runner user: ${RUNNER_USER}"
    echo "  - Home: /home/${RUNNER_USER}"
    echo "  - Groups: $(groups ${RUNNER_USER})"
    echo ""
    echo "Next steps:"
    echo ""
    echo "  IMPORTANT: Follow these steps in order. Do NOT skip the 'exit' step!"
    echo ""
    echo "  1. Switch to runner user:"
    echo "     sudo su - ${RUNNER_USER}"
    echo ""
    echo "  2. Download and configure runner (as ${RUNNER_USER}):"
    echo "     mkdir actions-runner && cd actions-runner"
    echo "     curl -o actions-runner-linux-x64-${RUNNER_VERSION}.tar.gz -L \\"
    echo "       https://github.com/actions/runner/releases/download/v${RUNNER_VERSION}/actions-runner-linux-x64-${RUNNER_VERSION}.tar.gz"
    echo "     tar xzf ./actions-runner-linux-x64-${RUNNER_VERSION}.tar.gz"
    echo "     ./config.sh --url https://github.com/tomtom215/cartographus --token YOUR_TOKEN"
    echo ""
    echo "  3. Make directory accessible for service installation (as ${RUNNER_USER}):"
    echo "     chmod 755 /home/${RUNNER_USER}/actions-runner"
    echo ""
    echo "  4. Exit back to your original user (CRITICAL STEP):"
    echo "     exit"
    echo ""
    echo "  5. Install and start service (from your original user with sudo):"
    echo "     cd /home/${RUNNER_USER}/actions-runner"
    echo "     sudo ./svc.sh install ${RUNNER_USER}"
    echo "     sudo ./svc.sh start"
    echo "     sudo ./svc.sh status"
    echo ""
    echo "  Why step 4 is critical:"
    echo "  - The svc.sh script must be run with sudo from OUTSIDE the runner user session"
    echo "  - Running 'sudo' from within 'sudo su - username' causes permission errors"
    echo "  - The service needs to be installed by a user with proper sudo privileges"
    echo ""
    echo "  Why step 3 is needed:"
    echo "  - By default, the actions-runner directory has 700 permissions (owner-only access)"
    echo "  - chmod 755 allows your admin user to access it for service installation"
    echo "  - Without this, 'cd /home/${RUNNER_USER}/actions-runner' will fail with permission denied"
    echo ""
    echo "See docs/SELF_HOSTED_RUNNER.md for detailed instructions."
    echo ""
}

# Uninstall runner
uninstall_runner() {
    echo "========================================"
    echo "GitHub Actions Runner Removal"
    echo "========================================"
    echo ""

    RUNNER_USER="${1:-github-runner}"
    RUNNER_HOME="/home/${RUNNER_USER}"
    RUNNER_DIR="${RUNNER_HOME}/actions-runner"

    log_info "This will remove the GitHub Actions runner from this host"
    echo ""
    echo "Runner user: ${RUNNER_USER}"
    echo "Runner directory: ${RUNNER_DIR}"
    echo ""

    # Check if runner directory exists
    if [ ! -d "$RUNNER_DIR" ]; then
        log_error "Runner directory not found: ${RUNNER_DIR}"
        log_info "Nothing to uninstall"
        exit 0
    fi

    # Step 1: Stop and uninstall service
    log_info "Step 1: Stopping and uninstalling runner service..."

    # Find the service name
    SERVICE_NAME=$(systemctl list-units --type=service --all | grep "actions.runner.tomtom215-map" | awk '{print $1}' | head -1 || true)

    if [ -n "$SERVICE_NAME" ]; then
        log_info "Found service: ${SERVICE_NAME}"

        # Stop service
        if systemctl is-active --quiet "$SERVICE_NAME"; then
            log_info "Stopping service..."
            systemctl stop "$SERVICE_NAME" || true
        fi

        # Uninstall service using svc.sh (must run as root, not as runner user)
        if [ -f "${RUNNER_DIR}/svc.sh" ]; then
            log_info "Uninstalling service..."
            cd "$RUNNER_DIR"
            ./svc.sh uninstall || true
            cd -
        fi

        log_success "Service stopped and uninstalled"
    else
        log_warn "No active service found"
    fi

    # Step 2: Remove runner from GitHub
    log_info "Step 2: Removing runner from GitHub..."
    echo ""
    log_warn "To remove the runner from GitHub, you need a removal token."
    echo ""
    echo "Get a removal token from:"
    echo "  https://github.com/tomtom215/cartographus/settings/actions/runners"
    echo ""
    echo "Click on your runner, then click 'Remove' to get the token."
    echo ""
    read -r -p "Enter removal token (or press Enter to skip): " REMOVAL_TOKEN

    if [ -n "$REMOVAL_TOKEN" ]; then
        if [ -f "${RUNNER_DIR}/config.sh" ]; then
            log_info "Removing runner from GitHub..."
            cd "$RUNNER_DIR"
            sudo -u "$RUNNER_USER" ./config.sh remove --token "$REMOVAL_TOKEN" || log_warn "Failed to remove from GitHub (may be already removed)"
            cd -
            log_success "Runner removed from GitHub"
        else
            log_warn "config.sh not found, skipping GitHub removal"
        fi
    else
        log_warn "Skipping GitHub removal (you can manually remove it from GitHub UI)"
    fi

    # Step 3: Delete runner directory
    echo ""
    log_info "Step 3: Deleting runner directory..."
    read -r -p "Delete ${RUNNER_DIR}? [y/N]: " DELETE_DIR

    if [[ "$DELETE_DIR" =~ ^[Yy]$ ]]; then
        log_info "Deleting ${RUNNER_DIR}..."
        rm -rf "$RUNNER_DIR"
        log_success "Runner directory deleted"
    else
        log_warn "Keeping runner directory: ${RUNNER_DIR}"
    fi

    # Step 4: Remove user
    echo ""
    log_info "Step 4: Remove runner user..."
    echo ""
    log_warn "The ${RUNNER_USER} user can be removed, but this will delete:"
    echo "  - User home directory: ${RUNNER_HOME}"
    echo "  - Any other files owned by ${RUNNER_USER}"
    echo ""
    read -r -p "Remove ${RUNNER_USER} user? [y/N]: " DELETE_USER

    if [[ "$DELETE_USER" =~ ^[Yy]$ ]]; then
        log_info "Removing ${RUNNER_USER} user..."
        userdel -r "$RUNNER_USER" 2>/dev/null || log_warn "Failed to remove user (may not exist)"
        log_success "User removed"
    else
        log_warn "Keeping ${RUNNER_USER} user"
    fi

    # Summary
    echo ""
    echo "========================================"
    log_success "Runner Removal Complete!"
    echo "========================================"
    echo ""
    echo "What was removed:"
    echo "  - Runner service (if running)"
    if [ -n "$REMOVAL_TOKEN" ]; then
        echo "  - Runner from GitHub"
    fi
    if [[ "$DELETE_DIR" =~ ^[Yy]$ ]]; then
        echo "  - Runner directory: ${RUNNER_DIR}"
    fi
    if [[ "$DELETE_USER" =~ ^[Yy]$ ]]; then
        echo "  - User: ${RUNNER_USER}"
    fi
    echo ""
    echo "What was NOT removed (may be used by other applications):"
    echo "  - Go ($(command -v go &>/dev/null && go version || echo 'not installed'))"
    echo "  - Node.js ($(command -v node &>/dev/null && node --version || echo 'not installed'))"
    echo "  - Docker ($(command -v docker &>/dev/null && docker --version || echo 'not installed'))"
    echo ""
    log_info "To remove these manually, use standard package management tools"
    echo ""
}

# Main installation flow
main_install() {
    echo "========================================"
    echo "GitHub Actions Self-Hosted Runner Setup"
    echo "========================================"
    echo ""

    check_root
    acquire_lock
    check_os
    check_required_commands
    check_disk_space
    check_network

    echo ""
    log_info "Starting installation..."
    echo ""

    update_system
    # Optional: Enable automatic security updates (disabled by default for CI stability)
    # Uncomment the next line if you prefer automatic security patches over strict reproducibility
    # setup_auto_security_updates
    install_core_deps
    install_scc
    install_webgl_deps
    install_playwright_deps
    install_cross_toolchains
    install_osxcross_deps
    install_go
    install_go_tools "install"
    install_nodejs
    install_docker
    setup_docker_buildx
    create_runner_user

    # Setup DuckDB extensions (self-contained, no external script needed)
    setup_duckdb_extensions

    print_summary
}

# Update existing installation without breaking services
# This is safe to run on a live runner - it won't interrupt running jobs
main_update() {
    echo "========================================"
    echo "GitHub Actions Runner - Dependency Update"
    echo "========================================"
    echo ""

    check_root
    acquire_lock
    check_os
    check_network

    echo ""
    log_info "Starting dependency update..."
    log_info "Note: This will not interrupt running jobs"
    echo ""

    # Verify basic installation exists
    if ! id "$RUNNER_USER" &>/dev/null; then
        log_error "Runner user '$RUNNER_USER' does not exist"
        log_error "Run 'install' first to set up the runner"
        exit 1
    fi

    local runner_home="/home/${RUNNER_USER}"
    if [ ! -d "$runner_home" ]; then
        log_error "Runner home directory does not exist: $runner_home"
        exit 1
    fi

    # Update system packages (security updates only)
    log_info "Checking for system updates..."
    apt-get update -qq
    apt-get upgrade -y -qq

    # Update Go to latest stable if auto mode
    if [ "$GO_VERSION" = "auto" ]; then
        log_info "Checking for Go updates..."
        local latest_go
        latest_go=$(fetch_latest_go_version)
        if [ -n "$latest_go" ]; then
            local current_go=""
            if command -v go &>/dev/null; then
                current_go=$(go version | awk '{print $3}' | sed 's/go//')
            fi

            if [ "$current_go" != "$latest_go" ]; then
                log_info "Upgrading Go from $current_go to $latest_go..."
                GO_VERSION="$latest_go"
                install_go
            else
                log_success "Go is already at latest version: $current_go"
            fi
        fi
    fi

    # Update Go tools
    log_info "Updating Go tools..."
    install_go_tools "update"

    # Refresh DuckDB extensions
    log_info "Refreshing DuckDB extensions..."
    setup_duckdb_extensions

    # Fix permissions on runner directories
    log_info "Verifying directory permissions..."
    local dirs_to_check=(
        "$runner_home"
        "$runner_home/.cache"
        "$runner_home/.config"
        "$runner_home/.duckdb"
        "$runner_home/go"
    )

    for dir in "${dirs_to_check[@]}"; do
        if [ -d "$dir" ]; then
            if ! verify_permissions "$dir" "$RUNNER_USER" "755"; then
                log_info "Fixing permissions on $dir..."
                chown -R "${RUNNER_USER}:${RUNNER_USER}" "$dir"
                chmod 755 "$dir"
            fi
        fi
    done

    # Verify Docker access
    log_info "Verifying Docker access..."
    if ! sudo -u "$RUNNER_USER" docker ps &>/dev/null; then
        log_warn "Docker access issue detected, fixing..."
        usermod -aG docker "$RUNNER_USER"
        log_warn "Docker group membership updated - runner service may need restart"
    fi

    echo ""
    echo "========================================"
    log_success "Dependency Update Complete!"
    echo "========================================"
    echo ""
    echo "Updated components:"
    echo "  - System packages (security updates)"
    if [ "$GO_VERSION" != "auto" ]; then
        echo "  - Go: $(/usr/local/go/bin/go version 2>/dev/null | awk '{print $3}' || echo 'N/A')"
    fi
    echo "  - Go tools (gotestsum, gocovmerge, benchstat)"
    echo "  - DuckDB extensions"
    echo "  - Directory permissions"
    echo ""
    echo "Note: If the runner service is running, it will continue with the"
    echo "updated dependencies for the next job."
    echo ""
}

# Verify installation without reinstalling
verify_installation() {
    echo "========================================"
    echo "Verifying Self-Hosted Runner Installation"
    echo "========================================"
    echo ""

    local all_ok=true

    # Check Go
    if command -v go &> /dev/null; then
        log_success "Go: $(/usr/local/go/bin/go version 2>/dev/null | awk '{print $3}' || echo 'installed')"
    else
        log_error "Go: NOT INSTALLED"
        all_ok=false
    fi

    # Check Node.js
    if command -v node &> /dev/null; then
        log_success "Node.js: $(node --version)"
    else
        log_error "Node.js: NOT INSTALLED"
        all_ok=false
    fi

    # Check Docker
    if command -v docker &> /dev/null; then
        log_success "Docker: $(docker --version 2>/dev/null | awk '{print $3}' | sed 's/,//' || echo 'installed')"
    else
        log_error "Docker: NOT INSTALLED"
        all_ok=false
    fi

    # Check Go tools
    for tool in gotestsum gocovmerge benchstat; do
        if command -v $tool &> /dev/null; then
            log_success "$tool: installed"
        else
            log_error "$tool: NOT INSTALLED"
            all_ok=false
        fi
    done

    # Check scc
    if command -v scc &> /dev/null; then
        log_success "scc: installed"
    else
        log_error "scc: NOT INSTALLED"
        all_ok=false
    fi

    # Check cross-compilation toolchains
    for tool in aarch64-linux-gnu-gcc x86_64-w64-mingw32-gcc; do
        if command -v $tool &> /dev/null; then
            log_success "$tool: installed"
        else
            log_warn "$tool: not installed (optional, for cross-compilation)"
        fi
    done

    # Check WebGL deps
    local webgl_ok=true
    for pkg in libegl1 libgles2 libgl1-mesa-dri; do
        if ! dpkg-query -W -f='${Status}' "$pkg" 2>/dev/null | grep -q "install ok installed"; then
            webgl_ok=false
            break
        fi
    done
    if [ "$webgl_ok" = true ]; then
        log_success "WebGL deps: installed"
    else
        log_error "WebGL deps: MISSING"
        all_ok=false
    fi

    # Check Playwright deps (the critical ones)
    local playwright_ok=true
    for lib in libatk-1.0.so.0 libatk-bridge-2.0.so.0 libcups.so.2 libdrm.so.2 libgbm.so.1 libnss3.so; do
        if [ ! -f "/usr/lib/x86_64-linux-gnu/$lib" ]; then
            log_error "Playwright: missing $lib"
            playwright_ok=false
            all_ok=false
        fi
    done
    if [ "$playwright_ok" = true ]; then
        log_success "Playwright deps: installed"
    fi

    # Check runner user
    if id "github-runner" &>/dev/null; then
        log_success "Runner user: github-runner exists"
        if groups "github-runner" | grep -q docker; then
            log_success "Docker group: github-runner is member"
        else
            log_error "Docker group: github-runner is NOT a member"
            all_ok=false
        fi
    else
        log_warn "Runner user: github-runner does not exist (create with install command)"
    fi

    echo ""
    if [ "$all_ok" = true ]; then
        log_success "All required components are installed!"
        exit 0
    else
        log_error "Some components are missing. Run: sudo bash scripts/setup-runner-host.sh install"
        exit 1
    fi
}

# Clear all caches to fix dependency version mismatches
# This matches the cache invalidation logic in .github/workflows/_test.yml
clear_caches() {
    echo "========================================"
    echo "Clearing Dependency Caches"
    echo "========================================"
    echo ""

    local runner_user="${1:-github-runner}"

    log_info "This will clear all cached dependencies to fix version mismatches."
    log_info "Use this when CI fails due to DuckDB/Go module cache issues."
    echo ""

    # Clear Go module cache
    log_info "Clearing Go module cache..."
    if [ -d "/home/${runner_user}/go/pkg/mod" ]; then
        rm -rf "/home/${runner_user}/go/pkg/mod"
        log_success "Go module cache cleared"
    else
        log_info "No Go module cache found at /home/${runner_user}/go/pkg/mod"
    fi

    # Clear system Go module cache
    if command -v go &> /dev/null; then
        log_info "Running go clean -modcache..."
        sudo -u "$runner_user" go clean -modcache 2>/dev/null || true
        log_success "go clean -modcache completed"
    fi

    # Clear DuckDB extensions cache
    log_info "Clearing DuckDB extensions cache..."
    if [ -d "/home/${runner_user}/.duckdb/extensions" ]; then
        rm -rf "/home/${runner_user}/.duckdb/extensions"
        log_success "DuckDB extensions cache cleared"
    else
        log_info "No DuckDB extensions cache found"
    fi

    # Clear CI cache invalidation marker
    log_info "Clearing CI cache invalidation marker..."
    local cache_marker="/home/${runner_user}/.cache/${CI_CACHE_MARKER_NAME}"
    if [ -f "$cache_marker" ]; then
        rm -f "$cache_marker"
        log_success "Cache marker cleared: ${cache_marker}"
    else
        log_info "No cache marker found at ${cache_marker}"
    fi

    # Clear workflow tool caches
    log_info "Clearing workflow tool caches..."
    if [ -d "/home/${runner_user}/actions-runner/_work/_tool" ]; then
        rm -rf "/home/${runner_user}/actions-runner/_work/_tool"
        log_success "Workflow tool cache cleared"
    else
        log_info "No workflow tool cache found"
    fi

    echo ""
    log_success "All caches cleared!"
    echo ""
    echo "Next steps:"
    echo "  1. Re-trigger your CI workflow"
    echo "  2. Fresh dependencies will be downloaded"
    echo "  3. DuckDB extensions will be reinstalled"
    echo ""
}

# Health check - verify runner can actually execute workflows
health_check() {
    echo "========================================"
    echo "Runner Health Check"
    echo "========================================"
    echo ""

    local all_ok=true
    local runner_user="${1:-github-runner}"

    # 1. Check runner service is running
    log_info "Checking runner service..."
    local service_name
    service_name=$(systemctl list-units --type=service --all 2>/dev/null | grep "actions.runner.tomtom215-map" | awk '{print $1}' | head -1 || true)

    if [ -n "$service_name" ] && systemctl is-active --quiet "$service_name" 2>/dev/null; then
        log_success "Runner service is running: ${service_name}"
    else
        log_error "Runner service is NOT running"
        log_info "Start with: sudo systemctl start actions.runner.tomtom215-map.*.service"
        all_ok=false
    fi

    # 2. Check GitHub connectivity
    log_info "Checking GitHub API connectivity..."
    # Use -L to follow redirects, longer timeout for API
    if curl -sfL --max-time 15 "https://api.github.com/zen" -o /dev/null 2>/dev/null; then
        log_success "GitHub API is reachable"
    else
        log_error "Cannot reach GitHub API"
        log_error "Debug: Try 'curl -vL https://api.github.com/zen'"
        all_ok=false
    fi

    # 3. Check Docker daemon
    log_info "Checking Docker daemon..."
    if docker info &>/dev/null; then
        log_success "Docker daemon is running"
    else
        log_error "Docker daemon is not accessible"
        all_ok=false
    fi

    # 4. Check runner user can access Docker
    log_info "Checking Docker access for ${runner_user}..."
    if sudo -u "$runner_user" docker ps &>/dev/null; then
        log_success "${runner_user} can access Docker"
    else
        log_error "${runner_user} cannot access Docker"
        log_info "Fix with: sudo usermod -aG docker ${runner_user}"
        all_ok=false
    fi

    # 5. Check Go is accessible
    log_info "Checking Go installation..."
    if command -v go &>/dev/null; then
        GO_VER=$(go version | awk '{print $3}')
        log_success "Go is installed: ${GO_VER}"
    else
        log_error "Go is not installed or not in PATH"
        all_ok=false
    fi

    # 6. Check DuckDB extensions
    log_info "Checking DuckDB extensions..."
    local ext_dir="/home/${runner_user}/.duckdb/extensions"
    if [ -d "$ext_dir" ] && [ "$(ls -A "$ext_dir" 2>/dev/null)" ]; then
        local ext_count
        ext_count=$(find "$ext_dir" -name "*.duckdb_extension" 2>/dev/null | wc -l)
        log_success "DuckDB extensions installed: ${ext_count} extensions"
    else
        log_warn "DuckDB extensions not found (will be installed on first workflow run)"
    fi

    # 7. Quick compile test
    log_info "Running quick Go compile test..."
    if echo 'package main; func main() {}' | go run -; then
        log_success "Go compilation works"
    else
        log_error "Go compilation failed"
        all_ok=false
    fi

    # 8. Check disk space
    log_info "Checking disk space..."
    local available_gb
    available_gb=$(df -BG / --output=avail | tail -1 | tr -d 'G ')
    if [ "$available_gb" -ge 10 ]; then
        log_success "Disk space OK: ${available_gb}GB available"
    else
        log_warn "Low disk space: ${available_gb}GB available (recommend 10GB+)"
    fi

    echo ""
    if [ "$all_ok" = true ]; then
        log_success "Health check PASSED - runner is ready for workflows"
        exit 0
    else
        log_error "Health check FAILED - some components need attention"
        exit 1
    fi
}

# Main function - parse arguments and dispatch
main() {
    local command=""
    local positional_args=()

    # Parse options
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --dry-run)
                DRY_RUN=true
                shift
                ;;
            --verbose|-v)
                VERBOSE=true
                shift
                ;;
            --log)
                LOG_FILE="${2:-/var/log/setup-runner-host.log}"
                shift 2
                ;;
            -*)
                # Check if it's a help flag
                if [[ "$1" == "-h" || "$1" == "--help" ]]; then
                    positional_args+=("help")
                else
                    log_error "Unknown option: $1"
                    exit 1
                fi
                shift
                ;;
            *)
                positional_args+=("$1")
                shift
                ;;
        esac
    done

    command="${positional_args[0]:-}"

    # Show dry-run banner if enabled
    if [ "$DRY_RUN" = true ]; then
        echo ""
        echo -e "${YELLOW}========================================${NC}"
        echo -e "${YELLOW}  DRY-RUN MODE - No changes will be made${NC}"
        echo -e "${YELLOW}========================================${NC}"
        echo ""
    fi

    case "$command" in
        install)
            main_install
            ;;
        update|upgrade)
            main_update
            ;;
        verify|check)
            verify_installation
            ;;
        uninstall|remove)
            check_root
            uninstall_runner "${positional_args[1]:-github-runner}"
            ;;
        clear-cache|cache-clear)
            check_root
            clear_caches "${positional_args[1]:-github-runner}"
            ;;
        health|health-check)
            health_check "${positional_args[1]:-github-runner}"
            ;;
        -h|--help|help)
            echo "Usage: sudo bash scripts/setup-runner-host.sh [OPTIONS] [COMMAND]"
            echo ""
            echo "Commands:"
            echo "  install       Install all prerequisites and create runner user"
            echo "  update        Update all dependencies without breaking services"
            echo "  verify        Check if all prerequisites are installed"
            echo "  uninstall     Remove runner, service, and optionally user"
            echo "  clear-cache   Clear all dependency caches (Go modules, DuckDB, etc.)"
            echo "  health-check  Verify runner can execute workflows"
            echo "  help          Show this help message"
            echo ""
            echo "Options:"
            echo "  --dry-run     Preview what would be done without making changes"
            echo "  --verbose     Show detailed debug output"
            echo "  --log FILE    Write logs to FILE (default: /var/log/setup-runner-host.log)"
            echo ""
            echo "Examples:"
            echo "  sudo bash scripts/setup-runner-host.sh install"
            echo "  sudo bash scripts/setup-runner-host.sh update"
            echo "  sudo bash scripts/setup-runner-host.sh --dry-run install"
            echo "  bash scripts/setup-runner-host.sh verify"
            echo "  sudo bash scripts/setup-runner-host.sh clear-cache"
            echo "  bash scripts/setup-runner-host.sh health-check"
            echo ""
            echo "Version Information:"
            echo "  Go:       ${GO_VERSION}"
            echo "  Node.js:  ${NODE_VERSION}.x"
            echo "  Runner:   ${RUNNER_VERSION}"
            echo ""
            exit 0
            ;;
        "")
            log_error "No command specified"
            echo ""
            echo "Usage: sudo bash scripts/setup-runner-host.sh [OPTIONS] [COMMAND]"
            echo ""
            echo "Commands: install, update, verify, uninstall, clear-cache, health-check, help"
            echo ""
            echo "For more information, run:"
            echo "  bash scripts/setup-runner-host.sh help"
            echo ""
            exit 1
            ;;
        *)
            log_error "Unknown command: $command"
            echo ""
            echo "Valid commands: install, update, verify, uninstall, clear-cache, health-check, help"
            echo ""
            exit 1
            ;;
    esac
}

# Run main function
main "$@"
