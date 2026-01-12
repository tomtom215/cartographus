// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Progressive Onboarding Manager (Phase 3)
 * Provides contextual tips that appear as users explore features for the first time.
 * Tips are non-intrusive, dismissible, and persist discovery state.
 *
 * Features:
 * - Context-aware tips triggered by user interaction
 * - Feature discovery tracking
 * - Dismissible tooltips
 * - Keyboard accessible (Escape to dismiss)
 * - WCAG 2.1 AA compliant
 * - SafeStorage for private browsing fallback
 * - Blocked during wizard/tour to avoid conflicts
 */

import { SafeStorage } from '../lib/utils/SafeStorage';
import { createLogger } from '../lib/logger';

const logger = createLogger('ProgressiveOnboardingManager');

export interface ProgressiveTip {
  /** Unique identifier for the tip */
  id: string;
  /** CSS selector or element ID for the target element */
  targetSelector: string;
  /** Tip title */
  title: string;
  /** Tip description */
  description: string;
  /** Trigger event type */
  trigger: 'hover' | 'click' | 'focus' | 'visible';
  /** Position relative to target */
  position: 'top' | 'bottom' | 'left' | 'right';
  /** Category for grouping related tips */
  category: 'map' | 'analytics' | 'filters' | 'navigation' | 'settings' | 'general';
  /** Priority (lower = show sooner) */
  priority: number;
  /** Delay before showing tip (ms) */
  delay?: number;
  /** Prerequisites - other tips that must be dismissed first */
  prerequisites?: string[];
}

const PROGRESSIVE_TIPS: ProgressiveTip[] = [
  // =========================================================================
  // MAP TIPS (5 tips)
  // =========================================================================
  {
    id: 'map-mode-switch',
    targetSelector: '.map-mode-toggle',
    title: 'Map Visualization Modes',
    description: 'Switch between points, clusters, heatmap, and hexagon views to visualize your data differently.',
    trigger: 'hover',
    position: 'bottom',
    category: 'map',
    priority: 1
  },
  {
    id: 'view-mode-switch',
    targetSelector: '#view-mode-2d',
    title: '2D and 3D Views',
    description: 'Toggle between the flat map and interactive 3D globe for different perspectives on your data.',
    trigger: 'hover',
    position: 'bottom',
    category: 'map',
    priority: 2
  },
  {
    id: 'arc-overlay',
    targetSelector: '#arc-toggle',
    title: 'Server Connection Arcs',
    description: 'Enable arc overlay to see connections between your server and playback locations around the world.',
    trigger: 'hover',
    position: 'bottom',
    category: 'map',
    priority: 3
  },
  {
    id: 'hexagon-resolution',
    targetSelector: '#hexagon-resolution',
    title: 'Hexagon Detail Level',
    description: 'Adjust H3 resolution to see playback data aggregated at different geographic scales.',
    trigger: 'focus',
    position: 'bottom',
    category: 'map',
    priority: 4
  },
  {
    id: 'fullscreen-mode',
    targetSelector: '#fullscreen-toggle',
    title: 'Fullscreen Mode',
    description: 'Enter fullscreen mode for an immersive map viewing experience.',
    trigger: 'hover',
    position: 'left',
    category: 'map',
    priority: 5
  },

  // =========================================================================
  // MAIN NAVIGATION TABS (8 tips - one for each tab)
  // =========================================================================
  {
    id: 'nav-tab-maps',
    targetSelector: '#tab-maps',
    title: 'Maps View',
    description: 'Visualize playback locations on interactive 2D maps, 3D globe, heatmaps, or hexagon grids.',
    trigger: 'hover',
    position: 'bottom',
    category: 'navigation',
    priority: 1
  },
  {
    id: 'nav-tab-activity',
    targetSelector: '#tab-activity',
    title: 'Live Activity',
    description: 'Monitor real-time streaming activity. See who is watching, what they are watching, and streaming quality.',
    trigger: 'hover',
    position: 'bottom',
    category: 'navigation',
    priority: 2
  },
  {
    id: 'nav-tab-analytics',
    targetSelector: '#tab-analytics',
    title: 'Analytics Dashboard',
    description: 'Explore 47 charts across 10 analytics pages including content trends, user behavior, and performance metrics.',
    trigger: 'hover',
    position: 'bottom',
    category: 'navigation',
    priority: 3
  },
  {
    id: 'nav-tab-recently-added',
    targetSelector: '#tab-recently-added',
    title: 'Recently Added',
    description: 'Browse recently added movies, shows, and music. See new content across all your libraries.',
    trigger: 'hover',
    position: 'bottom',
    category: 'navigation',
    priority: 4
  },
  {
    id: 'nav-tab-server',
    targetSelector: '#tab-server',
    title: 'Server Management',
    description: 'View server status, sync data from Tautulli/Plex/Jellyfin/Emby, and manage server connections.',
    trigger: 'hover',
    position: 'bottom',
    category: 'navigation',
    priority: 5
  },
  {
    id: 'nav-tab-cross-platform',
    targetSelector: '#tab-cross-platform',
    title: 'Cross-Platform Analytics',
    description: 'Compare viewing habits across platforms. Link user accounts and see unified statistics.',
    trigger: 'hover',
    position: 'bottom',
    category: 'navigation',
    priority: 6
  },
  {
    id: 'nav-tab-data-governance',
    targetSelector: '#tab-data-governance',
    title: 'Data Governance',
    description: 'Manage data retention, GDPR compliance, backups, audit logs, and data quality controls.',
    trigger: 'hover',
    position: 'bottom',
    category: 'navigation',
    priority: 7
  },
  {
    id: 'nav-tab-newsletter',
    targetSelector: '#tab-newsletter',
    title: 'Newsletter System',
    description: 'Create and schedule automated newsletters with viewing stats, new content, and recommendations.',
    trigger: 'hover',
    position: 'bottom',
    category: 'navigation',
    priority: 8
  },

  // =========================================================================
  // ANALYTICS SUB-TABS (10 tips)
  // =========================================================================
  {
    id: 'analytics-overview',
    targetSelector: '#tab-analytics-overview',
    title: 'Analytics Overview',
    description: 'Summary dashboard with key metrics, trends, and quick insights about your media consumption.',
    trigger: 'hover',
    position: 'bottom',
    category: 'analytics',
    priority: 1
  },
  {
    id: 'analytics-content',
    targetSelector: '#tab-analytics-content',
    title: 'Content Analytics',
    description: 'Analyze content popularity, genre trends, collections, and playlists. See what content performs best.',
    trigger: 'hover',
    position: 'bottom',
    category: 'analytics',
    priority: 2
  },
  {
    id: 'analytics-users',
    targetSelector: '#tab-analytics-users',
    title: 'User Analytics',
    description: 'Track user engagement, watch patterns, and viewing preferences across your user base.',
    trigger: 'hover',
    position: 'bottom',
    category: 'analytics',
    priority: 3
  },
  {
    id: 'analytics-performance',
    targetSelector: '#tab-analytics-performance',
    title: 'Performance Metrics',
    description: 'Monitor transcoding, bandwidth usage, stream quality, and server performance over time.',
    trigger: 'hover',
    position: 'bottom',
    category: 'analytics',
    priority: 4
  },
  {
    id: 'analytics-geographic',
    targetSelector: '#tab-analytics-geographic',
    title: 'Geographic Analytics',
    description: 'Analyze viewing patterns by location, country, city, and timezone distribution.',
    trigger: 'hover',
    position: 'bottom',
    category: 'analytics',
    priority: 5
  },
  {
    id: 'analytics-advanced',
    targetSelector: '#tab-analytics-advanced',
    title: 'Advanced Analytics',
    description: 'Deep-dive analysis with correlation studies, trend predictions, and comparative insights.',
    trigger: 'hover',
    position: 'bottom',
    category: 'analytics',
    priority: 6
  },
  {
    id: 'analytics-library',
    targetSelector: '#tab-analytics-library',
    title: 'Library Analytics',
    description: 'Analyze your media library composition, growth, and content distribution across libraries.',
    trigger: 'hover',
    position: 'bottom',
    category: 'analytics',
    priority: 7
  },
  {
    id: 'analytics-users-profile',
    targetSelector: '#tab-analytics-users-profile',
    title: 'User Profiles',
    description: 'View detailed profiles for individual users including watch history, preferences, and IP history.',
    trigger: 'hover',
    position: 'bottom',
    category: 'analytics',
    priority: 8
  },
  {
    id: 'analytics-tautulli',
    targetSelector: '#tab-analytics-tautulli',
    title: 'Tautulli Data',
    description: 'Access Tautulli-specific data including metadata deep-dives, rating keys, and stream data.',
    trigger: 'hover',
    position: 'bottom',
    category: 'analytics',
    priority: 9
  },
  {
    id: 'analytics-wrapped',
    targetSelector: '#tab-analytics-wrapped',
    title: 'Annual Wrapped',
    description: 'Generate Spotify-style annual viewing reports with personalized statistics and achievements.',
    trigger: 'hover',
    position: 'bottom',
    category: 'analytics',
    priority: 10
  },

  // =========================================================================
  // CHART INTERACTION TIPS (3 tips)
  // =========================================================================
  {
    id: 'chart-export',
    targetSelector: '.chart-export-btn',
    title: 'Export Charts',
    description: 'Export any chart as PNG, SVG, or raw data for reports and presentations.',
    trigger: 'hover',
    position: 'left',
    category: 'analytics',
    priority: 11
  },
  {
    id: 'chart-maximize',
    targetSelector: '.chart-maximize-btn',
    title: 'Maximize Charts',
    description: 'Click to expand any chart to full screen for detailed analysis.',
    trigger: 'hover',
    position: 'left',
    category: 'analytics',
    priority: 12
  },
  {
    id: 'quick-insights',
    targetSelector: '#quick-insights-panel',
    title: 'Quick Insights',
    description: 'View AI-generated insights and trends automatically derived from your data.',
    trigger: 'visible',
    position: 'left',
    category: 'analytics',
    priority: 13
  },

  // =========================================================================
  // FILTER TIPS (4 tips)
  // =========================================================================
  {
    id: 'date-range-filter',
    targetSelector: '#filter-days',
    title: 'Quick Date Selection',
    description: 'Use preset date ranges or select custom dates for your analysis.',
    trigger: 'focus',
    position: 'bottom',
    category: 'filters',
    priority: 1
  },
  {
    id: 'quick-date-buttons',
    targetSelector: '.quick-date-buttons',
    title: 'Quick Date Presets',
    description: 'Click these buttons for instant time range selection: Today, Week, Month, Year.',
    trigger: 'hover',
    position: 'bottom',
    category: 'filters',
    priority: 2
  },
  {
    id: 'filter-presets',
    targetSelector: '#filter-preset-dropdown',
    title: 'Save Filter Presets',
    description: 'Save your current filter combination as a preset for quick access later.',
    trigger: 'hover',
    position: 'bottom',
    category: 'filters',
    priority: 3
  },
  {
    id: 'clear-filters',
    targetSelector: '#btn-clear-filters',
    title: 'Reset Filters',
    description: 'Clear all filters and return to the default 90-day view.',
    trigger: 'hover',
    position: 'bottom',
    category: 'filters',
    priority: 4
  },

  // =========================================================================
  // EXPORT TIPS (3 tips)
  // =========================================================================
  {
    id: 'export-csv',
    targetSelector: '#btn-export-csv',
    title: 'Export to CSV',
    description: 'Download playback data as a CSV file for spreadsheet analysis or external tools.',
    trigger: 'hover',
    position: 'bottom',
    category: 'general',
    priority: 1
  },
  {
    id: 'export-geojson',
    targetSelector: '#btn-export-geojson',
    title: 'Export to GeoJSON',
    description: 'Download location data in GeoJSON format for use in mapping and GIS applications.',
    trigger: 'hover',
    position: 'bottom',
    category: 'general',
    priority: 2
  },
  {
    id: 'export-geoparquet',
    targetSelector: '#btn-export-geoparquet',
    title: 'Export to GeoParquet',
    description: 'Download data in high-performance GeoParquet format for large-scale GIS analysis.',
    trigger: 'hover',
    position: 'bottom',
    category: 'general',
    priority: 3
  },

  // =========================================================================
  // SEARCH TIPS (2 tips)
  // =========================================================================
  {
    id: 'global-search',
    targetSelector: '#global-search',
    title: 'Global Search',
    description: 'Search across all content, users, and locations. Use filters to narrow results.',
    trigger: 'focus',
    position: 'bottom',
    category: 'general',
    priority: 4
  },
  {
    id: 'search-panel',
    targetSelector: '#tab-search',
    title: 'Advanced Search',
    description: 'Perform detailed searches with multiple criteria and view search history.',
    trigger: 'hover',
    position: 'bottom',
    category: 'general',
    priority: 5
  },

  // =========================================================================
  // DATA GOVERNANCE TIPS (5 tips)
  // =========================================================================
  {
    id: 'backup-restore',
    targetSelector: '#btn-create-backup',
    title: 'Database Backup',
    description: 'Create backups of your database for disaster recovery. Schedule automatic backups for peace of mind.',
    trigger: 'hover',
    position: 'bottom',
    category: 'settings',
    priority: 5
  },
  {
    id: 'data-retention',
    targetSelector: '.data-retention-settings',
    title: 'Data Retention',
    description: 'Configure how long to keep playback history. Older data can be automatically archived or deleted.',
    trigger: 'visible',
    position: 'right',
    category: 'settings',
    priority: 6
  },
  {
    id: 'audit-logs',
    targetSelector: '.audit-log-panel',
    title: 'Audit Logs',
    description: 'Review all system activities including logins, data access, and configuration changes.',
    trigger: 'visible',
    position: 'right',
    category: 'settings',
    priority: 7
  },
  {
    id: 'gdpr-compliance',
    targetSelector: '.gdpr-settings',
    title: 'GDPR Compliance',
    description: 'Manage user data requests, anonymization, and data subject access rights.',
    trigger: 'visible',
    position: 'right',
    category: 'settings',
    priority: 8
  },
  {
    id: 'dedupe-audit',
    targetSelector: '.dedupe-panel',
    title: 'Duplicate Detection',
    description: 'Find and manage duplicate playback records to ensure data accuracy.',
    trigger: 'visible',
    position: 'right',
    category: 'settings',
    priority: 9
  },

  // =========================================================================
  // SECURITY TIPS (3 tips)
  // =========================================================================
  {
    id: 'detection-rules',
    targetSelector: '.detection-rules-panel',
    title: 'Detection Rules',
    description: 'Configure security rules to detect impossible travel, concurrent streams, and suspicious activity.',
    trigger: 'visible',
    position: 'right',
    category: 'settings',
    priority: 10
  },
  {
    id: 'security-alerts',
    targetSelector: '.security-alerts-panel',
    title: 'Security Alerts',
    description: 'View and manage security alerts triggered by detection rules.',
    trigger: 'visible',
    position: 'right',
    category: 'settings',
    priority: 11
  },
  {
    id: 'api-tokens',
    targetSelector: '.api-tokens-panel',
    title: 'API Access Tokens',
    description: 'Create personal access tokens for API integration with external tools and automation.',
    trigger: 'visible',
    position: 'right',
    category: 'settings',
    priority: 12
  },

  // =========================================================================
  // SERVER MANAGEMENT TIPS (4 tips)
  // =========================================================================
  {
    id: 'sync-status',
    targetSelector: '.sync-status-indicator',
    title: 'Sync Status',
    description: 'Shows the current sync status with your media servers. Green indicates healthy connection.',
    trigger: 'hover',
    position: 'bottom',
    category: 'general',
    priority: 6
  },
  {
    id: 'trigger-sync',
    targetSelector: '.btn-trigger-sync',
    title: 'Manual Sync',
    description: 'Trigger an immediate sync to fetch the latest data from your media servers.',
    trigger: 'hover',
    position: 'bottom',
    category: 'general',
    priority: 7
  },
  {
    id: 'server-config',
    targetSelector: '.server-config-panel',
    title: 'Server Configuration',
    description: 'Configure connection settings for Tautulli, Plex, Jellyfin, and Emby servers.',
    trigger: 'visible',
    position: 'right',
    category: 'settings',
    priority: 13
  },
  {
    id: 'connection-health',
    targetSelector: '.connection-health-panel',
    title: 'Connection Health',
    description: 'Monitor connection status, latency, and error rates for all configured servers.',
    trigger: 'visible',
    position: 'right',
    category: 'general',
    priority: 8
  },

  // =========================================================================
  // NEWSLETTER TIPS (4 tips)
  // =========================================================================
  {
    id: 'newsletter-templates',
    targetSelector: '.newsletter-templates-tab',
    title: 'Newsletter Templates',
    description: 'Create and customize email templates with dynamic content blocks and personalization.',
    trigger: 'hover',
    position: 'bottom',
    category: 'general',
    priority: 9
  },
  {
    id: 'newsletter-schedules',
    targetSelector: '.newsletter-schedules-tab',
    title: 'Newsletter Schedules',
    description: 'Set up automated newsletter delivery on daily, weekly, or custom schedules.',
    trigger: 'hover',
    position: 'bottom',
    category: 'general',
    priority: 10
  },
  {
    id: 'newsletter-preview',
    targetSelector: '.newsletter-preview-btn',
    title: 'Preview Newsletter',
    description: 'Preview how your newsletter will look before sending to recipients.',
    trigger: 'hover',
    position: 'left',
    category: 'general',
    priority: 11
  },
  {
    id: 'newsletter-channels',
    targetSelector: '.delivery-channels-section',
    title: 'Delivery Channels',
    description: 'Configure email, Discord, Slack, Telegram, or webhook delivery for newsletters.',
    trigger: 'visible',
    position: 'right',
    category: 'general',
    priority: 12
  },

  // =========================================================================
  // RECOMMENDATIONS TIPS (2 tips)
  // =========================================================================
  {
    id: 'recommendations-panel',
    targetSelector: '.recommendations-panel',
    title: 'Content Recommendations',
    description: 'View personalized content recommendations based on viewing history and preferences.',
    trigger: 'visible',
    position: 'left',
    category: 'general',
    priority: 13
  },
  {
    id: 'recommendation-settings',
    targetSelector: '.recommendation-settings',
    title: 'Recommendation Settings',
    description: 'Configure recommendation algorithms and adjust personalization preferences.',
    trigger: 'visible',
    position: 'right',
    category: 'settings',
    priority: 14
  },

  // =========================================================================
  // SETTINGS TIPS (4 tips)
  // =========================================================================
  {
    id: 'theme-toggle',
    targetSelector: '#theme-toggle',
    title: 'Theme Preference',
    description: 'Toggle between dark, light, and high-contrast themes for your viewing comfort.',
    trigger: 'hover',
    position: 'bottom',
    category: 'settings',
    priority: 1
  },
  {
    id: 'refresh-data',
    targetSelector: '#btn-refresh',
    title: 'Refresh Data',
    description: 'Click to manually refresh all dashboard data. Data also auto-refreshes periodically.',
    trigger: 'hover',
    position: 'bottom',
    category: 'settings',
    priority: 2
  },
  {
    id: 'keyboard-shortcuts',
    targetSelector: '#keyboard-shortcuts-btn',
    title: 'Keyboard Shortcuts',
    description: 'Press ? anytime to view all available keyboard shortcuts for faster navigation.',
    trigger: 'hover',
    position: 'left',
    category: 'settings',
    priority: 3
  },
  {
    id: 'notification-center',
    targetSelector: '#notification-center-btn',
    title: 'Notification History',
    description: 'View your notification history and manage alert preferences.',
    trigger: 'hover',
    position: 'left',
    category: 'settings',
    priority: 4
  },

  // =========================================================================
  // SIDEBAR & UI TIPS (2 tips)
  // =========================================================================
  {
    id: 'sidebar-collapse',
    targetSelector: '#sidebar-collapse-btn',
    title: 'Collapse Sidebar',
    description: 'Collapse the sidebar to maximize your map and chart viewing area.',
    trigger: 'hover',
    position: 'right',
    category: 'navigation',
    priority: 9
  },
  {
    id: 'breadcrumb-navigation',
    targetSelector: '.breadcrumb-navigation',
    title: 'Breadcrumb Navigation',
    description: 'Track your location and quickly navigate back to previous views.',
    trigger: 'hover',
    position: 'bottom',
    category: 'navigation',
    priority: 10
  }
];

export class ProgressiveOnboardingManager {
  private activeTooltip: HTMLElement | null = null;
  private discoveredTips: Set<string> = new Set();
  private abortController: AbortController | null = null;
  private observerMap: Map<string, IntersectionObserver> = new Map();
  private hoverTimers: Map<string, ReturnType<typeof setTimeout>> = new Map();
  private isEnabled: boolean = true;
  private isPaused: boolean = false; // Paused during wizard/tour
  private isDestroyed: boolean = false;

  constructor() {
    this.loadDiscoveredState();
  }

  /**
   * Initialize the progressive onboarding system
   */
  init(): void {
    if (this.isDestroyed) {
      logger.warn('[ProgressiveOnboarding] Cannot init - already destroyed');
      return;
    }

    // Check if progressive tips are enabled (can be disabled in settings)
    const disabled = SafeStorage.getItem('progressive_tips_disabled');
    if (disabled === 'true') {
      this.isEnabled = false;
      return;
    }

    // Check if wizard or tour is active - pause if so
    this.checkForBlockingModals();

    this.createTooltipContainer();
    this.setupEventListeners();
    this.setupModalObserver();
    this.attachTipTriggers();

    logger.debug('[ProgressiveOnboarding] Initialized with', this.discoveredTips.size, 'tips already discovered');
  }

  /**
   * Check for blocking modals (wizard/tour) and pause if active
   */
  private checkForBlockingModals(): void {
    const wizardModal = document.getElementById('setup-wizard-modal');
    const onboardingModal = document.getElementById('onboarding-modal');
    const tourTooltip = document.querySelector('.onboarding-tooltip');

    const isBlocked = Boolean(
      wizardModal?.style.display === 'flex' ||
      onboardingModal?.style.display === 'flex' ||
      (tourTooltip && (tourTooltip as HTMLElement).style.display === 'block')
    );

    this.isPaused = isBlocked;
  }

  /**
   * Setup mutation observer to watch for modal changes
   */
  private setupModalObserver(): void {
    // Watch for wizard/tour modals appearing or disappearing
    const observer = new MutationObserver(() => {
      this.checkForBlockingModals();
    });

    observer.observe(document.body, {
      childList: true,
      subtree: true,
      attributes: true,
      attributeFilter: ['style', 'class']
    });
  }

  /**
   * Load discovered tips from SafeStorage
   */
  private loadDiscoveredState(): void {
    const discovered = SafeStorage.getJSON<string[]>('progressive_tips_discovered', []);
    this.discoveredTips = new Set(discovered);
  }

  /**
   * Save discovered tips to SafeStorage
   */
  private saveDiscoveredState(): void {
    SafeStorage.setJSON('progressive_tips_discovered', [...this.discoveredTips]);
  }

  /**
   * Create the tooltip container element
   */
  private createTooltipContainer(): void {
    // Reuse existing tooltip if present
    let container = document.getElementById('progressive-tip-container');
    if (!container) {
      container = document.createElement('div');
      container.id = 'progressive-tip-container';
      container.className = 'progressive-tip-container';
      container.setAttribute('role', 'tooltip');
      container.setAttribute('aria-live', 'polite');
      container.style.display = 'none';
      document.body.appendChild(container);
    }
  }

  /**
   * Setup global event listeners
   */
  private setupEventListeners(): void {
    this.abortController = new AbortController();
    const signal = this.abortController.signal;

    // Escape key to dismiss active tooltip
    document.addEventListener('keydown', (e) => {
      if (e.key === 'Escape' && this.activeTooltip) {
        this.dismissActiveTip();
      }
    }, { signal });

    // Click outside to dismiss
    document.addEventListener('click', (e) => {
      if (this.activeTooltip && !this.activeTooltip.contains(e.target as Node)) {
        this.dismissActiveTip();
      }
    }, { signal });
  }

  /**
   * Attach triggers to all tip targets
   */
  private attachTipTriggers(): void {
    if (!this.abortController) return;
    const signal = this.abortController.signal;

    for (const tip of PROGRESSIVE_TIPS) {
      // Skip already discovered tips
      if (this.discoveredTips.has(tip.id)) continue;

      // Check prerequisites
      if (tip.prerequisites?.some(prereq => !this.discoveredTips.has(prereq))) continue;

      const target = document.querySelector(tip.targetSelector);
      if (!target) continue;

      switch (tip.trigger) {
        case 'hover':
          target.addEventListener('mouseenter', () => this.handleTrigger(tip), { signal });
          target.addEventListener('mouseleave', () => this.clearHoverTimer(tip.id), { signal });
          break;

        case 'click':
          target.addEventListener('click', () => this.handleTrigger(tip), { signal, once: true });
          break;

        case 'focus':
          target.addEventListener('focus', () => this.handleTrigger(tip), { signal });
          break;

        case 'visible':
          this.setupIntersectionObserver(tip, target);
          break;
      }
    }
  }

  /**
   * Setup intersection observer for visibility-triggered tips
   */
  private setupIntersectionObserver(tip: ProgressiveTip, target: Element): void {
    const observer = new IntersectionObserver((entries) => {
      for (const entry of entries) {
        if (entry.isIntersecting && !this.discoveredTips.has(tip.id)) {
          this.handleTrigger(tip);
          observer.disconnect();
          this.observerMap.delete(tip.id);
        }
      }
    }, { threshold: 0.5 });

    observer.observe(target);
    this.observerMap.set(tip.id, observer);
  }

  /**
   * Handle a tip trigger event
   */
  private handleTrigger(tip: ProgressiveTip): void {
    // Don't show if disabled, paused, or already discovered
    if (!this.isEnabled || this.isPaused || this.discoveredTips.has(tip.id)) return;

    // Don't show if destroyed
    if (this.isDestroyed) return;

    // Don't show if another tip is active
    if (this.activeTooltip) return;

    // Re-check for blocking modals (defensive)
    this.checkForBlockingModals();
    if (this.isPaused) return;

    const delay = tip.delay ?? 500; // Default 500ms delay

    // For hover, use a timer to prevent accidental triggers
    if (tip.trigger === 'hover') {
      const timer = setTimeout(() => {
        // Re-check paused state before showing
        if (!this.isPaused && !this.isDestroyed) {
          this.showTip(tip);
        }
        this.hoverTimers.delete(tip.id);
      }, delay);
      this.hoverTimers.set(tip.id, timer);
    } else {
      // For other triggers, show after delay
      setTimeout(() => {
        // Re-check paused state before showing
        if (!this.isPaused && !this.isDestroyed) {
          this.showTip(tip);
        }
      }, delay);
    }
  }

  /**
   * Clear hover timer for a tip
   */
  private clearHoverTimer(tipId: string): void {
    const timer = this.hoverTimers.get(tipId);
    if (timer) {
      clearTimeout(timer);
      this.hoverTimers.delete(tipId);
    }
  }

  /**
   * Show a tip tooltip
   */
  private showTip(tip: ProgressiveTip): void {
    const target = document.querySelector(tip.targetSelector);
    if (!target) return;

    const container = document.getElementById('progressive-tip-container');
    if (!container) return;

    // Build tooltip content
    container.innerHTML = `
      <div class="progressive-tip" data-tip-id="${tip.id}" role="tooltip">
        <div class="progressive-tip-header">
          <span class="progressive-tip-icon" aria-hidden="true">
            ${this.getCategoryIcon(tip.category)}
          </span>
          <h4 class="progressive-tip-title">${this.escapeHtml(tip.title)}</h4>
          <button class="progressive-tip-dismiss" aria-label="Dismiss tip" title="Dismiss">
            <svg width="14" height="14" viewBox="0 0 14 14" fill="none" xmlns="http://www.w3.org/2000/svg">
              <path d="M1 1L13 13M1 13L13 1" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
            </svg>
          </button>
        </div>
        <p class="progressive-tip-description">${this.escapeHtml(tip.description)}</p>
        <div class="progressive-tip-footer">
          <button class="progressive-tip-got-it">Got it</button>
          <button class="progressive-tip-disable-all">Don't show tips</button>
        </div>
        <div class="progressive-tip-arrow" data-position="${tip.position}"></div>
      </div>
    `;

    // Position the tooltip
    this.positionTooltip(container, target as HTMLElement, tip.position);

    // Show the tooltip
    container.style.display = 'block';
    this.activeTooltip = container;

    // Highlight target element
    target.classList.add('progressive-tip-target');

    // Setup tooltip event listeners
    const dismissBtn = container.querySelector('.progressive-tip-dismiss');
    const gotItBtn = container.querySelector('.progressive-tip-got-it');
    const disableAllBtn = container.querySelector('.progressive-tip-disable-all');

    dismissBtn?.addEventListener('click', () => this.dismissTip(tip.id));
    gotItBtn?.addEventListener('click', () => this.dismissTip(tip.id));
    disableAllBtn?.addEventListener('click', () => this.disableAllTips());

    // Focus management for accessibility
    gotItBtn?.setAttribute('tabindex', '0');
    (gotItBtn as HTMLElement)?.focus();

    logger.debug('[ProgressiveOnboarding] Showing tip:', tip.id);
  }

  /**
   * Position the tooltip relative to target
   */
  private positionTooltip(tooltip: HTMLElement, target: HTMLElement, position: ProgressiveTip['position']): void {
    const targetRect = target.getBoundingClientRect();
    const padding = 12;

    // Reset position for measurement
    tooltip.style.top = '0';
    tooltip.style.left = '0';
    tooltip.style.display = 'block';

    const tooltipRect = tooltip.getBoundingClientRect();

    let top = 0;
    let left = 0;

    switch (position) {
      case 'top':
        top = targetRect.top - tooltipRect.height - padding + window.scrollY;
        left = targetRect.left + (targetRect.width - tooltipRect.width) / 2 + window.scrollX;
        break;
      case 'bottom':
        top = targetRect.bottom + padding + window.scrollY;
        left = targetRect.left + (targetRect.width - tooltipRect.width) / 2 + window.scrollX;
        break;
      case 'left':
        top = targetRect.top + (targetRect.height - tooltipRect.height) / 2 + window.scrollY;
        left = targetRect.left - tooltipRect.width - padding + window.scrollX;
        break;
      case 'right':
        top = targetRect.top + (targetRect.height - tooltipRect.height) / 2 + window.scrollY;
        left = targetRect.right + padding + window.scrollX;
        break;
    }

    // Ensure tooltip stays within viewport
    const maxLeft = window.innerWidth - tooltipRect.width - padding;
    const minLeft = padding;

    left = Math.max(minLeft, Math.min(left, maxLeft));
    top = Math.max(padding, top);

    tooltip.style.top = `${top}px`;
    tooltip.style.left = `${left}px`;
  }

  /**
   * Get category icon
   */
  private getCategoryIcon(category: ProgressiveTip['category']): string {
    const icons: Record<ProgressiveTip['category'], string> = {
      map: '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 10c0 7-9 13-9 13s-9-6-9-13a9 9 0 0 1 18 0z"/><circle cx="12" cy="10" r="3"/></svg>',
      analytics: '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M18 20V10M12 20V4M6 20v-6"/></svg>',
      filters: '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="22 3 2 3 10 12.46 10 19 14 21 14 12.46 22 3"/></svg>',
      navigation: '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="3 11 22 2 13 21 11 13 3 11"/></svg>',
      settings: '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"/></svg>',
      general: '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><path d="M12 16v-4M12 8h.01"/></svg>'
    };
    return icons[category];
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
   * Dismiss a specific tip
   */
  private dismissTip(tipId: string): void {
    this.discoveredTips.add(tipId);
    this.saveDiscoveredState();
    this.dismissActiveTip();

    logger.debug('[ProgressiveOnboarding] Dismissed tip:', tipId);

    // Re-attach triggers for newly available tips (those with met prerequisites)
    this.attachTipTriggers();
  }

  /**
   * Dismiss the currently active tip
   */
  private dismissActiveTip(): void {
    if (this.activeTooltip) {
      this.activeTooltip.style.display = 'none';
      this.activeTooltip = null;
    }

    // Remove highlight from all targets
    document.querySelectorAll('.progressive-tip-target').forEach(el => {
      el.classList.remove('progressive-tip-target');
    });
  }

  /**
   * Disable all progressive tips
   */
  private disableAllTips(): void {
    this.isEnabled = false;
    SafeStorage.setItem('progressive_tips_disabled', 'true');
    this.dismissActiveTip();

    logger.debug('[ProgressiveOnboarding] All tips disabled by user');
  }

  /**
   * Enable progressive tips (can be called from settings)
   */
  enable(): void {
    if (this.isDestroyed) return;

    this.isEnabled = true;
    SafeStorage.removeItem('progressive_tips_disabled');
    this.attachTipTriggers();

    logger.debug('[ProgressiveOnboarding] Tips enabled');
  }

  /**
   * Reset all discovered tips (for testing or user preference)
   */
  reset(): void {
    if (this.isDestroyed) return;

    this.discoveredTips.clear();
    SafeStorage.removeItem('progressive_tips_discovered');
    SafeStorage.removeItem('progressive_tips_disabled');
    this.isEnabled = true;
    this.isPaused = false;
    this.attachTipTriggers();

    logger.debug('[ProgressiveOnboarding] All tips reset');
  }

  /**
   * Pause tips (called when wizard/tour starts)
   */
  pause(): void {
    this.isPaused = true;
    this.dismissActiveTip();
  }

  /**
   * Resume tips (called when wizard/tour ends)
   */
  resume(): void {
    this.isPaused = false;
  }

  /**
   * Get discovery progress
   */
  getProgress(): { discovered: number; total: number; percentage: number } {
    const total = PROGRESSIVE_TIPS.length;
    const discovered = this.discoveredTips.size;
    return {
      discovered,
      total,
      percentage: Math.round((discovered / total) * 100)
    };
  }

  /**
   * Check if tips are enabled
   */
  isActive(): boolean {
    return this.isEnabled;
  }

  /**
   * Cleanup resources
   */
  destroy(): void {
    // Mark as destroyed first to prevent any concurrent operations
    this.isDestroyed = true;

    // Clear all hover timers
    for (const timer of this.hoverTimers.values()) {
      clearTimeout(timer);
    }
    this.hoverTimers.clear();

    // Disconnect all observers
    for (const observer of this.observerMap.values()) {
      observer.disconnect();
    }
    this.observerMap.clear();

    // Abort event listeners
    if (this.abortController) {
      this.abortController.abort();
      this.abortController = null;
    }

    // Remove tooltip container
    const container = document.getElementById('progressive-tip-container');
    if (container) {
      container.remove();
    }

    // Remove highlights
    document.querySelectorAll('.progressive-tip-target').forEach(el => {
      el.classList.remove('progressive-tip-target');
    });

    this.activeTooltip = null;
    this.isEnabled = false;
    this.isPaused = false;
  }
}
