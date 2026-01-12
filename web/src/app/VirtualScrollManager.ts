// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * VirtualScrollManager - Efficient virtual scrolling for large lists
 *
 * Features:
 * - Renders only visible items plus configurable buffer
 * - Uses IntersectionObserver for efficient visibility detection
 * - Maintains scroll position during updates
 * - Supports variable height items with grid layout
 * - Smooth scrolling and momentum preservation
 *
 * @see /docs/working/UI_UX_AUDIT.md
 */

interface VirtualScrollOptions {
  /** Container element for the scrollable content */
  container: HTMLElement;
  /** Height of each item row in pixels (for grid, this is row height) */
  itemHeight: number;
  /** Number of items per row (for grid layout) */
  itemsPerRow: number;
  /** Number of extra rows to render above/below viewport */
  bufferRows?: number;
  /** Callback to render an item at a given index */
  renderItem: (index: number) => HTMLElement | null;
  /** Total number of items */
  totalItems: number;
  /** Optional callback when visible range changes */
  onRangeChange?: (startIndex: number, endIndex: number) => void;
}

interface VisibleRange {
  startRow: number;
  endRow: number;
  startIndex: number;
  endIndex: number;
}

export class VirtualScrollManager {
  private options: Required<VirtualScrollOptions>;
  private scrollContainer: HTMLElement;
  private contentContainer: HTMLElement;
  private spacerTop: HTMLElement;
  private spacerBottom: HTMLElement;
  private visibleRange: VisibleRange = { startRow: 0, endRow: 0, startIndex: 0, endIndex: 0 };
  private renderedItems: Map<number, HTMLElement> = new Map();
  private scrollRAF: number | null = null;
  private resizeObserver: ResizeObserver | null = null;
  private isEnabled = false;
  private itemsData: unknown[] = [];

  constructor(options: VirtualScrollOptions) {
    this.options = {
      bufferRows: 3,
      onRangeChange: () => {},
      ...options
    };

    // Create scroll structure
    this.scrollContainer = this.createScrollContainer();
    this.contentContainer = this.createContentContainer();
    this.spacerTop = this.createSpacer('virtual-scroll-spacer-top');
    this.spacerBottom = this.createSpacer('virtual-scroll-spacer-bottom');

    this.setupStructure();
    this.setupEventListeners();
  }

  /**
   * Enable virtual scrolling with the given data
   */
  enable<T>(items: T[]): void {
    this.itemsData = items;
    this.options.totalItems = items.length;
    this.isEnabled = true;
    this.updateTotalHeight();
    this.updateVisibleRange();
    this.render();
  }

  /**
   * Disable virtual scrolling and render all items normally
   */
  disable(): void {
    this.isEnabled = false;
    this.cleanup();
  }

  /**
   * Update total items count and refresh
   */
  setTotalItems(count: number): void {
    this.options.totalItems = count;
    if (this.isEnabled) {
      this.updateTotalHeight();
      this.updateVisibleRange();
      this.render();
    }
  }

  /**
   * Update items per row (for responsive layouts)
   */
  setItemsPerRow(count: number): void {
    this.options.itemsPerRow = count;
    if (this.isEnabled) {
      this.updateTotalHeight();
      this.updateVisibleRange();
      this.render();
    }
  }

  /**
   * Scroll to a specific item index
   */
  scrollToIndex(index: number, behavior: ScrollBehavior = 'smooth'): void {
    const row = Math.floor(index / this.options.itemsPerRow);
    const scrollTop = row * this.options.itemHeight;
    this.scrollContainer.scrollTo({ top: scrollTop, behavior });
  }

  /**
   * Get the currently visible range
   */
  getVisibleRange(): VisibleRange {
    return { ...this.visibleRange };
  }

  /**
   * Force a re-render of visible items
   */
  refresh(): void {
    if (this.isEnabled) {
      this.clearRenderedItems();
      this.updateVisibleRange();
      this.render();
    }
  }

  /**
   * Check if virtual scrolling is active
   */
  isActive(): boolean {
    return this.isEnabled;
  }

  /**
   * Get the data item at a given index
   */
  getItem<T>(index: number): T | undefined {
    return this.itemsData[index] as T | undefined;
  }

  /**
   * Destroy the virtual scroll manager
   */
  destroy(): void {
    this.cleanup();
    this.resizeObserver?.disconnect();
    this.scrollContainer.removeEventListener('scroll', this.handleScroll);
    window.removeEventListener('resize', this.handleResize);
  }

  private createScrollContainer(): HTMLElement {
    const container = document.createElement('div');
    container.className = 'virtual-scroll-container';
    container.setAttribute('role', 'list');
    container.setAttribute('aria-label', 'Virtual scrolling content');
    return container;
  }

  private createContentContainer(): HTMLElement {
    const container = document.createElement('div');
    container.className = 'virtual-scroll-content';
    return container;
  }

  private createSpacer(className: string): HTMLElement {
    const spacer = document.createElement('div');
    spacer.className = className;
    spacer.setAttribute('aria-hidden', 'true');
    return spacer;
  }

  private setupStructure(): void {
    const { container } = this.options;

    // Wrap existing content
    this.scrollContainer.appendChild(this.spacerTop);
    this.scrollContainer.appendChild(this.contentContainer);
    this.scrollContainer.appendChild(this.spacerBottom);

    // Clear container and add scroll structure
    container.innerHTML = '';
    container.appendChild(this.scrollContainer);
  }

  private setupEventListeners(): void {
    this.handleScroll = this.handleScroll.bind(this);
    this.handleResize = this.handleResize.bind(this);

    this.scrollContainer.addEventListener('scroll', this.handleScroll, { passive: true });
    window.addEventListener('resize', this.handleResize);

    // Watch for container size changes
    this.resizeObserver = new ResizeObserver(() => {
      if (this.isEnabled) {
        this.updateVisibleRange();
        this.render();
      }
    });
    this.resizeObserver.observe(this.options.container);
  }

  private handleScroll(): void {
    if (!this.isEnabled) return;

    // Use RAF to throttle scroll handling
    if (this.scrollRAF !== null) {
      cancelAnimationFrame(this.scrollRAF);
    }

    this.scrollRAF = requestAnimationFrame(() => {
      this.updateVisibleRange();
      this.render();
      this.scrollRAF = null;
    });
  }

  private handleResize(): void {
    if (!this.isEnabled) return;

    // Recalculate items per row based on container width
    const containerWidth = this.options.container.clientWidth;
    const itemWidth = this.getEstimatedItemWidth();

    if (itemWidth > 0) {
      const newItemsPerRow = Math.max(1, Math.floor(containerWidth / itemWidth));
      if (newItemsPerRow !== this.options.itemsPerRow) {
        this.setItemsPerRow(newItemsPerRow);
      }
    }
  }

  private getEstimatedItemWidth(): number {
    // Get width from CSS custom property or use default
    const style = getComputedStyle(this.options.container);
    const gridColumns = style.getPropertyValue('--virtual-scroll-columns');
    if (gridColumns) {
      return this.options.container.clientWidth / parseInt(gridColumns);
    }
    // Default card width estimate (including gap)
    return 220;
  }

  private updateTotalHeight(): void {
    // Calculate total rows for spacer heights (used in updateVisibleRange)
    // Scroll container height is set to allow full scrolling
    this.scrollContainer.style.height = '100%';
    this.scrollContainer.style.overflowY = 'auto';
  }

  private updateVisibleRange(): void {
    const { itemHeight, itemsPerRow, bufferRows, totalItems } = this.options;

    const scrollTop = this.scrollContainer.scrollTop;
    const viewportHeight = this.scrollContainer.clientHeight;

    // Calculate visible rows
    const firstVisibleRow = Math.floor(scrollTop / itemHeight);
    const visibleRowCount = Math.ceil(viewportHeight / itemHeight);

    // Add buffer
    const startRow = Math.max(0, firstVisibleRow - bufferRows);
    const endRow = Math.min(
      Math.ceil(totalItems / itemsPerRow),
      firstVisibleRow + visibleRowCount + bufferRows
    );

    // Calculate item indices
    const startIndex = startRow * itemsPerRow;
    const endIndex = Math.min(totalItems, endRow * itemsPerRow);

    const rangeChanged =
      this.visibleRange.startIndex !== startIndex ||
      this.visibleRange.endIndex !== endIndex;

    this.visibleRange = { startRow, endRow, startIndex, endIndex };

    // Update spacers
    this.spacerTop.style.height = `${startRow * itemHeight}px`;
    const totalRows = Math.ceil(totalItems / itemsPerRow);
    this.spacerBottom.style.height = `${Math.max(0, (totalRows - endRow) * itemHeight)}px`;

    if (rangeChanged) {
      this.options.onRangeChange(startIndex, endIndex);
    }
  }

  private render(): void {
    const { startIndex, endIndex } = this.visibleRange;

    // Remove items outside visible range
    this.renderedItems.forEach((element, index) => {
      if (index < startIndex || index >= endIndex) {
        element.remove();
        this.renderedItems.delete(index);
      }
    });

    // Render items in visible range
    const fragment = document.createDocumentFragment();

    for (let i = startIndex; i < endIndex; i++) {
      if (!this.renderedItems.has(i)) {
        const element = this.options.renderItem(i);
        if (element) {
          element.setAttribute('data-virtual-index', i.toString());
          element.setAttribute('role', 'listitem');
          this.renderedItems.set(i, element);
          fragment.appendChild(element);
        }
      }
    }

    this.contentContainer.appendChild(fragment);

    // Sort children by index for consistent ordering
    const children = Array.from(this.contentContainer.children) as HTMLElement[];
    children.sort((a, b) => {
      const indexA = parseInt(a.getAttribute('data-virtual-index') || '0');
      const indexB = parseInt(b.getAttribute('data-virtual-index') || '0');
      return indexA - indexB;
    });

    // Re-append in sorted order
    children.forEach(child => this.contentContainer.appendChild(child));
  }

  private clearRenderedItems(): void {
    this.renderedItems.forEach(element => element.remove());
    this.renderedItems.clear();
  }

  private cleanup(): void {
    if (this.scrollRAF !== null) {
      cancelAnimationFrame(this.scrollRAF);
      this.scrollRAF = null;
    }
    this.clearRenderedItems();
    this.spacerTop.style.height = '0';
    this.spacerBottom.style.height = '0';
  }
}

export default VirtualScrollManager;
