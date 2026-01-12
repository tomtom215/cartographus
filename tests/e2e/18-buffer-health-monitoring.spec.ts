// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
import {
  test,
  expect,
  Page,
  TIMEOUTS,
  gotoAppAndWaitReady,
} from './fixtures';

/**
 * E2E Tests for Buffer Health Monitoring (Phase 2.1: v1.41)
 *
 * Tests real-time buffer health monitoring with:
 * - WebSocket message handling
 * - UI panel updates with critical/risky sessions
 * - Progress bars and health indicators
 * - Predictive buffering warnings
 * - Toast notifications for critical/risky buffers
 * - Color-coded health status (🔴 critical, 🟡 risky, 🟢 healthy)
 *
 * NOTE: These tests simulate WebSocket messages without backend dependency
 */

test.describe('Buffer Health Monitoring', () => {
    test.beforeEach(async ({ page }) => {
        // Use storageState for authentication (configured in playwright.config.ts)
        await gotoAppAndWaitReady(page);

        // Wait for PlexMonitoringManager to be initialized (not null)
        await page.waitForFunction(() => {
            const app = (window as any).__app;
            return app && app.__testGetPlexMonitoringManager && app.__testGetPlexMonitoringManager() !== null;
        }, { timeout: TIMEOUTS.DEFAULT });

        // Wait for buffer health panel elements to be ready
        await page.waitForFunction(() => {
            const panel = document.getElementById('buffer-health-panel');
            const criticalBadge = document.getElementById('buffer-critical-badge');
            const riskyBadge = document.getElementById('buffer-risky-badge');
            return panel !== null && criticalBadge !== null && riskyBadge !== null;
        }, { timeout: TIMEOUTS.RENDER });
    });

    /**
     * Helper to verify buffer health panel DOM elements exist
     * Returns false if elements are missing (test should be skipped)
     */
    async function verifyBufferPanelExists(page: Page): Promise<boolean> {
        const panelExists = await page.evaluate(() => {
            const panel = document.getElementById('buffer-health-panel');
            const criticalBadge = document.getElementById('buffer-critical-badge');
            const riskyBadge = document.getElementById('buffer-risky-badge');
            return panel !== null && criticalBadge !== null && riskyBadge !== null;
        });
        return panelExists;
    }

    /**
     * Helper function to simulate buffer health WebSocket message
     * Uses the __app test helper to inject messages
     */
    async function simulateBufferHealthMessage(
        page: Page,
        sessions: any[],
        critical_count: number,
        risky_count: number
    ) {
        await page.evaluate(({ sessions, critical_count, risky_count }) => {
            const data = {
                sessions,
                timestamp: Math.floor(Date.now() / 1000),
                critical_count,
                risky_count
            };

            // Use the exposed app instance's test helper
            const app = (window as any).__app;
            if (app && app.__testSimulateWebSocketMessage) {
                app.__testSimulateWebSocketMessage('buffer_health_update', data);
            }
        }, { sessions, critical_count, risky_count });
    }

    test('01. Should register WebSocket callback for buffer health updates', async ({ page }) => {
        // Verify app has test helper and PlexMonitoringManager is initialized
        const hasHandler = await page.evaluate(() => {
            const app = (window as any).__app;
            return app && typeof app.__testSimulateWebSocketMessage === 'function' &&
                   app.__testGetPlexMonitoringManager() !== null;
        });

        expect(hasHandler).toBeTruthy();
    });

    test('02. Should hide buffer health panel when all sessions healthy', async ({ page }) => {
        // Skip if panel elements don't exist
        if (!await verifyBufferPanelExists(page)) { test.skip(); return; }

        // Simulate message with zero critical/risky sessions
        await simulateBufferHealthMessage(page, [], 0, 0);

        // Wait for panel to be hidden or remain hidden
        await page.waitForFunction(() => {
            const panel = document.getElementById('buffer-health-panel');
            return panel?.classList.contains('hidden') !== false;
        }, { timeout: TIMEOUTS.RENDER });

        // Panel should be hidden (or remain hidden since it starts hidden)
        const isHidden = await page.evaluate(() => {
            const panel = document.getElementById('buffer-health-panel');
            return panel ? panel.classList.contains('hidden') : true;
        });

        expect(isHidden).toBeTruthy();
    });

    test('03. Should show buffer health panel when critical sessions detected', async ({ page }) => {
        if (!await verifyBufferPanelExists(page)) { test.skip(); return; }
        // Simulate message with 1 critical session
        const sessions = [{
            sessionKey: 'critical-session-1',
            title: 'Test Movie (2024)',
            username: 'John Doe',
            player_device: 'Plex Web',
            bufferFillPercent: 15.0,      // Critical: below 20%
            bufferDrainRate: 1.5,          // Draining 50% faster
            bufferSeconds: 5.0,            // Only 5 seconds buffered
            healthStatus: 'critical',
            riskLevel: 2,
            maxOffsetAvailable: 125000,    // 125 seconds total
            viewOffset: 120000,            // 120 seconds played (5s ahead)
            transcodeSpeed: 1.0,
            timestamp: new Date().toISOString(),
            alertSent: false
        }];

        await simulateBufferHealthMessage(page, sessions, 1, 0);

        // Wait for panel to become visible and session card to render
        await page.waitForFunction(() => {
            const panel = document.getElementById('buffer-health-panel');
            const sessionCard = document.querySelector('.buffer-health-session-card');
            return panel && !panel.classList.contains('hidden') && sessionCard !== null;
        }, { timeout: TIMEOUTS.RENDER });

        // Panel should be visible
        const isVisible = await page.evaluate(() => {
            const panel = document.getElementById('buffer-health-panel');
            return panel && !panel.classList.contains('hidden');
        });

        expect(isVisible).toBeTruthy();

        // Verify session card is rendered
        const sessionCards = await page.locator('.buffer-health-session-card').count();
        expect(sessionCards).toBe(1);
    });

    test('04. Should update critical and risky badge counts correctly', async ({ page }) => {
        if (!await verifyBufferPanelExists(page)) { test.skip(); return; }
        // Simulate 2 critical and 3 risky sessions
        const sessions = [
            {
                sessionKey: 'critical-1',
                title: 'Critical Movie 1',
                username: 'User1',
                player_device: 'Device1',
                bufferFillPercent: 10.0,
                bufferDrainRate: 1.8,
                bufferSeconds: 3.0,
                healthStatus: 'critical',
                riskLevel: 2,
                maxOffsetAvailable: 100000,
                viewOffset: 97000,
                transcodeSpeed: 1.0,
                timestamp: new Date().toISOString(),
                alertSent: false
            },
            {
                sessionKey: 'critical-2',
                title: 'Critical Movie 2',
                username: 'User2',
                player_device: 'Device2',
                bufferFillPercent: 18.0,
                bufferDrainRate: 1.4,
                bufferSeconds: 6.0,
                healthStatus: 'critical',
                riskLevel: 2,
                maxOffsetAvailable: 100000,
                viewOffset: 94000,
                transcodeSpeed: 1.0,
                timestamp: new Date().toISOString(),
                alertSent: false
            },
            {
                sessionKey: 'risky-1',
                title: 'Risky Movie 1',
                username: 'User3',
                player_device: 'Device3',
                bufferFillPercent: 35.0,
                bufferDrainRate: 1.2,
                bufferSeconds: 10.0,
                healthStatus: 'risky',
                riskLevel: 1,
                maxOffsetAvailable: 100000,
                viewOffset: 90000,
                transcodeSpeed: 1.0,
                timestamp: new Date().toISOString(),
                alertSent: false
            }
        ];

        await simulateBufferHealthMessage(page, sessions, 2, 3);

        // Wait for badges to update with correct counts
        await page.waitForFunction(() => {
            const criticalBadge = document.getElementById('buffer-critical-badge');
            const riskyBadge = document.getElementById('buffer-risky-badge');
            return criticalBadge?.textContent === '2' && riskyBadge?.textContent === '3';
        }, { timeout: TIMEOUTS.RENDER });

        // Verify badge counts
        const criticalBadgeText = await page.textContent('#buffer-critical-badge');
        const riskyBadgeText = await page.textContent('#buffer-risky-badge');

        expect(criticalBadgeText).toBe('2');
        expect(riskyBadgeText).toBe('3');
    });

    test('05. Should render critical session card with correct health indicator', async ({ page }) => {
        if (!await verifyBufferPanelExists(page)) { test.skip(); return; }
        const sessions = [{
            sessionKey: 'critical-test',
            title: 'Critical Test Movie',
            username: 'TestUser',
            player_device: 'Chrome',
            bufferFillPercent: 12.5,
            bufferDrainRate: 1.7,
            bufferSeconds: 4.2,
            healthStatus: 'critical',
            riskLevel: 2,
            maxOffsetAvailable: 100000,
            viewOffset: 95800,
            transcodeSpeed: 1.1,
            timestamp: new Date().toISOString(),
            alertSent: false
        }];

        await simulateBufferHealthMessage(page, sessions, 1, 0);

        // Wait for critical session card to render
        await page.waitForFunction(() => {
            const criticalCard = document.querySelector('.buffer-health-session-card.critical');
            const title = document.querySelector('.buffer-health-session-title');
            return criticalCard !== null && title?.textContent?.includes('Critical Test Movie');
        }, { timeout: TIMEOUTS.RENDER });

        // Verify critical class is applied
        const hasCriticalClass = await page.locator('.buffer-health-session-card.critical').count();
        expect(hasCriticalClass).toBe(1);

        // Verify title is displayed
        const titleText = await page.textContent('.buffer-health-session-title');
        expect(titleText).toContain('Critical Test Movie');

        // Verify user info
        const userText = await page.textContent('.buffer-health-session-user');
        expect(userText).toContain('TestUser');
        expect(userText).toContain('Chrome');

        // Verify health indicator shows CRITICAL
        const indicatorText = await page.textContent('.buffer-health-indicator');
        expect(indicatorText).toContain('CRITICAL');
    });

    test('06. Should render risky session card with correct styling', async ({ page }) => {
        if (!await verifyBufferPanelExists(page)) { test.skip(); return; }
        const sessions = [{
            sessionKey: 'risky-test',
            title: 'Risky Test Show',
            username: 'RiskyUser',
            player_device: 'Safari',
            bufferFillPercent: 42.0,
            bufferDrainRate: 1.15,
            bufferSeconds: 13.0,
            healthStatus: 'risky',
            riskLevel: 1,
            maxOffsetAvailable: 110000,
            viewOffset: 97000,
            transcodeSpeed: 1.3,
            timestamp: new Date().toISOString(),
            alertSent: false
        }];

        await simulateBufferHealthMessage(page, sessions, 0, 1);

        // Wait for risky session card to render
        await page.waitForFunction(() => {
            const riskyCard = document.querySelector('.buffer-health-session-card.risky');
            const indicator = document.querySelector('.buffer-health-indicator');
            return riskyCard !== null && indicator?.textContent?.includes('RISKY');
        }, { timeout: TIMEOUTS.RENDER });

        // Verify risky class is applied
        const hasRiskyClass = await page.locator('.buffer-health-session-card.risky').count();
        expect(hasRiskyClass).toBe(1);

        // Verify health indicator shows RISKY
        const indicatorText = await page.textContent('.buffer-health-indicator');
        expect(indicatorText).toContain('RISKY');
    });

    test('07. Should display buffer metrics correctly', async ({ page }) => {
        if (!await verifyBufferPanelExists(page)) { test.skip(); return; }
        const sessions = [{
            sessionKey: 'metrics-test',
            title: 'Metrics Test',
            username: 'MetricsUser',
            player_device: 'Firefox',
            bufferFillPercent: 25.5,
            bufferDrainRate: 1.35,
            bufferSeconds: 8.7,
            healthStatus: 'risky',
            riskLevel: 1,
            maxOffsetAvailable: 100000,
            viewOffset: 91300,
            transcodeSpeed: 1.24,  // Use 1.24 so toFixed(1) rounds to "1.2"
            timestamp: new Date().toISOString(),
            alertSent: false
        }];

        await simulateBufferHealthMessage(page, sessions, 0, 1);

        // Wait for session card with metrics to render
        await page.waitForFunction(() => {
            const detailValues = document.querySelectorAll('.buffer-health-detail-value');
            return detailValues.length === 4;
        }, { timeout: TIMEOUTS.RENDER });

        // Get all detail values
        const detailValues = await page.locator('.buffer-health-detail-value').allTextContents();

        // Should have 4 metrics: Buffer Fill, Drain Rate, Buffer Time, Speed
        expect(detailValues).toHaveLength(4);

        // Verify values contain expected patterns (allow for formatting variations)
        const valuesText = detailValues.join(' ');
        expect(valuesText).toContain('25');  // Buffer fill percent
        expect(valuesText).toContain('1.35'); // Drain rate
        expect(valuesText).toContain('8.7');  // Buffer seconds
        expect(valuesText).toContain('1.2');  // Transcode speed
    });

    test('08. Should render progress bar with correct width', async ({ page }) => {
        if (!await verifyBufferPanelExists(page)) { test.skip(); return; }
        const sessions = [{
            sessionKey: 'progress-test',
            title: 'Progress Test',
            username: 'User',
            player_device: 'Device',
            bufferFillPercent: 67.3,
            bufferDrainRate: 1.1,
            bufferSeconds: 20.0,
            healthStatus: 'healthy',
            riskLevel: 0,
            maxOffsetAvailable: 150000,
            viewOffset: 130000,
            transcodeSpeed: 1.0,
            timestamp: new Date().toISOString(),
            alertSent: false
        }];

        // Note: healthy sessions are filtered out in UI, so let's use risky
        sessions[0].healthStatus = 'risky';
        sessions[0].riskLevel = 1;
        sessions[0].bufferFillPercent = 45.0;

        await simulateBufferHealthMessage(page, sessions, 0, 1);

        // Wait for progress bar to render with correct width
        await page.waitForFunction(() => {
            const progressFill = document.querySelector('.buffer-health-progress-fill');
            if (!progressFill) return false;
            const width = (progressFill as HTMLElement).style.width;
            return width && width.includes('45');
        }, { timeout: TIMEOUTS.RENDER });

        // Verify progress bar exists and has correct width style
        const progressFill = page.locator('.buffer-health-progress-fill').first();
        const width = await progressFill.evaluate((el) => el.style.width);

        // Should be approximately 45%
        expect(width).toContain('45');
    });

    test('09. Should display predictive buffering warning', async ({ page }) => {
        if (!await verifyBufferPanelExists(page)) { test.skip(); return; }
        // Session with high drain rate = imminent buffering
        const sessions = [{
            sessionKey: 'prediction-test',
            title: 'Prediction Test',
            username: 'User',
            player_device: 'Device',
            bufferFillPercent: 16.0,
            bufferDrainRate: 1.5,         // Fast drain
            bufferSeconds: 10.0,          // 10s buffer / 0.5 drain = 20s until buffering
            healthStatus: 'critical',
            riskLevel: 2,
            maxOffsetAvailable: 100000,
            viewOffset: 90000,
            transcodeSpeed: 1.0,
            timestamp: new Date().toISOString(),
            alertSent: false
        }];

        await simulateBufferHealthMessage(page, sessions, 1, 0);

        // Wait for prediction warning to appear
        await page.waitForFunction(() => {
            const prediction = document.querySelector('.buffer-health-prediction');
            return prediction?.textContent?.includes('Buffering in');
        }, { timeout: TIMEOUTS.RENDER });

        // Should show prediction warning
        const predictionText = await page.textContent('.buffer-health-prediction');
        expect(predictionText).toContain('Buffering in');
        expect(predictionText).toContain('s'); // Should have seconds
    });

    test('10. Should show critical toast notification', async ({ page }) => {
        if (!await verifyBufferPanelExists(page)) { test.skip(); return; }
        // Listen for toast notifications
        let toastShown = false;
        page.on('console', (msg) => {
            if (msg.text().includes('[Buffer Health]') && msg.text().includes('CRITICAL')) {
                toastShown = true;
            }
        });

        const sessions = [{
            sessionKey: 'toast-critical-test',
            title: 'Toast Critical Movie',
            username: 'User',
            player_device: 'Device',
            bufferFillPercent: 10.0,
            bufferDrainRate: 2.0,
            bufferSeconds: 3.0,
            healthStatus: 'critical',
            riskLevel: 2,
            maxOffsetAvailable: 100000,
            viewOffset: 97000,
            transcodeSpeed: 1.0,
            timestamp: new Date().toISOString(),
            alertSent: false
        }];

        await simulateBufferHealthMessage(page, sessions, 1, 0);

        // Wait for toast notification to appear
        await page.waitForFunction(() => {
            const toasts = document.querySelectorAll('.toast, [role="alert"]');
            return toasts.length > 0;
        }, { timeout: TIMEOUTS.DATA_LOAD });

        // Verify toast was shown (check for toast element)
        const toastCount = await page.locator('.toast, [role="alert"]').count();
        expect(toastCount).toBeGreaterThan(0);
    });

    test('11. Should show risky toast notification', async ({ page }) => {
        if (!await verifyBufferPanelExists(page)) { test.skip(); return; }
        const sessions = [{
            sessionKey: 'toast-risky-test',
            title: 'Toast Risky Movie',
            username: 'User',
            player_device: 'Device',
            bufferFillPercent: 35.0,
            bufferDrainRate: 1.25,
            bufferSeconds: 11.0,
            healthStatus: 'risky',
            riskLevel: 1,
            maxOffsetAvailable: 100000,
            viewOffset: 89000,
            transcodeSpeed: 1.0,
            timestamp: new Date().toISOString(),
            alertSent: false
        }];

        await simulateBufferHealthMessage(page, sessions, 0, 1);

        // Wait for toast notification to appear
        await page.waitForFunction(() => {
            const toasts = document.querySelectorAll('.toast, [role="alert"]');
            return toasts.length > 0;
        }, { timeout: TIMEOUTS.DATA_LOAD });

        // Verify toast was shown
        const toastCount = await page.locator('.toast, [role="alert"]').count();
        expect(toastCount).toBeGreaterThan(0);
    });

    test('12. Should handle multiple sessions sorted by risk level', async ({ page }) => {
        if (!await verifyBufferPanelExists(page)) { test.skip(); return; }
        // Mix of critical and risky sessions
        const sessions = [
            {
                sessionKey: 'risky-1',
                title: 'Risky Movie',
                username: 'User1',
                player_device: 'Device1',
                bufferFillPercent: 40.0,
                bufferDrainRate: 1.2,
                bufferSeconds: 12.0,
                healthStatus: 'risky',
                riskLevel: 1,
                maxOffsetAvailable: 100000,
                viewOffset: 88000,
                transcodeSpeed: 1.0,
                timestamp: new Date().toISOString(),
                alertSent: false
            },
            {
                sessionKey: 'critical-1',
                title: 'Critical Movie',
                username: 'User2',
                player_device: 'Device2',
                bufferFillPercent: 15.0,
                bufferDrainRate: 1.6,
                bufferSeconds: 5.0,
                healthStatus: 'critical',
                riskLevel: 2,
                maxOffsetAvailable: 100000,
                viewOffset: 95000,
                transcodeSpeed: 1.0,
                timestamp: new Date().toISOString(),
                alertSent: false
            },
            {
                sessionKey: 'risky-2',
                title: 'Another Risky Movie',
                username: 'User3',
                player_device: 'Device3',
                bufferFillPercent: 30.0,
                bufferDrainRate: 1.15,
                bufferSeconds: 9.0,
                healthStatus: 'risky',
                riskLevel: 1,
                maxOffsetAvailable: 100000,
                viewOffset: 91000,
                transcodeSpeed: 1.0,
                timestamp: new Date().toISOString(),
                alertSent: false
            }
        ];

        await simulateBufferHealthMessage(page, sessions, 1, 2);

        // Wait for all 3 session cards to render
        await page.waitForFunction(() => {
            const sessionCards = document.querySelectorAll('.buffer-health-session-card');
            return sessionCards.length === 3;
        }, { timeout: TIMEOUTS.RENDER });

        // Should have 3 session cards
        const sessionCards = await page.locator('.buffer-health-session-card').count();
        expect(sessionCards).toBe(3);

        // First card should be critical (highest risk)
        const firstCard = page.locator('.buffer-health-session-card').first();
        const hasCriticalClass = await firstCard.evaluate((el) => el.classList.contains('critical'));
        expect(hasCriticalClass).toBeTruthy();
    });

    test('13. Should log buffer health details to console', async ({ page }) => {
        if (!await verifyBufferPanelExists(page)) { test.skip(); return; }
        // Capture console logs
        const consoleLogs: string[] = [];
        page.on('console', (msg) => {
            consoleLogs.push(msg.text());
        });

        const sessions = [{
            sessionKey: 'log-test',
            title: 'Log Test Movie',
            username: 'LogUser',
            player_device: 'LogDevice',
            bufferFillPercent: 18.0,
            bufferDrainRate: 1.4,
            bufferSeconds: 6.0,
            healthStatus: 'critical',
            riskLevel: 2,
            maxOffsetAvailable: 100000,
            viewOffset: 94000,
            transcodeSpeed: 1.2,
            timestamp: new Date().toISOString(),
            alertSent: false
        }];

        await simulateBufferHealthMessage(page, sessions, 1, 0);

        // Wait for session card to render (which triggers logging)
        await page.waitForFunction(() => {
            const sessionCard = document.querySelector('.buffer-health-session-card');
            const title = document.querySelector('.buffer-health-session-title');
            return sessionCard !== null && title?.textContent?.includes('Log Test Movie');
        }, { timeout: TIMEOUTS.DATA_LOAD });

        // Verify console logs
        const bufferHealthLogs = consoleLogs.filter(log => log.includes('[Buffer Health]'));
        expect(bufferHealthLogs.length).toBeGreaterThan(0);

        // Should log session details
        const detailLogs = bufferHealthLogs.join(' ');
        expect(detailLogs).toContain('LogUser');
        expect(detailLogs).toContain('Log Test Movie');
    });

    test('14. Should handle empty sessions list gracefully', async ({ page }) => {
        if (!await verifyBufferPanelExists(page)) { test.skip(); return; }
        // First show some sessions
        const sessions = [{
            sessionKey: 'temp-session',
            title: 'Temp Movie',
            username: 'User',
            player_device: 'Device',
            bufferFillPercent: 10.0,
            bufferDrainRate: 1.5,
            bufferSeconds: 3.0,
            healthStatus: 'critical',
            riskLevel: 2,
            maxOffsetAvailable: 100000,
            viewOffset: 97000,
            transcodeSpeed: 1.0,
            timestamp: new Date().toISOString(),
            alertSent: false
        }];

        await simulateBufferHealthMessage(page, sessions, 1, 0);

        // Wait for panel to become visible with session card
        // State-based wait: panel visible AND session card exists
        await page.waitForFunction(() => {
            const panel = document.getElementById('buffer-health-panel');
            const sessionCard = document.querySelector('.buffer-health-session-card');
            return panel && !panel.classList.contains('hidden') && sessionCard !== null;
        }, { timeout: TIMEOUTS.MEDIUM }); // 10s for DOM updates in CI

        // Verify panel is visible
        const isVisible = await page.evaluate(() => {
            const panel = document.getElementById('buffer-health-panel');
            return panel && !panel.classList.contains('hidden');
        });
        expect(isVisible).toBeTruthy();

        // Now send empty sessions
        await simulateBufferHealthMessage(page, [], 0, 0);

        // DETERMINISTIC FIX: Wait for panel to become hidden
        // The implementation hides the panel when totalAlerts=0 but returns early,
        // so session cards may remain in the DOM (they're just not visible).
        // This is acceptable UX - what matters is the panel is hidden.
        await page.waitForFunction(() => {
            const panel = document.getElementById('buffer-health-panel');
            return panel?.classList.contains('hidden');
        }, { timeout: TIMEOUTS.MEDIUM }); // 10s for DOM updates in CI

        // Verify panel is hidden again
        const isHidden = await page.evaluate(() => {
            const panel = document.getElementById('buffer-health-panel');
            return panel?.classList.contains('hidden');
        });
        expect(isHidden).toBeTruthy();

        // Optional verification: session cards are no longer visible (even if in DOM)
        // The panel being hidden means cards are not visible to users
        const cardsVisible = await page.locator('.buffer-health-session-card').first().isVisible().catch(() => false);
        expect(cardsVisible).toBe(false);
    });

    test('15. Should filter out healthy sessions from display', async ({ page }) => {
        if (!await verifyBufferPanelExists(page)) { test.skip(); return; }
        // Mix of healthy and critical sessions
        const sessions = [
            {
                sessionKey: 'healthy-1',
                title: 'Healthy Movie',
                username: 'User1',
                player_device: 'Device1',
                bufferFillPercent: 85.0,  // Healthy: above 50%
                bufferDrainRate: 0.9,
                bufferSeconds: 25.0,
                healthStatus: 'healthy',
                riskLevel: 0,
                maxOffsetAvailable: 150000,
                viewOffset: 125000,
                transcodeSpeed: 1.5,
                timestamp: new Date().toISOString(),
                alertSent: false
            },
            {
                sessionKey: 'critical-1',
                title: 'Critical Movie',
                username: 'User2',
                player_device: 'Device2',
                bufferFillPercent: 12.0,
                bufferDrainRate: 1.7,
                bufferSeconds: 4.0,
                healthStatus: 'critical',
                riskLevel: 2,
                maxOffsetAvailable: 100000,
                viewOffset: 96000,
                transcodeSpeed: 1.0,
                timestamp: new Date().toISOString(),
                alertSent: false
            }
        ];

        await simulateBufferHealthMessage(page, sessions, 1, 0);

        // Wait for only the critical session card to render (healthy filtered out)
        await page.waitForFunction(() => {
            const sessionCards = document.querySelectorAll('.buffer-health-session-card');
            const title = document.querySelector('.buffer-health-session-title');
            return sessionCards.length === 1 && title?.textContent?.includes('Critical Movie');
        }, { timeout: TIMEOUTS.RENDER });

        // Should only show 1 card (critical), not the healthy one
        const sessionCards = await page.locator('.buffer-health-session-card').count();
        expect(sessionCards).toBe(1);

        // Verify it's the critical session
        const titleText = await page.textContent('.buffer-health-session-title');
        expect(titleText).toContain('Critical Movie');
    });
});
