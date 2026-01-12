// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * NotificationCenterManager - Toast history and notification center
 *
 * Features:
 * - Stores notification history
 * - Displays notification center panel
 * - Shows unread notification count badge
 * - Allows clearing notification history
 * - Persists history to localStorage
 */

import type { ToastType } from '../lib/toast';
import { createLogger } from '../lib/logger';
import { SafeStorage } from '../lib/utils/SafeStorage';

const logger = createLogger('NotificationCenterManager');

/**
 * Stored notification structure
 */
interface StoredNotification {
  id: string;
  type: ToastType;
  message: string;
  title?: string;
  timestamp: number;
  read: boolean;
}

/**
 * Configuration for notification center
 */
interface NotificationCenterConfig {
  maxHistory: number;       // Maximum notifications to store
  storageKey: string;       // localStorage key
  autoClearAfter: number;   // Auto-clear after N days (0 = never)
}

const DEFAULT_CONFIG: NotificationCenterConfig = {
  maxHistory: 100,
  storageKey: 'notification-history',
  autoClearAfter: 7  // 7 days
};

export class NotificationCenterManager {
  private config: NotificationCenterConfig;
  private notifications: StoredNotification[] = [];
  private isOpen: boolean = false;
  private panel: HTMLElement | null = null;
  private badge: HTMLElement | null = null;
  private nextId: number = 1;

  // Event handler references for cleanup
  private closeBtnClickHandler: (() => void) | null = null;
  private clearBtnClickHandler: (() => void) | null = null;
  private markReadBtnClickHandler: (() => void) | null = null;
  private documentClickHandler: ((e: MouseEvent) => void) | null = null;
  private documentKeydownHandler: ((e: KeyboardEvent) => void) | null = null;
  private toggleBtnClickHandler: (() => void) | null = null;
  private contentClickHandler: ((e: Event) => void) | null = null;

  constructor(config: Partial<NotificationCenterConfig> = {}) {
    this.config = { ...DEFAULT_CONFIG, ...config };
  }

  /**
   * Initialize the notification center
   */
  init(): void {
    this.loadFromStorage();
    this.createPanel();
    this.setupToggleButton();
    this.updateBadge();
    logger.debug('NotificationCenterManager initialized');
  }

  /**
   * Load notification history from localStorage
   */
  private loadFromStorage(): void {
    const stored = SafeStorage.getJSON<StoredNotification[]>(this.config.storageKey, []);
    if (Array.isArray(stored) && stored.length > 0) {
      this.notifications = stored;
      // Find highest ID for nextId
      const maxId = this.notifications.reduce((max, n) => {
        const id = parseInt(n.id.replace('notif-', ''), 10);
        return isNaN(id) ? max : Math.max(max, id);
      }, 0);
      this.nextId = maxId + 1;

      // Auto-clear old notifications
      if (this.config.autoClearAfter > 0) {
        const cutoff = Date.now() - (this.config.autoClearAfter * 24 * 60 * 60 * 1000);
        this.notifications = this.notifications.filter(n => n.timestamp > cutoff);
        this.saveToStorage();
      }
    }
  }

  /**
   * Save notification history to localStorage
   */
  private saveToStorage(): void {
    SafeStorage.setJSON(this.config.storageKey, this.notifications);
  }

  /**
   * Add a notification to history
   * This should be called by ToastManager when showing a toast
   */
  addNotification(type: ToastType, message: string, title?: string): void {
    const notification: StoredNotification = {
      id: `notif-${this.nextId++}`,
      type,
      message,
      title,
      timestamp: Date.now(),
      read: false
    };

    // Add to beginning of array
    this.notifications.unshift(notification);

    // Trim to max history
    if (this.notifications.length > this.config.maxHistory) {
      this.notifications = this.notifications.slice(0, this.config.maxHistory);
    }

    this.saveToStorage();
    this.updateBadge();

    // Update panel if open
    if (this.isOpen) {
      this.renderNotifications();
    }
  }

  /**
   * Create the notification center panel
   */
  private createPanel(): void {
    // Create panel container
    this.panel = document.createElement('div');
    this.panel.id = 'notification-center-panel';
    this.panel.className = 'notification-center-panel';
    this.panel.setAttribute('role', 'dialog');
    this.panel.setAttribute('aria-label', 'Notification center');
    this.panel.setAttribute('aria-hidden', 'true');

    // Create panel content
    this.panel.innerHTML = `
      <div class="notification-center-header">
        <h3 class="notification-center-title">Notifications</h3>
        <div class="notification-center-actions">
          <button class="notification-mark-read-btn" aria-label="Mark all as read" title="Mark all as read">
            <span aria-hidden="true">&#x2713;</span>
          </button>
          <button class="notification-clear-btn" aria-label="Clear all notifications" title="Clear all">
            <span aria-hidden="true">&#x1F5D1;</span>
          </button>
          <button class="notification-close-btn" aria-label="Close notification center">
            <span aria-hidden="true">&times;</span>
          </button>
        </div>
      </div>
      <div class="notification-center-content" role="list" aria-label="Notification list">
        <!-- Notifications will be rendered here -->
      </div>
    `;

    // Add event listeners with stored references
    const closeBtn = this.panel.querySelector('.notification-close-btn');
    if (closeBtn) {
      this.closeBtnClickHandler = () => this.close();
      closeBtn.addEventListener('click', this.closeBtnClickHandler);
    }

    const clearBtn = this.panel.querySelector('.notification-clear-btn');
    if (clearBtn) {
      this.clearBtnClickHandler = () => this.clearAll();
      clearBtn.addEventListener('click', this.clearBtnClickHandler);
    }

    const markReadBtn = this.panel.querySelector('.notification-mark-read-btn');
    if (markReadBtn) {
      this.markReadBtnClickHandler = () => this.markAllRead();
      markReadBtn.addEventListener('click', this.markReadBtnClickHandler);
    }

    // Event delegation for notification items (click and dismiss)
    const content = this.panel.querySelector('.notification-center-content');
    if (content) {
      this.contentClickHandler = (e: Event) => {
        const target = e.target as HTMLElement;

        // Handle dismiss button click
        const dismissBtn = target.closest('.notification-dismiss');
        if (dismissBtn) {
          e.stopPropagation();
          const item = dismissBtn.closest('.notification-item');
          const id = item?.getAttribute('data-id');
          if (id) {
            this.removeNotification(id);
          }
          return;
        }

        // Handle notification item click (mark as read)
        const item = target.closest('.notification-item');
        if (item) {
          const id = item.getAttribute('data-id');
          if (id) {
            this.markAsRead(id);
          }
        }
      };
      content.addEventListener('click', this.contentClickHandler);
    }

    // Close on outside click
    this.documentClickHandler = (e: MouseEvent) => {
      if (this.isOpen && this.panel && !this.panel.contains(e.target as Node)) {
        const toggleBtn = document.getElementById('notification-center-toggle');
        if (!toggleBtn || !toggleBtn.contains(e.target as Node)) {
          this.close();
        }
      }
    };
    document.addEventListener('click', this.documentClickHandler);

    // Close on Escape key
    this.documentKeydownHandler = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && this.isOpen) {
        this.close();
      }
    };
    document.addEventListener('keydown', this.documentKeydownHandler);

    document.body.appendChild(this.panel);
  }

  /**
   * Setup toggle button in header
   */
  private setupToggleButton(): void {
    const toggleBtn = document.getElementById('notification-center-toggle');
    if (toggleBtn) {
      this.toggleBtnClickHandler = () => this.toggle();
      toggleBtn.addEventListener('click', this.toggleBtnClickHandler);

      // Get or create badge
      this.badge = toggleBtn.querySelector('.notification-badge');
      if (!this.badge) {
        this.badge = document.createElement('span');
        this.badge.className = 'notification-badge';
        this.badge.setAttribute('aria-label', 'Unread notifications');
        toggleBtn.appendChild(this.badge);
      }
    }
  }

  /**
   * Update the unread badge count
   */
  private updateBadge(): void {
    const unreadCount = this.notifications.filter(n => !n.read).length;

    if (this.badge) {
      if (unreadCount > 0) {
        this.badge.textContent = unreadCount > 99 ? '99+' : unreadCount.toString();
        this.badge.hidden = false;
      } else {
        this.badge.hidden = true;
      }
    }

    // Update toggle button aria-label
    const toggleBtn = document.getElementById('notification-center-toggle');
    if (toggleBtn) {
      toggleBtn.setAttribute('aria-label', `Notifications (${unreadCount} unread)`);
    }
  }

  /**
   * Toggle the notification center panel
   */
  toggle(): void {
    if (this.isOpen) {
      this.close();
    } else {
      this.open();
    }
  }

  /**
   * Open the notification center panel
   */
  open(): void {
    if (!this.panel) return;

    this.isOpen = true;
    this.panel.classList.add('open');
    this.panel.setAttribute('aria-hidden', 'false');
    this.renderNotifications();

    // Focus the panel for accessibility
    const firstFocusable = this.panel.querySelector('button') as HTMLElement;
    if (firstFocusable) {
      firstFocusable.focus();
    }

    // Update toggle button
    const toggleBtn = document.getElementById('notification-center-toggle');
    if (toggleBtn) {
      toggleBtn.setAttribute('aria-expanded', 'true');
    }
  }

  /**
   * Close the notification center panel
   */
  close(): void {
    if (!this.panel) return;

    this.isOpen = false;
    this.panel.classList.remove('open');
    this.panel.setAttribute('aria-hidden', 'true');

    // Update toggle button
    const toggleBtn = document.getElementById('notification-center-toggle');
    if (toggleBtn) {
      toggleBtn.setAttribute('aria-expanded', 'false');
      toggleBtn.focus();
    }
  }

  /**
   * Render notifications list
   * Note: Click handlers are managed via event delegation in createPanel()
   */
  private renderNotifications(): void {
    const content = this.panel?.querySelector('.notification-center-content');
    if (!content) return;

    if (this.notifications.length === 0) {
      content.innerHTML = `
        <div class="notification-empty">
          <span class="notification-empty-icon" aria-hidden="true">&#x1F514;</span>
          <span>No notifications</span>
        </div>
      `;
      return;
    }

    const html = this.notifications.map(n => this.renderNotificationItem(n)).join('');
    content.innerHTML = html;
    // Click handlers for items are managed via event delegation in createPanel()
  }

  /**
   * Render a single notification item
   */
  private renderNotificationItem(notification: StoredNotification): string {
    const timeAgo = this.formatTimeAgo(notification.timestamp);
    const typeIcon = this.getTypeIcon(notification.type);
    const unreadClass = notification.read ? '' : 'unread';

    return `
      <div class="notification-item ${notification.type} ${unreadClass}"
           data-id="${notification.id}"
           role="listitem"
           tabindex="0">
        <div class="notification-item-icon" aria-hidden="true">${typeIcon}</div>
        <div class="notification-item-content">
          ${notification.title ? `<div class="notification-item-title">${this.escapeHtml(notification.title)}</div>` : ''}
          <div class="notification-item-message">${this.escapeHtml(notification.message)}</div>
          <div class="notification-item-time">${timeAgo}</div>
        </div>
        <button class="notification-dismiss" aria-label="Dismiss notification">
          <span aria-hidden="true">&times;</span>
        </button>
      </div>
    `;
  }

  /**
   * Get icon for notification type
   */
  private getTypeIcon(type: ToastType): string {
    const icons: Record<ToastType, string> = {
      info: '\u2139',      // ℹ
      success: '\u2713',   // ✓
      warning: '\u26A0',   // ⚠
      error: '\u2717'      // ✗
    };
    return icons[type] || icons.info;
  }

  /**
   * Format timestamp as "time ago" string
   */
  private formatTimeAgo(timestamp: number): string {
    const seconds = Math.floor((Date.now() - timestamp) / 1000);

    if (seconds < 60) return 'Just now';
    if (seconds < 3600) return `${Math.floor(seconds / 60)}m ago`;
    if (seconds < 86400) return `${Math.floor(seconds / 3600)}h ago`;
    if (seconds < 604800) return `${Math.floor(seconds / 86400)}d ago`;

    return new Date(timestamp).toLocaleDateString();
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
   * Mark a notification as read
   */
  markAsRead(id: string): void {
    const notification = this.notifications.find(n => n.id === id);
    if (notification && !notification.read) {
      notification.read = true;
      this.saveToStorage();
      this.updateBadge();
      this.renderNotifications();
    }
  }

  /**
   * Mark all notifications as read
   */
  markAllRead(): void {
    let changed = false;
    this.notifications.forEach(n => {
      if (!n.read) {
        n.read = true;
        changed = true;
      }
    });

    if (changed) {
      this.saveToStorage();
      this.updateBadge();
      this.renderNotifications();
    }
  }

  /**
   * Remove a single notification
   */
  removeNotification(id: string): void {
    this.notifications = this.notifications.filter(n => n.id !== id);
    this.saveToStorage();
    this.updateBadge();
    this.renderNotifications();
  }

  /**
   * Clear all notifications
   */
  clearAll(): void {
    this.notifications = [];
    this.saveToStorage();
    this.updateBadge();
    this.renderNotifications();
  }

  /**
   * Get unread count
   */
  getUnreadCount(): number {
    return this.notifications.filter(n => !n.read).length;
  }

  /**
   * Get all notifications
   */
  getNotifications(): StoredNotification[] {
    return [...this.notifications];
  }

  /**
   * Remove all event listeners for cleanup
   */
  private removeEventListeners(): void {
    // Remove panel button handlers
    if (this.panel) {
      const closeBtn = this.panel.querySelector('.notification-close-btn');
      if (closeBtn && this.closeBtnClickHandler) {
        closeBtn.removeEventListener('click', this.closeBtnClickHandler);
        this.closeBtnClickHandler = null;
      }

      const clearBtn = this.panel.querySelector('.notification-clear-btn');
      if (clearBtn && this.clearBtnClickHandler) {
        clearBtn.removeEventListener('click', this.clearBtnClickHandler);
        this.clearBtnClickHandler = null;
      }

      const markReadBtn = this.panel.querySelector('.notification-mark-read-btn');
      if (markReadBtn && this.markReadBtnClickHandler) {
        markReadBtn.removeEventListener('click', this.markReadBtnClickHandler);
        this.markReadBtnClickHandler = null;
      }

      // Remove content delegation handler
      const content = this.panel.querySelector('.notification-center-content');
      if (content && this.contentClickHandler) {
        content.removeEventListener('click', this.contentClickHandler);
        this.contentClickHandler = null;
      }
    }

    // Remove toggle button handler
    const toggleBtn = document.getElementById('notification-center-toggle');
    if (toggleBtn && this.toggleBtnClickHandler) {
      toggleBtn.removeEventListener('click', this.toggleBtnClickHandler);
      this.toggleBtnClickHandler = null;
    }

    // Remove document-level handlers
    if (this.documentClickHandler) {
      document.removeEventListener('click', this.documentClickHandler);
      this.documentClickHandler = null;
    }

    if (this.documentKeydownHandler) {
      document.removeEventListener('keydown', this.documentKeydownHandler);
      this.documentKeydownHandler = null;
    }
  }

  /**
   * Destroy the notification center manager and clean up resources
   */
  destroy(): void {
    this.removeEventListeners();
    this.close();

    // Remove panel from DOM
    if (this.panel && this.panel.parentNode) {
      this.panel.parentNode.removeChild(this.panel);
      this.panel = null;
    }
  }
}

export default NotificationCenterManager;
