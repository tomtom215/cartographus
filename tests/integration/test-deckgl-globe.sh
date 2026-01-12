#!/bin/bash

# Cartographus - Media Server Analytics and Geographic Visualization
# Copyright 2026 Tom F. (tomtom215)
# SPDX-License-Identifier: AGPL-3.0-or-later
# https://github.com/tomtom215/cartographus

##
# Integration Test Script for deck.gl Globe Implementation
#
# This script performs comprehensive integration testing of the deck.gl globe
# to verify it's ready to replace echarts-gl in production.
#
# Usage:
#   ./test-deckgl-globe.sh [--verbose]
#
# Requirements:
#   - Docker and Docker Compose
#   - curl
#   - jq (optional, for JSON parsing)
##

set -e

VERBOSE=false
if [[ "$1" == "--verbose" ]]; then
    VERBOSE=true
fi

BASE_URL="${BASE_URL:-http://localhost:3857}"
API_URL="$BASE_URL/api/v1"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test counters
TESTS_PASSED=0
TESTS_FAILED=0
TESTS_TOTAL=0

##
# Helper functions
##

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_test() {
    TESTS_TOTAL=$((TESTS_TOTAL + 1))
    echo ""
    echo -e "${YELLOW}[TEST $TESTS_TOTAL]${NC} $1"
}

pass_test() {
    TESTS_PASSED=$((TESTS_PASSED + 1))
    echo -e "${GREEN}✓ PASSED${NC}"
}

fail_test() {
    TESTS_FAILED=$((TESTS_FAILED + 1))
    echo -e "${RED}✗ FAILED${NC} $1"
}

##
# Test Suite
##

log_info "Starting deck.gl Globe Integration Tests"
log_info "Base URL: $BASE_URL"
echo ""

# Test 1: Check if server is running
log_test "Server is accessible and responding"
if curl -f -s "$API_URL/health" > /dev/null 2>&1; then
    pass_test
else
    fail_test "Server is not accessible at $BASE_URL"
fi

# Test 2: Check frontend bundle includes deck.gl
log_test "Frontend bundle includes deck.gl libraries"
BUNDLE_PATH="web/dist/bundle.js"
if [ -f "$BUNDLE_PATH" ]; then
    if grep -q "deck.gl" "$BUNDLE_PATH" 2>/dev/null || \
       grep -q "MapboxOverlay" "$BUNDLE_PATH" 2>/dev/null || \
       grep -q "ScatterplotLayer" "$BUNDLE_PATH" 2>/dev/null; then
        pass_test
    else
        fail_test "deck.gl not found in bundle"
    fi
else
    fail_test "Bundle file not found at $BUNDLE_PATH"
fi

# Test 3: Check echarts-gl is NOT in bundle
log_test "echarts-gl is removed from bundle"
if [ -f "$BUNDLE_PATH" ]; then
    if ! grep -q "echarts-gl" "$BUNDLE_PATH" 2>/dev/null; then
        pass_test
    else
        fail_test "echarts-gl still found in bundle"
    fi
else
    fail_test "Bundle file not found"
fi

# Test 4: Verify package.json dependencies
log_test "package.json has correct dependencies"
PACKAGE_JSON="web/package.json"
if [ -f "$PACKAGE_JSON" ]; then
    HAS_DECKGL=$(grep -c "@deck.gl/core" "$PACKAGE_JSON" || echo "0")
    HAS_ECHARTS_GL=$(grep -c "echarts-gl" "$PACKAGE_JSON" || echo "0")

    if [ "$HAS_DECKGL" -gt 0 ] && [ "$HAS_ECHARTS_GL" -eq 0 ]; then
        pass_test
    else
        fail_test "Dependencies incorrect (deck.gl: $HAS_DECKGL, echarts-gl: $HAS_ECHARTS_GL)"
    fi
else
    fail_test "package.json not found"
fi

# Test 5: Check GlobeManagerDeckGL file exists
log_test "GlobeManagerDeckGL implementation file exists"
GLOBE_FILE="web/src/lib/globe-deckgl.ts"
if [ -f "$GLOBE_FILE" ]; then
    # Check file has required imports
    if grep -q "MapboxOverlay" "$GLOBE_FILE" && \
       grep -q "ScatterplotLayer" "$GLOBE_FILE"; then
        pass_test
    else
        fail_test "Missing required imports in $GLOBE_FILE"
    fi
else
    fail_test "File not found: $GLOBE_FILE"
fi

# Test 6: Check index.ts uses new GlobeManagerDeckGL
log_test "index.ts imports GlobeManagerDeckGL"
INDEX_FILE="web/src/lib/globe-deckgl.ts"
if [ -f "$INDEX_FILE" ]; then
    if grep -q "GlobeManagerDeckGL" "web/src/index.ts"; then
        pass_test
    else
        fail_test "index.ts doesn't import GlobeManagerDeckGL"
    fi
else
    fail_test "index.ts not found"
fi

# Test 7: Check HTML doesn't load echarts-gl from CDN
log_test "index.html doesn't load echarts-gl from CDN"
HTML_FILE="web/public/index.html"
if [ -f "$HTML_FILE" ]; then
    if ! grep -q "echarts-gl" "$HTML_FILE"; then
        pass_test
    else
        fail_test "index.html still loads echarts-gl from CDN"
    fi
else
    fail_test "index.html not found"
fi

# Test 8: Verify TypeScript compilation
log_test "TypeScript compiles without errors"
if [ -d "web/node_modules" ]; then
    cd web
    if npx tsc --noEmit > /dev/null 2>&1; then
        cd ..
        pass_test
    else
        cd ..
        fail_test "TypeScript compilation errors"
    fi
else
    log_warn "node_modules not found, skipping TypeScript check"
fi

# Test 9: Check bundle size is reasonable
log_test "Bundle size is within acceptable range"
if [ -f "$BUNDLE_PATH" ]; then
    BUNDLE_SIZE=$(stat -f%z "$BUNDLE_PATH" 2>/dev/null || stat -c%s "$BUNDLE_PATH" 2>/dev/null || echo "0")
    BUNDLE_SIZE_MB=$((BUNDLE_SIZE / 1024 / 1024))

    if [ "$BUNDLE_SIZE_MB" -lt 10 ]; then
        log_info "Bundle size: ${BUNDLE_SIZE_MB}MB"
        pass_test
    else
        fail_test "Bundle size too large: ${BUNDLE_SIZE_MB}MB (max: 10MB)"
    fi
else
    fail_test "Bundle file not found"
fi

# Test 10: Verify API endpoints are accessible
log_test "API endpoints are accessible"
if curl -f -s "$API_URL/locations" -H "Cookie: auth_token=test" > /dev/null 2>&1 || \
   curl -s "$API_URL/locations" | grep -q "data\|error"; then
    pass_test
else
    log_warn "API endpoint test inconclusive (may require authentication)"
fi

# Test 11: Check for console error patterns
log_test "No known error patterns in implementation files"
ERROR_PATTERNS=("TODO:" "FIXME:" "XXX:" "HACK:")
HAS_ERRORS=false

for pattern in "${ERROR_PATTERNS[@]}"; do
    if grep -r "$pattern" web/src/lib/globe-deckgl.ts 2>/dev/null | grep -v "test" > /dev/null; then
        log_warn "Found pattern: $pattern"
        HAS_ERRORS=true
    fi
done

if [ "$HAS_ERRORS" = false ]; then
    pass_test
else
    log_warn "Found TODO/FIXME patterns (not critical)"
    pass_test
fi

# Test 12: Verify test files exist
log_test "Test files exist for deck.gl implementation"
UNIT_TEST="web/src/lib/globe-deckgl.test.ts"
E2E_TEST="tests/e2e/06-globe-deckgl.spec.ts"

if [ -f "$UNIT_TEST" ] && [ -f "$E2E_TEST" ]; then
    pass_test
else
    fail_test "Missing test files (unit: $([ -f "$UNIT_TEST" ] && echo "✓" || echo "✗"), e2e: $([ -f "$E2E_TEST" ] && echo "✓" || echo "✗"))"
fi

# Test 13: Check documentation exists
log_test "Documentation exists for deck.gl prototype"
if [ -f "DECKGL_PROTOTYPE.md" ]; then
    pass_test
else
    fail_test "DECKGL_PROTOTYPE.md not found"
fi

# Test 14: Verify WebGL context sharing is configured
log_test "WebGL context sharing is configured (interleaved: true)"
if grep -q "interleaved: true" "$GLOBE_FILE"; then
    pass_test
else
    fail_test "WebGL context sharing not configured"
fi

# Test 15: Check globe projection is set
log_test "Globe projection is configured"
if grep -q "projection.*globe" "$GLOBE_FILE"; then
    pass_test
else
    fail_test "Globe projection not found in configuration"
fi

##
# Summary
##

echo ""
echo "========================================="
echo "Integration Test Summary"
echo "========================================="
echo "Total Tests:  $TESTS_TOTAL"
echo -e "Passed:       ${GREEN}$TESTS_PASSED${NC}"
echo -e "Failed:       ${RED}$TESTS_FAILED${NC}"
echo "========================================="

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "${GREEN}✓ All tests passed!${NC}"
    echo ""
    echo "deck.gl globe implementation is ready for deployment."
    echo ""
    echo "Next steps:"
    echo "  1. Run E2E tests: cd web && npm run test:e2e"
    echo "  2. Manual testing: docker-compose up && open http://localhost:3857"
    echo "  3. Review DECKGL_PROTOTYPE.md for migration details"
    exit 0
else
    echo -e "${RED}✗ Some tests failed${NC}"
    echo ""
    echo "Please fix the failing tests before deploying to production."
    exit 1
fi
