// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
import type { GlobeManagerDeckGLEnhanced } from './globe-deckgl-enhanced';
import type { TemporalHeatmapResponse } from './api';
import { NominatimGeocoder } from './geocoder';
import { MapConfigManager } from './map-config';
import { createLogger } from './logger';

const logger = createLogger('GlobeControlsEnhanced');

/**
 * GlobeControlsEnhanced - Advanced UI control panel with geocoder and layer controls
 *
 * Features:
 * - Layer toggles (scatterplot, hexagon, arcs, trips, satellite)
 * - Nominatim Geocoder for location search (OSM-based, no API key)
 * - Animation controls with timeline scrubber
 * - Screenshot export
 * - Satellite imagery toggle
 * - Terrain toggle
 * - Tile provider selection
 * - Performance metrics display
 */
export class GlobeControlsEnhanced {
    private globe: GlobeManagerDeckGLEnhanced;
    private controlsContainer: HTMLElement | null = null;
    private geocoder: NominatimGeocoder | null = null;
    private temporalData: TemporalHeatmapResponse | null = null;
    private currentTimeIndex: number = 0;
    private animationInterval: number | null = null;
    private animationSpeed: number = 1000;

    constructor(globe: GlobeManagerDeckGLEnhanced, containerId: string) {
        this.globe = globe;
        this.controlsContainer = document.getElementById(containerId);

        if (!this.controlsContainer) {
            logger.error(`Globe controls container "${containerId}" not found`);
            return;
        }

        this.createControls();
        this.initializeGeocoder();
    }

    /**
     * Create the control panel UI
     */
    private createControls(): void {
        if (!this.controlsContainer) return;

        const controlsHTML = `
            <div class="globe-controls-enhanced" style="
                position: absolute;
                top: 10px;
                left: 10px;
                background: rgba(20, 20, 30, 0.95);
                border: 1px solid #4ecdc4;
                border-radius: 8px;
                padding: 16px;
                color: #eaeaea;
                font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
                font-size: 13px;
                z-index: 100;
                min-width: 280px;
                max-width: 320px;
                box-shadow: 0 4px 12px rgba(0, 0, 0, 0.4);
            ">
                <div style="font-weight: 600; margin-bottom: 12px; color: #4ecdc4; font-size: 14px;">
                    Enhanced Globe
                </div>

                <!-- Geocoder Search -->
                <div style="margin-bottom: 16px;">
                    <div id="geocoder-container" style="
                        margin-bottom: 8px;
                    "></div>
                </div>

                <!-- Layer Toggles -->
                <div class="layer-toggles" style="margin-bottom: 16px;">
                    <div style="font-weight: 500; margin-bottom: 8px; font-size: 12px; color: #a0a0a0;">
                        VISUALIZATION LAYERS
                    </div>
                    <label style="display: flex; align-items: center; margin-bottom: 6px; cursor: pointer;">
                        <input type="checkbox" id="toggle-scatterplot" checked style="margin-right: 8px; cursor: pointer;">
                        <span style="font-size: 12px;">Scatterplot Points</span>
                    </label>
                    <label style="display: flex; align-items: center; margin-bottom: 6px; cursor: pointer;">
                        <input type="checkbox" id="toggle-hexagon" style="margin-right: 8px; cursor: pointer;">
                        <span style="font-size: 12px;">3D Hexagon Aggregation</span>
                    </label>
                    <label style="display: flex; align-items: center; margin-bottom: 6px; cursor: pointer;">
                        <input type="checkbox" id="toggle-arcs" style="margin-right: 8px; cursor: pointer;">
                        <span style="font-size: 12px;">User to Server Arcs</span>
                    </label>
                    <label style="display: flex; align-items: center; margin-bottom: 6px; cursor: pointer;">
                        <input type="checkbox" id="toggle-trips" style="margin-right: 8px; cursor: pointer;">
                        <span style="font-size: 12px;">Temporal Animation</span>
                    </label>
                </div>

                <!-- Basemap Options -->
                <div style="margin-bottom: 16px;">
                    <div style="font-weight: 500; margin-bottom: 8px; font-size: 12px; color: #a0a0a0;">
                        BASEMAP
                    </div>
                    <div style="margin-bottom: 8px;">
                        <select id="tile-provider-select" style="
                            width: 100%;
                            padding: 6px 8px;
                            background: #3a3a4e;
                            color: #eaeaea;
                            border: 1px solid #4ecdc4;
                            border-radius: 4px;
                            font-size: 12px;
                            cursor: pointer;
                        ">
                            <option value="carto-dark">CartoDB Dark</option>
                            <option value="carto-light">CartoDB Light</option>
                            <option value="carto-voyager">CartoDB Voyager</option>
                            <option value="osm">OpenStreetMap</option>
                        </select>
                    </div>
                    <label style="display: flex; align-items: center; margin-bottom: 6px; cursor: pointer;">
                        <input type="checkbox" id="toggle-satellite" style="margin-right: 8px; cursor: pointer;">
                        <span style="font-size: 12px;">Satellite Imagery</span>
                    </label>
                    <label style="display: flex; align-items: center; cursor: pointer;">
                        <input type="checkbox" id="toggle-terrain" style="margin-right: 8px; cursor: pointer;">
                        <span style="font-size: 12px;">3D Terrain (AWS)</span>
                    </label>
                </div>

                <!-- Animation Controls -->
                <div id="animation-controls" style="margin-bottom: 16px; display: none;">
                    <div style="font-weight: 500; margin-bottom: 8px; font-size: 12px; color: #a0a0a0;">
                        ANIMATION
                    </div>
                    <div style="display: flex; gap: 8px; margin-bottom: 8px;">
                        <button id="btn-play-pause" style="
                            flex: 1;
                            padding: 6px 12px;
                            background: #4ecdc4;
                            color: #16213e;
                            border: none;
                            border-radius: 4px;
                            font-weight: 600;
                            cursor: pointer;
                            font-size: 12px;
                        ">Play</button>
                        <button id="btn-stop" style="
                            padding: 6px 12px;
                            background: #e94560;
                            color: white;
                            border: none;
                            border-radius: 4px;
                            font-weight: 600;
                            cursor: pointer;
                            font-size: 12px;
                        ">Stop</button>
                    </div>
                    <div style="margin-bottom: 6px;">
                        <label style="display: block; margin-bottom: 4px; font-size: 11px; color: #a0a0a0;">
                            Speed: <span id="speed-value">1.0x</span>
                        </label>
                        <input type="range" id="speed-slider" min="0.5" max="5" step="0.5" value="1" style="
                            width: 100%;
                            height: 4px;
                            background: #3a3a4e;
                            outline: none;
                            border-radius: 2px;
                        ">
                    </div>
                    <div id="timeline-scrubber" style="display: none;">
                        <label style="display: block; margin-bottom: 4px; font-size: 11px; color: #a0a0a0;">
                            Time: <span id="time-value">--</span>
                        </label>
                        <input type="range" id="time-slider" min="0" max="100" step="1" value="0" style="
                            width: 100%;
                            height: 4px;
                            background: #3a3a4e;
                            outline: none;
                            border-radius: 2px;
                        ">
                    </div>
                </div>

                <!-- Action Buttons -->
                <div class="action-buttons" style="display: flex; gap: 8px; margin-bottom: 12px;">
                    <button id="btn-screenshot" style="
                        flex: 1;
                        padding: 8px 12px;
                        background: #3a3a4e;
                        color: #eaeaea;
                        border: 1px solid #4ecdc4;
                        border-radius: 4px;
                        font-weight: 500;
                        cursor: pointer;
                        font-size: 12px;
                    ">Screenshot</button>
                    <button id="btn-reset-view" style="
                        flex: 1;
                        padding: 8px 12px;
                        background: #3a3a4e;
                        color: #eaeaea;
                        border: 1px solid #4ecdc4;
                        border-radius: 4px;
                        font-weight: 500;
                        cursor: pointer;
                        font-size: 12px;
                    ">Reset View</button>
                </div>

                <!-- Performance Stats -->
                <div id="performance-stats" style="
                    padding: 8px;
                    background: rgba(78, 205, 196, 0.1);
                    border-radius: 4px;
                    font-size: 11px;
                    color: #a0a0a0;
                    display: none;
                ">
                    <div><span>FPS:</span> <span id="fps-value">--</span></div>
                    <div><span>Layers:</span> <span id="layers-value">--</span></div>
                </div>
            </div>
        `;

        const controlsElement = document.createElement('div');
        controlsElement.innerHTML = controlsHTML;
        this.controlsContainer.appendChild(controlsElement.firstElementChild!);

        this.attachEventListeners();
    }

    /**
     * Initialize Nominatim Geocoder for location search
     */
    private initializeGeocoder(): void {
        const map = this.globe.getMap();
        if (!map) return;

        // Create Nominatim geocoder (OSM-based, no API key required)
        this.geocoder = new NominatimGeocoder({
            placeholder: 'Search locations...',
            limit: 5,
            debounceMs: 300,
            flyToOptions: {
                speed: 1.2,
                zoom: 10,
            },
        });

        // Add geocoder to container
        const geocoderContainer = document.getElementById('geocoder-container');
        if (geocoderContainer) {
            const geocoderElement = this.geocoder.onAdd(map);
            geocoderContainer.appendChild(geocoderElement);
        }
    }

    /**
     * Attach event listeners
     */
    private attachEventListeners(): void {
        // Layer toggles
        const scatterplotToggle = document.getElementById('toggle-scatterplot') as HTMLInputElement;
        const hexagonToggle = document.getElementById('toggle-hexagon') as HTMLInputElement;
        const arcsToggle = document.getElementById('toggle-arcs') as HTMLInputElement;
        const tripsToggle = document.getElementById('toggle-trips') as HTMLInputElement;
        const satelliteToggle = document.getElementById('toggle-satellite') as HTMLInputElement;
        const terrainToggle = document.getElementById('toggle-terrain') as HTMLInputElement;
        const tileProviderSelect = document.getElementById('tile-provider-select') as HTMLSelectElement;

        scatterplotToggle?.addEventListener('change', (e) => {
            this.globe.toggleLayer('scatterplot', (e.target as HTMLInputElement).checked);
            this.updateLayerCount();
        });

        hexagonToggle?.addEventListener('change', (e) => {
            this.globe.toggleLayer('hexagon', (e.target as HTMLInputElement).checked);
            this.updateLayerCount();
        });

        arcsToggle?.addEventListener('change', (e) => {
            this.globe.toggleLayer('arcs', (e.target as HTMLInputElement).checked);
            this.updateLayerCount();
        });

        tripsToggle?.addEventListener('change', (e) => {
            const enabled = (e.target as HTMLInputElement).checked;
            this.globe.toggleLayer('trips', enabled);

            const animControls = document.getElementById('animation-controls');
            if (animControls) {
                animControls.style.display = enabled ? 'block' : 'none';
            }
            this.updateLayerCount();
        });

        satelliteToggle?.addEventListener('change', (e) => {
            this.toggleSatelliteImagery((e.target as HTMLInputElement).checked);
        });

        terrainToggle?.addEventListener('change', (e) => {
            this.toggleTerrain((e.target as HTMLInputElement).checked);
        });

        tileProviderSelect?.addEventListener('change', (e) => {
            this.changeTileProvider((e.target as HTMLSelectElement).value as 'carto-dark' | 'carto-light' | 'carto-voyager' | 'osm');
        });

        // Animation controls
        const playPauseBtn = document.getElementById('btn-play-pause');
        const stopBtn = document.getElementById('btn-stop');
        const speedSlider = document.getElementById('speed-slider') as HTMLInputElement;
        const timeSlider = document.getElementById('time-slider') as HTMLInputElement;

        playPauseBtn?.addEventListener('click', () => this.togglePlayPause());
        stopBtn?.addEventListener('click', () => this.stopAnimation());

        speedSlider?.addEventListener('input', (e) => {
            const speed = parseFloat((e.target as HTMLInputElement).value);
            this.setAnimationSpeed(speed);
            const speedValue = document.getElementById('speed-value');
            if (speedValue) speedValue.textContent = `${speed.toFixed(1)}x`;
        });

        timeSlider?.addEventListener('input', (e) => {
            const index = parseInt((e.target as HTMLInputElement).value);
            this.scrubToTime(index);
        });

        // Action buttons
        const screenshotBtn = document.getElementById('btn-screenshot');
        const resetViewBtn = document.getElementById('btn-reset-view');

        screenshotBtn?.addEventListener('click', () => this.takeScreenshot());
        resetViewBtn?.addEventListener('click', () => this.globe.resetView());
    }

    /**
     * Change tile provider
     */
    private changeTileProvider(provider: 'carto-dark' | 'carto-light' | 'carto-voyager' | 'osm'): void {
        const map = this.globe.getMap();
        if (!map) return;

        const configManager = MapConfigManager.getInstance();
        const config = configManager.getConfig();
        configManager.setConfig({ ...config, tileProvider: provider });

        // Need to reload the map style
        // This is handled by the globe when theme changes
        const isDark = provider === 'carto-dark';
        this.globe.updateTheme(isDark);
    }

    /**
     * Toggle terrain
     */
    private toggleTerrain(enabled: boolean): void {
        const map = this.globe.getMap();
        if (!map) return;

        const configManager = MapConfigManager.getInstance();
        const config = configManager.getConfig();

        if (enabled) {
            configManager.setConfig({ ...config, terrainProvider: 'aws' });

            // Add terrain source if not exists
            if (!map.getSource('terrain')) {
                map.addSource('terrain', {
                    type: 'raster-dem',
                    tiles: ['https://s3.amazonaws.com/elevation-tiles-prod/terrarium/{z}/{x}/{y}.png'],
                    tileSize: 256,
                    maxzoom: 15,
                    encoding: 'terrarium',
                });
            }

            map.setTerrain({
                source: 'terrain',
                exaggeration: config.terrainExaggeration,
            });
        } else {
            configManager.setConfig({ ...config, terrainProvider: 'none' });
            map.setTerrain(null);
        }
    }

    /**
     * Toggle satellite imagery layer
     */
    private toggleSatelliteImagery(enabled: boolean): void {
        const map = this.globe.getMap();
        if (!map) return;

        if (enabled) {
            // Add ESRI World Imagery (free, no API key)
            if (!map.getSource('satellite')) {
                map.addSource('satellite', {
                    type: 'raster',
                    tiles: [
                        'https://server.arcgisonline.com/ArcGIS/rest/services/World_Imagery/MapServer/tile/{z}/{y}/{x}',
                    ],
                    tileSize: 256,
                    attribution: 'Tiles &copy; Esri',
                });

                // Get the first layer ID to insert before
                const style = map.getStyle();
                const layers = style?.layers;
                const firstLayerId = layers && layers.length > 0 ? layers[0].id : undefined;

                map.addLayer(
                    {
                        id: 'satellite-layer',
                        type: 'raster',
                        source: 'satellite',
                        minzoom: 0,
                        maxzoom: 22,
                    },
                    firstLayerId
                );
            }

            map.setLayoutProperty('satellite-layer', 'visibility', 'visible');

            // Hide the base layer
            const baseLayerId = this.getBaseLayerId();
            if (baseLayerId && map.getLayer(baseLayerId)) {
                map.setLayoutProperty(baseLayerId, 'visibility', 'none');
            }
        } else {
            if (map.getLayer('satellite-layer')) {
                map.setLayoutProperty('satellite-layer', 'visibility', 'none');

                // Show the base layer
                const baseLayerId = this.getBaseLayerId();
                if (baseLayerId && map.getLayer(baseLayerId)) {
                    map.setLayoutProperty(baseLayerId, 'visibility', 'visible');
                }
            }
        }
    }

    /**
     * Get the ID of the base raster layer
     */
    private getBaseLayerId(): string | null {
        const map = this.globe.getMap();
        if (!map) return null;

        const style = map.getStyle();
        if (!style || !style.layers) return null;

        // Find the base raster layer
        for (const layer of style.layers) {
            if (layer.type === 'raster' && layer.id !== 'satellite-layer') {
                return layer.id;
            }
        }

        return null;
    }

    /**
     * Update layer count display
     */
    private updateLayerCount(): void {
        const layerValue = document.getElementById('layers-value');
        if (layerValue) {
            const config = this.globe.getLayerConfig();
            const count = Object.values(config).filter((v) => v).length;
            layerValue.textContent = count.toString();
        }
    }

    /**
     * Set temporal data for animation (Task 28: pause during data update)
     * Pauses any running animation and resets to start when data changes
     */
    public setTemporalData(data: TemporalHeatmapResponse): void {
        // Pause animation if running (Task 28: prevent data race)
        const wasPlaying = this.isPlaying();
        if (wasPlaying) {
            this.pauseAnimation();
            const playPauseBtn = document.getElementById('btn-play-pause');
            if (playPauseBtn) playPauseBtn.textContent = 'Play';
        }

        // Reset to start
        this.currentTimeIndex = 0;
        this.temporalData = data;

        if (data.buckets.length > 0) {
            this.globe.updateTemporalData(data.buckets);

            const timelineScrubber = document.getElementById('timeline-scrubber');
            const timeSlider = document.getElementById('time-slider') as HTMLInputElement;

            if (timelineScrubber) timelineScrubber.style.display = 'block';
            if (timeSlider) {
                timeSlider.max = (data.buckets.length - 1).toString();
                timeSlider.value = '0';
            }

            this.updateTimeDisplay();
        } else {
            // Hide scrubber if no data
            const timelineScrubber = document.getElementById('timeline-scrubber');
            if (timelineScrubber) timelineScrubber.style.display = 'none';
        }
    }

    /**
     * Toggle play/pause animation
     */
    private togglePlayPause(): void {
        const playPauseBtn = document.getElementById('btn-play-pause');
        if (!playPauseBtn) return;

        if (this.animationInterval) {
            this.pauseAnimation();
            playPauseBtn.textContent = 'Play';
        } else {
            this.playAnimation();
            playPauseBtn.textContent = 'Pause';
        }
    }

    /**
     * Start animation playback (Task 28: fix edge cases)
     * Clears any existing interval before starting to prevent race conditions
     */
    private playAnimation(): void {
        if (!this.temporalData || this.temporalData.buckets.length === 0) return;

        // Clear any existing interval first (Task 28: prevent race condition)
        if (this.animationInterval) {
            clearInterval(this.animationInterval);
            this.animationInterval = null;
        }

        this.globe.startAnimation();

        this.animationInterval = window.setInterval(() => {
            // Safety check - temporalData could be cleared during animation
            if (!this.temporalData || this.temporalData.buckets.length === 0) {
                this.pauseAnimation();
                return;
            }

            this.currentTimeIndex++;
            if (this.currentTimeIndex >= this.temporalData.buckets.length) {
                this.currentTimeIndex = 0;
            }

            const bucket = this.temporalData.buckets[this.currentTimeIndex];
            const timestamp = new Date(bucket.start_time).getTime();
            this.globe.setAnimationTime(timestamp);

            this.updateTimeDisplay();

            const timeSlider = document.getElementById('time-slider') as HTMLInputElement;
            if (timeSlider) timeSlider.value = this.currentTimeIndex.toString();
        }, this.animationSpeed);
    }

    /**
     * Pause animation (Task 28: fix edge cases)
     * Safely clears interval and stops globe animation
     */
    private pauseAnimation(): void {
        // Clear interval first (before stopping globe to prevent race)
        if (this.animationInterval) {
            clearInterval(this.animationInterval);
            this.animationInterval = null;
        }
        this.globe.stopAnimation();
    }

    /**
     * Check if animation is currently playing
     */
    private isPlaying(): boolean {
        return this.animationInterval !== null;
    }

    /**
     * Stop animation and reset (Task 28: fix edge cases)
     */
    private stopAnimation(): void {
        // Pause first to clean up intervals
        this.pauseAnimation();
        this.currentTimeIndex = 0;

        const playPauseBtn = document.getElementById('btn-play-pause');
        if (playPauseBtn) playPauseBtn.textContent = 'Play';

        const timeSlider = document.getElementById('time-slider') as HTMLInputElement;
        if (timeSlider) timeSlider.value = '0';

        if (this.temporalData && this.temporalData.buckets.length > 0) {
            const bucket = this.temporalData.buckets[0];
            const timestamp = new Date(bucket.start_time).getTime();
            this.globe.setAnimationTime(timestamp);
        }

        this.updateTimeDisplay();
    }

    /**
     * Set animation speed (Task 28: preserve current position during speed change)
     * @param multiplier - Speed multiplier (e.g., 2 = 2x speed)
     */
    private setAnimationSpeed(multiplier: number): void {
        const wasPlaying = this.isPlaying();
        const currentIndex = this.currentTimeIndex; // Preserve position

        this.animationSpeed = 1000 / multiplier;

        if (wasPlaying) {
            // Pause and restart with new speed, maintaining position
            this.pauseAnimation();
            this.currentTimeIndex = currentIndex; // Restore position
            this.playAnimation();
        }
    }

    /**
     * Scrub to specific time index (Task 28: pause animation during scrub)
     * @param index - Time bucket index to scrub to
     */
    private scrubToTime(index: number): void {
        if (!this.temporalData || index >= this.temporalData.buckets.length) return;

        // Pause animation during scrub to prevent desync (Task 28)
        const wasPlaying = this.isPlaying();
        if (wasPlaying) {
            this.pauseAnimation();

            // Update play/pause button text
            const playPauseBtn = document.getElementById('btn-play-pause');
            if (playPauseBtn) playPauseBtn.textContent = 'Play';
        }

        this.currentTimeIndex = index;
        const bucket = this.temporalData.buckets[index];
        const timestamp = new Date(bucket.start_time).getTime();
        this.globe.setAnimationTime(timestamp);

        this.updateTimeDisplay();
    }

    /**
     * Update time display
     */
    private updateTimeDisplay(): void {
        const timeValue = document.getElementById('time-value');
        if (!timeValue || !this.temporalData) return;

        if (this.currentTimeIndex < this.temporalData.buckets.length) {
            const bucket = this.temporalData.buckets[this.currentTimeIndex];
            const date = new Date(bucket.start_time);
            timeValue.textContent = date.toLocaleString();
        } else {
            timeValue.textContent = '--';
        }
    }

    /**
     * Take screenshot
     */
    private takeScreenshot(): void {
        const globeContainer = document.getElementById('globe') || document.getElementById('multi-view-globe');
        if (!globeContainer) return;

        const canvas = globeContainer.querySelector('canvas');
        if (!canvas) {
            logger.error('Canvas not found for screenshot');
            return;
        }

        try {
            canvas.toBlob((blob) => {
                if (!blob) return;

                const url = URL.createObjectURL(blob);
                const link = document.createElement('a');
                link.href = url;
                link.download = `cartographus-globe-${Date.now()}.png`;
                link.click();

                URL.revokeObjectURL(url);
            }, 'image/png');
        } catch (error) {
            logger.error('Screenshot failed:', error);
        }
    }

    /**
     * Destroy controls and cleanup (Task 28: ensure animation cleanup)
     */
    public destroy(): void {
        // Stop any running animation first (clears interval and stops globe)
        this.pauseAnimation();

        // Reset animation state
        this.currentTimeIndex = 0;
        this.temporalData = null;

        if (this.geocoder) {
            this.geocoder.onRemove();
        }

        if (this.controlsContainer) {
            this.controlsContainer.innerHTML = '';
        }
    }
}
