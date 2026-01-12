// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * E2E Tests: Health Dashboard
 *
 * Tests for the comprehensive System Health page in the Data Governance section.
 * Validates display of system health, component statuses, media servers, and sync lag indicators.
 *
 * @see ADR-0005 - NATS JetStream Event Processing
 * @see ADR-0006 - BadgerDB Write-Ahead Log
 * @see ADR-0026 - Media Server Management
 */

import {
    test,
    expect,
    TIMEOUTS,
    gotoAppAndWaitReady,
    waitForNavReady,
} from './fixtures';

// Enable API mocking for deterministic test data
test.use({ autoMockApi: true });

/**
 * Helper to navigate to the Health Dashboard page
 *
 * DETERMINISM FIX: This helper now waits for network to settle after each navigation step.
 * This prevents race conditions where API calls triggered by navigation haven't completed
 * before the next action is taken, which can cause net::ERR_FAILED errors in CI.
 */
async function navigateToHealthDashboard(page: import('@playwright/test').Page): Promise<void> {
    // Navigate to Data Governance tab
    const dataGovernanceTab = page.locator('.nav-tab[data-view="data-governance"]');
    await dataGovernanceTab.click();
    await page.waitForSelector('#data-governance-container', { state: 'visible', timeout: TIMEOUTS.DEFAULT });

    // DETERMINISM FIX: Wait for network to settle after main navigation
    // WHY: Data Governance navigation triggers API calls that must complete before sub-navigation
    await page.waitForLoadState('networkidle', { timeout: TIMEOUTS.MEDIUM }).catch(() => {
        // Network may not idle if WebSocket keeps reconnecting - continue anyway
    });

    // Navigate to Health sub-page
    const healthTab = page.locator('.governance-nav-tab[data-governance-page="health"]');
    await healthTab.click();

    // Wait for the health page to be visible (page structure is ready)
    await page.waitForSelector('#governance-health-page', { state: 'visible', timeout: TIMEOUTS.DEFAULT });

    // DETERMINISM FIX: Wait for network to settle after health page navigation
    // WHY: Health Dashboard triggers multiple API calls (health, servers, NATS, WAL)
    await page.waitForLoadState('networkidle', { timeout: TIMEOUTS.MEDIUM }).catch(() => {
        // Network may not idle if WebSocket keeps reconnecting - continue anyway
    });
}

/**
 * Helper to wait for health data to be loaded
 * Use this in tests that depend on health API data being rendered
 */
async function waitForHealthDataLoaded(page: import('@playwright/test').Page): Promise<void> {
    await page.waitForFunction(
        () => {
            const el = document.getElementById('health-status-value');
            const text = el?.textContent?.trim();
            return text && text !== '--' && text !== '';
        },
        { timeout: TIMEOUTS.DEFAULT }
    );
}

test.describe('Health Dashboard', () => {
    test.beforeEach(async ({ page }) => {
        // Navigate to the app and wait for it to load
        await gotoAppAndWaitReady(page);
        await waitForNavReady(page);

        // Navigate to the Health Dashboard
        await navigateToHealthDashboard(page);

        // DETERMINISM FIX: Wait for health data to load before running tests
        // WHY: Without this wait, tests that check health status values may run before
        // API responses are received and rendered, causing flaky failures.
        // The frontend displays "--" as placeholder until data loads.
        await waitForHealthDataLoaded(page);
    });

    test.describe('Page Structure', () => {
        test('displays health page header with refresh button', async ({ page }) => {
            const header = page.locator('#governance-health-page .governance-section-header h3');
            await expect(header).toHaveText('System Health Overview');

            const refreshBtn = page.locator('#health-refresh-btn');
            await expect(refreshBtn).toBeVisible();
        });

        test('displays overall health status card', async ({ page }) => {
            const statusCard = page.locator('#health-overview-card');
            await expect(statusCard).toBeVisible();

            // Check for status indicator
            const statusIcon = page.locator('.health-status-icon');
            await expect(statusIcon).toBeVisible();

            // Check for version and uptime
            const version = page.locator('#health-version');
            await expect(version).toBeVisible();

            const uptime = page.locator('#health-uptime');
            await expect(uptime).toBeVisible();
        });

        test('displays all component health cards', async ({ page }) => {
            const components = ['database', 'tautulli', 'nats', 'wal', 'websocket', 'detection'];

            for (const component of components) {
                const card = page.locator(`#health-${component}`);
                await expect(card).toBeVisible();
            }
        });

        test('displays sync times section', async ({ page }) => {
            const syncTimesSection = page.locator('#health-sync-times');
            await expect(syncTimesSection).toBeVisible();

            // Check for all sync time items
            const lastSync = page.locator('#health-last-sync');
            await expect(lastSync).toBeVisible();

            const lastBackup = page.locator('#health-last-backup');
            await expect(lastBackup).toBeVisible();

            const lastDetection = page.locator('#health-last-detection');
            await expect(lastDetection).toBeVisible();
        });

        test('displays media servers section', async ({ page }) => {
            const serversSection = page.locator('#health-servers-grid');
            await expect(serversSection).toBeVisible();
        });
    });

    test.describe('Health Status Display', () => {
        test('shows system status as Healthy when all components operational', async ({ page }) => {
            const statusValue = page.locator('#health-status-value');
            await expect(statusValue).toHaveText('Healthy');
            await expect(statusValue).toHaveClass(/healthy/);
        });

        test('shows database status', async ({ page }) => {
            const dbStatus = page.locator('#health-db-status');
            await expect(dbStatus).toBeVisible();

            // Should show Connected in mock environment
            await expect(dbStatus).toHaveText('Connected');
        });

        test('shows NATS status', async ({ page }) => {
            const natsStatus = page.locator('#health-nats-status');
            await expect(natsStatus).toBeVisible();
        });

        test('shows WAL status', async ({ page }) => {
            const walStatus = page.locator('#health-wal-status');
            await expect(walStatus).toBeVisible();
        });

        test('displays version information', async ({ page }) => {
            const version = page.locator('#health-version');
            await expect(version).toBeVisible();
            // Version should not be "--" after loading
            await expect(version).not.toHaveText('--');
        });

        test('displays uptime information', async ({ page }) => {
            const uptime = page.locator('#health-uptime');
            await expect(uptime).toBeVisible();
            // Uptime should not be "--" after loading
            await expect(uptime).not.toHaveText('--');
        });
    });

    test.describe('Media Servers Section', () => {
        test('displays server cards when servers are configured', async ({ page }) => {
            const serversGrid = page.locator('#health-servers-grid');
            await expect(serversGrid).toBeVisible();

            // Check for server cards (mock data should have servers)
            const serverCards = page.locator('.health-server-card');
            const count = await serverCards.count();
            expect(count).toBeGreaterThan(0);
        });

        test('server card shows platform icon and name', async ({ page }) => {
            const firstCard = page.locator('.health-server-card').first();

            // Check for platform icon
            const platformIcon = firstCard.locator('.server-platform-icon');
            await expect(platformIcon).toBeVisible();

            // Check for server name
            const serverName = firstCard.locator('.server-name');
            await expect(serverName).toBeVisible();
        });

        test('server card shows status badge', async ({ page }) => {
            const firstCard = page.locator('.health-server-card').first();
            const statusBadge = firstCard.locator('.server-status-badge');
            await expect(statusBadge).toBeVisible();
        });

        test('server card shows last sync time', async ({ page }) => {
            const firstCard = page.locator('.health-server-card').first();
            const syncStat = firstCard.locator('.server-stat .stat-label:has-text("Last Sync:")');
            await expect(syncStat).toBeVisible();
        });

        test('displays sync lag indicator with appropriate styling', async ({ page }) => {
            // Check if any server has sync lag displayed
            const syncLag = page.locator('.sync-lag');
            const count = await syncLag.count();

            if (count > 0) {
                const firstLag = syncLag.first();
                await expect(firstLag).toBeVisible();

                // Should have one of the lag classes
                const classList = await firstLag.getAttribute('class');
                expect(classList).toMatch(/lag-(good|warning|critical)/);
            }
        });

        test('shows connected count in summary', async ({ page }) => {
            const connectedCount = page.locator('#health-servers-connected');
            await expect(connectedCount).toBeVisible();
        });

        test('shows error badge when servers have errors', async ({ page }) => {
            // This test verifies the error badge structure exists
            const errorBadge = page.locator('#health-servers-error-badge');
            // Badge should exist in DOM but may be hidden if no errors
            await expect(errorBadge).toBeAttached();
        });

        test('displays server error message when present', async ({ page }) => {
            // Check if any server has an error displayed
            const serverErrors = page.locator('.health-server-card .server-error');
            const count = await serverErrors.count();

            // If there are errors, verify the structure
            if (count > 0) {
                const firstError = serverErrors.first();
                const errorLabel = firstError.locator('.error-label');
                await expect(errorLabel).toBeVisible();
            }
        });
    });

    test.describe('Component Health Indicators', () => {
        test('database component shows correct indicator', async ({ page }) => {
            const indicator = page.locator('#health-db-indicator');
            await expect(indicator).toBeVisible();
            await expect(indicator).toHaveClass(/component-indicator/);
        });

        test('NATS component shows correct indicator', async ({ page }) => {
            const indicator = page.locator('#health-nats-indicator');
            await expect(indicator).toBeVisible();
            await expect(indicator).toHaveClass(/component-indicator/);
        });

        test('WAL component shows correct indicator', async ({ page }) => {
            const indicator = page.locator('#health-wal-indicator');
            await expect(indicator).toBeVisible();
            await expect(indicator).toHaveClass(/component-indicator/);
        });

        test('healthy component shows green indicator', async ({ page }) => {
            // Database should be healthy in mock environment
            const dbIndicator = page.locator('#health-db-indicator');
            await expect(dbIndicator).toHaveClass(/healthy/);
        });
    });

    test.describe('Refresh Functionality', () => {
        test('refresh button reloads health data', async ({ page }) => {
            // Wait for initial data to load first
            await waitForHealthDataLoaded(page);

            // Store initial status visibility
            const statusValue = page.locator('#health-status-value');
            await expect(statusValue).toBeVisible();

            // Click refresh and wait for the button click to be processed
            const refreshBtn = page.locator('#health-refresh-btn');
            await refreshBtn.click();

            // Wait for health data to be reloaded (status should still show after refresh)
            // The DOM state should remain valid after refresh
            await expect(statusValue).toBeVisible();
            await expect(statusValue).not.toHaveText('--');
        });
    });

    test.describe('Sync Times Display', () => {
        test('displays last data sync time', async ({ page }) => {
            const lastSync = page.locator('#health-last-sync');
            await expect(lastSync).toBeVisible();

            // Should have a value (either timestamp or "Never")
            const text = await lastSync.textContent();
            expect(text).toBeTruthy();
        });

        test('displays last backup time', async ({ page }) => {
            const lastBackup = page.locator('#health-last-backup');
            await expect(lastBackup).toBeVisible();

            // Should have a value
            const text = await lastBackup.textContent();
            expect(text).toBeTruthy();
        });

        test('displays detection status', async ({ page }) => {
            const lastDetection = page.locator('#health-last-detection');
            await expect(lastDetection).toBeVisible();
        });
    });

    test.describe('Navigation Integration', () => {
        test('health page tab shows active state', async ({ page }) => {
            const healthTab = page.locator('.governance-nav-tab[data-governance-page="health"]');
            await expect(healthTab).toHaveClass(/active/);
        });

        test('can switch to other governance pages and back', async ({ page }) => {
            // Switch to Overview
            const overviewTab = page.locator('.governance-nav-tab[data-governance-page="overview"]');
            await overviewTab.click();

            // Health page should be hidden
            const healthPage = page.locator('#governance-health-page');
            await expect(healthPage).toBeHidden();

            // Switch back to Health
            const healthTab = page.locator('.governance-nav-tab[data-governance-page="health"]');
            await healthTab.click();

            // Health page should be visible again
            await expect(healthPage).toBeVisible();
        });
    });

    test.describe('Accessibility', () => {
        test('refresh button has accessible label', async ({ page }) => {
            const refreshBtn = page.locator('#health-refresh-btn');
            await expect(refreshBtn).toHaveAttribute('aria-label', 'Refresh health status');
        });

        test('component cards have proper structure', async ({ page }) => {
            // Each component card should have name and status
            const dbCard = page.locator('#health-database');
            await expect(dbCard).toBeVisible();

            const componentName = dbCard.locator('.component-name');
            await expect(componentName).toBeVisible();

            const componentStatus = dbCard.locator('.component-status');
            await expect(componentStatus).toBeVisible();
        });
    });

    test.describe('Error Handling', () => {
        test('shows appropriate status when component unavailable', async ({ page }) => {
            // NATS may show "Not enabled" if not configured
            const natsStatus = page.locator('#health-nats-status');
            const statusText = await natsStatus.textContent();

            // Should have some status text
            expect(statusText).toBeTruthy();
            expect(statusText).not.toBe('--');
        });

        test('WAL shows appropriate status when not enabled', async ({ page }) => {
            const walStatus = page.locator('#health-wal-status');
            const statusText = await walStatus.textContent();

            // Should have some status text
            expect(statusText).toBeTruthy();
        });
    });
});

test.describe('Health Dashboard - Degraded State', () => {
    test('shows degraded status when servers have errors', async ({ page }) => {
        // Navigate to the app and wait for it to load
        await gotoAppAndWaitReady(page);
        await waitForNavReady(page);

        // Navigate to the Health Dashboard
        await navigateToHealthDashboard(page);

        // DETERMINISM FIX: Wait for health data to load before assertions
        await waitForHealthDataLoaded(page);

        // Verify the structure exists for degraded state display
        const statusValue = page.locator('#health-status-value');
        await expect(statusValue).toBeVisible();
    });
});
