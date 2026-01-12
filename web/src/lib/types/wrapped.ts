// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Wrapped Reports Types
 *
 * TypeScript types for the Annual Wrapped Reports feature (Spotify-style yearly analytics).
 * These types mirror the backend models in internal/models/analytics_wrapped.go
 */

/**
 * Content rank for wrapped report (movies, shows, episodes)
 */
export interface WrappedContentRank {
    rank: number;
    title: string;
    rating_key?: string;
    thumb?: string;
    watch_count: number;
    watch_time_hours: number;
    media_type: 'movie' | 'show' | 'episode';
}

/**
 * Genre rank for wrapped report
 */
export interface WrappedGenreRank {
    rank: number;
    genre: string;
    watch_count: number;
    watch_time_hours: number;
    percentage: number; // 0-100
}

/**
 * Person rank for wrapped report (actors, directors)
 */
export interface WrappedPersonRank {
    rank: number;
    name: string;
    watch_count: number;
    watch_time_hours: number;
}

/**
 * Monthly viewing statistics
 */
export interface WrappedMonthly {
    month: number; // 1-12
    month_name: string;
    playback_count: number;
    watch_time_hours: number;
    unique_content: number;
    top_content?: string;
}

/**
 * Binge session information
 */
export interface WrappedBingeInfo {
    show_name: string;
    episode_count: number;
    duration_hours: number;
    date: string;
}

/**
 * Achievement/badge in wrapped report
 */
export interface WrappedAchievement {
    id: string;
    name: string;
    description: string;
    icon: string;
    tier?: 'bronze' | 'silver' | 'gold' | 'platinum';
    earned_at?: string;
}

/**
 * Percentile rankings compared to other users
 */
export interface WrappedPercentiles {
    watch_time: number;      // 0-100
    content_count: number;   // 0-100
    binge_count: number;     // 0-100
    completion_rate: number; // 0-100
    early_adopter: number;   // 0-100
}

/**
 * Complete wrapped report for a user
 */
export interface WrappedReport {
    // Identification
    id: string;
    year: number;
    user_id: number;
    username: string;
    generated_at: string;
    share_token?: string;

    // Core Statistics
    total_watch_time_hours: number;
    total_playbacks: number;
    unique_content_count: number;
    completion_rate: number;         // 0-100
    days_active: number;
    longest_streak_days: number;
    avg_daily_watch_minutes: number;

    // Content Breakdown
    top_movies: WrappedContentRank[];
    top_shows: WrappedContentRank[];
    top_episodes?: WrappedContentRank[];
    top_genres: WrappedGenreRank[];
    top_actors?: WrappedPersonRank[];
    top_directors?: WrappedPersonRank[];

    // Viewing Patterns
    viewing_by_hour: number[];  // [24] playback counts by hour
    viewing_by_day: number[];   // [7] playback counts by day (0=Sunday)
    viewing_by_month: number[]; // [12] playback counts by month
    peak_hour: number;
    peak_day: string;
    peak_month: string;
    monthly_trends: WrappedMonthly[];

    // Binge Analysis
    binge_sessions: number;
    longest_binge?: WrappedBingeInfo;
    total_binge_hours: number;
    favorite_binge_show?: string;
    avg_binge_episodes: number;

    // Quality Metrics
    avg_bitrate_mbps: number;
    direct_play_rate: number;      // 0-100
    hdr_viewing_percent: number;   // 0-100
    '4k_viewing_percent': number;  // 0-100
    preferred_platform: string;
    preferred_player: string;

    // Discovery Metrics
    new_content_count: number;
    discovery_rate: number;
    first_watch_of_year: string;
    last_watch_of_year: string;

    // Achievements
    achievements: WrappedAchievement[];
    percentiles: WrappedPercentiles;

    // Shareable Summary
    shareable_text: string;
}

/**
 * Leaderboard entry for wrapped
 */
export interface WrappedLeaderboardEntry {
    rank: number;
    user_id: number;
    username: string;
    total_watch_time_hours: number;
    total_playbacks: number;
    unique_content: number;
    completion_rate: number;
}

/**
 * Server-wide wrapped statistics
 */
export interface WrappedServerStats {
    year: number;
    total_users: number;
    total_watch_time_hours: number;
    total_playbacks: number;
    unique_content_watched: number;
    top_movie: string;
    top_show: string;
    top_genre: string;
    peak_month: string;
    avg_completion_rate: number;
    generated_at: string;
}

/**
 * Request to generate wrapped reports
 */
export interface WrappedGenerateRequest {
    year: number;
    user_id?: number;
    force?: boolean;
}

/**
 * Response from wrapped report generation
 */
export interface WrappedGenerateResponse {
    year: number;
    reports_generated: number;
    duration_ms: number;
    generated_at: string;
}
