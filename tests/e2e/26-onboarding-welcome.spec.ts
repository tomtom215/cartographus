// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * * E2E Tests for Onboarding/Welcome Experience
 * TDD: These tests are written first, implementation follows
 */

import {
  test,
  expect,
  TIMEOUTS,
  waitForNavReady,
} from './fixtures';

/**
 * Helper to navigate to app with onboarding flags cleared (simulating first-time user)
 * Uses addInitScript to ensure flags are cleared BEFORE page JavaScript runs
 * Note: The setup wizard now shows first, so we skip the wizard to test onboarding directly
 */
async function gotoAppAsFirstTimeUser(page: import('@playwright/test').Page): Promise<void> {
  // Clear onboarding flags and skip wizard to test onboarding directly BEFORE page loads
  await page.addInitScript(() => {
    localStorage.removeItem('onboarding_completed');
    localStorage.removeItem('onboarding_skipped');
    // Skip the setup wizard to test onboarding flow directly
    localStorage.setItem('setup_wizard_completed', 'true');
    // Disable progressive tips to avoid interference
    localStorage.setItem('progressive_tips_disabled', 'true');
  });

  await page.goto('/', { waitUntil: 'commit', timeout: TIMEOUTS.EXTENDED });
  await page.waitForLoadState('domcontentloaded', { timeout: TIMEOUTS.DEFAULT });
  await page.waitForSelector('#app:not(.hidden)', { state: 'visible', timeout: TIMEOUTS.EXTENDED });
  await waitForNavReady(page);

  // Wait for OnboardingManager.init() to run and check for modal state
  // Either modal appears (first-time user) or doesn't (check completes)
  await page.waitForFunction(
    () => {
      const modal = document.querySelector('#onboarding-modal');
      return modal !== null; // Modal element exists in DOM
    },
    { timeout: TIMEOUTS.DEFAULT }
  );
}

/**
 * Helper to navigate to app with onboarding completed (simulating returning user)
 */
async function gotoAppAsReturningUser(page: import('@playwright/test').Page): Promise<void> {
  // Set all completion flags BEFORE page loads
  await page.addInitScript(() => {
    localStorage.setItem('onboarding_completed', 'true');
    localStorage.setItem('onboarding_skipped', 'true');
    localStorage.setItem('setup_wizard_completed', 'true');
    // Disable progressive tips to avoid interference
    localStorage.setItem('progressive_tips_disabled', 'true');
  });

  await page.goto('/', { waitUntil: 'commit', timeout: TIMEOUTS.EXTENDED });
  await page.waitForLoadState('domcontentloaded', { timeout: TIMEOUTS.DEFAULT });
  await page.waitForSelector('#app:not(.hidden)', { state: 'visible', timeout: TIMEOUTS.EXTENDED });
  await waitForNavReady(page);

  // Wait for OnboardingManager.init() to complete by checking initialization state
  await page.waitForFunction(
    () => {
      // OnboardingManager should have completed initialization
      // Check that the onboarding check has been performed
      const hasCompletedFlag = localStorage.getItem('onboarding_completed');
      const hasSkippedFlag = localStorage.getItem('onboarding_skipped');
      return hasCompletedFlag !== null || hasSkippedFlag !== null;
    },
    { timeout: TIMEOUTS.DEFAULT }
  );
}

test.describe('Onboarding/Welcome Experience', () => {
  test.describe('First-time User Detection', () => {
    test('should show welcome modal on first visit', async ({ page }) => {
      await gotoAppAsFirstTimeUser(page);

      // Wait for welcome modal to appear
      const welcomeModal = page.locator('#onboarding-modal');
      await expect(welcomeModal).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
    });

    test('should not show welcome modal for returning users', async ({ page }) => {
      await gotoAppAsReturningUser(page);

      // Welcome modal should not be visible
      const welcomeModal = page.locator('#onboarding-modal');
      await expect(welcomeModal).not.toBeVisible();
    });
  });

  test.describe('Welcome Modal Content', () => {
    test('should display welcome message and app name', async ({ page }) => {
      await gotoAppAsFirstTimeUser(page);

      const welcomeModal = page.locator('#onboarding-modal');
      await expect(welcomeModal).toBeVisible({ timeout: TIMEOUTS.MEDIUM });

      // Check for welcome title
      const title = welcomeModal.locator('.onboarding-title');
      await expect(title).toContainText(/welcome/i);

      // Check for app name
      await expect(welcomeModal).toContainText(/cartographus/i);
    });

    test('should have Start Tour and Skip buttons', async ({ page }) => {
      await gotoAppAsFirstTimeUser(page);

      const welcomeModal = page.locator('#onboarding-modal');
      await expect(welcomeModal).toBeVisible({ timeout: TIMEOUTS.MEDIUM });

      // Check for Start Tour button
      const startBtn = welcomeModal.locator('#onboarding-start-btn');
      await expect(startBtn).toBeVisible();
      await expect(startBtn).toHaveText(/start|tour|get started/i);

      // Check for Skip button
      const skipBtn = welcomeModal.locator('#onboarding-skip-btn');
      await expect(skipBtn).toBeVisible();
      await expect(skipBtn).toHaveText(/skip|later|dismiss/i);
    });

    test('should display key feature highlights', async ({ page }) => {
      await gotoAppAsFirstTimeUser(page);

      const welcomeModal = page.locator('#onboarding-modal');
      await expect(welcomeModal).toBeVisible({ timeout: TIMEOUTS.MEDIUM });

      // Check for feature list
      const features = welcomeModal.locator('.onboarding-feature');
      await expect(features).toHaveCount(4); // Map, Analytics, Activity, Themes
    });
  });

  test.describe('Skip Functionality', () => {
    test('should close modal when Skip clicked', async ({ page }) => {
      await gotoAppAsFirstTimeUser(page);

      const welcomeModal = page.locator('#onboarding-modal');
      await expect(welcomeModal).toBeVisible({ timeout: TIMEOUTS.MEDIUM });

      // Click Skip button
      const skipBtn = page.locator('#onboarding-skip-btn');
      await skipBtn.click();

      // Modal should close
      await expect(welcomeModal).not.toBeVisible();
    });

    test('should save skip preference to localStorage', async ({ page }) => {
      await gotoAppAsFirstTimeUser(page);

      const welcomeModal = page.locator('#onboarding-modal');
      await expect(welcomeModal).toBeVisible({ timeout: TIMEOUTS.MEDIUM });

      // Click Skip button
      await page.locator('#onboarding-skip-btn').click();

      // Check localStorage
      const skipped = await page.evaluate(() => localStorage.getItem('onboarding_skipped'));
      expect(skipped).toBe('true');
    });
  });

  test.describe('Tour Flow', () => {
    test('should start tour when Start Tour clicked', async ({ page }) => {
      await gotoAppAsFirstTimeUser(page);

      const welcomeModal = page.locator('#onboarding-modal');
      await expect(welcomeModal).toBeVisible({ timeout: TIMEOUTS.MEDIUM });

      // Click Start Tour button
      const startBtn = page.locator('#onboarding-start-btn');
      await startBtn.click();

      // Welcome modal should close
      await expect(welcomeModal).not.toBeVisible();

      // Tour tooltip should appear
      const tourTooltip = page.locator('.onboarding-tooltip');
      await expect(tourTooltip).toBeVisible();
    });

    test('should have Next and Previous navigation', async ({ page }) => {
      await gotoAppAsFirstTimeUser(page);

      const welcomeModal = page.locator('#onboarding-modal');
      await expect(welcomeModal).toBeVisible({ timeout: TIMEOUTS.MEDIUM });

      // Start tour
      await page.locator('#onboarding-start-btn').click();

      // Check for navigation buttons
      const nextBtn = page.locator('.onboarding-next-btn');
      await expect(nextBtn).toBeVisible();

      // Previous should be hidden on first step
      const prevBtn = page.locator('.onboarding-prev-btn');
      await expect(prevBtn).not.toBeVisible();
    });

    test('should mark tour completed on finish', async ({ page }) => {
      await gotoAppAsFirstTimeUser(page);

      const welcomeModal = page.locator('#onboarding-modal');
      await expect(welcomeModal).toBeVisible({ timeout: TIMEOUTS.MEDIUM });

      // Start tour
      await page.locator('#onboarding-start-btn').click();

      // Navigate through tour steps (8 steps covering all major features)
      for (let i = 0; i < 8; i++) {
        const nextBtn = page.locator('.onboarding-next-btn');
        if (await nextBtn.isVisible()) {
          await nextBtn.click();
          // Wait for tooltip to update or tour to complete
          await page.waitForFunction(
            (stepIndex) => {
              const tooltip = document.querySelector('.onboarding-tooltip');
              // Either tooltip updated to next step or tour completed (tooltip gone)
              return tooltip === null || tooltip.getAttribute('data-step') !== stepIndex.toString();
            },
            i,
            { timeout: TIMEOUTS.DEFAULT }
          );
        }
      }

      // Check localStorage for completion
      const completed = await page.evaluate(() => localStorage.getItem('onboarding_completed'));
      expect(completed).toBe('true');
    });
  });

  test.describe('Accessibility', () => {
    test('should have proper ARIA attributes', async ({ page }) => {
      await gotoAppAsFirstTimeUser(page);

      const welcomeModal = page.locator('#onboarding-modal');
      await expect(welcomeModal).toBeVisible({ timeout: TIMEOUTS.MEDIUM });

      // Check modal role and aria-label
      await expect(welcomeModal).toHaveAttribute('role', 'dialog');
      await expect(welcomeModal).toHaveAttribute('aria-modal', 'true');
      await expect(welcomeModal).toHaveAttribute('aria-labelledby');
    });

    test('should trap focus within modal', async ({ page }) => {
      await gotoAppAsFirstTimeUser(page);

      const welcomeModal = page.locator('#onboarding-modal');
      await expect(welcomeModal).toBeVisible({ timeout: TIMEOUTS.MEDIUM });

      // Focus should be on first focusable element
      // Note: OnboardingManager.showWelcomeModal() uses setTimeout(100ms) to set focus
      const startBtn = page.locator('#onboarding-start-btn');

      // Wait for focus to be applied (OnboardingManager has 100ms delay)
      await page.waitForFunction(
        (selector) => document.querySelector(selector) === document.activeElement,
        '#onboarding-start-btn',
        { timeout: TIMEOUTS.DEFAULT }
      );

      await expect(startBtn).toBeFocused();
    });

    test('should close on Escape key', async ({ page }) => {
      await gotoAppAsFirstTimeUser(page);

      const welcomeModal = page.locator('#onboarding-modal');
      await expect(welcomeModal).toBeVisible({ timeout: TIMEOUTS.MEDIUM });

      // Press Escape
      await page.keyboard.press('Escape');

      // Modal should close
      await expect(welcomeModal).not.toBeVisible();
    });
  });

  test.describe('Manual Access', () => {
    test('should have Help button to restart tour', async ({ page }) => {
      await gotoAppAsReturningUser(page);

      // Look for help/tour button in header
      const helpBtn = page.locator('#help-tour-btn');
      await expect(helpBtn).toBeVisible();
    });
  });
});
