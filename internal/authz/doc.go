// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package authz provides authorization functionality using Casbin.
//
// This package implements Role-Based Access Control (RBAC) for the Cartographus
// application, enforcing fine-grained access policies on API endpoints using
// the Casbin authorization library. It supports hierarchical roles, path-based
// permissions, policy caching, and automatic policy reload.
//
// # Architecture
//
// The authorization system follows ADR-0015 (Zero Trust Authentication & Authorization):
//
//	Request -> Auth Middleware -> Authz Middleware -> Handler
//	               |                    |
//	          Authenticate         Authorize (Casbin)
//	           (internal/auth)      (this package)
//
// # RBAC Model
//
// The package uses Casbin's ACL model with role inheritance:
//
//	[request_definition]
//	r = sub, obj, act
//
//	[policy_definition]
//	p = sub, obj, act
//
//	[role_definition]
//	g = _, _
//
//	[policy_effect]
//	e = some(where (p.eft == allow))
//
//	[matchers]
//	m = g(r.sub, p.sub) && keyMatch(r.obj, p.obj) && r.act == p.act
//
// # Policy Definition
//
// Policies are defined in CSV format:
//
//	# Role permissions
//	p, admin, /api/v1/*, read
//	p, admin, /api/v1/*, write
//	p, admin, /api/v1/*, delete
//	p, viewer, /api/v1/analytics/*, read
//	p, viewer, /api/v1/playbacks, read
//
//	# Role assignments
//	g, alice, admin
//	g, bob, viewer
//
// # Usage Example
//
// Creating an enforcer:
//
//	cfg := authz.DefaultEnforcerConfig()
//	enforcer, err := authz.NewEnforcer(ctx, cfg)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer enforcer.Close()
//
//	// Check permission
//	allowed, err := enforcer.Enforce("alice", "/api/v1/users", "write")
//	if err != nil {
//	    log.Printf("Authorization check failed: %v", err)
//	}
//
// Using middleware:
//
//	middleware := authz.NewMiddleware(enforcer)
//
//	// Protect a specific endpoint
//	http.HandleFunc("/api/v1/admin",
//	    middleware.Authorize("admin_area", "write", adminHandler))
//
//	// Dynamic authorization based on request path
//	http.HandleFunc("/api/v1/",
//	    middleware.AuthorizeRequest(apiHandler))
//
// Role management:
//
//	// Add role to user
//	_, err := enforcer.AddRoleForUser("alice", "admin")
//
//	// Remove role from user
//	_, err := enforcer.DeleteRoleForUser("alice", "admin")
//
//	// Get user roles
//	roles, err := enforcer.GetRolesForUser("alice")
//
// # Configuration Options
//
// The EnforcerConfig supports:
//
//	cfg := &authz.EnforcerConfig{
//	    ModelPath:      "",              // Path to model file (empty = embedded)
//	    PolicyPath:     "",              // Path to policy file (empty = embedded)
//	    AutoReload:     true,            // Enable hot policy reload
//	    ReloadInterval: 30 * time.Second, // Policy check interval
//	    DefaultRole:    "viewer",        // Role for unauthenticated users
//	    CacheEnabled:   true,            // Enable decision caching
//	    CacheTTL:       5 * time.Minute, // Cache TTL
//	}
//
// # Embedded Policies
//
// The package embeds default model and policy files for zero-configuration setup:
//   - model.conf: RBAC model with role hierarchy
//   - policy.csv: Default policies for common roles
//
// # Caching
//
// The enforcer includes an enforcement decision cache to improve performance:
//   - Cache key: (subject, object, action) tuple
//   - Automatic invalidation on policy/role changes
//   - Configurable TTL with periodic cleanup
//
// # HTTP Method Mapping
//
// The AuthorizeRequest middleware maps HTTP methods to actions:
//   - GET, HEAD, OPTIONS -> "read"
//   - POST, PUT, PATCH -> "write"
//   - DELETE -> "delete"
//
// # Thread Safety
//
// All components are safe for concurrent use:
//   - Casbin SyncedEnforcer provides built-in synchronization
//   - Cache uses sync.RWMutex for concurrent access
//   - Policy auto-reload runs in a separate goroutine
//
// # Performance
//
//   - Enforcement check: <100us (with cache hit)
//   - Cache miss: ~1ms (Casbin evaluation)
//   - Policy reload: ~10ms for typical policy files
//
// # See Also
//
//   - internal/auth: Authentication (runs before authorization)
//   - internal/audit: Audit logging for authorization decisions
//   - github.com/casbin/casbin/v2: Underlying authorization library
//   - docs/adr/0015-zero-trust-authentication-authorization.md: ADR
package authz
