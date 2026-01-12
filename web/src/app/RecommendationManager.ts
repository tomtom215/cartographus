// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * RecommendationManager
 *
 * Manages the recommendation UI panel, displaying personalized recommendations,
 * training status, and allowing users to provide feedback on suggestions.
 */

import type { API } from '../lib/api';
import type { ToastManager } from '../lib/toast';
import type {
    RecommendationResponse,
    TrainingStatus,
    RecommendConfig,
    ScoredItem,
    AlgorithmInfo,
    AlgorithmType,
    WhatsNextResponse,
    WhatsNextPrediction,
} from '../lib/types/recommend';
import { createLogger } from '../lib/logger';

const logger = createLogger('RecommendationManager');

/**
 * Static algorithm information for tooltips and display.
 * This provides detailed explanations for each algorithm type.
 */
const ALGORITHM_INFO: Record<AlgorithmType, AlgorithmInfo> = {
    covisit: {
        id: 'covisit',
        name: 'Co-Visitation',
        description: 'Items frequently watched together',
        tooltip: 'Tracks which items are watched in the same session. If users who watched "The Matrix" also watched "Inception", these items have high co-visitation scores. Great for finding related content.',
        category: 'basic',
        lightweight: true,
    },
    content: {
        id: 'content',
        name: 'Content-Based',
        description: 'Similar genres, actors, directors',
        tooltip: 'Analyzes item attributes like genre, actors, directors, and release year. Recommends items with similar characteristics to what you have watched. Works well for new users.',
        category: 'basic',
        lightweight: true,
    },
    popularity: {
        id: 'popularity',
        name: 'Popularity',
        description: 'Trending items with time decay',
        tooltip: 'Recommends currently popular items. Uses time decay so recent activity counts more than old views. Good for discovering what is trending on your server.',
        category: 'basic',
        lightweight: true,
    },
    ease: {
        id: 'ease',
        name: 'EASE',
        description: 'Embarrassingly Shallow Autoencoders',
        tooltip: 'A surprisingly effective matrix factorization method that learns item-item relationships. Despite its simplicity, it often outperforms complex deep learning models. Fast training.',
        category: 'matrix',
        lightweight: false,
    },
    als: {
        id: 'als',
        name: 'ALS',
        description: 'Alternating Least Squares',
        tooltip: 'Classic matrix factorization that learns latent factors for users and items. Handles implicit feedback (views) well. The foundation of many production recommendation systems.',
        category: 'matrix',
        lightweight: false,
    },
    usercf: {
        id: 'usercf',
        name: 'User-CF',
        description: 'User-based collaborative filtering',
        tooltip: 'Finds users with similar viewing patterns to you, then recommends what they watched. "Users like you also watched..." Great for serendipitous discoveries.',
        category: 'collaborative',
        lightweight: false,
    },
    itemcf: {
        id: 'itemcf',
        name: 'Item-CF',
        description: 'Item-based collaborative filtering',
        tooltip: 'Finds items that are similar based on who watched them. If two items are watched by the same users, they are considered similar. Very stable recommendations.',
        category: 'collaborative',
        lightweight: false,
    },
    fpmc: {
        id: 'fpmc',
        name: 'FPMC',
        description: 'Factorized Personalized Markov Chains',
        tooltip: 'Combines sequential patterns with personalization. Learns what YOU specifically watch after certain items, not just global patterns. Best for personalized "watch next".',
        category: 'sequential',
        lightweight: false,
    },
    markov: {
        id: 'markov',
        name: 'Markov Chain',
        description: 'Sequential viewing patterns',
        tooltip: 'Learns viewing sequences: "After watching X, people usually watch Y". Simple but effective for "What to Watch Next" predictions based on your last viewed item.',
        category: 'sequential',
        lightweight: true,
    },
    bpr: {
        id: 'bpr',
        name: 'BPR',
        description: 'Bayesian Personalized Ranking',
        tooltip: 'Optimizes for ranking quality, not rating prediction. Learns that you prefer watched items over unwatched ones. Excellent for implicit feedback scenarios.',
        category: 'advanced',
        lightweight: false,
    },
    timeaware: {
        id: 'timeaware',
        name: 'Time-Aware CF',
        description: 'Time-weighted collaborative filtering',
        tooltip: 'Weighs recent interactions more heavily than old ones. Your preferences evolve over time - this algorithm adapts to your changing tastes.',
        category: 'advanced',
        lightweight: false,
    },
    multihop: {
        id: 'multihop',
        name: 'Multi-Hop ItemCF',
        description: 'Graph-based item similarity',
        tooltip: 'Explores multi-hop connections between items. If A is similar to B, and B is similar to C, then A might be relevant to C. Discovers hidden connections.',
        category: 'advanced',
        lightweight: false,
    },
    linucb: {
        id: 'linucb',
        name: 'LinUCB',
        description: 'Contextual bandit exploration',
        tooltip: 'Balances exploitation (showing what you like) with exploration (trying new things). Learns from your feedback in real-time. Prevents filter bubbles.',
        category: 'bandit',
        lightweight: true,
    },
};

/**
 * Get algorithm info by ID.
 */
function getAlgorithmInfo(id: string): AlgorithmInfo | undefined {
    return ALGORITHM_INFO[id as AlgorithmType];
}

/**
 * Manages the recommendation display panel and user interactions.
 */
export class RecommendationManager {
    private api: API | null = null;
    private toast: ToastManager | null = null;
    private panel: HTMLElement | null = null;
    private overlay: HTMLElement | null = null;
    private isOpen = false;
    private currentUserId: number | null = null;
    private trainingStatus: TrainingStatus | null = null;
    private config: RecommendConfig | null = null;
    private refreshInterval: number | null = null;
    private lastWatchedItemId: number | null = null;
    private lastWatchedItemTitle: string | null = null;
    private whatsNextResponse: WhatsNextResponse | null = null;
    private activeTab: 'recommendations' | 'whatsnext' | 'algorithms' = 'recommendations';

    constructor(api?: API) {
        if (api) {
            this.api = api;
        }
    }

    /**
     * Set the API client.
     */
    setAPI(api: API): void {
        this.api = api;
    }

    /**
     * Set the toast manager for notifications.
     */
    setToastManager(toast: ToastManager): void {
        this.toast = toast;
    }

    /**
     * Initialize the recommendation manager.
     */
    init(): void {
        this.panel = document.getElementById('recommendation-panel');
        this.overlay = document.getElementById('recommendation-overlay');

        if (!this.panel) {
            logger.warn('Recommendation panel not found in DOM');
            return;
        }

        this.setupEventListeners();
        this.loadConfig();
    }

    /**
     * Set up event listeners for the recommendation panel.
     */
    private setupEventListeners(): void {
        this.setupPanelControls();
        this.setupModeAndTabSelectors();
        this.setupActionButtons();
        this.setupItemSelectionListener();
    }

    private setupPanelControls(): void {
        document.getElementById('recommendation-button')?.addEventListener('click', () => {
            this.open();
        });

        this.panel?.querySelector('[data-action="close-recommendations"]')?.addEventListener('click', () => {
            this.close();
        });

        this.overlay?.addEventListener('click', () => {
            this.close();
        });

        document.addEventListener('keydown', (e) => {
            if (e.key === 'Escape' && this.isOpen) {
                this.close();
            }
        });
    }

    private setupModeAndTabSelectors(): void {
        this.panel?.querySelectorAll('[data-mode]').forEach((btn) => {
            btn.addEventListener('click', (e) => {
                const mode = (e.target as HTMLElement).dataset.mode;
                if (mode) {
                    this.setMode(mode as 'personalized' | 'trending' | 'similar');
                }
            });
        });

        this.panel?.querySelectorAll('[data-tab]').forEach((btn) => {
            btn.addEventListener('click', (e) => {
                const tab = (e.target as HTMLElement).dataset.tab;
                if (tab) {
                    this.setActiveTab(tab as 'recommendations' | 'whatsnext' | 'algorithms');
                }
            });
        });
    }

    private setupActionButtons(): void {
        this.panel?.querySelector('[data-action="refresh-recommendations"]')?.addEventListener('click', () => {
            if (this.activeTab === 'whatsnext') {
                this.loadWhatsNext();
            } else {
                this.loadRecommendations();
            }
        });

        this.panel?.querySelector('[data-action="trigger-training"]')?.addEventListener('click', () => {
            this.triggerTraining();
        });
    }

    private setupItemSelectionListener(): void {
        document.addEventListener('item-selected', ((e: CustomEvent) => {
            if (e.detail?.itemId) {
                this.setLastWatchedItem(e.detail.itemId, e.detail.title);
            }
        }) as EventListener);
    }

    /**
     * Set the active tab in the recommendation panel.
     */
    private setActiveTab(tab: 'recommendations' | 'whatsnext' | 'algorithms'): void {
        this.activeTab = tab;
        this.updateTabButtonStates(tab);
        this.updateTabContentVisibility(tab);
        this.loadTabContent(tab);
    }

    private updateTabButtonStates(activeTab: string): void {
        this.panel?.querySelectorAll('[data-tab]').forEach((btn) => {
            btn.classList.toggle('active', (btn as HTMLElement).dataset.tab === activeTab);
        });
    }

    private updateTabContentVisibility(activeTab: string): void {
        const contentMap = {
            recommendations: '.recommendations-tab-content',
            whatsnext: '.whatsnext-tab-content',
            algorithms: '.algorithms-tab-content',
        };

        Object.entries(contentMap).forEach(([tab, selector]) => {
            const content = this.panel?.querySelector(selector);
            if (content) {
                (content as HTMLElement).style.display = tab === activeTab ? 'block' : 'none';
            }
        });
    }

    private loadTabContent(tab: 'recommendations' | 'whatsnext' | 'algorithms'): void {
        if (tab === 'whatsnext' && this.lastWatchedItemId) {
            this.loadWhatsNext();
        } else if (tab === 'algorithms') {
            this.renderAlgorithmsPanel();
        }
    }

    /**
     * Set the last watched item for What's Next predictions.
     */
    setLastWatchedItem(itemId: number, title?: string): void {
        this.lastWatchedItemId = itemId;
        this.lastWatchedItemTitle = title || null;

        // Update the What's Next header if visible
        const sourceEl = this.panel?.querySelector('.whatsnext-source');
        if (sourceEl && title) {
            sourceEl.textContent = `Based on: ${title}`;
        }

        // Auto-load if on What's Next tab
        if (this.isOpen && this.activeTab === 'whatsnext') {
            this.loadWhatsNext();
        }
    }

    /**
     * Open the recommendation panel.
     */
    open(): void {
        if (!this.panel || !this.overlay) return;

        this.panel.style.display = 'flex';
        this.overlay.style.display = 'block';
        this.panel.setAttribute('aria-hidden', 'false');
        this.isOpen = true;

        // Load recommendations if user is set
        if (this.currentUserId !== null) {
            this.loadRecommendations();
        }

        // Start status refresh
        this.startStatusRefresh();
    }

    /**
     * Close the recommendation panel.
     */
    close(): void {
        if (!this.panel || !this.overlay) return;

        this.panel.style.display = 'none';
        this.overlay.style.display = 'none';
        this.panel.setAttribute('aria-hidden', 'true');
        this.isOpen = false;

        // Stop status refresh
        this.stopStatusRefresh();
    }

    /**
     * Set the current user for recommendations.
     */
    setCurrentUser(userId: number): void {
        this.currentUserId = userId;
        if (this.isOpen) {
            this.loadRecommendations();
        }
    }

    /**
     * Load configuration from the server.
     */
    private async loadConfig(): Promise<void> {
        if (!this.api) return;

        try {
            this.config = await this.api.getRecommendConfig();
            this.updateConfigUI();
        } catch (error) {
            logger.error('Failed to load recommendation config', { error });
        }
    }

    /**
     * Load recommendations for the current user.
     */
    private async loadRecommendations(): Promise<void> {
        if (!this.api || this.currentUserId === null) {
            this.showEmptyState('Select a user to see recommendations');
            return;
        }

        this.showLoading();

        try {
            const response = await this.api.getRecommendations({
                user_id: this.currentUserId,
                k: 12,
                mode: 'personalized',
            });

            this.renderRecommendations(response);
        } catch (error) {
            logger.error('Failed to load recommendations', { error });
            this.showError('Failed to load recommendations');
        }
    }

    /**
     * Set the recommendation mode.
     */
    private async setMode(mode: 'personalized' | 'trending' | 'similar'): Promise<void> {
        this.updateModeButtonStates(mode);

        if (!this.api) return;

        this.showLoading();

        try {
            const response = await this.fetchRecommendationsByMode(mode);
            if (response) {
                this.renderRecommendations(response);
            }
        } catch (error) {
            logger.error('Failed to load recommendations', { error });
            this.showError('Failed to load recommendations');
        }
    }

    private updateModeButtonStates(activeMode: string): void {
        this.panel?.querySelectorAll('[data-mode]').forEach((btn) => {
            btn.classList.toggle('active', (btn as HTMLElement).dataset.mode === activeMode);
        });
    }

    private async fetchRecommendationsByMode(mode: string): Promise<RecommendationResponse | null> {
        if (!this.api) return null;

        if (mode === 'trending') {
            return await this.api.getTrending(12);
        }

        if (mode === 'personalized' && this.currentUserId !== null) {
            return await this.api.getRecommendations({
                user_id: this.currentUserId,
                k: 12,
                mode: 'personalized',
            });
        }

        this.showEmptyState('Select a user for personalized recommendations');
        return null;
    }

    /**
     * Render recommendations in the panel.
     */
    private renderRecommendations(response: RecommendationResponse): void {
        const container = this.panel?.querySelector('.recommendation-grid');
        if (!container) return;

        if (response.items.length === 0) {
            this.showEmptyState('No recommendations available yet. Try watching more content!');
            return;
        }

        container.innerHTML = response.items.map((item) => this.renderRecommendationCard(item)).join('');
        this.updateRecommendationMetadata(response.metadata);
        this.attachRecommendationCardHandlers(container);
    }

    private updateRecommendationMetadata(metadata: RecommendationResponse['metadata']): void {
        const metaEl = this.panel?.querySelector('.recommendation-meta');
        if (!metaEl) return;

        metaEl.innerHTML = `
            <span class="recommendation-meta-item">
                <span class="meta-label">Algorithms:</span>
                <span class="meta-value">${metadata.algorithms_used.join(', ') || 'N/A'}</span>
            </span>
            <span class="recommendation-meta-item">
                <span class="meta-label">Latency:</span>
                <span class="meta-value">${metadata.latency_ms}ms</span>
            </span>
            <span class="recommendation-meta-item">
                <span class="meta-label">Model:</span>
                <span class="meta-value">v${metadata.model_version}</span>
            </span>
            ${metadata.cache_hit ? '<span class="recommendation-meta-item cache-hit">Cached</span>' : ''}
        `;
    }

    private attachRecommendationCardHandlers(container: Element): void {
        container.querySelectorAll('.recommendation-card').forEach((card) => {
            card.addEventListener('click', () => {
                const itemId = parseInt((card as HTMLElement).dataset.itemId || '0', 10);
                if (itemId > 0) {
                    this.handleCardClick(itemId);
                }
            });
        });
    }

    /**
     * Render a single recommendation card.
     */
    private renderRecommendationCard(item: ScoredItem): string {
        const { item: media, score, explanation } = item;
        const scorePercent = Math.round(score * 100);

        return `
            <div class="recommendation-card" data-item-id="${media.id}" tabindex="0" role="button">
                ${this.renderPosterWithScore(media, scorePercent)}
                ${this.renderCardInfo(media, explanation)}
            </div>
        `;
    }

    private renderPosterWithScore(media: ScoredItem['item'], scorePercent: number): string {
        const posterContent = media.poster_url
            ? `<img src="${media.poster_url}" alt="${media.title}" loading="lazy" />`
            : `<div class="recommendation-poster-placeholder">${media.media_type === 'movie' ? '🎬' : '📺'}</div>`;

        return `
            <div class="recommendation-poster">
                ${posterContent}
                ${this.renderScoreRing(scorePercent)}
            </div>
        `;
    }

    private renderScoreRing(scorePercent: number): string {
        return `
            <div class="recommendation-score" title="Match score: ${scorePercent}%">
                <svg viewBox="0 0 36 36" class="score-ring">
                    <path class="score-ring-bg" d="M18 2.0845 a 15.9155 15.9155 0 0 1 0 31.831 a 15.9155 15.9155 0 0 1 0 -31.831" />
                    <path class="score-ring-fill" stroke-dasharray="${scorePercent}, 100" d="M18 2.0845 a 15.9155 15.9155 0 0 1 0 31.831 a 15.9155 15.9155 0 0 1 0 -31.831" />
                </svg>
                <span class="score-text">${scorePercent}</span>
            </div>
        `;
    }

    private renderCardInfo(media: ScoredItem['item'], explanation?: string): string {
        return `
            <div class="recommendation-info">
                <h4 class="recommendation-title" title="${media.title}">${media.title || 'Unknown'}</h4>
                ${media.parent_title ? `<span class="recommendation-parent">${media.parent_title}</span>` : ''}
                <div class="recommendation-metadata">
                    ${media.year ? `<span class="recommendation-year">${media.year}</span>` : ''}
                    ${media.genres?.length ? `<span class="recommendation-genre">${media.genres[0]}</span>` : ''}
                </div>
                ${explanation ? `<p class="recommendation-explanation">${explanation}</p>` : ''}
            </div>
        `;
    }

    /**
     * Handle card click - record feedback and potentially navigate.
     */
    private async handleCardClick(itemId: number): Promise<void> {
        if (!this.api || this.currentUserId === null) return;

        try {
            await this.api.recordRecommendFeedback(this.currentUserId, itemId, 'click');
        } catch (error) {
            logger.error('Failed to record feedback', { error, itemId });
        }

        // Could emit an event for navigation here
        const event = new CustomEvent('recommendation-selected', {
            detail: { itemId },
        });
        document.dispatchEvent(event);
    }

    /**
     * Trigger model training.
     */
    private async triggerTraining(): Promise<void> {
        if (!this.api) return;

        try {
            this.toast?.show({ message: 'Starting model training...', type: 'info' });
            await this.api.triggerRecommendTraining();
            this.toast?.show({ message: 'Training started successfully', type: 'success' });
            this.loadTrainingStatus();
        } catch (error) {
            logger.error('Failed to trigger training', { error });
            this.toast?.show({ message: 'Failed to start training', type: 'error' });
        }
    }

    /**
     * Load training status from the server.
     */
    private async loadTrainingStatus(): Promise<void> {
        if (!this.api) return;

        try {
            this.trainingStatus = await this.api.getRecommendTrainingStatus();
            this.updateTrainingUI();
        } catch (error) {
            logger.error('Failed to load training status', { error });
        }
    }

    /**
     * Update the training status UI.
     */
    private updateTrainingUI(): void {
        const statusEl = this.panel?.querySelector('.training-status');
        if (!statusEl || !this.trainingStatus) return;

        const { is_training } = this.trainingStatus;

        if (is_training) {
            this.renderActiveTrainingStatus(statusEl);
        } else {
            this.renderIdleTrainingStatus(statusEl);
        }
    }

    private renderActiveTrainingStatus(statusEl: Element): void {
        const { progress, current_algorithm } = this.trainingStatus!;

        statusEl.innerHTML = `
            <div class="training-progress">
                <div class="training-progress-bar" style="width: ${progress}%"></div>
            </div>
            <span class="training-label">Training: ${current_algorithm} (${progress}%)</span>
        `;
        statusEl.classList.add('training-active');
    }

    private renderIdleTrainingStatus(statusEl: Element): void {
        const { last_trained_at, model_version, interaction_count, user_count, item_count } = this.trainingStatus!;
        const trainedDate = last_trained_at ? new Date(last_trained_at).toLocaleString() : 'Never';

        statusEl.innerHTML = `
            <div class="training-info">
                <span class="training-label">Model v${model_version}</span>
                <span class="training-detail">Last trained: ${trainedDate}</span>
                <span class="training-detail">${interaction_count.toLocaleString()} interactions | ${user_count.toLocaleString()} users | ${item_count.toLocaleString()} items</span>
            </div>
        `;
        statusEl.classList.remove('training-active');
    }

    /**
     * Update the config display in the UI.
     */
    private updateConfigUI(): void {
        const configEl = this.panel?.querySelector('.recommendation-config');
        if (!configEl || !this.config) return;

        configEl.innerHTML = this.config.enabled
            ? this.renderEnabledConfigHTML()
            : this.renderDisabledConfigHTML();
    }

    private renderDisabledConfigHTML(): string {
        return `
            <div class="recommendation-disabled">
                <span class="disabled-icon">⚠️</span>
                <span class="disabled-text">Recommendations are disabled. Enable in settings with RECOMMEND_ENABLED=true</span>
            </div>
        `;
    }

    private renderEnabledConfigHTML(): string {
        if (!this.config) return '';

        return `
            <div class="config-item">
                <span class="config-label">Algorithms:</span>
                <span class="config-value">${this.config.algorithms.join(', ')}</span>
            </div>
            <div class="config-item">
                <span class="config-label">Train interval:</span>
                <span class="config-value">${this.config.train_interval}</span>
            </div>
            <div class="config-item">
                <span class="config-label">Diversity:</span>
                <span class="config-value">${Math.round(this.config.diversity_lambda * 100)}%</span>
            </div>
        `;
    }

    /**
     * Show loading state.
     */
    private showLoading(): void {
        const container = this.panel?.querySelector('.recommendation-grid');
        if (!container) return;

        container.innerHTML = `
            <div class="recommendation-loading">
                <div class="loading-spinner"></div>
                <span>Loading recommendations...</span>
            </div>
        `;
    }

    /**
     * Show empty state.
     */
    private showEmptyState(message: string): void {
        const container = this.panel?.querySelector('.recommendation-grid');
        if (!container) return;

        container.innerHTML = `
            <div class="recommendation-empty">
                <span class="empty-icon">📭</span>
                <span class="empty-text">${message}</span>
            </div>
        `;
    }

    /**
     * Show error state.
     */
    private showError(message: string): void {
        const container = this.panel?.querySelector('.recommendation-grid');
        if (!container) return;

        container.innerHTML = `
            <div class="recommendation-error">
                <span class="error-icon">⚠️</span>
                <span class="error-text">${message}</span>
                <button class="btn-retry" data-action="refresh-recommendations">Retry</button>
            </div>
        `;
    }

    /**
     * Start periodic status refresh.
     */
    private startStatusRefresh(): void {
        this.stopStatusRefresh();
        this.loadTrainingStatus();
        this.refreshInterval = window.setInterval(() => {
            this.loadTrainingStatus();
        }, 5000);
    }

    /**
     * Stop periodic status refresh.
     */
    private stopStatusRefresh(): void {
        if (this.refreshInterval !== null) {
            clearInterval(this.refreshInterval);
            this.refreshInterval = null;
        }
    }

    /**
     * Load What's Next predictions using Markov chain.
     */
    private async loadWhatsNext(): Promise<void> {
        if (!this.api || !this.lastWatchedItemId) {
            this.showWhatsNextEmpty('Watch something first to see predictions');
            return;
        }

        this.showWhatsNextLoading();

        try {
            this.whatsNextResponse = await this.api.getWhatsNext({
                last_item_id: this.lastWatchedItemId,
                k: 6,
                exclude_watched: true,
                user_id: this.currentUserId ?? undefined,
            });

            this.renderWhatsNext();
        } catch (error) {
            logger.error('Failed to load What\'s Next predictions', { error });
            this.showWhatsNextError('Failed to load predictions');
        }
    }

    /**
     * Render What's Next predictions.
     */
    private renderWhatsNext(): void {
        const container = this.panel?.querySelector('.whatsnext-grid');
        if (!container || !this.whatsNextResponse) return;

        if (this.whatsNextResponse.predictions.length === 0) {
            this.showWhatsNextEmpty('No predictions available for this item yet');
            return;
        }

        this.updateWhatsNextSource();
        container.innerHTML = this.whatsNextResponse.predictions
            .map((pred) => this.renderWhatsNextCard(pred))
            .join('');
        this.updateWhatsNextMetadata();
        this.attachWhatsNextCardHandlers(container);
    }

    private updateWhatsNextSource(): void {
        const sourceEl = this.panel?.querySelector('.whatsnext-source');
        if (!sourceEl || !this.whatsNextResponse) return;

        const title = this.whatsNextResponse.source_item_title || this.lastWatchedItemTitle || 'Unknown';
        sourceEl.innerHTML = `
            <span class="whatsnext-source-label">Based on:</span>
            <span class="whatsnext-source-title">${title}</span>
            <span class="whatsnext-source-transitions">(${this.whatsNextResponse.transition_count} viewing patterns)</span>
        `;
    }

    private updateWhatsNextMetadata(): void {
        const metaEl = this.panel?.querySelector('.whatsnext-meta');
        if (!metaEl || !this.whatsNextResponse?.metadata) return;

        metaEl.innerHTML = `
            <span class="meta-item">
                <span class="meta-label">Latency:</span>
                <span class="meta-value">${this.whatsNextResponse.metadata.latency_ms}ms</span>
            </span>
            <span class="meta-item">
                <span class="meta-label">Model:</span>
                <span class="meta-value">v${this.whatsNextResponse.metadata.model_version}</span>
            </span>
        `;
    }

    private attachWhatsNextCardHandlers(container: Element): void {
        container.querySelectorAll('.whatsnext-card').forEach((card) => {
            card.addEventListener('click', () => {
                const itemId = parseInt((card as HTMLElement).dataset.itemId || '0', 10);
                if (itemId > 0) {
                    this.handleWhatsNextClick(itemId);
                }
            });
        });
    }

    /**
     * Render a What's Next prediction card.
     */
    private renderWhatsNextCard(pred: WhatsNextPrediction): string {
        const { item, probability, transition_count, reason } = pred;
        const probPercent = Math.round(probability * 100);

        return `
            <div class="whatsnext-card" data-item-id="${item.id}" tabindex="0" role="button">
                ${this.renderWhatsNextPoster(item, probPercent)}
                ${this.renderWhatsNextInfo(item, transition_count, reason)}
            </div>
        `;
    }

    private renderWhatsNextPoster(item: WhatsNextPrediction['item'], probPercent: number): string {
        const posterContent = item.poster_url
            ? `<img src="${item.poster_url}" alt="${item.title}" loading="lazy" />`
            : `<div class="whatsnext-poster-placeholder">${item.media_type === 'movie' ? '🎬' : '📺'}</div>`;

        return `
            <div class="whatsnext-poster">
                ${posterContent}
                <div class="whatsnext-probability" title="Transition probability: ${probPercent}%">
                    <span class="probability-value">${probPercent}%</span>
                    <span class="probability-label">likely</span>
                </div>
            </div>
        `;
    }

    private renderWhatsNextInfo(item: WhatsNextPrediction['item'], transitionCount: number, reason?: string): string {
        return `
            <div class="whatsnext-info">
                <h4 class="whatsnext-title" title="${item.title}">${item.title || 'Unknown'}</h4>
                ${item.parent_title ? `<span class="whatsnext-parent">${item.parent_title}</span>` : ''}
                <div class="whatsnext-metadata">
                    ${item.year ? `<span class="whatsnext-year">${item.year}</span>` : ''}
                    <span class="whatsnext-count" title="Times this transition occurred">${transitionCount}x</span>
                </div>
                ${reason ? `<p class="whatsnext-reason">${reason}</p>` : ''}
            </div>
        `;
    }

    /**
     * Handle What's Next card click.
     */
    private handleWhatsNextClick(itemId: number): void {
        // Update the last watched item to enable chaining
        const pred = this.whatsNextResponse?.predictions.find((p) => p.item.id === itemId);
        if (pred) {
            this.setLastWatchedItem(itemId, pred.item.title);
        }

        // Emit event for navigation
        const event = new CustomEvent('whatsnext-selected', {
            detail: { itemId },
        });
        document.dispatchEvent(event);
    }

    /**
     * Show What's Next loading state.
     */
    private showWhatsNextLoading(): void {
        const container = this.panel?.querySelector('.whatsnext-grid');
        if (!container) return;

        container.innerHTML = `
            <div class="whatsnext-loading">
                <div class="loading-spinner"></div>
                <span>Analyzing viewing patterns...</span>
            </div>
        `;
    }

    /**
     * Show What's Next empty state.
     */
    private showWhatsNextEmpty(message: string): void {
        const container = this.panel?.querySelector('.whatsnext-grid');
        if (!container) return;

        container.innerHTML = `
            <div class="whatsnext-empty">
                <span class="empty-icon">🔮</span>
                <span class="empty-text">${message}</span>
            </div>
        `;
    }

    /**
     * Show What's Next error state.
     */
    private showWhatsNextError(message: string): void {
        const container = this.panel?.querySelector('.whatsnext-grid');
        if (!container) return;

        container.innerHTML = `
            <div class="whatsnext-error">
                <span class="error-icon">⚠</span>
                <span class="error-text">${message}</span>
                <button class="btn-retry" data-action="refresh-whatsnext">Retry</button>
            </div>
        `;
    }

    /**
     * Render the algorithms information panel with tooltips.
     */
    private renderAlgorithmsPanel(): void {
        const container = this.panel?.querySelector('.algorithms-tab-content');
        if (!container) return;

        const enabledAlgorithms = new Set(this.config?.algorithms || []);
        const categories = this.groupAlgorithmsByCategory();

        container.innerHTML = `
            <div class="algorithms-panel">
                ${this.renderAlgorithmsIntro()}
                ${this.renderAlgorithmCategories(categories, enabledAlgorithms)}
            </div>
        `;

        this.attachAlgorithmTooltips();
    }

    private groupAlgorithmsByCategory(): Record<string, AlgorithmInfo[]> {
        const categories: Record<string, AlgorithmInfo[]> = {
            basic: [],
            matrix: [],
            collaborative: [],
            sequential: [],
            advanced: [],
            bandit: [],
        };

        Object.values(ALGORITHM_INFO).forEach((info) => {
            if (categories[info.category]) {
                categories[info.category].push(info);
            }
        });

        return categories;
    }

    private renderAlgorithmsIntro(): string {
        return `
            <div class="algorithms-intro">
                <p>Cartographus uses a hybrid recommendation engine that combines multiple algorithms.
                Each algorithm has different strengths. Hover over any algorithm to learn more.</p>
            </div>
        `;
    }

    private renderAlgorithmCategories(categories: Record<string, AlgorithmInfo[]>, enabledAlgorithms: Set<string>): string {
        const categoryLabels: Record<string, string> = {
            basic: 'Basic Algorithms',
            matrix: 'Matrix Factorization',
            collaborative: 'Collaborative Filtering',
            sequential: 'Sequential Patterns',
            advanced: 'Advanced Methods',
            bandit: 'Exploration/Bandits',
        };

        return Object.entries(categories)
            .filter(([_, algs]) => algs.length > 0)
            .map(([cat, algs]) => this.renderAlgorithmCategory(cat, algs, categoryLabels[cat], enabledAlgorithms))
            .join('');
    }

    private renderAlgorithmCategory(_category: string, algorithms: AlgorithmInfo[], label: string, enabledAlgorithms: Set<string>): string {
        return `
            <div class="algorithm-category">
                <h4 class="category-title">${label}</h4>
                <div class="algorithm-list">
                    ${algorithms.map((alg) => this.renderAlgorithmCard(alg, enabledAlgorithms.has(alg.id))).join('')}
                </div>
            </div>
        `;
    }

    /**
     * Render a single algorithm info card.
     */
    private renderAlgorithmCard(info: AlgorithmInfo, enabled: boolean): string {
        return `
            <div class="algorithm-card ${enabled ? 'enabled' : 'disabled'}"
                 data-algorithm="${info.id}"
                 data-tooltip="${info.tooltip}">
                <div class="algorithm-header">
                    <span class="algorithm-name">${info.name}</span>
                    <span class="algorithm-status ${enabled ? 'status-enabled' : 'status-disabled'}">
                        ${enabled ? 'Active' : 'Inactive'}
                    </span>
                </div>
                <p class="algorithm-description">${info.description}</p>
                ${info.lightweight ? '<span class="algorithm-badge lightweight">Lightweight</span>' : ''}
            </div>
        `;
    }

    /**
     * Attach tooltip event handlers to algorithm cards.
     */
    private attachAlgorithmTooltips(): void {
        const cards = this.panel?.querySelectorAll('.algorithm-card[data-tooltip]');
        if (!cards) return;

        cards.forEach((card) => {
            const tooltip = (card as HTMLElement).dataset.tooltip;
            if (tooltip) {
                this.attachTooltipToCard(card as HTMLElement, tooltip);
            }
        });
    }

    private attachTooltipToCard(card: HTMLElement, tooltipText: string): void {
        let tooltipEl: HTMLElement | null = null;

        card.addEventListener('mouseenter', () => {
            tooltipEl = this.createAndPositionTooltip(card, tooltipText);
        });

        card.addEventListener('mouseleave', () => {
            if (tooltipEl) {
                tooltipEl.remove();
                tooltipEl = null;
            }
        });
    }

    private createAndPositionTooltip(card: HTMLElement, text: string): HTMLElement {
        const tooltipEl = document.createElement('div');
        tooltipEl.className = 'algorithm-tooltip';
        tooltipEl.textContent = text;
        document.body.appendChild(tooltipEl);

        const rect = card.getBoundingClientRect();
        tooltipEl.style.position = 'fixed';
        tooltipEl.style.left = `${rect.left}px`;
        tooltipEl.style.top = `${rect.bottom + 8}px`;
        tooltipEl.style.maxWidth = `${Math.min(400, window.innerWidth - rect.left - 20)}px`;

        return tooltipEl;
    }

    /**
     * Get algorithm info by ID (exposed for external use).
     */
    getAlgorithmInfo(algorithmId: string): AlgorithmInfo | undefined {
        return getAlgorithmInfo(algorithmId);
    }

    /**
     * Get all algorithm info (exposed for external use).
     */
    getAllAlgorithmInfo(): Record<AlgorithmType, AlgorithmInfo> {
        return { ...ALGORITHM_INFO };
    }

    /**
     * Clean up resources.
     */
    destroy(): void {
        this.stopStatusRefresh();
    }
}
