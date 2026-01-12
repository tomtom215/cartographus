// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * TimelineController - Handles timeline UI interactions and state updates
 *
 * Manages:
 * - Timeline playback controls (play/pause, reset)
 * - Timeline seeking via slider
 * - Playback speed adjustment
 * - Temporal interval selection (hourly/daily/weekly)
 * - UI updates for timeline state and playback count
 * - Time display formatting
 */

import type { TimelineManager } from '../lib/timeline';

export class TimelineController {
    // AbortController for clean event listener removal
    private abortController: AbortController | null = null;

    constructor(
        private timelineManager: TimelineManager | null
    ) {
        this.setupTimelineListeners();
    }

    /**
     * Set timeline manager reference (for lazy initialization)
     */
    setTimelineManager(timelineManager: TimelineManager | null): void {
        this.timelineManager = timelineManager;
    }

    /**
     * Setup all timeline control event listeners with AbortController for clean removal
     */
    private setupTimelineListeners(): void {
        // Create AbortController for cleanup
        this.abortController = new AbortController();
        const signal = this.abortController.signal;

        const btnTimelinePlayPause = document.getElementById('timeline-play-pause') as HTMLButtonElement;
        if (btnTimelinePlayPause) {
            btnTimelinePlayPause.addEventListener('click', () => this.togglePlayback(), { signal });
        }

        const btnTimelineReset = document.getElementById('timeline-reset') as HTMLButtonElement;
        if (btnTimelineReset) {
            btnTimelineReset.addEventListener('click', () => this.reset(), { signal });
        }

        const timelineSlider = document.getElementById('timeline-slider') as HTMLInputElement;
        if (timelineSlider) {
            timelineSlider.addEventListener('input', (e) => this.handleSeek(e), { signal });
        }

        const timelineSpeed = document.getElementById('timeline-speed') as HTMLSelectElement;
        if (timelineSpeed) {
            timelineSpeed.addEventListener('change', (e) => this.handleSpeedChange(e), { signal });
        }

        const timelineInterval = document.getElementById('timeline-interval') as HTMLSelectElement;
        if (timelineInterval) {
            timelineInterval.addEventListener('change', (e) => this.handleIntervalChange(e), { signal });
        }
    }

    /**
     * Toggle timeline playback (play/pause)
     */
    togglePlayback(): void {
        if (!this.timelineManager) return;

        if (this.timelineManager.isCurrentlyPlaying()) {
            this.timelineManager.pause();
        } else {
            this.timelineManager.play();
        }
    }

    /**
     * Reset timeline to beginning
     */
    reset(): void {
        if (!this.timelineManager) return;
        this.timelineManager.reset();
    }

    /**
     * Handle timeline slider seek event
     */
    private handleSeek(e: Event): void {
        if (!this.timelineManager) return;

        const slider = e.target as HTMLInputElement;
        const progress = parseFloat(slider.value) / 100;
        this.timelineManager.seekToProgress(progress);
    }

    /**
     * Handle playback speed change
     */
    private handleSpeedChange(e: Event): void {
        if (!this.timelineManager) return;

        const select = e.target as HTMLSelectElement;
        const speed = parseInt(select.value, 10) as 1 | 2 | 5 | 10;
        this.timelineManager.setSpeed(speed);
    }

    /**
     * Handle temporal interval change (hourly/daily/weekly)
     */
    private handleIntervalChange(e: Event): void {
        if (!this.timelineManager) return;

        const select = e.target as HTMLSelectElement;
        const interval = select.value as 'hourly' | 'daily' | 'weekly';
        this.timelineManager.setInterval(interval);
    }

    /**
     * Update timeline UI when time/playback changes
     * Called by TimelineManager callbacks
     */
    handleTimelineUpdate(currentTime: Date, _playbacks: unknown[], totalCount: number): void {
        const timeDisplay = document.getElementById('timeline-time');
        const countDisplay = document.getElementById('timeline-count');
        const slider = document.getElementById('timeline-slider') as HTMLInputElement;

        if (timeDisplay && this.timelineManager) {
            const formattedTime = this.formatTime(currentTime);
            timeDisplay.textContent = formattedTime;
        }

        if (countDisplay) {
            countDisplay.textContent = `${totalCount.toLocaleString()} playbacks`;
        }

        if (slider && this.timelineManager) {
            const progress = this.timelineManager.getProgress();
            slider.value = (progress * 100).toString();
        }
    }

    /**
     * Update play/pause button state
     * Called by TimelineManager callbacks
     */
    handlePlayStateChange(isPlaying: boolean): void {
        const playIcon = document.getElementById('timeline-play-icon');
        const pauseIcon = document.getElementById('timeline-pause-icon');
        const playPauseBtn = document.getElementById('timeline-play-pause');

        if (playIcon && pauseIcon) {
            playIcon.style.display = isPlaying ? 'none' : 'block';
            pauseIcon.style.display = isPlaying ? 'block' : 'none';
        }

        if (playPauseBtn) {
            playPauseBtn.setAttribute('aria-label', isPlaying ? 'Pause timeline' : 'Play timeline');
        }
    }

    /**
     * Format date for timeline display
     */
    private formatTime(date: Date): string {
        const year = date.getFullYear();
        const month = String(date.getMonth() + 1).padStart(2, '0');
        const day = String(date.getDate()).padStart(2, '0');
        const hours = String(date.getHours()).padStart(2, '0');
        const minutes = String(date.getMinutes()).padStart(2, '0');

        return `${year}-${month}-${day} ${hours}:${minutes}`;
    }

    /**
     * Clean up event listeners
     */
    destroy(): void {
        if (this.abortController) {
            this.abortController.abort();
            this.abortController = null;
        }
        this.timelineManager = null;
    }
}
