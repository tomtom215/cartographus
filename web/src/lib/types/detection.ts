// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Detection types for security monitoring and anomaly detection
 * ADR-0020: Detection rules engine for media playback security
 */

/** Rule types for detection engine */
export type DetectionRuleType =
  | 'impossible_travel'
  | 'concurrent_streams'
  | 'device_velocity'
  | 'geo_restriction'
  | 'simultaneous_locations'
  | 'user_agent_anomaly'
  | 'vpn_usage';

/** Severity levels for detection alerts */
export type DetectionSeverity = 'critical' | 'warning' | 'info';

/** Detection alert from the backend */
export interface DetectionAlert {
  id: number;
  rule_type: DetectionRuleType;
  user_id: number;
  username: string;
  machine_id?: string;
  ip_address?: string;
  severity: DetectionSeverity;
  title: string;
  message: string;
  metadata?: Record<string, unknown>;
  acknowledged: boolean;
  acknowledged_by?: string;
  acknowledged_at?: string;
  created_at: string;
}

/** Detection rule configuration */
export interface DetectionRule {
  id: number;
  rule_type: DetectionRuleType;
  name: string;
  enabled: boolean;
  config: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

/** User trust score */
export interface UserTrustScore {
  user_id: number;
  username?: string;
  score: number;
  violations_count: number;
  last_violation_at?: string;
  restricted: boolean;
  updated_at: string;
}

/** Alert filter for list queries */
export interface DetectionAlertFilter {
  limit?: number;
  offset?: number;
  user_id?: number;
  acknowledged?: boolean;
  severity?: DetectionSeverity;
  rule_type?: DetectionRuleType;
  start_date?: string;
  end_date?: string;
  order_by?: string;
  order_direction?: 'asc' | 'desc';
}

/** Response from list alerts endpoint */
export interface DetectionAlertsResponse {
  alerts: DetectionAlert[];
  total: number;
  limit: number;
  offset: number;
}

/** Response from list rules endpoint */
export interface DetectionRulesResponse {
  rules: DetectionRule[];
}

/** Response from list low trust users endpoint */
export interface LowTrustUsersResponse {
  users: UserTrustScore[];
  threshold: number;
}

/** Detection engine metrics */
export interface DetectionMetrics {
  events_processed: number;
  alerts_generated: number;
  detection_errors: number;
  processing_time_ms: number;
  last_processed_at: string;
  detector_metrics: Record<DetectionRuleType, DetectorMetrics>;
}

/** Individual detector metrics */
export interface DetectorMetrics {
  events_checked: number;
  alerts_generated: number;
  errors: number;
  avg_processing_ms: number;
  last_triggered_at?: string;
}

/** Alert statistics by category */
export interface DetectionAlertStats {
  by_severity: {
    critical: number;
    warning: number;
    info: number;
  };
  by_rule_type: {
    impossible_travel: number;
    concurrent_streams: number;
    device_velocity: number;
    geo_restriction?: number;
    simultaneous_locations?: number;
    user_agent_anomaly?: number;
    vpn_usage?: number;
  };
  unacknowledged: number;
  total: number;
}

/** Rule update request */
export interface UpdateRuleRequest {
  enabled: boolean;
  config?: Record<string, unknown>;
}

/** Acknowledge alert request */
export interface AcknowledgeAlertRequest {
  acknowledged_by: string;
}

/** Impossible travel configuration */
export interface ImpossibleTravelConfig {
  max_speed_kmh: number;
  min_distance_km: number;
  min_time_delta_minutes: number;
  severity: DetectionSeverity;
}

/** Concurrent streams configuration */
export interface ConcurrentStreamsConfig {
  default_limit: number;
  user_limits?: Record<string, number>;
  severity: DetectionSeverity;
}

/** Device velocity configuration */
export interface DeviceVelocityConfig {
  window_minutes: number;
  max_unique_ips: number;
  severity: DetectionSeverity;
}

/** Geographic restriction configuration */
export interface GeoRestrictionConfig {
  blocked_countries: string[];
  allowed_countries: string[];
  severity: DetectionSeverity;
}

/** Simultaneous locations configuration */
export interface SimultaneousLocationsConfig {
  window_minutes: number;
  min_distance_km: number;
  severity: DetectionSeverity;
}

/** User agent anomaly configuration */
export interface UserAgentAnomalyConfig {
  window_minutes: number;
  alert_on_new_user_agent: boolean;
  alert_on_platform_switch: boolean;
  min_history_for_anomaly: number;
  suspicious_patterns: string[];
  severity: DetectionSeverity;
}

/** VPN usage configuration */
export interface VPNUsageConfig {
  alert_on_first_use: boolean;
  alert_on_new_provider: boolean;
  alert_on_high_risk: boolean;
  excluded_providers: string[];
  excluded_users: number[];
  track_vpn_history: boolean;
  severity: DetectionSeverity;
}
