# Frequently Asked Questions

**[Home](Home)** | **[Troubleshooting](Troubleshooting)** | **FAQ**

---

## General Questions

### What is Cartographus?

Cartographus is a data analytics and geographic visualization platform for self-hosted media servers. It connects to Plex, Jellyfin, and Emby to collect playback data and present it through interactive maps, charts, and dashboards.

### How is this different from Tautulli?

| Feature | Cartographus | Tautulli |
|---------|--------------|----------|
| Geographic visualization | WebGL maps + 3D globe | Basic map |
| Multi-server support | Plex, Jellyfin, Emby simultaneously | Plex only |
| Cross-server deduplication | Automatic | N/A |
| Security detection | 7 detection rules | Limited |
| API endpoints | 302 | ~50 |
| Database | DuckDB (analytics-optimized) | SQLite |

Cartographus can import your existing Tautulli data to preserve historical records.

### Is my data sent anywhere?

No. Cartographus runs entirely on your server. There are no external API calls, telemetry, or cloud dependencies. Your data never leaves your network.

### What architectures are supported?

- **x86_64 (amd64)**: Full support
- **ARM64 (aarch64)**: Full support (Raspberry Pi 4+, Apple Silicon)
- **ARMv7**: Not supported (DuckDB limitation)

---

## Installation Questions

### What are the minimum requirements?

- 1 GB RAM
- 500 MB disk space (plus database growth)
- Docker 20.10+ (recommended)
- Network access to your media server(s)

### Can I run Cartographus without Docker?

Yes. You can build from source with Go 1.24+ and run the binary directly. See [Installation](Installation#binary-installation).

### Does Cartographus need to be on the same server as Plex/Jellyfin/Emby?

No. Cartographus can run anywhere that has network access to your media server(s). Many users run it on a separate VM or Raspberry Pi.

### Can I run multiple instances?

The database is single-writer, so only one instance should run at a time. For high availability, use Kubernetes with a single replica.

---

## Media Server Questions

### Can I connect multiple Plex servers?

Yes. Use unique `PLEX_SERVER_ID` values for each server. For more than one Plex server, use YAML configuration. See [Media Servers](Media-Servers#multi-server-setup).

### Can I connect Plex AND Jellyfin at the same time?

Yes. Cartographus supports connecting all three platforms (Plex, Jellyfin, Emby) simultaneously and will automatically deduplicate users who appear on multiple servers.

### Do I need Tautulli?

No. Cartographus connects directly to your media servers. Tautulli integration is optional and only used for importing historical data.

### How do I get my Plex token?

See [Media Servers](Media-Servers#getting-your-plex-token) for detailed instructions.

### Why isn't real-time working?

Ensure you've enabled the real-time setting for your server:

```yaml
ENABLE_PLEX_REALTIME=true
JELLYFIN_REALTIME_ENABLED=true
EMBY_REALTIME_ENABLED=true
```

Also verify your reverse proxy supports WebSocket connections.

---

## Data & Analytics Questions

### How long does initial sync take?

Typically 2-5 minutes for a standard library. Large libraries with years of history may take longer.

### How far back does data go?

By default, initial sync looks back 24 hours. Configure `SYNC_LOOKBACK` for more history, or use Tautulli import for years of historical data.

### How accurate is geolocation?

Geolocation is based on IP address and is typically accurate to the city level. VPN users will show their VPN exit location.

### Can I export my data?

Yes. Export in CSV, GeoJSON, or GeoParquet formats via the UI or API.

### What happens if I delete Cartographus?

Your data is stored in the `/data` volume. As long as you preserve this volume, you can restore your data by pointing a new instance at the same volume.

---

## Security Questions

### Is authentication required?

Yes, by default. `AUTH_MODE=jwt` is the default and requires credentials. You can use `AUTH_MODE=none` for development only (not recommended for production).

### Can I use my existing identity provider?

Yes. Cartographus supports OIDC authentication with any compatible provider (Authelia, Authentik, Keycloak, Okta, etc.). See [Authentication](Authentication).

### What is "Sign in with Plex"?

Plex authentication allows users to log in with their Plex credentials. Server owners automatically get admin access, while shared users get viewer access. See [Authentication](Authentication#plex-authentication).

### How does security detection work?

Cartographus monitors for suspicious patterns like:
- Impossible travel (streaming from distant locations too quickly)
- Too many concurrent streams
- Device velocity anomalies
- Geographic restrictions

See [Security Detection](Security-Detection) for details.

---

## Performance Questions

### How much disk space does the database use?

Depends on your library size and history:
- Small (< 1000 plays): ~50 MB
- Medium (< 10,000 plays): ~200 MB
- Large (< 100,000 plays): ~1 GB
- Very large (1M+ plays): ~10 GB

### How much RAM does Cartographus use?

Base memory usage is ~200-300 MB. DuckDB uses additional memory for queries (configurable via `DUCKDB_MAX_MEMORY`, default 2 GB).

### Can it handle large libraries?

Yes. DuckDB is designed for analytics workloads. Features like cursor pagination and approximate analytics (HyperLogLog, KLL) ensure consistent performance with millions of records.

### Why are maps slow?

Check:
1. WebGL is enabled in your browser
2. Hardware acceleration is on
3. You're using a supported browser (Chrome, Firefox, Safari)

---

## Backup & Recovery Questions

### Does Cartographus back up automatically?

Yes, if enabled. Configure with:

```yaml
BACKUP_ENABLED=true
BACKUP_INTERVAL=24h
```

### How do I restore from backup?

1. Stop Cartographus
2. Replace database file with backup
3. Start Cartographus

### Where are backups stored?

By default: `/data/backups/` (inside the data volume)

---

## Updating Questions

### How do I update Cartographus?

```bash
docker-compose pull
docker-compose up -d
```

### Are updates backward compatible?

Yes. Database migrations run automatically on startup. Always back up before major version updates.

### How do I check the current version?

```bash
curl http://localhost:3857/api/v1/health
```

---

## API Questions

### Is there an API?

Yes. 302 REST API endpoints covering all features. See the [API Reference](https://github.com/tomtom215/cartographus/blob/main/docs/API-REFERENCE.md).

### Is the API authenticated?

Yes. Use the same authentication method configured for the UI (JWT tokens, Basic auth, etc.).

### Can I use the API for automation?

Yes. Common uses include:
- Custom dashboards
- Discord/Slack bots
- Integration with other homelab tools

---

## Getting More Help

- **[Troubleshooting Guide](Troubleshooting)** - Common issues and solutions
- **[GitHub Issues](https://github.com/tomtom215/cartographus/issues)** - Bug reports and feature requests
- **[GitHub Discussions](https://github.com/tomtom215/cartographus/discussions)** - Community Q&A
