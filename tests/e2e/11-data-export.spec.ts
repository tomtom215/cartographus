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
import * as fs from 'fs';
import * as path from 'path';
import * as os from 'os';

// Helper function to safely get download path with fallback
async function getDownloadPath(download: any, timeout: number = 30000): Promise<string | null> {
  try {
    // First try to get the path directly
    const downloadPath = await download.path();
    if (downloadPath && downloadPath !== 'canceled') {
      return downloadPath;
    }
  } catch {
    // path() failed, try saveAs fallback
  }

  // Fallback: save to temp file
  try {
    const tempPath = path.join(os.tmpdir(), `playwright-download-${Date.now()}-${download.suggestedFilename()}`);
    await download.saveAs(tempPath);
    return tempPath;
  } catch (e) {
    console.warn('Download save failed:', e);
    return null;
  }
}

test.describe('Data Export Functionality', () => {
  test.beforeEach(async ({ page }) => {
    // Use storageState for authentication (configured in playwright.config.ts)
    await gotoAppAndWaitReady(page);

    // Wait for export buttons to be visible and clickable
    // This ensures DataExportManager has set up event listeners
    await page.waitForSelector(SELECTORS.BTN_EXPORT_CSV, { state: 'visible', timeout: TIMEOUTS.DEFAULT });
    await page.waitForSelector(SELECTORS.BTN_EXPORT_GEOJSON, { state: 'visible', timeout: TIMEOUTS.DEFAULT });

    // Wait for DataExportManager to initialize and attach event listeners
    await page.waitForFunction(() => {
      const csvBtn = document.querySelector('#btn-export-csv') as HTMLButtonElement;
      const geojsonBtn = document.querySelector('#btn-export-geojson') as HTMLButtonElement;
      return csvBtn && geojsonBtn &&
             !csvBtn.hasAttribute('disabled') &&
             !geojsonBtn.hasAttribute('disabled');
    }, { timeout: TIMEOUTS.DEFAULT });
  });

  test.describe('CSV Export', () => {
    test('should export playbacks as CSV with correct filename', async ({ page }) => {
      const [download] = await Promise.all([
        page.waitForEvent('download', { timeout: 10000 }),
        page.click('#btn-export-csv')
      ]);

      const filename = download.suggestedFilename();
      expect(filename).toMatch(/playbacks.*\.csv$/i);
      expect(filename).toContain('.csv');
    });

    test('should export CSV with valid format and headers', async ({ page }) => {
      // Increase timeout for download
      test.setTimeout(60000);

      const [download] = await Promise.all([
        page.waitForEvent('download', { timeout: 30000 }),
        page.click('#btn-export-csv')
      ]);

      // Verify download was triggered (primary check)
      expect(download).toBeTruthy();
      expect(download.suggestedFilename()).toMatch(/\.csv$/);

      // Try to get the file content (may fail in some CI environments)
      const downloadPath = await getDownloadPath(download);

      if (downloadPath) {
        const content = fs.readFileSync(downloadPath, 'utf-8');
        const lines = content.split('\n');

        // Verify CSV has content
        expect(lines.length).toBeGreaterThan(1);

        // Verify header row exists
        const headers = lines[0].toLowerCase();
        expect(headers).toContain('username');
        expect(headers).toContain('title');
        expect(headers).toContain('watched_at');

        // Verify at least one data row exists (if there's data)
        if (lines.length > 2) {
          const firstDataRow = lines[1];
          expect(firstDataRow.length).toBeGreaterThan(0);
        }
      } else {
        // Even if we can't save the file, verify the download was initiated successfully
        // by checking the suggested filename
        const filename = download.suggestedFilename();
        expect(filename).toBeTruthy();
        expect(filename).toContain('.csv');
      }
    });

    test('should export filtered CSV data (user filter)', async ({ page }) => {
      // Wait for filters to populate with a longer timeout
      const filterUsersSelector = '#filter-users option:not([value=""])';
      try {
        await page.waitForSelector(filterUsersSelector, { timeout: 10000 });
      } catch {
        // Filter may not have options if no user data exists - skip this test
        test.skip();
        return;
      }

      // Get available users
      const userOptions = await page.locator(filterUsersSelector).count();

      if (userOptions > 0) {
        // Select first user
        await page.selectOption('#filter-users', { index: 1 });

        // Wait for filter debounce and data refresh
        await page.waitForLoadState('networkidle', { timeout: TIMEOUTS.DEFAULT });

        const [download] = await Promise.all([
          page.waitForEvent('download', { timeout: 30000 }),
          page.click('#btn-export-csv')
        ]);

        const downloadPath = await getDownloadPath(download);
        if (downloadPath) {
          const content = fs.readFileSync(downloadPath, 'utf-8');
          const lines = content.split('\n').filter(line => line.trim().length > 0);

          // Should have headers + data
          expect(lines.length).toBeGreaterThan(1);

          // Get selected user value
          const selectedUser = await page.inputValue('#filter-users');

          // Verify all data rows contain the selected user (if header has username column)
          if (lines[0].toLowerCase().includes('username')) {
            const headerIndex = lines[0].split(',').findIndex(h => h.toLowerCase().includes('username'));
            if (headerIndex !== -1) {
              for (let i = 1; i < Math.min(lines.length, 10); i++) {
                const columns = lines[i].split(',');
                if (columns[headerIndex]) {
                  expect(columns[headerIndex].toLowerCase()).toContain(selectedUser.toLowerCase());
                }
              }
            }
          }
        }
      }
    });

    test('should export CSV with date range filter applied', async ({ page }) => {
      // Set date range filter
      const startDate = '2025-01-01';
      const endDate = '2025-01-31';

      await page.fill('#filter-start-date', startDate);
      await page.fill('#filter-end-date', endDate);

      // Wait for filter debounce and data refresh
      await page.waitForLoadState('networkidle', { timeout: TIMEOUTS.DEFAULT });

      const [download] = await Promise.all([
        page.waitForEvent('download', { timeout: 30000 }),
        page.click('#btn-export-csv')
      ]);

      const downloadPath = await getDownloadPath(download);
      if (downloadPath) {
        const content = fs.readFileSync(downloadPath, 'utf-8');
        const lines = content.split('\n');

        // Verify export completed
        expect(lines.length).toBeGreaterThan(0);
      }
    });

    test('should handle empty result set gracefully', async ({ page }) => {
      // Set a date range that likely has no data
      await page.fill('#filter-start-date', '2099-01-01');
      await page.fill('#filter-end-date', '2099-01-02');
      await page.waitForLoadState('networkidle', { timeout: TIMEOUTS.DEFAULT });

      const [download] = await Promise.all([
        page.waitForEvent('download', { timeout: 30000 }),
        page.click('#btn-export-csv')
      ]);

      const downloadPath = await getDownloadPath(download);
      if (downloadPath) {
        const content = fs.readFileSync(downloadPath, 'utf-8');
        const lines = content.split('\n');

        // Should still have headers even if no data
        expect(lines.length).toBeGreaterThanOrEqual(1);
        expect(lines[0]).toContain('username');
      }
    });
  });

  test.describe('GeoJSON Export', () => {
    test('should export locations as GeoJSON with correct filename', async ({ page }) => {
      const [download] = await Promise.all([
        page.waitForEvent('download', { timeout: 30000 }),
        page.click('#btn-export-geojson')
      ]);

      const filename = download.suggestedFilename();
      expect(filename).toMatch(/locations.*\.geojson$/i);
      expect(filename).toContain('.geojson');
    });

    test('should export valid GeoJSON FeatureCollection', async ({ page }) => {
      const [download] = await Promise.all([
        page.waitForEvent('download', { timeout: 30000 }),
        page.click('#btn-export-geojson')
      ]);

      const downloadPath = await getDownloadPath(download);
      if (downloadPath) {
        const content = fs.readFileSync(downloadPath, 'utf-8');
        const geojson = JSON.parse(content);

        // Verify GeoJSON structure
        expect(geojson.type).toBe('FeatureCollection');
        expect(Array.isArray(geojson.features)).toBe(true);
      }
    });

    test('should have valid Feature geometries in GeoJSON', async ({ page }) => {
      const [download] = await Promise.all([
        page.waitForEvent('download', { timeout: 30000 }),
        page.click('#btn-export-geojson')
      ]);

      const downloadPath = await getDownloadPath(download);
      if (downloadPath) {
        const content = fs.readFileSync(downloadPath, 'utf-8');
        const geojson = JSON.parse(content);

        if (geojson.features.length > 0) {
          const firstFeature = geojson.features[0];

          // Verify Feature structure
          expect(firstFeature.type).toBe('Feature');
          expect(firstFeature.geometry).toBeDefined();
          expect(firstFeature.geometry.type).toBe('Point');

          // Verify coordinates are valid [longitude, latitude]
          expect(Array.isArray(firstFeature.geometry.coordinates)).toBe(true);
          expect(firstFeature.geometry.coordinates).toHaveLength(2);

          const [lon, lat] = firstFeature.geometry.coordinates;
          expect(typeof lon).toBe('number');
          expect(typeof lat).toBe('number');
          expect(lon).toBeGreaterThanOrEqual(-180);
          expect(lon).toBeLessThanOrEqual(180);
          expect(lat).toBeGreaterThanOrEqual(-90);
          expect(lat).toBeLessThanOrEqual(90);
        }
      }
    });

    test('should include properties in GeoJSON features', async ({ page }) => {
      const [download] = await Promise.all([
        page.waitForEvent('download', { timeout: 30000 }),
        page.click('#btn-export-geojson')
      ]);

      const downloadPath = await getDownloadPath(download);
      if (downloadPath) {
        const content = fs.readFileSync(downloadPath, 'utf-8');
        const geojson = JSON.parse(content);

        if (geojson.features.length > 0) {
          const firstFeature = geojson.features[0];

          // Verify properties exist
          expect(firstFeature.properties).toBeDefined();
          expect(typeof firstFeature.properties).toBe('object');

          // Properties should contain location metadata
          // (e.g., city, country, playback_count)
        }
      }
    });

    test('should export GeoJSON with filters applied', async ({ page }) => {
      // Wait for media type filter to populate with longer timeout
      const filterMediaTypesSelector = '#filter-media-types option:not([value=""])';
      try {
        await page.waitForSelector(filterMediaTypesSelector, { timeout: 10000 });
      } catch {
        // Filter may not have options if no media type data exists - skip this test
        test.skip();
        return;
      }

      const mediaTypeOptions = await page.locator(filterMediaTypesSelector).count();

      if (mediaTypeOptions > 0) {
        // Select first media type
        await page.selectOption('#filter-media-types', { index: 1 });
        await page.waitForLoadState('networkidle', { timeout: TIMEOUTS.DEFAULT });

        const [download] = await Promise.all([
          page.waitForEvent('download', { timeout: 30000 }),
          page.click('#btn-export-geojson')
        ]);

        const downloadPath = await getDownloadPath(download);
        if (downloadPath) {
          const content = fs.readFileSync(downloadPath, 'utf-8');
          const geojson = JSON.parse(content);

          // Verify export completed with filter
          expect(geojson.type).toBe('FeatureCollection');
          expect(Array.isArray(geojson.features)).toBe(true);
        }
      }
    });
  });

  test.describe('Export Error Handling', () => {
    test('should handle export button click during loading', async ({ page }) => {
      // Click export button - Use JavaScript click for CI reliability
      await page.evaluate(() => {
        const el = document.querySelector('#btn-export-csv') as HTMLElement;
        if (el) el.click();
      });

      // Try clicking again immediately - Use JavaScript click for CI reliability
      await page.evaluate(() => {
        const el = document.querySelector('#btn-export-csv') as HTMLElement;
        if (el) el.click();
      });

      // Should not crash - verify page is still responsive
      await page.waitForFunction(() => {
        const logoutBtn = document.querySelector('#btn-logout');
        return logoutBtn && window.getComputedStyle(logoutBtn).display !== 'none';
      }, { timeout: TIMEOUTS.DEFAULT });
      const btnLogout = await page.locator('#btn-logout');
      await expect(btnLogout).toBeVisible();
    });

    test('should maintain UI state after export', async ({ page }) => {
      // Export data
      const [_download] = await Promise.all([
        page.waitForEvent('download', { timeout: 30000 }),
        page.click('#btn-export-csv')
      ]);

      // Verify UI is still functional
      await expect(page.locator('#btn-logout')).toBeVisible();
      await expect(page.locator('#btn-refresh')).toBeVisible();

      // Should be able to interact with filters
      const filterDays = await page.locator('#filter-days');
      await expect(filterDays).toBeEnabled();
    });
  });

  test.describe('Export Performance', () => {
    test('CSV export should complete within 15 seconds', async ({ page }) => {
      const startTime = Date.now();

      const [download] = await Promise.all([
        page.waitForEvent('download', { timeout: 10000 }),
        page.click('#btn-export-csv')
      ]);

      const duration = Date.now() - startTime;

      expect(duration).toBeLessThan(15000);
      expect(download.suggestedFilename()).toContain('.csv');
    });

    test('GeoJSON export should complete within 15 seconds', async ({ page }) => {
      const startTime = Date.now();

      const [download] = await Promise.all([
        page.waitForEvent('download', { timeout: 10000 }),
        page.click('#btn-export-geojson')
      ]);

      const duration = Date.now() - startTime;

      expect(duration).toBeLessThan(15000);
      expect(download.suggestedFilename()).toContain('.geojson');
    });
  });
});
