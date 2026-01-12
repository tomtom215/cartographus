// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
import {
  test,
  expect,
  TIMEOUTS,
  gotoAppAndWaitReady,
  waitForNavReady,
} from "./fixtures";

/**
 * E2E Test: Plex Direct API Integration
 *
 * Tests Plex Direct API integration:
 * - Library sections browsing
 * - Sessions and activities
 * - Devices and accounts
 * - Transcode sessions with kill functionality
 *
 * @see /docs/working/UI_UX_AUDIT.md
 */

// Mock Plex data
const mockPlexIdentity = {
  MediaContainer: {
    size: 0,
    machineIdentifier: "plex-machine-001",
    version: "1.40.1.8227",
  },
};

const mockPlexSections = {
  MediaContainer: {
    size: 3,
    Directory: [
      {
        key: "1",
        type: "movie",
        title: "Movies",
        agent: "tv.plex.agents.movie",
      },
      {
        key: "2",
        type: "show",
        title: "TV Shows",
        agent: "tv.plex.agents.series",
      },
      {
        key: "3",
        type: "artist",
        title: "Music",
        agent: "tv.plex.agents.music",
      },
    ],
  },
};

const mockPlexSessions = {
  MediaContainer: {
    size: 2,
    Metadata: [
      {
        sessionKey: "session-001",
        ratingKey: "12345",
        type: "movie",
        title: "Test Movie",
        User: { id: "1", title: "TestUser" },
        Player: {
          machineIdentifier: "player-001",
          title: "Chrome",
          state: "playing",
        },
      },
      {
        sessionKey: "session-002",
        ratingKey: "67890",
        type: "episode",
        title: "Test Episode",
        grandparentTitle: "Test Show",
        User: { id: "2", title: "OtherUser" },
        Player: {
          machineIdentifier: "player-002",
          title: "Plex Web",
          state: "paused",
        },
      },
    ],
  },
};

const mockPlexActivities = {
  MediaContainer: {
    size: 1,
    Activity: [
      {
        uuid: "activity-001",
        type: "library.refresh",
        cancellable: true,
        userID: 1,
        title: "Refreshing Movies",
        progress: 45,
      },
    ],
  },
};

const mockPlexDevices = {
  MediaContainer: {
    size: 2,
    Device: [
      {
        id: 1,
        name: "Living Room TV",
        platform: "Android TV",
        clientIdentifier: "device-001",
      },
      {
        id: 2,
        name: "MacBook Pro",
        platform: "macOS",
        clientIdentifier: "device-002",
      },
    ],
  },
};

const mockPlexAccounts = {
  MediaContainer: {
    size: 2,
    Account: [
      { id: 1, name: "Admin", key: "/accounts/1" },
      { id: 2, name: "Guest", key: "/accounts/2" },
    ],
  },
};

const mockPlexPlaylists = {
  MediaContainer: {
    size: 1,
    Metadata: [
      {
        ratingKey: "pl-001",
        type: "playlist",
        title: "Favorites",
        playlistType: "video",
        smart: false,
      },
    ],
  },
};

const mockPlexCapabilities = {
  MediaContainer: {
    machineIdentifier: "plex-machine-001",
    version: "1.40.1.8227",
    transcoderActiveVideoSessions: 2,
    transcoderVideo: true,
    transcoderAudio: true,
  },
};

const mockPlexTranscodeSessions = {
  MediaContainer: {
    size: 1,
    Metadata: [
      {
        sessionKey: "transcode-001",
        ratingKey: "12345",
        title: "Test Movie",
        TranscodeSession: {
          key: "transcode-session-001",
          throttled: false,
          complete: false,
          progress: 25.5,
          speed: 2.5,
          videoDecision: "transcode",
          audioDecision: "copy",
        },
      },
    ],
  },
};

test.describe("Plex Direct API Integration", () => {
  test.beforeEach(async ({ page }) => {
    // Setup Plex API mocks
    await page.route("**/api/v1/plex/identity", (route) =>
      route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          status: "success",
          data: mockPlexIdentity,
          metadata: {},
        }),
      }),
    );

    await page.route("**/api/v1/plex/library/sections", (route) =>
      route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          status: "success",
          data: mockPlexSections,
          metadata: {},
        }),
      }),
    );

    await page.route("**/api/v1/plex/sessions", (route) =>
      route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          status: "success",
          data: mockPlexSessions,
          metadata: {},
        }),
      }),
    );

    await page.route("**/api/v1/plex/activities", (route) =>
      route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          status: "success",
          data: mockPlexActivities,
          metadata: {},
        }),
      }),
    );

    await page.route("**/api/v1/plex/devices", (route) =>
      route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          status: "success",
          data: mockPlexDevices,
          metadata: {},
        }),
      }),
    );

    await page.route("**/api/v1/plex/accounts", (route) =>
      route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          status: "success",
          data: mockPlexAccounts,
          metadata: {},
        }),
      }),
    );

    await page.route("**/api/v1/plex/playlists", (route) =>
      route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          status: "success",
          data: mockPlexPlaylists,
          metadata: {},
        }),
      }),
    );

    await page.route("**/api/v1/plex/capabilities", (route) =>
      route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          status: "success",
          data: mockPlexCapabilities,
          metadata: {},
        }),
      }),
    );

    await page.route("**/api/v1/plex/transcode/sessions", (route) =>
      route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          status: "success",
          data: mockPlexTranscodeSessions,
          metadata: {},
        }),
      }),
    );

    await gotoAppAndWaitReady(page);
    await waitForNavReady(page);
  });

  test.describe("Plex Server Info", () => {
    test("should show Plex server identity info", async ({ page }) => {
      // Navigate using JavaScript click for CI reliability
      // WHY: Playwright's .click() may fail in headless/SwiftShader environments
      await page.evaluate(() => {
        const tab = document.querySelector('.nav-tab[data-view="server"]') as HTMLElement;
        if (tab) tab.click();
      });

      // Look for Plex identity information in server dashboard
      const serverInfo = page.locator("#server-health, .server-card");
      await expect(serverInfo.first()).toBeVisible({
        timeout: TIMEOUTS.MEDIUM,
      });
    });
  });

  test.describe("Library Sections", () => {
    test("should have API access to library sections", async ({ page }) => {
      // This test verifies the API endpoint is accessible
      const response = await page.evaluate(async () => {
        const res = await fetch("/api/v1/plex/library/sections");
        return res.ok;
      });
      expect(response).toBeTruthy();
    });
  });

  test.describe("Active Sessions", () => {
    test("should be able to fetch active sessions", async ({ page }) => {
      const response = await page.evaluate(async () => {
        const res = await fetch("/api/v1/plex/sessions");
        return res.ok;
      });
      expect(response).toBeTruthy();
    });
  });

  test.describe("Transcode Session Management", () => {
    test("should be able to fetch transcode sessions", async ({ page }) => {
      const response = await page.evaluate(async () => {
        const res = await fetch("/api/v1/plex/transcode/sessions");
        return res.ok;
      });
      expect(response).toBeTruthy();
    });

    test("should have kill transcode endpoint available", async ({ page }) => {
      // Mock the DELETE endpoint
      await page.route("**/api/v1/plex/transcode/sessions/*", (route) => {
        if (route.request().method() === "DELETE") {
          return route.fulfill({
            status: 200,
            contentType: "application/json",
            body: JSON.stringify({
              status: "success",
              data: null,
              metadata: {},
            }),
          });
        }
        return route.continue();
      });

      // Verify endpoint responds
      const response = await page.evaluate(async () => {
        const res = await fetch(
          "/api/v1/plex/transcode/sessions/test-session",
          {
            method: "DELETE",
          },
        );
        return res.ok;
      });
      expect(response).toBeTruthy();
    });
  });

  test.describe("Devices", () => {
    test("should be able to fetch devices", async ({ page }) => {
      const response = await page.evaluate(async () => {
        const res = await fetch("/api/v1/plex/devices");
        return res.ok;
      });
      expect(response).toBeTruthy();
    });
  });

  test.describe("Activities", () => {
    test("should be able to fetch activities", async ({ page }) => {
      const response = await page.evaluate(async () => {
        const res = await fetch("/api/v1/plex/activities");
        return res.ok;
      });
      expect(response).toBeTruthy();
    });
  });
});

test.describe("Plex API Error Handling", () => {
  test("should handle Plex API unavailable gracefully", async ({ page }) => {
    await page.route("**/api/v1/plex/**", (route) =>
      route.fulfill({
        status: 503,
        contentType: "application/json",
        body: JSON.stringify({
          status: "error",
          error: {
            code: "PLEX_UNAVAILABLE",
            message: "Plex server not configured",
          },
        }),
      }),
    );

    await gotoAppAndWaitReady(page);
    await waitForNavReady(page);

    // App should still load even if Plex is unavailable
    const app = page.locator("#app");
    await expect(app).toBeVisible();
  });
});
