// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * API Client for Cartographus
 *
 * This file re-exports from the modular API structure for backward compatibility.
 * The API has been refactored into domain-specific modules in ./api/
 *
 * Module Structure:
 * - api/client.ts    - Base API client with fetch, auth, caching
 * - api/auth.ts      - Authentication methods
 * - api/core.ts      - Stats, health, users, media types
 * - api/analytics.ts - All analytics endpoints
 * - api/spatial.ts   - H3, arcs, viewport, streaming GeoJSON
 * - api/tautulli.ts  - Tautulli-specific endpoints
 * - api/backup.ts    - Backup/restore operations
 * - api/plex.ts      - Plex direct API
 * - api/detection.ts - Security detection alerts/rules
 * - api/dedupe.ts    - Dedupe audit operations
 * - api/audit.ts     - Security audit log
 * - api/dlq.ts       - Dead letter queue
 * - api/wal.ts       - Write-ahead log
 * - api/cross-platform.ts - Cross-platform linking
 * - api/index.ts     - Unified API facade
 *
 * For new code, prefer importing from './api' directly:
 * ```typescript
 * import { API, AnalyticsAPI, BaseAPIClient } from './api';
 * ```
 */

// Re-export the unified API class and all domain modules
export { API, BaseAPIClient, type APIResponse, type CacheStatusCallback } from './api/index';
export { AuthAPI } from './api/auth';
export { CoreAPI, LocationsAPI } from './api/core';
export { AnalyticsAPI } from './api/analytics';
export { SpatialAPI } from './api/spatial';
export { TautulliAPI } from './api/tautulli';
export { BackupAPI } from './api/backup';
export { PlexAPI } from './api/plex';
export { DetectionAPI } from './api/detection';
export { DedupeAPI } from './api/dedupe';
export { AuditAPI } from './api/audit';
export { DLQAPI } from './api/dlq';
export { WALAPI } from './api/wal';
export { CrossPlatformAPI } from './api/cross-platform';

// Re-export all types from the types directory for backward compatibility
export type {
    // Core types
    LocationStats,
    PlaybackEvent,
    PaginationInfo,
    PlaybacksResponse,
    Stats,
    HealthStatus,
    LocationFilter,
    PlaybackTrend,
    TrendsResponse,
    TemporalSpatialPoint,
    BoundingBox,
    NearbySearchParams,
    GeoJSONFeature,
    StreamingGeoJSONResponse,
    // Auth types
    LoginRequest,
    LoginResponse,
    // Analytics types
    ViewingHoursHeatmap,
    UserActivity,
    MediaTypeStats,
    CityStats,
    CountryStats,
    PlatformStats,
    PlayerStats,
    CompletionBucket,
    ContentCompletionStats,
    TranscodeStats,
    ResolutionStats,
    CodecStats,
    LibraryStats,
    RatingStats,
    DurationByMediaType,
    DurationStats,
    YearStats,
    GeographicResponse,
    UsersResponse,
    BingeSession,
    BingeShowStats,
    BingeUserStats,
    BingesByDayOfWeek,
    BingeAnalyticsResponse,
    BandwidthByTranscode,
    BandwidthByResolution,
    BandwidthByCodec,
    BandwidthTrend,
    BandwidthByUser,
    BandwidthAnalyticsResponse,
    BitrateByResolutionItem,
    BitrateTimeSeriesItem,
    BitrateAnalyticsResponse,
    PopularContent,
    PopularAnalyticsResponse,
    WatchPartyParticipant,
    WatchParty,
    WatchPartyContentStats,
    WatchPartyUserStats,
    WatchPartyByDay,
    WatchPartyAnalyticsResponse,
    UserEngagement,
    ViewingPatternByHour,
    ViewingPatternByDay,
    UserEngagementSummary,
    UserEngagementAnalyticsResponse,
    PeriodMetrics,
    ComparativeMetrics,
    TopContentComparison,
    TopUserComparison,
    ComparativeAnalyticsResponse,
    TemporalHeatmapPoint,
    TemporalHeatmapBucket,
    TemporalHeatmapResponse,
    // Approximate Analytics (DataSketches) - ADR-0018
    ApproximateStats,
    ApproximateStatsResponse,
    ApproximateDistinctResponse,
    ApproximatePercentileResponse,
    ApproximateStatsFilter,
    // Advanced analytics types
    ResolutionMismatch,
    UserDowngradeStats,
    PlatformMismatch,
    ResolutionMismatchAnalytics,
    DynamicRangeDistribution,
    ToneMappingEvent,
    HDRDeviceStats,
    ContentFormatStats,
    HDRAnalytics,
    AudioChannelDistribution,
    AudioCodecDistribution,
    AudioDownmixEvent,
    AudioAnalytics,
    SubtitleLanguageStats,
    SubtitleCodecStats,
    UserSubtitlePreference,
    SubtitleAnalytics,
    ConnectionRelayStats,
    ConnectionLocalStats,
    UserConnectionStats,
    PlatformConnectionStats,
    ConnectionSecurityAnalytics,
    HighPauseContent,
    PauseDistribution,
    PauseTimingBucket,
    UserPauseStats,
    PauseQualityMetrics,
    PausePatternAnalytics,
    ConcurrentStreamsTimeBucket,
    ConcurrentStreamsByType,
    ConcurrentStreamsByDayOfWeek,
    ConcurrentStreamsByHour,
    ConcurrentStreamsAnalytics,
    HWCodecStats,
    FullPipelineStats,
    HardwareTranscodeStats,
    AbandonmentByType,
    AbandonmentByHour,
    AbandonedContent,
    AbandonmentStats,
    // Frame Rate Analytics
    FrameRateDistribution,
    FrameRateConversion,
    FrameRateAnalytics,
    // Container Format Analytics
    ContainerDistribution,
    ContainerRemux,
    PlatformContainer,
    ContainerAnalytics,
    // Hardware Transcode Trends
    HWTranscodeTrend,
    // Tautulli types
    TautulliActivitySession,
    TautulliActivityData,
    TautulliMetadataData,
    TautulliUserData,
    TautulliLibraryUserStat,
    TautulliRecentlyAddedItem,
    TautulliRecentlyAddedData,
    TautulliLibraryDetail,
    TautulliLibraryData,
    TautulliServerInfoData,
    TautulliSyncedItem,
    TautulliLibraryNameItem,
    // User Login History
    TautulliUserLoginRow,
    TautulliUserLoginsData,
    // User IP Geolocation
    TautulliUserIPData,
    // Stream Type by Platform
    TautulliStreamTypeByPlatform,
    // Activity History
    TautulliHistoryRow,
    // Plays Per Month
    TautulliPlaysByDateSeries,
    TautulliPlaysPerMonthData,
    // Concurrent Streams by Type
    TautulliConcurrentStreamsByType,
    // Server Management types
    TautulliInfo,
    ServerListItem,
    ServerIdentity,
    PMSUpdate,
    NATSHealth,
    // Library Details Enhancement
    TautulliLibraryUserStatsItem,
    TautulliLibraryMediaInfoItem,
    TautulliLibraryMediaInfoData,
    TautulliLibraryWatchTimeStatsItem,
    TautulliLibraryTableItem,
    TautulliLibraryTableData,
    // Export Types
    TautulliExportMetadata,
    TautulliExportFieldItem,
    TautulliExportsTableRow,
    TautulliExportsTableData,
    // Rating Keys Types
    TautulliRatingKeyMapping,
    TautulliNewRatingKeysData,
    TautulliOldRatingKeysData,
    // Metadata Types
    TautulliMediaInfo,
    TautulliFullMetadataData,
    // Children Metadata Types
    TautulliChildMetadataItem,
    TautulliChildrenMetadataData,
    // Search Types
    TautulliSearchData,
    // Fuzzy Search Types (RapidFuzz Extension)
    FuzzySearchResult,
    FuzzySearchResponse,
    FuzzyUserSearchResult,
    FuzzyUserSearchResponse,
    // Stream Data Types
    TautulliStreamDataInfo,
    // Phase 3: Additional Tautulli Types
    TautulliHomeStat,
    TautulliHomeStatRow,
    TautulliHomeStatsData,
    TautulliPlaysByDateCategory,
    TautulliPlaysByDateData,
    TautulliPlaysByDayOfWeekData,
    TautulliPlaysByHourOfDayData,
    TautulliPlaysByStreamTypeData,
    TautulliPlaysByResolutionData,
    TautulliPlaysByTop10Data,
    TautulliUserDetail,
    // Cross-Platform types (Phase 3)
    ContentMapping,
    ContentMappingRequest,
    ContentMappingResponse,
    ContentLookupParams,
    UserLink,
    UserLinkRequest,
    UserLinkResponse,
    LinkedUserInfo,
    LinkedUsersResponse,
    SuggestedLinksResponse,
    CrossPlatformUserStats,
    CrossPlatformContentStats,
    CrossPlatformSummary,
    CrossPlatformUserStatsResponse,
    CrossPlatformContentStatsResponse,
    CrossPlatformSummaryResponse,
    // Plex Direct API types
    PlexServerIdentity,
    PlexLibrarySection,
    PlexMediaMetadata,
    PlexDirectSession,
    PlexDevice,
    PlexAccount,
    PlexActivity,
    PlexPlaylist,
    PlexCapabilities,
    PlexBandwidthStatistics,
    PlexMediaContainer,
    // Visualization types
    H3HexagonStats,
    ArcStats,
    ServerInfo,
    // Backup types
    Backup,
    BackupStats,
    CreateBackupRequest,
    RestoreBackupRequest,
    RestoreResult,
    // Retention types (Task 29)
    RetentionPolicy,
    RetentionPreview,
    RetentionApplyResult,
    // Library analytics types
    LibraryUserStats,
    LibraryQualityStats,
    LibraryHealthMetrics,
    LibraryAnalytics,
    // User profile types
    UserProfileStats,
    UserActivityTrend,
    UserTopContent,
    UserPlatformUsage,
    UserProfileAnalytics,
} from './types';
