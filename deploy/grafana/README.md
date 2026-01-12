# Cartographus Grafana Dashboards

Production-grade Grafana dashboards for monitoring Cartographus.

## Dashboards

| Dashboard | UID | Description |
|-----------|-----|-------------|
| [Overview](dashboards/cartographus-overview.json) | `cartographus-overview` | High-level system health, request rates, latency, cache performance |
| [Performance](dashboards/cartographus-performance.json) | `cartographus-performance` | API latency percentiles, throughput, error rates, endpoint analysis |
| [Streaming](dashboards/cartographus-streaming.json) | `cartographus-streaming` | WebSocket connections, sync operations, NATS event processing |
| [Detection](dashboards/cartographus-detection.json) | `cartographus-detection` | Circuit breaker status, DLQ metrics, geolocation performance |
| [Database](dashboards/cartographus-database.json) | `cartographus-database` | DuckDB query performance, spatial operations, error analysis |
| [Auth](dashboards/cartographus-auth.json) | `cartographus-auth` | OIDC authentication, authorization decisions, PAT metrics |

## Quick Start

### Using Docker Compose

```bash
# From the repository root
docker compose -f docker-compose.yml -f deploy/docker-compose.monitoring.yml up -d

# Access Grafana at http://localhost:3000
# Default credentials: admin / admin
```

### Manual Setup

1. **Start Prometheus**:
   ```bash
   docker run -d \
     --name prometheus \
     -p 9090:9090 \
     -v $(pwd)/deploy/prometheus/prometheus.yml:/etc/prometheus/prometheus.yml \
     prom/prometheus:v2.48.0
   ```

2. **Start Grafana**:
   ```bash
   docker run -d \
     --name grafana \
     -p 3000:3000 \
     -v $(pwd)/deploy/grafana/dashboards:/var/lib/grafana/dashboards \
     -v $(pwd)/deploy/grafana/provisioning:/etc/grafana/provisioning \
     grafana/grafana:10.2.0
   ```

3. **Access Grafana** at http://localhost:3000 (admin/admin)

## Key Metrics

### Overview Dashboard
- **Uptime**: Application uptime in seconds
- **Active Requests**: Current in-flight API requests
- **WebSocket Connections**: Active WebSocket connections
- **Active Sessions**: OIDC sessions currently active
- **DLQ Entries**: Dead Letter Queue pending entries

### Performance Dashboard
- **p50/p90/p99 Latency**: API response time percentiles
- **Request Rate**: Requests per second by method/status
- **Error Rate**: 4xx/5xx errors per second
- **Rate Limit Hits**: Rate limiting rejections

### Database Dashboard
- **Query Rate**: DuckDB queries per second
- **Query Latency**: p50/p90/p99 query duration
- **Spatial Operations**: ST_* function usage
- **Query Errors**: Errors by operation type

### Auth Dashboard
- **OIDC Logins**: Login attempts and success rate
- **Active Sessions**: Current session count
- **AuthZ Decisions**: Allow/deny rates by role
- **PAT Validations**: Token usage patterns

## Alerting

Example Prometheus alerting rules (add to `prometheus/rules/cartographus.yml`):

```yaml
groups:
  - name: cartographus
    rules:
      # High error rate
      - alert: HighErrorRate
        expr: sum(rate(api_requests_total{status_code=~"5.."}[5m])) / sum(rate(api_requests_total[5m])) > 0.05
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "High 5xx error rate (> 5%)"

      # High latency
      - alert: HighLatency
        expr: histogram_quantile(0.99, sum(rate(api_request_duration_seconds_bucket[5m])) by (le)) > 1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "p99 latency > 1s"

      # Circuit breaker open
      - alert: CircuitBreakerOpen
        expr: circuit_breaker_state == 2
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "Circuit breaker is OPEN"

      # DLQ backlog
      - alert: DLQBacklog
        expr: dlq_entries_total > 100
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "DLQ has > 100 entries"
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `GRAFANA_ADMIN_PASSWORD` | `admin` | Grafana admin password |
| `GRAFANA_ROOT_URL` | `http://localhost:3000` | Grafana root URL |

## Customization

### Adding Custom Panels

1. Edit the dashboard JSON in `dashboards/`
2. Reload dashboards in Grafana (Settings > Provisioning > Dashboards > Reload)

### Adding New Metrics

1. Add metric definitions in `internal/metrics/metrics.go`
2. Instrument code with `metrics.RecordXxx()` calls
3. Add panels to appropriate dashboard

## Troubleshooting

### No Data in Dashboards

1. Verify Prometheus is scraping Cartographus:
   ```bash
   curl http://localhost:9090/api/v1/targets
   ```

2. Check Cartographus metrics endpoint:
   ```bash
   curl http://localhost:3857/metrics
   ```

3. Verify Grafana datasource is configured correctly

### Dashboard Not Loading

1. Check Grafana logs:
   ```bash
   docker logs cartographus-grafana
   ```

2. Verify provisioning paths are correct

3. Check dashboard JSON syntax:
   ```bash
   python3 -m json.tool dashboards/cartographus-overview.json > /dev/null
   ```
