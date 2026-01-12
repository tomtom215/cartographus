# Troubleshooting Guide

This document covers common issues and their solutions for Cartographus.

**Related Documentation**:
- [DEVELOPMENT.md](./DEVELOPMENT.md) - Development setup
- [README.md](../README.md) - User documentation

---

## Table of Contents

1. [Build Issues](#build-issues)
2. [Runtime Issues](#runtime-issues)
3. [Network and DNS Issues](#network-and-dns-issues)
4. [Database Issues](#database-issues)
5. [Frontend Issues](#frontend-issues)
6. [E2E Test Issues](#e2e-test-issues)
7. [Common Error Messages](#common-error-messages)
8. [FAQ](#faq)

---

## Build Issues

### CGO Build Errors

```
Error: C compiler not found
```

**Cause**: DuckDB requires CGO, which needs a C compiler.

**Fix**:
```bash
# Ubuntu/Debian
sudo apt-get install build-essential

# macOS
xcode-select --install

# Ensure CGO is enabled
export CGO_ENABLED=1
```

### Frontend Build Errors

```
Error: Cannot find module 'maplibre-gl'
```

**Fix**:
```bash
cd web
npm ci
npm run build
```

### TypeScript Errors

```
error TS2304: Cannot find name 'X'
```

**Fix**:
```bash
cd web
npm install
npx tsc --noEmit  # Check for type errors
```

---

## Runtime Issues

### DuckDB Connection Errors

```
Error: failed to open database: database is locked
```

**Cause**: Multiple processes accessing same database file.

**Fix**: Ensure only one instance of the application is running.

### Authentication Issues

```
Error: JWT_SECRET must be at least 32 characters
```

**Fix**: Generate a secure JWT secret:
```bash
openssl rand -base64 48
```

### Rate Limiting Errors

```
Error: Too many requests
```

**Solutions**:
1. Increase rate limit: `RATE_LIMIT_REQUESTS=200`
2. Configure trusted proxies: `TRUSTED_PROXIES=10.0.0.1`
3. Check logs: `docker logs map | grep "Rate limit"`

### Performance Issues

**Slow map rendering or chart loading**

1. Increase memory: `DUCKDB_MAX_MEMORY=4GB`
2. Reduce batch size: `SYNC_BATCH_SIZE=500`
3. Check query times: `docker logs map | grep "query_time_ms"`
4. Use filtering to reduce data

### CORS Errors

```
Error: CORS policy errors
```

**Fix**:
```bash
# Specific domains (production)
CORS_ORIGINS=https://yourdomain.com,https://www.yourdomain.com

# Wildcard (development only)
CORS_ORIGINS=*
```

---

## Network and DNS Issues

### IPv6 DNS Resolution Failures (GitHub Actions / CI)

```
Error: dial tcp: lookup storage.googleapis.com on [::1]:53: read udp [::1]:33006->[::1]:53: read: connection refused
```

**Cause**: GitHub Actions runners have non-functional IPv6 networking.

**Solution** (in CI workflows):
```bash
# Restart DNS resolver
sudo systemctl restart systemd-resolved || true

# Force IPv4 fallback
if ! host -W 2 storage.googleapis.com &>/dev/null; then
    sudo sh -c 'echo "nameserver 8.8.8.8" > /etc/resolv.conf'
fi

# Force system DNS resolver
export GODEBUG=netdns=cgo
export GOTOOLCHAIN=local
```

### Sandboxed Environment DNS Limitations

**Cause**: Containerized environments without sudo access.

**Workaround**: Add static DNS entry (if network access exists):
```bash
echo "142.251.214.155 storage.googleapis.com" | sudo tee -a /etc/hosts
```

---

## Database Issues

### H3 Extension Installation Failures

```
Warning: H3 extension unavailable, H3 indexing will be disabled
```

**Root Cause**: Community extensions require explicit repository specification.

**Fix** (already implemented in database.go):
```sql
-- Correct syntax for community extensions
INSTALL h3 FROM community;
LOAD h3;
```

### DuckDB Spatial Column Binding Errors

```
INTERNAL Error: Failed to bind column reference ""
```

**Cause**: Geometry columns must be in GROUP BY when using spatial predicates.

**Fix**:
```sql
-- Wrong
SELECT g.latitude, g.longitude
FROM playback_events p
JOIN geolocations g ON p.ip_address = g.ip_address
WHERE ST_Within(g.geom, ST_MakeEnvelope(...))
GROUP BY g.latitude, g.longitude

-- Correct: Include g.geom in GROUP BY
GROUP BY g.latitude, g.longitude, g.geom
```

### DuckDB Extensions Not Loading

**Fix for Claude Code Web / sandboxed environments**:
```bash
# Pre-download extensions
./scripts/setup-duckdb-extensions.sh

# Or manually download:
mkdir -p ~/.duckdb/extensions/v1.4.3/linux_amd64
curl -o ~/.duckdb/extensions/v1.4.3/linux_amd64/spatial.duckdb_extension.gz \
  https://extensions.duckdb.org/v1.4.3/linux_amd64/spatial.duckdb_extension.gz
gunzip ~/.duckdb/extensions/v1.4.3/linux_amd64/spatial.duckdb_extension.gz
```

---

## Frontend Issues

### Analytics Dashboard Not Loading

**Solutions**:
1. Check browser console (F12) for JavaScript errors
2. Verify API endpoints:
   ```bash
   curl "http://localhost:3857/api/v1/analytics/trends?days=30"
   ```
3. Ensure data exists in selected date range
4. Clear filters and select "All Time"
5. Hard refresh (Ctrl+Shift+R)

### Map Not Rendering

**Solutions**:
1. Check WebGL support: `chrome://gpu`
2. Verify tile loading in Network tab
3. Check for CSP blocking errors

---

## E2E Test Issues

### ECharts SVG vs Canvas Rendering

**Issue**: Mobile tests fail because ECharts uses SVG on touch devices.

**Fix**: Use combined selectors:
```typescript
// Wrong
await page.locator('#chart canvas');

// Correct
await page.locator('#chart canvas, #chart svg');
```

### Keyboard Navigation Interception

**Issue**: ArrowRight/ArrowLeft intercepted by NavigationManager.

**Fix**: Use Home/End keys for chart navigation tests:
```typescript
// Wrong - intercepted by page navigation
await page.keyboard.press('ArrowRight');

// Correct - chart-specific navigation
await page.keyboard.press('Home');
await page.keyboard.press('End');
```

### Analytics Container Visibility Order

**Issue**: Tests fail checking child before parent is visible.

**Fix**: Wait for parent container first:
```typescript
await page.waitForSelector('#analytics-container:not([style*="display: none"])');
await page.waitForSelector('#analytics-overview', { state: 'visible' });
```

### Authentication Flow Timing

**Issue**: Logout tests fail due to timing.

**Fix**: Wait for login form to hide first:
```typescript
await page.click('#logout-button');
await page.waitForSelector('#login-container', { state: 'hidden' });
await page.waitForSelector('#app', { state: 'visible' });
```

### Mobile Viewport Element Access

**Issue**: Elements outside viewport on mobile.

**Fix**: Scroll into view before clicking:
```typescript
const element = page.locator('#button');
await element.scrollIntoViewIfNeeded();
await element.click();
```

---

## Common Error Messages

| Error Message | Cause | Solution |
|---------------|-------|----------|
| `JWT_SECRET must be at least 32 characters` | Secret too short | `openssl rand -base64 48` |
| `ADMIN_PASSWORD must be at least 12 characters` | Password too short | Use 12+ char password with mixed case, digit, special char |
| `Failed to connect to Tautulli` | Wrong URL/API key | Verify TAUTULLI_URL and API key |
| `Database is locked` | Multiple instances | Run only one container |
| `Out of memory` | DuckDB memory limit | Increase DUCKDB_MAX_MEMORY |
| `H3 extension unavailable` | Wrong install syntax | Use `INSTALL h3 FROM community;` |

---

## FAQ

**Q: Can I use this without Docker?**

A: Yes, build from source and run the binary directly.

**Q: Does this work with ARMv7 (32-bit)?**

A: No, DuckDB doesn't provide ARMv7 binaries. Only amd64 and arm64 supported.

**Q: How much disk space does the database use?**

A: ~50KB per 1,000 playback events. 100,000 events ≈ 5MB.

**Q: Can I run multiple instances?**

A: Yes, each needs its own data directory and port:
```bash
docker run -p 3857:3857 -v ./data1:/data ...  # Instance 1
docker run -p 3858:3857 -v ./data2:/data ...  # Instance 2
```

**Q: How do I back up my data?**

A: Copy the database file:
```bash
cp data/cartographus.duckdb data/cartographus.duckdb.backup
```

**Q: Can I import historical data beyond 24 hours?**

A: Yes, increase SYNC_LOOKBACK:
```bash
SYNC_LOOKBACK=720h  # 30 days
```

---

## Known Limitations

### Plex Token Revocation

**Limitation**: When users log out via Plex OAuth, the access token cannot be revoked on Plex's servers.

**Details**:
- Plex does not provide a public token revocation endpoint
- Access tokens remain valid for approximately 90 days after logout
- Cartographus clears all local session state (cookies, stored tokens)
- The token becomes unusable for Cartographus but may still work with Plex

**Security Implications**:
- If a token is compromised before logout, the attacker could continue using it
- Users should rotate their Plex account password if they suspect token compromise
- This is a Plex API limitation, not a Cartographus bug

**Mitigations**:
1. Tokens are stored securely in HTTP-only cookies
2. Session cookies are cleared on logout
3. Consider using OIDC authentication for enterprise deployments (supports proper token revocation)

**Reference**: `internal/api/handlers_plex_oauth.go:PlexOAuthRevoke()`

---

## Getting Help

1. Check existing issues: https://github.com/tomtom215/cartographus/issues
2. Enable debug logging: `LOG_LEVEL=debug`
3. Collect logs:
   ```bash
   docker logs map > map-logs.txt
   docker exec map env > map-env.txt
   ```
4. Open a new issue with logs (redact sensitive data)

---

## Debug Mode

```bash
# Enable debug logging
LOG_LEVEL=debug ./cartographus

# View verbose Go logs
go run -v ./cmd/server

# Check Docker container logs
docker logs map
```
