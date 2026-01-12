// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
import { MapboxOverlay } from '@deck.gl/mapbox';
import { ScatterplotLayer } from '@deck.gl/layers';
import type { PickingInfo, Color } from '@deck.gl/core';
import maplibregl from 'maplibre-gl';
import type { Map as MapLibreMap } from 'maplibre-gl';
import type { LocationStats } from './api';
import { MapConfigManager, createMapStyle, getTerrainSource } from './map-config';
import { createLogger } from './logger';

const logger = createLogger('Globe');

/**
 * GlobeManagerDeckGL - 3D globe visualization using deck.gl + MapLibre GL JS
 *
 * Features:
 * - Globe projection for 3D visualization
 * - deck.gl ScatterplotLayer for location markers
 * - Configurable tile sources (CartoDB, OSM, custom)
 * - Optional terrain support
 * - Smooth transitions and animations
 */
export class GlobeManagerDeckGL {
    private overlay: MapboxOverlay | null = null;
    private map: MapLibreMap | null = null;
    private containerId: string;
    private currentData: LocationStats[] = [];
    private isDarkTheme: boolean = true;
    private tooltipElement: HTMLElement | null = null;
    private autoRotateAnimationId: number | null = null;
    private isVisible: boolean = false;
    private pendingResizeId: number | null = null;

    constructor(containerId: string) {
        this.containerId = containerId;
        this.isDarkTheme = !document.documentElement.hasAttribute('data-theme');
        this.createTooltipElement();
    }

    /**
     * Initialize the globe with MapLibre map instance
     */
    public initialize(): void {
        const container = document.getElementById(this.containerId);
        if (!container) {
            logger.error('Container not found', { containerId: this.containerId });
            return;
        }

        // Get map configuration
        const config = MapConfigManager.getInstance().getConfig();
        const style = createMapStyle(config);

        // Create MapLibre map with globe projection
        // Note: projection is supported in MapLibre v5 but types may not reflect it
        this.map = new maplibregl.Map({
            container: this.containerId,
            style: style,
            center: [0, 20],
            zoom: 2,
            maxPitch: 85,
        } as maplibregl.MapOptions & { projection?: { type: string } });

        // Add navigation controls
        this.map.addControl(
            new maplibregl.NavigationControl({
                visualizePitch: true,
                showCompass: true,
                showZoom: true,
            }),
            'top-right'
        );

        // Wait for map to load before adding deck.gl overlay
        this.map.on('load', () => {
            if (!this.map) return;

            try {
                // Set globe projection if supported (must be done after style loads)
                // Note: This can fail if the map container is hidden during initialization
                if ('setProjection' in this.map) {
                    (this.map as unknown as { setProjection: (p: { type: string }) => void }).setProjection({ type: 'globe' });
                }

                // Initialize terrain if configured
                this.initializeTerrain();
                // Initialize deck.gl overlay with shared WebGL context
                // Start with an empty ScatterplotLayer to avoid null layer issues during initial render
                const initialLayer = new ScatterplotLayer<LocationStats>({
                    id: 'playback-locations',
                    data: [],
                    getPosition: () => [0, 0, 0],
                    getRadius: () => 0,
                    getFillColor: () => [0, 0, 0, 0] as Color,
                    visible: false,
                });

                this.overlay = new MapboxOverlay({
                    interleaved: true, // Share WebGL context with MapLibre
                    layers: [initialLayer],
                });

                // Add overlay to MapLibre map
                this.map.addControl(this.overlay);

                // Update with current data if any
                if (this.currentData.length > 0) {
                    this.updateVisualization(this.currentData);
                }
            } catch (error) {
                logger.error('Failed to initialize deck.gl overlay', { error });
            }
        });
    }

    /**
     * Initialize terrain if configured
     * Note: Can throw if map is in invalid state (e.g., hidden container)
     */
    private initializeTerrain(): void {
        if (!this.map) return;

        const config = MapConfigManager.getInstance().getConfig();
        const terrainConfig = getTerrainSource(config);

        if (!terrainConfig) return;

        // Terrain is already configured in the style from createMapStyle
        // Just enable terrain rendering
        // Note: This can fail silently on hidden containers, which is fine
        // as it will be properly set up when the globe becomes visible
        try {
            this.map.setTerrain({
                source: terrainConfig.source,
                exaggeration: terrainConfig.exaggeration,
            });
        } catch (error) {
            logger.warn('Terrain initialization deferred', { error });
        }
    }

    /**
     * Update globe with location data
     */
    public updateLocations(locations: LocationStats[]): void {
        // Filter out invalid locations (0,0 coordinates indicate geolocation failures)
        const validLocations = locations.filter(
            (location) => location.latitude !== 0 || location.longitude !== 0
        );

        this.currentData = validLocations;
        this.updateVisualization(validLocations);
    }

    /**
     * Update the deck.gl visualization with new data
     * Safe to call during view transitions - will silently skip if not ready
     */
    private updateVisualization(locations: LocationStats[]): void {
        // Skip if globe is hidden or not initialized
        if (!this.overlay || !this.map || !this.isVisible) {
            return;
        }

        try {
            // Filter out any null or invalid data entries that could cause deck.gl errors
            const validData = locations.filter(
                (d): d is LocationStats =>
                    d != null &&
                    typeof d.longitude === 'number' &&
                    typeof d.latitude === 'number' &&
                    !isNaN(d.longitude) &&
                    !isNaN(d.latitude)
            );

            const layer = new ScatterplotLayer<LocationStats>({
                id: 'playback-locations',
                data: validData,

                // Position accessor with null guard
                getPosition: (d: LocationStats) => {
                    if (!d) return [0, 0, 0];
                    return [d.longitude ?? 0, d.latitude ?? 0, 0];
                },

                // Size based on playback count with null guard
                getRadius: (d: LocationStats) => {
                    if (!d) return 0;
                    const playbacks = d.playback_count ?? 0;
                    return Math.max(50000, Math.min(200000, Math.sqrt(playbacks) * 20000));
                },

                // Color based on playback count with null guard
                getFillColor: (d: LocationStats): Color => {
                    if (!d) return [0, 0, 0, 0];
                    return this.getColorByPlaybackCount(d.playback_count ?? 0);
                },

                // Styling
                opacity: 0.8,
                radiusScale: 1,
                radiusMinPixels: 3,
                radiusMaxPixels: 30,

                // Interactivity
                pickable: true,
                autoHighlight: true,

                // Event handlers
                onClick: (info: PickingInfo<LocationStats>) => {
                    if (info.object) {
                        this.showTooltip(info);
                    }
                },

                onHover: (info: PickingInfo<LocationStats>) => {
                    if (info.object) {
                        this.showTooltip(info);
                    } else {
                        this.hideTooltip();
                    }
                },

                // Transitions for smooth updates
                transitions: {
                    getPosition: 300,
                    getRadius: 300,
                    getFillColor: 300,
                },

                // Update triggers
                updateTriggers: {
                    getFillColor: [this.isDarkTheme],
                },
            });

            // Update overlay with new layers
            this.overlay.setProps({
                layers: [layer],
            });
        } catch (error) {
            // Suppress deck.gl errors during render - these can occur during
            // initialization or when data is transitioning
            logger.warn('Error updating visualization', { error });
        }
    }

    /**
     * Parse CSS color string to RGB array
     * Supports hex colors (#rrggbb, #rgb) and rgb() format
     */
    private parseCssColor(cssValue: string): [number, number, number] {
        const trimmed = cssValue.trim();

        // Handle hex colors
        if (trimmed.startsWith('#')) {
            let hex = trimmed.slice(1);
            // Expand 3-char hex to 6-char
            if (hex.length === 3) {
                hex = hex[0] + hex[0] + hex[1] + hex[1] + hex[2] + hex[2];
            }
            const r = parseInt(hex.slice(0, 2), 16);
            const g = parseInt(hex.slice(2, 4), 16);
            const b = parseInt(hex.slice(4, 6), 16);
            return [r, g, b];
        }

        // Handle rgb() format
        const rgbMatch = trimmed.match(/rgb\s*\(\s*(\d+)\s*,\s*(\d+)\s*,\s*(\d+)\s*\)/);
        if (rgbMatch) {
            return [parseInt(rgbMatch[1]), parseInt(rgbMatch[2]), parseInt(rgbMatch[3])];
        }

        // Fallback
        return [78, 205, 196]; // Default teal
    }

    /**
     * Get CSS variable value from document
     * Returns empty string if document is not ready or variable doesn't exist
     */
    private getCssVariable(name: string): string {
        try {
            const value = getComputedStyle(document.documentElement).getPropertyValue(name);
            return value ? value.trim() : '';
        } catch {
            // Document may not be ready during rapid view switches
            return '';
        }
    }

    /**
     * Get color based on playback count
     * Uses CSS variables for theming
     */
    private getColorByPlaybackCount(count: number): Color {
        const alpha = 220;

        if (count > 500) {
            const rgb = this.parseCssColor(this.getCssVariable('--globe-marker-high') || '#ec4899');
            return [rgb[0], rgb[1], rgb[2], alpha];
        }
        if (count > 200) {
            const rgb = this.parseCssColor(this.getCssVariable('--globe-marker-medium-high') || '#f97316');
            return [rgb[0], rgb[1], rgb[2], alpha];
        }
        if (count > 50) {
            const rgb = this.parseCssColor(this.getCssVariable('--globe-marker-medium') || '#f59e0b');
            return [rgb[0], rgb[1], rgb[2], alpha];
        }

        const rgb = this.parseCssColor(this.getCssVariable('--globe-marker-low') || '#14b8a6');
        return [rgb[0], rgb[1], rgb[2], alpha];
    }

    /**
     * Create tooltip element
     * Uses CSS variables for theming
     */
    private createTooltipElement(): void {
        this.tooltipElement = document.createElement('div');
        this.tooltipElement.id = 'deckgl-tooltip';
        this.tooltipElement.className = 'globe-tooltip';
        this.updateTooltipStyles();
        document.body.appendChild(this.tooltipElement);
    }

    /**
     * Update tooltip styles from CSS variables
     * Called on creation and theme changes
     */
    private updateTooltipStyles(): void {
        if (!this.tooltipElement) return;

        const bg = this.getCssVariable('--globe-tooltip-bg') || 'rgba(20, 20, 30, 0.95)';
        const border = this.getCssVariable('--globe-tooltip-border') || '#14b8a6';
        const text = this.getCssVariable('--globe-tooltip-text') || '#eaeaea';

        this.tooltipElement.style.cssText = `
            position: absolute;
            z-index: 1000;
            pointer-events: none;
            background: ${bg};
            border: 1px solid ${border};
            border-radius: 4px;
            padding: 12px;
            color: ${text};
            font-size: 12px;
            display: none;
            max-width: 300px;
        `;
    }

    /**
     * Show tooltip for location
     * Note: Uses textContent to prevent XSS from geolocation data
     */
    private showTooltip(info: PickingInfo<LocationStats>): void {
        if (!this.tooltipElement || !info.object) return;

        const location = info.object;

        // Safely escape location data to prevent XSS
        const escapeHtml = (text: string): string => {
            const div = document.createElement('div');
            div.textContent = text;
            return div.innerHTML;
        };

        const city = escapeHtml(location.city || 'Unknown');
        const region = location.region ? escapeHtml(location.region) : null;
        const country = escapeHtml(location.country || 'Unknown');
        const locationName = region ? `${city}, ${region}, ${country}` : `${city}, ${country}`;

        // Use nullish coalescing to handle missing data
        const playbackCount = location.playback_count ?? 0;
        const uniqueUsers = location.unique_users ?? 0;
        const avgCompletion = location.avg_completion ?? 0;

        const html = `
            <div style="padding: 4px;">
                <div style="font-weight: 600; font-size: 14px; margin-bottom: 8px; color: #fff;">
                    ${locationName}
                </div>
                <div style="line-height: 1.8;">
                    <div><span style="color: #a0a0a0;">Playbacks:</span> <strong>${playbackCount.toLocaleString()}</strong></div>
                    <div><span style="color: #a0a0a0;">Unique Users:</span> <strong>${uniqueUsers}</strong></div>
                    <div><span style="color: #a0a0a0;">Avg Completion:</span> <strong>${avgCompletion.toFixed(1)}%</strong></div>
                    ${location.first_seen ? `<div style="margin-top: 4px; padding-top: 4px; border-top: 1px solid rgba(255,255,255,0.1);"><span style="color: #a0a0a0;">First Seen:</span> ${new Date(location.first_seen).toLocaleDateString()}</div>` : ''}
                    ${location.last_seen ? `<div><span style="color: #a0a0a0;">Last Seen:</span> ${new Date(location.last_seen).toLocaleDateString()}</div>` : ''}
                </div>
            </div>
        `;

        this.tooltipElement.innerHTML = html;
        this.tooltipElement.style.display = 'block';
        this.tooltipElement.style.left = `${info.x}px`;
        this.tooltipElement.style.top = `${info.y}px`;
    }

    /**
     * Hide tooltip
     */
    private hideTooltip(): void {
        if (this.tooltipElement) {
            this.tooltipElement.style.display = 'none';
        }
    }

    /**
     * Update theme
     * Also updates tooltip styles for theming
     */
    public updateTheme(isDark: boolean): void {
        this.isDarkTheme = isDark;

        // Update map style
        if (this.map) {
            try {
                const configManager = MapConfigManager.getInstance();
                const config = configManager.getConfig();
                const tileProvider = isDark ? 'carto-dark' : 'carto-light';
                configManager.setConfig({ ...config, tileProvider, theme: isDark ? 'dark' : 'light' });

                const newStyle = createMapStyle(configManager.getConfig());
                this.map.setStyle(newStyle);
            } catch (error) {
                logger.warn('Theme update deferred', { error });
            }
        }

        // Update tooltip styles for new theme
        this.updateTooltipStyles();

        // Re-render with updated theme
        this.updateVisualization(this.currentData);
    }

    /**
     * Enable auto-rotation
     * Note: Implemented using map bearing animation
     */
    public enableAutoRotate(): void {
        // Don't start if not visible or map not ready
        if (!this.map || !this.isVisible) return;

        // Don't start if already rotating
        if (this.autoRotateAnimationId !== null) return;

        const rotateCamera = () => {
            // Stop if animation was cancelled, not visible, or map destroyed
            if (this.autoRotateAnimationId === null || !this.isVisible || !this.map) {
                this.autoRotateAnimationId = null;
                return;
            }
            try {
                // Double-check map exists before accessing methods
                const map = this.map;
                if (!map) {
                    this.autoRotateAnimationId = null;
                    return;
                }
                map.easeTo({
                    bearing: map.getBearing() + 1,
                    duration: 100,
                    easing: (t) => t,
                });
                this.autoRotateAnimationId = requestAnimationFrame(rotateCamera);
            } catch {
                // Stop rotation on error (e.g., map destroyed during animation)
                this.autoRotateAnimationId = null;
            }
        };

        this.autoRotateAnimationId = requestAnimationFrame(rotateCamera);
    }

    /**
     * Disable auto-rotation
     */
    public disableAutoRotate(): void {
        if (this.autoRotateAnimationId !== null) {
            cancelAnimationFrame(this.autoRotateAnimationId);
            this.autoRotateAnimationId = null;
        }
    }

    /**
     * Reset camera view
     */
    public resetView(): void {
        if (!this.map || !this.isVisible) return;

        try {
            this.map.easeTo({
                center: [0, 20],
                zoom: 2,
                pitch: 0,
                bearing: 0,
                duration: 1000,
            });
        } catch {
            // Ignore reset errors during view transitions
        }
    }

    /**
     * Destroy the globe instance
     */
    public destroy(): void {
        // Mark as not visible to prevent pending operations
        this.isVisible = false;

        // Cancel any pending resize operations
        if (this.pendingResizeId !== null) {
            clearTimeout(this.pendingResizeId);
            this.pendingResizeId = null;
        }

        // Stop any running auto-rotation animation
        this.disableAutoRotate();
        this.hideTooltip();

        // Clean up tooltip element
        if (this.tooltipElement && this.tooltipElement.parentNode) {
            this.tooltipElement.parentNode.removeChild(this.tooltipElement);
        }

        // Clean up map and overlay with error handling
        // These can throw if map is in an invalid state
        try {
            if (this.map && this.overlay) {
                this.map.removeControl(this.overlay);
            }
        } catch (error) {
            logger.warn('Error removing overlay', { error });
        }

        try {
            if (this.map) {
                this.map.remove();
            }
        } catch (error) {
            logger.warn('Error removing map', { error });
        }

        this.map = null;
        this.overlay = null;
    }

    /**
     * Show the globe (resize map to trigger deck.gl update)
     */
    public show(): void {
        if (!this.map) return;

        this.isVisible = true;

        // Cancel any pending resize from a previous show/hide cycle
        if (this.pendingResizeId !== null) {
            clearTimeout(this.pendingResizeId);
            this.pendingResizeId = null;
        }

        // Trigger map resize to ensure deck.gl updates
        // Use a tracked timeout that can be cancelled if hide() is called
        this.pendingResizeId = window.setTimeout(() => {
            this.pendingResizeId = null;
            // Only resize if still visible and map exists
            if (this.isVisible && this.map) {
                try {
                    this.map.resize();
                } catch (e) {
                    // Ignore resize errors during rapid view switching
                    logger.warn('Resize error (ignored)', { error: e });
                }
            }
        }, 100);
    }

    /**
     * Hide the globe
     * Cancels all pending operations and stops auto-rotation
     */
    public hide(): void {
        this.isVisible = false;

        // Cancel any pending resize operations
        if (this.pendingResizeId !== null) {
            clearTimeout(this.pendingResizeId);
            this.pendingResizeId = null;
        }

        // Stop auto-rotation to prevent errors during view switches
        this.disableAutoRotate();

        this.hideTooltip();
    }

    /**
     * Get the MapLibre map instance
     */
    public getMap(): MapLibreMap | null {
        return this.map;
    }
}
