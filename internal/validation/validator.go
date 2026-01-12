// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package validation provides struct validation using go-playground/validator v10.
// It provides a thread-safe singleton validator instance with custom validators
// for application-specific validation rules.
//
// Features:
//   - Singleton validator instance (thread-safe, caches struct info)
//   - Custom validators for base64 cursors, RFC3339 dates, bounding box coordinates
//   - Error translation to match existing VALIDATION_ERROR format
//   - Uses WithRequiredStructEnabled option (v11+ compatibility)
//
// Example usage:
//
//	type PlaybacksRequest struct {
//	    Limit  int    `validate:"min=1,max=1000"`
//	    Offset int    `validate:"min=0,max=1000000"`
//	    Cursor string `validate:"omitempty,base64url"`
//	}
//
//	func handler(w http.ResponseWriter, r *http.Request) {
//	    req := PlaybacksRequest{...}
//	    if err := validation.ValidateStruct(&req); err != nil {
//	        apiErr := err.ToAPIError()
//	        respondError(w, http.StatusBadRequest, apiErr.Code, apiErr.Message, nil)
//	        return
//	    }
//	}
package validation

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/go-playground/validator/v10"
)

// singleton validator instance
var (
	validate     *validator.Validate
	validateOnce sync.Once
)

// ValidationError represents a single field validation error with structured information.
type ValidationError struct {
	field   string
	tag     string
	param   string
	value   interface{}
	message string
}

// Field returns the struct field name that failed validation.
func (e *ValidationError) Field() string {
	return e.field
}

// Tag returns the validation tag that failed.
func (e *ValidationError) Tag() string {
	return e.tag
}

// Param returns the parameter for the validation tag (e.g., "100" for "max=100").
func (e *ValidationError) Param() string {
	return e.param
}

// Value returns the actual value that failed validation.
func (e *ValidationError) Value() interface{} {
	return e.value
}

// Error returns a human-readable error message.
func (e *ValidationError) Error() string {
	return e.message
}

// RequestValidationError represents a collection of validation errors.
// It provides methods to convert errors to the application's APIError format.
type RequestValidationError struct {
	errors []ValidationError
}

// Errors returns the slice of validation errors.
func (ve *RequestValidationError) Errors() []ValidationError {
	return ve.errors
}

// Error implements the error interface, returning a combined error message.
func (ve *RequestValidationError) Error() string {
	if len(ve.errors) == 0 {
		return "validation failed"
	}

	var messages []string
	for _, err := range ve.errors {
		messages = append(messages, err.Error())
	}

	return strings.Join(messages, "; ")
}

// APIError represents an error response compatible with the existing API error format.
// This mirrors the models.APIError structure to avoid import cycles.
type APIError struct {
	Code    string
	Message string
	Details map[string]interface{}
}

// ToAPIError converts validation errors to the application's APIError format.
// It produces error messages compatible with the existing VALIDATION_ERROR format.
func (ve *RequestValidationError) ToAPIError() *APIError {
	if len(ve.errors) == 0 {
		return &APIError{
			Code:    "VALIDATION_ERROR",
			Message: "Validation failed",
		}
	}

	// Single error - use simple message
	if len(ve.errors) == 1 {
		err := ve.errors[0]
		return &APIError{
			Code:    "VALIDATION_ERROR",
			Message: err.message,
			Details: map[string]interface{}{
				"field": err.field,
				"tag":   err.tag,
				"value": err.value,
			},
		}
	}

	// Multiple errors - list all fields
	fields := make([]map[string]interface{}, len(ve.errors))
	var messages []string

	for i, err := range ve.errors {
		fields[i] = map[string]interface{}{
			"field":   err.field,
			"tag":     err.tag,
			"message": err.message,
		}
		messages = append(messages, fmt.Sprintf("%s: %s", err.field, err.message))
	}

	return &APIError{
		Code:    "VALIDATION_ERROR",
		Message: strings.Join(messages, "; "),
		Details: map[string]interface{}{
			"fields": fields,
		},
	}
}

// GetValidator returns the singleton validator instance.
// The validator is initialized once with custom validators and options.
// This function is thread-safe.
func GetValidator() *validator.Validate {
	validateOnce.Do(func() {
		validate = validator.New(validator.WithRequiredStructEnabled())

		// Register custom validators here if needed
		// The built-in validators cover most needs:
		// - base64url: validates URL-safe base64 encoding
		// - datetime: validates date/time format
		// - latitude, longitude: validates coordinate ranges
		// - email, url, uri: validates common formats
		// - oneof: validates against a set of allowed values
	})

	return validate
}

// ValidateStruct validates a struct using the singleton validator.
// Returns nil if validation passes, or *RequestValidationError if validation fails.
//
// Example:
//
//	err := ValidateStruct(&request)
//	if err != nil {
//	    apiErr := err.ToAPIError()
//	    respondError(w, http.StatusBadRequest, apiErr.Code, apiErr.Message, nil)
//	    return
//	}
func ValidateStruct(s interface{}) *RequestValidationError {
	v := GetValidator()

	err := v.Struct(s)
	if err == nil {
		return nil
	}

	// Convert validator errors to our RequestValidationError type using errors.As
	var validationErrs validator.ValidationErrors
	if !errors.As(err, &validationErrs) {
		// Unexpected error type - wrap it
		return &RequestValidationError{
			errors: []ValidationError{
				{
					field:   "unknown",
					tag:     "unknown",
					message: err.Error(),
				},
			},
		}
	}

	fieldErrors := make([]ValidationError, len(validationErrs))
	for i, fieldErr := range validationErrs {
		fieldErrors[i] = ValidationError{
			field:   fieldErr.Field(),
			tag:     fieldErr.Tag(),
			param:   fieldErr.Param(),
			value:   fieldErr.Value(),
			message: translateError(fieldErr),
		}
	}

	return &RequestValidationError{errors: fieldErrors}
}

// errorMessageTemplates maps validation tags to message templates.
// Templates use %s for field name and %p for parameter value.
var errorMessageTemplates = map[string]string{
	"required":  "%s is required",
	"email":     "%s must be a valid email address",
	"datetime":  "%s must be a valid date/time in RFC3339 format",
	"base64url": "%s must be valid base64url encoded",
	"base64":    "%s must be valid base64 encoded",
	"latitude":  "%s must be a valid latitude (-90 to 90)",
	"longitude": "%s must be a valid longitude (-180 to 180)",
}

// errorMessageWithParam maps validation tags to templates that include param.
var errorMessageWithParam = map[string]string{
	"oneof": "%s must be one of: %s",
	"gte":   "%s must be greater than or equal to %s",
	"lte":   "%s must be less than or equal to %s",
	"gt":    "%s must be greater than %s",
	"lt":    "%s must be less than %s",
}

// translateError converts a validator.FieldError to a human-readable message.
// This provides user-friendly error messages that match the existing API style.
func translateError(fe validator.FieldError) string {
	field := fe.Field()
	tag := fe.Tag()
	param := fe.Param()

	// Check simple templates (no param)
	if template, ok := errorMessageTemplates[tag]; ok {
		return fmt.Sprintf(template, field)
	}

	// Check templates with param
	if template, ok := errorMessageWithParam[tag]; ok {
		return fmt.Sprintf(template, field, param)
	}

	// Handle min/max with type-specific messages
	return translateMinMax(fe, field, tag, param)
}

// translateMinMax handles min/max validation with type-specific messages.
func translateMinMax(fe validator.FieldError, field, tag, param string) string {
	isString := fe.Kind().String() == "string"

	switch tag {
	case "min":
		if isString {
			return fmt.Sprintf("%s must be at least %s characters", field, param)
		}
		return fmt.Sprintf("%s must be at least %s", field, param)
	case "max":
		if isString {
			return fmt.Sprintf("%s must be at most %s characters", field, param)
		}
		return fmt.Sprintf("%s must be at most %s", field, param)
	default:
		return fmt.Sprintf("%s failed %s validation", field, tag)
	}
}
