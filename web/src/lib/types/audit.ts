// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Audit Log Types
 *
 * Types for the security audit logging system.
 * Matches the Go types in internal/audit/types.go
 */

/** Event types for audit logging */
export type AuditEventType =
  | 'auth.success'
  | 'auth.failure'
  | 'auth.lockout'
  | 'auth.unlock'
  | 'auth.logout'
  | 'auth.logout_all'
  | 'auth.session_created'
  | 'auth.session_expired'
  | 'auth.token_revoked'
  | 'authz.granted'
  | 'authz.denied'
  | 'detection.alert'
  | 'detection.acknowledged'
  | 'detection.rule_changed'
  | 'user.created'
  | 'user.modified'
  | 'user.deleted'
  | 'user.role_assigned'
  | 'user.role_revoked'
  | 'config.changed'
  | 'data.export'
  | 'data.import'
  | 'data.backup'
  | 'admin.action';

/** Severity levels for audit events */
export type AuditSeverity = 'debug' | 'info' | 'warning' | 'error' | 'critical';

/** Outcome of an audit event */
export type AuditOutcome = 'success' | 'failure' | 'unknown';

/** Geographic location information */
export interface GeoLocation {
  city?: string;
  region?: string;
  country?: string;
  latitude?: number;
  longitude?: number;
}

/** Source of a request */
export interface AuditSource {
  ip_address: string;
  user_agent?: string;
  hostname?: string;
  port?: number;
  geo?: GeoLocation;
}

/** Actor who performed an action */
export interface AuditActor {
  id: string;
  type: string;
  name?: string;
  roles?: string[];
  session_id?: string;
  auth_method?: string;
}

/** Target of an action */
export interface AuditTarget {
  id: string;
  type: string;
  name?: string;
}

/** A single audit event */
export interface AuditEvent {
  id: string;
  timestamp: string;
  type: AuditEventType;
  severity: AuditSeverity;
  outcome: AuditOutcome;
  actor: AuditActor;
  target?: AuditTarget;
  source: AuditSource;
  action: string;
  description: string;
  metadata?: Record<string, unknown>;
  correlation_id?: string;
  request_id?: string;
}

/** Response from list audit events endpoint */
export interface AuditEventsResponse {
  events: AuditEvent[];
  total: number;
  limit: number;
  offset: number;
}

/** Statistics about audit events */
export interface AuditStats {
  total_events: number;
  events_by_type: Record<string, number>;
  events_by_severity: Record<string, number>;
  events_by_outcome: Record<string, number>;
  oldest_event?: string;
  newest_event?: string;
}

/** Filter options for querying audit events */
export interface AuditEventFilter {
  limit?: number;
  offset?: number;
  types?: AuditEventType[];
  severities?: AuditSeverity[];
  outcomes?: AuditOutcome[];
  actor_id?: string;
  actor_type?: string;
  target_id?: string;
  target_type?: string;
  source_ip?: string;
  start_time?: string;
  end_time?: string;
  search?: string;
  correlation_id?: string;
  request_id?: string;
  order_by?: string;
  order_direction?: 'asc' | 'desc';
}

/** Available audit event types response */
export interface AuditTypesResponse {
  types: string[];
}

/** Available severity levels response */
export interface AuditSeveritiesResponse {
  severities: string[];
}
