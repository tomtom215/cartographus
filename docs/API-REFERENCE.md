# API Reference

Complete API documentation for Cartographus. Base URL: `http://localhost:3857`

**Related Documentation**:
- [README.md](../README.md) - Quick start and configuration
- [ARCHITECTURE.md](./ARCHITECTURE.md) - System architecture

---

## Table of Contents

1. [Authentication](#authentication)
2. [Core Endpoints](#core-endpoints)
3. [Analytics Endpoints](#analytics-endpoints)
   - [Native Analytics](#native-analytics-duckdb)
   - [Approximate Analytics](#approximate-analytics-datasketches)
   - [Fuzzy Search](#fuzzy-search-rapidfuzz)
   - [Cross-Platform Analytics](#cross-platform-analytics)
4. [Cross-Platform Endpoints](#cross-platform-endpoints)
   - [Content Mapping](#content-mapping)
   - [User Linking](#user-linking)
5. [Tautulli Proxy Endpoints](#tautulli-proxy-endpoints)
6. [Export Endpoints](#export-endpoints)
7. [Real-Time Endpoints](#real-time-endpoints)
8. [Import Endpoints](#import-endpoints)
9. [Data Sync Endpoints](#data-sync-endpoints)
10. [Server Management Endpoints](#server-management-endpoints)
11. [Query Parameters](#query-parameters)
12. [Response Format](#response-format)

---

## Authentication

### Modes

| Mode | Description | Configuration |
|------|-------------|---------------|
| JWT | Token-based with sessions | `AUTH_MODE=jwt`, `JWT_SECRET` required |
| OIDC | OpenID Connect (Zitadel) | `AUTH_MODE=oidc`, `OIDC_*` vars required |
| Plex | Plex OAuth 2.0 | `AUTH_MODE=plex`, Plex credentials required |
| Multi | Multiple methods (OIDC, Plex, JWT) | `AUTH_MODE=multi` |
| Basic | HTTP Basic Auth | `AUTH_MODE=basic` |
| None | No authentication (dev only) | `AUTH_MODE=none` |

### Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/auth/login` | POST | Login (JWT mode) |
| `/api/v1/auth/plex/start` | GET | Initiate Plex OAuth PKCE flow |
| `/api/v1/auth/plex/callback` | GET | OAuth callback handler |
| `/api/v1/auth/plex/refresh` | POST | Refresh access token |
| `/api/v1/auth/plex/revoke` | POST | Revoke token (logout) |
| `/api/v1/auth/oidc/login` | GET | Initiate OIDC login flow (Zitadel) |
| `/api/v1/auth/oidc/callback` | GET | OIDC callback handler |
| `/api/v1/auth/oidc/logout` | POST | OIDC logout (RP-initiated) |
| `/api/v1/auth/oidc/refresh` | POST | Refresh OIDC tokens |
| `/api/v1/auth/oidc/backchannel-logout` | POST | Back-channel logout (IdP-initiated) |
| `/api/v1/auth/userinfo` | GET | Current user information |
| `/api/v1/auth/session` | GET | Current session status |

---

## Core Endpoints

| Endpoint | Method | Auth | Description |
|----------|--------|------|-------------|
| `/api/v1/health` | GET | No | Health check |
| `/api/v1/health/live` | GET | No | Kubernetes liveness probe |
| `/api/v1/health/ready` | GET | No | Kubernetes readiness probe |
| `/api/v1/stats` | GET | No | Overall statistics |
| `/api/v1/playbacks` | GET | No | Paginated playback history |
| `/api/v1/locations` | GET | No | Geographic aggregations (GeoJSON) |
| `/api/v1/users` | GET | No | Users with playback history |
| `/api/v1/media-types` | GET | No | Available media types |
| `/api/v1/sync` | POST | Yes | Trigger manual sync |

---

## Analytics Endpoints

All analytics endpoints support filter parameters (see [Query Parameters](#query-parameters)).

### Native Analytics (DuckDB)

| Endpoint | Description |
|----------|-------------|
| `/api/v1/analytics/trends` | Playback trends with auto-interval detection |
| `/api/v1/analytics/geographic` | Full geographic analytics (16 visualizations) |
| `/api/v1/analytics/users` | User activity leaderboard |
| `/api/v1/analytics/binge` | Binge-watching detection (3+ episodes/6h) |
| `/api/v1/analytics/bandwidth` | Bandwidth consumption analysis |
| `/api/v1/analytics/bitrate` | 3-level bitrate tracking |
| `/api/v1/analytics/popular` | Top movies, shows, episodes |
| `/api/v1/analytics/watch-parties` | Group viewing detection (15-min window) |
| `/api/v1/analytics/user-engagement` | User behavior metrics |
| `/api/v1/analytics/comparative` | Period-over-period comparison |
| `/api/v1/analytics/temporal-heatmap` | Geographic density over time |
| `/api/v1/analytics/resolution-mismatch` | Quality downgrade detection |
| `/api/v1/analytics/hdr` | HDR content distribution |
| `/api/v1/analytics/audio` | Audio format analysis |
| `/api/v1/analytics/subtitles` | Subtitle usage patterns |
| `/api/v1/analytics/frame-rate` | Video frame rate distribution |
| `/api/v1/analytics/container` | Container format analytics |
| `/api/v1/analytics/connection-security` | Secure vs insecure connections |
| `/api/v1/analytics/pause-patterns` | Content engagement analysis |
| `/api/v1/analytics/library` | Per-library statistics |
| `/api/v1/analytics/abandonment` | Content drop-off analysis |
| `/api/v1/analytics/concurrent-streams` | Peak concurrent stream analysis |
| `/api/v1/analytics/hardware-transcode` | Hardware transcoding statistics |
| `/api/v1/analytics/hardware-transcode/trends` | Hardware transcoding trends over time |
| `/api/v1/analytics/hdr-content` | HDR content availability and playback |

### Enhanced Analytics (Production-Grade Insights)

These endpoints provide advanced analytics for production monitoring and business intelligence.

| Endpoint | Description |
|----------|-------------|
| `/api/v1/analytics/cohort-retention` | Cohort retention analysis by user signup date |
| `/api/v1/analytics/qoe` | Quality of Experience dashboard (buffering, bitrate, errors) |
| `/api/v1/analytics/data-quality` | Data quality monitoring (missing fields, anomalies) |
| `/api/v1/analytics/user-network` | User relationship network (shared devices, IPs) |
| `/api/v1/analytics/device-migration` | Device/platform migration tracking |
| `/api/v1/analytics/content-discovery` | Content discovery and time-to-first-watch metrics |

### Approximate Analytics (DataSketches)

These endpoints use DataSketches HyperLogLog and KLL sketches for O(1) space approximate calculations on large datasets. Falls back to exact calculations when extension is unavailable.

| Endpoint | Description |
|----------|-------------|
| `/api/v1/analytics/approximate` | Dashboard-level approximate stats (unique users, titles, percentiles) |
| `/api/v1/analytics/approximate/distinct` | Approximate distinct count for any column (HyperLogLog) |
| `/api/v1/analytics/approximate/percentile` | Approximate percentile for numeric columns (KLL) |

**Query Parameters for `/api/v1/analytics/approximate/distinct`**:
- `column`: Column name (username, title, ip_address, platform, player, rating_key, media_type, location_type)

**Query Parameters for `/api/v1/analytics/approximate/percentile`**:
- `column`: Column name (play_duration, percent_complete, paused_counter)
- `percentile`: Value between 0 and 1 (e.g., 0.50 for median, 0.95 for P95)

### Fuzzy Search (RapidFuzz)

These endpoints use RapidFuzz for fuzzy string matching with Levenshtein distance and Jaro-Winkler similarity.

| Endpoint | Description |
|----------|-------------|
| `/api/v1/search/fuzzy` | Fuzzy title search with configurable similarity threshold |
| `/api/v1/users/deduplicate` | Find potential duplicate users by name similarity |
| `/api/v1/titles/similar` | Find similar titles for content recommendation |

### Cross-Platform Analytics

These endpoints provide aggregated analytics across linked users and content from multiple media servers.

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/analytics/cross-platform/user/{id}` | GET | Aggregated watch stats for linked users |
| `/api/v1/analytics/cross-platform/content/{id}` | GET | Watch stats for mapped content |
| `/api/v1/analytics/cross-platform/summary` | GET | Overall cross-platform usage summary |

**User Stats Response**:
```json
{
  "success": true,
  "user_id": 1,
  "linked_user_ids": [1, 2, 3],
  "total_plays": 150,
  "total_duration": 54000,
  "platforms_used": {"plex": 2, "jellyfin": 1},
  "linked_identities": 3
}
```

**Content Stats Response**:
```json
{
  "success": true,
  "content_mapping_id": 1,
  "title": "Movie Title",
  "media_type": "movie",
  "year": 2024,
  "total_plays": 25,
  "platforms_available": {"plex": true, "jellyfin": true},
  "external_ids": {"imdb": "tt1234567", "tmdb": 12345}
}
```

---

## Spatial Endpoints

These endpoints provide geographic data visualization using DuckDB's spatial and H3 extensions.

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/spatial/hexagons` | GET | H3 hexagon aggregation for heatmaps |
| `/api/v1/spatial/arcs` | GET | Distance-weighted arcs from server to users |
| `/api/v1/spatial/viewport` | GET | Locations within bounding box (map viewport) |
| `/api/v1/spatial/temporal-density` | GET | Spatial density over time periods |
| `/api/v1/spatial/nearby` | GET | Locations within radius of coordinates |

### Hexagon Aggregation

**GET** `/api/v1/spatial/hexagons`

Returns H3 hexagon aggregations for geographic heatmap visualization.

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `resolution` | integer | 7 | H3 resolution (4-10, lower = larger hexagons) |
| `limit` | integer | 1000 | Maximum hexagons to return |

**Response**:
```json
{
  "status": "success",
  "data": {
    "hexagons": [
      {
        "h3_index": "871f1c9c1ffffff",
        "lat": 40.7128,
        "lng": -74.0060,
        "count": 150,
        "total_duration": 45000
      }
    ],
    "resolution": 7
  }
}
```

### Arc Visualization

**GET** `/api/v1/spatial/arcs`

Returns distance-weighted arcs from server location to viewer locations for 3D globe visualization.

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `server_lat` | float | (config) | Server latitude |
| `server_lon` | float | (config) | Server longitude |

**Response**:
```json
{
  "status": "success",
  "data": {
    "arcs": [
      {
        "source": {"lat": 40.7128, "lng": -74.0060},
        "target": {"lat": 51.5074, "lng": -0.1278},
        "weight": 25,
        "distance_km": 5570.5
      }
    ],
    "server_location": {"lat": 40.7128, "lng": -74.0060}
  }
}
```

### Viewport Query

**GET** `/api/v1/spatial/viewport`

Returns locations within a map bounding box, optimized for map viewport queries.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `west` | float | Yes | Western longitude boundary |
| `south` | float | Yes | Southern latitude boundary |
| `east` | float | Yes | Eastern longitude boundary |
| `north` | float | Yes | Northern latitude boundary |

### Nearby Search

**GET** `/api/v1/spatial/nearby`

Returns locations within a radius of specified coordinates.

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `lat` | float | Required | Center latitude |
| `lon` | float | Required | Center longitude |
| `radius_km` | float | 100 | Search radius in kilometers |

---

## Cross-Platform Endpoints

These endpoints enable content reconciliation and user identity linking across Plex, Jellyfin, and Emby servers.

### Content Mapping

Link and lookup content across platforms using external IDs (IMDb, TMDB, TVDB).

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/content/link` | POST | Create/update content mapping |
| `/api/v1/content/lookup` | GET | Lookup content by external ID |
| `/api/v1/content/{id}/link/plex` | POST | Link Plex rating_key to mapping |
| `/api/v1/content/{id}/link/jellyfin` | POST | Link Jellyfin item ID to mapping |
| `/api/v1/content/{id}/link/emby` | POST | Link Emby item ID to mapping |

**Create Content Mapping (POST `/api/v1/content/link`)**:
```json
{
  "imdb_id": "tt1234567",
  "tmdb_id": 12345,
  "tvdb_id": null,
  "plex_rating_key": "abc123",
  "jellyfin_item_id": "uuid-string",
  "emby_item_id": null,
  "title": "Movie Title",
  "media_type": "movie",
  "year": 2024
}
```

**Lookup Content (GET `/api/v1/content/lookup`)**:
- `type`: ID type (`imdb`, `tmdb`, `tvdb`, `plex`, `jellyfin`, `emby`)
- `id`: The external ID value

**Link Platform Content (POST `/api/v1/content/{id}/link/plex`)**:
```json
{"rating_key": "12345"}
```

**Link Jellyfin/Emby Content**:
```json
{"item_id": "uuid-or-id-string"}
```

### User Linking

Link user identities across platforms for unified analytics.

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/users/link` | POST | Create user link |
| `/api/v1/users/{id}/linked` | GET | Get all linked users |
| `/api/v1/users/link` | DELETE | Remove user link |
| `/api/v1/users/suggest-links` | GET | Suggest links by email matching |

**Create User Link (POST `/api/v1/users/link`)**:
```json
{
  "primary_user_id": 1,
  "linked_user_id": 2,
  "link_type": "manual"
}
```

**Link Types**: `manual`, `email`, `plex_home`

**Delete User Link (DELETE `/api/v1/users/link`)**:
- `primary_id`: First user's internal ID
- `linked_id`: Second user's internal ID

**Get Linked Users Response**:
```json
{
  "success": true,
  "user_ids": [1, 2, 3],
  "users": [
    {
      "internal_user_id": 1,
      "username": "john",
      "source": "plex",
      "server_id": "abc123"
    }
  ]
}
```

---

## Tautulli Proxy Endpoints

These endpoints proxy Tautulli's pre-calculated analytics. All use GET method and have 5-minute cache TTL.

### Activity & Status

| Endpoint | Parameters | Description |
|----------|------------|-------------|
| `/api/v1/tautulli/activity` | `session_key` | Current "Now Playing" sessions |
| `/api/v1/tautulli/server-info` | - | Plex server details |
| `/api/v1/tautulli/tautulli-info` | - | Tautulli system info |
| `/api/v1/tautulli/pms-update` | - | PMS update availability |
| `/api/v1/tautulli/terminate-session` | `session_id`, `message` | Terminate streaming session |

### Statistics

| Endpoint | Parameters | Description |
|----------|------------|-------------|
| `/api/v1/tautulli/home-stats` | `time_range`, `stats_type`, `stats_count` | Top content/users/platforms |
| `/api/v1/tautulli/plays-by-date` | `time_range`, `y_axis`, `user_id` | Time series by date |
| `/api/v1/tautulli/plays-by-dayofweek` | `time_range`, `y_axis`, `user_id` | Day of week patterns |
| `/api/v1/tautulli/plays-by-hourofday` | `time_range`, `y_axis`, `user_id` | Hour of day patterns |
| `/api/v1/tautulli/plays-by-stream-type` | `time_range`, `y_axis`, `user_id` | Direct Play/Transcode trends |
| `/api/v1/tautulli/concurrent-streams-by-stream-type` | `time_range`, `user_id` | Peak concurrent streams |
| `/api/v1/tautulli/plays-per-month` | `time_range`, `y_axis`, `user_id` | Monthly trends |

### Content

| Endpoint | Parameters | Description |
|----------|------------|-------------|
| `/api/v1/tautulli/metadata` | `rating_key` | Rich content metadata |
| `/api/v1/tautulli/recently-added` | `count`, `start`, `media_type`, `section_id` | Latest library additions |
| `/api/v1/tautulli/children-metadata` | `rating_key`, `media_type` | Episodes/tracks |
| `/api/v1/tautulli/item-watch-time-stats` | `rating_key`, `query_days` | Item viewing duration |
| `/api/v1/tautulli/item-user-stats` | `rating_key` | Who watched what |

### Libraries

| Endpoint | Parameters | Description |
|----------|------------|-------------|
| `/api/v1/tautulli/libraries` | - | All library sections |
| `/api/v1/tautulli/library` | `section_id` | Specific library details |
| `/api/v1/tautulli/library-names` | - | Section ID to name mapping |
| `/api/v1/tautulli/library-user-stats` | `section_id` | Per-library user engagement |
| `/api/v1/tautulli/library-media-info` | `section_id`, pagination | Library content details |
| `/api/v1/tautulli/library-watch-time-stats` | `section_id`, `query_days` | Library watch statistics |
| `/api/v1/tautulli/libraries-table` | pagination, sorting | Paginated library data |

### Users

| Endpoint | Parameters | Description |
|----------|------------|-------------|
| `/api/v1/tautulli/user` | `user_id` | User profile details |
| `/api/v1/tautulli/user-player-stats` | `user_id` | User platform preferences |
| `/api/v1/tautulli/user-watch-time-stats` | `user_id`, `query_days` | User engagement metrics |
| `/api/v1/tautulli/user-ips` | `user_id` | User IP history |
| `/api/v1/tautulli/user-logins` | `user_id`, pagination | Login history |
| `/api/v1/tautulli/users-table` | pagination, sorting | Paginated user data |

### Leaderboards

| Endpoint | Description |
|----------|-------------|
| `/api/v1/tautulli/plays-by-top-10-platforms` | Top 10 client platforms |
| `/api/v1/tautulli/plays-by-top-10-users` | Top 10 users |
| `/api/v1/tautulli/stream-type-by-top-10-users` | User quality preferences |
| `/api/v1/tautulli/stream-type-by-top-10-platforms` | Platform transcoding patterns |
| `/api/v1/tautulli/plays-by-source-resolution` | Source quality distribution |
| `/api/v1/tautulli/plays-by-stream-resolution` | Delivered quality distribution |

### Other

| Endpoint | Parameters | Description |
|----------|------------|-------------|
| `/api/v1/tautulli/search` | `query`, `limit` | Global search |
| `/api/v1/tautulli/stream-data` | `row_id`, `session_key` | Stream quality details |
| `/api/v1/tautulli/synced-items` | `machine_id`, `user_id` | Offline sync status |
| `/api/v1/tautulli/collections-table` | `section_id`, pagination | Collections management |
| `/api/v1/tautulli/playlists-table` | `section_id`, pagination | Playlists management |
| `/api/v1/tautulli/export-metadata` | `section_id`, `export_type` | Initiate metadata export |
| `/api/v1/tautulli/export-fields` | `media_type` | Available export fields |
| `/api/v1/tautulli/exports-table` | pagination | Export history |
| `/api/v1/tautulli/download-export` | `export_id` | Download export file |
| `/api/v1/tautulli/delete-export` | `export_id` | Delete export file |

---

## Export Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/export/playbacks/csv` | GET | Export playbacks to CSV (max 100k records) |
| `/api/v1/export/geojson` | GET | Export locations to GeoJSON |
| `/api/v1/export/geoparquet` | GET | Export to GeoParquet (20% smaller, 10x faster) |
| `/api/v1/stream/locations-geojson` | GET | Stream GeoJSON (chunked, handles 100k+ locations) |
| `/api/v1/tiles/{z}/{x}/{y}.pbf` | GET | Vector tiles for 1M+ locations |

---

## Real-Time Endpoints

### WebSocket

| Endpoint | Protocol | Description |
|----------|----------|-------------|
| `/api/v1/ws` | WebSocket | Real-time notifications |

**Message Types**:
- `sync_completed`: New data synced
- `stats_update`: Updated statistics
- `playback`: Individual playback events
- `ping/pong`: Connection health

### Plex Webhook

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/plex/webhook` | POST | Receive Plex push notifications |

---

## Import Endpoints

> **Note**: Import endpoints require NATS to be enabled (`go build -tags nats`) and `IMPORT_ENABLED=true`.

These endpoints manage direct import of Tautulli database files for migration or backup restore scenarios.

### Import Management

| Endpoint | Method | Auth | Description |
|----------|--------|------|-------------|
| `/api/v1/import/tautulli` | POST | Yes | Start Tautulli database import |
| `/api/v1/import/status` | GET | Yes | Get current import status |
| `/api/v1/import/stop` | DELETE | Yes | Stop a running import |
| `/api/v1/import/progress` | DELETE | Yes | Clear saved import progress |
| `/api/v1/import/validate` | POST | Yes | Validate a database file |

### Start Import

**POST** `/api/v1/import/tautulli`

Request body (optional):
```json
{
  "resume": false,
  "dry_run": false
}
```

| Field | Type | Description |
|-------|------|-------------|
| `resume` | boolean | Continue from last saved progress |
| `dry_run` | boolean | Validate without actually importing |

Response:
```json
{
  "success": true,
  "message": "import started",
  "stats": {
    "status": "running",
    "progress_percent": 0,
    "total_records": 50000,
    "processed": 0,
    "imported": 0,
    "skipped": 0,
    "errors": 0,
    "records_per_second": 0
  }
}
```

### Get Status

**GET** `/api/v1/import/status`

Response:
```json
{
  "success": true,
  "stats": {
    "status": "running",
    "progress_percent": 45.5,
    "total_records": 50000,
    "processed": 22750,
    "imported": 22500,
    "skipped": 200,
    "errors": 50,
    "records_per_second": 1250.5,
    "estimated_remaining_seconds": 22
  }
}
```

### Validate Database

**POST** `/api/v1/import/validate`

Request body:
```json
{
  "db_path": "/path/to/tautulli.db"
}
```

Response:
```json
{
  "success": true,
  "total_records": 50000,
  "date_range": {
    "earliest": "2020-01-15",
    "latest": "2025-12-01"
  },
  "unique_users": 25,
  "media_types": {
    "movie": 15000,
    "episode": 30000,
    "track": 5000
  }
}
```

### Configuration

| Environment Variable | Default | Description |
|---------------------|---------|-------------|
| `IMPORT_ENABLED` | `false` | Enable import functionality |
| `IMPORT_DB_PATH` | - | Path to Tautulli SQLite database |
| `IMPORT_BATCH_SIZE` | `1000` | Records per batch |
| `IMPORT_DRY_RUN` | `false` | Validate without importing |
| `IMPORT_AUTO_START` | `false` | Start import on application start |

---

## Data Sync Endpoints

These endpoints provide unified access to data synchronization status and operations.

### Sync Status

| Endpoint | Method | Auth | Description |
|----------|--------|------|-------------|
| `/api/v1/sync/status` | GET | Yes | Get combined status of all sync operations |
| `/api/v1/sync/plex/historical` | POST | Yes | Start Plex historical sync |

### Get Combined Sync Status

**GET** `/api/v1/sync/status`

Returns the current status of all sync operations including Tautulli import and Plex historical sync.

Response:
```json
{
  "tautulli_import": {
    "status": "idle",
    "total_records": 50000,
    "processed_records": 50000,
    "imported_records": 49800,
    "skipped_records": 150,
    "error_count": 50,
    "progress_percent": 100,
    "records_per_second": 0,
    "elapsed_seconds": 120,
    "estimated_remaining_seconds": 0
  },
  "plex_historical": null,
  "server_syncs": {}
}
```

### Start Plex Historical Sync

**POST** `/api/v1/sync/plex/historical`

Starts a historical sync from connected Plex servers. Cannot run while Tautulli import is in progress.

Request body (optional):
```json
{
  "days_back": 30,
  "library_ids": ["1", "2", "3"]
}
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `days_back` | integer | 30 | Number of days of history to sync |
| `library_ids` | string[] | all | Limit sync to specific library IDs |

Response (success):
```json
{
  "success": true,
  "message": "Plex historical sync started",
  "correlation_id": "20260110120000-abc12345"
}
```

Response (conflict - Tautulli import running):
```json
{
  "success": false,
  "error": "cannot start Plex historical sync while Tautulli import is running"
}
```

### WebSocket Progress Updates

Real-time progress updates are broadcast via WebSocket using the `sync_progress` message type:

```json
{
  "type": "sync_progress",
  "data": {
    "operation": "tautulli_import",
    "status": "running",
    "progress": {
      "total_records": 50000,
      "processed_records": 25000,
      "imported_records": 24800,
      "skipped_records": 150,
      "error_count": 50,
      "progress_percent": 50,
      "records_per_second": 1250,
      "elapsed_seconds": 20,
      "estimated_remaining_seconds": 20
    },
    "correlation_id": "20260110120000-abc12345"
  }
}
```

| Field | Values | Description |
|-------|--------|-------------|
| `operation` | `tautulli_import`, `plex_historical`, `server_sync` | Type of sync operation |
| `status` | `running`, `completed`, `error`, `cancelled` | Current operation status |

---

## Server Management Endpoints

ADR-0026 Phase 4: Multi-server management UI APIs for CRUD operations on media servers.

| Endpoint | Method | Auth | Description |
|----------|--------|------|-------------|
| `/api/v1/admin/servers` | GET | Yes | List all configured servers (env + DB) |
| `/api/v1/admin/servers` | POST | Yes | Create new server |
| `/api/v1/admin/servers/test` | POST | Yes | Test server connectivity |
| `/api/v1/admin/servers/db` | GET | Yes | List only database-stored servers |
| `/api/v1/admin/servers/{id}` | GET | Yes | Get server by ID |
| `/api/v1/admin/servers/{id}` | PUT | Yes | Update server |
| `/api/v1/admin/servers/{id}` | DELETE | Yes | Delete server |

### List All Servers

**GET** `/api/v1/admin/servers`

Returns all configured media servers from both environment variables and database.
Requires admin role.

**Response**:
```json
{
  "status": "success",
  "data": {
    "servers": [
      {
        "id": "env-plex",
        "platform": "plex",
        "name": "Main Plex Server",
        "url": "http://localhost:32400",
        "enabled": true,
        "source": "env",
        "status": "connected",
        "realtime_enabled": true,
        "webhooks_enabled": false,
        "session_polling_enabled": false,
        "last_sync_at": "2026-01-07T10:00:00Z",
        "last_sync_status": "success",
        "immutable": true
      },
      {
        "id": "db-jellyfin-1",
        "platform": "jellyfin",
        "name": "Jellyfin Server",
        "url": "http://localhost:8096",
        "enabled": true,
        "source": "ui",
        "status": "syncing",
        "realtime_enabled": false,
        "webhooks_enabled": true,
        "session_polling_enabled": true,
        "immutable": false
      }
    ],
    "total_count": 2,
    "connected_count": 1,
    "syncing_count": 1,
    "error_count": 0,
    "last_checked": "2026-01-07T10:30:00Z"
  }
}
```

### Create Server

**POST** `/api/v1/admin/servers`

Creates a new media server configuration stored in the database.
Requires admin role.

**Request Body**:
```json
{
  "platform": "jellyfin",
  "name": "Jellyfin Server",
  "url": "http://localhost:8096",
  "token": "your-api-key",
  "realtime_enabled": false,
  "webhooks_enabled": true,
  "session_polling_enabled": true,
  "session_polling_interval": "30s"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `platform` | string | Yes | `plex`, `jellyfin`, `emby`, or `tautulli` |
| `name` | string | Yes | Display name (1-100 chars) |
| `url` | string | Yes | Server URL with protocol and port |
| `token` | string | Yes | API token (min 8 chars) |
| `realtime_enabled` | boolean | No | Enable real-time updates |
| `webhooks_enabled` | boolean | No | Enable webhooks |
| `session_polling_enabled` | boolean | No | Enable session polling |
| `session_polling_interval` | string | No | Polling interval (e.g., `30s`, `1m`) |

**Response** (201 Created):
```json
{
  "status": "success",
  "data": {
    "id": "db-jellyfin-1",
    "platform": "jellyfin",
    "name": "Jellyfin Server",
    "url": "http://localhost:8096",
    "token_masked": "****...key",
    "enabled": true,
    "source": "ui",
    "status": "configured",
    "immutable": false,
    "created_at": "2026-01-07T10:00:00Z",
    "updated_at": "2026-01-07T10:00:00Z"
  }
}
```

### Test Server Connection

**POST** `/api/v1/admin/servers/test`

Tests connectivity to a server without saving it.
Requires admin role.

**Request Body**:
```json
{
  "platform": "plex",
  "url": "http://localhost:32400",
  "token": "your-plex-token"
}
```

**Response**:
```json
{
  "status": "success",
  "data": {
    "success": true,
    "latency_ms": 45,
    "server_name": "My Plex Server",
    "version": "1.32.0.1234"
  }
}
```

### Update Server

**PUT** `/api/v1/admin/servers/{id}`

Updates an existing server configuration.
Only UI-created servers can be updated (env-var servers are immutable).
Requires admin role.

**Request Body** (partial update supported):
```json
{
  "name": "Updated Server Name",
  "enabled": false
}
```

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | New display name |
| `url` | string | New server URL |
| `token` | string | New API token (omit to keep current) |
| `enabled` | boolean | Enable/disable server |
| `realtime_enabled` | boolean | Enable real-time updates |
| `webhooks_enabled` | boolean | Enable webhooks |
| `session_polling_enabled` | boolean | Enable session polling |
| `session_polling_interval` | string | Polling interval |

### Delete Server

**DELETE** `/api/v1/admin/servers/{id}`

Deletes a server configuration.
Only UI-created servers can be deleted (env-var servers cannot be deleted).
Requires admin role.

**Response** (204 No Content on success)

### Error Responses

| Status | Code | Description |
|--------|------|-------------|
| 400 | `VALIDATION_ERROR` | Invalid request body |
| 403 | `IMMUTABLE_SERVER` | Cannot modify env-var server |
| 404 | `NOT_FOUND` | Server not found |
| 409 | `DUPLICATE_SERVER` | Server with same URL already exists |

---

## Query Parameters

### Filter Parameters

All analytics endpoints support these parameters:

| Parameter | Type | Description |
|-----------|------|-------------|
| `start_date` | ISO 8601 | Start of date range |
| `end_date` | ISO 8601 | End of date range |
| `days` | integer | Alternative: last N days |
| `user` | string | Filter by username (comma-separated) |
| `media_type` | string | Filter: `movie`, `episode`, `track` |
| `platforms` | string | Filter by platform |
| `players` | string | Filter by player app |
| `transcode_decisions` | string | Filter: `DirectPlay`, `Transcode` |
| `video_resolutions` | string | Filter: `1080p`, `4k` |
| `video_codecs` | string | Filter: `H.264`, `HEVC` |
| `audio_codecs` | string | Filter: `AAC`, `EAC3` |
| `libraries` | string | Filter by library name |
| `content_ratings` | string | Filter: `PG`, `R` |
| `years` | string | Filter by release year |
| `location_type` | string | Filter: `LAN`, `WAN` |
| `limit` | integer | Max results (default varies) |

### Pagination Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `cursor` | string | Cursor from previous response |
| `limit` | integer | Items per page (default: 100, max: 1000) |
| `offset` | integer | Legacy offset pagination |

### Tautulli Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `time_range` | integer | Days to analyze (default: 30) |
| `y_axis` | string | `plays` or `duration` |
| `user_id` | integer | Tautulli user ID |
| `rating_key` | string | Plex rating key |
| `section_id` | integer | Library section ID |
| `query_days` | string | Comma-separated: `1,7,30,0` |
| `order_column` | string | Sort column |
| `order_dir` | string | `asc` or `desc` |
| `length` | integer | Page size (default: 25) |
| `search` | string | Search filter |

---

## Response Format

### Success Response

```json
{
  "status": "success",
  "data": { /* payload */ },
  "metadata": {
    "timestamp": "2025-11-18T12:34:56Z",
    "query_time_ms": 23
  }
}
```

### Error Response

```json
{
  "status": "error",
  "data": null,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid date range",
    "details": { "field": "start_date" }
  }
}
```

### Pagination Response

```json
{
  "status": "success",
  "data": {
    "events": [...],
    "pagination": {
      "limit": 10,
      "has_more": true,
      "next_cursor": "eyJzdGFydGVkX2F0IjoiMjAyNS0xMS0..."
    }
  }
}
```

---

## Example Requests

**Get statistics**:
```bash
curl http://localhost:3857/api/v1/stats
```

**Get trends for last 30 days**:
```bash
curl "http://localhost:3857/api/v1/analytics/trends?days=30"
```

**Filtered analytics**:
```bash
curl "http://localhost:3857/api/v1/analytics/geographic?platforms=Roku&video_resolutions=4k"
```

**Cursor pagination**:
```bash
# First page
curl "http://localhost:3857/api/v1/playbacks?limit=10"

# Next page
curl "http://localhost:3857/api/v1/playbacks?limit=10&cursor=eyJzdGFydGVkX2F0..."
```
