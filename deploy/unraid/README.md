# Unraid Community App Template

This directory contains the Unraid Community Application template for Cartographus.

## Installation

### Option 1: Community Applications (Recommended)

Once the template is accepted into the Community Applications repository:

1. Open Unraid WebUI
2. Go to **Apps** tab
3. Search for "Cartographus"
4. Click **Install**
5. Configure the required settings
6. Click **Apply**

### Option 2: Manual Template Installation

1. In Unraid WebUI, go to **Docker** tab
2. Click **Add Container**
3. Toggle **Advanced View** in the top right
4. At the bottom, click **Template repositories**
5. Add: `https://github.com/tomtom215/cartographus`
6. Click **Save**
7. Select "cartographus" from the template dropdown

### Option 3: Direct XML Import

1. Copy `cartographus.xml` to `/boot/config/plugins/dockerMan/templates-user/`
2. Refresh the Docker page
3. Select "cartographus" from the template dropdown

## Configuration

### Required Settings

| Setting | Description |
|---------|-------------|
| **WebUI Port** | Default: 3857 (EPSG:3857 - Web Mercator) |
| **Data Storage** | Path for database, models, and NATS data |
| **Auth Mode** | `jwt` (recommended), `basic`, `none`, `oidc`, `plex`, or `multi` |
| **JWT Secret** | Generate with: `openssl rand -base64 32` |
| **Admin Password** | Strong password (NIST recommends 12+ chars) |

### Media Server Integration

Enable at least one media server:

**Tautulli (Recommended for Plex)**
- Best option for Plex users
- Provides enhanced analytics and history
- Set `TAUTULLI_ENABLED=true`, `TAUTULLI_URL`, and `TAUTULLI_API_KEY`

**Direct Plex**
- Alternative to Tautulli
- Real-time WebSocket events and webhooks
- Set `PLEX_ENABLED=true`, `PLEX_URL`, and `PLEX_TOKEN`
- Optional: Enable real-time with `PLEX_REALTIME_ENABLED=true`

**Jellyfin**
- Set `JELLYFIN_ENABLED=true`, `JELLYFIN_URL`, and `JELLYFIN_API_KEY`
- Optional: Enable real-time with `JELLYFIN_REALTIME_ENABLED=true`

**Emby**
- Set `EMBY_ENABLED=true`, `EMBY_URL`, and `EMBY_API_KEY`
- Optional: Enable real-time with `EMBY_REALTIME_ENABLED=true`

### Authentication Options

The template supports multiple authentication methods:

| Mode | Description |
|------|-------------|
| `none` | No authentication (development only) |
| `basic` | HTTP Basic Auth with username/password |
| `jwt` | JWT token-based authentication (recommended) |
| `oidc` | OIDC/OAuth 2.0 with external provider (Zitadel, Keycloak, etc.) |
| `plex` | Plex OAuth authentication with server owner detection |
| `multi` | Multiple authentication methods (OIDC + Plex + Basic) |

#### OIDC Configuration (Zero Trust)

For enterprise deployments with identity providers:

```
AUTH_MODE=oidc
OIDC_ISSUER_URL=https://auth.example.com/realms/main
OIDC_CLIENT_ID=cartographus
OIDC_REDIRECT_URL=http://your-unraid:3857/api/v1/auth/oidc/callback
```

#### Plex Authentication

For Plex server owners:

```
AUTH_MODE=plex
PLEX_AUTH_CLIENT_ID=your-plex-app-id
PLEX_AUTH_REDIRECT_URI=http://your-unraid:3857/api/v1/auth/plex/callback
```

Server owners are automatically assigned the `admin` role.

### Advanced Features

#### Newsletter Scheduler

Automated digest delivery via email, Discord, or webhooks:

```
NEWSLETTER_ENABLED=true
NEWSLETTER_CHECK_INTERVAL=1m
```

Configure delivery channels in the web UI after setup.

#### Detection Engine

Security monitoring for suspicious activity:

```
DETECTION_ENABLED=true
DISCORD_WEBHOOK_ENABLED=true
DISCORD_WEBHOOK_URL=https://discord.com/api/webhooks/...
```

Detects: impossible travel, concurrent streams, device velocity anomalies.

#### Recommendation Engine

Personalized media suggestions (requires 4+ cores, 8GB+ RAM):

```
RECOMMEND_ENABLED=true
RECOMMEND_ALGORITHMS=covisit,content,ease
```

#### VPN Detection

Identify VPN connections for accurate geolocation:

```
VPN_ENABLED=true
```

Uses gluetun VPN provider data (24+ providers, 10,000+ IPs).

### Geolocation

For IP geolocation without Tautulli:

**Option 1: MaxMind (Recommended)**
```
GEOIP_PROVIDER=maxmind
MAXMIND_ACCOUNT_ID=your-account-id
MAXMIND_LICENSE_KEY=your-license-key
```

Register free at: https://www.maxmind.com/en/geolite2/signup

**Option 2: ip-api.com (Free)**
```
GEOIP_PROVIDER=ipapi
```

Free tier: 45 requests/minute.

## Configuration Categories

The template organizes settings into these categories:

### Always Visible
- Port and storage
- Authentication basics
- Media server connections (Tautulli, Plex, Jellyfin, Emby)

### Advanced Settings
Click "Show more settings" to access:

- OIDC/OAuth 2.0 configuration
- Plex OAuth authentication
- Casbin RBAC authorization
- Session store settings
- Media server advanced options (WebSocket, webhooks, transcode monitoring)
- Tautulli database import
- Geolocation providers
- VPN detection
- Detection engine and alerts
- Discord/Webhook notifications
- Newsletter scheduler
- Recommendation engine
- NATS event processing
- Database tuning
- Sync configuration
- Server location (for globe visualization)
- API limits
- Rate limiting
- CORS settings
- Logging options

## First Run

1. Navigate to `http://[UNRAID_IP]:3857`
2. Log in with admin credentials
3. Configure your media server connections in Settings
4. Import existing data or wait for real-time events
5. Explore the map, globe, and analytics dashboards

## Troubleshooting

### Container won't start

Check logs in Unraid Docker tab or run:
```bash
docker logs cartographus
```

Common issues:
- JWT_SECRET too short (needs 32+ characters)
- Invalid AUTH_MODE value
- Port 3857 already in use
- Missing required media server credentials

### Can't connect to media server

- Verify media server URL is accessible from Unraid
- Use internal IP addresses, not `localhost`
- Check API key/token is correct
- Ensure media server allows API access
- For real-time: verify WebSocket connectivity

### OIDC authentication issues

- Verify issuer URL is accessible
- Check redirect URI matches exactly
- Ensure client ID/secret are correct
- Review OIDC provider logs for errors

### Database errors

- Verify `/data` path has correct permissions
- Check available disk space (DuckDB + NATS storage)
- Review container logs for specific errors
- Consider increasing `DUCKDB_MAX_MEMORY` for large datasets

### Recommendations not working

- Ensure `RECOMMEND_ENABLED=true`
- Wait for minimum interactions (`RECOMMEND_MIN_INTERACTIONS`)
- Check system resources (4+ cores, 8GB+ RAM recommended)
- Review logs for training errors

## Updates

The template uses `ghcr.io/tomtom215/cartographus:latest`. To update:

1. Go to Docker tab
2. Click the Cartographus icon
3. Click **Check for Updates**
4. Apply if available

Or use Unraid's auto-update feature.

## Resource Requirements

| Component | Minimum | Recommended |
|-----------|---------|-------------|
| CPU | 2 cores | 4+ cores |
| RAM | 2GB | 4-8GB |
| Storage | 1GB | 10GB+ |

For recommendation engine: 4+ cores, 8GB+ RAM recommended.

## Environment Variable Reference

See the full list of environment variables in the template XML or the project's
[config.go](https://github.com/tomtom215/cartographus/blob/main/internal/config/config.go).

## Support

- **Issues**: https://github.com/tomtom215/cartographus/issues
- **Documentation**: https://github.com/tomtom215/cartographus#readme

## Contributing

To submit improvements to this template:

1. Fork the repository
2. Edit `deploy/unraid/cartographus.xml`
3. Submit a pull request

For inclusion in Community Applications, the template must also be submitted to:
https://github.com/selfhosters/unRAID-CA-templates
