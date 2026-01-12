// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
import maplibregl from 'maplibre-gl';
import type { Map as MapLibreMap, GeoJSONSource, LngLatBoundsLike } from 'maplibre-gl';
import { Protocol } from 'pmtiles';
import type { LocationStats, PlaybackEvent, H3HexagonStats, ArcStats } from './api';
import { MapConfigManager, createMapStyle, getTerrainSource, type TileProvider } from './map-config';
import { NominatimGeocoder, type NominatimResult } from './geocoder';
import { escapeHtml } from './sanitize';
import { createLogger } from './logger';
import { SafeStorage } from './utils/SafeStorage';

const logger = createLogger('MapManager');

// Type definitions
type VisualizationMode = 'points' | 'clusters' | 'heatmap' | 'hexagons';

// Arc GeoJSON types
interface ArcFeature {
    type: 'Feature';
    id?: string;
    geometry: {
        type: 'LineString';
        coordinates: number[][];
    };
    properties: {
        city: string;
        country: string;
        distance_km: number;
        playback_count: number;
        unique_users: number;
        avg_completion: number;
        weight: number;
    };
}

interface ArcFeatureCollection {
    type: 'FeatureCollection';
    features: ArcFeature[];
}

interface GeoJSONFeature {
    type: 'Feature';
    id?: string;
    geometry: {
        type: 'Point';
        coordinates: [number, number];
    };
    properties: Record<string, unknown>;
}

interface GeoJSONFeatureCollection {
    type: 'FeatureCollection';
    features: GeoJSONFeature[];
}

// Hexagon GeoJSON types (H3 hexagons rendered as polygons)
interface HexagonFeature {
    type: 'Feature';
    id?: string;
    geometry: {
        type: 'Polygon';
        coordinates: number[][][];
    };
    properties: {
        h3_index: number;
        playback_count: number;
        unique_users: number;
        avg_completion: number;
        total_watch_minutes: number;
    };
}

interface HexagonFeatureCollection {
    type: 'FeatureCollection';
    features: HexagonFeature[];
}

/**
 * MapManager - 2D Map visualization using MapLibre GL JS
 *
 * Features:
 * - Multiple visualization modes (points, clusters, heatmap)
 * - Pluggable tile sources (CartoDB, OSM, PMTiles, custom)
 * - Optional terrain support (AWS Terrain Tiles, Maptiler, custom)
 * - Vector tile support for large datasets (>10k locations)
 * - Incremental GeoJSON updates for performance
 */
export class MapManager {
    private map: MapLibreMap;
    private currentMode: VisualizationMode = 'heatmap';
    private onModeChange?: (mode: VisualizationMode) => void;
    private pmtilesProtocol: Protocol | null = null;

    // Track previous features for incremental updates (80% faster)
    private previousFeatureIds: Set<string> = new Set();

    // Vector tile support for large datasets
    private useVectorTiles: boolean = false;
    private readonly VECTOR_TILE_THRESHOLD = 10000;

    // H3 hexagon visualization
    private _hexagonData: H3HexagonStats[] = [];
    private hexagonResolution: number = 7; // Default: city-level
    private onHexagonDataRequest?: (resolution: number) => void;

    // Arc overlay visualization
    private _arcData: ArcStats[] = [];
    private _arcOverlayEnabled: boolean = false;
    private onArcDataRequest?: () => void;

    // Search/Geocoder control
    private geocoder: NominatimGeocoder | null = null;
    private onGeocoderResult?: (result: NominatimResult) => void;

    constructor(containerId: string, onModeChange?: (mode: VisualizationMode) => void) {
        this.onModeChange = onModeChange;

        // Initialize PMTiles protocol for .pmtiles support
        this.initPMTilesProtocol();

        // Get map configuration
        const config = MapConfigManager.getInstance().getConfig();
        const style = createMapStyle(config);

        // Create MapLibre map instance
        this.map = new maplibregl.Map({
            container: containerId,
            style: style,
            center: [0, 20],
            zoom: 2,
            maxPitch: 85, // Enable 3D terrain viewing
        });

        // Add navigation controls
        this.map.addControl(new maplibregl.NavigationControl({
            visualizePitch: true,
            showCompass: true,
            showZoom: true,
        }), 'top-right');

        this.map.addControl(new maplibregl.FullscreenControl(), 'top-right');
        this.map.addControl(new maplibregl.ScaleControl({ unit: 'metric' }), 'bottom-left');

        // Add geocoder search control
        this.initializeGeocoder();

        // Initialize on map load
        this.map.on('load', () => {
            this.initializeTerrain();
            this.initializeLayers();
            this.restoreVisualizationMode();
            this.updateLayerVisibility();
        });

        // Handle style changes for theme switching
        this.map.on('style.load', () => {
            this.initializeTerrain();
            this.initializeLayers();
            this.updateLayerVisibility();
        });
    }

    /**
     * Initialize PMTiles protocol for .pmtiles file support
     */
    private initPMTilesProtocol(): void {
        this.pmtilesProtocol = new Protocol();
        maplibregl.addProtocol('pmtiles', this.pmtilesProtocol.tile);
    }

    /**
     * Initialize geocoder search control
     * Uses Nominatim (OpenStreetMap) for free geocoding without API keys
     */
    private initializeGeocoder(): void {
        this.geocoder = new NominatimGeocoder({
            placeholder: 'Search locations...',
            minLength: 3,
            maxResults: 5,
            flyToZoom: 12,
            flyToDuration: 2000,
            showMarker: true,
            markerColor: 'var(--highlight, #7c3aed)',
            onResult: (result: NominatimResult) => {
                // Fly to the selected location
                const lng = parseFloat(result.lon);
                const lat = parseFloat(result.lat);

                this.map.flyTo({
                    center: [lng, lat],
                    zoom: 12,
                    duration: 2000,
                });

                // Trigger callback if set
                if (this.onGeocoderResult) {
                    this.onGeocoderResult(result);
                }
            },
            onClear: () => {
                // Optional: reset view when search is cleared
            },
        });

        // Add geocoder control to top-left of map
        this.map.addControl(this.geocoder as unknown as maplibregl.IControl, 'top-left');
    }

    /**
     * Set geocoder result callback
     * @param callback - Function to call when a geocoder result is selected
     */
    public setGeocoderResultCallback(callback: (result: NominatimResult) => void): void {
        this.onGeocoderResult = callback;
    }

    /**
     * Initialize terrain if configured
     */
    private initializeTerrain(): void {
        const config = MapConfigManager.getInstance().getConfig();
        const terrainConfig = getTerrainSource(config);

        if (!terrainConfig) return;

        // Terrain is already configured in the style from createMapStyle
        // Just enable terrain rendering
        this.map.setTerrain({
            source: terrainConfig.source,
            exaggeration: terrainConfig.exaggeration,
        });
    }

    /**
     * Initialize visualization layers (heatmap, clusters, points)
     */
    private initializeLayers(): void {
        this.addLocationSource();
        this.addHeatmapLayer();
        this.addClusterLayers();
        this.addPointLayer();
        this.addHexagonLayers();
        this.addArcLayers();
        this.setupEventHandlers();
    }

    /**
     * Add GeoJSON source for location data
     */
    private addLocationSource(): void {
        if (!this.map.getSource('locations')) {
            this.map.addSource('locations', {
                type: 'geojson',
                data: {
                    type: 'FeatureCollection',
                    features: [],
                },
                cluster: true,
                clusterMaxZoom: 14,
                clusterRadius: 50,
            });
        }
    }

    /**
     * Add heatmap layer
     */
    private addHeatmapLayer(): void {
        if (this.map.getLayer('heatmap')) return;

        this.map.addLayer({
            id: 'heatmap',
            type: 'heatmap',
            source: 'locations',
            maxzoom: 15,
            paint: {
                'heatmap-weight': [
                    'interpolate',
                    ['linear'],
                    ['get', 'playback_count'],
                    0, 0,
                    50, 0.5,
                    100, 0.75,
                    500, 1,
                ],
                'heatmap-intensity': [
                    'interpolate',
                    ['linear'],
                    ['zoom'],
                    0, 0.5,
                    9, 1.5,
                ],
                'heatmap-color': [
                    'interpolate',
                    ['linear'],
                    ['heatmap-density'],
                    0, 'rgba(0, 255, 255, 0)',
                    0.2, 'rgba(0, 255, 255, 0.5)',
                    0.4, 'rgba(0, 255, 0, 0.7)',
                    0.6, 'rgba(255, 255, 0, 0.8)',
                    0.8, 'rgba(255, 102, 0, 0.9)',
                    1, 'rgba(255, 0, 0, 1)',
                ],
                'heatmap-radius': [
                    'interpolate',
                    ['linear'],
                    ['zoom'],
                    0, 2,
                    9, 20,
                    15, 30,
                ],
                'heatmap-opacity': [
                    'interpolate',
                    ['linear'],
                    ['zoom'],
                    7, 0.8,
                    9, 0.6,
                ],
            },
        });
    }

    /**
     * Add cluster layers (circles and count labels)
     */
    private addClusterLayers(): void {
        if (!this.map.getLayer('clusters')) {
            this.map.addLayer({
                id: 'clusters',
                type: 'circle',
                source: 'locations',
                filter: ['has', 'point_count'],
                paint: {
                    'circle-color': [
                        'step',
                        ['get', 'point_count'],
                        '#4ecdc4', // < 10
                        10, '#ffa500', // 10-30
                        30, '#ff6b6b', // 30-100
                        100, '#e94560', // > 100
                    ],
                    'circle-radius': [
                        'step',
                        ['get', 'point_count'],
                        15, // < 10
                        10, 20, // 10-30
                        30, 25, // 30-100
                        100, 30, // > 100
                    ],
                    'circle-opacity': 0.8,
                    'circle-stroke-width': 2,
                    'circle-stroke-color': 'rgba(255, 255, 255, 0.5)',
                },
            });
        }

        if (!this.map.getLayer('cluster-count')) {
            this.map.addLayer({
                id: 'cluster-count',
                type: 'symbol',
                source: 'locations',
                filter: ['has', 'point_count'],
                layout: {
                    'text-field': '{point_count_abbreviated}',
                    'text-font': ['Open Sans Bold', 'Arial Unicode MS Bold'],
                    'text-size': 12,
                },
                paint: {
                    'text-color': '#ffffff',
                },
            });
        }
    }

    /**
     * Add individual point layer
     */
    private addPointLayer(): void {
        if (this.map.getLayer('unclustered-point')) return;

        this.map.addLayer({
            id: 'unclustered-point',
            type: 'circle',
            source: 'locations',
            filter: ['!', ['has', 'point_count']],
            paint: {
                'circle-color': [
                    'interpolate',
                    ['linear'],
                    ['get', 'playback_count'],
                    0, '#4ecdc4',
                    50, '#ffa500',
                    200, '#ff6b6b',
                    500, '#e94560',
                ],
                'circle-radius': [
                    'interpolate',
                    ['linear'],
                    ['get', 'playback_count'],
                    0, 8,
                    50, 12,
                    200, 16,
                    500, 20,
                ],
                'circle-opacity': 0.8,
                'circle-stroke-width': 2,
                'circle-stroke-color': 'rgba(255, 255, 255, 0.5)',
            },
        });
    }

    /**
     * Add hexagon layers (fill and outline)
     */
    private addHexagonLayers(): void {
        if (!this.map.getSource('hexagons')) {
            this.map.addSource('hexagons', {
                type: 'geojson',
                data: {
                    type: 'FeatureCollection',
                    features: [],
                },
            });
        }

        if (!this.map.getLayer('hexagon-fill')) {
            this.map.addLayer({
                id: 'hexagon-fill',
                type: 'fill',
                source: 'hexagons',
                paint: {
                    'fill-color': [
                        'interpolate',
                        ['linear'],
                        ['get', 'playback_count'],
                        0, 'rgba(99, 102, 241, 0.2)',      // Soft indigo
                        10, 'rgba(124, 58, 237, 0.4)',    // Violet
                        50, 'rgba(168, 85, 247, 0.6)',    // Purple
                        100, 'rgba(236, 72, 153, 0.7)',   // Pink
                        250, 'rgba(239, 68, 68, 0.8)',    // Red
                        500, 'rgba(220, 38, 38, 0.9)',    // Dark red
                    ],
                    'fill-opacity': 0.7,
                },
            });
        }

        if (!this.map.getLayer('hexagon-outline')) {
            this.map.addLayer({
                id: 'hexagon-outline',
                type: 'line',
                source: 'hexagons',
                paint: {
                    'line-color': 'rgba(255, 255, 255, 0.3)',
                    'line-width': 1,
                },
            });
        }
    }

    /**
     * Add arc overlay layers
     */
    private addArcLayers(): void {
        if (!this.map.getSource('arcs')) {
            this.map.addSource('arcs', {
                type: 'geojson',
                data: {
                    type: 'FeatureCollection',
                    features: [],
                },
            });
        }

        if (!this.map.getLayer('arcs')) {
            this.map.addLayer({
                id: 'arcs',
                type: 'line',
                source: 'arcs',
                layout: {
                    'visibility': 'none',
                    'line-cap': 'round',
                    'line-join': 'round',
                },
                paint: {
                    'line-color': [
                        'interpolate',
                        ['linear'],
                        ['get', 'playback_count'],
                        0, 'rgba(99, 102, 241, 0.6)',     // Soft indigo
                        50, 'rgba(124, 58, 237, 0.7)',   // Violet
                        150, 'rgba(168, 85, 247, 0.8)',  // Purple
                        300, 'rgba(236, 72, 153, 0.85)', // Pink
                        500, 'rgba(239, 68, 68, 0.9)',   // Red
                    ],
                    'line-width': [
                        'interpolate',
                        ['linear'],
                        ['get', 'playback_count'],
                        0, 1,
                        50, 2,
                        150, 3,
                        300, 4,
                        500, 6,
                    ],
                    'line-opacity': 0.8,
                },
            });
        }
    }

    /**
     * Set up click and hover event handlers
     */
    private setupEventHandlers(): void {
        this.setupClusterHandlers();
        this.setupPointHandlers();
        this.setupHexagonHandlers();
        this.setupArcHandlers();
    }

    /**
     * Set up cluster click and hover handlers
     */
    private setupClusterHandlers(): void {
        this.map.on('click', 'clusters', (e) => {
            const features = this.map.queryRenderedFeatures(e.point, {
                layers: ['clusters'],
            });

            if (features.length === 0) return;

            const clusterId = features[0].properties?.cluster_id;
            const source = this.map.getSource('locations') as GeoJSONSource;

            source.getClusterExpansionZoom(clusterId).then((zoom) => {
                if (!features[0].geometry || features[0].geometry.type !== 'Point') return;

                this.map.easeTo({
                    center: features[0].geometry.coordinates as [number, number],
                    zoom: zoom ?? this.map.getZoom() + 2,
                });
            }).catch(() => {
                // Silently ignore cluster expansion errors
            });
        });

        this.setCursorHandlers('clusters');
    }

    /**
     * Set up point click and hover handlers
     */
    private setupPointHandlers(): void {
        this.map.on('click', 'unclustered-point', (e) => {
            if (!e.features || e.features.length === 0) return;

            const feature = e.features[0];
            const props = feature.properties;

            if (!props || !feature.geometry || feature.geometry.type !== 'Point') return;

            const coordinates = (feature.geometry.coordinates as [number, number]).slice() as [number, number];

            // Handle antimeridian wrapping
            while (Math.abs(e.lngLat.lng - coordinates[0]) > 180) {
                coordinates[0] += e.lngLat.lng > coordinates[0] ? 360 : -360;
            }

            new maplibregl.Popup()
                .setLngLat(coordinates)
                .setHTML(this.createPopupContent(props))
                .addTo(this.map);
        });

        this.setCursorHandlers('unclustered-point');
    }

    /**
     * Set up hexagon click and hover handlers
     */
    private setupHexagonHandlers(): void {
        this.map.on('click', 'hexagon-fill', (e) => {
            if (!e.features || e.features.length === 0) return;

            const feature = e.features[0];
            const props = feature.properties;

            if (!props) return;

            new maplibregl.Popup()
                .setLngLat(e.lngLat)
                .setHTML(this.createHexagonPopupContent(props))
                .addTo(this.map);
        });

        this.setCursorHandlers('hexagon-fill');
    }

    /**
     * Set up arc click and hover handlers
     */
    private setupArcHandlers(): void {
        this.map.on('click', 'arcs', (e) => {
            if (!e.features || e.features.length === 0) return;

            const feature = e.features[0];
            const props = feature.properties;

            if (!props) return;

            new maplibregl.Popup()
                .setLngLat(e.lngLat)
                .setHTML(this.createArcPopupContent(props))
                .addTo(this.map);
        });

        this.setCursorHandlers('arcs');
    }

    /**
     * Set cursor pointer on hover for a layer
     */
    private setCursorHandlers(layerId: string): void {
        this.map.on('mouseenter', layerId, () => {
            this.map.getCanvas().style.cursor = 'pointer';
        });

        this.map.on('mouseleave', layerId, () => {
            this.map.getCanvas().style.cursor = '';
        });
    }

    /**
     * Restore visualization mode from localStorage
     */
    private restoreVisualizationMode(): void {
        const savedMode = SafeStorage.getItem('map-visualization-mode') as VisualizationMode;
        if (savedMode && ['points', 'clusters', 'heatmap', 'hexagons'].includes(savedMode)) {
            this.currentMode = savedMode;
            // Trigger callback to update UI button states (state persistence fix)
            if (this.onModeChange) {
                this.onModeChange(savedMode);
            }
        }
    }

    /**
     * Set visualization mode (points, clusters, heatmap, or hexagons)
     */
    setVisualizationMode(mode: VisualizationMode): void {
        this.currentMode = mode;
        this.persistVisualizationMode(mode);
        this.updateLayerVisibility();
        this.requestDataIfNeeded(mode);
        this.notifyModeChange(mode);
    }

    /**
     * Persist visualization mode to localStorage
     */
    private persistVisualizationMode(mode: VisualizationMode): void {
        SafeStorage.setItem('map-visualization-mode', mode);
    }

    /**
     * Request hexagon data if entering hexagon mode
     */
    private requestDataIfNeeded(mode: VisualizationMode): void {
        if (mode !== 'hexagons' || !this.onHexagonDataRequest) return;

        try {
            this.onHexagonDataRequest(this.hexagonResolution);
        } catch (error) {
            logger.error('Failed to request hexagon data:', error);
        }
    }

    /**
     * Notify mode change callback
     */
    private notifyModeChange(mode: VisualizationMode): void {
        if (!this.onModeChange) return;

        try {
            this.onModeChange(mode);
        } catch (error) {
            logger.error('Failed to invoke mode change callback:', error);
        }
    }

    /**
     * Set callback for hexagon data requests
     */
    setHexagonDataRequestCallback(callback: (resolution: number) => void): void {
        this.onHexagonDataRequest = callback;
    }

    /**
     * Set hexagon resolution and request new data
     * @param resolution H3 resolution (6=country, 7=city, 8=neighborhood)
     */
    setHexagonResolution(resolution: number): void {
        if (resolution >= 4 && resolution <= 10) {
            this.hexagonResolution = resolution;
            if (this.currentMode === 'hexagons' && this.onHexagonDataRequest) {
                this.onHexagonDataRequest(resolution);
            }
        }
    }

    /**
     * Get current hexagon resolution
     */
    getHexagonResolution(): number {
        return this.hexagonResolution;
    }

    /**
     * Get current hexagon data count
     * Useful for testing and UI feedback
     */
    getHexagonCount(): number {
        return this._hexagonData.length;
    }

    /**
     * Get current visualization mode
     */
    getVisualizationMode(): VisualizationMode {
        return this.currentMode;
    }

    /**
     * Update layer visibility based on current mode
     */
    private updateLayerVisibility(): void {
        const heatmapVisible = this.currentMode === 'heatmap' ? 'visible' : 'none';
        const clustersVisible = this.currentMode === 'clusters' ? 'visible' : 'none';
        const hexagonsVisible = this.currentMode === 'hexagons' ? 'visible' : 'none';

        if (this.map.getLayer('heatmap')) {
            this.map.setLayoutProperty('heatmap', 'visibility', heatmapVisible);
        }

        if (this.map.getLayer('clusters')) {
            this.map.setLayoutProperty('clusters', 'visibility', clustersVisible);
        }

        if (this.map.getLayer('cluster-count')) {
            this.map.setLayoutProperty('cluster-count', 'visibility', clustersVisible);
        }

        if (this.map.getLayer('unclustered-point')) {
            const visible = this.currentMode === 'points' || this.currentMode === 'clusters' ? 'visible' : 'none';
            this.map.setLayoutProperty('unclustered-point', 'visibility', visible);
        }

        // H3 hexagon layers
        if (this.map.getLayer('hexagon-fill')) {
            this.map.setLayoutProperty('hexagon-fill', 'visibility', hexagonsVisible);
        }

        if (this.map.getLayer('hexagon-outline')) {
            this.map.setLayoutProperty('hexagon-outline', 'visibility', hexagonsVisible);
        }
    }

    /**
     * Switch to vector tiles for large datasets (>10k locations)
     */
    private switchToVectorTiles(): void {
        if (this.useVectorTiles) return;

        logger.debug('Switching to vector tiles for better performance (>10k locations)');

        // Remove existing layers
        const layers = ['heatmap', 'clusters', 'cluster-count', 'unclustered-point'];
        layers.forEach((layer) => {
            if (this.map.getLayer(layer)) this.map.removeLayer(layer);
        });

        // Remove existing source
        if (this.map.getSource('locations')) {
            this.map.removeSource('locations');
        }

        // Add vector tile source
        this.map.addSource('locations', {
            type: 'vector',
            tiles: [`${window.location.origin}/api/v1/tiles/{z}/{x}/{y}.pbf`],
            minzoom: 0,
            maxzoom: 14,
        });

        // Add vector tile heatmap layer
        this.map.addLayer({
            id: 'heatmap',
            type: 'heatmap',
            source: 'locations',
            'source-layer': 'locations',
            maxzoom: 15,
            paint: {
                'heatmap-weight': ['get', 'playback_count'],
                'heatmap-intensity': 1.5,
                'heatmap-color': [
                    'interpolate',
                    ['linear'],
                    ['heatmap-density'],
                    0, 'rgba(0, 255, 255, 0)',
                    0.2, 'rgba(0, 255, 255, 0.5)',
                    0.4, 'rgba(0, 255, 0, 0.7)',
                    0.6, 'rgba(255, 255, 0, 0.8)',
                    0.8, 'rgba(255, 102, 0, 0.9)',
                    1, 'rgba(255, 0, 0, 1)',
                ],
                'heatmap-radius': 20,
            },
        });

        // Add vector tile point layer
        this.map.addLayer({
            id: 'unclustered-point',
            type: 'circle',
            source: 'locations',
            'source-layer': 'locations',
            paint: {
                'circle-color': '#11b4da',
                'circle-radius': 6,
                'circle-stroke-width': 1,
                'circle-stroke-color': '#fff',
            },
        });

        this.useVectorTiles = true;
        this.updateLayerVisibility();
    }

    /**
     * Update map with location statistics
     */
    updateLocations(locations: LocationStats[]): void {
        const validLocations = this.filterValidLocations(locations);

        if (this.shouldSwitchToVectorTiles(validLocations)) {
            this.switchToVectorTiles();
            return;
        }

        if (this.useVectorTiles) return;

        const features = this.createLocationFeatures(validLocations);
        this.updateLocationSource(features);
        this.updateLayerVisibility();
        this.fitBoundsToLocations(validLocations);
    }

    /**
     * Filter out invalid locations (0,0 coordinates)
     */
    private filterValidLocations(locations: LocationStats[]): LocationStats[] {
        return locations.filter(
            (location) => location.latitude !== 0 || location.longitude !== 0
        );
    }

    /**
     * Check if we should switch to vector tiles
     */
    private shouldSwitchToVectorTiles(locations: LocationStats[]): boolean {
        return locations.length > this.VECTOR_TILE_THRESHOLD && !this.useVectorTiles;
    }

    /**
     * Create GeoJSON features from location stats
     */
    private createLocationFeatures(locations: LocationStats[]): GeoJSONFeature[] {
        return locations.map((location) => {
            const featureId = `${location.latitude.toFixed(6)}-${location.longitude.toFixed(6)}`;

            return {
                type: 'Feature',
                id: featureId,
                geometry: {
                    type: 'Point',
                    coordinates: [location.longitude, location.latitude],
                },
                properties: {
                    country: location.country,
                    region: location.region || '',
                    city: location.city || 'Unknown',
                    playback_count: location.playback_count,
                    unique_users: location.unique_users,
                    avg_completion: location.avg_completion,
                    first_seen: location.first_seen,
                    last_seen: location.last_seen,
                },
            };
        });
    }

    /**
     * Update location source with features (incremental or full update)
     */
    private updateLocationSource(features: GeoJSONFeature[]): void {
        const source = this.map.getSource('locations') as GeoJSONSource;
        if (!source) {
            this.previousFeatureIds = new Set(features.map((f) => f.id as string));
            return;
        }

        const newFeatureIds = new Set(features.map((f) => f.id as string));

        if (this.hasFeatureSetChanged(newFeatureIds)) {
            this.performFullUpdate(source, features, newFeatureIds);
        } else {
            this.performIncrementalUpdate(features);
        }
    }

    /**
     * Check if feature set has changed
     */
    private hasFeatureSetChanged(newFeatureIds: Set<string>): boolean {
        return (
            newFeatureIds.size !== this.previousFeatureIds.size ||
            ![...newFeatureIds].every((id) => this.previousFeatureIds.has(id))
        );
    }

    /**
     * Perform full GeoJSON update
     */
    private performFullUpdate(
        source: GeoJSONSource,
        features: GeoJSONFeature[],
        newFeatureIds: Set<string>
    ): void {
        const geojson: GeoJSONFeatureCollection = {
            type: 'FeatureCollection',
            features,
        };
        source.setData(geojson);
        this.previousFeatureIds = newFeatureIds;
    }

    /**
     * Perform incremental feature state update (80% faster)
     */
    private performIncrementalUpdate(features: GeoJSONFeature[]): void {
        features.forEach((feature) => {
            if (feature.id && feature.properties) {
                this.map.setFeatureState(
                    { source: 'locations', id: feature.id },
                    feature.properties
                );
            }
        });
    }

    /**
     * Fit map bounds to show all locations
     */
    private fitBoundsToLocations(locations: LocationStats[]): void {
        if (locations.length === 0) return;

        const bounds = new maplibregl.LngLatBounds();
        locations.forEach((location) => {
            bounds.extend([location.longitude, location.latitude]);
        });
        this.map.fitBounds(bounds as LngLatBoundsLike, { padding: 50, maxZoom: 12 });
    }

    /**
     * Update map with playback events and geolocations
     */
    updatePlaybackEvents(
        playbacks: PlaybackEvent[],
        geolocations: Map<string, { lat: number; lng: number; city?: string; country: string }>
    ): void {
        const locationCounts = this.aggregatePlaybacksByLocation(playbacks, geolocations);
        const features = this.createPlaybackFeatures(locationCounts);
        this.updatePlaybackSource(features);
        this.updateLayerVisibility();
    }

    /**
     * Aggregate playback counts by location
     */
    private aggregatePlaybacksByLocation(
        playbacks: PlaybackEvent[],
        geolocations: Map<string, { lat: number; lng: number; city?: string; country: string }>
    ): Map<string, { count: number; lat: number; lng: number; city?: string; country: string }> {
        const locationCounts = new Map<
            string,
            { count: number; lat: number; lng: number; city?: string; country: string }
        >();

        playbacks.forEach((playback) => {
            const geo = geolocations.get(playback.ip_address);
            if (!geo) return;

            const key = `${geo.lat},${geo.lng}`;
            const existing = locationCounts.get(key);

            if (existing) {
                existing.count++;
            } else {
                locationCounts.set(key, {
                    count: 1,
                    lat: geo.lat,
                    lng: geo.lng,
                    city: geo.city,
                    country: geo.country,
                });
            }
        });

        return locationCounts;
    }

    /**
     * Create GeoJSON features from playback location counts
     */
    private createPlaybackFeatures(
        locationCounts: Map<string, { count: number; lat: number; lng: number; city?: string; country: string }>
    ): GeoJSONFeature[] {
        return Array.from(locationCounts.values()).map((loc) => ({
            type: 'Feature',
            geometry: {
                type: 'Point',
                coordinates: [loc.lng, loc.lat],
            },
            properties: {
                country: loc.country,
                region: '',
                city: loc.city || 'Unknown',
                playback_count: loc.count,
                unique_users: 0,
                avg_completion: 0,
                first_seen: '',
                last_seen: '',
            },
        }));
    }

    /**
     * Update playback source with features
     */
    private updatePlaybackSource(features: GeoJSONFeature[]): void {
        const geojson: GeoJSONFeatureCollection = {
            type: 'FeatureCollection',
            features,
        };

        const source = this.map.getSource('locations') as GeoJSONSource;
        if (source) {
            source.setData(geojson);
        }
    }

    /**
     * Create HTML content for location popup
     * Uses escapeHtml to prevent XSS attacks from location names
     */
    private createPopupContent(props: Record<string, unknown>): string {
        const locationName = this.formatLocationName(props);
        const stats = [
            { label: 'Playbacks', value: (props.playback_count as number).toLocaleString() },
            { label: 'Users', value: String(props.unique_users as number) },
            { label: 'Avg Completion', value: `${(props.avg_completion as number).toFixed(1)}%` },
        ];

        return this.buildPopupHTML(locationName, stats);
    }

    /**
     * Format location name from properties
     */
    private formatLocationName(props: Record<string, unknown>): string {
        const city = escapeHtml((props.city as string) || 'Unknown');
        const region = escapeHtml((props.region as string) || '');
        const country = escapeHtml(props.country as string);

        return region ? `${city}, ${region}, ${country}` : `${city}, ${country}`;
    }

    /**
     * Build popup HTML from title and stats
     */
    private buildPopupHTML(title: string, stats: Array<{ label: string; value: string }>): string {
        const statsHTML = stats
            .map(
                (stat) => `
                <div class="popup-stat">
                    <span>${stat.label}:</span>
                    <strong>${stat.value}</strong>
                </div>
            `
            )
            .join('');

        return `
            <div class="popup-title">${title}</div>
            <div class="popup-details">
                ${statsHTML}
            </div>
        `;
    }

    /**
     * Create HTML content for hexagon popup
     */
    private createHexagonPopupContent(props: Record<string, unknown>): string {
        const playbackCount = (props.playback_count as number) || 0;
        const uniqueUsers = (props.unique_users as number) || 0;
        const avgCompletion = (props.avg_completion as number) || 0;
        const totalWatchMinutes = (props.total_watch_minutes as number) || 0;

        const watchTimeStr = this.formatWatchTime(totalWatchMinutes);
        const stats = [
            { label: 'Playbacks', value: playbackCount.toLocaleString() },
            { label: 'Unique Users', value: uniqueUsers.toLocaleString() },
            { label: 'Watch Time', value: watchTimeStr },
            { label: 'Avg Completion', value: `${avgCompletion.toFixed(1)}%` },
        ];

        return this.buildPopupHTML('Hexagon Area', stats);
    }

    /**
     * Format watch time from minutes to human-readable string
     */
    private formatWatchTime(totalMinutes: number): string {
        const hours = Math.floor(totalMinutes / 60);
        const minutes = totalMinutes % 60;
        return hours > 0 ? `${hours}h ${minutes}m` : `${minutes}m`;
    }

    /**
     * Update hexagon visualization with H3 data
     * Converts H3 hexagon stats into GeoJSON polygons for rendering
     */
    updateHexagons(hexagons: H3HexagonStats[]): void {
        this._hexagonData = hexagons;

        // Convert H3 hexagons to GeoJSON polygon features
        const features: HexagonFeature[] = hexagons.map((hex) => {
            // Generate approximate hexagon polygon from center point
            // Note: For production, use h3-js library for exact boundaries
            const hexPolygon = this.generateHexagonPolygon(
                hex.latitude,
                hex.longitude,
                this.hexagonResolution
            );

            return {
                type: 'Feature',
                id: hex.h3_index.toString(),
                geometry: {
                    type: 'Polygon',
                    coordinates: [hexPolygon],
                },
                properties: {
                    h3_index: hex.h3_index,
                    playback_count: hex.playback_count,
                    unique_users: hex.unique_users,
                    avg_completion: hex.avg_completion,
                    total_watch_minutes: hex.total_watch_minutes,
                },
            };
        });

        const geojson: HexagonFeatureCollection = {
            type: 'FeatureCollection',
            features,
        };

        const source = this.map.getSource('hexagons') as GeoJSONSource;
        if (source) {
            source.setData(geojson);
        }

        // Fit bounds if we have hexagons
        if (hexagons.length > 0) {
            const bounds = new maplibregl.LngLatBounds();
            hexagons.forEach((hex) => {
                bounds.extend([hex.longitude, hex.latitude]);
            });
            this.map.fitBounds(bounds as LngLatBoundsLike, { padding: 50, maxZoom: 10 });
        }
    }

    /**
     * Generate approximate hexagon polygon coordinates from center
     * Creates a regular hexagon approximation for visualization
     */
    private generateHexagonPolygon(
        centerLat: number,
        centerLng: number,
        resolution: number
    ): number[][] {
        const edgeKm = this.getH3EdgeLength(resolution);
        const { radiusLat, radiusLng } = this.calculateHexagonRadius(centerLat, edgeKm);
        const vertices = this.generateHexagonVertices(centerLat, centerLng, radiusLat, radiusLng);

        // Close the polygon by repeating the first vertex
        vertices.push(vertices[0]);

        return vertices;
    }

    /**
     * Get H3 edge length by resolution (approximate km)
     */
    private getH3EdgeLength(resolution: number): number {
        const edgeLengths: Record<number, number> = {
            4: 22.6,
            5: 8.5,
            6: 3.2,
            7: 1.2,
            8: 0.46,
            9: 0.17,
            10: 0.065,
        };

        return edgeLengths[resolution] || 1.2;
    }

    /**
     * Calculate hexagon radius in lat/lng degrees from edge length in km
     */
    private calculateHexagonRadius(
        centerLat: number,
        edgeKm: number
    ): { radiusLat: number; radiusLng: number } {
        const latDegPerKm = 1 / 111; // ~111km per degree of latitude
        const lngDegPerKm = 1 / (111 * Math.cos((centerLat * Math.PI) / 180));

        return {
            radiusLat: edgeKm * latDegPerKm,
            radiusLng: edgeKm * lngDegPerKm,
        };
    }

    /**
     * Generate 6 vertices of the hexagon (flat-topped orientation)
     */
    private generateHexagonVertices(
        centerLat: number,
        centerLng: number,
        radiusLat: number,
        radiusLng: number
    ): number[][] {
        const vertices: number[][] = [];

        for (let i = 0; i < 6; i++) {
            const angle = (60 * i - 30) * (Math.PI / 180); // Start at -30 degrees for flat-top
            const lng = centerLng + radiusLng * Math.cos(angle);
            const lat = centerLat + radiusLat * Math.sin(angle);
            vertices.push([lng, lat]);
        }

        return vertices;
    }

    /**
     * Create HTML content for arc popup
     * Uses escapeHtml to prevent XSS attacks from location names
     */
    private createArcPopupContent(props: Record<string, unknown>): string {
        const city = escapeHtml((props.city as string) || 'Unknown');
        const country = escapeHtml((props.country as string) || '');
        const distanceKm = (props.distance_km as number) || 0;
        const playbackCount = (props.playback_count as number) || 0;
        const uniqueUsers = (props.unique_users as number) || 0;
        const avgCompletion = (props.avg_completion as number) || 0;

        const location = country ? `${city}, ${country}` : city;
        const title = `Connection to ${location}`;
        const stats = [
            { label: 'Distance', value: this.formatDistance(distanceKm) },
            { label: 'Playbacks', value: playbackCount.toLocaleString() },
            { label: 'Unique Users', value: uniqueUsers.toLocaleString() },
            { label: 'Avg Completion', value: `${avgCompletion.toFixed(1)}%` },
        ];

        return this.buildPopupHTML(title, stats);
    }

    /**
     * Format distance in km with k suffix for large values
     */
    private formatDistance(distanceKm: number): string {
        return distanceKm >= 1000
            ? `${(distanceKm / 1000).toFixed(1)}k km`
            : `${distanceKm.toFixed(0)} km`;
    }

    /**
     * Set callback for arc data requests
     */
    setArcDataRequestCallback(callback: () => void): void {
        this.onArcDataRequest = callback;
    }

    /**
     * Enable or disable arc overlay
     */
    setArcOverlayEnabled(enabled: boolean): void {
        this._arcOverlayEnabled = enabled;
        SafeStorage.setItem('arc-overlay-enabled', String(enabled));

        // Update layer visibility
        if (this.map.getLayer('arcs')) {
            this.map.setLayoutProperty('arcs', 'visibility', enabled ? 'visible' : 'none');
        }

        // Request arc data when enabling
        if (enabled && this.onArcDataRequest) {
            this.onArcDataRequest();
        }
    }

    /**
     * Check if arc overlay is enabled
     */
    isArcOverlayEnabled(): boolean {
        return this._arcOverlayEnabled;
    }

    /**
     * Restore arc overlay state from localStorage
     */
    restoreArcOverlayState(): void {
        const saved = SafeStorage.getItem('arc-overlay-enabled');
        if (saved === 'true') {
            this._arcOverlayEnabled = true;
            if (this.map.getLayer('arcs')) {
                this.map.setLayoutProperty('arcs', 'visibility', 'visible');
            }
            // Request data on restore
            if (this.onArcDataRequest) {
                this.onArcDataRequest();
            }
        }
    }

    /**
     * Get arc data count
     */
    getArcCount(): number {
        return this._arcData.length;
    }

    /**
     * Update arc visualization with arc data
     * Converts arc stats into curved GeoJSON lines
     */
    updateArcs(arcs: ArcStats[]): void {
        this._arcData = arcs;

        // Convert arcs to GeoJSON line features with bezier curves
        const features: ArcFeature[] = arcs.map((arc, index) => {
            // Generate bezier curve points for smooth arc
            const curvePoints = this.generateBezierCurve(
                [arc.server_longitude, arc.server_latitude],
                [arc.user_longitude, arc.user_latitude],
                arc.distance_km
            );

            return {
                type: 'Feature',
                id: `arc-${index}`,
                geometry: {
                    type: 'LineString',
                    coordinates: curvePoints,
                },
                properties: {
                    city: arc.city,
                    country: arc.country,
                    distance_km: arc.distance_km,
                    playback_count: arc.playback_count,
                    unique_users: arc.unique_users,
                    avg_completion: arc.avg_completion,
                    weight: arc.weight,
                },
            };
        });

        const geojson: ArcFeatureCollection = {
            type: 'FeatureCollection',
            features,
        };

        const source = this.map.getSource('arcs') as GeoJSONSource;
        if (source) {
            source.setData(geojson);
        }
    }

    /**
     * Generate bezier curve points for arc visualization
     * Creates a curved line between two points with height based on distance
     */
    private generateBezierCurve(
        start: [number, number],
        end: [number, number],
        distanceKm: number
    ): number[][] {
        const controlPoint = this.calculateBezierControlPoint(start, end, distanceKm);
        return this.generateBezierPoints(start, end, controlPoint, 32);
    }

    /**
     * Calculate control point for bezier curve
     */
    private calculateBezierControlPoint(
        start: [number, number],
        end: [number, number],
        distanceKm: number
    ): [number, number] {
        const midLng = (start[0] + end[0]) / 2;
        const midLat = (start[1] + end[1]) / 2;

        // Calculate arc height based on distance (cap at 20 degrees offset)
        const arcHeight = Math.min(distanceKm / 100, 20);

        // Calculate perpendicular offset for curve
        const dx = end[0] - start[0];
        const dy = end[1] - start[1];
        const length = Math.sqrt(dx * dx + dy * dy);

        // Perpendicular direction (rotated 90 degrees)
        const perpX = -dy / length;
        const perpY = dx / length;

        // Control point offset from midpoint
        return [midLng + perpX * arcHeight, midLat + perpY * arcHeight];
    }

    /**
     * Generate quadratic bezier curve points
     */
    private generateBezierPoints(
        start: [number, number],
        end: [number, number],
        control: [number, number],
        segments: number
    ): number[][] {
        const points: number[][] = [];

        for (let i = 0; i <= segments; i++) {
            const t = i / segments;
            const point = this.calculateBezierPoint(start, end, control, t);
            points.push(point);
        }

        return points;
    }

    /**
     * Calculate single point on quadratic bezier curve
     * Formula: B(t) = (1-t)^2 * P0 + 2(1-t)t * P1 + t^2 * P2
     */
    private calculateBezierPoint(
        start: [number, number],
        end: [number, number],
        control: [number, number],
        t: number
    ): [number, number] {
        const oneMinusT = 1 - t;
        const oneMinusTSquared = oneMinusT * oneMinusT;
        const tSquared = t * t;

        const lng =
            oneMinusTSquared * start[0] +
            2 * oneMinusT * t * control[0] +
            tSquared * end[0];

        const lat =
            oneMinusTSquared * start[1] +
            2 * oneMinusT * t * control[1] +
            tSquared * end[1];

        return [lng, lat];
    }

    /**
     * Update the map style/theme
     * Supports dark, light, and high-contrast modes
     */
    setStyle(style: 'dark' | 'light' | 'high-contrast'): void {
        const configManager = MapConfigManager.getInstance();
        const config = configManager.getConfig();

        const tileProvider = this.getTileProviderForStyle(style);
        configManager.setConfig({ ...config, tileProvider, theme: style });

        const newStyle = createMapStyle(configManager.getConfig());
        this.map.setStyle(newStyle);
    }

    /**
     * Get tile provider for style/theme
     */
    private getTileProviderForStyle(style: 'dark' | 'light' | 'high-contrast'): TileProvider {
        const styleToProvider: Record<string, TileProvider> = {
            'high-contrast': 'high-contrast-dark',
            'light': 'carto-light',
            'dark': 'carto-dark',
        };

        return styleToProvider[style] || 'carto-dark';
    }

    /**
     * Enable or disable terrain
     */
    setTerrain(enabled: boolean, provider?: 'aws' | 'maptiler' | 'custom'): void {
        const configManager = MapConfigManager.getInstance();
        const config = configManager.getConfig();

        if (enabled) {
            configManager.setConfig({
                ...config,
                terrainProvider: provider || 'aws',
            });
            this.initializeTerrain();
        } else {
            configManager.setConfig({ ...config, terrainProvider: 'none' });
            this.map.setTerrain(null);
        }
    }

    /**
     * Get the MapLibre map instance for external access
     */
    getMap(): MapLibreMap {
        return this.map;
    }

    /**
     * Cleanup resources
     */
    destroy(): void {
        // Remove PMTiles protocol handler
        if (this.pmtilesProtocol) {
            maplibregl.removeProtocol('pmtiles');
            this.pmtilesProtocol = null;
        }

        // Clear geocoder reference
        if (this.geocoder) {
            try {
                this.map.removeControl(this.geocoder as unknown as maplibregl.IControl);
            } catch {
                // Control may already be removed
            }
            this.geocoder = null;
        }

        // Clear callback references to prevent memory leaks
        this.onModeChange = undefined;
        this.onHexagonDataRequest = undefined;
        this.onArcDataRequest = undefined;
        this.onGeocoderResult = undefined;

        // Clear data caches
        this.previousFeatureIds.clear();
        this._hexagonData = [];
        this._arcData = [];

        // Remove map (this also cleans up all event listeners and controls)
        this.map.remove();
    }
}
