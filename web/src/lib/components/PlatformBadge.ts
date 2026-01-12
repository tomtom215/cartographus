// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * PlatformBadge - Reusable platform indicator component
 *
 * Displays a badge for media server platforms (Plex, Jellyfin, Emby, Tautulli)
 * with appropriate styling and optional click handling.
 */

import { escapeHtml } from '../sanitize';

/**
 * Platform types supported by the badge
 */
export type Platform = 'plex' | 'jellyfin' | 'emby' | 'tautulli';

/**
 * Platform display configuration
 */
interface PlatformConfig {
  name: string;
  icon: string;
  color: string;
}

/**
 * Platform configurations with icons and colors
 */
const PLATFORM_CONFIGS: Record<Platform, PlatformConfig> = {
  plex: {
    name: 'Plex',
    icon: '\u25B6',  // Play triangle (represents Plex's logo concept)
    color: '#e5a00d'
  },
  jellyfin: {
    name: 'Jellyfin',
    icon: '\u2B22',  // Hexagon (represents Jellyfin's blob logo)
    color: '#884dff'
  },
  emby: {
    name: 'Emby',
    icon: '\u25CF',  // Circle (represents Emby's circular logo)
    color: '#52b54b'
  },
  tautulli: {
    name: 'Tautulli',
    icon: '\u2139',  // Info icon (represents Tautulli's analytics nature)
    color: '#dc7633'
  }
};

/**
 * Options for rendering a platform badge
 */
export interface PlatformBadgeOptions {
  /** The platform to display */
  platform: Platform;
  /** Whether the platform is available/linked */
  available?: boolean;
  /** Whether the badge is clickable */
  clickable?: boolean;
  /** Optional platform-specific ID to display in tooltip */
  platformId?: string;
  /** Optional click handler */
  onClick?: (platform: Platform) => void;
  /** Size variant */
  size?: 'small' | 'medium' | 'large';
  /** Show text label */
  showLabel?: boolean;
}

/**
 * PlatformBadge class for creating platform indicator badges
 */
export class PlatformBadge {
  private options: PlatformBadgeOptions;
  private element: HTMLElement | null = null;
  private clickHandler: (() => void) | null = null;

  constructor(options: PlatformBadgeOptions) {
    this.options = {
      available: true,
      clickable: false,
      size: 'medium',
      showLabel: true,
      ...options
    };
  }

  /**
   * Render the badge and return the HTML element
   */
  render(): HTMLElement {
    const config = PLATFORM_CONFIGS[this.options.platform];
    const { available, clickable, platformId, showLabel } = this.options;

    // Create badge element
    this.element = document.createElement('span');
    this.element.className = this.buildClassName();
    this.element.setAttribute('role', 'img');
    this.element.setAttribute('aria-label', `${config.name}${available ? '' : ' (not linked)'}`);

    if (platformId) {
      this.element.setAttribute('title', `${config.name}: ${platformId}`);
    }

    // Build inner HTML
    const iconHtml = `<span class="platform-badge-icon" aria-hidden="true">${escapeHtml(config.icon)}</span>`;
    const labelHtml = showLabel ? `<span class="platform-badge-label">${escapeHtml(config.name)}</span>` : '';

    this.element.innerHTML = iconHtml + labelHtml;

    // Add click handler if clickable
    if (clickable && this.options.onClick) {
      this.clickHandler = () => {
        if (this.options.onClick) {
          this.options.onClick(this.options.platform);
        }
      };
      this.element.addEventListener('click', this.clickHandler);
      this.element.style.cursor = 'pointer';
    }

    return this.element;
  }

  /**
   * Build CSS class name for the badge
   */
  private buildClassName(): string {
    const classes = ['platform-badge', `platform-badge--${this.options.platform}`];

    if (!this.options.available) {
      classes.push('unavailable');
    }

    if (this.options.clickable) {
      classes.push('clickable');
    }

    if (this.options.size && this.options.size !== 'medium') {
      classes.push(`platform-badge--${this.options.size}`);
    }

    return classes.join(' ');
  }

  /**
   * Update the badge availability status
   */
  setAvailable(available: boolean): void {
    this.options.available = available;
    if (this.element) {
      if (available) {
        this.element.classList.remove('unavailable');
      } else {
        this.element.classList.add('unavailable');
      }
    }
  }

  /**
   * Update the platform ID
   */
  setPlatformId(platformId: string | undefined): void {
    this.options.platformId = platformId;
    if (this.element && platformId) {
      const config = PLATFORM_CONFIGS[this.options.platform];
      this.element.setAttribute('title', `${config.name}: ${platformId}`);
    }
  }

  /**
   * Destroy the badge and cleanup
   */
  destroy(): void {
    if (this.element && this.clickHandler) {
      this.element.removeEventListener('click', this.clickHandler);
    }
    this.clickHandler = null;
    this.element = null;
  }
}

/**
 * Render a group of platform badges
 */
export function renderPlatformBadges(
  platforms: Array<{
    platform: Platform;
    available: boolean;
    platformId?: string;
  }>,
  onClick?: (platform: Platform) => void
): HTMLElement {
  const container = document.createElement('div');
  container.className = 'platform-badges-group';

  platforms.forEach(({ platform, available, platformId }) => {
    const badge = new PlatformBadge({
      platform,
      available,
      platformId,
      clickable: !!onClick,
      onClick
    });
    container.appendChild(badge.render());
  });

  return container;
}

/**
 * Get platform configuration
 */
export function getPlatformConfig(platform: Platform): PlatformConfig {
  return PLATFORM_CONFIGS[platform];
}

/**
 * Check if a string is a valid platform
 */
export function isValidPlatform(value: string): value is Platform {
  return value in PLATFORM_CONFIGS;
}

/**
 * Get all supported platforms
 */
export function getAllPlatforms(): Platform[] {
  return Object.keys(PLATFORM_CONFIGS) as Platform[];
}

export default PlatformBadge;
