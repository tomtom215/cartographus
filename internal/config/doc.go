// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

/*
Package config provides centralized configuration management for Cartographus.

This package handles loading, validation, and parsing of environment variables
for all application components. It ensures consistent configuration across the
backend services and provides sensible defaults for optional settings.

# Configuration Sources

The package reads configuration from:
  - Environment variables (primary source)
  - .env files (development/local)
  - System environment (production/Docker)

# Configuration Structure

The package organizes configuration into logical groups:

  - HTTPConfig: HTTP server settings (host, port, timeouts)
  - TautulliConfig: Tautulli API connection and sync settings
  - DatabaseConfig: DuckDB connection and performance tuning
  - AuthConfig: JWT and Basic Auth security settings
  - CacheConfig: In-memory caching parameters
  - MetricsConfig: Prometheus metrics and observability
  - PlexConfig: Plex server integration (v1.39+)

# Environment Variables

The package supports 52+ environment variables organized by component:

HTTP Server (HTTPConfig):
  - HTTP_HOST: Bind address (default: 0.0.0.0)
  - HTTP_PORT: Listen port (default: 3857)
  - HTTP_READ_TIMEOUT: Request read timeout (default: 30s)
  - HTTP_WRITE_TIMEOUT: Response write timeout (default: 30s)
  - HTTP_IDLE_TIMEOUT: Keep-alive idle timeout (default: 120s)
  - HTTP_MAX_HEADER_BYTES: Max header size (default: 1MB)

Authentication (AuthConfig):
  - AUTH_MODE: Authentication mode (jwt, basic, disabled)
  - JWT_SECRET: JWT signing secret (min 32 chars, required for jwt mode)
  - JWT_EXPIRY: JWT token expiration (default: 24h)
  - ADMIN_USERNAME: Admin login username (required for jwt/basic)
  - ADMIN_PASSWORD: Admin login password (min 8 chars, required)
  - TRUSTED_PROXIES: Comma-separated list of trusted proxy IPs

Tautulli Integration (TautulliConfig):
  - TAUTULLI_URL: Tautulli server URL (required)
  - TAUTULLI_API_KEY: Tautulli API key (required)
  - TAUTULLI_VERIFY_SSL: Verify SSL certificates (default: true)
  - SYNC_INTERVAL: Sync interval (default: 15m)
  - SYNC_BATCH_SIZE: Batch size for sync (default: 1000)

Database (DatabaseConfig):
  - DUCKDB_PATH: Database file path (default: /data/cartographus.duckdb)
  - DUCKDB_THREADS: Thread count (default: CPU count)
  - DUCKDB_MEMORY_LIMIT: Memory limit (default: 80% of RAM)

Caching (CacheConfig):
  - CACHE_ENABLED: Enable in-memory cache (default: true)
  - CACHE_TTL: Cache time-to-live (default: 5m)
  - CACHE_MAX_SIZE: Max cache size in MB (default: 100)

Metrics (MetricsConfig):
  - METRICS_ENABLED: Enable Prometheus metrics (default: true)
  - METRICS_PATH: Metrics endpoint path (default: /metrics)

Plex Integration (PlexConfig):
  - ENABLE_PLEX_SYNC: Enable Plex sync (default: false)
  - PLEX_URL: Plex server URL
  - PLEX_TOKEN: Plex auth token
  - ENABLE_PLEX_REALTIME: Enable WebSocket (default: false)
  - ENABLE_PLEX_TRANSCODE_MONITORING: Enable monitoring (default: false)
  - PLEX_TRANSCODE_MONITORING_INTERVAL: Poll interval (default: 10s)
  - ENABLE_BUFFER_HEALTH_MONITORING: Enable buffer monitoring (default: false)
  - BUFFER_HEALTH_POLL_INTERVAL: Poll interval (default: 5s)

# Usage Example

Basic configuration loading:

	import "github.com/tomtom215/cartographus/internal/config"

	// Load configuration from environment
	cfg, err := config.Load()
	if err != nil {
	    log.Fatalf("Failed to load config: %v", err)
	}

	// Access configuration values
	fmt.Printf("Starting server on %s:%d\n", cfg.HTTP.Host, cfg.HTTP.Port)
	fmt.Printf("Syncing from Tautulli: %s\n", cfg.Tautulli.URL)
	fmt.Printf("Database: %s\n", cfg.Database.Path)

Testing with custom configuration:

	// Override environment variables for testing
	os.Setenv("HTTP_PORT", "8080")
	os.Setenv("TAUTULLI_URL", "http://test-tautulli:8181")
	os.Setenv("JWT_SECRET", "test-secret-at-least-32-characters-long")

	cfg, err := config.Load()
	// Use cfg for testing

# Validation

The package performs comprehensive validation:

  - Required fields: TAUTULLI_URL, TAUTULLI_API_KEY, JWT_SECRET (if AUTH_MODE=jwt)
  - String length: JWT_SECRET ≥32 chars, ADMIN_PASSWORD ≥8 chars
  - Numeric ranges: HTTP_PORT (1-65535), SYNC_BATCH_SIZE (1-10000)
  - Duration ranges: SYNC_INTERVAL ≥1m, JWT_EXPIRY ≥1h
  - URL formats: TAUTULLI_URL, PLEX_URL must be valid HTTP(S) URLs
  - IP addresses: TRUSTED_PROXIES must be valid IPs/CIDRs

# Defaults

Sensible defaults are provided for all optional settings:

  - HTTP_PORT: 3857 (matches EPSG:3857 Web Mercator projection)
  - SYNC_INTERVAL: 15 minutes (balances freshness and API load)
  - CACHE_TTL: 5 minutes (analytics caching)
  - JWT_EXPIRY: 24 hours (session duration)
  - DUCKDB_THREADS: CPU count (max parallelism)
  - DUCKDB_MEMORY_LIMIT: 80% of system RAM (safe default)

# Security Best Practices

When configuring authentication:

 1. Use strong JWT secrets: Minimum 32 characters, cryptographically random
    Generate with: openssl rand -base64 48

 2. Use strong admin passwords: Minimum 8 characters, mixed case + numbers + symbols

 3. Configure trusted proxies: Only allow known reverse proxy IPs
    Example: TRUSTED_PROXIES=127.0.0.1,10.0.0.0/8

 4. Enable SSL verification: TAUTULLI_VERIFY_SSL=true in production

 5. Set appropriate timeouts: Prevent slowloris attacks with reasonable timeouts

# Environment Files

For local development, create a .env file:

	# .env
	TAUTULLI_URL=http://localhost:8181
	TAUTULLI_API_KEY=your-api-key-here
	JWT_SECRET=your-secure-secret-at-least-32-characters-long
	ADMIN_USERNAME=admin
	ADMIN_PASSWORD=secure-password
	HTTP_PORT=3857
	LOG_LEVEL=debug

The package automatically loads .env files when present.

# Docker Deployment

For Docker deployments, use environment variables or docker-compose.yml:

	services:
	  cartographus:
	    image: ghcr.io/tomtom215/cartographus:latest
	    environment:
	      TAUTULLI_URL: http://tautulli:8181
	      TAUTULLI_API_KEY: ${TAUTULLI_API_KEY}
	      JWT_SECRET: ${JWT_SECRET}
	      ADMIN_USERNAME: admin
	      ADMIN_PASSWORD: ${ADMIN_PASSWORD}
	    ports:
	      - "3857:3857"

# Thread Safety

The Config struct is immutable after Load() returns, making it safe for concurrent
access from multiple goroutines without synchronization.

# Performance

Configuration loading is fast (<10ms) and only happens once at startup. Values
are parsed and validated during Load(), so runtime access is direct field reads
with zero overhead.

# See Also

  - .env.example: Complete configuration template with all variables
  - README.md: User-facing configuration documentation
  - ARCHITECTURE.md: System architecture and component relationships
*/
package config
