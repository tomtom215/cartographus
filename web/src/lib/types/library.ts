// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Library analytics types
 */

import type { PlaybackTrend } from './core';

export interface LibraryUserStats {
    username: string;
    plays: number;
    watch_time_minutes: number;
    avg_completion: number;
}

export interface LibraryQualityStats {
    hdr_content_count: number;
    '4k_content_count': number;
    surround_sound_count: number;
    avg_bitrate_kbps: number;
}

export interface LibraryHealthMetrics {
    stale_content_count: number;
    popularity_score: number;
    engagement_score: number;
    growth_rate_percent: number;
}

export interface LibraryAnalytics {
    library_id: number;
    library_name: string;
    media_type: string;
    total_items: number;
    watched_items: number;
    unwatched_items: number;
    watched_percentage: number;
    total_playbacks: number;
    unique_users: number;
    total_watch_time_minutes: number;
    avg_completion: number;
    most_watched_item: string;
    top_users: LibraryUserStats[];
    plays_by_day: PlaybackTrend[];
    quality_distribution: LibraryQualityStats;
    content_health: LibraryHealthMetrics;
}
