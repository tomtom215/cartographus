// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build nats

package eventprocessor

import (
	"context"
	"errors"
	"math"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tomtom215/cartographus/internal/cache"
	"github.com/tomtom215/cartographus/internal/metrics"
)

// ErrorCategory categorizes errors for DLQ routing and metrics.
type ErrorCategory int

const (
	// ErrorCategoryUnknown is the default category for unclassified errors.
	ErrorCategoryUnknown ErrorCategory = iota
	// ErrorCategoryConnection indicates network or connection failures.
	ErrorCategoryConnection
	// ErrorCategoryTimeout indicates operation timeout.
	ErrorCategoryTimeout
	// ErrorCategoryValidation indicates data validation failures.
	ErrorCategoryValidation
	// ErrorCategoryDatabase indicates database operation failures.
	ErrorCategoryDatabase
	// ErrorCategoryCapacity indicates resource capacity issues.
	ErrorCategoryCapacity
)

// String returns the string representation of the error category.
func (c ErrorCategory) String() string {
	switch c {
	case ErrorCategoryConnection:
		return "connection"
	case ErrorCategoryTimeout:
		return "timeout"
	case ErrorCategoryValidation:
		return "validation"
	case ErrorCategoryDatabase:
		return "database"
	case ErrorCategoryCapacity:
		return "capacity"
	default:
		return "unknown"
	}
}

// RetryableError represents an error that can be retried.
// These errors are typically transient (network issues, timeouts).
type RetryableError struct {
	Message  string
	Cause    error
	Category ErrorCategory
}

// NewRetryableError creates a new retryable error.
func NewRetryableError(message string, cause error) *RetryableError {
	category := categorizeErrorMessage(message)
	return &RetryableError{
		Message:  message,
		Cause:    cause,
		Category: category,
	}
}

// Error implements the error interface.
func (e *RetryableError) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

// Unwrap returns the underlying cause for error unwrapping.
func (e *RetryableError) Unwrap() error {
	return e.Cause
}

// PermanentError represents an error that should not be retried.
// These errors indicate unrecoverable issues (validation, malformed data).
type PermanentError struct {
	Message  string
	Cause    error
	Category ErrorCategory
}

// NewPermanentError creates a new permanent error.
func NewPermanentError(message string, cause error) *PermanentError {
	category := categorizeErrorMessage(message)
	if category == ErrorCategoryUnknown {
		category = ErrorCategoryValidation
	}
	return &PermanentError{
		Message:  message,
		Cause:    cause,
		Category: category,
	}
}

// Error implements the error interface.
func (e *PermanentError) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

// Unwrap returns the underlying cause for error unwrapping.
func (e *PermanentError) Unwrap() error {
	return e.Cause
}

// categorizeErrorMessage attempts to categorize an error based on its message.
func categorizeErrorMessage(message string) ErrorCategory {
	switch {
	case containsAny(message, "connection", "connect", "refused", "reset", "network"):
		return ErrorCategoryConnection
	case containsAny(message, "timeout", "deadline", "timed out"):
		return ErrorCategoryTimeout
	case containsAny(message, "invalid", "validation", "malformed", "parse"):
		return ErrorCategoryValidation
	case containsAny(message, "database", "db", "sql", "query"):
		return ErrorCategoryDatabase
	case containsAny(message, "capacity", "full", "limit", "exceeded"):
		return ErrorCategoryCapacity
	default:
		return ErrorCategoryUnknown
	}
}

// containsAny checks if the string contains any of the substrings.
func containsAny(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if containsIgnoreCase(s, sub) {
			return true
		}
	}
	return false
}

// containsIgnoreCase performs case-insensitive substring search.
func containsIgnoreCase(s, substr string) bool {
	if len(substr) > len(s) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			c1 := s[i+j]
			c2 := substr[j]
			// Convert to lowercase for comparison
			if c1 >= 'A' && c1 <= 'Z' {
				c1 += 'a' - 'A'
			}
			if c2 >= 'A' && c2 <= 'Z' {
				c2 += 'a' - 'A'
			}
			if c1 != c2 {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

// IsRetryableError checks if the error is retryable.
func IsRetryableError(err error) bool {
	var retryErr *RetryableError
	return errors.As(err, &retryErr)
}

// IsPermanentError checks if the error is permanent (non-retryable).
func IsPermanentError(err error) bool {
	var permErr *PermanentError
	return errors.As(err, &permErr)
}

// DLQEntry represents a failed message entry in the Dead Letter Queue.
type DLQEntry struct {
	// Event is the original media event that failed processing.
	Event *MediaEvent

	// MessageID is the original NATS message ID.
	MessageID string

	// OriginalError is the error message from the first failure.
	OriginalError string

	// LastError is the error message from the most recent retry attempt.
	LastError string

	// RetryCount is the number of retry attempts made.
	RetryCount int

	// FirstFailure is when the event first failed.
	FirstFailure time.Time

	// LastFailure is when the most recent failure occurred.
	LastFailure time.Time

	// NextRetry is the earliest time for the next retry attempt.
	NextRetry time.Time

	// Category is the error category for routing and metrics.
	Category ErrorCategory
}

// NewDLQEntry creates a new DLQ entry for a failed event.
func NewDLQEntry(event *MediaEvent, err error, messageID string) *DLQEntry {
	now := time.Now()
	category := ErrorCategoryUnknown

	var retryErr *RetryableError
	var permErr *PermanentError
	if errors.As(err, &retryErr) {
		category = retryErr.Category
	} else if errors.As(err, &permErr) {
		category = permErr.Category
	}

	return &DLQEntry{
		Event:         event,
		MessageID:     messageID,
		OriginalError: err.Error(),
		LastError:     err.Error(),
		RetryCount:    0,
		FirstFailure:  now,
		LastFailure:   now,
		NextRetry:     now, // Immediately ready for first retry after backoff
		Category:      category,
	}
}

// DLQConfig holds configuration for the Dead Letter Queue handler.
type DLQConfig struct {
	// MaxRetries is the maximum number of retry attempts before permanent failure.
	MaxRetries int

	// MaxEntries is the maximum number of entries to keep in the DLQ.
	// When exceeded, oldest entries are evicted.
	MaxEntries int

	// RetentionTime is how long to keep entries before automatic cleanup.
	RetentionTime time.Duration

	// InitialBackoff is the initial backoff duration for retries.
	InitialBackoff time.Duration

	// MaxBackoff is the maximum backoff duration.
	MaxBackoff time.Duration

	// BackoffMultiplier is the exponential backoff multiplier (default: 2.0).
	BackoffMultiplier float64

	// JitterFraction is the random jitter fraction (0.0-1.0, default: 0.1).
	JitterFraction float64

	// RandomSeed is the seed for the random number generator used for jitter.
	// DETERMINISM: When set to a non-zero value, provides reproducible jitter
	// for testing scenarios. When 0 (default), uses time-based seed for production.
	RandomSeed int64
}

// DefaultDLQConfig returns production defaults for DLQ configuration.
func DefaultDLQConfig() DLQConfig {
	return DLQConfig{
		MaxRetries:        5,
		MaxEntries:        10000,
		RetentionTime:     7 * 24 * time.Hour, // 7 days
		InitialBackoff:    time.Second,
		MaxBackoff:        time.Minute,
		BackoffMultiplier: 2.0,
		JitterFraction:    0.1,
	}
}

// DLQStats holds runtime statistics for the DLQ.
type DLQStats struct {
	TotalEntries      int64     // Current number of entries
	TotalAdded        int64     // Total entries added
	TotalRemoved      int64     // Total entries removed (success or cleanup)
	TotalRetries      int64     // Total retry attempts
	TotalExpired      int64     // Total entries expired by cleanup
	OldestEntry       time.Time // Timestamp of oldest entry
	NewestEntry       time.Time // Timestamp of newest entry
	EntriesByCategory map[ErrorCategory]int64
}

// DLQHandler manages the Dead Letter Queue for failed messages.
// It provides retry scheduling, entry management, and cleanup.
//
// Performance: Uses MinHeap for O(log n) eviction instead of O(n) linear scan.
type DLQHandler struct {
	config DLQConfig

	// Entry storage using MinHeap for O(log n) eviction
	// Heap is ordered by FirstFailure timestamp
	mu      sync.RWMutex
	entries *cache.MinHeap[*DLQEntry]

	// Statistics
	totalAdded   atomic.Int64
	totalRemoved atomic.Int64
	totalRetries atomic.Int64
	totalExpired atomic.Int64

	// Random source for jitter (using atomic pointer for thread safety)
	randMu sync.Mutex
	rng    *rand.Rand
}

// NewDLQHandler creates a new Dead Letter Queue handler.
func NewDLQHandler(cfg DLQConfig) (*DLQHandler, error) {
	if cfg.MaxRetries <= 0 {
		return nil, errors.New("max retries must be positive")
	}
	if cfg.MaxEntries <= 0 {
		return nil, errors.New("max entries must be positive")
	}
	if cfg.InitialBackoff <= 0 {
		return nil, errors.New("initial backoff must be positive")
	}
	if cfg.MaxBackoff <= 0 {
		cfg.MaxBackoff = cfg.InitialBackoff * 64 // Default to reasonable max
	}
	if cfg.BackoffMultiplier <= 0 {
		cfg.BackoffMultiplier = 2.0
	}
	if cfg.JitterFraction <= 0 || cfg.JitterFraction > 1.0 {
		cfg.JitterFraction = 0.1
	}

	// DETERMINISM: Use configurable seed for reproducible testing.
	// When RandomSeed is 0 (default), use time-based seed for production randomness.
	// When non-zero, use the provided seed for deterministic jitter in tests.
	seed := cfg.RandomSeed
	if seed == 0 {
		seed = time.Now().UnixNano()
	}

	return &DLQHandler{
		config: cfg,
		// MinHeap with max capacity for automatic O(log n) eviction
		entries: cache.NewMinHeap[*DLQEntry](cfg.MaxEntries),
		//nolint:gosec // G404: Using weak random for non-cryptographic jitter in backoff timing
		rng: rand.New(rand.NewSource(seed)),
	}, nil
}

// AddEntry adds a failed event to the DLQ.
// Returns the created entry.
//
// Performance: Uses MinHeap for O(log n) insertion and automatic eviction.
func (h *DLQHandler) AddEntry(event *MediaEvent, err error, messageID string) *DLQEntry {
	entry := NewDLQEntry(event, err, messageID)

	h.mu.Lock()
	defer h.mu.Unlock()

	// Calculate initial backoff
	entry.NextRetry = time.Now().Add(h.calculateBackoffLocked(0))

	// Push to heap - automatic O(log n) eviction if at capacity
	evicted := h.entries.Push(event.EventID, entry, entry.FirstFailure)

	// Record eviction metrics if an entry was evicted
	if evicted != nil {
		h.totalExpired.Add(1)
		metrics.RecordDLQExpiry(evicted.Value.Category.String())
	}

	h.totalAdded.Add(1)

	// Record metrics
	metrics.RecordDLQEntry(entry.Category.String())

	return entry
}

// GetEntry retrieves an entry by event ID.
// Returns nil if not found.
func (h *DLQHandler) GetEntry(eventID string) *DLQEntry {
	h.mu.RLock()
	defer h.mu.RUnlock()

	heapEntry := h.entries.Get(eventID)
	if heapEntry == nil {
		return nil
	}
	return heapEntry.Value
}

// IncrementRetry increments the retry count and updates the next retry time.
// Returns true if more retries are allowed, false if max retries reached.
func (h *DLQHandler) IncrementRetry(eventID string, err error) bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	heapEntry := h.entries.Get(eventID)
	if heapEntry == nil {
		return false
	}

	entry := heapEntry.Value
	entry.RetryCount++
	entry.LastError = err.Error()
	entry.LastFailure = time.Now()
	entry.NextRetry = time.Now().Add(h.calculateBackoffLocked(entry.RetryCount))

	h.totalRetries.Add(1)

	moreRetries := entry.RetryCount < h.config.MaxRetries
	// Record retry metrics (failed if no more retries allowed)
	metrics.RecordDLQRetry(moreRetries)

	return moreRetries
}

// RemoveEntry removes an entry from the DLQ.
// Returns true if the entry was found and removed.
func (h *DLQHandler) RemoveEntry(eventID string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	removed := h.entries.Remove(eventID)
	if removed != nil {
		category := removed.Value.Category.String()
		h.totalRemoved.Add(1)
		// Record removal metrics
		metrics.RecordDLQRemoval(category)
		return true
	}
	return false
}

// GetPendingRetries returns entries that are ready for retry.
// Entries are ready when their NextRetry time has passed.
func (h *DLQHandler) GetPendingRetries() []*DLQEntry {
	h.mu.RLock()
	defer h.mu.RUnlock()

	now := time.Now()
	var pending []*DLQEntry

	for _, heapEntry := range h.entries.All() {
		entry := heapEntry.Value
		if entry.RetryCount < h.config.MaxRetries && !entry.NextRetry.After(now) {
			pending = append(pending, entry)
		}
	}

	return pending
}

// ListEntries returns all entries in the DLQ.
func (h *DLQHandler) ListEntries() []*DLQEntry {
	h.mu.RLock()
	defer h.mu.RUnlock()

	heapEntries := h.entries.All()
	entries := make([]*DLQEntry, 0, len(heapEntries))
	for _, heapEntry := range heapEntries {
		entries = append(entries, heapEntry.Value)
	}
	return entries
}

// Cleanup removes expired entries based on retention time.
// Returns the number of entries cleaned up.
//
// Performance: Uses MinHeap.PopBefore for O(k log n) removal of k expired entries.
func (h *DLQHandler) Cleanup() int {
	h.mu.Lock()
	defer h.mu.Unlock()

	cutoff := time.Now().Add(-h.config.RetentionTime)

	// PopBefore efficiently removes all entries older than cutoff
	removed := h.entries.PopBefore(cutoff)

	for _, heapEntry := range removed {
		h.totalExpired.Add(1)
		// Record expiry metrics
		metrics.RecordDLQExpiry(heapEntry.Value.Category.String())
	}

	return len(removed)
}

// Stats returns current DLQ statistics.
// This method also updates Prometheus gauge metrics.
func (h *DLQHandler) Stats() DLQStats {
	h.mu.RLock()
	defer h.mu.RUnlock()

	stats := DLQStats{
		TotalEntries:      int64(h.entries.Len()),
		TotalAdded:        h.totalAdded.Load(),
		TotalRemoved:      h.totalRemoved.Load(),
		TotalRetries:      h.totalRetries.Load(),
		TotalExpired:      h.totalExpired.Load(),
		EntriesByCategory: make(map[ErrorCategory]int64),
	}

	for _, heapEntry := range h.entries.All() {
		entry := heapEntry.Value
		stats.EntriesByCategory[entry.Category]++

		if stats.OldestEntry.IsZero() || entry.FirstFailure.Before(stats.OldestEntry) {
			stats.OldestEntry = entry.FirstFailure
		}
		if stats.NewestEntry.IsZero() || entry.FirstFailure.After(stats.NewestEntry) {
			stats.NewestEntry = entry.FirstFailure
		}
	}

	// Update Prometheus gauge metrics
	oldestAge := float64(0)
	if !stats.OldestEntry.IsZero() {
		oldestAge = time.Since(stats.OldestEntry).Seconds()
	}
	entriesByCategory := make(map[string]int64)
	for cat, count := range stats.EntriesByCategory {
		entriesByCategory[cat.String()] = count
	}
	metrics.UpdateDLQGauges(stats.TotalEntries, oldestAge, entriesByCategory)

	return stats
}

// calculateBackoffLocked calculates the backoff duration for a given retry count.
// Must be called with h.mu held or h.randMu held.
func (h *DLQHandler) calculateBackoffLocked(retryCount int) time.Duration {
	// Calculate base backoff with exponential growth
	backoff := float64(h.config.InitialBackoff) * math.Pow(h.config.BackoffMultiplier, float64(retryCount))

	// Cap at max backoff
	if backoff > float64(h.config.MaxBackoff) {
		backoff = float64(h.config.MaxBackoff)
	}

	// Add jitter
	h.randMu.Lock()
	jitter := backoff * h.config.JitterFraction * (h.rng.Float64()*2 - 1) // -jitter to +jitter
	h.randMu.Unlock()

	return time.Duration(backoff + jitter)
}

// Note: evictOldestLocked was removed - MinHeap handles eviction automatically
// with O(log n) complexity instead of O(n) linear scan.

// RetryPolicy defines the retry behavior for failed operations.
type RetryPolicy struct {
	// MaxRetries is the maximum number of retry attempts.
	MaxRetries int

	// InitialBackoff is the initial backoff duration.
	InitialBackoff time.Duration

	// MaxBackoff is the maximum backoff duration.
	MaxBackoff time.Duration

	// BackoffMultiplier is the exponential multiplier.
	BackoffMultiplier float64

	// JitterFraction is the random jitter fraction (0.0-1.0).
	JitterFraction float64

	rng   *rand.Rand
	rngMu sync.Mutex
}

// DefaultRetryPolicy returns production defaults for retry policy.
func DefaultRetryPolicy() *RetryPolicy {
	return NewRetryPolicyWithSeed(0)
}

// NewRetryPolicyWithSeed creates a RetryPolicy with a specific random seed.
// DETERMINISM: When seed is 0, uses time-based seed for production randomness.
// When non-zero, uses the provided seed for deterministic jitter in tests.
func NewRetryPolicyWithSeed(seed int64) *RetryPolicy {
	if seed == 0 {
		seed = time.Now().UnixNano()
	}
	return &RetryPolicy{
		MaxRetries:        5,
		InitialBackoff:    time.Second,
		MaxBackoff:        time.Minute,
		BackoffMultiplier: 2.0,
		JitterFraction:    0.1,
		//nolint:gosec // G404: Using weak random for non-cryptographic jitter in backoff timing
		rng: rand.New(rand.NewSource(seed)),
	}
}

// CalculateBackoff calculates the backoff duration for a given retry count.
func (p *RetryPolicy) CalculateBackoff(retryCount int) time.Duration {
	// Calculate base backoff with exponential growth
	backoff := float64(p.InitialBackoff) * math.Pow(p.BackoffMultiplier, float64(retryCount))

	// Cap at max backoff
	if backoff > float64(p.MaxBackoff) {
		backoff = float64(p.MaxBackoff)
	}

	// Add jitter
	p.rngMu.Lock()
	jitter := backoff * p.JitterFraction * (p.rng.Float64()*2 - 1) // -jitter to +jitter
	p.rngMu.Unlock()

	return time.Duration(backoff + jitter)
}

// ShouldRetry determines if an error should be retried.
func (p *RetryPolicy) ShouldRetry(err error, retryCount int) bool {
	// Check if max retries exceeded
	if retryCount >= p.MaxRetries {
		return false
	}

	// Permanent errors should not be retried
	if IsPermanentError(err) {
		return false
	}

	return true
}

// RetryHandler is a function that attempts to reprocess a DLQ entry.
// Returns nil on success, or an error if retry failed.
type RetryHandler func(entry *DLQEntry) error

// DLQAutoRetryConfig configures automatic retry behavior.
type DLQAutoRetryConfig struct {
	// RetryInterval is how often to check for pending retries.
	RetryInterval time.Duration

	// MaxConcurrentRetries limits concurrent retry operations.
	MaxConcurrentRetries int
}

// DefaultDLQAutoRetryConfig returns production defaults.
func DefaultDLQAutoRetryConfig() DLQAutoRetryConfig {
	return DLQAutoRetryConfig{
		RetryInterval:        30 * time.Second,
		MaxConcurrentRetries: 5,
	}
}

// AutoRetryWorker processes pending DLQ entries automatically.
// Phase 2.5: Implements background auto-retry for failed events.
type AutoRetryWorker struct {
	dlq     *DLQHandler
	handler RetryHandler
	config  DLQAutoRetryConfig
}

// NewAutoRetryWorker creates a new auto-retry worker.
func NewAutoRetryWorker(dlq *DLQHandler, handler RetryHandler, config DLQAutoRetryConfig) *AutoRetryWorker {
	return &AutoRetryWorker{
		dlq:     dlq,
		handler: handler,
		config:  config,
	}
}

// Start begins the auto-retry background process.
// It runs until the context is canceled.
// Phase 2.5: Implements failed event auto-retry.
func (w *AutoRetryWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.config.RetryInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.processPendingRetries(ctx)
		}
	}
}

// processPendingRetries attempts to retry all pending entries.
func (w *AutoRetryWorker) processPendingRetries(ctx context.Context) {
	entries := w.dlq.GetPendingRetries()
	if len(entries) == 0 {
		return
	}

	// Use semaphore to limit concurrent retries
	sem := make(chan struct{}, w.config.MaxConcurrentRetries)
	var wg sync.WaitGroup

	for _, entry := range entries {
		select {
		case <-ctx.Done():
			return
		case sem <- struct{}{}:
			wg.Add(1)
			go func(e *DLQEntry) {
				defer func() {
					<-sem
					wg.Done()
				}()
				w.retryEntry(ctx, e)
			}(entry)
		}
	}

	wg.Wait()
}

// retryEntry attempts to retry a single DLQ entry.
func (w *AutoRetryWorker) retryEntry(_ context.Context, entry *DLQEntry) {
	// Attempt retry
	err := w.handler(entry)
	if err != nil {
		// Retry failed - record failure and increment retry count
		metrics.RecordDLQRetry(false)
		w.dlq.IncrementRetry(entry.Event.EventID, err)
		// Entry stays in DLQ for next retry or manual intervention
		return
	}

	// Retry succeeded - record success and remove from DLQ
	metrics.RecordDLQRetry(true)
	w.dlq.RemoveEntry(entry.Event.EventID)
}
