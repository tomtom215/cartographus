// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
import { API, TautulliActivityData, TautulliActivitySession } from './api';
import type { ToastManager } from './toast';
import type { ConfirmationDialogManager } from '../app/ConfirmationDialogManager';
import { createLogger } from './logger';

const logger = createLogger('ActivityDashboard');

export class ActivityDashboardManager {
    private api: API;
    private toastManager: ToastManager | null = null;
    private confirmationManager: ConfirmationDialogManager | null = null;
    private refreshInterval: number | null = null;
    private readonly REFRESH_RATE_MS = 5000; // 5 seconds

    constructor(api: API) {
        this.api = api;
    }

    /**
     * Set toast manager reference for notifications
     */
    setToastManager(toast: ToastManager): void {
        this.toastManager = toast;
    }

    /**
     * Set confirmation dialog manager reference
     */
    setConfirmationManager(manager: ConfirmationDialogManager): void {
        this.confirmationManager = manager;
    }

    async init(): Promise<void> {
        await this.loadActivity();
        this.startAutoRefresh();
    }

    destroy(): void {
        this.stopAutoRefresh();
    }

    private async loadActivity(): Promise<void> {
        try {
            const activityData = await this.api.getTautulliActivity();
            this.renderActivity(activityData);
        } catch (error) {
            logger.error('Failed to load activity:', error);
            this.showError();
        }
    }

    private renderActivity(data: TautulliActivityData): void {
        // Update overview stats with null checks
        const streamCount = document.getElementById('activity-stream-count');
        if (streamCount) {
            streamCount.textContent = data.stream_count.toString();
        }

        const directPlay = document.getElementById('activity-direct-play');
        if (directPlay) {
            directPlay.textContent = data.stream_count_direct_play.toString();
        }

        const transcode = document.getElementById('activity-transcode');
        if (transcode) {
            transcode.textContent = data.stream_count_transcode.toString();
        }

        const bandwidth = document.getElementById('activity-bandwidth');
        if (bandwidth) {
            bandwidth.textContent = this.formatBandwidth(data.total_bandwidth);
        }

        // Render sessions
        const sessionsContainer = document.getElementById('activity-sessions');
        const emptyState = document.getElementById('activity-empty-state');

        if (!sessionsContainer) {
            logger.error('Activity sessions container not found');
            return;
        }

        if (data.sessions.length === 0) {
            if (emptyState) {
                emptyState.style.display = 'block';
            }
            // Clear any existing sessions
            Array.from(sessionsContainer.children).forEach(child => {
                if (child.id !== 'activity-empty-state') {
                    child.remove();
                }
            });
        } else {
            if (emptyState) {
                emptyState.style.display = 'none';
            }

            // Clear existing sessions except empty state
            Array.from(sessionsContainer.children).forEach(child => {
                if (child.id !== 'activity-empty-state') {
                    child.remove();
                }
            });

            // Render each session
            data.sessions.forEach(session => {
                const sessionCard = this.createSessionCard(session);
                sessionsContainer.appendChild(sessionCard);
            });
        }
    }

    private createSessionCard(session: TautulliActivitySession): HTMLElement {
        const card = document.createElement('div');
        card.className = 'activity-session-card';
        card.setAttribute('data-session-key', session.session_key);

        const title = session.grandparent_title
            ? `${session.grandparent_title} - ${session.title}`
            : session.title;

        const transcodeClass = this.getTranscodeClass(session.transcode_decision);
        const stateClass = session.state === 'playing' ? 'state-playing' : 'state-paused';

        card.innerHTML = `
            <div class="session-header">
                <div class="session-user">
                    <strong>${this.escapeHtml(session.friendly_name || session.username)}</strong>
                    <span class="session-state ${stateClass}">${session.state}</span>
                </div>
                <div class="session-header-actions">
                    <span class="session-transcode ${transcodeClass}">${session.transcode_decision}</span>
                    <button class="btn-kill-session"
                            data-session-key="${session.session_key}"
                            data-username="${this.escapeHtml(session.friendly_name || session.username)}"
                            data-title="${this.escapeHtml(title)}"
                            title="Terminate this stream"
                            aria-label="Terminate stream for ${this.escapeHtml(session.friendly_name || session.username)}">
                        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <circle cx="12" cy="12" r="10"/>
                            <line x1="15" y1="9" x2="9" y2="15"/>
                            <line x1="9" y1="9" x2="15" y2="15"/>
                        </svg>
                    </button>
                </div>
            </div>
            <div class="session-content">
                <div class="session-title">${this.escapeHtml(title)}</div>
                <div class="session-details">
                    <span>Device: ${this.escapeHtml(session.platform)}</span>
                    <span>Player: ${this.escapeHtml(session.player)}</span>
                    <span>Res: ${session.stream_video_resolution}</span>
                    <span>Video: ${session.stream_video_codec}</span>
                    <span>Audio: ${session.stream_audio_codec}</span>
                </div>
            </div>
            <div class="session-progress">
                <div class="progress-bar">
                    <div class="progress-fill" style="width: ${session.progress_percent}%"></div>
                </div>
                <div class="progress-info">
                    <span>${session.progress_percent}% complete</span>
                    <span>${this.formatBandwidth(session.bandwidth)}</span>
                </div>
            </div>
        `;

        // Attach kill button event listener
        const killButton = card.querySelector('.btn-kill-session');
        if (killButton) {
            killButton.addEventListener('click', (e) => {
                e.preventDefault();
                e.stopPropagation();
                const sessionKey = (e.currentTarget as HTMLElement).getAttribute('data-session-key');
                const username = (e.currentTarget as HTMLElement).getAttribute('data-username');
                const sessionTitle = (e.currentTarget as HTMLElement).getAttribute('data-title');
                if (sessionKey) {
                    this.confirmKillSession(sessionKey, username || 'Unknown', sessionTitle || 'Unknown');
                }
            });
        }

        return card;
    }

    /**
     * Confirm and kill a session
     */
    private async confirmKillSession(sessionKey: string, username: string, title: string): Promise<void> {
        const confirmMessage = `Are you sure you want to terminate "${title}" for ${username}? This will immediately stop their playback.`;

        if (this.confirmationManager) {
            const confirmed = await this.confirmationManager.show({
                title: 'Terminate Stream',
                message: confirmMessage,
                confirmText: 'Terminate',
                cancelText: 'Cancel',
                confirmButtonClass: 'btn-danger'
            });
            if (confirmed) {
                await this.killSession(sessionKey, username);
            }
        } else {
            // Fallback to native confirm
            if (confirm(confirmMessage)) {
                await this.killSession(sessionKey, username);
            }
        }
    }

    /**
     * Kill a session
     */
    private async killSession(sessionKey: string, username: string): Promise<void> {
        try {
            const message = 'Your stream has been terminated by the server administrator.';
            await this.api.terminateSession(sessionKey, message);

            this.toastManager?.success(
                `Stream terminated for ${username}`,
                'Session Terminated',
                5000
            );

            // Refresh the activity list
            await this.loadActivity();
        } catch (error) {
            logger.error('Failed to terminate session:', error);
            this.toastManager?.error(
                `Failed to terminate stream: ${error instanceof Error ? error.message : 'Unknown error'}`,
                'Error',
                7000
            );
        }
    }

    private getTranscodeClass(decision: string): string {
        switch (decision.toLowerCase()) {
            case 'direct play':
                return 'transcode-direct-play';
            case 'direct stream':
                return 'transcode-direct-stream';
            case 'transcode':
                return 'transcode-transcode';
            default:
                return '';
        }
    }

    private formatBandwidth(kbps: number): string {
        if (kbps === 0) return '0 Mbps';
        const mbps = kbps / 1000;
        return mbps >= 1 ? `${mbps.toFixed(1)} Mbps` : `${kbps} Kbps`;
    }

    private escapeHtml(unsafe: string): string {
        return unsafe
            .replace(/&/g, "&amp;")
            .replace(/</g, "&lt;")
            .replace(/>/g, "&gt;")
            .replace(/"/g, "&quot;")
            .replace(/'/g, "&#039;");
    }

    private showError(): void {
        // E2E-FIX: Explicitly reset stats to 0 on error for graceful fallback.
        // This ensures consistent behavior even if a previous successful request
        // had updated the stats. Tests expect either "0" or an error element.
        const streamCount = document.getElementById('activity-stream-count');
        if (streamCount) {
            streamCount.textContent = '0';
        }
        const directPlay = document.getElementById('activity-direct-play');
        if (directPlay) {
            directPlay.textContent = '0';
        }
        const transcode = document.getElementById('activity-transcode');
        if (transcode) {
            transcode.textContent = '0';
        }
        const bandwidth = document.getElementById('activity-bandwidth');
        if (bandwidth) {
            bandwidth.textContent = '0 Mbps';
        }

        // Show error message in sessions container (append, don't replace)
        const sessionsContainer = document.getElementById('activity-sessions');
        if (sessionsContainer) {
            // Hide empty state if visible
            const emptyState = document.getElementById('activity-empty-state');
            if (emptyState) {
                emptyState.style.display = 'none';
            }

            // Check if error message already exists
            const existingError = sessionsContainer.querySelector('.activity-error-message');
            if (!existingError) {
                const errorDiv = document.createElement('div');
                errorDiv.className = 'activity-error-message error-state';
                errorDiv.style.cssText = 'padding: 20px; text-align: center;';
                errorDiv.innerHTML = `
                    <p style="margin-bottom: 15px; color: #ef4444;">Failed to load activity data</p>
                    <button id="activity-retry" style="padding: 8px 16px; background: #ef4444; color: white; border: none; border-radius: 4px; cursor: pointer;">Retry</button>
                `;
                sessionsContainer.appendChild(errorDiv);

                // Attach retry listener
                const retryButton = document.getElementById('activity-retry');
                if (retryButton) {
                    retryButton.addEventListener('click', () => {
                        errorDiv.remove();
                        this.loadActivity();
                    });
                }
            }
        }
    }

    private startAutoRefresh(): void {
        this.refreshInterval = window.setInterval(() => {
            this.loadActivity();
        }, this.REFRESH_RATE_MS);
    }

    private stopAutoRefresh(): void {
        if (this.refreshInterval !== null) {
            clearInterval(this.refreshInterval);
            this.refreshInterval = null;
        }
    }
}
