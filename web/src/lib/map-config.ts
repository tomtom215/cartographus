// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Map Configuration System
 *
 * Centralized configuration for MapLibre GL JS map instances.
 * Supports multiple tile providers with sensible defaults that require no API keys.
 *
 * Architecture:
 * - Default: Free CartoDB raster tiles (no configuration required)
 * - Optional: PMTiles for self-hosted vector tiles
 * - Optional: Custom tile server URL
 * - Optional: Terrain from various providers
 *
 * @module map-config
 */

import type { StyleSpecification, SourceSpecification } from 'maplibre-gl';
import { createLogger } from './logger';
import { SafeStorage } from './utils/SafeStorage';

const logger = createLogger('MapConfig');

/**
 * Available tile providers
 */
export type TileProvider =
    | 'carto-dark'              // Free, no API key (default)
    | 'carto-light'             // Free, no API key
    | 'carto-voyager'           // Free, no API key (color basemap)
    | 'carto-dark-nolabels'     // Free, no API key (minimal, no labels)
    | 'carto-light-nolabels'    // Free, no API key (minimal, no labels)
    | 'high-contrast-dark'      // High contrast dark mode
    | 'high-contrast-light'     // High contrast light mode
    | 'osm'                     // OpenStreetMap raster tiles
    | 'pmtiles'                 // Self-hosted PMTiles vector tiles
    | 'custom';                 // User-provided tile URL

/**
 * Available terrain providers
 */
export type TerrainProvider =
    | 'none'            // No terrain (default)
    | 'aws'             // AWS Terrain Tiles (free, public)
    | 'maptiler'        // MapTiler (requires API key)
    | 'custom';         // User-provided terrain URL

/**
 * Map configuration options
 */
export interface MapConfig {
    /** Tile provider for basemap */
    tileProvider: TileProvider;

    /** Custom tile URL template (for 'custom' or 'pmtiles' providers) */
    customTileUrl?: string;

    /** Terrain provider */
    terrainProvider: TerrainProvider;

    /** Custom terrain URL (for 'custom' terrain provider) */
    customTerrainUrl?: string;

    /** MapTiler API key (required for maptiler terrain) */
    maptilerApiKey?: string;

    /** Terrain exaggeration factor (default: 1.5) */
    terrainExaggeration: number;

    /** Enable 3D buildings (requires vector tiles) */
    enable3DBuildings: boolean;

    /** Default theme - includes high-contrast for accessibility */
    theme: 'dark' | 'light' | 'high-contrast';
}

/**
 * Default configuration - works out of the box with no API keys
 */
export const DEFAULT_MAP_CONFIG: MapConfig = {
    tileProvider: 'carto-dark',
    terrainProvider: 'none',
    terrainExaggeration: 1.5,
    enable3DBuildings: false,
    theme: 'dark',
};

/**
 * CartoDB tile URLs - free, no API key required
 */
const CARTO_TILES: Record<string, string[]> = {
    'carto-dark': [
        'https://a.basemaps.cartocdn.com/dark_all/{z}/{x}/{y}@2x.png',
        'https://b.basemaps.cartocdn.com/dark_all/{z}/{x}/{y}@2x.png',
        'https://c.basemaps.cartocdn.com/dark_all/{z}/{x}/{y}@2x.png',
        'https://d.basemaps.cartocdn.com/dark_all/{z}/{x}/{y}@2x.png',
    ],
    'carto-light': [
        'https://a.basemaps.cartocdn.com/light_all/{z}/{x}/{y}@2x.png',
        'https://b.basemaps.cartocdn.com/light_all/{z}/{x}/{y}@2x.png',
        'https://c.basemaps.cartocdn.com/light_all/{z}/{x}/{y}@2x.png',
        'https://d.basemaps.cartocdn.com/light_all/{z}/{x}/{y}@2x.png',
    ],
    'carto-voyager': [
        'https://a.basemaps.cartocdn.com/rastertiles/voyager/{z}/{x}/{y}@2x.png',
        'https://b.basemaps.cartocdn.com/rastertiles/voyager/{z}/{x}/{y}@2x.png',
        'https://c.basemaps.cartocdn.com/rastertiles/voyager/{z}/{x}/{y}@2x.png',
        'https://d.basemaps.cartocdn.com/rastertiles/voyager/{z}/{x}/{y}@2x.png',
    ],
    // Minimal basemaps without labels - better for high contrast overlays
    'carto-dark-nolabels': [
        'https://a.basemaps.cartocdn.com/dark_nolabels/{z}/{x}/{y}@2x.png',
        'https://b.basemaps.cartocdn.com/dark_nolabels/{z}/{x}/{y}@2x.png',
        'https://c.basemaps.cartocdn.com/dark_nolabels/{z}/{x}/{y}@2x.png',
        'https://d.basemaps.cartocdn.com/dark_nolabels/{z}/{x}/{y}@2x.png',
    ],
    'carto-light-nolabels': [
        'https://a.basemaps.cartocdn.com/light_nolabels/{z}/{x}/{y}@2x.png',
        'https://b.basemaps.cartocdn.com/light_nolabels/{z}/{x}/{y}@2x.png',
        'https://c.basemaps.cartocdn.com/light_nolabels/{z}/{x}/{y}@2x.png',
        'https://d.basemaps.cartocdn.com/light_nolabels/{z}/{x}/{y}@2x.png',
    ],
    // High contrast mode uses simplified basemaps for maximum data visibility
    'high-contrast-dark': [
        'https://a.basemaps.cartocdn.com/dark_nolabels/{z}/{x}/{y}@2x.png',
        'https://b.basemaps.cartocdn.com/dark_nolabels/{z}/{x}/{y}@2x.png',
        'https://c.basemaps.cartocdn.com/dark_nolabels/{z}/{x}/{y}@2x.png',
        'https://d.basemaps.cartocdn.com/dark_nolabels/{z}/{x}/{y}@2x.png',
    ],
    'high-contrast-light': [
        'https://a.basemaps.cartocdn.com/light_nolabels/{z}/{x}/{y}@2x.png',
        'https://b.basemaps.cartocdn.com/light_nolabels/{z}/{x}/{y}@2x.png',
        'https://c.basemaps.cartocdn.com/light_nolabels/{z}/{x}/{y}@2x.png',
        'https://d.basemaps.cartocdn.com/light_nolabels/{z}/{x}/{y}@2x.png',
    ],
};

/**
 * OpenStreetMap tile URLs - free, follows usage policy
 */
const OSM_TILES = [
    'https://a.tile.openstreetmap.org/{z}/{x}/{y}.png',
    'https://b.tile.openstreetmap.org/{z}/{x}/{y}.png',
    'https://c.tile.openstreetmap.org/{z}/{x}/{y}.png',
];

/**
 * AWS Terrain Tiles - free, public dataset
 * https://registry.opendata.aws/terrain-tiles/
 */
const AWS_TERRAIN_URL = 'https://s3.amazonaws.com/elevation-tiles-prod/terrarium/{z}/{x}/{y}.png';

/**
 * Get attribution text for the current tile provider
 */
export function getAttribution(config: MapConfig): string {
    switch (config.tileProvider) {
        case 'carto-dark':
        case 'carto-light':
        case 'carto-voyager':
        case 'carto-dark-nolabels':
        case 'carto-light-nolabels':
        case 'high-contrast-dark':
        case 'high-contrast-light':
            return '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors &copy; <a href="https://carto.com/attributions">CARTO</a>';
        case 'osm':
            return '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors';
        case 'pmtiles':
            return '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors &copy; <a href="https://protomaps.com">Protomaps</a>';
        case 'custom':
            return config.customTileUrl?.includes('openstreetmap')
                ? '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors'
                : '';
        default:
            return '';
    }
}

/**
 * Build the basemap source specification
 */
function buildBasemapSource(config: MapConfig): Record<string, SourceSpecification> {
    const sources: Record<string, SourceSpecification> = {};

    switch (config.tileProvider) {
        case 'carto-dark':
        case 'carto-light':
        case 'carto-voyager':
        case 'carto-dark-nolabels':
        case 'carto-light-nolabels':
        case 'high-contrast-dark':
        case 'high-contrast-light':
            sources['basemap'] = {
                type: 'raster',
                tiles: CARTO_TILES[config.tileProvider],
                tileSize: 256,
                attribution: getAttribution(config),
            };
            break;

        case 'osm':
            sources['basemap'] = {
                type: 'raster',
                tiles: OSM_TILES,
                tileSize: 256,
                attribution: getAttribution(config),
            };
            break;

        case 'pmtiles':
            if (config.customTileUrl) {
                sources['basemap'] = {
                    type: 'vector',
                    url: `pmtiles://${config.customTileUrl}`,
                    attribution: getAttribution(config),
                };
            }
            break;

        case 'custom':
            if (config.customTileUrl) {
                // Detect if it's a vector or raster source
                const isVector = config.customTileUrl.includes('.pbf') ||
                                 config.customTileUrl.includes('pmtiles') ||
                                 config.customTileUrl.includes('.mvt');

                if (isVector) {
                    sources['basemap'] = {
                        type: 'vector',
                        tiles: [config.customTileUrl],
                        attribution: getAttribution(config),
                    };
                } else {
                    sources['basemap'] = {
                        type: 'raster',
                        tiles: [config.customTileUrl],
                        tileSize: 256,
                        attribution: getAttribution(config),
                    };
                }
            }
            break;
    }

    return sources;
}

/**
 * Build terrain source specification
 */
function buildTerrainSource(config: MapConfig): Record<string, SourceSpecification> {
    const sources: Record<string, SourceSpecification> = {};

    switch (config.terrainProvider) {
        case 'aws':
            sources['terrain'] = {
                type: 'raster-dem',
                tiles: [AWS_TERRAIN_URL],
                tileSize: 256,
                encoding: 'terrarium',
                maxzoom: 15,
            };
            break;

        case 'maptiler':
            if (config.maptilerApiKey) {
                sources['terrain'] = {
                    type: 'raster-dem',
                    url: `https://api.maptiler.com/tiles/terrain-rgb-v2/tiles.json?key=${config.maptilerApiKey}`,
                    tileSize: 256,
                };
            }
            break;

        case 'custom':
            if (config.customTerrainUrl) {
                sources['terrain'] = {
                    type: 'raster-dem',
                    tiles: [config.customTerrainUrl],
                    tileSize: 256,
                    encoding: 'terrarium',
                };
            }
            break;
    }

    // Add hillshade source if terrain is enabled
    if (config.terrainProvider !== 'none' && sources['terrain']) {
        sources['hillshade'] = sources['terrain'];
    }

    return sources;
}

/**
 * Build basemap layers based on provider type
 * Supports high-contrast mode with adjusted brightness/contrast
 */
function buildBasemapLayers(config: MapConfig): StyleSpecification['layers'] {
    const layers: StyleSpecification['layers'] = [];
    const isHighContrast = config.theme === 'high-contrast';

    // For raster tiles, add a simple raster layer
    if (config.tileProvider !== 'pmtiles' &&
        (config.tileProvider !== 'custom' || !config.customTileUrl?.includes('.pbf'))) {

        // High-contrast mode adjusts raster brightness/contrast
        const paint: Record<string, number> = {};
        if (isHighContrast) {
            // Darken the basemap to make overlays more visible
            paint['raster-brightness-min'] = 0;
            paint['raster-brightness-max'] = 0.6; // Reduce max brightness
            paint['raster-contrast'] = 0.3; // Increase contrast
            paint['raster-saturation'] = -0.5; // Reduce saturation for less distraction
        }

        layers.push({
            id: 'basemap-layer',
            type: 'raster',
            source: 'basemap',
            minzoom: 0,
            maxzoom: 22,
            ...(Object.keys(paint).length > 0 ? { paint } : {})
        });
    }

    // Add hillshade layer if terrain is enabled
    if (config.terrainProvider !== 'none') {
        // High-contrast mode uses more extreme hillshade colors
        const shadowColor = isHighContrast
            ? '#000000'
            : (config.theme === 'dark' ? '#000000' : '#333333');
        const highlightColor = isHighContrast
            ? '#ffffff'
            : (config.theme === 'dark' ? '#333333' : '#ffffff');

        layers.push({
            id: 'hillshade-layer',
            type: 'hillshade',
            source: 'hillshade',
            paint: {
                'hillshade-exaggeration': isHighContrast ? 0.8 : 0.5,
                'hillshade-shadow-color': shadowColor,
                'hillshade-highlight-color': highlightColor,
            },
        });
    }

    return layers;
}

/**
 * Build a complete MapLibre style specification from configuration
 */
export function buildMapStyle(config: Partial<MapConfig> = {}): StyleSpecification {
    const mergedConfig: MapConfig = { ...DEFAULT_MAP_CONFIG, ...config };

    const sources: Record<string, SourceSpecification> = {
        ...buildBasemapSource(mergedConfig),
        ...buildTerrainSource(mergedConfig),
    };

    const layers = buildBasemapLayers(mergedConfig);

    const style: StyleSpecification = {
        version: 8,
        sources,
        layers,
    };

    // Add terrain configuration if enabled
    if (mergedConfig.terrainProvider !== 'none' && sources['terrain']) {
        style.terrain = {
            source: 'terrain',
            exaggeration: mergedConfig.terrainExaggeration,
        };
    }

    // Add sky for 3D effect with terrain (with high-contrast support)
    if (mergedConfig.terrainProvider !== 'none') {
        const isHighContrast = mergedConfig.theme === 'high-contrast';
        const isDark = mergedConfig.theme === 'dark' || isHighContrast;

        style.sky = {
            'sky-color': isHighContrast ? '#000000' : (isDark ? '#0a0a1a' : '#87ceeb'),
            'horizon-color': isHighContrast ? '#000000' : (isDark ? '#16213e' : '#ffffff'),
            'fog-color': isHighContrast ? '#000000' : (isDark ? '#0a0a1a' : '#ffffff'),
            'sky-horizon-blend': isHighContrast ? 0 : 0.5,
            'horizon-fog-blend': isHighContrast ? 0 : 0.5,
        };
    }

    return style;
}

/**
 * Get terrain configuration for map instance
 * Call this after map loads to enable terrain
 */
export function getTerrainConfig(config: Partial<MapConfig> = {}): { source: string; exaggeration: number } | null {
    const mergedConfig: MapConfig = { ...DEFAULT_MAP_CONFIG, ...config };

    if (mergedConfig.terrainProvider === 'none') {
        return null;
    }

    return {
        source: 'terrain',
        exaggeration: mergedConfig.terrainExaggeration,
    };
}

/**
 * Load and apply PMTiles protocol handler
 * Must be called before creating maps that use PMTiles
 */
export async function initializePMTiles(): Promise<void> {
    try {
        const { Protocol } = await import('pmtiles');
        const maplibregl = await import('maplibre-gl');

        const protocol = new Protocol();
        maplibregl.addProtocol('pmtiles', protocol.tile);

        logger.debug('PMTiles protocol registered');
    } catch (error) {
        logger.warn('PMTiles not available:', error);
    }
}

/**
 * Configuration manager singleton
 * Allows runtime configuration updates
 */
export class MapConfigManager {
    private static instance: MapConfigManager;
    private config: MapConfig = { ...DEFAULT_MAP_CONFIG };
    private listeners: Set<(config: MapConfig) => void> = new Set();

    /**
     * Get the singleton instance
     */
    static getInstance(): MapConfigManager {
        if (!MapConfigManager.instance) {
            MapConfigManager.instance = new MapConfigManager();
        }
        return MapConfigManager.instance;
    }

    /**
     * Get current configuration
     */
    getConfig(): MapConfig {
        return { ...this.config };
    }

    /**
     * Update configuration
     */
    setConfig(updates: Partial<MapConfig>): void {
        this.config = { ...this.config, ...updates };
        this.notifyListeners();
    }

    /**
     * Subscribe to configuration changes
     */
    subscribe(listener: (config: MapConfig) => void): () => void {
        this.listeners.add(listener);
        return () => this.listeners.delete(listener);
    }

    private notifyListeners(): void {
        this.listeners.forEach(listener => listener(this.config));
    }

    /**
     * Load configuration from localStorage
     */
    loadFromStorage(): void {
        const stored = SafeStorage.getItem('map-config');
        if (stored) {
            try {
                const parsed = JSON.parse(stored);
                this.config = { ...DEFAULT_MAP_CONFIG, ...parsed };
            } catch (error) {
                logger.warn('Failed to parse map config:', error);
            }
        }
    }

    /**
     * Save configuration to localStorage
     */
    saveToStorage(): void {
        SafeStorage.setJSON('map-config', this.config);
    }

    /**
     * Cleanup resources to prevent memory leaks
     * Clears listeners and resets configuration
     */
    destroy(): void {
        this.listeners.clear();
        this.config = { ...DEFAULT_MAP_CONFIG };
    }
}

/**
 * Global map configuration manager
 */
export const mapConfigManager = new MapConfigManager();

/**
 * Convenience function to get a dark theme style
 */
export function getDarkStyle(overrides: Partial<MapConfig> = {}): StyleSpecification {
    return buildMapStyle({ ...overrides, theme: 'dark', tileProvider: 'carto-dark' });
}

/**
 * Convenience function to get a light theme style
 */
export function getLightStyle(overrides: Partial<MapConfig> = {}): StyleSpecification {
    return buildMapStyle({ ...overrides, theme: 'light', tileProvider: 'carto-light' });
}

/**
 * Convenience function to get a high-contrast theme style
 * Uses minimal basemap with adjusted brightness/contrast for accessibility
 */
export function getHighContrastStyle(overrides: Partial<MapConfig> = {}): StyleSpecification {
    return buildMapStyle({
        ...overrides,
        theme: 'high-contrast',
        tileProvider: 'high-contrast-dark'
    });
}

/**
 * Check if terrain is supported in the current browser
 * Properly cleans up WebGL context to prevent resource leaks
 */
export function isTerrainSupported(): boolean {
    let canvas: HTMLCanvasElement | null = null;
    let gl: WebGLRenderingContext | WebGL2RenderingContext | null = null;

    try {
        canvas = document.createElement('canvas');
        // Use small canvas to minimize memory allocation
        canvas.width = 1;
        canvas.height = 1;

        gl = canvas.getContext('webgl2') || canvas.getContext('webgl');
        if (!gl) return false;

        // Check for required extensions
        const ext = gl.getExtension('OES_texture_float');
        return ext !== null;
    } catch {
        return false;
    } finally {
        // Properly release WebGL context to prevent memory leak
        if (gl) {
            const loseContext = gl.getExtension('WEBGL_lose_context');
            if (loseContext) {
                loseContext.loseContext();
            }
        }
        // Release canvas memory
        if (canvas) {
            canvas.width = 0;
            canvas.height = 0;
        }
    }
}

// Aliases for backward compatibility
export const createMapStyle = buildMapStyle;
export const getTerrainSource = getTerrainConfig;
