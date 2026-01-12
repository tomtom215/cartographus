# ADR-0026: Multi-Server Management UI

**Date**: 2026-01-07
**Status**: Implemented

---

## Context

Cartographus supports multiple media servers (Plex, Jellyfin, Emby, Tautulli) configured via environment variables. The `config.go` already supports array configurations (`PlexServers[]`, `JellyfinServers[]`, `EmbyServers[]`) for multi-server deployments, but server management is currently limited to:

1. Environment variables set at container startup
2. No runtime modification without container restart
3. No visibility into server connection status in the UI
4. No ability to add/remove servers dynamically

### Requirements

1. **Runtime Configuration**: Add/remove media servers without container restart
2. **Connection Testing**: Validate server connectivity before saving
3. **Status Visibility**: Real-time view of server sync status, last sync time, error states
4. **Credential Security**: Secure storage of API tokens and secrets
5. **Backwards Compatibility**: Environment variable configuration must continue to work
6. **Multi-Server Deduplication**: Maintain existing `ServerID`-based deduplication
7. **RBAC**: Only admin users can manage servers
8. **Audit Trail**: Log all server configuration changes
9. **Graceful Degradation**: If a server becomes unreachable, continue with others
10. **Migration Path**: Optional migration from env vars to DB-stored config

### Technical Constraints

| Constraint | Impact |
|------------|--------|
| Suture supervisor tree | Services must be dynamically addable/removable |
| Single binary deployment | No external secrets manager dependency |
| Self-hosted environments | Limited resources (4-8GB RAM typical) |
| DuckDB single-writer | Credential storage must not conflict with analytics |
| NATS JetStream | Event sourcing architecture must be preserved |

### Alternatives Considered

| Approach | Pros | Cons |
|----------|------|------|
| **A: DB-Stored Config (Full)** | Runtime changes, single source of truth | Complex migration, credential security |
| **B: Hybrid (Env + DB)** | Backwards compatible, gradual adoption | Dual config sources, complexity |
| **C: Config File Hot-Reload** | Simple, no DB changes | File-based credentials less secure |
| **D: External Secrets Manager** | Enterprise-grade security | External dependency, complexity |

---

## Decision

Implement **Option B: Hybrid Configuration** with the following architecture:

1. **Environment variables remain the primary configuration source** for initial setup
2. **Database stores additional servers** added via UI (optional)
3. **Merged configuration** at runtime: env vars + DB-stored servers
4. **Encrypted credential storage** using AES-256-GCM with a key derived from JWT_SECRET
5. **Dynamic service management** via Suture supervisor

### Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        Configuration Sources                                 │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌──────────────────────┐     ┌──────────────────────┐                      │
│  │   Environment Vars   │     │   Database (Optional) │                     │
│  │   (Koanf Loader)     │     │   media_servers table │                     │
│  │                      │     │   (encrypted creds)   │                     │
│  │  PLEX_URL=...        │     │                       │                     │
│  │  PLEX_TOKEN=...      │     │  ┌─────────────────┐  │                     │
│  │  JELLYFIN_URL=...    │     │  │ id: uuid        │  │                     │
│  │                      │     │  │ platform: plex  │  │                     │
│  └──────────┬───────────┘     │  │ url: encrypted  │  │                     │
│             │                 │  │ token: encrypted│  │                     │
│             │                 │  │ enabled: bool   │  │                     │
│             │                 │  └─────────────────┘  │                     │
│             │                 └──────────┬────────────┘                     │
│             │                            │                                   │
│             └────────────┬───────────────┘                                   │
│                          │                                                   │
│                          ▼                                                   │
│             ┌────────────────────────┐                                       │
│             │   ServerConfigManager  │                                       │
│             │   (Merge & Validate)   │                                       │
│             └────────────┬───────────┘                                       │
│                          │                                                   │
│                          ▼                                                   │
│             ┌────────────────────────┐                                       │
│             │   []UnifiedServerConfig │                                      │
│             │   (Canonical format)    │                                      │
│             └────────────┬───────────┘                                       │
└──────────────────────────┼──────────────────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                        Supervisor Tree Integration                           │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌───────────────────────────────────────────────────────────────────────┐  │
│  │                    ServerSupervisor (Dynamic)                          │  │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐   │  │
│  │  │ PlexSync    │  │ PlexSync    │  │ JellyfinSync│  │ EmbySync    │   │  │
│  │  │ Server1     │  │ Server2     │  │ Server1     │  │ Server1     │   │  │
│  │  │ (env-var)   │  │ (db-stored) │  │ (env-var)   │  │ (db-stored) │   │  │
│  │  └─────────────┘  └─────────────┘  └─────────────┘  └─────────────┘   │  │
│  └───────────────────────────────────────────────────────────────────────┘  │
│                                                                              │
│  Methods:                                                                    │
│  - AddServer(config) → starts new sync service                              │
│  - RemoveServer(id) → gracefully stops service                              │
│  - UpdateServer(id, config) → restart with new config                       │
│  - GetServerStatus(id) → returns sync state, last sync, errors              │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Credential Encryption

API tokens and secrets are encrypted before storage using AES-256-GCM:

```go
// internal/config/encryption.go
type CredentialEncryptor struct {
    key []byte // Derived from JWT_SECRET using HKDF
}

func (e *CredentialEncryptor) Encrypt(plaintext string) (string, error) {
    // AES-256-GCM with random nonce
    // Returns base64-encoded ciphertext
}

func (e *CredentialEncryptor) Decrypt(ciphertext string) (string, error) {
    // Decrypts and returns plaintext
}
```

Key derivation:
- Input: `JWT_SECRET` (already required for auth)
- Algorithm: HKDF-SHA256 with salt "cartographus-server-credentials"
- Output: 32-byte AES key

### Database Schema

```sql
-- Store additional servers beyond env vars
CREATE TABLE IF NOT EXISTS media_servers (
    id TEXT PRIMARY KEY,
    platform TEXT NOT NULL CHECK (platform IN ('plex', 'jellyfin', 'emby', 'tautulli')),
    name TEXT NOT NULL,
    url_encrypted TEXT NOT NULL,
    token_encrypted TEXT NOT NULL,
    server_id TEXT UNIQUE,              -- For deduplication
    enabled BOOLEAN DEFAULT true,

    -- Platform-specific settings (JSON)
    settings JSON DEFAULT '{}',

    -- Sync configuration
    realtime_enabled BOOLEAN DEFAULT false,
    webhooks_enabled BOOLEAN DEFAULT false,
    session_polling_enabled BOOLEAN DEFAULT false,
    session_polling_interval TEXT DEFAULT '30s',

    -- Metadata
    source TEXT DEFAULT 'ui' CHECK (source IN ('env', 'ui', 'import')),
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    created_by TEXT,                    -- User ID who added

    -- Status (updated by sync service)
    last_sync_at TIMESTAMPTZ,
    last_sync_status TEXT,
    last_error TEXT,
    last_error_at TIMESTAMPTZ
);

CREATE INDEX idx_media_servers_platform ON media_servers(platform);
CREATE INDEX idx_media_servers_enabled ON media_servers(enabled);

-- Audit log for server changes
CREATE TABLE IF NOT EXISTS media_server_audit (
    id INTEGER PRIMARY KEY,
    server_id TEXT NOT NULL,
    action TEXT NOT NULL CHECK (action IN ('create', 'update', 'delete', 'enable', 'disable', 'test')),
    user_id TEXT NOT NULL,
    changes JSON,                       -- What changed (redacted credentials)
    ip_address TEXT,
    user_agent TEXT,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_media_server_audit_server ON media_server_audit(server_id);
CREATE INDEX idx_media_server_audit_time ON media_server_audit(created_at);
```

### API Endpoints

| Method | Path | Description | Auth |
|--------|------|-------------|------|
| GET | `/api/v1/admin/servers` | List all servers (env + DB) | Admin |
| POST | `/api/v1/admin/servers` | Add new server | Admin |
| POST | `/api/v1/admin/servers/test` | Test connectivity | Admin |
| GET | `/api/v1/admin/servers/db` | List DB-stored servers only | Admin |
| GET | `/api/v1/admin/servers/{id}` | Get server details | Admin |
| PUT | `/api/v1/admin/servers/{id}` | Update server config | Admin |
| DELETE | `/api/v1/admin/servers/{id}` | Remove server (DB-stored only) | Admin |

Note: Enable/disable is handled via `PUT /admin/servers/{id}` with `enabled` field.

### Request/Response Models

```go
// internal/models/media_server.go

type CreateMediaServerRequest struct {
    Platform               string         `json:"platform" validate:"required,oneof=plex jellyfin emby tautulli"`
    Name                   string         `json:"name" validate:"required,min=1,max=100"`
    URL                    string         `json:"url" validate:"required,url"`
    Token                  string         `json:"token" validate:"required,min=8"`
    RealtimeEnabled        bool           `json:"realtime_enabled"`
    WebhooksEnabled        bool           `json:"webhooks_enabled"`
    SessionPollingEnabled  bool           `json:"session_polling_enabled"`
    SessionPollingInterval string         `json:"session_polling_interval" validate:"omitempty"`
    Settings               map[string]any `json:"settings"`
}

type MediaServerResponse struct {
    ID                     string     `json:"id"`
    Platform               string     `json:"platform"`
    Name                   string     `json:"name"`
    URL                    string     `json:"url"`          // Decrypted and shown
    TokenMasked            string     `json:"token_masked"` // "****...last4"
    ServerID               string     `json:"server_id"`
    Enabled                bool       `json:"enabled"`
    Source                 string     `json:"source"`
    RealtimeEnabled        bool       `json:"realtime_enabled"`
    WebhooksEnabled        bool       `json:"webhooks_enabled"`
    SessionPollingEnabled  bool       `json:"session_polling_enabled"`
    SessionPollingInterval string     `json:"session_polling_interval"`
    Status                 string     `json:"status"` // connected, syncing, error, disabled
    LastSyncAt             *time.Time `json:"last_sync_at,omitempty"`
    LastSyncStatus         string     `json:"last_sync_status,omitempty"`
    LastError              string     `json:"last_error,omitempty"`
    LastErrorAt            *time.Time `json:"last_error_at,omitempty"`
    CreatedAt              time.Time  `json:"created_at"`
    UpdatedAt              time.Time  `json:"updated_at"`
    Immutable              bool       `json:"immutable"` // true for env-var servers
}

type MediaServerTestResponse struct {
    Success    bool   `json:"success"`
    LatencyMs  int64  `json:"latency_ms"`
    ServerName string `json:"server_name,omitempty"`
    Version    string `json:"version,omitempty"`
    Error      string `json:"error,omitempty"`
    ErrorCode  string `json:"error_code,omitempty"`
}
```

### Dynamic Service Management

```go
// internal/supervisor/server_supervisor.go

type ServerSupervisor struct {
    tree           *SupervisorTree
    configMgr      *config.ServerConfigManager
    services       map[string]*managedService // serverID -> managed service
    mu             sync.RWMutex

    // Dependencies for service creation
    db             ServerDB
    eventPublisher EventPublisher
    wsHub          WebSocketHub
    userResolver   UserResolver
}

// AddServer starts a new sync service for a server
func (s *ServerSupervisor) AddServer(ctx context.Context, cfg *config.UnifiedServerConfig) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    serverID := s.getServerKey(cfg)
    if _, exists := s.services[serverID]; exists {
        return ErrServerAlreadyExists
    }

    // Create appropriate sync service
    svc, err := s.createService(cfg)
    if err != nil {
        return fmt.Errorf("failed to create sync service: %w", err)
    }

    // Add to supervisor tree (messaging layer)
    token := s.tree.AddMessagingService(svc)
    s.services[serverID] = &managedService{
        token:     token,
        config:    cfg,
        service:   svc,
        startedAt: time.Now(),
    }

    return nil
}

// RemoveServer gracefully stops a server's sync service
func (s *ServerSupervisor) RemoveServer(ctx context.Context, serverID string) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    managed, exists := s.services[serverID]
    if !exists {
        return ErrServerNotRunning
    }

    // Remove from messaging layer supervisor (triggers graceful shutdown)
    if err := s.tree.RemoveMessagingService(managed.token); err != nil {
        return fmt.Errorf("failed to remove service from supervisor: %w", err)
    }

    delete(s.services, serverID)
    return nil
}

// GetServerStatus returns current status of a managed server
func (s *ServerSupervisor) GetServerStatus(serverID string) (*ServerStatus, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()

    managed, exists := s.services[serverID]
    if !exists {
        return nil, ErrServerNotRunning
    }

    return &ServerStatus{
        ServerID: managed.config.ServerID,
        Platform: managed.config.Platform,
        Name:     managed.config.Name,
        Running:  true,
        Status:   managed.config.Status,
    }, nil
}
```

### Frontend Components

```typescript
// web/src/app/MultiServerManager.ts

export class MultiServerManager {
    private api: API;
    private servers: MediaServerStatus[] = [];
    private editingServer: EditingServer | null = null;
    private modalMode: ModalMode = 'add';

    // Load all servers with status
    async loadServers(): Promise<void>

    // Show add server modal
    private showAddModal(): void

    // Show edit server modal
    private async showEditModal(serverId: string): Promise<void>

    // Test connection before saving
    private async testConnection(): Promise<void>

    // Save server (create or update)
    private async saveServer(): Promise<void>

    // Toggle server enabled state
    private async toggleServer(serverId: string, enabled: boolean): Promise<void>

    // Trigger sync for server
    private async triggerSync(serverId: string): Promise<void>

    // Delete server with confirmation
    private async confirmDelete(): Promise<void>
}
```

UI Components (rendered by MultiServerManager):
- **Server List Panel**: Grid of server cards with status indicators
- **Status Summary Cards**: Connected, syncing, and error counts
- **Add/Edit Modal**: Form for platform, URL, token, and sync options
- **Connection Test Result**: Real-time connectivity feedback
- **Delete Confirmation Modal**: Confirmation before removal

---

## Consequences

### Positive

- Runtime server management without container restart
- Visibility into server connection status
- Gradual adoption path (env vars still work)
- Audit trail for compliance
- Secure credential storage

### Negative

- Increased complexity (dual config sources)
- Additional database tables
- Credential encryption adds latency (~1ms)
- Migration complexity for existing deployments

### Neutral

- Env-var servers marked as immutable in UI
- JWT_SECRET becomes critical for credential encryption
- Requires careful handling of key rotation

---

## Implementation Phases

### Phase 1: Read-Only Status UI (1-2 days)
- Display servers from env vars in UI
- Show connection status, last sync time
- No DB storage, no modification

### Phase 2: Database Schema & Encryption (2-3 days)
- Create `media_servers` and `media_server_audit` tables
- Implement `CredentialEncryptor`
- API endpoints for CRUD operations

### Phase 3: Dynamic Service Management (3-4 days)
- `ServerSupervisor` with Add/Remove/Update
- Integration with existing sync services
- Graceful service lifecycle

### Phase 4: Frontend UI (2-3 days)
- Server list with status indicators
- Add server wizard
- Edit/delete for DB-stored servers
- Connection testing

### Phase 5: Testing & Documentation (2-3 days)
- Unit tests (90%+ coverage)
- Integration tests with mock servers
- E2E tests for UI flows
- Migration guide documentation

**Total Estimated Effort: 10-15 days**

---

## Security Considerations

1. **Credential Encryption**: AES-256-GCM with key derived from JWT_SECRET
2. **Key Rotation**: Changing JWT_SECRET requires re-encryption of all stored credentials
3. **Token Masking**: API never returns full tokens, only masked versions
4. **Audit Logging**: All configuration changes logged with user, IP, timestamp
5. **RBAC**: Only admin role can access server management endpoints
6. **Input Validation**: All URLs validated, tokens sanitized

---

## Testing Requirements

| Component | Coverage Target | Test Types |
|-----------|-----------------|------------|
| CredentialEncryptor | 100% | Unit |
| ServerConfigManager | 90%+ | Unit, Integration |
| ServerSupervisor | 90%+ | Unit, Integration |
| API Handlers | 90%+ | Unit, Integration |
| Frontend | 80%+ | E2E (Playwright) |

### Key Test Scenarios

1. **Encryption round-trip**: Encrypt → Decrypt returns original
2. **Config merging**: Env vars + DB servers merged correctly
3. **Duplicate detection**: Same URL rejected, ServerID enforced
4. **Service lifecycle**: Add → Sync → Update → Remove works
5. **Error handling**: Invalid URL, bad token, unreachable server
6. **RBAC**: Non-admin users rejected
7. **Audit trail**: All changes logged correctly

---

## Related ADRs

- [ADR-0004](0004-process-supervision-with-suture.md): Suture v4 process supervision (dynamic services)
- [ADR-0005](0005-nats-jetstream-event-processing.md): Event sourcing architecture
- [ADR-0009](0009-plex-direct-integration.md): Plex direct integration
- [ADR-0012](0012-configuration-management-koanf.md): Koanf configuration loading
- [ADR-0015](0015-zero-trust-authentication-authorization.md): RBAC authorization

---

## References

- [Suture v4 Documentation](https://pkg.go.dev/github.com/thejerf/suture/v4) - Dynamic service management
- [AES-GCM in Go](https://pkg.go.dev/crypto/cipher#NewGCM) - Authenticated encryption
- [HKDF (RFC 5869)](https://tools.ietf.org/html/rfc5869) - Key derivation function
- [Plex API Documentation](https://github.com/Arcanemagus/plex-api/wiki) - Server connectivity
- [Jellyfin API Documentation](https://api.jellyfin.org/) - Server connectivity
- [Emby API Documentation](https://github.com/MediaBrowser/Emby/wiki/Api) - Server connectivity

---

## Implementation Notes (2026-01-07)

All five phases have been completed:

### Phase 1: Backend Schema (Complete)
- Database schema added to `internal/database/database_schema.go`
- Tables: `media_servers`, `media_server_audit`
- CRUD methods in `internal/database/crud_media_servers.go`
- Credential encryption in `internal/config/encryption.go`

### Phase 2: Backend API (Complete)
- Handlers in `internal/api/handlers_server_management.go`
- Routes registered in `internal/api/chi_router.go`
- Models in `internal/models/media_server.go`

### Phase 3: Supervisor Integration (Complete)
- Dynamic service management in `internal/supervisor/server_supervisor.go`
- Integration with existing sync services via `internal/supervisor/sync_service_wrappers.go`

### Phase 4: Frontend UI (Complete)
- `web/src/app/MultiServerManager.ts` (~800 lines) - Full CRUD manager
- `web/src/lib/api/server.ts` - API client methods
- `web/src/lib/types/server.ts` - TypeScript types
- `web/src/styles/features/server-management.css` - Styling

### Phase 5: Testing & Documentation (Complete)
- E2E tests: `tests/e2e/20-server-management.spec.ts`
- Integration tests: `internal/api/handlers_server_management_test.go`
- Mock handlers: `tests/e2e/fixtures/mock-server.ts`
- ADR status updated to Implemented

### Key Files
| Category | Files |
|----------|-------|
| Backend Handlers | `internal/api/handlers_server_management.go` |
| Database | `internal/database/database_schema.go`, `internal/database/crud_media_servers.go` |
| Encryption | `internal/config/encryption.go` |
| Supervisor | `internal/supervisor/server_supervisor.go`, `internal/supervisor/sync_service_wrappers.go` |
| Frontend UI | `web/src/app/MultiServerManager.ts` |
| API Client | `web/src/lib/api/server.ts`, `web/src/lib/types/server.ts` |
| Tests | `internal/api/handlers_server_management_test.go`, `tests/e2e/20-server-management.spec.ts` |
