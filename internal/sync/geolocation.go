// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package sync

import (
	"context"
	"fmt"
	"time"

	"github.com/tomtom215/cartographus/internal/logging"

	"github.com/tomtom215/cartographus/internal/models"
	"github.com/tomtom215/cartographus/internal/models/tautulli"
)

// resolveGeolocation fetches or retrieves cached geolocation for an IP address
// Returns geolocation data, creating fallback entry if fetch fails
func (m *Manager) resolveGeolocation(ctx context.Context, record *tautulli.TautulliHistoryRecord) (*models.Geolocation, error) {
	// SessionKey is now a pointer - use getEffectiveSessionKey to handle null values
	sessionKey := getEffectiveSessionKey(record)
	return m.resolveGeolocationForIP(ctx, record.IPAddress, sessionKey)
}

// resolveGeolocationForIP fetches or retrieves cached geolocation for an IP address
// This is the core geolocation resolution function used by all data sources
func (m *Manager) resolveGeolocationForIP(ctx context.Context, ipAddress, sessionKey string) (*models.Geolocation, error) {
	// Normalize IP address (strip port if present)
	ipAddress = normalizeIPAddress(ipAddress)

	// Handle private/LAN IPs first - no need to look up
	if IsPrivateIP(ipAddress) {
		logging.Debug().Str("ip", ipAddress).Str("session", sessionKey).Msg("IP is private/LAN, using local geolocation")
		geo := CreateLocalGeolocation(ipAddress)
		// Cache it to avoid repeated checks
		if cacheErr := m.db.UpsertGeolocation(geo); cacheErr != nil {
			logging.Warn().Str("ip", ipAddress).Err(cacheErr).Msg("Failed to cache local geolocation")
		}
		return geo, nil
	}

	// Try to get cached geolocation
	geo, err := m.db.GetGeolocation(ctx, ipAddress)
	if err != nil {
		logging.Warn().Str("ip", ipAddress).Err(err).Msg("Failed to get cached geolocation - will attempt fetch")
		geo = nil
	}

	// If not cached, fetch from available source
	if geo == nil {
		geo, err = m.fetchAndCacheGeolocation(ctx, ipAddress)
		if err != nil {
			// Log geolocation failure but continue processing with unknown location
			// This prevents playback events from being lost due to geolocation issues
			logging.Warn().Str("ip", ipAddress).Str("session", sessionKey).Err(err).Msg("Failed to fetch geolocation - using unknown location")

			// Create a fallback geolocation entry with unknown coordinates
			// We use 0,0 coordinates which will be filtered out in map visualizations
			geo = &models.Geolocation{
				IPAddress:   ipAddress,
				Latitude:    0,
				Longitude:   0,
				Country:     "Unknown",
				LastUpdated: time.Now(),
			}

			// Cache the unknown location to avoid repeated failed lookups
			if cacheErr := m.db.UpsertGeolocation(geo); cacheErr != nil {
				logging.Warn().Str("ip", ipAddress).Err(cacheErr).Msg("Failed to cache unknown geolocation")
			}
		}
	}

	return geo, nil
}

// fetchAndCacheGeolocation fetches geolocation from available sources and caches it.
// Priority order:
//  1. Tautulli (if enabled) - uses Tautulli's built-in GeoIP
//  2. External GeoIP service (ip-api.com) - free, no API key required
//
// When Tautulli is disabled (standalone mode), the external GeoIP service is used automatically.
// The context is used for cancellation during retry backoff waits.
func (m *Manager) fetchAndCacheGeolocation(ctx context.Context, ipAddress string) (*models.Geolocation, error) {
	var geo *models.Geolocation
	var err error

	// Try Tautulli first if available
	if m.client != nil {
		geo, err = m.fetchFromTautulli(ctx, ipAddress)
		if err == nil && geo != nil {
			return geo, nil
		}
		logging.Debug().Str("ip", ipAddress).Err(err).Msg("Tautulli GeoIP lookup failed, falling back to external service")
	}

	// Fall back to external GeoIP service (ip-api.com)
	geo, err = m.fetchFromExternalGeoIP(ctx, ipAddress)
	if err != nil {
		return nil, fmt.Errorf("all GeoIP sources failed for %s: %w", ipAddress, err)
	}

	return geo, nil
}

// fetchFromTautulli fetches geolocation from Tautulli's built-in GeoIP service.
// The context is used for cancellation during retry backoff waits.
func (m *Manager) fetchFromTautulli(ctx context.Context, ipAddress string) (*models.Geolocation, error) {
	var geoIP *tautulli.TautulliGeoIP
	var err error

	err = m.retryWithBackoff(ctx, func() error {
		geoIP, err = m.client.GetGeoIPLookup(ctx, ipAddress)
		return err
	})

	if err != nil {
		return nil, err
	}

	// Validate geolocation response - check for empty country instead of (0,0) coordinates
	// (0,0) is a valid location (Null Island in Gulf of Guinea off coast of Africa)
	if geoIP.Response.Data.Country == "" {
		return nil, fmt.Errorf("geolocation returned empty country for IP %s", ipAddress)
	}

	geo := &models.Geolocation{
		IPAddress:   ipAddress,
		Latitude:    geoIP.Response.Data.Latitude,
		Longitude:   geoIP.Response.Data.Longitude,
		Country:     geoIP.Response.Data.Country,
		LastUpdated: time.Now(),
	}

	if geoIP.Response.Data.City != "" {
		geo.City = &geoIP.Response.Data.City
	}
	if geoIP.Response.Data.Region != "" {
		geo.Region = &geoIP.Response.Data.Region
	}
	if geoIP.Response.Data.PostalCode != "" {
		geo.PostalCode = &geoIP.Response.Data.PostalCode
	}
	if geoIP.Response.Data.Timezone != "" {
		geo.Timezone = &geoIP.Response.Data.Timezone
	}
	if geoIP.Response.Data.AccuracyRadius > 0 {
		geo.AccuracyRadius = &geoIP.Response.Data.AccuracyRadius
	}

	if err := m.db.UpsertGeolocation(geo); err != nil {
		return nil, fmt.Errorf("failed to cache geolocation: %w", err)
	}

	return geo, nil
}

// fetchFromExternalGeoIP fetches geolocation from configured external GeoIP services.
// This is used when Tautulli is not available (standalone mode).
//
// Provider priority (first available wins):
//  1. MaxMind GeoLite2 (if MAXMIND_ACCOUNT_ID and MAXMIND_LICENSE_KEY configured)
//  2. ip-api.com (free, no API key required, 45 req/min limit)
//
// Users who already have Tautulli configured likely have MaxMind credentials,
// as Tautulli uses MaxMind for geolocation.
//
// The context is used for cancellation. If the context doesn't have a deadline,
// a 30-second timeout is applied to prevent indefinite hangs.
func (m *Manager) fetchFromExternalGeoIP(ctx context.Context, ipAddress string) (*models.Geolocation, error) {
	// Add timeout if context doesn't have one
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
	}

	// Build provider list based on configuration
	var providers []GeoIPProvider

	// Add MaxMind if configured (preferred - same as Tautulli uses)
	if m.cfg.GeoIP.MaxMindAccountID != "" && m.cfg.GeoIP.MaxMindLicenseKey != "" {
		maxmind := NewMaxMindProvider(m.cfg.GeoIP.MaxMindAccountID, m.cfg.GeoIP.MaxMindLicenseKey)
		providers = append(providers, maxmind)
		logging.Debug().Msg("MaxMind GeoLite2 provider configured")
	}

	// Always add ip-api.com as fallback (free, no API key required)
	providers = append(providers, NewIPAPIProvider())

	// Try each provider in order
	var lastErr error
	for _, provider := range providers {
		if !provider.IsAvailable() {
			continue
		}

		geo, err := provider.Lookup(ctx, ipAddress)
		if err != nil {
			logging.Debug().Str("provider", provider.Name()).Str("ip", ipAddress).Err(err).Msg("GeoIP provider failed")
			lastErr = err
			continue
		}

		logging.Debug().Str("provider", provider.Name()).Str("ip", ipAddress).Msg("GeoIP lookup successful")

		// Cache the result
		if err := m.db.UpsertGeolocation(geo); err != nil {
			return nil, fmt.Errorf("failed to cache geolocation: %w", err)
		}

		return geo, nil
	}

	if lastErr != nil {
		return nil, fmt.Errorf("all GeoIP providers failed for %s: %w", ipAddress, lastErr)
	}

	return nil, fmt.Errorf("no GeoIP providers available")
}
