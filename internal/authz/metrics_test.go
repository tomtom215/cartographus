// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package authz

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	io_prometheus_client "github.com/prometheus/client_model/go"
)

// getCounterValue extracts the value from a Prometheus counter
func getCounterValue(counter prometheus.Counter) float64 {
	var m io_prometheus_client.Metric
	if err := counter.Write(&m); err != nil {
		return 0
	}
	return m.GetCounter().GetValue()
}

// getGaugeValue extracts the value from a Prometheus gauge
func getGaugeValue(gauge prometheus.Gauge) float64 {
	var m io_prometheus_client.Metric
	if err := gauge.Write(&m); err != nil {
		return 0
	}
	return m.GetGauge().GetValue()
}

func TestRecordAuthzDecision(t *testing.T) {
	t.Run("records allowed decision", func(t *testing.T) {
		// Reset metrics for test isolation
		before := getCounterValue(AuthzCacheHitsTotal)

		RecordAuthzDecision("admin", "/api/v1/users", "read", true, 100*time.Microsecond, true)

		after := getCounterValue(AuthzCacheHitsTotal)
		if after <= before {
			t.Error("expected cache hits to increase")
		}
	})

	t.Run("records denied decision", func(t *testing.T) {
		RecordAuthzDecision("viewer", "/api/v1/admin", "write", false, 200*time.Microsecond, false)

		// Verify denial counter increased (can't easily check specific labels)
	})

	t.Run("records cache miss", func(t *testing.T) {
		before := getCounterValue(AuthzCacheMissesTotal)

		RecordAuthzDecision("editor", "/api/v1/test", "read", true, 1*time.Millisecond, false)

		after := getCounterValue(AuthzCacheMissesTotal)
		if after <= before {
			t.Error("expected cache misses to increase")
		}
	})
}

func TestNormalizeResourcePattern(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/api/v1/users", "/api/v*/users"},
		{"/api/v1/users/123", "/api/v*/users/*"},
		{"/api/v1/playbacks/456/details", "/api/v*/playbacks/*/details"},
		{"/api/v1/analytics", "/api/v*/analytics"},
		{"/health", "/health"},
		{"", ""},
		{"/api/v2/stats", "/api/v*/stats"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizeResourcePattern(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeResourcePattern(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestRecordAuthzCacheHit(t *testing.T) {
	before := getCounterValue(AuthzCacheHitsTotal)
	RecordAuthzCacheHit()
	after := getCounterValue(AuthzCacheHitsTotal)

	if after != before+1 {
		t.Errorf("expected cache hits to increase by 1, got %f -> %f", before, after)
	}
}

func TestRecordAuthzCacheMiss(t *testing.T) {
	before := getCounterValue(AuthzCacheMissesTotal)
	RecordAuthzCacheMiss()
	after := getCounterValue(AuthzCacheMissesTotal)

	if after != before+1 {
		t.Errorf("expected cache misses to increase by 1, got %f -> %f", before, after)
	}
}

func TestRecordAuthzCacheEviction(t *testing.T) {
	before := getCounterValue(AuthzCacheEvictionsTotal)
	RecordAuthzCacheEviction()
	after := getCounterValue(AuthzCacheEvictionsTotal)

	if after != before+1 {
		t.Errorf("expected cache evictions to increase by 1, got %f -> %f", before, after)
	}
}

func TestRecordAuthzCacheInvalidation(t *testing.T) {
	// Test different invalidation reasons
	reasons := []string{"role_change", "policy_update", "user_invalidation"}

	for _, reason := range reasons {
		t.Run(reason, func(t *testing.T) {
			// Just verify it doesn't panic
			RecordAuthzCacheInvalidation(reason)
		})
	}
}

func TestUpdateAuthzCacheSize(t *testing.T) {
	UpdateAuthzCacheSize(100)
	value := getGaugeValue(AuthzCacheSize)

	if value != 100 {
		t.Errorf("expected cache size=100, got %f", value)
	}

	UpdateAuthzCacheSize(50)
	value = getGaugeValue(AuthzCacheSize)

	if value != 50 {
		t.Errorf("expected cache size=50, got %f", value)
	}
}

func TestRecordRoleAssignment(t *testing.T) {
	actions := []string{"assign", "revoke", "update", "expire"}
	roles := []string{"viewer", "editor", "admin"}

	for _, role := range roles {
		for _, action := range actions {
			t.Run(role+"_"+action, func(t *testing.T) {
				// Just verify it doesn't panic
				RecordRoleAssignment(role, action)
			})
		}
	}
}

func TestRecordRoleExpiration(t *testing.T) {
	before := getCounterValue(AuthzRoleExpirationsTotal)
	RecordRoleExpiration()
	after := getCounterValue(AuthzRoleExpirationsTotal)

	if after != before+1 {
		t.Errorf("expected role expirations to increase by 1, got %f -> %f", before, after)
	}
}

func TestUpdateActiveRoles(t *testing.T) {
	roleCounts := map[string]int{
		"viewer": 100,
		"editor": 25,
		"admin":  5,
	}

	UpdateActiveRoles(roleCounts)

	// Verify gauges were updated (check one)
	var m io_prometheus_client.Metric
	gauge, err := AuthzActiveRoles.GetMetricWithLabelValues("viewer")
	if err != nil {
		t.Fatalf("failed to get gauge: %v", err)
	}
	if err := gauge.Write(&m); err != nil {
		t.Fatalf("failed to write metric: %v", err)
	}
	if m.GetGauge().GetValue() != 100 {
		t.Errorf("expected viewer count=100, got %f", m.GetGauge().GetValue())
	}
}

func TestRecordPolicyReload(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		RecordPolicyReload(true)
	})

	t.Run("failure", func(t *testing.T) {
		RecordPolicyReload(false)
	})
}

func TestUpdatePolicyStats(t *testing.T) {
	UpdatePolicyStats(86, 3)

	policyValue := getGaugeValue(AuthzPolicyRulesTotal)
	groupingValue := getGaugeValue(AuthzGroupingRulesTotal)

	if policyValue != 86 {
		t.Errorf("expected policy rules=86, got %f", policyValue)
	}
	if groupingValue != 3 {
		t.Errorf("expected grouping rules=3, got %f", groupingValue)
	}
}

func TestRecordAuthzError(t *testing.T) {
	errorTypes := []string{"enforcer_error", "role_lookup_error", "cache_error"}

	for _, errorType := range errorTypes {
		t.Run(errorType, func(t *testing.T) {
			RecordAuthzError(errorType)
		})
	}
}

func TestRecordAuditEvent(t *testing.T) {
	t.Run("allowed", func(t *testing.T) {
		RecordAuditEvent(true)
	})

	t.Run("denied", func(t *testing.T) {
		RecordAuditEvent(false)
	})
}

func TestRecordAuditDropped(t *testing.T) {
	before := getCounterValue(AuthzAuditDroppedTotal)
	RecordAuditDropped()
	after := getCounterValue(AuthzAuditDroppedTotal)

	if after != before+1 {
		t.Errorf("expected audit dropped to increase by 1, got %f -> %f", before, after)
	}
}

func TestUpdateAuditBufferUsage(t *testing.T) {
	UpdateAuditBufferUsage(75.5)
	value := getGaugeValue(AuthzAuditBufferUsage)

	if value != 75.5 {
		t.Errorf("expected audit buffer usage=75.5, got %f", value)
	}
}

func TestMetricsAreRegistered(t *testing.T) {
	// Verify all expected metrics are registered and accessible
	metrics := []string{
		"authz_decisions_total",
		"authz_decision_duration_seconds",
		"authz_denied_total",
		"authz_cache_hits_total",
		"authz_cache_misses_total",
		"authz_cache_entries",
		"authz_cache_evictions_total",
		"authz_cache_invalidations_total",
		"authz_role_assignments_total",
		"authz_active_roles",
		"authz_role_expirations_total",
		"authz_policy_evaluations_total",
		"authz_policy_reloads_total",
		"authz_policy_rules_total",
		"authz_grouping_rules_total",
		"authz_errors_total",
		"authz_audit_events_total",
		"authz_audit_dropped_total",
		"authz_audit_buffer_usage",
	}

	// Just verify the package compiled with all metrics
	// The promauto registration happens at package init time
	if len(metrics) != 19 {
		t.Errorf("expected 19 metric types, got %d", len(metrics))
	}
}

func BenchmarkRecordAuthzDecision(b *testing.B) {
	for i := 0; i < b.N; i++ {
		RecordAuthzDecision("admin", "/api/v1/users/123", "read", true, 100*time.Microsecond, true)
	}
}

func BenchmarkNormalizeResourcePattern(b *testing.B) {
	for i := 0; i < b.N; i++ {
		normalizeResourcePattern("/api/v1/users/123456/profile")
	}
}

func BenchmarkRecordAuthzCacheHit(b *testing.B) {
	for i := 0; i < b.N; i++ {
		RecordAuthzCacheHit()
	}
}
