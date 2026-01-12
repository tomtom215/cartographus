// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * ImageOptimizationManager - Image Lazy Loading and Optimization
 *
 * Provides utilities for optimizing image loading performance:
 * - Intersection Observer-based lazy loading
 * - Low-Quality Image Placeholders (LQIP)
 * - Graceful fallback handling
 * - Progressive image loading
 *
 * @see /docs/working/UI_UX_AUDIT.md
 */

import { createLogger } from '../lib/logger';

const logger = createLogger('ImageOptimization');

/**
 * Configuration options for image optimization
 */
interface ImageOptimizationConfig {
    /** Root margin for IntersectionObserver (e.g., '100px' to preload) */
    rootMargin?: string;
    /** Threshold for triggering load (0-1) */
    threshold?: number;
    /** Placeholder data URI or URL */
    placeholderSrc?: string;
    /** Enable fade-in animation on load */
    fadeIn?: boolean;
    /** Fade-in duration in ms */
    fadeDuration?: number;
    /** Fallback image for errors */
    fallbackSrc?: string;
    /** Enable native lazy loading attribute */
    useNativeLazy?: boolean;
}

const DEFAULT_CONFIG: ImageOptimizationConfig = {
    rootMargin: '100px', // Preload images 100px before visible
    threshold: 0.1,
    fadeIn: true,
    fadeDuration: 200,
    useNativeLazy: true
};

/**
 * Default placeholder - simple gray rectangle
 */
const DEFAULT_PLACEHOLDER = 'data:image/svg+xml,%3Csvg xmlns="http://www.w3.org/2000/svg" width="100" height="150" viewBox="0 0 100 150"%3E%3Crect fill="%232a2d3e" width="100" height="150"/%3E%3C/svg%3E';

/**
 * Default fallback - simple error state
 */
const DEFAULT_FALLBACK = 'data:image/svg+xml,%3Csvg xmlns="http://www.w3.org/2000/svg" width="100" height="150" viewBox="0 0 100 150"%3E%3Crect fill="%232a2d3e" width="100" height="150"/%3E%3Ctext x="50" y="75" text-anchor="middle" fill="%23666" font-size="12"%3ENo Image%3C/text%3E%3C/svg%3E';

export class ImageOptimizationManager {
    private observer: IntersectionObserver | null = null;
    private config: ImageOptimizationConfig;
    private loadedImages: Set<string> = new Set();

    constructor(config: Partial<ImageOptimizationConfig> = {}) {
        this.config = { ...DEFAULT_CONFIG, ...config };
        this.initObserver();
    }

    /**
     * Initialize IntersectionObserver for lazy loading
     */
    private initObserver(): void {
        if (!('IntersectionObserver' in window)) {
            // Fallback for older browsers - load all images immediately
            logger.warn('IntersectionObserver not supported. Images will load immediately.');
            return;
        }

        this.observer = new IntersectionObserver(
            (entries) => this.handleIntersection(entries),
            {
                rootMargin: this.config.rootMargin,
                threshold: this.config.threshold
            }
        );
    }

    /**
     * Handle intersection events
     */
    private handleIntersection(entries: IntersectionObserverEntry[]): void {
        entries.forEach(entry => {
            if (entry.isIntersecting) {
                const img = entry.target as HTMLImageElement;
                this.loadImage(img);
                this.observer?.unobserve(img);
            }
        });
    }

    /**
     * Load the actual image source
     */
    private loadImage(img: HTMLImageElement): void {
        const actualSrc = img.dataset.src;
        if (!actualSrc) return;

        // Create a preload image to check if it loads successfully
        const preloadImg = new Image();

        preloadImg.onload = () => {
            img.src = actualSrc;
            img.removeAttribute('data-src');
            this.loadedImages.add(actualSrc);

            if (this.config.fadeIn) {
                img.style.opacity = '0';
                img.style.transition = `opacity ${this.config.fadeDuration}ms ease`;
                // Trigger reflow
                void img.offsetHeight;
                img.style.opacity = '1';
            }

            img.classList.remove('lazy-loading');
            img.classList.add('lazy-loaded');
        };

        preloadImg.onerror = () => {
            // Use fallback image on error
            const fallback = this.config.fallbackSrc || DEFAULT_FALLBACK;
            img.src = fallback;
            img.removeAttribute('data-src');
            img.classList.remove('lazy-loading');
            img.classList.add('lazy-error');
        };

        img.classList.add('lazy-loading');
        preloadImg.src = actualSrc;
    }

    /**
     * Observe an image element for lazy loading
     *
     * @param img - Image element to observe
     * @param actualSrc - The actual image source to load
     * @param placeholderSrc - Optional placeholder source
     */
    observe(img: HTMLImageElement, actualSrc: string, placeholderSrc?: string): void {
        // Store actual source in data attribute
        img.dataset.src = actualSrc;

        // Set placeholder
        const placeholder = placeholderSrc || this.config.placeholderSrc || DEFAULT_PLACEHOLDER;
        img.src = placeholder;

        // Add native lazy loading if supported and enabled
        if (this.config.useNativeLazy) {
            img.loading = 'lazy';
        }

        // Add to observer
        if (this.observer) {
            this.observer.observe(img);
        } else {
            // Fallback - load immediately
            this.loadImage(img);
        }
    }

    /**
     * Stop observing an image
     */
    unobserve(img: HTMLImageElement): void {
        this.observer?.unobserve(img);
    }

    /**
     * Create an optimized image element
     *
     * @param src - The image source URL
     * @param alt - Alt text for accessibility
     * @param options - Additional options
     */
    createOptimizedImage(
        src: string,
        alt: string,
        options: {
            className?: string;
            placeholder?: string;
            fallback?: string;
            width?: number;
            height?: number;
        } = {}
    ): HTMLImageElement {
        const img = document.createElement('img');

        img.alt = alt;
        img.className = `lazy-image ${options.className || ''}`.trim();

        if (options.width) img.width = options.width;
        if (options.height) img.height = options.height;

        // Observe for lazy loading
        this.observe(img, src, options.placeholder);

        return img;
    }

    /**
     * Create an image with HTML string (for use in templates)
     *
     * @param src - The image source URL
     * @param alt - Alt text for accessibility
     * @param options - Additional options
     */
    static createImageHTML(
        src: string,
        alt: string,
        options: {
            className?: string;
            placeholder?: string;
            width?: number;
            height?: number;
            loading?: 'lazy' | 'eager';
        } = {}
    ): string {
        const placeholder = options.placeholder || DEFAULT_PLACEHOLDER;
        const escapedSrc = ImageOptimizationManager.escapeHtml(src);
        const escapedAlt = ImageOptimizationManager.escapeHtml(alt);

        const attrs = [
            `src="${placeholder}"`,
            `data-src="${escapedSrc}"`,
            `alt="${escapedAlt}"`,
            `class="lazy-image ${options.className || ''}"`,
            `loading="${options.loading || 'lazy'}"`,
        ];

        if (options.width) attrs.push(`width="${options.width}"`);
        if (options.height) attrs.push(`height="${options.height}"`);

        return `<img ${attrs.join(' ')}>`;
    }

    /**
     * Escape HTML special characters
     */
    private static escapeHtml(unsafe: string): string {
        return unsafe
            .replace(/&/g, '&amp;')
            .replace(/</g, '&lt;')
            .replace(/>/g, '&gt;')
            .replace(/"/g, '&quot;')
            .replace(/'/g, '&#039;');
    }

    /**
     * Preload critical images (above the fold)
     *
     * @param urls - Array of image URLs to preload
     */
    preloadImages(urls: string[]): Promise<void[]> {
        const promises = urls.map(url => {
            return new Promise<void>((resolve, reject) => {
                if (this.loadedImages.has(url)) {
                    resolve();
                    return;
                }

                const img = new Image();
                img.onload = () => {
                    this.loadedImages.add(url);
                    resolve();
                };
                img.onerror = () => reject(new Error(`Failed to preload: ${url}`));
                img.src = url;
            });
        });

        return Promise.all(promises);
    }

    /**
     * Initialize lazy loading for existing images in a container
     *
     * @param container - Container element to scan for images
     * @param selector - CSS selector for images (default: 'img[data-src]')
     */
    initializeContainer(container: HTMLElement, selector = 'img[data-src]'): void {
        const images = container.querySelectorAll<HTMLImageElement>(selector);
        images.forEach(img => {
            const src = img.dataset.src;
            if (src && !this.loadedImages.has(src)) {
                this.observe(img, src);
            }
        });
    }

    /**
     * Check if an image has been loaded
     */
    isLoaded(url: string): boolean {
        return this.loadedImages.has(url);
    }

    /**
     * Get count of loaded images
     */
    getLoadedCount(): number {
        return this.loadedImages.size;
    }

    /**
     * Cleanup - disconnect observer
     */
    destroy(): void {
        this.observer?.disconnect();
        this.observer = null;
        this.loadedImages.clear();
    }
}

/**
 * Default singleton instance
 */
let defaultInstance: ImageOptimizationManager | null = null;

/**
 * Get or create the default ImageOptimizationManager instance
 */
export function getImageOptimizer(): ImageOptimizationManager {
    if (!defaultInstance) {
        defaultInstance = new ImageOptimizationManager();
    }
    return defaultInstance;
}

/**
 * CSS classes used by the manager (for documentation)
 *
 * .lazy-image - Base class for lazy-loaded images
 * .lazy-loading - Applied while image is loading
 * .lazy-loaded - Applied after image loads successfully
 * .lazy-error - Applied when image fails to load
 */
