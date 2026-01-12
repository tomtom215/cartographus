// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package sync

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/goccy/go-json"
)

// apiRequest holds parameters for a Tautulli API request
type apiRequest struct {
	cmd    string
	params map[string]string
}

// newAPIRequest creates a new API request with the given command
func newAPIRequest(cmd string) *apiRequest {
	return &apiRequest{
		cmd:    cmd,
		params: make(map[string]string),
	}
}

// addParam adds a parameter to the request
func (r *apiRequest) addParam(key, value string) *apiRequest {
	if value != "" {
		r.params[key] = value
	}
	return r
}

// addIntParam adds an integer parameter to the request (only if > 0)
func (r *apiRequest) addIntParam(key string, value int) *apiRequest {
	if value > 0 {
		r.params[key] = fmt.Sprintf("%d", value)
	}
	return r
}

// addIntParamZero adds an integer parameter to the request (even if 0)
func (r *apiRequest) addIntParamZero(key string, value int) *apiRequest {
	if value >= 0 {
		r.params[key] = fmt.Sprintf("%d", value)
	}
	return r
}

// buildURL constructs the full URL with all parameters
func (r *apiRequest) buildURL(baseURL, apiKey string) string {
	params := url.Values{}
	params.Set("apikey", apiKey)
	params.Set("cmd", r.cmd)

	for key, value := range r.params {
		params.Set(key, value)
	}

	return fmt.Sprintf("%s/api/v2?%s", baseURL, params.Encode())
}

// executeRequest executes an HTTP GET request and returns the response body
func executeRequest(ctx context.Context, client *http.Client, reqURL string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		body := readBodyForError(resp.Body)
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return resp.Body, nil
}

// decodeResponse decodes JSON response and checks for success
func decodeResponse[T any](body io.ReadCloser, result *T, getResult func(*T) string, getMessage func(*T) *string) error {
	defer body.Close()

	if err := json.NewDecoder(body).Decode(result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if getResult(result) != "success" {
		msg := "unknown error"
		if getMessage(result) != nil {
			msg = *getMessage(result)
		}
		return fmt.Errorf("request failed: %s", msg)
	}

	return nil
}

// executeAPIRequest is a generic helper that executes a Tautulli API request
// It handles URL building, HTTP request, JSON decoding, and error handling
func executeAPIRequest[T any](
	ctx context.Context,
	c *TautulliClient,
	req *apiRequest,
	getResult func(*T) string,
	getMessage func(*T) *string,
) (*T, error) {
	// Build URL
	reqURL := req.buildURL(c.baseURL, c.apiKey)

	// Execute HTTP request
	body, err := executeRequest(ctx, c.client, reqURL)
	if err != nil {
		return nil, err
	}

	// Decode and validate response
	var result T
	if err := decodeResponse(body, &result, getResult, getMessage); err != nil {
		return nil, err
	}

	return &result, nil
}

// Common parameter builders for reusable parameter patterns

// addTimeRangeParams adds time_range, y_axis, user_id, and grouping parameters
func addTimeRangeParams(req *apiRequest, timeRange int, yAxis string, userID int, grouping int) {
	req.addIntParam("time_range", timeRange)
	req.addParam("y_axis", yAxis)
	req.addIntParam("user_id", userID)
	req.addIntParamZero("grouping", grouping)
}
