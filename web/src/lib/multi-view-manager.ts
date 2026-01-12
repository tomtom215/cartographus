// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
import {MapManager} from './map';
import {GlobeManagerDeckGLEnhanced} from './globe-deckgl-enhanced';
import type {LocationStats, LocationFilter, ServerInfo, TemporalHeatmapResponse} from './api';
import { createLogger } from './logger';

const logger = createLogger('MultiView');

/**
 * MultiViewManager - Manages synchronized 2D map and 3D globe views
 *
 * Features:
 * - Side-by-side 2D map and 3D globe
 * - Synchronized camera movements
 * - Synchronized data updates
 * - Synchronized filter application
 * - Responsive layout (stack vertically on mobile)
 * - Toggleable views (show one, show both)
 */
export class MultiViewManager {
    private mapManager: MapManager | null = null;
    private globeManager: GlobeManagerDeckGLEnhanced | null = null;
    private containerId: string;
    private syncEnabled: boolean = true;
    private currentLayout: 'side-by-side' | 'map-only' | 'globe-only' = 'side-by-side';

    constructor(containerId: string) {
        this.containerId = containerId;
    }

    /**
     * Initialize both views
     */
    public initialize(onModeChange?: (mode: any) => void): void {
        const container = document.getElementById(this.containerId);
        if (!container) {
            logger.error(`MultiView container "${this.containerId}" not found`);
            return;
        }

        // Create layout structure
        this.createLayout(container);

        // Initialize map manager
        this.mapManager = new MapManager('multi-view-map', onModeChange);

        // Initialize globe manager
        this.globeManager = new GlobeManagerDeckGLEnhanced('multi-view-globe');
        this.globeManager.initialize();

        // Set up camera synchronization
        if (this.syncEnabled) {
            this.setupCameraSync();
        }

        // Apply initial layout
        this.applyLayout(this.currentLayout);
    }

    /**
     * Create the multi-view layout structure
     */
    private createLayout(container: HTMLElement): void {
        container.innerHTML = `
            <div id="multi-view-container" style="
                display: flex;
                width: 100%;
                height: 100%;
                gap: 0;
                position: relative;
            ">
                <!-- 2D Map View -->
                <div id="multi-view-map-container" style="
                    flex: 1;
                    height: 100%;
                    position: relative;
                    border-right: 2px solid #4ecdc4;
                ">
                    <div id="multi-view-map" style="width: 100%; height: 100%;"></div>
                    <div style="
                        position: absolute;
                        top: 10px;
                        left: 10px;
                        background: rgba(20, 20, 30, 0.95);
                        border: 1px solid #4ecdc4;
                        border-radius: 4px;
                        padding: 8px 12px;
                        color: #4ecdc4;
                        font-size: 12px;
                        font-weight: 600;
                        z-index: 10;
                    ">2D MAP VIEW</div>
                </div>

                <!-- 3D Globe View -->
                <div id="multi-view-globe-container" style="
                    flex: 1;
                    height: 100%;
                    position: relative;
                ">
                    <div id="multi-view-globe" style="width: 100%; height: 100%;"></div>
                    <div style="
                        position: absolute;
                        top: 10px;
                        left: 10px;
                        background: rgba(20, 20, 30, 0.95);
                        border: 1px solid #4ecdc4;
                        border-radius: 4px;
                        padding: 8px 12px;
                        color: #4ecdc4;
                        font-size: 12px;
                        font-weight: 600;
                        z-index: 10;
                    ">3D GLOBE VIEW</div>
                </div>

                <!-- View Controls -->
                <div id="multi-view-controls" style="
                    position: absolute;
                    top: 10px;
                    right: 10px;
                    background: rgba(20, 20, 30, 0.95);
                    border: 1px solid #4ecdc4;
                    border-radius: 8px;
                    padding: 12px;
                    z-index: 100;
                    display: flex;
                    flex-direction: column;
                    gap: 8px;
                ">
                    <div style="
                        font-weight: 600;
                        color: #4ecdc4;
                        font-size: 12px;
                        margin-bottom: 4px;
                    ">VIEW LAYOUT</div>

                    <button id="btn-view-both" class="view-btn active" style="
                        padding: 6px 12px;
                        background: #4ecdc4;
                        color: #16213e;
                        border: none;
                        border-radius: 4px;
                        font-weight: 600;
                        cursor: pointer;
                        font-size: 11px;
                    ">Both Views</button>

                    <button id="btn-view-map" class="view-btn" style="
                        padding: 6px 12px;
                        background: #3a3a4e;
                        color: #eaeaea;
                        border: 1px solid #4ecdc4;
                        border-radius: 4px;
                        font-weight: 500;
                        cursor: pointer;
                        font-size: 11px;
                    ">Map Only</button>

                    <button id="btn-view-globe" class="view-btn" style="
                        padding: 6px 12px;
                        background: #3a3a4e;
                        color: #eaeaea;
                        border: 1px solid #4ecdc4;
                        border-radius: 4px;
                        font-weight: 500;
                        cursor: pointer;
                        font-size: 11px;
                    ">Globe Only</button>

                    <div style="
                        margin-top: 8px;
                        padding-top: 8px;
                        border-top: 1px solid rgba(78, 205, 196, 0.3);
                    ">
                        <label style="
                            display: flex;
                            align-items: center;
                            color: #eaeaea;
                            font-size: 11px;
                            cursor: pointer;
                        ">
                            <input type="checkbox" id="sync-cameras" checked style="
                                margin-right: 6px;
                                cursor: pointer;
                            ">
                            Sync Cameras
                        </label>
                    </div>
                </div>
            </div>

            <!-- Mobile-responsive styles -->
            <style>
                @media (max-width: 768px) {
                    #multi-view-container {
                        flex-direction: column !important;
                    }
                    #multi-view-map-container {
                        border-right: none !important;
                        border-bottom: 2px solid #4ecdc4 !important;
                    }
                }

                .view-btn.active {
                    background: #4ecdc4 !important;
                    color: #16213e !important;
                }

                .view-btn:not(.active) {
                    background: #3a3a4e !important;
                    color: #eaeaea !important;
                }
            </style>
        `;

        // Attach event listeners
        this.attachLayoutControls();
    }

    /**
     * Attach event listeners to layout controls
     */
    private attachLayoutControls(): void {
        const btnBoth = document.getElementById('btn-view-both');
        const btnMap = document.getElementById('btn-view-map');
        const btnGlobe = document.getElementById('btn-view-globe');
        const syncCheckbox = document.getElementById('sync-cameras') as HTMLInputElement;

        btnBoth?.addEventListener('click', () => {
            this.setLayout('side-by-side');
            this.updateActiveButton('btn-view-both');
        });

        btnMap?.addEventListener('click', () => {
            this.setLayout('map-only');
            this.updateActiveButton('btn-view-map');
        });

        btnGlobe?.addEventListener('click', () => {
            this.setLayout('globe-only');
            this.updateActiveButton('btn-view-globe');
        });

        syncCheckbox?.addEventListener('change', (e) => {
            this.syncEnabled = (e.target as HTMLInputElement).checked;
            if (this.syncEnabled) {
                this.setupCameraSync();
            }
        });
    }

    /**
     * Update active button styling
     */
    private updateActiveButton(activeId: string): void {
        const buttons = document.querySelectorAll('.view-btn');
        buttons.forEach(btn => {
            if (btn.id === activeId) {
                btn.classList.add('active');
            } else {
                btn.classList.remove('active');
            }
        });
    }

    /**
     * Set layout mode
     */
    public setLayout(layout: 'side-by-side' | 'map-only' | 'globe-only'): void {
        this.currentLayout = layout;
        this.applyLayout(layout);
    }

    /**
     * Apply layout changes to DOM
     */
    private applyLayout(layout: 'side-by-side' | 'map-only' | 'globe-only'): void {
        const mapContainer = document.getElementById('multi-view-map-container');
        const globeContainer = document.getElementById('multi-view-globe-container');

        if (!mapContainer || !globeContainer) return;

        switch (layout) {
            case 'side-by-side':
                mapContainer.style.display = 'block';
                globeContainer.style.display = 'block';
                mapContainer.style.flex = '1';
                globeContainer.style.flex = '1';
                break;

            case 'map-only':
                mapContainer.style.display = 'block';
                globeContainer.style.display = 'none';
                mapContainer.style.flex = '1';
                break;

            case 'globe-only':
                mapContainer.style.display = 'none';
                globeContainer.style.display = 'block';
                globeContainer.style.flex = '1';
                break;
        }

        // Trigger resize on visible views
        setTimeout(() => {
            if (layout !== 'globe-only' && this.mapManager) {
                this.mapManager.getMap()?.resize();
            }
            if (layout !== 'map-only' && this.globeManager) {
                this.globeManager.show();
            }
        }, 100);
    }

    /**
     * Set up synchronized camera movements between map and globe
     */
    private setupCameraSync(): void {
        if (!this.mapManager || !this.globeManager) return;

        const map = this.mapManager.getMap();
        const globe = this.globeManager.getMap();

        if (!map || !globe) return;

        // Sync map → globe
        map.on('move', () => {
            if (!this.syncEnabled) return;

            const center = map.getCenter();
            const zoom = map.getZoom();

            // Update globe camera (debounced to avoid circular updates)
            if (globe && !globe.isMoving()) {
                globe.jumpTo({
                    center: [center.lng, center.lat],
                    zoom: zoom
                });
            }
        });

        // Sync globe → map
        globe.on('move', () => {
            if (!this.syncEnabled) return;

            const center = globe.getCenter();
            const zoom = globe.getZoom();

            // Update map camera (debounced to avoid circular updates)
            if (map && !map.isMoving()) {
                map.jumpTo({
                    center: [center.lng, center.lat],
                    zoom: zoom
                });
            }
        });
    }

    /**
     * Update locations in both views
     */
    public updateLocations(locations: LocationStats[]): void {
        if (this.mapManager) {
            this.mapManager.updateLocations(locations);
        }
        if (this.globeManager) {
            this.globeManager.updateLocations(locations);
        }
    }

    /**
     * Update temporal data for globe
     */
    public updateTemporalData(data: TemporalHeatmapResponse): void {
        if (this.globeManager) {
            this.globeManager.updateTemporalData(data.buckets);
        }
    }

    /**
     * Set server location for globe arcs
     */
    public setServerLocation(serverInfo: ServerInfo): void {
        if (this.globeManager) {
            this.globeManager.setServerLocation(serverInfo);
        }
    }

    /**
     * Apply filters to both views
     */
    public applyFilter(_filter: LocationFilter): void {
        // Both views automatically get updated data through updateLocations()
        // No additional filter logic needed here
    }

    /**
     * Get map manager instance
     */
    public getMapManager(): MapManager | null {
        return this.mapManager;
    }

    /**
     * Get globe manager instance
     */
    public getGlobeManager(): GlobeManagerDeckGLEnhanced | null {
        return this.globeManager;
    }

    /**
     * Show the multi-view
     */
    public show(): void {
        const container = document.getElementById(this.containerId);
        if (container) {
            container.style.display = 'block';
        }

        // Resize both views
        setTimeout(() => {
            if (this.currentLayout !== 'globe-only' && this.mapManager) {
                this.mapManager.getMap()?.resize();
            }
            if (this.currentLayout !== 'map-only' && this.globeManager) {
                this.globeManager.show();
            }
        }, 100);
    }

    /**
     * Hide the multi-view
     */
    public hide(): void {
        const container = document.getElementById(this.containerId);
        if (container) {
            container.style.display = 'none';
        }

        if (this.globeManager) {
            this.globeManager.hide();
        }
    }

    /**
     * Destroy both views
     */
    public destroy(): void {
        if (this.globeManager) {
            this.globeManager.destroy();
        }
        // MapManager doesn't have destroy method yet
    }
}
