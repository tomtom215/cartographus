// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * WAL (Write-Ahead Log) Types
 *
 * Types for the WAL statistics and health monitoring.
 * Matches the Go types in internal/api/handlers_wal.go
 */

/** WAL health status */
export type WALHealthStatus = 'idle' | 'healthy' | 'moderate' | 'elevated' | 'critical' | 'unavailable';

/** WAL statistics */
export interface WALStats {
  pending_count: number;
  confirmed_count: number;
  total_writes: number;
  total_confirms: number;
  total_retries: number;
  last_compaction?: string;
  db_size_bytes: number;
  db_size_formatted: string;
  write_rate_per_min?: number;
  confirm_rate_per_min?: number;
  status: WALHealthStatus;
  healthy: boolean;
}

/** WAL health check response */
export interface WALHealthResponse {
  status: WALHealthStatus;
  healthy: boolean;
  pending_count: number;
  message: string;
}

/** WAL compaction response */
export interface WALCompactionResponse {
  message: string;
  status: 'queued' | 'running' | 'completed' | 'failed';
}
