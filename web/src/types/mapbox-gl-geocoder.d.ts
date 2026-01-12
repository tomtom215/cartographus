// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
declare module '@mapbox/mapbox-gl-geocoder' {
    import { IControl, Map } from 'mapbox-gl';

    export interface GeocoderOptions {
        accessToken: string;
        mapboxgl?: any;
        zoom?: number;
        flyTo?: boolean | object;
        placeholder?: string;
        proximity?: { longitude: number; longitude: number };
        trackProximity?: boolean;
        collapsed?: boolean;
        clearAndBlurOnEsc?: boolean;
        clearOnBlur?: boolean;
        bbox?: [number, number, number, number];
        countries?: string;
        types?: string;
        minLength?: number;
        limit?: number;
        language?: string;
        filter?: (feature: any) => boolean;
        localGeocoder?: (query: string) => any[];
        externalGeocoder?: (query: string) => Promise<any[]>;
        reverseGeocode?: boolean;
        enableEventLogging?: boolean;
        marker?: boolean | object;
        render?: (feature: any) => string;
        getItemValue?: (feature: any) => string;
        mode?: 'mapbox.places' | 'mapbox.places-permanent';
        localGeocoderOnly?: boolean;
    }

    export default class MapboxGeocoder implements IControl {
        constructor(options: GeocoderOptions);
        onAdd(map: Map): HTMLElement;
        onRemove(map: Map): void;
        query(searchInput: string): this;
        setInput(searchInput: string): this;
        setProximity(proximity: { longitude: number; latitude: number } | null): this;
        getProximity(): { longitude: number; latitude: number } | undefined;
        setRenderFunction(fn: (feature: any) => string): this;
        getRenderFunction(): ((feature: any) => string) | undefined;
        setLanguage(language: string): this;
        getLanguage(): string | undefined;
        setZoom(zoom: number): this;
        getZoom(): number | undefined;
        setFlyTo(flyTo: boolean | object): this;
        getFlyTo(): boolean | object | undefined;
        setPlaceholder(placeholder: string): this;
        getPlaceholder(): string | undefined;
        setBbox(bbox: [number, number, number, number]): this;
        getBbox(): [number, number, number, number] | undefined;
        setCountries(countries: string): this;
        getCountries(): string | undefined;
        setTypes(types: string): this;
        getTypes(): string | undefined;
        setMinLength(minLength: number): this;
        getMinLength(): number | undefined;
        setLimit(limit: number): this;
        getLimit(): number | undefined;
        setFilter(filter: (feature: any) => boolean): this;
        setOrigin(origin: string): this;
        getOrigin(): string | undefined;
        setAutocomplete(autocomplete: boolean): this;
        getAutocomplete(): boolean | undefined;
        setFuzzyMatch(fuzzyMatch: boolean): this;
        getFuzzyMatch(): boolean | undefined;
        setRouting(routing: boolean): this;
        getRouting(): boolean | undefined;
        setWorldview(worldview: string): this;
        getWorldview(): string | undefined;
        on(type: string, fn: (e: any) => void): this;
        off(type: string, fn: (e: any) => void): this;
        clear(): this;
    }
}
