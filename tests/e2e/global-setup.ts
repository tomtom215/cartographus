// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Playwright Global Setup
 *
 * This runs ONCE before all tests to start the mock API server.
 * The mock server provides deterministic, concurrent-safe HTTP responses.
 *
 * ARCHITECTURE:
 * - Express mock server runs on port 3900 (fixed, separate from app)
 * - Real backend runs on ports 3857-3860 (per shard)
 * - Playwright intercepts API requests and proxies them to Express
 * - Express handles concurrent requests properly (no CDP race conditions)
 *
 * This separation ensures the mock server can always start regardless
 * of whether the real backend is running.
 *
 * @see tests/e2e/mock-api-server.ts - The Express mock server
 * @see ADR-0025: Deterministic E2E Test Mocking
 */

import { FullConfig } from '@playwright/test';
import { startMockServer } from './mock-api-server';

// Store server reference for teardown
declare global {
  // eslint-disable-next-line no-var
  var __mockServerInstance: import('http').Server | undefined;
}

// The mock server always runs on port 3900, separate from the app
// This matches the MOCK_SERVER_PORT in mock-server.ts proxy configuration
const MOCK_SERVER_PORT = 3900;

async function globalSetup(_config: FullConfig) {
  console.log(`[GLOBAL-SETUP] Starting mock API server on port ${MOCK_SERVER_PORT}...`);

  try {
    const server = await startMockServer({
      port: MOCK_SERVER_PORT,
      logRequests: process.env.E2E_VERBOSE === 'true'
    });

    // Store for teardown
    global.__mockServerInstance = server;

    console.log(`[GLOBAL-SETUP] Mock API server ready on port ${MOCK_SERVER_PORT}`);
  } catch (error) {
    // If port is already in use, another shard may have started it
    if ((error as NodeJS.ErrnoException).code === 'EADDRINUSE') {
      console.log(`[GLOBAL-SETUP] Port ${MOCK_SERVER_PORT} already in use - using existing instance`);
      // This is OK - another shard started the server
    } else {
      console.error('[GLOBAL-SETUP] Failed to start mock server:', error);
      throw error;
    }
  }
}

export default globalSetup;
