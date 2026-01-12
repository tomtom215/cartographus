# Monitoring and Alerting Guide

This guide covers monitoring, alerting, and observability for Cartographus deployments.

## Overview

Cartographus provides several observability features:

| Feature | Endpoint | Description |
|---------|----------|-------------|
| Health Check | `/api/v1/health` | Liveness and readiness probes |
| Prometheus Metrics | `/metrics` | Application and runtime metrics |
| Structured Logging | stdout/stderr | JSON-formatted logs |
| Real-time Events | WebSocket | Live event stream |

---

## Health Checks

### Endpoints

```bash
# Basic health check
curl http://localhost:3857/api/v1/health
# Response: {"status":"healthy","timestamp":"2025-12-18T10:30:00Z"}

# Detailed health (if enabled)
curl http://localhost:3857/api/v1/health/detailed
# Response: {"status":"healthy","database":"connected","cache":"active",...}
```

### Health Check Configuration

```bash
# In .env or environment
HEALTH_CHECK_ENABLED=true
HEALTH_CHECK_DETAILED=true
```

### Container Probes

```yaml
# Docker Compose
healthcheck:
  test: ["CMD", "wget", "-qO-", "http://localhost:3857/api/v1/health"]
  interval: 30s
  timeout: 10s
  retries: 3
  start_period: 10s

# Kubernetes (in deployment.yaml)
livenessProbe:
  httpGet:
    path: /api/v1/health
    port: 3857
  initialDelaySeconds: 10
  periodSeconds: 30

readinessProbe:
  httpGet:
    path: /api/v1/health
    port: 3857
  initialDelaySeconds: 5
  periodSeconds: 10
```

---

## Prometheus Metrics

### Available Metrics

Cartographus exposes the following Prometheus metrics:

#### HTTP Metrics
```prometheus
# Request duration histogram
http_request_duration_seconds_bucket{method="GET",path="/api/stats/locations",le="0.1"}
http_request_duration_seconds_sum{method="GET",path="/api/stats/locations"}
http_request_duration_seconds_count{method="GET",path="/api/stats/locations"}

# Request counter
http_requests_total{method="GET",path="/api/stats/locations",status="200"}

# Active connections
http_connections_active
```

#### Database Metrics
```prometheus
# Query duration
db_query_duration_seconds_bucket{query="GetLocationStats",le="0.05"}
db_query_duration_seconds_sum{query="GetLocationStats"}
db_query_duration_seconds_count{query="GetLocationStats"}

# Connection pool
db_connections_active
db_connections_idle
db_connections_max
```

#### Application Metrics
```prometheus
# Playback events processed
playback_events_total{source="plex",type="play"}
playback_events_total{source="jellyfin",type="stop"}

# Sync operations
sync_operations_total{source="tautulli",status="success"}
sync_last_timestamp_seconds{source="plex"}

# Cache metrics
cache_hits_total{cache="analytics"}
cache_misses_total{cache="analytics"}
cache_size_bytes{cache="analytics"}

# WebSocket connections
websocket_connections_active
websocket_messages_total{type="playback_update"}
```

#### Go Runtime Metrics
```prometheus
# Standard Go metrics
go_goroutines
go_memstats_alloc_bytes
go_memstats_heap_inuse_bytes
go_gc_duration_seconds
```

### Prometheus Configuration

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'cartographus'
    static_configs:
      - targets: ['cartographus:3857']
    metrics_path: /metrics
    scrape_interval: 15s
```

### Kubernetes ServiceMonitor

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: cartographus
  namespace: cartographus
  labels:
    release: prometheus
spec:
  selector:
    matchLabels:
      app.kubernetes.io/name: cartographus
  endpoints:
    - port: http
      path: /metrics
      interval: 30s
      scrapeTimeout: 10s
```

---

## Logging

### Log Configuration

```bash
# Environment variables
LOG_LEVEL=info           # debug, info, warn, error
LOG_FORMAT=json          # json, text
LOG_OUTPUT=stdout        # stdout, stderr, file
LOG_FILE=/var/log/cartographus.log  # if LOG_OUTPUT=file
```

### Log Format

JSON logs include:
```json
{
  "level": "info",
  "ts": "2025-12-18T10:30:00.000Z",
  "caller": "api/handlers.go:123",
  "msg": "Request completed",
  "method": "GET",
  "path": "/api/stats/locations",
  "status": 200,
  "duration_ms": 45.2,
  "request_id": "abc123",
  "user": "admin"
}
```

### Log Aggregation

#### Loki (Recommended for Kubernetes)

```yaml
# promtail-config.yml
scrape_configs:
  - job_name: cartographus
    kubernetes_sd_configs:
      - role: pod
    relabel_configs:
      - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_name]
        regex: cartographus
        action: keep
    pipeline_stages:
      - json:
          expressions:
            level: level
            msg: msg
            method: method
            path: path
            status: status
      - labels:
          level:
          method:
```

#### ELK Stack

```yaml
# filebeat.yml
filebeat.inputs:
  - type: container
    paths:
      - /var/lib/docker/containers/*/*.log
    processors:
      - add_kubernetes_metadata:
      - decode_json_fields:
          fields: ["message"]
          target: ""
          overwrite_keys: true

output.elasticsearch:
  hosts: ["elasticsearch:9200"]
  index: "cartographus-%{+yyyy.MM.dd}"
```

---

## Alerting

### Prometheus Alert Rules

```yaml
# alerts.yml
groups:
  - name: cartographus
    rules:
      # High error rate
      - alert: CartographusHighErrorRate
        expr: |
          sum(rate(http_requests_total{job="cartographus",status=~"5.."}[5m]))
          /
          sum(rate(http_requests_total{job="cartographus"}[5m])) > 0.05
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "High error rate in Cartographus"
          description: "Error rate is {{ $value | humanizePercentage }} (threshold: 5%)"

      # Slow responses
      - alert: CartographusSlowResponses
        expr: |
          histogram_quantile(0.95,
            sum(rate(http_request_duration_seconds_bucket{job="cartographus"}[5m])) by (le)
          ) > 2
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "Slow response times in Cartographus"
          description: "95th percentile latency is {{ $value | humanizeDuration }}"

      # Service down
      - alert: CartographusDown
        expr: up{job="cartographus"} == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "Cartographus is down"
          description: "Cartographus instance has been down for more than 1 minute"

      # Database connection issues
      - alert: CartographusDatabaseConnectionLow
        expr: db_connections_active{job="cartographus"} < 1
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "Database connection issues"
          description: "No active database connections"

      # High memory usage
      - alert: CartographusHighMemoryUsage
        expr: |
          go_memstats_heap_inuse_bytes{job="cartographus"}
          /
          (1024 * 1024 * 1024) > 0.8
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "High memory usage"
          description: "Memory usage is {{ $value | humanize }}GB"

      # Sync failures
      - alert: CartographusSyncFailing
        expr: |
          increase(sync_operations_total{job="cartographus",status="error"}[1h]) > 5
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Sync operations failing"
          description: "{{ $value }} sync failures in the last hour"
```

### AlertManager Configuration

```yaml
# alertmanager.yml
route:
  receiver: 'default'
  routes:
    - match:
        severity: critical
      receiver: 'pagerduty'
    - match:
        severity: warning
      receiver: 'slack'

receivers:
  - name: 'default'
    email_configs:
      - to: 'ops@example.com'

  - name: 'slack'
    slack_configs:
      - api_url: 'https://hooks.slack.com/services/...'
        channel: '#alerts'
        title: '{{ .GroupLabels.alertname }}'
        text: '{{ .CommonAnnotations.description }}'

  - name: 'pagerduty'
    pagerduty_configs:
      - service_key: 'your-pagerduty-key'
```

---

## Dashboards

### Grafana Dashboard

Import the included Grafana dashboard or create one with these panels:

#### Overview Panel
- Request rate (total, by status code)
- Error rate percentage
- Response time (p50, p95, p99)
- Active connections

#### Database Panel
- Query duration histogram
- Connection pool utilization
- Queries per second

#### Application Panel
- Playback events by source
- Sync operation status
- Cache hit/miss ratio
- WebSocket connections

#### System Panel
- Memory usage (heap, stack)
- Goroutine count
- GC pause duration
- CPU usage (if available)

### Example Dashboard JSON

```json
{
  "title": "Cartographus Overview",
  "panels": [
    {
      "title": "Request Rate",
      "type": "graph",
      "targets": [
        {
          "expr": "sum(rate(http_requests_total{job=\"cartographus\"}[5m])) by (status)",
          "legendFormat": "{{status}}"
        }
      ]
    },
    {
      "title": "Response Time",
      "type": "graph",
      "targets": [
        {
          "expr": "histogram_quantile(0.95, sum(rate(http_request_duration_seconds_bucket{job=\"cartographus\"}[5m])) by (le))",
          "legendFormat": "p95"
        },
        {
          "expr": "histogram_quantile(0.50, sum(rate(http_request_duration_seconds_bucket{job=\"cartographus\"}[5m])) by (le))",
          "legendFormat": "p50"
        }
      ]
    }
  ]
}
```

---

## Troubleshooting

### Common Issues

#### High Memory Usage
```bash
# Check current memory
curl -s http://localhost:3857/metrics | grep go_memstats_heap_inuse_bytes

# Possible causes:
# - Large result sets in memory
# - Cache size too large
# - Memory leak (check goroutine count)
```

#### Slow Queries
```bash
# Enable debug logging
LOG_LEVEL=debug ./cartographus

# Check query metrics
curl -s http://localhost:3857/metrics | grep db_query_duration
```

#### Connection Pool Exhaustion
```bash
# Check connection counts
curl -s http://localhost:3857/metrics | grep db_connections

# Increase pool size in config
DB_MAX_CONNECTIONS=50
```

### Debug Endpoints

```bash
# Runtime profiling (if enabled)
curl http://localhost:3857/debug/pprof/heap > heap.prof
go tool pprof heap.prof

# Goroutine dump
curl http://localhost:3857/debug/pprof/goroutine?debug=2
```

---

## Best Practices

1. **Set appropriate alert thresholds** based on your baseline
2. **Use recording rules** for frequently computed metrics
3. **Implement log rotation** to prevent disk exhaustion
4. **Monitor the monitoring** - alert on Prometheus/AlertManager issues
5. **Review dashboards regularly** and remove unused panels
6. **Test alerts** periodically to ensure they fire correctly
7. **Document runbooks** for each alert with remediation steps

---

## Related Documentation

- [SECURITY.md](../SECURITY.md) - Security configuration
- [SECRETS_MANAGEMENT.md](./SECRETS_MANAGEMENT.md) - Secrets handling
- [Kubernetes Deployment](../deploy/kubernetes/README.md) - K8s deployment
- [API Reference](./API-REFERENCE.md) - API documentation
