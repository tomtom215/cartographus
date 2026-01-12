// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Stream Data Manager
 *
 * Displays detailed stream information for active or historical sessions:
 * - Video/Audio codec details
 * - Resolution and bitrate information
 * - Transcode decisions
 * - Container format
 */

import { API } from '../lib/api';
import type { ToastManager } from '../lib/toast';
import type { TautulliStreamDataInfo } from '../lib/types';
import { createLogger } from '../lib/logger';

const logger = createLogger('StreamDataManager');

interface StreamDataManagerOptions {
    api: API;
    containerId: string;
    toastManager?: ToastManager;
}

export class StreamDataManager {
    private api: API;
    private containerId: string;
    private toastManager: ToastManager | null = null;
    private initialized = false;

    // State
    private sessionKey: string = '';
    private rowId: number | null = null;
    private streamData: TautulliStreamDataInfo | null = null;
    private loading = false;
    private error: string | null = null;

    constructor(options: StreamDataManagerOptions) {
        this.api = options.api;
        this.containerId = options.containerId;
        if (options.toastManager) {
            this.toastManager = options.toastManager;
        }
    }

    /**
     * Set toast manager
     */
    setToastManager(toast: ToastManager): void {
        this.toastManager = toast;
    }

    /**
     * Initialize the manager
     */
    async init(): Promise<void> {
        if (this.initialized) return;
        this.render();
        this.initialized = true;
    }

    /**
     * Load stream data for a session or history row
     */
    async loadStreamData(sessionKey?: string, rowId?: number): Promise<void> {
        this.sessionKey = sessionKey || '';
        this.rowId = rowId ?? null;
        this.loading = true;
        this.error = null;
        this.streamData = null;
        this.render();

        try {
            this.streamData = await this.api.getStreamData(sessionKey, rowId);
            this.loading = false;
            this.render();
        } catch (err) {
            logger.error('Failed to load stream data:', err);
            this.loading = false;
            this.error = 'Failed to load stream data';
            this.render();
        }
    }

    /**
     * Clear current data
     */
    clear(): void {
        this.sessionKey = '';
        this.rowId = null;
        this.streamData = null;
        this.error = null;
        this.render();
    }

    /**
     * Render the component
     */
    private render(): void {
        const container = document.getElementById(this.containerId);
        if (!container) return;

        container.innerHTML = `
            <div class="stream-data-manager">
                <div class="stream-data-header">
                    <h3>Stream Details</h3>
                    <p class="hint">View detailed stream information for active or historical sessions</p>
                </div>

                <div class="stream-data-form">
                    <div class="form-row">
                        <div class="form-group">
                            <label for="session-key-input">Session Key</label>
                            <input type="text"
                                   id="session-key-input"
                                   class="form-input"
                                   placeholder="Enter session key"
                                   value="${this.escapeHtml(this.sessionKey)}">
                        </div>
                        <div class="form-group">
                            <label for="row-id-input">History Row ID</label>
                            <input type="number"
                                   id="row-id-input"
                                   class="form-input"
                                   placeholder="Enter row ID"
                                   value="${this.rowId !== null ? this.rowId : ''}">
                        </div>
                        <button class="btn-primary" id="load-stream-btn" ${this.loading ? 'disabled' : ''}>
                            ${this.loading ? 'Loading...' : 'Load Stream Data'}
                        </button>
                    </div>
                    <p class="form-hint">Enter either a session key (for active streams) or a history row ID (for past sessions)</p>
                </div>

                ${this.renderContent()}
            </div>
        `;

        this.attachEventListeners();
    }

    /**
     * Render content section
     */
    private renderContent(): string {
        if (this.loading) {
            return `
                <div class="stream-data-content loading">
                    <div class="spinner"></div>
                    <p>Loading stream data...</p>
                </div>
            `;
        }

        if (this.error) {
            return `
                <div class="stream-data-content error">
                    <p class="error-message">${this.escapeHtml(this.error)}</p>
                    <button class="btn-retry" id="retry-btn">Retry</button>
                </div>
            `;
        }

        if (!this.streamData) {
            return `
                <div class="stream-data-content empty">
                    <p>Enter a session key or row ID to view stream details</p>
                </div>
            `;
        }

        return this.renderStreamData();
    }

    /**
     * Render stream data details
     */
    private renderStreamData(): string {
        const data = this.streamData!;

        return `
            <div class="stream-data-content" data-testid="stream-data-results">
                <!-- Transcode Decisions -->
                <div class="stream-section">
                    <h4>Transcode Decisions</h4>
                    <div class="decision-grid">
                        ${this.renderDecisionBadge('Overall', data.transcode_decision)}
                        ${this.renderDecisionBadge('Video', data.video_decision)}
                        ${this.renderDecisionBadge('Audio', data.audio_decision)}
                        ${data.subtitle_decision ? this.renderDecisionBadge('Subtitle', data.subtitle_decision) : ''}
                    </div>
                </div>

                <!-- Source Media -->
                <div class="stream-section">
                    <h4>Source Media</h4>
                    <div class="info-grid">
                        <div class="info-item">
                            <span class="label">Container</span>
                            <span class="value badge">${this.escapeHtml(data.container || '-')}</span>
                        </div>
                        <div class="info-item">
                            <span class="label">Bitrate</span>
                            <span class="value">${this.formatBitrate(data.bitrate)}</span>
                        </div>
                    </div>

                    <div class="media-subsection">
                        <h5>Video</h5>
                        <div class="info-grid">
                            <div class="info-item">
                                <span class="label">Codec</span>
                                <span class="value code">${this.escapeHtml(data.video_codec || '-')}</span>
                            </div>
                            <div class="info-item">
                                <span class="label">Resolution</span>
                                <span class="value">${data.video_width}x${data.video_height} (${this.escapeHtml(data.video_resolution || '-')})</span>
                            </div>
                            <div class="info-item">
                                <span class="label">Framerate</span>
                                <span class="value">${this.escapeHtml(data.video_framerate || '-')}</span>
                            </div>
                            <div class="info-item">
                                <span class="label">Bitrate</span>
                                <span class="value">${this.formatBitrate(data.video_bitrate)}</span>
                            </div>
                        </div>
                    </div>

                    <div class="media-subsection">
                        <h5>Audio</h5>
                        <div class="info-grid">
                            <div class="info-item">
                                <span class="label">Codec</span>
                                <span class="value code">${this.escapeHtml(data.audio_codec || '-')}</span>
                            </div>
                            <div class="info-item">
                                <span class="label">Channels</span>
                                <span class="value">${data.audio_channels}${data.audio_channel_layout ? ` (${this.escapeHtml(data.audio_channel_layout)})` : ''}</span>
                            </div>
                            <div class="info-item">
                                <span class="label">Bitrate</span>
                                <span class="value">${this.formatBitrate(data.audio_bitrate)}</span>
                            </div>
                            <div class="info-item">
                                <span class="label">Sample Rate</span>
                                <span class="value">${this.formatSampleRate(data.audio_sample_rate)}</span>
                            </div>
                        </div>
                    </div>
                </div>

                <!-- Stream Output -->
                ${this.renderStreamOutput(data)}

                <!-- Status -->
                <div class="stream-section">
                    <h4>Status</h4>
                    <div class="status-grid">
                        <div class="status-item ${data.optimized ? 'active' : ''}">
                            <span class="status-icon">${data.optimized ? '&#10003;' : '&#10007;'}</span>
                            <span class="status-label">Optimized Version</span>
                        </div>
                        <div class="status-item ${data.throttled ? 'warning' : ''}">
                            <span class="status-icon">${data.throttled ? '&#9888;' : '&#10003;'}</span>
                            <span class="status-label">${data.throttled ? 'Throttled' : 'Not Throttled'}</span>
                        </div>
                    </div>
                </div>
            </div>
        `;
    }

    /**
     * Render stream output section
     */
    private renderStreamOutput(data: TautulliStreamDataInfo): string {
        // Check if there's any stream output data
        const hasStreamData = data.stream_container || data.stream_video_codec || data.stream_audio_codec;

        if (!hasStreamData) {
            return '';
        }

        return `
            <div class="stream-section">
                <h4>Stream Output</h4>
                <div class="info-grid">
                    ${data.stream_container ? `
                        <div class="info-item">
                            <span class="label">Container</span>
                            <span class="value badge">${this.escapeHtml(data.stream_container)}</span>
                        </div>
                    ` : ''}
                    ${data.stream_bitrate ? `
                        <div class="info-item">
                            <span class="label">Bitrate</span>
                            <span class="value">${this.formatBitrate(data.stream_bitrate)}</span>
                        </div>
                    ` : ''}
                </div>

                ${data.stream_video_codec ? `
                    <div class="media-subsection">
                        <h5>Video Output</h5>
                        <div class="info-grid">
                            <div class="info-item">
                                <span class="label">Codec</span>
                                <span class="value code">${this.escapeHtml(data.stream_video_codec)}</span>
                            </div>
                            ${data.stream_video_resolution ? `
                                <div class="info-item">
                                    <span class="label">Resolution</span>
                                    <span class="value">${data.stream_video_width}x${data.stream_video_height} (${this.escapeHtml(data.stream_video_resolution)})</span>
                                </div>
                            ` : ''}
                            ${data.stream_video_framerate ? `
                                <div class="info-item">
                                    <span class="label">Framerate</span>
                                    <span class="value">${this.escapeHtml(data.stream_video_framerate)}</span>
                                </div>
                            ` : ''}
                            ${data.stream_video_bitrate ? `
                                <div class="info-item">
                                    <span class="label">Bitrate</span>
                                    <span class="value">${this.formatBitrate(data.stream_video_bitrate)}</span>
                                </div>
                            ` : ''}
                        </div>
                    </div>
                ` : ''}

                ${data.stream_audio_codec ? `
                    <div class="media-subsection">
                        <h5>Audio Output</h5>
                        <div class="info-grid">
                            <div class="info-item">
                                <span class="label">Codec</span>
                                <span class="value code">${this.escapeHtml(data.stream_audio_codec)}</span>
                            </div>
                            ${data.stream_audio_channels ? `
                                <div class="info-item">
                                    <span class="label">Channels</span>
                                    <span class="value">${data.stream_audio_channels}</span>
                                </div>
                            ` : ''}
                            ${data.stream_audio_bitrate ? `
                                <div class="info-item">
                                    <span class="label">Bitrate</span>
                                    <span class="value">${this.formatBitrate(data.stream_audio_bitrate)}</span>
                                </div>
                            ` : ''}
                            ${data.stream_audio_sample_rate ? `
                                <div class="info-item">
                                    <span class="label">Sample Rate</span>
                                    <span class="value">${this.formatSampleRate(data.stream_audio_sample_rate)}</span>
                                </div>
                            ` : ''}
                        </div>
                    </div>
                ` : ''}
            </div>
        `;
    }

    /**
     * Render decision badge
     */
    private renderDecisionBadge(label: string, decision: string): string {
        const decisionClass = this.getDecisionClass(decision);
        return `
            <div class="decision-item">
                <span class="decision-label">${label}</span>
                <span class="decision-badge ${decisionClass}">${this.escapeHtml(decision || '-')}</span>
            </div>
        `;
    }

    /**
     * Get CSS class for decision type
     */
    private getDecisionClass(decision: string): string {
        const normalized = (decision || '').toLowerCase();
        if (normalized === 'direct play') return 'direct-play';
        if (normalized === 'copy') return 'copy';
        if (normalized === 'transcode') return 'transcode';
        return '';
    }

    /**
     * Format bitrate to human-readable string
     */
    private formatBitrate(bitrate: number): string {
        if (!bitrate) return '-';
        if (bitrate >= 1000) {
            return `${(bitrate / 1000).toFixed(1)} Mbps`;
        }
        return `${bitrate} kbps`;
    }

    /**
     * Format sample rate to human-readable string
     */
    private formatSampleRate(sampleRate: number): string {
        if (!sampleRate) return '-';
        if (sampleRate >= 1000) {
            return `${(sampleRate / 1000).toFixed(1)} kHz`;
        }
        return `${sampleRate} Hz`;
    }

    /**
     * Attach event listeners
     */
    private attachEventListeners(): void {
        const container = document.getElementById(this.containerId);
        if (!container) return;

        // Load button
        const loadBtn = container.querySelector('#load-stream-btn');
        if (loadBtn) {
            loadBtn.addEventListener('click', () => this.handleLoad());
        }

        // Enter key on inputs
        const sessionInput = container.querySelector('#session-key-input') as HTMLInputElement;
        const rowIdInput = container.querySelector('#row-id-input') as HTMLInputElement;

        if (sessionInput) {
            sessionInput.addEventListener('keypress', (e) => {
                if (e.key === 'Enter') {
                    this.handleLoad();
                }
            });
        }

        if (rowIdInput) {
            rowIdInput.addEventListener('keypress', (e) => {
                if (e.key === 'Enter') {
                    this.handleLoad();
                }
            });
        }

        // Retry button
        const retryBtn = container.querySelector('#retry-btn');
        if (retryBtn) {
            retryBtn.addEventListener('click', () => {
                this.loadStreamData(this.sessionKey || undefined, this.rowId ?? undefined);
            });
        }
    }

    /**
     * Handle load action
     */
    private handleLoad(): void {
        const container = document.getElementById(this.containerId);
        if (!container) return;

        const sessionInput = container.querySelector('#session-key-input') as HTMLInputElement;
        const rowIdInput = container.querySelector('#row-id-input') as HTMLInputElement;

        const sessionKey = sessionInput?.value?.trim() || undefined;
        const rowIdValue = rowIdInput?.value?.trim();
        const rowId = rowIdValue ? parseInt(rowIdValue, 10) : undefined;

        if (!sessionKey && rowId === undefined) {
            this.toastManager?.warning('Please enter a session key or row ID', 'Input Required', 3000);
            return;
        }

        this.loadStreamData(sessionKey, rowId);
    }

    /**
     * Escape HTML to prevent XSS
     */
    private escapeHtml(unsafe: string): string {
        return unsafe
            .replace(/&/g, '&amp;')
            .replace(/</g, '&lt;')
            .replace(/>/g, '&gt;')
            .replace(/"/g, '&quot;')
            .replace(/'/g, '&#039;');
    }

    /**
     * Clean up resources
     */
    destroy(): void {
        this.initialized = false;
        this.streamData = null;
    }
}
