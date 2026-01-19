// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Core API Module
 *
 * Basic stats, health, users, and media type endpoints.
 */

import type {
    Stats,
    HealthStatus,
    LocationStats,
    LocationFilter,
    PlaybackEvent,
    PlaybacksResponse,
} from '../types/core';
import type { ServerInfo } from '../types/visualization';
import { BaseAPIClient } from './client';

/**
 * Core API methods for basic operations
 */
export class CoreAPI extends BaseAPIClient {
    async getStats(): Promise<Stats> {
        const response = await this.fetch<Stats>('/stats');
        return response.data;
    }

    /**
     * Get system health status for data quality indicators
     */
    async getHealthStatus(): Promise<HealthStatus> {
        const response = await this.fetch<HealthStatus>('/health');
        return response.data;
    }

    async getUsers(): Promise<string[]> {
        const response = await this.fetch<string[]>('/users');
        return response.data;
    }

    async getMediaTypes(): Promise<string[]> {
        const response = await this.fetch<string[]>('/media-types');
        return response.data;
    }

    async triggerSync(): Promise<void> {
        await this.fetch('/sync', { method: 'POST' });
    }

    /**
     * Get server location info
     */
    async getServerInfo(): Promise<ServerInfo> {
        return this.fetchSimple<ServerInfo>('/server-info');
    }
}

/**
 * Locations and Playbacks API methods
 */
export class LocationsAPI extends BaseAPIClient {
    async getLocations(filter: LocationFilter = {}): Promise<LocationStats[]> {
        const params = this.buildFilterParams(filter);

        if (filter.limit !== undefined) {
            params.append('limit', filter.limit.toString());
        }

        const queryString = params.toString();
        const url = queryString ? `/locations?${queryString}` : '/locations';

        const response = await this.fetch<LocationStats[]>(url);
        return response.data;
    }

    /**
     * Get playback events with offset-based pagination (legacy)
     *
     * @deprecated Since v1.0.0 - Use {@link getPlaybacksWithCursor} instead.
     * Offset-based pagination has O(n) performance for large datasets because
     * the database must scan and skip `offset` rows. Cursor-based pagination
     * maintains O(1) performance regardless of dataset size.
     *
     * @see {@link getPlaybacksWithCursor} - Recommended cursor-based alternative
     * @see {@link getAllPlaybacks} - Fetches all pages using cursor pagination
     */
    async getPlaybacks(filter: LocationFilter = {}, limit: number = 100, offset: number = 0): Promise<PlaybackEvent[]> {
        const params = this.buildFilterParams(filter);
        params.append('limit', limit.toString());
        params.append('offset', offset.toString());

        const queryString = params.toString();
        const url = queryString ? `/playbacks?${queryString}` : '/playbacks';

        const response = await this.fetch<PlaybacksResponse | PlaybackEvent[]>(url);
        if (Array.isArray(response.data)) {
            return response.data;
        }
        return (response.data as PlaybacksResponse).events || [];
    }

    /**
     * Get playback events with cursor-based pagination (recommended)
     */
    async getPlaybacksWithCursor(
        filter: LocationFilter = {},
        limit: number = 100,
        cursor?: string
    ): Promise<PlaybacksResponse> {
        const params = this.buildFilterParams(filter);
        params.append('limit', limit.toString());
        if (cursor) {
            params.append('cursor', cursor);
        }

        const queryString = params.toString();
        const url = queryString ? `/playbacks?${queryString}` : '/playbacks';

        const response = await this.fetch<PlaybacksResponse>(url);
        return response.data;
    }

    /**
     * Fetch all playback events using cursor-based pagination
     */
    async getAllPlaybacks(
        filter: LocationFilter = {},
        pageSize: number = 100,
        onProgress?: (loaded: number, hasMore: boolean) => void
    ): Promise<PlaybackEvent[]> {
        const allEvents: PlaybackEvent[] = [];
        let cursor: string | undefined;
        let hasMore = true;

        while (hasMore) {
            const response = await this.getPlaybacksWithCursor(filter, pageSize, cursor);
            allEvents.push(...response.events);
            hasMore = response.pagination.has_more;
            cursor = response.pagination.next_cursor;

            if (onProgress) {
                onProgress(allEvents.length, hasMore);
            }
        }

        return allEvents;
    }
}
