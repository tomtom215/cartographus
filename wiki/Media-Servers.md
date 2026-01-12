# Media Server Configuration

Connect Cartographus to your media servers for real-time analytics and historical data.

**[Home](Home)** | **[Configuration](Configuration)** | **Media Servers** | **[Authentication](Authentication)**

---

## Overview

Cartographus connects **directly** to your media servers using their native APIs. No middleware or additional software is required.

| Server | Data Collection | Real-Time | Webhooks |
|--------|-----------------|-----------|----------|
| **Plex** | Direct API + WebSocket | Yes | Yes |
| **Jellyfin** | Direct API + WebSocket | Yes | Yes |
| **Emby** | Direct API + WebSocket | Yes | Yes |
| **Tautulli** | REST API (import only) | No | No |

You can connect **multiple servers simultaneously** - Cartographus automatically deduplicates users who appear on more than one server.

---

## Plex Configuration

### Getting Your Plex Token

Your Plex token authenticates Cartographus with your Plex server.

#### Method 1: Browser (Easiest)

1. Sign in to [Plex Web](https://app.plex.tv)
2. Navigate to any media item
3. Click the **...** menu and select **Get Info**
4. Click **View XML** in the popup
5. Look at the URL - find `X-Plex-Token=` followed by your token

#### Method 2: Command Line

```bash
curl -s "https://plex.tv/api/v2/users/signin" \
  -X POST \
  -H "X-Plex-Client-Identifier: cartographus" \
  -d "login=YOUR_USERNAME&password=YOUR_PASSWORD" | \
  grep -o '"authToken":"[^"]*"' | cut -d'"' -f4
```

### Basic Plex Setup

```yaml
environment:
  - ENABLE_PLEX_SYNC=true
  - PLEX_URL=http://plex:32400
  - PLEX_TOKEN=your_plex_token
  - ENABLE_PLEX_REALTIME=true
```

### Full Plex Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `ENABLE_PLEX_SYNC` | `false` | Enable Plex integration |
| `PLEX_URL` | - | Server URL (e.g., `http://plex:32400`) |
| `PLEX_TOKEN` | - | Authentication token |
| `PLEX_SERVER_ID` | `default` | Unique ID for multi-server setups |
| `ENABLE_PLEX_REALTIME` | `false` | WebSocket real-time updates |
| `PLEX_HISTORICAL_SYNC` | `false` | One-time historical backfill |
| `PLEX_SYNC_DAYS_BACK` | `365` | Days of history to sync (7-3650) |

### Plex Webhooks (Optional)

For additional real-time events, configure Plex webhooks:

1. In Plex, go to **Settings** > **Webhooks**
2. Add: `http://cartographus:3857/api/v1/webhooks/plex`
3. Enable relevant events

```yaml
environment:
  - ENABLE_PLEX_WEBHOOKS=true
  - PLEX_WEBHOOK_SECRET=your_webhook_secret  # Optional HMAC verification
```

### Transcode Monitoring

Track hardware acceleration and transcode efficiency:

```yaml
environment:
  - ENABLE_PLEX_TRANSCODE_MONITORING=true
  - PLEX_TRANSCODE_MONITORING_INTERVAL=10s
```

### Buffer Health Monitoring

Predictive buffering detection warns before playback issues occur:

```yaml
environment:
  - ENABLE_BUFFER_HEALTH_MONITORING=true
  - BUFFER_HEALTH_POLL_INTERVAL=5s
  - BUFFER_HEALTH_CRITICAL_THRESHOLD=20.0
  - BUFFER_HEALTH_RISKY_THRESHOLD=50.0
```

---

## Jellyfin Configuration

### Getting Your Jellyfin API Key

1. Open Jellyfin Dashboard
2. Go to **Administration** > **API Keys** (under Advanced)
3. Click the **+** button
4. Enter "Cartographus" as the name
5. Copy the generated key

### Basic Jellyfin Setup

```yaml
environment:
  - JELLYFIN_ENABLED=true
  - JELLYFIN_URL=http://jellyfin:8096
  - JELLYFIN_API_KEY=your_jellyfin_api_key
  - JELLYFIN_REALTIME_ENABLED=true
```

### Full Jellyfin Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `JELLYFIN_ENABLED` | `false` | Enable Jellyfin integration |
| `JELLYFIN_URL` | - | Server URL (e.g., `http://jellyfin:8096`) |
| `JELLYFIN_API_KEY` | - | API key |
| `JELLYFIN_SERVER_ID` | `default` | Unique ID for multi-server setups |
| `JELLYFIN_REALTIME_ENABLED` | `false` | WebSocket real-time updates |
| `JELLYFIN_USER_ID` | - | Optional: Scope to specific user |
| `JELLYFIN_SESSION_POLLING_ENABLED` | `false` | Backup polling if WebSocket fails |
| `JELLYFIN_SESSION_POLLING_INTERVAL` | `30s` | Polling interval |

### Jellyfin Webhooks (Optional)

1. Install the Jellyfin Webhooks plugin
2. Add a new webhook pointing to: `http://cartographus:3857/api/v1/webhooks/jellyfin`
3. Select events to monitor

```yaml
environment:
  - JELLYFIN_WEBHOOKS_ENABLED=true
  - JELLYFIN_WEBHOOK_SECRET=your_secret  # Optional verification
```

---

## Emby Configuration

### Getting Your Emby API Key

1. Open Emby Dashboard
2. Go to **Settings** > **API Keys**
3. Click **New API Key**
4. Enter "Cartographus" as the name
5. Copy the generated key

### Basic Emby Setup

```yaml
environment:
  - EMBY_ENABLED=true
  - EMBY_URL=http://emby:8096
  - EMBY_API_KEY=your_emby_api_key
  - EMBY_REALTIME_ENABLED=true
```

### Full Emby Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `EMBY_ENABLED` | `false` | Enable Emby integration |
| `EMBY_URL` | - | Server URL (e.g., `http://emby:8096`) |
| `EMBY_API_KEY` | - | API key |
| `EMBY_SERVER_ID` | `default` | Unique ID for multi-server setups |
| `EMBY_REALTIME_ENABLED` | `false` | WebSocket real-time updates |
| `EMBY_USER_ID` | - | Optional: Scope to specific user |

---

## Tautulli Configuration (Optional)

Tautulli integration is for **historical data import only**. If you're connecting directly to Plex, you don't need Tautulli.

### Use Cases

- Import years of historical playback data from existing Tautulli installation
- Access Tautulli's pre-calculated analytics

### Getting Your Tautulli API Key

1. Open Tautulli
2. Go to **Settings** > **Web Interface**
3. Find **API Key** and copy it

### Tautulli Setup

```yaml
environment:
  - TAUTULLI_ENABLED=true
  - TAUTULLI_URL=http://tautulli:8181
  - TAUTULLI_API_KEY=your_tautulli_api_key
```

### Importing Historical Data

To import your Tautulli database:

1. Access the Data Sync UI at `http://cartographus:3857/sync`
2. Select "Tautulli Import"
3. Choose date range
4. Monitor progress via WebSocket updates

---

## Multi-Server Setup

Connect multiple servers of the same type using unique server IDs.

### Environment Variables (Single Server Each)

```yaml
environment:
  # Plex servers
  - ENABLE_PLEX_SYNC=true
  - PLEX_URL=http://plex-main:32400
  - PLEX_TOKEN=token1
  - PLEX_SERVER_ID=plex-main

  # Jellyfin servers
  - JELLYFIN_ENABLED=true
  - JELLYFIN_URL=http://jellyfin:8096
  - JELLYFIN_API_KEY=key1
  - JELLYFIN_SERVER_ID=jellyfin-home
```

### YAML Configuration (Multiple of Same Type)

For multiple servers of the same type, use YAML configuration:

```yaml
# config.yaml
plex_servers:
  - enabled: true
    server_id: plex-main
    url: http://plex-main:32400
    token: token1
    realtime_enabled: true

  - enabled: true
    server_id: plex-backup
    url: http://plex-backup:32400
    token: token2
    realtime_enabled: true

jellyfin_servers:
  - enabled: true
    server_id: jellyfin-home
    url: http://jellyfin:8096
    api_key: key1
    realtime_enabled: true

  - enabled: true
    server_id: jellyfin-office
    url: http://192.168.2.100:8096
    api_key: key2
    realtime_enabled: true
```

### Cross-Server Deduplication

Cartographus automatically identifies users across servers by matching:
- Email addresses
- Usernames
- Plex user IDs

This ensures a user watching on both Plex and Jellyfin appears once in analytics.

---

## Network Configuration

### Docker Networking

When running in Docker, use container names for internal communication:

```yaml
services:
  cartographus:
    environment:
      - PLEX_URL=http://plex:32400  # Container name, not localhost
    networks:
      - media

  plex:
    networks:
      - media

networks:
  media:
```

### External Servers

For servers outside Docker, use the actual IP or hostname:

```yaml
environment:
  - PLEX_URL=http://192.168.1.100:32400
  - JELLYFIN_URL=http://media.local:8096
```

### Firewall Considerations

Ensure these ports are accessible from Cartographus:

| Server | Port | Protocol |
|--------|------|----------|
| Plex | 32400 | HTTP/HTTPS |
| Jellyfin | 8096 | HTTP |
| Emby | 8096 | HTTP |
| Tautulli | 8181 | HTTP |

---

## Verification

After configuration, verify connections:

### Health Check

```bash
curl http://localhost:3857/api/v1/health
```

### Sync Status

```bash
curl http://localhost:3857/api/v1/sync/status
```

### Server List

```bash
curl http://localhost:3857/api/v1/servers
```

---

## Troubleshooting

| Issue | Solution |
|-------|----------|
| "Connection refused" | Verify URL and port. Check firewall rules. |
| "Unauthorized" | Regenerate token/API key. Verify it's not expired. |
| "No data appearing" | Wait for initial sync (2-5 minutes). Check logs. |
| "WebSocket disconnects" | Check network stability. Verify ENABLE_*_REALTIME is true. |
| "Duplicate users" | Ensure SYNC_INTERVAL isn't too aggressive. Deduplication runs automatically. |

See **[Troubleshooting](Troubleshooting)** for more detailed solutions.

---

## Next Steps

- **[Configuration](Configuration)** - All configuration options
- **[Authentication](Authentication)** - Set up user authentication
- **[Features](Features)** - Explore analytics features
