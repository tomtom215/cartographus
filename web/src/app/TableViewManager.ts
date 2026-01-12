// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * TableViewManager - Table View Option
 *
 * Provides an alternative table view for chart data.
 * Users can toggle between chart visualization and tabular data view.
 *
 * Features:
 * - Toggle button on chart cards
 * - Sortable table columns
 * - Copy to clipboard functionality
 * - Keyboard accessible
 * - Screen reader friendly with proper ARIA attributes
 */

import * as echarts from 'echarts';
import type { EChartsChartOption, EChartsSeriesOption } from '../lib/charts/types';

/**
 * Table data structure
 */
interface TableData {
  headers: string[];
  rows: (string | number)[][];
}

/**
 * Sort state for columns
 */
interface SortState {
  column: number;
  direction: 'asc' | 'desc';
}

export class TableViewManager {
  private chartContainers: Map<string, HTMLElement> = new Map();
  private tableContainers: Map<string, HTMLElement> = new Map();
  private toggleButtons: Map<string, HTMLElement> = new Map();
  private viewStates: Map<string, 'chart' | 'table'> = new Map();
  private sortStates: Map<string, SortState> = new Map();
  private chartDataCache: Map<string, TableData> = new Map();

  constructor() {
    this.initializeToggleButtons();
  }

  /**
   * Initialize toggle buttons for all chart cards
   */
  private initializeToggleButtons(): void {
    // Find all chart cards and add toggle buttons
    const chartCards = document.querySelectorAll('.chart-card');
    chartCards.forEach((card) => {
      const chartContent = card.querySelector('.chart-content') as HTMLElement;
      if (!chartContent || !chartContent.id) return;

      const chartId = chartContent.id;
      this.chartContainers.set(chartId, chartContent);
      this.viewStates.set(chartId, 'chart');

      // Create toggle button
      this.createToggleButton(card as HTMLElement, chartId);

      // Create table container (hidden by default)
      this.createTableContainer(card as HTMLElement, chartId);
    });
  }

  /**
   * Create toggle button for a chart card
   */
  private createToggleButton(card: HTMLElement, chartId: string): void {
    const chartHeader = card.querySelector('.chart-header');
    if (!chartHeader) return;

    // Check if toggle already exists
    if (chartHeader.querySelector('.table-view-toggle')) return;

    const toggleButton = document.createElement('button');
    toggleButton.type = 'button';
    toggleButton.className = 'table-view-toggle';
    toggleButton.setAttribute('aria-label', 'Toggle table view');
    toggleButton.setAttribute('aria-pressed', 'false');
    toggleButton.setAttribute('data-chart-id', chartId);
    toggleButton.innerHTML = `
      <svg class="table-icon" width="16" height="16" viewBox="0 0 16 16" fill="currentColor" aria-hidden="true">
        <path d="M0 2a2 2 0 0 1 2-2h12a2 2 0 0 1 2 2v12a2 2 0 0 1-2 2H2a2 2 0 0 1-2-2V2zm15 2h-4v3h4V4zm0 4h-4v3h4V8zm0 4h-4v3h3a1 1 0 0 0 1-1v-2zm-5 3v-3H6v3h4zm-5 0v-3H1v2a1 1 0 0 0 1 1h3zm-4-4h4V8H1v3zm0-4h4V4H1v3zm5-3v3h4V4H6zm4 4H6v3h4V8z"/>
      </svg>
      <span class="toggle-label">Table</span>
    `;

    toggleButton.addEventListener('click', () => this.toggleView(chartId));

    // Insert before export button if exists, otherwise append
    const exportButton = chartHeader.querySelector('.chart-export');
    if (exportButton) {
      chartHeader.insertBefore(toggleButton, exportButton);
    } else {
      chartHeader.appendChild(toggleButton);
    }

    this.toggleButtons.set(chartId, toggleButton);
  }

  /**
   * Create table container for a chart
   */
  private createTableContainer(card: HTMLElement, chartId: string): void {
    const tableContainer = document.createElement('div');
    tableContainer.className = 'chart-table-container';
    tableContainer.id = `${chartId}-table`;
    tableContainer.setAttribute('role', 'region');
    tableContainer.setAttribute('aria-label', 'Chart data table');
    tableContainer.style.display = 'none';

    // Insert after chart content
    const chartContent = card.querySelector('.chart-content');
    if (chartContent && chartContent.parentNode) {
      chartContent.parentNode.insertBefore(tableContainer, chartContent.nextSibling);
    }

    this.tableContainers.set(chartId, tableContainer);
  }

  /**
   * Toggle between chart and table view
   */
  toggleView(chartId: string): void {
    const currentState = this.viewStates.get(chartId) || 'chart';
    const newState = currentState === 'chart' ? 'table' : 'chart';

    const chartContainer = this.chartContainers.get(chartId);
    const tableContainer = this.tableContainers.get(chartId);
    const toggleButton = this.toggleButtons.get(chartId);

    if (!chartContainer || !tableContainer || !toggleButton) return;

    if (newState === 'table') {
      // Switch to table view
      chartContainer.style.display = 'none';
      tableContainer.style.display = 'block';
      toggleButton.setAttribute('aria-pressed', 'true');
      toggleButton.classList.add('active');
      toggleButton.querySelector('.toggle-label')!.textContent = 'Chart';

      // Extract and render table data
      this.renderTable(chartId);
    } else {
      // Switch to chart view
      chartContainer.style.display = 'block';
      tableContainer.style.display = 'none';
      toggleButton.setAttribute('aria-pressed', 'false');
      toggleButton.classList.remove('active');
      toggleButton.querySelector('.toggle-label')!.textContent = 'Table';
    }

    this.viewStates.set(chartId, newState);

    // Announce state change to screen readers
    this.announceViewChange(chartId, newState);
  }

  /**
   * Extract data from chart and render as table
   */
  private renderTable(chartId: string): void {
    const tableContainer = this.tableContainers.get(chartId);
    if (!tableContainer) return;

    // Try to get cached data or extract from chart
    let tableData: TableData | undefined = this.chartDataCache.get(chartId);
    if (!tableData) {
      const extractedData = this.extractChartData(chartId);
      if (extractedData) {
        this.chartDataCache.set(chartId, extractedData);
        tableData = extractedData;
      }
    }

    if (!tableData || tableData.rows.length === 0) {
      tableContainer.innerHTML = `
        <div class="table-empty-state">
          <p>No data available for table view</p>
        </div>
      `;
      return;
    }

    // Apply sorting if exists
    const sortState = this.sortStates.get(chartId);
    const sortedRows = sortState ? this.sortRows(tableData.rows, sortState) : tableData.rows;

    // Render table
    tableContainer.innerHTML = `
      <div class="table-toolbar">
        <button type="button" class="table-copy-btn" aria-label="Copy table data to clipboard">
          <svg width="14" height="14" viewBox="0 0 16 16" fill="currentColor" aria-hidden="true">
            <path d="M4 1.5H3a2 2 0 0 0-2 2V14a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2V3.5a2 2 0 0 0-2-2h-1v1h1a1 1 0 0 1 1 1V14a1 1 0 0 1-1 1H3a1 1 0 0 1-1-1V3.5a1 1 0 0 1 1-1h1v-1z"/>
            <path d="M9.5 1a.5.5 0 0 1 .5.5v1a.5.5 0 0 1-.5.5h-3a.5.5 0 0 1-.5-.5v-1a.5.5 0 0 1 .5-.5h3zm-3-1A1.5 1.5 0 0 0 5 1.5v1A1.5 1.5 0 0 0 6.5 4h3A1.5 1.5 0 0 0 11 2.5v-1A1.5 1.5 0 0 0 9.5 0h-3z"/>
          </svg>
          Copy
        </button>
        <span class="table-row-count">${sortedRows.length} rows</span>
      </div>
      <div class="table-wrapper">
        <table class="data-table" aria-label="Chart data">
          <thead>
            <tr>
              ${tableData.headers.map((header, i) => `
                <th scope="col" data-col="${i}" tabindex="0" role="columnheader" aria-sort="${this.getAriaSortValue(chartId, i)}">
                  <span class="th-content">${this.escapeHtml(header)}</span>
                  <span class="sort-indicator" aria-hidden="true"></span>
                </th>
              `).join('')}
            </tr>
          </thead>
          <tbody>
            ${sortedRows.map(row => `
              <tr>
                ${row.map(cell => `<td>${this.formatCell(cell)}</td>`).join('')}
              </tr>
            `).join('')}
          </tbody>
        </table>
      </div>
    `;

    // Setup event listeners
    this.setupTableEventListeners(chartId, tableContainer, tableData);
  }

  /**
   * Extract data from ECharts instance
   */
  private extractChartData(chartId: string): TableData | null {
    // Get chart instance from DOM
    const chartContainer = document.getElementById(chartId);
    if (!chartContainer) return null;

    // Access ECharts instance
    const chart = echarts.getInstanceByDom(chartContainer);
    if (!chart) return null;

    const option = chart.getOption() as EChartsChartOption;
    if (!option) return null;

    const headers: string[] = [];
    const rows: (string | number)[][] = [];

    // Try to extract data based on chart type
    if (option.xAxis && option.xAxis[0]?.data && option.series) {
      // Line/Bar chart with category axis
      const xAxisData = option.xAxis[0].data;
      headers.push(option.xAxis[0].name || 'Category');

      option.series.forEach((series: EChartsSeriesOption) => {
        if (series.name) {
          headers.push(series.name);
        }
      });

      xAxisData.forEach((category, index) => {
        const row: (string | number)[] = [category];
        option.series?.forEach((series: EChartsSeriesOption) => {
          const value = series.data?.[index];
          row.push(typeof value === 'object' && value !== null ? (value as { value?: number }).value ?? 0 : (value as number) ?? 0);
        });
        rows.push(row);
      });
    } else if (option.series && option.series[0]?.type === 'pie') {
      // Pie chart
      headers.push('Name', 'Value', 'Percentage');

      const series = option.series[0];
      const seriesData = series.data ?? [];
      const total = seriesData.reduce((sum: number, item) => {
        if (typeof item === 'object' && item !== null && 'value' in item) {
          return sum + ((item.value as number) || 0);
        }
        return sum;
      }, 0);

      seriesData.forEach((item) => {
        if (typeof item === 'object' && item !== null && 'name' in item) {
          const itemValue = (item.value as number) || 0;
          const percentage = total > 0 ? ((itemValue / total) * 100).toFixed(1) + '%' : '0%';
          rows.push([item.name || 'Unknown', itemValue, percentage]);
        }
      });
    } else if (option.yAxis && option.yAxis[0]?.data && option.series) {
      // Horizontal bar chart
      const yAxisData = option.yAxis[0].data;
      headers.push(option.yAxis[0].name || 'Category');

      option.series.forEach((series: EChartsSeriesOption) => {
        if (series.name) {
          headers.push(series.name);
        }
      });

      yAxisData.forEach((category, index) => {
        const row: (string | number)[] = [category];
        option.series?.forEach((series: EChartsSeriesOption) => {
          const value = series.data?.[index];
          row.push(typeof value === 'object' && value !== null ? (value as { value?: number }).value ?? 0 : (value as number) ?? 0);
        });
        rows.push(row);
      });
    }

    if (headers.length === 0 || rows.length === 0) {
      return null;
    }

    return { headers, rows };
  }

  /**
   * Setup event listeners for table interactions
   */
  private setupTableEventListeners(chartId: string, container: HTMLElement, tableData: TableData): void {
    // Copy button
    const copyBtn = container.querySelector('.table-copy-btn');
    copyBtn?.addEventListener('click', () => this.copyTableData(chartId, tableData));

    // Sortable headers
    const headers = container.querySelectorAll('th[data-col]');
    headers.forEach((th) => {
      th.addEventListener('click', () => {
        const col = parseInt(th.getAttribute('data-col') || '0', 10);
        this.sortByColumn(chartId, col);
      });
      th.addEventListener('keydown', (e: Event) => {
        const keyEvent = e as KeyboardEvent;
        if (keyEvent.key === 'Enter' || keyEvent.key === ' ') {
          keyEvent.preventDefault();
          const col = parseInt((th as HTMLElement).getAttribute('data-col') || '0', 10);
          this.sortByColumn(chartId, col);
        }
      });
    });
  }

  /**
   * Sort table by column
   */
  private sortByColumn(chartId: string, column: number): void {
    const currentSort = this.sortStates.get(chartId);
    let newDirection: 'asc' | 'desc' = 'asc';

    if (currentSort && currentSort.column === column) {
      newDirection = currentSort.direction === 'asc' ? 'desc' : 'asc';
    }

    this.sortStates.set(chartId, { column, direction: newDirection });
    this.renderTable(chartId);
  }

  /**
   * Sort rows by column and direction
   */
  private sortRows(rows: (string | number)[][], sortState: SortState): (string | number)[][] {
    const { column, direction } = sortState;
    const sorted = [...rows].sort((a, b) => {
      const valA = a[column];
      const valB = b[column];

      // Handle numeric comparison
      const numA = typeof valA === 'number' ? valA : parseFloat(String(valA).replace(/[^0-9.-]/g, ''));
      const numB = typeof valB === 'number' ? valB : parseFloat(String(valB).replace(/[^0-9.-]/g, ''));

      if (!isNaN(numA) && !isNaN(numB)) {
        return direction === 'asc' ? numA - numB : numB - numA;
      }

      // String comparison
      const strA = String(valA).toLowerCase();
      const strB = String(valB).toLowerCase();
      const comparison = strA.localeCompare(strB);
      return direction === 'asc' ? comparison : -comparison;
    });

    return sorted;
  }

  /**
   * Get ARIA sort value for column
   */
  private getAriaSortValue(chartId: string, column: number): string {
    const sortState = this.sortStates.get(chartId);
    if (!sortState || sortState.column !== column) {
      return 'none';
    }
    return sortState.direction === 'asc' ? 'ascending' : 'descending';
  }

  /**
   * Copy table data to clipboard
   */
  private async copyTableData(chartId: string, tableData: TableData): Promise<void> {
    const sortState = this.sortStates.get(chartId);
    const rows = sortState ? this.sortRows(tableData.rows, sortState) : tableData.rows;

    // Format as TSV for spreadsheet compatibility
    const tsv = [
      tableData.headers.join('\t'),
      ...rows.map(row => row.join('\t'))
    ].join('\n');

    try {
      await navigator.clipboard.writeText(tsv);
      this.showCopyFeedback(chartId, true);
    } catch {
      // Fallback for older browsers
      const textarea = document.createElement('textarea');
      textarea.value = tsv;
      document.body.appendChild(textarea);
      textarea.select();
      document.execCommand('copy');
      document.body.removeChild(textarea);
      this.showCopyFeedback(chartId, true);
    }
  }

  /**
   * Show copy feedback
   */
  private showCopyFeedback(chartId: string, success: boolean): void {
    const container = this.tableContainers.get(chartId);
    if (!container) return;

    const copyBtn = container.querySelector('.table-copy-btn');
    if (copyBtn) {
      const originalText = copyBtn.innerHTML;
      copyBtn.innerHTML = success
        ? '<svg width="14" height="14" viewBox="0 0 16 16" fill="currentColor"><path d="M13.854 3.646a.5.5 0 0 1 0 .708l-7 7a.5.5 0 0 1-.708 0l-3.5-3.5a.5.5 0 1 1 .708-.708L6.5 10.293l6.646-6.647a.5.5 0 0 1 .708 0z"/></svg> Copied!'
        : 'Failed';
      copyBtn.classList.add(success ? 'copied' : 'error');

      setTimeout(() => {
        copyBtn.innerHTML = originalText;
        copyBtn.classList.remove('copied', 'error');
      }, 2000);
    }
  }

  /**
   * Format cell value for display
   */
  private formatCell(value: string | number): string {
    if (typeof value === 'number') {
      // Format large numbers with commas
      return value.toLocaleString();
    }
    return this.escapeHtml(String(value));
  }

  /**
   * Escape HTML to prevent XSS
   */
  private escapeHtml(text: string): string {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
  }

  /**
   * Announce view change to screen readers
   */
  private announceViewChange(chartId: string, view: 'chart' | 'table'): void {
    const announcer = document.getElementById('chart-announcer');
    if (!announcer) return;

    const chartTitle = this.getChartTitle(chartId);
    const message = view === 'table'
      ? `${chartTitle} now showing table view`
      : `${chartTitle} now showing chart view`;

    announcer.textContent = '';
    setTimeout(() => {
      announcer.textContent = message;
    }, 100);
  }

  /**
   * Get chart title for announcements
   */
  private getChartTitle(chartId: string): string {
    const container = this.chartContainers.get(chartId);
    if (!container) return 'Chart';

    const card = container.closest('.chart-card');
    const title = card?.querySelector('.chart-title, h3, h4');
    return title?.textContent || chartId.replace('chart-', '').replace(/-/g, ' ');
  }

  /**
   * Clear cached data for a chart (call when chart data updates)
   */
  clearCache(chartId?: string): void {
    if (chartId) {
      this.chartDataCache.delete(chartId);
      // Re-render if in table view
      if (this.viewStates.get(chartId) === 'table') {
        this.renderTable(chartId);
      }
    } else {
      this.chartDataCache.clear();
      // Re-render all visible tables
      this.viewStates.forEach((state, id) => {
        if (state === 'table') {
          this.renderTable(id);
        }
      });
    }
  }

  /**
   * Refresh toggle buttons (call when new charts are added to DOM)
   */
  refresh(): void {
    this.initializeToggleButtons();
  }

  /**
   * Destroy the manager
   */
  destroy(): void {
    // Remove toggle buttons
    this.toggleButtons.forEach((button) => {
      button.remove();
    });
    this.toggleButtons.clear();

    // Remove table containers
    this.tableContainers.forEach((container) => {
      container.remove();
    });
    this.tableContainers.clear();

    // Restore chart containers
    this.chartContainers.forEach((container) => {
      container.style.display = 'block';
    });
    this.chartContainers.clear();

    this.viewStates.clear();
    this.sortStates.clear();
    this.chartDataCache.clear();
  }
}
