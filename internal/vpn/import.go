// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package vpn

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/goccy/go-json"
	"github.com/tomtom215/cartographus/internal/logging"
)

// Importer handles importing VPN data from various sources.
type Importer struct {
	lookup *Lookup
}

// NewImporter creates a new VPN data importer.
func NewImporter(lookup *Lookup) *Importer {
	return &Importer{lookup: lookup}
}

// ImportFromFile imports VPN data from a gluetun servers.json file.
func (i *Importer) ImportFromFile(filename string) (*ImportResult, error) {
	file, err := os.Open(filename) //nolint:gosec // G304: filename is trusted input from configuration
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			logging.Error().Err(closeErr).Str("filename", filename).Msg("Error closing VPN file")
		}
	}()

	return i.ImportFromReader(file)
}

// ImportFromReader imports VPN data from a reader containing gluetun JSON.
func (i *Importer) ImportFromReader(r io.Reader) (*ImportResult, error) {
	start := time.Now()
	result := &ImportResult{}

	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read data: %w", err)
	}

	// Use RawMessage to handle mixed types (root "version" field is an int, providers are objects)
	var rawData map[string]json.RawMessage
	if err := json.Unmarshal(data, &rawData); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Clear existing data before import
	i.lookup.Clear()

	for providerName, rawProvider := range rawData {
		// Skip the root "version" field if present
		if providerName == "version" {
			continue
		}

		// Try to unmarshal as provider data
		var providerData GluetunProvider
		if err := json.Unmarshal(rawProvider, &providerData); err != nil {
			// Skip entries that don't match provider format
			continue
		}

		providerIPCount := 0
		serverCount := 0

		for idx := range providerData.Servers {
			server := &providerData.Servers[idx]
			vpnServer := &Server{
				Provider:   providerName,
				Country:    server.Country,
				Region:     server.Region,
				City:       server.City,
				Hostname:   server.Hostname,
				ServerName: server.ServerName,
				IPs:        server.IPs,
				VPNType:    server.VPN,
				ISP:        server.ISP,
				Categories: server.Categories,
				TCP:        server.TCP,
				UDP:        server.UDP,
				Number:     server.Number,
			}

			if err := i.lookup.AddServer(vpnServer); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("failed to add server %s: %v", server.Hostname, err))
				continue
			}

			providerIPCount += len(server.IPs)
			serverCount++
			result.ServersImported++
			result.IPsImported += len(server.IPs)
		}

		// Add provider metadata
		provider := &Provider{
			Name:        providerName,
			DisplayName: GetDisplayName(providerName),
			ServerCount: serverCount,
			IPCount:     providerIPCount,
			LastUpdated: time.Now(),
			Version:     providerData.Version,
			Timestamp:   providerData.Timestamp,
		}
		i.lookup.AddProvider(provider)
		result.ProvidersImported++
	}

	result.Duration = time.Since(start)
	i.lookup.stats.LastUpdated = time.Now()

	return result, nil
}

// ImportFromBytes imports VPN data from a byte slice containing gluetun JSON.
func (i *Importer) ImportFromBytes(data []byte) (*ImportResult, error) {
	start := time.Now()
	result := &ImportResult{}

	// Use RawMessage to handle mixed types (root "version" field is an int, providers are objects)
	var rawData map[string]json.RawMessage
	if err := json.Unmarshal(data, &rawData); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Clear existing data before import
	i.lookup.Clear()

	for providerName, rawProvider := range rawData {
		// Skip the root "version" field if present
		if providerName == "version" {
			continue
		}

		// Try to unmarshal as provider data
		var providerData GluetunProvider
		if err := json.Unmarshal(rawProvider, &providerData); err != nil {
			// Skip entries that don't match provider format
			continue
		}

		providerIPCount := 0
		serverCount := 0

		for idx := range providerData.Servers {
			server := &providerData.Servers[idx]
			vpnServer := &Server{
				Provider:   providerName,
				Country:    server.Country,
				Region:     server.Region,
				City:       server.City,
				Hostname:   server.Hostname,
				ServerName: server.ServerName,
				IPs:        server.IPs,
				VPNType:    server.VPN,
				ISP:        server.ISP,
				Categories: server.Categories,
				TCP:        server.TCP,
				UDP:        server.UDP,
				Number:     server.Number,
			}

			if err := i.lookup.AddServer(vpnServer); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("failed to add server %s: %v", server.Hostname, err))
				continue
			}

			providerIPCount += len(server.IPs)
			serverCount++
			result.ServersImported++
			result.IPsImported += len(server.IPs)
		}

		// Add provider metadata
		provider := &Provider{
			Name:        providerName,
			DisplayName: GetDisplayName(providerName),
			ServerCount: serverCount,
			IPCount:     providerIPCount,
			LastUpdated: time.Now(),
			Version:     providerData.Version,
			Timestamp:   providerData.Timestamp,
		}
		i.lookup.AddProvider(provider)
		result.ProvidersImported++
	}

	result.Duration = time.Since(start)
	i.lookup.stats.LastUpdated = time.Now()

	return result, nil
}

// MergeFromReader imports VPN data without clearing existing data.
// Useful for incrementally adding providers.
func (i *Importer) MergeFromReader(r io.Reader) (*ImportResult, error) {
	start := time.Now()
	result := &ImportResult{}

	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read data: %w", err)
	}

	// Use RawMessage to handle mixed types (root "version" field is an int, providers are objects)
	var rawData map[string]json.RawMessage
	if err := json.Unmarshal(data, &rawData); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	for providerName, rawProvider := range rawData {
		// Skip the root "version" field if present
		if providerName == "version" {
			continue
		}

		// Try to unmarshal as provider data
		var providerData GluetunProvider
		if err := json.Unmarshal(rawProvider, &providerData); err != nil {
			// Skip entries that don't match provider format
			continue
		}

		providerIPCount := 0
		serverCount := 0

		for idx := range providerData.Servers {
			server := &providerData.Servers[idx]
			vpnServer := &Server{
				Provider:   providerName,
				Country:    server.Country,
				Region:     server.Region,
				City:       server.City,
				Hostname:   server.Hostname,
				ServerName: server.ServerName,
				IPs:        server.IPs,
				VPNType:    server.VPN,
				ISP:        server.ISP,
				Categories: server.Categories,
				TCP:        server.TCP,
				UDP:        server.UDP,
				Number:     server.Number,
			}

			if err := i.lookup.AddServer(vpnServer); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("failed to add server %s: %v", server.Hostname, err))
				continue
			}

			providerIPCount += len(server.IPs)
			serverCount++
			result.ServersImported++
			result.IPsImported += len(server.IPs)
		}

		// Add provider metadata
		provider := &Provider{
			Name:        providerName,
			DisplayName: GetDisplayName(providerName),
			ServerCount: serverCount,
			IPCount:     providerIPCount,
			LastUpdated: time.Now(),
			Version:     providerData.Version,
			Timestamp:   providerData.Timestamp,
		}
		i.lookup.AddProvider(provider)
		result.ProvidersImported++
	}

	result.Duration = time.Since(start)
	i.lookup.stats.LastUpdated = time.Now()

	return result, nil
}
