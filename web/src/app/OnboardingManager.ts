// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Onboarding Manager
 * Handles first-time user onboarding experience including:
 * - Welcome modal display
 * - Tour navigation
 * - Progress persistence
 * - Keyboard accessibility
 */

import { SafeStorage } from '../lib/utils/SafeStorage';

export interface OnboardingStep {
  title: string;
  description: string;
  targetSelector: string;
  position: 'top' | 'bottom' | 'left' | 'right';
}

const TOUR_STEPS: OnboardingStep[] = [
  // Step 1: Map Overview
  {
    title: 'Interactive Map',
    description: 'View your playback locations on an interactive map. Switch between 2D, 3D globe, heatmap, and hexagon visualizations using the controls above.',
    targetSelector: '#map',
    position: 'bottom'
  },
  // Step 2: Navigation
  {
    title: 'Dashboard Navigation',
    description: 'Navigate between 8 main sections: Maps, Live Activity, Analytics, Recently Added, Server, Cross-Platform, Data Governance, and Newsletter.',
    targetSelector: '.navigation-tabs',
    position: 'bottom'
  },
  // Step 3: Filters
  {
    title: 'Filter Your Data',
    description: 'Filter by date range, users, platforms, media types, and more. Save your favorite filter combinations as presets.',
    targetSelector: '#filter-panel',
    position: 'right'
  },
  // Step 4: Analytics
  {
    title: 'Analytics Dashboard',
    description: 'Explore 47 charts across 10 analytics pages. View content trends, user behavior, performance metrics, geographic distribution, and annual wrapped reports.',
    targetSelector: '#tab-analytics',
    position: 'bottom'
  },
  // Step 5: Export
  {
    title: 'Export Your Data',
    description: 'Export data in CSV, GeoJSON, or GeoParquet formats. Export individual charts as images for reports and presentations.',
    targetSelector: '#btn-export-csv',
    position: 'bottom'
  },
  // Step 6: Server Management
  {
    title: 'Server Management',
    description: 'View connection status for Tautulli, Plex, Jellyfin, and Emby. Trigger manual syncs and monitor server health.',
    targetSelector: '#tab-server',
    position: 'bottom'
  },
  // Step 7: Data Governance
  {
    title: 'Data Governance',
    description: 'Manage backups, data retention policies, GDPR compliance, audit logs, and data quality. Your data, your control.',
    targetSelector: '#tab-data-governance',
    position: 'bottom'
  },
  // Step 8: Theme & Shortcuts
  {
    title: 'Theme and Keyboard Shortcuts',
    description: 'Choose dark, light, or high-contrast themes. Press ? anytime to view all keyboard shortcuts for faster navigation.',
    targetSelector: '#theme-toggle',
    position: 'bottom'
  }
];

export class OnboardingManager {
  private modal: HTMLElement | null = null;
  private tooltip: HTMLElement | null = null;
  private currentStep: number = 0;
  private isActive: boolean = false;
  // AbortController for clean event listener removal
  private abortController: AbortController | null = null;

  constructor() {
    this.createModal();
    this.createTooltip();
    this.setupEventListeners();
  }

  /**
   * Initialize onboarding - check if should show
   */
  init(): void {
    if (this.shouldShowOnboarding()) {
      this.showWelcomeModal();
    }
  }

  /**
   * Check if onboarding should be shown
   */
  private shouldShowOnboarding(): boolean {
    const completed = SafeStorage.getItem('onboarding_completed');
    const skipped = SafeStorage.getItem('onboarding_skipped');
    return completed !== 'true' && skipped !== 'true';
  }

  /**
   * Create the welcome modal
   */
  private createModal(): void {
    const modal = document.createElement('div');
    modal.id = 'onboarding-modal';
    modal.className = 'onboarding-modal';
    modal.setAttribute('role', 'dialog');
    modal.setAttribute('aria-modal', 'true');
    modal.setAttribute('aria-labelledby', 'onboarding-title');
    modal.style.display = 'none';

    modal.innerHTML = `
      <div class="onboarding-modal-content">
        <div class="onboarding-header">
          <h2 id="onboarding-title" class="onboarding-title">Welcome to Cartographus</h2>
          <p class="onboarding-subtitle">Your geographic visualization dashboard for Plex media server activity</p>
        </div>

        <div class="onboarding-features">
          <div class="onboarding-feature">
            <div class="feature-icon" aria-hidden="true">
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M21 10c0 7-9 13-9 13s-9-6-9-13a9 9 0 0 1 18 0z"/>
                <circle cx="12" cy="10" r="3"/>
              </svg>
            </div>
            <div class="feature-text">
              <strong>Interactive Maps</strong>
              <span>View playback locations on 2D and 3D maps</span>
            </div>
          </div>
          <div class="onboarding-feature">
            <div class="feature-icon" aria-hidden="true">
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M18 20V10M12 20V4M6 20v-6"/>
              </svg>
            </div>
            <div class="feature-text">
              <strong>Rich Analytics</strong>
              <span>47 charts across 6 analytics pages</span>
            </div>
          </div>
          <div class="onboarding-feature">
            <div class="feature-icon" aria-hidden="true">
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"/>
              </svg>
            </div>
            <div class="feature-text">
              <strong>Real-Time Updates</strong>
              <span>Live activity monitoring via WebSocket</span>
            </div>
          </div>
          <div class="onboarding-feature">
            <div class="feature-icon" aria-hidden="true">
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <circle cx="12" cy="12" r="10"/>
                <circle cx="12" cy="12" r="4"/>
                <line x1="21.17" y1="8" x2="12" y2="8"/>
                <line x1="3.95" y1="6.06" x2="8.54" y2="14"/>
                <line x1="10.88" y1="21.94" x2="15.46" y2="14"/>
              </svg>
            </div>
            <div class="feature-text">
              <strong>Customizable Themes</strong>
              <span>Dark, light, and high-contrast modes</span>
            </div>
          </div>
        </div>

        <div class="onboarding-actions">
          <button id="onboarding-start-btn" class="btn btn-primary">
            Get Started
          </button>
          <button id="onboarding-skip-btn" class="btn btn-secondary">
            Skip for Now
          </button>
        </div>

        <p class="onboarding-hint">Press <kbd>?</kbd> anytime to see keyboard shortcuts</p>
      </div>
    `;

    document.body.appendChild(modal);
    this.modal = modal;
  }

  /**
   * Create the tour tooltip element
   */
  private createTooltip(): void {
    const tooltip = document.createElement('div');
    tooltip.className = 'onboarding-tooltip';
    tooltip.style.display = 'none';

    tooltip.innerHTML = `
      <div class="tooltip-content">
        <div class="tooltip-header">
          <span class="tooltip-step-indicator"></span>
          <button class="tooltip-close" aria-label="Close tour">&times;</button>
        </div>
        <h3 class="tooltip-title"></h3>
        <p class="tooltip-description"></p>
        <div class="tooltip-navigation">
          <button class="onboarding-prev-btn btn btn-sm">Previous</button>
          <button class="onboarding-next-btn btn btn-sm btn-primary">Next</button>
        </div>
      </div>
      <div class="tooltip-arrow"></div>
    `;

    document.body.appendChild(tooltip);
    this.tooltip = tooltip;
  }

  /**
   * Setup event listeners with AbortController for clean removal
   */
  private setupEventListeners(): void {
    // Create AbortController for cleanup
    this.abortController = new AbortController();
    const signal = this.abortController.signal;

    // Start tour button
    document.getElementById('onboarding-start-btn')?.addEventListener('click', () => {
      this.hideWelcomeModal();
      this.startTour();
    }, { signal });

    // Skip button
    document.getElementById('onboarding-skip-btn')?.addEventListener('click', () => {
      this.skipOnboarding();
    }, { signal });

    // Tooltip navigation
    this.tooltip?.querySelector('.onboarding-prev-btn')?.addEventListener('click', () => {
      this.previousStep();
    }, { signal });

    this.tooltip?.querySelector('.onboarding-next-btn')?.addEventListener('click', () => {
      this.nextStep();
    }, { signal });

    this.tooltip?.querySelector('.tooltip-close')?.addEventListener('click', () => {
      this.endTour();
    }, { signal });

    // Keyboard navigation
    document.addEventListener('keydown', (e) => {
      if (this.modal?.style.display === 'flex') {
        if (e.key === 'Escape') {
          this.skipOnboarding();
        }
      } else if (this.isActive) {
        if (e.key === 'Escape') {
          this.endTour();
        } else if (e.key === 'ArrowRight') {
          this.nextStep();
        } else if (e.key === 'ArrowLeft') {
          this.previousStep();
        }
      }
    }, { signal });
  }

  /**
   * Show the welcome modal
   */
  showWelcomeModal(): void {
    if (this.modal) {
      this.modal.style.display = 'flex';
      // Focus on start button
      setTimeout(() => {
        document.getElementById('onboarding-start-btn')?.focus();
      }, 100);
    }
  }

  /**
   * Hide the welcome modal
   */
  private hideWelcomeModal(): void {
    if (this.modal) {
      this.modal.style.display = 'none';
    }
  }

  /**
   * Skip onboarding
   */
  private skipOnboarding(): void {
    SafeStorage.setItem('onboarding_skipped', 'true');
    this.hideWelcomeModal();
  }

  /**
   * Start the tour
   */
  startTour(): void {
    this.isActive = true;
    this.currentStep = 0;
    this.showStep(0);
  }

  /**
   * Show a specific tour step
   */
  private showStep(stepIndex: number): void {
    if (stepIndex < 0 || stepIndex >= TOUR_STEPS.length) {
      this.completeTour();
      return;
    }

    const step = TOUR_STEPS[stepIndex];
    const target = document.querySelector(step.targetSelector);

    if (!target || !this.tooltip) {
      // Skip to next step if target not found
      this.currentStep++;
      this.showStep(this.currentStep);
      return;
    }

    // Update tooltip content
    const title = this.tooltip.querySelector('.tooltip-title');
    const description = this.tooltip.querySelector('.tooltip-description');
    const stepIndicator = this.tooltip.querySelector('.tooltip-step-indicator');
    const prevBtn = this.tooltip.querySelector('.onboarding-prev-btn') as HTMLElement;
    const nextBtn = this.tooltip.querySelector('.onboarding-next-btn');

    if (title) title.textContent = step.title;
    if (description) description.textContent = step.description;
    if (stepIndicator) stepIndicator.textContent = `Step ${stepIndex + 1} of ${TOUR_STEPS.length}`;

    // Show/hide previous button
    if (prevBtn) {
      prevBtn.style.display = stepIndex === 0 ? 'none' : 'inline-block';
    }

    // Change next button text on last step
    if (nextBtn) {
      nextBtn.textContent = stepIndex === TOUR_STEPS.length - 1 ? 'Finish' : 'Next';
    }

    // Position tooltip
    this.positionTooltip(target as HTMLElement, step.position);

    // Show tooltip
    this.tooltip.style.display = 'block';

    // Highlight target
    this.highlightElement(target as HTMLElement);
  }

  /**
   * Position the tooltip relative to target
   */
  private positionTooltip(target: HTMLElement, position: OnboardingStep['position']): void {
    if (!this.tooltip) return;

    const targetRect = target.getBoundingClientRect();
    const tooltipRect = this.tooltip.getBoundingClientRect();
    const padding = 16;

    let top = 0;
    let left = 0;

    switch (position) {
      case 'top':
        top = targetRect.top - tooltipRect.height - padding;
        left = targetRect.left + (targetRect.width - tooltipRect.width) / 2;
        break;
      case 'bottom':
        top = targetRect.bottom + padding;
        left = targetRect.left + (targetRect.width - tooltipRect.width) / 2;
        break;
      case 'left':
        top = targetRect.top + (targetRect.height - tooltipRect.height) / 2;
        left = targetRect.left - tooltipRect.width - padding;
        break;
      case 'right':
        top = targetRect.top + (targetRect.height - tooltipRect.height) / 2;
        left = targetRect.right + padding;
        break;
    }

    // Ensure tooltip stays within viewport
    const maxLeft = window.innerWidth - tooltipRect.width - padding;
    const maxTop = window.innerHeight - tooltipRect.height - padding;

    top = Math.max(padding, Math.min(top, maxTop));
    left = Math.max(padding, Math.min(left, maxLeft));

    this.tooltip.style.top = `${top}px`;
    this.tooltip.style.left = `${left}px`;
    this.tooltip.setAttribute('data-position', position);
  }

  /**
   * Highlight an element during tour
   */
  private highlightElement(element: HTMLElement): void {
    // Remove previous highlights
    document.querySelectorAll('.onboarding-highlight').forEach(el => {
      el.classList.remove('onboarding-highlight');
    });

    // Add highlight to current element
    element.classList.add('onboarding-highlight');

    // Scroll element into view if needed
    element.scrollIntoView({ behavior: 'smooth', block: 'center' });
  }

  /**
   * Go to next step
   */
  nextStep(): void {
    this.currentStep++;
    this.showStep(this.currentStep);
  }

  /**
   * Go to previous step
   */
  previousStep(): void {
    if (this.currentStep > 0) {
      this.currentStep--;
      this.showStep(this.currentStep);
    }
  }

  /**
   * Complete the tour
   */
  private completeTour(): void {
    SafeStorage.setItem('onboarding_completed', 'true');
    this.endTour();
  }

  /**
   * End the tour (whether completed or exited early)
   */
  private endTour(): void {
    this.isActive = false;

    // Hide tooltip
    if (this.tooltip) {
      this.tooltip.style.display = 'none';
    }

    // Remove highlights
    document.querySelectorAll('.onboarding-highlight').forEach(el => {
      el.classList.remove('onboarding-highlight');
    });
  }

  /**
   * Restart tour (for help button)
   */
  restartTour(): void {
    this.showWelcomeModal();
  }

  /**
   * Clean up event listeners and resources
   */
  destroy(): void {
    // Abort all event listeners
    if (this.abortController) {
      this.abortController.abort();
      this.abortController = null;
    }

    // Remove highlights
    document.querySelectorAll('.onboarding-highlight').forEach(el => {
      el.classList.remove('onboarding-highlight');
    });

    // Remove modal
    if (this.modal) {
      this.modal.remove();
      this.modal = null;
    }

    // Remove tooltip
    if (this.tooltip) {
      this.tooltip.remove();
      this.tooltip = null;
    }

    this.isActive = false;
  }
}
