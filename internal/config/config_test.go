// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package config

import (
	"os"
	"strings"
	"testing"
	"time"
)

// Test helpers to reduce cyclomatic complexity

// setupTestEnv sets up test environment variables and returns cleanup function
func setupTestEnv(t *testing.T, envVars map[string]string) func() {
	t.Helper()
	os.Clearenv()
	for k, v := range envVars {
		if err := os.Setenv(k, v); err != nil {
			t.Fatalf("failed to set env var %s: %v", k, err)
		}
	}
	return func() {
		os.Clearenv()
	}
}

// assertNoError checks that error is nil
func assertNoError(t *testing.T, err error, testName string) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: unexpected error: %v", testName, err)
	}
}

// assertError checks that error occurred and optionally matches message
func assertError(t *testing.T, err error, expectedMsg, testName string) {
	t.Helper()
	if err == nil {
		t.Fatalf("%s: expected error containing %q, got nil", testName, expectedMsg)
	}
	if expectedMsg != "" && err.Error() != expectedMsg {
		t.Errorf("%s: error = %v, want error containing %q", testName, err, expectedMsg)
	}
}

// assertConfigNotNil checks that config is not nil
func assertConfigNotNil(t *testing.T, cfg *Config, testName string) {
	t.Helper()
	if cfg == nil {
		t.Fatalf("%s: config is nil", testName)
	}
}

// assertIntEqual checks integer equality
func assertIntEqual(t *testing.T, got, want int, field, testName string) {
	t.Helper()
	if got != want {
		t.Errorf("%s: %s = %v, want %v", testName, field, got, want)
	}
}

// assertStringEqual checks string equality
func assertStringEqual(t *testing.T, got, want, field string) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %v, want %v", field, got, want)
	}
}

// assertBoolEqual checks boolean equality
func assertBoolEqual(t *testing.T, got, want bool, field string) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %v, want %v", field, got, want)
	}
}

// assertFloatEqual checks float equality
func assertFloatEqual(t *testing.T, got, want float64, field string) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %v, want %v", field, got, want)
	}
}

// assertDurationEqual checks time.Duration equality
func assertDurationEqual(t *testing.T, got, want time.Duration, field string) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %v, want %v", field, got, want)
	}
}

// assertSliceLength checks slice length
func assertSliceLength(t *testing.T, got, want int, field string) {
	t.Helper()
	if got != want {
		t.Errorf("%s length = %v, want %v", field, got, want)
	}
}

// assertTautulliConfig validates Tautulli configuration section
func assertTautulliConfig(t *testing.T, cfg *Config, url, apiKey string) {
	t.Helper()
	assertStringEqual(t, cfg.Tautulli.URL, url, "Tautulli.URL")
	assertStringEqual(t, cfg.Tautulli.APIKey, apiKey, "Tautulli.APIKey")
}

// assertDatabaseConfig validates Database configuration section
func assertDatabaseConfig(t *testing.T, cfg *Config, path, maxMemory string, seedMockData bool) {
	t.Helper()
	assertStringEqual(t, cfg.Database.Path, path, "Database.Path")
	assertStringEqual(t, cfg.Database.MaxMemory, maxMemory, "Database.MaxMemory")
	assertBoolEqual(t, cfg.Database.SeedMockData, seedMockData, "Database.SeedMockData")
}

// assertSyncConfig validates Sync configuration section
func assertSyncConfig(t *testing.T, cfg *Config, interval, lookback, retryDelay time.Duration, batchSize, retryAttempts int) {
	t.Helper()
	assertDurationEqual(t, cfg.Sync.Interval, interval, "Sync.Interval")
	assertDurationEqual(t, cfg.Sync.Lookback, lookback, "Sync.Lookback")
	assertIntEqual(t, cfg.Sync.BatchSize, batchSize, "Sync.BatchSize", "")
	assertIntEqual(t, cfg.Sync.RetryAttempts, retryAttempts, "Sync.RetryAttempts", "")
	assertDurationEqual(t, cfg.Sync.RetryDelay, retryDelay, "Sync.RetryDelay")
}

// assertServerConfig validates Server configuration section
func assertServerConfig(t *testing.T, cfg *Config, port int, host string, timeout time.Duration, lat, lon float64) {
	t.Helper()
	assertIntEqual(t, cfg.Server.Port, port, "Server.Port", "")
	assertStringEqual(t, cfg.Server.Host, host, "Server.Host")
	assertDurationEqual(t, cfg.Server.Timeout, timeout, "Server.Timeout")
	assertFloatEqual(t, cfg.Server.Latitude, lat, "Server.Latitude")
	assertFloatEqual(t, cfg.Server.Longitude, lon, "Server.Longitude")
}

// assertAPIConfig validates API configuration section
func assertAPIConfig(t *testing.T, cfg *Config, defaultPageSize, maxPageSize int) {
	t.Helper()
	assertIntEqual(t, cfg.API.DefaultPageSize, defaultPageSize, "API.DefaultPageSize", "")
	assertIntEqual(t, cfg.API.MaxPageSize, maxPageSize, "API.MaxPageSize", "")
}

// assertSecurityConfig validates Security configuration section
func assertSecurityConfig(t *testing.T, cfg *Config, authMode string, rateLimitReqs int, rateLimitWindow time.Duration, rateLimitDisabled bool, corsOriginsLen, trustedProxiesLen int) {
	t.Helper()
	assertStringEqual(t, cfg.Security.AuthMode, authMode, "Security.AuthMode")
	assertIntEqual(t, cfg.Security.RateLimitReqs, rateLimitReqs, "Security.RateLimitReqs", "")
	assertDurationEqual(t, cfg.Security.RateLimitWindow, rateLimitWindow, "Security.RateLimitWindow")
	assertBoolEqual(t, cfg.Security.RateLimitDisabled, rateLimitDisabled, "Security.RateLimitDisabled")
	assertSliceLength(t, len(cfg.Security.CORSOrigins), corsOriginsLen, "Security.CORSOrigins")
	assertSliceLength(t, len(cfg.Security.TrustedProxies), trustedProxiesLen, "Security.TrustedProxies")
}

// assertLoggingConfig validates Logging configuration section
func assertLoggingConfig(t *testing.T, cfg *Config, level string) {
	t.Helper()
	assertStringEqual(t, cfg.Logging.Level, level, "Logging.Level")
}

// assertPlexConfig validates Plex configuration section
func assertPlexConfig(t *testing.T, cfg *Config, enabled bool, url, token string, historicalSync bool, syncDaysBack int, syncInterval time.Duration, realtimeEnabled bool, oauthClientID, oauthClientSecret, oauthRedirectURI string, transcodeMonitoring bool, transcodeMonitoringInterval, bufferHealthPollInterval time.Duration, bufferHealthMonitoring bool, bufferHealthCriticalThreshold, bufferHealthRiskyThreshold float64) {
	t.Helper()
	assertBoolEqual(t, cfg.Plex.Enabled, enabled, "Plex.Enabled")
	assertStringEqual(t, cfg.Plex.URL, url, "Plex.URL")
	assertStringEqual(t, cfg.Plex.Token, token, "Plex.Token")
	assertBoolEqual(t, cfg.Plex.HistoricalSync, historicalSync, "Plex.HistoricalSync")
	assertIntEqual(t, cfg.Plex.SyncDaysBack, syncDaysBack, "Plex.SyncDaysBack", "")
	assertDurationEqual(t, cfg.Plex.SyncInterval, syncInterval, "Plex.SyncInterval")
	assertBoolEqual(t, cfg.Plex.RealtimeEnabled, realtimeEnabled, "Plex.RealtimeEnabled")
	assertStringEqual(t, cfg.Plex.OAuthClientID, oauthClientID, "Plex.OAuthClientID")
	assertStringEqual(t, cfg.Plex.OAuthClientSecret, oauthClientSecret, "Plex.OAuthClientSecret")
	assertStringEqual(t, cfg.Plex.OAuthRedirectURI, oauthRedirectURI, "Plex.OAuthRedirectURI")
	assertBoolEqual(t, cfg.Plex.TranscodeMonitoring, transcodeMonitoring, "Plex.TranscodeMonitoring")
	assertDurationEqual(t, cfg.Plex.TranscodeMonitoringInterval, transcodeMonitoringInterval, "Plex.TranscodeMonitoringInterval")
	assertBoolEqual(t, cfg.Plex.BufferHealthMonitoring, bufferHealthMonitoring, "Plex.BufferHealthMonitoring")
	assertDurationEqual(t, cfg.Plex.BufferHealthPollInterval, bufferHealthPollInterval, "Plex.BufferHealthPollInterval")
	assertFloatEqual(t, cfg.Plex.BufferHealthCriticalThreshold, bufferHealthCriticalThreshold, "Plex.BufferHealthCriticalThreshold")
	assertFloatEqual(t, cfg.Plex.BufferHealthRiskyThreshold, bufferHealthRiskyThreshold, "Plex.BufferHealthRiskyThreshold")
}

func TestLoad(t *testing.T) {
	tests := []struct {
		name    string
		envVars map[string]string
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid configuration",
			envVars: map[string]string{
				"TAUTULLI_URL":     "http://localhost:8181",
				"TAUTULLI_API_KEY": "test_api_key_12345678",
				"AUTH_MODE":        "jwt",
				"JWT_SECRET":       "this_is_a_very_long_secret_key_with_more_than_32_characters",
				"ADMIN_USERNAME":   "admin",
				"ADMIN_PASSWORD":   "SecureP@ss123!",
			},
			wantErr: false,
		},
		{
			name: "missing TAUTULLI_URL when enabled",
			envVars: map[string]string{
				"TAUTULLI_ENABLED": "true",
				"TAUTULLI_API_KEY": "test_api_key",
				"AUTH_MODE":        "none",
			},
			wantErr: true,
			errMsg:  "configuration validation failed: TAUTULLI_URL is required when TAUTULLI_ENABLED=true",
		},
		{
			name: "missing TAUTULLI_API_KEY when enabled",
			envVars: map[string]string{
				"TAUTULLI_ENABLED": "true",
				"TAUTULLI_URL":     "http://localhost:8181",
				"AUTH_MODE":        "none",
			},
			wantErr: true,
			errMsg:  "configuration validation failed: TAUTULLI_API_KEY is required when TAUTULLI_ENABLED=true",
		},
		{
			name: "standalone mode - no Tautulli required",
			envVars: map[string]string{
				"AUTH_MODE": "none",
			},
			wantErr: false,
		},
		{
			name: "JWT mode requires JWT_SECRET",
			envVars: map[string]string{
				"TAUTULLI_URL":     "http://localhost:8181",
				"TAUTULLI_API_KEY": "test_api_key",
				"AUTH_MODE":        "jwt",
			},
			wantErr: true,
			errMsg:  "configuration validation failed: JWT_SECRET is required when AUTH_MODE is jwt",
		},
		{
			name: "JWT_SECRET too short",
			envVars: map[string]string{
				"TAUTULLI_URL":     "http://localhost:8181",
				"TAUTULLI_API_KEY": "test_api_key",
				"AUTH_MODE":        "jwt",
				"JWT_SECRET":       "short",
				"ADMIN_USERNAME":   "admin",
				"ADMIN_PASSWORD":   "SecureP@ss123!",
			},
			wantErr: true,
			errMsg:  "configuration validation failed: JWT_SECRET must be at least 32 characters for security",
		},
		{
			name: "JWT mode requires ADMIN_USERNAME",
			envVars: map[string]string{
				"TAUTULLI_URL":     "http://localhost:8181",
				"TAUTULLI_API_KEY": "test_api_key",
				"AUTH_MODE":        "jwt",
				"JWT_SECRET":       "this_is_a_very_long_secret_key_with_more_than_32_characters",
				"ADMIN_PASSWORD":   "SecureP@ss123!",
			},
			wantErr: true,
			errMsg:  "configuration validation failed: ADMIN_USERNAME is required when AUTH_MODE is jwt",
		},
		{
			name: "JWT mode requires ADMIN_PASSWORD",
			envVars: map[string]string{
				"TAUTULLI_URL":     "http://localhost:8181",
				"TAUTULLI_API_KEY": "test_api_key",
				"AUTH_MODE":        "jwt",
				"JWT_SECRET":       "this_is_a_very_long_secret_key_with_more_than_32_characters",
				"ADMIN_USERNAME":   "admin",
			},
			wantErr: true,
			errMsg:  "configuration validation failed: ADMIN_PASSWORD is required when AUTH_MODE is jwt",
		},
		{
			name: "ADMIN_PASSWORD too short",
			envVars: map[string]string{
				"TAUTULLI_URL":     "http://localhost:8181",
				"TAUTULLI_API_KEY": "test_api_key",
				"AUTH_MODE":        "jwt",
				"JWT_SECRET":       "this_is_a_very_long_secret_key_with_more_than_32_characters",
				"ADMIN_USERNAME":   "admin",
				"ADMIN_PASSWORD":   "Sh0rt!",
			},
			wantErr: true,
			errMsg:  "configuration validation failed: ADMIN_PASSWORD: password must be at least 12 characters (got 6)",
		},
		{
			name: "invalid AUTH_MODE",
			envVars: map[string]string{
				"TAUTULLI_URL":     "http://localhost:8181",
				"TAUTULLI_API_KEY": "test_api_key",
				"AUTH_MODE":        "invalid_mode",
			},
			wantErr: true,
			errMsg:  "configuration validation failed: AUTH_MODE must be one of: none, jwt, basic, oidc, plex, multi",
		},
		{
			name: "invalid port",
			envVars: map[string]string{
				"TAUTULLI_URL":     "http://localhost:8181",
				"TAUTULLI_API_KEY": "test_api_key",
				"HTTP_PORT":        "99999",
				"AUTH_MODE":        "none",
			},
			wantErr: true,
			errMsg:  "configuration validation failed: HTTP_PORT must be between 1 and 65535",
		},
		{
			name: "invalid port (zero)",
			envVars: map[string]string{
				"TAUTULLI_URL":     "http://localhost:8181",
				"TAUTULLI_API_KEY": "test_api_key",
				"HTTP_PORT":        "0",
				"AUTH_MODE":        "none",
			},
			wantErr: true,
			errMsg:  "configuration validation failed: HTTP_PORT must be between 1 and 65535",
		},
		{
			name: "invalid log level",
			envVars: map[string]string{
				"TAUTULLI_URL":     "http://localhost:8181",
				"TAUTULLI_API_KEY": "test_api_key",
				"LOG_LEVEL":        "invalid_level",
				"AUTH_MODE":        "none",
			},
			wantErr: true,
			errMsg:  "configuration validation failed: LOG_LEVEL must be one of: trace, debug, info, warn, error",
		},
		{
			name: "basic auth mode requires ADMIN_USERNAME",
			envVars: map[string]string{
				"TAUTULLI_URL":     "http://localhost:8181",
				"TAUTULLI_API_KEY": "test_api_key",
				"AUTH_MODE":        "basic",
				"ADMIN_PASSWORD":   "SecureP@ss123!",
			},
			wantErr: true,
			errMsg:  "configuration validation failed: ADMIN_USERNAME is required when AUTH_MODE is basic",
		},
		{
			name: "basic auth mode requires ADMIN_PASSWORD",
			envVars: map[string]string{
				"TAUTULLI_URL":     "http://localhost:8181",
				"TAUTULLI_API_KEY": "test_api_key",
				"AUTH_MODE":        "basic",
				"ADMIN_USERNAME":   "admin",
			},
			wantErr: true,
			errMsg:  "configuration validation failed: ADMIN_PASSWORD is required when AUTH_MODE is basic",
		},
		{
			name: "basic auth ADMIN_PASSWORD too short",
			envVars: map[string]string{
				"TAUTULLI_URL":     "http://localhost:8181",
				"TAUTULLI_API_KEY": "test_api_key",
				"AUTH_MODE":        "basic",
				"ADMIN_USERNAME":   "admin",
				"ADMIN_PASSWORD":   "Sh0rt!",
			},
			wantErr: true,
			errMsg:  "configuration validation failed: ADMIN_PASSWORD: password must be at least 12 characters (got 6)",
		},
		{
			name: "valid basic auth configuration",
			envVars: map[string]string{
				"TAUTULLI_URL":     "http://localhost:8181",
				"TAUTULLI_API_KEY": "test_api_key_12345678",
				"AUTH_MODE":        "basic",
				"ADMIN_USERNAME":   "admin",
				"ADMIN_PASSWORD":   "SecureP@ss123!",
			},
			wantErr: false,
		},
		{
			name: "valid auth_mode=none configuration",
			envVars: map[string]string{
				"TAUTULLI_URL":     "http://localhost:8181",
				"TAUTULLI_API_KEY": "test_api_key_12345678",
				"AUTH_MODE":        "none",
			},
			wantErr: false,
		},
		{
			name: "JWT_SECRET placeholder detection - REPLACE",
			envVars: map[string]string{
				"AUTH_MODE":      "jwt",
				"JWT_SECRET":     "REPLACE_WITH_RANDOM_STRING_MIN_32_CHARS",
				"ADMIN_USERNAME": "admin",
				"ADMIN_PASSWORD": "SecureP@ss123!",
			},
			wantErr: true,
			errMsg:  "configuration validation failed: JWT_SECRET contains a placeholder value - generate a secure secret with: openssl rand -base64 32",
		},
		{
			name: "JWT_SECRET placeholder detection - CHANGEME",
			envVars: map[string]string{
				"AUTH_MODE":      "jwt",
				"JWT_SECRET":     "changeme_this_is_a_very_long_secret_key_placeholder",
				"ADMIN_USERNAME": "admin",
				"ADMIN_PASSWORD": "SecureP@ss123!",
			},
			wantErr: true,
			errMsg:  "configuration validation failed: JWT_SECRET contains a placeholder value - generate a secure secret with: openssl rand -base64 32",
		},
		{
			name: "ADMIN_PASSWORD placeholder detection - REPLACE",
			envVars: map[string]string{
				"AUTH_MODE":      "jwt",
				"JWT_SECRET":     "this_is_a_very_long_secret_key_with_more_than_32_characters",
				"ADMIN_USERNAME": "admin",
				"ADMIN_PASSWORD": "REPLACE_WITH_SECURE_PASSWORD",
			},
			wantErr: true,
			errMsg:  "configuration validation failed: ADMIN_PASSWORD contains a placeholder value - set a secure password",
		},
		{
			name: "ADMIN_PASSWORD placeholder detection basic auth - CHANGEME",
			envVars: map[string]string{
				"AUTH_MODE":      "basic",
				"ADMIN_USERNAME": "admin",
				"ADMIN_PASSWORD": "changeme123",
			},
			wantErr: true,
			errMsg:  "configuration validation failed: ADMIN_PASSWORD contains a placeholder value - set a secure password",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := setupTestEnv(t, tt.envVars)
			defer cleanup()

			cfg, err := Load()

			if tt.wantErr {
				assertError(t, err, tt.errMsg, tt.name)
			} else {
				assertNoError(t, err, tt.name)
				assertConfigNotNil(t, cfg, tt.name)
			}
		})
	}
}

func TestGetIntEnv(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		value        string
		defaultValue int
		want         int
	}{
		{
			name:         "valid integer",
			key:          "TEST_INT",
			value:        "42",
			defaultValue: 10,
			want:         42,
		},
		{
			name:         "empty value uses default",
			key:          "TEST_INT",
			value:        "",
			defaultValue: 10,
			want:         10,
		},
		{
			name:         "invalid value uses default",
			key:          "TEST_INT",
			value:        "not_a_number",
			defaultValue: 10,
			want:         10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			envVars := map[string]string{}
			if tt.value != "" {
				envVars[tt.key] = tt.value
			}
			cleanup := setupTestEnv(t, envVars)
			defer cleanup()

			got := getIntEnv(tt.key, tt.defaultValue)
			assertIntEqual(t, got, tt.want, "getIntEnv", tt.name)
		})
	}
}

func TestGetDurationEnv(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		value        string
		defaultValue time.Duration
		want         time.Duration
	}{
		{
			name:         "valid duration",
			key:          "TEST_DURATION",
			value:        "5m",
			defaultValue: 1 * time.Minute,
			want:         5 * time.Minute,
		},
		{
			name:         "empty value uses default",
			key:          "TEST_DURATION",
			value:        "",
			defaultValue: 1 * time.Minute,
			want:         1 * time.Minute,
		},
		{
			name:         "invalid value uses default",
			key:          "TEST_DURATION",
			value:        "invalid",
			defaultValue: 1 * time.Minute,
			want:         1 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Clearenv()
			if tt.value != "" {
				_ = os.Setenv(tt.key, tt.value)
			}

			got := getDurationEnv(tt.key, tt.defaultValue)
			if got != tt.want {
				t.Errorf("getDurationEnv() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateTautulliURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
		errMsg  string
	}{
		// Valid URLs - HTTP
		{
			name:    "valid HTTP with hostname and port",
			url:     "http://localhost:8181",
			wantErr: false,
		},
		{
			name:    "valid HTTP with IP address and port",
			url:     "http://192.168.1.100:8181",
			wantErr: false,
		},
		{
			name:    "valid HTTP with hostname (no port)",
			url:     "http://tautulli.example.com",
			wantErr: false,
		},
		{
			name:    "valid HTTP with IP address (no explicit port)",
			url:     "http://192.168.1.100",
			wantErr: false,
		},
		{
			name:    "valid HTTP with trailing slash",
			url:     "http://localhost:8181/",
			wantErr: false,
		},
		// Valid URLs - HTTPS
		{
			name:    "valid HTTPS with hostname and port",
			url:     "https://tautulli.example.com:8181",
			wantErr: false,
		},
		{
			name:    "valid HTTPS with IP address and port",
			url:     "https://192.168.1.100:8181",
			wantErr: false,
		},
		{
			name:    "valid HTTPS with hostname (default port)",
			url:     "https://tautulli.example.com",
			wantErr: false,
		},
		{
			name:    "valid HTTPS with subdomain",
			url:     "https://tautulli.home.example.com:8181",
			wantErr: false,
		},
		// IPv6 addresses
		{
			name:    "valid HTTP with IPv6 address",
			url:     "http://[::1]:8181",
			wantErr: false,
		},
		{
			name:    "valid HTTP with full IPv6 address",
			url:     "http://[2001:db8::1]:8181",
			wantErr: false,
		},
		// Invalid URLs - Missing scheme
		{
			name:    "missing scheme",
			url:     "localhost:8181",
			wantErr: true,
			errMsg:  "scheme must be http or https",
		},
		{
			name:    "invalid scheme (ftp)",
			url:     "ftp://localhost:8181",
			wantErr: true,
			errMsg:  "scheme must be http or https, got: ftp",
		},
		{
			name:    "invalid scheme (ws)",
			url:     "ws://localhost:8181",
			wantErr: true,
			errMsg:  "scheme must be http or https, got: ws",
		},
		// Invalid URLs - Missing host
		{
			name:    "missing host",
			url:     "http://",
			wantErr: true,
			errMsg:  "host is required",
		},
		{
			name:    "missing host with path",
			url:     "http:///api/v1",
			wantErr: true,
			errMsg:  "host is required",
		},
		// Invalid URLs - Path/Query parameters
		{
			name:    "has path component",
			url:     "http://localhost:8181/api/v1",
			wantErr: true,
			errMsg:  "URL should be base URL only",
		},
		{
			name:    "has query parameters",
			url:     "http://localhost:8181?apikey=test",
			wantErr: true,
			errMsg:  "URL should not contain query parameters",
		},
		{
			name:    "has path and query",
			url:     "http://localhost:8181/api?key=value",
			wantErr: true,
			errMsg:  "URL should be base URL only",
		},
		// Edge cases
		{
			name:    "empty string",
			url:     "",
			wantErr: true,
			errMsg:  "scheme must be http or https",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTautulliURL(tt.url)

			if tt.wantErr {
				if err == nil {
					t.Errorf("validateTautulliURL(%q) expected error containing %q, got nil", tt.url, tt.errMsg)
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("validateTautulliURL(%q) error = %v, want error containing %q", tt.url, err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("validateTautulliURL(%q) unexpected error = %v", tt.url, err)
				}
			}
		})
	}
}

func TestGetSliceEnv(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		value        string
		defaultValue []string
		want         []string
	}{
		{
			name:         "single value",
			key:          "TEST_SLICE",
			value:        "value1",
			defaultValue: []string{"default"},
			want:         []string{"value1"},
		},
		{
			name:         "multiple values",
			key:          "TEST_SLICE",
			value:        "value1,value2,value3",
			defaultValue: []string{"default"},
			want:         []string{"value1", "value2", "value3"},
		},
		{
			name:         "values with spaces",
			key:          "TEST_SLICE",
			value:        " value1 , value2 , value3 ",
			defaultValue: []string{"default"},
			want:         []string{"value1", "value2", "value3"},
		},
		{
			name:         "empty value uses default",
			key:          "TEST_SLICE",
			value:        "",
			defaultValue: []string{"default"},
			want:         []string{"default"},
		},
		{
			name:         "only commas uses default",
			key:          "TEST_SLICE",
			value:        ",,,",
			defaultValue: []string{"default"},
			want:         []string{"default"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Clearenv()
			if tt.value != "" {
				_ = os.Setenv(tt.key, tt.value)
			}

			got := getSliceEnv(tt.key, tt.defaultValue)
			if len(got) != len(tt.want) {
				t.Errorf("getSliceEnv() length = %v, want %v", len(got), len(tt.want))
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("getSliceEnv()[%d] = %v, want %v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

// =====================================================
// Additional Tests for Full Coverage
// =====================================================

func TestGetEnv(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		value        string
		defaultValue string
		want         string
	}{
		{
			name:         "value set",
			key:          "TEST_ENV",
			value:        "test_value",
			defaultValue: "default",
			want:         "test_value",
		},
		{
			name:         "empty value uses default",
			key:          "TEST_ENV",
			value:        "",
			defaultValue: "default",
			want:         "default",
		},
		{
			name:         "env not set uses default",
			key:          "TEST_ENV_NOT_SET",
			value:        "",
			defaultValue: "default",
			want:         "default",
		},
		{
			name:         "whitespace value is kept",
			key:          "TEST_ENV",
			value:        "  spaces  ",
			defaultValue: "default",
			want:         "  spaces  ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Clearenv()
			if tt.value != "" {
				_ = os.Setenv(tt.key, tt.value)
			}

			got := getEnv(tt.key, tt.defaultValue)
			if got != tt.want {
				t.Errorf("getEnv() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetBoolEnv(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		value        string
		defaultValue bool
		want         bool
	}{
		{
			name:         "true string",
			key:          "TEST_BOOL",
			value:        "true",
			defaultValue: false,
			want:         true,
		},
		{
			name:         "false string",
			key:          "TEST_BOOL",
			value:        "false",
			defaultValue: true,
			want:         false,
		},
		{
			name:         "1 is true",
			key:          "TEST_BOOL",
			value:        "1",
			defaultValue: false,
			want:         true,
		},
		{
			name:         "0 is false",
			key:          "TEST_BOOL",
			value:        "0",
			defaultValue: true,
			want:         false,
		},
		{
			name:         "TRUE uppercase",
			key:          "TEST_BOOL",
			value:        "TRUE",
			defaultValue: false,
			want:         true,
		},
		{
			name:         "FALSE uppercase",
			key:          "TEST_BOOL",
			value:        "FALSE",
			defaultValue: true,
			want:         false,
		},
		{
			name:         "empty value uses default true",
			key:          "TEST_BOOL",
			value:        "",
			defaultValue: true,
			want:         true,
		},
		{
			name:         "empty value uses default false",
			key:          "TEST_BOOL",
			value:        "",
			defaultValue: false,
			want:         false,
		},
		{
			name:         "invalid value uses default",
			key:          "TEST_BOOL",
			value:        "invalid",
			defaultValue: true,
			want:         true,
		},
		{
			name:         "yes is invalid",
			key:          "TEST_BOOL",
			value:        "yes",
			defaultValue: false,
			want:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Clearenv()
			if tt.value != "" {
				_ = os.Setenv(tt.key, tt.value)
			}

			got := getBoolEnv(tt.key, tt.defaultValue)
			if got != tt.want {
				t.Errorf("getBoolEnv() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetFloatEnv(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		value        string
		defaultValue float64
		want         float64
	}{
		{
			name:         "valid float",
			key:          "TEST_FLOAT",
			value:        "3.14",
			defaultValue: 1.0,
			want:         3.14,
		},
		{
			name:         "integer value",
			key:          "TEST_FLOAT",
			value:        "42",
			defaultValue: 1.0,
			want:         42.0,
		},
		{
			name:         "negative float",
			key:          "TEST_FLOAT",
			value:        "-123.456",
			defaultValue: 1.0,
			want:         -123.456,
		},
		{
			name:         "zero value",
			key:          "TEST_FLOAT",
			value:        "0.0",
			defaultValue: 1.0,
			want:         0.0,
		},
		{
			name:         "scientific notation",
			key:          "TEST_FLOAT",
			value:        "1.5e10",
			defaultValue: 1.0,
			want:         1.5e10,
		},
		{
			name:         "empty value uses default",
			key:          "TEST_FLOAT",
			value:        "",
			defaultValue: 99.9,
			want:         99.9,
		},
		{
			name:         "invalid value uses default",
			key:          "TEST_FLOAT",
			value:        "not_a_float",
			defaultValue: 99.9,
			want:         99.9,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Clearenv()
			if tt.value != "" {
				_ = os.Setenv(tt.key, tt.value)
			}

			got := getFloatEnv(tt.key, tt.defaultValue)
			if got != tt.want {
				t.Errorf("getFloatEnv() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidatePlexURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
		errMsg  string
	}{
		// Valid URLs - HTTP
		{
			name:    "valid HTTP with hostname and port",
			url:     "http://localhost:32400",
			wantErr: false,
		},
		{
			name:    "valid HTTP with IP address and port",
			url:     "http://192.168.1.100:32400",
			wantErr: false,
		},
		{
			name:    "valid HTTP with hostname (no port)",
			url:     "http://plex.example.com",
			wantErr: false,
		},
		{
			name:    "valid HTTP with IP address (no explicit port)",
			url:     "http://192.168.1.100",
			wantErr: false,
		},
		{
			name:    "valid HTTP with trailing slash",
			url:     "http://localhost:32400/",
			wantErr: false,
		},
		// Valid URLs - HTTPS
		{
			name:    "valid HTTPS with hostname and port",
			url:     "https://plex.example.com:32400",
			wantErr: false,
		},
		{
			name:    "valid HTTPS with IP address and port",
			url:     "https://192.168.1.100:32400",
			wantErr: false,
		},
		{
			name:    "valid HTTPS with hostname (default port)",
			url:     "https://plex.example.com",
			wantErr: false,
		},
		{
			name:    "valid HTTPS with subdomain",
			url:     "https://plex.home.example.com:32400",
			wantErr: false,
		},
		// Invalid URLs - Missing scheme
		{
			name:    "missing scheme",
			url:     "localhost:32400",
			wantErr: true,
			errMsg:  "scheme must be http or https",
		},
		{
			name:    "invalid scheme (ftp)",
			url:     "ftp://localhost:32400",
			wantErr: true,
			errMsg:  "scheme must be http or https, got: ftp",
		},
		// Invalid URLs - Missing host
		{
			name:    "missing host",
			url:     "http://",
			wantErr: true,
			errMsg:  "host is required",
		},
		// Invalid URLs - Path/Query parameters
		{
			name:    "has path component",
			url:     "http://localhost:32400/web/index.html",
			wantErr: true,
			errMsg:  "URL should be base URL only",
		},
		{
			name:    "has query parameters",
			url:     "http://localhost:32400?X-Plex-Token=test",
			wantErr: true,
			errMsg:  "URL should not contain query parameters",
		},
		// Edge cases
		{
			name:    "empty string",
			url:     "",
			wantErr: true,
			errMsg:  "scheme must be http or https",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePlexURL(tt.url)

			if tt.wantErr {
				if err == nil {
					t.Errorf("validatePlexURL(%q) expected error containing %q, got nil", tt.url, tt.errMsg)
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("validatePlexURL(%q) error = %v, want error containing %q", tt.url, err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("validatePlexURL(%q) unexpected error = %v", tt.url, err)
				}
			}
		})
	}
}

func TestLoad_PlexConfiguration(t *testing.T) {
	tests := []struct {
		name    string
		envVars map[string]string
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid Plex configuration",
			envVars: map[string]string{
				"TAUTULLI_URL":        "http://localhost:8181",
				"TAUTULLI_API_KEY":    "test_api_key_12345678",
				"AUTH_MODE":           "none",
				"ENABLE_PLEX_SYNC":    "true",
				"PLEX_URL":            "http://localhost:32400",
				"PLEX_TOKEN":          "abcdefghij1234567890",
				"PLEX_SYNC_DAYS_BACK": "365",
			},
			wantErr: false,
		},
		{
			name: "Plex enabled but missing URL",
			envVars: map[string]string{
				"TAUTULLI_URL":     "http://localhost:8181",
				"TAUTULLI_API_KEY": "test_api_key_12345678",
				"AUTH_MODE":        "none",
				"ENABLE_PLEX_SYNC": "true",
				"PLEX_TOKEN":       "abcdefghij1234567890",
			},
			wantErr: true,
			errMsg:  "PLEX_URL is required when ENABLE_PLEX_SYNC=true",
		},
		{
			name: "Plex enabled but missing token",
			envVars: map[string]string{
				"TAUTULLI_URL":     "http://localhost:8181",
				"TAUTULLI_API_KEY": "test_api_key_12345678",
				"AUTH_MODE":        "none",
				"ENABLE_PLEX_SYNC": "true",
				"PLEX_URL":         "http://localhost:32400",
			},
			wantErr: true,
			errMsg:  "PLEX_TOKEN is required when ENABLE_PLEX_SYNC=true",
		},
		{
			name: "Plex token too short",
			envVars: map[string]string{
				"TAUTULLI_URL":     "http://localhost:8181",
				"TAUTULLI_API_KEY": "test_api_key_12345678",
				"AUTH_MODE":        "none",
				"ENABLE_PLEX_SYNC": "true",
				"PLEX_URL":         "http://localhost:32400",
				"PLEX_TOKEN":       "short",
			},
			wantErr: true,
			errMsg:  "PLEX_TOKEN appears invalid (too short, expected 20+ characters)",
		},
		{
			name: "Plex invalid URL format",
			envVars: map[string]string{
				"TAUTULLI_URL":     "http://localhost:8181",
				"TAUTULLI_API_KEY": "test_api_key_12345678",
				"AUTH_MODE":        "none",
				"ENABLE_PLEX_SYNC": "true",
				"PLEX_URL":         "ftp://localhost:32400",
				"PLEX_TOKEN":       "abcdefghij1234567890",
			},
			wantErr: true,
			errMsg:  "PLEX_URL is invalid",
		},
		{
			name: "Plex sync days back too low",
			envVars: map[string]string{
				"TAUTULLI_URL":        "http://localhost:8181",
				"TAUTULLI_API_KEY":    "test_api_key_12345678",
				"AUTH_MODE":           "none",
				"ENABLE_PLEX_SYNC":    "true",
				"PLEX_URL":            "http://localhost:32400",
				"PLEX_TOKEN":          "abcdefghij1234567890",
				"PLEX_SYNC_DAYS_BACK": "5",
			},
			wantErr: true,
			errMsg:  "PLEX_SYNC_DAYS_BACK must be between 7 and 3650 days",
		},
		{
			name: "Plex sync days back too high",
			envVars: map[string]string{
				"TAUTULLI_URL":        "http://localhost:8181",
				"TAUTULLI_API_KEY":    "test_api_key_12345678",
				"AUTH_MODE":           "none",
				"ENABLE_PLEX_SYNC":    "true",
				"PLEX_URL":            "http://localhost:32400",
				"PLEX_TOKEN":          "abcdefghij1234567890",
				"PLEX_SYNC_DAYS_BACK": "4000",
			},
			wantErr: true,
			errMsg:  "PLEX_SYNC_DAYS_BACK must be between 7 and 3650 days",
		},
		{
			name: "Plex disabled doesn't validate Plex config",
			envVars: map[string]string{
				"TAUTULLI_URL":     "http://localhost:8181",
				"TAUTULLI_API_KEY": "test_api_key_12345678",
				"AUTH_MODE":        "none",
				"ENABLE_PLEX_SYNC": "false",
				// No PLEX_URL or PLEX_TOKEN needed when disabled
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Clearenv()
			for k, v := range tt.envVars {
				_ = os.Setenv(k, v)
			}

			cfg, err := Load()

			if tt.wantErr {
				if err == nil {
					t.Errorf("Load() expected error containing %q, got nil", tt.errMsg)
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Load() error = %v, want error containing %q", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Load() unexpected error = %v", err)
				}
				if cfg == nil {
					t.Error("Load() returned nil config")
				}
			}
		})
	}
}

func TestLoad_ConfigValues(t *testing.T) {
	os.Clearenv()

	// Set up a valid configuration with various custom values
	envVars := map[string]string{
		"TAUTULLI_URL":                       "http://tautulli.local:8181",
		"TAUTULLI_API_KEY":                   "my_tautulli_api_key_here",
		"AUTH_MODE":                          "none",
		"DUCKDB_PATH":                        "/custom/path/db.duckdb",
		"DUCKDB_MAX_MEMORY":                  "4GB",
		"SEED_MOCK_DATA":                     "true",
		"SYNC_INTERVAL":                      "10m",
		"SYNC_LOOKBACK":                      "48h",
		"SYNC_BATCH_SIZE":                    "500",
		"SYNC_RETRY_ATTEMPTS":                "3",
		"SYNC_RETRY_DELAY":                   "5s",
		"HTTP_PORT":                          "8080",
		"HTTP_HOST":                          "127.0.0.1",
		"HTTP_TIMEOUT":                       "60s",
		"SERVER_LATITUDE":                    "40.7128",
		"SERVER_LONGITUDE":                   "-74.0060",
		"API_DEFAULT_PAGE_SIZE":              "50",
		"API_MAX_PAGE_SIZE":                  "200",
		"RATE_LIMIT_REQUESTS":                "200",
		"RATE_LIMIT_WINDOW":                  "2m",
		"DISABLE_RATE_LIMIT":                 "true",
		"CORS_ORIGINS":                       "http://localhost:3000,http://localhost:8080",
		"TRUSTED_PROXIES":                    "10.0.0.1,10.0.0.2",
		"LOG_LEVEL":                          "debug",
		"ENABLE_PLEX_SYNC":                   "true",
		"PLEX_URL":                           "http://plex.local:32400",
		"PLEX_TOKEN":                         "my_plex_token_20chars",
		"PLEX_HISTORICAL_SYNC":               "true",
		"PLEX_SYNC_DAYS_BACK":                "180",
		"PLEX_SYNC_INTERVAL":                 "12h",
		"ENABLE_PLEX_REALTIME":               "true",
		"PLEX_OAUTH_CLIENT_ID":               "oauth_client_id",
		"PLEX_OAUTH_CLIENT_SECRET":           "oauth_client_secret",
		"PLEX_OAUTH_REDIRECT_URI":            "http://localhost:3857/callback",
		"ENABLE_PLEX_TRANSCODE_MONITORING":   "true",
		"PLEX_TRANSCODE_MONITORING_INTERVAL": "15s",
		"ENABLE_BUFFER_HEALTH_MONITORING":    "true",
		"BUFFER_HEALTH_POLL_INTERVAL":        "10s",
		"BUFFER_HEALTH_CRITICAL_THRESHOLD":   "15.0",
		"BUFFER_HEALTH_RISKY_THRESHOLD":      "40.0",
	}

	for k, v := range envVars {
		_ = os.Setenv(k, v)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Verify all configuration sections using helper functions
	assertTautulliConfig(t, cfg, "http://tautulli.local:8181", "my_tautulli_api_key_here")
	assertDatabaseConfig(t, cfg, "/custom/path/db.duckdb", "4GB", true)
	assertSyncConfig(t, cfg, 10*time.Minute, 48*time.Hour, 5*time.Second, 500, 3)
	assertServerConfig(t, cfg, 8080, "127.0.0.1", 60*time.Second, 40.7128, -74.0060)
	assertAPIConfig(t, cfg, 50, 200)
	assertSecurityConfig(t, cfg, "none", 200, 2*time.Minute, true, 2, 2)
	assertLoggingConfig(t, cfg, "debug")
	assertPlexConfig(t, cfg, true, "http://plex.local:32400", "my_plex_token_20chars", true, 180, 12*time.Hour, true, "oauth_client_id", "oauth_client_secret", "http://localhost:3857/callback", true, 15*time.Second, 10*time.Second, true, 15.0, 40.0)
}

func TestLoad_DefaultValues(t *testing.T) {
	os.Clearenv()

	// Set only required values
	envVars := map[string]string{
		"TAUTULLI_URL":     "http://localhost:8181",
		"TAUTULLI_API_KEY": "test_api_key_12345678",
		"AUTH_MODE":        "none",
	}

	for k, v := range envVars {
		_ = os.Setenv(k, v)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Verify default values using helper functions
	assertDatabaseConfig(t, cfg, "/data/cartographus.duckdb", "2GB", false)
	assertIntEqual(t, cfg.Sync.BatchSize, 1000, "Sync.BatchSize", "")
	assertDurationEqual(t, cfg.Sync.Interval, 5*time.Minute, "Sync.Interval")
	assertDurationEqual(t, cfg.Sync.Lookback, 24*time.Hour, "Sync.Lookback")
	assertIntEqual(t, cfg.Server.Port, 3857, "Server.Port", "")
	assertStringEqual(t, cfg.Server.Host, "0.0.0.0", "Server.Host")
	assertAPIConfig(t, cfg, 20, 100)
	assertLoggingConfig(t, cfg, "info")
	assertBoolEqual(t, cfg.Plex.Enabled, false, "Plex.Enabled")
	assertIntEqual(t, cfg.Plex.SyncDaysBack, 365, "Plex.SyncDaysBack", "")
}

func TestValidate_AllLogLevels(t *testing.T) {
	validLevels := []string{"debug", "info", "warn", "error"}

	for _, level := range validLevels {
		t.Run("valid_"+level, func(t *testing.T) {
			os.Clearenv()
			os.Setenv("TAUTULLI_URL", "http://localhost:8181")
			os.Setenv("TAUTULLI_API_KEY", "test_api_key")
			os.Setenv("AUTH_MODE", "none")
			os.Setenv("LOG_LEVEL", level)

			cfg, err := Load()
			if err != nil {
				t.Errorf("Load() with LOG_LEVEL=%s unexpected error = %v", level, err)
			}
			if cfg.Logging.Level != level {
				t.Errorf("Logging.Level = %v, want %v", cfg.Logging.Level, level)
			}
		})
	}
}

// =====================================================
// NATS Configuration Tests
// =====================================================

func TestGetInt64Env(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		value        string
		defaultValue int64
		want         int64
	}{
		{
			name:         "valid int64",
			key:          "TEST_INT64",
			value:        "1073741824",
			defaultValue: 0,
			want:         1073741824,
		},
		{
			name:         "negative int64",
			key:          "TEST_INT64",
			value:        "-123456789",
			defaultValue: 0,
			want:         -123456789,
		},
		{
			name:         "empty value uses default",
			key:          "TEST_INT64",
			value:        "",
			defaultValue: 1073741824,
			want:         1073741824,
		},
		{
			name:         "invalid value uses default",
			key:          "TEST_INT64",
			value:        "not_a_number",
			defaultValue: 1073741824,
			want:         1073741824,
		},
		{
			name:         "large value",
			key:          "TEST_INT64",
			value:        "10737418240",
			defaultValue: 0,
			want:         10737418240,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Clearenv()
			if tt.value != "" {
				_ = os.Setenv(tt.key, tt.value)
			}

			got := getInt64Env(tt.key, tt.defaultValue)
			if got != tt.want {
				t.Errorf("getInt64Env() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateNATSURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
		errMsg  string
	}{
		// Valid URLs - nats://
		{
			name:    "valid NATS with hostname and port",
			url:     "nats://localhost:4222",
			wantErr: false,
		},
		{
			name:    "valid NATS with IP address and port",
			url:     "nats://192.168.1.100:4222",
			wantErr: false,
		},
		{
			name:    "valid NATS with hostname (no port)",
			url:     "nats://nats.example.com",
			wantErr: false,
		},
		// Valid URLs - tls://
		{
			name:    "valid TLS with hostname and port",
			url:     "tls://nats.example.com:4222",
			wantErr: false,
		},
		// Valid URLs - ws:// and wss://
		{
			name:    "valid WebSocket",
			url:     "ws://localhost:8080",
			wantErr: false,
		},
		{
			name:    "valid secure WebSocket",
			url:     "wss://nats.example.com:443",
			wantErr: false,
		},
		// Invalid URLs - Wrong scheme
		{
			name:    "invalid scheme (http)",
			url:     "http://localhost:4222",
			wantErr: true,
			errMsg:  "scheme must be nats, tls, ws, or wss",
		},
		{
			name:    "invalid scheme (https)",
			url:     "https://localhost:4222",
			wantErr: true,
			errMsg:  "scheme must be nats, tls, ws, or wss",
		},
		// Invalid URLs - Missing host
		{
			name:    "missing host",
			url:     "nats://",
			wantErr: true,
			errMsg:  "host is required",
		},
		// Edge cases
		{
			name:    "empty string",
			url:     "",
			wantErr: true,
			errMsg:  "scheme must be nats, tls, ws, or wss",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateNATSURL(tt.url)

			if tt.wantErr {
				if err == nil {
					t.Errorf("validateNATSURL(%q) expected error containing %q, got nil", tt.url, tt.errMsg)
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("validateNATSURL(%q) error = %v, want error containing %q", tt.url, err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("validateNATSURL(%q) unexpected error = %v", tt.url, err)
				}
			}
		})
	}
}

func TestLoad_NATSConfiguration(t *testing.T) {
	tests := []struct {
		name    string
		envVars map[string]string
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid NATS configuration",
			envVars: map[string]string{
				"TAUTULLI_URL":        "http://localhost:8181",
				"TAUTULLI_API_KEY":    "test_api_key_12345678",
				"AUTH_MODE":           "none",
				"NATS_ENABLED":        "true",
				"NATS_URL":            "nats://localhost:4222",
				"NATS_EMBEDDED":       "true",
				"NATS_STORE_DIR":      "/data/nats/jetstream",
				"NATS_MAX_MEMORY":     "1073741824",
				"NATS_MAX_STORE":      "10737418240",
				"NATS_RETENTION_DAYS": "7",
				"NATS_BATCH_SIZE":     "1000",
				"NATS_FLUSH_INTERVAL": "5s",
				"NATS_SUBSCRIBERS":    "4",
			},
			wantErr: false,
		},
		{
			name: "NATS invalid URL scheme",
			envVars: map[string]string{
				"TAUTULLI_URL":     "http://localhost:8181",
				"TAUTULLI_API_KEY": "test_api_key_12345678",
				"AUTH_MODE":        "none",
				"NATS_ENABLED":     "true",
				"NATS_URL":         "http://localhost:4222",
			},
			wantErr: true,
			errMsg:  "NATS_URL is invalid",
		},
		{
			name: "NATS max memory too low",
			envVars: map[string]string{
				"TAUTULLI_URL":     "http://localhost:8181",
				"TAUTULLI_API_KEY": "test_api_key_12345678",
				"AUTH_MODE":        "none",
				"NATS_ENABLED":     "true",
				"NATS_URL":         "nats://localhost:4222",
				"NATS_MAX_MEMORY":  "1000000", // Less than 64MB
			},
			wantErr: true,
			errMsg:  "NATS_MAX_MEMORY must be at least 64MB",
		},
		{
			name: "NATS max store too low",
			envVars: map[string]string{
				"TAUTULLI_URL":     "http://localhost:8181",
				"TAUTULLI_API_KEY": "test_api_key_12345678",
				"AUTH_MODE":        "none",
				"NATS_ENABLED":     "true",
				"NATS_URL":         "nats://localhost:4222",
				"NATS_MAX_MEMORY":  "67108864",
				"NATS_MAX_STORE":   "1000000", // Less than 100MB
			},
			wantErr: true,
			errMsg:  "NATS_MAX_STORE must be at least 100MB",
		},
		{
			name: "NATS retention days too low",
			envVars: map[string]string{
				"TAUTULLI_URL":        "http://localhost:8181",
				"TAUTULLI_API_KEY":    "test_api_key_12345678",
				"AUTH_MODE":           "none",
				"NATS_ENABLED":        "true",
				"NATS_URL":            "nats://localhost:4222",
				"NATS_RETENTION_DAYS": "0",
			},
			wantErr: true,
			errMsg:  "NATS_RETENTION_DAYS must be between 1 and 365",
		},
		{
			name: "NATS retention days too high",
			envVars: map[string]string{
				"TAUTULLI_URL":        "http://localhost:8181",
				"TAUTULLI_API_KEY":    "test_api_key_12345678",
				"AUTH_MODE":           "none",
				"NATS_ENABLED":        "true",
				"NATS_URL":            "nats://localhost:4222",
				"NATS_RETENTION_DAYS": "400",
			},
			wantErr: true,
			errMsg:  "NATS_RETENTION_DAYS must be between 1 and 365",
		},
		{
			name: "NATS batch size too low",
			envVars: map[string]string{
				"TAUTULLI_URL":     "http://localhost:8181",
				"TAUTULLI_API_KEY": "test_api_key_12345678",
				"AUTH_MODE":        "none",
				"NATS_ENABLED":     "true",
				"NATS_URL":         "nats://localhost:4222",
				"NATS_BATCH_SIZE":  "0",
			},
			wantErr: true,
			errMsg:  "NATS_BATCH_SIZE must be between 1 and 10000",
		},
		{
			name: "NATS batch size too high",
			envVars: map[string]string{
				"TAUTULLI_URL":     "http://localhost:8181",
				"TAUTULLI_API_KEY": "test_api_key_12345678",
				"AUTH_MODE":        "none",
				"NATS_ENABLED":     "true",
				"NATS_URL":         "nats://localhost:4222",
				"NATS_BATCH_SIZE":  "50000",
			},
			wantErr: true,
			errMsg:  "NATS_BATCH_SIZE must be between 1 and 10000",
		},
		{
			name: "NATS flush interval too short",
			envVars: map[string]string{
				"TAUTULLI_URL":        "http://localhost:8181",
				"TAUTULLI_API_KEY":    "test_api_key_12345678",
				"AUTH_MODE":           "none",
				"NATS_ENABLED":        "true",
				"NATS_URL":            "nats://localhost:4222",
				"NATS_FLUSH_INTERVAL": "100ms",
			},
			wantErr: true,
			errMsg:  "NATS_FLUSH_INTERVAL must be between 1s and 1h",
		},
		{
			name: "NATS flush interval too long",
			envVars: map[string]string{
				"TAUTULLI_URL":        "http://localhost:8181",
				"TAUTULLI_API_KEY":    "test_api_key_12345678",
				"AUTH_MODE":           "none",
				"NATS_ENABLED":        "true",
				"NATS_URL":            "nats://localhost:4222",
				"NATS_FLUSH_INTERVAL": "2h",
			},
			wantErr: true,
			errMsg:  "NATS_FLUSH_INTERVAL must be between 1s and 1h",
		},
		{
			name: "NATS subscribers too low",
			envVars: map[string]string{
				"TAUTULLI_URL":     "http://localhost:8181",
				"TAUTULLI_API_KEY": "test_api_key_12345678",
				"AUTH_MODE":        "none",
				"NATS_ENABLED":     "true",
				"NATS_URL":         "nats://localhost:4222",
				"NATS_SUBSCRIBERS": "0",
			},
			wantErr: true,
			errMsg:  "NATS_SUBSCRIBERS must be between 1 and 32",
		},
		{
			name: "NATS subscribers too high",
			envVars: map[string]string{
				"TAUTULLI_URL":     "http://localhost:8181",
				"TAUTULLI_API_KEY": "test_api_key_12345678",
				"AUTH_MODE":        "none",
				"NATS_ENABLED":     "true",
				"NATS_URL":         "nats://localhost:4222",
				"NATS_SUBSCRIBERS": "100",
			},
			wantErr: true,
			errMsg:  "NATS_SUBSCRIBERS must be between 1 and 32",
		},
		{
			name: "NATS disabled doesn't validate NATS config",
			envVars: map[string]string{
				"TAUTULLI_URL":     "http://localhost:8181",
				"TAUTULLI_API_KEY": "test_api_key_12345678",
				"AUTH_MODE":        "none",
				"NATS_ENABLED":     "false",
				// Invalid values should be ignored when disabled
				"NATS_URL":        "invalid_url",
				"NATS_BATCH_SIZE": "0",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Clearenv()
			for k, v := range tt.envVars {
				_ = os.Setenv(k, v)
			}

			cfg, err := Load()

			if tt.wantErr {
				if err == nil {
					t.Errorf("Load() expected error containing %q, got nil", tt.errMsg)
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Load() error = %v, want error containing %q", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Load() unexpected error = %v", err)
				}
				if cfg == nil {
					t.Error("Load() returned nil config")
				}
			}
		})
	}
}

func TestLoad_NATSConfigValues(t *testing.T) {
	os.Clearenv()

	// Set up a valid configuration with NATS values
	envVars := map[string]string{
		"TAUTULLI_URL":        "http://localhost:8181",
		"TAUTULLI_API_KEY":    "test_api_key_12345678",
		"AUTH_MODE":           "none",
		"NATS_ENABLED":        "true",
		"NATS_URL":            "nats://nats-server:4222",
		"NATS_EMBEDDED":       "false",
		"NATS_STORE_DIR":      "/custom/nats/store",
		"NATS_MAX_MEMORY":     "2147483648",
		"NATS_MAX_STORE":      "21474836480",
		"NATS_RETENTION_DAYS": "14",
		"NATS_BATCH_SIZE":     "500",
		"NATS_FLUSH_INTERVAL": "10s",
		"NATS_SUBSCRIBERS":    "8",
		"NATS_DURABLE_NAME":   "custom-processor",
		"NATS_QUEUE_GROUP":    "custom-group",
	}

	for k, v := range envVars {
		_ = os.Setenv(k, v)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Verify NATS configuration
	assertBoolEqual(t, cfg.NATS.Enabled, true, "NATS.Enabled")
	assertStringEqual(t, cfg.NATS.URL, "nats://nats-server:4222", "NATS.URL")
	assertBoolEqual(t, cfg.NATS.EmbeddedServer, false, "NATS.EmbeddedServer")
	assertStringEqual(t, cfg.NATS.StoreDir, "/custom/nats/store", "NATS.StoreDir")
	if cfg.NATS.MaxMemory != 2147483648 {
		t.Errorf("NATS.MaxMemory = %v, want %v", cfg.NATS.MaxMemory, 2147483648)
	}
	if cfg.NATS.MaxStore != 21474836480 {
		t.Errorf("NATS.MaxStore = %v, want %v", cfg.NATS.MaxStore, 21474836480)
	}
	assertIntEqual(t, cfg.NATS.StreamRetentionDays, 14, "NATS.StreamRetentionDays", "")
	assertIntEqual(t, cfg.NATS.BatchSize, 500, "NATS.BatchSize", "")
	assertDurationEqual(t, cfg.NATS.FlushInterval, 10*time.Second, "NATS.FlushInterval")
	assertIntEqual(t, cfg.NATS.SubscribersCount, 8, "NATS.SubscribersCount", "")
	assertStringEqual(t, cfg.NATS.DurableName, "custom-processor", "NATS.DurableName")
	assertStringEqual(t, cfg.NATS.QueueGroup, "custom-group", "NATS.QueueGroup")
}

func TestLoad_NATSDefaultValues(t *testing.T) {
	os.Clearenv()

	// Set only required values (NATS disabled by default)
	envVars := map[string]string{
		"TAUTULLI_URL":     "http://localhost:8181",
		"TAUTULLI_API_KEY": "test_api_key_12345678",
		"AUTH_MODE":        "none",
	}

	for k, v := range envVars {
		_ = os.Setenv(k, v)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Verify NATS default values (v1.48+: NATS enabled by default)
	assertBoolEqual(t, cfg.NATS.Enabled, true, "NATS.Enabled")
	assertBoolEqual(t, cfg.NATS.EventSourcing, true, "NATS.EventSourcing")
	assertStringEqual(t, cfg.NATS.URL, "nats://127.0.0.1:4222", "NATS.URL")
	assertBoolEqual(t, cfg.NATS.EmbeddedServer, true, "NATS.EmbeddedServer")
	assertStringEqual(t, cfg.NATS.StoreDir, "/data/nats/jetstream", "NATS.StoreDir")
	if cfg.NATS.MaxMemory != 1<<30 { // 1GB
		t.Errorf("NATS.MaxMemory = %v, want %v", cfg.NATS.MaxMemory, 1<<30)
	}
	if cfg.NATS.MaxStore != 10<<30 { // 10GB
		t.Errorf("NATS.MaxStore = %v, want %v", cfg.NATS.MaxStore, 10<<30)
	}
	assertIntEqual(t, cfg.NATS.StreamRetentionDays, 7, "NATS.StreamRetentionDays", "")
	assertIntEqual(t, cfg.NATS.BatchSize, 1000, "NATS.BatchSize", "")
	assertDurationEqual(t, cfg.NATS.FlushInterval, 5*time.Second, "NATS.FlushInterval")
	assertIntEqual(t, cfg.NATS.SubscribersCount, 4, "NATS.SubscribersCount", "")
	assertStringEqual(t, cfg.NATS.DurableName, "media-processor", "NATS.DurableName")
	assertStringEqual(t, cfg.NATS.QueueGroup, "processors", "NATS.QueueGroup")
}

// ========================================
// Multi-Server Config Tests (v2.1)
// ========================================

func TestGetPlexServers_SingleConfig(t *testing.T) {
	cfg := &Config{
		Plex: PlexConfig{
			Enabled:  true,
			URL:      "http://plex1:32400",
			Token:    "test-token",
			ServerID: "plex-main",
		},
	}

	servers := cfg.GetPlexServers()

	if len(servers) != 1 {
		t.Fatalf("GetPlexServers() returned %d servers, want 1", len(servers))
	}
	if servers[0].URL != "http://plex1:32400" {
		t.Errorf("GetPlexServers()[0].URL = %q, want %q", servers[0].URL, "http://plex1:32400")
	}
	if servers[0].ServerID != "plex-main" {
		t.Errorf("GetPlexServers()[0].ServerID = %q, want %q", servers[0].ServerID, "plex-main")
	}
}

func TestGetPlexServers_ArrayConfig(t *testing.T) {
	cfg := &Config{
		Plex: PlexConfig{
			Enabled:  true,
			URL:      "http://old-plex:32400",
			ServerID: "old-plex",
		},
		PlexServers: []PlexConfig{
			{
				Enabled:  true,
				URL:      "http://plex1:32400",
				Token:    "token1",
				ServerID: "plex-prod",
			},
			{
				Enabled:  true,
				URL:      "http://plex2:32400",
				Token:    "token2",
				ServerID: "plex-dev",
			},
			{
				Enabled:  false, // Disabled server should not be returned
				URL:      "http://plex3:32400",
				Token:    "token3",
				ServerID: "plex-disabled",
			},
		},
	}

	servers := cfg.GetPlexServers()

	// Should return array config (takes precedence), excluding disabled
	if len(servers) != 2 {
		t.Fatalf("GetPlexServers() returned %d servers, want 2", len(servers))
	}
	// Should NOT include the old single config
	for _, srv := range servers {
		if srv.URL == "http://old-plex:32400" {
			t.Error("GetPlexServers() should not return single config when array is configured")
		}
	}
}

func TestGetPlexServers_AutoGenerateServerID(t *testing.T) {
	cfg := &Config{
		Plex: PlexConfig{
			Enabled: true,
			URL:     "http://plex:32400",
			Token:   "token",
			// No ServerID set
		},
	}

	servers := cfg.GetPlexServers()

	if len(servers) != 1 {
		t.Fatalf("GetPlexServers() returned %d servers, want 1", len(servers))
	}
	if servers[0].ServerID == "" {
		t.Error("GetPlexServers() should auto-generate ServerID when not set")
	}
	if !strings.HasPrefix(servers[0].ServerID, "plex-") {
		t.Errorf("Auto-generated ServerID should start with 'plex-', got %q", servers[0].ServerID)
	}
}

func TestGetPlexServers_Disabled(t *testing.T) {
	cfg := &Config{
		Plex: PlexConfig{
			Enabled: false,
			URL:     "http://plex:32400",
		},
	}

	servers := cfg.GetPlexServers()

	if len(servers) != 0 {
		t.Errorf("GetPlexServers() returned %d servers, want 0 (disabled)", len(servers))
	}
}

func TestGetJellyfinServers_ArrayConfig(t *testing.T) {
	cfg := &Config{
		JellyfinServers: []JellyfinConfig{
			{
				Enabled:  true,
				URL:      "http://jellyfin1:8096",
				APIKey:   "key1",
				ServerID: "jf-1",
			},
			{
				Enabled:  true,
				URL:      "http://jellyfin2:8096",
				APIKey:   "key2",
				ServerID: "jf-2",
			},
		},
	}

	servers := cfg.GetJellyfinServers()

	if len(servers) != 2 {
		t.Fatalf("GetJellyfinServers() returned %d servers, want 2", len(servers))
	}
}

func TestGetEmbyServers_SingleConfig(t *testing.T) {
	cfg := &Config{
		Emby: EmbyConfig{
			Enabled:  true,
			URL:      "http://emby:8096",
			APIKey:   "emby-key",
			ServerID: "emby-main",
		},
	}

	servers := cfg.GetEmbyServers()

	if len(servers) != 1 {
		t.Fatalf("GetEmbyServers() returned %d servers, want 1", len(servers))
	}
	if servers[0].ServerID != "emby-main" {
		t.Errorf("GetEmbyServers()[0].ServerID = %q, want %q", servers[0].ServerID, "emby-main")
	}
}

func TestHasAnyMediaServer(t *testing.T) {
	tests := []struct {
		name string
		cfg  *Config
		want bool
	}{
		{
			name: "no servers configured",
			cfg:  &Config{},
			want: false,
		},
		{
			name: "plex enabled",
			cfg: &Config{
				Plex: PlexConfig{Enabled: true, URL: "http://plex:32400"},
			},
			want: true,
		},
		{
			name: "jellyfin enabled",
			cfg: &Config{
				Jellyfin: JellyfinConfig{Enabled: true, URL: "http://jf:8096"},
			},
			want: true,
		},
		{
			name: "emby enabled",
			cfg: &Config{
				Emby: EmbyConfig{Enabled: true, URL: "http://emby:8096"},
			},
			want: true,
		},
		{
			name: "tautulli enabled",
			cfg: &Config{
				Tautulli: TautulliConfig{Enabled: true, URL: "http://tautulli:8181"},
			},
			want: true,
		},
		{
			name: "multiple servers via array",
			cfg: &Config{
				PlexServers: []PlexConfig{
					{Enabled: true, URL: "http://plex1:32400"},
					{Enabled: true, URL: "http://plex2:32400"},
				},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cfg.HasAnyMediaServer()
			if got != tt.want {
				t.Errorf("HasAnyMediaServer() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetTotalServerCount(t *testing.T) {
	cfg := &Config{
		Plex:     PlexConfig{Enabled: true, URL: "http://plex:32400"},
		Jellyfin: JellyfinConfig{Enabled: true, URL: "http://jf:8096"},
		Emby:     EmbyConfig{Enabled: true, URL: "http://emby:8096"},
	}

	count := cfg.GetTotalServerCount()

	if count != 3 {
		t.Errorf("GetTotalServerCount() = %d, want 3", count)
	}
}

func TestGetTotalServerCount_MultiplePerPlatform(t *testing.T) {
	cfg := &Config{
		PlexServers: []PlexConfig{
			{Enabled: true, URL: "http://plex1:32400"},
			{Enabled: true, URL: "http://plex2:32400"},
		},
		JellyfinServers: []JellyfinConfig{
			{Enabled: true, URL: "http://jf1:8096"},
			{Enabled: true, URL: "http://jf2:8096"},
			{Enabled: true, URL: "http://jf3:8096"},
		},
	}

	count := cfg.GetTotalServerCount()

	if count != 5 {
		t.Errorf("GetTotalServerCount() = %d, want 5", count)
	}
}

func TestGenerateServerID_Deterministic(t *testing.T) {
	// Same URL should always generate same ID
	id1 := generateServerID("plex", "http://plex:32400")
	id2 := generateServerID("plex", "http://plex:32400")

	if id1 != id2 {
		t.Errorf("generateServerID() not deterministic: %q != %q", id1, id2)
	}

	if !strings.HasPrefix(id1, "plex-") {
		t.Errorf("generateServerID() should start with platform-, got %q", id1)
	}
}

func TestGenerateServerID_UniquePerURL(t *testing.T) {
	id1 := generateServerID("plex", "http://plex1:32400")
	id2 := generateServerID("plex", "http://plex2:32400")

	if id1 == id2 {
		t.Errorf("generateServerID() should generate different IDs for different URLs: %q == %q", id1, id2)
	}
}

func TestGenerateServerID_EmptyURL(t *testing.T) {
	id := generateServerID("plex", "")

	if id != "plex-default" {
		t.Errorf("generateServerID() with empty URL = %q, want %q", id, "plex-default")
	}
}

// TestValidateRateLimits tests rate limit bounds validation (issue 4.3)
func TestValidateRateLimits(t *testing.T) {
	tests := []struct {
		name        string
		requests    int
		window      time.Duration
		disabled    bool
		wantErr     bool
		errContains string
	}{
		{
			name:     "valid defaults",
			requests: 100,
			window:   time.Minute,
			disabled: false,
			wantErr:  false,
		},
		{
			name:     "valid minimum requests",
			requests: 1,
			window:   time.Minute,
			disabled: false,
			wantErr:  false,
		},
		{
			name:     "valid maximum requests",
			requests: 100000,
			window:   time.Minute,
			disabled: false,
			wantErr:  false,
		},
		{
			name:     "valid minimum window",
			requests: 100,
			window:   time.Second,
			disabled: false,
			wantErr:  false,
		},
		{
			name:     "valid maximum window",
			requests: 100,
			window:   time.Hour,
			disabled: false,
			wantErr:  false,
		},
		{
			name:        "invalid zero requests",
			requests:    0,
			window:      time.Minute,
			disabled:    false,
			wantErr:     true,
			errContains: "RATE_LIMIT_REQUESTS",
		},
		{
			name:        "invalid negative requests",
			requests:    -1,
			window:      time.Minute,
			disabled:    false,
			wantErr:     true,
			errContains: "RATE_LIMIT_REQUESTS",
		},
		{
			name:        "invalid too many requests",
			requests:    100001,
			window:      time.Minute,
			disabled:    false,
			wantErr:     true,
			errContains: "RATE_LIMIT_REQUESTS",
		},
		{
			name:        "invalid zero window",
			requests:    100,
			window:      0,
			disabled:    false,
			wantErr:     true,
			errContains: "RATE_LIMIT_WINDOW",
		},
		{
			name:        "invalid window too small",
			requests:    100,
			window:      500 * time.Millisecond,
			disabled:    false,
			wantErr:     true,
			errContains: "RATE_LIMIT_WINDOW",
		},
		{
			name:        "invalid window too large",
			requests:    100,
			window:      2 * time.Hour,
			disabled:    false,
			wantErr:     true,
			errContains: "RATE_LIMIT_WINDOW",
		},
		{
			name:     "disabled skips validation",
			requests: 0, // Would be invalid if enabled
			window:   0, // Would be invalid if enabled
			disabled: true,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Security: SecurityConfig{
					RateLimitReqs:     tt.requests,
					RateLimitWindow:   tt.window,
					RateLimitDisabled: tt.disabled,
				},
			}

			err := cfg.validateRateLimits()

			if tt.wantErr {
				if err == nil {
					t.Errorf("validateRateLimits() expected error containing %q, got nil", tt.errContains)
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("validateRateLimits() error = %v, want error containing %q", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("validateRateLimits() unexpected error = %v", err)
				}
			}
		})
	}
}
