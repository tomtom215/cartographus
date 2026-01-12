// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * ChartTooltipManager - Adds informational tooltips to charts
 *
 * Features:
 * - Adds info icons next to chart titles
 * - Shows metric explanations on hover/focus
 * - Keyboard accessible (Tab to focus, visible tooltip)
 * - Responsive positioning
 */

import { escapeHtml } from '../lib/sanitize';
import { createLogger } from '../lib/logger';

const logger = createLogger('ChartTooltipManager');

interface ChartExplanation {
  title: string;
  metrics: string[];
  tips?: string;
}

// Detailed explanations for each chart, keyed by chart ID
const CHART_EXPLANATIONS: Record<string, ChartExplanation> = {
  // Overview Charts
  'chart-trends': {
    title: 'Playback Trends',
    metrics: [
      'Y-axis: Number of playback sessions',
      'X-axis: Date/time period',
      'Line represents total playbacks over time'
    ],
    tips: 'Use the zoom slider to focus on specific time periods. Hover over points for exact values.'
  },
  'chart-countries': {
    title: 'Top Countries',
    metrics: [
      'Bar length: Total playback count from each country',
      'Colors: Consistent palette for each country',
      'Percentage shows share of total playbacks'
    ],
    tips: 'Based on IP geolocation. VPN users may show incorrect countries.'
  },
  'chart-cities': {
    title: 'Top Cities',
    metrics: [
      'Bar length: Total playback count from each city',
      'Geographic aggregation at city level',
      'Percentage shows share of total playbacks'
    ],
    tips: 'City-level accuracy depends on IP geolocation provider.'
  },
  'chart-media': {
    title: 'Media Distribution',
    metrics: [
      'Slice size: Proportion of each media type',
      'Movies, TV, Music, Live TV categories',
      'Percentage labels show exact distribution'
    ]
  },
  'chart-users': {
    title: 'Top Users',
    metrics: [
      'Bar length: Total playbacks per user',
      'Includes all media types',
      'User-friendly names from Plex'
    ]
  },
  'chart-heatmap': {
    title: 'Viewing Heatmap',
    metrics: [
      'X-axis: Hour of day (0-23)',
      'Y-axis: Day of week',
      'Color intensity: Playback volume (darker = more)'
    ],
    tips: 'Identifies peak viewing times. Useful for maintenance scheduling.'
  },

  // Content Charts
  'chart-platforms': {
    title: 'Platform Distribution',
    metrics: [
      'Slice size: Proportion of each platform',
      'Platforms: Web, iOS, Android, Roku, etc.',
      'Based on player device type'
    ]
  },
  'chart-players': {
    title: 'Player Distribution',
    metrics: [
      'Slice size: Proportion of each player app',
      'Player names from Plex client apps',
      'E.g., Plex for Apple TV, Plex Web'
    ]
  },
  'chart-libraries': {
    title: 'Library Distribution',
    metrics: [
      'Slice size: Playbacks per library',
      'All Plex libraries included',
      'Useful for content usage analysis'
    ]
  },
  'chart-ratings': {
    title: 'Content Ratings',
    metrics: [
      'Bar length: Playbacks per rating',
      'Ratings: G, PG, PG-13, R, etc.',
      'Based on content metadata'
    ]
  },
  'chart-years': {
    title: 'Release Years',
    metrics: [
      'Bar length: Playbacks per release year',
      'Top 10 years by popularity',
      'Based on content metadata'
    ]
  },
  'chart-popular-movies': {
    title: 'Top Movies',
    metrics: [
      'Bar length: Total playback count',
      'Movie titles from Plex metadata',
      'Top 10 by popularity'
    ]
  },
  'chart-popular-shows': {
    title: 'Top TV Shows',
    metrics: [
      'Bar length: Total playback count',
      'Aggregated across all episodes',
      'Top 10 by popularity'
    ]
  },
  'chart-popular-episodes': {
    title: 'Top Episodes',
    metrics: [
      'Bar length: Total playback count',
      'Individual episode titles',
      'Top 10 by popularity'
    ]
  },

  // Performance Charts
  'chart-transcode': {
    title: 'Transcode vs Direct Play',
    metrics: [
      'Direct Play: No server processing required',
      'Direct Stream: Container changed, codecs unchanged',
      'Transcode: Video/audio re-encoded'
    ],
    tips: 'High direct play % indicates well-optimized library for your devices.'
  },
  'chart-resolution': {
    title: 'Video Resolution',
    metrics: [
      'Bar length: Playbacks at each resolution',
      'Resolutions: 4K, 1080p, 720p, SD',
      'Original file resolution (not transcoded)'
    ]
  },
  'chart-codec': {
    title: 'Codec Combinations',
    metrics: [
      'Video codec + Audio codec pairs',
      'E.g., H.264 + AAC, HEVC + EAC3',
      'Affects transcode requirements'
    ]
  },
  'chart-bandwidth-trends': {
    title: 'Bandwidth Trends',
    metrics: [
      'Y-axis: Bandwidth in Mbps',
      'X-axis: Time period',
      'Shows LAN vs WAN bandwidth usage'
    ],
    tips: 'High WAN bandwidth may indicate remote streaming costs.'
  },
  'chart-bandwidth-transcode': {
    title: 'Bandwidth by Transcode',
    metrics: [
      'Average bandwidth per transcode type',
      'Direct play typically uses more bandwidth',
      'Transcode reduces bandwidth at quality cost'
    ]
  },
  'chart-bandwidth-users': {
    title: 'Top Bandwidth Users',
    metrics: [
      'Total bandwidth consumed per user',
      'Includes all sessions',
      'Useful for capacity planning'
    ]
  },
  'chart-bitrate-distribution': {
    title: 'Bitrate Distribution',
    metrics: [
      'X-axis: Bitrate in Mbps',
      'Y-axis: Number of sessions',
      'Shows common bitrate ranges'
    ],
    tips: 'Bi-modal distribution may indicate transcoding activity.'
  },
  'chart-hardware-transcode': {
    title: 'Hardware Transcode',
    metrics: [
      'GPU utilization: NVENC, Quick Sync, VCE',
      'Software vs hardware transcode ratio',
      'Efficiency metrics'
    ],
    tips: 'Hardware transcode is faster and uses less CPU.'
  },
  'chart-abandonment': {
    title: 'Abandonment Analysis',
    metrics: [
      'Y-axis: Drop-off rate percentage',
      'X-axis: Time into content',
      'Shows when viewers stop watching'
    ],
    tips: 'High early abandonment may indicate buffering or quality issues.'
  },

  // User Charts
  'chart-completion': {
    title: 'Completion Rates',
    metrics: [
      'Percentage of content watched',
      'Grouped by media type',
      '100% = fully watched'
    ]
  },
  'chart-binge-summary': {
    title: 'Binge Watching Summary',
    metrics: [
      'Binge sessions: 3+ episodes in a row',
      'Average episodes per session',
      'Total binge time'
    ],
    tips: 'High binge rate indicates engaging content.'
  },
  'chart-binge-shows': {
    title: 'Top Binge-Watched Shows',
    metrics: [
      'Bar length: Number of binge sessions',
      'Shows with 3+ consecutive episodes',
      'Ranked by binge frequency'
    ]
  },
  'chart-binge-users': {
    title: 'Top Binge Watchers',
    metrics: [
      'Bar length: Total binge sessions',
      'Users who watch 3+ episodes at once',
      'Ranked by binge frequency'
    ]
  },
  'chart-watch-parties-summary': {
    title: 'Watch Parties Summary',
    metrics: [
      'Total watch party events',
      'Average participants per party',
      'Total social viewing time'
    ],
    tips: 'Watch parties: multiple users watching same content simultaneously.'
  },
  'chart-engagement-summary': {
    title: 'Engagement Summary',
    metrics: [
      'Average session length',
      'Pause frequency',
      'Completion rate'
    ]
  },
  'chart-engagement-hours': {
    title: 'Viewing by Hour',
    metrics: [
      'X-axis: Hour of day (0-23)',
      'Y-axis: Number of playbacks',
      'Identifies peak viewing times'
    ]
  },

  // Tautulli Charts
  'chart-user-logins': {
    title: 'User Login History',
    metrics: [
      'Stacked bars: successful vs failed logins',
      'X-axis: Date',
      'Success rate percentage'
    ],
    tips: 'High failure rate may indicate security concerns.'
  },
  'chart-user-ips': {
    title: 'User IP Addresses',
    metrics: [
      'Bar length: Playback count per IP',
      'Platform and player info',
      'Last played timestamp'
    ]
  },
  'chart-synced-items': {
    title: 'Synced Items',
    metrics: [
      'Complete: Items fully synced',
      'Downloading: Items in progress',
      'Grouped by user'
    ],
    tips: 'Monitor offline sync activity for storage planning.'
  },
  'chart-stream-type-platform': {
    title: 'Stream Type by Platform',
    metrics: [
      'Direct Play, Direct Stream, Transcode',
      'Grouped by platform type',
      'Total plays per platform'
    ],
    tips: 'Helps identify platforms needing transcoding.'
  },
  'chart-activity-history': {
    title: 'Activity History',
    metrics: [
      'Y-axis: Number of playbacks',
      'X-axis: Time (hourly)',
      'Last 48 hours of activity'
    ]
  },
  'chart-plays-per-month': {
    title: 'Plays Per Month',
    metrics: [
      'Stacked bars: Movies, TV, Music, Live TV',
      'X-axis: Month',
      'Total plays per month'
    ]
  },
  'chart-concurrent-streams-type': {
    title: 'Concurrent Streams',
    metrics: [
      'Slice size: Proportion by stream type',
      'Peak and average concurrent values',
      'Direct play vs transcode split'
    ]
  }
};

export class ChartTooltipManager {
  private tooltipElement: HTMLElement | null = null;
  private activeTooltip: string | null = null;

  // Event handler references for cleanup
  private documentKeydownHandler: ((e: KeyboardEvent) => void) | null = null;
  private documentClickHandler: ((e: MouseEvent) => void) | null = null;
  private documentFocusinHandler: ((e: FocusEvent) => void) | null = null;
  private documentFocusoutHandler: ((e: FocusEvent) => void) | null = null;
  private documentMouseoverHandler: ((e: MouseEvent) => void) | null = null;
  private documentMouseoutHandler: ((e: MouseEvent) => void) | null = null;

  constructor() {
    this.createTooltipElement();
    this.setupEventListeners();
  }

  /**
   * Initialize the tooltip manager
   */
  init(): void {
    this.addInfoIconsToCharts();
    logger.info('ChartTooltipManager initialized');
  }

  /**
   * Create the shared tooltip element
   */
  private createTooltipElement(): void {
    const tooltip = document.createElement('div');
    tooltip.id = 'chart-info-tooltip';
    tooltip.className = 'chart-info-tooltip';
    tooltip.setAttribute('role', 'tooltip');
    tooltip.setAttribute('aria-hidden', 'true');
    tooltip.innerHTML = `
      <div class="chart-tooltip-header">
        <span class="chart-tooltip-title"></span>
      </div>
      <div class="chart-tooltip-metrics">
        <ul class="chart-tooltip-list"></ul>
      </div>
      <div class="chart-tooltip-tips"></div>
    `;
    document.body.appendChild(tooltip);
    this.tooltipElement = tooltip;
  }

  /**
   * Set up global event listeners with event delegation
   */
  private setupEventListeners(): void {
    // Close tooltip on escape key
    this.documentKeydownHandler = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && this.activeTooltip) {
        this.hideTooltip();
      }
    };
    document.addEventListener('keydown', this.documentKeydownHandler);

    // Handle click on info icons and outside clicks (delegation)
    this.documentClickHandler = (e: MouseEvent) => {
      const target = e.target as HTMLElement;
      const infoIcon = target.closest('.chart-info-icon') as HTMLElement;

      if (infoIcon) {
        // Handle info icon click
        e.stopPropagation();
        const chartId = infoIcon.getAttribute('data-chart-id');
        if (chartId) {
          if (this.activeTooltip === chartId) {
            this.hideTooltip();
          } else {
            this.showTooltip(chartId, infoIcon);
          }
        }
      } else if (!target.closest('.chart-info-tooltip')) {
        // Close tooltip when clicking outside
        this.hideTooltip();
      }
    };
    document.addEventListener('click', this.documentClickHandler);

    // Handle mouseenter/mouseleave via delegation
    this.documentMouseoverHandler = (e: MouseEvent) => {
      const target = e.target as HTMLElement;
      const infoIcon = target.closest('.chart-info-icon') as HTMLElement;
      if (infoIcon) {
        const chartId = infoIcon.getAttribute('data-chart-id');
        if (chartId) {
          this.showTooltip(chartId, infoIcon);
        }
      }
    };
    document.addEventListener('mouseover', this.documentMouseoverHandler);

    this.documentMouseoutHandler = (e: MouseEvent) => {
      const target = e.target as HTMLElement;
      const infoIcon = target.closest('.chart-info-icon') as HTMLElement;
      if (infoIcon) {
        this.hideTooltipDelayed();
      }
    };
    document.addEventListener('mouseout', this.documentMouseoutHandler);

    // Handle focus/blur via delegation
    this.documentFocusinHandler = (e: FocusEvent) => {
      const target = e.target as HTMLElement;
      if (target.classList.contains('chart-info-icon')) {
        const chartId = target.getAttribute('data-chart-id');
        if (chartId) {
          this.showTooltip(chartId, target);
        }
      }
    };
    document.addEventListener('focusin', this.documentFocusinHandler);

    this.documentFocusoutHandler = (e: FocusEvent) => {
      const target = e.target as HTMLElement;
      if (target.classList.contains('chart-info-icon')) {
        this.hideTooltipDelayed();
      }
    };
    document.addEventListener('focusout', this.documentFocusoutHandler);
  }

  /**
   * Add info icons to all registered charts
   */
  addInfoIconsToCharts(): void {
    Object.keys(CHART_EXPLANATIONS).forEach(chartId => {
      this.addInfoIconToChart(chartId);
    });
  }

  /**
   * Add info icon to a specific chart
   */
  private addInfoIconToChart(chartId: string): void {
    const chartElement = document.getElementById(chartId);
    if (!chartElement) return;

    // Find the chart wrapper or create one
    const wrapper = chartElement.closest('.chart-wrapper');
    if (!wrapper) return;

    // Check if info icon already exists
    if (wrapper.querySelector('.chart-info-icon')) return;

    // Find the header or title area
    let header = wrapper.querySelector('.chart-header');
    if (!header) {
      // Create header if it doesn't exist
      const title = wrapper.querySelector('h4, h3, .chart-title');
      if (title) {
        header = document.createElement('div');
        header.className = 'chart-header-with-info';
        title.parentNode?.insertBefore(header, title);
        header.appendChild(title);
      } else {
        // Create header at the start of wrapper
        header = document.createElement('div');
        header.className = 'chart-header-with-info';
        wrapper.insertBefore(header, wrapper.firstChild);
      }
    }

    // Create the info icon button
    // Note: Event handlers are managed via delegation in setupEventListeners()
    const infoIcon = document.createElement('button');
    infoIcon.className = 'chart-info-icon';
    infoIcon.setAttribute('type', 'button');
    infoIcon.setAttribute('aria-label', `Information about this chart`);
    infoIcon.setAttribute('aria-describedby', `tooltip-${chartId}`);
    infoIcon.setAttribute('data-chart-id', chartId);
    infoIcon.innerHTML = `
      <svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor" aria-hidden="true">
        <path d="M8 1a7 7 0 1 0 0 14A7 7 0 0 0 8 1zM6.5 7.5a.5.5 0 0 1 .5-.5h1a.5.5 0 0 1 .5.5v4a.5.5 0 0 1-.5.5H7a.5.5 0 0 1-.5-.5v-4zM8 4.5a1 1 0 1 1 0 2 1 1 0 0 1 0-2z"/>
      </svg>
    `;

    header.appendChild(infoIcon);
  }

  /**
   * Show tooltip for a chart
   */
  private showTooltip(chartId: string, anchor: HTMLElement): void {
    const explanation = CHART_EXPLANATIONS[chartId];
    if (!explanation || !this.tooltipElement) return;

    this.activeTooltip = chartId;

    // Update tooltip content
    const titleEl = this.tooltipElement.querySelector('.chart-tooltip-title');
    const listEl = this.tooltipElement.querySelector('.chart-tooltip-list');
    const tipsEl = this.tooltipElement.querySelector('.chart-tooltip-tips');

    if (titleEl) titleEl.textContent = explanation.title;
    if (listEl) {
      // Defense in depth: escape content even though it's from static config
      listEl.innerHTML = explanation.metrics
        .map(m => `<li>${escapeHtml(m)}</li>`)
        .join('');
    }
    if (tipsEl) {
      const tipsDiv = tipsEl as HTMLElement;
      if (explanation.tips) {
        // Defense in depth: escape content even though it's from static config
        tipsDiv.innerHTML = `<strong>Tip:</strong> ${escapeHtml(explanation.tips)}`;
        tipsDiv.style.display = 'block';
      } else {
        tipsDiv.style.display = 'none';
      }
    }

    // Position tooltip
    this.positionTooltip(anchor);

    // Show tooltip
    this.tooltipElement.classList.add('visible');
    this.tooltipElement.setAttribute('aria-hidden', 'false');
  }

  /**
   * Position tooltip relative to anchor element
   */
  private positionTooltip(anchor: HTMLElement): void {
    if (!this.tooltipElement) return;

    const anchorRect = anchor.getBoundingClientRect();
    const tooltipRect = this.tooltipElement.getBoundingClientRect();
    const padding = 8;

    let top = anchorRect.bottom + padding;
    let left = anchorRect.left + (anchorRect.width / 2) - (tooltipRect.width / 2);

    // Adjust if tooltip goes off screen
    const maxRight = window.innerWidth - padding;
    const maxBottom = window.innerHeight - padding;

    if (left < padding) left = padding;
    if (left + tooltipRect.width > maxRight) {
      left = maxRight - tooltipRect.width;
    }

    // If no room below, show above
    if (top + tooltipRect.height > maxBottom) {
      top = anchorRect.top - tooltipRect.height - padding;
    }

    this.tooltipElement.style.top = `${top}px`;
    this.tooltipElement.style.left = `${left}px`;
  }

  /**
   * Hide tooltip with delay (for mouse interactions)
   */
  private hideTooltipDelayed(): void {
    setTimeout(() => {
      // Check if mouse is over tooltip
      if (!this.tooltipElement?.matches(':hover')) {
        this.hideTooltip();
      }
    }, 150);
  }

  /**
   * Hide tooltip immediately
   */
  private hideTooltip(): void {
    if (!this.tooltipElement) return;

    this.tooltipElement.classList.remove('visible');
    this.tooltipElement.setAttribute('aria-hidden', 'true');
    this.activeTooltip = null;
  }

  /**
   * Refresh info icons (call after charts are re-rendered)
   */
  refresh(): void {
    this.addInfoIconsToCharts();
  }

  /**
   * Remove all event listeners for cleanup
   */
  private removeEventListeners(): void {
    if (this.documentKeydownHandler) {
      document.removeEventListener('keydown', this.documentKeydownHandler);
      this.documentKeydownHandler = null;
    }

    if (this.documentClickHandler) {
      document.removeEventListener('click', this.documentClickHandler);
      this.documentClickHandler = null;
    }

    if (this.documentMouseoverHandler) {
      document.removeEventListener('mouseover', this.documentMouseoverHandler);
      this.documentMouseoverHandler = null;
    }

    if (this.documentMouseoutHandler) {
      document.removeEventListener('mouseout', this.documentMouseoutHandler);
      this.documentMouseoutHandler = null;
    }

    if (this.documentFocusinHandler) {
      document.removeEventListener('focusin', this.documentFocusinHandler);
      this.documentFocusinHandler = null;
    }

    if (this.documentFocusoutHandler) {
      document.removeEventListener('focusout', this.documentFocusoutHandler);
      this.documentFocusoutHandler = null;
    }
  }

  /**
   * Destroy the chart tooltip manager and clean up resources
   */
  destroy(): void {
    this.removeEventListeners();
    this.hideTooltip();

    // Remove tooltip element from DOM
    if (this.tooltipElement && this.tooltipElement.parentNode) {
      this.tooltipElement.parentNode.removeChild(this.tooltipElement);
      this.tooltipElement = null;
    }
  }
}

export default ChartTooltipManager;
