# Cartographus

**Data analytics and geographic visualization for self-hosted media servers.**

Your media server has been collecting data for years: every stream, every user session, every transcode decision. Cartographus transforms that data into actionable insights with 30+ analytics endpoints, 47+ interactive charts, geographic visualization, and real-time monitoring.

Built for self-hosters who want the same depth of analytics that streaming services use internally, without sending data to the cloud.

[![Build Status](https://github.com/tomtom215/cartographus/actions/workflows/build-and-test.yml/badge.svg)](https://github.com/tomtom215/cartographus/actions/workflows/build-and-test.yml)
[![License: AGPL-3.0](https://img.shields.io/badge/License-AGPL--3.0-blue.svg)](https://www.gnu.org/licenses/agpl-3.0)
[![GHCR](https://ghcr-badge.egpl.dev/tomtom215/cartographus/latest_tag?trim=major&label=GHCR)](https://github.com/tomtom215/cartographus/pkgs/container/cartographus)
[![Go Version](https://img.shields.io/github/go-mod-go-version/tomtom215/cartographus?label=Go)](https://github.com/tomtom215/cartographus/blob/main/go.mod)

---

<!-- TODO: Add screenshots here -->
<!-- ![Map View](docs/screenshots/map-view.png) -->
<!-- ![Analytics Dashboard](docs/screenshots/analytics.png) -->
<!-- ![3D Globe](docs/screenshots/globe.png) -->

---

## Why Cartographus?

**For users of Tautulli, Jellystat, or similar tools**: Cartographus builds on that foundation with geographic visualization, cross-server analytics, security detection, and deeper behavioral insights. Import your existing Tautulli database to preserve years of historical data.

**For new self-hosters**: Start fresh with a unified analytics platform that works directly with Plex, Jellyfin, and Emby from day one.

### What Sets It Apart

- **Geographic visualization** - See playback locations on an interactive WebGL map with clustering for 10,000+ points, or spin a 3D globe with user-to-server arcs and H3 hexagonal aggregation
- **Cross-server analytics** - Connect Plex, Jellyfin, and Emby simultaneously with automatic deduplication when users appear on multiple servers
- **Behavioral analytics** - Binge detection, watch party identification, cohort retention, content discovery patterns, and user engagement scoring
- **Quality of Experience metrics** - Buffer prediction, transcode efficiency, resolution mismatch detection, and hardware acceleration monitoring
- **Security detection** - Catch account sharing with impossible travel detection, concurrent stream limits, device velocity tracking, and geographic restrictions
- **Real-time monitoring** - Sub-second WebSocket updates for live sessions, transcoding status, and buffer health
- **Approximate analytics** - HyperLogLog distinct counts and KLL percentile sketches for O(1) queries on large datasets
- **Self-hosted and private** - Your data stays on your server. No cloud dependencies. No telemetry. AGPL-3.0 licensed.

---

## Quick Start (Docker)

Get running in under 2 minutes with a single media server:

```bash
# Create directory
mkdir cartographus && cd cartographus

# Create docker-compose.yml
cat > docker-compose.yml << 'EOF'
services:
  cartographus:
    image: ghcr.io/tomtom215/cartographus:latest
    container_name: cartographus
    ports:
      - "3857:3857"
    environment:
      - JWT_SECRET=replace_with_random_string_at_least_32_characters
      - ADMIN_USERNAME=admin
      - ADMIN_PASSWORD=YourSecurePassword123!
      # Plex
      - ENABLE_PLEX_SYNC=true
      - PLEX_URL=http://your-plex-server:32400
      - PLEX_TOKEN=your_plex_token
    volumes:
      - ./data:/data
    restart: unless-stopped
EOF

# Start
docker-compose up -d
```

Open `http://localhost:3857` and log in with your admin credentials.

**Want "Sign in with Plex" instead?** Replace the auth environment variables:

```yaml
environment:
  - AUTH_MODE=plex
  - PLEX_AUTH_CLIENT_ID=your-plex-app-client-id
  - PLEX_AUTH_REDIRECT_URI=http://localhost:3857/api/v1/auth/plex/callback
  # Plex data source
  - ENABLE_PLEX_SYNC=true
  - PLEX_URL=http://your-plex-server:32400
  - PLEX_TOKEN=your_plex_token
```

Server owners automatically get admin access. See [Plex Authentication](#plex-authentication-sign-in-with-plex) for details.

**Using Jellyfin or Emby instead?** See [Media Server Configuration](#media-server-configuration) below.

---

## How It Works

```
     Your Media Servers                    Cartographus                       You See
  ════════════════════════            ═══════════════════════            ════════════════

  ┌──────────────────────┐           ┌───────────────────────┐          ┌──────────────┐
  │  Plex Server(s)      │══════════▶│                       │          │   2D Maps    │
  │  • Real-time streams │ WebSocket │   ┌───────────────┐   │          │   (WebGL)    │
  │  • Transcoding info  │           │   │  Unified      │   │─────────▶│  10,000+ pts │
  └──────────────────────┘           │   │  Analytics    │   │          └──────────────┘
                                     │   │  Engine       │   │
  ┌──────────────────────┐           │   └───────┬───────┘   │          ┌──────────────┐
  │  Jellyfin Server(s)  │══════════▶│           │           │          │   3D Globe   │
  │  • Real-time streams │ WebSocket │   ┌───────▼───────┐   │─────────▶│   (deck.gl)  │
  │  • Session data      │           │   │  Cross-Source │   │          │  Arcs & Hex  │
  └──────────────────────┘           │   │  Deduplication│   │          └──────────────┘
                                     │   └───────┬───────┘   │
  ┌──────────────────────┐           │           │           │          ┌──────────────┐
  │  Emby Server(s)      │══════════▶│   ┌───────▼───────┐   │          │   Analytics  │
  │  • Real-time streams │ WebSocket │   │  Detection    │   │─────────▶│   47+ Charts │
  │  • Session data      │           │   │  Engine       │   │          │   6 Pages    │
  └──────────────────────┘           │   └───────────────┘   │          └──────────────┘
                                     │                       │
  ┌ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─┐           │   • IP Geolocation    │          ┌──────────────┐
  │  Tautulli (optional) │─ ─ ─ ─ ─ ▶│   • Security Alerts   │─────────▶│  Live Stream │
  │  • Historical import │  REST API │   • User Insights     │          │  Monitoring  │
  └ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─┘           │   • Bandwidth Stats   │          └──────────────┘
                                     └───────────────────────┘
```

**Connect any combination of media servers** - Cartographus captures real-time playback events, geolocates by IP address, and visualizes everything on interactive maps.

---

## Features

### Analytics Engine

30+ dedicated analytics endpoints covering:

| Category | Metrics |
|----------|---------|
| **User Behavior** | Binge detection, watch parties, cohort retention, engagement scoring, pause patterns, abandonment rates |
| **Content** | Popular titles, library distribution, content discovery latency, time-to-first-watch |
| **Performance** | Transcode efficiency, hardware acceleration usage, resolution mismatch, bitrate distribution, buffer health |
| **Quality of Experience** | QoE scoring, connection security, codec compatibility, HDR/audio/subtitle analytics |
| **Comparative** | Period-over-period trends, user network relationships, device migration tracking |

47+ interactive charts across 6 themed dashboard pages powered by ECharts.

### Geographic Visualization

| Feature | Description |
|---------|-------------|
| **WebGL Map** | Smart clustering, color-coded markers, detailed popups - handles 10,000+ locations smoothly |
| **3D Globe** | Scatterplot points, H3 hexagonal aggregation, animated user-to-server arcs |
| **Temporal Heatmap** | Watch playback activity spread across geography over time |
| **Spatial Queries** | Viewport filtering, nearby search, density analysis |

### Real-Time Monitoring

- **Live Activity Dashboard** - Active sessions with transcoding status, progress, and quality
- **Buffer Health Tracking** - Predictive detection warns 10-15 seconds before buffering occurs
- **Hardware Transcode Monitoring** - GPU acceleration usage and quality transitions
- **WebSocket Updates** - Sub-second refresh from all connected media servers

### Security Detection

7 detection rules for account sharing and suspicious activity:

| Rule | Example |
|------|---------|
| **Impossible Travel** | User streams from NYC, then London 5 minutes later |
| **Concurrent Streams** | Same user watching 4 streams simultaneously |
| **Device Velocity** | Same device appears from multiple IPs rapidly |
| **Geo Restriction** | Block or allow streaming by country |
| **Simultaneous Locations** | Active streams from distant locations at once |
| **User Agent Anomaly** | Unusual or spoofed client software |
| **VPN Usage** | Streaming through known VPN services |

Configurable alerts via Discord webhooks or HTTP endpoints.

### Data Management

- **16 Filter Dimensions** - Date range, users, platforms, codecs, libraries, resolutions, transcode decisions, and more
- **Fuzzy Search** - RapidFuzz-powered title matching with typo tolerance
- **Approximate Analytics** - HyperLogLog distinct counts and KLL percentiles for O(1) queries on large datasets
- **Cursor Pagination** - Consistent O(1) performance across millions of records
- **Data Export** - CSV, GeoJSON, GeoParquet formats
- **Tautulli Import** - Migrate historical data from existing Tautulli SQLite databases with real-time progress tracking
- **Plex Historical Sync** - Sync playback history directly from Plex servers with configurable date ranges
- **Data Sync UI** - Web interface for initiating imports and monitoring progress with WebSocket updates
- **Full Backup/Restore** - Scheduled backups with configurable retention policies

---

## Media Server Configuration

Cartographus connects **directly** to your media servers. No middleware required.

| Server | Integration | Real-Time | Features |
|--------|-------------|-----------|----------|
| **Plex** | Direct + Webhooks | WebSocket | Sessions, transcode, library sync |
| **Jellyfin** | Direct | WebSocket | Sessions, cross-platform linking |
| **Emby** | Direct | WebSocket | Sessions, cross-platform linking |
| **Tautulli** | Optional | REST API | Historical data import only |

### Plex

```yaml
environment:
  - ENABLE_PLEX_SYNC=true
  - PLEX_URL=http://plex:32400
  - PLEX_TOKEN=your_plex_token
  - ENABLE_PLEX_REALTIME=true
```

### Jellyfin

```yaml
environment:
  - JELLYFIN_ENABLED=true
  - JELLYFIN_URL=http://jellyfin:8096
  - JELLYFIN_API_KEY=your_jellyfin_api_key
  - JELLYFIN_REALTIME_ENABLED=true
```

### Emby

```yaml
environment:
  - EMBY_ENABLED=true
  - EMBY_URL=http://emby:8096
  - EMBY_API_KEY=your_emby_api_key
  - EMBY_REALTIME_ENABLED=true
```

### Multiple Servers

For enterprise setups with multiple servers of the same type:

```yaml
environment:
  - PLEX_SERVERS=[{"ServerID":"main","URL":"http://plex1:32400","Token":"xxx"},{"ServerID":"backup","URL":"http://plex2:32400","Token":"yyy"}]
```

See [.env.example](.env.example) for all configuration options.

---

## Configuration Reference

### Required

| Variable | Description |
|----------|-------------|
| `JWT_SECRET` | 32+ character secret. Generate with: `openssl rand -base64 48` |
| `ADMIN_USERNAME` | Admin login username |
| `ADMIN_PASSWORD` | Admin password (12+ chars, mixed case, number, special char) |

Plus at least one media server configured.

### Optional

| Variable | Default | Description |
|----------|---------|-------------|
| `HTTP_PORT` | `3857` | Server port (3857 = EPSG:3857 Web Mercator) |
| `DUCKDB_PATH` | `/data/cartographus.duckdb` | Database file location |
| `AUTH_MODE` | `jwt` | Auth mode: `jwt`, `basic`, `oidc`, `plex` (with server detection), `multi`, or `none` |
| `SYNC_INTERVAL` | `5m` | How often to sync with media servers |
| `LOG_LEVEL` | `info` | Logging: `debug`, `info`, `warn`, `error` |
| `DETECTION_ENABLED` | `true` | Enable security anomaly detection |

### OIDC Authentication (Enterprise)

Connect to your existing identity provider (Authelia, Authentik, Keycloak, Okta, etc.):

| Variable | Required | Description |
|----------|----------|-------------|
| `AUTH_MODE` | Yes | Set to `oidc` or `multi` |
| `OIDC_ENABLED` | Yes | `true` to enable OIDC |
| `OIDC_ISSUER_URL` | Yes | Your IdP issuer URL (e.g., `https://auth.example.com`) |
| `OIDC_CLIENT_ID` | Yes | OAuth2 client ID |
| `OIDC_CLIENT_SECRET` | Yes | OAuth2 client secret |
| `OIDC_REDIRECT_URL` | Yes | Callback URL (e.g., `https://cartographus.example.com/api/auth/oidc/callback`) |
| `OIDC_SCOPES` | No | Scopes (default: `openid profile email`) |
| `OIDC_PKCE_ENABLED` | No | PKCE support (default: `true`, recommended) |

Example OIDC configuration:

```yaml
environment:
  - AUTH_MODE=oidc
  - OIDC_ENABLED=true
  - OIDC_ISSUER_URL=https://auth.example.com
  - OIDC_CLIENT_ID=cartographus
  - OIDC_CLIENT_SECRET=your-secret-here
  - OIDC_REDIRECT_URL=https://cartographus.example.com/api/auth/oidc/callback
```

Uses [Zitadel OIDC](https://github.com/zitadel/oidc) - an OpenID Foundation certified library with PKCE, nonce validation, and back-channel logout support.

### Plex Authentication (Sign in with Plex)

Enable "Sign in with Plex" for automatic role assignment based on Plex server ownership:

```yaml
environment:
  - AUTH_MODE=plex
  - PLEX_AUTH_CLIENT_ID=your-plex-app-client-id
  - PLEX_AUTH_REDIRECT_URI=http://localhost:3857/api/v1/auth/plex/callback
```

Unlike traditional OAuth flows, Cartographus uses Plex's PIN-based authentication (same approach as [Overseerr](https://github.com/sct/overseerr)). This means it works with:
- Local IP addresses (`http://192.168.1.100:3857`)
- Localhost (`http://localhost:3857`)
- Private networks without public DNS

**How it works:**

| User Type | Automatic Role | Data Access |
|-----------|----------------|-------------|
| Plex server owner | `admin` | All users' data |
| Shared user | `viewer` | Own data only |

When a Plex server owner clicks "Sign in with Plex":
1. A popup opens to plex.tv for authentication
2. User approves access on Plex's site
3. Cartographus detects server ownership and assigns the appropriate role

**Configuration options:**

| Variable | Default | Description |
|----------|---------|-------------|
| `PLEX_AUTH_CLIENT_ID` | *required* | Your Plex application client ID |
| `PLEX_AUTH_REDIRECT_URI` | *required* | OAuth callback URL (e.g., `http://localhost:3857/api/v1/auth/plex/callback`) |
| `PLEX_AUTH_ENABLE_SERVER_DETECTION` | `true` | Enable automatic server ownership detection |
| `PLEX_AUTH_SERVER_OWNER_ROLE` | `admin` | Role for server owners |
| `PLEX_AUTH_SERVER_ADMIN_ROLE` | `editor` | Role for shared server admins |
| `PLEX_AUTH_SERVER_MACHINE_ID` | - | Limit to specific server (optional) |

This makes onboarding seamless: deploy Cartographus anywhere, share the URL with your Plex users, and everyone can sign in with their existing Plex credentials with appropriate access levels automatically enforced.

### Event Processing

| Variable | Default | Description |
|----------|---------|-------------|
| `NATS_ENABLED` | `true` | Enable NATS JetStream for event processing |
| `WAL_ENABLED` | `true` | Enable BadgerDB Write-Ahead Log for durability |

---

## Deployment

### Reverse Proxy (Nginx)

```nginx
server {
    listen 443 ssl http2;
    server_name cartographus.example.com;

    location / {
        proxy_pass http://localhost:3857;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

### Kubernetes

Experimental Kubernetes support available. See [deploy/kubernetes/README.md](deploy/kubernetes/README.md).

---

## Technology Stack

| Layer | Technology | Purpose |
|-------|------------|---------|
| **Backend** | Go 1.24+ | High-performance, single binary |
| **Database** | DuckDB 1.4.3 | Analytics-optimized with spatial, H3, INET extensions |
| **Event Processing** | NATS JetStream + Watermill | Reliable async messaging with exactly-once delivery |
| **Durability** | BadgerDB WAL | Crash-safe event persistence |
| **Process Supervision** | Suture 4.0.6 | Erlang-style supervisor trees with automatic restart |
| **HTTP Router** | Chi 5.2.3 | Lightweight router with middleware composition |
| **Validation** | go-playground/validator | Request validation with struct tags |
| **Circuit Breaker** | sony/gobreaker | Resilient external service calls |
| **Authentication** | Zitadel OIDC v3.45.1 | OpenID Foundation certified OIDC |
| **Authorization** | Casbin | RBAC policy engine |
| **Frontend** | TypeScript 5.9.3 | Strict type-safe UI code |
| **Maps** | MapLibre GL JS 5.15.0 | WebGL vector tile rendering |
| **3D Globe** | deck.gl 9.2.5 | Large-scale WebGL data visualization |
| **Charts** | ECharts 6.0.0 | Interactive data visualization |
| **Map Tiles** | PMTiles 4.3.2 | Self-hosted vector tiles (no external tile server) |
| **Build** | esbuild 0.27.2 | Fast TypeScript bundling |
| **Container** | Docker (amd64, arm64) | Multi-architecture images |

---

## For Developers

### Building from Source

```bash
git clone https://github.com/tomtom215/cartographus.git && cd map
export GOTOOLCHAIN=local
make build
./cartographus
```

### API

302 REST endpoints organized by domain. See [docs/API-REFERENCE.md](docs/API-REFERENCE.md).

Key endpoints:
- `GET /api/v1/health` - Health check
- `GET /api/v1/stats` - Summary statistics
- `GET /api/v1/locations` - Playback locations with geo data
- `GET /api/v1/analytics/*` - 30+ analytics endpoints
- `GET /api/v1/spatial/hexagons` - H3 hexagon aggregation

### Architecture

See [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for system design and [docs/adr/](docs/adr/) for 29 Architecture Decision Records.

---

## Troubleshooting

| Issue | Solution |
|-------|----------|
| Database locked | Ensure only one instance is running |
| JWT secret error | Generate 32+ char secret: `openssl rand -base64 48` |
| Connection refused | Verify media server URL and credentials |
| Maps not loading | Check WebGL support in browser |
| No playback data | Check `/api/v1/health` - verify sync is running |

See [docs/TROUBLESHOOTING.md](docs/TROUBLESHOOTING.md) for more.

---

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Write tests (TDD required)
4. Run checks: `gofmt -s -w . && go vet ./... && go test -race ./...`
5. Commit with conventional format (`feat: add amazing feature`)
6. Open a Pull Request

---

## License

AGPL-3.0 License - see [LICENSE](LICENSE).

This means you can freely use, modify, and distribute this software, but if you run a modified version as a network service, you must make the source code available to users of that service.

---

## Support

- **Issues**: https://github.com/tomtom215/cartographus/issues
- **Discussions**: https://github.com/tomtom215/cartographus/discussions

---

## Acknowledgments

- [DuckDB](https://duckdb.org/) - Analytics database with spatial extensions
- [MapLibre GL JS](https://maplibre.org/) - Open-source WebGL maps
- [deck.gl](https://deck.gl/) - Large-scale WebGL visualizations
- [ECharts](https://echarts.apache.org/) - Interactive charts
- [NATS](https://nats.io/) - Cloud-native messaging
- [Zitadel OIDC](https://github.com/zitadel/oidc) - OpenID Foundation certified OIDC library
- [Casbin](https://casbin.org/) - Authorization library with RBAC support
- [Tautulli](https://github.com/Tautulli/Tautulli) - Plex monitoring inspiration
