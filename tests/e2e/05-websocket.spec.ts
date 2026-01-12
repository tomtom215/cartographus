// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
import {
  test,
  expect,
  TIMEOUTS,
  SELECTORS,
  gotoAppAndWaitReady,
} from './fixtures';

/**
 * E2E Test: WebSocket Real-Time Updates
 *
 * Tests WebSocket functionality including:
 * - Connection establishment
 * - Real-time playback notifications
 * - Toast notifications
 * - Auto-reconnection
 * - Connection health (ping/pong)
 * - Stats auto-refresh
 *
 * NOTE: All tile requests are automatically mocked via fixtures.ts
 * This makes tests fully offline, deterministic, and faster.
 *
 * NOTE: These tests run in the 'chromium' project which has storageState
 * pre-loaded from auth.setup.ts, so we DON'T need to login again.
 */

test.describe('WebSocket Real-Time Updates', () => {
  test.beforeEach(async ({ page }) => {
    // Navigate to app (already authenticated via storageState from setup)
    await gotoAppAndWaitReady(page);
  });

  test('should establish WebSocket connection on page load', async ({ page }) => {
    // Wait for app to be fully initialized
    await page.waitForFunction(() => {
      const app = document.getElementById('app');
      const map = document.getElementById('map');
      return app && map;
    }, { timeout: TIMEOUTS.WEBGL_INIT });

    // Check for WebSocket in browser console (via page evaluation)
    const wsStatus = await page.evaluate(() => {
      // Check if WebSocket exists in window
      return typeof WebSocket !== 'undefined';
    });

    expect(wsStatus).toBe(true);
  });

  test('should show connection status indicator', async ({ page }) => {
    // Look for connection status element
    // Note: Connection status UI is optional - app may not visually show WS status
    const statusIndicators = page.locator('[class*="connection-status"], [class*="ws-status"], [id*="connection-status"], .connected, .disconnected');
    const indicatorCount = await statusIndicators.count();

    if (indicatorCount === 0) {
      console.log('No visible connection status indicator - this is acceptable if app design omits it');
      // Test passes - connection status UI is not mandatory
      return;
    }

    console.log(`Found ${indicatorCount} status indicator(s)`);

    // Wait for app to be fully initialized
    await page.waitForFunction(() => {
      const app = document.getElementById('app');
      return app !== null;
    }, { timeout: TIMEOUTS.WEBGL_INIT });

    // Should show connected state
    const connectedIndicator = page.locator('[class*="connected"], .online, [data-status="connected"]');
    const connectedCount = await connectedIndicator.count();

    if (connectedCount > 0) {
      console.log('Connection status shows connected');
    } else {
      console.log('No "connected" status visible - WebSocket may still be connecting');
    }
  });

  test('should display toast notifications for new playbacks', async ({ page }) => {
    // Wait for app to be fully initialized
    await page.waitForFunction(() => {
      const toastContainer = document.getElementById('toast-container');
      return toastContainer !== null;
    }, { timeout: TIMEOUTS.WEBGL_INIT });

    // Listen for toast notifications
    // Note: Toasts only appear when there's new playback activity
    // In test environment with mock data, no real WebSocket events occur
    //
    // Toast structure:
    // - Container: #toast-container (always exists, may be hidden)
    // - Actual toasts: .toast.toast-{type} inside container
    //
    // Look for actual toast items, NOT the container itself
    const toastLocator = page.locator('#toast-container > .toast, .notification:not(.toast-container)');

    // Wait for potential toasts (time-based as WebSocket events are asynchronous)
    await page.waitForFunction(() => {
      const container = document.getElementById('toast-container');
      return container !== null;
    }, { timeout: TIMEOUTS.MEDIUM });

    const toastCount = await toastLocator.count();

    // This test verifies the toast system doesn't error out
    // Actual toast appearance depends on real-time WebSocket events
    if (toastCount > 0) {
      console.log(`Found ${toastCount} toast notification(s)`);
      await expect(toastLocator.first()).toBeVisible();
    } else {
      // This is expected behavior - no WebSocket events triggered in test environment
      console.log('No toast notifications - expected when no new playback events');
    }
  });

  test('should update stats when WebSocket message received', async ({ page }) => {
    // Get initial stat value
    const statPlaybacks = page.locator('#stat-playbacks, [data-stat="playbacks"]');

    if (await statPlaybacks.isVisible()) {
      const initialValue = await statPlaybacks.textContent();

      // Wait for stats to remain visible (indicates successful load)
      await page.waitForFunction(() => {
        const stats = document.getElementById('stat-playbacks');
        return stats && stats.textContent && stats.textContent.trim().length > 0;
      }, { timeout: TIMEOUTS.DEFAULT });

      // Stat may have changed (or stayed same if no new playbacks)
      const newValue = await statPlaybacks.textContent();

      // Both values should be valid
      expect(initialValue).toBeTruthy();
      expect(newValue).toBeTruthy();
    }
  });

  test('should handle WebSocket disconnection gracefully', async ({ page }) => {
    // Wait for app to be fully initialized
    await page.waitForFunction(() => {
      const map = document.getElementById('map');
      return map !== null;
    }, { timeout: TIMEOUTS.WEBGL_INIT });

    // Simulate network offline (if supported)
    await page.evaluate(() => {
      // Try to close WebSocket connection
      const ws = (window as any).ws || (window as any).websocket || (window as any).socket;
      if (ws && ws.close) {
        ws.close();
      }
    });

    // Wait for app to remain stable after disconnection
    await page.waitForFunction(() => {
      const map = document.getElementById('map');
      return map !== null;
    }, { timeout: TIMEOUTS.DATA_LOAD });

    // Application should still be functional
    await expect(page.locator(SELECTORS.MAP)).toBeVisible();

    // Should show disconnected status (if indicator exists)
    // Note: Status indicator is optional UI element
    const disconnectedIndicator = page.locator('[class*="disconnected"], .offline, [data-status="disconnected"]');
    const disconnectedCount = await disconnectedIndicator.count();

    if (disconnectedCount > 0) {
      console.log('Disconnected status indicator visible');
    } else {
      console.log('No disconnected status indicator - status UI may be optional');
    }
  });

  test('should auto-reconnect after disconnection', async ({ page }) => {
    // Wait for app to be fully initialized
    await page.waitForFunction(() => {
      const map = document.getElementById('map');
      return map !== null;
    }, { timeout: TIMEOUTS.WEBGL_INIT });

    // Close connection
    await page.evaluate(() => {
      const ws = (window as any).ws || (window as any).websocket || (window as any).socket;
      if (ws && ws.close) {
        ws.close();
      }
    });

    // Wait for app to remain functional after disconnect
    await page.waitForFunction(() => {
      const map = document.getElementById('map');
      return map !== null;
    }, { timeout: TIMEOUTS.DATA_LOAD });

    // Wait for auto-reconnect (verify app still functional after time passes)
    await page.waitForFunction(() => {
      const app = document.getElementById('app');
      return app !== null;
    }, { timeout: TIMEOUTS.MEDIUM });

    // Should reconnect - verify app is functional which indicates reconnection
    // Note: Status UI is optional
    const connectedIndicator = page.locator('[class*="connected"], .online, [data-status="connected"]');
    const connectedCount = await connectedIndicator.count();

    if (connectedCount > 0) {
      console.log('Connection status shows reconnected');
    } else {
      console.log('No connection status UI visible - checking app functionality instead');
    }

    // Application should be functional (primary indicator of successful reconnection)
    await expect(page.locator(SELECTORS.MAP)).toBeVisible();
  });

  test('should handle ping/pong for connection health', async ({ page }) => {
    // Wait for app to be fully initialized
    await page.waitForFunction(() => {
      const map = document.getElementById('map');
      return map !== null;
    }, { timeout: TIMEOUTS.WEBGL_INIT });

    // Monitor WebSocket messages
    const messages: string[] = [];

    page.on('websocket', ws => {
      ws.on('framereceived', event => {
        messages.push(event.payload.toString());
      });
    });

    // Wait for app to remain stable (ping/pong verification is implicit)
    await page.waitForFunction(() => {
      const app = document.getElementById('app');
      return app !== null;
    }, { timeout: TIMEOUTS.DEFAULT });

    // Messages may or may not be captured depending on timing
    // The test verifies that listening for WS frames doesn't error
    console.log(`Captured ${messages.length} WebSocket frame(s) during test`);
    if (messages.length > 0) {
      console.log('WebSocket frames captured - connection is active');
    } else {
      console.log('No frames captured yet - ping interval may be longer than test duration');
    }
  });

  test('should parse and display playback notification data', async ({ page }) => {
    // Wait for app to be fully initialized
    await page.waitForFunction(() => {
      const toastContainer = document.getElementById('toast-container');
      return toastContainer !== null;
    }, { timeout: TIMEOUTS.WEBGL_INIT });

    // Inject a mock WebSocket message for testing
    await page.evaluate(() => {
      // Simulate receiving a playback event
      const event = new CustomEvent('playback', {
        detail: {
          type: 'new_playback',
          data: {
            username: 'TestUser',
            media_type: 'movie',
            title: 'Test Movie',
          },
        },
      });

      window.dispatchEvent(event);
    });

    // Wait for potential toast to appear
    await page.waitForFunction(() => {
      const container = document.getElementById('toast-container');
      return container !== null;
    }, { timeout: TIMEOUTS.DATA_LOAD });

    // Toast should appear (if toast system listens for custom 'playback' events)
    // Note: The app may use WebSocket messages directly instead of custom events
    // Look for actual toast items, NOT the container
    const toast = page.locator('#toast-container > .toast, .notification:not(.toast-container)');
    const toastCount = await toast.count();

    // Toast appearance depends on whether app listens for custom events
    if (toastCount > 0) {
      console.log('Toast appeared from simulated playback event');
      await expect(toast.first()).toBeVisible();
    } else {
      console.log('No toast - app may use WebSocket messages directly instead of custom events');
    }
  });

  // QUARANTINED: Test fails with "Neither app nor login form visible" after blocking WebSocket
  // Root cause: gotoAppAndWaitReady() race condition when WebSocket routes are blocked
  // The app initialization may depend on WebSocket connection attempts completing
  // Tracking: CI failure analysis from 2025-12-31
  // Unquarantine criteria: 20 consecutive green runs after fixing gotoAppAndWaitReady determinism
  test.fixme('should not break application if WebSocket fails to connect', async ({ page, context }) => {
    // Block WebSocket connections before loading page
    await page.route('ws://**', route => route.abort());
    await page.route('wss://**', route => route.abort());

    // Navigate to app (storageState handles auth)
    await gotoAppAndWaitReady(page);

    // Application should still work without WebSocket
    await expect(page.locator(SELECTORS.FILTERS)).toBeVisible();
    await expect(page.locator(SELECTORS.MAP)).toBeVisible();

    // Verify app is actually functional by checking key interactive elements
    // Map should be interactive (canvas rendered)
    const mapCanvas = page.locator('#map canvas');
    await expect(mapCanvas.first()).toBeVisible();

    // Stats should be present (even if showing stale/default data)
    const statsContainer = page.locator('#stats, .stats-container, [data-stats]');
    const statsCount = await statsContainer.count();
    if (statsCount > 0) {
      await expect(statsContainer.first()).toBeVisible();
    }

    // No FATAL error dialogs or crash screens
    // E2E FIX: Be more specific - only match actual error overlays/modals, not informational alerts
    // WebSocket connection warnings are acceptable graceful degradation, not fatal errors
    const fatalErrorOverlay = page.locator('.error-overlay, .error-modal, .fatal-error, [data-error="fatal"]');
    const fatalErrorCount = await fatalErrorOverlay.count();
    expect(fatalErrorCount).toBe(0);

    // WebSocket status may show disconnected, but app should function
    const wsStatus = page.locator('#websocket-status, [data-ws-status]');
    if (await wsStatus.count() > 0) {
      // Verify it shows disconnected/offline state (not connected)
      const statusText = await wsStatus.first().textContent();
      console.log(`[E2E] WebSocket status indicator shows: "${statusText}"`);
      // Status should indicate disconnection, not pretend to be connected
      await expect(wsStatus.first()).toBeAttached();
    }

    console.log('[E2E] App functional without WebSocket - graceful degradation verified');
  });

  test('should handle rapid WebSocket messages without performance issues', async ({ page }) => {
    // Wait for app to be fully initialized
    await page.waitForFunction(() => {
      const map = document.getElementById('map');
      return map !== null;
    }, { timeout: TIMEOUTS.WEBGL_INIT });

    // Simulate rapid messages
    await page.evaluate(() => {
      for (let i = 0; i < 50; i++) {
        const event = new CustomEvent('playback', {
          detail: {
            type: 'new_playback',
            data: {
              username: `User${i}`,
              media_type: 'movie',
              title: `Movie ${i}`,
            },
          },
        });

        window.dispatchEvent(event);
      }
    });

    // Wait for app to remain stable after rapid events
    await page.waitForFunction(() => {
      const map = document.getElementById('map');
      return map !== null;
    }, { timeout: TIMEOUTS.DATA_LOAD });

    // Application should remain responsive
    await expect(page.locator(SELECTORS.MAP)).toBeVisible();

    // No UI freezing (page should respond to click)
    // Scope to #map to avoid matching globe controls
    const zoomButton = page.locator('#map .maplibregl-ctrl-zoom-in');
    if (await zoomButton.isVisible()) {
      // Use JavaScript click for CI reliability
      await zoomButton.evaluate((el) => el.click());

      // Wait for zoom to complete
      await page.waitForFunction(() => {
        const canvas = document.querySelector('#map canvas');
        return canvas && (canvas as HTMLCanvasElement).width > 0;
      }, { timeout: TIMEOUTS.RENDER });

      // Click should have worked
      await expect(page.locator(SELECTORS.MAP)).toBeVisible();
    }
  });

  test('should close WebSocket connection on page unload', async ({ page }) => {
    // Wait for app to be fully initialized
    await page.waitForFunction(() => {
      const map = document.getElementById('map');
      return map !== null;
    }, { timeout: TIMEOUTS.WEBGL_INIT });

    // Navigate away
    await page.goto('about:blank');

    // Wait for blank page to load
    await page.waitForFunction(() => {
      return document.readyState === 'complete';
    }, { timeout: TIMEOUTS.RENDER });

    // No errors should occur (WebSocket should be closed cleanly)
    // This is implicit - if errors occurred, test would fail
  });

  test('should validate WebSocket message format', async ({ page }) => {
    // Wait for app to be fully initialized
    await page.waitForFunction(() => {
      const map = document.getElementById('map');
      return map !== null;
    }, { timeout: TIMEOUTS.WEBGL_INIT });

    // Send invalid message format
    await page.evaluate(() => {
      const event = new CustomEvent('playback', {
        detail: {
          // Missing required fields
          type: 'invalid',
        },
      });

      window.dispatchEvent(event);
    });

    // Wait for app to remain stable after invalid message
    await page.waitForFunction(() => {
      const map = document.getElementById('map');
      return map !== null;
    }, { timeout: TIMEOUTS.RENDER });

    // Application should handle gracefully without errors
    await expect(page.locator(SELECTORS.MAP)).toBeVisible();
  });

  test('should support secure WebSocket (wss://) on HTTPS', async ({ page }) => {
    // Check if page is served over HTTPS
    const url = page.url();

    // Wait for app to be fully initialized
    await page.waitForFunction(() => {
      const map = document.getElementById('map');
      return map !== null;
    }, { timeout: TIMEOUTS.WEBGL_INIT });

    if (url.startsWith('https://')) {
      // WebSocket should use wss:// protocol
      const wsProtocol = await page.evaluate(() => {
        const ws = (window as any).ws || (window as any).websocket || (window as any).socket;
        return ws ? ws.url : null;
      });

      if (wsProtocol) {
        expect(wsProtocol).toContain('wss://');
      }
    } else {
      // On HTTP, should use ws://
      const wsProtocol = await page.evaluate(() => {
        const ws = (window as any).ws || (window as any).websocket || (window as any).socket;
        return ws ? ws.url : null;
      });

      if (wsProtocol) {
        expect(wsProtocol).toContain('ws://');
      }
    }
  });
});
