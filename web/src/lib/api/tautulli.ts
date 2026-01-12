// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Tautulli API Module
 *
 * All Tautulli-specific endpoints including activity, server info,
 * libraries, user management, and statistics.
 */

import type {
    TautulliActivityData,
    TautulliServerInfoData,
    TautulliRecentlyAddedData,
    TautulliLibraryDetail,
    TautulliLibraryNameItem,
    TautulliSyncedItem,
    TautulliUserLoginsData,
    TautulliUserIPData,
    TautulliStreamTypeByPlatform,
    TautulliHistoryRow,
    TautulliPlaysPerMonthData,
    TautulliConcurrentStreamsByType,
    TautulliHomeStatsData,
    TautulliPlaysByDateData,
    TautulliPlaysByDayOfWeekData,
    TautulliPlaysByHourOfDayData,
    TautulliPlaysByStreamTypeData,
    TautulliPlaysByResolutionData,
    TautulliPlaysByTop10Data,
    TautulliUserDetail,
    TautulliInfo,
    ServerListItem,
    ServerIdentity,
    PMSUpdate,
    NATSHealth,
    TautulliCollectionsTableData,
    TautulliPlaylistsTableData,
    TautulliUserIP,
    TautulliLibraryUserStatsItem,
    TautulliLibraryMediaInfoData,
    TautulliLibraryWatchTimeStatsItem,
    TautulliLibraryTableData,
    TautulliExportMetadata,
    TautulliExportFieldItem,
    TautulliExportsTableData,
    TautulliNewRatingKeysData,
    TautulliOldRatingKeysData,
    TautulliRatingKeyMapping,
    TautulliFullMetadataData,
    TautulliChildrenMetadataData,
    TautulliSearchData,
    TautulliStreamDataInfo,
} from '../types/tautulli';
import type { FuzzySearchResponse, FuzzyUserSearchResponse } from '../types/tautulli';
import { BaseAPIClient } from './client';

/**
 * Tautulli API methods
 */
export class TautulliAPI extends BaseAPIClient {
    // ========================================================================
    // Activity & Server
    // ========================================================================

    async getTautulliActivity(): Promise<TautulliActivityData> {
        const response = await this.fetch<TautulliActivityData>('/tautulli/activity');
        return response.data;
    }

    async getTautulliServerInfo(): Promise<TautulliServerInfoData> {
        const response = await this.fetch<TautulliServerInfoData>('/tautulli/server-info');
        return response.data;
    }

    async getTautulliRecentlyAdded(
        count: number = 25,
        start: number = 0,
        mediaType?: string,
        sectionId?: number
    ): Promise<TautulliRecentlyAddedData> {
        const params = new URLSearchParams();
        params.append('count', count.toString());
        if (start > 0) {
            params.append('start', start.toString());
        }
        if (mediaType) {
            params.append('media_type', mediaType);
        }
        if (sectionId !== undefined && sectionId > 0) {
            params.append('section_id', sectionId.toString());
        }
        const response = await this.fetch<TautulliRecentlyAddedData>(`/tautulli/recently-added?${params.toString()}`);
        return response.data;
    }

    async getTautulliLibraries(): Promise<TautulliLibraryDetail[]> {
        const response = await this.fetch<TautulliLibraryDetail[]>('/tautulli/libraries');
        return response.data;
    }

    async getTautulliLibraryNames(): Promise<TautulliLibraryNameItem[]> {
        return this.fetchSimple<TautulliLibraryNameItem[]>('/tautulli/library-names');
    }

    // ========================================================================
    // User Management
    // ========================================================================

    async getTautulliUserLogins(userId?: number): Promise<TautulliUserLoginsData> {
        const params = new URLSearchParams();
        if (userId !== undefined) {
            params.append('user_id', userId.toString());
        }
        const queryString = params.toString();
        const url = queryString ? `/tautulli/user-logins?${queryString}` : '/tautulli/user-logins';

        const response = await this.fetch<TautulliUserLoginsData>(url);
        return response.data;
    }

    async getTautulliUserIPs(userId: number): Promise<TautulliUserIPData[]> {
        const url = `/tautulli/user-ips?user_id=${userId}`;
        const response = await this.fetch<TautulliUserIPData[]>(url);
        return response.data;
    }

    async getTautulliUser(userId: number): Promise<TautulliUserDetail> {
        const response = await this.fetch<TautulliUserDetail>(`/tautulli/user?user_id=${userId}`);
        return response.data;
    }

    async getTautulliUsers(): Promise<TautulliUserDetail[]> {
        return this.fetchSimple<TautulliUserDetail[]>('/tautulli/users');
    }

    async getUserIPDetails(userId: number): Promise<TautulliUserIP[]> {
        const response = await this.fetch<TautulliUserIP[]>(`/tautulli/user-ips-detail?user_id=${userId}`);
        return response.data;
    }

    // ========================================================================
    // Activity & History
    // ========================================================================

    async getTautulliSyncedItems(): Promise<TautulliSyncedItem[]> {
        return this.fetchSimple<TautulliSyncedItem[]>('/tautulli/synced-items');
    }

    async getTautulliStreamTypeByPlatform(): Promise<TautulliStreamTypeByPlatform[]> {
        return this.fetchSimple<TautulliStreamTypeByPlatform[]>('/tautulli/stream-type-by-top-10-platforms');
    }

    async getTautulliHistory(start: number = 0, length: number = 25): Promise<TautulliHistoryRow[]> {
        const url = `/tautulli/activity?start=${start}&length=${length}`;
        const response = await this.fetch<TautulliHistoryRow[]>(url);
        return response.data;
    }

    async getTautulliPlaysPerMonth(timeRange: number = 365, yAxis: string = 'plays'): Promise<TautulliPlaysPerMonthData> {
        const url = `/tautulli/plays-per-month?time_range=${timeRange}&y_axis=${yAxis}`;
        const response = await this.fetch<TautulliPlaysPerMonthData>(url);
        return response.data;
    }

    async getTautulliConcurrentStreamsByType(): Promise<TautulliConcurrentStreamsByType[]> {
        return this.fetchSimple<TautulliConcurrentStreamsByType[]>('/tautulli/concurrent-streams-by-stream-type');
    }

    // ========================================================================
    // Statistics
    // ========================================================================

    async getTautulliHomeStats(timeRange: number = 30, statId?: string): Promise<TautulliHomeStatsData> {
        const params = new URLSearchParams();
        params.append('time_range', timeRange.toString());
        if (statId) {
            params.append('stat_id', statId);
        }
        const response = await this.fetch<TautulliHomeStatsData>(`/tautulli/home-stats?${params.toString()}`);
        return response.data;
    }

    async getTautulliPlaysByDate(timeRange: number = 30, yAxis: string = 'plays'): Promise<TautulliPlaysByDateData> {
        const url = `/tautulli/plays-by-date?time_range=${timeRange}&y_axis=${yAxis}`;
        const response = await this.fetch<TautulliPlaysByDateData>(url);
        return response.data;
    }

    async getTautulliPlaysByDayOfWeek(timeRange: number = 30, yAxis: string = 'plays'): Promise<TautulliPlaysByDayOfWeekData> {
        const url = `/tautulli/plays-by-dayofweek?time_range=${timeRange}&y_axis=${yAxis}`;
        const response = await this.fetch<TautulliPlaysByDayOfWeekData>(url);
        return response.data;
    }

    async getTautulliPlaysByHourOfDay(timeRange: number = 30, yAxis: string = 'plays'): Promise<TautulliPlaysByHourOfDayData> {
        const url = `/tautulli/plays-by-hourofday?time_range=${timeRange}&y_axis=${yAxis}`;
        const response = await this.fetch<TautulliPlaysByHourOfDayData>(url);
        return response.data;
    }

    async getTautulliPlaysByStreamType(timeRange: number = 30, yAxis: string = 'plays'): Promise<TautulliPlaysByStreamTypeData> {
        const url = `/tautulli/plays-by-stream-type?time_range=${timeRange}&y_axis=${yAxis}`;
        const response = await this.fetch<TautulliPlaysByStreamTypeData>(url);
        return response.data;
    }

    async getTautulliPlaysBySourceResolution(timeRange: number = 30, yAxis: string = 'plays'): Promise<TautulliPlaysByResolutionData> {
        const url = `/tautulli/plays-by-source-resolution?time_range=${timeRange}&y_axis=${yAxis}`;
        const response = await this.fetch<TautulliPlaysByResolutionData>(url);
        return response.data;
    }

    async getTautulliPlaysByStreamResolution(timeRange: number = 30, yAxis: string = 'plays'): Promise<TautulliPlaysByResolutionData> {
        const url = `/tautulli/plays-by-stream-resolution?time_range=${timeRange}&y_axis=${yAxis}`;
        const response = await this.fetch<TautulliPlaysByResolutionData>(url);
        return response.data;
    }

    async getTautulliPlaysByTop10Platforms(timeRange: number = 30, yAxis: string = 'plays'): Promise<TautulliPlaysByTop10Data> {
        const url = `/tautulli/plays-by-top-10-platforms?time_range=${timeRange}&y_axis=${yAxis}`;
        const response = await this.fetch<TautulliPlaysByTop10Data>(url);
        return response.data;
    }

    async getTautulliPlaysByTop10Users(timeRange: number = 30, yAxis: string = 'plays'): Promise<TautulliPlaysByTop10Data> {
        const url = `/tautulli/plays-by-top-10-users?time_range=${timeRange}&y_axis=${yAxis}`;
        const response = await this.fetch<TautulliPlaysByTop10Data>(url);
        return response.data;
    }

    // ========================================================================
    // Server Management
    // ========================================================================

    async getTautulliInfo(): Promise<TautulliInfo> {
        return this.fetchSimple<TautulliInfo>('/tautulli/tautulli-info');
    }

    async getTautulliServerList(): Promise<ServerListItem[]> {
        return this.fetchSimple<ServerListItem[]>('/tautulli/server-list');
    }

    async getTautulliServerIdentity(): Promise<ServerIdentity> {
        return this.fetchSimple<ServerIdentity>('/tautulli/server-identity');
    }

    async getTautulliPMSUpdate(): Promise<PMSUpdate> {
        return this.fetchSimple<PMSUpdate>('/tautulli/pms-update');
    }

    async getNATSHealth(): Promise<NATSHealth> {
        return this.fetchSimple<NATSHealth>('/health/nats');
    }

    async terminateSession(sessionId: string, message?: string): Promise<{ result: string }> {
        const params = new URLSearchParams();
        params.append('session_id', sessionId);
        if (message) {
            params.append('message', message);
        }
        const response = await this.fetch<{ result: string }>(`/tautulli/terminate-session?${params.toString()}`, {
            method: 'POST'
        });
        return response.data;
    }

    // ========================================================================
    // Collections & Playlists
    // ========================================================================

    async getCollectionsTable(
        sectionId?: number,
        start: number = 0,
        length: number = 25
    ): Promise<TautulliCollectionsTableData> {
        const params = new URLSearchParams();
        if (sectionId !== undefined) {
            params.append('section_id', sectionId.toString());
        }
        params.append('start', start.toString());
        params.append('length', length.toString());
        const response = await this.fetch<TautulliCollectionsTableData>(`/tautulli/collections-table?${params.toString()}`);
        return response.data;
    }

    async getPlaylistsTable(
        start: number = 0,
        length: number = 25
    ): Promise<TautulliPlaylistsTableData> {
        const params = new URLSearchParams();
        params.append('start', start.toString());
        params.append('length', length.toString());
        const response = await this.fetch<TautulliPlaylistsTableData>(`/tautulli/playlists-table?${params.toString()}`);
        return response.data;
    }

    // ========================================================================
    // Library Details
    // ========================================================================

    async getLibraryUserStats(sectionId: number): Promise<TautulliLibraryUserStatsItem[]> {
        const response = await this.fetch<TautulliLibraryUserStatsItem[]>(
            `/tautulli/library-user-stats?section_id=${sectionId}`
        );
        return response.data;
    }

    async getLibraryMediaInfo(
        sectionId: number,
        start: number = 0,
        length: number = 25,
        orderColumn?: string,
        orderDir: 'asc' | 'desc' = 'desc'
    ): Promise<TautulliLibraryMediaInfoData> {
        const params = new URLSearchParams();
        params.append('section_id', sectionId.toString());
        params.append('start', start.toString());
        params.append('length', length.toString());
        if (orderColumn) {
            params.append('order_column', orderColumn);
            params.append('order_dir', orderDir);
        }
        const response = await this.fetch<TautulliLibraryMediaInfoData>(
            `/tautulli/library-media-info?${params.toString()}`
        );
        return response.data;
    }

    async getLibraryWatchTimeStats(
        sectionId: number,
        queryDays: number[] = [1, 7, 30, 0]
    ): Promise<TautulliLibraryWatchTimeStatsItem[]> {
        const params = new URLSearchParams();
        params.append('section_id', sectionId.toString());
        params.append('query_days', queryDays.join(','));
        const response = await this.fetch<TautulliLibraryWatchTimeStatsItem[]>(
            `/tautulli/library-watch-time-stats?${params.toString()}`
        );
        return response.data;
    }

    async getLibrariesTable(): Promise<TautulliLibraryTableData> {
        return this.fetchSimple<TautulliLibraryTableData>('/tautulli/libraries-table');
    }

    // ========================================================================
    // Export
    // ========================================================================

    async getExportFields(mediaType: string): Promise<TautulliExportFieldItem[]> {
        const response = await this.fetch<TautulliExportFieldItem[]>(
            `/tautulli/export-fields?media_type=${encodeURIComponent(mediaType)}`
        );
        return response.data;
    }

    async getExportsTable(
        start: number = 0,
        length: number = 25,
        orderColumn?: string,
        orderDir: 'asc' | 'desc' = 'desc'
    ): Promise<TautulliExportsTableData> {
        const params = new URLSearchParams();
        params.append('start', start.toString());
        params.append('length', length.toString());
        if (orderColumn) {
            params.append('order_column', orderColumn);
            params.append('order_dir', orderDir);
        }
        const response = await this.fetch<TautulliExportsTableData>(
            `/tautulli/exports-table?${params.toString()}`
        );
        return response.data;
    }

    async createExport(
        sectionId: number,
        exportType: string,
        fileFormat: string = 'csv',
        userId?: number,
        ratingKey?: number
    ): Promise<TautulliExportMetadata> {
        const params = new URLSearchParams();
        params.append('section_id', sectionId.toString());
        params.append('export_type', exportType);
        params.append('file_format', fileFormat);
        if (userId !== undefined) {
            params.append('user_id', userId.toString());
        }
        if (ratingKey !== undefined) {
            params.append('rating_key', ratingKey.toString());
        }
        const response = await this.fetch<TautulliExportMetadata>(
            `/tautulli/export-metadata?${params.toString()}`
        );
        return response.data;
    }

    getExportDownloadUrl(exportId: number): string {
        return `${this.baseURL}/tautulli/download-export?export_id=${exportId}`;
    }

    async deleteExport(exportId: number): Promise<void> {
        await this.fetch<void>(`/tautulli/delete-export?export_id=${exportId}`, {
            method: 'DELETE'
        });
    }

    // ========================================================================
    // Metadata
    // ========================================================================

    async getNewRatingKeys(ratingKey: string): Promise<TautulliRatingKeyMapping[]> {
        const response = await this.fetch<TautulliNewRatingKeysData>(
            `/tautulli/new-rating-keys?rating_key=${encodeURIComponent(ratingKey)}`
        );
        return response.data.rating_keys || [];
    }

    async getOldRatingKeys(ratingKey: string): Promise<TautulliRatingKeyMapping[]> {
        const response = await this.fetch<TautulliOldRatingKeysData>(
            `/tautulli/old-rating-keys?rating_key=${encodeURIComponent(ratingKey)}`
        );
        return response.data.rating_keys || [];
    }

    async getMetadata(ratingKey: string): Promise<TautulliFullMetadataData> {
        const response = await this.fetch<TautulliFullMetadataData>(
            `/tautulli/metadata?rating_key=${encodeURIComponent(ratingKey)}`
        );
        return response.data;
    }

    async getChildrenMetadata(ratingKey: string): Promise<TautulliChildrenMetadataData> {
        const response = await this.fetch<TautulliChildrenMetadataData>(
            `/tautulli/children-metadata?rating_key=${encodeURIComponent(ratingKey)}`
        );
        return response.data;
    }

    // ========================================================================
    // Search
    // ========================================================================

    async search(query: string, limit: number = 25): Promise<TautulliSearchData> {
        const params = new URLSearchParams();
        params.append('query', query);
        params.append('limit', limit.toString());
        const response = await this.fetch<TautulliSearchData>(
            `/tautulli/search?${params.toString()}`
        );
        return response.data;
    }

    async fuzzySearch(
        query: string,
        options?: { minScore?: number; limit?: number }
    ): Promise<FuzzySearchResponse> {
        const params = new URLSearchParams();
        params.append('q', query);
        if (options?.minScore !== undefined) {
            params.append('min_score', options.minScore.toString());
        }
        if (options?.limit !== undefined) {
            params.append('limit', options.limit.toString());
        }
        const response = await this.fetch<FuzzySearchResponse>(
            `/search/fuzzy?${params.toString()}`
        );
        return response.data;
    }

    async fuzzySearchUsers(
        query: string,
        options?: { minScore?: number; limit?: number }
    ): Promise<FuzzyUserSearchResponse> {
        const params = new URLSearchParams();
        params.append('q', query);
        if (options?.minScore !== undefined) {
            params.append('min_score', options.minScore.toString());
        }
        if (options?.limit !== undefined) {
            params.append('limit', options.limit.toString());
        }
        const response = await this.fetch<FuzzyUserSearchResponse>(
            `/search/users?${params.toString()}`
        );
        return response.data;
    }

    // ========================================================================
    // Stream Data
    // ========================================================================

    async getStreamData(sessionKey?: string, rowId?: number): Promise<TautulliStreamDataInfo> {
        const params = new URLSearchParams();
        if (sessionKey) {
            params.append('session_key', sessionKey);
        }
        if (rowId !== undefined) {
            params.append('row_id', rowId.toString());
        }
        const response = await this.fetch<TautulliStreamDataInfo>(
            `/tautulli/stream-data?${params.toString()}`
        );
        return response.data;
    }
}
