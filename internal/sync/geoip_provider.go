// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package sync

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/goccy/go-json"

	"github.com/tomtom215/cartographus/internal/logging"
	"github.com/tomtom215/cartographus/internal/models"
)

// GeoIPProvider defines the interface for geolocation lookup services.
// Implementations can use external APIs (ip-api.com, MaxMind) or local databases.
type GeoIPProvider interface {
	// Lookup returns geolocation data for the given IP address.
	// Returns nil and an error if the lookup fails or the IP is invalid.
	Lookup(ctx context.Context, ipAddress string) (*models.Geolocation, error)

	// Name returns the provider name for logging and metrics.
	Name() string

	// IsAvailable checks if the provider is properly configured and available.
	IsAvailable() bool
}

// ========================================
// MaxMind GeoLite2 Provider
// ========================================

// MaxMindProvider implements GeoIPProvider using MaxMind's GeoLite2 web service.
// Requires a free MaxMind account and license key (same as Tautulli uses).
// Register at: https://www.maxmind.com/en/geolite2/signup
// Rate limit: 1,000 lookups/day for GeoLite2 free tier.
type MaxMindProvider struct {
	client     *http.Client
	accountID  string
	licenseKey string
	baseURL    string
}

// maxMindResponse represents the JSON response from MaxMind GeoLite2 web service
type maxMindResponse struct {
	City struct {
		Names map[string]string `json:"names"`
	} `json:"city"`
	Continent struct {
		Code  string            `json:"code"`
		Names map[string]string `json:"names"`
	} `json:"continent"`
	Country struct {
		ISOCode string            `json:"iso_code"`
		Names   map[string]string `json:"names"`
	} `json:"country"`
	Location struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
		TimeZone  string  `json:"time_zone"`
	} `json:"location"`
	Postal struct {
		Code string `json:"code"`
	} `json:"postal"`
	Subdivisions []struct {
		ISOCode string            `json:"iso_code"`
		Names   map[string]string `json:"names"`
	} `json:"subdivisions"`
	Traits struct {
		IPAddress string `json:"ip_address"`
	} `json:"traits"`
}

// maxMindErrorResponse represents error responses from MaxMind
type maxMindErrorResponse struct {
	Code  string `json:"code"`
	Error string `json:"error"`
}

// NewMaxMindProvider creates a new MaxMind GeoLite2 provider.
// accountID and licenseKey can be obtained from https://www.maxmind.com/en/account
// These are the same credentials Tautulli uses for geolocation.
func NewMaxMindProvider(accountID, licenseKey string) *MaxMindProvider {
	return &MaxMindProvider{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		accountID:  accountID,
		licenseKey: licenseKey,
		baseURL:    "https://geolite.info/geoip/v2.1/city",
	}
}

// Name returns the provider name.
func (p *MaxMindProvider) Name() string {
	return "maxmind-geolite2"
}

// IsAvailable returns true if account ID and license key are configured.
func (p *MaxMindProvider) IsAvailable() bool {
	return p.accountID != "" && p.licenseKey != ""
}

// Lookup queries MaxMind GeoLite2 web service for geolocation data.
func (p *MaxMindProvider) Lookup(ctx context.Context, ipAddress string) (*models.Geolocation, error) {
	if err := p.validateMaxMindLookup(ipAddress); err != nil {
		return nil, err
	}

	result, err := p.queryMaxMind(ctx, ipAddress)
	if err != nil {
		return nil, err
	}

	return convertMaxMindResponse(result, ipAddress), nil
}

func (p *MaxMindProvider) validateMaxMindLookup(ipAddress string) error {
	if !p.IsAvailable() {
		return fmt.Errorf("MaxMind credentials not configured")
	}

	if ip := net.ParseIP(ipAddress); ip == nil {
		return fmt.Errorf("invalid IP address: %s", ipAddress)
	}

	return nil
}

func (p *MaxMindProvider) queryMaxMind(ctx context.Context, ipAddress string) (*maxMindResponse, error) {
	url := fmt.Sprintf("%s/%s", p.baseURL, ipAddress)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// MaxMind uses Basic Auth with account ID as username and license key as password
	req.SetBasicAuth(p.accountID, p.licenseKey)
	req.Header.Set("Accept", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to query MaxMind: %w", err)
	}
	defer resp.Body.Close()

	if err := checkMaxMindResponse(resp); err != nil {
		return nil, err
	}

	var result maxMindResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode MaxMind response: %w", err)
	}

	return &result, nil
}

func checkMaxMindResponse(resp *http.Response) error {
	if resp.StatusCode == http.StatusOK {
		return nil
	}

	var errResp maxMindErrorResponse
	if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil && errResp.Error != "" {
		return fmt.Errorf("MaxMind error (%s): %s", errResp.Code, errResp.Error)
	}

	return fmt.Errorf("MaxMind returned status %d", resp.StatusCode)
}

func convertMaxMindResponse(result *maxMindResponse, ipAddress string) *models.Geolocation {
	geo := &models.Geolocation{
		IPAddress:   ipAddress,
		Latitude:    result.Location.Latitude,
		Longitude:   result.Location.Longitude,
		Country:     result.Country.Names["en"],
		LastUpdated: time.Now(),
	}

	setOptionalMaxMindFields(geo, result)
	return geo
}

func setOptionalMaxMindFields(geo *models.Geolocation, result *maxMindResponse) {
	if cityName := result.City.Names["en"]; cityName != "" {
		geo.City = &cityName
	}

	if len(result.Subdivisions) > 0 {
		if regionName := result.Subdivisions[0].Names["en"]; regionName != "" {
			geo.Region = &regionName
		}
	}

	if result.Postal.Code != "" {
		geo.PostalCode = &result.Postal.Code
	}

	if result.Location.TimeZone != "" {
		geo.Timezone = &result.Location.TimeZone
	}
}

// ========================================
// ip-api.com Provider (Free, No API Key)
// ========================================

// IPAPIProvider implements GeoIPProvider using the free ip-api.com service.
// Rate limit: 45 requests per minute (free tier, no API key required).
// For higher limits, commercial endpoints are available at pro.ip-api.com.
type IPAPIProvider struct {
	client      *http.Client
	rateLimiter *rateLimiter
	baseURL     string
}

// ipAPIResponse represents the JSON response from ip-api.com
type ipAPIResponse struct {
	Status      string  `json:"status"`      // "success" or "fail"
	Message     string  `json:"message"`     // Error message if status is "fail"
	Country     string  `json:"country"`     // Country name
	CountryCode string  `json:"countryCode"` // ISO 3166-1 alpha-2 country code
	Region      string  `json:"region"`      // Region/state code
	RegionName  string  `json:"regionName"`  // Region/state name
	City        string  `json:"city"`        // City name
	Zip         string  `json:"zip"`         // Postal code
	Lat         float64 `json:"lat"`         // Latitude
	Lon         float64 `json:"lon"`         // Longitude
	Timezone    string  `json:"timezone"`    // Timezone (e.g., "America/New_York")
	ISP         string  `json:"isp"`         // ISP name
	Query       string  `json:"query"`       // IP address queried
}

// rateLimiter implements a simple token bucket rate limiter
type rateLimiter struct {
	mu         sync.Mutex
	tokens     int
	maxTokens  int
	refillRate time.Duration
	lastRefill time.Time
}

func newRateLimiter(maxTokens int, refillRate time.Duration) *rateLimiter {
	return &rateLimiter{
		tokens:     maxTokens,
		maxTokens:  maxTokens,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

func (r *rateLimiter) Allow() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Refill tokens based on elapsed time
	now := time.Now()
	elapsed := now.Sub(r.lastRefill)
	tokensToAdd := int(elapsed / r.refillRate)
	if tokensToAdd > 0 {
		r.tokens = min(r.maxTokens, r.tokens+tokensToAdd)
		r.lastRefill = now
	}

	if r.tokens > 0 {
		r.tokens--
		return true
	}
	return false
}

// NewIPAPIProvider creates a new ip-api.com provider.
// Uses the free tier with 45 requests/minute rate limit.
func NewIPAPIProvider() *IPAPIProvider {
	return &IPAPIProvider{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		// ip-api.com allows 45 requests per minute on free tier
		rateLimiter: newRateLimiter(45, time.Minute/45),
		baseURL:     "http://ip-api.com/json",
	}
}

// Name returns the provider name.
func (p *IPAPIProvider) Name() string {
	return "ip-api.com"
}

// IsAvailable returns true (ip-api.com doesn't require API key).
func (p *IPAPIProvider) IsAvailable() bool {
	return true
}

// Lookup queries ip-api.com for geolocation data.
func (p *IPAPIProvider) Lookup(ctx context.Context, ipAddress string) (*models.Geolocation, error) {
	if err := p.validateIPAPILookup(ipAddress); err != nil {
		return nil, err
	}

	result, err := p.queryIPAPI(ctx, ipAddress)
	if err != nil {
		return nil, err
	}

	return convertIPAPIResponse(result, ipAddress), nil
}

func (p *IPAPIProvider) validateIPAPILookup(ipAddress string) error {
	if !p.rateLimiter.Allow() {
		return fmt.Errorf("rate limit exceeded for ip-api.com (45 req/min)")
	}

	if ip := net.ParseIP(ipAddress); ip == nil {
		return fmt.Errorf("invalid IP address: %s", ipAddress)
	}

	return nil
}

func (p *IPAPIProvider) queryIPAPI(ctx context.Context, ipAddress string) (*ipAPIResponse, error) {
	// Build request URL with fields we need
	// fields parameter optimizes response size and ensures we get all needed data
	url := fmt.Sprintf("%s/%s?fields=status,message,country,countryCode,region,regionName,city,zip,lat,lon,timezone,isp,query",
		p.baseURL, ipAddress)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to query ip-api.com: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ip-api.com returned status %d", resp.StatusCode)
	}

	var result ipAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode ip-api.com response: %w", err)
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("ip-api.com lookup failed: %s", result.Message)
	}

	return &result, nil
}

func convertIPAPIResponse(result *ipAPIResponse, ipAddress string) *models.Geolocation {
	geo := &models.Geolocation{
		IPAddress:   ipAddress,
		Latitude:    result.Lat,
		Longitude:   result.Lon,
		Country:     result.Country,
		LastUpdated: time.Now(),
	}

	setOptionalIPAPIFields(geo, result)
	return geo
}

func setOptionalIPAPIFields(geo *models.Geolocation, result *ipAPIResponse) {
	if result.City != "" {
		geo.City = &result.City
	}
	if result.RegionName != "" {
		geo.Region = &result.RegionName
	}
	if result.Zip != "" {
		geo.PostalCode = &result.Zip
	}
	if result.Timezone != "" {
		geo.Timezone = &result.Timezone
	}
}

// IsPrivateIP checks if the IP address is in a private/local range.
// Private IPs cannot be geolocated and should be handled specially.
func IsPrivateIP(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}

	return isInPrivateRanges(ip)
}

func isInPrivateRanges(ip net.IP) bool {
	// Check for IPv4 private ranges
	// RFC 1918: 10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16
	// Loopback: 127.0.0.0/8
	// Link-local: 169.254.0.0/16
	privateRanges := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"127.0.0.0/8",
		"169.254.0.0/16",
		"::1/128",   // IPv6 loopback
		"fc00::/7",  // IPv6 unique local
		"fe80::/10", // IPv6 link-local
	}

	for _, cidr := range privateRanges {
		if isInCIDR(ip, cidr) {
			return true
		}
	}

	return false
}

func isInCIDR(ip net.IP, cidr string) bool {
	_, network, err := net.ParseCIDR(cidr)
	if err != nil {
		return false
	}
	return network.Contains(ip)
}

// IsValidPublicIP checks if the IP address is a valid public (routable) IP.
func IsValidPublicIP(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}

	// Empty or unspecified IP
	if ip.IsUnspecified() {
		return false
	}

	// Not a private IP
	if IsPrivateIP(ipStr) {
		return false
	}

	return true
}

// CreateLocalGeolocation creates a geolocation entry for private/LAN IPs.
// These are marked with "Local Network" as the country for filtering purposes.
func CreateLocalGeolocation(ipAddress string) *models.Geolocation {
	local := "Local Network"
	return &models.Geolocation{
		IPAddress:   ipAddress,
		Latitude:    0,
		Longitude:   0,
		Country:     "Local",
		City:        &local,
		LastUpdated: time.Now(),
	}
}

// GeoIPResolver handles geolocation resolution with provider fallback and caching.
type GeoIPResolver struct {
	providers []GeoIPProvider
	db        GeolocationDB
}

// GeolocationDB defines the database interface for geolocation caching.
type GeolocationDB interface {
	GetGeolocation(ctx context.Context, ipAddress string) (*models.Geolocation, error)
	UpsertGeolocation(geo *models.Geolocation) error
}

// NewGeoIPResolver creates a new resolver with the given providers.
// Providers are tried in order until one succeeds.
func NewGeoIPResolver(db GeolocationDB, providers ...GeoIPProvider) *GeoIPResolver {
	return &GeoIPResolver{
		providers: providers,
		db:        db,
	}
}

// Resolve fetches geolocation for an IP, using cache first, then providers.
func (r *GeoIPResolver) Resolve(ctx context.Context, ipAddress string) (*models.Geolocation, error) {
	ipAddress = normalizeIPAddress(ipAddress)

	if IsPrivateIP(ipAddress) {
		return r.handlePrivateIP(ctx, ipAddress)
	}

	if geo := r.tryCache(ctx, ipAddress); geo != nil {
		return geo, nil
	}

	return r.tryProviders(ctx, ipAddress)
}

func (r *GeoIPResolver) handlePrivateIP(_ context.Context, ipAddress string) (*models.Geolocation, error) {
	logging.Debug().Str("ip", ipAddress).Msg("IP is private/LAN, creating local geolocation")
	geo := CreateLocalGeolocation(ipAddress)

	// Cache it to avoid repeated checks
	if r.db != nil {
		if err := r.db.UpsertGeolocation(geo); err != nil {
			logging.Warn().Err(err).Str("ip", ipAddress).Msg("Failed to cache local geolocation")
		}
	}

	return geo, nil
}

func (r *GeoIPResolver) tryCache(ctx context.Context, ipAddress string) *models.Geolocation {
	if r.db == nil {
		return nil
	}

	geo, err := r.db.GetGeolocation(ctx, ipAddress)
	if err == nil && geo != nil {
		return geo
	}

	return nil
}

func (r *GeoIPResolver) tryProviders(ctx context.Context, ipAddress string) (*models.Geolocation, error) {
	var lastErr error

	for _, provider := range r.providers {
		if !provider.IsAvailable() {
			continue
		}

		geo, err := provider.Lookup(ctx, ipAddress)
		if err != nil {
			logging.Debug().Err(err).Str("provider", provider.Name()).Str("ip", ipAddress).Msg("GeoIP provider failed")
			lastErr = err
			continue
		}

		r.cacheGeolocation(ipAddress, geo)
		return geo, nil
	}

	return nil, r.buildProviderError(ipAddress, lastErr)
}

func (r *GeoIPResolver) cacheGeolocation(ipAddress string, geo *models.Geolocation) {
	if r.db == nil {
		return
	}

	if err := r.db.UpsertGeolocation(geo); err != nil {
		logging.Warn().Err(err).Str("ip", ipAddress).Msg("Failed to cache geolocation")
	}
}

func (r *GeoIPResolver) buildProviderError(ipAddress string, lastErr error) error {
	if lastErr != nil {
		return fmt.Errorf("all GeoIP providers failed for %s: %w", ipAddress, lastErr)
	}
	return fmt.Errorf("no GeoIP providers available")
}

// normalizeIPAddress strips port from IP address if present
func normalizeIPAddress(ipAddr string) string {
	if strings.HasPrefix(ipAddr, "[") {
		return normalizeIPv6Address(ipAddr)
	}
	return normalizeIPv4Address(ipAddr)
}

func normalizeIPv6Address(ipAddr string) string {
	// Handle IPv6 with port: [::1]:8096 -> ::1
	if idx := strings.LastIndex(ipAddr, "]:"); idx != -1 {
		return ipAddr[1:idx]
	}
	// Remove brackets if no port
	return strings.Trim(ipAddr, "[]")
}

func normalizeIPv4Address(ipAddr string) string {
	// Handle IPv4 with port: 192.168.1.1:8096 -> 192.168.1.1
	// Only strip if it looks like host:port (single colon)
	if strings.Count(ipAddr, ":") != 1 {
		return ipAddr
	}

	if idx := strings.LastIndex(ipAddr, ":"); idx != -1 {
		return ipAddr[:idx]
	}

	return ipAddr
}
