// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Data Provenance Manager for Cartographus
 *
 * Displays query metadata and provenance information for data auditability.
 * Shows query hash, execution time, data range, and cache status.
 *
 * This component provides:
 * - Query reproducibility through deterministic hashes
 * - Performance visibility through timing metrics
 * - Data freshness indicators
 * - Audit trail for compliance
 */

import { getEventBus, type EventBus } from '../lib/event-bus';
import type { ProvenanceAvailableEvent, Unsubscribe } from '../lib/types/events';

/**
 * Provenance data structure
 */
export interface ProvenanceInfo {
	queryHash: string;
	generatedAt: Date;
	queryTimeMs: number;
	eventCount: number;
	cached: boolean;
	dataRangeStart?: Date;
	dataRangeEnd?: Date;
	source: string;
}

/**
 * Configuration for DataProvenanceManager
 */
export interface DataProvenanceConfig {
	/** Container element ID for provenance display */
	containerId?: string;
	/** Show detailed provenance info (expanded mode) */
	showDetails?: boolean;
	/** Auto-hide after N milliseconds (0 = never) */
	autoHideMs?: number;
	/** Position of the provenance display */
	position?: 'top-right' | 'top-left' | 'bottom-right' | 'bottom-left';
}

/**
 * Data Provenance Manager
 *
 * Manages the display of query provenance information for data auditability
 */
export class DataProvenanceManager {
	private eventBus: EventBus;
	private container: HTMLElement | null = null;
	private currentProvenance: ProvenanceInfo | null = null;
	private unsubscribe: Unsubscribe | null = null;
	private autoHideTimeout: ReturnType<typeof setTimeout> | null = null;
	private config: Required<DataProvenanceConfig>;
	private isExpanded: boolean = false;

	constructor(config: DataProvenanceConfig = {}) {
		this.config = {
			containerId: config.containerId ?? 'data-provenance',
			showDetails: config.showDetails ?? false,
			autoHideMs: config.autoHideMs ?? 0,
			position: config.position ?? 'bottom-right',
		};

		this.eventBus = getEventBus();
		this.initialize();
	}

	/**
	 * Initialize the provenance manager
	 */
	private initialize(): void {
		this.createContainer();
		this.subscribeToEvents();
	}

	/**
	 * Create or get the container element
	 */
	private createContainer(): void {
		let container = document.getElementById(this.config.containerId);

		if (!container) {
			container = document.createElement('div');
			container.id = this.config.containerId;
			document.body.appendChild(container);
		}

		// Apply base styles
		container.className = `data-provenance data-provenance--${this.config.position}`;
		container.setAttribute('role', 'status');
		container.setAttribute('aria-live', 'polite');
		container.setAttribute('aria-label', 'Data provenance information');

		// Add click handler for expand/collapse
		container.addEventListener('click', () => this.toggleExpanded());

		this.container = container;
	}

	/**
	 * Subscribe to provenance events
	 */
	private subscribeToEvents(): void {
		this.unsubscribe = this.eventBus.on('provenance:available', (event: ProvenanceAvailableEvent) => {
			this.updateProvenance({
				queryHash: event.queryHash,
				generatedAt: new Date(event.generatedAt),
				queryTimeMs: event.queryTimeMs,
				eventCount: event.eventCount,
				cached: event.cached,
				source: event.source,
			});
		});
	}

	/**
	 * Update the provenance display
	 */
	public updateProvenance(provenance: ProvenanceInfo): void {
		this.currentProvenance = provenance;
		this.render();

		// Handle auto-hide
		if (this.config.autoHideMs > 0) {
			if (this.autoHideTimeout) {
				clearTimeout(this.autoHideTimeout);
			}
			this.autoHideTimeout = setTimeout(() => {
				this.hide();
			}, this.config.autoHideMs);
		}
	}

	/**
	 * Render the provenance display
	 */
	private render(): void {
		if (!this.container || !this.currentProvenance) {
			return;
		}

		const p = this.currentProvenance;
		const formattedTime = this.formatTime(p.generatedAt);
		const formattedDuration = this.formatDuration(p.queryTimeMs);
		const formattedCount = this.formatNumber(p.eventCount);

		if (this.isExpanded || this.config.showDetails) {
			this.container.innerHTML = `
				<div class="provenance-expanded">
					<div class="provenance-header">
						<span class="provenance-icon">${this.getStatusIcon(p)}</span>
						<span class="provenance-title">Query Provenance</span>
						<button class="provenance-close" aria-label="Close">&times;</button>
					</div>
					<div class="provenance-body">
						<div class="provenance-row">
							<span class="provenance-label">Query Hash</span>
							<span class="provenance-value provenance-hash" title="Click to copy">${p.queryHash}</span>
						</div>
						<div class="provenance-row">
							<span class="provenance-label">Generated</span>
							<span class="provenance-value">${formattedTime}</span>
						</div>
						<div class="provenance-row">
							<span class="provenance-label">Query Time</span>
							<span class="provenance-value ${this.getTimingClass(p.queryTimeMs)}">${formattedDuration}</span>
						</div>
						<div class="provenance-row">
							<span class="provenance-label">Events</span>
							<span class="provenance-value">${formattedCount}</span>
						</div>
						<div class="provenance-row">
							<span class="provenance-label">Source</span>
							<span class="provenance-value">${p.cached ? 'Cache' : 'Database'}</span>
						</div>
						<div class="provenance-row">
							<span class="provenance-label">Endpoint</span>
							<span class="provenance-value provenance-source">${p.source}</span>
						</div>
					</div>
					<div class="provenance-footer">
						<span class="provenance-info">Query hash enables reproducibility</span>
					</div>
				</div>
			`;

			// Add copy handler for hash
			const hashElement = this.container.querySelector('.provenance-hash');
			if (hashElement) {
				hashElement.addEventListener('click', (e) => {
					e.stopPropagation();
					this.copyToClipboard(p.queryHash);
				});
			}

			// Add close button handler
			const closeBtn = this.container.querySelector('.provenance-close');
			if (closeBtn) {
				closeBtn.addEventListener('click', (e) => {
					e.stopPropagation();
					this.isExpanded = false;
					this.render();
				});
			}
		} else {
			// Compact view
			this.container.innerHTML = `
				<div class="provenance-compact" title="Click for details">
					<span class="provenance-icon">${this.getStatusIcon(p)}</span>
					<span class="provenance-hash-short">${p.queryHash.substring(0, 8)}</span>
					<span class="provenance-timing ${this.getTimingClass(p.queryTimeMs)}">${formattedDuration}</span>
					${p.cached ? '<span class="provenance-cached">cached</span>' : ''}
				</div>
			`;
		}

		this.container.classList.add('provenance-visible');
	}

	/**
	 * Get status icon based on provenance data
	 */
	private getStatusIcon(p: ProvenanceInfo): string {
		if (p.cached) {
			return '<svg class="icon icon-cached" viewBox="0 0 24 24" width="16" height="16"><path fill="currentColor" d="M19 8l-4 4h3c0 3.31-2.69 6-6 6-1.01 0-1.97-.25-2.8-.7l-1.46 1.46C8.97 19.54 10.43 20 12 20c4.42 0 8-3.58 8-8h3l-4-4zM6 12c0-3.31 2.69-6 6-6 1.01 0 1.97.25 2.8.7l1.46-1.46C15.03 4.46 13.57 4 12 4c-4.42 0-8 3.58-8 8H1l4 4 4-4H6z"/></svg>';
		}
		if (p.queryTimeMs < 100) {
			return '<svg class="icon icon-fast" viewBox="0 0 24 24" width="16" height="16"><path fill="currentColor" d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z"/></svg>';
		}
		if (p.queryTimeMs < 500) {
			return '<svg class="icon icon-normal" viewBox="0 0 24 24" width="16" height="16"><path fill="currentColor" d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-2 15l-5-5 1.41-1.41L10 14.17l7.59-7.59L19 8l-9 9z"/></svg>';
		}
		return '<svg class="icon icon-slow" viewBox="0 0 24 24" width="16" height="16"><path fill="currentColor" d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-2h2v2zm0-4h-2V7h2v6z"/></svg>';
	}

	/**
	 * Get CSS class based on query timing
	 */
	private getTimingClass(ms: number): string {
		if (ms < 100) return 'timing-fast';
		if (ms < 500) return 'timing-normal';
		if (ms < 2000) return 'timing-slow';
		return 'timing-very-slow';
	}

	/**
	 * Format timestamp for display
	 */
	private formatTime(date: Date): string {
		return date.toLocaleTimeString(undefined, {
			hour: '2-digit',
			minute: '2-digit',
			second: '2-digit',
		});
	}

	/**
	 * Format duration in milliseconds
	 */
	private formatDuration(ms: number): string {
		if (ms < 1) return '<1ms';
		if (ms < 1000) return `${Math.round(ms)}ms`;
		return `${(ms / 1000).toFixed(2)}s`;
	}

	/**
	 * Format large numbers with commas
	 */
	private formatNumber(n: number): string {
		return n.toLocaleString();
	}

	/**
	 * Copy text to clipboard
	 */
	private async copyToClipboard(text: string): Promise<void> {
		try {
			await navigator.clipboard.writeText(text);
			this.showCopiedFeedback();
		} catch {
			// Fallback for older browsers
			const textarea = document.createElement('textarea');
			textarea.value = text;
			textarea.style.position = 'fixed';
			textarea.style.opacity = '0';
			document.body.appendChild(textarea);
			textarea.select();
			document.execCommand('copy');
			document.body.removeChild(textarea);
			this.showCopiedFeedback();
		}
	}

	/**
	 * Show "Copied!" feedback
	 */
	private showCopiedFeedback(): void {
		const hashElement = this.container?.querySelector('.provenance-hash');
		if (hashElement) {
			const original = hashElement.textContent;
			hashElement.textContent = 'Copied!';
			hashElement.classList.add('provenance-copied');
			setTimeout(() => {
				if (hashElement) {
					hashElement.textContent = original;
					hashElement.classList.remove('provenance-copied');
				}
			}, 1500);
		}
	}

	/**
	 * Toggle expanded view
	 */
	public toggleExpanded(): void {
		this.isExpanded = !this.isExpanded;
		this.render();
	}

	/**
	 * Show the provenance display
	 */
	public show(): void {
		if (this.container) {
			this.container.classList.add('provenance-visible');
		}
	}

	/**
	 * Hide the provenance display
	 */
	public hide(): void {
		if (this.container) {
			this.container.classList.remove('provenance-visible');
		}
	}

	/**
	 * Get current provenance info
	 */
	public getProvenance(): ProvenanceInfo | null {
		return this.currentProvenance;
	}

	/**
	 * Manually emit a provenance event (for managers that load data)
	 */
	public emitProvenance(metadata: {
		queryHash: string;
		generatedAt: string;
		queryTimeMs: number;
		eventCount: number;
		cached: boolean;
		source: string;
	}): void {
		this.eventBus.emit({
			type: 'provenance:available',
			...metadata,
		});
	}

	/**
	 * Clean up resources
	 */
	public destroy(): void {
		if (this.unsubscribe) {
			this.unsubscribe();
			this.unsubscribe = null;
		}

		if (this.autoHideTimeout) {
			clearTimeout(this.autoHideTimeout);
			this.autoHideTimeout = null;
		}

		if (this.container) {
			this.container.remove();
			this.container = null;
		}

		this.currentProvenance = null;
	}
}

// ============================================================================
// CSS Styles (to be added to stylesheet)
// ============================================================================

/**
 * Add these styles to your CSS:
 *
 * .data-provenance {
 *   position: fixed;
 *   z-index: 1000;
 *   font-family: var(--font-mono, monospace);
 *   font-size: 12px;
 *   transition: opacity 0.2s, transform 0.2s;
 *   opacity: 0;
 *   transform: translateY(10px);
 *   pointer-events: none;
 * }
 *
 * .data-provenance.provenance-visible {
 *   opacity: 1;
 *   transform: translateY(0);
 *   pointer-events: auto;
 * }
 *
 * .data-provenance--bottom-right {
 *   bottom: 20px;
 *   right: 20px;
 * }
 *
 * .data-provenance--bottom-left {
 *   bottom: 20px;
 *   left: 20px;
 * }
 *
 * .data-provenance--top-right {
 *   top: 80px;
 *   right: 20px;
 * }
 *
 * .data-provenance--top-left {
 *   top: 80px;
 *   left: 20px;
 * }
 *
 * .provenance-compact {
 *   display: flex;
 *   align-items: center;
 *   gap: 8px;
 *   padding: 8px 12px;
 *   background: var(--bg-secondary, #1e1e1e);
 *   border: 1px solid var(--border-color, #333);
 *   border-radius: 6px;
 *   cursor: pointer;
 *   color: var(--text-secondary, #888);
 * }
 *
 * .provenance-compact:hover {
 *   border-color: var(--primary-color, #7c3aed);
 *   color: var(--text-primary, #fff);
 * }
 *
 * .provenance-expanded {
 *   width: 300px;
 *   background: var(--bg-secondary, #1e1e1e);
 *   border: 1px solid var(--border-color, #333);
 *   border-radius: 8px;
 *   box-shadow: 0 4px 12px rgba(0,0,0,0.3);
 * }
 *
 * .provenance-header {
 *   display: flex;
 *   align-items: center;
 *   gap: 8px;
 *   padding: 12px;
 *   border-bottom: 1px solid var(--border-color, #333);
 * }
 *
 * .provenance-title {
 *   flex: 1;
 *   font-weight: 600;
 *   color: var(--text-primary, #fff);
 * }
 *
 * .provenance-close {
 *   background: none;
 *   border: none;
 *   color: var(--text-secondary, #888);
 *   cursor: pointer;
 *   font-size: 18px;
 *   padding: 0 4px;
 * }
 *
 * .provenance-body {
 *   padding: 12px;
 * }
 *
 * .provenance-row {
 *   display: flex;
 *   justify-content: space-between;
 *   padding: 4px 0;
 * }
 *
 * .provenance-label {
 *   color: var(--text-secondary, #888);
 * }
 *
 * .provenance-value {
 *   color: var(--text-primary, #fff);
 *   font-weight: 500;
 * }
 *
 * .provenance-hash {
 *   cursor: pointer;
 *   padding: 2px 6px;
 *   background: var(--bg-tertiary, #2a2a2a);
 *   border-radius: 4px;
 * }
 *
 * .provenance-hash:hover {
 *   background: var(--primary-color, #7c3aed);
 * }
 *
 * .provenance-copied {
 *   color: var(--success-color, #10b981);
 * }
 *
 * .timing-fast { color: var(--success-color, #10b981); }
 * .timing-normal { color: var(--text-primary, #fff); }
 * .timing-slow { color: var(--warning-color, #f59e0b); }
 * .timing-very-slow { color: var(--error-color, #ef4444); }
 *
 * .provenance-cached {
 *   padding: 2px 6px;
 *   background: var(--info-bg, #1e3a5f);
 *   color: var(--info-color, #3b82f6);
 *   border-radius: 4px;
 *   font-size: 10px;
 *   text-transform: uppercase;
 * }
 *
 * .provenance-footer {
 *   padding: 8px 12px;
 *   border-top: 1px solid var(--border-color, #333);
 *   color: var(--text-tertiary, #666);
 *   font-size: 10px;
 * }
 *
 * .icon { vertical-align: middle; }
 * .icon-fast { color: var(--success-color, #10b981); }
 * .icon-normal { color: var(--text-secondary, #888); }
 * .icon-slow { color: var(--warning-color, #f59e0b); }
 * .icon-cached { color: var(--info-color, #3b82f6); }
 */

export default DataProvenanceManager;
