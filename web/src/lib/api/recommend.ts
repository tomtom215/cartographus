// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Recommendation API Module
 *
 * Provides methods for interacting with the recommendation engine endpoints.
 * Supports getting recommendations, checking training status, and managing preferences.
 */

import { BaseAPIClient } from './client';
import type {
    RecommendationRequest,
    RecommendationResponse,
    TrainingStatus,
    RecommendConfig,
    RecommendMetrics,
    RecommendStatusResponse,
    UserPreference,
    WhatsNextRequest,
    WhatsNextResponse,
    AlgorithmInfo,
} from '../types/recommend';

/**
 * API client for recommendation engine endpoints.
 */
export class RecommendAPI extends BaseAPIClient {
    /**
     * Get personalized recommendations for a user.
     *
     * @param request - Recommendation request parameters
     * @returns Recommendation response with scored items
     */
    async getRecommendations(request: RecommendationRequest): Promise<RecommendationResponse> {
        const params = new URLSearchParams();
        params.set('user_id', request.user_id.toString());

        if (request.k) {
            params.set('k', request.k.toString());
        }
        if (request.mode) {
            params.set('mode', request.mode);
        }
        if (request.current_item_id) {
            params.set('current_item_id', request.current_item_id.toString());
        }
        if (request.exclude_ids && request.exclude_ids.length > 0) {
            params.set('exclude_ids', request.exclude_ids.join(','));
        }

        const response = await this.fetch<RecommendationResponse>(`/recommendations/user/${request.user_id}?${params.toString()}`);
        return response.data;
    }

    /**
     * Get similar items to a given item.
     *
     * @param itemId - Item ID to find similar items for
     * @param k - Number of similar items to return
     * @returns Recommendation response with similar items
     */
    async getSimilarItems(itemId: number, k: number = 10): Promise<RecommendationResponse> {
        return this.getRecommendations({
            user_id: 0, // Not used for similar items
            k,
            mode: 'similar',
            current_item_id: itemId,
        });
    }

    /**
     * Get trending recommendations.
     *
     * @param k - Number of items to return
     * @returns Recommendation response with trending items
     */
    async getTrending(k: number = 10): Promise<RecommendationResponse> {
        return this.getRecommendations({
            user_id: 0,
            k,
            mode: 'trending',
        });
    }

    /**
     * Get the current training status.
     *
     * @returns Training status including progress and last trained time
     */
    async getTrainingStatus(): Promise<TrainingStatus> {
        // FIX: Backend returns combined {training, metrics} - extract training part
        const response = await this.fetch<RecommendStatusResponse>('/recommendations/status');
        return response.data.training;
    }

    /**
     * Trigger model training.
     *
     * @returns Training status after starting training
     */
    async triggerTraining(): Promise<TrainingStatus> {
        const response = await this.fetch<TrainingStatus>('/recommendations/train', {
            method: 'POST',
            body: JSON.stringify({}),
        });
        return response.data;
    }

    /**
     * Get the current recommendation engine configuration.
     *
     * @returns Engine configuration
     */
    async getConfig(): Promise<RecommendConfig> {
        const response = await this.fetch<RecommendConfig>('/recommendations/config');
        return response.data;
    }

    /**
     * Get recommendation engine metrics.
     *
     * @returns Engine performance metrics
     */
    async getMetrics(): Promise<RecommendMetrics> {
        // FIX: Backend returns combined {training, metrics} - extract metrics part
        const response = await this.fetch<RecommendStatusResponse>('/recommendations/status');
        return response.data.metrics;
    }

    /**
     * Get user preferences for recommendations.
     *
     * @param userId - User ID
     * @returns User preference settings
     */
    async getUserPreferences(userId: number): Promise<UserPreference> {
        const response = await this.fetch<UserPreference>(`/recommendations/user/${userId}/preferences`);
        return response.data;
    }

    /**
     * Update user preferences for recommendations.
     *
     * @param userId - User ID
     * @param preferences - Updated preferences
     * @returns Updated preferences
     */
    async updateUserPreferences(userId: number, preferences: Partial<UserPreference>): Promise<UserPreference> {
        const response = await this.fetch<UserPreference>(`/recommendations/user/${userId}/preferences`, {
            method: 'PUT',
            body: JSON.stringify(preferences),
        });
        return response.data;
    }

    /**
     * Record user feedback on a recommendation.
     * Used for online learning (e.g., LinUCB).
     *
     * @param userId - User ID
     * @param itemId - Item ID that was recommended
     * @param feedback - Feedback type: 'click', 'watch', 'skip', 'dismiss'
     */
    async recordFeedback(userId: number, itemId: number, feedback: 'click' | 'watch' | 'skip' | 'dismiss'): Promise<void> {
        await this.fetch('/recommendations/feedback', {
            method: 'POST',
            body: JSON.stringify({
                user_id: userId,
                item_id: itemId,
                feedback,
            }),
        });
    }

    /**
     * Get "What's Next" predictions using Markov chain.
     * Predicts what a user is likely to watch next based on viewing patterns.
     *
     * @param request - What's Next request parameters
     * @returns Predictions with transition probabilities
     */
    async getWhatsNext(request: WhatsNextRequest): Promise<WhatsNextResponse> {
        const params = new URLSearchParams();

        if (request.k) {
            params.set('k', request.k.toString());
        }
        if (request.exclude_watched !== undefined) {
            params.set('exclude_watched', request.exclude_watched.toString());
        }
        if (request.user_id) {
            params.set('user_id', request.user_id.toString());
        }

        const response = await this.fetch<WhatsNextResponse>(`/recommendations/next/${request.last_item_id}?${params.toString()}`);
        return response.data;
    }

    /**
     * Get information about available algorithms.
     * Returns descriptions and configuration info for UI display.
     *
     * @returns List of algorithm information objects
     */
    async getAlgorithms(): Promise<AlgorithmInfo[]> {
        const response = await this.fetch<AlgorithmInfo[]>('/recommendations/algorithms');
        return response.data;
    }

    /**
     * Get per-algorithm performance breakdown.
     *
     * @returns Map of algorithm name to metrics
     */
    async getAlgorithmMetrics(): Promise<Record<string, { latency_ms: number; predictions: number; trained: boolean }>> {
        const response = await this.fetch<Record<string, { latency_ms: number; predictions: number; trained: boolean }>>('/recommendations/algorithms/metrics');
        return response.data;
    }
}
