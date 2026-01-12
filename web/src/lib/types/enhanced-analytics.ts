// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Enhanced Analytics Types
 *
 * Types for advanced analytics features:
 * - Cohort Retention Analysis
 * - Quality of Experience (QoE) Dashboard
 * - Data Quality Monitoring
 * - User Network Graph
 */

// =============================================================================
// Cohort Retention Analysis
// =============================================================================

/**
 * Complete cohort retention analysis response
 */
export interface CohortRetentionAnalytics {
	cohorts: CohortData[];
	summary: CohortRetentionSummary;
	retention_curve: RetentionPoint[];
	metadata: CohortQueryMetadata;
}

/**
 * A single cohort (users who started in a specific period)
 */
export interface CohortData {
	cohort_week: string;
	cohort_start_date: string;
	initial_users: number;
	retention: WeekRetention[];
	average_retention: number;
	churn_rate: number;
}

/**
 * Retention data for a specific week offset
 */
export interface WeekRetention {
	week_offset: number;
	active_users: number;
	retention_rate: number;
	week_date: string;
}

/**
 * Aggregate retention statistics across all cohorts
 */
export interface CohortRetentionSummary {
	total_cohorts: number;
	total_users_tracked: number;
	week1_retention: number;
	week4_retention: number;
	week12_retention: number;
	median_retention_week1: number;
	best_performing_cohort: string;
	worst_performing_cohort: string;
	overall_average_retention: number;
	retention_trend: 'improving' | 'declining' | 'stable';
}

/**
 * Single point on the aggregate retention curve
 */
export interface RetentionPoint {
	week_offset: number;
	average_retention: number;
	median_retention: number;
	min_retention: number;
	max_retention: number;
	cohorts_with_data: number;
}

/**
 * Cohort query metadata for provenance
 */
export interface CohortQueryMetadata {
	query_hash: string;
	data_range_start: string;
	data_range_end: string;
	cohort_granularity: 'week' | 'month';
	max_weeks_tracked: number;
	event_count: number;
	generated_at: string;
	query_time_ms: number;
	cached: boolean;
}

// =============================================================================
// Quality of Experience (QoE) Dashboard
// =============================================================================

/**
 * Complete QoE dashboard response
 */
export interface QoEDashboard {
	summary: QoESummary;
	trends: QoETrendPoint[];
	by_platform: QoEByPlatform[];
	by_transcode_decision: QoEByTranscode[];
	top_issues: QoEIssue[];
	metadata: QoEQueryMetadata;
}

/**
 * Aggregate QoE health metrics
 */
export interface QoESummary {
	total_sessions: number;
	ebvs_rate: number;
	ebvs_count: number;
	quality_degrade_rate: number;
	quality_degrade_count: number;
	transcode_rate: number;
	transcode_count: number;
	direct_play_rate: number;
	direct_play_count: number;
	avg_completion: number;
	high_completion_rate: number;
	pause_rate: number;
	avg_pause_count: number;
	relayed_rate: number;
	secure_connection_rate: number;
	avg_bitrate_mbps: number;
	bitrate_p50_mbps: number;
	bitrate_p95_mbps: number;
	qoe_score: number;
	qoe_grade: 'A' | 'B' | 'C' | 'D' | 'F';
}

/**
 * QoE metrics at a specific point in time
 */
export interface QoETrendPoint {
	timestamp: string;
	session_count: number;
	ebvs_rate: number;
	quality_degrade_rate: number;
	transcode_rate: number;
	avg_completion: number;
	avg_bitrate_mbps: number;
	qoe_score: number;
}

/**
 * QoE breakdown by platform/device
 */
export interface QoEByPlatform {
	platform: string;
	session_count: number;
	session_percentage: number;
	ebvs_rate: number;
	quality_degrade_rate: number;
	transcode_rate: number;
	direct_play_rate: number;
	avg_completion: number;
	avg_bitrate_mbps: number;
	qoe_score: number;
	qoe_grade: 'A' | 'B' | 'C' | 'D' | 'F';
}

/**
 * QoE comparison between transcode decisions
 */
export interface QoEByTranscode {
	transcode_decision: string;
	session_count: number;
	session_percentage: number;
	ebvs_rate: number;
	avg_completion: number;
	avg_bitrate_mbps: number;
	qoe_score: number;
}

/**
 * A specific quality issue with impact assessment
 */
export interface QoEIssue {
	issue_type: 'high_ebvs' | 'quality_degradation' | 'high_transcode' | 'high_pause' | 'low_completion';
	severity: 'critical' | 'warning' | 'info';
	title: string;
	description: string;
	affected_sessions: number;
	impact_percentage: number;
	recommendation: string;
	related_dimension?: string;
}

/**
 * QoE query metadata for provenance
 */
export interface QoEQueryMetadata {
	query_hash: string;
	data_range_start: string;
	data_range_end: string;
	trend_interval: 'hour' | 'day';
	event_count: number;
	generated_at: string;
	query_time_ms: number;
	cached: boolean;
}

// =============================================================================
// Data Quality Monitoring
// =============================================================================

/**
 * Complete data quality report
 */
export interface DataQualityReport {
	summary: DataQualitySummary;
	field_quality: FieldQualityMetric[];
	daily_trends: DailyQualityTrend[];
	issues: DataQualityIssue[];
	source_breakdown: SourceQuality[];
	metadata: DataQualityMetadata;
}

/**
 * Overall data quality health metrics
 */
export interface DataQualitySummary {
	total_events: number;
	overall_score: number;
	grade: 'A' | 'B' | 'C' | 'D' | 'F';
	completeness_score: number;
	validity_score: number;
	consistency_score: number;
	null_field_rate: number;
	invalid_value_rate: number;
	duplicate_rate: number;
	future_date_rate: number;
	orphaned_geo_rate: number;
	issue_count: number;
	critical_issue_count: number;
	trend_direction: 'improving' | 'declining' | 'stable';
}

/**
 * Quality metrics for a specific field
 */
export interface FieldQualityMetric {
	field_name: string;
	category: 'identity' | 'content' | 'quality' | 'network' | 'temporal';
	total_records: number;
	null_count: number;
	null_rate: number;
	invalid_count: number;
	invalid_rate: number;
	unique_count: number;
	cardinality: number;
	quality_score: number;
	is_required: boolean;
	status: 'healthy' | 'warning' | 'critical';
}

/**
 * Data quality metrics for a single day
 */
export interface DailyQualityTrend {
	date: string;
	event_count: number;
	overall_score: number;
	null_rate: number;
	invalid_rate: number;
	new_issues: number;
}

/**
 * A specific data quality problem
 */
export interface DataQualityIssue {
	id: string;
	type: 'null_required' | 'invalid_value' | 'future_date' | 'duplicate' | 'orphaned_geo' | 'inconsistent' | 'outlier' | 'missing_relation';
	severity: 'critical' | 'warning' | 'info';
	field?: string;
	title: string;
	description: string;
	affected_records: number;
	impact_percentage: number;
	first_detected: string;
	last_seen: string;
	example_values?: string[];
	recommendation: string;
	auto_resolvable: boolean;
}

/**
 * Data quality breakdown by source
 */
export interface SourceQuality {
	source: 'plex' | 'jellyfin' | 'emby' | 'tautulli';
	server_id?: string;
	event_count: number;
	event_percentage: number;
	quality_score: number;
	null_rate: number;
	invalid_rate: number;
	status: 'healthy' | 'warning' | 'critical';
	top_issue?: string;
}

/**
 * Data quality metadata for provenance
 */
export interface DataQualityMetadata {
	query_hash: string;
	data_range_start: string;
	data_range_end: string;
	analyzed_tables: string[];
	rules_applied: string[];
	generated_at: string;
	query_time_ms: number;
	cached: boolean;
}

// =============================================================================
// User Network Graph
// =============================================================================

/**
 * Complete user relationship network
 */
export interface UserNetworkGraph {
	nodes: UserNode[];
	edges: UserEdge[];
	clusters: UserCluster[];
	summary: NetworkSummary;
	metadata: NetworkMetadata;
}

/**
 * A user in the network graph
 */
export interface UserNode {
	id: string;
	username: string;
	playback_count: number;
	connection_count: number;
	cluster_id: number;
	total_watch_hours: number;
	top_genre?: string;
	last_active: string;
	is_central: boolean;
	node_size: number;
	node_color: string;
}

/**
 * A relationship between two users
 */
export interface UserEdge {
	source: string;
	target: string;
	connection_type: 'shared_session' | 'content_overlap' | 'watch_party' | 'sequential';
	weight: number;
	shared_sessions?: number;
	content_overlap?: number;
	top_shared_content?: string[];
	first_interaction?: string;
	last_interaction?: string;
	edge_width: number;
}

/**
 * A detected community of users
 */
export interface UserCluster {
	id: number;
	name: string;
	user_count: number;
	user_ids: string[];
	density: number;
	top_content?: string[];
	top_genres?: string[];
	avg_watch_hours: number;
	cluster_color: string;
	characteristic_type: string;
}

/**
 * Aggregate statistics about the network
 */
export interface NetworkSummary {
	total_users: number;
	total_connections: number;
	total_clusters: number;
	network_density: number;
	avg_connections_per_user: number;
	max_connections_user: string;
	max_connections_count: number;
	isolated_users: number;
	shared_session_count: number;
	watch_party_count: number;
	largest_cluster_size: number;
	network_type: 'fragmented' | 'centralized' | 'distributed' | 'hierarchical';
}

/**
 * Network metadata for provenance
 */
export interface NetworkMetadata {
	query_hash: string;
	data_range_start: string;
	data_range_end: string;
	min_shared_sessions: number;
	min_content_overlap: number;
	clustering_algorithm: string;
	event_count: number;
	generated_at: string;
	query_time_ms: number;
	cached: boolean;
}

// =============================================================================
// Device Migration Tracking
// =============================================================================

/**
 * Complete device migration analytics response
 */
export interface DeviceMigrationAnalytics {
	summary: DeviceMigrationSummary;
	top_user_profiles: UserDeviceProfile[];
	recent_migrations: DeviceMigration[];
	adoption_trends: PlatformAdoptionTrend[];
	common_transitions: PlatformTransition[];
	platform_distribution: PlatformDistribution[];
	metadata: DeviceMigrationMetadata;
}

/**
 * Aggregate device migration statistics
 */
export interface DeviceMigrationSummary {
	total_users: number;
	multi_device_users: number;
	multi_device_percentage: number;
	total_migrations: number;
	avg_platforms_per_user: number;
	most_common_primary_platform: string;
	fastest_growing_platform: string;
	declining_platforms?: string[];
}

/**
 * A single device platform migration event
 */
export interface DeviceMigration {
	user_id: number;
	username: string;
	from_platform: string;
	to_platform: string;
	migration_date: string;
	sessions_before_migration: number;
	sessions_after_migration: number;
	is_permanent_switch: boolean;
}

/**
 * User's complete device usage profile
 */
export interface UserDeviceProfile {
	user_id: number;
	username: string;
	total_platforms_used: number;
	primary_platform: string;
	primary_platform_percentage: number;
	first_seen_at: string;
	last_seen_at: string;
	days_since_first_seen: number;
	days_since_last_seen: number;
	total_sessions: number;
	total_migrations: number;
	platform_history: UserPlatformUsage[];
	is_multi_device: boolean;
}

/**
 * User's usage of a specific platform
 */
export interface UserPlatformUsage {
	platform: string;
	first_used: string;
	last_used: string;
	session_count: number;
	total_watch_time_minutes: number;
	percentage: number;
	is_active: boolean;
	is_primary: boolean;
}

/**
 * Platform adoption trend over time
 */
export interface PlatformAdoptionTrend {
	date: string;
	platform: string;
	new_users: number;
	active_users: number;
	returning_users: number;
	session_count: number;
	market_share: number;
}

/**
 * Common platform transition path
 */
export interface PlatformTransition {
	from_platform: string;
	to_platform: string;
	transition_count: number;
	unique_users: number;
	avg_days_before_switch: number;
	return_rate: number;
}

/**
 * Platform usage distribution
 */
export interface PlatformDistribution {
	platform: string;
	playback_count: number;
	unique_users: number;
}

/**
 * Device migration query metadata
 */
export interface DeviceMigrationMetadata {
	query_hash: string;
	execution_time_ms: number;
	data_range_start: string;
	data_range_end: string;
	total_events_analyzed: number;
	unique_platforms_found: number;
	migration_window_days: number;
}

// =============================================================================
// Content Discovery Analytics
// =============================================================================

/**
 * Complete content discovery analytics response
 */
export interface ContentDiscoveryAnalytics {
	summary: ContentDiscoverySummary;
	time_buckets: DiscoveryTimeBucket[];
	early_adopters: EarlyAdopter[];
	recently_discovered: ContentDiscoveryItem[];
	stale_content: StaleContent[];
	library_stats: LibraryDiscoveryStats[];
	trends: DiscoveryTrend[];
	metadata: ContentDiscoveryMetadata;
}

/**
 * Aggregate content discovery statistics
 */
export interface ContentDiscoverySummary {
	total_content_with_added_at: number;
	total_discovered: number;
	total_never_watched: number;
	overall_discovery_rate: number;
	avg_time_to_discovery_hours: number;
	median_time_to_discovery_hours: number;
	early_discovery_rate: number;
	fastest_discovery_hours: number;
	slowest_discovery_days: number;
	recent_additions_count: number;
	recent_discovered_count: number;
}

/**
 * Discovery rate by time bucket
 */
export interface DiscoveryTimeBucket {
	bucket: string;
	bucket_min_hours: number;
	bucket_max_hours: number;
	content_count: number;
	percentage: number;
}

/**
 * User who discovers content quickly
 */
export interface EarlyAdopter {
	user_id: number;
	username: string;
	early_discovery_count: number;
	total_discoveries: number;
	early_discovery_rate: number;
	avg_time_to_discovery_hours: number;
	first_seen_at: string;
	favorite_library?: string;
}

/**
 * Discovery metrics for a single content item
 */
export interface ContentDiscoveryItem {
	rating_key: string;
	title: string;
	media_type: string;
	library_name: string;
	added_at: string;
	first_watched_at?: string;
	time_to_first_watch_hours?: number;
	total_playbacks: number;
	unique_viewers: number;
	avg_completion: number;
	discovery_velocity: 'fast' | 'medium' | 'slow' | 'not_discovered';
	year?: number;
	genres?: string;
}

/**
 * Content added but never watched
 */
export interface StaleContent {
	rating_key: string;
	title: string;
	media_type: string;
	library_name: string;
	added_at: string;
	days_since_added: number;
	year?: number;
	genres?: string;
	content_rating?: string;
}

/**
 * Discovery statistics for a library
 */
export interface LibraryDiscoveryStats {
	library_name: string;
	total_items: number;
	watched_items: number;
	unwatched_items: number;
	discovery_rate: number;
	avg_time_to_discovery_hours: number;
	median_time_to_discovery_hours: number;
	early_discovery_rate: number;
}

/**
 * Discovery trend over time
 */
export interface DiscoveryTrend {
	date: string;
	content_added: number;
	content_discovered: number;
	discovery_rate: number;
	avg_time_to_discovery_hours: number;
}

/**
 * Content discovery query metadata
 */
export interface ContentDiscoveryMetadata {
	query_hash: string;
	execution_time_ms: number;
	data_range_start: string;
	data_range_end: string;
	total_events_analyzed: number;
	unique_content_analyzed: number;
	early_discovery_threshold_hours: number;
	stale_content_threshold_days: number;
}
