// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/tomtom215/cartographus/internal/database"
	"github.com/tomtom215/cartographus/internal/logging"
	"github.com/tomtom215/cartographus/internal/models"
	"github.com/tomtom215/cartographus/internal/recommend"
)

// RecommendHandler handles recommendation API endpoints.
type RecommendHandler struct {
	engine       *recommend.Engine
	dataProvider *database.RecommendationDataProvider
	db           *database.DB
}

// NewRecommendHandler creates a new recommendation handler.
func NewRecommendHandler(db *database.DB) (*RecommendHandler, error) {
	cfg := recommend.DefaultConfig()
	logger := logging.Logger()

	engine, err := recommend.NewEngine(cfg, logger)
	if err != nil {
		return nil, err
	}

	dataProvider := database.NewRecommendationDataProvider(db)
	engine.SetDataProvider(dataProvider)

	return &RecommendHandler{
		engine:       engine,
		dataProvider: dataProvider,
		db:           db,
	}, nil
}

// GetRecommendations handles GET /api/v1/recommendations/user/{userID}
// Returns personalized recommendations for a user.
func (h *RecommendHandler) GetRecommendations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	userIDStr := chi.URLParam(r, "userID")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_USER_ID", "Invalid user ID", err)
		return
	}

	// Parse query parameters
	k := 20
	if kStr := r.URL.Query().Get("k"); kStr != "" {
		if parsed, err := strconv.Atoi(kStr); err == nil && parsed > 0 {
			k = parsed
		}
	}

	req := recommend.Request{
		UserID:    userID,
		K:         k,
		Mode:      recommend.ModePersonalized,
		RequestID: r.Header.Get("X-Request-ID"),
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	resp, err := h.engine.Recommend(ctx, req)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "RECOMMENDATION_ERROR", "Failed to generate recommendations", err)
		return
	}

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data:   resp,
		Metadata: models.Metadata{
			Timestamp:   time.Now(),
			QueryTimeMS: resp.Metadata.LatencyMS,
		},
	})
}

// GetContinueWatching handles GET /api/v1/recommendations/user/{userID}/continue
// Returns in-progress content for a user.
func (h *RecommendHandler) GetContinueWatching(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	userIDStr := chi.URLParam(r, "userID")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_USER_ID", "Invalid user ID", err)
		return
	}

	limit := 10
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	items, err := h.db.GetContinueWatchingItems(ctx, userID, limit)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "QUERY_ERROR", "Failed to get continue watching items", err)
		return
	}

	if items == nil {
		items = []recommend.ScoredItem{}
	}

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data: map[string]interface{}{
			"items": items,
			"count": len(items),
		},
		Metadata: models.Metadata{
			Timestamp: time.Now(),
		},
	})
}

// GetSimilar handles GET /api/v1/recommendations/similar/{itemID}
// Returns items similar to the given item.
func (h *RecommendHandler) GetSimilar(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	itemIDStr := chi.URLParam(r, "itemID")
	itemID, err := strconv.Atoi(itemIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_ITEM_ID", "Invalid item ID", err)
		return
	}

	k := 10
	if kStr := r.URL.Query().Get("k"); kStr != "" {
		if parsed, err := strconv.Atoi(kStr); err == nil && parsed > 0 {
			k = parsed
		}
	}

	req := recommend.Request{
		UserID:        0, // Not user-specific
		K:             k,
		Mode:          recommend.ModeSimilar,
		CurrentItemID: itemID,
		RequestID:     r.Header.Get("X-Request-ID"),
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	resp, err := h.engine.Recommend(ctx, req)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "RECOMMENDATION_ERROR", "Failed to find similar items", err)
		return
	}

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data:   resp,
		Metadata: models.Metadata{
			Timestamp:   time.Now(),
			QueryTimeMS: resp.Metadata.LatencyMS,
		},
	})
}

// GetRecommendationStatus handles GET /api/v1/recommendations/status
// Returns the current training status and metrics.
func (h *RecommendHandler) GetRecommendationStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	status := h.engine.GetStatus()
	metrics := h.engine.GetMetrics()

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data: map[string]interface{}{
			"training": status,
			"metrics":  metrics,
		},
		Metadata: models.Metadata{
			Timestamp: time.Now(),
		},
	})
}

// GetRecommendationConfig handles GET /api/v1/recommendations/config
// Returns the current recommendation configuration.
func (h *RecommendHandler) GetRecommendationConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	cfg := h.engine.GetConfig()

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data:   cfg,
		Metadata: models.Metadata{
			Timestamp: time.Now(),
		},
	})
}

// UpdateRecommendationConfig handles PUT /api/v1/recommendations/config
// Updates the recommendation configuration.
func (h *RecommendHandler) UpdateRecommendationConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	var cfg recommend.Config
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON body", err)
		return
	}

	if err := h.engine.UpdateConfig(&cfg); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_CONFIG", "Invalid configuration", err)
		return
	}

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data: map[string]string{
			"message": "Configuration updated",
		},
		Metadata: models.Metadata{
			Timestamp: time.Now(),
		},
	})
}

// TriggerTraining handles POST /api/v1/recommendations/train
// Triggers model retraining (admin only).
func (h *RecommendHandler) TriggerTraining(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	// Check if already training
	status := h.engine.GetStatus()
	if status.IsTraining {
		respondError(w, http.StatusConflict, "TRAINING_IN_PROGRESS", "Training is already in progress", nil)
		return
	}

	// Start training in background
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()

		if err := h.engine.Train(ctx); err != nil {
			logging.Error().Err(err).Msg("recommendation training failed")
		} else {
			logging.Info().Msg("recommendation training completed")
		}
	}()

	respondJSON(w, http.StatusAccepted, &models.APIResponse{
		Status: "success",
		Data: map[string]string{
			"message": "Training started",
		},
		Metadata: models.Metadata{
			Timestamp: time.Now(),
		},
	})
}

// GetExploreRecommendations handles GET /api/v1/recommendations/user/{userID}/explore
// Returns discovery-focused recommendations.
func (h *RecommendHandler) GetExploreRecommendations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	userIDStr := chi.URLParam(r, "userID")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_USER_ID", "Invalid user ID", err)
		return
	}

	k := 20
	if kStr := r.URL.Query().Get("k"); kStr != "" {
		if parsed, err := strconv.Atoi(kStr); err == nil && parsed > 0 {
			k = parsed
		}
	}

	req := recommend.Request{
		UserID:    userID,
		K:         k,
		Mode:      recommend.ModeExplore,
		RequestID: r.Header.Get("X-Request-ID"),
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	resp, err := h.engine.Recommend(ctx, req)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "RECOMMENDATION_ERROR", "Failed to generate explore recommendations", err)
		return
	}

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data:   resp,
		Metadata: models.Metadata{
			Timestamp:   time.Now(),
			QueryTimeMS: resp.Metadata.LatencyMS,
		},
	})
}

// GetWhatsNext handles GET /api/v1/recommendations/next/{itemID}
// Returns sequential predictions based on Markov chain patterns.
func (h *RecommendHandler) GetWhatsNext(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	itemIDStr := chi.URLParam(r, "itemID")
	itemID, err := strconv.Atoi(itemIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_ITEM_ID", "Invalid item ID", err)
		return
	}

	k := 6
	if kStr := r.URL.Query().Get("k"); kStr != "" {
		if parsed, err := strconv.Atoi(kStr); err == nil && parsed > 0 && parsed <= 20 {
			k = parsed
		}
	}

	// Use similar mode which uses item-based algorithms including Markov chain
	req := recommend.Request{
		UserID:        0,
		K:             k,
		Mode:          recommend.ModeSimilar,
		CurrentItemID: itemID,
		RequestID:     r.Header.Get("X-Request-ID"),
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	resp, err := h.engine.Recommend(ctx, req)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "RECOMMENDATION_ERROR", "Failed to generate next predictions", err)
		return
	}

	// Get item title for the source
	var sourceTitle string
	if item, err := h.db.GetMediaItemByID(ctx, itemID); err == nil && item != nil {
		sourceTitle = item.Title
	}

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data: map[string]interface{}{
			"predictions":       resp.Items,
			"source_item_id":    itemID,
			"source_item_title": sourceTitle,
			"transition_count":  len(resp.Items),
			"metadata": map[string]interface{}{
				"latency_ms":    resp.Metadata.LatencyMS,
				"found":         len(resp.Items) > 0,
				"model_version": resp.Metadata.ModelVersion,
			},
		},
		Metadata: models.Metadata{
			Timestamp:   time.Now(),
			QueryTimeMS: resp.Metadata.LatencyMS,
		},
	})
}

// AlgorithmInfo represents information about a recommendation algorithm.
type AlgorithmInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Tooltip     string `json:"tooltip"`
	Category    string `json:"category"`
	Lightweight bool   `json:"lightweight"`
}

// algorithmInfoMap contains static algorithm information for the UI.
var algorithmInfoMap = []AlgorithmInfo{
	{ID: "covisit", Name: "Co-Visitation", Description: "Items frequently watched together", Tooltip: "Tracks which items are watched in the same session.", Category: "basic", Lightweight: true},
	{ID: "content", Name: "Content-Based", Description: "Similar genres, actors, directors", Tooltip: "Analyzes item attributes like genre, actors, directors.", Category: "basic", Lightweight: true},
	{ID: "popularity", Name: "Popularity", Description: "Trending items with time decay", Tooltip: "Recommends currently popular items with time decay.", Category: "basic", Lightweight: true},
	{ID: "ease", Name: "EASE", Description: "Embarrassingly Shallow Autoencoders", Tooltip: "Matrix factorization that learns item-item relationships.", Category: "matrix", Lightweight: false},
	{ID: "als", Name: "ALS", Description: "Alternating Least Squares", Tooltip: "Classic matrix factorization for implicit feedback.", Category: "matrix", Lightweight: false},
	{ID: "usercf", Name: "User-CF", Description: "User-based collaborative filtering", Tooltip: "Finds users with similar viewing patterns.", Category: "collaborative", Lightweight: false},
	{ID: "itemcf", Name: "Item-CF", Description: "Item-based collaborative filtering", Tooltip: "Finds similar items based on who watched them.", Category: "collaborative", Lightweight: false},
	{ID: "fpmc", Name: "FPMC", Description: "Factorized Personalized Markov Chains", Tooltip: "Combines sequential patterns with personalization.", Category: "sequential", Lightweight: false},
	{ID: "markov", Name: "Markov Chain", Description: "Sequential viewing patterns", Tooltip: "Learns viewing sequences for what to watch next.", Category: "sequential", Lightweight: true},
	{ID: "bpr", Name: "BPR", Description: "Bayesian Personalized Ranking", Tooltip: "Optimizes for ranking quality with pairwise learning.", Category: "advanced", Lightweight: false},
	{ID: "timeaware", Name: "Time-Aware CF", Description: "Time-weighted collaborative filtering", Tooltip: "Weighs recent interactions more heavily.", Category: "advanced", Lightweight: false},
	{ID: "multihop", Name: "Multi-Hop ItemCF", Description: "Graph-based item similarity", Tooltip: "Explores multi-hop connections between items.", Category: "advanced", Lightweight: false},
	{ID: "linucb", Name: "LinUCB", Description: "Contextual bandit exploration", Tooltip: "Balances exploitation with exploration.", Category: "bandit", Lightweight: true},
}

// GetAlgorithms handles GET /api/v1/recommendations/algorithms
// Returns information about available recommendation algorithms.
func (h *RecommendHandler) GetAlgorithms(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data:   algorithmInfoMap,
		Metadata: models.Metadata{
			Timestamp: time.Now(),
		},
	})
}

// GetAlgorithmMetrics handles GET /api/v1/recommendations/algorithms/metrics
// Returns per-algorithm performance metrics.
func (h *RecommendHandler) GetAlgorithmMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	status := h.engine.GetStatus()
	isTrained := status.ModelVersion > 0

	// Build per-algorithm metrics
	algMetrics := make(map[string]interface{})
	for _, info := range algorithmInfoMap {
		algMetrics[info.ID] = map[string]interface{}{
			"latency_ms":  0, // Would need engine support for per-algorithm latency
			"predictions": 0,
			"trained":     isTrained,
		}
	}

	respondJSON(w, http.StatusOK, &models.APIResponse{
		Status: "success",
		Data:   algMetrics,
		Metadata: models.Metadata{
			Timestamp: time.Now(),
		},
	})
}
