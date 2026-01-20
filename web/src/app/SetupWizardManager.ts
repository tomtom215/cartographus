// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * SetupWizardManager - Multi-step Setup Wizard for First-Time Users
 *
 * This manager guides users through initial configuration by checking:
 * - Database connectivity
 * - Tautulli configuration and connectivity
 * - Media server detection (Plex, Jellyfin, Emby)
 * - Data availability
 * - Provides recommendations
 *
 * Features:
 * - Fetches status from /api/v1/health/setup
 * - Multi-step wizard with progress indicator
 * - WCAG 2.1 AA accessible with keyboard navigation
 * - Deterministic for E2E testing via localStorage flags
 * - SafeStorage for private browsing fallback
 * - Race condition guards for concurrent calls
 * - AbortController for fetch cancellation
 */

import { SafeStorage } from '../lib/utils/SafeStorage';
import { queuedFetch } from '../lib/api/client';
import { createLogger } from '../lib/logger';

const logger = createLogger('SetupWizard');

/**
 * Setup status from API
 */
export interface SetupStatus {
  ready: boolean;
  database: {
    connected: boolean;
  };
  data_sources: {
    tautulli: {
      configured: boolean;
      connected: boolean;
      url?: string;
      error?: string;
    };
    plex: {
      configured: boolean;
      connected?: boolean;
      server_count?: number;
      error?: string;
    };
    jellyfin: {
      configured: boolean;
      connected?: boolean;
      server_count?: number;
      error?: string;
    };
    emby: {
      configured: boolean;
      connected?: boolean;
      server_count?: number;
      error?: string;
    };
    nats: {
      enabled: boolean;
      connected?: boolean;
      error?: string;
    };
  };
  data_available: {
    has_playbacks: boolean;
    playback_count: number;
    has_geolocations: boolean;
  };
  recommendations?: string[];
}

/**
 * Wizard step definition
 */
interface WizardStep {
  id: string;
  title: string;
  icon: string;
  render: (status: SetupStatus) => string;
  validate?: (status: SetupStatus) => boolean;
}

/**
 * Callbacks for wizard actions
 */
export interface SetupWizardCallbacks {
  onComplete: () => void;
  onSkip: () => void;
  onStartTour: () => void;
  onTriggerSync: () => void;
}

/**
 * SetupWizardManager class
 */
export class SetupWizardManager {
  private modal: HTMLElement | null = null;
  private currentStep: number = 0;
  private status: SetupStatus | null = null;
  private callbacks: SetupWizardCallbacks | null = null;
  private abortController: AbortController | null = null;
  private fetchAbortController: AbortController | null = null;
  private isVisible: boolean = false;
  private isShowing: boolean = false; // Guard against concurrent show() calls
  private isInitialized: boolean = false;
  private isDestroyed: boolean = false;

  private readonly steps: WizardStep[] = [
    {
      id: 'welcome',
      title: 'Welcome',
      icon: '1',
      render: this.renderWelcomeStep.bind(this)
    },
    {
      id: 'database',
      title: 'Database',
      icon: '2',
      render: this.renderDatabaseStep.bind(this),
      validate: (status) => status.database.connected
    },
    {
      id: 'data-sources',
      title: 'Data Sources',
      icon: '3',
      render: this.renderDataSourcesStep.bind(this),
      validate: (status) => this.hasAnyDataSource(status)
    },
    {
      id: 'data',
      title: 'Data',
      icon: '4',
      render: this.renderDataStep.bind(this)
    },
    {
      id: 'complete',
      title: 'Complete',
      icon: '5',
      render: this.renderCompleteStep.bind(this)
    }
  ];

  constructor() {}

  /**
   * Initialize the wizard with callbacks
   */
  init(callbacks: SetupWizardCallbacks): void {
    if (this.isDestroyed) {
      logger.warn('Cannot init - already destroyed');
      return;
    }
    if (this.isInitialized) {
      logger.warn('Already initialized');
      return;
    }

    this.callbacks = callbacks;
    this.createModal();
    this.setupEventListeners();
    this.isInitialized = true;
  }

  /**
   * Check if wizard should be shown
   */
  shouldShow(): boolean {
    // Check for E2E test override (use SafeStorage for private browsing support)
    const forceShow = SafeStorage.getItem('setup_wizard_force_show');
    if (forceShow === 'true') {
      return true;
    }

    // Check if already completed
    const completed = SafeStorage.getItem('setup_wizard_completed');
    const skipped = SafeStorage.getItem('setup_wizard_skipped');
    return completed !== 'true' && skipped !== 'true';
  }

  /**
   * Show the wizard
   * @returns true if wizard was shown, false if skipped
   */
  async show(): Promise<boolean> {
    // Guard against concurrent calls
    if (this.isShowing) {
      logger.warn('show() already in progress');
      return false;
    }

    // Guard against destroyed state
    if (this.isDestroyed) {
      logger.warn('Cannot show - already destroyed');
      return false;
    }

    // Guard against uninitialized state
    if (!this.isInitialized) {
      logger.warn('Cannot show - not initialized');
      return false;
    }

    if (!this.shouldShow()) {
      // Still fetch status in background for recommendations
      this.fetchStatus().catch(() => {
        // Silently ignore - wizard was skipped anyway
      });
      return false;
    }

    this.isShowing = true;

    try {
      this.status = await this.fetchStatus();

      // Check if destroyed during fetch
      if (this.isDestroyed) {
        this.isShowing = false;
        return false;
      }

      this.currentStep = 0;
      this.renderCurrentStep();
      this.showModal();
      return true;
    } catch (error) {
      // Check if aborted
      if (error instanceof Error && error.name === 'AbortError') {
        logger.debug('Fetch aborted');
        this.isShowing = false;
        return false;
      }

      logger.error('Failed to fetch status', { error });
      // Show wizard anyway with error state
      this.status = this.getErrorStatus();
      this.currentStep = 0;
      this.renderCurrentStep();
      this.showModal();
      return true;
    } finally {
      this.isShowing = false;
    }
  }

  /**
   * Fetch setup status from API with abort support
   */
  private async fetchStatus(): Promise<SetupStatus> {
    // Cancel any previous fetch
    if (this.fetchAbortController) {
      this.fetchAbortController.abort();
    }
    this.fetchAbortController = new AbortController();

    const response = await queuedFetch('/api/v1/health/setup', {
      signal: this.fetchAbortController.signal
    });

    if (!response.ok) {
      throw new Error(`Failed to fetch setup status: ${response.status}`);
    }

    const data = await response.json();
    return data.data as SetupStatus;
  }

  /**
   * Get error status when API fails
   */
  private getErrorStatus(): SetupStatus {
    return {
      ready: false,
      database: { connected: false },
      data_sources: {
        tautulli: { configured: false, connected: false },
        plex: { configured: false },
        jellyfin: { configured: false },
        emby: { configured: false },
        nats: { enabled: false }
      },
      data_available: {
        has_playbacks: false,
        playback_count: 0,
        has_geolocations: false
      },
      recommendations: ['Unable to connect to the server. Please check your network connection.']
    };
  }

  /**
   * Check if any data source is configured
   */
  private hasAnyDataSource(status: SetupStatus): boolean {
    const ds = status.data_sources;
    return ds.tautulli.connected ||
           ds.plex.configured ||
           ds.jellyfin.configured ||
           ds.emby.configured;
  }

  /**
   * Create the wizard modal
   */
  private createModal(): void {
    const modal = document.createElement('div');
    modal.id = 'setup-wizard-modal';
    modal.className = 'setup-wizard-overlay';
    modal.setAttribute('role', 'dialog');
    modal.setAttribute('aria-modal', 'true');
    modal.setAttribute('aria-labelledby', 'setup-wizard-title');
    modal.style.display = 'none';

    modal.innerHTML = `
      <div class="setup-wizard-content">
        <div class="setup-wizard-header">
          <h2 id="setup-wizard-title" class="setup-wizard-title">Setup Wizard</h2>
          <button class="setup-wizard-close" aria-label="Close wizard" title="Close">&times;</button>
        </div>

        <div class="setup-wizard-progress" role="navigation" aria-label="Setup progress">
          ${this.renderProgressIndicator()}
        </div>

        <div class="setup-wizard-body" role="main">
          <!-- Step content will be rendered here -->
        </div>

        <div class="setup-wizard-footer">
          <button id="wizard-prev-btn" class="btn btn-secondary" style="display: none;">
            Previous
          </button>
          <div class="wizard-footer-spacer"></div>
          <button id="wizard-skip-btn" class="btn btn-ghost">
            Skip Setup
          </button>
          <button id="wizard-next-btn" class="btn btn-primary">
            Next
          </button>
        </div>
      </div>
    `;

    document.body.appendChild(modal);
    this.modal = modal;
  }

  /**
   * Render progress indicator
   */
  private renderProgressIndicator(): string {
    return this.steps.map((step, index) => {
      const isActive = index === this.currentStep;
      const isCompleted = index < this.currentStep;
      const statusClass = isActive ? 'active' : isCompleted ? 'completed' : '';

      return `
        <div class="progress-step ${statusClass}" data-step="${index}">
          <div class="progress-step-icon" aria-hidden="true">
            ${isCompleted ? '&#10003;' : step.icon}
          </div>
          <span class="progress-step-label">${step.title}</span>
        </div>
      `;
    }).join('<div class="progress-connector" aria-hidden="true"></div>');
  }

  /**
   * Update progress indicator
   */
  private updateProgressIndicator(): void {
    const progressEl = this.modal?.querySelector('.setup-wizard-progress');
    if (progressEl) {
      progressEl.innerHTML = this.renderProgressIndicator();
    }
  }

  /**
   * Setup event listeners
   */
  private setupEventListeners(): void {
    this.abortController = new AbortController();
    const signal = this.abortController.signal;

    // Close button
    this.modal?.querySelector('.setup-wizard-close')?.addEventListener('click', () => {
      this.skip();
    }, { signal });

    // Navigation buttons
    document.getElementById('wizard-prev-btn')?.addEventListener('click', () => {
      this.previousStep();
    }, { signal });

    document.getElementById('wizard-next-btn')?.addEventListener('click', () => {
      this.nextStep();
    }, { signal });

    document.getElementById('wizard-skip-btn')?.addEventListener('click', () => {
      this.skip();
    }, { signal });

    // Keyboard navigation
    document.addEventListener('keydown', (e) => {
      if (!this.isVisible) return;

      if (e.key === 'Escape') {
        this.skip();
      } else if (e.key === 'Enter') {
        e.preventDefault();
        this.nextStep();
      }
    }, { signal });

    // Click outside to close
    this.modal?.addEventListener('click', (e) => {
      if (e.target === this.modal) {
        this.skip();
      }
    }, { signal });
  }

  /**
   * Show the modal
   */
  private showModal(): void {
    if (this.modal) {
      this.modal.style.display = 'flex';
      this.isVisible = true;

      // Focus on next button
      setTimeout(() => {
        document.getElementById('wizard-next-btn')?.focus();
      }, 100);
    }
  }

  /**
   * Hide the modal
   */
  private hideModal(): void {
    if (this.modal) {
      this.modal.style.display = 'none';
      this.isVisible = false;
    }
  }

  /**
   * Render current step content
   */
  private renderCurrentStep(): void {
    const body = this.modal?.querySelector('.setup-wizard-body');
    if (!body || !this.status) return;

    const step = this.steps[this.currentStep];
    body.innerHTML = step.render(this.status);

    // Update progress
    this.updateProgressIndicator();

    // Update navigation buttons
    this.updateNavigationButtons();

    // Setup step-specific event listeners
    this.setupStepEventListeners();
  }

  /**
   * Update navigation buttons based on current step
   */
  private updateNavigationButtons(): void {
    const prevBtn = document.getElementById('wizard-prev-btn');
    const nextBtn = document.getElementById('wizard-next-btn');
    const skipBtn = document.getElementById('wizard-skip-btn');

    if (prevBtn) {
      prevBtn.style.display = this.currentStep > 0 ? 'inline-block' : 'none';
    }

    if (nextBtn) {
      const isLastStep = this.currentStep === this.steps.length - 1;
      nextBtn.textContent = isLastStep ? 'Finish' : 'Next';
    }

    if (skipBtn) {
      // Hide skip on last step
      skipBtn.style.display = this.currentStep === this.steps.length - 1 ? 'none' : 'inline-block';
    }
  }

  /**
   * Setup step-specific event listeners
   */
  private setupStepEventListeners(): void {
    const signal = this.abortController?.signal;

    // Sync button
    document.getElementById('wizard-sync-btn')?.addEventListener('click', () => {
      this.callbacks?.onTriggerSync();
      this.refreshStatus();
    }, { signal });

    // Refresh button
    document.getElementById('wizard-refresh-btn')?.addEventListener('click', () => {
      this.refreshStatus();
    }, { signal });

    // Start tour button
    document.getElementById('wizard-start-tour-btn')?.addEventListener('click', () => {
      this.complete();
      this.callbacks?.onStartTour();
    }, { signal });

    // Explore button
    document.getElementById('wizard-explore-btn')?.addEventListener('click', () => {
      this.complete();
    }, { signal });
  }

  /**
   * Refresh status from API
   */
  private async refreshStatus(): Promise<void> {
    try {
      const refreshBtn = document.getElementById('wizard-refresh-btn');
      if (refreshBtn) {
        refreshBtn.textContent = 'Refreshing...';
        refreshBtn.setAttribute('disabled', 'true');
      }

      this.status = await this.fetchStatus();
      this.renderCurrentStep();
    } catch (error) {
      logger.error('Failed to refresh status', { error });
    }
  }

  /**
   * Navigate to next step
   */
  private nextStep(): void {
    if (this.currentStep < this.steps.length - 1) {
      this.currentStep++;
      this.renderCurrentStep();
    } else {
      this.complete();
    }
  }

  /**
   * Navigate to previous step
   */
  private previousStep(): void {
    if (this.currentStep > 0) {
      this.currentStep--;
      this.renderCurrentStep();
    }
  }

  /**
   * Skip the wizard
   */
  private skip(): void {
    SafeStorage.setItem('setup_wizard_skipped', 'true');
    this.hideModal();

    // Safely invoke onSkip callback with error handling
    // This allows the app to proceed with data loading after user skips setup
    try {
      this.callbacks?.onSkip();
    } catch (error) {
      logger.error('Error in onSkip callback', { error });
    }
  }

  /**
   * Complete the wizard
   */
  private complete(): void {
    SafeStorage.setItem('setup_wizard_completed', 'true');
    this.hideModal();

    // Safely invoke callback with error handling
    try {
      this.callbacks?.onComplete();
    } catch (error) {
      logger.error('Error in onComplete callback', { error });
    }
  }

  // ============================================
  // Step Rendering Methods
  // ============================================

  /**
   * Render welcome step
   */
  private renderWelcomeStep(_status: SetupStatus): string {
    return `
      <div class="wizard-step wizard-step-welcome">
        <div class="wizard-step-icon" aria-hidden="true">
          <svg width="64" height="64" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="12" cy="12" r="10"/>
            <path d="M8 14s1.5 2 4 2 4-2 4-2"/>
            <line x1="9" y1="9" x2="9.01" y2="9"/>
            <line x1="15" y1="9" x2="15.01" y2="9"/>
          </svg>
        </div>
        <h3 class="wizard-step-title">Welcome to Cartographus!</h3>
        <p class="wizard-step-description">
          Let's make sure everything is configured correctly for your media analytics dashboard.
        </p>
        <p class="wizard-step-hint">
          This wizard will check your connections and help you get started.
        </p>
      </div>
    `;
  }

  /**
   * Render database step
   */
  private renderDatabaseStep(status: SetupStatus): string {
    const connected = status.database.connected;
    const statusClass = connected ? 'status-success' : 'status-error';
    const statusText = connected ? 'Connected' : 'Not Connected';
    const statusIcon = connected ? '&#10003;' : '&#10007;';

    return `
      <div class="wizard-step wizard-step-database">
        <h3 class="wizard-step-title">Database Connection</h3>
        <p class="wizard-step-description">
          Checking your database connection status.
        </p>

        <div class="wizard-status-card ${statusClass}">
          <div class="status-icon" aria-hidden="true">${statusIcon}</div>
          <div class="status-info">
            <strong>DuckDB Database</strong>
            <span class="status-text">${statusText}</span>
          </div>
        </div>

        ${!connected ? `
          <div class="wizard-alert wizard-alert-warning">
            <strong>Database connection failed</strong>
            <p>Please check your database configuration and ensure the server is running.</p>
          </div>
        ` : `
          <div class="wizard-alert wizard-alert-success">
            <strong>Database is ready</strong>
            <p>Your analytics data will be stored in a high-performance DuckDB database.</p>
          </div>
        `}
      </div>
    `;
  }

  /**
   * Render data sources step
   */
  private renderDataSourcesStep(status: SetupStatus): string {
    const ds = status.data_sources;

    return `
      <div class="wizard-step wizard-step-data-sources">
        <h3 class="wizard-step-title">Data Sources</h3>
        <p class="wizard-step-description">
          Configure at least one data source to start collecting analytics.
        </p>

        <div class="wizard-source-grid">
          ${this.renderSourceCard('Tautulli', ds.tautulli.configured, ds.tautulli.connected, ds.tautulli.error)}
          ${this.renderSourceCard('Plex', ds.plex.configured, ds.plex.connected, ds.plex.error, ds.plex.server_count)}
          ${this.renderSourceCard('Jellyfin', ds.jellyfin.configured, ds.jellyfin.connected, ds.jellyfin.error, ds.jellyfin.server_count)}
          ${this.renderSourceCard('Emby', ds.emby.configured, ds.emby.connected, ds.emby.error, ds.emby.server_count)}
        </div>

        ${ds.nats.enabled ? `
          <div class="wizard-optional-feature">
            <span class="feature-badge">Optional</span>
            <strong>NATS JetStream</strong> is enabled for real-time event processing.
          </div>
        ` : ''}

        ${!this.hasAnyDataSource(status) ? `
          <div class="wizard-alert wizard-alert-warning">
            <strong>No data sources configured</strong>
            <p>Configure Tautulli, Plex, Jellyfin, or Emby in your server configuration to start collecting data.</p>
          </div>
        ` : `
          <div class="wizard-actions">
            <button id="wizard-refresh-btn" class="btn btn-secondary btn-sm">
              Refresh Status
            </button>
          </div>
        `}
      </div>
    `;
  }

  /**
   * Render a source card
   */
  private renderSourceCard(
    name: string,
    configured: boolean,
    connected?: boolean,
    error?: string,
    serverCount?: number
  ): string {
    let statusClass = 'status-inactive';
    let statusText = 'Not Configured';
    let statusIcon = '&#8211;';

    if (configured) {
      if (connected) {
        statusClass = 'status-success';
        statusText = serverCount && serverCount > 1 ? `${serverCount} Servers` : 'Connected';
        statusIcon = '&#10003;';
      } else if (error) {
        statusClass = 'status-error';
        statusText = 'Connection Failed';
        statusIcon = '&#10007;';
      } else {
        statusClass = 'status-pending';
        statusText = serverCount ? `${serverCount} Configured` : 'Configured';
        statusIcon = '&#8226;';
      }
    }

    return `
      <div class="wizard-source-card ${statusClass}">
        <div class="source-icon" aria-hidden="true">${statusIcon}</div>
        <div class="source-info">
          <strong>${name}</strong>
          <span class="source-status">${statusText}</span>
        </div>
      </div>
    `;
  }

  /**
   * Render data step
   */
  private renderDataStep(status: SetupStatus): string {
    const da = status.data_available;
    const hasData = da.has_playbacks;

    return `
      <div class="wizard-step wizard-step-data">
        <h3 class="wizard-step-title">Data Availability</h3>
        <p class="wizard-step-description">
          Checking for existing playback data in your database.
        </p>

        <div class="wizard-data-stats">
          <div class="stat-card">
            <div class="stat-value">${this.formatNumber(da.playback_count)}</div>
            <div class="stat-label">Playback Events</div>
          </div>
          <div class="stat-card">
            <div class="stat-value">${da.has_geolocations ? 'Yes' : 'No'}</div>
            <div class="stat-label">Geolocations</div>
          </div>
        </div>

        ${!hasData ? `
          <div class="wizard-alert wizard-alert-info">
            <strong>No playback data yet</strong>
            <p>Once your data sources are connected and synced, playback events will appear here automatically.</p>
          </div>

          ${this.hasAnyDataSource(status) ? `
            <div class="wizard-actions">
              <button id="wizard-sync-btn" class="btn btn-primary">
                Trigger Sync Now
              </button>
            </div>
          ` : ''}
        ` : `
          <div class="wizard-alert wizard-alert-success">
            <strong>Data is ready!</strong>
            <p>You have ${this.formatNumber(da.playback_count)} playback events ready for visualization.</p>
          </div>
        `}

        ${status.recommendations && status.recommendations.length > 0 ? `
          <div class="wizard-recommendations">
            <h4>Recommendations</h4>
            <ul>
              ${status.recommendations.map(rec => `<li>${this.escapeHtml(rec)}</li>`).join('')}
            </ul>
          </div>
        ` : ''}
      </div>
    `;
  }

  /**
   * Render complete step
   */
  private renderCompleteStep(status: SetupStatus): string {
    const isReady = status.ready;

    return `
      <div class="wizard-step wizard-step-complete">
        <div class="wizard-step-icon ${isReady ? 'icon-success' : 'icon-warning'}" aria-hidden="true">
          ${isReady ? `
            <svg width="64" height="64" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="12" cy="12" r="10"/>
              <path d="M9 12l2 2 4-4"/>
            </svg>
          ` : `
            <svg width="64" height="64" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="12" cy="12" r="10"/>
              <line x1="12" y1="8" x2="12" y2="12"/>
              <line x1="12" y1="16" x2="12.01" y2="16"/>
            </svg>
          `}
        </div>

        <h3 class="wizard-step-title">
          ${isReady ? 'Setup Complete!' : 'Almost There...'}
        </h3>

        <p class="wizard-step-description">
          ${isReady
            ? 'Your Cartographus instance is configured and ready to use.'
            : 'Some configuration is still needed, but you can start exploring the dashboard.'}
        </p>

        <div class="wizard-complete-actions">
          <button id="wizard-start-tour-btn" class="btn btn-primary btn-lg">
            Take a Quick Tour
          </button>
          <button id="wizard-explore-btn" class="btn btn-secondary btn-lg">
            Start Exploring
          </button>
        </div>

        <p class="wizard-step-hint">
          You can always access the tour again by pressing <kbd>?</kbd> for keyboard shortcuts.
        </p>
      </div>
    `;
  }

  // ============================================
  // Utility Methods
  // ============================================

  /**
   * Format number with locale
   */
  private formatNumber(num: number): string {
    return num.toLocaleString();
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
   * Get current status (for external access)
   */
  getStatus(): SetupStatus | null {
    return this.status;
  }

  /**
   * Destroy the wizard and cleanup
   */
  destroy(): void {
    // Mark as destroyed first to prevent any concurrent operations
    this.isDestroyed = true;

    // Abort any pending fetch
    if (this.fetchAbortController) {
      this.fetchAbortController.abort();
      this.fetchAbortController = null;
    }

    // Abort event listeners
    if (this.abortController) {
      this.abortController.abort();
      this.abortController = null;
    }

    // Remove modal
    if (this.modal) {
      this.modal.remove();
      this.modal = null;
    }

    this.isVisible = false;
    this.isShowing = false;
    this.isInitialized = false;
    this.status = null;
    this.callbacks = null;
  }

  /**
   * Check if wizard is currently visible
   */
  isWizardVisible(): boolean {
    return this.isVisible;
  }
}

export default SetupWizardManager;
