// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * PlexMonitoringManager - Handles all Plex-related real-time monitoring
 *
 * Responsibilities:
 * - Plex real-time WebSocket playback notifications (v1.39)
 * - Transcode session monitoring with quality/codec/HW acceleration tracking (v1.40)
 * - Buffer health monitoring with predictive buffering alerts (v1.41)
 * - UI panel updates for transcode and buffer health
 *
 * Extracted from index.ts to reduce complexity (refactoring v1.44)
 */

import type {
    PlexRealtimePlaybackData,
    PlexTranscodeSessionsData,
    PlexSession,
    BufferHealthUpdateData,
    BufferHealth
} from './websocket';
import type { ToastManager } from './toast';
import type { StatsManager } from './stats';
import { createLogger } from './logger';

const logger = createLogger('PlexMonitoring');

export class PlexMonitoringManager {
    private toastManager: ToastManager | null;
    private statsManager: StatsManager | null;

    constructor(toastManager: ToastManager | null, statsManager: StatsManager | null) {
        this.toastManager = toastManager;
        this.statsManager = statsManager;
    }

    /**
     * Handle Plex real-time playback updates (Phase 1.1: v1.39)
     * Provides instant notifications for buffering, new sessions, and state changes
     */
    handlePlexRealtimePlayback(data: PlexRealtimePlaybackData): void {
        if (!this.toastManager) return;

        // Handle buffering state - instant notification for performance issues
        if (data.is_buffering && data.state === 'buffering') {
            this.toastManager.warning(
                `Session ${data.session_key} is buffering (may indicate network or transcoding issues)`,
                'Buffering Detected',
                5000
            );
            logger.debug('[Plex Real-Time] Buffering detected:', data);
        }

        // Handle new sessions that Tautulli missed - instant map update
        if (data.is_new_session) {
            this.toastManager.info(
                `New playback session started (session ${data.session_key})`,
                'Real-Time Update',
                5000
            );
            logger.debug('[Plex Real-Time] New session detected:', data);

            // Silently refresh stats to show the new session
            this.statsManager?.loadStats();
        }

        // Handle state changes for existing sessions (playing → paused, etc.)
        if (data.state && !data.is_new_session && !data.is_buffering) {
            const stateMessages: { [key: string]: string } = {
                'playing': 'resumed playback',
                'paused': 'paused playback',
                'stopped': 'stopped playback'
            };

            const action = stateMessages[data.state] || `changed state to ${data.state}`;
            logger.debug(`[Plex Real-Time] Session ${data.session_key} ${action}`, data);

            // Only show toast for stopped state (less noisy)
            if (data.state === 'stopped') {
                this.toastManager.info(
                    `Session ${data.session_key} stopped`,
                    'Playback Ended',
                    3000
                );

                // Refresh stats when session ends
                this.statsManager?.loadStats();
            }
        }
    }

    /**
     * Handle Plex transcode sessions updates (Phase 1.2: v1.40)
     * Shows real-time transcode progress, quality transitions, and hardware acceleration status
     */
    handlePlexTranscodeSessions(data: PlexTranscodeSessionsData): void {
        // Note: toastManager may be null but panel updates should still work
        // Toast notifications use optional chaining (this.toastManager?.warning())

        const { sessions, count } = data;

        // Log to console for debugging
        logger.debug(`[Plex Transcode] ${count.transcoding}/${count.total} active sessions`, sessions);

        // Update UI panel
        this.updateTranscodePanel(sessions, count);

        // Process each session for notifications
        sessions.forEach(session => {
            if (!session.transcodeSession) return;

            const ts = session.transcodeSession;
            const isTranscoding = ts.videoDecision === 'transcode';

            if (!isTranscoding) return;

            // Get hardware acceleration type (user-friendly name)
            const hwType = ts.transcodeHwDecoding || ts.transcodeHwEncoding || 'software';
            let hwName = 'Software';
            if (hwType.includes('qsv')) hwName = 'Quick Sync (QSV)';
            else if (hwType.includes('nvenc')) hwName = 'NVENC (NVIDIA)';
            else if (hwType.includes('vaapi')) hwName = 'VAAPI';
            else if (hwType.includes('videotoolbox')) hwName = 'VideoToolbox (Apple)';
            else if (hwType.includes('mediacodec')) hwName = 'MediaCodec (Android)';
            else if (hwType.includes('mf')) hwName = 'Media Foundation';

            // Quality transition (4K→1080p, HEVC→H.264)
            const sourceRes = session.media?.[0]?.videoResolution || 'Unknown';
            const targetHeight = ts.height;
            let targetRes = 'SD';
            if (targetHeight >= 2160) targetRes = '4K';
            else if (targetHeight >= 1080) targetRes = '1080p';
            else if (targetHeight >= 720) targetRes = '720p';

            const codecTransition = ts.sourceVideoCodec && ts.videoCodec
                ? `${ts.sourceVideoCodec.toUpperCase()}→${ts.videoCodec.toUpperCase()}`
                : '';

            // Transcode speed (1.5 = 1.5x realtime)
            const speed = ts.speed ? `${ts.speed.toFixed(1)}x` : 'N/A';

            // Check for throttling (system load too high)
            if (ts.throttled) {
                this.toastManager?.warning(
                    `Transcode throttled: ${session.title} (high system load)`,
                    'Transcode Warning ⚠️',
                    10000
                );
                logger.warn(`[Plex Transcode] THROTTLED: ${session.title}`, session);
            }

            // Log detailed transcode info
            const user = session.user?.title || 'Unknown User';
            const player = session.player?.title || 'Unknown Player';
            logger.debug(
                `[Plex Transcode] ${user} on ${player}: ${session.title}\n` +
                `  Quality: ${sourceRes}→${targetRes} | Codec: ${codecTransition}\n` +
                `  Hardware: ${hwName} | Speed: ${speed} | Progress: ${ts.progress.toFixed(1)}%`
            );
        });

        // Show summary toast for new transcode sessions (only if count > 0)
        if (count.transcoding > 0) {
            // You could track previous state to detect NEW transcodes
            // For now, just log the count (no toast to reduce noise)
            logger.debug(`[Plex Transcode] Summary: ${count.transcoding} active transcode(s)`);
        }
    }

    /**
     * Update the transcode panel UI with current session data
     */
    private updateTranscodePanel(sessions: PlexSession[], count: { total: number; transcoding: number }): void {
        const panel = document.getElementById('transcode-panel');
        const badge = document.getElementById('transcode-count-badge');
        const sessionsList = document.getElementById('transcode-sessions-list');

        if (!panel || !badge || !sessionsList) return;

        // Update badge count
        badge.textContent = count.transcoding.toString();

        // Update badge color based on count
        badge.className = 'badge';
        if (count.transcoding === 0) {
            badge.classList.add('badge-info');
        } else if (count.transcoding <= 2) {
            badge.classList.add('badge-success');
        } else {
            badge.classList.add('badge-warning');
        }

        // Show/hide panel based on transcode count
        if (count.transcoding === 0) {
            panel.classList.add('hidden');
            return;
        }

        panel.classList.remove('hidden');

        // Filter and render only transcoding sessions
        const transcodingSessions = sessions.filter(s =>
            s.transcodeSession && s.transcodeSession.videoDecision === 'transcode'
        );

        if (transcodingSessions.length === 0) {
            sessionsList.innerHTML = '<p class="transcode-empty-message">No active transcodes</p>';
            return;
        }

        // Helper function to escape HTML and prevent XSS attacks
        // Escapes <, >, &, ", ' to their HTML entity equivalents
        const escapeHtml = (unsafe: string): string => {
            return unsafe
                .replace(/&/g, '&amp;')
                .replace(/</g, '&lt;')
                .replace(/>/g, '&gt;')
                .replace(/"/g, '&quot;')
                .replace(/'/g, '&#039;');
        };

        // Render session cards
        sessionsList.innerHTML = transcodingSessions.map(session => {
            const ts = session.transcodeSession!;
            const user = escapeHtml(session.user?.title || 'Unknown User');
            const player = escapeHtml(session.player?.title || 'Unknown Player');

            // Hardware acceleration type
            const hwType = ts.transcodeHwDecoding || ts.transcodeHwEncoding || 'software';
            let hwName = 'Software';
            let hwClass = 'software';
            if (hwType.includes('qsv')) { hwName = '⚡ Quick Sync'; hwClass = 'hardware'; }
            else if (hwType.includes('nvenc')) { hwName = '🎮 NVENC'; hwClass = 'hardware'; }
            else if (hwType.includes('vaapi')) { hwName = '🔧 VAAPI'; hwClass = 'hardware'; }
            else if (hwType.includes('videotoolbox')) { hwName = '🍎 VideoToolbox'; hwClass = 'hardware'; }
            else if (hwType.includes('mediacodec')) { hwName = '🤖 MediaCodec'; hwClass = 'hardware'; }
            else if (hwType.includes('mf')) { hwName = '🪟 Media Foundation'; hwClass = 'hardware'; }

            // Quality transition
            const sourceRes = escapeHtml(session.media?.[0]?.videoResolution || 'Unknown');
            const targetHeight = ts.height;
            let targetRes = 'SD';
            if (targetHeight >= 2160) targetRes = '4K';
            else if (targetHeight >= 1080) targetRes = '1080p';
            else if (targetHeight >= 720) targetRes = '720p';

            const codecTransition = ts.sourceVideoCodec && ts.videoCodec
                ? escapeHtml(`${ts.sourceVideoCodec.toUpperCase()}→${ts.videoCodec.toUpperCase()}`)
                : 'Unknown';

            const throttledClass = ts.throttled ? ' throttled' : '';
            const throttledWarning = ts.throttled
                ? '<div class="transcode-throttle-warning">⚠️ THROTTLED - High system load</div>'
                : '';

            return `
                <div class="transcode-session-card${throttledClass}">
                    <div class="transcode-session-header">
                        <div>
                            <h4 class="transcode-session-title">${escapeHtml(session.title || 'Unknown Title')}</h4>
                            <p class="transcode-session-user">${user} • ${player}</p>
                        </div>
                        <span class="transcode-hw-badge ${hwClass}">${hwName}</span>
                    </div>
                    <div class="transcode-session-details">
                        <div class="transcode-detail-row">
                            <span class="transcode-detail-label">Quality:</span>
                            <span class="transcode-detail-value">${sourceRes}→${targetRes}</span>
                        </div>
                        <div class="transcode-detail-row">
                            <span class="transcode-detail-label">Codec:</span>
                            <span class="transcode-detail-value">${codecTransition}</span>
                        </div>
                        <div class="transcode-detail-row">
                            <span class="transcode-detail-label">Speed:</span>
                            <span class="transcode-detail-value">${ts.speed.toFixed(1)}x</span>
                        </div>
                    </div>
                    <div class="transcode-progress-bar">
                        <div class="transcode-progress-fill" style="width: ${ts.progress.toFixed(1)}%"></div>
                    </div>
                    ${throttledWarning}
                </div>
            `;
        }).join('');
    }

    /**
     * Handle buffer health updates from Plex WebSocket (v1.41)
     */
    handleBufferHealthUpdate(data: BufferHealthUpdateData): void {
        // Note: toastManager may be null but panel updates should still work
        // Toast notifications use optional chaining (this.toastManager?.error/warning())

        const { sessions, critical_count, risky_count } = data;

        // Log to console for debugging
        logger.debug(`[Buffer Health] ${critical_count} critical, ${risky_count} risky sessions`, sessions);

        // Update UI panel
        this.updateBufferHealthPanel(sessions, critical_count, risky_count);

        // Process each session for critical/risky toast notifications
        sessions.forEach(session => {
            // Only show toast for critical or risky sessions
            if (session.healthStatus === 'critical') {
                const predictedSeconds = this.calculatePredictedBuffering(session);
                const message = predictedSeconds > 0 && predictedSeconds < 30
                    ? `${session.title} buffering in ${Math.floor(predictedSeconds)}s (buffer: ${session.bufferFillPercent.toFixed(0)}%)`
                    : `${session.title} low buffer (${session.bufferFillPercent.toFixed(0)}%)`;

                this.toastManager?.error(
                    message,
                    'Critical Buffer 🔴',
                    10000
                );
                logger.error(`[Buffer Health] CRITICAL: ${session.title}`, session);
            } else if (session.healthStatus === 'risky') {
                this.toastManager?.warning(
                    `${session.title} buffer dropping (${session.bufferFillPercent.toFixed(0)}%)`,
                    'Risky Buffer 🟡',
                    7000
                );
                logger.warn(`[Buffer Health] RISKY: ${session.title}`, session);
            }

            // Log detailed buffer health info
            const user = session.username || 'Unknown User';
            const player = session.player_device || 'Unknown Player';
            logger.debug(
                `[Buffer Health] ${user} on ${player}: ${session.title}\n` +
                `  Buffer: ${session.bufferFillPercent.toFixed(1)}% | Drain Rate: ${session.bufferDrainRate.toFixed(2)}x\n` +
                `  Buffer Seconds: ${session.bufferSeconds.toFixed(1)}s | Health: ${session.healthStatus} | Speed: ${session.transcodeSpeed.toFixed(1)}x`
            );
        });

        // Show summary toast for critical sessions
        if (critical_count > 0) {
            logger.debug(`[Buffer Health] Summary: ${critical_count} critical, ${risky_count} risky session(s)`);
        }
    }

    /**
     * Calculate predicted seconds until buffering
     * Returns positive number = seconds until buffering, 0 = buffering now, negative = buffer growing
     */
    private calculatePredictedBuffering(session: BufferHealth): number {
        if (session.bufferSeconds <= 0) {
            return 0; // Already buffering
        }

        if (session.bufferDrainRate > 1.0) {
            // Buffer draining faster than playback
            const drainExcess = session.bufferDrainRate - 1.0;
            return session.bufferSeconds / drainExcess;
        }

        // Buffer stable or growing
        return -1;
    }

    /**
     * Update the buffer health panel UI with current session data
     */
    private updateBufferHealthPanel(sessions: BufferHealth[], critical_count: number, risky_count: number): void {
        const panel = document.getElementById('buffer-health-panel');
        const criticalBadge = document.getElementById('buffer-critical-badge');
        const riskyBadge = document.getElementById('buffer-risky-badge');
        const sessionsList = document.getElementById('buffer-health-sessions-list');

        if (!panel || !criticalBadge || !riskyBadge || !sessionsList) return;

        // Update badge counts
        criticalBadge.textContent = critical_count.toString();
        riskyBadge.textContent = risky_count.toString();

        // Show/hide panel based on critical+risky count
        const totalAlerts = critical_count + risky_count;
        if (totalAlerts === 0) {
            panel.classList.add('hidden');
            return;
        }

        panel.classList.remove('hidden');

        // Filter to only critical and risky sessions
        const alertSessions = sessions.filter(s =>
            s.healthStatus === 'critical' || s.healthStatus === 'risky'
        );

        if (alertSessions.length === 0) {
            sessionsList.innerHTML = '<p class="buffer-health-empty-message">All sessions healthy</p>';
            return;
        }

        // Render session cards (sorted by risk level descending)
        sessionsList.innerHTML = alertSessions
            .sort((a, b) => b.riskLevel - a.riskLevel)
            .map(session => {
                const user = session.username || 'Unknown User';
                const player = session.player_device || 'Unknown Player';

                // Health indicator emoji and class
                let healthEmoji = '🟢';
                let healthClass = 'healthy';
                if (session.healthStatus === 'critical') {
                    healthEmoji = '🔴';
                    healthClass = 'critical';
                } else if (session.healthStatus === 'risky') {
                    healthEmoji = '🟡';
                    healthClass = 'risky';
                }

                // Predicted buffering time
                const predictedSeconds = this.calculatePredictedBuffering(session);
                let predictionText = '';
                if (predictedSeconds > 0 && predictedSeconds < 30) {
                    predictionText = `<div class="buffer-health-prediction">⏱️ Buffering in ${Math.floor(predictedSeconds)}s</div>`;
                } else if (predictedSeconds === 0) {
                    predictionText = '<div class="buffer-health-prediction">⚠️ Currently buffering</div>';
                }

                return `
                    <div class="buffer-health-session-card ${healthClass}">
                        <div class="buffer-health-session-header">
                            <div>
                                <h4 class="buffer-health-session-title">${session.title || 'Unknown Title'}</h4>
                                <p class="buffer-health-session-user">${user} • ${player}</p>
                            </div>
                            <span class="buffer-health-indicator">${healthEmoji} ${session.healthStatus.toUpperCase()}</span>
                        </div>
                        <div class="buffer-health-session-details">
                            <div class="buffer-health-detail-row">
                                <span class="buffer-health-detail-label">Buffer Fill:</span>
                                <span class="buffer-health-detail-value">${session.bufferFillPercent.toFixed(1)}%</span>
                            </div>
                            <div class="buffer-health-detail-row">
                                <span class="buffer-health-detail-label">Drain Rate:</span>
                                <span class="buffer-health-detail-value">${session.bufferDrainRate.toFixed(2)}x</span>
                            </div>
                            <div class="buffer-health-detail-row">
                                <span class="buffer-health-detail-label">Buffer Time:</span>
                                <span class="buffer-health-detail-value">${session.bufferSeconds.toFixed(1)}s</span>
                            </div>
                            <div class="buffer-health-detail-row">
                                <span class="buffer-health-detail-label">Speed:</span>
                                <span class="buffer-health-detail-value">${session.transcodeSpeed.toFixed(1)}x</span>
                            </div>
                        </div>
                        <div class="buffer-health-progress-bar">
                            <div class="buffer-health-progress-fill ${healthClass}" style="width: ${session.bufferFillPercent.toFixed(1)}%"></div>
                        </div>
                        ${predictionText}
                    </div>
                `;
            }).join('');
    }

    /**
     * Cleanup resources to prevent memory leaks
     * Clears manager references
     */
    destroy(): void {
        this.toastManager = null;
        this.statsManager = null;
    }
}
