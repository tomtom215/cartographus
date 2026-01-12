// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * ChartExportManager - Enhanced chart export with multiple formats
 *
 * Features:
 * - Export to PNG (high/low quality)
 * - Export to SVG (vector format)
 * - Export data to CSV
 * - Bulk export all visible charts
 * - Format selection dropdown
 * - Progress indication for bulk exports
 */

import type * as echarts from 'echarts';
import { createLogger } from '../lib/logger';
import type { ToastManager } from '../lib/toast';

const logger = createLogger('ChartExportManager');

/**
 * Export format types
 */
export type ExportFormat = 'png-high' | 'png-low' | 'svg' | 'csv';

/**
 * ECharts series data item for CSV export
 */
interface EChartsSeriesDataItem {
  name?: string;
  value?: number | string | number[];
}

/**
 * ECharts series option for CSV export
 */
interface EChartsSeriesOption {
  name?: string;
  type?: string;
  data?: (number | string | EChartsSeriesDataItem)[];
}

/**
 * ECharts option subset for CSV export
 */
interface EChartsExportOption {
  xAxis?: { data?: string[] } | { data?: string[] }[];
  series?: EChartsSeriesOption[];
}

/**
 * Export options interface
 */
interface ExportOptions {
  format: ExportFormat;
  chartId: string;
  filename?: string;
}

export class ChartExportManager {
  private charts: Map<string, echarts.ECharts> = new Map();
  private toastManager: ToastManager | null = null;
  private exportMenus: Map<string, HTMLElement> = new Map();

  constructor() {
    this.setupGlobalExportButton();
  }

  /**
   * Set toast manager for notifications
   */
  setToastManager(toastManager: ToastManager): void {
    this.toastManager = toastManager;
  }

  /**
   * Register a chart for enhanced export
   */
  registerChart(chartId: string, chart: echarts.ECharts): void {
    this.charts.set(chartId, chart);
    this.createExportMenu(chartId);
  }

  /**
   * Create export menu dropdown for a chart
   */
  private createExportMenu(chartId: string): void {
    const chartContainer = document.getElementById(chartId);
    if (!chartContainer) return;

    const wrapper = chartContainer.closest('.chart-container') || chartContainer.parentElement;
    if (!wrapper) return;

    // Check if export menu already exists
    if (wrapper.querySelector('.chart-export-menu')) return;

    // Create export button with dropdown
    const exportWrapper = document.createElement('div');
    exportWrapper.className = 'chart-export-wrapper';

    const exportButton = document.createElement('button');
    exportButton.className = 'chart-export-btn';
    exportButton.innerHTML = `
      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
        <polyline points="7 10 12 15 17 10"/>
        <line x1="12" y1="15" x2="12" y2="3"/>
      </svg>
      <span>Export</span>
    `;
    exportButton.setAttribute('aria-label', 'Export chart');
    exportButton.setAttribute('aria-haspopup', 'true');
    exportButton.setAttribute('aria-expanded', 'false');

    const dropdown = document.createElement('div');
    dropdown.className = 'chart-export-dropdown';
    dropdown.setAttribute('role', 'menu');
    dropdown.innerHTML = `
      <button class="chart-export-option" data-format="png-high" role="menuitem">
        <span class="export-icon">&#128247;</span>
        PNG (High Quality)
      </button>
      <button class="chart-export-option" data-format="png-low" role="menuitem">
        <span class="export-icon">&#128247;</span>
        PNG (Low Quality)
      </button>
      <button class="chart-export-option" data-format="svg" role="menuitem">
        <span class="export-icon">&#128196;</span>
        SVG (Vector)
      </button>
      <button class="chart-export-option" data-format="csv" role="menuitem">
        <span class="export-icon">&#128202;</span>
        CSV (Data)
      </button>
    `;

    exportWrapper.appendChild(exportButton);
    exportWrapper.appendChild(dropdown);

    // Add to wrapper
    const header = wrapper.querySelector('.chart-header, h4, h3');
    if (header) {
      header.appendChild(exportWrapper);
    } else {
      wrapper.insertBefore(exportWrapper, wrapper.firstChild);
    }

    // Event listeners
    exportButton.addEventListener('click', (e) => {
      e.stopPropagation();
      this.toggleDropdown(exportWrapper, exportButton);
    });

    // Handle option clicks
    dropdown.querySelectorAll('.chart-export-option').forEach(option => {
      option.addEventListener('click', (e) => {
        e.stopPropagation();
        const format = (option as HTMLElement).getAttribute('data-format') as ExportFormat;
        this.exportChart({ format, chartId });
        this.closeAllDropdowns();
      });
    });

    // Close on outside click
    document.addEventListener('click', () => this.closeAllDropdowns());

    this.exportMenus.set(chartId, exportWrapper);
  }

  /**
   * Toggle dropdown visibility
   */
  private toggleDropdown(wrapper: HTMLElement, button: HTMLElement): void {
    const isOpen = wrapper.classList.contains('open');
    this.closeAllDropdowns();

    if (!isOpen) {
      wrapper.classList.add('open');
      button.setAttribute('aria-expanded', 'true');
    }
  }

  /**
   * Close all open dropdowns
   */
  private closeAllDropdowns(): void {
    this.exportMenus.forEach((wrapper) => {
      wrapper.classList.remove('open');
      const btn = wrapper.querySelector('.chart-export-btn');
      if (btn) {
        btn.setAttribute('aria-expanded', 'false');
      }
    });
  }

  /**
   * Export chart in specified format
   */
  async exportChart(options: ExportOptions): Promise<void> {
    const chart = this.charts.get(options.chartId);
    if (!chart) {
      logger.error(`Chart not found: ${options.chartId}`);
      return;
    }

    const filename = options.filename || this.generateFilename(options.chartId, options.format);

    switch (options.format) {
      case 'png-high':
        this.exportAsPNG(chart, filename, 3);
        break;
      case 'png-low':
        this.exportAsPNG(chart, filename, 1);
        break;
      case 'svg':
        this.exportAsSVG(chart, filename);
        break;
      case 'csv':
        this.exportAsCSV(chart, filename, options.chartId);
        break;
    }

    if (this.toastManager) {
      this.toastManager.success(`Exported ${filename}`, 'Export Complete', 3000);
    }

    logger.debug(`Exported chart: ${options.chartId} as ${options.format}`);
  }

  /**
   * Export chart as PNG
   */
  private exportAsPNG(chart: echarts.ECharts, filename: string, pixelRatio: number): void {
    const url = chart.getDataURL({
      type: 'png',
      pixelRatio,
      backgroundColor: '#1a1a2e',
      excludeComponents: ['toolbox']
    });

    this.downloadFile(url, `${filename}.png`);
  }

  /**
   * Export chart as SVG
   */
  private exportAsSVG(chart: echarts.ECharts, filename: string): void {
    const url = chart.getDataURL({
      type: 'svg',
      backgroundColor: '#1a1a2e',
      excludeComponents: ['toolbox']
    });

    this.downloadFile(url, `${filename}.svg`);
  }

  /**
   * Export chart data as CSV
   */
  private exportAsCSV(chart: echarts.ECharts, filename: string, chartId: string): void {
    const option = chart.getOption() as EChartsExportOption;
    if (!option || !option.series || option.series.length === 0) {
      if (this.toastManager) {
        this.toastManager.warning('No data available to export');
      }
      return;
    }

    const csvData = this.convertToCSV(option, chartId);
    const blob = new Blob([csvData], { type: 'text/csv;charset=utf-8;' });
    const url = URL.createObjectURL(blob);

    this.downloadFile(url, `${filename}.csv`);

    // Cleanup blob URL
    setTimeout(() => URL.revokeObjectURL(url), 100);
  }

  /**
   * Convert ECharts option to CSV format
   */
  private convertToCSV(option: EChartsExportOption, _chartId: string): string {
    const rows: string[][] = [];

    // Get x-axis data if available
    const xAxis = option.xAxis;
    const xAxisData: string[] = Array.isArray(xAxis)
      ? (xAxis[0]?.data || [])
      : (xAxis?.data || []);

    // Get series data
    const series = option.series || [];

    if (series.length === 0) {
      return 'No data';
    }

    // Create header row
    const headers = ['Category/Label'];
    series.forEach((s: EChartsSeriesOption, index: number) => {
      headers.push(s.name || `Series ${index + 1}`);
    });
    rows.push(headers);

    // Handle pie chart data
    if (series[0].type === 'pie') {
      const pieData = series[0].data || [];
      pieData.forEach((item: number | string | EChartsSeriesDataItem) => {
        if (typeof item === 'object' && item !== null) {
          const name = item.name || 'Unknown';
          const value = item.value !== undefined ? item.value : 0;
          rows.push([name, String(value)]);
        } else {
          rows.push(['Unknown', String(item)]);
        }
      });
    }
    // Handle bar/line charts with categories
    else if (xAxisData.length > 0) {
      xAxisData.forEach((category: string, index: number) => {
        const row = [category];
        series.forEach((s: EChartsSeriesOption) => {
          const dataValue = s.data?.[index];
          let value = '';
          if (typeof dataValue === 'object' && dataValue !== null) {
            const dataItem = dataValue as EChartsSeriesDataItem;
            value = String(dataItem.value || 0);
          } else {
            value = String(dataValue || 0);
          }
          row.push(value);
        });
        rows.push(row);
      });
    }
    // Handle other data formats
    else {
      series.forEach((s: EChartsSeriesOption) => {
        const seriesData = s.data || [];
        seriesData.forEach((item: number | string | EChartsSeriesDataItem, index: number) => {
          const row: string[] = [];
          if (typeof item === 'object' && item !== null) {
            const dataItem = item as EChartsSeriesDataItem;
            row.push(dataItem.name || `Item ${index + 1}`);
            row.push(String(dataItem.value || 0));
          } else {
            row.push(`Item ${index + 1}`);
            row.push(String(item || 0));
          }
          rows.push(row);
        });
      });
    }

    // Convert to CSV string
    return rows.map(row =>
      row.map(cell => {
        // Escape quotes and wrap in quotes if contains comma or quote
        const escaped = String(cell).replace(/"/g, '""');
        return escaped.includes(',') || escaped.includes('"') || escaped.includes('\n')
          ? `"${escaped}"`
          : escaped;
      }).join(',')
    ).join('\n');
  }

  /**
   * Download file
   */
  private downloadFile(url: string, filename: string): void {
    const link = document.createElement('a');
    link.href = url;
    link.download = filename;
    link.style.display = 'none';
    document.body.appendChild(link);
    link.click();

    setTimeout(() => {
      if (link.parentNode) {
        document.body.removeChild(link);
      }
    }, 100);
  }

  /**
   * Generate filename from chart ID
   */
  private generateFilename(chartId: string, format: ExportFormat): string {
    const baseName = chartId.replace('chart-', '').replace(/-/g, '_');
    const timestamp = new Date().toISOString().slice(0, 10);
    const quality = format.includes('high') ? '_hq' : format.includes('low') ? '_lq' : '';
    return `${baseName}${quality}_${timestamp}`;
  }

  /**
   * Setup global export all button
   */
  private setupGlobalExportButton(): void {
    const exportAllBtn = document.getElementById('export-all-charts');
    if (exportAllBtn) {
      exportAllBtn.addEventListener('click', () => this.exportAllCharts());
    }
  }

  /**
   * Export all visible charts
   */
  async exportAllCharts(format: ExportFormat = 'png-high'): Promise<void> {
    const visibleCharts: string[] = [];

    this.charts.forEach((_chart, chartId) => {
      const container = document.getElementById(chartId);
      if (container && container.offsetParent !== null) {
        visibleCharts.push(chartId);
      }
    });

    if (visibleCharts.length === 0) {
      if (this.toastManager) {
        this.toastManager.warning('No charts visible to export');
      }
      return;
    }

    if (this.toastManager) {
      this.toastManager.info(`Exporting ${visibleCharts.length} charts...`, 'Bulk Export');
    }

    // Export with delay to prevent browser from freezing
    for (let i = 0; i < visibleCharts.length; i++) {
      const chartId = visibleCharts[i];
      await this.exportChart({ format, chartId });
      // Small delay between exports
      await new Promise(resolve => setTimeout(resolve, 200));
    }

    if (this.toastManager) {
      this.toastManager.success(`Exported ${visibleCharts.length} charts`, 'Bulk Export Complete');
    }
  }

  /**
   * Initialize the export manager
   */
  init(): void {
    logger.debug('ChartExportManager initialized');
  }

  /**
   * Cleanup
   */
  destroy(): void {
    this.exportMenus.forEach((wrapper) => {
      if (wrapper.parentNode) {
        wrapper.parentNode.removeChild(wrapper);
      }
    });
    this.exportMenus.clear();
    this.charts.clear();
  }
}

export default ChartExportManager;
