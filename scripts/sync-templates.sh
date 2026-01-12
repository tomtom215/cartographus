#!/bin/bash

# Cartographus - Media Server Analytics and Geographic Visualization
# Copyright 2026 Tom F. (tomtom215)
# SPDX-License-Identifier: AGPL-3.0-or-later
# https://github.com/tomtom215/cartographus
# =============================================================================
# Template Sync Script
# =============================================================================
# Purpose: Maintain synchronization between development and production HTML templates
#
# The development template (web/public/index.html) is the source of truth for UI.
# The production template (internal/templates/index.html.tmpl) adds Go template
# syntax (nonce attributes) for CSP compliance.
#
# This script ensures both files stay in sync, preventing E2E test failures
# caused by missing HTML elements in the production template.
#
# Usage:
#   ./scripts/sync-templates.sh [--check|--sync|--diff|--help]
#
# Modes:
#   --check   Validate templates are in sync (exit 0 if synced, 1 if not)
#   --sync    Update production template from development source
#   --diff    Show differences between templates
#   --help    Show this help message
#
# Exit Codes:
#   0 - Success (templates in sync or sync completed)
#   1 - Templates out of sync (--check mode) or error occurred
#
# =============================================================================

set -euo pipefail

# Configuration
readonly SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
readonly DEV_TEMPLATE="$PROJECT_ROOT/web/public/index.html"
readonly PROD_TEMPLATE="$PROJECT_ROOT/internal/templates/index.html.tmpl"
readonly TEMP_DIR="${TMPDIR:-/tmp}/cartographus-template-sync"

# Colors for output (disabled if not a terminal)
if [[ -t 1 ]]; then
    readonly RED='\033[0;31m'
    readonly GREEN='\033[0;32m'
    readonly YELLOW='\033[1;33m'
    readonly BLUE='\033[0;34m'
    readonly NC='\033[0m' # No Color
else
    readonly RED=''
    readonly GREEN=''
    readonly YELLOW=''
    readonly BLUE=''
    readonly NC=''
fi

# Logging functions
log_info() { echo -e "${BLUE}[INFO]${NC} $*"; }
log_success() { echo -e "${GREEN}[OK]${NC} $*"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $*"; }
log_error() { echo -e "${RED}[ERROR]${NC} $*" >&2; }

# Show usage information
show_help() {
    cat << 'EOF'
Template Sync Script - Synchronize HTML templates for Cartographus

USAGE:
    ./scripts/sync-templates.sh [OPTIONS]

OPTIONS:
    --check     Validate templates are in sync (CI/pre-commit mode)
                Exit 0 if synced, exit 1 if drift detected

    --sync      Update production template from development source
                Adds Go template nonce attributes for CSP compliance

    --diff      Show differences between current templates
                Useful for reviewing what would change

    --help      Show this help message

EXAMPLES:
    # Check if templates are in sync (for CI)
    ./scripts/sync-templates.sh --check

    # Update production template after UI changes
    ./scripts/sync-templates.sh --sync

    # Review differences before syncing
    ./scripts/sync-templates.sh --diff

FILES:
    Source (development):  web/public/index.html
    Target (production):   internal/templates/index.html.tmpl

TRANSFORMATIONS APPLIED:
    The following Go template syntax is added to production template:
    - <script> -> <script nonce="{{.Nonce}}">
    - <script type="module" -> <script nonce="{{.Nonce}}" type="module"

WHY THIS MATTERS:
    The production template is used when running the server. If it's missing
    HTML elements that exist in development, E2E tests will fail because they
    can't find expected DOM elements.

EOF
}

# Validate required files exist
validate_files() {
    local errors=0

    if [[ ! -f "$DEV_TEMPLATE" ]]; then
        log_error "Development template not found: $DEV_TEMPLATE"
        ((errors++))
    fi

    if [[ ! -d "$(dirname "$PROD_TEMPLATE")" ]]; then
        log_error "Production template directory not found: $(dirname "$PROD_TEMPLATE")"
        ((errors++))
    fi

    return $errors
}

# Transform development HTML to production template
# This adds Go template nonce attributes for CSP compliance
transform_to_template() {
    local input="$1"
    local output="$2"

    # Create temp directory for intermediate files
    mkdir -p "$TEMP_DIR"
    local temp_file="$TEMP_DIR/transformed.html"

    # Copy source file
    cp "$input" "$temp_file"

    # Transformation 1: Add nonce to inline scripts (no src attribute)
    # Pattern: <script> at start of line (with whitespace) -> <script nonce="{{.Nonce}}">
    sed -i 's|^\([[:space:]]*\)<script>|\1<script nonce="{{.Nonce}}">|g' "$temp_file"

    # Transformation 2: Add nonce to module scripts
    # Pattern: <script type="module" -> <script nonce="{{.Nonce}}" type="module"
    sed -i 's|<script type="module"|<script nonce="{{.Nonce}}" type="module"|g' "$temp_file"

    # Move to final location
    mv "$temp_file" "$output"

    # Cleanup
    rmdir "$TEMP_DIR" 2>/dev/null || true
}

# Generate expected production template from development source
generate_expected() {
    local expected_file="$TEMP_DIR/expected.html.tmpl"
    mkdir -p "$TEMP_DIR"
    transform_to_template "$DEV_TEMPLATE" "$expected_file"
    echo "$expected_file"
}

# Check if templates are in sync
check_sync() {
    log_info "Checking template synchronization..."
    log_info "  Source: $DEV_TEMPLATE"
    log_info "  Target: $PROD_TEMPLATE"

    if ! validate_files; then
        return 1
    fi

    if [[ ! -f "$PROD_TEMPLATE" ]]; then
        log_error "Production template does not exist!"
        log_error "Run './scripts/sync-templates.sh --sync' to create it"
        return 1
    fi

    # Generate expected template
    local expected_file
    expected_file=$(generate_expected)

    # Compare
    if diff -q "$expected_file" "$PROD_TEMPLATE" > /dev/null 2>&1; then
        log_success "Templates are in sync!"

        # Show metrics
        local dev_lines prod_lines nonce_count testid_count
        dev_lines=$(wc -l < "$DEV_TEMPLATE")
        prod_lines=$(wc -l < "$PROD_TEMPLATE")
        nonce_count=$(grep -c '{{.Nonce}}' "$PROD_TEMPLATE" || echo "0")
        testid_count=$(grep -c 'data-testid' "$PROD_TEMPLATE" || echo "0")

        log_info "  Development lines: $dev_lines"
        log_info "  Production lines:  $prod_lines"
        log_info "  Nonce templates:   $nonce_count"
        log_info "  Test IDs:          $testid_count"

        # Cleanup
        rm -f "$expected_file"
        rmdir "$TEMP_DIR" 2>/dev/null || true

        return 0
    else
        log_error "Templates are OUT OF SYNC!"
        log_error ""
        log_error "The production template differs from what would be generated"
        log_error "from the development template."
        log_error ""
        log_error "This can cause E2E test failures due to missing HTML elements."
        log_error ""
        log_error "To fix, run:"
        log_error "  ./scripts/sync-templates.sh --sync"
        log_error ""
        log_error "To see differences, run:"
        log_error "  ./scripts/sync-templates.sh --diff"

        # Cleanup
        rm -f "$expected_file"
        rmdir "$TEMP_DIR" 2>/dev/null || true

        return 1
    fi
}

# Show diff between templates
show_diff() {
    log_info "Comparing templates..."

    if ! validate_files; then
        return 1
    fi

    if [[ ! -f "$PROD_TEMPLATE" ]]; then
        log_warn "Production template does not exist. Showing what would be created:"
        echo ""
        local expected_file
        expected_file=$(generate_expected)
        head -50 "$expected_file"
        echo "..."
        echo "(Showing first 50 lines)"
        rm -f "$expected_file"
        rmdir "$TEMP_DIR" 2>/dev/null || true
        return 0
    fi

    # Generate expected template
    local expected_file
    expected_file=$(generate_expected)

    # Show diff with context
    log_info "Diff: expected (from dev) vs current (production)"
    log_info "  < lines would be in generated template"
    log_info "  > lines are in current production template"
    echo ""

    if diff -u "$expected_file" "$PROD_TEMPLATE" --label "expected (from development)" --label "current (production)"; then
        log_success "No differences - templates are in sync!"
    else
        echo ""
        log_warn "Differences found. Run './scripts/sync-templates.sh --sync' to update."
    fi

    # Cleanup
    rm -f "$expected_file"
    rmdir "$TEMP_DIR" 2>/dev/null || true

    return 0
}

# Sync templates (update production from development)
sync_templates() {
    log_info "Syncing templates..."
    log_info "  Source: $DEV_TEMPLATE"
    log_info "  Target: $PROD_TEMPLATE"

    if ! validate_files; then
        return 1
    fi

    # Check if we're making changes
    local had_changes=false
    if [[ -f "$PROD_TEMPLATE" ]]; then
        local expected_file
        expected_file=$(generate_expected)
        if ! diff -q "$expected_file" "$PROD_TEMPLATE" > /dev/null 2>&1; then
            had_changes=true
        fi
        rm -f "$expected_file"
        rmdir "$TEMP_DIR" 2>/dev/null || true
    else
        had_changes=true
    fi

    # Create backup if production template exists
    if [[ -f "$PROD_TEMPLATE" ]]; then
        local backup_file="$PROD_TEMPLATE.backup.$(date +%Y%m%d_%H%M%S)"
        cp "$PROD_TEMPLATE" "$backup_file"
        log_info "  Backup: $backup_file"
    fi

    # Generate new production template
    transform_to_template "$DEV_TEMPLATE" "$PROD_TEMPLATE"

    # Validate the result
    local nonce_count testid_count prod_lines
    prod_lines=$(wc -l < "$PROD_TEMPLATE")
    nonce_count=$(grep -c '{{.Nonce}}' "$PROD_TEMPLATE" || echo "0")
    testid_count=$(grep -c 'data-testid' "$PROD_TEMPLATE" || echo "0")

    # Sanity checks
    if [[ "$nonce_count" -lt 1 ]]; then
        log_error "Transformation failed: No nonce templates found!"
        log_error "Expected at least 1 {{.Nonce}} in output"
        return 1
    fi

    if [[ "$testid_count" -lt 10 ]]; then
        log_warn "Low test ID count ($testid_count). Verify template is complete."
    fi

    log_success "Templates synced successfully!"
    log_info "  Production lines: $prod_lines"
    log_info "  Nonce templates:  $nonce_count"
    log_info "  Test IDs:         $testid_count"

    if $had_changes; then
        log_info ""
        log_info "Changes were made. Don't forget to:"
        log_info "  1. Review the changes: git diff internal/templates/index.html.tmpl"
        log_info "  2. Commit: git add internal/templates/index.html.tmpl"
    else
        log_info "No changes needed - templates were already in sync."
    fi

    return 0
}

# Main entry point
main() {
    local mode="${1:-}"

    case "$mode" in
        --check|-c)
            check_sync
            ;;
        --sync|-s)
            sync_templates
            ;;
        --diff|-d)
            show_diff
            ;;
        --help|-h)
            show_help
            ;;
        "")
            log_error "No mode specified."
            echo ""
            show_help
            exit 1
            ;;
        *)
            log_error "Unknown option: $mode"
            echo ""
            show_help
            exit 1
            ;;
    esac
}

main "$@"
