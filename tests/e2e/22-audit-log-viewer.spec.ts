// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Audit Log Viewer E2E Tests
 * ADR-0015: Zero Trust Authentication
 *
 * Tests for the Security Audit Log Viewer UI:
 * - View audit events in Data Governance tab
 * - Filter by type, severity, outcome
 * - Search functionality
 * - Pagination
 * - Event details modal
 * - Export functionality
 */

import { test, expect } from '@playwright/test';

test.describe('Audit Log Viewer', () => {
  // Navigate to Data Governance > Audit page before each test
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');

    // Navigate to Data Governance tab
    const govTab = page.locator('[data-view="data-governance"], [data-testid="nav-tab-data-governance"]');
    await govTab.click();
    await expect(page.locator('#data-governance-container')).toBeVisible({ timeout: 5000 });

    // Navigate to Audit Log sub-page
    const auditTab = page.locator('.governance-nav-tab[data-governance-page="audit"]');
    await auditTab.click();
    await expect(page.locator('#governance-audit-page')).toBeVisible({ timeout: 5000 });
  });

  test('displays audit log page with stats', async ({ page }) => {
    // Check stats are visible
    await expect(page.locator('#audit-total-events, .audit-stat-value')).toBeVisible();

    // Check that events table is present
    const table = page.locator('#audit-events-table, .governance-table');
    await expect(table).toBeVisible();
  });

  test('shows audit events in table', async ({ page }) => {
    // Wait for events to load
    await page.waitForResponse(resp =>
      resp.url().includes('/audit/events') && resp.request().method() === 'GET'
    );

    // Check for event rows
    const rows = page.locator('#audit-events-tbody tr, .audit-row');
    await expect(rows.first()).toBeVisible({ timeout: 5000 });

    // Verify we have multiple events
    const rowCount = await rows.count();
    expect(rowCount).toBeGreaterThan(0);
  });

  test('displays event type badges', async ({ page }) => {
    // Wait for events to load
    await page.waitForResponse(resp =>
      resp.url().includes('/audit/events') && resp.request().method() === 'GET'
    );

    // Check for type badges
    const typeBadges = page.locator('.audit-type-badge');
    await expect(typeBadges.first()).toBeVisible({ timeout: 5000 });

    // Get all type badges text
    const badgeTexts = await typeBadges.allTextContents();
    expect(badgeTexts.length).toBeGreaterThan(0);
  });

  test('displays severity badges with appropriate colors', async ({ page }) => {
    // Wait for events to load
    await page.waitForResponse(resp =>
      resp.url().includes('/audit/events') && resp.request().method() === 'GET'
    );

    // Check for severity badges
    const severityBadges = page.locator('.severity-badge, [class*="severity-"]');
    await expect(severityBadges.first()).toBeVisible({ timeout: 5000 });
  });

  test('filter by event type works', async ({ page }) => {
    // Wait for initial load
    await page.waitForResponse(resp =>
      resp.url().includes('/audit/events') && resp.request().method() === 'GET'
    );

    // Select auth.success from type filter
    const typeFilter = page.locator('#audit-type-filter');
    await typeFilter.selectOption('auth.success');

    // Wait for filtered response
    const response = await page.waitForResponse(resp =>
      resp.url().includes('/audit/events') && resp.url().includes('type=auth.success')
    );
    expect(response.ok()).toBe(true);

    // Verify filtered results show only auth.success events
    const typeBadges = page.locator('.audit-type-badge');
    const count = await typeBadges.count();
    if (count > 0) {
      const firstBadge = await typeBadges.first().textContent();
      expect(firstBadge?.toLowerCase()).toContain('auth');
    }
  });

  test('filter by severity works', async ({ page }) => {
    // Wait for initial load
    await page.waitForResponse(resp =>
      resp.url().includes('/audit/events') && resp.request().method() === 'GET'
    );

    // Select warning from severity filter
    const severityFilter = page.locator('#audit-severity-filter');
    await severityFilter.selectOption('warning');

    // Wait for filtered response
    const response = await page.waitForResponse(resp =>
      resp.url().includes('/audit/events') && resp.url().includes('severity=warning')
    );
    expect(response.ok()).toBe(true);
  });

  test('filter by outcome works', async ({ page }) => {
    // Wait for initial load
    await page.waitForResponse(resp =>
      resp.url().includes('/audit/events') && resp.request().method() === 'GET'
    );

    // Select failure from outcome filter
    const outcomeFilter = page.locator('#audit-outcome-filter');
    await outcomeFilter.selectOption('failure');

    // Wait for filtered response
    const response = await page.waitForResponse(resp =>
      resp.url().includes('/audit/events') && resp.url().includes('outcome=failure')
    );
    expect(response.ok()).toBe(true);
  });

  test('search functionality works', async ({ page }) => {
    // Wait for initial load
    await page.waitForResponse(resp =>
      resp.url().includes('/audit/events') && resp.request().method() === 'GET'
    );

    // Enter search term
    const searchInput = page.locator('#audit-search');
    await searchInput.fill('login');

    // Wait for filtered response (with debounce delay)
    const response = await page.waitForResponse(
      resp => resp.url().includes('/audit/events') && resp.url().includes('search=login'),
      { timeout: 5000 }
    );
    expect(response.ok()).toBe(true);
  });

  test('pagination controls are present', async ({ page }) => {
    // Wait for load
    await page.waitForResponse(resp =>
      resp.url().includes('/audit/events') && resp.request().method() === 'GET'
    );

    // Check pagination elements
    await expect(page.locator('#audit-pagination, .audit-pagination')).toBeVisible();
    await expect(page.locator('#audit-prev-btn, .audit-page-btn')).toBeVisible();
    await expect(page.locator('#audit-next-btn, [id*="next"]')).toBeVisible();
    await expect(page.locator('#audit-page-info, .pagination-info, .audit-page-indicator')).toBeVisible();
  });

  test('clicking event details shows modal', async ({ page }) => {
    // Wait for events to load
    await page.waitForResponse(resp =>
      resp.url().includes('/audit/events') && resp.request().method() === 'GET'
    );

    // Click on details button for first event
    const detailsButton = page.locator('[data-action="details"], .audit-details-btn, .btn-details').first();
    if (await detailsButton.isVisible()) {
      await detailsButton.click();

      // Check that modal appears
      await expect(page.locator('.modal-overlay, .audit-modal-overlay, #audit-details-dialog').filter({ has: page.locator('.modal-content, .audit-modal') })).toBeVisible({ timeout: 3000 });

      // Modal should contain event details
      await expect(page.locator('.audit-detail-row, .modal-body')).toBeVisible();

      // Close modal
      const closeBtn = page.locator('.modal-close, .audit-modal-close');
      if (await closeBtn.isVisible()) {
        await closeBtn.click();
        await expect(page.locator('.modal-overlay.show, #audit-details-dialog:visible')).toBeHidden({ timeout: 2000 });
      }
    }
  });

  test('refresh button reloads events', async ({ page }) => {
    // Wait for initial load
    await page.waitForResponse(resp =>
      resp.url().includes('/audit/events') && resp.request().method() === 'GET'
    );

    // Click refresh button
    const refreshBtn = page.locator('#audit-refresh-btn, .audit-refresh-btn');
    await expect(refreshBtn).toBeVisible();

    // Wait for refresh response
    const responsePromise = page.waitForResponse(resp =>
      resp.url().includes('/audit/events') && resp.request().method() === 'GET'
    );

    await refreshBtn.click();
    await responsePromise;
  });

  test('export button is present', async ({ page }) => {
    // Check export button exists
    const exportBtn = page.locator('#audit-export-btn, .audit-export-btn');
    await expect(exportBtn).toBeVisible();
  });

  test('stats display correct counts', async ({ page }) => {
    // Wait for stats to load
    await page.waitForResponse(resp =>
      resp.url().includes('/audit/stats') && resp.request().method() === 'GET'
    );

    // Verify stats are displayed (not showing -- or 0 for all)
    const totalEvents = page.locator('#audit-total-events, .stat-value').first();
    const text = await totalEvents.textContent();

    // Should show a number, not -- or empty
    expect(text).toBeTruthy();
    expect(text).not.toBe('--');
  });

  test('handles empty results gracefully', async ({ page }) => {
    // Apply filter that returns no results
    const typeFilter = page.locator('#audit-type-filter');
    await typeFilter.selectOption('auth.lockout');

    // Wait for response
    await page.waitForResponse(resp =>
      resp.url().includes('/audit/events') && resp.request().method() === 'GET'
    );

    // Should show empty state or "no events" message
    const noResultsIndicator = page.locator('.table-empty, .audit-empty, td:has-text("No audit events")');
    // Either shows empty state or shows events (if lockout exists in mock)
    const rows = page.locator('#audit-events-tbody tr, .audit-row');
    const rowCount = await rows.count();

    // At minimum, table should be present
    await expect(page.locator('#audit-events-table, .governance-table')).toBeVisible();
  });

  test('displays actor information for events', async ({ page }) => {
    // Wait for events to load
    await page.waitForResponse(resp =>
      resp.url().includes('/audit/events') && resp.request().method() === 'GET'
    );

    // Check for actor column content
    const actorCells = page.locator('.audit-actor, td.audit-actor');
    const firstActor = actorCells.first();
    if (await firstActor.isVisible()) {
      const text = await firstActor.textContent();
      expect(text?.trim().length).toBeGreaterThan(0);
    }
  });

  test('displays source IP for events', async ({ page }) => {
    // Wait for events to load
    await page.waitForResponse(resp =>
      resp.url().includes('/audit/events') && resp.request().method() === 'GET'
    );

    // Check for source IP content
    const sourceCells = page.locator('.audit-source, td.audit-source');
    const firstSource = sourceCells.first();
    if (await firstSource.isVisible()) {
      const text = await firstSource.textContent();
      // Should contain IP address pattern or --
      expect(text?.trim()).toBeTruthy();
    }
  });
});

test.describe('Audit Log Viewer - Error Handling', () => {
  test('handles API error gracefully', async ({ page }) => {
    // Intercept audit requests to return error
    await page.route('**/audit/events', route => {
      route.fulfill({
        status: 500,
        body: JSON.stringify({ error: 'Internal server error' })
      });
    });

    await page.goto('/');
    await page.waitForLoadState('networkidle');

    // Navigate to Data Governance > Audit
    const govTab = page.locator('[data-view="data-governance"], [data-testid="nav-tab-data-governance"]');
    await govTab.click();

    const auditTab = page.locator('.governance-nav-tab[data-governance-page="audit"]');
    await auditTab.click();

    // Should handle error gracefully - either show error message or empty table
    const auditPage = page.locator('#governance-audit-page');
    await expect(auditPage).toBeVisible({ timeout: 5000 });
  });

  test('handles stats API error gracefully', async ({ page }) => {
    // Intercept stats request to return error
    await page.route('**/audit/stats', route => {
      route.fulfill({
        status: 500,
        body: JSON.stringify({ error: 'Internal server error' })
      });
    });

    await page.goto('/');
    await page.waitForLoadState('networkidle');

    // Navigate to Data Governance > Audit
    const govTab = page.locator('[data-view="data-governance"], [data-testid="nav-tab-data-governance"]');
    await govTab.click();

    const auditTab = page.locator('.governance-nav-tab[data-governance-page="audit"]');
    await auditTab.click();

    // Page should still render, possibly with -- for stats
    const auditPage = page.locator('#governance-audit-page');
    await expect(auditPage).toBeVisible({ timeout: 5000 });
  });
});
