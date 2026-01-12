# Troubleshooting Guide

Common issues and their solutions.

**[Home](Home)** | **[Configuration](Configuration)** | **Troubleshooting** | **[FAQ](FAQ)**

---

## Quick Diagnostics

### Check System Health

```bash
curl http://localhost:3857/api/v1/health
```

Expected response:

```json
{
  "status": "healthy",
  "version": "...",
  "database": "connected",
  "uptime": "..."
}
```

### Check Logs

```bash
# Docker
docker logs cartographus --tail 100

# Docker Compose
docker-compose logs -f cartographus

# Binary/systemd
journalctl -u cartographus -f
```

### Check Sync Status

```bash
curl http://localhost:3857/api/v1/sync/status
```

---

## Installation Issues

### Container Won't Start

**Symptoms**: Container exits immediately or restarts continuously.

**Solutions**:

1. **Check logs for errors:**
   ```bash
   docker logs cartographus
   ```

2. **Verify required environment variables:**
   - `JWT_SECRET` (minimum 32 characters)
   - `ADMIN_USERNAME`
   - `ADMIN_PASSWORD` (minimum 12 characters)
   - At least one media server enabled

3. **Check volume permissions:**
   ```bash
   # Ensure data directory exists and is writable
   mkdir -p ./data
   chmod 755 ./data
   ```

4. **Verify port availability:**
   ```bash
   # Check if port 3857 is in use
   lsof -i :3857
   ```

### "JWT secret too short" Error

**Solution**: Generate a proper secret:

```bash
openssl rand -base64 48
```

Use the full output as your `JWT_SECRET`.

### "Database locked" Error

**Cause**: Another process is accessing the database.

**Solutions**:

1. Ensure only one Cartographus instance is running
2. Stop any backup processes
3. Check for zombie processes:
   ```bash
   ps aux | grep cartographus
   ```

---

## Connection Issues

### "Connection refused" to Media Server

**Symptoms**: Cannot connect to Plex, Jellyfin, or Emby.

**Solutions**:

1. **Verify URL format:**
   - Include protocol: `http://` or `https://`
   - Include port: `:32400`, `:8096`
   - Example: `http://192.168.1.100:32400`

2. **Check network connectivity:**
   ```bash
   # From the Cartographus container
   docker exec cartographus curl -s http://plex:32400/identity
   ```

3. **Docker networking:**
   - Use container names for internal Docker communication
   - Use IP addresses for external servers
   - Ensure containers are on the same Docker network

4. **Firewall rules:**
   - Port 32400 open for Plex
   - Port 8096 open for Jellyfin/Emby

### "Unauthorized" or "Invalid token"

**Symptoms**: Authentication errors when connecting to media servers.

**Solutions**:

1. **Plex token:**
   - Regenerate token if expired
   - Verify token works:
     ```bash
     curl -H "X-Plex-Token: YOUR_TOKEN" http://plex:32400/identity
     ```

2. **Jellyfin/Emby API key:**
   - Create a new API key in the dashboard
   - Ensure no extra whitespace when copying

3. **Token scope:**
   - Ensure the token has access to the specific server

### WebSocket Disconnections

**Symptoms**: Real-time updates stop working.

**Solutions**:

1. **Enable real-time for your server:**
   ```yaml
   ENABLE_PLEX_REALTIME=true
   JELLYFIN_REALTIME_ENABLED=true
   EMBY_REALTIME_ENABLED=true
   ```

2. **Check reverse proxy configuration:**
   - WebSocket upgrade must be enabled
   - Timeout settings may need adjustment

3. **Enable backup polling:**
   ```yaml
   PLEX_SESSION_POLLING_ENABLED=true
   PLEX_SESSION_POLLING_INTERVAL=30s
   ```

---

## Data Issues

### No Data Appearing

**Symptoms**: Dashboard shows no playback data.

**Solutions**:

1. **Wait for initial sync:**
   - First sync can take 2-5 minutes
   - Check sync status:
     ```bash
     curl http://localhost:3857/api/v1/sync/status
     ```

2. **Verify media server connection:**
   - Check logs for connection errors
   - Verify server URL and credentials

3. **Check date range:**
   - Default view may filter to recent data
   - Expand date range in the UI

4. **Verify playback activity exists:**
   - Ensure there's been actual playback on your server
   - Check your media server's native activity log

### Duplicate Users

**Symptoms**: Same user appears multiple times.

**Solutions**:

1. **Wait for deduplication:**
   - Automatic deduplication runs periodically
   - Force with sync restart

2. **Check user matching:**
   - Users are matched by email, username, and server ID
   - Inconsistent data across servers may cause duplicates

### Stale Data

**Symptoms**: Data doesn't reflect recent activity.

**Solutions**:

1. **Check sync interval:**
   ```yaml
   SYNC_INTERVAL=5m  # Default 5 minutes
   ```

2. **Enable real-time updates:**
   - WebSocket provides sub-second updates

3. **Force sync:**
   ```bash
   curl -X POST http://localhost:3857/api/v1/sync/trigger
   ```

---

## Performance Issues

### Slow Queries

**Symptoms**: Dashboard loads slowly, API responses are slow.

**Solutions**:

1. **Increase DuckDB memory:**
   ```yaml
   DUCKDB_MAX_MEMORY=4GB  # Increase from default 2GB
   ```

2. **Check database size:**
   ```bash
   ls -lh data/cartographus.duckdb
   ```

3. **Enable pagination:**
   - Use page size limits in API calls
   - Avoid loading all data at once

### High Memory Usage

**Solutions**:

1. **Limit DuckDB memory:**
   ```yaml
   DUCKDB_MAX_MEMORY=1GB  # Reduce if needed
   ```

2. **Reduce NATS memory:**
   ```yaml
   NATS_MAX_MEMORY=536870912  # 512MB
   ```

3. **Limit concurrent operations:**
   - Reduce sync batch size
   - Limit WebSocket subscribers

### Maps Not Loading

**Symptoms**: Map view shows blank or error.

**Solutions**:

1. **Check WebGL support:**
   - Open browser console (F12)
   - Look for WebGL errors
   - Try Chrome or Firefox

2. **Browser requirements:**
   - Hardware acceleration enabled
   - Modern browser (Chrome 90+, Firefox 88+, Safari 14+)

3. **Data availability:**
   - Maps require geolocation data
   - Check that IP addresses are being resolved

---

## Authentication Issues

### Cannot Log In

**Solutions**:

1. **Verify credentials:**
   - Check `ADMIN_USERNAME` and `ADMIN_PASSWORD` in configuration
   - Passwords are case-sensitive

2. **Clear browser cookies:**
   - Old session cookies may conflict

3. **Check auth mode:**
   - Ensure `AUTH_MODE` matches your setup (jwt, basic, oidc, plex)

### OIDC Errors

**Solutions**:

1. **Verify issuer URL:**
   - Must be accessible from Cartographus container
   - Include full URL with protocol

2. **Check redirect URL:**
   - Must match exactly what's configured in your IdP
   - Include full path: `https://cartographus.example.com/api/auth/oidc/callback`

3. **Verify client credentials:**
   - Client ID and secret must match IdP configuration

### Plex Auth Not Working

**Solutions**:

1. **Verify client ID:**
   - Must be a valid Plex app ID

2. **Check callback URL:**
   - Plex auth uses popup flow, not redirect
   - Should work on localhost and private IPs

---

## Reverse Proxy Issues

### WebSocket Not Working Behind Proxy

**Nginx solution**:

```nginx
location / {
    proxy_pass http://localhost:3857;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "upgrade";
    proxy_set_header Host $host;
    proxy_read_timeout 86400;
}
```

**Caddy solution**:

```
cartographus.example.com {
    reverse_proxy localhost:3857
}
```

### X-Forwarded Headers Not Recognized

**Solution**: Configure trusted proxies:

```yaml
TRUSTED_PROXIES=172.17.0.1,10.0.0.0/8
```

---

## Backup Issues

### Backup Failed

**Solutions**:

1. **Check disk space:**
   ```bash
   df -h /data/backups
   ```

2. **Verify backup directory permissions:**
   ```bash
   ls -la /data/backups
   ```

3. **Check logs for specific errors:**
   ```bash
   docker logs cartographus | grep -i backup
   ```

### Cannot Restore Backup

**Solutions**:

1. **Verify backup file integrity:**
   - Check file size is reasonable
   - Try decompressing manually if gzipped

2. **Stop Cartographus before restore:**
   - Database must not be in use

---

## Getting Help

If you've tried these solutions and still have issues:

1. **Search existing issues:**
   [GitHub Issues](https://github.com/tomtom215/cartographus/issues)

2. **Gather diagnostic information:**
   ```bash
   # Version
   curl http://localhost:3857/api/v1/health

   # Recent logs
   docker logs cartographus --tail 200

   # Configuration (redact secrets)
   docker exec cartographus env | grep -v SECRET | grep -v PASSWORD | grep -v TOKEN
   ```

3. **Open a new issue:**
   Include version, logs, configuration, and steps to reproduce.

---

## Next Steps

- **[FAQ](FAQ)** - Frequently asked questions
- **[Configuration](Configuration)** - Configuration reference
- **[Home](Home)** - Back to wiki home
