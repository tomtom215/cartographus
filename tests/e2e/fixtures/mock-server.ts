// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Unified Mock Server for E2E Tests
 *
 * This module provides a SINGLE source of truth for all API mocking.
 * It uses a HYBRID architecture for 100% deterministic behavior:
 *
 * ARCHITECTURE (v2 - Express Proxy):
 * 1. Express mock server runs on port 3900 (started by global-setup.ts)
 * 2. Playwright intercepts API requests at the browser context level
 * 3. Requests are PROXIED to Express server via real HTTP fetch
 * 4. Express handles concurrency properly (no CDP race conditions)
 * 5. Fallback to inline handlers if Express is unavailable
 *
 * WHY THIS WORKS:
 * The previous implementation had race conditions because Playwright's
 * route.fulfill() uses CDP (Chrome DevTools Protocol) which doesn't
 * handle high-concurrency request fulfillment reliably. By proxying
 * to a real HTTP server, we get:
 * - Proper TCP connection handling
 * - Concurrent request processing by Express
 * - Simple forwarding by Playwright (no complex inline logic)
 *
 * FALLBACK BEHAVIOR:
 * If the Express mock server is not running (e.g., running a single
 * test locally), the inline handlers are used as a fallback.
 *
 * @see tests/e2e/mock-api-server.ts - Express mock server
 * @see tests/e2e/global-setup.ts - Starts Express server
 * @see ADR-0025: Deterministic E2E Test Mocking
 */

import { BrowserContext, Route, Request } from '@playwright/test';

// ============================================================================
// Express Proxy Configuration
// ============================================================================

const MOCK_SERVER_PORT = parseInt(process.env.MOCK_SERVER_PORT || '3900', 10);
const MOCK_SERVER_URL = `http://localhost:${MOCK_SERVER_PORT}`;

// Track whether Express server is available (checked once on first request)
let expressServerAvailable: boolean | null = null;

// ============================================================================
// Fulfillment Semaphore (limits concurrent CDP route.fulfill() calls)
// ============================================================================

/**
 * ROOT CAUSE FIX for net::ERR_FAILED errors in CI.
 *
 * PROBLEM: The previous FulfillmentQueue serialized ALL route.fulfill() calls,
 * meaning request #20 had to wait for requests #1-19 to complete. With 20+
 * concurrent requests on page load, later requests would timeout waiting.
 *
 * SOLUTION: Use a semaphore that allows LIMITED concurrency (10 parallel
 * fulfillments). This prevents:
 * 1. CDP from being overwhelmed with 50+ concurrent calls
 * 2. Requests from timing out waiting in a serial queue
 *
 * The limit of 10 is chosen because:
 * - Playwright's CDP connection can handle ~10 concurrent operations reliably
 * - Most page loads make 15-25 API requests concurrently
 * - With limit=10, worst case wait is ~2 fulfillment cycles (not 25)
 *
 * UPDATE (v2): We now use TWO semaphores:
 * 1. routeHandlerSemaphore (limit=15): Limits concurrent route handler executions
 *    including proxy calls to Express. This prevents overwhelming Express and
 *    the Node.js event loop with too many concurrent operations.
 * 2. fulfillmentSemaphore (limit=10): Limits concurrent CDP route.fulfill() calls.
 *    This is the final bottleneck - CDP can only handle ~10 concurrent fulfills.
 */
class FulfillmentSemaphore {
  private running = 0;
  private readonly limit: number;
  private waitQueue: Array<() => void> = [];

  constructor(limit = 10) {
    this.limit = limit;
  }

  async run<T>(fn: () => Promise<T>): Promise<T> {
    // Wait for a slot if at limit
    if (this.running >= this.limit) {
      await new Promise<void>((resolve) => {
        this.waitQueue.push(resolve);
      });
    }

    this.running++;
    try {
      return await fn();
    } finally {
      this.running--;
      // Release next waiter
      const next = this.waitQueue.shift();
      if (next) next();
    }
  }
}

// Route handler semaphore: limits entire route handling (proxy + fulfill)
// Higher limit (15) because this includes async operations like fetch
const routeHandlerSemaphore = new FulfillmentSemaphore(15);

// Fulfillment semaphore: limits only CDP route.fulfill() calls
// Lower limit (10) because CDP is the final bottleneck
const fulfillmentSemaphore = new FulfillmentSemaphore(10);

/**
 * Proxy a request to the Express mock server.
 * Returns null if the Express server is not available.
 *
 * @param method - HTTP method
 * @param path - Request path
 * @param body - Request body (for POST/PUT/PATCH)
 * @param requestHeaders - Headers from the original request to forward (X-Mock-* headers)
 */
async function proxyToExpressServer(
  method: string,
  path: string,
  body?: string,
  requestHeaders?: Record<string, string>
): Promise<{ status: number; headers: Record<string, string>; body: string } | null> {
  // Skip proxy check if we know Express is unavailable
  if (expressServerAvailable === false) {
    return null;
  }

  try {
    // Build headers: start with Content-Type and Accept, then forward X-Mock-* headers
    const outgoingHeaders: Record<string, string> = {
      'Content-Type': 'application/json',
      'Accept': 'application/json',
    };

    // Forward all X-Mock-* headers from original request (case-insensitive)
    if (requestHeaders) {
      for (const [key, value] of Object.entries(requestHeaders)) {
        if (key.toLowerCase().startsWith('x-mock-')) {
          outgoingHeaders[key] = value;
        }
      }
    }

    const response = await fetch(`${MOCK_SERVER_URL}${path}`, {
      method,
      headers: outgoingHeaders,
      body: method !== 'GET' && method !== 'HEAD' ? body : undefined,
    });

    // Mark Express as available
    if (expressServerAvailable === null) {
      expressServerAvailable = true;
      console.log('[MOCK] Express proxy server available on port', MOCK_SERVER_PORT);
    }

    // Extract headers
    const responseHeaders: Record<string, string> = {};
    response.headers.forEach((value, key) => {
      responseHeaders[key] = value;
    });

    return {
      status: response.status,
      headers: responseHeaders,
      body: await response.text(),
    };
  } catch (error) {
    // Mark Express as unavailable on first check
    if (expressServerAvailable === null) {
      expressServerAvailable = false;
      console.warn('[MOCK] Express proxy server not available, using inline handlers');
    }
    return null;
  }
}

// ============================================================================
// Types
// ============================================================================

interface MockResponse {
  status?: number;
  contentType?: string;
  headers?: Record<string, string>;
  body: string | Buffer;
}

interface RouteHandler {
  pattern: RegExp;
  method?: string;
  handler: (route: Route, request: Request, url: URL) => Promise<MockResponse | null>;
}

// ============================================================================
// Mock Data Generation (Deterministic)
// ============================================================================

// Fixed seed for deterministic random generation
let seed = 12345;

function resetSeed(): void {
  seed = 12345;
}

function seededRandom(): number {
  seed = (seed * 1103515245 + 12345) & 0x7fffffff;
  return seed / 0x7fffffff;
}

function randomInt(min: number, max: number): number {
  return Math.floor(seededRandom() * (max - min + 1)) + min;
}

/**
 * Generate a random date within the past N days.
 *
 * FLAKINESS FIX: Use current date as base instead of hardcoded date.
 * Previously used '2025-01-15T12:00:00Z' which caused mock data to appear
 * very old when tests run in 2026+, causing chart assertions to fail.
 *
 * Note: For determinism, we cache the base date at module load time.
 * This ensures all mock data within a test run uses the same base.
 */
const mockBaseDate = new Date(); // Capture at module load, not per-call
mockBaseDate.setHours(12, 0, 0, 0); // Normalize to noon for consistency

function randomDateWithinDays(days: number): string {
  const past = new Date(mockBaseDate.getTime() - days * 24 * 60 * 60 * 1000);
  const randomTime = past.getTime() + seededRandom() * (mockBaseDate.getTime() - past.getTime());
  return new Date(randomTime).toISOString();
}

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
];

const MOCK_CITIES = [
  { name: 'New York', country: 'US', lat: 40.7128, lon: -74.006, count: 85 },
  { name: 'Los Angeles', country: 'US', lat: 34.0522, lon: -118.2437, count: 75 },
  { name: 'London', country: 'GB', lat: 51.5074, lon: -0.1278, count: 65 },
  { name: 'Chicago', country: 'US', lat: 41.8781, lon: -87.6298, count: 55 },
  { name: 'Toronto', country: 'CA', lat: 43.6532, lon: -79.3832, count: 45 },
];

const MOCK_MOVIES = [
  'The Shawshank Redemption', 'The Dark Knight', 'Inception', 'Pulp Fiction',
  'The Matrix', 'Forrest Gump', 'Interstellar', 'The Godfather',
  'Fight Club', 'The Lord of the Rings'
];

const MOCK_TV_SHOWS = [
  'Breaking Bad', 'Game of Thrones', 'The Office', 'Friends',
  'Stranger Things', 'The Crown', 'The Mandalorian', 'Succession',
];

// ============================================================================
// Mock Data Generators
// ============================================================================

function generateMockStats() {
  return {
    total_playbacks: 2847,
    unique_locations: 156,
    unique_users: 42,
    recent_activity: 67,  // Frontend expects recent_activity, not recent_24h
    recent_24h: 67,       // Keep for backwards compatibility
    total_watch_time: 15840000,
    avg_watch_time: 92
  };
}

function generateMockPlaybacks(count: number): any[] {
  const events = [];
  for (let i = 0; i < count; i++) {
    const mediaType = MOCK_MEDIA_TYPES[randomInt(0, MOCK_MEDIA_TYPES.length - 1)];
    const city = MOCK_CITIES[randomInt(0, MOCK_CITIES.length - 1)];
    const startedAt = randomDateWithinDays(90);
    const duration = randomInt(300000, 7200000);

    events.push({
      id: generateUUID(),
      session_key: `session-${i}`,
      user: MOCK_USERS[randomInt(0, MOCK_USERS.length - 1)],
      media_type: mediaType,
      title: mediaType === 'movie'
        ? MOCK_MOVIES[randomInt(0, MOCK_MOVIES.length - 1)]
        : MOCK_TV_SHOWS[randomInt(0, MOCK_TV_SHOWS.length - 1)],
      started_at: startedAt,
      stopped_at: new Date(new Date(startedAt).getTime() + duration).toISOString(),
      duration: duration,
      platform: MOCK_PLATFORMS[randomInt(0, MOCK_PLATFORMS.length - 1)],
      player: MOCK_PLAYERS[randomInt(0, MOCK_PLAYERS.length - 1)],
      latitude: city.lat + (seededRandom() - 0.5) * 0.5,
      longitude: city.lon + (seededRandom() - 0.5) * 0.5,
      city: city.name,
      country: city.country,
      transcode_decision: seededRandom() > 0.5 ? 'transcode' : 'direct play',
      video_resolution: ['4k', '1080p', '720p', '480p'][randomInt(0, 3)],
      bandwidth: randomInt(1000, 50000)
    });
  }
  return events.sort((a, b) => new Date(b.started_at).getTime() - new Date(a.started_at).getTime());
}

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

// Mock media servers data (ADR-0026)
function getMockMediaServers() {
  return [
    {
      id: '11111111-1111-4111-8111-111111111111',
      platform: 'plex',
      name: 'Main Plex Server',
      url: 'http://plex.local:32400',
      token_masked: '****...****',
      server_id: 'plex-server-001',
      enabled: true,
      source: 'env',
      realtime_enabled: true,
      webhooks_enabled: true,
      session_polling_enabled: false,
      session_polling_interval: '30s',
      status: 'connected',
      last_sync_at: new Date(Date.now() - 300000).toISOString(), // 5 min ago
      last_sync_status: 'success',
      last_error: null,
      last_error_at: null,
      created_at: new Date(Date.now() - 86400000 * 30).toISOString(),
      updated_at: new Date(Date.now() - 86400000).toISOString(),
      immutable: true
    },
    {
      id: '22222222-2222-4222-8222-222222222222',
      platform: 'jellyfin',
      name: 'Jellyfin Media Server',
      url: 'http://jellyfin.local:8096',
      token_masked: '****...****',
      server_id: 'jellyfin-server-001',
      enabled: true,
      source: 'ui',
      realtime_enabled: false,
      webhooks_enabled: false,
      session_polling_enabled: true,
      session_polling_interval: '1m',
      status: 'connected',
      last_sync_at: new Date(Date.now() - 600000).toISOString(), // 10 min ago
      last_sync_status: 'success',
      last_error: null,
      last_error_at: null,
      created_at: new Date(Date.now() - 86400000 * 7).toISOString(),
      updated_at: new Date(Date.now() - 3600000).toISOString(),
      immutable: false
    },
    {
      id: '33333333-3333-4333-8333-333333333333',
      platform: 'tautulli',
      name: 'Tautulli Analytics',
      url: 'http://tautulli.local:8181',
      token_masked: '****...****',
      server_id: 'tautulli-server-001',
      enabled: true,
      source: 'ui',
      realtime_enabled: false,
      webhooks_enabled: false,
      session_polling_enabled: false,
      session_polling_interval: '30s',
      status: 'error',
      last_sync_at: new Date(Date.now() - 3600000).toISOString(), // 1 hour ago
      last_sync_status: 'error',
      last_error: 'Connection timeout after 30 seconds',
      last_error_at: new Date(Date.now() - 1800000).toISOString(), // 30 min ago
      created_at: new Date(Date.now() - 86400000 * 14).toISOString(),
      updated_at: new Date(Date.now() - 1800000).toISOString(),
      immutable: false
    }
  ];
}

function getMockDBServers() {
  // Return only UI-added servers (source: 'ui')
  return getMockMediaServers().filter((s: any) => s.source === 'ui');
}

function generateMockAnalytics() {
  const playback_trends = [];
  for (let i = 29; i >= 0; i--) {
    const date = new Date(Date.now() - i * 24 * 60 * 60 * 1000);
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
    countries: MOCK_COUNTRIES,
    cities: MOCK_CITIES,
    // Heatmap data must be array of objects with hour, day_of_week, playback_count
    // This matches the structure expected by GeographicChartRenderer.renderHeatmap()
    heatmap: Array.from({ length: 7 }, (_, day) =>
      Array.from({ length: 24 }, (_, hour) => ({
        hour,
        day_of_week: day,
        playback_count: randomInt(0, hour >= 18 && hour <= 23 ? 15 : 5)
      }))
    ).flat(),
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
    ]
  };
}

// ============================================================================
// Response Helpers
// ============================================================================

function jsonResponse(data: any, queryTimeMs = 15): MockResponse {
  return {
    status: 200,
    contentType: 'application/json',
    body: JSON.stringify({
      status: 'success',
      data,
      metadata: { timestamp: new Date().toISOString(), query_time_ms: queryTimeMs }
    })
  };
}

function errorResponse(code: string, message: string, status = 500): MockResponse {
  return {
    status,
    contentType: 'application/json',
    body: JSON.stringify({
      status: 'error',
      error: { code, message },
      metadata: { timestamp: new Date().toISOString() }
    })
  };
}

// ============================================================================
// Route Handlers
// ============================================================================

// Pre-generate mock data once for consistency
let mockData: {
  stats: any;
  analytics: any;
  playbacks: any[];
  locations: any[];
} | null = null;

function getMockData() {
  if (!mockData) {
    resetSeed();
    uuidCounter = 0;
    mockData = {
      stats: generateMockStats(),
      analytics: generateMockAnalytics(),
      playbacks: generateMockPlaybacks(50),
      locations: generateMockLocations(100),
    };
  }
  return mockData;
}

/**
 * All API route handlers in priority order (first match wins).
 * Each handler returns null to fall through to the next handler.
 */
const API_HANDLERS: RouteHandler[] = [
  // =========================================================================
  // Auth endpoints
  // =========================================================================
  {
    pattern: /\/api\/v1\/auth\/verify$/,
    handler: async () => jsonResponse({
      valid: true,
      username: 'admin',
      // CRITICAL: role and user_id required for AuthContext to recognize admin
      role: 'admin',
      user_id: 'mock-admin-user-id',
      expires_at: new Date(Date.now() + 86400000).toISOString()
    })
  },
  {
    pattern: /\/api\/v1\/auth\/login$/,
    method: 'POST',
    handler: async () => jsonResponse({
      token: 'mock-jwt-token-for-e2e-testing',
      username: 'admin',
      // CRITICAL: role and user_id required for AuthContext.setAuth()
      // Without role: 'admin', RoleGuard hides Data Governance tab
      role: 'admin',
      user_id: 'mock-admin-user-id',
      expires_at: new Date(Date.now() + 86400000).toISOString()
    })
  },
  {
    pattern: /\/api\/v1\/auth\/logout$/,
    handler: async () => jsonResponse({ success: true })
  },

  // =========================================================================
  // Health endpoints
  // =========================================================================
  {
    pattern: /\/api\/v1\/health\/nats$/,
    handler: async () => jsonResponse({
      status: 'healthy',
      healthy: true,
      connected: true,
      jetstream_enabled: true,
      streams: 3,
      consumers: 5,
      server_id: 'mock-nats-server',
      version: '2.10.0'
    })
  },
  {
    pattern: /\/api\/v1\/health\/setup$/,
    handler: async () => jsonResponse({ setup_complete: true })
  },
  {
    // Main health endpoint - status must be lowercase 'healthy' or 'degraded' to match HealthStatus type
    // The frontend's HealthRenderer.calculateOverallHealth() determines display text
    pattern: /\/api\/v1\/health$/,
    handler: async () => jsonResponse({
      status: 'healthy',
      version: '1.0.0-mock',
      database_connected: true,
      tautulli_connected: true,
      nats_connected: true,
      wal_healthy: true,
      websocket_connected: true,
      detection_enabled: true,
      last_sync_time: new Date(Date.now() - 5 * 60 * 1000).toISOString(),
      last_backup_time: new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString(),
      last_detection_time: new Date(Date.now() - 10 * 60 * 1000).toISOString(),
      uptime: 86400,
      uptime_formatted: '1 day'
    })
  },

  // =========================================================================
  // Stats endpoint
  // =========================================================================
  {
    pattern: /\/api\/v1\/stats$/,
    handler: async () => jsonResponse(getMockData().stats)
  },

  // =========================================================================
  // Users and media types
  // =========================================================================
  {
    pattern: /\/api\/v1\/users$/,
    handler: async () => jsonResponse(MOCK_USERS)
  },
  {
    pattern: /\/api\/v1\/media-types$/,
    handler: async () => jsonResponse(MOCK_MEDIA_TYPES)
  },
  {
    pattern: /\/api\/v1\/filters$/,
    handler: async () => jsonResponse({
      users: MOCK_USERS,
      media_types: MOCK_MEDIA_TYPES,
      platforms: MOCK_PLATFORMS,
      players: MOCK_PLAYERS,
      libraries: ['Movies', 'TV Shows', 'Music'],
      countries: MOCK_COUNTRIES.map(c => c.country)
    })
  },

  // =========================================================================
  // Playbacks endpoint
  // =========================================================================
  {
    pattern: /\/api\/v1\/playbacks/,
    handler: async (_route, _request, url) => {
      const data = getMockData();
      const limit = parseInt(url.searchParams.get('limit') || '50');
      const events = data.playbacks.slice(0, limit);
      const hasMore = data.playbacks.length > limit;

      return jsonResponse({
        events,
        pagination: {
          limit,
          has_more: hasMore,
          next_cursor: hasMore ? 'mock-cursor-next' : undefined
        }
      }, 25);
    }
  },

  // =========================================================================
  // Locations endpoint
  // =========================================================================
  {
    pattern: /\/api\/v1\/locations/,
    handler: async () => jsonResponse(getMockData().locations, 20)
  },

  // =========================================================================
  // Spatial endpoints
  // =========================================================================
  {
    pattern: /\/api\/v1\/spatial\/hexagons/,
    handler: async () => {
      const hexagons = MOCK_CITIES.slice(0, 8).map((city, i) => ({
        h3_index: 617700169518678015 + i,
        latitude: city.lat,
        longitude: city.lon,
        playback_count: city.count * 2,
        unique_users: Math.ceil(city.count / 3),
        avg_completion: 65 + randomInt(0, 30)
      }));
      return jsonResponse(hexagons, 25);
    }
  },
  {
    pattern: /\/api\/v1\/spatial\/arcs/,
    handler: async () => {
      const arcs = MOCK_CITIES.slice(0, 5).map((city, i) => ({
        source: { lat: 37.7749, lon: -122.4194 },
        target: { lat: city.lat, lon: city.lon },
        city: city.name,
        country: city.country,
        playback_count: city.count,
        bandwidth: 5000 + i * 1000
      }));
      return jsonResponse(arcs, 25);
    }
  },
  {
    pattern: /\/api\/v1\/spatial/,
    handler: async () => {
      const data = getMockData();
      return jsonResponse({
        type: 'FeatureCollection',
        features: data.locations.map((loc: any) => ({
          type: 'Feature',
          geometry: { type: 'Point', coordinates: [loc.longitude, loc.latitude] },
          properties: { city: loc.city, country: loc.country, playback_count: loc.playback_count }
        }))
      }, 30);
    }
  },

  // =========================================================================
  // Analytics endpoints (must be BEFORE the catch-all)
  // =========================================================================
  {
    pattern: /\/api\/v1\/analytics\/trends/,
    handler: async () => {
      const data = getMockData();
      return jsonResponse({
        playback_trends: data.analytics.playback_trends,
        interval: 'day'
      }, 35);
    }
  },
  {
    pattern: /\/api\/v1\/analytics\/media/,
    handler: async () => jsonResponse(getMockData().analytics.media_distribution, 20)
  },
  {
    pattern: /\/api\/v1\/analytics\/user-engagement/,
    handler: async () => jsonResponse(MOCK_USERS.slice(0, 10).map((user, i) => ({
      username: user,
      total_plays: 100 - i * 8,
      watch_time_hours: 50 - i * 4,
      avg_completion: 75 + randomInt(-10, 10),
      favorite_genre: ['Action', 'Drama', 'Comedy', 'Sci-Fi', 'Documentary'][i % 5],
      last_activity: new Date(Date.now() - i * 86400000).toISOString()
    })), 25)
  },
  {
    pattern: /\/api\/v1\/analytics\/users/,
    handler: async () => jsonResponse(getMockData().analytics.top_users, 25)
  },
  {
    pattern: /\/api\/v1\/analytics\/platforms/,
    handler: async () => jsonResponse(getMockData().analytics.platforms, 15)
  },
  {
    pattern: /\/api\/v1\/analytics\/geographic/,
    handler: async () => {
      const data = getMockData();
      return jsonResponse({
        top_countries: data.analytics.countries,
        top_cities: data.analytics.cities,
        media_type_distribution: data.analytics.media_distribution,
        viewing_hours_heatmap: data.analytics.heatmap,
        platform_distribution: data.analytics.platforms
      }, 30);
    }
  },
  {
    pattern: /\/api\/v1\/analytics\/heatmap/,
    handler: async () => jsonResponse(getMockData().analytics.heatmap, 20)
  },
  {
    pattern: /\/api\/v1\/analytics\/transcode/,
    handler: async () => jsonResponse(getMockData().analytics.transcode_stats, 15)
  },
  {
    pattern: /\/api\/v1\/analytics\/resolution/,
    handler: async () => jsonResponse(getMockData().analytics.resolution_stats, 15)
  },
  {
    pattern: /\/api\/v1\/analytics\/popular/,
    handler: async () => jsonResponse({
      movies: MOCK_MOVIES.slice(0, 5).map((title, i) => ({ title, count: 50 - i * 8 })),
      shows: MOCK_TV_SHOWS.slice(0, 5).map((title, i) => ({ title, count: 60 - i * 10 }))
    }, 25)
  },
  {
    pattern: /\/api\/v1\/analytics\/bitrate/,
    handler: async () => jsonResponse({
      avg_source_bitrate: 25000,
      avg_transcode_bitrate: 8000,
      peak_bitrate: 65000,
      median_bitrate: 18000,
      // Required fields for BitrateAnalytics structure
      bandwidth_utilization: 72.5,
      constrained_sessions: 45,
      bitrate_by_resolution: [
        { resolution: '4K', avg_bitrate: 35000, count: 50 },
        { resolution: '1080p', avg_bitrate: 12000, count: 200 },
        { resolution: '720p', avg_bitrate: 5000, count: 100 },
        { resolution: '480p', avg_bitrate: 2000, count: 30 }
      ],
      bitrate_timeseries: Array.from({ length: 24 }, (_, i) => ({
        hour: i,
        avg_bitrate: 15000 + (i >= 18 && i <= 23 ? 8000 : 0),
        peak_bitrate: 45000 + (i >= 19 && i <= 22 ? 15000 : 0)
      }))
    }, 40)
  },
  {
    pattern: /\/api\/v1\/analytics\/concurrent/,
    handler: async () => jsonResponse({
      max_concurrent: 8,
      avg_concurrent: 3.2,
      peak_time: '21:00',
      by_hour: Array.from({ length: 24 }, (_, i) => ({ hour: i, max: 2 + Math.floor(i / 4) }))
    }, 25)
  },
  {
    pattern: /\/api\/v1\/analytics\/abandonment/,
    handler: async () => jsonResponse({
      total_abandoned: 120,
      abandonment_rate: 15.2,
      avg_drop_off_percent: 32
    }, 30)
  },
  {
    pattern: /\/api\/v1\/analytics\/hardware-transcode/,
    handler: async () => jsonResponse({
      total_sessions: 500,
      hw_transcode_sessions: 180,
      sw_transcode_sessions: 120,
      direct_play_sessions: 200,
      hw_percentage: 36,
      decoder_stats: [
        { codec: 'h264', session_count: 100, percentage: 55.6 },
        { codec: 'hevc', session_count: 60, percentage: 33.3 },
        { codec: 'av1', session_count: 20, percentage: 11.1 }
      ],
      encoder_stats: [
        { codec: 'h264', session_count: 120, percentage: 66.7 },
        { codec: 'hevc', session_count: 60, percentage: 33.3 }
      ],
      full_pipeline_stats: {
        full_hw_count: 150,
        mixed_count: 30,
        full_sw_count: 120,
        full_hw_percentage: 50
      }
    }, 20)
  },
  {
    pattern: /\/api\/v1\/analytics\/library\//,
    handler: async () => jsonResponse({
      total_items: 450,
      watched_items: 320,
      total_playbacks: 1250,
      unique_users: 28
    }, 50)
  },
  {
    pattern: /\/api\/v1\/analytics\/user\//,
    handler: async () => jsonResponse({
      username: 'JohnDoe',
      total_playbacks: 250,
      watch_time_hours: 180,
      favorite_media_type: 'movie'
    }, 40)
  },
  {
    pattern: /\/api\/v1\/analytics\/connection-security/,
    handler: async () => jsonResponse({
      total_playbacks: 500,
      secure_connections: 420,
      insecure_connections: 30,
      secure_percent: 84,
      relayed_connections: { count: 50, percent: 10, users: ['user1', 'user2'], reason: 'NAT traversal' },
      local_connections: { count: 150, percent: 30 },
      by_user: MOCK_USERS.slice(0, 5).map(username => ({
        username,
        total_streams: randomInt(10, 50),
        secure_rate: 80 + randomInt(0, 15),
        relay_rate: randomInt(0, 10),
        local_rate: randomInt(20, 40)
      })),
      by_platform: MOCK_PLATFORMS.slice(0, 4).map(platform => ({
        platform,
        secure_rate: 75 + randomInt(0, 20),
        relay_rate: randomInt(0, 15)
      }))
    }, 25)
  },
  {
    pattern: /\/api\/v1\/analytics\/pause-patterns/,
    handler: async () => jsonResponse({
      total_pauses: 1250,
      avg_pauses_per_play: 2.3,
      avg_pause_duration: 45,
      pause_by_hour: Array.from({ length: 24 }, (_, i) => ({ hour: i, count: randomInt(10, 100) })),
      high_pause_content: MOCK_MOVIES.slice(0, 5).map(title => ({
        title,
        media_type: 'movie',
        average_pauses: randomInt(2, 8)
      }))
    }, 20)
  },
  {
    pattern: /\/api\/v1\/analytics\/binge/,
    handler: async () => jsonResponse({
      binge_sessions: 45,
      avg_episodes_per_binge: 4.2,
      top_binged_shows: MOCK_TV_SHOWS.slice(0, 5).map((title, i) => ({
        title,
        binge_count: 20 - i * 3,
        avg_episodes: 3 + randomInt(0, 4)
      })),
      binge_by_day: ['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun'].map(day => ({
        day,
        count: randomInt(3, 15)
      }))
    }, 25)
  },
  {
    pattern: /\/api\/v1\/analytics\/watch-parties/,
    handler: async () => jsonResponse({
      total_parties: 12,
      avg_participants: 3.5,
      popular_content: MOCK_MOVIES.slice(0, 3).map(title => ({
        title,
        party_count: randomInt(2, 8)
      }))
    }, 20)
  },
  {
    pattern: /\/api\/v1\/analytics\/bandwidth/,
    handler: async () => jsonResponse({
      avg_bandwidth: 15000,
      peak_bandwidth: 45000,
      by_quality: [
        { quality: '4K', bandwidth: 25000, count: 50 },
        { quality: '1080p', bandwidth: 12000, count: 200 },
        { quality: '720p', bandwidth: 5000, count: 100 }
      ]
    }, 20)
  },
  {
    pattern: /\/api\/v1\/analytics\/engagement/,
    handler: async () => jsonResponse({
      avg_completion_rate: 72,
      avg_watch_time: 5400,
      engagement_by_type: [
        { type: 'movie', completion_rate: 68, avg_duration: 6800 },
        { type: 'episode', completion_rate: 85, avg_duration: 2400 }
      ]
    }, 20)
  },
  {
    pattern: /\/api\/v1\/analytics\/comparative/,
    handler: async () => jsonResponse({
      current_period: { playbacks: 500, users: 42, watch_time: 150000 },
      previous_period: { playbacks: 420, users: 38, watch_time: 130000 },
      changes: { playbacks_pct: 19, users_pct: 10.5, watch_time_pct: 15.4 }
    }, 30)
  },
  {
    pattern: /\/api\/v1\/analytics\/temporal-heatmap/,
    handler: async () => jsonResponse({
      data: Array.from({ length: 7 }, (_, day) =>
        Array.from({ length: 24 }, (_, hour) => ({
          day, hour, value: randomInt(0, hour >= 18 && hour <= 23 ? 20 : 8)
        }))
      ).flat()
    }, 25)
  },
  {
    pattern: /\/api\/v1\/analytics\/resolution-mismatch/,
    handler: async () => jsonResponse({
      total_mismatches: 85,
      mismatch_rate: 12.5,
      by_reason: [
        { reason: 'bandwidth', count: 45 },
        { reason: 'device_capability', count: 30 },
        { reason: 'transcoding', count: 10 }
      ]
    }, 20)
  },
  {
    pattern: /\/api\/v1\/analytics\/hdr/,
    handler: async () => jsonResponse({
      hdr_playbacks: 120,
      sdr_playbacks: 380,
      hdr_capable_users: 25,
      by_format: [
        { format: 'HDR10', count: 80 },
        { format: 'Dolby Vision', count: 35 },
        { format: 'HLG', count: 5 }
      ]
    }, 20)
  },
  {
    pattern: /\/api\/v1\/analytics\/audio/,
    handler: async () => jsonResponse({
      surround_playbacks: 200,
      stereo_playbacks: 300,
      by_codec: [
        { codec: 'TrueHD Atmos', count: 50 },
        { codec: 'DTS-HD MA', count: 80 },
        { codec: 'EAC3', count: 70 },
        { codec: 'AAC', count: 300 }
      ]
    }, 20)
  },
  {
    pattern: /\/api\/v1\/analytics\/subtitle/,
    handler: async () => jsonResponse({
      subtitle_usage_rate: 35,
      by_language: [
        { language: 'English', count: 150 },
        { language: 'Spanish', count: 30 },
        { language: 'French', count: 20 }
      ],
      forced_subtitles: 45
    }, 20)
  },
  {
    pattern: /\/api\/v1\/analytics\/frame-rate/,
    handler: async () => jsonResponse({
      by_fps: [
        { fps: 24, count: 300, completion_rate: 72 },
        { fps: 30, count: 150, completion_rate: 68 },
        { fps: 60, count: 50, completion_rate: 85 }
      ]
    }, 20)
  },
  {
    pattern: /\/api\/v1\/analytics/,
    handler: async () => jsonResponse(getMockData().analytics, 50)
  },

  // =========================================================================
  // Tautulli endpoints
  // =========================================================================
  {
    pattern: /\/api\/v1\/tautulli\/activity/,
    handler: async () => jsonResponse({
      stream_count: 2,
      stream_count_direct_play: 1,
      stream_count_direct_stream: 0,
      stream_count_transcode: 1,
      total_bandwidth: 15000,
      lan_bandwidth: 5000,
      wan_bandwidth: 10000,
      sessions: [
        {
          session_key: 'session-001',
          user_id: 1,
          username: 'testuser',
          friendly_name: 'Test User',
          title: 'Test Movie',
          parent_title: null,
          grandparent_title: null,
          media_type: 'movie',
          year: 2024,
          platform: 'Chrome',
          player: 'Plex Web',
          ip_address: '192.168.1.100',
          stream_container: 'mkv',
          stream_video_codec: 'h264',
          stream_video_resolution: '1080',
          stream_audio_codec: 'aac',
          transcode_decision: 'direct play',
          bandwidth: 8000,
          progress_percent: 45,
          state: 'playing'
        },
        {
          session_key: 'session-002',
          user_id: 2,
          username: 'otheruser',
          friendly_name: 'Other User',
          title: 'Episode 5',
          parent_title: 'Season 1',
          grandparent_title: 'Test Show',
          media_type: 'episode',
          year: 2024,
          platform: 'Android',
          player: 'Plex for Android',
          ip_address: '192.168.1.101',
          stream_container: 'mp4',
          stream_video_codec: 'hevc',
          stream_video_resolution: '4K',
          stream_audio_codec: 'eac3',
          transcode_decision: 'transcode',
          bandwidth: 7000,
          progress_percent: 72,
          state: 'paused'
        }
      ]
    }, 20)
  },
  {
    pattern: /\/api\/v1\/tautulli\/recently-added/,
    handler: async () => jsonResponse({
      records_total: 0,
      recently_added: []
    }, 15)
  },
  {
    pattern: /\/api\/v1\/tautulli\/server-info/,
    handler: async () => jsonResponse({
      name: 'Test Plex Server',
      version: '1.32.5.7349',
      platform: 'Linux',
      online: true
    }, 10)
  },
  {
    pattern: /\/api\/v1\/tautulli\/server-list/,
    handler: async () => jsonResponse([{
      name: 'Plex Media Server',
      machine_identifier: 'mock-server-id',
      host: '192.168.1.100',
      port: 32400
    }], 10)
  },
  {
    pattern: /\/api\/v1\/tautulli\/tautulli-info/,
    handler: async () => jsonResponse({
      tautulli_version: '2.13.4',
      tautulli_branch: 'master'
    }, 10)
  },
  {
    pattern: /\/api\/v1\/tautulli\/pms-update/,
    handler: async () => jsonResponse({ update_available: false }, 10)
  },
  {
    pattern: /\/api\/v1\/tautulli\/library-names/,
    handler: async () => jsonResponse([
      { section_id: 1, section_name: 'Movies', section_type: 'movie' },
      { section_id: 2, section_name: 'TV Shows', section_type: 'show' },
      { section_id: 3, section_name: 'Music', section_type: 'artist' }
    ], 10)
  },
  {
    pattern: /\/api\/v1\/tautulli\/terminate-session/,
    method: 'POST',
    handler: async () => jsonResponse({ result: 'success' }, 10)
  },

  // =========================================================================
  // Detection endpoints
  // =========================================================================
  {
    pattern: /\/api\/v1\/detection\/alerts/,
    handler: async () => jsonResponse({ alerts: [], total: 0, limit: 20, offset: 0 }, 5)
  },
  {
    pattern: /\/api\/v1\/detection\/stats$/,
    handler: async () => jsonResponse({
      by_severity: { critical: 0, warning: 0, info: 0 },
      by_rule_type: {},
      unacknowledged: 0,
      total: 0
    }, 3)
  },
  {
    pattern: /\/api\/v1\/detection\/rules/,
    handler: async () => jsonResponse({ rules: [] }, 2)
  },
  {
    pattern: /\/api\/v1\/detection\/metrics$/,
    handler: async () => jsonResponse({
      events_processed: 1250,
      alerts_generated: 0,
      detection_errors: 0
    }, 8)
  },
  {
    pattern: /\/api\/v1\/detection\/users\/.*\/trust/,
    handler: async () => jsonResponse({
      user_id: 1,
      username: 'testuser',
      score: 100,
      restricted: false
    }, 2)
  },
  {
    pattern: /\/api\/v1\/detection\/users\/low-trust/,
    handler: async () => jsonResponse({ users: [], threshold: 50 }, 3)
  },

  // =========================================================================
  // Dedupe endpoints
  // =========================================================================
  {
    pattern: /\/api\/v1\/dedupe\/audit\/stats$/,
    handler: async () => jsonResponse({
      total_deduped: 0,
      pending_review: 0,
      user_restored: 0,
      accuracy_rate: 100.0
    }, 4)
  },
  {
    pattern: /\/api\/v1\/dedupe\/audit/,
    handler: async () => jsonResponse({
      entries: [],
      total_count: 0,
      limit: 50,
      offset: 0
    }, 5)
  },

  // =========================================================================
  // WAL health endpoint (needed by Health Dashboard)
  // =========================================================================
  {
    pattern: /\/api\/v1\/wal\/health$/,
    handler: async () => jsonResponse({
      status: 'healthy',
      enabled: true,
      pending_entries: 0,
      processed_entries: 1250,
      last_checkpoint: new Date(Date.now() - 5 * 60 * 1000).toISOString(),
      last_checkpoint_entries: 50,
      storage_bytes: 10485760
    })
  },

  // =========================================================================
  // Backup endpoints
  // =========================================================================
  {
    pattern: /\/api\/v1\/backups$/,
    handler: async () => jsonResponse([], 15)
  },
  {
    pattern: /\/api\/v1\/backup\/stats$/,
    handler: async () => jsonResponse({
      total_backups: 0,
      total_size_bytes: 0,
      oldest_backup: null,
      newest_backup: null
    }, 10)
  },
  {
    // Backup schedule GET endpoint - needed by Backup Scheduling tests
    pattern: /\/api\/v1\/backup\/schedule$/,
    method: 'GET',
    handler: async () => jsonResponse({
      enabled: true,
      interval_hours: 24,
      preferred_hour: 2,
      backup_type: 'full',
      pre_sync_backup: false
    })
  },
  {
    // Backup schedule PUT endpoint - for saving schedule changes
    pattern: /\/api\/v1\/backup\/schedule$/,
    method: 'PUT',
    handler: async () => jsonResponse({
      enabled: true,
      interval_hours: 24,
      preferred_hour: 2,
      backup_type: 'full',
      pre_sync_backup: false
    })
  },
  {
    // Backup schedule trigger endpoint - for running backup now
    pattern: /\/api\/v1\/backup\/schedule\/trigger$/,
    method: 'POST',
    handler: async () => jsonResponse({
      id: 'backup-triggered-' + Date.now(),
      type: 'full',
      filename: 'backup-triggered.tar.gz',
      size_bytes: 52428800,
      created_at: new Date().toISOString(),
      notes: 'Manually triggered scheduled backup',
      database_records: 15000,
      is_valid: true
    })
  },
  {
    pattern: /\/api\/v1\/backup$/,
    method: 'POST',
    handler: async () => jsonResponse({
      id: 'backup-new-' + Date.now(),
      type: 'full',
      filename: 'backup-new.tar.gz',
      is_valid: true
    })
  },

  // =========================================================================
  // Insights and export
  // =========================================================================
  {
    pattern: /\/api\/v1\/insights/,
    handler: async () => jsonResponse({
      insights: [
        { type: 'peak_usage', message: 'Peak viewing time is 8-10 PM', severity: 'info' },
        { type: 'trending', message: 'Movie watching up 15% this week', severity: 'positive' }
      ],
      generated_at: new Date().toISOString()
    }, 50)
  },
  {
    pattern: /\/api\/v1\/export\/csv/,
    handler: async () => ({
      status: 200,
      contentType: 'text/csv',
      headers: { 'Content-Disposition': 'attachment; filename="export.csv"' },
      body: 'username,title,media_type\nJohnDoe,Inception,movie\n'
    })
  },
  {
    pattern: /\/api\/v1\/export\/geojson/,
    handler: async () => ({
      status: 200,
      contentType: 'application/json',
      headers: { 'Content-Disposition': 'attachment; filename="export.geojson"' },
      body: JSON.stringify({
        type: 'FeatureCollection',
        features: MOCK_CITIES.map(city => ({
          type: 'Feature',
          geometry: { type: 'Point', coordinates: [city.lon, city.lat] },
          properties: { city: city.name, country: city.country }
        }))
      })
    })
  },

  // =========================================================================
  // WebSocket (return 503 - not available during tests)
  // =========================================================================
  {
    pattern: /\/api\/v1\/ws/,
    handler: async () => ({
      status: 503,
      contentType: 'text/plain',
      body: 'WebSocket not available during E2E tests'
    })
  },

  // =========================================================================
  // Sync endpoint
  // =========================================================================
  {
    pattern: /\/api\/v1\/sync/,
    method: 'POST',
    handler: async () => jsonResponse({
      synced: true,
      new_playbacks: 0,
      sync_duration_ms: 150
    })
  },

  // =========================================================================
  // Server Management endpoints (ADR-0026)
  // =========================================================================
  {
    pattern: /\/api\/v1\/admin\/servers\/test$/,
    method: 'POST',
    handler: async () => jsonResponse({
      success: true,
      server_name: 'Test Plex Server',
      version: '1.32.5.7349',
      latency_ms: 45,
    })
  },
  {
    pattern: /\/api\/v1\/admin\/servers\/db$/,
    handler: async () => {
      return jsonResponse(getMockDBServers(), 15);
    }
  },
  {
    pattern: /\/api\/v1\/admin\/servers\/([0-9a-f-]+)$/,
    method: 'GET',
    handler: async (_route, _request, url) => {
      const match = url.pathname.match(/\/api\/v1\/admin\/servers\/([0-9a-f-]+)$/);
      const serverId = match ? match[1] : '';
      const server = getMockDBServers().find((s: any) => s.id === serverId);
      if (!server) {
        return errorResponse('NOT_FOUND', 'Server not found', 404);
      }
      return jsonResponse(server);
    }
  },
  {
    pattern: /\/api\/v1\/admin\/servers\/([0-9a-f-]+)$/,
    method: 'PUT',
    handler: async (_route, _request, url) => {
      const match = url.pathname.match(/\/api\/v1\/admin\/servers\/([0-9a-f-]+)$/);
      const serverId = match ? match[1] : '';
      const server = getMockDBServers().find((s: any) => s.id === serverId);
      if (!server) {
        return errorResponse('NOT_FOUND', 'Server not found', 404);
      }
      if (server.source === 'env') {
        return errorResponse('IMMUTABLE', 'Cannot modify server configured via environment variables', 403);
      }
      // Return updated server (mock doesn't actually update)
      return jsonResponse({
        ...server,
        updated_at: new Date().toISOString()
      });
    }
  },
  {
    pattern: /\/api\/v1\/admin\/servers\/([0-9a-f-]+)$/,
    method: 'DELETE',
    handler: async (_route, _request, url) => {
      const match = url.pathname.match(/\/api\/v1\/admin\/servers\/([0-9a-f-]+)$/);
      const serverId = match ? match[1] : '';
      const server = getMockDBServers().find((s: any) => s.id === serverId);
      if (!server) {
        return errorResponse('NOT_FOUND', 'Server not found', 404);
      }
      if (server.source === 'env') {
        return errorResponse('IMMUTABLE', 'Cannot delete server configured via environment variables', 403);
      }
      return jsonResponse({ message: 'Server deleted successfully' });
    }
  },
  {
    pattern: /\/api\/v1\/admin\/servers$/,
    method: 'POST',
    handler: async () => jsonResponse({
      id: generateUUID(),
      platform: 'plex',
      name: 'New Test Server',
      url: 'http://localhost:32400',
      token_masked: '****...****',
      server_id: 'mock-server-id',
      enabled: true,
      source: 'ui',
      realtime_enabled: false,
      webhooks_enabled: false,
      session_polling_enabled: false,
      session_polling_interval: '30s',
      status: 'configured',
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
      immutable: false
    }, 25)
  },
  {
    pattern: /\/api\/v1\/admin\/servers$/,
    handler: async () => {
      // GET /api/v1/admin/servers - returns all servers (env + DB)
      return jsonResponse({
        servers: getMockMediaServers(),
        total_count: 3,
        connected_count: 2,
        syncing_count: 0,
        error_count: 1
      }, 20);
    }
  },
  {
    pattern: /\/api\/v1\/server-status$/,
    handler: async () => {
      // Alias for server status used by MultiServerManager
      return jsonResponse({
        servers: getMockMediaServers(),
        total_count: 3,
        connected_count: 2,
        syncing_count: 0,
        error_count: 1
      }, 20);
    }
  },
  {
    pattern: /\/api\/v1\/sync\/server\/([0-9a-f-]+)$/,
    method: 'POST',
    handler: async () => jsonResponse({
      triggered: true,
      server_id: 'mock-server-id',
      message: 'Sync triggered successfully'
    })
  },
];

// ============================================================================
// Non-API Route Handlers (tiles, geocoder)
// ============================================================================

const PLACEHOLDER_TILE_PNG = Buffer.from(
  'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==',
  'base64'
);

const OTHER_HANDLERS: RouteHandler[] = [
  // Map tiles
  {
    pattern: /\.(png|pbf|mvt)(\?.*)?$/,
    handler: async () => ({
      status: 200,
      contentType: 'image/png',
      headers: { 'Cache-Control': 'public, max-age=86400' },
      body: PLACEHOLDER_TILE_PNG
    })
  },
  // Geocoder
  {
    pattern: /\/geocoding\//,
    handler: async () => ({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ features: [] })
    })
  },
  // External tile CDNs
  // Note: Patterns include trailing / to properly anchor the hostname
  {
    pattern: /^https?:\/\/[^/]*basemaps\.cartocdn\.com\//,
    handler: async () => ({
      status: 200,
      contentType: 'image/png',
      body: PLACEHOLDER_TILE_PNG
    })
  },
  {
    pattern: /^https?:\/\/[^/]*tile\.openstreetmap\.org\//,
    handler: async () => ({
      status: 200,
      contentType: 'image/png',
      body: PLACEHOLDER_TILE_PNG
    })
  },
  {
    pattern: /^https?:\/\/[^/]*nominatim\.openstreetmap\.org\//,
    handler: async () => ({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify([])
    })
  },
];

// ============================================================================
// Public API
// ============================================================================

/**
 * Setup all API mocking on a browser context.
 *
 * CRITICAL: This must be called on the context BEFORE any page is created.
 * Context-level routing is deterministic and applies to all pages.
 *
 * @param context - Playwright browser context
 * @param options - Configuration options
 */
export async function setupMockServer(
  context: BrowserContext,
  options: {
    enableApiMocking?: boolean;
    enableTileMocking?: boolean;
    logRequests?: boolean;
  } = {}
): Promise<void> {
  const {
    enableApiMocking = true,
    enableTileMocking = true,
    // Only log requests if E2E_VERBOSE is set (reduces CI noise significantly)
    logRequests = process.env.E2E_VERBOSE === 'true'
  } = options;

  // Reset mock data for fresh test run
  mockData = null;
  resetSeed();
  uuidCounter = 0;

  // Counters for diagnostics
  let requestCount = 0;
  let interceptedCount = 0;

  // Register a SINGLE route handler for all API requests
  if (enableApiMocking) {
    await context.route(/\/api\/v1\//, async (route, request) => {
      // Wrap entire route handling in semaphore to limit concurrent operations
      // This prevents overwhelming Express proxy and CDP with 30+ parallel requests
      await routeHandlerSemaphore.run(async () => {
        await handleApiRoute(route, request, logRequests, () => requestCount++, () => interceptedCount++);
      });
    });
  }

  // Extracted route handler for cleaner semaphore wrapping
  async function handleApiRoute(
    route: Route,
    request: Request,
    logRequests: boolean,
    incrementRequest: () => void,
    incrementIntercepted: () => void
  ): Promise<void> {
    incrementRequest();
    const currentCount = requestCount;
    const url = new URL(request.url());
    const path = url.pathname;
    const method = request.method();

    if (logRequests && currentCount <= 30) {
      console.log(`[MOCK] #${currentCount} ${method} ${path}`);
    }

    // Helper to safely fulfill route (handles page closure/race conditions)
    // Uses FulfillmentSemaphore to limit concurrent CDP calls (prevents timeout)
    const safeFulfill = async (response: { status?: number; contentType?: string; headers?: Record<string, string>; body: string | Buffer }) => {
      const responseContentType = response.contentType ?? 'application/json';
      // Merge Content-Type into headers explicitly to ensure it's always set
      const mergedHeaders: Record<string, string> = {
        'Content-Type': responseContentType,
        ...(response.headers ?? {})
      };

      // Use semaphore to limit concurrent fulfillments (prevents timeout from serial queue)
      await fulfillmentSemaphore.run(async () => {
        try {
          await route.fulfill({
            status: response.status ?? 200,
            headers: mergedHeaders,
            body: response.body
          });
          if (logRequests && currentCount <= 5) {
            console.log(`[MOCK] Fulfilled #${currentCount} ${path} (${response.status ?? 200})`);
          }
        } catch {
          // Route may have been aborted due to page navigation/closure
          // CRITICAL FIX: Do NOT call route.abort() here!
          // Calling abort('failed') causes net::ERR_FAILED in the browser.
          // If fulfill fails, the route was already handled (navigated away,
          // page closed, or fulfilled by another handler). Silently swallow
          // the error - this is expected behavior during page transitions.
        }
      });
    };

    // =====================================================================
    // STRATEGY 1: Try Express proxy first (deterministic, concurrent-safe)
    // =====================================================================
    const requestBody = method !== 'GET' && method !== 'HEAD'
      ? request.postData() || undefined
      : undefined;

    // Get request headers to forward X-Mock-* headers to Express proxy
    const requestHeaders = request.headers();

    const proxyResponse = await proxyToExpressServer(method, path, requestBody, requestHeaders);
    if (proxyResponse) {
      incrementIntercepted();
      await safeFulfill({
        status: proxyResponse.status,
        headers: proxyResponse.headers,
        body: proxyResponse.body
      });
      return;
    }

    // =====================================================================
    // STRATEGY 2: Fallback to inline handlers (for local dev without Express)
    // =====================================================================
    for (const handler of API_HANDLERS) {
      if (handler.pattern.test(path)) {
        if (handler.method && handler.method !== method) {
          continue; // Method doesn't match
        }

        try {
          const response = await handler.handler(route, request, url);
          if (response) {
            incrementIntercepted();
            await safeFulfill(response);
            return;
          }
        } catch (error) {
          console.error(`[MOCK] Handler error for ${path}:`, error);
          await safeFulfill(errorResponse('HANDLER_ERROR', String(error)));
          return;
        }
      }
    }

    // No handler matched - return 501 with diagnostic info
    if (logRequests) {
      console.warn(`[MOCK] UNMOCKED: ${method} ${path}`);
    }
    await safeFulfill({
      status: 501,
      contentType: 'application/json',
      body: JSON.stringify({
        status: 'error',
        error: {
          code: 'UNMOCKED_ENDPOINT',
          message: `E2E Test: Add handler for ${method} ${path}`
        }
      })
    });
  }

  // Register tile/geocoder handlers (uses semaphore to limit concurrent CDP calls)
  if (enableTileMocking) {
    for (const handler of OTHER_HANDLERS) {
      await context.route(handler.pattern, async (route, request) => {
        try {
          const response = await handler.handler(route, request, new URL(request.url()));
          if (response) {
            // Use semaphore to limit concurrent fulfillments (prevents CDP race conditions)
            await fulfillmentSemaphore.run(async () => {
              try {
                const responseContentType = response.contentType ?? 'application/octet-stream';
                const mergedHeaders: Record<string, string> = {
                  'Content-Type': responseContentType,
                  ...(response.headers ?? {})
                };
                await route.fulfill({
                  status: response.status ?? 200,
                  headers: mergedHeaders,
                  body: response.body
                });
              } catch {
                // Route may have been aborted due to page navigation/closure
                // CRITICAL FIX: Do NOT call route.abort() here - it causes net::ERR_FAILED
                // Silently swallow the error - this is expected behavior during page transitions.
              }
            });
          } else {
            try {
              await route.continue();
            } catch {
              // Route may have been aborted - OK
            }
          }
        } catch {
          try {
            await route.continue();
          } catch {
            // Route may have been aborted - OK
          }
        }
      });
    }
  }

  if (logRequests) {
    console.log('[MOCK] Mock server initialized on browser context');
  }
}

/**
 * Reset mock data to initial state.
 * Call this between tests if needed for isolation.
 */
export function resetMockData(): void {
  mockData = null;
  resetSeed();
  uuidCounter = 0;
  // Note: expressServerAvailable is intentionally NOT reset here
  // It's checked once per test run and remains stable
}

/**
 * Get current mock data (for assertions in tests).
 */
export function getCurrentMockData() {
  return getMockData();
}
