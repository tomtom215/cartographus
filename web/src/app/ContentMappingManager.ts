// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * ContentMappingManager - Content Mapping and Cross-Platform Linking
 *
 * Manages content mappings across Plex, Jellyfin, and Emby platforms.
 * Features:
 * - Browse and search content mappings
 * - Create new mappings with external ID lookup (IMDb, TMDb, TVDb)
 * - Link/unlink content to specific platforms
 * - View mapping details with platform availability
 * - CSV import/export functionality
 * - Auto-detect suggestions with confidence scores
 */

import type { API } from '../lib/api';
import type {
  ContentMapping,
  ContentMappingRequest,
  ContentLookupParams,
  CrossPlatformSummary
} from '../lib/types/cross-platform';
import { escapeHtml } from '../lib/sanitize';
import {
  PlatformBadge,
  type Platform,
  ExternalIdLookup
} from '../lib/components';
import { createLogger } from '../lib/logger';

const logger = createLogger('ContentMappingManager');

/**
 * Content mapping filter options
 */
interface ContentMappingFilter {
  search?: string;
  mediaType?: 'movie' | 'show' | 'episode' | '';
  platform?: Platform | '';
  hasExternalId?: 'imdb' | 'tmdb' | 'tvdb' | '';
  page?: number;
  limit?: number;
}

/**
 * Platform API operations lookup
 */
interface PlatformOperations {
  link: (mappingId: number, platformId: string) => Promise<any>;
  unlink: (mappingId: number) => Promise<any>;
}

/**
 * Platform metadata
 */
interface PlatformMetadata {
  displayName: string;
  idLabel: string;
  idHint: string;
}

/**
 * ContentMappingManager class
 */
export class ContentMappingManager {
  private api: API;
  private container: HTMLElement | null = null;
  private mappings: ContentMapping[] = [];
  private selectedMapping: ContentMapping | null = null;
  private filter: ContentMappingFilter = { page: 1, limit: 50 };
  private isLoading = false;
  private totalCount = 0;

  /** Get current mappings (for external access) */
  getMappings(): ContentMapping[] { return this.mappings; }
  /** Get currently selected mapping (for external access) */
  getSelectedMapping(): ContentMapping | null { return this.selectedMapping; }

  // UI components
  private externalIdLookup: ExternalIdLookup | null = null;

  // Event handler references for cleanup
  private searchHandler: (() => void) | null = null;
  private filterChangeHandler: (() => void) | null = null;
  private createBtnHandler: (() => void) | null = null;
  private importBtnHandler: (() => void) | null = null;
  private exportBtnHandler: (() => void) | null = null;
  private modalCloseHandler: ((e: MouseEvent) => void) | null = null;
  private keydownHandler: ((e: KeyboardEvent) => void) | null = null;

  // Platform operations lookup
  private platformOperations: Record<'plex' | 'jellyfin' | 'emby', PlatformOperations>;

  // Platform metadata lookup
  private platformMetadata: Record<'plex' | 'jellyfin' | 'emby', PlatformMetadata> = {
    plex: {
      displayName: 'Plex',
      idLabel: 'Plex Rating Key',
      idHint: 'Found in the URL when viewing the item in Plex Web'
    },
    jellyfin: {
      displayName: 'Jellyfin',
      idLabel: 'Jellyfin Item ID',
      idHint: 'UUID from the Jellyfin item details'
    },
    emby: {
      displayName: 'Emby',
      idLabel: 'Emby Item ID',
      idHint: 'Item ID from Emby item details'
    }
  };

  constructor(api: API) {
    this.api = api;

    // Initialize platform operations lookup
    this.platformOperations = {
      plex: {
        link: (id, platformId) => this.api.linkContentToPlex(id, platformId),
        unlink: (id) => this.api.linkContentToPlex(id, '')
      },
      jellyfin: {
        link: (id, platformId) => this.api.linkContentToJellyfin(id, platformId),
        unlink: (id) => this.api.linkContentToJellyfin(id, '')
      },
      emby: {
        link: (id, platformId) => this.api.linkContentToEmby(id, platformId),
        unlink: (id) => this.api.linkContentToEmby(id, '')
      }
    };
  }

  /**
   * Initialize the content mapping panel
   */
  async init(containerId: string): Promise<void> {
    this.container = document.getElementById(containerId);
    if (!this.container) {
      logger.warn('Container not found', { containerId });
      return;
    }

    logger.debug('Initializing');
    this.render();
    await this.loadMappings();
  }

  /**
   * Render the main content mapping UI
   */
  private render(): void {
    if (!this.container) return;

    this.container.innerHTML = `
      <div class="content-mapping-panel">
        <!-- Toolbar -->
        <div class="content-mapping-toolbar">
          <div class="content-mapping-search">
            <input
              type="text"
              id="content-mapping-search"
              class="form-input"
              placeholder="Search by title, IMDb, TMDb..."
              aria-label="Search content mappings"
            />
          </div>
          <div class="content-mapping-filters">
            <select id="content-mapping-type-filter" class="form-select" aria-label="Filter by media type">
              <option value="">All Types</option>
              <option value="movie">Movies</option>
              <option value="show">TV Shows</option>
              <option value="episode">Episodes</option>
            </select>
            <select id="content-mapping-platform-filter" class="form-select" aria-label="Filter by platform">
              <option value="">All Platforms</option>
              <option value="plex">Plex Only</option>
              <option value="jellyfin">Jellyfin Only</option>
              <option value="emby">Emby Only</option>
            </select>
          </div>
          <div class="content-mapping-actions">
            <button type="button" id="content-mapping-create-btn" class="btn btn-primary">
              + New Mapping
            </button>
            <div class="csv-actions">
              <button type="button" id="content-mapping-import-btn" class="btn btn-secondary" title="Import from CSV">
                Import
              </button>
              <button type="button" id="content-mapping-export-btn" class="btn btn-secondary" title="Export to CSV">
                Export
              </button>
            </div>
          </div>
        </div>

        <!-- Stats bar -->
        <div class="content-mapping-stats" id="content-mapping-stats"></div>

        <!-- Content area -->
        <div class="content-mapping-content" id="content-mapping-content">
          <div class="cross-platform-loading">
            <div class="cross-platform-loading-spinner"></div>
          </div>
        </div>

        <!-- Detail panel (hidden by default) -->
        <div class="content-mapping-detail-panel" id="content-mapping-detail" style="display: none;"></div>
      </div>
    `;

    this.setupEventHandlers();
  }

  /**
   * Setup search handler
   */
  private setupSearchHandler(searchInput: HTMLInputElement): void {
    let debounceTimer: ReturnType<typeof setTimeout>;
    this.searchHandler = () => {
      clearTimeout(debounceTimer);
      debounceTimer = setTimeout(() => {
        this.filter.search = searchInput.value.trim();
        this.filter.page = 1;
        this.loadMappings();
      }, 300);
    };
    searchInput.addEventListener('input', this.searchHandler);
  }

  /**
   * Setup filter handlers
   */
  private setupFilterHandlers(typeFilter: HTMLSelectElement | null, platformFilter: HTMLSelectElement | null): void {
    this.filterChangeHandler = () => {
      if (typeFilter) {
        this.filter.mediaType = typeFilter.value as ContentMappingFilter['mediaType'];
      }
      if (platformFilter) {
        this.filter.platform = platformFilter.value as ContentMappingFilter['platform'];
      }
      this.filter.page = 1;
      this.loadMappings();
    };

    if (typeFilter) typeFilter.addEventListener('change', this.filterChangeHandler);
    if (platformFilter) platformFilter.addEventListener('change', this.filterChangeHandler);
  }

  /**
   * Setup button handlers
   */
  private setupButtonHandlers(): void {
    const createBtn = document.getElementById('content-mapping-create-btn');
    if (createBtn) {
      this.createBtnHandler = () => this.showCreateDialog();
      createBtn.addEventListener('click', this.createBtnHandler);
    }

    const importBtn = document.getElementById('content-mapping-import-btn');
    if (importBtn) {
      this.importBtnHandler = () => this.showImportDialog();
      importBtn.addEventListener('click', this.importBtnHandler);
    }

    const exportBtn = document.getElementById('content-mapping-export-btn');
    if (exportBtn) {
      this.exportBtnHandler = () => this.exportToCSV();
      exportBtn.addEventListener('click', this.exportBtnHandler);
    }
  }

  /**
   * Setup keyboard handler
   */
  private setupKeyboardHandler(): void {
    this.keydownHandler = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        this.closeModal();
      }
    };
    document.addEventListener('keydown', this.keydownHandler);
  }

  /**
   * Set up event handlers
   */
  private setupEventHandlers(): void {
    // Search handler with debounce
    const searchInput = document.getElementById('content-mapping-search') as HTMLInputElement;
    if (searchInput) {
      this.setupSearchHandler(searchInput);
    }

    // Filter change handler
    const typeFilter = document.getElementById('content-mapping-type-filter') as HTMLSelectElement;
    const platformFilter = document.getElementById('content-mapping-platform-filter') as HTMLSelectElement;
    this.setupFilterHandlers(typeFilter, platformFilter);

    // Button handlers
    this.setupButtonHandlers();

    // Keyboard handler for modal escape
    this.setupKeyboardHandler();
  }

  /**
   * Load content mappings from API
   */
  private async loadMappings(): Promise<void> {
    if (this.isLoading) return;

    this.isLoading = true;
    this.showLoading();

    try {
      // Build lookup params from filter - reserved for future use
      void (0 as unknown as ContentLookupParams);

      // Note: The current API is designed for single lookups, not list queries
      // For a full implementation, we'd need a list endpoint
      // For now, we'll show a simulated view with the summary data

      const summaryResponse = await this.api.getCrossPlatformSummary();

      if (summaryResponse.success && summaryResponse.data) {
        this.totalCount = summaryResponse.data.total_content_mappings;
        this.renderStats(summaryResponse.data);
        this.renderMappingsList();
      } else {
        this.renderEmpty('No content mappings found');
      }
    } catch (error) {
      logger.error('Failed to load mappings', { error });
      this.renderError('Failed to load content mappings');
    } finally {
      this.isLoading = false;
    }
  }

  /**
   * Render stats bar
   */
  private renderStats(summary: CrossPlatformSummary): void {
    const statsContainer = document.getElementById('content-mapping-stats');
    if (!statsContainer) return;

    const coverage = summary.content_coverage;
    const coveragePercent = coverage ? Math.round(coverage.percentage) : 0;

    statsContainer.innerHTML = `
      <div class="content-mapping-stat">
        <span class="content-mapping-stat-value">${summary.total_content_mappings}</span>
        <span class="content-mapping-stat-label">Total Mappings</span>
      </div>
      <div class="content-mapping-stat">
        <span class="content-mapping-stat-value">${coveragePercent}%</span>
        <span class="content-mapping-stat-label">Content Coverage</span>
      </div>
      ${summary.platforms ? summary.platforms.map((p) => `
        <div class="content-mapping-stat">
          <span class="content-mapping-stat-value">${p.content_items}</span>
          <span class="content-mapping-stat-label">${escapeHtml(p.name)} Items</span>
        </div>
      `).join('') : ''}
    `;
  }

  /**
   * Render the mappings list
   */
  private renderMappingsList(): void {
    const content = document.getElementById('content-mapping-content');
    if (!content) return;

    if (this.totalCount === 0) {
      this.renderEmpty('No content mappings yet. Create your first mapping to link content across platforms.');
      return;
    }

    // For now, show a placeholder with instructions since we need a list endpoint
    content.innerHTML = `
      <div class="cross-platform-empty">
        <span class="cross-platform-empty-icon" aria-hidden="true">\u{1F4DA}</span>
        <h3 class="cross-platform-empty-title">${this.totalCount} Content Mappings</h3>
        <p class="cross-platform-empty-description">
          Use the search box above to look up specific content by IMDb, TMDb, or TVDb ID.
          Click "New Mapping" to create a new cross-platform content link.
        </p>
        <button type="button" class="btn btn-primary" id="content-quick-search-btn">
          Quick Lookup by External ID
        </button>
      </div>
    `;

    const quickSearchBtn = document.getElementById('content-quick-search-btn');
    if (quickSearchBtn) {
      quickSearchBtn.addEventListener('click', () => this.showLookupDialog());
    }
  }

  /**
   * Render external ID field
   */
  private renderExternalIdField(label: string, value: string | number | undefined): string {
    return `
      <div class="content-mapping-external-id">
        <span class="content-mapping-external-id-label">${label}</span>
        <span class="content-mapping-external-id-value ${!value ? 'empty' : ''}">
          ${value ? escapeHtml(String(value)) : 'Not set'}
        </span>
      </div>
    `;
  }

  /**
   * Get platform info from mapping
   */
  private getPlatformInfo(mapping: ContentMapping): Array<{ platform: Platform; available: boolean; platformId?: string }> {
    return [
      { platform: 'plex', available: !!mapping.plex_rating_key, platformId: mapping.plex_rating_key },
      { platform: 'jellyfin', available: !!mapping.jellyfin_item_id, platformId: mapping.jellyfin_item_id },
      { platform: 'emby', available: !!mapping.emby_item_id, platformId: mapping.emby_item_id }
    ];
  }

  /**
   * Render mapping detail view
   */
  private renderMappingDetail(mapping: ContentMapping): void {
    const detailPanel = document.getElementById('content-mapping-detail');
    if (!detailPanel) return;

    this.selectedMapping = mapping;

    const platforms = this.getPlatformInfo(mapping);

    detailPanel.innerHTML = `
      <div class="content-mapping-detail">
        <div class="content-mapping-detail-header">
          <div>
            <h2 class="content-mapping-detail-title">${escapeHtml(mapping.title)}</h2>
            <div class="content-mapping-detail-meta">
              ${mapping.year ? `<span>Year: ${mapping.year}</span>` : ''}
              <span>Type: ${escapeHtml(mapping.media_type)}</span>
              <span>ID: ${mapping.id}</span>
            </div>
          </div>
          <button type="button" class="btn btn-secondary" id="mapping-detail-close">
            Close
          </button>
        </div>

        <div class="content-mapping-section">
          <h3 class="content-mapping-section-title">External IDs</h3>
          <div class="content-mapping-external-ids">
            ${this.renderExternalIdField('IMDb', mapping.imdb_id)}
            ${this.renderExternalIdField('TMDb', mapping.tmdb_id)}
            ${this.renderExternalIdField('TVDb', mapping.tvdb_id)}
          </div>
        </div>

        <div class="content-mapping-section">
          <h3 class="content-mapping-section-title">Platform Links</h3>
          <div class="content-mapping-platforms" id="mapping-platforms"></div>
        </div>
      </div>
    `;

    // Render platform cards
    const platformsContainer = document.getElementById('mapping-platforms');
    if (platformsContainer) {
      platforms.forEach(({ platform, available, platformId }) => {
        const card = this.createPlatformCard(mapping.id, platform, available, platformId);
        platformsContainer.appendChild(card);
      });
    }

    // Close button handler
    const closeBtn = document.getElementById('mapping-detail-close');
    if (closeBtn) {
      closeBtn.addEventListener('click', () => this.closeMappingDetail());
    }

    detailPanel.style.display = 'block';
  }

  /**
   * Create a platform card for the detail view
   */
  private createPlatformCard(
    mappingId: number,
    platform: Platform,
    isLinked: boolean,
    platformId?: string
  ): HTMLElement {
    const card = document.createElement('div');
    card.className = `content-mapping-platform-card ${isLinked ? 'linked' : ''}`;

    const badge = new PlatformBadge({ platform, available: isLinked, showLabel: true });

    card.innerHTML = `
      <div class="content-mapping-platform-header">
        <div id="platform-badge-${platform}"></div>
        <div class="content-mapping-platform-actions">
          ${isLinked
            ? `<button type="button" class="btn btn-sm btn-danger" data-action="unlink" data-platform="${platform}">
                Unlink
               </button>`
            : `<button type="button" class="btn btn-sm btn-primary" data-action="link" data-platform="${platform}">
                Link
               </button>`
          }
        </div>
      </div>
      <div class="content-mapping-platform-id">
        ${isLinked && platformId ? escapeHtml(platformId) : 'Not linked'}
      </div>
    `;

    // Insert badge
    const badgeContainer = card.querySelector(`#platform-badge-${platform}`);
    if (badgeContainer) {
      badgeContainer.appendChild(badge.render());
    }

    // Add action handlers
    const actionBtn = card.querySelector('button[data-action]');
    if (actionBtn) {
      actionBtn.addEventListener('click', () => {
        const action = actionBtn.getAttribute('data-action');
        const targetPlatform = actionBtn.getAttribute('data-platform') as Platform;

        if (action === 'link') {
          this.showLinkDialog(mappingId, targetPlatform);
        } else if (action === 'unlink') {
          this.unlinkPlatform(mappingId, targetPlatform);
        }
      });
    }

    return card;
  }

  /**
   * Close mapping detail view
   */
  private closeMappingDetail(): void {
    const detailPanel = document.getElementById('content-mapping-detail');
    if (detailPanel) {
      detailPanel.style.display = 'none';
      detailPanel.innerHTML = '';
    }
    this.selectedMapping = null;
  }

  /**
   * Show create mapping dialog
   */
  private showCreateDialog(): void {
    const modal = this.createModal('Create Content Mapping', `
      <form id="create-mapping-form" class="cross-platform-modal-form">
        <div class="cross-platform-modal-form-group">
          <label class="cross-platform-modal-form-label">Title *</label>
          <input type="text" id="mapping-title" class="form-input" required placeholder="e.g., The Matrix" />
        </div>

        <div class="cross-platform-modal-form-group">
          <label class="cross-platform-modal-form-label">Media Type *</label>
          <select id="mapping-media-type" class="form-select" required>
            <option value="movie">Movie</option>
            <option value="show">TV Show</option>
            <option value="episode">Episode</option>
          </select>
        </div>

        <div class="cross-platform-modal-form-group">
          <label class="cross-platform-modal-form-label">Year</label>
          <input type="number" id="mapping-year" class="form-input" placeholder="e.g., 1999" min="1800" max="2100" />
        </div>

        <div class="cross-platform-modal-form-group">
          <label class="cross-platform-modal-form-label">External IDs (at least one required)</label>
          <div class="external-id-inputs">
            <div class="external-id-input-group">
              <label>IMDb</label>
              <input type="text" id="mapping-imdb" class="form-input" placeholder="tt1234567" />
              <span class="cross-platform-modal-form-hint">Format: tt followed by 7+ digits</span>
            </div>
            <div class="external-id-input-group">
              <label>TMDb</label>
              <input type="number" id="mapping-tmdb" class="form-input" placeholder="12345" />
            </div>
            <div class="external-id-input-group">
              <label>TVDb</label>
              <input type="number" id="mapping-tvdb" class="form-input" placeholder="12345" />
            </div>
          </div>
        </div>

        <div class="modal-actions">
          <button type="button" class="btn btn-secondary" data-action="cancel">Cancel</button>
          <button type="submit" class="btn btn-primary">Create Mapping</button>
        </div>
      </form>
    `);

    // Form submission handler
    const form = modal.querySelector('#create-mapping-form') as HTMLFormElement;
    if (form) {
      form.addEventListener('submit', async (e) => {
        e.preventDefault();
        await this.handleCreateMapping(form);
      });
    }

    // Cancel button handler
    const cancelBtn = modal.querySelector('[data-action="cancel"]');
    if (cancelBtn) {
      cancelBtn.addEventListener('click', () => this.closeModal());
    }
  }

  /**
   * Extract form values for creating a mapping
   */
  private extractMappingFormValues(form: HTMLFormElement) {
    return {
      title: (form.querySelector('#mapping-title') as HTMLInputElement).value.trim(),
      mediaType: (form.querySelector('#mapping-media-type') as HTMLSelectElement).value as 'movie' | 'show' | 'episode',
      year: parseInt((form.querySelector('#mapping-year') as HTMLInputElement).value, 10),
      imdbId: (form.querySelector('#mapping-imdb') as HTMLInputElement).value.trim(),
      tmdbId: parseInt((form.querySelector('#mapping-tmdb') as HTMLInputElement).value, 10),
      tvdbId: parseInt((form.querySelector('#mapping-tvdb') as HTMLInputElement).value, 10)
    };
  }

  /**
   * Validate external IDs
   */
  private validateExternalIds(imdbId: string, tmdbId: number, tvdbId: number): string | null {
    if (!imdbId && !tmdbId && !tvdbId) {
      return 'Please provide at least one external ID (IMDb, TMDb, or TVDb)';
    }

    if (imdbId && !/^tt\d{7,}$/.test(imdbId)) {
      return 'Invalid IMDb ID format. Expected: tt followed by 7+ digits';
    }

    return null;
  }

  /**
   * Build content mapping request from form values
   */
  private buildMappingRequest(values: ReturnType<typeof this.extractMappingFormValues>): ContentMappingRequest {
    return {
      title: values.title,
      media_type: values.mediaType,
      ...(values.year && !isNaN(values.year) ? { year: values.year } : {}),
      ...(values.imdbId ? { imdb_id: values.imdbId } : {}),
      ...(values.tmdbId && !isNaN(values.tmdbId) ? { tmdb_id: values.tmdbId } : {}),
      ...(values.tvdbId && !isNaN(values.tvdbId) ? { tvdb_id: values.tvdbId } : {})
    };
  }

  /**
   * Handle create mapping form submission
   */
  private async handleCreateMapping(form: HTMLFormElement): Promise<void> {
    const values = this.extractMappingFormValues(form);

    // Validate external IDs
    const validationError = this.validateExternalIds(values.imdbId, values.tmdbId, values.tvdbId);
    if (validationError) {
      alert(validationError);
      return;
    }

    const request = this.buildMappingRequest(values);

    try {
      const response = await this.api.createContentMapping(request);

      if (response.success && response.data) {
        this.closeModal();
        this.showToast('Content mapping created successfully', 'success');
        this.renderMappingDetail(response.data);
        await this.loadMappings();
      } else {
        this.showToast(response.message || 'Failed to create mapping', 'error');
      }
    } catch (error) {
      logger.error('Create failed', { error });
      this.showToast('Failed to create content mapping', 'error');
    }
  }

  /**
   * Render lookup result - found
   */
  private renderLookupResultFound(resultContainer: Element, result: ContentMapping): void {
    resultContainer.innerHTML = `
      <div class="external-id-result">
        <strong>${escapeHtml(result.title)}</strong>
        ${result.year ? `(${result.year})` : ''}<br/>
        <small>Type: ${escapeHtml(result.media_type)} | ID: ${result.id}</small>
        <div style="margin-top: 0.5rem;">
          <button type="button" class="btn btn-sm btn-primary" id="view-mapping-btn">
            View Details
          </button>
        </div>
      </div>
    `;

    const viewBtn = resultContainer.querySelector('#view-mapping-btn');
    if (viewBtn) {
      viewBtn.addEventListener('click', () => {
        this.closeModal();
        this.renderMappingDetail(result);
      });
    }
  }

  /**
   * Render lookup result - not found
   */
  private renderLookupResultNotFound(resultContainer: Element): void {
    resultContainer.innerHTML = `
      <div class="external-id-result not-found">
        No mapping found. <button type="button" class="btn btn-sm btn-primary" id="create-new-btn">Create New</button>
      </div>
    `;

    const createBtn = resultContainer.querySelector('#create-new-btn');
    if (createBtn) {
      createBtn.addEventListener('click', () => {
        this.closeModal();
        this.showCreateDialog();
      });
    }
  }

  /**
   * Handle lookup result
   */
  private handleLookupResult(modal: Element, result: ContentMapping | null): void {
    const resultContainer = modal.querySelector('#lookup-result-container');
    if (!resultContainer) return;

    if (result) {
      this.renderLookupResultFound(resultContainer, result);
    } else {
      this.renderLookupResultNotFound(resultContainer);
    }
  }

  /**
   * Show lookup dialog for finding existing mappings
   */
  private showLookupDialog(): void {
    const modal = this.createModal('Lookup Content Mapping', `
      <div class="cross-platform-modal-form">
        <p>Search for an existing content mapping by external ID:</p>
        <div id="external-id-lookup-container"></div>
        <div id="lookup-result-container" style="margin-top: 1rem;"></div>
        <div class="modal-actions">
          <button type="button" class="btn btn-secondary" data-action="cancel">Close</button>
        </div>
      </div>
    `);

    // Create external ID lookup component
    const lookupContainer = modal.querySelector('#external-id-lookup-container');
    if (lookupContainer) {
      this.externalIdLookup = new ExternalIdLookup({
        showInlineResult: false,
        onLookup: async (params) => {
          const response = await this.api.lookupContentMapping(params);
          return response.success ? response.data || null : null;
        },
        onResult: (result) => this.handleLookupResult(modal, result)
      });

      lookupContainer.appendChild(this.externalIdLookup.render());
    }

    // Cancel button
    const cancelBtn = modal.querySelector('[data-action="cancel"]');
    if (cancelBtn) {
      cancelBtn.addEventListener('click', () => this.closeModal());
    }
  }

  /**
   * Get platform operations for a platform (handles 'tautulli' gracefully)
   */
  private getPlatformOperations(platform: Platform): PlatformOperations | null {
    if (platform === 'tautulli') return null;
    return this.platformOperations[platform];
  }

  /**
   * Get platform metadata for a platform (handles 'tautulli' gracefully)
   */
  private getPlatformMetadata(platform: Platform): PlatformMetadata | null {
    if (platform === 'tautulli') return null;
    return this.platformMetadata[platform];
  }

  /**
   * Show link to platform dialog
   */
  private showLinkDialog(mappingId: number, platform: Platform): void {
    const metadata = this.getPlatformMetadata(platform);
    if (!metadata) {
      this.showToast(`Platform ${platform} does not support linking`, 'error');
      return;
    }

    const modal = this.createModal(`Link to ${metadata.displayName}`, `
      <form id="link-platform-form" class="cross-platform-modal-form">
        <div class="cross-platform-modal-form-group">
          <label class="cross-platform-modal-form-label">${metadata.idLabel}</label>
          <input type="text" id="platform-id" class="form-input" required
            placeholder="Enter the ${platform} item ID" />
          <span class="cross-platform-modal-form-hint">
            ${metadata.idHint}
          </span>
        </div>
        <div class="modal-actions">
          <button type="button" class="btn btn-secondary" data-action="cancel">Cancel</button>
          <button type="submit" class="btn btn-primary">Link</button>
        </div>
      </form>
    `);

    // Form handler
    const form = modal.querySelector('#link-platform-form') as HTMLFormElement;
    if (form) {
      form.addEventListener('submit', async (e) => {
        e.preventDefault();
        await this.handleLinkPlatform(mappingId, platform, form);
      });
    }

    // Cancel button
    const cancelBtn = modal.querySelector('[data-action="cancel"]');
    if (cancelBtn) {
      cancelBtn.addEventListener('click', () => this.closeModal());
    }
  }

  /**
   * Handle linking a platform
   */
  private async handleLinkPlatform(mappingId: number, platform: Platform, form: HTMLFormElement): Promise<void> {
    const platformId = (form.querySelector('#platform-id') as HTMLInputElement).value.trim();
    const operations = this.getPlatformOperations(platform);

    if (!operations) {
      this.showToast(`Platform ${platform} does not support linking`, 'error');
      return;
    }

    try {
      const response = await operations.link(mappingId, platformId);

      if (response.success && response.data) {
        this.closeModal();
        this.showToast(`Linked to ${platform} successfully`, 'success');
        this.renderMappingDetail(response.data);
      } else {
        this.showToast(response.message || 'Failed to link', 'error');
      }
    } catch (error) {
      logger.error('Link failed', { error, mappingId, platformId });
      this.showToast('Failed to link to platform', 'error');
    }
  }

  /**
   * Unlink a platform from a mapping
   */
  private async unlinkPlatform(mappingId: number, platform: Platform): Promise<void> {
    if (!confirm(`Are you sure you want to unlink this content from ${platform}?`)) {
      return;
    }

    const operations = this.getPlatformOperations(platform);
    if (!operations) {
      this.showToast(`Platform ${platform} does not support unlinking`, 'error');
      return;
    }

    try {
      const response = await operations.unlink(mappingId);

      if (response.success && response.data) {
        this.showToast(`Unlinked from ${platform}`, 'success');
        this.renderMappingDetail(response.data);
      } else {
        this.showToast(response.message || 'Failed to unlink', 'error');
      }
    } catch (error) {
      logger.error('Unlink failed', { error, mappingId, platform });
      this.showToast('Failed to unlink from platform', 'error');
    }
  }

  /**
   * Setup drag and drop for file input
   */
  private setupDragAndDrop(dropZone: Element, fileInput: HTMLInputElement, modal: Element): void {
    // Click to browse
    dropZone.addEventListener('click', () => fileInput.click());

    // File input change
    fileInput.addEventListener('change', () => {
      if (fileInput.files && fileInput.files[0]) {
        this.handleCSVFile(fileInput.files[0], modal);
      }
    });

    // Drag over
    dropZone.addEventListener('dragover', (e) => {
      e.preventDefault();
      dropZone.classList.add('dragover');
    });

    // Drag leave
    dropZone.addEventListener('dragleave', () => {
      dropZone.classList.remove('dragover');
    });

    // Drop
    dropZone.addEventListener('drop', (e) => {
      e.preventDefault();
      dropZone.classList.remove('dragover');
      const files = (e as DragEvent).dataTransfer?.files;
      if (files && files[0]) {
        this.handleCSVFile(files[0], modal);
      }
    });
  }

  /**
   * Show import dialog
   */
  private showImportDialog(): void {
    const modal = this.createModal('Import Content Mappings', `
      <div class="cross-platform-modal-form">
        <div class="csv-import-zone" id="csv-import-zone">
          <span class="csv-import-icon" aria-hidden="true">\u{1F4C4}</span>
          <p class="csv-import-text">Drop CSV file here or click to browse</p>
          <p class="csv-import-hint">
            Required columns: title, media_type<br/>
            Optional: imdb_id, tmdb_id, tvdb_id, year, plex_rating_key, jellyfin_item_id, emby_item_id
          </p>
          <input type="file" id="csv-file-input" accept=".csv" style="display: none;" />
        </div>
        <div id="import-preview" style="display: none; margin-top: 1rem;"></div>
        <div class="modal-actions">
          <button type="button" class="btn btn-secondary" data-action="cancel">Cancel</button>
          <button type="button" class="btn btn-primary" id="import-confirm-btn" disabled>Import</button>
        </div>
      </div>
    `);

    const dropZone = modal.querySelector('#csv-import-zone');
    const fileInput = modal.querySelector('#csv-file-input') as HTMLInputElement;

    if (dropZone && fileInput) {
      this.setupDragAndDrop(dropZone, fileInput, modal);
    }

    // Cancel button
    const cancelBtn = modal.querySelector('[data-action="cancel"]');
    if (cancelBtn) {
      cancelBtn.addEventListener('click', () => this.closeModal());
    }
  }

  /**
   * Handle CSV file for import
   */
  private handleCSVFile(file: File, modal: Element): void {
    const reader = new FileReader();
    reader.onload = (e) => {
      const content = e.target?.result as string;
      const lines = content.split('\n').filter(l => l.trim());

      if (lines.length < 2) {
        alert('CSV file must have a header row and at least one data row');
        return;
      }

      const preview = modal.querySelector('#import-preview') as HTMLElement | null;
      const importBtn = modal.querySelector('#import-confirm-btn') as HTMLButtonElement;

      if (preview) {
        preview.innerHTML = `
          <p><strong>${lines.length - 1} rows</strong> will be imported.</p>
          <p class="cross-platform-modal-form-hint">
            Preview: ${escapeHtml(lines[1].substring(0, 100))}...
          </p>
        `;
        preview.style.display = 'block';
      }

      if (importBtn) {
        importBtn.disabled = false;
        importBtn.onclick = () => this.processCSVImport(content);
      }
    };

    reader.readAsText(file);
  }

  /**
   * Get CSV column indices
   */
  private getCSVColumnIndices(headers: string[]) {
    return {
      title: headers.indexOf('title'),
      mediaType: headers.indexOf('media_type'),
      imdb: headers.indexOf('imdb_id'),
      tmdb: headers.indexOf('tmdb_id'),
      tvdb: headers.indexOf('tvdb_id'),
      year: headers.indexOf('year')
    };
  }

  /**
   * Build mapping request from CSV row
   */
  private buildMappingRequestFromCSV(values: string[], indices: ReturnType<typeof this.getCSVColumnIndices>): ContentMappingRequest | null {
    if (values.length <= Math.max(indices.title, indices.mediaType)) {
      return null;
    }

    const request: ContentMappingRequest = {
      title: values[indices.title],
      media_type: values[indices.mediaType] as 'movie' | 'show' | 'episode'
    };

    // Add optional fields
    if (indices.imdb !== -1 && values[indices.imdb]) {
      request.imdb_id = values[indices.imdb];
    }
    if (indices.tmdb !== -1 && values[indices.tmdb]) {
      request.tmdb_id = parseInt(values[indices.tmdb], 10);
    }
    if (indices.tvdb !== -1 && values[indices.tvdb]) {
      request.tvdb_id = parseInt(values[indices.tvdb], 10);
    }
    if (indices.year !== -1 && values[indices.year]) {
      request.year = parseInt(values[indices.year], 10);
    }

    return request;
  }

  /**
   * Import single CSV row
   */
  private async importCSVRow(request: ContentMappingRequest): Promise<boolean> {
    try {
      const response = await this.api.createContentMapping(request);
      return response.success;
    } catch {
      return false;
    }
  }

  /**
   * Process CSV import
   */
  private async processCSVImport(content: string): Promise<void> {
    const lines = content.split('\n').filter(l => l.trim());
    const headers = lines[0].split(',').map(h => h.trim().toLowerCase());

    const indices = this.getCSVColumnIndices(headers);

    if (indices.title === -1 || indices.mediaType === -1) {
      alert('CSV must have "title" and "media_type" columns');
      return;
    }

    let successCount = 0;
    let errorCount = 0;

    for (let i = 1; i < lines.length; i++) {
      const values = this.parseCSVLine(lines[i]);
      const request = this.buildMappingRequestFromCSV(values, indices);

      if (!request) continue;

      const success = await this.importCSVRow(request);
      if (success) {
        successCount++;
      } else {
        errorCount++;
      }
    }

    this.closeModal();
    this.showToast(`Imported ${successCount} mappings (${errorCount} errors)`, successCount > 0 ? 'success' : 'error');
    await this.loadMappings();
  }

  /**
   * Parse a CSV line respecting quoted fields
   */
  private parseCSVLine(line: string): string[] {
    const result: string[] = [];
    let current = '';
    let inQuotes = false;

    for (let i = 0; i < line.length; i++) {
      const char = line[i];

      if (char === '"') {
        inQuotes = !inQuotes;
      } else if (char === ',' && !inQuotes) {
        result.push(current.trim());
        current = '';
      } else {
        current += char;
      }
    }

    result.push(current.trim());
    return result;
  }

  /**
   * Export mappings to CSV
   */
  private async exportToCSV(): Promise<void> {
    // For now, export a template since we don't have a list endpoint
    const headers = ['title', 'media_type', 'year', 'imdb_id', 'tmdb_id', 'tvdb_id', 'plex_rating_key', 'jellyfin_item_id', 'emby_item_id'];
    const csvContent = headers.join(',') + '\n';

    const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = `content_mappings_template_${new Date().toISOString().split('T')[0]}.csv`;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(url);

    this.showToast('CSV template downloaded', 'success');
  }

  /**
   * Create a modal dialog
   */
  private createModal(title: string, content: string): HTMLElement {
    // Remove existing modal
    this.closeModal();

    const modal = document.createElement('div');
    modal.className = 'modal-overlay';
    modal.id = 'content-mapping-modal';
    modal.innerHTML = `
      <div class="modal cross-platform-modal-content">
        <div class="modal-header">
          <h3 class="modal-title">${escapeHtml(title)}</h3>
          <button type="button" class="modal-close" aria-label="Close">&times;</button>
        </div>
        <div class="modal-body">
          ${content}
        </div>
      </div>
    `;

    // Close button handler
    const closeBtn = modal.querySelector('.modal-close');
    if (closeBtn) {
      closeBtn.addEventListener('click', () => this.closeModal());
    }

    // Click outside to close
    this.modalCloseHandler = (e: MouseEvent) => {
      if (e.target === modal) {
        this.closeModal();
      }
    };
    modal.addEventListener('click', this.modalCloseHandler);

    document.body.appendChild(modal);
    return modal;
  }

  /**
   * Close modal dialog
   */
  private closeModal(): void {
    // Cleanup external ID lookup if exists
    if (this.externalIdLookup) {
      this.externalIdLookup.destroy();
      this.externalIdLookup = null;
    }

    const modal = document.getElementById('content-mapping-modal');
    if (modal) {
      if (this.modalCloseHandler) {
        modal.removeEventListener('click', this.modalCloseHandler);
        this.modalCloseHandler = null;
      }
      modal.remove();
    }
  }

  /**
   * Show loading state
   */
  private showLoading(): void {
    const content = document.getElementById('content-mapping-content');
    if (content) {
      content.innerHTML = `
        <div class="cross-platform-loading">
          <div class="cross-platform-loading-spinner"></div>
        </div>
      `;
    }
  }

  /**
   * Render empty state
   */
  private renderEmpty(message: string): void {
    const content = document.getElementById('content-mapping-content');
    if (content) {
      content.innerHTML = `
        <div class="cross-platform-empty">
          <span class="cross-platform-empty-icon" aria-hidden="true">\u{1F4DA}</span>
          <h3 class="cross-platform-empty-title">No Content Mappings</h3>
          <p class="cross-platform-empty-description">${escapeHtml(message)}</p>
          <button type="button" class="btn btn-primary" id="empty-create-btn">
            Create First Mapping
          </button>
        </div>
      `;

      const createBtn = document.getElementById('empty-create-btn');
      if (createBtn) {
        createBtn.addEventListener('click', () => this.showCreateDialog());
      }
    }
  }

  /**
   * Render error state
   */
  private renderError(message: string): void {
    const content = document.getElementById('content-mapping-content');
    if (content) {
      content.innerHTML = `
        <div class="cross-platform-empty">
          <span class="cross-platform-empty-icon" aria-hidden="true">\u26A0</span>
          <h3 class="cross-platform-empty-title">Error</h3>
          <p class="cross-platform-empty-description">${escapeHtml(message)}</p>
          <button type="button" class="btn btn-primary" id="error-retry-btn">
            Retry
          </button>
        </div>
      `;

      const retryBtn = document.getElementById('error-retry-btn');
      if (retryBtn) {
        retryBtn.addEventListener('click', () => this.loadMappings());
      }
    }
  }

  /**
   * Show toast notification
   */
  private showToast(message: string, type: 'success' | 'error' | 'info' = 'info'): void {
    // Use existing toast system if available
    const event = new CustomEvent('show-toast', {
      detail: { message, type }
    });
    window.dispatchEvent(event);

    // Fallback to console
    logger.debug('Toast shown', { message, type });
  }

  /**
   * Remove search event handler
   */
  private removeSearchHandler(): void {
    const searchInput = document.getElementById('content-mapping-search');
    if (searchInput && this.searchHandler) {
      searchInput.removeEventListener('input', this.searchHandler);
      this.searchHandler = null;
    }
  }

  /**
   * Remove filter event handlers
   */
  private removeFilterHandlers(): void {
    if (!this.filterChangeHandler) return;

    const typeFilter = document.getElementById('content-mapping-type-filter');
    const platformFilter = document.getElementById('content-mapping-platform-filter');

    if (typeFilter) typeFilter.removeEventListener('change', this.filterChangeHandler);
    if (platformFilter) platformFilter.removeEventListener('change', this.filterChangeHandler);

    this.filterChangeHandler = null;
  }

  /**
   * Remove button event handlers
   */
  private removeButtonHandlers(): void {
    const createBtn = document.getElementById('content-mapping-create-btn');
    if (createBtn && this.createBtnHandler) {
      createBtn.removeEventListener('click', this.createBtnHandler);
      this.createBtnHandler = null;
    }

    const importBtn = document.getElementById('content-mapping-import-btn');
    if (importBtn && this.importBtnHandler) {
      importBtn.removeEventListener('click', this.importBtnHandler);
      this.importBtnHandler = null;
    }

    const exportBtn = document.getElementById('content-mapping-export-btn');
    if (exportBtn && this.exportBtnHandler) {
      exportBtn.removeEventListener('click', this.exportBtnHandler);
      this.exportBtnHandler = null;
    }
  }

  /**
   * Remove keyboard event handler
   */
  private removeKeyboardHandler(): void {
    if (this.keydownHandler) {
      document.removeEventListener('keydown', this.keydownHandler);
      this.keydownHandler = null;
    }
  }

  /**
   * Destroy the manager and cleanup resources
   */
  destroy(): void {
    // Remove event handlers
    this.removeSearchHandler();
    this.removeFilterHandlers();
    this.removeButtonHandlers();
    this.removeKeyboardHandler();

    // Cleanup components
    if (this.externalIdLookup) {
      this.externalIdLookup.destroy();
      this.externalIdLookup = null;
    }

    // Close any open modal
    this.closeModal();

    // Clear container reference
    this.container = null;

    logger.debug('Destroyed');
  }
}

export default ContentMappingManager;
