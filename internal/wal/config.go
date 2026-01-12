// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package wal provides a durable Write-Ahead Log (WAL) using BadgerDB for event persistence.
// Events are written to the WAL before being published to NATS, ensuring no event loss
// in case of NATS failures, process crashes, or power loss.
package wal

import (
	"os"
	"strconv"
	"time"
)

// Config holds WAL configuration loaded from environment variables.
//
// The WAL provides durability guarantees for events before NATS publishing.
// When enabled, events are first persisted to BadgerDB (ACID, fsync) and
// only removed after successful NATS acknowledgment.
//
// Environment Variables:
//   - WAL_ENABLED: Enable WAL durability (default: true)
//   - WAL_PATH: Directory for BadgerDB storage (default: /data/wal)
//   - WAL_SYNC_WRITES: Force fsync on every write (default: true for durability)
//   - WAL_RETRY_INTERVAL: Interval between retry attempts (default: 30s)
//   - WAL_MAX_RETRIES: Maximum retry attempts before giving up (default: 100)
//   - WAL_RETRY_BACKOFF: Initial backoff duration for retries (default: 5s)
//   - WAL_COMPACT_INTERVAL: Interval between compaction runs (default: 1h)
//   - WAL_ENTRY_TTL: Time-to-live for unconfirmed entries (default: 168h/7days)
//   - WAL_MEMTABLE_SIZE: BadgerDB memtable size in bytes (default: 16MB)
//   - WAL_VLOG_SIZE: BadgerDB value log file size (default: 64MB)
//   - WAL_NUM_COMPACTORS: Number of BadgerDB compaction workers (default: 2)
//
// Example - Enable WAL with defaults:
//
//	export WAL_ENABLED=true
//	export WAL_PATH=/data/wal
//
// Example - Disable WAL (events may be lost if NATS fails):
//
//	export WAL_ENABLED=false
type Config struct {
	// Enabled controls whether the WAL is active.
	// When disabled, events are published directly to NATS without durability.
	Enabled bool

	// Path is the directory where BadgerDB stores its files.
	// Should be on a durable filesystem (not tmpfs).
	Path string

	// SyncWrites forces fsync after every write for maximum durability.
	// Set to false for higher throughput but risk of data loss on power failure.
	SyncWrites bool

	// RetryInterval is the time between retry loop iterations.
	RetryInterval time.Duration

	// MaxRetries is the maximum number of retry attempts for a failed entry.
	// After this many retries, the entry is logged as permanently failed.
	MaxRetries int

	// RetryBackoff is the initial backoff duration for exponential backoff.
	RetryBackoff time.Duration

	// CompactInterval is the time between compaction runs.
	// Compaction removes confirmed entries to free disk space.
	CompactInterval time.Duration

	// EntryTTL is the time-to-live for unconfirmed entries.
	// Entries older than this are cleaned up regardless of confirmation status.
	EntryTTL time.Duration

	// BadgerDB tuning options
	// MemTableSize is the size of each memtable in bytes.
	MemTableSize int64

	// ValueLogFileSize is the size of each value log file in bytes.
	ValueLogFileSize int64

	// NumCompactors is the number of compaction workers.
	NumCompactors int

	// Compression enables Snappy compression for WAL entries.
	// Reduces disk usage by 40-60% for JSON payloads with slight CPU overhead.
	// Default: true
	Compression bool

	// GCRatio is the ratio for value log garbage collection.
	// Lower values reclaim more space but use more CPU.
	// Default: 0.5
	GCRatio float64

	// CloseTimeout is the maximum time to wait for graceful shutdown.
	// If the database doesn't close within this time, Close() returns with an error.
	// Default: 30s
	CloseTimeout time.Duration

	// NumMemtables is the number of memtables to keep in memory.
	// Higher values use more memory but improve write performance.
	// Default: 5 (BadgerDB default)
	NumMemtables int

	// BlockCacheSize is the size of the block cache in bytes.
	// Default: 256MB
	BlockCacheSize int64

	// IndexCacheSize is the size of the index cache in bytes.
	// Default: 0 (disabled, uses block cache)
	IndexCacheSize int64

	// LeaseDuration is how long a processing lease is held before expiring.
	// This enables durable leasing to prevent concurrent processing of the same entry.
	// When an entry is claimed, a lease expiry is set to now + LeaseDuration.
	// If the process crashes, the lease will naturally expire, allowing recovery.
	// Default: 2 minutes (should be longer than expected processing time)
	LeaseDuration time.Duration
}

// DefaultConfig returns a Config with sensible defaults for home lab deployments.
// These defaults prioritize durability over performance.
func DefaultConfig() Config {
	return Config{
		Enabled:          true,
		Path:             "/data/wal",
		SyncWrites:       true,
		RetryInterval:    30 * time.Second,
		MaxRetries:       100,
		RetryBackoff:     5 * time.Second,
		CompactInterval:  1 * time.Hour,
		EntryTTL:         168 * time.Hour, // 7 days
		MemTableSize:     16 * 1024 * 1024,
		ValueLogFileSize: 64 * 1024 * 1024,
		NumCompactors:    2,
		Compression:      true,
		GCRatio:          0.5,
		CloseTimeout:     30 * time.Second,
		NumMemtables:     5,
		BlockCacheSize:   256 * 1024 * 1024, // 256MB
		IndexCacheSize:   0,                 // Disabled, uses block cache
		LeaseDuration:    2 * time.Minute,   // Durable lease for concurrent processing prevention
	}
}

// LoadConfig loads WAL configuration from environment variables.
// Missing variables fall back to DefaultConfig values.
func LoadConfig() Config {
	defaults := DefaultConfig()

	return Config{
		Enabled:          getEnvBool("WAL_ENABLED", defaults.Enabled),
		Path:             getEnv("WAL_PATH", defaults.Path),
		SyncWrites:       getEnvBool("WAL_SYNC_WRITES", defaults.SyncWrites),
		RetryInterval:    getEnvDuration("WAL_RETRY_INTERVAL", defaults.RetryInterval),
		MaxRetries:       getEnvInt("WAL_MAX_RETRIES", defaults.MaxRetries),
		RetryBackoff:     getEnvDuration("WAL_RETRY_BACKOFF", defaults.RetryBackoff),
		CompactInterval:  getEnvDuration("WAL_COMPACT_INTERVAL", defaults.CompactInterval),
		EntryTTL:         getEnvDuration("WAL_ENTRY_TTL", defaults.EntryTTL),
		MemTableSize:     getEnvInt64("WAL_MEMTABLE_SIZE", defaults.MemTableSize),
		ValueLogFileSize: getEnvInt64("WAL_VLOG_SIZE", defaults.ValueLogFileSize),
		NumCompactors:    getEnvInt("WAL_NUM_COMPACTORS", defaults.NumCompactors),
		Compression:      getEnvBool("WAL_COMPRESSION", defaults.Compression),
		GCRatio:          getEnvFloat64("WAL_GC_RATIO", defaults.GCRatio),
		CloseTimeout:     getEnvDuration("WAL_CLOSE_TIMEOUT", defaults.CloseTimeout),
		NumMemtables:     getEnvInt("WAL_NUM_MEMTABLES", defaults.NumMemtables),
		BlockCacheSize:   getEnvInt64("WAL_BLOCK_CACHE_SIZE", defaults.BlockCacheSize),
		IndexCacheSize:   getEnvInt64("WAL_INDEX_CACHE_SIZE", defaults.IndexCacheSize),
		LeaseDuration:    getEnvDuration("WAL_LEASE_DURATION", defaults.LeaseDuration),
	}
}

// Validate checks that the configuration is valid.
func (c *Config) Validate() error {
	if !c.Enabled {
		return nil // No validation needed if disabled
	}

	if c.Path == "" {
		return &ConfigError{Field: "Path", Message: "WAL path is required"}
	}

	if c.RetryInterval < time.Second {
		return &ConfigError{Field: "RetryInterval", Message: "must be at least 1 second"}
	}

	if c.MaxRetries < 1 {
		return &ConfigError{Field: "MaxRetries", Message: "must be at least 1"}
	}

	if c.RetryBackoff < time.Second {
		return &ConfigError{Field: "RetryBackoff", Message: "must be at least 1 second"}
	}

	if c.CompactInterval < time.Minute {
		return &ConfigError{Field: "CompactInterval", Message: "must be at least 1 minute"}
	}

	if c.EntryTTL < time.Hour {
		return &ConfigError{Field: "EntryTTL", Message: "must be at least 1 hour"}
	}

	if c.MemTableSize < 1024*1024 { // 1MB minimum
		return &ConfigError{Field: "MemTableSize", Message: "must be at least 1MB"}
	}

	if c.ValueLogFileSize < 1024*1024 { // 1MB minimum
		return &ConfigError{Field: "ValueLogFileSize", Message: "must be at least 1MB"}
	}

	if c.NumCompactors < 2 {
		return &ConfigError{Field: "NumCompactors", Message: "must be at least 2 (BadgerDB requirement)"}
	}

	if c.LeaseDuration < 30*time.Second {
		return &ConfigError{Field: "LeaseDuration", Message: "must be at least 30 seconds"}
	}

	return nil
}

// ConfigError represents a configuration validation error.
type ConfigError struct {
	Field   string
	Message string
}

func (e *ConfigError) Error() string {
	return "WAL config error: " + e.Field + ": " + e.Message
}

// Environment variable helpers (following patterns from internal/config/config.go)

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvInt64(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

func getEnvFloat64(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
			return floatVal
		}
	}
	return defaultValue
}
