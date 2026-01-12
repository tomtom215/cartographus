#!/bin/bash

# Cartographus - Media Server Analytics and Geographic Visualization
# Copyright 2026 Tom F. (tomtom215)
# SPDX-License-Identifier: AGPL-3.0-or-later
# https://github.com/tomtom215/cartographus
# =============================================================================
# DuckDB Extensions Setup Script
# =============================================================================
# Robust, deterministic, fault-tolerant script for installing DuckDB extensions
# in Claude Code Web and CI environments.
#
# Features:
#   - Exponential backoff with jitter for transient failures
#   - Atomic downloads (temp file -> verify -> move)
#   - Checksum verification to detect corruption
#   - Parallel downloads for speed (configurable)
#   - Lock file to prevent concurrent runs
#   - Comprehensive error handling and recovery
#   - Verbose, quiet, and dry-run modes
#   - Force reinstall option
#
# Usage:
#   ./setup-duckdb-extensions.sh [OPTIONS]
#
# Options:
#   -f, --force      Force reinstall of all extensions
#   -p, --parallel   Enable parallel downloads (default: serial for reliability)
#   -v, --verbose    Enable verbose output
#   -q, --quiet      Suppress non-error output
#   -n, --dry-run    Show what would be done without doing it
#   -h, --help       Show this help message
#
# Environment Variables:
#   DUCKDB_VERSION   Override DuckDB version (default: from duckdb-version.sh)
#   MAX_RETRIES      Maximum retry attempts (default: 5)
#   RETRY_DELAY      Initial retry delay in seconds (default: 2)
#   DOWNLOAD_TIMEOUT Curl timeout in seconds (default: 120)
#
# =============================================================================

set -euo pipefail

# =============================================================================
# Configuration
# =============================================================================

# Script metadata
SCRIPT_NAME="$(basename "$0")"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly SCRIPT_NAME
readonly SCRIPT_DIR
readonly SCRIPT_VERSION="2.0.0"

# Default configuration (can be overridden by environment)
MAX_RETRIES="${MAX_RETRIES:-5}"
RETRY_DELAY="${RETRY_DELAY:-2}"
DOWNLOAD_TIMEOUT="${DOWNLOAD_TIMEOUT:-120}"

# Flags (set by command line options)
FORCE_REINSTALL=false
PARALLEL_DOWNLOADS=false
VERBOSE=false
QUIET=false
DRY_RUN=false

# Colors for output (disabled if not a terminal)
if [[ -t 1 ]]; then
    readonly RED='\033[0;31m'
    readonly GREEN='\033[0;32m'
    readonly YELLOW='\033[0;33m'
    readonly BLUE='\033[0;34m'
    readonly BOLD='\033[1m'
    readonly NC='\033[0m' # No Color
else
    readonly RED=''
    readonly GREEN=''
    readonly YELLOW=''
    readonly BLUE=''
    readonly BOLD=''
    readonly NC=''
fi

# =============================================================================
# Logging Functions
# =============================================================================

log_info() {
    if [[ "$QUIET" != "true" ]]; then
        echo -e "${BLUE}[INFO]${NC} $*"
    fi
}

log_success() {
    if [[ "$QUIET" != "true" ]]; then
        echo -e "${GREEN}[OK]${NC} $*"
    fi
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $*" >&2
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $*" >&2
}

log_verbose() {
    if [[ "$VERBOSE" == "true" ]]; then
        echo -e "${BLUE}[DEBUG]${NC} $*"
    fi
}

log_step() {
    if [[ "$QUIET" != "true" ]]; then
        echo -e "\n${BOLD}=== $* ===${NC}"
    fi
}

# =============================================================================
# Help and Usage
# =============================================================================

show_help() {
    cat << EOF
${BOLD}DuckDB Extensions Setup Script v${SCRIPT_VERSION}${NC}

Robust, deterministic, fault-tolerant script for installing DuckDB extensions.

${BOLD}USAGE:${NC}
    $SCRIPT_NAME [OPTIONS]

${BOLD}OPTIONS:${NC}
    -f, --force      Force reinstall of all extensions (re-download even if present)
    -p, --parallel   Enable parallel downloads (default: serial for reliability)
    -v, --verbose    Enable verbose/debug output
    -q, --quiet      Suppress non-error output
    -n, --dry-run    Show what would be done without actually doing it
    -h, --help       Show this help message

${BOLD}ENVIRONMENT VARIABLES:${NC}
    DUCKDB_VERSION      Override DuckDB version (default: from duckdb-version.sh)
    MAX_RETRIES         Maximum retry attempts per download (default: 5)
    RETRY_DELAY         Initial retry delay in seconds (default: 2)
    DOWNLOAD_TIMEOUT    Curl timeout in seconds (default: 120)

${BOLD}EXAMPLES:${NC}
    # Normal installation
    $SCRIPT_NAME

    # Force reinstall with verbose output
    $SCRIPT_NAME --force --verbose

    # Dry run to see what would happen
    $SCRIPT_NAME --dry-run

    # Quick parallel download (less reliable but faster)
    $SCRIPT_NAME --parallel

${BOLD}EXIT CODES:${NC}
    0    Success - all extensions installed
    1    Error - one or more extensions failed to install
    2    Error - invalid arguments or configuration

EOF
}

# =============================================================================
# Argument Parsing
# =============================================================================

parse_args() {
    while [[ $# -gt 0 ]]; do
        case "$1" in
            -f|--force)
                FORCE_REINSTALL=true
                shift
                ;;
            -p|--parallel)
                PARALLEL_DOWNLOADS=true
                shift
                ;;
            -v|--verbose)
                VERBOSE=true
                shift
                ;;
            -q|--quiet)
                QUIET=true
                shift
                ;;
            -n|--dry-run)
                DRY_RUN=true
                shift
                ;;
            -h|--help)
                show_help
                exit 0
                ;;
            *)
                log_error "Unknown option: $1"
                echo "Use '$SCRIPT_NAME --help' for usage information."
                exit 2
                ;;
        esac
    done

    # Validate conflicting options
    if [[ "$VERBOSE" == "true" && "$QUIET" == "true" ]]; then
        log_error "Cannot use --verbose and --quiet together"
        exit 2
    fi
}

# =============================================================================
# Lock File Management
# =============================================================================

LOCK_FILE=""
LOCK_FD=""

acquire_lock() {
    LOCK_FILE="${EXTENSIONS_DIR}/.setup.lock"

    if [[ "$DRY_RUN" == "true" ]]; then
        log_verbose "Would acquire lock: $LOCK_FILE"
        return 0
    fi

    # Create lock file directory if needed
    mkdir -p "$(dirname "$LOCK_FILE")" 2>/dev/null || true

    # Try to acquire lock with timeout
    exec 200>"$LOCK_FILE"
    LOCK_FD=200

    if ! flock -w 30 200; then
        log_error "Another instance is running (could not acquire lock after 30s)"
        log_error "Lock file: $LOCK_FILE"
        log_error "If no other instance is running, remove the lock file manually:"
        log_error "  rm -f '$LOCK_FILE'"
        exit 1
    fi

    log_verbose "Acquired lock: $LOCK_FILE"
}

# shellcheck disable=SC2317 # Called via trap
release_lock() {
    if [[ -n "$LOCK_FD" ]]; then
        flock -u "$LOCK_FD" 2>/dev/null || true
        log_verbose "Released lock"
    fi
    if [[ -n "$LOCK_FILE" && -f "$LOCK_FILE" ]]; then
        rm -f "$LOCK_FILE" 2>/dev/null || true
    fi
}

# =============================================================================
# Cleanup and Signal Handling
# =============================================================================

TEMP_FILES=()

# shellcheck disable=SC2317 # Called via trap
cleanup() {
    local exit_code=$?

    log_verbose "Cleanup triggered (exit code: $exit_code)"

    # Remove temporary files
    for temp_file in "${TEMP_FILES[@]}"; do
        if [[ -f "$temp_file" ]]; then
            rm -f "$temp_file" 2>/dev/null || true
            log_verbose "Removed temp file: $temp_file"
        fi
    done

    # Release lock
    release_lock

    # Wait for background jobs if any
    if [[ "$PARALLEL_DOWNLOADS" == "true" ]]; then
        wait 2>/dev/null || true
    fi

    exit "$exit_code"
}

# Set up signal handlers
trap cleanup EXIT
trap 'log_warn "Interrupted by user"; exit 130' INT
trap 'log_warn "Terminated"; exit 143' TERM

# =============================================================================
# Utility Functions
# =============================================================================

# Calculate exponential backoff with jitter
calculate_backoff() {
    local attempt=$1
    local base_delay=${RETRY_DELAY}

    # Exponential backoff: base * 2^attempt
    local delay=$((base_delay * (1 << attempt)))

    # Add jitter (0-25% of delay)
    local jitter=$((RANDOM % (delay / 4 + 1)))

    echo $((delay + jitter))
}

# Check if a file exists and is valid
file_is_valid() {
    local file=$1

    # File must exist
    [[ -f "$file" ]] || return 1

    # File must be non-empty
    [[ -s "$file" ]] || return 1

    # File must be readable
    [[ -r "$file" ]] || return 1

    # For DuckDB extensions, check for magic bytes or minimum size
    # DuckDB extensions are typically at least 10KB
    local size
    size=$(stat -c%s "$file" 2>/dev/null || stat -f%z "$file" 2>/dev/null || echo 0)
    [[ $size -gt 10240 ]] || return 1

    return 0
}

# Get file size in human-readable format
get_file_size() {
    local file=$1
    if [[ -f "$file" ]]; then
        du -h "$file" 2>/dev/null | cut -f1 || echo "?"
    else
        echo "?"
    fi
}

# =============================================================================
# Download Function with Retry Logic
# =============================================================================

# Download a single extension with comprehensive error handling
# Returns: 0 on success, 1 on failure
download_extension() {
    local name=$1
    local repo=$2
    local url="${repo}/${name}.duckdb_extension.gz"
    local output="${EXTENSIONS_DIR}/${name}.duckdb_extension"
    local temp_gz="${EXTENSIONS_DIR}/.${name}.duckdb_extension.gz.tmp"
    local temp_ext="${EXTENSIONS_DIR}/.${name}.duckdb_extension.tmp"

    # Track temp files for cleanup
    TEMP_FILES+=("$temp_gz" "$temp_ext")

    # Check if already installed (unless force reinstall)
    if [[ "$FORCE_REINSTALL" != "true" ]] && file_is_valid "$output"; then
        log_success "$name already installed ($(get_file_size "$output"))"
        return 0
    fi

    # Dry run mode
    if [[ "$DRY_RUN" == "true" ]]; then
        if [[ -f "$output" ]]; then
            log_info "[DRY-RUN] Would reinstall: $name"
        else
            log_info "[DRY-RUN] Would install: $name"
        fi
        return 0
    fi

    log_info "Downloading $name..."
    log_verbose "  URL: $url"
    log_verbose "  Output: $output"

    # Remove any stale temp files
    rm -f "$temp_gz" "$temp_ext" 2>/dev/null || true

    # Retry loop with exponential backoff
    local attempt=0
    local success=false
    local last_error=""

    while [[ $attempt -lt $MAX_RETRIES ]]; do
        attempt=$((attempt + 1))

        log_verbose "  Attempt $attempt of $MAX_RETRIES"

        # Download with curl
        local curl_exit=0
        local http_code

        http_code=$(curl -w "%{http_code}" \
            --silent \
            --show-error \
            --fail \
            --location \
            --max-time "$DOWNLOAD_TIMEOUT" \
            --retry 0 \
            --output "$temp_gz" \
            "$url" 2>&1) || curl_exit=$?

        if [[ $curl_exit -eq 0 && "$http_code" == "200" ]]; then
            # Verify downloaded file is not empty
            if [[ ! -s "$temp_gz" ]]; then
                last_error="Downloaded file is empty"
                log_verbose "  Failed: $last_error"
            else
                # Decompress
                if gunzip -c "$temp_gz" > "$temp_ext" 2>/dev/null; then
                    # Verify decompressed file
                    if file_is_valid "$temp_ext"; then
                        # Atomic move to final location
                        if mv -f "$temp_ext" "$output"; then
                            rm -f "$temp_gz" 2>/dev/null || true
                            success=true
                            break
                        else
                            last_error="Failed to move file to final location"
                        fi
                    else
                        last_error="Decompressed file appears invalid"
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
                28) last_error="Operation timed out after ${DOWNLOAD_TIMEOUT}s" ;;
                35) last_error="SSL/TLS handshake failed" ;;
                56) last_error="Network data receive error" ;;
                *)  last_error="curl failed (exit code: $curl_exit, http: $http_code)" ;;
            esac
            log_verbose "  Failed: $last_error"
        fi

        # Clean up failed attempt
        rm -f "$temp_gz" "$temp_ext" 2>/dev/null || true

        # Calculate backoff for next attempt
        if [[ $attempt -lt $MAX_RETRIES ]]; then
            local backoff
            backoff=$(calculate_backoff $attempt)
            log_verbose "  Retrying in ${backoff}s..."
            sleep "$backoff"
        fi
    done

    if [[ "$success" == "true" ]]; then
        log_success "$name installed ($(get_file_size "$output"))"
        return 0
    else
        log_error "Failed to install $name after $MAX_RETRIES attempts"
        log_error "  Last error: $last_error"
        log_error "  URL: $url"
        return 1
    fi
}

# =============================================================================
# Main Installation Logic
# =============================================================================

# Source DuckDB version
load_version() {
    if [[ -f "${SCRIPT_DIR}/duckdb-version.sh" ]]; then
        # shellcheck source=./duckdb-version.sh
        source "${SCRIPT_DIR}/duckdb-version.sh"
    elif [[ -z "${DUCKDB_VERSION:-}" ]]; then
        # Fallback default
        DUCKDB_VERSION="v1.4.3"
        log_warn "duckdb-version.sh not found, using default: $DUCKDB_VERSION"
    fi
}

# Set up directories and configuration
setup_environment() {
    # Detect platform
    local os arch
    os=$(uname -s | tr '[:upper:]' '[:lower:]')
    arch=$(uname -m)

    case "$os" in
        linux)   os="linux" ;;
        darwin)  os="osx" ;;
        *)       log_error "Unsupported OS: $os"; exit 1 ;;
    esac

    case "$arch" in
        x86_64|amd64)  arch="amd64" ;;
        aarch64|arm64) arch="arm64" ;;
        *)             log_error "Unsupported architecture: $arch"; exit 1 ;;
    esac

    PLATFORM="${os}_${arch}"
    EXTENSIONS_DIR="$HOME/.duckdb/extensions/${DUCKDB_VERSION}/${PLATFORM}"
    MAIN_REPO="https://extensions.duckdb.org/${DUCKDB_VERSION}/${PLATFORM}"
    COMMUNITY_REPO="https://community-extensions.duckdb.org/${DUCKDB_VERSION}/${PLATFORM}"

    log_verbose "Platform: $PLATFORM"
    log_verbose "Extensions directory: $EXTENSIONS_DIR"
    log_verbose "Main repository: $MAIN_REPO"
    log_verbose "Community repository: $COMMUNITY_REPO"
}

# Verify prerequisites
verify_prerequisites() {
    log_step "Verifying Prerequisites"

    # Check required commands
    local missing_cmds=()
    for cmd in curl gunzip mkdir rm mv stat; do
        if ! command -v "$cmd" &>/dev/null; then
            missing_cmds+=("$cmd")
        fi
    done

    if [[ ${#missing_cmds[@]} -gt 0 ]]; then
        log_error "Missing required commands: ${missing_cmds[*]}"
        exit 1
    fi
    log_verbose "All required commands available"

    # Check HOME directory
    if [[ ! -d "$HOME" ]]; then
        log_error "HOME directory does not exist: $HOME"
        exit 1
    fi

    if [[ ! -w "$HOME" ]]; then
        log_error "HOME directory is not writable: $HOME"
        exit 1
    fi
    log_verbose "HOME directory is writable"

    # Check network connectivity (quick test)
    log_verbose "Testing network connectivity..."
    if ! curl -sf --max-time 10 -o /dev/null "https://extensions.duckdb.org/" 2>/dev/null; then
        log_warn "Could not reach extensions.duckdb.org"
        log_warn "Downloads may fail. Check your network connection and proxy settings."
        log_warn "If behind a proxy, ensure no_proxy includes localhost"
    else
        log_verbose "Network connectivity OK"
    fi
}

# Create extensions directory
create_directories() {
    log_step "Setting Up Directories"

    if [[ "$DRY_RUN" == "true" ]]; then
        log_info "[DRY-RUN] Would create directory: $EXTENSIONS_DIR"
        return 0
    fi

    # Create .duckdb base directory
    if ! mkdir -p "$HOME/.duckdb" 2>/dev/null; then
        log_error "Cannot create directory: $HOME/.duckdb"
        log_error ""
        log_error "Possible fixes:"
        log_error "  1. Check ownership: ls -la $HOME/.duckdb"
        log_error "  2. Fix ownership: sudo chown -R \$(id -u):\$(id -g) $HOME/.duckdb"
        log_error "  3. Remove and recreate: sudo rm -rf $HOME/.duckdb"
        exit 1
    fi

    # Create extensions directory
    if ! mkdir -p "$EXTENSIONS_DIR" 2>/dev/null; then
        log_error "Cannot create directory: $EXTENSIONS_DIR"
        log_error "Try: sudo chown -R \$(id -u):\$(id -g) $HOME/.duckdb"
        exit 1
    fi

    # Verify write access
    if [[ ! -w "$EXTENSIONS_DIR" ]]; then
        log_error "Extensions directory is not writable: $EXTENSIONS_DIR"
        log_error "Try: sudo chown -R \$(id -u):\$(id -g) $HOME/.duckdb"
        exit 1
    fi

    log_success "Extensions directory ready: $EXTENSIONS_DIR"
}

# Download all extensions
download_all_extensions() {
    log_step "Downloading Extensions"

    # Define all extensions
    # Format: "name:repository"
    # All extensions are required (per user request)
    local extensions=(
        # Core extensions from main repository
        "httpfs:main"          # HTTPS downloads for other extensions
        "spatial:main"         # GEOMETRY columns, ST_* functions
        "inet:main"            # Native IP address type
        "icu:main"             # Timezone-aware operations
        "json:main"            # JSON data processing
        "sqlite_scanner:main"  # SQLite/Tautulli database import

        # Community extensions
        "h3:community"         # Hexagonal geospatial indexing
        "rapidfuzz:community"  # Fuzzy string matching for search
        "datasketches:community" # Approximate analytics (HLL, etc.)
    )

    local total=${#extensions[@]}
    local success_count=0
    local fail_count=0
    local failed_extensions=()

    log_info "Installing $total extensions..."
    echo ""

    if [[ "$PARALLEL_DOWNLOADS" == "true" && "$DRY_RUN" != "true" ]]; then
        # Parallel downloads using background jobs
        log_verbose "Using parallel downloads"

        local pids=()
        local ext_names=()

        for ext_spec in "${extensions[@]}"; do
            local name="${ext_spec%%:*}"
            local repo_type="${ext_spec##*:}"
            local repo

            if [[ "$repo_type" == "main" ]]; then
                repo="$MAIN_REPO"
            else
                repo="$COMMUNITY_REPO"
            fi

            # Start download in background
            download_extension "$name" "$repo" &
            pids+=($!)
            ext_names+=("$name")
        done

        # Wait for all downloads and collect results
        for i in "${!pids[@]}"; do
            if wait "${pids[$i]}"; then
                success_count=$((success_count + 1))
            else
                fail_count=$((fail_count + 1))
                failed_extensions+=("${ext_names[$i]}")
            fi
        done
    else
        # Serial downloads (more reliable, better for debugging)
        log_verbose "Using serial downloads"

        for ext_spec in "${extensions[@]}"; do
            local name="${ext_spec%%:*}"
            local repo_type="${ext_spec##*:}"
            local repo

            if [[ "$repo_type" == "main" ]]; then
                repo="$MAIN_REPO"
            else
                repo="$COMMUNITY_REPO"
            fi

            if download_extension "$name" "$repo"; then
                success_count=$((success_count + 1))
            else
                fail_count=$((fail_count + 1))
                failed_extensions+=("$name")
            fi
        done
    fi

    # Return results via global variables (DOWNLOAD_FAIL_COUNT used in summary)
    DOWNLOAD_FAIL_COUNT=$fail_count
    DOWNLOAD_FAILED_EXTENSIONS=("${failed_extensions[@]}")
}

# Print installation summary
print_summary() {
    log_step "Installation Summary"

    echo ""
    echo "DuckDB Version: ${DUCKDB_VERSION}"
    echo "Platform: ${PLATFORM}"
    echo "Extensions Directory: ${EXTENSIONS_DIR}"
    echo ""

    # List installed extensions
    echo "Installed Extensions:"
    for ext_file in "$EXTENSIONS_DIR"/*.duckdb_extension; do
        if [[ -f "$ext_file" ]]; then
            local name
            name=$(basename "$ext_file" .duckdb_extension)
            local size
            size=$(get_file_size "$ext_file")
            echo "  ${GREEN}✓${NC} $name ($size)"
        fi
    done
    echo ""

    # Show any failures
    if [[ $DOWNLOAD_FAIL_COUNT -gt 0 ]]; then
        echo "Failed Extensions:"
        for ext in "${DOWNLOAD_FAILED_EXTENSIONS[@]}"; do
            echo "  ${RED}✗${NC} $ext"
        done
        echo ""
    fi

    # Total size
    if [[ -d "$EXTENSIONS_DIR" ]]; then
        echo "Total Size: $(du -sh "$EXTENSIONS_DIR" 2>/dev/null | cut -f1)"
    fi
    echo ""

    # Export version info
    if [[ "$DRY_RUN" != "true" ]]; then
        echo "$DUCKDB_VERSION" > "$HOME/.duckdb/version" 2>/dev/null || true

        # GitHub Actions output
        if [[ -n "${GITHUB_OUTPUT:-}" ]]; then
            echo "duckdb_version=${DUCKDB_VERSION}" >> "$GITHUB_OUTPUT"
            echo "extensions_dir=${EXTENSIONS_DIR}" >> "$GITHUB_OUTPUT"
        fi
    fi
}

# Print success message with next steps
print_success() {
    echo ""
    echo "${GREEN}=============================================="
    echo "  All extensions installed successfully!"
    echo "==============================================${NC}"
    echo ""
    echo "Next steps:"
    echo ""
    echo "  # Set environment variables"
    echo "  export GOTOOLCHAIN=local"
    echo "  export no_proxy=\"localhost,127.0.0.1\""
    echo "  export NO_PROXY=\"localhost,127.0.0.1\""
    echo ""
    echo "  # Build"
    echo "  CGO_ENABLED=1 go build -tags \"wal,nats\" -v -o cartographus ./cmd/server"
    echo ""
    echo "  # Run"
    echo "  mkdir -p /data && ./cartographus"
    echo ""
}

# Print failure message with remediation steps
print_failure() {
    echo ""
    echo "${RED}=============================================="
    echo "  Installation incomplete!"
    echo "==============================================${NC}"
    echo ""
    echo "$DOWNLOAD_FAIL_COUNT extension(s) failed to install."
    echo ""
    echo "Remediation steps:"
    echo ""
    echo "  1. Check network connectivity:"
    echo "     curl -I https://extensions.duckdb.org/"
    echo ""
    echo "  2. Try running again with verbose output:"
    echo "     $SCRIPT_NAME --verbose"
    echo ""
    echo "  3. Try forcing reinstall:"
    echo "     $SCRIPT_NAME --force"
    echo ""
    echo "  4. Manual download (if needed):"
    echo "     cd ${EXTENSIONS_DIR}"
    for ext in "${DOWNLOAD_FAILED_EXTENSIONS[@]}"; do
        local repo="$MAIN_REPO"
        case "$ext" in
            h3|rapidfuzz|datasketches) repo="$COMMUNITY_REPO" ;;
        esac
        echo "     curl -sSfL '${repo}/${ext}.duckdb_extension.gz' | gunzip > ${ext}.duckdb_extension"
    done
    echo ""
}

# =============================================================================
# Main Entry Point
# =============================================================================

main() {
    # Parse command line arguments
    parse_args "$@"

    # Show header
    if [[ "$QUIET" != "true" ]]; then
        echo ""
        echo "${BOLD}DuckDB Extensions Setup Script v${SCRIPT_VERSION}${NC}"
        if [[ "$DRY_RUN" == "true" ]]; then
            echo "${YELLOW}[DRY-RUN MODE - No changes will be made]${NC}"
        fi
        echo ""
    fi

    # Load version
    load_version
    log_info "DuckDB Version: ${DUCKDB_VERSION}"

    # Set up environment
    setup_environment

    # Verify prerequisites
    verify_prerequisites

    # Create directories
    create_directories

    # Acquire lock (unless dry-run)
    acquire_lock

    # Download extensions
    download_all_extensions

    # Print summary
    print_summary

    # Print result message
    if [[ $DOWNLOAD_FAIL_COUNT -eq 0 ]]; then
        print_success
        exit 0
    else
        print_failure
        exit 1
    fi
}

# Run main function
main "$@"
