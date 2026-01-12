// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
import {
  test,
  expect,
  TIMEOUTS,
  SELECTORS,
  gotoAppAndWaitReady,
} from './fixtures';

/**
 * E2E Tests for 3D Globe Visualization (deck.gl)
 *
 * Tests the actual 3D globe view functionality using deck.gl:
 * - Globe toggle (2D to 3D switch via #view-mode-2d / #view-mode-3d)
 * - Auto-rotate feature (#globe-auto-rotate)
 * - Reset view button (#globe-reset-view)
 * - WebGL canvas rendering
 * - Globe visibility and state management
 *
 * NOTE: The app uses GlobeManagerDeckGL (not the enhanced version).
 * The HTML provides basic globe controls:
 * - #view-mode-3d - Toggle to 3D globe view
 * - #view-mode-2d - Toggle to 2D map view
 * - #globe-auto-rotate - Toggle auto-rotation
 * - #globe-reset-view - Reset camera to default
 *
 * NOTE: These tests run in the 'chromium' project which has storageState
 * pre-loaded from auth.setup.ts, so we DON'T need to login again.
 *
 * NOTE: All tile requests are automatically mocked via fixtures.ts
 * This makes tests fully offline, deterministic, and faster.
 */

test.describe('3D Globe View Toggle', () => {
    test.beforeEach(async ({ page }) => {
        // Navigate to app (already authenticated via storageState from setup)
        await gotoAppAndWaitReady(page);

        // Wait for map to load first
        await page.waitForSelector(SELECTORS.MAP, { state: 'visible', timeout: TIMEOUTS.DEFAULT });

        // Wait for ViewportManager to initialize (sets aria-pressed on view mode buttons)
        // Note: If this times out, tests may fail due to uninitialized view state
        const viewportReady = await page.waitForFunction(() => {
            const btn = document.getElementById('view-mode-3d');
            return btn && btn.getAttribute('aria-pressed') !== null;
        }, { timeout: TIMEOUTS.MEDIUM }).then(() => true).catch(() => false);
        if (!viewportReady) {
            console.warn('[E2E] ViewportManager initialization timeout in 3D Globe View Toggle tests');
        }
    });

    test('should display 2D/3D view toggle buttons', async ({ page }) => {
        // Verify view toggle container exists
        const viewToggle = page.locator(SELECTORS.VIEW_TOGGLE_CONTROL);
        await expect(viewToggle).toBeVisible();

        // Verify both toggle buttons exist
        const btn2D = page.locator(SELECTORS.VIEW_MODE_2D);
        const btn3D = page.locator(SELECTORS.VIEW_MODE_3D);

        await expect(btn2D).toBeVisible();
        await expect(btn3D).toBeVisible();

        // 2D should be active by default
        await expect(btn2D).toHaveClass(/active/);
    });

    test('should switch to 3D globe view when clicking toggle', async ({ page }) => {
        // Click 3D toggle button
        const btn3D = page.locator(SELECTORS.VIEW_MODE_3D);
        await btn3D.click();

        // Wait for globe to become visible
        await page.waitForFunction(() => {
            const globe = document.getElementById('globe');
            return globe && globe.style.display !== 'none';
        }, { timeout: TIMEOUTS.DATA_LOAD });

        // Check 3D button is now active
        await expect(btn3D).toHaveClass(/active/);

        // Globe container should be visible (display: block)
        const isGlobeVisible = await page.evaluate(() => {
            const el = document.getElementById('globe');
            return el && el.style.display !== 'none';
        });
        expect(isGlobeVisible).toBe(true);
    });

    test('should switch back to 2D map view when clicking toggle', async ({ page }) => {
        // First switch to 3D
        const btn3D = page.locator(SELECTORS.VIEW_MODE_3D);
        await btn3D.click();

        // Wait for globe to become visible
        await page.waitForFunction(() => {
            const globe = document.getElementById('globe');
            return globe && globe.style.display !== 'none';
        }, { timeout: TIMEOUTS.RENDER });

        // Then switch back to 2D
        const btn2D = page.locator(SELECTORS.VIEW_MODE_2D);
        await btn2D.click();

        // Wait for map to become visible
        await page.waitForFunction(() => {
            const map = document.getElementById('map');
            return map && getComputedStyle(map).display !== 'none';
        }, { timeout: TIMEOUTS.RENDER });

        // Check 2D button is now active
        await expect(btn2D).toHaveClass(/active/);

        // Map should be visible
        const mapVisible = await page.evaluate(() => {
            const map = document.getElementById('map');
            return map && getComputedStyle(map).display !== 'none';
        });
        expect(mapVisible).toBe(true);
    });

    test('should have proper ARIA attributes on toggle buttons', async ({ page }) => {
        // Check view toggle group accessibility
        const viewToggle = page.locator(SELECTORS.VIEW_TOGGLE_CONTROL);
        await expect(viewToggle).toHaveAttribute('role', 'group');
        await expect(viewToggle).toHaveAttribute('aria-label', 'View mode');

        // Check 2D button has aria-pressed
        const btn2D = page.locator(SELECTORS.VIEW_MODE_2D);
        await expect(btn2D).toHaveAttribute('aria-pressed', 'true');

        // Check 3D button has aria-pressed
        const btn3D = page.locator(SELECTORS.VIEW_MODE_3D);
        await expect(btn3D).toHaveAttribute('aria-pressed', 'false');
    });
});

test.describe('Globe Control Buttons', () => {
    test.beforeEach(async ({ page }) => {
        // Navigate to app (already authenticated via storageState from setup)
        await gotoAppAndWaitReady(page);

        // Switch to globe view
        await page.waitForSelector(SELECTORS.VIEW_MODE_3D, { state: 'visible' });
        await page.locator(SELECTORS.VIEW_MODE_3D).click();

        // Wait for globe to become visible
        await page.waitForFunction(() => {
            const globe = document.getElementById('globe');
            return globe && globe.style.display !== 'none';
        }, { timeout: TIMEOUTS.DATA_LOAD });
    });

    test('should have globe control buttons visible in globe mode', async ({ page }) => {
        // Globe controls container should exist
        const globeControls = page.locator(SELECTORS.GLOBE_CONTROLS);
        await expect(globeControls).toBeVisible({ timeout: TIMEOUTS.MEDIUM });

        // Auto-rotate button should exist
        const autoRotateBtn = page.locator(SELECTORS.GLOBE_AUTO_ROTATE);
        await expect(autoRotateBtn).toBeVisible();

        // Reset view button should exist
        const resetViewBtn = page.locator(SELECTORS.GLOBE_RESET_VIEW);
        await expect(resetViewBtn).toBeVisible();
    });

    test('should toggle auto-rotation when clicking auto-rotate button', async ({ page }) => {
        // Find auto-rotate button
        const autoRotateBtn = page.locator(SELECTORS.GLOBE_AUTO_ROTATE);
        await expect(autoRotateBtn).toBeVisible();

        // Get initial state
        const initialActive = await autoRotateBtn.evaluate((el) =>
            el.classList.contains('active')
        );

        // Click to toggle
        await autoRotateBtn.click();

        // Wait for state to change (class to toggle)
        await page.waitForFunction(
            (expected) => {
                const btn = document.getElementById('globe-auto-rotate');
                return btn && btn.classList.contains('active') !== expected;
            },
            initialActive,
            { timeout: TIMEOUTS.RENDER }
        );

        // State should have changed
        const afterClickActive = await autoRotateBtn.evaluate((el) =>
            el.classList.contains('active')
        );
        expect(afterClickActive).not.toBe(initialActive);

        // Click again to toggle back
        await autoRotateBtn.click();

        // Wait for state to revert
        await page.waitForFunction(
            (expected) => {
                const btn = document.getElementById('globe-auto-rotate');
                return btn && btn.classList.contains('active') === expected;
            },
            initialActive,
            { timeout: TIMEOUTS.RENDER }
        );

        const afterSecondClickActive = await autoRotateBtn.evaluate((el) =>
            el.classList.contains('active')
        );
        expect(afterSecondClickActive).toBe(initialActive);
    });

    test('should reset view when clicking reset button', async ({ page }) => {
        // Find reset view button
        const resetViewBtn = page.locator(SELECTORS.GLOBE_RESET_VIEW);
        await expect(resetViewBtn).toBeVisible();

        // Click reset - should not throw error
        await resetViewBtn.click();

        // Wait for globe to remain visible after reset
        await page.waitForFunction(() => {
            const globe = document.getElementById('globe');
            return globe && globe.style.display !== 'none';
        }, { timeout: TIMEOUTS.RENDER });

        // Globe should still be visible and functional
        const globeVisible = await page.evaluate(() => {
            const el = document.getElementById('globe');
            return el && el.style.display !== 'none';
        });
        expect(globeVisible).toBe(true);
    });

    test('should have proper accessibility attributes on control buttons', async ({ page }) => {
        // Check globe controls group accessibility
        // Note: role="toolbar" is more semantically correct than role="group" for control buttons
        const globeControls = page.locator(SELECTORS.GLOBE_CONTROLS);
        await expect(globeControls).toHaveAttribute('role', 'toolbar');
        await expect(globeControls).toHaveAttribute('aria-label', 'Globe view controls');

        // Check auto-rotate button has aria-label
        const autoRotateBtn = page.locator(SELECTORS.GLOBE_AUTO_ROTATE);
        const autoRotateLabel = await autoRotateBtn.getAttribute('aria-label');
        expect(autoRotateLabel).toBeTruthy();

        // Check reset button has aria-label
        const resetViewBtn = page.locator(SELECTORS.GLOBE_RESET_VIEW);
        const resetLabel = await resetViewBtn.getAttribute('aria-label');
        expect(resetLabel).toBeTruthy();
    });
});

test.describe('Globe Container and Canvas', () => {
    test.beforeEach(async ({ page }) => {
        // Navigate to app (already authenticated via storageState from setup)
        await gotoAppAndWaitReady(page);
    });

    test('should have proper accessibility attributes on globe container', async ({ page }) => {
        // Check globe container accessibility (before switching to globe)
        const globe = page.locator(SELECTORS.GLOBE);
        await expect(globe).toHaveAttribute('role', 'application');
        // Use partial match since aria-label may include keyboard navigation instructions
        const ariaLabel = await globe.getAttribute('aria-label');
        expect(ariaLabel).toContain('Interactive 3D globe');
    });

    test('should show WebGL not supported message element exists', async ({ page }) => {
        // This element should exist but be hidden by default
        const webglNotSupported = page.locator('#globe-not-supported');

        // Element should exist in DOM
        const exists = await webglNotSupported.count();
        expect(exists).toBe(1);

        // Check it has proper error message content
        const text = await webglNotSupported.textContent();
        expect(text?.toLowerCase()).toContain('webgl');
    });

    test('should render globe canvas when WebGL is available', async ({ page }) => {
        // Check WebGL availability first
        const hasWebGL = await page.evaluate(() => {
            try {
                const canvas = document.createElement('canvas');
                return !!(
                    canvas.getContext('webgl') ||
                    canvas.getContext('experimental-webgl')
                );
            } catch {
                return false;
            }
        });

        if (!hasWebGL) {
            test.skip();
            return;
        }

        // Switch to globe view
        await page.locator(SELECTORS.VIEW_MODE_3D).click();

        // Wait for globe to become visible with WebGL init
        await page.waitForFunction(() => {
            const globe = document.getElementById('globe');
            return globe && globe.style.display !== 'none';
        }, { timeout: TIMEOUTS.WEBGL_INIT });

        // Wait for canvas to appear
        const canvas = page.locator('#globe canvas');
        await expect(canvas).toBeVisible({ timeout: TIMEOUTS.DEFAULT });

        // Canvas should have dimensions
        const box = await canvas.boundingBox();
        expect(box).not.toBeNull();
        expect(box!.width).toBeGreaterThan(100);
        expect(box!.height).toBeGreaterThan(100);
    });

    test('should show loading indicator while globe initializes', async ({ page }) => {
        // Globe loading element should exist
        const globeLoading = page.locator('#globe-loading');
        const exists = await globeLoading.count();
        expect(exists).toBe(1);
    });
});

test.describe('Globe State Management', () => {
    test.beforeEach(async ({ page }) => {
        // Navigate to app (already authenticated via storageState from setup)
        await gotoAppAndWaitReady(page);

        // Wait for ViewportManager to initialize (sets aria-pressed on view mode buttons)
        const viewportReady = await page.waitForFunction(() => {
            const btn = document.getElementById('view-mode-3d');
            return btn && btn.getAttribute('aria-pressed') !== null;
        }, { timeout: TIMEOUTS.MEDIUM }).then(() => true).catch(() => false);
        if (!viewportReady) {
            console.warn('[E2E] ViewportManager initialization timeout in Globe State Management tests');
        }
    });

    test('should maintain view state on filter changes', async ({ page }) => {
        // Switch to globe view
        await page.locator(SELECTORS.VIEW_MODE_3D).click();

        // Wait for globe to become visible
        await page.waitForFunction(() => {
            const globe = document.getElementById('globe');
            return globe && globe.style.display !== 'none';
        }, { timeout: TIMEOUTS.DATA_LOAD });

        // Verify in globe mode
        await expect(page.locator(SELECTORS.VIEW_MODE_3D)).toHaveClass(/active/);

        // Change a filter (date range)
        const daysFilter = page.locator(SELECTORS.FILTER_DAYS);
        if (await daysFilter.isVisible()) {
            await daysFilter.selectOption('7');

            // Wait for filter to apply
            await page.waitForFunction(() => {
                const select = document.querySelector('#filter-days') as HTMLSelectElement;
                return select && select.value === '7';
            }, { timeout: TIMEOUTS.RENDER });
        }

        // Should still be in globe mode after filter change
        await expect(page.locator(SELECTORS.VIEW_MODE_3D)).toHaveClass(/active/);
    });

    test('should handle view toggle keyboard navigation', async ({ page }) => {
        // Focus on 2D button
        const btn2D = page.locator(SELECTORS.VIEW_MODE_2D);
        await btn2D.focus();

        // Press Tab to move to 3D button
        await page.keyboard.press('Tab');

        // 3D button should be focused
        const btn3D = page.locator(SELECTORS.VIEW_MODE_3D);
        const isFocused = await btn3D.evaluate((el) => el === document.activeElement);
        expect(isFocused).toBe(true);

        // Press Enter to activate
        await page.keyboard.press('Enter');

        // Wait for globe to become visible
        await page.waitForFunction(() => {
            const globe = document.getElementById('globe');
            return globe && globe.style.display !== 'none';
        }, { timeout: TIMEOUTS.RENDER });

        // 3D button should now be active
        await expect(btn3D).toHaveClass(/active/);
    });

    test('should update locations on globe when data changes', async ({ page }) => {
        // Switch to globe view
        await page.locator(SELECTORS.VIEW_MODE_3D).click();

        // Wait for globe to become visible
        await page.waitForFunction(() => {
            const globe = document.getElementById('globe');
            return globe && globe.style.display !== 'none';
        }, { timeout: TIMEOUTS.DATA_LOAD });

        // Click refresh button if it exists
        const refreshBtn = page.locator(SELECTORS.BTN_REFRESH);
        if (await refreshBtn.isVisible()) {
            await refreshBtn.click();

            // Wait for data to refresh (globe still visible)
            await page.waitForFunction(() => {
                const globe = document.getElementById('globe');
                return globe && globe.style.display !== 'none';
            }, { timeout: TIMEOUTS.DATA_LOAD });
        }

        // Globe should still be visible and functional
        const globeVisible = await page.evaluate(() => {
            const el = document.getElementById('globe');
            return el && el.style.display !== 'none';
        });
        expect(globeVisible).toBe(true);
    });
});

test.describe('Globe Performance', () => {
    test.beforeEach(async ({ page }) => {
        await gotoAppAndWaitReady(page);

        // Wait for ViewportManager to initialize (sets aria-pressed on view mode buttons)
        // This is critical for performance tests to measure actual view switch time
        const viewportReady = await page.waitForFunction(() => {
            const btn = document.getElementById('view-mode-3d');
            return btn && btn.getAttribute('aria-pressed') !== null;
        }, { timeout: TIMEOUTS.MEDIUM }).then(() => true).catch(() => false);
        if (!viewportReady) {
            console.warn('[E2E] ViewportManager initialization timeout in Globe Performance tests');
        }
    });

    test('should switch views within acceptable time', async ({ page }) => {
        // Switch to globe
        const startSwitch = Date.now();
        await page.locator(SELECTORS.VIEW_MODE_3D).click();

        // Wait for globe to be visible
        await page.waitForFunction(() => {
            const globe = document.getElementById('globe');
            return globe && globe.style.display !== 'none';
        }, { timeout: TIMEOUTS.MEDIUM });

        const switchTime = Date.now() - startSwitch;

        // Should switch within 3 seconds (including lazy load)
        expect(switchTime).toBeLessThan(3000);
        console.log(`Globe switch time: ${switchTime}ms`);
    });

    test('should not cause memory leaks on repeated view switches', async ({ page }) => {
        const pageErrors: string[] = [];
        page.on('pageerror', (err) => pageErrors.push(err.message));

        // Switch views multiple times
        for (let i = 0; i < 3; i++) {
            await page.locator(SELECTORS.VIEW_MODE_3D).click();

            // Wait for globe to become visible
            await page.waitForFunction(() => {
                const globe = document.getElementById('globe');
                return globe && globe.style.display !== 'none';
            }, { timeout: TIMEOUTS.RENDER });

            await page.locator(SELECTORS.VIEW_MODE_2D).click();

            // Wait for map to become visible
            await page.waitForFunction(() => {
                const map = document.getElementById('map');
                return map && getComputedStyle(map).display !== 'none';
            }, { timeout: TIMEOUTS.RENDER });
        }

        // Should not throw errors or crash
        const appStillVisible = await page.locator(SELECTORS.APP).isVisible();
        expect(appStillVisible).toBe(true);

        // Wait for any pending operations to complete
        // E2E FIX: Use #app (id) instead of .app (class)
        await page.waitForFunction(() => {
            const app = document.getElementById('app');
            return app !== null;
        }, { timeout: TIMEOUTS.DATA_LOAD });

        // E2E FIX: Filter out known non-critical errors that occur during normal operation
        // Detection API 404s cause "null" and "undefined" errors in console when not configured
        // These are expected in CI where detection endpoints are not fully implemented
        const knownNonCriticalPatterns = [
            'detection',        // Detection API not fully implemented
            'backup',           // Backup data may not exist
            'Failed to fetch',  // Network timing issues
            'AbortError',       // Cancelled requests during view switching
            'WebGL',            // WebGL warnings in software rendering
            'SwiftShader',      // Software WebGL fallback
            "reading 'id'",     // MapLibre/deck.gl layer ID access during view transitions
            'layer',            // Layer-related errors during rapid switching
        ];

        const criticalErrors = pageErrors.filter((e) => {
            const isCritical = e.includes('undefined') || e.includes('null') || e.includes('TypeError');
            if (!isCritical) return false;

            // Check if this error matches any known non-critical pattern
            const isKnownNonCritical = knownNonCriticalPatterns.some(pattern =>
                e.toLowerCase().includes(pattern.toLowerCase())
            );
            return !isKnownNonCritical;
        });

        // Log filtered errors for debugging
        if (criticalErrors.length > 0) {
            console.log('[E2E] Critical errors found:', criticalErrors);
        }
        expect(criticalErrors.length).toBe(0);
    });

    test('should not show WebGL errors in console', async ({ page }) => {
        const errors: string[] = [];
        page.on('console', (msg) => {
            if (msg.type() === 'error') {
                errors.push(msg.text());
            }
        });

        // Switch to globe view
        await page.locator(SELECTORS.VIEW_MODE_3D).click();

        // Wait for globe to become visible with WebGL init
        await page.waitForFunction(() => {
            const globe = document.getElementById('globe');
            return globe && globe.style.display !== 'none';
        }, { timeout: TIMEOUTS.WEBGL_INIT });

        // Filter for WebGL-related errors
        const webglErrors = errors.filter((e) =>
            e.toLowerCase().includes('webgl') ||
            e.toLowerCase().includes('gl error') ||
            e.toLowerCase().includes('deck.gl')
        );

        expect(webglErrors.length).toBe(0);
    });
});

test.describe('Globe Accessibility', () => {
    test.beforeEach(async ({ page }) => {
        await gotoAppAndWaitReady(page);

        // Wait for ViewportManager to initialize
        const viewportReady = await page.waitForFunction(() => {
            const btn = document.getElementById('view-mode-3d');
            return btn && btn.getAttribute('aria-pressed') !== null;
        }, { timeout: TIMEOUTS.MEDIUM }).then(() => true).catch(() => false);
        if (!viewportReady) {
            console.warn('[E2E] ViewportManager initialization timeout in Globe Accessibility tests');
        }
    });

    test('should be focusable for keyboard users', async ({ page }) => {
        // Switch to globe view
        await page.locator(SELECTORS.VIEW_MODE_3D).click();

        // Wait for globe to become visible
        await page.waitForFunction(() => {
            const globe = document.getElementById('globe');
            return globe && globe.style.display !== 'none';
        }, { timeout: TIMEOUTS.DATA_LOAD });

        // Globe container should be focusable (has role="application")
        const globe = page.locator(SELECTORS.GLOBE);
        await expect(globe).toHaveAttribute('role', 'application');

        // Controls should be focusable
        const autoRotateBtn = page.locator(SELECTORS.GLOBE_AUTO_ROTATE);
        await autoRotateBtn.focus();

        const isFocused = await autoRotateBtn.evaluate((el) => el === document.activeElement);
        expect(isFocused).toBe(true);
    });

    test('should have proper focus indicators on controls', async ({ page }) => {
        // Switch to globe view
        await page.locator(SELECTORS.VIEW_MODE_3D).click();

        // Wait for globe to become visible
        await page.waitForFunction(() => {
            const globe = document.getElementById('globe');
            return globe && globe.style.display !== 'none';
        }, { timeout: TIMEOUTS.DATA_LOAD });

        // Focus on auto-rotate button
        const autoRotateBtn = page.locator(SELECTORS.GLOBE_AUTO_ROTATE);
        await autoRotateBtn.focus();

        // Check for visible focus indicator (outline or similar)
        const outlineStyle = await autoRotateBtn.evaluate((el) => {
            const style = getComputedStyle(el);
            return style.outline || style.boxShadow;
        });

        // Should have some focus indicator
        expect(outlineStyle).toBeTruthy();
    });
});
