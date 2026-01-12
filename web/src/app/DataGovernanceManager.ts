// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * DataGovernanceManager - Data Governance Tab Orchestrator
 *
 * Main orchestrator for the Data Governance tab providing centralized visibility
 * into data integrity, audit trails, and system operations.
 *
 * This module coordinates 9 specialized page renderers:
 * - Overview: Key metrics dashboard
 * - Deduplication: Dedupe audit log (ADR-0022)
 * - Detection: Security alerts (ADR-0020)
 * - Sync Status: Event pipeline metrics
 * - Backups: Backup/restore management
 * - Health: System health monitoring
 * - Audit: Security audit log
 * - Lineage: Event tracing
 * - Failed Events: Dead letter queue
 */

import type { API } from '../lib/api';
import { createLogger } from '../lib/logger';

const logger = createLogger('DataGovernanceManager');
import {
    DEFAULT_GOVERNANCE_CONFIG,
    GovernanceConfig,
    OverviewRenderer,
    DedupeRenderer,
    DetectionRenderer,
    SyncRenderer,
    BackupsRenderer,
    HealthRenderer,
    AuditRenderer,
    LineageRenderer,
    DLQRenderer,
} from './governance';

/** Available governance sub-pages */
export type GovernancePage = 'overview' | 'deduplication' | 'detection' | 'sync' | 'backups' | 'health' | 'lineage' | 'audit' | 'failed';

export class DataGovernanceManager {
    private config: GovernanceConfig;
    private currentPage: GovernancePage = 'overview';
    private initialized = false;
    private refreshInterval: ReturnType<typeof setInterval> | null = null;

    // Page renderers
    private readonly overview: OverviewRenderer;
    private readonly dedupe: DedupeRenderer;
    private readonly detection: DetectionRenderer;
    private readonly sync: SyncRenderer;
    private readonly backups: BackupsRenderer;
    private readonly health: HealthRenderer;
    private readonly audit: AuditRenderer;
    private readonly lineage: LineageRenderer;
    private readonly dlq: DLQRenderer;

    // Event handler references for cleanup
    private navClickHandler: ((e: Event) => void) | null = null;

    constructor(api: API, config: Partial<GovernanceConfig> = {}) {
        this.config = { ...DEFAULT_GOVERNANCE_CONFIG, ...config };

        // Initialize all page renderers
        this.overview = new OverviewRenderer(api, this.config);
        this.dedupe = new DedupeRenderer(api, this.config);
        this.detection = new DetectionRenderer(api, this.config);
        this.sync = new SyncRenderer(api, this.config);
        this.backups = new BackupsRenderer(api, this.config);
        this.health = new HealthRenderer(api, this.config);
        this.audit = new AuditRenderer(api, this.config);
        this.lineage = new LineageRenderer(api, this.config);
        this.dlq = new DLQRenderer(api, this.config);
    }

    /**
     * Initialize the data governance manager
     */
    async init(): Promise<void> {
        if (this.initialized) {
            // Already initialized, just refresh data
            await this.refreshCurrentPage();
            return;
        }

        logger.debug('[DataGovernanceManager] Initializing...');

        this.setupNavigation();
        this.setupAllEventListeners();
        await this.loadOverviewData();
        this.startAutoRefresh();
        this.initialized = true;

        logger.debug('[DataGovernanceManager] Initialization complete');
    }

    /**
     * Set up sub-page navigation
     */
    private setupNavigation(): void {
        const navTabs = document.querySelectorAll('.governance-nav-tab');

        this.navClickHandler = (e: Event) => {
            const button = e.currentTarget as HTMLButtonElement;
            const page = button.getAttribute('data-governance-page') as GovernancePage;
            if (page) {
                this.switchPage(page);
            }
        };

        navTabs.forEach(tab => {
            tab.addEventListener('click', this.navClickHandler!);
        });
    }

    /**
     * Set up event listeners for all page renderers
     */
    private setupAllEventListeners(): void {
        this.overview.setupEventListeners();
        this.dedupe.setupEventListeners();
        this.detection.setupEventListeners();
        this.sync.setupEventListeners();
        this.backups.setupEventListeners();
        this.health.setupEventListeners();
        this.audit.setupEventListeners();
        this.lineage.setupEventListeners();
        this.dlq.setupEventListeners();
    }

    /**
     * Switch to a different governance page
     */
    async switchPage(page: GovernancePage): Promise<void> {
        if (page === this.currentPage) return;

        logger.debug(`[DataGovernanceManager] Switching to page: ${page}`);
        this.currentPage = page;

        // Update tab active states
        const navTabs = document.querySelectorAll('.governance-nav-tab');
        navTabs.forEach(tab => {
            const tabPage = tab.getAttribute('data-governance-page');
            const isActive = tabPage === page;
            tab.classList.toggle('active', isActive);
            tab.setAttribute('aria-selected', String(isActive));
        });

        // Hide all pages and show the selected one
        const pages = document.querySelectorAll('.governance-page');
        pages.forEach(pageEl => {
            const pageName = pageEl.getAttribute('data-page');
            (pageEl as HTMLElement).style.display = pageName === page ? 'block' : 'none';
        });

        // Load data for the page
        await this.loadPageData(page);
    }

    /**
     * Load data for a specific page
     */
    private async loadPageData(page: GovernancePage): Promise<void> {
        // Sync shared data between renderers before loading
        this.syncSharedData();

        switch (page) {
            case 'overview':
                await this.loadOverviewData();
                break;
            case 'deduplication':
                await this.dedupe.load();
                break;
            case 'detection':
                await this.detection.load();
                break;
            case 'sync':
                await this.sync.load();
                break;
            case 'backups':
                await this.backups.load();
                break;
            case 'health':
                await this.health.load();
                break;
            case 'audit':
                await this.audit.load();
                break;
            case 'lineage':
                await this.lineage.load();
                break;
            case 'failed':
                await this.dlq.load();
                break;
        }
    }

    /**
     * Load overview dashboard data
     */
    private async loadOverviewData(): Promise<void> {
        // Load dedupe and detection data first for the overview
        await Promise.all([
            this.dedupe.load(),
            this.detection.load(),
            this.dlq.load(),
        ]);

        // Sync data to overview renderer
        this.syncSharedData();

        // Now load overview
        await this.overview.load();
    }

    /**
     * Sync shared data between renderers
     */
    private syncSharedData(): void {
        // Overview needs data from other renderers
        this.overview.setDedupeStats(this.dedupe.getStats());
        this.overview.setDedupeEntries(this.dedupe.getEntries());
        this.overview.setDetectionStats(this.detection.getStats());
        this.overview.setDetectionAlerts(this.detection.getAlerts());
        this.overview.setDLQStats(this.dlq.getStats());

        // Sync renderer needs dedupe and DLQ stats
        this.sync.setDedupeStats(this.dedupe.getStats());
        this.sync.setDLQStats(this.dlq.getStats());
        this.sync.setSyncSources(this.overview.getSyncSources());

        // Health renderer needs backup stats
        this.health.setBackupStats(this.backups.getStats());

        // Lineage renderer needs dedupe entries
        this.lineage.setDedupeEntries(this.dedupe.getEntries());
    }

    /**
     * Refresh data for the current page
     */
    private async refreshCurrentPage(): Promise<void> {
        await this.loadPageData(this.currentPage);
    }

    /**
     * Start auto-refresh interval
     */
    private startAutoRefresh(): void {
        this.refreshInterval = setInterval(() => {
            this.refreshCurrentPage();
        }, this.config.autoRefreshMs);
    }

    /**
     * Cleanup resources
     */
    destroy(): void {
        if (this.refreshInterval) {
            clearInterval(this.refreshInterval);
            this.refreshInterval = null;
        }

        // Remove navigation listeners
        if (this.navClickHandler) {
            const navTabs = document.querySelectorAll('.governance-nav-tab');
            navTabs.forEach(tab => {
                tab.removeEventListener('click', this.navClickHandler!);
            });
        }

        this.initialized = false;
        logger.debug('[DataGovernanceManager] Destroyed');
    }
}
