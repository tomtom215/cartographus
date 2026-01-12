# ADR-0016: Chi Router Adoption

**Date**: 2025-12-08
**Status**: Accepted (Implemented - Migration Complete)

---

## Context

Cartographus previously used Go's standard library `http.ServeMux` (Go 1.24) for HTTP routing. The implementation exhibited significant boilerplate due to nested middleware wrapping.

### Pre-Migration State (Historical)

| Metric | Value | Source |
|--------|-------|--------|
| Total endpoints | 167 | `router.go` (142), `zerotrust_routes.go` (20), `router_import.go` (5) |
| Middleware wrapper calls | 285+ | Nested CORS/RateLimit/Authenticate patterns |
| Custom middleware | 4 | Compression, RequestID, Prometheus, Performance |
| Auth middleware | 4 | CORS, RateLimit, Authenticate, SecurityHeaders |

### Post-Migration State (Current)

| Metric | Value | Source |
|--------|-------|--------|
| Total endpoints | 302 | `internal/api/chi_router.go` |
| Route groups | 25+ | Domain-organized with group-level middleware |
| Global middleware | 5 | RequestID, E2EDebug, RealIP, Recoverer, CORS |
| E2E test suites | 75 | `tests/e2e/*.spec.ts` |

### Pre-Migration Middleware Pattern (Historical)

```go
// internal/api/router.go (no longer exists) - Each endpoint required 3-6 nested wrappers
mux.HandleFunc("/api/v1/analytics/trends",
    router.middleware.CORS(
        router.middleware.RateLimit(
            middleware.PrometheusMetrics(
                middleware.Compression(
                    middleware.RequestID(
                        router.handler.AnalyticsTrends,
                    ),
                ),
            ),
        ),
    ),
)
```

This pattern had issues:
1. Repeated for 167 endpoints
2. Made middleware stack difficult to modify globally
3. Created visual noise that obscured route definitions
4. Required careful nesting order to maintain correctness

### Requirements

1. **Reduce Boilerplate**: Eliminate repetitive middleware wrapping
2. **Route Grouping**: Organize endpoints by domain (analytics, tautulli, plex, etc.)
3. **Maintainability**: Simplify adding new endpoints and modifying middleware
4. **Compatibility**: Preserve existing handlers and custom middleware
5. **Production Ready**: Battle-tested router used at scale

### Alternatives Considered

| Router | Pros | Cons |
|--------|------|------|
| **http.ServeMux (current)** | Standard library, no dependencies | Verbose middleware, no grouping |
| **Chi v5** | Route groups, built-in middleware, net/http compatible | Additional dependency |
| **Gin** | Fast, popular | Different handler signature, heavier |
| **gorilla/mux** | Feature-rich | Archived/maintenance mode |
| **Echo** | Fast, middleware-first | Different handler signature |

---

## Decision

Adopt **Chi v5.2.3** as the HTTP router to replace `http.ServeMux`.

### Key Factors

1. **100% net/http Compatible**: Existing handlers require zero changes
   - `func(w http.ResponseWriter, r *http.Request)` signature preserved
   - Custom middleware continues to work unmodified

2. **Route Grouping with Middleware**: Apply middleware once per group
   ```go
   r.Route("/api/v1/analytics", func(r chi.Router) {
       r.Use(middleware.CORS, middleware.RateLimit, middleware.PrometheusMetrics)
       r.Get("/trends", handler.AnalyticsTrends)
       r.Get("/geographic", handler.AnalyticsGeographic)
       // 20+ endpoints with middleware applied automatically
   })
   ```

3. **Production Proven**: Used at Cloudflare, Heroku, 99Designs, Pressly

4. **Built-in Middleware**: Replace custom implementations with battle-tested ones
   - `chi/middleware.Compress` - gzip compression
   - `chi/middleware.RequestID` - request tracking (extended with `RequestIDWithLogging`)
   - `chi/middleware.Logger` - structured logging
   - `chi/middleware.Recoverer` - panic recovery
   - `chi/middleware.RealIP` - proxy IP extraction
   - `go-chi/httprate` - rate limiting (separate package, not built-in)

5. **Zero External Dependencies**: Chi depends only on standard library

6. **Patricia Radix Trie**: Efficient routing performance (~1000 LOC)

### Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                              Chi Router                                  │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                    Global Middleware Stack                       │   │
│  │  RequestIDWithLogging → E2EDebugLogging → RealIP → Recoverer   │   │
│  │                            → CORS                                │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                    │                                     │
│        ┌───────────────────────────┼───────────────────────────┐        │
│        │                           │                           │        │
│        ▼                           ▼                           ▼        │
│  ┌──────────────┐          ┌──────────────┐          ┌──────────────┐  │
│  │  /api/v1/    │          │  /api/auth/  │          │  /api/admin/ │  │
│  │  analytics   │          │              │          │              │  │
│  │  ─────────── │          │  ─────────── │          │  ─────────── │  │
│  │  + RateLimit │          │  + RateLimit │          │  + RateLimit │  │
│  │  + Metrics   │          │  + Session   │          │  + AdminAuth │  │
│  │  + Auth      │          │              │          │              │  │
│  └──────────────┘          └──────────────┘          └──────────────┘  │
│        │                           │                           │        │
│   43 endpoints                14 endpoints               12 endpoints   │
└─────────────────────────────────────────────────────────────────────────┘
```

### Route Group Organization

| Group | Path Prefix | Middleware | Endpoints |
|-------|-------------|------------|-----------|
| Health | `/api/v1/health` | RateLimit(1000/min), SecurityHeaders | 6 |
| Auth | `/api/v1/auth` | RateLimit(5/min), Login(5/5min) | 4 |
| Core API | `/api/v1` | RateLimit, Metrics, Auth | 7 |
| Analytics | `/api/v1/analytics` | RateLimit(1000/min), Metrics, Auth | 43 |
| Spatial | `/api/v1/spatial` | RateLimit, Metrics, Auth | 5 |
| Tautulli Proxy | `/api/v1/tautulli` | RateLimit, Auth | 55 |
| Plex Direct | `/api/v1/plex` | RateLimit, Auth (except webhook) | 29 |
| Export | `/api/v1/export` | RateLimit(10/min), Auth | 4 |
| Stream | `/api/v1/stream` | RateLimit(10/min), Auth | 1 |
| Backup | `/api/v1/backup` | RateLimit, Auth | 11 |
| Backups | `/api/v1/backups` | RateLimit, Auth | 7 |
| Search | `/api/v1/search` | RateLimit, Metrics, Auth | 2 |
| Content | `/api/v1/content` | RateLimit, Metrics, Auth | 5 |
| Users | `/api/v1/users` | RateLimit, Metrics, Auth | 5 |
| Newsletter | `/api/v1/newsletter/*` | RateLimit, Metrics, Auth | 15 |
| Wrapped | `/api/v1/wrapped` | RateLimit(1000/min), Metrics, Auth | 5 |
| PAT Tokens | `/api/v1/user/tokens` | RateLimit, Metrics, Auth | 6 |
| Detection | `/api/v1/detection` | RateLimit, Metrics, Auth | 11 |
| Audit | `/api/v1/audit` | RateLimit, Metrics, Auth | 6 |
| DLQ | `/api/v1/dlq` | RateLimit, Metrics, Auth | 8 |
| WAL | `/api/v1/wal` | RateLimit, Metrics, Auth | 3 |
| Replay | `/api/v1/admin/replay` | RateLimit, Metrics, Auth | 6 |
| Recommendations | `/api/v1/recommendations` | RateLimit, Metrics, Auth | 11 |
| Dedupe | `/api/v1/dedupe` | RateLimit, Metrics, Auth | 6 |
| Sync | `/api/v1/sync` | RateLimit, Metrics, Auth | 3 |
| Zero Trust | `/api/auth/*` | RateLimit, Session | 14 |
| Admin Roles | `/api/admin/roles` | RateLimit, AdminAuth | 4 |
| Admin Servers | `/api/v1/admin/servers` | RateLimit, Metrics, Auth | 8 |
| Tiles | `/api/v1/tiles` | None (cached) | 1 |
| Observability | `/metrics`, `/swagger` | None | 2 |
| Static/SPA | `/*` | SecurityHeaders (CSP nonce) | 1 |

---

## Consequences

### Positive

- **80% Reduction in Middleware Boilerplate**: From 285+ wrapper calls to ~15 `Use()` calls
- **Improved Readability**: Route definitions clearly visible without nesting
- **Easier Maintenance**: Add/remove middleware from groups in one place
- **Built-in Middleware Options**: Can optionally replace custom implementations
- **Incremental Migration**: Chi and ServeMux can coexist during transition
- **Better IDE Support**: Route structure visible in code navigation
- **Context-Based URL Parameters**: `chi.URLParam(r, "key")` vs manual parsing

### Negative

- **New Dependencies**: Chi ecosystem packages added to go.mod:
  - `github.com/go-chi/chi/v5 v5.2.3` - Core router
  - `github.com/go-chi/cors v1.2.2` - Production-hardened CORS middleware
  - `github.com/go-chi/httprate v0.15.0` - Battle-tested rate limiting
- **Migration Effort**: 167 endpoints refactored (now 302 endpoints with new features)
- **Learning Curve**: Team must learn Chi-specific patterns
- **Test Updates**: Router tests updated to use Chi

### Neutral

- **Performance**: Similar to ServeMux for this endpoint count (trie vs map lookup negligible)
- **Handler Signatures**: Unchanged (both use `http.HandlerFunc`)
- **Custom Middleware**: Continues to work without modification

---

## Implementation

### Phase 1: Add Chi Dependency

```bash
go get github.com/go-chi/chi/v5@v5.2.3
```

### Phase 2: Create Chi Router (Actual Implementation)

```go
// internal/api/chi_router.go
package api

import (
    "net/http"
    "github.com/go-chi/chi/v5"
    chimiddleware "github.com/go-chi/chi/v5/middleware"
)

// SetupChi configures all HTTP routes using Chi router.
// This replaces the http.ServeMux-based Setup() method.
func (router *Router) SetupChi() http.Handler {
    r := chi.NewRouter()

    // Global middleware (applied to all routes)
    r.Use(RequestIDWithLogging())      // Add X-Request-ID header with logging context
    r.Use(E2EDebugLogging())           // E2E diagnostic logging (enabled via E2E_DEBUG=true)
    r.Use(chimiddleware.RealIP)        // Extract real IP from X-Forwarded-For
    r.Use(chimiddleware.Recoverer)     // Recover from panics
    r.Use(router.chiMiddleware.CORS()) // CORS must be global to handle OPTIONS preflight

    // Health endpoints (permissive rate limiting - 1000/min)
    r.Route("/api/v1/health", func(r chi.Router) {
        r.Use(router.chiMiddleware.RateLimitHealth())
        r.Use(APISecurityHeaders())
        r.Get("/live", router.handler.HealthLive)
        r.Get("/ready", router.handler.HealthReady)
        r.Get("/", router.handler.Health)
        r.Get("/setup", router.handler.SetupStatus)
        r.Get("/nats", router.handler.HealthNATS)
        r.Get("/nats/component", router.handler.HealthNATSComponent)
    })

    // Analytics endpoints (1000/min - cached read operations)
    r.Route("/api/v1/analytics", func(r chi.Router) {
        r.Use(router.chiMiddleware.RateLimitAnalytics())
        r.Use(APISecurityHeaders())
        r.Use(chiMiddleware(middleware.PrometheusMetrics))
        r.Use(chiMiddleware(router.middleware.Authenticate))

        r.Get("/trends", router.handler.AnalyticsTrends)
        r.Get("/geographic", router.handler.AnalyticsGeographic)
        // ... 43 total analytics endpoints
    })

    // Plex endpoints with authentication (except webhook)
    r.Route("/api/v1/plex", func(r chi.Router) {
        r.Use(router.chiMiddleware.RateLimit())
        r.Use(chiPathValue) // Bridge Chi URL params to r.PathValue()

        // Webhook uses its own HMAC-SHA256 verification
        r.Post("/webhook", router.handler.PlexWebhook)

        // All other Plex endpoints require authentication
        r.Group(func(r chi.Router) {
            r.Use(chiMiddleware(router.middleware.Authenticate))
            r.Get("/sessions", router.handler.PlexSessions)
            r.Get("/library/sections/{key}/all", router.handler.PlexLibrarySectionContent)
            // ... 28 more authenticated endpoints
        })
    })

    // ... additional route groups

    // Observability (no rate limiting)
    r.Handle("/metrics", promhttp.Handler())
    r.Get("/swagger/*", httpSwagger.Handler(...))

    // Static files and SPA fallback (with CSP nonce)
    r.Group(func(r chi.Router) {
        r.Use(chiMiddleware(router.middleware.SecurityHeaders))
        r.Get("/*", router.serveStaticOrIndex)
    })

    return r
}
```

### Phase 3: URL Parameter Migration

```go
// Before (manual path parsing in zerotrust_routes.go:223-238)
func (r *ZeroTrustRouter) handleRevokeSession(w http.ResponseWriter, req *http.Request) {
    path := req.URL.Path
    prefix := "/api/auth/sessions/"
    sessionID := strings.TrimPrefix(path, prefix)
    // ...
}

// After (Chi URL parameters)
func (r *ZeroTrustRouter) handleRevokeSession(w http.ResponseWriter, req *http.Request) {
    sessionID := chi.URLParam(req, "id")
    // ...
}
```

### Phase 4: Middleware Adapter (Optional)

If Chi's built-in middleware is preferred over custom implementations:

```go
// Replace custom with Chi built-in (optional, can be done incrementally)
// internal/middleware/compression.go → chi/middleware.Compress
// internal/middleware/requestid.go → chi/middleware.RequestID
```

### Migration Strategy (Completed)

| Phase | Scope | Status |
|-------|-------|--------|
| 1. Add dependency | `go.mod` | Complete |
| 2. Create `chi_router.go` | New file with all routes | Complete |
| 3. Migrate route groups | Domain-organized groups | Complete |
| 4. Update path params | `chi.URLParam()` and `chiPathValue` bridge | Complete |
| 5. Update tests | Router initialization | Complete |
| 6. Remove old router | `router.go` deprecated, `zerotrust_routes.go` kept for tests | Complete |

### Configuration

No new environment variables required. Chi uses the same middleware configuration.

### Code References (Current)

| Component | File | Notes |
|-----------|------|-------|
| Chi router setup | `internal/api/chi_router.go` | Main router with 302 endpoints |
| Chi middleware | `internal/api/chi_middleware.go` | Production-hardened CORS, rate limiting |
| Router core | `internal/api/router_core.go` | Router struct and initialization |
| Zero Trust routes | `internal/api/zerotrust_routes.go` | Deprecated (test-only), Chi routes in chi_router.go |
| URL parameters | Handler files | `chi.URLParam(r, "key")` or `chiPathValue` bridge |
| Auth middleware | `internal/auth/middleware.go` | Unchanged |
| Custom middleware | `internal/middleware/*.go` | Unchanged |

---

## Verification

### Current State Verification

| Claim | Source | Verified |
|-------|--------|----------|
| 302 endpoints | `grep -c "r\.\(Get\|Post\|Put\|Delete\|Patch\)" internal/api/chi_router.go` | Yes |
| Chi v5.2.3 | `go.mod:13` | Yes |
| go-chi/cors v1.2.2 | `go.mod:43` | Yes |
| go-chi/httprate v0.15.0 | `go.mod:44` | Yes |
| 75 E2E test suites | `ls tests/e2e/*.spec.ts \| wc -l` | Yes |
| Go 1.24.0 | `go.mod:3` | Yes |

### Post-Migration Verification (Completed)

| Criterion | Status |
|-----------|--------|
| All 302 endpoints accessible | E2E test suite passes |
| Middleware applied correctly | Integration tests pass |
| Path parameters work | `chi.URLParam()` and `chiPathValue` bridge verified |
| Swagger works | `/swagger/` accessible |
| Prometheus metrics work | `/metrics` returns data |

### Test Coverage

- Existing handler tests: Unchanged (same signatures)
- Router tests: Updated to use Chi
- Integration tests: All routes verified
- E2E tests: 75 Playwright test suites validate functionality

---

## Rollback Plan

Migration is complete and validated. If rollback is needed:

1. Chi and ServeMux can coexist (both implement `http.Handler`)
2. `zerotrust_routes.go` preserved for test compatibility
3. Handlers are unchanged, so rollback would be router-only
4. All 75 E2E test suites validate current Chi implementation

---

## Related ADRs

- [ADR-0003](0003-authentication-architecture.md): Auth middleware preserved
- [ADR-0004](0004-process-supervision-with-suture.md): HTTP server service unchanged
- [ADR-0015](0015-zero-trust-authentication-authorization.md): Zero Trust routes migrated

---

## References

- [Chi v5 Documentation](https://pkg.go.dev/github.com/go-chi/chi/v5)
- [Chi GitHub Repository](https://github.com/go-chi/chi)
- [Chi Middleware](https://pkg.go.dev/github.com/go-chi/chi/v5/middleware)
- [Go ServeMux vs Chi](https://www.calhoun.io/go-servemux-vs-chi/)
- [HTTP Routing Benchmarks](https://github.com/pkieltyka/go-http-routing-benchmark)
