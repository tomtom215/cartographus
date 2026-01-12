# Security Hardening Guide

This guide covers security configuration and best practices for Cartographus deployments. Follow these guidelines to ensure your deployment is secure against common threats.

**Related Documentation**:
- [SECRETS_MANAGEMENT.md](./SECRETS_MANAGEMENT.md) - Managing sensitive credentials
- [PRODUCTION_DEPLOYMENT.md](./PRODUCTION_DEPLOYMENT.md) - Production deployment guide
- [ADR-0003](./adr/0003-authentication-architecture.md) - Authentication architecture
- [ADR-0015](./adr/0015-zero-trust-authentication-authorization.md) - Zero Trust authentication

---

## Table of Contents

1. [Security Checklist](#security-checklist)
2. [Authentication Modes](#authentication-modes)
3. [TLS/SSL Configuration](#tlsssl-configuration)
4. [Authorization (RBAC)](#authorization-rbac)
5. [Rate Limiting](#rate-limiting)
6. [CORS Configuration](#cors-configuration)
7. [Security Headers](#security-headers)
8. [Audit Logging](#audit-logging)
9. [Network Security](#network-security)
10. [Data Protection](#data-protection)
11. [Incident Response](#incident-response)

---

## Security Checklist

Complete these items before production deployment:

### Critical (Must Do)

- [ ] Use HTTPS/TLS if internet-facing (see [TLS Configuration](#tlsssl-configuration))
- [ ] Set `AUTH_MODE` to `jwt`, `oidc`, or `plex` (never `none` in production)
- [ ] Generate strong `JWT_SECRET` (32+ chars, cryptographically random)
- [ ] Use strong `ADMIN_PASSWORD` (16+ chars, mixed case/numbers/symbols)
- [ ] Configure specific `CORS_ORIGINS` (never use `*` in production)
- [ ] Enable rate limiting (`RATE_LIMIT_REQUESTS`, `RATE_LIMIT_WINDOW`)
- [ ] Review and restrict firewall rules

> **Note for Home Lab Users:** TLS is optional for deployments on private networks (LAN-only).
> See [TLS Configuration](#tlsssl-configuration) for guidance on when TLS is recommended.

### Important (Should Do)

- [ ] Enable audit logging for security-relevant events
- [ ] Configure authorization policies (Casbin)
- [ ] Set up monitoring and alerting for security events
- [ ] Implement backup encryption (`BACKUP_ENCRYPTION_ENABLED`)
- [ ] Configure trusted proxies if behind load balancer

### Recommended (Nice to Have)

- [ ] Enable detection engine for anomaly detection
- [ ] Set up Discord/webhook notifications for security alerts
- [ ] Implement OIDC for enterprise identity management
- [ ] Regular security audits and penetration testing

---

## Authentication Modes

Cartographus supports multiple authentication modes to fit different deployment scenarios.

### Mode Comparison

| Mode | Security Level | Use Case | Configuration |
|------|----------------|----------|---------------|
| `none` | None | Development only | `AUTH_MODE=none` |
| `basic` | Low-Medium | Simple deployments | `AUTH_MODE=basic` |
| `jwt` | Medium-High | Standard production | `AUTH_MODE=jwt` |
| `oidc` | High | Enterprise with IdP | `AUTH_MODE=oidc` |
| `plex` | Medium-High | Plex-centric | `AUTH_MODE=plex` |
| `multi` | High | Flexible production | `AUTH_MODE=multi` |

### JWT Authentication (Recommended)

```bash
# Required settings
AUTH_MODE=jwt
JWT_SECRET=<cryptographically-random-32+-chars>
SESSION_TIMEOUT=24h

# Admin credentials
ADMIN_USERNAME=admin
ADMIN_PASSWORD=<strong-password>
```

**Best Practices:**
- Generate `JWT_SECRET` with: `openssl rand -base64 32`
- Rotate `JWT_SECRET` quarterly
- Use short token expiration (24h default)
- Never expose `JWT_SECRET` in logs or error messages

### OIDC Authentication (Enterprise)

```bash
# Enable OIDC
AUTH_MODE=oidc
OIDC_ENABLED=true

# Provider configuration
OIDC_ISSUER_URL=https://your-idp.example.com
OIDC_CLIENT_ID=cartographus
OIDC_CLIENT_SECRET=<client-secret>
OIDC_REDIRECT_URL=https://cartographus.example.com/api/v1/auth/oidc/callback

# Optional settings
OIDC_SCOPES=openid profile email
OIDC_PKCE_ENABLED=true
```

**Supported Providers:**
- Authelia
- Authentik
- Keycloak
- Google
- Okta
- Any OIDC-compliant provider

**Best Practices:**
- Enable PKCE for additional security
- Use short-lived tokens
- Configure proper scopes (minimum: `openid profile email`)
- Validate redirect URIs strictly

### Plex Authentication

```bash
AUTH_MODE=plex
PLEX_OAUTH_CLIENT_ID=<your-plex-app-id>
PLEX_OAUTH_REDIRECT_URI=https://cartographus.example.com/api/v1/auth/plex/callback
```

**Note:** Plex uses OAuth 2.0 with PKCE. Tokens are validated against Plex servers.

### Multi-Mode Authentication

```bash
# Enable multiple authentication methods
AUTH_MODE=multi

# Configure available methods
OIDC_ENABLED=true
OIDC_ISSUER_URL=https://your-idp.example.com
# ... other OIDC settings

# Basic auth as fallback
ADMIN_USERNAME=admin
ADMIN_PASSWORD=<strong-password>
```

**Priority Order:** OIDC > Plex > JWT > Basic

---

## TLS/SSL Configuration

### When is TLS Required?

| Deployment Type | TLS Required | Recommendation |
|-----------------|--------------|----------------|
| Internet-facing | **Required** | Use reverse proxy with Let's Encrypt |
| Home lab (LAN only) | Optional | Recommended if handling sensitive data |
| Docker behind VPN | Optional | VPN provides encryption layer |
| Development/testing | Not required | Use `AUTH_MODE=none` |

**For Home Lab Users:** TLS is **optional** for deployments on private networks. Cartographus will run without TLS, making setup easier for users who don't have domain names or public-facing infrastructure. This is intentional to lower the barrier to entry for self-hosters.

### When TLS is Strongly Recommended

Even for home labs, consider using TLS if:
- You access Cartographus from outside your home network (port forwarding, Tailscale, etc.)
- You use authentication (credentials are transmitted over the network)
- Your network has untrusted devices or users
- You handle sensitive playback analytics data

### Setting Up TLS (Optional)

If you choose to use TLS, it is typically handled by a reverse proxy. Below are configuration examples for common setups.

### Option 1: Nginx with Let's Encrypt

```nginx
server {
    listen 443 ssl http2;
    server_name cartographus.example.com;

    # TLS certificates
    ssl_certificate /etc/letsencrypt/live/cartographus.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/cartographus.example.com/privkey.pem;

    # TLS configuration
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384;
    ssl_prefer_server_ciphers off;
    ssl_session_cache shared:SSL:10m;
    ssl_session_timeout 1d;
    ssl_session_tickets off;

    # OCSP Stapling
    ssl_stapling on;
    ssl_stapling_verify on;
    resolver 8.8.8.8 8.8.4.4 valid=300s;

    # Security headers (see below)
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains; preload" always;

    location / {
        proxy_pass http://127.0.0.1:3857;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

### Option 2: Traefik with Auto-Cert

```yaml
# traefik.yml
entryPoints:
  websecure:
    address: ":443"
    http:
      tls:
        certResolver: letsencrypt

certificatesResolvers:
  letsencrypt:
    acme:
      email: admin@example.com
      storage: /letsencrypt/acme.json
      httpChallenge:
        entryPoint: web

tls:
  options:
    default:
      minVersion: VersionTLS12
      cipherSuites:
        - TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256
        - TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256
        - TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384
        - TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384
```

### Option 3: Cloudflare

Configure Cloudflare with:
- SSL/TLS mode: Full (strict)
- Minimum TLS version: 1.2
- Enable HSTS
- Configure trusted proxies in Cartographus:

```bash
TRUSTED_PROXIES=173.245.48.0/20,103.21.244.0/22,103.22.200.0/22,103.31.4.0/22
```

---

## Authorization (RBAC)

Cartographus uses Casbin for role-based access control.

### Role Hierarchy

```
viewer → Can read maps, analytics, own profile
   ↓
editor → inherits viewer + can create/edit maps
   ↓
admin  → inherits editor + full API access
```

### Configuration

```bash
# Enable Casbin authorization
AUTHZ_ENABLED=true

# Default role for authenticated users
AUTHZ_DEFAULT_ROLE=viewer

# Custom policy file (optional)
AUTHZ_POLICY_PATH=/etc/cartographus/policy.csv
```

### Custom Policy Example

```csv
# policy.csv
# Format: p, subject, resource, action

# Admin can do anything
p, admin, *, *

# Editor permissions
p, editor, maps, read
p, editor, maps, write
p, editor, analytics, read

# Viewer permissions
p, viewer, maps, read
p, viewer, analytics, read
p, viewer, profile, read
p, viewer, profile, write

# Role inheritance
g, editor, viewer
g, admin, editor
```

### API Endpoint Permissions

| Endpoint | Minimum Role | Description |
|----------|--------------|-------------|
| `GET /api/v1/stats/*` | viewer | Analytics data |
| `GET /api/v1/locations` | viewer | Location data |
| `POST /api/v1/backup/*` | admin | Backup operations |
| `POST /api/v1/admin/*` | admin | Administrative functions |
| `DELETE /api/v1/*` | admin | Delete operations |

---

## Rate Limiting

Rate limiting protects against brute force attacks and DoS.

### Configuration

```bash
# Global rate limit
RATE_LIMIT_REQUESTS=100
RATE_LIMIT_WINDOW=1m

# Disable for internal networks (not recommended)
# RATE_LIMIT_DISABLED=true
```

### Endpoint-Specific Limits

Cartographus applies different limits per endpoint:

| Endpoint Category | Limit | Window | Rationale |
|-------------------|-------|--------|-----------|
| Authentication (`/auth/*`) | 5 | 1 min | Brute force protection |
| Analytics (`/stats/*`) | 1000 | 1 min | High-traffic endpoints |
| Admin (`/admin/*`) | 10 | 1 min | Sensitive operations |
| Default | 100 | 1 min | General API usage |

### Rate Limit Response

When rate limited, clients receive:

```http
HTTP/1.1 429 Too Many Requests
Retry-After: 60
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 0
X-RateLimit-Reset: 1609459261
```

---

## CORS Configuration

Cross-Origin Resource Sharing controls which domains can access the API.

### Production Configuration

```bash
# Specific origins only - NEVER use * in production
CORS_ORIGINS=https://cartographus.example.com,https://app.example.com

# Separate multiple origins with commas
# No trailing slashes
# Include protocol (https://)
```

### CORS Headers Applied

```http
Access-Control-Allow-Origin: https://cartographus.example.com
Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS
Access-Control-Allow-Headers: Authorization, Content-Type
Access-Control-Max-Age: 86400
Access-Control-Allow-Credentials: true
```

### WebSocket CORS

WebSocket connections require valid Origin header:

```bash
# WebSocket origin validation is strict
# Empty Origin headers are rejected
# Only configured CORS_ORIGINS are allowed
```

---

## Security Headers

Configure these headers via your reverse proxy:

### Recommended Headers

```nginx
# Strict Transport Security (HSTS)
add_header Strict-Transport-Security "max-age=31536000; includeSubDomains; preload" always;

# Content Type Options
add_header X-Content-Type-Options nosniff always;

# Frame Options (clickjacking protection)
add_header X-Frame-Options DENY always;

# XSS Protection
add_header X-XSS-Protection "1; mode=block" always;

# Content Security Policy
add_header Content-Security-Policy "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; connect-src 'self' wss:" always;

# Referrer Policy
add_header Referrer-Policy "strict-origin-when-cross-origin" always;

# Permissions Policy
add_header Permissions-Policy "geolocation=(), microphone=(), camera=()" always;
```

### CSP for Cartographus

The recommended Content-Security-Policy:

```
default-src 'self';
script-src 'self' 'unsafe-inline';
style-src 'self' 'unsafe-inline';
img-src 'self' data: https:;
connect-src 'self' wss: https://api.maxmind.com https://plex.tv;
font-src 'self';
object-src 'none';
base-uri 'self';
frame-ancestors 'none';
```

---

## Audit Logging

Enable comprehensive audit logging for security monitoring.

### Configuration

```bash
# Enable structured logging
LOG_LEVEL=info
LOG_FORMAT=json

# Include caller information
LOG_CALLER=true
```

### Security Events Logged

| Event | Log Level | Example |
|-------|-----------|---------|
| Authentication success | INFO | `{"event":"auth_success","user":"admin","ip":"192.168.1.1"}` |
| Authentication failure | WARN | `{"event":"auth_failure","user":"admin","ip":"192.168.1.1","reason":"invalid_password"}` |
| Rate limit exceeded | WARN | `{"event":"rate_limit","ip":"192.168.1.1","endpoint":"/api/v1/auth/login"}` |
| Authorization denied | WARN | `{"event":"authz_denied","user":"viewer","resource":"/api/v1/admin/backup"}` |
| Detection alert | WARN | `{"event":"detection_alert","rule":"impossible_travel","user":"john"}` |
| Configuration change | INFO | `{"event":"config_change","setting":"rate_limit","old":"100","new":"200"}` |

### Audit Export

Export audit logs in CEF format for SIEM integration:

```bash
# API endpoint for audit export
curl -H "Authorization: Bearer $TOKEN" \
  "https://cartographus.example.com/api/v1/admin/audit/export?format=cef&from=2024-01-01"
```

### Log Retention

Configure log retention based on compliance requirements:

```bash
# Docker logging
docker run -d \
  --log-driver json-file \
  --log-opt max-size=100m \
  --log-opt max-file=10 \
  ghcr.io/tomtom215/cartographus:v1.0.0
```

---

## Network Security

### Firewall Rules

Allow only necessary traffic:

```bash
# Allow HTTPS (443)
ufw allow 443/tcp

# Allow HTTP for redirect (optional)
ufw allow 80/tcp

# Block direct access to application port from internet
# (if using reverse proxy)
ufw deny 3857/tcp
```

### Docker Network Isolation

```yaml
# docker-compose.yml
services:
  cartographus:
    networks:
      - internal
      - proxy

  nginx:
    networks:
      - proxy

networks:
  internal:
    internal: true  # No external access
  proxy:
    # External access via proxy only
```

### Trusted Proxies

When behind a load balancer, configure trusted proxies:

```bash
# Trust specific IPs/ranges
TRUSTED_PROXIES=172.17.0.0/16,10.0.0.0/8

# For Cloudflare
TRUSTED_PROXIES=173.245.48.0/20,103.21.244.0/22,103.22.200.0/22
```

This ensures:
- Correct client IP detection from `X-Forwarded-For`
- Rate limiting applies to actual client IPs
- Audit logs show real client IPs

---

## Data Protection

### Database Security

DuckDB database is stored locally. Protect it:

```bash
# Set restrictive permissions
chmod 600 /data/cartographus.duckdb

# Use encrypted storage (dm-crypt/LUKS)
# Mount encrypted volume to /data
```

### Backup Encryption

```bash
# Enable backup encryption
BACKUP_ENCRYPTION_ENABLED=true
BACKUP_ENCRYPTION_KEY=<32+-char-key>

# Generate key
openssl rand -base64 32
```

**Critical:** Store encryption key securely. Lost key = lost backups.

### Sensitive Data Handling

Cartographus handles the following sensitive data:

| Data Type | Storage | Protection |
|-----------|---------|------------|
| User passwords | Hashed (bcrypt) | Never stored plaintext |
| JWT tokens | Memory only | Short-lived, signed |
| API keys | Environment vars | Not logged |
| IP addresses | Database | Retention policy |
| Geolocation | Database | Anonymization optional |

### Data Retention

Configure data retention for compliance:

```bash
# Event retention (DuckDB)
# Configure via admin API or maintenance procedures

# Backup retention
BACKUP_RETENTION_MAX_DAYS=90
```

---

## Detection Engine

Enable security anomaly detection:

```bash
# Enable detection
DETECTION_ENABLED=true

# Trust score settings
DETECTION_TRUST_SCORE_DECREMENT=10
DETECTION_TRUST_SCORE_RECOVERY=1
DETECTION_TRUST_SCORE_THRESHOLD=50
```

### Detection Rules

| Rule | Description | Action |
|------|-------------|--------|
| Impossible Travel | User connects from geographically impossible locations | Alert |
| Concurrent Streams | Excessive simultaneous streams from one user | Alert |
| Device Velocity | Rapid device switching pattern | Alert |
| Geo Restriction | Access from blocked regions | Block |
| Simultaneous Locations | Same user from multiple locations | Alert |

### Alert Notifications

```bash
# Discord notifications
DISCORD_WEBHOOK_ENABLED=true
DISCORD_WEBHOOK_URL=https://discord.com/api/webhooks/xxx/yyy

# Generic webhook (Slack, PagerDuty, etc.)
WEBHOOK_ENABLED=true
WEBHOOK_URL=https://your-alerting-system.com/api/alerts
WEBHOOK_HEADERS=Authorization=Bearer xyz
```

---

## Incident Response

### Suspected Compromise

1. **Immediate Actions:**
   - Rotate `JWT_SECRET` (invalidates all sessions)
   - Change `ADMIN_PASSWORD`
   - Review audit logs for unauthorized access
   - Block suspicious IPs at firewall level

2. **Investigation:**
   ```bash
   # Search for failed login attempts
   docker logs cartographus 2>&1 | grep "auth_failure"

   # Search for rate limit violations
   docker logs cartographus 2>&1 | grep "rate_limit"

   # Export audit logs
   curl -H "Authorization: Bearer $TOKEN" \
     "https://cartographus.example.com/api/v1/admin/audit/export?format=json"
   ```

3. **Recovery:**
   - Restore from known-good backup if data compromised
   - Re-enable services with new credentials
   - Implement additional controls based on findings

### Security Contact

For security vulnerabilities, please:
1. **Do not** open public issues
2. Email security concerns privately
3. Allow 90 days for patch before disclosure

---

## Regular Security Tasks

### Weekly

- [ ] Review authentication failure logs
- [ ] Check rate limiting alerts
- [ ] Verify backup integrity

### Monthly

- [ ] Update container images to latest patch versions
- [ ] Review and rotate API keys if needed
- [ ] Audit user access and roles

### Quarterly

- [ ] Rotate `JWT_SECRET`
- [ ] Review security configuration
- [ ] Update TLS certificates (if not auto-renewed)
- [ ] Security training for team members

### Annually

- [ ] Full security audit
- [ ] Penetration testing
- [ ] Incident response drill
- [ ] Review and update this security guide

---

## References

- [OWASP Top 10](https://owasp.org/www-project-top-ten/)
- [NIST SP 800-63B - Authentication Guidelines](https://pages.nist.gov/800-63-3/sp800-63b.html)
- [CIS Docker Benchmark](https://www.cisecurity.org/benchmark/docker)
- [Mozilla SSL Configuration Generator](https://ssl-config.mozilla.org/)
