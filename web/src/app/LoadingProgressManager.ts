// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * LoadingProgressManager - Tracks and displays loading progress percentage
 *
 * Features:
 * - Tracks multiple concurrent loading operations
 * - Calculates actual percentage based on completed operations
 * - Smooth progress animation with easing
 * - ARIA-compliant progress bar updates
 * - Supports weighted operations for more accurate progress
 */

import { createLogger } from '../lib/logger';

const logger = createLogger('LoadingProgressManager');

/**
 * Loading operation with optional weight
 */
interface LoadingOperation {
  id: string;
  label: string;
  weight: number;
  completed: boolean;
  startTime: number;
}

/**
 * Configuration for loading progress
 */
interface LoadingProgressConfig {
  /** Minimum time to show progress (prevents flash) */
  minDisplayTime: number;
  /** Animation interval for smooth progress */
  animationInterval: number;
  /** Progress bar element ID */
  progressBarId: string;
}

const DEFAULT_CONFIG: LoadingProgressConfig = {
  minDisplayTime: 500,
  animationInterval: 50,
  progressBarId: 'global-progress-bar'
};

export class LoadingProgressManager {
  private config: LoadingProgressConfig;
  private operations: Map<string, LoadingOperation> = new Map();
  private isActive: boolean = false;
  private startTime: number = 0;
  private targetPercent: number = 0;
  private currentPercent: number = 0;
  private animationTimer: ReturnType<typeof setInterval> | null = null;
  private progressBar: HTMLElement | null = null;
  private progressFill: HTMLElement | null = null;
  private statusText: HTMLElement | null = null;

  constructor(config: Partial<LoadingProgressConfig> = {}) {
    this.config = { ...DEFAULT_CONFIG, ...config };
  }

  /**
   * Initialize the loading progress manager
   */
  init(): void {
    this.progressBar = document.getElementById(this.config.progressBarId);
    if (this.progressBar) {
      this.progressFill = this.progressBar.querySelector('.progress-fill');
      this.createStatusText();
    }
    logger.info('LoadingProgressManager initialized');
  }

  /**
   * Create a status text element for accessibility
   */
  private createStatusText(): void {
    if (!this.progressBar) return;

    // Check if status text already exists
    this.statusText = this.progressBar.querySelector('.progress-status-text');
    if (!this.statusText) {
      this.statusText = document.createElement('span');
      this.statusText.className = 'progress-status-text visually-hidden';
      this.statusText.setAttribute('role', 'status');
      this.statusText.setAttribute('aria-live', 'polite');
      this.progressBar.appendChild(this.statusText);
    }
  }

  /**
   * Start tracking a set of loading operations
   * Call this at the beginning of a loading sequence
   */
  startLoading(operations: Array<{ id: string; label: string; weight?: number }>): void {
    this.reset();
    this.isActive = true;
    this.startTime = Date.now();

    // Register all operations
    operations.forEach(op => {
      this.operations.set(op.id, {
        id: op.id,
        label: op.label,
        weight: op.weight ?? 1,
        completed: false,
        startTime: Date.now()
      });
    });

    // Show progress bar
    this.showProgressBar();
    this.startAnimation();

    // Set initial progress (show some movement to indicate activity)
    this.targetPercent = 5;
  }

  /**
   * Mark an operation as complete
   */
  completeOperation(operationId: string): void {
    const operation = this.operations.get(operationId);
    if (!operation) return;

    operation.completed = true;
    this.updateProgress();
  }

  /**
   * Mark all operations as complete and finish loading
   */
  finishLoading(): void {
    // Mark all operations complete
    this.operations.forEach(op => {
      op.completed = true;
    });

    this.targetPercent = 100;

    // Ensure minimum display time
    const elapsed = Date.now() - this.startTime;
    const remainingTime = Math.max(0, this.config.minDisplayTime - elapsed);

    setTimeout(() => {
      this.hideProgressBar();
    }, remainingTime + 300); // Add 300ms for animation to reach 100%
  }

  /**
   * Cancel loading and hide progress bar
   */
  cancelLoading(): void {
    this.reset();
    this.hideProgressBar();
  }

  /**
   * Update progress based on completed operations
   */
  private updateProgress(): void {
    if (!this.isActive) return;

    const totalWeight = Array.from(this.operations.values())
      .reduce((sum, op) => sum + op.weight, 0);

    const completedWeight = Array.from(this.operations.values())
      .filter(op => op.completed)
      .reduce((sum, op) => sum + op.weight, 0);

    // Calculate percentage (reserve 5% at start and 5% at end for visual smoothness)
    const baseProgress = totalWeight > 0 ? (completedWeight / totalWeight) : 0;
    this.targetPercent = 5 + (baseProgress * 90);

    // Update accessibility status
    this.updateStatusText();
  }

  /**
   * Update the accessible status text
   */
  private updateStatusText(): void {
    if (!this.statusText) return;

    const completed = Array.from(this.operations.values()).filter(op => op.completed).length;
    const total = this.operations.size;
    const percent = Math.round(this.targetPercent);

    this.statusText.textContent = `Loading ${completed} of ${total} items, ${percent}% complete`;
  }

  /**
   * Start smooth animation towards target percentage
   */
  private startAnimation(): void {
    if (this.animationTimer) return;

    this.animationTimer = setInterval(() => {
      if (!this.isActive) {
        this.stopAnimation();
        return;
      }

      // Ease towards target
      const diff = this.targetPercent - this.currentPercent;
      if (Math.abs(diff) < 0.5) {
        this.currentPercent = this.targetPercent;
      } else {
        // Ease-out animation
        this.currentPercent += diff * 0.15;
      }

      this.renderProgress();

      // Check if we've reached 100% and animation should stop
      if (this.currentPercent >= 99.5 && this.targetPercent >= 100) {
        this.currentPercent = 100;
        this.renderProgress();
        this.stopAnimation();
      }
    }, this.config.animationInterval);
  }

  /**
   * Stop the animation timer
   */
  private stopAnimation(): void {
    if (this.animationTimer) {
      clearInterval(this.animationTimer);
      this.animationTimer = null;
    }
  }

  /**
   * Render the current progress to the DOM
   */
  private renderProgress(): void {
    if (!this.progressBar || !this.progressFill) return;

    const percent = Math.round(this.currentPercent);
    this.progressFill.style.width = `${percent}%`;
    this.progressBar.setAttribute('aria-valuenow', String(percent));

    // Update progress bar class for visual styling
    this.progressBar.classList.add('determinate');
  }

  /**
   * Show the progress bar
   */
  private showProgressBar(): void {
    if (!this.progressBar) return;

    this.progressBar.classList.remove('hidden');
    this.progressBar.classList.add('determinate');

    // Reset fill width
    if (this.progressFill) {
      this.progressFill.style.width = '0%';
    }
  }

  /**
   * Hide the progress bar
   */
  private hideProgressBar(): void {
    if (!this.progressBar) return;

    this.progressBar.classList.add('hidden');
    this.progressBar.classList.remove('determinate');

    // Reset state
    this.isActive = false;
    this.stopAnimation();
  }

  /**
   * Reset all state
   */
  private reset(): void {
    this.operations.clear();
    this.isActive = false;
    this.startTime = 0;
    this.targetPercent = 0;
    this.currentPercent = 0;
    this.stopAnimation();
  }

  /**
   * Get current loading state
   */
  getState(): { isActive: boolean; percent: number; operations: number; completed: number } {
    const completed = Array.from(this.operations.values()).filter(op => op.completed).length;
    return {
      isActive: this.isActive,
      percent: Math.round(this.currentPercent),
      operations: this.operations.size,
      completed
    };
  }

  /**
   * Utility: Wrap a promise to automatically track its completion
   */
  trackOperation<T>(operationId: string, promise: Promise<T>): Promise<T> {
    return promise
      .then(result => {
        this.completeOperation(operationId);
        return result;
      })
      .catch(error => {
        this.completeOperation(operationId);
        throw error;
      });
  }

  /**
   * Utility: Create tracked promises for common operations
   */
  createTrackedLoader(operationId: string, label: string, weight: number = 1): {
    start: () => void;
    complete: () => void;
    operation: { id: string; label: string; weight: number };
  } {
    return {
      operation: { id: operationId, label, weight },
      start: () => {
        // Operation is registered via startLoading
      },
      complete: () => this.completeOperation(operationId)
    };
  }

  /**
   * Clean up timers and resources
   */
  destroy(): void {
    this.reset();
    this.progressBar = null;
    this.progressFill = null;
    this.statusText = null;
  }
}

export default LoadingProgressManager;
