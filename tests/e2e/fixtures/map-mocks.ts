// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
import { Page, Route } from '@playwright/test';

/**
 * Map Tile Mocking Utilities for E2E Tests
 *
 * Intercepts all external tile requests and returns placeholder images,
 * making tests fully offline, deterministic, and faster.
 */

// 256x256 gray PNG tile (base64 encoded) - minimal valid PNG
// This is a solid gray tile that renders quickly and consistently
const PLACEHOLDER_TILE_PNG = Buffer.from(
  'iVBORw0KGgoAAAANSUhEUgAAAQAAAAEACAYAAABccqhmAAAABHNCSVQICAgIfAhkiAAAAAlwSFlz' +
  'AAAOxAAADsQBlSsOGwAAABl0RVh0U29mdHdhcmUAd3d3Lmlua3NjYXBlLm9yZ5vuPBoAAAGESURB' +
  'VHic7cExAQAAAMKg9U9tCj+gAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA' +
  'AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA' +
  'AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA' +
  'AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA' +
  'AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA' +
  'AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA' +
  'AAAAAAAAAAAAAAAAAAAAeA3MBAABkpirjAAAAABJRU5ErkJggg==',
  'base64'
);

// Simple 1x1 transparent PNG for terrain/other tiles
const TRANSPARENT_TILE_PNG = Buffer.from(
  'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==',
  'base64'
);

// Mock geocoder response (Nominatim format)
const MOCK_GEOCODER_RESULTS = [
  {
    place_id: 123456,
    licence: 'Data © OpenStreetMap contributors',
    osm_type: 'node',
    osm_id: 123456789,
    lat: '40.7128',
    lon: '-74.0060',
    display_name: 'New York City, New York, USA',
    class: 'place',
    type: 'city',
    importance: 0.9,
    boundingbox: ['40.4774', '40.9176', '-74.2591', '-73.7004'],
  },
  {
    place_id: 234567,
    licence: 'Data © OpenStreetMap contributors',
    osm_type: 'node',
    osm_id: 234567890,
    lat: '51.5074',
    lon: '-0.1278',
    display_name: 'London, England, United Kingdom',
    class: 'place',
    type: 'city',
    importance: 0.85,
    boundingbox: ['51.2867', '51.6919', '-0.5103', '0.3340'],
  },
];

/**
 * Tile provider URL patterns to intercept
 */
const TILE_URL_PATTERNS = [
  // CartoDB raster tiles (default basemap)
  '**/basemaps.cartocdn.com/**',
  '**/*.basemaps.cartocdn.com/**',

  // OpenStreetMap tiles (fallback)
  '**/tile.openstreetmap.org/**',
  '**/*.tile.openstreetmap.org/**',

  // MapTiler tiles
  '**/api.maptiler.com/**',

  // Stadia Maps
  '**/tiles.stadiamaps.com/**',

  // ESRI World Imagery
  '**/server.arcgisonline.com/**',
  '**/services.arcgisonline.com/**',

  // AWS Terrain tiles
  '**/s3.amazonaws.com/elevation-tiles-prod/**',

  // Mapbox (in case any legacy references exist)
  '**/api.mapbox.com/**',
  '**/tiles.mapbox.com/**',

  // PMTiles (if fetched remotely)
  '**/*.pmtiles',
];

/**
 * Setup tile request mocking for a Playwright page
 *
 * Intercepts all tile requests and returns placeholder images,
 * making tests offline-capable and deterministic.
 *
 * @param page - Playwright page instance
 */
export async function setupTileMocking(page: Page): Promise<void> {
  // Mock all tile requests
  for (const pattern of TILE_URL_PATTERNS) {
    await page.route(pattern, async (route: Route) => {
      const url = route.request().url();

      // Terrain tiles get transparent PNG
      if (url.includes('elevation') || url.includes('terrain')) {
        await route.fulfill({
          status: 200,
          contentType: 'image/png',
          body: TRANSPARENT_TILE_PNG,
          headers: {
            'Cache-Control': 'public, max-age=86400',
            'Access-Control-Allow-Origin': '*',
          },
        });
      } else {
        // Regular map tiles get gray placeholder
        await route.fulfill({
          status: 200,
          contentType: 'image/png',
          body: PLACEHOLDER_TILE_PNG,
          headers: {
            'Cache-Control': 'public, max-age=86400',
            'Access-Control-Allow-Origin': '*',
          },
        });
      }
    });
  }
}

/**
 * Setup geocoder API mocking
 *
 * Intercepts Nominatim geocoder requests and returns mock results.
 *
 * @param page - Playwright page instance
 */
export async function setupGeocoderMocking(page: Page): Promise<void> {
  await page.route('**/nominatim.openstreetmap.org/**', async (route: Route) => {
    const url = route.request().url();

    if (url.includes('/search')) {
      // Return mock search results
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(MOCK_GEOCODER_RESULTS),
        headers: {
          'Access-Control-Allow-Origin': '*',
        },
      });
    } else if (url.includes('/reverse')) {
      // Return single mock result for reverse geocoding
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(MOCK_GEOCODER_RESULTS[0]),
        headers: {
          'Access-Control-Allow-Origin': '*',
        },
      });
    } else {
      // Pass through other requests
      await route.continue();
    }
  });
}

/**
 * Setup all map-related mocking (tiles + geocoder)
 *
 * Call this in beforeEach to make map tests fully offline.
 *
 * @param page - Playwright page instance
 *
 * @example
 * ```typescript
 * test.beforeEach(async ({ page }) => {
 *   await setupMapMocking(page);
 *   // ... rest of setup
 * });
 * ```
 */
export async function setupMapMocking(page: Page): Promise<void> {
  await setupTileMocking(page);
  await setupGeocoderMocking(page);
}

/**
 * Verify that tile mocking is working
 *
 * Useful for debugging - logs intercepted tile requests.
 *
 * @param page - Playwright page instance
 */
export async function verifyTileMocking(page: Page): Promise<{ intercepted: number; passedThrough: number }> {
  let intercepted = 0;
  let passedThrough = 0;

  page.on('response', (response) => {
    const url = response.url();
    const isTileUrl = TILE_URL_PATTERNS.some(pattern => {
      const regex = new RegExp(pattern.replace(/\*\*/g, '.*').replace(/\*/g, '[^/]*'));
      return regex.test(url);
    });

    if (isTileUrl) {
      if (response.status() === 200 && response.headers()['x-mocked'] === 'true') {
        intercepted++;
      } else {
        passedThrough++;
        console.warn(`[Tile Mocking] Request passed through: ${url}`);
      }
    }
  });

  return { intercepted, passedThrough };
}
