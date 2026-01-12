// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Cross-Filter Event Types for Cartographus
 *
 * This module defines the event types used by the EventBus for cross-filter
 * communication between managers. Events enable decoupled communication
 * for linked chart interactions, filter propagation, and data synchronization.
 *
 * Event Naming Convention:
 * - namespace:action (e.g., "filter:changed", "chart:clicked")
 * - Use present tense for actions
 * - Be specific about what happened
 */

// ============================================================================
// Filter Events
// ============================================================================

/**
 * Emitted when any filter value changes
 */
export interface FilterChangedEvent {
	type: 'filter:changed';
	filters: FilterState;
	changedField: string | null;
	source: string;
}

/**
 * Emitted when filters are cleared/reset
 */
export interface FilterClearedEvent {
	type: 'filter:cleared';
	previousFilters: FilterState;
	source: string;
}

/**
 * Filter state representation
 */
export interface FilterState {
	startDate?: string;
	endDate?: string;
	users?: string[];
	mediaTypes?: string[];
	platforms?: string[];
	players?: string[];
	libraries?: string[];
	locations?: string[];
	servers?: string[];
	transcodeDecisions?: string[];
	qualities?: string[];
	sources?: string[];
}

// ============================================================================
// Chart Events
// ============================================================================

/**
 * Emitted when user clicks on a chart element for drill-down
 */
export interface ChartClickedEvent {
	type: 'chart:clicked';
	chartId: string;
	dimension: string;
	value: string | number;
	seriesName?: string;
	dataIndex?: number;
	source: string;
}

/**
 * Emitted when user hovers over a chart element
 */
export interface ChartHoverEvent {
	type: 'chart:hover';
	chartId: string;
	dimension: string;
	value: string | number;
	x?: number;
	y?: number;
	source: string;
}

/**
 * Emitted when hover ends
 */
export interface ChartHoverEndEvent {
	type: 'chart:hoverEnd';
	chartId: string;
	source: string;
}

/**
 * Emitted when a chart requests data refresh
 */
export interface ChartRefreshEvent {
	type: 'chart:refresh';
	chartId: string;
	source: string;
}

/**
 * Emitted when chart rendering completes
 */
export interface ChartRenderedEvent {
	type: 'chart:rendered';
	chartId: string;
	renderTime: number;
	source: string;
}

// ============================================================================
// Date Range Events
// ============================================================================

/**
 * Emitted when date range is selected (e.g., brush selection)
 */
export interface DateRangeSelectedEvent {
	type: 'daterange:selected';
	startDate: string;
	endDate: string;
	source: string;
}

/**
 * Emitted when date range brush is being dragged (for preview)
 */
export interface DateRangeBrushingEvent {
	type: 'daterange:brushing';
	startDate: string;
	endDate: string;
	source: string;
}

// ============================================================================
// Geographic Events
// ============================================================================

/**
 * Emitted when a location is selected on the map
 */
export interface LocationSelectedEvent {
	type: 'location:selected';
	locationId: string;
	locationName: string;
	country?: string;
	city?: string;
	coordinates?: { lat: number; lng: number };
	source: string;
}

/**
 * Emitted when map zoom level changes
 */
export interface MapZoomChangedEvent {
	type: 'map:zoomChanged';
	zoomLevel: number;
	center: { lat: number; lng: number };
	bounds?: { north: number; south: number; east: number; west: number };
	source: string;
}

/**
 * Emitted when user clicks on map
 */
export interface MapClickedEvent {
	type: 'map:clicked';
	coordinates: { lat: number; lng: number };
	features?: Array<{ id: string; properties: Record<string, unknown> }>;
	source: string;
}

// ============================================================================
// User Events
// ============================================================================

/**
 * Emitted when a user is selected for filtering
 */
export interface UserSelectedEvent {
	type: 'user:selected';
	userId: string;
	username: string;
	source: string;
}

/**
 * Emitted when multiple users are selected
 */
export interface UsersSelectedEvent {
	type: 'users:selected';
	userIds: string[];
	usernames: string[];
	source: string;
}

// ============================================================================
// Media Events
// ============================================================================

/**
 * Emitted when media type is selected
 */
export interface MediaTypeSelectedEvent {
	type: 'mediaType:selected';
	mediaType: string;
	source: string;
}

/**
 * Emitted when content/title is selected
 */
export interface ContentSelectedEvent {
	type: 'content:selected';
	contentId: string;
	title: string;
	mediaType?: string;
	source: string;
}

// ============================================================================
// Data Events
// ============================================================================

/**
 * Emitted when new data is loaded
 */
export interface DataLoadedEvent {
	type: 'data:loaded';
	dataType: string;
	recordCount: number;
	loadTime: number;
	source: string;
}

/**
 * Emitted when data is refreshed
 */
export interface DataRefreshedEvent {
	type: 'data:refreshed';
	dataType: string;
	source: string;
}

/**
 * Emitted when WebSocket receives new playback event
 */
export interface PlaybackReceivedEvent {
	type: 'playback:received';
	eventId: string;
	userId: string;
	username: string;
	title: string;
	mediaType: string;
	source: string;
}

// ============================================================================
// Navigation Events
// ============================================================================

/**
 * Emitted when dashboard view changes
 */
export interface ViewChangedEvent {
	type: 'view:changed';
	view: 'maps' | 'activity' | 'analytics' | 'recently-added' | 'server';
	previousView?: string;
	source: string;
}

/**
 * Emitted when analytics page changes
 */
export interface AnalyticsPageChangedEvent {
	type: 'analyticsPage:changed';
	page: string;
	previousPage?: string;
	source: string;
}

// ============================================================================
// Cross-Filter Highlight Events
// ============================================================================

/**
 * Emitted to highlight data points across all charts
 * Used for linked highlighting (Tableau-style)
 */
export interface CrossHighlightEvent {
	type: 'crossHighlight:show';
	dimension: string;
	values: (string | number)[];
	source: string;
}

/**
 * Emitted to clear cross-chart highlighting
 */
export interface CrossHighlightClearEvent {
	type: 'crossHighlight:clear';
	source: string;
}

// ============================================================================
// Provenance Events
// ============================================================================

/**
 * Emitted when query metadata is available (for data provenance display)
 */
export interface ProvenanceAvailableEvent {
	type: 'provenance:available';
	queryHash: string;
	generatedAt: string;
	queryTimeMs: number;
	eventCount: number;
	cached: boolean;
	source: string;
}

// ============================================================================
// Union Type for All Events
// ============================================================================

export type CrossFilterEvent =
	| FilterChangedEvent
	| FilterClearedEvent
	| ChartClickedEvent
	| ChartHoverEvent
	| ChartHoverEndEvent
	| ChartRefreshEvent
	| ChartRenderedEvent
	| DateRangeSelectedEvent
	| DateRangeBrushingEvent
	| LocationSelectedEvent
	| MapZoomChangedEvent
	| MapClickedEvent
	| UserSelectedEvent
	| UsersSelectedEvent
	| MediaTypeSelectedEvent
	| ContentSelectedEvent
	| DataLoadedEvent
	| DataRefreshedEvent
	| PlaybackReceivedEvent
	| ViewChangedEvent
	| AnalyticsPageChangedEvent
	| CrossHighlightEvent
	| CrossHighlightClearEvent
	| ProvenanceAvailableEvent;

// ============================================================================
// Event Type Constants
// ============================================================================

export const EventTypes = {
	// Filter events
	FILTER_CHANGED: 'filter:changed',
	FILTER_CLEARED: 'filter:cleared',

	// Chart events
	CHART_CLICKED: 'chart:clicked',
	CHART_HOVER: 'chart:hover',
	CHART_HOVER_END: 'chart:hoverEnd',
	CHART_REFRESH: 'chart:refresh',
	CHART_RENDERED: 'chart:rendered',

	// Date range events
	DATERANGE_SELECTED: 'daterange:selected',
	DATERANGE_BRUSHING: 'daterange:brushing',

	// Geographic events
	LOCATION_SELECTED: 'location:selected',
	MAP_ZOOM_CHANGED: 'map:zoomChanged',
	MAP_CLICKED: 'map:clicked',

	// User events
	USER_SELECTED: 'user:selected',
	USERS_SELECTED: 'users:selected',

	// Media events
	MEDIA_TYPE_SELECTED: 'mediaType:selected',
	CONTENT_SELECTED: 'content:selected',

	// Data events
	DATA_LOADED: 'data:loaded',
	DATA_REFRESHED: 'data:refreshed',
	PLAYBACK_RECEIVED: 'playback:received',

	// Navigation events
	VIEW_CHANGED: 'view:changed',
	ANALYTICS_PAGE_CHANGED: 'analyticsPage:changed',

	// Cross-filter highlight events
	CROSS_HIGHLIGHT_SHOW: 'crossHighlight:show',
	CROSS_HIGHLIGHT_CLEAR: 'crossHighlight:clear',

	// Provenance events
	PROVENANCE_AVAILABLE: 'provenance:available',
} as const;

export type EventType = (typeof EventTypes)[keyof typeof EventTypes];

// ============================================================================
// Helper Types
// ============================================================================

/**
 * Extract event payload type by event type string
 */
export type EventPayload<T extends EventType> = Extract<CrossFilterEvent, { type: T }>;

/**
 * Callback type for event subscribers
 */
export type EventCallback<T extends CrossFilterEvent = CrossFilterEvent> = (event: T) => void;

/**
 * Unsubscribe function returned by subscribe
 */
export type Unsubscribe = () => void;
