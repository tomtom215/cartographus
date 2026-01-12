# ADR-0013: Request Validation with go-playground/validator

**Date**: 2025-12-04
**Status**: Accepted

---

## Context

Cartographus API endpoints require request validation:

1. **Query Parameters**: Date ranges, pagination, filters
2. **Request Bodies**: JSON payloads for POST/PUT
3. **Path Parameters**: IDs, slugs
4. **Security**: Prevent injection, malformed input

### Issues with Manual Validation

- Scattered validation logic across handlers
- Inconsistent error messages
- Missing edge case coverage
- Duplicate validation code

### Requirements

- Declarative validation via struct tags
- Custom validators for domain-specific rules
- Consistent error responses
- Integration with request parsing

### Alternatives Considered

| Library | Pros | Cons |
|---------|------|------|
| **Manual validation** | No dependencies | Error-prone, verbose |
| **go-playground/validator** | Full-featured, popular | Learning curve |
| **ozzo-validation** | Fluent API | Less ecosystem support |
| **gookit/validate** | Simple API | Less features |

---

## Decision

Use **go-playground/validator v10** for request validation:

- **Struct Tags**: Declarative validation rules
- **Built-in Validators**: Rich set of validators for common patterns
- **Error Translation**: Human-readable messages
- **Integration**: Helper function for consistent validation

### Key Factors

1. **Industry Standard**: Most popular Go validation library
2. **Rich Validators**: 100+ built-in validators
3. **Thread-Safe Singleton**: Validator caches struct info for performance
4. **Performance**: Compiled validation, minimal overhead

---

## Consequences

### Positive

- **Declarative**: Validation rules in struct tags
- **Consistent**: Same patterns across all endpoints
- **Testable**: Validators can be unit tested
- **Extensible**: Custom validators can be added if needed
- **Error Messages**: Clear, actionable feedback with field context

### Negative

- **Additional Dependency**: `github.com/go-playground/validator/v10`
- **Struct Tags**: Can become verbose
- **Learning Curve**: Tag syntax to learn

### Neutral

- **Validation Location**: Validated in handlers before business logic
- **Custom Error Format**: Translated to VALIDATION_ERROR API format

---

## Implementation

### Request Structs with Validation

```go
// internal/api/requests.go
type PlaybacksRequest struct {
    Limit  int    `validate:"min=1,max=1000"`
    Offset int    `validate:"min=0,max=1000000"`
    Cursor string `validate:"omitempty,base64url"`
}

type LocationsRequest struct {
    Limit      int    `validate:"min=1,max=1000"`
    Days       int    `validate:"omitempty,min=1,max=3650"`
    StartDate  string `validate:"omitempty,datetime=2006-01-02T15:04:05Z07:00"`
    EndDate    string `validate:"omitempty,datetime=2006-01-02T15:04:05Z07:00"`
    Users      string // Comma-separated, no validation needed
    MediaTypes string // Comma-separated, no validation needed
}

type CreateBackupRequestValidation struct {
    Type  string `validate:"omitempty,oneof=full database config"`
    Notes string `validate:"omitempty,max=500"`
}

type LoginRequestValidation struct {
    Username   string `validate:"required,min=1"`
    Password   string `validate:"required,min=1"`
    RememberMe bool
}

type SpatialViewportRequest struct {
    West      float64 `validate:"min=-180,max=180"`
    South     float64 `validate:"min=-90,max=90"`
    East      float64 `validate:"min=-180,max=180"`
    North     float64 `validate:"min=-90,max=90"`
    StartDate string  // Date validation done by parseDateFilter
    EndDate   string  // Date validation done by parseDateFilter
}

type SpatialNearbyRequest struct {
    Lat       float64 `validate:"latitude"`
    Lon       float64 `validate:"longitude"`
    Radius    float64 `validate:"min=1,max=20000"`
    StartDate string  // Date validation done by parseDateFilter
    EndDate   string  // Date validation done by parseDateFilter
}
```

### Validator Initialization

```go
// internal/validation/validator.go
import (
    "sync"
    "github.com/go-playground/validator/v10"
)

var (
    validate     *validator.Validate
    validateOnce sync.Once
)

// GetValidator returns the singleton validator instance.
// The validator is initialized once with options for v11+ compatibility.
func GetValidator() *validator.Validate {
    validateOnce.Do(func() {
        validate = validator.New(validator.WithRequiredStructEnabled())
        // Built-in validators cover most needs:
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
func ValidateStruct(s interface{}) *RequestValidationError {
    v := GetValidator()
    err := v.Struct(s)
    if err == nil {
        return nil
    }
    // Convert validator errors to RequestValidationError
    // ... (error translation logic)
}
```

### Handler Integration

```go
// internal/api/handlers_helpers.go

// validateRequest validates a struct using go-playground/validator.
// Returns nil if validation passes, or a models.APIError if validation fails.
func validateRequest(v interface{}) *models.APIError {
    validationErr := validation.ValidateStruct(v)
    if validationErr == nil {
        return nil
    }
    apiErr := validationErr.ToAPIError()
    return &models.APIError{
        Code:    apiErr.Code,
        Message: apiErr.Message,
        Details: apiErr.Details,
    }
}

// internal/api/handlers_spatial.go
func (h *Handler) HandleExportPlaybacksCSV(w http.ResponseWriter, r *http.Request) {
    limit := getIntParam(r, "limit", 10000)
    offset := getIntParam(r, "offset", 0)

    req := ExportPlaybacksCSVRequest{
        Limit:  limit,
        Offset: offset,
    }
    if apiErr := validateRequest(&req); apiErr != nil {
        respondError(w, http.StatusBadRequest, apiErr.Code, apiErr.Message, nil)
        return
    }
    // Proceed with validated request...
}

// internal/api/handlers_backup.go
func (h *Handler) HandleCreateBackup(w http.ResponseWriter, r *http.Request) {
    var req CreateBackupRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        req.Type = "full" // Default to full backup if no body provided
    }

    validationReq := CreateBackupRequestValidation(req)
    if apiErr := validateRequest(&validationReq); apiErr != nil {
        respondError(w, http.StatusBadRequest, apiErr.Code, apiErr.Message, nil)
        return
    }
    // Proceed with validated request...
}
```

### Error Response Formatting

```go
// internal/validation/validator.go

// ValidationError represents a single field validation error.
type ValidationError struct {
    field   string
    tag     string
    param   string
    value   interface{}
    message string
}

// RequestValidationError represents a collection of validation errors.
type RequestValidationError struct {
    errors []ValidationError
}

// ToAPIError converts validation errors to the application's APIError format.
func (ve *RequestValidationError) ToAPIError() *APIError {
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
        Details: map[string]interface{}{"fields": fields},
    }
}

// translateError converts a validator.FieldError to a human-readable message.
func translateError(fe validator.FieldError) string {
    field := fe.Field()
    tag := fe.Tag()
    param := fe.Param()

    switch tag {
    case "required":
        return fmt.Sprintf("%s is required", field)
    case "min":
        if fe.Kind().String() == "string" {
            return fmt.Sprintf("%s must be at least %s characters", field, param)
        }
        return fmt.Sprintf("%s must be at least %s", field, param)
    case "max":
        if fe.Kind().String() == "string" {
            return fmt.Sprintf("%s must be at most %s characters", field, param)
        }
        return fmt.Sprintf("%s must be at most %s", field, param)
    case "oneof":
        return fmt.Sprintf("%s must be one of: %s", field, param)
    case "latitude":
        return fmt.Sprintf("%s must be a valid latitude (-90 to 90)", field)
    case "longitude":
        return fmt.Sprintf("%s must be a valid longitude (-180 to 180)", field)
    default:
        return fmt.Sprintf("%s failed %s validation", field, tag)
    }
}
```

### API Response Example

```json
// Invalid request
POST /api/v1/backups
{"type": "invalid", "notes": "..."}

// Response (400 Bad Request)
{
  "status": "error",
  "data": null,
  "metadata": {"timestamp": "2025-01-10T12:00:00Z"},
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Type must be one of: full database config",
    "details": {"field": "Type", "tag": "oneof", "value": "invalid"}
  }
}
```

### Built-in Validators Used

| Validator | Description | Example |
|-----------|-------------|---------|
| `required` | Field must be present | `validate:"required"` |
| `omitempty` | Skip if empty | `validate:"omitempty,min=1"` |
| `min` | Minimum value/length | `validate:"min=1"` |
| `max` | Maximum value/length | `validate:"max=1000"` |
| `oneof` | One of values | `validate:"oneof=full database config"` |
| `datetime` | Date/time format | `validate:"datetime=2006-01-02T15:04:05Z07:00"` |
| `base64url` | URL-safe base64 | `validate:"base64url"` |
| `latitude` | Valid latitude | `validate:"latitude"` |
| `longitude` | Valid longitude | `validate:"longitude"` |

### Code References

| Component | File | Notes |
|-----------|------|-------|
| Validator singleton | `internal/validation/validator.go` | Thread-safe init, error translation |
| Request types | `internal/api/requests.go` | Struct definitions with validation tags |
| Helper function | `internal/api/handlers_helpers.go` | validateRequest, respondError |
| Handler examples | `internal/api/handlers_*.go` | Integration patterns |

---

## Verification

### Verified Claims

| Claim | Source | Verified |
|-------|--------|----------|
| validator v10.30.1 | `go.mod` | Yes |
| Singleton validator | `internal/validation/validator.go` | Yes |
| Error translation | `internal/validation/validator.go:224-284` | Yes |
| Handler integration | `internal/api/handlers_backup.go`, `handlers_spatial.go` | Yes |

### Test Coverage

- Validator tests: `internal/validation/validator_test.go`
- Request validation tests: `internal/api/requests_test.go`
- Handler integration tests: `internal/api/handlers_*_test.go`
- Coverage target: 90%+ for validation functions

---

## Related ADRs

- [ADR-0003](0003-authentication-architecture.md): Auth request validation
- [ADR-0012](0012-configuration-management-koanf.md): Config validation

---

## References

- [go-playground/validator](https://github.com/go-playground/validator)
- [Validator Documentation](https://pkg.go.dev/github.com/go-playground/validator/v10)
- [Built-in Validators](https://pkg.go.dev/github.com/go-playground/validator/v10#hdr-Baked_In_Validators_and_Tags)
