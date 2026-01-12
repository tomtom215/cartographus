// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package detection

import (
	"context"
	"math"
	"time"

	"github.com/goccy/go-json"
)

// CoordinateEpsilon is the threshold for considering coordinates as effectively zero.
// DETERMINISM: This provides a consistent epsilon for coordinate comparisons across
// all detection rules. A coordinate is considered "unknown" (sentinel value 0,0) if
// both latitude and longitude are within this epsilon of zero.
//
// Value rationale: 1e-7 degrees ≈ 1.1cm at equator, which is well below GPS accuracy
// and any meaningful coordinate difference, but provides reliable float comparison.
const CoordinateEpsilon = 1e-7

// IsUnknownLocation returns true if the coordinates represent an unknown location.
// DETERMINISM: Uses epsilon comparison instead of direct float equality to handle
// floating-point precision issues. The sentinel value (0, 0) is used to indicate
// that geolocation data is unavailable.
//
// This should be used instead of direct comparison like `lat == 0 && lon == 0`
// which is non-deterministic due to IEEE 754 floating-point representation.
func IsUnknownLocation(lat, lon float64) bool {
	return math.Abs(lat) < CoordinateEpsilon && math.Abs(lon) < CoordinateEpsilon
}

// HasValidCoordinates returns true if the coordinates are valid (not unknown).
// This is the inverse of IsUnknownLocation for readability.
func HasValidCoordinates(lat, lon float64) bool {
	return !IsUnknownLocation(lat, lon)
}

// RuleType identifies the type of detection rule.
type RuleType string

const (
	// RuleTypeImpossibleTravel detects implausible geographic transitions.
	RuleTypeImpossibleTravel RuleType = "impossible_travel"

	// RuleTypeConcurrentStreams enforces per-user stream limits.
	RuleTypeConcurrentStreams RuleType = "concurrent_streams"

	// RuleTypeDeviceVelocity flags devices appearing from multiple IPs rapidly.
	RuleTypeDeviceVelocity RuleType = "device_velocity"

	// RuleTypeGeoRestriction blocks streaming from specified countries.
	RuleTypeGeoRestriction RuleType = "geo_restriction"

	// RuleTypeSimultaneousLocations flags same account from multiple cities.
	RuleTypeSimultaneousLocations RuleType = "simultaneous_locations"
)

// Severity indicates the severity level of an alert.
type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityCritical Severity = "critical"
)

// Rule represents a detection rule configuration.
type Rule struct {
	ID        int64           `json:"id"`
	RuleType  RuleType        `json:"rule_type"`
	Name      string          `json:"name"`
	Enabled   bool            `json:"enabled"`
	Config    json.RawMessage `json:"config"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

// ImpossibleTravelConfig configures the impossible travel detector.
type ImpossibleTravelConfig struct {
	// MaxSpeedKmH is the maximum plausible travel speed (default: 900 km/h for flights).
	MaxSpeedKmH float64 `json:"max_speed_kmh"`

	// MinDistanceKm is the minimum distance to trigger (avoids false positives for nearby locations).
	MinDistanceKm float64 `json:"min_distance_km"`

	// MinTimeDeltaMinutes is the minimum time between events to consider.
	MinTimeDeltaMinutes int `json:"min_time_delta_minutes"`

	// Severity for generated alerts.
	Severity Severity `json:"severity"`
}

// DefaultImpossibleTravelConfig returns sensible defaults.
func DefaultImpossibleTravelConfig() ImpossibleTravelConfig {
	return ImpossibleTravelConfig{
		MaxSpeedKmH:         900, // Commercial flight speed
		MinDistanceKm:       100, // Ignore close locations
		MinTimeDeltaMinutes: 5,   // Ignore events within 5 minutes
		Severity:            SeverityCritical,
	}
}

// ConcurrentStreamsConfig configures the concurrent streams detector.
type ConcurrentStreamsConfig struct {
	// DefaultLimit is the default maximum concurrent streams per user.
	DefaultLimit int `json:"default_limit"`

	// UserLimits allows per-user overrides.
	UserLimits map[int]int `json:"user_limits,omitempty"`

	// Severity for generated alerts.
	Severity Severity `json:"severity"`
}

// DefaultConcurrentStreamsConfig returns sensible defaults.
func DefaultConcurrentStreamsConfig() ConcurrentStreamsConfig {
	return ConcurrentStreamsConfig{
		DefaultLimit: 3,
		UserLimits:   make(map[int]int),
		Severity:     SeverityWarning,
	}
}

// DeviceVelocityConfig configures the device velocity detector.
type DeviceVelocityConfig struct {
	// WindowMinutes is the time window to track IP changes.
	WindowMinutes int `json:"window_minutes"`

	// MaxUniqueIPs is the maximum unique IPs allowed within the window.
	MaxUniqueIPs int `json:"max_unique_ips"`

	// Severity for generated alerts.
	Severity Severity `json:"severity"`
}

// DefaultDeviceVelocityConfig returns sensible defaults.
func DefaultDeviceVelocityConfig() DeviceVelocityConfig {
	return DeviceVelocityConfig{
		WindowMinutes: 5,
		MaxUniqueIPs:  3,
		Severity:      SeverityWarning,
	}
}

// GeoRestrictionConfig configures geographic restrictions.
type GeoRestrictionConfig struct {
	// BlockedCountries is a list of ISO country codes to block.
	BlockedCountries []string `json:"blocked_countries"`

	// AllowedCountries is a list of ISO country codes to allow (whitelist mode).
	// If non-empty, only these countries are allowed.
	AllowedCountries []string `json:"allowed_countries,omitempty"`

	// Severity for generated alerts.
	Severity Severity `json:"severity"`
}

// DefaultGeoRestrictionConfig returns sensible defaults.
func DefaultGeoRestrictionConfig() GeoRestrictionConfig {
	return GeoRestrictionConfig{
		BlockedCountries: []string{},
		AllowedCountries: []string{},
		Severity:         SeverityWarning,
	}
}

// SimultaneousLocationsConfig configures simultaneous locations detection.
type SimultaneousLocationsConfig struct {
	// WindowMinutes is the time window to consider sessions simultaneous.
	WindowMinutes int `json:"window_minutes"`

	// MinDistanceKm is the minimum distance between locations to trigger.
	MinDistanceKm float64 `json:"min_distance_km"`

	// Severity for generated alerts.
	Severity Severity `json:"severity"`
}

// DefaultSimultaneousLocationsConfig returns sensible defaults.
func DefaultSimultaneousLocationsConfig() SimultaneousLocationsConfig {
	return SimultaneousLocationsConfig{
		WindowMinutes: 30,
		MinDistanceKm: 50, // Different cities
		Severity:      SeverityCritical,
	}
}

// Alert represents a detection alert.
type Alert struct {
	ID             int64           `json:"id"`
	RuleType       RuleType        `json:"rule_type"`
	UserID         int             `json:"user_id"`
	Username       string          `json:"username,omitempty"`
	ServerID       string          `json:"server_id,omitempty"` // v2.1: Multi-server support - server that triggered alert
	MachineID      string          `json:"machine_id,omitempty"`
	IPAddress      string          `json:"ip_address,omitempty"`
	Severity       Severity        `json:"severity"`
	Title          string          `json:"title"`
	Message        string          `json:"message"`
	Metadata       json.RawMessage `json:"metadata,omitempty"`
	Acknowledged   bool            `json:"acknowledged"`
	AcknowledgedBy string          `json:"acknowledged_by,omitempty"`
	AcknowledgedAt *time.Time      `json:"acknowledged_at,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
}

// ImpossibleTravelMetadata contains details for impossible travel alerts.
type ImpossibleTravelMetadata struct {
	FromCity       string    `json:"from_city"`
	FromCountry    string    `json:"from_country"`
	FromLatitude   float64   `json:"from_latitude"`
	FromLongitude  float64   `json:"from_longitude"`
	FromTimestamp  time.Time `json:"from_timestamp"`
	ToCity         string    `json:"to_city"`
	ToCountry      string    `json:"to_country"`
	ToLatitude     float64   `json:"to_latitude"`
	ToLongitude    float64   `json:"to_longitude"`
	ToTimestamp    time.Time `json:"to_timestamp"`
	DistanceKm     float64   `json:"distance_km"`
	TimeDeltaMins  float64   `json:"time_delta_mins"`
	RequiredSpeedK float64   `json:"required_speed_kmh"`
}

// ConcurrentStreamsMetadata contains details for concurrent stream alerts.
type ConcurrentStreamsMetadata struct {
	ActiveStreams int      `json:"active_streams"`
	StreamLimit   int      `json:"stream_limit"`
	SessionKeys   []string `json:"session_keys"`
}

// DeviceVelocityMetadata contains details for device velocity alerts.
type DeviceVelocityMetadata struct {
	MachineID   string    `json:"machine_id"`
	IPAddresses []string  `json:"ip_addresses"`
	WindowStart time.Time `json:"window_start"`
	WindowEnd   time.Time `json:"window_end"`
}

// TrustScore represents a user's trust score.
type TrustScore struct {
	UserID          int        `json:"user_id"`
	Username        string     `json:"username,omitempty"`
	Score           int        `json:"score"` // 0-100
	ViolationsCount int        `json:"violations_count"`
	LastViolationAt *time.Time `json:"last_violation_at,omitempty"`
	Restricted      bool       `json:"restricted"` // Auto-restricted at score < 50
	UpdatedAt       time.Time  `json:"updated_at"`
}

// Detector is the interface that all detection rules implement.
type Detector interface {
	// Type returns the rule type this detector handles.
	Type() RuleType

	// Check evaluates the event against the detection rule.
	// Returns an alert if a violation is detected, nil otherwise.
	Check(ctx context.Context, event *DetectionEvent) (*Alert, error)

	// Configure updates the detector configuration.
	Configure(config json.RawMessage) error

	// Enabled returns whether this detector is currently enabled.
	Enabled() bool

	// SetEnabled enables or disables the detector.
	SetEnabled(enabled bool)
}

// DetectionEvent is the event format consumed by detectors.
// It combines playback event data with geolocation information.
type DetectionEvent struct {
	// Event identification
	EventID        string    `json:"event_id"`
	SessionKey     string    `json:"session_key"`
	CorrelationKey string    `json:"correlation_key,omitempty"`
	EventType      string    `json:"event_type"` // start, stop, pause, resume
	Source         string    `json:"source"`     // plex, tautulli, jellyfin
	ServerID       string    `json:"server_id"`  // v2.1: Multi-server support - identifies source server instance
	Timestamp      time.Time `json:"timestamp"`

	// User information
	UserID       int    `json:"user_id"`
	Username     string `json:"username"`
	FriendlyName string `json:"friendly_name,omitempty"`

	// Device information
	MachineID string `json:"machine_id,omitempty"`
	Platform  string `json:"platform,omitempty"`
	Player    string `json:"player,omitempty"`
	Device    string `json:"device,omitempty"`

	// Media information
	MediaType        string `json:"media_type"`
	Title            string `json:"title"`
	GrandparentTitle string `json:"grandparent_title,omitempty"` // Show name

	// Network information
	IPAddress    string `json:"ip_address"`
	LocationType string `json:"location_type"` // wan, lan

	// Geolocation (enriched from geolocations table)
	Latitude  float64 `json:"latitude,omitempty"`
	Longitude float64 `json:"longitude,omitempty"`
	City      string  `json:"city,omitempty"`
	Region    string  `json:"region,omitempty"`
	Country   string  `json:"country,omitempty"`
}

// AlertStore defines the interface for alert persistence.
type AlertStore interface {
	// SaveAlert persists a new alert.
	SaveAlert(ctx context.Context, alert *Alert) error

	// GetAlert retrieves an alert by ID.
	GetAlert(ctx context.Context, id int64) (*Alert, error)

	// ListAlerts retrieves alerts with optional filtering.
	ListAlerts(ctx context.Context, filter AlertFilter) ([]Alert, error)

	// AcknowledgeAlert marks an alert as acknowledged.
	AcknowledgeAlert(ctx context.Context, id int64, acknowledgedBy string) error

	// GetAlertCount returns the count of alerts matching the filter.
	GetAlertCount(ctx context.Context, filter AlertFilter) (int, error)
}

// AlertFilter defines filtering options for alert queries.
type AlertFilter struct {
	RuleTypes      []RuleType `json:"rule_types,omitempty"`
	Severities     []Severity `json:"severities,omitempty"`
	UserID         *int       `json:"user_id,omitempty"`
	ServerID       *string    `json:"server_id,omitempty"` // v2.1: Multi-server support - filter by server
	Acknowledged   *bool      `json:"acknowledged,omitempty"`
	StartDate      *time.Time `json:"start_date,omitempty"`
	EndDate        *time.Time `json:"end_date,omitempty"`
	Limit          int        `json:"limit,omitempty"`
	Offset         int        `json:"offset,omitempty"`
	OrderBy        string     `json:"order_by,omitempty"`        // created_at, severity
	OrderDirection string     `json:"order_direction,omitempty"` // asc, desc
}

// RuleStore defines the interface for rule configuration persistence.
type RuleStore interface {
	// GetRule retrieves a rule by type.
	GetRule(ctx context.Context, ruleType RuleType) (*Rule, error)

	// ListRules retrieves all rules.
	ListRules(ctx context.Context) ([]Rule, error)

	// SaveRule persists a rule configuration.
	SaveRule(ctx context.Context, rule *Rule) error

	// SetRuleEnabled enables or disables a rule.
	SetRuleEnabled(ctx context.Context, ruleType RuleType, enabled bool) error
}

// TrustStore defines the interface for trust score persistence.
type TrustStore interface {
	// GetTrustScore retrieves a user's trust score.
	GetTrustScore(ctx context.Context, userID int) (*TrustScore, error)

	// UpdateTrustScore updates a user's trust score.
	UpdateTrustScore(ctx context.Context, score *TrustScore) error

	// DecrementTrustScore decreases a user's trust score by the given amount.
	DecrementTrustScore(ctx context.Context, userID int, amount int) error

	// RecoverTrustScores increases all users' trust scores (daily job).
	RecoverTrustScores(ctx context.Context, amount int) error

	// ListLowTrustUsers returns users with trust scores below threshold.
	ListLowTrustUsers(ctx context.Context, threshold int) ([]TrustScore, error)
}

// EventHistory provides access to recent events for detection rules.
// v2.1: All methods now include serverID parameter for multi-server support.
// Pass empty string for serverID to query across all servers (legacy behavior).
type EventHistory interface {
	// GetLastEventForUser retrieves the most recent event for a user on a specific server.
	// Pass empty serverID to get last event across all servers.
	GetLastEventForUser(ctx context.Context, userID int, serverID string) (*DetectionEvent, error)

	// GetActiveStreamsForUser retrieves currently active streams for a user on a specific server.
	// Pass empty serverID to get active streams across all servers.
	GetActiveStreamsForUser(ctx context.Context, userID int, serverID string) ([]DetectionEvent, error)

	// GetRecentIPsForDevice retrieves recent IPs for a device within window on a specific server.
	// Pass empty serverID to get IPs across all servers.
	GetRecentIPsForDevice(ctx context.Context, machineID string, serverID string, window time.Duration) ([]string, error)

	// GetSimultaneousLocations retrieves concurrent sessions at different locations for a user on a specific server.
	// Pass empty serverID to get locations across all servers.
	GetSimultaneousLocations(ctx context.Context, userID int, serverID string, window time.Duration) ([]DetectionEvent, error)

	// GetGeolocation retrieves geolocation for an IP address.
	GetGeolocation(ctx context.Context, ipAddress string) (*Geolocation, error)
}

// Geolocation contains geographic information for an IP address.
type Geolocation struct {
	IPAddress string  `json:"ip_address"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	City      string  `json:"city,omitempty"`
	Region    string  `json:"region,omitempty"`
	Country   string  `json:"country"`
}

// Notifier sends alerts to external systems.
type Notifier interface {
	// Send delivers an alert to the notification channel.
	Send(ctx context.Context, alert *Alert) error

	// Name returns the notifier name (e.g., "discord", "webhook").
	Name() string

	// Enabled returns whether this notifier is enabled.
	Enabled() bool
}
