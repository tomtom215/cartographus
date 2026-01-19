// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Global Authentication Setup
 *
 * This file runs ONCE before all tests to establish authentication.
 * It logs in with admin credentials and saves the authenticated state
 * (cookies + localStorage) to a file that all other tests can reuse.
 *
 * ARCHITECTURE (ADR-0024: Deterministic E2E Test Mocking)
 * =========================================================
 * This setup uses the unified mock-server.ts for API mocking.
 * Routes are registered at the CONTEXT level for deterministic behavior.
 *
 * This approach prevents every test from having to login individually,
 * which is faster and more reliable.
 */

import { test as setup, expect } from '@playwright/test';
import { setupMockServer } from './fixtures/mock-server';
import { setupAllErrorFilters } from './fixtures/auth-mocks';
import { TIMEOUTS, SELECTORS, getTestCredentials } from './fixtures/constants';

const authFile = 'playwright/.auth/user.json';

setup('authenticate', async ({ browser }) => {
  // Increase timeout for this test (authentication can be slow in CI)
  setup.setTimeout(TIMEOUTS.MAX);

  // Create a NEW context with mocking enabled BEFORE any page is created
  // This is the key to deterministic mocking
  const context = await browser.newContext();

  // Setup mock server on context BEFORE creating the page
  await setupMockServer(context, {
    enableApiMocking: true,
    enableTileMocking: true,
    logRequests: !!process.env.CI,
  });

  console.log('[AUTH] Created context with mock server');

  // Now create the page from the mocked context
  const page = await context.newPage();

  // Set up error filters to reduce noise from benign errors during auth
  setupAllErrorFilters(page);

  // E2E Debug Network Logging
  const e2eDebug = process.env.E2E_DEBUG === 'true';
  if (e2eDebug) {
    console.log('[E2E-DEBUG] Network logging enabled');
    let requestCount = 0;
    let failedCount = 0;

    page.on('request', request => {
      requestCount++;
      console.log(`[E2E-DEBUG] >> ${request.method()} ${request.url()}`);
    });

    page.on('requestfailed', request => {
      failedCount++;
      const failure = request.failure();
      console.error(`[E2E-DEBUG] XX FAILED: ${request.method()} ${request.url()}`);
      console.error(`[E2E-DEBUG]    Error: ${failure?.errorText || 'unknown'}`);
    });

    page.on('close', () => {
      console.log(`[E2E-DEBUG] === Summary: ${requestCount} requests, ${failedCount} failed ===`);
    });
  }

  // Set onboarding, setup wizard, and cookie consent flags BEFORE navigating
  await page.addInitScript(() => {
    localStorage.setItem('onboarding_completed', 'true');
    localStorage.setItem('onboarding_skipped', 'true');
    localStorage.setItem('setup_wizard_completed', 'true');
    localStorage.setItem('setup_wizard_skipped', 'true');
    const cookiePreferences = {
      necessary: true,
      analytics: false,
      preferences: false,
      consentGiven: true,
      consentDate: new Date().toISOString(),
    };
    localStorage.setItem('cartographus-cookie-consent', JSON.stringify(cookiePreferences));
  });

  // Navigate to login page with shorter initial timeout - fail fast if server down
  try {
    await page.goto('/', { waitUntil: 'commit', timeout: TIMEOUTS.MEDIUM });
    await page.waitForLoadState('domcontentloaded', { timeout: TIMEOUTS.DEFAULT });
  } catch (error) {
    await context.close();
    throw new Error(`Failed to load app - server may not be running at localhost:3857. Error: ${error}`);
  }

  // Wait for login form to be visible
  await expect(page.locator(SELECTORS.LOGIN_CONTAINER)).toBeVisible({ timeout: TIMEOUTS.DEFAULT });

  // Wait for JS to initialize (button should be enabled and have correct text)
  await expect(page.locator(SELECTORS.LOGIN_BUTTON)).toBeEnabled({ timeout: TIMEOUTS.MEDIUM });
  await expect(page.locator(SELECTORS.LOGIN_BUTTON)).toHaveText('Sign In');
  console.log('[AUTH] Login form JS initialized');

  // Get credentials from environment or use defaults
  const { username, password } = getTestCredentials();

  console.log(`[AUTH] Authenticating as: ${username}`);

  // Fill in credentials
  await page.fill(SELECTORS.LOGIN_USERNAME, username);
  await page.fill(SELECTORS.LOGIN_PASSWORD, password);

  // Click sign in button
  await page.click(SELECTORS.LOGIN_SUBMIT);

  console.log('[AUTH] Clicked sign in button, waiting for authentication...');

  // Wait for login form to become HIDDEN (not removed from DOM)
  await expect(page.locator(SELECTORS.LOGIN_CONTAINER)).not.toBeVisible({ timeout: TIMEOUTS.LONG });
  console.log('[AUTH] Login form disappeared');

  // Wait for app container to be visible (appears immediately after auth)
  await expect(page.locator(SELECTORS.APP)).toBeVisible({ timeout: TIMEOUTS.DEFAULT });
  console.log('[AUTH] App container visible');

  // Verify we have navigation (appears before data loading)
  const nav = page.locator('nav, #nav, .nav, [role="navigation"]').first();
  await expect(nav).toBeVisible({ timeout: TIMEOUTS.DEFAULT });
  console.log('[AUTH] Navigation visible');

  // Verify auth tokens are actually saved in localStorage
  const authState = await page.evaluate(() => {
    return {
      token: localStorage.getItem('auth_token'),
      username: localStorage.getItem('auth_username'),
      expiresAt: localStorage.getItem('auth_expires_at'),
    };
  });

  if (!authState.token) {
    await context.close();
    throw new Error('Authentication failed: auth_token not found in localStorage after login');
  }
  if (!authState.expiresAt) {
    await context.close();
    throw new Error('Authentication failed: auth_expires_at not found in localStorage after login');
  }

  // Verify token isn't expired
  const expiresAt = new Date(authState.expiresAt);
  const now = new Date();
  if (expiresAt <= now) {
    await context.close();
    throw new Error(`Authentication failed: token already expired at ${authState.expiresAt}`);
  }

  // Calculate token validity
  const validForMs = expiresAt.getTime() - now.getTime();
  const validForMinutes = Math.round(validForMs / 60000);
  console.log(`[AUTH] Token valid for ${validForMinutes} minutes (expires: ${authState.expiresAt})`);

  if (validForMinutes < 10) {
    console.warn(`[AUTH] WARNING: Token expires in ${validForMinutes} minutes - tests may fail`);
  }

  // DETERMINISTIC: Wait for network to settle (all initialization requests complete)
  await page.waitForLoadState('networkidle', { timeout: TIMEOUTS.DATA_LOAD }).catch(() => {});

  console.log('[AUTH] Authentication successful - saving state');

  // Save authentication state to file
  await context.storageState({ path: authFile });

  console.log(`[AUTH] Saved authentication state to: ${authFile}`);
  console.log(`[AUTH] State contains: token=${authState.token ? 'present' : 'missing'}, username=${authState.username || 'unknown'}`);

  // Cleanup
  await context.close();
});
