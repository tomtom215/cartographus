# ADR-0009: Plex Direct Integration for Standalone Operation

**Date**: 2025-11-24
**Status**: Accepted

---

## Context

Cartographus was originally designed to work exclusively with Tautulli as a data source. However, users requested:

1. **Standalone Operation**: Run without Tautulli installation
2. **Real-Time Events**: Direct Plex webhooks for immediate updates
3. **Multi-Server Support**: Aggregate data from multiple Plex servers
4. **Reduced Latency**: Skip Tautulli intermediary for live events

### Plex API Capabilities

| Endpoint | Description | Use Case |
|----------|-------------|----------|
| `/status/sessions` | Active playback sessions | Real-time monitoring |
| `/library/sections` | Library metadata | Content analytics |
| `/accounts` | User information | User mapping |
| `/identity` | Server identification | Multi-server support |
| Webhooks | Playback events | Real-time notifications |

### Alternatives Considered

| Approach | Pros | Cons |
|----------|------|------|
| **Tautulli Only** | Simpler, proven | External dependency |
| **Plex Only** | Direct, real-time | Less historical data |
| **Dual Mode** | Best of both | More complex |

---

## Decision

Implement **Plex Direct Integration** as an optional alternative to Tautulli:

- **Plex Webhooks**: Receive real-time playback events
- **Plex REST API**: Server status, sessions, library data
- **Plex OAuth**: Secure authentication with plex.tv
- **Dual Mode**: Can run alongside Tautulli or standalone

### Architecture

```
                     STANDALONE MODE
                     ═══════════════

┌─────────────┐                        ┌─────────────┐
│  Plex       │ ───── Webhook ─────▶   │ Cartographus│
│  Server     │                        │             │
└──────┬──────┘                        └──────┬──────┘
       │                                      │
       │ REST API                             │
       │ (/status/sessions)                   │
       │                                      ▼
       │                              ┌─────────────┐
       └────────────────────────────▶ │   DuckDB    │
                                      │ (Analytics) │
                                      └─────────────┘


                     HYBRID MODE
                     ═══════════

┌─────────────┐      ┌─────────────┐      ┌─────────────┐
│  Plex       │ ───▶ │  Tautulli   │ ───▶ │Cartographus │
│  Server     │      │  (History)  │      │             │
└──────┬──────┘      └─────────────┘      └──────┬──────┘
       │                                         │
       │ Webhook (Real-time)                     │
       └─────────────────────────────────────────┘
```

### Key Factors

1. **User Choice**: Not everyone uses Tautulli
2. **Real-Time Updates**: Webhooks faster than polling
3. **Reduced Dependencies**: Fewer external requirements
4. **Data Aggregation**: Multiple Plex servers in one view

---

## Consequences

### Positive

- **Standalone Operation**: No Tautulli required
- **Real-Time Events**: WebSocket and webhook delivery
- **OAuth Security**: PKCE-based Plex authentication flow
- **Comprehensive API Coverage**: Sessions, library, accounts, transcode monitoring

### Negative

- **Historical Gap**: Plex doesn't provide history like Tautulli
- **API Complexity**: More endpoints to maintain
- **Rate Limiting**: Plex API has stricter limits
- **Session Management**: Must track active sessions manually

### Neutral

- **Optional Feature**: Default still uses Tautulli
- **Hybrid Mode**: Can use both simultaneously

---

## Implementation

### Plex OAuth Flow

The OAuth implementation uses PKCE (Proof Key for Code Exchange) for secure authentication:

```go
// internal/api/handlers_plex_oauth.go
func (h *Handler) PlexOAuthStart(w http.ResponseWriter, r *http.Request) {
    // Generate PKCE challenge
    pkce, err := h.plexOAuthClient.GeneratePKCE()
    if err != nil {
        respondError(w, http.StatusInternalServerError, "OAUTH_ERROR", "Failed to generate PKCE challenge", err)
        return
    }

    // Generate random state token for CSRF protection
    stateBytes := make([]byte, 32)
    rand.Read(stateBytes)
    state := base64.RawURLEncoding.EncodeToString(stateBytes)

    // Store PKCE verifier and state in HTTP-only cookie
    // Build authorization URL
    authURL := h.plexOAuthClient.BuildAuthorizationURL(pkce.CodeChallenge, state)

    respondJSON(w, http.StatusOK, &models.APIResponse{
        Status: "success",
        Data: map[string]interface{}{
            "authorization_url": authURL,
            "state":             state,
        },
    })
}

func (h *Handler) PlexOAuthCallback(w http.ResponseWriter, r *http.Request) {
    code := r.URL.Query().Get("code")
    state := r.URL.Query().Get("state")

    // Validate state parameter (CSRF protection)
    // Exchange authorization code for access token
    plexToken, err := h.plexOAuthClient.ExchangeCodeForToken(r.Context(), code, codeVerifier)
    if err != nil {
        respondError(w, http.StatusUnauthorized, "TOKEN_EXCHANGE_FAILED", "Failed to exchange code", err)
        return
    }

    // Fetch Plex user information and generate JWT session token
    // Set HTTP-only cookies for session and Plex token
    respondJSON(w, http.StatusOK, &models.APIResponse{Status: "success", ...})
}
```

### Plex Webhook Handler

```go
// internal/api/handlers_plex_webhook.go
func (h *Handler) PlexWebhook(w http.ResponseWriter, r *http.Request) {
    // Check if webhooks are enabled
    if !h.config.Plex.WebhooksEnabled {
        respondError(w, http.StatusNotFound, "WEBHOOKS_DISABLED", "Plex webhooks are not enabled", nil)
        return
    }

    // Verify HMAC-SHA256 signature if secret is configured
    if h.config.Plex.WebhookSecret != "" {
        signature := r.Header.Get("X-Plex-Signature")
        if !h.verifyWebhookSignature(body, signature, h.config.Plex.WebhookSecret) {
            respondError(w, http.StatusUnauthorized, "INVALID_SIGNATURE", "Webhook signature verification failed", nil)
            return
        }
    }

    // Parse webhook payload
    var webhook models.PlexWebhook
    json.Unmarshal(body, &webhook)

    // Route by event type: media.play, media.pause, media.resume, media.stop, etc.
    // Publish to NATS for event-driven processing
    h.publishWebhookEvent(r.Context(), &webhook)

    // Broadcast to WebSocket clients
    h.broadcastWebhookEvent(&webhook)

    w.WriteHeader(http.StatusOK)
}
```

### Plex Session Polling

```go
// internal/sync/plex_session_poller.go
type PlexSessionPoller struct {
    manager      *Manager
    config       SessionPollerConfig
    seenSessions *cache.LRUCache // O(1) session tracking
}

func (p *PlexSessionPoller) pollLoop(ctx context.Context) {
    ticker := time.NewTicker(p.config.Interval) // Default: 30 seconds
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            sessions, err := p.manager.plexClient.GetTranscodeSessions(ctx)
            if err != nil {
                continue
            }

            for i := range sessions {
                session := &sessions[i]
                // Skip if seen recently (uses LRU cache)
                if p.seenSessions.IsDuplicate(session.SessionKey) {
                    continue
                }

                // Convert to PlaybackEvent and publish to NATS
                event := p.manager.plexSessionToPlaybackEvent(ctx, session)
                p.manager.publishEvent(ctx, event)
            }
        }
    }
}
```

### Plex WebSocket Client

```go
// internal/sync/plex_websocket.go
type PlexWebSocketClient struct {
    baseURL string
    token   string
    conn    *websocket.Conn
    // Callbacks for notification types
    onPlaying  func(models.PlexPlayingNotification)
    onTimeline func(models.PlexTimelineNotification)
    onActivity func(models.PlexActivityNotification)
    onStatus   func(models.PlexStatusNotification)
}

func (c *PlexWebSocketClient) Connect(ctx context.Context) error {
    // Build WebSocket URL: ws://{host}/:/websockets/notifications?X-Plex-Token={token}
    wsURL, _ := c.buildWebSocketURL()

    dialer := websocket.Dialer{HandshakeTimeout: 10 * time.Second}
    conn, _, err := dialer.DialContext(ctx, wsURL, nil)
    if err != nil {
        return err
    }
    c.conn = conn

    // Start message listener and ping/pong keepalive goroutines
    go c.listen(ctx)
    go c.pingLoop(ctx)
    return nil
}
```

### API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/auth/plex/start` | GET | Initiate Plex OAuth (PKCE) |
| `/api/v1/auth/plex/callback` | GET | OAuth callback |
| `/api/v1/auth/plex/refresh` | POST | Refresh access token |
| `/api/v1/auth/plex/revoke` | POST | Revoke access token |
| `/api/v1/plex/webhook` | POST | Receive Plex webhooks |
| `/api/v1/plex/sessions` | GET | Active sessions |
| `/api/v1/plex/identity` | GET | Server identification |
| `/api/v1/plex/capabilities` | GET | Server capabilities |
| `/api/v1/plex/devices` | GET | Connected devices |
| `/api/v1/plex/accounts` | GET | User accounts |
| `/api/v1/plex/activities` | GET | Server activities |
| `/api/v1/plex/playlists` | GET | User playlists |
| `/api/v1/plex/library/sections` | GET | Library sections |
| `/api/v1/plex/library/onDeck` | GET | On-deck content |
| `/api/v1/plex/library/metadata/{ratingKey}` | GET | Media metadata |
| `/api/v1/plex/transcode/sessions` | GET | Transcode sessions |
| `/api/v1/plex/transcode/sessions/{sessionKey}` | DELETE | Cancel transcode |

### Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `ENABLE_PLEX_SYNC` | `false` | Enable Plex direct integration |
| `PLEX_URL` | (none) | Plex Media Server URL (e.g., `http://localhost:32400`) |
| `PLEX_TOKEN` | (none) | X-Plex-Token for authentication |
| `PLEX_SYNC_INTERVAL` | `5m` | How often to check for missed events |
| `PLEX_REALTIME_ENABLED` | `true` | Enable WebSocket for real-time updates |
| `ENABLE_PLEX_WEBHOOKS` | `false` | Enable Plex webhook endpoint |
| `PLEX_WEBHOOK_SECRET` | (none) | HMAC-SHA256 secret for webhook signature |
| `PLEX_OAUTH_CLIENT_ID` | (none) | OAuth client ID for Plex authentication |
| `PLEX_OAUTH_REDIRECT_URI` | (none) | OAuth callback URL |

### Code References

| Component | File | Notes |
|-----------|------|-------|
| Webhook handler | `internal/api/handlers_plex_webhook.go` | Event processing |
| OAuth handlers | `internal/api/handlers_plex_oauth.go` | PKCE authentication |
| REST API handlers | `internal/api/handlers_plex_api.go` | Library, sessions, devices, etc. |
| Session poller | `internal/sync/plex_session_poller.go` | Fallback polling with LRU cache |
| Session methods | `internal/sync/plex_sessions.go` | `GetTranscodeSessions`, `CancelTranscode` |
| WebSocket client | `internal/sync/plex_websocket.go` | Real-time event stream |
| API models | `internal/models/plex_api_expanded.go` | Server capabilities, transcode, metadata |
| Notification models | `internal/models/plex_notifications.go` | Session and WebSocket types |
| OAuth client | `internal/auth/plex_oauth.go` | PKCE flow implementation |

---

## Verification

### Verified Claims

| Claim | Source | Verified |
|-------|--------|----------|
| Plex webhook support | `internal/api/handlers_plex_webhook.go` | Yes |
| OAuth PKCE flow | `internal/api/handlers_plex_oauth.go` | Yes |
| REST API coverage | `internal/api/handlers_plex_api.go` | Yes |
| WebSocket real-time events | `internal/sync/plex_websocket.go` | Yes |
| Session polling | `internal/sync/plex_session_poller.go` | Yes |

### Test Coverage

- Plex API tests: `internal/api/handlers_plex_api_test.go`, `internal/api/handlers_plex_api_extended_test.go`
- Webhook tests: `internal/api/handlers_plex_webhook_test.go`, `internal/api/handlers_plex_webhook_integration_test.go`
- OAuth tests: `internal/api/handlers_plex_oauth_test.go`
- WebSocket tests: `internal/sync/plex_websocket_test.go`
- Session tests: `internal/sync/plex_session_poller_test.go`
- Coverage target: 90%+ for Plex handlers

---

## Related ADRs

- [ADR-0005](0005-nats-jetstream-event-processing.md): Webhook events to NATS
- [ADR-0007](0007-event-sourcing-architecture.md): Plex as event source
- [ADR-0008](0008-circuit-breaker-pattern.md): Plex API circuit breaker

---

## References

- [Plex API Documentation](https://github.com/Arcanemagus/plex-api/wiki)
- [Plex Webhooks](https://support.plex.tv/articles/115002267687-webhooks/)
- [Plex OAuth Flow](https://forums.plex.tv/t/authenticating-with-plex/609370)
