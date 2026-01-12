// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Tooltip - Accessible, Reusable Tooltip Component
 *
 * Features:
 * - WCAG 2.1 AA compliant with ARIA attributes
 * - Keyboard support (focus triggers, Escape dismisses)
 * - Configurable positioning (top, bottom, left, right)
 * - Smart positioning that avoids viewport edges
 * - Configurable show/hide delays
 * - Rich content support (HTML content)
 * - Touch device support
 * - Reduced motion support
 * - Auto-dismiss when trigger element is hidden
 *
 * Reference: Production Readiness - Frontend Enhancements 4.1
 */

/**
 * Tooltip position options
 */
export type TooltipPosition = 'top' | 'bottom' | 'left' | 'right';

/**
 * Tooltip configuration options
 */
export interface TooltipOptions {
    /** Tooltip content (text or HTML) */
    content: string;
    /** Preferred position (default: 'top') */
    position?: TooltipPosition;
    /** Delay before showing in ms (default: 200) */
    showDelay?: number;
    /** Delay before hiding in ms (default: 100) */
    hideDelay?: number;
    /** Allow HTML content (default: false for security) */
    allowHtml?: boolean;
    /** Custom CSS class for styling */
    className?: string;
    /** Maximum width in pixels (default: 250) */
    maxWidth?: number;
    /** Offset from trigger element in pixels (default: 8) */
    offset?: number;
    /** z-index for stacking (default: 10000) */
    zIndex?: number;
    /** Show arrow pointer (default: true) */
    showArrow?: boolean;
    /** ARIA role (default: 'tooltip') */
    role?: 'tooltip' | 'status';
    /** ID for aria-describedby linking */
    id?: string;
}

/**
 * Internal state for tooltip instance
 */
interface TooltipState {
    isVisible: boolean;
    currentPosition: TooltipPosition;
    showTimeout: ReturnType<typeof setTimeout> | null;
    hideTimeout: ReturnType<typeof setTimeout> | null;
}

/**
 * Tooltip instance returned by createTooltip
 */
export interface TooltipInstance {
    /** Show the tooltip */
    show: () => void;
    /** Hide the tooltip */
    hide: () => void;
    /** Toggle visibility */
    toggle: () => void;
    /** Update tooltip content */
    setContent: (content: string) => void;
    /** Update tooltip options */
    setOptions: (options: Partial<TooltipOptions>) => void;
    /** Check if tooltip is visible */
    isVisible: () => boolean;
    /** Destroy the tooltip and clean up */
    destroy: () => void;
}

// Counter for unique IDs
let tooltipIdCounter = 0;

/**
 * Generate unique tooltip ID
 */
function generateId(): string {
    return `tooltip-${++tooltipIdCounter}`;
}

/**
 * Check if user prefers reduced motion
 */
function prefersReducedMotion(): boolean {
    return window.matchMedia('(prefers-reduced-motion: reduce)').matches;
}

/**
 * Get viewport dimensions
 */
function getViewport(): { width: number; height: number } {
    return {
        width: window.innerWidth || document.documentElement.clientWidth,
        height: window.innerHeight || document.documentElement.clientHeight,
    };
}

/**
 * Calculate optimal position avoiding viewport edges
 */
function calculateOptimalPosition(
    triggerRect: DOMRect,
    tooltipRect: { width: number; height: number },
    preferredPosition: TooltipPosition,
    offset: number
): { position: TooltipPosition; x: number; y: number } {
    const viewport = getViewport();
    const positions: TooltipPosition[] = ['top', 'bottom', 'left', 'right'];

    // Calculate positions for each option
    const positionCalcs: Record<TooltipPosition, { x: number; y: number; fits: boolean }> = {
        top: {
            x: triggerRect.left + (triggerRect.width - tooltipRect.width) / 2,
            y: triggerRect.top - tooltipRect.height - offset,
            fits: triggerRect.top - tooltipRect.height - offset >= 0,
        },
        bottom: {
            x: triggerRect.left + (triggerRect.width - tooltipRect.width) / 2,
            y: triggerRect.bottom + offset,
            fits: triggerRect.bottom + tooltipRect.height + offset <= viewport.height,
        },
        left: {
            x: triggerRect.left - tooltipRect.width - offset,
            y: triggerRect.top + (triggerRect.height - tooltipRect.height) / 2,
            fits: triggerRect.left - tooltipRect.width - offset >= 0,
        },
        right: {
            x: triggerRect.right + offset,
            y: triggerRect.top + (triggerRect.height - tooltipRect.height) / 2,
            fits: triggerRect.right + tooltipRect.width + offset <= viewport.width,
        },
    };

    // Try preferred position first
    if (positionCalcs[preferredPosition].fits) {
        const calc = positionCalcs[preferredPosition];
        return { position: preferredPosition, x: calc.x, y: calc.y };
    }

    // Try opposite position
    const opposites: Record<TooltipPosition, TooltipPosition> = {
        top: 'bottom',
        bottom: 'top',
        left: 'right',
        right: 'left',
    };
    const opposite = opposites[preferredPosition];
    if (positionCalcs[opposite].fits) {
        const calc = positionCalcs[opposite];
        return { position: opposite, x: calc.x, y: calc.y };
    }

    // Try other positions
    for (const pos of positions) {
        if (pos !== preferredPosition && pos !== opposite && positionCalcs[pos].fits) {
            const calc = positionCalcs[pos];
            return { position: pos, x: calc.x, y: calc.y };
        }
    }

    // Fallback to preferred position with clamping
    const calc = positionCalcs[preferredPosition];
    return {
        position: preferredPosition,
        x: Math.max(8, Math.min(calc.x, viewport.width - tooltipRect.width - 8)),
        y: Math.max(8, Math.min(calc.y, viewport.height - tooltipRect.height - 8)),
    };
}

/**
 * Create a tooltip for a trigger element
 */
export function createTooltip(
    trigger: HTMLElement,
    options: TooltipOptions
): TooltipInstance {
    // Apply defaults
    const config: Required<TooltipOptions> = {
        content: options.content,
        position: options.position ?? 'top',
        showDelay: options.showDelay ?? 200,
        hideDelay: options.hideDelay ?? 100,
        allowHtml: options.allowHtml ?? false,
        className: options.className ?? '',
        maxWidth: options.maxWidth ?? 250,
        offset: options.offset ?? 8,
        zIndex: options.zIndex ?? 10000,
        showArrow: options.showArrow ?? true,
        role: options.role ?? 'tooltip',
        id: options.id ?? generateId(),
    };

    // State
    const state: TooltipState = {
        isVisible: false,
        currentPosition: config.position,
        showTimeout: null,
        hideTimeout: null,
    };

    // Create tooltip element
    const tooltipEl = document.createElement('div');
    tooltipEl.id = config.id;
    tooltipEl.setAttribute('role', config.role);
    tooltipEl.setAttribute('aria-hidden', 'true');
    tooltipEl.className = `tooltip ${config.className}`.trim();
    tooltipEl.style.cssText = `
        position: fixed;
        visibility: hidden;
        opacity: 0;
        max-width: ${config.maxWidth}px;
        z-index: ${config.zIndex};
        pointer-events: none;
        transition: ${prefersReducedMotion() ? 'none' : 'opacity 0.15s ease, visibility 0.15s ease'};
    `;

    // Create content container
    const contentEl = document.createElement('div');
    contentEl.className = 'tooltip-content';
    tooltipEl.appendChild(contentEl);

    // Create arrow if enabled
    let arrowEl: HTMLElement | null = null;
    if (config.showArrow) {
        arrowEl = document.createElement('div');
        arrowEl.className = 'tooltip-arrow';
        arrowEl.setAttribute('aria-hidden', 'true');
        tooltipEl.appendChild(arrowEl);
    }

    // Set content
    function updateContent(content: string): void {
        if (config.allowHtml) {
            contentEl.innerHTML = content;
        } else {
            contentEl.textContent = content;
        }
    }
    updateContent(config.content);

    // Add to DOM
    document.body.appendChild(tooltipEl);

    // Link trigger to tooltip for accessibility
    trigger.setAttribute('aria-describedby', config.id);

    // Position the tooltip
    function positionTooltip(): void {
        const triggerRect = trigger.getBoundingClientRect();
        const tooltipRect = tooltipEl.getBoundingClientRect();

        const { position, x, y } = calculateOptimalPosition(
            triggerRect,
            { width: tooltipRect.width, height: tooltipRect.height },
            config.position,
            config.offset
        );

        state.currentPosition = position;
        tooltipEl.style.left = `${x}px`;
        tooltipEl.style.top = `${y}px`;

        // Update position class for arrow styling
        tooltipEl.classList.remove('tooltip-top', 'tooltip-bottom', 'tooltip-left', 'tooltip-right');
        tooltipEl.classList.add(`tooltip-${position}`);

        // Position arrow
        if (arrowEl) {
            const arrowSize = 6; // Should match CSS
            arrowEl.style.cssText = '';

            switch (position) {
                case 'top':
                    arrowEl.style.left = '50%';
                    arrowEl.style.bottom = `-${arrowSize}px`;
                    arrowEl.style.transform = 'translateX(-50%)';
                    break;
                case 'bottom':
                    arrowEl.style.left = '50%';
                    arrowEl.style.top = `-${arrowSize}px`;
                    arrowEl.style.transform = 'translateX(-50%)';
                    break;
                case 'left':
                    arrowEl.style.top = '50%';
                    arrowEl.style.right = `-${arrowSize}px`;
                    arrowEl.style.transform = 'translateY(-50%)';
                    break;
                case 'right':
                    arrowEl.style.top = '50%';
                    arrowEl.style.left = `-${arrowSize}px`;
                    arrowEl.style.transform = 'translateY(-50%)';
                    break;
            }
        }
    }

    // Show tooltip
    function show(): void {
        if (state.hideTimeout) {
            clearTimeout(state.hideTimeout);
            state.hideTimeout = null;
        }

        if (state.isVisible) return;

        state.showTimeout = setTimeout(() => {
            positionTooltip();
            tooltipEl.style.visibility = 'visible';
            tooltipEl.style.opacity = '1';
            tooltipEl.setAttribute('aria-hidden', 'false');
            state.isVisible = true;
            state.showTimeout = null;
        }, config.showDelay);
    }

    // Hide tooltip
    function hide(): void {
        if (state.showTimeout) {
            clearTimeout(state.showTimeout);
            state.showTimeout = null;
        }

        if (!state.isVisible) return;

        state.hideTimeout = setTimeout(() => {
            tooltipEl.style.visibility = 'hidden';
            tooltipEl.style.opacity = '0';
            tooltipEl.setAttribute('aria-hidden', 'true');
            state.isVisible = false;
            state.hideTimeout = null;
        }, config.hideDelay);
    }

    // Event handlers
    function handleMouseEnter(): void {
        show();
    }

    function handleMouseLeave(): void {
        hide();
    }

    function handleFocus(): void {
        show();
    }

    function handleBlur(): void {
        hide();
    }

    function handleKeydown(e: KeyboardEvent): void {
        if (e.key === 'Escape' && state.isVisible) {
            hide();
        }
    }

    function handleTouchStart(): void {
        if (state.isVisible) {
            hide();
        } else {
            show();
        }
    }

    // Attach event listeners
    trigger.addEventListener('mouseenter', handleMouseEnter);
    trigger.addEventListener('mouseleave', handleMouseLeave);
    trigger.addEventListener('focus', handleFocus);
    trigger.addEventListener('blur', handleBlur);
    trigger.addEventListener('keydown', handleKeydown);
    trigger.addEventListener('touchstart', handleTouchStart, { passive: true });

    // Handle scroll and resize
    function handleScrollResize(): void {
        if (state.isVisible) {
            positionTooltip();
        }
    }

    window.addEventListener('scroll', handleScrollResize, { passive: true });
    window.addEventListener('resize', handleScrollResize, { passive: true });

    // Return tooltip instance
    const instance: TooltipInstance = {
        show,
        hide,
        toggle: () => {
            if (state.isVisible) {
                hide();
            } else {
                show();
            }
        },
        setContent: (content: string) => {
            config.content = content;
            updateContent(content);
        },
        setOptions: (newOptions: Partial<TooltipOptions>) => {
            if (newOptions.content !== undefined) {
                config.content = newOptions.content;
                updateContent(newOptions.content);
            }
            if (newOptions.position !== undefined) {
                config.position = newOptions.position;
            }
            if (newOptions.showDelay !== undefined) {
                config.showDelay = newOptions.showDelay;
            }
            if (newOptions.hideDelay !== undefined) {
                config.hideDelay = newOptions.hideDelay;
            }
            if (newOptions.maxWidth !== undefined) {
                config.maxWidth = newOptions.maxWidth;
                tooltipEl.style.maxWidth = `${config.maxWidth}px`;
            }
            if (newOptions.className !== undefined) {
                tooltipEl.className = `tooltip ${newOptions.className}`.trim();
            }
        },
        isVisible: () => state.isVisible,
        destroy: () => {
            // Clear timeouts
            if (state.showTimeout) clearTimeout(state.showTimeout);
            if (state.hideTimeout) clearTimeout(state.hideTimeout);

            // Remove event listeners
            trigger.removeEventListener('mouseenter', handleMouseEnter);
            trigger.removeEventListener('mouseleave', handleMouseLeave);
            trigger.removeEventListener('focus', handleFocus);
            trigger.removeEventListener('blur', handleBlur);
            trigger.removeEventListener('keydown', handleKeydown);
            trigger.removeEventListener('touchstart', handleTouchStart);
            window.removeEventListener('scroll', handleScrollResize);
            window.removeEventListener('resize', handleScrollResize);

            // Remove aria attribute
            trigger.removeAttribute('aria-describedby');

            // Remove from DOM
            if (tooltipEl.parentNode) {
                tooltipEl.parentNode.removeChild(tooltipEl);
            }
        },
    };

    return instance;
}

/**
 * TooltipManager - Manages multiple tooltips declaratively
 *
 * Automatically creates tooltips for elements with data-tooltip attribute:
 * <button data-tooltip="Click to save" data-tooltip-position="bottom">Save</button>
 */
export class TooltipManager {
    private tooltips: Map<HTMLElement, TooltipInstance> = new Map();
    private observer: MutationObserver | null = null;
    private defaultOptions: Partial<TooltipOptions>;

    constructor(options: Partial<TooltipOptions> = {}) {
        this.defaultOptions = options;
        this.init();
    }

    /**
     * Initialize the tooltip manager
     */
    private init(): void {
        // Create tooltips for existing elements
        this.scanAndCreate();

        // Watch for DOM changes
        this.observer = new MutationObserver((mutations) => {
            for (const mutation of mutations) {
                // Check added nodes
                for (const node of mutation.addedNodes) {
                    if (node instanceof HTMLElement) {
                        this.createForElement(node);
                        node.querySelectorAll<HTMLElement>('[data-tooltip]').forEach(el => {
                            this.createForElement(el);
                        });
                    }
                }
                // Check removed nodes
                for (const node of mutation.removedNodes) {
                    if (node instanceof HTMLElement) {
                        this.destroyForElement(node);
                        node.querySelectorAll<HTMLElement>('[data-tooltip]').forEach(el => {
                            this.destroyForElement(el);
                        });
                    }
                }
                // Check attribute changes
                if (mutation.type === 'attributes' && mutation.target instanceof HTMLElement) {
                    const el = mutation.target;
                    if (mutation.attributeName === 'data-tooltip') {
                        if (el.hasAttribute('data-tooltip')) {
                            this.createForElement(el);
                        } else {
                            this.destroyForElement(el);
                        }
                    }
                }
            }
        });

        this.observer.observe(document.body, {
            childList: true,
            subtree: true,
            attributes: true,
            attributeFilter: ['data-tooltip'],
        });
    }

    /**
     * Scan document and create tooltips
     */
    private scanAndCreate(): void {
        document.querySelectorAll<HTMLElement>('[data-tooltip]').forEach(el => {
            this.createForElement(el);
        });
    }

    /**
     * Create tooltip for a single element
     */
    private createForElement(element: HTMLElement): void {
        if (this.tooltips.has(element)) return;
        if (!element.hasAttribute('data-tooltip')) return;

        const content = element.getAttribute('data-tooltip');
        if (!content) return;

        const position = element.getAttribute('data-tooltip-position') as TooltipPosition | null;
        const showDelay = element.getAttribute('data-tooltip-delay');
        const maxWidth = element.getAttribute('data-tooltip-max-width');

        const options: TooltipOptions = {
            ...this.defaultOptions,
            content,
        };

        if (position) options.position = position;
        if (showDelay) options.showDelay = parseInt(showDelay, 10);
        if (maxWidth) options.maxWidth = parseInt(maxWidth, 10);

        const tooltip = createTooltip(element, options);
        this.tooltips.set(element, tooltip);
    }

    /**
     * Destroy tooltip for a single element
     */
    private destroyForElement(element: HTMLElement): void {
        const tooltip = this.tooltips.get(element);
        if (tooltip) {
            tooltip.destroy();
            this.tooltips.delete(element);
        }
    }

    /**
     * Manually create a tooltip for an element
     */
    create(element: HTMLElement, options: TooltipOptions): TooltipInstance {
        this.destroyForElement(element);
        const tooltip = createTooltip(element, options);
        this.tooltips.set(element, tooltip);
        return tooltip;
    }

    /**
     * Get tooltip instance for an element
     */
    get(element: HTMLElement): TooltipInstance | undefined {
        return this.tooltips.get(element);
    }

    /**
     * Destroy all tooltips and clean up
     */
    destroy(): void {
        if (this.observer) {
            this.observer.disconnect();
            this.observer = null;
        }

        for (const tooltip of this.tooltips.values()) {
            tooltip.destroy();
        }
        this.tooltips.clear();
    }

    /**
     * Hide all visible tooltips
     */
    hideAll(): void {
        for (const tooltip of this.tooltips.values()) {
            tooltip.hide();
        }
    }

    /**
     * Get count of managed tooltips
     */
    get count(): number {
        return this.tooltips.size;
    }
}

// Singleton instance for declarative usage
let globalManager: TooltipManager | null = null;

/**
 * Initialize global tooltip manager for declarative usage
 * Call once during app initialization
 */
export function initTooltips(options: Partial<TooltipOptions> = {}): TooltipManager {
    if (globalManager) {
        globalManager.destroy();
    }
    globalManager = new TooltipManager(options);
    return globalManager;
}

/**
 * Get the global tooltip manager
 */
export function getTooltipManager(): TooltipManager | null {
    return globalManager;
}

export default { createTooltip, TooltipManager, initTooltips, getTooltipManager };
