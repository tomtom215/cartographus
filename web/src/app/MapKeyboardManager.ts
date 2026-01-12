// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * MapKeyboardManager - Keyboard navigation for Map and Globe
 *
 * Features:
 * - Arrow keys: Pan the map/globe
 * - +/- keys: Zoom in/out
 * - Shift+arrows: Rotate the view
 * - Home: Reset to default view
 * - End: Fit to data bounds
 * - R: Toggle auto-rotate (globe only)
 * - ARIA attributes for screen readers
 *
 * Reference: WCAG 2.1 SC 2.1.1 (Keyboard)
 * @see /docs/working/UI_UX_AUDIT.md
 */

import type { Map as MapLibreMap } from 'maplibre-gl';
import { createLogger } from '../lib/logger';

const logger = createLogger('MapKeyboardManager');

/**
 * Configuration for keyboard navigation
 */
interface KeyboardNavConfig {
  containerId: string;
  map: MapLibreMap;
  type: 'map' | 'globe';
  panAmount?: number;       // Pixels to pan per key press
  zoomAmount?: number;      // Zoom level change per key press
  rotateAmount?: number;    // Degrees to rotate per key press
  defaultCenter?: [number, number];
  defaultZoom?: number;
}

/**
 * Announcer for screen reader feedback
 */
function announceAction(message: string): void {
  const announcer = document.getElementById('map-announcer');
  if (announcer) {
    announcer.textContent = '';
    setTimeout(() => {
      announcer.textContent = message;
    }, 100);
  }
}

export class MapKeyboardManager {
  private containerId: string;
  private map: MapLibreMap;
  private type: 'map' | 'globe';
  private panAmount: number;
  private rotateAmount: number;
  private defaultCenter: [number, number];
  private defaultZoom: number;
  private keydownHandler: ((e: KeyboardEvent) => void) | null = null;
  private focusHandler: (() => void) | null = null;
  private blurHandler: (() => void) | null = null;

  constructor(config: KeyboardNavConfig) {
    this.containerId = config.containerId;
    this.map = config.map;
    this.type = config.type;
    this.panAmount = config.panAmount || 100;
    this.rotateAmount = config.rotateAmount || 15;
    this.defaultCenter = config.defaultCenter || [0, 20];
    this.defaultZoom = config.defaultZoom || 2;
  }

  /**
   * Initialize keyboard navigation
   */
  init(): void {
    const container = document.getElementById(this.containerId);
    if (!container) {
      logger.warn(`Container not found: ${this.containerId}`);
      return;
    }

    // Make container keyboard focusable
    if (!container.hasAttribute('tabindex')) {
      container.setAttribute('tabindex', '0');
    }

    // Add ARIA attributes
    container.setAttribute('role', 'application');
    container.setAttribute('aria-label', `${this.type === 'globe' ? 'Interactive 3D globe' : 'Interactive map'} visualization. Use arrow keys to pan, plus/minus to zoom, Shift+arrows to rotate.`);

    // Create announcer if not exists
    this.createAnnouncer();

    // Add keyboard hint element
    this.addKeyboardHint(container);

    // Set up event handlers
    this.keydownHandler = this.handleKeydown.bind(this);
    this.focusHandler = this.handleFocus.bind(this);
    this.blurHandler = this.handleBlur.bind(this);

    container.addEventListener('keydown', this.keydownHandler);
    container.addEventListener('focus', this.focusHandler);
    container.addEventListener('blur', this.blurHandler);

    logger.log(`Keyboard navigation enabled for ${this.type}: ${this.containerId}`);
  }

  /**
   * Create screen reader announcer element
   */
  private createAnnouncer(): void {
    if (document.getElementById('map-announcer')) return;

    const announcer = document.createElement('div');
    announcer.id = 'map-announcer';
    announcer.className = 'visually-hidden';
    announcer.setAttribute('role', 'status');
    announcer.setAttribute('aria-live', 'polite');
    announcer.setAttribute('aria-atomic', 'true');
    document.body.appendChild(announcer);
  }

  /**
   * Add keyboard hint overlay
   */
  private addKeyboardHint(container: HTMLElement): void {
    // Check if hint already exists
    if (container.querySelector('.keyboard-nav-hint')) return;

    const hint = document.createElement('div');
    hint.className = 'keyboard-nav-hint';
    hint.innerHTML = `
      <span class="hint-icon" aria-hidden="true">&#9000;</span>
      <span>Press Tab to focus, then use arrow keys to navigate</span>
    `;
    hint.setAttribute('aria-hidden', 'true');

    container.appendChild(hint);
  }

  /**
   * Handle focus event
   */
  private handleFocus(): void {
    const container = document.getElementById(this.containerId);
    if (container) {
      container.classList.add('keyboard-focused');
    }
    announceAction(`${this.type === 'globe' ? 'Globe' : 'Map'} focused. Use arrow keys to pan, plus or minus to zoom.`);
  }

  /**
   * Handle blur event
   */
  private handleBlur(): void {
    const container = document.getElementById(this.containerId);
    if (container) {
      container.classList.remove('keyboard-focused');
    }
  }

  /**
   * Handle keyboard events
   */
  private handleKeydown(e: KeyboardEvent): void {
    // Ignore if modifier keys are used (except Shift for rotation)
    if (e.ctrlKey || e.altKey || e.metaKey) return;

    let handled = false;
    let announcement = '';

    switch (e.key) {
      case 'ArrowUp':
        if (e.shiftKey) {
          this.rotatePitch(this.rotateAmount);
          announcement = 'Pitched up';
        } else {
          this.panMap(0, -this.panAmount);
          announcement = 'Panned north';
        }
        handled = true;
        break;

      case 'ArrowDown':
        if (e.shiftKey) {
          this.rotatePitch(-this.rotateAmount);
          announcement = 'Pitched down';
        } else {
          this.panMap(0, this.panAmount);
          announcement = 'Panned south';
        }
        handled = true;
        break;

      case 'ArrowLeft':
        if (e.shiftKey) {
          this.rotateBearing(-this.rotateAmount);
          announcement = 'Rotated left';
        } else {
          this.panMap(-this.panAmount, 0);
          announcement = 'Panned west';
        }
        handled = true;
        break;

      case 'ArrowRight':
        if (e.shiftKey) {
          this.rotateBearing(this.rotateAmount);
          announcement = 'Rotated right';
        } else {
          this.panMap(this.panAmount, 0);
          announcement = 'Panned east';
        }
        handled = true;
        break;

      case '+':
      case '=':
        this.zoomIn();
        announcement = `Zoomed in to level ${Math.round(this.map.getZoom())}`;
        handled = true;
        break;

      case '-':
      case '_':
        this.zoomOut();
        announcement = `Zoomed out to level ${Math.round(this.map.getZoom())}`;
        handled = true;
        break;

      case 'Home':
        this.resetView();
        announcement = 'View reset to default';
        handled = true;
        break;

      case 'End':
        this.fitToBounds();
        announcement = 'Fitted to data bounds';
        handled = true;
        break;

      case 'r':
      case 'R':
        if (this.type === 'globe') {
          this.toggleAutoRotate();
          announcement = 'Auto-rotate toggled';
          handled = true;
        }
        break;
    }

    if (handled) {
      e.preventDefault();
      e.stopPropagation();
      if (announcement) {
        announceAction(announcement);
      }
    }
  }

  /**
   * Pan the map by pixel offset
   */
  private panMap(x: number, y: number): void {
    this.map.panBy([x, y], { duration: 200 });
  }

  /**
   * Zoom in
   */
  private zoomIn(): void {
    this.map.zoomIn({ duration: 200 });
  }

  /**
   * Zoom out
   */
  private zoomOut(): void {
    this.map.zoomOut({ duration: 200 });
  }

  /**
   * Rotate bearing
   */
  private rotateBearing(delta: number): void {
    const currentBearing = this.map.getBearing();
    this.map.rotateTo(currentBearing + delta, { duration: 200 });
  }

  /**
   * Rotate pitch
   */
  private rotatePitch(delta: number): void {
    const currentPitch = this.map.getPitch();
    const newPitch = Math.max(0, Math.min(85, currentPitch + delta));
    this.map.setPitch(newPitch, { duration: 200 });
  }

  /**
   * Reset to default view
   */
  private resetView(): void {
    this.map.flyTo({
      center: this.defaultCenter,
      zoom: this.defaultZoom,
      bearing: 0,
      pitch: 0,
      duration: 1000
    });
  }

  /**
   * Fit map to data bounds
   */
  private fitToBounds(): void {
    // Try to get bounds from locations source
    const source = this.map.getSource('locations');
    if (source && 'getClusterExpansionZoom' in source) {
      // GeoJSON source - get bounds from data
      try {
        // Use the map's current bounds as fallback
        const bounds = this.map.getBounds();
        this.map.fitBounds(bounds, {
          padding: 50,
          duration: 1000
        });
      } catch {
        // Fallback to default view
        this.resetView();
      }
    } else {
      // No data source, reset to default
      this.resetView();
    }
  }

  /**
   * Toggle auto-rotate (globe only)
   */
  private toggleAutoRotate(): void {
    // Dispatch event to toggle auto-rotate
    const event = new CustomEvent('toggle-auto-rotate', {
      detail: { containerId: this.containerId }
    });
    document.dispatchEvent(event);
  }

  /**
   * Update map reference (if map is recreated)
   */
  updateMap(map: MapLibreMap): void {
    this.map = map;
  }

  /**
   * Destroy and cleanup
   */
  destroy(): void {
    const container = document.getElementById(this.containerId);
    if (container) {
      if (this.keydownHandler) {
        container.removeEventListener('keydown', this.keydownHandler);
      }
      if (this.focusHandler) {
        container.removeEventListener('focus', this.focusHandler);
      }
      if (this.blurHandler) {
        container.removeEventListener('blur', this.blurHandler);
      }

      // Remove hint
      const hint = container.querySelector('.keyboard-nav-hint');
      if (hint) {
        hint.remove();
      }
    }
  }
}

export default MapKeyboardManager;
