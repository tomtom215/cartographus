// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package sync

import (
	"context"
	"errors"
	"time"

	gobreaker "github.com/sony/gobreaker/v2"
	"github.com/tomtom215/cartographus/internal/logging"
	"github.com/tomtom215/cartographus/internal/metrics"
	"github.com/tomtom215/cartographus/internal/models"
)

// Ensure JellyfinCircuitBreakerClient implements JellyfinClientInterface
var _ JellyfinClientInterface = (*JellyfinCircuitBreakerClient)(nil)

// JellyfinCircuitBreakerClient wraps JellyfinClient with circuit breaker pattern
// Prevents cascading failures when Jellyfin API is unavailable or slow
//
// DETERMINISM NOTE: The circuit breaker uses real time (via sony/gobreaker) for its
// interval and timeout calculations. This is intentional for production resilience.
type JellyfinCircuitBreakerClient struct {
	client *JellyfinClient
	cb     *gobreaker.CircuitBreaker[interface{}]
	name   string
}

// JellyfinCircuitBreakerConfig holds configuration for the Jellyfin circuit breaker
type JellyfinCircuitBreakerConfig struct {
	BaseURL string
	APIKey  string
	UserID  string
}

// NewJellyfinCircuitBreakerClient creates a new Jellyfin client with circuit breaker
// Circuit breaker configuration:
// - Max 3 concurrent requests in half-open state
// - 1 minute measurement window
// - 2 minute timeout before attempting recovery
// - Opens after 60% failure rate with minimum 10 requests
func NewJellyfinCircuitBreakerClient(cfg JellyfinCircuitBreakerConfig) *JellyfinCircuitBreakerClient {
	client := NewJellyfinClient(cfg.BaseURL, cfg.APIKey, cfg.UserID)
	cbName := "jellyfin-api"

	// Initialize circuit breaker state metrics
	metrics.CircuitBreakerState.WithLabelValues(cbName).Set(0) // 0 = closed
	metrics.CircuitBreakerConsecutiveFailures.WithLabelValues(cbName).Set(0)

	cb := gobreaker.NewCircuitBreaker[interface{}](gobreaker.Settings{
		Name:        cbName,
		MaxRequests: 3,               // Allow 3 concurrent requests in half-open state
		Interval:    time.Minute,     // Reset counts after 1 minute in closed state
		Timeout:     2 * time.Minute, // Wait 2 minutes before transitioning from open to half-open

		// ReadyToTrip determines when to open the circuit
		// Opens when failure rate >= 60% with minimum 10 requests
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			if counts.Requests < 10 {
				return false // Need at least 10 requests for statistical significance
			}

			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			shouldTrip := failureRatio >= 0.6

			if shouldTrip {
				logging.Warn().Uint32("failures", counts.TotalFailures).Float64("failure_rate", failureRatio*100).Msg("[CIRCUIT BREAKER] Opening Jellyfin circuit")
			}

			return shouldTrip
		},

		// OnStateChange is called whenever the circuit breaker changes state
		OnStateChange: func(name string, from, to gobreaker.State) {
			fromStr := stateToString(from)
			toStr := stateToString(to)

			logging.Info().Str("from", fromStr).Str("to", toStr).Msg("[CIRCUIT BREAKER] Jellyfin state transition")

			// Update metrics
			metrics.CircuitBreakerState.WithLabelValues(name).Set(stateToFloat(to))
			metrics.CircuitBreakerTransitions.WithLabelValues(name, fromStr, toStr).Inc()

			// Reset consecutive failures when transitioning to closed
			if to == gobreaker.StateClosed {
				metrics.CircuitBreakerConsecutiveFailures.WithLabelValues(name).Set(0)
			}
		},
	})

	return &JellyfinCircuitBreakerClient{
		client: client,
		cb:     cb,
		name:   cbName,
	}
}

// execute wraps a Jellyfin API call with circuit breaker protection
func (cbc *JellyfinCircuitBreakerClient) execute(fn func() (interface{}, error)) (interface{}, error) {
	result, err := cbc.cb.Execute(func() (interface{}, error) {
		return fn()
	})

	// Update metrics based on result
	if err != nil {
		if errors.Is(err, gobreaker.ErrOpenState) || errors.Is(err, gobreaker.ErrTooManyRequests) {
			metrics.CircuitBreakerRequests.WithLabelValues(cbc.name, "rejected").Inc()
			logging.Warn().Err(err).Msg("[CIRCUIT BREAKER] Jellyfin request rejected")
		} else {
			metrics.CircuitBreakerRequests.WithLabelValues(cbc.name, "failure").Inc()
			counts := cbc.cb.Counts()
			metrics.CircuitBreakerConsecutiveFailures.WithLabelValues(cbc.name).Set(float64(counts.ConsecutiveFailures))
		}
		return nil, err
	}

	metrics.CircuitBreakerRequests.WithLabelValues(cbc.name, "success").Inc()
	metrics.CircuitBreakerConsecutiveFailures.WithLabelValues(cbc.name).Set(0)

	return result, nil
}

// Ping tests connectivity to the Jellyfin server with circuit breaker protection
func (cbc *JellyfinCircuitBreakerClient) Ping(ctx context.Context) error {
	_, err := cbc.execute(func() (interface{}, error) {
		return nil, cbc.client.Ping(ctx)
	})
	return err
}

// GetSessions retrieves all active sessions with circuit breaker protection
func (cbc *JellyfinCircuitBreakerClient) GetSessions(ctx context.Context) ([]models.JellyfinSession, error) {
	result, err := cbc.execute(func() (interface{}, error) {
		return cbc.client.GetSessions(ctx)
	})
	if err != nil {
		return nil, err
	}
	sessions, ok := result.([]models.JellyfinSession)
	if !ok {
		return nil, errors.New("circuit breaker: unexpected result type for GetSessions")
	}
	return sessions, nil
}

// GetActiveSessions retrieves only sessions with active playback with circuit breaker protection
func (cbc *JellyfinCircuitBreakerClient) GetActiveSessions(ctx context.Context) ([]models.JellyfinSession, error) {
	result, err := cbc.execute(func() (interface{}, error) {
		return cbc.client.GetActiveSessions(ctx)
	})
	if err != nil {
		return nil, err
	}
	sessions, ok := result.([]models.JellyfinSession)
	if !ok {
		return nil, errors.New("circuit breaker: unexpected result type for GetActiveSessions")
	}
	return sessions, nil
}

// GetSystemInfo retrieves Jellyfin server system information with circuit breaker protection
func (cbc *JellyfinCircuitBreakerClient) GetSystemInfo(ctx context.Context) (*JellyfinSystemInfo, error) {
	result, err := cbc.execute(func() (interface{}, error) {
		return cbc.client.GetSystemInfo(ctx)
	})
	if err != nil {
		return nil, err
	}
	info, ok := result.(*JellyfinSystemInfo)
	if !ok {
		return nil, errors.New("circuit breaker: unexpected result type for GetSystemInfo")
	}
	return info, nil
}

// GetUsers retrieves all users with circuit breaker protection
func (cbc *JellyfinCircuitBreakerClient) GetUsers(ctx context.Context) ([]JellyfinUser, error) {
	result, err := cbc.execute(func() (interface{}, error) {
		return cbc.client.GetUsers(ctx)
	})
	if err != nil {
		return nil, err
	}
	users, ok := result.([]JellyfinUser)
	if !ok {
		return nil, errors.New("circuit breaker: unexpected result type for GetUsers")
	}
	return users, nil
}

// StopSession stops/terminates a playback session with circuit breaker protection
func (cbc *JellyfinCircuitBreakerClient) StopSession(ctx context.Context, sessionID string) error {
	_, err := cbc.execute(func() (interface{}, error) {
		return nil, cbc.client.StopSession(ctx, sessionID)
	})
	return err
}

// GetWebSocketURL returns the WebSocket URL for real-time notifications
// This is a passthrough method as it doesn't make network calls
func (cbc *JellyfinCircuitBreakerClient) GetWebSocketURL() (string, error) {
	return cbc.client.GetWebSocketURL()
}

// State returns the current circuit breaker state
func (cbc *JellyfinCircuitBreakerClient) State() gobreaker.State {
	return cbc.cb.State()
}

// Counts returns the current circuit breaker counts
func (cbc *JellyfinCircuitBreakerClient) Counts() gobreaker.Counts {
	return cbc.cb.Counts()
}

// Name returns the circuit breaker name
func (cbc *JellyfinCircuitBreakerClient) Name() string {
	return cbc.name
}
