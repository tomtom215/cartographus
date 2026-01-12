// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
import type {GlobeManagerDeckGLEnhanced} from './globe-deckgl-enhanced';
import type {TemporalHeatmapResponse} from './api';
import { createLogger } from './logger';

const logger = createLogger('GlobeControls');

/**
 * GlobeControls - UI control panel for enhanced globe visualization
 *
 * Provides:
 * - Layer toggle buttons (scatterplot, hexagon, arcs, trips)
 * - Animation controls (play/pause, speed, scrubber)
 * - Screenshot button
 * - Reset view button
 */
export class GlobeControls {
    private globe: GlobeManagerDeckGLEnhanced;
    private controlsContainer: HTMLElement | null = null;
    private temporalData: TemporalHeatmapResponse | null = null;
    private currentTimeIndex: number = 0;
    private animationInterval: number | null = null;
    private animationSpeed: number = 1000; // ms per frame

    constructor(globe: GlobeManagerDeckGLEnhanced, containerId: string) {
        this.globe = globe;
        this.controlsContainer = document.getElementById(containerId);

        if (!this.controlsContainer) {
            logger.error(`Globe controls container "${containerId}" not found`);
            return;
        }

        this.createControls();
    }

    /**
     * Create the control panel UI
     */
    private createControls(): void {
        if (!this.controlsContainer) return;

        const controlsHTML = `
            <div class="globe-controls" style="
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
                min-width: 250px;
                box-shadow: 0 4px 12px rgba(0, 0, 0, 0.4);
            ">
                <div style="font-weight: 600; margin-bottom: 12px; color: #4ecdc4; font-size: 14px;">
                    🌍 Globe Visualization
                </div>

                <!-- Layer Toggles -->
                <div class="layer-toggles" style="margin-bottom: 16px;">
                    <div style="font-weight: 500; margin-bottom: 8px; font-size: 12px; color: #a0a0a0;">
                        VISUALIZATION LAYERS
                    </div>
                    <label style="display: flex; align-items: center; margin-bottom: 6px; cursor: pointer;">
                        <input type="checkbox" id="toggle-scatterplot" checked style="margin-right: 8px; cursor: pointer;">
                        <span>Scatterplot Points</span>
                    </label>
                    <label style="display: flex; align-items: center; margin-bottom: 6px; cursor: pointer;">
                        <input type="checkbox" id="toggle-hexagon" style="margin-right: 8px; cursor: pointer;">
                        <span>3D Hexagon Aggregation</span>
                    </label>
                    <label style="display: flex; align-items: center; margin-bottom: 6px; cursor: pointer;">
                        <input type="checkbox" id="toggle-arcs" style="margin-right: 8px; cursor: pointer;">
                        <span>User → Server Arcs</span>
                    </label>
                    <label style="display: flex; align-items: center; margin-bottom: 6px; cursor: pointer;">
                        <input type="checkbox" id="toggle-trips" style="margin-right: 8px; cursor: pointer;">
                        <span>Temporal Animation</span>
                    </label>
                </div>

                <!-- Animation Controls (hidden by default) -->
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
                        ">▶ Play</button>
                        <button id="btn-stop" style="
                            padding: 6px 12px;
                            background: #e94560;
                            color: white;
                            border: none;
                            border-radius: 4px;
                            font-weight: 600;
                            cursor: pointer;
                            font-size: 12px;
                        ">■</button>
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
                <div class="action-buttons" style="display: flex; gap: 8px;">
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
                    ">📸 Screenshot</button>
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
                    ">↻ Reset View</button>
                </div>
            </div>
        `;

        const controlsElement = document.createElement('div');
        controlsElement.innerHTML = controlsHTML;
        this.controlsContainer.appendChild(controlsElement.firstElementChild!);

        this.attachEventListeners();
    }

    /**
     * Attach event listeners to control buttons
     */
    private attachEventListeners(): void {
        // Layer toggles
        const scatterplotToggle = document.getElementById('toggle-scatterplot') as HTMLInputElement;
        const hexagonToggle = document.getElementById('toggle-hexagon') as HTMLInputElement;
        const arcsToggle = document.getElementById('toggle-arcs') as HTMLInputElement;
        const tripsToggle = document.getElementById('toggle-trips') as HTMLInputElement;

        scatterplotToggle?.addEventListener('change', (e) => {
            this.globe.toggleLayer('scatterplot', (e.target as HTMLInputElement).checked);
        });

        hexagonToggle?.addEventListener('change', (e) => {
            this.globe.toggleLayer('hexagon', (e.target as HTMLInputElement).checked);
        });

        arcsToggle?.addEventListener('change', (e) => {
            this.globe.toggleLayer('arcs', (e.target as HTMLInputElement).checked);
        });

        tripsToggle?.addEventListener('change', (e) => {
            const enabled = (e.target as HTMLInputElement).checked;
            this.globe.toggleLayer('trips', enabled);

            // Show/hide animation controls
            const animControls = document.getElementById('animation-controls');
            if (animControls) {
                animControls.style.display = enabled ? 'block' : 'none';
            }
        });

        // Animation controls
        const playPauseBtn = document.getElementById('btn-play-pause');
        const stopBtn = document.getElementById('btn-stop');
        const speedSlider = document.getElementById('speed-slider') as HTMLInputElement;
        const timeSlider = document.getElementById('time-slider') as HTMLInputElement;

        playPauseBtn?.addEventListener('click', () => {
            this.togglePlayPause();
        });

        stopBtn?.addEventListener('click', () => {
            this.stopAnimation();
        });

        speedSlider?.addEventListener('input', (e) => {
            const speed = parseFloat((e.target as HTMLInputElement).value);
            this.setAnimationSpeed(speed);

            const speedValue = document.getElementById('speed-value');
            if (speedValue) {
                speedValue.textContent = `${speed.toFixed(1)}x`;
            }
        });

        timeSlider?.addEventListener('input', (e) => {
            const index = parseInt((e.target as HTMLInputElement).value);
            this.scrubToTime(index);
        });

        // Action buttons
        const screenshotBtn = document.getElementById('btn-screenshot');
        const resetViewBtn = document.getElementById('btn-reset-view');

        screenshotBtn?.addEventListener('click', () => {
            this.takeScreenshot();
        });

        resetViewBtn?.addEventListener('click', () => {
            this.globe.resetView();
        });
    }

    /**
     * Set temporal data for animation
     */
    public setTemporalData(data: TemporalHeatmapResponse): void {
        this.temporalData = data;

        if (data.buckets.length > 0) {
            this.globe.updateTemporalData(data.buckets);

            // Show timeline scrubber
            const timelineScrubber = document.getElementById('timeline-scrubber');
            const timeSlider = document.getElementById('time-slider') as HTMLInputElement;

            if (timelineScrubber) {
                timelineScrubber.style.display = 'block';
            }

            if (timeSlider) {
                timeSlider.max = (data.buckets.length - 1).toString();
                timeSlider.value = '0';
            }

            this.updateTimeDisplay();
        }
    }

    /**
     * Toggle play/pause animation
     */
    private togglePlayPause(): void {
        const playPauseBtn = document.getElementById('btn-play-pause');
        if (!playPauseBtn) return;

        if (this.animationInterval) {
            // Pause
            this.pauseAnimation();
            playPauseBtn.textContent = '▶ Play';
        } else {
            // Play
            this.playAnimation();
            playPauseBtn.textContent = '⏸ Pause';
        }
    }

    /**
     * Start animation
     */
    private playAnimation(): void {
        if (!this.temporalData || this.temporalData.buckets.length === 0) return;

        this.globe.startAnimation();

        this.animationInterval = window.setInterval(() => {
            this.currentTimeIndex++;

            if (this.currentTimeIndex >= this.temporalData!.buckets.length) {
                this.currentTimeIndex = 0;
            }

            const bucket = this.temporalData!.buckets[this.currentTimeIndex];
            const timestamp = new Date(bucket.start_time).getTime();
            this.globe.setAnimationTime(timestamp);

            this.updateTimeDisplay();

            const timeSlider = document.getElementById('time-slider') as HTMLInputElement;
            if (timeSlider) {
                timeSlider.value = this.currentTimeIndex.toString();
            }
        }, this.animationSpeed);
    }

    /**
     * Pause animation
     */
    private pauseAnimation(): void {
        if (this.animationInterval) {
            clearInterval(this.animationInterval);
            this.animationInterval = null;
        }
        this.globe.stopAnimation();
    }

    /**
     * Stop animation and reset
     */
    private stopAnimation(): void {
        this.pauseAnimation();
        this.currentTimeIndex = 0;

        const playPauseBtn = document.getElementById('btn-play-pause');
        if (playPauseBtn) {
            playPauseBtn.textContent = '▶ Play';
        }

        const timeSlider = document.getElementById('time-slider') as HTMLInputElement;
        if (timeSlider) {
            timeSlider.value = '0';
        }

        if (this.temporalData && this.temporalData.buckets.length > 0) {
            const bucket = this.temporalData.buckets[0];
            const timestamp = new Date(bucket.start_time).getTime();
            this.globe.setAnimationTime(timestamp);
        }

        this.updateTimeDisplay();
    }

    /**
     * Set animation speed multiplier
     */
    private setAnimationSpeed(multiplier: number): void {
        this.animationSpeed = 1000 / multiplier;

        // Restart animation if currently playing
        if (this.animationInterval) {
            this.pauseAnimation();
            this.playAnimation();
        }
    }

    /**
     * Scrub to specific time index
     */
    private scrubToTime(index: number): void {
        if (!this.temporalData || index >= this.temporalData.buckets.length) return;

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
     * Take screenshot of the globe
     */
    private takeScreenshot(): void {
        // Get the canvas element from the globe container
        const globeContainer = document.getElementById('globe');
        if (!globeContainer) return;

        const canvas = globeContainer.querySelector('canvas');
        if (!canvas) {
            logger.error('Canvas not found for screenshot');
            return;
        }

        try {
            // Convert canvas to blob
            canvas.toBlob((blob) => {
                if (!blob) return;

                // Create download link
                const url = URL.createObjectURL(blob);
                const link = document.createElement('a');
                link.href = url;
                link.download = `cartographus-globe-${Date.now()}.png`;
                link.click();

                // Clean up
                URL.revokeObjectURL(url);
            }, 'image/png');
        } catch (error) {
            logger.error('Screenshot failed:', error);
        }
    }

    /**
     * Destroy controls and cleanup
     */
    public destroy(): void {
        if (this.animationInterval) {
            clearInterval(this.animationInterval);
        }

        if (this.controlsContainer) {
            this.controlsContainer.innerHTML = '';
        }
    }
}
