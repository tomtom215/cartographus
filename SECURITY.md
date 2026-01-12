# Security Policy

**Last Updated**: 2026-01-11

## Reporting a Vulnerability

We take security seriously. If you discover a security vulnerability in Cartographus, please report it responsibly:

1. **DO NOT** open a public GitHub issue for security vulnerabilities
2. **DO** use GitHub Security Advisories: https://github.com/tomtom215/cartographus/security/advisories/new
3. **DO** include:
   - Description of the vulnerability
   - Steps to reproduce
   - Potential impact
   - Suggested fix (if any)

### Response Timeline

- **Initial Response**: Within 48 hours
- **Status Updates**: Every 7 days until resolved
- **Fix Timeline**: Critical issues within 7 days, High within 14 days, Medium within 30 days

---

## Supported Versions

| Version | Supported | Notes |
|---------|-----------|-------|
| latest (main branch) | ✅ Yes | Active development, security patches applied immediately |
| Tagged releases | ✅ Yes | Security patches backported to latest stable release |
| Development branches | ⚠️ Limited | Security updates on best-effort basis |

---

## Security Vulnerability Assessment

### Current Status (2025-11-27)

**Base Image**: Debian 13 (Trixie)
**Security Scan Status**: 2 vulnerabilities fixed, 1 false positive documented

### Recent Vulnerabilities (Debian 12 Bookworm)

The following vulnerabilities were detected in the previous Debian 12 (Bookworm) base image:

| CVE | Package | Severity | Status in Bookworm | Status in Trixie |
|-----|---------|----------|-------------------|------------------|
| CVE-2023-2953 | libldap-2.5-0 | HIGH | Affected | ✅ FIXED (v2.6.10+dfsg-1) |
| CVE-2025-6020 | libpam-* | HIGH | Affected | ✅ FIXED (v1.7.0-5) |
| CVE-2023-45853 | zlib1g | CRITICAL | Ignored | ⚠️ FALSE POSITIVE |

---

## Mitigation: Debian 13 (Trixie) Migration

**Date**: 2025-11-27
**Action**: Upgraded all Dockerfile base images from Debian 12 to Debian 13

### Changes Made
```dockerfile
# Before (Debian 12 Bookworm)
FROM node:25-bookworm-slim
FROM golang:1.25.5-bookworm
FROM debian:bookworm-slim

# After (Debian 13 Trixie)
FROM node:25-trixie-slim
FROM golang:1.25.5-trixie
FROM debian:trixie-slim
```

**Result**: 2 HIGH severity vulnerabilities resolved

---

## Vulnerability Details

### CVE-2023-2953: OpenLDAP Null Pointer Dereference (HIGH)

**Status**: ✅ FIXED in Debian 13 (Trixie)

**Description**: Null pointer dereference in `ber_memalloc_x()` function. Low risk in practice but flagged as HIGH severity.

**Fix Details**:
- Fixed in OpenLDAP 2.5.16+dfsg-1
- Trixie ships with 2.6.10+dfsg-1 (contains fix)

**References**:
- [Debian Security Tracker](https://security-tracker.debian.org/tracker/CVE-2023-2953)
- [NVD Details](https://nvd.nist.gov/vuln/detail/CVE-2023-2953)

---

### CVE-2025-6020: Linux-PAM Directory Traversal (HIGH)

**Status**: ✅ FIXED in Debian 13 (Trixie)

**Description**: `pam_namespace` module allows privilege escalation via symlink attacks and race conditions.

**Fix Details**:
- Fixed in linux-pam 1.7.0-5 (upstream 1.7.1)
- Converted functions to use file descriptors instead of paths

**References**:
- [Debian Security Tracker](https://security-tracker.debian.org/tracker/CVE-2025-6020)
- [Debian Bug #1107919](https://bugs.debian.org/cgi-bin/bugreport.cgi?bug=1107919)
- [NVD Details](https://nvd.nist.gov/vuln/detail/CVE-2025-6020)

---

### CVE-2023-45853: zlib Integer Overflow (FALSE POSITIVE)

**Status**: ⚠️ FALSE POSITIVE (No actual risk)

**Why This Is Safe**:
The vulnerability exists in the **MiniZip** component, which is **NOT built or shipped** in Debian's zlib binary packages. Only the source code exists; no vulnerable binaries are installed.

**Explanation**:
- Debian's `src:zlib` contains MiniZip source code
- MiniZip binaries are **NOT produced** from this source
- Only core zlib library is built and shipped
- Security scanners flag source code, not actual binaries

**Debian Rationale**: "contrib/minizip not built and src:zlib not producing binary packages"

**References**:
- [Debian Security Tracker](https://security-tracker.debian.org/tracker/CVE-2023-45853)
- [Debian Bug #1054290](https://bugs.debian.org/cgi-bin/bugreport.cgi?bug=1054290)
- [Trivy Discussion #6722](https://github.com/aquasecurity/trivy/discussions/6722)

**Suppress in Scanners** (optional):
```bash
# Create .trivyignore
echo "CVE-2023-45853  # zlib MiniZip not built in Debian binaries" > .trivyignore
```

---

## Security Best Practices

### Configuration Security

**CRITICAL**: Before deploying to production, ensure:

1. **JWT_SECRET**: Must be a unique, randomly generated string (32+ characters)
   ```bash
   # Generate a secure secret:
   openssl rand -base64 32
   ```
   - **NEVER** use placeholder values like `REPLACE_WITH...` or `CHANGEME`
   - The application will refuse to start with detected placeholder values

2. **ADMIN_PASSWORD**: Must be a strong password (12+ characters for JWT auth)
   - Requires: uppercase, lowercase, digit, and special character (NIST SP 800-63B)
   - Use a password manager to generate
   - **NEVER** use placeholder values

3. **AUTH_MODE**:
   - **NEVER** use `AUTH_MODE=none` in production
   - Recommended: `jwt` or `oidc` for production environments
   - The application logs prominent warnings when authentication is disabled

4. **HTTPS/TLS**:
   - **ALWAYS** deploy behind a TLS-terminating reverse proxy (Nginx, Caddy, Traefik)
   - Set `TRUSTED_PROXIES` to your proxy IP(s)
   - Never expose HTTP directly to the internet

5. **Rate Limiting**:
   - **NEVER** disable rate limiting in production (`DISABLE_RATE_LIMIT=true`)
   - The application logs warnings when rate limiting is disabled

### Application Security

1. **Authentication**: JWT tokens with HTTP-only cookies
2. **Authorization**: Role-based access control
3. **Rate Limiting**: Sliding window algorithm per IP (auth 5/min, analytics 1000/min, default 100/min)
4. **Input Validation**: All user input validated and sanitized
5. **SQL Injection Prevention**: Parameterized queries only (DuckDB)
6. **XSS Prevention**: Content Security Policy (CSP) headers + HTML escaping
7. **CSRF Protection**: SameSite cookies + CORS configuration

### Infrastructure Security

1. **Minimal Base Image**: Debian Trixie slim variant
2. **Non-Root User**: Runs as user `map` (UID 1000)
3. **Least Privilege**: Minimal system packages installed
4. **Security Headers**: HSTS, X-Frame-Options, X-Content-Type-Options
5. **TLS/HTTPS**: Recommended via reverse proxy (Nginx, Caddy)

### Dependency Management

1. **Go Modules**: Locked versions in `go.sum`
2. **npm Packages**: Locked versions in `package-lock.json`
3. **Regular Updates**: Monthly dependency review and updates
4. **Vulnerability Scanning**: Trivy in CI/CD pipeline

---

## Security Scanning

### Automated Scanning (CI/CD)

Security scans run automatically on:
- Every push to `main` branch
- Every pull request
- Weekly scheduled scans

**Tools Used**:
- **Trivy**: Container image and dependency scanning
- **Gitleaks**: Secret detection in git history
- **Cosign**: Container image signing (keyless)
- **Syft**: SBOM generation

### Manual Scanning

```bash
# Scan Docker image
docker build -t map:security-test .
docker run --rm -v /var/run/docker.sock:/var/run/docker.sock \
  aquasec/trivy:latest image map:security-test

# Scan dependencies
trivy fs --scanners vuln,misconfig .

# Scan for secrets
docker run --rm -v $(pwd):/path zricethezav/gitleaks:latest \
  detect --source /path
```

---

## Continuous Monitoring

### Recommended Practices

1. **Base Image Updates**: Rebuild monthly for security patches
2. **Dependency Updates**: Review and update quarterly
3. **CVE Monitoring**: Subscribe to Debian security announcements
4. **Security Audits**: Annual third-party security review
5. **Penetration Testing**: Recommended for production deployments

### Security Contacts

- **GitHub Security Advisories**: https://github.com/tomtom215/cartographus/security/advisories
- **Issue Tracker**: https://github.com/tomtom215/cartographus/issues
- **Debian Security**: https://www.debian.org/security/

---

## Changelog

### 2025-11-27: Debian 13 (Trixie) Migration
- **Action**: Upgraded all Dockerfile base images from Debian 12 to Debian 13
- **Vulnerabilities Fixed**: CVE-2023-2953, CVE-2025-6020
- **False Positives Documented**: CVE-2023-45853
- **Impact**: 2 HIGH severity vulnerabilities resolved

---

## Acknowledgments

We appreciate security researchers who responsibly disclose vulnerabilities. Contributors will be acknowledged in release notes (unless they prefer to remain anonymous).

---

## License

This security policy is part of the Cartographus project and is subject to the same license terms.
