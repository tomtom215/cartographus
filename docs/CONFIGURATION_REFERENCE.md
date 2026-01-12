# Configuration Reference

Complete reference for all Cartographus configuration options.

**Related Documentation**:
- [PRODUCTION_DEPLOYMENT.md](./PRODUCTION_DEPLOYMENT.md) - Production deployment guide
- [SECURITY_HARDENING.md](./SECURITY_HARDENING.md) - Security configuration
- [ADR-0012](./adr/0012-configuration-management-koanf.md) - Configuration architecture

---

## Table of Contents

1. [Configuration Methods](#configuration-methods)
2. [Quick Start](#quick-start)
3. [Configuration Sections](#configuration-sections)
   - [Tautulli](#tautulli-configuration)
   - [Plex](#plex-configuration)
   - [Jellyfin](#jellyfin-configuration)
   - [Emby](#emby-configuration)
   - [Database](#database-configuration)
   - [Sync](#sync-configuration)
   - [Server](#server-configuration)
   - [API](#api-configuration)
   - [Security](#security-configuration)
   - [NATS JetStream](#nats-jetstream-configuration)
   - [Write-Ahead Log](#write-ahead-log-configuration)
   - [Backup](#backup-configuration)
   - [Detection Engine](#detection-engine-configuration)
   - [Notifications](#notification-configuration)
   - [Logging](#logging-configuration)
   - [GeoIP](#geoip-configuration)
   - [Recommendation Engine](#recommendation-engine-configuration)

---

## Configuration Methods

Cartographus supports three configuration methods with the following precedence (later overrides earlier):

1. **Built-in Defaults** - Sensible production defaults
2. **Configuration File** - YAML file (`config.yaml`)
3. **Environment Variables** - Highest priority, overrides all others

### Configuration File Locations

The application searches for config files in this order:
- `./config.yaml` (current directory)
- `./config.yml`
- `/etc/cartographus/config.yaml`
- `/etc/cartographus/config.yml`
- Path specified by `CONFIG_PATH` environment variable

### YAML vs Environment Variables

| Format | Best For | Example |
|--------|----------|---------|
| YAML | Complex nested configs, multiple servers | Multi-server setups |
| Env Vars | Docker, Kubernetes, CI/CD, secrets | Single-server deployments |

---

## Quick Start

### Minimal Configuration (Environment Variables)

```bash
# Required for Tautulli integration
TAUTULLI_URL=http://tautulli:8181
TAUTULLI_API_KEY=your_api_key_here

# Required for security
AUTH_MODE=jwt
JWT_SECRET=your-32-character-minimum-secret
ADMIN_USERNAME=admin
ADMIN_PASSWORD=YourSecurePassword123!
```

### Minimal Configuration (YAML)

```yaml
tautulli:
  url: "http://tautulli:8181"
  api_key: "your_api_key_here"

security:
  auth_mode: "jwt"
  jwt_secret: "your-32-character-minimum-secret"
  admin_username: "admin"
  admin_password: "YourSecurePassword123!"
```

---

## Configuration Sections

### Tautulli Configuration

Primary data source for playback history (optional, for Plex via Tautulli).

| Environment Variable | YAML Path | Type | Default | Description |
|---------------------|-----------|------|---------|-------------|
| `TAUTULLI_ENABLED` | `tautulli.enabled` | boolean | `false` | Enable Tautulli integration |
| `TAUTULLI_URL` | `tautulli.url` | string | `""` | Tautulli server URL (include http/https) |
| `TAUTULLI_API_KEY` | `tautulli.api_key` | string | `""` | API key from Settings > Web Interface |
| `TAUTULLI_SERVER_ID` | `tautulli.server_id` | string | Auto | Unique identifier for multi-server setups |

**Example:**
```yaml
tautulli:
  enabled: true
  url: "http://tautulli:8181"
  api_key: "your_api_key_here"
```

---

### Plex Configuration

Direct Plex integration for real-time and historical playback data.

#### Basic Plex Settings

| Environment Variable | YAML Path | Type | Default | Description |
|---------------------|-----------|------|---------|-------------|
| `ENABLE_PLEX_SYNC` | `plex.enabled` | boolean | `false` | Enable Plex integration |
| `PLEX_SERVER_ID` | `plex.server_id` | string | Auto | Unique identifier for multi-server |
| `PLEX_URL` | `plex.url` | string | `""` | Plex server URL |
| `PLEX_TOKEN` | `plex.token` | string | `""` | X-Plex-Token for authentication |

#### Historical Sync

| Environment Variable | YAML Path | Type | Default | Description |
|---------------------|-----------|------|---------|-------------|
| `PLEX_HISTORICAL_SYNC` | `plex.historical_sync` | boolean | `false` | One-time historical backfill |
| `PLEX_SYNC_DAYS_BACK` | `plex.sync_days_back` | int | `365` | Days of history to sync (7-3650) |
| `PLEX_SYNC_INTERVAL` | `plex.sync_interval` | duration | `24h` | Periodic sync interval |

#### Real-Time

| Environment Variable | YAML Path | Type | Default | Description |
|---------------------|-----------|------|---------|-------------|
| `ENABLE_PLEX_REALTIME` | `plex.realtime_enabled` | boolean | `false` | WebSocket real-time updates |
| `PLEX_SESSION_POLLING_ENABLED` | `plex.session_polling_enabled` | boolean | `false` | Backup polling mechanism |
| `PLEX_SESSION_POLLING_INTERVAL` | `plex.session_polling_interval` | duration | `30s` | Polling interval (min 10s) |

#### OAuth

| Environment Variable | YAML Path | Type | Default | Description |
|---------------------|-----------|------|---------|-------------|
| `PLEX_OAUTH_CLIENT_ID` | `plex.oauth_client_id` | string | `""` | Plex app client ID |
| `PLEX_OAUTH_CLIENT_SECRET` | `plex.oauth_client_secret` | string | `""` | Plex app client secret |
| `PLEX_OAUTH_REDIRECT_URI` | `plex.oauth_redirect_uri` | string | `""` | OAuth callback URL |

#### Transcode Monitoring

| Environment Variable | YAML Path | Type | Default | Description |
|---------------------|-----------|------|---------|-------------|
| `ENABLE_PLEX_TRANSCODE_MONITORING` | `plex.transcode_monitoring` | boolean | `false` | Track transcode sessions |
| `PLEX_TRANSCODE_MONITORING_INTERVAL` | `plex.transcode_monitoring_interval` | duration | `10s` | Polling interval (5s-60s) |

#### Buffer Health Monitoring

| Environment Variable | YAML Path | Type | Default | Description |
|---------------------|-----------|------|---------|-------------|
| `ENABLE_BUFFER_HEALTH_MONITORING` | `plex.buffer_health_monitoring` | boolean | `false` | Predictive buffering detection |
| `BUFFER_HEALTH_POLL_INTERVAL` | `plex.buffer_health_poll_interval` | duration | `5s` | Poll interval (3s-30s) |
| `BUFFER_HEALTH_CRITICAL_THRESHOLD` | `plex.buffer_health_critical_threshold` | float | `20.0` | Critical alert threshold (%) |
| `BUFFER_HEALTH_RISKY_THRESHOLD` | `plex.buffer_health_risky_threshold` | float | `50.0` | Warning threshold (%) |

#### Webhooks

| Environment Variable | YAML Path | Type | Default | Description |
|---------------------|-----------|------|---------|-------------|
| `ENABLE_PLEX_WEBHOOKS` | `plex.webhooks_enabled` | boolean | `false` | Enable webhook receiver |
| `PLEX_WEBHOOK_SECRET` | `plex.webhook_secret` | string | `""` | HMAC-SHA256 verification secret |

**Example:**
```yaml
plex:
  enabled: true
  url: "http://plex:32400"
  token: "your_plex_token"
  historical_sync: false
  sync_days_back: 365
  sync_interval: "24h"
  realtime_enabled: true
  webhooks_enabled: true
  webhook_secret: "your_webhook_secret"
```

---

### Jellyfin Configuration

Direct Jellyfin integration for playback data.

| Environment Variable | YAML Path | Type | Default | Description |
|---------------------|-----------|------|---------|-------------|
| `JELLYFIN_ENABLED` | `jellyfin.enabled` | boolean | `false` | Enable Jellyfin integration |
| `JELLYFIN_SERVER_ID` | `jellyfin.server_id` | string | Auto | Unique identifier |
| `JELLYFIN_URL` | `jellyfin.url` | string | `""` | Jellyfin server URL |
| `JELLYFIN_API_KEY` | `jellyfin.api_key` | string | `""` | API key from Dashboard > API Keys |
| `JELLYFIN_USER_ID` | `jellyfin.user_id` | string | `""` | Optional user scope |
| `JELLYFIN_REALTIME_ENABLED` | `jellyfin.realtime_enabled` | boolean | `false` | WebSocket updates |
| `JELLYFIN_SESSION_POLLING_ENABLED` | `jellyfin.session_polling_enabled` | boolean | `false` | Backup polling |
| `JELLYFIN_SESSION_POLLING_INTERVAL` | `jellyfin.session_polling_interval` | duration | `30s` | Poll interval |
| `JELLYFIN_WEBHOOKS_ENABLED` | `jellyfin.webhooks_enabled` | boolean | `false` | Webhook receiver |
| `JELLYFIN_WEBHOOK_SECRET` | `jellyfin.webhook_secret` | string | `""` | Webhook verification |

**Example:**
```yaml
jellyfin:
  enabled: true
  url: "http://jellyfin:8096"
  api_key: "your_jellyfin_api_key"
  realtime_enabled: true
```

---

### Emby Configuration

Direct Emby integration for playback data.

| Environment Variable | YAML Path | Type | Default | Description |
|---------------------|-----------|------|---------|-------------|
| `EMBY_ENABLED` | `emby.enabled` | boolean | `false` | Enable Emby integration |
| `EMBY_SERVER_ID` | `emby.server_id` | string | Auto | Unique identifier |
| `EMBY_URL` | `emby.url` | string | `""` | Emby server URL |
| `EMBY_API_KEY` | `emby.api_key` | string | `""` | API key from Dashboard |
| `EMBY_USER_ID` | `emby.user_id` | string | `""` | Optional user scope |
| `EMBY_REALTIME_ENABLED` | `emby.realtime_enabled` | boolean | `false` | WebSocket updates |
| `EMBY_SESSION_POLLING_ENABLED` | `emby.session_polling_enabled` | boolean | `false` | Backup polling |
| `EMBY_SESSION_POLLING_INTERVAL` | `emby.session_polling_interval` | duration | `30s` | Poll interval |
| `EMBY_WEBHOOKS_ENABLED` | `emby.webhooks_enabled` | boolean | `false` | Webhook receiver |
| `EMBY_WEBHOOK_SECRET` | `emby.webhook_secret` | string | `""` | Webhook verification |

---

### Database Configuration

DuckDB analytics database settings.

| Environment Variable | YAML Path | Type | Default | Description |
|---------------------|-----------|------|---------|-------------|
| `DUCKDB_PATH` | `database.path` | string | `/data/cartographus.duckdb` | Database file path |
| `DUCKDB_MAX_MEMORY` | `database.max_memory` | string | `2GB` | Maximum memory for queries |
| `DUCKDB_THREADS` | `database.threads` | int | `0` | Worker threads (0 = NumCPU) |
| `SEED_MOCK_DATA` | `database.seed_mock_data` | boolean | `false` | Seed test data (CI only) |

**Memory Sizing Recommendations:**

| System RAM | Recommended Setting | Use Case |
|------------|---------------------|----------|
| 4GB | `1GB` | Small deployments |
| 8GB | `2GB` | Standard deployments |
| 16GB | `4GB` | Large libraries |
| 32GB+ | `8GB` | Enterprise |

---

### Sync Configuration

Playback data synchronization settings.

| Environment Variable | YAML Path | Type | Default | Description |
|---------------------|-----------|------|---------|-------------|
| `SYNC_INTERVAL` | `sync.interval` | duration | `5m` | Sync frequency |
| `SYNC_LOOKBACK` | `sync.lookback` | duration | `24h` | Initial sync lookback |
| `SYNC_ALL` | `sync.sync_all` | boolean | `false` | Sync all history (not incremental) |
| `SYNC_BATCH_SIZE` | `sync.batch_size` | int | `1000` | Records per API request |
| `SYNC_RETRY_ATTEMPTS` | `sync.retry_attempts` | int | `5` | Retry attempts on failure |
| `SYNC_RETRY_DELAY` | `sync.retry_delay` | duration | `2s` | Initial retry delay |

---

### Server Configuration

HTTP server settings.

| Environment Variable | YAML Path | Type | Default | Description |
|---------------------|-----------|------|---------|-------------|
| `HTTP_PORT` | `server.port` | int | `3857` | HTTP server port |
| `HTTP_HOST` | `server.host` | string | `0.0.0.0` | Bind address |
| `HTTP_TIMEOUT` | `server.timeout` | duration | `30s` | Request timeout |
| `SERVER_LATITUDE` | `server.latitude` | float | `0.0` | Server location (globe view) |
| `SERVER_LONGITUDE` | `server.longitude` | float | `0.0` | Server location (globe view) |

---

### API Configuration

REST API behavior settings.

| Environment Variable | YAML Path | Type | Default | Description |
|---------------------|-----------|------|---------|-------------|
| `API_DEFAULT_PAGE_SIZE` | `api.default_page_size` | int | `20` | Default pagination size |
| `API_MAX_PAGE_SIZE` | `api.max_page_size` | int | `100` | Maximum pagination size |

---

### Security Configuration

Authentication and authorization settings.

#### Authentication

| Environment Variable | YAML Path | Type | Default | Description |
|---------------------|-----------|------|---------|-------------|
| `AUTH_MODE` | `security.auth_mode` | string | `jwt` | Auth mode: `none`, `basic`, `jwt`, `oidc`, `plex`, `multi` |
| `JWT_SECRET` | `security.jwt_secret` | string | `""` | JWT signing secret (32+ chars) |
| `SESSION_TIMEOUT` | `security.session_timeout` | duration | `24h` | Token/session validity |
| `ADMIN_USERNAME` | `security.admin_username` | string | `""` | Admin username |
| `ADMIN_PASSWORD` | `security.admin_password` | string | `""` | Admin password (12+ chars, uppercase, lowercase, digit, special char) |

#### Rate Limiting

| Environment Variable | YAML Path | Type | Default | Description |
|---------------------|-----------|------|---------|-------------|
| `RATE_LIMIT_REQUESTS` | `security.rate_limit_reqs` | int | `100` | Requests per window |
| `RATE_LIMIT_WINDOW` | `security.rate_limit_window` | duration | `1m` | Rate limit window |
| `DISABLE_RATE_LIMIT` | `security.rate_limit_disabled` | boolean | `false` | Disable rate limiting |

#### CORS

| Environment Variable | YAML Path | Type | Default | Description |
|---------------------|-----------|------|---------|-------------|
| `CORS_ORIGINS` | `security.cors_origins` | []string | `["*"]` | Allowed origins |
| `TRUSTED_PROXIES` | `security.trusted_proxies` | []string | `[]` | Proxy IPs for X-Forwarded-For |

#### Session Store

| Environment Variable | YAML Path | Type | Default | Description |
|---------------------|-----------|------|---------|-------------|
| `SESSION_STORE` | `security.session_store` | string | `badger` | Store type: `memory` or `badger` |
| `SESSION_STORE_PATH` | `security.session_store_path` | string | `/data/sessions` | BadgerDB store path (required if session_store=badger) |

#### OIDC

| Environment Variable | YAML Path | Type | Default | Description |
|---------------------|-----------|------|---------|-------------|
| `OIDC_ISSUER_URL` | `security.oidc.issuer_url` | string | `""` | OIDC provider URL |
| `OIDC_CLIENT_ID` | `security.oidc.client_id` | string | `""` | Client ID |
| `OIDC_CLIENT_SECRET` | `security.oidc.client_secret` | string | `""` | Client secret |
| `OIDC_REDIRECT_URL` | `security.oidc.redirect_url` | string | `""` | OAuth callback URL |
| `OIDC_SCOPES` | `security.oidc.scopes` | []string | `["openid","profile","email"]` | OAuth scopes |
| `OIDC_PKCE_ENABLED` | `security.oidc.pkce_enabled` | boolean | `true` | Enable PKCE |
| `OIDC_SESSION_MAX_AGE` | `security.oidc.session_max_age` | duration | `24h` | Session duration |
| `OIDC_COOKIE_SECURE` | `security.oidc.cookie_secure` | boolean | `true` | Secure cookies |
| `OIDC_ROLES_CLAIM` | `security.oidc.roles_claim` | string | `roles` | JWT claim for roles |
| `OIDC_DEFAULT_ROLES` | `security.oidc.default_roles` | []string | `["viewer"]` | Default user roles |

#### Plex Auth

| Environment Variable | YAML Path | Type | Default | Description |
|---------------------|-----------|------|---------|-------------|
| `PLEX_AUTH_CLIENT_ID` | `security.plex_auth.client_id` | string | `""` | Plex OAuth client ID |
| `PLEX_AUTH_REDIRECT_URI` | `security.plex_auth.redirect_uri` | string | `""` | OAuth callback URL |
| `PLEX_AUTH_DEFAULT_ROLES` | `security.plex_auth.default_roles` | []string | `["viewer"]` | Default roles |
| `PLEX_AUTH_PLEX_PASS_ROLE` | `security.plex_auth.plex_pass_role` | string | `""` | Extra role for Plex Pass |

#### Casbin (RBAC)

| Environment Variable | YAML Path | Type | Default | Description |
|---------------------|-----------|------|---------|-------------|
| `CASBIN_MODEL_PATH` | `security.casbin.model_path` | string | `""` | RBAC model file |
| `CASBIN_POLICY_PATH` | `security.casbin.policy_path` | string | `""` | Policy CSV file |
| `CASBIN_DEFAULT_ROLE` | `security.casbin.default_role` | string | `viewer` | Default role |
| `CASBIN_CACHE_ENABLED` | `security.casbin.cache_enabled` | boolean | `true` | Enable policy cache |
| `CASBIN_CACHE_TTL` | `security.casbin.cache_ttl` | duration | `5m` | Cache TTL |

---

### NATS JetStream Configuration

Event streaming and message processing.

#### Core Settings

| Environment Variable | YAML Path | Type | Default | Description |
|---------------------|-----------|------|---------|-------------|
| `NATS_ENABLED` | `nats.enabled` | boolean | `true` | Enable NATS |
| `NATS_EVENT_SOURCING` | `nats.event_sourcing` | boolean | `true` | NATS-first architecture |
| `NATS_URL` | `nats.url` | string | `nats://127.0.0.1:4222` | Server URL |
| `NATS_EMBEDDED` | `nats.embedded_server` | boolean | `true` | Use embedded server |
| `NATS_STORE_DIR` | `nats.store_dir` | string | `/data/nats/jetstream` | Storage directory |

#### JetStream Settings

| Environment Variable | YAML Path | Type | Default | Description |
|---------------------|-----------|------|---------|-------------|
| `NATS_MAX_MEMORY` | `nats.max_memory` | int64 | `1073741824` | Max memory (1GB) |
| `NATS_MAX_STORE` | `nats.max_store` | int64 | `10737418240` | Max disk (10GB) |
| `NATS_RETENTION_DAYS` | `nats.stream_retention_days` | int | `7` | Event retention |
| `NATS_BATCH_SIZE` | `nats.batch_size` | int | `1000` | Batch write size |
| `NATS_FLUSH_INTERVAL` | `nats.flush_interval` | duration | `5s` | Max batch wait |
| `NATS_SUBSCRIBERS` | `nats.subscribers_count` | int | `4` | Parallel processors |
| `NATS_DURABLE_NAME` | `nats.durable_name` | string | `media-processor` | Consumer name |
| `NATS_QUEUE_GROUP` | `nats.queue_group` | string | `processors` | Queue group |

#### Router Middleware

| Environment Variable | YAML Path | Type | Default | Description |
|---------------------|-----------|------|---------|-------------|
| `NATS_ROUTER_RETRY_COUNT` | `nats.router_retry_count` | int | `3` | Message retry count |
| `NATS_ROUTER_RETRY_INTERVAL` | `nats.router_retry_initial_interval` | duration | `100ms` | Initial retry delay |
| `NATS_ROUTER_THROTTLE` | `nats.router_throttle_per_second` | int | `0` | Rate limit (0=unlimited) |
| `NATS_ROUTER_POISON_ENABLED` | `nats.router_poison_queue_enabled` | boolean | `true` | Dead letter queue |
| `NATS_ROUTER_POISON_TOPIC` | `nats.router_poison_queue_topic` | string | `playback.poison` | DLQ topic |
| `NATS_ROUTER_CLOSE_TIMEOUT` | `nats.router_close_timeout` | duration | `30s` | Shutdown timeout |

---

### Write-Ahead Log Configuration

BadgerDB durability layer for event publishing.

| Environment Variable | YAML Path | Type | Default | Description |
|---------------------|-----------|------|---------|-------------|
| `WAL_ENABLED` | `wal.enabled` | boolean | `true` | Enable WAL |
| `WAL_PATH` | `wal.path` | string | `/data/wal` | Storage directory |
| `WAL_SYNC_WRITES` | `wal.sync_writes` | boolean | `true` | Fsync every write |
| `WAL_RETRY_INTERVAL` | `wal.retry_interval` | duration | `30s` | Retry loop interval |
| `WAL_MAX_RETRIES` | `wal.max_retries` | int | `100` | Max retry attempts |
| `WAL_RETRY_BACKOFF` | `wal.retry_backoff` | duration | `5s` | Initial backoff |
| `WAL_COMPACT_INTERVAL` | `wal.compact_interval` | duration | `1h` | Compaction interval |
| `WAL_ENTRY_TTL` | `wal.entry_ttl` | duration | `168h` | Entry expiration |
| `WAL_MEMTABLE_SIZE` | `wal.memtable_size` | int64 | `16777216` | Memtable size (16MB) |
| `WAL_VLOG_SIZE` | `wal.vlog_size` | int64 | `67108864` | Value log size (64MB) |
| `WAL_NUM_COMPACTORS` | `wal.num_compactors` | int | `2` | Compaction workers |

---

### Backup Configuration

Automated backup and retention settings.

#### Core Settings

| Environment Variable | YAML Path | Type | Default | Description |
|---------------------|-----------|------|---------|-------------|
| `BACKUP_ENABLED` | `backup.enabled` | boolean | `true` | Enable backups |
| `BACKUP_DIR` | `backup.dir` | string | `/data/backups` | Backup directory |
| `BACKUP_TYPE` | `backup.type` | string | `full` | Type: `full`, `database`, `config` |

#### Schedule

| Environment Variable | YAML Path | Type | Default | Description |
|---------------------|-----------|------|---------|-------------|
| `BACKUP_SCHEDULE_ENABLED` | `backup.schedule_enabled` | boolean | `true` | Enable auto backup |
| `BACKUP_INTERVAL` | `backup.interval` | duration | `24h` | Backup frequency |
| `BACKUP_PREFERRED_HOUR` | `backup.preferred_hour` | int | `2` | Preferred hour (0-23) |
| `BACKUP_PRE_SYNC` | `backup.pre_sync` | boolean | `false` | Backup before sync |

#### Retention

| Environment Variable | YAML Path | Type | Default | Description |
|---------------------|-----------|------|---------|-------------|
| `BACKUP_RETENTION_MIN_COUNT` | `backup.retention.min_count` | int | `3` | Minimum backups |
| `BACKUP_RETENTION_MAX_COUNT` | `backup.retention.max_count` | int | `50` | Maximum backups |
| `BACKUP_RETENTION_MAX_DAYS` | `backup.retention.max_days` | int | `90` | Maximum age |
| `BACKUP_RETENTION_KEEP_RECENT_HOURS` | `backup.retention.keep_recent_hours` | int | `24` | Protect recent |
| `BACKUP_RETENTION_KEEP_DAILY_DAYS` | `backup.retention.keep_daily_days` | int | `7` | Keep daily |
| `BACKUP_RETENTION_KEEP_WEEKLY_WEEKS` | `backup.retention.keep_weekly_weeks` | int | `4` | Keep weekly |
| `BACKUP_RETENTION_KEEP_MONTHLY_MONTHS` | `backup.retention.keep_monthly_months` | int | `6` | Keep monthly |

#### Compression & Encryption

| Environment Variable | YAML Path | Type | Default | Description |
|---------------------|-----------|------|---------|-------------|
| `BACKUP_COMPRESSION_ENABLED` | `backup.compression.enabled` | boolean | `true` | Enable compression |
| `BACKUP_COMPRESSION_LEVEL` | `backup.compression.level` | int | `6` | Level (1-9) |
| `BACKUP_COMPRESSION_ALGORITHM` | `backup.compression.algorithm` | string | `gzip` | Algorithm |
| `BACKUP_ENCRYPTION_ENABLED` | `backup.encryption.enabled` | boolean | `false` | Enable encryption |
| `BACKUP_ENCRYPTION_KEY` | `backup.encryption.key` | string | `""` | AES-256 key (32+ chars) |

#### Notifications

| Environment Variable | YAML Path | Type | Default | Description |
|---------------------|-----------|------|---------|-------------|
| `BACKUP_NOTIFY_SUCCESS` | `backup.notify.success` | boolean | `false` | Notify on success |
| `BACKUP_NOTIFY_FAILURE` | `backup.notify.failure` | boolean | `true` | Notify on failure |
| `BACKUP_NOTIFY_CLEANUP` | `backup.notify.cleanup` | boolean | `false` | Notify on cleanup |
| `BACKUP_WEBHOOK_URL` | `backup.webhook_url` | string | `""` | Notification webhook |

---

### Detection Engine Configuration

Security anomaly detection settings.

| Environment Variable | YAML Path | Type | Default | Description |
|---------------------|-----------|------|---------|-------------|
| `DETECTION_ENABLED` | `detection.enabled` | boolean | `true` | Enable detection |
| `DETECTION_TRUST_SCORE_DECREMENT` | `detection.trust_score_decrement` | int | `10` | Score penalty |
| `DETECTION_TRUST_SCORE_RECOVERY` | `detection.trust_score_recovery` | int | `1` | Daily recovery |
| `DETECTION_TRUST_SCORE_THRESHOLD` | `detection.trust_score_threshold` | int | `50` | Restriction threshold |

---

### Notification Configuration

Alert notification settings.

#### Discord

| Environment Variable | YAML Path | Type | Default | Description |
|---------------------|-----------|------|---------|-------------|
| `DISCORD_WEBHOOK_ENABLED` | `discord.enabled` | boolean | `false` | Enable Discord |
| `DISCORD_WEBHOOK_URL` | `discord.webhook_url` | string | `""` | Webhook URL |
| `DISCORD_RATE_LIMIT_MS` | `discord.rate_limit_ms` | int | `1000` | Rate limit (ms) |

#### Generic Webhook

| Environment Variable | YAML Path | Type | Default | Description |
|---------------------|-----------|------|---------|-------------|
| `WEBHOOK_ENABLED` | `webhook.enabled` | boolean | `false` | Enable webhook |
| `WEBHOOK_URL` | `webhook.url` | string | `""` | Target URL |
| `WEBHOOK_RATE_LIMIT_MS` | `webhook.rate_limit_ms` | int | `500` | Rate limit (ms) |
| `WEBHOOK_HEADERS` | `webhook.headers` | string | `""` | Custom headers (key=value,key=value) |

---

### Logging Configuration

Structured logging settings (zerolog).

| Environment Variable | YAML Path | Type | Default | Description |
|---------------------|-----------|------|---------|-------------|
| `LOG_LEVEL` | `logging.level` | string | `info` | Level: `trace`, `debug`, `info`, `warn`, `error` |
| `LOG_FORMAT` | `logging.format` | string | `json` | Format: `json`, `console` |
| `LOG_CALLER` | `logging.caller` | boolean | `false` | Include file:line |

---

### GeoIP Configuration

Geolocation services for standalone mode.

| Environment Variable | YAML Path | Type | Default | Description |
|---------------------|-----------|------|---------|-------------|
| `GEOIP_PROVIDER` | `geoip.provider` | string | `""` | Provider: `maxmind`, `ipapi`, or auto |
| `MAXMIND_ACCOUNT_ID` | `geoip.maxmind_account_id` | string | `""` | MaxMind account ID |
| `MAXMIND_LICENSE_KEY` | `geoip.maxmind_license_key` | string | `""` | MaxMind license key |

---

### VPN Detection Configuration

Identifies connections from known VPN providers for accurate geolocation.

| Environment Variable | YAML Path | Type | Default | Description |
|---------------------|-----------|------|---------|-------------|
| `VPN_ENABLED` | `vpn.enabled` | boolean | `true` | Enable VPN detection |
| `VPN_DATA_FILE` | `vpn.data_file` | string | `""` | Path to gluetun servers.json |
| `VPN_CACHE_SIZE` | `vpn.cache_size` | int | `10000` | Lookup cache entries |
| `VPN_AUTO_UPDATE` | `vpn.auto_update` | boolean | `false` | Auto-update data (future) |
| `VPN_UPDATE_INTERVAL` | `vpn.update_interval` | duration | `24h` | Update check interval |

---

### Import Configuration

Direct import from Tautulli SQLite database files.

| Environment Variable | YAML Path | Type | Default | Description |
|---------------------|-----------|------|---------|-------------|
| `IMPORT_ENABLED` | `import.enabled` | boolean | `false` | Enable import functionality |
| `IMPORT_DB_PATH` | `import.db_path` | string | `""` | Path to tautulli.db file |
| `IMPORT_BATCH_SIZE` | `import.batch_size` | int | `1000` | Records per batch (1-10000) |
| `IMPORT_DRY_RUN` | `import.dry_run` | boolean | `false` | Validate without importing |
| `IMPORT_AUTO_START` | `import.auto_start` | boolean | `false` | Start import on startup |
| `IMPORT_RESUME_FROM_ID` | `import.resume_from_id` | int | `0` | Resume from session ID |
| `IMPORT_SKIP_GEOLOCATION` | `import.skip_geolocation` | boolean | `false` | Skip GeoIP enrichment |

---

### Newsletter Configuration

Automated digest and newsletter delivery.

| Environment Variable | YAML Path | Type | Default | Description |
|---------------------|-----------|------|---------|-------------|
| `NEWSLETTER_ENABLED` | `newsletter.enabled` | boolean | `false` | Enable newsletter scheduler |
| `NEWSLETTER_CHECK_INTERVAL` | `newsletter.check_interval` | duration | `1m` | Schedule check frequency |
| `NEWSLETTER_MAX_CONCURRENT` | `newsletter.max_concurrent` | int | `5` | Concurrent newsletter jobs |
| `NEWSLETTER_EXEC_TIMEOUT` | `newsletter.exec_timeout` | duration | `5m` | Job execution timeout |

---

### Environment Mode Configuration

Controls security behaviors and production safeguards.

| Environment Variable | YAML Path | Type | Default | Description |
|---------------------|-----------|------|---------|-------------|
| `ENVIRONMENT` | `server.environment` | string | `development` | Mode: `development`, `staging`, `production` |

**Production Mode Safeguards:**
- `AUTH_MODE=none` is **rejected** (authentication required)
- `CORS_ORIGINS=*` is **rejected** when authentication is enabled
- Stricter validation of security settings

---

### Recommendation Engine Configuration

Content recommendation settings (ADR-0024).

#### Core Settings

| Environment Variable | YAML Path | Type | Default | Description |
|---------------------|-----------|------|---------|-------------|
| `RECOMMEND_ENABLED` | `recommend.enabled` | boolean | `false` | Enable engine |
| `RECOMMEND_TRAIN_INTERVAL` | `recommend.train_interval` | duration | `24h` | Training frequency |
| `RECOMMEND_TRAIN_ON_STARTUP` | `recommend.train_on_startup` | boolean | `false` | Train on start |
| `RECOMMEND_MIN_INTERACTIONS` | `recommend.min_interactions` | int | `100` | Min data for training |
| `RECOMMEND_MODEL_PATH` | `recommend.model_path` | string | `/data/recommend` | Model storage |
| `RECOMMEND_ALGORITHMS` | `recommend.algorithms` | []string | `["covisit","content"]` | Enabled algorithms |
| `RECOMMEND_CACHE_TTL` | `recommend.cache_ttl` | duration | `5m` | Result cache TTL |
| `RECOMMEND_MAX_CANDIDATES` | `recommend.max_candidates` | int | `1000` | Max candidates |
| `RECOMMEND_DIVERSITY_LAMBDA` | `recommend.diversity_lambda` | float | `0.7` | Diversity factor |
| `RECOMMEND_CALIBRATION_ENABLED` | `recommend.calibration_enabled` | boolean | `true` | Enable calibration |

#### Algorithm Settings

See `config.yaml.example` for detailed algorithm-specific settings (EASE, ALS, KNN, FPMC, LinUCB).

---

## Multi-Server Configuration

For multiple servers per platform, use YAML configuration:

```yaml
jellyfin_servers:
  - enabled: true
    server_id: jellyfin-home
    url: http://192.168.1.100:8096
    api_key: api-key-1
    realtime_enabled: true

  - enabled: true
    server_id: jellyfin-office
    url: http://192.168.1.200:8096
    api_key: api-key-2
    realtime_enabled: true

emby_servers:
  - enabled: true
    server_id: emby-basement
    url: http://192.168.1.150:8096
    api_key: emby-key
```

---

## Duration Format

Duration values support Go duration format:
- `5s` - 5 seconds
- `5m` - 5 minutes
- `1h` - 1 hour
- `24h` - 24 hours
- `7d` - 7 days (some settings)
- `1h30m` - 1 hour 30 minutes

---

## Complete YAML Example

See [config.yaml.example](../config.yaml.example) for a complete example.

---

## Environment Variable to YAML Mapping

All environment variables map to nested YAML paths. The general pattern is:

```
SECTION_SETTING -> section.setting
SECTION_SUBSECTION_SETTING -> section.subsection.setting
```

For the complete mapping, see the `envTransformFunc` in `internal/config/koanf.go`.
