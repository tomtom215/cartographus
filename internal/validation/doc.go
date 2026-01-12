// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package validation provides struct validation using go-playground/validator v10.
//
// This package wraps the go-playground/validator library to provide a thread-safe
// singleton validator instance with custom validators and user-friendly error
// messages. It integrates with the application's API error format for consistent
// error responses.
//
// # Overview
//
// The package provides:
//   - Thread-safe singleton validator (initialized once, cached struct info)
//   - Comprehensive error translation to human-readable messages
//   - APIError conversion matching the application's error format
//   - Built-in validator support (email, url, latitude, longitude, etc.)
//   - Future v11 compatibility with WithRequiredStructEnabled
//
// # Quick Start
//
//	type CreateUserRequest struct {
//	    Username string `validate:"required,min=3,max=50"`
//	    Email    string `validate:"required,email"`
//	    Age      int    `validate:"gte=13,lte=120"`
//	}
//
//	func handler(w http.ResponseWriter, r *http.Request) {
//	    var req CreateUserRequest
//	    if err := json.Decode(r.Body, &req); err != nil {
//	        // handle decode error
//	    }
//
//	    if verr := validation.ValidateStruct(&req); verr != nil {
//	        apiErr := verr.ToAPIError()
//	        respondError(w, http.StatusBadRequest, apiErr.Code, apiErr.Message, nil)
//	        return
//	    }
//
//	    // proceed with valid request
//	}
//
// # Common Validation Tags
//
// String validations:
//   - required: Field must not be empty
//   - min=n: Minimum length n characters
//   - max=n: Maximum length n characters
//   - email: Valid email format
//   - url: Valid URL format
//   - base64url: URL-safe base64 encoding
//
// Numeric validations:
//   - gte=n: Greater than or equal to n
//   - lte=n: Less than or equal to n
//   - gt=n: Greater than n
//   - lt=n: Less than n
//   - min=n: Minimum value n
//   - max=n: Maximum value n
//
// Enum validations:
//   - oneof=a b c: Must be one of the specified values
//
// Coordinate validations:
//   - latitude: Valid latitude (-90 to 90)
//   - longitude: Valid longitude (-180 to 180)
//
// # Error Types
//
// ValidationError represents a single field validation failure:
//
//	type ValidationError struct {
//	    Field()   string      // Struct field name
//	    Tag()     string      // Validation tag that failed
//	    Param()   string      // Tag parameter (e.g., "100" for max=100)
//	    Value()   interface{} // Actual value that failed
//	    Error()   string      // Human-readable message
//	}
//
// RequestValidationError aggregates multiple field errors:
//
//	type RequestValidationError struct {
//	    Errors() []ValidationError
//	    Error()  string           // Combined message
//	    ToAPIError() *APIError    // Convert to API error format
//	}
//
// # API Error Integration
//
// The ToAPIError method produces errors matching the application format:
//
//	// Single field error
//	{
//	    "code": "VALIDATION_ERROR",
//	    "message": "Email must be a valid email address",
//	    "details": {"field": "Email", "tag": "email", "value": "invalid"}
//	}
//
//	// Multiple field errors
//	{
//	    "code": "VALIDATION_ERROR",
//	    "message": "Username: must be at least 3 characters; Email: required",
//	    "details": {
//	        "fields": [
//	            {"field": "Username", "tag": "min", "message": "..."},
//	            {"field": "Email", "tag": "required", "message": "..."}
//	        ]
//	    }
//	}
//
// # Error Message Translation
//
// Human-readable messages are generated for common validation tags:
//
//	required   -> "Username is required"
//	email      -> "Email must be a valid email address"
//	min=3      -> "Username must be at least 3 characters"
//	max=100    -> "Description must be at most 100 characters"
//	gte=1      -> "Limit must be greater than or equal to 1"
//	lte=1000   -> "Limit must be less than or equal to 1000"
//	oneof=a b  -> "Status must be one of: a b"
//	latitude   -> "Lat must be a valid latitude (-90 to 90)"
//	longitude  -> "Lon must be a valid longitude (-180 to 180)"
//
// # Struct Tag Examples
//
// API request validation:
//
//	type PlaybacksRequest struct {
//	    Limit    int    `validate:"min=1,max=1000"`
//	    Offset   int    `validate:"min=0,max=1000000"`
//	    Cursor   string `validate:"omitempty,base64url"`
//	    Order    string `validate:"omitempty,oneof=asc desc"`
//	}
//
// Geographic bounds:
//
//	type BoundingBox struct {
//	    MinLat float64 `validate:"required,latitude"`
//	    MaxLat float64 `validate:"required,latitude,gtfield=MinLat"`
//	    MinLon float64 `validate:"required,longitude"`
//	    MaxLon float64 `validate:"required,longitude,gtfield=MinLon"`
//	}
//
// # Thread Safety
//
// The singleton validator is initialized once and safe for concurrent use:
//
//	validate := validation.GetValidator()  // Thread-safe
//	err := validation.ValidateStruct(&req) // Thread-safe
//
// # Performance
//
// The validator caches struct reflection information:
//   - First validation of a struct type: ~1ms (reflection + caching)
//   - Subsequent validations: ~10us (cached)
//   - Memory: ~500 bytes per cached struct type
//
// # See Also
//
//   - internal/api: Request handlers using validation
//   - github.com/go-playground/validator/v10: Underlying library
//   - docs/adr/0013-request-validation.md: ADR for validator choice
package validation
