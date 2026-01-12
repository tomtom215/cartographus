# Alertmanager Configuration

This directory contains the Alertmanager configuration for Cartographus monitoring.

## Overview

Alertmanager handles alerts from Prometheus and routes them to various notification channels (Discord, Slack, Email, Telegram, PagerDuty, etc.).

## Files

| File | Purpose |
|------|---------|
| `alertmanager.yml` | Main configuration (routing, receivers, inhibitions) |
| `templates/cartographus.tmpl` | Notification message templates |

## Quick Start

1. Edit `alertmanager.yml` to uncomment your preferred notification channels
2. Set environment variables for credentials (see below)
3. Start the monitoring stack:

```bash
docker compose -f docker-compose.yml -f deploy/docker-compose.monitoring.yml up -d
```

## Notification Channels

### Discord

1. Create a webhook in your Discord server (Server Settings > Integrations > Webhooks)
2. Set the environment variable:

```bash
export DISCORD_WEBHOOK_URL="https://discord.com/api/webhooks/..."
```

3. Uncomment the `webhook_configs` section in `alertmanager.yml`

### Slack

1. Create a Slack App with Incoming Webhooks
2. Set the environment variable:

```bash
export SLACK_WEBHOOK_URL="https://hooks.slack.com/services/..."
```

3. Uncomment the `slack_configs` section in `alertmanager.yml`

### Email (SMTP)

1. Configure SMTP settings in the `global` section:

```yaml
global:
  smtp_smarthost: 'smtp.gmail.com:587'
  smtp_from: 'alerts@yourdomain.com'
  smtp_auth_username: 'your-email@gmail.com'
  smtp_auth_password: 'your-app-password'
  smtp_require_tls: true
```

2. Uncomment the `email_configs` section in receivers

### Telegram

1. Create a bot via @BotFather
2. Get your chat ID
3. Set environment variables:

```bash
export TELEGRAM_BOT_TOKEN="123456789:ABC..."
export TELEGRAM_CHAT_ID="123456789"
```

4. Uncomment the `telegram_configs` section

### PagerDuty

1. Create a service in PagerDuty and get the integration key
2. Set the environment variable:

```bash
export PAGERDUTY_SERVICE_KEY="your-integration-key"
```

3. Uncomment the `pagerduty_configs` section

## Alert Severity Levels

| Severity | Description | Default Action |
|----------|-------------|----------------|
| `critical` | Service down or major functionality impacted | Immediate notification, 1h repeat |
| `warning` | Degraded performance or potential issues | 1m group wait, 4h repeat |
| `info` | Informational events | 5m group wait, 24h repeat |

## Alert Categories

### Health Alerts (`cartographus_health`)
- `CartographusDown` - Application unreachable
- `HighErrorRate` - >5% HTTP 5xx responses
- `CriticalErrorRate` - >15% HTTP 5xx responses

### Latency Alerts (`cartographus_latency`)
- `HighP99Latency` - p99 >2s
- `CriticalP99Latency` - p99 >5s
- `SlowEndpoint` - Individual endpoint p95 >3s

### Database Alerts (`cartographus_database`)
- `DatabaseSlowQueries` - p95 query time >1s
- `DatabaseQueryErrors` - Error rate increasing
- `HighDatabaseConnections` - >10 active connections

### Circuit Breaker Alerts (`cartographus_circuit_breaker`)
- `CircuitBreakerOpen` - Upstream service failure
- `CircuitBreakerHalfOpen` - Recovery testing
- `CircuitBreakerTrips` - Multiple trips in 15min

### DLQ Alerts (`cartographus_dlq`)
- `DLQEntriesGrowing` - >100 entries
- `DLQCritical` - >1000 entries
- `DLQProcessingStalled` - Entries not being processed

### Authentication Alerts (`cartographus_auth`)
- `HighAuthFailureRate` - >30% auth failures
- `AuthServiceSlow` - p95 >5s
- `BruteForceAttemptDetected` - High failure rate from single IP

### WebSocket Alerts (`cartographus_websocket`)
- `WebSocketConnectionDrop` - >50% connection loss
- `HighWebSocketErrors` - High error rate

### Sync Alerts (`cartographus_sync`)
- `SyncServiceStalled` - No sync in 1 hour
- `SyncErrorsHigh` - High error rate
- `MediaServerUnreachable` - Cannot connect to server

### Security Alerts (`cartographus_detection`)
- `SecurityAlertTriggered` - Detection rule fired
- `ImpossibleTravelDetected` - Suspicious travel pattern
- `HighTrustScoreDecrements` - Unusual trust activity

## Inhibition Rules

1. **CartographusDown inhibits warnings** - If the app is down, don't spam with secondary alerts
2. **Critical inhibits warning** - For same alertname, critical supersedes warning

## Testing

### Check Configuration

```bash
docker exec cartographus-alertmanager amtool check-config /etc/alertmanager/alertmanager.yml
```

### Send Test Alert

```bash
curl -X POST http://localhost:9093/api/v1/alerts \
  -H "Content-Type: application/json" \
  -d '[{
    "labels": {
      "alertname": "TestAlert",
      "severity": "warning"
    },
    "annotations": {
      "summary": "Test alert",
      "description": "This is a test alert from manual curl"
    }
  }]'
```

### View Active Alerts

```bash
curl -s http://localhost:9093/api/v1/alerts | jq
```

### Silence an Alert

```bash
docker exec cartographus-alertmanager amtool silence add \
  alertname="TestAlert" \
  --comment="Testing silences" \
  --author="admin" \
  --duration="1h"
```

## Runbook URLs

Each alert includes a `runbook_url` annotation pointing to troubleshooting documentation:
- https://github.com/tomtom215/cartographus/blob/main/docs/TROUBLESHOOTING.md

## Grafana Integration

The monitoring stack is pre-configured to:
1. Use Prometheus as the default datasource
2. Enable Grafana Unified Alerting
3. Allow forwarding alerts to Alertmanager

To create alerts in Grafana:
1. Open a dashboard panel
2. Click "Alert" tab
3. Configure conditions and notifications
4. Alerts will route through Alertmanager

## Environment Variables

| Variable | Description |
|----------|-------------|
| `DISCORD_WEBHOOK_URL` | Discord webhook URL |
| `SLACK_WEBHOOK_URL` | Slack webhook URL |
| `TELEGRAM_BOT_TOKEN` | Telegram bot token |
| `TELEGRAM_CHAT_ID` | Telegram chat/channel ID |
| `PAGERDUTY_SERVICE_KEY` | PagerDuty integration key |
| `SECURITY_WEBHOOK_URL` | Custom webhook for security alerts |
| `ALERTMANAGER_EXTERNAL_URL` | External URL for Alertmanager links |

## Troubleshooting

### Alerts Not Firing

1. Check Prometheus targets: http://localhost:9090/targets
2. Check alert rules: http://localhost:9090/alerts
3. Check Alertmanager status: http://localhost:9093/#/status

### Notifications Not Sent

1. Check Alertmanager logs: `docker logs cartographus-alertmanager`
2. Verify receiver configuration in `alertmanager.yml`
3. Check environment variables are set correctly

### Too Many Alerts

1. Adjust alert thresholds in `prometheus/rules/cartographus.yml`
2. Add silences for known issues
3. Configure inhibition rules
