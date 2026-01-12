# Database Migration Guide

This guide covers database migration procedures for Cartographus, including version upgrades, DuckDB schema changes, NATS JetStream migrations, and data recovery procedures.

**Related Documentation**:
- [PRODUCTION_DEPLOYMENT.md](./PRODUCTION_DEPLOYMENT.md) - Production deployment guide
- [TROUBLESHOOTING.md](./TROUBLESHOOTING.md) - Common issues and solutions
- [ADR-0001](./adr/0001-use-duckdb-for-analytics.md) - DuckDB architecture decision

---

## Table of Contents

1. [Pre-Migration Checklist](#pre-migration-checklist)
2. [Version Upgrade Procedures](#version-upgrade-procedures)
3. [DuckDB Migrations](#duckdb-migrations)
4. [NATS JetStream Migration](#nats-jetstream-migration)
5. [Data Import and Export](#data-import-and-export)
6. [Rollback Procedures](#rollback-procedures)
7. [Troubleshooting](#troubleshooting)

---

## Pre-Migration Checklist

Complete these steps before any migration:

### Critical Checks

- [ ] **Create backup** before any migration
  ```bash
  # API backup
  curl -X POST http://localhost:3857/api/v1/backup/create?type=full

  # Or stop the service and copy files
  systemctl stop cartographus
  cp -r /data /data.backup.$(date +%Y%m%d)
  ```

- [ ] **Verify backup integrity**
  ```bash
  # Check backup was created
  curl http://localhost:3857/api/v1/backup/list | jq '.[-1]'

  # Verify backup can be read
  tar -tzf /data/backups/backup-*.tar.gz | head
  ```

- [ ] **Check current version**
  ```bash
  curl http://localhost:3857/api/v1/version | jq
  # Or: ./cartographus --version
  ```

- [ ] **Review changelog** for breaking changes
  - Check [CHANGELOG.md](../CHANGELOG.md) for migration notes
  - Look for "BREAKING" or "Migration Required" sections

- [ ] **Plan maintenance window**
  - Notify users if applicable
  - Schedule during low-activity period

- [ ] **Test migration in staging** (if available)
  - Restore backup to staging environment
  - Perform upgrade and verify functionality

---

## Version Upgrade Procedures

### Minor Version Upgrade (e.g., v1.50.0 to v1.50.1)

Minor versions typically require no migration steps.

```bash
# 1. Pull new image
docker pull ghcr.io/tomtom215/cartographus:v1.50.1

# 2. Update and restart
docker-compose up -d

# 3. Verify health
curl http://localhost:3857/api/v1/health
```

### Major Version Upgrade (e.g., v1.x to v2.x)

Major versions may include breaking changes and schema migrations.

```bash
# 1. Stop the service
docker-compose down

# 2. Create full backup
cp -r /var/lib/cartographus /var/lib/cartographus.backup.v1

# 3. Review migration notes in CHANGELOG.md

# 4. Update image tag in docker-compose.yml
# image: ghcr.io/tomtom215/cartographus:v2.0.0

# 5. Start with new version (migrations run automatically)
docker-compose up -d

# 6. Check logs for migration status
docker logs cartographus 2>&1 | grep -i migration

# 7. Verify health and functionality
curl http://localhost:3857/api/v1/health
curl http://localhost:3857/api/v1/stats/summary
```

### Binary Upgrade

```bash
# 1. Stop the service
sudo systemctl stop cartographus

# 2. Backup current binary
sudo mv /opt/cartographus/cartographus-linux-amd64 \
        /opt/cartographus/cartographus-linux-amd64.old

# 3. Download new binary
sudo curl -LO https://github.com/tomtom215/cartographus/releases/download/v1.51.0/cartographus-linux-amd64

# 4. Make executable
sudo chmod +x cartographus-linux-amd64

# 5. Start service
sudo systemctl start cartographus

# 6. Verify
sudo systemctl status cartographus
curl http://localhost:3857/api/v1/health
```

---

## DuckDB Migrations

Cartographus uses DuckDB as its primary analytics database. Schema migrations are handled automatically on startup.

### Automatic Migration

When Cartographus starts, it:
1. Checks the current schema version in `schema_migrations` table
2. Applies any pending migrations in order
3. Updates the schema version

You can monitor migration progress in logs:
```bash
docker logs cartographus 2>&1 | grep -E "(migration|schema)"
```

### Manual Schema Inspection

Connect to DuckDB directly for inspection (read-only while service is running):

```bash
# Stop service first for write access
systemctl stop cartographus

# Connect with DuckDB CLI
duckdb /data/cartographus.duckdb

# Check schema version
SELECT * FROM schema_migrations ORDER BY version DESC LIMIT 5;

# List tables
SHOW TABLES;

# Check table schema
DESCRIBE playback_events;

# Exit
.exit
```

### DuckDB-Specific Considerations

DuckDB differs from SQLite in several ways that affect migrations:

| Feature | DuckDB | SQLite |
|---------|--------|--------|
| Concurrent Writes | Single writer | Single writer |
| ALTER TABLE ADD COLUMN | Supported | Supported |
| ALTER TABLE DROP COLUMN | Supported | Limited |
| Partial Indexes | NOT Supported | Supported |
| IDENTITY | NOT with PRIMARY KEY | Supported |

**Migration Patterns for DuckDB:**

```sql
-- Adding a column (safe)
ALTER TABLE playback_events ADD COLUMN new_field VARCHAR;

-- Adding a column with default (safe)
ALTER TABLE playback_events ADD COLUMN status VARCHAR DEFAULT 'active';

-- Renaming a column (DuckDB supports this)
ALTER TABLE playback_events RENAME COLUMN old_name TO new_name;

-- Creating an index (without WHERE clause - DuckDB doesn't support partial indexes)
CREATE INDEX idx_user_date ON playback_events(user_id, started_at);

-- Auto-increment pattern (use COALESCE instead of IDENTITY)
INSERT INTO my_table (id, name)
SELECT COALESCE(MAX(id), 0) + 1, 'new_name' FROM my_table;
```

### Handling Failed Migrations

If a migration fails:

1. **Check logs for error details**
   ```bash
   docker logs cartographus 2>&1 | grep -A 10 "migration failed"
   ```

2. **Restore from backup**
   ```bash
   systemctl stop cartographus
   rm -rf /data/cartographus.duckdb*
   tar -xzf /data/backups/backup-latest.tar.gz -C /data
   systemctl start cartographus
   ```

3. **Report the issue** if it's a bug in the migration

---

## NATS JetStream Migration

### Embedded to External NATS

If you're moving from embedded NATS to an external cluster:

**1. Export Current State (Optional)**

The embedded NATS JetStream data is stored in `NATS_STORE_DIR` (default: `/data/nats/jetstream`).

```bash
# Check current stream status
nats stream info MEDIA_EVENTS --server=nats://localhost:4222
```

**2. Configure External NATS**

Update environment variables:

```bash
# Before (embedded)
NATS_EMBEDDED=true
NATS_URL=nats://127.0.0.1:4222

# After (external cluster)
NATS_EMBEDDED=false
NATS_URL=nats://nats-cluster.example.com:4222
# Or for multiple nodes:
# NATS_URL=nats://nats1:4222,nats://nats2:4222,nats://nats3:4222
```

**3. Restart with External NATS**

```bash
docker-compose down
# Update docker-compose.yml with new NATS settings
docker-compose up -d
```

**4. Verify Connection**

```bash
# Check logs for NATS connection
docker logs cartographus 2>&1 | grep -i nats

# Expected output:
# Connected to NATS server at nats://nats-cluster.example.com:4222
```

### External NATS Cluster Requirements

Your external NATS cluster must have JetStream enabled:

```hcl
# nats-server.conf
jetstream {
    store_dir: /data/jetstream
    max_memory_store: 1GB
    max_file_store: 10GB
}
```

### Stream and Consumer Recreation

Cartographus automatically creates the required stream and consumers on startup. If you need to manually recreate:

```bash
# Create stream (Cartographus creates this automatically)
nats stream add MEDIA_EVENTS \
    --subjects="playback.events.>" \
    --retention=workqueue \
    --storage=file \
    --max-age=7d \
    --discard=old \
    --server=nats://your-cluster:4222

# Create consumer (Cartographus creates this automatically)
nats consumer add MEDIA_EVENTS media-processor \
    --ack=explicit \
    --max-deliver=5 \
    --wait=30s \
    --server=nats://your-cluster:4222
```

---

## Data Import and Export

### Tautulli Database Import

Import historical data from an existing Tautulli installation:

```bash
# 1. Copy Tautulli database (usually ~/.local/share/Tautulli/tautulli.db)
cp /path/to/tautulli.db /data/import/tautulli.db

# 2. Configure import
export IMPORT_ENABLED=true
export IMPORT_DB_PATH=/data/import/tautulli.db
export IMPORT_BATCH_SIZE=1000
export IMPORT_DRY_RUN=true  # Test first

# 3. Restart to run import
docker-compose restart

# 4. Check import progress
curl http://localhost:3857/api/v1/import/status

# 5. After dry run succeeds, run actual import
export IMPORT_DRY_RUN=false
docker-compose restart
```

### Export Data

Export data for backup or analysis:

```bash
# Export via API
curl -H "Authorization: Bearer $TOKEN" \
    "http://localhost:3857/api/v1/export/playbacks?format=csv" \
    -o playbacks.csv

# Export as GeoJSON (for geographic analysis)
curl -H "Authorization: Bearer $TOKEN" \
    "http://localhost:3857/api/v1/export/playbacks?format=geojson" \
    -o playbacks.geojson

# Export as GeoParquet (for big data tools)
curl -H "Authorization: Bearer $TOKEN" \
    "http://localhost:3857/api/v1/export/playbacks?format=geoparquet" \
    -o playbacks.parquet
```

### Direct DuckDB Export

For advanced exports, use DuckDB directly:

```bash
# Stop service for exclusive access
systemctl stop cartographus

# Connect and export
duckdb /data/cartographus.duckdb << 'EOF'
COPY (
    SELECT * FROM playback_events
    WHERE started_at > '2024-01-01'
) TO '/data/export/playbacks_2024.parquet' (FORMAT PARQUET);
EOF

# Restart service
systemctl start cartographus
```

---

## Rollback Procedures

### Quick Rollback (Docker)

```bash
# 1. Stop current version
docker-compose down

# 2. Update image to previous version
# Edit docker-compose.yml: image: ghcr.io/tomtom215/cartographus:v1.49.0

# 3. Restore database backup if needed
rm -rf /data/cartographus.duckdb*
tar -xzf /data/backups/pre-upgrade-backup.tar.gz -C /data

# 4. Start previous version
docker-compose up -d

# 5. Verify
curl http://localhost:3857/api/v1/health
```

### Full Rollback with Data

If you need to rollback both application and data:

```bash
# 1. Stop the service
docker-compose down

# 2. Restore full backup
rm -rf /data/*
tar -xzf /var/lib/cartographus.backup.v1/backup-full.tar.gz -C /data

# 3. Restore previous application version
# Edit docker-compose.yml with old image tag

# 4. Restore previous configuration
cp /etc/cartographus/env.backup /etc/cartographus/env

# 5. Start
docker-compose up -d

# 6. Verify all data is present
curl http://localhost:3857/api/v1/stats/summary
```

### Point-in-Time Recovery

If you need to recover to a specific point in time:

```bash
# 1. List available backups
curl http://localhost:3857/api/v1/backup/list | jq

# 2. Find the backup closest to your target time

# 3. Stop service
docker-compose down

# 4. Restore specific backup
rm -rf /data/cartographus.duckdb*
tar -xzf /data/backups/backup-20240115-020000.tar.gz -C /data

# 5. Start service
docker-compose up -d
```

---

## Troubleshooting

### Migration Fails on Startup

**Symptom:** Service fails to start with migration error.

**Solution:**
1. Check logs for specific error
2. Restore from backup
3. Report issue if it's a bug

```bash
docker logs cartographus 2>&1 | grep -i "error\|failed\|migration"
```

### Database Locked Error

**Symptom:** "database is locked" or "could not open database"

**Solution:** DuckDB supports only one writer. Ensure no other process is accessing the database.

```bash
# Find processes using the database file
fuser -v /data/cartographus.duckdb

# Kill if necessary (after verifying it's safe)
fuser -k /data/cartographus.duckdb
```

### NATS Connection Failed

**Symptom:** Service starts but can't connect to NATS.

**Solution:**

```bash
# Check if NATS is running (embedded mode)
docker logs cartographus 2>&1 | grep -i nats

# Verify NATS URL is correct
echo $NATS_URL

# Test connection (if external)
nats server check connection --server=$NATS_URL
```

### Schema Version Mismatch

**Symptom:** "schema version mismatch" or "unknown schema version"

**Solution:** This typically means you're running an older application version with a newer database schema.

```bash
# Check schema version
duckdb /data/cartographus.duckdb -c "SELECT * FROM schema_migrations ORDER BY version DESC LIMIT 1;"

# Upgrade to matching application version
docker pull ghcr.io/tomtom215/cartographus:v1.51.0  # Match the schema version
```

### Disk Space Issues During Migration

**Symptom:** Migration fails due to disk space.

**Solution:**

```bash
# Check disk usage
df -h /data

# Clean up old backups
find /data/backups -name "*.tar.gz" -mtime +30 -delete

# Compact DuckDB (run maintenance)
curl -X POST http://localhost:3857/api/v1/admin/compact
```

---

## Version Compatibility Matrix

| Application Version | DuckDB Version | Schema Version | NATS Protocol |
|---------------------|----------------|----------------|---------------|
| v1.48.x | 1.4.3 | 20 | 2.9 |
| v1.49.x | 1.4.3 | 21 | 2.9 |
| v1.50.x | 1.4.3 | 22 | 2.10 |
| v1.51.x | 1.4.3 | 23 | 2.10 |

**Notes:**
- Always upgrade sequentially (don't skip major versions)
- DuckDB version changes require special handling (see DuckDB docs)
- NATS protocol is backward compatible

---

## Best Practices

### Before Any Migration

1. **Always backup** - No exceptions
2. **Test in staging** - If possible
3. **Read the changelog** - Look for breaking changes
4. **Plan maintenance window** - Avoid peak usage times

### During Migration

1. **Monitor logs** - Watch for errors
2. **Verify health** - Check endpoints after startup
3. **Test functionality** - Ensure core features work

### After Migration

1. **Verify data integrity** - Check counts and recent data
2. **Monitor performance** - Watch for regressions
3. **Keep backups** - Don't delete pre-migration backups immediately
4. **Document changes** - Note any issues for future reference

---

## Support

If you encounter migration issues:

1. Check [TROUBLESHOOTING.md](./TROUBLESHOOTING.md) for common issues
2. Search [GitHub Issues](https://github.com/tomtom215/cartographus/issues)
3. Open a new issue with:
   - Current version
   - Target version
   - Error messages
   - Relevant log output
