// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package backup

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// Config holds all backup-related configuration
type Config struct {
	// Enable backup functionality
	Enabled bool

	// Directory to store backups
	BackupDir string

	// Schedule configuration
	Schedule ScheduleConfig

	// Retention policy
	Retention RetentionPolicy

	// Compression settings
	Compression CompressionConfig

	// Encryption settings (optional)
	Encryption EncryptionConfig

	// Notification settings
	Notifications NotificationConfig
}

// CompressionConfig defines compression settings for backups
type CompressionConfig struct {
	// Enable compression
	Enabled bool

	// Compression level (1-9, where 9 is maximum compression)
	Level int

	// Compression algorithm (gzip, zstd)
	Algorithm string
}

// EncryptionConfig defines encryption settings for backups
type EncryptionConfig struct {
	// Enable encryption
	Enabled bool

	// Encryption key (AES-256)
	// This should be loaded from a secure source (env var, vault, etc.)
	Key string

	// Key ID for key rotation support
	KeyID string
}

// NotificationConfig defines notification settings for backup events
type NotificationConfig struct {
	// Notify on backup completion
	OnSuccess bool

	// Notify on backup failure
	OnFailure bool

	// Notify on retention cleanup
	OnCleanup bool

	// Webhook URL for notifications
	WebhookURL string
}

// LoadConfig loads backup configuration from environment variables
func LoadConfig() (*Config, error) {
	cfg := &Config{
		Enabled:   getBoolEnv("BACKUP_ENABLED", true),
		BackupDir: getEnv("BACKUP_DIR", "/data/backups"),

		Schedule: ScheduleConfig{
			Enabled:       getBoolEnv("BACKUP_SCHEDULE_ENABLED", true),
			Interval:      getDurationEnv("BACKUP_INTERVAL", 24*time.Hour),
			PreferredHour: getIntEnv("BACKUP_PREFERRED_HOUR", 2),
			BackupType:    BackupType(getEnv("BACKUP_TYPE", string(TypeFull))),
			PreSyncBackup: getBoolEnv("BACKUP_PRE_SYNC", false),
		},

		Retention: RetentionPolicy{
			MinCount:             getIntEnv("BACKUP_RETENTION_MIN_COUNT", 3),
			MaxCount:             getIntEnv("BACKUP_RETENTION_MAX_COUNT", 50),
			MaxAgeDays:           getIntEnv("BACKUP_RETENTION_MAX_DAYS", 90),
			KeepRecentHours:      getIntEnv("BACKUP_RETENTION_KEEP_RECENT_HOURS", 24),
			KeepDailyForDays:     getIntEnv("BACKUP_RETENTION_KEEP_DAILY_DAYS", 7),
			KeepWeeklyForWeeks:   getIntEnv("BACKUP_RETENTION_KEEP_WEEKLY_WEEKS", 4),
			KeepMonthlyForMonths: getIntEnv("BACKUP_RETENTION_KEEP_MONTHLY_MONTHS", 6),
		},

		Compression: CompressionConfig{
			Enabled:   getBoolEnv("BACKUP_COMPRESSION_ENABLED", true),
			Level:     getIntEnv("BACKUP_COMPRESSION_LEVEL", 6),
			Algorithm: getEnv("BACKUP_COMPRESSION_ALGORITHM", "gzip"),
		},

		Encryption: EncryptionConfig{
			Enabled: getBoolEnv("BACKUP_ENCRYPTION_ENABLED", false),
			Key:     getEnv("BACKUP_ENCRYPTION_KEY", ""),
			KeyID:   getEnv("BACKUP_ENCRYPTION_KEY_ID", ""),
		},

		Notifications: NotificationConfig{
			OnSuccess:  getBoolEnv("BACKUP_NOTIFY_SUCCESS", false),
			OnFailure:  getBoolEnv("BACKUP_NOTIFY_FAILURE", true),
			OnCleanup:  getBoolEnv("BACKUP_NOTIFY_CLEANUP", false),
			WebhookURL: getEnv("BACKUP_WEBHOOK_URL", ""),
		},
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("backup configuration validation failed: %w", err)
	}

	return cfg, nil
}

// Validate checks that the configuration is valid
//
//nolint:gocyclo // Validation function with many sequential checks
func (c *Config) Validate() error {
	if !c.Enabled {
		return nil // No validation needed if backups are disabled
	}

	// Validate backup directory
	if c.BackupDir == "" {
		return fmt.Errorf("BACKUP_DIR is required when backups are enabled")
	}

	// Ensure backup directory is absolute path
	if !filepath.IsAbs(c.BackupDir) {
		return fmt.Errorf("BACKUP_DIR must be an absolute path, got: %s", c.BackupDir)
	}

	// Validate schedule
	if c.Schedule.Enabled {
		if c.Schedule.Interval < time.Hour {
			return fmt.Errorf("BACKUP_INTERVAL must be at least 1 hour, got: %s", c.Schedule.Interval)
		}
		if c.Schedule.PreferredHour < 0 || c.Schedule.PreferredHour > 23 {
			return fmt.Errorf("BACKUP_PREFERRED_HOUR must be between 0 and 23, got: %d", c.Schedule.PreferredHour)
		}
		if c.Schedule.BackupType != TypeFull && c.Schedule.BackupType != TypeDatabase && c.Schedule.BackupType != TypeConfig {
			return fmt.Errorf("BACKUP_TYPE must be one of: full, database, config")
		}
	}

	// Validate retention policy
	if c.Retention.MinCount < 1 {
		return fmt.Errorf("BACKUP_RETENTION_MIN_COUNT must be at least 1")
	}
	if c.Retention.MaxCount > 0 && c.Retention.MaxCount < c.Retention.MinCount {
		return fmt.Errorf("BACKUP_RETENTION_MAX_COUNT (%d) must be >= BACKUP_RETENTION_MIN_COUNT (%d)",
			c.Retention.MaxCount, c.Retention.MinCount)
	}

	// Validate compression
	if c.Compression.Enabled {
		if c.Compression.Level < 1 || c.Compression.Level > 9 {
			return fmt.Errorf("BACKUP_COMPRESSION_LEVEL must be between 1 and 9, got: %d", c.Compression.Level)
		}
		if c.Compression.Algorithm != "gzip" && c.Compression.Algorithm != "zstd" {
			return fmt.Errorf("BACKUP_COMPRESSION_ALGORITHM must be one of: gzip, zstd")
		}
	}

	// Validate encryption
	if c.Encryption.Enabled {
		if len(c.Encryption.Key) < 32 {
			return fmt.Errorf("BACKUP_ENCRYPTION_KEY must be at least 32 characters for AES-256")
		}
	}

	return nil
}

// EnsureBackupDir creates the backup directory if it doesn't exist
func (c *Config) EnsureBackupDir() error {
	if err := os.MkdirAll(c.BackupDir, 0o750); err != nil {
		return fmt.Errorf("failed to create backup directory %s: %w", c.BackupDir, err)
	}
	return nil
}

// Helper functions to read environment variables

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

func getBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return defaultValue
}
