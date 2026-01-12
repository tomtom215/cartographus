# Backup and Disaster Recovery Guide

Comprehensive guide for backup management and disaster recovery in Cartographus.

**Related Documentation**:
- [CONFIGURATION_REFERENCE.md](./CONFIGURATION_REFERENCE.md) - All configuration options
- [PRODUCTION_DEPLOYMENT.md](./PRODUCTION_DEPLOYMENT.md) - Deployment strategies
- [DATABASE_MIGRATION.md](./DATABASE_MIGRATION.md) - Database migration procedures

---

## Table of Contents

1. [Overview](#overview)
2. [Backup Types](#backup-types)
3. [Configuration](#configuration)
4. [Manual Backup Operations](#manual-backup-operations)
5. [Automated Backups](#automated-backups)
6. [Retention Policies](#retention-policies)
7. [Restore Procedures](#restore-procedures)
8. [Disaster Recovery Playbook](#disaster-recovery-playbook)
9. [Monitoring and Alerts](#monitoring-and-alerts)
10. [Best Practices](#best-practices)

---

## Overview

Cartographus includes a production-ready backup system with:

- **Multiple backup types**: Full, database-only, and configuration-only backups
- **Automatic scheduling**: Configurable intervals with preferred time-of-day
- **Retention policies**: Grandfather-father-son (GFS) style retention
- **Compression**: Gzip or Zstd compression to reduce storage
- **Integrity verification**: SHA-256 checksums for all backups
- **Pre-restore safety backups**: Automatic backup before any restore operation
- **Post-restore verification**: Database integrity checks after restoration

### Architecture

```
┌──────────────┐     ┌─────────────────┐     ┌──────────────┐
│   Scheduler  │────▶│  BackupManager  │────▶│   Storage    │
└──────────────┘     └─────────────────┘     └──────────────┘
                            │                       │
                            ▼                       ▼
                     ┌──────────────┐      ┌──────────────┐
                     │   DuckDB     │      │   Archives   │
                     │   Database   │      │   (.tar.gz)  │
                     └──────────────┘      └──────────────┘
```

---

## Backup Types

### Full Backup (`full`)

Complete backup including database, configuration, and metadata.

**Contents**:
- DuckDB database file (`cartographus.duckdb`)
- DuckDB WAL file (if present)
- Configuration snapshot (sanitized, no secrets)
- Backup metadata

**Use When**:
- Scheduled automatic backups
- Before major upgrades
- Before database migrations
- Disaster recovery preparation

### Database Backup (`database`)

DuckDB database files only.

**Contents**:
- DuckDB database file (`cartographus.duckdb`)
- DuckDB WAL file (if present)

**Use When**:
- Quick data-only backups
- Before data manipulation operations
- Storage-constrained environments

### Configuration Backup (`config`)

Application configuration only (sanitized).

**Contents**:
- Configuration values (secrets excluded)
- Environment variable mappings
- Feature flags

**Use When**:
- Before configuration changes
- Documentation purposes
- Environment replication

---

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `BACKUP_ENABLED` | `true` | Enable backup functionality |
| `BACKUP_DIR` | `/data/backups` | Directory for backup storage (must be absolute path) |

### Schedule Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `BACKUP_SCHEDULE_ENABLED` | `true` | Enable automatic scheduled backups |
| `BACKUP_INTERVAL` | `24h` | Backup interval (minimum 1h) |
| `BACKUP_PREFERRED_HOUR` | `2` | Hour of day for backups (0-23, 24h format) |
| `BACKUP_TYPE` | `full` | Type for scheduled backups: `full`, `database`, `config` |
| `BACKUP_PRE_SYNC` | `false` | Create backup before each sync operation |

### Retention Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `BACKUP_RETENTION_MIN_COUNT` | `3` | Minimum backups to keep regardless of age |
| `BACKUP_RETENTION_MAX_COUNT` | `50` | Maximum backups to keep (0 = unlimited) |
| `BACKUP_RETENTION_MAX_DAYS` | `90` | Delete backups older than N days |
| `BACKUP_RETENTION_KEEP_RECENT_HOURS` | `24` | Keep all backups from last N hours |
| `BACKUP_RETENTION_KEEP_DAILY_DAYS` | `7` | Keep one daily backup for N days |
| `BACKUP_RETENTION_KEEP_WEEKLY_WEEKS` | `4` | Keep one weekly backup for N weeks |
| `BACKUP_RETENTION_KEEP_MONTHLY_MONTHS` | `6` | Keep one monthly backup for N months |

### Compression Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `BACKUP_COMPRESSION_ENABLED` | `true` | Enable backup compression |
| `BACKUP_COMPRESSION_LEVEL` | `6` | Compression level (1-9, higher = more compression) |
| `BACKUP_COMPRESSION_ALGORITHM` | `gzip` | Algorithm: `gzip` or `zstd` |

### Encryption Configuration (Optional)

| Variable | Default | Description |
|----------|---------|-------------|
| `BACKUP_ENCRYPTION_ENABLED` | `false` | Enable backup encryption |
| `BACKUP_ENCRYPTION_KEY` | (none) | AES-256 encryption key (min 32 characters) |
| `BACKUP_ENCRYPTION_KEY_ID` | (none) | Key identifier for rotation support |

### Notification Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `BACKUP_NOTIFY_SUCCESS` | `false` | Send notification on successful backup |
| `BACKUP_NOTIFY_FAILURE` | `true` | Send notification on backup failure |
| `BACKUP_NOTIFY_CLEANUP` | `false` | Send notification on retention cleanup |
| `BACKUP_WEBHOOK_URL` | (none) | Webhook URL for notifications |

### Example Configuration

```bash
# .env file
BACKUP_ENABLED=true
BACKUP_DIR=/data/backups

# Schedule: Daily at 2 AM
BACKUP_SCHEDULE_ENABLED=true
BACKUP_INTERVAL=24h
BACKUP_PREFERRED_HOUR=2
BACKUP_TYPE=full

# Retention: Keep backups for 90 days with GFS rotation
BACKUP_RETENTION_MIN_COUNT=5
BACKUP_RETENTION_MAX_COUNT=100
BACKUP_RETENTION_MAX_DAYS=90
BACKUP_RETENTION_KEEP_DAILY_DAYS=14
BACKUP_RETENTION_KEEP_WEEKLY_WEEKS=8
BACKUP_RETENTION_KEEP_MONTHLY_MONTHS=12

# Compression: High compression for long-term storage
BACKUP_COMPRESSION_ENABLED=true
BACKUP_COMPRESSION_LEVEL=9
BACKUP_COMPRESSION_ALGORITHM=gzip

# Notifications: Alert on failures
BACKUP_NOTIFY_FAILURE=true
BACKUP_WEBHOOK_URL=https://hooks.slack.com/services/xxx
```

---

## Manual Backup Operations

### Creating a Manual Backup

```bash
# Using the CLI (if available)
./cartographus backup create --type full --notes "Pre-upgrade backup"

# Using the API
curl -X POST http://localhost:3857/api/v1/backup/create \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"type": "full", "notes": "Pre-upgrade backup"}'
```

### Listing Backups

```bash
# List all backups
curl http://localhost:3857/api/v1/backup/list \
  -H "Authorization: Bearer $TOKEN"

# List with filters
curl "http://localhost:3857/api/v1/backup/list?type=full&status=completed&limit=10" \
  -H "Authorization: Bearer $TOKEN"
```

### Downloading a Backup

```bash
# Download by ID
curl -o backup.tar.gz http://localhost:3857/api/v1/backup/download/{backup_id} \
  -H "Authorization: Bearer $TOKEN"
```

### Validating a Backup

```bash
# Validate backup integrity
curl -X POST http://localhost:3857/api/v1/backup/validate/{backup_id} \
  -H "Authorization: Bearer $TOKEN"
```

Response:
```json
{
  "valid": true,
  "checksum_valid": true,
  "archive_readable": true,
  "files_complete": true,
  "database_valid": true,
  "errors": [],
  "warnings": []
}
```

---

## Automated Backups

### Backup Scheduler

When `BACKUP_SCHEDULE_ENABLED=true`, the backup manager runs a scheduler that:

1. Calculates next backup time based on `BACKUP_INTERVAL` and `BACKUP_PREFERRED_HOUR`
2. Creates backups of type specified by `BACKUP_TYPE`
3. Runs retention cleanup after each successful backup
4. Sends notifications based on configuration

### Pre-Sync Backups

When `BACKUP_PRE_SYNC=true`, a backup is created before each sync operation:

- Triggered by: Manual sync, scheduled sync, webhook-triggered sync
- Type: `database` (for speed)
- Retention: Subject to normal retention policies

### Backup Triggers

| Trigger | Description |
|---------|-------------|
| `manual` | User-initiated via API or CLI |
| `scheduled` | Automatic scheduled backup |
| `pre_sync` | Created before sync operation |
| `pre_restore` | Created before restore operation |
| `retention` | Created by retention policy |

---

## Retention Policies

### Grandfather-Father-Son (GFS) Retention

The default retention policy uses a GFS-style rotation:

```
Timeline (days from today):
│
├─ 0-1:   Keep ALL backups (recent recovery)
│
├─ 1-7:   Keep DAILY backups (7 total)
│
├─ 7-28:  Keep WEEKLY backups (4 total)
│
├─ 28-180: Keep MONTHLY backups (6 total)
│
└─ >180:   Delete (unless min_count not met)
```

### Retention Algorithm

1. **Keep Recent**: All backups within `KEEP_RECENT_HOURS` are kept
2. **Keep Daily**: Best backup per day for `KEEP_DAILY_DAYS` days
3. **Keep Weekly**: Best backup per week for `KEEP_WEEKLY_WEEKS` weeks
4. **Keep Monthly**: Best backup per month for `KEEP_MONTHLY_MONTHS` months
5. **Apply Max Age**: Delete backups older than `MAX_DAYS`
6. **Apply Max Count**: Delete oldest backups exceeding `MAX_COUNT`
7. **Enforce Minimum**: Never delete below `MIN_COUNT`

### Storage Estimation

| Data Size | Daily Backups | Monthly Storage |
|-----------|---------------|-----------------|
| 100 MB | 100 MB/day | ~3 GB |
| 1 GB | 1 GB/day | ~30 GB |
| 10 GB | 10 GB/day | ~300 GB |

*With compression (level 6), expect 60-70% reduction.*

---

## Restore Procedures

### Standard Restore

```bash
# 1. List available backups
curl http://localhost:3857/api/v1/backup/list \
  -H "Authorization: Bearer $TOKEN"

# 2. Validate the backup first
curl -X POST http://localhost:3857/api/v1/backup/validate/{backup_id} \
  -H "Authorization: Bearer $TOKEN"

# 3. Restore with pre-restore backup (recommended)
curl -X POST http://localhost:3857/api/v1/backup/restore/{backup_id} \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "create_pre_restore_backup": true,
    "verify_after_restore": true
  }'
```

### Restore Options

| Option | Default | Description |
|--------|---------|-------------|
| `validate_only` | `false` | Only validate, don't restore |
| `create_pre_restore_backup` | `true` | Create safety backup before restore |
| `stop_services` | `true` | Stop services during restore |
| `restore_database` | auto | Restore database (based on backup type) |
| `restore_config` | auto | Restore configuration (based on backup type) |
| `force_restore` | `false` | Skip validation (dangerous) |
| `verify_after_restore` | `true` | Verify database after restore |

### Restore Response

```json
{
  "success": true,
  "backup_id": "20260101-020000-full",
  "pre_restore_backup_id": "20260102-143025-pre-restore",
  "database_restored": true,
  "config_restored": true,
  "records_restored": 150000,
  "duration_ms": 45000,
  "restart_required": true,
  "warnings": []
}
```

### Post-Restore Steps

1. **Restart Required**: If `restart_required: true`, restart the application
2. **Verify Data**: Check `/api/v1/stats` endpoint for expected record counts
3. **Test Functionality**: Verify core features work correctly
4. **Monitor Logs**: Watch for any errors in application logs

---

## Disaster Recovery Playbook

### Scenario 1: Database Corruption

**Symptoms**: Application errors, query failures, data inconsistency

**Recovery Steps**:

1. **Stop the application** to prevent further corruption
   ```bash
   docker stop cartographus
   ```

2. **Identify the latest valid backup**
   ```bash
   # Check backup metadata
   ls -la /data/backups/
   cat /data/backups/metadata.json | jq '.backups | sort_by(.created_at) | reverse | .[0:5]'
   ```

3. **Validate the backup**
   ```bash
   # Manual validation
   tar -tzf /data/backups/backup-20260101-020000.tar.gz
   ```

4. **Restore from backup**
   ```bash
   # Start application in maintenance mode
   docker run -e MAINTENANCE_MODE=true cartographus

   # Trigger restore via API
   curl -X POST http://localhost:3857/api/v1/backup/restore/{backup_id} \
     -H "Authorization: Bearer $TOKEN" \
     -d '{"create_pre_restore_backup": false, "verify_after_restore": true}'
   ```

5. **Restart in normal mode**
   ```bash
   docker restart cartographus
   ```

6. **Verify recovery**
   ```bash
   curl http://localhost:3857/api/v1/health/ready
   curl http://localhost:3857/api/v1/stats
   ```

### Scenario 2: Complete System Loss

**Symptoms**: Server failure, disk loss, cloud instance termination

**Recovery Steps**:

1. **Provision new infrastructure**
   - Deploy new server/container
   - Ensure backup storage is accessible (or restore from off-site backup)

2. **Restore backup files**
   ```bash
   # From off-site backup
   aws s3 sync s3://my-backups/cartographus /data/backups/

   # Or from volume snapshot
   mount /dev/snapshot-volume /data/backups
   ```

3. **Deploy application with backup directory mounted**
   ```bash
   docker run -d \
     -v /data/backups:/data/backups \
     -v /data/db:/data/db \
     -e BACKUP_DIR=/data/backups \
     cartographus
   ```

4. **Restore from the latest backup**
   ```bash
   # The application will detect and can restore from existing backups
   curl -X POST http://localhost:3857/api/v1/backup/restore/latest \
     -H "Authorization: Bearer $TOKEN"
   ```

5. **Reconfigure and verify**
   - Update DNS if IP changed
   - Verify webhooks and integrations
   - Test end-to-end functionality

### Scenario 3: Accidental Data Deletion

**Symptoms**: Missing playback data, users, or configurations

**Recovery Steps**:

1. **Identify when deletion occurred**
   ```bash
   # Check application logs
   grep -i "delete" /var/log/cartographus/*.log
   ```

2. **Find backup before deletion**
   ```bash
   # List backups with timestamps
   curl "http://localhost:3857/api/v1/backup/list?sort_desc=true" \
     -H "Authorization: Bearer $TOKEN" | jq '.backups[] | {id, created_at, record_count}'
   ```

3. **Restore specific tables** (if possible) or full database
   ```bash
   # Full restore with safety backup
   curl -X POST http://localhost:3857/api/v1/backup/restore/{backup_id} \
     -H "Authorization: Bearer $TOKEN" \
     -d '{"create_pre_restore_backup": true}'
   ```

4. **Re-sync recent data** after restore
   ```bash
   curl -X POST http://localhost:3857/api/v1/sync \
     -H "Authorization: Bearer $TOKEN"
   ```

---

## Monitoring and Alerts

### Health Checks

```bash
# Check backup system status
curl http://localhost:3857/api/v1/backup/status

# Response
{
  "enabled": true,
  "scheduler_running": true,
  "last_backup": {
    "id": "20260101-020000-full",
    "status": "completed",
    "created_at": "2026-01-01T02:00:00Z"
  },
  "next_scheduled": "2026-01-02T02:00:00Z",
  "stats": {
    "total_count": 45,
    "total_size_bytes": 15728640,
    "success_rate": 0.98
  }
}
```

### Prometheus Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `cartographus_backup_total` | Counter | Total backups by type and status |
| `cartographus_backup_duration_seconds` | Histogram | Backup duration |
| `cartographus_backup_size_bytes` | Gauge | Latest backup size |
| `cartographus_backup_last_success_timestamp` | Gauge | Timestamp of last successful backup |

### Alert Rules (Prometheus)

```yaml
groups:
  - name: cartographus-backup
    rules:
      - alert: BackupFailed
        expr: increase(cartographus_backup_total{status="failed"}[1h]) > 0
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "Backup failed"

      - alert: BackupOverdue
        expr: time() - cartographus_backup_last_success_timestamp > 86400 * 2
        for: 1h
        labels:
          severity: warning
        annotations:
          summary: "No successful backup in 48 hours"

      - alert: BackupStorageLow
        expr: node_filesystem_avail_bytes{mountpoint="/data/backups"} < 1073741824
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "Less than 1GB free for backups"
```

---

## Best Practices

### Backup Strategy

1. **3-2-1 Rule**: Keep 3 copies, on 2 different media, with 1 off-site
2. **Test Restores**: Regularly test restore procedures (monthly recommended)
3. **Monitor Storage**: Set alerts for backup storage capacity
4. **Encrypt Sensitive Data**: Use `BACKUP_ENCRYPTION_ENABLED=true` for sensitive deployments

### Off-Site Backups

```bash
# Sync to cloud storage (example: AWS S3)
aws s3 sync /data/backups/ s3://my-bucket/cartographus-backups/ \
  --exclude "*.tmp" \
  --storage-class STANDARD_IA

# Sync to remote server
rsync -avz --progress /data/backups/ backup-server:/data/cartographus-backups/
```

### Docker Volume Backups

```bash
# Backup Docker volume
docker run --rm \
  -v cartographus_data:/data \
  -v $(pwd):/backup \
  alpine tar czf /backup/volume-backup.tar.gz /data

# Restore Docker volume
docker run --rm \
  -v cartographus_data:/data \
  -v $(pwd):/backup \
  alpine tar xzf /backup/volume-backup.tar.gz -C /
```

### Security Considerations

1. **Backup Directory Permissions**: Restrict to application user only
   ```bash
   chmod 750 /data/backups
   chown cartographus:cartographus /data/backups
   ```

2. **Encryption Key Management**: Store encryption keys separately from backups
   - Use a secrets manager (HashiCorp Vault, AWS Secrets Manager)
   - Never commit keys to version control

3. **Network Security**: Use encrypted connections for off-site backups
   - SSH/SFTP for server-to-server
   - HTTPS/TLS for cloud storage

4. **Access Control**: Limit who can trigger backups and restores
   - Require authentication for all backup endpoints
   - Log all backup/restore operations

---

## Troubleshooting

### Common Issues

| Issue | Cause | Solution |
|-------|-------|----------|
| Backup fails with "disk full" | Insufficient storage | Increase storage or reduce retention |
| Backup takes too long | Large database | Enable compression, schedule during low usage |
| Restore fails with checksum error | Corrupted backup | Try earlier backup, verify storage integrity |
| Scheduler not running | Configuration error | Check `BACKUP_SCHEDULE_ENABLED` and logs |

### Debug Commands

```bash
# Check backup directory permissions
ls -la /data/backups/

# Verify backup file integrity
sha256sum /data/backups/backup-*.tar.gz

# Check application logs
docker logs cartographus 2>&1 | grep -i backup

# Manually trigger a backup for testing
curl -X POST http://localhost:3857/api/v1/backup/create \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"type": "database", "notes": "Manual test backup"}'
```
