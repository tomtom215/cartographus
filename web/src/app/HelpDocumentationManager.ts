// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * HelpDocumentationManager - In-app help and documentation system
 *
 * Features:
 * - Comprehensive feature documentation
 * - FAQ section with common questions
 * - Quick reference guide
 * - Contextual help tips
 * - Links to external documentation
 * - Accessible modal with keyboard navigation
 */

import { createLogger } from '../lib/logger';

const logger = createLogger('HelpDocumentationManager');

interface HelpSection {
  id: string;
  title: string;
  icon: string;
  content: string;
}

interface FAQItem {
  question: string;
  answer: string;
}

const HELP_SECTIONS: HelpSection[] = [
  {
    id: 'maps',
    title: 'Maps & Visualization',
    icon: '&#x1F5FA;', // map emoji
    content: `
      <h4>Interactive Map Views</h4>
      <p>Cartographus offers multiple visualization modes:</p>
      <ul>
        <li><strong>Points Mode:</strong> Shows individual playback locations as markers</li>
        <li><strong>Clusters Mode:</strong> Groups nearby locations into clusters with counts</li>
        <li><strong>Heatmap Mode:</strong> Displays density visualization of playback activity</li>
        <li><strong>Hexagons Mode:</strong> H3 hexagon grid for precise geographic aggregation</li>
      </ul>

      <h4>3D Globe View</h4>
      <p>Switch to the 3D globe for a world-wide perspective of your playback data. Features include:</p>
      <ul>
        <li>Auto-rotate for presentation mode</li>
        <li>Click locations to view details</li>
        <li>Arc visualization showing server-to-user connections</li>
      </ul>

      <h4>Touch Gestures (Mobile/Tablet)</h4>
      <p>Use these touch gestures on mobile and tablet devices:</p>
      <div class="touch-gestures-grid">
        <div class="gesture-item">
          <span class="gesture-icon" aria-hidden="true">&#x1F446;</span>
          <div class="gesture-info">
            <strong>One Finger Drag</strong>
            <span>Pan the map or rotate the globe</span>
          </div>
        </div>
        <div class="gesture-item">
          <span class="gesture-icon" aria-hidden="true">&#x1F90F;</span>
          <div class="gesture-info">
            <strong>Pinch In/Out</strong>
            <span>Zoom in or out of the map</span>
          </div>
        </div>
        <div class="gesture-item">
          <span class="gesture-icon" aria-hidden="true">&#x270B;</span>
          <div class="gesture-info">
            <strong>Two Finger Drag</strong>
            <span>Tilt the map (3D perspective)</span>
          </div>
        </div>
        <div class="gesture-item">
          <span class="gesture-icon" aria-hidden="true">&#x1F504;</span>
          <div class="gesture-info">
            <strong>Two Finger Rotate</strong>
            <span>Rotate the map bearing</span>
          </div>
        </div>
        <div class="gesture-item">
          <span class="gesture-icon" aria-hidden="true">&#x1F448;</span>
          <div class="gesture-info">
            <strong>Double Tap</strong>
            <span>Zoom in at tap location</span>
          </div>
        </div>
        <div class="gesture-item">
          <span class="gesture-icon" aria-hidden="true">&#x1F446;</span>
          <div class="gesture-info">
            <strong>Single Tap</strong>
            <span>Select a location marker</span>
          </div>
        </div>
      </div>

      <h4>Arc Visualization</h4>
      <p>Enable connection arcs to see geographic paths from your server to viewers. Arc colors indicate playback volume.</p>
    `
  },
  {
    id: 'analytics',
    title: 'Analytics Dashboard',
    icon: '&#x1F4CA;', // chart emoji
    content: `
      <h4>Analytics Pages</h4>
      <p>Six comprehensive analytics pages with 47+ charts:</p>
      <ul>
        <li><strong>Overview:</strong> High-level metrics, trends, and automated insights</li>
        <li><strong>Content:</strong> Library analysis, media types, and viewing patterns</li>
        <li><strong>Users:</strong> User engagement, binge watching, and watch parties</li>
        <li><strong>Performance:</strong> Streaming quality, transcoding, and bandwidth</li>
        <li><strong>Geographic:</strong> Location-based analysis and regional trends</li>
        <li><strong>Advanced:</strong> Hardware utilization, abandonment rates, and more</li>
      </ul>

      <h4>Chart Interactions</h4>
      <ul>
        <li><strong>Hover:</strong> View detailed tooltips with metric explanations</li>
        <li><strong>Zoom:</strong> Use the slider or scroll to zoom time-series charts</li>
        <li><strong>Export:</strong> Download charts as PNG images</li>
        <li><strong>Keyboard:</strong> Navigate charts using arrow keys when focused</li>
      </ul>
    `
  },
  {
    id: 'filters',
    title: 'Filters & Data',
    icon: '&#x1F50D;', // magnifying glass
    content: `
      <h4>Data Filters</h4>
      <p>Refine your data using multiple filter dimensions:</p>
      <ul>
        <li><strong>Time Range:</strong> Quick presets or custom date ranges</li>
        <li><strong>Users:</strong> Filter by specific Plex users</li>
        <li><strong>Media Types:</strong> Movies, TV Shows, Music, Live TV</li>
        <li><strong>Libraries:</strong> Specific Plex libraries</li>
        <li><strong>Platforms:</strong> Device types (Web, Mobile, TV, etc.)</li>
      </ul>

      <h4>Filter Presets</h4>
      <p>Save frequently used filter combinations as presets for quick access. Click "Save" to create a new preset.</p>

      <h4>Quick Date Selection</h4>
      <p>Use the quick date buttons for common time ranges: Today, Yesterday, This Week, This Month.</p>

      <h4>Data Export</h4>
      <p>Export your data in multiple formats:</p>
      <ul>
        <li><strong>CSV:</strong> Playback data for spreadsheets</li>
        <li><strong>GeoJSON:</strong> Location data for GIS applications</li>
      </ul>
    `
  },
  {
    id: 'realtime',
    title: 'Real-Time Monitoring',
    icon: '&#x26A1;', // lightning bolt
    content: `
      <h4>Live Activity Dashboard</h4>
      <p>Monitor current streaming activity in real-time:</p>
      <ul>
        <li>Active streams with user and media details</li>
        <li>Bandwidth utilization (LAN vs WAN)</li>
        <li>Transcode status and progress</li>
        <li>Buffer health indicators</li>
      </ul>

      <h4>WebSocket Connection</h4>
      <p>The connection indicator in the header shows:</p>
      <ul>
        <li><strong>Green (Connected):</strong> Real-time updates active</li>
        <li><strong>Yellow (Reconnecting):</strong> Temporarily disconnected, attempting to reconnect</li>
        <li><strong>Red (Disconnected):</strong> Connection lost, click to retry</li>
      </ul>

      <h4>Timeline Playback</h4>
      <p>Use the timeline controls to replay historical playback activity with animated markers.</p>
    `
  },
  {
    id: 'settings',
    title: 'Settings & Themes',
    icon: '&#x2699;', // gear
    content: `
      <h4>Theme Options</h4>
      <p>Three display themes optimized for different preferences:</p>
      <ul>
        <li><strong>Dark Mode:</strong> Violet accent, reduced eye strain</li>
        <li><strong>Light Mode:</strong> Bright, high-contrast interface</li>
        <li><strong>High Contrast:</strong> Maximum contrast for accessibility</li>
      </ul>

      <h4>Colorblind Mode</h4>
      <p>Enable colorblind-safe palette for better visibility of charts and data visualizations.</p>

      <h4>Auto-Refresh</h4>
      <p>Toggle automatic data refresh every 60 seconds. Disable for manual control over data updates.</p>

      <h4>Backup & Restore</h4>
      <p>Backup your configuration and data through the Server dashboard. Supports full backup download and restore.</p>
    `
  }
];

const FAQ_ITEMS: FAQItem[] = [
  {
    question: 'Why are some locations showing in the wrong place?',
    answer: 'IP geolocation is approximate and typically accurate to city level. VPN usage, proxy servers, and mobile networks can affect location accuracy.'
  },
  {
    question: 'How often is data synchronized from Tautulli?',
    answer: 'Data synchronizes automatically based on your sync settings. Real-time activity updates occur every few seconds via WebSocket. Historical data syncs on a configurable schedule.'
  },
  {
    question: 'What do the different chart colors mean?',
    answer: 'Chart colors follow a consistent palette: Violet (primary), Blue (secondary), Emerald (success), Amber (warning), and Red (errors). In colorblind mode, colors are adjusted for better differentiation.'
  },
  {
    question: 'Can I export chart images?',
    answer: 'Yes! Each chart has an export button that downloads a PNG image. You can also export raw data in CSV or GeoJSON formats from the filter panel.'
  },
  {
    question: 'How do I filter data for a specific user?',
    answer: 'Use the User filter dropdown in the sidebar to select a specific user. All charts and map data will update to show only that user\'s activity.'
  },
  {
    question: 'What are the keyboard shortcuts?',
    answer: 'Press ? (question mark) to open the keyboard shortcuts modal with a complete list of available shortcuts for navigation, charts, and more.'
  },
  {
    question: 'Why is my WebSocket showing disconnected?',
    answer: 'WebSocket disconnection can occur due to network issues, server restarts, or extended idle time. The connection will attempt to reconnect automatically. Click the indicator to manually reconnect.'
  },
  {
    question: 'How do I interpret the trend indicators?',
    answer: 'Trend arrows show percentage change vs the previous period. Green up-arrow means increase, red down-arrow means decrease. Sparklines show the historical trend pattern.'
  },
  {
    question: 'What touch gestures work on mobile devices?',
    answer: 'On mobile and tablets: one-finger drag to pan, pinch to zoom, two-finger drag to tilt (3D), two-finger rotate to change bearing, double-tap to zoom in, and single tap to select markers. See the Maps section for a complete gesture guide.'
  }
];

export class HelpDocumentationManager {
  private modal: HTMLElement | null = null;
  private activeSection: string = 'maps';
  private isOpen: boolean = false;
  private previouslyFocusedElement: HTMLElement | null = null;

  // Store bound event handler references for cleanup
  private boundHandleKeyDown: (e: KeyboardEvent) => void;
  private boundHandleHelpKey: (e: KeyboardEvent) => void;
  private boundHandleHelpButtonClick: () => void;
  private boundHandleCloseClick: () => void;
  private boundHandleOverlayClick: (e: MouseEvent) => void;
  private helpButton: HTMLElement | null = null;
  private closeBtn: Element | null = null;
  private navItems: NodeListOf<Element> | null = null;
  private navItemClickHandlers: Map<Element, () => void> = new Map();

  constructor() {
    // Bind handlers for cleanup
    this.boundHandleKeyDown = this.handleKeyDown.bind(this);
    this.boundHandleHelpKey = this.handleHelpKeyShortcut.bind(this);
    this.boundHandleHelpButtonClick = () => this.open();
    this.boundHandleCloseClick = () => this.close();
    this.boundHandleOverlayClick = (e: MouseEvent) => {
      if (e.target === this.modal) {
        this.close();
      }
    };

    this.createModal();
    this.setupEventListeners();
  }

  /**
   * Initialize the help documentation system
   */
  init(): void {
    logger.debug('HelpDocumentationManager initialized');
  }

  /**
   * Create the help modal dynamically
   */
  private createModal(): void {
    const modal = document.createElement('div');
    modal.id = 'help-documentation-modal';
    modal.className = 'modal-overlay help-modal-overlay';
    modal.setAttribute('role', 'dialog');
    modal.setAttribute('aria-modal', 'true');
    modal.setAttribute('aria-labelledby', 'help-modal-title');
    modal.setAttribute('aria-hidden', 'true');
    modal.style.display = 'none';

    modal.innerHTML = `
      <div class="modal-content help-modal-content">
        <div class="modal-header help-modal-header">
          <h2 id="help-modal-title">Help & Documentation</h2>
          <button class="modal-close help-modal-close" aria-label="Close help modal">&times;</button>
        </div>
        <div class="help-modal-body">
          <nav class="help-nav" role="tablist" aria-label="Help sections">
            ${HELP_SECTIONS.map(section => `
              <button
                class="help-nav-item ${section.id === 'maps' ? 'active' : ''}"
                data-section="${section.id}"
                role="tab"
                aria-selected="${section.id === 'maps' ? 'true' : 'false'}"
                aria-controls="help-content-${section.id}"
                id="help-tab-${section.id}">
                <span class="help-nav-icon" aria-hidden="true">${section.icon}</span>
                <span class="help-nav-title">${section.title}</span>
              </button>
            `).join('')}
            <button
              class="help-nav-item"
              data-section="faq"
              role="tab"
              aria-selected="false"
              aria-controls="help-content-faq"
              id="help-tab-faq">
              <span class="help-nav-icon" aria-hidden="true">&#x2753;</span>
              <span class="help-nav-title">FAQ</span>
            </button>
          </nav>
          <div class="help-content-area">
            ${HELP_SECTIONS.map(section => `
              <section
                class="help-content-section ${section.id === 'maps' ? 'active' : ''}"
                id="help-content-${section.id}"
                role="tabpanel"
                aria-labelledby="help-tab-${section.id}"
                ${section.id !== 'maps' ? 'hidden' : ''}>
                <h3>${section.title}</h3>
                ${section.content}
              </section>
            `).join('')}
            <section
              class="help-content-section"
              id="help-content-faq"
              role="tabpanel"
              aria-labelledby="help-tab-faq"
              hidden>
              <h3>Frequently Asked Questions</h3>
              <div class="faq-list">
                ${FAQ_ITEMS.map((item, index) => `
                  <details class="faq-item" id="faq-item-${index}">
                    <summary class="faq-question">
                      <span class="faq-icon" aria-hidden="true">&#x25B6;</span>
                      ${item.question}
                    </summary>
                    <div class="faq-answer">
                      <p>${item.answer}</p>
                    </div>
                  </details>
                `).join('')}
              </div>
            </section>
          </div>
        </div>
        <div class="help-modal-footer">
          <div class="help-footer-links">
            <a href="https://github.com/tomtom215/cartographus" target="_blank" rel="noopener noreferrer" class="help-link">
              <span aria-hidden="true">&#x1F4BB;</span> GitHub Repository
            </a>
            <a href="https://github.com/tomtom215/cartographus/issues" target="_blank" rel="noopener noreferrer" class="help-link">
              <span aria-hidden="true">&#x1F41B;</span> Report Issue
            </a>
          </div>
          <p class="help-footer-hint">Press <kbd>?</kbd> for keyboard shortcuts</p>
        </div>
      </div>
    `;

    document.body.appendChild(modal);
    this.modal = modal;
  }

  /**
   * Set up event listeners
   */
  private setupEventListeners(): void {
    // Help button click
    this.helpButton = document.getElementById('help-docs-btn');
    if (this.helpButton) {
      this.helpButton.addEventListener('click', this.boundHandleHelpButtonClick);
    }

    // Close button
    this.closeBtn = this.modal?.querySelector('.help-modal-close') || null;
    if (this.closeBtn) {
      this.closeBtn.addEventListener('click', this.boundHandleCloseClick);
    }

    // Overlay click to close
    if (this.modal) {
      this.modal.addEventListener('click', this.boundHandleOverlayClick);
    }

    // Navigation tabs - store handlers for cleanup
    this.navItems = this.modal?.querySelectorAll('.help-nav-item') || null;
    this.navItems?.forEach(item => {
      const handler = () => {
        const section = item.getAttribute('data-section');
        if (section) {
          this.showSection(section);
        }
      };
      this.navItemClickHandlers.set(item, handler);
      item.addEventListener('click', handler);
    });

    // Keyboard navigation
    document.addEventListener('keydown', this.boundHandleKeyDown);

    // Global keyboard shortcut 'h' for help (when not in input)
    document.addEventListener('keydown', this.boundHandleHelpKey);
  }

  /**
   * Handle 'h' key shortcut to open help
   */
  private handleHelpKeyShortcut(e: KeyboardEvent): void {
    if (e.target instanceof HTMLInputElement ||
        e.target instanceof HTMLTextAreaElement ||
        e.target instanceof HTMLSelectElement) {
      return;
    }

    if (e.key === 'h' && !e.ctrlKey && !e.metaKey && !e.altKey && !this.isOpen) {
      const keyboardModal = document.getElementById('keyboard-shortcuts-modal');
      const settingsModal = document.getElementById('settings-modal');
      // Don't open if other modals are open
      if (keyboardModal?.style.display === 'flex' ||
          settingsModal?.style.display === 'flex') {
        return;
      }
      e.preventDefault();
      this.open();
    }
  }

  /**
   * Handle keyboard events when modal is open
   */
  private handleKeyDown(e: KeyboardEvent): void {
    if (!this.isOpen) return;

    if (e.key === 'Escape') {
      e.preventDefault();
      this.close();
      return;
    }

    // Tab key for focus trapping
    if (e.key === 'Tab') {
      this.handleTabKey(e);
    }

    // Arrow keys for navigation between sections
    if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
      const activeElement = document.activeElement;
      if (activeElement?.classList.contains('help-nav-item')) {
        e.preventDefault();
        this.navigateSection(e.key === 'ArrowDown' ? 1 : -1);
      }
    }
  }

  /**
   * Handle Tab key for focus trapping
   */
  private handleTabKey(e: KeyboardEvent): void {
    if (!this.modal) return;

    const focusableElements = this.modal.querySelectorAll(
      'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"]), details summary'
    );

    if (focusableElements.length === 0) return;

    const firstElement = focusableElements[0] as HTMLElement;
    const lastElement = focusableElements[focusableElements.length - 1] as HTMLElement;

    if (e.shiftKey) {
      if (document.activeElement === firstElement) {
        e.preventDefault();
        lastElement.focus();
      }
    } else {
      if (document.activeElement === lastElement) {
        e.preventDefault();
        firstElement.focus();
      }
    }
  }

  /**
   * Navigate between sections using arrow keys
   */
  private navigateSection(direction: number): void {
    const sections = ['maps', 'analytics', 'filters', 'realtime', 'settings', 'faq'];
    const currentIndex = sections.indexOf(this.activeSection);
    const newIndex = (currentIndex + direction + sections.length) % sections.length;
    this.showSection(sections[newIndex]);

    // Focus the new nav item
    const navItem = this.modal?.querySelector(`[data-section="${sections[newIndex]}"]`) as HTMLElement;
    navItem?.focus();
  }

  /**
   * Show a specific help section
   */
  private showSection(sectionId: string): void {
    this.activeSection = sectionId;

    // Update nav items
    const navItems = this.modal?.querySelectorAll('.help-nav-item');
    navItems?.forEach(item => {
      const isActive = item.getAttribute('data-section') === sectionId;
      item.classList.toggle('active', isActive);
      item.setAttribute('aria-selected', isActive.toString());
    });

    // Update content sections
    const sections = this.modal?.querySelectorAll('.help-content-section');
    sections?.forEach(section => {
      const isActive = section.id === `help-content-${sectionId}`;
      section.classList.toggle('active', isActive);
      if (isActive) {
        section.removeAttribute('hidden');
      } else {
        section.setAttribute('hidden', '');
      }
    });
  }

  /**
   * Open the help modal
   */
  open(): void {
    if (!this.modal) return;

    this.previouslyFocusedElement = document.activeElement as HTMLElement;

    this.modal.style.display = 'flex';
    this.modal.setAttribute('aria-hidden', 'false');
    this.isOpen = true;

    // Reset to first section
    this.showSection('maps');

    // Focus close button
    setTimeout(() => {
      const closeBtn = this.modal?.querySelector('.help-modal-close') as HTMLElement;
      closeBtn?.focus();
    }, 100);

    logger.debug('Help documentation modal opened');
  }

  /**
   * Close the help modal
   */
  close(): void {
    if (!this.modal) return;

    this.modal.setAttribute('aria-hidden', 'true');
    this.isOpen = false;

    setTimeout(() => {
      if (this.modal) {
        this.modal.style.display = 'none';
      }
    }, 200);

    // Restore focus
    if (this.previouslyFocusedElement) {
      this.previouslyFocusedElement.focus();
      this.previouslyFocusedElement = null;
    }

    logger.debug('Help documentation modal closed');
  }

  /**
   * Check if modal is open
   */
  isModalOpen(): boolean {
    return this.isOpen;
  }

  /**
   * Open modal to a specific section
   */
  openToSection(sectionId: string): void {
    this.open();
    setTimeout(() => this.showSection(sectionId), 100);
  }

  /**
   * Destroy the manager and clean up all event listeners
   */
  destroy(): void {
    // Remove document-level keyboard listeners
    document.removeEventListener('keydown', this.boundHandleKeyDown);
    document.removeEventListener('keydown', this.boundHandleHelpKey);

    // Remove help button listener
    if (this.helpButton) {
      this.helpButton.removeEventListener('click', this.boundHandleHelpButtonClick);
      this.helpButton = null;
    }

    // Remove close button listener
    if (this.closeBtn) {
      this.closeBtn.removeEventListener('click', this.boundHandleCloseClick);
      this.closeBtn = null;
    }

    // Remove overlay click listener
    if (this.modal) {
      this.modal.removeEventListener('click', this.boundHandleOverlayClick);
    }

    // Remove nav item listeners
    this.navItemClickHandlers.forEach((handler, item) => {
      item.removeEventListener('click', handler);
    });
    this.navItemClickHandlers.clear();
    this.navItems = null;

    // Remove modal from DOM
    if (this.modal) {
      this.modal.remove();
      this.modal = null;
    }

    logger.debug('HelpDocumentationManager destroyed');
  }
}

export default HelpDocumentationManager;
