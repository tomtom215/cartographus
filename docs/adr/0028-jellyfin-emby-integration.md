# ADR-0028: Jellyfin and Emby Media Server Integration

**Date**: 2026-01-11
**Status**: Accepted
**Last Verified**: 2026-01-11

---

## Context

While ADR-0009 covers Plex integration, Cartographus also supports two additional media servers:

- **Jellyfin**: Open-source, self-hosted media server (fork of Emby)
- **Emby**: Commercial media server with similar API structure

### Requirements

1. **Multi-Source Analytics**: Aggregate playback data across Plex, Jellyfin, and Emby
2. **Consistent Data Model**: Normalize events to unified `PlaybackEvent` schema
3. **Real-Time Support**: WebSocket notifications for live playback updates
4. **Session Polling**: Periodic polling as fallback for WebSocket gaps
5. **Circuit Breaker**: Fault tolerance per media server
6. **Cross-Server Deduplication**: Prevent duplicate events from overlapping sources

### API Similarities

Jellyfin and Emby share API heritage, with nearly identical endpoints:

| Endpoint | Jellyfin | Emby | Notes |
|----------|----------|------|-------|
| Ping | `/System/Ping` | `/System/Ping` | Same |
| Sessions | `/Sessions` | `/Sessions` | Same |
| System Info | `/System/Info` | `/System/Info` | Same |
| Users | `/Users` | `/Users` | Same |
| WebSocket | `/socket` | `/embywebsocket` | Different path |
| Auth Header | `X-Emby-Token` | `X-Emby-Token` | Same (Jellyfin uses Emby header) |

---

## Decision

Implement **parallel client architectures** for Jellyfin and Emby with:

1. **Separate Client Implementations**: Despite API similarity, maintain distinct clients for versioning flexibility
2. **Common Interface Pattern**: Both implement analogous interface contracts
3. **Unified Event Mapping**: Convert to common `PlaybackEvent` schema
4. **Per-Server Circuit Breakers**: Independent failure isolation
5. **WebSocket + Polling Hybrid**: Real-time notifications with polling backup

### Architecture

```
┌─────────────────┐   ┌─────────────────┐   ┌─────────────────┐
│  Plex Server    │   │ Jellyfin Server │   │  Emby Server    │
└────────┬────────┘   └────────┬────────┘   └────────┬────────┘
         │                     │                     │
         ▼                     ▼                     ▼
┌─────────────────┐   ┌─────────────────┐   ┌─────────────────┐
│  PlexClient     │   │ JellyfinClient  │   │  EmbyClient     │
│  + WebSocket    │   │  + WebSocket    │   │  + WebSocket    │
│  + SessionPoll  │   │  + SessionPoll  │   │  + SessionPoll  │
│  + CircuitBrkr  │   │  + CircuitBrkr  │   │  + CircuitBrkr  │
└────────┬────────┘   └────────┬────────┘   └────────┬────────┘
         │                     │                     │
         └──────────────┬──────┴──────────────┬──────┘
                        │                     │
                        ▼                     ▼
              ┌─────────────────┐   ┌─────────────────┐
              │  Event Mapper   │   │  Sync Manager   │
              │ (Normalization) │   │ (Orchestration) │
              └────────┬────────┘   └─────────────────┘
                       │
                       ▼
              ┌─────────────────┐
              │  PlaybackEvent  │
              │ (Unified Model) │
              └─────────────────┘
```

---

## Consequences

### Positive

- **Source Agnostic**: Users can mix Plex, Jellyfin, and Emby servers
- **Consistent UX**: Same analytics experience regardless of media server
- **Failure Isolation**: One server's outage doesn't affect others
- **Future Proof**: Easy to add new media server types

### Negative

- **Code Duplication**: Jellyfin and Emby clients are nearly identical (by design)
- **Maintenance Burden**: API changes must be tracked for three platforms
- **User ID Mapping**: Jellyfin/Emby use string UUIDs vs Plex integer IDs

### Neutral

- **HTTP Timeout**: 30-second timeout for all API calls
- **Source Field**: Events tagged with `source: "jellyfin"` or `source: "emby"`

---

## Implementation

### Client Interfaces

```go
// JellyfinClientInterface defines Jellyfin API operations
// Location: internal/sync/jellyfin_client.go:26-36
type JellyfinClientInterface interface {
    Ping(ctx context.Context) error
    GetSessions(ctx context.Context) ([]models.JellyfinSession, error)
    GetActiveSessions(ctx context.Context) ([]models.JellyfinSession, error)
    GetSystemInfo(ctx context.Context) (*JellyfinSystemInfo, error)
    GetUsers(ctx context.Context) ([]JellyfinUser, error)
    StopSession(ctx context.Context, sessionID string) error
    GetWebSocketURL() (string, error)
}

// EmbyClientInterface defines Emby API operations
// Location: internal/sync/emby_client.go:26-36
type EmbyClientInterface interface {
    Ping(ctx context.Context) error
    GetSessions(ctx context.Context) ([]models.EmbySession, error)
    GetActiveSessions(ctx context.Context) ([]models.EmbySession, error)
    GetSystemInfo(ctx context.Context) (*EmbySystemInfo, error)
    GetUsers(ctx context.Context) ([]EmbyUser, error)
    StopSession(ctx context.Context, sessionID string) error
    GetWebSocketURL() (string, error)
}
```

### Session to PlaybackEvent Mapping

```go
// SessionToPlaybackEvent converts a Jellyfin session to a PlaybackEvent
// Location: internal/sync/jellyfin_client.go:331-462
func SessionToPlaybackEvent(session *models.JellyfinSession, _ string) *models.PlaybackEvent {
    if session == nil || session.NowPlayingItem == nil {
        return nil
    }

    item := session.NowPlayingItem
    event := &models.PlaybackEvent{
        SessionKey:        session.ID,
        Source:            "jellyfin",  // or "emby" for EmbySessionToPlaybackEvent
        TranscodeDecision: &transcodeDecision,
        Platform:          session.Client,
        Player:            session.DeviceName,
        // ... field mapping
    }
    return event
}
```

### WebSocket URL Generation

```go
// GetWebSocketURL returns the WebSocket URL for real-time notifications
// Location: internal/sync/jellyfin_client.go:256-284
func (c *JellyfinClient) GetWebSocketURL() (string, error) {
    parsedURL, err := url.Parse(c.baseURL)
    // Convert http(s) to ws(s)
    switch parsedURL.Scheme {
    case "http":
        parsedURL.Scheme = "ws"
    case "https":
        parsedURL.Scheme = "wss"
    }
    parsedURL.Path = "/socket"  // "/embywebsocket" for Emby
    query := parsedURL.Query()
    query.Set("api_key", c.apiKey)
    query.Set("deviceId", "cartographus")
    parsedURL.RawQuery = query.Encode()
    return parsedURL.String(), nil
}
```

### HTTP Request Headers

Both clients use the same authentication headers (Jellyfin inherited Emby's API):

```go
// Location: internal/sync/jellyfin_client.go:296-302
req.Header.Set("X-Emby-Token", c.apiKey)
req.Header.Set("X-Emby-Client", "Cartographus")
req.Header.Set("X-Emby-Device-Name", "Cartographus")
req.Header.Set("X-Emby-Device-Id", "cartographus")
req.Header.Set("X-Emby-Client-Version", "1.0.0")
req.Header.Set("Accept", "application/json")
req.Header.Set("Content-Type", "application/json")
```

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `JELLYFIN_URL` | Jellyfin server URL | - |
| `JELLYFIN_API_KEY` | Jellyfin API key | - |
| `JELLYFIN_USER_ID` | Optional user ID | - |
| `JELLYFIN_SYNC_ENABLED` | Enable Jellyfin sync | `false` |
| `JELLYFIN_SYNC_INTERVAL` | Polling interval | `1m` |
| `EMBY_URL` | Emby server URL | - |
| `EMBY_API_KEY` | Emby API key | - |
| `EMBY_USER_ID` | Optional user ID | - |
| `EMBY_SYNC_ENABLED` | Enable Emby sync | `false` |
| `EMBY_SYNC_INTERVAL` | Polling interval | `1m` |

---

## Code References

| Component | File | Lines | Notes |
|-----------|------|-------|-------|
| Jellyfin Client | `internal/sync/jellyfin_client.go` | 462 | REST API client |
| Jellyfin Manager | `internal/sync/jellyfin_manager.go` | 282 | Sync orchestration |
| Jellyfin Session Poller | `internal/sync/jellyfin_session_poller.go` | 172 | Periodic polling |
| Jellyfin WebSocket | `internal/sync/jellyfin_websocket.go` | 369 | Real-time events |
| Jellyfin Circuit Breaker | `internal/sync/jellyfin_circuit_breaker.go` | 217 | Fault tolerance |
| Emby Client | `internal/sync/emby_client.go` | 463 | REST API client |
| Emby Manager | `internal/sync/emby_manager.go` | 282 | Sync orchestration |
| Emby Session Poller | `internal/sync/emby_session_poller.go` | 172 | Periodic polling |
| Emby WebSocket | `internal/sync/emby_websocket.go` | 369 | Real-time events |
| Emby Circuit Breaker | `internal/sync/emby_circuit_breaker.go` | 217 | Fault tolerance |
| Jellyfin Session Model | `internal/models/jellyfin_session.go` | - | Session data structures |
| Emby Session Model | `internal/models/emby_session.go` | - | Session data structures |

### Total Lines of Code

- **Jellyfin**: ~4,497 lines (source + tests)
- **Emby**: ~4,925 lines (source + tests)
- **Combined**: 9,422 lines across 20 files

---

## Verification

### Verified Claims

| Claim | Source | Verified |
|-------|--------|----------|
| Jellyfin uses `/socket` WebSocket path | `jellyfin_client.go:273` | Yes |
| Emby uses `/embywebsocket` path | `emby_client.go:274` | Yes |
| Both use `X-Emby-Token` header | `jellyfin_client.go:296`, `emby_client.go:297` | Yes |
| Source field set to "jellyfin" | `jellyfin_client.go:344` | Yes |
| Source field set to "emby" | `emby_client.go:345` | Yes |
| 30-second HTTP timeout | `jellyfin_client.go:79`, `emby_client.go:79` | Yes |
| Circuit breaker per server | `jellyfin_circuit_breaker.go`, `emby_circuit_breaker.go` | Yes |

### Test Coverage

- `internal/sync/jellyfin_client_test.go` - 902 lines
- `internal/sync/jellyfin_manager_test.go` - 636 lines
- `internal/sync/jellyfin_session_poller_test.go` - 465 lines
- `internal/sync/jellyfin_websocket_test.go` - 702 lines
- `internal/sync/jellyfin_circuit_breaker_test.go` - 290 lines
- `internal/sync/emby_client_test.go` - 902 lines
- `internal/sync/emby_manager_test.go` - 1,036 lines
- `internal/sync/emby_session_poller_test.go` - 465 lines
- `internal/sync/emby_websocket_test.go` - 693 lines
- `internal/sync/emby_circuit_breaker_test.go` - 326 lines

---

## Related ADRs

- [ADR-0008](0008-circuit-breaker-pattern.md): Circuit breaker pattern used by both clients
- [ADR-0009](0009-plex-direct-integration.md): Plex integration with similar patterns
- [ADR-0026](0026-multi-server-management-ui.md): UI for managing multiple media servers

---

## References

- [Jellyfin API Documentation](https://api.jellyfin.org/)
- [Emby API Documentation](https://dev.emby.media/doc/restapi/index.html)
- Internal: `internal/sync/doc.go` - Package documentation
