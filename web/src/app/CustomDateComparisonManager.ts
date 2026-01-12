// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * CustomDateComparisonManager - Enables arbitrary date range comparison
 *
 * Features:
 * - Custom date picker inputs for period selection
 * - Compare any two date ranges
 * - Shows/hides date pickers based on period selection
 * - Validates date ranges
 * - Integrates with ComparativeInsightsManager
 */

import type { API, LocationFilter } from '../lib/api';
import { createLogger } from '../lib/logger';
import type { ComparativeAnalyticsResponse } from '../lib/types/analytics';

const logger = createLogger('CustomDateComparison');

export interface CustomDateRange {
  startDate: string;
  endDate: string;
}

export class CustomDateComparisonManager {
  private api: API;
  private isLoading: boolean = false;
  private periodSelect: HTMLSelectElement | null = null;
  private customDateContainer: HTMLElement | null = null;
  private startDateInput: HTMLInputElement | null = null;
  private endDateInput: HTMLInputElement | null = null;
  private applyButton: HTMLButtonElement | null = null;
  private onComparisonLoad: ((data: ComparativeAnalyticsResponse) => void) | null = null;
  // AbortController for clean event listener removal
  private abortController: AbortController | null = null;
  // Track active timeouts for cleanup
  private activeTimeouts: Set<ReturnType<typeof setTimeout>> = new Set();

  constructor(api: API) {
    this.api = api;
  }

  /**
   * Initialize the custom date comparison manager
   */
  init(): void {
    this.setupElements();
    this.setupEventListeners();
    this.setDefaultDates();
    logger.debug('[CustomDateComparison] CustomDateComparisonManager initialized');
  }

  /**
   * Set callback for when comparison data is loaded
   */
  setOnComparisonLoad(callback: (data: ComparativeAnalyticsResponse) => void): void {
    this.onComparisonLoad = callback;
  }

  /**
   * Set up DOM element references
   */
  private setupElements(): void {
    this.periodSelect = document.getElementById('comparison-period-select') as HTMLSelectElement;
    this.customDateContainer = document.getElementById('custom-date-container');
    this.startDateInput = document.getElementById('comparison-start-date') as HTMLInputElement;
    this.endDateInput = document.getElementById('comparison-end-date') as HTMLInputElement;
    this.applyButton = document.getElementById('apply-custom-comparison') as HTMLButtonElement;
  }

  /**
   * Set up event listeners with AbortController for clean removal
   */
  private setupEventListeners(): void {
    // Create AbortController for cleanup
    this.abortController = new AbortController();
    const signal = this.abortController.signal;

    // Period select change - show/hide custom date pickers
    if (this.periodSelect) {
      this.periodSelect.addEventListener('change', (e) => {
        const target = e.target as HTMLSelectElement;
        this.handlePeriodChange(target.value);
      }, { signal });
    }

    // Apply button click
    if (this.applyButton) {
      this.applyButton.addEventListener('click', () => {
        this.applyCustomComparison();
      }, { signal });
    }

    // Date input changes - validate and enable/disable apply button
    if (this.startDateInput) {
      this.startDateInput.addEventListener('change', () => this.validateDates(), { signal });
    }
    if (this.endDateInput) {
      this.endDateInput.addEventListener('change', () => this.validateDates(), { signal });
    }

    // Enter key on date inputs
    if (this.startDateInput) {
      this.startDateInput.addEventListener('keypress', (e) => {
        if (e.key === 'Enter') this.applyCustomComparison();
      }, { signal });
    }
    if (this.endDateInput) {
      this.endDateInput.addEventListener('keypress', (e) => {
        if (e.key === 'Enter') this.applyCustomComparison();
      }, { signal });
    }
  }

  /**
   * Set default dates (last 30 days)
   */
  private setDefaultDates(): void {
    const today = new Date();
    const thirtyDaysAgo = new Date();
    thirtyDaysAgo.setDate(today.getDate() - 30);

    if (this.endDateInput) {
      this.endDateInput.value = this.formatDateForInput(today);
      this.endDateInput.max = this.formatDateForInput(today);
    }
    if (this.startDateInput) {
      this.startDateInput.value = this.formatDateForInput(thirtyDaysAgo);
      this.startDateInput.max = this.formatDateForInput(today);
    }
  }

  /**
   * Format date for input value (YYYY-MM-DD)
   */
  private formatDateForInput(date: Date): string {
    return date.toISOString().split('T')[0];
  }

  /**
   * Handle period selection change
   */
  private handlePeriodChange(period: string): void {
    if (this.customDateContainer) {
      if (period === 'custom') {
        this.customDateContainer.classList.remove('hidden');
        this.customDateContainer.setAttribute('aria-hidden', 'false');
        // Focus on start date input for accessibility
        if (this.startDateInput) {
          setTimeout(() => this.startDateInput?.focus(), 100);
        }
      } else {
        this.customDateContainer.classList.add('hidden');
        this.customDateContainer.setAttribute('aria-hidden', 'true');
      }
    }
  }

  /**
   * Validate date inputs
   */
  private validateDates(): boolean {
    if (!this.startDateInput || !this.endDateInput) {
      return false;
    }

    const startDate = this.startDateInput.value;
    const endDate = this.endDateInput.value;

    // Check both dates are filled
    if (!startDate || !endDate) {
      this.setValidationState(false, 'Please select both start and end dates');
      return false;
    }

    // Check start is before end
    const start = new Date(startDate);
    const end = new Date(endDate);
    if (start >= end) {
      this.setValidationState(false, 'Start date must be before end date');
      return false;
    }

    // Check range is not too large (max 365 days)
    const daysDiff = Math.ceil((end.getTime() - start.getTime()) / (1000 * 60 * 60 * 24));
    if (daysDiff > 365) {
      this.setValidationState(false, 'Date range cannot exceed 365 days');
      return false;
    }

    this.setValidationState(true, '');
    return true;
  }

  /**
   * Set validation state on inputs and button
   */
  private setValidationState(isValid: boolean, message: string): void {
    if (this.applyButton) {
      this.applyButton.disabled = !isValid;
    }

    const errorEl = document.getElementById('custom-date-error');
    if (errorEl) {
      errorEl.textContent = message;
      errorEl.classList.toggle('hidden', isValid);
    }

    // Add/remove error styling on inputs
    if (this.startDateInput) {
      this.startDateInput.classList.toggle('input-error', !isValid && !!message);
    }
    if (this.endDateInput) {
      this.endDateInput.classList.toggle('input-error', !isValid && !!message);
    }
  }

  /**
   * Apply custom date comparison
   */
  async applyCustomComparison(): Promise<void> {
    if (!this.validateDates() || this.isLoading) {
      return;
    }

    if (!this.startDateInput || !this.endDateInput) {
      return;
    }

    this.isLoading = true;
    this.setLoadingState(true);

    try {
      const filter: LocationFilter = {
        start_date: this.startDateInput.value,
        end_date: this.endDateInput.value
      };

      const data = await this.api.getAnalyticsComparative(filter, 'custom');

      // Call the callback with the data
      if (this.onComparisonLoad) {
        this.onComparisonLoad(data);
      }

      // Show success feedback
      this.showSuccessFeedback();
    } catch (error) {
      logger.error('[CustomDateComparison] Failed to load custom comparison:', error);
      this.showErrorFeedback();
    } finally {
      this.isLoading = false;
      this.setLoadingState(false);
    }
  }

  /**
   * Set loading state on UI elements
   */
  private setLoadingState(loading: boolean): void {
    if (this.applyButton) {
      this.applyButton.disabled = loading;
      this.applyButton.textContent = loading ? 'Loading...' : 'Apply';
    }
    if (this.startDateInput) {
      this.startDateInput.disabled = loading;
    }
    if (this.endDateInput) {
      this.endDateInput.disabled = loading;
    }
  }

  /**
   * Show success feedback
   */
  private showSuccessFeedback(): void {
    if (this.applyButton) {
      const originalText = this.applyButton.textContent;
      this.applyButton.textContent = 'Applied';
      this.applyButton.classList.add('success');
      const timeoutId = setTimeout(() => {
        this.activeTimeouts.delete(timeoutId);
        if (this.applyButton) {
          this.applyButton.textContent = originalText || 'Apply';
          this.applyButton.classList.remove('success');
        }
      }, 2000);
      this.activeTimeouts.add(timeoutId);
    }
  }

  /**
   * Show error feedback
   */
  private showErrorFeedback(): void {
    const errorEl = document.getElementById('custom-date-error');
    if (errorEl) {
      errorEl.textContent = 'Failed to load comparison data';
      errorEl.classList.remove('hidden');
    }
  }

  /**
   * Get the current custom date range
   */
  getCustomDateRange(): CustomDateRange | null {
    if (!this.startDateInput || !this.endDateInput) {
      return null;
    }
    if (!this.startDateInput.value || !this.endDateInput.value) {
      return null;
    }
    return {
      startDate: this.startDateInput.value,
      endDate: this.endDateInput.value
    };
  }

  /**
   * Set custom date range programmatically
   */
  setCustomDateRange(startDate: string, endDate: string): void {
    if (this.startDateInput) {
      this.startDateInput.value = startDate;
    }
    if (this.endDateInput) {
      this.endDateInput.value = endDate;
    }

    // Switch to custom mode
    if (this.periodSelect) {
      this.periodSelect.value = 'custom';
      this.handlePeriodChange('custom');
    }

    this.validateDates();
  }

  /**
   * Check if custom mode is active
   */
  isCustomModeActive(): boolean {
    return this.periodSelect?.value === 'custom';
  }

  /**
   * Clean up event listeners and timers
   */
  destroy(): void {
    // Abort all event listeners
    if (this.abortController) {
      this.abortController.abort();
      this.abortController = null;
    }

    // Clear all active timeouts
    for (const timeoutId of this.activeTimeouts) {
      clearTimeout(timeoutId);
    }
    this.activeTimeouts.clear();

    // Clear DOM references
    this.periodSelect = null;
    this.customDateContainer = null;
    this.startDateInput = null;
    this.endDateInput = null;
    this.applyButton = null;
    this.onComparisonLoad = null;

    logger.debug('[CustomDateComparison] CustomDateComparisonManager destroyed');
  }
}

export default CustomDateComparisonManager;
