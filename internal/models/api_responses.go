// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package models

import (
	"time"
)

// APIResponse represents a standardized API response wrapper used by all HTTP endpoints.
// It provides consistent structure for both successful and error responses, with metadata
// for observability and caching information.
//
// Status field values:
//   - "success": Request completed successfully, see Data field
//   - "error": Request failed, see Error field for details
//
// Fields:
//   - Status: Response status ("success" or "error")
//   - Data: Response payload (any JSON-serializable type)
//   - Metadata: Query execution metadata (timing, caching, timestamp)
//   - Error: Error details (populated only when Status is "error")
//
// Example successful response:
//
//	{
//	  "status": "success",
//	  "data": {"total": 100, "results": [...]},
//	  "metadata": {
//	    "timestamp": "2025-11-28T12:00:00Z",
//	    "query_time_ms": 45,
//	    "cached": false
//	  }
//	}
//
// Example error response:
//
//	{
//	  "status": "error",
//	  "error": {
//	    "code": "VALIDATION_ERROR",
//	    "message": "Invalid date range",
//	    "details": {"field": "start_date"}
//	  },
//	  "metadata": {"timestamp": "2025-11-28T12:00:00Z"}
//	}
type APIResponse struct {
	Status   string      `json:"status"`
	Data     interface{} `json:"data"`
	Metadata Metadata    `json:"metadata"`
	Error    *APIError   `json:"error,omitempty"`
}

// Metadata contains response metadata for observability and performance tracking.
// All API responses include this metadata for monitoring query performance and
// cache effectiveness.
//
// Fields:
//   - Timestamp: Server time when response was generated (RFC3339 format)
//   - QueryTimeMS: Database query execution time in milliseconds (0 if cached)
//   - Cached: Whether response was served from cache (omitted if false)
//
// Query time tracking:
//   - Cached responses: QueryTimeMS is 0, Cached is true
//   - Fresh queries: QueryTimeMS shows actual DB execution time
//   - Sub-100ms p95 target: Most queries complete in <50ms with R-tree indexes
//
// Example cache hit:
//
//	{
//	  "timestamp": "2025-11-28T12:00:00Z",
//	  "query_time_ms": 0,
//	  "cached": true
//	}
//
// Example cache miss:
//
//	{
//	  "timestamp": "2025-11-28T12:00:00Z",
//	  "query_time_ms": 23
//	}
type Metadata struct {
	Timestamp   time.Time `json:"timestamp"`
	QueryTimeMS int64     `json:"query_time_ms,omitempty"`
	Cached      bool      `json:"cached,omitempty"`
}

// APIError represents an error response with structured error details.
// Provides consistent error format across all API endpoints for better client handling.
//
// Fields:
//   - Code: Machine-readable error code (e.g., "VALIDATION_ERROR", "DATABASE_ERROR")
//   - Message: Human-readable error message
//   - Details: Additional context (field names, constraints, etc.)
//
// Common error codes:
//   - VALIDATION_ERROR: Invalid input parameters
//   - DATABASE_ERROR: Query execution failure
//   - AUTHENTICATION_ERROR: Invalid/missing credentials
//   - AUTHORIZATION_ERROR: Insufficient permissions
//   - NOT_FOUND: Resource doesn't exist
//   - RATE_LIMIT_EXCEEDED: Too many requests
//
// Example:
//
//	{
//	  "code": "VALIDATION_ERROR",
//	  "message": "Invalid limit parameter (must be 1 to 100)",
//	  "details": {
//	    "field": "limit",
//	    "value": 500,
//	    "constraint": "max_100"
//	  }
//	}
type APIError struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// PaginationInfo contains cursor-based pagination metadata for efficient traversal.
// Supports infinite scrolling and large result sets without offset-based inefficiency.
//
// Fields:
//   - Limit: Maximum results per page (from request)
//   - HasMore: Whether more results exist beyond current page
//   - NextCursor: Opaque cursor for next page (null if no more results)
//   - PrevCursor: Opaque cursor for previous page (null if on first page)
//   - TotalCount: Total results matching filter (optional, expensive for large datasets)
//
// Cursor format: Base64-encoded JSON with timestamp + ID for stable sorting
//
// Example first page:
//
//	{
//	  "limit": 100,
//	  "has_more": true,
//	  "next_cursor": "eyJzdGFydGVkX2F0IjoiMjAyNS0wMS0wMVQxMjowMDowMFoiLCJpZCI6ImFiYzEyMyJ9",
//	  "prev_cursor": null
//	}
//
// Example middle page:
//
//	{
//	  "limit": 100,
//	  "has_more": true,
//	  "next_cursor": "...",
//	  "prev_cursor": "..."
//	}
type PaginationInfo struct {
	Limit      int     `json:"limit"`
	HasMore    bool    `json:"has_more"`
	NextCursor *string `json:"next_cursor,omitempty"`
	PrevCursor *string `json:"prev_cursor,omitempty"`
	TotalCount *int    `json:"total_count,omitempty"` // Optional, expensive for large datasets
}

// PlaybackCursor represents the cursor for playback pagination (v1.40+).
// Encodes the position in the result set using timestamp + ID for stable sorting.
// Encoded as base64 JSON for opaque client usage (prevents manipulation).
//
// Fields:
//   - StartedAt: Timestamp of the last playback event on previous page
//   - ID: UUID of the last playback event on previous page (tie-breaker)
//
// Encoding example:
//
//	cursor := PlaybackCursor{
//	    StartedAt: time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
//	    ID: "abc123",
//	}
//	json, _ := json.Marshal(cursor)
//	encoded := base64.StdEncoding.EncodeToString(json)
//	// encoded: "eyJzdGFydGVkX2F0IjoiMjAyNS0wMS0wMVQxMjowMDowMFoiLCJpZCI6ImFiYzEyMyJ9"
//
// Benefits:
//   - Stable: Results don't shift when new data is inserted
//   - Efficient: Uses index-optimized range scan (no OFFSET)
//   - Opaque: Clients can't manipulate cursor to skip data
type PlaybackCursor struct {
	StartedAt time.Time `json:"started_at"`
	ID        string    `json:"id"`
}

// PlaybacksResponse wraps playback events with cursor-based pagination info (v1.40+).
// Supports efficient infinite scrolling through playback history.
//
// Fields:
//   - Events: Array of playback events for current page
//   - Pagination: Cursor-based pagination metadata
//
// Example response:
//
//	{
//	  "events": [
//	    {"id": "abc123", "title": "Inception", ...},
//	    {"id": "def456", "title": "The Matrix", ...}
//	  ],
//	  "pagination": {
//	    "limit": 100,
//	    "has_more": true,
//	    "next_cursor": "..."
//	  }
//	}
type PlaybacksResponse struct {
	Events     []PlaybackEvent `json:"events"`
	Pagination PaginationInfo  `json:"pagination"`
}

// LoginRequest represents a login request for JWT authentication.
// Supports both standard and "remember me" login flows.
//
// Fields:
//   - Username: User's login name
//   - Password: User's password (plaintext, transmitted over HTTPS)
//   - RememberMe: If true, extends token expiration to 30 days (default 24h)
//
// Example:
//
//	{
//	  "username": "admin",
//	  "password": "securepassword123",
//	  "remember_me": true
//	}
//
// Security:
//   - Password is transmitted in plaintext (HTTPS required)
//   - Password is hashed with bcrypt (cost 12) before storage
//   - JWT tokens are HTTP-only cookies (XSS protection)
//   - Rate limited to 5 attempts per minute per IP
type LoginRequest struct {
	Username   string `json:"username"`
	Password   string `json:"password"`
	RememberMe bool   `json:"remember_me"`
}

// LoginResponse represents a successful login response with JWT token.
// Returns signed JWT token for subsequent authenticated requests.
//
// Fields:
//   - Token: Signed JWT token (RS256 algorithm)
//   - ExpiresAt: Token expiration timestamp (24h standard, 30d remember me)
//   - Username: Authenticated username
//   - Role: User's role (viewer, editor, admin)
//   - UserID: Unique user identifier
//
// Example:
//
//	{
//	  "token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...",
//	  "expires_at": "2025-11-29T12:00:00Z",
//	  "username": "admin",
//	  "role": "admin",
//	  "user_id": "admin-001"
//	}
//
// Token usage:
//   - Set as HTTP-only cookie by server (XSS protection)
//   - OR sent as Authorization: Bearer <token> header
//   - Validated on every protected endpoint
//   - Auto-refresh before expiration (client responsibility)
type LoginResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	Username  string    `json:"username"`
	Role      string    `json:"role"`
	UserID    string    `json:"user_id"`
}
