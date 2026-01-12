// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Authentication Setup Utilities
 *
 * ARCHITECTURE (ADR-0024: Deterministic E2E Test Mocking)
 * =========================================================
 *
 * This module now ONLY provides error filtering utilities.
 * All API mocking has been moved to mock-server.ts for centralization.
 *
 * WHAT WAS REMOVED:
 * - setupAuthMocks() - API routes are now in mock-server.ts
 * - All page.route() calls - replaced by context.route() in mock-server.ts
 *
 * WHAT REMAINS:
 * - Error filtering utilities (for reducing test noise)
 * - Known benign error patterns
 *
 * This refactoring eliminates the root cause of flaky tests:
 * - No more conflicting route registrations between files
 * - Single source of truth for all API mocking
 * - Deterministic context-level routing
 */

import type { Page } from '@playwright/test';

/**
 * Known benign console errors that occur during app initialization.
 * Filter these out to avoid test noise.
 */
export const KNOWN_BENIGN_ERRORS = [
  // Manager initialization errors (may initialize before containers visible)
  '[BackupRestore]',      // BackupRestoreManager - container may not be visible during auth
  '[Settings]',           // SettingsManager - panel may not be visible during auth
  '[ErrorBoundary]',      // ErrorBoundaryManager / deck.gl - may initialize before data loads
  '[ADR-0020]',           // Detection engine - may initialize before data available
  '[ADR-0022]',           // Dedupe audit - may initialize before data available

  // WebGL/Graphics errors (expected in headless/CI environments)
  'WebSocket',   // WebSocket connection - mocked during auth setup
  'deck.gl',     // deck.gl initialization - may render before data available
  '_drawLayers', // deck.gl internal - renders before data ready
  'WebGL',       // WebGL context errors in headless mode
  'SwiftShader', // Software renderer messages

  // Network/API errors (expected with mocked endpoints)
  'Failed to fetch',    // Network errors during API mocking
  'NetworkError',       // Network errors during route interception
  'UNMOCKED_ENDPOINT',  // Catch-all mock handler responses
  '404',                // API endpoints that may not be mocked yet
  '501',                // Unmocked endpoint responses

  // Chart/visualization errors (expected during data loading)
  'echarts',     // ECharts may render before data loads

  // Service worker errors (expected during test setup)
  'service-worker',     // SW may not register during tests
  'ServiceWorker',      // SW registration errors
] as const;

/**
 * Check if an error message is a known benign error.
 */
export function isBenignError(errorText: string): boolean {
  return KNOWN_BENIGN_ERRORS.some(pattern => errorText.includes(pattern));
}

/**
 * Check if a page error is a deck.gl null reference error during init.
 */
export function isDeckGLInitError(errorMessage: string): boolean {
  return errorMessage.includes('null') && (errorMessage.includes('id') || errorMessage.includes('_drawLayers'));
}

/**
 * Set up console error filtering for tests.
 * Filters out known benign errors that occur during initialization.
 *
 * @param page - Playwright page
 * @param logPrefix - Optional prefix for error logs (default: '')
 */
export function setupConsoleErrorFilter(page: Page, logPrefix = ''): void {
  page.on('console', msg => {
    if (msg.type() === 'error') {
      const text = msg.text();
      if (!isBenignError(text)) {
        console.error(`${logPrefix}Browser console error: ${text}`);
      }
    }
  });
}

/**
 * Set up page error filtering for tests.
 * Filters out deck.gl errors that occur during initial render.
 *
 * @param page - Playwright page
 * @param logPrefix - Optional prefix for error logs (default: '')
 */
export function setupPageErrorFilter(page: Page, logPrefix = ''): void {
  page.on('pageerror', error => {
    const msg = error.message;
    if (!isDeckGLInitError(msg)) {
      console.error(`${logPrefix}Page error: ${msg}`);
    }
  });
}

/**
 * Set up all error filters for tests.
 * Convenience function that calls both console and page error filters.
 *
 * @param page - Playwright page
 */
export function setupAllErrorFilters(page: Page): void {
  setupConsoleErrorFilter(page, '  ');
  setupPageErrorFilter(page, '  ');
}

/**
 * @deprecated API mocking is now handled by mock-server.ts
 * This function is kept for backwards compatibility but does nothing.
 * Use setupMockServer() from mock-server.ts instead.
 */
export async function setupAuthMocks(_page: Page): Promise<void> {
  console.warn('[DEPRECATED] setupAuthMocks() is deprecated. API mocking is now in mock-server.ts');
  // No-op - mocking is now done at context level in mock-server.ts
}
