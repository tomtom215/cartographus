// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Analytics types for various dashboard and reporting features
 */

export interface ViewingHoursHeatmap {
    day_of_week: number;
    hour: number;
    playback_count: number;
}

export interface UserActivity {
    username: string;
    playback_count: number;
    total_duration_minutes: number;
    avg_completion: number;
    unique_media: number;
}

export interface MediaTypeStats {
    media_type: string;
    playback_count: number;
    unique_users: number;
}

export interface CityStats {
    city: string;
    country: string;
    playback_count: number;
    unique_users: number;
}

export interface CountryStats {
    country: string;
    playback_count: number;
    unique_users: number;
}

export interface PlatformStats {
    platform: string;
    playback_count: number;
    unique_users: number;
}

export interface PlayerStats {
    player: string;
    playback_count: number;
    unique_users: number;
}

export interface CompletionBucket {
    bucket: string;
    min_percent: number;
    max_percent: number;
    playback_count: number;
    avg_completion: number;
}

export interface ContentCompletionStats {
    buckets: CompletionBucket[];
    total_playbacks: number;
    avg_completion: number;
    fully_watched: number;
    fully_watched_pct: number;
}

export interface TranscodeStats {
    transcode_decision: string;
    playback_count: number;
    percentage: number;
}

export interface ResolutionStats {
    video_resolution: string;
    playback_count: number;
    percentage: number;
}

export interface CodecStats {
    video_codec: string;
    audio_codec: string;
    playback_count: number;
    percentage: number;
}

export interface LibraryStats {
    section_id: number;
    library_name: string;
    playback_count: number;
    unique_users: number;
    total_duration_minutes: number;
    avg_completion: number;
}

export interface RatingStats {
    content_rating: string;
    playback_count: number;
    percentage: number;
}

export interface DurationByMediaType {
    media_type: string;
    avg_duration: number;
    total_duration: number;
    playback_count: number;
    avg_completion: number;
}

export interface DurationStats {
    avg_duration: number;
    median_duration: number;
    total_duration: number;
    avg_completion: number;
    fully_watched: number;
    fully_watched_pct: number;
    partially_watched: number;
    duration_by_media_type: DurationByMediaType[];
}

export interface YearStats {
    year: number;
    playback_count: number;
}

export interface GeographicResponse {
    top_cities: CityStats[];
    top_countries: CountryStats[];
    media_type_distribution: MediaTypeStats[];
    viewing_hours_heatmap: ViewingHoursHeatmap[];
    platform_distribution: PlatformStats[];
    player_distribution: PlayerStats[];
    content_completion_stats: ContentCompletionStats;
    transcode_distribution: TranscodeStats[];
    resolution_distribution: ResolutionStats[];
    codec_distribution: CodecStats[];
    library_distribution: LibraryStats[];
    rating_distribution: RatingStats[];
    duration_stats: DurationStats;
    year_distribution: YearStats[];
}

export interface UsersResponse {
    top_users: UserActivity[];
}

// Binge Analytics
export interface BingeSession {
    user_id: number;
    username: string;
    show_name: string;
    episode_count: number;
    first_episode_time: string;
    last_episode_time: string;
    total_duration: number;
    avg_completion: number;
}

export interface BingeShowStats {
    show_name: string;
    binge_count: number;
    total_episodes: number;
    unique_watchers: number;
    avg_episodes_per_binge: number;
}

export interface BingeUserStats {
    user_id: number;
    username: string;
    binge_count: number;
    total_episodes: number;
    avg_episodes_per_binge: number;
    favorite_show: string;
}

export interface BingesByDayOfWeek {
    day_of_week: number;
    binge_count: number;
    avg_episodes: number;
}

export interface BingeAnalyticsResponse {
    total_binge_sessions: number;
    total_episodes_binged: number;
    avg_episodes_per_binge: number;
    avg_binge_duration: number;
    top_binge_shows: BingeShowStats[];
    top_binge_watchers: BingeUserStats[];
    recent_binge_sessions: BingeSession[];
    binges_by_day: BingesByDayOfWeek[];
}

// Bandwidth Analytics
export interface BandwidthByTranscode {
    transcode_decision: string;
    bandwidth_gb: number;
    playback_count: number;
    avg_bandwidth_mbps: number;
    percentage: number;
}

export interface BandwidthByResolution {
    resolution: string;
    bandwidth_gb: number;
    playback_count: number;
    avg_bandwidth_mbps: number;
    percentage: number;
}

export interface BandwidthByCodec {
    video_codec: string;
    audio_codec: string;
    bandwidth_gb: number;
    playback_count: number;
    avg_bandwidth_mbps: number;
}

export interface BandwidthTrend {
    date: string;
    bandwidth_gb: number;
    playback_count: number;
    avg_mbps: number;
}

export interface BandwidthByUser {
    user_id: number;
    username: string;
    bandwidth_gb: number;
    playback_count: number;
    direct_play_count: number;
    transcode_count: number;
    avg_bandwidth_mbps: number;
}

export interface BandwidthAnalyticsResponse {
    total_bandwidth_gb: number;
    direct_play_bandwidth_gb: number;
    transcode_bandwidth_gb: number;
    avg_bandwidth_mbps: number;
    peak_bandwidth_mbps: number;
    by_transcode: BandwidthByTranscode[];
    by_resolution: BandwidthByResolution[];
    by_codec: BandwidthByCodec[];
    trends: BandwidthTrend[];
    top_users: BandwidthByUser[];
}

// Bitrate Analytics
export interface BitrateByResolutionItem {
    resolution: string;
    avg_bitrate: number;
    session_count: number;
    transcode_rate: number;
}

export interface BitrateTimeSeriesItem {
    timestamp: string;
    avg_bitrate: number;
    peak_bitrate: number;
    active_sessions: number;
}

export interface BitrateAnalyticsResponse {
    avg_source_bitrate: number;
    avg_transcode_bitrate: number;
    peak_bitrate: number;
    median_bitrate: number;
    bandwidth_utilization: number;
    constrained_sessions: number;
    bitrate_by_resolution: BitrateByResolutionItem[];
    bitrate_timeseries: BitrateTimeSeriesItem[];
}

// Popular Content Analytics
export interface PopularContent {
    media_type: string;
    title: string;
    parent_title?: string;
    grandparent_title?: string;
    playback_count: number;
    unique_users: number;
    avg_completion: number;
    first_played: string;
    last_played: string;
    year?: number;
    content_rating?: string;
    total_watch_time_minutes: number;
}

export interface PopularAnalyticsResponse {
    top_movies: PopularContent[];
    top_shows: PopularContent[];
    top_episodes: PopularContent[];
}

// Watch Party Analytics
export interface WatchPartyParticipant {
    user_id: number;
    username: string;
    ip_address: string;
    city?: string;
    country?: string;
    started_at: string;
    percent_complete: number;
    play_duration_minutes?: number;
}

export interface WatchParty {
    media_type: string;
    title: string;
    parent_title?: string;
    grandparent_title?: string;
    party_time: string;
    participant_count: number;
    participants: WatchPartyParticipant[];
    same_location: boolean;
    location_name?: string;
    avg_completion: number;
    total_duration_minutes: number;
}

export interface WatchPartyContentStats {
    media_type: string;
    title: string;
    parent_title?: string;
    grandparent_title?: string;
    party_count: number;
    total_participants: number;
    avg_participants_per_party: number;
    unique_users: number;
}

export interface WatchPartyUserStats {
    user_id: number;
    username: string;
    party_count: number;
    total_co_watchers: number;
    avg_party_size: number;
    favorite_content: string;
    same_location_parties: number;
}

export interface WatchPartyByDay {
    day_of_week: number;
    party_count: number;
    avg_participants: number;
}

export interface WatchPartyAnalyticsResponse {
    total_watch_parties: number;
    total_participants: number;
    avg_participants_per_party: number;
    same_location_parties: number;
    top_content: WatchPartyContentStats[];
    top_social_users: WatchPartyUserStats[];
    recent_watch_parties: WatchParty[];
    parties_by_day: WatchPartyByDay[];
}

// User Engagement Analytics
export interface UserEngagement {
    user_id: number;
    username: string;
    total_watch_time_minutes: number;
    total_sessions: number;
    average_session_minutes: number;
    first_seen_at: string;
    last_seen_at: string;
    days_since_first_seen: number;
    days_since_last_seen: number;
    total_content_items: number;
    unique_content_items: number;
    most_watched_type?: string;
    most_watched_title?: string;
    activity_score: number;
    avg_completion_rate: number;
    fully_watched_count: number;
    return_visitor_rate: number;
    unique_locations: number;
    unique_platforms: number;
}

export interface ViewingPatternByHour {
    hour_of_day: number;
    session_count: number;
    watch_time_minutes: number;
    unique_users: number;
    avg_completion: number;
}

export interface ViewingPatternByDay {
    day_of_week: number;
    day_name: string;
    session_count: number;
    watch_time_minutes: number;
    unique_users: number;
    avg_completion: number;
}

export interface UserEngagementSummary {
    total_users: number;
    active_users: number;
    total_watch_time_minutes: number;
    total_sessions: number;
    avg_session_minutes: number;
    avg_user_watch_time_minutes: number;
    avg_completion_rate: number;
    return_visitor_rate: number;
}

export interface UserEngagementAnalyticsResponse {
    summary: UserEngagementSummary;
    top_users: UserEngagement[];
    viewing_patterns_by_hour: ViewingPatternByHour[];
    viewing_patterns_by_day: ViewingPatternByDay[];
    most_active_hour?: number;
    most_active_day?: number;
}

// Comparative Analytics
export interface PeriodMetrics {
    start_date: string;
    end_date: string;
    playback_count: number;
    unique_users: number;
    watch_time_minutes: number;
    avg_session_mins: number;
    avg_completion: number;
    unique_content: number;
    unique_locations: number;
}

export interface ComparativeMetrics {
    metric: string;
    current_value: number;
    previous_value: number;
    absolute_change: number;
    percentage_change: number;
    growth_direction: string;
    is_improvement: boolean;
    current_value_formatted?: string;
    previous_value_formatted?: string;
}

export interface TopContentComparison {
    title: string;
    current_rank: number;
    previous_rank: number;
    rank_change: number;
    current_count: number;
    previous_count: number;
    count_change: number;
    count_change_pct: number;
    trending: string;
}

export interface TopUserComparison {
    username: string;
    current_rank: number;
    previous_rank: number;
    rank_change: number;
    current_watch_time: number;
    previous_watch_time: number;
    watch_time_change: number;
    watch_time_change_pct: number;
    trending: string;
}

export interface ComparativeAnalyticsResponse {
    current_period: PeriodMetrics;
    previous_period: PeriodMetrics;
    comparison_type: string;
    metrics_comparison: ComparativeMetrics[];
    top_content_comparison: TopContentComparison[];
    top_user_comparison: TopUserComparison[];
    overall_trend: string;
    key_insights: string[];
}

// Temporal Heatmap
export interface TemporalHeatmapPoint {
    latitude: number;
    longitude: number;
    weight: number;
}

export interface TemporalHeatmapBucket {
    start_time: string;
    end_time: string;
    label: string;
    points: TemporalHeatmapPoint[];
    count: number;
}

export interface TemporalHeatmapResponse {
    interval: string;
    buckets: TemporalHeatmapBucket[];
    total_count: number;
    start_date: string;
    end_date: string;
}

// Approximate Analytics (DataSketches) - ADR-0018
// Uses HyperLogLog for distinct counts and KLL sketches for percentiles
export interface ApproximateStats {
    /** Approximate unique user count (HyperLogLog) */
    unique_users: number;
    /** Approximate unique title count (HyperLogLog) */
    unique_titles: number;
    /** Approximate unique IP count (HyperLogLog) */
    unique_ips: number;
    /** Approximate unique platform count (HyperLogLog) */
    unique_platforms: number;
    /** Approximate unique player count (HyperLogLog) */
    unique_players: number;
    /** Median watch time in minutes (KLL 50th percentile) */
    watch_time_p50: number;
    /** 75th percentile watch time */
    watch_time_p75: number;
    /** 90th percentile watch time */
    watch_time_p90: number;
    /** 95th percentile watch time */
    watch_time_p95: number;
    /** 99th percentile watch time */
    watch_time_p99: number;
    /** Total playback count (exact) */
    total_playbacks: number;
    /** True if DataSketches extension was used */
    is_approximate: boolean;
    /** Estimated error bound for HLL (typically <2%) */
    error_bound?: number;
    /** Query execution time in milliseconds */
    query_time_ms: number;
}

export interface ApproximateStatsResponse {
    stats: ApproximateStats;
    /** True if DataSketches extension is available */
    datasketches_available: boolean;
}

export interface ApproximateDistinctResponse {
    /** Column that was queried */
    column: string;
    /** Approximate distinct count */
    count: number;
    /** True if DataSketches was used */
    is_approximate: boolean;
}

export interface ApproximatePercentileResponse {
    /** Column that was queried */
    column: string;
    /** Percentile value (0-1) */
    percentile: number;
    /** Calculated percentile value */
    value: number;
    /** True if DataSketches was used */
    is_approximate: boolean;
}

export interface ApproximateStatsFilter {
    start_date?: string;
    end_date?: string;
    users?: string[];
    media_types?: string[];
}

// ============================================================================
// Advanced Chart Types (Sankey, Chord, Radar, Treemap)
// ============================================================================

// Content Flow Analytics (Sankey Diagram)

/** Node in a Sankey diagram */
export interface SankeyNode {
    /** Unique identifier */
    id: string;
    /** Display name */
    name: string;
    /** Depth level (0=Show, 1=Season, 2=Episode) */
    depth: number;
    /** Total watch count through this node */
    value: number;
}

/** Link between two Sankey nodes */
export interface SankeyLink {
    /** Source node ID */
    source: string;
    /** Target node ID */
    target: string;
    /** Flow volume (watch count) */
    value: number;
}

/** A single viewing journey path */
export interface ContentFlowJourney {
    showTitle: string;
    seasonNumber: number;
    episodeNumber: number;
    episodeTitle: string;
    watchCount: number;
    /** Average completion rate (0-100) */
    avgCompletion: number;
}

/** Response for content flow Sankey diagram */
export interface ContentFlowAnalytics {
    nodes: SankeyNode[];
    links: SankeyLink[];
    /** Top viewing paths */
    journeys: ContentFlowJourney[];
    totalShows: number;
    /** Total viewing journeys */
    totalFlows: number;
    /** Percentage who start but don't finish */
    dropOffRate: number;
}

// User Content Overlap Analytics (Chord Diagram)

/** Content overlap between two users */
export interface UserOverlapPair {
    user1Id: string;
    user1Name: string;
    user2Id: string;
    user2Name: string;
    /** Number of items both watched */
    sharedItems: number;
    /** Jaccard similarity (0-1) */
    overlapPercent: number;
    /** Total shared watch time in seconds */
    sharedWatchTime: number;
    /** Most common shared genre */
    topSharedGenre: string;
}

/** User overlap matrix for chord diagram */
export interface UserOverlapMatrix {
    /** User names (matrix row/column labels) */
    users: string[];
    /** NxN overlap values (0-1 normalized) */
    matrix: number[][];
}

/** Response for user content overlap chord diagram */
export interface UserContentOverlapAnalytics {
    matrix: UserOverlapMatrix;
    /** Top overlapping user pairs */
    topPairs: UserOverlapPair[];
    /** Average overlap across all pairs */
    avgOverlap: number;
    totalUsers: number;
    /** Non-zero matrix cells */
    totalConnections: number;
    /** Detected viewing clusters */
    clusterCount: number;
}

// User Profile Analytics (Radar Chart)

/** User's score on radar chart axes */
export interface UserProfileScore {
    userId: string;
    username: string;
    /** Values 0-100 for each axis */
    scores: number[];
    /** Overall engagement rank */
    rank: number;
}

/** Response for user profile radar charts */
export interface RadarUserProfileAnalytics {
    /** Dimension labels (Watch Time, Completion, Diversity, Quality, Discovery, Social) */
    axes: string[];
    /** User scores */
    profiles: UserProfileScore[];
    /** Average scores across all users */
    averageProfile: number[];
    /** Users with highest overall scores */
    topPerformers: UserProfileScore[];
    totalUsers: number;
}

// Library Utilization Analytics (Treemap)

/** Additional metrics for treemap nodes */
export interface TreemapMetrics {
    playCount: number;
    uniqueViewers: number;
    avgCompletion: number;
    /** Media duration in seconds */
    totalDuration: number;
    /** Watched/Total duration percentage */
    utilizationPct: number;
}

/** Node in hierarchical treemap */
export interface TreemapNode {
    id: string;
    name: string;
    /** Watch time in seconds or play count */
    value: number;
    children?: TreemapNode[];
    /** "library", "section", "genre", "content" */
    itemType: string;
    metrics?: TreemapMetrics;
}

/** Response for library utilization treemap */
export interface LibraryUtilizationAnalytics {
    /** Hierarchical data */
    root: TreemapNode;
    /** Total watch time in seconds */
    totalWatchTime: number;
    /** Total content items */
    totalContent: number;
    /** Content with at least 1 play */
    utilizedContent: number;
    /** Percentage of content watched */
    utilizationRate: number;
    /** Content with zero plays */
    topUnwatched: string[];
    /** Most-watched hierarchy path */
    mostPopularPath: string;
}

// Calendar Heatmap Analytics

/** Activity for a single day */
export interface CalendarDayActivity {
    /** ISO date (YYYY-MM-DD) */
    date: string;
    /** Seconds watched */
    watchTime: number;
    /** Number of plays */
    playCount: number;
    /** 0-1 normalized activity level */
    intensity: number;
}

/** Response for calendar heatmap */
export interface CalendarHeatmapAnalytics {
    /** 365 days of activity */
    days: CalendarDayActivity[];
    totalWatchTime: number;
    totalPlayCount: number;
    /** Max watch time in a day */
    maxDaily: number;
    /** Average daily watch time */
    avgDaily: number;
    /** Days with any activity */
    activeDays: number;
    /** Consecutive active days */
    longestStreak: number;
    /** Current streak */
    currentStreak: number;
}

// Bump Chart Analytics (Ranking Changes)

/** Content's rank at a point in time */
export interface RankingEntry {
    contentId: string;
    contentTitle: string;
    /** 1-based rank */
    rank: number;
    playCount: number;
    /** Week/Month identifier */
    period: string;
}

/** Content with significant rank changes */
export interface ContentMover {
    contentId: string;
    contentTitle: string;
    startRank: number;
    endRank: number;
    /** Positive = improved */
    rankChange: number;
}

/** Response for ranking bump chart */
export interface BumpChartAnalytics {
    /** Time period labels */
    periods: string[];
    /** Rankings per period */
    rankings: RankingEntry[][];
    /** Biggest rank changes */
    topMovers: ContentMover[];
    /** Content that entered top 10 */
    newEntries: RankingEntry[];
    /** Content that exited top 10 */
    exits: RankingEntry[];
    totalPeriods: number;
}
