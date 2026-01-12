// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * ExternalIdLookup - Search component for IMDb/TMDb/TVDb IDs
 *
 * Provides a search interface for looking up content by external database IDs
 * with validation and lookup functionality.
 */

import { escapeHtml } from '../sanitize';
import type { ContentMapping, ContentLookupParams } from '../types/cross-platform';
import { createLogger } from '../logger';

const logger = createLogger('ExternalIdLookup');

/**
 * External ID types supported
 */
export type ExternalIdType = 'imdb' | 'tmdb' | 'tvdb';

/**
 * External ID configuration
 */
interface ExternalIdConfig {
  name: string;
  placeholder: string;
  pattern: RegExp;
  hint: string;
  formatForApi: (value: string) => ContentLookupParams;
}

/**
 * External ID configurations
 */
const EXTERNAL_ID_CONFIGS: Record<ExternalIdType, ExternalIdConfig> = {
  imdb: {
    name: 'IMDb',
    placeholder: 'tt1234567',
    pattern: /^tt\d{7,}$/,
    hint: 'Format: tt followed by 7+ digits (e.g., tt1234567)',
    formatForApi: (value: string) => ({ imdb_id: value })
  },
  tmdb: {
    name: 'TMDb',
    placeholder: '12345',
    pattern: /^\d+$/,
    hint: 'Format: Numeric ID (e.g., 12345)',
    formatForApi: (value: string) => ({ tmdb_id: parseInt(value, 10) })
  },
  tvdb: {
    name: 'TVDb',
    placeholder: '12345',
    pattern: /^\d+$/,
    hint: 'Format: Numeric ID (e.g., 12345)',
    formatForApi: (value: string) => ({ tvdb_id: parseInt(value, 10) })
  }
};

/**
 * Lookup result callback
 */
export type LookupCallback = (result: ContentMapping | null, created: boolean) => void;

/**
 * Options for the ExternalIdLookup component
 */
export interface ExternalIdLookupOptions {
  /** Initial ID type */
  initialType?: ExternalIdType;
  /** Initial ID value */
  initialValue?: string;
  /** Callback when lookup is performed */
  onLookup?: (params: ContentLookupParams) => Promise<ContentMapping | null>;
  /** Callback when result is received */
  onResult?: LookupCallback;
  /** Show inline result */
  showInlineResult?: boolean;
  /** Placeholder for custom hint text */
  customHint?: string;
}

/**
 * ExternalIdLookup class for searching content by external IDs
 */
export class ExternalIdLookup {
  private options: ExternalIdLookupOptions;
  private element: HTMLElement | null = null;
  private typeSelect: HTMLSelectElement | null = null;
  private idInput: HTMLInputElement | null = null;
  private searchButton: HTMLButtonElement | null = null;
  private resultContainer: HTMLElement | null = null;
  private hintElement: HTMLElement | null = null;

  // Event handlers for cleanup
  private typeChangeHandler: (() => void) | null = null;
  private inputHandler: (() => void) | null = null;
  private searchHandler: (() => void) | null = null;
  private keydownHandler: ((e: KeyboardEvent) => void) | null = null;

  private currentType: ExternalIdType;
  private isLoading: boolean = false;

  constructor(options: ExternalIdLookupOptions = {}) {
    this.options = {
      initialType: 'imdb',
      showInlineResult: true,
      ...options
    };
    this.currentType = this.options.initialType || 'imdb';
  }

  /**
   * Render the lookup component
   */
  render(): HTMLElement {
    this.element = document.createElement('div');
    this.element.className = 'external-id-lookup';

    const config = EXTERNAL_ID_CONFIGS[this.currentType];

    this.element.innerHTML = `
      <div class="external-id-row">
        <select class="external-id-type-select form-select" aria-label="External ID type">
          ${Object.entries(EXTERNAL_ID_CONFIGS).map(([type, cfg]) => `
            <option value="${type}" ${type === this.currentType ? 'selected' : ''}>
              ${escapeHtml(cfg.name)}
            </option>
          `).join('')}
        </select>
        <div class="external-id-input-wrapper">
          <input
            type="text"
            class="external-id-input form-input"
            placeholder="${escapeHtml(config.placeholder)}"
            value="${escapeHtml(this.options.initialValue || '')}"
            aria-label="External ID value"
          />
        </div>
        <button type="button" class="btn btn-primary external-id-search-btn" aria-label="Search">
          Search
        </button>
      </div>
      <div class="external-id-hint">${escapeHtml(this.options.customHint || config.hint)}</div>
      ${this.options.showInlineResult ? '<div class="external-id-result-container"></div>' : ''}
    `;

    // Get element references
    this.typeSelect = this.element.querySelector('.external-id-type-select');
    this.idInput = this.element.querySelector('.external-id-input');
    this.searchButton = this.element.querySelector('.external-id-search-btn');
    this.hintElement = this.element.querySelector('.external-id-hint');
    this.resultContainer = this.element.querySelector('.external-id-result-container');

    // Set up event handlers
    this.setupEventHandlers();

    return this.element;
  }

  /**
   * Set up event handlers
   */
  private setupEventHandlers(): void {
    // Type change handler
    if (this.typeSelect) {
      this.typeChangeHandler = () => {
        this.currentType = this.typeSelect!.value as ExternalIdType;
        this.updateForType();
      };
      this.typeSelect.addEventListener('change', this.typeChangeHandler);
    }

    // Input validation handler
    if (this.idInput) {
      this.inputHandler = () => {
        this.validateInput();
      };
      this.idInput.addEventListener('input', this.inputHandler);

      // Enter key handler
      this.keydownHandler = (e: KeyboardEvent) => {
        if (e.key === 'Enter' && !this.isLoading) {
          this.performSearch();
        }
      };
      this.idInput.addEventListener('keydown', this.keydownHandler);
    }

    // Search button handler
    if (this.searchButton) {
      this.searchHandler = () => {
        this.performSearch();
      };
      this.searchButton.addEventListener('click', this.searchHandler);
    }
  }

  /**
   * Update UI for current type
   */
  private updateForType(): void {
    const config = EXTERNAL_ID_CONFIGS[this.currentType];

    if (this.idInput) {
      this.idInput.placeholder = config.placeholder;
      this.idInput.value = '';
      this.idInput.classList.remove('valid', 'invalid');
    }

    if (this.hintElement) {
      this.hintElement.textContent = this.options.customHint || config.hint;
    }

    if (this.resultContainer) {
      this.resultContainer.innerHTML = '';
    }
  }

  /**
   * Validate the current input
   */
  private validateInput(): boolean {
    if (!this.idInput) return false;

    const value = this.idInput.value.trim();
    const config = EXTERNAL_ID_CONFIGS[this.currentType];

    if (!value) {
      this.idInput.classList.remove('valid', 'invalid');
      return false;
    }

    const isValid = config.pattern.test(value);
    this.idInput.classList.toggle('valid', isValid);
    this.idInput.classList.toggle('invalid', !isValid);

    return isValid;
  }

  /**
   * Perform the search
   */
  private async performSearch(): Promise<void> {
    if (!this.idInput || !this.options.onLookup) return;

    const value = this.idInput.value.trim();
    if (!this.validateInput()) {
      this.showError('Please enter a valid ID in the correct format.');
      return;
    }

    const config = EXTERNAL_ID_CONFIGS[this.currentType];
    const params = config.formatForApi(value);

    this.setLoading(true);

    try {
      const result = await this.options.onLookup(params);

      if (result) {
        this.showResult(result);
        if (this.options.onResult) {
          this.options.onResult(result, false);
        }
      } else {
        this.showNotFound(value);
        if (this.options.onResult) {
          this.options.onResult(null, false);
        }
      }
    } catch (error) {
      logger.error('Search failed:', error);
      this.showError('Search failed. Please try again.');
    } finally {
      this.setLoading(false);
    }
  }

  /**
   * Set loading state
   */
  private setLoading(loading: boolean): void {
    this.isLoading = loading;

    if (this.searchButton) {
      this.searchButton.disabled = loading;
      this.searchButton.textContent = loading ? 'Searching...' : 'Search';
    }

    if (this.idInput) {
      this.idInput.disabled = loading;
    }

    if (this.typeSelect) {
      this.typeSelect.disabled = loading;
    }
  }

  /**
   * Show search result
   */
  private showResult(mapping: ContentMapping): void {
    if (!this.resultContainer) return;

    const platformLinks = [];
    if (mapping.plex_rating_key) platformLinks.push('Plex');
    if (mapping.jellyfin_item_id) platformLinks.push('Jellyfin');
    if (mapping.emby_item_id) platformLinks.push('Emby');

    this.resultContainer.innerHTML = `
      <div class="external-id-result">
        <div class="external-id-result-header">
          <strong>${escapeHtml(mapping.title)}</strong>
          ${mapping.year ? `<span class="external-id-result-year">(${mapping.year})</span>` : ''}
        </div>
        <div class="external-id-result-meta">
          <span class="external-id-result-type">${escapeHtml(mapping.media_type)}</span>
          <span class="external-id-result-id">ID: ${mapping.id}</span>
        </div>
        ${platformLinks.length > 0 ? `
          <div class="external-id-result-platforms">
            Linked to: ${platformLinks.join(', ')}
          </div>
        ` : `
          <div class="external-id-result-no-platforms">
            Not yet linked to any platform
          </div>
        `}
      </div>
    `;
  }

  /**
   * Show not found message
   */
  private showNotFound(searchValue: string): void {
    if (!this.resultContainer) return;

    const config = EXTERNAL_ID_CONFIGS[this.currentType];

    this.resultContainer.innerHTML = `
      <div class="external-id-result not-found">
        <div class="external-id-result-header">
          <strong>No mapping found</strong>
        </div>
        <div class="external-id-result-meta">
          No content mapping exists for ${config.name} ID: ${escapeHtml(searchValue)}
        </div>
        <div class="external-id-result-action">
          You can create a new mapping with this ID.
        </div>
      </div>
    `;
  }

  /**
   * Show error message
   */
  private showError(message: string): void {
    if (!this.resultContainer) return;

    this.resultContainer.innerHTML = `
      <div class="external-id-result error">
        <span class="external-id-result-error-icon" aria-hidden="true">\u26A0</span>
        <span>${escapeHtml(message)}</span>
      </div>
    `;
  }

  /**
   * Get current values
   */
  getValues(): { type: ExternalIdType; value: string } {
    return {
      type: this.currentType,
      value: this.idInput?.value.trim() || ''
    };
  }

  /**
   * Get API params for current values
   */
  getApiParams(): ContentLookupParams | null {
    const { type, value } = this.getValues();
    if (!value || !this.validateInput()) return null;

    return EXTERNAL_ID_CONFIGS[type].formatForApi(value);
  }

  /**
   * Set values programmatically
   */
  setValues(type: ExternalIdType, value: string): void {
    this.currentType = type;

    if (this.typeSelect) {
      this.typeSelect.value = type;
    }

    if (this.idInput) {
      this.idInput.value = value;
    }

    this.updateForType();
    this.validateInput();
  }

  /**
   * Clear the input
   */
  clear(): void {
    if (this.idInput) {
      this.idInput.value = '';
      this.idInput.classList.remove('valid', 'invalid');
    }

    if (this.resultContainer) {
      this.resultContainer.innerHTML = '';
    }
  }

  /**
   * Destroy the component and cleanup
   */
  destroy(): void {
    if (this.typeSelect && this.typeChangeHandler) {
      this.typeSelect.removeEventListener('change', this.typeChangeHandler);
    }

    if (this.idInput) {
      if (this.inputHandler) {
        this.idInput.removeEventListener('input', this.inputHandler);
      }
      if (this.keydownHandler) {
        this.idInput.removeEventListener('keydown', this.keydownHandler);
      }
    }

    if (this.searchButton && this.searchHandler) {
      this.searchButton.removeEventListener('click', this.searchHandler);
    }

    this.typeChangeHandler = null;
    this.inputHandler = null;
    this.searchHandler = null;
    this.keydownHandler = null;
    this.element = null;
    this.typeSelect = null;
    this.idInput = null;
    this.searchButton = null;
    this.resultContainer = null;
    this.hintElement = null;
  }
}

/**
 * Validate an external ID value
 */
export function validateExternalId(type: ExternalIdType, value: string): boolean {
  const config = EXTERNAL_ID_CONFIGS[type];
  return config.pattern.test(value.trim());
}

/**
 * Format an external ID for display
 */
export function formatExternalId(type: ExternalIdType, value: string | number): string {
  const config = EXTERNAL_ID_CONFIGS[type];
  return `${config.name}: ${value}`;
}

/**
 * Get external ID config
 */
export function getExternalIdConfig(type: ExternalIdType): ExternalIdConfig {
  return EXTERNAL_ID_CONFIGS[type];
}

export default ExternalIdLookup;
