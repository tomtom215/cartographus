// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package config

import (
	"fmt"
	"time"
)

// Config holds all application configuration loaded from environment variables and config files.
// Provides centralized configuration management for all application components including
// data sources (Tautulli, Plex), database, synchronization, server, API, security, and logging.
//
// Configuration Loading Order (Koanf v2):
//  1. Defaults: Built-in sensible defaults for all optional settings
//  2. Config File: Optional YAML config file (config.yaml) for persistent settings
//  3. Environment Variables: Override any setting via environment variables
//
// Configuration Categories:
//
//  1. Data Sources:
//     - Tautulli: Primary data source for playback history
//     - Plex: Optional hybrid mode for real-time updates and historical sync
//
//  2. Infrastructure:
//     - Database: DuckDB configuration (path, memory, mock data)
//     - Sync: Periodic synchronization settings
//     - Server: HTTP server configuration (port, host, timeout, location)
//     - NATS: Event processing with Watermill/NATS JetStream (optional)
//
//  3. API & Security:
//     - API: Pagination and response limits
//     - Security: Authentication, rate limiting, session management
//
//  4. Observability:
//     - Logging: Log levels and output formats
//
// Example - Load configuration from environment:
//
//	cfg, err := config.Load()
//	if err != nil {
//	    log.Fatal("Failed to load config:", err)
//	}
//	// cfg.Tautulli.URL, cfg.Database.Path, etc. are now populated
//
// Example - Access configuration values:
//
//	db, err := database.New(cfg.Database)
//	syncManager := sync.NewManager(cfg.Sync, cfg.Tautulli)
//	server := http.Server{Addr: fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)}
//
// Validation:
// The Load() function validates all required fields and returns an error if:
//   - Required environment variables are missing (TAUTULLI_URL, TAUTULLI_API_KEY)
//   - Values are malformed (invalid URL format, negative numbers)
//   - Authentication is enabled but credentials are incomplete
//
// Thread Safety:
// Config is immutable after Load() and safe for concurrent read access from multiple goroutines.
type Config struct {
	Tautulli   TautulliConfig   `koanf:"tautulli"`
	Plex       PlexConfig       `koanf:"plex"`       // Optional: Single Plex server (v1.37) - use PlexServers for multiple
	Jellyfin   JellyfinConfig   `koanf:"jellyfin"`   // Optional: Single Jellyfin server (v1.51) - use JellyfinServers for multiple
	Emby       EmbyConfig       `koanf:"emby"`       // Optional: Single Emby server (v1.51) - use EmbyServers for multiple
	NATS       NATSConfig       `koanf:"nats"`       // Optional: Event-driven processing with Watermill/NATS JetStream (v1.48)
	Import     ImportConfig     `koanf:"import"`     // Optional: Direct Tautulli database import (v1.49)
	Detection  DetectionConfig  `koanf:"detection"`  // Optional: Detection engine for security monitoring (ADR-0020)
	VPN        VPNConfig        `koanf:"vpn"`        // Optional: VPN detection service configuration
	Recommend  RecommendConfig  `koanf:"recommend"`  // Optional: Recommendation engine (ADR-0024)
	GeoIP      GeoIPConfig      `koanf:"geoip"`      // Optional: Standalone GeoIP provider configuration (v2.0)
	Newsletter NewsletterConfig `koanf:"newsletter"` // Optional: Newsletter scheduler for automated digest delivery
	Database   DatabaseConfig   `koanf:"database"`
	Sync       SyncConfig       `koanf:"sync"`
	Server     ServerConfig     `koanf:"server"`
	API        APIConfig        `koanf:"api"`
	Security   SecurityConfig   `koanf:"security"`
	Logging    LoggingConfig    `koanf:"logging"`

	// Multi-Server Support (v2.1)
	// Use these arrays to configure multiple servers of the same platform type.
	// If arrays are configured, they take precedence over the single-server configs above.
	// Each server MUST have a unique ServerID for proper deduplication and user mapping.
	PlexServers     []PlexConfig     `koanf:"plex_servers"`     // Optional: Multiple Plex servers (v2.1)
	JellyfinServers []JellyfinConfig `koanf:"jellyfin_servers"` // Optional: Multiple Jellyfin servers (v2.1)
	EmbyServers     []EmbyConfig     `koanf:"emby_servers"`     // Optional: Multiple Emby servers (v2.1)
}

// TautulliConfig holds Tautulli connection settings for optional data source.
// Tautulli provides enhanced playback history, geolocation, and metadata enrichment.
// As of v2.0, Tautulli is OPTIONAL - Cartographus can work standalone with direct
// Plex, Jellyfin, and/or Emby integrations.
//
// Use Cases for Tautulli:
//   - Historical data migration from existing Tautulli installations
//   - Enhanced metadata enrichment (Tautulli collects more fields than direct APIs)
//   - Geolocation data (Tautulli has built-in GeoIP lookup)
//   - Backward compatibility with existing deployments
//
// Environment Variables:
//   - TAUTULLI_ENABLED: Enable Tautulli integration (default: false)
//   - TAUTULLI_URL: Tautulli server URL (e.g., http://localhost:8181)
//   - TAUTULLI_API_KEY: Tautulli API key from Settings > Web Interface
//
// Example - Enable Tautulli:
//
//	cfg := TautulliConfig{
//	    Enabled: true,
//	    URL:     "http://localhost:8181",
//	    APIKey:  "your-api-key-here",
//	}
//
// Example - Standalone mode (no Tautulli):
//
//	cfg := TautulliConfig{
//	    Enabled: false, // Default - Tautulli not required
//	}
type TautulliConfig struct {
	Enabled  bool   `koanf:"enabled"` // Master toggle for Tautulli integration (default: false)
	URL      string `koanf:"url"`
	APIKey   string `koanf:"api_key"`
	ServerID string `koanf:"server_id"` // Unique identifier for this Tautulli instance (for multi-server deduplication)
}

// PlexConfig holds Plex API connection settings for hybrid data architecture (v1.37+).
// Provides optional Plex integration for real-time updates, historical backfill,
// transcode monitoring, and buffer health tracking.
//
// Hybrid Architecture Benefits:
//   - Real-time playback updates via WebSocket (sub-second latency)
//   - Historical data backfill for complete playback history
//   - Active transcode session monitoring with quality tracking
//   - Predictive buffer health monitoring (10-15s advance warning)
//   - Webhook receiver for event-driven updates
//
// Environment Variables:
//   - PLEX_ENABLED: Enable Plex integration (default: false)
//   - PLEX_URL: Plex Media Server URL (e.g., http://localhost:32400)
//   - PLEX_TOKEN: X-Plex-Token for authentication
//   - PLEX_HISTORICAL_SYNC: One-time historical backfill (default: false)
//   - PLEX_SYNC_DAYS_BACK: Historical sync lookback period in days (default: 30)
//   - PLEX_SYNC_INTERVAL: Missed event check interval (default: 5m)
//   - PLEX_REALTIME_ENABLED: Enable WebSocket for real-time updates (default: false)
//   - PLEX_OAUTH_CLIENT_ID: OAuth 2.0 client ID
//   - PLEX_OAUTH_CLIENT_SECRET: OAuth 2.0 client secret (optional)
//   - PLEX_OAUTH_REDIRECT_URI: OAuth callback URL
//   - PLEX_TRANSCODE_MONITORING: Enable transcode monitoring (default: false)
//   - PLEX_TRANSCODE_MONITORING_INTERVAL: Poll interval (default: 10s)
//   - ENABLE_BUFFER_HEALTH_MONITORING: Enable buffer health tracking (default: false)
//   - BUFFER_HEALTH_POLL_INTERVAL: Buffer health poll interval (default: 5s)
//   - BUFFER_HEALTH_CRITICAL_THRESHOLD: Critical threshold % (default: 20)
//   - BUFFER_HEALTH_RISKY_THRESHOLD: Risky threshold % (default: 50)
//   - PLEX_WEBHOOKS_ENABLED: Enable webhook receiver (default: false)
//   - PLEX_WEBHOOK_SECRET: HMAC-SHA256 signature secret (required if webhooks enabled)
//
// Example - Basic Plex integration:
//
//	cfg := PlexConfig{
//	    Enabled:         true,
//	    URL:            "http://localhost:32400",
//	    Token:          "your-plex-token",
//	    RealtimeEnabled: true,
//	}
//
// Example - Full feature set:
//
//	cfg := PlexConfig{
//	    Enabled:                       true,
//	    URL:                          "http://localhost:32400",
//	    Token:                        "your-plex-token",
//	    HistoricalSync:               true,
//	    SyncDaysBack:                 90,
//	    RealtimeEnabled:              true,
//	    TranscodeMonitoring:          true,
//	    TranscodeMonitoringInterval:  10 * time.Second,
//	    BufferHealthMonitoring:       true,
//	    BufferHealthPollInterval:     5 * time.Second,
//	    BufferHealthCriticalThreshold: 20.0,
//	    BufferHealthRiskyThreshold:    50.0,
//	    WebhooksEnabled:              true,
//	    WebhookSecret:                "webhook-secret-key",
//	}
type PlexConfig struct {
	Enabled         bool          `koanf:"enabled"`          // Master toggle for Plex integration
	ServerID        string        `koanf:"server_id"`        // Unique identifier for this Plex server (for multi-server deduplication)
	URL             string        `koanf:"url"`              // Plex Media Server URL (http://localhost:32400)
	Token           string        `koanf:"token"`            // X-Plex-Token for authentication
	HistoricalSync  bool          `koanf:"historical_sync"`  // One-time backfill of all history
	SyncDaysBack    int           `koanf:"sync_days_back"`   // How far back to sync (days)
	SyncInterval    time.Duration `koanf:"sync_interval"`    // How often to check for missed events
	RealtimeEnabled bool          `koanf:"realtime_enabled"` // Enable Plex WebSocket for real-time updates (v1.39)

	// OAuth 2.0 PKCE Authentication (Sprint 1, Task 1.1)
	OAuthClientID     string `koanf:"oauth_client_id"`     // Plex OAuth client ID (obtain from Plex app registration)
	OAuthClientSecret string `koanf:"oauth_client_secret"` // Plex OAuth client secret (optional for public clients)
	OAuthRedirectURI  string `koanf:"oauth_redirect_uri"`  // OAuth callback URL (e.g., http://localhost:3857/api/v1/auth/plex/callback)

	// Active Transcode Monitoring (Phase 1.2: v1.40)
	TranscodeMonitoring         bool          `koanf:"transcode_monitoring"`          // Enable periodic transcode session polling
	TranscodeMonitoringInterval time.Duration `koanf:"transcode_monitoring_interval"` // Polling interval (default: 10s, WARNING: don't set too low to avoid overloading Plex)

	// Buffer Health Monitoring (Phase 2.1: v1.41)
	BufferHealthMonitoring        bool          `koanf:"buffer_health_monitoring"`         // Enable real-time buffer health monitoring for predictive buffering detection
	BufferHealthPollInterval      time.Duration `koanf:"buffer_health_poll_interval"`      // Polling interval (default: 5s, recommended: 5-10s for balance between responsiveness and load)
	BufferHealthCriticalThreshold float64       `koanf:"buffer_health_critical_threshold"` // Critical health threshold percentage (default: 20%, alert when buffer <20%)
	BufferHealthRiskyThreshold    float64       `koanf:"buffer_health_risky_threshold"`    // Risky health threshold percentage (default: 50%, warn when buffer 20-50%)

	// Webhook Receiver (Sprint 1, Task 1.3: v1.43)
	WebhooksEnabled bool   `koanf:"webhooks_enabled"` // Enable Plex webhook endpoint
	WebhookSecret   string `koanf:"webhook_secret"`   // HMAC-SHA256 secret for webhook signature verification (required if webhooks enabled)

	// Session Polling (Optional Backup Mechanism)
	//
	// WHY THIS EXISTS:
	// The Plex WebSocket (PLEX_REALTIME_ENABLED) is the primary real-time mechanism
	// and is RECOMMENDED for most deployments. It provides sub-second notifications
	// for playback state changes.
	//
	// WHY IT'S USUALLY UNNECESSARY:
	// - WebSocket provides instant notifications (~50ms latency)
	// - Tautulli sync catches any sessions WebSocket might miss
	// - Webhooks provide another layer of redundancy
	// - Three mechanisms (WebSocket + Tautulli + Webhooks) = comprehensive coverage
	//
	// WHY SOME MAY STILL WANT IT:
	// - Extra paranoia: Belt-and-suspenders approach for mission-critical deployments
	// - Unreliable networks: If WebSocket connections frequently drop
	// - Firewall issues: Some networks block WebSocket upgrades but allow HTTP polling
	// - Testing/debugging: Verify session detection is working independently of WebSocket
	// - Plex server issues: Some Plex versions have WebSocket reliability problems
	//
	// PERFORMANCE NOTES:
	// - Polling hits Plex API every SessionPollingInterval (default: 30s)
	// - Each poll = 1 HTTP request to /status/sessions
	// - Deduplication prevents duplicate events (tracks seen session keys)
	// - Minimal server load at 30s intervals (2 requests/minute)
	//
	// RECOMMENDATION:
	// Leave disabled (default: false). Only enable if you experience detection gaps
	// that WebSocket and Tautulli sync are not catching.
	SessionPollingEnabled  bool          `koanf:"session_polling_enabled"`  // Enable periodic session polling as backup to WebSocket
	SessionPollingInterval time.Duration `koanf:"session_polling_interval"` // Polling interval (default: 30s, minimum: 10s)
}

// JellyfinConfig holds Jellyfin API connection settings for direct integration (v1.51+).
// Provides optional Jellyfin integration for real-time playback updates via WebSocket,
// REST API session polling, and webhook receiver for event-driven notifications.
//
// Integration Features:
//   - Real-time playback updates via WebSocket (SessionsStart subscription)
//   - REST API session polling as backup mechanism
//   - Webhook receiver for playback events (requires jellyfin-plugin-webhook)
//   - Cross-source deduplication via CorrelationKey pattern
//   - Integration with NATS JetStream event processing
//
// Environment Variables:
//   - JELLYFIN_ENABLED: Enable Jellyfin integration (default: false)
//   - JELLYFIN_URL: Jellyfin server URL (e.g., http://localhost:8096)
//   - JELLYFIN_API_KEY: Jellyfin API key from Admin Dashboard > API Keys
//   - JELLYFIN_USER_ID: Optional user ID for authentication (for user-scoped API keys)
//   - JELLYFIN_REALTIME_ENABLED: Enable WebSocket for real-time updates (default: false)
//   - JELLYFIN_SESSION_POLLING_ENABLED: Enable session polling as backup (default: false)
//   - JELLYFIN_SESSION_POLLING_INTERVAL: Polling interval (default: 30s)
//   - JELLYFIN_WEBHOOKS_ENABLED: Enable webhook receiver (default: false)
//   - JELLYFIN_WEBHOOK_SECRET: Secret for webhook signature verification (optional)
//
// Example - Basic Jellyfin integration:
//
//	cfg := JellyfinConfig{
//	    Enabled:         true,
//	    URL:            "http://localhost:8096",
//	    APIKey:         "your-jellyfin-api-key",
//	    RealtimeEnabled: true,
//	}
//
// Example - Full feature set:
//
//	cfg := JellyfinConfig{
//	    Enabled:                true,
//	    URL:                   "http://localhost:8096",
//	    APIKey:                "your-jellyfin-api-key",
//	    RealtimeEnabled:        true,
//	    SessionPollingEnabled:  true,
//	    SessionPollingInterval: 30 * time.Second,
//	    WebhooksEnabled:        true,
//	    WebhookSecret:          "webhook-secret-key",
//	}
type JellyfinConfig struct {
	Enabled  bool   `koanf:"enabled"`   // Master toggle for Jellyfin integration
	ServerID string `koanf:"server_id"` // Unique identifier for this Jellyfin server (for multi-server deduplication)
	URL      string `koanf:"url"`       // Jellyfin server URL (http://localhost:8096)
	APIKey   string `koanf:"api_key"`   // Jellyfin API key for authentication
	UserID   string `koanf:"user_id"`   // Optional: User ID for user-scoped API keys

	// Real-time WebSocket
	RealtimeEnabled bool `koanf:"realtime_enabled"` // Enable WebSocket for real-time updates

	// Session Polling (backup mechanism)
	SessionPollingEnabled  bool          `koanf:"session_polling_enabled"`  // Enable session polling as backup
	SessionPollingInterval time.Duration `koanf:"session_polling_interval"` // Polling interval (default: 30s)

	// Webhooks (requires jellyfin-plugin-webhook)
	WebhooksEnabled bool   `koanf:"webhooks_enabled"` // Enable webhook receiver endpoint
	WebhookSecret   string `koanf:"webhook_secret"`   // Secret for webhook signature verification (optional)
}

// EmbyConfig holds Emby API connection settings for direct integration (v1.51+).
// Provides optional Emby integration for real-time playback updates via WebSocket,
// REST API session polling, and webhook receiver for event-driven notifications.
//
// Integration Features:
//   - Real-time playback updates via WebSocket (SessionsStart subscription)
//   - REST API session polling as backup mechanism
//   - Webhook receiver for playback events (requires Emby webhook plugin)
//   - Cross-source deduplication via CorrelationKey pattern
//   - Integration with NATS JetStream event processing
//
// Environment Variables:
//   - EMBY_ENABLED: Enable Emby integration (default: false)
//   - EMBY_URL: Emby server URL (e.g., http://localhost:8096)
//   - EMBY_API_KEY: Emby API key from Admin Dashboard > API Keys
//   - EMBY_USER_ID: Optional user ID for authentication
//   - EMBY_REALTIME_ENABLED: Enable WebSocket for real-time updates (default: false)
//   - EMBY_SESSION_POLLING_ENABLED: Enable session polling as backup (default: false)
//   - EMBY_SESSION_POLLING_INTERVAL: Polling interval (default: 30s)
//   - EMBY_WEBHOOKS_ENABLED: Enable webhook receiver (default: false)
//   - EMBY_WEBHOOK_SECRET: Secret for webhook signature verification (optional)
//
// Example - Basic Emby integration:
//
//	cfg := EmbyConfig{
//	    Enabled:         true,
//	    URL:            "http://localhost:8096",
//	    APIKey:         "your-emby-api-key",
//	    RealtimeEnabled: true,
//	}
type EmbyConfig struct {
	Enabled  bool   `koanf:"enabled"`   // Master toggle for Emby integration
	ServerID string `koanf:"server_id"` // Unique identifier for this Emby server (for multi-server deduplication)
	URL      string `koanf:"url"`       // Emby server URL (http://localhost:8096)
	APIKey   string `koanf:"api_key"`   // Emby API key for authentication
	UserID   string `koanf:"user_id"`   // Optional: User ID for user-scoped API keys

	// Real-time WebSocket
	RealtimeEnabled bool `koanf:"realtime_enabled"` // Enable WebSocket for real-time updates

	// Session Polling (backup mechanism)
	SessionPollingEnabled  bool          `koanf:"session_polling_enabled"`  // Enable session polling as backup
	SessionPollingInterval time.Duration `koanf:"session_polling_interval"` // Polling interval (default: 30s)

	// Webhooks
	WebhooksEnabled bool   `koanf:"webhooks_enabled"` // Enable webhook receiver endpoint
	WebhookSecret   string `koanf:"webhook_secret"`   // Secret for webhook signature verification (optional)
}

// NATSConfig holds NATS JetStream configuration for event-driven processing (v1.48+).
// Enables Watermill-based event processing for reliable playback event handling
// with exactly-once semantics, deduplication, and guaranteed delivery.
//
// Architecture Modes:
//
//   - Event Sourcing Mode (default): NATS_EVENT_SOURCING=true
//     NATS JetStream is the single source of truth. All data sources (Tautulli, Plex,
//     Jellyfin) publish to NATS, and DuckDBConsumer is the ONLY writer to DuckDB.
//     Enables cross-source deduplication, event replay, and multi-server support.
//     Required for setups with synced Plex + Jellyfin instances.
//
//   - Notification Mode (legacy): NATS_EVENT_SOURCING=false
//     Data is written to DuckDB first, then published to NATS for real-time notifications.
//     Only use this mode if you need backward-compatible DB-first writes.
//
// Architecture Benefits:
//   - Decoupled event processing from HTTP handlers
//   - Exactly-once delivery via JetStream acknowledgments
//   - Message deduplication via correlation key (cross-source)
//   - Circuit breaker protection for resilience
//   - Batch processing for high-throughput DuckDB writes
//   - Real-time WebSocket notifications via NATS pub/sub
//   - Event replay for debugging and audit (event sourcing mode)
//
// Environment Variables:
//   - NATS_ENABLED: Enable event processing (default: true)
//   - NATS_EVENT_SOURCING: Enable full event sourcing mode (default: true)
//   - NATS_URL: NATS server connection URL (default: nats://127.0.0.1:4222)
//   - NATS_EMBEDDED: Use embedded NATS server (default: true)
//   - NATS_STORE_DIR: JetStream storage directory (default: /data/nats/jetstream)
//   - NATS_MAX_MEMORY: Max memory for JetStream in bytes (default: 1073741824 = 1GB)
//   - NATS_MAX_STORE: Max disk storage for JetStream in bytes (default: 10737418240 = 10GB)
//   - NATS_RETENTION_DAYS: Event retention period in days (default: 7)
//   - NATS_BATCH_SIZE: Batch size for DuckDB writes (default: 1000)
//   - NATS_FLUSH_INTERVAL: Max time between DuckDB flushes (default: 5s)
//   - NATS_SUBSCRIBERS: Number of concurrent message processors (default: 4)
//   - NATS_DURABLE_NAME: Consumer durable name (default: media-processor)
//   - NATS_QUEUE_GROUP: Queue group for load balancing (default: processors)
//
// Example - Default configuration (event sourcing mode):
//
//	cfg := NATSConfig{
//	    Enabled:        true,  // default
//	    EventSourcing:  true,  // default - NATS-first architecture
//	    EmbeddedServer: true,  // default - embedded NATS server
//	}
//
// Example - Legacy notification mode (DB-first writes):
//
//	cfg := NATSConfig{
//	    Enabled:        true,
//	    EventSourcing:  false, // legacy mode
//	    EmbeddedServer: true,
//	}
//
// Example - External NATS cluster:
//
//	cfg := NATSConfig{
//	    Enabled:        true,
//	    URL:           "nats://nats-cluster:4222",
//	    EmbeddedServer: false,
//	}
type NATSConfig struct {
	// Enabled controls whether event processing is active.
	Enabled bool `koanf:"enabled"`

	// EventSourcing enables full event-sourcing mode where NATS JetStream
	// is the single source of truth. When true:
	//   - All data sources (Tautulli, Plex, Jellyfin) publish to NATS only
	//   - DuckDBConsumer is the ONLY writer to DuckDB
	//   - Cross-source deduplication is handled at the event level
	// When false (default):
	//   - Legacy mode: Sync Manager writes to DuckDB first, then publishes to NATS
	//   - NATS is used only for real-time WebSocket notifications
	// Enable this for multi-server setups (Plex + Jellyfin) or when you need
	// event replay capabilities.
	EventSourcing bool `koanf:"event_sourcing"`

	// URL is the NATS server connection URL.
	URL string `koanf:"url"`

	// EmbeddedServer enables embedded NATS server.
	// If false, expects external NATS server at URL.
	EmbeddedServer bool `koanf:"embedded_server"`

	// StoreDir is the JetStream storage directory.
	StoreDir string `koanf:"store_dir"`

	// MaxMemory is the maximum memory for JetStream in bytes.
	MaxMemory int64 `koanf:"max_memory"`

	// MaxStore is the maximum disk storage for JetStream in bytes.
	MaxStore int64 `koanf:"max_store"`

	// StreamRetentionDays is how long to keep events.
	StreamRetentionDays int `koanf:"stream_retention_days"`

	// BatchSize is the number of events to batch before writing to DuckDB.
	BatchSize int `koanf:"batch_size"`

	// FlushInterval is the maximum time between DuckDB flushes.
	FlushInterval time.Duration `koanf:"flush_interval"`

	// SubscribersCount is the number of concurrent message processors.
	SubscribersCount int `koanf:"subscribers_count"`

	// DurableName is the consumer durable name for message tracking.
	DurableName string `koanf:"durable_name"`

	// QueueGroup is the queue group for load balancing.
	QueueGroup string `koanf:"queue_group"`

	// Router configuration (Watermill Router-based message processing)
	// These settings control the middleware stack for message handling.

	// RouterRetryCount is the maximum number of retries for failed messages.
	// Default: 3
	RouterRetryCount int `koanf:"router_retry_count"`

	// RouterRetryInitialInterval is the initial backoff interval for retries.
	// Default: 100ms
	RouterRetryInitialInterval time.Duration `koanf:"router_retry_initial_interval"`

	// RouterThrottlePerSecond limits messages processed per second (0 = unlimited).
	// Default: 0 (unlimited)
	RouterThrottlePerSecond int `koanf:"router_throttle_per_second"`

	// RouterDeduplicationEnabled enables message ID deduplication in the Router.
	// Default: true
	RouterDeduplicationEnabled bool `koanf:"router_deduplication_enabled"`

	// RouterDeduplicationTTL is how long to remember message IDs for deduplication.
	// Default: 5m
	RouterDeduplicationTTL time.Duration `koanf:"router_deduplication_ttl"`

	// RouterPoisonQueueEnabled enables routing of permanently failed messages to a poison queue.
	// Default: true
	RouterPoisonQueueEnabled bool `koanf:"router_poison_queue_enabled"`

	// RouterPoisonQueueTopic is the topic for permanently failed messages.
	// Default: "playback.poison"
	RouterPoisonQueueTopic string `koanf:"router_poison_queue_topic"`

	// RouterCloseTimeout is the maximum time to wait for graceful router shutdown.
	// Default: 30s
	RouterCloseTimeout time.Duration `koanf:"router_close_timeout"`
}

// ImportConfig holds Tautulli database import settings (v1.49+).
// Enables direct import from Tautulli SQLite database files or backups for:
//   - Migration from existing Tautulli setups without API connection
//   - Testing with production-like data without affecting live instances
//   - Historical data import from backup files
//
// The import process integrates with the existing NATS JetStream pipeline:
//   - Events are published to NATS for reliable delivery
//   - BadgerDB WAL ensures durability during import
//   - Correlation key deduplication prevents duplicate entries
//   - Progress tracking enables resumable imports
//
// Environment Variables:
//   - IMPORT_ENABLED: Enable import functionality (default: false)
//   - IMPORT_DB_PATH: Path to Tautulli SQLite database file (tautulli.db)
//   - IMPORT_BATCH_SIZE: Records per batch (default: 1000)
//   - IMPORT_DRY_RUN: Validate without importing (default: false)
//   - IMPORT_AUTO_START: Start import automatically on startup (default: false)
//
// Example - One-time import:
//
//	cfg := ImportConfig{
//	    Enabled:   true,
//	    DBPath:    "/path/to/tautulli.db",
//	    BatchSize: 1000,
//	}
//
// Example - Testing with dry run:
//
//	cfg := ImportConfig{
//	    Enabled: true,
//	    DBPath:  "/path/to/tautulli-backup.db",
//	    DryRun:  true,
//	}
type ImportConfig struct {
	// Enabled controls whether import functionality is active.
	Enabled bool `koanf:"enabled"`

	// DBPath is the path to the Tautulli SQLite database file.
	// Supports .db files and .zip backups (will extract automatically).
	DBPath string `koanf:"db_path"`

	// BatchSize is the number of records to process per batch.
	// Higher values improve throughput but use more memory.
	// Default: 1000
	BatchSize int `koanf:"batch_size"`

	// DryRun validates the import without writing to the database.
	// Useful for testing import compatibility before committing.
	DryRun bool `koanf:"dry_run"`

	// AutoStart triggers import automatically on application startup.
	// When false, import must be triggered via API endpoint.
	AutoStart bool `koanf:"auto_start"`

	// ResumeFromID allows resuming an interrupted import from a specific session ID.
	// Set to 0 to start from the beginning.
	ResumeFromID int64 `koanf:"resume_from_id"`

	// SkipGeolocation skips geolocation enrichment during import.
	// Set to true if geolocation data is already present in the source.
	SkipGeolocation bool `koanf:"skip_geolocation"`
}

// DatabaseConfig holds DuckDB settings
type DatabaseConfig struct {
	Path                   string `koanf:"path"`
	MaxMemory              string `koanf:"max_memory"`
	Threads                int    `koanf:"threads"`                  // Number of DuckDB threads (0 = use NumCPU)
	PreserveInsertionOrder bool   `koanf:"preserve_insertion_order"` // Whether to preserve insertion order (default true)
	SeedMockData           bool   `koanf:"seed_mock_data"`           // Enable mock data seeding for CI/CD screenshot tests
	SkipIndexes            bool   `koanf:"skip_indexes"`             // Skip index creation (for fast test setup - 97 indexes per DB)
}

// SyncConfig holds data synchronization settings
type SyncConfig struct {
	Interval      time.Duration `koanf:"interval"`
	Lookback      time.Duration `koanf:"lookback"`
	SyncAll       bool          `koanf:"sync_all"` // When true, sync all data ignoring Lookback (for testing)
	BatchSize     int           `koanf:"batch_size"`
	RetryAttempts int           `koanf:"retry_attempts"`
	RetryDelay    time.Duration `koanf:"retry_delay"`
}

// ServerConfig holds HTTP server settings
type ServerConfig struct {
	Port        int           `koanf:"port"`
	Host        string        `koanf:"host"`
	Timeout     time.Duration `koanf:"timeout"`
	Latitude    float64       `koanf:"latitude"`    // Server physical location latitude (optional, for visualization)
	Longitude   float64       `koanf:"longitude"`   // Server physical location longitude (optional, for visualization)
	Environment string        `koanf:"environment"` // Environment mode: "development", "staging", "production" (default: "development")
}

// APIConfig holds API pagination and response settings
type APIConfig struct {
	DefaultPageSize int `koanf:"default_page_size"`
	MaxPageSize     int `koanf:"max_page_size"`
}

// SecurityConfig holds authentication and authorization settings
type SecurityConfig struct {
	AuthMode          string        `koanf:"auth_mode"`
	JWTSecret         string        `koanf:"jwt_secret"`
	SessionTimeout    time.Duration `koanf:"session_timeout"`
	AdminUsername     string        `koanf:"admin_username"`
	AdminPassword     string        `koanf:"admin_password"`
	RateLimitReqs     int           `koanf:"rate_limit_reqs"`
	RateLimitWindow   time.Duration `koanf:"rate_limit_window"`
	RateLimitDisabled bool          `koanf:"rate_limit_disabled"`
	CORSOrigins       []string      `koanf:"cors_origins"`
	TrustedProxies    []string      `koanf:"trusted_proxies"`

	// Basic Auth RBAC Configuration
	// BasicAuthDefaultRole is the default role for Basic Auth users (default: viewer)
	// The configured admin user (ADMIN_USERNAME) is automatically assigned admin role
	// Other users start with this role and can be elevated via /api/admin/roles/assign
	BasicAuthDefaultRole string `koanf:"basic_auth_default_role"`

	// Session Store Configuration (ADR-0015 Phase 4)
	// SessionStore specifies the session storage backend: "memory" (default) or "badger"
	SessionStore string `koanf:"session_store"`
	// SessionStorePath is the path for BadgerDB storage (required when session_store=badger)
	SessionStorePath string `koanf:"session_store_path"`

	// Zero Trust Authentication & Authorization (ADR-0015)
	OIDC     OIDCConfig     `koanf:"oidc"`      // OIDC/OAuth 2.0 authentication
	PlexAuth PlexAuthConfig `koanf:"plex_auth"` // Plex OAuth authentication
	Casbin   CasbinConfig   `koanf:"casbin"`    // Casbin RBAC authorization
}

// OIDCConfig holds OIDC/OAuth 2.0 authentication settings.
// ADR-0015: Zero Trust Authentication & Authorization
//
// Environment Variables:
//   - OIDC_ISSUER_URL: OIDC provider issuer URL (required for oidc auth mode)
//   - OIDC_CLIENT_ID: OAuth 2.0 client ID (required for oidc auth mode)
//   - OIDC_CLIENT_SECRET: OAuth 2.0 client secret (optional for public clients)
//   - OIDC_REDIRECT_URL: OAuth callback URL (required for oidc auth mode)
//   - OIDC_SCOPES: Comma-separated list of OAuth scopes (default: openid,profile,email)
//   - OIDC_PKCE_ENABLED: Enable PKCE for public clients (default: true)
//   - OIDC_JWKS_CACHE_TTL: JWKS cache TTL (default: 1h)
//   - OIDC_SESSION_MAX_AGE: Session max age (default: 24h)
//   - OIDC_SESSION_SECRET: Secret for session encryption (optional)
//   - OIDC_COOKIE_NAME: Session cookie name (default: tautulli_session)
//   - OIDC_COOKIE_SECURE: Use secure cookies (default: true)
//   - OIDC_ROLES_CLAIM: JWT claim containing user roles (default: roles)
//   - OIDC_DEFAULT_ROLES: Default roles for new users (default: viewer)
//   - OIDC_USERNAME_CLAIMS: Claims to use for username (default: preferred_username,name,email)
type OIDCConfig struct {
	IssuerURL             string        `koanf:"issuer_url"`
	ClientID              string        `koanf:"client_id"`
	ClientSecret          string        `koanf:"client_secret"`
	RedirectURL           string        `koanf:"redirect_url"`
	PostLogoutRedirectURI string        `koanf:"post_logout_redirect_uri"` // URI to redirect after OIDC logout (Phase 4B)
	Scopes                []string      `koanf:"scopes"`
	PKCEEnabled           bool          `koanf:"pkce_enabled"`
	JWKSCacheTTL          time.Duration `koanf:"jwks_cache_ttl"`
	SessionMaxAge         time.Duration `koanf:"session_max_age"`
	SessionSecret         string        `koanf:"session_secret"`
	CookieName            string        `koanf:"cookie_name"`
	CookieSecure          bool          `koanf:"cookie_secure"`
	RolesClaim            string        `koanf:"roles_claim"`
	DefaultRoles          []string      `koanf:"default_roles"`
	UsernameClaims        []string      `koanf:"username_claims"`
}

// PlexAuthConfig holds Plex OAuth authentication settings.
// ADR-0015: Zero Trust Authentication & Authorization
//
// Environment Variables:
//   - PLEX_AUTH_CLIENT_ID: Plex app client ID (required for plex auth mode)
//   - PLEX_AUTH_CLIENT_SECRET: Plex app client secret (optional)
//   - PLEX_AUTH_REDIRECT_URI: Plex OAuth callback URL (required for plex auth mode)
//   - PLEX_AUTH_DEFAULT_ROLES: Default roles for Plex users (default: viewer)
//   - PLEX_AUTH_PLEX_PASS_ROLE: Additional role for Plex Pass users (optional)
//   - PLEX_AUTH_TIMEOUT: HTTP timeout for Plex API calls (default: 30s)
//   - PLEX_AUTH_SERVER_OWNER_ROLE: Role for Plex server owners (default: admin)
//   - PLEX_AUTH_SERVER_ADMIN_ROLE: Role for Plex server admins (default: editor)
//   - PLEX_AUTH_ENABLE_SERVER_DETECTION: Auto-detect server ownership (default: true)
//   - PLEX_AUTH_SERVER_MACHINE_ID: Limit detection to specific server (optional)
type PlexAuthConfig struct {
	ClientID     string        `koanf:"client_id"`
	ClientSecret string        `koanf:"client_secret"`
	RedirectURI  string        `koanf:"redirect_uri"`
	DefaultRoles []string      `koanf:"default_roles"`
	PlexPassRole string        `koanf:"plex_pass_role"`
	Timeout      time.Duration `koanf:"timeout"`

	// Server ownership detection settings
	// When enabled, the authenticator queries Plex API for server resources
	// and assigns roles based on ownership and admin status.
	ServerOwnerRole         string `koanf:"server_owner_role"`         // Role for server owners (default: admin)
	ServerAdminRole         string `koanf:"server_admin_role"`         // Role for server admins (default: editor)
	EnableServerDetection   bool   `koanf:"enable_server_detection"`   // Enable auto server detection (default: true)
	ServerMachineIdentifier string `koanf:"server_machine_identifier"` // Optional: limit to specific server
}

// CasbinConfig holds Casbin RBAC authorization settings.
// ADR-0015: Zero Trust Authentication & Authorization
//
// Environment Variables:
//   - CASBIN_MODEL_PATH: Path to Casbin model file (default: embedded)
//   - CASBIN_POLICY_PATH: Path to Casbin policy file (default: embedded)
//   - CASBIN_DEFAULT_ROLE: Default role for users without explicit role (default: viewer)
//   - CASBIN_AUTO_RELOAD: Enable automatic policy reload (default: true)
//   - CASBIN_RELOAD_INTERVAL: Policy reload interval (default: 30s)
//   - CASBIN_CACHE_ENABLED: Enable authorization decision caching (default: true)
//   - CASBIN_CACHE_TTL: Authorization cache TTL (default: 5m)
type CasbinConfig struct {
	ModelPath      string        `koanf:"model_path"`
	PolicyPath     string        `koanf:"policy_path"`
	DefaultRole    string        `koanf:"default_role"`
	AutoReload     bool          `koanf:"auto_reload"`
	ReloadInterval time.Duration `koanf:"reload_interval"`
	CacheEnabled   bool          `koanf:"cache_enabled"`
	CacheTTL       time.Duration `koanf:"cache_ttl"`
}

// LoggingConfig holds logging settings for zerolog.
//
// Environment Variables:
//   - LOG_LEVEL: trace, debug, info, warn, error (default: info)
//   - LOG_FORMAT: json, console (default: json)
//   - LOG_CALLER: true/false - include caller file:line (default: false)
type LoggingConfig struct {
	// Level is the minimum log level: trace, debug, info, warn, error.
	// Default: info
	Level string `koanf:"level"`

	// Format is the output format: json or console.
	// JSON is recommended for production (structured, machine-parseable).
	// Console is human-readable for development.
	// Default: json
	Format string `koanf:"format"`

	// Caller includes caller file and line number in logs.
	// Adds slight performance overhead.
	// Default: false
	Caller bool `koanf:"caller"`
}

// DetectionConfig holds detection engine configuration (ADR-0020).
// Enables security monitoring features like impossible travel detection,
// concurrent stream limits, and device velocity tracking.
//
// Environment Variables:
//   - DETECTION_ENABLED: Enable detection engine (default: true)
//   - DETECTION_TRUST_SCORE_DECREMENT: Points to deduct per violation (default: 10)
//   - DETECTION_TRUST_SCORE_RECOVERY: Daily recovery points (default: 1)
//   - DETECTION_TRUST_SCORE_THRESHOLD: Auto-restrict below this score (default: 50)
//   - DISCORD_WEBHOOK_URL: Discord webhook URL for alerts
//   - DISCORD_WEBHOOK_ENABLED: Enable Discord notifications (default: false)
//   - DISCORD_RATE_LIMIT_MS: Rate limit between messages (default: 1000)
//   - WEBHOOK_URL: Generic webhook URL for alerts
//   - WEBHOOK_ENABLED: Enable generic webhook notifications (default: false)
//   - WEBHOOK_RATE_LIMIT_MS: Rate limit between messages (default: 500)
//   - WEBHOOK_HEADERS: Comma-separated key=value headers (e.g., "Authorization=Bearer xyz,X-Custom=value")
type DetectionConfig struct {
	// Engine configuration
	Enabled             bool `koanf:"enabled"`
	TrustScoreDecrement int  `koanf:"trust_score_decrement"`
	TrustScoreRecovery  int  `koanf:"trust_score_recovery"`
	TrustScoreThreshold int  `koanf:"trust_score_threshold"`

	// Discord notifier configuration
	Discord DiscordNotifierConfig `koanf:"discord"`

	// Generic webhook notifier configuration
	Webhook WebhookNotifierConfig `koanf:"webhook"`
}

// DiscordNotifierConfig holds Discord webhook notification settings.
type DiscordNotifierConfig struct {
	WebhookURL  string `koanf:"webhook_url"`
	Enabled     bool   `koanf:"enabled"`
	RateLimitMs int    `koanf:"rate_limit_ms"`
}

// WebhookNotifierConfig holds generic webhook notification settings.
type WebhookNotifierConfig struct {
	WebhookURL  string            `koanf:"webhook_url"`
	Enabled     bool              `koanf:"enabled"`
	RateLimitMs int               `koanf:"rate_limit_ms"`
	Headers     map[string]string `koanf:"headers"`
}

// VPNConfig holds VPN detection service configuration.
// The VPN detection service identifies connections from known VPN providers
// to improve geolocation accuracy and flag potentially misleading analytics data.
//
// VPN connections can significantly skew geographic analytics since the apparent
// location reflects the VPN server, not the actual user location.
//
// Data Source:
// Uses gluetun VPN provider data (24+ providers, 10,000+ IPs) from:
// https://github.com/qdm12/gluetun
//
// Environment Variables:
//   - VPN_ENABLED: Enable VPN detection (default: true)
//   - VPN_DATA_FILE: Path to gluetun servers.json file (optional)
//   - VPN_CACHE_SIZE: Maximum lookup cache entries (default: 10000)
//   - VPN_AUTO_UPDATE: Enable automatic data updates (default: false)
//   - VPN_UPDATE_INTERVAL: Update check interval (default: 24h)
//
// Example - Basic VPN detection:
//
//	cfg := VPNConfig{
//	    Enabled:   true,
//	    CacheSize: 10000,
//	}
//
// Example - With custom data file:
//
//	cfg := VPNConfig{
//	    Enabled:   true,
//	    DataFile:  "/data/vpn/servers.json",
//	    CacheSize: 10000,
//	}
type VPNConfig struct {
	// Enabled controls whether VPN detection is active.
	// Default: true
	Enabled bool `koanf:"enabled"`

	// DataFile is the path to the gluetun servers.json file.
	// If empty, data is loaded from the database only.
	// Default: "" (empty)
	DataFile string `koanf:"data_file"`

	// CacheSize is the maximum number of lookup results to cache.
	// Default: 10000
	CacheSize int `koanf:"cache_size"`

	// AutoUpdate enables automatic data updates.
	// Default: false (future feature)
	AutoUpdate bool `koanf:"auto_update"`

	// UpdateInterval is how often to check for updates.
	// Default: 24h (future feature)
	UpdateInterval time.Duration `koanf:"update_interval"`
}

// RecommendConfig holds recommendation engine configuration (ADR-0024).
// The recommendation engine provides personalized media suggestions based on
// viewing history using a hybrid approach combining multiple algorithms.
//
// IMPORTANT: This feature is DISABLED by default due to computational requirements.
// Enable only on systems with sufficient resources (recommended: 4+ cores, 8GB+ RAM).
//
// Architecture Overview:
// The engine uses a multi-phase hybrid approach:
//   - Phase 1: Co-Visitation + Content-Based (lightweight, always available)
//   - Phase 2: EASE matrix factorization (medium compute, good accuracy)
//   - Phase 3: ALS + Collaborative Filtering (parallel training)
//   - Phase 4: Calibration reranking + FPMC sequential (personalization)
//   - Phase 5: LinUCB contextual bandits (exploration/exploitation)
//
// Environment Variables:
//   - RECOMMEND_ENABLED: Enable recommendation engine (default: false)
//   - RECOMMEND_TRAIN_INTERVAL: Training schedule interval (default: 24h)
//   - RECOMMEND_MIN_INTERACTIONS: Minimum interactions before training (default: 100)
//   - RECOMMEND_MODEL_PATH: Path to store trained models (default: /data/recommend)
//   - RECOMMEND_ALGORITHMS: Comma-separated list of enabled algorithms
//     (default: covisit,content - lightweight only)
//   - RECOMMEND_CACHE_TTL: Recommendation cache TTL (default: 5m)
//   - RECOMMEND_MAX_CANDIDATES: Maximum candidates to score (default: 1000)
//   - RECOMMEND_DIVERSITY_LAMBDA: MMR diversity parameter 0-1 (default: 0.7)
//
// Algorithm-Specific Settings:
//   - RECOMMEND_EASE_REGULARIZATION: EASE L2 regularization (default: 500.0)
//   - RECOMMEND_ALS_FACTORS: ALS latent factors (default: 50)
//   - RECOMMEND_ALS_ITERATIONS: ALS training iterations (default: 15)
//   - RECOMMEND_ALS_REGULARIZATION: ALS regularization (default: 0.1)
//   - RECOMMEND_KNN_NEIGHBORS: KNN neighbor count (default: 50)
//   - RECOMMEND_LINUCB_ALPHA: LinUCB exploration parameter (default: 1.0)
//
// Example - Enable lightweight recommendations only:
//
//	cfg := RecommendConfig{
//	    Enabled:    true,
//	    Algorithms: []string{"covisit", "content"},
//	}
//
// Example - Full recommendation suite (requires more resources):
//
//	cfg := RecommendConfig{
//	    Enabled:    true,
//	    Algorithms: []string{"covisit", "content", "ease", "als", "usercf", "itemcf"},
//	}
type RecommendConfig struct {
	// Enabled controls whether recommendation engine is active.
	// IMPORTANT: Disabled by default due to computational requirements.
	Enabled bool `koanf:"enabled"`

	// TrainInterval is how often to retrain models.
	// Default: 24h (once daily)
	TrainInterval time.Duration `koanf:"train_interval"`

	// TrainOnStartup triggers model training on application startup.
	// Useful for deployments with pre-seeded data.
	// Default: false (wait for scheduled training)
	TrainOnStartup bool `koanf:"train_on_startup"`

	// MinInteractions is the minimum interactions required before training.
	// Below this threshold, only content-based recommendations are available.
	// Default: 100
	MinInteractions int `koanf:"min_interactions"`

	// ModelPath is the directory for persisting trained models.
	// Default: /data/recommend
	ModelPath string `koanf:"model_path"`

	// Algorithms is the list of enabled recommendation algorithms.
	// Available: covisit, content, popularity, ease, als, usercf, itemcf, fpmc, linucb
	// Default: covisit, content (lightweight only)
	Algorithms []string `koanf:"algorithms"`

	// CacheTTL is how long to cache recommendation results.
	// Default: 5m
	CacheTTL time.Duration `koanf:"cache_ttl"`

	// MaxCandidates limits the number of items to score per request.
	// Higher values improve quality but increase latency.
	// Default: 1000
	MaxCandidates int `koanf:"max_candidates"`

	// DiversityLambda controls the relevance vs diversity tradeoff (0-1).
	// 1.0 = pure relevance, 0.0 = maximum diversity
	// Default: 0.7
	DiversityLambda float64 `koanf:"diversity_lambda"`

	// CalibrationEnabled enables calibration reranking to match user preferences.
	// Default: true (when enabled algorithms include calibration-aware ones)
	CalibrationEnabled bool `koanf:"calibration_enabled"`

	// Algorithm-specific configuration
	EASE   EASEAlgorithmConfig   `koanf:"ease"`
	ALS    ALSAlgorithmConfig    `koanf:"als"`
	KNN    KNNAlgorithmConfig    `koanf:"knn"`
	FPMC   FPMCAlgorithmConfig   `koanf:"fpmc"`
	LinUCB LinUCBAlgorithmConfig `koanf:"linucb"`
}

// EASEAlgorithmConfig holds EASE (Embarrassingly Shallow Autoencoders) settings.
type EASEAlgorithmConfig struct {
	// L2Regularization controls model complexity.
	// Higher values = simpler model, less overfitting.
	// Default: 500.0
	L2Regularization float64 `koanf:"l2_regularization"`

	// MinConfidence filters weak signals from the interaction matrix.
	// Default: 0.1
	MinConfidence float64 `koanf:"min_confidence"`

	// UseParallel enables parallel matrix operations for large datasets.
	// Default: true
	UseParallel bool `koanf:"use_parallel"`
}

// ALSAlgorithmConfig holds ALS (Alternating Least Squares) settings.
type ALSAlgorithmConfig struct {
	// Factors is the number of latent factors.
	// More factors capture more complex patterns but require more compute.
	// Default: 50
	Factors int `koanf:"factors"`

	// Iterations is the number of optimization iterations.
	// More iterations improve convergence but take longer.
	// Default: 15
	Iterations int `koanf:"iterations"`

	// Regularization controls overfitting.
	// Default: 0.1
	Regularization float64 `koanf:"regularization"`

	// Alpha is the confidence scaling factor for implicit feedback.
	// Default: 40.0
	Alpha float64 `koanf:"alpha"`

	// NumWorkers is the number of parallel workers for training.
	// Default: 0 (use runtime.NumCPU())
	NumWorkers int `koanf:"num_workers"`
}

// KNNAlgorithmConfig holds User-Based and Item-Based CF settings.
type KNNAlgorithmConfig struct {
	// Neighbors is the number of similar users/items to consider.
	// Default: 50
	Neighbors int `koanf:"neighbors"`

	// Similarity is the similarity metric: cosine, pearson, jaccard
	// Default: cosine
	Similarity string `koanf:"similarity"`

	// Shrinkage penalizes pairs with few co-ratings.
	// Default: 100.0
	Shrinkage float64 `koanf:"shrinkage"`

	// MinCommonItems filters pairs with too few common interactions.
	// Default: 3
	MinCommonItems int `koanf:"min_common_items"`
}

// FPMCAlgorithmConfig holds FPMC (Factorized Personalized Markov Chains) settings.
type FPMCAlgorithmConfig struct {
	// Factors is the number of latent factors.
	// Default: 32
	Factors int `koanf:"factors"`

	// LearningRate for SGD optimization.
	// Default: 0.01
	LearningRate float64 `koanf:"learning_rate"`

	// Regularization controls overfitting.
	// Default: 0.01
	Regularization float64 `koanf:"regularization"`

	// Epochs is the number of training epochs.
	// Default: 20
	Epochs int `koanf:"epochs"`

	// NegativeSamples is the number of negative samples per positive.
	// Default: 5
	NegativeSamples int `koanf:"negative_samples"`
}

// LinUCBAlgorithmConfig holds LinUCB contextual bandit settings.
type LinUCBAlgorithmConfig struct {
	// Alpha controls exploration vs exploitation.
	// Higher values = more exploration of uncertain items.
	// Default: 1.0
	Alpha float64 `koanf:"alpha"`

	// NumFeatures is the context vector dimension.
	// Default: 32
	NumFeatures int `koanf:"num_features"`

	// DecayRate controls how quickly old observations lose influence.
	// 0 = no decay, 1 = maximum decay
	// Default: 0.0
	DecayRate float64 `koanf:"decay_rate"`
}

// GeoIPConfig holds configuration for standalone geolocation lookup.
// When Tautulli is not available, Cartographus can use external GeoIP services
// to resolve IP addresses to geographic locations.
//
// Provider Priority (first available wins):
//  1. MaxMind GeoLite2 (if credentials configured) - same service Tautulli uses
//  2. ip-api.com (free, no API key required, 45 req/min limit)
//
// Environment Variables:
//   - GEOIP_PROVIDER: Preferred provider ("maxmind" or "ipapi", default: auto-detect)
//   - MAXMIND_ACCOUNT_ID: MaxMind account ID (from https://www.maxmind.com/en/account)
//   - MAXMIND_LICENSE_KEY: MaxMind license key (same as Tautulli uses)
//
// If you already use Tautulli, you likely have MaxMind credentials configured there.
// Check Tautulli Settings > General > GeoIP Provider for your existing credentials.
type GeoIPConfig struct {
	// Provider specifies the preferred GeoIP provider.
	// Options: "maxmind", "ipapi", "" (auto-detect based on available credentials)
	Provider string `koanf:"provider"`

	// MaxMind GeoLite2 credentials (same as Tautulli uses)
	// Register free at: https://www.maxmind.com/en/geolite2/signup
	MaxMindAccountID  string `koanf:"maxmind_account_id"`
	MaxMindLicenseKey string `koanf:"maxmind_license_key"`
}

// NewsletterConfig holds configuration for the newsletter scheduler service.
// The scheduler automatically sends newsletters based on cron schedules.
//
// Environment Variables:
//   - NEWSLETTER_ENABLED: Enable newsletter scheduler (default: false)
//   - NEWSLETTER_CHECK_INTERVAL: How often to check for due schedules (default: 1m)
//   - NEWSLETTER_MAX_CONCURRENT: Maximum concurrent deliveries (default: 5)
//   - NEWSLETTER_EXECUTION_TIMEOUT: Max time for single newsletter execution (default: 5m)
//
// Example - Enable newsletter scheduler:
//
//	cfg := NewsletterConfig{
//	    Enabled:       true,
//	    CheckInterval: time.Minute,
//	}
type NewsletterConfig struct {
	// Enabled controls whether the newsletter scheduler is active.
	// Default: false (disabled)
	Enabled bool `koanf:"enabled"`

	// CheckInterval is how often to check for due schedules.
	// Default: 1 minute
	CheckInterval time.Duration `koanf:"check_interval"`

	// MaxConcurrentDeliveries is the maximum number of newsletters to deliver concurrently.
	// Higher values increase throughput but may strain external delivery services.
	// Default: 5
	MaxConcurrentDeliveries int `koanf:"max_concurrent"`

	// ExecutionTimeout is the maximum time allowed for a single newsletter execution.
	// Includes content resolution, rendering, and delivery across all channels.
	// Default: 5 minutes
	ExecutionTimeout time.Duration `koanf:"execution_timeout"`
}

// ========================================
// Multi-Server Helper Methods (v2.1)
// ========================================

// GetPlexServers returns the effective list of Plex servers to use.
// If PlexServers array is configured (non-empty), it takes precedence.
// Otherwise, returns the single Plex config (if enabled) as a single-element slice.
// Returns empty slice if no Plex servers are configured.
func (c *Config) GetPlexServers() []PlexConfig {
	// Prefer array config if configured
	if len(c.PlexServers) > 0 {
		// Filter to only enabled servers
		var enabled []PlexConfig
		for i := range c.PlexServers {
			if c.PlexServers[i].Enabled {
				srv := c.PlexServers[i]
				// Auto-generate ServerID if not set
				if srv.ServerID == "" {
					srv.ServerID = generateServerID("plex", srv.URL)
				}
				enabled = append(enabled, srv)
			}
		}
		return enabled
	}

	// Fall back to single config
	if c.Plex.Enabled {
		cfg := c.Plex
		// Auto-generate ServerID if not set
		if cfg.ServerID == "" {
			cfg.ServerID = generateServerID("plex", cfg.URL)
		}
		return []PlexConfig{cfg}
	}

	return nil
}

// GetJellyfinServers returns the effective list of Jellyfin servers to use.
// If JellyfinServers array is configured (non-empty), it takes precedence.
// Otherwise, returns the single Jellyfin config (if enabled) as a single-element slice.
// Returns empty slice if no Jellyfin servers are configured.
func (c *Config) GetJellyfinServers() []JellyfinConfig {
	// Prefer array config if configured
	if len(c.JellyfinServers) > 0 {
		// Filter to only enabled servers
		var enabled []JellyfinConfig
		for i := range c.JellyfinServers {
			if c.JellyfinServers[i].Enabled {
				srv := c.JellyfinServers[i]
				// Auto-generate ServerID if not set
				if srv.ServerID == "" {
					srv.ServerID = generateServerID("jellyfin", srv.URL)
				}
				enabled = append(enabled, srv)
			}
		}
		return enabled
	}

	// Fall back to single config
	if c.Jellyfin.Enabled {
		cfg := c.Jellyfin
		// Auto-generate ServerID if not set
		if cfg.ServerID == "" {
			cfg.ServerID = generateServerID("jellyfin", cfg.URL)
		}
		return []JellyfinConfig{cfg}
	}

	return nil
}

// GetEmbyServers returns the effective list of Emby servers to use.
// If EmbyServers array is configured (non-empty), it takes precedence.
// Otherwise, returns the single Emby config (if enabled) as a single-element slice.
// Returns empty slice if no Emby servers are configured.
func (c *Config) GetEmbyServers() []EmbyConfig {
	// Prefer array config if configured
	if len(c.EmbyServers) > 0 {
		// Filter to only enabled servers
		var enabled []EmbyConfig
		for i := range c.EmbyServers {
			if c.EmbyServers[i].Enabled {
				srv := c.EmbyServers[i]
				// Auto-generate ServerID if not set
				if srv.ServerID == "" {
					srv.ServerID = generateServerID("emby", srv.URL)
				}
				enabled = append(enabled, srv)
			}
		}
		return enabled
	}

	// Fall back to single config
	if c.Emby.Enabled {
		cfg := c.Emby
		// Auto-generate ServerID if not set
		if cfg.ServerID == "" {
			cfg.ServerID = generateServerID("emby", cfg.URL)
		}
		return []EmbyConfig{cfg}
	}

	return nil
}

// generateServerID creates a deterministic server ID from platform and URL.
// This ensures the same URL always generates the same ID for consistency.
// Format: {platform}-{hash} where hash is first 8 chars of URL hash.
func generateServerID(platform, url string) string {
	if url == "" {
		return platform + "-default"
	}

	// Simple hash of URL for deterministic ID
	hash := uint32(0)
	for _, c := range url {
		hash = hash*31 + uint32(c)
	}

	return fmt.Sprintf("%s-%08x", platform, hash)
}

// GenerateServerID creates a deterministic server ID from platform and URL.
// Exported wrapper for generateServerID for use by other packages.
func GenerateServerID(platform, url string) string {
	return generateServerID(platform, url)
}

// HasAnyMediaServer returns true if at least one media server is configured.
// Used for validation to ensure Cartographus has at least one data source.
func (c *Config) HasAnyMediaServer() bool {
	return len(c.GetPlexServers()) > 0 ||
		len(c.GetJellyfinServers()) > 0 ||
		len(c.GetEmbyServers()) > 0 ||
		c.Tautulli.Enabled
}

// GetTotalServerCount returns the total number of configured media servers.
// Useful for logging and metrics.
func (c *Config) GetTotalServerCount() int {
	return len(c.GetPlexServers()) + len(c.GetJellyfinServers()) + len(c.GetEmbyServers())
}

// Load reads configuration from environment variables and optional config file.
// Configuration is loaded in the following order (later sources override earlier ones):
//  1. Built-in defaults
//  2. Config file (config.yaml if exists, or path specified in CONFIG_PATH env var)
//  3. Environment variables
//
// This function uses Koanf v2 for flexible, layered configuration management.
// For backward compatibility, all existing environment variables continue to work.
//
// See LoadWithKoanf() for the underlying implementation.
func Load() (*Config, error) {
	return LoadWithKoanf()
}

// LoadLegacy reads configuration directly from environment variables only.
// This is the legacy loading method preserved for testing and backward compatibility.
// For production use, prefer Load() which supports config files and layered loading.
//
// Deprecated: Use Load() instead for new code.
func LoadLegacy() (*Config, error) {
	cfg := &Config{
		Tautulli: TautulliConfig{
			Enabled:  getBoolEnv("TAUTULLI_ENABLED", false),
			URL:      getEnv("TAUTULLI_URL", ""),
			APIKey:   getEnv("TAUTULLI_API_KEY", ""),
			ServerID: getEnv("TAUTULLI_SERVER_ID", ""),
		},
		Plex: PlexConfig{
			Enabled:         getBoolEnv("ENABLE_PLEX_SYNC", false),
			ServerID:        getEnv("PLEX_SERVER_ID", ""),
			URL:             getEnv("PLEX_URL", ""),
			Token:           getEnv("PLEX_TOKEN", ""),
			HistoricalSync:  getBoolEnv("PLEX_HISTORICAL_SYNC", false),
			SyncDaysBack:    getIntEnv("PLEX_SYNC_DAYS_BACK", 365),
			SyncInterval:    getDurationEnv("PLEX_SYNC_INTERVAL", 24*time.Hour),
			RealtimeEnabled: getBoolEnv("ENABLE_PLEX_REALTIME", false),

			// OAuth 2.0 PKCE Authentication (Sprint 1, Task 1.1)
			OAuthClientID:     getEnv("PLEX_OAUTH_CLIENT_ID", ""),
			OAuthClientSecret: getEnv("PLEX_OAUTH_CLIENT_SECRET", ""),
			OAuthRedirectURI:  getEnv("PLEX_OAUTH_REDIRECT_URI", ""),

			// Active Transcode Monitoring (Phase 1.2: v1.40)
			TranscodeMonitoring:         getBoolEnv("ENABLE_PLEX_TRANSCODE_MONITORING", false),
			TranscodeMonitoringInterval: getDurationEnv("PLEX_TRANSCODE_MONITORING_INTERVAL", 10*time.Second),

			// Buffer Health Monitoring (Phase 2.1: v1.41)
			BufferHealthMonitoring:        getBoolEnv("ENABLE_BUFFER_HEALTH_MONITORING", false),
			BufferHealthPollInterval:      getDurationEnv("BUFFER_HEALTH_POLL_INTERVAL", 5*time.Second),
			BufferHealthCriticalThreshold: getFloatEnv("BUFFER_HEALTH_CRITICAL_THRESHOLD", 20.0),
			BufferHealthRiskyThreshold:    getFloatEnv("BUFFER_HEALTH_RISKY_THRESHOLD", 50.0),

			// Webhook Receiver (Sprint 1, Task 1.3: v1.43)
			WebhooksEnabled: getBoolEnv("ENABLE_PLEX_WEBHOOKS", false),
			WebhookSecret:   getEnv("PLEX_WEBHOOK_SECRET", ""),
		},
		// Jellyfin direct integration (v1.51)
		Jellyfin: JellyfinConfig{
			Enabled:                getBoolEnv("JELLYFIN_ENABLED", false),
			ServerID:               getEnv("JELLYFIN_SERVER_ID", ""),
			URL:                    getEnv("JELLYFIN_URL", ""),
			APIKey:                 getEnv("JELLYFIN_API_KEY", ""),
			UserID:                 getEnv("JELLYFIN_USER_ID", ""),
			RealtimeEnabled:        getBoolEnv("JELLYFIN_REALTIME_ENABLED", false),
			SessionPollingEnabled:  getBoolEnv("JELLYFIN_SESSION_POLLING_ENABLED", false),
			SessionPollingInterval: getDurationEnv("JELLYFIN_SESSION_POLLING_INTERVAL", 30*time.Second),
			WebhooksEnabled:        getBoolEnv("JELLYFIN_WEBHOOKS_ENABLED", false),
			WebhookSecret:          getEnv("JELLYFIN_WEBHOOK_SECRET", ""),
		},
		// Emby direct integration (v1.51)
		Emby: EmbyConfig{
			Enabled:                getBoolEnv("EMBY_ENABLED", false),
			ServerID:               getEnv("EMBY_SERVER_ID", ""),
			URL:                    getEnv("EMBY_URL", ""),
			APIKey:                 getEnv("EMBY_API_KEY", ""),
			UserID:                 getEnv("EMBY_USER_ID", ""),
			RealtimeEnabled:        getBoolEnv("EMBY_REALTIME_ENABLED", false),
			SessionPollingEnabled:  getBoolEnv("EMBY_SESSION_POLLING_ENABLED", false),
			SessionPollingInterval: getDurationEnv("EMBY_SESSION_POLLING_INTERVAL", 30*time.Second),
			WebhooksEnabled:        getBoolEnv("EMBY_WEBHOOKS_ENABLED", false),
			WebhookSecret:          getEnv("EMBY_WEBHOOK_SECRET", ""),
		},
		NATS: NATSConfig{
			Enabled:             getBoolEnv("NATS_ENABLED", true),
			EventSourcing:       getBoolEnv("NATS_EVENT_SOURCING", true),
			URL:                 getEnv("NATS_URL", "nats://127.0.0.1:4222"),
			EmbeddedServer:      getBoolEnv("NATS_EMBEDDED", true),
			StoreDir:            getEnv("NATS_STORE_DIR", "/data/nats/jetstream"),
			MaxMemory:           getInt64Env("NATS_MAX_MEMORY", 1<<30), // 1GB default
			MaxStore:            getInt64Env("NATS_MAX_STORE", 10<<30), // 10GB default
			StreamRetentionDays: getIntEnv("NATS_RETENTION_DAYS", 7),
			BatchSize:           getIntEnv("NATS_BATCH_SIZE", 1000),
			FlushInterval:       getDurationEnv("NATS_FLUSH_INTERVAL", 5*time.Second),
			SubscribersCount:    getIntEnv("NATS_SUBSCRIBERS", 4),
			DurableName:         getEnv("NATS_DURABLE_NAME", "media-processor"),
			QueueGroup:          getEnv("NATS_QUEUE_GROUP", "processors"),
			// Router configuration defaults
			RouterRetryCount:           getIntEnv("NATS_ROUTER_RETRY_COUNT", 3),
			RouterRetryInitialInterval: getDurationEnv("NATS_ROUTER_RETRY_INTERVAL", 100*time.Millisecond),
			RouterThrottlePerSecond:    getIntEnv("NATS_ROUTER_THROTTLE", 0),
			// IMPORTANT: Router dedup is DISABLED by default because:
			// 1. DuckDBHandler has comprehensive application-level deduplication (EventID, SessionKey,
			//    CorrelationKey, CrossSourceKey) that is context-aware and supports cross-source dedup
			// 2. JetStream stream already has a 2-minute DuplicateWindow for exact message ID dedup
			// 3. Watermill may regenerate message UUIDs on receive, causing Router dedup to use
			//    different keys than the publisher intended
			// Enable Router dedup only if you need simple message-ID deduplication without
			// the context-aware application-level dedup.
			RouterDeduplicationEnabled: getBoolEnv("NATS_ROUTER_DEDUP_ENABLED", false),
			RouterDeduplicationTTL:     getDurationEnv("NATS_ROUTER_DEDUP_TTL", 5*time.Minute),
			RouterPoisonQueueEnabled:   getBoolEnv("NATS_ROUTER_POISON_ENABLED", true),
			RouterPoisonQueueTopic:     getEnv("NATS_ROUTER_POISON_TOPIC", "playback.poison"),
			RouterCloseTimeout:         getDurationEnv("NATS_ROUTER_CLOSE_TIMEOUT", 30*time.Second),
		},
		Import: ImportConfig{
			Enabled:         getBoolEnv("IMPORT_ENABLED", false),
			DBPath:          getEnv("IMPORT_DB_PATH", ""),
			BatchSize:       getIntEnv("IMPORT_BATCH_SIZE", 1000),
			DryRun:          getBoolEnv("IMPORT_DRY_RUN", false),
			AutoStart:       getBoolEnv("IMPORT_AUTO_START", false),
			ResumeFromID:    getInt64Env("IMPORT_RESUME_FROM_ID", 0),
			SkipGeolocation: getBoolEnv("IMPORT_SKIP_GEOLOCATION", false),
		},
		Database: DatabaseConfig{
			Path:                   getEnv("DUCKDB_PATH", "/data/cartographus.duckdb"),
			MaxMemory:              getEnv("DUCKDB_MAX_MEMORY", "2GB"),
			Threads:                getIntEnv("DUCKDB_THREADS", 0), // 0 means use runtime.NumCPU()
			PreserveInsertionOrder: getBoolEnv("DUCKDB_PRESERVE_INSERTION_ORDER", true),
			SeedMockData:           getBoolEnv("SEED_MOCK_DATA", false),
		},
		Sync: SyncConfig{
			Interval:      getDurationEnv("SYNC_INTERVAL", 5*time.Minute),
			Lookback:      getDurationEnv("SYNC_LOOKBACK", 24*time.Hour),
			SyncAll:       getBoolEnv("SYNC_ALL", false),
			BatchSize:     getIntEnv("SYNC_BATCH_SIZE", 1000),
			RetryAttempts: getIntEnv("SYNC_RETRY_ATTEMPTS", 5),
			RetryDelay:    getDurationEnv("SYNC_RETRY_DELAY", 2*time.Second),
		},
		Server: ServerConfig{
			Port:      getIntEnv("HTTP_PORT", 3857),
			Host:      getEnv("HTTP_HOST", "0.0.0.0"),
			Timeout:   getDurationEnv("HTTP_TIMEOUT", 30*time.Second),
			Latitude:  getFloatEnv("SERVER_LATITUDE", 0.0),
			Longitude: getFloatEnv("SERVER_LONGITUDE", 0.0),
		},
		API: APIConfig{
			DefaultPageSize: getIntEnv("API_DEFAULT_PAGE_SIZE", 20),
			MaxPageSize:     getIntEnv("API_MAX_PAGE_SIZE", 100),
		},
		Security: SecurityConfig{
			AuthMode:             getEnv("AUTH_MODE", "jwt"),
			JWTSecret:            getEnv("JWT_SECRET", ""),
			SessionTimeout:       getDurationEnv("SESSION_TIMEOUT", 24*time.Hour),
			AdminUsername:        getEnv("ADMIN_USERNAME", ""),
			AdminPassword:        getEnv("ADMIN_PASSWORD", ""),
			BasicAuthDefaultRole: getEnv("BASIC_AUTH_DEFAULT_ROLE", "viewer"),
			RateLimitReqs:        getIntEnv("RATE_LIMIT_REQUESTS", 100),
			RateLimitWindow:      getDurationEnv("RATE_LIMIT_WINDOW", 1*time.Minute),
			RateLimitDisabled:    getBoolEnv("DISABLE_RATE_LIMIT", false),
			CORSOrigins:          getSliceEnv("CORS_ORIGINS", []string{"*"}),
			TrustedProxies:       getSliceEnv("TRUSTED_PROXIES", []string{}),

			// Zero Trust Authentication & Authorization (ADR-0015)
			OIDC: OIDCConfig{
				IssuerURL:      getEnv("OIDC_ISSUER_URL", ""),
				ClientID:       getEnv("OIDC_CLIENT_ID", ""),
				ClientSecret:   getEnv("OIDC_CLIENT_SECRET", ""),
				RedirectURL:    getEnv("OIDC_REDIRECT_URL", ""),
				Scopes:         getSliceEnv("OIDC_SCOPES", []string{"openid", "profile", "email"}),
				PKCEEnabled:    getBoolEnv("OIDC_PKCE_ENABLED", true),
				JWKSCacheTTL:   getDurationEnv("OIDC_JWKS_CACHE_TTL", 1*time.Hour),
				SessionMaxAge:  getDurationEnv("OIDC_SESSION_MAX_AGE", 24*time.Hour),
				SessionSecret:  getEnv("OIDC_SESSION_SECRET", ""),
				CookieName:     getEnv("OIDC_COOKIE_NAME", "tautulli_session"),
				CookieSecure:   getBoolEnv("OIDC_COOKIE_SECURE", true),
				RolesClaim:     getEnv("OIDC_ROLES_CLAIM", "roles"),
				DefaultRoles:   getSliceEnv("OIDC_DEFAULT_ROLES", []string{"viewer"}),
				UsernameClaims: getSliceEnv("OIDC_USERNAME_CLAIMS", []string{"preferred_username", "name", "email"}),
			},
			PlexAuth: PlexAuthConfig{
				ClientID:                getEnv("PLEX_AUTH_CLIENT_ID", ""),
				ClientSecret:            getEnv("PLEX_AUTH_CLIENT_SECRET", ""),
				RedirectURI:             getEnv("PLEX_AUTH_REDIRECT_URI", ""),
				DefaultRoles:            getSliceEnv("PLEX_AUTH_DEFAULT_ROLES", []string{"viewer"}),
				PlexPassRole:            getEnv("PLEX_AUTH_PLEX_PASS_ROLE", ""),
				Timeout:                 getDurationEnv("PLEX_AUTH_TIMEOUT", 30*time.Second),
				ServerOwnerRole:         getEnv("PLEX_AUTH_SERVER_OWNER_ROLE", "admin"),
				ServerAdminRole:         getEnv("PLEX_AUTH_SERVER_ADMIN_ROLE", "editor"),
				EnableServerDetection:   getBoolEnv("PLEX_AUTH_ENABLE_SERVER_DETECTION", true),
				ServerMachineIdentifier: getEnv("PLEX_AUTH_SERVER_MACHINE_ID", ""),
			},
			Casbin: CasbinConfig{
				ModelPath:      getEnv("CASBIN_MODEL_PATH", ""),
				PolicyPath:     getEnv("CASBIN_POLICY_PATH", ""),
				DefaultRole:    getEnv("CASBIN_DEFAULT_ROLE", "viewer"),
				AutoReload:     getBoolEnv("CASBIN_AUTO_RELOAD", true),
				ReloadInterval: getDurationEnv("CASBIN_RELOAD_INTERVAL", 30*time.Second),
				CacheEnabled:   getBoolEnv("CASBIN_CACHE_ENABLED", true),
				CacheTTL:       getDurationEnv("CASBIN_CACHE_TTL", 5*time.Minute),
			},
		},
		Logging: LoggingConfig{
			Level:  getEnv("LOG_LEVEL", "info"),
			Format: getEnv("LOG_FORMAT", "json"),
			Caller: getBoolEnv("LOG_CALLER", false),
		},
		// Detection engine configuration (ADR-0020)
		Detection: DetectionConfig{
			Enabled:             getBoolEnv("DETECTION_ENABLED", true),
			TrustScoreDecrement: getIntEnv("DETECTION_TRUST_SCORE_DECREMENT", 10),
			TrustScoreRecovery:  getIntEnv("DETECTION_TRUST_SCORE_RECOVERY", 1),
			TrustScoreThreshold: getIntEnv("DETECTION_TRUST_SCORE_THRESHOLD", 50),
			Discord: DiscordNotifierConfig{
				WebhookURL:  getEnv("DISCORD_WEBHOOK_URL", ""),
				Enabled:     getBoolEnv("DISCORD_WEBHOOK_ENABLED", false),
				RateLimitMs: getIntEnv("DISCORD_RATE_LIMIT_MS", 1000),
			},
			Webhook: WebhookNotifierConfig{
				WebhookURL:  getEnv("WEBHOOK_URL", ""),
				Enabled:     getBoolEnv("WEBHOOK_ENABLED", false),
				RateLimitMs: getIntEnv("WEBHOOK_RATE_LIMIT_MS", 500),
				Headers:     getMapEnv("WEBHOOK_HEADERS"),
			},
		},
		// Recommendation engine configuration (ADR-0024)
		// IMPORTANT: Disabled by default due to computational requirements
		Recommend: RecommendConfig{
			Enabled:            getBoolEnv("RECOMMEND_ENABLED", false), // Disabled by default
			TrainInterval:      getDurationEnv("RECOMMEND_TRAIN_INTERVAL", 24*time.Hour),
			TrainOnStartup:     getBoolEnv("RECOMMEND_TRAIN_ON_STARTUP", false),
			MinInteractions:    getIntEnv("RECOMMEND_MIN_INTERACTIONS", 100),
			ModelPath:          getEnv("RECOMMEND_MODEL_PATH", "/data/recommend"),
			Algorithms:         getSliceEnv("RECOMMEND_ALGORITHMS", []string{"covisit", "content"}), // Lightweight only
			CacheTTL:           getDurationEnv("RECOMMEND_CACHE_TTL", 5*time.Minute),
			MaxCandidates:      getIntEnv("RECOMMEND_MAX_CANDIDATES", 1000),
			DiversityLambda:    getFloatEnv("RECOMMEND_DIVERSITY_LAMBDA", 0.7),
			CalibrationEnabled: getBoolEnv("RECOMMEND_CALIBRATION_ENABLED", true),
			EASE: EASEAlgorithmConfig{
				L2Regularization: getFloatEnv("RECOMMEND_EASE_REGULARIZATION", 500.0),
				MinConfidence:    getFloatEnv("RECOMMEND_EASE_MIN_CONFIDENCE", 0.1),
				UseParallel:      getBoolEnv("RECOMMEND_EASE_PARALLEL", true),
			},
			ALS: ALSAlgorithmConfig{
				Factors:        getIntEnv("RECOMMEND_ALS_FACTORS", 50),
				Iterations:     getIntEnv("RECOMMEND_ALS_ITERATIONS", 15),
				Regularization: getFloatEnv("RECOMMEND_ALS_REGULARIZATION", 0.1),
				Alpha:          getFloatEnv("RECOMMEND_ALS_ALPHA", 40.0),
				NumWorkers:     getIntEnv("RECOMMEND_ALS_WORKERS", 0),
			},
			KNN: KNNAlgorithmConfig{
				Neighbors:      getIntEnv("RECOMMEND_KNN_NEIGHBORS", 50),
				Similarity:     getEnv("RECOMMEND_KNN_SIMILARITY", "cosine"),
				Shrinkage:      getFloatEnv("RECOMMEND_KNN_SHRINKAGE", 100.0),
				MinCommonItems: getIntEnv("RECOMMEND_KNN_MIN_COMMON", 3),
			},
			FPMC: FPMCAlgorithmConfig{
				Factors:         getIntEnv("RECOMMEND_FPMC_FACTORS", 32),
				LearningRate:    getFloatEnv("RECOMMEND_FPMC_LEARNING_RATE", 0.01),
				Regularization:  getFloatEnv("RECOMMEND_FPMC_REGULARIZATION", 0.01),
				Epochs:          getIntEnv("RECOMMEND_FPMC_EPOCHS", 20),
				NegativeSamples: getIntEnv("RECOMMEND_FPMC_NEGATIVE_SAMPLES", 5),
			},
			LinUCB: LinUCBAlgorithmConfig{
				Alpha:       getFloatEnv("RECOMMEND_LINUCB_ALPHA", 1.0),
				NumFeatures: getIntEnv("RECOMMEND_LINUCB_FEATURES", 32),
				DecayRate:   getFloatEnv("RECOMMEND_LINUCB_DECAY", 0.0),
			},
		},
		// GeoIP configuration for standalone geolocation (v2.0)
		GeoIP: GeoIPConfig{
			Provider:          getEnv("GEOIP_PROVIDER", ""),      // "" = auto-detect
			MaxMindAccountID:  getEnv("MAXMIND_ACCOUNT_ID", ""),  // MaxMind account ID
			MaxMindLicenseKey: getEnv("MAXMIND_LICENSE_KEY", ""), // MaxMind license key
		},
		// VPN detection configuration
		VPN: VPNConfig{
			Enabled:        getBoolEnv("VPN_ENABLED", true), // Enabled by default
			DataFile:       getEnv("VPN_DATA_FILE", ""),
			CacheSize:      getIntEnv("VPN_CACHE_SIZE", 10000),
			AutoUpdate:     getBoolEnv("VPN_AUTO_UPDATE", false),
			UpdateInterval: getDurationEnv("VPN_UPDATE_INTERVAL", 24*time.Hour),
		},
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return cfg, nil
}

// NOTE: Validate() method moved to config_validate.go
// NOTE: URL validation functions moved to config_url.go
// NOTE: Environment variable helpers moved to config_env.go
