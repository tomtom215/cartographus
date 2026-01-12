# ADR-0029: Backup and Restore with GFS Retention Strategy

**Date**: 2026-01-11
**Status**: Accepted
**Last Verified**: 2026-01-11

---

## Context

Cartographus stores valuable playback analytics data that requires protection against:
- Accidental deletion or corruption
- Failed upgrades or schema migrations
- Hardware failures
- Ransomware or security incidents

### Requirements

1. **Data Protection**: Regular automated backups with integrity verification
2. **Flexible Retention**: Keep appropriate backup history without unlimited growth
3. **Compression**: Reduce storage footprint for large databases
4. **Optional Encryption**: Protect sensitive data at rest
5. **Point-in-Time Recovery**: Restore to any backup point
6. **Pre-Operation Snapshots**: Automatic backups before risky operations

### Alternatives Considered

| Approach | Pros | Cons |
|----------|------|------|
| Simple Count-Based | Easy to implement | No time-based recovery |
| Age-Only Retention | Predictable cleanup | May keep too many recent |
| GFS (Grandfather-Father-Son) | Balanced coverage | More complex rules |
| Continuous Replication | Minimal data loss | High resource usage |

---

## Decision

Implement a **Grandfather-Father-Son (GFS) retention strategy** with:

1. **Backup Types**: Full, Database-only, Config-only
2. **Compression**: GZIP (default) or ZSTD with configurable levels
3. **Encryption**: Optional AES-256 encryption at rest
4. **GFS Retention**: Keep hourly/daily/weekly/monthly backups
5. **Integrity Verification**: SHA-256 checksums for all backups
6. **Scheduler Integration**: Automatic backups with preferred hour

### GFS Strategy

The GFS strategy provides optimal coverage:

```
Time Axis ──────────────────────────────────────────────────▶

│ Keep all in last 24 hours │ Keep 1/day for 7 days │
│◀─────────────────────────▶│◀──────────────────────▶│

│ Keep 1/week for 4 weeks │ Keep 1/month for 6 months │
│◀───────────────────────▶│◀─────────────────────────▶│

Plus: Always keep at least MinCount (3) backups
      Never exceed MaxCount (50) backups
```

---

## Consequences

### Positive

- **Balanced Coverage**: Recent high-frequency, historical low-frequency
- **Predictable Growth**: MaxCount cap prevents unlimited storage use
- **Fast Recovery**: Recent backups readily available
- **Audit Trail**: Complete backup history with metadata

### Negative

- **Complexity**: GFS rules more complex than simple retention
- **Storage Cost**: Keeping monthly backups uses more space than age-only
- **Cleanup Delay**: Backups not immediately deleted when policy changes

### Neutral

- **JSON Metadata**: All backup info stored in `metadata.json`
- **Tar.gz Archives**: Standard archive format for portability

---

## Implementation

### Backup Types

```go
// Location: internal/backup/types.go:51-65
const (
    TypeFull        BackupType = "full"        // Database + config
    TypeDatabase    BackupType = "database"    // DuckDB files only
    TypeConfig      BackupType = "config"      // Config only (sanitized)
    TypeIncremental BackupType = "incremental" // Future enhancement
)
```

### Backup Status Lifecycle

```go
// Location: internal/backup/types.go:70-85
const (
    StatusPending    BackupStatus = "pending"     // Queued
    StatusInProgress BackupStatus = "in_progress" // Running
    StatusCompleted  BackupStatus = "completed"   // Success
    StatusFailed     BackupStatus = "failed"      // Error
    StatusCorrupted  BackupStatus = "corrupted"   // Checksum mismatch
)
```

### Retention Policy Structure

```go
// Location: internal/backup/types.go:309-331
type RetentionPolicy struct {
    MinCount             int `json:"min_count"`              // Always keep (default: 3)
    MaxCount             int `json:"max_count"`              // Maximum total (default: 50)
    MaxAgeDays           int `json:"max_age_days"`           // Delete older (default: 90)
    KeepRecentHours      int `json:"keep_recent_hours"`      // Keep all in window (default: 24)
    KeepDailyForDays     int `json:"keep_daily_for_days"`    // Daily retention (default: 7)
    KeepWeeklyForWeeks   int `json:"keep_weekly_for_weeks"`  // Weekly retention (default: 4)
    KeepMonthlyForMonths int `json:"keep_monthly_for_months"`// Monthly retention (default: 6)
}
```

### Default Retention Policy

```go
// Location: internal/backup/types.go:334-344
func DefaultRetentionPolicy() RetentionPolicy {
    return RetentionPolicy{
        MinCount:             3,  // Always keep at least 3 backups
        MaxCount:             50, // Maximum 50 backups
        MaxAgeDays:           90, // Delete backups older than 90 days
        KeepRecentHours:      24, // Keep all backups from last 24 hours
        KeepDailyForDays:     7,  // Keep daily backups for 7 days
        KeepWeeklyForWeeks:   4,  // Keep weekly backups for 4 weeks
        KeepMonthlyForMonths: 6,  // Keep monthly backups for 6 months
    }
}
```

### Retention Rule Application

```go
// Location: internal/backup/retention.go:54-84
func (m *Manager) applyRetentionRules(backups []*Backup, policy RetentionPolicy, now time.Time) map[string]bool {
    keepSet := make(map[string]bool)

    // Rule 1: Always keep MinCount backups
    addMinCountToKeepSet(keepSet, backups, policy.MinCount)

    // Rule 2: Keep all backups from the last KeepRecentHours
    if policy.KeepRecentHours > 0 {
        addRecentBackupsToKeepSet(keepSet, backups, policy.KeepRecentHours, now)
    }

    // Rule 3: Keep at least one backup per day for KeepDailyForDays
    if policy.KeepDailyForDays > 0 {
        dailyBackups := m.selectDailyBackups(backups, policy.KeepDailyForDays, now)
        addBackupsToKeepSet(keepSet, dailyBackups)
    }

    // Rule 4: Keep at least one backup per week for KeepWeeklyForWeeks
    if policy.KeepWeeklyForWeeks > 0 {
        weeklyBackups := m.selectWeeklyBackups(backups, policy.KeepWeeklyForWeeks, now)
        addBackupsToKeepSet(keepSet, weeklyBackups)
    }

    // Rule 5: Keep at least one backup per month for KeepMonthlyForMonths
    if policy.KeepMonthlyForMonths > 0 {
        monthlyBackups := m.selectMonthlyBackups(backups, policy.KeepMonthlyForMonths, now)
        addBackupsToKeepSet(keepSet, monthlyBackups)
    }

    return keepSet
}
```

### Backup Metadata

```go
// Location: internal/backup/types.go:108-162
type Backup struct {
    ID          string        `json:"id"`
    Type        BackupType    `json:"type"`
    Status      BackupStatus  `json:"status"`
    Trigger     BackupTrigger `json:"trigger"`
    CreatedAt   time.Time     `json:"created_at"`
    CompletedAt *time.Time    `json:"completed_at,omitempty"`
    Duration    time.Duration `json:"duration_ms"`
    FilePath    string        `json:"file_path"`
    FileSize    int64         `json:"file_size"`
    Checksum    string        `json:"checksum"`   // SHA-256
    Compressed  bool          `json:"compressed"`
    Encrypted   bool          `json:"encrypted"`
    AppVersion  string        `json:"app_version"`
    DBVersion   string        `json:"db_version"`
    RecordCount int64         `json:"record_count"`
    Notes       string        `json:"notes,omitempty"`
    Error       string        `json:"error,omitempty"`
    Contents    BackupContents `json:"contents"`
}
```

### Configuration

```go
// Location: internal/backup/config.go:12-33
type Config struct {
    Enabled       bool
    BackupDir     string
    Schedule      ScheduleConfig
    Retention     RetentionPolicy
    Compression   CompressionConfig   // gzip or zstd
    Encryption    EncryptionConfig    // AES-256
    Notifications NotificationConfig
}
```

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `BACKUP_ENABLED` | Enable backup system | `true` |
| `BACKUP_DIR` | Backup storage path | `/data/backups` |
| `BACKUP_SCHEDULE_ENABLED` | Enable automatic backups | `true` |
| `BACKUP_INTERVAL` | Backup frequency | `24h` |
| `BACKUP_PREFERRED_HOUR` | Time for scheduled backups (0-23) | `2` |
| `BACKUP_TYPE` | Default backup type | `full` |
| `BACKUP_PRE_SYNC` | Backup before sync operations | `false` |
| `BACKUP_RETENTION_MIN_COUNT` | Minimum backups to keep | `3` |
| `BACKUP_RETENTION_MAX_COUNT` | Maximum backups to keep | `50` |
| `BACKUP_RETENTION_MAX_DAYS` | Maximum backup age | `90` |
| `BACKUP_RETENTION_KEEP_RECENT_HOURS` | Keep all in window | `24` |
| `BACKUP_RETENTION_KEEP_DAILY_DAYS` | Daily backup retention | `7` |
| `BACKUP_RETENTION_KEEP_WEEKLY_WEEKS` | Weekly backup retention | `4` |
| `BACKUP_RETENTION_KEEP_MONTHLY_MONTHS` | Monthly backup retention | `6` |
| `BACKUP_COMPRESSION_ENABLED` | Enable compression | `true` |
| `BACKUP_COMPRESSION_LEVEL` | Compression level (1-9) | `6` |
| `BACKUP_COMPRESSION_ALGORITHM` | `gzip` or `zstd` | `gzip` |
| `BACKUP_ENCRYPTION_ENABLED` | Enable AES-256 encryption | `false` |
| `BACKUP_ENCRYPTION_KEY` | Encryption key (32+ chars) | - |

---

## Code References

| Component | File | Notes |
|-----------|------|-------|
| Manager | `internal/backup/manager.go` | Core backup orchestration |
| Config | `internal/backup/config.go` | Configuration loading |
| Types | `internal/backup/types.go` | Data structures (461 lines) |
| Retention | `internal/backup/retention.go` | GFS policy implementation (602 lines) |
| Archive | `internal/backup/manager_archive.go` | Tar.gz creation |
| Restore | `internal/backup/restore.go` | Point-in-time recovery (836 lines) |
| Scheduler | `internal/backup/manager_scheduler.go` | Automatic backups |
| Package Doc | `internal/backup/doc.go` | Architecture documentation |

### Total Lines of Code

- **Source**: ~3,200 lines
- **Tests**: ~4,000 lines
- **Combined**: 7,271 lines

---

## Verification

### Verified Claims

| Claim | Source | Verified |
|-------|--------|----------|
| GFS retention rules | `internal/backup/retention.go:54-84` | Yes |
| Daily/Weekly/Monthly selection | `internal/backup/retention.go:279-304` | Yes |
| SHA-256 checksum | `internal/backup/types.go:137` | Yes |
| AES-256 encryption | `internal/backup/config.go:53` | Yes |
| GZIP/ZSTD compression | `internal/backup/config.go:42-44` | Yes |
| Default 3 min backups | `internal/backup/types.go:337` | Yes |
| Default 50 max backups | `internal/backup/types.go:338` | Yes |
| Default 90 day max age | `internal/backup/types.go:339` | Yes |
| Pre-sync backup trigger | `internal/backup/types.go:98` | Yes |
| Preferred hour scheduling | `internal/backup/config.go:84` | Yes |

### Test Coverage

- `internal/backup/backup_test.go` - 833 lines
- `internal/backup/retention_edge_cases_test.go` - 574 lines
- `internal/backup/restore_edge_cases_test.go` - 852 lines
- `internal/backup/manager_edge_cases_test.go` - 623 lines
- `internal/backup/config_edge_cases_test.go` - 316 lines
- `internal/backup/helpers_edge_cases_test.go` - 189 lines
- `internal/backup/benchmark_test.go` - 103 lines

---

## Related ADRs

- [ADR-0001](0001-use-duckdb-for-analytics.md): DuckDB database being backed up
- [ADR-0006](0006-badgerdb-write-ahead-log.md): WAL files included in backups
- [ADR-0012](0012-configuration-management-koanf.md): Config files being backed up

---

## References

- [GFS Retention Strategy](https://en.wikipedia.org/wiki/Grandfather-Father-Son_backup): Backup rotation scheme
- Internal: `internal/backup/doc.go` - Package documentation
