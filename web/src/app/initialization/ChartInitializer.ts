// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Chart Managers Initializer
 *
 * Initializes chart-related managers: ChartTooltip, ChartDrillDown,
 * ChartExport, ChartMaximize, ChartTimelineAnimation, DateRangeBrush.
 */

import type { ToastManager } from '../../lib/toast';
import type { FilterManager } from '../../lib/filters';
import type { API } from '../../lib/api';
import { ChartTooltipManager } from '../ChartTooltipManager';
import { ChartDrillDownManager } from '../ChartDrillDownManager';
import { ChartExportManager } from '../ChartExportManager';
import { ChartMaximizeManager } from '../ChartMaximizeManager';
import { ChartTimelineAnimationManager } from '../ChartTimelineAnimationManager';
import { DateRangeBrushManager } from '../DateRangeBrushManager';

export interface ChartManagers {
    chartTooltipManager: ChartTooltipManager;
    chartDrillDownManager: ChartDrillDownManager;
    chartExportManager: ChartExportManager;
    chartMaximizeManager: ChartMaximizeManager;
    chartTimelineAnimationManager: ChartTimelineAnimationManager;
    dateRangeBrushManager: DateRangeBrushManager | null;
}

export interface ChartInitConfig {
    api: API;
    toastManager: ToastManager;
    filterManager: FilterManager;
}

/**
 * Initialize chart-related managers
 */
export async function initializeChartManagers(config: ChartInitConfig): Promise<ChartManagers> {
    // Chart tooltip manager
    const chartTooltipManager = new ChartTooltipManager();
    // Tooltip initialization is deferred until after charts are rendered

    // Chart drill-down manager
    const chartDrillDownManager = new ChartDrillDownManager();
    chartDrillDownManager.setToastManager(config.toastManager);
    chartDrillDownManager.setFilterManager(config.filterManager);
    chartDrillDownManager.init();

    // Chart export manager
    const chartExportManager = new ChartExportManager();
    chartExportManager.setToastManager(config.toastManager);
    chartExportManager.init();

    // Chart maximize manager
    const chartMaximizeManager = new ChartMaximizeManager();
    chartMaximizeManager.init();

    // Chart timeline animation manager
    const chartTimelineAnimationManager = new ChartTimelineAnimationManager(config.api);
    await chartTimelineAnimationManager.init();

    // Date range brush manager (requires chart manager reference)
    const dateRangeBrushManager = new DateRangeBrushManager(
        (start: string, end: string) => {
            config.filterManager.setDateRange(start, end);
        }
    );

    return {
        chartTooltipManager,
        chartDrillDownManager,
        chartExportManager,
        chartMaximizeManager,
        chartTimelineAnimationManager,
        dateRangeBrushManager
    };
}
