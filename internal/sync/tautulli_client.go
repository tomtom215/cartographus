// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

/*
tautulli_client.go - Core Tautulli API Client

This file provides the core TautulliClient struct and HTTP communication layer
for interacting with Tautulli's REST API.

Client Features:
  - HTTP client with configurable timeout
  - API key authentication
  - Circuit breaker protection (v1.35)
  - Automatic HTTP 429 rate limit handling with exponential backoff
  - JSON response parsing with generic type support
  - Context support for cancellation and timeouts

Resilience Mechanisms:
  - Circuit Breaker: Opens after 3 consecutive failures (60s open period)
  - Rate Limiting: Exponential backoff (1s, 2s, 4s, 8s, 16s) on HTTP 429
  - Retries: Max 5 attempts for rate-limited requests
  - Context: All methods accept context for cancellation

TautulliClientInterface:
The interface exposes 54 API methods organized by category:
  - Core: Ping, GetHistorySince, GetGeoIPLookup
  - Activity: GetActivity, GetHomeStats
  - Analytics: GetPlaysByDate, GetPlaysByDayOfWeek, etc.
  - Users: GetUser, GetUserIPs, GetUsersTable
  - Libraries: GetLibraries, GetLibraryMediaInfo
  - Metadata: GetMetadata, GetChildrenMetadata
  - Export: ExportMetadata, GetExportsTable

Related Files:
  - tautulli_history.go: Playback history methods
  - tautulli_analytics.go: Analytics and reporting methods
  - tautulli_users.go: User management methods
  - tautulli_library.go: Library content methods
  - tautulli_server.go: Server info and export methods
*/

//nolint:staticcheck // File documentation, not package doc
package sync

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"time"

	"github.com/goccy/go-json"

	"github.com/tomtom215/cartographus/internal/config"
	"github.com/tomtom215/cartographus/internal/models/tautulli"
)

// maxErrorBodySize limits the maximum amount of response body read for error reporting
// This prevents unbounded memory allocation when reading large error responses
const maxErrorBodySize = 64 * 1024 // 64KB

// readBodyForError reads the response body for error reporting (max 64KB)
// Returns the body content or a placeholder message if reading fails
// This satisfies errcheck linter while providing best-effort error diagnostics
// Uses io.LimitReader to prevent unbounded memory allocation
func readBodyForError(r io.Reader) []byte {
	// Limit reading to prevent memory issues with large responses
	limitedReader := io.LimitReader(r, maxErrorBodySize)
	body, err := io.ReadAll(limitedReader)
	if err != nil {
		return []byte("(failed to read response body)")
	}
	// If we hit the limit, indicate truncation
	if len(body) == maxErrorBodySize {
		return append(body, []byte("\n... (truncated)")...)
	}
	return body
}

// TautulliClientInterface defines the interface for all Tautulli API operations.
//
// This interface is implemented by TautulliClient for production use and by mock
// implementations for testing. It provides access to 54 Tautulli API endpoints
// organized into functional categories.
//
// Categories:
//   - Core: Ping, GetHistorySince, GetGeoIPLookup (connectivity and data sync)
//   - Activity: GetActivity, GetHomeStats (real-time session monitoring)
//   - Analytics: GetPlaysByDate, GetPlaysByDayOfWeek, etc. (chart data)
//   - Users: GetUser, GetUserIPs, GetUsersTable (user management)
//   - Libraries: GetLibraries, GetLibraryMediaInfo (content management)
//   - Metadata: GetMetadata, GetChildrenMetadata (media information)
//   - Export: ExportMetadata, GetExportsTable (data export)
//   - Server: GetServerInfo, GetTautulliInfo (server status)
//
// All methods follow a consistent pattern:
//   - Accept context.Context as first parameter for cancellation/timeout support
//   - Return typed response structs from internal/models/tautulli
//   - Return error on HTTP failures, API errors, or JSON parse failures
//   - Use appropriate timeouts from circuit breaker configuration
//
// Thread Safety: All methods are safe for concurrent use.
type TautulliClientInterface interface {
	Ping(ctx context.Context) error
	GetHistorySince(ctx context.Context, since time.Time, start, length int) (*tautulli.TautulliHistory, error)
	GetGeoIPLookup(ctx context.Context, ipAddress string) (*tautulli.TautulliGeoIP, error)
	GetHomeStats(ctx context.Context, timeRange int, statsType string, statsCount int) (*tautulli.TautulliHomeStats, error)
	GetPlaysByDate(ctx context.Context, timeRange int, yAxis string, userID int, grouping int) (*tautulli.TautulliPlaysByDate, error)
	GetPlaysByDayOfWeek(ctx context.Context, timeRange int, yAxis string, userID int, grouping int) (*tautulli.TautulliPlaysByDayOfWeek, error)
	GetPlaysByHourOfDay(ctx context.Context, timeRange int, yAxis string, userID int, grouping int) (*tautulli.TautulliPlaysByHourOfDay, error)
	GetPlaysByStreamType(ctx context.Context, timeRange int, yAxis string, userID int, grouping int) (*tautulli.TautulliPlaysByStreamType, error)
	GetConcurrentStreamsByStreamType(ctx context.Context, timeRange int, userID int) (*tautulli.TautulliConcurrentStreamsByStreamType, error)
	GetItemWatchTimeStats(ctx context.Context, ratingKey string, grouping int, queryDays string) (*tautulli.TautulliItemWatchTimeStats, error)
	GetActivity(ctx context.Context, sessionKey string) (*tautulli.TautulliActivity, error)
	GetMetadata(ctx context.Context, ratingKey string) (*tautulli.TautulliMetadata, error)
	GetUser(ctx context.Context, userID int) (*tautulli.TautulliUser, error)
	GetUsers(ctx context.Context) (*tautulli.TautulliUsers, error)
	GetLibraryUserStats(ctx context.Context, sectionID int, grouping int) (*tautulli.TautulliLibraryUserStats, error)
	GetRecentlyAdded(ctx context.Context, count int, start int, mediaType string, sectionID int) (*tautulli.TautulliRecentlyAdded, error)
	GetLibraries(ctx context.Context) (*tautulli.TautulliLibraries, error)
	GetLibrary(ctx context.Context, sectionID int) (*tautulli.TautulliLibrary, error)
	GetServerInfo(ctx context.Context) (*tautulli.TautulliServerInfo, error)
	GetSyncedItems(ctx context.Context, machineID string, userID int) (*tautulli.TautulliSyncedItems, error)
	TerminateSession(ctx context.Context, sessionID string, message string) (*tautulli.TautulliTerminateSession, error)

	// Priority 1: Analytics Dashboard Completion (8 endpoints)
	GetPlaysBySourceResolution(ctx context.Context, timeRange int, yAxis string, userID int, grouping int) (*tautulli.TautulliPlaysBySourceResolution, error)
	GetPlaysByStreamResolution(ctx context.Context, timeRange int, yAxis string, userID int, grouping int) (*tautulli.TautulliPlaysByStreamResolution, error)
	GetPlaysByTop10Platforms(ctx context.Context, timeRange int, yAxis string, userID int, grouping int) (*tautulli.TautulliPlaysByTop10Platforms, error)
	GetPlaysByTop10Users(ctx context.Context, timeRange int, yAxis string, userID int, grouping int) (*tautulli.TautulliPlaysByTop10Users, error)
	GetPlaysPerMonth(ctx context.Context, timeRange int, yAxis string, userID int, grouping int) (*tautulli.TautulliPlaysPerMonth, error)
	GetUserPlayerStats(ctx context.Context, userID int) (*tautulli.TautulliUserPlayerStats, error)
	GetUserWatchTimeStats(ctx context.Context, userID int, queryDays string) (*tautulli.TautulliUserWatchTimeStats, error)
	GetItemUserStats(ctx context.Context, ratingKey string, grouping int) (*tautulli.TautulliItemUserStats, error)

	// Priority 2: Library-Specific Analytics (4 endpoints)
	GetLibrariesTable(ctx context.Context, grouping int, orderColumn string, orderDir string, start int, length int, search string) (*tautulli.TautulliLibrariesTable, error)
	GetLibraryMediaInfo(ctx context.Context, sectionID int, orderColumn string, orderDir string, start int, length int) (*tautulli.TautulliLibraryMediaInfo, error)
	GetLibraryWatchTimeStats(ctx context.Context, sectionID int, grouping int, queryDays string) (*tautulli.TautulliLibraryWatchTimeStats, error)
	GetChildrenMetadata(ctx context.Context, ratingKey string, mediaType string) (*tautulli.TautulliChildrenMetadata, error)

	// Priority 1: User Geography & Management (3 endpoints)
	GetUserIPs(ctx context.Context, userID int) (*tautulli.TautulliUserIPs, error)
	GetUsersTable(ctx context.Context, grouping int, orderColumn string, orderDir string, start int, length int, search string) (*tautulli.TautulliUsersTable, error)
	GetUserLogins(ctx context.Context, userID int, orderColumn string, orderDir string, start int, length int, search string) (*tautulli.TautulliUserLogins, error)

	// Priority 2: Enhanced Metadata & Export (4 endpoints)
	GetStreamData(ctx context.Context, rowID int, sessionKey string) (*tautulli.TautulliStreamData, error)
	GetLibraryNames(ctx context.Context) (*tautulli.TautulliLibraryNames, error)
	ExportMetadata(ctx context.Context, sectionID int, exportType string, userID int, ratingKey string, fileFormat string) (*tautulli.TautulliExportMetadata, error)
	GetExportFields(ctx context.Context, mediaType string) (*tautulli.TautulliExportFields, error)

	// Priority 3: Advanced Analytics & Metadata (5 endpoints)
	GetStreamTypeByTop10Users(ctx context.Context, timeRange int, yAxis string, userID int, grouping int) (*tautulli.TautulliStreamTypeByTop10Users, error)
	GetStreamTypeByTop10Platforms(ctx context.Context, timeRange int, yAxis string, userID int, grouping int) (*tautulli.TautulliStreamTypeByTop10Platforms, error)
	Search(ctx context.Context, query string, limit int) (*tautulli.TautulliSearch, error)
	GetNewRatingKeys(ctx context.Context, ratingKey string) (*tautulli.TautulliNewRatingKeys, error)
	GetOldRatingKeys(ctx context.Context, ratingKey string) (*tautulli.TautulliOldRatingKeys, error)

	// Collections & Playlists (2 endpoints)
	GetCollectionsTable(ctx context.Context, sectionID int, orderColumn string, orderDir string, start int, length int, search string) (*tautulli.TautulliCollectionsTable, error)
	GetPlaylistsTable(ctx context.Context, sectionID int, orderColumn string, orderDir string, start int, length int, search string) (*tautulli.TautulliPlaylistsTable, error)

	// Server Information (3 endpoints)
	GetServerFriendlyName(ctx context.Context) (*tautulli.TautulliServerFriendlyName, error)
	GetServerID(ctx context.Context) (*tautulli.TautulliServerID, error)
	GetServerIdentity(ctx context.Context) (*tautulli.TautulliServerIdentity, error)

	// Data Export Management (3 endpoints)
	GetExportsTable(ctx context.Context, orderColumn string, orderDir string, start int, length int, search string) (*tautulli.TautulliExportsTable, error)
	DownloadExport(ctx context.Context, exportID int) (*tautulli.TautulliDownloadExport, error)
	DeleteExport(ctx context.Context, exportID int) (*tautulli.TautulliDeleteExport, error)

	// Server Management (5 endpoints)
	GetTautulliInfo(ctx context.Context) (*tautulli.TautulliTautulliInfo, error)
	GetServerPref(ctx context.Context, pref string) (*tautulli.TautulliServerPref, error)
	GetServerList(ctx context.Context) (*tautulli.TautulliServerList, error)
	GetServersInfo(ctx context.Context) (*tautulli.TautulliServersInfo, error)
	GetPMSUpdate(ctx context.Context) (*tautulli.TautulliPMSUpdate, error)
}

// TautulliClient handles communication with the Tautulli HTTP API.
//
// This client implements TautulliClientInterface and provides access to all
// Tautulli API endpoints used by Cartographus. It includes built-in rate
// limiting with exponential backoff for HTTP 429 responses.
//
// Features:
//   - 30-second request timeout
//   - Automatic retry on rate limiting (up to 5 retries)
//   - Exponential backoff (1s, 2s, 4s, 8s, 16s delays)
//   - JSON parsing with typed response structs
//   - Generic API call helper for consistent error handling
//
// Thread Safety: Safe for concurrent use. Each request creates its own HTTP request.
//
// Example:
//
//	client := sync.NewTautulliClient(cfg.Tautulli)
//	if err := client.Ping(); err != nil {
//	    log.Fatal("Tautulli not reachable:", err)
//	}
//	history, err := client.GetHistorySince(time.Now().AddDate(0, 0, -7), 0, 1000)
type TautulliClient struct {
	baseURL        string
	apiKey         string
	client         *http.Client
	maxRetries     int           // Maximum retries for rate limiting
	retryBaseDelay time.Duration // Base delay for exponential backoff
}

// NewTautulliClient creates a new Tautulli API client with the provided configuration.
//
// The client is configured with:
//   - 30-second HTTP timeout
//   - 5 maximum retries for rate limiting
//   - 1-second base delay for exponential backoff
//
// Parameters:
//   - cfg: Tautulli configuration containing URL and API key
//
// Returns a configured TautulliClient ready for API calls.
func NewTautulliClient(cfg *config.TautulliConfig) *TautulliClient {
	return &TautulliClient{
		baseURL: cfg.URL,
		apiKey:  cfg.APIKey,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		maxRetries:     5,               // Allow up to 5 retries for rate limiting
		retryBaseDelay: 1 * time.Second, // Start with 1 second, doubles each retry
	}
}

// doRequestWithRateLimit performs an HTTP request with automatic rate limit handling.
// Implements exponential backoff for HTTP 429 responses (1s, 2s, 4s, 8s, 16s).
// The context is used for cancellation during backoff waits.
func (c *TautulliClient) doRequestWithRateLimit(ctx context.Context, reqURL string) (*http.Response, error) {
	var lastErr error

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		// Check context before attempting request
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		// Create request with context
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, http.NoBody)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		resp, err := c.client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("HTTP request failed: %w", err)
		}

		// Success - return response
		if resp.StatusCode != http.StatusTooManyRequests {
			return resp, nil
		}

		// Rate limited (HTTP 429) - close body and retry with backoff
		_ = resp.Body.Close() // Explicitly ignore error - will retry anyway

		// Last attempt - return error
		if attempt == c.maxRetries {
			lastErr = fmt.Errorf("rate limit exceeded after %d retries (HTTP 429)", c.maxRetries)
			break
		}

		// Calculate exponential backoff delay: 1s, 2s, 4s, 8s, 16s
		delay := c.retryBaseDelay * time.Duration(1<<uint(attempt))

		// Check for Retry-After header (RFC 6585)
		if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
			// Try parsing as seconds (integer)
			if seconds, err := time.ParseDuration(retryAfter + "s"); err == nil {
				delay = seconds
			}
		}

		// Use cancellable wait instead of time.Sleep
		select {
		case <-time.After(delay):
			// Continue to next attempt
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	return nil, lastErr
}

// makeRequest is a generic helper that handles common Tautulli API request boilerplate.
// It builds the URL with API key and command, makes the request, checks HTTP status,
// decodes JSON response, and validates the Tautulli response wrapper.
//
// Parameters:
//   - ctx: Context for cancellation and timeout support
//   - cmd: Tautulli API command name (e.g., "get_server_info")
//   - params: Additional URL parameters (without apikey/cmd which are added automatically)
//   - result: Pointer to response struct that will be populated
//
// Returns error if request fails, HTTP status is not 200, JSON decode fails,
// or Tautulli response.result != "success".
//
// The result parameter must be a pointer to a struct that embeds tautulli.BaseResponse,
// which provides the common response wrapper with Result and Message fields.
func (c *TautulliClient) makeRequest(ctx context.Context, cmd string, params url.Values, result interface{}) error {
	// Add required parameters
	if params == nil {
		params = url.Values{}
	}
	params.Set("apikey", c.apiKey)
	params.Set("cmd", cmd)

	reqURL := fmt.Sprintf("%s/api/v2?%s", c.baseURL, params.Encode())

	resp, err := c.doRequestWithRateLimit(ctx, reqURL)
	if err != nil {
		return fmt.Errorf("failed to make %s request: %w", cmd, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body := readBodyForError(resp.Body)
		return fmt.Errorf("%s request failed with status %d: %s", cmd, resp.StatusCode, string(body))
	}

	// Decode JSON response
	if err := decodeJSONResponse(resp, result); err != nil {
		return fmt.Errorf("failed to decode %s response: %w", cmd, err)
	}

	// Validate Tautulli response wrapper
	return validateTautulliResponse(result, cmd)
}

// decodeJSONResponse decodes HTTP response body into the provided result struct
func decodeJSONResponse(resp *http.Response, result interface{}) error {
	decoder := json.NewDecoder(resp.Body)
	return decoder.Decode(result)
}

// validateTautulliResponse checks if the Tautulli API returned success.
// All Tautulli responses have a common wrapper with response.result field.
// This uses reflection to access the Response field since all Tautulli types follow the same pattern.
func validateTautulliResponse(result interface{}, cmd string) error {
	// Use reflection to access the Response field
	// All Tautulli response types have a Response field with Result and Message
	v := reflect.ValueOf(result)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return nil // Skip validation for non-struct types
	}

	responseField := v.FieldByName("Response")
	if !responseField.IsValid() {
		return nil // No Response field, skip validation
	}

	resultField := responseField.FieldByName("Result")
	if !resultField.IsValid() || resultField.Kind() != reflect.String {
		return nil // No Result field or not a string
	}

	if resultField.String() != "success" {
		msg := "unknown error"
		messageField := responseField.FieldByName("Message")
		if messageField.IsValid() && messageField.Kind() == reflect.Ptr && !messageField.IsNil() {
			if messageField.Elem().Kind() == reflect.String {
				msg = messageField.Elem().String()
			}
		}
		return fmt.Errorf("%s request failed: %s", cmd, msg)
	}

	return nil
}

// Ping verifies connectivity to Tautulli API.
// The context is used for cancellation and timeout support.
func (c *TautulliClient) Ping(ctx context.Context) error {
	params := url.Values{}
	params.Set("apikey", c.apiKey)
	params.Set("cmd", "arnold")

	reqURL := fmt.Sprintf("%s/api/v2?%s", c.baseURL, params.Encode())

	resp, err := c.doRequestWithRateLimit(ctx, reqURL)
	if err != nil {
		return fmt.Errorf("failed to ping Tautulli: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Tautulli ping failed with status: %d", resp.StatusCode)
	}

	return nil
}
