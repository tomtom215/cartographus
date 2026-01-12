// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package eventprocessor

import (
	"os"
	"strconv"
	"time"
)

// Environment variable helper functions to reduce cyclomatic complexity

func getEnvBool(key string, defaultVal bool) bool {
	if v := os.Getenv(key); v != "" {
		return v == "true" || v == "1"
	}
	return defaultVal
}

func getEnvString(key string, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func getEnvInt64(key string, defaultVal int64) int64 {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.ParseInt(v, 10, 64); err == nil {
			return i
		}
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return defaultVal
}

func getEnvDuration(key string, defaultVal time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return defaultVal
}

// NATSConfig holds NATS JetStream configuration for event processing.
// Environment variables override defaults when present.
type NATSConfig struct {
	// Enabled controls whether event processing is active.
	// Env: NATS_ENABLED (default: false)
	Enabled bool

	// URL is the NATS server connection URL.
	// Env: NATS_URL (default: nats://127.0.0.1:4222)
	URL string

	// EmbeddedServer enables embedded NATS server.
	// If false, expects external NATS server at URL.
	// Env: NATS_EMBEDDED (default: true)
	EmbeddedServer bool

	// StoreDir is the JetStream storage directory.
	// Env: NATS_STORE_DIR (default: /data/nats/jetstream)
	StoreDir string

	// MaxMemory is the maximum memory for JetStream in bytes.
	// Env: NATS_MAX_MEMORY (default: 1073741824 = 1GB)
	MaxMemory int64

	// MaxStore is the maximum disk storage for JetStream in bytes.
	// Env: NATS_MAX_STORE (default: 10737418240 = 10GB)
	MaxStore int64

	// StreamRetentionDays is how long to keep events.
	// Env: NATS_RETENTION_DAYS (default: 7)
	StreamRetentionDays int

	// BatchSize is the number of events to batch before writing to DuckDB.
	// Env: NATS_BATCH_SIZE (default: 1000)
	BatchSize int

	// FlushInterval is the maximum time between DuckDB flushes.
	// Env: NATS_FLUSH_INTERVAL (default: 5s)
	FlushInterval time.Duration

	// SubscribersCount is the number of concurrent message processors.
	// Env: NATS_SUBSCRIBERS (default: 4)
	//
	// DETERMINISM WARNING: When SubscribersCount > 1, messages may be processed
	// out of order because multiple goroutines consume from the same queue.
	// For strict ordering guarantees (system of record requirements):
	//   - Set SubscribersCount = 1 for single-threaded processing
	//   - Use correlation keys for event deduplication across sources
	//   - Rely on DuckDB's UPSERT with CorrelationKey for idempotency
	//
	// Trade-off: Higher values improve throughput but sacrifice ordering.
	// For most media analytics use cases, eventual consistency is acceptable
	// since playback events are naturally timestamped and can be reordered
	// during query time. Set to 1 only when strict insert ordering is required.
	SubscribersCount int

	// DurableName is the consumer durable name for message tracking.
	// Env: NATS_DURABLE_NAME (default: media-processor)
	DurableName string

	// QueueGroup is the queue group for load balancing.
	// Env: NATS_QUEUE_GROUP (default: processors)
	QueueGroup string
}

// DefaultNATSConfig returns production defaults for NATS configuration.
func DefaultNATSConfig() NATSConfig {
	return NATSConfig{
		Enabled:             false,
		URL:                 "nats://127.0.0.1:4222",
		EmbeddedServer:      true,
		StoreDir:            "/data/nats/jetstream",
		MaxMemory:           1 << 30,  // 1GB
		MaxStore:            10 << 30, // 10GB
		StreamRetentionDays: 7,
		BatchSize:           1000,
		FlushInterval:       5 * time.Second,
		SubscribersCount:    4,
		DurableName:         "media-processor",
		QueueGroup:          "processors",
	}
}

// LoadNATSConfig loads NATS configuration from environment variables.
// Unset variables use defaults from DefaultNATSConfig.
func LoadNATSConfig() NATSConfig {
	cfg := DefaultNATSConfig()

	cfg.Enabled = getEnvBool("NATS_ENABLED", cfg.Enabled)
	cfg.URL = getEnvString("NATS_URL", cfg.URL)
	cfg.EmbeddedServer = getEnvBool("NATS_EMBEDDED", cfg.EmbeddedServer)
	cfg.StoreDir = getEnvString("NATS_STORE_DIR", cfg.StoreDir)
	cfg.MaxMemory = getEnvInt64("NATS_MAX_MEMORY", cfg.MaxMemory)
	cfg.MaxStore = getEnvInt64("NATS_MAX_STORE", cfg.MaxStore)
	cfg.StreamRetentionDays = getEnvInt("NATS_RETENTION_DAYS", cfg.StreamRetentionDays)
	cfg.BatchSize = getEnvInt("NATS_BATCH_SIZE", cfg.BatchSize)
	cfg.FlushInterval = getEnvDuration("NATS_FLUSH_INTERVAL", cfg.FlushInterval)
	cfg.SubscribersCount = getEnvInt("NATS_SUBSCRIBERS", cfg.SubscribersCount)
	cfg.DurableName = getEnvString("NATS_DURABLE_NAME", cfg.DurableName)
	cfg.QueueGroup = getEnvString("NATS_QUEUE_GROUP", cfg.QueueGroup)

	return cfg
}

// ServerConfig holds embedded NATS server configuration.
type ServerConfig struct {
	Host              string
	Port              int
	StoreDir          string
	JetStreamMaxMem   int64
	JetStreamMaxStore int64
	EnableClustering  bool
	ClusterName       string
	Routes            []string
}

// DefaultServerConfig returns production defaults for embedded NATS server.
func DefaultServerConfig() ServerConfig {
	return ServerConfig{
		Host:              "127.0.0.1",
		Port:              4222,
		StoreDir:          "/data/nats/jetstream",
		JetStreamMaxMem:   1 << 30,  // 1GB
		JetStreamMaxStore: 10 << 30, // 10GB
		EnableClustering:  false,
	}
}

// PublisherConfig holds publisher configuration.
type PublisherConfig struct {
	URL              string
	MaxReconnects    int
	ReconnectWait    time.Duration
	ReconnectBuffer  int
	EnableTrackMsgID bool // nolint:revive // ID is correct per Go conventions
}

// DefaultPublisherConfig returns production defaults for publisher.
func DefaultPublisherConfig(url string) PublisherConfig {
	return PublisherConfig{
		URL:              url,
		MaxReconnects:    -1, // Unlimited
		ReconnectWait:    2 * time.Second,
		ReconnectBuffer:  8 * 1024 * 1024, // 8MB
		EnableTrackMsgID: true,
	}
}

// SubscriberConfig holds subscriber configuration.
type SubscriberConfig struct {
	URL              string
	DurableName      string
	QueueGroup       string
	SubscribersCount int
	AckWaitTimeout   time.Duration
	MaxDeliver       int
	MaxAckPending    int
	CloseTimeout     time.Duration
	MaxReconnects    int
	ReconnectWait    time.Duration
	// StreamName is the name of the JetStream stream to bind to.
	// When set, AutoProvision is disabled and the subscriber binds to
	// an existing stream using nats.BindStream(). This is required when
	// subscribing to topics with wildcards (e.g., "playback.>") because
	// NATS stream names cannot contain wildcards.
	StreamName string
}

// DefaultSubscriberConfig returns production defaults for subscriber.
func DefaultSubscriberConfig(url string) SubscriberConfig {
	return SubscriberConfig{
		URL:              url,
		DurableName:      "media-processor",
		QueueGroup:       "processors",
		SubscribersCount: 4,
		AckWaitTimeout:   30 * time.Second,
		MaxDeliver:       5,    // Max redelivery attempts
		MaxAckPending:    1000, // Flow control
		CloseTimeout:     30 * time.Second,
		MaxReconnects:    -1,
		ReconnectWait:    2 * time.Second,
	}
}

// StreamConfig defines media event stream settings.
type StreamConfig struct {
	Name            string
	Subjects        []string
	MaxAge          time.Duration
	MaxBytes        int64
	MaxMsgs         int64
	DuplicateWindow time.Duration
	Replicas        int
}

// DefaultStreamConfig returns production stream configuration.
func DefaultStreamConfig() StreamConfig {
	return StreamConfig{
		Name: "MEDIA_EVENTS",
		Subjects: []string{
			"playback.>",
			"plex.>",
			"jellyfin.>",
			"tautulli.>",
		},
		MaxAge:          7 * 24 * time.Hour,      // 7 days
		MaxBytes:        10 * 1024 * 1024 * 1024, // 10GB
		MaxMsgs:         -1,                      // Unlimited
		DuplicateWindow: 2 * time.Minute,
		Replicas:        1, // Increase for clustering
	}
}

// AppenderConfig holds batch appender configuration.
type AppenderConfig struct {
	Table         string
	BatchSize     int
	FlushInterval time.Duration
}

// DefaultAppenderConfig returns production defaults for appender.
func DefaultAppenderConfig() AppenderConfig {
	return AppenderConfig{
		Table:         "playback_events",
		BatchSize:     1000,
		FlushInterval: 5 * time.Second,
	}
}

// CircuitBreakerConfig holds circuit breaker settings.
type CircuitBreakerConfig struct {
	Name             string
	MaxRequests      uint32        // Allowed in half-open state
	Interval         time.Duration // Reset interval for counts
	Timeout          time.Duration // Time to stay open
	FailureThreshold uint32        // Failures before opening
}

// DefaultCircuitBreakerConfig returns production defaults.
func DefaultCircuitBreakerConfig(name string) CircuitBreakerConfig {
	return CircuitBreakerConfig{
		Name:             name,
		MaxRequests:      3,
		Interval:         30 * time.Second,
		Timeout:          10 * time.Second,
		FailureThreshold: 5,
	}
}
