// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package authz provides Prometheus metrics for the authorization system,
// enabling production observability and alerting.
//
// Metrics Categories:
//   - Authorization Decisions: allow/deny counts, latency histograms
//   - Cache Performance: hit/miss rates, size, evictions
//   - Role Management: assignments, revocations, active roles
//   - Policy Evaluation: policy checks, rule matches
//
// ADR-0015: Zero Trust Authentication & Authorization
//
// Usage:
//
//	// Record an authorization decision
//	RecordAuthzDecision("admin", "/api/v1/users", "read", true, 150*time.Microsecond)
//
//	// Record a cache hit/miss
//	RecordAuthzCacheHit()
//	RecordAuthzCacheMiss()
//
//	// Record role assignment
//	RecordRoleAssignment("admin")
package authz

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Authorization Decision Metrics

	// AuthzDecisionsTotal counts authorization decisions by role, resource, action, and outcome.
	AuthzDecisionsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "authz_decisions_total",
			Help: "Total number of authorization decisions",
		},
		[]string{"role", "resource_pattern", "action", "decision"},
	)

	// AuthzDecisionDuration tracks the latency of authorization decisions.
	AuthzDecisionDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "authz_decision_duration_seconds",
			Help: "Duration of authorization decisions in seconds",
			// Buckets optimized for authz checks (microseconds to milliseconds)
			Buckets: []float64{0.00001, 0.00005, 0.0001, 0.0005, 0.001, 0.005, 0.01, 0.05, 0.1},
		},
		[]string{"role", "cache_hit"},
	)

	// AuthzDeniedTotal specifically tracks denied requests for alerting.
	AuthzDeniedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "authz_denied_total",
			Help: "Total number of authorization denials (for alerting)",
		},
		[]string{"role", "resource_pattern", "action"},
	)

	// Cache Metrics

	// AuthzCacheHitsTotal counts cache hits for authorization decisions.
	AuthzCacheHitsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "authz_cache_hits_total",
			Help: "Total number of authorization cache hits",
		},
	)

	// AuthzCacheMissesTotal counts cache misses for authorization decisions.
	AuthzCacheMissesTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "authz_cache_misses_total",
			Help: "Total number of authorization cache misses",
		},
	)

	// AuthzCacheSize tracks the current size of the authorization cache.
	AuthzCacheSize = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "authz_cache_entries",
			Help: "Current number of entries in the authorization cache",
		},
	)

	// AuthzCacheEvictionsTotal counts cache evictions.
	AuthzCacheEvictionsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "authz_cache_evictions_total",
			Help: "Total number of authorization cache evictions (TTL expiry)",
		},
	)

	// AuthzCacheInvalidationsTotal counts cache invalidations (role changes, policy updates).
	AuthzCacheInvalidationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "authz_cache_invalidations_total",
			Help: "Total number of authorization cache invalidations",
		},
		[]string{"reason"}, // "role_change", "policy_update", "user_invalidation"
	)

	// Role Management Metrics

	// AuthzRoleAssignmentsTotal counts role assignments.
	AuthzRoleAssignmentsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "authz_role_assignments_total",
			Help: "Total number of role assignments",
		},
		[]string{"role", "action"}, // action: "assign", "revoke", "update", "expire"
	)

	// AuthzActiveRoles tracks the current count of active role assignments per role.
	AuthzActiveRoles = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "authz_active_roles",
			Help: "Current number of active role assignments",
		},
		[]string{"role"},
	)

	// AuthzRoleExpirationsTotal counts automatic role expirations.
	AuthzRoleExpirationsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "authz_role_expirations_total",
			Help: "Total number of automatic role expirations",
		},
	)

	// Policy Metrics

	// AuthzPolicyEvaluationsTotal counts policy evaluations by the Casbin enforcer.
	AuthzPolicyEvaluationsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "authz_policy_evaluations_total",
			Help: "Total number of Casbin policy evaluations",
		},
	)

	// AuthzPolicyReloadsTotal counts policy reloads.
	AuthzPolicyReloadsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "authz_policy_reloads_total",
			Help: "Total number of policy reloads",
		},
		[]string{"result"}, // "success", "failure"
	)

	// AuthzPolicyRulesTotal tracks the current number of policy rules.
	AuthzPolicyRulesTotal = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "authz_policy_rules_total",
			Help: "Current number of policy rules loaded",
		},
	)

	// AuthzGroupingRulesTotal tracks the current number of grouping rules (role hierarchy).
	AuthzGroupingRulesTotal = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "authz_grouping_rules_total",
			Help: "Current number of grouping rules (role hierarchy)",
		},
	)

	// Error Metrics

	// AuthzErrorsTotal counts authorization errors (not denials, but actual errors).
	AuthzErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "authz_errors_total",
			Help: "Total number of authorization errors",
		},
		[]string{"error_type"}, // "enforcer_error", "role_lookup_error", "cache_error"
	)

	// Audit Metrics

	// AuthzAuditEventsTotal counts audit events logged.
	AuthzAuditEventsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "authz_audit_events_total",
			Help: "Total number of audit events logged",
		},
		[]string{"decision"}, // "allowed", "denied"
	)

	// AuthzAuditDroppedTotal counts audit events dropped due to buffer overflow.
	AuthzAuditDroppedTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "authz_audit_dropped_total",
			Help: "Total number of audit events dropped (buffer overflow)",
		},
	)

	// AuthzAuditBufferUsage tracks current audit buffer usage.
	AuthzAuditBufferUsage = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "authz_audit_buffer_usage",
			Help: "Current audit buffer usage (percentage)",
		},
	)
)

// RecordAuthzDecision records an authorization decision metric.
// Parameters:
//   - role: The effective role used for the decision (e.g., "admin", "viewer")
//   - resource: The resource path (will be normalized to pattern)
//   - action: The action (e.g., "read", "write", "delete")
//   - allowed: Whether the request was allowed
//   - duration: How long the authorization check took
//   - cacheHit: Whether the decision came from cache
func RecordAuthzDecision(role, resource, action string, allowed bool, duration time.Duration, cacheHit bool) {
	decision := "denied"
	if allowed {
		decision = "allowed"
	}

	// Normalize resource to a pattern for cardinality control
	resourcePattern := normalizeResourcePattern(resource)

	// Record decision count
	AuthzDecisionsTotal.WithLabelValues(role, resourcePattern, action, decision).Inc()

	// Record duration
	cacheHitLabel := "false"
	if cacheHit {
		cacheHitLabel = "true"
	}
	AuthzDecisionDuration.WithLabelValues(role, cacheHitLabel).Observe(duration.Seconds())

	// Record denials separately for alerting
	if !allowed {
		AuthzDeniedTotal.WithLabelValues(role, resourcePattern, action).Inc()
	}

	// Record cache hit/miss
	if cacheHit {
		AuthzCacheHitsTotal.Inc()
	} else {
		AuthzCacheMissesTotal.Inc()
	}

	// Increment policy evaluations (only on cache miss)
	if !cacheHit {
		AuthzPolicyEvaluationsTotal.Inc()
	}
}

// normalizeResourcePattern converts specific resource paths to patterns
// to prevent high cardinality in metrics.
// Examples:
//
//	/api/v1/users/123 -> /api/v1/users/*
//	/api/v1/playbacks/456/details -> /api/v1/playbacks/*/details
func normalizeResourcePattern(resource string) string {
	// Simple pattern normalization - replace numeric segments with *
	// This is a basic implementation; can be enhanced if needed
	result := make([]byte, 0, len(resource))
	inNumeric := false

	for i := 0; i < len(resource); i++ {
		c := resource[i]
		if c >= '0' && c <= '9' {
			if !inNumeric {
				result = append(result, '*')
				inNumeric = true
			}
			// Skip additional digits
		} else {
			inNumeric = false
			result = append(result, c)
		}
	}

	return string(result)
}

// RecordAuthzCacheHit records a cache hit.
func RecordAuthzCacheHit() {
	AuthzCacheHitsTotal.Inc()
}

// RecordAuthzCacheMiss records a cache miss.
func RecordAuthzCacheMiss() {
	AuthzCacheMissesTotal.Inc()
}

// RecordAuthzCacheEviction records a cache eviction.
func RecordAuthzCacheEviction() {
	AuthzCacheEvictionsTotal.Inc()
}

// RecordAuthzCacheInvalidation records a cache invalidation with reason.
func RecordAuthzCacheInvalidation(reason string) {
	AuthzCacheInvalidationsTotal.WithLabelValues(reason).Inc()
}

// UpdateAuthzCacheSize updates the current cache size gauge.
func UpdateAuthzCacheSize(size int) {
	AuthzCacheSize.Set(float64(size))
}

// RecordRoleAssignment records a role assignment event.
func RecordRoleAssignment(role, action string) {
	AuthzRoleAssignmentsTotal.WithLabelValues(role, action).Inc()
}

// RecordRoleExpiration records an automatic role expiration.
func RecordRoleExpiration() {
	AuthzRoleExpirationsTotal.Inc()
}

// UpdateActiveRoles updates the count of active roles per role type.
func UpdateActiveRoles(roleCounts map[string]int) {
	for role, count := range roleCounts {
		AuthzActiveRoles.WithLabelValues(role).Set(float64(count))
	}
}

// RecordPolicyReload records a policy reload event.
func RecordPolicyReload(success bool) {
	result := "success"
	if !success {
		result = "failure"
	}
	AuthzPolicyReloadsTotal.WithLabelValues(result).Inc()
}

// UpdatePolicyStats updates policy-related gauges.
func UpdatePolicyStats(policyRules, groupingRules int) {
	AuthzPolicyRulesTotal.Set(float64(policyRules))
	AuthzGroupingRulesTotal.Set(float64(groupingRules))
}

// RecordAuthzError records an authorization error.
func RecordAuthzError(errorType string) {
	AuthzErrorsTotal.WithLabelValues(errorType).Inc()
}

// RecordAuditEvent records an audit event being logged.
func RecordAuditEvent(allowed bool) {
	decision := "denied"
	if allowed {
		decision = "allowed"
	}
	AuthzAuditEventsTotal.WithLabelValues(decision).Inc()
}

// RecordAuditDropped records an audit event being dropped.
func RecordAuditDropped() {
	AuthzAuditDroppedTotal.Inc()
}

// UpdateAuditBufferUsage updates the audit buffer usage gauge.
func UpdateAuditBufferUsage(usedPercent float64) {
	AuthzAuditBufferUsage.Set(usedPercent)
}
