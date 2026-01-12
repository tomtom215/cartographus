// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Analytics API Module
 *
 * All analytics endpoints including trends, geographic, user engagement,
 * bandwidth, bitrate, binge, comparative, and advanced analytics.
 */

import type { LocationFilter, TrendsResponse } from '../types/core';
import type {
    GeographicResponse,
    UsersResponse,
    BingeAnalyticsResponse,
    BandwidthAnalyticsResponse,
    BitrateAnalyticsResponse,
    PopularAnalyticsResponse,
    WatchPartyAnalyticsResponse,
    UserEngagementAnalyticsResponse,
    ComparativeAnalyticsResponse,
    TemporalHeatmapResponse,
    ApproximateStatsResponse,
    ApproximateDistinctResponse,
    ApproximatePercentileResponse,
    ApproximateStatsFilter,
    ContentFlowAnalytics,
    UserContentOverlapAnalytics,
    RadarUserProfileAnalytics,
    LibraryUtilizationAnalytics,
    CalendarHeatmapAnalytics,
    BumpChartAnalytics,
} from '../types/analytics';
import type {
    ResolutionMismatchAnalytics,
    HDRAnalytics,
    AudioAnalytics,
    SubtitleAnalytics,
    ConnectionSecurityAnalytics,
    PausePatternAnalytics,
    ConcurrentStreamsAnalytics,
    HardwareTranscodeStats,
    AbandonmentStats,
    FrameRateAnalytics,
    ContainerAnalytics,
    HWTranscodeTrend,
} from '../types/advanced-analytics';
import type { LibraryAnalytics } from '../types/library';
import type { UserProfileAnalytics } from '../types/user-profile';
import { BaseAPIClient } from './client';

/**
 * Analytics API methods
 */
export class AnalyticsAPI extends BaseAPIClient {
    // ========================================================================
    // Basic Analytics
    // ========================================================================

    async getAnalyticsTrends(filter: LocationFilter = {}): Promise<TrendsResponse> {
        return this.fetchWithFilter<TrendsResponse>('/analytics/trends', filter);
    }

    async getAnalyticsGeographic(filter: LocationFilter = {}): Promise<GeographicResponse> {
        return this.fetchWithFilter<GeographicResponse>('/analytics/geographic', filter);
    }

    async getAnalyticsUsers(filter: LocationFilter = {}, limit: number = 10): Promise<UsersResponse> {
        const params = this.buildFilterParams(filter);
        params.append('limit', limit.toString());
        const queryString = params.toString();
        const url = queryString ? `/analytics/users?${queryString}` : '/analytics/users';

        const response = await this.fetch<UsersResponse>(url);
        return response.data;
    }

    async getAnalyticsBinge(filter: LocationFilter = {}): Promise<BingeAnalyticsResponse> {
        return this.fetchWithFilter<BingeAnalyticsResponse>('/analytics/binge', filter);
    }

    async getAnalyticsBandwidth(filter: LocationFilter = {}): Promise<BandwidthAnalyticsResponse> {
        return this.fetchWithFilter<BandwidthAnalyticsResponse>('/analytics/bandwidth', filter);
    }

    async getAnalyticsBitrate(filter: LocationFilter = {}): Promise<BitrateAnalyticsResponse> {
        return this.fetchWithFilter<BitrateAnalyticsResponse>('/analytics/bitrate', filter);
    }

    async getAnalyticsPopular(filter: LocationFilter = {}, limit: number = 10): Promise<PopularAnalyticsResponse> {
        const params = this.buildFilterParams(filter);
        params.append('limit', limit.toString());
        const queryString = params.toString();
        const url = queryString ? `/analytics/popular?${queryString}` : '/analytics/popular';

        const response = await this.fetch<PopularAnalyticsResponse>(url);
        return response.data;
    }

    async getAnalyticsWatchParties(filter: LocationFilter = {}): Promise<WatchPartyAnalyticsResponse> {
        return this.fetchWithFilter<WatchPartyAnalyticsResponse>('/analytics/watch-parties', filter);
    }

    async getAnalyticsUserEngagement(filter: LocationFilter = {}, limit: number = 10): Promise<UserEngagementAnalyticsResponse> {
        const params = this.buildFilterParams(filter);

        if (limit !== undefined && limit >= 1 && limit <= 100) {
            params.append('limit', limit.toString());
        }

        const queryString = params.toString();
        const url = queryString ? `/analytics/user-engagement?${queryString}` : '/analytics/user-engagement';

        const response = await this.fetch<UserEngagementAnalyticsResponse>(url);
        return response.data;
    }

    async getAnalyticsComparative(
        filter: LocationFilter = {},
        comparisonType: 'week' | 'month' | 'quarter' | 'year' | 'custom' = 'week'
    ): Promise<ComparativeAnalyticsResponse> {
        const params = this.buildFilterParams(filter);
        params.append('comparison_type', comparisonType);

        const queryString = params.toString();
        const url = `/analytics/comparative?${queryString}`;

        const response = await this.fetch<ComparativeAnalyticsResponse>(url);
        return response.data;
    }

    async getTemporalHeatmap(
        filter: LocationFilter = {},
        interval: 'hour' | 'day' | 'week' | 'month' = 'hour'
    ): Promise<TemporalHeatmapResponse> {
        const params = this.buildFilterParams(filter);
        params.append('interval', interval);
        const queryString = params.toString();
        const url = queryString ? `/analytics/temporal-heatmap?${queryString}` : '/analytics/temporal-heatmap';

        const response = await this.fetch<TemporalHeatmapResponse>(url);
        return response.data;
    }

    // ========================================================================
    // Advanced Analytics (Phase 3.1)
    // ========================================================================

    async getAnalyticsResolutionMismatch(filter: LocationFilter = {}): Promise<ResolutionMismatchAnalytics> {
        return this.fetchWithFilter<ResolutionMismatchAnalytics>('/analytics/resolution-mismatch', filter);
    }

    async getAnalyticsHDR(filter: LocationFilter = {}): Promise<HDRAnalytics> {
        return this.fetchWithFilter<HDRAnalytics>('/analytics/hdr', filter);
    }

    async getAnalyticsAudio(filter: LocationFilter = {}): Promise<AudioAnalytics> {
        return this.fetchWithFilter<AudioAnalytics>('/analytics/audio', filter);
    }

    async getAnalyticsSubtitles(filter: LocationFilter = {}): Promise<SubtitleAnalytics> {
        return this.fetchWithFilter<SubtitleAnalytics>('/analytics/subtitles', filter);
    }

    async getAnalyticsConnectionSecurity(filter: LocationFilter = {}): Promise<ConnectionSecurityAnalytics> {
        return this.fetchWithFilter<ConnectionSecurityAnalytics>('/analytics/connection-security', filter);
    }

    async getAnalyticsPausePatterns(filter: LocationFilter = {}): Promise<PausePatternAnalytics> {
        return this.fetchWithFilter<PausePatternAnalytics>('/analytics/pause-patterns', filter);
    }

    async getAnalyticsConcurrentStreams(filter: LocationFilter = {}): Promise<ConcurrentStreamsAnalytics> {
        return this.fetchWithFilter<ConcurrentStreamsAnalytics>('/analytics/concurrent-streams', filter);
    }

    async getAnalyticsHardwareTranscode(filter: LocationFilter = {}): Promise<HardwareTranscodeStats> {
        return this.fetchWithFilter<HardwareTranscodeStats>('/analytics/hardware-transcode', filter);
    }

    async getAnalyticsAbandonment(filter: LocationFilter = {}): Promise<AbandonmentStats> {
        return this.fetchWithFilter<AbandonmentStats>('/analytics/abandonment', filter);
    }

    async getAnalyticsFrameRate(filter: LocationFilter = {}): Promise<FrameRateAnalytics> {
        return this.fetchWithFilter<FrameRateAnalytics>('/analytics/frame-rate', filter);
    }

    async getAnalyticsContainer(filter: LocationFilter = {}): Promise<ContainerAnalytics> {
        return this.fetchWithFilter<ContainerAnalytics>('/analytics/container', filter);
    }

    async getAnalyticsHardwareTranscodeTrends(filter: LocationFilter = {}): Promise<HWTranscodeTrend[]> {
        return this.fetchWithFilter<HWTranscodeTrend[]>('/analytics/hardware-transcode/trends', filter);
    }

    // ========================================================================
    // Library and User Profile Analytics
    // ========================================================================

    async getAnalyticsLibrary(sectionId: number, filter: LocationFilter = {}): Promise<LibraryAnalytics> {
        const params = this.buildFilterParams(filter);
        params.append('section_id', sectionId.toString());
        const queryString = params.toString();
        const url = `/analytics/library?${queryString}`;

        const response = await this.fetch<LibraryAnalytics>(url);
        return response.data;
    }

    async getAnalyticsUserProfile(username: string, filter: LocationFilter = {}): Promise<UserProfileAnalytics> {
        const params = this.buildFilterParams(filter);
        params.set('username', username);
        const queryString = params.toString();
        const url = queryString ? `/analytics/user-profile?${queryString}` : `/analytics/user-profile?username=${username}`;
        const response = await this.fetch<UserProfileAnalytics>(url);
        return response.data;
    }

    // ========================================================================
    // Approximate Analytics (DataSketches) - ADR-0018
    // ========================================================================

    async getAnalyticsApproximate(filter: ApproximateStatsFilter = {}): Promise<ApproximateStatsResponse> {
        const params = new URLSearchParams();
        if (filter.start_date) params.append('start_date', filter.start_date);
        if (filter.end_date) params.append('end_date', filter.end_date);
        if (filter.users?.length) params.append('users', filter.users.join(','));
        if (filter.media_types?.length) params.append('media_types', filter.media_types.join(','));

        const queryString = params.toString();
        const url = queryString ? `/analytics/approximate?${queryString}` : '/analytics/approximate';

        const response = await this.fetch<ApproximateStatsResponse>(url);
        return response.data;
    }

    async getAnalyticsApproximateDistinct(column: string, filter: ApproximateStatsFilter = {}): Promise<ApproximateDistinctResponse> {
        const params = new URLSearchParams();
        params.append('column', column);
        if (filter.start_date) params.append('start_date', filter.start_date);
        if (filter.end_date) params.append('end_date', filter.end_date);
        if (filter.users?.length) params.append('users', filter.users.join(','));
        if (filter.media_types?.length) params.append('media_types', filter.media_types.join(','));

        const url = `/analytics/approximate/distinct?${params.toString()}`;
        const response = await this.fetch<ApproximateDistinctResponse>(url);
        return response.data;
    }

    async getAnalyticsApproximatePercentile(
        column: string,
        percentile: number,
        filter: ApproximateStatsFilter = {}
    ): Promise<ApproximatePercentileResponse> {
        const params = new URLSearchParams();
        params.append('column', column);
        params.append('percentile', percentile.toString());
        if (filter.start_date) params.append('start_date', filter.start_date);
        if (filter.end_date) params.append('end_date', filter.end_date);
        if (filter.users?.length) params.append('users', filter.users.join(','));
        if (filter.media_types?.length) params.append('media_types', filter.media_types.join(','));

        const url = `/analytics/approximate/percentile?${params.toString()}`;
        const response = await this.fetch<ApproximatePercentileResponse>(url);
        return response.data;
    }

    // ========================================================================
    // Advanced Charts (Sankey, Chord, Radar, Treemap)
    // ========================================================================

    /**
     * Get content flow analytics for Sankey diagram visualization.
     * Shows viewing journeys from Show -> Season -> Episode.
     */
    async getAnalyticsContentFlow(filter: LocationFilter = {}): Promise<ContentFlowAnalytics> {
        return this.fetchWithFilter<ContentFlowAnalytics>('/analytics/content-flow', filter);
    }

    /**
     * Get user-user content overlap for Chord diagram visualization.
     * Shows Jaccard similarity between users based on shared content consumption.
     */
    async getAnalyticsUserOverlap(filter: LocationFilter = {}): Promise<UserContentOverlapAnalytics> {
        return this.fetchWithFilter<UserContentOverlapAnalytics>('/analytics/user-overlap', filter);
    }

    /**
     * Get multi-dimensional user engagement scores for Radar chart.
     * Dimensions: Watch Time, Completion, Diversity, Quality, Discovery, Social.
     */
    async getAnalyticsUserProfileRadar(filter: LocationFilter = {}): Promise<RadarUserProfileAnalytics> {
        return this.fetchWithFilter<RadarUserProfileAnalytics>('/analytics/user-profile', filter);
    }

    /**
     * Get hierarchical library usage for Treemap visualization.
     * Structure: Library -> Section Type -> Genre -> Content.
     */
    async getAnalyticsLibraryUtilization(filter: LocationFilter = {}): Promise<LibraryUtilizationAnalytics> {
        return this.fetchWithFilter<LibraryUtilizationAnalytics>('/analytics/library-utilization', filter);
    }

    /**
     * Get daily activity for Calendar heatmap visualization.
     * GitHub-style contribution graph showing daily viewing activity.
     */
    async getAnalyticsCalendarHeatmap(filter: LocationFilter = {}): Promise<CalendarHeatmapAnalytics> {
        return this.fetchWithFilter<CalendarHeatmapAnalytics>('/analytics/calendar-heatmap', filter);
    }

    /**
     * Get content ranking changes for Bump chart visualization.
     * Shows how top 10 content items change positions week-by-week.
     */
    async getAnalyticsBumpChart(filter: LocationFilter = {}): Promise<BumpChartAnalytics> {
        return this.fetchWithFilter<BumpChartAnalytics>('/analytics/bump-chart', filter);
    }
}
