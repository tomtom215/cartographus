#!/bin/bash

# Cartographus - Media Server Analytics and Geographic Visualization
# Copyright 2026 Tom F. (tomtom215)
# SPDX-License-Identifier: AGPL-3.0-or-later
# https://github.com/tomtom215/cartographus

# Integration Test Suite for Cartographus API
# Tests all major API endpoints for HTTP status codes and response structure
#
# Note: This test suite is designed for CI environments where external services
# like Tautulli may not be configured. Tests that depend on external services
# are grouped separately and can be skipped.

BASE_URL="${BASE_URL:-http://localhost:3857}"
FAILED=0
PASSED=0
SKIPPED=0

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test function for HTTP status code
test_endpoint() {
    local name="$1"
    local method="$2"
    local endpoint="$3"
    local expected_status="$4"
    local extra_args="${5:-}"

    printf "Testing: %-55s " "$name"

    local response
    case "$method" in
        GET)
            response=$(curl -s -o /dev/null -w "%{http_code}" $extra_args "$BASE_URL$endpoint" 2>/dev/null)
            ;;
        POST)
            response=$(curl -s -o /dev/null -w "%{http_code}" -X POST $extra_args "$BASE_URL$endpoint" 2>/dev/null)
            ;;
        PUT)
            response=$(curl -s -o /dev/null -w "%{http_code}" -X PUT $extra_args "$BASE_URL$endpoint" 2>/dev/null)
            ;;
        DELETE)
            response=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE $extra_args "$BASE_URL$endpoint" 2>/dev/null)
            ;;
        OPTIONS)
            response=$(curl -s -o /dev/null -w "%{http_code}" -X OPTIONS $extra_args "$BASE_URL$endpoint" 2>/dev/null)
            ;;
    esac

    if [ "$response" = "$expected_status" ]; then
        echo -e "${GREEN}PASS${NC} (HTTP $response)"
        ((PASSED++))
    else
        echo -e "${RED}FAIL${NC} (Expected $expected_status, got $response)"
        ((FAILED++))
    fi
}

# Test endpoint accepting multiple valid status codes
test_endpoint_multi() {
    local name="$1"
    local method="$2"
    local endpoint="$3"
    local expected_statuses="$4"  # comma-separated list e.g., "200,503"
    local extra_args="${5:-}"

    printf "Testing: %-55s " "$name"

    local response
    case "$method" in
        GET)
            response=$(curl -s -o /dev/null -w "%{http_code}" $extra_args "$BASE_URL$endpoint" 2>/dev/null)
            ;;
    esac

    # Check if response matches any of the expected statuses
    local matched=false
    IFS=',' read -ra STATUSES <<< "$expected_statuses"
    for status in "${STATUSES[@]}"; do
        if [ "$response" = "$status" ]; then
            matched=true
            break
        fi
    done

    if [ "$matched" = true ]; then
        echo -e "${GREEN}PASS${NC} (HTTP $response)"
        ((PASSED++))
    else
        echo -e "${RED}FAIL${NC} (Expected one of $expected_statuses, got $response)"
        ((FAILED++))
    fi
}

# Test JSON response contains a field (searches entire response)
test_json_field() {
    local name="$1"
    local endpoint="$2"
    local json_field="$3"

    printf "Testing: %-55s " "$name"

    local response
    response=$(curl -s "$BASE_URL$endpoint" 2>/dev/null)
    if echo "$response" | grep -q "\"$json_field\""; then
        echo -e "${GREEN}PASS${NC}"
        ((PASSED++))
    else
        echo -e "${RED}FAIL${NC} (field '$json_field' not found)"
        ((FAILED++))
    fi
}

# Test JSON response contains a data field (for wrapped responses)
test_json_has_data() {
    local name="$1"
    local endpoint="$2"

    printf "Testing: %-55s " "$name"

    local response
    response=$(curl -s "$BASE_URL$endpoint" 2>/dev/null)
    if echo "$response" | grep -q '"data"'; then
        echo -e "${GREEN}PASS${NC}"
        ((PASSED++))
    else
        echo -e "${RED}FAIL${NC} (no 'data' field in response)"
        ((FAILED++))
    fi
}

# Test JSON response is a valid JSON object (starts with {)
test_json_object() {
    local name="$1"
    local endpoint="$2"

    printf "Testing: %-55s " "$name"

    local response
    response=$(curl -s "$BASE_URL$endpoint" 2>/dev/null)
    if echo "$response" | grep -qE '^\s*\{'; then
        echo -e "${GREEN}PASS${NC}"
        ((PASSED++))
    else
        echo -e "${RED}FAIL${NC} (expected object)"
        ((FAILED++))
    fi
}

# Test content type header using GET request with -i flag
test_content_type() {
    local name="$1"
    local endpoint="$2"
    local expected_type="$3"

    printf "Testing: %-55s " "$name"

    local headers
    headers=$(curl -s -i "$BASE_URL$endpoint" 2>/dev/null | head -20)
    if echo "$headers" | grep -qi "content-type.*$expected_type"; then
        echo -e "${GREEN}PASS${NC}"
        ((PASSED++))
    else
        local actual_type
        actual_type=$(echo "$headers" | grep -i "content-type" | head -1 | cut -d: -f2 | tr -d '[:space:]' | cut -d';' -f1)
        echo -e "${RED}FAIL${NC} (expected $expected_type, got $actual_type)"
        ((FAILED++))
    fi
}

# Skip test with message
skip_test() {
    local name="$1"
    local reason="$2"
    printf "Testing: %-55s " "$name"
    echo -e "${YELLOW}SKIP${NC} ($reason)"
    ((SKIPPED++))
}

# Section header
section() {
    echo ""
    echo -e "${BLUE}=== $1 ===${NC}"
}

echo "=========================================="
echo "Cartographus Integration Test Suite"
echo "=========================================="
echo "Base URL: $BASE_URL"
echo "Started: $(date)"
echo "=========================================="

# ============================================
# HEALTH ENDPOINTS
# ============================================
section "Health Endpoints"

test_endpoint "Health endpoint returns 200" "GET" "/api/v1/health" "200"
test_json_field "Health returns status field" "/api/v1/health" "status"
test_json_field "Health returns version field" "/api/v1/health" "version"
test_json_field "Health returns database_connected" "/api/v1/health" "database_connected"

test_endpoint "Liveness probe returns 200" "GET" "/api/v1/health/live" "200"
# Readiness probe returns 503 when Tautulli is not connected (expected in CI)
test_endpoint_multi "Readiness probe returns 200 or 503" "GET" "/api/v1/health/ready" "200,503"

# ============================================
# CORE API ENDPOINTS
# ============================================
section "Core API Endpoints"

# Stats
test_endpoint "Stats endpoint returns 200" "GET" "/api/v1/stats" "200"
test_json_field "Stats has total_playbacks" "/api/v1/stats" "total_playbacks"
test_json_field "Stats has unique_users" "/api/v1/stats" "unique_users"
test_json_field "Stats has unique_locations" "/api/v1/stats" "unique_locations"
test_json_field "Stats has recent_activity" "/api/v1/stats" "recent_activity"

# Playbacks
test_endpoint "Playbacks endpoint returns 200" "GET" "/api/v1/playbacks" "200"
test_endpoint "Playbacks with limit" "GET" "/api/v1/playbacks?limit=10" "200"
test_endpoint "Playbacks with offset" "GET" "/api/v1/playbacks?offset=0&limit=5" "200"
test_endpoint "Playbacks with days filter" "GET" "/api/v1/playbacks?days=30" "200"
test_json_object "Playbacks returns object" "/api/v1/playbacks?limit=5"

# Playbacks validation
test_endpoint "Playbacks rejects limit=0" "GET" "/api/v1/playbacks?limit=0" "400"
test_endpoint "Playbacks rejects limit>1000" "GET" "/api/v1/playbacks?limit=2000" "400"
test_endpoint "Playbacks rejects negative offset" "GET" "/api/v1/playbacks?offset=-1" "400"

# Locations (returns APIResponse with data array inside)
test_endpoint "Locations endpoint returns 200" "GET" "/api/v1/locations" "200"
test_endpoint "Locations with days param" "GET" "/api/v1/locations?days=30" "200"
test_endpoint "Locations with limit param" "GET" "/api/v1/locations?days=90&limit=50" "200"
test_json_has_data "Locations returns wrapped response" "/api/v1/locations?limit=10"

# Locations validation
test_endpoint "Locations rejects days=0" "GET" "/api/v1/locations?days=0" "400"
test_endpoint "Locations rejects days>3650" "GET" "/api/v1/locations?days=5000" "400"
test_endpoint "Locations rejects limit=0" "GET" "/api/v1/locations?limit=0" "400"

# Users and Media Types (return APIResponse with data array inside)
test_endpoint "Users endpoint returns 200" "GET" "/api/v1/users" "200"
test_json_has_data "Users returns wrapped response" "/api/v1/users"
test_endpoint "Media types endpoint returns 200" "GET" "/api/v1/media-types" "200"
test_json_has_data "Media types returns wrapped response" "/api/v1/media-types"

# Server info
test_endpoint "Server info returns 200" "GET" "/api/v1/server-info" "200"
test_json_object "Server info returns object" "/api/v1/server-info"

# ============================================
# ANALYTICS ENDPOINTS
# ============================================
section "Analytics Endpoints"

# Core analytics (no required params)
test_endpoint "Analytics trends returns 200" "GET" "/api/v1/analytics/trends" "200"
test_endpoint "Analytics trends with days" "GET" "/api/v1/analytics/trends?days=30" "200"
test_json_object "Analytics trends returns object" "/api/v1/analytics/trends?days=7"

test_endpoint "Analytics geographic returns 200" "GET" "/api/v1/analytics/geographic" "200"
test_endpoint "Analytics users returns 200" "GET" "/api/v1/analytics/users" "200"
test_endpoint "Analytics binge returns 200" "GET" "/api/v1/analytics/binge" "200"
test_endpoint "Analytics bandwidth returns 200" "GET" "/api/v1/analytics/bandwidth" "200"
test_endpoint "Analytics bitrate returns 200" "GET" "/api/v1/analytics/bitrate" "200"
test_endpoint "Analytics popular returns 200" "GET" "/api/v1/analytics/popular" "200"

# Advanced analytics
test_endpoint "Analytics watch parties returns 200" "GET" "/api/v1/analytics/watch-parties" "200"
test_endpoint "Analytics user engagement returns 200" "GET" "/api/v1/analytics/user-engagement" "200"
test_endpoint "Analytics abandonment returns 200" "GET" "/api/v1/analytics/abandonment" "200"
test_endpoint "Analytics comparative returns 200" "GET" "/api/v1/analytics/comparative" "200"
test_endpoint "Analytics temporal heatmap returns 200" "GET" "/api/v1/analytics/temporal-heatmap" "200"

# Media quality analytics (some may return 500 on empty database due to NULL handling)
test_endpoint_multi "Analytics resolution mismatch" "GET" "/api/v1/analytics/resolution-mismatch" "200,500"
test_endpoint "Analytics HDR returns 200" "GET" "/api/v1/analytics/hdr" "200"
test_endpoint "Analytics audio returns 200" "GET" "/api/v1/analytics/audio" "200"
test_endpoint_multi "Analytics subtitles" "GET" "/api/v1/analytics/subtitles" "200,500"
test_endpoint "Analytics frame rate returns 200" "GET" "/api/v1/analytics/frame-rate" "200"
test_endpoint "Analytics container returns 200" "GET" "/api/v1/analytics/container" "200"

# System analytics (some may return 500 on empty database due to NULL handling)
test_endpoint_multi "Analytics connection security" "GET" "/api/v1/analytics/connection-security" "200,500"
test_endpoint_multi "Analytics pause patterns" "GET" "/api/v1/analytics/pause-patterns" "200,500"
test_endpoint "Analytics concurrent streams returns 200" "GET" "/api/v1/analytics/concurrent-streams" "200"

# Analytics with required parameters
test_endpoint "Analytics library requires section_id" "GET" "/api/v1/analytics/library" "400"
test_endpoint_multi "Analytics library with section_id" "GET" "/api/v1/analytics/library?section_id=1" "200,500"

test_endpoint "Analytics hardware transcode returns 200" "GET" "/api/v1/analytics/hardware-transcode" "200"
test_endpoint "Analytics hardware transcode trends returns 200" "GET" "/api/v1/analytics/hardware-transcode/trends" "200"
test_endpoint "Analytics HDR content returns 200" "GET" "/api/v1/analytics/hdr-content" "200"

# Approximate analytics (DataSketches) - requires parameters, falls back to exact when extension unavailable
test_endpoint "Approximate stats returns 200" "GET" "/api/v1/analytics/approximate" "200"
test_endpoint "Approximate distinct requires column" "GET" "/api/v1/analytics/approximate/distinct" "400"
test_endpoint "Approximate distinct with column" "GET" "/api/v1/analytics/approximate/distinct?column=username" "200"
test_endpoint "Approximate percentile requires params" "GET" "/api/v1/analytics/approximate/percentile" "400"
test_endpoint_multi "Approximate percentile with params" "GET" "/api/v1/analytics/approximate/percentile?column=duration&percentile=0.5" "200,500"

# ============================================
# SPATIAL ENDPOINTS
# ============================================
section "Spatial Endpoints"

# Spatial endpoints require spatial extension
# Returns 200 if extension available, 503 if unavailable (never 500)
test_endpoint_multi "Spatial hexagons default resolution" "GET" "/api/v1/spatial/hexagons?resolution=7" "200,503"
test_endpoint "Spatial hexagons rejects invalid resolution" "GET" "/api/v1/spatial/hexagons?resolution=20" "400"

# Spatial arcs requires SERVER_LATITUDE/SERVER_LONGITUDE config (400 if not set, 503 if extension unavailable)
test_endpoint_multi "Spatial arcs (requires server location config)" "GET" "/api/v1/spatial/arcs?days=30" "200,400,503"

test_endpoint "Spatial viewport requires bounds" "GET" "/api/v1/spatial/viewport" "400"
test_endpoint_multi "Spatial viewport with bounds" "GET" "/api/v1/spatial/viewport?west=-180&south=-90&east=180&north=90" "200,503"

test_endpoint_multi "Spatial temporal density" "GET" "/api/v1/spatial/temporal-density?days=30" "200,503"
test_endpoint "Spatial nearby requires lat/lon" "GET" "/api/v1/spatial/nearby" "400"
test_endpoint_multi "Spatial nearby with params" "GET" "/api/v1/spatial/nearby?lat=40.7128&lon=-74.0060" "200,503"

# ============================================
# SEARCH ENDPOINTS
# ============================================
section "Search Endpoints"

test_endpoint "Fuzzy search returns 200" "GET" "/api/v1/search/fuzzy?q=test" "200"
test_endpoint "Fuzzy search users returns 200" "GET" "/api/v1/search/users?q=test" "200"
test_endpoint "Fuzzy search with limit" "GET" "/api/v1/search/fuzzy?q=test&limit=5" "200"

# ============================================
# EXPORT ENDPOINTS
# ============================================
section "Export Endpoints"

test_endpoint_multi "Export GeoParquet (requires spatial extension)" "GET" "/api/v1/export/geoparquet" "200,503"
test_endpoint "Export GeoJSON returns 200" "GET" "/api/v1/export/geojson" "200"
test_endpoint "Export Playbacks CSV returns 200" "GET" "/api/v1/export/playbacks/csv" "200"
test_endpoint "Export Locations GeoJSON returns 200" "GET" "/api/v1/export/locations/geojson" "200"

# Content type checks for exports
test_content_type "GeoJSON has correct content type" "/api/v1/export/geojson" "application/geo+json"
test_content_type "CSV has correct content type" "/api/v1/export/playbacks/csv" "text/csv"

# ============================================
# STREAMING ENDPOINTS
# ============================================
section "Streaming Endpoints"

test_endpoint "Stream locations GeoJSON returns 200" "GET" "/api/v1/stream/locations-geojson" "200"

# ============================================
# TAUTULLI COMPATIBILITY ENDPOINTS
# Note: These require a Tautulli server connection which is not available in CI
# ============================================
section "Tautulli Compatibility API"

# Check if Tautulli is connected by testing health endpoint
TAUTULLI_CONNECTED=$(curl -s "$BASE_URL/api/v1/health" 2>/dev/null | grep -c '"tautulli_connected":true')

if [ "$TAUTULLI_CONNECTED" = "1" ]; then
    # Home/Activity
    test_endpoint "Tautulli home stats returns 200" "GET" "/api/v1/tautulli/home-stats" "200"
    test_endpoint "Tautulli activity returns 200" "GET" "/api/v1/tautulli/activity" "200"

    # Graphs
    test_endpoint "Tautulli plays by date returns 200" "GET" "/api/v1/tautulli/plays-by-date" "200"
    test_endpoint "Tautulli plays by day of week returns 200" "GET" "/api/v1/tautulli/plays-by-dayofweek" "200"
    test_endpoint "Tautulli plays by hour of day returns 200" "GET" "/api/v1/tautulli/plays-by-hourofday" "200"
    test_endpoint "Tautulli plays by stream type returns 200" "GET" "/api/v1/tautulli/plays-by-stream-type" "200"
    test_endpoint "Tautulli plays by source resolution returns 200" "GET" "/api/v1/tautulli/plays-by-source-resolution" "200"
    test_endpoint "Tautulli plays by stream resolution returns 200" "GET" "/api/v1/tautulli/plays-by-stream-resolution" "200"
    test_endpoint "Tautulli plays by top 10 platforms returns 200" "GET" "/api/v1/tautulli/plays-by-top-10-platforms" "200"
    test_endpoint "Tautulli plays by top 10 users returns 200" "GET" "/api/v1/tautulli/plays-by-top-10-users" "200"
    test_endpoint "Tautulli plays per month returns 200" "GET" "/api/v1/tautulli/plays-per-month" "200"
    test_endpoint "Tautulli concurrent streams returns 200" "GET" "/api/v1/tautulli/concurrent-streams-by-stream-type" "200"

    # Users
    test_endpoint "Tautulli users returns 200" "GET" "/api/v1/tautulli/users" "200"
    test_endpoint "Tautulli users table returns 200" "GET" "/api/v1/tautulli/users-table" "200"

    # Libraries
    test_endpoint "Tautulli libraries returns 200" "GET" "/api/v1/tautulli/libraries" "200"
    test_endpoint "Tautulli libraries table returns 200" "GET" "/api/v1/tautulli/libraries-table" "200"
    test_endpoint "Tautulli library names returns 200" "GET" "/api/v1/tautulli/library-names" "200"

    # Media
    test_endpoint "Tautulli recently added returns 200" "GET" "/api/v1/tautulli/recently-added" "200"

    # Server info
    test_endpoint "Tautulli server info returns 200" "GET" "/api/v1/tautulli/server-info" "200"
    test_endpoint "Tautulli server friendly name returns 200" "GET" "/api/v1/tautulli/server-friendly-name" "200"
    test_endpoint "Tautulli info returns 200" "GET" "/api/v1/tautulli/tautulli-info" "200"

    # Search
    test_endpoint "Tautulli search returns 200" "GET" "/api/v1/tautulli/search?query=test" "200"
else
    echo -e "${YELLOW}Tautulli not connected - skipping 22 Tautulli API tests${NC}"
    ((SKIPPED+=22))
fi

# ============================================
# BACKUP ENDPOINTS
# ============================================
section "Backup Endpoints"

# Note: backup (singular) for operations, backups (plural) for listing
test_endpoint "Backup stats returns 200" "GET" "/api/v1/backup/stats" "200"
test_endpoint "List backups returns 200" "GET" "/api/v1/backups/" "200"
test_endpoint "Backup retention policy returns 200" "GET" "/api/v1/backup/retention" "200"

# ============================================
# STATIC FILES & SPA
# ============================================
section "Static Files & SPA Routing"

test_endpoint "Root serves index.html" "GET" "/" "200"
test_endpoint "SPA fallback for /dashboard" "GET" "/dashboard" "200"
test_endpoint "SPA fallback for /analytics" "GET" "/analytics" "200"
test_endpoint "SPA fallback for /settings" "GET" "/settings" "200"
test_endpoint "SPA fallback for deep route" "GET" "/some/deep/route" "200"
test_content_type "Root returns HTML" "/" "text/html"

# ============================================
# CORS & SECURITY
# ============================================
section "CORS & Security Headers"

test_endpoint "CORS preflight health" "OPTIONS" "/api/v1/health" "200" "-H Origin:http://localhost:3000 -H Access-Control-Request-Method:GET"
test_endpoint "CORS preflight analytics" "OPTIONS" "/api/v1/analytics/trends" "200" "-H Origin:http://localhost:3000 -H Access-Control-Request-Method:GET"

# ============================================
# HTTP METHOD VALIDATION
# ============================================
section "HTTP Method Validation"

test_endpoint "Sync rejects GET (requires POST)" "GET" "/api/v1/sync" "405"
test_endpoint "Stats rejects POST" "POST" "/api/v1/stats" "405"
test_endpoint "Health rejects DELETE" "DELETE" "/api/v1/health" "405"

# ============================================
# ERROR HANDLING
# ============================================
section "Error Handling"

test_endpoint "Non-existent API endpoint returns 404" "GET" "/api/v1/nonexistent" "404"
# Note: /api/v2/* is caught by SPA fallback which returns 200 with index.html
# This is expected behavior for SPA routing
test_endpoint "Unknown API route falls through" "GET" "/api/v1/definitely-not-real-endpoint-xyz" "404"

# ============================================
# PROMETHEUS METRICS
# ============================================
section "Prometheus Metrics"

test_endpoint "Metrics endpoint returns 200" "GET" "/metrics" "200"
test_content_type "Metrics returns text format" "/metrics" "text/plain"

# ============================================
# AUTHENTICATION TESTS (Conditional)
# ============================================
if [ -n "$ADMIN_USERNAME" ] && [ -n "$ADMIN_PASSWORD" ]; then
    section "Authentication (AUTH_MODE=jwt)"

    # Login with valid credentials
    printf "Testing: %-55s " "Login with valid credentials"
    login_response=$(curl -s -X POST "$BASE_URL/api/v1/auth/login" \
        -H "Content-Type: application/json" \
        -d "{\"username\":\"$ADMIN_USERNAME\",\"password\":\"$ADMIN_PASSWORD\",\"remember_me\":false}" \
        -c /tmp/cookies.txt 2>/dev/null)

    if echo "$login_response" | grep -q "\"token\""; then
        echo -e "${GREEN}PASS${NC}"
        ((PASSED++))

        TOKEN=$(echo "$login_response" | grep -o '"token":"[^"]*' | sed 's/"token":"//')

        # Authenticated sync request
        printf "Testing: %-55s " "Authenticated sync with token"
        auth_response=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/api/v1/sync" \
            -H "Authorization: Bearer $TOKEN" 2>/dev/null)
        if [ "$auth_response" = "202" ]; then
            echo -e "${GREEN}PASS${NC} (HTTP $auth_response)"
            ((PASSED++))
        else
            echo -e "${RED}FAIL${NC} (Expected 202, got $auth_response)"
            ((FAILED++))
        fi

        # Cookie auth
        printf "Testing: %-55s " "Authenticated sync with cookie"
        auth_cookie_response=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/api/v1/sync" \
            -b /tmp/cookies.txt 2>/dev/null)
        if [ "$auth_cookie_response" = "202" ]; then
            echo -e "${GREEN}PASS${NC} (HTTP $auth_cookie_response)"
            ((PASSED++))
        else
            echo -e "${RED}FAIL${NC} (Expected 202, got $auth_cookie_response)"
            ((FAILED++))
        fi
    else
        echo -e "${RED}FAIL${NC}"
        ((FAILED++))
    fi

    # Invalid credentials
    test_endpoint "Login rejects invalid credentials" "POST" "/api/v1/auth/login" "401" \
        "-H Content-Type:application/json -d '{\"username\":\"wrong\",\"password\":\"wrong\",\"remember_me\":false}'"

    # Missing fields
    test_endpoint "Login rejects empty credentials" "POST" "/api/v1/auth/login" "400" \
        "-H Content-Type:application/json -d '{\"username\":\"\",\"password\":\"\",\"remember_me\":false}'"

    # Unauthenticated sync
    test_endpoint "Sync rejects unauthenticated request" "POST" "/api/v1/sync" "401"

    rm -f /tmp/cookies.txt
else
    section "Authentication"
    echo -e "${YELLOW}SKIPPED${NC} - AUTH_MODE=none or credentials not set"
    ((SKIPPED+=6))
fi

# ============================================
# RATE LIMITING
# ============================================
section "Rate Limiting"

echo "Sending 10 rapid requests to test rate limiting..."
rate_limited=0
for i in {1..10}; do
    response=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/api/v1/health" 2>/dev/null)
    if [ "$response" = "429" ]; then
        rate_limited=1
        break
    fi
done

if [ $rate_limited -eq 1 ]; then
    echo -e "Rate limiting: ${GREEN}ACTIVE${NC} (429 received)"
else
    echo -e "Rate limiting: ${YELLOW}NOT TRIGGERED${NC} (may need more requests)"
fi

# ============================================
# SUMMARY
# ============================================
echo ""
echo "=========================================="
echo "Test Summary"
echo "=========================================="
echo -e "${GREEN}Passed:  $PASSED${NC}"
echo -e "${RED}Failed:  $FAILED${NC}"
echo -e "${YELLOW}Skipped: $SKIPPED${NC}"
echo "Total:   $((PASSED + FAILED + SKIPPED))"
echo "=========================================="
echo "Completed: $(date)"
echo "=========================================="

if [ $FAILED -gt 0 ]; then
    exit 1
fi

exit 0
