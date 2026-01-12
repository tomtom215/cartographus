// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * ViewportManager - Handles 2D/3D view switching and browser capability detection
 *
 * Manages:
 * - 2D map visualization (MapLibre GL)
 * - 3D globe visualization (deck.gl)
 * - Map visualization modes (points, clusters, heatmap)
 * - WebGL capability detection and fallbacks
 * - Globe controls (auto-rotate, reset view)
 *
 * Browser Support:
 * - WebGL 2.0: Best (all features)
 * - WebGL 1.0: Good (basic features)
 * - Canvas 2D: Limited (charts only, no maps)
 * - None: Critical (upgrade required)
 */

import type { MapManager } from '../lib/map';
import type { GlobeManagerDeckGL } from '../lib/globe-deckgl';
import type { ToastManager } from '../lib/toast';
import { createLogger } from '../lib/logger';
import { SafeStorage } from '../lib/utils/SafeStorage';

const logger = createLogger('ViewportManager');

export type ViewMode = '2d' | '3d';
export type MapMode = 'points' | 'clusters' | 'heatmap' | 'hexagons';

export class ViewportManager {
    private mapManager: MapManager;
    private globeManager: GlobeManagerDeckGL | null = null;
    private toastManager: ToastManager | null = null;
    private currentViewMode: ViewMode = '2d';
    private globeAutoRotateEnabled: boolean = false;
    private webglSupported: boolean = true;
    private webgl2Supported: boolean = false;
    private canvasSupported: boolean = true;

    constructor(mapManager: MapManager) {
        this.mapManager = mapManager;
        this.detectBrowserCapabilities();
    }

    /**
     * Set globe manager reference for 3D view
     */
    setGlobeManager(globeManager: GlobeManagerDeckGL | null): void {
        this.globeManager = globeManager;
    }

    /**
     * Set toast manager for user notifications
     */
    setToastManager(toastManager: ToastManager | null): void {
        this.toastManager = toastManager;
    }

    /**
     * Get current view mode
     */
    getCurrentViewMode(): ViewMode {
        return this.currentViewMode;
    }

    /**
     * Check if WebGL is supported
     */
    isWebGLSupported(): boolean {
        return this.webglSupported;
    }

    /**
     * Detect browser rendering capabilities with granular feature detection
     * This allows graceful degradation: disable 3D globe but keep 2D map working
     */
    private detectBrowserCapabilities(): void {
        // Detect Canvas 2D support (fallback for non-WebGL charts)
        try {
            const canvas = document.createElement('canvas');
            this.canvasSupported = !!(canvas.getContext && canvas.getContext('2d'));
        } catch (e) {
            this.canvasSupported = false;
            logger.warn('[Canvas] Canvas 2D not supported:', e);
        }

        // Detect WebGL 1.0 support (required for MapLibre GL and deck.gl)
        try {
            const canvas = document.createElement('canvas');
            const gl = canvas.getContext('webgl') || canvas.getContext('experimental-webgl');
            this.webglSupported = !!gl;

            if (gl) {
                // Cast to WebGLRenderingContext for TypeScript
                const webglContext = gl as WebGLRenderingContext;

                // Check WebGL capabilities for diagnostics
                const maxTextureSize = webglContext.getParameter(webglContext.MAX_TEXTURE_SIZE);
                const maxRenderbufferSize = webglContext.getParameter(webglContext.MAX_RENDERBUFFER_SIZE);
                const vendor = webglContext.getParameter(webglContext.VENDOR);
                const renderer = webglContext.getParameter(webglContext.RENDERER);

                logger.debug('[WebGL 1.0] Supported', {
                    maxTextureSize: `${maxTextureSize}px`,
                    maxRenderbufferSize: `${maxRenderbufferSize}px`,
                    vendor,
                    renderer
                });

                // Check for important extensions
                const extensions = {
                    floatTextures: !!webglContext.getExtension('OES_texture_float'),
                    anisotropicFiltering: !!webglContext.getExtension('EXT_texture_filter_anisotropic'),
                    depthTexture: !!webglContext.getExtension('WEBGL_depth_texture')
                };
                logger.debug('[WebGL] Extensions:', extensions);
            }
        } catch (e) {
            this.webglSupported = false;
            logger.warn('[WebGL] WebGL 1.0 detection failed:', e);
        }

        // Detect WebGL 2.0 support (enhanced features for deck.gl)
        try {
            const canvas = document.createElement('canvas');
            const gl2 = canvas.getContext('webgl2');
            this.webgl2Supported = !!gl2;
            if (gl2) {
                logger.debug('[WebGL 2.0] Supported - Enhanced deck.gl features available');
            }
        } catch (e) {
            this.webgl2Supported = false;
        }

        // Log final capabilities summary
        logger.debug('[Browser Capabilities]', {
            canvas2D: this.canvasSupported,
            webgl: this.webglSupported,
            webgl2: this.webgl2Supported,
            recommendation: this.getRendererRecommendation()
        });

        // Show warning if critical features are missing
        if (!this.canvasSupported) {
            logger.error('[Critical] Canvas 2D not supported - charts may not render');
        }
    }

    /**
     * Get renderer recommendation based on detected capabilities
     */
    private getRendererRecommendation(): string {
        if (this.webgl2Supported) return 'WebGL 2.0 (Best)';
        if (this.webglSupported) return 'WebGL 1.0 (Good)';
        if (this.canvasSupported) return 'Canvas 2D (Limited - no maps)';
        return 'None (Critical - upgrade browser)';
    }

    /**
     * Set map visualization mode (points, clusters, heatmap, hexagons)
     */
    setMapMode(mode: MapMode): void {
        if (this.mapManager) {
            try {
                this.mapManager.setVisualizationMode(mode);
            } catch (error) {
                // Log error but continue to update button state
                // The mode may have been set before the error occurred
                logger.error('[ViewportManager] Error setting visualization mode:', error);
            }
            // Always update button states to ensure UI reflects the change
            // This is more reliable than relying solely on the callback chain
            // and handles edge cases where setVisualizationMode partially succeeds
            this.updateMapModeButtons();
        }
    }

    /**
     * Called when map mode changes to update UI buttons
     */
    onMapModeChange(): void {
        this.updateMapModeButtons();
    }

    /**
     * Update map mode button states
     */
    private updateMapModeButtons(): void {
        if (!this.mapManager) return;

        const currentMode = this.mapManager.getVisualizationMode();
        const btnPoints = document.getElementById('map-mode-points') as HTMLButtonElement;
        const btnClusters = document.getElementById('map-mode-clusters') as HTMLButtonElement;
        const btnHeatmap = document.getElementById('map-mode-heatmap') as HTMLButtonElement;
        const btnHexagons = document.getElementById('map-mode-hexagons') as HTMLButtonElement;
        const hexagonResolutionControl = document.getElementById('hexagon-resolution-control');

        if (btnPoints) {
            btnPoints.classList.toggle('active', currentMode === 'points');
            btnPoints.setAttribute('aria-pressed', String(currentMode === 'points'));
        }

        if (btnClusters) {
            btnClusters.classList.toggle('active', currentMode === 'clusters');
            btnClusters.setAttribute('aria-pressed', String(currentMode === 'clusters'));
        }

        if (btnHeatmap) {
            btnHeatmap.classList.toggle('active', currentMode === 'heatmap');
            btnHeatmap.setAttribute('aria-pressed', String(currentMode === 'heatmap'));
        }

        // H3 Hexagon mode button
        if (btnHexagons) {
            btnHexagons.classList.toggle('active', currentMode === 'hexagons');
            btnHexagons.setAttribute('aria-pressed', String(currentMode === 'hexagons'));
        }

        // Show/hide hexagon resolution control based on mode
        if (hexagonResolutionControl) {
            hexagonResolutionControl.style.display = currentMode === 'hexagons' ? 'flex' : 'none';
        }
    }

    /**
     * Set view mode (2D map or 3D globe)
     */
    setViewMode(mode: ViewMode): void {
        this.currentViewMode = mode;
        SafeStorage.setItem('view-mode', mode);

        const mapContainer = document.getElementById('map');
        const globeContainer = document.getElementById('globe');
        const mapModeControl = document.getElementById('map-mode-control');

        if (mode === '2d') {
            // Hide globe first to cancel any pending operations
            if (this.globeManager) {
                this.globeManager.hide();
            }
            if (mapContainer) mapContainer.style.display = 'block';
            if (globeContainer) globeContainer.style.display = 'none';
            if (mapModeControl) mapModeControl.style.display = 'flex';
        } else {
            if (mapContainer) mapContainer.style.display = 'none';
            if (globeContainer) globeContainer.style.display = 'block';
            if (mapModeControl) mapModeControl.style.display = 'none';

            // Trigger resize after switching to globe
            if (this.globeManager) {
                setTimeout(() => {
                    if (this.globeManager) {
                        this.globeManager.show();
                    }
                }, 50);
            }
        }

        this.updateViewModeButtons();
    }

    /**
     * Update view mode button states (2D/3D)
     */
    private updateViewModeButtons(): void {
        const btn2D = document.getElementById('view-mode-2d') as HTMLButtonElement;
        const btn3D = document.getElementById('view-mode-3d') as HTMLButtonElement;

        if (btn2D) {
            btn2D.classList.toggle('active', this.currentViewMode === '2d');
            btn2D.setAttribute('aria-pressed', String(this.currentViewMode === '2d'));
        }

        if (btn3D) {
            btn3D.classList.toggle('active', this.currentViewMode === '3d');
            btn3D.setAttribute('aria-pressed', String(this.currentViewMode === '3d'));
        }
    }

    /**
     * Toggle globe auto-rotate
     */
    toggleGlobeAutoRotate(): void {
        if (!this.globeManager) return;

        this.globeAutoRotateEnabled = !this.globeAutoRotateEnabled;

        if (this.globeAutoRotateEnabled) {
            this.globeManager.enableAutoRotate();
        } else {
            this.globeManager.disableAutoRotate();
        }

        const btn = document.getElementById('globe-auto-rotate') as HTMLButtonElement;
        if (btn) {
            btn.classList.toggle('active', this.globeAutoRotateEnabled);
        }
    }

    /**
     * Reset globe view to default position
     */
    resetGlobeView(): void {
        if (!this.globeManager) return;
        this.globeManager.resetView();
    }

    /**
     * Check WebGL support and provide granular fallback messages
     * Gracefully disables 3D globe but keeps 2D map and charts working if possible
     */
    checkWebGLForMaps(): void {
        const webglNotSupported = document.getElementById('webgl-not-supported');
        const globeNotSupported = document.getElementById('globe-not-supported');
        const view3DButton = document.getElementById('view-mode-3d') as HTMLButtonElement;

        if (!this.webglSupported) {
            // CRITICAL: WebGL not supported - disable all map features
            if (this.currentViewMode === '2d' && webglNotSupported) {
                webglNotSupported.style.display = 'block';
            }

            if (this.currentViewMode === '3d' && globeNotSupported) {
                globeNotSupported.style.display = 'block';
            }

            // Disable 3D globe button completely
            if (view3DButton) {
                view3DButton.disabled = true;
                view3DButton.title = '3D Globe requires WebGL support (not available)';
                view3DButton.style.opacity = '0.5';
                view3DButton.style.cursor = 'not-allowed';
            }

            // Force switch to 2D if user is on 3D mode
            if (this.currentViewMode === '3d') {
                logger.warn('[WebGL] 3D Globe disabled - switching to 2D map view');
                this.currentViewMode = '2d';
                this.updateViewModeButtons();
            }

            // Show contextual toast based on what's affected
            if (this.toastManager) {
                const message = this.canvasSupported
                    ? 'Maps require WebGL support. Analytics charts will still work with Canvas 2D.'
                    : 'WebGL and Canvas not supported. Please upgrade your browser.';

                this.toastManager.warning(
                    message,
                    'Limited Browser Support',
                    10000
                );
            }

            logger.warn('[WebGL Fallback]', {
                mapsDisabled: true,
                globeDisabled: true,
                chartsWorking: this.canvasSupported,
                recommendation: 'Upgrade to Chrome 90+, Firefox 88+, Edge 90+, or Safari 14+'
            });
        } else {
            // WebGL is supported - hide warnings and enable features
            if (webglNotSupported) webglNotSupported.style.display = 'none';
            if (globeNotSupported) globeNotSupported.style.display = 'none';

            // Re-enable 3D globe button if it was disabled
            if (view3DButton) {
                view3DButton.disabled = false;
                view3DButton.title = '3D Globe View';
                view3DButton.style.opacity = '1';
                view3DButton.style.cursor = 'pointer';
            }

            // Log successful WebGL initialization
            if (this.webgl2Supported) {
                logger.debug('[WebGL] All features enabled with WebGL 2.0');
            } else {
                logger.debug('[WebGL] All features enabled with WebGL 1.0');
            }
        }
    }

    /**
     * Cleanup resources to prevent memory leaks
     * Releases manager references
     */
    destroy(): void {
        // Clear manager references
        this.globeManager = null;
        this.toastManager = null;
    }
}
