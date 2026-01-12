# Authentication Guide

Configure authentication for single-user or multi-user deployments.

**[Home](Home)** | **[Configuration](Configuration)** | **Authentication** | **[Reverse Proxy](Reverse-Proxy)**

---

## Authentication Modes

Cartographus supports multiple authentication modes:

| Mode | Best For | Configuration |
|------|----------|---------------|
| **[JWT](#jwt-authentication)** | Most deployments | `AUTH_MODE=jwt` |
| **[OIDC](#oidc-authentication)** | Enterprise SSO | `AUTH_MODE=oidc` |
| **[Plex](#plex-authentication)** | Plex communities | `AUTH_MODE=plex` |
| **[Multi](#multi-authentication)** | Multiple auth methods | `AUTH_MODE=multi` |
| **[Basic](#basic-authentication)** | Simple setups | `AUTH_MODE=basic` |
| **None** | Development only | `AUTH_MODE=none` |

---

## JWT Authentication

**Default mode.** Simple username/password with JWT tokens.

### Configuration

```yaml
environment:
  - AUTH_MODE=jwt
  - JWT_SECRET=your-secure-secret-at-least-32-characters-long
  - ADMIN_USERNAME=admin
  - ADMIN_PASSWORD=YourSecurePassword123!
  - SESSION_TIMEOUT=24h
```

### Generating a Secure Secret

```bash
openssl rand -base64 48
```

### How It Works

1. User logs in with username/password
2. Server issues a JWT token
3. Token stored in HTTP-only cookie
4. Token valid for `SESSION_TIMEOUT` duration

---

## OIDC Authentication

Integrate with enterprise identity providers: Authelia, Authentik, Keycloak, Okta, Google, etc.

### Prerequisites

1. OIDC-compatible identity provider
2. Client ID and secret from your IdP
3. Public URL for Cartographus (for redirect)

### Configuration

```yaml
environment:
  - AUTH_MODE=oidc
  - OIDC_ENABLED=true
  - OIDC_ISSUER_URL=https://auth.example.com
  - OIDC_CLIENT_ID=cartographus
  - OIDC_CLIENT_SECRET=your-client-secret
  - OIDC_REDIRECT_URL=https://cartographus.example.com/api/auth/oidc/callback
  - OIDC_SCOPES=openid profile email
  - OIDC_PKCE_ENABLED=true
```

### IdP Configuration

Configure your IdP with:

- **Redirect URI**: `https://your-domain/api/auth/oidc/callback`
- **Scopes**: `openid`, `profile`, `email`
- **Grant Type**: Authorization Code with PKCE

### Role Mapping

Map IdP roles to Cartographus roles:

```yaml
environment:
  - OIDC_ROLES_CLAIM=roles
  - OIDC_DEFAULT_ROLES=viewer
```

### Example: Authelia

```yaml
# Authelia configuration
identity_providers:
  oidc:
    clients:
      - id: cartographus
        secret: '$argon2...'  # hashed secret
        redirect_uris:
          - https://cartographus.example.com/api/auth/oidc/callback
        scopes:
          - openid
          - profile
          - email
        authorization_policy: one_factor
```

### Example: Authentik

1. Create an OAuth2/OpenID Provider
2. Create an Application linked to the provider
3. Set redirect URI to `https://cartographus.example.com/api/auth/oidc/callback`
4. Copy Client ID and Secret

---

## Plex Authentication

Allow users to sign in with their Plex accounts. **Works on localhost and private networks without HTTPS.**

### How It Works

1. User clicks "Sign in with Plex"
2. Popup opens to plex.tv
3. User authenticates on Plex's servers
4. Cartographus receives authentication via PIN
5. Server ownership determines role assignment

### Configuration

```yaml
environment:
  - AUTH_MODE=plex
  - PLEX_CLIENT_ID=your-plex-app-id
```

### Automatic Role Assignment

| User Type | Role | Access |
|-----------|------|--------|
| Plex server owner | `admin` | Full access |
| Shared server admin | `editor` | Limited admin |
| Shared user | `viewer` | Own data only |

### Advanced Options

```yaml
environment:
  - PLEX_SERVER_DETECTION=true
  - PLEX_SERVER_OWNER_ROLE=admin
  - PLEX_SERVER_ADMIN_ROLE=editor
  - PLEX_SERVER_MACHINE_ID=your-server-machine-id  # Optional: limit to specific server
```

### Why This Works Without HTTPS

Plex auth uses PIN-based authentication (like how Overseerr works):

1. Cartographus generates a PIN
2. User authenticates at plex.tv
3. plex.tv associates the PIN with the user
4. Cartographus polls for PIN completion
5. No redirect required - works anywhere

---

## Multi Authentication

Enable multiple authentication methods simultaneously.

### Configuration

```yaml
environment:
  - AUTH_MODE=multi
  # JWT (for admin)
  - JWT_SECRET=your-secret
  - ADMIN_USERNAME=admin
  - ADMIN_PASSWORD=YourSecurePassword123!
  # OIDC (for enterprise users)
  - OIDC_ENABLED=true
  - OIDC_ISSUER_URL=https://auth.example.com
  - OIDC_CLIENT_ID=cartographus
  - OIDC_CLIENT_SECRET=secret
  - OIDC_REDIRECT_URL=https://cartographus.example.com/api/auth/oidc/callback
  # Plex (for media server users)
  - PLEX_CLIENT_ID=your-plex-app-id
```

### Login Flow

Users see options for:
- Username/Password (JWT)
- Sign in with Plex
- Sign in with SSO (OIDC)

---

## Basic Authentication

HTTP Basic Authentication. **Requires HTTPS in production.**

```yaml
environment:
  - AUTH_MODE=basic
  - ADMIN_USERNAME=admin
  - ADMIN_PASSWORD=YourSecurePassword123!
```

Password is bcrypt-hashed at startup.

---

## Role-Based Access Control (RBAC)

Cartographus uses Casbin for fine-grained access control.

### Default Roles

| Role | Permissions |
|------|-------------|
| `admin` | Full access to all features and data |
| `editor` | Create and edit, limited admin access |
| `viewer` | Read-only access to own data |

### Custom Policies

```yaml
environment:
  - CASBIN_MODEL_PATH=/config/model.conf
  - CASBIN_POLICY_PATH=/config/policy.csv
```

Example policy.csv:

```csv
p, admin, /api/*, *
p, viewer, /api/v1/stats, GET
p, viewer, /api/v1/analytics/*, GET
```

---

## Session Management

### Session Timeout

```yaml
SESSION_TIMEOUT=24h  # How long sessions remain valid
```

### Session Store

```yaml
# Memory (default, lost on restart)
SESSION_STORE=memory

# File (persistent)
SESSION_STORE=file
SESSION_STORE_PATH=/data/sessions

# Redis (distributed)
SESSION_STORE=redis
REDIS_URL=redis://localhost:6379
```

---

## Security Best Practices

1. **Always use HTTPS** in production (except for Plex auth on local networks)
2. **Use strong secrets** - generate with `openssl rand -base64 48`
3. **Rotate secrets periodically** - changing JWT_SECRET invalidates all sessions
4. **Use OIDC/Plex** for multi-user deployments
5. **Enable rate limiting** - protects against brute force

---

## Troubleshooting

| Issue | Solution |
|-------|----------|
| "Invalid credentials" | Verify username/password. Passwords are case-sensitive. |
| "OIDC callback error" | Check redirect URL matches IdP configuration exactly. |
| "Plex auth popup closes" | Check browser popup blocker. Verify PLEX_CLIENT_ID. |
| "Session expired too quickly" | Increase `SESSION_TIMEOUT` value. |
| "Permission denied" | Check user's role has required permissions. |

---

## Next Steps

- **[Reverse Proxy](Reverse-Proxy)** - HTTPS configuration
- **[Configuration](Configuration)** - All configuration options
- **[Troubleshooting](Troubleshooting)** - Common issues
