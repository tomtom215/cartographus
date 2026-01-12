// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package auth provides authentication functionality including OIDC support.
// ADR-0015: Zero Trust Authentication & Authorization
package auth

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// OIDC Authentication Metrics
// ADR-0015: Zero Trust Authentication (Zitadel Amendment)
// Production-grade observability for OIDC authentication operations.

var (
	// OIDCLoginAttempts counts OIDC login attempts.
	// Labels:
	//   - provider: IdP identifier (e.g., "keycloak", "auth0", "okta")
	//   - outcome: "success", "failure", "error"
	OIDCLoginAttempts = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "oidc_login_attempts_total",
			Help: "Total number of OIDC login attempts",
		},
		[]string{"provider", "outcome"},
	)

	// OIDCLoginDuration measures the duration of OIDC login flows.
	// This includes the time from callback receipt to session creation.
	OIDCLoginDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "oidc_login_duration_seconds",
			Help: "Duration of OIDC login operations in seconds",
			// Optimized for auth latency: 10ms to 10s
			Buckets: []float64{0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		},
		[]string{"provider"},
	)

	// OIDCTokenExchangeDuration measures the token exchange latency.
	OIDCTokenExchangeDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "oidc_token_exchange_duration_seconds",
			Help:    "Duration of OIDC token exchange operations",
			Buckets: []float64{0.05, 0.1, 0.25, 0.5, 1, 2, 5},
		},
		[]string{"provider"},
	)

	// OIDCLogoutTotal counts OIDC logout operations.
	// Labels:
	//   - type: "rp_initiated" (user-initiated), "back_channel" (IdP-initiated)
	//   - outcome: "success", "failure"
	OIDCLogoutTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "oidc_logout_total",
			Help: "Total number of OIDC logout operations",
		},
		[]string{"type", "outcome"},
	)

	// OIDCTokenRefreshTotal counts token refresh attempts.
	OIDCTokenRefreshTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "oidc_token_refresh_total",
			Help: "Total number of OIDC token refresh attempts",
		},
		[]string{"provider", "outcome"},
	)

	// OIDCTokenRefreshDuration measures token refresh latency.
	OIDCTokenRefreshDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "oidc_token_refresh_duration_seconds",
			Help:    "Duration of OIDC token refresh operations",
			Buckets: []float64{0.05, 0.1, 0.25, 0.5, 1, 2, 5},
		},
		[]string{"provider"},
	)

	// OIDCStateStoreOperations counts state store operations.
	// Labels:
	//   - operation: "store", "get", "delete", "cleanup"
	//   - outcome: "success", "failure", "not_found", "expired"
	OIDCStateStoreOperations = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "oidc_state_store_operations_total",
			Help: "Total number of OIDC state store operations",
		},
		[]string{"operation", "outcome"},
	)

	// OIDCStateStoreSize tracks the current number of active states.
	OIDCStateStoreSize = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "oidc_state_store_size",
			Help: "Current number of active OIDC states in the store",
		},
	)

	// OIDCJWKSFetchDuration measures JWKS fetch latency.
	OIDCJWKSFetchDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "oidc_jwks_fetch_duration_seconds",
			Help:    "Duration of JWKS fetch operations",
			Buckets: []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1, 2},
		},
		[]string{"provider"},
	)

	// OIDCJWKSCacheHits counts JWKS cache hits.
	OIDCJWKSCacheHits = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "oidc_jwks_cache_hits_total",
			Help: "Total number of JWKS cache hits",
		},
	)

	// OIDCJWKSCacheMisses counts JWKS cache misses (requires fetch).
	OIDCJWKSCacheMisses = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "oidc_jwks_cache_misses_total",
			Help: "Total number of JWKS cache misses",
		},
	)

	// OIDCSessionsCreated counts sessions created via OIDC.
	OIDCSessionsCreated = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "oidc_sessions_created_total",
			Help: "Total number of sessions created via OIDC authentication",
		},
		[]string{"provider"},
	)

	// OIDCSessionsTerminated counts sessions terminated.
	// Labels:
	//   - reason: "logout", "expired", "back_channel", "admin"
	OIDCSessionsTerminated = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "oidc_sessions_terminated_total",
			Help: "Total number of OIDC sessions terminated",
		},
		[]string{"reason"},
	)

	// OIDCBackChannelLogout counts back-channel logout operations.
	OIDCBackChannelLogout = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "oidc_backchannel_logout_total",
			Help: "Total number of OIDC back-channel logout requests",
		},
		[]string{"outcome"}, // "success", "invalid_token", "validation_failed"
	)

	// OIDCValidationErrors counts token validation errors by type.
	OIDCValidationErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "oidc_validation_errors_total",
			Help: "Total number of OIDC token validation errors",
		},
		[]string{"error_type"}, // "expired", "invalid_signature", "invalid_issuer", "invalid_audience", "missing_claims"
	)

	// OIDCActiveSessions tracks currently active OIDC sessions.
	OIDCActiveSessions = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "oidc_active_sessions",
			Help: "Current number of active OIDC sessions",
		},
	)
)

// RecordOIDCLogin records a login attempt and its outcome.
func RecordOIDCLogin(provider, outcome string, duration time.Duration) {
	OIDCLoginAttempts.WithLabelValues(provider, outcome).Inc()
	if outcome == "success" {
		OIDCLoginDuration.WithLabelValues(provider).Observe(duration.Seconds())
	}
}

// RecordOIDCTokenExchange records a token exchange operation.
func RecordOIDCTokenExchange(provider string, duration time.Duration) {
	OIDCTokenExchangeDuration.WithLabelValues(provider).Observe(duration.Seconds())
}

// RecordOIDCLogout records a logout operation.
func RecordOIDCLogout(logoutType, outcome string) {
	OIDCLogoutTotal.WithLabelValues(logoutType, outcome).Inc()
}

// RecordOIDCTokenRefresh records a token refresh operation.
func RecordOIDCTokenRefresh(provider, outcome string, duration time.Duration) {
	OIDCTokenRefreshTotal.WithLabelValues(provider, outcome).Inc()
	if outcome == "success" {
		OIDCTokenRefreshDuration.WithLabelValues(provider).Observe(duration.Seconds())
	}
}

// RecordOIDCStateOperation records a state store operation.
func RecordOIDCStateOperation(operation, outcome string) {
	OIDCStateStoreOperations.WithLabelValues(operation, outcome).Inc()
}

// UpdateOIDCStateStoreSize updates the state store size gauge.
func UpdateOIDCStateStoreSize(size int) {
	OIDCStateStoreSize.Set(float64(size))
}

// RecordOIDCJWKSFetch records a JWKS fetch operation.
func RecordOIDCJWKSFetch(provider string, duration time.Duration, cacheHit bool) {
	OIDCJWKSFetchDuration.WithLabelValues(provider).Observe(duration.Seconds())
	if cacheHit {
		OIDCJWKSCacheHits.Inc()
	} else {
		OIDCJWKSCacheMisses.Inc()
	}
}

// RecordOIDCSessionCreated records a new session creation.
func RecordOIDCSessionCreated(provider string) {
	OIDCSessionsCreated.WithLabelValues(provider).Inc()
	OIDCActiveSessions.Inc()
}

// RecordOIDCSessionTerminated records a session termination.
func RecordOIDCSessionTerminated(reason string) {
	OIDCSessionsTerminated.WithLabelValues(reason).Inc()
	OIDCActiveSessions.Dec()
}

// RecordOIDCBackChannelLogout records a back-channel logout.
func RecordOIDCBackChannelLogout(outcome string) {
	OIDCBackChannelLogout.WithLabelValues(outcome).Inc()
}

// RecordOIDCValidationError records a token validation error.
func RecordOIDCValidationError(errorType string) {
	OIDCValidationErrors.WithLabelValues(errorType).Inc()
}

// UpdateOIDCActiveSessions sets the active session count.
func UpdateOIDCActiveSessions(count int) {
	OIDCActiveSessions.Set(float64(count))
}
