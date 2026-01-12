// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Types for map visualization features (H3 hexagons, arcs, server info)
 */

// H3 Hexagon aggregation for spatial analytics
export interface H3HexagonStats {
    h3_index: number;
    latitude: number;
    longitude: number;
    playback_count: number;
    unique_users: number;
    avg_completion: number;
    total_watch_minutes: number;
}

// Arc visualization for server-to-user connections
export interface ArcStats {
    user_latitude: number;
    user_longitude: number;
    server_latitude: number;
    server_longitude: number;
    city: string;
    country: string;
    distance_km: number;
    playback_count: number;
    unique_users: number;
    avg_completion: number;
    weight: number;
}

// Server location info for visualization
export interface ServerInfo {
    latitude: number;
    longitude: number;
    has_location: boolean;
}
