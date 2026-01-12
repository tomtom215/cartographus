// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * DLQ (Dead Letter Queue) API Module
 *
 * Dead letter queue management operations.
 */

import type {
    DLQEntry,
    DLQEntriesResponse,
    DLQStats,
    DLQRetryResponse,
    DLQCleanupResponse,
    DLQEntryFilter,
    DLQCategoriesResponse,
} from '../types/dlq';
import { BaseAPIClient } from './client';

/**
 * DLQ API methods
 */
export class DLQAPI extends BaseAPIClient {
    async getDLQEntries(filter: DLQEntryFilter = {}): Promise<DLQEntriesResponse> {
        const params = new URLSearchParams();
        if (filter.limit !== undefined) params.append('limit', filter.limit.toString());
        if (filter.offset !== undefined) params.append('offset', filter.offset.toString());
        if (filter.category) params.append('category', filter.category);
        if (filter.status) params.append('status', filter.status);

        const queryString = params.toString();
        const url = queryString ? `/dlq/entries?${queryString}` : '/dlq/entries';

        const response = await this.fetch<DLQEntriesResponse>(url);
        return response.data;
    }

    async getDLQEntry(eventId: string): Promise<DLQEntry> {
        const response = await this.fetch<DLQEntry>(`/dlq/entries/${encodeURIComponent(eventId)}`);
        return response.data;
    }

    async deleteDLQEntry(eventId: string): Promise<void> {
        await this.fetch<void>(`/dlq/entries/${encodeURIComponent(eventId)}`, { method: 'DELETE' });
    }

    async retryDLQEntry(eventId: string): Promise<DLQRetryResponse> {
        const response = await this.fetch<DLQRetryResponse>(
            `/dlq/entries/${encodeURIComponent(eventId)}/retry`,
            { method: 'POST' }
        );
        return response.data;
    }

    async retryAllDLQEntries(): Promise<DLQRetryResponse> {
        const response = await this.fetch<DLQRetryResponse>('/dlq/retry-all', { method: 'POST' });
        return response.data;
    }

    async getDLQStats(): Promise<DLQStats> {
        const response = await this.fetch<DLQStats>('/dlq/stats');
        return response.data;
    }

    async getDLQCategories(): Promise<string[]> {
        const response = await this.fetch<DLQCategoriesResponse>('/dlq/categories');
        return response.data.categories;
    }

    async cleanupDLQ(): Promise<DLQCleanupResponse> {
        const response = await this.fetch<DLQCleanupResponse>('/dlq/cleanup', { method: 'POST' });
        return response.data;
    }
}
