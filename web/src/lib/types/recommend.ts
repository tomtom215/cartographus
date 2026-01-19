// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Recommendation Engine Types
 *
 * Type definitions for the recommendation API endpoints.
 */

/**
 * Recommendation request parameters.
 */
export interface RecommendationRequest {
    /** User ID to get recommendations for */
    user_id: number;
    /** Number of recommendations to return (default: 10) */
    k?: number;
    /** Recommendation mode */
    mode?: RecommendationMode;
    /** Current item ID for "similar items" recommendations */
    current_item_id?: number;
    /** Item IDs to exclude from recommendations */
    exclude_ids?: number[];
}

/**
 * Recommendation mode types.
 */
export type RecommendationMode = 'personalized' | 'similar' | 'trending' | 'new';

/**
 * A single scored recommendation item.
 */
export interface ScoredItem {
    /** Item details */
    item: RecommendedItem;
    /** Combined recommendation score (0-1) */
    score: number;
    /** Per-algorithm score breakdown */
    scores?: Record<string, number>;
    /** Explanation for why this was recommended */
    explanation?: string;
}

/**
 * Recommended item metadata.
 */
export interface RecommendedItem {
    /** Item ID */
    id: number;
    /** Item title */
    title?: string;
    /** Media type: movie, episode, track */
    media_type?: string;
    /** Genres */
    genres?: string[];
    /** Release year */
    year?: number;
    /** Poster/thumbnail URL */
    poster_url?: string;
    /** Show/album title for episodes/tracks */
    parent_title?: string;
}

/**
 * Response from recommendation endpoint.
 */
export interface RecommendationResponse {
    /** Recommended items with scores */
    items: ScoredItem[];
    /** Total candidates considered */
    total_candidates: number;
    /** Response metadata */
    metadata: RecommendationMetadata;
}

/**
 * Recommendation response metadata.
 */
export interface RecommendationMetadata {
    /** Unique request ID for tracing */
    request_id: string;
    /** User ID recommendations are for */
    user_id: number;
    /** Recommendation mode used */
    mode: string;
    /** Algorithms that contributed to scores */
    algorithms_used: string[];
    /** Response latency in milliseconds */
    latency_ms: number;
    /** Whether response came from cache */
    cache_hit: boolean;
    /** Current model version */
    model_version: number;
    /** When models were last trained */
    trained_at: string;
    /** Response timestamp */
    timestamp: string;
}

/**
 * Training status for the recommendation engine.
 */
export interface TrainingStatus {
    /** Whether training is currently in progress */
    is_training: boolean;
    /** Training progress percentage (0-100) */
    progress: number;
    /** Currently training algorithm */
    current_algorithm: string;
    /** Last error message if training failed */
    last_error: string;
    /** Last successful training timestamp */
    last_trained_at: string;
    /** Duration of last training in milliseconds */
    last_training_duration_ms: number;
    /** Current model version */
    model_version: number;
    /** Number of interactions in training data */
    interaction_count: number;
    /** Number of unique users */
    user_count: number;
    /** Number of unique items */
    item_count: number;
}

/**
 * Recommendation engine configuration.
 */
export interface RecommendConfig {
    /** Whether engine is enabled */
    enabled: boolean;
    /** Enabled algorithm names */
    algorithms: string[];
    /** Training interval */
    train_interval: string;
    /** Whether to train on startup */
    train_on_startup: boolean;
    /** Minimum interactions before training */
    min_interactions: number;
    /** Cache TTL */
    cache_ttl: string;
    /** Maximum candidates to score */
    max_candidates: number;
    /** Diversity parameter (0-1) */
    diversity_lambda: number;
    /** Whether calibration reranking is enabled */
    calibration_enabled: boolean;
}

/**
 * Recommendation engine metrics.
 */
export interface RecommendMetrics {
    /** Total requests served */
    request_count: number;
    /** Cache hit count */
    cache_hits: number;
    /** Cache miss count */
    cache_misses: number;
    /** Error count */
    error_count: number;
    /** Per-algorithm metrics */
    algorithm_metrics: Record<string, AlgorithmMetrics>;
}

/**
 * Per-algorithm performance metrics.
 */
export interface AlgorithmMetrics {
    /** Algorithm name */
    name: string;
    /** Average prediction latency in milliseconds */
    avg_latency_ms: number;
    /** Number of predictions made */
    prediction_count: number;
    /** Whether algorithm is trained */
    is_trained: boolean;
    /** Training duration in milliseconds */
    training_duration_ms: number;
}

/**
 * User preference for recommendations.
 */
export interface UserPreference {
    /** User ID */
    user_id: number;
    /** Preferred genres with weights */
    genre_weights?: Record<string, number>;
    /** Exploration preference (0=exploitation, 1=exploration) */
    exploration_level?: number;
    /** Whether to include already-watched items */
    include_watched?: boolean;
    /** Minimum year filter */
    min_year?: number;
    /** Maximum year filter */
    max_year?: number;
}

/**
 * Available recommendation algorithm types.
 */
export type AlgorithmType =
    | 'covisit'       // Co-Visitation: Items frequently watched together
    | 'content'       // Content-Based: Similar genres, actors, directors
    | 'popularity'    // Popularity: Trending items with time decay
    | 'ease'          // EASE: Embarrassingly shallow autoencoders
    | 'als'           // ALS: Alternating least squares matrix factorization
    | 'usercf'        // User-CF: User-based collaborative filtering
    | 'itemcf'        // Item-CF: Item-based collaborative filtering
    | 'fpmc'          // FPMC: Factorized Personalized Markov Chains
    | 'markov'        // Markov: First-order Markov chain for sequential
    | 'bpr'           // BPR: Bayesian Personalized Ranking
    | 'timeaware'     // Time-Aware CF: Time-weighted collaborative filtering
    | 'multihop'      // Multi-Hop: Graph-like item similarity propagation
    | 'linucb';       // LinUCB: Contextual bandit exploration

/**
 * Algorithm information for display in UI.
 */
export interface AlgorithmInfo {
    /** Algorithm ID */
    id: AlgorithmType;
    /** Display name */
    name: string;
    /** Short description */
    description: string;
    /** Detailed tooltip explanation */
    tooltip: string;
    /** Algorithm category for grouping */
    category: 'basic' | 'matrix' | 'collaborative' | 'sequential' | 'advanced' | 'bandit';
    /** Whether this algorithm is lightweight (fast) */
    lightweight: boolean;
}

/**
 * Combined status response from /recommendations/status endpoint.
 * The backend returns both training status and metrics in a single response.
 */
export interface RecommendStatusResponse {
    /** Current training status */
    training: TrainingStatus;
    /** Engine performance metrics */
    metrics: RecommendMetrics;
}

/**
 * Request for "What's Next" predictions (Markov chain).
 */
export interface WhatsNextRequest {
    /** Last watched item ID */
    last_item_id: number;
    /** Number of predictions to return */
    k?: number;
    /** Exclude already-watched items */
    exclude_watched?: boolean;
    /** User ID for filtering watched items */
    user_id?: number;
}

/**
 * Response from "What's Next" endpoint.
 */
export interface WhatsNextResponse {
    /** Predicted next items with probabilities */
    predictions: WhatsNextPrediction[];
    /** Source item ID */
    source_item_id: number;
    /** Source item title */
    source_item_title?: string;
    /** Total transitions from this item */
    transition_count: number;
    /** Response metadata */
    metadata: {
        /** Response latency in ms */
        latency_ms: number;
        /** Whether the item was found */
        found: boolean;
        /** Algorithm version */
        model_version: number;
    };
}

/**
 * A single "What's Next" prediction.
 */
export interface WhatsNextPrediction {
    /** Predicted item details */
    item: RecommendedItem;
    /** Transition probability (0-1) */
    probability: number;
    /** Number of times this transition occurred */
    transition_count: number;
    /** Explanation for the prediction */
    reason?: string;
}
