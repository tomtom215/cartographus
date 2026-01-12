// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package vpn

import (
	"net/netip"
	"sync"
)

// Lookup provides efficient VPN IP address lookup.
// It uses a map-based approach for exact IP matches, which provides O(1) lookup
// time for the common case of individual IP addresses in the VPN database.
type Lookup struct {
	// ipv4Map stores IPv4 address to server info mapping.
	ipv4Map map[netip.Addr]*serverInfo

	// ipv6Map stores IPv6 address to server info mapping.
	ipv6Map map[netip.Addr]*serverInfo

	// providers maps provider name to provider metadata.
	providers map[string]*Provider

	// stats contains database statistics.
	stats *Stats

	// mu protects concurrent access.
	mu sync.RWMutex
}

// serverInfo contains metadata about a VPN server for lookup results.
type serverInfo struct {
	provider   string
	country    string
	city       string
	hostname   string
	serverName string
}

// NewLookup creates a new VPN IP lookup service.
func NewLookup() *Lookup {
	return &Lookup{
		ipv4Map:   make(map[netip.Addr]*serverInfo),
		ipv6Map:   make(map[netip.Addr]*serverInfo),
		providers: make(map[string]*Provider),
		stats: &Stats{
			ProviderStats: make([]Provider, 0),
		},
	}
}

// LookupIP checks if an IP address belongs to a known VPN provider.
// Returns a LookupResult with VPN details if found, or a negative result if not.
func (l *Lookup) LookupIP(ipStr string) *LookupResult {
	addr, err := netip.ParseAddr(ipStr)
	if err != nil {
		return &LookupResult{IsVPN: false, Confidence: 0}
	}

	l.mu.RLock()
	defer l.mu.RUnlock()

	var info *serverInfo
	var found bool

	if addr.Is4() {
		info, found = l.ipv4Map[addr]
	} else if addr.Is6() {
		info, found = l.ipv6Map[addr]
	}

	if !found {
		return &LookupResult{IsVPN: false, Confidence: 100}
	}

	return &LookupResult{
		IsVPN:               true,
		Provider:            info.provider,
		ProviderDisplayName: GetDisplayName(info.provider),
		ServerCountry:       info.country,
		ServerCity:          info.city,
		ServerHostname:      info.hostname,
		Confidence:          100, // Exact IP match
	}
}

// AddServer adds a VPN server's IPs to the lookup database.
func (l *Lookup) AddServer(server *Server) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	info := &serverInfo{
		provider:   server.Provider,
		country:    server.Country,
		city:       server.City,
		hostname:   server.Hostname,
		serverName: server.ServerName,
	}

	for _, ipStr := range server.IPs {
		addr, err := netip.ParseAddr(ipStr)
		if err != nil {
			continue // Skip invalid IPs
		}

		if addr.Is4() {
			l.ipv4Map[addr] = info
			l.stats.IPv4Count++
		} else if addr.Is6() {
			l.ipv6Map[addr] = info
			l.stats.IPv6Count++
		}
		l.stats.TotalIPs++
	}

	l.stats.TotalServers++

	return nil
}

// AddProvider adds or updates a provider's metadata.
func (l *Lookup) AddProvider(provider *Provider) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.providers[provider.Name] = provider

	// Update provider stats
	found := false
	for i, p := range l.stats.ProviderStats {
		if p.Name == provider.Name {
			l.stats.ProviderStats[i] = *provider
			found = true
			break
		}
	}
	if !found {
		l.stats.ProviderStats = append(l.stats.ProviderStats, *provider)
		l.stats.TotalProviders++
	}
}

// GetStats returns statistics about the VPN database.
func (l *Lookup) GetStats() *Stats {
	l.mu.RLock()
	defer l.mu.RUnlock()

	// Return a copy to avoid race conditions
	statsCopy := *l.stats
	statsCopy.ProviderStats = make([]Provider, len(l.stats.ProviderStats))
	copy(statsCopy.ProviderStats, l.stats.ProviderStats)

	return &statsCopy
}

// GetProvider returns metadata for a specific provider.
func (l *Lookup) GetProvider(name string) *Provider {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if p, ok := l.providers[name]; ok {
		providerCopy := *p
		return &providerCopy
	}
	return nil
}

// ListProviders returns all providers in the database.
func (l *Lookup) ListProviders() []Provider {
	l.mu.RLock()
	defer l.mu.RUnlock()

	providers := make([]Provider, 0, len(l.providers))
	for _, p := range l.providers {
		providers = append(providers, *p)
	}
	return providers
}

// Clear removes all data from the lookup database.
func (l *Lookup) Clear() {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.ipv4Map = make(map[netip.Addr]*serverInfo)
	l.ipv6Map = make(map[netip.Addr]*serverInfo)
	l.providers = make(map[string]*Provider)
	l.stats = &Stats{
		ProviderStats: make([]Provider, 0),
	}
}

// Count returns the number of IPs in the database.
func (l *Lookup) Count() int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return len(l.ipv4Map) + len(l.ipv6Map)
}

// ContainsIP checks if an IP is in the VPN database without returning full details.
// This is a faster check when only the boolean result is needed.
func (l *Lookup) ContainsIP(ipStr string) bool {
	addr, err := netip.ParseAddr(ipStr)
	if err != nil {
		return false
	}

	l.mu.RLock()
	defer l.mu.RUnlock()

	if addr.Is4() {
		_, found := l.ipv4Map[addr]
		return found
	}
	if addr.Is6() {
		_, found := l.ipv6Map[addr]
		return found
	}
	return false
}
