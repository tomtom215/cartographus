// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

/*
tautulli_history.go - Tautulli Playback History Methods

This file provides methods for retrieving playback history from Tautulli,
which is the primary data source for the sync manager.

NOTE: This file uses encoding/json instead of go-json (ADR-0021) because
go-json issue #340 causes "expected comma after object element" parsing
errors with large Tautulli API responses (500+ records).

History Methods:
  - GetHistory(): Paginated history retrieval (newest first)
  - GetHistorySince(): History since a specific timestamp
  - GetStreamData(): Detailed stream metadata for a session

Data Richness:
Tautulli history provides comprehensive playback data:
  - User identification (username, user_id, IP address)
  - Geolocation data via GeoIP API
  - Media metadata (title, year, rating, duration)
  - Quality metrics (resolution, codec, bitrate)
  - Platform information (player, device, product)
  - Transcode decision and stream statistics
  - Connection details (secure, relayed, local)

Pagination:
History endpoints support pagination via:
  - start: Starting record index
  - length: Number of records to retrieve
  - order_column/order_dir: Sorting configuration

The default configuration orders by "started" timestamp descending (newest first).
*/

//nolint:staticcheck // File documentation, not package doc
package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/tomtom215/cartographus/internal/logging"
	"github.com/tomtom215/cartographus/internal/models/tautulli"
)

// GetHistory retrieves playback history from Tautulli
func (c *TautulliClient) GetHistory(ctx context.Context, start, length int) (*tautulli.TautulliHistory, error) {
	params := url.Values{}
	params.Set("apikey", c.apiKey)
	params.Set("cmd", "get_history")
	params.Set("start", fmt.Sprintf("%d", start))
	params.Set("length", fmt.Sprintf("%d", length))
	params.Set("order_column", "started")
	params.Set("order_dir", "desc")
	// Disable session grouping to get individual playback records
	// Without this, Tautulli groups consecutive plays of the same content by the same user
	params.Set("grouping", "0")

	return c.doHistoryRequest(ctx, params)
}

// GetHistorySince retrieves playback history since a specific timestamp
func (c *TautulliClient) GetHistorySince(ctx context.Context, since time.Time, start, length int) (*tautulli.TautulliHistory, error) {
	params := url.Values{}
	params.Set("apikey", c.apiKey)
	params.Set("cmd", "get_history")
	params.Set("start", fmt.Sprintf("%d", start))
	params.Set("length", fmt.Sprintf("%d", length))
	params.Set("order_column", "started")
	params.Set("order_dir", "desc")
	// Tautulli API expects date in "YYYY-MM-DD" format, not Unix timestamp
	params.Set("after", since.Format("2006-01-02"))
	// Disable session grouping to get individual playback records
	// Without this, Tautulli groups consecutive plays of the same content by the same user
	params.Set("grouping", "0")

	return c.doHistoryRequest(ctx, params)
}

// GetGeoIPLookup retrieves geolocation information for an IP address
func (c *TautulliClient) GetGeoIPLookup(ctx context.Context, ipAddress string) (*tautulli.TautulliGeoIP, error) {
	params := url.Values{}
	params.Set("apikey", c.apiKey)
	params.Set("cmd", "get_geoip_lookup")
	params.Set("ip_address", ipAddress)

	reqURL := fmt.Sprintf("%s/api/v2?%s", c.baseURL, params.Encode())

	resp, err := c.doRequestWithRateLimit(ctx, reqURL)
	if err != nil {
		return nil, fmt.Errorf("failed to make geoip request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body := readBodyForError(resp.Body)
		return nil, fmt.Errorf("geoip request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var geoIP tautulli.TautulliGeoIP
	if err := json.NewDecoder(resp.Body).Decode(&geoIP); err != nil {
		return nil, fmt.Errorf("failed to decode geoip response: %w", err)
	}

	if geoIP.Response.Result != "success" {
		msg := "unknown error"
		if geoIP.Response.Message != nil {
			msg = *geoIP.Response.Message
		}
		return nil, fmt.Errorf("geoip lookup failed: %s", msg)
	}

	return &geoIP, nil
}

// GetActivity retrieves current streaming activity from Tautulli
func (c *TautulliClient) GetActivity(ctx context.Context, sessionKey string) (*tautulli.TautulliActivity, error) {
	params := url.Values{}
	params.Set("apikey", c.apiKey)
	params.Set("cmd", "get_activity")
	if sessionKey != "" {
		params.Set("session_key", sessionKey)
	}

	reqURL := fmt.Sprintf("%s/api/v2?%s", c.baseURL, params.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make activity request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body := readBodyForError(resp.Body)
		return nil, fmt.Errorf("activity request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var activity tautulli.TautulliActivity
	if err := json.NewDecoder(resp.Body).Decode(&activity); err != nil {
		return nil, fmt.Errorf("failed to decode activity response: %w", err)
	}

	if activity.Response.Result != "success" {
		msg := "unknown error"
		if activity.Response.Message != nil {
			msg = *activity.Response.Message
		}
		return nil, fmt.Errorf("activity request failed: %s", msg)
	}

	return &activity, nil
}

// doHistoryRequest performs a history API request with common error handling.
// NOTE: This file uses encoding/json instead of go-json due to go-json issue #340
// causing "expected comma after object element" errors with large API responses.
func (c *TautulliClient) doHistoryRequest(ctx context.Context, params url.Values) (*tautulli.TautulliHistory, error) {
	reqURL := fmt.Sprintf("%s/api/v2?%s", c.baseURL, params.Encode())

	resp, err := c.doRequestWithRateLimit(ctx, reqURL)
	if err != nil {
		return nil, fmt.Errorf("failed to make history request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body := readBodyForError(resp.Body)
		return nil, fmt.Errorf("history request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Read the full response body for JSON parsing
	// Note: Use io.ReadAll here instead of readBodyForError because history responses
	// can be large (500+ records = several MB). readBodyForError has a 64KB limit
	// designed only for error response diagnostics.
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read history response body: %w", err)
	}

	// DEBUG: Decode the response and capture detailed error if it fails
	var history tautulli.TautulliHistory
	if err := json.Unmarshal(bodyBytes, &history); err != nil {
		// If unmarshaling fails, log the first 2000 characters of the response
		maxLen := 2000
		if len(bodyBytes) < maxLen {
			maxLen = len(bodyBytes)
		}
		return nil, fmt.Errorf("failed to decode history response (showing first %d chars): %w\nJSON: %s", maxLen, err, string(bodyBytes[:maxLen]))
	}

	// Check if the API call was successful
	if history.Response.Result != "success" {
		msg := "unknown error"
		if history.Response.Message != nil {
			msg = *history.Response.Message
		}
		return nil, fmt.Errorf("history request failed: %s", msg)
	}

	// TRACING: Log session keys fetched from Tautulli API
	// This enables end-to-end tracing for data loss investigation
	recordCount := len(history.Response.Data.Data)
	if recordCount > 0 {
		// Log summary
		logging.Trace().Int("count", recordCount).Msg("TRACE API: Fetched records from Tautulli")

		// Log first 5 and last 5 session keys for traceability
		for i := range history.Response.Data.Data {
			record := &history.Response.Data.Data[i]
			sessionKey := ""
			if record.SessionKey != nil && *record.SessionKey != "" {
				sessionKey = *record.SessionKey
			} else {
				sessionKey = fmt.Sprintf("tautulli-%d", record.RowID)
			}
			if i < 5 || i >= recordCount-5 {
				logging.Trace().Int("index", i+1).Int("total", recordCount).Str("session", sessionKey).Str("user", record.User).Str("title", record.Title).Msg("TRACE API")
			} else if i == 5 {
				logging.Trace().Int("omitted", recordCount-10).Msg("TRACE API: ... (records omitted) ...")
			}
		}
	}

	return &history, nil
}
