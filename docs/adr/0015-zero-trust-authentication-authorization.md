# ADR-0015: Zero Trust Authentication and Authorization

**Date**: 2025-12-05
**Updated**: 2026-01-11
**Status**: Accepted (Revised - Zitadel Migration)

---

## Context

Cartographus requires secure authentication and fine-grained authorization for production deployments. The existing ADR-0003 defined multi-mode authentication with Basic and JWT support. Users have requested OIDC integration to connect with existing identity providers in their homelab environments.

### Requirements

1. **OIDC Support**: Integration with external identity providers (Authelia, Authentik, Keycloak, etc.)
2. **Fine-Grained Authorization**: Role-based access control (RBAC) for API endpoints
3. **Backwards Compatibility**: Existing `AUTH_MODE=none|basic|jwt` must continue to work
4. **Offline Capability**: Application should function when IdP is temporarily unavailable
5. **Single Binary**: All components embedded, no external services required
6. **Plex Integration**: Leverage existing Plex OAuth implementation
7. **OpenID Certification**: Use certified library for security compliance
8. **Production-Grade Observability**: Prometheus metrics, audit logging, durable state storage

### Alternatives Considered

| Approach | Pros | Cons |
|----------|------|------|
| **External OAuth Proxy** (oauth2-proxy) | Simple, well-tested | Extra service to deploy, not self-contained |
| **Zitadel OIDC Library** | OpenID Foundation certified, RP+OP modes | Larger dependency footprint |
| **Custom OIDC Implementation** | Minimal dependencies | Security risk, maintenance burden |
| **Casbin for Authorization** | Sub-millisecond RBAC, battle-tested | Learning curve for policy language |

---

## Decision

Implement zero trust authentication and authorization using:

1. **Zitadel OIDC v3.45.1** - OpenID Foundation certified library for OIDC Relying Party
2. **Casbin v2.135.0** for embedded RBAC authorization
3. **BadgerDB-backed state storage** for ACID-compliant OIDC state persistence
4. **Prometheus metrics** for OIDC observability
5. **Audit logging** for security event tracking
6. **AuthSubject abstraction** for unified identity across auth methods

### Why Zitadel OIDC (Revision 2026-01-04)

The initial implementation used a custom OIDC library. This was replaced with the OpenID Foundation certified Zitadel library for:

| Aspect | Custom Implementation | Zitadel OIDC |
|--------|----------------------|--------------|
| **Certification** | None | OpenID Foundation certified |
| **PKCE** | Manual implementation | Built-in RFC 7636 compliance |
| **Nonce validation** | Manual | Automatic with replay protection |
| **Token validation** | Custom JWT parsing | Certified token verification |
| **Back-channel logout** | Not supported | Full OIDC Back-Channel Logout spec |
| **Maintenance** | Internal burden | Community-maintained |

### Authentication Modes (Extended)

| Mode | Description | Use Case |
|------|-------------|----------|
| `none` | No authentication | Development, trusted networks |
| `basic` | HTTP Basic Auth | Simple deployments |
| `jwt` | JWT Bearer tokens | API access |
| `oidc` | OpenID Connect | Enterprise IdP integration |
| `plex` | Plex OAuth 2.0 | Plex-centric deployments with automatic role assignment |
| `multi` | Try multiple methods | Flexible production setup |

### Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         HTTP Request                                     │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                    AuthMiddleware (auth/middleware.go)                   │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                     Mode Selection (ADR-0003 + ADR-0015)         │   │
│  │  ┌────────┐ ┌───────┐ ┌─────┐ ┌──────┐ ┌──────┐ ┌───────┐      │   │
│  │  │  none  │ │ basic │ │ jwt │ │ oidc │ │ plex │ │ multi │      │   │
│  │  └────────┘ └───────┘ └─────┘ └──────┘ └──────┘ └───────┘      │   │
│  └─────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                      AuthSubject (Normalized Claims)                     │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                    AuthzMiddleware (authz/middleware.go)                 │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                     Casbin SyncedEnforcer                        │   │
│  │     e.Enforce(subject.ID, resource, action) → bool               │   │
│  └─────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                         HTTP Handler                                     │
└─────────────────────────────────────────────────────────────────────────┘
```

### Authorization Model

RBAC with role hierarchy using Casbin:

```
viewer → Can read maps, analytics, own profile
   ↓
editor → inherits viewer + can create/edit maps
   ↓
admin  → inherits editor + full API access
```

### Key Design Decisions

1. **AuthSubject as Common Interface**: All auth methods produce an `AuthSubject` struct, enabling:
   - Consistent authorization checks
   - Backwards compatibility via `ToClaims()` conversion
   - Future extensibility for new auth methods

2. **Embedded Policies**: Casbin model and default policies are embedded via `go:embed`, with optional file-based overrides for customization.

3. **JWKS Caching**: Offline token verification via cached JWKS with TTL, enabling operation during IdP outages.

4. **Priority-Based Multi-Auth**: In `multi` mode, authenticators are tried in priority order (OIDC → Plex → JWT → Basic).

### Plex Server Owner Detection

When using `AUTH_MODE=plex` or `AUTH_MODE=multi`, Cartographus automatically detects Plex server ownership and assigns appropriate roles. This enables zero-configuration onboarding for Plex server owners.

**How It Works:**

1. User authenticates via Plex OAuth 2.0 with PKCE
2. After token validation, Cartographus queries the Plex `/api/v2/resources` endpoint
3. Server ownership is detected by checking the `owned` field on server resources
4. Roles are assigned based on ownership status:

| User Type | Detection | Assigned Role |
|-----------|-----------|---------------|
| **Server Owner** | `resource.owned == true` | `admin` (full access) |
| **Shared Admin** | Has access token, not owner | `editor` (configurable) |
| **Shared User** | Has access, no special privileges | `viewer` (default) |
| **Plex Pass Subscriber** | `subscription.active == true` | Additional role (optional) |

**Configuration:**

| Variable | Default | Description |
|----------|---------|-------------|
| `PLEX_SERVER_DETECTION` | `true` | Enable automatic server ownership detection |
| `PLEX_SERVER_OWNER_ROLE` | `admin` | Role assigned to server owners |
| `PLEX_SERVER_ADMIN_ROLE` | `editor` | Role for shared admins (if detected) |
| `PLEX_SERVER_MACHINE_ID` | (empty) | Limit detection to specific server |

**User Journey:**

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          Plex Server Owner                                   │
│                                                                              │
│  1. Clicks "Sign in with Plex" on Cartographus                              │
│  2. Redirected to plex.tv OAuth                                             │
│  3. Authorizes Cartographus                                                 │
│  4. Cartographus receives Plex token                                        │
│  5. Queries /api/v2/resources → detects owned server                        │
│  6. Automatically assigned "admin" role                                     │
│  7. Full dashboard access, all users visible                                │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│                          Shared Plex User                                    │
│                                                                              │
│  1. Server owner shares Cartographus URL                                    │
│  2. User clicks "Sign in with Plex"                                         │
│  3. Redirected to plex.tv OAuth                                             │
│  4. Authorizes Cartographus                                                 │
│  5. Queries /api/v2/resources → no owned servers, has access                │
│  6. Automatically assigned "viewer" role                                    │
│  7. Dashboard shows only their own playback data (RBAC enforced)            │
└─────────────────────────────────────────────────────────────────────────────┘
```

**Security Considerations:**

- Server ownership is verified via Plex API at every authentication
- Roles are assigned based on real-time Plex server status
- Graceful degradation: if detection fails, user gets default roles
- Server machine identifier filter prevents role escalation via secondary servers

---

## Consequences

### Positive

- Support for all major OIDC providers (Authelia, Authentik, Keycloak, Google, Okta)
- Fine-grained RBAC with sub-millisecond policy evaluation
- Offline capability via JWKS caching
- Single binary deployment maintained
- Backwards compatible with existing auth modes
- Plex OAuth integration preserved and formalized

### Negative

- Increased binary size (~8-12 MB from new dependencies)
- Additional configuration complexity for OIDC
- Users must configure OIDC provider for enterprise features

### Neutral

- Learning curve for Casbin policy language
- Migration path required for existing deployments wanting RBAC

---

## Production-Grade Features

### Durable State Storage (BadgerDB)

OIDC state (code verifier, nonce, redirect) is stored in BadgerDB for ACID compliance:

```go
// State survives server restarts
store, err := auth.NewBadgerZitadelStateStore("/data/oidc-state")
if err != nil {
    log.Fatal(err)
}
defer store.Close()

// Automatic cleanup of expired states
store.StartCleanupRoutine(ctx, 5*time.Minute)
```

| Feature | Benefit |
|---------|---------|
| ACID transactions | No state corruption on crash |
| TTL-based expiry | Automatic cleanup of stale states |
| Disk persistence | Survives container restarts |
| Concurrent access | Thread-safe operations |

### Prometheus Metrics

15+ metrics for OIDC observability at `/metrics`:

| Metric | Type | Description |
|--------|------|-------------|
| `oidc_login_attempts_total` | Counter | Login attempts by provider and outcome |
| `oidc_login_duration_seconds` | Histogram | Login latency distribution |
| `oidc_logout_total` | Counter | Logout events by type (RP-initiated, back-channel) |
| `oidc_token_refresh_total` | Counter | Token refresh attempts |
| `oidc_token_refresh_duration_seconds` | Histogram | Refresh latency |
| `oidc_state_store_operations_total` | Counter | State store operations |
| `oidc_state_store_size` | Gauge | Current number of pending states |
| `oidc_back_channel_logout_total` | Counter | Back-channel logout events |
| `oidc_validation_errors_total` | Counter | Token validation errors by type |
| `oidc_sessions_created_total` | Counter | Session creation events |
| `oidc_sessions_terminated_total` | Counter | Session termination by reason |
| `oidc_active_sessions` | Gauge | Currently active OIDC sessions |

### Audit Logging

Security events logged for compliance and forensics:

```go
logger := auth.NewOIDCAuditLogger(auditStore, "keycloak")

// Logged events:
// - Login success/failure with IP, user agent, duration
// - Logout (RP-initiated and back-channel)
// - Token refresh success/failure
// - Session creation/termination
// - Validation errors
```

| Event | Severity | Fields |
|-------|----------|--------|
| Login success | Info | user_id, username, email, roles, duration, ip, user_agent |
| Login failure | Warning | error_code, error_description, ip, user_agent |
| Logout | Info | user_id, session_id, logout_type, has_id_token_hint |
| Back-channel logout | Info | subject, session_id, sessions_terminated |
| Token refresh | Info/Warning | user_id, success, duration, new_expiry |
| Session created | Info | session_id, user_id, expiry |

---

## Dependencies

```go
// go.mod additions
require (
    github.com/zitadel/oidc/v3 v3.45.1  // OpenID Foundation certified OIDC library
    github.com/casbin/casbin/v2 v2.135.0
    github.com/dgraph-io/badger/v4 v4.9.0  // ACID-compliant state storage
    github.com/prometheus/client_golang v1.23.2  // Metrics
)
```

---

## Configuration

### Environment Variables (OIDC)

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `AUTH_MODE` | Yes | `none` | Auth mode: none, basic, jwt, oidc, plex, multi |
| `OIDC_ENABLED` | No | `false` | Enable OIDC authentication |
| `OIDC_ISSUER_URL` | If OIDC | - | OIDC provider issuer URL |
| `OIDC_CLIENT_ID` | If OIDC | - | OAuth2 client ID |
| `OIDC_CLIENT_SECRET` | If OIDC | - | OAuth2 client secret |
| `OIDC_REDIRECT_URL` | If OIDC | - | Callback URL |
| `OIDC_SCOPES` | No | `openid profile email` | Requested scopes |
| `OIDC_PKCE_ENABLED` | No | `true` | Enable PKCE |

### Environment Variables (Plex)

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `PLEX_ENABLED` | No | `false` | Enable Plex OAuth authentication |
| `PLEX_CLIENT_ID` | If Plex | - | Plex application client ID |
| `PLEX_CLIENT_SECRET` | No | - | Plex application client secret (optional) |
| `PLEX_REDIRECT_URI` | If Plex | - | OAuth callback URL |
| `PLEX_DEFAULT_ROLES` | No | `viewer` | Default roles for Plex users |
| `PLEX_PLEX_PASS_ROLE` | No | - | Additional role for Plex Pass subscribers |
| `PLEX_SERVER_DETECTION` | No | `true` | Enable automatic server ownership detection |
| `PLEX_SERVER_OWNER_ROLE` | No | `admin` | Role assigned to Plex server owners |
| `PLEX_SERVER_ADMIN_ROLE` | No | `editor` | Role for shared server admins |
| `PLEX_SERVER_MACHINE_ID` | No | - | Limit detection to specific server |
| `PLEX_TIMEOUT` | No | `30s` | Timeout for Plex API requests |

### Environment Variables (Authorization)

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `AUTHZ_ENABLED` | No | `true` | Enable Casbin authorization |
| `AUTHZ_MODEL_PATH` | No | embedded | Path to Casbin model file |
| `AUTHZ_POLICY_PATH` | No | embedded | Path to Casbin policy file |
| `AUTHZ_DEFAULT_ROLE` | No | `viewer` | Default role for authenticated users |

---

## Implementation Files

### Core OIDC Files (Zitadel)

| File | Purpose |
|------|---------|
| `internal/auth/zitadel_rp.go` | Zitadel OIDC Relying Party setup and configuration |
| `internal/auth/zitadel_flow.go` | ZitadelStateStore interface, memory implementation, flow logic |
| `internal/auth/zitadel_state_store_badger.go` | BadgerDB-backed durable state storage (ACID-compliant) |
| `internal/auth/flow_handlers_oidc.go` | OIDC login, callback, logout, refresh handlers |
| `internal/auth/flow_handlers_backchannel.go` | Back-channel logout endpoint (OIDC spec) |
| `internal/auth/metrics.go` | 15+ Prometheus metrics for OIDC observability |
| `internal/auth/audit_logger.go` | Security event logging for all auth events |

### Plex Authentication Files

| File | Purpose |
|------|---------|
| `internal/auth/plex_authenticator.go` | Plex OAuth + server owner detection |
| `internal/auth/plex_authenticator_test.go` | 18 tests covering server detection and authentication |

### Authorization Files

| File | Purpose |
|------|---------|
| `internal/auth/subject.go` | AuthSubject struct, AuthMode enum, Authenticator interface |
| `internal/auth/oidc_config.go` | OIDC configuration and validation |
| `internal/authz/enforcer.go` | Casbin enforcer wrapper |
| `internal/authz/cache.go` | Authorization decision cache |
| `internal/authz/model.conf` | Casbin RBAC model (embedded) |
| `internal/authz/policy.csv` | Default authorization policies (embedded) |
| `configs/authz/model.conf` | Casbin model for file-based config |
| `configs/authz/policy.csv` | Policies for file-based config |

### Modified Files

| File | Changes |
|------|---------|
| `go.mod` | Add zitadel/oidc/v3, casbin/v2, badger/v4, prometheus dependencies |
| `internal/auth/middleware.go` | Integrate with AuthSubject and new auth modes |
| `internal/auth/flow_handlers.go` | Added auditLogger field for security event logging |
| `internal/api/router_core.go` | BadgerDB state store initialization, audit logger setup |

---

## Verification

### Test Coverage

| Component | Coverage | Test File |
|-----------|----------|-----------|
| AuthSubject | 100% | `subject_test.go` |
| AuthMode parsing | 100% | `subject_test.go` |
| OIDCConfig validation | 100% | `oidc_config_test.go` |
| Casbin enforcer | 100% | `internal/authz/enforcer_test.go` |
| RBAC policies | 100% | `internal/authz/enforcer_test.go` |
| Zitadel OIDC flow | 100% | `zitadel_flow_test.go` |
| BadgerDB state store | 100% | `zitadel_state_store_badger_test.go` |
| OIDC metrics | 100% | `metrics_test.go` |
| Audit logger | 100% | `audit_logger_test.go` |
| Flow handlers | 90%+ | `flow_handlers_test.go` |
| Back-channel logout | 100% | `flow_handlers_oidc_test.go` |

### Verified Claims

| Claim | Source | Verified |
|-------|--------|----------|
| Zitadel OIDC v3.45.1 | `go.mod` | Yes |
| OpenID Foundation certified | [OIDC Certification](https://openid.net/certification/) | Yes |
| Casbin v2.135.0 | `go.mod` | Yes |
| BadgerDB v4.9.0 | `go.mod` | Yes |
| RBAC model works | `internal/authz/enforcer_test.go` | Yes |
| AuthSubject ↔ Claims conversion | `internal/auth/subject_test.go` | Yes |
| PKCE (RFC 7636) | `zitadel_rp_test.go` | Yes |
| Nonce validation | `zitadel_flow_test.go` | Yes |
| Back-channel logout | `flow_handlers_oidc_test.go` | Yes |
| Prometheus metrics | `metrics_test.go` | Yes |
| Audit logging | `audit_logger_test.go` | Yes |

---

## Migration Guide

### From AUTH_MODE=basic to AUTH_MODE=oidc

1. Configure OIDC provider (create OAuth2 client)
2. Set environment variables:
   ```bash
   export AUTH_MODE=multi  # Keep basic as fallback
   export OIDC_ENABLED=true
   export OIDC_ISSUER_URL=https://your-idp
   export OIDC_CLIENT_ID=your-client
   export OIDC_CLIENT_SECRET=your-secret
   export OIDC_REDIRECT_URL=https://cartographus.local/auth/callback
   ```
3. Test authentication via `/auth/login`
4. Verify user info at `/auth/userinfo`
5. Optionally disable fallback: `AUTH_MODE=oidc`

---

## Related ADRs

- [ADR-0003](0003-authentication-architecture.md): Multi-mode authentication (foundation)
- [ADR-0009](0009-plex-direct-integration.md): Plex direct integration
- [ADR-0012](0012-configuration-management-koanf.md): Configuration management

---

## References

- [Zitadel OIDC Library](https://github.com/zitadel/oidc) - OpenID Foundation certified Go library
- [OpenID Foundation Certification](https://openid.net/certification/) - Certified implementations list
- [Casbin Documentation](https://casbin.org/docs/overview) - RBAC policy engine
- [OpenID Connect Core 1.0](https://openid.net/specs/openid-connect-core-1_0.html)
- [OpenID Connect Back-Channel Logout 1.0](https://openid.net/specs/openid-connect-backchannel-1_0.html)
- [RFC 7636 - PKCE](https://tools.ietf.org/html/rfc7636) - Proof Key for Code Exchange
- [RFC 6749 - OAuth 2.0](https://tools.ietf.org/html/rfc6749)
- [OWASP Authentication Guidelines](https://cheatsheetseries.owasp.org/cheatsheets/Authentication_Cheat_Sheet.html)
- [BadgerDB Documentation](https://dgraph.io/docs/badger/) - ACID-compliant key-value store
