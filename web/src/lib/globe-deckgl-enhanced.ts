// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
import { MapboxOverlay } from '@deck.gl/mapbox';
import { ScatterplotLayer, ArcLayer } from '@deck.gl/layers';
import { HexagonLayer } from '@deck.gl/aggregation-layers';
import { TripsLayer } from '@deck.gl/geo-layers';
import type { PickingInfo, Color, Layer } from '@deck.gl/core';
import maplibregl from 'maplibre-gl';
import type { Map as MapLibreMap } from 'maplibre-gl';
import type { LocationStats, TemporalHeatmapBucket, ServerInfo } from './api';
import { MapConfigManager, createMapStyle, getTerrainSource } from './map-config';
import { createLogger } from './logger';

const logger = createLogger('GlobeDeckGL');

/**
 * Helper to read CSS variable value from the root document
 * Used to theme globe visualizations with CSS variables
 */
function getCSSVariable(name: string, fallback: string = ''): string {
    return getComputedStyle(document.documentElement).getPropertyValue(name).trim() || fallback;
}

/**
 * Convert CSS color (hex or rgb) to deck.gl Color array [r, g, b, a]
 * Supports hex (#RRGGBB, #RGB), rgb(), and rgba() formats
 */
function cssColorToRGBA(cssColor: string, alpha: number = 255): Color {
    // Handle hex colors
    if (cssColor.startsWith('#')) {
        let hex = cssColor.slice(1);
        // Expand shorthand hex (#RGB -> #RRGGBB)
        if (hex.length === 3) {
            hex = hex[0] + hex[0] + hex[1] + hex[1] + hex[2] + hex[2];
        }
        const r = parseInt(hex.slice(0, 2), 16);
        const g = parseInt(hex.slice(2, 4), 16);
        const b = parseInt(hex.slice(4, 6), 16);
        return [r, g, b, alpha];
    }

    // Handle rgb() and rgba() formats
    const rgbMatch = cssColor.match(/rgba?\s*\(\s*(\d+)\s*,\s*(\d+)\s*,\s*(\d+)(?:\s*,\s*([\d.]+))?\s*\)/);
    if (rgbMatch) {
        return [
            parseInt(rgbMatch[1], 10),
            parseInt(rgbMatch[2], 10),
            parseInt(rgbMatch[3], 10),
            rgbMatch[4] ? Math.round(parseFloat(rgbMatch[4]) * 255) : alpha
        ];
    }

    // Fallback to teal if parsing fails
    return [78, 205, 196, alpha];
}

/**
 * Get themed colors from CSS variables for globe visualization
 * Returns color arrays suitable for deck.gl layers
 */
interface GlobeThemeColors {
    markerHigh: Color;      // High playback count (500+)
    markerMediumHigh: Color; // Medium-high (200-500)
    markerMedium: Color;    // Medium (50-200)
    markerLow: Color;       // Low (<50)
    arcSource: Color;       // Arc source (user location)
    arcTarget: Color;       // Arc target (server)
    tooltipBg: string;
    tooltipBorder: string;
    tooltipText: string;
    textSecondary: string;
}

function getGlobeThemeColors(): GlobeThemeColors {
    // Read colors from CSS variables with fallbacks
    const markerHigh = getCSSVariable('--globe-marker-high', '#ec4899');
    const markerMediumHigh = getCSSVariable('--globe-marker-medium-high', '#f97316');
    const markerMedium = getCSSVariable('--globe-marker-medium', '#f59e0b');
    const markerLow = getCSSVariable('--globe-marker-low', '#14b8a6');
    const tooltipBg = getCSSVariable('--globe-tooltip-bg', 'rgba(20, 20, 30, 0.95)');
    const tooltipBorder = getCSSVariable('--globe-tooltip-border', '#14b8a6');
    const tooltipText = getCSSVariable('--globe-tooltip-text', '#eaeaea');
    const textSecondary = getCSSVariable('--text-secondary', '#a0a0a0');

    return {
        markerHigh: cssColorToRGBA(markerHigh, 220),
        markerMediumHigh: cssColorToRGBA(markerMediumHigh, 220),
        markerMedium: cssColorToRGBA(markerMedium, 220),
        markerLow: cssColorToRGBA(markerLow, 220),
        arcSource: cssColorToRGBA(markerLow, 180),  // Teal for user
        arcTarget: cssColorToRGBA(markerHigh, 180), // Pink for server
        tooltipBg,
        tooltipBorder,
        tooltipText,
        textSecondary
    };
}

/**
 * Layer types available in the enhanced globe visualization
 */
export type GlobeLayerType = 'scatterplot' | 'hexagon' | 'arcs' | 'trips' | 'heatmap';

/**
 * Configuration for which layers are enabled
 */
export interface LayerConfig {
    scatterplot: boolean;
    hexagon: boolean;
    arcs: boolean;
    trips: boolean;
}

/**
 * Trip data structure for TripsLayer
 */
interface TripData {
    path: [number, number, number][];
    timestamps: number[];
    weight: number;
}

/**
 * Keyboard command definition for globe navigation
 */
interface KeyboardCommand {
    action: () => void;
    announcement: string | (() => string);
}

/**
 * GlobeManagerDeckGLEnhanced - Advanced 3D globe visualization using deck.gl 9.2 + MapLibre
 *
 * Features:
 * - Multiple visualization layers (scatterplot, hexagon, arcs, trips)
 * - Temporal animation support with TripsLayer
 * - User→Server connection visualization with ArcLayer
 * - 3D aggregation with HexagonLayer
 * - High-performance WebGL rendering
 * - Smooth transitions and animations
 * - Pluggable tile sources and terrain support
 */
export class GlobeManagerDeckGLEnhanced {
    private overlay: MapboxOverlay | null = null;
    private map: MapLibreMap | null = null;
    private containerId: string;
    private currentData: LocationStats[] = [];
    private temporalData: TemporalHeatmapBucket[] = [];
    private serverLocation: { latitude: number; longitude: number } | null = null;
    private isDarkTheme: boolean = true;
    private tooltipElement: HTMLElement | null = null;

    // Layer configuration
    private enabledLayers: LayerConfig = {
        scatterplot: true,
        hexagon: false,
        arcs: false,
        trips: false,
    };

    // Animation state for TripsLayer
    private animationTime: number = 0;
    private animationFrameId: number | null = null;
    private isAnimating: boolean = false;

    // Layer update optimization - track data changes to avoid unnecessary recreation
    private dataChanged: boolean = true;
    private layerCache: Map<string, Layer> = new Map();

    // Cached theme colors - updated when theme changes
    private themeColors: GlobeThemeColors;

    // Accessibility: keyboard navigation state
    private keyboardHandler: ((e: KeyboardEvent) => void) | null = null;
    private focusHandler: ((e: FocusEvent) => void) | null = null;
    private blurHandler: ((e: FocusEvent) => void) | null = null;

    constructor(containerId: string) {
        this.containerId = containerId;
        this.isDarkTheme = !document.documentElement.hasAttribute('data-theme');
        this.themeColors = getGlobeThemeColors();
        this.createTooltipElement();
    }

    /**
     * Accessibility: Announce a message to screen readers via live region
     * @param message - The message to announce
     * @param priority - 'polite' (default) or 'assertive' for urgent messages
     */
    public announceToScreenReader(message: string, priority: 'polite' | 'assertive' = 'polite'): void {
        const announcer = document.getElementById('globe-announcer');
        if (announcer) {
            announcer.setAttribute('aria-live', priority);
            // Clear first to ensure re-announcement of same message
            announcer.textContent = '';
            // Use requestAnimationFrame to ensure DOM update is processed
            requestAnimationFrame(() => {
                announcer.textContent = message;
            });
        }
    }

    /**
     * Accessibility: Update data summary for screen readers
     * Called when location data changes
     */
    private updateDataSummary(): void {
        const summary = document.getElementById('globe-data-summary');
        if (!summary) return;

        const locationCount = this.currentData.length;
        const totalPlaybacks = this.currentData.reduce((sum, loc) => sum + loc.playback_count, 0);
        const uniqueCountries = new Set(this.currentData.map(loc => loc.country)).size;

        summary.textContent = `Globe displaying ${locationCount} locations across ${uniqueCountries} countries with ${totalPlaybacks.toLocaleString()} total playbacks.`;
    }

    /**
     * Accessibility: Set up keyboard navigation for the globe
     * Supports arrow keys for panning, +/- for zoom, R for reset
     */
    private setupKeyboardNavigation(): void {
        const container = document.getElementById(this.containerId);
        if (!container || !this.map) return;

        // Remove existing handlers if any
        this.removeKeyboardNavigation();

        this.keyboardHandler = (e: KeyboardEvent) => this.handleKeyboardEvent(e, container);
        this.focusHandler = () => this.handleGlobeFocus(container);
        this.blurHandler = () => this.handleGlobeBlur(container);

        container.addEventListener('keydown', this.keyboardHandler);
        container.addEventListener('focus', this.focusHandler);
        container.addEventListener('blur', this.blurHandler);
    }

    /**
     * Handle keyboard events for globe navigation
     */
    private handleKeyboardEvent(e: KeyboardEvent, container: HTMLElement): void {
        if (!this.map) return;
        if (!this.shouldHandleKeyboardEvent(e, container)) return;

        const command = this.getKeyboardCommand(e.key);
        if (command) {
            this.executeKeyboardCommand(command, e);
        }
    }

    /**
     * Check if keyboard event should be handled
     */
    private shouldHandleKeyboardEvent(e: KeyboardEvent, container: HTMLElement): boolean {
        const target = e.target as HTMLElement;

        // Only handle keys when globe container is focused
        if (target.id !== this.containerId && !container.contains(target)) {
            return false;
        }

        // Don't intercept keys if user is in a form field
        const formTags = ['INPUT', 'TEXTAREA', 'SELECT'];
        return !formTags.includes(target.tagName);
    }

    /**
     * Execute keyboard command and announce action
     */
    private executeKeyboardCommand(command: KeyboardCommand, e: KeyboardEvent): void {
        command.action();
        const announcement = typeof command.announcement === 'function'
            ? command.announcement()
            : command.announcement;
        this.announceToScreenReader(announcement);
        e.preventDefault();
        e.stopPropagation();
    }

    /**
     * Handle globe container focus
     */
    private handleGlobeFocus(container: HTMLElement): void {
        container.classList.add('globe-focused');
        this.announceToScreenReader('Globe focused. Use arrow keys to pan, plus and minus to zoom, R to reset view.');
    }

    /**
     * Handle globe container blur
     */
    private handleGlobeBlur(container: HTMLElement): void {
        container.classList.remove('globe-focused');
    }

    /**
     * Get keyboard command for a given key
     */
    private getKeyboardCommand(key: string): KeyboardCommand | undefined {
        const commands = this.createKeyboardCommands();
        return commands[key];
    }

    /**
     * Create keyboard command mappings
     */
    private createKeyboardCommands(): Record<string, KeyboardCommand> {
        const panAmount = 50;
        const zoomAmount = 0.5;
        const currentZoom = this.map?.getZoom() ?? 2;

        // Navigation command builders
        const panCommand = (x: number, y: number, direction: string): KeyboardCommand => ({
            action: () => this.map?.panBy([x, y], { duration: 100 }),
            announcement: `Panning ${direction}`
        });

        const zoomCommand = (delta: number): KeyboardCommand => ({
            action: () => this.map?.zoomTo(currentZoom + delta, { duration: 200 }),
            announcement: () => {
                const newZoom = Math.round(currentZoom + delta);
                const action = delta > 0 ? 'in' : 'out';
                return `Zooming ${action} to level ${newZoom}`;
            }
        });

        // Build command map with all key aliases
        return {
            // Pan north
            'ArrowUp': panCommand(0, -panAmount, 'north'),
            'w': panCommand(0, -panAmount, 'north'),
            'W': panCommand(0, -panAmount, 'north'),
            // Pan south
            'ArrowDown': panCommand(0, panAmount, 'south'),
            's': panCommand(0, panAmount, 'south'),
            'S': panCommand(0, panAmount, 'south'),
            // Pan west
            'ArrowLeft': panCommand(-panAmount, 0, 'west'),
            'a': panCommand(-panAmount, 0, 'west'),
            'A': panCommand(-panAmount, 0, 'west'),
            // Pan east
            'ArrowRight': panCommand(panAmount, 0, 'east'),
            'd': panCommand(panAmount, 0, 'east'),
            'D': panCommand(panAmount, 0, 'east'),
            // Zoom in
            '+': zoomCommand(zoomAmount),
            '=': zoomCommand(zoomAmount),
            // Zoom out
            '-': zoomCommand(-zoomAmount),
            '_': zoomCommand(-zoomAmount),
            // Reset view
            'r': { action: () => this.resetView(), announcement: 'View reset to default position' },
            'R': { action: () => this.resetView(), announcement: 'View reset to default position' },
            // Exit focus
            'Escape': { action: () => document.getElementById(this.containerId)?.blur(), announcement: 'Exited globe focus' },
            // Center view
            'Home': {
                action: () => this.map?.easeTo({ center: [0, 0], zoom: 2, duration: 500 }),
                announcement: 'Centered on Atlantic Ocean'
            }
        };
    }

    /**
     * Accessibility: Remove keyboard navigation handlers
     */
    private removeKeyboardNavigation(): void {
        const container = document.getElementById(this.containerId);
        if (!container) return;

        if (this.keyboardHandler) {
            container.removeEventListener('keydown', this.keyboardHandler);
            this.keyboardHandler = null;
        }
        if (this.focusHandler) {
            container.removeEventListener('focus', this.focusHandler);
            this.focusHandler = null;
        }
        if (this.blurHandler) {
            container.removeEventListener('blur', this.blurHandler);
            this.blurHandler = null;
        }
    }

    /**
     * Initialize the globe with MapLibre map instance
     */
    public initialize(): void {
        const container = document.getElementById(this.containerId);
        if (!container) {
            logger.error(`Container with id "${this.containerId}" not found`);
            return;
        }

        this.createMapInstance();
        this.addMapControls();
        this.setupMapLoadHandler();
    }

    /**
     * Create MapLibre map instance with configuration
     */
    private createMapInstance(): void {
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
    }

    /**
     * Add navigation controls to map
     */
    private addMapControls(): void {
        if (!this.map) return;

        this.map.addControl(
            new maplibregl.NavigationControl({
                visualizePitch: true,
                showCompass: true,
                showZoom: true,
            }),
            'top-right'
        );
    }

    /**
     * Setup handler for when map finishes loading
     */
    private setupMapLoadHandler(): void {
        if (!this.map) return;

        this.map.on('load', () => {
            this.onMapLoaded();
        });
    }

    /**
     * Handle map loaded event
     */
    private onMapLoaded(): void {
        if (!this.map) return;

        this.setupGlobeProjection();
        this.initializeTerrain();
        this.initializeDeckGLOverlay();
        this.setupMapEventHandlers();
        this.setupKeyboardNavigation();
        this.updateAccessibilityState();

        // Update with current data if any
        if (this.currentData.length > 0) {
            this.updateVisualization();
        }
    }

    /**
     * Setup globe projection if supported by MapLibre
     */
    private setupGlobeProjection(): void {
        if (!this.map) return;

        // Set globe projection if supported (must be done after style loads)
        if ('setProjection' in this.map) {
            (this.map as unknown as { setProjection: (p: { type: string }) => void }).setProjection({ type: 'globe' });
        }
    }

    /**
     * Initialize deck.gl overlay
     */
    private initializeDeckGLOverlay(): void {
        if (!this.map) return;

        // Initialize deck.gl overlay with shared WebGL context
        this.overlay = new MapboxOverlay({
            interleaved: true, // Share WebGL context with MapLibre
            layers: [],
        });

        // Add overlay to MapLibre map
        this.map.addControl(this.overlay);
    }

    /**
     * Setup map event handlers (zoom, etc.)
     */
    private setupMapEventHandlers(): void {
        if (!this.map) return;

        // LOD management: Update visualization when zoom level changes
        this.map.on('zoom', () => {
            if (this.currentData.length > 0) {
                this.dataChanged = true;
                this.layerCache.delete('scatterplot');
                this.updateVisualization();
            }
        });
    }

    /**
     * Update accessibility state after initialization
     */
    private updateAccessibilityState(): void {
        // Update aria-busy state to indicate loading complete
        const globeContainer = document.getElementById(this.containerId);
        if (globeContainer) {
            globeContainer.setAttribute('aria-busy', 'false');
        }

        // Hide loading indicator
        const loadingEl = document.getElementById('globe-loading');
        if (loadingEl) {
            loadingEl.style.display = 'none';
            loadingEl.setAttribute('aria-busy', 'false');
        }

        // Announce ready state to screen readers
        this.announceToScreenReader('Globe visualization ready. Press Tab to access controls or focus the globe to use keyboard navigation.');
    }

    /**
     * Initialize terrain if configured
     */
    private initializeTerrain(): void {
        if (!this.map) return;

        const config = MapConfigManager.getInstance().getConfig();
        const terrainConfig = getTerrainSource(config);

        if (!terrainConfig) return;

        // Terrain source is already configured in the style from createMapStyle
        // Just enable terrain rendering
        this.map.setTerrain({
            source: terrainConfig.source,
            exaggeration: terrainConfig.exaggeration,
        });
    }

    /**
     * Update globe with location data
     */
    public updateLocations(locations: LocationStats[]): void {
        // Filter out invalid locations (0,0 coordinates indicate geolocation failures)
        const validLocations = locations.filter(
            (location) => location.latitude !== 0 || location.longitude !== 0
        );

        this.dataChanged = true;
        this.currentData = validLocations;
        this.updateVisualization();

        // Accessibility: Update data summary for screen readers (Task 21)
        this.updateDataSummary();

        // Announce data update if there's data
        if (validLocations.length > 0) {
            const uniqueCountries = new Set(validLocations.map(loc => loc.country)).size;
            this.announceToScreenReader(
                `Globe updated with ${validLocations.length} locations across ${uniqueCountries} countries.`
            );
        }
    }

    /**
     * Update temporal data for animation
     */
    public updateTemporalData(buckets: TemporalHeatmapBucket[]): void {
        this.dataChanged = true;
        this.temporalData = buckets;

        if (this.enabledLayers.trips && buckets.length > 0) {
            this.updateVisualization();
        }
    }

    /**
     * Set server location for arc visualization
     */
    public setServerLocation(serverInfo: ServerInfo): void {
        if (serverInfo.has_location) {
            this.serverLocation = {
                latitude: serverInfo.latitude,
                longitude: serverInfo.longitude,
            };

            if (this.enabledLayers.arcs) {
                this.updateVisualization();
            }
        }
    }

    /**
     * Toggle a specific layer on/off
     */
    public toggleLayer(layerType: keyof LayerConfig, enabled: boolean): void {
        this.enabledLayers[layerType] = enabled;
        this.layerCache.delete(layerType);
        this.updateVisualization();
    }

    /**
     * Get current layer configuration
     */
    public getLayerConfig(): LayerConfig {
        return { ...this.enabledLayers };
    }

    /**
     * Update the deck.gl visualization with all enabled layers
     * Optimized to reuse existing layers when data hasn't changed
     */
    private updateVisualization(): void {
        if (!this.overlay || !this.map) {
            return;
        }

        const layers: Layer[] = [];

        // Layer configuration: each entry defines when a layer should be added
        // and how to create it. This table-driven approach reduces repetition.
        const layerConfigs: Array<{
            key: keyof LayerConfig;
            hasData: boolean;
            createLayer: () => Layer | null;
        }> = [
            {
                key: 'scatterplot',
                hasData: this.currentData.length > 0,
                createLayer: () => this.createScatterplotLayer()
            },
            {
                key: 'hexagon',
                hasData: this.currentData.length > 0,
                createLayer: () => this.createHexagonLayer()
            },
            {
                key: 'arcs',
                hasData: this.serverLocation !== null && this.currentData.length > 0,
                createLayer: () => this.createArcLayer()
            },
            {
                key: 'trips',
                hasData: this.temporalData.length > 0,
                createLayer: () => this.createTripsLayer()
            }
        ];

        // Process each layer configuration
        for (const config of layerConfigs) {
            if (this.enabledLayers[config.key] && config.hasData) {
                const layer = this.getOrCreateLayer(config.key, config.createLayer);
                if (layer) {
                    layers.push(layer);
                }
            }
        }

        // Mark data as unchanged after update
        this.dataChanged = false;

        // Update overlay with new layers
        this.overlay.setProps({
            layers: layers,
        });
    }

    /**
     * Get a cached layer or create a new one if data has changed
     */
    private getOrCreateLayer(key: string, createFn: () => Layer | null): Layer | null {
        const cachedLayer = this.layerCache.get(key);
        if (cachedLayer && !this.dataChanged) {
            return cachedLayer;
        }

        const newLayer = createFn();
        if (newLayer) {
            this.layerCache.set(key, newLayer);
        }
        return newLayer;
    }

    /**
     * Filter data based on zoom level for Level of Detail (LOD) management
     * Reduces points rendered at low zoom levels for better performance
     */
    private getFilteredDataByZoom(): LocationStats[] {
        if (!this.map) {
            return this.currentData;
        }

        const currentZoom = this.map.getZoom();
        const minPlaybackCount = this.getMinPlaybackCountForZoom(currentZoom);

        return this.currentData.filter((d) => d.playback_count > minPlaybackCount);
    }

    /**
     * Get minimum playback count threshold for a given zoom level
     * Higher zoom = more detail = lower threshold
     */
    private getMinPlaybackCountForZoom(zoom: number): number {
        const zoomThresholds = [
            { maxZoom: 4, minPlaybacks: 100 },  // World view: high-activity only
            { maxZoom: 6, minPlaybacks: 10 },   // Country view: medium+ activity
            { maxZoom: Infinity, minPlaybacks: 0 } // City+ view: all locations
        ];

        for (const threshold of zoomThresholds) {
            if (zoom < threshold.maxZoom) {
                return threshold.minPlaybacks;
            }
        }

        return 0; // Fallback: show all
    }

    /**
     * Create ScatterplotLayer for individual location points
     * Uses LOD filtering to improve performance at low zoom levels
     */
    private createScatterplotLayer(): ScatterplotLayer<LocationStats> {
        const filteredData = this.getFilteredDataByZoom();

        return new ScatterplotLayer<LocationStats>({
            id: 'playback-locations-scatter',
            data: filteredData,

            // Position accessor
            getPosition: (d: LocationStats) => [d.longitude, d.latitude, 0],

            // Size based on playback count
            getRadius: (d: LocationStats) => {
                const playbacks = d.playback_count;
                return Math.max(50000, Math.min(200000, Math.sqrt(playbacks) * 20000));
            },

            // Color based on playback count
            getFillColor: (d: LocationStats): Color => {
                return this.getColorByPlaybackCount(d.playback_count);
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
                    this.showTooltip(info, 'scatterplot');
                }
                return false;
            },

            onHover: (info: PickingInfo<LocationStats>) => {
                if (info.object) {
                    this.showTooltip(info, 'scatterplot');
                } else {
                    this.hideTooltip();
                }
            },

            // Smooth transitions
            transitions: {
                getPosition: 300,
                getRadius: 300,
                getFillColor: 300,
            },

            updateTriggers: {
                getFillColor: [this.isDarkTheme],
            },
        });
    }

    /**
     * Create HexagonLayer for 3D aggregated visualization
     */
    private createHexagonLayer(): HexagonLayer<LocationStats> {
        return new HexagonLayer<LocationStats>({
            id: 'playback-locations-hexagon',
            data: this.currentData,

            // Position and weight
            getPosition: (d: LocationStats) => [d.longitude, d.latitude],
            getElevationWeight: (d: LocationStats) => d.playback_count,
            getColorWeight: (d: LocationStats) => d.playback_count,

            // Hexagon configuration
            radius: 100000, // 100km radius per hexagon
            coverage: 0.9,
            extruded: true,
            elevationScale: 1000,

            // Color scheme - uses themed colors
            colorRange: [
                this.themeColors.markerLow.slice(0, 3) as [number, number, number], // Low
                this.themeColors.markerMedium.slice(0, 3) as [number, number, number], // Medium
                this.themeColors.markerMediumHigh.slice(0, 3) as [number, number, number], // High
                this.themeColors.markerHigh.slice(0, 3) as [number, number, number], // Very high
            ],

            // Material properties for realistic 3D look
            material: {
                ambient: 0.64,
                diffuse: 0.6,
                shininess: 32,
                specularColor: [51, 51, 51],
            },

            // Interactivity
            pickable: true,
            autoHighlight: true,

            onClick: (info) => {
                if (info.object) {
                    this.showHexagonTooltip(info);
                }
                return false;
            },

            onHover: (info) => {
                if (info.object) {
                    this.showHexagonTooltip(info);
                } else {
                    this.hideTooltip();
                }
                return false;
            },

            // Smooth transitions
            transitions: {
                elevationScale: {
                    duration: 500,
                    easing: (t: number) => t * (2 - t), // Ease out quad
                },
            },
        });
    }

    /**
     * Create ArcLayer for user→server connection visualization
     */
    private createArcLayer(): ArcLayer<LocationStats> | null {
        if (!this.serverLocation) return null;

        return new ArcLayer<LocationStats>({
            id: 'user-server-arcs',
            data: this.currentData,

            // Source (user location) → Target (server location)
            getSourcePosition: (d: LocationStats) => [d.longitude, d.latitude],
            getTargetPosition: () => [this.serverLocation!.longitude, this.serverLocation!.latitude],

            // Color scheme - uses themed colors
            getSourceColor: (): Color => this.themeColors.arcSource, // User location
            getTargetColor: (): Color => this.themeColors.arcTarget, // Server

            // Arc width based on playback count (simulating bandwidth)
            getWidth: (d: LocationStats) => Math.sqrt(d.playback_count) / 2,

            // Arc configuration
            greatCircle: true, // Follow Earth's curvature
            getHeight: 0.3, // Arc height (0 = flat, 1 = very curved)
            widthMinPixels: 1,
            widthMaxPixels: 8,

            // Interactivity
            pickable: true,
            autoHighlight: true,

            onClick: (info) => {
                if (info.object) {
                    this.showTooltip(info as PickingInfo<LocationStats>, 'arc');
                }
            },

            onHover: (info) => {
                if (info.object) {
                    this.showTooltip(info as PickingInfo<LocationStats>, 'arc');
                } else {
                    this.hideTooltip();
                }
            },

            // Smooth transitions
            transitions: {
                getSourcePosition: 300,
                getTargetPosition: 300,
                getWidth: 300,
            },
        });
    }

    /**
     * Create TripsLayer for temporal playback animation
     */
    private createTripsLayer(): TripsLayer<TripData> {
        const trips = this.convertTemporalDataToTrips();

        return new TripsLayer<TripData>({
            id: 'playback-trips',
            data: trips,

            getPath: (d: TripData) => d.path,
            getTimestamps: (d: TripData) => d.timestamps,
            getColor: (d: TripData): Color => this.getColorByTripWeight(d.weight),

            widthMinPixels: 2,
            widthMaxPixels: 10,
            opacity: 0.7,
            rounded: true,
            trailLength: 300000, // 5 minutes trail in milliseconds
            currentTime: this.animationTime,

            // Smooth animation
            fadeTrail: true,

            // Transitions
            transitions: {
                currentTime: {
                    duration: 100,
                },
            },
        });
    }

    /**
     * Convert temporal buckets to trips data for TripsLayer
     */
    private convertTemporalDataToTrips(): TripData[] {
        return this.temporalData.flatMap((bucket) => {
            return bucket.points.map((point) => ({
                path: [
                    [point.longitude, point.latitude, 0] as [number, number, number],
                    [point.longitude, point.latitude, point.weight * 1000] as [number, number, number],
                ],
                timestamps: [new Date(bucket.start_time).getTime(), new Date(bucket.end_time).getTime()],
                weight: point.weight,
            }));
        });
    }

    /**
     * Get color based on playback count
     * Uses CSS variable-based theme colors
     */
    private getColorByPlaybackCount(count: number): Color {
        return this.selectColorByThreshold(count, [
            { threshold: 500, color: this.themeColors.markerHigh },
            { threshold: 200, color: this.themeColors.markerMediumHigh },
            { threshold: 50, color: this.themeColors.markerMedium },
            { threshold: 0, color: this.themeColors.markerLow }
        ]);
    }

    /**
     * Get color for trip weight (used in TripsLayer)
     */
    private getColorByTripWeight(weight: number): [number, number, number] {
        const color = this.selectColorByThreshold(weight, [
            { threshold: 100, color: this.themeColors.markerHigh },
            { threshold: 50, color: this.themeColors.markerMediumHigh },
            { threshold: 10, color: this.themeColors.markerMedium },
            { threshold: 0, color: this.themeColors.markerLow }
        ]);
        return color.slice(0, 3) as [number, number, number];
    }

    /**
     * Select color based on threshold lookup table
     * Thresholds should be ordered from highest to lowest
     */
    private selectColorByThreshold(value: number, thresholds: Array<{ threshold: number; color: Color }>): Color {
        for (const entry of thresholds) {
            if (value > entry.threshold) {
                return entry.color;
            }
        }
        // Return last color as fallback (should match threshold: 0)
        return thresholds[thresholds.length - 1].color;
    }

    /**
     * Create tooltip element with CSS variable theming
     */
    private createTooltipElement(): void {
        this.tooltipElement = document.createElement('div');
        this.tooltipElement.id = 'deckgl-tooltip-enhanced';
        this.tooltipElement.className = 'globe-tooltip';
        this.tooltipElement.style.cssText = `
            position: absolute;
            z-index: 1000;
            pointer-events: none;
            background: var(--globe-tooltip-bg, rgba(20, 20, 30, 0.95));
            border: 1px solid var(--globe-tooltip-border, #14b8a6);
            border-radius: 4px;
            padding: 12px;
            color: var(--globe-tooltip-text, #eaeaea);
            font-size: 12px;
            display: none;
            max-width: 300px;
        `;
        document.body.appendChild(this.tooltipElement);
    }

    /**
     * Show tooltip for location with themed colors
     */
    private showTooltip(info: PickingInfo<LocationStats>, layerType: 'scatterplot' | 'arc'): void {
        if (!this.tooltipElement || !info.object) return;

        const html = this.generateLocationTooltipHTML(info.object, layerType);
        this.displayTooltip(html, info.x, info.y);
    }

    /**
     * Generate HTML for location tooltip
     */
    private generateLocationTooltipHTML(location: LocationStats, layerType: 'scatterplot' | 'arc'): string {
        const locationName = this.formatLocationName(location);
        const statsHTML = this.generateLocationStatsHTML(location);
        const layerInfoHTML = this.generateLayerInfoHTML(layerType);

        return `
            <div class="globe-tooltip-content" style="padding: 4px;">
                <div class="globe-tooltip-title" style="font-weight: 600; font-size: 14px; margin-bottom: 8px; color: var(--globe-tooltip-text, #fff);">
                    ${locationName}
                </div>
                <div class="globe-tooltip-stats" style="line-height: 1.8;">
                    ${statsHTML}
                </div>
                ${layerInfoHTML}
            </div>
        `;
    }

    /**
     * Format location name with HTML escaping
     */
    private formatLocationName(location: LocationStats): string {
        const escapeHtml = (text: string): string => {
            const div = document.createElement('div');
            div.textContent = text;
            return div.innerHTML;
        };

        const city = escapeHtml(location.city || 'Unknown');
        const region = location.region ? escapeHtml(location.region) : null;
        const country = escapeHtml(location.country);

        return region ? `${city}, ${region}, ${country}` : `${city}, ${country}`;
    }

    /**
     * Generate stats section HTML for location tooltip
     */
    private generateLocationStatsHTML(location: LocationStats): string {
        const textSecondary = 'var(--text-secondary, #a0a0a0)';
        const stats: string[] = [];

        stats.push(`<div><span style="color: ${textSecondary};">Playbacks:</span> <strong>${location.playback_count.toLocaleString()}</strong></div>`);
        stats.push(`<div><span style="color: ${textSecondary};">Unique Users:</span> <strong>${location.unique_users}</strong></div>`);
        stats.push(`<div><span style="color: ${textSecondary};">Avg Completion:</span> <strong>${location.avg_completion.toFixed(1)}%</strong></div>`);

        if (location.first_seen) {
            const date = new Date(location.first_seen).toLocaleDateString();
            stats.push(`<div style="margin-top: 4px; padding-top: 4px; border-top: 1px solid rgba(255,255,255,0.1);"><span style="color: ${textSecondary};">First Seen:</span> ${date}</div>`);
        }

        if (location.last_seen) {
            const date = new Date(location.last_seen).toLocaleDateString();
            stats.push(`<div><span style="color: ${textSecondary};">Last Seen:</span> ${date}</div>`);
        }

        return stats.join('');
    }

    /**
     * Generate layer info HTML for arc visualization
     */
    private generateLayerInfoHTML(layerType: 'scatterplot' | 'arc'): string {
        if (layerType !== 'arc' || !this.serverLocation) {
            return '';
        }

        const markerLow = 'var(--globe-marker-low, #14b8a6)';
        const markerHigh = 'var(--globe-marker-high, #ec4899)';

        return `<div style="margin-top: 8px; padding-top: 8px; border-top: 1px solid rgba(255,255,255,0.1);"><span style="color: ${markerLow};">● User Location</span> → <span style="color: ${markerHigh};">● Server</span></div>`;
    }

    /**
     * Display tooltip at given coordinates
     */
    private displayTooltip(html: string, x: number, y: number): void {
        if (!this.tooltipElement) return;

        this.tooltipElement.innerHTML = html;
        this.tooltipElement.style.display = 'block';
        this.tooltipElement.style.left = `${x}px`;
        this.tooltipElement.style.top = `${y}px`;
    }

    /**
     * Show tooltip for hexagon aggregation with themed colors
     */
    private showHexagonTooltip(info: PickingInfo<unknown>): void {
        if (!this.tooltipElement || !info.object) return;

        const hexData = info.object as { elevationValue?: number; colorValue?: number; count?: number };
        const html = this.generateHexagonTooltipHTML(hexData);
        this.displayTooltip(html, info.x, info.y);
    }

    /**
     * Generate HTML for hexagon tooltip
     */
    private generateHexagonTooltipHTML(hexData: { elevationValue?: number; colorValue?: number; count?: number }): string {
        const totalPlaybacks = hexData.elevationValue || hexData.colorValue || 0;
        const count = hexData.count || 0;
        const avgPerLocation = count > 0 ? Math.round(totalPlaybacks / count) : 0;

        const textSecondary = 'var(--text-secondary, #a0a0a0)';

        return `
            <div class="globe-tooltip-content" style="padding: 4px;">
                <div class="globe-tooltip-title" style="font-weight: 600; font-size: 14px; margin-bottom: 8px; color: var(--globe-tooltip-text, #fff);">
                    Regional Aggregate
                </div>
                <div class="globe-tooltip-stats" style="line-height: 1.8;">
                    <div><span style="color: ${textSecondary};">Total Playbacks:</span> <strong>${Math.round(totalPlaybacks).toLocaleString()}</strong></div>
                    <div><span style="color: ${textSecondary};">Locations:</span> <strong>${count}</strong></div>
                    <div><span style="color: ${textSecondary};">Avg per Location:</span> <strong>${avgPerLocation.toLocaleString()}</strong></div>
                </div>
            </div>
        `;
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
     * Start temporal animation (Task 28: fix edge cases)
     * Sets animation state flag - timing is controlled by globe-controls-enhanced.ts
     * This prevents the double animation loop race condition.
     */
    public startAnimation(): void {
        if (this.temporalData.length === 0) return;

        // Cancel any pending animation frame from previous runs (prevent memory leak)
        if (this.animationFrameId !== null) {
            cancelAnimationFrame(this.animationFrameId);
            this.animationFrameId = null;
        }

        this.isAnimating = true;

        // Initialize animation time if not already set
        if (this.animationTime === 0 && this.temporalData.length > 0) {
            this.animationTime = new Date(this.temporalData[0].start_time).getTime();
        }

        // Update visualization immediately
        this.updateVisualization();
    }

    /**
     * Stop temporal animation (Task 28: fix edge cases)
     * Safely cancels animation and clears state.
     */
    public stopAnimation(): void {
        this.isAnimating = false;

        // Cancel any pending animation frame
        if (this.animationFrameId !== null) {
            cancelAnimationFrame(this.animationFrameId);
            this.animationFrameId = null;
        }
    }

    /**
     * Check if animation is currently running
     */
    public getIsAnimating(): boolean {
        return this.isAnimating;
    }

    /**
     * Get current animation time
     */
    public getAnimationTime(): number {
        return this.animationTime;
    }

    /**
     * Get animation time range (Task 28: bounds validation)
     */
    public getAnimationTimeRange(): { start: number; end: number } | null {
        if (this.temporalData.length === 0) return null;

        const startTime = new Date(this.temporalData[0].start_time).getTime();
        const endTime = new Date(this.temporalData[this.temporalData.length - 1].end_time).getTime();

        return { start: startTime, end: endTime };
    }

    /**
     * Set animation time manually (for timeline scrubbing) (Task 28: bounds validation)
     * @param timestamp - Unix timestamp in milliseconds
     * @param updateVisualization - Whether to update the visualization (default: true)
     */
    public setAnimationTime(timestamp: number, updateVisualization: boolean = true): void {
        // Validate timestamp against temporal data range (Task 28)
        const range = this.getAnimationTimeRange();
        if (range) {
            // Clamp timestamp to valid range
            this.animationTime = Math.max(range.start, Math.min(range.end, timestamp));
        } else {
            this.animationTime = timestamp;
        }

        if (updateVisualization) {
            this.updateVisualization();
        }
    }

    /**
     * Update theme - refreshes cached colors and map style
     * Supports dark, light, and high-contrast themes
     * @param themeOrIsDark - Either a theme string ('dark' | 'light' | 'high-contrast') or boolean for backwards compat
     */
    public updateTheme(themeOrIsDark: boolean | 'dark' | 'light' | 'high-contrast'): void {
        const theme = this.normalizeTheme(themeOrIsDark);
        this.isDarkTheme = theme === 'dark' || theme === 'high-contrast';

        // Refresh cached theme colors from CSS variables
        // Wait a tick for CSS variables to update after theme change
        requestAnimationFrame(() => {
            this.themeColors = getGlobeThemeColors();

            // Clear layer cache to force re-creation with new colors
            this.layerCache.clear();
            this.dataChanged = true;

            // Update map style to support high-contrast tiles
            if (this.map) {
                this.updateMapStyleForTheme(theme);
            }

            this.updateVisualization();
        });
    }

    /**
     * Normalize theme parameter for backwards compatibility
     */
    private normalizeTheme(themeOrIsDark: boolean | 'dark' | 'light' | 'high-contrast'): 'dark' | 'light' | 'high-contrast' {
        if (typeof themeOrIsDark === 'boolean') {
            return themeOrIsDark ? 'dark' : 'light';
        }
        return themeOrIsDark;
    }

    /**
     * Update map style based on theme
     */
    private updateMapStyleForTheme(theme: 'dark' | 'light' | 'high-contrast'): void {
        const configManager = MapConfigManager.getInstance();
        const config = configManager.getConfig();

        const tileProvider = this.getTileProviderForTheme(theme);
        configManager.setConfig({ ...config, tileProvider, theme });

        const newStyle = createMapStyle(configManager.getConfig());
        this.map?.setStyle(newStyle);
    }

    /**
     * Get tile provider for a given theme
     */
    private getTileProviderForTheme(theme: 'dark' | 'light' | 'high-contrast'): 'carto-dark' | 'carto-light' | 'high-contrast-dark' {
        const themeToProviderMap: Record<'dark' | 'light' | 'high-contrast', 'carto-dark' | 'carto-light' | 'high-contrast-dark'> = {
            'high-contrast': 'high-contrast-dark',
            'light': 'carto-light',
            'dark': 'carto-dark'
        };

        return themeToProviderMap[theme];
    }

    /**
     * Reset camera view
     */
    public resetView(): void {
        if (!this.map) return;

        this.map.easeTo({
            center: [0, 20],
            zoom: 2,
            pitch: 0,
            bearing: 0,
            duration: 1000,
        });
    }

    /**
     * Destroy the globe instance
     */
    public destroy(): void {
        this.stopAnimation();
        this.hideTooltip();
        this.removeKeyboardNavigation();
        this.cleanupTooltipElement();
        this.cleanupMapAndOverlay();
    }

    /**
     * Cleanup tooltip element from DOM
     */
    private cleanupTooltipElement(): void {
        if (this.tooltipElement && this.tooltipElement.parentNode) {
            this.tooltipElement.parentNode.removeChild(this.tooltipElement);
            this.tooltipElement = null;
        }
    }

    /**
     * Cleanup map and deck.gl overlay
     */
    private cleanupMapAndOverlay(): void {
        if (this.map && this.overlay) {
            this.map.removeControl(this.overlay);
        }
        if (this.map) {
            this.map.remove();
        }
        this.map = null;
        this.overlay = null;
    }

    /**
     * Show the globe (resize map to trigger deck.gl update)
     */
    public show(): void {
        if (!this.map) return;

        setTimeout(() => {
            if (this.map) {
                this.map.resize();
            }
        }, 100);
    }

    /**
     * Hide the globe
     */
    public hide(): void {
        this.hideTooltip();
        this.stopAnimation();
    }

    /**
     * Get the MapLibre map instance for external access
     */
    public getMap(): MapLibreMap | null {
        return this.map;
    }
}
