// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * ChartMaximizeManager - Chart fullscreen/maximize feature
 *
 * Features:
 * - Maximize any chart to fullscreen overlay
 * - Escape key to close
 * - Click outside to close
 * - Chart resizes to fill available space
 * - Keyboard accessible (Enter to maximize when focused)
 */

import * as echarts from 'echarts';
import { createLogger } from '../lib/logger';

const logger = createLogger('ChartMaximizeManager');

export class ChartMaximizeManager {
  private charts: Map<string, echarts.ECharts> = new Map();
  private overlay: HTMLElement | null = null;
  private maximizedChart: { chartId: string; chart: echarts.ECharts } | null = null;
  /** Maximized chart instance for cleanup */
  private maximizedChartInstance: echarts.ECharts | null = null;

  // Bound event handler for proper cleanup
  private readonly boundHandleKeyDown: (e: KeyboardEvent) => void;

  constructor() {
    // Bind the handler once to ensure same reference for add/remove
    this.boundHandleKeyDown = this.handleKeyDown.bind(this);

    this.createOverlay();
    this.setupKeyboardListeners();
  }

  /**
   * Handle keyboard events
   */
  private handleKeyDown(e: KeyboardEvent): void {
    if (e.key === 'Escape' && this.maximizedChart) {
      this.closeMaximized();
    }
  }

  /**
   * Register a chart for maximize support
   */
  registerChart(chartId: string, chart: echarts.ECharts): void {
    this.charts.set(chartId, chart);
    this.addMaximizeButton(chartId);
    this.addDoubleClickHandler(chartId, chart);
  }

  /**
   * Create fullscreen overlay
   */
  private createOverlay(): void {
    if (this.overlay) return;

    const overlay = document.createElement('div');
    overlay.id = 'chart-maximize-overlay';
    overlay.className = 'chart-maximize-overlay';
    overlay.setAttribute('role', 'dialog');
    overlay.setAttribute('aria-modal', 'true');
    overlay.setAttribute('aria-label', 'Maximized chart view');
    overlay.innerHTML = `
      <div class="chart-maximize-header">
        <h2 class="chart-maximize-title" id="maximized-chart-title"></h2>
        <button class="chart-maximize-close" aria-label="Close maximized view">
          <span aria-hidden="true">&times;</span>
        </button>
      </div>
      <div class="chart-maximize-content" id="maximized-chart-content"></div>
    `;

    // Close button handler
    const closeBtn = overlay.querySelector('.chart-maximize-close');
    if (closeBtn) {
      closeBtn.addEventListener('click', () => this.closeMaximized());
    }

    // Click outside to close
    overlay.addEventListener('click', (e) => {
      if (e.target === overlay) {
        this.closeMaximized();
      }
    });

    document.body.appendChild(overlay);
    this.overlay = overlay;
  }

  /**
   * Setup keyboard listeners
   */
  private setupKeyboardListeners(): void {
    document.addEventListener('keydown', this.boundHandleKeyDown);
  }

  /**
   * Add maximize button to chart header
   */
  private addMaximizeButton(chartId: string): void {
    const container = document.getElementById(chartId);
    if (!container) return;

    const wrapper = container.closest('.chart-container') || container.parentElement;
    if (!wrapper) return;

    // Check if button already exists
    if (wrapper.querySelector('.chart-maximize-btn')) return;

    const button = document.createElement('button');
    button.className = 'chart-maximize-btn';
    button.innerHTML = `
      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M8 3H5a2 2 0 0 0-2 2v3m18 0V5a2 2 0 0 0-2-2h-3m0 18h3a2 2 0 0 0 2-2v-3M3 16v3a2 2 0 0 0 2 2h3"/>
      </svg>
    `;
    button.setAttribute('aria-label', 'Maximize chart');
    button.setAttribute('title', 'Maximize');
    button.setAttribute('data-chart-id', chartId);

    button.addEventListener('click', (e) => {
      e.stopPropagation();
      this.maximizeChart(chartId);
    });

    // Add to header if exists, otherwise to wrapper
    const header = wrapper.querySelector('.chart-header, h4, h3');
    if (header) {
      header.appendChild(button);
    } else {
      wrapper.insertBefore(button, wrapper.firstChild);
    }
  }

  /**
   * Add double-click handler to maximize
   */
  private addDoubleClickHandler(chartId: string, _chart: echarts.ECharts): void {
    const container = document.getElementById(chartId);
    if (!container) return;

    // Double-click on chart area to maximize
    container.addEventListener('dblclick', () => {
      this.maximizeChart(chartId);
    });

    // Add data attribute for CSS cursor hint
    container.setAttribute('data-maximizable', 'true');
  }

  /**
   * Maximize a chart
   */
  maximizeChart(chartId: string): void {
    const chart = this.charts.get(chartId);
    const originalContainer = document.getElementById(chartId);

    if (!chart || !originalContainer || !this.overlay) return;

    // Store reference
    this.maximizedChart = { chartId, chart };

    // Get chart title
    const wrapper = originalContainer.closest('.chart-container') || originalContainer.parentElement;
    const titleEl = wrapper?.querySelector('h3, h4, .chart-title');
    const title = titleEl?.textContent || chartId.replace('chart-', '').replace(/-/g, ' ');

    // Update overlay title
    const overlayTitle = this.overlay.querySelector('#maximized-chart-title');
    if (overlayTitle) {
      overlayTitle.textContent = title;
    }

    // Get content container
    const content = this.overlay.querySelector('#maximized-chart-content');
    if (!content) return;

    // Clone the chart container
    const clonedContainer = document.createElement('div');
    clonedContainer.id = 'maximized-chart-canvas';
    clonedContainer.className = 'maximized-chart-canvas';
    content.innerHTML = '';
    content.appendChild(clonedContainer);

    // Show overlay
    this.overlay.classList.add('active');
    document.body.style.overflow = 'hidden';

    // Wait for overlay to be visible, then resize chart
    requestAnimationFrame(() => {
      // Get original chart option
      const option = chart.getOption();

      // Initialize new chart in overlay
      const maximizedChartInstance = echarts.init(clonedContainer, 'dark', { renderer: 'canvas' });
      maximizedChartInstance.setOption(option);

      // Store reference for cleanup
      this.maximizedChartInstance = maximizedChartInstance;
    });

    // Focus close button for accessibility
    const closeBtn = this.overlay.querySelector('.chart-maximize-close') as HTMLElement;
    if (closeBtn) {
      setTimeout(() => closeBtn.focus(), 100);
    }

    logger.debug(`Chart maximized: ${chartId}`);
  }

  /**
   * Close maximized view
   */
  closeMaximized(): void {
    if (!this.overlay || !this.maximizedChart) return;

    // Dispose maximized chart instance
    if (this.maximizedChartInstance) {
      this.maximizedChartInstance.dispose();
      this.maximizedChartInstance = null;
    }

    // Hide overlay
    this.overlay.classList.remove('active');
    document.body.style.overflow = '';

    // Clear content
    const content = this.overlay.querySelector('#maximized-chart-content');
    if (content) {
      content.innerHTML = '';
    }

    const chartId = this.maximizedChart.chartId;
    this.maximizedChart = null;

    // Return focus to maximize button
    const originalContainer = document.getElementById(chartId);
    if (originalContainer) {
      const wrapper = originalContainer.closest('.chart-container') || originalContainer.parentElement;
      const maximizeBtn = wrapper?.querySelector('.chart-maximize-btn') as HTMLElement;
      if (maximizeBtn) {
        maximizeBtn.focus();
      }
    }

    logger.debug(`Chart minimized: ${chartId}`);
  }

  /**
   * Initialize the maximize manager
   */
  init(): void {
    logger.debug('ChartMaximizeManager initialized');
  }

  /**
   * Cleanup - properly removes all event listeners and DOM elements
   */
  destroy(): void {
    // Close any maximized chart first
    if (this.maximizedChart) {
      this.closeMaximized();
    }

    // Remove keyboard listener
    document.removeEventListener('keydown', this.boundHandleKeyDown);

    // Remove overlay from DOM
    if (this.overlay && this.overlay.parentNode) {
      this.overlay.parentNode.removeChild(this.overlay);
    }
    this.overlay = null;

    // Clear chart references
    this.charts.clear();
  }
}

export default ChartMaximizeManager;
