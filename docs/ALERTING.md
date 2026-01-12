# Alerting Configuration Guide

This guide covers Prometheus alerting rules and Alertmanager configuration for Cartographus deployments.

## Overview

Cartographus provides pre-built alerting rules for comprehensive operational monitoring. The alerting system is organized into functional groups:

| Alert Group | Purpose | Alert Count |
|-------------|---------|-------------|
| cartographus_general | Core application health | 6 alerts |
| cartographus_database | DuckDB query performance | 6 alerts |
| cartographus_sync | Multi-server sync operations | 6 alerts |
| cartographus_circuit_breaker | External API resilience | 4 alerts |
| cartographus_auth | Authentication and rate limiting | 6 alerts |
| cartographus_dlq | Dead Letter Queue monitoring | 4 alerts |
| cartographus_detection | Security detection engine | 4 alerts |
| cartographus_wal | Write-Ahead Log health | 12 alerts |
| cartographus_backup | Database backup operations | 6 alerts |

**Total: 54 pre-built alerting rules**

---

## Quick Start

### 1. Deploy Alert Rules

Copy the alert rules to your Prometheus configuration:

```bash
# Copy alert rules
cp deploy/prometheus/rules/cartographus.yml /etc/prometheus/rules/

# Validate rules
promtool check rules /etc/prometheus/rules/cartographus.yml

# Reload Prometheus
curl -X POST http://localhost:9090/-/reload
```

### 2. Configure Alertmanager

Copy the Alertmanager configuration:

```bash
# Copy configuration
cp deploy/alertmanager/alertmanager.yml /etc/alertmanager/
cp -r deploy/alertmanager/templates/ /etc/alertmanager/templates/

# Validate configuration
amtool check-config /etc/alertmanager/alertmanager.yml

# Reload Alertmanager
curl -X POST http://localhost:9093/-/reload
```

### 3. Update Receiver Credentials

Edit `/etc/alertmanager/alertmanager.yml` to add your notification credentials:

```yaml
receivers:
  - name: 'slack-critical'
    slack_configs:
      - api_url: 'YOUR_SLACK_WEBHOOK_URL'  # Replace with actual URL
        channel: '#alerts-critical'

  - name: 'pagerduty-critical'
    pagerduty_configs:
      - routing_key: 'YOUR_PAGERDUTY_KEY'  # Replace with actual key
```

---

## Alert Categories

### WAL (Write-Ahead Log) Alerts

The Write-Ahead Log ensures event durability by persisting events to BadgerDB before database insertion. Issues here may cause data loss.

| Alert | Severity | Condition | Description |
|-------|----------|-----------|-------------|
| WALPendingEntriesHigh | warning | > 1000 entries for 5m | Events queued but not committed to database |
| WALPendingEntriesCritical | critical | > 5000 entries for 2m | WAL backlog is critically high |
| WALWriteFailures | critical | > 0 failures in 5m | Events failing to persist to WAL |
| WALNATSPublishFailures | warning | > 10 failures in 5m | Events failing to publish to NATS |
| WALMaxRetriesExceeded | critical | > 0 in 15m | Events exceeding max retry attempts |
| WALExpiredEntries | warning | > 0 in 1h | Events expiring before processing |
| WALCompactionLag | warning | > 1 hour | WAL compaction is delayed |
| WALDBSizeHigh | warning | > 1GB for 30m | BadgerDB size is growing |
| WALDBSizeCritical | critical | > 5GB for 10m | BadgerDB size is critical |
| WALSlowWrites | warning | p95 > 100ms for 5m | WAL writes are slow |
| ConsumerWALPendingHigh | warning | > 500 entries for 5m | Consumer WAL has high pending entries |
| ConsumerWALFailures | critical | > 0 in 5m | Consumer WAL processing failures |

**Runbook:** When WAL alerts fire:
1. Check BadgerDB disk space: `df -h /data/wal`
2. Check WAL metrics: `curl localhost:3857/metrics | grep wal_`
3. If backlog is high, consider scaling or investigating downstream issues
4. For write failures, check BadgerDB logs and disk I/O

### Backup Alerts

Backup alerts monitor the scheduled backup system to ensure data protection.

| Alert | Severity | Condition | Description |
|-------|----------|-----------|-------------|
| BackupNotRunRecently | warning | > 48 hours since last backup | No recent backups detected |
| BackupFailed | critical | Failure in last 2 hours | A backup operation failed |
| BackupStorageLow | warning | < 20% space remaining | Backup storage is getting low |
| BackupStorageCritical | critical | < 5% space remaining | Backup storage is critically low |
| ScheduledBackupMissed | warning | Missed scheduled backup | Scheduled backup didn't run |
| RetentionNotApplied | info | > 7 days since retention applied | Retention policy hasn't run recently |

**Runbook:** When backup alerts fire:
1. Check backup status: `curl localhost:3857/api/v1/backups | jq`
2. Check backup storage: `df -h /data/backups`
3. Verify schedule configuration: `curl localhost:3857/api/v1/backups/schedule`
4. Check backup logs for errors
5. Manually trigger backup if needed: `curl -X POST localhost:3857/api/v1/backups/schedule/trigger`

### Authentication Alerts

| Alert | Severity | Condition | Description |
|-------|----------|-----------|-------------|
| HighAuthenticationFailures | warning | > 10 failures/min for 5m | Elevated login failures |
| AuthenticationFailureSpike | critical | > 50 failures/min for 2m | Possible brute force attack |
| RateLimitExceeded | warning | > 100 rejections/min for 5m | API rate limiting triggered |
| JWTValidationFailures | warning | > 10 failures/min for 5m | JWT token validation issues |
| SessionCreationFailures | warning | > 5 failures/min for 5m | Session creation problems |
| OIDCProviderUnreachable | critical | OIDC errors for 5m | Cannot reach identity provider |

### Circuit Breaker Alerts

| Alert | Severity | Condition | Description |
|-------|----------|-----------|-------------|
| CircuitBreakerOpen | warning | Open for 5m | Circuit breaker triggered |
| CircuitBreakerHalfOpen | info | Half-open for 10m | Testing if external service recovered |
| HighCircuitBreakerTrips | warning | > 5 trips in 1h | Frequent circuit breaker activations |
| TautulliCircuitBreakerOpen | critical | Open for 10m | Tautulli API unavailable |

### Database Alerts

| Alert | Severity | Condition | Description |
|-------|----------|-----------|-------------|
| HighDatabaseQueryLatency | warning | p95 > 500ms for 5m | Slow database queries |
| DatabaseQueryLatencyCritical | critical | p95 > 2s for 2m | Very slow database queries |
| HighDatabaseErrorRate | warning | > 1% errors for 5m | Database query failures |
| DatabaseConnectionPoolExhausted | critical | Pool exhausted for 2m | No available DB connections |
| DuckDBExtensionFailure | critical | Extension load failure | Required extension unavailable |
| LargeQueryResultSet | warning | Result sets > 100K rows | Query returning too much data |

### Sync Alerts

| Alert | Severity | Condition | Description |
|-------|----------|-----------|-------------|
| SyncOperationFailed | warning | Failure in last 1h | Sync operation failed |
| SyncNotRunning | critical | No sync for > 1h | Sync operations have stopped |
| HighSyncLatency | warning | Sync taking > 5m | Sync operations are slow |
| MediaServerUnreachable | critical | Cannot reach server for 5m | Media server connection lost |
| SyncDataLag | warning | Data > 30m behind | Sync is falling behind |
| DuplicateEventsDetected | info | Duplicates detected | Deduplication triggered |

---

## Notification Templates

Custom notification templates are provided for different alert types:

### Slack Templates

| Template | Use Case |
|----------|----------|
| `slack.title` | General alert title |
| `slack.text` | General alert body |
| `slack.wal` | WAL-specific formatting |
| `slack.backup` | Backup-specific formatting |

### Email Templates

| Template | Use Case |
|----------|----------|
| `email.subject` | Email subject line |
| `email.html` | HTML email body |
| `email.wal.section` | WAL alert email section |
| `email.backup.section` | Backup alert email section |

### Other Templates

| Template | Use Case |
|----------|----------|
| `telegram.message` | Telegram bot messages |
| `discord.message` | Discord webhook JSON |
| `webhook.json` | Generic webhook payload |
| `pagerduty.wal.details` | PagerDuty WAL custom details |
| `pagerduty.backup.details` | PagerDuty backup custom details |

---

## Alertmanager Routes

The default routing configuration prioritizes alerts by severity:

```
                    ┌─────────────────┐
                    │  All Alerts     │
                    └────────┬────────┘
                             │
        ┌────────────────────┼────────────────────┐
        │                    │                    │
        ▼                    ▼                    ▼
  ┌───────────┐        ┌───────────┐        ┌───────────┐
  │ Critical  │        │  Warning  │        │   Info    │
  │ (page)    │        │  (slack)  │        │  (email)  │
  └───────────┘        └───────────┘        └───────────┘
        │                    │                    │
        ▼                    ▼                    ▼
  PagerDuty +          Slack +              Email only
  Slack + Email        Email
```

### Route Configuration

```yaml
route:
  receiver: 'default-email'
  group_by: ['alertname', 'severity', 'component']
  group_wait: 30s
  group_interval: 5m
  repeat_interval: 4h

  routes:
    # Critical alerts: PagerDuty + Slack + Email
    - match:
        severity: critical
      receiver: 'pagerduty-critical'
      continue: true
    - match:
        severity: critical
      receiver: 'slack-critical'
      continue: true

    # Warning alerts: Slack + Email
    - match:
        severity: warning
      receiver: 'slack-warning'
      continue: true

    # WAL-specific routing
    - match:
        component: wal
      receiver: 'slack-ops'
      group_wait: 1m

    # Backup-specific routing
    - match:
        component: backup
      receiver: 'slack-ops'
```

---

## Inhibition Rules

Inhibition rules prevent alert floods by suppressing related alerts:

| When This Fires | Suppress These |
|-----------------|----------------|
| CartographusDown | All other Cartographus alerts |
| DatabaseConnectionPoolExhausted | Database query alerts |
| WALPendingEntriesCritical | WALPendingEntriesHigh |
| BackupStorageCritical | BackupStorageLow |

---

## Customization

### Adding Custom Receivers

Add new receivers in `/etc/alertmanager/alertmanager.yml`:

```yaml
receivers:
  - name: 'my-team-slack'
    slack_configs:
      - api_url: 'https://hooks.slack.com/services/XXX/YYY/ZZZ'
        channel: '#my-team-alerts'
        username: 'Cartographus'
        icon_emoji: ':world_map:'
        title: '{{ template "slack.title" . }}'
        text: '{{ template "slack.text" . }}'
        send_resolved: true
```

### Adding Custom Alert Rules

Add new rules to `/etc/prometheus/rules/cartographus.yml`:

```yaml
groups:
  - name: my_custom_alerts
    rules:
      - alert: MyCustomAlert
        expr: my_custom_metric > 100
        for: 5m
        labels:
          severity: warning
          component: custom
        annotations:
          summary: "Custom alert fired"
          description: "Value is {{ $value }}"
```

### Modifying Alert Thresholds

Edit thresholds in the alert rules file. For example, to change WAL pending entries threshold:

```yaml
# Original
- alert: WALPendingEntriesHigh
  expr: wal_pending_entries > 1000

# Modified for higher throughput environments
- alert: WALPendingEntriesHigh
  expr: wal_pending_entries > 5000
```

---

## Testing Alerts

### Test Alert Rules

```bash
# Validate alert rules syntax
promtool check rules /etc/prometheus/rules/cartographus.yml

# Test specific expressions
promtool query instant http://localhost:9090 'wal_pending_entries > 1000'
```

### Test Alertmanager Configuration

```bash
# Validate configuration
amtool check-config /etc/alertmanager/alertmanager.yml

# Test routing
amtool config routes test \
  --alertmanager.url=http://localhost:9093 \
  severity=critical component=wal

# Send test alert
amtool alert add \
  --alertmanager.url=http://localhost:9093 \
  alertname=TestAlert severity=warning \
  --annotation.summary="Test alert" \
  --annotation.description="This is a test"
```

### Generate Test Alerts

```bash
# Trigger WAL backlog (for testing only)
curl -X POST http://localhost:3857/api/v1/debug/wal/simulate-backlog

# View active alerts
curl http://localhost:9090/api/v1/alerts | jq '.data.alerts[]'
```

---

## Monitoring the Monitoring

Add these meta-alerts to monitor your alerting infrastructure:

```yaml
groups:
  - name: alerting_meta
    rules:
      - alert: PrometheusTargetDown
        expr: up{job="cartographus"} == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "Prometheus cannot scrape Cartographus"

      - alert: AlertmanagerNotificationFailing
        expr: alertmanager_notifications_failed_total > 0
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Alertmanager failing to send notifications"
```

---

## Troubleshooting

### Alerts Not Firing

1. Check Prometheus is scraping the target:
   ```bash
   curl http://localhost:9090/api/v1/targets | jq '.data.activeTargets[] | select(.labels.job=="cartographus")'
   ```

2. Verify the metric exists:
   ```bash
   curl http://localhost:3857/metrics | grep <metric_name>
   ```

3. Check if alert expression evaluates correctly:
   ```bash
   promtool query instant http://localhost:9090 '<alert_expression>'
   ```

### Notifications Not Sending

1. Check Alertmanager logs:
   ```bash
   docker logs alertmanager 2>&1 | tail -50
   ```

2. Verify receiver configuration:
   ```bash
   amtool config routes show --alertmanager.url=http://localhost:9093
   ```

3. Check notification status:
   ```bash
   curl http://localhost:9093/api/v2/alerts | jq
   ```

### Too Many Alerts

1. Review alert thresholds - they may be too sensitive
2. Add inhibition rules to suppress related alerts
3. Increase `group_wait` and `group_interval` to batch alerts
4. Consider using `for` duration to avoid flapping alerts

---

## File Locations

| File | Purpose |
|------|---------|
| `deploy/prometheus/rules/cartographus.yml` | Prometheus alert rules |
| `deploy/alertmanager/alertmanager.yml` | Alertmanager configuration |
| `deploy/alertmanager/templates/cartographus.tmpl` | Notification templates |
| `deploy/prometheus/prometheus.yml` | Prometheus scrape configuration |

---

## Related Documentation

- [MONITORING.md](./MONITORING.md) - General monitoring guide
- [PROMETHEUS_METRICS.md](./PROMETHEUS_METRICS.md) - Available metrics reference
- [BACKUP_DISASTER_RECOVERY.md](./BACKUP_DISASTER_RECOVERY.md) - Backup configuration
- [TROUBLESHOOTING.md](./TROUBLESHOOTING.md) - General troubleshooting

---

**Last Updated:** 2026-01-07
