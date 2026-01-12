// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Spatial API Module
 *
 * H3 hexagon aggregation, arc visualization, viewport queries,
 * and streaming GeoJSON endpoints.
 */

import type {
    LocationStats,
    LocationFilter,
    TemporalSpatialPoint,
    GeoJSONFeature,
    StreamingGeoJSONResponse,
} from '../types/core';
import type { H3HexagonStats, ArcStats } from '../types/visualization';
import { BaseAPIClient, queuedFetch } from './client';

/**
 * Spatial API methods
 */
export class SpatialAPI extends BaseAPIClient {
    /**
     * Get H3 hexagon aggregated data
     */
    async getH3HexagonData(filter: LocationFilter = {}, resolution: number = 4): Promise<H3HexagonStats[]> {
        const params = this.buildFilterParams(filter);
        params.append('resolution', resolution.toString());
        const queryString = params.toString();
        const url = `/spatial/hexagons?${queryString}`;

        const response = await this.fetch<H3HexagonStats[]>(url);
        return response.data;
    }

    /**
     * Get arc data for server-to-user connections
     */
    async getArcData(filter: LocationFilter = {}): Promise<ArcStats[]> {
        const params = this.buildFilterParams(filter);
        const queryString = params.toString();
        const url = queryString ? `/spatial/arcs?${queryString}` : '/spatial/arcs';

        const response = await this.fetch<ArcStats[]>(url);
        return response.data;
    }

    /**
     * Get locations within a viewport bounding box
     */
    async getSpatialViewport(
        bbox: { west: number; south: number; east: number; north: number },
        filter: LocationFilter = {}
    ): Promise<LocationStats[]> {
        const params = this.buildFilterParams(filter);
        params.append('west', bbox.west.toString());
        params.append('south', bbox.south.toString());
        params.append('east', bbox.east.toString());
        params.append('north', bbox.north.toString());
        const queryString = params.toString();
        const url = `/spatial/viewport?${queryString}`;

        const response = await this.fetch<LocationStats[]>(url);
        return response.data;
    }

    /**
     * Get temporal-spatial density data for animated heatmaps
     */
    async getSpatialTemporalDensity(
        interval: 'hour' | 'day' | 'week' | 'month' = 'hour',
        resolution: number = 7,
        filter: LocationFilter = {}
    ): Promise<TemporalSpatialPoint[]> {
        const params = this.buildFilterParams(filter);
        params.append('interval', interval);
        params.append('resolution', resolution.toString());
        const queryString = params.toString();
        const url = `/spatial/temporal-density?${queryString}`;

        const response = await this.fetch<TemporalSpatialPoint[]>(url);
        return response.data;
    }

    /**
     * Get locations near a specific point
     */
    async getSpatialNearby(
        lat: number,
        lon: number,
        radius: number = 100,
        filter: LocationFilter = {}
    ): Promise<LocationStats[]> {
        const params = this.buildFilterParams(filter);
        params.append('lat', lat.toString());
        params.append('lon', lon.toString());
        params.append('radius', radius.toString());
        const queryString = params.toString();
        const url = `/spatial/nearby?${queryString}`;

        const response = await this.fetch<LocationStats[]>(url);
        return response.data;
    }

    /**
     * Get streaming GeoJSON URL for large datasets
     */
    getStreamingGeoJSONUrl(filter: LocationFilter = {}): string {
        const params = this.buildFilterParams(filter);
        const queryString = params.toString();
        return queryString
            ? `${this.baseURL}/stream/locations-geojson?${queryString}`
            : `${this.baseURL}/stream/locations-geojson`;
    }

    /**
     * Stream GeoJSON locations with progress callback
     */
    async streamLocationsGeoJSON(
        filter: LocationFilter = {},
        onProgress?: (loaded: number, total: number) => void,
        signal?: AbortSignal
    ): Promise<StreamingGeoJSONResponse> {
        const url = this.getStreamingGeoJSONUrl(filter);

        const response = await queuedFetch(url, {
            method: 'GET',
            headers: {
                'Accept': 'application/json',
            },
            signal,
        });

        if (!response.ok) {
            const errorText = await response.text();
            throw new Error(`Streaming GeoJSON failed: ${response.status} ${errorText}`);
        }

        if (!response.body) {
            throw new Error('ReadableStream not supported');
        }

        const reader = response.body.getReader();
        const decoder = new TextDecoder();
        let buffer = '';
        const features: GeoJSONFeature[] = [];
        const estimatedTotal = parseInt(response.headers.get('X-Total-Count') || '10000', 10);

        try {
            while (true) {
                const { done, value } = await reader.read();
                if (done) break;

                buffer += decoder.decode(value, { stream: true });

                const parseResult = this.parseGeoJSONBuffer(buffer);
                if (parseResult.features.length > 0) {
                    features.push(...parseResult.features);
                    buffer = parseResult.remainder;

                    if (onProgress) {
                        onProgress(features.length, estimatedTotal);
                    }
                }
            }

            const finalResult = this.parseGeoJSONBuffer(buffer, true);
            features.push(...finalResult.features);

            return {
                type: 'FeatureCollection',
                features,
                totalLoaded: features.length,
            };
        } catch (error) {
            if ((error as Error).name === 'AbortError') {
                throw new Error('Streaming aborted by user');
            }
            throw error;
        }
    }

    /**
     * Parse GeoJSON features from a streaming buffer
     */
    private parseGeoJSONBuffer(buffer: string, isFinal: boolean = false): {
        features: GeoJSONFeature[];
        remainder: string;
    } {
        const features: GeoJSONFeature[] = [];
        let startIdx = 0;
        let braceCount = 0;
        let featureStart = -1;

        const featuresArrayStart = buffer.indexOf('"features":[');
        if (featuresArrayStart !== -1) {
            startIdx = featuresArrayStart + 12;
        }

        for (let i = startIdx; i < buffer.length; i++) {
            const char = buffer[i];
            if (char === '{') {
                if (braceCount === 0) featureStart = i;
                braceCount++;
            } else if (char === '}') {
                braceCount--;
                if (braceCount === 0 && featureStart !== -1) {
                    const featureStr = buffer.substring(featureStart, i + 1);
                    try {
                        const feature = JSON.parse(featureStr) as GeoJSONFeature;
                        if (feature.type === 'Feature' && feature.geometry) {
                            features.push(feature);
                        }
                    } catch {
                        // Incomplete or malformed JSON, skip
                    }
                    featureStart = -1;
                }
            }
        }

        let remainder = '';
        if (!isFinal && featureStart !== -1) {
            remainder = buffer.substring(featureStart);
        }

        return { features, remainder };
    }
}
