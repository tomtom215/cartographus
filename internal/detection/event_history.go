// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package detection

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// DuckDBEventHistory implements EventHistory using the existing DuckDB tables.
type DuckDBEventHistory struct {
	db *sql.DB
}

// NewDuckDBEventHistory creates a new event history backed by DuckDB.
func NewDuckDBEventHistory(db *sql.DB) *DuckDBEventHistory {
	return &DuckDBEventHistory{db: db}
}

// GetLastEventForUser retrieves the most recent event for a user on a specific server.
// v2.1: Added serverID parameter for multi-server support. Pass empty string for all servers.
func (h *DuckDBEventHistory) GetLastEventForUser(ctx context.Context, userID int, serverID string) (*DetectionEvent, error) {
	// Build query with optional server_id filter
	serverFilter := ""
	args := []interface{}{userID}
	if serverID != "" {
		serverFilter = " AND p.server_id = ?"
		args = append(args, serverID)
	}

	query := fmt.Sprintf(`
		SELECT
			p.session_key,
			p.started_at,
			p.user_id,
			p.username,
			COALESCE(p.friendly_name, '') as friendly_name,
			COALESCE(p.server_id, '') as server_id,
			COALESCE(p.machine_id, '') as machine_id,
			COALESCE(p.platform, '') as platform,
			COALESCE(p.player, '') as player,
			COALESCE(p.device, '') as device,
			p.media_type,
			p.title,
			COALESCE(p.grandparent_title, '') as grandparent_title,
			p.ip_address,
			COALESCE(p.location_type, '') as location_type,
			COALESCE(g.latitude, 0) as latitude,
			COALESCE(g.longitude, 0) as longitude,
			COALESCE(g.city, '') as city,
			COALESCE(g.region, '') as region,
			COALESCE(g.country, '') as country
		FROM playback_events p
		LEFT JOIN geolocations g ON p.ip_address = g.ip_address
		WHERE p.user_id = ?%s
		ORDER BY p.started_at DESC
		LIMIT 1`, serverFilter)

	event := &DetectionEvent{}
	err := h.db.QueryRowContext(ctx, query, args...).Scan(
		&event.SessionKey,
		&event.Timestamp,
		&event.UserID,
		&event.Username,
		&event.FriendlyName,
		&event.ServerID,
		&event.MachineID,
		&event.Platform,
		&event.Player,
		&event.Device,
		&event.MediaType,
		&event.Title,
		&event.GrandparentTitle,
		&event.IPAddress,
		&event.LocationType,
		&event.Latitude,
		&event.Longitude,
		&event.City,
		&event.Region,
		&event.Country,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get last event: %w", err)
	}

	return event, nil
}

// GetActiveStreamsForUser retrieves currently active streams for a user on a specific server.
// Active streams are those that started within the last 4 hours and haven't stopped.
// v2.1: Added serverID parameter for multi-server support. Pass empty string for all servers.
func (h *DuckDBEventHistory) GetActiveStreamsForUser(ctx context.Context, userID int, serverID string) ([]DetectionEvent, error) {
	// Build query with optional server_id filter
	serverFilter := ""
	args := []interface{}{userID}
	if serverID != "" {
		serverFilter = " AND p.server_id = ?"
		args = append(args, serverID)
	}

	query := fmt.Sprintf(`
		SELECT
			p.session_key,
			p.started_at,
			p.user_id,
			p.username,
			COALESCE(p.friendly_name, '') as friendly_name,
			COALESCE(p.server_id, '') as server_id,
			COALESCE(p.machine_id, '') as machine_id,
			COALESCE(p.platform, '') as platform,
			COALESCE(p.player, '') as player,
			COALESCE(p.device, '') as device,
			p.media_type,
			p.title,
			COALESCE(p.grandparent_title, '') as grandparent_title,
			p.ip_address,
			COALESCE(p.location_type, '') as location_type,
			COALESCE(g.latitude, 0) as latitude,
			COALESCE(g.longitude, 0) as longitude,
			COALESCE(g.city, '') as city,
			COALESCE(g.region, '') as region,
			COALESCE(g.country, '') as country
		FROM playback_events p
		LEFT JOIN geolocations g ON p.ip_address = g.ip_address
		WHERE p.user_id = ?
		  AND p.stopped_at IS NULL
		  AND p.started_at >= CURRENT_TIMESTAMP - INTERVAL '4 hours'%s
		ORDER BY p.started_at DESC`, serverFilter)

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query active streams: %w", err)
	}
	defer rows.Close()

	var events []DetectionEvent
	for rows.Next() {
		event := DetectionEvent{}
		err := rows.Scan(
			&event.SessionKey,
			&event.Timestamp,
			&event.UserID,
			&event.Username,
			&event.FriendlyName,
			&event.ServerID,
			&event.MachineID,
			&event.Platform,
			&event.Player,
			&event.Device,
			&event.MediaType,
			&event.Title,
			&event.GrandparentTitle,
			&event.IPAddress,
			&event.LocationType,
			&event.Latitude,
			&event.Longitude,
			&event.City,
			&event.Region,
			&event.Country,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan active stream: %w", err)
		}
		events = append(events, event)
	}

	return events, rows.Err()
}

// GetRecentIPsForDevice retrieves recent IPs for a device within window on a specific server.
// v2.1: Added serverID parameter for multi-server support. Pass empty string for all servers.
func (h *DuckDBEventHistory) GetRecentIPsForDevice(ctx context.Context, machineID string, serverID string, window time.Duration) ([]string, error) {
	// Build query with optional server_id filter
	windowStart := time.Now().Add(-window)
	serverFilter := ""
	args := []interface{}{machineID, windowStart}
	if serverID != "" {
		serverFilter = " AND server_id = ?"
		args = append(args, serverID)
	}

	query := fmt.Sprintf(`
		SELECT DISTINCT ip_address
		FROM playback_events
		WHERE machine_id = ?
		  AND started_at >= ?%s
		ORDER BY started_at DESC`, serverFilter)

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent IPs: %w", err)
	}
	defer rows.Close()

	var ips []string
	for rows.Next() {
		var ip string
		if err := rows.Scan(&ip); err != nil {
			return nil, fmt.Errorf("failed to scan IP: %w", err)
		}
		ips = append(ips, ip)
	}

	return ips, rows.Err()
}

// GetSimultaneousLocations retrieves concurrent sessions at different locations on a specific server.
// v2.1: Added serverID parameter for multi-server support. Pass empty string for all servers.
func (h *DuckDBEventHistory) GetSimultaneousLocations(ctx context.Context, userID int, serverID string, window time.Duration) ([]DetectionEvent, error) {
	// Build query with optional server_id filter
	windowStart := time.Now().Add(-window)
	serverFilter := ""
	args := []interface{}{userID, windowStart}
	if serverID != "" {
		serverFilter = " AND p.server_id = ?"
		args = append(args, serverID)
	}

	query := fmt.Sprintf(`
		SELECT
			p.session_key,
			p.started_at,
			p.user_id,
			p.username,
			COALESCE(p.friendly_name, '') as friendly_name,
			COALESCE(p.server_id, '') as server_id,
			COALESCE(p.machine_id, '') as machine_id,
			COALESCE(p.platform, '') as platform,
			COALESCE(p.player, '') as player,
			COALESCE(p.device, '') as device,
			p.media_type,
			p.title,
			COALESCE(p.grandparent_title, '') as grandparent_title,
			p.ip_address,
			COALESCE(p.location_type, '') as location_type,
			COALESCE(g.latitude, 0) as latitude,
			COALESCE(g.longitude, 0) as longitude,
			COALESCE(g.city, '') as city,
			COALESCE(g.region, '') as region,
			COALESCE(g.country, '') as country
		FROM playback_events p
		LEFT JOIN geolocations g ON p.ip_address = g.ip_address
		WHERE p.user_id = ?
		  AND p.stopped_at IS NULL
		  AND p.started_at >= ?
		  AND g.latitude IS NOT NULL
		  AND g.longitude IS NOT NULL%s
		ORDER BY p.started_at DESC`, serverFilter)

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query simultaneous locations: %w", err)
	}
	defer rows.Close()

	var events []DetectionEvent
	for rows.Next() {
		event := DetectionEvent{}
		err := rows.Scan(
			&event.SessionKey,
			&event.Timestamp,
			&event.UserID,
			&event.Username,
			&event.FriendlyName,
			&event.ServerID,
			&event.MachineID,
			&event.Platform,
			&event.Player,
			&event.Device,
			&event.MediaType,
			&event.Title,
			&event.GrandparentTitle,
			&event.IPAddress,
			&event.LocationType,
			&event.Latitude,
			&event.Longitude,
			&event.City,
			&event.Region,
			&event.Country,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan location: %w", err)
		}
		events = append(events, event)
	}

	return events, rows.Err()
}

// GetGeolocation retrieves geolocation for an IP address.
func (h *DuckDBEventHistory) GetGeolocation(ctx context.Context, ipAddress string) (*Geolocation, error) {
	query := `
		SELECT ip_address, latitude, longitude,
			COALESCE(city, '') as city,
			COALESCE(region, '') as region,
			country
		FROM geolocations
		WHERE ip_address = ?`

	geo := &Geolocation{}
	err := h.db.QueryRowContext(ctx, query, ipAddress).Scan(
		&geo.IPAddress,
		&geo.Latitude,
		&geo.Longitude,
		&geo.City,
		&geo.Region,
		&geo.Country,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get geolocation: %w", err)
	}

	return geo, nil
}
