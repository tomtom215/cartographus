#!/bin/bash

# Cartographus - Media Server Analytics and Geographic Visualization
# Copyright 2026 Tom F. (tomtom215)
# SPDX-License-Identifier: AGPL-3.0-or-later
# https://github.com/tomtom215/cartographus
# =============================================================================
# Claude Code Web Session Setup Script
# =============================================================================
# Run this script at the START of every Claude Code Web session to configure
# the environment for building, testing, and running Cartographus.
#
# Usage:
#   source scripts/session-setup.sh          # Full setup with build
#   source scripts/session-setup.sh --quick  # Env vars only, skip build
#   source scripts/session-setup.sh --verify # Verify setup without changes
#   source scripts/session-setup.sh --help   # Show help
#
# IMPORTANT: Use 'source' (or '.') to ensure environment variables persist
# in your shell session. Running with './scripts/session-setup.sh' will not
# export variables to your current shell.
#
# What this script does:
#   1. Sets required environment variables (GOTOOLCHAIN, no_proxy, etc.)
#   2. Verifies Go installation and configuration
#   3. Installs DuckDB extensions (if not present)
#   4. Optionally builds the Go binary
#   5. Optionally sets up frontend (npm ci)
#   6. Verifies the setup is complete
#
# =============================================================================

# Colors (disabled if not a terminal or if sourced)
if [[ -t 1 ]] && [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    RED='\033[0;31m'
    GREEN='\033[0;32m'
    YELLOW='\033[0;33m'
    BLUE='\033[0;34m'
    BOLD='\033[1m'
    NC='\033[0m'
else
    RED=''
    GREEN=''
    YELLOW=''
    BLUE=''
    BOLD=''
    NC=''
fi

# =============================================================================
# Configuration
# =============================================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Default options
QUICK_MODE=false
VERIFY_ONLY=false
SKIP_BUILD=false
SKIP_FRONTEND=true  # Skip frontend by default (faster)
SKIP_DEPS=true      # Skip deps download by default (use --deps or --all)
VERBOSE=false

# =============================================================================
# Helper Functions
# =============================================================================

log_header() {
    echo -e "\n${BOLD}=== $* ===${NC}"
}

log_info() {
    echo -e "${BLUE}[INFO]${NC} $*"
}

log_success() {
    echo -e "${GREEN}[OK]${NC} $*"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $*"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $*"
}

show_help() {
    cat << 'EOF'
Claude Code Web Session Setup Script

USAGE:
    source scripts/session-setup.sh [OPTIONS]

    IMPORTANT: Use 'source' (or '.') to ensure environment variables persist!

OPTIONS:
    --quick, -q      Quick mode: set env vars only, skip build/extensions
    --all, -a        Full setup: env + deps + extensions + build + frontend
    --verify, -v     Verify current setup without making changes
    --build, -b      Force rebuild even if binary exists
    --frontend, -f   Also set up frontend (npm ci, npm run build)
    --deps, -d       Download Go and npm dependencies
    --verbose        Show detailed output
    --help, -h       Show this help message

EXAMPLES:
    # Standard setup (env + extensions + build)
    source scripts/session-setup.sh

    # Complete setup with all dependencies (first time)
    source scripts/session-setup.sh --all

    # Quick setup (just env vars, for subsequent runs)
    source scripts/session-setup.sh --quick

    # Verify everything is set up correctly
    source scripts/session-setup.sh --verify

    # Full setup including frontend only
    source scripts/session-setup.sh --frontend

ENVIRONMENT VARIABLES SET:
    GOTOOLCHAIN=local           Prevents Go toolchain downloads
    no_proxy=localhost,127.0.0.1  Allows proxy for googleapis.com
    NO_PROXY=localhost,127.0.0.1  Same as above (uppercase variant)
    CGO_ENABLED=1               Required for DuckDB

WHAT GETS VERIFIED:
    - Go installation (1.24+)
    - DuckDB extensions (~/.duckdb/extensions/v1.4.3/linux_amd64/)
    - Binary build (./cartographus ~102MB)
    - Build tags support (wal,nats)

EOF
}

# =============================================================================
# Argument Parsing
# =============================================================================

parse_args() {
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --quick|-q)
                QUICK_MODE=true
                SKIP_BUILD=true
                shift
                ;;
            --all|-a)
                # Full setup: everything including deps and frontend
                SKIP_DEPS=false
                SKIP_FRONTEND=false
                SKIP_BUILD=false
                shift
                ;;
            --verify|-v)
                VERIFY_ONLY=true
                shift
                ;;
            --build|-b)
                SKIP_BUILD=false
                shift
                ;;
            --frontend|-f)
                SKIP_FRONTEND=false
                shift
                ;;
            --deps|-d)
                SKIP_DEPS=false
                shift
                ;;
            --verbose)
                VERBOSE=true
                shift
                ;;
            --help|-h)
                show_help
                return 1  # Return instead of exit when sourced
                ;;
            *)
                log_error "Unknown option: $1"
                echo "Use '--help' for usage information."
                return 1
                ;;
        esac
    done
}

# =============================================================================
# Environment Setup
# =============================================================================

setup_environment() {
    log_header "Setting Environment Variables"

    # CRITICAL: These must be set before any Go commands
    export GOTOOLCHAIN=local
    export no_proxy="localhost,127.0.0.1"
    export NO_PROXY="localhost,127.0.0.1"
    export CGO_ENABLED=1

    log_success "GOTOOLCHAIN=local"
    log_success "no_proxy=localhost,127.0.0.1"
    log_success "NO_PROXY=localhost,127.0.0.1"
    log_success "CGO_ENABLED=1"

    # Unset GOPROXY=off if it's set (can cause issues)
    if [[ "${GOPROXY:-}" == "off" ]]; then
        unset GOPROXY
        log_warn "Unset GOPROXY=off (was blocking module downloads)"
    fi
}

# =============================================================================
# Verification Functions
# =============================================================================

verify_go() {
    log_header "Verifying Go Installation"

    if ! command -v go &>/dev/null; then
        log_error "Go is not installed or not in PATH"
        return 1
    fi

    local go_version
    go_version=$(go version 2>/dev/null | grep -oP 'go\d+\.\d+' | head -1)
    log_success "Go version: $(go version)"

    # Verify GOTOOLCHAIN
    local toolchain
    toolchain=$(go env GOTOOLCHAIN 2>/dev/null)
    if [[ "$toolchain" == "local" ]]; then
        log_success "GOTOOLCHAIN: $toolchain"
    else
        log_warn "GOTOOLCHAIN is '$toolchain', should be 'local'"
        log_warn "Run: export GOTOOLCHAIN=local"
    fi

    return 0
}

verify_duckdb_extensions() {
    log_header "Verifying DuckDB Extensions"

    # Source version
    if [[ -f "$SCRIPT_DIR/duckdb-version.sh" ]]; then
        source "$SCRIPT_DIR/duckdb-version.sh"
    else
        DUCKDB_VERSION="${DUCKDB_VERSION:-v1.4.3}"
    fi

    local ext_dir="$HOME/.duckdb/extensions/${DUCKDB_VERSION}/linux_amd64"
    local required_extensions=(httpfs spatial inet icu json sqlite_scanner h3 rapidfuzz)
    local optional_extensions=(datasketches)
    local missing=()
    local present=()

    # Check required extensions
    for ext in "${required_extensions[@]}"; do
        if [[ -f "${ext_dir}/${ext}.duckdb_extension" ]]; then
            present+=("$ext")
        else
            missing+=("$ext")
        fi
    done

    # Report status
    if [[ ${#present[@]} -gt 0 ]]; then
        log_success "Present: ${present[*]}"
    fi

    if [[ ${#missing[@]} -gt 0 ]]; then
        log_warn "Missing: ${missing[*]}"
        log_info "Run: ./scripts/setup-duckdb-extensions.sh"
        return 1
    fi

    # Check optional
    for ext in "${optional_extensions[@]}"; do
        if [[ -f "${ext_dir}/${ext}.duckdb_extension" ]]; then
            log_success "Optional present: $ext"
        else
            log_info "Optional missing: $ext (non-critical)"
        fi
    done

    log_success "Extensions directory: $ext_dir"
    return 0
}

verify_binary() {
    log_header "Verifying Binary"

    local binary="$PROJECT_ROOT/cartographus"

    if [[ ! -f "$binary" ]]; then
        log_warn "Binary not found: $binary"
        log_info "Run: CGO_ENABLED=1 go build -tags \"wal,nats\" -o cartographus ./cmd/server"
        return 1
    fi

    # Check size (should be ~100MB+)
    local size
    size=$(stat -c%s "$binary" 2>/dev/null || stat -f%z "$binary" 2>/dev/null || echo 0)
    local size_mb=$((size / 1024 / 1024))

    if [[ $size_mb -lt 50 ]]; then
        log_warn "Binary is smaller than expected (${size_mb}MB < 50MB)"
        log_warn "May be missing build tags. Rebuild with: -tags \"wal,nats\""
        return 1
    fi

    log_success "Binary: $binary (${size_mb}MB)"

    # Verify it's executable
    if [[ -x "$binary" ]]; then
        log_success "Binary is executable"
    else
        log_warn "Binary is not executable. Run: chmod +x $binary"
    fi

    return 0
}

verify_frontend() {
    log_header "Verifying Frontend"

    local web_dir="$PROJECT_ROOT/web"

    if [[ ! -d "$web_dir/node_modules" ]]; then
        log_warn "node_modules not found"
        log_info "Run: cd web && npm ci"
        return 1
    fi

    if [[ ! -f "$web_dir/dist/index.js" ]]; then
        log_warn "Frontend not built (dist/index.js missing)"
        log_info "Run: cd web && npm run build"
        return 1
    fi

    log_success "Frontend: node_modules present, dist built"
    return 0
}

# =============================================================================
# Setup Functions
# =============================================================================

setup_dependencies() {
    log_header "Downloading Dependencies"

    cd "$PROJECT_ROOT" || return 1

    # Go modules
    log_info "Downloading Go modules..."
    if go mod download; then
        log_success "Go modules downloaded"
    else
        log_warn "Some Go modules may have failed to download"
    fi

    # Tidy go.mod
    log_info "Tidying go.mod..."
    go mod tidy || log_warn "go mod tidy had issues"

    # npm dependencies (if node_modules doesn't exist or is incomplete)
    if [[ ! -d "$PROJECT_ROOT/web/node_modules" ]] || [[ ! -f "$PROJECT_ROOT/web/node_modules/.package-lock.json" ]]; then
        log_info "Installing npm dependencies..."
        cd "$PROJECT_ROOT/web" || return 1
        if npm ci; then
            log_success "npm dependencies installed"
        else
            log_warn "npm ci failed, trying npm install..."
            npm install || log_error "npm install also failed"
        fi
        cd "$PROJECT_ROOT" || return 1
    else
        log_success "npm dependencies already installed"
    fi

    return 0
}

setup_duckdb_extensions() {
    log_header "Setting Up DuckDB Extensions"

    if verify_duckdb_extensions 2>/dev/null; then
        log_success "All required extensions already installed"
        return 0
    fi

    log_info "Installing DuckDB extensions..."
    if [[ -f "$SCRIPT_DIR/setup-duckdb-extensions.sh" ]]; then
        "$SCRIPT_DIR/setup-duckdb-extensions.sh"
        return $?
    else
        log_error "setup-duckdb-extensions.sh not found"
        return 1
    fi
}

build_binary() {
    log_header "Building Binary"

    cd "$PROJECT_ROOT" || return 1

    # Check if binary exists and is recent
    local binary="$PROJECT_ROOT/cartographus"
    if [[ -f "$binary" ]] && [[ "$SKIP_BUILD" == "true" ]]; then
        log_success "Binary exists, skipping build (use --build to force)"
        return 0
    fi

    log_info "Building with -tags \"wal,nats\"..."
    if CGO_ENABLED=1 go build -tags "wal,nats" -o cartographus ./cmd/server; then
        local size
        size=$(du -h "$binary" | cut -f1)
        log_success "Built: $binary ($size)"
        return 0
    else
        log_error "Build failed"
        return 1
    fi
}

setup_frontend() {
    log_header "Setting Up Frontend"

    cd "$PROJECT_ROOT/web" || return 1

    if [[ ! -d "node_modules" ]]; then
        log_info "Installing npm dependencies..."
        npm ci || return 1
    else
        log_success "node_modules already present"
    fi

    if [[ ! -f "dist/index.js" ]]; then
        log_info "Building frontend..."
        npm run build || return 1
    else
        log_success "Frontend already built"
    fi

    cd "$PROJECT_ROOT" || return 1
    return 0
}

# =============================================================================
# Summary
# =============================================================================

print_summary() {
    log_header "Setup Summary"

    local status=0

    echo ""
    echo "Environment Variables:"
    echo "  GOTOOLCHAIN=$(go env GOTOOLCHAIN 2>/dev/null || echo 'not set')"
    echo "  CGO_ENABLED=${CGO_ENABLED:-not set}"
    echo "  no_proxy=${no_proxy:-not set}"
    echo ""

    # Quick status checks
    echo "Component Status:"

    if command -v go &>/dev/null; then
        echo -e "  ${GREEN}✓${NC} Go: $(go version | grep -oP 'go\d+\.\d+\.\d+')"
    else
        echo -e "  ${RED}✗${NC} Go: not found"
        status=1
    fi

    # DuckDB extensions
    source "$SCRIPT_DIR/duckdb-version.sh" 2>/dev/null || DUCKDB_VERSION="v1.4.3"
    local ext_dir="$HOME/.duckdb/extensions/${DUCKDB_VERSION}/linux_amd64"
    local ext_count=$(ls -1 "$ext_dir"/*.duckdb_extension 2>/dev/null | wc -l)
    if [[ $ext_count -ge 8 ]]; then
        echo -e "  ${GREEN}✓${NC} DuckDB extensions: $ext_count installed"
    else
        echo -e "  ${YELLOW}!${NC} DuckDB extensions: $ext_count installed (need 8+)"
        status=1
    fi

    # Binary
    if [[ -f "$PROJECT_ROOT/cartographus" ]]; then
        local size=$(du -h "$PROJECT_ROOT/cartographus" | cut -f1)
        echo -e "  ${GREEN}✓${NC} Binary: cartographus ($size)"
    else
        echo -e "  ${YELLOW}!${NC} Binary: not built"
        status=1
    fi

    # Frontend (optional)
    if [[ -d "$PROJECT_ROOT/web/node_modules" ]]; then
        echo -e "  ${GREEN}✓${NC} Frontend: node_modules present"
    else
        echo -e "  ${BLUE}○${NC} Frontend: not set up (optional)"
    fi

    echo ""

    # Next steps
    if [[ $status -eq 0 ]]; then
        echo -e "${GREEN}Ready to develop!${NC}"
        echo ""
        echo "Quick commands:"
        echo "  go test -tags \"wal,nats\" -v -race ./...  # Run tests"
        echo "  go vet -tags \"wal,nats\" ./...            # Lint"
        echo "  ./cartographus                            # Run server"
        echo ""
    else
        echo -e "${YELLOW}Setup incomplete. Run without --quick to fix.${NC}"
        echo ""
    fi

    return $status
}

# =============================================================================
# Main
# =============================================================================

main() {
    # Parse arguments
    if ! parse_args "$@"; then
        return 1
    fi

    echo ""
    echo -e "${BOLD}Cartographus - Claude Code Web Session Setup${NC}"
    echo ""

    # Always set environment variables first
    setup_environment

    # Verify-only mode
    if [[ "$VERIFY_ONLY" == "true" ]]; then
        log_header "Verification Mode"
        verify_go
        verify_duckdb_extensions
        verify_binary
        verify_frontend
        print_summary
        return $?
    fi

    # Quick mode - just env vars
    if [[ "$QUICK_MODE" == "true" ]]; then
        log_info "Quick mode - environment variables set"
        verify_go
        print_summary
        return 0
    fi

    # Full setup
    verify_go || return 1

    # Download dependencies (if requested with --deps or --all)
    if [[ "$SKIP_DEPS" != "true" ]]; then
        setup_dependencies || {
            log_warn "Some dependencies may not have downloaded correctly"
        }
    fi

    # DuckDB extensions
    setup_duckdb_extensions || {
        log_error "Failed to set up DuckDB extensions"
        return 1
    }

    # Build binary (unless skipped)
    if [[ "$SKIP_BUILD" != "true" ]]; then
        build_binary || {
            log_error "Failed to build binary"
            return 1
        }
    fi

    # Frontend (if requested)
    if [[ "$SKIP_FRONTEND" != "true" ]]; then
        setup_frontend || {
            log_warn "Failed to set up frontend (non-critical)"
        }
    fi

    # Print summary
    print_summary
}

# Run main function
main "$@"
