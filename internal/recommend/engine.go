// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package recommend

import (
	"context"
	"fmt"
	"math/rand"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog"
)

// Note: This package has no external dependencies on other internal packages
// to maintain clean separation. The DataProvider interface allows integration
// with the database package without creating circular imports.

// Engine coordinates multiple recommendation algorithms and produces final recommendations.
// It is safe for concurrent use.
type Engine struct {
	// Configuration
	config *Config
	logger zerolog.Logger

	// Registered algorithms and rerankers
	algorithms []Algorithm
	rerankers  []Reranker
	algMu      sync.RWMutex

	// Training state
	trainMu       sync.RWMutex
	trainStatus   TrainingStatus
	modelVersion  int32
	lastTrainedAt time.Time

	// Metrics
	metrics      Metrics
	metricsMu    sync.RWMutex
	requestCount atomic.Int64
	cacheHits    atomic.Int64
	cacheMisses  atomic.Int64
	errorCount   atomic.Int64

	// Cache (simple in-memory LRU)
	cache   map[string]cacheEntry
	cacheMu sync.RWMutex

	// Random source for determinism (protected by rngMu for concurrent access)
	rng   *rand.Rand
	rngMu sync.Mutex

	// Data provider interface
	dataProvider DataProvider
}

// cacheEntry holds a cached recommendation response.
type cacheEntry struct {
	response  *Response
	expiresAt time.Time
}

// DataProvider defines the interface for fetching training and prediction data.
// This is typically implemented by the database layer.
type DataProvider interface {
	// GetInteractions returns user-item interactions for training.
	GetInteractions(ctx context.Context, since time.Time) ([]Interaction, error)

	// GetItems returns item metadata.
	GetItems(ctx context.Context) ([]Item, error)

	// GetUserHistory returns item IDs the user has interacted with.
	GetUserHistory(ctx context.Context, userID int) ([]int, error)

	// GetCandidates returns candidate item IDs for recommendations.
	// Excludes items the user has already interacted with.
	GetCandidates(ctx context.Context, userID int, limit int) ([]int, error)
}

// NewEngine creates a new recommendation engine.
//
//nolint:gocritic // logger passed by value is acceptable for zerolog
func NewEngine(cfg *Config, logger zerolog.Logger) (*Engine, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Use provided seed or default for determinism
	seed := cfg.Seed
	if seed == 0 {
		seed = 42
	}

	return &Engine{
		config:     cfg,
		logger:     logger.With().Str("component", "recommend").Logger(),
		algorithms: make([]Algorithm, 0),
		rerankers:  make([]Reranker, 0),
		cache:      make(map[string]cacheEntry),
		rng:        rand.New(rand.NewSource(seed)), //nolint:gosec // math/rand is fine for recommendation shuffling
		metrics: Metrics{
			AlgorithmMetrics: make(map[string]AlgorithmMetrics),
		},
	}, nil
}

// SetDataProvider sets the data provider for training and prediction.
func (e *Engine) SetDataProvider(dp DataProvider) {
	e.dataProvider = dp
}

// RegisterAlgorithm adds an algorithm to the ensemble.
func (e *Engine) RegisterAlgorithm(alg Algorithm) {
	e.algMu.Lock()
	defer e.algMu.Unlock()

	e.algorithms = append(e.algorithms, alg)
	e.logger.Info().
		Str("algorithm", alg.Name()).
		Msg("registered algorithm")
}

// RegisterReranker adds a reranker to the post-processing pipeline.
func (e *Engine) RegisterReranker(rr Reranker) {
	e.algMu.Lock()
	defer e.algMu.Unlock()

	e.rerankers = append(e.rerankers, rr)
	e.logger.Info().
		Str("reranker", rr.Name()).
		Msg("registered reranker")
}

// Recommend generates recommendations for a user.
//
//nolint:gocritic // hugeParam: req passed by value for immutability
func (e *Engine) Recommend(ctx context.Context, req Request) (*Response, error) {
	start := time.Now()
	e.requestCount.Add(1)

	// Prepare request
	req = e.prepareRequest(req)
	logger := e.createRequestLogger(req)
	logger.Debug().Msg("processing recommendation request")

	// Check cache early return
	if resp := e.tryGetCachedResponse(req, start, logger); resp != nil {
		return resp, nil
	}

	// Build exclude set
	e.buildExcludeSet(&req)

	// Get and validate candidates
	candidates, err := e.getCandidates(ctx, req)
	if err != nil {
		e.errorCount.Add(1)
		return nil, fmt.Errorf("get candidates: %w", err)
	}

	if len(candidates) == 0 {
		logger.Debug().Msg("no candidates available")
		return e.emptyResponse(req, start), nil
	}

	// Score and rank items
	scoredItems, algorithmsUsed, err := e.scoreAndRankItems(ctx, req, candidates)
	if err != nil {
		e.errorCount.Add(1)
		return nil, fmt.Errorf("score candidates: %w", err)
	}

	// Build and cache response
	resp := e.buildResponse(req, scoredItems, algorithmsUsed, candidates, start)
	e.cacheResponse(req, resp)

	logger.Debug().
		Int("candidates", len(candidates)).
		Int("returned", len(scoredItems)).
		Int64("latency_ms", resp.Metadata.LatencyMS).
		Msg("recommendation complete")

	return resp, nil
}

// prepareRequest applies defaults and generates request ID if needed.
//
//nolint:gocritic // hugeParam: req passed by value for immutability
func (e *Engine) prepareRequest(req Request) Request {
	if req.RequestID == "" {
		req.RequestID = e.generateRequestID()
	}

	if req.K == 0 {
		req.K = e.config.Limits.DefaultK
	}
	if req.K > e.config.Limits.MaxK {
		req.K = e.config.Limits.MaxK
	}

	return req
}

// createRequestLogger creates a logger with request context.
//
//nolint:gocritic // hugeParam: req passed by value for immutability
func (e *Engine) createRequestLogger(req Request) zerolog.Logger {
	return e.logger.With().
		Str("request_id", req.RequestID).
		Int("user_id", req.UserID).
		Str("mode", req.Mode.String()).
		Logger()
}

// tryGetCachedResponse attempts to retrieve a cached response.
//
//nolint:gocritic // hugeParam: req passed by value for immutability
func (e *Engine) tryGetCachedResponse(req Request, start time.Time, logger zerolog.Logger) *Response {
	if !e.config.Cache.Enabled {
		return nil
	}

	cacheKey := e.cacheKey(req)
	resp := e.checkCache(cacheKey)
	if resp == nil {
		e.cacheMisses.Add(1)
		return nil
	}

	e.cacheHits.Add(1)
	resp.Metadata.CacheHit = true
	resp.Metadata.LatencyMS = time.Since(start).Milliseconds()
	logger.Debug().Msg("cache hit")
	return resp
}

// buildExcludeSet builds the exclude set from provided IDs.
func (e *Engine) buildExcludeSet(req *Request) {
	if req.Exclude == nil && len(req.ExcludeIDs) > 0 {
		req.Exclude = make(map[int]struct{}, len(req.ExcludeIDs))
		for _, id := range req.ExcludeIDs {
			req.Exclude[id] = struct{}{}
		}
	}
}

// scoreAndRankItems scores candidates and applies reranking.
//
//nolint:gocritic // hugeParam: req passed by value for immutability
func (e *Engine) scoreAndRankItems(ctx context.Context, req Request, candidates []int) ([]ScoredItem, []string, error) {
	scoredItems, algorithmsUsed, err := e.scoreCandidates(ctx, req, candidates)
	if err != nil {
		return nil, nil, err
	}

	sort.Slice(scoredItems, func(i, j int) bool {
		return scoredItems[i].Score > scoredItems[j].Score
	})

	scoredItems = e.applyRerankers(ctx, scoredItems, req.K)

	if len(scoredItems) > req.K {
		scoredItems = scoredItems[:req.K]
	}

	return scoredItems, algorithmsUsed, nil
}

// buildResponse constructs the final response.
//
//nolint:gocritic // hugeParam: req passed by value for immutability
func (e *Engine) buildResponse(req Request, scoredItems []ScoredItem, algorithmsUsed []string, candidates []int, start time.Time) *Response {
	return &Response{
		Items:           scoredItems,
		TotalCandidates: len(candidates),
		Metadata:        e.buildResponseMetadata(req, algorithmsUsed, start, false),
	}
}

// buildResponseMetadata constructs response metadata.
//
//nolint:gocritic // hugeParam: req passed by value for immutability
func (e *Engine) buildResponseMetadata(req Request, algorithmsUsed []string, start time.Time, cacheHit bool) ResponseMetadata {
	e.trainMu.RLock()
	trainedAt := e.lastTrainedAt
	e.trainMu.RUnlock()

	return ResponseMetadata{
		RequestID:      req.RequestID,
		UserID:         req.UserID,
		Mode:           req.Mode.String(),
		AlgorithmsUsed: algorithmsUsed,
		LatencyMS:      time.Since(start).Milliseconds(),
		CacheHit:       cacheHit,
		ModelVersion:   int(atomic.LoadInt32(&e.modelVersion)),
		TrainedAt:      trainedAt,
		Timestamp:      time.Now(),
	}
}

// cacheResponse stores the response in cache if enabled.
//
//nolint:gocritic // hugeParam: req passed by value for immutability
func (e *Engine) cacheResponse(req Request, resp *Response) {
	if e.config.Cache.Enabled {
		cacheKey := e.cacheKey(req)
		e.storeCache(cacheKey, resp)
	}
}

// getCandidates retrieves candidate items for scoring.
//
//nolint:gocritic // hugeParam: req passed by value for immutability
func (e *Engine) getCandidates(ctx context.Context, req Request) ([]int, error) {
	if e.dataProvider == nil {
		return nil, fmt.Errorf("data provider not set")
	}

	history, err := e.dataProvider.GetUserHistory(ctx, req.UserID)
	if err != nil {
		return nil, fmt.Errorf("get user history: %w", err)
	}

	exclude := e.buildExclusionSet(history, req.Exclude)

	candidates, err := e.dataProvider.GetCandidates(ctx, req.UserID, e.config.Limits.MaxCandidates)
	if err != nil {
		return nil, fmt.Errorf("get candidates: %w", err)
	}

	return e.filterCandidates(candidates, exclude), nil
}

// buildExclusionSet creates a combined exclusion set from history and request exclusions.
func (e *Engine) buildExclusionSet(history []int, requestExclude map[int]struct{}) map[int]struct{} {
	exclude := make(map[int]struct{}, len(history)+len(requestExclude))

	for _, id := range history {
		exclude[id] = struct{}{}
	}

	for id := range requestExclude {
		exclude[id] = struct{}{}
	}

	return exclude
}

// filterCandidates removes excluded items from candidates.
func (e *Engine) filterCandidates(candidates []int, exclude map[int]struct{}) []int {
	filtered := make([]int, 0, len(candidates))
	for _, id := range candidates {
		if _, excluded := exclude[id]; !excluded {
			filtered = append(filtered, id)
		}
	}
	return filtered
}

// scoreCandidates scores candidate items using all registered algorithms.
//
//nolint:gocritic // hugeParam: req passed by value for immutability
func (e *Engine) scoreCandidates(ctx context.Context, req Request, candidates []int) ([]ScoredItem, []string, error) {
	algorithms := e.getAlgorithms()
	if len(algorithms) == 0 {
		return nil, nil, fmt.Errorf("no algorithms registered")
	}

	weights := e.config.Weights.Normalize().ToMap()
	results := e.runAlgorithmPredictions(ctx, req, algorithms, candidates)
	return e.combineAlgorithmScores(results, weights)
}

// getAlgorithms returns a copy of registered algorithms.
func (e *Engine) getAlgorithms() []Algorithm {
	e.algMu.RLock()
	defer e.algMu.RUnlock()
	return e.algorithms
}

// algResult holds the result of a single algorithm prediction.
type algResult struct {
	name   string
	scores map[int]float64
	err    error
}

// runAlgorithmPredictions runs all algorithms in parallel.
//
//nolint:gocritic // hugeParam: req passed by value for immutability
func (e *Engine) runAlgorithmPredictions(ctx context.Context, req Request, algorithms []Algorithm, candidates []int) []algResult {
	results := make([]algResult, len(algorithms))
	var wg sync.WaitGroup

	for i, alg := range algorithms {
		wg.Add(1)
		go func(idx int, a Algorithm) {
			defer wg.Done()
			results[idx] = e.runSingleAlgorithm(ctx, req, a, candidates)
		}(i, alg)
	}

	wg.Wait()
	return results
}

// runSingleAlgorithm runs a single algorithm prediction.
//
//nolint:gocritic // hugeParam: req passed by value for immutability
func (e *Engine) runSingleAlgorithm(ctx context.Context, req Request, alg Algorithm, candidates []int) algResult {
	result := algResult{name: alg.Name()}

	if !alg.IsTrained() {
		return result
	}

	algCtx, cancel := context.WithTimeout(ctx, e.config.Limits.PredictionTimeout)
	defer cancel()

	scores, err := e.predictWithAlgorithm(algCtx, req, alg, candidates)
	result.scores = scores
	result.err = err

	return result
}

// predictWithAlgorithm calls the appropriate prediction method based on mode.
//
//nolint:gocritic // hugeParam: req passed by value for immutability
func (e *Engine) predictWithAlgorithm(ctx context.Context, req Request, alg Algorithm, candidates []int) (map[int]float64, error) {
	if req.Mode == ModeSimilar && req.CurrentItemID > 0 {
		return alg.PredictSimilar(ctx, req.CurrentItemID, candidates)
	}
	return alg.Predict(ctx, req.UserID, candidates)
}

// combineAlgorithmScores combines scores from multiple algorithms.
func (e *Engine) combineAlgorithmScores(results []algResult, weights map[string]float64) ([]ScoredItem, []string, error) {
	combinedScores := make(map[int]float64)
	scoreBreakdown := make(map[int]map[string]float64)
	algorithmsUsed := make([]string, 0, len(results))

	for _, result := range results {
		if !e.shouldUseResult(result, weights) {
			continue
		}

		algorithmsUsed = append(algorithmsUsed, result.name)
		weight := weights[result.name]

		for itemID, score := range result.scores {
			combinedScores[itemID] += weight * score
			e.addToScoreBreakdown(scoreBreakdown, itemID, result.name, score)
		}
	}

	return e.buildScoredItems(combinedScores, scoreBreakdown), algorithmsUsed, nil
}

// shouldUseResult checks if an algorithm result should be used.
func (e *Engine) shouldUseResult(result algResult, weights map[string]float64) bool {
	if result.err != nil {
		e.logger.Warn().
			Str("algorithm", result.name).
			Err(result.err).
			Msg("algorithm prediction failed")
		return false
	}

	if len(result.scores) == 0 {
		return false
	}

	return weights[result.name] > 0
}

// addToScoreBreakdown adds a score to the breakdown map.
func (e *Engine) addToScoreBreakdown(breakdown map[int]map[string]float64, itemID int, algName string, score float64) {
	if breakdown[itemID] == nil {
		breakdown[itemID] = make(map[string]float64)
	}
	breakdown[itemID][algName] = score
}

// buildScoredItems converts score maps to ScoredItem slice.
func (e *Engine) buildScoredItems(combinedScores map[int]float64, scoreBreakdown map[int]map[string]float64) []ScoredItem {
	items := make([]ScoredItem, 0, len(combinedScores))
	for itemID, score := range combinedScores {
		items = append(items, ScoredItem{
			Item:   Item{ID: itemID},
			Score:  score,
			Scores: scoreBreakdown[itemID],
		})
	}
	return items
}

// applyRerankers applies post-processing rerankers to the scored items.
func (e *Engine) applyRerankers(ctx context.Context, items []ScoredItem, k int) []ScoredItem {
	e.algMu.RLock()
	rerankers := e.rerankers
	e.algMu.RUnlock()

	for _, rr := range rerankers {
		items = rr.Rerank(ctx, items, k)
	}

	return items
}

// Train trains all registered algorithms on the available data.
// Returns immediately with an error if training is already in progress.
func (e *Engine) Train(ctx context.Context) error {
	if err := e.acquireTrainingLock(); err != nil {
		return err
	}
	defer e.trainMu.Unlock()

	if e.dataProvider == nil {
		return fmt.Errorf("data provider not set")
	}

	start := time.Now()
	e.initializeTrainingStatus()
	e.logger.Info().Msg("starting model training")

	defer func() {
		e.finalizeTrainingStatus(start)
	}()

	trainCtx, cancel := context.WithTimeout(ctx, e.config.Training.Timeout)
	defer cancel()

	// Load and validate training data
	interactions, items, err := e.loadTrainingData(trainCtx)
	if err != nil {
		e.trainStatus.LastError = err.Error()
		return err
	}

	// Train all algorithms
	if err := e.trainAllAlgorithms(trainCtx, interactions, items); err != nil {
		return err
	}

	// Finalize training
	e.completeTraining()

	e.logger.Info().
		Int("version", e.trainStatus.ModelVersion).
		Int64("duration_ms", e.trainStatus.LastTrainingDurationMS).
		Msg("model training complete")

	return nil
}

// acquireTrainingLock attempts to acquire the training lock.
func (e *Engine) acquireTrainingLock() error {
	if !e.trainMu.TryLock() {
		return fmt.Errorf("training already in progress")
	}

	if e.trainStatus.IsTraining {
		e.trainMu.Unlock()
		return fmt.Errorf("training already in progress")
	}

	return nil
}

// initializeTrainingStatus prepares the training status.
func (e *Engine) initializeTrainingStatus() {
	e.trainStatus.IsTraining = true
	e.trainStatus.Progress = 0
	e.trainStatus.LastError = ""
}

// finalizeTrainingStatus updates the training status after completion.
func (e *Engine) finalizeTrainingStatus(start time.Time) {
	e.trainStatus.IsTraining = false
	e.trainStatus.LastTrainingDurationMS = time.Since(start).Milliseconds()
}

// loadTrainingData loads and validates training data.
func (e *Engine) loadTrainingData(ctx context.Context) ([]Interaction, []Item, error) {
	since := time.Time{} // Get all interactions
	interactions, err := e.dataProvider.GetInteractions(ctx, since)
	if err != nil {
		return nil, nil, fmt.Errorf("get interactions: %w", err)
	}

	if err := e.validateInteractionCount(interactions); err != nil {
		return nil, nil, err
	}

	items, err := e.dataProvider.GetItems(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("get items: %w", err)
	}

	e.updateTrainingDataStats(interactions, items)
	return interactions, items, nil
}

// validateInteractionCount checks if there are sufficient interactions.
func (e *Engine) validateInteractionCount(interactions []Interaction) error {
	if len(interactions) < e.config.Training.MinInteractions {
		return fmt.Errorf("insufficient interactions: %d < %d", len(interactions), e.config.Training.MinInteractions)
	}
	return nil
}

// updateTrainingDataStats updates the training status with data statistics.
func (e *Engine) updateTrainingDataStats(interactions []Interaction, items []Item) {
	e.trainStatus.InteractionCount = len(interactions)
	e.trainStatus.ItemCount = len(items)
	e.trainStatus.UserCount = countUniqueUsers(interactions)

	e.logger.Info().
		Int("interactions", len(interactions)).
		Int("items", len(items)).
		Int("users", e.trainStatus.UserCount).
		Msg("loaded training data")
}

// trainAllAlgorithms trains each registered algorithm.
// Individual algorithm failures are logged but don't stop training of other algorithms.
//
//nolint:unparam // error return kept for future use; individual errors are logged but don't stop training
func (e *Engine) trainAllAlgorithms(ctx context.Context, interactions []Interaction, items []Item) error {
	algorithms := e.getAlgorithms()

	for i, alg := range algorithms {
		e.updateAlgorithmProgress(alg.Name(), i, len(algorithms))

		if err := alg.Train(ctx, interactions, items); err != nil {
			e.logger.Error().
				Str("algorithm", alg.Name()).
				Err(err).
				Msg("algorithm training failed")
			// Continue with other algorithms
			continue
		}

		e.logger.Debug().
			Str("algorithm", alg.Name()).
			Msg("algorithm training complete")
	}

	return nil
}

// updateAlgorithmProgress updates the training progress for an algorithm.
func (e *Engine) updateAlgorithmProgress(algName string, current, total int) {
	e.trainStatus.CurrentAlgorithm = algName
	e.trainStatus.Progress = (current * 100) / total

	e.logger.Debug().
		Str("algorithm", algName).
		Int("progress", e.trainStatus.Progress).
		Msg("training algorithm")
}

// completeTraining finalizes the training process.
func (e *Engine) completeTraining() {
	atomic.AddInt32(&e.modelVersion, 1)
	e.lastTrainedAt = time.Now()
	e.trainStatus.LastTrainedAt = e.lastTrainedAt
	e.trainStatus.ModelVersion = int(atomic.LoadInt32(&e.modelVersion))
	e.trainStatus.Progress = 100
	e.trainStatus.CurrentAlgorithm = ""

	if e.config.Cache.InvalidateOnTrain {
		e.clearCache()
	}
}

// GetStatus returns the current training status.
func (e *Engine) GetStatus() TrainingStatus {
	e.trainMu.RLock()
	defer e.trainMu.RUnlock()

	return e.trainStatus
}

// GetMetrics returns the current engine metrics.
func (e *Engine) GetMetrics() Metrics {
	e.metricsMu.RLock()
	defer e.metricsMu.RUnlock()

	// Update with atomic values
	m := e.metrics
	m.RequestCount = e.requestCount.Load()
	m.CacheHits = e.cacheHits.Load()
	m.CacheMisses = e.cacheMisses.Load()
	m.ErrorCount = e.errorCount.Load()

	return m
}

// GetConfig returns a copy of the current configuration.
func (e *Engine) GetConfig() *Config {
	return e.config.Clone()
}

// UpdateConfig updates the engine configuration.
func (e *Engine) UpdateConfig(cfg *Config) error {
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	e.config = cfg
	e.logger.Info().Msg("configuration updated")

	return nil
}

// cacheKey generates a cache key for a request.
//
//nolint:gocritic // hugeParam: req passed by value for simplicity
func (e *Engine) cacheKey(req Request) string {
	return fmt.Sprintf("rec:%d:%d:%s", req.UserID, req.K, req.Mode.String())
}

// checkCache checks if a cached response exists and is valid.
// Returns a copy of the cached response to avoid concurrent modification.
func (e *Engine) checkCache(key string) *Response {
	e.cacheMu.RLock()
	defer e.cacheMu.RUnlock()

	entry, ok := e.cache[key]
	if !ok {
		return nil
	}

	if time.Now().After(entry.expiresAt) {
		return nil
	}

	return e.copyCachedResponse(entry.response)
}

// copyCachedResponse creates a copy of a cached response.
func (e *Engine) copyCachedResponse(resp *Response) *Response {
	items := make([]ScoredItem, len(resp.Items))
	copy(items, resp.Items)

	return &Response{
		Items:           items,
		TotalCandidates: resp.TotalCandidates,
		Metadata:        resp.Metadata, // Metadata is a value type, safe to copy
	}
}

// storeCache stores a response in the cache.
func (e *Engine) storeCache(key string, resp *Response) {
	e.cacheMu.Lock()
	defer e.cacheMu.Unlock()

	e.evictIfCacheFull()

	e.cache[key] = cacheEntry{
		response:  resp,
		expiresAt: time.Now().Add(e.config.Cache.TTL),
	}
}

// evictIfCacheFull evicts expired entries if cache is at capacity.
// Must be called with cacheMu held.
func (e *Engine) evictIfCacheFull() {
	if len(e.cache) >= e.config.Cache.MaxEntries {
		e.evictExpiredLocked()
	}
}

// clearCache removes all cached entries.
func (e *Engine) clearCache() {
	e.cacheMu.Lock()
	defer e.cacheMu.Unlock()

	e.cache = make(map[string]cacheEntry)
	e.logger.Debug().Msg("cache cleared")
}

// evictExpiredLocked removes expired cache entries.
// Must be called with cacheMu held.
func (e *Engine) evictExpiredLocked() {
	now := time.Now()
	for key, entry := range e.cache {
		if now.After(entry.expiresAt) {
			delete(e.cache, key)
		}
	}
}

// emptyResponse returns an empty response for cases with no candidates.
//
//nolint:gocritic // hugeParam: req passed by value for immutability
func (e *Engine) emptyResponse(req Request, start time.Time) *Response {
	return &Response{
		Items:           []ScoredItem{},
		TotalCandidates: 0,
		Metadata:        e.buildResponseMetadata(req, []string{}, start, false),
	}
}

// generateRequestID generates a unique request ID for tracing.
// This method is safe for concurrent use.
func (e *Engine) generateRequestID() string {
	e.rngMu.Lock()
	n := e.rng.Intn(10000)
	e.rngMu.Unlock()
	return fmt.Sprintf("rec-%d-%d", time.Now().UnixNano(), n)
}

// countUniqueUsers counts unique users in interactions.
func countUniqueUsers(interactions []Interaction) int {
	users := make(map[int]struct{}, len(interactions))
	for _, i := range interactions {
		users[i.UserID] = struct{}{}
	}
	return len(users)
}
