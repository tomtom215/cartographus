// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * UserSelector - Searchable user dropdown component
 *
 * Provides a searchable dropdown for selecting users across platforms
 * with filtering and multi-select capabilities.
 */

import { escapeHtml } from '../sanitize';
import type { LinkedUserInfo } from '../types/cross-platform';
import { getPlatformConfig, type Platform, isValidPlatform } from './PlatformBadge';

/**
 * User item for the selector
 */
export interface UserSelectorItem {
  id: number;
  username?: string;
  friendly_name?: string;
  email?: string;
  source: string;
  server_id: string;
}

/**
 * Callback when user(s) are selected
 */
export type UserSelectCallback = (users: UserSelectorItem[]) => void;

/**
 * Options for the UserSelector component
 */
export interface UserSelectorOptions {
  /** Available users to select from */
  users: UserSelectorItem[];
  /** Allow multiple selection */
  multiple?: boolean;
  /** Initial selected user IDs */
  selectedIds?: number[];
  /** Placeholder text */
  placeholder?: string;
  /** Filter by platform */
  platformFilter?: Platform;
  /** Exclude certain user IDs */
  excludeIds?: number[];
  /** Callback when selection changes */
  onSelect?: UserSelectCallback;
  /** Maximum items to show in dropdown */
  maxItems?: number;
  /** Show platform badges */
  showPlatformBadges?: boolean;
}

/**
 * UserSelector class for selecting users
 */
export class UserSelector {
  private options: UserSelectorOptions;
  private element: HTMLElement | null = null;
  private inputElement: HTMLInputElement | null = null;
  private dropdownElement: HTMLElement | null = null;
  private selectedTagsElement: HTMLElement | null = null;

  // Event handlers for cleanup
  private inputHandler: (() => void) | null = null;
  private focusHandler: (() => void) | null = null;
  private blurHandler: (() => void) | null = null;
  private keydownHandler: ((e: KeyboardEvent) => void) | null = null;
  private documentClickHandler: ((e: MouseEvent) => void) | null = null;

  private filteredUsers: UserSelectorItem[] = [];
  private selectedUsers: UserSelectorItem[] = [];
  private isOpen: boolean = false;
  private highlightedIndex: number = -1;

  constructor(options: UserSelectorOptions) {
    this.options = {
      multiple: false,
      placeholder: 'Search users...',
      maxItems: 50,
      showPlatformBadges: true,
      excludeIds: [],
      ...options
    };

    // Initialize selected users
    if (this.options.selectedIds) {
      this.selectedUsers = this.options.users.filter(u =>
        this.options.selectedIds!.includes(u.id)
      );
    }

    this.filteredUsers = this.getFilteredUsers('');
  }

  /**
   * Render the user selector
   */
  render(): HTMLElement {
    this.element = document.createElement('div');
    this.element.className = 'user-selector';

    const selectedDisplay = this.options.multiple
      ? `<div class="user-selector-tags"></div>`
      : '';

    this.element.innerHTML = `
      ${selectedDisplay}
      <div class="user-selector-input-wrapper">
        <input
          type="text"
          class="user-selector-input form-input"
          placeholder="${escapeHtml(this.options.placeholder || '')}"
          autocomplete="off"
          aria-label="Search users"
          aria-expanded="false"
          aria-haspopup="listbox"
          role="combobox"
        />
        <span class="user-selector-arrow" aria-hidden="true">\u25BC</span>
      </div>
      <div class="user-selector-dropdown" role="listbox" aria-label="User options"></div>
    `;

    // Get element references
    this.inputElement = this.element.querySelector('.user-selector-input');
    this.dropdownElement = this.element.querySelector('.user-selector-dropdown');
    this.selectedTagsElement = this.element.querySelector('.user-selector-tags');

    // Set up event handlers
    this.setupEventHandlers();

    // Render initial state
    if (this.options.multiple && this.selectedUsers.length > 0) {
      this.renderSelectedTags();
    }

    return this.element;
  }

  /**
   * Set up event handlers
   */
  private setupEventHandlers(): void {
    if (this.inputElement) {
      // Input handler for filtering
      this.inputHandler = () => {
        const query = this.inputElement!.value.trim();
        this.filteredUsers = this.getFilteredUsers(query);
        this.renderDropdown();
        this.openDropdown();
      };
      this.inputElement.addEventListener('input', this.inputHandler);

      // Focus handler
      this.focusHandler = () => {
        this.openDropdown();
      };
      this.inputElement.addEventListener('focus', this.focusHandler);

      // Blur handler with delay to allow click
      this.blurHandler = () => {
        setTimeout(() => {
          if (!this.element?.contains(document.activeElement)) {
            this.closeDropdown();
          }
        }, 150);
      };
      this.inputElement.addEventListener('blur', this.blurHandler);

      // Keyboard navigation
      this.keydownHandler = (e: KeyboardEvent) => {
        this.handleKeydown(e);
      };
      this.inputElement.addEventListener('keydown', this.keydownHandler);
    }

    // Document click handler for closing dropdown
    this.documentClickHandler = (e: MouseEvent) => {
      if (this.element && !this.element.contains(e.target as Node)) {
        this.closeDropdown();
      }
    };
    document.addEventListener('click', this.documentClickHandler);
  }

  /**
   * Handle keyboard navigation
   */
  private handleKeydown(e: KeyboardEvent): void {
    switch (e.key) {
      case 'ArrowDown':
        e.preventDefault();
        this.highlightNext();
        break;
      case 'ArrowUp':
        e.preventDefault();
        this.highlightPrevious();
        break;
      case 'Enter':
        e.preventDefault();
        if (this.highlightedIndex >= 0 && this.highlightedIndex < this.filteredUsers.length) {
          this.selectUser(this.filteredUsers[this.highlightedIndex]);
        }
        break;
      case 'Escape':
        this.closeDropdown();
        break;
    }
  }

  /**
   * Highlight next item
   */
  private highlightNext(): void {
    if (this.filteredUsers.length === 0) return;
    this.highlightedIndex = Math.min(this.highlightedIndex + 1, this.filteredUsers.length - 1);
    this.updateHighlight();
  }

  /**
   * Highlight previous item
   */
  private highlightPrevious(): void {
    if (this.filteredUsers.length === 0) return;
    this.highlightedIndex = Math.max(this.highlightedIndex - 1, 0);
    this.updateHighlight();
  }

  /**
   * Update highlight in dropdown
   */
  private updateHighlight(): void {
    if (!this.dropdownElement) return;

    const options = this.dropdownElement.querySelectorAll('.user-selector-option');
    options.forEach((option, index) => {
      option.classList.toggle('highlighted', index === this.highlightedIndex);
      if (index === this.highlightedIndex) {
        option.scrollIntoView({ block: 'nearest' });
      }
    });
  }

  /**
   * Get filtered users based on search query
   */
  private getFilteredUsers(query: string): UserSelectorItem[] {
    let users = this.options.users;

    // Exclude specified IDs
    if (this.options.excludeIds && this.options.excludeIds.length > 0) {
      users = users.filter(u => !this.options.excludeIds!.includes(u.id));
    }

    // Exclude already selected (for multiple selection)
    if (this.options.multiple) {
      const selectedIds = new Set(this.selectedUsers.map(u => u.id));
      users = users.filter(u => !selectedIds.has(u.id));
    }

    // Filter by platform
    if (this.options.platformFilter) {
      users = users.filter(u => u.source === this.options.platformFilter);
    }

    // Filter by search query
    if (query) {
      const lowerQuery = query.toLowerCase();
      users = users.filter(u => {
        const username = u.username?.toLowerCase() || '';
        const friendlyName = u.friendly_name?.toLowerCase() || '';
        const email = u.email?.toLowerCase() || '';
        return username.includes(lowerQuery) ||
               friendlyName.includes(lowerQuery) ||
               email.includes(lowerQuery);
      });
    }

    // Limit results
    return users.slice(0, this.options.maxItems);
  }

  /**
   * Render the dropdown options
   */
  private renderDropdown(): void {
    if (!this.dropdownElement) return;

    if (this.filteredUsers.length === 0) {
      this.dropdownElement.innerHTML = `
        <div class="user-selector-empty">No users found</div>
      `;
      return;
    }

    this.dropdownElement.innerHTML = this.filteredUsers.map((user, index) => {
      const displayName = user.friendly_name || user.username || `User ${user.id}`;
      const subtitle = user.username && user.friendly_name ? user.username : '';
      const isSelected = this.selectedUsers.some(u => u.id === user.id);
      const platformBadge = this.options.showPlatformBadges && isValidPlatform(user.source)
        ? this.renderPlatformBadge(user.source as Platform)
        : '';

      return `
        <div
          class="user-selector-option ${isSelected ? 'selected' : ''}"
          role="option"
          aria-selected="${isSelected}"
          data-index="${index}"
          data-user-id="${user.id}"
        >
          <div class="user-selector-option-info">
            <span class="user-selector-name">${escapeHtml(displayName)}</span>
            ${subtitle ? `<span class="user-selector-subtitle">${escapeHtml(subtitle)}</span>` : ''}
          </div>
          <div class="user-selector-platforms">
            ${platformBadge}
          </div>
        </div>
      `;
    }).join('');

    // Add click handlers to options
    const options = this.dropdownElement.querySelectorAll('.user-selector-option');
    options.forEach((option) => {
      option.addEventListener('click', () => {
        const index = parseInt(option.getAttribute('data-index') || '0', 10);
        if (index >= 0 && index < this.filteredUsers.length) {
          this.selectUser(this.filteredUsers[index]);
        }
      });
    });
  }

  /**
   * Render a platform badge
   */
  private renderPlatformBadge(platform: Platform): string {
    const config = getPlatformConfig(platform);
    return `<span class="platform-badge platform-badge--${platform}" title="${escapeHtml(config.name)}">${escapeHtml(config.icon)}</span>`;
  }

  /**
   * Select a user
   */
  private selectUser(user: UserSelectorItem): void {
    if (this.options.multiple) {
      // Add to selection
      if (!this.selectedUsers.some(u => u.id === user.id)) {
        this.selectedUsers.push(user);
        this.renderSelectedTags();
        this.filteredUsers = this.getFilteredUsers(this.inputElement?.value || '');
        this.renderDropdown();
      }
    } else {
      // Single selection
      this.selectedUsers = [user];
      if (this.inputElement) {
        this.inputElement.value = user.friendly_name || user.username || `User ${user.id}`;
      }
      this.closeDropdown();
    }

    // Trigger callback
    if (this.options.onSelect) {
      this.options.onSelect([...this.selectedUsers]);
    }
  }

  /**
   * Remove a user from selection
   */
  private deselectUser(userId: number): void {
    this.selectedUsers = this.selectedUsers.filter(u => u.id !== userId);
    this.renderSelectedTags();
    this.filteredUsers = this.getFilteredUsers(this.inputElement?.value || '');
    this.renderDropdown();

    // Trigger callback
    if (this.options.onSelect) {
      this.options.onSelect([...this.selectedUsers]);
    }
  }

  /**
   * Render selected user tags (for multiple selection)
   */
  private renderSelectedTags(): void {
    if (!this.selectedTagsElement) return;

    this.selectedTagsElement.innerHTML = this.selectedUsers.map(user => {
      const displayName = user.friendly_name || user.username || `User ${user.id}`;
      return `
        <span class="user-selector-tag" data-user-id="${user.id}">
          <span class="user-selector-tag-name">${escapeHtml(displayName)}</span>
          <button type="button" class="user-selector-tag-remove" aria-label="Remove ${escapeHtml(displayName)}">&times;</button>
        </span>
      `;
    }).join('');

    // Add remove handlers
    const removeButtons = this.selectedTagsElement.querySelectorAll('.user-selector-tag-remove');
    removeButtons.forEach(btn => {
      btn.addEventListener('click', (e) => {
        e.stopPropagation();
        const tag = (btn as HTMLElement).closest('.user-selector-tag');
        const userId = parseInt(tag?.getAttribute('data-user-id') || '0', 10);
        if (userId) {
          this.deselectUser(userId);
        }
      });
    });
  }

  /**
   * Open the dropdown
   */
  private openDropdown(): void {
    if (this.isOpen) return;

    this.isOpen = true;
    this.highlightedIndex = -1;
    this.renderDropdown();

    if (this.dropdownElement) {
      this.dropdownElement.classList.add('open');
    }

    if (this.inputElement) {
      this.inputElement.setAttribute('aria-expanded', 'true');
    }
  }

  /**
   * Close the dropdown
   */
  private closeDropdown(): void {
    if (!this.isOpen) return;

    this.isOpen = false;
    this.highlightedIndex = -1;

    if (this.dropdownElement) {
      this.dropdownElement.classList.remove('open');
    }

    if (this.inputElement) {
      this.inputElement.setAttribute('aria-expanded', 'false');
    }
  }

  /**
   * Get selected users
   */
  getSelectedUsers(): UserSelectorItem[] {
    return [...this.selectedUsers];
  }

  /**
   * Get selected user IDs
   */
  getSelectedIds(): number[] {
    return this.selectedUsers.map(u => u.id);
  }

  /**
   * Set selected users
   */
  setSelectedUsers(users: UserSelectorItem[]): void {
    this.selectedUsers = [...users];

    if (this.options.multiple) {
      this.renderSelectedTags();
    } else if (users.length > 0) {
      const user = users[0];
      if (this.inputElement) {
        this.inputElement.value = user.friendly_name || user.username || `User ${user.id}`;
      }
    }

    this.filteredUsers = this.getFilteredUsers('');
  }

  /**
   * Clear selection
   */
  clear(): void {
    this.selectedUsers = [];

    if (this.inputElement) {
      this.inputElement.value = '';
    }

    if (this.selectedTagsElement) {
      this.selectedTagsElement.innerHTML = '';
    }

    this.filteredUsers = this.getFilteredUsers('');

    if (this.options.onSelect) {
      this.options.onSelect([]);
    }
  }

  /**
   * Update available users
   */
  setUsers(users: UserSelectorItem[]): void {
    this.options.users = users;
    this.filteredUsers = this.getFilteredUsers(this.inputElement?.value || '');
    this.renderDropdown();
  }

  /**
   * Set platform filter
   */
  setPlatformFilter(platform: Platform | undefined): void {
    this.options.platformFilter = platform;
    this.filteredUsers = this.getFilteredUsers(this.inputElement?.value || '');
    this.renderDropdown();
  }

  /**
   * Destroy the component and cleanup
   */
  destroy(): void {
    if (this.inputElement) {
      if (this.inputHandler) {
        this.inputElement.removeEventListener('input', this.inputHandler);
      }
      if (this.focusHandler) {
        this.inputElement.removeEventListener('focus', this.focusHandler);
      }
      if (this.blurHandler) {
        this.inputElement.removeEventListener('blur', this.blurHandler);
      }
      if (this.keydownHandler) {
        this.inputElement.removeEventListener('keydown', this.keydownHandler);
      }
    }

    if (this.documentClickHandler) {
      document.removeEventListener('click', this.documentClickHandler);
    }

    this.inputHandler = null;
    this.focusHandler = null;
    this.blurHandler = null;
    this.keydownHandler = null;
    this.documentClickHandler = null;
    this.element = null;
    this.inputElement = null;
    this.dropdownElement = null;
    this.selectedTagsElement = null;
  }
}

/**
 * Convert LinkedUserInfo to UserSelectorItem
 */
export function linkedUserToSelectorItem(user: LinkedUserInfo): UserSelectorItem {
  return {
    id: user.internal_user_id,
    username: user.username,
    friendly_name: user.friendly_name,
    source: user.source,
    server_id: user.server_id
  };
}

export default UserSelector;
