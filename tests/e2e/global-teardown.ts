// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Playwright Global Teardown
 *
 * This runs ONCE after all tests to stop the mock API server.
 */

import { FullConfig } from '@playwright/test';
import { stopMockServer } from './mock-api-server';

async function globalTeardown(_config: FullConfig) {
  console.log('[GLOBAL-TEARDOWN] Stopping mock API server...');

  try {
    await stopMockServer();
    console.log('[GLOBAL-TEARDOWN] Mock API server stopped');
  } catch (error) {
    // Non-fatal - server may have already stopped
    console.warn('[GLOBAL-TEARDOWN] Error stopping mock server:', error);
  }
}

export default globalTeardown;
