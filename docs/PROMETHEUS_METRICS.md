# Prometheus Metrics Integration

**Implementation Date:** 2025-11-23
**Optimization:** LOW-4 - Prometheus Metrics Integration (6-8 hours)
**Status:** ✅ COMPLETE

---

## Overview

Comprehensive Prometheus metrics instrumentation for production observability, monitoring, and alerting. This implementation provides real-time insights into:

- API endpoint performance and throughput
- Database query latency and errors
- Sync operation metrics
- Cache efficiency (vector tiles, geolocations)
- WebSocket connection health

---

## Metrics Exposed

### API Metrics

| Metric Name | Type | Labels | Description |
|-------------|------|--------|-------------|
| `api_requests_total` | Counter | method, endpoint, status_code | Total number of API requests |
| `api_request_duration_seconds` | Histogram | method, endpoint | API request duration in seconds |
| `api_active_requests` | Gauge | - | Current number of active API requests |
| `api_rate_limit_hits_total` | Counter | endpoint | Total number of rate limit rejections |

**Example Queries:**
```promql
# Request rate by endpoint
rate(api_requests_total[5m])

# p95 latency for all endpoints
histogram_quantile(0.95, rate(api_request_duration_seconds_bucket[5m]))

# Error rate (5xx responses)
rate(api_requests_total{status_code=~"5.."}[5m])
```

---

### Database Metrics (DuckDB)

| Metric Name | Type | Labels | Description |
|-------------|------|--------|-------------|
| `duckdb_query_duration_seconds` | Histogram | operation, table | Duration of DuckDB queries in seconds |
| `duckdb_query_errors_total` | Counter | operation, table, error_type | Total number of DuckDB query errors |
| `duckdb_connection_pool_size` | Gauge | - | Current number of database connections in use |
| `duckdb_spatial_operations_total` | Counter | operation_type | Total number of spatial operations (ST_*) |

**Example Queries:**
```promql
# p95 query latency by operation
histogram_quantile(0.95, rate(duckdb_query_duration_seconds_bucket[5m]))

# Query error rate
rate(duckdb_query_errors_total[5m])

# Spatial operations rate
rate(duckdb_spatial_operations_total[5m])
```

---

### Vector Tile Cache Metrics (MEDIUM-1)

| Metric Name | Type | Labels | Description |
|-------------|------|--------|-------------|
| `tile_cache_hits_total` | Counter | - | Total number of vector tile cache hits |
| `tile_cache_misses_total` | Counter | - | Total number of vector tile cache misses |
| `tile_cache_entries` | Gauge | - | Current number of cached vector tiles |
| `tile_cache_data_version` | Gauge | - | Current tile cache data version (increments on data changes) |

**Example Queries:**
```promql
# Cache hit ratio
rate(tile_cache_hits_total[5m]) / (rate(tile_cache_hits_total[5m]) + rate(tile_cache_misses_total[5m]))

# Cache size
tile_cache_entries

# Data version (for cache invalidation tracking)
tile_cache_data_version
```

**Performance Impact:** With cache hit ratio >80%, expect 5-8x improvement in tile serving latency.

---

### Sync Operation Metrics

| Metric Name | Type | Labels | Description |
|-------------|------|--------|-------------|
| `sync_duration_seconds` | Histogram | - | Duration of sync operations in seconds |
| `sync_records_processed_total` | Counter | - | Total number of playback records processed during sync |
| `sync_errors_total` | Counter | error_type | Total number of sync errors |
| `sync_last_success_timestamp` | Gauge | - | Unix timestamp of last successful sync |
| `sync_batch_size` | Histogram | - | Number of records in sync batches |

**Example Queries:**
```promql
# Sync duration
sync_duration_seconds

# Records processed rate
rate(sync_records_processed_total[5m])

# Sync error rate
rate(sync_errors_total[5m])

# Time since last successful sync
time() - sync_last_success_timestamp
```

---

### Geolocation Metrics (MEDIUM-2)

| Metric Name | Type | Labels | Description |
|-------------|------|--------|-------------|
| `geolocation_batch_size` | Histogram | - | Number of IPs in geolocation batch lookups |
| `geolocation_cache_hits_total` | Counter | - | Total number of geolocation cache hits (DB) |
| `geolocation_cache_misses_total` | Counter | - | Total number of geolocation cache misses (API fetch required) |
| `geolocation_api_call_duration_seconds` | Histogram | - | Duration of Tautulli geolocation API calls |

**Example Queries:**
```promql
# Geolocation cache hit ratio
rate(geolocation_cache_hits_total[5m]) / (rate(geolocation_cache_hits_total[5m]) + rate(geolocation_cache_misses_total[5m]))

# Average batch size
avg(geolocation_batch_size)

# API call latency
histogram_quantile(0.95, rate(geolocation_api_call_duration_seconds_bucket[5m]))
```

**Performance Impact:** With cache hit ratio >90%, expect 10-20x improvement in sync performance.

---

### WebSocket Metrics

| Metric Name | Type | Labels | Description |
|-------------|------|--------|-------------|
| `websocket_connections` | Gauge | - | Current number of active WebSocket connections |
| `websocket_messages_sent_total` | Counter | - | Total number of WebSocket messages sent |
| `websocket_messages_received_total` | Counter | - | Total number of WebSocket messages received |
| `websocket_errors_total` | Counter | error_type | Total number of WebSocket errors |

**Example Queries:**
```promql
# Active connections
websocket_connections

# Message throughput
rate(websocket_messages_sent_total[5m])

# Error rate
rate(websocket_errors_total[5m])
```

---

### OIDC Authentication Metrics (Zitadel)

| Metric Name | Type | Labels | Description |
|-------------|------|--------|-------------|
| `oidc_login_attempts_total` | Counter | provider, outcome | Total OIDC login attempts |
| `oidc_login_duration_seconds` | Histogram | provider | Duration of OIDC login flows |
| `oidc_logout_total` | Counter | logout_type, outcome | Total logout events (RP-initiated, back-channel) |
| `oidc_token_refresh_total` | Counter | provider, outcome | Total token refresh attempts |
| `oidc_token_refresh_duration_seconds` | Histogram | provider | Duration of token refresh operations |
| `oidc_state_store_operations_total` | Counter | operation, outcome | State store operations (store, get, delete, cleanup) |
| `oidc_state_store_size` | Gauge | - | Current number of pending OIDC states |
| `oidc_back_channel_logout_total` | Counter | outcome | Back-channel logout events |
| `oidc_validation_errors_total` | Counter | error_type | Token validation errors by type |
| `oidc_sessions_created_total` | Counter | provider | Session creation events |
| `oidc_sessions_terminated_total` | Counter | reason | Session termination by reason |
| `oidc_active_sessions` | Gauge | - | Currently active OIDC sessions |
| `oidc_token_exchange_duration_seconds` | Histogram | provider | Duration of token exchange operations |
| `oidc_jwks_fetch_duration_seconds` | Histogram | provider | Duration of JWKS fetch operations |

**Example Queries:**
```promql
# Login success rate
rate(oidc_login_attempts_total{outcome="success"}[5m]) / rate(oidc_login_attempts_total[5m])

# Login latency p95
histogram_quantile(0.95, rate(oidc_login_duration_seconds_bucket[5m]))

# Active sessions
oidc_active_sessions

# Back-channel logout success rate
rate(oidc_back_channel_logout_total{outcome="success"}[5m])

# Token validation error rate by type
rate(oidc_validation_errors_total[5m])

# State store size (pending auth flows)
oidc_state_store_size
```

**Implementation:** See [ADR-0015](docs/adr/0015-zero-trust-authentication-authorization.md) for OIDC architecture details.

---

## Metrics Endpoint

**URL:** `http://localhost:3857/metrics`

**Format:** Prometheus exposition format

**Access:** Public (no authentication required for Prometheus scraping)

**Example Response:**
```
# HELP api_requests_total Total number of API requests
# TYPE api_requests_total counter
api_requests_total{endpoint="/api/v1/locations",method="GET",status_code="200"} 1234
api_requests_total{endpoint="/api/v1/playbacks",method="GET",status_code="200"} 5678

# HELP api_request_duration_seconds API request duration in seconds
# TYPE api_request_duration_seconds histogram
api_request_duration_seconds_bucket{endpoint="/api/v1/locations",method="GET",le="0.01"} 1000
api_request_duration_seconds_bucket{endpoint="/api/v1/locations",method="GET",le="0.025"} 1200
...
```

---

## Prometheus Configuration

Add the following to your `prometheus.yml`:

```yaml
scrape_configs:
  - job_name: 'cartographus'
    scrape_interval: 15s
    scrape_timeout: 10s
    static_configs:
      - targets: ['localhost:3857']
        labels:
          environment: 'production'
          service: 'cartographus'
```

---

## Grafana Dashboard

A comprehensive Grafana dashboard is provided in `grafana-dashboard.json` with 12 panels:

1. **API Request Rate** - Requests per second by endpoint
2. **API Request Duration (p50, p95)** - Latency percentiles
3. **Active API Requests** - Concurrent request gauge
4. **Active WebSocket Connections** - WebSocket health
5. **Cached Vector Tiles** - Cache size
6. **Tile Cache Data Version** - Invalidation tracking
7. **Vector Tile Cache Hit Rate** - Cache efficiency
8. **Sync Operation Duration** - Sync performance
9. **Sync Records Processed Rate** - Throughput
10. **Geolocation Cache Hit Rate** - Batch lookup efficiency
11. **DuckDB Query Duration (p50, p95)** - Database performance
12. **DuckDB Query Errors** - Database error rate

### Importing the Dashboard

1. Open Grafana UI
2. Navigate to **Dashboards** → **Import**
3. Upload `grafana-dashboard.json`
4. Select your Prometheus datasource
5. Click **Import**

---

## Alerting Rules

Example Prometheus alerting rules (`alerts.yml`):

```yaml
groups:
  - name: cartographus
    interval: 30s
    rules:
      # API alerts
      - alert: HighAPILatency
        expr: histogram_quantile(0.95, rate(api_request_duration_seconds_bucket[5m])) > 1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "API p95 latency is high (> 1s)"
          description: "Endpoint {{ $labels.endpoint }} has p95 latency of {{ $value }}s"

      - alert: HighAPIErrorRate
        expr: rate(api_requests_total{status_code=~"5.."}[5m]) > 10
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "High API error rate (> 10/s)"
          description: "Endpoint {{ $labels.endpoint }} has error rate of {{ $value }}/s"

      # Sync alerts
      - alert: SyncFailure
        expr: (time() - sync_last_success_timestamp) > 3600
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "Sync has not succeeded in over 1 hour"
          description: "Last successful sync was {{ $value }}s ago"

      - alert: HighSyncErrors
        expr: rate(sync_errors_total[5m]) > 1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High sync error rate (> 1/s)"
          description: "Sync error rate is {{ $value }}/s"

      # Cache alerts
      - alert: LowCacheHitRatio
        expr: |
          rate(tile_cache_hits_total[5m]) /
          (rate(tile_cache_hits_total[5m]) + rate(tile_cache_misses_total[5m])) < 0.5
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "Tile cache hit ratio is low (< 50%)"
          description: "Cache hit ratio is {{ $value }}"

      # Database alerts
      - alert: HighDatabaseLatency
        expr: histogram_quantile(0.95, rate(duckdb_query_duration_seconds_bucket[5m])) > 1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Database p95 latency is high (> 1s)"
          description: "Operation {{ $labels.operation }} has p95 latency of {{ $value }}s"
```

---

## Performance Targets

| Metric | Target | Threshold |
|--------|--------|-----------|
| API p95 latency | < 100ms | ⚠️ 500ms, 🔴 1s |
| Database p95 latency | < 30ms | ⚠️ 100ms, 🔴 500ms |
| Tile cache hit ratio | > 80% | ⚠️ 50%, 🔴 30% |
| Geolocation cache hit ratio | > 90% | ⚠️ 70%, 🔴 50% |
| Sync duration | < 30s | ⚠️ 60s, 🔴 120s |
| Active WebSocket connections | < 1000 | ⚠️ 5000, 🔴 10000 |

---

## Implementation Details

### Code Locations

- **Metrics Package:** `internal/metrics/metrics.go` (370 lines)
- **Metrics Middleware:** `internal/middleware/prometheus.go` (50 lines)
- **Database Instrumentation:** `internal/database/database.go` (tile cache methods)
- **Sync Instrumentation:** `internal/sync/manager.go` (processBatch, syncDataSince)
- **API Instrumentation:** `internal/api/chi_router.go` (middleware chain)
- **Metrics Endpoint:** `internal/api/chi_router.go` (`/metrics` endpoint)

### Key Functions

1. **`metrics.RecordDBQuery(operation, table, duration, err)`** - Database query instrumentation
2. **`metrics.RecordAPIRequest(method, endpoint, statusCode, duration)`** - API request instrumentation
3. **`metrics.RecordSyncOperation(duration, recordsProcessed, err)`** - Sync operation instrumentation
4. **`metrics.TrackActiveRequest(inc bool)`** - Active request tracking
5. **`middleware.PrometheusMetrics(next)`** - Middleware for automatic API instrumentation

---

## Testing

### Verify Metrics Endpoint

```bash
# Check metrics endpoint is accessible
curl http://localhost:3857/metrics

# Filter specific metrics
curl http://localhost:3857/metrics | grep api_requests_total

# Check for specific metric
curl http://localhost:3857/metrics | grep tile_cache_hits_total
```

### Generate Load

```bash
# Generate API requests
for i in {1..100}; do
  curl http://localhost:3857/api/v1/locations
done

# Trigger sync
curl -X POST http://localhost:3857/api/v1/sync \
  -H "Authorization: Bearer <token>"
```

### Verify Metrics

```bash
# Check API request counter increased
curl http://localhost:3857/metrics | grep 'api_requests_total{.*endpoint="/api/v1/locations"'

# Check tile cache metrics
curl http://localhost:3857/metrics | grep tile_cache
```

---

## Benefits

1. **Real-Time Visibility** - Instant insights into system performance and health
2. **Proactive Alerting** - Detect issues before they impact users
3. **Performance Optimization** - Identify bottlenecks with detailed latency histograms
4. **Capacity Planning** - Track resource utilization trends
5. **Debugging** - Correlate errors with specific operations
6. **SLA Monitoring** - Measure compliance with performance targets
7. **Cache Efficiency** - Optimize cache strategies with hit ratio metrics

---

## Future Enhancements

1. **Custom Metrics** - Add application-specific business metrics (user engagement, content popularity)
2. **Distributed Tracing** - Integrate with Jaeger or Zipkin for request tracing
3. **Log Aggregation** - Connect with Loki for unified logs + metrics
4. **Service Mesh** - Integrate with Istio/Linkerd for infrastructure-level metrics
5. **Cost Tracking** - Add cloud resource usage metrics for cost optimization

---

**End of Prometheus Metrics Documentation**
**Last Updated:** 2026-01-04
