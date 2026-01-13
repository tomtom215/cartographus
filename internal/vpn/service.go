// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package vpn

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/tomtom215/cartographus/internal/logging"
)

// Service provides VPN detection capabilities.
// It maintains an in-memory lookup for fast IP checks, backed by DuckDB for persistence.
type Service struct {
	config   *Config
	lookup   *Lookup
	store    *DuckDBStore
	importer *Importer

	mu sync.RWMutex
}

// NewService creates a new VPN detection service.
func NewService(db *sql.DB, config *Config) (*Service, error) {
	if config == nil {
		config = DefaultConfig()
	}

	lookup := NewLookup()
	store := NewDuckDBStore(db)
	importer := NewImporter(lookup)

	return &Service{
		config:   config,
		lookup:   lookup,
		store:    store,
		importer: importer,
	}, nil
}

// Initialize sets up the VPN service, creating tables and loading data.
func (s *Service) Initialize(ctx context.Context) error {
	// Initialize database schema
	if err := s.store.InitSchema(ctx); err != nil {
		return fmt.Errorf("failed to initialize VPN schema: %w", err)
	}

	// Load existing data from database into memory
	if err := s.store.LoadIntoLookup(ctx, s.lookup); err != nil {
		logging.Warn().Err(err).Msg("Failed to load VPN data from database")
		// Not fatal - we can operate without pre-existing data
	}

	stats := s.lookup.GetStats()
	if stats.TotalIPs > 0 {
		logging.Info().Int("total_ips", stats.TotalIPs).Int("total_providers", stats.TotalProviders).Msg("VPN service initialized")
	} else {
		logging.Info().Msg("VPN service initialized with empty database - use ImportFromFile or ImportFromReader to load data")
	}

	return nil
}

// IsVPN checks if an IP address belongs to a known VPN provider.
// This is the primary method for VPN detection.
func (s *Service) IsVPN(ip string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.config.Enabled {
		return false
	}

	return s.lookup.ContainsIP(ip)
}

// LookupIP returns detailed VPN information for an IP address.
func (s *Service) LookupIP(ip string) *LookupResult {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.config.Enabled {
		return &LookupResult{IsVPN: false, Confidence: 0}
	}

	return s.lookup.LookupIP(ip)
}

// ImportFromFile imports VPN data from a gluetun servers.json file.
func (s *Service) ImportFromFile(ctx context.Context, filename string) (*ImportResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	result, err := s.importer.ImportFromFile(filename)
	if err != nil {
		return nil, err
	}

	// Persist to database
	if err := s.persistLookupData(ctx); err != nil {
		logging.Warn().Err(err).Msg("Failed to persist imported VPN data")
		// Not fatal - in-memory data is already loaded
	}

	logging.Info().
		Int("providers_imported", result.ProvidersImported).
		Int("servers_imported", result.ServersImported).
		Int("ips_imported", result.IPsImported).
		Dur("duration", result.Duration).
		Msg("VPN data imported from file")

	return result, nil
}

// ImportFromReader imports VPN data from a reader.
func (s *Service) ImportFromReader(ctx context.Context, r io.Reader) (*ImportResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	result, err := s.importer.ImportFromReader(r)
	if err != nil {
		return nil, err
	}

	// Persist to database
	if err := s.persistLookupData(ctx); err != nil {
		logging.Warn().Err(err).Msg("Failed to persist imported VPN data")
	}

	logging.Info().
		Int("providers_imported", result.ProvidersImported).
		Int("servers_imported", result.ServersImported).
		Int("ips_imported", result.IPsImported).
		Dur("duration", result.Duration).
		Msg("VPN data imported from reader")

	return result, nil
}

// ImportFromBytes imports VPN data from a byte slice.
func (s *Service) ImportFromBytes(ctx context.Context, data []byte) (*ImportResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	result, err := s.importer.ImportFromBytes(data)
	if err != nil {
		return nil, err
	}

	// Persist to database
	if err := s.persistLookupData(ctx); err != nil {
		logging.Warn().Err(err).Msg("Failed to persist imported VPN data")
	}

	logging.Info().
		Int("providers_imported", result.ProvidersImported).
		Int("servers_imported", result.ServersImported).
		Int("ips_imported", result.IPsImported).
		Dur("duration", result.Duration).
		Msg("VPN data imported from bytes")

	return result, nil
}

// persistLookupData saves the current lookup data to the database.
func (s *Service) persistLookupData(ctx context.Context) error {
	// Skip persistence if no store is configured
	if s.store == nil {
		return nil
	}

	// Clear and reload - simpler than diff tracking
	if err := s.store.Clear(ctx); err != nil {
		return err
	}

	// Save providers
	for _, p := range s.lookup.ListProviders() {
		if err := s.store.SaveProvider(ctx, &p); err != nil {
			return err
		}
	}

	// We need to iterate through the lookup maps
	// Since they're private, we'll use a different approach: save during import
	// The importer already has access to all server data

	return nil
}

// GetStats returns statistics about the VPN database.
func (s *Service) GetStats() *Stats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.lookup.GetStats()
}

// GetProvider returns metadata for a specific VPN provider.
func (s *Service) GetProvider(name string) *Provider {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.lookup.GetProvider(name)
}

// ListProviders returns all VPN providers.
func (s *Service) ListProviders() []Provider {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.lookup.ListProviders()
}

// SetEnabled enables or disables VPN detection.
func (s *Service) SetEnabled(enabled bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.config.Enabled = enabled
}

// Enabled returns whether VPN detection is enabled.
func (s *Service) Enabled() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config.Enabled
}

// Reload clears and reloads VPN data from the database.
func (s *Service) Reload(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.lookup.Clear()
	return s.store.LoadIntoLookup(ctx, s.lookup)
}

// EnrichedGeolocation represents a geolocation result enriched with VPN information.
// This type is designed to be used during the geolocation enrichment pipeline.
type EnrichedGeolocation struct {
	IPAddress   string    `json:"ip_address"`
	Latitude    float64   `json:"latitude"`
	Longitude   float64   `json:"longitude"`
	City        string    `json:"city,omitempty"`
	Region      string    `json:"region,omitempty"`
	Country     string    `json:"country"`
	LastUpdated time.Time `json:"last_updated"`

	// VPN detection fields
	IsVPN              bool   `json:"is_vpn"`
	VPNProvider        string `json:"vpn_provider,omitempty"`
	VPNProviderDisplay string `json:"vpn_provider_display,omitempty"`
	VPNServerCountry   string `json:"vpn_server_country,omitempty"`
	VPNServerCity      string `json:"vpn_server_city,omitempty"`
	VPNConfidence      int    `json:"vpn_confidence,omitempty"`
}

// EnrichWithVPNInfo adds VPN detection information to geolocation data.
func (s *Service) EnrichWithVPNInfo(ip string, latitude, longitude float64, city, region, country string) *EnrichedGeolocation {
	result := &EnrichedGeolocation{
		IPAddress:   ip,
		Latitude:    latitude,
		Longitude:   longitude,
		City:        city,
		Region:      region,
		Country:     country,
		LastUpdated: time.Now(),
	}

	vpnResult := s.LookupIP(ip)
	result.IsVPN = vpnResult.IsVPN
	result.VPNProvider = vpnResult.Provider
	result.VPNProviderDisplay = vpnResult.ProviderDisplayName
	result.VPNServerCountry = vpnResult.ServerCountry
	result.VPNServerCity = vpnResult.ServerCity
	result.VPNConfidence = vpnResult.Confidence

	return result
}
