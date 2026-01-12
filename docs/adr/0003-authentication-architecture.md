# ADR-0003: Multi-Mode Authentication Architecture

**Date**: 2025-11-16
**Status**: Accepted

---

## Context

Cartographus needs to support diverse deployment scenarios:

1. **Home Users**: Single-user deployments behind VPN or local network
2. **Shared Servers**: Multi-user Plex servers with access control
3. **Public Dashboards**: Read-only public access for community servers
4. **Development**: Quick iteration without authentication overhead
5. **Enterprise**: OIDC integration with identity providers

### Requirements

- Optional authentication (not all users need it)
- Multiple authentication methods for different use cases
- JWT for API access with configurable expiration
- Rate limiting for public-facing deployments
- CORS configuration for cross-origin API access
- OIDC/OAuth 2.0 for enterprise identity providers (ADR-0015)
- Plex OAuth for media server authentication (ADR-0015)

### Alternatives Considered

| Approach | Pros | Cons |
|----------|------|------|
| **JWT Only** | Stateless, scalable | Requires token management |
| **Basic Auth Only** | Simple, browser-native | Less secure for APIs |
| **OAuth2** | Industry standard | Overkill for self-hosted |
| **Multi-Mode** | Flexible for all use cases | More complex configuration |

---

## Decision

Implement a **multi-mode authentication architecture** with six configurable modes:

| Mode | Environment Variable | Use Case |
|------|---------------------|----------|
| `none` | `AUTH_MODE=none` | Home use, local network, development |
| `basic` | `AUTH_MODE=basic` | Simple username/password protection |
| `jwt` | `AUTH_MODE=jwt` | Full API authentication with tokens (default) |
| `oidc` | `AUTH_MODE=oidc` | OpenID Connect with enterprise IdPs |
| `plex` | `AUTH_MODE=plex` | Plex OAuth 2.0 for media server users |
| `multi` | `AUTH_MODE=multi` | Try multiple methods (OIDC -> Plex -> JWT -> Basic) |

### Key Factors

1. **Deployment Flexibility**: Different users have different security needs
2. **Progressive Enhancement**: Start simple, add security as needed
3. **API Compatibility**: JWT for programmatic access, Basic for browser access
4. **Defense in Depth**: Rate limiting and CORS regardless of auth mode
5. **Enterprise Ready**: OIDC support for corporate identity providers

---

## Consequences

### Positive

- **Zero-Configuration Start**: `AUTH_MODE=none` for immediate development
- **Browser-Native Login**: Basic auth works in all browsers
- **Stateless API Access**: JWT tokens for external integrations
- **Configurable Security**: Match authentication to deployment context
- **Enterprise Integration**: OIDC/Plex for advanced deployments

### Negative

- **Configuration Complexity**: Six modes to document and support
- **Mode-Specific Bugs**: Each mode path needs testing
- **Security Responsibility**: Users must choose appropriate mode

### Neutral

- **Session Storage**: BadgerDB or memory-based sessions for OIDC/Plex
- **Password Hashing**: bcrypt (cost 12) for Basic auth passwords

---

## Implementation

### Authentication Flow

```
                    ┌─────────────────┐
                    │   HTTP Request  │
                    └────────┬────────┘
                             │
                    ┌────────▼────────┐
                    │  Rate Limiter   │
                    │ (all requests)  │
                    └────────┬────────┘
                             │
                    ┌────────▼────────┐
                    │   CORS Check    │
                    │ (all requests)  │
                    └────────┬────────┘
                             │
                    ┌────────▼────────┐
                    │  Auth Mode?     │
                    └────────┬────────┘
                             │
    ┌──────────┬─────────────┼─────────────┬──────────┐
    ▼          ▼             ▼             ▼          ▼
  none      basic          jwt          oidc       plex
    │          │             │             │          │
  Pass    Validate      Validate      Validate   Validate
through   Header         Token       OIDC Token  Plex Token
    │   (Basic b64)   (Bearer JWT)   (Session)   (Session)
    ▼          ▼             ▼             ▼          ▼
                    ┌────────────────────┐
                    │      Handler       │
                    └────────────────────┘
```

### JWT Configuration

```go
// internal/auth/jwt.go

// Claims represents JWT claims
type Claims struct {
    Username string `json:"username"`
    Role     string `json:"role"`  // "admin" or "viewer"
    jwt.RegisteredClaims
}

// JWTManager handles JWT token creation and validation
type JWTManager struct {
    secret  []byte
    timeout time.Duration
}

// NewJWTManager creates a new JWT token manager
func NewJWTManager(cfg *config.SecurityConfig) (*JWTManager, error) {
    secret := cfg.JWTSecret
    if secret == "" {
        return nil, fmt.Errorf("JWT_SECRET is required but was empty")
    }
    return &JWTManager{
        secret:  []byte(secret),
        timeout: cfg.SessionTimeout,
    }, nil
}

func (m *JWTManager) GenerateToken(username, role string) (string, error) {
    claims := &Claims{
        Username: username,
        Role:     role,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(m.timeout)),
            IssuedAt:  jwt.NewNumericDate(time.Now()),
            NotBefore: jwt.NewNumericDate(time.Now()),
        },
    }
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString(m.secret)
}
```

### Basic Auth Configuration

```go
// internal/auth/basic.go

// BasicAuthManager handles HTTP Basic Authentication with secure password verification
type BasicAuthManager struct {
    username     string
    passwordHash []byte // bcrypt hash of password
}

// NewBasicAuthManager creates a new Basic Auth manager with bcrypt-hashed password
// The password is hashed at initialization to avoid hashing on every request
// Note: Basic auth uses 8-char minimum; JWT mode uses 12-char NIST policy via config validation
func NewBasicAuthManager(username, password string) (*BasicAuthManager, error) {
    if username == "" {
        return nil, fmt.Errorf("username is required")
    }
    if password == "" {
        return nil, fmt.Errorf("password is required")
    }
    if len(password) < 8 {
        return nil, fmt.Errorf("password must be at least 8 characters for security")
    }
    // Hash the password using bcrypt (cost factor 12)
    hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
    if err != nil {
        return nil, fmt.Errorf("failed to hash password: %w", err)
    }
    return &BasicAuthManager{
        username:     username,
        passwordHash: hash,
    }, nil
}

// validateUsernamePassword performs constant-time comparison of credentials
func (m *BasicAuthManager) validateUsernamePassword(username, password string) bool {
    usernameMatch := subtle.ConstantTimeCompare([]byte(username), []byte(m.username)) == 1
    passwordMatch := bcrypt.CompareHashAndPassword(m.passwordHash, []byte(password)) == nil
    return usernameMatch && passwordMatch
}
```

### Middleware Chain

```go
// internal/auth/middleware.go

// Middleware provides authentication and rate limiting middleware
type Middleware struct {
    jwtManager             *JWTManager
    basicAuthManager       *BasicAuthManager
    authMode               string
    rateLimiter            *RateLimiter
    rateLimitDisabled      bool
    corsOrigins            []string
    trustedProxies         map[string]bool
    basicAuthDefaultRole   string // Default role for Basic Auth users (default: viewer)
    basicAuthAdminUsername string // Username that gets admin role
}

// Authenticate is middleware that enforces authentication
func (m *Middleware) Authenticate(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        if m.authMode == "none" {
            next(w, r)
            return
        }

        authHeader := r.Header.Get("Authorization")

        // Handle Basic Authentication
        if m.authMode == string(AuthModeBasic) {
            m.handleBasicAuth(w, r, next, authHeader)
            return
        }

        // Handle JWT Authentication
        m.handleJWTAuth(w, r, next, authHeader)
    }
}

// RateLimit is middleware that enforces rate limiting
func (m *Middleware) RateLimit(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        if m.rateLimitDisabled {
            next(w, r)
            return
        }
        ip := m.getClientIP(r)
        if !m.rateLimiter.Allow(ip) {
            http.Error(w, "Too many requests", http.StatusTooManyRequests)
            return
        }
        next(w, r)
    }
}
```

### Environment Variables

#### Core Authentication

| Variable | Default | Description |
|----------|---------|-------------|
| `AUTH_MODE` | `jwt` | Authentication mode: `none`, `basic`, `jwt`, `oidc`, `plex`, `multi` |
| `JWT_SECRET` | (required for jwt) | 256-bit secret for JWT signing (min 32 chars) |
| `SESSION_TIMEOUT` | `24h` | Token/session expiration time |
| `ADMIN_USERNAME` | (required for basic) | Basic auth username |
| `ADMIN_PASSWORD` | (required for basic) | Basic auth password (plaintext, hashed at init) |
| `BASIC_AUTH_DEFAULT_ROLE` | `viewer` | Default role for basic auth users |

#### Rate Limiting and Security

| Variable | Default | Description |
|----------|---------|-------------|
| `RATE_LIMIT_REQUESTS` | `100` | Requests per window |
| `RATE_LIMIT_WINDOW` | `1m` | Rate limit time window |
| `DISABLE_RATE_LIMIT` | `false` | Disable rate limiting |
| `CORS_ORIGINS` | `*` | Allowed CORS origins (comma-separated) |
| `TRUSTED_PROXIES` | (none) | Trusted proxy IPs for X-Forwarded-For |

#### OIDC Configuration (AUTH_MODE=oidc)

| Variable | Default | Description |
|----------|---------|-------------|
| `OIDC_ISSUER_URL` | (required) | OIDC provider issuer URL |
| `OIDC_CLIENT_ID` | (required) | OAuth 2.0 client ID |
| `OIDC_CLIENT_SECRET` | (optional) | OAuth 2.0 client secret |
| `OIDC_REDIRECT_URL` | (required) | OAuth callback URL |
| `OIDC_SCOPES` | `openid,profile,email` | OAuth scopes |
| `OIDC_PKCE_ENABLED` | `true` | Enable PKCE for public clients |

#### Plex Configuration (AUTH_MODE=plex)

| Variable | Default | Description |
|----------|---------|-------------|
| `PLEX_AUTH_CLIENT_ID` | (required) | Plex app client ID |
| `PLEX_AUTH_REDIRECT_URI` | (required) | Plex OAuth callback URL |
| `PLEX_AUTH_DEFAULT_ROLES` | `viewer` | Default roles for Plex users |
| `PLEX_AUTH_SERVER_OWNER_ROLE` | `admin` | Role for server owners |

### Code References

| Component | File | Notes |
|-----------|------|-------|
| JWT Manager | `internal/auth/jwt.go` | Token generation and validation |
| Basic Auth Manager | `internal/auth/basic.go` | bcrypt password validation |
| Middleware | `internal/auth/middleware.go` | Request authentication, rate limiting, CORS |
| Auth Modes | `internal/auth/subject.go` | AuthMode constants, AuthSubject struct |
| OIDC Authenticator | `internal/auth/oidc_authenticator.go` | OpenID Connect validation |
| Plex Authenticator | `internal/auth/plex_authenticator.go` | Plex OAuth validation |
| Configuration | `internal/config/config.go` | Security settings (SecurityConfig) |

---

## Verification

### Verified Claims

| Claim | Source | Verified |
|-------|--------|----------|
| JWT v5.3.0 | `go.mod:16` | Yes |
| bcrypt for passwords | `internal/auth/basic.go` (cost 12) | Yes |
| Six auth modes | `internal/auth/subject.go` | Yes |
| Rate limiting | `internal/auth/middleware.go` (RateLimiter struct) | Yes |
| CORS support | `internal/auth/middleware.go` | Yes |
| OIDC support | `internal/auth/oidc_authenticator.go` | Yes |
| Plex OAuth support | `internal/auth/plex_authenticator.go` | Yes |

### Test Coverage

- Auth tests: `internal/auth/*_test.go` (39 test files)
- Coverage target: 100% for authentication package
- Security tests: Rate limiting, token validation, password hashing, OIDC, Plex

---

## Related ADRs

- [ADR-0012](0012-configuration-management-koanf.md): Environment variable handling
- [ADR-0013](0013-request-validation.md): Request validation integration
- [ADR-0015](0015-zero-trust-authentication-authorization.md): Zero Trust auth (OIDC, Plex, Casbin)

---

## References

- [JWT Introduction](https://jwt.io/introduction)
- [golang-jwt/jwt](https://github.com/golang-jwt/jwt)
- [bcrypt Password Hashing](https://pkg.go.dev/golang.org/x/crypto/bcrypt)
- [OWASP Authentication Guidelines](https://cheatsheetseries.owasp.org/cheatsheets/Authentication_Cheat_Sheet.html)
