// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * User Profile Analytics types
 */

export interface UserProfileStats {
    username: string;
    friendly_name: string;
    user_id: number;
    total_plays: number;
    total_watch_time_minutes: number;
    unique_content_count: number;
    avg_completion_rate: number;
    last_played_date: string;
    first_played_date: string;
    favorite_library: string;
    favorite_platform: string;
    is_active: boolean;
}

export interface UserActivityTrend {
    date: string;
    plays: number;
    watch_time_minutes: number;
}

export interface UserTopContent {
    title: string;
    media_type: string;
    plays: number;
    watch_time_minutes: number;
    last_played: string;
}

export interface UserPlatformUsage {
    platform: string;
    plays: number;
    watch_time_minutes: number;
    percentage: number;
}

export interface UserProfileAnalytics {
    profile: UserProfileStats;
    activity_trend: UserActivityTrend[];
    top_content: UserTopContent[];
    platform_usage: UserPlatformUsage[];
}
