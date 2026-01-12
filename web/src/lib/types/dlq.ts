// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * DLQ (Dead Letter Queue) Types
 *
 * Types for the Dead Letter Queue management system.
 * Matches the Go types in internal/api/handlers_dlq.go
 */

/** Error categories for DLQ entries */
export type DLQErrorCategory =
  | 'unknown'
  | 'connection'
  | 'timeout'
  | 'validation'
  | 'database'
  | 'capacity';

/** Status of a DLQ entry */
export type DLQEntryStatus = 'pending' | 'retrying' | 'permanent';

/** A single DLQ entry */
export interface DLQEntry {
  event_id: string;
  message_id: string;
  source: string;
  username?: string;
  media_title?: string;
  original_error: string;
  last_error: string;
  retry_count: number;
  max_retries: number;
  first_failure: string;
  last_failure: string;
  next_retry: string;
  category: DLQErrorCategory;
  status: DLQEntryStatus;
}

/** Response from list DLQ entries endpoint */
export interface DLQEntriesResponse {
  entries: DLQEntry[];
  total: number;
  limit: number;
  offset: number;
}

/** DLQ statistics */
export interface DLQStats {
  total_entries: number;
  total_added: number;
  total_removed: number;
  total_retries: number;
  total_expired: number;
  oldest_entry_age_seconds?: number;
  newest_entry_age_seconds?: number;
  entries_by_category: Record<DLQErrorCategory, number>;
  entries_by_status: Record<DLQEntryStatus, number>;
}

/** Response from retry operations */
export interface DLQRetryResponse {
  success: boolean;
  message: string;
  retried_count?: number;
}

/** Response from cleanup operation */
export interface DLQCleanupResponse {
  cleaned_count: number;
  message: string;
}

/** Filter options for querying DLQ entries */
export interface DLQEntryFilter {
  limit?: number;
  offset?: number;
  category?: DLQErrorCategory;
  status?: DLQEntryStatus;
}

/** Available error categories response */
export interface DLQCategoriesResponse {
  categories: DLQErrorCategory[];
}
