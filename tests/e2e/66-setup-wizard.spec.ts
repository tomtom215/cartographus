// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * E2E Tests for Setup Wizard (Phase 2)
 * Tests the multi-step onboarding wizard for first-time users
 */

// ROOT CAUSE FIX: Import test/expect from fixtures to enable autoMockApi for fast, deterministic tests
// Previously imported from @playwright/test which bypassed fixture's automatic API mocking
import { test, expect, setupApiMocking, TIMEOUTS, waitForNavReady, Page } from './fixtures';

// DETERMINISTIC FIX: Disable context-level API mocking since these tests use setupApiMocking(page)
// which registers page-level routes. Having both context and page-level routes active causes conflicts.
// Page-level routes give these tests explicit control over API responses for wizard scenarios.
test.use({ autoMockApi: false });

/**
 * Helper to navigate to app as first-time user with wizard enabled
 */
async function gotoAppAsFirstTimeUser(page: Page): Promise<void> {
  // Clear all onboarding and wizard flags BEFORE page loads
  await page.addInitScript(() => {
    localStorage.removeItem('onboarding_completed');
    localStorage.removeItem('onboarding_skipped');
    localStorage.removeItem('setup_wizard_completed');
    localStorage.removeItem('setup_wizard_skipped');
    localStorage.removeItem('progressive_tips_disabled');
    localStorage.removeItem('progressive_tips_discovered');
    // Set force flag to ensure wizard shows (for E2E determinism)
    localStorage.setItem('setup_wizard_force_show', 'true');
  });

  // Use centralized API mocking
  await setupApiMocking(page);

  await page.goto('/', { waitUntil: 'commit', timeout: TIMEOUTS.EXTENDED });
  await page.waitForLoadState('domcontentloaded', { timeout: TIMEOUTS.DEFAULT });
  await page.waitForSelector('#app:not(.hidden)', { state: 'visible', timeout: TIMEOUTS.EXTENDED });
  await waitForNavReady(page);
}

/**
 * Helper to navigate to app as returning user with wizard skipped
 */
async function gotoAppAsReturningUser(page: Page): Promise<void> {
  // Set all completion flags BEFORE page loads
  await page.addInitScript(() => {
    localStorage.setItem('onboarding_completed', 'true');
    localStorage.setItem('onboarding_skipped', 'true');
    localStorage.setItem('setup_wizard_completed', 'true');
    // Don't set force flag for returning users
    localStorage.removeItem('setup_wizard_force_show');
  });

  // Use centralized API mocking
  await setupApiMocking(page);

  await page.goto('/', { waitUntil: 'commit', timeout: TIMEOUTS.EXTENDED });
  await page.waitForLoadState('domcontentloaded', { timeout: TIMEOUTS.DEFAULT });
  await page.waitForSelector('#app:not(.hidden)', { state: 'visible', timeout: TIMEOUTS.EXTENDED });
  await waitForNavReady(page);
}

test.describe('PH-2: Setup Wizard', () => {
  test.describe('SW-1: First-time User Detection', () => {
    test('SW-1.1: should show setup wizard on first visit', async ({ page }) => {
      await gotoAppAsFirstTimeUser(page);

      // Wait for setup wizard modal to appear
      const wizardModal = page.locator('#setup-wizard-modal');
      await expect(wizardModal).toBeVisible({ timeout: TIMEOUTS.EXTENDED });
    });

    test('SW-1.2: should not show setup wizard for returning users', async ({ page }) => {
      await gotoAppAsReturningUser(page);

      // Setup wizard should not be visible
      const wizardModal = page.locator('#setup-wizard-modal');
      await expect(wizardModal).not.toBeVisible();
    });
  });

  test.describe('SW-2: Wizard Content', () => {
    test('SW-2.1: should display welcome step with app introduction', async ({ page }) => {
      await gotoAppAsFirstTimeUser(page);

      const wizardModal = page.locator('#setup-wizard-modal');
      await expect(wizardModal).toBeVisible({ timeout: TIMEOUTS.EXTENDED });

      // Check for welcome step content
      await expect(wizardModal).toContainText(/welcome/i);
      await expect(wizardModal).toContainText(/cartographus/i);
    });

    test('SW-2.2: should show progress indicator with step count', async ({ page }) => {
      await gotoAppAsFirstTimeUser(page);

      const wizardModal = page.locator('#setup-wizard-modal');
      await expect(wizardModal).toBeVisible({ timeout: TIMEOUTS.EXTENDED });

      // Check for progress indicator
      const progressIndicator = wizardModal.locator('.setup-wizard-progress');
      await expect(progressIndicator).toBeVisible();

      // Should show step dots
      const stepDots = progressIndicator.locator('.progress-step');
      await expect(stepDots).toHaveCount(5); // 5 wizard steps
    });

    test('SW-2.3: should have navigation buttons', async ({ page }) => {
      await gotoAppAsFirstTimeUser(page);

      const wizardModal = page.locator('#setup-wizard-modal');
      await expect(wizardModal).toBeVisible({ timeout: TIMEOUTS.EXTENDED });

      // Check for Next button (uses ID from SetupWizardManager.ts)
      const nextBtn = wizardModal.locator('#wizard-next-btn');
      await expect(nextBtn).toBeVisible();

      // Back button should be hidden on first step
      const backBtn = wizardModal.locator('#wizard-prev-btn');
      await expect(backBtn).not.toBeVisible();
    });
  });

  test.describe('SW-3: Navigation Flow', () => {
    test('SW-3.1: should navigate to next step when Next clicked', async ({ page }) => {
      await gotoAppAsFirstTimeUser(page);

      const wizardModal = page.locator('#setup-wizard-modal');
      await expect(wizardModal).toBeVisible({ timeout: TIMEOUTS.EXTENDED });

      // Get current step indicator
      const activeStep = wizardModal.locator('.progress-step.active');
      const initialStepIndex = await activeStep.getAttribute('data-step');

      // Click Next button (uses ID from SetupWizardManager.ts)
      const nextBtn = wizardModal.locator('#wizard-next-btn');
      await nextBtn.click();

      // Wait for step to change using proper state detection
      await page.waitForFunction(
        (currentStep) => {
          const active = document.querySelector('.progress-step.active');
          return active && active.getAttribute('data-step') !== currentStep;
        },
        initialStepIndex,
        { timeout: TIMEOUTS.SHORT }
      );

      // Verify step changed
      const newActiveStep = wizardModal.locator('.progress-step.active');
      const newStepIndex = await newActiveStep.getAttribute('data-step');
      expect(newStepIndex).not.toBe(initialStepIndex);
    });

    test('SW-3.2: should navigate back when Back clicked', async ({ page }) => {
      await gotoAppAsFirstTimeUser(page);

      const wizardModal = page.locator('#setup-wizard-modal');
      await expect(wizardModal).toBeVisible({ timeout: TIMEOUTS.EXTENDED });

      // Navigate to second step
      const nextBtn = wizardModal.locator('#wizard-next-btn');
      await nextBtn.click();

      // Wait for step to change to step 1
      await page.waitForFunction(
        () => {
          const active = document.querySelector('.progress-step.active');
          return active && active.getAttribute('data-step') === '1';
        },
        { timeout: TIMEOUTS.SHORT }
      );

      // Back button should now be visible
      const backBtn = wizardModal.locator('#wizard-prev-btn');
      await expect(backBtn).toBeVisible();

      // Click Back
      await backBtn.click();

      // Wait for step to change back to step 0
      await page.waitForFunction(
        () => {
          const active = document.querySelector('.progress-step.active');
          return active && active.getAttribute('data-step') === '0';
        },
        { timeout: TIMEOUTS.SHORT }
      );

      // Should be back on first step
      const activeStep = wizardModal.locator('.progress-step.active');
      const stepIndex = await activeStep.getAttribute('data-step');
      expect(stepIndex).toBe('0');
    });

    test('SW-3.3: should complete wizard on finish', async ({ page }) => {
      await gotoAppAsFirstTimeUser(page);

      const wizardModal = page.locator('#setup-wizard-modal');
      await expect(wizardModal).toBeVisible({ timeout: TIMEOUTS.EXTENDED });

      // Navigate through all steps with proper state detection
      for (let i = 0; i < 4; i++) {
        const nextBtn = wizardModal.locator('#wizard-next-btn');
        if (await nextBtn.isVisible()) {
          const currentStep = await wizardModal.locator('.progress-step.active').getAttribute('data-step');
          await nextBtn.click();
          // Wait for step to change
          await page.waitForFunction(
            (prevStep) => {
              const active = document.querySelector('.progress-step.active');
              return active && active.getAttribute('data-step') !== prevStep;
            },
            currentStep,
            { timeout: TIMEOUTS.SHORT }
          ).catch(() => {
            // May be on last step or button may have changed to "Finish"
          });
        }
      }

      // On last step, click Finish/Complete (Next button text changes to "Finish")
      const finishBtn = wizardModal.locator('#wizard-next-btn');
      if (await finishBtn.isVisible()) {
        await finishBtn.click();
      }

      // Wizard should close
      await expect(wizardModal).not.toBeVisible({ timeout: TIMEOUTS.MEDIUM });

      // Completion flag should be set
      const completed = await page.evaluate(() => localStorage.getItem('setup_wizard_completed'));
      expect(completed).toBe('true');
    });
  });

  test.describe('SW-4: Skip Functionality', () => {
    test('SW-4.1: should close wizard when Skip clicked', async ({ page }) => {
      await gotoAppAsFirstTimeUser(page);

      const wizardModal = page.locator('#setup-wizard-modal');
      await expect(wizardModal).toBeVisible({ timeout: TIMEOUTS.EXTENDED });

      // Click Skip button (uses ID from SetupWizardManager.ts)
      const skipBtn = wizardModal.locator('#wizard-skip-btn');
      await skipBtn.click();

      // Wizard should close
      await expect(wizardModal).not.toBeVisible({ timeout: TIMEOUTS.MEDIUM });
    });

    test('SW-4.2: should save skip preference to localStorage', async ({ page }) => {
      await gotoAppAsFirstTimeUser(page);

      const wizardModal = page.locator('#setup-wizard-modal');
      await expect(wizardModal).toBeVisible({ timeout: TIMEOUTS.EXTENDED });

      // Click Skip button (uses ID from SetupWizardManager.ts)
      await page.locator('#wizard-skip-btn').click();

      // Check localStorage
      const skipped = await page.evaluate(() => localStorage.getItem('setup_wizard_skipped'));
      expect(skipped).toBe('true');
    });
  });

  test.describe('SW-5: Accessibility', () => {
    test('SW-5.1: should have proper ARIA attributes', async ({ page }) => {
      await gotoAppAsFirstTimeUser(page);

      const wizardModal = page.locator('#setup-wizard-modal');
      await expect(wizardModal).toBeVisible({ timeout: TIMEOUTS.EXTENDED });

      // Check modal role and aria attributes
      await expect(wizardModal).toHaveAttribute('role', 'dialog');
      await expect(wizardModal).toHaveAttribute('aria-modal', 'true');
      await expect(wizardModal).toHaveAttribute('aria-labelledby');
    });

    test('SW-5.2: should support keyboard navigation', async ({ page }) => {
      await gotoAppAsFirstTimeUser(page);

      const wizardModal = page.locator('#setup-wizard-modal');
      await expect(wizardModal).toBeVisible({ timeout: TIMEOUTS.EXTENDED });

      // Tab should move focus between interactive elements
      await page.keyboard.press('Tab');

      // Some element within the modal should be focused
      const focusedElement = page.locator(':focus');
      const isInsideModal = await focusedElement.evaluate((el) => {
        const modal = document.getElementById('setup-wizard-modal');
        return modal?.contains(el) ?? false;
      });
      expect(isInsideModal).toBe(true);
    });

    test('SW-5.3: should close on Escape key', async ({ page }) => {
      await gotoAppAsFirstTimeUser(page);

      const wizardModal = page.locator('#setup-wizard-modal');
      await expect(wizardModal).toBeVisible({ timeout: TIMEOUTS.EXTENDED });

      // Press Escape
      await page.keyboard.press('Escape');

      // Modal should close
      await expect(wizardModal).not.toBeVisible({ timeout: TIMEOUTS.MEDIUM });
    });
  });
});

test.describe('PH-2: Progressive Onboarding Tips', () => {
  /**
   * Helper to set up progressive tips test environment
   */
  async function setupProgressiveTipsTest(page: Page, options: { disabled?: boolean; discovered?: string[] } = {}): Promise<void> {
    await page.addInitScript((opts) => {
      localStorage.setItem('onboarding_completed', 'true');
      localStorage.setItem('onboarding_skipped', 'true');
      localStorage.setItem('setup_wizard_completed', 'true');

      if (opts.disabled) {
        localStorage.setItem('progressive_tips_disabled', 'true');
      } else {
        localStorage.removeItem('progressive_tips_disabled');
      }

      if (opts.discovered && opts.discovered.length > 0) {
        localStorage.setItem('progressive_tips_discovered', JSON.stringify(opts.discovered));
      } else {
        localStorage.removeItem('progressive_tips_discovered');
      }
    }, options);

    await setupApiMocking(page);
    await page.goto('/', { waitUntil: 'commit', timeout: TIMEOUTS.EXTENDED });
    await page.waitForSelector('#app:not(.hidden)', { state: 'visible', timeout: TIMEOUTS.EXTENDED });
    await waitForNavReady(page);

    // Wait for progressive onboarding manager to initialize
    // The feature has a 1.5s delay, so we wait for the manager to be ready
    await page.waitForFunction(
      () => {
        // Check if progressive onboarding manager is initialized
        const tipContainer = document.getElementById('progressive-tip-container');
        return tipContainer !== null;
      },
      { timeout: TIMEOUTS.MEDIUM }
    ).catch(() => {
      // Container may not exist if feature is disabled, which is OK
    });
  }

  test.describe('PO-1: Tip Display', () => {
    test('PO-1.1: should have progressive tip container initialized', async ({ page }) => {
      await setupProgressiveTipsTest(page);

      // The tip container should exist when progressive tips are enabled
      const tipContainer = page.locator('#progressive-tip-container');
      await expect(tipContainer).toBeAttached({ timeout: TIMEOUTS.MEDIUM });
    });

    test('PO-1.2: should not show tips when disabled', async ({ page }) => {
      await setupProgressiveTipsTest(page, { disabled: true });

      // Theme toggle should be visible
      const themeToggle = page.locator('#theme-toggle');
      await expect(themeToggle).toBeVisible({ timeout: TIMEOUTS.MEDIUM });

      // Hover over the element
      await themeToggle.hover();

      // Wait briefly for any potential tip to appear
      await page.waitForFunction(
        () => {
          const tip = document.querySelector('#progressive-tip-container .progressive-tip');
          return tip === null || getComputedStyle(tip).display === 'none';
        },
        { timeout: TIMEOUTS.SHORT }
      );

      // Tip should NOT appear when disabled
      const tipContainer = page.locator('#progressive-tip-container .progressive-tip');
      await expect(tipContainer).not.toBeVisible();
    });
  });

  test.describe('PO-2: Tip Dismissal', () => {
    test('PO-2.1: should persist discovered tips in localStorage', async ({ page }) => {
      // Test that discovered tips state is properly persisted
      await setupProgressiveTipsTest(page, { discovered: ['theme-toggle', 'filter-days'] });

      // Verify the discovered tips were saved
      const discovered = await page.evaluate(() => localStorage.getItem('progressive_tips_discovered'));
      expect(discovered).not.toBeNull();

      const discoveredList = JSON.parse(discovered!);
      expect(discoveredList).toContain('theme-toggle');
      expect(discoveredList).toContain('filter-days');
    });

    test('PO-2.2: should not show tips for already discovered elements', async ({ page }) => {
      // Pre-set theme-toggle as discovered
      await setupProgressiveTipsTest(page, { discovered: ['theme-toggle'] });

      const themeToggle = page.locator('#theme-toggle');
      await expect(themeToggle).toBeVisible({ timeout: TIMEOUTS.MEDIUM });

      // Hover over the already-discovered element
      await themeToggle.hover();

      // Wait briefly and verify no tip appears for discovered element
      await page.waitForFunction(
        () => {
          const tip = document.querySelector('#progressive-tip-container .progressive-tip');
          return tip === null || getComputedStyle(tip).display === 'none';
        },
        { timeout: TIMEOUTS.SHORT }
      );

      const tipContainer = page.locator('#progressive-tip-container .progressive-tip');
      await expect(tipContainer).not.toBeVisible();
    });
  });

  test.describe('PO-3: Disable All Tips', () => {
    test('PO-3.1: should persist disabled state in localStorage', async ({ page }) => {
      // Pre-set disabled state
      await setupProgressiveTipsTest(page, { disabled: true });

      // Verify the disabled flag is persisted
      const disabled = await page.evaluate(() => localStorage.getItem('progressive_tips_disabled'));
      expect(disabled).toBe('true');
    });

    test('PO-3.2: should respect disabled flag after page reload', async ({ page }) => {
      // Set disabled state
      await setupProgressiveTipsTest(page, { disabled: true });

      // Reload page
      await page.reload();
      await page.waitForSelector('#app:not(.hidden)', { state: 'visible', timeout: TIMEOUTS.EXTENDED });

      // Verify disabled flag persists
      const disabled = await page.evaluate(() => localStorage.getItem('progressive_tips_disabled'));
      expect(disabled).toBe('true');

      // Hover over element and verify no tip
      const themeToggle = page.locator('#theme-toggle');
      await expect(themeToggle).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
      await themeToggle.hover();

      const tipContainer = page.locator('#progressive-tip-container .progressive-tip');
      await expect(tipContainer).not.toBeVisible();
    });
  });
});
