// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Advanced analytics types for specialized analysis features
 */

// Resolution Mismatch Analytics interfaces
export interface ResolutionMismatch {
    source_resolution: string;
    stream_resolution: string;
    playback_count: number;
    percentage: number;
    affected_users: string[];
    common_platforms: string[];
    transcode_required: boolean;
}

export interface UserDowngradeStats {
    username: string;
    total_playbacks: number;
    downgrade_count: number;
    downgrade_rate: number;
    common_reason: string;
}

export interface PlatformMismatch {
    platform: string;
    mismatch_count: number;
    total_playbacks: number;
    mismatch_rate: number;
}

export interface ResolutionMismatchAnalytics {
    total_playbacks: number;
    mismatched_playbacks: number;
    mismatch_rate: number;
    mismatches: ResolutionMismatch[];
    direct_play_count: number;
    transcode_count: number;
    top_downgrade_users: UserDowngradeStats[];
    mismatch_by_platform: PlatformMismatch[];
}

// HDR Analytics interfaces
export interface DynamicRangeDistribution {
    dynamic_range: string;
    playback_count: number;
    percentage: number;
    unique_users: number;
}

export interface ToneMappingEvent {
    source_format: string;
    stream_format: string;
    occurrence_count: number;
    affected_users: string[];
    platforms: string[];
}

export interface HDRDeviceStats {
    platform: string;
    hdr_playbacks: number;
    sdr_playbacks: number;
    hdr_capable: boolean;
}

export interface ContentFormatStats {
    dynamic_range: string;
    unique_content: number;
    avg_completion: number;
    most_watched_title: string;
}

export interface HDRAnalytics {
    total_playbacks: number;
    format_distribution: DynamicRangeDistribution[];
    hdr_adoption_rate: number;
    tone_mapping_events: ToneMappingEvent[];
    hdr_capable_devices: HDRDeviceStats[];
    content_by_format: ContentFormatStats[];
}

// Audio Analytics interfaces
export interface AudioChannelDistribution {
    channels: string;
    layout?: string;
    playback_count: number;
    percentage: number;
    avg_completion: number;
}

export interface AudioCodecDistribution {
    codec: string;
    playback_count: number;
    percentage: number;
    is_lossless: boolean;
    avg_bitrate: number;
}

export interface AudioDownmixEvent {
    source_channels: string;
    stream_channels: string;
    occurrence_count: number;
    platforms: string[];
    users: string[];
}

export interface AudioAnalytics {
    total_playbacks: number;
    channel_distribution: AudioChannelDistribution[];
    codec_distribution: AudioCodecDistribution[];
    downmix_events: AudioDownmixEvent[];
    surround_sound_adoption: number;
    lossless_adoption: number;
    avg_bitrate: number;
    atmos_playbacks: number;
}

// Subtitle Analytics interfaces
export interface SubtitleLanguageStats {
    language: string;
    playback_count: number;
    percentage: number;
    unique_users: number;
}

export interface SubtitleCodecStats {
    codec: string;
    playback_count: number;
    percentage: number;
}

export interface UserSubtitlePreference {
    username: string;
    total_playbacks: number;
    subtitle_usage_count: number;
    subtitle_usage_rate: number;
    preferred_languages: string[];
}

export interface SubtitleAnalytics {
    total_playbacks: number;
    subtitle_usage_rate: number;
    playbacks_with_subtitles: number;
    playbacks_without_subtitles: number;
    language_distribution: SubtitleLanguageStats[];
    codec_distribution: SubtitleCodecStats[];
    user_preferences: UserSubtitlePreference[];
}

// Connection Security Analytics interfaces
export interface ConnectionRelayStats {
    count: number;
    percent: number;
    users: string[];
    reason: string;
}

export interface ConnectionLocalStats {
    count: number;
    percent: number;
}

export interface UserConnectionStats {
    username: string;
    total_streams: number;
    secure_rate: number;
    relay_rate: number;
    local_rate: number;
}

export interface PlatformConnectionStats {
    platform: string;
    secure_rate: number;
    relay_rate: number;
}

export interface ConnectionSecurityAnalytics {
    total_playbacks: number;
    secure_connections: number;
    insecure_connections: number;
    secure_percent: number;
    relayed_connections: ConnectionRelayStats;
    local_connections: ConnectionLocalStats;
    by_user: UserConnectionStats[];
    by_platform: PlatformConnectionStats[];
}

// Pause Pattern Analytics interfaces
export interface HighPauseContent {
    title: string;
    media_type: string;
    average_pauses: number;
    completion_rate: number;
    potential_quality_issue: boolean;
    playback_count: number;
}

export interface PauseDistribution {
    pause_bucket: string;
    playback_count: number;
    percentage: number;
    avg_completion: number;
}

export interface PauseTimingBucket {
    duration_percent: number;
    pause_count: number;
}

export interface UserPauseStats {
    username: string;
    avg_pauses: number;
    binge_watcher: boolean;
    total_sessions: number;
}

export interface PauseQualityMetrics {
    high_engagement_threshold_pauses: number;
    low_engagement_count: number;
    potential_issues_detected: number;
}

export interface PausePatternAnalytics {
    total_playbacks: number;
    avg_pauses_per_session: number;
    high_pause_content: HighPauseContent[];
    pause_distribution: PauseDistribution[];
    pause_timing_heatmap: PauseTimingBucket[];
    user_pause_patterns: UserPauseStats[];
    quality_indicators: PauseQualityMetrics;
}

// Concurrent Streams Analytics interfaces
export interface ConcurrentStreamsTimeBucket {
    timestamp: string;
    concurrent_count: number;
    direct_play: number;
    direct_stream: number;
    transcode: number;
}

export interface ConcurrentStreamsByType {
    transcode_decision: string;
    avg_concurrent: number;
    max_concurrent: number;
    percentage: number;
}

export interface ConcurrentStreamsByDayOfWeek {
    day_of_week: number; // 0=Sunday, 6=Saturday
    avg_concurrent: number;
    peak_concurrent: number;
}

export interface ConcurrentStreamsByHour {
    hour: number; // 0-23
    avg_concurrent: number;
    peak_concurrent: number;
}

export interface ConcurrentStreamsAnalytics {
    peak_concurrent: number;
    peak_time: string;
    avg_concurrent: number;
    total_sessions: number;
    time_series_data: ConcurrentStreamsTimeBucket[];
    by_transcode_decision: ConcurrentStreamsByType[];
    by_day_of_week: ConcurrentStreamsByDayOfWeek[];
    by_hour_of_day: ConcurrentStreamsByHour[];
    capacity_recommendation: string;
}

// Hardware transcode statistics
export interface HWCodecStats {
    codec: string;
    session_count: number;
    percentage: number;
}

export interface FullPipelineStats {
    full_hw_count: number;
    mixed_count: number;
    full_sw_count: number;
    full_hw_percentage: number;
}

export interface HardwareTranscodeStats {
    total_sessions: number;
    hw_transcode_sessions: number;
    sw_transcode_sessions: number;
    direct_play_sessions: number;
    hw_percentage: number;
    decoder_stats: HWCodecStats[];
    encoder_stats: HWCodecStats[];
    full_pipeline_stats: FullPipelineStats;
}

// Abandonment analytics
export interface AbandonmentByType {
    type: string;
    count: number;
    rate: number;
}

export interface AbandonmentByHour {
    hour: number;
    count: number;
    rate: number;
}

export interface AbandonedContent {
    title: string;
    media_type: string;
    abandon_count: number;
    avg_abandon_point: number;
}

export interface AbandonmentStats {
    total_sessions: number;
    abandoned_sessions: number;
    abandonment_rate: number;
    avg_abandonment_point: number;
    by_media_type: AbandonmentByType[];
    by_platform: AbandonmentByType[];
    by_hour: AbandonmentByHour[];
    top_abandoned_content: AbandonedContent[];
}

// Frame Rate Analytics
export interface FrameRateDistribution {
    frame_rate: string;
    playback_count: number;
    percentage: number;
    avg_completion: number;
}

export interface FrameRateConversion {
    source_fps: string;
    stream_fps: string;
    occurrence_count: number;
    affected_users: string[];
}

export interface FrameRateAnalytics {
    total_playbacks: number;
    frame_rate_distribution: FrameRateDistribution[];
    by_media_type: Record<string, FrameRateDistribution[]>;
    high_frame_rate_adoption_percent: number;
    conversion_events: FrameRateConversion[];
}

// Container Format Analytics
export interface ContainerDistribution {
    container: string;
    playback_count: number;
    percentage: number;
    direct_play_rate_percent: number;
}

export interface ContainerRemux {
    source_container: string;
    target_container: string;
    occurrence_count: number;
    platforms: string[];
}

export interface PlatformContainer {
    platform: string;
    preferred_containers: string[];
    direct_play_rate: number;
    remux_required_rate: number;
}

export interface ContainerAnalytics {
    total_playbacks: number;
    format_distribution: ContainerDistribution[];
    direct_play_rates: Record<string, number>;
    remux_events: ContainerRemux[];
    platform_compatibility: PlatformContainer[];
}

// Hardware Transcode Trends
export interface HWTranscodeTrend {
    day: string;
    total_sessions: number;
    hw_sessions: number;
    full_pipeline_sessions: number;
    hw_percentage: number;
}
