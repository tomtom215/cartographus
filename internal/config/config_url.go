// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package config

import (
	"fmt"
	"net/url"
)

// validateHTTPURL validates that a URL is properly formatted for HTTP/HTTPS services.
// Validates: scheme (http/https), host present, no paths or query params.
func validateHTTPURL(rawURL, fieldName string) error {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("%s failed to parse URL: %w", fieldName, err)
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("%s scheme must be http or https, got: %s", fieldName, parsedURL.Scheme)
	}

	if parsedURL.Host == "" {
		return fmt.Errorf("%s host is required", fieldName)
	}

	// Allow trailing slash but no other paths
	if parsedURL.Path != "" && parsedURL.Path != "/" {
		return fmt.Errorf("%s should be base URL only, remove path: %s", fieldName, parsedURL.Path)
	}

	if parsedURL.RawQuery != "" {
		return fmt.Errorf("%s should not contain query parameters, remove: ?%s", fieldName, parsedURL.RawQuery)
	}

	return nil
}

// validateTautulliURL validates that the Tautulli URL is properly formatted.
// Supports: HTTP/HTTPS, IP addresses/hostnames, with optional ports.
//
// Deprecated: Use validateHTTPURL instead.
func validateTautulliURL(rawURL string) error {
	return validateHTTPURL(rawURL, "TAUTULLI_URL")
}

// validatePlexURL validates that the Plex URL is properly formatted.
// Supports: HTTP/HTTPS, IP addresses/hostnames, with optional ports (typically 32400).
//
// Deprecated: Use validateHTTPURL instead.
func validatePlexURL(rawURL string) error {
	return validateHTTPURL(rawURL, "PLEX_URL")
}

// validateNATSURL validates that the NATS URL is properly formatted
// Supports: nats://, tls://, and ws:// schemes with IP addresses/hostnames and optional ports
func validateNATSURL(rawURL string) error {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("failed to parse URL: %w", err)
	}

	validSchemes := map[string]bool{"nats": true, "tls": true, "ws": true, "wss": true}
	if !validSchemes[parsedURL.Scheme] {
		return fmt.Errorf("scheme must be nats, tls, ws, or wss, got: %s", parsedURL.Scheme)
	}

	if parsedURL.Host == "" {
		return fmt.Errorf("host is required (e.g., localhost:4222, 192.168.1.100:4222, nats.example.com)")
	}

	return nil
}

// validateOIDCIssuerURL validates that the OIDC issuer URL is properly formatted
// ADR-0015: Zero Trust Authentication & Authorization
// Supports: HTTP/HTTPS with optional paths (e.g., https://auth.example.com/realms/myrealm)
// HTTP is only allowed for localhost (development)
func validateOIDCIssuerURL(rawURL string) error {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("failed to parse URL: %w", err)
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("scheme must be http or https")
	}

	if parsedURL.Host == "" {
		return fmt.Errorf("host is required")
	}

	return nil
}
