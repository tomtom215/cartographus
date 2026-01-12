# API Reference

Cartographus provides 302 REST API endpoints for comprehensive programmatic access.

**[Home](Home)** | **[Features](Features)** | **API Reference** | **[FAQ](FAQ)**

---

## Overview

The Cartographus API is RESTful and returns JSON responses. All endpoints are prefixed with `/api/v1/`.

### Base URL

```
http://localhost:3857/api/v1
```

### Authentication

Most endpoints require authentication. Include credentials using:

- **JWT Token** (Cookie): Automatically included after login
- **Authorization Header**: `Authorization: Bearer <token>`
- **Basic Auth**: `Authorization: Basic <base64(username:password)>`

### Response Format

```json
{
  "data": { ... },
  "meta": {
    "total": 100,
    "page": 1,
    "per_page": 20
  }
}
```

Error responses:

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid date range"
  }
}
```

---

## Quick Reference

| Category | Endpoints | Description |
|----------|-----------|-------------|
| Health | 3 | System health and status |
| Auth | 12 | Authentication and sessions |
| Users | 18 | User management and stats |
| Analytics | 45 | Analytics queries |
| Locations | 15 | Geographic data |
| Spatial | 12 | Map and globe data |
| Sessions | 20 | Playback sessions |
| Detection | 10 | Security detection |
| Sync | 15 | Data synchronization |
| Servers | 8 | Media server management |
| Admin | 25 | Administrative functions |
| Backup | 8 | Backup and restore |
| Export | 6 | Data export |

---

## Common Endpoints

### Health Check

```http
GET /api/v1/health
```

**Response:**

```json
{
  "status": "healthy",
  "version": "1.0.0",
  "database": "connected",
  "uptime": "24h32m15s"
}
```

### Summary Statistics

```http
GET /api/v1/stats
```

**Response:**

```json
{
  "total_plays": 15432,
  "total_users": 23,
  "total_watch_time": 1234567,
  "active_sessions": 3,
  "libraries": 5
}
```

### Current Sessions

```http
GET /api/v1/sessions/active
```

**Response:**

```json
{
  "sessions": [
    {
      "id": "abc123",
      "user": "john",
      "title": "Movie Title",
      "progress": 0.45,
      "state": "playing",
      "transcoding": false
    }
  ]
}
```

---

## Locations API

### Get All Locations

```http
GET /api/v1/locations
```

**Query Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `start_date` | string | ISO 8601 date |
| `end_date` | string | ISO 8601 date |
| `user` | string | Filter by username |
| `limit` | integer | Max results (default 100) |
| `cursor` | string | Pagination cursor |

**Response:**

```json
{
  "locations": [
    {
      "id": "loc123",
      "latitude": 40.7128,
      "longitude": -74.0060,
      "city": "New York",
      "country": "United States",
      "user": "john",
      "timestamp": "2024-01-15T10:30:00Z"
    }
  ],
  "next_cursor": "abc123"
}
```

### Get Location Clusters

```http
GET /api/v1/locations/clusters
```

For map visualization with point clustering.

---

## Analytics API

### User Engagement

```http
GET /api/v1/analytics/users/engagement
```

**Query Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `period` | string | `day`, `week`, `month` |
| `start_date` | string | ISO 8601 date |
| `end_date` | string | ISO 8601 date |

### Popular Content

```http
GET /api/v1/analytics/content/popular
```

**Query Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `metric` | string | `plays`, `duration`, `users` |
| `limit` | integer | Max results |
| `library` | string | Filter by library |

### Transcode Efficiency

```http
GET /api/v1/analytics/performance/transcode
```

### Watch Patterns

```http
GET /api/v1/analytics/behavior/patterns
```

---

## Spatial API

### H3 Hexagons

```http
GET /api/v1/spatial/hexagons
```

For globe view hexagonal aggregation.

**Query Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `resolution` | integer | H3 resolution (0-15) |
| `start_date` | string | ISO 8601 date |
| `end_date` | string | ISO 8601 date |

**Response:**

```json
{
  "hexagons": [
    {
      "h3_index": "8928308280fffff",
      "count": 150,
      "center": [40.7128, -74.0060]
    }
  ]
}
```

### Arcs (User to Server)

```http
GET /api/v1/spatial/arcs
```

For globe view connection visualization.

---

## Detection API

### Get Violations

```http
GET /api/v1/detection/violations
```

**Query Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `type` | string | Violation type filter |
| `user` | string | Filter by username |
| `status` | string | `active`, `resolved` |

### User Trust Score

```http
GET /api/v1/detection/users/{username}/trust
```

**Response:**

```json
{
  "username": "john",
  "trust_score": 85,
  "violations": 2,
  "last_violation": "2024-01-10T15:30:00Z"
}
```

---

## Sync API

### Trigger Sync

```http
POST /api/v1/sync/trigger
```

### Sync Status

```http
GET /api/v1/sync/status
```

**Response:**

```json
{
  "servers": [
    {
      "id": "plex-main",
      "type": "plex",
      "status": "syncing",
      "last_sync": "2024-01-15T10:00:00Z",
      "progress": 0.75
    }
  ]
}
```

---

## WebSocket API

### Real-Time Updates

```
WS /api/v1/ws
```

**Event Types:**

| Event | Description |
|-------|-------------|
| `session.started` | Playback started |
| `session.stopped` | Playback stopped |
| `session.progress` | Progress update |
| `detection.alert` | Security detection |
| `sync.progress` | Sync progress update |

**Example Message:**

```json
{
  "type": "session.started",
  "data": {
    "session_id": "abc123",
    "user": "john",
    "title": "Movie Title"
  },
  "timestamp": "2024-01-15T10:30:00Z"
}
```

---

## Pagination

### Cursor-Based Pagination

All list endpoints support cursor-based pagination for consistent O(1) performance.

**Request:**

```http
GET /api/v1/locations?limit=50&cursor=abc123
```

**Response:**

```json
{
  "data": [...],
  "meta": {
    "next_cursor": "xyz789",
    "has_more": true
  }
}
```

---

## Rate Limiting

Default limits:

| Endpoint Type | Limit |
|---------------|-------|
| Authentication | 5/min |
| Analytics | 1000/min |
| Default | 100/min |

Rate limit headers:

```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1705312800
```

---

## Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `UNAUTHORIZED` | 401 | Missing or invalid authentication |
| `FORBIDDEN` | 403 | Insufficient permissions |
| `NOT_FOUND` | 404 | Resource not found |
| `VALIDATION_ERROR` | 400 | Invalid request parameters |
| `RATE_LIMITED` | 429 | Too many requests |
| `INTERNAL_ERROR` | 500 | Server error |

---

## Full Documentation

For complete API documentation with all 302 endpoints, see:

**[docs/API-REFERENCE.md](https://github.com/tomtom215/cartographus/blob/main/docs/API-REFERENCE.md)**

---

## Examples

### cURL

```bash
# Get health
curl http://localhost:3857/api/v1/health

# Get stats (authenticated)
curl -u admin:password http://localhost:3857/api/v1/stats

# Get locations for last 7 days
curl -u admin:password \
  "http://localhost:3857/api/v1/locations?start_date=2024-01-08&end_date=2024-01-15"
```

### Python

```python
import requests

base_url = "http://localhost:3857/api/v1"
auth = ("admin", "password")

# Get active sessions
response = requests.get(f"{base_url}/sessions/active", auth=auth)
sessions = response.json()

# Get analytics
response = requests.get(
    f"{base_url}/analytics/content/popular",
    params={"limit": 10, "metric": "plays"},
    auth=auth
)
popular = response.json()
```

### JavaScript

```javascript
const baseUrl = 'http://localhost:3857/api/v1';
const headers = {
  'Authorization': 'Basic ' + btoa('admin:password')
};

// Get stats
const response = await fetch(`${baseUrl}/stats`, { headers });
const stats = await response.json();

// WebSocket for real-time
const ws = new WebSocket('ws://localhost:3857/api/v1/ws');
ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log('Event:', data.type, data.data);
};
```

---

## Next Steps

- **[Features](Features)** - Feature overview
- **[Configuration](Configuration)** - API configuration options
- **[Troubleshooting](Troubleshooting)** - API troubleshooting
