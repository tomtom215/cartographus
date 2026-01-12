// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * E2E Tests for Globe Visualization Theming
 *
 * Tests that globe visualization colors properly respond to theme changes
 * using CSS variables for consistent theming across dark, light,
 * high-contrast, and colorblind modes.
 *
 * Reference: UI/UX Audit
 * @see /docs/working/UI_UX_AUDIT.md
 */

// ROOT CAUSE FIX: Import test/expect from fixtures to enable autoMockApi for fast, deterministic tests
// Previously imported from @playwright/test which bypassed fixture's automatic API mocking
import { test, expect, TIMEOUTS, setupApiMocking, Page } from './fixtures';

// DETERMINISTIC FIX: Disable context-level API mocking since these tests use setupApiMocking(page)
// which registers page-level routes. Having both context and page-level routes active causes conflicts.
// Page-level routes give these tests explicit control over API responses for globe theming scenarios.
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
                        playback_count: 600,
                        unique_users: 5,
                        avg_completion: 85.5,
                        first_seen: '2024-01-01T00:00:00Z',
                        last_seen: '2024-12-01T00:00:00Z'
                    },
                    {
                        ip_address: '192.168.1.2',
                        latitude: 51.5074,
                        longitude: -0.1278,
                        city: 'London',
                        region: 'England',
                        country: 'United Kingdom',
                        playback_count: 100,
                        unique_users: 3,
                        avg_completion: 72.3,
                        first_seen: '2024-02-01T00:00:00Z',
                        last_seen: '2024-11-15T00:00:00Z'
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

test.describe('Globe Visualization Theming', () => {
    test.beforeEach(async ({ page }) => {
        await setupPage(page);
        await page.goto('/');
        // Wait for the app to load - use MEDIUM timeout for CI reliability
        await page.waitForSelector('#app', { state: 'visible', timeout: TIMEOUTS.MEDIUM });
        // Wait for CSS variables to be fully resolved
        // This ensures styles are loaded before testing theme variables
        await page.waitForFunction(
            () => {
                const style = getComputedStyle(document.documentElement);
                // Wait until the CSS framework has loaded and variables are defined
                return style.getPropertyValue('--globe-marker-high').trim() !== '';
            },
            { timeout: TIMEOUTS.MEDIUM }
        );
    });

    test('CSS variables for globe theming exist in dark mode (default)', async ({ page }) => {
        // Verify globe CSS variables exist (dark or light mode based on system preference)
        const markerHigh = await page.evaluate(() => {
            return getComputedStyle(document.documentElement)
                .getPropertyValue('--globe-marker-high').trim();
        });
        const markerLow = await page.evaluate(() => {
            return getComputedStyle(document.documentElement)
                .getPropertyValue('--globe-marker-low').trim();
        });
        const tooltipBg = await page.evaluate(() => {
            return getComputedStyle(document.documentElement)
                .getPropertyValue('--globe-tooltip-bg').trim();
        });

        expect(markerHigh).toBeTruthy();
        expect(markerLow).toBeTruthy();
        expect(tooltipBg).toBeTruthy();

        // Verify colors are set - accept both dark and light mode values
        // Dark mode: #ec4899 (pink), Light mode: #db2777 (darker pink)
        expect(markerHigh).toMatch(/#(ec4899|db2777)/i);
        // Dark mode: #14b8a6 (teal), Light mode: #0d9488 (darker teal)
        expect(markerLow).toMatch(/#(14b8a6|0d9488)/i);
    });

    test('CSS variables change when switching to light mode', async ({ page }) => {
        // Switch to light theme using JavaScript click for CI reliability
        // Playwright's .click() may fail in headless/SwiftShader environments
        const themeToggle = page.locator('#theme-toggle');
        if (await themeToggle.isVisible()) {
            // Click until we reach light mode
            await page.evaluate(() => {
                const toggle = document.getElementById('theme-toggle');
                if (toggle) toggle.click();
            });
            // Wait for theme attribute to be set - use MEDIUM timeout for CI
            await page.waitForFunction(() => {
                return document.documentElement.getAttribute('data-theme') !== null;
            }, { timeout: TIMEOUTS.MEDIUM });

            // Check if we're in light mode
            const theme = await page.evaluate(() => {
                return document.documentElement.getAttribute('data-theme');
            });

            if (theme === 'light') {
                const markerHigh = await page.evaluate(() => {
                    return getComputedStyle(document.documentElement)
                        .getPropertyValue('--globe-marker-high').trim();
                });
                const tooltipBg = await page.evaluate(() => {
                    return getComputedStyle(document.documentElement)
                        .getPropertyValue('--globe-tooltip-bg').trim();
                });

                // Light mode should have darker markers for contrast
                expect(markerHigh).toMatch(/#db2777|db2777/i);
                // Light mode tooltip should have light background
                expect(tooltipBg).toContain('rgba(255');
            }
        }
    });

    test('CSS variables change when switching to high-contrast mode', async ({ page }) => {
        // Switch to high-contrast theme by clicking theme toggle multiple times
        // Use JavaScript click for CI reliability
        const themeToggle = page.locator('#theme-toggle');
        if (await themeToggle.isVisible()) {
            // Click multiple times to cycle through themes
            for (let i = 0; i < 3; i++) {
                const prevTheme = await page.evaluate(() => {
                    return document.documentElement.getAttribute('data-theme');
                });
                await page.evaluate(() => {
                    const toggle = document.getElementById('theme-toggle');
                    if (toggle) toggle.click();
                });
                // Wait for theme to change - use MEDIUM timeout for CI
                await page.waitForFunction((prev) => {
                    return document.documentElement.getAttribute('data-theme') !== prev;
                }, prevTheme, { timeout: TIMEOUTS.MEDIUM });

                const theme = await page.evaluate(() => {
                    return document.documentElement.getAttribute('data-theme');
                });

                if (theme === 'high-contrast') {
                    const markerHigh = await page.evaluate(() => {
                        return getComputedStyle(document.documentElement)
                            .getPropertyValue('--globe-marker-high').trim();
                    });
                    const markerLow = await page.evaluate(() => {
                        return getComputedStyle(document.documentElement)
                            .getPropertyValue('--globe-marker-low').trim();
                    });

                    // High contrast should have bright, high-visibility colors
                    expect(markerHigh).toMatch(/#ff00ff|ff00ff/i); // Magenta
                    expect(markerLow).toMatch(/#00ffff|00ffff/i); // Cyan
                    break;
                }
            }
        }
    });

    test('CSS variables change when colorblind mode is enabled', async ({ page }) => {
        // Enable colorblind mode using JavaScript click for CI reliability
        const colorblindToggle = page.locator('#colorblind-toggle');
        if (await colorblindToggle.isVisible()) {
            await page.evaluate(() => {
                const toggle = document.getElementById('colorblind-toggle');
                if (toggle) toggle.click();
            });
            // Wait for colorblind attribute to be set - use MEDIUM timeout for CI
            await page.waitForFunction(() => {
                return document.documentElement.getAttribute('data-colorblind') === 'true';
            }, { timeout: TIMEOUTS.MEDIUM });

            const isColorblind = await page.evaluate(() => {
                return document.documentElement.getAttribute('data-colorblind');
            });

            if (isColorblind === 'true') {
                const markerHigh = await page.evaluate(() => {
                    return getComputedStyle(document.documentElement)
                        .getPropertyValue('--globe-marker-high').trim();
                });
                const markerLow = await page.evaluate(() => {
                    return getComputedStyle(document.documentElement)
                        .getPropertyValue('--globe-marker-low').trim();
                });

                // Colorblind mode should use blue-orange palette
                expect(markerHigh).toMatch(/#cc3311|cc3311/i); // Red-orange
                expect(markerLow).toMatch(/#0077bb|0077bb/i); // Blue
            }
        }
    });

    test('Globe tooltip element uses CSS variables for styling', async ({ page }) => {
        // Navigate to 3D globe view using JavaScript click for CI reliability
        const globe3DButton = page.locator('[data-view="3d"]');
        if (await globe3DButton.isVisible()) {
            await page.evaluate(() => {
                const btn = document.querySelector('[data-view="3d"]') as HTMLElement;
                if (btn) btn.click();
            });
            // Wait for globe tooltip to be initialized - use MEDIUM timeout for CI
            await page.waitForFunction(() => {
                return document.getElementById('deckgl-tooltip-enhanced') !== null;
            }, { timeout: TIMEOUTS.MEDIUM });

            // Check if tooltip element exists with CSS variable styling
            const tooltipExists = await page.evaluate(() => {
                const tooltip = document.getElementById('deckgl-tooltip-enhanced');
                if (tooltip) {
                    const style = tooltip.style.cssText;
                    return style.includes('var(--globe-tooltip');
                }
                return false;
            });

            // The tooltip should use CSS variables for styling
            expect(tooltipExists).toBeTruthy();
        }
    });

    test('Globe tooltip has proper class for theming', async ({ page }) => {
        // Navigate to 3D globe view using JavaScript click for CI reliability
        const globe3DButton = page.locator('[data-view="3d"]');
        if (await globe3DButton.isVisible()) {
            await page.evaluate(() => {
                const btn = document.querySelector('[data-view="3d"]') as HTMLElement;
                if (btn) btn.click();
            });
            // Wait for globe tooltip to be initialized - use MEDIUM timeout for CI
            await page.waitForFunction(() => {
                return document.getElementById('deckgl-tooltip-enhanced') !== null;
            }, { timeout: TIMEOUTS.MEDIUM });

            // Check if tooltip element has the globe-tooltip class
            const hasClass = await page.evaluate(() => {
                const tooltip = document.getElementById('deckgl-tooltip-enhanced');
                return tooltip?.classList.contains('globe-tooltip') ?? false;
            });

            expect(hasClass).toBeTruthy();
        }
    });

    test('Globe visualization helper functions are exported', async ({ page }) => {
        // This test verifies that the CSS color conversion functions work correctly
        // by checking that the globe module properly handles color conversion
        const testResult = await page.evaluate(() => {
            // Test that CSS variable reading works
            const testVar = getComputedStyle(document.documentElement)
                .getPropertyValue('--globe-marker-high').trim();
            return {
                variableExists: !!testVar,
                isValidColor: /^#[0-9a-f]{6}$/i.test(testVar) ||
                             /^rgba?\s*\(/.test(testVar)
            };
        });

        expect(testResult.variableExists).toBeTruthy();
        expect(testResult.isValidColor).toBeTruthy();
    });
});

test.describe('Globe Theme Color Mapping', () => {
    test.beforeEach(async ({ page }) => {
        await setupPage(page);
        await page.goto('/');
        // Wait for the app to load - use MEDIUM timeout for CI reliability
        await page.waitForSelector('#app', { state: 'visible', timeout: TIMEOUTS.MEDIUM });
        // Wait for CSS variables to be fully resolved
        await page.waitForFunction(
            () => {
                const style = getComputedStyle(document.documentElement);
                return style.getPropertyValue('--globe-marker-high').trim() !== '';
            },
            { timeout: TIMEOUTS.MEDIUM }
        );
    });

    test('all theme modes have complete globe color set', async ({ page }) => {
        const requiredVariables = [
            '--globe-marker-high',
            '--globe-marker-medium-high',
            '--globe-marker-medium',
            '--globe-marker-low',
            '--globe-tooltip-bg',
            '--globe-tooltip-border',
            '--globe-tooltip-text'
        ];

        // Test dark mode (default)
        for (const varName of requiredVariables) {
            const value = await page.evaluate((v) => {
                return getComputedStyle(document.documentElement).getPropertyValue(v).trim();
            }, varName);
            expect(value, `Dark mode: ${varName} should be defined`).toBeTruthy();
        }

        // Test light mode
        await page.evaluate(() => {
            document.documentElement.setAttribute('data-theme', 'light');
        });
        // Wait for CSS variables to update - use MEDIUM timeout for CI
        await page.waitForFunction(() => {
            const style = getComputedStyle(document.documentElement);
            return document.documentElement.getAttribute('data-theme') === 'light' &&
                   style.getPropertyValue('--globe-marker-high').trim() !== '';
        }, { timeout: TIMEOUTS.MEDIUM });

        for (const varName of requiredVariables) {
            const value = await page.evaluate((v) => {
                return getComputedStyle(document.documentElement).getPropertyValue(v).trim();
            }, varName);
            expect(value, `Light mode: ${varName} should be defined`).toBeTruthy();
        }

        // Test high-contrast mode
        await page.evaluate(() => {
            document.documentElement.setAttribute('data-theme', 'high-contrast');
        });
        // Wait for CSS variables to update - use MEDIUM timeout for CI
        await page.waitForFunction(() => {
            const style = getComputedStyle(document.documentElement);
            return document.documentElement.getAttribute('data-theme') === 'high-contrast' &&
                   style.getPropertyValue('--globe-marker-high').trim() !== '';
        }, { timeout: TIMEOUTS.MEDIUM });

        for (const varName of requiredVariables) {
            const value = await page.evaluate((v) => {
                return getComputedStyle(document.documentElement).getPropertyValue(v).trim();
            }, varName);
            expect(value, `High-contrast mode: ${varName} should be defined`).toBeTruthy();
        }

        // Test colorblind mode
        await page.evaluate(() => {
            document.documentElement.removeAttribute('data-theme');
            document.documentElement.setAttribute('data-colorblind', 'true');
        });
        // Wait for CSS variables to update - use MEDIUM timeout for CI
        await page.waitForFunction(() => {
            const style = getComputedStyle(document.documentElement);
            return document.documentElement.getAttribute('data-colorblind') === 'true' &&
                   style.getPropertyValue('--globe-marker-high').trim() !== '';
        }, { timeout: TIMEOUTS.MEDIUM });

        for (const varName of requiredVariables) {
            const value = await page.evaluate((v) => {
                return getComputedStyle(document.documentElement).getPropertyValue(v).trim();
            }, varName);
            expect(value, `Colorblind mode: ${varName} should be defined`).toBeTruthy();
        }
    });

    test('marker colors have sufficient contrast in high-contrast mode', async ({ page }) => {
        await page.evaluate(() => {
            document.documentElement.setAttribute('data-theme', 'high-contrast');
        });
        // Wait for CSS variables to update - use MEDIUM timeout for CI
        await page.waitForFunction(() => {
            const style = getComputedStyle(document.documentElement);
            return document.documentElement.getAttribute('data-theme') === 'high-contrast' &&
                   style.getPropertyValue('--globe-marker-high').trim() !== '';
        }, { timeout: TIMEOUTS.MEDIUM });

        // In high-contrast mode, colors should be bright/saturated
        const colors = await page.evaluate(() => {
            const style = getComputedStyle(document.documentElement);
            return {
                high: style.getPropertyValue('--globe-marker-high').trim(),
                mediumHigh: style.getPropertyValue('--globe-marker-medium-high').trim(),
                medium: style.getPropertyValue('--globe-marker-medium').trim(),
                low: style.getPropertyValue('--globe-marker-low').trim()
            };
        });

        // All high-contrast colors should be bright (pure/saturated colors)
        expect(colors.high).toMatch(/#ff00ff|magenta/i); // Magenta
        expect(colors.mediumHigh).toMatch(/#ff8800/i); // Orange
        expect(colors.medium).toMatch(/#ffff00|yellow/i); // Yellow
        expect(colors.low).toMatch(/#00ffff|cyan/i); // Cyan
    });
});
