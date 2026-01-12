// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * RecentSearchesManager - Recent Searches
 *
 * Provides quick access to recently used filter selections.
 * Displays recent users and media types as quick-select options.
 *
 * Features:
 * - Tracks recently selected users and media types
 * - Persists history to localStorage
 * - Maximum 5 recent items per category
 * - One-click to apply recent selection
 * - Keyboard accessible
 */

import { SafeStorage } from '../lib/utils/SafeStorage';

const STORAGE_KEY = 'cartographus-recent-searches';
const MAX_RECENT_ITEMS = 5;

interface RecentSearches {
  users: string[];
  mediaTypes: string[];
}

/**
 * Callback when a recent search is selected
 */
export type RecentSearchCallback = (type: 'user' | 'mediaType', value: string) => void;

export class RecentSearchesManager {
  private recentSearches: RecentSearches = { users: [], mediaTypes: [] };
  private onSelectCallback: RecentSearchCallback | null = null;
  private userDropdownContainer: HTMLElement | null = null;
  private mediaTypeDropdownContainer: HTMLElement | null = null;

  // Event handler references for cleanup
  private userFilterHandler: (() => void) | null = null;
  private mediaTypeFilterHandler: (() => void) | null = null;

  constructor(onSelectCallback: RecentSearchCallback) {
    this.onSelectCallback = onSelectCallback;
    this.loadFromStorage();
    this.createRecentSearchContainers();
    this.setupFilterListeners();
    this.render();
  }

  /**
   * Load recent searches from localStorage
   */
  private loadFromStorage(): void {
    const stored = SafeStorage.getJSON<RecentSearches | null>(STORAGE_KEY, null);
    if (stored) {
      this.recentSearches = {
        users: Array.isArray(stored.users) ? stored.users.slice(0, MAX_RECENT_ITEMS) : [],
        mediaTypes: Array.isArray(stored.mediaTypes) ? stored.mediaTypes.slice(0, MAX_RECENT_ITEMS) : [],
      };
    }
  }

  /**
   * Save recent searches to localStorage
   */
  private saveToStorage(): void {
    SafeStorage.setJSON(STORAGE_KEY, this.recentSearches);
  }

  /**
   * Create containers for recent search dropdowns
   */
  private createRecentSearchContainers(): void {
    // Find filter groups
    const userFilterGroup = document.querySelector('#filter-users')?.closest('.filter-group');
    const mediaTypeFilterGroup = document.querySelector('#filter-media-types')?.closest('.filter-group');

    if (userFilterGroup) {
      this.userDropdownContainer = this.createDropdownContainer('recent-users', userFilterGroup as HTMLElement);
    }

    if (mediaTypeFilterGroup) {
      this.mediaTypeDropdownContainer = this.createDropdownContainer('recent-media-types', mediaTypeFilterGroup as HTMLElement);
    }
  }

  /**
   * Create a dropdown container for recent searches
   */
  private createDropdownContainer(id: string, filterGroup: HTMLElement): HTMLElement {
    // Check if already exists
    let container = filterGroup.querySelector(`.recent-searches-container`) as HTMLElement;
    if (container) return container;

    container = document.createElement('div');
    container.id = id;
    container.className = 'recent-searches-container';
    container.setAttribute('role', 'region');
    container.setAttribute('aria-label', 'Recent selections');

    // Insert after the select element
    const select = filterGroup.querySelector('select');
    if (select) {
      select.parentNode?.insertBefore(container, select.nextSibling);
    } else {
      filterGroup.appendChild(container);
    }

    return container;
  }

  /**
   * Setup listeners on filter dropdowns to track selections
   */
  private setupFilterListeners(): void {
    const userFilter = document.getElementById('filter-users') as HTMLSelectElement;
    const mediaTypeFilter = document.getElementById('filter-media-types') as HTMLSelectElement;

    if (userFilter) {
      this.userFilterHandler = () => {
        if (userFilter.value) {
          this.addRecentSearch('users', userFilter.value);
        }
      };
      userFilter.addEventListener('change', this.userFilterHandler);
    }

    if (mediaTypeFilter) {
      this.mediaTypeFilterHandler = () => {
        if (mediaTypeFilter.value) {
          this.addRecentSearch('mediaTypes', mediaTypeFilter.value);
        }
      };
      mediaTypeFilter.addEventListener('change', this.mediaTypeFilterHandler);
    }
  }

  /**
   * Add a recent search
   */
  addRecentSearch(type: 'users' | 'mediaTypes', value: string): void {
    if (!value) return;

    // Remove if already exists (will be moved to top)
    const index = this.recentSearches[type].indexOf(value);
    if (index > -1) {
      this.recentSearches[type].splice(index, 1);
    }

    // Add to beginning
    this.recentSearches[type].unshift(value);

    // Limit to max items
    if (this.recentSearches[type].length > MAX_RECENT_ITEMS) {
      this.recentSearches[type] = this.recentSearches[type].slice(0, MAX_RECENT_ITEMS);
    }

    this.saveToStorage();
    this.render();
  }

  /**
   * Remove a recent search
   */
  removeRecentSearch(type: 'users' | 'mediaTypes', value: string): void {
    const index = this.recentSearches[type].indexOf(value);
    if (index > -1) {
      this.recentSearches[type].splice(index, 1);
      this.saveToStorage();
      this.render();
    }
  }

  /**
   * Clear all recent searches
   */
  clearAll(): void {
    this.recentSearches = { users: [], mediaTypes: [] };
    this.saveToStorage();
    this.render();
  }

  /**
   * Render the recent searches UI
   */
  private render(): void {
    if (this.userDropdownContainer) {
      this.renderDropdown(this.userDropdownContainer, 'user', this.recentSearches.users);
    }

    if (this.mediaTypeDropdownContainer) {
      this.renderDropdown(this.mediaTypeDropdownContainer, 'mediaType', this.recentSearches.mediaTypes);
    }
  }

  /**
   * Render a single dropdown with recent items
   */
  private renderDropdown(
    container: HTMLElement,
    type: 'user' | 'mediaType',
    items: string[]
  ): void {
    if (items.length === 0) {
      container.innerHTML = '';
      container.style.display = 'none';
      return;
    }

    container.style.display = 'block';
    const label = type === 'user' ? 'Recent users' : 'Recent media types';

    container.innerHTML = `
      <div class="recent-searches-header">
        <span class="recent-searches-label">${label}</span>
        <button type="button" class="recent-searches-clear" aria-label="Clear ${label}">Clear</button>
      </div>
      <div class="recent-searches-list" role="listbox" aria-label="${label}">
        ${items.map((item, index) => `
          <button
            type="button"
            class="recent-search-item"
            data-value="${this.escapeHtml(item)}"
            data-type="${type}"
            role="option"
            tabindex="${index === 0 ? '0' : '-1'}"
            aria-label="Select ${this.escapeHtml(item)}"
          >
            <span class="recent-search-icon" aria-hidden="true">
              <svg width="12" height="12" viewBox="0 0 16 16" fill="currentColor">
                <path d="M8 3.5a.5.5 0 0 0-1 0V9a.5.5 0 0 0 .252.434l3.5 2a.5.5 0 0 0 .496-.868L8 8.71V3.5z"/>
                <path d="M8 16A8 8 0 1 0 8 0a8 8 0 0 0 0 16zm7-8A7 7 0 1 1 1 8a7 7 0 0 1 14 0z"/>
              </svg>
            </span>
            <span class="recent-search-text">${this.escapeHtml(item)}</span>
            <span class="recent-search-remove" aria-hidden="true" data-remove="true">&times;</span>
          </button>
        `).join('')}
      </div>
    `;

    // Setup event listeners
    this.setupDropdownListeners(container, type);
  }

  /**
   * Setup event listeners for a dropdown
   */
  private setupDropdownListeners(container: HTMLElement, type: 'user' | 'mediaType'): void {
    // Clear button
    const clearBtn = container.querySelector('.recent-searches-clear');
    clearBtn?.addEventListener('click', () => {
      const typeKey = type === 'user' ? 'users' : 'mediaTypes';
      this.recentSearches[typeKey] = [];
      this.saveToStorage();
      this.render();
    });

    // Item buttons
    const items = container.querySelectorAll('.recent-search-item');
    items.forEach((item) => {
      item.addEventListener('click', (e) => {
        const target = e.target as HTMLElement;
        const value = (item as HTMLElement).getAttribute('data-value');
        const itemType = (item as HTMLElement).getAttribute('data-type') as 'user' | 'mediaType';

        // Check if remove button was clicked
        if (target.closest('.recent-search-remove')) {
          e.stopPropagation();
          if (value) {
            const typeKey = itemType === 'user' ? 'users' : 'mediaTypes';
            this.removeRecentSearch(typeKey, value);
          }
          return;
        }

        // Apply the selection
        if (value && this.onSelectCallback) {
          this.onSelectCallback(itemType, value);
        }
      });

      // Keyboard navigation
      item.addEventListener('keydown', (e: Event) => {
        const keyEvent = e as KeyboardEvent;
        const itemElement = item as HTMLElement;
        const list = itemElement.closest('.recent-searches-list');
        const allItems = list?.querySelectorAll('.recent-search-item');

        if (!allItems) return;

        const currentIndex = Array.from(allItems).indexOf(item);

        switch (keyEvent.key) {
          case 'ArrowDown':
            keyEvent.preventDefault();
            if (currentIndex < allItems.length - 1) {
              (allItems[currentIndex + 1] as HTMLElement).focus();
            }
            break;
          case 'ArrowUp':
            keyEvent.preventDefault();
            if (currentIndex > 0) {
              (allItems[currentIndex - 1] as HTMLElement).focus();
            }
            break;
          case 'Delete':
          case 'Backspace':
            keyEvent.preventDefault();
            const value = itemElement.getAttribute('data-value');
            const itemType = itemElement.getAttribute('data-type') as 'user' | 'mediaType';
            if (value) {
              const typeKey = itemType === 'user' ? 'users' : 'mediaTypes';
              this.removeRecentSearch(typeKey, value);
              // Focus next or previous item
              if (currentIndex < allItems.length - 1) {
                (allItems[currentIndex] as HTMLElement)?.focus();
              } else if (currentIndex > 0) {
                (allItems[currentIndex - 1] as HTMLElement)?.focus();
              }
            }
            break;
        }
      });
    });
  }

  /**
   * Escape HTML for safe rendering
   */
  private escapeHtml(text: string): string {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
  }

  /**
   * Get recent users
   */
  getRecentUsers(): string[] {
    return [...this.recentSearches.users];
  }

  /**
   * Get recent media types
   */
  getRecentMediaTypes(): string[] {
    return [...this.recentSearches.mediaTypes];
  }

  /**
   * Destroy the manager
   */
  destroy(): void {
    // Remove filter event listeners
    const userFilter = document.getElementById('filter-users') as HTMLSelectElement;
    const mediaTypeFilter = document.getElementById('filter-media-types') as HTMLSelectElement;

    if (userFilter && this.userFilterHandler) {
      userFilter.removeEventListener('change', this.userFilterHandler);
      this.userFilterHandler = null;
    }

    if (mediaTypeFilter && this.mediaTypeFilterHandler) {
      mediaTypeFilter.removeEventListener('change', this.mediaTypeFilterHandler);
      this.mediaTypeFilterHandler = null;
    }

    // Remove dropdown containers
    if (this.userDropdownContainer) {
      this.userDropdownContainer.remove();
      this.userDropdownContainer = null;
    }

    if (this.mediaTypeDropdownContainer) {
      this.mediaTypeDropdownContainer.remove();
      this.mediaTypeDropdownContainer = null;
    }

    this.onSelectCallback = null;
  }
}
