// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * User IP Details Modal Tests
 *
 * Tests the IP History modal on the User Profile page.
 * UserIPDetailsManager is integrated as a modal triggered by "View IP History" button.
 *
 * Test Strategy:
 * - Uses JavaScript clicks for CI reliability (Playwright's .click() can be flaky in headless/SwiftShader)
 * - Uses proper waitFor conditions instead of arbitrary timeouts
 * - All selectors use data-testid for stability
 */

// ROOT CAUSE FIX: Import from fixtures to enable autoMockApi for fast, deterministic tests
// Previously imported from @playwright/test which bypassed API mocking, causing 30+ second test times
import { test, expect, TIMEOUTS } from './fixtures';

test.describe('User IP Details Modal', () => {
    test.beforeEach(async ({ page }) => {
        // Set up E2E flags before navigation
        await page.addInitScript(() => {
            localStorage.setItem('onboarding_completed', 'true');
            localStorage.setItem('onboarding_skipped', 'true');
        });

        await page.goto('/');

        // Wait for app to be ready - verified by nav tabs being visible
        // WHY: The app loads asynchronously; nav-tabs indicates the main app is rendered
        await page.waitForSelector('#nav-tabs', { state: 'visible', timeout: TIMEOUTS.NAVIGATION });

        // Navigate to Analytics view using JavaScript click
        // WHY: JavaScript clicks are more reliable in CI/headless environments than Playwright's .click()
        await page.evaluate(() => {
            const analyticsTab = document.querySelector('.nav-tab[data-view="analytics"]') as HTMLElement;
            if (analyticsTab) analyticsTab.click();
        });

        // Wait for analytics navigation to render
        await page.waitForSelector('#analytics-nav', { state: 'visible', timeout: TIMEOUTS.NAVIGATION });

        // Navigate to User Profile page using JavaScript click
        await page.evaluate(() => {
            const userProfileTab = document.querySelector('[data-analytics-page="users-profile"]') as HTMLElement;
            if (userProfileTab) userProfileTab.click();
        });

        // Wait for User Profile page to be visible
        await page.waitForSelector('#analytics-users-profile', { state: 'visible', timeout: TIMEOUTS.NAVIGATION });
    });

    test('should have View IP History button (hidden by default)', async ({ page }) => {
        // Verify button exists but is hidden when no user is selected
        // WHY: Button should only show when a user is selected
        const ipHistoryBtn = page.locator('[data-testid="btn-view-ip-history"]');
        await expect(ipHistoryBtn).toBeAttached();

        // Button should be hidden (display: none) when no user selected
        await expect(ipHistoryBtn).not.toBeVisible();
    });

    test('should have user selector on User Profile page', async ({ page }) => {
        // Verify user selector exists
        const userSelector = page.locator('#user-profile-selector');
        await expect(userSelector).toBeVisible({ timeout: TIMEOUTS.ELEMENT_VISIBLE });

        // Verify it's a select element
        const tagName = await userSelector.evaluate(el => el.tagName.toLowerCase());
        expect(tagName).toBe('select');
    });

    test('should have IP History modal in the DOM (hidden)', async ({ page }) => {
        // Verify modal exists in DOM but is hidden
        // WHY: Modal is pre-rendered in HTML, hidden until triggered
        const modal = page.locator('[data-testid="user-ip-history-modal"]');
        await expect(modal).toBeAttached();

        // Modal should have aria-hidden="true" when closed
        await expect(modal).toHaveAttribute('aria-hidden', 'true');
    });

    test('should have modal with proper structure', async ({ page }) => {
        const modal = page.locator('[data-testid="user-ip-history-modal"]');

        // Verify modal has dialog role for accessibility
        await expect(modal).toHaveAttribute('role', 'dialog');
        await expect(modal).toHaveAttribute('aria-modal', 'true');

        // Verify modal has title
        const title = page.locator('#ip-history-title');
        await expect(title).toBeAttached();
        await expect(title).toHaveText('IP Address History');

        // Verify close button exists
        const closeBtn = page.locator('#btn-close-ip-history');
        await expect(closeBtn).toBeAttached();

        // Verify container for IP details exists
        const container = page.locator('[data-testid="user-ip-details-container"]');
        await expect(container).toBeAttached();
    });

    test('should have user profile info section with button placement', async ({ page }) => {
        // Verify user profile info section exists
        // WHY: This is where the "View IP History" button is placed
        const userProfileInfo = page.locator('#user-profile-info');
        await expect(userProfileInfo).toBeAttached();

        // Verify the button is inside the profile info section
        const btnInSection = userProfileInfo.locator('[data-testid="btn-view-ip-history"]');
        await expect(btnInSection).toBeAttached();
    });

    test('analytics tabs should not have emojis', async ({ page }) => {
        // Go back to check analytics nav tabs
        // WHY: User requested emoji removal from analytics tabs
        await page.waitForSelector('#analytics-nav', { state: 'visible', timeout: TIMEOUTS.NAVIGATION });

        // Get all analytics tab text content
        const tabs = page.locator('.analytics-tab');
        const tabTexts = await tabs.allTextContents();

        // Verify no emojis in tab text
        // Common analytics emoji characters to check for absence
        const emojiPatterns = [
            /[\u{1F4CA}\u{1F4C8}\u{1F4C9}]/u,  // Chart emojis
            /[\u{1F3AC}\u{1F3A6}]/u,            // Movie/film emojis
            /[\u{1F465}\u{1F464}]/u,            // User emojis
            /[\u{2699}\u{1F527}]/u,             // Gear/settings emojis
            /[\u{1F5FA}\u{1F30D}\u{1F30E}\u{1F30F}]/u,  // Map/globe emojis
            /[\u{1F4DA}\u{1F4D6}]/u,            // Book emojis
            /[\u{1F4E1}\u{1F4FB}]/u             // Antenna/signal emojis
        ];

        for (const text of tabTexts) {
            for (const pattern of emojiPatterns) {
                expect(text).not.toMatch(pattern);
            }
        }

        // Verify expected tab labels exist (without emojis)
        const expectedLabels = ['Overview', 'Content', 'Users', 'Performance', 'Geographic', 'Advanced', 'Library', 'User Profile', 'Tautulli Data'];
        for (const label of expectedLabels) {
            // Check at least one tab contains this label
            const hasLabel = tabTexts.some(text => text.trim().includes(label));
            expect(hasLabel).toBe(true);
        }
    });
});
