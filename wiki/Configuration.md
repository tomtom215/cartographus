# Configuration Reference

Complete reference for all Cartographus configuration options.

**[Home](Home)** | **[Quick Start](Quick-Start)** | **[Installation](Installation)** | **Configuration**

---

## Overview

Cartographus is configured using environment variables. All settings have sensible defaults - you only need to configure what you want to change.

### Configuration Priority

1. Environment variables (highest priority)
2. Configuration file (config.yaml)
3. Built-in defaults

---

## Required Configuration

At minimum, you need:

1. **Security credentials** (JWT secret and admin account)
2. **At least one media server** (Plex, Jellyfin, or Emby)

```bash
# Required security
JWT_SECRET=your-32-character-minimum-secret
ADMIN_USERNAME=admin
ADMIN_PASSWORD=YourSecurePassword123!

# Plus one media server (see Media Servers section)
```

---

## Security Configuration

### Authentication

| Variable | Default | Description |
|----------|---------|-------------|
| `AUTH_MODE` | `jwt` | Authentication mode: `jwt`, `basic`, `oidc`, `plex`, `multi`, or `none` |
| `JWT_SECRET` | *required* | Secret for signing JWT tokens (minimum 32 characters) |
| `ADMIN_USERNAME` | *required* | Admin account username |
| `ADMIN_PASSWORD` | *required* | Admin account password (minimum 12 characters) |
| `SESSION_TIMEOUT` | `24h` | How long sessions remain valid |

Generate a secure JWT secret:

```bash
openssl rand -base64 48
```

### Rate Limiting

| Variable | Default | Description |
|----------|---------|-------------|
| `RATE_LIMIT_REQUESTS` | `100` | Requests allowed per window |
| `RATE_LIMIT_WINDOW` | `1m` | Rate limit window duration |
| `DISABLE_RATE_LIMIT` | `false` | Disable rate limiting (not recommended) |

### CORS

| Variable | Default | Description |
|----------|---------|-------------|
| `CORS_ORIGINS` | `*` | Allowed origins (comma-separated, or `*` for all) |
| `TRUSTED_PROXIES` | *empty* | Trusted proxy IPs for X-Forwarded-For |

---

## Media Server Configuration

See **[Media Servers](Media-Servers)** for detailed setup instructions.

### Plex

| Variable | Default | Description |
|----------|---------|-------------|
| `ENABLE_PLEX_SYNC` | `false` | Enable Plex integration |
| `PLEX_URL` | *required if enabled* | Plex server URL (e.g., `http://plex:32400`) |
| `PLEX_TOKEN` | *required if enabled* | Plex authentication token |
| `ENABLE_PLEX_REALTIME` | `false` | Enable WebSocket real-time updates |
| `PLEX_SERVER_ID` | `default` | Identifier for multi-server setups |

### Jellyfin

| Variable | Default | Description |
|----------|---------|-------------|
| `JELLYFIN_ENABLED` | `false` | Enable Jellyfin integration |
| `JELLYFIN_URL` | *required if enabled* | Jellyfin server URL |
| `JELLYFIN_API_KEY` | *required if enabled* | Jellyfin API key |
| `JELLYFIN_REALTIME_ENABLED` | `false` | Enable WebSocket real-time updates |
| `JELLYFIN_SERVER_ID` | `default` | Identifier for multi-server setups |

### Emby

| Variable | Default | Description |
|----------|---------|-------------|
| `EMBY_ENABLED` | `false` | Enable Emby integration |
| `EMBY_URL` | *required if enabled* | Emby server URL |
| `EMBY_API_KEY` | *required if enabled* | Emby API key |
| `EMBY_REALTIME_ENABLED` | `false` | Enable WebSocket real-time updates |
| `EMBY_SERVER_ID` | `default` | Identifier for multi-server setups |

### Tautulli (Optional)

| Variable | Default | Description |
|----------|---------|-------------|
| `TAUTULLI_ENABLED` | `false` | Enable Tautulli for historical import |
| `TAUTULLI_URL` | *required if enabled* | Tautulli server URL |
| `TAUTULLI_API_KEY` | *required if enabled* | Tautulli API key |

---

## Server Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `HTTP_PORT` | `3857` | HTTP server port |
| `HTTP_HOST` | `0.0.0.0` | Bind address |
| `HTTP_TIMEOUT` | `30s` | Request timeout |
| `SERVER_LATITUDE` | `0.0` | Server location for globe view |
| `SERVER_LONGITUDE` | `0.0` | Server location for globe view |

> **Note**: Port 3857 references EPSG:3857, the Web Mercator map projection.

---

## Database Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `DUCKDB_PATH` | `/data/cartographus.duckdb` | Database file path |
| `DUCKDB_MAX_MEMORY` | `2GB` | Maximum memory for queries |
| `DUCKDB_THREADS` | *auto* | Worker threads (0 = CPU count) |

### Memory Recommendations

| System RAM | Recommended Setting |
|------------|---------------------|
| 4 GB | `1GB` |
| 8 GB | `2GB` |
| 16 GB | `4GB` |
| 32 GB+ | `8GB` |

---

## Sync Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `SYNC_INTERVAL` | `5m` | How often to sync with media servers |
| `SYNC_LOOKBACK` | `24h` | Initial sync lookback period |
| `SYNC_BATCH_SIZE` | `1000` | Records per API request |
| `SYNC_RETRY_ATTEMPTS` | `5` | Retry attempts on failure |
| `SYNC_RETRY_DELAY` | `2s` | Initial retry delay |

---

## Logging Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `LOG_LEVEL` | `info` | Log level: `trace`, `debug`, `info`, `warn`, `error` |
| `LOG_FORMAT` | `json` | Log format: `json` or `console` |
| `LOG_CALLER` | `false` | Include file:line in logs |

---

## Backup Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `BACKUP_ENABLED` | `true` | Enable automated backups |
| `BACKUP_DIR` | `/data/backups` | Backup storage directory |
| `BACKUP_INTERVAL` | `24h` | Backup frequency |
| `BACKUP_RETENTION_MAX_COUNT` | `50` | Maximum backups to keep |
| `BACKUP_RETENTION_MAX_DAYS` | `90` | Maximum backup age |

---

## Detection Engine

Security anomaly detection for account sharing.

| Variable | Default | Description |
|----------|---------|-------------|
| `DETECTION_ENABLED` | `true` | Enable detection engine |
| `DETECTION_TRUST_SCORE_DECREMENT` | `10` | Score penalty per violation |
| `DETECTION_TRUST_SCORE_THRESHOLD` | `50` | Threshold for restrictions |

See **[Security Detection](Security-Detection)** for details on detection rules.

---

## OIDC Configuration

For enterprise SSO with Authelia, Authentik, Keycloak, Okta, etc.

| Variable | Default | Description |
|----------|---------|-------------|
| `AUTH_MODE` | - | Set to `oidc` or `multi` to enable OIDC |
| `OIDC_ISSUER_URL` | *required* | Identity provider URL |
| `OIDC_CLIENT_ID` | *required* | OAuth2 client ID |
| `OIDC_CLIENT_SECRET` | - | OAuth2 client secret (optional for public clients) |
| `OIDC_REDIRECT_URL` | *required* | Callback URL |
| `OIDC_SCOPES` | `openid,profile,email` | OAuth2 scopes (comma-separated) |
| `OIDC_PKCE_ENABLED` | `true` | Enable PKCE (recommended) |
| `OIDC_SESSION_MAX_AGE` | `24h` | Session duration |
| `OIDC_COOKIE_SECURE` | `true` | Use secure cookies (HTTPS only) |

See **[Authentication](Authentication)** for complete OIDC setup.

---

## Plex Authentication

"Sign in with Plex" for zero-configuration user access.

| Variable | Default | Description |
|----------|---------|-------------|
| `AUTH_MODE` | - | Set to `plex` or `multi` to enable Plex auth |
| `PLEX_AUTH_CLIENT_ID` | *required* | Plex application client identifier |
| `PLEX_AUTH_REDIRECT_URI` | *required* | OAuth callback URL |
| `PLEX_AUTH_ENABLE_SERVER_DETECTION` | `true` | Auto-detect server ownership |
| `PLEX_AUTH_SERVER_MACHINE_ID` | - | Target Plex server machine ID |
| `PLEX_AUTH_SERVER_OWNER_ROLE` | `admin` | Role assigned to server owners |
| `PLEX_AUTH_SERVER_ADMIN_ROLE` | `editor` | Role assigned to shared library admins |

See **[Authentication](Authentication)** for complete Plex auth setup.

---

## Environment Mode

Controls security behaviors and validation rules.

| Variable | Default | Description |
|----------|---------|-------------|
| `ENVIRONMENT` | `development` | Environment mode: `development`, `staging`, or `production` |

**Production Mode Restrictions:**
- `AUTH_MODE=none` is rejected (security requirement)
- `CORS_ORIGINS=*` is rejected when authentication is enabled
- More strict validation of security settings

---

## Event Processing (NATS)

| Variable | Default | Description |
|----------|---------|-------------|
| `NATS_ENABLED` | `true` | Enable NATS JetStream |
| `NATS_EVENT_SOURCING` | `true` | Enable event sourcing mode (NATS-first architecture) |
| `NATS_URL` | `nats://127.0.0.1:4222` | NATS server URL |
| `NATS_EMBEDDED` | `true` | Use embedded NATS server |
| `NATS_STORE_DIR` | `/data/nats/jetstream` | JetStream storage directory |
| `NATS_MAX_MEMORY` | `1073741824` | Max memory for JetStream (bytes, default 1GB) |
| `NATS_MAX_STORE` | `10737418240` | Max disk storage for JetStream (bytes, default 10GB) |
| `NATS_RETENTION_DAYS` | `7` | Event retention period in days (1-365) |
| `NATS_BATCH_SIZE` | `1000` | Batch size for DuckDB writes (1-10000) |
| `NATS_FLUSH_INTERVAL` | `5s` | Max time between DuckDB flushes (1s-1h) |
| `NATS_SUBSCRIBERS` | `4` | Number of concurrent message processors (1-32) |
| `NATS_DURABLE_NAME` | `media-processor` | Consumer durable name |
| `NATS_QUEUE_GROUP` | `processors` | Queue group for load balancing |

---

## Write-Ahead Log (WAL)

BadgerDB-based durability layer for event persistence.

| Variable | Default | Description |
|----------|---------|-------------|
| `WAL_ENABLED` | `true` | Enable write-ahead log |
| `WAL_PATH` | `/data/wal` | WAL storage directory |
| `WAL_SYNC_WRITES` | `true` | Force fsync on every write (maximum durability) |
| `WAL_RETRY_INTERVAL` | `30s` | Interval between retry loop iterations |
| `WAL_MAX_RETRIES` | `100` | Maximum retry attempts for failed entries |
| `WAL_RETRY_BACKOFF` | `5s` | Initial backoff duration for retries |
| `WAL_COMPACT_INTERVAL` | `1h` | Interval between compaction runs |
| `WAL_ENTRY_TTL` | `168h` | Time-to-live for unconfirmed entries (7 days) |
| `WAL_MEMTABLE_SIZE` | `16777216` | BadgerDB memtable size in bytes (16MB) |
| `WAL_VLOG_SIZE` | `67108864` | BadgerDB value log file size (64MB) |
| `WAL_NUM_COMPACTORS` | `2` | Number of compaction workers |
| `WAL_COMPRESSION` | `true` | Enable Snappy compression |
| `WAL_LEASE_DURATION` | `2m` | Processing lease duration |

---

## VPN Detection

Identifies connections from known VPN providers for accurate geolocation.

| Variable | Default | Description |
|----------|---------|-------------|
| `VPN_ENABLED` | `true` | Enable VPN detection |
| `VPN_DATA_FILE` | - | Path to gluetun servers.json file (optional) |
| `VPN_CACHE_SIZE` | `10000` | Maximum lookup cache entries |
| `VPN_AUTO_UPDATE` | `false` | Enable automatic data updates (future) |
| `VPN_UPDATE_INTERVAL` | `24h` | Update check interval (future) |

---

## Import Configuration

Direct import from Tautulli SQLite database files.

| Variable | Default | Description |
|----------|---------|-------------|
| `IMPORT_ENABLED` | `false` | Enable import functionality |
| `IMPORT_DB_PATH` | *required if enabled* | Path to Tautulli SQLite database file |
| `IMPORT_BATCH_SIZE` | `1000` | Records per batch (1-10000) |
| `IMPORT_DRY_RUN` | `false` | Validate without importing |
| `IMPORT_AUTO_START` | `false` | Start import automatically on startup |
| `IMPORT_RESUME_FROM_ID` | `0` | Resume from specific session ID |
| `IMPORT_SKIP_GEOLOCATION` | `false` | Skip geolocation enrichment |

---

## Recommendation Engine

Personalized media suggestions based on viewing history.

> **Note**: Disabled by default due to computational requirements. Recommended: 4+ cores, 8GB+ RAM.

| Variable | Default | Description |
|----------|---------|-------------|
| `RECOMMEND_ENABLED` | `false` | Enable recommendation engine |
| `RECOMMEND_TRAIN_INTERVAL` | `24h` | Training schedule interval |
| `RECOMMEND_TRAIN_ON_STARTUP` | `false` | Trigger training on startup |
| `RECOMMEND_MIN_INTERACTIONS` | `100` | Minimum interactions before training |
| `RECOMMEND_MODEL_PATH` | `/data/recommend` | Path to store trained models |
| `RECOMMEND_ALGORITHMS` | `covisit,content` | Enabled algorithms (comma-separated) |
| `RECOMMEND_CACHE_TTL` | `5m` | Recommendation cache TTL |
| `RECOMMEND_MAX_CANDIDATES` | `1000` | Maximum candidates to score |
| `RECOMMEND_DIVERSITY_LAMBDA` | `0.7` | Relevance vs diversity (0-1) |

**Available Algorithms**: `covisit`, `content`, `popularity`, `ease`, `als`, `usercf`, `itemcf`, `fpmc`, `linucb`

---

## Notifications

### Discord Webhooks

| Variable | Default | Description |
|----------|---------|-------------|
| `DISCORD_WEBHOOK_ENABLED` | `false` | Enable Discord notifications |
| `DISCORD_WEBHOOK_URL` | *required* | Discord webhook URL |

### Generic Webhooks

| Variable | Default | Description |
|----------|---------|-------------|
| `WEBHOOK_ENABLED` | `false` | Enable generic webhooks |
| `WEBHOOK_URL` | *required* | Webhook endpoint URL |
| `WEBHOOK_HEADERS` | *empty* | Custom headers (key=value,key=value) |

---

## Duration Format

Duration values use Go duration format:

| Format | Meaning |
|--------|---------|
| `5s` | 5 seconds |
| `5m` | 5 minutes |
| `1h` | 1 hour |
| `24h` | 24 hours |
| `1h30m` | 1 hour 30 minutes |

---

## Example Configurations

### Minimal (Plex only)

```bash
JWT_SECRET=your_secure_secret_at_least_32_characters
ADMIN_USERNAME=admin
ADMIN_PASSWORD=YourSecurePassword123!
ENABLE_PLEX_SYNC=true
PLEX_URL=http://plex:32400
PLEX_TOKEN=your_plex_token
```

### Full Featured

```bash
# Security
JWT_SECRET=your_secure_secret_at_least_32_characters
ADMIN_USERNAME=admin
ADMIN_PASSWORD=YourSecurePassword123!

# Plex
ENABLE_PLEX_SYNC=true
PLEX_URL=http://plex:32400
PLEX_TOKEN=your_plex_token
ENABLE_PLEX_REALTIME=true

# Database
DUCKDB_MAX_MEMORY=4GB

# Sync
SYNC_INTERVAL=5m
SYNC_LOOKBACK=7d

# Backups
BACKUP_ENABLED=true
BACKUP_INTERVAL=12h

# Logging
LOG_LEVEL=info
LOG_FORMAT=json

# Detection
DETECTION_ENABLED=true
```

---

## Configuration File (YAML)

You can also use a YAML configuration file:

```yaml
# /etc/cartographus/config.yaml
security:
  auth_mode: jwt
  jwt_secret: "your-32-character-secret"
  admin_username: admin
  admin_password: "YourSecurePassword123!"

plex:
  enabled: true
  url: "http://plex:32400"
  token: "your-plex-token"
  realtime_enabled: true

database:
  path: "/data/cartographus.duckdb"
  max_memory: "4GB"

logging:
  level: info
  format: json
```

---

## Next Steps

- **[Media Servers](Media-Servers)** - Detailed media server configuration
- **[Authentication](Authentication)** - Set up SSO and multi-user access
- **[Troubleshooting](Troubleshooting)** - Configuration issues and solutions
