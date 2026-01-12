// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
import type { API, PlaybackEvent, LocationFilter } from './api';
import { createLogger } from './logger';

const logger = createLogger('Timeline');

type PlaybackSpeed = 1 | 2 | 5 | 10;
type TimeInterval = 'hourly' | 'daily' | 'weekly';

export class TimelineManager {
    private api: API;
    private playbacks: PlaybackEvent[] = [];
    private currentIndex: number = 0;
    private isPlaying: boolean = false;
    private playbackSpeed: PlaybackSpeed = 1;
    private timeInterval: TimeInterval = 'hourly';
    private animationFrameId: number | null = null;
    private lastUpdateTime: number = 0;
    private startTime: Date | null = null;
    private endTime: Date | null = null;
    private currentTime: Date | null = null;
    private onTimeUpdate?: (currentTime: Date, playbacks: PlaybackEvent[], totalCount: number) => void;
    private onPlayStateChange?: (isPlaying: boolean) => void;

    constructor(api: API, onTimeUpdate?: (currentTime: Date, playbacks: PlaybackEvent[], totalCount: number) => void, onPlayStateChange?: (isPlaying: boolean) => void) {
        this.api = api;
        this.onTimeUpdate = onTimeUpdate;
        this.onPlayStateChange = onPlayStateChange;
    }

    async loadData(filter: LocationFilter): Promise<void> {
        try {
            // API has a max limit of 1000, use that instead of 10000
            const playbacks = await this.api.getPlaybacks(filter, 1000, 0);

            this.playbacks = playbacks.sort((a, b) => {
                const dateA = new Date(a.started_at).getTime();
                const dateB = new Date(b.started_at).getTime();
                return dateA - dateB;
            });

            if (this.playbacks.length > 0) {
                this.startTime = new Date(this.playbacks[0].started_at);
                this.endTime = new Date(this.playbacks[this.playbacks.length - 1].started_at);
                this.currentTime = new Date(this.startTime);
                this.currentIndex = 0;
                this.updateCurrentPlaybacks();
            }
        } catch (error) {
            logger.error('Failed to load timeline data:', error);
        }
    }

    play(): void {
        if (this.isPlaying || !this.currentTime || !this.endTime) return;

        this.isPlaying = true;
        this.lastUpdateTime = performance.now();
        this.animate();

        if (this.onPlayStateChange) {
            this.onPlayStateChange(true);
        }
    }

    pause(): void {
        this.isPlaying = false;

        if (this.animationFrameId !== null) {
            cancelAnimationFrame(this.animationFrameId);
            this.animationFrameId = null;
        }

        if (this.onPlayStateChange) {
            this.onPlayStateChange(false);
        }
    }

    reset(): void {
        this.pause();

        if (this.startTime) {
            this.currentTime = new Date(this.startTime);
            this.currentIndex = 0;
            this.updateCurrentPlaybacks();
        }
    }

    setSpeed(speed: PlaybackSpeed): void {
        this.playbackSpeed = speed;
    }

    setInterval(interval: TimeInterval): void {
        this.timeInterval = interval;
    }

    seekToProgress(progress: number): void {
        if (!this.startTime || !this.endTime) return;

        const clampedProgress = Math.max(0, Math.min(1, progress));
        const totalDuration = this.endTime.getTime() - this.startTime.getTime();
        const newTime = new Date(this.startTime.getTime() + totalDuration * clampedProgress);

        this.currentTime = newTime;
        this.updateCurrentIndex();
        this.updateCurrentPlaybacks();
    }

    seekToTime(time: Date): void {
        if (!this.startTime || !this.endTime) return;

        const clampedTime = new Date(Math.max(this.startTime.getTime(), Math.min(this.endTime.getTime(), time.getTime())));
        this.currentTime = clampedTime;
        this.updateCurrentIndex();
        this.updateCurrentPlaybacks();
    }

    getProgress(): number {
        if (!this.startTime || !this.endTime || !this.currentTime) return 0;

        const totalDuration = this.endTime.getTime() - this.startTime.getTime();
        const currentDuration = this.currentTime.getTime() - this.startTime.getTime();

        return totalDuration > 0 ? currentDuration / totalDuration : 0;
    }

    getCurrentTime(): Date | null {
        return this.currentTime;
    }

    getStartTime(): Date | null {
        return this.startTime;
    }

    getEndTime(): Date | null {
        return this.endTime;
    }

    getTotalPlaybacks(): number {
        return this.playbacks.length;
    }

    isCurrentlyPlaying(): boolean {
        return this.isPlaying;
    }

    private animate(): void {
        if (!this.isPlaying) return;

        const now = performance.now();
        const deltaTime = now - this.lastUpdateTime;
        this.lastUpdateTime = now;

        if (this.currentTime && this.endTime) {
            const intervalMs = this.getIntervalMilliseconds();
            const speedMultiplier = this.playbackSpeed;
            const timeIncrement = (deltaTime / 1000) * intervalMs * speedMultiplier;

            const newTime = new Date(this.currentTime.getTime() + timeIncrement);

            if (newTime.getTime() >= this.endTime.getTime()) {
                this.currentTime = new Date(this.endTime);
                this.pause();
            } else {
                this.currentTime = newTime;
            }

            this.updateCurrentIndex();
            this.updateCurrentPlaybacks();
        }

        if (this.isPlaying) {
            this.animationFrameId = requestAnimationFrame(() => this.animate());
        }
    }

    private getIntervalMilliseconds(): number {
        switch (this.timeInterval) {
            case 'hourly':
                return 3600 * 1000;
            case 'daily':
                return 24 * 3600 * 1000;
            case 'weekly':
                return 7 * 24 * 3600 * 1000;
            default:
                return 3600 * 1000;
        }
    }

    private updateCurrentIndex(): void {
        if (!this.currentTime) return;

        const currentTimeMs = this.currentTime.getTime();

        for (let i = 0; i < this.playbacks.length; i++) {
            const playbackTime = new Date(this.playbacks[i].started_at).getTime();
            if (playbackTime > currentTimeMs) {
                this.currentIndex = i;
                return;
            }
        }

        this.currentIndex = this.playbacks.length;
    }

    private updateCurrentPlaybacks(): void {
        if (!this.currentTime || !this.onTimeUpdate) return;

        const currentTimeMs = this.currentTime.getTime();
        const windowMs = this.getIntervalMilliseconds();
        const windowStart = currentTimeMs - windowMs;

        const currentPlaybacks = this.playbacks.filter(playback => {
            const playbackTime = new Date(playback.started_at).getTime();
            return playbackTime >= windowStart && playbackTime <= currentTimeMs;
        });

        this.onTimeUpdate(this.currentTime, currentPlaybacks, this.currentIndex);
    }

    destroy(): void {
        this.pause();
        this.playbacks = [];
    }
}
