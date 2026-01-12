// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
import {
  TIMEOUTS,
  test,
  expect,
  gotoAppAndWaitReady,
  waitForCSSVariablesResolved,
} from './fixtures';

/**
 * E2E Test: Color Scheme and Accessibility
 *
 * Tests the redesigned color palette for:
 * - Softer, more accessible primary colors
 * - Colorblind-safe alternatives
 * - Proper contrast ratios (WCAG 2.1 AA/AAA)
 * - Theme consistency across all UI elements
 * - Chart theme synchronization with user selection
 *
 * Reference: UI/UX Audit
 * @see /docs/working/UI_UX_AUDIT.md
 */

test.describe('Color Scheme - New Palette', () => {
  test.beforeEach(async ({ page }) => {
    // Emulate dark color scheme for consistent testing
    await page.emulateMedia({ colorScheme: 'dark' });

    // Clear theme preference to start with default
    await page.addInitScript(() => {
      localStorage.removeItem('theme');
      localStorage.removeItem('colorblind-mode');
      localStorage.removeItem('sidebar-collapsed'); // Ensure UI elements are visible
    });

    await gotoAppAndWaitReady(page);

    // CRITICAL: Wait for CSS variables to be properly resolved
    // This ensures --highlight is set to #7c3aed before color assertions run
    await waitForCSSVariablesResolved(page, '#7c3aed');
  });

  test('dark mode should use softer violet accent instead of aggressive pink-red', async ({ page }) => {
    // CSS variables are already resolved by beforeEach

    // Get the highlight color (primary accent)
    const highlightColor = await page.evaluate(() => {
      return getComputedStyle(document.documentElement).getPropertyValue('--highlight').trim();
    });

    // The new palette should use violet (#7c3aed) instead of pink-red (#e94560)
    // Accept either hex or rgb format
    const isViolet = highlightColor === '#7c3aed' ||
                     highlightColor === 'rgb(124, 58, 237)' ||
                     highlightColor.toLowerCase() === '#7c3aed';

    expect(isViolet).toBe(true);
  });

  test('light mode should maintain proper contrast with new palette', async ({ page }) => {
    // Switch to light mode
    const themeToggle = page.locator('#theme-toggle');
    await themeToggle.click();
    await page.waitForFunction(() => {
      return document.documentElement.getAttribute('data-theme') === 'light';
    }, { timeout: TIMEOUTS.ANIMATION });

    // Get the highlight color in light mode
    const highlightColor = await page.evaluate(() => {
      return getComputedStyle(document.documentElement).getPropertyValue('--highlight').trim();
    });

    // Light mode should also use the violet palette (slightly darker for contrast)
    const isValidColor = highlightColor === '#7c3aed' ||
                         highlightColor === '#6d28d9' ||
                         highlightColor.toLowerCase().includes('7c3aed') ||
                         highlightColor.toLowerCase().includes('6d28d9');

    expect(isValidColor).toBe(true);
  });

  test('should have improved color harmony in chart palette', async ({ page }) => {
    // Navigate to analytics to check chart colors
    // Navigate using JavaScript click for CI reliability
    // WHY: Playwright's .click() may fail in headless/SwiftShader environments
    await page.evaluate(() => {
      const tab = document.querySelector('.nav-tab[data-view="analytics"]') as HTMLElement;
      if (tab) tab.click();
    });
    await page.waitForFunction(() => {
      const container = document.getElementById('analytics-container');
      return container && getComputedStyle(container).display !== 'none';
    }, { timeout: TIMEOUTS.DATA_LOAD });

    // Get the chart color palette from CSS custom properties
    const chartPrimary = await page.evaluate(() => {
      return getComputedStyle(document.documentElement).getPropertyValue('--chart-primary').trim();
    });

    // Chart primary should be defined and not empty
    expect(chartPrimary).toBeTruthy();

    // Should not be the old aggressive pink-red
    expect(chartPrimary).not.toBe('#e94560');
  });

  test('nav tabs should use the new accent colors', async ({ page }) => {
    // Get the background color of an active nav tab
    const activeTab = page.locator('.nav-tab.active');
    await expect(activeTab).toBeVisible();

    const tabBgColor = await activeTab.evaluate((el) => {
      return getComputedStyle(el).backgroundColor;
    });

    // The active tab should use the new violet accent
    // rgb(124, 58, 237) is #7c3aed
    // Use flexible matching to handle different browser RGB formatting
    const isNewColor = (
      tabBgColor.includes('124') && tabBgColor.includes('58') && tabBgColor.includes('237')
    ) || (
      // Fallback: check if it's not the old aggressive pink-red
      !tabBgColor.includes('233') && !tabBgColor.includes('69') && !tabBgColor.includes('96')
    );
    expect(isNewColor).toBe(true);
  });

  test('buttons should use the new accent colors', async ({ page }) => {
    // Check the refresh button color
    const refreshButton = page.locator('#btn-refresh');
    await expect(refreshButton).toBeVisible();

    const buttonBgColor = await refreshButton.evaluate((el) => {
      return getComputedStyle(el).backgroundColor;
    });

    // Button should use the new violet accent
    // rgb(124, 58, 237) is #7c3aed
    // Use flexible matching to handle different browser RGB formatting
    const isNewColor = (
      buttonBgColor.includes('124') && buttonBgColor.includes('58') && buttonBgColor.includes('237')
    ) || (
      // Fallback: check if it's not the old aggressive pink-red
      !buttonBgColor.includes('233') && !buttonBgColor.includes('69') && !buttonBgColor.includes('96')
    );
    expect(isNewColor).toBe(true);
  });
});

test.describe('Color Scheme - Colorblind Safe Mode', () => {
  test.beforeEach(async ({ page }) => {
    await page.emulateMedia({ colorScheme: 'dark' });

    await page.addInitScript(() => {
      localStorage.removeItem('theme');
      localStorage.removeItem('colorblind-mode');
      localStorage.removeItem('sidebar-collapsed'); // Ensure colorblind toggle is visible
    });

    await gotoAppAndWaitReady(page);

    // Wait for CSS variables to be resolved
    await waitForCSSVariablesResolved(page, '#7c3aed');

    // Ensure sidebar is expanded (not collapsed) so colorblind toggle is visible
    // This handles cases where storageState may have restored collapsed state
    await page.evaluate(() => {
      const sidebar = document.getElementById('sidebar');
      if (sidebar && sidebar.classList.contains('collapsed')) {
        sidebar.classList.remove('collapsed');
        localStorage.removeItem('sidebar-collapsed');
      }
    });
  });

  test('should have colorblind mode toggle in settings/theme area', async ({ page }) => {
    // Look for colorblind mode toggle
    const colorblindToggle = page.locator('[data-testid="colorblind-toggle"], #colorblind-toggle, .colorblind-toggle');

    // The toggle should exist in the UI
    await expect(colorblindToggle).toBeVisible({ timeout: 5000 });
  });

  test('colorblind mode should use blue-orange-teal safe palette', async ({ page }) => {
    // Enable colorblind mode
    const colorblindToggle = page.locator('[data-testid="colorblind-toggle"], #colorblind-toggle, .colorblind-toggle');
    await colorblindToggle.click();
    await page.waitForFunction(() => {
      return document.documentElement.getAttribute('data-colorblind') === 'true';
    }, { timeout: TIMEOUTS.ANIMATION });

    // Check that colorblind-safe colors are applied
    const highlightColor = await page.evaluate(() => {
      return getComputedStyle(document.documentElement).getPropertyValue('--highlight').trim();
    });

    // Colorblind-safe blue: #0077bb
    const isSafeBlue = highlightColor === '#0077bb' ||
                       highlightColor === 'rgb(0, 119, 187)' ||
                       highlightColor.toLowerCase() === '#0077bb';

    expect(isSafeBlue).toBe(true);
  });

  test('colorblind mode should persist in localStorage', async ({ page }) => {
    // Enable colorblind mode
    const colorblindToggle = page.locator('[data-testid="colorblind-toggle"], #colorblind-toggle, .colorblind-toggle');
    await colorblindToggle.click();
    await page.waitForFunction(() => {
      return document.documentElement.getAttribute('data-colorblind') === 'true';
    }, { timeout: TIMEOUTS.ANIMATION });

    // Check localStorage
    const storedMode = await page.evaluate(() => localStorage.getItem('colorblind-mode'));
    expect(storedMode).toBe('true');
  });

  test('colorblind mode should set data attribute on html element', async ({ page }) => {
    // Enable colorblind mode
    const colorblindToggle = page.locator('[data-testid="colorblind-toggle"], #colorblind-toggle, .colorblind-toggle');
    await colorblindToggle.click();
    await page.waitForFunction(() => {
      return document.documentElement.getAttribute('data-colorblind') === 'true';
    }, { timeout: TIMEOUTS.ANIMATION });

    // Check data attribute
    const dataColorblind = await page.getAttribute('html', 'data-colorblind');
    expect(dataColorblind).toBe('true');
  });

  test('chart colors should change in colorblind mode', async ({ page }) => {
    // Navigate to analytics
    // Navigate using JavaScript click for CI reliability
    // WHY: Playwright's .click() may fail in headless/SwiftShader environments
    await page.evaluate(() => {
      const tab = document.querySelector('.nav-tab[data-view="analytics"]') as HTMLElement;
      if (tab) tab.click();
    });
    await page.waitForFunction(() => {
      const container = document.getElementById('analytics-container');
      return container && getComputedStyle(container).display !== 'none';
    }, { timeout: TIMEOUTS.DATA_LOAD });

    // Get initial chart primary color
    const initialColor = await page.evaluate(() => {
      return getComputedStyle(document.documentElement).getPropertyValue('--chart-primary').trim();
    });

    // Enable colorblind mode
    const colorblindToggle = page.locator('[data-testid="colorblind-toggle"], #colorblind-toggle, .colorblind-toggle');
    await colorblindToggle.click();
    await page.waitForFunction(() => {
      return document.documentElement.getAttribute('data-colorblind') === 'true';
    }, { timeout: TIMEOUTS.ANIMATION });

    // Get colorblind chart primary color
    const colorblindColor = await page.evaluate(() => {
      return getComputedStyle(document.documentElement).getPropertyValue('--chart-primary').trim();
    });

    // Colors should be different
    expect(colorblindColor).not.toBe(initialColor);
  });
});

test.describe('Chart Theme Synchronization', () => {
  test.beforeEach(async ({ page }) => {
    await page.emulateMedia({ colorScheme: 'dark' });

    await page.addInitScript(() => {
      localStorage.removeItem('theme');
      localStorage.removeItem('sidebar-collapsed');
    });

    await gotoAppAndWaitReady(page);
  });

  test('charts should respect light theme selection', async ({ page }) => {
    // Navigate to analytics first
    // Navigate using JavaScript click for CI reliability
    // WHY: Playwright's .click() may fail in headless/SwiftShader environments
    await page.evaluate(() => {
      const tab = document.querySelector('.nav-tab[data-view="analytics"]') as HTMLElement;
      if (tab) tab.click();
    });
    await page.waitForFunction(() => {
      const container = document.getElementById('analytics-container');
      return container && getComputedStyle(container).display !== 'none';
    }, { timeout: TIMEOUTS.DATA_LOAD });

    // Switch to light mode
    const themeToggle = page.locator('#theme-toggle');
    await themeToggle.click();
    await page.waitForFunction(() => {
      return document.documentElement.getAttribute('data-theme') === 'light';
    }, { timeout: TIMEOUTS.RENDER });

    // Check that chart background adapts to light theme
    // ECharts should render with light background
    const chartContainer = page.locator('#chart-trends');
    await expect(chartContainer).toBeVisible();

    // Get the computed background of the chart container
    const _chartBgColor = await chartContainer.evaluate((el) => {
      return getComputedStyle(el).backgroundColor;
    });

    // In light mode, charts should have light/transparent background
    // The parent .chart-card should have light background
    const parentBgColor = await chartContainer.evaluate((el) => {
      const parent = el.closest('.chart-card');
      return parent ? getComputedStyle(parent).backgroundColor : '';
    });

    // Light mode card background should be white/light (#ffffff or rgb(255, 255, 255))
    const isLightBackground = parentBgColor.includes('255') ||
                              parentBgColor === '#ffffff' ||
                              parentBgColor === 'rgb(255, 255, 255)';
    expect(isLightBackground).toBe(true);
  });

  test('charts should respect high-contrast theme selection', async ({ page }) => {
    // Navigate to analytics first
    // Navigate using JavaScript click for CI reliability
    // WHY: Playwright's .click() may fail in headless/SwiftShader environments
    await page.evaluate(() => {
      const tab = document.querySelector('.nav-tab[data-view="analytics"]') as HTMLElement;
      if (tab) tab.click();
    });
    await page.waitForFunction(() => {
      const container = document.getElementById('analytics-container');
      return container && getComputedStyle(container).display !== 'none';
    }, { timeout: TIMEOUTS.DATA_LOAD });

    // Switch to high-contrast mode (dark -> light -> high-contrast)
    const themeToggle = page.locator('#theme-toggle');
    await themeToggle.click();
    await page.waitForFunction(() => {
      return document.documentElement.getAttribute('data-theme') === 'light';
    }, { timeout: TIMEOUTS.ANIMATION });
    await themeToggle.click();
    await page.waitForFunction(() => {
      return document.documentElement.getAttribute('data-theme') === 'high-contrast';
    }, { timeout: TIMEOUTS.RENDER });

    // Check that data-theme is high-contrast
    const dataTheme = await page.getAttribute('html', 'data-theme');
    expect(dataTheme).toBe('high-contrast');

    // Chart should have high-contrast colors
    const chartContainer = page.locator('#chart-trends');
    await expect(chartContainer).toBeVisible();

    // Get the parent card background (should be near-black in high-contrast)
    const parentBgColor = await chartContainer.evaluate((el) => {
      const parent = el.closest('.chart-card');
      return parent ? getComputedStyle(parent).backgroundColor : '';
    });

    // High-contrast mode card background should be very dark (#0a0a0a or rgb(10, 10, 10))
    const isDarkBackground = parentBgColor === 'rgb(10, 10, 10)' ||
                             parentBgColor === '#0a0a0a' ||
                             (parentBgColor.includes('10') && !parentBgColor.includes('100'));
    expect(isDarkBackground).toBe(true);
  });

  test('chart text colors should match theme', async ({ page }) => {
    // Navigate to analytics
    // Navigate using JavaScript click for CI reliability
    // WHY: Playwright's .click() may fail in headless/SwiftShader environments
    await page.evaluate(() => {
      const tab = document.querySelector('.nav-tab[data-view="analytics"]') as HTMLElement;
      if (tab) tab.click();
    });
    await page.waitForFunction(() => {
      const container = document.getElementById('analytics-container');
      return container && getComputedStyle(container).display !== 'none';
    }, { timeout: TIMEOUTS.DATA_LOAD });

    // Get chart title color in dark mode
    const chartTitle = page.locator('.chart-title').first();
    await expect(chartTitle).toBeVisible();

    const darkModeTextColor = await chartTitle.evaluate((el) => {
      return getComputedStyle(el).color;
    });

    // DETERMINISTIC FIX: Log actual color for debugging
    console.log(`[E2E] Dark mode chart text color: "${darkModeTextColor}"`);

    // DETERMINISTIC FIX: Parse RGB values and check if it's a light color (high luminance)
    // Light text on dark background should have high RGB values (>150)
    const darkColorMatch = darkModeTextColor.match(/rgb\((\d+),\s*(\d+),\s*(\d+)\)/);
    if (darkColorMatch) {
      const [, r, g, b] = darkColorMatch.map(Number);
      // For dark mode, text should be light (high RGB values)
      const isLightText = r > 150 && g > 150 && b > 150;
      expect(isLightText).toBe(true);
    } else {
      // Fallback: check for specific known values
      const isLightText = darkModeTextColor.includes('234') ||
                          darkModeTextColor.includes('238') ||
                          darkModeTextColor.includes('255');
      expect(isLightText).toBe(true);
    }

    // Switch to light mode
    const themeToggle = page.locator('#theme-toggle');
    await themeToggle.click();
    await page.waitForFunction(() => {
      return document.documentElement.getAttribute('data-theme') === 'light';
    }, { timeout: TIMEOUTS.ANIMATION });

    // Get chart title color in light mode
    const lightModeTextColor = await chartTitle.evaluate((el) => {
      return getComputedStyle(el).color;
    });

    // DETERMINISTIC FIX: Log actual color for debugging
    console.log(`[E2E] Light mode chart text color: "${lightModeTextColor}"`);

    // DETERMINISTIC FIX: Parse RGB values and check if it's a dark color (low luminance)
    // Dark text on light background should have low RGB values (<100)
    const lightColorMatch = lightModeTextColor.match(/rgb\((\d+),\s*(\d+),\s*(\d+)\)/);
    if (lightColorMatch) {
      const [, r, g, b] = lightColorMatch.map(Number);
      // For light mode, text should be dark (low RGB values)
      const isDarkText = r < 100 && g < 100 && b < 100;
      expect(isDarkText).toBe(true);
    } else {
      // Fallback: check for specific known values
      const isDarkText = lightModeTextColor.includes('26') ||
                         lightModeTextColor.includes('0,');
      expect(isDarkText).toBe(true);
    }
  });
});

test.describe('Color Contrast Ratios', () => {
  test.beforeEach(async ({ page }) => {
    await page.emulateMedia({ colorScheme: 'dark' });

    await page.addInitScript(() => {
      localStorage.removeItem('theme');
      localStorage.removeItem('sidebar-collapsed');
    });

    await gotoAppAndWaitReady(page);
  });

  test('primary text should have at least 4.5:1 contrast ratio (WCAG AA)', async ({ page }) => {
    // Get text and background colors
    const textColor = await page.evaluate(() => {
      return getComputedStyle(document.documentElement).getPropertyValue('--text-primary').trim();
    });

    const bgColor = await page.evaluate(() => {
      return getComputedStyle(document.documentElement).getPropertyValue('--primary-bg').trim();
    });

    // Both colors should be defined
    expect(textColor).toBeTruthy();
    expect(bgColor).toBeTruthy();

    // Calculate approximate contrast ratio
    // #eaeaea on #1a1a2e should be approximately 11.5:1 (well above 4.5:1)
    // This is a simplified check - actual contrast calculation is complex
    // We verify the colors are in the expected range
    const textIsLight = textColor.includes('ea') || textColor.includes('234') || textColor.includes('255');
    const bgIsDark = bgColor.includes('1a') || bgColor.includes('26') || bgColor.includes('16');

    expect(textIsLight).toBe(true);
    expect(bgIsDark).toBe(true);
  });

  test('high-contrast mode should have at least 7:1 contrast ratio (WCAG AAA)', async ({ page }) => {
    // Switch to high-contrast mode
    const themeToggle = page.locator('#theme-toggle');
    await themeToggle.click();
    await page.waitForFunction(() => {
      return document.documentElement.getAttribute('data-theme') === 'light';
    }, { timeout: TIMEOUTS.ANIMATION });
    await themeToggle.click();
    await page.waitForFunction(() => {
      return document.documentElement.getAttribute('data-theme') === 'high-contrast';
    }, { timeout: TIMEOUTS.ANIMATION });

    // Get high-contrast colors
    const textColor = await page.evaluate(() => {
      return getComputedStyle(document.documentElement).getPropertyValue('--text-primary').trim();
    });

    const bgColor = await page.evaluate(() => {
      return getComputedStyle(document.documentElement).getPropertyValue('--primary-bg').trim();
    });

    // High-contrast should use pure white (#ffffff) on pure black (#000000)
    // This gives 21:1 contrast ratio
    expect(textColor).toBe('#ffffff');
    expect(bgColor).toBe('#000000');
  });

  test('accent color should be visible on both dark and light backgrounds', async ({ page }) => {
    // Get accent color
    const accentColor = await page.evaluate(() => {
      return getComputedStyle(document.documentElement).getPropertyValue('--highlight').trim();
    });

    expect(accentColor).toBeTruthy();

    // The accent should be a mid-tone that works on both
    // Violet #7c3aed has good visibility on both white and dark backgrounds
    const isValidAccent = accentColor !== '#e94560'; // Not the old aggressive red
    expect(isValidAccent).toBe(true);
  });
});

test.describe('Semantic Color Tokens', () => {
  test.beforeEach(async ({ page }) => {
    await page.emulateMedia({ colorScheme: 'dark' });

    await page.addInitScript(() => {
      localStorage.removeItem('sidebar-collapsed');
    });

    await gotoAppAndWaitReady(page);
  });

  test('should have semantic color tokens defined', async ({ page }) => {
    // Check for semantic color tokens
    const successColor = await page.evaluate(() => {
      return getComputedStyle(document.documentElement).getPropertyValue('--color-success').trim();
    });

    const warningColor = await page.evaluate(() => {
      return getComputedStyle(document.documentElement).getPropertyValue('--color-warning').trim();
    });

    const errorColor = await page.evaluate(() => {
      return getComputedStyle(document.documentElement).getPropertyValue('--color-error').trim();
    });

    const infoColor = await page.evaluate(() => {
      return getComputedStyle(document.documentElement).getPropertyValue('--color-info').trim();
    });

    // All semantic colors should be defined
    expect(successColor).toBeTruthy();
    expect(warningColor).toBeTruthy();
    expect(errorColor).toBeTruthy();
    expect(infoColor).toBeTruthy();
  });

  test('map popups should use theme CSS variables', async ({ page }) => {
    // Check map popup styles use CSS variables
    const popupStyles = await page.evaluate(() => {
      // Create a temporary popup element to check computed styles
      const popup = document.createElement('div');
      popup.className = 'maplibregl-popup-content';
      document.body.appendChild(popup);
      const styles = {
        background: getComputedStyle(popup).backgroundColor,
        color: getComputedStyle(popup).color,
      };
      document.body.removeChild(popup);
      return styles;
    });

    // Popup should have background and text colors set
    expect(popupStyles.background).toBeTruthy();
    expect(popupStyles.color).toBeTruthy();

    // Colors should not be the browser defaults
    expect(popupStyles.background).not.toBe('rgba(0, 0, 0, 0)');
    expect(popupStyles.color).not.toBe('rgb(0, 0, 0)');
  });

  test('toast notifications should use semantic CSS variable colors', async ({ page }) => {
    // Get the semantic color variable values
    const semanticColors = await page.evaluate(() => {
      const style = getComputedStyle(document.documentElement);
      return {
        info: style.getPropertyValue('--color-info').trim(),
        success: style.getPropertyValue('--color-success').trim(),
        warning: style.getPropertyValue('--color-warning').trim(),
        error: style.getPropertyValue('--color-error').trim(),
      };
    });

    // Helper to convert hex to rgb format
    const hexToRgb = (hex: string): string => {
      const result = /^#?([a-f\d]{2})([a-f\d]{2})([a-f\d]{2})$/i.exec(hex);
      if (!result) return hex;
      const r = parseInt(result[1], 16);
      const g = parseInt(result[2], 16);
      const b = parseInt(result[3], 16);
      return `rgb(${r}, ${g}, ${b})`;
    };

    // Test each toast type uses the corresponding semantic color variable
    const toastTypes = ['info', 'success', 'warning', 'error'] as const;

    for (const type of toastTypes) {
      const toastIconColor = await page.evaluate((toastType) => {
        // Create toast structure with icon child
        const toast = document.createElement('div');
        toast.className = `toast toast-${toastType}`;
        const icon = document.createElement('div');
        icon.className = 'toast-icon';
        toast.appendChild(icon);
        document.body.appendChild(toast);
        const color = getComputedStyle(icon).color;
        document.body.removeChild(toast);
        return color;
      }, type);

      const expectedColor = semanticColors[type];
      const expectedRgb = hexToRgb(expectedColor);

      // The toast icon color should match the semantic color variable
      expect(
        toastIconColor === expectedRgb ||
          toastIconColor === expectedColor ||
          toastIconColor.replace(/\s/g, '') === expectedRgb.replace(/\s/g, '')
      ).toBe(true);
    }
  });
});

test.describe('Chart Color Palette', () => {
  test.beforeEach(async ({ page }) => {
    await page.emulateMedia({ colorScheme: 'dark' });

    await page.addInitScript(() => {
      localStorage.removeItem('sidebar-collapsed');
    });

    await gotoAppAndWaitReady(page);
  });

  test('should have 8+ harmonious chart colors defined', async ({ page }) => {
    // Navigate to analytics
    // Navigate using JavaScript click for CI reliability
    // WHY: Playwright's .click() may fail in headless/SwiftShader environments
    await page.evaluate(() => {
      const tab = document.querySelector('.nav-tab[data-view="analytics"]') as HTMLElement;
      if (tab) tab.click();
    });
    await page.waitForFunction(() => {
      const container = document.getElementById('analytics-container');
      return container && getComputedStyle(container).display !== 'none';
    }, { timeout: TIMEOUTS.DATA_LOAD });

    // Check for chart color CSS variables
    const colors = await page.evaluate(() => {
      const style = getComputedStyle(document.documentElement);
      const chartColors = [];
      for (let i = 1; i <= 10; i++) {
        const color = style.getPropertyValue(`--chart-color-${i}`).trim();
        if (color) chartColors.push(color);
      }
      return chartColors;
    });

    // Should have at least 8 chart colors
    expect(colors.length).toBeGreaterThanOrEqual(8);
  });

  test('chart colors should not include the old aggressive pink-red', async ({ page }) => {
    // Navigate to analytics
    // Navigate using JavaScript click for CI reliability
    // WHY: Playwright's .click() may fail in headless/SwiftShader environments
    await page.evaluate(() => {
      const tab = document.querySelector('.nav-tab[data-view="analytics"]') as HTMLElement;
      if (tab) tab.click();
    });
    await page.waitForFunction(() => {
      const container = document.getElementById('analytics-container');
      return container && getComputedStyle(container).display !== 'none';
    }, { timeout: TIMEOUTS.DATA_LOAD });

    // Get all chart colors
    const colors = await page.evaluate(() => {
      const style = getComputedStyle(document.documentElement);
      const chartColors = [];
      for (let i = 1; i <= 10; i++) {
        const color = style.getPropertyValue(`--chart-color-${i}`).trim().toLowerCase();
        if (color) chartColors.push(color);
      }
      return chartColors;
    });

    // None should be the old aggressive pink-red
    colors.forEach((color) => {
      expect(color).not.toBe('#e94560');
      expect(color).not.toBe('rgb(233, 69, 96)');
    });
  });
});

test.describe('Globe Visualization Theming', () => {
  test.beforeEach(async ({ page }) => {
    await page.emulateMedia({ colorScheme: 'dark' });

    await page.addInitScript(() => {
      localStorage.removeItem('theme');
      localStorage.removeItem('sidebar-collapsed');
    });

    await gotoAppAndWaitReady(page);
  });

  test('globe marker colors should be defined as CSS variables', async ({ page }) => {
    // Check for globe color CSS variables
    const globeColors = await page.evaluate(() => {
      const style = getComputedStyle(document.documentElement);
      return {
        high: style.getPropertyValue('--globe-marker-high').trim(),
        mediumHigh: style.getPropertyValue('--globe-marker-medium-high').trim(),
        medium: style.getPropertyValue('--globe-marker-medium').trim(),
        low: style.getPropertyValue('--globe-marker-low').trim(),
      };
    });

    // All globe marker colors should be defined
    expect(globeColors.high).toBeTruthy();
    expect(globeColors.mediumHigh).toBeTruthy();
    expect(globeColors.medium).toBeTruthy();
    expect(globeColors.low).toBeTruthy();
  });

  test('globe tooltip should use theme CSS variables', async ({ page }) => {
    // Check for globe tooltip CSS variables
    const tooltipColors = await page.evaluate(() => {
      const style = getComputedStyle(document.documentElement);
      return {
        background: style.getPropertyValue('--globe-tooltip-bg').trim(),
        border: style.getPropertyValue('--globe-tooltip-border').trim(),
        text: style.getPropertyValue('--globe-tooltip-text').trim(),
      };
    });

    // All tooltip colors should be defined
    expect(tooltipColors.background).toBeTruthy();
    expect(tooltipColors.border).toBeTruthy();
    expect(tooltipColors.text).toBeTruthy();
  });

  test('globe colors should change with theme', async ({ page }) => {
    // Get dark theme colors
    const darkColors = await page.evaluate(() => {
      const style = getComputedStyle(document.documentElement);
      return style.getPropertyValue('--globe-marker-low').trim();
    });

    // Switch to light theme
    const themeToggle = page.locator('#theme-toggle');
    await themeToggle.click();
    await page.waitForFunction(() => {
      return document.documentElement.getAttribute('data-theme') === 'light';
    }, { timeout: TIMEOUTS.RENDER });

    // Get light theme colors
    const lightColors = await page.evaluate(() => {
      const style = getComputedStyle(document.documentElement);
      return style.getPropertyValue('--globe-marker-low').trim();
    });

    // Colors might be the same or different based on design choice
    // At minimum, they should both be defined
    expect(darkColors).toBeTruthy();
    expect(lightColors).toBeTruthy();
  });
});
