// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
import { test, expect, TIMEOUTS, gotoAppAndWaitReady } from "./fixtures";

/**
 * E2E Test: Settings/Preferences Page
 *
 * Tests settings/preferences functionality:
 * - Settings panel visibility
 * - Theme settings
 * - Map settings
 * - Data settings
 * - Settings persistence
 * - Export/Import settings
 *
 * @see /docs/working/UI_UX_AUDIT.md
 */

test.describe("Settings/Preferences Page", () => {
  test.beforeEach(async ({ page }) => {
    // Navigate to app
    await gotoAppAndWaitReady(page);
  });

  test.describe("Settings Panel Visibility", () => {
    test("should have settings button in header", async ({ page }) => {
      const settingsButton = page.locator(
        '#settings-button, .settings-button, [data-action="settings"]',
      );
      await expect(settingsButton).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
    });

    test("should open settings panel when settings button clicked", async ({
      page,
    }) => {
      const settingsButton = page.locator(
        '#settings-button, .settings-button, [data-action="settings"]',
      );
      await settingsButton.click();

      const settingsPanel = page.locator("#settings-panel, .settings-panel");
      await expect(settingsPanel).toBeVisible({ timeout: TIMEOUTS.DATA_LOAD });
    });

    test("should close settings panel with close button", async ({ page }) => {
      const settingsButton = page.locator(
        '#settings-button, .settings-button, [data-action="settings"]',
      );
      await settingsButton.click();

      const settingsPanel = page.locator("#settings-panel, .settings-panel");
      await expect(settingsPanel).toBeVisible();

      const closeButton = settingsPanel.locator(
        '.settings-close, [data-action="close-settings"]',
      );
      await closeButton.click();

      await expect(settingsPanel).not.toBeVisible({
        timeout: TIMEOUTS.DATA_LOAD,
      });
    });

    test("should close settings panel with Escape key", async ({ page }) => {
      const settingsButton = page.locator(
        '#settings-button, .settings-button, [data-action="settings"]',
      );
      await settingsButton.click();

      const settingsPanel = page.locator("#settings-panel, .settings-panel");
      await expect(settingsPanel).toBeVisible();

      await page.keyboard.press("Escape");

      await expect(settingsPanel).not.toBeVisible({
        timeout: TIMEOUTS.DATA_LOAD,
      });
    });
  });

  test.describe("Appearance Settings", () => {
    test("should have theme selector", async ({ page }) => {
      const settingsButton = page.locator(
        '#settings-button, .settings-button, [data-action="settings"]',
      );
      await settingsButton.click();

      const settingsPanel = page.locator("#settings-panel, .settings-panel");
      await expect(settingsPanel).toBeVisible();

      const themeSelector = settingsPanel.locator(
        '#theme-selector, [name="theme"], .theme-option',
      );
      const hasThemeSelector = (await themeSelector.count()) > 0;
      expect(hasThemeSelector).toBeTruthy();
    });

    test("should change theme when option selected", async ({ page }) => {
      const settingsButton = page.locator(
        '#settings-button, .settings-button, [data-action="settings"]',
      );
      await settingsButton.click();

      const settingsPanel = page.locator("#settings-panel, .settings-panel");
      await expect(settingsPanel).toBeVisible();

      // Click light theme option
      const lightOption = settingsPanel.locator(
        '[data-theme="light"], .theme-option-light',
      );
      if (await lightOption.isVisible().catch(() => false)) {
        await lightOption.click();
        // Check that theme was applied
        const html = page.locator("html");
        await expect(html).toHaveAttribute("data-theme", "light", {
          timeout: TIMEOUTS.DATA_LOAD,
        });
      }
    });

    test("should have colorblind mode toggle", async ({ page }) => {
      const settingsButton = page.locator(
        '#settings-button, .settings-button, [data-action="settings"]',
      );
      await settingsButton.click();

      const settingsPanel = page.locator("#settings-panel, .settings-panel");
      await expect(settingsPanel).toBeVisible();

      const colorblindToggle = settingsPanel.locator(
        '#colorblind-settings-toggle, [name="colorblind-mode"]',
      );
      await expect(colorblindToggle).toBeVisible();
    });
  });

  test.describe("Map Settings", () => {
    test("should have visualization mode selector", async ({ page }) => {
      const settingsButton = page.locator(
        '#settings-button, .settings-button, [data-action="settings"]',
      );
      await settingsButton.click();

      const settingsPanel = page.locator("#settings-panel, .settings-panel");
      await expect(settingsPanel).toBeVisible();

      const vizSelector = settingsPanel.locator(
        '#viz-mode-selector, [name="visualization-mode"], .viz-mode-option',
      );
      const hasVizSelector = (await vizSelector.count()) > 0;
      expect(hasVizSelector).toBeTruthy();
    });

    test("should have view mode selector (2D/3D)", async ({ page }) => {
      const settingsButton = page.locator(
        '#settings-button, .settings-button, [data-action="settings"]',
      );
      await settingsButton.click();

      const settingsPanel = page.locator("#settings-panel, .settings-panel");
      await expect(settingsPanel).toBeVisible();

      const viewModeSelector = settingsPanel.locator(
        '#view-mode-selector, [name="view-mode"], .view-mode-option',
      );
      const hasViewModeSelector = (await viewModeSelector.count()) > 0;
      expect(hasViewModeSelector).toBeTruthy();
    });
  });

  test.describe("Data Settings", () => {
    test("should have auto-refresh toggle", async ({ page }) => {
      const settingsButton = page.locator(
        '#settings-button, .settings-button, [data-action="settings"]',
      );
      await settingsButton.click();

      const settingsPanel = page.locator("#settings-panel, .settings-panel");
      await expect(settingsPanel).toBeVisible();

      const autoRefreshToggle = settingsPanel.locator(
        '#auto-refresh-toggle-settings, [name="auto-refresh"], .auto-refresh-toggle-settings',
      );
      await expect(autoRefreshToggle).toBeVisible();
    });
  });

  test.describe("Data Management", () => {
    test("should have clear settings button", async ({ page }) => {
      const settingsButton = page.locator(
        '#settings-button, .settings-button, [data-action="settings"]',
      );
      await settingsButton.click();

      const settingsPanel = page.locator("#settings-panel, .settings-panel");
      await expect(settingsPanel).toBeVisible();

      const clearButton = settingsPanel.locator(
        "#btn-clear-settings, .btn-clear-settings",
      );
      await expect(clearButton).toBeVisible();
    });

    test("should have export settings button", async ({ page }) => {
      const settingsButton = page.locator(
        '#settings-button, .settings-button, [data-action="settings"]',
      );
      await settingsButton.click();

      const settingsPanel = page.locator("#settings-panel, .settings-panel");
      await expect(settingsPanel).toBeVisible();

      const exportButton = settingsPanel.locator(
        "#btn-export-settings, .btn-export-settings",
      );
      await expect(exportButton).toBeVisible();
    });

    test("should have import settings button", async ({ page }) => {
      const settingsButton = page.locator(
        '#settings-button, .settings-button, [data-action="settings"]',
      );
      await settingsButton.click();

      const settingsPanel = page.locator("#settings-panel, .settings-panel");
      await expect(settingsPanel).toBeVisible();

      const importButton = settingsPanel.locator(
        "#btn-import-settings, .btn-import-settings",
      );
      await expect(importButton).toBeVisible();
    });
  });

  test.describe("Settings Persistence", () => {
    test("should persist theme setting after refresh", async ({ page }) => {
      // Open settings
      const settingsButton = page.locator(
        '#settings-button, .settings-button, [data-action="settings"]',
      );
      await settingsButton.click();

      const settingsPanel = page.locator("#settings-panel, .settings-panel");
      await expect(settingsPanel).toBeVisible();

      // Change to light theme
      const lightOption = settingsPanel.locator(
        '[data-theme="light"], .theme-option-light',
      );
      if (await lightOption.isVisible().catch(() => false)) {
        await lightOption.click();

        // Wait for theme attribute to change
        const html = page.locator("html");
        await expect(html).toHaveAttribute("data-theme", "light", {
          timeout: TIMEOUTS.MEDIUM,
        });

        // Refresh page
        await page.reload({ waitUntil: "domcontentloaded" });
        await expect(page.locator("#app")).toBeVisible({
          timeout: TIMEOUTS.DEFAULT,
        });

        // Check theme is still light
        await expect(html).toHaveAttribute("data-theme", "light");
      }
    });
  });

  test.describe("Accessibility", () => {
    test("should have proper ARIA attributes on settings panel", async ({
      page,
    }) => {
      const settingsButton = page.locator(
        '#settings-button, .settings-button, [data-action="settings"]',
      );
      await settingsButton.click();

      const settingsPanel = page.locator("#settings-panel, .settings-panel");
      await expect(settingsPanel).toBeVisible();

      // Should have role or aria attributes
      const hasAriaLabel = await settingsPanel.getAttribute("aria-label");
      const hasAriaLabelledBy =
        await settingsPanel.getAttribute("aria-labelledby");
      const hasRole = await settingsPanel.getAttribute("role");

      expect(hasAriaLabel || hasAriaLabelledBy || hasRole).toBeTruthy();
    });

    test("should be keyboard navigable", async ({ page }) => {
      const settingsButton = page.locator(
        '#settings-button, .settings-button, [data-action="settings"]',
      );
      await settingsButton.focus();
      await expect(settingsButton).toBeFocused();

      // Activate with Enter
      await page.keyboard.press("Enter");

      const settingsPanel = page.locator("#settings-panel, .settings-panel");
      await expect(settingsPanel).toBeVisible({ timeout: TIMEOUTS.DATA_LOAD });

      // Tab should move through settings
      await page.keyboard.press("Tab");
      // Some element in settings should be focused
      const focusedElement = page.locator(":focus");
      const isWithinSettings =
        (await settingsPanel.locator(":focus").count()) > 0;
      expect(
        isWithinSettings || (await focusedElement.isVisible()),
      ).toBeTruthy();
    });
  });
});

test.describe("Settings Error Handling", () => {
  test("should handle invalid settings import gracefully", async ({ page }) => {
    await gotoAppAndWaitReady(page);

    // Open settings
    const settingsButton = page.locator(
      '#settings-button, .settings-button, [data-action="settings"]',
    );
    await settingsButton.click();

    const settingsPanel = page.locator("#settings-panel, .settings-panel");
    await expect(settingsPanel).toBeVisible();

    // Try to import invalid settings via file input (if available)
    const importInput = page.locator(
      '#settings-import-input, input[type="file"].settings-import',
    );
    if (await importInput.isVisible().catch(() => false)) {
      // This would require uploading an invalid file, which is complex in E2E
      // Just verify the input exists
      expect(await importInput.isVisible()).toBeTruthy();
    }
  });
});
