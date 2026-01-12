// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build !nats

package eventprocessor

import (
	"time"
)

// ErrorCategory categorizes errors for DLQ routing and metrics.
type ErrorCategory int

const (
	ErrorCategoryUnknown ErrorCategory = iota
	ErrorCategoryConnection
	ErrorCategoryTimeout
	ErrorCategoryValidation
	ErrorCategoryDatabase
	ErrorCategoryCapacity
)

// String returns the string representation of the error category.
func (c ErrorCategory) String() string {
	return "unknown"
}

// RetryableError represents an error that can be retried.
type RetryableError struct {
	Message  string
	Cause    error
	Category ErrorCategory
}

// NewRetryableError creates a new retryable error.
func NewRetryableError(message string, cause error) *RetryableError {
	return &RetryableError{Message: message, Cause: cause}
}

// Error implements the error interface.
func (e *RetryableError) Error() string {
	return e.Message
}

// Unwrap returns the underlying cause.
func (e *RetryableError) Unwrap() error {
	return e.Cause
}

// PermanentError represents an error that should not be retried.
type PermanentError struct {
	Message  string
	Cause    error
	Category ErrorCategory
}

// NewPermanentError creates a new permanent error.
func NewPermanentError(message string, cause error) *PermanentError {
	return &PermanentError{Message: message, Cause: cause}
}

// Error implements the error interface.
func (e *PermanentError) Error() string {
	return e.Message
}

// Unwrap returns the underlying cause.
func (e *PermanentError) Unwrap() error {
	return e.Cause
}

// IsRetryableError checks if the error is retryable.
func IsRetryableError(err error) bool {
	return false
}

// IsPermanentError checks if the error is permanent.
func IsPermanentError(err error) bool {
	return false
}

// DLQEntry represents a failed message entry in the Dead Letter Queue.
type DLQEntry struct {
	Event         *MediaEvent
	MessageID     string
	OriginalError string
	LastError     string
	RetryCount    int
	FirstFailure  time.Time
	LastFailure   time.Time
	NextRetry     time.Time
	Category      ErrorCategory
}

// NewDLQEntry creates a new DLQ entry for a failed event.
func NewDLQEntry(event *MediaEvent, err error, messageID string) *DLQEntry {
	return nil
}

// DLQConfig holds configuration for the Dead Letter Queue handler.
type DLQConfig struct {
	MaxRetries        int
	MaxEntries        int
	RetentionTime     time.Duration
	InitialBackoff    time.Duration
	MaxBackoff        time.Duration
	BackoffMultiplier float64
	JitterFraction    float64
}

// DefaultDLQConfig returns production defaults for DLQ configuration.
func DefaultDLQConfig() DLQConfig {
	return DLQConfig{}
}

// DLQStats holds runtime statistics for the DLQ.
type DLQStats struct {
	TotalEntries      int64
	TotalAdded        int64
	TotalRemoved      int64
	TotalRetries      int64
	TotalExpired      int64
	OldestEntry       time.Time
	NewestEntry       time.Time
	EntriesByCategory map[ErrorCategory]int64
}

// DLQHandler manages the Dead Letter Queue for failed messages.
type DLQHandler struct{}

// NewDLQHandler creates a new Dead Letter Queue handler.
func NewDLQHandler(cfg DLQConfig) (*DLQHandler, error) {
	return nil, ErrNATSNotEnabled
}

// AddEntry adds a failed event to the DLQ.
func (h *DLQHandler) AddEntry(event *MediaEvent, err error, messageID string) *DLQEntry {
	return nil
}

// GetEntry retrieves an entry by event ID.
func (h *DLQHandler) GetEntry(eventID string) *DLQEntry {
	return nil
}

// IncrementRetry increments the retry count.
func (h *DLQHandler) IncrementRetry(eventID string, err error) bool {
	return false
}

// RemoveEntry removes an entry from the DLQ.
func (h *DLQHandler) RemoveEntry(eventID string) bool {
	return false
}

// GetPendingRetries returns entries ready for retry.
func (h *DLQHandler) GetPendingRetries() []*DLQEntry {
	return nil
}

// ListEntries returns all entries in the DLQ.
func (h *DLQHandler) ListEntries() []*DLQEntry {
	return nil
}

// Cleanup removes expired entries.
func (h *DLQHandler) Cleanup() int {
	return 0
}

// Stats returns current DLQ statistics.
func (h *DLQHandler) Stats() DLQStats {
	return DLQStats{}
}

// RetryPolicy defines the retry behavior for failed operations.
type RetryPolicy struct {
	MaxRetries        int
	InitialBackoff    time.Duration
	MaxBackoff        time.Duration
	BackoffMultiplier float64
	JitterFraction    float64
}

// DefaultRetryPolicy returns production defaults for retry policy.
func DefaultRetryPolicy() *RetryPolicy {
	return &RetryPolicy{}
}

// CalculateBackoff calculates the backoff duration.
func (p *RetryPolicy) CalculateBackoff(retryCount int) time.Duration {
	return 0
}

// ShouldRetry determines if an error should be retried.
func (p *RetryPolicy) ShouldRetry(err error, retryCount int) bool {
	return false
}
