// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

/*
plex_request.go - Plex HTTP Request Helpers

This file provides helper functions for building and executing HTTP requests
to the Plex Media Server API with consistent configuration.

Request Configuration:
  - Authentication: X-Plex-Token header on all requests
  - JSON Accept: Optional Accept: application/json header
  - Status Validation: Check for expected HTTP status codes
  - Rate Limiting: Automatic retry with exponential backoff

Helper Functions:
  - doRequest(): Execute request with full configuration options
  - doJSONRequest(): Convenience wrapper for JSON API requests
  - doJSONRequestWithQuery(): JSON request with URL query parameters

Request Config Options:
  - method: HTTP method (GET, POST, DELETE)
  - path: API endpoint path (e.g., "/status/sessions")
  - query: URL query parameters
  - acceptJSON: Add Accept: application/json header
  - expectOK: Require HTTP 200 status
  - expectNoErr: Accept both 200 OK and 204 No Content

All requests automatically use doRequestWithRateLimit() for HTTP 429 handling.
*/

//nolint:staticcheck // File documentation, not package doc
package sync

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/goccy/go-json"
)

// requestConfig holds configuration for building HTTP requests
type requestConfig struct {
	method      string
	path        string
	query       url.Values
	acceptJSON  bool
	expectOK    bool // if true, check for 200 OK status
	expectNoErr bool // if true, also accept 204 No Content
}

// doRequest is a helper that executes a standard Plex API request and decodes the response
func (c *PlexClient) doRequest(ctx context.Context, cfg requestConfig, result interface{}) error {
	reqURL := fmt.Sprintf("%s%s", c.baseURL, cfg.path)

	req, err := http.NewRequestWithContext(ctx, cfg.method, reqURL, http.NoBody)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	// Add authentication
	req.Header.Set("X-Plex-Token", c.token)

	// Add Accept header for JSON responses
	if cfg.acceptJSON {
		req.Header.Set("Accept", "application/json")
	}

	// Add query parameters
	if len(cfg.query) > 0 {
		req.URL.RawQuery = cfg.query.Encode()
	}

	// Execute request with rate limiting
	resp, err := c.doRequestWithRateLimit(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Check status code
	if cfg.expectNoErr {
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
			return fmt.Errorf("unexpected status: %d %s", resp.StatusCode, resp.Status)
		}
	} else if cfg.expectOK && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %d %s", resp.StatusCode, resp.Status)
	}

	// Decode response if result pointer provided
	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}

	return nil
}

// doJSONRequest is a convenience wrapper for JSON API requests
func (c *PlexClient) doJSONRequest(ctx context.Context, path string, result interface{}) error {
	return c.doRequest(ctx, requestConfig{
		method:     http.MethodGet,
		path:       path,
		acceptJSON: true,
		expectOK:   true,
	}, result)
}

// doJSONRequestWithQuery is a convenience wrapper for JSON API requests with query parameters
func (c *PlexClient) doJSONRequestWithQuery(ctx context.Context, path string, query url.Values, result interface{}) error {
	return c.doRequest(ctx, requestConfig{
		method:     http.MethodGet,
		path:       path,
		query:      query,
		acceptJSON: true,
		expectOK:   true,
	}, result)
}
