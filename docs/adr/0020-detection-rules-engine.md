# ADR-0020: Detection Rules Engine for Media Playback Security

**Date**: 2025-12-16
**Status**: Accepted

---

## Context

Cartographus aggregates playback data from Plex media servers, which creates an opportunity to detect anomalous access patterns that may indicate unauthorized account sharing, credential compromise, or other security concerns. Currently, users have no visibility into suspicious activity patterns.

Key use cases requiring detection capabilities:

1. **Account Sharing Detection**: Multiple simultaneous streams from distant locations
2. **Credential Theft Detection**: Impossible travel patterns (NYC to London in 30 minutes)
3. **Device Compromise**: Single device appearing on many IPs rapidly
4. **Geographic Restrictions**: Block access from specific countries
5. **Stream Limit Enforcement**: Enforce per-user concurrent stream limits

### Alternatives Considered

| Approach | Pros | Cons |
|----------|------|------|
| **External SIEM** | Full-featured, centralized | Requires additional infrastructure, cost |
| **Plex Built-in** | Native integration | Limited to Plex features, no customization |
| **Manual Monitoring** | No development | Doesn't scale, error-prone |
| **Custom Detection Engine** | Tailored to media patterns, integrated | Development effort |

---

## Decision

Implement a **custom detection rules engine** as an internal package (`internal/detection`) that:

1. Processes playback events in real-time
2. Evaluates configurable detection rules
3. Generates alerts with severity levels
4. Tracks user trust scores
5. Integrates with WebSocket for real-time notifications
6. Supports Discord/Webhook notifications

### Architecture

```
                    Detection Engine Architecture
                    ════════════════════════════

┌─────────────────────────────────────────────────────────────────────────────┐
│                           Detection Engine                                   │
│  ┌────────────────────────────────────────────────────────────────────────┐ │
│  │                         Detection Event                                 │ │
│  │  (UserID, IP, Lat/Lon, Timestamp, Device, Session, Platform, Player)   │ │
│  └────────────────────────────────────────────────────────────────────────┘ │
│                                    │                                         │
│    ┌───────────┬───────────┬───────┴───────┬───────────┬───────────┐        │
│    ▼           ▼           ▼               ▼           ▼           ▼        │
│ ┌────────┐ ┌────────┐ ┌────────┐     ┌────────┐ ┌────────┐ ┌────────┐      │
│ │Impossi-│ │Concur- │ │Device  │     │Geo     │ │Simulta-│ │User    │      │
│ │ble     │ │rent    │ │Velocity│     │Restric-│ │neous   │ │Agent   │      │
│ │Travel  │ │Streams │ │        │     │tion    │ │Location│ │Anomaly │      │
│ └────────┘ └────────┘ └────────┘     └────────┘ └────────┘ └────────┘      │
│                                                              ┌────────┐      │
│                                                              │VPN     │      │
│                                                              │Usage   │      │
│                                                              └────────┘      │
│    └───────────┴───────────┴───────────────┴───────────┴───────────┘        │
│                                    │                                         │
│                         ┌──────────────────┐                                 │
│                         │   Alert Store    │                                 │
│                         │   (DuckDB)       │                                 │
│                         └──────────────────┘                                 │
│                                    │                                         │
│              ┌─────────────────────┼─────────────────────┐                  │
│              ▼                     ▼                     ▼                  │
│       ┌──────────────┐      ┌──────────────┐     ┌──────────────┐          │
│       │ WebSocket    │      │ Discord      │     │ Webhook      │          │
│       │ Broadcast    │      │ Notifier     │     │ Notifier     │          │
│       └──────────────┘      └──────────────┘     └──────────────┘          │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Detection Rules

| Rule | Description | Default Config |
|------|-------------|----------------|
| **Impossible Travel** | Flags when user appears in two distant locations faster than possible | max_speed: 900 km/h, min_distance: 100 km |
| **Concurrent Streams** | Limits simultaneous streams per user | default_limit: 3 |
| **Device Velocity** | Detects devices on many IPs quickly | window: 5 min, max_ips: 3 |
| **Geo Restriction** | Block/allow by country | blocklist or allowlist mode (disabled by default) |
| **Simultaneous Locations** | Detects active streams from distant locations | min_distance: 50 km, window: 30 min |
| **User Agent Anomaly** | Detects suspicious user agent patterns and platform switches | window: 30 min, suspicious patterns: curl, bot, etc. |
| **VPN Usage** | Detects streaming from known VPN IP addresses | alert on first use and new provider |

### Key Components

**Engine** (`engine.go`):
- Coordinates all detection rules
- Manages detector registration
- Processes events through all enabled detectors
- Handles alert persistence and notification

**Detectors** (implement `Detector` interface):
- `ImpossibleTravelDetector`: Haversine distance + time analysis
- `ConcurrentStreamsDetector`: Active session counting
- `DeviceVelocityDetector`: IP change frequency tracking
- `GeoRestrictionDetector`: Country blocklist/allowlist
- `SimultaneousLocationsDetector`: Active stream location comparison
- `UserAgentAnomalyDetector`: Platform switches and suspicious patterns
- `VPNUsageDetector`: VPN IP detection via lookup service

**Store** (`store.go`):
- `DuckDBStore`: Implements AlertStore, RuleStore, TrustStore, EventHistory
- Creates tables: `detection_rules`, `detection_alerts`, `user_trust_scores`

**Notifiers**:
- `DiscordNotifier`: Webhook-based Discord alerts with embeds
- `WebhookNotifier`: Generic HTTP webhook notifications

**Handler** (`handler.go`):
- `WatermillHandler`: Integrates with NATS event stream
- Processes playback events through detection engine

---

## Consequences

### Positive

- **Real-time Detection**: Alerts generated immediately on suspicious activity
- **Configurable Rules**: Each rule can be tuned or disabled independently
- **Trust Scoring**: Users accumulate trust score impacts from violations
- **Multi-channel Alerts**: WebSocket, Discord, and webhook notifications
- **Integrated Storage**: Uses existing DuckDB infrastructure
- **Supervisor Managed**: Detection service supervised for reliability

### Negative

- **Additional Load**: Event processing adds CPU overhead
- **False Positives**: Travel detection may flag legitimate travelers
- **Maintenance**: Rules need tuning over time

### Neutral

- **NATS Integration**: Uses Watermill handler (requires `-tags nats`)
- **Build Tags**: Detection code compiles without NATS but handler is a stub
- **Frontend Required**: Alerts panel and rules config need UI integration

---

## Implementation

### File Structure

```
internal/detection/
├── doc.go                          # Package documentation
├── types.go                        # Core types and interfaces
├── types_test.go                   # Type tests
├── engine.go                       # Detection engine coordinator
├── engine_test.go                  # Engine tests
├── impossible_travel.go            # Impossible travel detector
├── impossible_travel_test.go       # Impossible travel tests
├── concurrent_streams.go           # Concurrent streams detector
├── concurrent_streams_test.go      # Concurrent streams tests
├── device_velocity.go              # Device velocity detector
├── device_velocity_test.go         # Device velocity tests
├── geo_restriction.go              # Geographic restriction detector
├── geo_restriction_test.go         # Geo restriction tests
├── simultaneous_locations.go       # Simultaneous locations detector
├── simultaneous_locations_test.go  # Simultaneous locations tests
├── user_agent_anomaly.go           # User agent anomaly detector
├── user_agent_anomaly_test.go      # User agent anomaly tests
├── vpn_usage.go                    # VPN usage detector
├── vpn_usage_test.go               # VPN usage tests
├── store.go                        # DuckDB storage implementation
├── store_test.go                   # Store tests
├── event_history.go                # Event history interface
├── cached_event_history.go         # Cached event history implementation
├── cached_event_history_test.go    # Cached event history tests
├── notifier_discord.go             # Discord webhook notifier
├── notifier_discord_test.go        # Discord notifier tests
├── notifier_webhook.go             # Generic webhook notifier
├── notifier_webhook_test.go        # Webhook notifier tests
├── handler.go                      # Watermill handler (NATS build tag)
└── handler_stub.go                 # Non-NATS stub (Topic functions only)

internal/supervisor/services/
├── detection_service.go            # Suture service wrapper
└── detection_service_test.go       # Detection service tests

internal/api/
├── handlers_detection.go           # HTTP API handlers
├── handlers_detection_test.go      # Handler tests
└── chi_router.go                   # registerChiDetectionRoutes()

web/src/
├── app/SecurityAlertsManager.ts    # Alerts display UI
├── app/DetectionRulesManager.ts    # Rules configuration UI
├── app/AnomalyDetectionManager.ts  # Anomaly detection UI
├── lib/types/detection.ts          # TypeScript types
└── lib/api.ts                      # API methods
```

### API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/detection/alerts` | List alerts with filtering |
| GET | `/api/v1/detection/alerts/{id}` | Get single alert |
| POST | `/api/v1/detection/alerts/{id}/acknowledge` | Acknowledge alert |
| GET | `/api/v1/detection/rules` | List all rules |
| GET | `/api/v1/detection/rules/{type}` | Get single rule |
| PUT | `/api/v1/detection/rules/{type}` | Update rule config |
| POST | `/api/v1/detection/rules/{type}/enable` | Enable/disable rule |
| GET | `/api/v1/detection/users/{id}/trust` | Get user trust score |
| GET | `/api/v1/detection/users/low-trust` | List low trust users |
| GET | `/api/v1/detection/metrics` | Engine metrics |
| GET | `/api/v1/detection/stats` | Alert statistics |

### Usage Example

```go
// In main.go
store := detection.NewDuckDBStore(db.Conn())
store.InitSchema(ctx)

// NewEngine(alertStore, trustStore, eventHistory, broadcaster)
engine := detection.NewEngine(store, store, store, wsHub)

// Register core detectors (use store as EventHistory)
engine.RegisterDetector(detection.NewImpossibleTravelDetector(store))
engine.RegisterDetector(detection.NewConcurrentStreamsDetector(store))
engine.RegisterDetector(detection.NewDeviceVelocityDetector(store))
engine.RegisterDetector(detection.NewSimultaneousLocationsDetector(store))
engine.RegisterDetector(detection.NewGeoRestrictionDetector(store))
engine.RegisterDetector(detection.NewUserAgentAnomalyDetector(store))

// VPN detector requires a VPN lookup service
if vpnService != nil {
    engine.RegisterDetector(detection.NewVPNUsageDetector(vpnService))
}

// Add to supervisor
tree.AddMessagingService(services.NewDetectionService(engine))
```

---

## Verification

### Verified Claims

| Claim | Source | Verified |
|-------|--------|----------|
| Haversine distance calculation | `impossible_travel.go:haversineDistance()` | Yes |
| DuckDB storage | `store.go:DuckDBStore` | Yes |
| WebSocket integration | `engine.go:broadcast()` | Yes |
| Discord notifications | `notifier_discord.go:DiscordNotifier` | Yes |
| Suture supervision | `detection_service.go:DetectionService` | Yes |
| 7 detection rules | `types.go` + `user_agent_anomaly.go` + `vpn_usage.go` | Yes |
| Trust score management | `store.go:TrustStore` interface | Yes |
| NATS integration | `handler.go:WatermillHandler` | Yes |

### Test Coverage

- `engine_test.go`: Engine lifecycle and detection flow
- `impossible_travel_test.go`: Distance calculations and travel detection
- `concurrent_streams_test.go`: Stream counting and limits
- `device_velocity_test.go`: IP velocity tracking
- `geo_restriction_test.go`: Country blocklist/allowlist
- `simultaneous_locations_test.go`: Concurrent location detection
- `user_agent_anomaly_test.go`: User agent pattern detection
- `vpn_usage_test.go`: VPN IP detection
- `store_test.go`: DuckDB storage operations
- `notifier_discord_test.go`: Discord webhook notifications
- `notifier_webhook_test.go`: Generic webhook notifications

Run tests:
```bash
go test -tags "wal,nats" -v ./internal/detection/...
```

---

## Related ADRs

- [ADR-0005](0005-nats-jetstream-event-processing.md): NATS JetStream for event delivery
- [ADR-0017](0017-watermill-router-and-middleware.md): Watermill handler integration
- [ADR-0004](0004-process-supervision-with-suture.md): Suture supervision for detection service

---

## References

- [Haversine Formula](https://en.wikipedia.org/wiki/Haversine_formula) - Geographic distance calculation
- [Discord Webhook API](https://discord.com/developers/docs/resources/webhook) - Notification integration
