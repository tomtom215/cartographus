// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * API Mocks for E2E Tests (OPTIONAL - NOT ENABLED BY DEFAULT)
 *
 * E2E tests now use REAL Tautulli integration by default:
 * - A real Tautulli container with 550+ seeded playback sessions
 * - Cartographus syncs from Tautulli (production-representative)
 * - Tests run against actual API responses from synced data
 *
 * These mocks are available for tests that need DETERMINISTIC data:
 * - Testing specific data scenarios (empty states, error states)
 * - Tests that assert on specific data values
 * - Snapshot/visual regression tests
 *
 * To enable mocks in a test file, use:
 *   import { testWithMockApi } from '../fixtures';
 *   testWithMockApi('my test', async ({ page }) => { ... });
 *
 * Or enable for a specific test:
 *   test.use({ autoMockApi: true });
 *
 * This file provides deterministic data for:
 * - Stats panel (Total Playbacks, Locations, Users, Recent)
 * - Charts (all 47 ECharts across 6 analytics pages)
 * - Map visualizations (markers, clusters, heatmap)
 * - Globe visualizations (3D data points)
 * - Activity feeds and server info
 *
 * ROOT CAUSE FIX (2025-01-xx):
 * - Previous implementation used 100+ sequential `await page.route()` calls
 * - Routes registered EARLIER in the sequence would fail with net::ERR_FAILED
 * - Routes registered LATER would succeed
 * - FIX: Use a SINGLE consolidated route handler that dispatches internally
 * - This eliminates race conditions in sequential route registration
 */

import { Page, Route } from '@playwright/test';

// ============================================================================
// Mock Data Generation Utilities
// ============================================================================

/**
 * Seeded random number generator for deterministic test data
 * Uses a simple Linear Congruential Generator (LCG) algorithm
 */
let seed = 12345; // Fixed seed for deterministic results

function resetSeed(): void {
    seed = 12345;
}

function seededRandom(): number {
    seed = (seed * 1103515245 + 12345) & 0x7fffffff;
    return seed / 0x7fffffff;
}

/**
 * Generate a deterministic integer between min and max (inclusive)
 */
function randomInt(min: number, max: number): number {
    return Math.floor(seededRandom() * (max - min + 1)) + min;
}

/**
 * Generate a deterministic date within the last N days
 * Uses a fixed base date for reproducibility
 */
function randomDateWithinDays(days: number): string {
    // Use a fixed reference date for deterministic results
    const baseDate = new Date('2025-01-15T12:00:00Z');
    const past = new Date(baseDate.getTime() - days * 24 * 60 * 60 * 1000);
    const randomTime = past.getTime() + seededRandom() * (baseDate.getTime() - past.getTime());
    return new Date(randomTime).toISOString();
}

/**
 * Generate a deterministic UUID based on index
 */
let uuidCounter = 0;
function generateUUID(): string {
    uuidCounter++;
    const hex = uuidCounter.toString(16).padStart(8, '0');
    return `${hex}-0000-4000-8000-000000000000`;
}

// ============================================================================
// Mock Data Constants
// ============================================================================

const MOCK_USERS = [
    'JohnDoe', 'JaneSmith', 'MovieBuff', 'TVFanatic', 'StreamKing',
    'BingeWatcher', 'CinemaLover', 'SeriesAddict', 'FilmEnthusiast', 'ViewerOne'
];

const MOCK_MEDIA_TYPES = ['movie', 'episode', 'track'];

const MOCK_PLATFORMS = ['Roku', 'Apple TV', 'Chrome', 'Firefox', 'Safari', 'Android TV', 'iOS', 'Plex Web'];

const MOCK_PLAYERS = ['Roku Express', 'Apple TV 4K', 'Plex for Windows', 'Plex for Mac', 'Chrome Browser', 'Firefox Browser'];

const MOCK_COUNTRIES = [
    { code: 'US', country: 'United States', playback_count: 450, unique_users: 85 },
    { code: 'GB', country: 'United Kingdom', playback_count: 120, unique_users: 32 },
    { code: 'CA', country: 'Canada', playback_count: 95, unique_users: 24 },
    { code: 'DE', country: 'Germany', playback_count: 80, unique_users: 18 },
    { code: 'FR', country: 'France', playback_count: 65, unique_users: 15 },
    { code: 'AU', country: 'Australia', playback_count: 55, unique_users: 12 },
    { code: 'JP', country: 'Japan', playback_count: 40, unique_users: 9 },
    { code: 'NL', country: 'Netherlands', playback_count: 35, unique_users: 8 },
    { code: 'SE', country: 'Sweden', playback_count: 30, unique_users: 7 },
    { code: 'BR', country: 'Brazil', playback_count: 25, unique_users: 6 }
];

const MOCK_CITIES = [
    { name: 'New York', country: 'US', lat: 40.7128, lon: -74.006, count: 85 },
    { name: 'Los Angeles', country: 'US', lat: 34.0522, lon: -118.2437, count: 75 },
    { name: 'London', country: 'GB', lat: 51.5074, lon: -0.1278, count: 65 },
    { name: 'Chicago', country: 'US', lat: 41.8781, lon: -87.6298, count: 55 },
    { name: 'Toronto', country: 'CA', lat: 43.6532, lon: -79.3832, count: 45 },
    { name: 'Sydney', country: 'AU', lat: -33.8688, lon: 151.2093, count: 40 },
    { name: 'Berlin', country: 'DE', lat: 52.52, lon: 13.405, count: 35 },
    { name: 'Paris', country: 'FR', lat: 48.8566, lon: 2.3522, count: 30 },
    { name: 'Tokyo', country: 'JP', lat: 35.6762, lon: 139.6503, count: 25 },
    { name: 'San Francisco', country: 'US', lat: 37.7749, lon: -122.4194, count: 22 }
];

const MOCK_MOVIES = [
    'The Shawshank Redemption', 'The Dark Knight', 'Inception', 'Pulp Fiction',
    'The Matrix', 'Forrest Gump', 'Interstellar', 'The Godfather',
    'Fight Club', 'The Lord of the Rings'
];

const MOCK_TV_SHOWS = [
    'Breaking Bad', 'Game of Thrones', 'The Office', 'Friends',
    'Stranger Things', 'The Crown', 'The Mandalorian', 'Succession',
    'Better Call Saul', 'The Wire'
];

// ============================================================================
// Mock Data Generators
// ============================================================================

/**
 * Generate mock playback events
 */
function generateMockPlaybacks(count: number): any[] {
    const events = [];
    for (let i = 0; i < count; i++) {
        const mediaType = MOCK_MEDIA_TYPES[randomInt(0, MOCK_MEDIA_TYPES.length - 1)];
        const city = MOCK_CITIES[randomInt(0, MOCK_CITIES.length - 1)];
        const startedAt = randomDateWithinDays(90);
        const duration = randomInt(300000, 7200000); // 5 min to 2 hours

        events.push({
            id: generateUUID(),
            session_key: `session-${i}-${Date.now()}`,
            user: MOCK_USERS[randomInt(0, MOCK_USERS.length - 1)],
            media_type: mediaType,
            title: mediaType === 'movie'
                ? MOCK_MOVIES[randomInt(0, MOCK_MOVIES.length - 1)]
                : MOCK_TV_SHOWS[randomInt(0, MOCK_TV_SHOWS.length - 1)],
            started_at: startedAt,
            stopped_at: new Date(new Date(startedAt).getTime() + duration).toISOString(),
            duration: duration,
            view_offset: randomInt(0, duration),
            percent_complete: randomInt(10, 100),
            platform: MOCK_PLATFORMS[randomInt(0, MOCK_PLATFORMS.length - 1)],
            player: MOCK_PLAYERS[randomInt(0, MOCK_PLAYERS.length - 1)],
            ip_address: `192.168.1.${randomInt(1, 254)}`,
            latitude: city.lat + (seededRandom() - 0.5) * 0.5,
            longitude: city.lon + (seededRandom() - 0.5) * 0.5,
            city: city.name,
            country: city.country,
            transcode_decision: seededRandom() > 0.5 ? 'transcode' : 'direct play',
            video_resolution: ['4k', '1080p', '720p', '480p'][randomInt(0, 3)],
            video_codec: ['h264', 'hevc', 'vp9'][randomInt(0, 2)],
            audio_codec: ['aac', 'ac3', 'dts'][randomInt(0, 2)],
            bandwidth: randomInt(1000, 50000)
        });
    }
    return events.sort((a, b) => new Date(b.started_at).getTime() - new Date(a.started_at).getTime());
}

/**
 * Generate mock stats
 */
function generateMockStats() {
    return {
        total_playbacks: randomInt(1000, 5000),
        unique_locations: randomInt(50, 200),
        unique_users: randomInt(10, 50),
        recent_24h: randomInt(10, 100),
        total_watch_time: randomInt(1000000, 50000000),
        avg_watch_time: randomInt(30, 120)
    };
}

/**
 * Generate mock analytics data for charts
 */
function generateMockAnalytics() {
    const days = 30;
    const playback_trends = [];
    const now = new Date();

    for (let i = days - 1; i >= 0; i--) {
        const date = new Date(now.getTime() - i * 24 * 60 * 60 * 1000);
        playback_trends.push({
            date: date.toISOString().split('T')[0],
            playback_count: randomInt(10, 100),
            unique_users: randomInt(3, 20)
        });
    }

    return {
        playback_trends,
        media_distribution: [
            { type: 'movie', count: randomInt(200, 600) },
            { type: 'episode', count: randomInt(400, 800) },
            { type: 'track', count: randomInt(50, 200) }
        ],
        top_users: MOCK_USERS.slice(0, 10).map((user, i) => ({
            user,
            count: randomInt(50 - i * 5, 100 - i * 5),
            watch_time: randomInt(100000, 1000000)
        })),
        platforms: MOCK_PLATFORMS.map(platform => ({
            platform,
            count: randomInt(20, 150)
        })),
        players: MOCK_PLAYERS.map(player => ({
            player,
            count: randomInt(20, 150)
        })),
        countries: MOCK_COUNTRIES,
        cities: MOCK_CITIES,
        heatmap: generateHeatmapData(),
        transcode_stats: {
            direct_play: randomInt(300, 700),
            transcode: randomInt(200, 500),
            copy: randomInt(50, 150)
        },
        resolution_stats: [
            { resolution: '4K', count: randomInt(50, 200) },
            { resolution: '1080p', count: randomInt(200, 500) },
            { resolution: '720p', count: randomInt(100, 300) },
            { resolution: '480p', count: randomInt(20, 100) }
        ],
        libraries: [
            { name: 'Movies', count: randomInt(200, 600) },
            { name: 'TV Shows', count: randomInt(400, 800) },
            { name: 'Music', count: randomInt(50, 200) }
        ],
        top_content: {
            movies: MOCK_MOVIES.slice(0, 5).map((title, i) => ({
                title,
                count: randomInt(20 - i * 3, 50 - i * 3)
            })),
            shows: MOCK_TV_SHOWS.slice(0, 5).map((title, i) => ({
                title,
                count: randomInt(30 - i * 4, 60 - i * 4)
            }))
        }
    };
}

/**
 * Generate heatmap data (hour x day matrix)
 */
function generateHeatmapData(): number[][] {
    const heatmap: number[][] = [];
    for (let day = 0; day < 7; day++) {
        const dayData: number[] = [];
        for (let hour = 0; hour < 24; hour++) {
            // More activity in evening hours
            const baseActivity = hour >= 18 && hour <= 23 ? 15 : 5;
            dayData.push(randomInt(0, baseActivity));
        }
        heatmap.push(dayData);
    }
    return heatmap;
}

/**
 * Generate mock locations for map
 */
function generateMockLocations(count: number): any[] {
    const locations = [];
    for (let i = 0; i < count; i++) {
        const city = MOCK_CITIES[randomInt(0, MOCK_CITIES.length - 1)];
        locations.push({
            id: generateUUID(),
            latitude: city.lat + (seededRandom() - 0.5) * 2,
            longitude: city.lon + (seededRandom() - 0.5) * 2,
            city: city.name,
            country: city.country,
            playback_count: randomInt(1, 50),
            last_activity: randomDateWithinDays(30)
        });
    }
    return locations;
}

/**
 * Generate mock activity sessions
 */
function generateMockActivitySessions(count: number): any[] {
    const sessions = [];
    for (let i = 0; i < count; i++) {
        const city = MOCK_CITIES[randomInt(0, MOCK_CITIES.length - 1)];
        const mediaType = MOCK_MEDIA_TYPES[randomInt(0, 1)]; // movie or episode only

        sessions.push({
            session_key: `active-session-${i}`,
            user: MOCK_USERS[randomInt(0, MOCK_USERS.length - 1)],
            media_type: mediaType,
            title: mediaType === 'movie'
                ? MOCK_MOVIES[randomInt(0, MOCK_MOVIES.length - 1)]
                : MOCK_TV_SHOWS[randomInt(0, MOCK_TV_SHOWS.length - 1)],
            state: ['playing', 'paused', 'buffering'][randomInt(0, 2)],
            progress_percent: randomInt(0, 100),
            transcode_decision: seededRandom() > 0.5 ? 'transcode' : 'direct play',
            video_resolution: ['4k', '1080p', '720p'][randomInt(0, 2)],
            bandwidth: randomInt(1000, 25000),
            platform: MOCK_PLATFORMS[randomInt(0, MOCK_PLATFORMS.length - 1)],
            player: MOCK_PLAYERS[randomInt(0, MOCK_PLAYERS.length - 1)],
            location: {
                city: city.name,
                country: city.country,
                latitude: city.lat,
                longitude: city.lon
            },
            started_at: randomDateWithinDays(1),
            duration: randomInt(0, 7200000)
        });
    }
    return sessions;
}

/**
 * Generate mock server info
 */
function generateMockServerInfo() {
    return {
        name: 'Test Plex Server',
        version: '1.32.5.7349',
        platform: 'Linux',
        platform_version: 'Ubuntu 22.04 LTS',
        machine_identifier: 'test-machine-id-12345',
        online: true,
        local: true,
        update_available: false,
        libraries: [
            { id: 1, name: 'Movies', type: 'movie', count: 450 },
            { id: 2, name: 'TV Shows', type: 'show', count: 120 },
            { id: 3, name: 'Music', type: 'artist', count: 200 }
        ]
    };
}

/**
 * Generate mock recently added items
 */
function generateMockRecentlyAdded(count: number): any[] {
    const items = [];
    for (let i = 0; i < count; i++) {
        const mediaType = MOCK_MEDIA_TYPES[randomInt(0, 1)];
        items.push({
            id: generateUUID(),
            rating_key: `${randomInt(10000, 99999)}`,
            title: mediaType === 'movie'
                ? MOCK_MOVIES[randomInt(0, MOCK_MOVIES.length - 1)]
                : MOCK_TV_SHOWS[randomInt(0, MOCK_TV_SHOWS.length - 1)],
            media_type: mediaType,
            year: randomInt(1990, 2024),
            added_at: randomDateWithinDays(30),
            thumb: null,
            library_name: mediaType === 'movie' ? 'Movies' : 'TV Shows',
            duration: randomInt(1800000, 10800000)
        });
    }
    return items.sort((a, b) => new Date(b.added_at).getTime() - new Date(a.added_at).getTime());
}

/**
 * Generate mock bitrate analytics data
 */
function generateMockBitrateAnalytics() {
    return {
        avg_source_bitrate: randomInt(15000, 35000),
        avg_transcode_bitrate: randomInt(4000, 12000),
        peak_bitrate: randomInt(50000, 80000),
        median_bitrate: randomInt(10000, 25000),
        bandwidth_utilization: randomInt(30, 80),
        constrained_sessions: randomInt(0, 20),
        bitrate_by_resolution: [
            { resolution: '4K', avg_bitrate: randomInt(25000, 45000), count: randomInt(50, 200) },
            { resolution: '1080p', avg_bitrate: randomInt(8000, 15000), count: randomInt(200, 500) },
            { resolution: '720p', avg_bitrate: randomInt(3000, 6000), count: randomInt(100, 300) },
            { resolution: 'SD', avg_bitrate: randomInt(1000, 3000), count: randomInt(20, 100) }
        ],
        bitrate_timeseries: Array.from({ length: 30 }, (_, i) => {
            const date = new Date();
            date.setDate(date.getDate() - (29 - i));
            return {
                date: date.toISOString().split('T')[0],
                avg_bitrate: randomInt(8000, 20000),
                peak_bitrate: randomInt(30000, 60000),
                utilization: randomInt(20, 90)
            };
        })
    };
}

// ============================================================================
// Consolidated Route Handler
// ============================================================================

/**
 * Create a consolidated API route handler that dispatches based on URL patterns.
 *
 * ROOT CAUSE FIX: This replaces 100+ sequential `page.route()` calls with a SINGLE
 * route handler. The previous approach had race conditions where routes registered
 * earlier in the sequence would fail with net::ERR_FAILED, while later routes worked.
 *
 * By using a single handler with internal URL dispatch, we eliminate:
 * 1. Sequential route registration timing issues
 * 2. LIFO priority confusion
 * 3. Route table overflow/eviction issues
 *
 * DIAGNOSTIC FIX (2025-01-xx):
 * - Added logging INSIDE the handler to verify all requests are being intercepted
 * - Fixed regex patterns that used `$` anchors (failed with query strings)
 * - Added try-catch to prevent silent handler failures
 * - Added missing /api/v1/tautulli/library-names endpoint
 */
function createConsolidatedApiHandler(mockData: {
    stats: any;
    analytics: any;
    playbacks: any[];
    locations: any[];
    activitySessions: any[];
    serverInfo: any;
    recentlyAdded: any[];
    bitrateAnalytics: any;
}) {
    // Track interceptions for diagnostic logging
    const isCI = !!process.env.CI;
    let interceptCount = 0;
    const maxLoggedInterceptions = 50; // Increased to capture more requests

    return async (route: Route) => {
        // CRITICAL DIAGNOSTIC: Log BEFORE try block to verify handler is invoked
        // If we see this log but NOT the INTERCEPTED log, the try block is throwing
        const requestUrl = route.request().url();
        if (isCI) {
            interceptCount++;
            console.log(`[E2E-MOCK] HANDLER INVOKED #${interceptCount}: ${requestUrl}`);
        }

        // CRITICAL: Wrap entire handler in try-catch to prevent silent failures
        // An unhandled exception here would cause the request to fail with net::ERR_FAILED
        try {
            const url = requestUrl;
            const method = route.request().method();
            const fullPath = url.replace(/^https?:\/\/[^/]+/, '');

            // Extract path WITHOUT query string for pattern matching
            // ROOT CAUSE FIX: Patterns with $ anchors were failing for URLs with query params
            // e.g., /\/api\/v1\/stats$/ would NOT match /api/v1/stats?days=90
            const path = fullPath.split('?')[0];

            // Diagnostic logging INSIDE handler to verify interception is working
            if (isCI && interceptCount <= maxLoggedInterceptions) {
                console.log(`[E2E-MOCK] PROCESSING #${interceptCount}: ${method} ${path}`);
            }

            // Helper to create JSON response
            const jsonResponse = (data: any, queryTimeMs = 15) => ({
                status: 200,
                contentType: 'application/json',
                body: JSON.stringify({
                    status: 'success',
                    data,
                    metadata: { timestamp: new Date().toISOString(), query_time_ms: queryTimeMs }
                })
            });

        // ========================================================================
        // Stats endpoint
        // ========================================================================
        if (/\/api\/v1\/stats$/.test(path)) {
            return route.fulfill(jsonResponse(mockData.stats));
        }

        // ========================================================================
        // Playbacks endpoint (with filtering)
        // ========================================================================
        if (/\/api\/v1\/playbacks/.test(path)) {
            const urlObj = new URL(url);
            const limit = parseInt(urlObj.searchParams.get('limit') || '50');
            const cursor = urlObj.searchParams.get('cursor');
            const startDate = urlObj.searchParams.get('start_date');
            const endDate = urlObj.searchParams.get('end_date');
            const mediaTypes = urlObj.searchParams.get('media_types');

            let filteredPlaybacks = [...mockData.playbacks];

            if (startDate && endDate) {
                const start = new Date(startDate).getTime();
                const end = new Date(endDate).getTime();
                filteredPlaybacks = filteredPlaybacks.filter(p => {
                    const eventTime = new Date(p.started_at).getTime();
                    return eventTime >= start && eventTime <= end;
                });
            }

            if (mediaTypes) {
                const types = mediaTypes.split(',');
                filteredPlaybacks = filteredPlaybacks.filter(p => types.includes(p.media_type));
            }

            if (cursor && cursor !== 'mock-cursor-next') {
                return route.fulfill({
                    status: 400,
                    contentType: 'application/json',
                    body: JSON.stringify({
                        status: 'error',
                        error: { code: 'VALIDATION_ERROR', message: 'Invalid cursor format' }
                    })
                });
            }

            const events = filteredPlaybacks.slice(0, limit);
            const hasMore = filteredPlaybacks.length > limit;

            return route.fulfill(jsonResponse({
                events,
                pagination: { limit, has_more: hasMore, next_cursor: hasMore ? 'mock-cursor-next' : undefined }
            }, 25));
        }

        // ========================================================================
        // Locations endpoint
        // ========================================================================
        if (/\/api\/v1\/locations/.test(path)) {
            return route.fulfill(jsonResponse(mockData.locations, 20));
        }

        // ========================================================================
        // Spatial endpoints
        // ========================================================================
        if (/\/api\/v1\/spatial\/hexagons/.test(path)) {
            const mockHexagons = MOCK_CITIES.slice(0, 8).map((city, i) => ({
                h3_index: 617700169518678015 + i,
                latitude: city.lat,
                longitude: city.lon,
                playback_count: city.count * 2,
                unique_users: Math.ceil(city.count / 3),
                avg_completion: 65 + randomInt(0, 30),
                total_watch_minutes: city.count * 45
            }));
            return route.fulfill(jsonResponse(mockHexagons, 25));
        }

        if (/\/api\/v1\/spatial\/arcs/.test(path)) {
            const arcData = MOCK_CITIES.slice(0, 5).map((city, i) => ({
                source: { lat: 37.7749, lon: -122.4194 },
                target: { lat: city.lat, lon: city.lon },
                city: city.name,
                country: city.country,
                playback_count: city.count,
                bandwidth: 5000 + i * 1000
            }));
            return route.fulfill(jsonResponse(arcData, 25));
        }

        if (/\/api\/v1\/spatial/.test(path)) {
            return route.fulfill(jsonResponse({
                type: 'FeatureCollection',
                features: mockData.locations.map((loc: any) => ({
                    type: 'Feature',
                    geometry: { type: 'Point', coordinates: [loc.longitude, loc.latitude] },
                    properties: { city: loc.city, country: loc.country, playback_count: loc.playback_count }
                }))
            }, 30));
        }

        // ========================================================================
        // Analytics endpoints
        // ========================================================================
        if (/\/api\/v1\/analytics\/trends/.test(path)) {
            return route.fulfill(jsonResponse({
                playback_trends: mockData.analytics.playback_trends,
                interval: 'day'
            }, 35));
        }

        if (/\/api\/v1\/analytics\/media/.test(path)) {
            return route.fulfill(jsonResponse(mockData.analytics.media_distribution, 20));
        }

        if (/\/api\/v1\/analytics\/user-engagement/.test(path)) {
            return route.fulfill(jsonResponse(MOCK_USERS.slice(0, 10).map((user, i) => ({
                username: user,
                total_plays: 100 - i * 8,
                watch_time_hours: 50 - i * 4,
                avg_completion: 75 + randomInt(-10, 10),
                favorite_genre: ['Action', 'Drama', 'Comedy', 'Sci-Fi', 'Documentary'][i % 5],
                last_activity: new Date(Date.now() - i * 86400000).toISOString()
            })), 25));
        }

        if (/\/api\/v1\/analytics\/users/.test(path)) {
            return route.fulfill(jsonResponse(mockData.analytics.top_users, 25));
        }

        if (/\/api\/v1\/analytics\/platforms/.test(path)) {
            return route.fulfill(jsonResponse(mockData.analytics.platforms, 15));
        }

        if (/\/api\/v1\/analytics\/players/.test(path)) {
            return route.fulfill(jsonResponse(mockData.analytics.players, 15));
        }

        if (/\/api\/v1\/analytics\/geographic/.test(path)) {
            return route.fulfill(jsonResponse({
                top_countries: mockData.analytics.countries,
                top_cities: mockData.analytics.cities,
                media_type_distribution: mockData.analytics.media_distribution,
                viewing_hours_heatmap: mockData.analytics.heatmap,
                platform_distribution: mockData.analytics.platforms,
                player_distribution: mockData.analytics.players
            }, 30));
        }

        if (/\/api\/v1\/analytics\/heatmap/.test(path)) {
            return route.fulfill(jsonResponse(mockData.analytics.heatmap, 20));
        }

        if (/\/api\/v1\/analytics\/transcode/.test(path)) {
            return route.fulfill(jsonResponse(mockData.analytics.transcode_stats, 15));
        }

        if (/\/api\/v1\/analytics\/resolution-mismatch/.test(path)) {
            return route.fulfill(jsonResponse([
                { source: '4K', target: '4K', count: 80 },
                { source: '4K', target: '1080p', count: 45 },
                { source: '1080p', target: '1080p', count: 280 },
                { source: '1080p', target: '720p', count: 65 },
                { source: '720p', target: '720p', count: 95 },
                { source: '720p', target: '480p', count: 25 }
            ], 20));
        }

        if (/\/api\/v1\/analytics\/resolution/.test(path)) {
            return route.fulfill(jsonResponse(mockData.analytics.resolution_stats, 15));
        }

        if (/\/api\/v1\/analytics\/libraries/.test(path)) {
            return route.fulfill(jsonResponse(mockData.analytics.libraries, 15));
        }

        if (/\/api\/v1\/analytics\/popular/.test(path)) {
            return route.fulfill(jsonResponse(mockData.analytics.top_content, 25));
        }

        if (/\/api\/v1\/analytics\/bitrate/.test(path)) {
            return route.fulfill(jsonResponse(mockData.bitrateAnalytics, 40));
        }

        if (/\/api\/v1\/analytics\/duration/.test(path)) {
            return route.fulfill(jsonResponse({
                movie: { avg: 7200, median: 6900, min: 3600, max: 14400 },
                episode: { avg: 2700, median: 2400, min: 1200, max: 5400 },
                track: { avg: 240, median: 210, min: 120, max: 480 }
            }, 20));
        }

        if (/\/api\/v1\/analytics\/ratings/.test(path)) {
            return route.fulfill(jsonResponse([
                { rating: 'PG-13', count: 250 },
                { rating: 'R', count: 180 },
                { rating: 'PG', count: 120 },
                { rating: 'TV-MA', count: 200 },
                { rating: 'TV-14', count: 150 }
            ], 15));
        }

        if (/\/api\/v1\/analytics\/years/.test(path)) {
            return route.fulfill(jsonResponse([
                { year: 2024, count: 180 },
                { year: 2023, count: 220 },
                { year: 2022, count: 150 },
                { year: 2021, count: 130 },
                { year: 2020, count: 110 }
            ], 15));
        }

        if (/\/api\/v1\/analytics\/codec/.test(path)) {
            return route.fulfill(jsonResponse([
                { video_codec: 'h264', audio_codec: 'aac', count: 350 },
                { video_codec: 'hevc', audio_codec: 'aac', count: 200 },
                { video_codec: 'h264', audio_codec: 'ac3', count: 150 },
                { video_codec: 'hevc', audio_codec: 'eac3', count: 100 }
            ], 20));
        }

        if (/\/api\/v1\/analytics\/engagement/.test(path)) {
            return route.fulfill(jsonResponse({
                summary: { avg_session_duration: 5400, avg_completion: 72, active_users: 25 },
                by_hour: Array.from({ length: 24 }, (_, i) => ({ hour: i, count: 10 + Math.floor(i / 3) * 5 })),
                by_day: ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'].map((day, i) => ({ day, count: 80 + i * 10 }))
            }, 30));
        }

        if (/\/api\/v1\/analytics\/completion/.test(path)) {
            return route.fulfill(jsonResponse({ completed: 650, partial: 230, abandoned: 120, avg_percent: 72 }, 20));
        }

        if (/\/api\/v1\/analytics\/binge/.test(path)) {
            return route.fulfill(jsonResponse({
                summary: { total_binges: 45, avg_episodes: 4.2, peak_day: 'Saturday' },
                top_shows: MOCK_TV_SHOWS.slice(0, 5).map((show, i) => ({ title: show, binge_count: 20 - i * 3 })),
                top_users: MOCK_USERS.slice(0, 5).map((user, i) => ({ username: user, binge_count: 15 - i * 2 }))
            }, 35));
        }

        if (/\/api\/v1\/analytics\/watch-parties/.test(path)) {
            return route.fulfill(jsonResponse({
                summary: { total_parties: 28, avg_participants: 3.2, total_participants: 89 },
                top_content: MOCK_MOVIES.slice(0, 5).map((movie, i) => ({ title: movie, party_count: 8 - i })),
                top_users: MOCK_USERS.slice(0, 5).map((user, i) => ({ username: user, party_count: 12 - i * 2 }))
            }, 30));
        }

        if (/\/api\/v1\/analytics\/bandwidth/.test(path)) {
            return route.fulfill(jsonResponse({
                trends: Array.from({ length: 30 }, (_, i) => ({
                    date: new Date(Date.now() - (29 - i) * 86400000).toISOString().split('T')[0],
                    avg_bandwidth: 8000 + i * 100,
                    peak_bandwidth: 25000 + i * 200
                })),
                by_transcode: [
                    { decision: 'direct play', avg_bandwidth: 15000, count: 400 },
                    { decision: 'transcode', avg_bandwidth: 8000, count: 250 }
                ],
                by_resolution: [
                    { resolution: '4K', avg_bandwidth: 25000, count: 100 },
                    { resolution: '1080p', avg_bandwidth: 12000, count: 350 },
                    { resolution: '720p', avg_bandwidth: 5000, count: 150 }
                ],
                top_users: MOCK_USERS.slice(0, 10).map((user, i) => ({ username: user, total_bandwidth: 500000 - i * 30000 }))
            }, 45));
        }

        if (/\/api\/v1\/analytics\/hdr/.test(path)) {
            return route.fulfill(jsonResponse([
                { type: 'SDR', count: 450 },
                { type: 'HDR10', count: 180 },
                { type: 'Dolby Vision', count: 80 },
                { type: 'HDR10+', count: 40 }
            ], 15));
        }

        if (/\/api\/v1\/analytics\/audio/.test(path)) {
            return route.fulfill(jsonResponse([
                { codec: 'AAC', channels: '2.0', count: 300 },
                { codec: 'AC3', channels: '5.1', count: 220 },
                { codec: 'EAC3', channels: '5.1', count: 150 },
                { codec: 'TrueHD', channels: '7.1', count: 80 }
            ], 15));
        }

        if (/\/api\/v1\/analytics\/subtitles/.test(path)) {
            return route.fulfill(jsonResponse({
                with_subtitles: 320,
                without_subtitles: 480,
                languages: [
                    { language: 'English', count: 180 },
                    { language: 'Spanish', count: 80 },
                    { language: 'French', count: 40 }
                ]
            }, 15));
        }

        if (/\/api\/v1\/analytics\/connection-security/.test(path)) {
            return route.fulfill(jsonResponse({ secure: 720, insecure: 80, local: 450, remote: 350 }, 15));
        }

        if (/\/api\/v1\/analytics\/concurrent-streams/.test(path) || /\/api\/v1\/analytics\/concurrent/.test(path)) {
            return route.fulfill(jsonResponse({
                max_concurrent: 8,
                avg_concurrent: 3.2,
                peak_time: '21:00',
                by_hour: Array.from({ length: 24 }, (_, i) => ({ hour: i, max: 2 + Math.floor(i / 4) }))
            }, 25));
        }

        if (/\/api\/v1\/analytics\/pause-patterns/.test(path)) {
            return route.fulfill(jsonResponse({
                avg_pauses_per_session: 2.4,
                avg_pause_duration: 180,
                pause_reasons: [
                    { reason: 'user_initiated', count: 450 },
                    { reason: 'buffering', count: 80 },
                    { reason: 'network_issue', count: 30 }
                ]
            }, 20));
        }

        if (/\/api\/v1\/analytics\/comparative/.test(path)) {
            return route.fulfill(jsonResponse({
                current_period: { playbacks: 450, watch_time: 850000, users: 28 },
                previous_period: { playbacks: 380, watch_time: 720000, users: 25 },
                change: { playbacks: 18.4, watch_time: 18.1, users: 12.0 },
                top_content_current: MOCK_MOVIES.slice(0, 5).map((m, i) => ({ title: m, count: 25 - i * 3 })),
                top_content_previous: MOCK_MOVIES.slice(2, 7).map((m, i) => ({ title: m, count: 22 - i * 3 })),
                top_users_current: MOCK_USERS.slice(0, 5).map((u, i) => ({ username: u, count: 35 - i * 5 })),
                top_users_previous: MOCK_USERS.slice(1, 6).map((u, i) => ({ username: u, count: 30 - i * 4 }))
            }, 60));
        }

        if (/\/api\/v1\/analytics\/temporal-heatmap/.test(path) || /\/api\/v1\/analytics\/temporal/.test(path)) {
            return route.fulfill(jsonResponse({
                intervals: Array.from({ length: 30 }, (_, i) => ({
                    timestamp: new Date(Date.now() - (29 - i) * 86400000).toISOString(),
                    locations: MOCK_CITIES.slice(0, 5).map(city => ({
                        lat: city.lat,
                        lon: city.lon,
                        count: city.count + i * 2
                    }))
                }))
            }, 100));
        }

        if (/\/api\/v1\/analytics\/hardware-transcode/.test(path)) {
            return route.fulfill(jsonResponse({
                hw_transcode: 180,
                sw_transcode: 120,
                gpu_utilization: 45,
                gpu_type: 'NVIDIA QuickSync'
            }, 20));
        }

        if (/\/api\/v1\/analytics\/abandonment/.test(path)) {
            return route.fulfill(jsonResponse({
                total_abandoned: 120,
                abandonment_rate: 15.2,
                avg_drop_off_percent: 32,
                by_media_type: [
                    { type: 'movie', abandoned: 45, rate: 12.5 },
                    { type: 'episode', abandoned: 65, rate: 18.2 },
                    { type: 'track', abandoned: 10, rate: 8.0 }
                ],
                top_abandoned: MOCK_MOVIES.slice(0, 5).map((m, i) => ({ title: m, abandoned_count: 12 - i * 2 }))
            }, 30));
        }

        if (/\/api\/v1\/analytics\/library\//.test(path)) {
            return route.fulfill(jsonResponse({
                total_items: 450,
                watched_items: 320,
                total_playbacks: 1250,
                unique_users: 28,
                watch_time_hours: 2400,
                avg_completion: 72,
                top_content: MOCK_MOVIES.slice(0, 10).map((m, i) => ({
                    title: m,
                    playback_count: 50 - i * 4,
                    avg_completion: 70 + i * 2
                })),
                trends: Array.from({ length: 30 }, (_, i) => ({
                    date: new Date(Date.now() - (29 - i) * 86400000).toISOString().split('T')[0],
                    playbacks: 30 + i * 2,
                    watch_time: 8000 + i * 500
                }))
            }, 50));
        }

        if (/\/api\/v1\/analytics\/user\//.test(path)) {
            return route.fulfill(jsonResponse({
                username: 'JohnDoe',
                total_playbacks: 250,
                watch_time_hours: 180,
                favorite_media_type: 'movie',
                avg_completion: 78,
                first_activity: '2024-01-01T00:00:00Z',
                last_activity: '2025-01-15T12:00:00Z',
                top_content: MOCK_MOVIES.slice(0, 5).map((m, i) => ({ title: m, count: 15 - i * 2 })),
                viewing_trends: Array.from({ length: 7 }, (_, i) => ({
                    day: ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'][i],
                    count: 20 + i * 3
                })),
                device_breakdown: MOCK_PLATFORMS.slice(0, 4).map((p, i) => ({ platform: p, count: 30 - i * 5 }))
            }, 40));
        }

        // ========================================================================
        // Users and Filters endpoints
        // ========================================================================
        if (/\/api\/v1\/users/.test(path)) {
            return route.fulfill(jsonResponse(MOCK_USERS, 10));
        }

        if (/\/api\/v1\/media-types/.test(path)) {
            return route.fulfill(jsonResponse(MOCK_MEDIA_TYPES, 10));
        }

        if (/\/api\/v1\/filters/.test(path)) {
            return route.fulfill(jsonResponse({
                users: MOCK_USERS,
                media_types: MOCK_MEDIA_TYPES,
                platforms: MOCK_PLATFORMS,
                players: MOCK_PLAYERS,
                libraries: ['Movies', 'TV Shows', 'Music'],
                countries: MOCK_COUNTRIES.map(c => c.country)
            }, 10));
        }

        // ========================================================================
        // Health endpoints
        // ========================================================================
        if (/\/api\/v1\/health\/nats$/.test(path)) {
            return route.fulfill(jsonResponse({
                healthy: true,
                connected: true,
                server_id: 'mock-nats-server',
                version: '2.10.0',
                uptime_seconds: randomInt(1000, 100000)
            }));
        }

        if (/\/api\/v1\/health/.test(path)) {
            return route.fulfill(jsonResponse({
                healthy: true,
                version: '1.0.0',
                uptime: randomInt(1000, 100000)
            }));
        }

        // ========================================================================
        // Backup endpoints
        // ========================================================================
        if (/\/api\/v1\/backups$/.test(path)) {
            return route.fulfill(jsonResponse([
                {
                    id: 'backup-mock-001',
                    type: 'full',
                    filename: 'backup-2024-01-15.tar.gz',
                    size_bytes: 15728640,
                    created_at: new Date(Date.now() - 86400000).toISOString(),
                    notes: 'Automated backup',
                    database_records: 12500,
                    is_valid: true
                },
                {
                    id: 'backup-mock-002',
                    type: 'database',
                    filename: 'backup-2024-01-14.tar.gz',
                    size_bytes: 10485760,
                    created_at: new Date(Date.now() - 172800000).toISOString(),
                    notes: '',
                    database_records: 12000,
                    is_valid: true
                }
            ], 15));
        }

        if (/\/api\/v1\/backup\/stats$/.test(path)) {
            return route.fulfill(jsonResponse({
                total_backups: 5,
                total_size_bytes: 52428800,
                oldest_backup: new Date(Date.now() - 604800000).toISOString(),
                newest_backup: new Date(Date.now() - 86400000).toISOString(),
                full_backups: 3,
                database_backups: 2,
                config_backups: 0
            }, 10));
        }

        if (/\/api\/v1\/backup$/.test(path) && method === 'POST') {
            return route.fulfill(jsonResponse({
                id: 'backup-new-' + Date.now(),
                type: 'full',
                filename: 'backup-new.tar.gz',
                size_bytes: 1024000,
                created_at: new Date().toISOString(),
                is_valid: true
            }));
        }

        // ========================================================================
        // Tautulli endpoints
        // ========================================================================
        if (/\/api\/v1\/tautulli\/activity/.test(path)) {
            return route.fulfill(jsonResponse({
                sessions: mockData.activitySessions,
                stream_count: mockData.activitySessions.length,
                transcode_count: mockData.activitySessions.filter((s: any) => s.transcode_decision === 'transcode').length,
                total_bandwidth: mockData.activitySessions.reduce((sum: number, s: any) => sum + s.bandwidth, 0)
            }, 20));
        }

        if (/\/api\/v1\/tautulli\/recently-added/.test(path)) {
            const urlObj = new URL(url);
            const count = parseInt(urlObj.searchParams.get('count') || '25');
            const start = parseInt(urlObj.searchParams.get('start') || '0');
            const items = mockData.recentlyAdded.slice(start, start + count);
            return route.fulfill(jsonResponse({
                records_total: mockData.recentlyAdded.length,
                recently_added: items
            }, 15));
        }

        if (/\/api\/v1\/tautulli\/server-info/.test(path)) {
            return route.fulfill(jsonResponse(mockData.serverInfo, 10));
        }

        if (/\/api\/v1\/tautulli\/server-list/.test(path)) {
            return route.fulfill(jsonResponse([{
                name: 'Plex Media Server',
                machine_identifier: 'mock-server-id-12345',
                host: '192.168.1.100',
                port: 32400,
                ssl: false,
                is_cloud: false,
                platform: 'Linux',
                version: '1.32.0.0'
            }], 10));
        }

        if (/\/api\/v1\/tautulli\/tautulli-info/.test(path)) {
            return route.fulfill(jsonResponse({
                tautulli_version: '2.13.4',
                tautulli_branch: 'master',
                tautulli_commit: 'abc123',
                tautulli_platform: 'Linux',
                tautulli_platform_release: '5.15.0',
                tautulli_platform_version: '#1 SMP',
                tautulli_platform_linux_distro: 'Ubuntu',
                tautulli_python_version: '3.10.0'
            }, 10));
        }

        if (/\/api\/v1\/tautulli\/pms-update/.test(path)) {
            return route.fulfill(jsonResponse({
                update_available: false,
                platform: 'Linux',
                release_date: '2024-01-15',
                version: '1.32.0.0',
                requirements: '',
                extra_info: '',
                changelog_added: '',
                changelog_fixed: ''
            }, 10));
        }

        if (/\/api\/v1\/tautulli\/servers-info/.test(path)) {
            return route.fulfill(jsonResponse([{
                name: 'Plex Media Server',
                machine_identifier: 'mock-server-id-12345',
                host: '192.168.1.100',
                port: 32400,
                ssl: false,
                is_cloud: false,
                platform: 'Linux',
                version: '1.32.0.0'
            }], 10));
        }

        // ROOT CAUSE FIX: Add missing library-names endpoint
        // This endpoint was being called but not mocked, causing 501 responses
        if (/\/api\/v1\/tautulli\/library-names/.test(path)) {
            return route.fulfill(jsonResponse([
                { section_id: 1, section_name: 'Movies', section_type: 'movie' },
                { section_id: 2, section_name: 'TV Shows', section_type: 'show' },
                { section_id: 3, section_name: 'Music', section_type: 'artist' }
            ], 10));
        }

        // ========================================================================
        // Detection endpoints (ADR-0020)
        // ========================================================================
        if (/\/api\/v1\/detection\/alerts/.test(path)) {
            return route.fulfill(jsonResponse({
                alerts: [],
                total: 0,
                limit: 20,
                offset: 0
            }, 5));
        }

        if (/\/api\/v1\/detection\/stats$/.test(path)) {
            return route.fulfill(jsonResponse({
                by_severity: { critical: 0, warning: 0, info: 0 },
                by_rule_type: {
                    impossible_travel: 0,
                    concurrent_streams: 0,
                    device_velocity: 0,
                    geo_restriction: 0,
                    simultaneous_locations: 0
                },
                unacknowledged: 0,
                total: 0
            }, 3));
        }

        if (/\/api\/v1\/detection\/rules/.test(path)) {
            return route.fulfill(jsonResponse({
                rules: [
                    { id: 1, rule_type: 'impossible_travel', name: 'Impossible Travel Detection', enabled: true, config: { max_speed_kmh: 800, min_distance_km: 500, min_time_delta_minutes: 60, severity: 'warning' }, created_at: '2025-01-01T00:00:00Z', updated_at: '2025-01-01T00:00:00Z' },
                    { id: 2, rule_type: 'concurrent_streams', name: 'Concurrent Streams Detection', enabled: true, config: { default_limit: 3, severity: 'warning' }, created_at: '2025-01-01T00:00:00Z', updated_at: '2025-01-01T00:00:00Z' },
                    { id: 3, rule_type: 'device_velocity', name: 'Device Velocity Detection', enabled: true, config: { window_minutes: 60, max_unique_ips: 10, severity: 'info' }, created_at: '2025-01-01T00:00:00Z', updated_at: '2025-01-01T00:00:00Z' },
                    { id: 4, rule_type: 'geo_restriction', name: 'Geographic Restriction', enabled: false, config: { blocked_countries: [], allowed_countries: [], severity: 'critical' }, created_at: '2025-01-01T00:00:00Z', updated_at: '2025-01-01T00:00:00Z' },
                    { id: 5, rule_type: 'simultaneous_locations', name: 'Simultaneous Locations Detection', enabled: true, config: { window_minutes: 30, min_distance_km: 100, severity: 'warning' }, created_at: '2025-01-01T00:00:00Z', updated_at: '2025-01-01T00:00:00Z' }
                ]
            }, 2));
        }

        if (/\/api\/v1\/detection\/metrics$/.test(path)) {
            return route.fulfill(jsonResponse({
                events_processed: 1250,
                alerts_generated: 0,
                detection_errors: 0,
                processing_time_ms: 45,
                last_processed_at: new Date().toISOString(),
                detector_metrics: {
                    impossible_travel: { events_checked: 250, alerts_generated: 0, errors: 0, avg_processing_ms: 8 },
                    concurrent_streams: { events_checked: 250, alerts_generated: 0, errors: 0, avg_processing_ms: 5 },
                    device_velocity: { events_checked: 250, alerts_generated: 0, errors: 0, avg_processing_ms: 6 },
                    geo_restriction: { events_checked: 250, alerts_generated: 0, errors: 0, avg_processing_ms: 3 },
                    simultaneous_locations: { events_checked: 250, alerts_generated: 0, errors: 0, avg_processing_ms: 7 }
                }
            }, 8));
        }

        if (/\/api\/v1\/detection\/users\/.*\/trust/.test(path)) {
            return route.fulfill(jsonResponse({
                user_id: 1,
                username: 'testuser',
                score: 100,
                violations_count: 0,
                restricted: false,
                updated_at: new Date().toISOString()
            }, 2));
        }

        if (/\/api\/v1\/detection\/users\/low-trust/.test(path)) {
            return route.fulfill(jsonResponse({ users: [], threshold: 50 }, 3));
        }

        // ========================================================================
        // Dedupe Audit endpoints (ADR-0022)
        // ========================================================================
        if (/\/api\/v1\/dedupe\/audit\/stats$/.test(path)) {
            return route.fulfill(jsonResponse({
                total_deduped: 0,
                pending_review: 0,
                user_restored: 0,
                user_confirmed: 0,
                accuracy_rate: 100.0,
                dedupe_by_reason: {},
                dedupe_by_layer: {},
                dedupe_by_source: {},
                last_24_hours: 0,
                last_7_days: 0,
                last_30_days: 0
            }, 4));
        }

        if (/\/api\/v1\/dedupe\/audit\/.*\/confirm/.test(path)) {
            return route.fulfill(jsonResponse({ success: true }));
        }

        if (/\/api\/v1\/dedupe\/audit\/.*\/restore/.test(path)) {
            return route.fulfill(jsonResponse({
                success: true,
                message: 'Entry restored successfully',
                restored_event_id: 'restored-event-001'
            }));
        }

        if (/\/api\/v1\/dedupe\/audit\/export/.test(path)) {
            return route.fulfill({
                status: 200,
                contentType: 'text/csv',
                body: 'id,timestamp,discarded_event_id,reason,layer,status\n'
            });
        }

        if (/\/api\/v1\/dedupe\/audit/.test(path) && method === 'GET') {
            return route.fulfill(jsonResponse({
                entries: [],
                total_count: 0,
                limit: 50,
                offset: 0
            }, 5));
        }

        // ========================================================================
        // Server info and miscellaneous
        // ========================================================================
        if (/\/api\/v1\/server-info/.test(path)) {
            return route.fulfill(jsonResponse(mockData.serverInfo, 10));
        }

        if (/\/api\/v1\/insights/.test(path)) {
            return route.fulfill(jsonResponse({
                insights: [
                    { type: 'peak_usage', message: 'Peak viewing time is 8-10 PM', severity: 'info' },
                    { type: 'trending', message: 'Movie watching up 15% this week', severity: 'positive' },
                    { type: 'alert', message: 'High transcode usage detected', severity: 'warning' }
                ],
                generated_at: new Date().toISOString()
            }, 50));
        }

        // ========================================================================
        // Export endpoints
        // ========================================================================
        if (/\/api\/v1\/export\/csv/.test(path)) {
            const csvContent = 'username,title,watched_at,media_type,platform\n' +
                MOCK_USERS.slice(0, 10).map((user, i) =>
                    `${user},${MOCK_MOVIES[i % MOCK_MOVIES.length]},2025-01-15T12:00:00Z,movie,${MOCK_PLATFORMS[i % MOCK_PLATFORMS.length]}`
                ).join('\n');
            return route.fulfill({
                status: 200,
                contentType: 'text/csv',
                headers: { 'Content-Disposition': 'attachment; filename="playbacks-export.csv"' },
                body: csvContent
            });
        }

        if (/\/api\/v1\/export\/geojson/.test(path)) {
            const geojson = {
                type: 'FeatureCollection',
                features: MOCK_CITIES.map(city => ({
                    type: 'Feature',
                    geometry: { type: 'Point', coordinates: [city.lon, city.lat] },
                    properties: { city: city.name, country: city.country, playback_count: city.count }
                }))
            };
            return route.fulfill({
                status: 200,
                contentType: 'application/json',
                headers: { 'Content-Disposition': 'attachment; filename="locations-export.geojson"' },
                body: JSON.stringify(geojson)
            });
        }

        // ========================================================================
        // Auth endpoints
        // ========================================================================
        if (/\/api\/v1\/auth\/logout/.test(path)) {
            return route.fulfill(jsonResponse({ success: true }));
        }

        // ========================================================================
        // WebSocket and Sync endpoints
        // ========================================================================
        if (/\/api\/v1\/ws$/.test(path)) {
            return route.fulfill({
                status: 503,
                contentType: 'text/plain',
                body: 'WebSocket not available during E2E tests'
            });
        }

        if (/\/api\/v1\/sync/.test(path) && method === 'POST') {
            return route.fulfill(jsonResponse({
                synced: true,
                new_playbacks: 0,
                sync_duration_ms: 150
            }));
        }

        // ========================================================================
        // Catch-all for unmocked endpoints
        // ========================================================================
        if (isCI) {
            console.warn(`[E2E-MOCK] UNMOCKED ENDPOINT #${interceptCount}: ${method} ${path}`);
            console.warn(`[E2E-MOCK] Full URL: ${fullPath}`);
        }
        return route.fulfill({
            status: 501,
            contentType: 'application/json',
            body: JSON.stringify({
                status: 'error',
                error: {
                    code: 'UNMOCKED_ENDPOINT',
                    message: `E2E Test: This endpoint is not mocked. Add a handler for: ${method} ${path}`
                },
                metadata: { timestamp: new Date().toISOString() }
            })
        });

        } catch (error) {
            // CRITICAL: Catch any handler errors to prevent net::ERR_FAILED
            // Log the error and return a 500 response instead of crashing
            if (isCI) {
                console.error(`[E2E-MOCK] HANDLER ERROR #${interceptCount}:`, error);
                console.error(`[E2E-MOCK] Error occurred for URL: ${requestUrl}`);
            }
            return route.fulfill({
                status: 500,
                contentType: 'application/json',
                body: JSON.stringify({
                    status: 'error',
                    error: {
                        code: 'HANDLER_ERROR',
                        message: `E2E Mock handler error: ${error}`
                    },
                    metadata: { timestamp: new Date().toISOString() }
                })
            });
        }
    };
}

// ============================================================================
// Public API
// ============================================================================

/**
 * Set up all API route mocking for a page using a SINGLE consolidated route handler.
 *
 * ROOT CAUSE FIX: This replaces 100+ sequential `page.route()` calls with ONE route.
 * The previous approach had race conditions where routes registered earlier in the
 * sequence would fail with net::ERR_FAILED, while routes registered later worked.
 *
 * DIAGNOSTIC IMPROVEMENTS (2025-01-xx):
 * - Handler now logs INTERCEPTED requests (proves route is working)
 * - External listeners log ALL requests and failures (for comparison)
 * - Query string handling fixed (path extracted without ?params for matching)
 * - Added missing endpoints and error handling
 *
 * RACE CONDITION FIX (2025-12-31):
 * - Added route verification step to ensure routes are active before returning
 * - Added small stabilization delay to allow Playwright's routing to settle
 * - Setup listeners BEFORE registering routes to catch all events
 *
 * CRITICAL FIX (2025-12-31):
 * - Changed from glob and regex patterns to URL matcher function for route matching
 * - Both glob and regex patterns failed to intercept certain endpoints:
 *   WORKED: detection, dedupe, backup, health, tautulli/server-list
 *   FAILED: users, media-types, stats, locations, analytics, playbacks
 * - URL matcher function gives explicit control and better debugging
 * - Added URL matcher logging to diagnose which URLs are being tested
 * - Increased stabilization delay from 50ms to 200ms for CI environments
 */
export async function setupApiMocking(page: Page): Promise<void> {
    const startTime = Date.now();
    const isCI = !!process.env.CI;

    console.log(`[E2E-MOCK] setupApiMocking starting with CONSOLIDATED handler (CI: ${isCI})`);

    // Reset seed and counters for deterministic data on every test run
    resetSeed();
    uuidCounter = 0;

    // Generate all mock data upfront
    const mockData = {
        stats: generateMockStats(),
        analytics: generateMockAnalytics(),
        playbacks: generateMockPlaybacks(50),
        locations: generateMockLocations(100),
        activitySessions: generateMockActivitySessions(3),
        serverInfo: generateMockServerInfo(),
        recentlyAdded: generateMockRecentlyAdded(100),
        bitrateAnalytics: generateMockBitrateAnalytics()
    };

    // RACE CONDITION FIX: Set up diagnostic listeners BEFORE registering routes
    // This ensures we catch all requests including early ones
    let requestCount = 0;
    let failedCount = 0;
    const maxLoggedRequests = 30;

    if (isCI) {
        // Log all API requests (BEFORE interception decision)
        page.on('request', request => {
            const url = request.url();
            if (url.includes('/api/v1/')) {
                requestCount++;
                if (requestCount <= maxLoggedRequests) {
                    console.log(`[E2E-MOCK] REQUEST #${requestCount}: ${request.method()} ${url.replace(/^https?:\/\/[^/]+/, '')}`);
                }
            }
        });

        // Log failed requests - these should NOT happen if interception works
        page.on('requestfailed', request => {
            const url = request.url();
            if (url.includes('/api/v1/')) {
                failedCount++;
                const failure = request.failure();
                console.error(`[E2E-MOCK] FAILED #${failedCount}: ${request.method()} ${url.replace(/^https?:\/\/[^/]+/, '')}`);
                console.error(`[E2E-MOCK] Failure reason: ${failure?.errorText || 'unknown'}`);
                console.error(`[E2E-MOCK] This indicates the route did NOT intercept this request!`);
            }
        });

        // Log when responses are received (should show mocked responses)
        page.on('response', response => {
            const url = response.url();
            if (url.includes('/api/v1/') && requestCount <= maxLoggedRequests) {
                const status = response.status();
                // Log non-200 responses as they may indicate issues
                if (status !== 200) {
                    console.log(`[E2E-MOCK] RESPONSE: ${status} ${url.replace(/^https?:\/\/[^/]+/, '')}`);
                }
            }
        });
    }

    // Create the consolidated handler ONCE (reused for all requests)
    const consolidatedHandler = createConsolidatedApiHandler(mockData);

    // Track route invocations for diagnostics
    let matcherCalls = 0;
    let urlMatcherCalls = 0;

    // CRITICAL FIX (2025-12-31): Use URL matcher function instead of regex pattern
    // Previous attempts with both glob pattern '**/api/v1/**' and regex /\/api\/v1\//
    // failed to intercept certain endpoints:
    // - WORKED: detection/*, dedupe/*, backup/*, health/*, tautulli/server-list
    // - FAILED: users, media-types, stats, locations, analytics/*, playbacks
    //
    // The URL matcher function approach gives the most control and allows us to
    // log exactly which URLs are being tested, helping diagnose why some don't match.
    //
    // Using a function also bypasses any potential bugs in Playwright's glob or
    // regex URL matching implementations.
    const urlMatcher = (url: URL): boolean => {
        const urlString = url.href;
        const matches = urlString.includes('/api/v1/');

        // Log URL matcher invocations for first N requests to help diagnose issues
        if (isCI && urlMatcherCalls < maxLoggedRequests) {
            urlMatcherCalls++;
            const shortUrl = urlString.replace(/^https?:\/\/[^/]+/, '');
            if (matches) {
                console.log(`[E2E-MOCK] URL MATCHER #${urlMatcherCalls}: ${shortUrl} -> MATCH`);
            } else {
                // Log non-API requests that were tested but didn't match (helps verify matcher is called)
                if (urlMatcherCalls <= 5) {
                    console.log(`[E2E-MOCK] URL MATCHER #${urlMatcherCalls}: ${shortUrl} -> NO MATCH (not API)`);
                }
            }
        }

        return matches;
    };

    await page.route(urlMatcher, async (route, request) => {
        // DIAGNOSTIC: Log every route invocation to verify interception
        if (isCI) {
            matcherCalls++;
            if (matcherCalls <= maxLoggedRequests) {
                console.log(`[E2E-MOCK] ROUTE MATCHED #${matcherCalls}: ${request.url()}`);
            }
        }

        // Delegate to pre-created consolidated handler
        return consolidatedHandler(route);
    });

    // RACE CONDITION FIX: Allow Playwright's routing mechanism to fully settle
    // This delay ensures the route handler is fully registered in Playwright's
    // internal routing table before any page navigation triggers requests.
    // Without this, the first few requests may escape interception.
    // Increased from 50ms to 200ms based on CI log analysis still showing timing issues.
    const stabilizationDelay = isCI ? 200 : 50;
    await new Promise(resolve => setTimeout(resolve, stabilizationDelay));

    const duration = Date.now() - startTime;
    console.log(`[E2E-MOCK] setupApiMocking completed in ${duration}ms - SINGLE consolidated route registered (URL matcher function)`);
}

/**
 * Set up API mocking with empty data (for testing empty states)
 */
export async function setupEmptyApiMocking(page: Page): Promise<void> {
    const emptyResponse = (data: any = []) => ({
        status: 'success',
        data,
        metadata: { timestamp: new Date().toISOString(), query_time_ms: 5 }
    });

    await page.route(/\/api\/v1\/stats$/, route => route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(emptyResponse({
            total_playbacks: 0,
            unique_locations: 0,
            unique_users: 0,
            recent_24h: 0
        }))
    }));

    await page.route(/\/api\/v1\/playbacks/, route => route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(emptyResponse({
            events: [],
            pagination: { limit: 50, has_more: false }
        }))
    }));

    await page.route(/\/api\/v1\/locations/, route => route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(emptyResponse([]))
    }));

    await page.route(/\/api\/v1\/analytics/, route => route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(emptyResponse({}))
    }));

    await page.route(/\/api\/v1\/activity/, route => route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(emptyResponse({
            sessions: [],
            stream_count: 0,
            transcode_count: 0,
            total_bandwidth: 0
        }))
    }));

    await page.route(/\/api\/v1\/backups$/, route => route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(emptyResponse([]))
    }));

    await page.route(/\/api\/v1\/backup\/stats$/, route => route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(emptyResponse({
            total_backups: 0,
            total_size_bytes: 0,
            oldest_backup: null,
            newest_backup: null,
            full_backups: 0,
            database_backups: 0,
            config_backups: 0
        }))
    }));

    await new Promise(resolve => setTimeout(resolve, 0));
}

/**
 * Set up API mocking with error responses (for testing error states).
 *
 * DETERMINISTIC FIX: This function sets up PAGE-level routes (not context-level)
 * that return 500 errors for all API endpoints except auth and health.
 * Tests using this must set { autoMockApi: false, autoMockTiles: false, autoLoadAuthState: false }
 * to ensure no context-level routes interfere.
 */
export async function setupErrorApiMocking(page: Page): Promise<void> {
    const isCI = !!process.env.CI;
    const errorResponse = {
        status: 'error',
        error: {
            code: 'DATABASE_ERROR',
            message: 'Failed to retrieve data'
        }
    };

    // Catch-all error handler (lowest priority due to LIFO)
    await page.route(/\/api\/v1\//, route => route.fulfill({
        status: 500,
        contentType: 'application/json',
        body: JSON.stringify(errorResponse)
    }));

    // Auth endpoints succeed (higher priority due to LIFO)
    await page.route(/\/api\/v1\/auth\/verify$/, route => route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
            status: 'success',
            data: { valid: true, username: 'testuser', expires_at: new Date(Date.now() + 86400000).toISOString() }
        })
    }));

    await page.route(/\/api\/v1\/auth\/login$/, route => route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
            status: 'success',
            data: { token: 'mock-token', username: 'testuser', expires_at: new Date(Date.now() + 86400000).toISOString() }
        })
    }));

    await page.route(/\/api\/v1\/health/, route => route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ status: 'success', data: { healthy: true } })
    }));

    // STABILIZATION FIX: Allow Playwright's route registration to fully settle
    // This is critical in CI environments where timing is less predictable.
    // Without this delay, early requests may escape interception.
    const stabilizationDelay = isCI ? 100 : 10;
    await new Promise(resolve => setTimeout(resolve, stabilizationDelay));
}
