# First Steps After Installation

What to do after you've installed Cartographus.

**[Home](Home)** | **[Quick Start](Quick-Start)** | **First Steps** | **[Features](Features)**

---

## Step 1: Verify Installation

### Check Health

Open your browser and navigate to:

```
http://localhost:3857/api/v1/health
```

You should see:

```json
{
  "status": "healthy",
  "version": "...",
  "database": "connected"
}
```

### Log In

Go to `http://localhost:3857` and log in with your admin credentials.

---

## Step 2: Wait for Initial Sync

After connecting a media server, Cartographus performs an initial sync:

1. **Connection test** - Verifies credentials and connectivity
2. **Library scan** - Discovers available libraries
3. **History fetch** - Pulls recent playback data (last 24 hours by default)

**This typically takes 2-5 minutes.** Check sync progress:

```bash
curl http://localhost:3857/api/v1/sync/status
```

Or view the Sync page in the web interface.

---

## Step 3: Explore the Dashboard

### Overview Page

The Overview page shows:

- **Active streams** - Currently playing content
- **Recent activity** - Latest playback sessions
- **Quick stats** - Users, plays, watch time

### Maps Page

Geographic visualization of playback activity:

- **2D Map** - WebGL map with clustered markers
- **3D Globe** - Interactive globe with location points
- **Controls** - Zoom, pan, filter by date range

### Analytics Page

47+ charts across multiple categories:

- **Users** - Engagement, retention, behavior
- **Content** - Popular titles, library stats
- **Performance** - Transcode efficiency, quality metrics

---

## Step 4: Enable Real-Time Updates

For live activity tracking, enable real-time for your server:

### Plex

```yaml
environment:
  - ENABLE_PLEX_REALTIME=true
```

### Jellyfin

```yaml
environment:
  - JELLYFIN_REALTIME_ENABLED=true
```

### Emby

```yaml
environment:
  - EMBY_REALTIME_ENABLED=true
```

Restart Cartographus after changing these settings.

---

## Step 5: Import Historical Data (Optional)

If you have years of playback history in Tautulli, import it:

1. Go to the **Sync** page
2. Select **Tautulli Import**
3. Enter your Tautulli URL and API key
4. Choose date range
5. Start import

Progress is tracked in real-time via WebSocket.

---

## Step 6: Configure Notifications (Optional)

Get alerts for security detections:

### Discord

```yaml
environment:
  - DISCORD_WEBHOOK_ENABLED=true
  - DISCORD_WEBHOOK_URL=https://discord.com/api/webhooks/...
```

### Generic Webhook

```yaml
environment:
  - WEBHOOK_ENABLED=true
  - WEBHOOK_URL=https://your-endpoint.example.com/webhook
```

---

## Step 7: Set Up Backups

Enable automated backups:

```yaml
environment:
  - BACKUP_ENABLED=true
  - BACKUP_INTERVAL=24h
  - BACKUP_RETENTION_MAX_COUNT=30
```

Backups are stored in `/data/backups/` by default.

---

## Common First-Time Questions

### Why is the map empty?

- **Wait for sync** - Initial sync takes 2-5 minutes
- **Need playback data** - Maps require actual playback with IP addresses
- **Check browser** - WebGL must be enabled

### Why don't I see historical data?

Default sync looks back 24 hours. To get more history:

1. **Increase lookback**: Set `SYNC_LOOKBACK=7d` or `SYNC_LOOKBACK=30d`
2. **Import from Tautulli**: Use the Sync page to import historical data
3. **Plex historical sync**: Enable `PLEX_HISTORICAL_SYNC=true`

### How do I add more users?

With default JWT auth, there's one admin account. For multi-user:

1. **OIDC**: Connect your identity provider
2. **Plex Auth**: Let users sign in with Plex credentials
3. **Multi Auth**: Combine multiple methods

See [Authentication](Authentication).

### How do I access remotely?

1. Set up a [Reverse Proxy](Reverse-Proxy) with HTTPS
2. Configure DNS or use a dynamic DNS service
3. Open required ports on your router/firewall

---

## Recommended Configuration

After getting started, consider these enhancements:

### Performance

```yaml
environment:
  - DUCKDB_MAX_MEMORY=4GB      # If you have RAM to spare
  - SYNC_INTERVAL=5m           # Default is fine for most
```

### Security

```yaml
environment:
  - DETECTION_ENABLED=true     # Account sharing detection
  - RATE_LIMIT_REQUESTS=100    # Protect against abuse
```

### Reliability

```yaml
environment:
  - BACKUP_ENABLED=true        # Automated backups
  - WAL_ENABLED=true           # Durability (default on)
```

---

## Next Steps

- **[Features](Features)** - Explore all features
- **[Configuration](Configuration)** - Fine-tune your setup
- **[Authentication](Authentication)** - Add more users
- **[Reverse Proxy](Reverse-Proxy)** - Enable remote access
