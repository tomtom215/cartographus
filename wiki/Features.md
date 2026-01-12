# Features Overview

Comprehensive guide to Cartographus features and capabilities.

**[Home](Home)** | **[Configuration](Configuration)** | **Features** | **[FAQ](FAQ)**

---

## Feature Categories

- [Analytics Engine](#analytics-engine)
- [Geographic Visualization](#geographic-visualization)
- [Real-Time Monitoring](#real-time-monitoring)
- [Security Detection](#security-detection)
- [Data Management](#data-management)
- [API](#api)

---

## Analytics Engine

30+ dedicated analytics endpoints covering user behavior, content performance, and system health.

### User Behavior Analytics

| Metric | Description |
|--------|-------------|
| **Binge Detection** | Identifies users watching 3+ episodes consecutively |
| **Watch Parties** | Detects multiple users watching the same content simultaneously |
| **Cohort Retention** | Week-over-week user engagement trends |
| **Engagement Scoring** | Per-user engagement score based on activity patterns |
| **Pause Patterns** | Tracks where users pause, rewind, or abandon content |
| **Abandonment Rates** | Content that users frequently don't finish |

### Content Analytics

| Metric | Description |
|--------|-------------|
| **Popular Titles** | Most-watched content by plays, duration, or unique users |
| **Library Distribution** | Content breakdown by type, genre, and library |
| **Discovery Latency** | Time between content addition and first watch |
| **Time-to-First-Watch** | How quickly new content gets viewed |

### Performance Analytics

| Metric | Description |
|--------|-------------|
| **Transcode Efficiency** | Direct play vs. transcode ratio |
| **Hardware Acceleration** | GPU usage for transcoding |
| **Resolution Mismatch** | Users watching at lower quality than source |
| **Bitrate Distribution** | Streaming quality distribution |
| **Buffer Health** | Playback stability metrics |

### Quality of Experience (QoE)

| Metric | Description |
|--------|-------------|
| **QoE Scoring** | Composite quality score per session |
| **Connection Security** | HTTP vs HTTPS breakdown |
| **Codec Compatibility** | Client codec support analysis |
| **HDR Analytics** | HDR content playback patterns |
| **Audio Analytics** | Audio codec and channel usage |
| **Subtitle Analytics** | Subtitle language preferences |

### Comparative Analytics

| Metric | Description |
|--------|-------------|
| **Period Comparison** | This week vs. last week, this month vs. last |
| **User Networks** | Social connections based on shared viewing |
| **Device Migration** | Users switching between devices |

---

## Geographic Visualization

See where your users are watching from with interactive maps.

### 2D WebGL Map

High-performance vector map with:

- **Smart Clustering** - Handles 10,000+ points smoothly
- **Color-Coded Markers** - By user, content type, or activity
- **Detailed Popups** - User, content, and session information
- **Viewport Filtering** - Load only visible data

### 3D Globe View

Immersive geographic visualization with:

- **Scatterplot Points** - User locations on the globe
- **H3 Hexagons** - Aggregated activity by geographic area
- **Animated Arcs** - User-to-server connection visualization
- **Rotation & Zoom** - Full interactive control

### Temporal Heatmap

Watch activity spread across geography over time:

- **Time Slider** - Scrub through historical data
- **Animation Mode** - Play back activity patterns
- **Aggregation** - By hour, day, or week

### Spatial Queries

- **Viewport Filtering** - Only load data for visible area
- **Nearby Search** - Find activity within radius
- **Density Analysis** - Identify activity hotspots

---

## Real-Time Monitoring

Sub-second updates on current activity via WebSocket connections.

### Live Activity Dashboard

- **Active Sessions** - Currently playing content
- **Transcoding Status** - CPU/GPU usage, speed
- **Progress Tracking** - Playback position and duration
- **Quality Indicators** - Resolution, bitrate, codec

### Buffer Health Tracking

Predictive buffering detection:

- **Warning Threshold** - Alerts at 50% buffer health
- **Critical Threshold** - Alerts at 20% buffer health
- **10-15 Second Warning** - Predicts buffering before it occurs

### Hardware Transcode Monitoring

- **GPU Acceleration Status** - NVENC, QSV, VAAPI
- **Transcode Queue** - Pending transcode jobs
- **Quality Transitions** - Resolution/bitrate changes mid-stream

### WebSocket Updates

- **All Connected Servers** - Unified view across Plex, Jellyfin, Emby
- **< 1 Second Latency** - Near-instant updates
- **Automatic Reconnection** - Handles network interruptions

---

## Security Detection

Detect account sharing and suspicious activity with 7 detection rules.

### Detection Rules

| Rule | Description | Example |
|------|-------------|---------|
| **Impossible Travel** | User streams from distant locations too quickly | NYC, then London 5 minutes later |
| **Concurrent Streams** | Same user watching multiple streams | 4 simultaneous streams |
| **Device Velocity** | Device appears from multiple IPs rapidly | Same device, 3 countries in 1 hour |
| **Geo Restriction** | Streaming from blocked countries | Blocked country access attempt |
| **Simultaneous Locations** | Active streams from distant locations | LA and NYC at the same time |
| **User Agent Anomaly** | Unusual client software patterns | Spoofed or unknown clients |
| **VPN Usage** | Streaming through VPN services | Known VPN IP addresses |

### Trust Scoring

- **Per-User Trust Score** - Starts at 100, decreases with violations
- **Automatic Recovery** - Score recovers over time
- **Threshold Actions** - Configurable restrictions at score thresholds

### Alert Configuration

Send alerts via:

- **Discord Webhooks** - Rich embeds with violation details
- **Generic Webhooks** - POST to any HTTP endpoint
- **API Query** - Retrieve violations programmatically

### Configuration

```yaml
environment:
  - DETECTION_ENABLED=true
  - DETECTION_TRUST_SCORE_DECREMENT=10
  - DETECTION_TRUST_SCORE_THRESHOLD=50
  - DISCORD_WEBHOOK_ENABLED=true
  - DISCORD_WEBHOOK_URL=https://discord.com/api/webhooks/...
```

---

## Data Management

### Filtering

16 filter dimensions for all analytics:

- Date range
- Users
- Platforms (Plex, Jellyfin, Emby)
- Devices
- Libraries
- Content types
- Resolutions
- Codecs
- Transcode decisions
- Countries
- And more...

### Search

**RapidFuzz-Powered Fuzzy Search**

- Typo tolerance
- Partial matching
- Relevance scoring

### Approximate Analytics

For large datasets, use O(1) approximate analytics:

- **HyperLogLog** - Distinct count estimates
- **KLL Percentiles** - Approximate p50, p95, p99

### Pagination

- **Cursor-Based** - Consistent O(1) performance
- **Millions of Records** - No performance degradation

### Export

- **CSV** - Spreadsheet-compatible
- **GeoJSON** - Geographic data for GIS tools
- **GeoParquet** - Efficient geographic data format

### Import

- **Tautulli Database** - Import SQLite with progress tracking
- **Plex Historical Sync** - Backfill from Plex API

### Backup & Restore

- **Scheduled Backups** - Automatic daily/weekly
- **Retention Policies** - Keep daily, weekly, monthly backups
- **Compression** - gzip with configurable level
- **Encryption** - AES-256 encryption option

---

## API

302 REST API endpoints organized by domain.

### Endpoint Categories

| Category | Endpoints | Description |
|----------|-----------|-------------|
| **Health** | 3 | System health and status |
| **Auth** | 12 | Authentication and sessions |
| **Users** | 18 | User management and analytics |
| **Analytics** | 45 | Analytics queries |
| **Locations** | 15 | Geographic data |
| **Spatial** | 12 | Map and globe data |
| **Sessions** | 20 | Playback session data |
| **Detection** | 10 | Security detection |
| **Sync** | 15 | Data synchronization |
| **Admin** | 25 | Administrative functions |
| **... and more** | 127 | Additional endpoints |

### Key Endpoints

```bash
# Health check
GET /api/v1/health

# Summary statistics
GET /api/v1/stats

# Playback locations with geo data
GET /api/v1/locations

# Analytics (30+ endpoints)
GET /api/v1/analytics/users/engagement
GET /api/v1/analytics/content/popular
GET /api/v1/analytics/performance/transcode

# H3 hexagon aggregation for globe
GET /api/v1/spatial/hexagons

# Real-time WebSocket
WS /api/v1/ws
```

### API Documentation

Full API documentation: [docs/API-REFERENCE.md](https://github.com/tomtom215/cartographus/blob/main/docs/API-REFERENCE.md)

---

## Dashboard Pages

6 themed dashboard pages with 47+ interactive charts:

1. **Overview** - At-a-glance summary
2. **Users** - User behavior and engagement
3. **Content** - Library and content analytics
4. **Performance** - System and transcode metrics
5. **Geographic** - Map and location data
6. **Security** - Detection alerts and trust scores

---

## Next Steps

- **[Analytics](Analytics)** - Deep dive into analytics features
- **[Maps & Globe](Maps-and-Globe)** - Geographic visualization guide
- **[Security Detection](Security-Detection)** - Account sharing detection
- **[API Reference](API-Reference)** - Complete API documentation
