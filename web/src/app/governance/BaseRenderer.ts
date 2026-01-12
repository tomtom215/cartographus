// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * BaseRenderer - Base class for governance page renderers
 *
 * Provides shared utilities for all governance sub-page renderers.
 */

import type { API } from '../../lib/api';

/** Configuration for governance renderers */
export interface GovernanceConfig {
    autoRefreshMs: number;
    maxTableEntries: number;
}

export const DEFAULT_GOVERNANCE_CONFIG: GovernanceConfig = {
    autoRefreshMs: 60000, // 1 minute
    maxTableEntries: 50,
};

/** Status display configuration */
export interface StatusDisplayConfig {
    name: string;
    color: string;
    icon: string;
}

/** Reason display names */
export const REASON_NAMES: Record<string, string> = {
    event_id: 'Event ID Match',
    session_key: 'Session Key Match',
    correlation_key: 'Correlation Key Match',
    cross_source_key: 'Cross-Source Match',
    db_constraint: 'Database Constraint',
};

/** Layer display names */
export const LAYER_NAMES: Record<string, string> = {
    bloom_cache: 'BloomLRU Cache',
    nats_dedup: 'NATS JetStream',
    db_unique: 'DuckDB Index',
};

/** Status display names and colors */
export const STATUS_CONFIG: Record<string, StatusDisplayConfig> = {
    auto_dedupe: { name: 'Pending Review', color: '#f39c12', icon: '\u26A0' },
    user_confirmed: { name: 'Confirmed', color: '#27ae60', icon: '\u2714' },
    user_restored: { name: 'Restored', color: '#3498db', icon: '\u21A9' },
};

/** Severity colors for alerts */
export const SEVERITY_COLORS: Record<string, string> = {
    critical: '#e74c3c',
    error: '#c0392b',
    warning: '#f39c12',
    info: '#3498db',
    debug: '#95a5a6',
};

/**
 * Base renderer class with shared utilities
 */
export abstract class BaseRenderer {
    protected api: API;
    protected config: GovernanceConfig;

    constructor(api: API, config: GovernanceConfig = DEFAULT_GOVERNANCE_CONFIG) {
        this.api = api;
        this.config = config;
    }

    /**
     * Load data for this page
     */
    abstract load(): Promise<void>;

    /**
     * Setup event listeners for this page
     */
    abstract setupEventListeners(): void;

    // =========================================================================
    // DOM Utilities
    // =========================================================================

    protected setElementText(id: string, text: string): void {
        const el = document.getElementById(id);
        if (el) el.textContent = text;
    }

    protected escapeHtml(str: string): string {
        const div = document.createElement('div');
        div.textContent = str;
        return div.innerHTML;
    }

    // =========================================================================
    // Formatting Utilities
    // =========================================================================

    protected formatTimestamp(timestamp: string): string {
        try {
            const date = new Date(timestamp);
            return date.toLocaleString();
        } catch {
            return timestamp;
        }
    }

    protected formatTimeAgo(date: Date): string {
        const now = new Date();
        const diffMs = now.getTime() - date.getTime();
        const diffMins = Math.floor(diffMs / 60000);
        const diffHours = Math.floor(diffMins / 60);
        const diffDays = Math.floor(diffHours / 24);

        if (diffMins < 1) return 'just now';
        if (diffMins < 60) return `${diffMins}m ago`;
        if (diffHours < 24) return `${diffHours}h ago`;
        return `${diffDays}d ago`;
    }

    protected formatRuleType(ruleType: string): string {
        return ruleType.replace(/_/g, ' ').replace(/\b\w/g, l => l.toUpperCase());
    }

    protected formatBytes(bytes: number): string {
        if (bytes === 0) return '0 B';
        const k = 1024;
        const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
        const i = Math.floor(Math.log(bytes) / Math.log(k));
        return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
    }

    protected formatUptime(seconds: number): string {
        if (seconds < 60) return `${seconds}s`;
        if (seconds < 3600) return `${Math.floor(seconds / 60)}m ${seconds % 60}s`;
        if (seconds < 86400) {
            const hours = Math.floor(seconds / 3600);
            const mins = Math.floor((seconds % 3600) / 60);
            return `${hours}h ${mins}m`;
        }
        const days = Math.floor(seconds / 86400);
        const hours = Math.floor((seconds % 86400) / 3600);
        return `${days}d ${hours}h`;
    }

    // =========================================================================
    // Input Utilities
    // =========================================================================

    protected debounce<T extends (...args: any[]) => void>(
        fn: T,
        delay: number
    ): (...args: Parameters<T>) => void {
        let timeoutId: ReturnType<typeof setTimeout> | null = null;
        return (...args: Parameters<T>) => {
            if (timeoutId) {
                clearTimeout(timeoutId);
            }
            timeoutId = setTimeout(() => fn(...args), delay);
        };
    }
}
