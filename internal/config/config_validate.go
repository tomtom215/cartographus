// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package config

import (
	"fmt"
	"strings"
	"time"
)

// Validate checks that required configuration is present and valid
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

	if err := c.validateImport(); err != nil {
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

// validateTautulli validates Tautulli configuration (only if enabled)
// As of v2.0, Tautulli is OPTIONAL - Cartographus can run standalone with direct
// Plex, Jellyfin, and/or Emby integrations without requiring Tautulli.
func (c *Config) validateTautulli() error {
	if !c.Tautulli.Enabled {
		return nil // Tautulli is optional - no validation needed when disabled
	}

	if err := c.validateTautulliURL(); err != nil {
		return err
	}
	return c.validateTautulliAPIKey()
}

// validateTautulliURL validates the Tautulli URL
func (c *Config) validateTautulliURL() error {
	if c.Tautulli.URL == "" {
		return fmt.Errorf("TAUTULLI_URL is required when TAUTULLI_ENABLED=true")
	}
	if err := validateHTTPURL(c.Tautulli.URL, "TAUTULLI_URL"); err != nil {
		return fmt.Errorf("TAUTULLI_URL is invalid: %w", err)
	}
	return nil
}

// validateTautulliAPIKey validates the Tautulli API key
func (c *Config) validateTautulliAPIKey() error {
	if c.Tautulli.APIKey == "" {
		return fmt.Errorf("TAUTULLI_API_KEY is required when TAUTULLI_ENABLED=true")
	}
	return nil
}

// validatePlex validates Plex configuration (only if enabled)
func (c *Config) validatePlex() error {
	if !c.Plex.Enabled {
		return nil
	}

	if err := c.validatePlexURL(); err != nil {
		return err
	}
	if err := c.validatePlexToken(); err != nil {
		return err
	}
	return c.validatePlexSyncDaysBack()
}

// validatePlexURL validates the Plex URL
func (c *Config) validatePlexURL() error {
	if c.Plex.URL == "" {
		return fmt.Errorf("PLEX_URL is required when ENABLE_PLEX_SYNC=true")
	}
	if err := validateHTTPURL(c.Plex.URL, "PLEX_URL"); err != nil {
		return fmt.Errorf("PLEX_URL is invalid: %w", err)
	}
	return nil
}

// validatePlexToken validates the Plex token
func (c *Config) validatePlexToken() error {
	if c.Plex.Token == "" {
		return fmt.Errorf("PLEX_TOKEN is required when ENABLE_PLEX_SYNC=true")
	}
	if len(c.Plex.Token) < 20 {
		return fmt.Errorf("PLEX_TOKEN appears invalid (too short, expected 20+ characters)")
	}
	return nil
}

// validatePlexSyncDaysBack validates the Plex sync days back setting
func (c *Config) validatePlexSyncDaysBack() error {
	if c.Plex.SyncDaysBack < 7 || c.Plex.SyncDaysBack > 3650 {
		return fmt.Errorf("PLEX_SYNC_DAYS_BACK must be between 7 and 3650 days")
	}
	return nil
}

// validateNATS validates NATS configuration (only if enabled)
func (c *Config) validateNATS() error {
	if !c.NATS.Enabled {
		return nil
	}

	if err := validateNATSURL(c.NATS.URL); err != nil {
		return fmt.Errorf("NATS_URL is invalid: %w", err)
	}

	return c.validateNATSLimits()
}

// NATS limit constants
const (
	natsMinMemory      = 64 * 1024 * 1024  // 64MB
	natsMinStore       = 100 * 1024 * 1024 // 100MB
	natsMaxRetention   = 365
	natsMinRetention   = 1
	natsMaxBatchSize   = 10000
	natsMaxSubscribers = 32
	natsMinFlush       = time.Second
	natsMaxFlush       = time.Hour
)

// validateNATSLimits validates NATS storage and processing limits
func (c *Config) validateNATSLimits() error {
	validators := []func() error{
		c.validateNATSMemory,
		c.validateNATSStore,
		c.validateNATSRetention,
		c.validateNATSBatchSize,
		c.validateNATSFlushInterval,
		c.validateNATSSubscribers,
	}

	for _, validator := range validators {
		if err := validator(); err != nil {
			return err
		}
	}
	return nil
}

// validateNATSMemory validates NATS max memory setting
func (c *Config) validateNATSMemory() error {
	if c.NATS.MaxMemory < natsMinMemory {
		return fmt.Errorf("NATS_MAX_MEMORY must be at least 64MB (67108864 bytes)")
	}
	return nil
}

// validateNATSStore validates NATS max store setting
func (c *Config) validateNATSStore() error {
	if c.NATS.MaxStore < natsMinStore {
		return fmt.Errorf("NATS_MAX_STORE must be at least 100MB (104857600 bytes)")
	}
	return nil
}

// validateNATSRetention validates NATS stream retention days
func (c *Config) validateNATSRetention() error {
	if c.NATS.StreamRetentionDays < natsMinRetention || c.NATS.StreamRetentionDays > natsMaxRetention {
		return fmt.Errorf("NATS_RETENTION_DAYS must be between 1 and 365")
	}
	return nil
}

// validateNATSBatchSize validates NATS batch size setting
func (c *Config) validateNATSBatchSize() error {
	if c.NATS.BatchSize < 1 || c.NATS.BatchSize > natsMaxBatchSize {
		return fmt.Errorf("NATS_BATCH_SIZE must be between 1 and 10000")
	}
	return nil
}

// validateNATSFlushInterval validates NATS flush interval setting
func (c *Config) validateNATSFlushInterval() error {
	if c.NATS.FlushInterval < natsMinFlush || c.NATS.FlushInterval > natsMaxFlush {
		return fmt.Errorf("NATS_FLUSH_INTERVAL must be between 1s and 1h")
	}
	return nil
}

// validateNATSSubscribers validates NATS subscribers count
func (c *Config) validateNATSSubscribers() error {
	if c.NATS.SubscribersCount < 1 || c.NATS.SubscribersCount > natsMaxSubscribers {
		return fmt.Errorf("NATS_SUBSCRIBERS must be between 1 and 32")
	}
	return nil
}

// validateImport validates Import configuration (only if enabled)
func (c *Config) validateImport() error {
	if !c.Import.Enabled {
		return nil
	}

	if err := c.validateImportDBPath(); err != nil {
		return err
	}
	if err := c.validateImportBatchSize(); err != nil {
		return err
	}
	return c.validateImportResumeFromID()
}

// validateImportDBPath validates the import database path
func (c *Config) validateImportDBPath() error {
	if c.Import.DBPath == "" {
		return fmt.Errorf("IMPORT_DB_PATH is required when IMPORT_ENABLED=true")
	}
	return nil
}

// validateImportBatchSize validates the import batch size
func (c *Config) validateImportBatchSize() error {
	if c.Import.BatchSize < 1 || c.Import.BatchSize > 10000 {
		return fmt.Errorf("IMPORT_BATCH_SIZE must be between 1 and 10000")
	}
	return nil
}

// validateImportResumeFromID validates the import resume ID
func (c *Config) validateImportResumeFromID() error {
	if c.Import.ResumeFromID < 0 {
		return fmt.Errorf("IMPORT_RESUME_FROM_ID must be non-negative")
	}
	return nil
}

// validateServer validates server configuration
func (c *Config) validateServer() error {
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("HTTP_PORT must be between 1 and 65535")
	}
	return nil
}

// validateSecurity validates security configuration
func (c *Config) validateSecurity() error {
	if err := c.validateAuthMode(); err != nil {
		return err
	}

	// Validate CORS configuration - CRITICAL-005 fix
	if err := c.validateCORS(); err != nil {
		return err
	}

	// Validate rate limiting bounds (issue 4.3)
	if err := c.validateRateLimits(); err != nil {
		return err
	}

	return c.validateAuthModeConfig()
}

// validateAuthModeConfig validates configuration for the selected auth mode
func (c *Config) validateAuthModeConfig() error {
	validators := map[string]func() error{
		"jwt":   c.validateJWTAuth,
		"basic": c.validateBasicAuth,
		"oidc":  c.validateOIDCAuth,
		"plex":  c.validatePlexAuth,
		"multi": c.validateMultiAuth,
	}

	validator, exists := validators[c.Security.AuthMode]
	if !exists {
		return nil // "none" mode has no additional validation
	}

	return validator()
}

// validateCORS validates CORS configuration for security best practices.
// In production mode with authentication enabled, wildcard CORS is rejected
// as it creates a security vulnerability where any origin can access
// protected resources using stolen credentials.
func (c *Config) validateCORS() error {
	// M-01 Security Fix: Reject wildcard CORS in production with authentication
	// Wildcard CORS + authentication = credential theft vulnerability
	if c.Security.AuthMode != "none" && c.hasWildcardCORS() && c.IsProduction() {
		return fmt.Errorf("CORS_ORIGINS=* (wildcard) is not allowed in production with authentication enabled. " +
			"This creates a security vulnerability where attackers can steal credentials via malicious websites. " +
			"Either set specific origins: CORS_ORIGINS=https://yourdomain.com,https://app.yourdomain.com " +
			"or use ENVIRONMENT=development for testing purposes")
	}
	return nil
}

// hasWildcardCORS checks if CORS is configured with wildcard origins
func (c *Config) hasWildcardCORS() bool {
	for _, origin := range c.Security.CORSOrigins {
		if origin == "*" {
			return true
		}
	}
	return false
}

// ShouldWarnAboutCORS returns true if CORS configuration has security concerns
// that should be logged at startup
func (c *Config) ShouldWarnAboutCORS() bool {
	return c.Security.AuthMode != "none" && c.hasWildcardCORS()
}

// Rate limit constants
const (
	minRateLimitRequests = 1           // Minimum 1 request allowed
	maxRateLimitRequests = 100000      // Maximum 100K requests per window
	minRateLimitWindow   = time.Second // Minimum 1 second window
	maxRateLimitWindow   = time.Hour   // Maximum 1 hour window
)

// validateRateLimits validates rate limiting configuration bounds.
// Ensures rate limit values are within sensible ranges to prevent
// misconfiguration that could lead to DoS or ineffective protection.
func (c *Config) validateRateLimits() error {
	if c.Security.RateLimitDisabled {
		return nil
	}

	if err := c.validateRateLimitRequests(); err != nil {
		return err
	}
	return c.validateRateLimitWindow()
}

// validateRateLimitRequests validates the rate limit requests value
func (c *Config) validateRateLimitRequests() error {
	if c.Security.RateLimitReqs < minRateLimitRequests || c.Security.RateLimitReqs > maxRateLimitRequests {
		return fmt.Errorf("RATE_LIMIT_REQUESTS must be between %d and %d", minRateLimitRequests, maxRateLimitRequests)
	}
	return nil
}

// validateRateLimitWindow validates the rate limit window value
func (c *Config) validateRateLimitWindow() error {
	if c.Security.RateLimitWindow < minRateLimitWindow || c.Security.RateLimitWindow > maxRateLimitWindow {
		return fmt.Errorf("RATE_LIMIT_WINDOW must be between %v and %v", minRateLimitWindow, maxRateLimitWindow)
	}
	return nil
}

// validAuthModes defines the allowed authentication modes
var validAuthModes = map[string]bool{
	"none":  true,
	"jwt":   true,
	"basic": true,
	"oidc":  true,
	"plex":  true,
	"multi": true,
}

// validateAuthMode checks if auth mode is valid
func (c *Config) validateAuthMode() error {
	if !validAuthModes[c.Security.AuthMode] {
		return fmt.Errorf("AUTH_MODE must be one of: none, jwt, basic, oidc, plex, multi")
	}

	return c.validateAuthModeForEnvironment()
}

// validateAuthModeForEnvironment ensures AUTH_MODE is appropriate for the environment
func (c *Config) validateAuthModeForEnvironment() error {
	// M-02 Security Fix: Refuse to start with AUTH_MODE=none in production environment
	// This prevents accidental deployment of insecure configurations to production.
	if c.Security.AuthMode == "none" && c.IsProduction() {
		return fmt.Errorf("AUTH_MODE=none is not allowed when ENVIRONMENT=production. " +
			"Either set AUTH_MODE to a secure option (jwt, basic, oidc, plex, multi) " +
			"or use ENVIRONMENT=development for testing purposes")
	}

	return nil
}

// IsProduction returns true if the application is running in production mode.
// Production mode is determined by the ENVIRONMENT environment variable.
func (c *Config) IsProduction() bool {
	env := strings.ToLower(c.Server.Environment)
	return env == "production" || env == "prod"
}

// IsDevelopment returns true if the application is running in development mode.
func (c *Config) IsDevelopment() bool {
	env := strings.ToLower(c.Server.Environment)
	return env == "" || env == "development" || env == "dev"
}

// validateJWTAuth validates JWT authentication configuration
func (c *Config) validateJWTAuth() error {
	if err := c.validateJWTSecret(); err != nil {
		return err
	}
	return c.validateAdminCredentials("jwt")
}

// validateJWTSecret validates the JWT secret configuration
func (c *Config) validateJWTSecret() error {
	if c.Security.JWTSecret == "" {
		return fmt.Errorf("JWT_SECRET is required when AUTH_MODE is jwt")
	}
	if len(c.Security.JWTSecret) < 32 {
		return fmt.Errorf("JWT_SECRET must be at least 32 characters for security")
	}
	if containsPlaceholder(c.Security.JWTSecret) {
		return fmt.Errorf("JWT_SECRET contains a placeholder value - generate a secure secret with: openssl rand -base64 32")
	}
	return nil
}

// validateBasicAuth validates Basic authentication configuration
func (c *Config) validateBasicAuth() error {
	return c.validateAdminCredentials("basic")
}

// validateAdminCredentials validates admin username and password
func (c *Config) validateAdminCredentials(authMode string) error {
	if err := c.validateAdminUsername(authMode); err != nil {
		return err
	}
	return c.validateAdminPassword(authMode)
}

// validateAdminUsername validates the admin username configuration
func (c *Config) validateAdminUsername(authMode string) error {
	if c.Security.AdminUsername == "" {
		return fmt.Errorf("ADMIN_USERNAME is required when AUTH_MODE is %s", authMode)
	}
	return nil
}

// validateAdminPassword validates the admin password configuration
func (c *Config) validateAdminPassword(authMode string) error {
	if c.Security.AdminPassword == "" {
		return fmt.Errorf("ADMIN_PASSWORD is required when AUTH_MODE is %s", authMode)
	}
	if containsPlaceholder(c.Security.AdminPassword) {
		return fmt.Errorf("ADMIN_PASSWORD contains a placeholder value - set a secure password")
	}
	// Phase 3: Enforce password policy for admin credentials
	if err := c.validatePasswordPolicy(c.Security.AdminPassword, c.Security.AdminUsername); err != nil {
		return fmt.Errorf("ADMIN_PASSWORD: %w", err)
	}
	return nil
}

// validatePasswordPolicy validates a password against the configured password policy.
// Phase 3: Enforces strong password requirements for production security.
func (c *Config) validatePasswordPolicy(password, username string) error {
	policy := DefaultPasswordPolicy()
	return policy.ValidateWithError(password, username)
}

// validateOIDCAuth validates OIDC authentication configuration
func (c *Config) validateOIDCAuth() error {
	if err := c.validateOIDCIssuer(); err != nil {
		return err
	}
	if err := c.validateOIDCClientID(); err != nil {
		return err
	}
	return c.validateOIDCRedirectURL()
}

// validateOIDCIssuer validates the OIDC issuer URL
func (c *Config) validateOIDCIssuer() error {
	if c.Security.OIDC.IssuerURL == "" {
		return fmt.Errorf("OIDC_ISSUER_URL is required when AUTH_MODE is oidc")
	}
	if err := validateOIDCIssuerURL(c.Security.OIDC.IssuerURL); err != nil {
		return fmt.Errorf("OIDC_ISSUER_URL is invalid: %w", err)
	}
	return nil
}

// validateOIDCClientID validates the OIDC client ID
func (c *Config) validateOIDCClientID() error {
	if c.Security.OIDC.ClientID == "" {
		return fmt.Errorf("OIDC_CLIENT_ID is required when AUTH_MODE is oidc")
	}
	return nil
}

// validateOIDCRedirectURL validates the OIDC redirect URL
func (c *Config) validateOIDCRedirectURL() error {
	if c.Security.OIDC.RedirectURL == "" {
		return fmt.Errorf("OIDC_REDIRECT_URL is required when AUTH_MODE is oidc")
	}
	return nil
}

// validatePlexAuth validates Plex authentication configuration
func (c *Config) validatePlexAuth() error {
	if err := c.validatePlexClientID(); err != nil {
		return err
	}
	return c.validatePlexRedirectURI()
}

// validatePlexClientID validates the Plex client ID
func (c *Config) validatePlexClientID() error {
	if c.Security.PlexAuth.ClientID == "" {
		return fmt.Errorf("PLEX_AUTH_CLIENT_ID is required when AUTH_MODE is plex")
	}
	return nil
}

// validatePlexRedirectURI validates the Plex redirect URI
func (c *Config) validatePlexRedirectURI() error {
	if c.Security.PlexAuth.RedirectURI == "" {
		return fmt.Errorf("PLEX_AUTH_REDIRECT_URI is required when AUTH_MODE is plex")
	}
	return nil
}

// validateMultiAuth validates multi-mode authentication configuration
func (c *Config) validateMultiAuth() error {
	if c.hasAnyAuthenticator() {
		return nil
	}
	return fmt.Errorf("multi auth mode requires at least one authenticator (JWT, Basic, OIDC, or Plex)")
}

// hasAnyAuthenticator checks if at least one authenticator is properly configured
func (c *Config) hasAnyAuthenticator() bool {
	authenticators := []func() bool{
		c.hasJWTAuthenticator,
		c.hasBasicAuthenticator,
		c.hasOIDCAuthenticator,
		c.hasPlexAuthenticator,
	}

	for _, check := range authenticators {
		if check() {
			return true
		}
	}
	return false
}

// hasJWTAuthenticator checks if JWT is properly configured
func (c *Config) hasJWTAuthenticator() bool {
	return c.Security.JWTSecret != "" && len(c.Security.JWTSecret) >= 32
}

// hasBasicAuthenticator checks if Basic auth is properly configured
func (c *Config) hasBasicAuthenticator() bool {
	return c.Security.AdminUsername != "" && c.Security.AdminPassword != ""
}

// hasOIDCAuthenticator checks if OIDC is properly configured
func (c *Config) hasOIDCAuthenticator() bool {
	if c.Security.OIDC.IssuerURL == "" || c.Security.OIDC.ClientID == "" || c.Security.OIDC.RedirectURL == "" {
		return false
	}
	return validateOIDCIssuerURL(c.Security.OIDC.IssuerURL) == nil
}

// hasPlexAuthenticator checks if Plex auth is properly configured
func (c *Config) hasPlexAuthenticator() bool {
	return c.Security.PlexAuth.ClientID != "" && c.Security.PlexAuth.RedirectURI != ""
}

// validLogLevels defines the allowed log levels
var validLogLevels = map[string]bool{
	"trace": true,
	"debug": true,
	"info":  true,
	"warn":  true,
	"error": true,
}

// validLogFormats defines the allowed log formats
var validLogFormats = map[string]bool{
	"json":    true,
	"console": true,
}

// validateLogging validates logging configuration
func (c *Config) validateLogging() error {
	if err := c.validateLogLevel(); err != nil {
		return err
	}
	return c.validateLogFormat()
}

// validateLogLevel validates the log level configuration
func (c *Config) validateLogLevel() error {
	if !validLogLevels[c.Logging.Level] {
		return fmt.Errorf("LOG_LEVEL must be one of: trace, debug, info, warn, error")
	}
	return nil
}

// validateLogFormat validates the log format configuration
func (c *Config) validateLogFormat() error {
	if c.Logging.Format == "" {
		return nil
	}
	if !validLogFormats[c.Logging.Format] {
		return fmt.Errorf("LOG_FORMAT must be one of: json, console")
	}
	return nil
}

// placeholderPatterns defines common placeholder patterns that indicate
// the user forgot to set a real value. This prevents accidental deployment
// with insecure default credentials.
var placeholderPatterns = []string{
	"REPLACE",
	"CHANGEME",
	"CHANGE_ME",
	"YOUR_SECRET",
	"YOUR_PASSWORD",
	"PLACEHOLDER",
	"TODO",
	"FIXME",
	"XXX",
	"EXAMPLE",
}

// containsPlaceholder checks if a value contains common placeholder patterns
// that indicate the user forgot to set a real value. This prevents accidental
// deployment with insecure default credentials.
func containsPlaceholder(value string) bool {
	upperValue := strings.ToUpper(value)
	return containsAnyPattern(upperValue, placeholderPatterns)
}

// containsAnyPattern checks if a string contains any of the provided patterns
func containsAnyPattern(s string, patterns []string) bool {
	for _, pattern := range patterns {
		if strings.Contains(s, pattern) {
			return true
		}
	}
	return false
}
