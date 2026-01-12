// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
import * as echarts from 'echarts';
import type { LocationStats } from './api';
import { createLogger } from './logger';

const logger = createLogger('Globe');

// Note: echarts-gl must be loaded separately to avoid build issues with esbuild
// The globe functionality will be available after echarts-gl is loaded
declare global {
    interface Window {
        'echarts-gl': any;
    }
}

export class GlobeManager {
    private chart: echarts.ECharts | null = null;
    private containerId: string;
    private currentData: LocationStats[] = [];
    private isDarkTheme: boolean = true;
    private resizeHandler: (() => void) | null = null;

    constructor(containerId: string) {
        this.containerId = containerId;
    }

    /**
     * Initialize the 3D globe visualization
     */
    public initialize(): void {
        const container = document.getElementById(this.containerId);
        if (!container) {
            logger.error(`Container with id "${this.containerId}" not found`);
            return;
        }

        // Check theme
        this.isDarkTheme = !document.documentElement.hasAttribute('data-theme');

        // Initialize ECharts instance
        this.chart = echarts.init(container);

        // Set initial empty option
        this.updateGlobeVisualization([]);

        // Handle window resize - store reference for proper cleanup
        this.resizeHandler = () => this.handleResize();
        window.addEventListener('resize', this.resizeHandler);
    }

    /**
     * Update globe with location data
     */
    public updateLocations(locations: LocationStats[]): void {
        // Filter out invalid locations (0,0 coordinates indicate geolocation failures)
        const validLocations = locations.filter(location =>
            location.latitude !== 0 || location.longitude !== 0
        );

        this.currentData = validLocations;
        this.updateGlobeVisualization(validLocations);
    }

    /**
     * Update the globe visualization with new data
     */
    private updateGlobeVisualization(locations: LocationStats[]): void {
        if (!this.chart) return;

        // Prepare scatter data for globe
        const scatterData = locations.map(location => ({
            name: `${location.city || 'Unknown'}, ${location.country}`,
            value: [
                location.longitude,
                location.latitude,
                location.playback_count
            ],
            itemStyle: {
                color: this.getColorByPlaybackCount(location.playback_count),
                opacity: 0.9
            },
            location,
        }));

        const option: echarts.EChartsOption = {
            backgroundColor: this.isDarkTheme ? '#0a0e27' : '#f0f2f5',
            globe: {
                baseTexture: this.getGlobeBaseTexture(),
                heightTexture: this.getGlobeHeightTexture(),
                displacementScale: 0.04,
                shading: 'realistic',
                environment: this.getGlobeEnvironment(),
                realisticMaterial: {
                    roughness: 0.9,
                    metalness: 0.0
                },
                postEffect: {
                    enable: true,
                    bloom: {
                        enable: true,
                        intensity: 0.3
                    }
                },
                light: {
                    main: {
                        intensity: 2.0,
                        shadow: false
                    },
                    ambient: {
                        intensity: 0.3
                    }
                },
                viewControl: {
                    autoRotate: false,
                    autoRotateSpeed: 5,
                    rotateSensitivity: 1,
                    zoomSensitivity: 1,
                    panSensitivity: 1,
                    distance: 200,
                    minDistance: 100,
                    maxDistance: 400,
                    alpha: 30,
                    beta: 0,
                    center: [0, 0, 0]
                },
                layers: [
                    {
                        type: 'blend',
                        blendTo: 'emission',
                        texture: this.getGlobeNightTexture()
                    }
                ]
            },
            // Type assertion needed because scatter3D is from echarts-gl, not included in base echarts types
            series: [
                {
                    name: 'Playback Locations',
                    type: 'scatter3D',
                    coordinateSystem: 'globe',
                    blendMode: 'lighter',
                    symbolSize: (val: number[]) => {
                        const playbacks = val[2];
                        return Math.max(8, Math.min(30, Math.sqrt(playbacks) * 2));
                    },
                    itemStyle: {
                        opacity: 0.8,
                        borderWidth: 1,
                        borderColor: 'rgba(255, 255, 255, 0.5)'
                    },
                    emphasis: {
                        itemStyle: {
                            opacity: 1,
                            borderWidth: 2,
                            borderColor: '#fff'
                        },
                        label: {
                            show: true,
                            formatter: (params: any) => {
                                const location = params.data.location as LocationStats;
                                return `${params.data.name}\nPlaybacks: ${location.playback_count.toLocaleString()}\nUsers: ${location.unique_users}`;
                            },
                            textStyle: {
                                color: '#fff',
                                fontSize: 14,
                                backgroundColor: 'rgba(0, 0, 0, 0.7)',
                                padding: 8,
                                borderRadius: 4
                            }
                        }
                    },
                    data: scatterData
                }
            ] as any,
            tooltip: {
                show: true,
                formatter: (params: any) => {
                    const location = params.data.location as LocationStats;
                    if (!location) return '';

                    const locationName = location.region
                        ? `${location.city || 'Unknown'}, ${location.region}, ${location.country}`
                        : `${location.city || 'Unknown'}, ${location.country}`;

                    return `
                        <div style="padding: 8px;">
                            <div style="font-weight: 600; font-size: 14px; margin-bottom: 8px; color: #fff;">
                                ${locationName}
                            </div>
                            <div style="font-size: 12px; line-height: 1.8;">
                                <div><span style="color: #a0a0a0;">Playbacks:</span> <strong>${location.playback_count.toLocaleString()}</strong></div>
                                <div><span style="color: #a0a0a0;">Unique Users:</span> <strong>${location.unique_users}</strong></div>
                                <div><span style="color: #a0a0a0;">Avg Completion:</span> <strong>${location.avg_completion.toFixed(1)}%</strong></div>
                                ${location.first_seen ? `<div style="margin-top: 4px; padding-top: 4px; border-top: 1px solid rgba(255,255,255,0.1);"><span style="color: #a0a0a0;">First Seen:</span> ${new Date(location.first_seen).toLocaleDateString()}</div>` : ''}
                                ${location.last_seen ? `<div><span style="color: #a0a0a0;">Last Seen:</span> ${new Date(location.last_seen).toLocaleDateString()}</div>` : ''}
                            </div>
                        </div>
                    `;
                },
                backgroundColor: 'rgba(20, 20, 30, 0.95)',
                borderColor: '#4ecdc4',
                borderWidth: 1,
                textStyle: {
                    color: '#eaeaea',
                    fontSize: 12
                },
                padding: 0
            }
        };

        this.chart.setOption(option, true);
    }

    /**
     * Get color based on playback count
     */
    private getColorByPlaybackCount(count: number): string {
        if (count > 500) return '#e94560';
        if (count > 200) return '#ff6b6b';
        if (count > 50) return '#ffa500';
        return '#4ecdc4';
    }

    /**
     * Get globe base texture URL
     */
    private getGlobeBaseTexture(): string {
        // Using NASA Blue Marble texture
        return 'data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iMSIgaGVpZ2h0PSIxIiB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciPjxyZWN0IHdpZHRoPSIxIiBoZWlnaHQ9IjEiIGZpbGw9IiMxYTFhMmUiLz48L3N2Zz4=';
    }

    /**
     * Get globe height texture URL for topography
     */
    private getGlobeHeightTexture(): string {
        return '';
    }

    /**
     * Get globe environment texture
     */
    private getGlobeEnvironment(): string {
        // Using solid color as environment
        return this.isDarkTheme ? '#000' : '#fff';
    }

    /**
     * Get globe night texture (city lights)
     */
    private getGlobeNightTexture(): string {
        return '';
    }

    /**
     * Update theme
     */
    public updateTheme(isDark: boolean): void {
        this.isDarkTheme = isDark;
        this.updateGlobeVisualization(this.currentData);
    }

    /**
     * Handle window resize
     */
    private handleResize(): void {
        if (this.chart) {
            this.chart.resize();
        }
    }

    /**
     * Enable auto-rotation
     */
    public enableAutoRotate(): void {
        if (!this.chart) return;

        const option = this.chart.getOption();
        if (option.globe && Array.isArray(option.globe) && option.globe[0]) {
            (option.globe[0] as any).viewControl.autoRotate = true;
            this.chart.setOption(option);
        }
    }

    /**
     * Disable auto-rotation
     */
    public disableAutoRotate(): void {
        if (!this.chart) return;

        const option = this.chart.getOption();
        if (option.globe && Array.isArray(option.globe) && option.globe[0]) {
            (option.globe[0] as any).viewControl.autoRotate = false;
            this.chart.setOption(option);
        }
    }

    /**
     * Reset camera view
     */
    public resetView(): void {
        if (!this.chart) return;

        const option = this.chart.getOption();
        if (option.globe && Array.isArray(option.globe) && option.globe[0]) {
            const globeOption = option.globe[0] as any;
            globeOption.viewControl.distance = 200;
            globeOption.viewControl.alpha = 30;
            globeOption.viewControl.beta = 0;
            this.chart.setOption(option);
        }
    }

    /**
     * Destroy the globe instance
     */
    public destroy(): void {
        // Remove event listener using stored reference
        if (this.resizeHandler) {
            window.removeEventListener('resize', this.resizeHandler);
            this.resizeHandler = null;
        }
        if (this.chart) {
            this.chart.dispose();
            this.chart = null;
        }
    }

    /**
     * Show the globe container
     */
    public show(): void {
        const container = document.getElementById(this.containerId);
        if (container) {
            container.style.display = 'block';
            if (this.chart) {
                this.chart.resize();
            }
        }
    }

    /**
     * Hide the globe container
     */
    public hide(): void {
        const container = document.getElementById(this.containerId);
        if (container) {
            container.style.display = 'none';
        }
    }
}
