// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Cross-Filter Event Bus for Cartographus
 *
 * A lightweight, type-safe publish/subscribe event bus for decoupled
 * communication between managers. Enables Tableau-style linked charts
 * and cross-filter interactions.
 *
 * Features:
 * - Type-safe event publishing and subscribing
 * - Wildcard subscriptions for debugging/logging
 * - Event history for debugging
 * - Automatic cleanup with unsubscribe functions
 * - Performance metrics
 *
 * Usage:
 * ```typescript
 * // Subscribe to events
 * const unsubscribe = eventBus.on('filter:changed', (event) => {
 *   console.log('Filters changed:', event.filters);
 * });
 *
 * // Publish events
 * eventBus.emit({
 *   type: 'filter:changed',
 *   filters: { users: ['user1'] },
 *   changedField: 'users',
 *   source: 'FilterManager'
 * });
 *
 * // Cleanup
 * unsubscribe();
 * ```
 */

import type {
	CrossFilterEvent,
	EventType,
	EventCallback,
	EventPayload,
	Unsubscribe,
} from './types/events';
import { createLogger } from './logger';

const logger = createLogger('EventBus');

/**
 * Configuration options for EventBus
 */
export interface EventBusOptions {
	/** Enable debug logging */
	debug?: boolean;
	/** Maximum events to keep in history (0 = disabled) */
	historySize?: number;
	/** Enable performance metrics */
	metrics?: boolean;
}

/**
 * Event history entry for debugging
 */
interface EventHistoryEntry {
	event: CrossFilterEvent;
	timestamp: number;
	listenerCount: number;
}

/**
 * Performance metrics for the event bus
 */
export interface EventBusMetrics {
	totalEventsPublished: number;
	totalSubscriptions: number;
	activeSubscriptions: number;
	eventCounts: Record<string, number>;
	averageListenersPerEvent: number;
}

/**
 * Cross-Filter Event Bus
 *
 * Singleton pattern for global event communication
 */
export class EventBus {
	private listeners: Map<string, Set<EventCallback>> = new Map();
	private wildcardListeners: Set<EventCallback> = new Set();
	private history: EventHistoryEntry[] = [];
	private options: Required<EventBusOptions>;
	private metrics: EventBusMetrics;

	constructor(options: EventBusOptions = {}) {
		this.options = {
			debug: options.debug ?? false,
			historySize: options.historySize ?? 0,
			metrics: options.metrics ?? true,
		};

		this.metrics = {
			totalEventsPublished: 0,
			totalSubscriptions: 0,
			activeSubscriptions: 0,
			eventCounts: {},
			averageListenersPerEvent: 0,
		};
	}

	/**
	 * Subscribe to a specific event type
	 *
	 * @param eventType - The event type to subscribe to
	 * @param callback - Function to call when event is published
	 * @returns Unsubscribe function
	 */
	on<T extends EventType>(
		eventType: T,
		callback: EventCallback<EventPayload<T>>
	): Unsubscribe {
		if (!this.listeners.has(eventType)) {
			this.listeners.set(eventType, new Set());
		}

		const listeners = this.listeners.get(eventType)!;
		listeners.add(callback as EventCallback);

		this.metrics.totalSubscriptions++;
		this.metrics.activeSubscriptions++;

		if (this.options.debug) {
			logger.debug(`Subscribed to "${eventType}" (${listeners.size} listeners)`);
		}

		// Return unsubscribe function
		return () => {
			listeners.delete(callback as EventCallback);
			this.metrics.activeSubscriptions--;

			if (this.options.debug) {
				logger.debug(`Unsubscribed from "${eventType}" (${listeners.size} listeners)`);
			}

			// Clean up empty listener sets
			if (listeners.size === 0) {
				this.listeners.delete(eventType);
			}
		};
	}

	/**
	 * Subscribe to all events (wildcard listener)
	 *
	 * Useful for debugging, logging, or analytics
	 *
	 * @param callback - Function to call for all events
	 * @returns Unsubscribe function
	 */
	onAll(callback: EventCallback<CrossFilterEvent>): Unsubscribe {
		this.wildcardListeners.add(callback);

		this.metrics.totalSubscriptions++;
		this.metrics.activeSubscriptions++;

		if (this.options.debug) {
			logger.debug(`Subscribed to all events (${this.wildcardListeners.size} wildcard listeners)`);
		}

		return () => {
			this.wildcardListeners.delete(callback);
			this.metrics.activeSubscriptions--;

			if (this.options.debug) {
				logger.debug(`Unsubscribed from all events`);
			}
		};
	}

	/**
	 * Subscribe to an event type, automatically unsubscribe after first event
	 *
	 * @param eventType - The event type to subscribe to
	 * @param callback - Function to call when event is published
	 * @returns Unsubscribe function (can be called to cancel before event fires)
	 */
	once<T extends EventType>(
		eventType: T,
		callback: EventCallback<EventPayload<T>>
	): Unsubscribe {
		const unsubscribe = this.on(eventType, (event) => {
			unsubscribe();
			callback(event);
		});
		return unsubscribe;
	}

	/**
	 * Publish an event to all subscribers
	 *
	 * @param event - The event to publish
	 */
	emit<T extends CrossFilterEvent>(event: T): void {
		const startTime = performance.now();
		const eventType = event.type;
		let listenerCount = 0;

		// Update metrics
		this.metrics.totalEventsPublished++;
		this.metrics.eventCounts[eventType] = (this.metrics.eventCounts[eventType] || 0) + 1;

		if (this.options.debug) {
			logger.debug(`Emitting "${eventType}"`, event);
		}

		// Notify specific listeners
		const listeners = this.listeners.get(eventType);
		if (listeners) {
			listenerCount += listeners.size;
			listeners.forEach((callback) => {
				try {
					callback(event);
				} catch (error) {
					logger.error(`Error in listener for "${eventType}":`, error);
				}
			});
		}

		// Notify wildcard listeners
		listenerCount += this.wildcardListeners.size;
		this.wildcardListeners.forEach((callback) => {
			try {
				callback(event);
			} catch (error) {
				logger.error(`Error in wildcard listener:`, error);
			}
		});

		// Update average listeners metric
		const totalEvents = this.metrics.totalEventsPublished;
		this.metrics.averageListenersPerEvent =
			(this.metrics.averageListenersPerEvent * (totalEvents - 1) + listenerCount) / totalEvents;

		// Add to history if enabled
		if (this.options.historySize > 0) {
			this.history.push({
				event,
				timestamp: Date.now(),
				listenerCount,
			});

			// Trim history to max size
			if (this.history.length > this.options.historySize) {
				this.history.shift();
			}
		}

		if (this.options.debug) {
			const elapsed = performance.now() - startTime;
			logger.debug(
				`"${eventType}" delivered to ${listenerCount} listeners in ${elapsed.toFixed(2)}ms`
			);
		}
	}

	/**
	 * Check if there are any subscribers for an event type
	 *
	 * @param eventType - The event type to check
	 * @returns True if there are subscribers
	 */
	hasListeners(eventType: EventType): boolean {
		const listeners = this.listeners.get(eventType);
		return (listeners?.size ?? 0) > 0 || this.wildcardListeners.size > 0;
	}

	/**
	 * Get the number of subscribers for an event type
	 *
	 * @param eventType - The event type to check
	 * @returns Number of subscribers
	 */
	listenerCount(eventType: EventType): number {
		const listeners = this.listeners.get(eventType);
		return (listeners?.size ?? 0) + this.wildcardListeners.size;
	}

	/**
	 * Remove all listeners for a specific event type
	 *
	 * @param eventType - The event type to clear
	 */
	off(eventType: EventType): void {
		const listeners = this.listeners.get(eventType);
		if (listeners) {
			this.metrics.activeSubscriptions -= listeners.size;
			this.listeners.delete(eventType);

			if (this.options.debug) {
				logger.debug(`Removed all listeners for "${eventType}"`);
			}
		}
	}

	/**
	 * Remove all listeners (full reset)
	 */
	clear(): void {
		const totalCleared = this.metrics.activeSubscriptions;
		this.listeners.clear();
		this.wildcardListeners.clear();
		this.metrics.activeSubscriptions = 0;

		if (this.options.debug) {
			logger.debug(`Cleared all listeners (${totalCleared} removed)`);
		}
	}

	/**
	 * Get event history (if enabled)
	 */
	getHistory(): readonly EventHistoryEntry[] {
		return this.history;
	}

	/**
	 * Clear event history
	 */
	clearHistory(): void {
		this.history = [];
	}

	/**
	 * Get performance metrics
	 */
	getMetrics(): Readonly<EventBusMetrics> {
		return { ...this.metrics };
	}

	/**
	 * Reset performance metrics
	 */
	resetMetrics(): void {
		this.metrics = {
			totalEventsPublished: 0,
			totalSubscriptions: this.metrics.activeSubscriptions, // Keep current count
			activeSubscriptions: this.metrics.activeSubscriptions,
			eventCounts: {},
			averageListenersPerEvent: 0,
		};
	}

	/**
	 * Enable or disable debug mode
	 */
	setDebug(enabled: boolean): void {
		this.options.debug = enabled;
	}

	/**
	 * Get list of all event types with active listeners
	 */
	getActiveEventTypes(): EventType[] {
		return Array.from(this.listeners.keys()) as EventType[];
	}
}

// ============================================================================
// Singleton Instance
// ============================================================================

let instance: EventBus | null = null;

/**
 * Get the global EventBus instance
 *
 * @param options - Options for creating the instance (only used on first call)
 * @returns The global EventBus instance
 */
export function getEventBus(options?: EventBusOptions): EventBus {
	if (!instance) {
		instance = new EventBus(options);
	}
	return instance;
}

/**
 * Reset the global EventBus instance
 *
 * Useful for testing or reinitializing the application
 */
export function resetEventBus(): void {
	if (instance) {
		instance.clear();
		instance = null;
	}
}

// ============================================================================
// Helper Functions
// ============================================================================

/**
 * Create a filter changed event
 */
export function createFilterChangedEvent(
	filters: import('./types/events').FilterState,
	changedField: string | null,
	source: string
): import('./types/events').FilterChangedEvent {
	return {
		type: 'filter:changed',
		filters,
		changedField,
		source,
	};
}

/**
 * Create a chart clicked event
 */
export function createChartClickedEvent(
	chartId: string,
	dimension: string,
	value: string | number,
	source: string,
	seriesName?: string,
	dataIndex?: number
): import('./types/events').ChartClickedEvent {
	return {
		type: 'chart:clicked',
		chartId,
		dimension,
		value,
		seriesName,
		dataIndex,
		source,
	};
}

/**
 * Create a cross-highlight event
 */
export function createCrossHighlightEvent(
	dimension: string,
	values: (string | number)[],
	source: string
): import('./types/events').CrossHighlightEvent {
	return {
		type: 'crossHighlight:show',
		dimension,
		values,
		source,
	};
}

/**
 * Create a provenance available event
 */
export function createProvenanceEvent(
	queryHash: string,
	generatedAt: string,
	queryTimeMs: number,
	eventCount: number,
	cached: boolean,
	source: string
): import('./types/events').ProvenanceAvailableEvent {
	return {
		type: 'provenance:available',
		queryHash,
		generatedAt,
		queryTimeMs,
		eventCount,
		cached,
		source,
	};
}

// Default export for convenience
export default EventBus;
