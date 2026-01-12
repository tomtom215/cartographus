// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * NewsletterManager - Newsletter Generator Dashboard Management
 *
 * Orchestrates all newsletter sub-page renderers and handles
 * navigation between newsletter management views.
 */

import { createLogger } from '../lib/logger';
import type { API } from '../lib/api';
import {
    DEFAULT_NEWSLETTER_CONFIG,
    OverviewRenderer,
    TemplatesRenderer,
    SchedulesRenderer,
    DeliveriesRenderer,
} from './newsletter';
import type { NewsletterConfig } from './newsletter';

const logger = createLogger('NewsletterManager');

/** Newsletter page types */
export type NewsletterPage = 'overview' | 'templates' | 'schedules' | 'deliveries';

/**
 * NewsletterManager - Main orchestrator for newsletter UI
 */
export class NewsletterManager {
    private config: NewsletterConfig;
    private currentPage: NewsletterPage = 'overview';
    private refreshInterval: ReturnType<typeof setInterval> | null = null;

    // Renderers
    private overviewRenderer: OverviewRenderer;
    private templatesRenderer: TemplatesRenderer;
    private schedulesRenderer: SchedulesRenderer;
    private deliveriesRenderer: DeliveriesRenderer;

    constructor(api: API, config: NewsletterConfig = DEFAULT_NEWSLETTER_CONFIG) {
        this.config = config;

        // Initialize renderers
        this.overviewRenderer = new OverviewRenderer(api, config);
        this.templatesRenderer = new TemplatesRenderer(api, config);
        this.schedulesRenderer = new SchedulesRenderer(api, config);
        this.deliveriesRenderer = new DeliveriesRenderer(api, config);
    }

    // =========================================================================
    // Lifecycle
    // =========================================================================

    /**
     * Initialize the newsletter manager
     */
    async init(): Promise<void> {
        logger.debug('Initializing NewsletterManager');

        // Setup event listeners for all renderers
        this.overviewRenderer.setupEventListeners();
        this.templatesRenderer.setupEventListeners();
        this.schedulesRenderer.setupEventListeners();
        this.deliveriesRenderer.setupEventListeners();

        // Setup navigation
        this.setupNavigation();

        // Load initial page
        await this.showPage('overview');

        // Start auto-refresh
        this.startAutoRefresh();

        logger.info('NewsletterManager initialized');
    }

    /**
     * Destroy the newsletter manager and cleanup resources
     */
    destroy(): void {
        logger.debug('Destroying NewsletterManager');
        this.stopAutoRefresh();
    }

    // =========================================================================
    // Navigation
    // =========================================================================

    private setupNavigation(): void {
        // Tab navigation
        document.querySelectorAll('[data-newsletter-page]').forEach(tab => {
            tab.addEventListener('click', async (e) => {
                e.preventDefault();
                const page = (e.currentTarget as HTMLElement).getAttribute('data-newsletter-page') as NewsletterPage;
                if (page) {
                    await this.showPage(page);
                }
            });
        });
    }

    /**
     * Show a specific newsletter page
     */
    async showPage(page: NewsletterPage): Promise<void> {
        if (page === this.currentPage) return;

        this.currentPage = page;

        // Update tab active states
        document.querySelectorAll('[data-newsletter-page]').forEach(tab => {
            const tabPage = tab.getAttribute('data-newsletter-page');
            tab.classList.toggle('active', tabPage === page);
            tab.setAttribute('aria-selected', String(tabPage === page));
        });

        // Hide all pages
        document.querySelectorAll('.newsletter-page').forEach(pageEl => {
            (pageEl as HTMLElement).style.display = 'none';
        });

        // Show current page
        const pageEl = document.getElementById(`newsletter-${page}-page`);
        if (pageEl) {
            pageEl.style.display = 'block';
        }

        // Load page data
        await this.loadCurrentPage();
    }

    /**
     * Load data for the current page
     */
    private async loadCurrentPage(): Promise<void> {
        try {
            switch (this.currentPage) {
                case 'overview':
                    await this.overviewRenderer.load();
                    break;
                case 'templates':
                    await this.templatesRenderer.load();
                    break;
                case 'schedules':
                    await this.schedulesRenderer.load();
                    break;
                case 'deliveries':
                    await this.deliveriesRenderer.load();
                    break;
            }
        } catch (error) {
            logger.error('Failed to load newsletter page:', error);
        }
    }

    /**
     * Refresh the current page
     */
    async refresh(): Promise<void> {
        await this.loadCurrentPage();
    }

    // =========================================================================
    // Auto-Refresh
    // =========================================================================

    private startAutoRefresh(): void {
        if (this.refreshInterval) return;

        this.refreshInterval = setInterval(() => {
            // Only auto-refresh overview and deliveries pages
            if (this.currentPage === 'overview' || this.currentPage === 'deliveries') {
                this.loadCurrentPage().catch(err => {
                    logger.error('Auto-refresh failed:', err);
                });
            }
        }, this.config.autoRefreshMs);
    }

    private stopAutoRefresh(): void {
        if (this.refreshInterval) {
            clearInterval(this.refreshInterval);
            this.refreshInterval = null;
        }
    }

    // =========================================================================
    // Public API
    // =========================================================================

    /**
     * Get the current page
     */
    getCurrentPage(): NewsletterPage {
        return this.currentPage;
    }

    /**
     * Navigate to a specific template for editing
     */
    async editTemplate(_templateId: string): Promise<void> {
        await this.showPage('templates');
        // Trigger edit modal through the templates renderer
        // This could be enhanced with a direct method using _templateId
    }

    /**
     * Navigate to a specific schedule
     */
    async viewSchedule(_scheduleId: string): Promise<void> {
        await this.showPage('schedules');
        // Could highlight the specific schedule using _scheduleId
    }

    /**
     * Navigate to deliveries for a specific schedule
     */
    async viewScheduleDeliveries(_scheduleId: string): Promise<void> {
        await this.showPage('deliveries');
        // Could filter by schedule using _scheduleId
    }
}
