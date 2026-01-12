// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build nats

package eventprocessor

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	gobreaker "github.com/sony/gobreaker/v2"

	"github.com/tomtom215/cartographus/internal/logging"
)

// ResilientReaderConfig configures the resilient reader behavior.
type ResilientReaderConfig struct {
	// NATSURL is the NATS server connection URL.
	NATSURL string

	// CircuitBreakerName identifies the circuit breaker instance.
	CircuitBreakerName string

	// MaxRequests is the number of requests allowed in half-open state.
	MaxRequests uint32

	// Interval is the cyclic reset period for counts.
	Interval time.Duration

	// Timeout is the duration in open state before transitioning to half-open.
	Timeout time.Duration

	// FailureThreshold is the number of consecutive failures before opening.
	FailureThreshold uint32

	// HealthCheckInterval is how often to check primary reader availability.
	HealthCheckInterval time.Duration

	// EnablePrimaryReader enables the primary nats_js extension reader.
	// If false, always uses fallback reader (useful when extension unavailable).
	EnablePrimaryReader bool
}

// DefaultResilientReaderConfig returns production defaults.
func DefaultResilientReaderConfig(natsURL string) ResilientReaderConfig {
	return ResilientReaderConfig{
		NATSURL:             natsURL,
		CircuitBreakerName:  "stream-reader",
		MaxRequests:         3,
		Interval:            30 * time.Second,
		Timeout:             10 * time.Second,
		FailureThreshold:    5,
		HealthCheckInterval: 30 * time.Second,
		EnablePrimaryReader: false, // Default to fallback until nats_js extension is available
	}
}

// ResilientReader provides automatic failover between primary and fallback readers.
// It uses a circuit breaker to detect primary reader failures and automatically
// falls back to the Go NATS client when the primary is unavailable.
type ResilientReader struct {
	config          ResilientReaderConfig
	primary         StreamReader // nats_js extension reader (may be nil)
	fallback        *FallbackReader
	circuitBreaker  *gobreaker.CircuitBreaker[[]StreamMessage]
	mu              sync.RWMutex
	closed          bool
	stopHealthCheck chan struct{}

	// Statistics
	queriesTotal     atomic.Int64
	errorsTotal      atomic.Int64
	fallbacksTotal   atomic.Int64
	lastQueryTime    atomic.Value // time.Time
	currentReader    atomic.Value // ReaderType
	primaryAvailable atomic.Bool
}

// NewResilientReader creates a resilient reader with automatic fallback.
func NewResilientReader(cfg *ResilientReaderConfig) (*ResilientReader, error) {
	// Create fallback reader (always needed)
	fallback, err := NewFallbackReader(cfg.NATSURL)
	if err != nil {
		return nil, fmt.Errorf("create fallback reader: %w", err)
	}

	r := &ResilientReader{
		config:          *cfg,
		fallback:        fallback,
		stopHealthCheck: make(chan struct{}),
	}

	// Initialize atomic values
	r.currentReader.Store(ReaderTypeFallback)
	r.lastQueryTime.Store(time.Time{})
	r.primaryAvailable.Store(false)

	// Configure circuit breaker for primary reader
	cbSettings := gobreaker.Settings{
		Name:        cfg.CircuitBreakerName,
		MaxRequests: cfg.MaxRequests,
		Interval:    cfg.Interval,
		Timeout:     cfg.Timeout,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= cfg.FailureThreshold
		},
		OnStateChange: func(name string, from, to gobreaker.State) {
			// Log state changes for observability
			if to == gobreaker.StateOpen {
				r.currentReader.Store(ReaderTypeFallback)
			}
		},
	}
	r.circuitBreaker = gobreaker.NewCircuitBreaker[[]StreamMessage](cbSettings)

	// Start health check routine if primary reader is enabled
	if cfg.EnablePrimaryReader && cfg.HealthCheckInterval > 0 {
		go r.healthCheckLoop()
	}

	return r, nil
}

// SetPrimaryReader sets the primary reader (nats_js extension).
// This allows late binding when the extension becomes available.
func (r *ResilientReader) SetPrimaryReader(primary StreamReader) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.primary = primary
	r.primaryAvailable.Store(primary != nil)
}

// Query retrieves messages using the best available reader.
func (r *ResilientReader) Query(ctx context.Context, stream string, opts *QueryOptions) ([]StreamMessage, error) {
	r.mu.RLock()
	if r.closed {
		r.mu.RUnlock()
		return nil, fmt.Errorf("reader is closed")
	}
	primary := r.primary
	r.mu.RUnlock()

	r.queriesTotal.Add(1)
	r.lastQueryTime.Store(time.Now())

	// Try primary reader with circuit breaker if available
	if primary != nil && r.config.EnablePrimaryReader {
		result, err := r.circuitBreaker.Execute(func() ([]StreamMessage, error) {
			return primary.Query(ctx, stream, opts)
		})
		if err == nil {
			r.currentReader.Store(ReaderTypeNatsJS)
			return result, nil
		}
		// Circuit breaker open or primary failed
		r.errorsTotal.Add(1)
		r.fallbacksTotal.Add(1)
	}

	// Use fallback reader
	r.currentReader.Store(ReaderTypeFallback)
	result, err := r.fallback.Query(ctx, stream, opts)
	if err != nil {
		r.errorsTotal.Add(1)
		return nil, fmt.Errorf("fallback query: %w", err)
	}
	return result, nil
}

// GetMessage retrieves a single message using the best available reader.
func (r *ResilientReader) GetMessage(ctx context.Context, stream string, seq uint64) (*StreamMessage, error) {
	r.mu.RLock()
	if r.closed {
		r.mu.RUnlock()
		return nil, fmt.Errorf("reader is closed")
	}
	primary := r.primary
	r.mu.RUnlock()

	// Try primary reader with circuit breaker if available
	if primary != nil && r.config.EnablePrimaryReader {
		// Wrap single message in circuit breaker
		result, err := r.circuitBreaker.Execute(func() ([]StreamMessage, error) {
			msg, err := primary.GetMessage(ctx, stream, seq)
			if err != nil {
				return nil, err
			}
			return []StreamMessage{*msg}, nil
		})
		if err == nil && len(result) > 0 {
			r.currentReader.Store(ReaderTypeNatsJS)
			return &result[0], nil
		}
		r.fallbacksTotal.Add(1)
	}

	// Use fallback reader
	r.currentReader.Store(ReaderTypeFallback)
	return r.fallback.GetMessage(ctx, stream, seq)
}

// GetLastSequence returns the latest sequence number.
func (r *ResilientReader) GetLastSequence(ctx context.Context, stream string) (uint64, error) {
	r.mu.RLock()
	if r.closed {
		r.mu.RUnlock()
		return 0, fmt.Errorf("reader is closed")
	}
	primary := r.primary
	r.mu.RUnlock()

	// Try primary reader if available
	if primary != nil && r.config.EnablePrimaryReader {
		seq, err := primary.GetLastSequence(ctx, stream)
		if err == nil {
			return seq, nil
		}
	}

	// Use fallback reader
	return r.fallback.GetLastSequence(ctx, stream)
}

// Health checks the current reader availability.
func (r *ResilientReader) Health(ctx context.Context) error {
	r.mu.RLock()
	if r.closed {
		r.mu.RUnlock()
		return fmt.Errorf("reader is closed")
	}
	r.mu.RUnlock()

	// Check fallback (always required)
	return r.fallback.Health(ctx)
}

// Close releases all resources.
func (r *ResilientReader) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return nil
	}
	r.closed = true

	// Stop health check routine
	close(r.stopHealthCheck)

	// Close primary if present
	if r.primary != nil {
		if err := r.primary.Close(); err != nil {
			// Log but continue - still need to close fallback
			logging.Warn().Err(err).Msg("Error closing primary reader")
		}
	}

	// Close fallback
	return r.fallback.Close()
}

// Stats returns current reader statistics.
func (r *ResilientReader) Stats() ReaderStats {
	var lastQuery time.Time
	if t, ok := r.lastQueryTime.Load().(time.Time); ok {
		lastQuery = t
	}
	var currentReader ReaderType
	if rt, ok := r.currentReader.Load().(ReaderType); ok {
		currentReader = rt
	}

	return ReaderStats{
		CurrentReader:       currentReader,
		CircuitBreakerState: r.circuitBreaker.State().String(),
		PrimaryAvailable:    r.primaryAvailable.Load(),
		QueriesTotal:        r.queriesTotal.Load(),
		ErrorsTotal:         r.errorsTotal.Load(),
		LastQueryTime:       lastQuery,
	}
}

// FallbackCount returns the total number of fallback operations.
func (r *ResilientReader) FallbackCount() int64 {
	return r.fallbacksTotal.Load()
}

// healthCheckLoop periodically checks primary reader availability.
func (r *ResilientReader) healthCheckLoop() {
	ticker := time.NewTicker(r.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-r.stopHealthCheck:
			return
		case <-ticker.C:
			r.checkPrimaryHealth()
		}
	}
}

// checkPrimaryHealth verifies primary reader availability.
func (r *ResilientReader) checkPrimaryHealth() {
	r.mu.RLock()
	primary := r.primary
	r.mu.RUnlock()

	if primary == nil {
		r.primaryAvailable.Store(false)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := primary.Health(ctx)
	r.primaryAvailable.Store(err == nil)
}
