// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package vpn provides VPN IP detection and lookup services.
//
// This package enables detection of connections from known VPN providers
// to improve geolocation accuracy and flag potentially misleading analytics data.
// VPN connections can significantly skew geographic analytics since the apparent
// location reflects the VPN server, not the actual user location.
//
// Key features:
//   - Efficient IP lookup using radix tree (O(1) for exact match, O(prefix length) for CIDR)
//   - Support for both IPv4 and IPv6 addresses
//   - Import from gluetun VPN provider data (24+ providers, 10,000+ IPs)
//   - DuckDB persistence with in-memory cache for fast lookups
//   - Integration with geolocation enrichment pipeline
//
// Data sources:
//   - Primary: github.com/qdm12/gluetun servers.json
//   - Format: {provider: {version, timestamp, servers: [{country, city, ips[], ...}]}}
package vpn

import (
	"time"

	"github.com/goccy/go-json"
)

// Provider represents a VPN service provider.
type Provider struct {
	// Name is the provider identifier (e.g., "nordvpn", "expressvpn").
	Name string `json:"name"`

	// DisplayName is the human-readable provider name.
	DisplayName string `json:"display_name"`

	// Website is the provider's official website (optional).
	Website string `json:"website,omitempty"`

	// ServerCount is the number of servers from this provider.
	ServerCount int `json:"server_count"`

	// IPCount is the total number of IP addresses from this provider.
	IPCount int `json:"ip_count"`

	// LastUpdated is when the provider data was last updated.
	LastUpdated time.Time `json:"last_updated"`

	// Version is the data version from the source.
	Version int `json:"version,omitempty"`

	// Timestamp is the Unix timestamp from the source data.
	Timestamp int64 `json:"timestamp,omitempty"`
}

// Server represents a VPN server.
type Server struct {
	// Provider is the VPN provider name.
	Provider string `json:"provider"`

	// Country is the server's country.
	Country string `json:"country"`

	// Region is the geographic region (e.g., "Europe", "America").
	Region string `json:"region,omitempty"`

	// City is the server's city.
	City string `json:"city,omitempty"`

	// Hostname is the server's DNS hostname.
	Hostname string `json:"hostname,omitempty"`

	// ServerName is the internal server name (e.g., "Alderamin" for AirVPN).
	ServerName string `json:"server_name,omitempty"`

	// IPs is the list of IP addresses for this server.
	IPs []string `json:"ips"`

	// VPNType is the VPN protocol (e.g., "openvpn", "wireguard").
	VPNType string `json:"vpn_type,omitempty"`

	// ISP is the server's ISP (if available).
	ISP string `json:"isp,omitempty"`

	// Categories are server categories (e.g., "P2P", "Double VPN").
	Categories []string `json:"categories,omitempty"`

	// TCP indicates OpenVPN TCP support.
	TCP bool `json:"tcp,omitempty"`

	// UDP indicates OpenVPN UDP support.
	UDP bool `json:"udp,omitempty"`

	// Number is the server number (provider-specific).
	Number int `json:"number,omitempty"`
}

// LookupResult contains the result of a VPN IP lookup.
type LookupResult struct {
	// IsVPN indicates whether the IP belongs to a known VPN provider.
	IsVPN bool `json:"is_vpn"`

	// Provider is the VPN provider name (empty if not VPN).
	Provider string `json:"provider,omitempty"`

	// ProviderDisplayName is the human-readable provider name.
	ProviderDisplayName string `json:"provider_display_name,omitempty"`

	// ServerCountry is the VPN server's country (may differ from user's actual location).
	ServerCountry string `json:"server_country,omitempty"`

	// ServerCity is the VPN server's city.
	ServerCity string `json:"server_city,omitempty"`

	// ServerHostname is the VPN server's hostname.
	ServerHostname string `json:"server_hostname,omitempty"`

	// Confidence indicates lookup confidence (0-100).
	// Higher values indicate more certain matches (exact IP vs CIDR range).
	Confidence int `json:"confidence"`
}

// Stats contains statistics about the VPN database.
type Stats struct {
	// TotalProviders is the number of VPN providers in the database.
	TotalProviders int `json:"total_providers"`

	// TotalServers is the total number of VPN servers.
	TotalServers int `json:"total_servers"`

	// TotalIPs is the total number of IP addresses.
	TotalIPs int `json:"total_ips"`

	// IPv4Count is the number of IPv4 addresses.
	IPv4Count int `json:"ipv4_count"`

	// IPv6Count is the number of IPv6 addresses.
	IPv6Count int `json:"ipv6_count"`

	// LastUpdated is when the database was last updated.
	LastUpdated time.Time `json:"last_updated"`

	// ProviderStats contains per-provider statistics.
	ProviderStats []Provider `json:"provider_stats,omitempty"`
}

// ImportResult contains the result of importing VPN data.
type ImportResult struct {
	// ProvidersImported is the number of providers imported.
	ProvidersImported int `json:"providers_imported"`

	// ServersImported is the number of servers imported.
	ServersImported int `json:"servers_imported"`

	// IPsImported is the number of IP addresses imported.
	IPsImported int `json:"ips_imported"`

	// Errors contains any non-fatal errors encountered during import.
	Errors []string `json:"errors,omitempty"`

	// Duration is how long the import took.
	Duration time.Duration `json:"duration"`
}

// GluetunData represents the root structure of gluetun's servers.json.
type GluetunData map[string]GluetunProvider

// GluetunProvider represents a provider's data in gluetun format.
type GluetunProvider struct {
	Version   int             `json:"version"`
	Timestamp int64           `json:"timestamp"`
	Servers   []GluetunServer `json:"servers"`
}

// GluetunServer represents a server entry in gluetun format.
type GluetunServer struct {
	VPN        string   `json:"vpn,omitempty"`
	Country    string   `json:"country"`
	Region     string   `json:"region,omitempty"`
	City       string   `json:"city,omitempty"`
	ISP        string   `json:"isp,omitempty"`
	Owned      bool     `json:"owned,omitempty"`
	Number     int      `json:"number,omitempty"`
	ServerName string   `json:"server_name,omitempty"`
	Hostname   string   `json:"hostname,omitempty"`
	TCP        bool     `json:"tcp,omitempty"`
	UDP        bool     `json:"udp,omitempty"`
	WgPubKey   string   `json:"wgpubkey,omitempty"`
	IPs        []string `json:"ips"`
	Categories []string `json:"categories,omitempty"`
}

// Config holds configuration for the VPN lookup service.
type Config struct {
	// Enabled controls whether VPN detection is active.
	Enabled bool `json:"enabled"`

	// CacheSize is the maximum number of lookup results to cache.
	CacheSize int `json:"cache_size"`

	// DataFile is the path to the gluetun servers.json file (optional).
	// If empty, data is loaded from the database only.
	DataFile string `json:"data_file,omitempty"`

	// AutoUpdate enables automatic data updates (future feature).
	AutoUpdate bool `json:"auto_update"`

	// UpdateInterval is how often to check for updates (future feature).
	UpdateInterval time.Duration `json:"update_interval,omitempty"`
}

// DefaultConfig returns sensible defaults for VPN detection.
func DefaultConfig() *Config {
	return &Config{
		Enabled:        true,
		CacheSize:      10000,
		AutoUpdate:     false,
		UpdateInterval: 24 * time.Hour,
	}
}

// providerDisplayNames maps provider identifiers to human-readable names.
var providerDisplayNames = map[string]string{
	"airvpn":         "AirVPN",
	"cyberghost":     "CyberGhost",
	"expressvpn":     "ExpressVPN",
	"fastestvpn":     "FastestVPN",
	"hidemyass":      "HideMyAss",
	"ipvanish":       "IPVanish",
	"ivpn":           "IVPN",
	"mullvad":        "Mullvad",
	"nordvpn":        "NordVPN",
	"perfectprivacy": "Perfect Privacy",
	"privado":        "Privado VPN",
	"privatevpn":     "PrivateVPN",
	"protonvpn":      "ProtonVPN",
	"purevpn":        "PureVPN",
	"slickvpn":       "SlickVPN",
	"surfshark":      "Surfshark",
	"torguard":       "TorGuard",
	"vpnunlimited":   "VPN Unlimited",
	"vyprvpn":        "VyprVPN",
	"wevpn":          "WeVPN",
	"windscribe":     "Windscribe",
	"pia":            "Private Internet Access",
}

// GetDisplayName returns the human-readable name for a provider.
func GetDisplayName(provider string) string {
	if name, ok := providerDisplayNames[provider]; ok {
		return name
	}
	return provider
}

// MarshalJSON implements custom JSON marshaling for Duration.
func (c Config) MarshalJSON() ([]byte, error) {
	type Alias Config
	return json.Marshal(&struct {
		Alias
		UpdateInterval string `json:"update_interval,omitempty"`
	}{
		Alias:          Alias(c),
		UpdateInterval: c.UpdateInterval.String(),
	})
}
