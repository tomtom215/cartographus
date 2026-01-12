// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package models provides data structures for the Cartographus application.
// This file contains data quality monitoring models for production-grade observability.
//
// Data quality is essential for:
// - Ensuring analytics accuracy and trustworthiness
// - Detecting data pipeline issues early
// - Maintaining auditability and compliance
// - Building user confidence in the system
package models

import "time"

// DataQualityReport represents a comprehensive data quality assessment
type DataQualityReport struct {
	// Summary provides overall data quality health
	Summary DataQualitySummary `json:"summary"`

	// FieldQuality provides per-field completeness and validity metrics
	FieldQuality []FieldQualityMetric `json:"field_quality"`

	// DailyTrends shows data quality trends over time
	DailyTrends []DailyQualityTrend `json:"daily_trends"`

	// Issues lists specific data quality problems detected
	Issues []DataQualityIssue `json:"issues"`

	// SourceBreakdown shows quality by data source
	SourceBreakdown []SourceQuality `json:"source_breakdown"`

	// Metadata provides query provenance
	Metadata DataQualityMetadata `json:"metadata"`
}

// DataQualitySummary provides overall data quality health metrics
type DataQualitySummary struct {
	// TotalEvents is the count of events analyzed
	TotalEvents int64 `json:"total_events"`

	// OverallScore is a composite quality score (0-100)
	// Formula: Weighted average of completeness, validity, and consistency
	OverallScore float64 `json:"overall_score"`

	// Grade is a letter grade (A, B, C, D, F) based on OverallScore
	Grade string `json:"grade"`

	// CompletenessScore measures presence of required fields (0-100)
	CompletenessScore float64 `json:"completeness_score"`

	// ValidityScore measures correctness of field values (0-100)
	ValidityScore float64 `json:"validity_score"`

	// ConsistencyScore measures data consistency across records (0-100)
	ConsistencyScore float64 `json:"consistency_score"`

	// NullFieldRate is the percentage of null/empty critical fields
	NullFieldRate float64 `json:"null_field_rate"`

	// InvalidValueRate is the percentage of fields with invalid values
	InvalidValueRate float64 `json:"invalid_value_rate"`

	// DuplicateRate is the percentage of potential duplicate records
	DuplicateRate float64 `json:"duplicate_rate"`

	// FutureDateRate is the percentage of events with future timestamps
	FutureDateRate float64 `json:"future_date_rate"`

	// OrphanedGeoRate is the percentage of events with missing geolocation
	OrphanedGeoRate float64 `json:"orphaned_geo_rate"`

	// IssueCount is the total number of issues detected
	IssueCount int `json:"issue_count"`

	// CriticalIssueCount is issues with severity "critical"
	CriticalIssueCount int `json:"critical_issue_count"`

	// TrendDirection indicates if quality is "improving", "declining", or "stable"
	TrendDirection string `json:"trend_direction"`
}

// FieldQualityMetric provides quality metrics for a specific field
type FieldQualityMetric struct {
	// FieldName is the database column name
	FieldName string `json:"field_name"`

	// Category groups related fields ("identity", "content", "quality", "network", "temporal")
	Category string `json:"category"`

	// TotalRecords analyzed for this field
	TotalRecords int64 `json:"total_records"`

	// NullCount is records with null/empty values
	NullCount int64 `json:"null_count"`

	// NullRate is NullCount / TotalRecords * 100
	NullRate float64 `json:"null_rate"`

	// InvalidCount is records with invalid values
	InvalidCount int64 `json:"invalid_count"`

	// InvalidRate is InvalidCount / TotalRecords * 100
	InvalidRate float64 `json:"invalid_rate"`

	// UniqueCount is distinct non-null values
	UniqueCount int64 `json:"unique_count"`

	// Cardinality is UniqueCount / NonNullCount (diversity measure)
	Cardinality float64 `json:"cardinality"`

	// QualityScore is overall field quality (0-100)
	QualityScore float64 `json:"quality_score"`

	// IsRequired indicates if this field should never be null
	IsRequired bool `json:"is_required"`

	// Status is "healthy", "warning", or "critical" based on quality
	Status string `json:"status"`
}

// DailyQualityTrend represents data quality metrics for a single day
type DailyQualityTrend struct {
	// Date for this data point
	Date time.Time `json:"date"`

	// EventCount for this day
	EventCount int64 `json:"event_count"`

	// OverallScore for this day
	OverallScore float64 `json:"overall_score"`

	// NullRate for this day
	NullRate float64 `json:"null_rate"`

	// InvalidRate for this day
	InvalidRate float64 `json:"invalid_rate"`

	// NewIssues detected on this day
	NewIssues int `json:"new_issues"`
}

// DataQualityIssue represents a specific data quality problem
type DataQualityIssue struct {
	// ID is a unique identifier for this issue
	ID string `json:"id"`

	// Type categorizes the issue
	// Types: "null_required", "invalid_value", "future_date", "duplicate",
	//        "orphaned_geo", "inconsistent", "outlier", "missing_relation"
	Type string `json:"type"`

	// Severity is "critical", "warning", or "info"
	Severity string `json:"severity"`

	// Field is the affected field name (if applicable)
	Field string `json:"field,omitempty"`

	// Title is a human-readable issue summary
	Title string `json:"title"`

	// Description provides details about the issue
	Description string `json:"description"`

	// AffectedRecords is the count of records with this issue
	AffectedRecords int64 `json:"affected_records"`

	// ImpactPercentage is AffectedRecords / TotalRecords * 100
	ImpactPercentage float64 `json:"impact_percentage"`

	// FirstDetected is when this issue was first seen
	FirstDetected time.Time `json:"first_detected"`

	// LastSeen is when this issue was last observed
	LastSeen time.Time `json:"last_seen"`

	// ExampleValues provides sample problematic values (for debugging)
	ExampleValues []string `json:"example_values,omitempty"`

	// Recommendation provides guidance for resolution
	Recommendation string `json:"recommendation"`

	// AutoResolvable indicates if the system can auto-fix this issue
	AutoResolvable bool `json:"auto_resolvable"`
}

// SourceQuality provides data quality breakdown by source
type SourceQuality struct {
	// Source is the data source ("plex", "jellyfin", "emby", "tautulli")
	Source string `json:"source"`

	// ServerID if applicable
	ServerID string `json:"server_id,omitempty"`

	// EventCount from this source
	EventCount int64 `json:"event_count"`

	// EventPercentage is this source's share of total events
	EventPercentage float64 `json:"event_percentage"`

	// QualityScore for this source
	QualityScore float64 `json:"quality_score"`

	// NullRate for this source
	NullRate float64 `json:"null_rate"`

	// InvalidRate for this source
	InvalidRate float64 `json:"invalid_rate"`

	// Status is "healthy", "warning", or "critical"
	Status string `json:"status"`

	// TopIssue is the most impactful issue for this source
	TopIssue string `json:"top_issue,omitempty"`
}

// DataQualityMetadata provides provenance information
type DataQualityMetadata struct {
	// QueryHash for reproducibility
	QueryHash string `json:"query_hash"`

	// DataRangeStart is the earliest data analyzed
	DataRangeStart time.Time `json:"data_range_start"`

	// DataRangeEnd is the latest data analyzed
	DataRangeEnd time.Time `json:"data_range_end"`

	// AnalyzedTables lists tables that were checked
	AnalyzedTables []string `json:"analyzed_tables"`

	// RulesApplied lists quality rules that were checked
	RulesApplied []string `json:"rules_applied"`

	// GeneratedAt is when this report was generated
	GeneratedAt time.Time `json:"generated_at"`

	// QueryTimeMs is execution time
	QueryTimeMs int64 `json:"query_time_ms"`

	// Cached indicates if from cache
	Cached bool `json:"cached"`
}
