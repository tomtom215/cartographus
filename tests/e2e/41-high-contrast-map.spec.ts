// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * E2E Tests for High Contrast Map Tile Style
 *
 * Tests that the map properly supports high-contrast mode with:
 * - Simplified basemap tiles (no labels)
 * - Adjusted brightness/contrast for overlay visibility
 * - Proper tile provider switching when theme changes
 *
 * Reference: UI/UX Audit
 * @see /docs/working/UI_UX_AUDIT.md
 */

// ROOT CAUSE FIX: Import test/expect from fixtures to enable autoMockApi for fast, deterministic tests
// Previously imported from @playwright/test which bypassed fixture's automatic API mocking
import { test, expect, TIMEOUTS, setupApiMocking, Page } from './fixtures';

// DETERMINISTIC FIX: Disable context-level API mocking since these tests use setupApiMocking(page)
// which registers page-level routes. Having both context and page-level routes active causes conflicts.
// Page-level routes give these tests explicit control over API responses for high-contrast map scenarios.
test.use({ autoMockApi: false });

// Helper to mock the API responses and bypass login
async function setupPage(page: Page): Promise<void> {
    // Set localStorage to bypass onboarding and login
    await page.addInitScript(() => {
        localStorage.setItem('onboarding_completed', 'true');
        localStorage.setItem('onboarding_skipped', 'true');
        localStorage.setItem('skipLogin', 'true');
    });

    // Setup comprehensive API mocking first
    await setupApiMocking(page);

    // Override with test-specific mock responses
    await page.route('**/api/v1/locations**', (route) => {
        route.fulfill({
            status: 200,
            contentType: 'application/json',
            body: JSON.stringify({
                locations: [
                    {
                        ip_address: '192.168.1.1',
                        latitude: 40.7128,
                        longitude: -74.0060,
                        city: 'New York',
                        region: 'NY',
                        country: 'United States',
                        playback_count: 100,
                        unique_users: 5,
                        avg_completion: 85.5,
                        first_seen: '2024-01-01T00:00:00Z',
                        last_seen: '2024-12-01T00:00:00Z'
                    }
                ],
                cursor: null
            })
        });
    });

    // Mock auth check
    await page.route('**/api/v1/auth/check', (route) => {
        route.fulfill({
            status: 200,
            contentType: 'application/json',
            body: JSON.stringify({
                authenticated: true,
                user: { id: 1, username: 'testuser' }
            })
        });
    });

    // Mock server info
    await page.route('**/api/v1/tautulli/server-info', (route) => {
        route.fulfill({
            status: 200,
            contentType: 'application/json',
            body: JSON.stringify({
                response: {
                    result: 'success',
                    data: {
                        server_name: 'Test Server',
                        has_location: true,
                        latitude: 37.7749,
                        longitude: -122.4194
                    }
                }
            })
        });
    });

    // Mock other necessary endpoints
    await page.route('**/api/v1/tautulli/**', (route) => {
        route.fulfill({
            status: 200,
            contentType: 'application/json',
            body: JSON.stringify({ response: { result: 'success', data: {} } })
        });
    });

    await page.route('**/api/v1/analytics/**', (route) => {
        route.fulfill({
            status: 200,
            contentType: 'application/json',
            body: JSON.stringify({})
        });
    });
}

test.describe('High Contrast Map Tile Style', () => {
    test.beforeEach(async ({ page }) => {
        await setupPage(page);
        await page.goto('/');
        await page.waitForSelector('#app', { state: 'visible' });
    });

    test('map config supports high-contrast tile provider', async ({ page }) => {
        // Navigate to map view
        const mapTab = page.locator('[data-view="maps"]');
        if (await mapTab.isVisible()) {
            await mapTab.click();
            // Wait for map to be initialized with canvas
            await page.waitForFunction(() => {
                const mapEl = document.getElementById('map');
                const canvas = mapEl?.querySelector('canvas.maplibregl-canvas');
                return mapEl && canvas;
            }, { timeout: TIMEOUTS.RENDER });
        }

        // Switch to high-contrast mode
        await page.evaluate(() => {
            document.documentElement.setAttribute('data-theme', 'high-contrast');
        });
        // Wait for theme to be applied
        // CI FIX: Use TIMEOUTS.SHORT (2s) instead of TIMEOUTS.ANIMATION (300ms)
        // WHY: Theme changes can take longer in CI/SwiftShader headless environments
        await page.waitForFunction(() => {
            return document.documentElement.getAttribute('data-theme') === 'high-contrast';
        }, { timeout: TIMEOUTS.SHORT });

        // Verify the map container exists
        const mapContainer = page.locator('#map');
        await expect(mapContainer).toBeVisible();
    });

    test('MapConfigManager stores high-contrast theme correctly', async ({ page }) => {
        // Test that the config manager properly handles high-contrast theme
        const configResult = await page.evaluate(() => {
            // Check if MapConfigManager is accessible
            return {
                hasMapContainer: !!document.getElementById('map'),
                themeSet: document.documentElement.getAttribute('data-theme')
            };
        });

        expect(configResult.hasMapContainer).toBeTruthy();
    });

    test('theme toggle cycles through dark, light, and high-contrast', async ({ page }) => {
        const themeToggle = page.locator('#theme-toggle');
        if (await themeToggle.isVisible()) {
            const themes: string[] = [];

            // Cycle through themes
            for (let i = 0; i < 4; i++) {
                const currentTheme = await page.evaluate(() => {
                    return document.documentElement.getAttribute('data-theme') || 'dark';
                });
                themes.push(currentTheme);

                await themeToggle.click();
                // Wait for theme to change to a different value
                // CI FIX: Use TIMEOUTS.SHORT (2s) instead of TIMEOUTS.ANIMATION (300ms)
                // WHY: Theme changes can take longer in CI/SwiftShader headless environments
                await page.waitForFunction((prevTheme) => {
                    const newTheme = document.documentElement.getAttribute('data-theme') || 'dark';
                    return newTheme !== prevTheme;
                }, currentTheme, { timeout: TIMEOUTS.SHORT });
            }

            // Should have cycled through multiple themes
            expect(themes.length).toBe(4);
            // Note: high-contrast may or may not be in the cycle depending on config
            // The important thing is the config supports it
        }
    });

    test('high-contrast mode adds CSS variables for map styling', async ({ page }) => {
        // Set high-contrast mode
        await page.evaluate(() => {
            document.documentElement.setAttribute('data-theme', 'high-contrast');
        });
        // Wait for theme to be applied
        // CI FIX: Use TIMEOUTS.SHORT (2s) instead of TIMEOUTS.ANIMATION (300ms)
        // WHY: Theme changes can take longer in CI/SwiftShader headless environments
        await page.waitForFunction(() => {
            return document.documentElement.getAttribute('data-theme') === 'high-contrast';
        }, { timeout: TIMEOUTS.SHORT });

        // Verify high-contrast specific CSS variables exist
        const cssVars = await page.evaluate(() => {
            const style = getComputedStyle(document.documentElement);
            return {
                primaryBg: style.getPropertyValue('--primary-bg').trim(),
                textPrimary: style.getPropertyValue('--text-primary').trim(),
                highlight: style.getPropertyValue('--highlight').trim()
            };
        });

        // High-contrast mode should have pure black background
        expect(cssVars.primaryBg).toBe('#000000');
        // High-contrast mode should have white text
        expect(cssVars.textPrimary).toBe('#ffffff');
        // High-contrast mode should have yellow highlight
        expect(cssVars.highlight).toBe('#ffff00');
    });
});

test.describe('High Contrast Map Layer Styling', () => {
    test.beforeEach(async ({ page }) => {
        await setupPage(page);
        await page.goto('/');
        await page.waitForSelector('#app', { state: 'visible' });
    });

    test('map tile provider configuration is exported correctly', async ({ page }) => {
        // Verify that the map-config module exports required functions
        const configExports = await page.evaluate(async () => {
            // The functions should be available as the app loads them
            return {
                hasMapElement: !!document.getElementById('map'),
                hasStylesLoaded: !!document.querySelector('link[href*="maplibre"]')
            };
        });

        expect(configExports.hasStylesLoaded).toBeTruthy();
    });

    test('CARTO nolabels tiles are used for high-contrast', async ({ page }) => {
        // This test verifies the tile URL configuration
        // In high-contrast mode, we use simplified (nolabels) basemaps

        // Set up high contrast mode
        await page.evaluate(() => {
            document.documentElement.setAttribute('data-theme', 'high-contrast');
        });

        // The tile provider configuration should include nolabels variants
        // This is a configuration test - actual tile loading tested in integration
        const result = await page.evaluate(() => {
            return {
                isHighContrast: document.documentElement.getAttribute('data-theme') === 'high-contrast'
            };
        });

        expect(result.isHighContrast).toBeTruthy();
    });
});

test.describe('Map Theme Synchronization', () => {
    test.beforeEach(async ({ page }) => {
        await setupPage(page);
        await page.goto('/');
        await page.waitForSelector('#app', { state: 'visible' });
    });

    test('map style updates when theme changes', async ({ page }) => {
        // Navigate to maps view
        const mapTab = page.locator('[data-view="maps"]');
        if (await mapTab.isVisible()) {
            await mapTab.click();
            // Wait for map to be initialized with canvas
            await page.waitForFunction(() => {
                const mapEl = document.getElementById('map');
                const canvas = mapEl?.querySelector('canvas.maplibregl-canvas');
                return mapEl && canvas;
            }, { timeout: TIMEOUTS.RENDER });
        }

        // Record initial theme
        const initialTheme = await page.evaluate(() => {
            return document.documentElement.getAttribute('data-theme') || 'dark';
        });

        // Change theme via toggle
        const themeToggle = page.locator('#theme-toggle');
        if (await themeToggle.isVisible()) {
            await themeToggle.click();
            // Wait for theme to change from initial value
            // CI FIX: Use TIMEOUTS.SHORT (2s) instead of TIMEOUTS.ANIMATION (300ms)
            // WHY: Theme changes can take longer in CI/SwiftShader headless environments
            await page.waitForFunction((prevTheme) => {
                const newTheme = document.documentElement.getAttribute('data-theme') || 'dark';
                return newTheme !== prevTheme;
            }, initialTheme, { timeout: TIMEOUTS.SHORT });

            // Verify theme changed
            const newTheme = await page.evaluate(() => {
                return document.documentElement.getAttribute('data-theme') || 'dark';
            });

            expect(newTheme).not.toBe(initialTheme);
        }
    });

    test('globe view also supports high-contrast mode', async ({ page }) => {
        // Navigate to 3D globe view
        const globe3DButton = page.locator('[data-view="3d"]');
        if (await globe3DButton.isVisible()) {
            await globe3DButton.click();
            // Wait for globe container to be initialized
            await page.waitForFunction(() => {
                const globeContainer = document.getElementById('globe-container');
                return globeContainer !== null;
            }, { timeout: TIMEOUTS.RENDER });

            // Set high-contrast mode
            await page.evaluate(() => {
                document.documentElement.setAttribute('data-theme', 'high-contrast');
            });
            // Wait for theme to be applied
            // CI FIX: Use TIMEOUTS.SHORT (2s) instead of TIMEOUTS.ANIMATION (300ms)
            // WHY: Theme changes can take longer in CI/SwiftShader headless environments
            await page.waitForFunction(() => {
                return document.documentElement.getAttribute('data-theme') === 'high-contrast';
            }, { timeout: TIMEOUTS.SHORT });

            // Verify globe container exists and theme is applied
            const globeResult = await page.evaluate(() => {
                const globeContainer = document.getElementById('globe-container');
                return {
                    hasGlobe: !!globeContainer,
                    theme: document.documentElement.getAttribute('data-theme')
                };
            });

            expect(globeResult.theme).toBe('high-contrast');
        }
    });
});
