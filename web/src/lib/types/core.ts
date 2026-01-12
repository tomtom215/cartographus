// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Core types for location, playback, and statistics data
 */

export interface LocationStats {
    country: string;
    region?: string;
    city?: string;
    latitude: number;
    longitude: number;
    playback_count: number;
    unique_users: number;
    first_seen: string;
    last_seen: string;
    avg_completion: number;
}

export interface PlaybackEvent {
    id: string;
    session_key: string;
    started_at: string;
    stopped_at?: string;
    user_id: number;
    username: string;
    ip_address: string;
    media_type: string;
    title: string;
    parent_title?: string;
    grandparent_title?: string;
    platform: string;
    player: string;
    location_type: string;
    percent_complete: number;
    paused_counter: number;
    created_at: string;
}

// Cursor-based pagination types
export interface PaginationInfo {
    limit: number;
    has_more: boolean;
    next_cursor?: string;
    prev_cursor?: string;
    total_count?: number;
}

export interface PlaybacksResponse {
    events: PlaybackEvent[];
    pagination: PaginationInfo;
}

export interface Stats {
    total_playbacks: number;
    unique_locations: number;
    unique_users: number;
    recent_activity: number;
    top_countries: Array<{
        country: string;
        playback_count: number;
        unique_users: number;
    }>;
}

/**
 * Health status for data quality indicators
 */
export interface HealthStatus {
    status: 'healthy' | 'degraded';
    version: string;
    database_connected: boolean;
    tautulli_connected: boolean;
    last_sync_time: string | null;
    uptime: number;
}

export interface LocationFilter {
    start_date?: string;
    end_date?: string;
    users?: string[];
    media_types?: string[];
    platforms?: string[];
    players?: string[];
    transcode_decisions?: string[];
    video_resolutions?: string[];
    video_codecs?: string[];
    audio_codecs?: string[];
    libraries?: string[];
    content_ratings?: string[];
    years?: number[];
    location_types?: string[];
    days?: number;
    limit?: number;
}

export interface PlaybackTrend {
    date: string;
    playback_count: number;
    unique_users: number;
}

export interface TrendsResponse {
    playback_trends: PlaybackTrend[];
    interval: string;
}

/**
 * Temporal-spatial density point for animated heatmaps
 * Used by /spatial/temporal-density endpoint
 */
export interface TemporalSpatialPoint {
    time_bucket: string;           // ISO8601 time bucket (hour/day/week/month)
    h3_index: string;              // H3 hexagon index (as string for JS compatibility)
    latitude: number;              // Hexagon center latitude
    longitude: number;             // Hexagon center longitude
    playback_count: number;        // Playbacks in this time-space bucket
    unique_users: number;          // Unique users in this time-space bucket
    rolling_avg_playbacks: number; // 7-period rolling average
    cumulative_playbacks: number;  // Cumulative playbacks up to this time
}

/**
 * Bounding box filter for viewport queries
 */
export interface BoundingBox {
    west: number;   // Min longitude (-180 to 180)
    south: number;  // Min latitude (-90 to 90)
    east: number;   // Max longitude (-180 to 180)
    north: number;  // Max latitude (-90 to 90)
}

/**
 * Nearby search parameters
 */
export interface NearbySearchParams {
    lat: number;     // Center latitude
    lon: number;     // Center longitude
    radius: number;  // Radius in kilometers (1-20000, default 100)
}

/**
 * GeoJSON Feature for streaming location data
 * Follows RFC 7946 GeoJSON specification
 */
export interface GeoJSONFeature {
    type: 'Feature';
    geometry: {
        type: 'Point';
        coordinates: [number, number];  // [longitude, latitude]
    };
    properties: {
        country?: string;
        region?: string;
        city?: string;
        playback_count?: number;
        unique_users?: number;
        first_seen?: string;
        last_seen?: string;
        avg_completion?: number;
    };
}

/**
 * GeoJSON FeatureCollection response for streaming
 * Used by /stream/locations-geojson endpoint
 */
export interface StreamingGeoJSONResponse {
    type: 'FeatureCollection';
    features: GeoJSONFeature[];
    totalLoaded: number;
}
