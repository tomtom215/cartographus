// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package auth provides authentication functionality including OIDC support.
// ADR-0015: Zero Trust Authentication & Authorization
package auth

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

// TestOIDCMetrics_RecordOIDCLogin tests login metric recording.
func TestOIDCMetrics_RecordOIDCLogin(t *testing.T) {
	// Note: Prometheus metrics are global, so we test that calls don't panic
	// and verify basic counter behavior

	t.Run("record successful login", func(t *testing.T) {
		beforeSuccess := testutil.ToFloat64(OIDCLoginAttempts.WithLabelValues("oidc", "success"))

		RecordOIDCLogin("oidc", "success", 100*time.Millisecond)

		afterSuccess := testutil.ToFloat64(OIDCLoginAttempts.WithLabelValues("oidc", "success"))

		if afterSuccess <= beforeSuccess {
			t.Error("Expected success counter to increment")
		}
	})

	t.Run("record failed login", func(t *testing.T) {
		beforeFailure := testutil.ToFloat64(OIDCLoginAttempts.WithLabelValues("oidc", "failure"))

		RecordOIDCLogin("oidc", "failure", 50*time.Millisecond)

		afterFailure := testutil.ToFloat64(OIDCLoginAttempts.WithLabelValues("oidc", "failure"))

		if afterFailure <= beforeFailure {
			t.Error("Expected failure counter to increment")
		}
	})
}

// TestOIDCMetrics_RecordOIDCLogout tests logout metric recording.
func TestOIDCMetrics_RecordOIDCLogout(t *testing.T) {
	t.Run("record rp_initiated logout", func(t *testing.T) {
		before := testutil.ToFloat64(OIDCLogoutTotal.WithLabelValues("rp_initiated", "success"))

		RecordOIDCLogout("rp_initiated", "success")

		after := testutil.ToFloat64(OIDCLogoutTotal.WithLabelValues("rp_initiated", "success"))

		if after <= before {
			t.Error("Expected logout counter to increment")
		}
	})

	t.Run("record back_channel logout", func(t *testing.T) {
		before := testutil.ToFloat64(OIDCLogoutTotal.WithLabelValues("back_channel", "success"))

		RecordOIDCLogout("back_channel", "success")

		after := testutil.ToFloat64(OIDCLogoutTotal.WithLabelValues("back_channel", "success"))

		if after <= before {
			t.Error("Expected logout counter to increment")
		}
	})
}

// TestOIDCMetrics_RecordOIDCTokenRefresh tests token refresh metric recording.
func TestOIDCMetrics_RecordOIDCTokenRefresh(t *testing.T) {
	t.Run("record successful refresh", func(t *testing.T) {
		before := testutil.ToFloat64(OIDCTokenRefreshTotal.WithLabelValues("oidc", "success"))

		RecordOIDCTokenRefresh("oidc", "success", 200*time.Millisecond)

		after := testutil.ToFloat64(OIDCTokenRefreshTotal.WithLabelValues("oidc", "success"))

		if after <= before {
			t.Error("Expected refresh counter to increment")
		}
	})

	t.Run("record failed refresh", func(t *testing.T) {
		before := testutil.ToFloat64(OIDCTokenRefreshTotal.WithLabelValues("oidc", "failure"))

		RecordOIDCTokenRefresh("oidc", "failure", 100*time.Millisecond)

		after := testutil.ToFloat64(OIDCTokenRefreshTotal.WithLabelValues("oidc", "failure"))

		if after <= before {
			t.Error("Expected refresh counter to increment")
		}
	})
}

// TestOIDCMetrics_RecordOIDCStateOperation tests state store operation metric recording.
func TestOIDCMetrics_RecordOIDCStateOperation(t *testing.T) {
	operations := []struct {
		operation string
		outcome   string
	}{
		{"store", "success"},
		{"get", "success"},
		{"get", "not_found"},
		{"delete", "success"},
		{"cleanup", "success"},
	}

	for _, op := range operations {
		t.Run(op.operation+"_"+op.outcome, func(t *testing.T) {
			before := testutil.ToFloat64(OIDCStateStoreOperations.WithLabelValues(op.operation, op.outcome))

			RecordOIDCStateOperation(op.operation, op.outcome)

			after := testutil.ToFloat64(OIDCStateStoreOperations.WithLabelValues(op.operation, op.outcome))

			if after <= before {
				t.Errorf("Expected counter to increment for %s/%s", op.operation, op.outcome)
			}
		})
	}
}

// TestOIDCMetrics_UpdateOIDCStateStoreSize tests state store size gauge.
func TestOIDCMetrics_UpdateOIDCStateStoreSize(t *testing.T) {
	UpdateOIDCStateStoreSize(42)

	size := testutil.ToFloat64(OIDCStateStoreSize)
	if size != 42 {
		t.Errorf("Expected state store size to be 42, got: %f", size)
	}

	UpdateOIDCStateStoreSize(0)

	size = testutil.ToFloat64(OIDCStateStoreSize)
	if size != 0 {
		t.Errorf("Expected state store size to be 0, got: %f", size)
	}
}

// TestOIDCMetrics_RecordOIDCBackChannelLogout tests back-channel logout metric recording.
func TestOIDCMetrics_RecordOIDCBackChannelLogout(t *testing.T) {
	outcomes := []string{"success", "validation_failed", "invalid_request", "missing_token"}

	for _, outcome := range outcomes {
		t.Run(outcome, func(t *testing.T) {
			before := testutil.ToFloat64(OIDCBackChannelLogout.WithLabelValues(outcome))

			RecordOIDCBackChannelLogout(outcome)

			after := testutil.ToFloat64(OIDCBackChannelLogout.WithLabelValues(outcome))

			if after <= before {
				t.Errorf("Expected counter to increment for outcome %s", outcome)
			}
		})
	}
}

// TestOIDCMetrics_RecordOIDCValidationError tests validation error metric recording.
func TestOIDCMetrics_RecordOIDCValidationError(t *testing.T) {
	errorTypes := []string{"expired", "invalid_signature", "invalid_issuer", "invalid_audience", "missing_claims"}

	for _, errorType := range errorTypes {
		t.Run(errorType, func(t *testing.T) {
			before := testutil.ToFloat64(OIDCValidationErrors.WithLabelValues(errorType))

			RecordOIDCValidationError(errorType)

			after := testutil.ToFloat64(OIDCValidationErrors.WithLabelValues(errorType))

			if after <= before {
				t.Errorf("Expected counter to increment for error type %s", errorType)
			}
		})
	}
}

// TestOIDCMetrics_SessionMetrics tests session creation/termination metrics.
func TestOIDCMetrics_SessionMetrics(t *testing.T) {
	t.Run("session created", func(t *testing.T) {
		before := testutil.ToFloat64(OIDCSessionsCreated.WithLabelValues("oidc"))

		RecordOIDCSessionCreated("oidc")

		after := testutil.ToFloat64(OIDCSessionsCreated.WithLabelValues("oidc"))

		if after <= before {
			t.Error("Expected sessions created counter to increment")
		}
	})

	t.Run("session terminated", func(t *testing.T) {
		reasons := []string{"logout", "expired", "back_channel", "admin"}

		for _, reason := range reasons {
			before := testutil.ToFloat64(OIDCSessionsTerminated.WithLabelValues(reason))

			RecordOIDCSessionTerminated(reason)

			after := testutil.ToFloat64(OIDCSessionsTerminated.WithLabelValues(reason))

			if after <= before {
				t.Errorf("Expected sessions terminated counter to increment for reason %s", reason)
			}
		}
	})
}

// TestOIDCMetrics_UpdateOIDCActiveSessions tests active session gauge.
func TestOIDCMetrics_UpdateOIDCActiveSessions(t *testing.T) {
	UpdateOIDCActiveSessions(10)

	count := testutil.ToFloat64(OIDCActiveSessions)
	if count != 10 {
		t.Errorf("Expected active sessions to be 10, got: %f", count)
	}

	UpdateOIDCActiveSessions(5)

	count = testutil.ToFloat64(OIDCActiveSessions)
	if count != 5 {
		t.Errorf("Expected active sessions to be 5, got: %f", count)
	}

	UpdateOIDCActiveSessions(0)

	count = testutil.ToFloat64(OIDCActiveSessions)
	if count != 0 {
		t.Errorf("Expected active sessions to be 0, got: %f", count)
	}
}

// TestOIDCMetrics_MetricsRegistered verifies all metrics are registered.
func TestOIDCMetrics_MetricsRegistered(t *testing.T) {
	// This test verifies that metrics are properly registered with Prometheus
	// by checking that they can be collected

	ch := make(chan prometheus.Metric, 100)

	// Collect from counter vecs
	OIDCLoginAttempts.Collect(ch)
	OIDCLogoutTotal.Collect(ch)
	OIDCTokenRefreshTotal.Collect(ch)
	OIDCStateStoreOperations.Collect(ch)
	OIDCBackChannelLogout.Collect(ch)
	OIDCValidationErrors.Collect(ch)
	OIDCSessionsCreated.Collect(ch)
	OIDCSessionsTerminated.Collect(ch)

	// Collect from gauges
	OIDCStateStoreSize.Collect(ch)
	OIDCActiveSessions.Collect(ch)

	// Collect from histogram vecs
	OIDCLoginDuration.Collect(ch)
	OIDCTokenExchangeDuration.Collect(ch)
	OIDCTokenRefreshDuration.Collect(ch)
	OIDCJWKSFetchDuration.Collect(ch)

	close(ch)

	// Drain channel - just verify no panic
	for range ch {
	}
}
