// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Type exports - re-exports all types for backward compatibility
 *
 * Types are organized into domain-specific modules:
 * - core.ts: Core location, playback, and statistics types
 * - auth.ts: Authentication types
 * - analytics.ts: Analytics and reporting types
 * - advanced-analytics.ts: Specialized analytics (HDR, audio, subtitle, etc.)
 * - tautulli.ts: Tautulli-specific types
 * - visualization.ts: Map visualization types (H3, arcs)
 * - backup.ts: Backup/restore types
 * - library.ts: Library analytics types
 * - user-profile.ts: User profile analytics types
 */

// Core types
export type {
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
} from './core';

// Auth types
export type {
    LoginRequest,
    LoginResponse,
} from './auth';

// Analytics types
export type {
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
} from './analytics';

// Advanced analytics types
export type {
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
} from './advanced-analytics';

// Tautulli types
export type {
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
    // Server Management
    TautulliInfo,
    ServerListItem,
    ServerIdentity,
    PMSUpdate,
    NATSHealth,
    // Collections & Playlists
    TautulliCollectionItem,
    TautulliCollectionsTableData,
    TautulliPlaylistItem,
    TautulliPlaylistsTableData,
    // User IP Details
    TautulliUserIP,
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
    TautulliSearchResult,
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
} from './tautulli';

// Visualization types
export type {
    H3HexagonStats,
    ArcStats,
    ServerInfo,
} from './visualization';

// Backup types
export type {
    Backup,
    BackupStats,
    CreateBackupRequest,
    RestoreBackupRequest,
    RestoreResult,
    // Retention types (Task 29)
    RetentionPolicy,
    BackupPreviewItem,
    RetentionPreview,
    RetentionApplyResult,
    // Schedule types
    ScheduleConfig,
    SetScheduleConfigRequest,
} from './backup';

// Library analytics types
export type {
    LibraryUserStats,
    LibraryQualityStats,
    LibraryHealthMetrics,
    LibraryAnalytics,
} from './library';

// User profile types
export type {
    UserProfileStats,
    UserActivityTrend,
    UserTopContent,
    UserPlatformUsage,
    UserProfileAnalytics,
} from './user-profile';

// Plex Direct API types
export type {
    PlexServerIdentity,
    PlexLibrarySection,
    PlexLocation,
    PlexMediaMetadata,
    PlexTag,
    PlexMedia,
    PlexMediaPart,
    PlexStream,
    PlexDirectSession,
    PlexUser,
    PlexPlayer,
    PlexSessionInfo,
    PlexTranscodeSession,
    PlexDevice,
    PlexConnection,
    PlexAccount,
    PlexActivity,
    PlexActivityContext,
    PlexPlaylist,
    PlexCapabilities,
    PlexBandwidthStatistics,
    PlexMediaContainer,
} from './plex';

// Detection types (ADR-0020)
export type {
    DetectionRuleType,
    DetectionSeverity,
    DetectionAlert,
    DetectionRule,
    UserTrustScore,
    DetectionAlertFilter,
    DetectionAlertsResponse,
    DetectionRulesResponse,
    LowTrustUsersResponse,
    DetectionMetrics,
    DetectorMetrics,
    DetectionAlertStats,
    UpdateRuleRequest,
    AcknowledgeAlertRequest,
    ImpossibleTravelConfig,
    ConcurrentStreamsConfig,
    DeviceVelocityConfig,
    GeoRestrictionConfig,
    SimultaneousLocationsConfig,
} from './detection';

// Dedupe Audit types (ADR-0022)
export type {
    DedupeReason,
    DedupeLayer,
    DedupeStatus,
    DedupeAuditEntry,
    DedupeAuditListResponse,
    DedupeAuditStats,
    DedupeAuditActionRequest,
    DedupeAuditRestoreResponse,
    DedupeAuditFilter,
} from './dedupe';

// Cross-Platform types (Phase 3)
export type {
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
} from './cross-platform';

// Audit Log types
export type {
    AuditEventType,
    AuditSeverity,
    AuditOutcome,
    GeoLocation,
    AuditSource,
    AuditActor,
    AuditTarget,
    AuditEvent,
    AuditEventsResponse,
    AuditStats,
    AuditEventFilter,
    AuditTypesResponse,
    AuditSeveritiesResponse,
} from './audit';

// DLQ types
export type {
    DLQErrorCategory,
    DLQEntryStatus,
    DLQEntry,
    DLQEntriesResponse,
    DLQStats,
    DLQRetryResponse,
    DLQCleanupResponse,
    DLQEntryFilter,
    DLQCategoriesResponse,
} from './dlq';

// WAL types
export type {
    WALHealthStatus,
    WALStats,
    WALHealthResponse,
    WALCompactionResponse,
} from './wal';

// Wrapped Reports types (Annual Year-in-Review)
export type {
    WrappedContentRank,
    WrappedGenreRank,
    WrappedPersonRank,
    WrappedMonthly,
    WrappedBingeInfo,
    WrappedAchievement,
    WrappedPercentiles,
    WrappedReport,
    WrappedLeaderboardEntry,
    WrappedServerStats,
    WrappedGenerateRequest,
    WrappedGenerateResponse,
} from './wrapped';

// Enhanced Analytics types (Cohort, QoE, Data Quality, User Network, Device Migration, Content Discovery)
export type {
    // Cohort Retention
    CohortRetentionAnalytics,
    CohortData,
    WeekRetention,
    CohortRetentionSummary,
    RetentionPoint,
    CohortQueryMetadata,
    // QoE Dashboard
    QoEDashboard,
    QoESummary,
    QoETrendPoint,
    QoEByPlatform,
    QoEByTranscode,
    QoEIssue,
    QoEQueryMetadata,
    // Data Quality
    DataQualityReport,
    DataQualitySummary,
    FieldQualityMetric,
    DailyQualityTrend,
    DataQualityIssue,
    SourceQuality,
    DataQualityMetadata,
    // User Network Graph
    UserNetworkGraph,
    UserNode,
    UserEdge,
    UserCluster,
    NetworkSummary,
    NetworkMetadata,
    // Device Migration Tracking
    DeviceMigrationAnalytics,
    DeviceMigrationSummary,
    DeviceMigration,
    UserDeviceProfile,
    UserPlatformUsage as DeviceUserPlatformUsage,
    PlatformAdoptionTrend,
    PlatformTransition,
    PlatformDistribution,
    DeviceMigrationMetadata,
    // Content Discovery Analytics
    ContentDiscoveryAnalytics,
    ContentDiscoverySummary,
    DiscoveryTimeBucket,
    EarlyAdopter,
    ContentDiscoveryItem,
    StaleContent,
    LibraryDiscoveryStats,
    DiscoveryTrend,
    ContentDiscoveryMetadata,
} from './enhanced-analytics';

// Newsletter types
export type {
    NewsletterType,
    DeliveryChannel,
    DeliveryStatus,
    TimeFrameUnit,
    TemplateVariable,
    TemplateConfig,
    NewsletterTemplate,
    ChannelConfig,
    NewsletterSchedule,
    RecipientDeliveryDetail,
    DeliveryContentStats,
    NewsletterDelivery,
    NewsletterUserPreferences,
    NewsletterAuditEntry,
    NewsletterStats,
    CreateTemplateRequest,
    UpdateTemplateRequest,
    CreateScheduleRequest,
    UpdateScheduleRequest,
    PreviewNewsletterRequest,
    PreviewNewsletterResponse,
    ListTemplatesResponse,
    ListSchedulesResponse,
    ListDeliveriesResponse,
    ListAuditLogResponse,
    NewsletterFilter,
    NewsletterAuditFilter,
    NewsletterTypeConfig,
    ChannelDisplayConfig,
} from './newsletter';

export {
    NEWSLETTER_TYPES,
    CHANNEL_CONFIG,
    STATUS_COLORS as NEWSLETTER_STATUS_COLORS,
    STATUS_NAMES as NEWSLETTER_STATUS_NAMES,
} from './newsletter';
