// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * PlexIntegrationManager - Plex Integration Status and Control Panel
 *
 * This manager displays the status of Plex integration including:
 * - WebSocket connection status
 * - Session polling status
 * - Active sessions from Plex
 * - Real-time playback events
 *
 * Architecture (ADR-0009, ADR-0020):
 * - Plex WebSocket provides real-time notifications
 * - Session poller provides fallback and refresh
 * - Events flow to detection engine for security monitoring
 */

import { createLogger } from '../lib/logger';
import { PlexDirectSession } from '../lib/types';
import { queuedFetch } from '../lib/api/client';

const logger = createLogger('PlexIntegration');

export interface PlexIntegrationStatus {
    plexEnabled: boolean;
    websocketConnected: boolean;
    pollerActive: boolean;
    activeSessions: number;
    lastUpdate: Date | null;
}

export interface PlexRealtimeEvent {
    sessionKey: string;
    state: string;
    ratingKey: string;
    viewOffset: number;
    isBuffering: boolean;
    isNewSession?: boolean;
    transcodeSession?: string;
}

export class PlexIntegrationManager {
    private container: HTMLElement | null = null;
    private statusPanel: HTMLElement | null = null;
    private sessionsPanel: HTMLElement | null = null;
    private status: PlexIntegrationStatus;
    private sessions: PlexDirectSession[] = [];
    private refreshInterval: ReturnType<typeof setInterval> | null = null;
    private wsListenerRegistered = false;

    // Event handler references for cleanup
    private refreshButtonHandler: (() => void) | null = null;
    private wsEventHandler: ((event: Event) => void) | null = null;

    constructor() {
        this.status = {
            plexEnabled: false,
            websocketConnected: false,
            pollerActive: false,
            activeSessions: 0,
            lastUpdate: null
        };
    }

    /**
     * Initialize the Plex integration panel
     */
    public async init(containerId: string): Promise<void> {
        this.container = document.getElementById(containerId);
        if (!this.container) {
            logger.warn(`Container ${containerId} not found`);
            return;
        }

        this.render();
        await this.fetchStatus();
        this.startAutoRefresh();
        this.registerWebSocketListener();
    }

    /**
     * Destroy the manager and cleanup resources
     */
    public destroy(): void {
        // Clear refresh interval
        if (this.refreshInterval) {
            clearInterval(this.refreshInterval);
            this.refreshInterval = null;
        }

        // Remove refresh button handler
        const refreshBtn = document.getElementById('plex-refresh-btn');
        if (refreshBtn && this.refreshButtonHandler) {
            refreshBtn.removeEventListener('click', this.refreshButtonHandler);
            this.refreshButtonHandler = null;
        }

        // Remove WebSocket event listener
        if (this.wsEventHandler) {
            window.removeEventListener('plex_realtime_playback', this.wsEventHandler);
            this.wsEventHandler = null;
            this.wsListenerRegistered = false;
        }

        // Clear container
        if (this.container) {
            this.container.innerHTML = '';
        }
    }

    /**
     * Fetch Plex integration status from API
     */
    private async fetchStatus(): Promise<void> {
        try {
            // Fetch Plex sessions to determine status
            const response = await queuedFetch('/api/v1/plex/sessions');

            if (response.ok) {
                const data = await response.json();
                this.status.plexEnabled = true;
                this.sessions = data.data?.MediaContainer?.Metadata || [];
                this.status.activeSessions = this.sessions.length;
                this.status.lastUpdate = new Date();
            } else if (response.status === 503) {
                // Plex disabled
                this.status.plexEnabled = false;
                this.sessions = [];
                this.status.activeSessions = 0;
            }

            this.updateUI();
        } catch (error) {
            logger.error('Failed to fetch status:', error);
            this.status.plexEnabled = false;
            this.updateUI();
        }
    }

    /**
     * Render the integration panel
     */
    private render(): void {
        if (!this.container) return;

        this.container.innerHTML = `
            <div class="plex-integration-panel">
                <div class="plex-integration-header">
                    <h3 class="plex-integration-title">Plex Integration</h3>
                    <button id="plex-refresh-btn" class="plex-refresh-btn" title="Refresh status">
                        Refresh
                    </button>
                </div>

                <div id="plex-status-panel" class="plex-status-panel">
                    <!-- Status indicators will be rendered here -->
                </div>

                <div id="plex-sessions-panel" class="plex-sessions-panel">
                    <!-- Active sessions will be rendered here -->
                </div>
            </div>
        `;

        this.statusPanel = document.getElementById('plex-status-panel');
        this.sessionsPanel = document.getElementById('plex-sessions-panel');

        // Add refresh button handler (store reference for cleanup)
        const refreshBtn = document.getElementById('plex-refresh-btn');
        if (refreshBtn) {
            this.refreshButtonHandler = () => this.fetchStatus();
            refreshBtn.addEventListener('click', this.refreshButtonHandler);
        }
    }

    /**
     * Update the UI with current status
     */
    private updateUI(): void {
        this.renderStatusPanel();
        this.renderSessionsPanel();
    }

    /**
     * Render the status indicators
     */
    private renderStatusPanel(): void {
        if (!this.statusPanel) return;

        const statusClass = this.status.plexEnabled ? 'status-online' : 'status-offline';
        const wsClass = this.status.websocketConnected ? 'status-online' : 'status-offline';

        this.statusPanel.innerHTML = `
            <div class="plex-status-grid">
                <div class="plex-status-item">
                    <span class="plex-status-indicator ${statusClass}"></span>
                    <span class="plex-status-label">Plex Integration</span>
                    <span class="plex-status-value">${this.status.plexEnabled ? 'Enabled' : 'Disabled'}</span>
                </div>
                <div class="plex-status-item">
                    <span class="plex-status-indicator ${wsClass}"></span>
                    <span class="plex-status-label">WebSocket</span>
                    <span class="plex-status-value">${this.status.websocketConnected ? 'Connected' : 'Disconnected'}</span>
                </div>
                <div class="plex-status-item">
                    <span class="plex-status-indicator status-online"></span>
                    <span class="plex-status-label">Active Sessions</span>
                    <span class="plex-status-value">${this.status.activeSessions}</span>
                </div>
                <div class="plex-status-item">
                    <span class="plex-status-label">Last Update</span>
                    <span class="plex-status-value">${this.status.lastUpdate ? this.formatTime(this.status.lastUpdate) : 'Never'}</span>
                </div>
            </div>
        `;
    }

    /**
     * Render the active sessions list
     */
    private renderSessionsPanel(): void {
        if (!this.sessionsPanel) return;

        if (!this.status.plexEnabled) {
            this.sessionsPanel.innerHTML = `
                <div class="plex-sessions-empty">
                    <p>Plex integration is not enabled.</p>
                    <p>Configure PLEX_URL and PLEX_TOKEN to enable.</p>
                </div>
            `;
            return;
        }

        if (this.sessions.length === 0) {
            this.sessionsPanel.innerHTML = `
                <div class="plex-sessions-empty">
                    <p>No active sessions</p>
                </div>
            `;
            return;
        }

        const sessionsHtml = this.sessions.map(session => this.renderSessionCard(session)).join('');

        this.sessionsPanel.innerHTML = `
            <div class="plex-sessions-header">
                <h4>Active Sessions</h4>
            </div>
            <div class="plex-sessions-list">
                ${sessionsHtml}
            </div>
        `;
    }

    /**
     * Render a single session card
     */
    private renderSessionCard(session: PlexDirectSession): string {
        const user = session.User?.title || 'Unknown User';
        const title = session.grandparentTitle
            ? `${session.grandparentTitle} - ${session.title}`
            : session.title;
        const player = session.Player?.title || session.Player?.device || 'Unknown Device';
        const platform = session.Player?.platform || 'Unknown';
        const state = session.Player?.state || 'unknown';
        const stateClass = state === 'playing' ? 'state-playing' : 'state-paused';
        const isTranscoding = session.TranscodeSession ? true : false;
        const transcodeLabel = isTranscoding ? 'Transcoding' : 'Direct Play';
        const transcodeClass = isTranscoding ? 'transcode-active' : 'transcode-direct';

        return `
            <div class="plex-session-card">
                <div class="plex-session-header">
                    <span class="plex-session-user">${this.escapeHtml(user)}</span>
                    <span class="plex-session-state ${stateClass}">${state}</span>
                </div>
                <div class="plex-session-content">
                    <p class="plex-session-title">${this.escapeHtml(title)}</p>
                    <p class="plex-session-device">${this.escapeHtml(player)} (${this.escapeHtml(platform)})</p>
                </div>
                <div class="plex-session-footer">
                    <span class="plex-session-transcode ${transcodeClass}">${transcodeLabel}</span>
                    ${session.TranscodeSession?.progress ? `<span class="plex-session-progress">${session.TranscodeSession.progress.toFixed(0)}%</span>` : ''}
                </div>
            </div>
        `;
    }

    /**
     * Handle WebSocket events for real-time updates
     */
    private registerWebSocketListener(): void {
        if (this.wsListenerRegistered) return;

        // Store handler reference for cleanup
        this.wsEventHandler = ((event: CustomEvent<PlexRealtimeEvent>) => {
            this.handleRealtimeEvent(event.detail);
        }) as EventListener;

        // Listen for custom WebSocket events
        window.addEventListener('plex_realtime_playback', this.wsEventHandler);

        this.wsListenerRegistered = true;
    }

    /**
     * Handle a real-time playback event
     */
    private handleRealtimeEvent(event: PlexRealtimeEvent): void {
        this.status.websocketConnected = true;
        this.status.lastUpdate = new Date();

        // If it's a new session, refresh the full list
        if (event.isNewSession) {
            this.fetchStatus();
            return;
        }

        // Update existing session state
        const session = this.sessions.find(s => s.sessionKey === event.sessionKey);
        if (session && session.Player) {
            session.Player.state = event.state;
        }

        this.updateUI();
    }

    /**
     * Start auto-refresh interval
     */
    private startAutoRefresh(): void {
        // Refresh every 30 seconds
        this.refreshInterval = setInterval(() => {
            this.fetchStatus();
        }, 30000);
    }

    /**
     * Format time for display
     */
    private formatTime(date: Date): string {
        return date.toLocaleTimeString();
    }

    /**
     * Escape HTML to prevent XSS
     */
    private escapeHtml(text: string): string {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }
}
