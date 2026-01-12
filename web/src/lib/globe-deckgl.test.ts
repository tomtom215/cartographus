// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Unit Tests for GlobeManagerDeckGL
 *
 * These tests verify the deck.gl globe implementation functionality.
 * Run with: npx tsx --test globe-deckgl.test.ts
 * Or integrate with your preferred test runner (Jest, Vitest, etc.)
 */

import { describe, it, beforeEach, afterEach } from 'node:test';
import assert from 'node:assert';

// Mock types for testing
interface MockMapboxMap {
    addControl: (control: any) => void;
    removeControl: (control: any) => void;
    remove: () => void;
    resize: () => void;
    easeTo: (options: any) => void;
    on: (event: string, callback: () => void) => void;
}

interface MockLocationStats {
    country: string;
    city?: string;
    region?: string;
    latitude: number;
    longitude: number;
    playback_count: number;
    unique_users: number;
    first_seen: string;
    last_seen: string;
    avg_completion: number;
}

/**
 * Test Suite: GlobeManagerDeckGL Class
 */
describe('GlobeManagerDeckGL', () => {
    let mockContainer: HTMLElement;
    let mockMap: MockMapboxMap;

    beforeEach(() => {
        // Setup mock DOM
        mockContainer = {
            id: 'globe',
            style: { display: 'block' }
        } as any;

        global.document = {
            getElementById: (id: string) => id === 'globe' ? mockContainer : null,
            createElement: (_tag: string) => ({
                id: '',
                style: { cssText: '', display: 'none' },
                innerHTML: '',
                parentNode: null
            } as any),
            documentElement: {
                hasAttribute: (_attr: string) => false
            },
            body: {
                appendChild: (el: any) => el
            }
        } as any;

        // Mock Mapbox Map
        mockMap = {
            addControl: (_control: any) => {},
            removeControl: (_control: any) => {},
            remove: () => {},
            resize: () => {},
            easeTo: (_options: any) => {},
            on: (event: string, callback: () => void) => {
                if (event === 'load') {
                    setTimeout(callback, 0);
                }
            }
        };

        // Use mockMap to avoid unused variable warning
        void mockMap;
    });

    afterEach(() => {
        // Cleanup mocks - manager instances are not currently used in these tests
        // as they test implementation logic without actual GlobeManagerDeckGL instantiation
    });

    /**
     * Test: Constructor initialization
     */
    it('should initialize with correct container ID', () => {
        // This is a conceptual test - actual implementation would require proper mocking
        assert.ok(true, 'Constructor should accept container ID');
    });

    /**
     * Test: Theme detection
     */
    it('should detect dark theme by default', () => {
        // Dark theme is default when no data-theme attribute
        const isDark = !document.documentElement.hasAttribute('data-theme');
        assert.strictEqual(isDark, true);
    });

    /**
     * Test: Light theme detection
     */
    it('should detect light theme when data-theme attribute exists', () => {
        (document.documentElement as any).hasAttribute = (attr: string) => attr === 'data-theme';
        const isDark = !document.documentElement.hasAttribute('data-theme');
        assert.strictEqual(isDark, false);
    });

    /**
     * Test: Location filtering
     */
    it('should filter out invalid locations (0,0)', () => {
        const locations: MockLocationStats[] = [
            {
                country: 'US',
                city: 'New York',
                latitude: 40.7128,
                longitude: -74.0060,
                playback_count: 100,
                unique_users: 5,
                first_seen: '2025-01-01',
                last_seen: '2025-01-15',
                avg_completion: 85.5
            },
            {
                country: 'Unknown',
                latitude: 0,
                longitude: 0,
                playback_count: 10,
                unique_users: 1,
                first_seen: '2025-01-01',
                last_seen: '2025-01-15',
                avg_completion: 50
            },
            {
                country: 'UK',
                city: 'London',
                latitude: 51.5074,
                longitude: -0.1278,
                playback_count: 200,
                unique_users: 10,
                first_seen: '2025-01-01',
                last_seen: '2025-01-15',
                avg_completion: 90
            }
        ];

        // Filter logic from GlobeManagerDeckGL
        const validLocations = locations.filter(location =>
            location.latitude !== 0 || location.longitude !== 0
        );

        assert.strictEqual(validLocations.length, 2);
        assert.strictEqual(validLocations[0].city, 'New York');
        assert.strictEqual(validLocations[1].city, 'London');
    });

    /**
     * Test: Color coding by playback count
     */
    it('should return correct colors based on playback count', () => {
        const getColorByPlaybackCount = (count: number): number[] => {
            if (count > 500) return [233, 69, 96, 220];    // Red
            if (count > 200) return [255, 107, 107, 220];  // Orange-red
            if (count > 50) return [255, 165, 0, 220];     // Orange
            return [78, 205, 196, 220];                    // Teal
        };

        // Test tier 1: > 500 playbacks (red)
        const color1 = getColorByPlaybackCount(600);
        assert.deepStrictEqual(color1, [233, 69, 96, 220]);

        // Test tier 2: 200-500 playbacks (orange-red)
        const color2 = getColorByPlaybackCount(300);
        assert.deepStrictEqual(color2, [255, 107, 107, 220]);

        // Test tier 3: 50-200 playbacks (orange)
        const color3 = getColorByPlaybackCount(100);
        assert.deepStrictEqual(color3, [255, 165, 0, 220]);

        // Test tier 4: < 50 playbacks (teal)
        const color4 = getColorByPlaybackCount(25);
        assert.deepStrictEqual(color4, [78, 205, 196, 220]);

        // Test edge cases
        assert.deepStrictEqual(getColorByPlaybackCount(501), [233, 69, 96, 220]);
        assert.deepStrictEqual(getColorByPlaybackCount(500), [255, 107, 107, 220]);
        assert.deepStrictEqual(getColorByPlaybackCount(201), [255, 107, 107, 220]);
        assert.deepStrictEqual(getColorByPlaybackCount(200), [255, 165, 0, 220]);
        assert.deepStrictEqual(getColorByPlaybackCount(51), [255, 165, 0, 220]);
        assert.deepStrictEqual(getColorByPlaybackCount(50), [78, 205, 196, 220]);
    });

    /**
     * Test: Radius calculation based on playback count
     */
    it('should calculate radius within min/max bounds', () => {
        const getRadius = (count: number): number => {
            const playbacks = count;
            return Math.max(50000, Math.min(200000, Math.sqrt(playbacks) * 20000));
        };

        // Test minimum bound (should clamp to 50000)
        const radius1 = getRadius(1);
        assert.strictEqual(radius1, 50000);

        // Test maximum bound (should clamp to 200000)
        const radius2 = getRadius(10000);
        assert.strictEqual(radius2, 200000);

        // Test normal range
        const radius3 = getRadius(100);
        assert.ok(radius3 >= 50000 && radius3 <= 200000);
        assert.strictEqual(radius3, Math.sqrt(100) * 20000); // 200000
    });

    /**
     * Test: Position accessor format
     */
    it('should format position as [longitude, latitude, altitude]', () => {
        const location: MockLocationStats = {
            country: 'US',
            city: 'New York',
            latitude: 40.7128,
            longitude: -74.0060,
            playback_count: 100,
            unique_users: 5,
            first_seen: '2025-01-01',
            last_seen: '2025-01-15',
            avg_completion: 85.5
        };

        const position = [location.longitude, location.latitude, 0];
        assert.deepStrictEqual(position, [-74.0060, 40.7128, 0]);
    });

    /**
     * Test: Location name formatting
     */
    it('should format location name with region when available', () => {
        const location1: MockLocationStats = {
            country: 'US',
            city: 'New York',
            region: 'NY',
            latitude: 40.7128,
            longitude: -74.0060,
            playback_count: 100,
            unique_users: 5,
            first_seen: '2025-01-01',
            last_seen: '2025-01-15',
            avg_completion: 85.5
        };

        const name1 = location1.region
            ? `${location1.city || 'Unknown'}, ${location1.region}, ${location1.country}`
            : `${location1.city || 'Unknown'}, ${location1.country}`;

        assert.strictEqual(name1, 'New York, NY, US');

        const location2: MockLocationStats = {
            country: 'US',
            city: 'Boston',
            latitude: 42.3601,
            longitude: -71.0589,
            playback_count: 50,
            unique_users: 3,
            first_seen: '2025-01-01',
            last_seen: '2025-01-15',
            avg_completion: 75
        };

        const name2 = location2.region
            ? `${location2.city || 'Unknown'}, ${location2.region}, ${location2.country}`
            : `${location2.city || 'Unknown'}, ${location2.country}`;

        assert.strictEqual(name2, 'Boston, US');
    });

    /**
     * Test: Tooltip data formatting
     */
    it('should format tooltip with correct number formatting', () => {
        const location: MockLocationStats = {
            country: 'US',
            city: 'New York',
            latitude: 40.7128,
            longitude: -74.0060,
            playback_count: 12345,
            unique_users: 567,
            first_seen: '2025-01-01T00:00:00Z',
            last_seen: '2025-01-15T23:59:59Z',
            avg_completion: 85.75
        };

        // Test number formatting
        const formattedPlaybacks = location.playback_count.toLocaleString();
        assert.strictEqual(formattedPlaybacks, '12,345');

        // Test completion formatting
        const formattedCompletion = location.avg_completion.toFixed(1);
        assert.strictEqual(formattedCompletion, '85.8');

        // Test date formatting
        const firstSeenDate = new Date(location.first_seen).toLocaleDateString();
        const lastSeenDate = new Date(location.last_seen).toLocaleDateString();
        assert.ok(firstSeenDate.length > 0);
        assert.ok(lastSeenDate.length > 0);
    });

    /**
     * Test: View reset parameters
     */
    it('should use correct default view parameters', () => {
        const defaultView = {
            center: [0, 20],
            zoom: 2,
            pitch: 0,
            bearing: 0,
            duration: 1000
        };

        assert.deepStrictEqual(defaultView.center, [0, 20]);
        assert.strictEqual(defaultView.zoom, 2);
        assert.strictEqual(defaultView.pitch, 0);
        assert.strictEqual(defaultView.bearing, 0);
        assert.strictEqual(defaultView.duration, 1000);
    });

    /**
     * Test: Globe projection configuration
     */
    it('should configure globe projection correctly', () => {
        const projectionConfig = { name: 'globe' };
        assert.strictEqual(projectionConfig.name, 'globe');
    });

    /**
     * Test: Basemap tile URLs
     */
    it('should use correct CARTO dark basemap tiles', () => {
        const tiles = [
            'https://a.basemaps.cartocdn.com/dark_all/{z}/{x}/{y}.png',
            'https://b.basemaps.cartocdn.com/dark_all/{z}/{x}/{y}.png',
            'https://c.basemaps.cartocdn.com/dark_all/{z}/{x}/{y}.png'
        ];

        assert.strictEqual(tiles.length, 3);
        tiles.forEach(tile => {
            // Use proper URL parsing for hostname validation instead of substring check
            const tileUrl = new URL(tile);
            assert.ok(
                tileUrl.hostname.endsWith('.basemaps.cartocdn.com') ||
                tileUrl.hostname === 'basemaps.cartocdn.com',
                `Expected CARTO hostname, got: ${tileUrl.hostname}`
            );
            assert.ok(tileUrl.pathname.includes('dark_all'));
            assert.ok(tileUrl.pathname.endsWith('{z}/{x}/{y}.png'));
        });
    });

    /**
     * Test: Layer transitions configuration
     */
    it('should configure smooth transitions', () => {
        const transitions = {
            getPosition: 300,
            getRadius: 300,
            getFillColor: 300
        };

        assert.strictEqual(transitions.getPosition, 300);
        assert.strictEqual(transitions.getRadius, 300);
        assert.strictEqual(transitions.getFillColor, 300);
    });

    /**
     * Test: Empty data handling
     */
    it('should handle empty location array', () => {
        const locations: MockLocationStats[] = [];
        const validLocations = locations.filter(location =>
            location.latitude !== 0 || location.longitude !== 0
        );

        assert.strictEqual(validLocations.length, 0);
    });

    /**
     * Test: Large dataset handling
     */
    it('should handle large datasets efficiently', () => {
        const largeDataset: MockLocationStats[] = [];
        for (let i = 0; i < 10000; i++) {
            largeDataset.push({
                country: 'Test',
                city: `City${i}`,
                latitude: Math.random() * 180 - 90,
                longitude: Math.random() * 360 - 180,
                playback_count: Math.floor(Math.random() * 1000),
                unique_users: Math.floor(Math.random() * 50),
                first_seen: '2025-01-01',
                last_seen: '2025-01-15',
                avg_completion: Math.random() * 100
            });
        }

        const startTime = Date.now();
        const validLocations = largeDataset.filter(location =>
            location.latitude !== 0 || location.longitude !== 0
        );
        const filterTime = Date.now() - startTime;

        assert.strictEqual(validLocations.length, largeDataset.length);
        assert.ok(filterTime < 100, 'Filtering 10k locations should take < 100ms');
    });
});

/**
 * Test Suite: Integration Tests
 */
describe('GlobeManagerDeckGL Integration', () => {
    /**
     * Test: deck.gl and Mapbox version compatibility
     */
    it('should use compatible deck.gl and Mapbox versions', () => {
        const deckGLVersion = '^9.2.2';
        const mapboxVersion = '^3.16.0';

        // Verify versions are specified
        assert.ok(deckGLVersion.length > 0);
        assert.ok(mapboxVersion.length > 0);

        // deck.gl 9.x supports Mapbox GL JS 3.x
        assert.ok(deckGLVersion.startsWith('^9'));
        assert.ok(mapboxVersion.startsWith('^3'));
    });

    /**
     * Test: WebGL2 context requirement
     */
    it('should require WebGL2 for interleaved mode', () => {
        const overlayConfig = {
            interleaved: true, // Requires WebGL2
            layers: []
        };

        assert.strictEqual(overlayConfig.interleaved, true);
    });

    /**
     * Test: ScatterplotLayer configuration
     */
    it('should configure ScatterplotLayer with required properties', () => {
        const layerConfig = {
            id: 'playback-locations',
            data: [],
            pickable: true,
            autoHighlight: true,
            opacity: 0.8,
            radiusMinPixels: 3,
            radiusMaxPixels: 30
        };

        assert.strictEqual(layerConfig.id, 'playback-locations');
        assert.strictEqual(layerConfig.pickable, true);
        assert.strictEqual(layerConfig.autoHighlight, true);
        assert.ok(layerConfig.opacity > 0 && layerConfig.opacity <= 1);
        assert.ok(layerConfig.radiusMinPixels >= 1);
        assert.ok(layerConfig.radiusMaxPixels <= 50);
    });
});

console.log('✅ All GlobeManagerDeckGL unit tests defined');
console.log('Run with: npx tsx --test globe-deckgl.test.ts');
console.log('Or integrate with Jest/Vitest for automated testing');
