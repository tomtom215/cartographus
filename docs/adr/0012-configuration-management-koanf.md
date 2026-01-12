# ADR-0012: Configuration Management with Koanf v2

**Date**: 2025-12-03
**Status**: Accepted
**Last Verified**: 2026-01-11

---

## Context

Cartographus configuration grew organically with ~52 environment variables. Issues arose:

1. **No Config File**: Users couldn't persist configuration easily
2. **Scattered Parsing**: Environment variables parsed in multiple locations
3. **Type Unsafety**: Manual string-to-type conversions
4. **No Defaults**: Missing values caused runtime errors
5. **No Validation**: Invalid values discovered at runtime

### Requirements

- Environment variable support (backward compatibility)
- Config file support (YAML, for persistence)
- Default values for all optional settings
- Type-safe unmarshaling
- Layered configuration (defaults < file < env)

### Alternatives Considered

| Library | Pros | Cons |
|---------|------|------|
| **envconfig** | Simple, struct tags | No config files |
| **viper** | Full-featured, popular | Heavy, complex |
| **koanf** | Modular, lightweight | Less documentation |
| **godotenv** | .env files | No config files |

---

## Decision

Use **Koanf v2** for layered configuration management:

- **Providers**: structs (defaults), file (YAML), env
- **Parsers**: YAML parser for config files
- **Priority**: Environment variables override config file overrides defaults

### Architecture

```
┌───────────────────────────────────────────────────────┐
│                    Configuration                       │
└───────────────────────────────────────────────────────┘
                           │
    ┌──────────────────────┼──────────────────────────┐
    │                      │                          │
    ▼                      ▼                          ▼
┌─────────┐          ┌─────────┐              ┌─────────────┐
│ Defaults│ ◄─────── │ Config  │ ◄─────────── │ Environment │
│ (struct)│  lowest  │  File   │   middle     │  Variables  │  highest
│         │ priority │ (YAML)  │   priority   │             │  priority
└─────────┘          └─────────┘              └─────────────┘
```

### Key Factors

1. **Backward Compatible**: Environment variables still work
2. **File Support**: Users can version control configuration
3. **Type Safety**: Struct-based unmarshaling
4. **Sensible Defaults**: All optional values have defaults
5. **Modular**: Only include needed providers

---

## Consequences

### Positive

- **Layered Configuration**: Clear precedence rules
- **Type Safety**: Compile-time type checking
- **Default Values**: No runtime panics from missing config
- **Config Files**: Users can persist settings
- **Clean Code**: Single config loading location

### Negative

- **Additional Dependency**: `github.com/knadh/koanf/v2`
- **Learning Curve**: New API to understand
- **Migration**: Existing deployments may need adjustment

### Neutral

- **YAML Format**: Standard, human-readable
- **Struct Tags**: New `koanf` tags alongside existing ones

---

## Implementation

### Configuration Structure

The main Config struct in `internal/config/config.go` contains all application configuration:

```go
// internal/config/config.go
type Config struct {
    Tautulli   TautulliConfig   `koanf:"tautulli"`   // Optional data source (v2.0+)
    Plex       PlexConfig       `koanf:"plex"`       // Single Plex server
    Jellyfin   JellyfinConfig   `koanf:"jellyfin"`   // Single Jellyfin server (v1.51+)
    Emby       EmbyConfig       `koanf:"emby"`       // Single Emby server (v1.51+)
    NATS       NATSConfig       `koanf:"nats"`       // Event processing (v1.48+)
    Import     ImportConfig     `koanf:"import"`     // Database import (v1.49+)
    Detection  DetectionConfig  `koanf:"detection"`  // Security monitoring (ADR-0020)
    VPN        VPNConfig        `koanf:"vpn"`        // VPN detection
    Recommend  RecommendConfig  `koanf:"recommend"`  // Recommendation engine (ADR-0024)
    GeoIP      GeoIPConfig      `koanf:"geoip"`      // Standalone GeoIP (v2.0+)
    Newsletter NewsletterConfig `koanf:"newsletter"` // Newsletter scheduler
    Database   DatabaseConfig   `koanf:"database"`
    Sync       SyncConfig       `koanf:"sync"`
    Server     ServerConfig     `koanf:"server"`
    API        APIConfig        `koanf:"api"`
    Security   SecurityConfig   `koanf:"security"`
    Logging    LoggingConfig    `koanf:"logging"`

    // Multi-Server Support (v2.1)
    PlexServers     []PlexConfig     `koanf:"plex_servers"`
    JellyfinServers []JellyfinConfig `koanf:"jellyfin_servers"`
    EmbyServers     []EmbyConfig     `koanf:"emby_servers"`
}

type ServerConfig struct {
    Port        int           `koanf:"port"`        // HTTP_PORT, default: 3857
    Host        string        `koanf:"host"`        // HTTP_HOST, default: "0.0.0.0"
    Timeout     time.Duration `koanf:"timeout"`     // HTTP_TIMEOUT, default: 30s
    Latitude    float64       `koanf:"latitude"`    // SERVER_LATITUDE, default: 0
    Longitude   float64       `koanf:"longitude"`   // SERVER_LONGITUDE, default: 0
    Environment string        `koanf:"environment"` // ENVIRONMENT, default: "development"
}

type DatabaseConfig struct {
    Path                   string `koanf:"path"`                   // DUCKDB_PATH, default: /data/cartographus.duckdb
    MaxMemory              string `koanf:"max_memory"`             // DUCKDB_MAX_MEMORY, default: 2GB
    Threads                int    `koanf:"threads"`                // DUCKDB_THREADS, default: 0 (NumCPU)
    PreserveInsertionOrder bool   `koanf:"preserve_insertion_order"` // default: true
    SeedMockData           bool   `koanf:"seed_mock_data"`         // SEED_MOCK_DATA, default: false
    SkipIndexes            bool   `koanf:"skip_indexes"`           // For fast test setup
}

type TautulliConfig struct {
    Enabled  bool   `koanf:"enabled"`   // TAUTULLI_ENABLED, default: false (optional v2.0+)
    URL      string `koanf:"url"`       // TAUTULLI_URL
    APIKey   string `koanf:"api_key"`   // TAUTULLI_API_KEY
    ServerID string `koanf:"server_id"` // For multi-server deduplication
}

type SecurityConfig struct {
    AuthMode          string        `koanf:"auth_mode"`           // AUTH_MODE, default: jwt
    JWTSecret         string        `koanf:"jwt_secret"`          // JWT_SECRET
    SessionTimeout    time.Duration `koanf:"session_timeout"`     // default: 24h
    AdminUsername     string        `koanf:"admin_username"`      // ADMIN_USERNAME
    AdminPassword     string        `koanf:"admin_password"`      // ADMIN_PASSWORD
    RateLimitReqs     int           `koanf:"rate_limit_reqs"`     // default: 100
    RateLimitWindow   time.Duration `koanf:"rate_limit_window"`   // default: 1m
    RateLimitDisabled bool          `koanf:"rate_limit_disabled"` // default: false
    CORSOrigins       []string      `koanf:"cors_origins"`        // default: ["*"]
    TrustedProxies    []string      `koanf:"trusted_proxies"`     // default: []
    SessionStore      string        `koanf:"session_store"`       // memory or badger
    SessionStorePath  string        `koanf:"session_store_path"`  // BadgerDB path

    // Nested auth configs (ADR-0015)
    OIDC     OIDCConfig     `koanf:"oidc"`
    PlexAuth PlexAuthConfig `koanf:"plex_auth"`
    Casbin   CasbinConfig   `koanf:"casbin"`
}
```

### Configuration Loading

The `LoadWithKoanf()` function in `internal/config/koanf.go` handles layered configuration:

```go
// internal/config/koanf.go
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

// Load is the public entry point that calls LoadWithKoanf
func Load() (*Config, error) {
    return LoadWithKoanf()
}
```

Config file search paths are defined in `koanf.go`:

```go
var DefaultConfigPaths = []string{
    "config.yaml",
    "config.yml",
    "/etc/cartographus/config.yaml",
    "/etc/cartographus/config.yml",
}

const ConfigPathEnvVar = "CONFIG_PATH"

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
```

### Environment Variable Mapping

The `envTransformFunc` in `internal/config/koanf.go` maps environment variables to config paths:

```go
// internal/config/koanf.go
func envTransformFunc(key string) string {
    key = strings.ToLower(key)

    // Map legacy environment variable prefixes to config sections
    envMappings := map[string]string{
        // Tautulli mappings
        "tautulli_enabled":   "tautulli.enabled",
        "tautulli_url":       "tautulli.url",
        "tautulli_api_key":   "tautulli.api_key",

        // Plex mappings
        "enable_plex_sync":   "plex.enabled",
        "plex_url":           "plex.url",
        "plex_token":         "plex.token",

        // Database mappings
        "duckdb_path":        "database.path",
        "duckdb_max_memory":  "database.max_memory",

        // Server mappings
        "http_port":          "server.port",
        "http_host":          "server.host",
        "http_timeout":       "server.timeout",

        // Security mappings
        "auth_mode":          "security.auth_mode",
        "jwt_secret":         "security.jwt_secret",
        "admin_username":     "security.admin_username",
        "admin_password":     "security.admin_password",

        // NATS mappings
        "nats_enabled":       "nats.enabled",
        "nats_event_sourcing": "nats.event_sourcing",
        // ... 100+ more mappings
    }

    if mapped, ok := envMappings[key]; ok {
        return mapped
    }
    // Unmapped keys return empty string to skip
    return ""
}
```

### Config File Example

```yaml
# config.yaml
server:
  port: 3857
  host: "0.0.0.0"
  timeout: 30s
  latitude: 40.7128
  longitude: -74.0060
  environment: production

database:
  path: /data/cartographus.duckdb
  max_memory: 4GB
  threads: 8

tautulli:
  enabled: true
  url: http://tautulli:8181
  api_key: ${TAUTULLI_API_KEY}

security:
  auth_mode: jwt
  session_timeout: 24h
  rate_limit_reqs: 100
  rate_limit_window: 1m
  cors_origins:
    - https://maps.example.com
```

### Validation

Validation is implemented in `internal/config/config_validate.go`:

```go
// internal/config/config_validate.go
func (c *Config) Validate() error {
    if err := c.validateTautulli(); err != nil {
        return err
    }
    if err := c.validatePlex(); err != nil {
        return err
    }
    if err := c.validateNATS(); err != nil {
        return err
    }
    if err := c.validateServer(); err != nil {
        return err
    }
    if err := c.validateSecurity(); err != nil {
        return err
    }
    return c.validateLogging()
}

func (c *Config) validateServer() error {
    if c.Server.Port < 1 || c.Server.Port > 65535 {
        return fmt.Errorf("HTTP_PORT must be between 1 and 65535")
    }
    return nil
}

func (c *Config) validateSecurity() error {
    // Validates auth mode, CORS, rate limits, and auth-mode-specific config
    // See config_validate.go for full implementation
}
```

### Code References

| Component | File | Notes |
|-----------|------|-------|
| Config struct | `internal/config/config.go` | Type definitions |
| Load function | `internal/config/koanf.go` | Koanf loading with `LoadWithKoanf()` |
| Env mapping | `internal/config/koanf.go` | `envTransformFunc()` function |
| Env helpers | `internal/config/config_env.go` | `getEnv()`, `getBoolEnv()`, etc. |
| Validation | `internal/config/config_validate.go` | Config validation |
| Defaults | `internal/config/koanf.go` | `defaultConfig()` function |

---

## Verification

### Verified Claims

| Claim | Source | Verified |
|-------|--------|----------|
| Koanf v2.3.0 | `go.mod:37` | Yes |
| YAML parser v1.1.0 | `go.mod:33` | Yes |
| Env provider v1.1.0 | `go.mod:34` | Yes |
| File provider v1.2.1 | `go.mod:35` | Yes |
| Structs provider v1.0.0 | `go.mod:36` | Yes |

### Test Coverage

- Config tests: `internal/config/config_test.go`
- Koanf loading tests: `internal/config/koanf_test.go`
- Password policy tests: `internal/config/password_policy_test.go`
- Zero Trust config tests: `internal/config/zerotrust_config_test.go`
- Coverage target: 90%+ for config package

---

## Related ADRs

- [ADR-0003](0003-authentication-architecture.md): Security configuration
- [ADR-0013](0013-request-validation.md): Request validation
- [ADR-0015](0015-zero-trust-authentication-authorization.md): Zero Trust auth config

---

## References

- [Koanf Documentation](https://github.com/knadh/koanf)
- [12-Factor App Configuration](https://12factor.net/config)
- [docs/DEVELOPMENT.md](../DEVELOPMENT.md)
