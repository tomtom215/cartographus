// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Standalone Mock API Server for E2E Tests
 *
 * This is the INDUSTRY STANDARD approach for deterministic E2E testing.
 * Instead of using Playwright's route interception (which has race conditions
 * with concurrent requests), we run a real HTTP server that serves mock data.
 *
 * ADVANTAGES:
 * - 100% deterministic - real HTTP requests through full network stack
 * - No race conditions - Express handles concurrency properly
 * - Traceable - all requests are logged
 * - Debuggable - can attach debugger, add breakpoints
 * - Portable - works with any test framework
 *
 * USAGE:
 * 1. Start server before tests: `npx ts-node mock-api-server.ts`
 * 2. Or use programmatically: `const server = await startMockServer(3900)`
 * 3. Configure app to use mock server URL in tests
 *
 * @see ADR-0025: Deterministic E2E Test Mocking
 */

import express, { Request, Response, NextFunction } from 'express';
import cors from 'cors';
import { Server } from 'http';
import * as path from 'path';
import * as fs from 'fs';

// ============================================================================
// Types
// ============================================================================

interface MockServerOptions {
  port: number;
  logRequests?: boolean;
}

// ============================================================================
// Mock Data (Deterministic - no random values)
// ============================================================================

const MOCK_USERS = [
  'JohnDoe', 'JaneSmith', 'MovieBuff', 'TVFanatic', 'StreamKing',
  'BingeWatcher', 'CinemaLover', 'SeriesAddict', 'FilmEnthusiast', 'ViewerOne'
];

const MOCK_PLATFORMS = ['Roku', 'Apple TV', 'Chrome', 'Firefox', 'Safari', 'Android TV', 'iOS', 'Plex Web'];

const MOCK_MOVIES = [
  'The Shawshank Redemption', 'The Dark Knight', 'Inception', 'Pulp Fiction',
  'The Matrix', 'Forrest Gump', 'Interstellar', 'The Godfather',
  'Fight Club', 'The Lord of the Rings'
];

const MOCK_TV_SHOWS = [
  'Breaking Bad', 'Game of Thrones', 'The Office', 'Friends',
  'Stranger Things', 'The Crown', 'The Mandalorian', 'Succession',
];

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

// ============================================================================
// Response Helpers
// ============================================================================

function jsonSuccess(data: unknown, queryTimeMs = 15) {
  return {
    status: 'success',
    data,
    metadata: {
      timestamp: new Date().toISOString(),
      query_time_ms: queryTimeMs
    }
  };
}

function jsonError(code: string, message: string) {
  return {
    status: 'error',
    error: { code, message },
    metadata: { timestamp: new Date().toISOString() }
  };
}

// ============================================================================
// Mock Data Generators (Deterministic)
// ============================================================================

function generateStats() {
  return {
    total_playbacks: 2847,
    unique_locations: 156,
    unique_users: 42,
    recent_activity: 67,
    recent_24h: 67,
    total_watch_time: 15840000,
    avg_watch_time: 92
  };
}

function generatePlaybackTrends() {
  const trends = [];
  const baseDate = new Date('2025-12-15');
  for (let i = 29; i >= 0; i--) {
    const date = new Date(baseDate);
    date.setDate(date.getDate() - i);
    trends.push({
      date: date.toISOString().split('T')[0],
      playback_count: 50 + (i % 10) * 5,
      unique_users: 10 + (i % 5)
    });
  }
  return trends;
}

function generateLocations() {
  return MOCK_CITIES.map((city, i) => ({
    id: `loc-${i + 1}`,
    latitude: city.lat,
    longitude: city.lon,
    city: city.name,
    country: city.country,
    playback_count: city.count,
    last_activity: new Date(Date.now() - i * 86400000).toISOString()
  }));
}

function generatePlaybacks() {
  const events = [];
  for (let i = 0; i < 50; i++) {
    const city = MOCK_CITIES[i % MOCK_CITIES.length];
    events.push({
      id: `event-${i + 1}`,
      session_key: `session-${i}`,
      user: MOCK_USERS[i % MOCK_USERS.length],
      media_type: i % 3 === 0 ? 'movie' : 'episode',
      title: i % 3 === 0 ? MOCK_MOVIES[i % MOCK_MOVIES.length] : MOCK_TV_SHOWS[i % MOCK_TV_SHOWS.length],
      started_at: new Date(Date.now() - i * 3600000).toISOString(),
      stopped_at: new Date(Date.now() - i * 3600000 + 5400000).toISOString(),
      duration: 5400000,
      platform: MOCK_PLATFORMS[i % MOCK_PLATFORMS.length],
      player: 'Plex Web',
      latitude: city.lat,
      longitude: city.lon,
      city: city.name,
      country: city.country,
      transcode_decision: i % 2 === 0 ? 'direct play' : 'transcode',
      video_resolution: ['4k', '1080p', '720p'][i % 3],
      bandwidth: 15000 + i * 100
    });
  }
  return events;
}

// ============================================================================
// Express App Setup
// ============================================================================

export function createMockApp(options: { logRequests?: boolean } = {}): express.Application {
  const app = express();
  const { logRequests = false } = options;

  // Middleware
  app.use(cors());
  app.use(express.json());

  // Request logging
  if (logRequests) {
    app.use((req: Request, _res: Response, next: NextFunction) => {
      console.log(`[MOCK-SERVER] ${req.method} ${req.path}`);
      next();
    });
  }

  // =========================================================================
  // Auth Endpoints
  // =========================================================================

  app.post('/api/v1/auth/login', (_req, res) => {
    res.json(jsonSuccess({
      token: 'mock-jwt-token-for-e2e-testing',
      username: 'admin',
      // CRITICAL: role and user_id are required for AuthContext.setAuth()
      // Without role: 'admin', the RoleGuard hides admin-only UI elements
      // like the Data Governance tab, causing tests to fail
      role: 'admin',
      user_id: 'mock-admin-user-id',
      expires_at: new Date(Date.now() + 86400000).toISOString()
    }));
  });

  app.get('/api/v1/auth/verify', (_req, res) => {
    res.json(jsonSuccess({
      valid: true,
      username: 'admin',
      role: 'admin',
      user_id: 'mock-admin-user-id',
      expires_at: new Date(Date.now() + 86400000).toISOString()
    }));
  });

  app.post('/api/v1/auth/logout', (_req, res) => {
    res.json(jsonSuccess({ success: true }));
  });

  // =========================================================================
  // Session Management Endpoints (ADR-0015)
  // =========================================================================

  // Get active sessions for current user
  app.get('/api/v1/oidc/sessions', (_req, res) => {
    res.json({
      sessions: [
        {
          id: 'session-current-001',
          provider: 'jwt',
          created_at: new Date(Date.now() - 3600000).toISOString(),
          last_accessed_at: new Date().toISOString(),
          current: true
        },
        {
          id: 'session-plex-002',
          provider: 'plex',
          created_at: new Date(Date.now() - 86400000 * 2).toISOString(),
          last_accessed_at: new Date(Date.now() - 3600000 * 5).toISOString(),
          current: false
        },
        {
          id: 'session-oidc-003',
          provider: 'oidc',
          created_at: new Date(Date.now() - 86400000 * 7).toISOString(),
          last_accessed_at: new Date(Date.now() - 86400000).toISOString(),
          current: false
        }
      ]
    });
  });

  // Revoke a specific session
  app.delete('/api/v1/oidc/sessions/:sessionId', (req, res) => {
    const { sessionId } = req.params;
    res.json({ message: `Session ${sessionId} revoked successfully` });
  });

  // Logout from all sessions
  app.post('/api/v1/oidc/logout/all', (_req, res) => {
    res.json({
      message: 'All sessions logged out successfully',
      sessions_count: 3
    });
  });

  // Get user info
  app.get('/api/v1/oidc/userinfo', (_req, res) => {
    res.json({
      id: 'user-001',
      username: 'admin',
      email: 'admin@example.com',
      roles: ['admin'],
      provider: 'jwt'
    });
  });

  // =========================================================================
  // Health Endpoints
  // =========================================================================

  app.get('/api/v1/health', (_req, res) => {
    res.json(jsonSuccess({
      // Health status must be lowercase 'healthy' or 'degraded' to match HealthStatus type
      // The frontend's HealthRenderer.calculateOverallHealth() determines display text
      status: 'healthy',
      version: '1.0.0-mock',
      database_connected: true,
      tautulli_connected: true,
      nats_connected: true,
      wal_healthy: true,
      websocket_connected: true,
      detection_enabled: true,
      last_sync_time: new Date(Date.now() - 5 * 60 * 1000).toISOString(), // 5 minutes ago
      last_backup_time: new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString(), // 1 day ago
      last_detection_time: new Date(Date.now() - 10 * 60 * 1000).toISOString(), // 10 minutes ago
      uptime: 86400, // 1 day in seconds
      uptime_formatted: '1 day',
    }));
  });

  app.get('/api/v1/health/nats', (_req, res) => {
    res.json(jsonSuccess({
      status: 'healthy',
      connected: true,
      jetstream_enabled: true,
      streams: 3,
      consumers: 5,
      server_id: 'mock-nats-server',
      version: '2.10.0'
    }));
  });

  app.get('/api/v1/health/setup', (_req, res) => {
    res.json(jsonSuccess({ setup_complete: true }));
  });

  // WAL (Write-Ahead Log) endpoints
  app.get('/api/v1/wal/stats', (_req, res) => {
    res.json(jsonSuccess({
      status: 'healthy',
      healthy: true,
      pending_count: 12,
      processed_count: 15847,
      failed_count: 3,
      db_size: 2097152, // 2MB
      db_size_formatted: '2.00 MB',
      last_compaction: new Date(Date.now() - 60 * 60 * 1000).toISOString(), // 1 hour ago
    }));
  });

  app.get('/api/v1/wal/health', (_req, res) => {
    res.json(jsonSuccess({
      healthy: true,
      status: 'healthy',
      message: 'WAL is operating normally',
    }));
  });

  // Media server status endpoint (ADR-0026)
  app.get('/api/v1/admin/servers', (_req, res) => {
    const now = new Date();
    const fiveMinutesAgo = new Date(now.getTime() - 5 * 60 * 1000);
    const twoHoursAgo = new Date(now.getTime() - 2 * 60 * 60 * 1000);

    res.json(jsonSuccess({
      servers: [
        {
          id: 'srv-plex-001',
          name: 'Main Plex Server',
          platform: 'plex',
          url: 'http://192.168.1.100:32400',
          status: 'connected',
          enabled: true,
          last_sync_at: fiveMinutesAgo.toISOString(),
          library_count: 5,
          user_count: 12,
          created_at: '2024-01-15T10:00:00Z',
          updated_at: fiveMinutesAgo.toISOString(),
        },
        {
          id: 'srv-jellyfin-001',
          name: 'Jellyfin Media',
          platform: 'jellyfin',
          url: 'http://192.168.1.101:8096',
          status: 'connected',
          enabled: true,
          last_sync_at: fiveMinutesAgo.toISOString(),
          library_count: 3,
          user_count: 8,
          created_at: '2024-02-20T14:00:00Z',
          updated_at: fiveMinutesAgo.toISOString(),
        },
        {
          id: 'srv-tautulli-001',
          name: 'Tautulli Monitor',
          platform: 'tautulli',
          url: 'http://192.168.1.100:8181',
          status: 'connected',
          enabled: true,
          last_sync_at: twoHoursAgo.toISOString(),
          created_at: '2024-01-15T10:30:00Z',
          updated_at: twoHoursAgo.toISOString(),
        },
      ],
      total_count: 3,
      connected_count: 3,
      error_count: 0,
      disabled_count: 0,
    }));
  });

  // =========================================================================
  // Stats Endpoint
  // =========================================================================

  app.get('/api/v1/stats', (_req, res) => {
    res.json(jsonSuccess(generateStats()));
  });

  // =========================================================================
  // Users and Filters
  // =========================================================================

  app.get('/api/v1/users', (_req, res) => {
    res.json(jsonSuccess(MOCK_USERS));
  });

  app.get('/api/v1/media-types', (_req, res) => {
    res.json(jsonSuccess(['movie', 'episode', 'track']));
  });

  app.get('/api/v1/filters', (_req, res) => {
    res.json(jsonSuccess({
      users: MOCK_USERS,
      media_types: ['movie', 'episode', 'track'],
      platforms: MOCK_PLATFORMS,
      players: ['Plex Web', 'Plex for Windows', 'Plex for Mac'],
      libraries: ['Movies', 'TV Shows', 'Music'],
      countries: MOCK_COUNTRIES.map(c => c.country)
    }));
  });

  // =========================================================================
  // Playbacks and Locations
  // =========================================================================

  app.get('/api/v1/playbacks', (_req, res) => {
    const events = generatePlaybacks();
    res.json(jsonSuccess({
      events,
      pagination: { limit: 50, has_more: false }
    }));
  });

  app.get('/api/v1/locations', (_req, res) => {
    res.json(jsonSuccess(generateLocations()));
  });

  // =========================================================================
  // Analytics Endpoints
  // =========================================================================

  app.get('/api/v1/analytics/trends', (_req, res) => {
    res.json(jsonSuccess({
      playback_trends: generatePlaybackTrends(),
      interval: 'day'
    }));
  });

  app.get('/api/v1/analytics/geographic', (_req, res) => {
    res.json(jsonSuccess({
      top_countries: MOCK_COUNTRIES,
      top_cities: MOCK_CITIES,
      media_type_distribution: [
        { type: 'movie', count: 400 },
        { type: 'episode', count: 600 },
        { type: 'track', count: 100 }
      ],
      // Heatmap data must be array of objects with hour, day_of_week, playback_count
      // This matches the structure expected by GeographicChartRenderer.renderHeatmap()
      viewing_hours_heatmap: Array.from({ length: 7 }, (_, day) =>
        Array.from({ length: 24 }, (_, hour) => ({
          hour,
          day_of_week: day,
          playback_count: hour >= 18 && hour <= 23 ? 10 : 3
        }))
      ).flat(),
      platform_distribution: MOCK_PLATFORMS.map(p => ({ platform: p, count: 50 }))
    }));
  });

  app.get('/api/v1/analytics/users', (_req, res) => {
    res.json(jsonSuccess(MOCK_USERS.slice(0, 10).map((user, i) => ({
      user,
      count: 100 - i * 8,
      watch_time: 500000 - i * 40000
    }))));
  });

  app.get('/api/v1/analytics/user-engagement', (_req, res) => {
    res.json(jsonSuccess(MOCK_USERS.slice(0, 10).map((user, i) => ({
      username: user,
      total_plays: 100 - i * 8,
      watch_time_hours: 50 - i * 4,
      avg_completion: 75,
      favorite_genre: ['Action', 'Drama', 'Comedy'][i % 3],
      last_activity: new Date(Date.now() - i * 86400000).toISOString()
    }))));
  });

  app.get('/api/v1/analytics/popular', (_req, res) => {
    res.json(jsonSuccess({
      movies: MOCK_MOVIES.slice(0, 5).map((title, i) => ({ title, count: 50 - i * 8 })),
      shows: MOCK_TV_SHOWS.slice(0, 5).map((title, i) => ({ title, count: 60 - i * 10 }))
    }));
  });

  app.get('/api/v1/analytics/platforms', (_req, res) => {
    res.json(jsonSuccess(MOCK_PLATFORMS.map(platform => ({
      platform,
      count: 50
    }))));
  });

  app.get('/api/v1/analytics/heatmap', (_req, res) => {
    res.json(jsonSuccess(Array.from({ length: 7 }, () =>
      Array.from({ length: 24 }, (_, h) => (h >= 18 && h <= 23 ? 10 : 3))
    )));
  });

  app.get('/api/v1/analytics/transcode', (_req, res) => {
    res.json(jsonSuccess({
      direct_play: 500,
      transcode: 300,
      copy: 100
    }));
  });

  app.get('/api/v1/analytics/resolution', (_req, res) => {
    res.json(jsonSuccess([
      { resolution: '4K', count: 150 },
      { resolution: '1080p', count: 400 },
      { resolution: '720p', count: 200 },
      { resolution: '480p', count: 50 }
    ]));
  });

  app.get('/api/v1/analytics/bitrate', (_req, res) => {
    res.json(jsonSuccess({
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
    }));
  });

  app.get('/api/v1/analytics/concurrent', (_req, res) => {
    res.json(jsonSuccess({
      max_concurrent: 8,
      avg_concurrent: 3.2,
      peak_time: '21:00',
      by_hour: Array.from({ length: 24 }, (_, i) => ({ hour: i, max: 2 + Math.floor(i / 4) }))
    }));
  });

  app.get('/api/v1/analytics/abandonment', (_req, res) => {
    res.json(jsonSuccess({
      total_abandoned: 120,
      abandonment_rate: 15.2,
      avg_drop_off_percent: 32
    }));
  });

  app.get('/api/v1/analytics/hardware-transcode', (_req, res) => {
    res.json(jsonSuccess({
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
    }));
  });

  app.get('/api/v1/analytics/connection-security', (_req, res) => {
    res.json(jsonSuccess({
      total_playbacks: 500,
      secure_connections: 420,
      insecure_connections: 30,
      secure_percent: 84,
      relayed_connections: { count: 50, percent: 10, users: ['user1', 'user2'], reason: 'NAT traversal' },
      local_connections: { count: 150, percent: 30 },
      by_user: MOCK_USERS.slice(0, 5).map(username => ({
        username,
        total_streams: 30,
        secure_rate: 85,
        relay_rate: 5,
        local_rate: 30
      })),
      by_platform: MOCK_PLATFORMS.slice(0, 4).map(platform => ({
        platform,
        secure_rate: 80,
        relay_rate: 10
      }))
    }));
  });

  app.get('/api/v1/analytics/pause-patterns', (_req, res) => {
    res.json(jsonSuccess({
      total_pauses: 1250,
      avg_pauses_per_play: 2.3,
      avg_pause_duration: 45,
      pause_by_hour: Array.from({ length: 24 }, (_, i) => ({ hour: i, count: 50 })),
      high_pause_content: MOCK_MOVIES.slice(0, 5).map(title => ({
        title,
        media_type: 'movie',
        average_pauses: 4
      }))
    }));
  });

  app.get('/api/v1/analytics/binge', (_req, res) => {
    res.json(jsonSuccess({
      binge_sessions: 45,
      avg_episodes_per_binge: 4.2,
      top_binged_shows: MOCK_TV_SHOWS.slice(0, 5).map((title, i) => ({
        title,
        binge_count: 20 - i * 3,
        avg_episodes: 4
      })),
      binge_by_day: ['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun'].map(day => ({
        day,
        count: 8
      }))
    }));
  });

  app.get('/api/v1/analytics/watch-parties', (_req, res) => {
    res.json(jsonSuccess({
      total_parties: 12,
      avg_participants: 3.5,
      popular_content: MOCK_MOVIES.slice(0, 3).map(title => ({
        title,
        party_count: 5
      }))
    }));
  });

  app.get('/api/v1/analytics/bandwidth', (_req, res) => {
    res.json(jsonSuccess({
      avg_bandwidth: 15000,
      peak_bandwidth: 45000,
      by_quality: [
        { quality: '4K', bandwidth: 25000, count: 50 },
        { quality: '1080p', bandwidth: 12000, count: 200 },
        { quality: '720p', bandwidth: 5000, count: 100 }
      ]
    }));
  });

  app.get('/api/v1/analytics/engagement', (_req, res) => {
    res.json(jsonSuccess({
      avg_completion_rate: 72,
      avg_watch_time: 5400,
      engagement_by_type: [
        { type: 'movie', completion_rate: 68, avg_duration: 6800 },
        { type: 'episode', completion_rate: 85, avg_duration: 2400 }
      ]
    }));
  });

  app.get('/api/v1/analytics/comparative', (_req, res) => {
    res.json(jsonSuccess({
      current_period: { playbacks: 500, users: 42, watch_time: 150000 },
      previous_period: { playbacks: 420, users: 38, watch_time: 130000 },
      changes: { playbacks_pct: 19, users_pct: 10.5, watch_time_pct: 15.4 }
    }));
  });

  app.get('/api/v1/analytics/temporal-heatmap', (_req, res) => {
    res.json(jsonSuccess({
      data: Array.from({ length: 7 }, (_, day) =>
        Array.from({ length: 24 }, (_, hour) => ({
          day, hour, value: hour >= 18 && hour <= 23 ? 15 : 5
        }))
      ).flat()
    }));
  });

  app.get('/api/v1/analytics/resolution-mismatch', (_req, res) => {
    res.json(jsonSuccess({
      total_mismatches: 85,
      mismatch_rate: 12.5,
      by_reason: [
        { reason: 'bandwidth', count: 45 },
        { reason: 'device_capability', count: 30 },
        { reason: 'transcoding', count: 10 }
      ]
    }));
  });

  app.get('/api/v1/analytics/hdr', (_req, res) => {
    res.json(jsonSuccess({
      hdr_playbacks: 120,
      sdr_playbacks: 380,
      hdr_capable_users: 25,
      by_format: [
        { format: 'HDR10', count: 80 },
        { format: 'Dolby Vision', count: 35 },
        { format: 'HLG', count: 5 }
      ]
    }));
  });

  app.get('/api/v1/analytics/audio', (_req, res) => {
    res.json(jsonSuccess({
      surround_playbacks: 200,
      stereo_playbacks: 300,
      by_codec: [
        { codec: 'TrueHD Atmos', count: 50 },
        { codec: 'DTS-HD MA', count: 80 },
        { codec: 'EAC3', count: 70 },
        { codec: 'AAC', count: 300 }
      ]
    }));
  });

  app.get('/api/v1/analytics/subtitle', (_req, res) => {
    res.json(jsonSuccess({
      subtitle_usage_rate: 35,
      by_language: [
        { language: 'English', count: 150 },
        { language: 'Spanish', count: 30 },
        { language: 'French', count: 20 }
      ],
      forced_subtitles: 45
    }));
  });

  // Alias for subtitles (frontend uses both)
  app.get('/api/v1/analytics/subtitles', (_req, res) => {
    res.json(jsonSuccess({
      subtitle_usage_rate: 35,
      by_language: [
        { language: 'English', count: 150 },
        { language: 'Spanish', count: 30 },
        { language: 'French', count: 20 }
      ],
      forced_subtitles: 45
    }));
  });

  app.get('/api/v1/analytics/frame-rate', (_req, res) => {
    res.json(jsonSuccess({
      by_fps: [
        { fps: 24, count: 300, completion_rate: 72 },
        { fps: 30, count: 150, completion_rate: 68 },
        { fps: 60, count: 50, completion_rate: 85 }
      ]
    }));
  });

  app.get('/api/v1/analytics/concurrent-streams', (_req, res) => {
    res.json(jsonSuccess({
      max_concurrent: 8,
      avg_concurrent: 3.2,
      peak_time: '21:00',
      by_hour: Array.from({ length: 24 }, (_, i) => ({ hour: i, max: 2 + Math.floor(i / 4) }))
    }));
  });

  // Catch-all for /api/v1/analytics
  app.get('/api/v1/analytics', (_req, res) => {
    res.json(jsonSuccess({
      playback_trends: generatePlaybackTrends(),
      media_distribution: [
        { type: 'movie', count: 400 },
        { type: 'episode', count: 600 },
        { type: 'track', count: 100 }
      ],
      top_users: MOCK_USERS.slice(0, 10).map((user, i) => ({
        user,
        count: 100 - i * 8,
        watch_time: 500000 - i * 40000
      })),
      platforms: MOCK_PLATFORMS.map(p => ({ platform: p, count: 50 })),
      countries: MOCK_COUNTRIES,
      cities: MOCK_CITIES
    }));
  });

  // =========================================================================
  // Tautulli Endpoints
  // =========================================================================

  app.get('/api/v1/tautulli/activity', (req, res) => {
    // Check for empty sessions header (for testing empty state)
    if (req.headers['x-mock-empty-sessions'] === 'true') {
      res.json(jsonSuccess({
        sessions: [],
        stream_count: 0,
        transcode_count: 0,
        total_bandwidth: 0
      }));
      return;
    }

    // Default: return mock session data for session management tests
    res.json(jsonSuccess({
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
    }));
  });

  app.get('/api/v1/tautulli/recently-added', (_req, res) => {
    res.json(jsonSuccess({
      records_total: 0,
      recently_added: []
    }));
  });

  app.get('/api/v1/tautulli/server-info', (_req, res) => {
    res.json(jsonSuccess({
      name: 'Test Plex Server',
      version: '1.32.5.7349',
      platform: 'Linux',
      online: true
    }));
  });

  app.get('/api/v1/tautulli/server-list', (_req, res) => {
    res.json(jsonSuccess([{
      name: 'Plex Media Server',
      machine_identifier: 'mock-server-id',
      host: '192.168.1.100',
      port: 32400
    }]));
  });

  app.get('/api/v1/tautulli/tautulli-info', (_req, res) => {
    res.json(jsonSuccess({
      tautulli_version: '2.13.4',
      tautulli_branch: 'master'
    }));
  });

  app.get('/api/v1/tautulli/pms-update', (_req, res) => {
    res.json(jsonSuccess({ update_available: false }));
  });

  app.get('/api/v1/tautulli/library-names', (_req, res) => {
    res.json(jsonSuccess([
      { section_id: 1, section_name: 'Movies', section_type: 'movie' },
      { section_id: 2, section_name: 'TV Shows', section_type: 'show' },
      { section_id: 3, section_name: 'Music', section_type: 'artist' }
    ]));
  });

  app.post('/api/v1/tautulli/terminate-session', (req, res) => {
    // Check for error simulation header (for testing error handling)
    if (req.headers['x-mock-terminate-error'] === 'true') {
      res.status(500).json(jsonError('TERMINATION_FAILED', 'Failed to terminate session'));
      return;
    }
    res.json(jsonSuccess({ result: 'success' }));
  });

  // Rating Keys Lookup endpoints
  // Supports X-Mock-* headers for test scenarios:
  //   X-Mock-Rating-Keys-Error: 'true' - returns 500 error
  //   X-Mock-Rating-Keys-Empty: 'true' - returns empty results
  app.get('/api/v1/tautulli/new-rating-keys', (req, res) => {
    // Check for error simulation header
    if (req.headers['x-mock-rating-keys-error'] === 'true') {
      res.status(500).json(jsonError('LOOKUP_FAILED', 'Failed to lookup rating keys'));
      return;
    }
    // Check for empty results header
    if (req.headers['x-mock-rating-keys-empty'] === 'true') {
      res.json(jsonSuccess({ rating_keys: [] }));
      return;
    }
    const ratingKey = req.query.rating_key;
    if (!ratingKey) {
      res.status(400).json(jsonError('MISSING_PARAM', 'rating_key is required'));
      return;
    }
    // Return mock rating key mappings (field name must match API client: rating_keys)
    res.json(jsonSuccess({
      rating_keys: [
        { old_rating_key: '1001', new_rating_key: ratingKey, title: 'Test Movie', media_type: 'movie', updated_at: 1702400000 },
        { old_rating_key: '1002', new_rating_key: ratingKey, title: 'Test Episode', media_type: 'episode', updated_at: 1702500000 }
      ]
    }));
  });

  app.get('/api/v1/tautulli/old-rating-keys', (req, res) => {
    // Check for error simulation header
    if (req.headers['x-mock-rating-keys-error'] === 'true') {
      res.status(500).json(jsonError('LOOKUP_FAILED', 'Failed to lookup rating keys'));
      return;
    }
    // Check for empty results header
    if (req.headers['x-mock-rating-keys-empty'] === 'true') {
      res.json(jsonSuccess({ rating_keys: [] }));
      return;
    }
    const ratingKey = req.query.rating_key;
    if (!ratingKey) {
      res.status(400).json(jsonError('MISSING_PARAM', 'rating_key is required'));
      return;
    }
    // Return mock old rating key mappings (field name must match API client: rating_keys)
    res.json(jsonSuccess({
      rating_keys: [
        { old_rating_key: ratingKey, new_rating_key: '2001', title: 'Migrated Movie', media_type: 'movie', updated_at: 1702300000 },
        { old_rating_key: ratingKey, new_rating_key: '2002', title: 'Migrated Episode', media_type: 'episode', updated_at: 1702350000 }
      ]
    }));
  });

  // Metadata Deep-Dive endpoint
  app.get('/api/v1/tautulli/metadata', (req, res) => {
    const ratingKey = req.query.rating_key;
    if (!ratingKey) {
      res.status(400).json(jsonError('MISSING_PARAM', 'rating_key is required'));
      return;
    }
    // Return comprehensive mock metadata
    res.json(jsonSuccess({
      rating_key: ratingKey,
      title: 'Test Movie',
      year: 2024,
      media_type: 'movie',
      duration: 7200000,
      content_rating: 'PG-13',
      summary: 'A test movie for E2E testing purposes.',
      studio: 'Test Studios',
      genres: ['Action', 'Drama', 'Sci-Fi'],
      directors: ['Test Director'],
      writers: ['Test Writer 1', 'Test Writer 2'],
      actors: ['Actor One', 'Actor Two', 'Actor Three'],
      audience_rating: 8.5,
      critic_rating: 7.8,
      added_at: new Date(Date.now() - 30 * 24 * 60 * 60 * 1000).toISOString(),
      updated_at: new Date().toISOString(),
      last_viewed_at: new Date(Date.now() - 7 * 24 * 60 * 60 * 1000).toISOString(),
      view_count: 42,
      media_info: {
        video_codec: 'hevc',
        video_resolution: '4k',
        audio_codec: 'truehd',
        audio_channels: 8,
        container: 'mkv',
        bitrate: 45000,
        aspect_ratio: '2.39'
      }
    }));
  });

  // Exports endpoints (48-exports.spec.ts)
  app.get('/api/v1/tautulli/exports-table', (_req, res) => {
    res.json(jsonSuccess({
      exports: [
        { id: 1, filename: 'export-2024-01.csv', created_at: new Date().toISOString(), size: 1024000 },
        { id: 2, filename: 'export-2024-02.csv', created_at: new Date().toISOString(), size: 2048000 }
      ],
      total: 2
    }));
  });

  app.get('/api/v1/tautulli/export-fields', (_req, res) => {
    res.json(jsonSuccess({
      fields: ['title', 'year', 'rating_key', 'media_type', 'duration', 'play_count', 'last_played']
    }));
  });

  app.get('/api/v1/tautulli/libraries-table', (_req, res) => {
    // Libraries table with full details
    res.json(jsonSuccess({
      draw: 1,
      recordsTotal: 3,
      recordsFiltered: 3,
      data: [
        {
          section_id: 1,
          section_name: 'Movies',
          section_type: 'movie',
          agent: 'tv.plex.agents.movie',
          thumb: '/library/sections/1/thumb',
          count: 500,
          parent_count: 0,
          child_count: 500,
          is_active: 1,
          do_notify: 1,
          do_notify_created: 1,
          keep_history: 1,
          deleted_section: 0,
          last_accessed: Date.now() / 1000,
          plays: 1250,
          duration: 450000
        },
        {
          section_id: 2,
          section_name: 'TV Shows',
          section_type: 'show',
          agent: 'tv.plex.agents.series',
          thumb: '/library/sections/2/thumb',
          count: 150,
          parent_count: 150,
          child_count: 2500,
          is_active: 1,
          do_notify: 1,
          do_notify_created: 1,
          keep_history: 1,
          deleted_section: 0,
          last_accessed: Date.now() / 1000,
          plays: 3200,
          duration: 980000
        },
        {
          section_id: 3,
          section_name: 'Music',
          section_type: 'artist',
          agent: 'tv.plex.agents.music',
          thumb: '/library/sections/3/thumb',
          count: 1000,
          parent_count: 1000,
          child_count: 15000,
          is_active: 1,
          do_notify: 0,
          do_notify_created: 0,
          keep_history: 1,
          deleted_section: 0,
          last_accessed: Date.now() / 1000,
          plays: 8500,
          duration: 320000
        }
      ]
    }));
  });

  // Library user stats
  app.get('/api/v1/tautulli/library-user-stats', (req, res) => {
    const sectionId = req.query.section_id;
    if (!sectionId) {
      res.status(400).json(jsonError('MISSING_PARAM', 'section_id is required'));
      return;
    }
    res.json(jsonSuccess([
      {
        user_id: 1,
        username: 'admin',
        friendly_name: 'Admin User',
        total_plays: 450,
        total_duration: 162000,
        last_watch: Date.now() / 1000 - 3600,
        last_played: 'Inception',
        user_thumb: '/users/1/thumb'
      },
      {
        user_id: 2,
        username: 'moviebuff',
        friendly_name: 'Movie Buff',
        total_plays: 320,
        total_duration: 115200,
        last_watch: Date.now() / 1000 - 7200,
        last_played: 'The Dark Knight',
        user_thumb: '/users/2/thumb'
      },
      {
        user_id: 3,
        username: 'casual',
        friendly_name: 'Casual Viewer',
        total_plays: 85,
        total_duration: 30600,
        last_watch: Date.now() / 1000 - 86400,
        last_played: 'Friends S01E01',
        user_thumb: '/users/3/thumb'
      }
    ]));
  });

  // Library media info
  app.get('/api/v1/tautulli/library-media-info', (req, res) => {
    const sectionId = req.query.section_id;
    if (!sectionId) {
      res.status(400).json(jsonError('MISSING_PARAM', 'section_id is required'));
      return;
    }
    res.json(jsonSuccess({
      draw: 1,
      recordsTotal: 500,
      recordsFiltered: 500,
      filtered_file_size: 2500000000000,
      total_file_size: 2500000000000,
      data: [
        {
          rating_key: '1001',
          title: 'Inception',
          year: 2010,
          media_type: 'movie',
          thumb: '/library/metadata/1001/thumb',
          added_at: Date.now() / 1000 - 86400 * 30,
          last_played: Date.now() / 1000 - 3600,
          play_count: 15,
          file_size: 45000000000,
          bitrate: 35000,
          video_resolution: '4k',
          video_codec: 'hevc',
          video_full_resolution: '2160p',
          audio_codec: 'truehd',
          audio_channels: 8,
          container: 'mkv',
          duration: 8880000
        },
        {
          rating_key: '1002',
          title: 'The Dark Knight',
          year: 2008,
          media_type: 'movie',
          thumb: '/library/metadata/1002/thumb',
          added_at: Date.now() / 1000 - 86400 * 60,
          last_played: Date.now() / 1000 - 7200,
          play_count: 22,
          file_size: 38000000000,
          bitrate: 28000,
          video_resolution: '1080',
          video_codec: 'h264',
          video_full_resolution: '1080p',
          audio_codec: 'dts',
          audio_channels: 6,
          container: 'mkv',
          duration: 9120000
        },
        {
          rating_key: '1003',
          title: 'Interstellar',
          year: 2014,
          media_type: 'movie',
          thumb: '/library/metadata/1003/thumb',
          added_at: Date.now() / 1000 - 86400 * 45,
          last_played: Date.now() / 1000 - 14400,
          play_count: 18,
          file_size: 52000000000,
          bitrate: 42000,
          video_resolution: '4k',
          video_codec: 'hevc',
          video_full_resolution: '2160p',
          audio_codec: 'truehd',
          audio_channels: 8,
          container: 'mkv',
          duration: 10140000
        }
      ]
    }));
  });

  // Library watch time stats
  app.get('/api/v1/tautulli/library-watch-time-stats', (req, res) => {
    const sectionId = req.query.section_id;
    if (!sectionId) {
      res.status(400).json(jsonError('MISSING_PARAM', 'section_id is required'));
      return;
    }
    res.json(jsonSuccess([
      { query_days: 1, total_plays: 12, total_duration: 43200 },
      { query_days: 7, total_plays: 85, total_duration: 306000 },
      { query_days: 30, total_plays: 320, total_duration: 1152000 },
      { query_days: 365, total_plays: 1250, total_duration: 4500000 }
    ]));
  });

  // Collections table
  app.get('/api/v1/tautulli/collections-table', (req, res) => {
    const sectionId = req.query.section_id;
    res.json(jsonSuccess({
      draw: 1,
      recordsTotal: 5,
      recordsFiltered: sectionId ? 3 : 5,
      data: [
        {
          rating_key: 'col-1001',
          section_id: 1,
          title: 'Marvel Cinematic Universe',
          sort_title: 'Marvel Cinematic Universe',
          summary: 'All Marvel movies in chronological order',
          thumb: '/library/collections/1001/thumb',
          child_count: 32,
          min_year: 2008,
          max_year: 2024,
          added_at: Date.now() / 1000 - 86400 * 365,
          updated_at: Date.now() / 1000 - 86400 * 7,
          content_rating: 'PG-13'
        },
        {
          rating_key: 'col-1002',
          section_id: 1,
          title: 'Christopher Nolan Collection',
          sort_title: 'Christopher Nolan Collection',
          summary: 'Films directed by Christopher Nolan',
          thumb: '/library/collections/1002/thumb',
          child_count: 12,
          min_year: 1998,
          max_year: 2023,
          added_at: Date.now() / 1000 - 86400 * 180,
          updated_at: Date.now() / 1000 - 86400 * 14,
          content_rating: 'PG-13'
        },
        {
          rating_key: 'col-1003',
          section_id: 1,
          title: 'Oscar Best Picture Winners',
          sort_title: 'Oscar Best Picture Winners',
          summary: 'Academy Award winners for Best Picture',
          thumb: '/library/collections/1003/thumb',
          child_count: 45,
          min_year: 1980,
          max_year: 2024,
          added_at: Date.now() / 1000 - 86400 * 90,
          updated_at: Date.now() / 1000 - 86400 * 30
        },
        {
          rating_key: 'col-2001',
          section_id: 2,
          title: 'HBO Originals',
          sort_title: 'HBO Originals',
          summary: 'Premium HBO series',
          thumb: '/library/collections/2001/thumb',
          child_count: 28,
          min_year: 1999,
          max_year: 2024,
          added_at: Date.now() / 1000 - 86400 * 200,
          updated_at: Date.now() / 1000 - 86400 * 3
        },
        {
          rating_key: 'col-2002',
          section_id: 2,
          title: 'Sci-Fi Classics',
          sort_title: 'Sci-Fi Classics',
          summary: 'Classic science fiction television',
          thumb: '/library/collections/2002/thumb',
          child_count: 15,
          min_year: 1966,
          max_year: 2005,
          added_at: Date.now() / 1000 - 86400 * 120,
          updated_at: Date.now() / 1000 - 86400 * 60
        }
      ]
    }));
  });

  // Playlists table
  app.get('/api/v1/tautulli/playlists-table', (_req, res) => {
    res.json(jsonSuccess({
      draw: 1,
      recordsTotal: 4,
      recordsFiltered: 4,
      data: [
        {
          rating_key: 'pl-1001',
          title: 'Weekend Movie Marathon',
          sort_title: 'Weekend Movie Marathon',
          summary: 'Action movies for weekend viewing',
          thumb: '/playlists/1001/composite',
          composite: '/playlists/1001/composite',
          duration: 36000,
          leaf_count: 10,
          smart: false,
          playlist_type: 'video',
          user: 'admin',
          username: 'admin',
          added_at: Date.now() / 1000 - 86400 * 30,
          updated_at: Date.now() / 1000 - 86400 * 2
        },
        {
          rating_key: 'pl-1002',
          title: 'Background Music',
          sort_title: 'Background Music',
          summary: 'Relaxing instrumental tracks',
          thumb: '/playlists/1002/composite',
          composite: '/playlists/1002/composite',
          duration: 14400,
          leaf_count: 85,
          smart: true,
          playlist_type: 'audio',
          user: 'admin',
          username: 'admin',
          added_at: Date.now() / 1000 - 86400 * 60,
          updated_at: Date.now() / 1000 - 86400 * 1
        },
        {
          rating_key: 'pl-1003',
          title: 'TV Show Catchup',
          sort_title: 'TV Show Catchup',
          summary: 'Episodes to watch',
          thumb: '/playlists/1003/composite',
          composite: '/playlists/1003/composite',
          duration: 21600,
          leaf_count: 24,
          smart: false,
          playlist_type: 'video',
          user: 'moviebuff',
          username: 'moviebuff',
          added_at: Date.now() / 1000 - 86400 * 14,
          updated_at: Date.now() / 1000 - 86400 * 1
        },
        {
          rating_key: 'pl-1004',
          title: 'Vacation Photos',
          sort_title: 'Vacation Photos',
          summary: 'Summer 2024 photos',
          thumb: '/playlists/1004/composite',
          composite: '/playlists/1004/composite',
          duration: 0,
          leaf_count: 150,
          smart: false,
          playlist_type: 'photo',
          user: 'admin',
          username: 'admin',
          added_at: Date.now() / 1000 - 86400 * 7,
          updated_at: Date.now() / 1000 - 86400 * 5
        }
      ]
    }));
  });

  app.get('/api/v1/tautulli/export-metadata', (_req, res) => {
    res.json(jsonSuccess({
      export_id: 1,
      filename: 'export-2024-01.csv',
      created_at: new Date().toISOString(),
      fields: ['title', 'year', 'rating_key'],
      row_count: 500
    }));
  });

  app.delete('/api/v1/tautulli/delete-export', (_req, res) => {
    res.json(jsonSuccess({ deleted: true }));
  });

  // Also support POST for delete-export (some implementations use POST)
  app.post('/api/v1/tautulli/delete-export', (_req, res) => {
    res.json(jsonSuccess({ deleted: true }));
  });

  // Search endpoint (51-search.spec.ts)
  app.get('/api/v1/tautulli/search', (req, res) => {
    const query = req.query.query || req.query.q || '';
    res.json(jsonSuccess({
      results: [
        { rating_key: '1001', title: `Search Result 1 for "${query}"`, media_type: 'movie', year: 2024 },
        { rating_key: '1002', title: `Search Result 2 for "${query}"`, media_type: 'episode', year: 2024 }
      ],
      total: 2
    }));
  });

  // Stream data endpoint (52-stream-data.spec.ts)
  app.get('/api/v1/tautulli/stream-data', (req, res) => {
    const sessionKey = req.query.session_key;
    if (!sessionKey) {
      res.status(400).json(jsonError('MISSING_PARAM', 'session_key is required'));
      return;
    }
    res.json(jsonSuccess({
      session_key: sessionKey,
      transcode_decision: 'transcode',
      video_decision: 'transcode',
      audio_decision: 'copy',
      subtitle_decision: 'burn',
      stream_container: 'mkv',
      stream_video_codec: 'h264',
      stream_video_resolution: '1080',
      stream_audio_codec: 'aac',
      stream_audio_channels: 2,
      bandwidth: 8000,
      quality_profile: '1080p (8 Mbps)',
      optimized_version: false,
      stream_duration: 3600,
      buffer_count: 2,
      buffer_last_time: new Date(Date.now() - 60000).toISOString()
    }));
  });

  // =========================================================================
  // Detection Endpoints
  // =========================================================================

  app.get('/api/v1/detection/alerts', (_req, res) => {
    res.json(jsonSuccess({ alerts: [], total: 0, limit: 20, offset: 0 }));
  });

  app.get('/api/v1/detection/stats', (_req, res) => {
    res.json(jsonSuccess({
      by_severity: { critical: 0, warning: 0, info: 0 },
      by_rule_type: {},
      unacknowledged: 0,
      total: 0
    }));
  });

  app.get('/api/v1/detection/rules', (_req, res) => {
    res.json(jsonSuccess({ rules: [] }));
  });

  app.get('/api/v1/detection/metrics', (_req, res) => {
    res.json(jsonSuccess({
      events_processed: 1250,
      alerts_generated: 0,
      detection_errors: 0
    }));
  });

  // =========================================================================
  // Dedupe Endpoints
  // =========================================================================

  app.get('/api/v1/dedupe/audit/stats', (_req, res) => {
    res.json(jsonSuccess({
      total_deduped: 0,
      pending_review: 0,
      user_restored: 0,
      accuracy_rate: 100.0
    }));
  });

  app.get('/api/v1/dedupe/audit', (_req, res) => {
    res.json(jsonSuccess({
      entries: [],
      total_count: 0,
      limit: 50,
      offset: 0
    }));
  });

  // =========================================================================
  // Audit Log Endpoints (ADR-0015)
  // =========================================================================

  const MOCK_AUDIT_EVENTS = [
    {
      id: 'audit-001',
      timestamp: new Date(Date.now() - 5 * 60 * 1000).toISOString(),
      type: 'auth.success',
      severity: 'info',
      outcome: 'success',
      actor: { id: 'user-1', type: 'user', name: 'JohnDoe', roles: ['admin'] },
      target: null,
      source: { ip_address: '192.168.1.100', user_agent: 'Mozilla/5.0 Chrome', hostname: 'localhost' },
      action: 'login',
      description: 'User logged in successfully',
      metadata: { provider: 'oidc', auth_method: 'pkce' },
      request_id: 'req-001'
    },
    {
      id: 'audit-002',
      timestamp: new Date(Date.now() - 15 * 60 * 1000).toISOString(),
      type: 'auth.failure',
      severity: 'warning',
      outcome: 'failure',
      actor: { id: 'user-2', type: 'user', name: 'UnknownUser' },
      target: null,
      source: { ip_address: '10.0.0.50', user_agent: 'Mozilla/5.0 Firefox' },
      action: 'login',
      description: 'Failed login attempt - invalid credentials',
      metadata: { error: 'invalid_password', attempts: 2 },
      request_id: 'req-002'
    },
    {
      id: 'audit-003',
      timestamp: new Date(Date.now() - 30 * 60 * 1000).toISOString(),
      type: 'authz.denied',
      severity: 'warning',
      outcome: 'failure',
      actor: { id: 'user-3', type: 'user', name: 'JaneSmith', roles: ['viewer'] },
      target: { id: '/api/v1/admin/users', type: 'endpoint', name: 'User Management' },
      source: { ip_address: '192.168.1.101' },
      action: 'access',
      description: 'User attempted to access admin endpoint without permission',
      correlation_id: 'corr-001',
      request_id: 'req-003'
    },
    {
      id: 'audit-004',
      timestamp: new Date(Date.now() - 60 * 60 * 1000).toISOString(),
      type: 'detection.alert',
      severity: 'error',
      outcome: 'success',
      actor: { id: 'user-4', type: 'user', name: 'SuspiciousUser' },
      target: null,
      source: { ip_address: '203.0.113.50', geo: { country: 'Unknown', city: 'Unknown' } },
      action: 'impossible_travel',
      description: 'Impossible travel detected: login from New York to Tokyo in 5 minutes',
      metadata: { rule: 'impossible_travel', distance_km: 10838, time_diff_mins: 5 },
      request_id: 'req-004'
    },
    {
      id: 'audit-005',
      timestamp: new Date(Date.now() - 2 * 60 * 60 * 1000).toISOString(),
      type: 'user.created',
      severity: 'info',
      outcome: 'success',
      actor: { id: 'user-1', type: 'user', name: 'AdminUser', roles: ['admin'] },
      target: { id: 'user-5', type: 'user', name: 'NewUser' },
      source: { ip_address: '192.168.1.100' },
      action: 'create',
      description: 'Admin created new user account',
      request_id: 'req-005'
    },
    {
      id: 'audit-006',
      timestamp: new Date(Date.now() - 3 * 60 * 60 * 1000).toISOString(),
      type: 'data.export',
      severity: 'info',
      outcome: 'success',
      actor: { id: 'user-1', type: 'user', name: 'JohnDoe', roles: ['admin'] },
      target: { id: 'playbacks', type: 'dataset', name: 'Playback History' },
      source: { ip_address: '192.168.1.100' },
      action: 'export',
      description: 'Exported playback data to CSV format',
      metadata: { format: 'csv', rows: 1500 },
      request_id: 'req-006'
    }
  ];

  app.get('/api/v1/audit/events', (req, res) => {
    let events = [...MOCK_AUDIT_EVENTS];
    const limit = parseInt(req.query.limit as string) || 50;
    const offset = parseInt(req.query.offset as string) || 0;

    // Apply filters
    const typeFilter = req.query.type || req.query['type[]'];
    if (typeFilter) {
      const types = Array.isArray(typeFilter) ? typeFilter : [typeFilter];
      events = events.filter(e => types.includes(e.type));
    }

    const severityFilter = req.query.severity || req.query['severity[]'];
    if (severityFilter) {
      const severities = Array.isArray(severityFilter) ? severityFilter : [severityFilter];
      events = events.filter(e => severities.includes(e.severity));
    }

    const outcomeFilter = req.query.outcome || req.query['outcome[]'];
    if (outcomeFilter) {
      const outcomes = Array.isArray(outcomeFilter) ? outcomeFilter : [outcomeFilter];
      events = events.filter(e => outcomes.includes(e.outcome));
    }

    const search = req.query.search as string;
    if (search) {
      const searchLower = search.toLowerCase();
      events = events.filter(e =>
        e.description.toLowerCase().includes(searchLower) ||
        e.action.toLowerCase().includes(searchLower) ||
        e.actor?.name?.toLowerCase().includes(searchLower)
      );
    }

    const total = events.length;
    events = events.slice(offset, offset + limit);

    res.json(jsonSuccess({
      events,
      total,
      limit,
      offset
    }));
  });

  app.get('/api/v1/audit/events/:id', (req, res) => {
    const event = MOCK_AUDIT_EVENTS.find(e => e.id === req.params.id);
    if (!event) {
      return res.status(404).json(jsonError('NOT_FOUND', 'Audit event not found'));
    }
    res.json(jsonSuccess(event));
  });

  app.get('/api/v1/audit/stats', (_req, res) => {
    res.json(jsonSuccess({
      total_events: MOCK_AUDIT_EVENTS.length,
      events_by_type: {
        'auth.success': 1,
        'auth.failure': 1,
        'authz.denied': 1,
        'detection.alert': 1,
        'user.created': 1,
        'data.export': 1
      },
      events_by_severity: {
        info: 3,
        warning: 2,
        error: 1
      },
      events_by_outcome: {
        success: 4,
        failure: 2
      },
      oldest_event: MOCK_AUDIT_EVENTS[MOCK_AUDIT_EVENTS.length - 1].timestamp,
      newest_event: MOCK_AUDIT_EVENTS[0].timestamp
    }));
  });

  app.get('/api/v1/audit/types', (_req, res) => {
    res.json(jsonSuccess({
      types: [
        'auth.success', 'auth.failure', 'auth.lockout', 'auth.unlock', 'auth.logout',
        'authz.granted', 'authz.denied',
        'detection.alert', 'detection.acknowledged',
        'user.created', 'user.modified', 'user.deleted',
        'config.changed', 'data.export', 'data.import', 'data.backup', 'admin.action'
      ]
    }));
  });

  app.get('/api/v1/audit/severities', (_req, res) => {
    res.json(jsonSuccess({
      severities: ['debug', 'info', 'warning', 'error', 'critical']
    }));
  });

  app.get('/api/v1/audit/export', (req, res) => {
    const format = req.query.format as string || 'json';

    if (format === 'cef') {
      res.setHeader('Content-Type', 'text/plain');
      res.setHeader('Content-Disposition', 'attachment; filename="audit-log.cef"');
      const cefLines = MOCK_AUDIT_EVENTS.map(e =>
        `CEF:0|Cartographus|AuditLog|1.0|${e.type}|${e.description}|${e.severity === 'critical' ? 10 : e.severity === 'error' ? 7 : e.severity === 'warning' ? 4 : 1}|src=${e.source.ip_address} act=${e.action}`
      );
      res.send(cefLines.join('\n'));
    } else {
      res.setHeader('Content-Type', 'application/json');
      res.setHeader('Content-Disposition', 'attachment; filename="audit-log.json"');
      res.json(MOCK_AUDIT_EVENTS);
    }
  });

  // =========================================================================
  // Backup Endpoints
  // =========================================================================

  app.get('/api/v1/backups', (_req, res) => {
    res.json(jsonSuccess([]));
  });

  app.get('/api/v1/backup/stats', (_req, res) => {
    res.json(jsonSuccess({
      total_backups: 0,
      total_size_bytes: 0,
      oldest_backup: null,
      newest_backup: null
    }));
  });

  app.post('/api/v1/backup', (_req, res) => {
    res.json(jsonSuccess({
      id: 'backup-new-1',
      type: 'full',
      filename: 'backup-new.tar.gz',
      is_valid: true
    }));
  });

  app.get('/api/v1/backup/retention', (_req, res) => {
    res.json(jsonSuccess({
      min_count: 3,
      max_count: 50,
      max_age_days: 90,
      keep_recent_hours: 24,
      keep_daily_for_days: 7,
      keep_weekly_for_weeks: 4,
      keep_monthly_for_months: 6
    }));
  });

  // Schedule endpoints
  app.get('/api/v1/backup/schedule', (_req, res) => {
    res.json(jsonSuccess({
      enabled: true,
      interval_hours: 24,
      preferred_hour: 2,
      backup_type: 'full',
      pre_sync_backup: false
    }));
  });

  app.put('/api/v1/backup/schedule', (req, res) => {
    res.json(jsonSuccess({
      enabled: req.body.enabled ?? true,
      interval_hours: req.body.interval_hours ?? 24,
      preferred_hour: req.body.preferred_hour ?? 2,
      backup_type: req.body.backup_type ?? 'full',
      pre_sync_backup: req.body.pre_sync_backup ?? false
    }));
  });

  app.post('/api/v1/backup/schedule/trigger', (_req, res) => {
    res.json(jsonSuccess({
      id: 'backup-triggered-1',
      type: 'full',
      filename: 'backup-triggered.tar.gz',
      size_bytes: 52428800,
      created_at: new Date().toISOString(),
      notes: 'Manually triggered scheduled backup',
      database_records: 15000,
      is_valid: true
    }));
  });

  // =========================================================================
  // Spatial Endpoints
  // =========================================================================

  app.get('/api/v1/spatial/hexagons', (_req, res) => {
    res.json(jsonSuccess(MOCK_CITIES.map((city, i) => ({
      h3_index: 617700169518678015 + i,
      latitude: city.lat,
      longitude: city.lon,
      playback_count: city.count * 2,
      unique_users: Math.ceil(city.count / 3),
      avg_completion: 75
    }))));
  });

  app.get('/api/v1/spatial/arcs', (_req, res) => {
    res.json(jsonSuccess(MOCK_CITIES.slice(0, 5).map((city, i) => ({
      source: { lat: 37.7749, lon: -122.4194 },
      target: { lat: city.lat, lon: city.lon },
      city: city.name,
      country: city.country,
      playback_count: city.count,
      bandwidth: 5000 + i * 1000
    }))));
  });

  app.get('/api/v1/spatial', (_req, res) => {
    res.json(jsonSuccess({
      type: 'FeatureCollection',
      features: generateLocations().map(loc => ({
        type: 'Feature',
        geometry: { type: 'Point', coordinates: [loc.longitude, loc.latitude] },
        properties: { city: loc.city, country: loc.country, playback_count: loc.playback_count }
      }))
    }));
  });

  // =========================================================================
  // Sync Endpoint
  // =========================================================================

  app.post('/api/v1/sync', (_req, res) => {
    res.json(jsonSuccess({
      synced: true,
      new_playbacks: 0,
      sync_duration_ms: 150
    }));
  });

  // =========================================================================
  // Insights Endpoint
  // =========================================================================

  app.get('/api/v1/insights', (_req, res) => {
    res.json(jsonSuccess({
      insights: [
        { type: 'peak_usage', message: 'Peak viewing time is 8-10 PM', severity: 'info' },
        { type: 'trending', message: 'Movie watching up 15% this week', severity: 'positive' }
      ],
      generated_at: new Date().toISOString()
    }));
  });

  // =========================================================================
  // Export Endpoints
  // =========================================================================

  app.get('/api/v1/export/csv', (_req, res) => {
    res.setHeader('Content-Type', 'text/csv');
    res.setHeader('Content-Disposition', 'attachment; filename="export.csv"');
    res.send('username,title,media_type\nJohnDoe,Inception,movie\n');
  });

  app.get('/api/v1/export/geojson', (_req, res) => {
    res.json({
      type: 'FeatureCollection',
      features: MOCK_CITIES.map(city => ({
        type: 'Feature',
        geometry: { type: 'Point', coordinates: [city.lon, city.lat] },
        properties: { city: city.name, country: city.country }
      }))
    });
  });

  // =========================================================================
  // Catch-all for unhandled API routes
  // =========================================================================

  // Express 5.x requires named wildcard parameter (not bare *)
  app.all('/api/{*path}', (req, res) => {
    console.warn(`[MOCK-SERVER] Unhandled: ${req.method} ${req.path}`);
    res.status(501).json(jsonError('NOT_IMPLEMENTED', `Mock not implemented: ${req.method} ${req.path}`));
  });

  // =========================================================================
  // Static file serving (for standalone mode - no Playwright route interception)
  // =========================================================================

  // Find the web/dist directory (works from both project root and tests/e2e)
  const possibleDistPaths = [
    path.resolve(__dirname, '../../web/dist'),      // From tests/e2e/
    path.resolve(__dirname, '../../../web/dist'),   // Alternative
    path.resolve(process.cwd(), 'web/dist'),        // From project root
  ];

  let distPath: string | null = null;
  for (const p of possibleDistPaths) {
    if (fs.existsSync(p) && fs.existsSync(path.join(p, 'index.html'))) {
      distPath = p;
      break;
    }
  }

  if (distPath) {
    console.log(`[MOCK-SERVER] Serving static files from: ${distPath}`);

    // Serve static files
    app.use(express.static(distPath, {
      // Set proper MIME types
      setHeaders: (res, filePath) => {
        if (filePath.endsWith('.js')) {
          res.setHeader('Content-Type', 'application/javascript');
        } else if (filePath.endsWith('.css')) {
          res.setHeader('Content-Type', 'text/css');
        }
      }
    }));

    // SPA fallback - serve index.html for all non-API, non-file routes
    // lgtm[js/missing-rate-limiting] - This is a test mock server, rate limiting not needed
    // codeql[js/missing-rate-limiting]: Test mock server, rate limiting unnecessary
    app.get('/{*path}', (req, res) => {
      // Don't serve index.html for files that exist
      const filePath = path.join(distPath!, req.path);

      // Prevent path traversal attacks by verifying resolved path is within distPath
      const resolvedPath = path.resolve(filePath);
      const resolvedDistPath = path.resolve(distPath!);
      if (!resolvedPath.startsWith(resolvedDistPath + path.sep) && resolvedPath !== resolvedDistPath) {
        res.status(403).send('Forbidden');
        return;
      }

      if (fs.existsSync(resolvedPath) && fs.statSync(resolvedPath).isFile()) {
        res.sendFile(resolvedPath);
      } else {
        res.sendFile(path.join(resolvedDistPath, 'index.html'));
      }
    });
  } else {
    console.warn('[MOCK-SERVER] web/dist not found - static file serving disabled');
    console.warn('[MOCK-SERVER] Searched:', possibleDistPaths.join(', '));
  }

  return app;
}

// ============================================================================
// Server Management
// ============================================================================

let serverInstance: Server | null = null;

export async function startMockServer(options: MockServerOptions): Promise<Server> {
  const app = createMockApp({ logRequests: options.logRequests });

  return new Promise((resolve, reject) => {
    serverInstance = app.listen(options.port, () => {
      console.log(`[MOCK-SERVER] Started on port ${options.port}`);
      resolve(serverInstance!);
    });

    serverInstance.on('error', (err: NodeJS.ErrnoException) => {
      if (err.code === 'EADDRINUSE') {
        console.log(`[MOCK-SERVER] Port ${options.port} already in use, assuming server is running`);
        resolve(serverInstance!);
      } else {
        reject(err);
      }
    });
  });
}

export async function stopMockServer(): Promise<void> {
  if (serverInstance) {
    return new Promise((resolve) => {
      serverInstance!.close(() => {
        console.log('[MOCK-SERVER] Stopped');
        serverInstance = null;
        resolve();
      });
    });
  }
}

// ============================================================================
// CLI Entry Point
// ============================================================================

if (require.main === module) {
  const port = parseInt(process.env.MOCK_SERVER_PORT || '3900', 10);
  startMockServer({ port, logRequests: true });
}
