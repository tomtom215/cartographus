// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Dedupe Audit Types (ADR-0022)
 *
 * Types for the deduplication audit and management system.
 */

/** Dedupe reason - why an event was deduplicated */
export type DedupeReason =
  | 'event_id'
  | 'session_key'
  | 'correlation_key'
  | 'cross_source_key'
  | 'db_constraint';

/** Dedupe layer - where deduplication occurred */
export type DedupeLayer =
  | 'bloom_cache'
  | 'nats_dedup'
  | 'db_unique';

/** Dedupe status - resolution status of the audit entry */
export type DedupeStatus =
  | 'auto_dedupe'
  | 'user_confirmed'
  | 'user_restored';

/** A single dedupe audit entry */
export interface DedupeAuditEntry {
  id: string;
  timestamp: string;

  // Discarded event info
  discarded_event_id: string;
  discarded_session_key?: string;
  discarded_correlation_key?: string;
  discarded_source: string;
  discarded_started_at?: string;
  discarded_raw_payload?: string;

  // Matched event info
  matched_event_id?: string;
  matched_session_key?: string;
  matched_correlation_key?: string;
  matched_source?: string;

  // Dedupe details
  dedupe_reason: DedupeReason;
  dedupe_layer: DedupeLayer;
  similarity_score?: number;

  // User info
  user_id: number;
  username?: string;

  // Media info
  media_type?: string;
  title?: string;
  rating_key?: string;

  // Resolution
  status: DedupeStatus;
  resolved_by?: string;
  resolved_at?: string;
  resolution_notes?: string;

  created_at: string;
}

/** Response from listing dedupe audit entries */
export interface DedupeAuditListResponse {
  entries: DedupeAuditEntry[];
  total_count: number;
  limit: number;
  offset: number;
}

/** Dedupe audit statistics */
export interface DedupeAuditStats {
  total_deduped: number;
  pending_review: number;
  user_restored: number;
  user_confirmed: number;
  accuracy_rate: number;
  dedupe_by_reason: Record<string, number>;
  dedupe_by_layer: Record<string, number>;
  dedupe_by_source: Record<string, number>;
  last_24_hours: number;
  last_7_days: number;
  last_30_days: number;
}

/** Request for confirm/restore actions */
export interface DedupeAuditActionRequest {
  resolved_by?: string;
  notes?: string;
}

/** Response from restore action */
export interface DedupeAuditRestoreResponse {
  success: boolean;
  message: string;
  restored_event_id?: string;
  original_entry?: DedupeAuditEntry;
}

/** Filter options for listing dedupe entries */
export interface DedupeAuditFilter {
  user_id?: number;
  source?: string;
  status?: DedupeStatus;
  reason?: DedupeReason;
  layer?: DedupeLayer;
  from?: string;
  to?: string;
  limit?: number;
  offset?: number;
}
