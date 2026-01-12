// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Documentation Screenshot Capture Script
 *
 * This script captures all screenshots needed for the README and documentation.
 * It is designed to be:
 *
 * - Deterministic: Same inputs produce same outputs
 * - Idempotent: Safe to run multiple times
 * - Observable: Detailed logging for debugging
 * - Durable: Continues on individual failures, reports all at end
 * - Testable: Each screenshot is a separate test case
 *
 * Screenshots are organized into categories:
 * 1. Authentication (login page)
 * 2. Maps (2D map view)
 * 3. Globe (3D visualization)
 * 4. Analytics (dashboard overview)
 * 5. Live Activity (real-time monitoring)
 * 6. Additional views (server, data governance, etc.)
 *
 * @see playwright.screenshots.config.ts for configuration
 */

import { test, expect, Page, BrowserContext } from '@playwright/test';
import * as fs from 'fs';
import * as path from 'path';

// =============================================================================
// Configuration
// =============================================================================

/**
 * Screenshot configuration interface
 */
interface ScreenshotConfig {
  /** Unique identifier for the screenshot */
  id: string;
  /** Human-readable name for the screenshot */
  name: string;
  /** Description for documentation */
  description: string;
  /** URL path to navigate to (relative to baseURL) */
  path: string;
  /** CSS selector to wait for before capturing */
  waitForSelector: string;
  /** Additional selectors that must be visible */
  additionalSelectors?: string[];
  /** Whether this screenshot requires authentication */
  requiresAuth: boolean;
  /** Navigation tab to click (data-view attribute value) */
  navTab?: string;
  /** Sub-tab to click (for analytics pages) */
  subTab?: string;
  /** Custom wait time after navigation (ms) */
  waitAfterNavigation?: number;
  /** Custom viewport for this screenshot */
  viewport?: { width: number; height: number };
  /** Clip region for partial screenshot */
  clip?: { x: number; y: number; width: number; height: number };
  /** Category for organization */
  category: ScreenshotCategory;
}

type ScreenshotCategory =
  | 'authentication'
  | 'maps'
  | 'globe'
  | 'analytics'
  | 'activity'
  | 'server'
  | 'features'
  | 'governance'
  | 'newsletter';

/**
 * All screenshots to capture for documentation
 *
 * These are ordered from lightweight (no WebGL) to heavy (WebGL)
 * to minimize resource exhaustion issues in CI.
 *
 * Naming Convention:
 * - Main views: {view-name}.png (e.g., map-view.png)
 * - Sub-pages: {view}-{subpage}.png (e.g., analytics-content.png)
 * - Modes: {view}-{mode}.png (e.g., map-heatmap.png)
 * - With theme suffix when capturing multiple: {name}-{theme}.png
 */
const SCREENSHOTS: ScreenshotConfig[] = [
  // ==========================================================================
  // Category 1: Authentication (no WebGL, lightweight)
  // ==========================================================================
  {
    id: 'login-page',
    name: 'Login Page',
    description: 'The authentication login page with username/password and Plex OAuth options',
    path: '/',
    waitForSelector: '#login-container',
    additionalSelectors: ['#login-box', '#login-form', '#btn-plex-oauth'],
    requiresAuth: false,
    category: 'authentication',
  },

  // ==========================================================================
  // Category 2: Live Activity (minimal WebGL)
  // ==========================================================================
  {
    id: 'live-activity',
    name: 'Live Activity Dashboard',
    description: 'Real-time monitoring of active playback sessions with stream details',
    path: '/',
    waitForSelector: '#activity-container',
    additionalSelectors: ['#activity-sessions-list'],
    requiresAuth: true,
    navTab: 'activity',
    waitAfterNavigation: 2000,
    category: 'activity',
  },

  // ==========================================================================
  // Category 3: Server & Features (minimal WebGL)
  // ==========================================================================
  {
    id: 'server-dashboard',
    name: 'Server Dashboard',
    description: 'Server statistics, connection status, and health monitoring',
    path: '/',
    waitForSelector: '#server-container',
    requiresAuth: true,
    navTab: 'server',
    waitAfterNavigation: 2000,
    category: 'server',
  },
  {
    id: 'recently-added',
    name: 'Recently Added',
    description: 'Recently added content discovery view with poster grid',
    path: '/',
    waitForSelector: '#recently-added-container',
    requiresAuth: true,
    navTab: 'recently-added',
    waitAfterNavigation: 2000,
    category: 'features',
  },
  {
    id: 'cross-platform',
    name: 'Cross-Platform View',
    description: 'Cross-platform comparison and analytics across media servers',
    path: '/',
    waitForSelector: '#cross-platform-container',
    requiresAuth: true,
    navTab: 'cross-platform',
    waitAfterNavigation: 2000,
    category: 'features',
  },

  // ==========================================================================
  // Category 4: Newsletter (minimal WebGL)
  // ==========================================================================
  {
    id: 'newsletter-overview',
    name: 'Newsletter Overview',
    description: 'Newsletter management dashboard',
    path: '/',
    waitForSelector: '#newsletter-container',
    requiresAuth: true,
    navTab: 'newsletter',
    waitAfterNavigation: 2000,
    category: 'newsletter',
  },

  // ==========================================================================
  // Category 5: Data Governance (minimal WebGL)
  // ==========================================================================
  {
    id: 'governance-overview',
    name: 'Data Governance Overview',
    description: 'Data governance dashboard with compliance metrics',
    path: '/',
    waitForSelector: '#data-governance-container',
    requiresAuth: true,
    navTab: 'data-governance',
    waitAfterNavigation: 2000,
    category: 'governance',
  },
  {
    id: 'governance-detection',
    name: 'Security Detection',
    description: 'Security detection rules and anomaly monitoring',
    path: '/',
    waitForSelector: '#data-governance-container',
    requiresAuth: true,
    navTab: 'data-governance',
    subTab: 'detection',
    waitAfterNavigation: 2000,
    category: 'governance',
  },
  {
    id: 'governance-sync',
    name: 'Sync Status',
    description: 'Data synchronization status across all sources',
    path: '/',
    waitForSelector: '#data-governance-container',
    requiresAuth: true,
    navTab: 'data-governance',
    subTab: 'sync',
    waitAfterNavigation: 2000,
    category: 'governance',
  },
  {
    id: 'governance-audit',
    name: 'Audit Log',
    description: 'Complete audit trail of system events',
    path: '/',
    waitForSelector: '#data-governance-container',
    requiresAuth: true,
    navTab: 'data-governance',
    subTab: 'audit',
    waitAfterNavigation: 2000,
    category: 'governance',
  },

  // ==========================================================================
  // Category 6: Analytics - All 10 Sub-pages (ECharts - medium WebGL)
  // ==========================================================================
  {
    id: 'analytics-overview',
    name: 'Analytics Overview',
    description: 'Main analytics dashboard with key metrics and summary charts',
    path: '/',
    waitForSelector: '#analytics-overview',
    additionalSelectors: ['.analytics-charts-container'],
    requiresAuth: true,
    navTab: 'analytics',
    subTab: 'overview',
    waitAfterNavigation: 5000,
    category: 'analytics',
  },
  {
    id: 'analytics-content',
    name: 'Content Analytics',
    description: 'Content consumption analytics with genre and media type breakdowns',
    path: '/',
    waitForSelector: '#analytics-content',
    requiresAuth: true,
    navTab: 'analytics',
    subTab: 'content',
    waitAfterNavigation: 5000,
    category: 'analytics',
  },
  {
    id: 'analytics-users',
    name: 'User Analytics',
    description: 'User engagement metrics and viewing patterns',
    path: '/',
    waitForSelector: '#analytics-users',
    requiresAuth: true,
    navTab: 'analytics',
    subTab: 'users',
    waitAfterNavigation: 5000,
    category: 'analytics',
  },
  {
    id: 'analytics-performance',
    name: 'Performance Analytics',
    description: 'Stream quality, bandwidth, and transcoding metrics',
    path: '/',
    waitForSelector: '#analytics-performance',
    requiresAuth: true,
    navTab: 'analytics',
    subTab: 'performance',
    waitAfterNavigation: 5000,
    category: 'analytics',
  },
  {
    id: 'analytics-geographic',
    name: 'Geographic Analytics',
    description: 'Geographic distribution of playback activity',
    path: '/',
    waitForSelector: '#analytics-geographic',
    requiresAuth: true,
    navTab: 'analytics',
    subTab: 'geographic',
    waitAfterNavigation: 5000,
    category: 'analytics',
  },
  {
    id: 'analytics-advanced',
    name: 'Advanced Analytics',
    description: 'Advanced analytics with custom queries and visualizations',
    path: '/',
    waitForSelector: '#analytics-advanced',
    requiresAuth: true,
    navTab: 'analytics',
    subTab: 'advanced',
    waitAfterNavigation: 5000,
    category: 'analytics',
  },
  {
    id: 'analytics-library',
    name: 'Library Analytics',
    description: 'Library composition and content distribution analytics',
    path: '/',
    waitForSelector: '#analytics-library',
    requiresAuth: true,
    navTab: 'analytics',
    subTab: 'library',
    waitAfterNavigation: 5000,
    category: 'analytics',
  },
  {
    id: 'analytics-wrapped',
    name: 'Annual Wrapped',
    description: 'Spotify Wrapped-style annual viewing summary',
    path: '/',
    waitForSelector: '#analytics-wrapped',
    requiresAuth: true,
    navTab: 'analytics',
    subTab: 'wrapped',
    waitAfterNavigation: 5000,
    category: 'analytics',
  },

  // ==========================================================================
  // Category 7: Maps - All Visualization Modes (MapLibre GL - heavy WebGL)
  // ==========================================================================
  {
    id: 'map-view',
    name: '2D Map View',
    description: 'Interactive 2D map with clustered playback location markers',
    path: '/',
    waitForSelector: '#map-container',
    additionalSelectors: ['#map', '.maplibregl-canvas'],
    requiresAuth: true,
    navTab: 'maps',
    waitAfterNavigation: 5000,
    category: 'maps',
  },
  {
    id: 'map-heatmap',
    name: 'Heatmap View',
    description: 'Heatmap density visualization of playback locations',
    path: '/',
    waitForSelector: '#map-container',
    additionalSelectors: ['.maplibregl-canvas'],
    requiresAuth: true,
    navTab: 'maps',
    subTab: 'heatmap',
    waitAfterNavigation: 5000,
    category: 'maps',
  },
  {
    id: 'map-hexagons',
    name: 'H3 Hexagon View',
    description: 'H3 hexagonal grid aggregation of playback activity',
    path: '/',
    waitForSelector: '#map-container',
    additionalSelectors: ['.maplibregl-canvas'],
    requiresAuth: true,
    navTab: 'maps',
    subTab: 'hexagons',
    waitAfterNavigation: 5000,
    category: 'maps',
  },

  // ==========================================================================
  // Category 8: Globe (deck.gl - heaviest WebGL)
  // ==========================================================================
  {
    id: 'globe-view',
    name: '3D Globe View',
    description: 'Interactive 3D globe with arc connections to server location',
    path: '/',
    waitForSelector: '#globe-container',
    additionalSelectors: ['#deck-canvas'],
    requiresAuth: true,
    navTab: 'maps',
    subTab: '3d',
    waitAfterNavigation: 7000,
    category: 'globe',
  },
];

// =============================================================================
// Constants
// =============================================================================

/** Output directory for screenshots */
const OUTPUT_DIR = path.join(__dirname, '..', 'screenshots', 'captured');

/** Test credentials (matches docker-compose.e2e.yml) */
const TEST_CREDENTIALS = {
  username: process.env.TEST_USERNAME || 'admin',
  password: process.env.TEST_PASSWORD || 'E2eTestPass123!',
};

/**
 * Theme mode for screenshots
 * - 'dark': Only dark theme screenshots
 * - 'light': Only light theme screenshots
 * - 'high-contrast': Only high-contrast theme screenshots
 * - 'dark-and-light': Capture each screenshot in dark and light themes
 * - 'all-themes': Capture each screenshot in all three themes
 */
type ThemeMode = 'dark' | 'light' | 'high-contrast' | 'dark-and-light' | 'all-themes';
const THEME_MODE: ThemeMode = (process.env.SCREENSHOT_THEME as ThemeMode) || 'dark';

/**
 * Available themes in the application
 */
type AppTheme = 'dark' | 'light' | 'high-contrast';

/** Timeouts */
const TIMEOUTS = {
  elementVisible: 30_000,
  navigation: 60_000,
  animation: 3_000,
  webglCleanup: 2_000,
  themeSwitch: 1_000,
};

// =============================================================================
// Test Fixtures and Setup
// =============================================================================

// Track screenshot results for summary
const screenshotResults: Array<{
  id: string;
  name: string;
  theme: AppTheme;
  success: boolean;
  error?: string;
  path?: string;
  duration: number;
}> = [];

/**
 * Ensure output directory exists
 */
function ensureOutputDir(): void {
  if (!fs.existsSync(OUTPUT_DIR)) {
    fs.mkdirSync(OUTPUT_DIR, { recursive: true });
    console.log(`Created output directory: ${OUTPUT_DIR}`);
  }
}

/**
 * Clean up WebGL resources to prevent memory exhaustion
 */
async function cleanupWebGL(page: Page): Promise<void> {
  try {
    await page.evaluate(() => {
      const canvases = document.querySelectorAll('canvas');
      canvases.forEach((canvas) => {
        const contexts = ['webgl', 'webgl2', 'experimental-webgl'];
        for (const contextType of contexts) {
          try {
            const gl = canvas.getContext(contextType as 'webgl') as WebGLRenderingContext | null;
            if (gl) {
              const ext = gl.getExtension('WEBGL_lose_context');
              if (ext) {
                ext.loseContext();
              }
            }
          } catch {
            // Context not available, continue
          }
        }
      });
    });
    // Allow time for cleanup
    await page.waitForTimeout(TIMEOUTS.webglCleanup);
  } catch (error) {
    console.warn('WebGL cleanup warning:', error);
  }
}

/**
 * Disable CSS animations for deterministic screenshots.
 * NOTE: This is now handled by setupDeterministicPage() via addInitScript
 * which runs earlier. This function is kept for backward compatibility.
 */
async function disableAnimations(page: Page): Promise<void> {
  // Check if already injected by setupDeterministicPage
  const alreadyInjected = await page.evaluate(() => {
    return document.getElementById('screenshot-no-animations') !== null;
  }).catch(() => false);

  if (alreadyInjected) {
    return;
  }

  await page.addStyleTag({
    content: `
      *, *::before, *::after {
        animation-duration: 0s !important;
        animation-delay: 0s !important;
        transition-duration: 0s !important;
        transition-delay: 0s !important;
        scroll-behavior: auto !important;
      }
    `,
  });
}

/**
 * Wait for all fonts to be loaded for consistent text rendering.
 * This is critical for deterministic screenshots across different runs.
 */
async function waitForFonts(page: Page): Promise<void> {
  try {
    await page.evaluate(async () => {
      // Wait for document.fonts.ready
      await document.fonts.ready;

      // Additional check: wait for all font faces to be loaded
      const fontFaces = Array.from(document.fonts);
      await Promise.all(
        fontFaces
          .filter((font) => font.status === 'loading')
          .map((font) => font.load().catch(() => {}))
      );
    });
    console.log('Fonts loaded');
  } catch (error) {
    console.warn('Font loading check failed, continuing:', error);
  }
}

/**
 * Wait for WebGL canvas to be ready and rendered.
 * This ensures map tiles and 3D content are fully rendered.
 */
async function waitForWebGLReady(page: Page, selector: string): Promise<void> {
  try {
    // Wait for canvas to exist
    await page.waitForSelector(selector, { state: 'visible', timeout: 10_000 });

    // Wait for WebGL context to be ready and have rendered at least one frame
    await page.evaluate(async (sel) => {
      const canvas = document.querySelector(sel) as HTMLCanvasElement;
      if (!canvas) return;

      // Check if WebGL context exists
      const gl = canvas.getContext('webgl') || canvas.getContext('webgl2');
      if (!gl) return;

      // Wait for next animation frame to ensure render
      await new Promise((resolve) => requestAnimationFrame(resolve));
      await new Promise((resolve) => requestAnimationFrame(resolve));
    }, selector);

    console.log(`WebGL ready: ${selector}`);
  } catch (error) {
    console.warn(`WebGL wait for ${selector} failed:`, error);
  }
}

/**
 * Switch the page's color scheme/theme.
 * This sets both the browser's color scheme AND the app's theme via localStorage/data-theme.
 * High-contrast theme requires special handling since emulateMedia doesn't support it.
 */
async function setColorScheme(page: Page, theme: AppTheme): Promise<void> {
  // Set browser color scheme (for CSS media queries)
  // High-contrast uses dark as base
  const browserScheme = theme === 'light' ? 'light' : 'dark';
  await page.emulateMedia({ colorScheme: browserScheme });

  // Set app theme via localStorage and data-theme attribute
  // This matches how ThemeManager.ts applies themes
  await page.evaluate((t) => {
    // Set localStorage so theme persists across navigation
    localStorage.setItem('theme', t);

    // Apply data-theme attribute directly (matches ThemeManager.applyTheme)
    if (t === 'light') {
      document.documentElement.setAttribute('data-theme', 'light');
    } else if (t === 'high-contrast') {
      document.documentElement.setAttribute('data-theme', 'high-contrast');
    } else {
      document.documentElement.removeAttribute('data-theme');
    }
  }, theme);

  // Wait for theme CSS variables to update
  await page.waitForTimeout(TIMEOUTS.themeSwitch);
}

/**
 * Setup page for deterministic screenshots BEFORE any navigation.
 * This must be called once per page/context to ensure:
 * 1. All onboarding/tour flags are set before page loads
 * 2. Animations are disabled from the very first render
 * 3. Fonts are preloaded
 */
async function setupDeterministicPage(page: Page): Promise<void> {
  // CRITICAL: Set localStorage flags BEFORE any navigation via addInitScript
  // This runs before page JavaScript, ensuring tours/wizards never appear
  await page.addInitScript(() => {
    // Onboarding flags (OnboardingManager.ts)
    localStorage.setItem('onboarding_completed', 'true');
    localStorage.setItem('onboarding_skipped', 'true');

    // Setup wizard flags (SetupWizardManager.ts)
    localStorage.setItem('setup_wizard_completed', 'true');
    localStorage.setItem('setup_wizard_skipped', 'true');

    // Tour flags
    localStorage.setItem('tour_completed', 'true');
    localStorage.setItem('help_dismissed', 'true');

    // Progressive onboarding
    localStorage.setItem('progressive_onboarding_completed', 'true');

    // Inject animation-disabling CSS immediately
    const style = document.createElement('style');
    style.id = 'screenshot-no-animations';
    style.textContent = `
      *, *::before, *::after {
        animation-duration: 0s !important;
        animation-delay: 0s !important;
        transition-duration: 0s !important;
        transition-delay: 0s !important;
        scroll-behavior: auto !important;
      }
      /* Disable MapLibre GL JS animations */
      .maplibregl-map {
        transition: none !important;
      }
      /* Disable ECharts animations */
      .echarts-tooltip {
        transition: none !important;
      }
    `;
    document.head.appendChild(style);
  });
}

/**
 * Dismiss any modals or overlays that might interfere with screenshots.
 * This is a fallback for any modals that appear despite localStorage flags.
 */
async function dismissOverlays(page: Page): Promise<void> {
  // Close any visible modals (fallback)
  const modalCloseButtons = [
    '[data-dismiss="modal"]',
    '.modal-close',
    '.dialog-close',
    '[aria-label="Close"]',
    '.setup-wizard-close',
    '.onboarding-modal .btn-secondary', // Skip button
    '#onboarding-skip-btn',
    '#wizard-skip-btn',
  ];

  for (const selector of modalCloseButtons) {
    const button = page.locator(selector).first();
    if (await button.isVisible({ timeout: 500 }).catch(() => false)) {
      console.log(`Dismissing overlay: ${selector}`);
      await button.click().catch(() => {});
      await page.waitForTimeout(200);
    }
  }

  // Also hide any visible overlay elements directly
  await page.evaluate(() => {
    const overlays = document.querySelectorAll(
      '#onboarding-modal, #setup-wizard-modal, .onboarding-tooltip, .setup-wizard-overlay'
    );
    overlays.forEach((el) => {
      (el as HTMLElement).style.display = 'none';
    });
  }).catch(() => {});
}

/**
 * Log in to the application.
 * Handles both authenticated and no-auth modes:
 * - AUTH_MODE=basic: Login form visible, credentials required
 * - AUTH_MODE=none: App may be directly accessible, login disabled
 */
async function login(page: Page): Promise<boolean> {
  console.log('Attempting login...');

  try {
    // Navigate to root and wait for initial page load
    await page.goto('/', { waitUntil: 'domcontentloaded' });

    // Give the page a moment to render and determine auth state
    await page.waitForTimeout(1000);

    // Check if app is already visible (AUTH_MODE=none or already authenticated)
    const appVisible = await page.locator('#app:not(.hidden)').isVisible({ timeout: 3000 }).catch(() => false);
    if (appVisible) {
      console.log('App already visible (no auth required or already authenticated)');
      // Wait for nav tabs to ensure app is fully loaded
      await page.waitForSelector('#nav-tabs', { timeout: TIMEOUTS.elementVisible }).catch(() => {
        console.log('Nav tabs not found, app may still be loading');
      });
      return true;
    }

    // Check if login form exists
    const loginFormVisible = await page.locator('#login-form').isVisible({ timeout: 3000 }).catch(() => false);
    if (!loginFormVisible) {
      // No login form and no app visible - something is wrong
      console.log('Neither app nor login form visible, waiting longer...');
      // Try waiting for either
      const result = await Promise.race([
        page.waitForSelector('#app:not(.hidden)', { timeout: 10000 }).then(() => 'app'),
        page.waitForSelector('#login-form', { timeout: 10000 }).then(() => 'login'),
      ]).catch(() => 'timeout');

      if (result === 'app') {
        console.log('App became visible after waiting');
        return true;
      } else if (result === 'timeout') {
        console.error('Timeout waiting for app or login form');
        return false;
      }
    }

    // Login form is visible - proceed with authentication
    console.log('Login form visible, entering credentials...');

    // Wait for form elements to be ready
    await page.waitForSelector('#username', { timeout: TIMEOUTS.elementVisible });
    await page.waitForSelector('#password', { timeout: TIMEOUTS.elementVisible });

    // Fill credentials
    await page.fill('#username', TEST_CREDENTIALS.username);
    await page.fill('#password', TEST_CREDENTIALS.password);

    // Submit form
    await page.click('#btn-login');

    // Wait for app to load after login, with longer timeout
    try {
      await page.waitForSelector('#app:not(.hidden)', { timeout: TIMEOUTS.navigation });
      await page.waitForSelector('#nav-tabs', { timeout: TIMEOUTS.elementVisible });
      console.log('Login successful');
      return true;
    } catch (loginError) {
      // Check if there's an error message displayed
      const errorMessage = await page.locator('#login-error').textContent().catch(() => null);
      if (errorMessage) {
        console.error('Login failed with error:', errorMessage);
      }

      // Check if app became visible despite error (edge case)
      const appVisibleAfterError = await page.locator('#app:not(.hidden)').isVisible({ timeout: 2000 }).catch(() => false);
      if (appVisibleAfterError) {
        console.log('App visible despite login error');
        return true;
      }

      throw loginError;
    }
  } catch (error) {
    console.error('Login failed:', error);
    return false;
  }
}

/**
 * Get the list of themes to capture based on THEME_MODE
 */
function getThemesToCapture(): AppTheme[] {
  switch (THEME_MODE) {
    case 'light':
      return ['light'];
    case 'high-contrast':
      return ['high-contrast'];
    case 'dark-and-light':
      return ['dark', 'light'];
    case 'all-themes':
      return ['dark', 'light', 'high-contrast'];
    case 'dark':
    default:
      return ['dark'];
  }
}

/**
 * Navigate to a specific tab
 */
async function navigateToTab(page: Page, tabName: string): Promise<void> {
  const tabSelector = `[data-view="${tabName}"]`;
  console.log(`Navigating to tab: ${tabName}`);

  // Click the tab
  await page.click(tabSelector);

  // Wait for tab to become active
  await page.waitForSelector(`${tabSelector}.active`, { timeout: TIMEOUTS.elementVisible });
}

/**
 * Navigate to a sub-tab or mode within a view.
 * Handles different navigation patterns:
 * - Analytics: data-analytics-page attribute
 * - Governance: data-governance-page attribute
 * - Newsletter: data-newsletter-page attribute
 * - Maps: data-viz-mode (heatmap, hexagons) or data-view-mode (2d, 3d)
 */
async function navigateToSubTab(page: Page, config: ScreenshotConfig): Promise<void> {
  if (!config.subTab) return;

  console.log(`Navigating to sub-tab: ${config.subTab}`);

  // Different selectors based on category
  let subTabSelector: string;

  switch (config.category) {
    case 'analytics':
      subTabSelector = `[data-analytics-page="${config.subTab}"]`;
      break;
    case 'governance':
      subTabSelector = `[data-governance-page="${config.subTab}"]`;
      break;
    case 'newsletter':
      subTabSelector = `[data-newsletter-page="${config.subTab}"]`;
      break;
    case 'maps':
      // Map modes use either viz-mode buttons or view-mode buttons
      if (config.subTab === 'heatmap' || config.subTab === 'hexagons') {
        subTabSelector = `[data-viz-mode="${config.subTab}"], [data-mode="${config.subTab}"]`;
      } else {
        // 2d/3d view mode
        subTabSelector = `[data-view-mode="${config.subTab}"]`;
      }
      break;
    case 'globe':
      // Globe uses view-mode="3d"
      subTabSelector = '[data-view-mode="3d"]';
      break;
    default:
      console.warn(`Unknown sub-tab category: ${config.category}`);
      return;
  }

  try {
    // Click the sub-tab
    const subTab = page.locator(subTabSelector).first();
    if (await subTab.isVisible({ timeout: 2000 })) {
      await subTab.click();
      // Wait for the click to take effect
      await page.waitForTimeout(1000);
    } else {
      console.warn(`Sub-tab not visible: ${subTabSelector}`);
    }
  } catch (error) {
    console.warn(`Failed to navigate to sub-tab ${config.subTab}:`, error);
  }
}

/**
 * Wait for page to be fully ready for screenshot.
 * This ensures deterministic rendering by waiting for:
 * 1. DOM elements to be visible
 * 2. Fonts to be loaded
 * 3. Network to be idle
 * 4. WebGL content to be rendered (for maps/globe)
 * 5. Any modals to be dismissed
 */
async function waitForReady(page: Page, config: ScreenshotConfig): Promise<void> {
  // Wait for main selector
  await page.waitForSelector(config.waitForSelector, {
    state: 'visible',
    timeout: TIMEOUTS.elementVisible,
  });

  // Wait for additional selectors
  if (config.additionalSelectors) {
    for (const selector of config.additionalSelectors) {
      try {
        await page.waitForSelector(selector, {
          state: 'visible',
          timeout: TIMEOUTS.elementVisible,
        });
      } catch (error) {
        console.warn(`Optional selector not found: ${selector}`);
      }
    }
  }

  // Wait for fonts to be loaded (critical for consistent text rendering)
  await waitForFonts(page);

  // Wait for network to be idle
  await page.waitForLoadState('networkidle', { timeout: TIMEOUTS.navigation }).catch(() => {
    console.warn('Network did not become idle, continuing...');
  });

  // Dismiss any overlays that might have appeared (fallback)
  await dismissOverlays(page);

  // Wait for WebGL content if this is a map/globe/analytics view
  if (config.category === 'maps') {
    await waitForWebGLReady(page, '.maplibregl-canvas');
  } else if (config.category === 'globe') {
    await waitForWebGLReady(page, '#deck-canvas');
  } else if (config.category === 'analytics') {
    // Wait for ECharts to render
    await page.waitForSelector('canvas', { state: 'visible', timeout: 5000 }).catch(() => {});
  }

  // Wait for any additional time (for data loading, rendering stabilization)
  if (config.waitAfterNavigation) {
    await page.waitForTimeout(config.waitAfterNavigation);
  }

  // Final stabilization wait (allows any micro-animations to complete)
  await page.waitForTimeout(500);
}

/**
 * Capture a single screenshot with specified theme
 */
async function captureScreenshot(
  page: Page,
  config: ScreenshotConfig,
  theme: AppTheme = 'dark',
): Promise<{ success: boolean; path?: string; error?: string }> {
  // Add theme suffix to filename when capturing multiple themes
  const themeSuffix = (THEME_MODE === 'dark-and-light' || THEME_MODE === 'all-themes') ? `-${theme}` : '';
  const filename = `${config.id}${themeSuffix}.png`;
  const filepath = path.join(OUTPUT_DIR, filename);

  console.log(`\n${'='.repeat(60)}`);
  console.log(`Capturing: ${config.name} (${config.id}) [${theme} theme]`);
  console.log(`Description: ${config.description}`);
  console.log(`${'='.repeat(60)}`);

  try {
    // Navigate if needed
    if (config.navTab) {
      await navigateToTab(page, config.navTab);
    }

    // Handle sub-tabs (analytics pages, governance pages, map modes, etc.)
    if (config.subTab) {
      await navigateToSubTab(page, config);
    }

    // Wait for page to be ready
    await waitForReady(page, config);

    // Take screenshot
    await page.screenshot({
      path: filepath,
      fullPage: false, // Capture viewport only
      clip: config.clip,
      animations: 'disabled',
    });

    console.log(`Screenshot saved: ${filepath}`);
    return { success: true, path: filepath };
  } catch (error) {
    const errorMessage = error instanceof Error ? error.message : String(error);
    console.error(`Failed to capture ${config.id}: ${errorMessage}`);
    return { success: false, error: errorMessage };
  }
}

// =============================================================================
// Test Suite
// =============================================================================

test.describe('Documentation Screenshots', () => {
  test.beforeAll(async () => {
    ensureOutputDir();
    const themesToCapture = getThemesToCapture();
    const totalCaptures = SCREENSHOTS.length * themesToCapture.length;
    console.log('\n');
    console.log('╔════════════════════════════════════════════════════════════╗');
    console.log('║     DOCUMENTATION SCREENSHOT CAPTURE                       ║');
    console.log('╠════════════════════════════════════════════════════════════╣');
    console.log(`║ Output Directory: ${OUTPUT_DIR.substring(0, 40).padEnd(40)}║`);
    console.log(`║ Theme Mode: ${THEME_MODE.padEnd(46)}║`);
    console.log(`║ Themes: ${themesToCapture.join(', ').padEnd(50)}║`);
    console.log(`║ Screenshots: ${SCREENSHOTS.length.toString().padEnd(45)}║`);
    console.log(`║ Total Captures: ${totalCaptures.toString().padEnd(41)}║`);
    console.log('╚════════════════════════════════════════════════════════════╝');
    console.log('\n');
  });

  test.afterAll(async () => {
    // Print summary
    console.log('\n');
    console.log('╔════════════════════════════════════════════════════════════╗');
    console.log('║     SCREENSHOT CAPTURE SUMMARY                             ║');
    console.log('╚════════════════════════════════════════════════════════════╝');

    const successful = screenshotResults.filter((r) => r.success);
    const failed = screenshotResults.filter((r) => !r.success);

    console.log(`\nTotal: ${screenshotResults.length}`);
    console.log(`Successful: ${successful.length}`);
    console.log(`Failed: ${failed.length}`);

    const showThemeLabel = getThemesToCapture().length > 1;

    if (successful.length > 0) {
      console.log('\n✓ Successful Screenshots:');
      for (const result of successful) {
        const themeLabel = showThemeLabel ? ` [${result.theme}]` : '';
        console.log(`  - ${result.name}${themeLabel}: ${result.path} (${result.duration}ms)`);
      }
    }

    if (failed.length > 0) {
      console.log('\n✗ Failed Screenshots:');
      for (const result of failed) {
        const themeLabel = showThemeLabel ? ` [${result.theme}]` : '';
        console.log(`  - ${result.name}${themeLabel}: ${result.error}`);
      }
    }

    // Write results to JSON for CI processing
    const resultsPath = path.join(OUTPUT_DIR, 'results.json');
    fs.writeFileSync(
      resultsPath,
      JSON.stringify(
        {
          timestamp: new Date().toISOString(),
          themeMode: THEME_MODE,
          total: screenshotResults.length,
          successful: successful.length,
          failed: failed.length,
          screenshots: screenshotResults,
        },
        null,
        2,
      ),
    );
    console.log(`\nResults written to: ${resultsPath}`);
  });

  // Test: Login page screenshot (unauthenticated)
  test('01-login-page', async ({ page }) => {
    const config = SCREENSHOTS.find((s) => s.id === 'login-page')!;

    // CRITICAL: Setup deterministic page BEFORE any navigation
    // This sets localStorage flags and injects animation-disabling CSS
    await setupDeterministicPage(page);
    await disableAnimations(page); // Fallback if addInitScript didn't run yet
    await page.goto('/', { waitUntil: 'domcontentloaded' });

    // Capture in all requested themes
    const themes = getThemesToCapture();

    for (const theme of themes) {
      const startTime = Date.now();

      // Set color scheme for this capture
      await setColorScheme(page, theme);

      const result = await captureScreenshot(page, config, theme);

      screenshotResults.push({
        id: config.id,
        name: config.name,
        theme,
        success: result.success,
        error: result.error,
        path: result.path,
        duration: Date.now() - startTime,
      });

      expect(result.success, `Screenshot ${config.id} [${theme}] should succeed: ${result.error}`).toBe(
        true,
      );
    }
  });

  // Test: Authenticated views
  test('02-authenticated-views', async ({ page, context }) => {
    // CRITICAL: Setup deterministic page BEFORE any navigation
    // This sets localStorage flags and injects animation-disabling CSS
    // so that tour/wizard modals never appear in the first place
    await setupDeterministicPage(page);
    await disableAnimations(page); // Fallback

    // Login first
    const loginSuccess = await login(page);
    expect(loginSuccess, 'Login should succeed').toBe(true);

    // Dismiss any overlays that might have appeared despite localStorage flags
    await dismissOverlays(page);

    // Save auth state for potential future use
    await context.storageState({ path: path.join(OUTPUT_DIR, '.auth-state.json') });

    // Capture all authenticated screenshots in order (lightweight to heavy)
    const authenticatedScreenshots = SCREENSHOTS.filter((s) => s.requiresAuth);

    // Get themes to capture based on THEME_MODE
    const themes = getThemesToCapture();

    for (const config of authenticatedScreenshots) {
      console.log(`\nProcessing: ${config.name}`);

      // Clean up WebGL between heavy screenshots to prevent memory issues
      if (config.category === 'maps' || config.category === 'globe' || config.category === 'analytics') {
        await cleanupWebGL(page);
      }

      // Capture screenshot in each theme
      for (const theme of themes) {
        const startTime = Date.now();

        // Set color scheme for this capture
        await setColorScheme(page, theme);

        const result = await captureScreenshot(page, config, theme);

        screenshotResults.push({
          id: config.id,
          name: config.name,
          theme,
          success: result.success,
          error: result.error,
          path: result.path,
          duration: Date.now() - startTime,
        });

        // Don't fail the entire test on individual screenshot failure
        // This allows capturing as many screenshots as possible
        if (!result.success) {
          console.warn(`Warning: ${config.id} [${theme}] failed, continuing with remaining screenshots`);
        }
      }
    }

    // Final cleanup
    await cleanupWebGL(page);

    // Verify at least core screenshots were captured (in at least one theme)
    const coreScreenshots = ['live-activity', 'analytics-overview', 'map-view'];
    const capturedCore = screenshotResults.filter(
      (r) => r.success && coreScreenshots.includes(r.id),
    );

    // Expect at least half of core screenshots (accounting for theme variants)
    const expectedMinimum = Math.ceil(coreScreenshots.length / 2);
    expect(
      capturedCore.length,
      `At least ${expectedMinimum} core screenshots should be captured`,
    ).toBeGreaterThanOrEqual(expectedMinimum);
  });
});
