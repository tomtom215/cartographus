// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * E2E Test: Login Flow with JWT Authentication
 *
 * Tests the authentication flow including:
 * - Login form rendering
 * - Credential validation
 * - JWT token issuance
 * - Session persistence
 * - Protected route access
 * - Logout functionality
 *
 * NOTE: This test file disables API mocking because it tests real
 * authentication flows that require the actual backend APIs.
 */

import {
  test,
  expect,
  TIMEOUTS,
  SELECTORS,
  getTestCredentials,
  performLogin,
} from "./fixtures";

// Disable API mocking for authentication tests - we need real API responses
// Disable auth state loading - login tests MUST start unauthenticated
test.use({ autoMockApi: false, autoLoadAuthState: false });

test.describe("Authentication Flow", () => {
  // This test file runs in the 'login-tests' project which has NO storageState
  // configured, so tests start completely unauthenticated (like auth.setup.ts)

  test.beforeEach(async ({ page }) => {
    // Navigate to the page - no auth cookies/localStorage since project has no storageState
    await page.goto("/", {
      waitUntil: "domcontentloaded",
      timeout: TIMEOUTS.EXTENDED,
    });

    // Wait for login form to be visible
    await expect(page.locator(SELECTORS.LOGIN_CONTAINER)).toBeVisible({
      timeout: TIMEOUTS.DEFAULT,
    });
  });

  test("should redirect to login when not authenticated", async ({ page }) => {
    // Login form visibility is already verified in beforeEach
    // Just verify the form elements exist
    await expect(page.locator(SELECTORS.LOGIN_USERNAME)).toBeVisible();
    await expect(page.locator(SELECTORS.LOGIN_PASSWORD)).toBeVisible();
    await expect(page.locator(SELECTORS.LOGIN_SUBMIT)).toBeVisible();
  });

  test("should show error on invalid credentials", async ({ page }) => {
    // ROOT CAUSE FIX: Wait for JavaScript to fully load before interacting
    // WHY: The form might submit as a regular GET request (URL shows query params)
    // if JavaScript hasn't attached its event handlers yet
    await expect(page.locator(SELECTORS.LOGIN_SUBMIT)).toBeEnabled({
      timeout: TIMEOUTS.MEDIUM,
    });

    // Enter invalid credentials
    await page.fill(SELECTORS.LOGIN_USERNAME, "invalid_user");
    await page.fill(SELECTORS.LOGIN_PASSWORD, "wrong_password");

    // Use JavaScript click for CI reliability
    // WHY: Playwright's .click() may fail in headless/SwiftShader environments
    await page.evaluate(() => {
      const btn = document.querySelector('button[type="submit"]') as HTMLElement;
      if (btn) btn.click();
    });

    // Wait for form submission to complete (either error or redirect)
    // ROOT CAUSE FIX: The error might take time to appear after API response
    await page.waitForFunction(
      (selector) => {
        const errorEl = document.querySelector(selector);
        return errorEl !== null;
      },
      '.error-message, #login-error',
      { timeout: TIMEOUTS.MEDIUM }
    );

    // Should show error message (has .show class added which makes it visible)
    await expect(page.locator(SELECTORS.LOGIN_ERROR)).toBeVisible({
      timeout: TIMEOUTS.MEDIUM,
    });
  });

  test("should successfully login with valid credentials", async ({ page }) => {
    // ROOT CAUSE FIX: Wait for JavaScript to fully load before interacting
    // WHY: The form might submit as a regular GET request if JS hasn't loaded
    await expect(page.locator(SELECTORS.LOGIN_SUBMIT)).toBeEnabled({
      timeout: TIMEOUTS.MEDIUM,
    });

    const { username, password } = getTestCredentials();

    await page.fill(SELECTORS.LOGIN_USERNAME, username);
    await page.fill(SELECTORS.LOGIN_PASSWORD, password);

    // Use JavaScript click for CI reliability
    // WHY: Playwright's .click() may fail in headless/SwiftShader environments
    await page.evaluate(() => {
      const btn = document.querySelector('button[type="submit"]') as HTMLElement;
      if (btn) btn.click();
    });

    // DETERMINISTIC WAIT: Wait for login state transition using condition-based wait
    // WHY: The app transition from login to main view can take variable time
    await page.waitForFunction(
      (selectors) => {
        const loginContainer = document.querySelector(selectors.login);
        const appContainer = document.querySelector(selectors.app);
        // Login is complete when: login container is hidden AND app is visible
        const loginHidden = !loginContainer ||
          (loginContainer as HTMLElement).style.display === 'none' ||
          !(loginContainer as HTMLElement).offsetParent;
        const appVisible = appContainer &&
          (appContainer as HTMLElement).offsetParent !== null;
        return loginHidden && appVisible;
      },
      { login: SELECTORS.LOGIN_CONTAINER, app: SELECTORS.APP },
      { timeout: TIMEOUTS.DEFAULT }
    );

    // Should redirect to main application
    await expect(page.locator(SELECTORS.APP)).toBeVisible({
      timeout: TIMEOUTS.DEFAULT,
    });
    await expect(page.locator(SELECTORS.LOGIN_CONTAINER)).not.toBeVisible();
  });

  test("should persist session with Remember Me", async ({ page, context }) => {
    // Login container already visible from beforeEach
    const { username, password } = getTestCredentials();

    // Check "Remember Me" if available
    const rememberCheckbox = page.locator(
      'input[name="remember_me"], input#remember-me',
    );
    if (await rememberCheckbox.isVisible()) {
      await rememberCheckbox.check();
    }

    await page.fill(SELECTORS.LOGIN_USERNAME, username);
    await page.fill(SELECTORS.LOGIN_PASSWORD, password);

    // Use JavaScript click for CI reliability
    // WHY: Playwright's .click() may fail in headless/SwiftShader environments
    await page.evaluate(() => {
      const btn = document.querySelector('button[type="submit"]') as HTMLElement;
      if (btn) btn.click();
    });

    // Wait for login success
    await expect(page.locator(SELECTORS.APP)).toBeVisible({
      timeout: TIMEOUTS.DEFAULT,
    });

    // Verify auth cookie exists
    const cookies = await context.cookies();
    const authCookie = cookies.find((c) => c.name === "token");
    expect(authCookie).toBeDefined();
    expect(authCookie?.httpOnly).toBe(true);
  });

  test("should allow access to protected routes after login", async ({
    page,
  }) => {
    // Login container already visible from beforeEach
    // Use the performLogin helper
    await performLogin(page);

    // Test protected API endpoint access
    const response = await page.request.post("/api/v1/sync");

    // Should not return 401 Unauthorized
    expect(response.status()).not.toBe(401);
  });

  test("should handle logout correctly", async ({ page }) => {
    // Login container already visible from beforeEach
    // Use the performLogin helper
    await performLogin(page);

    // Wait for app to be fully visible and interactive
    await expect(page.locator(SELECTORS.APP)).toBeVisible({
      timeout: TIMEOUTS.DEFAULT,
    });

    // Find the logout button - it should be in the header
    const logoutButton = page.locator(SELECTORS.LOGOUT_BUTTON);

    // Wait for logout button to be visible and enabled
    await expect(logoutButton).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
    await expect(logoutButton).toBeEnabled({ timeout: TIMEOUTS.SHORT });

    // Click the logout button
    await logoutButton.click();

    // Should redirect to login - wait for login container to be visible
    await expect(page.locator(SELECTORS.LOGIN_CONTAINER)).toBeVisible({
      timeout: TIMEOUTS.DEFAULT,
    });

    // App should be hidden after logout
    await expect(page.locator(SELECTORS.APP)).not.toBeVisible();
  });

  test("should validate input fields", async ({ page }) => {
    // Try to submit with empty fields
    // Use JavaScript click for CI reliability
    // WHY: Playwright's .click() may fail in headless/SwiftShader environments
    await page.evaluate(() => {
      const btn = document.querySelector('button[type="submit"]') as HTMLElement;
      if (btn) btn.click();
    });

    // Browser validation should prevent submission or show error
    const usernameInput = page.locator(SELECTORS.LOGIN_USERNAME);
    const isRequired = await usernameInput.getAttribute("required");
    expect(isRequired).not.toBeNull();
  });

  test("should show/hide password functionality", async ({ page }) => {
    const passwordInput = page.locator(SELECTORS.LOGIN_PASSWORD);

    // Password should be hidden by default
    await expect(passwordInput).toHaveAttribute("type", "password");

    // Look for toggle button
    const toggleButton = page.locator(
      'button:has([class*="eye"]), button:has([class*="show-password"])',
    );

    if (await toggleButton.isVisible()) {
      await toggleButton.click();

      // Password should now be visible
      await expect(passwordInput).toHaveAttribute("type", "text");
    }
  });
});
