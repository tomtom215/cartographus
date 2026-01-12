// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Chart Registry - Single source of truth for all chart configurations
 *
 * This eliminates duplicate chart ID lists and provides centralized
 * configuration for chart initialization, rendering, and descriptions.
 */

import { TrendsChartRenderer } from './renderers/TrendsChartRenderer';
import { GeographicChartRenderer } from './renderers/GeographicChartRenderer';
import { UserChartRenderer } from './renderers/UserChartRenderer';
import { BingeChartRenderer } from './renderers/BingeChartRenderer';
import { WatchPartyChartRenderer } from './renderers/WatchPartyChartRenderer';
import { PopularChartRenderer } from './renderers/PopularChartRenderer';
import { BandwidthChartRenderer } from './renderers/BandwidthChartRenderer';
import { BitrateChartRenderer } from './renderers/BitrateChartRenderer';
import { EngagementChartRenderer } from './renderers/EngagementChartRenderer';
import { ComparativeChartRenderer } from './renderers/ComparativeChartRenderer';
import { AdvancedChartRenderer } from './renderers/AdvancedChartRenderer';
import { LibraryChartRenderer } from './renderers/LibraryChartRenderer';
import { UserProfileChartRenderer } from './renderers/UserProfileChartRenderer';
import { TautulliChartRenderer } from './renderers/TautulliChartRenderer';
import type { ChartRendererConstructor } from './types';

// Chart configuration interface
export interface ChartConfig {
  id: string;
  description: string;  // For accessibility (aria-label)
  renderer: ChartRendererConstructor;  // Renderer class constructor
  renderMethod: string; // Method name to call on renderer
  dataKey: string;      // Key in data map (e.g., 'trendsData', 'geographicData')
}

// Complete registry of all 47 charts across 6 analytics pages
export const CHART_REGISTRY: ChartConfig[] = [
  // Overview Page (6 charts)
  { id: 'chart-trends', description: 'Line chart showing playback trends over time', renderer: TrendsChartRenderer, renderMethod: 'render', dataKey: 'trendsData' },
  { id: 'chart-countries', description: 'Bar chart showing top 10 countries by playback count', renderer: GeographicChartRenderer, renderMethod: 'renderCountries', dataKey: 'geographicData' },
  { id: 'chart-cities', description: 'Bar chart showing top 10 cities by playback count', renderer: GeographicChartRenderer, renderMethod: 'renderCities', dataKey: 'geographicData' },
  { id: 'chart-media', description: 'Pie chart showing media type distribution', renderer: GeographicChartRenderer, renderMethod: 'renderMediaDistribution', dataKey: 'geographicData' },
  { id: 'chart-users', description: 'Bar chart showing top users by playback count', renderer: UserChartRenderer, renderMethod: 'render', dataKey: 'usersData' },
  { id: 'chart-heatmap', description: 'Heatmap showing viewing hours by day and time', renderer: GeographicChartRenderer, renderMethod: 'renderHeatmap', dataKey: 'geographicData' },

  // Content Page (8 charts)
  { id: 'chart-platforms', description: 'Pie chart showing platform distribution', renderer: GeographicChartRenderer, renderMethod: 'renderPlatforms', dataKey: 'geographicData' },
  { id: 'chart-players', description: 'Pie chart showing player distribution', renderer: GeographicChartRenderer, renderMethod: 'renderPlayers', dataKey: 'geographicData' },
  { id: 'chart-libraries', description: 'Pie chart showing library distribution', renderer: GeographicChartRenderer, renderMethod: 'renderLibraries', dataKey: 'geographicData' },
  { id: 'chart-ratings', description: 'Bar chart showing content rating distribution', renderer: GeographicChartRenderer, renderMethod: 'renderRatings', dataKey: 'geographicData' },
  { id: 'chart-years', description: 'Bar chart showing top release years', renderer: GeographicChartRenderer, renderMethod: 'renderYears', dataKey: 'geographicData' },
  { id: 'chart-popular-movies', description: 'Bar chart showing top movies by playback count', renderer: PopularChartRenderer, renderMethod: 'renderMovies', dataKey: 'popularData' },
  { id: 'chart-popular-shows', description: 'Bar chart showing top TV shows by playback count', renderer: PopularChartRenderer, renderMethod: 'renderShows', dataKey: 'popularData' },
  { id: 'chart-popular-episodes', description: 'Bar chart showing top episodes by playback count', renderer: PopularChartRenderer, renderMethod: 'renderEpisodes', dataKey: 'popularData' },

  // Users & Behavior Page (10 charts)
  { id: 'chart-completion', description: 'Bar chart showing content completion analytics', renderer: GeographicChartRenderer, renderMethod: 'renderCompletion', dataKey: 'geographicData' },
  { id: 'chart-duration', description: 'Box plot showing duration statistics by media type', renderer: GeographicChartRenderer, renderMethod: 'renderDuration', dataKey: 'geographicData' },
  { id: 'chart-binge-summary', description: 'Multi-metric chart showing binge watching analytics summary', renderer: BingeChartRenderer, renderMethod: 'renderSummary', dataKey: 'bingeData' },
  { id: 'chart-binge-shows', description: 'Bar chart showing top binge-watched shows', renderer: BingeChartRenderer, renderMethod: 'renderShows', dataKey: 'bingeData' },
  { id: 'chart-binge-users', description: 'Bar chart showing top binge watchers', renderer: BingeChartRenderer, renderMethod: 'renderUsers', dataKey: 'bingeData' },
  { id: 'chart-watch-parties-summary', description: 'Multi-metric chart showing watch party and social viewing summary', renderer: WatchPartyChartRenderer, renderMethod: 'renderSummary', dataKey: 'watchPartiesData' },
  { id: 'chart-watch-parties-content', description: 'Bar chart showing top watch party content', renderer: WatchPartyChartRenderer, renderMethod: 'renderContent', dataKey: 'watchPartiesData' },
  { id: 'chart-watch-parties-users', description: 'Bar chart showing most social users', renderer: WatchPartyChartRenderer, renderMethod: 'renderUsers', dataKey: 'watchPartiesData' },
  { id: 'chart-engagement-summary', description: 'Multi-metric chart showing engagement summary', renderer: EngagementChartRenderer, renderMethod: 'renderSummary', dataKey: 'engagementData' },
  { id: 'chart-engagement-hours', description: 'Bar chart showing viewing patterns by hour of day', renderer: EngagementChartRenderer, renderMethod: 'renderHours', dataKey: 'engagementData' },
  { id: 'chart-engagement-days', description: 'Bar chart showing viewing patterns by day of week', renderer: EngagementChartRenderer, renderMethod: 'renderDays', dataKey: 'engagementData' },

  // Performance Page (16 charts)
  { id: 'chart-transcode', description: 'Pie chart showing transcode vs direct play distribution', renderer: GeographicChartRenderer, renderMethod: 'renderTranscode', dataKey: 'geographicData' },
  { id: 'chart-resolution', description: 'Bar chart showing video resolution distribution', renderer: GeographicChartRenderer, renderMethod: 'renderResolution', dataKey: 'geographicData' },
  { id: 'chart-codec', description: 'Bar chart showing codec combination distribution', renderer: GeographicChartRenderer, renderMethod: 'renderCodec', dataKey: 'geographicData' },
  { id: 'chart-bandwidth-trends', description: 'Line chart showing bandwidth usage trends over time', renderer: BandwidthChartRenderer, renderMethod: 'renderTrends', dataKey: 'bandwidthData' },
  { id: 'chart-bandwidth-transcode', description: 'Bar chart showing bandwidth by transcode decision', renderer: BandwidthChartRenderer, renderMethod: 'renderTranscode', dataKey: 'bandwidthData' },
  { id: 'chart-bandwidth-resolution', description: 'Bar chart showing bandwidth by resolution', renderer: BandwidthChartRenderer, renderMethod: 'renderResolution', dataKey: 'bandwidthData' },
  { id: 'chart-bandwidth-users', description: 'Bar chart showing top bandwidth users', renderer: BandwidthChartRenderer, renderMethod: 'renderUsers', dataKey: 'bandwidthData' },
  { id: 'chart-bitrate-distribution', description: 'Histogram showing bitrate distribution across all sessions', renderer: BitrateChartRenderer, renderMethod: 'renderDistribution', dataKey: 'bitrateData' },
  { id: 'chart-bitrate-utilization', description: 'Area chart showing bandwidth utilization over 30-day period', renderer: BitrateChartRenderer, renderMethod: 'renderUtilization', dataKey: 'bitrateData' },
  { id: 'chart-bitrate-resolution', description: 'Bar chart comparing average bitrate by resolution tier', renderer: BitrateChartRenderer, renderMethod: 'renderByResolution', dataKey: 'bitrateData' },
  { id: 'chart-resolution-mismatch', description: 'Bar chart showing resolution quality loss analysis', renderer: AdvancedChartRenderer, renderMethod: 'renderResolutionMismatch', dataKey: 'resolutionMismatchData' },
  { id: 'chart-hdr-analytics', description: 'Pie chart showing HDR and dynamic range distribution', renderer: AdvancedChartRenderer, renderMethod: 'renderHDRAnalytics', dataKey: 'hdrData' },
  { id: 'chart-audio-analytics', description: 'Bar chart showing audio quality distribution', renderer: AdvancedChartRenderer, renderMethod: 'renderAudioAnalytics', dataKey: 'audioData' },
  { id: 'chart-subtitle-analytics', description: 'Pie chart showing subtitle usage analytics', renderer: AdvancedChartRenderer, renderMethod: 'renderSubtitleAnalytics', dataKey: 'subtitleData' },
  { id: 'chart-connection-security', description: 'Pie chart showing connection security overview', renderer: AdvancedChartRenderer, renderMethod: 'renderConnectionSecurity', dataKey: 'connectionSecurityData' },
  { id: 'chart-pause-patterns', description: 'Line chart showing pause pattern and engagement analysis', renderer: AdvancedChartRenderer, renderMethod: 'renderPausePatterns', dataKey: 'pausePatternData' },
  { id: 'chart-concurrent-streams', description: 'Line chart showing concurrent streams analytics', renderer: AdvancedChartRenderer, renderMethod: 'renderConcurrentStreams', dataKey: 'concurrentStreamsData' },

  // Geographic Page (2 charts) - using existing charts
  // Note: These are duplicates from Overview for dedicated geographic view

  // Advanced Analytics Page (7 charts)
  { id: 'chart-comparative-metrics', description: 'Multi-series bar chart showing period comparison metrics', renderer: ComparativeChartRenderer, renderMethod: 'renderMetrics', dataKey: 'comparativeData' },
  { id: 'chart-comparative-content', description: 'Bar chart showing top content comparison between periods', renderer: ComparativeChartRenderer, renderMethod: 'renderContent', dataKey: 'comparativeData' },
  { id: 'chart-comparative-users', description: 'Bar chart showing top users comparison between periods', renderer: ComparativeChartRenderer, renderMethod: 'renderUsers', dataKey: 'comparativeData' },
  { id: 'chart-temporal-heatmap', description: 'Animated heatmap showing geographic density over time', renderer: AdvancedChartRenderer, renderMethod: 'renderTemporalHeatmap', dataKey: 'temporalHeatmapData' },

  // Hardware Transcode Analytics
  { id: 'chart-hardware-transcode', description: 'Chart showing GPU utilization breakdown (NVIDIA NVENC, Intel Quick Sync, AMD VCE)', renderer: AdvancedChartRenderer, renderMethod: 'renderHardwareTranscode', dataKey: 'hardwareTranscodeData' },

  // Abandonment Analytics
  { id: 'chart-abandonment', description: 'Chart showing content abandonment patterns and drop-off analysis', renderer: AdvancedChartRenderer, renderMethod: 'renderAbandonment', dataKey: 'abandonmentData' },

  // Library Deep-Dive Dashboard
  { id: 'chart-library-users', description: 'Chart showing top users for selected library', renderer: LibraryChartRenderer, renderMethod: 'renderLibraryUsers', dataKey: 'libraryData' },
  { id: 'chart-library-trend', description: 'Chart showing daily playback trend for selected library', renderer: LibraryChartRenderer, renderMethod: 'renderLibraryTrend', dataKey: 'libraryData' },
  { id: 'chart-library-quality', description: 'Chart showing quality distribution (HDR, 4K, Surround) for selected library', renderer: LibraryChartRenderer, renderMethod: 'renderLibraryQuality', dataKey: 'libraryData' },

  // User Profile Analytics
  { id: 'chart-user-activity-trend', description: 'Chart showing user activity trend over time', renderer: UserProfileChartRenderer, renderMethod: 'renderUserActivityTrend', dataKey: 'userProfileData' },
  { id: 'chart-user-top-content', description: 'Chart showing user top watched content', renderer: UserProfileChartRenderer, renderMethod: 'renderUserTopContent', dataKey: 'userProfileData' },
  { id: 'chart-user-platforms', description: 'Chart showing user platform usage distribution', renderer: UserProfileChartRenderer, renderMethod: 'renderUserPlatforms', dataKey: 'userProfileData' },

  // Frame Rate Analytics
  { id: 'chart-frame-rate', description: 'Chart showing frame rate distribution (24/30/60fps) with completion rates', renderer: AdvancedChartRenderer, renderMethod: 'renderFrameRate', dataKey: 'frameRateData' },

  // Container Format Analytics
  { id: 'chart-container-format', description: 'Chart showing container format distribution (MKV, MP4, AVI) with direct play rates', renderer: AdvancedChartRenderer, renderMethod: 'renderContainer', dataKey: 'containerData' },

  // Hardware Transcode Trends
  { id: 'chart-hw-transcode-trends', description: 'Line chart showing historical hardware vs software transcoding trends', renderer: AdvancedChartRenderer, renderMethod: 'renderHardwareTranscodeTrends', dataKey: 'hwTranscodeTrendsData' },

  // Tautulli Data Analytics
  // User Login History
  { id: 'chart-user-logins', description: 'Stacked bar chart showing user login attempts with success/failure breakdown', renderer: TautulliChartRenderer, renderMethod: 'renderUserLogins', dataKey: 'userLoginsData' },

  // User IP Geolocation
  { id: 'chart-user-ips', description: 'Horizontal bar chart showing IP addresses and their usage patterns', renderer: TautulliChartRenderer, renderMethod: 'renderUserIPs', dataKey: 'userIPsData' },

  // Synced Items Dashboard
  { id: 'chart-synced-items', description: 'Stacked bar chart showing offline sync status by user', renderer: TautulliChartRenderer, renderMethod: 'renderSyncedItems', dataKey: 'syncedItemsData' },

  // Stream Type by Platform
  { id: 'chart-stream-type-platform', description: 'Stacked bar chart showing direct play vs transcode by platform', renderer: TautulliChartRenderer, renderMethod: 'renderStreamTypeByPlatform', dataKey: 'streamTypeByPlatformData' },

  // Activity History Timeline
  { id: 'chart-activity-history', description: 'Area line chart showing recent playback activity timeline', renderer: TautulliChartRenderer, renderMethod: 'renderActivityHistory', dataKey: 'activityHistoryData' },

  // Plays Per Month
  { id: 'chart-plays-per-month', description: 'Stacked bar chart showing monthly playback statistics by media type', renderer: TautulliChartRenderer, renderMethod: 'renderPlaysPerMonth', dataKey: 'playsPerMonthData' },

  // Concurrent Streams by Type
  { id: 'chart-concurrent-streams-type', description: 'Pie chart showing concurrent stream distribution by transcode decision', renderer: TautulliChartRenderer, renderMethod: 'renderConcurrentStreamsByType', dataKey: 'concurrentStreamsByTypeData' },
];

// Derived constants for convenient access
export const ALL_CHART_IDS = CHART_REGISTRY.map(c => c.id);
export const CHART_DESCRIPTIONS = new Map(CHART_REGISTRY.map(c => [c.id, c.description]));
export const CHART_DATA_KEYS = new Map(CHART_REGISTRY.map(c => [c.id, c.dataKey]));

/**
 * Get chart configuration by ID
 */
export function getChartConfig(chartId: string): ChartConfig | undefined {
  return CHART_REGISTRY.find(c => c.id === chartId);
}

/**
 * Get all chart IDs for a specific data key
 * Useful for updating multiple charts that share the same data source
 */
export function getChartIdsByDataKey(dataKey: string): string[] {
  return CHART_REGISTRY.filter(c => c.dataKey === dataKey).map(c => c.id);
}
