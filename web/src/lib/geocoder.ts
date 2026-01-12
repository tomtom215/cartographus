// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Nominatim Geocoder for MapLibre GL JS
 *
 * A free, OpenStreetMap-based geocoder that requires no API keys.
 * Drop-in replacement for Mapbox Geocoder with similar UI and behavior.
 *
 * Features:
 * - Search for locations by name, address, or coordinates
 * - Autocomplete suggestions with debouncing
 * - Fly-to animation on result selection
 * - Reverse geocoding support
 * - Customizable UI styling
 * - No API key required
 *
 * @module geocoder
 */

import type { Map as MapLibreMap, LngLatLike } from 'maplibre-gl';
import { createLogger } from './logger';

const logger = createLogger('Geocoder');

/**
 * Nominatim search result
 */
export interface NominatimResult {
    place_id: number;
    osm_type: string;
    osm_id: number;
    lat: string;
    lon: string;
    display_name: string;
    type: string;
    importance: number;
    boundingbox?: [string, string, string, string];
    address?: {
        city?: string;
        town?: string;
        village?: string;
        county?: string;
        state?: string;
        country?: string;
        country_code?: string;
    };
}

/**
 * Geocoder configuration options
 */
export interface GeocoderOptions {
    /** Nominatim server URL (default: OpenStreetMap's public server) */
    nominatimUrl?: string;

    /** Placeholder text for the search input */
    placeholder?: string;

    /** Minimum characters before searching */
    minLength?: number;

    /** Debounce delay in milliseconds */
    debounceMs?: number;

    /** Maximum number of results to show */
    maxResults?: number;

    /** Zoom level to fly to on result selection */
    flyToZoom?: number;

    /** Animation duration in milliseconds */
    flyToDuration?: number;

    /** Limit search to specific country codes (ISO 3166-1 alpha-2) */
    countryCodes?: string[];

    /** Bias results towards a location [lng, lat] */
    proximity?: [number, number];

    /** Show a marker at the selected location */
    showMarker?: boolean;

    /** Marker color (CSS color string) */
    markerColor?: string;

    /** Callback when a result is selected */
    onResult?: (result: NominatimResult) => void;

    /** Callback when search is cleared */
    onClear?: () => void;

    /** CSS class prefix for styling */
    cssPrefix?: string;

    /** Alias for maxResults (for compatibility) */
    limit?: number;

    /** Fly-to options for result selection */
    flyToOptions?: {
        speed?: number;
        zoom?: number;
    };
}

/**
 * Default geocoder options
 */
const DEFAULT_OPTIONS: Required<GeocoderOptions> = {
    nominatimUrl: 'https://nominatim.openstreetmap.org',
    placeholder: 'Search locations...',
    minLength: 3,
    debounceMs: 300,
    maxResults: 5,
    flyToZoom: 14,
    flyToDuration: 2000,
    countryCodes: [],
    proximity: [0, 0],
    showMarker: true,
    markerColor: '#4ecdc4',
    onResult: () => {},
    onClear: () => {},
    cssPrefix: 'geocoder',
    limit: 5,
    flyToOptions: { speed: 1.2, zoom: 14 },
};

/**
 * Nominatim Geocoder Control for MapLibre GL JS
 *
 * Usage:
 * ```typescript
 * const geocoder = new NominatimGeocoder({
 *     placeholder: 'Search for a city...',
 *     flyToZoom: 12,
 *     onResult: (result) => console.log('Selected:', result),
 * });
 *
 * map.addControl(geocoder, 'top-left');
 * ```
 */
export class NominatimGeocoder {
    private options: Required<GeocoderOptions>;
    private container: HTMLDivElement | null = null;
    private input: HTMLInputElement | null = null;
    private suggestionsContainer: HTMLDivElement | null = null;
    private clearButton: HTMLButtonElement | null = null;
    private map: MapLibreMap | null = null;
    private marker: any = null; // MapLibre Marker
    private debounceTimer: number | null = null;
    private abortController: AbortController | null = null;

    // Event handler references for proper cleanup
    private inputHandler: (() => void) | null = null;
    private focusHandler: (() => void) | null = null;
    private keydownHandler: ((e: KeyboardEvent) => void) | null = null;
    private clearHandler: (() => void) | null = null;
    private documentClickHandler: ((e: MouseEvent) => void) | null = null;

    constructor(options: GeocoderOptions = {}) {
        this.options = { ...DEFAULT_OPTIONS, ...options };
    }

    /**
     * IControl interface - called when added to map
     */
    onAdd(map: MapLibreMap): HTMLElement {
        this.map = map;
        this.container = this.createContainer();
        this.injectStyles();
        return this.container;
    }

    /**
     * IControl interface - called when removed from map
     */
    onRemove(): void {
        // Cancel any pending operations
        if (this.debounceTimer) {
            clearTimeout(this.debounceTimer);
            this.debounceTimer = null;
        }
        if (this.abortController) {
            this.abortController.abort();
            this.abortController = null;
        }

        // Remove event listeners from input
        if (this.input) {
            if (this.inputHandler) {
                this.input.removeEventListener('input', this.inputHandler);
            }
            if (this.focusHandler) {
                this.input.removeEventListener('focus', this.focusHandler);
            }
            if (this.keydownHandler) {
                this.input.removeEventListener('keydown', this.keydownHandler);
            }
        }

        // Remove event listener from clear button
        if (this.clearButton && this.clearHandler) {
            this.clearButton.removeEventListener('click', this.clearHandler);
        }

        // Remove document click listener
        if (this.documentClickHandler) {
            document.removeEventListener('click', this.documentClickHandler);
            this.documentClickHandler = null;
        }

        // Clear handler references
        this.inputHandler = null;
        this.focusHandler = null;
        this.keydownHandler = null;
        this.clearHandler = null;

        // Clean up marker and container
        this.clearMarker();
        if (this.container?.parentNode) {
            this.container.parentNode.removeChild(this.container);
        }
        this.container = null;
        this.input = null;
        this.clearButton = null;
        this.suggestionsContainer = null;
        this.map = null;
    }

    /**
     * Create the geocoder container and UI elements
     */
    private createContainer(): HTMLDivElement {
        const prefix = this.options.cssPrefix;
        const container = document.createElement('div');
        container.className = `${prefix}-container maplibregl-ctrl`;

        // Search wrapper
        const searchWrapper = document.createElement('div');
        searchWrapper.className = `${prefix}-search-wrapper`;

        // Search icon
        const searchIcon = document.createElement('span');
        searchIcon.className = `${prefix}-search-icon`;
        searchIcon.innerHTML = this.getSearchIconSvg();
        searchWrapper.appendChild(searchIcon);

        // Input field
        this.input = document.createElement('input');
        this.input.type = 'text';
        this.input.className = `${prefix}-input`;
        this.input.placeholder = this.options.placeholder;
        this.input.autocomplete = 'off';
        this.input.spellcheck = false;
        // Store handler references for proper cleanup
        this.inputHandler = () => this.handleInput();
        this.focusHandler = () => this.handleFocus();
        this.keydownHandler = (e: KeyboardEvent) => this.handleKeyDown(e);
        this.input.addEventListener('input', this.inputHandler);
        this.input.addEventListener('focus', this.focusHandler);
        this.input.addEventListener('keydown', this.keydownHandler);
        searchWrapper.appendChild(this.input);

        // Clear button
        this.clearButton = document.createElement('button');
        this.clearButton.type = 'button';
        this.clearButton.className = `${prefix}-clear-button`;
        this.clearButton.innerHTML = '&times;';
        this.clearButton.style.display = 'none';
        this.clearHandler = () => this.handleClear();
        this.clearButton.addEventListener('click', this.clearHandler);
        searchWrapper.appendChild(this.clearButton);

        // Loading spinner
        const spinner = document.createElement('span');
        spinner.className = `${prefix}-spinner`;
        spinner.style.display = 'none';
        searchWrapper.appendChild(spinner);

        container.appendChild(searchWrapper);

        // Suggestions dropdown
        this.suggestionsContainer = document.createElement('div');
        this.suggestionsContainer.className = `${prefix}-suggestions`;
        this.suggestionsContainer.style.display = 'none';
        container.appendChild(this.suggestionsContainer);

        // Close suggestions when clicking outside - store reference for cleanup
        this.documentClickHandler = (e: MouseEvent) => {
            if (!container.contains(e.target as Node)) {
                this.hideSuggestions();
            }
        };
        document.addEventListener('click', this.documentClickHandler);

        return container;
    }

    /**
     * Handle input changes with debouncing
     */
    private handleInput(): void {
        const query = this.input?.value.trim() || '';

        // Show/hide clear button
        if (this.clearButton) {
            this.clearButton.style.display = query.length > 0 ? 'block' : 'none';
        }

        // Cancel previous search
        if (this.debounceTimer) {
            clearTimeout(this.debounceTimer);
        }
        if (this.abortController) {
            this.abortController.abort();
        }

        // Check minimum length
        if (query.length < this.options.minLength) {
            this.hideSuggestions();
            return;
        }

        // Debounce the search
        this.debounceTimer = window.setTimeout(() => {
            this.search(query);
        }, this.options.debounceMs);
    }

    /**
     * Handle input focus
     */
    private handleFocus(): void {
        const query = this.input?.value.trim() || '';
        if (query.length >= this.options.minLength) {
            this.search(query);
        }
    }

    /**
     * Handle keyboard navigation
     */
    private handleKeyDown(e: KeyboardEvent): void {
        const suggestions = this.suggestionsContainer?.querySelectorAll(`.${this.options.cssPrefix}-suggestion`);
        if (!suggestions || suggestions.length === 0) return;

        const activeClass = `${this.options.cssPrefix}-suggestion-active`;
        let activeIndex = -1;

        suggestions.forEach((el, i) => {
            if (el.classList.contains(activeClass)) {
                activeIndex = i;
            }
        });

        switch (e.key) {
            case 'ArrowDown':
                e.preventDefault();
                if (activeIndex >= 0) {
                    suggestions[activeIndex].classList.remove(activeClass);
                }
                activeIndex = (activeIndex + 1) % suggestions.length;
                suggestions[activeIndex].classList.add(activeClass);
                break;

            case 'ArrowUp':
                e.preventDefault();
                if (activeIndex >= 0) {
                    suggestions[activeIndex].classList.remove(activeClass);
                }
                activeIndex = activeIndex <= 0 ? suggestions.length - 1 : activeIndex - 1;
                suggestions[activeIndex].classList.add(activeClass);
                break;

            case 'Enter':
                e.preventDefault();
                if (activeIndex >= 0) {
                    (suggestions[activeIndex] as HTMLElement).click();
                }
                break;

            case 'Escape':
                this.hideSuggestions();
                this.input?.blur();
                break;
        }
    }

    /**
     * Handle clear button click
     */
    private handleClear(): void {
        if (this.input) {
            this.input.value = '';
            this.input.focus();
        }
        if (this.clearButton) {
            this.clearButton.style.display = 'none';
        }
        this.hideSuggestions();
        this.clearMarker();
        this.options.onClear();
    }

    /**
     * Search Nominatim for locations
     */
    private async search(query: string): Promise<void> {
        this.showLoading(true);

        try {
            this.abortController = new AbortController();

            const params = new URLSearchParams({
                q: query,
                format: 'json',
                addressdetails: '1',
                limit: this.options.maxResults.toString(),
            });

            // Add country filter if specified
            if (this.options.countryCodes.length > 0) {
                params.set('countrycodes', this.options.countryCodes.join(','));
            }

            // Add proximity bias if specified
            if (this.options.proximity[0] !== 0 || this.options.proximity[1] !== 0) {
                params.set('viewbox', this.getViewbox(this.options.proximity, 5));
                params.set('bounded', '0'); // Don't strictly limit to viewbox
            }

            const response = await fetch(
                `${this.options.nominatimUrl}/search?${params.toString()}`,
                {
                    signal: this.abortController.signal,
                    headers: {
                        'Accept': 'application/json',
                        'User-Agent': 'Cartographus/1.0',
                    },
                }
            );

            if (!response.ok) {
                throw new Error(`Nominatim error: ${response.status}`);
            }

            const results: NominatimResult[] = await response.json();
            this.showSuggestions(results);
        } catch (error) {
            if ((error as Error).name !== 'AbortError') {
                logger.error('Search failed:', error);
                this.showError('Search failed. Please try again.');
            }
        } finally {
            this.showLoading(false);
        }
    }

    /**
     * Display search suggestions
     */
    private showSuggestions(results: NominatimResult[]): void {
        if (!this.suggestionsContainer) return;

        this.suggestionsContainer.innerHTML = '';

        if (results.length === 0) {
            const noResults = document.createElement('div');
            noResults.className = `${this.options.cssPrefix}-no-results`;
            noResults.textContent = 'No results found';
            this.suggestionsContainer.appendChild(noResults);
        } else {
            results.forEach((result, index) => {
                const suggestion = this.createSuggestionElement(result, index);
                this.suggestionsContainer!.appendChild(suggestion);
            });
        }

        this.suggestionsContainer.style.display = 'block';
    }

    /**
     * Create a suggestion list item
     */
    private createSuggestionElement(result: NominatimResult, index: number): HTMLDivElement {
        const prefix = this.options.cssPrefix;
        const suggestion = document.createElement('div');
        suggestion.className = `${prefix}-suggestion`;
        suggestion.tabIndex = 0;

        // Format the display name nicely
        const parts = result.display_name.split(', ');
        const primary = parts[0];
        const secondary = parts.slice(1, 3).join(', ');

        suggestion.innerHTML = `
            <div class="${prefix}-suggestion-icon">${this.getPlaceIconSvg(result.type)}</div>
            <div class="${prefix}-suggestion-text">
                <div class="${prefix}-suggestion-primary">${this.escapeHtml(primary)}</div>
                <div class="${prefix}-suggestion-secondary">${this.escapeHtml(secondary)}</div>
            </div>
        `;

        suggestion.addEventListener('click', () => this.selectResult(result));
        suggestion.addEventListener('keydown', (e) => {
            if (e.key === 'Enter') {
                this.selectResult(result);
            }
        });

        // Highlight first result by default
        if (index === 0) {
            suggestion.classList.add(`${prefix}-suggestion-active`);
        }

        return suggestion;
    }

    /**
     * Handle result selection
     */
    private selectResult(result: NominatimResult): void {
        const lng = parseFloat(result.lon);
        const lat = parseFloat(result.lat);

        // Update input value
        if (this.input) {
            this.input.value = result.display_name;
        }

        // Hide suggestions
        this.hideSuggestions();

        // Fly to location
        if (this.map) {
            // Use bounding box if available for better framing
            if (result.boundingbox) {
                const [south, north, west, east] = result.boundingbox.map(Number);
                this.map.fitBounds(
                    [[west, south], [east, north]],
                    {
                        padding: 50,
                        maxZoom: this.options.flyToZoom,
                        duration: this.options.flyToDuration,
                    }
                );
            } else {
                this.map.flyTo({
                    center: [lng, lat] as LngLatLike,
                    zoom: this.options.flyToZoom,
                    duration: this.options.flyToDuration,
                });
            }

            // Add marker if enabled
            if (this.options.showMarker) {
                this.addMarker([lng, lat]);
            }
        }

        // Trigger callback
        this.options.onResult(result);
    }

    /**
     * Add a marker at the selected location
     */
    private async addMarker(lngLat: [number, number]): Promise<void> {
        this.clearMarker();

        if (!this.map) return;

        try {
            const maplibregl = await import('maplibre-gl');

            // Create custom marker element
            const el = document.createElement('div');
            el.className = `${this.options.cssPrefix}-marker`;
            el.innerHTML = this.getMarkerSvg(this.options.markerColor);

            this.marker = new maplibregl.Marker({ element: el })
                .setLngLat(lngLat)
                .addTo(this.map);
        } catch (error) {
            logger.warn('Failed to create marker:', error);
        }
    }

    /**
     * Remove the current marker
     */
    private clearMarker(): void {
        if (this.marker) {
            this.marker.remove();
            this.marker = null;
        }
    }

    /**
     * Hide the suggestions dropdown
     */
    private hideSuggestions(): void {
        if (this.suggestionsContainer) {
            this.suggestionsContainer.style.display = 'none';
        }
    }

    /**
     * Show loading spinner
     */
    private showLoading(show: boolean): void {
        const spinner = this.container?.querySelector(`.${this.options.cssPrefix}-spinner`) as HTMLElement;
        if (spinner) {
            spinner.style.display = show ? 'block' : 'none';
        }
    }

    /**
     * Show error message
     */
    private showError(message: string): void {
        if (!this.suggestionsContainer) return;

        this.suggestionsContainer.innerHTML = `
            <div class="${this.options.cssPrefix}-error">${this.escapeHtml(message)}</div>
        `;
        this.suggestionsContainer.style.display = 'block';
    }

    /**
     * Calculate viewbox for proximity bias
     */
    private getViewbox(center: [number, number], radiusDegrees: number): string {
        const [lng, lat] = center;
        return `${lng - radiusDegrees},${lat - radiusDegrees},${lng + radiusDegrees},${lat + radiusDegrees}`;
    }

    /**
     * Escape HTML to prevent XSS
     */
    private escapeHtml(text: string): string {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

    /**
     * Get search icon SVG
     */
    private getSearchIconSvg(): string {
        return `<svg viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="11" cy="11" r="8"/>
            <path d="M21 21l-4.35-4.35"/>
        </svg>`;
    }

    /**
     * Get place icon SVG based on type
     */
    private getPlaceIconSvg(type: string): string {
        // Return appropriate icon based on place type
        if (['city', 'town', 'village', 'hamlet'].includes(type)) {
            return `<svg viewBox="0 0 24 24" width="16" height="16" fill="currentColor">
                <path d="M12 2L2 7l10 5 10-5-10-5zM2 17l10 5 10-5M2 12l10 5 10-5"/>
            </svg>`;
        }
        return `<svg viewBox="0 0 24 24" width="16" height="16" fill="currentColor">
            <path d="M12 2C8.13 2 5 5.13 5 9c0 5.25 7 13 7 13s7-7.75 7-13c0-3.87-3.13-7-7-7zm0 9.5c-1.38 0-2.5-1.12-2.5-2.5s1.12-2.5 2.5-2.5 2.5 1.12 2.5 2.5-1.12 2.5-2.5 2.5z"/>
        </svg>`;
    }

    /**
     * Get marker SVG
     */
    private getMarkerSvg(color: string): string {
        return `<svg viewBox="0 0 24 36" width="24" height="36">
            <path fill="${color}" d="M12 0C5.4 0 0 5.4 0 12c0 9 12 24 12 24s12-15 12-24C24 5.4 18.6 0 12 0z"/>
            <circle fill="white" cx="12" cy="12" r="5"/>
        </svg>`;
    }

    /**
     * Inject component styles
     */
    private injectStyles(): void {
        const styleId = `${this.options.cssPrefix}-styles`;
        if (document.getElementById(styleId)) return;

        const prefix = this.options.cssPrefix;
        const style = document.createElement('style');
        style.id = styleId;
        style.textContent = `
            .${prefix}-container {
                position: relative;
                font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
                font-size: 13px;
            }

            .${prefix}-search-wrapper {
                display: flex;
                align-items: center;
                background: rgba(20, 20, 30, 0.95);
                border: 1px solid #4ecdc4;
                border-radius: 4px;
                padding: 0 10px;
                min-width: 240px;
            }

            .${prefix}-search-icon {
                color: #4ecdc4;
                display: flex;
                align-items: center;
                margin-right: 8px;
            }

            .${prefix}-input {
                flex: 1;
                background: transparent;
                border: none;
                color: #eaeaea;
                padding: 10px 0;
                outline: none;
                font-size: 13px;
                min-width: 180px;
            }

            .${prefix}-input::placeholder {
                color: #a0a0a0;
            }

            .${prefix}-clear-button {
                background: transparent;
                border: none;
                color: #a0a0a0;
                cursor: pointer;
                padding: 4px 8px;
                font-size: 18px;
                line-height: 1;
            }

            .${prefix}-clear-button:hover {
                color: #eaeaea;
            }

            .${prefix}-spinner {
                width: 16px;
                height: 16px;
                border: 2px solid #4ecdc4;
                border-top-color: transparent;
                border-radius: 50%;
                animation: ${prefix}-spin 0.8s linear infinite;
                margin-left: 8px;
            }

            @keyframes ${prefix}-spin {
                to { transform: rotate(360deg); }
            }

            .${prefix}-suggestions {
                position: absolute;
                top: 100%;
                left: 0;
                right: 0;
                background: rgba(20, 20, 30, 0.98);
                border: 1px solid #4ecdc4;
                border-top: none;
                border-radius: 0 0 4px 4px;
                max-height: 300px;
                overflow-y: auto;
                z-index: 1000;
            }

            .${prefix}-suggestion {
                display: flex;
                align-items: center;
                padding: 10px 12px;
                cursor: pointer;
                transition: background-color 0.15s;
            }

            .${prefix}-suggestion:hover,
            .${prefix}-suggestion-active {
                background: rgba(78, 205, 196, 0.2);
            }

            .${prefix}-suggestion-icon {
                color: #4ecdc4;
                margin-right: 10px;
                flex-shrink: 0;
            }

            .${prefix}-suggestion-text {
                flex: 1;
                min-width: 0;
            }

            .${prefix}-suggestion-primary {
                color: #eaeaea;
                font-weight: 500;
                white-space: nowrap;
                overflow: hidden;
                text-overflow: ellipsis;
            }

            .${prefix}-suggestion-secondary {
                color: #a0a0a0;
                font-size: 11px;
                white-space: nowrap;
                overflow: hidden;
                text-overflow: ellipsis;
            }

            .${prefix}-no-results,
            .${prefix}-error {
                padding: 12px;
                color: #a0a0a0;
                text-align: center;
            }

            .${prefix}-error {
                color: #e94560;
            }

            .${prefix}-marker {
                cursor: pointer;
            }

            .${prefix}-marker svg {
                filter: drop-shadow(0 2px 4px rgba(0,0,0,0.3));
            }

            /* Light theme support */
            [data-theme="light"] .${prefix}-search-wrapper {
                background: rgba(255, 255, 255, 0.95);
                border-color: #3a3a4e;
            }

            [data-theme="light"] .${prefix}-input {
                color: #16213e;
            }

            [data-theme="light"] .${prefix}-suggestions {
                background: rgba(255, 255, 255, 0.98);
                border-color: #3a3a4e;
            }

            [data-theme="light"] .${prefix}-suggestion-primary {
                color: #16213e;
            }

            /* High contrast mode */
            [data-theme="high-contrast"] .${prefix}-search-wrapper {
                background: #000000;
                border-color: #00ff00;
            }

            [data-theme="high-contrast"] .${prefix}-search-icon {
                color: #00ff00;
            }

            [data-theme="high-contrast"] .${prefix}-suggestions {
                background: #000000;
                border-color: #00ff00;
            }
        `;

        document.head.appendChild(style);
    }

    /**
     * Programmatically set the search query
     */
    public setQuery(query: string): void {
        if (this.input) {
            this.input.value = query;
            this.handleInput();
        }
    }

    /**
     * Clear the current search
     */
    public clear(): void {
        this.handleClear();
    }

    /**
     * Get the current query
     */
    public getQuery(): string {
        return this.input?.value || '';
    }
}

/**
 * Factory function to create a geocoder with map integration
 */
export function createGeocoder(options: GeocoderOptions = {}): NominatimGeocoder {
    return new NominatimGeocoder(options);
}

/**
 * Reverse geocode a coordinate to an address
 */
export async function reverseGeocode(
    lngLat: [number, number],
    nominatimUrl: string = 'https://nominatim.openstreetmap.org'
): Promise<NominatimResult | null> {
    try {
        const [lng, lat] = lngLat;
        const params = new URLSearchParams({
            lat: lat.toString(),
            lon: lng.toString(),
            format: 'json',
            addressdetails: '1',
        });

        const response = await fetch(
            `${nominatimUrl}/reverse?${params.toString()}`,
            {
                headers: {
                    'Accept': 'application/json',
                    'User-Agent': 'Cartographus/1.0',
                },
            }
        );

        if (!response.ok) {
            throw new Error(`Reverse geocode failed: ${response.status}`);
        }

        return await response.json();
    } catch (error) {
        logger.error('Reverse geocode failed:', error);
        return null;
    }
}
