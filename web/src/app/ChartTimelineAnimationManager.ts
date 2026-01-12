// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * ChartTimelineAnimationManager - Timeline Animation for Analytics Charts
 *
 * Features:
 * - Animates through time-series data in ECharts visualizations
 * - Play/pause/reset controls for chart animations
 * - Speed control (1x, 2x, 5x, 10x)
 * - Progress slider for scrubbing through time
 * - Works with any chart that has time-series data
 * - Smooth transitions between data points
 */

import type { API, LocationFilter } from '../lib/api';
import { createLogger } from '../lib/logger';

const logger = createLogger('ChartTimelineAnimationManager');

type PlaybackSpeed = 1 | 2 | 5 | 10;
type AnimationInterval = 'daily' | 'weekly' | 'monthly';

interface TimeSeriesDataPoint {
  date: string;
  values: Record<string, number>;
}

interface ChartAnimationConfig {
  chartId: string;
  chartInstance: unknown;
  dataKey: string;
  labelKey?: string;
}

interface AnimationCallbacks {
  onFrameUpdate?: (frame: number, totalFrames: number, data: TimeSeriesDataPoint) => void;
  onPlayStateChange?: (isPlaying: boolean) => void;
  onComplete?: () => void;
}

export class ChartTimelineAnimationManager {
  private api: API;
  private containerId: string;
  private isPlaying: boolean = false;
  private currentFrame: number = 0;
  private totalFrames: number = 0;
  private playbackSpeed: PlaybackSpeed = 1;
  private animationInterval: AnimationInterval = 'daily';
  private animationFrameId: number | null = null;
  private lastUpdateTime: number = 0;
  private timeSeriesData: TimeSeriesDataPoint[] = [];
  private registeredCharts: Map<string, ChartAnimationConfig> = new Map();
  private callbacks: AnimationCallbacks = {};
  private frameAccumulator: number = 0;
  // AbortController for clean event listener removal
  private abortController: AbortController | null = null;

  constructor(api: API, containerId: string = 'chart-timeline-animation-container') {
    this.api = api;
    this.containerId = containerId;
  }

  /**
   * Initialize the chart timeline animation manager
   */
  async init(): Promise<void> {
    this.setupContainer();
    this.setupEventListeners();
    logger.debug('ChartTimelineAnimationManager initialized');
  }

  /**
   * Set up the container HTML
   */
  private setupContainer(): void {
    const container = document.getElementById(this.containerId);
    if (!container) {
      logger.warn('Chart timeline animation container not found', { containerId: this.containerId });
      return;
    }

    container.setAttribute('role', 'region');
    container.setAttribute('aria-label', 'Chart timeline animation controls');

    container.innerHTML = `
      <div class="chart-timeline-header">
        <h4 class="chart-timeline-title">Timeline Animation</h4>
        <span class="chart-timeline-status" id="chart-timeline-status" aria-live="polite"></span>
      </div>
      <div class="chart-timeline-controls">
        <div class="chart-timeline-buttons">
          <button id="chart-timeline-play-pause" class="chart-timeline-btn" aria-label="Play timeline animation" title="Play/Pause">
            <span id="chart-timeline-play-icon" class="icon-play" aria-hidden="true">&#9658;</span>
            <span id="chart-timeline-pause-icon" class="icon-pause" style="display:none" aria-hidden="true">&#10074;&#10074;</span>
          </button>
          <button id="chart-timeline-reset" class="chart-timeline-btn" aria-label="Reset timeline" title="Reset">
            <span class="icon-reset" aria-hidden="true">&#8634;</span>
          </button>
        </div>
        <div class="chart-timeline-slider-container">
          <input type="range" id="chart-timeline-slider" class="chart-timeline-slider"
                 min="0" max="100" value="0" step="1"
                 aria-label="Timeline progress"
                 aria-valuemin="0" aria-valuemax="100" aria-valuenow="0">
          <div class="chart-timeline-labels">
            <span id="chart-timeline-current-date" class="chart-timeline-date">--</span>
            <span id="chart-timeline-frame-count" class="chart-timeline-frame">0 / 0</span>
          </div>
        </div>
        <div class="chart-timeline-options">
          <label class="chart-timeline-option">
            <span>Speed:</span>
            <select id="chart-timeline-speed" aria-label="Animation speed">
              <option value="1">1x</option>
              <option value="2">2x</option>
              <option value="5">5x</option>
              <option value="10">10x</option>
            </select>
          </label>
          <label class="chart-timeline-option">
            <span>Interval:</span>
            <select id="chart-timeline-interval" aria-label="Time interval">
              <option value="daily">Daily</option>
              <option value="weekly">Weekly</option>
              <option value="monthly">Monthly</option>
            </select>
          </label>
        </div>
      </div>
      <div class="chart-timeline-progress" role="progressbar" aria-valuemin="0" aria-valuemax="100" aria-valuenow="0">
        <div id="chart-timeline-progress-bar" class="chart-timeline-progress-bar"></div>
      </div>
    `;
  }

  /**
   * Set up event listeners with AbortController for clean removal
   */
  private setupEventListeners(): void {
    // Create AbortController for cleanup
    this.abortController = new AbortController();
    const signal = this.abortController.signal;

    const playPauseBtn = document.getElementById('chart-timeline-play-pause');
    if (playPauseBtn) {
      playPauseBtn.addEventListener('click', () => this.togglePlayback(), { signal });
    }

    const resetBtn = document.getElementById('chart-timeline-reset');
    if (resetBtn) {
      resetBtn.addEventListener('click', () => this.reset(), { signal });
    }

    const slider = document.getElementById('chart-timeline-slider') as HTMLInputElement;
    if (slider) {
      slider.addEventListener('input', (e) => this.handleSeek(e), { signal });
    }

    const speedSelect = document.getElementById('chart-timeline-speed') as HTMLSelectElement;
    if (speedSelect) {
      speedSelect.addEventListener('change', (e) => this.handleSpeedChange(e), { signal });
    }

    const intervalSelect = document.getElementById('chart-timeline-interval') as HTMLSelectElement;
    if (intervalSelect) {
      intervalSelect.addEventListener('change', (e) => this.handleIntervalChange(e), { signal });
    }
  }

  /**
   * Load time-series data from API
   */
  async loadData(filter: LocationFilter = {}): Promise<void> {
    this.updateStatus('Loading data...');

    try {
      // Get analytics trends data (time series)
      const trendsData = await this.api.getAnalyticsTrends(filter);

      if (trendsData && trendsData.playback_trends && trendsData.playback_trends.length > 0) {
        this.timeSeriesData = this.processTimeSeriesData(trendsData.playback_trends);
        this.totalFrames = this.timeSeriesData.length;
        this.currentFrame = 0;
        this.updateUI();
        this.updateStatus(`${this.totalFrames} data points loaded`);
      } else {
        this.updateStatus('No time series data available');
      }
    } catch (error) {
      logger.error('Failed to load time series data', { error });
      this.updateStatus('Failed to load data');
    }
  }

  /**
   * Process raw API data into time series format
   */
  private processTimeSeriesData(rawData: unknown[]): TimeSeriesDataPoint[] {
    // Handle different data structures from analytics endpoints
    const processed: TimeSeriesDataPoint[] = [];

    for (const item of rawData) {
      const dataItem = item as Record<string, unknown>;
      const date = (dataItem.date || dataItem.timestamp || dataItem.period) as string;

      if (!date) continue;

      const values: Record<string, number> = {};

      // Extract numeric values
      for (const [key, val] of Object.entries(dataItem)) {
        if (key !== 'date' && key !== 'timestamp' && key !== 'period' && typeof val === 'number') {
          values[key] = val;
        }
      }

      processed.push({ date, values });
    }

    // Sort by date
    return processed.sort((a, b) =>
      new Date(a.date).getTime() - new Date(b.date).getTime()
    );
  }

  /**
   * Register a chart for animation
   */
  registerChart(config: ChartAnimationConfig): void {
    this.registeredCharts.set(config.chartId, config);
    logger.debug(`Registered chart: ${config.chartId}`);
  }

  /**
   * Unregister a chart from animation
   */
  unregisterChart(chartId: string): void {
    this.registeredCharts.delete(chartId);
  }

  /**
   * Set animation callbacks
   */
  setCallbacks(callbacks: AnimationCallbacks): void {
    this.callbacks = callbacks;
  }

  /**
   * Toggle playback (play/pause)
   */
  togglePlayback(): void {
    if (this.isPlaying) {
      this.pause();
    } else {
      this.play();
    }
  }

  /**
   * Start playback
   */
  play(): void {
    if (this.isPlaying || this.totalFrames === 0) return;

    if (this.currentFrame >= this.totalFrames - 1) {
      this.currentFrame = 0;
    }

    this.isPlaying = true;
    this.lastUpdateTime = performance.now();
    this.frameAccumulator = 0;
    this.animate();
    this.updatePlayPauseButton(true);
    this.updateStatus('Playing...');

    if (this.callbacks.onPlayStateChange) {
      this.callbacks.onPlayStateChange(true);
    }
  }

  /**
   * Pause playback
   */
  pause(): void {
    this.isPlaying = false;

    if (this.animationFrameId !== null) {
      cancelAnimationFrame(this.animationFrameId);
      this.animationFrameId = null;
    }

    this.updatePlayPauseButton(false);
    this.updateStatus('Paused');

    if (this.callbacks.onPlayStateChange) {
      this.callbacks.onPlayStateChange(false);
    }
  }

  /**
   * Reset to beginning
   */
  reset(): void {
    this.pause();
    this.currentFrame = 0;
    this.updateUI();
    this.updateCharts();
    this.updateStatus('Reset');
  }

  /**
   * Animation loop
   */
  private animate(): void {
    if (!this.isPlaying) return;

    const now = performance.now();
    const deltaTime = now - this.lastUpdateTime;
    this.lastUpdateTime = now;

    // Calculate frame advancement based on speed and interval
    const msPerFrame = this.getMillisecondsPerFrame();
    this.frameAccumulator += deltaTime * this.playbackSpeed;

    if (this.frameAccumulator >= msPerFrame) {
      this.frameAccumulator -= msPerFrame;
      this.currentFrame++;

      if (this.currentFrame >= this.totalFrames) {
        this.currentFrame = this.totalFrames - 1;
        this.pause();
        this.updateStatus('Complete');

        if (this.callbacks.onComplete) {
          this.callbacks.onComplete();
        }
        return;
      }

      this.updateUI();
      this.updateCharts();

      if (this.callbacks.onFrameUpdate && this.timeSeriesData[this.currentFrame]) {
        this.callbacks.onFrameUpdate(
          this.currentFrame,
          this.totalFrames,
          this.timeSeriesData[this.currentFrame]
        );
      }
    }

    if (this.isPlaying) {
      this.animationFrameId = requestAnimationFrame(() => this.animate());
    }
  }

  /**
   * Get milliseconds per frame based on interval
   */
  private getMillisecondsPerFrame(): number {
    switch (this.animationInterval) {
      case 'daily':
        return 500; // 0.5 seconds per day
      case 'weekly':
        return 1000; // 1 second per week
      case 'monthly':
        return 1500; // 1.5 seconds per month
      default:
        return 500;
    }
  }

  /**
   * Handle slider seek
   */
  private handleSeek(e: Event): void {
    const slider = e.target as HTMLInputElement;
    const progress = parseFloat(slider.value) / 100;
    this.currentFrame = Math.floor(progress * (this.totalFrames - 1));
    this.updateUI();
    this.updateCharts();
  }

  /**
   * Handle speed change
   */
  private handleSpeedChange(e: Event): void {
    const select = e.target as HTMLSelectElement;
    this.playbackSpeed = parseInt(select.value, 10) as PlaybackSpeed;
  }

  /**
   * Handle interval change
   */
  private handleIntervalChange(e: Event): void {
    const select = e.target as HTMLSelectElement;
    this.animationInterval = select.value as AnimationInterval;
  }

  /**
   * Update UI elements
   */
  private updateUI(): void {
    const progress = this.totalFrames > 1
      ? (this.currentFrame / (this.totalFrames - 1)) * 100
      : 0;

    // Update slider
    const slider = document.getElementById('chart-timeline-slider') as HTMLInputElement;
    if (slider) {
      slider.value = progress.toString();
      slider.setAttribute('aria-valuenow', progress.toString());
    }

    // Update progress bar
    const progressBar = document.getElementById('chart-timeline-progress-bar');
    if (progressBar) {
      progressBar.style.width = `${progress}%`;
    }

    // Update frame count
    const frameCount = document.getElementById('chart-timeline-frame-count');
    if (frameCount) {
      frameCount.textContent = `${this.currentFrame + 1} / ${this.totalFrames}`;
    }

    // Update current date
    const currentDate = document.getElementById('chart-timeline-current-date');
    if (currentDate && this.timeSeriesData[this.currentFrame]) {
      const date = new Date(this.timeSeriesData[this.currentFrame].date);
      currentDate.textContent = this.formatDate(date);
    }

    // Update progress bar ARIA
    const progressContainer = document.querySelector('.chart-timeline-progress');
    if (progressContainer) {
      progressContainer.setAttribute('aria-valuenow', progress.toString());
    }
  }

  /**
   * Update registered charts with current frame data
   */
  private updateCharts(): void {
    if (!this.timeSeriesData[this.currentFrame]) return;

    const currentData = this.timeSeriesData[this.currentFrame];

    for (const [chartId, config] of this.registeredCharts) {
      try {
        const chartInstance = config.chartInstance as {
          setOption: (opt: unknown, opts?: { notMerge?: boolean }) => void;
          getOption: () => Record<string, unknown>;
        };

        if (chartInstance && typeof chartInstance.setOption === 'function') {
          // Get current chart option
          const option = chartInstance.getOption();

          // Update series data with animation
          if (option && option.series && Array.isArray(option.series)) {
            const series = option.series as Array<{ data?: number[] }>;

            for (const s of series) {
              if (s.data && config.dataKey && currentData.values[config.dataKey] !== undefined) {
                // For cumulative charts, show data up to current frame
                const cumulativeData = this.timeSeriesData
                  .slice(0, this.currentFrame + 1)
                  .map(d => d.values[config.dataKey] || 0);
                s.data = cumulativeData;
              }
            }

            chartInstance.setOption({
              series: option.series,
              animation: true,
              animationDuration: 300,
              animationEasing: 'cubicOut'
            });
          }
        }
      } catch (error) {
        logger.error(`Failed to update chart ${chartId}:`, error);
      }
    }
  }

  /**
   * Update play/pause button state
   */
  private updatePlayPauseButton(isPlaying: boolean): void {
    const playIcon = document.getElementById('chart-timeline-play-icon');
    const pauseIcon = document.getElementById('chart-timeline-pause-icon');
    const playPauseBtn = document.getElementById('chart-timeline-play-pause');

    if (playIcon && pauseIcon) {
      playIcon.style.display = isPlaying ? 'none' : 'inline';
      pauseIcon.style.display = isPlaying ? 'inline' : 'none';
    }

    if (playPauseBtn) {
      playPauseBtn.setAttribute('aria-label', isPlaying ? 'Pause timeline animation' : 'Play timeline animation');
    }
  }

  /**
   * Update status display
   */
  private updateStatus(message: string): void {
    const status = document.getElementById('chart-timeline-status');
    if (status) {
      status.textContent = message;
    }
  }

  /**
   * Format date for display
   */
  private formatDate(date: Date): string {
    const year = date.getFullYear();
    const month = String(date.getMonth() + 1).padStart(2, '0');
    const day = String(date.getDate()).padStart(2, '0');
    return `${year}-${month}-${day}`;
  }

  /**
   * Get current frame
   */
  getCurrentFrame(): number {
    return this.currentFrame;
  }

  /**
   * Get total frames
   */
  getTotalFrames(): number {
    return this.totalFrames;
  }

  /**
   * Get current data point
   */
  getCurrentDataPoint(): TimeSeriesDataPoint | null {
    return this.timeSeriesData[this.currentFrame] || null;
  }

  /**
   * Check if currently playing
   */
  isCurrentlyPlaying(): boolean {
    return this.isPlaying;
  }

  /**
   * Dispose resources (alias for destroy)
   */
  dispose(): void {
    this.destroy();
  }

  /**
   * Clean up event listeners and resources
   */
  destroy(): void {
    // Stop any running animation
    this.pause();

    // Abort all event listeners
    if (this.abortController) {
      this.abortController.abort();
      this.abortController = null;
    }

    // Clear registered charts
    this.registeredCharts.clear();

    // Clear data
    this.timeSeriesData = [];
    this.callbacks = {};

    // Clear container
    const container = document.getElementById(this.containerId);
    if (container) {
      container.innerHTML = '';
    }

    logger.debug('ChartTimelineAnimationManager destroyed');
  }
}

export default ChartTimelineAnimationManager;
