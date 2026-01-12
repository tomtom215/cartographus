#!/bin/bash

# Cartographus - Media Server Analytics and Geographic Visualization
# Copyright 2026 Tom F. (tomtom215)
# SPDX-License-Identifier: AGPL-3.0-or-later
# https://github.com/tomtom215/cartographus
# =============================================================================
# Unit Tests for Template Sync Script
# =============================================================================
# Tests the sync-templates.sh script to ensure it correctly validates and
# syncs HTML templates.
#
# Usage:
#   ./scripts/test-sync-templates.sh
#
# Exit Codes:
#   0 - All tests passed
#   1 - One or more tests failed
#
# =============================================================================

# Don't use set -e as we want to continue on test failures
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
SYNC_SCRIPT="$SCRIPT_DIR/sync-templates.sh"

# Test directory
TEST_DIR=$(mktemp -d)
trap 'rm -rf "$TEST_DIR"' EXIT

# Colors
if [[ -t 1 ]]; then
    RED='\033[0;31m'
    GREEN='\033[0;32m'
    NC='\033[0m'
else
    RED=''
    GREEN=''
    NC=''
fi

# Test counters
PASSED=0
FAILED=0

# Test helper functions
pass() {
    echo -e "  ${GREEN}PASS${NC}: $1"
    ((PASSED++))
}

fail() {
    echo -e "  ${RED}FAIL${NC}: $1"
    ((FAILED++))
}

# =============================================================================
# Test 1: Script exists and is executable
# =============================================================================
test_script_exists() {
    echo "Test 1: Script exists and is executable"

    if [[ -x "$SYNC_SCRIPT" ]]; then
        pass "Script is executable"
    else
        fail "Script is not executable"
    fi
}

# =============================================================================
# Test 2: Help flag works
# =============================================================================
test_help_flag() {
    echo "Test 2: Help flag works"

    local output
    output=$("$SYNC_SCRIPT" --help 2>&1) || true

    if echo "$output" | grep -q "Template Sync Script"; then
        pass "--help shows usage information"
    else
        fail "--help did not show expected output"
    fi
}

# =============================================================================
# Test 3: Check mode returns success when in sync
# =============================================================================
test_check_when_synced() {
    echo "Test 3: Check mode returns success when in sync"

    # First sync to ensure they're aligned
    "$SYNC_SCRIPT" --sync > /dev/null 2>&1 || true

    if "$SYNC_SCRIPT" --check > /dev/null 2>&1; then
        pass "--check succeeds when templates are in sync"
    else
        fail "--check failed when templates should be in sync"
    fi
}

# =============================================================================
# Test 4: Nonce transformation is applied correctly
# =============================================================================
test_nonce_transformation() {
    echo "Test 4: Nonce transformation is applied correctly"

    local nonce_count
    nonce_count=$(grep -c '{{.Nonce}}' "$PROJECT_ROOT/internal/templates/index.html.tmpl" || echo "0")

    if [[ "$nonce_count" -ge 1 ]]; then
        pass "Nonce templates are present (count: $nonce_count)"
    else
        fail "No nonce templates found in production template"
    fi
}

# =============================================================================
# Test 5: Inline script nonce is correct
# =============================================================================
test_inline_script_nonce() {
    echo "Test 5: Inline script nonce is correct"

    if grep -q '<script nonce="{{.Nonce}}">' "$PROJECT_ROOT/internal/templates/index.html.tmpl"; then
        pass "Inline script has nonce attribute"
    else
        fail "Inline script missing nonce attribute"
    fi
}

# =============================================================================
# Test 6: Module script nonce is correct
# =============================================================================
test_module_script_nonce() {
    echo "Test 6: Module script nonce is correct"

    if grep -q '<script nonce="{{.Nonce}}" type="module"' "$PROJECT_ROOT/internal/templates/index.html.tmpl"; then
        pass "Module script has nonce attribute in correct position"
    else
        fail "Module script missing or incorrect nonce attribute"
    fi
}

# =============================================================================
# Test 7: Test ID counts match between templates
# =============================================================================
test_testid_counts_match() {
    echo "Test 7: Test ID counts match between templates"

    local dev_count prod_count
    dev_count=$(grep -c 'data-testid' "$PROJECT_ROOT/web/public/index.html" || echo "0")
    prod_count=$(grep -c 'data-testid' "$PROJECT_ROOT/internal/templates/index.html.tmpl" || echo "0")

    if [[ "$dev_count" -eq "$prod_count" ]]; then
        pass "Test ID counts match (count: $dev_count)"
    else
        fail "Test ID counts differ (dev: $dev_count, prod: $prod_count)"
    fi
}

# =============================================================================
# Test 8: Line counts are equal (or very close)
# =============================================================================
test_line_counts() {
    echo "Test 8: Line counts are equal"

    local dev_lines prod_lines
    dev_lines=$(wc -l < "$PROJECT_ROOT/web/public/index.html")
    prod_lines=$(wc -l < "$PROJECT_ROOT/internal/templates/index.html.tmpl")

    if [[ "$dev_lines" -eq "$prod_lines" ]]; then
        pass "Line counts match (lines: $dev_lines)"
    else
        fail "Line counts differ (dev: $dev_lines, prod: $prod_lines)"
    fi
}

# =============================================================================
# Test 9: Critical UI elements exist in production template
# =============================================================================
test_critical_elements() {
    echo "Test 9: Critical UI elements exist in production template"

    local missing=0
    local elements=(
        "content-sub-tabs"
        "content-collections-panel"
        "content-playlists-panel"
        "library-sub-tabs"
        "library-details-panel"
        "library-charts-panel"
        "user-ip-history-modal"
        "collections-container"
        "playlists-container"
    )

    for element in "${elements[@]}"; do
        if ! grep -q "$element" "$PROJECT_ROOT/internal/templates/index.html.tmpl"; then
            echo "    Missing: $element"
            ((missing++))
        fi
    done

    if [[ "$missing" -eq 0 ]]; then
        pass "All critical UI elements present"
    else
        fail "Missing $missing critical UI elements"
    fi
}

# =============================================================================
# Test 10: Diff mode runs without error
# =============================================================================
test_diff_mode() {
    echo "Test 10: Diff mode runs without error"

    local result
    "$SYNC_SCRIPT" --diff > /dev/null 2>&1 && result=0 || result=$?

    if [[ "$result" -eq 0 ]]; then
        pass "--diff runs without error"
    else
        fail "--diff failed with exit code $result"
    fi
}

# =============================================================================
# Test 11: Invalid option shows error
# =============================================================================
test_invalid_option() {
    echo "Test 11: Invalid option shows error"

    local output
    output=$("$SYNC_SCRIPT" --invalid 2>&1) || true

    if echo "$output" | grep -q "Unknown option"; then
        pass "Invalid option shows error message"
    else
        fail "Invalid option should show error"
    fi
}

# =============================================================================
# Test 12: Sync is idempotent
# =============================================================================
test_sync_idempotent() {
    echo "Test 12: Sync is idempotent"

    # Get checksum before sync
    local before_md5 after_md5
    before_md5=$(md5sum "$PROJECT_ROOT/internal/templates/index.html.tmpl" | cut -d' ' -f1)

    # Run sync
    "$SYNC_SCRIPT" --sync > /dev/null 2>&1 || true

    # Get checksum after sync
    after_md5=$(md5sum "$PROJECT_ROOT/internal/templates/index.html.tmpl" | cut -d' ' -f1)

    if [[ "$before_md5" == "$after_md5" ]]; then
        pass "Sync is idempotent (no changes when already synced)"
    else
        fail "Sync changed file when already synced"
    fi
}

# =============================================================================
# Run all tests
# =============================================================================
echo "========================================"
echo "Template Sync Script Tests"
echo "========================================"
echo ""

test_script_exists
test_help_flag
test_check_when_synced
test_nonce_transformation
test_inline_script_nonce
test_module_script_nonce
test_testid_counts_match
test_line_counts
test_critical_elements
test_diff_mode
test_invalid_option
test_sync_idempotent

# Summary
echo ""
echo "========================================"
echo "Test Summary"
echo "========================================"
echo -e "  ${GREEN}Passed${NC}: $PASSED"
echo -e "  ${RED}Failed${NC}: $FAILED"
echo ""

if [[ "$FAILED" -eq 0 ]]; then
    echo -e "${GREEN}All tests passed!${NC}"
    exit 0
else
    echo -e "${RED}Some tests failed.${NC}"
    exit 1
fi
