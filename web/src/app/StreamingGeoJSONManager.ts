// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * StreamingGeoJSONManager - Progressive loading for large datasets
 *
 * Features:
 * - Streams GeoJSON data using chunked transfer encoding
 * - Progressive map updates as data arrives
 * - Progress indicator during loading
 * - Memory-efficient handling of 100k+ locations
 * - Toggle for streaming vs regular loading
 */

import { createLogger } from '../lib/logger';
import type { API, LocationFilter } from '../lib/api';
import { queuedFetch } from '../lib/api/client';

const logger = createLogger('StreamingGeoJSONManager');

interface GeoJSONFeature {
  type: 'Feature';
  geometry: {
    type: 'Point';
    coordinates: [number, number];
  };
  properties: {
    country: string;
    city: string;
    playback_count: number;
  };
}

interface GeoJSONFeatureCollection {
  type: 'FeatureCollection';
  features: GeoJSONFeature[];
}

interface StreamingCallbacks {
  onProgress?: (loaded: number, estimatedTotal: number) => void;
  onFeaturesBatch?: (features: GeoJSONFeature[], totalLoaded: number) => void;
  onComplete?: (featureCollection: GeoJSONFeatureCollection) => void;
  onError?: (error: Error) => void;
}

export class StreamingGeoJSONManager {
  // @ts-expect-error - API instance stored for potential future use (streaming uses direct fetch for chunked transfer)
  private api: API;
  private isStreaming: boolean = false;
  private abortController: AbortController | null = null;
  private useStreamingThreshold: number = 5000; // Use streaming for >5k locations
  private toggleElement: HTMLElement | null = null;
  private statusElement: HTMLElement | null = null;

  // Event handler reference for cleanup
  private toggleChangeHandler: ((e: Event) => void) | null = null;

  constructor(api: API) {
    this.api = api;
  }

  /**
   * Initialize the streaming manager
   */
  init(): void {
    this.setupUI();
    logger.log('StreamingGeoJSONManager initialized');
  }

  /**
   * Set up UI elements
   */
  private setupUI(): void {
    const container = document.getElementById('streaming-geojson-controls');
    if (!container) {
      // Create controls if container exists
      return;
    }

    container.innerHTML = `
      <div class="streaming-toggle">
        <label class="streaming-toggle-label" for="streaming-toggle-checkbox">
          <input type="checkbox" id="streaming-toggle-checkbox" class="streaming-checkbox" checked />
          <span class="streaming-toggle-text">Use streaming for large datasets</span>
        </label>
        <span class="streaming-threshold-info">(>${this.useStreamingThreshold.toLocaleString()} locations)</span>
      </div>
      <div class="streaming-status" id="streaming-status" role="status" aria-live="polite"></div>
    `;

    this.toggleElement = document.getElementById('streaming-toggle-checkbox');
    this.statusElement = document.getElementById('streaming-status');

    // Set up toggle listener with stored reference for cleanup
    if (this.toggleElement) {
      this.toggleChangeHandler = (e: Event) => {
        const target = e.target as HTMLInputElement;
        logger.log(`Streaming ${target.checked ? 'enabled' : 'disabled'}`);
      };
      this.toggleElement.addEventListener('change', this.toggleChangeHandler);
    }
  }

  /**
   * Check if streaming should be used based on estimated location count
   */
  shouldUseStreaming(estimatedCount: number): boolean {
    if (!this.toggleElement) return estimatedCount > this.useStreamingThreshold;
    const checkbox = this.toggleElement as HTMLInputElement;
    return checkbox.checked && estimatedCount > this.useStreamingThreshold;
  }

  /**
   * Stream GeoJSON data from the server
   */
  async streamLocations(
    filter: LocationFilter,
    callbacks: StreamingCallbacks
  ): Promise<GeoJSONFeatureCollection> {
    if (this.isStreaming) {
      this.abort();
    }

    this.isStreaming = true;
    this.abortController = new AbortController();
    const features: GeoJSONFeature[] = [];

    try {
      this.updateStatus('Connecting...');

      // Build query string
      const params = new URLSearchParams();
      if (filter.start_date) params.append('start_date', filter.start_date);
      if (filter.end_date) params.append('end_date', filter.end_date);
      if (filter.days) params.append('days', filter.days.toString());

      const url = `/api/v1/stream/locations-geojson?${params.toString()}`;

      const response = await queuedFetch(url, {
        method: 'GET',
        headers: {
          'Accept': 'application/json',
        },
        signal: this.abortController.signal,
      });

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      // Check for streaming support
      if (!response.body) {
        throw new Error('ReadableStream not supported');
      }

      this.updateStatus('Receiving data...');

      const reader = response.body.getReader();
      const decoder = new TextDecoder();
      let buffer = '';
      let featuresLoaded = 0;
      const estimatedTotal = parseInt(response.headers.get('X-Total-Count') || '10000', 10);

      // Process the stream
      while (true) {
        const { done, value } = await reader.read();

        if (done) break;

        buffer += decoder.decode(value, { stream: true });

        // Try to parse features from buffer
        const parsedFeatures = this.parseStreamBuffer(buffer);
        if (parsedFeatures.features.length > 0) {
          features.push(...parsedFeatures.features);
          buffer = parsedFeatures.remainder;
          featuresLoaded = features.length;

          // Report progress
          if (callbacks.onProgress) {
            callbacks.onProgress(featuresLoaded, estimatedTotal);
          }
          if (callbacks.onFeaturesBatch) {
            callbacks.onFeaturesBatch(parsedFeatures.features, featuresLoaded);
          }

          this.updateStatus(`Loaded ${featuresLoaded.toLocaleString()} locations...`);
        }
      }

      // Final parse for any remaining content
      const finalParse = this.parseStreamBuffer(buffer, true);
      if (finalParse.features.length > 0) {
        features.push(...finalParse.features);
        if (callbacks.onFeaturesBatch) {
          callbacks.onFeaturesBatch(finalParse.features, features.length);
        }
      }

      const featureCollection: GeoJSONFeatureCollection = {
        type: 'FeatureCollection',
        features: features
      };

      this.updateStatus(`Complete: ${features.length.toLocaleString()} locations loaded`);

      if (callbacks.onComplete) {
        callbacks.onComplete(featureCollection);
      }

      return featureCollection;

    } catch (error) {
      if ((error as Error).name === 'AbortError') {
        this.updateStatus('Loading cancelled');
        throw new Error('Stream aborted');
      }

      this.updateStatus('Error loading data');
      if (callbacks.onError) {
        callbacks.onError(error as Error);
      }
      throw error;

    } finally {
      this.isStreaming = false;
      this.abortController = null;
    }
  }

  /**
   * Parse features from stream buffer
   */
  private parseStreamBuffer(buffer: string, isFinal: boolean = false): {
    features: GeoJSONFeature[];
    remainder: string;
  } {
    const features: GeoJSONFeature[] = [];

    // Look for complete feature objects
    // Features are separated by commas after the opening [
    let startIdx = 0;
    let braceCount = 0;
    let featureStart = -1;

    // Skip the opening {"type":"FeatureCollection","features":[
    const featuresArrayStart = buffer.indexOf('"features":[');
    if (featuresArrayStart !== -1) {
      startIdx = featuresArrayStart + 12;
    }

    for (let i = startIdx; i < buffer.length; i++) {
      const char = buffer[i];

      if (char === '{') {
        if (braceCount === 0) {
          featureStart = i;
        }
        braceCount++;
      } else if (char === '}') {
        braceCount--;
        if (braceCount === 0 && featureStart !== -1) {
          // Complete feature found
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

    // Calculate remainder (unparsed portion)
    let remainder = '';
    if (!isFinal && featureStart !== -1) {
      remainder = buffer.substring(featureStart);
    } else if (!isFinal) {
      // Keep the last unparsed portion
      const lastBrace = buffer.lastIndexOf('}');
      if (lastBrace !== -1 && lastBrace < buffer.length - 1) {
        remainder = buffer.substring(lastBrace + 1);
      }
    }

    return { features, remainder };
  }

  /**
   * Update status display
   */
  private updateStatus(message: string): void {
    if (this.statusElement) {
      this.statusElement.textContent = message;
    }
  }

  /**
   * Abort current streaming operation
   */
  abort(): void {
    if (this.abortController) {
      this.abortController.abort();
      this.abortController = null;
    }
    this.isStreaming = false;
    this.updateStatus('Cancelled');
  }

  /**
   * Check if currently streaming
   */
  isCurrentlyStreaming(): boolean {
    return this.isStreaming;
  }

  /**
   * Set the threshold for using streaming
   */
  setStreamingThreshold(threshold: number): void {
    this.useStreamingThreshold = threshold;
  }

  /**
   * Get the streaming threshold
   */
  getStreamingThreshold(): number {
    return this.useStreamingThreshold;
  }

  /**
   * Convert features to location stats format for map
   */
  featuresToLocationStats(features: GeoJSONFeature[]): Array<{
    country: string;
    city: string;
    latitude: number;
    longitude: number;
    playback_count: number;
    unique_users: number;
    first_seen: string;
    last_seen: string;
    avg_completion: number;
  }> {
    return features.map(feature => ({
      country: feature.properties.country || 'Unknown',
      city: feature.properties.city || 'Unknown',
      latitude: feature.geometry.coordinates[1],
      longitude: feature.geometry.coordinates[0],
      playback_count: feature.properties.playback_count || 0,
      unique_users: 1,
      first_seen: new Date().toISOString(),
      last_seen: new Date().toISOString(),
      avg_completion: 0
    }));
  }

  /**
   * Destroy the manager and cleanup resources
   */
  destroy(): void {
    // Abort any ongoing streaming
    this.abort();

    // Remove toggle event listener
    if (this.toggleElement && this.toggleChangeHandler) {
      this.toggleElement.removeEventListener('change', this.toggleChangeHandler);
      this.toggleChangeHandler = null;
    }

    // Clear element references
    this.toggleElement = null;
    this.statusElement = null;
  }
}

export default StreamingGeoJSONManager;
