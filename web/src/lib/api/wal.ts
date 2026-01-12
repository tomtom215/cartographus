// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * WAL (Write-Ahead Log) API Module
 *
 * WAL statistics and management operations.
 */

import type {
    WALStats,
    WALHealthResponse,
    WALCompactionResponse,
} from '../types/wal';
import { BaseAPIClient } from './client';

/**
 * WAL API methods
 */
export class WALAPI extends BaseAPIClient {
    async getWALStats(): Promise<WALStats> {
        const response = await this.fetch<WALStats>('/wal/stats');
        return response.data;
    }

    async getWALHealth(): Promise<WALHealthResponse> {
        const response = await this.fetch<WALHealthResponse>('/wal/health');
        return response.data;
    }

    async triggerWALCompaction(): Promise<WALCompactionResponse> {
        const response = await this.fetch<WALCompactionResponse>('/wal/compact', { method: 'POST' });
        return response.data;
    }
}
