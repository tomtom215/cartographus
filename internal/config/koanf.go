// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/structs"
	"github.com/knadh/koanf/v2"
)

// DefaultConfigPaths lists the paths where config files are searched in order of priority.
// The first file found will be used.
var DefaultConfigPaths = []string{
	"config.yaml",
	"config.yml",
	"/etc/cartographus/config.yaml",
	"/etc/cartographus/config.yml",
}

// ConfigPathEnvVar is the environment variable that can override the config file path.
const ConfigPathEnvVar = "CONFIG_PATH"

// defaultConfig returns a Config struct with all sensible default values.
// These defaults are applied first, then overridden by config file and env vars.
func defaultConfig() *Config {
	return &Config{
		Tautulli: TautulliConfig{
			Enabled:  false, // Tautulli is optional - standalone mode by default
			URL:      "",
			APIKey:   "",
			ServerID: "", // Auto-generated if empty (for multi-server support)
		},
		Plex: PlexConfig{
			Enabled:                       false,
			ServerID:                      "", // Auto-generated if empty (for multi-server support)
			URL:                           "",
			Token:                         "",
			HistoricalSync:                false,
			SyncDaysBack:                  365,
			SyncInterval:                  24 * time.Hour,
			RealtimeEnabled:               false,
			OAuthClientID:                 "",
			OAuthClientSecret:             "",
			OAuthRedirectURI:              "",
			TranscodeMonitoring:           false,
			TranscodeMonitoringInterval:   10 * time.Second,
			BufferHealthMonitoring:        false,
			BufferHealthPollInterval:      5 * time.Second,
			BufferHealthCriticalThreshold: 20.0,
			BufferHealthRiskyThreshold:    50.0,
			WebhooksEnabled:               false,
			WebhookSecret:                 "",
			SessionPollingEnabled:         false,            // Disabled by default - WebSocket is the primary mechanism
			SessionPollingInterval:        30 * time.Second, // Conservative default to minimize Plex API load
		},
		// Jellyfin direct integration (v1.51)
		Jellyfin: JellyfinConfig{
			Enabled:                false,
			ServerID:               "", // Auto-generated if empty (for multi-server support)
			URL:                    "",
			APIKey:                 "",
			UserID:                 "",
			RealtimeEnabled:        false,
			SessionPollingEnabled:  false,
			SessionPollingInterval: 30 * time.Second,
			WebhooksEnabled:        false,
			WebhookSecret:          "",
		},
		// Emby direct integration (v1.51)
		Emby: EmbyConfig{
			Enabled:                false,
			ServerID:               "", // Auto-generated if empty (for multi-server support)
			URL:                    "",
			APIKey:                 "",
			UserID:                 "",
			RealtimeEnabled:        false,
			SessionPollingEnabled:  false,
			SessionPollingInterval: 30 * time.Second,
			WebhooksEnabled:        false,
			WebhookSecret:          "",
		},
		NATS: NATSConfig{
			Enabled:             true,
			EventSourcing:       true,
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
			// Router defaults (Watermill Router middleware)
			RouterRetryCount:           3,
			RouterRetryInitialInterval: 100 * time.Millisecond,
			RouterThrottlePerSecond:    0,     // Unlimited
			RouterDeduplicationEnabled: false, // Handler has comprehensive dedup
			RouterDeduplicationTTL:     5 * time.Minute,
			RouterPoisonQueueEnabled:   true,
			RouterPoisonQueueTopic:     "playback.poison",
			RouterCloseTimeout:         30 * time.Second,
		},
		Database: DatabaseConfig{
			Path:                   "/data/cartographus.duckdb",
			MaxMemory:              "2GB",
			Threads:                0,    // 0 = use runtime.NumCPU()
			PreserveInsertionOrder: true, // DuckDB default
			SeedMockData:           false,
		},
		Sync: SyncConfig{
			Interval:      5 * time.Minute,
			Lookback:      24 * time.Hour,
			SyncAll:       false,
			BatchSize:     1000,
			RetryAttempts: 5,
			RetryDelay:    2 * time.Second,
		},
		Server: ServerConfig{
			Port:        3857,
			Host:        "0.0.0.0",
			Timeout:     30 * time.Second,
			Latitude:    0.0,
			Longitude:   0.0,
			Environment: "development", // Default to development; set ENVIRONMENT=production for production checks
		},
		API: APIConfig{
			DefaultPageSize: 20,
			MaxPageSize:     100,
		},
		Security: SecurityConfig{
			AuthMode:          "jwt",
			JWTSecret:         "",
			SessionTimeout:    24 * time.Hour,
			AdminUsername:     "",
			AdminPassword:     "",
			RateLimitReqs:     100,
			RateLimitWindow:   1 * time.Minute,
			RateLimitDisabled: false,
			CORSOrigins:       []string{"*"},
			TrustedProxies:    []string{},

			// Session Store Configuration (ADR-0015 Phase 4)
			// Default to persistent storage for production-grade UX (sessions survive restarts)
			SessionStore:     "badger",
			SessionStorePath: "/data/sessions",

			// Zero Trust Authentication & Authorization (ADR-0015)
			OIDC: OIDCConfig{
				IssuerURL:             "",
				ClientID:              "",
				ClientSecret:          "",
				RedirectURL:           "",
				PostLogoutRedirectURI: "/",
				Scopes:                []string{"openid", "profile", "email"},
				PKCEEnabled:           true,
				JWKSCacheTTL:          1 * time.Hour,
				SessionMaxAge:         24 * time.Hour,
				SessionSecret:         "",
				CookieName:            "tautulli_session",
				CookieSecure:          true,
				RolesClaim:            "roles",
				DefaultRoles:          []string{"viewer"},
				UsernameClaims:        []string{"preferred_username", "name", "email"},
			},
			PlexAuth: PlexAuthConfig{
				ClientID:                "",
				ClientSecret:            "",
				RedirectURI:             "",
				DefaultRoles:            []string{"viewer"},
				PlexPassRole:            "",
				Timeout:                 30 * time.Second,
				ServerOwnerRole:         "admin",
				ServerAdminRole:         "editor",
				EnableServerDetection:   true,
				ServerMachineIdentifier: "",
			},
			Casbin: CasbinConfig{
				ModelPath:      "",
				PolicyPath:     "",
				DefaultRole:    "viewer",
				AutoReload:     true,
				ReloadInterval: 30 * time.Second,
				CacheEnabled:   true,
				CacheTTL:       5 * time.Minute,
			},
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "json",
			Caller: false,
		},
		// Recommendation engine configuration (ADR-0024)
		// IMPORTANT: Disabled by default due to computational requirements
		Recommend: RecommendConfig{
			Enabled:            false, // Disabled by default - opt-in only
			TrainInterval:      24 * time.Hour,
			TrainOnStartup:     false,
			MinInteractions:    100,
			ModelPath:          "/data/recommend",
			Algorithms:         []string{"covisit", "content"}, // Lightweight only by default
			CacheTTL:           5 * time.Minute,
			MaxCandidates:      1000,
			DiversityLambda:    0.7,
			CalibrationEnabled: true,
			EASE: EASEAlgorithmConfig{
				L2Regularization: 500.0,
				MinConfidence:    0.1,
				UseParallel:      true,
			},
			ALS: ALSAlgorithmConfig{
				Factors:        50,
				Iterations:     15,
				Regularization: 0.1,
				Alpha:          40.0,
				NumWorkers:     0, // 0 = use runtime.NumCPU()
			},
			KNN: KNNAlgorithmConfig{
				Neighbors:      50,
				Similarity:     "cosine",
				Shrinkage:      100.0,
				MinCommonItems: 3,
			},
			FPMC: FPMCAlgorithmConfig{
				Factors:         32,
				LearningRate:    0.01,
				Regularization:  0.01,
				Epochs:          20,
				NegativeSamples: 5,
			},
			LinUCB: LinUCBAlgorithmConfig{
				Alpha:       1.0,
				NumFeatures: 32,
				DecayRate:   0.0,
			},
		},
		// Newsletter scheduler configuration
		Newsletter: NewsletterConfig{
			Enabled:                 false,           // Disabled by default - opt-in only
			CheckInterval:           time.Minute,     // How often to check for due schedules
			MaxConcurrentDeliveries: 5,               // Max newsletters to deliver concurrently
			ExecutionTimeout:        5 * time.Minute, // Max time for a single newsletter execution
		},
	}
}

// LoadWithKoanf loads configuration using Koanf v2 with layered sources:
//  1. Defaults: Built-in sensible defaults
//  2. Config File: Optional YAML config file (if exists)
//  3. Environment Variables: Override any setting
//
// This function is the preferred way to load configuration and provides:
//   - Type-safe configuration unmarshaling
//   - Clear precedence: ENV > File > Defaults
//   - Support for nested configuration via koanf struct tags
//   - Backward compatibility with existing environment variables
func LoadWithKoanf() (*Config, error) {
	k := koanf.New(".")

	// Layer 1: Load defaults from struct
	defaults := defaultConfig()
	if err := k.Load(structs.Provider(defaults, "koanf"), nil); err != nil {
		return nil, fmt.Errorf("failed to load defaults: %w", err)
	}

	// Layer 2: Load config file (optional)
	configPath := findConfigFile()
	if configPath != "" {
		if err := k.Load(file.Provider(configPath), yaml.Parser()); err != nil {
			return nil, fmt.Errorf("failed to load config file %s: %w", configPath, err)
		}
	}

	// Layer 3: Load environment variables (highest priority)
	// Transform environment variable names to koanf paths:
	// TAUTULLI_URL -> tautulli.url
	// PLEX_SYNC_DAYS_BACK -> plex.sync_days_back
	envProvider := env.Provider("", ".", envTransformFunc)
	if err := k.Load(envProvider, nil); err != nil {
		return nil, fmt.Errorf("failed to load environment variables: %w", err)
	}

	// Post-process slice fields from comma-separated strings
	if err := processSliceFields(k); err != nil {
		return nil, fmt.Errorf("failed to process slice fields: %w", err)
	}

	// Unmarshal into Config struct
	cfg := &Config{}
	if err := k.Unmarshal("", cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal configuration: %w", err)
	}

	// Validate the configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return cfg, nil
}

// findConfigFile searches for a config file in the default paths.
// Returns the path to the first file found, or empty string if none found.
func findConfigFile() string {
	// Check environment variable first
	if envPath := os.Getenv(ConfigPathEnvVar); envPath != "" {
		if _, err := os.Stat(envPath); err == nil {
			return envPath
		}
	}

	// Search default paths
	for _, path := range DefaultConfigPaths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
}

// sliceConfigPaths defines which config paths should be parsed as comma-separated slices
var sliceConfigPaths = []string{
	"security.cors_origins",
	"security.trusted_proxies",
	// Zero Trust Authentication & Authorization (ADR-0015)
	"security.oidc.scopes",
	"security.oidc.default_roles",
	"security.oidc.username_claims",
	"security.plex_auth.default_roles",
	// Recommendation engine (ADR-0024)
	"recommend.algorithms",
}

// processSliceFields converts comma-separated string values to slices for known slice fields.
// This is necessary because env vars come in as strings, but the config expects slices.
func processSliceFields(k *koanf.Koanf) error {
	for _, path := range sliceConfigPaths {
		val := k.Get(path)
		if val == nil {
			continue
		}

		// If it's already a slice (from YAML file), skip
		if _, ok := val.([]interface{}); ok {
			continue
		}
		if _, ok := val.([]string); ok {
			continue
		}

		// If it's a string, split by comma
		if strVal, ok := val.(string); ok {
			if strVal == "" {
				continue
			}
			parts := strings.Split(strVal, ",")
			trimmed := make([]string, 0, len(parts))
			for _, p := range parts {
				p = strings.TrimSpace(p)
				if p != "" {
					trimmed = append(trimmed, p)
				}
			}
			if len(trimmed) > 0 {
				if err := k.Set(path, trimmed); err != nil {
					return fmt.Errorf("failed to set %s: %w", path, err)
				}
			}
		}
	}
	return nil
}

// envTransformFunc transforms environment variable names to koanf config paths.
// It handles the mapping from legacy environment variable names to the new
// nested configuration structure.
//
// Examples:
//   - TAUTULLI_URL -> tautulli.url
//   - TAUTULLI_API_KEY -> tautulli.api_key
//   - PLEX_SYNC_DAYS_BACK -> plex.sync_days_back
//   - ENABLE_PLEX_SYNC -> plex.enabled
//   - DUCKDB_PATH -> database.path
//   - HTTP_PORT -> server.port
func envTransformFunc(key string) string {
	key = strings.ToLower(key)

	// Map legacy environment variable prefixes to config sections
	envMappings := map[string]string{
		// Tautulli mappings (optional data source as of v2.0)
		"tautulli_enabled":   "tautulli.enabled",
		"tautulli_url":       "tautulli.url",
		"tautulli_api_key":   "tautulli.api_key",
		"tautulli_server_id": "tautulli.server_id",

		// Plex mappings (handle ENABLE_ prefix)
		"enable_plex_sync":                   "plex.enabled",
		"plex_server_id":                     "plex.server_id",
		"plex_url":                           "plex.url",
		"plex_token":                         "plex.token",
		"plex_historical_sync":               "plex.historical_sync",
		"plex_sync_days_back":                "plex.sync_days_back",
		"plex_sync_interval":                 "plex.sync_interval",
		"enable_plex_realtime":               "plex.realtime_enabled",
		"plex_oauth_client_id":               "plex.oauth_client_id",
		"plex_oauth_client_secret":           "plex.oauth_client_secret",
		"plex_oauth_redirect_uri":            "plex.oauth_redirect_uri",
		"enable_plex_transcode_monitoring":   "plex.transcode_monitoring",
		"plex_transcode_monitoring_interval": "plex.transcode_monitoring_interval",
		"enable_buffer_health_monitoring":    "plex.buffer_health_monitoring",
		"buffer_health_poll_interval":        "plex.buffer_health_poll_interval",
		"buffer_health_critical_threshold":   "plex.buffer_health_critical_threshold",
		"buffer_health_risky_threshold":      "plex.buffer_health_risky_threshold",
		"enable_plex_webhooks":               "plex.webhooks_enabled",
		"plex_webhook_secret":                "plex.webhook_secret",
		"plex_session_polling_enabled":       "plex.session_polling_enabled",
		"plex_session_polling_interval":      "plex.session_polling_interval",

		// Jellyfin mappings (v1.51)
		"jellyfin_enabled":                  "jellyfin.enabled",
		"jellyfin_server_id":                "jellyfin.server_id",
		"jellyfin_url":                      "jellyfin.url",
		"jellyfin_api_key":                  "jellyfin.api_key",
		"jellyfin_user_id":                  "jellyfin.user_id",
		"jellyfin_realtime_enabled":         "jellyfin.realtime_enabled",
		"jellyfin_session_polling_enabled":  "jellyfin.session_polling_enabled",
		"jellyfin_session_polling_interval": "jellyfin.session_polling_interval",
		"jellyfin_webhooks_enabled":         "jellyfin.webhooks_enabled",
		"jellyfin_webhook_secret":           "jellyfin.webhook_secret",

		// Emby mappings (v1.51)
		"emby_enabled":                  "emby.enabled",
		"emby_server_id":                "emby.server_id",
		"emby_url":                      "emby.url",
		"emby_api_key":                  "emby.api_key",
		"emby_user_id":                  "emby.user_id",
		"emby_realtime_enabled":         "emby.realtime_enabled",
		"emby_session_polling_enabled":  "emby.session_polling_enabled",
		"emby_session_polling_interval": "emby.session_polling_interval",
		"emby_webhooks_enabled":         "emby.webhooks_enabled",
		"emby_webhook_secret":           "emby.webhook_secret",

		// NATS mappings
		"nats_enabled":        "nats.enabled",
		"nats_event_sourcing": "nats.event_sourcing",
		"nats_url":            "nats.url",
		"nats_embedded":       "nats.embedded_server",
		"nats_store_dir":      "nats.store_dir",
		"nats_max_memory":     "nats.max_memory",
		"nats_max_store":      "nats.max_store",
		"nats_retention_days": "nats.stream_retention_days",
		"nats_batch_size":     "nats.batch_size",
		"nats_flush_interval": "nats.flush_interval",
		"nats_subscribers":    "nats.subscribers_count",
		"nats_durable_name":   "nats.durable_name",
		"nats_queue_group":    "nats.queue_group",
		// Router configuration environment mappings
		"nats_router_retry_count":    "nats.router_retry_count",
		"nats_router_retry_interval": "nats.router_retry_initial_interval",
		"nats_router_throttle":       "nats.router_throttle_per_second",
		"nats_router_dedup_enabled":  "nats.router_deduplication_enabled",
		"nats_router_dedup_ttl":      "nats.router_deduplication_ttl",
		"nats_router_poison_enabled": "nats.router_poison_queue_enabled",
		"nats_router_poison_topic":   "nats.router_poison_queue_topic",
		"nats_router_close_timeout":  "nats.router_close_timeout",

		// Database mappings
		"duckdb_path":       "database.path",
		"duckdb_max_memory": "database.max_memory",
		"seed_mock_data":    "database.seed_mock_data",

		// Sync mappings
		"sync_interval":       "sync.interval",
		"sync_lookback":       "sync.lookback",
		"sync_batch_size":     "sync.batch_size",
		"sync_retry_attempts": "sync.retry_attempts",
		"sync_retry_delay":    "sync.retry_delay",

		// Server mappings
		"http_port":        "server.port",
		"http_host":        "server.host",
		"http_timeout":     "server.timeout",
		"server_latitude":  "server.latitude",
		"server_longitude": "server.longitude",
		"environment":      "server.environment", // M-02: Environment mode for security validation

		// API mappings
		"api_default_page_size": "api.default_page_size",
		"api_max_page_size":     "api.max_page_size",

		// Security mappings
		"auth_mode":           "security.auth_mode",
		"jwt_secret":          "security.jwt_secret",
		"session_timeout":     "security.session_timeout",
		"admin_username":      "security.admin_username",
		"admin_password":      "security.admin_password",
		"rate_limit_requests": "security.rate_limit_reqs",
		"rate_limit_window":   "security.rate_limit_window",
		"disable_rate_limit":  "security.rate_limit_disabled",
		"cors_origins":        "security.cors_origins",
		"trusted_proxies":     "security.trusted_proxies",

		// Session Store mappings (ADR-0015 Phase 4)
		"session_store":      "security.session_store",
		"session_store_path": "security.session_store_path",

		// OIDC mappings (ADR-0015)
		"oidc_issuer_url":               "security.oidc.issuer_url",
		"oidc_client_id":                "security.oidc.client_id",
		"oidc_client_secret":            "security.oidc.client_secret",
		"oidc_redirect_url":             "security.oidc.redirect_url",
		"oidc_post_logout_redirect_uri": "security.oidc.post_logout_redirect_uri",
		"oidc_scopes":                   "security.oidc.scopes",
		"oidc_pkce_enabled":             "security.oidc.pkce_enabled",
		"oidc_jwks_cache_ttl":           "security.oidc.jwks_cache_ttl",
		"oidc_session_max_age":          "security.oidc.session_max_age",
		"oidc_session_secret":           "security.oidc.session_secret",
		"oidc_cookie_name":              "security.oidc.cookie_name",
		"oidc_cookie_secure":            "security.oidc.cookie_secure",
		"oidc_roles_claim":              "security.oidc.roles_claim",
		"oidc_default_roles":            "security.oidc.default_roles",
		"oidc_username_claims":          "security.oidc.username_claims",

		// Plex Auth mappings (ADR-0015)
		"plex_auth_client_id":      "security.plex_auth.client_id",
		"plex_auth_client_secret":  "security.plex_auth.client_secret",
		"plex_auth_redirect_uri":   "security.plex_auth.redirect_uri",
		"plex_auth_default_roles":  "security.plex_auth.default_roles",
		"plex_auth_plex_pass_role": "security.plex_auth.plex_pass_role",
		"plex_auth_timeout":        "security.plex_auth.timeout",

		// Casbin mappings (ADR-0015)
		"casbin_model_path":      "security.casbin.model_path",
		"casbin_policy_path":     "security.casbin.policy_path",
		"casbin_default_role":    "security.casbin.default_role",
		"casbin_auto_reload":     "security.casbin.auto_reload",
		"casbin_reload_interval": "security.casbin.reload_interval",
		"casbin_cache_enabled":   "security.casbin.cache_enabled",
		"casbin_cache_ttl":       "security.casbin.cache_ttl",

		// Logging mappings
		"log_level":  "logging.level",
		"log_format": "logging.format",
		"log_caller": "logging.caller",

		// Recommendation engine mappings (ADR-0024)
		"recommend_enabled":             "recommend.enabled",
		"recommend_train_interval":      "recommend.train_interval",
		"recommend_train_on_startup":    "recommend.train_on_startup",
		"recommend_min_interactions":    "recommend.min_interactions",
		"recommend_model_path":          "recommend.model_path",
		"recommend_algorithms":          "recommend.algorithms",
		"recommend_cache_ttl":           "recommend.cache_ttl",
		"recommend_max_candidates":      "recommend.max_candidates",
		"recommend_diversity_lambda":    "recommend.diversity_lambda",
		"recommend_calibration_enabled": "recommend.calibration_enabled",
		// EASE algorithm settings
		"recommend_ease_regularization": "recommend.ease.l2_regularization",
		"recommend_ease_min_confidence": "recommend.ease.min_confidence",
		"recommend_ease_parallel":       "recommend.ease.use_parallel",
		// ALS algorithm settings
		"recommend_als_factors":        "recommend.als.factors",
		"recommend_als_iterations":     "recommend.als.iterations",
		"recommend_als_regularization": "recommend.als.regularization",
		"recommend_als_alpha":          "recommend.als.alpha",
		"recommend_als_workers":        "recommend.als.num_workers",
		// KNN algorithm settings
		"recommend_knn_neighbors":  "recommend.knn.neighbors",
		"recommend_knn_similarity": "recommend.knn.similarity",
		"recommend_knn_shrinkage":  "recommend.knn.shrinkage",
		"recommend_knn_min_common": "recommend.knn.min_common_items",
		// FPMC algorithm settings
		"recommend_fpmc_factors":          "recommend.fpmc.factors",
		"recommend_fpmc_learning_rate":    "recommend.fpmc.learning_rate",
		"recommend_fpmc_regularization":   "recommend.fpmc.regularization",
		"recommend_fpmc_epochs":           "recommend.fpmc.epochs",
		"recommend_fpmc_negative_samples": "recommend.fpmc.negative_samples",
		// LinUCB algorithm settings
		"recommend_linucb_alpha":    "recommend.linucb.alpha",
		"recommend_linucb_features": "recommend.linucb.num_features",
		"recommend_linucb_decay":    "recommend.linucb.decay_rate",

		// Newsletter scheduler mappings
		"newsletter_enabled":        "newsletter.enabled",
		"newsletter_check_interval": "newsletter.check_interval",
		"newsletter_max_concurrent": "newsletter.max_concurrent",
		"newsletter_exec_timeout":   "newsletter.execution_timeout",
	}

	if mapped, ok := envMappings[key]; ok {
		return mapped
	}

	// For unmapped keys, return empty string to skip them
	// This prevents random environment variables from polluting config
	return ""
}

// GetKoanfInstance returns a new Koanf instance for advanced usage.
// This is useful for:
//   - Hot-reload scenarios (with proper mutex protection)
//   - Custom configuration sources
//   - Testing with mock configurations
func GetKoanfInstance() *koanf.Koanf {
	return koanf.New(".")
}

// WatchConfigFile sets up a file watcher for hot-reload capability.
// Note: The caller is responsible for mutex protection when accessing
// configuration during reloads.
//
// Example usage:
//
//	var cfgMu sync.RWMutex
//	var cfg *Config
//
//	err := WatchConfigFile(configPath, func() {
//	    cfgMu.Lock()
//	    defer cfgMu.Unlock()
//	    newCfg, err := LoadWithKoanf()
//	    if err != nil {
//	        log.Printf("Config reload failed: %v", err)
//	        return
//	    }
//	    cfg = newCfg
//	    log.Println("Configuration reloaded successfully")
//	})
func WatchConfigFile(path string, callback func()) error {
	provider := file.Provider(path)

	// Start watching the file for changes
	return provider.Watch(func(event interface{}, err error) {
		if err != nil {
			return
		}
		callback()
	})
}
