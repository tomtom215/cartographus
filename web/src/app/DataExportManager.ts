// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * DataExportManager - Handles data export functionality
 *
 * Manages:
 * - CSV export of playback events with applied filters
 * - GeoJSON export of locations with applied filters
 * - GeoParquet export for GIS analysis
 * - Download trigger and filename generation
 * - Success notifications via ToastManager
 */

import type { FilterManager } from '../lib/filters';
import type { ToastManager } from '../lib/toast';

export class DataExportManager {
    // AbortController for clean event listener removal
    private abortController: AbortController | null = null;

    constructor(
        private filterManager: FilterManager | null,
        private toastManager: ToastManager | null
    ) {
        this.setupExportListeners();
    }

    /**
     * Setup export button event listeners with AbortController for clean removal
     */
    private setupExportListeners(): void {
        // Create AbortController for cleanup
        this.abortController = new AbortController();
        const signal = this.abortController.signal;

        const btnExportCSV = document.getElementById('btn-export-csv') as HTMLButtonElement;
        if (btnExportCSV) {
            btnExportCSV.addEventListener('click', () => this.exportPlaybacksCSV(), { signal });
        }

        const btnExportGeoJSON = document.getElementById('btn-export-geojson') as HTMLButtonElement;
        if (btnExportGeoJSON) {
            btnExportGeoJSON.addEventListener('click', () => this.exportLocationsGeoJSON(), { signal });
        }

        // GeoParquet export
        const btnExportGeoParquet = document.getElementById('btn-export-geoparquet') as HTMLButtonElement;
        if (btnExportGeoParquet) {
            btnExportGeoParquet.addEventListener('click', () => this.exportLocationsGeoParquet(), { signal });
        }
    }

    /**
     * Export playback events to CSV with current filters
     */
    exportPlaybacksCSV(): void {
        if (!this.filterManager) return;

        const filter = this.filterManager.buildFilter();
        const params = new URLSearchParams();

        if (filter.days) params.set('days', filter.days.toString());
        if (filter.start_date) params.set('start_date', filter.start_date);
        if (filter.end_date) params.set('end_date', filter.end_date);
        if (filter.users && filter.users.length > 0) params.set('users', filter.users.join(','));
        if (filter.media_types && filter.media_types.length > 0) params.set('media_types', filter.media_types.join(','));

        const url = `/api/v1/export/playbacks/csv?${params.toString()}`;

        // Generate filename with timestamp
        const timestamp = new Date().toISOString().replace(/[:.]/g, '-').slice(0, 19);
        const filename = `playbacks-${timestamp}.csv`;

        // Trigger download using anchor tag - reliable cross-browser approach
        // that properly triggers browser download events (compatible with Playwright)
        const link = document.createElement('a');
        link.href = url;
        link.download = filename; // Set explicit filename for Playwright detection
        link.style.display = 'none';
        document.body.appendChild(link);
        link.click();
        // Delay removal to ensure download starts (Playwright needs this)
        setTimeout(() => document.body.removeChild(link), 100);

        if (this.toastManager) {
            this.toastManager.success('CSV export started', 'Export', 3000);
        }
    }

    /**
     * Export location data to GeoJSON with current filters
     */
    exportLocationsGeoJSON(): void {
        if (!this.filterManager) return;

        const filter = this.filterManager.buildFilter();
        const params = new URLSearchParams();

        if (filter.days) params.set('days', filter.days.toString());
        if (filter.start_date) params.set('start_date', filter.start_date);
        if (filter.end_date) params.set('end_date', filter.end_date);
        if (filter.users && filter.users.length > 0) params.set('users', filter.users.join(','));
        if (filter.media_types && filter.media_types.length > 0) params.set('media_types', filter.media_types.join(','));

        const url = `/api/v1/export/locations/geojson?${params.toString()}`;

        // Generate filename with timestamp
        const timestamp = new Date().toISOString().replace(/[:.]/g, '-').slice(0, 19);
        const filename = `locations-${timestamp}.geojson`;

        // Trigger download using anchor tag - reliable cross-browser approach
        // that properly triggers browser download events (compatible with Playwright)
        const link = document.createElement('a');
        link.href = url;
        link.download = filename; // Set explicit filename for Playwright detection
        link.style.display = 'none';
        document.body.appendChild(link);
        link.click();
        // Delay removal to ensure download starts (Playwright needs this)
        setTimeout(() => document.body.removeChild(link), 100);

        if (this.toastManager) {
            this.toastManager.success('GeoJSON export started', 'Export', 3000);
        }
    }

    /**
     * Export location data to GeoParquet with current filters
     * GeoParquet is optimized for GIS tools (QGIS, PostGIS, etc.)
     */
    exportLocationsGeoParquet(): void {
        if (!this.filterManager) return;

        const filter = this.filterManager.buildFilter();
        const params = new URLSearchParams();

        if (filter.days) params.set('days', filter.days.toString());
        if (filter.start_date) params.set('start_date', filter.start_date);
        if (filter.end_date) params.set('end_date', filter.end_date);
        if (filter.users && filter.users.length > 0) params.set('users', filter.users.join(','));
        if (filter.media_types && filter.media_types.length > 0) params.set('media_types', filter.media_types.join(','));

        const url = `/api/v1/export/geoparquet?${params.toString()}`;

        // Generate filename with timestamp
        const timestamp = new Date().toISOString().replace(/[:.]/g, '-').slice(0, 19);
        const filename = `locations-${timestamp}.parquet`;

        // Trigger download using anchor tag
        const link = document.createElement('a');
        link.href = url;
        link.download = filename;
        link.style.display = 'none';
        document.body.appendChild(link);
        link.click();
        setTimeout(() => document.body.removeChild(link), 100);

        if (this.toastManager) {
            this.toastManager.success('GeoParquet export started', 'Export', 3000);
        }
    }

    /**
     * Clean up event listeners
     */
    destroy(): void {
        if (this.abortController) {
            this.abortController.abort();
            this.abortController = null;
        }
    }
}
