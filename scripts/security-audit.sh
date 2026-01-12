#!/bin/bash

# Cartographus - Media Server Analytics and Geographic Visualization
# Copyright 2026 Tom F. (tomtom215)
# SPDX-License-Identifier: AGPL-3.0-or-later
# https://github.com/tomtom215/cartographus
# RBAC Security Audit Script
# Tests authorization boundaries, privilege escalation, and security headers

set -e

BASE_URL="http://localhost:3857"
ADMIN_USER="admin"
ADMIN_PASS="SecureP@ss123!"
ADMIN_AUTH=$(echo -n "$ADMIN_USER:$ADMIN_PASS" | base64)

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

PASS_COUNT=0
FAIL_COUNT=0
WARN_COUNT=0

pass() {
    echo -e "${GREEN}[PASS]${NC} $1"
    ((PASS_COUNT++))
}

fail() {
    echo -e "${RED}[FAIL]${NC} $1"
    ((FAIL_COUNT++))
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
    ((WARN_COUNT++))
}

info() {
    echo -e "[INFO] $1"
}

echo "=================================================="
echo "     RBAC Security Audit - Cartographus"
echo "=================================================="
echo ""

# ===================================
# 1. PUBLIC ENDPOINT TESTS
# ===================================
echo "=== 1. PUBLIC ENDPOINT TESTS ==="

# Health endpoint should be public
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/api/v1/health")
if [ "$HTTP_CODE" == "200" ]; then
    pass "Health endpoint is public (200)"
else
    fail "Health endpoint should be public (got $HTTP_CODE)"
fi

# Health/live endpoint should be public
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/api/v1/health/live")
if [ "$HTTP_CODE" == "200" ]; then
    pass "Health/live endpoint is public (200)"
else
    fail "Health/live endpoint should be public (got $HTTP_CODE)"
fi

# Health/ready endpoint should be public
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/api/v1/health/ready")
if [ "$HTTP_CODE" == "200" ]; then
    pass "Health/ready endpoint is public (200)"
else
    fail "Health/ready endpoint should be public (got $HTTP_CODE)"
fi

echo ""

# ===================================
# 2. UNAUTHENTICATED ACCESS TESTS
# ===================================
echo "=== 2. UNAUTHENTICATED ACCESS TESTS ==="

# Protected endpoints should require auth
PROTECTED_ENDPOINTS=(
    "/api/v1/stats"
    "/api/v1/playbacks"
    "/api/v1/analytics/trends"
    "/api/v1/users"
    "/api/admin/roles"
    "/api/v1/backup/list"
    "/api/v1/sync"
)

for endpoint in "${PROTECTED_ENDPOINTS[@]}"; do
    HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL$endpoint")
    if [ "$HTTP_CODE" == "401" ]; then
        pass "Unauthenticated access to $endpoint returns 401"
    elif [ "$HTTP_CODE" == "403" ]; then
        pass "Unauthenticated access to $endpoint returns 403"
    else
        fail "Unauthenticated access to $endpoint should return 401/403 (got $HTTP_CODE)"
    fi
done

echo ""

# ===================================
# 3. ADMIN ENDPOINT AUTHORIZATION
# ===================================
echo "=== 3. ADMIN ENDPOINT AUTHORIZATION ==="

# Admin endpoints with valid admin auth
ADMIN_ENDPOINTS=(
    "/api/admin/roles"
    "/api/v1/backup/list"
)

for endpoint in "${ADMIN_ENDPOINTS[@]}"; do
    HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Basic $ADMIN_AUTH" "$BASE_URL$endpoint")
    if [ "$HTTP_CODE" == "200" ]; then
        pass "Admin can access $endpoint"
    else
        warn "Admin access to $endpoint returned $HTTP_CODE (might need additional setup)"
    fi
done

echo ""

# ===================================
# 4. AUTHENTICATION BYPASS TESTS
# ===================================
echo "=== 4. AUTHENTICATION BYPASS TESTS ==="

# Test invalid auth header formats
info "Testing invalid Authorization header formats..."

# Empty Authorization header
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: " "$BASE_URL/api/v1/stats")
if [ "$HTTP_CODE" == "401" ]; then
    pass "Empty Authorization header rejected"
else
    fail "Empty Authorization header should be rejected (got $HTTP_CODE)"
fi

# Malformed Basic auth (no Base64)
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Basic notbase64" "$BASE_URL/api/v1/stats")
if [ "$HTTP_CODE" == "401" ]; then
    pass "Malformed Basic auth rejected"
else
    fail "Malformed Basic auth should be rejected (got $HTTP_CODE)"
fi

# Wrong auth type
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer invalid" "$BASE_URL/api/v1/stats")
if [ "$HTTP_CODE" == "401" ]; then
    pass "Wrong auth type rejected"
else
    fail "Wrong auth type should be rejected (got $HTTP_CODE)"
fi

# Invalid credentials
INVALID_AUTH=$(echo -n "admin:wrongpassword" | base64)
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Basic $INVALID_AUTH" "$BASE_URL/api/v1/stats")
if [ "$HTTP_CODE" == "401" ]; then
    pass "Invalid credentials rejected"
else
    fail "Invalid credentials should be rejected (got $HTTP_CODE)"
fi

# SQL injection in username
SQL_AUTH=$(echo -n "admin' OR '1'='1:password" | base64)
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Basic $SQL_AUTH" "$BASE_URL/api/v1/stats")
if [ "$HTTP_CODE" == "401" ]; then
    pass "SQL injection in username rejected"
else
    fail "SQL injection in username should be rejected (got $HTTP_CODE)"
fi

# Null byte injection
NULL_AUTH=$(printf "admin\x00admin:password" | base64)
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Basic $NULL_AUTH" "$BASE_URL/api/v1/stats")
if [ "$HTTP_CODE" == "401" ]; then
    pass "Null byte injection rejected"
else
    fail "Null byte injection should be rejected (got $HTTP_CODE)"
fi

echo ""

# ===================================
# 5. SECURITY HEADERS TESTS
# ===================================
echo "=== 5. SECURITY HEADERS TESTS ==="

HEADERS=$(curl -s -I "$BASE_URL/api/v1/health")

# X-Content-Type-Options
if echo "$HEADERS" | grep -qi "X-Content-Type-Options: nosniff"; then
    pass "X-Content-Type-Options: nosniff present"
else
    fail "X-Content-Type-Options header missing or incorrect"
fi

# X-Frame-Options
if echo "$HEADERS" | grep -qi "X-Frame-Options"; then
    pass "X-Frame-Options header present"
else
    warn "X-Frame-Options header missing"
fi

# Content-Security-Policy
if echo "$HEADERS" | grep -qi "Content-Security-Policy"; then
    pass "Content-Security-Policy header present"
else
    warn "Content-Security-Policy header missing"
fi

# Strict-Transport-Security (might not be set for HTTP)
if echo "$HEADERS" | grep -qi "Strict-Transport-Security"; then
    pass "HSTS header present"
else
    warn "HSTS header missing (OK for HTTP, required for HTTPS)"
fi

# X-XSS-Protection (deprecated but still useful)
if echo "$HEADERS" | grep -qi "X-XSS-Protection"; then
    pass "X-XSS-Protection header present"
else
    info "X-XSS-Protection header missing (deprecated, CSP preferred)"
fi

# Cache-Control for API endpoints
if echo "$HEADERS" | grep -qi "Cache-Control"; then
    pass "Cache-Control header present"
else
    warn "Cache-Control header missing"
fi

echo ""

# ===================================
# 6. RATE LIMITING TESTS
# ===================================
echo "=== 6. RATE LIMITING TESTS ==="

info "Testing rate limiting (sending 20 rapid requests)..."
RATE_LIMITED=false
for i in {1..20}; do
    HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/api/v1/health")
    if [ "$HTTP_CODE" == "429" ]; then
        RATE_LIMITED=true
        pass "Rate limiting triggered after $i requests"
        break
    fi
done

if [ "$RATE_LIMITED" = false ]; then
    warn "Rate limiting not triggered for public endpoint (may be intentional)"
fi

# Test auth endpoint rate limiting (more strict)
info "Testing auth endpoint rate limiting..."
for i in {1..10}; do
    INVALID_AUTH=$(echo -n "admin:wrongpassword$i" | base64)
    HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Basic $INVALID_AUTH" "$BASE_URL/api/v1/stats")
    if [ "$HTTP_CODE" == "429" ]; then
        pass "Auth rate limiting triggered after $i failed attempts"
        break
    fi
done

echo ""

# ===================================
# 7. SQL INJECTION TESTS
# ===================================
echo "=== 7. SQL INJECTION TESTS ==="

info "Testing SQL injection in query parameters..."

SQL_PAYLOADS=(
    "1' OR '1'='1"
    "1; DROP TABLE users;--"
    "1 UNION SELECT * FROM users--"
    "' OR 1=1--"
    "'; EXEC xp_cmdshell('whoami');--"
    "1 AND 1=1"
    "1' AND '1'='1"
)

for payload in "${SQL_PAYLOADS[@]}"; do
    ENCODED=$(python3 -c "import urllib.parse; print(urllib.parse.quote('$payload'))")

    # Test in query parameter
    HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Basic $ADMIN_AUTH" \
        "$BASE_URL/api/v1/playbacks?filter=$ENCODED")

    if [ "$HTTP_CODE" == "500" ]; then
        fail "SQL injection might be possible - server error on: $payload"
    else
        pass "SQL injection attempt handled safely: $payload"
    fi
done

echo ""

# ===================================
# 8. PATH TRAVERSAL TESTS
# ===================================
echo "=== 8. PATH TRAVERSAL TESTS ==="

PATH_PAYLOADS=(
    "../../../etc/passwd"
    "..%2f..%2f..%2fetc%2fpasswd"
    "....//....//....//etc/passwd"
    "..%252f..%252f..%252fetc%252fpasswd"
)

for payload in "${PATH_PAYLOADS[@]}"; do
    HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/api/v1/$payload")

    if [ "$HTTP_CODE" == "404" ] || [ "$HTTP_CODE" == "400" ] || [ "$HTTP_CODE" == "401" ]; then
        pass "Path traversal blocked: $payload"
    else
        fail "Path traversal might be possible (got $HTTP_CODE): $payload"
    fi
done

echo ""

# ===================================
# 9. CORS TESTS
# ===================================
echo "=== 9. CORS TESTS ==="

# Test CORS preflight
CORS_RESPONSE=$(curl -s -I -X OPTIONS \
    -H "Origin: https://evil.com" \
    -H "Access-Control-Request-Method: GET" \
    "$BASE_URL/api/v1/stats")

# Check if evil origin is reflected (potential vulnerability)
if echo "$CORS_RESPONSE" | grep -qi "Access-Control-Allow-Origin: https://evil.com"; then
    fail "CORS allows arbitrary origins - security vulnerability!"
elif echo "$CORS_RESPONSE" | grep -qi "Access-Control-Allow-Origin: \*"; then
    warn "CORS allows all origins (*) - review if intentional"
else
    pass "CORS does not reflect arbitrary origins"
fi

# Check if credentials are allowed
if echo "$CORS_RESPONSE" | grep -qi "Access-Control-Allow-Credentials: true"; then
    if echo "$CORS_RESPONSE" | grep -qi "Access-Control-Allow-Origin: \*"; then
        fail "CORS allows credentials with wildcard origin - critical vulnerability!"
    else
        pass "CORS credentials allowed but origin is restricted"
    fi
fi

echo ""

# ===================================
# 10. PRIVILEGE ESCALATION TESTS
# ===================================
echo "=== 10. PRIVILEGE ESCALATION TESTS ==="

# Test role assignment API (should require admin)
info "Testing role assignment as unauthenticated user..."
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST \
    -H "Content-Type: application/json" \
    -d '{"user_id":"victim","role":"admin"}' \
    "$BASE_URL/api/admin/roles/assign")

if [ "$HTTP_CODE" == "401" ] || [ "$HTTP_CODE" == "403" ]; then
    pass "Unauthenticated role assignment blocked"
else
    fail "Unauthenticated role assignment not blocked (got $HTTP_CODE)"
fi

# Test changing policy as unauthenticated
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST \
    -H "Content-Type: application/json" \
    -d '{"subject":"hacker","object":"/*","action":"*"}' \
    "$BASE_URL/api/admin/policies")

if [ "$HTTP_CODE" == "401" ] || [ "$HTTP_CODE" == "403" ] || [ "$HTTP_CODE" == "404" ]; then
    pass "Unauthenticated policy modification blocked"
else
    fail "Unauthenticated policy modification not blocked (got $HTTP_CODE)"
fi

echo ""

# ===================================
# 11. XSS PREVENTION TESTS
# ===================================
echo "=== 11. XSS PREVENTION TESTS ==="

XSS_PAYLOADS=(
    "<script>alert('XSS')</script>"
    "<img src=x onerror=alert('XSS')>"
    "javascript:alert('XSS')"
    "<svg onload=alert('XSS')>"
)

info "Testing XSS payload handling in responses..."
for payload in "${XSS_PAYLOADS[@]}"; do
    ENCODED=$(python3 -c "import urllib.parse; print(urllib.parse.quote('$payload'))")

    RESPONSE=$(curl -s -H "Authorization: Basic $ADMIN_AUTH" \
        "$BASE_URL/api/v1/playbacks?filter=$ENCODED" 2>/dev/null)

    # Check if payload is reflected unescaped
    if echo "$RESPONSE" | grep -q "<script>"; then
        fail "XSS payload reflected unescaped in response"
    else
        pass "XSS payload handled safely: ${payload:0:30}..."
    fi
done

echo ""

# ===================================
# 12. HTTP METHOD TESTS
# ===================================
echo "=== 12. HTTP METHOD TESTS ==="

# Test that DELETE on protected resources requires auth
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "$BASE_URL/api/v1/playbacks/1")
if [ "$HTTP_CODE" == "401" ] || [ "$HTTP_CODE" == "403" ]; then
    pass "DELETE without auth blocked"
else
    fail "DELETE without auth not blocked (got $HTTP_CODE)"
fi

# Test that PATCH on protected resources requires auth
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X PATCH "$BASE_URL/api/v1/users/1")
if [ "$HTTP_CODE" == "401" ] || [ "$HTTP_CODE" == "403" ]; then
    pass "PATCH without auth blocked"
else
    fail "PATCH without auth not blocked (got $HTTP_CODE)"
fi

# Test TRACE method (should be disabled)
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X TRACE "$BASE_URL/")
if [ "$HTTP_CODE" == "405" ] || [ "$HTTP_CODE" == "501" ]; then
    pass "TRACE method disabled"
else
    warn "TRACE method might be enabled (got $HTTP_CODE)"
fi

echo ""

# ===================================
# 13. API VERSIONING TESTS
# ===================================
echo "=== 13. API VERSIONING TESTS ==="

# Test that old API versions don't expose vulnerabilities
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/api/v0/health")
if [ "$HTTP_CODE" == "404" ]; then
    pass "Old API version (v0) not accessible"
else
    warn "Old API version (v0) might be accessible (got $HTTP_CODE)"
fi

HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/api/health")
if [ "$HTTP_CODE" == "404" ]; then
    pass "Unversioned API not accessible"
else
    info "Unversioned API accessible (might be intentional, got $HTTP_CODE)"
fi

echo ""

# ===================================
# 14. ERROR DISCLOSURE TESTS
# ===================================
echo "=== 14. ERROR DISCLOSURE TESTS ==="

# Test that errors don't leak sensitive information
ERROR_RESPONSE=$(curl -s -H "Authorization: Basic $ADMIN_AUTH" \
    "$BASE_URL/api/v1/nonexistent/endpoint")

# Check for stack traces
if echo "$ERROR_RESPONSE" | grep -qi "stack\|trace\|panic\|goroutine"; then
    fail "Error response contains stack trace"
else
    pass "Error response does not leak stack traces"
fi

# Check for internal paths
if echo "$ERROR_RESPONSE" | grep -qi "/home/\|/root/\|/var/\|internal/"; then
    fail "Error response contains internal paths"
else
    pass "Error response does not leak internal paths"
fi

# Check for database errors
if echo "$ERROR_RESPONSE" | grep -qi "sql\|query\|database\|duckdb"; then
    warn "Error response might contain database information"
else
    pass "Error response does not leak database details"
fi

echo ""

# ===================================
# SUMMARY
# ===================================
echo "=================================================="
echo "     SECURITY AUDIT SUMMARY"
echo "=================================================="
echo -e "${GREEN}PASSED:${NC} $PASS_COUNT"
echo -e "${RED}FAILED:${NC} $FAIL_COUNT"
echo -e "${YELLOW}WARNINGS:${NC} $WARN_COUNT"
echo "=================================================="

if [ $FAIL_COUNT -gt 0 ]; then
    echo -e "${RED}SECURITY AUDIT FAILED - Critical issues found${NC}"
    exit 1
else
    if [ $WARN_COUNT -gt 0 ]; then
        echo -e "${YELLOW}SECURITY AUDIT PASSED WITH WARNINGS${NC}"
    else
        echo -e "${GREEN}SECURITY AUDIT PASSED${NC}"
    fi
    exit 0
fi
